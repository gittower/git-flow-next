// Package config provides configuration handling for git-flow
package config

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gittower/git-flow-next/internal/git"
)

//
// Types and constants
//

// Config represents the git-flow configuration
type Config struct {
	Version       string
	Branches      map[string]BranchConfig
	Remote        string            // Name of the remote to use for all operations
	CommandConfig map[string]string // All gitflow.* command-specific config (Layer 2)
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
	Worktree           bool   // whether 'start' creates a worktree for this branch type by default
}

// ResolveBranchName resolves a branch reference to its canonical (stored) name
// by matching case-insensitively against the configured branch identities.
// It returns the canonical name and true on a match, or ("", false) if no
// configured branch folds to name. An exact-case match is preferred; otherwise
// the first case-insensitive match is returned.
func (c *Config) ResolveBranchName(name string) (canonical string, found bool) {
	if _, ok := c.Branches[name]; ok {
		return name, true
	}
	for branchName := range c.Branches {
		if strings.EqualFold(branchName, name) {
			return branchName, true
		}
	}
	return "", false
}

// TrunkBranch returns the configuration's trunk branch — the base branch with no
// parent. When several base branches have no parent (no shipped preset produces
// that) the lexicographically smallest name wins, so the result never depends on
// Go's randomized map iteration order. It returns "" for a configuration with no
// trunk.
func (c *Config) TrunkBranch() string {
	trunk := ""
	for name, branch := range c.Branches {
		if branch.Type != string(BranchTypeBase) || branch.Parent != "" {
			continue
		}
		if trunk == "" || name < trunk {
			trunk = name
		}
	}
	return trunk
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
	MainBranch       string // Name of the main branch (trunk branch)
	DevelopBranch    string // Name of the develop branch
	ProductionBranch string // Name of the production branch (for GitLab flow)
	StagingBranch    string // Name of the staging branch (for GitLab flow)
	FeaturePrefix    string // Prefix for feature branches
	BugfixPrefix     string // Prefix for bugfix branches
	ReleasePrefix    string // Prefix for release branches
	HotfixPrefix     string // Prefix for hotfix branches
	SupportPrefix    string // Prefix for support branches
	TagPrefix        string // Prefix for tags
}

//
// Loading and initialization functions
//

// DefaultConfig returns a default git-flow configuration
func DefaultConfig() *Config {
	return &Config{
		Version:       "1.0",
		Remote:        "origin",                // Default remote name
		CommandConfig: make(map[string]string), // Initialize command config map
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
				Tag:                false, // Feature branches typically don't create tags by default
				TagPrefix:          "",    // No default tag prefix
			},
			"bugfix": {
				Type:               string(BranchTypeTopic),
				Parent:             "develop",
				StartPoint:         "develop",
				UpstreamStrategy:   string(MergeStrategyMerge),
				DownstreamStrategy: string(MergeStrategyRebase),
				Prefix:             "bugfix/",
				Tag:                false, // Bugfix branches typically don't create tags by default
				TagPrefix:          "",    // No default tag prefix
			},
			"release": {
				Type:               string(BranchTypeTopic),
				Parent:             "main",
				StartPoint:         "develop",
				UpstreamStrategy:   string(MergeStrategyMerge),
				DownstreamStrategy: string(MergeStrategyMerge),
				Prefix:             "release/",
				Tag:                true, // Enable tagging by default
				TagPrefix:          "",   // No default prefix, will be set during init
			},
			"hotfix": {
				Type:               string(BranchTypeTopic),
				Parent:             "main",
				StartPoint:         "main",
				UpstreamStrategy:   string(MergeStrategyMerge),
				DownstreamStrategy: string(MergeStrategyRebase),
				Prefix:             "hotfix/",
				Tag:                true, // Enable tagging by default
				TagPrefix:          "",   // No default prefix, will be set during init
			},
			"support": {
				Type:               string(BranchTypeTopic),
				Parent:             "main",
				StartPoint:         "main",
				UpstreamStrategy:   string(MergeStrategyNone),
				DownstreamStrategy: string(MergeStrategyNone),
				Prefix:             "support/",
				Tag:                false, // Support branches typically don't create tags by default
				TagPrefix:          "",    // No default tag prefix
			},
		},
	}
}

