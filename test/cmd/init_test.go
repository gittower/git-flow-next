package cmd_test

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gittower/git-flow-next/internal/config"
	"github.com/gittower/git-flow-next/test/testutil"
)

// setupGitFlowAVH sets up git-flow-avh configuration in the test repository
func setupGitFlowAVH(t *testing.T, dir string) {
	// Set git-flow-avh configuration
	cmd := exec.Command("git", "config", "gitflow.branch.master", "main")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to set gitflow.branch.master: %v", err)
	}

	cmd = exec.Command("git", "config", "gitflow.branch.develop", "dev")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to set gitflow.branch.develop: %v", err)
	}

	cmd = exec.Command("git", "config", "gitflow.prefix.feature", "feat/")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to set gitflow.prefix.feature: %v", err)
	}

	cmd = exec.Command("git", "config", "gitflow.prefix.release", "rel/")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to set gitflow.prefix.release: %v", err)
	}

	cmd = exec.Command("git", "config", "gitflow.prefix.hotfix", "fix/")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to set gitflow.prefix.hotfix: %v", err)
	}
}

// runGitFlow runs the git-flow command with the given arguments
func runGitFlow(t *testing.T, dir string, args ...string) (string, error) {
	// Get path to the git-flow binary
	gitFlowPath, err := filepath.Abs(filepath.Join("..", "..", "git-flow"))
	if err != nil {
		t.Fatalf("Failed to get absolute path to git-flow: %v", err)
	}

	// Run the git-flow command
	cmd := exec.Command(gitFlowPath, args...)
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()

	// Return the combined output
	return stdout.String() + stderr.String(), err
}

// runGitFlowWithInput runs the git-flow command with the given arguments and input
func runGitFlowWithInput(t *testing.T, dir string, input string, args ...string) (string, error) {
	// Get path to the git-flow binary
	gitFlowPath, err := filepath.Abs(filepath.Join("..", "..", "git-flow"))
	if err != nil {
		t.Fatalf("Failed to get absolute path to git-flow: %v", err)
	}

	// Run the git-flow command
	cmd := exec.Command(gitFlowPath, args...)
	cmd.Dir = dir

	// Set up input
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatalf("Failed to get stdin pipe: %v", err)
	}

	// Set up output
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Start the command
	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start command: %v", err)
	}

	// Write input
	io.WriteString(stdin, input)
	stdin.Close()

	// Wait for the command to finish
	err = cmd.Wait()

	// Return the combined output
	return stdout.String() + stderr.String(), err
}

// getGitConfig gets the Git configuration value for the given key
func getGitConfig(t *testing.T, dir string, key string) string {
	cmd := exec.Command("git", "config", "--get", key)
	cmd.Dir = dir
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	err := cmd.Run()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(stdout.String())
}

// getGitConfigWithScope gets a Git config value at a specific scope
func getGitConfigWithScope(t *testing.T, dir string, key string, scope string) string {
	args := []string{"config", "--" + scope, "--get", key}
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	err := cmd.Run()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(stdout.String())
}

// getGitConfigFromFile reads a config value from a specific file
func getGitConfigFromFile(t *testing.T, filePath string, key string) string {
	cmd := exec.Command("git", "config", "--file", filePath, "--get", key)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	err := cmd.Run()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(stdout.String())
}

// runGitFlowWithEnv runs git-flow with custom environment variables.
func runGitFlowWithEnv(t *testing.T, dir string, env []string, args ...string) (string, error) {
	gitFlowPath, err := filepath.Abs(filepath.Join("..", "..", "git-flow"))
	if err != nil {
		t.Fatalf("Failed to get absolute path to git-flow: %v", err)
	}
	cmd := exec.Command(gitFlowPath, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), env...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()
	return stdout.String() + stderr.String(), err
}

// branchExists checks if a branch exists in the repository
func branchExists(t *testing.T, dir string, branch string) bool {
	cmd := exec.Command("git", "show-ref", "--verify", "--quiet", "refs/heads/"+branch)
	cmd.Dir = dir
	err := cmd.Run()
	return err == nil
}

// TestInitWithDefaults tests the init command with --defaults flag
func TestInitWithDefaults(t *testing.T) {
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Run git-flow init --defaults
	output, err := runGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to run git-flow init --defaults: %v\nOutput: %s", err, output)
	}

	// Check if the output contains the expected message
	if !strings.Contains(output, "Initializing git-flow with default settings") {
		t.Errorf("Expected output to contain 'Initializing git-flow with default settings', got: %s", output)
	}

	// Check if the configuration was saved correctly
	version := getGitConfig(t, dir, "gitflow.version")
	if version != "1.0" {
		t.Errorf("Expected gitflow.version to be '1.0', got: %s", version)
	}

	// Check if the branch configurations were saved correctly
	mainType := getGitConfig(t, dir, "gitflow.branch.main.type")
	if mainType != "base" {
		t.Errorf("Expected gitflow.branch.main.type to be 'base', got: %s", mainType)
	}

	developParent := getGitConfig(t, dir, "gitflow.branch.develop.parent")
	if developParent != "main" {
		t.Errorf("Expected gitflow.branch.develop.parent to be 'main', got: %s", developParent)
	}

	featurePrefix := getGitConfig(t, dir, "gitflow.branch.feature.prefix")
	if featurePrefix != "feature/" {
		t.Errorf("Expected gitflow.branch.feature.prefix to be 'feature/', got: %s", featurePrefix)
	}

	// Check if tag configuration was set correctly for release and hotfix branches
	releaseTag := getGitConfig(t, dir, "gitflow.branch.release.tag")
	if releaseTag != "true" {
		t.Errorf("Expected gitflow.branch.release.tag to be 'true', got: %s", releaseTag)
	}

	releaseTagPrefix := getGitConfig(t, dir, "gitflow.branch.release.tagprefix")
	if releaseTagPrefix != "" {
		t.Errorf("Expected gitflow.branch.release.tagprefix to be empty, got: %s", releaseTagPrefix)
	}

	hotfixTag := getGitConfig(t, dir, "gitflow.branch.hotfix.tag")
	if hotfixTag != "true" {
		t.Errorf("Expected gitflow.branch.hotfix.tag to be 'true', got: %s", hotfixTag)
	}

	hotfixTagPrefix := getGitConfig(t, dir, "gitflow.branch.hotfix.tagprefix")
	if hotfixTagPrefix != "" {
		t.Errorf("Expected gitflow.branch.hotfix.tagprefix to be empty, got: %s", hotfixTagPrefix)
	}
}

