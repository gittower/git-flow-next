// Package cmd — integrate command.
//
// The integrate command merges a base branch upstream into its configured
// parent (e.g. develop -> main), honoring the branch type's upstream merge
// strategy, optionally tagging the parent, and auto-updating the parent's
// autoUpdate children. It reuses finish's conflict-resumable merge -> tag ->
// update-children state machine but OMITS the delete-branch step: base branches
// are permanent and are never deleted, created, or renamed.
//
// Key differences from finish:
//   - Operates on base branches only (BranchConfig.Type == "base").
//   - Never deletes the integrated branch; the state machine terminates after
//     updating children (Action == "integrate").
//   - --tag <version> is a string flag that both enables tagging and sets the
//     name; there is no version-derived default, so tagging enabled without a
//     name errors before merging.
//   - Tagging and fetching default OFF (fetch is opt-in).
//   - --continue/--abort gate on state.Action == "integrate" so an in-progress
//     finish/update is never hijacked.
package cmd

import (
	"fmt"
	"os"

	"github.com/gittower/git-flow-next/internal/config"
	"github.com/gittower/git-flow-next/internal/errors"
	"github.com/gittower/git-flow-next/internal/git"
	"github.com/gittower/git-flow-next/internal/mergestate"
	"github.com/spf13/cobra"
)

// IntegrateCommand is the public entry point for the integrate command. It maps
// git-flow errors to their exit codes and exits non-zero on failure.
func IntegrateCommand(name string, continueOp bool, abortOp bool, tagOptions *config.TagOptions, mergeOptions *config.MergeStrategyOptions, fetch *bool) {
	if err := executeIntegrate(name, continueOp, abortOp, tagOptions, mergeOptions, fetch); err != nil {
		var exitCode errors.ExitCode
		if flowErr, ok := err.(errors.Error); ok {
			exitCode = flowErr.ExitCode()
		} else {
			exitCode = errors.ExitCodeGitError
		}
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(int(exitCode))
	}
}

// executeIntegrate performs the integrate logic and returns any error.
func executeIntegrate(name string, continueOp bool, abortOp bool, tagOptions *config.TagOptions, mergeOptions *config.MergeStrategyOptions, fetch *bool) error {
	// Validate that git-flow is initialized.
	initialized, err := config.IsInitialized()
	if err != nil {
		return &errors.GitError{Operation: "check if git-flow is initialized", Err: err}
	}
	if !initialized {
		return &errors.NotInitializedError{}
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		return &errors.GitError{Operation: "load configuration", Err: err}
	}

	// Foreign-state guard. Refuse to touch an in-progress finish or update.
	// We load the raw state (without the stale-clearing side effect of
	// IsMergeInProgress) and check whether git is genuinely mid-operation,
	// because some operations (e.g. update) persist state without a BranchType,
	// which IsMergeInProgress would otherwise treat as stale and clear.
	if rawState, _ := mergestate.LoadMergeState(); rawState != nil && rawState.Action != "integrate" {
		if git.IsGitMergeInProgress() || git.IsGitRebaseInProgress() || git.IsGitSquashMergeInProgress() {
			return &errors.MergeInProgressError{BranchName: rawState.FullBranchName}
		}
	}

	// Merge-in-progress dispatch for integrate's own state.
	if mergestate.IsMergeInProgress() {
		state, err := mergestate.LoadMergeState()
		if err != nil {
			return &errors.GitError{Operation: "load merge state", Err: err}
		}
		if state.Action != "integrate" {
			return &errors.MergeInProgressError{BranchName: state.FullBranchName}
		}
		if abortOp {
			return handleAbort(state)
		}
		if continueOp {
			resolved := config.ResolveIntegrateOptions(cfg, state.BranchName, tagOptions, mergeOptions, fetch)
			// Restore the tag decision made on the initial run: --continue does
			// not repeat --tag, so re-resolving from CLI would drop the tag.
			resolved.ShouldTag = state.ShouldTag
			resolved.TagName = state.TagName
			resolved.TagMessage = state.TagMessage
			resolved.MessageFile = state.TagMessageFile
			resolved.ShouldSign = state.ShouldSign
			resolved.SigningKey = state.SigningKey
			branchConfig := cfg.Branches[state.BranchType]
			return handleContinue(cfg, state, branchConfig, resolved, mergeOptions)
		}
		return &errors.MergeInProgressError{BranchName: state.FullBranchName}
	}

	// No merge in progress: abort is a forgiving no-op, continue has nothing to
	// resume (matches finish's dispatch).
	if abortOp {
		return nil
	}
	if continueOp {
		return &errors.NoMergeInProgressError{}
	}

	// Resolve the source base branch: explicit argument or the current branch.
	if name == "" {
		currentBranch, err := git.GetCurrentBranch()
		if err != nil {
			return &errors.GitError{Operation: "get current branch", Err: err}
		}
		name = currentBranch
	}

	canonical, found := cfg.ResolveBranchName(name)
	if !found {
		// A real branch that is not a configured base branch is a topic/other
		// branch; a name that matches no branch at all is simply not found.
		if git.BranchExists(name) == nil {
			return &errors.NotBaseBranchError{BranchName: name}
		}
		return &errors.BranchNotFoundError{BranchName: name}
	}
	name = canonical
	branchConfig := cfg.Branches[name]

	// Validation gates — all before any branch/tag mutation.
	if branchConfig.Type != string(config.BranchTypeBase) {
		return &errors.NotBaseBranchError{BranchName: name}
	}
	if branchConfig.Parent == "" {
		return &errors.NoParentBranchError{BranchName: name}
	}
	if branchConfig.Parent == name {
		return &errors.SelfParentError{BranchName: name}
	}
	if branchConfig.UpstreamStrategy == string(config.MergeStrategyNone) {
		return &errors.NoUpstreamStrategyError{BranchName: name}
	}

	parent := branchConfig.Parent
	if err := git.BranchExists(parent); err != nil {
		return &errors.BranchNotFoundError{BranchName: parent}
	}

	// Resolve all options once before starting.
	resolved := config.ResolveIntegrateOptions(cfg, name, tagOptions, mergeOptions, fetch)

	// Tag-name pre-check: base branches have no version-derived default name, so
	// tagging enabled without a resolvable name must error before merging.
	if resolved.ShouldTag && resolved.TagName == "" {
		return &errors.TagEnabledNoNameError{}
	}

	// Optional fetch (opt-in). Fast-forward the local parent from the remote so
	// the integration merges against up-to-date remote history; failures are
	// non-fatal (the remote branch may not exist).
	if resolved.ShouldFetch && git.RemoteExists(cfg.Remote) {
		fmt.Printf("Fetching from remote '%s'...\n", cfg.Remote)
		if err := git.FetchBranch(cfg.Remote, fmt.Sprintf("%s:%s", parent, parent)); err != nil {
			fmt.Printf("Note: Could not fetch parent branch '%s': %v\n", parent, err)
		}
		if err := git.FetchBranch(cfg.Remote, name); err != nil {
			fmt.Printf("Note: Could not fetch branch '%s': %v\n", name, err)
		}
		fmt.Printf("Fetch completed\n")
	}

	// Discover auto-update child base branches and capture their strategies.
	childBranches := []string{}
	childStrategies := make(map[string]string)
	for branchName, branch := range cfg.Branches {
		if branch.Type == string(config.BranchTypeBase) && branch.Parent == parent && branch.AutoUpdate {
			fmt.Printf("Found child base branch '%s' with auto-update enabled\n", branchName)
			childBranches = append(childBranches, branchName)
			childStrategies[branchName] = branch.DownstreamStrategy
		}
	}

	// Build and persist the integrate merge state. Base branches have no prefix,
	// so FullBranchName == BranchName == the base name.
	state := &mergestate.MergeState{
		Action:          "integrate",
		BranchType:      name,
		BranchName:      name,
		CurrentStep:     stepMerge,
		ParentBranch:    parent,
		MergeStrategy:   resolved.MergeStrategy,
		FullBranchName:  name,
		ChildBranches:   childBranches,
		UpdatedBranches: []string{},
		ChildStrategies: childStrategies,
		SquashMessage:   resolved.SquashMessage,
		MergeMessage:    resolved.MergeMessage,
		UpdateMessage:   resolved.UpdateMessage,
		// Persist the tag decision so --continue recreates the same tag.
		ShouldTag:      resolved.ShouldTag,
		TagName:        resolved.TagName,
		TagMessage:     resolved.TagMessage,
		TagMessageFile: resolved.MessageFile,
		ShouldSign:     resolved.ShouldSign,
		SigningKey:     resolved.SigningKey,
	}
	if err := mergestate.SaveMergeState(state); err != nil {
		return &errors.GitError{Operation: "save merge state", Err: err}
	}

	// Drive the shared state machine: merge -> create_tag -> update_children ->
	// terminate. No delete step.
	return executeSteps(cfg, state, branchConfig, resolved)
}

