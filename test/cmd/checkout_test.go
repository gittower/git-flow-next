package cmd_test

import (
	"strings"
	"testing"

	"github.com/gittower/git-flow-next/test/testutil"
)

// TestCheckoutFeature tests the basic branch checkout functionality for feature branches.
// Steps:
// 1. Sets up a test repository and initializes git-flow
// 2. Creates a feature branch
// 3. Switches back to develop branch
// 4. Checks out the feature branch using git-flow
// 5. Verifies we're on the correct branch
func TestCheckoutFeature(t *testing.T) {
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Create a feature branch
	output, err = testutil.RunGitFlow(t, dir, "feature", "start", "test-feature")
	if err != nil {
		t.Fatalf("Failed to create feature branch: %v\nOutput: %s", err, output)
	}

	// Switch back to develop
	_, err = testutil.RunGit(t, dir, "checkout", "develop")
	if err != nil {
		t.Fatalf("Failed to checkout develop: %v", err)
	}

	// Checkout the feature branch
	output, err = testutil.RunGitFlow(t, dir, "feature", "checkout", "test-feature")
	if err != nil {
		t.Fatalf("Failed to checkout feature branch: %v\nOutput: %s", err, output)
	}

	// Verify we're on the feature branch
	currentBranch := testutil.GetCurrentBranch(t, dir)
	if currentBranch != "feature/test-feature" {
		t.Errorf("Expected to be on feature/test-feature branch, got %s", currentBranch)
	}
}

// TestCheckoutFeatureWithPrefix tests checking out a feature branch using a unique prefix.
// Steps:
// 1. Sets up a test repository and initializes git-flow
// 2. Creates multiple feature branches
// 3. Switches back to develop branch
// 4. Checks out a feature branch using a unique prefix
// 5. Verifies we're on the correct branch
// 6. Attempts to checkout with ambiguous prefix (should fail)
func TestCheckoutFeatureWithPrefix(t *testing.T) {
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Create feature branches
	output, err = testutil.RunGitFlow(t, dir, "feature", "start", "test1-feature")
	if err != nil {
		t.Fatalf("Failed to create first feature branch: %v\nOutput: %s", err, output)
	}

	output, err = testutil.RunGitFlow(t, dir, "feature", "start", "test2-feature")
	if err != nil {
		t.Fatalf("Failed to create second feature branch: %v\nOutput: %s", err, output)
	}

	// Switch back to develop
	_, err = testutil.RunGit(t, dir, "checkout", "develop")
	if err != nil {
		t.Fatalf("Failed to checkout develop: %v", err)
	}

	// Checkout using unique prefix
	output, err = testutil.RunGitFlow(t, dir, "feature", "checkout", "test1")
	if err != nil {
		t.Fatalf("Failed to checkout feature branch by prefix: %v\nOutput: %s", err, output)
	}

	// Verify we're on the correct branch
	currentBranch := testutil.GetCurrentBranch(t, dir)
	if currentBranch != "feature/test1-feature" {
		t.Errorf("Expected to be on feature/test1-feature branch, got %s", currentBranch)
	}

	// Try to checkout with ambiguous prefix (should fail)
	output, err = testutil.RunGitFlow(t, dir, "feature", "checkout", "test")
	if err == nil {
		t.Fatal("Expected checkout to fail with ambiguous prefix")
	}
}

// TestCheckoutNonExistentFeature tests the behavior when attempting to checkout a non-existent branch.
// Steps:
// 1. Sets up a test repository and initializes git-flow
// 2. Attempts to checkout a non-existent branch
// 3. Verifies the operation fails with appropriate error
func TestCheckoutNonExistentFeature(t *testing.T) {
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Try to checkout non-existent branch
	output, err = testutil.RunGitFlow(t, dir, "feature", "checkout", "nonexistent")
	if err == nil {
		t.Fatal("Expected checkout to fail for non-existent branch")
	}
}

// TestCheckoutFeatureWithShowCommands tests checking out a feature branch with the --showcommands flag.
// Steps:
// 1. Sets up a test repository and initializes git-flow
// 2. Creates a feature branch
// 3. Switches back to develop branch
// 4. Checks out the feature branch with --showcommands flag
// 5. Verifies the output contains the git command
// 6. Verifies we're on the correct branch
func TestCheckoutFeatureWithShowCommands(t *testing.T) {
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Create a feature branch
	output, err = testutil.RunGitFlow(t, dir, "feature", "start", "test-feature")
	if err != nil {
		t.Fatalf("Failed to create feature branch: %v\nOutput: %s", err, output)
	}

	// Switch back to develop
	_, err = testutil.RunGit(t, dir, "checkout", "develop")
	if err != nil {
		t.Fatalf("Failed to checkout develop: %v", err)
	}

	// Checkout with --showcommands flag
	output, err = testutil.RunGitFlow(t, dir, "feature", "checkout", "--showcommands", "test-feature")
	if err != nil {
		t.Fatalf("Failed to checkout feature branch: %v\nOutput: %s", err, output)
	}

	// Verify the output contains the git command
	if !strings.Contains(output, "$ git checkout feature/test-feature") {
		t.Error("Expected output to show git command")
	}

	// Verify we're on the feature branch
	currentBranch := testutil.GetCurrentBranch(t, dir)
	if currentBranch != "feature/test-feature" {
		t.Errorf("Expected to be on feature/test-feature branch, got %s", currentBranch)
	}
}

// TestCheckoutWithInvalidBranchType tests the behavior when attempting to checkout with an invalid branch type.
// Steps:
// 1. Sets up a test repository and initializes git-flow
// 2. Attempts to checkout using an invalid branch type
// 3. Verifies the operation fails with appropriate error
func TestCheckoutWithInvalidBranchType(t *testing.T) {
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Try to checkout with invalid branch type
	output, err = testutil.RunGitFlow(t, dir, "invalid", "checkout", "some-branch")
	if err == nil {
		t.Fatal("Expected checkout to fail with invalid branch type")
	}
}

// TestCheckoutFeatureNoArgs tests the behavior when running checkout without providing a branch name.
// Steps:
// 1. Sets up a test repository and initializes git-flow
// 2. Creates multiple feature branches
// 3. Runs checkout without args to list branches
// 4. Verifies the output contains all feature branches
func TestCheckoutFeatureNoArgs(t *testing.T) {
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Create some feature branches
	output, err = testutil.RunGitFlow(t, dir, "feature", "start", "test1-feature")
	if err != nil {
		t.Fatalf("Failed to create first feature branch: %v\nOutput: %s", err, output)
	}

	output, err = testutil.RunGitFlow(t, dir, "feature", "start", "test2-feature")
	if err != nil {
		t.Fatalf("Failed to create second feature branch: %v\nOutput: %s", err, output)
	}

	// Run checkout without args to list branches
	output, err = testutil.RunGitFlow(t, dir, "feature", "checkout")
	if err != nil {
		t.Fatalf("Failed to list feature branches: %v\nOutput: %s", err, output)
	}

	// Verify output contains both branches
	if !strings.Contains(output, "test1-feature") || !strings.Contains(output, "test2-feature") {
		t.Error("Expected output to list all feature branches")
	}
}
