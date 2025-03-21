package cmd

import (
	"fmt"
	"os"

	"github.com/gittower/git-flow-next/config"
	"github.com/gittower/git-flow-next/errors"
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
			// Get flags
			continueOp, _ := cmd.Flags().GetBool("continue")
			abortOp, _ := cmd.Flags().GetBool("abort")

			// Call the generic finish command with the branch type and name
			FinishCommand(branchType, args[0], continueOp, abortOp)
		},
	}

	// Add flags
	finishCmd.Flags().BoolP("continue", "c", false, "Continue the finish operation after resolving conflicts")
	finishCmd.Flags().BoolP("abort", "a", false, "Abort the finish operation and return to the original state")

	branchCmd.AddCommand(finishCmd)

	// Add list subcommand
	listCmd := &cobra.Command{
		Use:     "list",
		Short:   fmt.Sprintf("List all %s branches", branchType),
		Long:    fmt.Sprintf("List all %s branches in the repository", branchType),
		Example: fmt.Sprintf("  git flow %s list", branchType),
		Args:    cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			// Call the generic list command with the branch type
			ListCommand(branchType)
		},
	}
	branchCmd.AddCommand(listCmd)

	// Add update subcommand
	updateCmd := &cobra.Command{
		Use:     "update [name]",
		Short:   fmt.Sprintf("Update a %s branch with changes from its parent branch", branchType),
		Long:    fmt.Sprintf("Update a %s branch with changes from its parent branch using the configured downstream strategy", branchType),
		Example: fmt.Sprintf("  git flow %s update my-feature", branchType),
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var name string
			if len(args) > 0 {
				name = args[0]
			}
			if err := executeUpdate(branchType, name); err != nil {
				var exitCode errors.ExitCode
				if flowErr, ok := err.(errors.Error); ok {
					exitCode = flowErr.ExitCode()
				} else {
					exitCode = errors.ExitCodeGitError
				}
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(int(exitCode))
			}
			return nil
		},
	}
	branchCmd.AddCommand(updateCmd)

	// Add delete subcommand
	deleteCmd := &cobra.Command{
		Use:     "delete [name]",
		Short:   fmt.Sprintf("Delete a %s branch", branchType),
		Long:    fmt.Sprintf("Delete a %s branch from the repository", branchType),
		Example: fmt.Sprintf("  git flow %s delete my-feature\n  git flow %s delete -f my-feature", branchType, branchType),
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			force, _ := cmd.Flags().GetBool("force")
			if err := DeleteCommand(branchType, args[0], force); err != nil {
				var exitCode errors.ExitCode
				if flowErr, ok := err.(errors.Error); ok {
					exitCode = flowErr.ExitCode()
				} else {
					exitCode = errors.ExitCodeGitError
				}
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(int(exitCode))
			}
			return nil
		},
	}

	// Add flags
	deleteCmd.Flags().BoolP("force", "f", false, "Force delete the branch even if it has unmerged changes")

	branchCmd.AddCommand(deleteCmd)

	// Add rename subcommand
	renameCmd := &cobra.Command{
		Use:     "rename [old-name] [new-name]",
		Short:   fmt.Sprintf("Rename a %s branch", branchType),
		Long:    fmt.Sprintf("Rename a %s branch to a new name", branchType),
		Example: fmt.Sprintf("  git flow %s rename old-feature new-feature", branchType),
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := RenameCommand(branchType, args[0], args[1]); err != nil {
				var exitCode errors.ExitCode
				if flowErr, ok := err.(errors.Error); ok {
					exitCode = flowErr.ExitCode()
				} else {
					exitCode = errors.ExitCodeGitError
				}
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(int(exitCode))
			}
			return nil
		},
	}

	branchCmd.AddCommand(renameCmd)

	// Add the branch command to the root command
	rootCmd.AddCommand(branchCmd)
}

func init() {
	// Register topic branch commands
	RegisterTopicBranchCommands()
}
