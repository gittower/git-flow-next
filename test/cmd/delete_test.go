package cmd_test

import (
	"testing"

	"github.com/gittower/git-flow-next/test/testutil"
)

// TestDeleteFeature tests the basic branch deletion functionality for feature branches.
// Steps:
// 1. Sets up a test repository and initializes git-flow
// 2. Creates a feature branch
// 3. Adds unmerged changes to the feature branch
// 4. Attempts to delete without force flag (should fail)
// 5. Deletes with force flag
// 6. Verifies the branch is deleted
func TestDeleteFeature(t *testing.T) {
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

// TestDeleteCurrentFeature tests deleting a feature branch while it is checked out.
// Steps:
// 1. Sets up a test repository and initializes git-flow
// 2. Creates and checks out a feature branch
// 3. Deletes the current branch with force flag
// 4. Verifies we're automatically switched to develop branch
// 5. Verifies the feature branch is deleted
func TestDeleteCurrentFeature(t *testing.T) {
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
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

// TestDeleteNonExistentFeature tests the behavior when attempting to delete a branch that doesn't exist.
// Steps:
// 1. Sets up a test repository and initializes git-flow
// 2. Attempts to delete a non-existent branch
// 3. Verifies the operation fails with appropriate error
func TestDeleteNonExistentFeature(t *testing.T) {
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Try to delete non-existent branch
	output, err = testutil.RunGitFlow(t, dir, "feature", "delete", "nonexistent")
	if err == nil {
		t.Fatal("Expected delete to fail for non-existent branch")
	}
}

// TestDeleteMergedFeature tests the behavior when attempting to delete a branch that has already been merged.
// Steps:
// 1. Sets up a test repository and initializes git-flow
// 2. Creates a feature branch
// 3. Adds changes and commits them
// 4. Finishes the feature branch (merges it)
// 5. Attempts to delete the merged branch
// 6. Verifies the operation fails with appropriate error
func TestDeleteMergedFeature(t *testing.T) {
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
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

// TestDeleteWithInvalidBranchType tests the behavior when attempting to delete a branch with an invalid branch type.
// Steps:
// 1. Sets up a test repository and initializes git-flow
// 2. Attempts to delete a branch using an invalid branch type
// 3. Verifies the operation fails with appropriate error
func TestDeleteWithInvalidBranchType(t *testing.T) {
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Try to delete with invalid branch type
	output, err = testutil.RunGitFlow(t, dir, "invalid", "delete", "some-branch")
	if err == nil {
		t.Fatal("Expected delete to fail with invalid branch type")
	}
}

// TestDeleteFeatureWithRemote tests the basic remote deletion functionality using the --remote flag.
// Steps:
// 1. Sets up a test repository and initializes git-flow
// 2. Creates a feature branch
// 3. Adds a remote repository and pushes the branch
// 4. Verifies the branch exists on remote
// 5. Deletes the branch with --remote flag
// 6. Verifies the branch is deleted both locally and remotely
func TestDeleteFeatureWithRemote(t *testing.T) {
	// Setup test repository
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow
	_, err := testutil.RunGitFlow(t, dir, "init")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v", err)
	}

	// Create a feature branch
	_, err = testutil.RunGitFlow(t, dir, "feature", "start", "test-feature")
	if err != nil {
		t.Fatalf("Failed to create feature branch: %v", err)
	}

	// Create and add remote
	bareDir, err := testutil.AddRemote(t, dir, "origin", true)
	if err != nil {
		t.Fatalf("Failed to add remote: %v", err)
	}
	defer testutil.CleanupTestRepo(t, bareDir)

	// Verify feature branch exists on remote
	remoteBranch := "feature/test-feature"
	if !testutil.BranchExists(t, bareDir, remoteBranch) {
		t.Fatalf("Feature branch not found on remote")
	}

	// Delete feature branch with remote deletion
	_, err = testutil.RunGitFlow(t, dir, "feature", "delete", "test-feature", "--remote")
	if err != nil {
		t.Fatalf("Failed to delete feature branch: %v", err)
	}

	// Verify branch is deleted locally
	if testutil.BranchExists(t, dir, remoteBranch) {
		t.Errorf("Feature branch still exists locally")
	}

	// Verify branch is deleted on remote
	if testutil.BranchExists(t, bareDir, remoteBranch) {
		t.Errorf("Feature branch still exists on remote")
	}
}

// TestDeleteFeatureWithConfigEnabled tests remote deletion when enabled through configuration.
// Steps:
// 1. Sets up a test repository and initializes git-flow
// 2. Enables remote deletion in git-flow config
// 3. Creates a feature branch
// 4. Adds a remote repository and pushes the branch
// 5. Verifies the branch exists on remote
// 6. Deletes the branch without --remote flag (should use config)
// 7. Verifies the branch is deleted both locally and remotely
func TestDeleteFeatureWithConfigEnabled(t *testing.T) {
	// Setup test repository
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow
	_, err := testutil.RunGitFlow(t, dir, "init")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v", err)
	}

	// Enable remote deletion in config
	_, err = testutil.RunGit(t, dir, "config", "gitflow.branch.feature.deleteRemote", "true")
	if err != nil {
		t.Fatalf("Failed to set config: %v", err)
	}

	// Create a feature branch
	_, err = testutil.RunGitFlow(t, dir, "feature", "start", "test-feature")
	if err != nil {
		t.Fatalf("Failed to create feature branch: %v", err)
	}

	// Create and add remote
	bareDir, err := testutil.AddRemote(t, dir, "origin", true)
	if err != nil {
		t.Fatalf("Failed to add remote: %v", err)
	}
	defer testutil.CleanupTestRepo(t, bareDir)

	// Verify feature branch exists on remote
	remoteBranch := "feature/test-feature"
	if !testutil.BranchExists(t, bareDir, remoteBranch) {
		t.Fatalf("Feature branch not found on remote")
	}

	// Delete feature branch without remote flag
	_, err = testutil.RunGitFlow(t, dir, "feature", "delete", "test-feature")
	if err != nil {
		t.Fatalf("Failed to delete feature branch: %v", err)
	}

	// Verify branch is deleted locally
	if testutil.BranchExists(t, dir, remoteBranch) {
		t.Errorf("Feature branch still exists locally")
	}

	// Verify branch is deleted on remote
	if testutil.BranchExists(t, bareDir, remoteBranch) {
		t.Errorf("Feature branch still exists on remote")
	}
}

// TestDeleteFeatureWithConfigDisabled tests that remote deletion is disabled when configured.
// Steps:
// 1. Sets up a test repository and initializes git-flow
// 2. Disables remote deletion in git-flow config
// 3. Creates a feature branch
// 4. Adds a remote repository and pushes the branch
// 5. Verifies the branch exists on remote
// 6. Deletes the branch without --remote flag
// 7. Verifies the branch is deleted locally but remains on remote
func TestDeleteFeatureWithConfigDisabled(t *testing.T) {
	// Setup test repository
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow
	_, err := testutil.RunGitFlow(t, dir, "init")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v", err)
	}

	// Disable remote deletion in config
	_, err = testutil.RunGit(t, dir, "config", "gitflow.branch.feature.deleteRemote", "false")
	if err != nil {
		t.Fatalf("Failed to set config: %v", err)
	}

	// Create a feature branch
	_, err = testutil.RunGitFlow(t, dir, "feature", "start", "test-feature")
	if err != nil {
		t.Fatalf("Failed to create feature branch: %v", err)
	}

	// Create and add remote
	bareDir, err := testutil.AddRemote(t, dir, "origin", true)
	if err != nil {
		t.Fatalf("Failed to add remote: %v", err)
	}
	defer testutil.CleanupTestRepo(t, bareDir)

	// Verify feature branch exists on remote
	remoteBranch := "feature/test-feature"
	if !testutil.BranchExists(t, bareDir, remoteBranch) {
		t.Fatalf("Feature branch not found on remote")
	}

	// Delete feature branch without remote flag
	_, err = testutil.RunGitFlow(t, dir, "feature", "delete", "test-feature")
	if err != nil {
		t.Fatalf("Failed to delete feature branch: %v", err)
	}

	// Verify branch is deleted locally
	if testutil.BranchExists(t, dir, remoteBranch) {
		t.Errorf("Feature branch still exists locally")
	}

	// Verify branch still exists on remote
	if !testutil.BranchExists(t, bareDir, remoteBranch) {
		t.Errorf("Feature branch should still exist on remote")
	}
}

