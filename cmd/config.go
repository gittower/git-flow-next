package cmd

import (
	"fmt"
	"os"

	"github.com/gittower/git-flow-next/internal/config"
	"github.com/gittower/git-flow-next/internal/errors"
	"github.com/gittower/git-flow-next/internal/git"
	"github.com/gittower/git-flow-next/internal/util"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage git-flow configuration",
	Long: `Manage git-flow configuration for base branches and topic branch types.

Base branches are long-living branches like main, develop, staging, production.
Topic branches are short-living branches like feature, release, hotfix.

Examples:
  git-flow config add base develop main --auto-update=true
  git-flow config add topic feature develop --prefix=feat/
  git-flow config edit base develop --auto-update=false
  git-flow config rename base develop integration
  git-flow config delete topic support
  git-flow config list`,
}

var configAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add base branch or topic branch type configuration",
	Long: `Add a new base branch or topic branch type to the git-flow configuration.

Base branches are created immediately when added.
Topic branch configurations are saved for use with start command.`,
}

var configAddBaseCmd = &cobra.Command{
	Use:   "base <name> [<parent>]",
	Short: "Add a base branch configuration",
	Long: `Add a base branch to the git-flow configuration.

The branch will be created immediately if it doesn't exist.
If parent is specified, the branch will be created from the parent branch.

Examples:
  git-flow config add base production
  git-flow config add base develop main --auto-update=true
  git-flow config add base staging production --upstream-strategy=merge`,
	Args: cobra.RangeArgs(1, 2),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		parent := ""
		if len(args) > 1 {
			parent = args[1]
		}

		upstreamStrategy, _ := cmd.Flags().GetString("upstream-strategy")
		downstreamStrategy, _ := cmd.Flags().GetString("downstream-strategy")
		autoUpdate, _ := cmd.Flags().GetBool("auto-update")

		ConfigAddBaseCommand(name, parent, upstreamStrategy, downstreamStrategy, autoUpdate)
	},
}

var configAddTopicCmd = &cobra.Command{
	Use:   "topic <name> <parent>",
	Short: "Add a topic branch type configuration",
	Long: `Add a topic branch type to the git-flow configuration.

The configuration is saved immediately and can be used with start command.
Parent branch must be a configured base branch.

Examples:
  git-flow config add topic feature develop --prefix=feat/
  git-flow config add topic release main --starting-point=develop --tag=true
  git-flow config add topic hotfix main --upstream-strategy=squash`,
	Args: cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		parent := args[1]

		prefix, _ := cmd.Flags().GetString("prefix")
		startingPoint, _ := cmd.Flags().GetString("starting-point")
		upstreamStrategy, _ := cmd.Flags().GetString("upstream-strategy")
		downstreamStrategy, _ := cmd.Flags().GetString("downstream-strategy")
		tag, _ := cmd.Flags().GetBool("tag")

		ConfigAddTopicCommand(name, parent, prefix, startingPoint, upstreamStrategy, downstreamStrategy, tag)
	},
}

var configEditCmd = &cobra.Command{
	Use:   "edit",
	Short: "Edit base branch or topic branch type configuration",
	Long:  `Edit an existing base branch or topic branch type configuration.`,
}

var configEditBaseCmd = &cobra.Command{
	Use:   "base <name>",
	Short: "Edit a base branch configuration",
	Long: `Edit an existing base branch configuration.

Examples:
  git-flow config edit base develop --auto-update=false
  git-flow config edit base staging --upstream-strategy=rebase`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]

		upstreamStrategy, _ := cmd.Flags().GetString("upstream-strategy")
		downstreamStrategy, _ := cmd.Flags().GetString("downstream-strategy")
		autoUpdate, _ := cmd.Flags().GetBool("auto-update")

		ConfigEditBaseCommand(name, upstreamStrategy, downstreamStrategy, autoUpdate)
	},
}

var configEditTopicCmd = &cobra.Command{
	Use:   "topic <name>",
	Short: "Edit a topic branch type configuration",
	Long: `Edit an existing topic branch type configuration.

Examples:
  git-flow config edit topic feature --prefix=feat/
  git-flow config edit topic release --tag=true`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]

		prefix, _ := cmd.Flags().GetString("prefix")
		startingPoint, _ := cmd.Flags().GetString("starting-point")
		upstreamStrategy, _ := cmd.Flags().GetString("upstream-strategy")
		downstreamStrategy, _ := cmd.Flags().GetString("downstream-strategy")
		tag, _ := cmd.Flags().GetBool("tag")

		ConfigEditTopicCommand(name, prefix, startingPoint, upstreamStrategy, downstreamStrategy, tag)
	},
}

