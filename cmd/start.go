package cmd

import (
	"fmt"
	"os"

	"github.com/gittower/git-flow-next/internal/config"
	"github.com/gittower/git-flow-next/internal/errors"
	"github.com/gittower/git-flow-next/internal/git"
	"github.com/gittower/git-flow-next/internal/hooks"
)

// StartCommand is the implementation of the start command for topic branches
// If shouldFetch is nil, the function will check config for fetch preference
// If base is empty, the function will use the configured starting point
func StartCommand(branchType string, name string, base string, shouldFetch *bool) {
	if err := start(branchType, name, base, shouldFetch); err != nil {
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
func start(branchType string, name string, base string, shouldFetch *bool) error {
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

	// Get git directory for hooks and filters
	gitDir, err := git.GetGitDir()
	if err != nil {
		return &errors.GitError{Operation: "get git directory", Err: err}
	}

	// Apply version filter for any branch type
	// The filter script (filter-flow-{branchType}-start-version) decides what to do
	filteredName, err := hooks.RunVersionFilter(gitDir, branchType, name)
	if err != nil {
		return &errors.GitError{Operation: "run version filter", Err: err}
	}
	if filteredName != name {
		fmt.Printf("Version filter changed '%s' to '%s'\n", name, filteredName)
		name = filteredName
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

	// Get full branch name
	fullBranchName := branchConfig.Prefix + name

	// Get start point
	startPoint := branchConfig.Parent
	if branchConfig.StartPoint != "" {
		// If start point is specified in config, use it instead of parent
		startPoint = branchConfig.StartPoint
	}
	if base != "" {
		// If base argument is provided, it overrides the configured starting point
		startPoint = base
	}

	// Build hook context
	hookCtx := hooks.HookContext{
		BranchType: branchType,
		BranchName: name,
		FullBranch: fullBranchName,
		BaseBranch: startPoint,
		Origin:     cfg.Remote,
	}
	// Set version for branches configured with tagging
	if branchConfig.Tag {
		hookCtx.Version = name
	}

	// Run start operation wrapped with hooks
	return hooks.WithHooks(gitDir, branchType, hooks.HookActionStart, hookCtx, func() error {
		return executeStart(branchType, name, base, shouldFetch, cfg, branchConfig, fullBranchName, startPoint)
	})
}

// executeStart performs the actual start operation (called within hooks wrapper)
func executeStart(branchType string, name string, base string, shouldFetch *bool, cfg *config.Config, branchConfig config.BranchConfig, fullBranchName string, startPoint string) error {
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
		if git.RemoteExists(remoteName) {
			fmt.Printf("Fetching from %s...\n", remoteName)
			if err := git.Fetch(remoteName); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
			}
		}
	}

	// Check if branch already exists
	if err := git.BranchExists(fullBranchName); err == nil {
		return &errors.BranchExistsError{BranchName: fullBranchName}
	}

	// Check if start point exists (can be branch, tag, or commit)
	if err := git.BranchOrCommitExists(startPoint); err != nil {
		return &errors.BranchNotFoundError{BranchName: startPoint}
	}

	// Create branch
	err := git.CreateBranch(fullBranchName, startPoint)
	if err != nil {
		return &errors.GitError{Operation: "create branch", Err: err}
	}

	// Store the start point in Git config
	if err := git.SetBaseBranch(fullBranchName, startPoint); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to store base branch: %v\n", err)
	}

	fmt.Printf("Created branch '%s' from '%s'\n", fullBranchName, startPoint)
	return nil
}
