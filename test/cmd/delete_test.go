package cmd_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/gittower/git-flow-next/internal/errors"
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
	t.Parallel()
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
	t.Parallel()
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
//  1. Sets up a test repository and initializes git-flow
//  2. Attempts to delete a non-existent branch
//  3. Verifies the operation fails with a branch-not-found error (exit code 5),
//     not the "not initialized" error — proving the uninitialized gate does not
//     over-trigger in an initialized repository
func TestDeleteNonExistentFeature(t *testing.T) {
	t.Parallel()
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

	// It must fail as branch-not-found, not as "not initialized"
	if exitErr, ok := err.(*testutil.ExitError); ok {
		if exitErr.ExitCode != int(errors.ExitCodeBranchNotFound) {
			t.Errorf("Expected exit code %d (branch not found), got %d", errors.ExitCodeBranchNotFound, exitErr.ExitCode)
		}
	} else {
		t.Error("Expected ExitError")
	}

	// Verify the initialized repo does not report the uninitialized error
	if strings.Contains(output, "not initialized") {
		t.Errorf("Did not expect 'not initialized' error in an initialized repo, got: %s", output)
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
// 6. Verifies no error occurs when trying to delete the non-existent remote branch
func TestDeleteFeatureWithNonExistentRemote(t *testing.T) {
	t.Parallel()
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

	// Delete feature branch with remote deletion - should succeed
	_, err = testutil.RunGitFlow(t, dir, "feature", "delete", "test-feature", "--remote")
	if err != nil {
		t.Fatalf("Failed to delete remote: %v", err)
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
	t.Parallel()
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
	_, err = testutil.RunGit(t, dir, "config", "gitflow.origin", customRemote)
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
	t.Parallel()
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

// TestDeleteCleansUpBaseBranchConfig tests that delete command cleans up base branch config.
// Steps:
// 1. Sets up a test repository and initializes git-flow with defaults
// 2. Creates a feature branch (which stores base config)
// 3. Verifies base config exists
// 4. Deletes the feature branch
// 5. Verifies the base branch config is cleaned up after deletion
func TestDeleteCleansUpBaseBranchConfig(t *testing.T) {
	t.Parallel()
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Create feature branch
	output, err = testutil.RunGitFlow(t, dir, "feature", "start", "config-cleanup-test")
	if err != nil {
		t.Fatalf("Failed to start feature branch: %v\nOutput: %s", err, output)
	}

	// Verify base config exists
	_, err = testutil.RunGit(t, dir, "config", "--get", "gitflow.branch.feature/config-cleanup-test.base")
	if err != nil {
		t.Fatalf("Expected base config to exist after start: %v", err)
	}

	// Delete the feature branch
	output, err = testutil.RunGitFlow(t, dir, "feature", "delete", "config-cleanup-test")
	if err != nil {
		t.Fatalf("Failed to delete feature branch: %v\nOutput: %s", err, output)
	}

	// Verify base config is cleaned up
	_, err = testutil.RunGit(t, dir, "config", "--get", "gitflow.branch.feature/config-cleanup-test.base")
	if err == nil {
		t.Error("Expected base config to be cleaned up after delete, but it still exists")
	}

	// Verify branch is deleted
	if testutil.BranchExists(t, dir, "feature/config-cleanup-test") {
		t.Error("Expected feature branch to be deleted")
	}
}

// TestDeleteForceWithConfig tests that the gitflow.<type>.delete.force config enables force deletion.
// Steps:
// 1. Sets up a test repository and initializes git-flow
// 2. Enables force delete in config for feature branches
// 3. Creates a feature branch with unmerged changes
// 4. Deletes without --force flag (should succeed due to config)
// 5. Verifies the branch is deleted
func TestDeleteForceWithConfig(t *testing.T) {
	t.Parallel()
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Enable force delete in config
	_, err = testutil.RunGit(t, dir, "config", "gitflow.feature.delete.force", "true")
	if err != nil {
		t.Fatalf("Failed to set config: %v", err)
	}

	// Create a feature branch
	output, err = testutil.RunGitFlow(t, dir, "feature", "start", "test-force-config")
	if err != nil {
		t.Fatalf("Failed to create feature branch: %v\nOutput: %s", err, output)
	}

	// Add some changes to make it unmerged
	testutil.WriteFile(t, dir, "force-config.txt", "unmerged content")
	_, err = testutil.RunGit(t, dir, "add", "force-config.txt")
	if err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Add unmerged file")
	if err != nil {
		t.Fatalf("Failed to commit file: %v", err)
	}

	// Delete without --force flag (should succeed due to config)
	output, err = testutil.RunGitFlow(t, dir, "feature", "delete", "test-force-config")
	if err != nil {
		t.Fatalf("Expected delete to succeed with force config enabled: %v\nOutput: %s", err, output)
	}

	// Verify branch is deleted
	if testutil.BranchExists(t, dir, "feature/test-force-config") {
		t.Error("Expected feature branch to be deleted")
	}
}

// TestDeleteNoForceOverridesConfig tests that --no-force overrides the config setting.
// Steps:
// 1. Sets up a test repository and initializes git-flow
// 2. Enables force delete in config for feature branches
// 3. Creates a feature branch with unmerged changes
// 4. Deletes with --no-force flag (should fail despite config)
// 5. Verifies the branch still exists
func TestDeleteNoForceOverridesConfig(t *testing.T) {
	t.Parallel()
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Enable force delete in config
	_, err = testutil.RunGit(t, dir, "config", "gitflow.feature.delete.force", "true")
	if err != nil {
		t.Fatalf("Failed to set config: %v", err)
	}

	// Create a feature branch
	output, err = testutil.RunGitFlow(t, dir, "feature", "start", "test-no-force")
	if err != nil {
		t.Fatalf("Failed to create feature branch: %v\nOutput: %s", err, output)
	}

	// Add some changes to make it unmerged
	testutil.WriteFile(t, dir, "no-force.txt", "unmerged content")
	_, err = testutil.RunGit(t, dir, "add", "no-force.txt")
	if err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Add unmerged file")
	if err != nil {
		t.Fatalf("Failed to commit file: %v", err)
	}

	// Delete with --no-force flag (should fail despite config)
	output, err = testutil.RunGitFlow(t, dir, "feature", "delete", "test-no-force", "--no-force")
	if err == nil {
		t.Fatal("Expected delete to fail with --no-force flag")
	}

	// Verify branch still exists
	if !testutil.BranchExists(t, dir, "feature/test-no-force") {
		t.Error("Expected feature branch to still exist")
	}
}

// TestDeleteForceConfigAcrossBranchTypes tests that force config works for different branch types.
// Steps:
// 1. Sets up a test repository and initializes git-flow
// 2. Enables force delete in config for release branches
// 3. Creates a release branch with unmerged changes
// 4. Deletes without --force flag (should succeed due to config)
// 5. Verifies the branch is deleted
func TestDeleteForceConfigAcrossBranchTypes(t *testing.T) {
	t.Parallel()
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Enable force delete in config for release branches
	_, err = testutil.RunGit(t, dir, "config", "gitflow.release.delete.force", "true")
	if err != nil {
		t.Fatalf("Failed to set config: %v", err)
	}

	// Create a release branch
	output, err = testutil.RunGitFlow(t, dir, "release", "start", "1.0.0")
	if err != nil {
		t.Fatalf("Failed to create release branch: %v\nOutput: %s", err, output)
	}

	// Add some changes to make it unmerged
	testutil.WriteFile(t, dir, "release-change.txt", "release content")
	_, err = testutil.RunGit(t, dir, "add", "release-change.txt")
	if err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Add release file")
	if err != nil {
		t.Fatalf("Failed to commit file: %v", err)
	}

	// Delete without --force flag (should succeed due to config)
	output, err = testutil.RunGitFlow(t, dir, "release", "delete", "1.0.0")
	if err != nil {
		t.Fatalf("Expected delete to succeed with force config enabled: %v\nOutput: %s", err, output)
	}

	// Verify branch is deleted
	if testutil.BranchExists(t, dir, "release/1.0.0") {
		t.Error("Expected release branch to be deleted")
	}
}

// TestDeleteFeatureBranchRemoteNoRemoteError tests that delete --remote returns a clear error when no remote is configured.
// Steps:
// 1. Sets up a test repository (no remote) and initializes git-flow with defaults
// 2. Creates a feature branch with 'git flow feature start test-feature'
// 3. Adds a commit to the feature branch
// 4. Runs 'git flow feature delete test-feature --remote'
// 5. Verifies the command fails with an error
// 6. Verifies the error message mentions the missing remote (contains "No remote")
// 7. Verifies the feature branch still exists locally (not deleted despite the error)
func TestDeleteFeatureBranchRemoteNoRemoteError(t *testing.T) {
	t.Parallel()
	// Setup test repository without remote
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	_, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v", err)
	}

	// Create a feature branch
	_, err = testutil.RunGitFlow(t, dir, "feature", "start", "test-feature")
	if err != nil {
		t.Fatalf("Failed to start feature branch: %v", err)
	}

	// Add a commit so the branch has content
	testutil.WriteFile(t, dir, "test.txt", "test content")
	_, err = testutil.RunGit(t, dir, "add", "test.txt")
	if err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Add test file")
	if err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Switch to develop so we can attempt to delete the feature branch
	_, err = testutil.RunGit(t, dir, "checkout", "develop")
	if err != nil {
		t.Fatalf("Failed to checkout develop: %v", err)
	}

	// Attempt to delete with --remote (should fail with clear error)
	output, err := testutil.RunGitFlow(t, dir, "feature", "delete", "test-feature", "--remote")
	if err == nil {
		t.Fatalf("Expected error when deleting with --remote without remote configured, but command succeeded.\nOutput: %s", output)
	}

	// Verify error message mentions missing remote
	if !strings.Contains(output, "No remote") {
		t.Errorf("Expected error message to contain 'No remote', got: %s", output)
	}

	// Verify branch still exists locally (not deleted because error occurred before deletion)
	if !testutil.BranchExists(t, dir, "feature/test-feature") {
		t.Error("Expected feature/test-feature branch to still exist after failed delete --remote")
	}
}

// TestDeleteWithMissingBaseConfigNoWarning tests that deleting a feature branch
// whose base config was never stored locally (a collaborator checked out a
// published branch instead of starting it) does not print a spurious base-config
// cleanup warning.
// Steps:
// 1. Sets up a test repository and initializes git-flow with defaults
// 2. Creates a feature branch (which stores base config)
// 3. Removes the base config to simulate a checked-out (not started) branch
// 4. Deletes the feature branch
// 5. Verifies no cleanup warning is printed and the branch is deleted
func TestDeleteWithMissingBaseConfigNoWarning(t *testing.T) {
	t.Parallel()
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Create feature branch (stores base config)
	output, err = testutil.RunGitFlow(t, dir, "feature", "start", "missing-base")
	if err != nil {
		t.Fatalf("Failed to start feature branch: %v\nOutput: %s", err, output)
	}

	// Remove the base config to simulate a checked-out (not started) branch
	_, err = testutil.RunGit(t, dir, "config", "--unset", "gitflow.branch.feature/missing-base.base")
	if err != nil {
		t.Fatalf("Failed to unset base config: %v", err)
	}

	// Delete the feature branch
	output, err = testutil.RunGitFlow(t, dir, "feature", "delete", "missing-base")
	if err != nil {
		t.Fatalf("Failed to delete feature branch: %v\nOutput: %s", err, output)
	}

	// Verify no spurious cleanup warning was printed
	if strings.Contains(output, "Warning: Failed to clean up base config") {
		t.Errorf("Expected no base-config cleanup warning, but got:\n%s", output)
	}

	// Verify branch is deleted
	if testutil.BranchExists(t, dir, "feature/missing-base") {
		t.Error("Expected feature branch to be deleted")
	}
}

// TestDeleteWithMultiValueBaseConfigWarns tests that deleting a feature branch
// whose base config key holds multiple values still prints the cleanup warning,
// because a multi-value key is a genuine failure that --unset cannot resolve and
// must not be silently swallowed like a missing key.
// Steps:
// 1. Sets up a test repository and initializes git-flow with defaults
// 2. Creates a feature branch (which stores base config)
// 3. Adds a second value to the base config key
// 4. Deletes the feature branch
// 5. Verifies the cleanup warning is printed and the branch is still deleted
func TestDeleteWithMultiValueBaseConfigWarns(t *testing.T) {
	t.Parallel()
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Create feature branch (stores base config)
	output, err = testutil.RunGitFlow(t, dir, "feature", "start", "multi-base")
	if err != nil {
		t.Fatalf("Failed to start feature branch: %v\nOutput: %s", err, output)
	}

	// Add a second value so the key becomes multi-value
	_, err = testutil.RunGit(t, dir, "config", "--add", "gitflow.branch.feature/multi-base.base", "other-base")
	if err != nil {
		t.Fatalf("Failed to add second base config value: %v", err)
	}

	// Delete the feature branch
	output, err = testutil.RunGitFlow(t, dir, "feature", "delete", "multi-base")
	if err != nil {
		t.Fatalf("Failed to delete feature branch: %v\nOutput: %s", err, output)
	}

	// Verify the cleanup warning is still printed for a genuine failure
	if !strings.Contains(output, "Warning: Failed to clean up base config") {
		t.Errorf("Expected base-config cleanup warning for multi-value key, but got:\n%s", output)
	}

	// Verify branch is still deleted (warning is non-fatal)
	if testutil.BranchExists(t, dir, "feature/multi-base") {
		t.Error("Expected feature branch to be deleted")
	}
}

// TestDeleteWithBaseConfigInNonLocalScopeNoWarning tests that deleting a feature
// branch whose base config key is absent from local config but present only in a
// non-local scope (global) does not print a spurious cleanup warning. The
// presence probe must be scoped to local config — the same scope the unset
// operates on — so a global-only value is not mistaken for a local key that then
// fails to unset.
// Steps:
// 1. Isolates the global config to a temp file so the real global config is untouched
// 2. Sets up a test repository and initializes git-flow with defaults
// 3. Creates a feature branch (which stores the base config locally)
// 4. Unsets the local base key to simulate a branch not started locally
// 5. Adds the base key to the global scope only
// 6. Deletes the feature branch
// 7. Verifies no cleanup warning is printed and the branch is deleted
func TestDeleteWithBaseConfigInNonLocalScopeNoWarning(t *testing.T) {
	t.Parallel()
	// Isolate the global config to a temp file so nothing touches the
	// developer's real global config. The override is passed through each
	// subprocess env (not the test process env) so it stays scoped to this
	// test and is safe under parallel execution.
	env := []string{"GIT_CONFIG_GLOBAL=" + filepath.Join(t.TempDir(), "global-gitconfig")}

	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	output, err := runGitFlowWithEnv(t, dir, env, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Create feature branch (stores base config locally)
	output, err = runGitFlowWithEnv(t, dir, env, "feature", "start", "scoped-base")
	if err != nil {
		t.Fatalf("Failed to start feature branch: %v\nOutput: %s", err, output)
	}

	// Remove the local base key to simulate a branch not started locally
	_, err = testutil.RunGitWithEnv(t, dir, env, "config", "--local", "--unset", "gitflow.branch.feature/scoped-base.base")
	if err != nil {
		t.Fatalf("Failed to unset local base config: %v", err)
	}

	// Add the base key to the global scope only (lands in the isolated GIT_CONFIG_GLOBAL file)
	_, err = testutil.RunGitWithEnv(t, dir, env, "config", "--global", "--add", "gitflow.branch.feature/scoped-base.base", "develop")
	if err != nil {
		t.Fatalf("Failed to add global base config: %v", err)
	}

	// Delete the feature branch
	output, err = runGitFlowWithEnv(t, dir, env, "feature", "delete", "scoped-base")
	if err != nil {
		t.Fatalf("Failed to delete feature branch: %v\nOutput: %s", err, output)
	}

	// Verify no spurious cleanup warning was printed
	if strings.Contains(output, "Warning: Failed to clean up base config") {
		t.Errorf("Expected no base-config cleanup warning, but got:\n%s", output)
	}

	// Verify branch is deleted
	if testutil.BranchExists(t, dir, "feature/scoped-base") {
		t.Error("Expected feature branch to be deleted")
	}
}

// TestDeleteFeatureWithFetchFlag tests that the --fetch flag fetches before deleting.
// Steps:
// 1. Sets up a test repository with remote (includes git-flow init)
// 2. Starts a feature branch
// 3. Runs 'git flow feature delete my-feature --fetch'
// 4. Verifies output contains "Fetching from remote"
// 5. Verifies branch is deleted
func TestDeleteFeatureWithFetchFlag(t *testing.T) {
	t.Parallel()
	// Setup test repository with remote
	dir, remoteDir := testutil.SetupTestRepoWithRemote(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer testutil.CleanupTestRepo(t, remoteDir)

	// Create a feature branch
	_, err := testutil.RunGitFlow(t, dir, "feature", "start", "test-fetch")
	if err != nil {
		t.Fatalf("Failed to create feature branch: %v", err)
	}

	// Delete with --fetch flag
	output, err := testutil.RunGitFlow(t, dir, "feature", "delete", "test-fetch", "--fetch")
	if err != nil {
		t.Fatalf("Failed to delete feature branch: %v\nOutput: %s", err, output)
	}

	// Verify that fetch occurred
	if !strings.Contains(output, "Fetching from remote") {
		t.Error("Expected fetch to occur with --fetch flag")
	}

	// Verify branch is deleted
	if testutil.BranchExists(t, dir, "feature/test-fetch") {
		t.Error("Expected feature branch to be deleted")
	}
}

// TestDeleteFeatureWithFetchConfig tests that gitflow.feature.delete.fetch config triggers fetch.
// Steps:
// 1. Sets up a test repository with remote (includes git-flow init)
// 2. Starts a feature branch
// 3. Sets gitflow.feature.delete.fetch to true
// 4. Runs 'git flow feature delete my-feature' (no flag)
// 5. Verifies output contains "Fetching from remote"
// 6. Verifies branch is deleted
func TestDeleteFeatureWithFetchConfig(t *testing.T) {
	t.Parallel()
	// Setup test repository with remote
	dir, remoteDir := testutil.SetupTestRepoWithRemote(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer testutil.CleanupTestRepo(t, remoteDir)

	// Create a feature branch
	_, err := testutil.RunGitFlow(t, dir, "feature", "start", "test-fetch-config")
	if err != nil {
		t.Fatalf("Failed to create feature branch: %v", err)
	}

	// Set fetch config
	_, err = testutil.RunGit(t, dir, "config", "gitflow.feature.delete.fetch", "true")
	if err != nil {
		t.Fatalf("Failed to set config: %v", err)
	}

	// Delete without flag (should use config)
	output, err := testutil.RunGitFlow(t, dir, "feature", "delete", "test-fetch-config")
	if err != nil {
		t.Fatalf("Failed to delete feature branch: %v\nOutput: %s", err, output)
	}

	// Verify that fetch occurred
	if !strings.Contains(output, "Fetching from remote") {
		t.Error("Expected fetch to occur with fetch config enabled")
	}

	// Verify branch is deleted
	if testutil.BranchExists(t, dir, "feature/test-fetch-config") {
		t.Error("Expected feature branch to be deleted")
	}
}

// TestDeleteFeatureWithNoFetchOverridesConfig tests that --no-fetch overrides fetch config.
// Steps:
// 1. Sets up a test repository with remote (includes git-flow init)
// 2. Starts a feature branch
// 3. Sets gitflow.feature.delete.fetch to true
// 4. Runs 'git flow feature delete my-feature --no-fetch'
// 5. Verifies output does NOT contain "Fetching"
// 6. Verifies branch is deleted
func TestDeleteFeatureWithNoFetchOverridesConfig(t *testing.T) {
	t.Parallel()
	// Setup test repository with remote
	dir, remoteDir := testutil.SetupTestRepoWithRemote(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer testutil.CleanupTestRepo(t, remoteDir)

	// Create a feature branch
	_, err := testutil.RunGitFlow(t, dir, "feature", "start", "test-no-fetch")
	if err != nil {
		t.Fatalf("Failed to create feature branch: %v", err)
	}

	// Set fetch config
	_, err = testutil.RunGit(t, dir, "config", "gitflow.feature.delete.fetch", "true")
	if err != nil {
		t.Fatalf("Failed to set config: %v", err)
	}

	// Delete with --no-fetch flag (should override config)
	output, err := testutil.RunGitFlow(t, dir, "feature", "delete", "test-no-fetch", "--no-fetch")
	if err != nil {
		t.Fatalf("Failed to delete feature branch: %v\nOutput: %s", err, output)
	}

	// Verify that fetch did NOT occur
	if strings.Contains(output, "Fetching") {
		t.Error("Expected --no-fetch to override config and skip fetch")
	}

	// Verify branch is deleted
	if testutil.BranchExists(t, dir, "feature/test-no-fetch") {
		t.Error("Expected feature branch to be deleted")
	}
}

// TestDeleteFeatureNoRemoteFetchSkipped tests that --fetch silently skips when no remote exists.
// Steps:
// 1. Sets up a test repository (no remote) and initializes git-flow with defaults
// 2. Starts a feature branch
// 3. Runs 'git flow feature delete my-feature --fetch'
// 4. Verifies output does NOT contain "Fetching" (fetch skipped silently)
// 5. Verifies no error about missing remote
// 6. Verifies branch is deleted
func TestDeleteFeatureNoRemoteFetchSkipped(t *testing.T) {
	t.Parallel()
	// Setup test repository without remote
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	_, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v", err)
	}

	// Create a feature branch
	_, err = testutil.RunGitFlow(t, dir, "feature", "start", "test-no-remote")
	if err != nil {
		t.Fatalf("Failed to create feature branch: %v", err)
	}

	// Delete with --fetch flag (no remote configured)
	output, err := testutil.RunGitFlow(t, dir, "feature", "delete", "test-no-remote", "--fetch")
	if err != nil {
		t.Fatalf("Failed to delete feature branch: %v\nOutput: %s", err, output)
	}

	// Verify fetch was skipped silently
	if strings.Contains(output, "Fetching") {
		t.Error("Expected fetch to be skipped silently when no remote exists")
	}

	// Verify no confusing error messages
	if strings.Contains(output, "does not appear to be a git repository") {
		t.Error("Expected no confusing error about missing remote")
	}

	// Verify branch is deleted
	if testutil.BranchExists(t, dir, "feature/test-no-remote") {
		t.Error("Expected feature branch to be deleted")
	}
}

// setupBehindDeleteRepo sets up a repo whose local feature branch is behind its remote: it starts a
// feature branch, commits, pushes it with tracking, then advances the remote branch from a second
// clone so the local branch falls behind. It returns the working dir (already registered for
// cleanup) for the test to run delete against.
func setupBehindDeleteRepo(t *testing.T, feature string) string {
	t.Helper()

	dir, remoteDir := testutil.SetupTestRepoWithRemote(t)
	t.Cleanup(func() { testutil.CleanupTestRepo(t, dir) })
	t.Cleanup(func() { testutil.CleanupTestRepo(t, remoteDir) })

	// Create a feature branch with a commit and push it with tracking
	if _, err := testutil.RunGitFlow(t, dir, "feature", "start", feature); err != nil {
		t.Fatalf("Failed to start feature: %v", err)
	}
	if err := testutil.WriteFile(t, dir, "local.txt", "local content"); err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}
	if _, err := testutil.RunGit(t, dir, "add", "local.txt"); err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	if _, err := testutil.RunGit(t, dir, "commit", "-m", "Local feature commit"); err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}
	if _, err := testutil.RunGit(t, dir, "push", "--set-upstream", "origin", "feature/"+feature); err != nil {
		t.Fatalf("Failed to push feature: %v", err)
	}

	// In a second clone, add a commit to the remote feature branch so local falls behind
	secondDir := t.TempDir()
	if _, err := testutil.RunGit(t, secondDir, "clone", remoteDir, "."); err != nil {
		t.Fatalf("Failed to clone: %v", err)
	}
	testutil.ConfigureGitIdentity(t, secondDir)
	if _, err := testutil.RunGit(t, secondDir, "checkout", "feature/"+feature); err != nil {
		t.Fatalf("Failed to checkout feature in clone: %v", err)
	}
	if err := testutil.WriteFile(t, secondDir, "remote.txt", "remote content"); err != nil {
		t.Fatalf("Failed to write file in clone: %v", err)
	}
	if _, err := testutil.RunGit(t, secondDir, "add", "remote.txt"); err != nil {
		t.Fatalf("Failed to add file in clone: %v", err)
	}
	if _, err := testutil.RunGit(t, secondDir, "commit", "-m", "Remote feature commit"); err != nil {
		t.Fatalf("Failed to commit in clone: %v", err)
	}
	if _, err := testutil.RunGit(t, secondDir, "push", "origin", "feature/"+feature); err != nil {
		t.Fatalf("Failed to push from clone: %v", err)
	}

	return dir
}

// TestDeleteFeatureBehindRemoteAborts verifies the fetch/sync preflight blocks deleting a topic
// branch that is behind its remote, and that the abort recommends the delete-specific command (not
// finish).
// Steps:
// 1. Sets up a repo whose local feature branch is behind its remote
// 2. Runs 'git flow feature delete --fetch' and expects an abort mentioning "behind"
// 3. Confirms the error recommends 'git flow feature delete --force' and not 'finish --force'
// 4. Confirms the branch still exists
func TestDeleteFeatureBehindRemoteAborts(t *testing.T) {
	t.Parallel()
	dir := setupBehindDeleteRepo(t, "behind-del")

	// Delete with --fetch must abort because the local branch is behind its remote
	output, err := testutil.RunGitFlow(t, dir, "feature", "delete", "behind-del", "--fetch")
	if err == nil {
		t.Fatalf("Expected delete --fetch to fail when behind remote. Output: %s", output)
	}
	if !strings.Contains(output, "behind") {
		t.Errorf("Expected abort message to mention 'behind'. Output: %s", output)
	}
	// The abort must recommend the delete command, not finish (#88 operation-aware message).
	if !strings.Contains(output, "git flow feature delete --force") {
		t.Errorf("Expected abort to recommend 'git flow feature delete --force'. Output: %s", output)
	}
	if strings.Contains(output, "finish --force") {
		t.Errorf("Expected abort not to mention 'finish --force'. Output: %s", output)
	}
	if !testutil.BranchExists(t, dir, "feature/behind-del") {
		t.Error("Expected feature branch to still exist after aborted delete")
	}
}

// TestDeleteFeatureBehindRemoteForceOverrides verifies that --force overrides the sync gate and
// deletes a branch that is behind its remote.
// Steps:
// 1. Sets up a repo whose local feature branch is behind its remote
// 2. Runs 'git flow feature delete --fetch --force' and expects it to succeed
// 3. Confirms the branch is deleted
func TestDeleteFeatureBehindRemoteForceOverrides(t *testing.T) {
	t.Parallel()
	dir := setupBehindDeleteRepo(t, "behind-del-force")

	output, err := testutil.RunGitFlow(t, dir, "feature", "delete", "behind-del-force", "--fetch", "--force")
	if err != nil {
		t.Fatalf("Expected forced delete to succeed. Output: %s\nerr: %v", output, err)
	}
	if testutil.BranchExists(t, dir, "feature/behind-del-force") {
		t.Error("Expected feature branch to be deleted with --force")
	}
}

// TestDeleteFeatureNoFetchStillAbortsWhenBehind verifies that --no-fetch skips only the fetch, not
// the sync gate: a branch already known (via cached tracking data) to be behind still aborts.
// Steps:
// 1. Sets up a repo whose local feature branch is behind its remote
// 2. Runs a plain 'git fetch' so the remote-tracking ref reflects the behind state locally
// 3. Runs 'git flow feature delete --no-fetch' and expects an abort mentioning "behind"
// 4. Confirms no "Fetching" line was printed (the fetch really was skipped)
// 5. Confirms the branch still exists
func TestDeleteFeatureNoFetchStillAbortsWhenBehind(t *testing.T) {
	t.Parallel()
	dir := setupBehindDeleteRepo(t, "no-fetch-behind")

	// Update the remote-tracking ref locally without git-flow fetching, so the cached tracking
	// data already shows the branch is behind.
	if _, err := testutil.RunGit(t, dir, "fetch", "origin"); err != nil {
		t.Fatalf("Failed to fetch: %v", err)
	}

	output, err := testutil.RunGitFlow(t, dir, "feature", "delete", "no-fetch-behind", "--no-fetch")
	if err == nil {
		t.Fatalf("Expected delete --no-fetch to fail when behind remote. Output: %s", output)
	}
	if !strings.Contains(output, "behind") {
		t.Errorf("Expected abort message to mention 'behind'. Output: %s", output)
	}
	// --no-fetch must skip the fetch itself but still run the sync check.
	if strings.Contains(output, "Fetching") {
		t.Errorf("Expected --no-fetch to skip the fetch. Output: %s", output)
	}
	if !testutil.BranchExists(t, dir, "feature/no-fetch-behind") {
		t.Error("Expected feature branch to still exist after aborted delete")
	}
}

// TestDeleteFeatureDivergedRemoteAborts verifies the sync preflight blocks deleting a topic branch
// that has diverged from its remote, and that the abort recommends the delete-specific command.
// Steps:
// 1. Sets up a repo with remote, starts a feature branch, commits, and pushes it with tracking
// 2. In a second clone, advances the remote feature branch (remote gains a commit)
// 3. Adds a different local commit so the branch diverges
// 4. Runs 'git flow feature delete --fetch' and expects an abort mentioning "diverged"
// 5. Confirms the error recommends 'git flow feature delete --force' and not 'finish --force'
// 6. Confirms the branch still exists
func TestDeleteFeatureDivergedRemoteAborts(t *testing.T) {
	t.Parallel()
	dir, remoteDir := testutil.SetupTestRepoWithRemote(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer testutil.CleanupTestRepo(t, remoteDir)

	// Create a feature branch with a commit and push it with tracking
	if _, err := testutil.RunGitFlow(t, dir, "feature", "start", "diverged-del"); err != nil {
		t.Fatalf("Failed to start feature: %v", err)
	}
	if err := testutil.WriteFile(t, dir, "base.txt", "base content"); err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}
	if _, err := testutil.RunGit(t, dir, "add", "base.txt"); err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	if _, err := testutil.RunGit(t, dir, "commit", "-m", "Base feature commit"); err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}
	if _, err := testutil.RunGit(t, dir, "push", "--set-upstream", "origin", "feature/diverged-del"); err != nil {
		t.Fatalf("Failed to push feature: %v", err)
	}

	// In a second clone, add a commit to the remote feature branch
	secondDir := t.TempDir()
	if _, err := testutil.RunGit(t, secondDir, "clone", remoteDir, "."); err != nil {
		t.Fatalf("Failed to clone: %v", err)
	}
	testutil.ConfigureGitIdentity(t, secondDir)
	if _, err := testutil.RunGit(t, secondDir, "checkout", "feature/diverged-del"); err != nil {
		t.Fatalf("Failed to checkout feature in clone: %v", err)
	}
	if err := testutil.WriteFile(t, secondDir, "remote.txt", "remote content"); err != nil {
		t.Fatalf("Failed to write file in clone: %v", err)
	}
	if _, err := testutil.RunGit(t, secondDir, "add", "remote.txt"); err != nil {
		t.Fatalf("Failed to add file in clone: %v", err)
	}
	if _, err := testutil.RunGit(t, secondDir, "commit", "-m", "Remote feature commit"); err != nil {
		t.Fatalf("Failed to commit in clone: %v", err)
	}
	if _, err := testutil.RunGit(t, secondDir, "push", "origin", "feature/diverged-del"); err != nil {
		t.Fatalf("Failed to push from clone: %v", err)
	}

	// Add a different local commit so the branch diverges from the remote
	if err := testutil.WriteFile(t, dir, "local.txt", "local content"); err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}
	if _, err := testutil.RunGit(t, dir, "add", "local.txt"); err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	if _, err := testutil.RunGit(t, dir, "commit", "-m", "Local feature commit"); err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Delete with --fetch must abort because the local branch has diverged
	output, err := testutil.RunGitFlow(t, dir, "feature", "delete", "diverged-del", "--fetch")
	if err == nil {
		t.Fatalf("Expected delete --fetch to fail when diverged. Output: %s", output)
	}
	if !strings.Contains(output, "diverged") {
		t.Errorf("Expected abort message to mention 'diverged'. Output: %s", output)
	}
	if !strings.Contains(output, "git flow feature delete --force") {
		t.Errorf("Expected abort to recommend 'git flow feature delete --force'. Output: %s", output)
	}
	if strings.Contains(output, "finish --force") {
		t.Errorf("Expected abort not to mention 'finish --force'. Output: %s", output)
	}
	if !testutil.BranchExists(t, dir, "feature/diverged-del") {
		t.Error("Expected feature branch to still exist after aborted delete")
	}
}

// TestDeleteFeatureAheadRemoteTolerated verifies the delete preflight tolerates a topic branch that
// is merely ahead of its remote: unlike finish, the sync gate does not abort on ahead
// (tolerateAhead). It prints a note and lets the operation proceed to `git branch -d`.
//
// Note: a branch that is genuinely ahead of its upstream cannot complete a non-force `git branch -d`
// (Git independently refuses a branch not merged into its upstream, even when merged into HEAD), so
// this test asserts what tolerateAhead actually governs — the sync gate does not fire — rather than
// full deletion, which would require --force (which bypasses the sync check entirely).
// Steps:
//  1. Sets up a repo with remote, starts a feature branch, commits, and pushes it with tracking
//  2. Adds a local commit that is not pushed, so the branch is ahead of its remote
//  3. Runs 'git flow feature delete --fetch' and confirms the ahead state is tolerated (a note is
//     printed) and the sync gate does NOT abort (no delete/finish --force recommendation)
func TestDeleteFeatureAheadRemoteTolerated(t *testing.T) {
	t.Parallel()
	dir, remoteDir := testutil.SetupTestRepoWithRemote(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer testutil.CleanupTestRepo(t, remoteDir)

	// Create a feature branch with a commit and push it with tracking (in sync)
	if _, err := testutil.RunGitFlow(t, dir, "feature", "start", "ahead-del"); err != nil {
		t.Fatalf("Failed to start feature: %v", err)
	}
	if err := testutil.WriteFile(t, dir, "base.txt", "base content"); err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}
	if _, err := testutil.RunGit(t, dir, "add", "base.txt"); err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	if _, err := testutil.RunGit(t, dir, "commit", "-m", "Base feature commit"); err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}
	if _, err := testutil.RunGit(t, dir, "push", "--set-upstream", "origin", "feature/ahead-del"); err != nil {
		t.Fatalf("Failed to push feature: %v", err)
	}

	// Add a local commit that is not pushed, so the branch is ahead of its remote
	if err := testutil.WriteFile(t, dir, "local.txt", "local content"); err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}
	if _, err := testutil.RunGit(t, dir, "add", "local.txt"); err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	if _, err := testutil.RunGit(t, dir, "commit", "-m", "Local unpushed commit"); err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Delete with --fetch: the sync gate must tolerate the ahead state (a note, not an abort).
	// git branch -d itself still refuses a branch that is not merged into its upstream, so the
	// command as a whole fails there — not at the sync gate. Asserting that failure (rather than
	// ignoring the error) keeps this test from passing vacuously.
	output, err := testutil.RunGitFlow(t, dir, "feature", "delete", "ahead-del", "--fetch")
	if err == nil {
		t.Errorf("Expected delete to fail (git branch -d refuses an ahead branch). Output: %s", output)
	}
	// The branch must still exist: deletion did not happen.
	if !testutil.BranchExists(t, dir, "feature/ahead-del") {
		t.Error("Expected feature/ahead-del to still exist after the refused delete")
	}
	// The sync gate must have TOLERATED ahead: it printed a note...
	if !strings.Contains(output, "ahead of remote") {
		t.Errorf("Expected a tolerated 'ahead of remote' note. Output: %s", output)
	}
	// ...and did NOT abort with a sync error (that would recommend `delete --force`).
	if strings.Contains(output, "To delete anyway") || strings.Contains(output, "--force ahead-del") {
		t.Errorf("Expected the sync gate not to abort on an ahead branch. Output: %s", output)
	}
}

