package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/gittower/git-flow-next/internal/config"
	"github.com/gittower/git-flow-next/internal/git"
	"github.com/spf13/cobra"
)

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
			force, _ := cmd.Flags().GetBool("force")
			var remote *bool
			if cmd.Flags().Changed("remote") {
				r, _ := cmd.Flags().GetBool("remote")
				remote = &r
			} else if cmd.Flags().Changed("no-remote") {
				f := false
				remote = &f
			}
			return DeleteCommand(branchType, name, force, remote)
		},
	}
	deleteCmd.Flags().BoolP("force", "f", false, "Force delete even if unmerged")
	deleteCmd.Flags().BoolP("remote", "r", false, "Delete remote tracking branch")
	deleteCmd.Flags().Bool("no-remote", false, "Don't delete remote tracking branch")
	rootCmd.AddCommand(deleteCmd)

	// Rebase (stub, as not in codebase; add full impl if needed)
	rebaseCmd := &cobra.Command{
		Use:   "rebase",
		Short: "Rebase the current topic branch",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, _, err := detectBranchTypeAndName()
			if err != nil {
				return err
			}
			// TODO: Implement RebaseCommand(branchType, name, options...)
			return fmt.Errorf("rebase not implemented")
		},
	}
	rootCmd.AddCommand(rebaseCmd)

	// Update
	updateCmd := &cobra.Command{
		Use:   "update",
		Short: "Update the current topic branch from parent",
		RunE: func(cmd *cobra.Command, args []string) error {
			branchType, name, err := detectBranchTypeAndName()
			if err == nil {
				return executeUpdate(branchType, name)
			}
			// Fallback to original if not topic
			var branchName string
			if len(args) > 0 {
				branchName = args[0]
			}
			return executeUpdate("", branchName)
		},
	}
	rootCmd.AddCommand(updateCmd)

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
			return RenameCommand(branchType, oldName, args[0])
		},
	}
	rootCmd.AddCommand(renameCmd)

	// Publish (stub)
	publishCmd := &cobra.Command{
		Use:   "publish",
		Short: "Publish the current topic branch to remote",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, _, err := detectBranchTypeAndName()
			if err != nil {
				return err
			}
			// TODO: Implement PublishCommand(branchType, name)
			return fmt.Errorf("publish not implemented")
		},
	}
	rootCmd.AddCommand(publishCmd)

	// Finish
	finishCmd := &cobra.Command{
		Use:   "finish",
		Short: "Finish the current topic branch",
		Run: func(cmd *cobra.Command, args []string) {
			branchType, name, err := detectBranchTypeAndName()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			continueOp, _ := cmd.Flags().GetBool("continue")
			abortOp, _ := cmd.Flags().GetBool("abort")
			force, _ := cmd.Flags().GetBool("force")
			tagOptions := &TagOptions{
				ShouldTag:   getBoolPtr(cmd, "tag", "notag"),
				ShouldSign:  getBoolPtr(cmd, "sign", "no-sign"),
				SigningKey:  cmd.Flag("signingkey").Value.String(),
				Message:     cmd.Flag("message").Value.String(),
				MessageFile: cmd.Flag("messagefile").Value.String(),
				TagName:     cmd.Flag("tagname").Value.String(),
			}
			retentionOptions := &BranchRetentionOptions{
				Keep:        getBoolPtr(cmd, "keep", "no-keep"),
				KeepRemote:  getBoolPtr(cmd, "keepremote", "no-keepremote"),
				KeepLocal:   getBoolPtr(cmd, "keeplocal", "no-keeplocal"),
				ForceDelete: getBoolPtr(cmd, "force-delete", "no-force-delete"),
			}
			FinishCommand(branchType, name, continueOp, abortOp, force, tagOptions, retentionOptions)
		},
	}

	addFinishFlags(finishCmd)
	rootCmd.AddCommand(finishCmd)
}

// detectBranchTypeAndName detects type and name from current branch
func detectBranchTypeAndName() (string, string, error) {
	cfg, err := config.LoadConfig()
	if err != nil {
		return "", "", err
	}
	currentBranch, err := git.GetCurrentBranch()
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
		return "", "", fmt.Errorf("current branch '%s' is not a valid topic branch (use explicit command, e.g., git flow feature finish)", currentBranch)
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
	cfg, err := config.LoadConfig()
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