// addIntegrateFlags registers the integrate command's flags. It mirrors
// addFinishFlags but drops all branch-retention flags (integrate never deletes),
// and makes --tag a string that both enables tagging and sets the tag name.
func addIntegrateFlags(cmd *cobra.Command) {
	// Operation control
	cmd.Flags().BoolP("continue", "c", false, "Continue the integrate operation after resolving conflicts")
	cmd.Flags().BoolP("abort", "a", false, "Abort the integrate operation; a no-op success when none is in progress")

	// Tag options (--tag is a string: it both enables tagging and sets the name)
	cmd.Flags().String("tag", "", "Create a tag with the given name on the parent branch")
	cmd.Flags().BoolP("notag", "n", false, "Do not create a tag (overrides a configured tag default)")
	cmd.Flags().BoolP("sign", "s", false, "Sign the tag cryptographically")
	cmd.Flags().Bool("no-sign", false, "Don't sign the tag cryptographically")
	cmd.Flags().StringP("signingkey", "u", "", "Use the given GPG key for the digital signature")
	cmd.Flags().StringP("message", "m", "", "Use the given message for the tag")
	cmd.Flags().String("messagefile", "", "Use contents of the given file as tag message")

	// Merge strategy
	cmd.Flags().BoolP("rebase", "r", false, "Rebase the branch onto its parent before merging")
	cmd.Flags().Bool("no-rebase", false, "Don't rebase (use the configured strategy)")
	cmd.Flags().BoolP("preserve-merges", "p", false, "Preserve merges during rebase")
	cmd.Flags().Bool("no-preserve-merges", false, "Flatten merges during rebase")
	cmd.Flags().Bool("no-ff", false, "Create a merge commit even for a fast-forward")
	cmd.Flags().Bool("ff", false, "Allow a fast-forward merge when possible")
	cmd.Flags().BoolP("squash", "S", false, "Squash all commits into a single commit")
	cmd.Flags().Bool("no-squash", false, "Keep individual commits (don't squash)")
	cmd.Flags().String("squash-message", "", "Custom commit message for a squash merge")
	cmd.Flags().StringP("merge-message", "M", "", "Custom commit message for the upstream merge")
	cmd.Flags().String("update-message", "", "Custom commit message for child branch updates")

	// Fetch (opt-in; default off)
	cmd.Flags().Bool("fetch", false, "Fetch from remote before integrating")
	cmd.Flags().Bool("no-fetch", false, "Don't fetch from remote before integrating")
}
