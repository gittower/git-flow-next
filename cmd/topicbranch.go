package cmd

import (
	"fmt"
	"os"

	"github.com/gittower/git-flow-next/internal/config"
	"github.com/gittower/git-flow-next/internal/errors"
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
			// Get fetch flag values
			fetch, _ := cmd.Flags().GetBool("fetch")
			noFetch, _ := cmd.Flags().GetBool("no-fetch")

			// Pass nil if no flags are set, otherwise create an appropriate bool pointer
			var shouldFetch *bool
			if fetch {
				t := true
				shouldFetch = &t
			} else if noFetch {
				f := false
				shouldFetch = &f
			}

			// Call the generic start command with the branch type, name, and fetch flags
			StartCommand(branchType, args[0], shouldFetch)
		},
	}

	// Add fetch-related flags
	startCmd.Flags().Bool("fetch", false, "Fetch from remote before creating branch")
	startCmd.Flags().Bool("no-fetch", false, "Don't fetch from remote before creating branch")

	branchCmd.AddCommand(startCmd)

	// Add finish subcommand
	finishCmd := &cobra.Command{
		Use:     "finish [name]",
		Short:   fmt.Sprintf("Finish a %s branch", branchType),
		Long:    fmt.Sprintf("Finish a %s branch by merging it into the appropriate base branch", branchType),
		Example: fmt.Sprintf("  git flow %s finish my-feature\n  git flow %s finish other/branch -f", branchType, branchType),
		Args:    cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			// Get flags
			continueOp, _ := cmd.Flags().GetBool("continue")
			abortOp, _ := cmd.Flags().GetBool("abort")
			force, _ := cmd.Flags().GetBool("force")

			// Get tag-related flags
			tag, _ := cmd.Flags().GetBool("tag")
			noTag, _ := cmd.Flags().GetBool("notag")
			sign, _ := cmd.Flags().GetBool("sign")
			noSign, _ := cmd.Flags().GetBool("no-sign")
			signingKey, _ := cmd.Flags().GetString("signingkey")
			message, _ := cmd.Flags().GetString("message")
			messageFile, _ := cmd.Flags().GetString("messagefile")
			tagName, _ := cmd.Flags().GetString("tagname")

			// Get branch retention flags
			keep, _ := cmd.Flags().GetBool("keep")
			noKeep, _ := cmd.Flags().GetBool("no-keep")
			keepRemote, _ := cmd.Flags().GetBool("keepremote")
			noKeepRemote, _ := cmd.Flags().GetBool("no-keepremote")
			keepLocal, _ := cmd.Flags().GetBool("keeplocal")
			noKeepLocal, _ := cmd.Flags().GetBool("no-keeplocal")
			forceDelete, _ := cmd.Flags().GetBool("force-delete")
			noForceDelete, _ := cmd.Flags().GetBool("no-force-delete")

			// Create tag options
			tagOptions := &TagOptions{
				ShouldTag:   getBoolFlag(tag, noTag),
				ShouldSign:  getBoolFlag(sign, noSign),
				SigningKey:  signingKey,
				Message:     message,
				MessageFile: messageFile,
				TagName:     tagName,
			}

			// Create branch retention options
			retentionOptions := &BranchRetentionOptions{
				Keep:        getBoolFlag(keep, noKeep),
				KeepRemote:  getBoolFlag(keepRemote, noKeepRemote),
				KeepLocal:   getBoolFlag(keepLocal, noKeepLocal),
				ForceDelete: getBoolFlag(forceDelete, noForceDelete),
			}

			// Call the generic finish command with the branch type and name
			FinishCommand(branchType, args[0], continueOp, abortOp, force, tagOptions, retentionOptions)
		},
	}

	addFinishFlags(finishCmd)
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
			if err := executeUpdate(branchType, name, false); err != nil {
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
			remote, _ := cmd.Flags().GetBool("remote")
			noRemote, _ := cmd.Flags().GetBool("no-remote")

			// Convert remote flags to a single *bool
			var remotePtr *bool
			if remote {
				remotePtr = &remote
			} else if noRemote {
				falseBool := false
				remotePtr = &falseBool
			}

			if err := DeleteCommand(branchType, args[0], force, remotePtr); err != nil {
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
	deleteCmd.Flags().BoolP("remote", "r", false, "Delete the remote tracking branch")
	deleteCmd.Flags().Bool("no-remote", false, "Don't delete the remote tracking branch")

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

	// Add checkout subcommand
	checkoutCmd := &cobra.Command{
		Use:     "checkout [name|nameprefix]",
		Short:   fmt.Sprintf("Switch to a %s branch", branchType),
		Long:    fmt.Sprintf("Switch to %s branch <name>. If only a prefix is provided, switch to the matching branch if unambiguous.", branchType),
		Example: fmt.Sprintf("  git flow %s checkout my-feature\n  git flow %s checkout my", branchType, branchType),
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			nameOrPrefix := ""
			if len(args) > 0 {
				nameOrPrefix = args[0]
			}
			showCommands, _ := cmd.Flags().GetBool("showcommands")
			if err := CheckoutCommand(branchType, nameOrPrefix, showCommands); err != nil {
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
	checkoutCmd.Flags().Bool("showcommands", false, "Show git commands while executing them")

	branchCmd.AddCommand(checkoutCmd)

	// Add the branch command to the root command
	rootCmd.AddCommand(branchCmd)
}

func init() {
	// Register topic branch commands
	RegisterTopicBranchCommands()
}

// addFinishFlags adds common finish flags to the given Cobra command
func addFinishFlags(cmd *cobra.Command) {
	// Operation Control Flags
	cmd.Flags().BoolP("continue", "c", false, "Continue the finish operation after resolving conflicts")
	cmd.Flags().BoolP("abort", "a", false, "Abort the finish operation and return to the original state")
	cmd.Flags().BoolP("force", "f", false, "Force finish a non-standard branch using this branch type's strategy")

	// Tag-related Flags
	cmd.Flags().Bool("tag", false, "Create a tag for the finished branch")
	cmd.Flags().Bool("notag", false, "Don't create a tag for the finished branch")
	cmd.Flags().Bool("sign", false, "Sign the tag cryptographically")
	cmd.Flags().Bool("no-sign", false, "Don't sign the tag cryptographically")
	cmd.Flags().String("signingkey", "", "Use the given GPG key for the digital signature")
	cmd.Flags().StringP("message", "m", "", "Use the given message for the tag")
	cmd.Flags().String("messagefile", "", "Use contents of the given file as tag message")
	cmd.Flags().String("tagname", "", "Use the given tag name instead of the default")

	// Branch Retention Flags
	cmd.Flags().Bool("keep", false, "Keep the branch after finishing")
	cmd.Flags().Bool("no-keep", false, "Delete the branch after finishing")
	cmd.Flags().Bool("keepremote", false, "Keep the remote branch after finishing")
	cmd.Flags().Bool("no-keepremote", false, "Delete the remote branch after finishing")
	cmd.Flags().Bool("keeplocal", false, "Keep the local branch after finishing")
	cmd.Flags().Bool("no-keeplocal", false, "Delete the local branch after finishing")
	cmd.Flags().Bool("force-delete", false, "Force delete the branch")
	cmd.Flags().Bool("no-force-delete", false, "Don't force delete the branch")
}
