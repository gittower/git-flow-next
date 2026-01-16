package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/gittower/git-flow-next/internal/config"
	"github.com/gittower/git-flow-next/internal/errors"
	"github.com/gittower/git-flow-next/internal/git"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize git-flow in a repository",
	Long: `Initialize git-flow in a repository.
This will set up the necessary configuration for git-flow to work.

You can use presets for common workflows:
  --preset=classic    Traditional GitFlow with main, develop, feature, release, hotfix
  --preset=github     GitHub Flow with main and feature branches
  --preset=gitlab     GitLab Flow with production, staging, main, feature, and hotfix

Use --custom for interactive custom configuration.
If git-flow-avh configuration exists, it will be imported.`,
	Run: func(cmd *cobra.Command, args []string) {
		useDefaults, _ := cmd.Flags().GetBool("defaults")
		noCreateBranches, _ := cmd.Flags().GetBool("no-create-branches")
		preset, _ := cmd.Flags().GetString("preset")
		custom, _ := cmd.Flags().GetBool("custom")
		mainBranch, _ := cmd.Flags().GetString("main")
		developBranch, _ := cmd.Flags().GetString("develop")
		featurePrefix, _ := cmd.Flags().GetString("feature")
		bugfixPrefix, _ := cmd.Flags().GetString("bugfix")
		releasePrefix, _ := cmd.Flags().GetString("release")
		hotfixPrefix, _ := cmd.Flags().GetString("hotfix")
		supportPrefix, _ := cmd.Flags().GetString("support")
		tagPrefix, _ := cmd.Flags().GetString("tag")
		force, _ := cmd.Flags().GetBool("force")
		noForce, _ := cmd.Flags().GetBool("no-force")
		forceFlag := getBoolFlagInit(force, noForce)
		InitCommand(useDefaults, !noCreateBranches, preset, custom, mainBranch, developBranch, featurePrefix, bugfixPrefix, releasePrefix, hotfixPrefix, supportPrefix, tagPrefix, forceFlag)
	},
}

