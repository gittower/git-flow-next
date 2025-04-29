package cmd

import (
	"fmt"

	"github.com/gittower/git-flow-next/internal/config"
	"github.com/gittower/git-flow-next/internal/errors"
	"github.com/gittower/git-flow-next/internal/git"
)

// DeleteCommand handles the deletion of a topic branch
func DeleteCommand(branchType string, name string, force bool, remote *bool) error {
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

	// Determine if we should delete remote branch
	deleteRemote := false
	if remote != nil {
		// Command line flag takes precedence
		deleteRemote = *remote
	} else {
		// Check config if not specified
		configKey := fmt.Sprintf("gitflow.branch.%s.deleteRemote", branchType)
		remoteConfig, err := git.GetConfig(configKey)
		if err == nil && remoteConfig == "true" {
			deleteRemote = true
		}
	}

	// Delete the branch with appropriate flag
	deleteErr := git.DeleteBranch(fullBranchName, force)
	if deleteErr != nil {
		return &errors.GitError{Operation: fmt.Sprintf("delete branch '%s'", fullBranchName), Err: deleteErr}
	}

	// Delete remote branch if requested
	if deleteRemote {
		// Get remote name from config
		remoteName, err := git.GetConfig("gitflow.remote")
		if err != nil {
			remoteName = "origin" // Default to origin if not configured
		}

		// Delete remote branch
		if err := git.DeleteRemoteBranch(remoteName, fullBranchName); err != nil {
			return &errors.GitError{Operation: fmt.Sprintf("delete remote branch '%s'", fullBranchName), Err: err}
		}
		fmt.Printf("Deleted branch %s and its remote tracking branch\n", fullBranchName)
	} else {
		fmt.Printf("Deleted branch %s\n", fullBranchName)
	}

	return nil
}
