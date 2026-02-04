package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/gittower/git-flow-next/internal/config"
	"github.com/gittower/git-flow-next/internal/errors"
	"github.com/gittower/git-flow-next/internal/git"
	"github.com/gittower/git-flow-next/internal/hooks"
)

// PublishCommand is the implementation of the publish command for topic branches.
// If name is empty, the current branch will be published.
// pushOptions are CLI-provided push options to transmit to the server.
// noPushOption suppresses all push options (both CLI and config defaults).
func PublishCommand(branchType string, name string, pushOptions []string, noPushOption bool) {
	if err := publish(branchType, name, pushOptions, noPushOption); err != nil {
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

// publish performs the actual publish logic and returns any errors
func publish(branchType string, name string, cliPushOptions []string, noPushOption bool) error {
	// Validate that git-flow is initialized
	initialized, err := config.IsInitialized()
	if err != nil {
		return &errors.GitError{Operation: "check if git-flow is initialized", Err: err}
	}
	if !initialized {
		return &errors.NotInitializedError{}
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

	// Determine branch name - if empty, use current branch
	var fullBranchName string
	var shortName string
	if name == "" {
		currentBranch, err := git.GetCurrentBranch()
		if err != nil {
			return &errors.GitError{Operation: "get current branch", Err: err}
		}
		fullBranchName = currentBranch

		// Verify current branch is of the correct type
		if branchConfig.Prefix != "" && !strings.HasPrefix(currentBranch, branchConfig.Prefix) {
			return fmt.Errorf("current branch '%s' is not a %s branch", currentBranch, branchType)
		}

		// Extract short name
		if branchConfig.Prefix != "" {
			shortName = strings.TrimPrefix(currentBranch, branchConfig.Prefix)
		} else {
			shortName = currentBranch
		}
	} else {
		// Construct full branch name from provided name
		fullBranchName = name
		shortName = name
		if branchConfig.Prefix != "" && !strings.HasPrefix(name, branchConfig.Prefix) {
			fullBranchName = branchConfig.Prefix + name
		} else if branchConfig.Prefix != "" {
			shortName = strings.TrimPrefix(name, branchConfig.Prefix)
		}
	}

	// Check if branch exists locally
	if err := git.BranchExists(fullBranchName); err != nil {
		return &errors.LocalBranchNotFoundError{BranchName: fullBranchName}
	}

	// Determine remote (from config or default to "origin")
	remote := cfg.Remote
	if remote == "" {
		remote = "origin"
	}

	// Get git directory for hooks
	gitDir, err := git.GetGitDir()
	if err != nil {
		return &errors.GitError{Operation: "get git directory", Err: err}
	}

	// Build hook context
	hookCtx := hooks.HookContext{
		BranchType: branchType,
		BranchName: shortName,
		FullBranch: fullBranchName,
		BaseBranch: branchConfig.Parent,
		Origin:     remote,
	}
	if branchType == "release" || branchType == "hotfix" {
		hookCtx.Version = shortName
	}

	// Resolve push options using three-layer precedence:
	// Layer 1: No default push options (branch config has no push option field)
	// Layer 2: Git config (gitflow.<branchType>.publish.push-option)
	// Layer 3: CLI flags add to config defaults
	// --no-push-option suppresses all options
	pushOptions := resolvePushOptions(cfg, branchType, cliPushOptions, noPushOption)

	// Run publish operation wrapped with hooks
	return hooks.WithHooks(gitDir, branchType, hooks.HookActionPublish, hookCtx, func() error {
		return executePublish(fullBranchName, shortName, branchType, remote, pushOptions)
	})
}

// resolvePushOptions resolves push options using three-layer precedence:
// - Layer 1: No default push options (branch config has no push option field)
// - Layer 2: Git config (gitflow.<branchType>.publish.push-option)
// - Layer 3: CLI flags add to config defaults
// If noPushOption is true, all push options are suppressed.
func resolvePushOptions(cfg *config.Config, branchType string, cliPushOptions []string, noPushOption bool) []string {
	// --no-push-option suppresses all options
	if noPushOption {
		return nil
	}

	var resolvedOptions []string

	// Layer 2: Load from git config (multi-value key)
	configKey := fmt.Sprintf("gitflow.%s.publish.push-option", branchType)
	configOptions, err := git.GetConfigAllValues(configKey)
	if err == nil {
		resolvedOptions = append(resolvedOptions, configOptions...)
	}

	// Layer 3: CLI flags add to config defaults
	resolvedOptions = append(resolvedOptions, cliPushOptions...)

	return resolvedOptions
}

// executePublish performs the actual publish operation (called within hooks wrapper)
func executePublish(fullBranchName, shortName, branchType, remote string, pushOptions []string) error {
	// Fetch to get latest remote refs
	fmt.Printf("Fetching from '%s'...\n", remote)
	if err := git.Fetch(remote); err != nil {
		// Don't fail if fetch fails - remote might not be reachable
		fmt.Fprintf(os.Stderr, "Warning: Could not fetch from '%s': %v\n", remote, err)
	}

	// Check if remote branch already exists
	if git.RemoteBranchExists(remote, fullBranchName) {
		return &errors.RemoteBranchExistsError{
			Remote:     remote,
			BranchName: fullBranchName,
		}
	}

	// Push the branch to remote with tracking
	fmt.Printf("Publishing '%s' to '%s'...\n", fullBranchName, remote)
	if err := git.PushBranch(remote, fullBranchName, pushOptions); err != nil {
		return &errors.GitError{
			Operation: fmt.Sprintf("push branch '%s' to '%s'", fullBranchName, remote),
			Err:       err,
		}
	}

	fmt.Printf("Successfully published '%s' to '%s/%s'\n", fullBranchName, remote, fullBranchName)
	fmt.Printf("Other team members can now track this branch with:\n")
	fmt.Printf("    git flow %s track %s\n", branchType, shortName)
	return nil
}
