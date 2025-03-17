package cmd

import (
	"fmt"

	"github.com/gittower/git-flow-next/config"
	"github.com/gittower/git-flow-next/git"
)

// StartCommand is the implementation of the start command for topic branches
func StartCommand(branchType string, name string) {
	// Validate that git-flow is initialized
	initialized, err := config.IsInitialized()
	if err != nil {
		fmt.Printf("Error checking if git-flow is initialized: %v\n", err)
		return
	}
	if !initialized {
		fmt.Println("Git flow is not initialized. Run 'git flow init' first.")
		return
	}

	// Validate inputs
	if name == "" {
		fmt.Println("Branch name cannot be empty")
		return
	}

	// Get configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Printf("Error loading configuration: %v\n", err)
		return
	}

	// Get branch configuration
	branchConfig, ok := cfg.Branches[branchType]
	if !ok {
		fmt.Printf("Unknown branch type: %s\n", branchType)
		return
	}

	// Get full branch name
	fullBranchName := branchConfig.Prefix + name

	// Check if branch already exists
	if git.BranchExists(fullBranchName) {
		fmt.Printf("Branch '%s' already exists\n", fullBranchName)
		return
	}

	// Get start point
	startPoint := branchConfig.Parent
	if branchConfig.StartPoint != "" {
		// If start point is specified, use it instead of parent
		startPoint = branchConfig.StartPoint
	}

	// Check if start point exists
	if !git.BranchExists(startPoint) {
		fmt.Printf("Start point branch '%s' does not exist\n", startPoint)
		return
	}

	// Create branch
	err = git.CreateBranch(fullBranchName, startPoint)
	if err != nil {
		fmt.Printf("Error creating branch: %v\n", err)
		return
	}

	// Store the start point in Git config
	configKey := fmt.Sprintf("gitflow.branch.%s.base", fullBranchName)
	err = git.SetConfig(configKey, startPoint)
	if err != nil {
		fmt.Printf("Warning: Failed to store start point in config: %v\n", err)
	}

	fmt.Printf("Created branch '%s' from '%s'\n", fullBranchName, startPoint)
}