var configRenameCmd = &cobra.Command{
	Use:   "rename",
	Short: "Rename base branch or topic branch type",
	Long:  `Rename an existing base branch or topic branch type.`,
}

var configRenameBaseCmd = &cobra.Command{
	Use:   "base <old-name> <new-name>",
	Short: "Rename a base branch",
	Long: `Rename a base branch in both configuration and Git.

This will rename the actual Git branch and update all references in the configuration.

Examples:
  git-flow config rename base develop integration
  git-flow config rename base master main`,
	Args: cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		oldName := args[0]
		newName := args[1]

		ConfigRenameBaseCommand(oldName, newName)
	},
}

var configRenameTopicCmd = &cobra.Command{
	Use:   "topic <old-name> <new-name>",
	Short: "Rename a topic branch type",
	Long: `Rename a topic branch type configuration.

This only updates the configuration, not any existing branches.

Examples:
  git-flow config rename topic feature feat
  git-flow config rename topic bugfix fix`,
	Args: cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		oldName := args[0]
		newName := args[1]

		ConfigRenameTopicCommand(oldName, newName)
	},
}

var configDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete base branch or topic branch type configuration",
	Long:  `Delete a base branch or topic branch type configuration.`,
}

var configDeleteBaseCmd = &cobra.Command{
	Use:   "base <name>",
	Short: "Delete a base branch configuration",
	Long: `Delete a base branch configuration.

This removes the configuration but keeps the Git branch.
Cannot delete if other branches depend on this branch.

Examples:
  git-flow config delete base staging
  git-flow config delete base old-main`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]

		ConfigDeleteBaseCommand(name)
	},
}

var configDeleteTopicCmd = &cobra.Command{
	Use:   "topic <name>",
	Short: "Delete a topic branch type configuration",
	Long: `Delete a topic branch type configuration.

This removes the configuration but does not affect existing branches of this type.

Examples:
  git-flow config delete topic support
  git-flow config delete topic bugfix`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]

		ConfigDeleteTopicCommand(name)
	},
}

var configListCmd = &cobra.Command{
	Use:   "list",
	Short: "List current git-flow configuration",
	Long: `List the current git-flow configuration showing all base branches and topic branch types.

Shows the branch hierarchy, merge strategies, and other settings.

Example:
  git-flow config list`,
	Run: func(cmd *cobra.Command, args []string) {
		ConfigListCommand()
	},
}

// ConfigAddBaseCommand adds a base branch configuration
func ConfigAddBaseCommand(name, parent, upstreamStrategy, downstreamStrategy string, autoUpdate bool) {
	if err := executeConfigAddBase(name, parent, upstreamStrategy, downstreamStrategy, autoUpdate); err != nil {
		var exitCode errors.ExitCode
		if flowErr, ok := err.(errors.Error); ok {
			exitCode = flowErr.ExitCode()
		} else {
			exitCode = errors.ExitCodeGitError
		}
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(int(exitCode))
	}
}

// ConfigAddTopicCommand adds a topic branch type configuration
func ConfigAddTopicCommand(name, parent, prefix, startingPoint, upstreamStrategy, downstreamStrategy string, tag bool) {
	if err := executeConfigAddTopic(name, parent, prefix, startingPoint, upstreamStrategy, downstreamStrategy, tag); err != nil {
		var exitCode errors.ExitCode
		if flowErr, ok := err.(errors.Error); ok {
			exitCode = flowErr.ExitCode()
		} else {
			exitCode = errors.ExitCodeGitError
		}
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(int(exitCode))
	}
}

// ConfigEditBaseCommand edits a base branch configuration
func ConfigEditBaseCommand(name, upstreamStrategy, downstreamStrategy string, autoUpdate bool) {
	if err := executeConfigEditBase(name, upstreamStrategy, downstreamStrategy, autoUpdate); err != nil {
		var exitCode errors.ExitCode
		if flowErr, ok := err.(errors.Error); ok {
			exitCode = flowErr.ExitCode()
		} else {
			exitCode = errors.ExitCodeGitError
		}
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(int(exitCode))
	}
}