// Load loads the git-flow configuration from the given repository's Git config.
func Load(repo *git.Repo) (*Config, error) {
	// Check if git-flow is initialized
	initialized, err := IsInitialized(repo)
	if err != nil {
		return nil, fmt.Errorf("failed to check if git-flow is initialized: %w", err)
	}

	if !initialized {
		// If not initialized, return default config
		return DefaultConfig(), nil
	}

	// Get git-flow version
	version, err := repo.GetConfig("gitflow.version")
	if err != nil {
		// If no version is set but AVH config exists, import AVH config
		if CheckGitFlowAVHConfig(repo) {
			return ImportGitFlowAVHConfig(repo)
		}
		// If no version is set, assume it's not initialized properly
		return DefaultConfig(), nil
	}

	// Create config with version
	config := &Config{
		Version:       version,
		Remote:        "origin", // Default remote
		CommandConfig: make(map[string]string),
		Branches:      make(map[string]BranchConfig),
	}

	// Load all gitflow.* config at once
	allGitflowConfig, err := loadAllGitflowConfig(repo)
	if err != nil {
		return nil, fmt.Errorf("failed to load gitflow config: %w", err)
	}
	config.CommandConfig = allGitflowConfig

	// Get custom remote name if set
	remote, ok := allGitflowConfig["gitflow.origin"]
	if ok && remote != "" {
		config.Remote = remote
	}

	// Get all gitflow.branch.* config entries
	branchLines, err := repo.GetConfigRegexpLines("gitflow\\.branch\\.")
	if err != nil {
		return nil, fmt.Errorf("failed to load gitflow branch config: %w", err)
	}

	config.Branches = ParseBranchConfigLines(branchLines)

	// If no branches were loaded, use default config
	if len(config.Branches) == 0 {
		return DefaultConfig(), nil
	}

	return config, nil
}

// ParseBranchConfigLines converts raw `git config --get-regexp gitflow.branch.`
// output lines into BranchConfig objects keyed by canonical branch name.
//
// Branch subsection names (gitflow.branch.<name>) are treated case-insensitively
// for identity but their original case is preserved as canonical (mirrors
// core.ignorecase semantics). The first-seen casing of a branch name wins as the
// canonical key; later properties of the same branch fold into that entry. The
// name itself may contain dots (e.g. gitflow.branch.custom.main.type), so it is
// reconstructed from all segments between "gitflow.branch." and the final one.
// Property names (the last segment) are legitimately case-insensitive in git
// config and are lowercased so the BranchConfig field lookups match regardless
// of stored case. Lines that are not branch *definitions* (e.g. the runtime
// gitflow.branch.<actual>.base key) are still parsed as properties but yield no
// recognized BranchConfig field.
func ParseBranchConfigLines(branchLines []string) map[string]BranchConfig {
	branchMap := make(map[string]map[string]string)
	// canonicalNames maps a lowercased fold key to the canonical (first-seen)
	// branch name used as the branchMap key.
	canonicalNames := make(map[string]string)

	for _, line := range branchLines {
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

		rawBranchName := strings.Join(keyParts[2:len(keyParts)-1], ".")
		foldKey := strings.ToLower(rawBranchName)
		branchName, ok := canonicalNames[foldKey]
		if !ok {
			branchName = rawBranchName
			canonicalNames[foldKey] = branchName
		}
		property := strings.ToLower(keyParts[len(keyParts)-1])

		if _, ok := branchMap[branchName]; !ok {
			branchMap[branchName] = make(map[string]string)
		}
		branchMap[branchName][property] = value
	}

	branches := make(map[string]BranchConfig)
	for branchName, properties := range branchMap {
		branchConfig := BranchConfig{
			Type:               properties["type"],
			Parent:             properties["parent"],
			StartPoint:         properties["startpoint"],
			UpstreamStrategy:   properties["upstreamstrategy"],
			DownstreamStrategy: properties["downstreamstrategy"],
			Prefix:             properties["prefix"],
		}

		if autoUpdate, ok := properties["autoupdate"]; ok {
			branchConfig.AutoUpdate = ParseBool(autoUpdate)
		}
		if tag, ok := properties["tag"]; ok {
			branchConfig.Tag = ParseBool(tag)
		}
		if tagPrefix, ok := properties["tagprefix"]; ok {
			branchConfig.TagPrefix = tagPrefix
		}
		if worktreeValue, ok := properties["worktree"]; ok {
			branchConfig.Worktree = ParseBool(worktreeValue)
		}

		branches[branchName] = branchConfig
	}

	return branches
}

