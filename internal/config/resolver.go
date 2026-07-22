package config

import "fmt"

// ResolvedFinishOptions contains all resolved configuration options for the finish command
type ResolvedFinishOptions struct {
	// Tag options
	ShouldTag   bool
	TagName     string
	ShouldSign  bool
	SigningKey  string
	TagMessage  string
	MessageFile string

	// Branch retention options
	Keep        bool
	KeepRemote  bool
	KeepLocal   bool
	ForceDelete bool

	// Merge strategy options
	MergeStrategy  string // Final resolved strategy (merge/rebase/squash)
	UseRebase      bool   // Whether to use rebase
	PreserveMerges bool   // Whether to preserve merges during rebase
	NoFastForward  bool   // Whether to create merge commit for fast-forward
	UseSquash      bool   // Whether to squash commits
	SquashMessage  string // Custom commit message for squash merge

	// Fetch options
	ShouldFetch bool // Whether to fetch from remote before finishing

	// Custom merge commit messages
	MergeMessage  string // Custom commit message for upstream merge
	UpdateMessage string // Custom commit message for child updates

	// Hook options
	NoVerify bool // Whether to skip pre-commit and commit-msg hooks
}

// TagOptions represents command-line tag options
// Note: This should match the TagOptions type in cmd package
type TagOptions struct {
	ShouldTag   *bool
	ShouldSign  *bool
	SigningKey  string
	Message     string
	MessageFile string
	TagName     string
}

// BranchRetentionOptions represents command-line retention options
// Note: This should match the BranchRetentionOptions type in cmd package
type BranchRetentionOptions struct {
	Keep        *bool
	KeepRemote  *bool
	KeepLocal   *bool
	ForceDelete *bool
}

// MergeStrategyOptions represents command-line merge strategy options
// Note: This should match the MergeStrategyOptions type in cmd package
type MergeStrategyOptions struct {
	Strategy       *string // Override for entire strategy
	Rebase         *bool   // --rebase/--no-rebase override
	PreserveMerges *bool   // --preserve-merges/--no-preserve-merges
	NoFF           *bool   // --no-ff/--ff
	Squash         *bool   // --squash/--no-squash override
	SquashMessage  *string // --squash-message custom commit message
	MergeMessage   *string // --merge-message custom commit message for upstream merge
	UpdateMessage  *string // --update-message custom commit message for child updates
}

// ResolveFinishOptions resolves all finish command options using three-layer precedence:
// Layer 1: Branch configuration defaults
// Layer 2: Command-specific git config (gitflow.<branchtype>.finish.*)
// Layer 3: Command-line arguments (highest priority)
func ResolveFinishOptions(cfg *Config, branchType string, branchName string, tagOpts *TagOptions, retentionOpts *BranchRetentionOptions, mergeOpts *MergeStrategyOptions, fetch *bool, noVerify *bool) *ResolvedFinishOptions {
	branchConfig := cfg.Branches[branchType]

	// Compute full branch name from prefix + branchName
	fullBranchName := branchConfig.Prefix + branchName

	// Resolve merge strategy components
	strategy, useRebase, preserveMerges, noFastForward, useSquash := resolveMergeStrategy(cfg, branchConfig, branchType, mergeOpts)

	return &ResolvedFinishOptions{
		// Tag resolution
		ShouldTag:   resolveFinishShouldTag(cfg, branchConfig, branchType, tagOpts),
		TagName:     resolveFinishTagName(branchConfig, branchType, branchName, tagOpts),
		ShouldSign:  resolveFinishShouldSign(cfg, branchType, tagOpts),
		SigningKey:  resolveFinishSigningKey(cfg, branchType, tagOpts),
		TagMessage:  resolveFinishTagMessage(branchName, tagOpts),
		MessageFile: resolveFinishMessageFile(cfg, branchType, tagOpts),

		// Retention resolution
		Keep:        resolveFinishKeep(cfg, branchType, retentionOpts),
		KeepRemote:  resolveFinishKeepRemote(cfg, branchType, retentionOpts),
		KeepLocal:   resolveFinishKeepLocal(cfg, branchType, retentionOpts),
		ForceDelete: resolveFinishForceDelete(cfg, branchType, retentionOpts),

		// Merge strategy resolution
		MergeStrategy:  strategy,
		UseRebase:      useRebase,
		PreserveMerges: preserveMerges,
		NoFastForward:  noFastForward,
		UseSquash:      useSquash,
		SquashMessage:  resolveSquashMessage(fullBranchName, mergeOpts),

		// Fetch resolution
		ShouldFetch: resolveFinishShouldFetch(cfg, branchType, fetch),

		// Merge commit message resolution
		MergeMessage:  resolveMergeMessage(cfg, branchType, fullBranchName, branchConfig.Parent, mergeOpts),
		UpdateMessage: resolveUpdateMessage(cfg, branchType, mergeOpts),

		// Hook resolution
		NoVerify: resolveFinishNoVerify(cfg, branchType, noVerify),
	}
}

