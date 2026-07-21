package cmd

import (
	goerrors "errors"
	"fmt"

	"github.com/gittower/git-flow-next/internal/config"
	"github.com/gittower/git-flow-next/internal/errors"
	"github.com/gittower/git-flow-next/internal/git"
)

// runFetchSyncPreflight guards the initial finish against an out-of-date topic branch. It fetches
// the topic (and, best-effort, the parent) and then verifies the topic is in sync with its remote.
//
// Behavior:
//   - shouldFetch=false or no remote configured: fetch is skipped (no "Fetching" line). The topic
//     sync check still runs against existing tracking data (so --no-fetch does not skip it).
//   - Topic fetch reports a missing remote ref (never pushed / deleted after a remote merge): the
//     missing ref is benign — the stale tracking ref is pruned and the sync check is skipped.
//   - Topic fetch fails with a transport/auth error: fatal (FetchFailedError) unless force.
//   - Topic ahead/behind/diverged from its remote: fatal (BranchNotInSyncError) unless force.
//
// The parent is fetched only best-effort here; it is intentionally NOT sync-checked. Future
// extension points (kept out of this signature to avoid unused hooks):
//   - #99: parent sync-check before merge.
//   - #88: fast-forward the parent from its remote before merge.
func runFetchSyncPreflight(cfg *config.Config, branchType, remote, topicBranch, shortName, parentBranch string, shouldFetch, force bool) error {
	// topicRefFound tracks whether the topic still has a remote ref to compare against. It stays
	// true when the fetch is skipped (we then rely on existing tracking data).
	topicRefFound := true

	if shouldFetch && git.RemoteExists(remote) {
		fmt.Printf("Fetching from remote '%s'...\n", remote)

		// Fetch the parent best-effort. Any failure (including a missing ref) is a non-fatal
		// note; the parent is not sync-checked here (see #99).
		if err := git.FetchBranch(remote, parentBranch); err != nil {
			fmt.Printf("Note: Could not fetch base branch '%s': %v\n", parentBranch, err)
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
				// note, not a reason to abort the finish.
				if derr := git.DeleteRemoteTrackingRef(remote, topicBranch); derr != nil {
					fmt.Printf("Note: Could not prune stale tracking ref for '%s': %v\n", topicBranch, derr)
				}
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
		if err == nil {
			switch status {
			case git.SyncStatusAhead, git.SyncStatusBehind, git.SyncStatusDiverged:
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
				}
			}
			// SyncStatusEqual or SyncStatusNoTracking: proceed normally.
		}
		// err != nil: unable to compare (no tracking data) — proceed normally.
	}

	return nil
}