// TestDeleteFeatureWithCommandLineOverride tests that command line flag overrides configuration.
// Steps:
// 1. Sets up a test repository and initializes git-flow
// 2. Disables remote deletion in git-flow config
// 3. Creates a feature branch
// 4. Adds a remote repository and pushes the branch
// 5. Verifies the branch exists on remote
// 6. Deletes the branch with --remote flag (should override config)
// 7. Verifies the branch is deleted both locally and remotely
func TestDeleteFeatureWithCommandLineOverride(t *testing.T) {
	// Setup test repository
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow
	_, err := testutil.RunGitFlow(t, dir, "init")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v", err)
	}

	// Disable remote deletion in config
	_, err = testutil.RunGit(t, dir, "config", "gitflow.branch.feature.deleteRemote", "false")
	if err != nil {
		t.Fatalf("Failed to set config: %v", err)
	}

	// Create a feature branch
	_, err = testutil.RunGitFlow(t, dir, "feature", "start", "test-feature")
	if err != nil {
		t.Fatalf("Failed to create feature branch: %v", err)
	}

	// Create and add remote
	bareDir, err := testutil.AddRemote(t, dir, "origin", true)
	if err != nil {
		t.Fatalf("Failed to add remote: %v", err)
	}
	defer testutil.CleanupTestRepo(t, bareDir)

	// Verify feature branch exists on remote
	remoteBranch := "feature/test-feature"
	if !testutil.BranchExists(t, bareDir, remoteBranch) {
		t.Fatalf("Feature branch not found on remote")
	}

	// Delete feature branch with remote flag to override config
	_, err = testutil.RunGitFlow(t, dir, "feature", "delete", "test-feature", "--remote")
	if err != nil {
		t.Fatalf("Failed to delete feature branch: %v", err)
	}

	// Verify branch is deleted locally
	if testutil.BranchExists(t, dir, remoteBranch) {
		t.Errorf("Feature branch still exists locally")
	}

	// Verify branch is deleted on remote
	if testutil.BranchExists(t, bareDir, remoteBranch) {
		t.Errorf("Feature branch still exists on remote")
	}
}