// ConfigEditTopicCommand edits a topic branch type configuration
func ConfigEditTopicCommand(name, prefix, startingPoint, upstreamStrategy, downstreamStrategy string, tag bool) {
	if err := executeConfigEditTopic(name, prefix, startingPoint, upstreamStrategy, downstreamStrategy, tag); err != nil {
		var exitCode errors.ExitCode
		if flowErr, ok := err.(errors.Error); ok {
			exitCode = flowErr.ExitCode()
		} else {
			exitCode = errors.ExitCodeGitError
		}
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(int(exitCode))
	}
}

// ConfigRenameBaseCommand renames a base branch
func ConfigRenameBaseCommand(oldName, newName string) {
	if err := executeConfigRenameBase(oldName, newName); err != nil {
		var exitCode errors.ExitCode
		if flowErr, ok := err.(errors.Error); ok {
			exitCode = flowErr.ExitCode()
		} else {
			exitCode = errors.ExitCodeGitError
		}
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(int(exitCode))
	}
}

// ConfigRenameTopicCommand renames a topic branch type
func ConfigRenameTopicCommand(oldName, newName string) {
	if err := executeConfigRenameTopic(oldName, newName); err != nil {
		var exitCode errors.ExitCode
		if flowErr, ok := err.(errors.Error); ok {
			exitCode = flowErr.ExitCode()
		} else {
			exitCode = errors.ExitCodeGitError
		}
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(int(exitCode))
	}
}

// ConfigDeleteBaseCommand deletes a base branch configuration
func ConfigDeleteBaseCommand(name string) {
	if err := executeConfigDeleteBase(name); err != nil {
		var exitCode errors.ExitCode
		if flowErr, ok := err.(errors.Error); ok {
			exitCode = flowErr.ExitCode()
		} else {
			exitCode = errors.ExitCodeGitError
		}
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(int(exitCode))
	}
}

// ConfigDeleteTopicCommand deletes a topic branch type configuration
func ConfigDeleteTopicCommand(name string) {
	if err := executeConfigDeleteTopic(name); err != nil {
		var exitCode errors.ExitCode
		if flowErr, ok := err.(errors.Error); ok {
			exitCode = flowErr.ExitCode()
		} else {
			exitCode = errors.ExitCodeGitError
		}
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(int(exitCode))
	}
}

// ConfigListCommand lists the current configuration
func ConfigListCommand() {
	if err := executeConfigList(); err != nil {
		var exitCode errors.ExitCode
		if flowErr, ok := err.(errors.Error); ok {
			exitCode = flowErr.ExitCode()
		} else {
			exitCode = errors.ExitCodeGitError
		}
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(int(exitCode))
	}
}

func executeConfigAddBase(name, parent, upstreamStrategy, downstreamStrategy string, autoUpdate bool) error {
	// Validate that git-flow is initialized
	initialized, err := config.IsInitialized()
	if err != nil {
		return &errors.GitError{Operation: "check if git-flow is initialized", Err: err}
	}
	if !initialized {
		return &errors.NotInitializedError{}
	}

	// Validate branch name
	if err := util.ValidateBranchName(name); err != nil {
		return &errors.InvalidBranchNameError{BranchName: name}
	}

	// Load current configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		return &errors.GitError{Operation: "load configuration", Err: err}
	}

	// Check if branch name already exists
	if _, exists := cfg.Branches[name]; exists {
		return &errors.BranchExistsError{BranchName: name}
	}

	// Validate parent exists if specified
	if parent != "" {
		if _, exists := cfg.Branches[parent]; !exists {
			return &errors.BranchNotFoundError{BranchName: parent}
		}

		// Check for circular dependencies
		if err := validateNoCycle(cfg, name, parent); err != nil {
			return err
		}
	}

	// Set defaults
	if upstreamStrategy == "" {
		upstreamStrategy = string(config.MergeStrategyMerge)
	}
	if downstreamStrategy == "" {
		downstreamStrategy = string(config.MergeStrategyMerge)
	}

	// Validate strategies
	if !isValidMergeStrategy(upstreamStrategy) {
		return &errors.InvalidMergeStrategyError{Strategy: upstreamStrategy}
	}
	if !isValidMergeStrategy(downstreamStrategy) {
		return &errors.InvalidMergeStrategyError{Strategy: downstreamStrategy}
	}

	// Create branch configuration
	branchConfig := config.BranchConfig{
		Type:               string(config.BranchTypeBase),
		Parent:             parent,
		UpstreamStrategy:   upstreamStrategy,
		DownstreamStrategy: downstreamStrategy,
		AutoUpdate:         autoUpdate,
	}

	// Add to configuration
	cfg.Branches[name] = branchConfig

	// Save configuration
	if err := config.SaveConfig(cfg); err != nil {
		return &errors.GitError{Operation: "save configuration", Err: err}
	}

	// Create Git branch if it doesn't exist
	if err := git.BranchExists(name); err != nil {
		// Branch doesn't exist, create it
		if err := git.CreateBranch(name, parent); err != nil {
			return &errors.GitError{Operation: fmt.Sprintf("create branch '%s'", name), Err: err}
		}
		fmt.Printf("✓ Created branch '%s'\n", name)
	}

	fmt.Printf("✓ Added base branch: %s\n", name)
	return nil
}