// resolveFinishShouldTag resolves whether to create a tag
func resolveFinishShouldTag(cfg *Config, branchConfig BranchConfig, branchType string, tagOpts *TagOptions) bool {
	// Layer 1: Branch configuration default
	shouldTag := branchConfig.Tag

	// Layer 2: Command-specific config (notag inverts the default)
	if notag := getCommandConfigBool(cfg, fmt.Sprintf("gitflow.%s.finish.notag", branchType)); notag {
		shouldTag = false
	}

	// Layer 3: Command-line flags override config
	if tagOpts != nil && tagOpts.ShouldTag != nil {
		shouldTag = *tagOpts.ShouldTag
	}

	return shouldTag
}

// resolveFinishTagName resolves the tag name to use
func resolveFinishTagName(branchConfig BranchConfig, branchType string, branchName string, tagOpts *TagOptions) string {
	// Layer 1: Default is branch name with prefix from branch config
	tagName := branchName
	if branchConfig.TagPrefix != "" {
		tagName = branchConfig.TagPrefix + branchName
	}

	// Layer 2: No command-specific config for tag name

	// Layer 3: Command-line custom tag name overrides default
	if tagOpts != nil && tagOpts.TagName != "" {
		tagName = tagOpts.TagName
	}

	return tagName
}

// resolveFinishShouldSign resolves whether to sign the tag
func resolveFinishShouldSign(cfg *Config, branchType string, tagOpts *TagOptions) bool {
	// Layer 1: Default is not signing
	shouldSign := false

	// Layer 2: Check command-specific signing config
	if sign := getCommandConfigBool(cfg, fmt.Sprintf("gitflow.%s.finish.sign", branchType)); sign {
		shouldSign = true
	}

	// Layer 3: Command-line signing flags override config
	if tagOpts != nil && tagOpts.ShouldSign != nil {
		shouldSign = *tagOpts.ShouldSign
	}

	return shouldSign
}

// resolveFinishSigningKey resolves the signing key to use
func resolveFinishSigningKey(cfg *Config, branchType string, tagOpts *TagOptions) string {
	// Layer 1: Default is empty
	signingKey := ""

	// Layer 2: Check command-specific signing key
	if key := getCommandConfigString(cfg, fmt.Sprintf("gitflow.%s.finish.signingkey", branchType)); key != "" {
		signingKey = key
	}

	// Layer 3: Command-line signing key overrides config
	if tagOpts != nil && tagOpts.SigningKey != "" {
		signingKey = tagOpts.SigningKey
	}

	return signingKey
}

// resolveFinishTagMessage resolves the tag message
func resolveFinishTagMessage(branchName string, tagOpts *TagOptions) string {
	// Layer 1: Default message
	message := fmt.Sprintf("Tagging version %s", branchName)

	// Layer 2: No command-specific config for tag message

	// Layer 3: Command-line message overrides default
	if tagOpts != nil && tagOpts.Message != "" {
		message = tagOpts.Message
	}

	return message
}

// resolveFinishMessageFile resolves the message file path
func resolveFinishMessageFile(cfg *Config, branchType string, tagOpts *TagOptions) string {
	// Layer 1: Default is empty
	messageFile := ""

	// Layer 2: Check command-specific message file config
	if file := getCommandConfigString(cfg, fmt.Sprintf("gitflow.%s.finish.messagefile", branchType)); file != "" {
		messageFile = file
	}

	// Layer 3: Command-line message file overrides config
	if tagOpts != nil && tagOpts.MessageFile != "" {
		messageFile = tagOpts.MessageFile
	}

	return messageFile
}

