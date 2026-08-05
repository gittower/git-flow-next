package cmd

import (
	"fmt"
	"strings"

	"github.com/gittower/git-flow-next/internal/errors"
	"github.com/gittower/git-flow-next/internal/git"
)

// removeWorktreeForBranch removes the linked worktree that has branch checked
// out, if any, so the branch can be deleted. It is a no-op when no worktree
// holds the branch. A dirty worktree (uncommitted or untracked changes) is only
// removed when force is set; otherwise an actionable error is returned so the
// user can clean up or opt into force removal. Shared by the finish and delete
// commands.
func removeWorktreeForBranch(repo *git.Repo, branch string, force bool) error {
	path, err := repo.WorktreeForBranch(branch)
	if err != nil {
		return &errors.GitError{Operation: fmt.Sprintf("find worktree for branch '%s'", branch), Err: err}
	}
	if path == "" {
		return nil
	}

	fmt.Printf("Removing worktree at '%s' for branch '%s'\n", path, branch)
	if err := repo.RemoveWorktree(path, force); err != nil {
		if !force && strings.Contains(err.Error(), "modified or untracked files") {
			return &errors.GitError{
				Operation: fmt.Sprintf("remove worktree at '%s'", path),
				Err:       fmt.Errorf("worktree has uncommitted or untracked changes; clean or commit them, or re-run with --force-remove-worktree"),
			}
		}
		return &errors.GitError{Operation: fmt.Sprintf("remove worktree at '%s'", path), Err: err}
	}

	fmt.Printf("Removed worktree at '%s'\n", path)
	return nil
}

// preflightWorktreeRemoval refuses an enabled, non-forced removal when the
// linked worktree is dirty. Finish calls this before beginning its merge flow
// so it does not complete a merge only to discover that cleanup is unsafe.
func preflightWorktreeRemoval(repo *git.Repo, branch string, force bool) error {
	if force {
		return nil
	}

	path, err := repo.WorktreeForBranch(branch)
	if err != nil {
		return &errors.GitError{Operation: fmt.Sprintf("find worktree for branch '%s'", branch), Err: err}
	}
	if path == "" {
		return nil
	}

	dirty, err := repo.WorktreeHasChanges(path)
	if err != nil {
		return &errors.GitError{Operation: fmt.Sprintf("inspect worktree at '%s'", path), Err: err}
	}
	if dirty {
		return &errors.GitError{
			Operation: fmt.Sprintf("remove worktree at '%s'", path),
			Err:       fmt.Errorf("worktree has uncommitted or untracked changes; clean or commit them, or re-run with --force-remove-worktree"),
		}
	}

	return nil
}