func executeConfigAddTopic(name, parent, prefix, startingPoint, upstreamStrategy, downstreamStrategy string, tag bool) error {
	// Validate that git-flow is initialized
	initialized, err := config.IsInitialized()
	if err != nil {
		return &errors.GitError{Operation: "check if git-flow is initialized", Err: err}
	}
	if !initialized {
		return &errors.NotInitializedError{}
	}

	// Validate branch name
	if err := util.ValidateBranchName(name); err != nil {
		return &errors.InvalidBranchNameError{BranchName: name}
	}

	// Load current configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		return &errors.GitError{Operation: "load configuration", Err: err}
	}

	// Check if branch name already exists
	if _, exists := cfg.Branches[name]; exists {
		return &errors.BranchExistsError{BranchName: name}
	}

	// Validate parent exists
	if _, exists := cfg.Branches[parent]; !exists {
		return &errors.BranchNotFoundError{BranchName: parent}
	}

	// Set defaults
	if prefix == "" {
		prefix = name + "/"
	}
	if startingPoint == "" {
		startingPoint = parent
	}
	if upstreamStrategy == "" {
		upstreamStrategy = string(config.MergeStrategyMerge)
	}
	if downstreamStrategy == "" {
		downstreamStrategy = string(config.MergeStrategyMerge)
	}

	// Validate strategies
	if !isValidMergeStrategy(upstreamStrategy) {
		return &errors.InvalidMergeStrategyError{Strategy: upstreamStrategy}
	}
	if !isValidMergeStrategy(downstreamStrategy) {
		return &errors.InvalidMergeStrategyError{Strategy: downstreamStrategy}
	}

	// Validate starting point exists
	if _, exists := cfg.Branches[startingPoint]; !exists {
		return &errors.BranchNotFoundError{BranchName: startingPoint}
	}

	// Create branch configuration
	branchConfig := config.BranchConfig{
		Type:               string(config.BranchTypeTopic),
		Parent:             parent,
		StartPoint:         startingPoint,
		UpstreamStrategy:   upstreamStrategy,
		DownstreamStrategy: downstreamStrategy,
		Prefix:             prefix,
		Tag:                tag,
	}

	// Add to configuration
	cfg.Branches[name] = branchConfig

	// Save configuration
	if err := config.SaveConfig(cfg); err != nil {
		return &errors.GitError{Operation: "save configuration", Err: err}
	}

	fmt.Printf("✓ Added topic branch type: %s\n", name)
	return nil
}

