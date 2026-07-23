package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Version information
var (
	Version   = "1.1.0"
	BuildDate = "unknown"
	GitCommit = "unknown"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Args:  cobra.NoArgs,
	Long:  `Display version information for git-flow-next.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("git-flow-next version %s\n", Version)
		if BuildDate != "unknown" {
			fmt.Printf("Build date: %s\n", BuildDate)
		}
		if GitCommit != "unknown" {
			fmt.Printf("Git commit: %s\n", GitCommit)
		}
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