// resolveFinishKeep resolves whether to keep the branch
func resolveFinishKeep(cfg *Config, branchType string, retentionOpts *BranchRetentionOptions) bool {
	// Layer 1: Default is to delete (not keep)
	keep := false

	// Layer 2: Check command-specific config
	if keepConfig := getCommandConfigBool(cfg, fmt.Sprintf("gitflow.%s.finish.keep", branchType)); keepConfig {
		keep = true
	}

	// Layer 3: Command-line flags override config
	if retentionOpts != nil && retentionOpts.Keep != nil {
		keep = *retentionOpts.Keep
	}

	return keep
}

// resolveFinishKeepRemote resolves whether to keep the remote branch
func resolveFinishKeepRemote(cfg *Config, branchType string, retentionOpts *BranchRetentionOptions) bool {
	// Layer 1: Default is to delete remote
	keepRemote := false

	// Layer 2: Check command-specific config
	if keepRemoteConfig := getCommandConfigBool(cfg, fmt.Sprintf("gitflow.%s.finish.keepremote", branchType)); keepRemoteConfig {
		keepRemote = true
	}

	// Layer 3: Command-line flags override config
	if retentionOpts != nil && retentionOpts.KeepRemote != nil {
		keepRemote = *retentionOpts.KeepRemote
	}

	return keepRemote
}

// resolveFinishKeepLocal resolves whether to keep the local branch
func resolveFinishKeepLocal(cfg *Config, branchType string, retentionOpts *BranchRetentionOptions) bool {
	// Layer 1: Default is to delete local
	keepLocal := false

	// Layer 2: Check command-specific config
	if keepLocalConfig := getCommandConfigBool(cfg, fmt.Sprintf("gitflow.%s.finish.keeplocal", branchType)); keepLocalConfig {
		keepLocal = true
	}

	// Layer 3: Command-line flags override config
	if retentionOpts != nil && retentionOpts.KeepLocal != nil {
		keepLocal = *retentionOpts.KeepLocal
	}

	return keepLocal
}

// resolveFinishForceDelete resolves whether to force delete the branch
func resolveFinishForceDelete(cfg *Config, branchType string, retentionOpts *BranchRetentionOptions) bool {
	// Layer 1: Default is not to force delete
	forceDelete := false

	// Layer 2: Check command-specific config
	if forceDeleteConfig := getCommandConfigBool(cfg, fmt.Sprintf("gitflow.%s.finish.force-delete", branchType)); forceDeleteConfig {
		forceDelete = true
	}

	// Layer 3: Command-line flags override config
	if retentionOpts != nil && retentionOpts.ForceDelete != nil {
		forceDelete = *retentionOpts.ForceDelete
	}

	return forceDelete
}

// getCommandConfigBool gets a boolean config value from preloaded config
func getCommandConfigBool(cfg *Config, configKey string) bool {
	value, exists := cfg.CommandConfig[configKey]
	return exists && value == "true"
}

// getCommandConfigString gets a string config value from preloaded config
func getCommandConfigString(cfg *Config, configKey string) string {
	value, exists := cfg.CommandConfig[configKey]
	if !exists {
		return ""
	}
	return value
}

// resolveMergeStrategy resolves merge strategy using three-layer precedence
func resolveMergeStrategy(cfg *Config, branchConfig BranchConfig, branchType string, mergeOpts *MergeStrategyOptions) (string, bool, bool, bool, bool) {
	// Layer 1: Get base strategy from branch configuration
	baseStrategy := branchConfig.UpstreamStrategy
	if baseStrategy == "" {
		baseStrategy = "merge" // Default fallback
	}

	// Layer 2: Check for command-specific overrides
	rebase := resolveFinishRebase(cfg, branchType, baseStrategy, mergeOpts)
	squash := resolveFinishSquash(cfg, branchType, baseStrategy, mergeOpts)
	preserveMerges := resolveFinishPreserveMerges(cfg, branchType, mergeOpts)
	noFF := resolveFinishNoFF(cfg, branchType, mergeOpts)

	// Determine final strategy based on precedence: squash > rebase > base strategy
	var finalStrategy string
	if squash {
		finalStrategy = "squash"
	} else if rebase {
		finalStrategy = "rebase"
	} else {
		finalStrategy = "merge"
	}

	return finalStrategy, rebase, preserveMerges, noFF, squash
}

