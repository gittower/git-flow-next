package cmd_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/gittower/git-flow-next/internal/errors"
	"github.com/gittower/git-flow-next/test/testutil"
)

// TestStartFeatureBranch tests the start command for feature branches
func TestStartFeatureBranch(t *testing.T) {
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Run git-flow feature start my-feature
	output, err = testutil.RunGitFlow(t, dir, "feature", "start", "my-feature")
	if err != nil {
		t.Fatalf("Failed to run git-flow feature start: %v\nOutput: %s", err, output)
	}

	// Check if the output contains the expected message
	if !strings.Contains(output, "Created branch 'feature/my-feature'") {
		t.Errorf("Expected output to contain 'Created branch 'feature/my-feature'', got: %s", output)
	}

	// Check if the branch was actually created
	if !testutil.BranchExists(t, dir, "feature/my-feature") {
		t.Errorf("Expected 'feature/my-feature' branch to exist")
	}

	// Check if the branch is based on develop
	_, err = testutil.RunGit(t, dir, "merge-base", "--is-ancestor", "develop", "feature/my-feature")
	if err != nil {
		t.Errorf("Expected 'feature/my-feature' to be based on 'develop'")
	}
}

// TestStartReleaseAndHotfixBranches tests the start command for release and hotfix branches
func TestStartReleaseAndHotfixBranches(t *testing.T) {
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Run git-flow release start 1.0.0
	output, err = testutil.RunGitFlow(t, dir, "release", "start", "1.0.0")
	if err != nil {
		t.Fatalf("Failed to run git-flow release start: %v\nOutput: %s", err, output)
	}

	// Check if the output contains the expected message
	if !strings.Contains(output, "Created branch 'release/1.0.0'") {
		t.Errorf("Expected output to contain 'Created branch 'release/1.0.0'', got: %s", output)
	}

	// Check if the branch was actually created
	if !testutil.BranchExists(t, dir, "release/1.0.0") {
		t.Errorf("Expected 'release/1.0.0' branch to exist")
	}

	// Run git-flow hotfix start 1.0.1
	output, err = testutil.RunGitFlow(t, dir, "hotfix", "start", "1.0.1")
	if err != nil {
		t.Fatalf("Failed to run git-flow hotfix start: %v\nOutput: %s", err, output)
	}

	// Check if the output contains the expected message
	if !strings.Contains(output, "Created branch 'hotfix/1.0.1'") {
		t.Errorf("Expected output to contain 'Created branch 'hotfix/1.0.1'', got: %s", output)
	}

	// Check if the branch was actually created
	if !testutil.BranchExists(t, dir, "hotfix/1.0.1") {
		t.Errorf("Expected 'hotfix/1.0.1' branch to exist")
	}
}

// TestStartWithCustomConfig tests the start command with custom configuration
func TestStartWithCustomConfig(t *testing.T) {
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with custom configuration
	input := "custom-main\ncustom-dev\nf/\nr/\nh/\ns/\n"
	output, err := testutil.RunGitFlowWithInput(t, dir, input, "init")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Run git-flow feature start my-feature
	output, err = testutil.RunGitFlow(t, dir, "feature", "start", "my-feature")
	if err != nil {
		t.Fatalf("Failed to run git-flow feature start: %v\nOutput: %s", err, output)
	}

	// Check if the output contains the expected message
	if !strings.Contains(output, "Created branch 'f/my-feature'") {
		t.Errorf("Expected output to contain 'Created branch 'f/my-feature'', got: %s", output)
	}

	// Check if the branch was actually created
	if !testutil.BranchExists(t, dir, "f/my-feature") {
		t.Errorf("Expected 'f/my-feature' branch to exist")
	}

	// Check if the branch is based on custom-dev
	_, err = testutil.RunGit(t, dir, "merge-base", "--is-ancestor", "custom-dev", "f/my-feature")
	if err != nil {
		t.Errorf("Expected 'f/my-feature' to be based on 'custom-dev'")
	}
}

