package cmd

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/gittower/git-flow-next/config"
	"github.com/gittower/git-flow-next/errors"
	"github.com/gittower/git-flow-next/git"
)

// DeleteCommand handles the deletion of a topic branch
func DeleteCommand(branchType string, name string, force bool) error {
	// Load configuration
	cfg, err := config.LoadConfig()
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
	err = git.BranchExists(fullBranchName)
	if err != nil {
		return &errors.BranchNotFoundError{BranchName: fullBranchName}
	}

	// Check if we're currently on the branch to be deleted
	currentBranch, err := git.GetCurrentBranch()
	if err != nil {
		return &errors.GitError{Operation: "get current branch", Err: err}
	}
	if currentBranch == fullBranchName {
		// If we're on the branch to be deleted, try to switch to its parent
		parentBranch := branchConfig.Parent
		if parentBranch != "" {
			if err := git.Checkout(parentBranch); err != nil {
				return &errors.GitError{Operation: fmt.Sprintf("checkout parent branch '%s'", parentBranch), Err: err}
			}
		} else {
			return &errors.GitError{Operation: "delete branch", Err: fmt.Errorf("cannot delete the current branch without a parent branch configured")}
		}
	}

	// Delete the branch with appropriate flag
	var deleteErr error
	if force {
		// Use -D for force delete
		cmd := exec.Command("git", "branch", "-D", fullBranchName)
		output, err := cmd.CombinedOutput()
		if err != nil {
			deleteErr = fmt.Errorf("%s", strings.TrimSpace(string(output)))
		}
	} else {
		// Use -d for safe delete
		deleteErr = git.DeleteBranch(fullBranchName)
	}

	if deleteErr != nil {
		return &errors.GitError{Operation: fmt.Sprintf("delete branch '%s'", fullBranchName), Err: deleteErr}
	}

	fmt.Printf("Deleted branch %s\n", fullBranchName)
	return nil
}
