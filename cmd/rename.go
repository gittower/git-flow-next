package cmd

import (
	"fmt"
	"os"

	"github.com/gittower/git-flow-next/internal/config"
	"github.com/gittower/git-flow-next/internal/errors"
	"github.com/gittower/git-flow-next/internal/git"
)

// RenameCommand handles renaming a topic branch
func RenameCommand(branchType string, oldName string, newName string) {
	repo := mustOpenRepo()
	if err := executeRename(repo, branchType, oldName, newName); err != nil {
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

// executeRename performs the actual branch renaming logic and returns any errors
func executeRename(repo *git.Repo, branchType string, oldName string, newName string) error {
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

	// Construct full branch names
	oldFullBranchName := oldName
	newFullBranchName := newName
	if branchConfig.Prefix != "" {
		oldFullBranchName = branchConfig.Prefix + oldName
		newFullBranchName = branchConfig.Prefix + newName
	}

	// Check if old branch exists
	err = repo.BranchExists(oldFullBranchName)
	if err != nil {
		return &errors.BranchNotFoundError{BranchName: oldFullBranchName}
	}

	// Check if new branch name already exists
	err = repo.BranchExists(newFullBranchName)
	if err == nil {
		return &errors.GitError{Operation: "rename branch", Err: fmt.Errorf("branch '%s' already exists", newFullBranchName)}
	}

	// Check if we're currently on the branch to be renamed
	currentBranch, err := repo.GetCurrentBranch()
	if err != nil {
		return &errors.GitError{Operation: "get current branch", Err: err}
	}

	// If we're on the branch to be renamed, we need to rename it while on it
	if currentBranch == oldFullBranchName {
		err = repo.RenameBranch(oldFullBranchName, newFullBranchName)
	} else {
		// Otherwise, rename it while staying on the current branch
		err = repo.RenameBranch(oldFullBranchName, newFullBranchName)
	}

	if err != nil {
		return &errors.GitError{Operation: fmt.Sprintf("rename branch '%s' to '%s'", oldFullBranchName, newFullBranchName), Err: err}
	}

	fmt.Printf("Renamed branch '%s' to '%s'\n", oldFullBranchName, newFullBranchName)
	return nil
}
