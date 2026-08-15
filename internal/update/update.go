package update

import (
	"fmt"
	"strings"

	"github.com/gittower/git-flow-next/internal/config"
	"github.com/gittower/git-flow-next/internal/errors"
	"github.com/gittower/git-flow-next/internal/git"
	"github.com/gittower/git-flow-next/internal/mergestate"
)

// UpdateBranchFromParent updates a branch with changes from its parent branch using the configured strategy
func UpdateBranchFromParent(repo *git.Repo, branchName string, parentBranch string, strategy string, saveState bool, state *mergestate.MergeState) error {
	return UpdateBranchFromParentWithMessage(repo, branchName, parentBranch, strategy, "", saveState, state)
}

// UpdateBranchFromParentWithMessage updates a branch with changes from its parent branch using the configured strategy and optional custom message
func UpdateBranchFromParentWithMessage(repo *git.Repo, branchName string, parentBranch string, strategy string, customMessage string, saveState bool, state *mergestate.MergeState) error {
	// Checkout the branch if needed
	currentBranch, err := repo.GetCurrentBranch()
	if err != nil {
		return &errors.GitError{Operation: "get current branch", Err: err}
	}
	if currentBranch != branchName {
		if err := repo.Checkout(branchName); err != nil {
			return &errors.GitError{Operation: fmt.Sprintf("checkout branch '%s'", branchName), Err: err}
		}
	}

	// Use the configured merge strategy
	// Note: noVerify is false for update operations (only finish supports --no-verify)
	var mergeErr error
	switch strings.ToLower(strategy) {
	case "rebase":
		fmt.Printf("Using rebase strategy for '%s'\n", branchName)
		mergeErr = repo.Rebase(parentBranch)
	case "squash":
		fmt.Printf("Using squash strategy for '%s'\n", branchName)
		if customMessage != "" {
			mergeErr = repo.MergeSquashWithMessage(parentBranch, customMessage, false)
		} else {
			mergeErr = repo.SquashMerge(parentBranch, false)
		}
	default:
		fmt.Printf("Using merge strategy for '%s'\n", branchName)
		if customMessage != "" {
			// noFF: a child base branch always records the update as a merge commit.
			// ffOnly and noVerify stay off.
			mergeErr = repo.MergeWithMessage(parentBranch, customMessage, true, false, false)
		} else {
			mergeErr = repo.Merge(parentBranch, false)
		}
	}

	if mergeErr != nil {
		if strings.Contains(mergeErr.Error(), "conflict") {
			if saveState && state != nil {
				// Save merge state if requested
				if err := mergestate.SaveMergeState(repo, state); err != nil {
					return &errors.GitError{Operation: "save merge state", Err: err}
				}
			}
			return &errors.UnresolvedConflictsError{}
		}
		return &errors.GitError{Operation: fmt.Sprintf("merge %s into %s", parentBranch, branchName), Err: mergeErr}
	}

	fmt.Printf("Successfully updated branch '%s' from '%s'\n", branchName, parentBranch)
	return nil
}

// GetParentBranch returns the parent branch for a given branch name
func GetParentBranch(cfg *config.Config, branchName string) (string, error) {
	// Find the branch type and its configuration
	var branchConfig *config.BranchConfig
	// First check if it's a base branch (main or develop)
	for branchKey, bc := range cfg.Branches {
		if bc.Type == string(config.BranchTypeBase) && branchKey == branchName {
			bc := bc // Create new variable to avoid taking address of range variable
			branchConfig = &bc
			break
		}
	}
	// If not a base branch, check topic branches by prefix
	if branchConfig == nil {
		for _, bc := range cfg.Branches {
			if bc.Type == string(config.BranchTypeTopic) && bc.Prefix != "" && strings.HasPrefix(branchName, bc.Prefix) {
				bc := bc // Create new variable to avoid taking address of range variable
				branchConfig = &bc
				break
			}
		}
	}

	if branchConfig == nil {
		return "", &errors.InvalidBranchTypeError{BranchType: branchName}
	}

	// Get parent branch from config
	parentBranch := branchConfig.Parent
	if parentBranch == "" {
		return "", &errors.GitError{Operation: "get parent branch", Err: fmt.Errorf("no parent branch configured for branch type")}
	}
	return parentBranch, nil
}