// resolveFinishRebase resolves whether to use rebase strategy
func resolveFinishRebase(cfg *Config, branchType string, baseStrategy string, mergeOpts *MergeStrategyOptions) bool {
	// Layer 1: Base strategy determines default
	useRebase := baseStrategy == "rebase"

	// Layer 2: Command-specific config
	if rebaseConfig := getCommandConfigBool(cfg, fmt.Sprintf("gitflow.%s.finish.rebase", branchType)); rebaseConfig {
		useRebase = true
	}
	// Check for explicit no-rebase config
	if noRebaseConfig := getCommandConfigBool(cfg, fmt.Sprintf("gitflow.%s.finish.no-rebase", branchType)); noRebaseConfig {
		useRebase = false
	}

	// Layer 3: Command-line flags override config
	if mergeOpts != nil {
		if mergeOpts.Rebase != nil {
			useRebase = *mergeOpts.Rebase
		}
		// Strategy override takes precedence
		if mergeOpts.Strategy != nil {
			useRebase = *mergeOpts.Strategy == "rebase"
		}
	}

	return useRebase
}

// resolveFinishSquash resolves whether to use squash strategy
func resolveFinishSquash(cfg *Config, branchType string, baseStrategy string, mergeOpts *MergeStrategyOptions) bool {
	// Layer 1: Base strategy determines default
	useSquash := baseStrategy == "squash"

	// Layer 2: Command-specific config
	if squashConfig := getCommandConfigBool(cfg, fmt.Sprintf("gitflow.%s.finish.squash", branchType)); squashConfig {
		useSquash = true
	}
	// Check for explicit no-squash config
	if noSquashConfig := getCommandConfigBool(cfg, fmt.Sprintf("gitflow.%s.finish.no-squash", branchType)); noSquashConfig {
		useSquash = false
	}

	// Layer 3: Command-line flags override config
	if mergeOpts != nil {
		if mergeOpts.Squash != nil {
			useSquash = *mergeOpts.Squash
		}
		// Strategy override takes precedence
		if mergeOpts.Strategy != nil {
			useSquash = *mergeOpts.Strategy == "squash"
		}
	}

	return useSquash
}

// resolveFinishPreserveMerges resolves whether to preserve merges during rebase
func resolveFinishPreserveMerges(cfg *Config, branchType string, mergeOpts *MergeStrategyOptions) bool {
	// Layer 1: Default is not to preserve merges (flatten)
	preserveMerges := false

	// Layer 2: Command-specific config
	if preserveConfig := getCommandConfigBool(cfg, fmt.Sprintf("gitflow.%s.finish.preserve-merges", branchType)); preserveConfig {
		preserveMerges = true
	}
	// Check for explicit no-preserve-merges config
	if noPreserveConfig := getCommandConfigBool(cfg, fmt.Sprintf("gitflow.%s.finish.no-preserve-merges", branchType)); noPreserveConfig {
		preserveMerges = false
	}

	// Layer 3: Command-line flags override config
	if mergeOpts != nil && mergeOpts.PreserveMerges != nil {
		preserveMerges = *mergeOpts.PreserveMerges
	}

	return preserveMerges
}

// resolveFinishNoFF resolves whether to create merge commit for fast-forward
func resolveFinishNoFF(cfg *Config, branchType string, mergeOpts *MergeStrategyOptions) bool {
	// Layer 1: Default is to allow fast-forward (no --no-ff)
	noFF := false

	// Layer 2: Command-specific config
	if noFFConfig := getCommandConfigBool(cfg, fmt.Sprintf("gitflow.%s.finish.no-ff", branchType)); noFFConfig {
		noFF = true
	}
	// Check for explicit fast-forward config
	if ffConfig := getCommandConfigBool(cfg, fmt.Sprintf("gitflow.%s.finish.ff", branchType)); ffConfig {
		noFF = false
	}

	// Layer 3: Command-line flags override config
	if mergeOpts != nil && mergeOpts.NoFF != nil {
		noFF = *mergeOpts.NoFF
	}

	return noFF
}

