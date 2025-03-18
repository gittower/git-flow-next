package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/gittower/git-flow-next/config"
	"github.com/gittower/git-flow-next/errors"
	"github.com/gittower/git-flow-next/git"
	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update [branch]",
	Short: "Update a branch with changes from its parent branch",
	Long: `Update a branch with changes from its parent branch.
This command will update the specified branch (or current branch if none specified)
with changes from its parent branch using the configured downstream strategy (merge or rebase).
If merge conflicts occur, they will be handled according to the configured merge state handling.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		var branchName string
		if len(args) > 0 {
			branchName = args[0]
		}
		if err := executeUpdate(branchName); err != nil {
			var exitCode errors.ExitCode
			if flowErr, ok := err.(errors.Error); ok {
				exitCode = flowErr.ExitCode()
			} else {
				exitCode = errors.ExitCodeGitError
			}
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(int(exitCode))
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(updateCmd)
}

// executeUpdate updates a branch with changes from its parent branch
func executeUpdate(branchName string) error {
	// Validate that git-flow is initialized
	initialized, err := config.IsInitialized()
	if err != nil {
		return &errors.GitError{Operation: "check if git-flow is initialized", Err: err}
	}
	if !initialized {
		return &errors.NotInitializedError{}
	}

	// Get current branch if none specified
	if branchName == "" {
		currentBranch, err := git.GetCurrentBranch()
		if err != nil {
			return &errors.GitError{Operation: "get current branch", Err: err}
		}
		branchName = currentBranch
	}

	// Check if branch exists
	if !git.BranchExists(branchName) {
		return &errors.BranchNotFoundError{BranchName: branchName}
	}

	// Get configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		return &errors.GitError{Operation: "load configuration", Err: err}
	}

	// Find the branch type and its configuration
	var branchConfig *config.BranchConfig
	for _, bc := range cfg.Branches {
		if bc.Prefix != "" && strings.HasPrefix(branchName, bc.Prefix) {
			bc := bc // Create new variable to avoid taking address of range variable
			branchConfig = &bc
			break
		}
	}

	if branchConfig == nil {
		return &errors.InvalidBranchTypeError{BranchType: branchName}
	}

	// Get parent branch from config
	parentBranch := branchConfig.Parent
	if parentBranch == "" {
		return &errors.GitError{Operation: "get parent branch", Err: fmt.Errorf("no parent branch configured for branch type")}
	}

	// Check if parent branch exists
	if !git.BranchExists(parentBranch) {
		return &errors.BranchNotFoundError{BranchName: parentBranch}
	}

	// Checkout the branch if needed
	currentBranch, err := git.GetCurrentBranch()
	if err != nil {
		return &errors.GitError{Operation: "get current branch", Err: err}
	}
	if currentBranch != branchName {
		if err := git.Checkout(branchName); err != nil {
			return &errors.GitError{Operation: fmt.Sprintf("checkout branch '%s'", branchName), Err: err}
		}
	}

	// Use the configured merge strategy
	if err := updateWithMerge(branchName, parentBranch); err != nil {
		return err
	}

	fmt.Printf("Successfully updated branch %s with changes from %s\n", branchName, parentBranch)
	return nil
}

func updateWithMerge(branchName, parentBranch string) error {
	// Merge parent branch
	if err := git.Merge(parentBranch); err != nil {
		if strings.Contains(err.Error(), "merge conflict") {
			fmt.Printf("Merge conflicts detected. Please resolve them and then:\n")
			fmt.Printf("1. git add <resolved-files>\n")
			fmt.Printf("2. git commit\n")
			fmt.Printf("Or to abort: git merge --abort\n")
			return &errors.UnresolvedConflictsError{}
		}
		if strings.Contains(err.Error(), "Already up to date") {
			// Not an error, just no changes to merge
			return nil
		}
		return &errors.GitError{Operation: fmt.Sprintf("merge %s into %s", parentBranch, branchName), Err: err}
	}
	return nil
}

func updateWithRebase(branchName, parentBranch string) error {
	// Rebase onto parent branch
	if err := git.Rebase(parentBranch); err != nil {
		if strings.Contains(err.Error(), "rebase conflict") {
			fmt.Printf("Rebase conflicts detected. Please resolve them and then:\n")
			fmt.Printf("1. git add <resolved-files>\n")
			fmt.Printf("2. git rebase --continue\n")
			fmt.Printf("Or to abort: git rebase --abort\n")
			return &errors.UnresolvedConflictsError{}
		}
		return &errors.GitError{Operation: fmt.Sprintf("rebase %s onto %s", branchName, parentBranch), Err: err}
	}
	return nil
}