// TestInitWithAVHConfig tests the init command with existing git-flow-avh configuration
func TestInitWithAVHConfig(t *testing.T) {
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Setup git-flow-avh configuration
	setupGitFlowAVH(t, dir)

	// Add tag configuration to git-flow-avh setup
	cmd := exec.Command("git", "config", "gitflow.prefix.versiontag", "ver-")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to set gitflow.prefix.versiontag: %v", err)
	}

	// Run git-flow init
	output, err := runGitFlow(t, dir, "init")
	if err != nil {
		t.Fatalf("Failed to run git-flow init: %v\nOutput: %s", err, output)
	}

	// Check if the output contains the expected message
	if !strings.Contains(output, "Found existing git-flow-avh configuration, importing") {
		t.Errorf("Expected output to contain 'Found existing git-flow-avh configuration, importing', got: %s", output)
	}

	// Check if the configuration was saved correctly
	version := getGitConfig(t, dir, "gitflow.version")
	if version != "1.0" {
		t.Errorf("Expected gitflow.version to be '1.0', got: %s", version)
	}

	// Check if the branch configurations were imported correctly
	mainType := getGitConfig(t, dir, "gitflow.branch.main.type")
	if mainType != "base" {
		t.Errorf("Expected gitflow.branch.main.type to be 'base', got: %s", mainType)
	}

	// Check if the old configuration is still there
	masterBranch := getGitConfig(t, dir, "gitflow.branch.master")
	if masterBranch != "main" {
		t.Errorf("Expected gitflow.branch.master to be 'main', got: %s", masterBranch)
	}

	// Check if the prefixes were imported correctly
	featurePrefix := getGitConfig(t, dir, "gitflow.branch.feature.prefix")
	if featurePrefix != "feat/" {
		t.Errorf("Expected gitflow.branch.feature.prefix to be 'feat/', got: %s", featurePrefix)
	}

	// Check if the tag configuration was imported correctly
	releaseTag := getGitConfig(t, dir, "gitflow.branch.release.tag")
	if releaseTag != "true" {
		t.Errorf("Expected gitflow.branch.release.tag to be 'true', got: %s", releaseTag)
	}

	releaseTagPrefix := getGitConfig(t, dir, "gitflow.branch.release.tagprefix")
	if releaseTagPrefix != "ver-" {
		t.Errorf("Expected gitflow.branch.release.tagprefix to be 'ver-', got: %s", releaseTagPrefix)
	}

	hotfixTag := getGitConfig(t, dir, "gitflow.branch.hotfix.tag")
	if hotfixTag != "true" {
		t.Errorf("Expected gitflow.branch.hotfix.tag to be 'true', got: %s", hotfixTag)
	}

	hotfixTagPrefix := getGitConfig(t, dir, "gitflow.branch.hotfix.tagprefix")
	if hotfixTagPrefix != "ver-" {
		t.Errorf("Expected gitflow.branch.hotfix.tagprefix to be 'ver-', got: %s", hotfixTagPrefix)
	}
}

// TestInitInteractive tests the interactive init command
func TestInitInteractive(t *testing.T) {
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Run git-flow init with input
	input := "custom-main\ncustom-dev\nf/\nr/\nh/\ns/\n"
	output, err := runGitFlowWithInput(t, dir, input, "init")
	if err != nil {
		t.Fatalf("Failed to run git-flow init: %v\nOutput: %s", err, output)
	}

	// Check if the output contains the expected prompts
	if !strings.Contains(output, "Branch name for production releases") {
		t.Errorf("Expected output to contain 'Branch name for production releases', got: %s", output)
	}

	// Check if the configuration was saved correctly
	version := getGitConfig(t, dir, "gitflow.version")
	if version != "1.0" {
		t.Errorf("Expected gitflow.version to be '1.0', got: %s", version)
	}

	// Check if the branch configurations were saved correctly
	mainName := getGitConfig(t, dir, "gitflow.branch.custom-main.type")
	if mainName != "base" {
		t.Errorf("Expected gitflow.branch.custom-main.type to be 'base', got: %s", mainName)
	}

	developName := getGitConfig(t, dir, "gitflow.branch.custom-dev.parent")
	if developName != "custom-main" {
		t.Errorf("Expected gitflow.branch.custom-dev.parent to be 'custom-main', got: %s", developName)
	}

	featurePrefix := getGitConfig(t, dir, "gitflow.branch.feature.prefix")
	if featurePrefix != "f/" {
		t.Errorf("Expected gitflow.branch.feature.prefix to be 'f/', got: %s", featurePrefix)
	}
}

