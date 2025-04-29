package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/gittower/git-flow-next/config"
	"github.com/gittower/git-flow-next/internal/errors"
	"github.com/gittower/git-flow-next/internal/git"
)

// ListCommand is the implementation of the list command for topic branches
func ListCommand(branchType string) {
	if err := list(branchType); err != nil {
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

// list performs the actual branch listing logic and returns any errors
func list(branchType string) error {
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

	// Get the prefix for this branch type
	prefix := branchConfig.Prefix

	// Get all branches
	branches, err := git.ListBranches()
	if err != nil {
		return &errors.GitError{Operation: "list branches", Err: err}
	}

	// Filter branches by prefix
	var topicBranches []string
	for _, branch := range branches {
		if strings.HasPrefix(branch, prefix) {
			// Remove the prefix to get the branch name
			name := strings.TrimPrefix(branch, prefix)
			topicBranches = append(topicBranches, name)
		}
	}

	// Print the branches
	if len(topicBranches) == 0 {
		fmt.Printf("No %s branches found\n", branchType)
		return nil
	}

	// Capitalize the first letter of the branch type
	branchTypeCapitalized := branchType
	if len(branchType) > 0 {
		branchTypeCapitalized = strings.ToUpper(branchType[:1]) + branchType[1:]
	}

	fmt.Printf("%s branches:\n", branchTypeCapitalized)
	for _, branch := range topicBranches {
		fmt.Printf("  %s\n", branch)
	}

	return nil
}