// TestDeleteFeatureWithNonExistentRemote tests the behavior when attempting to delete a remote branch that doesn't exist.
// Steps:
// 1. Sets up a test repository and initializes git-flow
// 2. Creates a feature branch
// 3. Adds a remote repository but doesn't push the branch
// 4. Attempts to delete the branch with --remote flag
// 5. Verifies the branch is deleted locally
// 6. Verifies an error occurs when trying to delete the non-existent remote branch
func TestDeleteFeatureWithNonExistentRemote(t *testing.T) {
	// Setup test repository
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow
	_, err := testutil.RunGitFlow(t, dir, "init")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v", err)
	}

	// Create a feature branch
	_, err = testutil.RunGitFlow(t, dir, "feature", "start", "test-feature")
	if err != nil {
		t.Fatalf("Failed to create feature branch: %v", err)
	}

	// Create and add remote without pushing branches
	bareDir, err := testutil.AddRemote(t, dir, "origin", false)
	if err != nil {
		t.Fatalf("Failed to add remote: %v", err)
	}
	defer testutil.CleanupTestRepo(t, bareDir)

	// Verify feature branch does not exist on remote
	remoteBranch := "feature/test-feature"
	if testutil.BranchExists(t, bareDir, remoteBranch) {
		t.Fatalf("Feature branch unexpectedly exists on remote")
	}

	// Delete feature branch with remote deletion - should fail
	_, err = testutil.RunGitFlow(t, dir, "feature", "delete", "test-feature", "--remote")
	if err == nil {
		t.Fatalf("Expected error when deleting non-existent remote branch")
	}

	// Verify branch is deleted locally
	if testutil.BranchExists(t, dir, remoteBranch) {
		t.Errorf("Feature branch still exists locally")
	}
}

