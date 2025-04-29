// Package config provides configuration handling for git-flow
package config

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/gittower/git-flow-next/internal/git"
)

//
// Types and constants
//

// Config represents the git-flow configuration
type Config struct {
	Version  string
	Branches map[string]BranchConfig
	Remote   string // Name of the remote to use for all operations
}

// BranchConfig represents the configuration for a branch type
type BranchConfig struct {
	Type               string
	Parent             string
	StartPoint         string
	UpstreamStrategy   string
	DownstreamStrategy string
	Prefix             string
	AutoUpdate         bool
	Tag                bool   // whether to create a tag when finishing
	TagPrefix          string // prefix to use for tag names
}

// MergeStrategy represents the strategy for merging branches
type MergeStrategy string

const (
	// MergeStrategyNone represents no merge strategy
	MergeStrategyNone MergeStrategy = "none"
	// MergeStrategyMerge represents a standard merge
	MergeStrategyMerge MergeStrategy = "merge"
	// MergeStrategyRebase represents a rebase merge
	MergeStrategyRebase MergeStrategy = "rebase"
	// MergeStrategySquash represents a squash merge
	MergeStrategySquash MergeStrategy = "squash"
)

// BranchType represents the type of branch
type BranchType string

const (
	// BranchTypeBase represents a base branch (main, develop)
	BranchTypeBase BranchType = "base"
	// BranchTypeTopic represents a topic branch (feature, release, hotfix)
	BranchTypeTopic BranchType = "topic"
)

// ConfigOverrides represents the overrides that can be applied to a Config
type ConfigOverrides struct {
	MainBranch    string // Name of the main branch
	DevelopBranch string // Name of the develop branch
	FeaturePrefix string // Prefix for feature branches
	BugfixPrefix  string // Prefix for bugfix branches
	ReleasePrefix string // Prefix for release branches
	HotfixPrefix  string // Prefix for hotfix branches
	SupportPrefix string // Prefix for support branches
	TagPrefix     string // Prefix for tags
}

//
// Loading and initialization functions
//

// DefaultConfig returns a default git-flow configuration
func DefaultConfig() *Config {
	return &Config{
		Version: "1.0",
		Remote:  "origin", // Default remote name
		Branches: map[string]BranchConfig{
			"main": {
				Type:               string(BranchTypeBase),
				Parent:             "",
				UpstreamStrategy:   string(MergeStrategyNone),
				DownstreamStrategy: string(MergeStrategyNone),
				AutoUpdate:         false,
			},
			"develop": {
				Type:               string(BranchTypeBase),
				Parent:             "main",
				UpstreamStrategy:   string(MergeStrategyMerge),
				DownstreamStrategy: string(MergeStrategyMerge),
				AutoUpdate:         true,
			},
			"feature": {
				Type:               string(BranchTypeTopic),
				Parent:             "develop",
				StartPoint:         "develop",
				UpstreamStrategy:   string(MergeStrategyMerge),
				DownstreamStrategy: string(MergeStrategyRebase),
				Prefix:             "feature/",
			},
			"release": {
				Type:               string(BranchTypeTopic),
				Parent:             "main",
				StartPoint:         "develop",
				UpstreamStrategy:   string(MergeStrategyMerge),
				DownstreamStrategy: string(MergeStrategyMerge),
				Prefix:             "release/",
				Tag:                true, // Enable tagging by default
				TagPrefix:          "",   // No default prefix, will be asked during init
			},
			"hotfix": {
				Type:               string(BranchTypeTopic),
				Parent:             "main",
				StartPoint:         "main",
				UpstreamStrategy:   string(MergeStrategyMerge),
				DownstreamStrategy: string(MergeStrategyRebase),
				Prefix:             "hotfix/",
				Tag:                true, // Enable tagging by default
				TagPrefix:          "",   // No default prefix, will be asked during init
			},
			"support": {
				Type:               string(BranchTypeTopic),
				Parent:             "main",
				StartPoint:         "main",
				UpstreamStrategy:   string(MergeStrategyNone),
				DownstreamStrategy: string(MergeStrategyNone),
				Prefix:             "support/",
			},
		},
	}
}

