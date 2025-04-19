package config

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/gittower/git-flow-next/git"
)

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
