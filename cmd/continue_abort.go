package cmd

import (
	"github.com/gittower/git-flow-next/internal/config"
	"github.com/gittower/git-flow-next/internal/errors"
	"github.com/gittower/git-flow-next/internal/git"
	"github.com/gittower/git-flow-next/internal/mergestate"
)

// refuseIfForeignOperation enforces the issue #143 invariant: a resumable command
// (finish, update, integrate) must not touch an in-progress operation owned by a
// different command. It reads the raw merge state (without the stale-clearing side
// effect of IsMergeInProgress, which would drop a legitimately-in-progress state)
// and, only when git is genuinely mid-operation, refuses a foreign owner with an
// actionable, owner-named error. It returns nil when there is nothing to refuse:
// no state, no real git operation in progress (stale state is left for the normal
// IsMergeInProgress auto-clear path), or the state is owned by currentCommand.
//
// An unknown or empty Action with a real git operation in progress is refused
// non-destructively via UnrecognizedOperationError and never auto-cleared.
func refuseIfForeignOperation(cfg *config.Config, currentCommand string) error {
	rawState, _ := mergestate.LoadMergeState()
	if rawState == nil {
		return nil
	}

	gitInProgress := git.IsGitMergeInProgress() || git.IsGitRebaseInProgress() || git.IsGitSquashMergeInProgress()
	if !gitInProgress {
		// No real operation underway: a stale state file. Let the normal
		// IsMergeInProgress path detect and auto-clear it.
		return nil
	}

	if rawState.Action == currentCommand {
		// Own operation: the caller proceeds to its own continue/abort dispatch.
		return nil
	}

	switch rawState.Action {
	case "finish", "update", "integrate":
		return &errors.MergeInProgressError{
			Action:     rawState.Action,
			BranchName: rawState.FullBranchName,
			BranchType: topicTypeOrEmpty(cfg, rawState.BranchType),
		}
	default:
		return &errors.UnrecognizedOperationError{BranchName: rawState.FullBranchName}
	}
}

// topicTypeOrEmpty returns branchType when it names a configured topic branch
// type, otherwise "". The owner-aware refusal message uses this to recommend the
// topic-scoped command (git flow <type> update) only for real topic types, and
// the base/top-level command (git flow update) for base branches.
func topicTypeOrEmpty(cfg *config.Config, branchType string) string {
	if bc, ok := cfg.Branches[branchType]; ok && bc.Type == string(config.BranchTypeTopic) {
		return branchType
	}
	return ""
}
