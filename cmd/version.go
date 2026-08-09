package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Version information
var (
	Version   = "1.2.0"
	BuildDate = "unknown"
	GitCommit = "unknown"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Args:  cobra.NoArgs,
	Long:  `Display version information for git-flow-next.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Version number first, then a parenthesized edition marker: the same
		// shape git-flow-avh uses ("1.12.3 (AVH Edition)"), so tooling that
		// parses the first whitespace-separated token works unchanged.
		fmt.Printf("%s (git-flow-next)\n", Version)
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
