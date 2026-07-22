package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/gittower/git-flow-next/internal/config"
	"github.com/gittower/git-flow-next/internal/errors"
	"github.com/gittower/git-flow-next/internal/git"
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
		Use:     "start [name] [base]",
		Short:   fmt.Sprintf("Start a new %s branch", branchType),
		Long:    fmt.Sprintf("Start a new %s branch from the appropriate base branch or specified base", branchType),
		Example: fmt.Sprintf("  git flow %s start my-new-feature\n  git flow %s start emergency-fix abc123def", branchType, branchType),
		Args:    cobra.RangeArgs(0, 2),
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

			// Get name argument if provided; when omitted, the version filter
			// may supply it (see start()).
			var name string
			if len(args) > 0 {
				name = args[0]
			}

			// Get base argument if provided
			var base string
			if len(args) > 1 {
				base = args[1]
			}

			// Call the generic start command with the branch type, name, base, and fetch flags
			StartCommand(branchType, name, base, shouldFetch)
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
		Long:    fmt.Sprintf("Finish a %s branch by merging it into the appropriate base branch. If no name is provided, finishes the current branch.", branchType),
		Example: fmt.Sprintf("  git flow %s finish\n  git flow %s finish my-feature\n  git flow %s finish other/branch -f", branchType, branchType, branchType),
		Args:    cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			// Validate that git-flow is initialized before any branch-name
			// detection. LoadConfig falls back to DefaultConfig when
			// uninitialized, so without this gate the detection below (and the
			// downstream finish logic) would emit a misleading "branch does not
			// exist"/"not a branch" error instead of "not initialized".
			initialized, err := config.IsInitialized()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", &errors.GitError{Operation: "check if git-flow is initialized", Err: err})
				os.Exit(int(errors.ExitCodeGitError))
			}
			if !initialized {
				notInit := &errors.NotInitializedError{}
				fmt.Fprintf(os.Stderr, "Error: %v\n", notInit)
				os.Exit(int(notInit.ExitCode()))
			}

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

			// Get merge strategy flags
			rebase, _ := cmd.Flags().GetBool("rebase")
			noRebase, _ := cmd.Flags().GetBool("no-rebase")
			preserveMerges, _ := cmd.Flags().GetBool("preserve-merges")
			noPreserveMerges, _ := cmd.Flags().GetBool("no-preserve-merges")
			noFF, _ := cmd.Flags().GetBool("no-ff")
			ff, _ := cmd.Flags().GetBool("ff")
			squash, _ := cmd.Flags().GetBool("squash")
			noSquash, _ := cmd.Flags().GetBool("no-squash")
			squashMessage, _ := cmd.Flags().GetString("squash-message")

			// Get fetch flags
			fetch, _ := cmd.Flags().GetBool("fetch")
			noFetch, _ := cmd.Flags().GetBool("no-fetch")

			// Get push flags
			push, _ := cmd.Flags().GetBool("push")
			noPush, _ := cmd.Flags().GetBool("no-push")
			pushtag, _ := cmd.Flags().GetBool("pushtag")
			noPushtag, _ := cmd.Flags().GetBool("no-pushtag")

			// Get hook bypass flag
			noVerify, _ := cmd.Flags().GetBool("no-verify")

			// Determine branch name - use provided arg or detect from current branch
			var name string
			if len(args) > 0 {
				name = args[0]
			} else {
				// No name provided, try to detect from current branch
				currentBranch, err := git.GetCurrentBranch()
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error: %v\n", err)
					os.Exit(int(errors.ExitCodeGitError))
				}
				// Load config to get prefix
				cfg, err := config.LoadConfig()
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error: %v\n", err)
					os.Exit(int(errors.ExitCodeGitError))
				}
				branchConfig, ok := cfg.Branches[branchType]
				if !ok {
					fmt.Fprintf(os.Stderr, "Error: invalid branch type '%s'\n", branchType)
					os.Exit(int(errors.ExitCodeInvalidInput))
				}
				// Verify current branch is of the correct type
				if !strings.HasPrefix(currentBranch, branchConfig.Prefix) {
					fmt.Fprintf(os.Stderr, "Error: current branch '%s' is not a %s branch\n", currentBranch, branchType)
					os.Exit(int(errors.ExitCodeBranchNotFound))
				}
				// Extract short name from current branch
				name = strings.TrimPrefix(currentBranch, branchConfig.Prefix)
			}

			// Create tag options
			tagOptions := &config.TagOptions{
				ShouldTag:   getBoolFlag(tag, noTag),
				ShouldSign:  getBoolFlag(sign, noSign),
				SigningKey:  signingKey,
				Message:     message,
				MessageFile: messageFile,
				TagName:     tagName,
			}

			// Create branch retention options
			retentionOptions := &config.BranchRetentionOptions{
				Keep:        getBoolFlag(keep, noKeep),
				KeepRemote:  getBoolFlag(keepRemote, noKeepRemote),
				KeepLocal:   getBoolFlag(keepLocal, noKeepLocal),
				ForceDelete: getBoolFlag(forceDelete, noForceDelete),
			}

			// Get merge message flags
			mergeMessage, _ := cmd.Flags().GetString("merge-message")
			updateMessage, _ := cmd.Flags().GetString("update-message")

			// Create merge strategy options
			mergeOptions := &config.MergeStrategyOptions{
				Rebase:         getBoolFlag(rebase, noRebase),
				PreserveMerges: getBoolFlag(preserveMerges, noPreserveMerges),
				NoFF:           getBoolFlag(noFF, ff),
				Squash:         getBoolFlag(squash, noSquash),
				SquashMessage:  getStringPtr(squashMessage),
				MergeMessage:   getStringPtr(mergeMessage),
				UpdateMessage:  getStringPtr(updateMessage),
			}

			// Call the generic finish command with the branch type and name
			FinishCommand(branchType, name, continueOp, abortOp, force, tagOptions, retentionOptions, mergeOptions, getBoolFlag(fetch, noFetch), getSingleBoolPtr(noVerify), getBoolFlag(push, noPush), getBoolFlag(pushtag, noPushtag))
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
			noForce, _ := cmd.Flags().GetBool("no-force")
			remote, _ := cmd.Flags().GetBool("remote")
			noRemote, _ := cmd.Flags().GetBool("no-remote")
			fetch, _ := cmd.Flags().GetBool("fetch")
			noFetch, _ := cmd.Flags().GetBool("no-fetch")

			DeleteCommand(branchType, args[0], getBoolFlag(force, noForce), getBoolFlag(remote, noRemote), getBoolFlag(fetch, noFetch))
			return nil
		},
	}

	// Add flags
	deleteCmd.Flags().BoolP("force", "f", false, "Force delete the branch even if it has unmerged changes")
	deleteCmd.Flags().Bool("no-force", false, "Don't force delete the branch (overrides config)")
	deleteCmd.Flags().BoolP("remote", "r", false, "Delete the remote tracking branch")
	deleteCmd.Flags().Bool("no-remote", false, "Don't delete the remote tracking branch")
	deleteCmd.Flags().Bool("fetch", false, "Fetch from remote before deleting")
	deleteCmd.Flags().Bool("no-fetch", false, "Don't fetch from remote before deleting")

	branchCmd.AddCommand(deleteCmd)

	// Add rename subcommand
	renameCmd := &cobra.Command{
		Use:     "rename [old-name] [new-name]",
		Short:   fmt.Sprintf("Rename a %s branch", branchType),
		Long:    fmt.Sprintf("Rename a %s branch to a new name", branchType),
		Example: fmt.Sprintf("  git flow %s rename old-feature new-feature", branchType),
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			RenameCommand(branchType, args[0], args[1])
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
			CheckoutCommand(branchType, nameOrPrefix, showCommands)
			return nil
		},
	}

	// Add flags
	checkoutCmd.Flags().Bool("showcommands", false, "Show git commands while executing them")

	branchCmd.AddCommand(checkoutCmd)

	// Add publish subcommand
	publishCmd := &cobra.Command{
		Use:   "publish [name]",
		Short: fmt.Sprintf("Publish a %s branch to the remote", branchType),
		Long: fmt.Sprintf(`Publishes a local %s branch to the configured remote repository.

This pushes the branch to the remote and sets up tracking between
the local and remote branches. After publishing, other team members
can track this branch using 'git flow %s track'.

If no name is provided, the current branch is published.`, branchType, branchType),
		Example: fmt.Sprintf("  git flow %s publish my-feature\n  git flow %s publish -o ci.skip", branchType, branchType),
		Args:    cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			name := ""
			if len(args) > 0 {
				name = args[0]
			}
			pushOptions, _ := cmd.Flags().GetStringArray("push-option")
			noPushOption, _ := cmd.Flags().GetBool("no-push-option")
			PublishCommand(branchType, name, pushOptions, noPushOption)
		},
	}
	publishCmd.Flags().StringArrayP("push-option", "o", nil, "Push option to transmit to the server (repeatable)")
	publishCmd.Flags().Bool("no-push-option", false, "Don't send any push options (overrides config defaults)")
	branchCmd.AddCommand(publishCmd)

	// Add track subcommand
	trackCmd := &cobra.Command{
		Use:   "track <name>",
		Short: fmt.Sprintf("Track a remote %s branch", branchType),
		Long: fmt.Sprintf(`Creates a local branch that tracks a remote %s branch.

This is useful when you want to work on a %s branch that was
started by someone else.`, branchType, branchType),
		Example: fmt.Sprintf("  git flow %s track my-feature", branchType),
		Args:    cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			TrackCommand(branchType, args[0])
		},
	}

	branchCmd.AddCommand(trackCmd)

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
	cmd.Flags().BoolP("abort", "a", false, "Abort the finish operation and return to the original state; a no-op success when none is in progress")
	cmd.Flags().BoolP("force", "f", false, "Force finish: skip remote branch sync check and allow finishing non-standard branches")

	// Tag-related Flags
	cmd.Flags().Bool("tag", false, "Create a tag for the finished branch")
	cmd.Flags().BoolP("notag", "n", false, "Don't create a tag for the finished branch")
	cmd.Flags().BoolP("sign", "s", false, "Sign the tag cryptographically")
	cmd.Flags().Bool("no-sign", false, "Don't sign the tag cryptographically")
	cmd.Flags().StringP("signingkey", "u", "", "Use the given GPG key for the digital signature")
	cmd.Flags().StringP("message", "m", "", "Use the given message for the tag")
	cmd.Flags().String("messagefile", "", "Use contents of the given file as tag message")
	cmd.Flags().StringP("tagname", "T", "", "Use the given tag name instead of the default")

	// Branch Retention Flags
	cmd.Flags().BoolP("keep", "k", false, "Keep the branch after finishing")
	cmd.Flags().Bool("no-keep", false, "Delete the branch after finishing")
	cmd.Flags().Bool("keepremote", false, "Keep the remote branch after finishing")
	cmd.Flags().Bool("no-keepremote", false, "Delete the remote branch after finishing")
	cmd.Flags().Bool("keeplocal", false, "Keep the local branch after finishing")
	cmd.Flags().Bool("no-keeplocal", false, "Delete the local branch after finishing")
	cmd.Flags().BoolP("force-delete", "D", false, "Force delete the branch")
	cmd.Flags().Bool("no-force-delete", false, "Don't force delete the branch")

	// Merge Strategy Flags
	cmd.Flags().BoolP("rebase", "r", false, "Rebase topic branch before merging")
	cmd.Flags().Bool("no-rebase", false, "Don't rebase topic branch (use configured strategy)")
	cmd.Flags().BoolP("preserve-merges", "p", false, "Preserve merges during rebase")
	cmd.Flags().Bool("no-preserve-merges", false, "Flatten merges during rebase")
	cmd.Flags().Bool("no-ff", false, "Create merge commit even for fast-forward")
	cmd.Flags().Bool("ff", false, "Allow fast-forward merge when possible")
	cmd.Flags().BoolP("squash", "S", false, "Squash all commits into single commit")
	cmd.Flags().Bool("no-squash", false, "Keep individual commits (don't squash)")
	cmd.Flags().String("squash-message", "", "Custom commit message for squash merge")
	cmd.Flags().StringP("merge-message", "M", "", "Custom commit message for the upstream merge operation")
	cmd.Flags().String("update-message", "", "Custom commit message for child branch update operations")

	// Fetch Flags
	cmd.Flags().Bool("fetch", false, "Fetch from remote before finishing")
	cmd.Flags().Bool("no-fetch", false, "Don't fetch from remote before finishing")

	// Remote Push Options
	cmd.Flags().Bool("push", false, "Push the target and updated child branches (and the created tag) after finishing")
	cmd.Flags().Bool("no-push", false, "Don't push after finishing (overrides config)")
	cmd.Flags().Bool("pushtag", false, "Push the created tag after finishing (defaults to the push setting)")
	cmd.Flags().Bool("no-pushtag", false, "Don't push the created tag after finishing")

	// Hook Control Flags
	cmd.Flags().Bool("no-verify", false, "Bypass pre-commit and commit-msg hooks during merge and commit operations")
}

// getBoolFlag converts two opposite boolean flags into a single *bool value
// If positive is true, returns &true
// If negative is true, returns &false
// If neither is set, returns nil
func getBoolFlag(positive, negative bool) *bool {
	if positive {
		return &positive
	}
	if negative {
		falseBool := false
		return &falseBool
	}
	return nil
}

// getStringPtr converts a string to a *string, returning nil for empty strings
func getStringPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// getSingleBoolPtr converts a bool to a *bool, returning nil for false
// This is used for flags that only have a positive form (e.g., --no-verify)
func getSingleBoolPtr(b bool) *bool {
	if !b {
		return nil
	}
	return &b
}