// LoadConfig loads the git-flow configuration from Git config
func LoadConfig() (*Config, error) {
	// Get current directory for git operations
	currentDir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current directory: %w", err)
	}

	// Check if git-flow is initialized
	initialized, err := IsInitialized()
	if err != nil {
		return nil, fmt.Errorf("failed to check if git-flow is initialized: %w", err)
	}

	if !initialized {
		// If not initialized, return default config
		return DefaultConfig(), nil
	}

	// Get git-flow version
	version, err := git.GetConfigInDir(currentDir, "gitflow.version")
	if err != nil {
		// If no version is set, assume it's not initialized properly
		return DefaultConfig(), nil
	}

	// Create config with version
	config := &Config{
		Version:  version,
		Remote:   "origin", // Default remote
		Branches: make(map[string]BranchConfig),
	}

	// Get custom remote name if set
	remote, err := git.GetConfigInDir(currentDir, "gitflow.origin")
	if err == nil && remote != "" {
		config.Remote = remote
	}

	// Get all gitflow.branch.* config entries
	// We need to adapt GetAllConfig to work with directory
	cmd := exec.Command("git", "config", "--get-regexp", "gitflow\\.branch\\.")
	cmd.Dir = currentDir
	output, err := cmd.Output()

	// Process branch configurations from command output
	branchMap := make(map[string]map[string]string)

	if err == nil {
		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			if line == "" {
				continue
			}

			parts := strings.SplitN(line, " ", 2)
			if len(parts) != 2 {
				continue
			}

			key := parts[0]
			value := parts[1]

			// Parse key: gitflow.branch.<branchname>.<property>
			keyParts := strings.Split(key, ".")
			if len(keyParts) < 4 {
				continue
			}

			branchName := strings.ToLower(keyParts[2])
			property := strings.ToLower(keyParts[3])

			// Initialize branch map if needed
			if _, ok := branchMap[branchName]; !ok {
				branchMap[branchName] = make(map[string]string)
			}

			// Add property to branch map
			branchMap[branchName][property] = value
		}
	}

	// Convert branch map to BranchConfig objects
	for branchName, properties := range branchMap {
		branchConfig := BranchConfig{
			Type:               properties["type"],
			Parent:             properties["parent"],
			StartPoint:         properties["startpoint"],
			UpstreamStrategy:   properties["upstreamstrategy"],
			DownstreamStrategy: properties["downstreamstrategy"],
			Prefix:             properties["prefix"],
		}

		// Handle boolean properties
		if autoUpdate, ok := properties["autoupdate"]; ok {
			branchConfig.AutoUpdate = autoUpdate == "true"
		}
		if tag, ok := properties["tag"]; ok {
			branchConfig.Tag = tag == "true"
		}

		// Handle tag prefix
		if tagPrefix, ok := properties["tagprefix"]; ok {
			branchConfig.TagPrefix = tagPrefix
		}

		// Add branch config to config
		config.Branches[branchName] = branchConfig
	}

	// If no branches were loaded, use default config
	if len(config.Branches) == 0 {
		return DefaultConfig(), nil
	}

	return config, nil
}

// IsInitialized checks if git-flow is initialized in the repository
func IsInitialized() (bool, error) {
	// Get current directory for git operations
	currentDir, err := os.Getwd()
	if err != nil {
		return false, fmt.Errorf("failed to get current directory: %w", err)
	}

	version, err := git.GetConfigInDir(currentDir, "gitflow.version")
	if err != nil {
		// If error is because the key doesn't exist, it's not initialized
		return false, nil
	}
	return version != "", nil
}

// CheckGitFlowAVHConfig checks if git-flow-avh configuration exists
func CheckGitFlowAVHConfig() bool {
	// Get current directory for git operations
	currentDir, err := os.Getwd()
	if err != nil {
		return false
	}

	// Check for gitflow.branch.master (used in git-flow-avh)
	master, err := git.GetConfigInDir(currentDir, "gitflow.branch.master")
	if err == nil && master != "" {
		return true
	}

	// Check for gitflow.prefix.feature (used in git-flow-avh)
	featurePrefix, err := git.GetConfigInDir(currentDir, "gitflow.prefix.feature")
	if err == nil && featurePrefix != "" {
		return true
	}

	return false
}

// ImportGitFlowAVHConfig imports git-flow-avh configuration
func ImportGitFlowAVHConfig() (*Config, error) {
	config := DefaultConfig()

	// Get current directory for git operations
	currentDir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current directory: %w", err)
	}

	// Check for custom remote in git-flow-avh config
	remote, err := git.GetConfigInDir(currentDir, "gitflow.origin")
	if err == nil && remote != "" {
		config.Remote = remote
	}

	// Map of git-flow-avh config keys to our branch names
	branchMap := map[string]string{
		"master":  "main",
		"develop": "develop",
	}

	// Get branch names from git-flow-avh config
	for avhName, ourName := range branchMap {
		branchName, err := git.GetConfigInDir(currentDir, "gitflow.branch."+avhName)
		if err == nil && branchName != "" {
			// Update branch name in our config
			branchConfig := config.Branches[ourName]
			delete(config.Branches, ourName)
			config.Branches[branchName] = branchConfig

			// Update parent references
			for name, branch := range config.Branches {
				if branch.Parent == ourName {
					branch.Parent = branchName
					config.Branches[name] = branch
				}
				if branch.StartPoint == ourName {
					branch.StartPoint = branchName
					config.Branches[name] = branch
				}
			}
		}
	}

	// Get prefixes from git-flow-avh config
	prefixMap := map[string]string{
		"feature":    "feature",
		"release":    "release",
		"hotfix":     "hotfix",
		"support":    "support",
		"versiontag": "release", // Map versiontag to release branch config
	}

	for avhName, ourName := range prefixMap {
		if avhName == "versiontag" {
			// Special handling for version tag prefix
			prefix, err := git.GetConfigInDir(currentDir, "gitflow.prefix."+avhName)
			if err == nil && prefix != "" {
				// Set the tag prefix for release and hotfix branches
				releaseConfig := config.Branches["release"]
				releaseConfig.TagPrefix = prefix
				releaseConfig.Tag = true // Enable tagging for releases
				config.Branches["release"] = releaseConfig

				hotfixConfig := config.Branches["hotfix"]
				hotfixConfig.TagPrefix = prefix
				hotfixConfig.Tag = true // Enable tagging for hotfixes
				config.Branches["hotfix"] = hotfixConfig
			}
			continue
		}

		if ourName == "" {
			continue
		}

		prefix, err := git.GetConfigInDir(currentDir, "gitflow.prefix."+avhName)
		if err == nil && prefix != "" {
			// Update prefix in our config
			branchConfig := config.Branches[ourName]
			branchConfig.Prefix = prefix
			config.Branches[ourName] = branchConfig
		}
	}

	return config, nil
}

