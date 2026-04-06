package cmd_test

import (
	"strings"
	"testing"

	"github.com/gittower/git-flow-next/test/testutil"
)

// TestFinishFeatureBranchWithFetchFlag tests finishing a feature branch with the --fetch flag.
// Steps:
// 1. Sets up a test repository and initializes git-flow
// 2. Creates a local bare remote repository
// 3. Pushes branches to establish tracking
// 4. Creates a feature branch
// 5. Adds a test file to the feature branch
// 6. Commits changes to the feature branch
// 7. Finishes the feature branch with --fetch flag
// 8. Verifies fetch message appears in output
// 9. Verifies the branch is merged into develop
// 10. Verifies the feature branch is deleted
func TestFinishFeatureBranchWithFetchFlag(t *testing.T) {
	// Setup test repository with remote
	dir, remoteDir := testutil.SetupTestRepoWithRemote(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer testutil.CleanupTestRepo(t, remoteDir)

	// Create a feature branch
	_, err := testutil.RunGitFlow(t, dir, "feature", "start", "test-fetch")
	if err != nil {
		t.Fatalf("Failed to create feature branch: %v", err)
	}

	// Create a test file
	testutil.WriteFile(t, dir, "fetch-test.txt", "test content")

	// Commit the changes
	_, err = testutil.RunGit(t, dir, "add", "fetch-test.txt")
	if err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Add fetch test file")
	if err != nil {
		t.Fatalf("Failed to commit file: %v", err)
	}

	// Finish with --fetch flag
	output, err := testutil.RunGitFlow(t, dir, "feature", "finish", "test-fetch", "--fetch")
	if err != nil {
		t.Fatalf("Failed to finish feature branch with --fetch: %v\nOutput: %s", err, output)
	}

	// Verify that fetch occurred
	if !strings.Contains(output, "Fetching from remote") {
		t.Error("Expected fetch to occur with --fetch flag")
	}

	// Verify that feature branch is deleted
	if testutil.BranchExists(t, dir, "feature/test-fetch") {
		t.Error("Expected feature branch to be deleted")
	}

	// Verify that changes are merged into develop
	_, err = testutil.RunGit(t, dir, "checkout", "develop")
	if err != nil {
		t.Fatalf("Failed to checkout develop: %v", err)
	}

	if !testutil.FileExists(t, dir, "fetch-test.txt") {
		t.Error("Expected fetch-test.txt to exist in develop branch")
	}
}

// TestFinishFeatureBranchWithNoFetchFlag tests finishing a feature branch with the --no-fetch flag.
// Steps:
// 1. Sets up a test repository and initializes git-flow
// 2. Creates a feature branch
// 3. Adds a test file to the feature branch
// 4. Commits changes to the feature branch
// 5. Finishes the feature branch with --no-fetch flag
// 6. Verifies no fetch message appears in output
// 7. Verifies the branch is merged into develop
// 8. Verifies the feature branch is deleted
func TestFinishFeatureBranchWithNoFetchFlag(t *testing.T) {
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	_, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v", err)
	}

	// Create a feature branch
	_, err = testutil.RunGitFlow(t, dir, "feature", "start", "test-no-fetch")
	if err != nil {
		t.Fatalf("Failed to create feature branch: %v", err)
	}

	// Create a test file
	testutil.WriteFile(t, dir, "no-fetch-test.txt", "test content")

	// Commit the changes
	_, err = testutil.RunGit(t, dir, "add", "no-fetch-test.txt")
	if err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Add no-fetch test file")
	if err != nil {
		t.Fatalf("Failed to commit file: %v", err)
	}

	// Finish with --no-fetch flag
	output, err := testutil.RunGitFlow(t, dir, "feature", "finish", "test-no-fetch", "--no-fetch")
	if err != nil {
		t.Fatalf("Failed to finish feature branch with --no-fetch: %v\nOutput: %s", err, output)
	}

	// Verify that no fetch occurred (check output doesn't mention fetch)
	if strings.Contains(output, "Fetching from remote") {
		t.Error("Expected no fetch to occur with --no-fetch flag")
	}

	// Verify that feature branch is deleted
	if testutil.BranchExists(t, dir, "feature/test-no-fetch") {
		t.Error("Expected feature branch to be deleted")
	}

	// Verify that changes are merged into develop
	_, err = testutil.RunGit(t, dir, "checkout", "develop")
	if err != nil {
		t.Fatalf("Failed to checkout develop: %v", err)
	}

	if !testutil.FileExists(t, dir, "no-fetch-test.txt") {
		t.Error("Expected no-fetch-test.txt to exist in develop branch")
	}
}

