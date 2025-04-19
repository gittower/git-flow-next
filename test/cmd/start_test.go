package cmd_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/gittower/git-flow-next/errors"
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

// TestStartWithoutFetch tests the default behavior (no fetch)
func TestStartWithoutFetch(t *testing.T) {
	// Setup test repo
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Run git-flow feature start without the fetch flag
	output, err = testutil.RunGitFlow(t, dir, "feature", "start", "no-fetch-test")
	if err != nil {
		t.Fatalf("Failed to run git-flow feature start: %v\nOutput: %s", err, output)
	}

	// Verify that output does not contain fetching info
	if strings.Contains(output, "Fetching from") {
		t.Errorf("Expected no fetch operation, but output indicates fetching: %s", output)
	}
}

// TestStartWithFetchFlag tests that the --fetch flag works
func TestStartWithFetchFlag(t *testing.T) {
	// Setup test repo
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Run git-flow feature start with the fetch flag
	output, err = testutil.RunGitFlow(t, dir, "feature", "start", "fetch-test", "--fetch")
	if err != nil {
		t.Fatalf("Failed to run git-flow feature start: %v\nOutput: %s", err, output)
	}

	// Verify that output contains fetching info
	if !strings.Contains(output, "Fetching from") {
		t.Errorf("Expected fetch operation, but output doesn't indicate fetching: %s", output)
	}
}

// TestStartWithFetchConfig tests that the gitflow.<topic>.start.fetch config works
func TestStartWithFetchConfig(t *testing.T) {
	// Setup test repo
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Set the config to enable fetch
	_, err = testutil.RunGit(t, dir, "config", "gitflow.feature.start.fetch", "true")
	if err != nil {
		t.Fatalf("Failed to set config: %v", err)
	}

	// Run git-flow feature start without explicit fetch flag
	output, err = testutil.RunGitFlow(t, dir, "feature", "start", "config-fetch-test")
	if err != nil {
		t.Fatalf("Failed to run git-flow feature start: %v\nOutput: %s", err, output)
	}

	// Verify that output contains fetching info
	if !strings.Contains(output, "Fetching from") {
		t.Errorf("Expected fetch operation due to config, but output doesn't indicate fetching: %s", output)
	}
}

// TestStartWithNoFetchOverridesConfig tests that --no-fetch overrides the config
func TestStartWithNoFetchOverridesConfig(t *testing.T) {
	// Setup test repo
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Set the config to enable fetch
	_, err = testutil.RunGit(t, dir, "config", "gitflow.feature.start.fetch", "true")
	if err != nil {
		t.Fatalf("Failed to set config: %v", err)
	}

	// Run git-flow feature start with --no-fetch to override config
	output, err = testutil.RunGitFlow(t, dir, "feature", "start", "no-fetch-override-test", "--no-fetch")
	if err != nil {
		t.Fatalf("Failed to run git-flow feature start: %v\nOutput: %s", err, output)
	}

	// Verify that output does not contain fetching info
	if strings.Contains(output, "Fetching from") {
		t.Errorf("Expected no fetch operation due to --no-fetch flag, but output indicates fetching: %s", output)
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

	// Set custom remote name
	customRemote := "custom-remote"
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
