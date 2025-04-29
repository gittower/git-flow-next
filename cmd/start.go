package cmd

import (
	"fmt"
	"os"

	"github.com/gittower/git-flow-next/config"
	"github.com/gittower/git-flow-next/errors"
	"github.com/gittower/git-flow-next/internal/git"
)

// StartCommand is the implementation of the start command for topic branches
// If shouldFetch is nil, the function will check config for fetch preference
func StartCommand(branchType string, name string, shouldFetch *bool) {
	if err := start(branchType, name, shouldFetch); err != nil {
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

// start performs the actual branch creation logic with optional fetch and returns any errors
func start(branchType string, name string, shouldFetch *bool) error {
	// Validate that git-flow is initialized
	initialized, err := config.IsInitialized()
	if err != nil {
		return &errors.GitError{Operation: "check if git-flow is initialized", Err: err}
	}
	if !initialized {
		return &errors.NotInitializedError{}
	}

	// Validate inputs
	if name == "" {
		return &errors.EmptyBranchNameError{}
	}

	// Get configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		return &errors.GitError{Operation: "load configuration", Err: err}
	}

	// Get branch configuration
	branchConfig, ok := cfg.Branches[branchType]
	if !ok {
		return &errors.InvalidBranchTypeError{BranchType: branchType}
	}

	// Determine if we should fetch
	fetchFromConfig := false
	if shouldFetch == nil {
		// If not explicitly specified, check config
		configKey := fmt.Sprintf("gitflow.%s.start.fetch", branchType)
		fetchConfig, err := git.GetConfig(configKey)
		if err == nil && fetchConfig == "true" {
			fetchFromConfig = true
		}
	}

	// Perform fetch if requested
	remoteName := cfg.Remote
	if shouldFetch != nil && *shouldFetch || shouldFetch == nil && fetchFromConfig {
		// Fetch from remote
		fmt.Printf("Fetching from %s...\n", remoteName)
		if err := git.Fetch(remoteName); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
		}
	}

	// Get full branch name
	fullBranchName := branchConfig.Prefix + name

	// Check if branch already exists
	if err := git.BranchExists(fullBranchName); err == nil {
		return &errors.BranchExistsError{BranchName: fullBranchName}
	}

	// Get start point
	startPoint := branchConfig.Parent
	if branchConfig.StartPoint != "" {
		// If start point is specified, use it instead of parent
		startPoint = branchConfig.StartPoint
	}

	// Check if start point exists
	if err := git.BranchExists(startPoint); err != nil {
		return &errors.BranchNotFoundError{BranchName: startPoint}
	}

	// Create branch
	err = git.CreateBranch(fullBranchName, startPoint)
	if err != nil {
		return &errors.GitError{Operation: "create branch", Err: err}
	}

	// Store the start point in Git config
	configKey := fmt.Sprintf("gitflow.branch.%s.base", fullBranchName)
	err = git.SetConfig(configKey, startPoint)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to store start point in config: %v\n", err)
	}

	fmt.Printf("Created branch '%s' from '%s'\n", fullBranchName, startPoint)
	return nil
}
