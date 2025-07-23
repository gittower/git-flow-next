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

	// Get all branches
	branches, err := git.ListBranches()
	if err != nil {
		return &errors.GitError{Operation: "list branches", Err: err}
	}

	// Get current branch
	currentBranch, err := git.GetCurrentBranch()
	if err != nil {
		return &errors.GitError{Operation: "get current branch", Err: err}
	}

	// Print base branches section
	fmt.Println("Base branches:")
	fmt.Println("=============")

	// Find base branches and sort them (develop first, then main)
	var baseBranches []string
	baseParentMap := make(map[string]string)

	for name, branch := range cfg.Branches {
		if branch.Type == string(config.BranchTypeBase) {
			baseBranches = append(baseBranches, name)
			baseParentMap[name] = branch.Parent
		}
	}

	// Print base branches with their relationships
	for _, name := range baseBranches {
		parent := baseParentMap[name]
		if parent == "" {
			parent = "(root)"
		}

		fmt.Printf("  %s -> %s\n", name, parent)

		// Add merge strategy information
		branch := cfg.Branches[name]
		if parent == "(root)" {
			fmt.Println("    Upstream: none, Downstream: none")
		} else {
			fmt.Printf("    Upstream: %s, Downstream: %s\n",
				branch.UpstreamStrategy,
				branch.DownstreamStrategy)
		}
	}
	fmt.Println()

	// Print topic branch configurations
	fmt.Println("Topic branch configurations:")
	fmt.Println("==========================")

	// Process topic branches
	for name, branch := range cfg.Branches {
		if branch.Type == string(config.BranchTypeTopic) {
			// Get parent branch
			parent := branch.Parent
			if parent == "" {
				parent = "develop" // Default parent
			}

			// Get start point
			startPoint := branch.StartPoint
			if startPoint == "" {
				startPoint = parent // Default start point
			}

			// Print topic branch configuration
			fmt.Printf("%s:\n", name)
			fmt.Printf("    Parent: %s\n", parent)
			fmt.Printf("    Start Point: %s\n", startPoint)
			fmt.Printf("    Prefix: %s\n", branch.Prefix)

			// Add merge strategy information based on configuration
			fmt.Printf("    Upstream: %s, Downstream: %s\n",
				branch.UpstreamStrategy,
				branch.DownstreamStrategy)

			// Add tag information if enabled
			if branch.Tag && branch.TagPrefix != "" {
				fmt.Printf("    Tag prefix: %s\n", branch.TagPrefix)
			}
		}
	}
	fmt.Println()

	// Print active topic branches
	fmt.Println("Active topic branches:")
	fmt.Println("====================")

	// Collect all topic branches
	var topicBranches []string
	branchTypeMap := make(map[string]string)

	for _, branchName := range branches {
		for name, branch := range cfg.Branches {
			if branch.Type == string(config.BranchTypeTopic) && strings.HasPrefix(branchName, branch.Prefix) {
				topicBranches = append(topicBranches, branchName)
				branchTypeMap[branchName] = name
				break
			}
		}
	}

	// Print active topic branches
	if len(topicBranches) > 0 {
		for _, branchName := range topicBranches {
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

func init() {
	rootCmd.AddCommand(overviewCmd)
}