func executeConfigEditBase(name, upstreamStrategy, downstreamStrategy string, autoUpdate bool) error {
	// Validate that git-flow is initialized
	initialized, err := config.IsInitialized()
	if err != nil {
		return &errors.GitError{Operation: "check if git-flow is initialized", Err: err}
	}
	if !initialized {
		return &errors.NotInitializedError{}
	}

	// Load current configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		return &errors.GitError{Operation: "load configuration", Err: err}
	}

	// Check if branch exists
	branchConfig, exists := cfg.Branches[name]
	if !exists {
		return &errors.BranchNotFoundError{BranchName: name}
	}

	// Check if it's a base branch
	if branchConfig.Type != string(config.BranchTypeBase) {
		return &errors.InvalidBranchTypeError{BranchType: branchConfig.Type}
	}

	// Update strategies if provided
	if upstreamStrategy != "" {
		if !isValidMergeStrategy(upstreamStrategy) {
			return &errors.InvalidMergeStrategyError{Strategy: upstreamStrategy}
		}
		branchConfig.UpstreamStrategy = upstreamStrategy
	}

	if downstreamStrategy != "" {
		if !isValidMergeStrategy(downstreamStrategy) {
			return &errors.InvalidMergeStrategyError{Strategy: downstreamStrategy}
		}
		branchConfig.DownstreamStrategy = downstreamStrategy
	}

	// Update auto-update setting
	branchConfig.AutoUpdate = autoUpdate

	// Update configuration
	cfg.Branches[name] = branchConfig

	// Save configuration
	if err := config.SaveConfig(cfg); err != nil {
		return &errors.GitError{Operation: "save configuration", Err: err}
	}

	fmt.Printf("✓ Updated base branch: %s\n", name)
	return nil
}

func executeConfigEditTopic(name, prefix, startingPoint, upstreamStrategy, downstreamStrategy string, tag bool) error {
	// Validate that git-flow is initialized
	initialized, err := config.IsInitialized()
	if err != nil {
		return &errors.GitError{Operation: "check if git-flow is initialized", Err: err}
	}
	if !initialized {
		return &errors.NotInitializedError{}
	}

	// Load current configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		return &errors.GitError{Operation: "load configuration", Err: err}
	}

	// Check if branch exists
	branchConfig, exists := cfg.Branches[name]
	if !exists {
		return &errors.BranchNotFoundError{BranchName: name}
	}

	// Check if it's a topic branch
	if branchConfig.Type != string(config.BranchTypeTopic) {
		return &errors.InvalidBranchTypeError{BranchType: branchConfig.Type}
	}

	// Update fields if provided
	if prefix != "" {
		branchConfig.Prefix = prefix
	}

	if startingPoint != "" {
		if _, exists := cfg.Branches[startingPoint]; !exists {
			return &errors.BranchNotFoundError{BranchName: startingPoint}
		}
		branchConfig.StartPoint = startingPoint
	}

	if upstreamStrategy != "" {
		if !isValidMergeStrategy(upstreamStrategy) {
			return &errors.InvalidMergeStrategyError{Strategy: upstreamStrategy}
		}
		branchConfig.UpstreamStrategy = upstreamStrategy
	}

	if downstreamStrategy != "" {
		if !isValidMergeStrategy(downstreamStrategy) {
			return &errors.InvalidMergeStrategyError{Strategy: downstreamStrategy}
		}
		branchConfig.DownstreamStrategy = downstreamStrategy
	}

	branchConfig.Tag = tag

	// Update configuration
	cfg.Branches[name] = branchConfig

	// Save configuration
	if err := config.SaveConfig(cfg); err != nil {
		return &errors.GitError{Operation: "save configuration", Err: err}
	}

	fmt.Printf("✓ Updated topic branch type: %s\n", name)
	return nil
}

func executeConfigRenameBase(oldName, newName string) error {
	// Validate that git-flow is initialized
	initialized, err := config.IsInitialized()
	if err != nil {
		return &errors.GitError{Operation: "check if git-flow is initialized", Err: err}
	}
	if !initialized {
		return &errors.NotInitializedError{}
	}

	// Validate new branch name
	if err := util.ValidateBranchName(newName); err != nil {
		return &errors.InvalidBranchNameError{BranchName: newName}
	}

	// Load current configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		return &errors.GitError{Operation: "load configuration", Err: err}
	}

	// Check if old branch exists
	branchConfig, exists := cfg.Branches[oldName]
	if !exists {
		return &errors.BranchNotFoundError{BranchName: oldName}
	}

	// Check if it's a base branch
	if branchConfig.Type != string(config.BranchTypeBase) {
		return &errors.InvalidBranchTypeError{BranchType: branchConfig.Type}
	}

	// Check if new name already exists
	if _, exists := cfg.Branches[newName]; exists {
		return &errors.BranchExistsError{BranchName: newName}
	}

	// Rename Git branch if it exists
	if err := git.BranchExists(oldName); err == nil {
		if err := git.RenameBranch(oldName, newName); err != nil {
			return &errors.GitError{Operation: fmt.Sprintf("rename branch '%s' to '%s'", oldName, newName), Err: err}
		}
		fmt.Printf("✓ Renamed Git branch: %s → %s\n", oldName, newName)
	}

	// Remove old branch config from git config
	if err := git.UnsetConfigSection(fmt.Sprintf("gitflow.branch.%s", oldName)); err != nil {
		return &errors.GitError{Operation: fmt.Sprintf("remove old branch config for '%s'", oldName), Err: err}
	}

	// Update configuration
	delete(cfg.Branches, oldName)
	cfg.Branches[newName] = branchConfig

	// Update all references to the old name
	for name, branch := range cfg.Branches {
		if branch.Parent == oldName {
			branch.Parent = newName
			cfg.Branches[name] = branch
		}
		if branch.StartPoint == oldName {
			branch.StartPoint = newName
			cfg.Branches[name] = branch
		}
	}

	// Save configuration
	if err := config.SaveConfig(cfg); err != nil {
		return &errors.GitError{Operation: "save configuration", Err: err}
	}

	fmt.Printf("✓ Renamed base branch: %s → %s\n", oldName, newName)
	return nil
}

