package cmd

import (
	goerrors "errors"
	"fmt"

	"github.com/gittower/git-flow-next/internal/config"
	"github.com/gittower/git-flow-next/internal/errors"
	"github.com/gittower/git-flow-next/internal/git"
)

// preflightOptions tunes runFetchSyncPreflight for a caller's pre-merge guard. The zero value is the
// strictest guard (fatal on any out-of-sync topic and any fetch failure); each field opts into
// relaxed or extra behavior. finish tolerates ahead; delete tolerates ahead and relaxes fetch
// failures and fast-forwards the parent (see #88).
type preflightOptions struct {
	// ffParent fast-forwards the local parent to its freshly fetched remote-tracking branch. This
	// is the #88 hook: it lets a branch that was merged remotely (e.g. via a PR) be recognized as
	// merged by `git branch -d`. `git merge --ff-only` operates on HEAD, so the fast-forward is
	// applied only when the parent is the current branch; otherwise it is a no-op. finish does not
	// fast-forward the parent.
	ffParent bool

	// tolerateAhead downgrades a topic that is *ahead* of its remote (local-only commits) from a
	// fatal out-of-sync error to a note. Both finish and delete set it: neither loses the local
	// commits by proceeding — finish merges them into the parent before deleting the topic, and a
	// non-force `git branch -d` still guards genuinely unmerged commits. Only *behind*/diverged
	// (losing remote work) stays fatal.
	tolerateAhead bool

	// fetchFailureNonFatal downgrades a transport/auth fetch failure from fatal to a note. delete's
	// --fetch is an opt-in convenience, so an unreachable remote must not block a local deletion
	// (matching how start treats fetch failures). finish aborts on a fatal fetch failure unless
	// forced.
	fetchFailureNonFatal bool

	// operation names the command driving the preflight ("finish" or "delete"). It is propagated
	// into BranchNotInSyncError so a sync abort suggests the right `git flow <type> <op> --force`
	// command. Empty renders as finish (the zero-value default).
	operation string

	// parentSyncCheck verifies the parent (merge-target) branch is in sync with its remote before
	// the caller merges into it (#99). Unlike the topic check, the parent aborts only when it is
	// *behind* or *diverged* (merging onto a stale base) and proceeds when ahead — being locally
	// ahead is the normal state right after a previous unpushed finish. finish sets it; delete does
	// not (delete fast-forwards the parent via ffParent instead, so the two are mutually exclusive).
	parentSyncCheck bool
}

