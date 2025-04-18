package cmd_test

import (
	"testing"

	"github.com/gittower/git-flow-next/test/testutil"
)

// TestRenameFeature tests renaming a feature branch
func TestRenameFeature(t *testing.T) {
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	output, err := testutil.RunGitFlow(t, dir, "init", "-d")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Create a feature branch
	output, err = testutil.RunGitFlow(t, dir, "feature", "start", "old-feature")
	if err != nil {
		t.Fatalf("Failed to create feature branch: %v\nOutput: %s", err, output)
	}

	// Add some changes
	testutil.WriteFile(t, dir, "feature.txt", "feature content")
	_, err = testutil.RunGit(t, dir, "add", "feature.txt")
	if err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Add feature file")
	if err != nil {
		t.Fatalf("Failed to commit file: %v", err)
	}

	// Rename the feature branch
	output, err = testutil.RunGitFlow(t, dir, "feature", "rename", "old-feature", "new-feature")
	if err != nil {
		t.Fatalf("Failed to rename feature branch: %v\nOutput: %s", err, output)
	}

	// Verify old branch doesn't exist
	if testutil.BranchExists(t, dir, "feature/old-feature") {
		t.Error("Expected old feature branch to be deleted")
	}

	// Verify new branch exists
	if !testutil.BranchExists(t, dir, "feature/new-feature") {
		t.Error("Expected new feature branch to exist")
	}

	// Verify the changes are in the new branch
	_, err = testutil.RunGit(t, dir, "checkout", "feature/new-feature")
	if err != nil {
		t.Fatalf("Failed to checkout new feature branch: %v", err)
	}

	content, err := testutil.RunGit(t, dir, "--no-pager", "show", "HEAD:feature.txt")
	if err != nil {
		t.Fatalf("Failed to read file content: %v", err)
	}
	if content != "feature content" {
		t.Errorf("Expected feature.txt content to be 'feature content', got '%s'", content)
	}
}

// TestRenameCurrentFeature tests renaming the current feature branch
func TestRenameCurrentFeature(t *testing.T) {
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	output, err := testutil.RunGitFlow(t, dir, "init", "-d")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Create and checkout a feature branch
	output, err = testutil.RunGitFlow(t, dir, "feature", "start", "current-feature")
	if err != nil {
		t.Fatalf("Failed to create feature branch: %v\nOutput: %s", err, output)
	}

	// Rename the current feature branch
	output, err = testutil.RunGitFlow(t, dir, "feature", "rename", "current-feature", "renamed-feature")
	if err != nil {
		t.Fatalf("Failed to rename current feature branch: %v\nOutput: %s", err, output)
	}

	// Verify we're on the renamed branch
	currentBranch := testutil.GetCurrentBranch(t, dir)
	if currentBranch != "feature/renamed-feature" {
		t.Errorf("Expected to be on feature/renamed-feature branch, got %s", currentBranch)
	}
}

// TestRenameNonExistentFeature tests renaming a non-existent feature branch
func TestRenameNonExistentFeature(t *testing.T) {
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	output, err := testutil.RunGitFlow(t, dir, "init", "-d")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Try to rename non-existent branch
	output, err = testutil.RunGitFlow(t, dir, "feature", "rename", "nonexistent", "new-name")
	if err == nil {
		t.Fatal("Expected rename to fail for non-existent branch")
	}
}

// TestRenameToExistingFeature tests renaming a feature branch to a name that already exists
func TestRenameToExistingFeature(t *testing.T) {
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	output, err := testutil.RunGitFlow(t, dir, "init", "-d")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Create first feature branch
	output, err = testutil.RunGitFlow(t, dir, "feature", "start", "first-feature")
	if err != nil {
		t.Fatalf("Failed to create first feature branch: %v\nOutput: %s", err, output)
	}

	// Create second feature branch
	output, err = testutil.RunGitFlow(t, dir, "feature", "start", "second-feature")
	if err != nil {
		t.Fatalf("Failed to create second feature branch: %v\nOutput: %s", err, output)
	}

	// Try to rename first branch to second branch name
	output, err = testutil.RunGitFlow(t, dir, "feature", "rename", "first-feature", "second-feature")
	if err == nil {
		t.Fatal("Expected rename to fail when target name already exists")
	}
}

// TestRenameWithInvalidBranchType tests renaming with an invalid branch type
func TestRenameWithInvalidBranchType(t *testing.T) {
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	output, err := testutil.RunGitFlow(t, dir, "init", "-d")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Try to rename with invalid branch type
	output, err = testutil.RunGitFlow(t, dir, "invalid", "rename", "old-name", "new-name")
	if err == nil {
		t.Fatal("Expected rename to fail with invalid branch type")
	}
}