func executeConfigRenameTopic(oldName, newName string) error {
	// Validate that git-flow is initialized
	initialized, err := config.IsInitialized()
	if err != nil {
		return &errors.GitError{Operation: "check if git-flow is initialized", Err: err}
	}
	if !initialized {
		return &errors.NotInitializedError{}
	}

	// Validate new branch name
	if err := util.ValidateBranchName(newName); err != nil {
		return &errors.InvalidBranchNameError{BranchName: newName}
	}

	// Load current configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		return &errors.GitError{Operation: "load configuration", Err: err}
	}

	// Check if old branch exists
	branchConfig, exists := cfg.Branches[oldName]
	if !exists {
		return &errors.BranchNotFoundError{BranchName: oldName}
	}

	// Check if it's a topic branch
	if branchConfig.Type != string(config.BranchTypeTopic) {
		return &errors.InvalidBranchTypeError{BranchType: branchConfig.Type}
	}

	// Check if new name already exists
	if _, exists := cfg.Branches[newName]; exists {
		return &errors.BranchExistsError{BranchName: newName}
	}

	// Update configuration
	delete(cfg.Branches, oldName)
	cfg.Branches[newName] = branchConfig

	// Save configuration
	if err := config.SaveConfig(cfg); err != nil {
		return &errors.GitError{Operation: "save configuration", Err: err}
	}

	fmt.Printf("✓ Renamed topic branch type: %s → %s\n", oldName, newName)
	return nil
}

func executeConfigDeleteBase(name string) error {
	// Validate that git-flow is initialized
	initialized, err := config.IsInitialized()
	if err != nil {
		return &errors.GitError{Operation: "check if git-flow is initialized", Err: err}
	}
	if !initialized {
		return &errors.NotInitializedError{}
	}

	// Load current configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		return &errors.GitError{Operation: "load configuration", Err: err}
	}

	// Check if branch exists
	branchConfig, exists := cfg.Branches[name]
	if !exists {
		return &errors.BranchNotFoundError{BranchName: name}
	}

	// Check if it's a base branch
	if branchConfig.Type != string(config.BranchTypeBase) {
		return &errors.InvalidBranchTypeError{BranchType: branchConfig.Type}
	}

	// Check if other branches depend on this branch
	for branchName, branch := range cfg.Branches {
		if branch.Parent == name {
			return &errors.BranchHasDependentsError{BranchName: name, Dependent: branchName}
		}
		if branch.StartPoint == name {
			return &errors.BranchHasDependentsError{BranchName: name, Dependent: branchName}
		}
	}

	// Remove branch config from git config
	if err := git.UnsetConfigSection(fmt.Sprintf("gitflow.branch.%s", name)); err != nil {
		return &errors.GitError{Operation: fmt.Sprintf("remove branch config for '%s'", name), Err: err}
	}

	// Remove from configuration
	delete(cfg.Branches, name)

	// Save configuration
	if err := config.SaveConfig(cfg); err != nil {
		return &errors.GitError{Operation: "save configuration", Err: err}
	}

	fmt.Printf("✓ Deleted base branch configuration: %s\n", name)
	fmt.Println("Note: Git branch was not deleted")
	return nil
}

