package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/gittower/git-flow-next/config"
	"github.com/gittower/git-flow-next/errors"
	"github.com/gittower/git-flow-next/git"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize git-flow in a repository",
	Long: `Initialize git-flow in a repository.
This will set up the necessary configuration for git-flow to work.
If git-flow-avh configuration exists, it will be imported.`,
	Run: func(cmd *cobra.Command, args []string) {
		useDefaults, _ := cmd.Flags().GetBool("defaults")
		noCreateBranches, _ := cmd.Flags().GetBool("no-create-branches")
		mainBranch, _ := cmd.Flags().GetString("main")
		developBranch, _ := cmd.Flags().GetString("develop")
		featurePrefix, _ := cmd.Flags().GetString("feature")
		bugfixPrefix, _ := cmd.Flags().GetString("bugfix")
		releasePrefix, _ := cmd.Flags().GetString("release")
		hotfixPrefix, _ := cmd.Flags().GetString("hotfix")
		supportPrefix, _ := cmd.Flags().GetString("support")
		tagPrefix, _ := cmd.Flags().GetString("tag")
		InitCommand(useDefaults, !noCreateBranches, mainBranch, developBranch, featurePrefix, bugfixPrefix, releasePrefix, hotfixPrefix, supportPrefix, tagPrefix)
	},
}

