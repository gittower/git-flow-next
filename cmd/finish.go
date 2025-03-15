package cmd

import (
	"fmt"

	"github.com/gittower/git-flow-next/config"
	"github.com/gittower/git-flow-next/git"
)

// FinishCommand is the implementation of the finish command for topic branches
func FinishCommand(branchType string, name string) {
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

	// Check if branch exists
	if !git.BranchExists(fullBranchName) {
		fmt.Printf("Branch '%s' does not exist\n", fullBranchName)
		return
	}

	// Get target branch (parent or start point)
	targetBranch := branchConfig.Parent
	if branchConfig.UpstreamStrategy != "" {
		// If upstream strategy is specified, use it to determine the target branch
		switch branchConfig.UpstreamStrategy {
		case "parent":
			targetBranch = branchConfig.Parent
		case "start-point":
			targetBranch = branchConfig.StartPoint
		default:
			fmt.Printf("Unknown upstream strategy: %s\n", branchConfig.UpstreamStrategy)
			return
		}
	}

	// Check if target branch exists
	if !git.BranchExists(targetBranch) {
		fmt.Printf("Target branch '%s' does not exist\n", targetBranch)
		return
	}

	// Get current branch
	currentBranch, err := git.GetCurrentBranch()
	if err != nil {
		fmt.Printf("Error getting current branch: %v\n", err)
		return
	}

	// Check if we're on the branch to finish
	if currentBranch != fullBranchName {
		// Checkout the branch to finish
		err = git.Checkout(fullBranchName)
		if err != nil {
			fmt.Printf("Error checking out branch '%s': %v\n", fullBranchName, err)
			return
		}
		fmt.Printf("Switched to branch '%s'\n", fullBranchName)
	}

	// Checkout target branch
	err = git.Checkout(targetBranch)
	if err != nil {
		fmt.Printf("Error checking out target branch '%s': %v\n", targetBranch, err)
		return
	}
	fmt.Printf("Switched to branch '%s'\n", targetBranch)

	// Merge the branch
	mergeStrategy := "merge" // Default strategy
	if branchConfig.DownstreamStrategy != "" {
		mergeStrategy = branchConfig.DownstreamStrategy
	}

	switch mergeStrategy {
	case "merge":
		err = git.Merge(fullBranchName)
	case "rebase":
		err = git.Rebase(fullBranchName)
	case "squash":
		err = git.SquashMerge(fullBranchName)
	default:
		fmt.Printf("Unknown merge strategy: %s\n", mergeStrategy)
		return
	}

	if err != nil {
		fmt.Printf("Error merging branch '%s': %v\n", fullBranchName, err)
		return
	}
	fmt.Printf("Merged branch '%s' into '%s'\n", fullBranchName, targetBranch)

	// Delete branch
	err = git.DeleteBranch(fullBranchName)
	if err != nil {
		fmt.Printf("Error deleting branch '%s': %v\n", fullBranchName, err)
		return
	}
	fmt.Printf("Deleted branch '%s'\n", fullBranchName)

	fmt.Printf("Successfully finished branch '%s'\n", fullBranchName)
}
