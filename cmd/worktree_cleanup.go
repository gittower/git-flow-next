// Worktree cleanup for finish and delete (#175): once a topic branch's
// deletion is certain, free the worktree git-flow created for it — or, for one
// the user made by hand, detach it and leave the directory exactly as it was.
// Both commands share the same three operations, defined once here.
package cmd

import (
	"fmt"
	"os"

	"github.com/gittower/git-flow-next/internal/errors"
	"github.com/gittower/git-flow-next/internal/git"
	"github.com/gittower/git-flow-next/internal/navigate"
	"github.com/gittower/git-flow-next/internal/worktree"
	"github.com/spf13/cobra"
)

// WorktreeCleanupOptions carries finish's and delete's two worktree flags.
// Grouped for the same reason CheckoutOptions is: they travel together through
// one call, and individually they are unrelated switches. Neither has a git
// config equivalent — Layer 3 only, like checkout's --worktree/--force.
type WorktreeCleanupOptions struct {
	// Keep detaches a git-flow-created worktree instead of removing it, so the
	// directory survives on a detached HEAD. It has no effect on a worktree
	// git-flow did not create, which is always detached rather than removed.
	Keep bool
	// Force allows removing a git-flow-created worktree that has uncommitted or
	// untracked changes. It never applies to detaching, which changes no files
	// and so never needs it.
	Force bool
}

// addWorktreeCleanupFlags registers --keep-worktree and --force-worktree/-W on
// cmd. Both finish's own flag registration and delete's (registered separately
// in cmd/topicbranch.go and cmd/shorthand.go, which do not share a common
// delete-flags function today) call this, so the two flags are declared in
// exactly one place.
func addWorktreeCleanupFlags(cmd *cobra.Command) {
	cmd.Flags().Bool("keep-worktree", false, "Keep the branch's worktree; detach it from the branch instead of removing it")
	cmd.Flags().BoolP("force-worktree", "W", false, "Remove a git-flow-created worktree even with uncommitted or untracked changes")
}

// redirectPreferringParentWorktree returns the repo handle finish/delete
// should run the rest of their operation against, redirecting away from repo
// when it is bound to the very worktree that holds branch, and reports
// whether a redirect happened.
//
// Both commands eventually free that worktree, but everything before the free
// step — finish's merge and child-branch checkouts, delete's own "switch to
// the parent if currently on the branch" step — checks another branch out
// first. Run from inside the worktree being freed, that checkout would either
// fail outright (the parent is commonly checked out elsewhere already) or
// silently repurpose the worktree onto the parent before the free step ever
// sees it, leaving nothing there to remove or detach. Redirecting once, up
// front, leaves the worktree untouched so the free step can act on it
// correctly — the same "operate from the main worktree once the current one
// may be affected" pattern executeWorktreeRemove already uses for its own
// destructive call.
//
// The destination, when a redirect is needed, is the PARENT branch's own
// worktree if it has one, else the main worktree — the same preference
// decision 7 of the #175 design uses for the navigation destination, and for
// the same reason: the checkout that follows targets the parent, and doing
// that in the main worktree would itself fail if the parent already has a
// dedicated worktree elsewhere (the parent would then be checked out in two
// places at once, which Git refuses). Both callers share this preference —
// delete has no merge target of its own to weigh against it — even though
// delete's NAVIGATION destination (see freeWorktreeForBranch's parentWorktree
// parameter) stays the main worktree regardless; the two are independent.
//
// It is a no-op whenever repo is not bound to that exact worktree: the branch
// has no worktree, its worktree is the main one, or the invocation is already
// running from somewhere else. A failure to look up the parent's own worktree
// is returned rather than silently treated as "no parent worktree" — that
// would risk landing the operation in the main worktree while the parent is
// actually checked out elsewhere, reproducing the exact failure this function
// exists to avoid.
func redirectPreferringParentWorktree(repo *git.Repo, branch string, parentBranch string) (*git.Repo, bool, error) {
	entry, err := repo.WorktreeForBranch(branch)
	if err != nil {
		return nil, false, err
	}
	if entry == nil || entry.Main || !git.SamePath(repo.WorkTree(), entry.Path) {
		return repo, false, nil
	}

	target, err := repo.MainWorkTree()
	if err != nil {
		return nil, false, err
	}
	parentEntry, err := repo.WorktreeForBranch(parentBranch)
	if err != nil {
		return nil, false, err
	}
	if parentEntry != nil && !parentEntry.Main {
		target = parentEntry.Path
	}
	redirected, err := git.Open(target)
	if err != nil {
		return nil, false, err
	}
	return redirected, true, nil
}

