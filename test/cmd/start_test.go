package cmd_test

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// runGit runs a git command in the specified directory and returns its output
func runGit(t *testing.T, dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = filepath.Join(dir)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// TestStartFeatureBranch tests the start command for feature branches
func TestStartFeatureBranch(t *testing.T) {
	// Setup
	dir := setupTestRepo(t)
	defer cleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	output, err := runGitFlow(t, dir, "init", "--defaults", "--create-branches")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Run git-flow feature start my-feature
	output, err = runGitFlow(t, dir, "feature", "start", "my-feature")
	if err != nil {
		t.Fatalf("Failed to run git-flow feature start: %v\nOutput: %s", err, output)
	}

	// Check if the output contains the expected message
	if !strings.Contains(output, "Created branch 'feature/my-feature'") {
		t.Errorf("Expected output to contain 'Created branch 'feature/my-feature'', got: %s", output)
	}

	// Check if the branch was actually created
	if !branchExists(t, dir, "feature/my-feature") {
		t.Errorf("Expected 'feature/my-feature' branch to exist")
	}

	// Check if the branch is based on develop
	cmd := exec.Command("git", "merge-base", "--is-ancestor", "develop", "feature/my-feature")
	cmd.Dir = dir
	err = cmd.Run()
	if err != nil {
		t.Errorf("Expected 'feature/my-feature' to be based on 'develop'")
	}
}

// TestStartReleaseAndHotfixBranches tests the start command for release and hotfix branches
func TestStartReleaseAndHotfixBranches(t *testing.T) {
	// Setup
	dir := setupTestRepo(t)
	defer cleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	output, err := runGitFlow(t, dir, "init", "--defaults", "--create-branches")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Run git-flow release start 1.0.0
	output, err = runGitFlow(t, dir, "release", "start", "1.0.0")
	if err != nil {
		t.Fatalf("Failed to run git-flow release start: %v\nOutput: %s", err, output)
	}

	// Check if the output contains the expected message
	if !strings.Contains(output, "Created branch 'release/1.0.0'") {
		t.Errorf("Expected output to contain 'Created branch 'release/1.0.0'', got: %s", output)
	}

	// Check if the branch was actually created
	if !branchExists(t, dir, "release/1.0.0") {
		t.Errorf("Expected 'release/1.0.0' branch to exist")
	}

	// Run git-flow hotfix start 1.0.1
	output, err = runGitFlow(t, dir, "hotfix", "start", "1.0.1")
	if err != nil {
		t.Fatalf("Failed to run git-flow hotfix start: %v\nOutput: %s", err, output)
	}

	// Check if the output contains the expected message
	if !strings.Contains(output, "Created branch 'hotfix/1.0.1'") {
		t.Errorf("Expected output to contain 'Created branch 'hotfix/1.0.1'', got: %s", output)
	}

	// Check if the branch was actually created
	if !branchExists(t, dir, "hotfix/1.0.1") {
		t.Errorf("Expected 'hotfix/1.0.1' branch to exist")
	}
}

// TestStartWithCustomConfig tests the start command with custom configuration
func TestStartWithCustomConfig(t *testing.T) {
	// Setup
	dir := setupTestRepo(t)
	defer cleanupTestRepo(t, dir)

	// Initialize git-flow with custom configuration
	input := "custom-main\ncustom-dev\nf/\nr/\nh/\ns/\n"
	output, err := runGitFlowWithInput(t, dir, input, "init", "--create-branches")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Run git-flow feature start my-feature
	output, err = runGitFlow(t, dir, "feature", "start", "my-feature")
	if err != nil {
		t.Fatalf("Failed to run git-flow feature start: %v\nOutput: %s", err, output)
	}

	// Check if the output contains the expected message
	if !strings.Contains(output, "Created branch 'f/my-feature'") {
		t.Errorf("Expected output to contain 'Created branch 'f/my-feature'', got: %s", output)
	}

	// Check if the branch was actually created
	if !branchExists(t, dir, "f/my-feature") {
		t.Errorf("Expected 'f/my-feature' branch to exist")
	}

	// Check if the branch is based on custom-dev
	cmd := exec.Command("git", "merge-base", "--is-ancestor", "custom-dev", "f/my-feature")
	cmd.Dir = dir
	err = cmd.Run()
	if err != nil {
		t.Errorf("Expected 'f/my-feature' to be based on 'custom-dev'")
	}
}

// TestStartWithExistingBranch tests the start command with an existing branch
func TestStartWithExistingBranch(t *testing.T) {
	// Setup
	dir := setupTestRepo(t)
	defer cleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	output, err := runGitFlow(t, dir, "init", "--defaults", "--create-branches")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Create a feature branch manually
	cmd := exec.Command("git", "checkout", "-b", "feature/existing-feature", "develop")
	cmd.Dir = dir
	err = cmd.Run()
	if err != nil {
		t.Fatalf("Failed to create feature branch: %v", err)
	}

	// Return to develop
	cmd = exec.Command("git", "checkout", "develop")
	cmd.Dir = dir
	err = cmd.Run()
	if err != nil {
		t.Fatalf("Failed to checkout develop: %v", err)
	}

	// Run git-flow feature start existing-feature
	output, err = runGitFlow(t, dir, "feature", "start", "existing-feature")
	if err != nil {
		t.Fatalf("Failed to run git-flow feature start: %v\nOutput: %s", err, output)
	}

	// Check if the output contains the expected message
	if !strings.Contains(output, "Branch 'feature/existing-feature' already exists") {
		t.Errorf("Expected output to contain 'Branch 'feature/existing-feature' already exists', got: %s", output)
	}
}

// TestStartWithNonExistentStartPoint tests the start command with a non-existent start point
func TestStartWithNonExistentStartPoint(t *testing.T) {
	// Setup
	dir := setupTestRepo(t)
	defer cleanupTestRepo(t, dir)

	// Initialize git-flow with defaults but don't create branches
	output, err := runGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Run git-flow feature start my-feature
	output, err = runGitFlow(t, dir, "feature", "start", "my-feature")
	if err != nil {
		t.Fatalf("Failed to run git-flow feature start: %v\nOutput: %s", err, output)
	}

	// Check if the output contains the expected message
	if !strings.Contains(output, "Start point branch 'develop' does not exist") {
		t.Errorf("Expected output to contain 'Start point branch 'develop' does not exist', got: %s", output)
	}

	// Check that the branch was not created
	if branchExists(t, dir, "feature/my-feature") {
		t.Errorf("Expected 'feature/my-feature' branch to not exist")
	}
}

// TestStartWithNoStartPoint tests that when start point is not specified, parent branch is used
func TestStartWithNoStartPoint(t *testing.T) {
	// Setup
	dir := setupTestRepo(t)
	defer cleanupTestRepo(t, dir)

	// Initialize git-flow with custom configuration
	input := "main\ndevelop\nf/\nr/\nh/\ns/\n"
	output, err := runGitFlowWithInput(t, dir, input, "init", "--create-branches")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Create a feature branch
	output, err = runGitFlow(t, dir, "feature", "start", "test-feature")
	if err != nil {
		t.Fatalf("Failed to create feature branch: %v\nOutput: %s", err, output)
	}

	// Verify that the branch was created from develop (parent branch)
	if !strings.Contains(output, "Created branch 'f/test-feature' from 'develop'") {
		t.Errorf("Expected branch to be created from 'develop', got: %s", output)
	}

	// Verify that the branch exists
	output, err = runGit(t, dir, "branch", "--list", "f/test-feature")
	if err != nil {
		t.Fatalf("Failed to list branches: %v\nOutput: %s", err, output)
	}
	if !strings.Contains(output, "f/test-feature") {
		t.Errorf("Expected branch 'f/test-feature' to exist, got: %s", output)
	}

	// Get the commit hash of develop
	developHash, err := runGit(t, dir, "rev-parse", "develop")
	if err != nil {
		t.Fatalf("Failed to get develop hash: %v\nOutput: %s", err, output)
	}

	// Get the commit hash of the feature branch
	featureHash, err := runGit(t, dir, "rev-parse", "f/test-feature")
	if err != nil {
		t.Fatalf("Failed to get feature hash: %v\nOutput: %s", err, output)
	}

	// Verify that the feature branch was created from develop
	if developHash != featureHash {
		t.Errorf("Expected feature branch to be at the same commit as develop")
	}
}