// TestInitWithBranchCreation tests the init command with branch creation
func TestInitWithBranchCreation(t *testing.T) {
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Run git-flow init --defaults
	output, err := runGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to run git-flow init --defaults: %v\nOutput: %s", err, output)
	}

	// Check if the output contains the expected message
	if !strings.Contains(output, "Initializing git-flow with default settings") {
		t.Errorf("Expected output to contain 'Initializing git-flow with default settings', got: %s", output)
	}

	// Check if branches were created
	if !branchExists(t, dir, "main") {
		t.Errorf("Expected 'main' branch to exist")
	}

	if !branchExists(t, dir, "develop") {
		t.Errorf("Expected 'develop' branch to exist")
	}
}

// TestInitInteractiveWithBranchCreation tests the init command with interactive input and branch creation
func TestInitInteractiveWithBranchCreation(t *testing.T) {
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Interactive input for init prompts:
	// main branch, develop branch, feature prefix, bugfix prefix, release prefix, hotfix prefix, support prefix, tag prefix
	input := "custom-main\ncustom-dev\nfeature/\nbugfix/\nrelease/\nhotfix/\nsupport/\n\n"

	// Run git-flow init with interactive input
	output, err := runGitFlowWithInput(t, dir, input, "init")
	if err != nil {
		t.Fatalf("Failed to run git-flow init: %v\nOutput: %s", err, output)
	}

	// Check if the branches were actually created
	if !branchExists(t, dir, "custom-main") {
		t.Errorf("Expected 'custom-main' branch to exist")
	}

	if !branchExists(t, dir, "custom-dev") {
		t.Errorf("Expected 'custom-dev' branch to exist")
	}
}

// TestInitInteractiveDefaultTagPrefixIsEmpty tests that pressing Enter for tag prefix
// in interactive mode results in an empty tag prefix (matching git-flow-avh behavior).
// Steps:
// 1. Sets up a test repository
// 2. Runs 'git flow init' with interactive input, pressing Enter for all prompts
// 3. Verifies gitflow.branch.release.tagprefix is empty (not written to config)
// 4. Verifies gitflow.branch.hotfix.tagprefix is empty (not written to config)
func TestInitInteractiveDefaultTagPrefixIsEmpty(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// All defaults: 8 empty lines (main, develop, feature, bugfix, release, hotfix, support, tag prefix)
	input := "\n\n\n\n\n\n\n\n"

	output, err := runGitFlowWithInput(t, dir, input, "init")
	if err != nil {
		t.Fatalf("Failed to run git-flow init: %v\nOutput: %s", err, output)
	}

	// Verify tag prefix is empty for release
	releaseTagPrefix := getGitConfig(t, dir, "gitflow.branch.release.tagprefix")
	if releaseTagPrefix != "" {
		t.Errorf("Expected gitflow.branch.release.tagprefix to be empty, got: %s", releaseTagPrefix)
	}

	// Verify tag prefix is empty for hotfix
	hotfixTagPrefix := getGitConfig(t, dir, "gitflow.branch.hotfix.tagprefix")
	if hotfixTagPrefix != "" {
		t.Errorf("Expected gitflow.branch.hotfix.tagprefix to be empty, got: %s", hotfixTagPrefix)
	}
}

// TestInitInteractiveExplicitTagPrefix tests that typing an explicit tag prefix
// in interactive mode sets it correctly.
// Steps:
// 1. Sets up a test repository
// 2. Runs 'git flow init' with interactive input, accepting defaults for all fields
//    except tag prefix which is set to 'v'
// 3. Verifies gitflow.branch.release.tagprefix is 'v'
// 4. Verifies gitflow.branch.hotfix.tagprefix is 'v'
func TestInitInteractiveExplicitTagPrefix(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// 7 empty lines for defaults, then 'v' for tag prefix
	input := "\n\n\n\n\n\n\nv\n"

	output, err := runGitFlowWithInput(t, dir, input, "init")
	if err != nil {
		t.Fatalf("Failed to run git-flow init: %v\nOutput: %s", err, output)
	}

	// Verify tag prefix is 'v' for release
	releaseTagPrefix := getGitConfig(t, dir, "gitflow.branch.release.tagprefix")
	if releaseTagPrefix != "v" {
		t.Errorf("Expected gitflow.branch.release.tagprefix to be 'v', got: %s", releaseTagPrefix)
	}

	// Verify tag prefix is 'v' for hotfix
	hotfixTagPrefix := getGitConfig(t, dir, "gitflow.branch.hotfix.tagprefix")
	if hotfixTagPrefix != "v" {
		t.Errorf("Expected gitflow.branch.hotfix.tagprefix to be 'v', got: %s", hotfixTagPrefix)
	}
}