// topicWorktreeIfSeparate looks up branch's own worktree and returns a repo
// handle bound to it, ONLY when that worktree exists and differs from the one
// repo is itself bound to.
//
// Once a redirect (redirectPreferringParentWorktree) has moved repo away from
// branch's own worktree, that worktree still has branch checked out
// throughout — the whole point of redirecting was to leave it untouched. Every
// finish step that would otherwise try to check branch out again on repo needs
// this: the rebase step (which genuinely needs to run wherever branch already
// is, not fail trying to check it out a second time), --abort's return-to-
// topic checkout, and the --ff-only failure recovery checkout (both of which
// have nothing to do at all in that case — branch is already exactly where it
// needs to be).
//
// It returns (nil, false, nil) — not an error — whenever branch has no
// worktree of its own, is checked out in the main worktree, or is already the
// one repo is bound to: every case where an ordinary checkout on repo is both
// safe and the right thing to do.
func topicWorktreeIfSeparate(repo *git.Repo, branch string) (*git.Repo, bool, error) {
	entry, err := repo.WorktreeForBranch(branch)
	if err != nil {
		return nil, false, err
	}
	if entry == nil || entry.Main || git.SamePath(repo.WorkTree(), entry.Path) {
		return nil, false, nil
	}
	opened, err := git.Open(entry.Path)
	if err != nil {
		return nil, false, err
	}
	return opened, true, nil
}

// preflightWorktreeCleanup checks, without changing anything, whether branch's
// worktree can be freed once the caller's operation reaches that point. It is
// the "refuse before anything destructive happens" half of the worktree
// lifecycle: finish calls it before the merge starts, and again identically at
// the top of a resumed --continue (which otherwise bypasses the first check
// entirely); delete calls it before deleting the branch.
//
// A branch with no worktree, or one checked out in the main worktree, passes
// trivially — the cleanup flags are no-ops in both cases. Otherwise:
//   - a worktree with a merge, rebase, bisect, cherry-pick, or revert in
//     progress always refuses, regardless of opts: it can never be removed
//     (force overrides dirty content, not an in-progress operation) and never
//     be detached (detaching would abandon the operation with no way back to
//     it).
//   - a git-flow-created worktree that will be REMOVED (managed, and
//     opts.Keep is not set) refuses if it has uncommitted or untracked
//     changes, unless opts.Force is given.
//   - a worktree that will be DETACHED instead (unmanaged, or opts.Keep is
//     set) never needs the dirty check: detaching changes no files.
func preflightWorktreeCleanup(repo *git.Repo, branch string, opts WorktreeCleanupOptions) error {
	entry, err := repo.WorktreeForBranch(branch)
	if err != nil {
		return &errors.GitError{Operation: "look up worktree for branch", Err: err}
	}
	if entry == nil || entry.Main {
		return nil
	}

	op, inProgress, err := repo.WorktreeOperationInProgress(entry.Path)
	if err != nil {
		return &errors.GitError{Operation: "check worktree for an operation in progress", Err: err}
	}
	if inProgress {
		return &errors.WorktreeOperationInProgressError{Branch: branch, Path: entry.Path, Operation: op}
	}

	willRemove := worktree.IsManaged(repo, branch) && !opts.Keep
	if !willRemove || opts.Force {
		return nil
	}
	dirty, err := repo.WorktreeHasChanges(entry.Path)
	if err != nil {
		return &errors.GitError{Operation: "check worktree for changes", Err: err}
	}
	if dirty {
		return &errors.WorktreeDirtyError{Branch: branch, Path: entry.Path, Flag: "--force-worktree"}
	}
	return nil
}