// IsInitialized checks if git-flow is initialized in the repository
// This includes both git-flow-next and git-flow-avh configurations
func IsInitialized(repo *git.Repo) (bool, error) {
	// Check for our own gitflow.version config
	version, err := repo.GetConfig("gitflow.version")
	if err == nil && version != "" {
		return true, nil
	}

	// Check for git-flow-avh configuration
	if CheckGitFlowAVHConfig(repo) {
		return true, nil
	}

	return false, nil
}

// IsGitFlowNextInitialized checks if git-flow-next specifically is initialized
// This only checks for our own configuration, not git-flow-avh
func IsGitFlowNextInitialized(repo *git.Repo) (bool, error) {
	// Check for our own gitflow.version config
	version, err := repo.GetConfig("gitflow.version")
	if err == nil && version != "" {
		return true, nil
	}

	return false, nil
}

// InitializedStatus contains information about initialization state and source
type InitializedStatus struct {
	Initialized bool
	SourceScope git.ConfigScope // which scope the config was found in (for messaging)
}

// IsGitFlowNextInitializedWithScope checks if git-flow-next is initialized at a specific scope.
// - ConfigScopeDefault: checks merged config, returns the scope where config was found
// - Specific scope: checks only that scope
func IsGitFlowNextInitializedWithScope(scope git.ConfigScope, filePath string) (InitializedStatus, error) {
	if scope == git.ConfigScopeDefault {
		// Check scopes in order: local > global > system
		// Return the first scope where gitflow.version is found
		for _, checkScope := range []git.ConfigScope{git.ConfigScopeLocal, git.ConfigScopeGlobal, git.ConfigScopeSystem} {
			version, err := git.GetConfigWithScope("gitflow.version", checkScope, "")
			if err == nil && version != "" {
				return InitializedStatus{Initialized: true, SourceScope: checkScope}, nil
			}
		}
		return InitializedStatus{Initialized: false}, nil
	}

	// Check only the specified scope
	version, err := git.GetConfigWithScope("gitflow.version", scope, filePath)
	if err == nil && version != "" {
		return InitializedStatus{Initialized: true, SourceScope: scope}, nil
	}
	return InitializedStatus{Initialized: false}, nil
}

// CheckGitFlowAVHConfig checks if git-flow-avh configuration exists
func CheckGitFlowAVHConfig(repo *git.Repo) bool {
	// Check for gitflow.branch.master (used in git-flow-avh)
	master, err := repo.GetConfig("gitflow.branch.master")
	if err == nil && master != "" {
		return true
	}

	// Check for gitflow.prefix.feature (used in git-flow-avh)
	featurePrefix, err := repo.GetConfig("gitflow.prefix.feature")
	if err == nil && featurePrefix != "" {
		return true
	}

	return false
}