// TestFinishFeatureBranchDefaultFetch tests the default finish behavior (fetch enabled).
// Steps:
// 1. Sets up a test repository with remote and initializes git-flow
// 2. Creates a feature branch
// 3. Adds a test file to the feature branch
// 4. Commits changes to the feature branch
// 5. Finishes the feature branch without fetch flags
// 6. Verifies fetch message appears in output (default is fetch)
// 7. Verifies the branch is merged into develop
// 8. Verifies the feature branch is deleted
func TestFinishFeatureBranchDefaultFetch(t *testing.T) {
	// Setup test repository with remote (needed to verify fetch occurs)
	dir, remoteDir := testutil.SetupTestRepoWithRemote(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer testutil.CleanupTestRepo(t, remoteDir)

	// Create a feature branch
	_, err := testutil.RunGitFlow(t, dir, "feature", "start", "test-default")
	if err != nil {
		t.Fatalf("Failed to create feature branch: %v", err)
	}

	// Create a test file
	testutil.WriteFile(t, dir, "default-test.txt", "test content")

	// Commit the changes
	_, err = testutil.RunGit(t, dir, "add", "default-test.txt")
	if err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Add default test file")
	if err != nil {
		t.Fatalf("Failed to commit file: %v", err)
	}

	// Finish without any fetch flags (default behavior)
	output, err := testutil.RunGitFlow(t, dir, "feature", "finish", "test-default")
	if err != nil {
		t.Fatalf("Failed to finish feature branch: %v\nOutput: %s", err, output)
	}

	// Verify that fetch occurred (default is fetch)
	if !strings.Contains(output, "Fetching from remote") {
		t.Error("Expected fetch to occur by default")
	}

	// Verify that feature branch is deleted
	if testutil.BranchExists(t, dir, "feature/test-default") {
		t.Error("Expected feature branch to be deleted")
	}

	// Verify that changes are merged into develop
	_, err = testutil.RunGit(t, dir, "checkout", "develop")
	if err != nil {
		t.Fatalf("Failed to checkout develop: %v", err)
	}

	if !testutil.FileExists(t, dir, "default-test.txt") {
		t.Error("Expected default-test.txt to exist in develop branch")
	}
}

// TestFinishFeatureBranchNoFetchFromConfig tests finishing with fetch disabled via git config.
// Steps:
// 1. Sets up a test repository and initializes git-flow
// 2. Configures gitflow.feature.finish.fetch = false in git config
// 3. Creates a feature branch
// 4. Adds a test file to the feature branch
// 5. Commits changes to the feature branch
// 6. Finishes the feature branch without flags
// 7. Verifies no fetch message appears in output (config disabled fetch)
// 8. Verifies the branch is merged into develop
// 9. Verifies the feature branch is deleted
func TestFinishFeatureBranchNoFetchFromConfig(t *testing.T) {
	// Setup test repository (no remote needed since we're testing no-fetch)
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	_, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v", err)
	}

	// Configure fetch to be disabled for feature finish
	_, err = testutil.RunGit(t, dir, "config", "gitflow.feature.finish.fetch", "false")
	if err != nil {
		t.Fatalf("Failed to configure fetch option: %v", err)
	}

	// Create a feature branch
	_, err = testutil.RunGitFlow(t, dir, "feature", "start", "test-config-no-fetch")
	if err != nil {
		t.Fatalf("Failed to create feature branch: %v", err)
	}

	// Create a test file
	testutil.WriteFile(t, dir, "config-no-fetch-test.txt", "test content")

	// Commit the changes
	_, err = testutil.RunGit(t, dir, "add", "config-no-fetch-test.txt")
	if err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Add config no-fetch test file")
	if err != nil {
		t.Fatalf("Failed to commit file: %v", err)
	}

	// Finish without flags (should use config setting)
	output, err := testutil.RunGitFlow(t, dir, "feature", "finish", "test-config-no-fetch")
	if err != nil {
		t.Fatalf("Failed to finish feature branch: %v\nOutput: %s", err, output)
	}

	// Verify that no fetch occurred based on config
	if strings.Contains(output, "Fetching from remote") {
		t.Error("Expected no fetch to occur based on config setting")
	}

	// Verify that feature branch is deleted
	if testutil.BranchExists(t, dir, "feature/test-config-no-fetch") {
		t.Error("Expected feature branch to be deleted")
	}

	// Verify that changes are merged into develop
	_, err = testutil.RunGit(t, dir, "checkout", "develop")
	if err != nil {
		t.Fatalf("Failed to checkout develop: %v", err)
	}

	if !testutil.FileExists(t, dir, "config-no-fetch-test.txt") {
		t.Error("Expected config-no-fetch-test.txt to exist in develop branch")
	}
}

