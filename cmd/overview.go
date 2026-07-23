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

// overviewCmd represents the overview command
var overviewCmd = &cobra.Command{
	Use:   "overview",
	Short: "Show an overview of the git-flow configuration and branches",
	Args:  cobra.NoArgs,
	Long: `Show an overview of the git-flow configuration and branches.
This command displays the current git-flow configuration and lists all active topic branches.`,
	Run: func(cmd *cobra.Command, args []string) {
		OverviewCommand()
	},
}

// OverviewCommand is the implementation of the overview command
func OverviewCommand() {
	if err := overview(); err != nil {
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

// overview performs the actual overview logic and returns any errors
func overview() error {
	// Validate that git-flow is initialized
	initialized, err := config.IsInitialized()
	if err != nil {
		return &errors.GitError{Operation: "check if git-flow is initialized", Err: err}
	}
	if !initialized {
		return &errors.NotInitializedError{}
	}

	// Get configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		return &errors.GitError{Operation: "load configuration", Err: err}
	}

	// Print base branches section with condensed format
	fmt.Println("Base branches:")
	fmt.Println("==============")

	// Find and sort base branches
	var trunkBranches []string
	var childBranches []string

	for name, branch := range cfg.Branches {
		if branch.Type == string(config.BranchTypeBase) {
			if branch.Parent == "" {
				trunkBranches = append(trunkBranches, name)
			} else {
				childBranches = append(childBranches, name)
			}
		}
	}

	// Print trunk branches
	for _, name := range trunkBranches {
		fmt.Printf("  %s (root)\n", name)
	}

	// Print child base branches with condensed info
	for _, name := range childBranches {
		branch := cfg.Branches[name]
		fmt.Printf("  %s → %s", name, branch.Parent)
		if branch.AutoUpdate {
			fmt.Printf(" [auto-update]")
		}
		fmt.Println()
	}
	fmt.Println()

	// Print topic branch configurations with condensed format
	fmt.Println("Topic branch types:")
	fmt.Println("===================")

	// Collect and sort topic branches
	var topicTypes []string
	for name, branch := range cfg.Branches {
		if branch.Type == string(config.BranchTypeTopic) {
			topicTypes = append(topicTypes, name)
		}
	}

	for _, name := range topicTypes {
		branch := cfg.Branches[name]

		// Use prefix with wildcard as title
		fmt.Printf("%s*:\n", branch.Prefix)

		// Parent (always show)
		fmt.Printf("  Parent: %s\n", branch.Parent)

		// Start point (only if different from parent)
		if branch.StartPoint != "" && branch.StartPoint != branch.Parent {
			fmt.Printf("  Start point: %s\n", branch.StartPoint)
		}

		// Merge strategies with directional description
		upstreamDesc := getMergeStrategyDescription(branch.UpstreamStrategy, true)
		downstreamDesc := getMergeStrategyDescription(branch.DownstreamStrategy, false)
		fmt.Printf("  %s into %s, %s from %s\n", upstreamDesc, branch.Parent, downstreamDesc, branch.Parent)

		// Tag creation (only if enabled)
		if branch.Tag {
			fmt.Println("  Creates tags on finish")
		}

		fmt.Println()
	}

	// Get all branches for active topic branches section
	branches, err := git.ListBranches()
	if err != nil {
		return &errors.GitError{Operation: "list branches", Err: err}
	}

	// Get current branch
	currentBranch, err := git.GetCurrentBranch()
	if err != nil {
		return &errors.GitError{Operation: "get current branch", Err: err}
	}

	// Print active topic branches
	fmt.Println("Active topic branches:")
	fmt.Println("======================")

	// Collect all active topic branches
	var activeTopicBranches []string
	branchTypeMap := make(map[string]string)

	for _, branchName := range branches {
		for name, branch := range cfg.Branches {
			if branch.Type == string(config.BranchTypeTopic) && strings.HasPrefix(branchName, branch.Prefix) {
				activeTopicBranches = append(activeTopicBranches, branchName)
				branchTypeMap[branchName] = name
				break
			}
		}
	}

	// Print active topic branches
	if len(activeTopicBranches) > 0 {
		for _, branchName := range activeTopicBranches {
			prefix := ""
			if branchName == currentBranch {
				prefix = "* "
			} else {
				prefix = "  "
			}

			branchType := branchTypeMap[branchName]
			fmt.Printf("%s%s (%s)\n", prefix, branchName, branchType)
		}
	} else {
		fmt.Println("  No active topic branches")
	}

	return nil
}

// getMergeStrategyDescription returns a human-readable description of the merge strategy
func getMergeStrategyDescription(strategy string, isUpstream bool) string {
	switch strategy {
	case "merge":
		if isUpstream {
			return "Merges"
		}
		return "Merges"
	case "rebase":
		if isUpstream {
			return "Rebases onto parent then merges"
		}
		return "Rebases onto"
	case "squash":
		if isUpstream {
			return "Squashes and merges"
		}
		return "Squash-merges"
	default:
		return strategy
	}
}

func init() {
	rootCmd.AddCommand(overviewCmd)
}