// TestDeleteFeatureMergedRemotely verifies the #88 scenario: a feature merged remotely (e.g. via a
// PR) can be deleted with a non-force `git branch -d` once --fetch fast-forwards the parent.
// Steps:
// 1. Sets up a repo with remote, starts a feature branch, commits, and pushes it
// 2. In a second clone, merges the feature into develop, pushes develop, deletes the remote feature
// 3. Checks out the parent and runs 'git flow feature delete --fetch' (no --force)
// 4. The fetch fast-forwards develop so `git branch -d` recognizes the branch as merged
func TestDeleteFeatureMergedRemotely(t *testing.T) {
	t.Parallel()
	dir, remoteDir := testutil.SetupTestRepoWithRemote(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer testutil.CleanupTestRepo(t, remoteDir)

	// Create a feature branch with a commit and publish it (no upstream tracking, so
	// `git branch -d` later falls back to checking HEAD)
	if _, err := testutil.RunGitFlow(t, dir, "feature", "start", "merged-remote"); err != nil {
		t.Fatalf("Failed to start feature: %v", err)
	}
	if err := testutil.WriteFile(t, dir, "feature.txt", "feature work"); err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}
	if _, err := testutil.RunGit(t, dir, "add", "feature.txt"); err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	if _, err := testutil.RunGit(t, dir, "commit", "-m", "Feature work"); err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}
	if _, err := testutil.RunGit(t, dir, "push", "origin", "feature/merged-remote"); err != nil {
		t.Fatalf("Failed to push feature: %v", err)
	}

	// Simulate a remote PR merge in a second clone: merge the feature into develop, push
	// develop, and delete the remote feature branch
	secondDir := t.TempDir()
	if _, err := testutil.RunGit(t, secondDir, "clone", remoteDir, "."); err != nil {
		t.Fatalf("Failed to clone: %v", err)
	}
	testutil.ConfigureGitIdentity(t, secondDir)
	if _, err := testutil.RunGit(t, secondDir, "checkout", "develop"); err != nil {
		t.Fatalf("Failed to checkout develop in clone: %v", err)
	}
	if _, err := testutil.RunGit(t, secondDir, "merge", "--no-ff", "-m", "Merge feature/merged-remote", "origin/feature/merged-remote"); err != nil {
		t.Fatalf("Failed to merge feature into develop: %v", err)
	}
	if _, err := testutil.RunGit(t, secondDir, "push", "origin", "develop"); err != nil {
		t.Fatalf("Failed to push develop: %v", err)
	}
	if _, err := testutil.RunGit(t, secondDir, "push", "origin", "--delete", "feature/merged-remote"); err != nil {
		t.Fatalf("Failed to delete remote feature: %v", err)
	}

	// Back in the original repo, move to the parent. Without --fetch the local develop does not
	// yet contain the merge, so a non-force delete would refuse.
	if _, err := testutil.RunGit(t, dir, "checkout", "develop"); err != nil {
		t.Fatalf("Failed to checkout develop: %v", err)
	}

	output, err := testutil.RunGitFlow(t, dir, "feature", "delete", "merged-remote", "--fetch")
	if err != nil {
		t.Fatalf("Expected delete --fetch to succeed for a remotely-merged branch. Output: %s\nerr: %v", output, err)
	}
	if !strings.Contains(output, "Fetching from remote") {
		t.Errorf("Expected fetch to occur. Output: %s", output)
	}
	if testutil.BranchExists(t, dir, "feature/merged-remote") {
		t.Error("Expected feature branch to be deleted after fast-forwarding the parent")
	}
}

