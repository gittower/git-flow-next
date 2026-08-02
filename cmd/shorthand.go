package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/gittower/git-flow-next/internal/config"
	"github.com/spf13/cobra"
)

type notTopicBranchError struct {
	branch string
}

func (err *notTopicBranchError) Error() string {
	return fmt.Sprintf("current branch '%s' is not a valid topic branch (use explicit command, e.g., git flow feature finish)", err.branch)
}

// init registers all shorthand commands automatically
func init() {
	RegisterShorthandCommands()
}

// RegisterShorthandCommands adds shorthand commands to the root
func RegisterShorthandCommands() {
	// Delete (with optional name for off-branch deletion, per issue test case)
	deleteCmd := &cobra.Command{
		Use:   "delete [name]",
		Short: "Delete the current topic branch (or specified if provided)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var branchType, name string
			var err error
			if len(args) > 0 {
				// Use provided name (detect type from it)
				branchType, name, err = detectBranchTypeAndNameFromString(args[0])
			} else {
				// Use current branch
				branchType, name, err = detectBranchTypeAndName()
			}
			if err != nil {
				return err
			}
			forceFlag, _ := cmd.Flags().GetBool("force")
			noForceFlag, _ := cmd.Flags().GetBool("no-force")
			force := getBoolFlag(forceFlag, noForceFlag)
			remoteFlag, _ := cmd.Flags().GetBool("remote")
			noRemoteFlag, _ := cmd.Flags().GetBool("no-remote")
			remote := getBoolFlag(remoteFlag, noRemoteFlag)
			fetchFlag, _ := cmd.Flags().GetBool("fetch")
			noFetchFlag, _ := cmd.Flags().GetBool("no-fetch")
			fetch := getBoolFlag(fetchFlag, noFetchFlag)
			DeleteCommand(branchType, name, force, remote, fetch)
			return nil
		},
	}
	deleteCmd.Flags().BoolP("force", "f", false, "Force delete even if unmerged")
	deleteCmd.Flags().Bool("no-force", false, "Don't force delete (overrides config)")
	deleteCmd.Flags().BoolP("remote", "r", false, "Delete remote tracking branch")
	deleteCmd.Flags().Bool("no-remote", false, "Don't delete remote tracking branch")
	deleteCmd.Flags().Bool("fetch", false, "Fetch from remote before deleting")
	deleteCmd.Flags().Bool("no-fetch", false, "Don't fetch from remote before deleting")
	rootCmd.AddCommand(deleteCmd)

	// Update
	updateCmd := &cobra.Command{
		Use:   "update [base-branch]",
		Short: "Update the current branch from its parent",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			continueOp, _ := cmd.Flags().GetBool("continue")
			abortOp, _ := cmd.Flags().GetBool("abort")
			useRebase, _ := cmd.Flags().GetBool("rebase")
			return runShorthandUpdate(useRebase, continueOp, abortOp, args)
		},
	}
	addUpdateFlags(updateCmd)
	rootCmd.AddCommand(updateCmd)

	// Rebase (shorthand for update --rebase)
	rebaseCmd := &cobra.Command{
		Use:   "rebase [base-branch]",
		Short: "Rebase the current branch onto its parent",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Always use rebase strategy for this shorthand
			continueOp, _ := cmd.Flags().GetBool("continue")
			abortOp, _ := cmd.Flags().GetBool("abort")
			return runShorthandUpdate(true, continueOp, abortOp, args)
		},
	}
	rebaseCmd.Flags().BoolP("continue", "c", false, "Continue the update operation after resolving conflicts")
	rebaseCmd.Flags().BoolP("abort", "a", false, "Abort the update operation and return to the original state; a no-op success when none is in progress")
	rootCmd.AddCommand(rebaseCmd)

	// Rename
	renameCmd := &cobra.Command{
		Use:   "rename [new-name]",
		Short: "Rename the current topic branch",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			branchType, oldName, err := detectBranchTypeAndName()
			if err != nil {
				return err
			}
			RenameCommand(branchType, oldName, args[0])
			return nil
		},
	}
	rootCmd.AddCommand(renameCmd)

	// Publish
	publishCmd := &cobra.Command{
		Use:   "publish",
		Short: "Publish the current topic branch to remote",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			branchType, name, err := detectBranchTypeAndName()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			pushOptions, _ := cmd.Flags().GetStringArray("push-option")
			noPushOption, _ := cmd.Flags().GetBool("no-push-option")
			PublishCommand(branchType, name, pushOptions, noPushOption)
		},
	}
	publishCmd.Flags().StringArrayP("push-option", "o", nil, "Push option to transmit to the server (repeatable)")
	publishCmd.Flags().Bool("no-push-option", false, "Don't send any push options (overrides config defaults)")
	rootCmd.AddCommand(publishCmd)

	// Finish
	finishCmd := &cobra.Command{
		Use:   "finish",
		Short: "Finish the current topic branch",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			branchType, name, err := detectBranchTypeAndName()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			continueOp, _ := cmd.Flags().GetBool("continue")
			abortOp, _ := cmd.Flags().GetBool("abort")
			force, _ := cmd.Flags().GetBool("force")
			tagOptions := &config.TagOptions{
				ShouldTag:   getBoolPtr(cmd, "tag", "notag"),
				ShouldSign:  getBoolPtr(cmd, "sign", "no-sign"),
				SigningKey:  cmd.Flag("signingkey").Value.String(),
				Message:     cmd.Flag("message").Value.String(),
				MessageFile: cmd.Flag("messagefile").Value.String(),
				TagName:     cmd.Flag("tagname").Value.String(),
			}
			retentionOptions := &config.BranchRetentionOptions{
				Keep:        getBoolPtr(cmd, "keep", "no-keep"),
				KeepRemote:  getBoolPtr(cmd, "keepremote", "no-keepremote"),
				KeepLocal:   getBoolPtr(cmd, "keeplocal", "no-keeplocal"),
				ForceDelete: getBoolPtr(cmd, "force-delete", "no-force-delete"),
			}
			// Create merge strategy options with squash message support
			mergeOptions := &config.MergeStrategyOptions{
				Rebase:         getBoolPtr(cmd, "rebase", "no-rebase"),
				PreserveMerges: getBoolPtr(cmd, "preserve-merges", "no-preserve-merges"),
				NoFF:           getBoolPtr(cmd, "no-ff", "ff"),
				Squash:         getBoolPtr(cmd, "squash", "no-squash"),
				SquashMessage:  getStringPtrFromFlag(cmd, "squash-message"),
			}
			// Get no-verify flag
			noVerify, _ := cmd.Flags().GetBool("no-verify")
			var noVerifyPtr *bool
			if noVerify {
				noVerifyPtr = &noVerify
			}
			// Resolve fetch and push flags (unset stays nil so config can apply)
			fetch := getBoolPtr(cmd, "fetch", "no-fetch")
			push := getBoolPtr(cmd, "push", "no-push")
			pushTag := getBoolPtr(cmd, "pushtag", "no-pushtag")
			FinishCommand(branchType, name, continueOp, abortOp, force, tagOptions, retentionOptions, mergeOptions, fetch, noVerifyPtr, push, pushTag)
		},
	}

	addFinishFlags(finishCmd)
	rootCmd.AddCommand(finishCmd)

	// Integrate (top-level: merge a base branch into its parent)
	integrateCmd := &cobra.Command{
		Use:   "integrate [<branch>]",
		Short: "Integrate a base branch into its parent",
		Long: `Integrate merges a base branch upstream into its configured parent
(e.g. develop into main), honoring the branch type's upstream merge strategy,
optionally tagging the parent, and auto-updating the parent's children.

Unlike finish, integrate operates on base branches only and never deletes,
creates, or renames a branch — base branches are permanent. If no branch is
given, the current branch is integrated into its parent.`,
		Example: "  git flow integrate develop\n  git flow integrate develop --tag v2.0.0\n  git flow integrate --continue",
		Args:    cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			name := ""
			if len(args) > 0 {
				name = args[0]
			}

			continueOp, _ := cmd.Flags().GetBool("continue")
			abortOp, _ := cmd.Flags().GetBool("abort")

			// --tag is a string: setting it both enables tagging and supplies
			// the tag name; --notag disables tagging.
			tagOptions := &config.TagOptions{
				ShouldSign:  getBoolPtr(cmd, "sign", "no-sign"),
				SigningKey:  cmd.Flag("signingkey").Value.String(),
				Message:     cmd.Flag("message").Value.String(),
				MessageFile: cmd.Flag("messagefile").Value.String(),
			}
			if cmd.Flags().Changed("tag") {
				enable := true
				tagOptions.ShouldTag = &enable
				tagOptions.TagName = cmd.Flag("tag").Value.String()
			} else if cmd.Flags().Changed("notag") {
				disable := false
				tagOptions.ShouldTag = &disable
			}

			mergeOptions := &config.MergeStrategyOptions{
				Rebase:         getBoolPtr(cmd, "rebase", "no-rebase"),
				PreserveMerges: getBoolPtr(cmd, "preserve-merges", "no-preserve-merges"),
				NoFF:           getBoolPtr(cmd, "no-ff", "ff"),
				Squash:         getBoolPtr(cmd, "squash", "no-squash"),
				SquashMessage:  getStringPtrFromFlag(cmd, "squash-message"),
				MergeMessage:   getStringPtrFromFlag(cmd, "merge-message"),
				UpdateMessage:  getStringPtrFromFlag(cmd, "update-message"),
			}

			fetch := getBoolPtr(cmd, "fetch", "no-fetch")

			IntegrateCommand(name, continueOp, abortOp, tagOptions, mergeOptions, fetch)
		},
	}
	addIntegrateFlags(integrateCmd)
	rootCmd.AddCommand(integrateCmd)
}

