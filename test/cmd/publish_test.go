package cmd_test

import (
	"strings"
	"testing"

	"github.com/gittower/git-flow-next/internal/errors"
	"github.com/gittower/git-flow-next/test/testutil"
)

// TestPublishFeatureBranch tests publishing a feature branch to remote.
// Steps:
// 1. Sets up a test repository with a remote
// 2. Creates a feature branch
// 3. Runs 'git flow feature publish my-feature'
// 4. Verifies branch exists on remote and tracking is set up
func TestPublishFeatureBranch(t *testing.T) {
	t.Parallel()
	// Setup test repo with remote
	dir, remoteDir := testutil.SetupTestRepoWithRemote(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer testutil.CleanupTestRepo(t, remoteDir)

	// Create a feature branch
	output, err := testutil.RunGitFlow(t, dir, "feature", "start", "my-feature")
	if err != nil {
		t.Fatalf("Failed to start feature: %v\nOutput: %s", err, output)
	}

	// Publish the feature
	output, err = testutil.RunGitFlow(t, dir, "feature", "publish", "my-feature")
	if err != nil {
		t.Fatalf("Failed to publish feature: %v\nOutput: %s", err, output)
	}

	// Verify output contains success message
	if !strings.Contains(output, "Successfully published 'feature/my-feature'") {
		t.Errorf("Expected success message in output, got: %s", output)
	}

	// Verify remote branch exists by fetching
	_, err = testutil.RunGit(t, dir, "fetch", "origin")
	if err != nil {
		t.Fatalf("Failed to fetch: %v", err)
	}

	if !testutil.RemoteBranchExists(t, dir, "origin", "feature/my-feature") {
		t.Error("Expected remote branch 'origin/feature/my-feature' to exist")
	}
}

// TestPublishCurrentBranch tests publishing without specifying a name.
// Steps:
// 1. Sets up a test repository with a remote
// 2. Creates and checks out a feature branch
// 3. Runs 'git flow feature publish' without a name
// 4. Verifies current branch is published
func TestPublishCurrentBranch(t *testing.T) {
	t.Parallel()
	// Setup test repo with remote
	dir, remoteDir := testutil.SetupTestRepoWithRemote(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer testutil.CleanupTestRepo(t, remoteDir)

	// Create a feature branch (this will also check it out)
	output, err := testutil.RunGitFlow(t, dir, "feature", "start", "current-test")
	if err != nil {
		t.Fatalf("Failed to start feature: %v\nOutput: %s", err, output)
	}

	// Verify we're on the feature branch
	currentBranch := testutil.GetCurrentBranch(t, dir)
	if currentBranch != "feature/current-test" {
		t.Fatalf("Expected to be on 'feature/current-test', got '%s'", currentBranch)
	}

	// Publish without specifying a name
	output, err = testutil.RunGitFlow(t, dir, "feature", "publish")
	if err != nil {
		t.Fatalf("Failed to publish current branch: %v\nOutput: %s", err, output)
	}

	// Verify success message
	if !strings.Contains(output, "Successfully published 'feature/current-test'") {
		t.Errorf("Expected success message in output, got: %s", output)
	}

	// Verify remote branch exists
	_, err = testutil.RunGit(t, dir, "fetch", "origin")
	if err != nil {
		t.Fatalf("Failed to fetch: %v", err)
	}

	if !testutil.RemoteBranchExists(t, dir, "origin", "feature/current-test") {
		t.Error("Expected remote branch to exist")
	}
}

// TestPublishBranchNotFoundLocally tests error when branch doesn't exist.
// Steps:
// 1. Sets up a test repository with a remote
// 2. Attempts to publish a non-existent branch
// 3. Verifies appropriate error is returned
func TestPublishBranchNotFoundLocally(t *testing.T) {
	t.Parallel()
	// Setup test repo with remote
	dir, remoteDir := testutil.SetupTestRepoWithRemote(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer testutil.CleanupTestRepo(t, remoteDir)

	// Try to publish a non-existent branch
	output, err := testutil.RunGitFlow(t, dir, "feature", "publish", "non-existent")
	if err == nil {
		t.Fatal("Expected error when publishing non-existent branch, but succeeded")
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
	if !strings.Contains(output, "local branch 'feature/non-existent' does not exist") {
		t.Errorf("Expected 'local branch not found' error message, got: %s", output)
	}
}

// TestPublishBranchAlreadyExistsOnRemote tests error when remote branch exists.
// Steps:
// 1. Sets up a test repository with a remote
// 2. Creates a local branch and publishes it
// 3. Attempts to publish the same branch again
// 4. Verifies appropriate error is returned
func TestPublishBranchAlreadyExistsOnRemote(t *testing.T) {
	t.Parallel()
	// Setup test repo with remote
	dir, remoteDir := testutil.SetupTestRepoWithRemote(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer testutil.CleanupTestRepo(t, remoteDir)

	// Create and publish a feature branch
	_, err := testutil.RunGitFlow(t, dir, "feature", "start", "duplicate")
	if err != nil {
		t.Fatalf("Failed to start feature: %v", err)
	}

	_, err = testutil.RunGitFlow(t, dir, "feature", "publish", "duplicate")
	if err != nil {
		t.Fatalf("Failed to publish feature first time: %v", err)
	}

	// Try to publish the same branch again
	output, err := testutil.RunGitFlow(t, dir, "feature", "publish", "duplicate")
	if err == nil {
		t.Fatal("Expected error when publishing branch that already exists on remote, but succeeded")
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
	if !strings.Contains(output, "already exists on remote") {
		t.Errorf("Expected 'already exists on remote' error message, got: %s", output)
	}
}

// TestPublishWithCustomRemote tests publishing with custom remote configuration.
// Steps:
// 1. Sets up a test repository with custom remote configured
// 2. Creates a feature branch
// 3. Publishes the branch
// 4. Verifies branch is pushed to the custom remote
func TestPublishWithCustomRemote(t *testing.T) {
	t.Parallel()
	// Setup test repo
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow
	_, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v", err)
	}

	// Add a custom remote
	customRemoteDir, err := testutil.AddRemote(t, dir, "upstream", false)
	if err != nil {
		t.Fatalf("Failed to add custom remote: %v", err)
	}
	defer testutil.CleanupTestRepo(t, customRemoteDir)

	// Push main and develop to custom remote
	_, err = testutil.RunGit(t, dir, "push", "upstream", "main")
	if err != nil {
		t.Fatalf("Failed to push main to upstream: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "push", "upstream", "develop")
	if err != nil {
		t.Fatalf("Failed to push develop to upstream: %v", err)
	}

	// Set custom remote name in config
	_, err = testutil.RunGit(t, dir, "config", "gitflow.origin", "upstream")
	if err != nil {
		t.Fatalf("Failed to set custom remote: %v", err)
	}

	// Create a feature branch
	_, err = testutil.RunGitFlow(t, dir, "feature", "start", "custom-remote-feature")
	if err != nil {
		t.Fatalf("Failed to start feature: %v", err)
	}

	// Publish the feature
	output, err := testutil.RunGitFlow(t, dir, "feature", "publish", "custom-remote-feature")
	if err != nil {
		t.Fatalf("Failed to publish feature: %v\nOutput: %s", err, output)
	}

	// Verify output mentions custom remote
	if !strings.Contains(output, "upstream") {
		t.Errorf("Expected output to mention 'upstream' remote, got: %s", output)
	}

	// Verify branch exists on custom remote
	_, err = testutil.RunGit(t, dir, "fetch", "upstream")
	if err != nil {
		t.Fatalf("Failed to fetch from upstream: %v", err)
	}

	if !testutil.RemoteBranchExists(t, dir, "upstream", "feature/custom-remote-feature") {
		t.Error("Expected remote branch on 'upstream' to exist")
	}
}

// TestPublishReleaseBranch tests publishing a release branch.
// Steps:
// 1. Sets up a test repository with a remote
// 2. Creates a release branch
// 3. Runs 'git flow release publish' for the branch
// 4. Verifies branch exists on remote with correct prefix
func TestPublishReleaseBranch(t *testing.T) {
	t.Parallel()
	// Setup test repo with remote
	dir, remoteDir := testutil.SetupTestRepoWithRemote(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer testutil.CleanupTestRepo(t, remoteDir)

	// Create a release branch
	output, err := testutil.RunGitFlow(t, dir, "release", "start", "1.0.0")
	if err != nil {
		t.Fatalf("Failed to start release: %v\nOutput: %s", err, output)
	}

	// Publish the release
	output, err = testutil.RunGitFlow(t, dir, "release", "publish", "1.0.0")
	if err != nil {
		t.Fatalf("Failed to publish release: %v\nOutput: %s", err, output)
	}

	// Verify success message
	if !strings.Contains(output, "Successfully published 'release/1.0.0'") {
		t.Errorf("Expected success message in output, got: %s", output)
	}

	// Verify remote branch exists
	_, err = testutil.RunGit(t, dir, "fetch", "origin")
	if err != nil {
		t.Fatalf("Failed to fetch: %v", err)
	}

	if !testutil.RemoteBranchExists(t, dir, "origin", "release/1.0.0") {
		t.Error("Expected remote branch 'origin/release/1.0.0' to exist")
	}
}

// TestPublishHotfixBranch tests publishing a hotfix branch.
// Steps:
// 1. Sets up a test repository with a remote
// 2. Creates a hotfix branch
// 3. Runs 'git flow hotfix publish' for the branch
// 4. Verifies branch exists on remote with correct prefix
func TestPublishHotfixBranch(t *testing.T) {
	t.Parallel()
	// Setup test repo with remote
	dir, remoteDir := testutil.SetupTestRepoWithRemote(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer testutil.CleanupTestRepo(t, remoteDir)

	// Create a hotfix branch
	output, err := testutil.RunGitFlow(t, dir, "hotfix", "start", "1.0.1")
	if err != nil {
		t.Fatalf("Failed to start hotfix: %v\nOutput: %s", err, output)
	}

	// Publish the hotfix
	output, err = testutil.RunGitFlow(t, dir, "hotfix", "publish", "1.0.1")
	if err != nil {
		t.Fatalf("Failed to publish hotfix: %v\nOutput: %s", err, output)
	}

	// Verify success message
	if !strings.Contains(output, "Successfully published 'hotfix/1.0.1'") {
		t.Errorf("Expected success message in output, got: %s", output)
	}

	// Verify remote branch exists
	_, err = testutil.RunGit(t, dir, "fetch", "origin")
	if err != nil {
		t.Fatalf("Failed to fetch: %v", err)
	}

	if !testutil.RemoteBranchExists(t, dir, "origin", "hotfix/1.0.1") {
		t.Error("Expected remote branch 'origin/hotfix/1.0.1' to exist")
	}
}

// TestPublishWithFullBranchName tests publishing with full branch name (including prefix).
// Steps:
// 1. Sets up a test repository with a remote
// 2. Creates a feature branch
// 3. Publishes using full name 'feature/my-feature'
// 4. Verifies branch is published correctly (without double prefix)
func TestPublishWithFullBranchName(t *testing.T) {
	t.Parallel()
	// Setup test repo with remote
	dir, remoteDir := testutil.SetupTestRepoWithRemote(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer testutil.CleanupTestRepo(t, remoteDir)

	// Create a feature branch
	_, err := testutil.RunGitFlow(t, dir, "feature", "start", "full-name-test")
	if err != nil {
		t.Fatalf("Failed to start feature: %v", err)
	}

	// Publish using full branch name (including prefix)
	output, err := testutil.RunGitFlow(t, dir, "feature", "publish", "feature/full-name-test")
	if err != nil {
		t.Fatalf("Failed to publish feature with full name: %v\nOutput: %s", err, output)
	}

	// Verify success message shows correct branch name (not double prefix)
	if !strings.Contains(output, "Successfully published 'feature/full-name-test'") {
		t.Errorf("Expected correct branch name in success message, got: %s", output)
	}

	// Verify there's no double prefix branch created
	_, err = testutil.RunGit(t, dir, "fetch", "origin")
	if err != nil {
		t.Fatalf("Failed to fetch: %v", err)
	}

	// Should exist with single prefix
	if !testutil.RemoteBranchExists(t, dir, "origin", "feature/full-name-test") {
		t.Error("Expected remote branch 'origin/feature/full-name-test' to exist")
	}

	// Should NOT exist with double prefix
	if testutil.RemoteBranchExists(t, dir, "origin", "feature/feature/full-name-test") {
		t.Error("Did not expect remote branch with double prefix 'origin/feature/feature/full-name-test'")
	}
}

// TestPublishWithoutInitialization tests the publish command without git-flow initialization.
// Steps:
// 1. Sets up a test repository without initializing git-flow
// 2. Attempts to publish a branch
// 3. Verifies appropriate error is returned
func TestPublishWithoutInitialization(t *testing.T) {
	t.Parallel()
	// Setup test repo without git-flow init
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Try to publish without git-flow initialization
	output, err := testutil.RunGitFlow(t, dir, "feature", "publish", "some-feature")
	if err == nil {
		t.Fatal("Expected error when publishing without git-flow initialization, but succeeded")
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
	if !strings.Contains(output, "git flow is not initialized") {
		t.Errorf("Expected 'not initialized' error message, got: %s", output)
	}
}

// TestPublishCurrentBranchWrongType tests error when publishing current branch of wrong type.
// Steps:
// 1. Sets up a test repository with a remote
// 2. Creates a feature branch but tries to publish as release
// 3. Verifies appropriate error is returned
func TestPublishCurrentBranchWrongType(t *testing.T) {
	t.Parallel()
	// Setup test repo with remote
	dir, remoteDir := testutil.SetupTestRepoWithRemote(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer testutil.CleanupTestRepo(t, remoteDir)

	// Create a feature branch
	_, err := testutil.RunGitFlow(t, dir, "feature", "start", "wrong-type-test")
	if err != nil {
		t.Fatalf("Failed to start feature: %v", err)
	}

	// Verify we're on the feature branch
	currentBranch := testutil.GetCurrentBranch(t, dir)
	if currentBranch != "feature/wrong-type-test" {
		t.Fatalf("Expected to be on 'feature/wrong-type-test', got '%s'", currentBranch)
	}

	// Try to publish as release (wrong type) without specifying name
	output, err := testutil.RunGitFlow(t, dir, "release", "publish")
	if err == nil {
		t.Fatal("Expected error when publishing current branch with wrong type, but succeeded")
	}

	// Verify error message mentions the branch type mismatch
	if !strings.Contains(output, "not a release branch") {
		t.Errorf("Expected 'not a release branch' error message, got: %s", output)
	}
}

// TestPublishFeatureBranchNoRemoteError tests that publish returns a clear error when no remote is configured.
// Steps:
// 1. Sets up a test repository (no remote) and initializes git-flow with defaults
// 2. Creates a feature branch with 'git flow feature start test-feature'
// 3. Runs 'git flow feature publish test-feature'
// 4. Verifies the command fails with an error
// 5. Verifies the error message mentions the missing remote (contains "No remote")
// 6. Verifies the feature branch still exists locally (not affected by the error)
func TestPublishFeatureBranchNoRemoteError(t *testing.T) {
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

	// Attempt to publish (should fail with clear error)
	output, err := testutil.RunGitFlow(t, dir, "feature", "publish", "test-feature")
	if err == nil {
		t.Fatalf("Expected error when publishing without remote, but command succeeded.\nOutput: %s", output)
	}

	// Verify error message mentions missing remote
	if !strings.Contains(output, "No remote") {
		t.Errorf("Expected error message to contain 'No remote', got: %s", output)
	}

	// Verify branch still exists locally
	if !testutil.BranchExists(t, dir, "feature/test-feature") {
		t.Error("Expected feature/test-feature branch to still exist after failed publish")
	}
}
