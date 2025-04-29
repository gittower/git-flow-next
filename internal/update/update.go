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
func UpdateBranchFromParent(branchName string, parentBranch string, strategy string, saveState bool, state *mergestate.MergeState) error {
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
	var mergeErr error
	switch strings.ToLower(strategy) {
	case "rebase":
		fmt.Printf("Using rebase strategy for '%s'\n", branchName)
		mergeErr = git.Rebase(parentBranch)
	case "squash":
		fmt.Printf("Using squash strategy for '%s'\n", branchName)
		mergeErr = git.SquashMerge(parentBranch)
	default:
		fmt.Printf("Using merge strategy for '%s'\n", branchName)
		mergeErr = git.Merge(parentBranch)
	}

	if mergeErr != nil {
		if strings.Contains(mergeErr.Error(), "conflict") {
			if saveState && state != nil {
				// Save merge state if requested
				if err := mergestate.SaveMergeState(state); err != nil {
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
func GetParentBranch(branchName string) (string, error) {
	// Get configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		return "", &errors.GitError{Operation: "load configuration", Err: err}
	}

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