// InitCommand is the implementation of the init command
func InitCommand(useDefaults, createBranches bool, preset string, custom bool, mainBranch, developBranch, featurePrefix, bugfixPrefix, releasePrefix, hotfixPrefix, supportPrefix, tagPrefix string, force *bool) {
	if err := initFlow(useDefaults, createBranches, preset, custom, mainBranch, developBranch, featurePrefix, bugfixPrefix, releasePrefix, hotfixPrefix, supportPrefix, tagPrefix, force); err != nil {
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
func initFlow(useDefaults, createBranches bool, preset string, custom bool, mainBranch, developBranch, featurePrefix, bugfixPrefix, releasePrefix, hotfixPrefix, supportPrefix, tagPrefix string, force *bool) error {
	// Check if we're in a git repo
	if !git.IsGitRepo() {
		return &errors.GitError{Operation: "check if git repository", Err: fmt.Errorf("not a git repository. Please run 'git init' first")}
	}

	// Determine if force is enabled (default: false)
	forceReinit := false
	if force != nil {
		forceReinit = *force
	}

	// Check if any configuration options are provided
	hasConfigFlags := mainBranch != "" || developBranch != "" || featurePrefix != "" || bugfixPrefix != "" || releasePrefix != "" || hotfixPrefix != "" || supportPrefix != "" || tagPrefix != ""

	// Check if already initialized
	initialized, err := config.IsInitialized()
	if err != nil {
		return &errors.GitError{Operation: "check if git-flow is initialized", Err: err}
	}

	if initialized {
		// Special case: AVH import should work without --force (migration, not reconfiguration)
		isAVHImport := config.CheckGitFlowAVHConfig() && preset == "" && !custom &&
			!useDefaults && !hasConfigFlags

		if !forceReinit && !isAVHImport {
			return &errors.AlreadyInitializedError{}
		}

		if forceReinit {
			fmt.Fprintln(os.Stderr, "Warning: Reinitializing git-flow. Existing configuration will be overwritten.")
		}
	}

	var cfg *config.Config

	// Check if git-flow-avh config exists and no explicit options are provided
	if config.CheckGitFlowAVHConfig() && preset == "" && !custom && !useDefaults && !hasConfigFlags {
		fmt.Println("Found existing git-flow-avh configuration, importing...")
		var err error
		cfg, err = config.ImportGitFlowAVHConfig()
		if err != nil {
			return &errors.GitError{Operation: "import git-flow-avh configuration", Err: err}
		}
		fmt.Println("Successfully imported git-flow-avh configuration")
	} else {
		// Determine configuration method
		if preset != "" {
			// Use preset configuration
			fmt.Printf("Initializing git-flow with %s preset\n", preset)
			presetType := config.PresetType(preset)
			cfg = config.PresetConfig(presetType)
		} else if custom {
			// Use custom configuration
			fmt.Println("Initializing git-flow with custom configuration")
			cfg = customConfiguration()
		} else if useDefaults {
			// Use default configuration
			fmt.Println("Initializing git-flow with default settings")
			cfg = config.DefaultConfig()
		} else if hasConfigFlags {
			// Use default configuration with command-line overrides
			fmt.Println("Initializing git-flow")
			cfg = config.DefaultConfig()
		} else {
			// Interactive mode - use legacy interactive config for backward compatibility
			cfg = config.DefaultConfig()
			interactiveOverrides := interactiveConfig()
			cfg = config.ApplyOverrides(cfg, interactiveOverrides)
		}
	}

	// Apply command-line overrides
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

	// Apply overrides if any were provided
	if mainBranch != "" || developBranch != "" || featurePrefix != "" || bugfixPrefix != "" || releasePrefix != "" || hotfixPrefix != "" || supportPrefix != "" || tagPrefix != "" {
		cfg = config.ApplyOverrides(cfg, overrides)
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

	// Collect base branches that need to be created
	type branchToCreate struct {
		name   string
		parent string
	}
	var toCreate []branchToCreate
	for name, branch := range cfg.Branches {
		if branch.Type == string(config.BranchTypeBase) && git.BranchExists(name) != nil {
			toCreate = append(toCreate, branchToCreate{name: name, parent: branch.Parent})
		}
	}

	// Sort branches topologically: parents before children
	sorted := make([]branchToCreate, 0, len(toCreate))
	added := make(map[string]bool)
	for len(sorted) < len(toCreate) {
		progress := false
		for _, b := range toCreate {
			if added[b.name] {
				continue
			}
			// Add if: no parent, parent already exists in git, or parent already in sorted list
			parentReady := b.parent == "" || git.BranchExists(b.parent) == nil || added[b.parent]
			if parentReady {
				sorted = append(sorted, b)
				added[b.name] = true
				progress = true
			}
		}
		if !progress {
			break // Prevent infinite loop on circular dependencies
		}
	}

	// Create branches in dependency order
	for _, b := range sorted {
		err := git.CreateBranch(b.name, b.parent)
		if err != nil {
			return &errors.GitError{Operation: fmt.Sprintf("create base branch '%s'", b.name), Err: err}
		}
		fmt.Printf("Created branch '%s'\n", b.name)
	}

	// Return to original branch if we had one and it still exists
	if currentBranch != "" {
		branchStillExists := false
		for name, branch := range cfg.Branches {
			if branch.Type == string(config.BranchTypeBase) && name == currentBranch {
				branchStillExists = true
				break
			}
		}
		if !branchStillExists {
			err = git.Checkout(currentBranch)
			if err != nil {
				return fmt.Errorf("failed to checkout original branch '%s': %w", currentBranch, err)
			}
		}
	}

	return nil
}

// interactiveInitialization prompts the user to choose initialization method
func interactiveInitialization() *config.Config {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("? Choose initialization method:")
	fmt.Println("  1. Use preset workflow")
	fmt.Println("  2. Custom configuration")
	fmt.Print("Enter your choice (1-2): ")

	choice, _ := reader.ReadString('\n')
	choice = strings.TrimSpace(choice)

	switch choice {
	case "1":
		return interactivePresetSelection()
	case "2":
		return customConfiguration()
	default:
		fmt.Println("Invalid choice, using preset workflow")
		return interactivePresetSelection()
	}
}

// interactivePresetSelection prompts the user to choose a preset
func interactivePresetSelection() *config.Config {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println()
	fmt.Println("? Choose a preset:")
	fmt.Println("  1. Classic GitFlow (main, develop, feature, release, hotfix)")
	fmt.Println("  2. GitHub Flow (main, feature)")
	fmt.Println("  3. GitLab Flow (production, staging, main, feature, hotfix)")
	fmt.Print("Enter your choice (1-3): ")

	choice, _ := reader.ReadString('\n')
	choice = strings.TrimSpace(choice)

	var preset config.PresetType
	switch choice {
	case "2":
		preset = config.PresetGitHub
		fmt.Println("✓ Selected GitHub Flow preset")
	case "3":
		preset = config.PresetGitLab
		fmt.Println("✓ Selected GitLab Flow preset")
	default:
		preset = config.PresetClassic
		fmt.Println("✓ Selected Classic GitFlow preset")
	}

	cfg := config.PresetConfig(preset)

	// Allow customization of branch names and prefixes
	fmt.Println()
	fmt.Println("You can customize branch names and prefixes (press Enter for defaults):")

	overrides := config.ConfigOverrides{}

	// Customize based on preset type
	if preset == config.PresetClassic {
		overrides = interactiveClassicCustomization()
	} else if preset == config.PresetGitHub {
		overrides = interactiveGitHubCustomization()
	} else if preset == config.PresetGitLab {
		overrides = interactiveGitLabCustomization()
	}

	return config.ApplyOverrides(cfg, overrides)
}

// customConfiguration provides custom configuration flow
func customConfiguration() *config.Config {
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("? What's your trunk branch (holds production code)? [main] ")
	trunkBranch, _ := reader.ReadString('\n')
	trunkBranch = strings.TrimSpace(trunkBranch)
	if trunkBranch == "" {
		trunkBranch = "main"
	}

	fmt.Printf("✓ Trunk branch: %s\n", trunkBranch)
	fmt.Println()
	fmt.Println("Configuration commands:")
	fmt.Println("  git-flow config add base <name> [<parent>] [options...]")
	fmt.Println("  git-flow config add topic <name> <parent> [options...]")
	fmt.Println("  git-flow config rename base <old-name> <new-name>")
	fmt.Println("  git-flow config rename topic <old-name> <new-name>")
	fmt.Println("  git-flow config edit base <name> [options...]")
	fmt.Println("  git-flow config edit topic <name> [options...]")
	fmt.Println("  git-flow config delete base <name>")
	fmt.Println("  git-flow config delete topic <name>")
	fmt.Println("  git-flow config list")
	fmt.Println()
	fmt.Println("Use these commands to configure your workflow after initialization.")

	// Create minimal config with just the trunk branch
	cfg := &config.Config{
		Version:       "1.0",
		Remote:        "origin",
		CommandConfig: make(map[string]string),
		Branches: map[string]config.BranchConfig{
			trunkBranch: {
				Type:               string(config.BranchTypeBase),
				Parent:             "",
				UpstreamStrategy:   string(config.MergeStrategyNone),
				DownstreamStrategy: string(config.MergeStrategyNone),
				AutoUpdate:         false,
			},
		},
	}

	return cfg
}

// interactiveClassicCustomization allows customization of Classic GitFlow preset
func interactiveClassicCustomization() config.ConfigOverrides {
	reader := bufio.NewReader(os.Stdin)
	overrides := config.ConfigOverrides{}

	fmt.Print("? Main branch name [main]: ")
	mainBranch, _ := reader.ReadString('\n')
	mainBranch = strings.TrimSpace(mainBranch)
	if mainBranch != "" {
		overrides.MainBranch = mainBranch
	}

	fmt.Print("? Develop branch name [develop]: ")
	developBranch, _ := reader.ReadString('\n')
	developBranch = strings.TrimSpace(developBranch)
	if developBranch != "" {
		overrides.DevelopBranch = developBranch
	}

	fmt.Print("? Feature prefix [feature/]: ")
	featurePrefix, _ := reader.ReadString('\n')
	featurePrefix = strings.TrimSpace(featurePrefix)
	if featurePrefix != "" {
		if !strings.HasSuffix(featurePrefix, "/") {
			featurePrefix += "/"
		}
		overrides.FeaturePrefix = featurePrefix
	}

	fmt.Print("? Release prefix [release/]: ")
	releasePrefix, _ := reader.ReadString('\n')
	releasePrefix = strings.TrimSpace(releasePrefix)
	if releasePrefix != "" {
		if !strings.HasSuffix(releasePrefix, "/") {
			releasePrefix += "/"
		}
		overrides.ReleasePrefix = releasePrefix
	}

	fmt.Print("? Hotfix prefix [hotfix/]: ")
	hotfixPrefix, _ := reader.ReadString('\n')
	hotfixPrefix = strings.TrimSpace(hotfixPrefix)
	if hotfixPrefix != "" {
		if !strings.HasSuffix(hotfixPrefix, "/") {
			hotfixPrefix += "/"
		}
		overrides.HotfixPrefix = hotfixPrefix
	}

	fmt.Print("? Version tag prefix [v]: ")
	tagPrefix, _ := reader.ReadString('\n')
	tagPrefix = strings.TrimSpace(tagPrefix)
	if tagPrefix == "" {
		tagPrefix = "v" // Default to "v" if empty
	}
	overrides.TagPrefix = tagPrefix

	return overrides
}

// interactiveGitHubCustomization allows customization of GitHub Flow preset
func interactiveGitHubCustomization() config.ConfigOverrides {
	reader := bufio.NewReader(os.Stdin)
	overrides := config.ConfigOverrides{}

	fmt.Print("? Main branch name [main]: ")
	mainBranch, _ := reader.ReadString('\n')
	mainBranch = strings.TrimSpace(mainBranch)
	if mainBranch != "" {
		overrides.MainBranch = mainBranch
	}

	fmt.Print("? Feature prefix [feature/]: ")
	featurePrefix, _ := reader.ReadString('\n')
	featurePrefix = strings.TrimSpace(featurePrefix)
	if featurePrefix != "" {
		if !strings.HasSuffix(featurePrefix, "/") {
			featurePrefix += "/"
		}
		overrides.FeaturePrefix = featurePrefix
	}

	return overrides
}

// interactiveGitLabCustomization allows customization of GitLab Flow preset
func interactiveGitLabCustomization() config.ConfigOverrides {
	reader := bufio.NewReader(os.Stdin)
	overrides := config.ConfigOverrides{}

	fmt.Print("? Production branch name [production]: ")
	productionBranch, _ := reader.ReadString('\n')
	productionBranch = strings.TrimSpace(productionBranch)
	if productionBranch != "" {
		overrides.ProductionBranch = productionBranch
	}

	fmt.Print("? Staging branch name [staging]: ")
	stagingBranch, _ := reader.ReadString('\n')
	stagingBranch = strings.TrimSpace(stagingBranch)
	if stagingBranch != "" {
		overrides.StagingBranch = stagingBranch
	}

	fmt.Print("? Main branch name [main]: ")
	mainBranch, _ := reader.ReadString('\n')
	mainBranch = strings.TrimSpace(mainBranch)
	if mainBranch != "" {
		overrides.MainBranch = mainBranch
	}

	fmt.Print("? Feature prefix [feature/]: ")
	featurePrefix, _ := reader.ReadString('\n')
	featurePrefix = strings.TrimSpace(featurePrefix)
	if featurePrefix != "" {
		if !strings.HasSuffix(featurePrefix, "/") {
			featurePrefix += "/"
		}
		overrides.FeaturePrefix = featurePrefix
	}

	fmt.Print("? Hotfix prefix [hotfix/]: ")
	hotfixPrefix, _ := reader.ReadString('\n')
	hotfixPrefix = strings.TrimSpace(hotfixPrefix)
	if hotfixPrefix != "" {
		if !strings.HasSuffix(hotfixPrefix, "/") {
			hotfixPrefix += "/"
		}
		overrides.HotfixPrefix = hotfixPrefix
	}

	return overrides
}

// interactiveConfig prompts the user for configuration values (legacy function)
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
	if tagPrefix == "" {
		tagPrefix = "v" // Default to "v" if empty
	}
	overrides.TagPrefix = tagPrefix

	return overrides
}

// getBoolFlagInit converts two opposite boolean flags into a single *bool value
func getBoolFlagInit(positive, negative bool) *bool {
	if positive {
		return &positive
	}
	if negative {
		falseBool := false
		return &falseBool
	}
	return nil
}

func init() {
	rootCmd.AddCommand(initCmd)

	// Add flags specific to init command
	initCmd.Flags().BoolP("defaults", "d", false, "Use default branch naming conventions")
	initCmd.Flags().Bool("no-create-branches", false, "Don't create branches even if they don't exist")
	initCmd.Flags().StringP("preset", "p", "", "Use preset configuration (classic|github|gitlab)")
	initCmd.Flags().Bool("custom", false, "Use custom configuration with interactive setup")
	initCmd.Flags().StringP("main", "m", "", "Main branch name")
	initCmd.Flags().StringP("develop", "e", "", "Develop branch name")
	initCmd.Flags().String("feature", "", "Feature branch prefix")
	initCmd.Flags().StringP("bugfix", "b", "", "Bugfix branch prefix")
	initCmd.Flags().StringP("release", "r", "", "Release branch prefix")
	initCmd.Flags().StringP("hotfix", "x", "", "Hotfix branch prefix")
	initCmd.Flags().StringP("support", "s", "", "Support branch prefix")
	initCmd.Flags().StringP("tag", "t", "", "Version tag prefix")
	initCmd.Flags().BoolP("force", "f", false, "Force reinitialization of an already initialized repository")
	initCmd.Flags().Bool("no-force", false, "Don't force reinitialize (default behavior)")
}
