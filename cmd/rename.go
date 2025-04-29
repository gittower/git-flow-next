package cmd

import (
	"fmt"

	"github.com/gittower/git-flow-next/config"
	"github.com/gittower/git-flow-next/internal/errors"
	"github.com/gittower/git-flow-next/internal/git"
)

// RenameCommand handles renaming a topic branch
func RenameCommand(branchType string, oldName string, newName string) error {
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

	// Construct full branch names
	oldFullBranchName := oldName
	newFullBranchName := newName
	if branchConfig.Prefix != "" {
		oldFullBranchName = branchConfig.Prefix + oldName
		newFullBranchName = branchConfig.Prefix + newName
	}

	// Check if old branch exists
	err = git.BranchExists(oldFullBranchName)
	if err != nil {
		return &errors.BranchNotFoundError{BranchName: oldFullBranchName}
	}

	// Check if new branch name already exists
	err = git.BranchExists(newFullBranchName)
	if err == nil {
		return &errors.GitError{Operation: "rename branch", Err: fmt.Errorf("branch '%s' already exists", newFullBranchName)}
	}

	// Check if we're currently on the branch to be renamed
	currentBranch, err := git.GetCurrentBranch()
	if err != nil {
		return &errors.GitError{Operation: "get current branch", Err: err}
	}

	// If we're on the branch to be renamed, we need to rename it while on it
	if currentBranch == oldFullBranchName {
		err = git.RenameBranch(newFullBranchName)
	} else {
		// Otherwise, rename it while staying on the current branch
		err = git.RenameBranch(newFullBranchName, oldFullBranchName)
	}

	if err != nil {
		return &errors.GitError{Operation: fmt.Sprintf("rename branch '%s' to '%s'", oldFullBranchName, newFullBranchName), Err: err}
	}

	fmt.Printf("Renamed branch '%s' to '%s'\n", oldFullBranchName, newFullBranchName)
	return nil
}