// ImportGitFlowAVHConfig imports git-flow-avh configuration
func ImportGitFlowAVHConfig(repo *git.Repo) (*Config, error) {
	config := DefaultConfig()

	// Load all gitflow.* config at once so that command-specific
	// settings like gitflow.release.finish.push are honoured even
	// when the repo was initialised with git-flow-avh.
	allGitflowConfig, err := loadAllGitflowConfig(repo)
	if err != nil {
		return nil, fmt.Errorf("failed to load gitflow config: %w", err)
	}
	config.CommandConfig = allGitflowConfig

	// Check for custom remote in git-flow-avh config
	remote, ok := allGitflowConfig["gitflow.origin"]
	if ok && remote != "" {
		config.Remote = remote
	}

	// Map of git-flow-avh config keys to our branch names
	branchMap := map[string]string{
		"master":  "main",
		"develop": "develop",
	}

	// Get branch names from git-flow-avh config
	for avhName, ourName := range branchMap {
		branchName, err := repo.GetConfig("gitflow.branch." + avhName)
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
		"bugfix":     "bugfix",
		"release":    "release",
		"hotfix":     "hotfix",
		"support":    "support",
		"versiontag": "release", // Map versiontag to release branch config
	}

	for avhName, ourName := range prefixMap {
		if avhName == "versiontag" {
			// Special handling for version tag prefix
			prefix, err := repo.GetConfig("gitflow.prefix." + avhName)
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

		prefix, err := repo.GetConfig("gitflow.prefix." + avhName)
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
	// Handle production branch override (for GitLab flow)
	if overrides.ProductionBranch != "" {
		if productionConfig, exists := cfg.Branches["production"]; exists {
			delete(cfg.Branches, "production")
			cfg.Branches[overrides.ProductionBranch] = productionConfig

			// Update all branches that reference production
			for name, branch := range cfg.Branches {
				if branch.Parent == "production" {
					branch.Parent = overrides.ProductionBranch
					cfg.Branches[name] = branch
				}
				if branch.StartPoint == "production" {
					branch.StartPoint = overrides.ProductionBranch
					cfg.Branches[name] = branch
				}
			}
		}
	}

	// Handle staging branch override (for GitLab flow)
	if overrides.StagingBranch != "" {
		if stagingConfig, exists := cfg.Branches["staging"]; exists {
			delete(cfg.Branches, "staging")
			cfg.Branches[overrides.StagingBranch] = stagingConfig

			// Update all branches that reference staging
			for name, branch := range cfg.Branches {
				if branch.Parent == "staging" {
					branch.Parent = overrides.StagingBranch
					cfg.Branches[name] = branch
				}
				if branch.StartPoint == "staging" {
					branch.StartPoint = overrides.StagingBranch
					cfg.Branches[name] = branch
				}
			}
		}
	}

	// Handle main branch override
	if overrides.MainBranch != "" {
		if mainConfig, exists := cfg.Branches["main"]; exists {
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

// SaveConfig saves the git-flow configuration to Git config (local scope).
// This is a convenience wrapper around SaveConfigWithScope for callers
// that don't need explicit scope control.
func SaveConfig(config *Config) error {
	return SaveConfigWithScope(config, git.ConfigScopeDefault, "")
}

// MarkRepoInitialized marks the repository as initialized with git-flow (local scope).
// This is a convenience wrapper around MarkRepoInitializedWithScope for callers
// that don't need explicit scope control.
func MarkRepoInitialized() error {
	return MarkRepoInitializedWithScope(git.ConfigScopeDefault, "")
}

// SaveConfigWithScope saves the git-flow configuration to Git config at a specific scope
func SaveConfigWithScope(config *Config, scope git.ConfigScope, filePath string) error {
	// Set git-flow version
	err := git.SetConfigWithScope("gitflow.version", config.Version, scope, filePath)
	if err != nil {
		return fmt.Errorf("failed to set gitflow.version: %w", err)
	}

	// Save branch configurations
	for branchName, branchConfig := range config.Branches {
		// Set branch type
		err = git.SetConfigWithScope(fmt.Sprintf("gitflow.branch.%s.type", branchName), branchConfig.Type, scope, filePath)
		if err != nil {
			return fmt.Errorf("failed to set branch type for %s: %w", branchName, err)
		}

		// Set parent branch if it exists
		if branchConfig.Parent != "" {
			err = git.SetConfigWithScope(fmt.Sprintf("gitflow.branch.%s.parent", branchName), branchConfig.Parent, scope, filePath)
			if err != nil {
				return fmt.Errorf("failed to set parent branch for %s: %w", branchName, err)
			}
		}

		// Set start point if it exists
		if branchConfig.StartPoint != "" {
			err = git.SetConfigWithScope(fmt.Sprintf("gitflow.branch.%s.startPoint", branchName), branchConfig.StartPoint, scope, filePath)
			if err != nil {
				return fmt.Errorf("failed to set start point for %s: %w", branchName, err)
			}
		}

		// Set upstream strategy if it exists
		if branchConfig.UpstreamStrategy != "" {
			err = git.SetConfigWithScope(fmt.Sprintf("gitflow.branch.%s.upstreamStrategy", branchName), branchConfig.UpstreamStrategy, scope, filePath)
			if err != nil {
				return fmt.Errorf("failed to set upstream strategy for %s: %w", branchName, err)
			}
		}

		// Set downstream strategy if it exists
		if branchConfig.DownstreamStrategy != "" {
			err = git.SetConfigWithScope(fmt.Sprintf("gitflow.branch.%s.downstreamStrategy", branchName), branchConfig.DownstreamStrategy, scope, filePath)
			if err != nil {
				return fmt.Errorf("failed to set downstream strategy for %s: %w", branchName, err)
			}
		}

		// Set prefix if it exists
		if branchConfig.Prefix != "" {
			err = git.SetConfigWithScope(fmt.Sprintf("gitflow.branch.%s.prefix", branchName), branchConfig.Prefix, scope, filePath)
			if err != nil {
				return fmt.Errorf("failed to set prefix for %s: %w", branchName, err)
			}
		}

		// Set auto update
		err = git.SetConfigWithScope(fmt.Sprintf("gitflow.branch.%s.autoUpdate", branchName), strconv.FormatBool(branchConfig.AutoUpdate), scope, filePath)
		if err != nil {
			return fmt.Errorf("failed to set auto update for %s: %w", branchName, err)
		}

		// Set tag configuration. Written explicitly as true/false rather than
		// omitted when false: an explicit local false must shadow a true inherited
		// from an outer git config scope, and the writer stays a faithful
		// serializer of the in-memory configuration with no special case.
		err = git.SetConfigWithScope(fmt.Sprintf("gitflow.branch.%s.tag", branchName), strconv.FormatBool(branchConfig.Tag), scope, filePath)
		if err != nil {
			return fmt.Errorf("failed to set tag configuration for %s: %w", branchName, err)
		}

		// Set the worktree default, for TOPIC types only. autoUpdate and tag
		// above are written unconditionally so an explicit local false shadows a
		// true inherited from an outer scope; the same reasoning applies here,
		// but only where the setting means something. A worktree default on a
		// base branch has no effect — start never creates one for a base branch —
		// and writing it would put noise into every committed .gitflow.
		if branchConfig.Type == string(BranchTypeTopic) {
			err = git.SetConfigWithScope(fmt.Sprintf("gitflow.branch.%s.worktree", branchName), strconv.FormatBool(branchConfig.Worktree), scope, filePath)
			if err != nil {
				return fmt.Errorf("failed to set worktree configuration for %s: %w", branchName, err)
			}
		}

		// Set tag prefix if it exists
		if branchConfig.TagPrefix != "" {
			err = git.SetConfigWithScope(fmt.Sprintf("gitflow.branch.%s.tagprefix", branchName), branchConfig.TagPrefix, scope, filePath)
			if err != nil {
				return fmt.Errorf("failed to set tag prefix for %s: %w", branchName, err)
			}
		}
	}

	return nil
}

// MarkRepoInitializedWithScope marks the repository as initialized with git-flow at a specific scope
func MarkRepoInitializedWithScope(scope git.ConfigScope, filePath string) error {
	err := git.SetConfigWithScope("gitflow.initialized", "true", scope, filePath)
	if err != nil {
		return fmt.Errorf("failed to mark repository as initialized: %w", err)
	}
	return nil
}

// ClearConfig removes all git-flow configuration
func ClearConfig(repo *git.Repo) error {
	// Get all gitflow.* config entries
	configs, err := repo.GetAllConfig("gitflow\\.")
	if err != nil {
		return fmt.Errorf("failed to get gitflow configurations: %w", err)
	}

	// Remove each config entry
	for key := range configs {
		if err := repo.UnsetConfig(key); err != nil {
			return fmt.Errorf("failed to unset %s: %w", key, err)
		}
	}

	return nil
}

// PresetType represents the type of workflow preset
type PresetType string

const (
	PresetClassic PresetType = "classic"
	PresetGitHub  PresetType = "github"
	PresetGitLab  PresetType = "gitlab"
)

// PresetConfig returns a preset configuration based on the specified type
func PresetConfig(preset PresetType) *Config {
	switch preset {
	case PresetGitHub:
		return githubFlowConfig()
	case PresetGitLab:
		return gitlabFlowConfig()
	case PresetClassic:
		fallthrough
	default:
		return DefaultConfig()
	}
}

// githubFlowConfig returns a GitHub Flow configuration
func githubFlowConfig() *Config {
	return &Config{
		Version:       "1.0",
		Remote:        "origin",
		CommandConfig: make(map[string]string),
		Branches: map[string]BranchConfig{
			"main": {
				Type:               string(BranchTypeBase),
				Parent:             "",
				UpstreamStrategy:   string(MergeStrategyNone),
				DownstreamStrategy: string(MergeStrategyNone),
				AutoUpdate:         false,
			},
			"feature": {
				Type:               string(BranchTypeTopic),
				Parent:             "main",
				StartPoint:         "main",
				UpstreamStrategy:   string(MergeStrategyMerge),
				DownstreamStrategy: string(MergeStrategyRebase),
				Prefix:             "feature/",
			},
		},
	}
}

// gitlabFlowConfig returns a GitLab Flow configuration
func gitlabFlowConfig() *Config {
	return &Config{
		Version:       "1.0",
		Remote:        "origin",
		CommandConfig: make(map[string]string),
		Branches: map[string]BranchConfig{
			"production": {
				Type:               string(BranchTypeBase),
				Parent:             "",
				UpstreamStrategy:   string(MergeStrategyNone),
				DownstreamStrategy: string(MergeStrategyNone),
				AutoUpdate:         false,
			},
			"staging": {
				Type:               string(BranchTypeBase),
				Parent:             "production",
				UpstreamStrategy:   string(MergeStrategyMerge),
				DownstreamStrategy: string(MergeStrategyMerge),
				AutoUpdate:         false,
			},
			"main": {
				Type:               string(BranchTypeBase),
				Parent:             "staging",
				UpstreamStrategy:   string(MergeStrategyMerge),
				DownstreamStrategy: string(MergeStrategyMerge),
				AutoUpdate:         false,
			},
			"feature": {
				Type:               string(BranchTypeTopic),
				Parent:             "main",
				StartPoint:         "main",
				UpstreamStrategy:   string(MergeStrategyMerge),
				DownstreamStrategy: string(MergeStrategyRebase),
				Prefix:             "feature/",
			},
			"hotfix": {
				Type:               string(BranchTypeTopic),
				Parent:             "production",
				StartPoint:         "production",
				UpstreamStrategy:   string(MergeStrategyMerge),
				DownstreamStrategy: string(MergeStrategyMerge),
				Prefix:             "hotfix/",
			},
		},
	}
}

// loadAllGitflowConfig loads all gitflow.* configuration keys at once
func loadAllGitflowConfig(repo *git.Repo) (map[string]string, error) {
	rawLines, err := repo.GetConfigRegexpLines("gitflow\\.")
	if err != nil {
		return nil, fmt.Errorf("failed to get gitflow config: %w", err)
	}

	result := make(map[string]string)

	// Join and re-trim to mirror the previous strings.TrimSpace(output) behavior.
	lines := strings.Split(strings.TrimSpace(strings.Join(rawLines, "\n")), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, " ", 2)
		if len(parts) == 2 {
			result[parts[0]] = parts[1]
		} else {
			// Handle case where config value is empty
			result[parts[0]] = ""
		}
	}

	return result, nil
}