// TestInitWithFlags tests the init command with custom branch prefixes
func TestInitWithFlags(t *testing.T) {
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Run git-flow init with custom prefixes and base branch names
	output, err := runGitFlow(t, dir, "init",
		"--main", "custom-main",
		"--develop", "custom-dev",
		"--feature", "feat/",
		"--bugfix", "bug/",
		"--release", "rel/",
		"--hotfix", "fix/",
		"--support", "sup/",
		"--tag", "v")
	if err != nil {
		t.Fatalf("Failed to run git-flow init with flags: %v\nOutput: %s", err, output)
	}

	// Check if the output contains the expected message
	if !strings.Contains(output, "Initializing git-flow") {
		t.Errorf("Expected output to contain 'Initializing git-flow', got: %s", output)
	}

	// Check if the configuration was saved correctly
	version := getGitConfig(t, dir, "gitflow.version")
	if version != "1.0" {
		t.Errorf("Expected gitflow.version to be '1.0', got: %s", version)
	}

	// Check if the base branch configurations were saved correctly
	mainType := getGitConfig(t, dir, "gitflow.branch.custom-main.type")
	if mainType != "base" {
		t.Errorf("Expected gitflow.branch.custom-main.type to be 'base', got: %s", mainType)
	}

	developParent := getGitConfig(t, dir, "gitflow.branch.custom-dev.parent")
	if developParent != "custom-main" {
		t.Errorf("Expected gitflow.branch.custom-dev.parent to be 'custom-main', got: %s", developParent)
	}

	// Check if the branch configurations were saved correctly
	featurePrefix := getGitConfig(t, dir, "gitflow.branch.feature.prefix")
	if featurePrefix != "feat/" {
		t.Errorf("Expected gitflow.branch.feature.prefix to be 'feat/', got: %s", featurePrefix)
	}

	bugfixPrefix := getGitConfig(t, dir, "gitflow.branch.bugfix.prefix")
	if bugfixPrefix != "bug/" {
		t.Errorf("Expected gitflow.branch.bugfix.prefix to be 'bug/', got: %s", bugfixPrefix)
	}

	releasePrefix := getGitConfig(t, dir, "gitflow.branch.release.prefix")
	if releasePrefix != "rel/" {
		t.Errorf("Expected gitflow.branch.release.prefix to be 'rel/', got: %s", releasePrefix)
	}

	hotfixPrefix := getGitConfig(t, dir, "gitflow.branch.hotfix.prefix")
	if hotfixPrefix != "fix/" {
		t.Errorf("Expected gitflow.branch.hotfix.prefix to be 'fix/', got: %s", hotfixPrefix)
	}

	supportPrefix := getGitConfig(t, dir, "gitflow.branch.support.prefix")
	if supportPrefix != "sup/" {
		t.Errorf("Expected gitflow.branch.support.prefix to be 'sup/', got: %s", supportPrefix)
	}

	// Check if tag configuration was set correctly
	releaseTagPrefix := getGitConfig(t, dir, "gitflow.branch.release.tagprefix")
	if releaseTagPrefix != "v" {
		t.Errorf("Expected gitflow.branch.release.tagprefix to be 'v', got: %s", releaseTagPrefix)
	}

	hotfixTagPrefix := getGitConfig(t, dir, "gitflow.branch.hotfix.tagprefix")
	if hotfixTagPrefix != "v" {
		t.Errorf("Expected gitflow.branch.hotfix.tagprefix to be 'v', got: %s", hotfixTagPrefix)
	}
}

// TestInitWithFlagsAndBranches tests the init command with custom prefixes and branch creation
func TestInitWithFlagsAndBranches(t *testing.T) {
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Run git-flow init with custom prefixes and branch creation
	output, err := runGitFlow(t, dir, "init",
		"--feature", "feat/",
		"--bugfix", "bug/",
		"--release", "rel/",
		"--hotfix", "fix/",
		"--support", "sup/",
		"--tag", "v",
	)
	if err != nil {
		t.Fatalf("Failed to run git-flow init with flags and branch creation: %v\nOutput: %s", err, output)
	}

	// Check if the output contains the expected messages
	if !strings.Contains(output, "Initializing git-flow") {
		t.Errorf("Expected output to contain 'Initializing git-flow', got: %s", output)
	}

	// Check if the branches were created
	if !branchExists(t, dir, "main") {
		t.Error("Expected main branch to exist")
	}
	if !branchExists(t, dir, "develop") {
		t.Error("Expected develop branch to exist")
	}

	// Check if the configuration was saved correctly
	featurePrefix := getGitConfig(t, dir, "gitflow.branch.feature.prefix")
	if featurePrefix != "feat/" {
		t.Errorf("Expected gitflow.branch.feature.prefix to be 'feat/', got: %s", featurePrefix)
	}
}

