package cmd_test

import (
	"testing"

	"github.com/gittower/git-flow-next/test/testutil"
)

// TestDeleteFeature tests the delete command for feature branches
func TestDeleteFeature(t *testing.T) {
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	output, err := testutil.RunGitFlow(t, dir, "init", "-d")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Create a feature branch
	output, err = testutil.RunGitFlow(t, dir, "feature", "start", "test-feature")
	if err != nil {
		t.Fatalf("Failed to create feature branch: %v\nOutput: %s", err, output)
	}

	// Add some changes to make it unmerged
	testutil.WriteFile(t, dir, "feature.txt", "feature content")
	_, err = testutil.RunGit(t, dir, "add", "feature.txt")
	if err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Add feature file")
	if err != nil {
		t.Fatalf("Failed to commit file: %v", err)
	}

	// Try to delete without force flag (should fail)
	output, err = testutil.RunGitFlow(t, dir, "feature", "delete", "test-feature")
	if err == nil {
		t.Fatal("Expected delete to fail without force flag")
	}

	// Delete with force flag
	output, err = testutil.RunGitFlow(t, dir, "feature", "delete", "-f", "test-feature")
	if err != nil {
		t.Fatalf("Failed to delete feature branch: %v\nOutput: %s", err, output)
	}

	// Verify branch is deleted
	if testutil.BranchExists(t, dir, "feature/test-feature") {
		t.Error("Expected feature branch to be deleted")
	}
}

// TestDeleteCurrentFeature tests deleting the current feature branch
func TestDeleteCurrentFeature(t *testing.T) {
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

	// Delete current branch with force flag
	output, err = testutil.RunGitFlow(t, dir, "feature", "delete", "-f", "current-feature")
	if err != nil {
		t.Fatalf("Failed to delete current feature branch: %v\nOutput: %s", err, output)
	}

	// Verify we're on develop branch
	currentBranch := testutil.GetCurrentBranch(t, dir)
	if currentBranch != "develop" {
		t.Errorf("Expected to be on develop branch, got %s", currentBranch)
	}

	// Verify branch is deleted
	if testutil.BranchExists(t, dir, "feature/current-feature") {
		t.Error("Expected feature branch to be deleted")
	}
}

// TestDeleteNonExistentFeature tests deleting a non-existent feature branch
func TestDeleteNonExistentFeature(t *testing.T) {
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	output, err := testutil.RunGitFlow(t, dir, "init", "-d")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Try to delete non-existent branch
	output, err = testutil.RunGitFlow(t, dir, "feature", "delete", "nonexistent")
	if err == nil {
		t.Fatal("Expected delete to fail for non-existent branch")
	}
}

// TestDeleteMergedFeature tests deleting a merged feature branch
func TestDeleteMergedFeature(t *testing.T) {
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	output, err := testutil.RunGitFlow(t, dir, "init", "-d")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Create a feature branch
	output, err = testutil.RunGitFlow(t, dir, "feature", "start", "merged-feature")
	if err != nil {
		t.Fatalf("Failed to create feature branch: %v\nOutput: %s", err, output)
	}

	// Add some changes
	testutil.WriteFile(t, dir, "merged.txt", "merged content")
	_, err = testutil.RunGit(t, dir, "add", "merged.txt")
	if err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Add merged file")
	if err != nil {
		t.Fatalf("Failed to commit file: %v", err)
	}

	// Finish the feature (which merges it)
	output, err = testutil.RunGitFlow(t, dir, "feature", "finish", "merged-feature")
	if err != nil {
		t.Fatalf("Failed to finish feature branch: %v\nOutput: %s", err, output)
	}

	// Try to delete the already merged branch (should fail)
	output, err = testutil.RunGitFlow(t, dir, "feature", "delete", "merged-feature")
	if err == nil {
		t.Fatal("Expected delete to fail for already merged branch")
	}
}

// TestDeleteWithInvalidBranchType tests deleting with an invalid branch type
func TestDeleteWithInvalidBranchType(t *testing.T) {
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	output, err := testutil.RunGitFlow(t, dir, "init", "-d")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Try to delete with invalid branch type
	output, err = testutil.RunGitFlow(t, dir, "invalid", "delete", "some-branch")
	if err == nil {
		t.Fatal("Expected delete to fail with invalid branch type")
	}
}