// TestStartWithExistingBranch tests the start command with an existing branch
func TestStartWithExistingBranch(t *testing.T) {
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Create a feature branch
	output, err = testutil.RunGitFlow(t, dir, "feature", "start", "my-feature")
	if err != nil {
		t.Fatalf("Failed to create feature branch: %v\nOutput: %s", err, output)
	}

	// Try to create the same feature branch again
	output, err = testutil.RunGitFlow(t, dir, "feature", "start", "my-feature")
	if err == nil {
		t.Error("Expected command to fail with existing branch, but it succeeded")
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
	expectedError := "Error: branch 'feature/my-feature' already exists"
	if !strings.Contains(output, expectedError) {
		t.Errorf("Expected error message to contain '%s', got: %s", expectedError, output)
	}
}

// TestStartWithNonExistentStartPoint tests the start command with a non-existent start point
func TestStartWithNonExistentStartPoint(t *testing.T) {
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Delete the develop branch to make it non-existent
	_, err = testutil.RunGit(t, dir, "checkout", "main")
	if err != nil {
		t.Fatalf("Failed to switch to main branch: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "branch", "-D", "develop")
	if err != nil {
		t.Fatalf("Failed to delete develop branch: %v", err)
	}

	// Try to create a feature branch
	output, err = testutil.RunGitFlow(t, dir, "feature", "start", "my-feature")
	if err == nil {
		t.Error("Expected command to fail when start point doesn't exist, but it succeeded")
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
	expectedError := "Error: start point branch 'develop' does not exist"
	if !strings.Contains(output, expectedError) {
		t.Errorf("Expected error message to contain '%s', got: %s", expectedError, output)
	}
}

// TestStartWithNoStartPoint tests that when start point is not specified, parent branch is used
func TestStartWithNoStartPoint(t *testing.T) {
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with custom configuration
	input := "main\ndevelop\nf/\nr/\nh/\ns/\n"
	output, err := testutil.RunGitFlowWithInput(t, dir, input, "init")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Create a feature branch
	output, err = testutil.RunGitFlow(t, dir, "feature", "start", "test-feature")
	if err != nil {
		t.Fatalf("Failed to create feature branch: %v\nOutput: %s", err, output)
	}

	// Verify that the branch was created from develop (parent branch)
	if !strings.Contains(output, "Created branch 'f/test-feature' from 'develop'") {
		t.Errorf("Expected branch to be created from 'develop', got: %s", output)
	}

	// Verify that the branch exists
	output, err = testutil.RunGit(t, dir, "branch", "--list", "f/test-feature")
	if err != nil {
		t.Fatalf("Failed to list branches: %v\nOutput: %s", err, output)
	}
	if !strings.Contains(output, "f/test-feature") {
		t.Errorf("Expected branch 'f/test-feature' to exist, got: %s", output)
	}

	// Get the commit hash of develop
	developHash, err := testutil.RunGit(t, dir, "rev-parse", "develop")
	if err != nil {
		t.Fatalf("Failed to get develop hash: %v\nOutput: %s", err, output)
	}

	// Get the commit hash of the feature branch
	featureHash, err := testutil.RunGit(t, dir, "rev-parse", "f/test-feature")
	if err != nil {
		t.Fatalf("Failed to get feature hash: %v\nOutput: %s", err, output)
	}

	// Verify that the feature branch was created from develop
	if developHash != featureHash {
		t.Errorf("Expected feature branch to be at the same commit as develop")
	}
}

// TestStartWithEmptyBranchName tests the start command with an empty branch name
func TestStartWithEmptyBranchName(t *testing.T) {
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Try to create a feature branch with empty name
	output, err = testutil.RunGitFlow(t, dir, "feature", "start", "")
	if err == nil {
		t.Error("Expected command to fail with empty branch name, but it succeeded")
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
	expectedError := "Error: branch name cannot be empty"
	if !strings.Contains(output, expectedError) {
		t.Errorf("Expected error message to contain '%s', got: %s", expectedError, output)
	}
}

// TestStartWithInvalidBranchType tests the start command with an invalid branch type
func TestStartWithInvalidBranchType(t *testing.T) {
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Try to start a branch with an invalid type
	output, err = testutil.RunGitFlow(t, dir, "invalid", "start", "test")
	if err == nil {
		t.Fatal("Expected error when using invalid branch type")
	}

	// Verify error code (Cobra's default exit code for unknown command is 1)
	if exitErr, ok := err.(*testutil.ExitError); !ok || exitErr.ExitCode != 1 {
		t.Errorf("Expected exit code 1, got %v", err)
	}

	// Verify error message matches Cobra's unknown command error
	expectedError := "Error: unknown command \"invalid\" for \"git-flow\""
	if !strings.Contains(output, expectedError) {
		t.Errorf("Expected error message to contain '%s', got: %s", expectedError, output)
	}

	// Also verify that Cobra's help suggestion is included
	if !strings.Contains(output, "Run 'git-flow --help' for usage") {
		t.Errorf("Expected error message to contain help suggestion, got: %s", output)
	}
}

// TestStartWithoutInitialization tests the start command without git-flow initialization
func TestStartWithoutInitialization(t *testing.T) {
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Verify git-flow is not initialized
	output, err := testutil.RunGit(t, dir, "config", "--get", "gitflow.initialized")
	if err == nil {
		t.Error("Expected git-flow to not be initialized, but it is")
	}

	// Try to create a feature branch without initializing git-flow
	output, err = testutil.RunGitFlow(t, dir, "feature", "start", "my-feature")
	if err == nil {
		t.Error("Expected command to fail without git-flow initialization, but it succeeded")
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
	expectedError := "Error: git flow is not initialized"
	if !strings.Contains(output, expectedError) {
		t.Errorf("Expected error message to contain '%s', got: %s", expectedError, output)
	}

	// Verify no branch was created
	if testutil.BranchExists(t, dir, "feature/my-feature") {
		t.Error("Expected no branch to be created, but 'feature/my-feature' exists")
	}

	// Verify git-flow is still not initialized
	_, err = testutil.RunGit(t, dir, "config", "--get", "gitflow.initialized")
	if err == nil {
		t.Error("Expected git-flow to still not be initialized after failed command")
	}

	// Verify only the default branch exists
	branches, err := testutil.RunGit(t, dir, "branch")
	if err != nil {
		t.Fatalf("Failed to list branches: %v", err)
	}
	expectedBranches := []string{"main", "master"}
	foundExpectedBranch := false
	for _, expectedBranch := range expectedBranches {
		if strings.Contains(branches, expectedBranch) {
			foundExpectedBranch = true
			break
		}
	}
	if !foundExpectedBranch {
		t.Errorf("Expected to find one of %v branches, but got: %s", expectedBranches, branches)
	}
	if strings.Contains(branches, "feature/") {
		t.Error("Found unexpected feature branch")
	}
	if strings.Contains(branches, "develop") {
		t.Error("Found unexpected develop branch")
	}
}

// TestStartWithoutFetch tests Scenario 19: the default start behavior does not fetch (default false).
// Uses a remote-backed fixture so the absence of "Fetching" reflects the default, not a missing remote.
// Steps:
// 1. Sets up a test repository with a remote and initializes git-flow
// 2. Runs 'git flow feature start' with no fetch flag and no fetch config
// 3. Verifies no "Fetching" line appears
// 4. Verifies the feature branch is created
func TestStartWithoutFetch(t *testing.T) {
	// Setup test repo with remote so absence of fetch reflects the default (not a missing remote)
	dir, remoteDir := testutil.SetupTestRepoWithRemote(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer testutil.CleanupTestRepo(t, remoteDir)

	// Run git-flow feature start without the fetch flag (default is no fetch)
	output, err := testutil.RunGitFlow(t, dir, "feature", "start", "no-fetch-test")
	if err != nil {
		t.Fatalf("Failed to run git-flow feature start: %v\nOutput: %s", err, output)
	}

	// Verify that output does not contain fetching info
	if strings.Contains(output, "Fetching from") {
		t.Errorf("Expected no fetch operation, but output indicates fetching: %s", output)
	}

	// Verify the branch was created
	if !testutil.BranchExists(t, dir, "feature/no-fetch-test") {
		t.Error("Expected feature/no-fetch-test branch to exist")
	}
}

// TestStartWithFetchFlag tests that the --fetch flag works
func TestStartWithFetchFlag(t *testing.T) {
	// Setup test repo with remote (includes git-flow init)
	dir, remoteDir := testutil.SetupTestRepoWithRemote(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer testutil.CleanupTestRepo(t, remoteDir)

	// Run git-flow feature start with the fetch flag
	output, err := testutil.RunGitFlow(t, dir, "feature", "start", "fetch-test", "--fetch")
	if err != nil {
		t.Fatalf("Failed to run git-flow feature start: %v\nOutput: %s", err, output)
	}

	// Verify that output contains fetching info
	if !strings.Contains(output, "Fetching from") {
		t.Errorf("Expected fetch operation, but output doesn't indicate fetching: %s", output)
	}
}

// TestStartWithFetchConfig tests Scenario 20: gitflow.<topic>.start.fetch=true drives a fetch.
// Steps:
// 1. Sets up a test repository with a remote and initializes git-flow
// 2. Sets gitflow.feature.start.fetch=true
// 3. Runs 'git flow feature start' with no explicit fetch flag
// 4. Verifies a "Fetching" line appears
// 5. Verifies the feature branch is created
func TestStartWithFetchConfig(t *testing.T) {
	// Setup test repo with remote (includes git-flow init)
	dir, remoteDir := testutil.SetupTestRepoWithRemote(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer testutil.CleanupTestRepo(t, remoteDir)

	// Set the config to enable fetch
	_, err := testutil.RunGit(t, dir, "config", "gitflow.feature.start.fetch", "true")
	if err != nil {
		t.Fatalf("Failed to set config: %v", err)
	}

	// Run git-flow feature start without explicit fetch flag
	output, err := testutil.RunGitFlow(t, dir, "feature", "start", "config-fetch-test")
	if err != nil {
		t.Fatalf("Failed to run git-flow feature start: %v\nOutput: %s", err, output)
	}

	// Verify that output contains fetching info
	if !strings.Contains(output, "Fetching from") {
		t.Errorf("Expected fetch operation due to config, but output doesn't indicate fetching: %s", output)
	}

	// Verify the branch was created
	if !testutil.BranchExists(t, dir, "feature/config-fetch-test") {
		t.Error("Expected feature/config-fetch-test branch to exist")
	}
}

// TestStartWithNoFetchOverridesConfig tests Scenario 21: --no-fetch overrides start.fetch=true.
// Uses a remote-backed fixture so the absence of "Fetching" reflects the flag overriding the config,
// not a missing remote.
// Steps:
// 1. Sets up a test repository with a remote and initializes git-flow
// 2. Sets gitflow.feature.start.fetch=true
// 3. Runs 'git flow feature start' with --no-fetch
// 4. Verifies no "Fetching" line appears (flag overrides config)
// 5. Verifies the feature branch is created
func TestStartWithNoFetchOverridesConfig(t *testing.T) {
	// Setup test repo with remote so absence of fetch reflects the flag, not a missing remote
	dir, remoteDir := testutil.SetupTestRepoWithRemote(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer testutil.CleanupTestRepo(t, remoteDir)

	// Set the config to enable fetch
	_, err := testutil.RunGit(t, dir, "config", "gitflow.feature.start.fetch", "true")
	if err != nil {
		t.Fatalf("Failed to set config: %v", err)
	}

	// Run git-flow feature start with --no-fetch to override config
	output, err := testutil.RunGitFlow(t, dir, "feature", "start", "no-fetch-override-test", "--no-fetch")
	if err != nil {
		t.Fatalf("Failed to run git-flow feature start: %v\nOutput: %s", err, output)
	}

	// Verify that output does not contain fetching info
	if strings.Contains(output, "Fetching from") {
		t.Errorf("Expected no fetch operation due to --no-fetch flag, but output indicates fetching: %s", output)
	}

	// Verify the branch was created
	if !testutil.BranchExists(t, dir, "feature/no-fetch-override-test") {
		t.Error("Expected feature/no-fetch-override-test branch to exist")
	}
}

// TestStartWithCustomRemote tests that the custom remote name is used for fetching
func TestStartWithCustomRemote(t *testing.T) {
	// Setup test repo
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Add a remote with custom name
	customRemote := "custom-remote"
	remoteDir, err := testutil.AddRemote(t, dir, customRemote, true)
	if err != nil {
		t.Fatalf("Failed to add custom remote: %v", err)
	}
	defer testutil.CleanupTestRepo(t, remoteDir)

	// Set custom remote name in gitflow config
	_, err = testutil.RunGit(t, dir, "config", "gitflow.origin", customRemote)
	if err != nil {
		t.Fatalf("Failed to set custom remote: %v", err)
	}

	// Run git-flow feature start with the fetch flag
	output, err = testutil.RunGitFlow(t, dir, "feature", "start", "custom-remote-test", "--fetch")
	if err != nil {
		t.Fatalf("Failed to run git-flow feature start: %v\nOutput: %s", err, output)
	}

	// Verify that output contains fetching from custom remote
	if !strings.Contains(output, fmt.Sprintf("Fetching from %s", customRemote)) {
		t.Errorf("Expected fetch operation from custom remote '%s', but output doesn't indicate it: %s", customRemote, output)
	}
}

// TestStartStoresBaseBranch tests that the start command stores the base branch in git config.
// Steps:
// 1. Sets up a test repository and initializes git-flow with defaults
// 2. Runs 'git flow feature start test-base-storage'
// 3. Verifies that gitflow.branch.feature/test-base-storage.base is set to 'develop'
// 4. Tests with release branch to verify base is set to 'develop' (start point)
func TestStartStoresBaseBranch(t *testing.T) {
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Create feature branch
	output, err = testutil.RunGitFlow(t, dir, "feature", "start", "test-base-storage")
	if err != nil {
		t.Fatalf("Failed to start feature branch: %v\nOutput: %s", err, output)
	}

	// Verify base branch is stored in config
	baseConfig, err := testutil.RunGit(t, dir, "config", "--get", "gitflow.branch.feature/test-base-storage.base")
	if err != nil {
		t.Fatalf("Failed to get base config: %v", err)
	}

	expectedBase := "develop"
	if strings.TrimSpace(baseConfig) != expectedBase {
		t.Errorf("Expected base branch to be '%s', got '%s'", expectedBase, strings.TrimSpace(baseConfig))
	}

	// Test with release branch (should store 'develop' as base, even though parent is 'main')
	output, err = testutil.RunGitFlow(t, dir, "release", "start", "1.0.0")
	if err != nil {
		t.Fatalf("Failed to start release branch: %v\nOutput: %s", err, output)
	}

	// Verify release base branch is stored as 'develop' (start point)
	releaseBaseConfig, err := testutil.RunGit(t, dir, "config", "--get", "gitflow.branch.release/1.0.0.base")
	if err != nil {
		t.Fatalf("Failed to get release base config: %v", err)
	}

	expectedReleaseBase := "develop"
	if strings.TrimSpace(releaseBaseConfig) != expectedReleaseBase {
		t.Errorf("Expected release base branch to be '%s', got '%s'", expectedReleaseBase, strings.TrimSpace(releaseBaseConfig))
	}
}

// TestStartFeatureBranchNoRemoteFetchSkipped tests that start skips fetch silently when no remote exists.
// Steps:
// 1. Sets up a test repository (no remote) and initializes git-flow with defaults
// 2. Enables fetch for start via git config gitflow.feature.start.fetch true
// 3. Runs 'git flow feature start no-remote-test'
// 4. Verifies output does NOT contain "Fetching" (fetch skipped silently)
// 5. Verifies output does NOT contain "does not appear to be a git repository"
// 6. Verifies feature/no-remote-test branch is created successfully
func TestStartFeatureBranchNoRemoteFetchSkipped(t *testing.T) {
	// Setup test repository without remote
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	_, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v", err)
	}

	// Enable fetch for start
	_, err = testutil.RunGit(t, dir, "config", "gitflow.feature.start.fetch", "true")
	if err != nil {
		t.Fatalf("Failed to set fetch config: %v", err)
	}

	// Start a feature branch (fetch should be silently skipped)
	output, err := testutil.RunGitFlow(t, dir, "feature", "start", "no-remote-test")
	if err != nil {
		t.Fatalf("Failed to start feature branch: %v\nOutput: %s", err, output)
	}

	// Verify fetch was skipped silently
	if strings.Contains(output, "Fetching") {
		t.Error("Expected fetch to be skipped silently, but output contains 'Fetching'")
	}
	if strings.Contains(output, "does not appear to be a git repository") {
		t.Error("Expected no confusing error messages about missing remote")
	}

	// Verify branch was created
	if !testutil.BranchExists(t, dir, "feature/no-remote-test") {
		t.Error("Expected feature/no-remote-test branch to exist")
	}
}

// TestStartFetchesAdvancedStartPoint tests Scenario 14: with start.fetch=true and a reachable
// remote whose start point has advanced, start fetches and the advance becomes visible locally.
// The advance is made from a second clone so the primary's origin/develop is not updated before
// start runs — otherwise the test would pass without ever fetching.
// Steps:
// 1. Sets up a test repository with a remote and initializes git-flow; enables start.fetch
// 2. Records OLD = the primary's origin/develop
// 3. Advances develop from a second clone (commit + push); records NEW = the bare remote develop
// 4. Asserts the primary's origin/develop is still OLD and the bare develop is NEW (out of sync)
// 5. Runs 'git flow feature start'
// 6. Verifies a "Fetching" line, the branch is created, and origin/develop now resolves to NEW
func TestStartFetchesAdvancedStartPoint(t *testing.T) {
	dir, remoteDir := testutil.SetupTestRepoWithRemote(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer testutil.CleanupTestRepo(t, remoteDir)

	// Enable fetch for start
	if _, err := testutil.RunGit(t, dir, "config", "gitflow.feature.start.fetch", "true"); err != nil {
		t.Fatalf("Failed to set fetch config: %v", err)
	}

	// Record OLD = the primary's origin/develop before advancing
	oldRaw, err := testutil.RunGit(t, dir, "rev-parse", "origin/develop")
	if err != nil {
		t.Fatalf("Failed to get origin/develop: %v", err)
	}
	old := strings.TrimSpace(oldRaw)

	// Advance develop from a second clone
	secondDir := t.TempDir()
	if _, err := testutil.RunGit(t, secondDir, "clone", remoteDir, "."); err != nil {
		t.Fatalf("Failed to clone: %v", err)
	}
	testutil.ConfigureGitIdentity(t, secondDir)
	if _, err := testutil.RunGit(t, secondDir, "checkout", "develop"); err != nil {
		t.Fatalf("Failed to checkout develop in second repo: %v", err)
	}
	testutil.WriteFile(t, secondDir, "advance.txt", "advanced content")
	if _, err := testutil.RunGit(t, secondDir, "add", "advance.txt"); err != nil {
		t.Fatalf("Failed to add file in second repo: %v", err)
	}
	if _, err := testutil.RunGit(t, secondDir, "commit", "-m", "Advance develop"); err != nil {
		t.Fatalf("Failed to commit in second repo: %v", err)
	}
	if _, err := testutil.RunGit(t, secondDir, "push", "origin", "develop"); err != nil {
		t.Fatalf("Failed to push from second repo: %v", err)
	}

	// Record NEW = the bare remote develop
	newRaw, err := testutil.RunGit(t, remoteDir, "rev-parse", "refs/heads/develop")
	if err != nil {
		t.Fatalf("Failed to get bare develop: %v", err)
	}
	newSHA := strings.TrimSpace(newRaw)

	// Precondition: primary origin/develop still OLD, bare develop is NEW, and they differ
	primaryBeforeRaw, err := testutil.RunGit(t, dir, "rev-parse", "origin/develop")
	if err != nil {
		t.Fatalf("Failed to get primary origin/develop: %v", err)
	}
	if strings.TrimSpace(primaryBeforeRaw) != old {
		t.Fatalf("Precondition failed: expected primary origin/develop to still be OLD (%s), got %s", old, strings.TrimSpace(primaryBeforeRaw))
	}
	if old == newSHA {
		t.Fatalf("Precondition failed: expected OLD (%s) != NEW (%s)", old, newSHA)
	}

	// Run start
	output, err := testutil.RunGitFlow(t, dir, "feature", "start", "advanced-test")
	if err != nil {
		t.Fatalf("Failed to run git-flow feature start: %v\nOutput: %s", err, output)
	}

	// Verify fetch ran
	if !strings.Contains(output, "Fetching from") {
		t.Errorf("Expected a fetch to run. Output: %s", output)
	}

	// Verify branch created
	if !testutil.BranchExists(t, dir, "feature/advanced-test") {
		t.Error("Expected feature/advanced-test branch to exist")
	}

	// Verify the advance is now visible: primary origin/develop == NEW
	afterRaw, err := testutil.RunGit(t, dir, "rev-parse", "origin/develop")
	if err != nil {
		t.Fatalf("Failed to get origin/develop after start: %v", err)
	}
	if strings.TrimSpace(afterRaw) != newSHA {
		t.Errorf("Expected origin/develop to advance to NEW (%s) after start, got %s", newSHA, strings.TrimSpace(afterRaw))
	}
}

// TestStartUnreachableRemoteWarnsAndCreates tests Scenario 15: with start.fetch=true and an
// unreachable remote, the fetch failure is a non-fatal warning and the branch is still created.
// RunGitFlow uses CombinedOutput, so the warning is asserted on the combined output.
// Steps:
// 1. Sets up a test repository with a remote and initializes git-flow; enables start.fetch
// 2. Points origin at a nonexistent path
// 3. Runs 'git flow feature start'
// 4. Verifies the command still succeeds, the combined output contains a Warning, and the branch exists
func TestStartUnreachableRemoteWarnsAndCreates(t *testing.T) {
	dir, remoteDir := testutil.SetupTestRepoWithRemote(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer testutil.CleanupTestRepo(t, remoteDir)

	if _, err := testutil.RunGit(t, dir, "config", "gitflow.feature.start.fetch", "true"); err != nil {
		t.Fatalf("Failed to set fetch config: %v", err)
	}
	if _, err := testutil.RunGit(t, dir, "remote", "set-url", "origin", "/nonexistent/repo.git"); err != nil {
		t.Fatalf("Failed to break remote: %v", err)
	}

	output, err := testutil.RunGitFlow(t, dir, "feature", "start", "unreachable-test")
	if err != nil {
		t.Fatalf("Expected start to succeed despite fetch failure. Error: %v\nOutput: %s", err, output)
	}

	// Verify a warning about the failed fetch (combined output)
	if !strings.Contains(output, "Warning:") {
		t.Errorf("Expected a Warning about the failed fetch. Output: %s", output)
	}

	// Verify branch created
	if !testutil.BranchExists(t, dir, "feature/unreachable-test") {
		t.Error("Expected feature/unreachable-test branch to exist")
	}
}