// freeWorktreeForBranch removes or detaches branch's worktree once the branch
// itself is about to be deleted. It re-derives WorktreeForBranch and IsManaged
// itself rather than taking an earlier lookup — cheap, and immune to
// staleness between an earlier preflightWorktreeCleanup call and this one —
// mirroring executeWorktreeRemove's own sequencing.
//
// A git-flow-created worktree (without opts.Keep) is removed and its
// provenance marker cleared. Every other worktree — one git-flow did not
// create, or a managed one kept via opts.Keep — is detached instead: the
// directory and everything in it, including uncommitted work, stay exactly as
// they were, and its marker (if any) is cleared too, since it would otherwise
// dangle on a branch name that no longer exists.
//
// parentWorktree, when non-empty, is offered as the navigation destination
// ahead of the main worktree — finish's caller passes the parent branch's
// worktree path when it has one; delete's caller always passes "", since
// delete has no merge target to prefer. The destination is written only when
// removal actually happens AND the caller is standing inside the worktree
// being removed (decided from the real process cwd, not from repo's own
// binding, which may already be redirected away from that worktree by
// redirectPreferringParentWorktree). Detaching never writes a destination:
// the directory stays exactly where it is, so nobody needs to move.
//
// It returns the repo the caller should keep using afterward: repo itself in
// the ordinary case, or a fresh handle on the main worktree in the one case
// repo's own binding cannot survive the removal — repo bound to the exact
// worktree being removed. Callers are expected to have already redirected
// away from that worktree (see redirectPreferringParentWorktree), so this is
// defensive rather than the common path; it
// must never swap merely because repo is bound to somewhere OTHER than main
// (e.g. finish redirected to the parent branch's own worktree), which would
// hand the caller a repo bound to the wrong place for its next checkout.
func freeWorktreeForBranch(repo *git.Repo, branch string, opts WorktreeCleanupOptions, parentWorktree string) (*git.Repo, error) {
	entry, err := repo.WorktreeForBranch(branch)
	if err != nil {
		return repo, &errors.GitError{Operation: "look up worktree for branch", Err: err}
	}
	if entry == nil || entry.Main {
		return repo, nil
	}

	managed := worktree.IsManaged(repo, branch)
	remove := managed && !opts.Keep

	if !remove {
		if err := repo.DetachWorktree(entry.Path); err != nil {
			return repo, &errors.GitError{Operation: "detach worktree", Err: err}
		}
		if err := worktree.ClearMarker(repo, branch); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
		}
		fmt.Printf("Detached worktree for branch '%s' at %s (kept)\n", branch, entry.Path)
		return repo, nil
	}

	mainWorkTree, err := repo.MainWorkTree()
	if err != nil {
		return repo, &errors.GitError{Operation: "resolve the main worktree", Err: err}
	}

	// Decide stranding and the destination file from the real OS cwd, not from
	// repo's own binding: repo may already be an opRepo redirected away from
	// the very worktree being removed, so repo.WorkTree() would never look
	// "inside" it even when the invoking shell still is.
	strandedUser := false
	if cwd, cwdErr := os.Getwd(); cwdErr == nil {
		strandedUser = git.IsWithin(cwd, entry.Path)
	}
	destination := parentWorktree
	if destination == "" {
		destination = mainWorkTree
	}
	destinationFile := ""
	if strandedUser {
		destinationFile = navigate.DestinationFile()
	}

	opRepo := repo
	if git.SamePath(repo.WorkTree(), entry.Path) {
		opRepo, err = git.Open(mainWorkTree)
		if err != nil {
			return repo, &errors.GitError{Operation: "open the main worktree", Err: err}
		}
	}

	if err := opRepo.RemoveWorktree(entry.Path, opts.Force); err != nil {
		return repo, &errors.GitError{Operation: "remove worktree", Err: err}
	}
	if err := worktree.ClearMarker(opRepo, branch); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
	}

	if destinationFile != "" {
		if err := navigate.WriteDestinationTo(destinationFile, destination); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
		}
	}

	fmt.Printf("Removed worktree for branch '%s' at %s\n", branch, entry.Path)
	return opRepo, nil
}
