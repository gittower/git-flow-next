package cmd

import (
	"fmt"

	"github.com/gittower/git-flow-next/config"
	"github.com/spf13/cobra"
)

// RegisterTopicBranchCommands dynamically creates commands for topic branches
// based on configuration.
func RegisterTopicBranchCommands() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		// If we can't load the config, fall back to standard branch types
		fmt.Println("Warning: Could not load git-flow configuration, using default branch types")
		registerDefaultBranchCommands()
		return
	}

	// Get topic branch types from configuration
	topicBranchTypes := []string{}
	for branchName, branchConfig := range cfg.Branches {
		if branchConfig.Type == string(config.BranchTypeTopic) {
			topicBranchTypes = append(topicBranchTypes, branchName)
		}
	}

	// If no topic branch types found, use defaults
	if len(topicBranchTypes) == 0 {
		registerDefaultBranchCommands()
		return
	}

	// Register commands for each topic branch type
	for _, branchType := range topicBranchTypes {
		registerBranchCommand(branchType)
	}
}

// registerDefaultBranchCommands registers commands for standard branch types
func registerDefaultBranchCommands() {
	// Standard branch types
	branchTypes := []string{"feature", "release", "hotfix", "support"}

	// Register commands for each branch type
	for _, branchType := range branchTypes {
		registerBranchCommand(branchType)
	}
}

// registerBranchCommand registers a command for a branch type
func registerBranchCommand(branchType string) {
	// Create command for this branch type
	branchCmd := &cobra.Command{
		Use:   branchType,
		Short: fmt.Sprintf("Manage %s branches", branchType),
		Long:  fmt.Sprintf("Manage %s branches according to git-flow model", branchType),
		Run: func(cmd *cobra.Command, args []string) {
			// If no subcommand is provided, print help
			cmd.Help()
		},
	}

	// Add start subcommand
	startCmd := &cobra.Command{
		Use:     "start [name]",
		Short:   fmt.Sprintf("Start a new %s branch", branchType),
		Long:    fmt.Sprintf("Start a new %s branch from the appropriate base branch", branchType),
		Example: fmt.Sprintf("  git flow %s start my-new-feature", branchType),
		Args:    cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			// Call the generic start command with the branch type and name
			StartCommand(branchType, args[0])
		},
	}
	branchCmd.AddCommand(startCmd)

	// Add finish subcommand
	finishCmd := &cobra.Command{
		Use:     "finish [name]",
		Short:   fmt.Sprintf("Finish a %s branch", branchType),
		Long:    fmt.Sprintf("Finish a %s branch by merging it into the appropriate base branch", branchType),
		Example: fmt.Sprintf("  git flow %s finish my-feature", branchType),
		Args:    cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			// Call the generic finish command with the branch type and name
			FinishCommand(branchType, args[0])
		},
	}
	branchCmd.AddCommand(finishCmd)

	// Add list subcommand
	listCmd := &cobra.Command{
		Use:     "list",
		Short:   fmt.Sprintf("List all %s branches", branchType),
		Long:    fmt.Sprintf("List all %s branches in the repository", branchType),
		Example: fmt.Sprintf("  git flow %s list", branchType),
		Args:    cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			// For now, just print a message
			fmt.Println("Not implemented yet")
		},
	}
	branchCmd.AddCommand(listCmd)

	// Add the branch command to the root command
	rootCmd.AddCommand(branchCmd)
}

func init() {
	// Register topic branch commands
	RegisterTopicBranchCommands()
}
