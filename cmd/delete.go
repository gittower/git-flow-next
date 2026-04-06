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
func DeleteCommand(branchType string, name string, force *bool, remote *bool) {
	if err := executeDelete(branchType, name, force, remote); err != nil {
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
func executeDelete(branchType string, name string, force *bool, remote *bool) error {
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

	// Get git directory for hooks
	gitDir, err := git.GetGitDir()
	if err != nil {
		return &errors.GitError{Operation: "get git directory", Err: err}
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

	// Run delete operation wrapped with hooks
	return hooks.WithHooks(gitDir, branchType, hooks.HookActionDelete, hookCtx, func() error {
		return performDelete(branchType, name, fullBranchName, branchConfig, force, remote, cfg)
	})
}

// performDelete performs the actual delete operation (called within hooks wrapper)
func performDelete(branchType, name, fullBranchName string, branchConfig config.BranchConfig, force *bool, remote *bool, cfg *config.Config) error {
	// Determine if we should force delete the branch
	forceDelete := false
	if force != nil {
		// Command line flag takes precedence
		forceDelete = *force
	} else {
		// Check config if not specified
		configKey := fmt.Sprintf("gitflow.%s.delete.force", branchType)
		forceConfig, err := git.GetConfig(configKey)
		if err == nil && forceConfig == "true" {
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
		remoteConfig, err := git.GetConfig(configKey)
		if err == nil && remoteConfig == "true" {
			deleteRemote = true
		}
	}

	// Validate remote exists if remote deletion is requested
	// This must happen before any state-changing operations (checkout, branch deletion)
	if deleteRemote && !git.RemoteExists(cfg.Remote) {
		return &errors.RemoteNotConfiguredError{Remote: cfg.Remote, Operation: "delete remote branch"}
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
	deleteErr := git.DeleteBranch(fullBranchName, forceDelete)
	if deleteErr != nil {
		return &errors.GitError{Operation: fmt.Sprintf("delete branch '%s'", fullBranchName), Err: deleteErr}
	}

	// Delete remote branch if requested
	if deleteRemote {
		remoteName := cfg.Remote

		// Delete remote branch
		if err := git.DeleteRemoteBranch(remoteName, fullBranchName); err != nil {
			return &errors.GitError{Operation: fmt.Sprintf("delete remote branch '%s'", fullBranchName), Err: err}
		}
		fmt.Printf("Deleted branch %s and its remote tracking branch\n", fullBranchName)
	} else {
		fmt.Printf("Deleted branch %s\n", fullBranchName)
	}

	// Clean up base branch configuration
	configKey := fmt.Sprintf("gitflow.branch.%s.base", fullBranchName)
	if err := git.UnsetConfig(configKey); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to clean up base config: %v\n", err)
	}

	return nil
}
