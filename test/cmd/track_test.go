package cmd_test

import (
	"strings"
	"testing"

	"github.com/gittower/git-flow-next/internal/errors"
	"github.com/gittower/git-flow-next/test/testutil"
)

// TestTrackFeatureBranch tests tracking a remote feature branch.
// Steps:
// 1. Sets up a test repository with a remote
// 2. Creates a feature branch on the remote (simulating another user)
// 3. Runs 'git flow feature track' for the remote branch
// 4. Verifies local tracking branch is created
func TestTrackFeatureBranch(t *testing.T) {
	t.Parallel()
	// Setup test repository with remote
	dir, remoteDir := testutil.SetupTestRepoWithRemote(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer testutil.CleanupTestRepo(t, remoteDir)

	// Create a feature branch on the "remote" by pushing from local
	_, err := testutil.RunGitFlow(t, dir, "feature", "start", "remote-feature")
	if err != nil {
		t.Fatalf("Failed to create feature branch: %v", err)
	}

	// Push the feature branch to remote
	_, err = testutil.RunGit(t, dir, "push", "-u", "origin", "feature/remote-feature")
	if err != nil {
		t.Fatalf("Failed to push feature branch to remote: %v", err)
	}

	// Delete the local branch to simulate tracking a branch from another user
	_, err = testutil.RunGit(t, dir, "checkout", "develop")
	if err != nil {
		t.Fatalf("Failed to checkout develop: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "branch", "-D", "feature/remote-feature")
	if err != nil {
		t.Fatalf("Failed to delete local feature branch: %v", err)
	}

	// Verify the local branch no longer exists
	if testutil.BranchExists(t, dir, "feature/remote-feature") {
		t.Fatal("Expected local feature branch to be deleted")
	}

	// Track the remote feature branch
	output, err := testutil.RunGitFlow(t, dir, "feature", "track", "remote-feature")
	if err != nil {
		t.Fatalf("Failed to track feature branch: %v\nOutput: %s", err, output)
	}

	// Verify success message
	if !strings.Contains(output, "Successfully created tracking branch") {
		t.Errorf("Expected success message, got: %s", output)
	}

	// Verify local tracking branch was created
	if !testutil.BranchExists(t, dir, "feature/remote-feature") {
		t.Error("Expected local tracking branch to exist")
	}

	// Verify we're on the tracking branch
	currentBranch := testutil.GetCurrentBranch(t, dir)
	if currentBranch != "feature/remote-feature" {
		t.Errorf("Expected to be on 'feature/remote-feature', got '%s'", currentBranch)
	}

	// Verify the branch is tracking the remote
	trackingInfo, err := testutil.RunGit(t, dir, "rev-parse", "--abbrev-ref", "feature/remote-feature@{upstream}")
	if err != nil {
		t.Fatalf("Failed to get tracking info: %v", err)
	}
	if strings.TrimSpace(trackingInfo) != "origin/feature/remote-feature" {
		t.Errorf("Expected branch to track 'origin/feature/remote-feature', got '%s'", strings.TrimSpace(trackingInfo))
	}
}

// TestTrackReleaseBranch tests tracking a remote release branch.
// Steps:
// 1. Sets up a test repository with a remote
// 2. Creates a release branch on the remote
// 3. Runs 'git flow release track' for the remote branch
// 4. Verifies local tracking branch is created with correct prefix
func TestTrackReleaseBranch(t *testing.T) {
	t.Parallel()
	// Setup test repository with remote
	dir, remoteDir := testutil.SetupTestRepoWithRemote(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer testutil.CleanupTestRepo(t, remoteDir)

	// Create a release branch
	_, err := testutil.RunGitFlow(t, dir, "release", "start", "1.0.0")
	if err != nil {
		t.Fatalf("Failed to create release branch: %v", err)
	}

	// Push the release branch to remote
	_, err = testutil.RunGit(t, dir, "push", "-u", "origin", "release/1.0.0")
	if err != nil {
		t.Fatalf("Failed to push release branch to remote: %v", err)
	}

	// Delete the local branch
	_, err = testutil.RunGit(t, dir, "checkout", "develop")
	if err != nil {
		t.Fatalf("Failed to checkout develop: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "branch", "-D", "release/1.0.0")
	if err != nil {
		t.Fatalf("Failed to delete local release branch: %v", err)
	}

	// Track the remote release branch
	output, err := testutil.RunGitFlow(t, dir, "release", "track", "1.0.0")
	if err != nil {
		t.Fatalf("Failed to track release branch: %v\nOutput: %s", err, output)
	}

	// Verify local tracking branch was created
	if !testutil.BranchExists(t, dir, "release/1.0.0") {
		t.Error("Expected local tracking branch to exist")
	}

	// Verify we're on the tracking branch
	currentBranch := testutil.GetCurrentBranch(t, dir)
	if currentBranch != "release/1.0.0" {
		t.Errorf("Expected to be on 'release/1.0.0', got '%s'", currentBranch)
	}
}

// TestTrackBranchAlreadyExistsLocally tests error when branch exists locally.
// Steps:
// 1. Sets up a test repository with git-flow initialized
// 2. Creates a local feature branch
// 3. Pushes it to remote
// 4. Attempts to track the same branch
// 5. Verifies appropriate error is returned
func TestTrackBranchAlreadyExistsLocally(t *testing.T) {
	t.Parallel()
	// Setup test repository with remote
	dir, remoteDir := testutil.SetupTestRepoWithRemote(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer testutil.CleanupTestRepo(t, remoteDir)

	// Create a feature branch
	_, err := testutil.RunGitFlow(t, dir, "feature", "start", "existing-feature")
	if err != nil {
		t.Fatalf("Failed to create feature branch: %v", err)
	}

	// Push the feature branch to remote
	_, err = testutil.RunGit(t, dir, "push", "-u", "origin", "feature/existing-feature")
	if err != nil {
		t.Fatalf("Failed to push feature branch to remote: %v", err)
	}

	// Try to track the branch that already exists locally
	output, err := testutil.RunGitFlow(t, dir, "feature", "track", "existing-feature")
	if err == nil {
		t.Fatal("Expected error when tracking existing local branch, but command succeeded")
	}

	// Check exit code
	if exitErr, ok := err.(*testutil.ExitError); ok {
		if exitErr.ExitCode != int(errors.ExitCodeBranchExists) {
			t.Errorf("Expected exit code %d, got %d", errors.ExitCodeBranchExists, exitErr.ExitCode)
		}
	} else {
		t.Error("Expected ExitError")
	}

	// Verify error message
	expectedError := "branch 'feature/existing-feature' already exists"
	if !strings.Contains(output, expectedError) {
		t.Errorf("Expected error message to contain '%s', got: %s", expectedError, output)
	}
}

// TestTrackBranchNotFoundOnRemote tests error when branch doesn't exist on remote.
// Steps:
// 1. Sets up a test repository with a remote
// 2. Attempts to track a non-existent remote branch
// 3. Verifies appropriate error is returned
func TestTrackBranchNotFoundOnRemote(t *testing.T) {
	t.Parallel()
	// Setup test repository with remote
	dir, remoteDir := testutil.SetupTestRepoWithRemote(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer testutil.CleanupTestRepo(t, remoteDir)

	// Try to track a non-existent remote branch
	output, err := testutil.RunGitFlow(t, dir, "feature", "track", "non-existent")
	if err == nil {
		t.Fatal("Expected error when tracking non-existent branch, but command succeeded")
	}

	// Check exit code
	if exitErr, ok := err.(*testutil.ExitError); ok {
		if exitErr.ExitCode != int(errors.ExitCodeBranchNotFound) {
			t.Errorf("Expected exit code %d, got %d", errors.ExitCodeBranchNotFound, exitErr.ExitCode)
		}
	} else {
		t.Error("Expected ExitError")
	}

	// Verify error message
	expectedError := "branch 'feature/non-existent' not found on remote 'origin'"
	if !strings.Contains(output, expectedError) {
		t.Errorf("Expected error message to contain '%s', got: %s", expectedError, output)
	}
}

// TestTrackWithCustomRemote tests tracking with custom remote configuration.
// Steps:
// 1. Sets up a test repository with a custom remote name
// 2. Creates a branch on the custom remote
// 3. Tracks the branch using the configured remote
// 4. Verifies tracking is set up correctly
func TestTrackWithCustomRemote(t *testing.T) {
	t.Parallel()
	// Setup test repository
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow
	_, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v", err)
	}

	// Create a custom remote
	customRemoteDir, err := testutil.AddRemote(t, dir, "upstream", false)
	if err != nil {
		t.Fatalf("Failed to add custom remote: %v", err)
	}
	defer testutil.CleanupTestRepo(t, customRemoteDir)

	// Push branches to custom remote
	_, err = testutil.RunGit(t, dir, "push", "upstream", "main")
	if err != nil {
		t.Fatalf("Failed to push main to upstream: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "push", "upstream", "develop")
	if err != nil {
		t.Fatalf("Failed to push develop to upstream: %v", err)
	}

	// Configure git-flow to use custom remote
	_, err = testutil.RunGit(t, dir, "config", "gitflow.origin", "upstream")
	if err != nil {
		t.Fatalf("Failed to set custom remote: %v", err)
	}

	// Create a feature branch and push to custom remote
	_, err = testutil.RunGitFlow(t, dir, "feature", "start", "custom-remote-feature")
	if err != nil {
		t.Fatalf("Failed to create feature branch: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "push", "-u", "upstream", "feature/custom-remote-feature")
	if err != nil {
		t.Fatalf("Failed to push feature branch to upstream: %v", err)
	}

	// Delete local branch
	_, err = testutil.RunGit(t, dir, "checkout", "develop")
	if err != nil {
		t.Fatalf("Failed to checkout develop: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "branch", "-D", "feature/custom-remote-feature")
	if err != nil {
		t.Fatalf("Failed to delete local feature branch: %v", err)
	}

	// Track the remote feature branch using custom remote
	output, err := testutil.RunGitFlow(t, dir, "feature", "track", "custom-remote-feature")
	if err != nil {
		t.Fatalf("Failed to track feature branch: %v\nOutput: %s", err, output)
	}

	// Verify output mentions the custom remote
	if !strings.Contains(output, "upstream") {
		t.Errorf("Expected output to mention 'upstream' remote, got: %s", output)
	}

	// Verify local tracking branch was created
	if !testutil.BranchExists(t, dir, "feature/custom-remote-feature") {
		t.Error("Expected local tracking branch to exist")
	}

	// Verify the branch is tracking the custom remote
	trackingInfo, err := testutil.RunGit(t, dir, "rev-parse", "--abbrev-ref", "feature/custom-remote-feature@{upstream}")
	if err != nil {
		t.Fatalf("Failed to get tracking info: %v", err)
	}
	if strings.TrimSpace(trackingInfo) != "upstream/feature/custom-remote-feature" {
		t.Errorf("Expected branch to track 'upstream/feature/custom-remote-feature', got '%s'", strings.TrimSpace(trackingInfo))
	}
}

// TestTrackWithFullBranchName tests tracking with full branch name (including prefix).
// Steps:
// 1. Sets up a test repository with a remote
// 2. Creates a feature branch on the remote
// 3. Runs 'git flow feature track feature/full-name' (with prefix)
// 4. Verifies local tracking branch is created correctly
func TestTrackWithFullBranchName(t *testing.T) {
	t.Parallel()
	// Setup test repository with remote
	dir, remoteDir := testutil.SetupTestRepoWithRemote(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer testutil.CleanupTestRepo(t, remoteDir)

	// Create a feature branch
	_, err := testutil.RunGitFlow(t, dir, "feature", "start", "full-name-test")
	if err != nil {
		t.Fatalf("Failed to create feature branch: %v", err)
	}

	// Push the feature branch to remote
	_, err = testutil.RunGit(t, dir, "push", "-u", "origin", "feature/full-name-test")
	if err != nil {
		t.Fatalf("Failed to push feature branch to remote: %v", err)
	}

	// Delete the local branch
	_, err = testutil.RunGit(t, dir, "checkout", "develop")
	if err != nil {
		t.Fatalf("Failed to checkout develop: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "branch", "-D", "feature/full-name-test")
	if err != nil {
		t.Fatalf("Failed to delete local feature branch: %v", err)
	}

	// Track the remote feature branch using full name (with prefix)
	output, err := testutil.RunGitFlow(t, dir, "feature", "track", "feature/full-name-test")
	if err != nil {
		t.Fatalf("Failed to track feature branch: %v\nOutput: %s", err, output)
	}

	// Verify local tracking branch was created (not feature/feature/full-name-test)
	if !testutil.BranchExists(t, dir, "feature/full-name-test") {
		t.Error("Expected local tracking branch to exist")
	}

	// Verify we didn't create a double-prefixed branch
	if testutil.BranchExists(t, dir, "feature/feature/full-name-test") {
		t.Error("Created double-prefixed branch, which is incorrect")
	}
}

// TestTrackWithoutInitialization tests the track command without git-flow initialization.
// Steps:
// 1. Sets up a test repository without git-flow initialized
// 2. Attempts to track a branch
// 3. Verifies appropriate error is returned
func TestTrackWithoutInitialization(t *testing.T) {
	t.Parallel()
	// Setup test repository
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Try to track a branch without initializing git-flow
	output, err := testutil.RunGitFlow(t, dir, "feature", "track", "some-feature")
	if err == nil {
		t.Fatal("Expected error when tracking without initialization, but command succeeded")
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
	expectedError := "git flow is not initialized"
	if !strings.Contains(output, expectedError) {
		t.Errorf("Expected error message to contain '%s', got: %s", expectedError, output)
	}
}

// TestTrackWithEmptyBranchName tests the track command with an empty branch name.
// Steps:
// 1. Sets up a test repository with git-flow initialized
// 2. Attempts to track with empty name
// 3. Verifies appropriate error is returned
func TestTrackWithEmptyBranchName(t *testing.T) {
	t.Parallel()
	// Setup test repository with remote
	dir, remoteDir := testutil.SetupTestRepoWithRemote(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer testutil.CleanupTestRepo(t, remoteDir)

	// Try to track with empty branch name
	output, err := testutil.RunGitFlow(t, dir, "feature", "track", "")
	if err == nil {
		t.Fatal("Expected error when tracking with empty name, but command succeeded")
	}

	// Check exit code
	if exitErr, ok := err.(*testutil.ExitError); ok {
		if exitErr.ExitCode != int(errors.ExitCodeInvalidInput) {
			t.Errorf("Expected exit code %d, got %d", errors.ExitCodeInvalidInput, exitErr.ExitCode)
		}
	} else {
		t.Error("Expected ExitError")
	}

	// Verify error message
	expectedError := "branch name cannot be empty"
	if !strings.Contains(output, expectedError) {
		t.Errorf("Expected error message to contain '%s', got: %s", expectedError, output)
	}
}

// TestTrackHotfixBranch tests tracking a remote hotfix branch.
// Steps:
// 1. Sets up a test repository with a remote
// 2. Creates a hotfix branch on the remote
// 3. Runs 'git flow hotfix track' for the remote branch
// 4. Verifies local tracking branch is created
func TestTrackHotfixBranch(t *testing.T) {
	t.Parallel()
	// Setup test repository with remote
	dir, remoteDir := testutil.SetupTestRepoWithRemote(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer testutil.CleanupTestRepo(t, remoteDir)

	// Create a hotfix branch
	_, err := testutil.RunGitFlow(t, dir, "hotfix", "start", "1.0.1")
	if err != nil {
		t.Fatalf("Failed to create hotfix branch: %v", err)
	}

	// Push the hotfix branch to remote
	_, err = testutil.RunGit(t, dir, "push", "-u", "origin", "hotfix/1.0.1")
	if err != nil {
		t.Fatalf("Failed to push hotfix branch to remote: %v", err)
	}

	// Delete the local branch
	_, err = testutil.RunGit(t, dir, "checkout", "develop")
	if err != nil {
		t.Fatalf("Failed to checkout develop: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "branch", "-D", "hotfix/1.0.1")
	if err != nil {
		t.Fatalf("Failed to delete local hotfix branch: %v", err)
	}

	// Track the remote hotfix branch
	output, err := testutil.RunGitFlow(t, dir, "hotfix", "track", "1.0.1")
	if err != nil {
		t.Fatalf("Failed to track hotfix branch: %v\nOutput: %s", err, output)
	}

	// Verify local tracking branch was created
	if !testutil.BranchExists(t, dir, "hotfix/1.0.1") {
		t.Error("Expected local tracking branch to exist")
	}

	// Verify we're on the tracking branch
	currentBranch := testutil.GetCurrentBranch(t, dir)
	if currentBranch != "hotfix/1.0.1" {
		t.Errorf("Expected to be on 'hotfix/1.0.1', got '%s'", currentBranch)
	}
}

// TestTrackFeatureBranchNoRemoteError tests that track returns a clear error when no remote is configured.
// Steps:
// 1. Sets up a test repository (no remote) and initializes git-flow with defaults
// 2. Runs 'git flow feature track some-feature'
// 3. Verifies the command fails with an error
// 4. Verifies the error message mentions the missing remote (contains "No remote")
func TestTrackFeatureBranchNoRemoteError(t *testing.T) {
	t.Parallel()
	// Setup test repository without remote
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	_, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v", err)
	}

	// Attempt to track (should fail with clear error)
	output, err := testutil.RunGitFlow(t, dir, "feature", "track", "some-feature")
	if err == nil {
		t.Fatalf("Expected error when tracking without remote, but command succeeded.\nOutput: %s", output)
	}

	// Verify error message mentions missing remote
	if !strings.Contains(output, "No remote") {
		t.Errorf("Expected error message to contain 'No remote', got: %s", output)
	}
}
