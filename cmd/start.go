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
	startPoint := branchConfig.StartPoint
	if startPoint == "" {
		// If no start point is specified, use the parent branch
		startPoint = branchConfig.Parent
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

	fmt.Printf("Created branch '%s' from '%s'\n", fullBranchName, startPoint)
}
