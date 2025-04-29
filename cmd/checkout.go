package cmd

import (
	"fmt"
	"strings"

	"github.com/gittower/git-flow-next/internal/config"
	"github.com/gittower/git-flow-next/internal/errors"
	"github.com/gittower/git-flow-next/internal/git"
)

// CheckoutCommand handles checking out a topic branch
func CheckoutCommand(branchType string, nameOrPrefix string, showCommands bool) error {
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

	// If no name/prefix provided, list available branches and return
	if nameOrPrefix == "" {
		branches, err := git.ListBranches()
		if err != nil {
			return &errors.GitError{Operation: "list branches", Err: err}
		}

		prefix := branchConfig.Prefix
		found := false
		fmt.Printf("Available %s branches:\n", branchType)
		for _, branch := range branches {
			if strings.HasPrefix(branch, prefix) {
				found = true
				fmt.Printf("  %s\n", strings.TrimPrefix(branch, prefix))
			}
		}
		if !found {
			fmt.Printf("No %s branches exist.\n", branchType)
		}
		return nil
	}

	// Construct full branch name
	fullBranchName := nameOrPrefix
	if branchConfig.Prefix != "" {
		fullBranchName = branchConfig.Prefix + nameOrPrefix
	}

	// Check if branch exists
	err = git.BranchExists(fullBranchName)
	if err != nil {
		// If exact match not found, try prefix match
		branches, err := git.ListBranches()
		if err != nil {
			return &errors.GitError{Operation: "list branches", Err: err}
		}

		matches := []string{}
		prefix := branchConfig.Prefix + nameOrPrefix
		for _, branch := range branches {
			if strings.HasPrefix(branch, prefix) {
				matches = append(matches, branch)
			}
		}

		switch len(matches) {
		case 0:
			return &errors.BranchNotFoundError{BranchName: fullBranchName}
		case 1:
			fullBranchName = matches[0]
		default:
			return &errors.GitError{Operation: "checkout branch", Err: fmt.Errorf("ambiguous branch name '%s' matches multiple branches:\n  %s", nameOrPrefix, strings.Join(matches, "\n  "))}
		}
	}

	// Show git command if requested
	if showCommands {
		fmt.Printf("$ git checkout %s\n", fullBranchName)
	}

	// Checkout the branch
	err = git.Checkout(fullBranchName)
	if err != nil {
		return &errors.GitError{Operation: fmt.Sprintf("checkout branch '%s'", fullBranchName), Err: err}
	}

	fmt.Printf("Switched to branch '%s'\n", fullBranchName)
	return nil
}
