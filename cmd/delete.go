package cmd

import (
	"fmt"
	"os"

	"github.com/gittower/git-flow-next/internal/config"
	"github.com/gittower/git-flow-next/internal/errors"
	"github.com/gittower/git-flow-next/internal/git"
	"github.com/gittower/git-flow-next/internal/hooks"
)

// DeleteCommand handles the deletion of a topic branch
func DeleteCommand(branchType string, name string, force *bool, remote *bool, fetch *bool, worktreeOpts WorktreeCleanupOptions) {
	repo := mustOpenRepo()
	if err := executeDelete(repo, branchType, name, force, remote, fetch, worktreeOpts); err != nil {
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

// executeDelete performs the actual branch deletion logic and returns any errors
func executeDelete(repo *git.Repo, branchType string, name string, force *bool, remote *bool, fetch *bool, worktreeOpts WorktreeCleanupOptions) error {
	// Validate that git-flow is initialized before resolving branch types.
	// LoadConfig falls back to DefaultConfig when uninitialized, so this gate
	// must run first or the default branch types mask the uninitialized state.
	initialized, err := config.IsInitialized(repo)
	if err != nil {
		return &errors.GitError{Operation: "check if git-flow is initialized", Err: err}
	}
	if !initialized {
		return &errors.NotInitializedError{}
	}

	// Load configuration
	cfg, err := config.Load(repo)
	if err != nil {
		return &errors.GitError{Operation: "load configuration", Err: err}
	}

	// Get branch configuration
	branchConfig, ok := cfg.Branches[branchType]
	if !ok {
		return &errors.InvalidBranchTypeError{BranchType: branchType}
	}

	// Construct full branch name
	fullBranchName := name
	if branchConfig.Prefix != "" {
		fullBranchName = branchConfig.Prefix + name
	}

	// Check if branch exists
	err = repo.BranchExists(fullBranchName)
	if err != nil {
		return &errors.BranchNotFoundError{BranchName: fullBranchName}
	}

	// Get remote name from config
	remoteName := cfg.Remote

	// Build hook context
	hookCtx := hooks.HookContext{
		BranchType: branchType,
		BranchName: name,
		FullBranch: fullBranchName,
		BaseBranch: branchConfig.Parent,
		Origin:     remoteName,
	}
	if branchType == "release" || branchType == "hotfix" {
		hookCtx.Version = name
	}

	// Redirect away from the branch's own worktree (#175) before anything else
	// below, including the hooks: run from inside that worktree, the "switch
	// to parent if currently on the branch" step further down would either
	// fail outright (the parent is commonly checked out elsewhere) or silently
	// repurpose the worktree onto the parent, leaving nothing there for the
	// free step to act on. Doing the redirect here, rather than inside
	// performDelete as originally written, matters for the hooks: WithHooks
	// captures repo once, up front, and runs the post-delete hook against that
	// SAME handle after the operation completes — an internal redirect inside
	// performDelete would leave WithHooks still holding the pre-redirect repo,
	// so a delete that just removed the worktree the user was standing in
	// would then run its post-hook with a working directory that no longer
	// exists. Redirecting before WithHooks is called means every stage — pre-
	// hook, the delete itself, post-hook — agrees on the same surviving repo.
	redirectedRepo, redirected, err := redirectPreferringParentWorktree(repo, fullBranchName, branchConfig.Parent)
	if err != nil {
		return &errors.GitError{Operation: "resolve worktree for branch", Err: err}
	}
	repo = redirectedRepo

	// Run delete operation wrapped with hooks
	return hooks.WithHooks(repo, branchType, hooks.HookActionDelete, hookCtx, func() error {
		return performDelete(repo, branchType, name, fullBranchName, branchConfig, force, remote, fetch, cfg, worktreeOpts, redirected)
	})
}

// performDelete performs the actual delete operation (called within hooks
// wrapper). redirected reports whether executeDelete already redirected repo
// away from fullBranchName's own worktree (#175) before calling in.
func performDelete(repo *git.Repo, branchType, name, fullBranchName string, branchConfig config.BranchConfig, force *bool, remote *bool, fetch *bool, cfg *config.Config, worktreeOpts WorktreeCleanupOptions, redirected bool) error {
	// Determine if we should fetch before deleting (flag > config, default false).
	shouldFetch := false
	if fetch != nil {
		shouldFetch = *fetch
	} else {
		configKey := fmt.Sprintf("gitflow.%s.delete.fetch", branchType)
		fetchConfig, err := repo.GetConfig(configKey)
		if err == nil && config.ParseBool(fetchConfig) {
			shouldFetch = true
		}
	}

	// Determine if we should force delete the branch (flag > config).
	forceDelete := false
	if force != nil {
		// Command line flag takes precedence
		forceDelete = *force
	} else {
		// Check config if not specified
		configKey := fmt.Sprintf("gitflow.%s.delete.force", branchType)
		forceConfig, err := repo.GetConfig(configKey)
		if err == nil && config.ParseBool(forceConfig) {
			forceDelete = true
		}
	}

	// Determine if we should delete remote branch
	deleteRemote := false
	if remote != nil {
		// Command line flag takes precedence
		deleteRemote = *remote
	} else {
		// Check config if not specified
		configKey := fmt.Sprintf("gitflow.branch.%s.deleteRemote", branchType)
		remoteConfig, err := repo.GetConfig(configKey)
		if err == nil && config.ParseBool(remoteConfig) {
			deleteRemote = true
		}
	}

	// Validate remote exists if remote deletion is requested
	// This must happen before any state-changing operations (checkout, branch deletion)
	if deleteRemote && !repo.RemoteExists(cfg.Remote) {
		return &errors.RemoteNotConfiguredError{Remote: cfg.Remote, Operation: "delete remote branch"}
	}

	// Worktree pre-flight (#175): refuse before ANY destructive step, not just
	// immediately before the free step. Moved here — ahead of the parent
	// checkout and the ffParent fast-forward below, both of which mutate
	// state — so a dirty or mid-operation worktree is caught before anything
	// else happens, matching finish's own ordering and the "pre-flight before
	// any destructive step" promise.
	if err := preflightWorktreeCleanup(repo, fullBranchName, worktreeOpts); err != nil {
		return err
	}

	// If we're on the branch to be deleted, switch to its parent first. This happens before the
	// fetch/sync preflight so that fast-forwarding the parent (see below) operates on HEAD, which
	// is what `git branch -d` checks a topic against when it has no upstream.
	//
	// currentBranch == fullBranchName is the ordinary case: deleting your current branch in a
	// single-worktree repo. It can never be true anymore, though, after a #175 redirect — the
	// redirect exists precisely because we WERE on fullBranchName, in its own linked worktree, and
	// redirecting moved repo somewhere else. That somewhere is either the parent's own worktree
	// (HEAD is already the parent — nothing to do) or the main worktree (HEAD is whatever was last
	// checked out there — not necessarily the parent). The second clause below catches that case:
	// without it, the mergedness check and the ffParent fast-forward further down would silently
	// run against an unrelated branch, and a genuinely merged branch could be refused as unmerged.
	currentBranch, err := repo.GetCurrentBranch()
	if err != nil {
		return &errors.GitError{Operation: "get current branch", Err: err}
	}
	if currentBranch == fullBranchName || (redirected && currentBranch != branchConfig.Parent) {
		parentBranch := branchConfig.Parent
		if parentBranch != "" {
			if err := repo.Checkout(parentBranch); err != nil {
				return &errors.GitError{Operation: fmt.Sprintf("checkout parent branch '%s'", parentBranch), Err: err}
			}
		} else {
			return &errors.GitError{Operation: "delete branch", Err: fmt.Errorf("cannot delete the current branch without a parent branch configured")}
		}
	}

	// Fetch the parent and topic, fast-forward the parent from its remote (so a branch that was
	// merged remotely, e.g. via a PR, is recognized as merged by `git branch -d`), and sync-check
	// the topic. Delete uses the relaxed preflight variant: a fetch failure is a non-fatal note and
	// a topic that is merely ahead of its remote is tolerated (a non-force `git branch -d` still
	// guards genuinely unmerged work). A topic that is behind/diverged aborts unless forced.
	if err := runFetchSyncPreflight(repo, cfg, branchType, cfg.Remote, fullBranchName, name, branchConfig.Parent, shouldFetch, forceDelete, preflightOptions{
		ffParent:             true,
		tolerateAhead:        true,
		fetchFailureNonFatal: true,
		operation:            "delete",
	}); err != nil {
		return err
	}

	// Confirm the branch can actually be deleted BEFORE freeing its worktree
	// (#175): freeing is a one-way trip (a git-flow-created worktree is
	// removed outright; even a hand-made one, detached, does not un-detach
	// itself), and 'git branch -d' below would otherwise be the first thing
	// to notice a clean-but-unmerged branch — by which point the worktree is
	// already gone. This mirrors 'git branch -d's own no-upstream mergedness
	// check (branch must be an ancestor of the branch now checked out, which
	// the steps above already arranged to be the parent whenever that
	// mattered) without trying to reproduce every rule 'git branch -d' itself
	// applies (a configured upstream, for instance): a false negative here
	// just means that real call further down — unreached in the cases that
	// matter — makes the final call.
	if !forceDelete {
		headBranch, err := repo.GetCurrentBranch()
		if err != nil {
			return &errors.GitError{Operation: "get current branch", Err: err}
		}
		merged, err := repo.IsAncestor(fullBranchName, headBranch)
		if err != nil {
			return &errors.GitError{Operation: "check whether branch is merged", Err: err}
		}
		if !merged {
			return &errors.GitError{Operation: fmt.Sprintf("delete branch '%s'", fullBranchName), Err: fmt.Errorf("the branch is not fully merged into '%s'; use --force to delete it anyway", headBranch)}
		}
	}

	// Free the branch's worktree (#175), right before the branch itself goes
	// away: a branch checked out in a linked worktree cannot be deleted while
	// checked out there. Delete always prefers the main worktree as the
	// navigation destination — unlike finish, it has no merge target to
	// prefer instead.
	freedRepo, err := freeWorktreeForBranch(repo, fullBranchName, worktreeOpts, "")
	if err != nil {
		return err
	}
	repo = freedRepo

	// Delete the branch with appropriate flag
	deleteErr := repo.DeleteBranch(fullBranchName, forceDelete)
	if deleteErr != nil {
		return &errors.GitError{Operation: fmt.Sprintf("delete branch '%s'", fullBranchName), Err: deleteErr}
	}

	// Delete remote branch if requested
	if deleteRemote {
		remoteName := cfg.Remote

		deletedRemote := false
		if repo.RemoteBranchExists(remoteName, fullBranchName) {
			// Delete remote branch
			if err := repo.DeleteRemoteBranch(remoteName, fullBranchName); err != nil {
				return &errors.GitError{Operation: fmt.Sprintf("delete remote branch '%s'", fullBranchName), Err: err}
			} else {
				deletedRemote = true
			}
		}
		if deletedRemote {
			fmt.Printf("Deleted branch %s and its remote tracking branch\n", fullBranchName)
		} else {
			fmt.Printf("Deleted branch %s (no remote tracking branch found)\n", fullBranchName)
		}
	} else {
		fmt.Printf("Deleted branch %s\n", fullBranchName)
	}

	// Clean up base branch configuration
	configKey := git.BaseBranchKey(fullBranchName)
	if err := repo.UnsetConfigIfPresent(configKey); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to clean up base config: %v\n", err)
	}

	return nil
}
