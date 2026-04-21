package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/gittower/git-flow-next/internal/git"
	"github.com/spf13/cobra"
)

// sharedGitFlowFile is the standard filename for a shared git-flow configuration
// stored in the repository root and committed alongside project code.
const sharedGitFlowFile = ".gitflow"

// sharedGitFlowIncludePath is the include.path value used in .git/config to load
// the shared .gitflow file. It is relative to .git/config (one directory up = repo root).
const sharedGitFlowIncludePath = "../.gitflow"

var rootCmd = &cobra.Command{
	Use:   "git-flow",
	Short: "git-flow-next is a modern reimplementation of git-flow",
	Long: `git-flow-next is a modern reimplementation of git-flow in Go that offers 
greater flexibility while maintaining backward compatibility with git-flow-avh.

It provides a set of commands to work with Git branches according to the git-flow model.`,
	Example: `  git flow init
  git flow feature start my-feature
  git flow feature finish my-feature
  git flow release start 1.0.0
  git flow release finish 1.0.0`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Skip for the init command — it handles its own .gitflow detection and wiring.
		if cmd.Name() == "init" {
			return
		}
		autoWireSharedGitFlowFile()
	},
	Run: func(cmd *cobra.Command, args []string) {
		// If no subcommand is provided, print help
		cmd.Help()
	},
}

// autoWireSharedGitFlowFile checks whether a .gitflow file exists in the repository
// root and, if it is not yet referenced in the local git config via include.path,
// adds the include automatically. This lets developers clone a repository and start
// using git-flow commands immediately without running git flow init.
func autoWireSharedGitFlowFile() {
	if !git.IsGitRepo() {
		return
	}
	topLevel, err := git.GetRepoTopLevel()
	if err != nil {
		return
	}
	gitflowPath := filepath.Join(topLevel, sharedGitFlowFile)
	if _, err := os.Stat(gitflowPath); os.IsNotExist(err) {
		return
	}

	// Check whether include.path already points to ../.gitflow in local config.
	includes, err := git.GetLocalConfigAllValues("include.path")
	if err != nil {
		return
	}
	for _, inc := range includes {
		if inc == sharedGitFlowIncludePath {
			return // already wired — nothing to do
		}
	}

	// Wire it up.
	if err := git.AddLocalConfigValue("include.path", sharedGitFlowIncludePath); err != nil {
		return
	}
	fmt.Fprintf(os.Stderr, "Note: Found %s in repository root, auto-configured git-flow.\n", sharedGitFlowFile)
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Enable verbose output")
}
