package git

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// IsGitRepo checks if the current directory is a Git repository
func IsGitRepo() bool {
	cmd := exec.Command("git", "rev-parse", "--is-inside-work-tree")
	err := cmd.Run()
	return err == nil
}

// GetCurrentBranch returns the current Git branch
func GetCurrentBranch() (string, error) {
	// Check if we have any commits
	hasCommits, err := HasCommits()
	if err != nil {
		return "", fmt.Errorf("failed to check if repository has commits: %w", err)
	}

	if !hasCommits {
		// If no commits, there's no current branch
		return "", nil
	}

	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get current branch: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

// BranchExists checks if a branch exists
func BranchExists(branch string) error {
	cmd := exec.Command("git", "rev-parse", "--verify", "--quiet", "refs/heads/"+branch)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("branch '%s' does not exist", branch)
	}
	return nil
}

// CreateBranch creates a new branch
func CreateBranch(name string, startPoint string) error {
	// Check if we have any commits
	hasCommits, err := HasCommits()
	if err != nil {
		return fmt.Errorf("failed to check if repository has commits: %w", err)
	}

	if !hasCommits {
		// If no commits, create an initial commit first
		err = CreateInitialCommit(name)
		if err != nil {
			return fmt.Errorf("failed to create initial commit: %w", err)
		}
		return nil
	}

	// If startPoint is empty, use the current branch
	if startPoint == "" {
		currentBranch, err := GetCurrentBranch()
		if err != nil {
			return fmt.Errorf("failed to get current branch: %w", err)
		}
		startPoint = currentBranch
	}

	cmd := exec.Command("git", "checkout", "-b", name, startPoint)
	_, err = cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to create branch: %w", err)
	}
	return nil
}

// Checkout checks out a branch
func Checkout(branch string) error {
	cmd := exec.Command("git", "checkout", branch)
	_, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to checkout branch: %w", err)
	}
	return nil
}

// DeleteBranch deletes a branch
func DeleteBranch(branch string, force bool) error {
	flag := "-d"
	if force {
		flag = "-D"
	}

	cmd := exec.Command("git", "branch", flag, branch)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to delete branch: %s", string(output))
	}
	return nil
}

// HasCommits checks if the repository has any commits
func HasCommits() (bool, error) {
	cmd := exec.Command("git", "rev-parse", "--verify", "HEAD")
	err := cmd.Run()
	if err != nil {
		// If error, there are no commits
		return false, nil
	}
	return true, nil
}

// CreateInitialCommit creates an initial commit and branch
func CreateInitialCommit(branch string) error {
	// Create a README.md file if it doesn't exist
	if _, err := os.Stat("README.md"); os.IsNotExist(err) {
		content := fmt.Sprintf("# Git Flow Repository\n\nThis repository is using git-flow with the following branches:\n- %s: Production releases\n- develop: Development\n", branch)
		err = os.WriteFile("README.md", []byte(content), 0644)
		if err != nil {
			return fmt.Errorf("failed to create README.md: %w", err)
		}
	}

	// Add the file to git
	cmd := exec.Command("git", "add", "README.md")
	_, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to add README.md: %w", err)
	}

	// Create the initial commit
	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	_, err = cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to create initial commit: %w", err)
	}

	// Create the branch (it will be created automatically as the first branch)
	cmd = exec.Command("git", "branch", "-m", branch)
	_, err = cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to rename branch to %s: %w", branch, err)
	}

	return nil
}

// Merge merges a branch into the current branch
func Merge(branch string) error {
	cmd := exec.Command("git", "merge", "--no-ff", branch)
	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	// Check for merge conflicts - Git returns exit code 1 and specific output patterns
	if err != nil {
		// Check if there are unmerged paths (conflicts)
		conflictCmd := exec.Command("git", "ls-files", "--unmerged")
		conflictOutput, _ := conflictCmd.Output()

		if len(conflictOutput) > 0 ||
			strings.Contains(outputStr, "Automatic merge failed") ||
			strings.Contains(outputStr, "CONFLICT") ||
			strings.Contains(outputStr, "merge failed") ||
			strings.Contains(outputStr, "needs merge") {
			return fmt.Errorf("merge conflict: %s", outputStr)
		}
		return fmt.Errorf("failed to merge branch: %s", outputStr)
	}

	return nil
}

