package config

import (
	"fmt"
	"strconv"

	"github.com/gittower/git-flow-next/internal/git"
)

// SaveConfig saves the git-flow configuration to Git config
func SaveConfig(config *Config) error {
	// Set git-flow version
	err := git.SetConfig("gitflow.version", config.Version)
	if err != nil {
		return fmt.Errorf("failed to set gitflow.version: %w", err)
	}

	// Save branch configurations
	for branchName, branchConfig := range config.Branches {
		// Set branch type
		err = git.SetConfig(fmt.Sprintf("gitflow.branch.%s.type", branchName), branchConfig.Type)
		if err != nil {
			return fmt.Errorf("failed to set branch type for %s: %w", branchName, err)
		}

		// Set parent branch if it exists
		if branchConfig.Parent != "" {
			err = git.SetConfig(fmt.Sprintf("gitflow.branch.%s.parent", branchName), branchConfig.Parent)
			if err != nil {
				return fmt.Errorf("failed to set parent branch for %s: %w", branchName, err)
			}
		}

		// Set start point if it exists
		if branchConfig.StartPoint != "" {
			err = git.SetConfig(fmt.Sprintf("gitflow.branch.%s.startPoint", branchName), branchConfig.StartPoint)
			if err != nil {
				return fmt.Errorf("failed to set start point for %s: %w", branchName, err)
			}
		}

		// Set upstream strategy if it exists
		if branchConfig.UpstreamStrategy != "" {
			err = git.SetConfig(fmt.Sprintf("gitflow.branch.%s.upstreamStrategy", branchName), branchConfig.UpstreamStrategy)
			if err != nil {
				return fmt.Errorf("failed to set upstream strategy for %s: %w", branchName, err)
			}
		}

		// Set downstream strategy if it exists
		if branchConfig.DownstreamStrategy != "" {
			err = git.SetConfig(fmt.Sprintf("gitflow.branch.%s.downstreamStrategy", branchName), branchConfig.DownstreamStrategy)
			if err != nil {
				return fmt.Errorf("failed to set downstream strategy for %s: %w", branchName, err)
			}
		}

		// Set prefix if it exists
		if branchConfig.Prefix != "" {
			err = git.SetConfig(fmt.Sprintf("gitflow.branch.%s.prefix", branchName), branchConfig.Prefix)
			if err != nil {
				return fmt.Errorf("failed to set prefix for %s: %w", branchName, err)
			}
		}

		// Set auto update
		err = git.SetConfig(fmt.Sprintf("gitflow.branch.%s.autoUpdate", branchName), strconv.FormatBool(branchConfig.AutoUpdate))
		if err != nil {
			return fmt.Errorf("failed to set auto update for %s: %w", branchName, err)
		}

		// Set tag configuration only if true (false is default)
		if branchConfig.Tag {
			err = git.SetConfig(fmt.Sprintf("gitflow.branch.%s.tag", branchName), "true")
			if err != nil {
				return fmt.Errorf("failed to set tag configuration for %s: %w", branchName, err)
			}
		}

		// Set tag prefix if it exists
		if branchConfig.TagPrefix != "" {
			err = git.SetConfig(fmt.Sprintf("gitflow.branch.%s.tagprefix", branchName), branchConfig.TagPrefix)
			if err != nil {
				return fmt.Errorf("failed to set tag prefix for %s: %w", branchName, err)
			}
		}
	}

	return nil
}

// MarkRepoInitialized marks the repository as initialized with git-flow
func MarkRepoInitialized() error {
	// This is effectively done by setting the gitflow.version in SaveConfig
	// But we'll add a specific initialized flag for clarity
	err := git.SetConfig("gitflow.initialized", "true")
	if err != nil {
		return fmt.Errorf("failed to mark repository as initialized: %w", err)
	}
	return nil
}

// ClearConfig removes all git-flow configuration
func ClearConfig() error {
	// Get all gitflow.* config entries
	configs, err := git.GetAllConfig("gitflow\\.")
	if err != nil {
		return fmt.Errorf("failed to get gitflow configurations: %w", err)
	}

	// Remove each config entry
	for key := range configs {
		err = git.UnsetConfig(key)
		if err != nil {
			return fmt.Errorf("failed to unset %s: %w", key, err)
		}
	}

	return nil
}