// TestFinishFeatureBranchNoFetchFlagOverridesConfig tests that --no-fetch overrides config.
// Steps:
// 1. Sets up a test repository and initializes git-flow
// 2. Configures gitflow.feature.finish.fetch = true in git config
// 3. Creates a feature branch
// 4. Adds a test file to the feature branch
// 5. Commits changes to the feature branch
// 6. Finishes the feature branch with --no-fetch flag
// 7. Verifies no fetch message appears (flag overrides config)
// 8. Verifies the branch is merged into develop
// 9. Verifies the feature branch is deleted
func TestFinishFeatureBranchNoFetchFlagOverridesConfig(t *testing.T) {
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	_, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v", err)
	}

	// Configure fetch to be enabled for feature finish
	_, err = testutil.RunGit(t, dir, "config", "gitflow.feature.finish.fetch", "true")
	if err != nil {
		t.Fatalf("Failed to configure fetch option: %v", err)
	}

	// Create a feature branch
	_, err = testutil.RunGitFlow(t, dir, "feature", "start", "test-override-no-fetch")
	if err != nil {
		t.Fatalf("Failed to create feature branch: %v", err)
	}

	// Create a test file
	testutil.WriteFile(t, dir, "override-no-fetch-test.txt", "test content")

	// Commit the changes
	_, err = testutil.RunGit(t, dir, "add", "override-no-fetch-test.txt")
	if err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Add override no-fetch test file")
	if err != nil {
		t.Fatalf("Failed to commit file: %v", err)
	}

	// Finish with --no-fetch flag (should override config)
	output, err := testutil.RunGitFlow(t, dir, "feature", "finish", "test-override-no-fetch", "--no-fetch")
	if err != nil {
		t.Fatalf("Failed to finish feature branch with --no-fetch: %v\nOutput: %s", err, output)
	}

	// Verify that no fetch occurred (flag overrides config)
	if strings.Contains(output, "Fetching from remote") {
		t.Error("Expected --no-fetch flag to override config setting")
	}

	// Verify that feature branch is deleted
	if testutil.BranchExists(t, dir, "feature/test-override-no-fetch") {
		t.Error("Expected feature branch to be deleted")
	}

	// Verify that changes are merged into develop
	_, err = testutil.RunGit(t, dir, "checkout", "develop")
	if err != nil {
		t.Fatalf("Failed to checkout develop: %v", err)
	}

	if !testutil.FileExists(t, dir, "override-no-fetch-test.txt") {
		t.Error("Expected override-no-fetch-test.txt to exist in develop branch")
	}
}

