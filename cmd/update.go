package cmd

import (
	"fmt"
	"strings"

	"github.com/gittower/git-flow-next/internal/config"
	"github.com/gittower/git-flow-next/internal/errors"
	"github.com/gittower/git-flow-next/internal/git"
	"github.com/gittower/git-flow-next/internal/hooks"
	"github.com/gittower/git-flow-next/internal/mergestate"
	"github.com/gittower/git-flow-next/internal/update"
)

// Note: The update command is registered in two places:
// 1. As a shorthand command in shorthand.go for "git flow update"
// 2. As subcommands of topic branches in topicbranch.go for "git flow <topic> update"
// This file only contains the shared executeUpdate function used by both.

// executeUpdate updates a branch with changes from its parent branch
func executeUpdate(branchType string, name string, useRebase bool) error {
	// Validate that git-flow is initialized
	initialized, err := config.IsInitialized()
	if err != nil {
		return &errors.GitError{Operation: "check if git-flow is initialized", Err: err}
	}
	if !initialized {
		return &errors.NotInitializedError{}
	}

	// Get configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		return &errors.GitError{Operation: "load configuration", Err: err}
	}

	var branchName string
	var shortName string
	var detectedBranchType string

	if branchType != "" {
		detectedBranchType = branchType
		// If branch type is specified, construct full branch name
		if name == "" {
			// If no name provided, try to get current branch and verify it's of the correct type
			currentBranch, err := git.GetCurrentBranch()
			if err != nil {
				return &errors.GitError{Operation: "get current branch", Err: err}
			}
			branchConfig, ok := cfg.Branches[branchType]
			if !ok {
				return &errors.InvalidBranchTypeError{BranchType: branchType}
			}
			if !strings.HasPrefix(currentBranch, branchConfig.Prefix) {
				return &errors.GitError{Operation: "validate current branch", Err: fmt.Errorf("current branch is not a %s branch", branchType)}
			}
			branchName = currentBranch
			shortName = strings.TrimPrefix(currentBranch, branchConfig.Prefix)
		} else {
			// Construct full branch name from type and name
			branchConfig, ok := cfg.Branches[branchType]
			if !ok {
				return &errors.InvalidBranchTypeError{BranchType: branchType}
			}
			branchName = branchConfig.Prefix + name
			shortName = name
		}
	} else {
		// No branch type specified, use provided branch name or current branch
		if name == "" {
			currentBranch, err := git.GetCurrentBranch()
			if err != nil {
				return &errors.GitError{Operation: "get current branch", Err: err}
			}
			branchName = currentBranch
		} else {
			branchName = name
		}
		// Try to detect branch type from branch name
		detectedBranchType, shortName = detectBranchTypeFromName(cfg, branchName)
	}

	// Check if branch exists
	if err := git.BranchExists(branchName); err != nil {
		return &errors.BranchNotFoundError{BranchName: branchName}
	}

	// Get parent branch
	parentBranch, err := update.GetParentBranch(cfg, branchName)
	if err != nil {
		return err
	}

	// Check if parent branch exists
	if err := git.BranchExists(parentBranch); err != nil {
		return &errors.BranchNotFoundError{BranchName: parentBranch}
	}

	// Get branch configuration for merge strategy
	var strategy string
	for branchKey, bc := range cfg.Branches {
		if bc.Type == string(config.BranchTypeBase) && branchKey == branchName {
			strategy = bc.DownstreamStrategy
			break
		}
		if bc.Type == string(config.BranchTypeTopic) && bc.Prefix != "" && strings.HasPrefix(branchName, bc.Prefix) {
			strategy = bc.DownstreamStrategy
			break
		}
	}

	if strategy == "" {
		strategy = "merge" // Default to merge if no strategy configured
	}

	// Override strategy if --rebase flag is set
	if useRebase {
		strategy = "rebase"
	}

	// Create merge state
	state := &mergestate.MergeState{
		Action:         "update",
		BranchName:     branchName,
		ParentBranch:   parentBranch,
		MergeStrategy:  strategy,
		CurrentStep:    "merge",
		FullBranchName: branchName,
	}

	// If we detected a branch type, run with hooks
	if detectedBranchType != "" {
		// Get git directory for hooks
		gitDir, err := git.GetGitDir()
		if err != nil {
			return &errors.GitError{Operation: "get git directory", Err: err}
		}

		// Get remote name from config
		remoteName := cfg.Remote

		// Build hook context
		hookCtx := hooks.HookContext{
			BranchType: detectedBranchType,
			BranchName: shortName,
			FullBranch: branchName,
			BaseBranch: parentBranch,
			Origin:     remoteName,
		}
		if detectedBranchType == "release" || detectedBranchType == "hotfix" {
			hookCtx.Version = shortName
		}

		// Run update operation wrapped with hooks
		return hooks.WithHooks(gitDir, detectedBranchType, hooks.HookActionUpdate, hookCtx, func() error {
			return update.UpdateBranchFromParent(branchName, parentBranch, strategy, true, state)
		})
	}

	// No branch type detected, run without hooks
	return update.UpdateBranchFromParent(branchName, parentBranch, strategy, true, state)
}

// detectBranchTypeFromName detects the branch type and short name from a full branch name
func detectBranchTypeFromName(cfg *config.Config, branchName string) (string, string) {
	for branchType, bc := range cfg.Branches {
		if bc.Type == string(config.BranchTypeTopic) && bc.Prefix != "" && strings.HasPrefix(branchName, bc.Prefix) {
			return branchType, strings.TrimPrefix(branchName, bc.Prefix)
		}
	}
	return "", branchName
}
