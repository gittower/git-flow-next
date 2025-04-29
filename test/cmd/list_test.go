package cmd_test

import (
	"strings"
	"testing"
)

// TestListFeatureBranches tests the listing of feature branches.
// Steps:
// 1. Sets up a test repository and initializes git-flow
// 2. Creates multiple feature branches
// 3. Lists feature branches
// 4. Verifies the output contains all created feature branches
func TestListFeatureBranches(t *testing.T) {
	// Setup
	dir := setupTestRepo(t)
	defer cleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	output, err := runGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Create a feature branch
	output, err = runGitFlow(t, dir, "feature", "start", "my-feature")
	if err != nil {
		t.Fatalf("Failed to create feature branch: %v\nOutput: %s", err, output)
	}

	// Create another feature branch
	output, err = runGitFlow(t, dir, "feature", "start", "another-feature")
	if err != nil {
		t.Fatalf("Failed to create another feature branch: %v\nOutput: %s", err, output)
	}

	// List feature branches
	output, err = runGitFlow(t, dir, "feature", "list")
	if err != nil {
		t.Fatalf("Failed to list feature branches: %v\nOutput: %s", err, output)
	}

	// Check if the output contains the expected branches
	if !strings.Contains(output, "Feature branches:") {
		t.Errorf("Expected output to contain 'Feature branches:', got: %s", output)
	}

	if !strings.Contains(output, "my-feature") {
		t.Errorf("Expected output to contain 'my-feature', got: %s", output)
	}

	if !strings.Contains(output, "another-feature") {
		t.Errorf("Expected output to contain 'another-feature', got: %s", output)
	}
}

// TestListReleaseAndHotfixBranches tests the listing of release and hotfix branches.
// Steps:
// 1. Sets up a test repository and initializes git-flow
// 2. Creates release and hotfix branches
// 3. Lists release and hotfix branches separately
// 4. Verifies the output contains the created branches
func TestListReleaseAndHotfixBranches(t *testing.T) {
	// Setup
	dir := setupTestRepo(t)
	defer cleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	output, err := runGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Create a release branch
	output, err = runGitFlow(t, dir, "release", "start", "1.0.0")
	if err != nil {
		t.Fatalf("Failed to create release branch: %v\nOutput: %s", err, output)
	}

	// Create a hotfix branch
	output, err = runGitFlow(t, dir, "hotfix", "start", "1.0.1")
	if err != nil {
		t.Fatalf("Failed to create hotfix branch: %v\nOutput: %s", err, output)
	}

	// List release branches
	output, err = runGitFlow(t, dir, "release", "list")
	if err != nil {
		t.Fatalf("Failed to list release branches: %v\nOutput: %s", err, output)
	}

	// Check if the output contains the expected branches
	if !strings.Contains(output, "Release branches:") {
		t.Errorf("Expected output to contain 'Release branches:', got: %s", output)
	}

	if !strings.Contains(output, "1.0.0") {
		t.Errorf("Expected output to contain '1.0.0', got: %s", output)
	}

	// List hotfix branches
	output, err = runGitFlow(t, dir, "hotfix", "list")
	if err != nil {
		t.Fatalf("Failed to list hotfix branches: %v\nOutput: %s", err, output)
	}

	// Check if the output contains the expected branches
	if !strings.Contains(output, "Hotfix branches:") {
		t.Errorf("Expected output to contain 'Hotfix branches:', got: %s", output)
	}

	if !strings.Contains(output, "1.0.1") {
		t.Errorf("Expected output to contain '1.0.1', got: %s", output)
	}
}

// TestListWithCustomConfig tests the listing of branches with custom configuration.
// Steps:
// 1. Sets up a test repository and initializes git-flow with custom config
// 2. Creates a feature branch with custom prefix
// 3. Lists feature branches
// 4. Verifies the output contains the branch with custom prefix
func TestListWithCustomConfig(t *testing.T) {
	// Setup
	dir := setupTestRepo(t)
	defer cleanupTestRepo(t, dir)

	// Initialize git-flow with custom configuration
	input := "custom-main\ncustom-dev\nf/\nr/\nh/\ns/\n"
	output, err := runGitFlowWithInput(t, dir, input, "init")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Create a feature branch
	output, err = runGitFlow(t, dir, "feature", "start", "my-feature")
	if err != nil {
		t.Fatalf("Failed to create feature branch: %v\nOutput: %s", err, output)
	}

	// List feature branches
	output, err = runGitFlow(t, dir, "feature", "list")
	if err != nil {
		t.Fatalf("Failed to list feature branches: %v\nOutput: %s", err, output)
	}

	// Check if the output contains the expected branches
	if !strings.Contains(output, "Feature branches:") {
		t.Errorf("Expected output to contain 'Feature branches:', got: %s", output)
	}

	if !strings.Contains(output, "my-feature") {
		t.Errorf("Expected output to contain 'my-feature', got: %s", output)
	}
}

// TestListEmptyBranches tests the listing of branches when no branches of a type exist.
// Steps:
// 1. Sets up a test repository and initializes git-flow
// 2. Lists feature branches without creating any
// 3. Verifies the output indicates no branches found
func TestListEmptyBranches(t *testing.T) {
	// Setup
	dir := setupTestRepo(t)
	defer cleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	output, err := runGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// List feature branches (should be empty)
	output, err = runGitFlow(t, dir, "feature", "list")
	if err != nil {
		t.Fatalf("Failed to list feature branches: %v\nOutput: %s", err, output)
	}

	// Check if the output contains the expected message
	if !strings.Contains(output, "No feature branches found") {
		t.Errorf("Expected output to contain 'No feature branches found', got: %s", output)
	}
}