// TestFinishFeatureBranchFetchFlagOverridesConfig tests that --fetch overrides config.
// Steps:
// 1. Sets up a test repository and initializes git-flow
// 2. Creates a local bare remote repository
// 3. Pushes branches to establish tracking
// 4. Configures gitflow.feature.finish.fetch = false in git config
// 5. Creates a feature branch
// 6. Adds a test file to the feature branch
// 7. Commits changes to the feature branch
// 8. Finishes the feature branch with --fetch flag
// 9. Verifies fetch message appears in output (flag overrides config)
// 10. Verifies the branch is merged into develop
// 11. Verifies the feature branch is deleted
func TestFinishFeatureBranchFetchFlagOverridesConfig(t *testing.T) {
	// Setup test repository with remote
	dir, remoteDir := testutil.SetupTestRepoWithRemote(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer testutil.CleanupTestRepo(t, remoteDir)

	// Configure fetch to be disabled for feature finish
	_, err := testutil.RunGit(t, dir, "config", "gitflow.feature.finish.fetch", "false")
	if err != nil {
		t.Fatalf("Failed to configure fetch option: %v", err)
	}

	// Create a feature branch
	_, err = testutil.RunGitFlow(t, dir, "feature", "start", "test-override-fetch")
	if err != nil {
		t.Fatalf("Failed to create feature branch: %v", err)
	}

	// Create a test file
	testutil.WriteFile(t, dir, "override-fetch-test.txt", "test content")

	// Commit the changes
	_, err = testutil.RunGit(t, dir, "add", "override-fetch-test.txt")
	if err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Add override fetch test file")
	if err != nil {
		t.Fatalf("Failed to commit file: %v", err)
	}

	// Finish with --fetch flag (should override config)
	output, err := testutil.RunGitFlow(t, dir, "feature", "finish", "test-override-fetch", "--fetch")
	if err != nil {
		t.Fatalf("Failed to finish feature branch: %v\nOutput: %s", err, output)
	}

	// Verify that fetch occurred (flag overrides config)
	if !strings.Contains(output, "Fetching from remote") {
		t.Error("Expected --fetch flag to override config setting and perform fetch")
	}

	// Verify that feature branch is deleted
	if testutil.BranchExists(t, dir, "feature/test-override-fetch") {
		t.Error("Expected feature branch to be deleted")
	}

	// Verify that changes are merged into develop
	_, err = testutil.RunGit(t, dir, "checkout", "develop")
	if err != nil {
		t.Fatalf("Failed to checkout develop: %v", err)
	}

	if !testutil.FileExists(t, dir, "override-fetch-test.txt") {
		t.Error("Expected override-fetch-test.txt to exist in develop branch")
	}
}

// TestFinishFeatureBranchContinueDoesNotFetch tests that continue operation doesn't fetch.
// Steps:
// 1. Sets up a test repository and initializes git-flow
// 2. Creates conflicting changes in develop
// 3. Creates a feature branch with conflicting changes
// 4. Adds more conflicting changes to develop
// 5. Attempts to finish (will conflict)
// 6. Verifies conflict is detected
// 7. Resolves the conflict manually
// 8. Continues finish with --continue
// 9. Verifies no fetch message appears during continue
// 10. Verifies the branch is merged into develop
// 11. Verifies the feature branch is deleted
func TestFinishFeatureBranchContinueDoesNotFetch(t *testing.T) {
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	_, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v", err)
	}

	// Create conflicting content in develop
	_, err = testutil.RunGit(t, dir, "checkout", "develop")
	if err != nil {
		t.Fatalf("Failed to checkout develop: %v", err)
	}
	testutil.WriteFile(t, dir, "conflict.txt", "Develop version\nLine 2\nLine 3")
	_, err = testutil.RunGit(t, dir, "add", "conflict.txt")
	if err != nil {
		t.Fatalf("Failed to add conflict.txt: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Develop changes")
	if err != nil {
		t.Fatalf("Failed to commit develop changes: %v", err)
	}

	// Create feature branch with conflicting changes
	_, err = testutil.RunGitFlow(t, dir, "feature", "start", "test-continue-no-fetch")
	if err != nil {
		t.Fatalf("Failed to create feature branch: %v", err)
	}

	testutil.WriteFile(t, dir, "conflict.txt", "Feature version\nLine 2 modified\nLine 3")
	_, err = testutil.RunGit(t, dir, "add", "conflict.txt")
	if err != nil {
		t.Fatalf("Failed to add conflict.txt: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Feature changes")
	if err != nil {
		t.Fatalf("Failed to commit feature changes: %v", err)
	}

	// Add more conflicting changes to develop
	_, err = testutil.RunGit(t, dir, "checkout", "develop")
	if err != nil {
		t.Fatalf("Failed to checkout develop: %v", err)
	}
	testutil.WriteFile(t, dir, "conflict.txt", "Develop version updated\nLine 2\nLine 3 modified")
	_, err = testutil.RunGit(t, dir, "add", "conflict.txt")
	if err != nil {
		t.Fatalf("Failed to add conflict.txt: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "More develop changes")
	if err != nil {
		t.Fatalf("Failed to commit more develop changes: %v", err)
	}

	// Switch back to feature branch
	_, err = testutil.RunGit(t, dir, "checkout", "feature/test-continue-no-fetch")
	if err != nil {
		t.Fatalf("Failed to checkout feature branch: %v", err)
	}

	// Try to finish with --no-fetch (will conflict)
	output, _ := testutil.RunGitFlow(t, dir, "feature", "finish", "--no-fetch", "test-continue-no-fetch")

	// Verify conflict was detected
	if !strings.Contains(output, "conflict") && !strings.Contains(output, "CONFLICT") {
		t.Errorf("Expected merge conflict to be detected. Output: %s", output)
	}

	// Verify no fetch occurred in initial finish (we're testing continue behavior)
	if strings.Contains(output, "Fetching from remote") {
		t.Error("Did not expect fetch in initial finish for this test")
	}

	// Resolve conflict
	testutil.WriteFile(t, dir, "conflict.txt", "Resolved version\nLine 2 resolved\nLine 3 resolved")
	_, err = testutil.RunGit(t, dir, "add", "conflict.txt")
	if err != nil {
		t.Fatalf("Failed to resolve conflict: %v", err)
	}

	// Continue finish operation
	continueOutput, err := testutil.RunGitFlow(t, dir, "feature", "finish", "--continue", "test-continue-no-fetch")
	if err != nil {
		t.Fatalf("Failed to continue finish: %v\nOutput: %s", err, continueOutput)
	}

	// Verify NO fetch occurred during continue
	if strings.Contains(continueOutput, "Fetching from remote") {
		t.Error("Expected no fetch to occur during --continue operation")
	}

	// Verify successful completion
	if !strings.Contains(continueOutput, "Successfully finished") {
		t.Error("Expected successful finish message after continue")
	}

	// Verify branch deleted
	if testutil.BranchExists(t, dir, "feature/test-continue-no-fetch") {
		t.Error("Expected feature branch to be deleted after successful finish")
	}

	// Verify changes merged into develop
	_, err = testutil.RunGit(t, dir, "checkout", "develop")
	if err != nil {
		t.Fatalf("Failed to checkout develop: %v", err)
	}

	content := testutil.ReadFile(t, dir, "conflict.txt")
	if !strings.Contains(content, "Resolved version") {
		t.Error("Expected resolved changes to be in develop branch")
	}
}

// TestFinishFeatureBranchNoRemote tests that finish skips fetch silently when no remote exists.
// Steps:
// 1. Sets up a test repository (no remote) and initializes git-flow with defaults
// 2. Creates a feature branch and adds a commit
// 3. Runs 'git flow feature finish' (default fetch=true, but no remote configured)
// 4. Verifies output does NOT contain "Fetching from remote" (fetch skipped silently)
// 5. Verifies output does NOT contain "does not appear to be a git repository" (no confusing error)
// 6. Verifies finish completes successfully (branch merged and deleted)
func TestFinishFeatureBranchNoRemote(t *testing.T) {
	// Setup test repository without remote
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	_, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v", err)
	}

	// Create a feature branch
	_, err = testutil.RunGitFlow(t, dir, "feature", "start", "no-remote-test")
	if err != nil {
		t.Fatalf("Failed to create feature branch: %v", err)
	}

	// Create a test file and commit
	testutil.WriteFile(t, dir, "no-remote-test.txt", "test content")
	_, err = testutil.RunGit(t, dir, "add", "no-remote-test.txt")
	if err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Add no-remote test file")
	if err != nil {
		t.Fatalf("Failed to commit file: %v", err)
	}

	// Finish the feature branch (fetch defaults to true, but no remote exists)
	output, err := testutil.RunGitFlow(t, dir, "feature", "finish", "no-remote-test")
	if err != nil {
		t.Fatalf("Failed to finish feature branch: %v\nOutput: %s", err, output)
	}

	// Verify fetch was skipped silently (no fetch output at all)
	if strings.Contains(output, "Fetching from remote") {
		t.Error("Expected fetch to be skipped silently when no remote exists")
	}

	// Verify no confusing error messages about missing remote
	if strings.Contains(output, "does not appear to be a git repository") {
		t.Error("Expected no confusing error about missing remote")
	}

	// Verify finish completed successfully
	if !strings.Contains(output, "Successfully finished") {
		t.Error("Expected successful finish message")
	}

	// Verify feature branch is deleted
	if testutil.BranchExists(t, dir, "feature/no-remote-test") {
		t.Error("Expected feature branch to be deleted")
	}

	// Verify changes are merged into develop
	_, err = testutil.RunGit(t, dir, "checkout", "develop")
	if err != nil {
		t.Fatalf("Failed to checkout develop: %v", err)
	}
	if !testutil.FileExists(t, dir, "no-remote-test.txt") {
		t.Error("Expected no-remote-test.txt to exist in develop branch")
	}
}