// runShorthandUpdate resolves the branch to update for the top-level update/rebase
// surface. A topic current branch is updated by its type; a non-topic current
// branch falls back to the current or named base branch (branchType == "").
func runShorthandUpdate(useRebase, continueOp, abortOp bool, args []string) error {
	branchType, name, err := detectBranchTypeAndName()
	if err == nil {
		UpdateCommand(branchType, name, useRebase, continueOp, abortOp)
		return nil
	}
	var notTopicErr *notTopicBranchError
	if !errors.As(err, &notTopicErr) {
		return err
	}
	// Not a topic branch: fall back to the current or named base branch.
	var branchName string
	if len(args) > 0 {
		branchName = args[0]
	}
	UpdateCommand("", branchName, useRebase, continueOp, abortOp)
	return nil
}

// detectBranchTypeAndName detects type and name from current branch
func detectBranchTypeAndName() (string, string, error) {
	repo, err := openRepo()
	if err != nil {
		return "", "", err
	}
	cfg, err := config.Load(repo)
	if err != nil {
		return "", "", err
	}
	currentBranch, err := repo.GetCurrentBranch()
	if err != nil {
		return "", "", err
	}
	if currentBranch == "" {
		return "", "", fmt.Errorf("no current branch")
	}

	matches := []struct{ Type, Prefix string }{}
	for typ, bc := range cfg.Branches {
		if bc.Type == string(config.BranchTypeTopic) && strings.HasPrefix(currentBranch, bc.Prefix) {
			matches = append(matches, struct{ Type, Prefix string }{typ, bc.Prefix})
		}
	}

	switch len(matches) {
	case 0:
		return "", "", &notTopicBranchError{branch: currentBranch}
	case 1:
		typ := matches[0].Type
		name := strings.TrimPrefix(currentBranch, matches[0].Prefix)
		return typ, name, nil
	default:
		// Ambiguous: Prompt
		typesStr := []string{}
		for _, m := range matches {
			typesStr = append(typesStr, m.Type)
		}
		fmt.Printf("Ambiguous branch '%s' matches multiple types: %s\n", currentBranch, strings.Join(typesStr, ", "))
		fmt.Print("Use explicit command? [Y/n]: ")
		reader := bufio.NewReader(os.Stdin)
		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(strings.ToLower(response))
		if response == "n" {
			return "", "", fmt.Errorf("operation cancelled")
		}
		return "", "", fmt.Errorf("please use explicit command (e.g., git flow feature finish)")
	}
}