func executeConfigDeleteTopic(name string) error {
	// Validate that git-flow is initialized
	initialized, err := config.IsInitialized()
	if err != nil {
		return &errors.GitError{Operation: "check if git-flow is initialized", Err: err}
	}
	if !initialized {
		return &errors.NotInitializedError{}
	}

	// Load current configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		return &errors.GitError{Operation: "load configuration", Err: err}
	}

	// Check if branch exists
	branchConfig, exists := cfg.Branches[name]
	if !exists {
		return &errors.BranchNotFoundError{BranchName: name}
	}

	// Check if it's a topic branch
	if branchConfig.Type != string(config.BranchTypeTopic) {
		return &errors.InvalidBranchTypeError{BranchType: branchConfig.Type}
	}

	// Remove branch config from git config
	if err := git.UnsetConfigSection(fmt.Sprintf("gitflow.branch.%s", name)); err != nil {
		return &errors.GitError{Operation: fmt.Sprintf("remove branch config for '%s'", name), Err: err}
	}

	// Remove from configuration
	delete(cfg.Branches, name)

	// Save configuration
	if err := config.SaveConfig(cfg); err != nil {
		return &errors.GitError{Operation: "save configuration", Err: err}
	}

	fmt.Printf("✓ Deleted topic branch type: %s\n", name)
	return nil
}

func executeConfigList() error {
	// Validate that git-flow is initialized
	initialized, err := config.IsInitialized()
	if err != nil {
		return &errors.GitError{Operation: "check if git-flow is initialized", Err: err}
	}
	if !initialized {
		fmt.Println("Git-flow is not initialized in this repository.")
		fmt.Println("Run 'git-flow init' to set up git-flow configuration.")
		return nil
	}

	// Load current configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		return &errors.GitError{Operation: "load configuration", Err: err}
	}

	if len(cfg.Branches) == 0 {
		fmt.Println("No git-flow configuration found.")
		return nil
	}

	fmt.Println("Git-flow Configuration")
	fmt.Println("======================")
	fmt.Println()

	// Find and categorize branches
	var trunkBranches []string
	var baseBranches []string
	var topicBranches []string

	for name, branch := range cfg.Branches {
		if branch.Type == string(config.BranchTypeBase) {
			if branch.Parent == "" {
				trunkBranches = append(trunkBranches, name)
			} else {
				baseBranches = append(baseBranches, name)
			}
		} else {
			topicBranches = append(topicBranches, name)
		}
	}

	// Display base branches
	fmt.Println("Base branches:")
	fmt.Println("--------------")

	// Display trunk branches
	for _, name := range trunkBranches {
		fmt.Printf("  %s → (root)\n", name)
		fmt.Println("    Upstream: none, Downstream: none")
		fmt.Println()
	}

	// Display child base branches
	for _, name := range baseBranches {
		branch := cfg.Branches[name]
		fmt.Printf("  %s → %s\n", name, branch.Parent)
		fmt.Printf("    Upstream: %s, Downstream: %s\n",
			branch.UpstreamStrategy, branch.DownstreamStrategy)
		if branch.AutoUpdate {
			fmt.Println("    Auto-update: enabled")
		}
		fmt.Println()
	}

	// Display topic branch types
	if len(topicBranches) > 0 {
		fmt.Println("Topic branch types:")
		fmt.Println("-------------------")
		for _, name := range topicBranches {
			branch := cfg.Branches[name]
			fmt.Printf("%s:\n", name)
			fmt.Printf("  Parent: %s\n", branch.Parent)

			// Show start point if different from parent
			if branch.StartPoint != "" && branch.StartPoint != branch.Parent {
				fmt.Printf("  Start point: %s\n", branch.StartPoint)
			} else {
				fmt.Printf("  Start point: %s\n", branch.Parent)
			}

			fmt.Printf("  Prefix: %s\n", branch.Prefix)
			fmt.Printf("  Upstream: %s, Downstream: %s\n",
				branch.UpstreamStrategy, branch.DownstreamStrategy)

			if branch.Tag {
				fmt.Println("  Creates tags: yes")
				if branch.TagPrefix != "" {
					fmt.Printf("  Tag prefix: %s\n", branch.TagPrefix)
				}
			} else {
				fmt.Println("  Creates tags: no")
			}
			fmt.Println()
		}
	}

	// Print configuration help
	fmt.Println("──────────────────────────────────────────────────────────")
	fmt.Println()
	fmt.Println("Configuration commands:")
	fmt.Println("  Add base branch:     git flow config add base <name> <parent>")
	fmt.Println("  Add topic type:      git flow config add topic <name> <parent> --prefix <prefix>")
	fmt.Println("  Edit base branch:    git flow config edit base <name> [options]")
	fmt.Println("  Edit topic type:     git flow config edit topic <name> [options]")
	fmt.Println("  Delete branch type:  git flow config delete <base|topic> <name>")
	fmt.Println()
	fmt.Println("Common options:")
	fmt.Println("  --upstream-strategy <merge|rebase|squash>")
	fmt.Println("  --downstream-strategy <merge|rebase>")
	fmt.Println("  --auto-update        Enable auto-update for base branches")
	fmt.Println("  --tag                Enable tag creation for topic types")
	fmt.Println()
	fmt.Println("For detailed help: git flow config --help")

	return nil
}