// Rebase rebases the current branch onto another branch
func Rebase(branch string) error {
	cmd := exec.Command("git", "rebase", branch)
	output, err := cmd.CombinedOutput()
	if err != nil {
		if strings.Contains(string(output), "conflict") {
			return fmt.Errorf("rebase conflict: %s", string(output))
		}
		return fmt.Errorf("failed to rebase branch: %s", string(output))
	}
	return nil
}

// SquashMerge performs a squash merge of a branch into the current branch
func SquashMerge(branch string) error {
	cmd := exec.Command("git", "merge", "--squash", branch)
	output, err := cmd.CombinedOutput()
	if err != nil {
		if strings.Contains(string(output), "conflict") {
			return fmt.Errorf("squash merge conflict: %s", string(output))
		}
		return fmt.Errorf("failed to squash merge branch: %s", string(output))
	}

	// Commit the squashed changes
	cmd = exec.Command("git", "commit", "-m", fmt.Sprintf("Squashed commit of branch '%s'", branch))
	output, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to commit squashed changes: %s", string(output))
	}

	return nil
}

// ListBranches returns a list of all branches in the repository
func ListBranches() ([]string, error) {
	cmd := exec.Command("git", "branch", "--format=%(refname:short)")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list branches: %w", err)
	}

	// Split the output by newlines and remove empty lines
	branches := []string{}
	for _, branch := range strings.Split(string(output), "\n") {
		if branch != "" {
			branches = append(branches, strings.TrimSpace(branch))
		}
	}

	return branches, nil
}

// HasConflicts checks if there are unresolved conflicts
func HasConflicts() bool {
	// Check for unmerged paths
	cmd := exec.Command("git", "diff", "--name-only", "--diff-filter=U")
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return len(output) > 0
}

// MergeAbort aborts the current merge
func MergeAbort() error {
	cmd := exec.Command("git", "merge", "--abort")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to abort merge: %w", err)
	}
	return nil
}

// RebaseAbort aborts the current rebase
func RebaseAbort() error {
	cmd := exec.Command("git", "rebase", "--abort")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to abort rebase: %s", string(output))
	}
	return nil
}

// RenameBranch renames a branch. If oldBranch is provided, it renames that branch to newBranch.
// If oldBranch is not provided, it renames the current branch to newBranch.
func RenameBranch(newBranch string, oldBranch ...string) error {
	args := []string{"branch", "-m"}
	if len(oldBranch) > 0 {
		args = append(args, oldBranch[0])
	}
	args = append(args, newBranch)

	cmd := exec.Command("git", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to rename branch: %s", strings.TrimSpace(string(output)))
	}
	return nil
}

// Fetch performs a git fetch from the specified remote
func Fetch(remote string) error {
	cmd := exec.Command("git", "fetch", remote)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to fetch from remote '%s': %s", remote, string(output))
	}
	return nil
}

// DeleteRemoteBranch deletes a branch from a remote repository
func DeleteRemoteBranch(remote, branch string) error {
	cmd := exec.Command("git", "push", remote, ":"+branch)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to delete remote branch: %s", string(output))
	}
	return nil
}

// RemoteBranchExists checks if a remote branch exists
func RemoteBranchExists(remote, branch string) bool {
	// Check if the remote tracking branch exists
	ref := fmt.Sprintf("refs/remotes/%s/%s", remote, branch)
	cmd := exec.Command("git", "rev-parse", "--verify", "--quiet", ref)
	return cmd.Run() == nil
}
