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
)

// Note: The update command is registered in two places:
// 1. As a shorthand command in shorthand.go for "git flow update"
// 2. As subcommands of topic branches in topicbranch.go for "git flow <topic> update"
// This file only contains the shared executeUpdate function used by both.

// UpdateCommand is the public entry point for the update command on both
// surfaces (topic "git flow <type> update" and top-level "git flow update"). It
// maps git-flow errors to their exit codes and exits non-zero on failure. Routing
// through this wrapper (instead of returning the error via RunE to main.go, which
// forces exit 1) is what makes the top-level surface exit 3 for merge-in-progress,
// no-merge, and unresolved-conflict conditions, consistent with finish/integrate.
func UpdateCommand(branchType string, name string, useRebase, continueOp, abortOp bool) {
	repo := mustOpenRepo()
	if err := executeUpdate(repo, branchType, name, useRebase, continueOp, abortOp); err != nil {
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

// executeUpdate updates a branch with changes from its parent branch. It owns the
// resumable dispatch for update: a foreign-operation guard, its own
// --continue/--abort handling, and the initial update attempt.
func executeUpdate(repo *git.Repo, branchType string, name string, useRebase, continueOp, abortOp bool) error {
	// Validate that git-flow is initialized
	initialized, err := config.IsInitialized(repo)
	if err != nil {
		return &errors.GitError{Operation: "check if git-flow is initialized", Err: err}
	}
	if !initialized {
		return &errors.NotInitializedError{}
	}

	// Get configuration
	cfg, err := config.Load(repo)
	if err != nil {
		return &errors.GitError{Operation: "load configuration", Err: err}
	}

	// Foreign-operation guard (#143): refuse a foreign in-progress finish/integrate
	// (or an unknown-Action state) before dispatch, so update never resumes or
	// aborts an operation it does not own.
	if err := refuseIfForeignOperation(repo, cfg, "update"); err != nil {
		return err
	}

	// Own-state dispatch: resume/abort an update we own.
	if mergestate.IsMergeInProgress(repo) {
		state, err := mergestate.LoadMergeState(repo)
		if err != nil {
			return &errors.GitError{Operation: "load merge state", Err: err}
		}
		// Belt-and-suspenders: the guard above already refused foreign state.
		if state.Action != "update" {
			return &errors.MergeInProgressError{Action: state.Action, BranchName: state.FullBranchName, BranchType: topicTypeOrEmpty(cfg, state.BranchType)}
		}
		if abortOp {
			return handleAbort(repo, state)
		}
		if continueOp {
			return handleUpdateContinue(repo, state)
		}
		// Plain invocation over our own in-progress update: report it rather than
		// silently restarting.
		return &errors.MergeInProgressError{Action: "update", BranchName: state.FullBranchName, BranchType: topicTypeOrEmpty(cfg, state.BranchType)}
	}

	// Nothing in progress: abort is a forgiving no-op, continue has nothing to
	// resume (matches finish/integrate dispatch).
	if abortOp {
		return nil
	}
	if continueOp {
		return &errors.NoMergeInProgressError{}
	}

	var branchName string
	var shortName string
	var detectedBranchType string

	if branchType != "" {
		detectedBranchType = branchType
		// If branch type is specified, construct full branch name
		if name == "" {
			// If no name provided, try to get current branch and verify it's of the correct type
			currentBranch, err := repo.GetCurrentBranch()
			if err != nil {
				return &errors.GitError{Operation: "get current branch", Err: err}
			}
			branchConfig, ok := cfg.Branches[branchType]
			if !ok {
				return &errors.InvalidBranchTypeError{BranchType: branchType}
			}
			if !strings.HasPrefix(currentBranch, branchConfig.Prefix) {
				return &errors.GitError{Operation: "validate current branch", Err: fmt.Errorf("current branch is not a %s branch", branchType)}
			}
			branchName = currentBranch
			shortName = strings.TrimPrefix(currentBranch, branchConfig.Prefix)
		} else {
			// Construct full branch name from type and name
			branchConfig, ok := cfg.Branches[branchType]
			if !ok {
				return &errors.InvalidBranchTypeError{BranchType: branchType}
			}
			branchName = branchConfig.Prefix + name
			shortName = name
		}
	} else {
		// No branch type specified, use provided branch name or current branch
		if name == "" {
			currentBranch, err := repo.GetCurrentBranch()
			if err != nil {
				return &errors.GitError{Operation: "get current branch", Err: err}
			}
			branchName = currentBranch
		} else {
			branchName = name
		}
		// Try to detect branch type from branch name
		detectedBranchType, shortName = detectBranchTypeFromName(cfg, branchName)
	}

	// Check if branch exists
	if err := repo.BranchExists(branchName); err != nil {
		return &errors.BranchNotFoundError{BranchName: branchName}
	}

	// Get parent branch
	parentBranch, err := update.GetParentBranch(cfg, branchName)
	if err != nil {
		return err
	}

	// Check if parent branch exists
	if err := repo.BranchExists(parentBranch); err != nil {
		return &errors.BranchNotFoundError{BranchName: parentBranch}
	}

	// Get branch configuration for merge strategy
	var strategy string
	for branchKey, bc := range cfg.Branches {
		if bc.Type == string(config.BranchTypeBase) && branchKey == branchName {
			strategy = bc.DownstreamStrategy
			break
		}
		if bc.Type == string(config.BranchTypeTopic) && bc.Prefix != "" && strings.HasPrefix(branchName, bc.Prefix) {
			strategy = bc.DownstreamStrategy
			break
		}
	}

	if strategy == "" {
		strategy = "merge" // Default to merge if no strategy configured
	}

	// Override strategy if --rebase flag is set
	if useRebase {
		strategy = "rebase"
	}

	// Store a resolvable identity in BranchType so the saved state survives
	// isStateValid and --continue/--abort can re-derive parent/strategy: the
	// detected topic type for topic branches, else the base branch key.
	stateBranchType := detectedBranchType
	if stateBranchType == "" {
		if _, ok := cfg.Branches[branchName]; ok {
			stateBranchType = branchName
		}
	}

	// Create merge state
	state := &mergestate.MergeState{
		Action:         "update",
		BranchType:     stateBranchType,
		BranchName:     branchName,
		ParentBranch:   parentBranch,
		MergeStrategy:  strategy,
		CurrentStep:    "merge",
		FullBranchName: branchName,
	}

	// If we detected a branch type, run with hooks
	if detectedBranchType != "" {
		// Get remote name from config
		remoteName := cfg.Remote

		// Build hook context
		hookCtx := hooks.HookContext{
			BranchType: detectedBranchType,
			BranchName: shortName,
			FullBranch: branchName,
			BaseBranch: parentBranch,
			Origin:     remoteName,
		}
		if detectedBranchType == "release" || detectedBranchType == "hotfix" {
			hookCtx.Version = shortName
		}

		// Run update operation wrapped with hooks
		return hooks.WithHooks(repo, detectedBranchType, hooks.HookActionUpdate, hookCtx, func() error {
			return update.UpdateBranchFromParent(repo, branchName, parentBranch, strategy, true, state)
		})
	}

	// No branch type detected, run without hooks
	return update.UpdateBranchFromParent(repo, branchName, parentBranch, strategy, true, state)
}

// handleUpdateContinue completes an in-progress update after conflict resolution.
// Unlike finish, it completes ONLY the merge/rebase — no tag, no child update, no
// branch deletion — then clears the merge state. A rebase that re-conflicts on a
// later replayed commit stays resumable (UnresolvedConflictsError, state kept).
func handleUpdateContinue(repo *git.Repo, state *mergestate.MergeState) error {
	if repo.HasConflicts() {
		return &errors.UnresolvedConflictsError{}
	}

	switch state.MergeStrategy {
	case strategyRebase:
		err := repo.RebaseContinue()
		if err != nil {
			if strings.Contains(err.Error(), "conflict") {
				// A later replayed commit conflicts: stay resumable.
				return &errors.UnresolvedConflictsError{}
			}
			return &errors.GitError{Operation: "continue rebase", Err: err}
		}

	case strategySquash:
		// Squash-strategy update resume is a known gap: there is no MERGE_HEAD to
		// commit and handleAbort cannot roll it back. Do not attempt completion.
		return &errors.GitError{
			Operation: "continue squash-strategy update",
			Err:       fmt.Errorf("resuming a squash-strategy update is not supported; complete the commit manually or run 'git reset --merge' and re-run the update"),
		}

	default: // merge
		mergeMsg := fmt.Sprintf("Merge branch '%s' into %s", state.ParentBranch, state.FullBranchName)
		if err := repo.Commit(mergeMsg, state.NoVerify); err != nil {
			return &errors.GitError{Operation: "commit merge", Err: err}
		}
	}

	if err := mergestate.ClearMergeState(repo); err != nil {
		return &errors.GitError{Operation: "clear merge state", Err: err}
	}

	fmt.Printf("Successfully updated branch '%s' from '%s'\n", state.FullBranchName, state.ParentBranch)
	return nil
}

// detectBranchTypeFromName detects the branch type and short name from a full branch name
func detectBranchTypeFromName(cfg *config.Config, branchName string) (string, string) {
	for branchType, bc := range cfg.Branches {
		if bc.Type == string(config.BranchTypeTopic) && bc.Prefix != "" && strings.HasPrefix(branchName, bc.Prefix) {
			return branchType, strings.TrimPrefix(branchName, bc.Prefix)
		}
	}
	return "", branchName
}