// TestDeleteFeatureWithCustomRemote tests remote deletion using a custom remote name configured in git-flow config.
// Steps:
// 1. Sets up a test repository and initializes git-flow
// 2. Configures a custom remote name in git-flow config
// 3. Creates a feature branch
// 4. Adds a remote repository with the custom name and pushes the branch
// 5. Verifies the branch exists on remote
// 6. Deletes the branch with --remote flag
// 7. Verifies the branch is deleted both locally and remotely
func TestDeleteFeatureWithCustomRemote(t *testing.T) {
	// Setup test repository
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow
	_, err := testutil.RunGitFlow(t, dir, "init")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v", err)
	}

	// Configure custom remote name
	customRemote := "upstream"
	_, err = testutil.RunGit(t, dir, "config", "gitflow.remote", customRemote)
	if err != nil {
		t.Fatalf("Failed to set custom remote name: %v", err)
	}

	// Create a feature branch
	_, err = testutil.RunGitFlow(t, dir, "feature", "start", "test-feature")
	if err != nil {
		t.Fatalf("Failed to create feature branch: %v", err)
	}

	// Create and add remote with custom name
	bareDir, err := testutil.AddRemote(t, dir, customRemote, true)
	if err != nil {
		t.Fatalf("Failed to add remote: %v", err)
	}
	defer testutil.CleanupTestRepo(t, bareDir)

	// Verify feature branch exists on remote
	remoteBranch := "feature/test-feature"
	if !testutil.BranchExists(t, bareDir, remoteBranch) {
		t.Fatalf("Feature branch not found on remote")
	}

	// Delete feature branch with remote deletion
	_, err = testutil.RunGitFlow(t, dir, "feature", "delete", "test-feature", "--remote")
	if err != nil {
		t.Fatalf("Failed to delete feature branch: %v", err)
	}

	// Verify branch is deleted locally
	if testutil.BranchExists(t, dir, remoteBranch) {
		t.Errorf("Feature branch still exists locally")
	}

	// Verify branch is deleted on remote
	if testutil.BranchExists(t, bareDir, remoteBranch) {
		t.Errorf("Feature branch still exists on remote")
	}
}

// TestDeleteFeatureWithNoRemoteOverride tests that the --no-remote flag overrides configuration.
// Steps:
// 1. Sets up a test repository and initializes git-flow
// 2. Enables remote deletion in git-flow config
// 3. Creates a feature branch
// 4. Adds a remote repository and pushes the branch
// 5. Verifies the branch exists on remote
// 6. Deletes the branch with --no-remote flag (should override config)
// 7. Verifies the branch is deleted locally but remains on remote
func TestDeleteFeatureWithNoRemoteOverride(t *testing.T) {
	// Setup test repository
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow
	_, err := testutil.RunGitFlow(t, dir, "init")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v", err)
	}

	// Enable remote deletion in config
	_, err = testutil.RunGit(t, dir, "config", "gitflow.branch.feature.deleteRemote", "true")
	if err != nil {
		t.Fatalf("Failed to set config: %v", err)
	}

	// Create a feature branch
	_, err = testutil.RunGitFlow(t, dir, "feature", "start", "test-feature")
	if err != nil {
		t.Fatalf("Failed to create feature branch: %v", err)
	}

	// Create and add remote
	bareDir, err := testutil.AddRemote(t, dir, "origin", true)
	if err != nil {
		t.Fatalf("Failed to add remote: %v", err)
	}
	defer testutil.CleanupTestRepo(t, bareDir)

	// Verify feature branch exists on remote
	remoteBranch := "feature/test-feature"
	if !testutil.BranchExists(t, bareDir, remoteBranch) {
		t.Fatalf("Feature branch not found on remote")
	}

	// Delete feature branch with no-remote flag to override config
	_, err = testutil.RunGitFlow(t, dir, "feature", "delete", "test-feature", "--no-remote")
	if err != nil {
		t.Fatalf("Failed to delete feature branch: %v", err)
	}

	// Verify branch is deleted locally
	if testutil.BranchExists(t, dir, remoteBranch) {
		t.Errorf("Feature branch still exists locally")
	}

	// Verify branch still exists on remote
	if !testutil.BranchExists(t, bareDir, remoteBranch) {
		t.Errorf("Feature branch should still exist on remote")
	}
}