// TestDeleteWithoutInitialization tests the delete command in a repository where
// git-flow has not been initialized.
// Steps:
//  1. Sets up a plain Git repository without running git flow init
//  2. Attempts to delete a feature branch
//  3. Verifies the command fails with the "not initialized" error and exit code,
//     rather than a misleading "branch does not exist" error
func TestDeleteWithoutInitialization(t *testing.T) {
	t.Parallel()
	// Setup: plain repo, git-flow NOT initialized
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Attempt to delete a feature branch without initialization
	output, err := testutil.RunGitFlow(t, dir, "feature", "delete", "foo")
	if err == nil {
		t.Fatal("Expected delete to fail without git-flow initialization, but it succeeded")
	}

	// Check exit code
	if exitErr, ok := err.(*testutil.ExitError); ok {
		if exitErr.ExitCode != int(errors.ExitCodeNotInitialized) {
			t.Errorf("Expected exit code %d, got %d", errors.ExitCodeNotInitialized, exitErr.ExitCode)
		}
	} else {
		t.Error("Expected ExitError")
	}

	// Verify error message is the "not initialized" message, not "does not exist"
	if !strings.Contains(output, "Error: git flow is not initialized") {
		t.Errorf("Expected 'not initialized' error, got: %s", output)
	}
	if strings.Contains(output, "does not exist") {
		t.Errorf("Expected no misleading 'does not exist' error, got: %s", output)
	}

	// Verify no branch was created and git-flow is still not initialized
	if testutil.BranchExists(t, dir, "feature/foo") {
		t.Error("Expected no branch to be created, but 'feature/foo' exists")
	}
	if _, err := testutil.RunGit(t, dir, "config", "--get", "gitflow.version"); err == nil {
		t.Error("Expected git-flow to still not be initialized after failed command")
	}
}

// TestDeleteHotfixWithoutInitialization tests that the delete command's
// initialization gate is branch-type agnostic by exercising a non-feature type.
// Steps:
// 1. Sets up a plain Git repository without running git flow init
// 2. Attempts to delete a hotfix branch
// 3. Verifies the command fails with the "not initialized" error and exit code
func TestDeleteHotfixWithoutInitialization(t *testing.T) {
	t.Parallel()
	// Setup: plain repo, git-flow NOT initialized
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Attempt to delete a hotfix branch without initialization
	output, err := testutil.RunGitFlow(t, dir, "hotfix", "delete", "x")
	if err == nil {
		t.Fatal("Expected delete to fail without git-flow initialization, but it succeeded")
	}

	// Check exit code
	if exitErr, ok := err.(*testutil.ExitError); ok {
		if exitErr.ExitCode != int(errors.ExitCodeNotInitialized) {
			t.Errorf("Expected exit code %d, got %d", errors.ExitCodeNotInitialized, exitErr.ExitCode)
		}
	} else {
		t.Error("Expected ExitError")
	}

	// Verify error message
	if !strings.Contains(output, "Error: git flow is not initialized") {
		t.Errorf("Expected 'not initialized' error, got: %s", output)
	}
}