// detectBranchTypeAndNameFromString detects from a given string (for delete [name])
func detectBranchTypeAndNameFromString(branch string) (string, string, error) {
	repo, err := openRepo()
	if err != nil {
		return "", "", err
	}
	cfg, err := config.Load(repo)
	if err != nil {
		return "", "", err
	}
	matches := []struct{ Type, Prefix string }{}
	for typ, bc := range cfg.Branches {
		if bc.Type == string(config.BranchTypeTopic) && strings.HasPrefix(branch, bc.Prefix) {
			matches = append(matches, struct{ Type, Prefix string }{typ, bc.Prefix})
		}
	}

	switch len(matches) {
	case 0:
		return "", "", fmt.Errorf("branch '%s' is not a valid topic branch", branch)
	case 1:
		typ := matches[0].Type
		name := strings.TrimPrefix(branch, matches[0].Prefix)
		return typ, name, nil
	default:
		return "", "", fmt.Errorf("ambiguous branch '%s' matches multiple types", branch)
	}
}

// getBoolPtr converts mutually exclusive bool flags to *bool
func getBoolPtr(cmd *cobra.Command, trueFlag, falseFlag string) *bool {
	if cmd.Flags().Changed(trueFlag) {
		t := true
		return &t
	}
	if cmd.Flags().Changed(falseFlag) {
		f := false
		return &f
	}
	return nil
}

// getStringPtrFromFlag gets a string flag value and returns nil if empty
func getStringPtrFromFlag(cmd *cobra.Command, flagName string) *string {
	value := cmd.Flag(flagName).Value.String()
	if value == "" {
		return nil
	}
	return &value
}
