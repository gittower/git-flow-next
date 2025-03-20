package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/gittower/git-flow-next/config"
	"github.com/gittower/git-flow-next/errors"
	"github.com/gittower/git-flow-next/git"
	"github.com/gittower/git-flow-next/internal/update"
	"github.com/gittower/git-flow-next/model"
	"github.com/gittower/git-flow-next/util"
)

// FinishCommand is the implementation of the finish command for topic branches
func FinishCommand(branchType string, name string, continueOp bool, abortOp bool) {
	if err := executeFinish(branchType, name, continueOp, abortOp); err != nil {
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

// executeFinish performs the actual branch finishing logic and returns any errors
func executeFinish(branchType string, name string, continueOp bool, abortOp bool) error {
	// Check if there's a merge in progress
	if util.IsMergeInProgress() {
		state, err := util.LoadMergeState()
		if err != nil {
			return &errors.GitError{Operation: "load merge state", Err: err}
		}

		if abortOp {
			return handleAbort(state)
		}

		if continueOp {
			return handleContinue(state)
		}

		return &errors.MergeInProgressError{BranchName: state.FullBranchName}
	}

	// Don't allow continue or abort if no merge is in progress
	if continueOp || abortOp {
		return &errors.NoMergeInProgressError{}
	}

	// Regular finish command flow
	return finishBranch(branchType, name)
}

func finishBranch(branchType string, name string) error {
	// Validate that git-flow is initialized
	initialized, err := config.IsInitialized()
	if err != nil {
		return &errors.GitError{Operation: "check if git-flow is initialized", Err: err}
	}
	if !initialized {
		return &errors.NotInitializedError{}
	}

	// Validate inputs
	if name == "" {
		return &errors.InvalidBranchNameError{Name: name}
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

	// Get full branch name
	fullBranchName := branchConfig.Prefix + name

	// Check if branch exists
	if err := git.BranchExists(fullBranchName); err != nil {
		return &errors.BranchNotFoundError{BranchName: fullBranchName}
	}

	// Get target branch (always the parent branch)
	targetBranch := branchConfig.Parent

	// Check if target branch exists
	if err := git.BranchExists(targetBranch); err != nil {
		return &errors.BranchNotFoundError{BranchName: targetBranch}
	}

	// Finish the branch
	return finish(branchType, name, branchConfig, targetBranch, fullBranchName)
}

func handleContinue(state *model.MergeState) error {
	switch state.CurrentStep {
	case "merge":
		// Check if there are still conflicts
		if git.HasConflicts() {
			return &errors.UnresolvedConflictsError{}
		}

		// Move to next step
		state.CurrentStep = "create_tag"
		if err := util.SaveMergeState(state); err != nil {
			return &errors.GitError{Operation: "save merge state", Err: err}
		}
		return handleContinue(state)

	case "create_tag":
		// Create tag if enabled for this branch type
		cfg, err := config.LoadConfig()
		if err != nil {
			return &errors.GitError{Operation: "load configuration", Err: err}
		}

		branchConfig, ok := cfg.Branches[state.BranchType]
		if !ok {
			return &errors.GitError{Operation: fmt.Sprintf("get config for branch '%s'", state.BranchType), Err: fmt.Errorf("branch config not found")}
		}

		if branchConfig.Tag {
			tagName := state.BranchName
			if branchConfig.TagPrefix != "" {
				tagName = branchConfig.TagPrefix + state.BranchName
			}
			message := fmt.Sprintf("Tagging version %s", tagName)
			if err := createTag(tagName, state.ParentBranch, message); err != nil {
				return &errors.GitError{Operation: fmt.Sprintf("create tag '%s'", tagName), Err: err}
			}
			fmt.Printf("Created tag '%s'\n", tagName)
		}

		// Move to next step
		state.CurrentStep = "update_children"
		if err := util.SaveMergeState(state); err != nil {
			return &errors.GitError{Operation: "save merge state", Err: err}
		}
		return handleContinue(state)

	case "update_children":
		// Find next child branch to update
		var nextBranch string
		for _, branch := range state.ChildBranches {
			alreadyUpdated := false
			for _, updated := range state.UpdatedBranches {
				if branch == updated {
					alreadyUpdated = true
					break
				}
			}
			if !alreadyUpdated {
				nextBranch = branch
				break
			}
		}

		// If no more branches to update, move to final step
		if nextBranch == "" {
			state.CurrentStep = "delete_branch"
			if err := util.SaveMergeState(state); err != nil {
				return &errors.GitError{Operation: "save merge state", Err: err}
			}
			return handleContinue(state)
		}

		// Update the next child branch
		fmt.Printf("Updating child base branch '%s' from '%s'...\n", nextBranch, state.ParentBranch)

		// Load config to get merge strategy for this base branch
		cfg, err := config.LoadConfig()
		if err != nil {
			return &errors.GitError{Operation: "load configuration", Err: err}
		}

		branchConfig, ok := cfg.Branches[nextBranch]
		if !ok {
			return &errors.GitError{Operation: fmt.Sprintf("get config for branch '%s'", nextBranch), Err: fmt.Errorf("branch config not found")}
		}

		// Use the shared update logic
		err = update.UpdateBranchFromParent(nextBranch, state.ParentBranch, branchConfig.DownstreamStrategy, true, state)
		if err != nil {
			if _, ok := err.(*errors.UnresolvedConflictsError); ok {
				msg := fmt.Sprintf("Merge conflicts detected while updating base branch '%s'. Resolve conflicts and run 'git flow %s finish --continue %s'\n", nextBranch, state.BranchType, state.BranchName)
				msg += fmt.Sprintf("To abort the merge, run 'git flow %s finish --abort %s'", state.BranchType, state.BranchName)
				fmt.Println(msg)
				return err
			}
			return err
		}

		// Mark this branch as updated
		state.UpdatedBranches = append(state.UpdatedBranches, nextBranch)
		if err := util.SaveMergeState(state); err != nil {
			return &errors.GitError{Operation: "save merge state", Err: err}
		}

		// Continue with next branch
		return handleContinue(state)

	case "delete_branch":
		// Delete the original branch
		err := git.DeleteBranch(state.FullBranchName)
		if err != nil {
			return &errors.GitError{Operation: fmt.Sprintf("delete branch '%s'", state.FullBranchName), Err: err}
		}

		// Ensure we're on the parent branch at the end
		if err := git.Checkout(state.ParentBranch); err != nil {
			return &errors.GitError{Operation: fmt.Sprintf("checkout parent branch '%s'", state.ParentBranch), Err: err}
		}

		// Clear the merge state
		if err := util.ClearMergeState(); err != nil {
			return &errors.GitError{Operation: "clear merge state", Err: err}
		}

		fmt.Printf("Successfully finished branch '%s' and updated %d child base branches\n", state.FullBranchName, len(state.UpdatedBranches))
		return nil

	default:
		return &errors.GitError{Operation: fmt.Sprintf("unknown step '%s'", state.CurrentStep), Err: nil}
	}
}

func handleAbort(state *model.MergeState) error {
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
		return &errors.GitError{Operation: "abort merge", Err: err}
	}

	// Checkout the original branch
	if err := git.Checkout(state.FullBranchName); err != nil {
		return &errors.GitError{Operation: fmt.Sprintf("checkout original branch '%s'", state.FullBranchName), Err: err}
	}

	// Clear the merge state
	if err := util.ClearMergeState(); err != nil {
		return &errors.GitError{Operation: "clear merge state", Err: err}
	}

	return nil
}

func finish(branchType string, name string, branchConfig config.BranchConfig, targetBranch string, fullBranchName string) error {
	// Check if we're in a merge state
	if util.IsMergeInProgress() {
		state, err := util.LoadMergeState()
		if err != nil {
			return &errors.GitError{Operation: "load merge state", Err: err}
		}
		return handleContinue(state)
	}

	// Checkout target branch
	err := git.Checkout(targetBranch)
	if err != nil {
		return &errors.GitError{Operation: fmt.Sprintf("checkout target branch '%s'", targetBranch), Err: err}
	}
	fmt.Printf("Switched to branch '%s'\n", targetBranch)

	// Find child base branches that need to be updated
	cfg, err := config.LoadConfig()
	if err != nil {
		return &errors.GitError{Operation: "load configuration", Err: err}
	}

	childBranches := []string{}
	for branchName, branch := range cfg.Branches {
		if branch.Type == string(config.BranchTypeBase) && branch.Parent == targetBranch {
			fmt.Printf("Found child base branch '%s' to update\n", branchName)
			childBranches = append(childBranches, branchName)
		}
	}

	// Save merge state before starting
	state := &model.MergeState{
		Action:          "finish",
		BranchType:      branchType,
		BranchName:      name,
		CurrentStep:     "merge",
		ParentBranch:    targetBranch,
		MergeStrategy:   branchConfig.UpstreamStrategy,
		FullBranchName:  fullBranchName,
		ChildBranches:   childBranches,
		UpdatedBranches: []string{},
	}
	if err := util.SaveMergeState(state); err != nil {
		return &errors.GitError{Operation: "save merge state", Err: err}
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
			return &errors.GitError{Operation: "checkout feature branch for rebase", Err: err}
		}
		// 2. Rebase onto target branch
		mergeErr = git.Rebase(targetBranch)
		if mergeErr == nil {
			// 3. If rebase succeeds, checkout target and merge (should be fast-forward)
			err = git.Checkout(targetBranch)
			if err != nil {
				return &errors.GitError{Operation: "checkout target branch after rebase", Err: err}
			}
			mergeErr = git.Merge(fullBranchName)
		}
	case "squash":
		mergeErr = git.SquashMerge(fullBranchName)
	case "merge":
		mergeErr = git.Merge(fullBranchName)
	default:
		return &errors.GitError{Operation: fmt.Sprintf("unknown merge strategy: %s", strings.ToLower(branchConfig.UpstreamStrategy)), Err: nil}
	}

	if mergeErr != nil {
		if strings.Contains(mergeErr.Error(), "conflict") {
			msg := fmt.Sprintf("Merge conflicts detected. Resolve conflicts and run 'git flow %s finish --continue %s'\n", branchType, name)
			msg += fmt.Sprintf("To abort the merge, run 'git flow %s finish --abort %s'", branchType, name)
			fmt.Println(msg)
			return &errors.UnresolvedConflictsError{}
		}
		return &errors.GitError{Operation: "merge branch", Err: mergeErr}
	}

	// Move to next step (tag creation)
	state.CurrentStep = "create_tag"
	if err := util.SaveMergeState(state); err != nil {
		return &errors.GitError{Operation: "save merge state", Err: err}
	}

	return handleContinue(state)
}

// createTag creates a new Git tag with the given name and message
func createTag(tagName, targetBranch, message string) error {
	// Check if tag already exists
	cmd := exec.Command("git", "show-ref", "--tags", tagName)
	if err := cmd.Run(); err == nil {
		// Tag exists
		return nil
	}

	// Create annotated tag
	cmd = exec.Command("git", "tag", "-a", tagName, "-m", message)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create tag: %w", err)
	}

	return nil
}