// TestInitWithDefaultsAndOverrides tests initializing with defaults but overriding specific branch configs
func TestInitWithDefaultsAndOverrides(t *testing.T) {
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults but override specific configs
	output, err := runGitFlow(t, dir, "init", "--defaults",
		"--main", "custom-main",
		"--develop", "custom-dev",
		"--feature", "f/",
		"--release", "r/",
		"--hotfix", "h/",
		"--support", "s/",
		"--tag", "v")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Verify branches were created
	if !branchExists(t, dir, "custom-main") {
		t.Error("Expected 'custom-main' branch to exist")
	}
	if !branchExists(t, dir, "custom-dev") {
		t.Error("Expected 'custom-dev' branch to exist")
	}

	// Change to the test directory before loading config
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(oldDir)
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Failed to change to test directory: %v", err)
	}

	// Verify configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load configuration: %v", err)
	}

	// Check main branch config
	if _, ok := cfg.Branches["custom-main"]; !ok {
		t.Error("Expected 'custom-main' branch configuration to exist")
	}

	// Check develop branch config
	if developCfg, ok := cfg.Branches["custom-dev"]; ok {
		if developCfg.Parent != "custom-main" {
			t.Errorf("Expected develop branch parent to be 'custom-main', got '%s'", developCfg.Parent)
		}
	} else {
		t.Error("Expected 'custom-dev' branch configuration to exist")
	}

	// Check feature branch config
	if featureCfg, ok := cfg.Branches["feature"]; ok {
		if featureCfg.Prefix != "f/" {
			t.Errorf("Expected feature branch prefix to be 'f/', got '%s'", featureCfg.Prefix)
		}
	} else {
		t.Error("Expected 'feature' branch configuration to exist")
	}

	// Check release branch config
	if releaseCfg, ok := cfg.Branches["release"]; ok {
		if releaseCfg.Prefix != "r/" {
			t.Errorf("Expected release branch prefix to be 'r/', got '%s'", releaseCfg.Prefix)
		}
		if releaseCfg.TagPrefix != "v" {
			t.Errorf("Expected release tag prefix to be 'v', got '%s'", releaseCfg.TagPrefix)
		}
	} else {
		t.Error("Expected 'release' branch configuration to exist")
	}

	// Check hotfix branch config
	if hotfixCfg, ok := cfg.Branches["hotfix"]; ok {
		if hotfixCfg.Prefix != "h/" {
			t.Errorf("Expected hotfix branch prefix to be 'h/', got '%s'", hotfixCfg.Prefix)
		}
		if hotfixCfg.TagPrefix != "v" {
			t.Errorf("Expected hotfix tag prefix to be 'v', got '%s'", hotfixCfg.TagPrefix)
		}
	} else {
		t.Error("Expected 'hotfix' branch configuration to exist")
	}
}

// TestInitWithLocalScope tests that --local flag stores config in the repository's .git/config.
// Steps:
// 1. Sets up a test repository
// 2. Runs 'git flow init --defaults --local'
// 3. Verifies gitflow.version is stored in local config (.git/config)
// 4. Verifies gitflow.branch.main.type is stored in local config
func TestInitWithLocalScope(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	output, err := runGitFlow(t, dir, "init", "--defaults", "--local")
	if err != nil {
		t.Fatalf("Failed to run git-flow init --defaults --local: %v\nOutput: %s", err, output)
	}

	// Verify config is in local scope
	version := getGitConfigWithScope(t, dir, "gitflow.version", "local")
	if version != "1.0" {
		t.Errorf("Expected gitflow.version in local scope to be '1.0', got: %s", version)
	}

	mainType := getGitConfigWithScope(t, dir, "gitflow.branch.main.type", "local")
	if mainType != "base" {
		t.Errorf("Expected gitflow.branch.main.type in local scope to be 'base', got: %s", mainType)
	}
}

// TestInitWithGlobalScope tests that --global flag stores config in the global git config.
// Steps:
// 1. Sets up a test repository with isolated GIT_CONFIG_GLOBAL
// 2. Runs 'git flow init --defaults --global' with isolated global config
// 3. Verifies gitflow.version is stored in global config
// 4. Verifies gitflow.version is NOT stored in local config
func TestInitWithGlobalScope(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Create isolated global config file
	globalConfigFile := filepath.Join(t.TempDir(), "gitconfig-global")
	env := []string{"GIT_CONFIG_GLOBAL=" + globalConfigFile}

	output, err := runGitFlowWithEnv(t, dir, env, "init", "--defaults", "--global")
	if err != nil {
		t.Fatalf("Failed to run git-flow init --defaults --global: %v\nOutput: %s", err, output)
	}

	// Verify config is in global scope (read from the isolated file)
	version := getGitConfigFromFile(t, globalConfigFile, "gitflow.version")
	if version != "1.0" {
		t.Errorf("Expected gitflow.version in global config to be '1.0', got: %s", version)
	}

	// Verify config is NOT in local scope
	localVersion := getGitConfigWithScope(t, dir, "gitflow.version", "local")
	if localVersion != "" {
		t.Errorf("Expected gitflow.version to NOT be in local config, got: %s", localVersion)
	}
}

// TestInitWithFileScope tests that --file flag stores config in the specified file.
// Steps:
// 1. Sets up a test repository
// 2. Creates a temp file path for config output
// 3. Runs 'git flow init --defaults --file <path>'
// 4. Verifies gitflow.version is stored in the specified file
// 5. Verifies gitflow.version is NOT stored in local config
func TestInitWithFileScope(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	configFile := filepath.Join(t.TempDir(), "custom-gitflow-config")

	output, err := runGitFlow(t, dir, "init", "--defaults", "--file", configFile)
	if err != nil {
		t.Fatalf("Failed to run git-flow init --defaults --file: %v\nOutput: %s", err, output)
	}

	// Verify config is in specified file
	version := getGitConfigFromFile(t, configFile, "gitflow.version")
	if version != "1.0" {
		t.Errorf("Expected gitflow.version in file to be '1.0', got: %s", version)
	}

	// Verify config is NOT in local scope
	localVersion := getGitConfigWithScope(t, dir, "gitflow.version", "local")
	if localVersion != "" {
		t.Errorf("Expected gitflow.version to NOT be in local config, got: %s", localVersion)
	}
}