// resolveShouldFetch resolves whether to fetch from the remote before a command runs, using the
// standard precedence: Layer-1 default (per command) -> gitflow.<type>.<cmd>.fetch config ->
// CLI flag. Shared by finish (default true) and start (default false).
func resolveShouldFetch(cfg *Config, branchType, cmd string, defaultFetch bool, fetch *bool) bool {
	// Layer 1: Command default
	shouldFetch := defaultFetch

	// Layer 2: Command-specific config (can set true OR false)
	configKey := fmt.Sprintf("gitflow.%s.%s.fetch", branchType, cmd)
	if value, exists := cfg.CommandConfig[configKey]; exists {
		shouldFetch = value == "true"
	}

	// Layer 3: Command-line flags override config
	if fetch != nil {
		shouldFetch = *fetch
	}

	return shouldFetch
}

// resolveFinishShouldFetch resolves whether to fetch from remote before finishing.
// The default is true (ensures the sync check runs against accurate data).
func resolveFinishShouldFetch(cfg *Config, branchType string, fetch *bool) bool {
	return resolveShouldFetch(cfg, branchType, "finish", true, fetch)
}

// ResolveStartShouldFetch resolves whether to fetch from remote before starting a branch.
// The default is false (start does not fetch unless explicitly enabled).
func ResolveStartShouldFetch(cfg *Config, branchType string, fetch *bool) bool {
	return resolveShouldFetch(cfg, branchType, "start", false, fetch)
}

// resolveFinishNoVerify resolves whether to skip pre-commit and commit-msg hooks
func resolveFinishNoVerify(cfg *Config, branchType string, noVerify *bool) bool {
	// Layer 1: Default is to run hooks (no-verify = false)
	skipVerify := false

	// Layer 2: Check command-specific config
	// Note: Git config keys are stored lowercase, so we use "noverify" not "noVerify"
	configKey := fmt.Sprintf("gitflow.%s.finish.noverify", branchType)
	if value, exists := cfg.CommandConfig[configKey]; exists {
		skipVerify = value == "true"
	}

	// Layer 3: Command-line flags override config
	if noVerify != nil {
		skipVerify = *noVerify
	}

	return skipVerify
}

// resolveSquashMessage resolves the squash commit message.
// Unlike most finish options, this is CLI-only (no git config support) because
// squash messages are specific to each branch being finished.
func resolveSquashMessage(fullBranchName string, mergeOpts *MergeStrategyOptions) string {
	// Command-line flag overrides default
	if mergeOpts != nil && mergeOpts.SquashMessage != nil && *mergeOpts.SquashMessage != "" {
		return *mergeOpts.SquashMessage
	}

	// Default message
	return fmt.Sprintf("Squashed commit of branch '%s'", fullBranchName)
}

// resolveMergeMessage resolves the merge commit message.
// Layer 2: gitflow.<branchtype>.finish.mergemessage
// Layer 3: --merge-message flag (highest priority)
func resolveMergeMessage(cfg *Config, branchType string, fullBranchName string, parentBranch string, mergeOpts *MergeStrategyOptions) string {
	// Layer 3: CLI flag overrides all
	if mergeOpts != nil && mergeOpts.MergeMessage != nil && *mergeOpts.MergeMessage != "" {
		return *mergeOpts.MergeMessage
	}

	// Layer 2: Command-specific config
	if msg := getCommandConfigString(cfg, fmt.Sprintf("gitflow.%s.finish.mergemessage", branchType)); msg != "" {
		return msg
	}

	// Empty string signals to use Git's default merge message
	return ""
}

// resolveUpdateMessage resolves the update commit message for child branch updates.
// Layer 2: gitflow.<branchtype>.finish.updateMessage
// Layer 3: --update-message flag (highest priority)
func resolveUpdateMessage(cfg *Config, branchType string, mergeOpts *MergeStrategyOptions) string {
	// Layer 3: CLI flag overrides all
	if mergeOpts != nil && mergeOpts.UpdateMessage != nil && *mergeOpts.UpdateMessage != "" {
		return *mergeOpts.UpdateMessage
	}

	// Layer 2: Command-specific config
	if msg := getCommandConfigString(cfg, fmt.Sprintf("gitflow.%s.finish.updatemessage", branchType)); msg != "" {
		return msg
	}

	// Empty string signals to use the default auto-generated message
	return ""
}
