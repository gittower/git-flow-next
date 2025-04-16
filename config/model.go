package config

// Config represents the git-flow configuration
type Config struct {
	Version  string
	Branches map[string]BranchConfig
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
