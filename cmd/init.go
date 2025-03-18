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
		createBranches, _ := cmd.Flags().GetBool("create-branches")
		InitCommand(useDefaults, createBranches)
	},
}

// InitCommand is the implementation of the init command
func InitCommand(useDefaults, createBranches bool) {
	if err := initFlow(useDefaults, createBranches); err != nil {
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
func initFlow(useDefaults, createBranches bool) error {
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
	} else if useDefaults {
		fmt.Println("Initializing git-flow with default settings")
		cfg = config.DefaultConfig()
	} else {
		fmt.Println("Initializing git-flow")
		cfg = interactiveConfig()
	}

	// Save the configuration
	err := config.SaveConfig(cfg)
	if err != nil {
		return &errors.GitError{Operation: "save configuration", Err: err}
	}

	// Mark the repository as initialized
	err = config.MarkRepoInitialized()
	if err != nil {
		return &errors.GitError{Operation: "mark repository as initialized", Err: err}
	}

	// Create branches if requested or if interactive mode and user confirms
	shouldCreateBranches := createBranches
	if !useDefaults && !createBranches {
		// In interactive mode, ask if branches should be created
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Do you want to create branches now? [y/N]: ")
		answer, _ := reader.ReadString('\n')
		answer = strings.TrimSpace(strings.ToLower(answer))
		shouldCreateBranches = answer == "y" || answer == "yes"
	}

	if shouldCreateBranches {
		err = createGitFlowBranches(cfg)
		if err != nil {
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
	if mainBranch != "" && !git.BranchExists(mainBranch) {
		if !hasCommits {
			// Create an initial commit
			err = git.CreateInitialCommit(mainBranch)
			if err != nil {
				return fmt.Errorf("failed to create initial commit: %w", err)
			}
			fmt.Printf("Created branch '%s' with initial commit\n", mainBranch)

			// Update current branch
			currentBranch = mainBranch
		} else {
			// Create branch from current branch
			err = git.CreateBranch(mainBranch, currentBranch)
			if err != nil {
				return fmt.Errorf("failed to create branch '%s': %w", mainBranch, err)
			}
			fmt.Printf("Created branch '%s'\n", mainBranch)
		}
	}

	// Create develop branch if it doesn't exist
	if developBranch != "" && !git.BranchExists(developBranch) {
		// Create branch from main branch
		err = git.CreateBranch(developBranch, mainBranch)
		if err != nil {
			return fmt.Errorf("failed to create branch '%s': %w", developBranch, err)
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
func interactiveConfig() *config.Config {
	// Start with default config
	cfg := config.DefaultConfig()

	// Create a reader for user input
	reader := bufio.NewReader(os.Stdin)

	// Prompt for main branch name
	fmt.Print("Branch name for production releases [main]: ")
	mainBranch, _ := reader.ReadString('\n')
	mainBranch = strings.TrimSpace(mainBranch)
	if mainBranch == "" {
		mainBranch = "main"
	}

	// Prompt for develop branch name
	fmt.Print("Branch name for development [develop]: ")
	developBranch, _ := reader.ReadString('\n')
	developBranch = strings.TrimSpace(developBranch)
	if developBranch == "" {
		developBranch = "develop"
	}

	// Prompt for feature prefix
	fmt.Print("Feature branch prefix [feature/]: ")
	featurePrefix, _ := reader.ReadString('\n')
	featurePrefix = strings.TrimSpace(featurePrefix)
	if featurePrefix == "" {
		featurePrefix = "feature/"
	} else if !strings.HasSuffix(featurePrefix, "/") {
		featurePrefix += "/"
	}

	// Prompt for release prefix
	fmt.Print("Release branch prefix [release/]: ")
	releasePrefix, _ := reader.ReadString('\n')
	releasePrefix = strings.TrimSpace(releasePrefix)
	if releasePrefix == "" {
		releasePrefix = "release/"
	} else if !strings.HasSuffix(releasePrefix, "/") {
		releasePrefix += "/"
	}

	// Prompt for hotfix prefix
	fmt.Print("Hotfix branch prefix [hotfix/]: ")
	hotfixPrefix, _ := reader.ReadString('\n')
	hotfixPrefix = strings.TrimSpace(hotfixPrefix)
	if hotfixPrefix == "" {
		hotfixPrefix = "hotfix/"
	} else if !strings.HasSuffix(hotfixPrefix, "/") {
		hotfixPrefix += "/"
	}

	// Prompt for support prefix
	fmt.Print("Support branch prefix [support/]: ")
	supportPrefix, _ := reader.ReadString('\n')
	supportPrefix = strings.TrimSpace(supportPrefix)
	if supportPrefix == "" {
		supportPrefix = "support/"
	} else if !strings.HasSuffix(supportPrefix, "/") {
		supportPrefix += "/"
	}

	// Update config with user input
	if mainBranch != "main" {
		// Create a new main branch config
		mainConfig := cfg.Branches["main"]
		delete(cfg.Branches, "main")
		cfg.Branches[mainBranch] = mainConfig

		// Update parent references
		for name, branch := range cfg.Branches {
			if branch.Parent == "main" {
				branch.Parent = mainBranch
				cfg.Branches[name] = branch
			}
			if branch.StartPoint == "main" {
				branch.StartPoint = mainBranch
				cfg.Branches[name] = branch
			}
		}
	}

	if developBranch != "develop" {
		// Create a new develop branch config
		developConfig := cfg.Branches["develop"]
		developConfig.Parent = mainBranch
		delete(cfg.Branches, "develop")
		cfg.Branches[developBranch] = developConfig

		// Update parent references
		for name, branch := range cfg.Branches {
			if branch.Parent == "develop" {
				branch.Parent = developBranch
				cfg.Branches[name] = branch
			}
			if branch.StartPoint == "develop" {
				branch.StartPoint = developBranch
				cfg.Branches[name] = branch
			}
		}
	} else {
		// Update develop parent to main branch
		developConfig := cfg.Branches["develop"]
		developConfig.Parent = mainBranch
		cfg.Branches["develop"] = developConfig
	}

	// Update prefixes
	featureConfig := cfg.Branches["feature"]
	featureConfig.Prefix = featurePrefix
	cfg.Branches["feature"] = featureConfig

	releaseConfig := cfg.Branches["release"]
	releaseConfig.Prefix = releasePrefix
	cfg.Branches["release"] = releaseConfig

	hotfixConfig := cfg.Branches["hotfix"]
	hotfixConfig.Prefix = hotfixPrefix
	cfg.Branches["hotfix"] = hotfixConfig

	supportConfig := cfg.Branches["support"]
	supportConfig.Prefix = supportPrefix
	cfg.Branches["support"] = supportConfig

	return cfg
}

func init() {
	rootCmd.AddCommand(initCmd)

	// Add flags specific to init command
	initCmd.Flags().BoolP("defaults", "d", false, "Use default branch naming conventions")
	initCmd.Flags().BoolP("create-branches", "c", false, "Create branches if they don't exist")
}