// runFetchSyncPreflight guards a topic operation against an out-of-date topic branch. It fetches
// the topic (and, best-effort, the parent) and then verifies the topic is in sync with its remote.
// Callers tune the behavior with opts; the zero value is the strictest guard (finish and delete
// both relax it, at least by tolerating ahead).
//
// Behavior:
//   - shouldFetch=false or no remote configured: fetch is skipped (no "Fetching" line). The topic
//     sync check still runs against existing tracking data (so --no-fetch does not skip it).
//   - Topic fetch reports a missing remote ref (never pushed / deleted after a remote merge): the
//     missing ref is benign — the stale tracking ref is pruned and the sync check is skipped.
//   - Topic fetch fails with a transport/auth error: fatal (FetchFailedError) unless force, or a
//     non-fatal note when opts.fetchFailureNonFatal is set.
//   - Topic behind/diverged from its remote: fatal (BranchNotInSyncError) unless force. Ahead is
//     fatal unless force too, unless opts.tolerateAhead downgrades it to a note (finish and delete
//     both set it).
//
// The parent is fetched best-effort. When opts.parentSyncCheck is set (finish) it is then
// sync-checked with a laxer rule than the topic — behind/diverged abort, ahead and equal proceed
// (#99). When opts.ffParent is set instead (delete) and the parent is the current branch, it is
// fast-forwarded from its remote (#88). The two are mutually exclusive.
func runFetchSyncPreflight(cfg *config.Config, branchType, remote, topicBranch, shortName, parentBranch string, shouldFetch, force bool, opts preflightOptions) error {
	// topicRefFound tracks whether the topic still has a remote ref to compare against. It stays
	// true when the fetch is skipped (we then rely on existing tracking data).
	topicRefFound := true

	// parentRefFound mirrors topicRefFound for the parent sync check (#99): a benign missing remote
	// ref (never pushed / deleted after a remote merge) leaves a stale tracking ref we must not
	// compare against. It stays true when the fetch is skipped.
	parentRefFound := true

	if shouldFetch && git.RemoteExists(remote) {
		fmt.Printf("Fetching from remote '%s'...\n", remote)

		// Fetch the parent best-effort. A benign missing remote ref means any local tracking ref is
		// stale, so the parent sync check is skipped against it (#99); a transport/auth failure is a
		// non-fatal note here because the same failure is already fatal at the topic fetch below.
		// Skip entirely when the branch type has no configured parent, so we never issue
		// `git fetch <remote> ""` or emit a spurious note.
		if parentBranch != "" {
			if err := git.FetchBranch(remote, parentBranch); err != nil {
				if goerrors.Is(err, git.ErrRemoteRefNotFound) {
					// The remote parent branch is gone; prune the stale tracking ref best-effort and
					// skip the parent sync check against it.
					fmt.Printf("Note: Remote branch for base '%s' not found; skipping parent sync check\n", parentBranch)
					parentRefFound = false
					if derr := git.DeleteRemoteTrackingRef(remote, parentBranch); derr != nil {
						fmt.Printf("Note: Could not prune stale tracking ref for '%s': %v\n", parentBranch, derr)
					}
				} else {
					fmt.Printf("Note: Could not fetch base branch '%s': %v\n", parentBranch, err)
				}
			} else if opts.ffParent {
				// #88: fast-forward the local parent to its just-fetched remote so that a branch merged
				// remotely is seen as merged by `git branch -d`. `git merge --ff-only` acts on HEAD, so
				// only fast-forward when the parent is checked out; otherwise leave HEAD untouched.
				if current, cerr := git.GetCurrentBranch(); cerr == nil && current == parentBranch {
					remoteParent := fmt.Sprintf("%s/%s", remote, parentBranch)
					if ferr := git.MergeFFOnly(remoteParent); ferr != nil {
						fmt.Printf("Note: Could not fast-forward '%s': %v\n", parentBranch, ferr)
					}
				}
			}
		}

		// Fetch the topic, distinguishing a benign missing ref from a fatal transport failure.
		if err := git.FetchBranch(remote, topicBranch); err != nil {
			if goerrors.Is(err, git.ErrRemoteRefNotFound) {
				// The remote branch is gone (or was never pushed). Prune the stale tracking ref
				// so later steps do not treat it as an existing remote branch, and skip the
				// sync check for this topic.
				fmt.Printf("Note: Remote branch for '%s' not found; skipping sync check\n", topicBranch)
				topicRefFound = false
				// Best-effort cleanup: the ref may already be absent, so a failure here is a
				// note, not a reason to abort.
				if derr := git.DeleteRemoteTrackingRef(remote, topicBranch); derr != nil {
					fmt.Printf("Note: Could not prune stale tracking ref for '%s': %v\n", topicBranch, derr)
				}
			} else if opts.fetchFailureNonFatal {
				// Opt-in convenience fetch (delete): an unreachable remote must not block the
				// operation. Warn and fall back to whatever tracking data already exists.
				fmt.Printf("Note: Could not fetch topic branch '%s': %v\n", topicBranch, err)
			} else if !force {
				return &errors.FetchFailedError{
					Remote: remote,
					Branch: topicBranch,
					Detail: err.Error(),
				}
			}
		}
	}

	// Sync-check the topic only when it still has a remote ref and a tracking branch.
	if !force && topicRefFound && git.HasTrackingBranch(topicBranch) {
		status, commitCount, err := git.CompareBranchWithRemote(topicBranch)
		if err != nil {
			// HasTrackingBranch already confirmed a tracking branch, and when fetching, the topic
			// fetch succeeded — so a compare failure here is unexpected (e.g. a corrupt/dangling
			// tracking ref or a rev-list failure). Fail closed rather than acting against an
			// undetermined sync status; --force skips this whole block.
			return &errors.GitError{
				Operation: fmt.Sprintf("determine sync status for branch '%s'", topicBranch),
				Err:       err,
			}
		}

		// Decide whether this status aborts. Behind/diverged always abort (remote work would be
		// lost). Ahead aborts only when the caller does not tolerate it; finish and delete both
		// tolerate it and downgrade it to a note.
		abort := false
		switch status {
		case git.SyncStatusBehind, git.SyncStatusDiverged:
			abort = true
		case git.SyncStatusAhead:
			if opts.tolerateAhead {
				fmt.Printf("Note: Local branch '%s' is %d commit(s) ahead of remote\n", topicBranch, commitCount)
			} else {
				abort = true
			}
		}
		if abort {
			trackingBranch, terr := git.GetTrackingBranch(topicBranch)
			if terr != nil {
				trackingBranch = "remote tracking branch"
			}
			return &errors.BranchNotInSyncError{
				BranchName:   topicBranch,
				ShortName:    shortName,
				RemoteBranch: trackingBranch,
				Status:       string(status),
				CommitCount:  commitCount,
				BranchType:   branchType,
				Operation:    opts.operation,
			}
		}
		// SyncStatusEqual: the topic is in sync, proceed normally. (SyncStatusNoTracking always
		// arrives with a non-nil error, handled fatally above.)
	}

	// Sync-check the parent (merge-target) branch when the caller asked for it (#99). Gated the same
	// way as the topic check: only when not forced, the parent still has a remote ref, and it has a
	// tracking branch. The parent invariant is laxer than the topic's — behind/diverged abort
	// (merging onto a stale base), but ahead proceeds (normal right after an unpushed finish).
	if !force && opts.parentSyncCheck && parentBranch != "" && parentRefFound && git.HasTrackingBranch(parentBranch) {
		status, commitCount, err := git.CompareBranchWithRemote(parentBranch)
		if err != nil {
			// As with the topic block, fail closed rather than merge onto an undetermined base.
			return &errors.GitError{
				Operation: fmt.Sprintf("determine sync status for branch '%s'", parentBranch),
				Err:       err,
			}
		}

		if status == git.SyncStatusBehind || status == git.SyncStatusDiverged {
			trackingBranch, terr := git.GetTrackingBranch(parentBranch)
			if terr != nil {
				trackingBranch = "remote tracking branch"
			}
			return &errors.BaseBranchNotInSyncError{
				BranchName:   parentBranch,
				RemoteBranch: trackingBranch,
				Status:       string(status),
				CommitCount:  commitCount,
			}
		}
		// SyncStatusAhead and SyncStatusEqual: the parent is safe to merge onto, proceed. Ahead is
		// left silent (unlike the topic ahead-note) because it is the common, expected state.
	}

	return nil
}