// TestInitWithMutuallyExclusiveScopeFlags tests that using multiple scope flags produces an error.
// Steps:
// 1. Sets up a test repository
// 2. Runs 'git flow init --defaults --local --global'
// 3. Verifies the command fails
// 4. Verifies the error message mentions mutual exclusivity
func TestInitWithMutuallyExclusiveScopeFlags(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	output, err := runGitFlow(t, dir, "init", "--defaults", "--local", "--global")
	if err == nil {
		t.Fatalf("Expected error when using --local and --global together, but succeeded\nOutput: %s", output)
	}

	if !strings.Contains(output, "cannot use multiple scope options") {
		t.Errorf("Expected error about mutual exclusivity, got: %s", output)
	}
}

// TestInitWithFileNonExistentDirectory tests that --file with non-existent parent directory fails.
// Steps:
// 1. Sets up a test repository
// 2. Runs 'git flow init --defaults --file /nonexistent/dir/config'
// 3. Verifies the command fails with "config file directory does not exist" error
func TestInitWithFileNonExistentDirectory(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	output, err := runGitFlow(t, dir, "init", "--defaults", "--file", "/nonexistent/dir/config")
	if err == nil {
		t.Fatalf("Expected error for non-existent directory, but succeeded\nOutput: %s", output)
	}

	// Error should mention the specific validation message
	if !strings.Contains(output, "config file directory does not exist") {
		t.Errorf("Expected error 'config file directory does not exist', got: %s", output)
	}
}

// TestInitDefaultScopeIsLocal tests that no scope flag defaults to local scope for writes (backward compatibility).
// This verifies the WRITE behavior of default scope. The READ behavior (merged config) is tested by
// TestInitDefaultMergedConfigCheck and TestInitDefaultShowsGlobalSourceMessage.
// Steps:
// 1. Sets up a test repository
// 2. Runs 'git flow init --defaults' without any scope flag
// 3. Verifies gitflow.version is stored in local config (.git/config)
func TestInitDefaultScopeIsLocal(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	output, err := runGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to run git-flow init --defaults: %v\nOutput: %s", err, output)
	}

	// Verify config is in local scope
	version := getGitConfigWithScope(t, dir, "gitflow.version", "local")
	if version != "1.0" {
		t.Errorf("Expected gitflow.version in local scope to be '1.0', got: %s", version)
	}
}

// TestInitForceWithGlobalScope tests that --force works with --global scope flag.
// Steps:
// 1. Sets up a test repository with isolated GIT_CONFIG_GLOBAL
// 2. Runs 'git flow init --defaults --global' to initialize
// 3. Runs 'git flow init --defaults --force --global' to reinitialize
// 4. Verifies the reconfiguration succeeds and config is in global scope
func TestInitForceWithGlobalScope(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	globalConfigFile := filepath.Join(t.TempDir(), "gitconfig-global")
	env := []string{"GIT_CONFIG_GLOBAL=" + globalConfigFile}

	// First initialization
	output, err := runGitFlowWithEnv(t, dir, env, "init", "--defaults", "--global")
	if err != nil {
		t.Fatalf("Failed initial git-flow init: %v\nOutput: %s", err, output)
	}

	// Second initialization with --force
	output, err = runGitFlowWithEnv(t, dir, env, "init", "--defaults", "--force", "--global")
	if err != nil {
		t.Fatalf("Failed to run git-flow init --defaults --force --global: %v\nOutput: %s", err, output)
	}

	// Verify global config is updated
	version := getGitConfigFromFile(t, globalConfigFile, "gitflow.version")
	if version != "1.0" {
		t.Errorf("Expected gitflow.version in global config to be '1.0', got: %s", version)
	}
}

// TestInitWithSystemScope tests that --system flag stores config in the system git config.
// Steps:
// 1. Sets up a test repository with isolated GIT_CONFIG_SYSTEM
// 2. Runs 'git flow init --defaults --system' with isolated system config
// 3. Verifies gitflow.version is stored in system config
// 4. Verifies gitflow.version is NOT stored in local config
func TestInitWithSystemScope(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Create isolated system config file
	systemConfigFile := filepath.Join(t.TempDir(), "gitconfig-system")
	env := []string{"GIT_CONFIG_SYSTEM=" + systemConfigFile}

	output, err := runGitFlowWithEnv(t, dir, env, "init", "--defaults", "--system")
	if err != nil {
		t.Fatalf("Failed to run git-flow init --defaults --system: %v\nOutput: %s", err, output)
	}

	// Verify config is in system scope (read from the isolated file)
	version := getGitConfigFromFile(t, systemConfigFile, "gitflow.version")
	if version != "1.0" {
		t.Errorf("Expected gitflow.version in system config to be '1.0', got: %s", version)
	}

	// Verify config is NOT in local scope
	localVersion := getGitConfigWithScope(t, dir, "gitflow.version", "local")
	if localVersion != "" {
		t.Errorf("Expected gitflow.version to NOT be in local config, got: %s", localVersion)
	}
}

