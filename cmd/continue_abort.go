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
// non-destructively via UnrecognizedOperationError and never auto-cleared. The
// same applies when the state file exists but fails to parse (e.g. a truncated
// write from a crash mid-save): if git is genuinely mid-operation the file must
// not be destroyed, so it is refused rather than left to IsMergeInProgress, which
// would delete it while the git marker is still present.
func refuseIfForeignOperation(cfg *config.Config, currentCommand string) error {
	rawState, loadErr := mergestate.LoadMergeState()

	gitInProgress := git.IsGitMergeInProgress() || git.IsGitRebaseInProgress() || git.IsGitSquashMergeInProgress()

	if loadErr != nil {
		// The state file exists but could not be parsed. If a real git operation
		// is underway, refuse non-destructively so IsMergeInProgress does not
		// auto-clear the file while the marker is still present. If nothing is in
		// progress, it is a truly stale corrupt file with no active operation to
		// protect — leave it to the normal auto-clear path.
		if gitInProgress {
			return &errors.UnrecognizedOperationError{}
		}
		return nil
	}

	if rawState == nil {
		return nil
	}

	if !gitInProgress {
		// No real operation underway: a stale state file. Let the normal
		// IsMergeInProgress path detect and auto-clear it.
		return nil
	}

	// Structural-completeness guard (mirrors the loadErr guard above). A state can
	// parse and carry a recognized Action yet still be missing a critical field —
	// e.g. a legacy pre-#143 update state with Action="update" but an empty
	// BranchType. Such a state matches the owner check below and returns nil, after
	// which the caller's IsMergeInProgress rejects the incomplete state via
	// isStateValid and auto-clears the file while the git marker is still present —
	// a destructive refusal. These are exactly the critical fields isStateValid
	// requires, so refuse non-destructively before the owner check whenever any is
	// empty. Valid new update/finish/integrate states always populate all three, so
	// they fall through to the owner/foreign dispatch unchanged.
	if rawState.BranchType == "" || rawState.FullBranchName == "" || rawState.CurrentStep == "" {
		return &errors.UnrecognizedOperationError{BranchName: rawState.FullBranchName}
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