// InitCommand is the implementation of the init command
func InitCommand(useDefaults, createBranches bool, mainBranch, developBranch, featurePrefix, bugfixPrefix, releasePrefix, hotfixPrefix, supportPrefix, tagPrefix string) {
	if err := initFlow(useDefaults, createBranches, mainBranch, developBranch, featurePrefix, bugfixPrefix, releasePrefix, hotfixPrefix, supportPrefix, tagPrefix); err != nil {
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

// initFlow performs the actual initialization logic and returns any errors
func initFlow(useDefaults, createBranches bool, mainBranch, developBranch, featurePrefix, bugfixPrefix, releasePrefix, hotfixPrefix, supportPrefix, tagPrefix string) error {
	// Check if we're in a git repo
	if !git.IsGitRepo() {
		return &errors.GitError{Operation: "check if git repository", Err: fmt.Errorf("not a git repository. Please run 'git init' first")}
	}

	var cfg *config.Config

	// Check if git-flow-avh config exists
	if config.CheckGitFlowAVHConfig() {
		fmt.Println("Found existing git-flow-avh configuration, importing...")
		var err error
		cfg, err = config.ImportGitFlowAVHConfig()
		if err != nil {
			return &errors.GitError{Operation: "import git-flow-avh configuration", Err: err}
		}
		fmt.Println("Successfully imported git-flow-avh configuration")
	} else {
		// Start with default config
		message := "Initializing git-flow"
		if useDefaults {
			message += " with default settings"
		}
		fmt.Println(message)
		cfg = config.DefaultConfig()
	}

	// Collect overrides from command line flags
	overrides := config.ConfigOverrides{
		MainBranch:    mainBranch,
		DevelopBranch: developBranch,
		FeaturePrefix: featurePrefix,
		BugfixPrefix:  bugfixPrefix,
		ReleasePrefix: releasePrefix,
		HotfixPrefix:  hotfixPrefix,
		SupportPrefix: supportPrefix,
		TagPrefix:     tagPrefix,
	}

	// Apply overrides if provided or if using defaults
	if useDefaults || mainBranch != "" || developBranch != "" || featurePrefix != "" || bugfixPrefix != "" || releasePrefix != "" || hotfixPrefix != "" || supportPrefix != "" || tagPrefix != "" {
		cfg = config.ApplyOverrides(cfg, overrides)
	} else {
		// Otherwise, prompt for input
		interactiveOverrides := interactiveConfig()
		cfg = config.ApplyOverrides(cfg, interactiveOverrides)
	}

	// Save configuration
	if err := config.SaveConfig(cfg); err != nil {
		return &errors.GitError{Operation: "save configuration", Err: err}
	}

	// Mark the repository as initialized
	if err := config.MarkRepoInitialized(); err != nil {
		return &errors.GitError{Operation: "mark repository as initialized", Err: err}
	}

	// Create branches if requested
	if createBranches {
		if err := createGitFlowBranches(cfg); err != nil {
			return &errors.GitError{Operation: "create branches", Err: err}
		}
	}

	fmt.Println("Git flow has been initialized")
	return nil
}

// createGitFlowBranches creates the base branches if they don't exist
func createGitFlowBranches(cfg *config.Config) error {
	// Find base branches
	var mainBranch, developBranch string
	for name, branch := range cfg.Branches {
		if branch.Type == string(config.BranchTypeBase) {
			if branch.Parent == "" {
				mainBranch = name
			} else {
				developBranch = name
			}
		}
	}

	// Check if we have any commits
	hasCommits, err := git.HasCommits()
	if err != nil {
		return fmt.Errorf("failed to check if repository has commits: %w", err)
	}

	// Get current branch if we have commits
	var currentBranch string
	if hasCommits {
		currentBranch, err = git.GetCurrentBranch()
		if err != nil {
			return fmt.Errorf("failed to get current branch: %w", err)
		}
	}

	// Create main branch if it doesn't exist
	if err := git.BranchExists(mainBranch); err != nil {
		// Create main branch
		err = git.CreateBranch(mainBranch, "")
		if err != nil {
			return &errors.GitError{Operation: fmt.Sprintf("create main branch '%s'", mainBranch), Err: err}
		}
		fmt.Printf("Created branch '%s'\n", mainBranch)
	}

	// Create develop branch if it doesn't exist
	if err := git.BranchExists(developBranch); err != nil {
		// Create develop branch from main
		err = git.CreateBranch(developBranch, mainBranch)
		if err != nil {
			return &errors.GitError{Operation: fmt.Sprintf("create develop branch '%s'", developBranch), Err: err}
		}
		fmt.Printf("Created branch '%s'\n", developBranch)
	}

	// Return to original branch if we had one
	if currentBranch != "" && currentBranch != mainBranch && currentBranch != developBranch {
		err = git.Checkout(currentBranch)
		if err != nil {
			return fmt.Errorf("failed to checkout original branch '%s': %w", currentBranch, err)
		}
	}

	return nil
}

// interactiveConfig prompts the user for configuration values
func interactiveConfig() config.ConfigOverrides {
	reader := bufio.NewReader(os.Stdin)
	overrides := config.ConfigOverrides{}

	// Prompt for main branch name
	fmt.Print("Branch name for production releases [main]: ")
	mainBranch, _ := reader.ReadString('\n')
	mainBranch = strings.TrimSpace(mainBranch)
	if mainBranch != "" {
		overrides.MainBranch = mainBranch
	}

	// Prompt for develop branch name
	fmt.Print("Branch name for development [develop]: ")
	developBranch, _ := reader.ReadString('\n')
	developBranch = strings.TrimSpace(developBranch)
	if developBranch != "" {
		overrides.DevelopBranch = developBranch
	}

	// Prompt for feature branch prefix
	fmt.Print("Feature branch prefix [feature/]: ")
	featurePrefix, _ := reader.ReadString('\n')
	featurePrefix = strings.TrimSpace(featurePrefix)
	if featurePrefix != "" {
		if !strings.HasSuffix(featurePrefix, "/") {
			featurePrefix += "/"
		}
		overrides.FeaturePrefix = featurePrefix
	}

	// Prompt for bugfix branch prefix
	fmt.Print("Bugfix branch prefix [bugfix/]: ")
	bugfixPrefix, _ := reader.ReadString('\n')
	bugfixPrefix = strings.TrimSpace(bugfixPrefix)
	if bugfixPrefix != "" {
		if !strings.HasSuffix(bugfixPrefix, "/") {
			bugfixPrefix += "/"
		}
		overrides.BugfixPrefix = bugfixPrefix
	}

	// Prompt for release branch prefix
	fmt.Print("Release branch prefix [release/]: ")
	releasePrefix, _ := reader.ReadString('\n')
	releasePrefix = strings.TrimSpace(releasePrefix)
	if releasePrefix != "" {
		if !strings.HasSuffix(releasePrefix, "/") {
			releasePrefix += "/"
		}
		overrides.ReleasePrefix = releasePrefix
	}

	// Prompt for hotfix branch prefix
	fmt.Print("Hotfix branch prefix [hotfix/]: ")
	hotfixPrefix, _ := reader.ReadString('\n')
	hotfixPrefix = strings.TrimSpace(hotfixPrefix)
	if hotfixPrefix != "" {
		if !strings.HasSuffix(hotfixPrefix, "/") {
			hotfixPrefix += "/"
		}
		overrides.HotfixPrefix = hotfixPrefix
	}

	// Prompt for support branch prefix
	fmt.Print("Support branch prefix [support/]: ")
	supportPrefix, _ := reader.ReadString('\n')
	supportPrefix = strings.TrimSpace(supportPrefix)
	if supportPrefix != "" {
		if !strings.HasSuffix(supportPrefix, "/") {
			supportPrefix += "/"
		}
		overrides.SupportPrefix = supportPrefix
	}

	// Prompt for version tag prefix
	fmt.Print("Version tag prefix [v]: ")
	tagPrefix, _ := reader.ReadString('\n')
	tagPrefix = strings.TrimSpace(tagPrefix)
	if tagPrefix != "" {
		overrides.TagPrefix = tagPrefix
	}

	return overrides
}

func init() {
	rootCmd.AddCommand(initCmd)

	// Add flags specific to init command
	initCmd.Flags().BoolP("defaults", "d", false, "Use default branch naming conventions")
	initCmd.Flags().Bool("no-create-branches", false, "Don't create branches even if they don't exist")
	initCmd.Flags().StringP("main", "m", "", "Main branch name")
	initCmd.Flags().StringP("develop", "e", "", "Develop branch name")
	initCmd.Flags().StringP("feature", "p", "", "Feature branch prefix")
	initCmd.Flags().StringP("bugfix", "b", "", "Bugfix branch prefix")
	initCmd.Flags().StringP("release", "r", "", "Release branch prefix")
	initCmd.Flags().StringP("hotfix", "x", "", "Hotfix branch prefix")
	initCmd.Flags().StringP("support", "s", "", "Support branch prefix")
	initCmd.Flags().StringP("tag", "t", "", "Version tag prefix")
}