// TestInitForceWithFileScope tests that --force works with --file scope flag.
// Steps:
// 1. Sets up a test repository
// 2. Creates a temp file path for config output
// 3. Runs 'git flow init --defaults --file <path>' to initialize
// 4. Runs 'git flow init --defaults --force --file <path>' to reinitialize
// 5. Verifies the reconfiguration succeeds and config is in the specified file
func TestInitForceWithFileScope(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	configFile := filepath.Join(t.TempDir(), "custom-gitflow-config")

	// First initialization
	output, err := runGitFlow(t, dir, "init", "--defaults", "--file", configFile)
	if err != nil {
		t.Fatalf("Failed initial git-flow init: %v\nOutput: %s", err, output)
	}

	// Second initialization with --force
	output, err = runGitFlow(t, dir, "init", "--defaults", "--force", "--file", configFile)
	if err != nil {
		t.Fatalf("Failed to run git-flow init --defaults --force --file: %v\nOutput: %s", err, output)
	}

	// Verify config is in specified file
	version := getGitConfigFromFile(t, configFile, "gitflow.version")
	if version != "1.0" {
		t.Errorf("Expected gitflow.version in file to be '1.0', got: %s", version)
	}
}

// TestInitLocalScopeIgnoresGlobalConfig tests that --local creates local config even when global config exists.
// This is the CORE FIX for issue #8 - explicit scope should check only that scope.
// Steps:
// 1. Sets up a test repository with isolated GIT_CONFIG_GLOBAL
// 2. Initializes git-flow with --global to create global config
// 3. Runs 'git flow init --defaults --local' in the same repo
// 4. Verifies the command succeeds (not "already initialized")
// 5. Verifies config is now in BOTH local and global scopes
func TestInitLocalScopeIgnoresGlobalConfig(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Create isolated global config file
	globalConfigFile := filepath.Join(t.TempDir(), "gitconfig-global")
	env := []string{"GIT_CONFIG_GLOBAL=" + globalConfigFile}

	// First: initialize with --global
	output, err := runGitFlowWithEnv(t, dir, env, "init", "--defaults", "--global")
	if err != nil {
		t.Fatalf("Failed to init with --global: %v\nOutput: %s", err, output)
	}

	// Verify global config exists
	globalVersion := getGitConfigFromFile(t, globalConfigFile, "gitflow.version")
	if globalVersion != "1.0" {
		t.Fatalf("Expected global config to have gitflow.version='1.0', got: %s", globalVersion)
	}

	// Verify local config does NOT exist yet
	localVersion := getGitConfigWithScope(t, dir, "gitflow.version", "local")
	if localVersion != "" {
		t.Fatalf("Expected NO local config before --local init, got: %s", localVersion)
	}

	// Now: initialize with --local (should succeed, not report "already initialized")
	output, err = runGitFlowWithEnv(t, dir, env, "init", "--defaults", "--local")
	if err != nil {
		t.Fatalf("Expected --local init to succeed even with global config present: %v\nOutput: %s", err, output)
	}

	// Verify local config now exists
	localVersion = getGitConfigWithScope(t, dir, "gitflow.version", "local")
	if localVersion != "1.0" {
		t.Errorf("Expected gitflow.version in local config to be '1.0', got: %s", localVersion)
	}
}

// TestInitDefaultShowsGlobalSourceMessage tests that when initialized via global config,
// the message indicates the source and suggests --local.
// Steps:
// 1. Sets up a test repository with isolated GIT_CONFIG_GLOBAL
// 2. Initializes git-flow with --global to create global config
// 3. Runs 'git flow init --defaults' (no scope flag) in the same repo
// 4. Verifies the output mentions "global config" and "--local"
func TestInitDefaultShowsGlobalSourceMessage(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Create isolated global config file
	globalConfigFile := filepath.Join(t.TempDir(), "gitconfig-global")
	env := []string{"GIT_CONFIG_GLOBAL=" + globalConfigFile}

	// First: initialize with --global
	output, err := runGitFlowWithEnv(t, dir, env, "init", "--defaults", "--global")
	if err != nil {
		t.Fatalf("Failed to init with --global: %v\nOutput: %s", err, output)
	}

	// Now: run init with --defaults (no scope flag) - should report already initialized via global
	output, err = runGitFlowWithEnv(t, dir, env, "init", "--defaults")
	if err == nil {
		t.Fatalf("Expected init to fail/report already initialized, but succeeded\nOutput: %s", output)
	}

	// Verify message mentions global config as source
	if !strings.Contains(output, "global config") {
		t.Errorf("Expected message to mention 'global config', got: %s", output)
	}

	// Verify message suggests --local
	if !strings.Contains(output, "--local") {
		t.Errorf("Expected message to suggest '--local', got: %s", output)
	}
}

// TestInitDefaultMergedConfigCheck tests that default (no scope flag) checks merged config.
// Steps:
// 1. Sets up a test repository with isolated GIT_CONFIG_GLOBAL
// 2. Initializes git-flow with --global to create global config
// 3. Runs 'git flow init --defaults' (no scope flag)
// 4. Verifies it reports already initialized (because merged config finds global)
func TestInitDefaultMergedConfigCheck(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Create isolated global config file
	globalConfigFile := filepath.Join(t.TempDir(), "gitconfig-global")
	env := []string{"GIT_CONFIG_GLOBAL=" + globalConfigFile}

	// First: initialize with --global
	output, err := runGitFlowWithEnv(t, dir, env, "init", "--defaults", "--global")
	if err != nil {
		t.Fatalf("Failed to init with --global: %v\nOutput: %s", err, output)
	}

	// Now: run init --defaults (should see global config via merged read)
	output, err = runGitFlowWithEnv(t, dir, env, "init", "--defaults")

	// Should fail because already initialized (found via merged config)
	if err == nil {
		t.Fatalf("Expected init to report already initialized, but succeeded\nOutput: %s", output)
	}

	// The error indicates it found existing config
	if !strings.Contains(strings.ToLower(output), "already") && !strings.Contains(strings.ToLower(output), "configured") {
		t.Errorf("Expected error about already initialized/configured, got: %s", output)
	}
}

