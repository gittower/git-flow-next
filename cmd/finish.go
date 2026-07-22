// Package cmd implements the finish command for completing topic branches.
//
// FINISH COMMAND STATE MACHINE
// =============================
//
// The finish command follows a strict state machine to ensure safe and consistent
// branch completion, even in the presence of merge conflicts. The state is persisted
// to disk to allow continuation after conflict resolution.
//
// State Flow:
// 1. MERGE STATE
//    - Creates merge state file with current step "merge"
//    - Executes merge into parent branch using topic branch's merge strategy
//    - On conflict: Saves state and exits for user to resolve
//    - On success: Advances to CREATE_TAG state
//
// 2. CREATE_TAG STATE
//    - Creates tag if configured (should not fail)
//    - Advances to UPDATE_CHILDREN state
//
// 3. UPDATE_CHILDREN STATE
//    - Identifies child branches with AutoUpdate=true
//    - For each child branch:
//      * Checks out child branch
//      * Merges parent branch using child's downstream strategy
//      * On conflict: Saves state (including which child) and exits
//      * On success: Marks child as updated, continues with next
//    - When all children updated: Advances to DELETE_BRANCH state
//
// 4. DELETE_BRANCH STATE
//    - Deletes topic branch (local/remote based on settings)
//    - Clears merge state file
//    - Operation complete
//
// Conflict Resolution:
// - User resolves conflicts manually
// - Runs 'git flow <type> finish --continue <name>' to resume
// - State machine continues from saved position
// - Can abort with 'git flow <type> finish --abort <name>'
//
// Critical Requirements:
// - State must ALWAYS be saved before exiting on conflicts
// - State must accurately reflect current branch and step
// - Child branch updates must track which child is being processed
// - All operations must be idempotent for safe continuation

package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/gittower/git-flow-next/internal/config"
	"github.com/gittower/git-flow-next/internal/errors"
	"github.com/gittower/git-flow-next/internal/git"
	"github.com/gittower/git-flow-next/internal/hooks"
	"github.com/gittower/git-flow-next/internal/mergestate"
	"github.com/gittower/git-flow-next/internal/update"
	"github.com/gittower/git-flow-next/internal/util"
)

// Step constants
const (
	stepMerge          = "merge"
	stepCreateTag      = "create_tag"
	stepUpdateChildren = "update_children"
	stepDeleteBranch   = "delete_branch"
)

// Strategy constants
const (
	strategyRebase = "rebase"
	strategySquash = "squash"
	strategyMerge  = "merge"
)

// =============================================================================
// PUBLIC ENTRY POINTS
// =============================================================================

