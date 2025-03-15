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
func BranchExists(branch string) bool {
	cmd := exec.Command("git", "show-ref", "--verify", "--quiet", "refs/heads/"+branch)
	err := cmd.Run()
	return err == nil
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
func DeleteBranch(branch string) error {
	cmd := exec.Command("git", "branch", "-d", branch)
	_, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to delete branch: %w", err)
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
	_, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to merge branch: %w", err)
	}
	return nil
}

// Rebase rebases the current branch onto another branch
func Rebase(branch string) error {
	cmd := exec.Command("git", "rebase", branch)
	_, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to rebase branch: %w", err)
	}
	return nil
}

// SquashMerge performs a squash merge of a branch into the current branch
func SquashMerge(branch string) error {
	cmd := exec.Command("git", "merge", "--squash", branch)
	_, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to squash merge branch: %w", err)
	}

	// Commit the squashed changes
	cmd = exec.Command("git", "commit", "-m", fmt.Sprintf("Squashed commit of branch '%s'", branch))
	_, err = cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to commit squashed changes: %w", err)
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
