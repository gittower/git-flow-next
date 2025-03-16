package cmd

import (
	"github.com/spf13/cobra"
)

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
	Run: func(cmd *cobra.Command, args []string) {
		// If no subcommand is provided, print help
		cmd.Help()
	},
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