// FinishCommand is the implementation of the finish command for topic branches
func FinishCommand(branchType string, name string, continueOp bool, abortOp bool, force bool, tagOptions *config.TagOptions, retentionOptions *config.BranchRetentionOptions, mergeOptions *config.MergeStrategyOptions, fetch *bool, noVerify *bool, push *bool, pushTag *bool) {
	if err := executeFinish(branchType, name, continueOp, abortOp, force, tagOptions, retentionOptions, mergeOptions, fetch, noVerify, push, pushTag); err != nil {
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

// =============================================================================
// MAIN EXECUTION FLOW
// =============================================================================

// executeFinish performs the actual branch finishing logic and returns any errors
func executeFinish(branchType string, name string, continueOp bool, abortOp bool, force bool, tagOptions *config.TagOptions, retentionOptions *config.BranchRetentionOptions, mergeOptions *config.MergeStrategyOptions, fetch *bool, noVerify *bool, push *bool, pushTag *bool) error {
	// Get configuration early
	cfg, err := config.LoadConfig()
	if err != nil {
		return &errors.GitError{Operation: "load configuration", Err: err}
	}

	// Get branch configuration
	branchConfig, ok := cfg.Branches[branchType]
	if !ok {
		return &errors.InvalidBranchTypeError{BranchType: branchType}
	}

	// Check if there's a merge in progress
	if mergestate.IsMergeInProgress() {
		state, err := mergestate.LoadMergeState()
		if err != nil {
			return &errors.GitError{Operation: "load merge state", Err: err}
		}

		// Get the branch config for the state's branch type
		stateBranchConfig, ok := cfg.Branches[state.BranchType]
		if !ok {
			return &errors.InvalidBranchTypeError{BranchType: state.BranchType}
		}

		if abortOp {
			return handleAbort(state)
		}

		if continueOp {
			// Resolve options for continue operation
			resolvedOptions := config.ResolveFinishOptions(cfg, state.BranchType, state.BranchName, tagOptions, retentionOptions, mergeOptions, fetch, noVerify, push, pushTag)
			return handleContinue(cfg, state, stateBranchConfig, resolvedOptions, mergeOptions)
		}

		return &errors.MergeInProgressError{BranchName: state.FullBranchName}
	}

	// Abort is forgiving: if there is no merge in progress (e.g. stale state
	// was just auto-cleared, or there was never a merge), --abort has nothing
	// to do and exits quietly. The repository is already in the clean state
	// that --abort promises. Continue still errors, since continuing implies
	// there is an operation to resume.
	if abortOp {
		return nil
	}
	if continueOp {
		return &errors.NoMergeInProgressError{}
	}

	// Resolve branch name (try with and without prefix)
	resolvedName, err := resolveBranchName(name, branchConfig)
	if err != nil {
		return err
	}
	name = resolvedName

	// If the branch exists but doesn't have the expected prefix
	if !strings.HasPrefix(name, branchConfig.Prefix) {
		if !force {
			// Get the short name for tag creation
			shortName := name
			if strings.Contains(name, "/") {
				parts := strings.Split(name, "/")
				shortName = parts[len(parts)-1]
			}

			// Prompt user for confirmation
			fmt.Printf("Warning: Branch '%s' is not a standard %s branch (missing prefix '%s').\n", name, branchType, branchConfig.Prefix)
			fmt.Printf("Finishing this branch will:\n")
			fmt.Printf("1. Merge it into '%s' using the %s strategy\n", branchConfig.Parent, branchConfig.UpstreamStrategy)

			// Resolve options early for confirmation dialog
			resolvedOptions := config.ResolveFinishOptions(cfg, branchType, shortName, tagOptions, retentionOptions, mergeOptions, fetch, noVerify, push, pushTag)

			if resolvedOptions.ShouldTag {
				fmt.Printf("2. Create a tag '%s'\n", resolvedOptions.TagName)
			}

			fmt.Printf("3. Delete the branch after successful merge\n\n")
			fmt.Printf("Do you want to continue? [y/N]: ")

			var response string
			fmt.Scanln(&response)
			if strings.ToLower(response) != "y" {
				return fmt.Errorf("operation cancelled by user")
			}
		}
	}

	// Get the short name for option resolution
	shortName := name
	if strings.HasPrefix(name, branchConfig.Prefix) {
		shortName = strings.TrimPrefix(name, branchConfig.Prefix)
	} else if strings.Contains(name, "/") {
		parts := strings.Split(name, "/")
		shortName = parts[len(parts)-1]
	}

	// Resolve all options once before starting operations
	resolvedOptions := config.ResolveFinishOptions(cfg, branchType, shortName, tagOptions, retentionOptions, mergeOptions, fetch, noVerify, push, pushTag)

	// Fetch the topic (and parent, best-effort) and verify the topic is in sync with its remote.
	// This runs only on the initial finish, never on --continue/--abort (handled above). A fatal
	// fetch failure or an out-of-sync topic aborts here, before any merge.
	if err := runFetchSyncPreflight(cfg, branchType, cfg.Remote, name, shortName, branchConfig.Parent, resolvedOptions.ShouldFetch, force); err != nil {
		return err
	}

	// Regular finish command flow
	return finishBranch(cfg, branchType, name, branchConfig, tagOptions, retentionOptions, mergeOptions, fetch, noVerify, push, pushTag)
}

func finishBranch(cfg *config.Config, branchType string, name string, branchConfig config.BranchConfig, tagOptions *config.TagOptions, retentionOptions *config.BranchRetentionOptions, mergeOptions *config.MergeStrategyOptions, fetch *bool, noVerify *bool, push *bool, pushTag *bool) error {
	// Validate that git-flow is initialized
	initialized, err := config.IsInitialized()
	if err != nil {
		return &errors.GitError{Operation: "check if git-flow is initialized", Err: err}
	}
	if !initialized {
		return &errors.NotInitializedError{}
	}

	// Validate inputs
	if name == "" {
		return &errors.InvalidBranchNameError{BranchName: name}
	}

	// Get the short name by removing the prefix if it exists
	shortName := name
	if strings.HasPrefix(name, branchConfig.Prefix) {
		shortName = strings.TrimPrefix(name, branchConfig.Prefix)
	} else if strings.Contains(name, "/") {
		// For non-standard branches, use the last part after the slash
		parts := strings.Split(name, "/")
		shortName = parts[len(parts)-1]
	}

	// Check if branch exists
	if err := git.BranchExists(name); err != nil {
		return &errors.BranchNotFoundError{BranchName: name}
	}

	// Get target branch (always the parent branch for finishing)
	targetBranch := branchConfig.Parent

	// Check if target branch exists
	if err := git.BranchExists(targetBranch); err != nil {
		return &errors.BranchNotFoundError{BranchName: targetBranch}
	}

	// Find child base branches that need to be updated and collect their strategies
	childBranches := []string{}
	childStrategies := make(map[string]string)
	for branchName, branch := range cfg.Branches {
		if branch.Type == string(config.BranchTypeBase) && branch.Parent == targetBranch && branch.AutoUpdate {
			fmt.Printf("Found child base branch '%s' with auto-update enabled\n", branchName)
			childBranches = append(childBranches, branchName)
			// Store the downstream strategy for this child branch
			childStrategies[branchName] = branch.DownstreamStrategy
		}
	}

	// Resolve all options once at the beginning
	resolvedOptions := config.ResolveFinishOptions(cfg, branchType, shortName, tagOptions, retentionOptions, mergeOptions, fetch, noVerify, push, pushTag)

	// Run pre-hook before starting finish operation
	gitDir, err := git.GetGitDir()
	if err != nil {
		return &errors.GitError{Operation: "get git directory", Err: err}
	}

	hookCtx := hooks.HookContext{
		BranchType: branchType,
		BranchName: shortName,
		FullBranch: name,
		BaseBranch: targetBranch,
		Origin:     cfg.Remote,
	}
	// Set version for branches configured with tagging
	if branchConfig.Tag {
		hookCtx.Version = shortName
	}

	if err := hooks.RunPreHook(gitDir, branchType, hooks.HookActionFinish, hookCtx); err != nil {
		return err
	}

	// Save merge state before starting
	state := &mergestate.MergeState{
		Action:          "finish",
		BranchType:      branchType,
		BranchName:      shortName,
		CurrentStep:     stepMerge,
		ParentBranch:    targetBranch,
		MergeStrategy:   branchConfig.UpstreamStrategy,
		FullBranchName:  name,
		ChildBranches:   childBranches,
		UpdatedBranches: []string{},
		ChildStrategies: childStrategies,
		SquashMessage:   resolvedOptions.SquashMessage,
		MergeMessage:    resolvedOptions.MergeMessage,
		UpdateMessage:   resolvedOptions.UpdateMessage,
		NoVerify:        resolvedOptions.NoVerify,
	}
	if err := mergestate.SaveMergeState(state); err != nil {
		return &errors.GitError{Operation: "save merge state", Err: err}
	}

	return executeSteps(cfg, state, branchConfig, resolvedOptions)
}

// =============================================================================
// STATE MACHINE AND CONTROL FLOW
// =============================================================================

// executeSteps runs the state machine for the finish operation
func executeSteps(cfg *config.Config, state *mergestate.MergeState, branchConfig config.BranchConfig, resolvedOptions *config.ResolvedFinishOptions) error {
	for {
		var err error
		switch state.CurrentStep {
		case stepMerge:
			err = handleMergeStep(cfg, state, branchConfig, resolvedOptions)
		case stepCreateTag:
			err = handleCreateTagStep(cfg, state, resolvedOptions)
		case stepUpdateChildren:
			err = handleUpdateChildrenStep(cfg, state, branchConfig, resolvedOptions)
		case stepDeleteBranch:
			return handleDeleteBranchStep(cfg, state, resolvedOptions) // Final step
		default:
			return &errors.GitError{Operation: fmt.Sprintf("unknown step '%s'", state.CurrentStep), Err: nil}
		}

		if err != nil {
			return err
		}
	}
}

func handleContinue(cfg *config.Config, state *mergestate.MergeState, branchConfig config.BranchConfig, resolvedOptions *config.ResolvedFinishOptions, mergeOptions *config.MergeStrategyOptions) error {
	// Handle continuation based on current step
	switch state.CurrentStep {
	case stepMerge:
		// For merge step continuation, check if conflicts are resolved
		if git.HasConflicts() {
			return &errors.UnresolvedConflictsError{}
		}

		// Complete the merge/rebase operation based on strategy
		var err error
		switch state.MergeStrategy {
		case strategyRebase:
			// Continue the rebase operation
			err = git.RebaseContinue()
			if err != nil {
				// Check if rebase is complete or if there are more commits to rebase
				if strings.Contains(err.Error(), "No rebase in progress") {
					// Rebase is already complete, proceed
				} else if strings.Contains(err.Error(), "conflict") {
					// More conflicts in subsequent commits
					return &errors.UnresolvedConflictsError{}
				} else {
					return &errors.GitError{Operation: "continue rebase", Err: err}
				}
			}

			// After successful rebase, checkout target and merge
			err = git.Checkout(state.ParentBranch)
			if err != nil {
				return &errors.GitError{Operation: "checkout target branch after rebase", Err: err}
			}
			// Use custom merge message if provided (from CLI or saved state), otherwise use default
			mergeMsg := state.MergeMessage
			if mergeOptions != nil && mergeOptions.MergeMessage != nil && *mergeOptions.MergeMessage != "" {
				mergeMsg = *mergeOptions.MergeMessage
			}
			if mergeMsg != "" {
				expandedMsg := util.ExpandMessagePlaceholders(mergeMsg, state.FullBranchName, state.ParentBranch)
				err = git.MergeWithMessage(state.FullBranchName, expandedMsg, resolvedOptions.NoFastForward, state.NoVerify)
			} else {
				err = git.MergeWithOptions(state.FullBranchName, resolvedOptions.NoFastForward, state.NoVerify)
			}
			if err != nil {
				return &errors.GitError{Operation: "merge rebased branch", Err: err}
			}

		case strategySquash:
			// For squash merge, commit the staged changes
			// Use CLI-provided message if given, otherwise use saved state message
			squashMsg := state.SquashMessage
			if mergeOptions != nil && mergeOptions.SquashMessage != nil && *mergeOptions.SquashMessage != "" {
				squashMsg = *mergeOptions.SquashMessage
			}
			err = git.Commit(squashMsg, state.NoVerify)
			if err != nil {
				return &errors.GitError{Operation: "commit squashed changes", Err: err}
			}

		case strategyMerge:
			// Complete the merge by committing
			// Use custom merge message if provided (from CLI or saved state), otherwise use default
			mergeMsg := state.MergeMessage
			if mergeOptions != nil && mergeOptions.MergeMessage != nil && *mergeOptions.MergeMessage != "" {
				mergeMsg = *mergeOptions.MergeMessage
			}
			if mergeMsg == "" {
				mergeMsg = fmt.Sprintf("Merge branch '%s' into %s", state.FullBranchName, state.ParentBranch)
			} else {
				mergeMsg = util.ExpandMessagePlaceholders(mergeMsg, state.FullBranchName, state.ParentBranch)
			}
			err = git.Commit(mergeMsg, state.NoVerify)
			if err != nil {
				return &errors.GitError{Operation: "commit merge", Err: err}
			}

		default:
			return &errors.GitError{Operation: fmt.Sprintf("unknown merge strategy: %s", state.MergeStrategy), Err: nil}
		}

		// Move to next step since merge conflicts are resolved and committed
		state.CurrentStep = stepCreateTag
		if err := mergestate.SaveMergeState(state); err != nil {
			return &errors.GitError{Operation: "save merge state", Err: err}
		}

	case stepUpdateChildren:
		// For child branch update continuation, check if conflicts are resolved
		if git.HasConflicts() {
			return &errors.UnresolvedConflictsError{}
		}

		// Determine which child branch we're updating
		currentChild := state.CurrentChildBranch
		if currentChild == "" {
			// Try to determine from current branch
			currentBranch, err := git.GetCurrentBranch()
			if err != nil {
				return &errors.GitError{Operation: "get current branch", Err: err}
			}
			currentChild = currentBranch
		}

		// Get the strategy for this child branch
		strategy := ""
		if state.ChildStrategies != nil {
			strategy = state.ChildStrategies[currentChild]
		}
		if strategy == "" {
			// Fallback: try to get from config
			if childConfig, ok := cfg.Branches[currentChild]; ok {
				strategy = childConfig.DownstreamStrategy
			} else {
				// Default to merge if we can't determine
				strategy = "merge"
			}
		}

		// Complete the operation based on strategy
		var err error
		switch strategy {
		case "rebase":
			// Continue the rebase operation
			err = git.RebaseContinue()
			if err != nil {
				if strings.Contains(err.Error(), "No rebase in progress") {
					// Rebase might be complete, try to proceed
					err = nil
				} else if strings.Contains(err.Error(), "conflict") {
					// More conflicts in subsequent commits
					return &errors.UnresolvedConflictsError{}
				} else {
					return &errors.GitError{Operation: "continue rebase for child update", Err: err}
				}
			}

		case "squash":
			// Commit the squashed changes
			// Use custom update message if provided (from CLI or saved state), otherwise use default
			updateMsg := state.UpdateMessage
			if mergeOptions != nil && mergeOptions.UpdateMessage != nil && *mergeOptions.UpdateMessage != "" {
				updateMsg = *mergeOptions.UpdateMessage
			}
			if updateMsg == "" {
				updateMsg = fmt.Sprintf("Update %s: squashed changes from %s",
					currentChild, state.ParentBranch)
			} else {
				// For child updates, the "branch" is the child and "parent" is the source
				updateMsg = util.ExpandMessagePlaceholders(updateMsg, currentChild, state.ParentBranch)
			}
			err = git.Commit(updateMsg, state.NoVerify)
			if err != nil {
				return &errors.GitError{Operation: "commit squashed child update", Err: err}
			}

		default: // "merge" or unknown
			// Complete the merge
			// Use custom update message if provided (from CLI or saved state), otherwise use default
			updateMsg := state.UpdateMessage
			if mergeOptions != nil && mergeOptions.UpdateMessage != nil && *mergeOptions.UpdateMessage != "" {
				updateMsg = *mergeOptions.UpdateMessage
			}
			if updateMsg == "" {
				updateMsg = fmt.Sprintf("Merge branch '%s' into %s",
					state.ParentBranch, currentChild)
			} else {
				// For child updates, the "branch" is the child and "parent" is the source
				updateMsg = util.ExpandMessagePlaceholders(updateMsg, currentChild, state.ParentBranch)
			}
			err = git.Commit(updateMsg, state.NoVerify)
			if err != nil {
				return &errors.GitError{Operation: "commit child branch update", Err: err}
			}
		}

		// Mark this child as updated if not already done
		if !isChildUpdated(state, currentChild) {
			state.UpdatedBranches = append(state.UpdatedBranches, currentChild)
		}
		state.CurrentChildBranch = "" // Clear current child

		// Save state and continue
		if err := mergestate.SaveMergeState(state); err != nil {
			return &errors.GitError{Operation: "save merge state", Err: err}
		}
	}

	return executeSteps(cfg, state, branchConfig, resolvedOptions)
}

func handleAbort(state *mergestate.MergeState) error {
	// Abort the merge based on strategy
	var err error
	switch state.MergeStrategy {
	case strategyMerge:
		err = git.MergeAbort()
	case strategyRebase:
		err = git.RebaseAbort()
	default:
		err = git.MergeAbort() // Default to merge abort
	}

	if err != nil {
		return &errors.GitError{Operation: "abort merge", Err: err}
	}

	// Checkout the original branch
	if err := git.Checkout(state.FullBranchName); err != nil {
		return &errors.GitError{Operation: fmt.Sprintf("checkout original branch '%s'", state.FullBranchName), Err: err}
	}

	// Clear the merge state
	if err := mergestate.ClearMergeState(); err != nil {
		return &errors.GitError{Operation: "clear merge state", Err: err}
	}

	return nil
}

// =============================================================================
// STEP HANDLERS (Called by state machine)
// =============================================================================

// handleMergeStep handles the merge step of the finish operation
func handleMergeStep(cfg *config.Config, state *mergestate.MergeState, branchConfig config.BranchConfig, resolvedOptions *config.ResolvedFinishOptions) error {
	// Checkout target branch
	err := git.Checkout(state.ParentBranch)
	if err != nil {
		return &errors.GitError{Operation: fmt.Sprintf("checkout target branch '%s'", state.ParentBranch), Err: err}
	}
	fmt.Printf("Switched to branch '%s'\n", state.ParentBranch)

	// Update state with the resolved strategy (might be different from branch default)
	state.MergeStrategy = resolvedOptions.MergeStrategy

	// Perform merge based on resolved strategy
	fmt.Printf("Merging using strategy: %v\n", resolvedOptions.MergeStrategy)
	var mergeErr error
	switch resolvedOptions.MergeStrategy {
	case strategyRebase:
		fmt.Printf("Rebase strategy selected\n")
		// For rebase, we need to:
		// 1. Stay on feature branch
		err = git.Checkout(state.FullBranchName)
		if err != nil {
			return &errors.GitError{Operation: "checkout feature branch for rebase", Err: err}
		}
		// 2. Rebase onto target branch with options
		mergeErr = git.RebaseWithOptions(state.ParentBranch, resolvedOptions.PreserveMerges)
		if mergeErr == nil {
			// 3. If rebase succeeds, checkout target and merge
			err = git.Checkout(state.ParentBranch)
			if err != nil {
				return &errors.GitError{Operation: "checkout target branch after rebase", Err: err}
			}
			// Use custom merge message if provided, otherwise use default
			if resolvedOptions.MergeMessage != "" {
				expandedMsg := util.ExpandMessagePlaceholders(resolvedOptions.MergeMessage, state.FullBranchName, state.ParentBranch)
				mergeErr = git.MergeWithMessage(state.FullBranchName, expandedMsg, resolvedOptions.NoFastForward, resolvedOptions.NoVerify)
			} else {
				mergeErr = git.MergeWithOptions(state.FullBranchName, resolvedOptions.NoFastForward, resolvedOptions.NoVerify)
			}
		}
	case strategySquash:
		mergeErr = git.MergeSquashWithMessage(state.FullBranchName, resolvedOptions.SquashMessage, resolvedOptions.NoVerify)
	case strategyMerge:
		if resolvedOptions.MergeMessage != "" {
			expandedMsg := util.ExpandMessagePlaceholders(resolvedOptions.MergeMessage, state.FullBranchName, state.ParentBranch)
			mergeErr = git.MergeWithMessage(state.FullBranchName, expandedMsg, resolvedOptions.NoFastForward, resolvedOptions.NoVerify)
		} else {
			mergeErr = git.MergeWithOptions(state.FullBranchName, resolvedOptions.NoFastForward, resolvedOptions.NoVerify)
		}
	default:
		return &errors.GitError{Operation: fmt.Sprintf("unknown merge strategy: %s", resolvedOptions.MergeStrategy), Err: nil}
	}

	if mergeErr != nil {
		if strings.Contains(mergeErr.Error(), "conflict") {
			// Save state before returning conflict error
			state.CurrentStep = stepMerge
			if err := mergestate.SaveMergeState(state); err != nil {
				return &errors.GitError{Operation: "save merge state", Err: err}
			}

			// Generate and print detailed conflict message
			msg := generateConflictMessage(state, cfg, resolvedOptions)
			fmt.Println(msg)
			return &errors.UnresolvedConflictsError{}
		}
		return &errors.GitError{Operation: "merge branch", Err: mergeErr}
	}

	// Move to next step (tag creation)
	state.CurrentStep = stepCreateTag
	if err := mergestate.SaveMergeState(state); err != nil {
		return &errors.GitError{Operation: "save merge state", Err: err}
	}

	return nil
}

// handleCreateTagStep handles the tag creation step
func handleCreateTagStep(cfg *config.Config, state *mergestate.MergeState, resolvedOptions *config.ResolvedFinishOptions) error {
	if resolvedOptions.ShouldTag {
		// Apply tag message filter for any branch type configured with tagging
		// The filter script (filter-flow-{branchType}-finish-tag-message) decides what to do
		gitDir, err := git.GetGitDir()
		if err != nil {
			return &errors.GitError{Operation: "get git directory", Err: err}
		}

		remote := cfg.Remote

		ctx := hooks.FilterContext{
			BranchType: state.BranchType,
			BranchName: state.BranchName,
			Version:    resolvedOptions.TagName,
			TagMessage: resolvedOptions.TagMessage,
			BaseBranch: state.ParentBranch,
			FullBranch: state.FullBranchName,
			Origin:     remote,
		}

		filteredMessage, err := hooks.RunTagMessageFilter(gitDir, state.BranchType, ctx)
		if err != nil {
			return &errors.GitError{Operation: "run tag message filter", Err: err}
		}
		if filteredMessage != resolvedOptions.TagMessage {
			fmt.Printf("Tag message filter modified the message\n")
			resolvedOptions.TagMessage = filteredMessage
		}

		if err := createTagForBranchResolved(state, resolvedOptions); err != nil {
			return err
		}
	}

	// Move to next step
	state.CurrentStep = stepUpdateChildren
	if err := mergestate.SaveMergeState(state); err != nil {
		return &errors.GitError{Operation: "save merge state", Err: err}
	}
	return nil
}

// handleUpdateChildrenStep handles updating child base branches
func handleUpdateChildrenStep(cfg *config.Config, state *mergestate.MergeState, branchConfig config.BranchConfig, resolvedOptions *config.ResolvedFinishOptions) error {
	// Find next child branch to update
	nextBranch := findNextBranchToUpdate(state)

	// If no more branches to update, move to final step
	if nextBranch == "" {
		state.CurrentStep = stepDeleteBranch
		if err := mergestate.SaveMergeState(state); err != nil {
			return &errors.GitError{Operation: "save merge state", Err: err}
		}
		return nil
	}

	// Update the next child branch
	if err := updateChildBranch(cfg, nextBranch, state); err != nil {
		return err
	}

	// Mark this branch as updated and clear current child
	state.UpdatedBranches = append(state.UpdatedBranches, nextBranch)
	state.CurrentChildBranch = "" // Clear after successful update
	if err := mergestate.SaveMergeState(state); err != nil {
		return &errors.GitError{Operation: "save merge state", Err: err}
	}

	// Continue with next branch
	return nil
}

// handleDeleteBranchStep handles branch deletion
func handleDeleteBranchStep(cfg *config.Config, state *mergestate.MergeState, resolvedOptions *config.ResolvedFinishOptions) error {
	// Ensure we're on the parent branch before deletion
	if err := git.Checkout(state.ParentBranch); err != nil {
		return &errors.GitError{Operation: fmt.Sprintf("checkout parent branch '%s'", state.ParentBranch), Err: err}
	}

	// Clear the merge state before branch deletion. By this point all merges,
	// tags, and child updates are complete — the state is only needed for conflict
	// recovery which is no longer possible. Clearing early ensures a failed branch
	// deletion (e.g. remote permission error) doesn't leave stale merge state.
	if err := mergestate.ClearMergeState(); err != nil {
		return &errors.GitError{Operation: "clear merge state", Err: err}
	}

	// Apply keep logic: if keep is set, it overrides individual settings
	keepRemote := resolvedOptions.KeepRemote
	keepLocal := resolvedOptions.KeepLocal
	if resolvedOptions.Keep {
		keepRemote = true
		keepLocal = true
	}

	// Delete branches based on settings
	// Use force delete since we've already merged the branch
	forceDelete := true
	if err := deleteBranchesIfNeeded(state, cfg.Remote, keepRemote, keepLocal, forceDelete); err != nil {
		return err
	}

	// Clean up base branch configuration if branch was deleted
	if !keepLocal {
		configKey := fmt.Sprintf("gitflow.branch.%s.base", state.FullBranchName)
		if err := git.UnsetConfigIfPresent(configKey); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to clean up base config: %v\n", err)
		}
	}

	// Push stage: runs after all local work is complete and the merge state is
	// cleared. A missing remote is a skip (exit 0); a rejected push is a real
	// error, leaving the completed local finish as-is.
	if err := pushFinishedBranches(cfg, state, resolvedOptions); err != nil {
		return err
	}

	fmt.Printf("Successfully finished branch '%s' and updated %d child base branches\n", state.FullBranchName, len(state.UpdatedBranches))

	// Run post-hook after successful completion
	gitDir, err := git.GetGitDir()
	if err == nil {
		hookCtx := hooks.HookContext{
			BranchType: state.BranchType,
			BranchName: state.BranchName,
			FullBranch: state.FullBranchName,
			BaseBranch: state.ParentBranch,
			Origin:     cfg.Remote,
			ExitCode:   0, // Success
		}
		// Set version for branches configured with tagging
		if resolvedOptions.ShouldTag {
			hookCtx.Version = state.BranchName
		}

		result := hooks.RunPostHook(gitDir, state.BranchType, hooks.HookActionFinish, hookCtx)
		if result.Executed && result.Output != "" {
			fmt.Print(result.Output)
		}
	}

	return nil
}

// pushFinishedBranches pushes the branches modified by the finish (the target
// branch plus each auto-updated child base branch) and the created tag to the
// configured remote, according to the resolved push options. It runs as the final
// stage of a completed finish.
//
// Behavior:
//   - Nothing to push (neither branches nor tag enabled): no-op, no output.
//   - Remote not configured: prints a skip note and returns nil (exit 0).
//   - A push failure (e.g. non-fast-forward rejection): returns the error
//     verbatim, which propagates to a non-zero exit. The completed local finish
//     is left as-is (nothing is rolled back).
func pushFinishedBranches(cfg *config.Config, state *mergestate.MergeState, resolvedOptions *config.ResolvedFinishOptions) error {
	// Only push the tag if one was actually created.
	tagToPush := resolvedOptions.PushTag && resolvedOptions.ShouldTag

	// Nothing to do.
	if !resolvedOptions.PushBranches && !tagToPush {
		return nil
	}

	remote := cfg.Remote

	// A missing remote is a skip-with-note, not an error.
	if !git.RemoteExists(remote) {
		fmt.Printf("Note: Remote '%s' not configured, skipping push\n", remote)
		return nil
	}

	fmt.Printf("Pushing to remote '%s'...\n", remote)

	if resolvedOptions.PushBranches {
		// Ordered, de-duplicated branch list: parent first, then updated children,
		// skipping any child equal to the parent.
		branches := []string{state.ParentBranch}
		for _, child := range state.UpdatedBranches {
			if child != state.ParentBranch {
				branches = append(branches, child)
			}
		}

		for _, branch := range branches {
			if err := git.PushRef(remote, branch); err != nil {
				return &errors.GitError{Operation: fmt.Sprintf("push branch '%s'", branch), Err: err}
			}
			fmt.Printf("  %s -> %s/%s\n", branch, remote, branch)
		}
	}

	if tagToPush {
		if err := git.PushTag(remote, resolvedOptions.TagName); err != nil {
			return &errors.GitError{Operation: fmt.Sprintf("push tag '%s'", resolvedOptions.TagName), Err: err}
		}
		fmt.Printf("  %s (tag) -> %s\n", resolvedOptions.TagName, remote)
	}

	return nil
}

// =============================================================================
// HELPER FUNCTIONS (Called by step handlers and main flow)
// =============================================================================

// resolveBranchName tries to find the branch name with and without prefix
func resolveBranchName(name string, branchConfig config.BranchConfig) (string, error) {
	// Try name as-is first
	if err := git.BranchExists(name); err == nil {
		return name, nil
	}

	// If not found as-is, try with prefix
	if !strings.HasPrefix(name, branchConfig.Prefix) {
		fullName := branchConfig.Prefix + name
		if err := git.BranchExists(fullName); err == nil {
			return fullName, nil
		}
	}

	return "", &errors.BranchNotFoundError{BranchName: name}
}

// createTagForBranchResolved creates a tag using resolved options
func createTagForBranchResolved(state *mergestate.MergeState, options *config.ResolvedFinishOptions) error {
	// Determine if we should use message file
	useMessageFile := options.MessageFile != ""

	// Create the tag using the git module
	gitTagOptions := &git.TagOptions{
		Message:     options.TagMessage,
		MessageFile: options.MessageFile,
		Sign:        options.ShouldSign,
		SigningKey:  options.SigningKey,
	}

	// Use MessageFile if specified, otherwise use Message
	if useMessageFile {
		gitTagOptions.Message = "" // Clear message since we're using file
	} else {
		gitTagOptions.MessageFile = "" // Clear file since we're using message
	}

	if err := git.CreateTag(options.TagName, gitTagOptions); err != nil {
		return &errors.GitError{Operation: fmt.Sprintf("create tag '%s'", options.TagName), Err: err}
	}
	fmt.Printf("Created tag '%s'\n", options.TagName)
	return nil
}

// findNextBranchToUpdate finds the next child branch that needs updating
func findNextBranchToUpdate(state *mergestate.MergeState) string {
	for _, branch := range state.ChildBranches {
		alreadyUpdated := false
		for _, updated := range state.UpdatedBranches {
			if branch == updated {
				alreadyUpdated = true
				break
			}
		}
		if !alreadyUpdated {
			return branch
		}
	}
	return ""
}

// updateChildBranch updates a single child branch
func updateChildBranch(cfg *config.Config, branchName string, state *mergestate.MergeState) error {
	fmt.Printf("Updating child base branch '%s' from '%s'...\n", branchName, state.ParentBranch)

	// Track which child branch we're updating
	state.CurrentChildBranch = branchName
	if err := mergestate.SaveMergeState(state); err != nil {
		return &errors.GitError{Operation: "save merge state", Err: err}
	}

	// Get strategy from saved state if available, fallback to config
	strategy := ""
	if state.ChildStrategies != nil {
		strategy = state.ChildStrategies[branchName]
	}

	if strategy == "" {
		// Fallback to current config if not in state
		childBranchConfig, ok := cfg.Branches[branchName]
		if !ok {
			return &errors.GitError{Operation: fmt.Sprintf("get config for branch '%s'", branchName), Err: fmt.Errorf("branch config not found")}
		}
		strategy = childBranchConfig.DownstreamStrategy
	}

	// Expand placeholders in update message if provided
	// For child updates, the "branch" is the child and "parent" is the source
	updateMsg := state.UpdateMessage
	if updateMsg != "" {
		updateMsg = util.ExpandMessagePlaceholders(updateMsg, branchName, state.ParentBranch)
	}

	// Use the shared update logic with the determined strategy and custom message if provided
	err := update.UpdateBranchFromParentWithMessage(branchName, state.ParentBranch, strategy, updateMsg, true, state)
	if err != nil {
		if _, ok := err.(*errors.UnresolvedConflictsError); ok {
			// Get resolved options for the message (might be nil, but generateConflictMessage handles that)
			var resolvedOptions *config.ResolvedFinishOptions
			if cfg != nil {
				// Try to resolve options for better tag information in message
				resolvedOptions = config.ResolveFinishOptions(cfg, state.BranchType, state.BranchName, nil, nil, nil, nil, nil, nil, nil)
			}

			// Generate and print detailed conflict message
			msg := generateConflictMessage(state, cfg, resolvedOptions)
			fmt.Println(msg)
			return err
		}
		return err
	}

	return nil
}

// deleteBranchesIfNeeded deletes branches based on retention settings
func deleteBranchesIfNeeded(state *mergestate.MergeState, remote string, keepRemote, keepLocal, forceDelete bool) error {
	// Delete remote branch if not keeping it and if remote branch exists
	if !keepRemote {
		// Only attempt to delete if the remote branch actually exists
		if git.RemoteBranchExists(remote, state.FullBranchName) {
			remoteBranch := fmt.Sprintf("%s/%s", remote, state.FullBranchName)
			if err := git.DeleteRemoteBranch(remote, state.FullBranchName); err != nil {
				return &errors.GitError{Operation: fmt.Sprintf("delete remote branch '%s'", remoteBranch), Err: err}
			}
		}
	}

	// Delete local branch if not keeping it
	if !keepLocal {
		if err := git.DeleteBranch(state.FullBranchName, forceDelete); err != nil {
			return &errors.GitError{Operation: fmt.Sprintf("delete branch '%s'", state.FullBranchName), Err: err}
		}
	}

	return nil
}

// isChildUpdated checks if a child branch has already been marked as updated
func isChildUpdated(state *mergestate.MergeState, childName string) bool {
	for _, updated := range state.UpdatedBranches {
		if updated == childName {
			return true
		}
	}
	return false
}

// generateConflictMessage generates a human-readable conflict message with progress information
func generateConflictMessage(state *mergestate.MergeState, cfg *config.Config, resolvedOptions *config.ResolvedFinishOptions) string {
	var msg strings.Builder

	// Header
	msg.WriteString(fmt.Sprintf("Merge conflict detected while finishing %s/%s\n\n", state.BranchType, state.BranchName))

	// What happened section
	msg.WriteString("What happened:\n")
	if state.CurrentStep == stepMerge {
		msg.WriteString(fmt.Sprintf("  Trying to merge '%s' into '%s' using %s strategy\n", state.FullBranchName, state.ParentBranch, state.MergeStrategy))
	} else if state.CurrentStep == stepUpdateChildren && state.CurrentChildBranch != "" {
		msg.WriteString(fmt.Sprintf("  Successfully merged '%s' into '%s'\n", state.FullBranchName, state.ParentBranch))
		strategy := "merge"
		if state.ChildStrategies != nil && state.ChildStrategies[state.CurrentChildBranch] != "" {
			strategy = state.ChildStrategies[state.CurrentChildBranch]
		}
		msg.WriteString(fmt.Sprintf("  Now updating '%s' from '%s' using %s strategy\n", state.CurrentChildBranch, state.ParentBranch, strategy))
	}

	// Where we are section - show all steps as natural progression
	msg.WriteString("\nWhere we are:\n")
	msg.WriteString("  ✓ Started finish operation\n")

	// Merge step
	if state.CurrentStep == stepMerge {
		msg.WriteString(fmt.Sprintf("  ✗ Merge into %s (conflict here)\n", state.ParentBranch))
	} else {
		msg.WriteString(fmt.Sprintf("  ✓ Merged into %s\n", state.ParentBranch))
	}

	// Tag step (only show if tags will be created)
	if state.CurrentStep == stepCreateTag || state.CurrentStep == stepUpdateChildren || state.CurrentStep == stepDeleteBranch {
		if resolvedOptions != nil && resolvedOptions.ShouldTag {
			msg.WriteString(fmt.Sprintf("  ✓ Created tag '%s'\n", resolvedOptions.TagName))
		}
	} else if resolvedOptions != nil && resolvedOptions.ShouldTag {
		msg.WriteString(fmt.Sprintf("  ⧖ Create tag '%s'\n", resolvedOptions.TagName))
	}

	// Child branch updates - show each as individual step
	if len(state.ChildBranches) > 0 {
		for _, child := range state.ChildBranches {
			isUpdated := isChildUpdated(state, child)
			isCurrentConflict := state.CurrentStep == stepUpdateChildren && state.CurrentChildBranch == child

			if isUpdated {
				msg.WriteString(fmt.Sprintf("  ✓ Update %s from %s\n", child, state.ParentBranch))
			} else if isCurrentConflict {
				msg.WriteString(fmt.Sprintf("  ✗ Update %s from %s (conflict here)\n", child, state.ParentBranch))
			} else {
				msg.WriteString(fmt.Sprintf("  ⧖ Update %s from %s\n", child, state.ParentBranch))
			}
		}
	}

	// Delete branch step
	if state.CurrentStep == stepDeleteBranch {
		msg.WriteString(fmt.Sprintf("  ✓ Delete %s branch\n", state.BranchType))
	} else {
		msg.WriteString(fmt.Sprintf("  ⧖ Delete %s branch\n", state.BranchType))
	}

	// Resolution instructions
	msg.WriteString("\nTo continue:\n")
	msg.WriteString("  1. Resolve the conflicts in your files\n")
	msg.WriteString("  2. Stage resolved files: git add <files>\n")
	msg.WriteString(fmt.Sprintf("  3. Continue: git flow %s finish --continue %s\n", state.BranchType, state.BranchName))
	msg.WriteString(fmt.Sprintf("\nTo abort: git flow %s finish --abort %s", state.BranchType, state.BranchName))

	return msg.String()
}
