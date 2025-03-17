package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/gittower/git-flow-next/config"
	"github.com/gittower/git-flow-next/git"
	"github.com/gittower/git-flow-next/model"
	"github.com/gittower/git-flow-next/util"
)

// FinishCommand is the implementation of the finish command for topic branches
func FinishCommand(branchType string, name string, continueOp bool, abortOp bool) {
	// Check if there's a merge in progress
	if util.IsMergeInProgress() {
		state, err := util.LoadMergeState()
		if err != nil {
			fmt.Printf("Error loading merge state: %v\n", err)
			return
		}

		if abortOp {
			handleAbort(state)
			return
		}

		if continueOp {
			handleContinue(state)
			return
		}

		fmt.Printf("A merge is already in progress for branch '%s'. Use --continue or --abort.\n", state.FullBranchName)
		return
	}

	// Don't allow continue or abort if no merge is in progress
	if continueOp || abortOp {
		fmt.Println("No merge in progress. Nothing to continue or abort.")
		return
	}

	// Regular finish command flow
	finishBranch(branchType, name)
}

func finishBranch(branchType string, name string) {
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

	// Get target branch (always the parent branch)
	targetBranch := branchConfig.Parent

	// Check if target branch exists
	if !git.BranchExists(targetBranch) {
		fmt.Printf("Target branch '%s' does not exist\n", targetBranch)
		return
	}

	// Finish the branch
	if err := finish(branchType, name, branchConfig, targetBranch, fullBranchName); err != nil {
		os.Exit(1)
	}
}

func handleContinue(state *model.MergeState) {
	// Check if there are still conflicts
	if git.HasConflicts() {
		fmt.Println("There are still unresolved conflicts. Resolve them and try again.")
		return
	}

	// Complete the merge
	completeMerge(state)
}

func handleAbort(state *model.MergeState) {
	// Abort the merge based on strategy
	var err error
	switch state.MergeStrategy {
	case "merge":
		err = git.MergeAbort()
	case "rebase":
		err = git.RebaseAbort()
	default:
		err = git.MergeAbort() // Default to merge abort
	}

	if err != nil {
		fmt.Printf("Error aborting merge: %v\n", err)
		return
	}

	// Checkout the original branch
	if err := git.Checkout(state.FullBranchName); err != nil {
		fmt.Printf("Error checking out original branch: %v\n", err)
	}

	// Clear the merge state
	if err := util.ClearMergeState(); err != nil {
		fmt.Printf("Error clearing merge state: %v\n", err)
		return
	}

	fmt.Printf("Merge aborted. Returned to branch '%s'\n", state.FullBranchName)
}

func completeMerge(state *model.MergeState) {
	// Delete branch
	err := git.DeleteBranch(state.FullBranchName)
	if err != nil {
		fmt.Printf("Error deleting branch '%s': %v\n", state.FullBranchName, err)
		return
	}
	fmt.Printf("Deleted branch '%s'\n", state.FullBranchName)

	// Clear the merge state
	if err := util.ClearMergeState(); err != nil {
		fmt.Printf("Error clearing merge state: %v\n", err)
		return
	}

	fmt.Printf("Successfully finished branch '%s'\n", state.FullBranchName)
}

func finish(branchType string, name string, branchConfig config.BranchConfig, targetBranch string, fullBranchName string) error {
	// Check if we're in a merge state
	if util.IsMergeInProgress() {
		state, err := util.LoadMergeState()
		if err != nil {
			fmt.Printf("Error loading merge state: %v\n", err)
			return err
		}
		handleContinue(state)
		return nil
	}

	// Get current branch
	currentBranch, err := git.GetCurrentBranch()
	if err != nil {
		fmt.Printf("Error getting current branch: %v\n", err)
		return err
	}

	// Check if we're on the branch to finish
	if currentBranch != fullBranchName {
		// Checkout the branch to finish
		err = git.Checkout(fullBranchName)
		if err != nil {
			fmt.Printf("Error checking out branch '%s': %v\n", fullBranchName, err)
			return err
		}
		fmt.Printf("Switched to branch '%s'\n", fullBranchName)
	}

	// Checkout target branch
	err = git.Checkout(targetBranch)
	if err != nil {
		fmt.Printf("Error checking out target branch '%s': %v\n", targetBranch, err)
		return err
	}
	fmt.Printf("Switched to branch '%s'\n", targetBranch)

	// Save merge state before starting
	state := &model.MergeState{
		Action:         "finish",
		BranchType:     branchType,
		BranchName:     name,
		CurrentStep:    "merge",
		ParentBranch:   targetBranch,
		MergeStrategy:  branchConfig.UpstreamStrategy,
		FullBranchName: fullBranchName,
	}
	if err := util.SaveMergeState(state); err != nil {
		fmt.Printf("Error saving merge state: %v\n", err)
		return err
	}

	// Perform merge based on strategy
	fmt.Printf("Merging using strategy: %v\n", strings.ToLower(branchConfig.UpstreamStrategy))
	var mergeErr error
	switch strings.ToLower(branchConfig.UpstreamStrategy) {
	case "rebase":
		fmt.Printf("Rebase strategy selected\n")
		// For rebase, we need to:
		// 1. Stay on feature branch
		err = git.Checkout(fullBranchName)
		if err != nil {
			fmt.Printf("Error checking out feature branch: %v\n", err)
			return err
		}
		// 2. Rebase onto target branch
		mergeErr = git.Rebase(targetBranch)
		if mergeErr == nil {
			// 3. If rebase succeeds, checkout target and merge (should be fast-forward)
			err = git.Checkout(targetBranch)
			if err != nil {
				fmt.Printf("Error checking out target branch: %v\n", err)
				return err
			}
			mergeErr = git.Merge(fullBranchName)
		}
	case "squash":
		mergeErr = git.SquashMerge(fullBranchName)
	case "merge":
		mergeErr = git.Merge(fullBranchName)
	default:
		err := fmt.Errorf("unknown merge strategy: %s", strings.ToLower(branchConfig.UpstreamStrategy))
		fmt.Println(err)
		return err
	}

	if mergeErr != nil {
		if strings.Contains(mergeErr.Error(), "conflict") {
			msg := fmt.Sprintf("Merge conflicts detected. Resolve conflicts and run 'git flow %s finish --continue %s'\n", branchType, name)
			msg += fmt.Sprintf("To abort the merge, run 'git flow %s finish --abort %s'", branchType, name)
			fmt.Println(msg)
			return fmt.Errorf("merge conflict: %v", mergeErr)
		}
		fmt.Printf("Error merging branch: %v\n", mergeErr)
		util.ClearMergeState()
		return mergeErr
	}

	// Complete the merge
	completeMerge(state)
	return nil
}