// ApplyOverrides applies the given overrides to the configuration.
// The overrides specify custom branch names and prefixes to use.
func ApplyOverrides(cfg *Config, overrides ConfigOverrides) *Config {
	// Handle main branch override
	if overrides.MainBranch != "" {
		mainConfig := cfg.Branches["main"]
		delete(cfg.Branches, "main")
		cfg.Branches[overrides.MainBranch] = mainConfig

		// Update all branches that reference main
		for name, branch := range cfg.Branches {
			if branch.Parent == "main" {
				branch.Parent = overrides.MainBranch
				cfg.Branches[name] = branch
			}
			if branch.StartPoint == "main" {
				branch.StartPoint = overrides.MainBranch
				cfg.Branches[name] = branch
			}
		}
	}

	// Handle develop branch override
	if overrides.DevelopBranch != "" {
		developConfig := cfg.Branches["develop"]
		delete(cfg.Branches, "develop")
		cfg.Branches[overrides.DevelopBranch] = developConfig

		// Update develop branch's parent reference
		if overrides.MainBranch != "" {
			developConfig.Parent = overrides.MainBranch
		}
		cfg.Branches[overrides.DevelopBranch] = developConfig

		// Update all branches that reference develop
		for name, branch := range cfg.Branches {
			if branch.Parent == "develop" {
				branch.Parent = overrides.DevelopBranch
				cfg.Branches[name] = branch
			}
			if branch.StartPoint == "develop" {
				branch.StartPoint = overrides.DevelopBranch
				cfg.Branches[name] = branch
			}
		}
	} else if overrides.MainBranch != "" {
		// If only main was overridden, update develop's parent
		developConfig := cfg.Branches["develop"]
		developConfig.Parent = overrides.MainBranch
		cfg.Branches["develop"] = developConfig
	}

	// Handle branch prefix overrides
	if overrides.FeaturePrefix != "" {
		featureConfig := cfg.Branches["feature"]
		featureConfig.Prefix = overrides.FeaturePrefix
		cfg.Branches["feature"] = featureConfig
	}

	if overrides.BugfixPrefix != "" {
		bugfixConfig := cfg.Branches["bugfix"]
		bugfixConfig.Prefix = overrides.BugfixPrefix
		cfg.Branches["bugfix"] = bugfixConfig
	}

	if overrides.ReleasePrefix != "" {
		releaseConfig := cfg.Branches["release"]
		releaseConfig.Prefix = overrides.ReleasePrefix
		cfg.Branches["release"] = releaseConfig
	}

	if overrides.HotfixPrefix != "" {
		hotfixConfig := cfg.Branches["hotfix"]
		hotfixConfig.Prefix = overrides.HotfixPrefix
		cfg.Branches["hotfix"] = hotfixConfig
	}

	if overrides.SupportPrefix != "" {
		supportConfig := cfg.Branches["support"]
		supportConfig.Prefix = overrides.SupportPrefix
		cfg.Branches["support"] = supportConfig
	}

	// Handle tag prefix override
	if overrides.TagPrefix != "" {
		releaseConfig := cfg.Branches["release"]
		releaseConfig.TagPrefix = overrides.TagPrefix
		releaseConfig.Tag = true
		cfg.Branches["release"] = releaseConfig

		hotfixConfig := cfg.Branches["hotfix"]
		hotfixConfig.TagPrefix = overrides.TagPrefix
		hotfixConfig.Tag = true
		cfg.Branches["hotfix"] = hotfixConfig
	}

	return cfg
}

//
// Writing and saving functions
//

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
