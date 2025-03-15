package cmd

import (
	"fmt"
	"strings"

	"github.com/gittower/git-flow-next/config"
	"github.com/gittower/git-flow-next/git"
)

// ListCommand is the implementation of the list command for topic branches
func ListCommand(branchType string) {
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

	// Get the prefix for this branch type
	prefix := branchConfig.Prefix

	// Get all branches
	branches, err := git.ListBranches()
	if err != nil {
		fmt.Printf("Error listing branches: %v\n", err)
		return
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
		return
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
}
