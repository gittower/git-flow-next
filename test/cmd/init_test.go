package cmd_test

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// setupTestRepo creates a temporary Git repository for testing
func setupTestRepo(t *testing.T) string {
	// Create a temporary directory
	tempDir, err := os.MkdirTemp("", "git-flow-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}

	// Initialize Git repository
	cmd := exec.Command("git", "init")
	cmd.Dir = tempDir
	if err := cmd.Run(); err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to initialize Git repository: %v", err)
	}

	// Set Git user configuration for the test repository
	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = tempDir
	if err := cmd.Run(); err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to set Git user.name: %v", err)
	}

	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = tempDir
	if err := cmd.Run(); err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to set Git user.email: %v", err)
	}

	return tempDir
}

// cleanupTestRepo removes the temporary Git repository
func cleanupTestRepo(t *testing.T, dir string) {
	if err := os.RemoveAll(dir); err != nil {
		t.Fatalf("Failed to remove temp directory: %v", err)
	}
}

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
	// Always build the git-flow binary before running tests
	gitFlowPath, err := filepath.Abs(filepath.Join("..", "..", "git-flow"))
	if err != nil {
		t.Fatalf("Failed to get absolute path to git-flow: %v", err)
	}

	buildCmd := exec.Command("go", "build", "-o", gitFlowPath)
	buildCmd.Dir = filepath.Join("..", "..")
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build git-flow: %v", err)
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
	// Build the git-flow binary if it doesn't exist
	gitFlowPath, err := filepath.Abs(filepath.Join("..", "..", "git-flow"))
	if err != nil {
		t.Fatalf("Failed to get absolute path to git-flow: %v", err)
	}

	if _, err := os.Stat(gitFlowPath); os.IsNotExist(err) {
		buildCmd := exec.Command("go", "build", "-o", gitFlowPath)
		buildCmd.Dir = filepath.Join("..", "..")
		if err := buildCmd.Run(); err != nil {
			t.Fatalf("Failed to build git-flow: %v", err)
		}
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
	dir := setupTestRepo(t)
	defer cleanupTestRepo(t, dir)

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
	dir := setupTestRepo(t)
	defer cleanupTestRepo(t, dir)

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
	dir := setupTestRepo(t)
	defer cleanupTestRepo(t, dir)

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
	dir := setupTestRepo(t)
	defer cleanupTestRepo(t, dir)

	// Run git-flow init --defaults --create-branches
	output, err := runGitFlow(t, dir, "init", "--defaults", "--create-branches")
	if err != nil {
		t.Fatalf("Failed to run git-flow init --defaults --create-branches: %v\nOutput: %s", err, output)
	}

	// Check if the output contains the expected message
	if !strings.Contains(output, "Created branch 'main'") {
		t.Errorf("Expected output to contain 'Created branch 'main'', got: %s", output)
	}

	if !strings.Contains(output, "Created branch 'develop'") {
		t.Errorf("Expected output to contain 'Created branch 'develop'', got: %s", output)
	}

	// Check if the branches were actually created
	if !branchExists(t, dir, "main") {
		t.Errorf("Expected 'main' branch to exist")
	}

	if !branchExists(t, dir, "develop") {
		t.Errorf("Expected 'develop' branch to exist")
	}
}

// TestInitInteractiveWithBranchCreation tests the interactive init command with branch creation
func TestInitInteractiveWithBranchCreation(t *testing.T) {
	// Setup
	dir := setupTestRepo(t)
	defer cleanupTestRepo(t, dir)

	// Build the git-flow binary if it doesn't exist
	gitFlowPath, err := filepath.Abs(filepath.Join("..", "..", "git-flow"))
	if err != nil {
		t.Fatalf("Failed to get absolute path to git-flow: %v", err)
	}

	if _, err := os.Stat(gitFlowPath); os.IsNotExist(err) {
		buildCmd := exec.Command("go", "build", "-o", gitFlowPath)
		buildCmd.Dir = filepath.Join("..", "..")
		if err := buildCmd.Run(); err != nil {
			t.Fatalf("Failed to build git-flow: %v", err)
		}
	}

	// Create a script file with the answers (without the 'y' for branch creation)
	scriptPath := filepath.Join(dir, "answers.txt")
	answers := "custom-main\ncustom-dev\nf/\nr/\nh/\ns/\n"
	err = os.WriteFile(scriptPath, []byte(answers), 0644)
	if err != nil {
		t.Fatalf("Failed to create answers file: %v", err)
	}

	// Run git-flow init with the script file as input and the --create-branches flag
	cmd := exec.Command("bash", "-c", fmt.Sprintf("cat %s | %s init --create-branches", scriptPath, gitFlowPath))
	cmd.Dir = dir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err != nil {
		t.Fatalf("Failed to run git-flow init: %v\nOutput: %s", err, stdout.String()+stderr.String())
	}

	// Log the output for debugging
	output := stdout.String() + stderr.String()
	t.Logf("Command output: %s", output)

	// Check if the branches were actually created
	if !branchExists(t, dir, "custom-main") {
		t.Errorf("Expected 'custom-main' branch to exist")
	}

	if !branchExists(t, dir, "custom-dev") {
		t.Errorf("Expected 'custom-dev' branch to exist")
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
}
