package cmd_test

import (
	"strings"
	"testing"

	"github.com/gittower/git-flow-next/test/testutil"
)

// TestCheckoutFeature tests checking out a feature branch
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

// TestCheckoutFeatureWithPrefix tests checking out a feature branch using a prefix
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

// TestCheckoutNonExistentFeature tests checking out a non-existent feature branch
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

// TestCheckoutFeatureWithShowCommands tests checking out a feature branch with --showcommands flag
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

// TestCheckoutWithInvalidBranchType tests checking out with an invalid branch type
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

// TestCheckoutFeatureNoArgs tests checking out without providing a branch name
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
