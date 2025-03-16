package config

import (
	"fmt"
	"strings"

	"github.com/gittower/git-flow-next/git"
)

// DefaultConfig returns a default git-flow configuration
func DefaultConfig() *Config {
	return &Config{
		Version: "1.0",
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
				UpstreamStrategy:   string(MergeStrategyRebase),
				DownstreamStrategy: string(MergeStrategySquash),
				Prefix:             "feature/",
			},
			"release": {
				Type:               string(BranchTypeTopic),
				Parent:             "develop",
				StartPoint:         "develop",
				UpstreamStrategy:   string(MergeStrategyMerge),
				DownstreamStrategy: string(MergeStrategyMerge),
				Prefix:             "release/",
			},
			"hotfix": {
				Type:               string(BranchTypeTopic),
				Parent:             "main",
				StartPoint:         "main",
				UpstreamStrategy:   string(MergeStrategyMerge),
				DownstreamStrategy: string(MergeStrategyMerge),
				Prefix:             "hotfix/",
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
	version, err := git.GetConfig("gitflow.version")
	if err != nil {
		// If no version is set, assume it's not initialized properly
		return DefaultConfig(), nil
	}

	// Create config with version
	config := &Config{
		Version:  version,
		Branches: make(map[string]BranchConfig),
	}

	// Get all gitflow.branch.* config entries
	branchConfigs, err := git.GetAllConfig("gitflow\\.branch\\.")
	if err != nil {
		return nil, fmt.Errorf("failed to get branch configurations: %w", err)
	}

	// Process branch configurations
	branchMap := make(map[string]map[string]string)
	for key, value := range branchConfigs {
		// Parse key: gitflow.branch.<branchname>.<property>
		parts := strings.Split(key, ".")
		if len(parts) < 4 {
			continue
		}

		branchName := parts[2]
		property := parts[3]

		// Initialize branch map if needed
		if _, ok := branchMap[branchName]; !ok {
			branchMap[branchName] = make(map[string]string)
		}

		// Add property to branch map
		branchMap[branchName][property] = value
	}

	// Convert branch map to BranchConfig objects
	for branchName, properties := range branchMap {
		branchConfig := BranchConfig{
			Type:               properties["type"],
			Parent:             properties["parent"],
			StartPoint:         properties["startPoint"],
			UpstreamStrategy:   properties["upstreamStrategy"],
			DownstreamStrategy: properties["downstreamStrategy"],
			Prefix:             properties["prefix"],
		}

		// Handle boolean properties
		if autoUpdate, ok := properties["autoUpdate"]; ok {
			branchConfig.AutoUpdate = autoUpdate == "true"
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
	version, err := git.GetConfig("gitflow.version")
	if err != nil {
		// If error is because the key doesn't exist, it's not initialized
		return false, nil
	}
	return version != "", nil
}

// CheckGitFlowAVHConfig checks if git-flow-avh configuration exists
func CheckGitFlowAVHConfig() bool {
	// Check for gitflow.branch.master (used in git-flow-avh)
	master, err := git.GetConfig("gitflow.branch.master")
	if err == nil && master != "" {
		return true
	}

	// Check for gitflow.prefix.feature (used in git-flow-avh)
	featurePrefix, err := git.GetConfig("gitflow.prefix.feature")
	if err == nil && featurePrefix != "" {
		return true
	}

	return false
}

// ImportGitFlowAVHConfig imports git-flow-avh configuration
func ImportGitFlowAVHConfig() (*Config, error) {
	config := DefaultConfig()

	// Map of git-flow-avh config keys to our branch names
	branchMap := map[string]string{
		"master":  "main",
		"develop": "develop",
	}

	// Get branch names from git-flow-avh config
	for avhName, ourName := range branchMap {
		branchName, err := git.GetConfig("gitflow.branch." + avhName)
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
		"versiontag": "", // Not used in our config
	}

	for avhName, ourName := range prefixMap {
		if ourName == "" {
			continue
		}

		prefix, err := git.GetConfig("gitflow.prefix." + avhName)
		if err == nil && prefix != "" {
			// Update prefix in our config
			branchConfig := config.Branches[ourName]
			branchConfig.Prefix = prefix
			config.Branches[ourName] = branchConfig
		}
	}

	return config, nil
}