// TestInitGlobalScopeIgnoresLocalConfig tests that --global creates global config even when local config exists.
// This is the symmetric test to TestInitLocalScopeIgnoresGlobalConfig, verifying scope isolation works both ways.
// Steps:
// 1. Sets up a test repository with isolated GIT_CONFIG_GLOBAL
// 2. Initializes git-flow with --local to create local config
// 3. Runs 'git flow init --defaults --global' in the same repo
// 4. Verifies the command succeeds (not "already initialized")
// 5. Verifies config is now in BOTH local and global scopes
func TestInitGlobalScopeIgnoresLocalConfig(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Create isolated global config file
	globalConfigFile := filepath.Join(t.TempDir(), "gitconfig-global")
	env := []string{"GIT_CONFIG_GLOBAL=" + globalConfigFile}

	// First: initialize with --local
	output, err := runGitFlowWithEnv(t, dir, env, "init", "--defaults", "--local")
	if err != nil {
		t.Fatalf("Failed to init with --local: %v\nOutput: %s", err, output)
	}

	// Verify local config exists
	localVersion := getGitConfigWithScope(t, dir, "gitflow.version", "local")
	if localVersion != "1.0" {
		t.Fatalf("Expected local config to have gitflow.version='1.0', got: %s", localVersion)
	}

	// Verify global config does NOT exist yet
	globalVersion := getGitConfigFromFile(t, globalConfigFile, "gitflow.version")
	if globalVersion != "" {
		t.Fatalf("Expected NO global config before --global init, got: %s", globalVersion)
	}

	// Now: initialize with --global (should succeed, not report "already initialized")
	output, err = runGitFlowWithEnv(t, dir, env, "init", "--defaults", "--global")
	if err != nil {
		t.Fatalf("Expected --global init to succeed even with local config present: %v\nOutput: %s", err, output)
	}

	// Verify global config now exists
	globalVersion = getGitConfigFromFile(t, globalConfigFile, "gitflow.version")
	if globalVersion != "1.0" {
		t.Errorf("Expected gitflow.version in global config to be '1.0', got: %s", globalVersion)
	}

	// Verify local config still exists (not overwritten)
	localVersion = getGitConfigWithScope(t, dir, "gitflow.version", "local")
	if localVersion != "1.0" {
		t.Errorf("Expected gitflow.version in local config to still be '1.0', got: %s", localVersion)
	}
}

// TestInitWithFileRelativePath tests that --file works with relative paths.
// Steps:
// 1. Sets up a test repository
// 2. Runs 'git flow init --defaults --file ./custom-config' with a relative path
// 3. Verifies gitflow.version is stored in the file at the relative path
// 4. Verifies the file is created in the expected location (repo directory)
func TestInitWithFileRelativePath(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Use a relative path
	relativeConfigPath := "./custom-gitflow-config"
	expectedAbsPath := filepath.Join(dir, "custom-gitflow-config")

	output, err := runGitFlow(t, dir, "init", "--defaults", "--file", relativeConfigPath)
	if err != nil {
		t.Fatalf("Failed to run git-flow init --defaults --file with relative path: %v\nOutput: %s", err, output)
	}

	// Verify the file was created at the expected absolute path
	if _, err := os.Stat(expectedAbsPath); os.IsNotExist(err) {
		t.Fatalf("Expected config file to exist at %s, but it does not", expectedAbsPath)
	}

	// Verify config is in specified file
	version := getGitConfigFromFile(t, expectedAbsPath, "gitflow.version")
	if version != "1.0" {
		t.Errorf("Expected gitflow.version in file to be '1.0', got: %s", version)
	}

	// Verify config is NOT in local scope
	localVersion := getGitConfigWithScope(t, dir, "gitflow.version", "local")
	if localVersion != "" {
		t.Errorf("Expected gitflow.version to NOT be in local config, got: %s", localVersion)
	}
}

// TestInitForceWithLocalScope tests that --force works with --local scope flag.
// Steps:
// 1. Sets up a test repository
// 2. Runs 'git flow init --defaults --local' to initialize
// 3. Runs 'git flow init --defaults --force --local' to reinitialize
// 4. Verifies the reconfiguration succeeds and config is in local scope
func TestInitForceWithLocalScope(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// First initialization
	output, err := runGitFlow(t, dir, "init", "--defaults", "--local")
	if err != nil {
		t.Fatalf("Failed initial git-flow init: %v\nOutput: %s", err, output)
	}

	// Verify config exists
	version := getGitConfigWithScope(t, dir, "gitflow.version", "local")
	if version != "1.0" {
		t.Fatalf("Expected gitflow.version in local config to be '1.0', got: %s", version)
	}

	// Second initialization with --force
	output, err = runGitFlow(t, dir, "init", "--defaults", "--force", "--local")
	if err != nil {
		t.Fatalf("Failed to run git-flow init --defaults --force --local: %v\nOutput: %s", err, output)
	}

	// Verify local config is still present (reconfigured)
	version = getGitConfigWithScope(t, dir, "gitflow.version", "local")
	if version != "1.0" {
		t.Errorf("Expected gitflow.version in local config to be '1.0', got: %s", version)
	}
}