// Helper functions

func isValidMergeStrategy(strategy string) bool {
	switch strategy {
	case string(config.MergeStrategyNone),
		string(config.MergeStrategyMerge),
		string(config.MergeStrategyRebase),
		string(config.MergeStrategySquash):
		return true
	}
	return false
}

func validateNoCycle(cfg *config.Config, name, parent string) error {
	visited := make(map[string]bool)

	current := parent
	for current != "" {
		if visited[current] {
			return &errors.CircularDependencyError{BranchName: name}
		}
		if current == name {
			return &errors.CircularDependencyError{BranchName: name}
		}

		visited[current] = true

		if branch, exists := cfg.Branches[current]; exists {
			current = branch.Parent
		} else {
			break
		}
	}

	return nil
}

func init() {
	rootCmd.AddCommand(configCmd)

	// Add subcommands
	configCmd.AddCommand(configAddCmd)
	configCmd.AddCommand(configEditCmd)
	configCmd.AddCommand(configRenameCmd)
	configCmd.AddCommand(configDeleteCmd)
	configCmd.AddCommand(configListCmd)

	// Add base/topic subcommands
	configAddCmd.AddCommand(configAddBaseCmd)
	configAddCmd.AddCommand(configAddTopicCmd)
	configEditCmd.AddCommand(configEditBaseCmd)
	configEditCmd.AddCommand(configEditTopicCmd)
	configRenameCmd.AddCommand(configRenameBaseCmd)
	configRenameCmd.AddCommand(configRenameTopicCmd)
	configDeleteCmd.AddCommand(configDeleteBaseCmd)
	configDeleteCmd.AddCommand(configDeleteTopicCmd)

	// Add flags for base commands
	configAddBaseCmd.Flags().String("upstream-strategy", "", "Merge strategy when merging to parent (merge|rebase|squash)")
	configAddBaseCmd.Flags().String("downstream-strategy", "", "Merge strategy when updating from parent (merge|rebase)")
	configAddBaseCmd.Flags().Bool("auto-update", false, "Auto-update from parent on finish")

	configEditBaseCmd.Flags().String("upstream-strategy", "", "Merge strategy when merging to parent (merge|rebase|squash)")
	configEditBaseCmd.Flags().String("downstream-strategy", "", "Merge strategy when updating from parent (merge|rebase)")
	configEditBaseCmd.Flags().Bool("auto-update", false, "Auto-update from parent on finish")

	// Add flags for topic commands
	configAddTopicCmd.Flags().String("prefix", "", "Branch name prefix")
	configAddTopicCmd.Flags().String("starting-point", "", "Branch to create from (defaults to parent)")
	configAddTopicCmd.Flags().String("upstream-strategy", "", "Merge strategy when merging to parent (merge|rebase|squash)")
	configAddTopicCmd.Flags().String("downstream-strategy", "", "Merge strategy when updating from parent (merge|rebase)")
	configAddTopicCmd.Flags().Bool("tag", false, "Create tags on finish")

	configEditTopicCmd.Flags().String("prefix", "", "Branch name prefix")
	configEditTopicCmd.Flags().String("starting-point", "", "Branch to create from")
	configEditTopicCmd.Flags().String("upstream-strategy", "", "Merge strategy when merging to parent (merge|rebase|squash)")
	configEditTopicCmd.Flags().String("downstream-strategy", "", "Merge strategy when updating from parent (merge|rebase)")
	configEditTopicCmd.Flags().Bool("tag", false, "Create tags on finish")
}
