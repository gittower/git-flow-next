package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/gittower/git-flow-next/internal/config"
	"github.com/gittower/git-flow-next/internal/errors"
	"github.com/gittower/git-flow-next/internal/git"
)

// CheckoutCommand handles checking out a topic branch
func CheckoutCommand(branchType string, nameOrPrefix string, showCommands bool) {
	repo := mustOpenRepo()
	if err := executeCheckout(repo, branchType, nameOrPrefix, showCommands); err != nil {
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

// executeCheckout performs the actual branch checkout logic and returns any errors
func executeCheckout(repo *git.Repo, branchType string, nameOrPrefix string, showCommands bool) error {
	// Load configuration
	cfg, err := config.Load(repo)
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
		branches, err := repo.ListBranches()
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
	err = repo.BranchExists(fullBranchName)
	if err != nil {
		// If exact match not found, try prefix match
		branches, err := repo.ListBranches()
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
	err = repo.Checkout(fullBranchName)
	if err != nil {
		return &errors.GitError{Operation: fmt.Sprintf("checkout branch '%s'", fullBranchName), Err: err}
	}

	fmt.Printf("Switched to branch '%s'\n", fullBranchName)
	return nil
}
