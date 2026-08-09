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
	t.Parallel()
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
	t.Parallel()
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
// Uses a remote-backed fixture so the absence of "Fetching" genuinely reflects the config
// (fetch=false), not the no-remote guard.
// Steps:
// 1. Sets up a test repository with a remote and initializes git-flow
// 2. Configures gitflow.feature.finish.fetch = false in git config
// 3. Creates a feature branch
// 4. Adds a test file to the feature branch
// 5. Commits changes to the feature branch
// 6. Finishes the feature branch without flags
// 7. Verifies no fetch message appears in output (config disabled fetch)
// 8. Verifies the branch is merged into develop
// 9. Verifies the feature branch is deleted
func TestFinishFeatureBranchNoFetchFromConfig(t *testing.T) {
	t.Parallel()
	// Setup test repository with remote so the absence of fetch reflects the config
	dir, remoteDir := testutil.SetupTestRepoWithRemote(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer testutil.CleanupTestRepo(t, remoteDir)

	// Configure fetch to be disabled for feature finish
	_, err := testutil.RunGit(t, dir, "config", "gitflow.feature.finish.fetch", "false")
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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

// commitFeatureAndPush is a small helper: writes and commits a file on the current branch,
// then pushes the given branch with upstream tracking. Returns the branch's SHA after the commit.
func commitFeatureAndPush(t *testing.T, dir, file, content, commitMsg, branch string) string {
	t.Helper()
	testutil.WriteFile(t, dir, file, content)
	if _, err := testutil.RunGit(t, dir, "add", file); err != nil {
		t.Fatalf("Failed to add %s: %v", file, err)
	}
	if _, err := testutil.RunGit(t, dir, "commit", "-m", commitMsg); err != nil {
		t.Fatalf("Failed to commit %s: %v", file, err)
	}
	if _, err := testutil.RunGit(t, dir, "push", "--set-upstream", "origin", branch); err != nil {
		t.Fatalf("Failed to push %s: %v", branch, err)
	}
	sha, err := testutil.RunGit(t, dir, "rev-parse", branch)
	if err != nil {
		t.Fatalf("Failed to get SHA of %s: %v", branch, err)
	}
	return strings.TrimSpace(sha)
}

// TestFinishNoUpstreamSkipsFetch tests Scenario 2: a remote is configured but the topic branch has
// no upstream. The topic fetch/sync is skipped (the missing remote ref is benign); the parent may
// still be fetched best-effort. Finish completes and merges the topic.
// Steps:
// 1. Sets up a test repository with remote and initializes git-flow
// 2. Creates a feature branch and commits (does NOT push — no tracking branch)
// 3. Finishes the feature branch
// 4. Verifies no sync abort and that the branch is merged into develop
func TestFinishNoUpstreamSkipsFetch(t *testing.T) {
	t.Parallel()
	dir, remoteDir := testutil.SetupTestRepoWithRemote(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer testutil.CleanupTestRepo(t, remoteDir)

	if _, err := testutil.RunGitFlow(t, dir, "feature", "start", "test-no-upstream"); err != nil {
		t.Fatalf("Failed to create feature branch: %v", err)
	}
	testutil.WriteFile(t, dir, "feature.txt", "feature content")
	if _, err := testutil.RunGit(t, dir, "add", "feature.txt"); err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	if _, err := testutil.RunGit(t, dir, "commit", "-m", "Add feature file"); err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Finish without pushing (topic has no upstream)
	output, err := testutil.RunGitFlow(t, dir, "feature", "finish", "test-no-upstream")
	if err != nil {
		t.Fatalf("Expected finish to complete with no upstream topic. Error: %v\nOutput: %s", err, output)
	}

	// Load-bearing: no sync abort
	if strings.Contains(output, "is behind") || strings.Contains(output, "is ahead") || strings.Contains(output, "has diverged") {
		t.Errorf("Expected no sync abort for a topic with no upstream. Output: %s", output)
	}

	// Load-bearing: topic merged into develop
	if testutil.BranchExists(t, dir, "feature/test-no-upstream") {
		t.Error("Expected feature branch to be deleted")
	}
	if _, err := testutil.RunGit(t, dir, "checkout", "develop"); err != nil {
		t.Fatalf("Failed to checkout develop: %v", err)
	}
	if !testutil.FileExists(t, dir, "feature.txt") {
		t.Error("Expected feature.txt to exist in develop branch")
	}
}

// TestFinishStaleRemoteRefIsBenign tests Scenario 4 (the linchpin): a stale remote-tracking ref
// remains after the remote branch was deleted remotely. The topic fetch reports ref-not-found,
// which is benign — the sync check against the stale ref is skipped, and finish completes.
// Steps:
// 1. Sets up a test repository with remote and initializes git-flow
// 2. Creates a feature branch, commits B, and pushes it (tracking ref origin/... = B)
// 3. Adds a local commit C (so local = C, stale tracking ref = B, B != C)
// 4. Deletes the branch directly in the bare remote (tracking ref not pruned)
// 5. Asserts the preconditions (remote ref absent, origin ref = B, local = C, B != C)
// 6. Finishes the feature branch WITHOUT --force
// 7. Verifies no ahead/diverged abort and that commit C is merged into develop
func TestFinishStaleRemoteRefIsBenign(t *testing.T) {
	t.Parallel()
	dir, remoteDir := testutil.SetupTestRepoWithRemote(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer testutil.CleanupTestRepo(t, remoteDir)

	if _, err := testutil.RunGitFlow(t, dir, "feature", "start", "test-stale"); err != nil {
		t.Fatalf("Failed to create feature branch: %v", err)
	}

	// Commit B and push (tracking ref origin/feature/test-stale = B)
	shaB := commitFeatureAndPush(t, dir, "stale-b.txt", "b", "Commit B", "feature/test-stale")

	// Add local commit C (not pushed)
	testutil.WriteFile(t, dir, "stale-c.txt", "c")
	if _, err := testutil.RunGit(t, dir, "add", "stale-c.txt"); err != nil {
		t.Fatalf("Failed to add stale-c.txt: %v", err)
	}
	if _, err := testutil.RunGit(t, dir, "commit", "-m", "Commit C"); err != nil {
		t.Fatalf("Failed to commit C: %v", err)
	}
	shaCraw, err := testutil.RunGit(t, dir, "rev-parse", "feature/test-stale")
	if err != nil {
		t.Fatalf("Failed to get local SHA: %v", err)
	}
	shaC := strings.TrimSpace(shaCraw)

	// Delete the branch directly in the bare remote (local tracking ref NOT pruned)
	if _, err := testutil.RunGit(t, remoteDir, "update-ref", "-d", "refs/heads/feature/test-stale"); err != nil {
		t.Fatalf("Failed to delete branch in bare remote: %v", err)
	}

	// Preconditions (guard against a vacuous test)
	if _, err := testutil.RunGit(t, remoteDir, "rev-parse", "--verify", "refs/heads/feature/test-stale"); err == nil {
		t.Fatalf("Precondition failed: expected the bare remote ref to be absent")
	}
	originRefRaw, err := testutil.RunGit(t, dir, "rev-parse", "origin/feature/test-stale")
	if err != nil {
		t.Fatalf("Precondition failed: stale tracking ref should resolve: %v", err)
	}
	if strings.TrimSpace(originRefRaw) != shaB {
		t.Fatalf("Precondition failed: expected origin/feature/test-stale = B (%s), got %s", shaB, strings.TrimSpace(originRefRaw))
	}
	if shaB == shaC {
		t.Fatalf("Precondition failed: expected B (%s) != C (%s)", shaB, shaC)
	}

	// Finish WITHOUT --force
	output, err := testutil.RunGitFlow(t, dir, "feature", "finish", "test-stale")
	if err != nil {
		t.Fatalf("Expected finish to complete with a stale remote ref. Error: %v\nOutput: %s", err, output)
	}

	// No ahead/diverged abort against the stale ref
	if strings.Contains(output, "is ahead") || strings.Contains(output, "has diverged") {
		t.Errorf("Expected no ahead/diverged abort against the stale ref. Output: %s", output)
	}

	// Commit C merged into develop
	if testutil.BranchExists(t, dir, "feature/test-stale") {
		t.Error("Expected feature branch to be deleted")
	}
	if _, err := testutil.RunGit(t, dir, "checkout", "develop"); err != nil {
		t.Fatalf("Failed to checkout develop: %v", err)
	}
	if !testutil.FileExists(t, dir, "stale-c.txt") {
		t.Error("Expected stale-c.txt (commit C) to be merged into develop")
	}
}

// TestFinishUnreachableRemoteAborts tests Scenario 5: a reachable-but-failing remote makes the
// topic fetch fail with a transport error, which is now fatal. Finish aborts and does not merge.
// Steps:
// 1. Sets up a test repository with remote and initializes git-flow
// 2. Creates a feature branch, commits, and pushes it (in sync)
// 3. Points origin at a nonexistent path so fetch fails with a transport error
// 4. Records develop's SHA before finishing
// 5. Finishes the feature branch
// 6. Verifies a fatal error naming the topic branch and suggesting --no-fetch / --force
// 7. Verifies the merge did not happen (develop unchanged, branch still exists)
func TestFinishUnreachableRemoteAborts(t *testing.T) {
	t.Parallel()
	dir, remoteDir := testutil.SetupTestRepoWithRemote(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer testutil.CleanupTestRepo(t, remoteDir)

	if _, err := testutil.RunGitFlow(t, dir, "feature", "start", "test-unreachable"); err != nil {
		t.Fatalf("Failed to create feature branch: %v", err)
	}
	commitFeatureAndPush(t, dir, "feature.txt", "feature content", "Add feature file", "feature/test-unreachable")

	// Break the remote so any fetch fails with a transport error
	if _, err := testutil.RunGit(t, dir, "remote", "set-url", "origin", "./nonexistent-remote-repo.git"); err != nil {
		t.Fatalf("Failed to break remote: %v", err)
	}

	developBefore, err := testutil.RunGit(t, dir, "rev-parse", "develop")
	if err != nil {
		t.Fatalf("Failed to get develop SHA: %v", err)
	}

	output, err := testutil.RunGitFlow(t, dir, "feature", "finish", "test-unreachable")
	if err == nil {
		t.Errorf("Expected finish to fail on an unreachable remote. Output: %s", output)
	}

	// Verify the fatal error is the topic fetch and offers the escape hatches
	if !strings.Contains(output, "feature/test-unreachable") {
		t.Errorf("Expected the fatal error to name the topic branch. Output: %s", output)
	}
	if !strings.Contains(output, "--no-fetch") || !strings.Contains(output, "--force") {
		t.Errorf("Expected the fatal error to suggest --no-fetch and --force. Output: %s", output)
	}

	// Verify no merge happened
	developAfter, err := testutil.RunGit(t, dir, "rev-parse", "develop")
	if err != nil {
		t.Fatalf("Failed to get develop SHA after: %v", err)
	}
	if developBefore != developAfter {
		t.Errorf("Expected develop unchanged after aborted finish. Before: %s After: %s", developBefore, developAfter)
	}
	if !testutil.BranchExists(t, dir, "feature/test-unreachable") {
		t.Error("Expected feature branch to still exist after aborted finish")
	}
}

// TestFinishUnreachableRemoteNoFetchCompletes tests Scenario 6: --no-fetch skips the fetch entirely,
// so an unreachable remote does not block finishing. keepremote avoids the unrelated remote-delete
// failure on the broken URL (delete behavior is out of scope for this change).
// Steps:
// 1. Sets up a test repository with remote and initializes git-flow; enables keepremote
// 2. Creates a feature branch, commits, and pushes it (in sync tracking data)
// 3. Points origin at a nonexistent path
// 4. Finishes with --no-fetch
// 5. Verifies no fetch occurred and the branch merged into develop
func TestFinishUnreachableRemoteNoFetchCompletes(t *testing.T) {
	t.Parallel()
	dir, remoteDir := testutil.SetupTestRepoWithRemote(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer testutil.CleanupTestRepo(t, remoteDir)

	// keepremote isolates the test to the fetch behavior (delete is out of scope)
	if _, err := testutil.RunGit(t, dir, "config", "gitflow.feature.finish.keepremote", "true"); err != nil {
		t.Fatalf("Failed to set keepremote config: %v", err)
	}

	if _, err := testutil.RunGitFlow(t, dir, "feature", "start", "test-unreachable-nofetch"); err != nil {
		t.Fatalf("Failed to create feature branch: %v", err)
	}
	commitFeatureAndPush(t, dir, "feature.txt", "feature content", "Add feature file", "feature/test-unreachable-nofetch")

	if _, err := testutil.RunGit(t, dir, "remote", "set-url", "origin", "./nonexistent-remote-repo.git"); err != nil {
		t.Fatalf("Failed to break remote: %v", err)
	}

	output, err := testutil.RunGitFlow(t, dir, "feature", "finish", "--no-fetch", "test-unreachable-nofetch")
	if err != nil {
		t.Fatalf("Expected finish to complete with --no-fetch. Error: %v\nOutput: %s", err, output)
	}
	if strings.Contains(output, "Fetching from remote") {
		t.Errorf("Expected no fetch with --no-fetch. Output: %s", output)
	}
	if testutil.BranchExists(t, dir, "feature/test-unreachable-nofetch") {
		t.Error("Expected feature branch to be deleted")
	}
	if _, err := testutil.RunGit(t, dir, "checkout", "develop"); err != nil {
		t.Fatalf("Failed to checkout develop: %v", err)
	}
	if !testutil.FileExists(t, dir, "feature.txt") {
		t.Error("Expected feature.txt to exist in develop branch")
	}
}

// TestFinishUnreachableRemoteForceCompletes tests Scenario 7: --force ignores the fetch failure.
// keepremote avoids the unrelated remote-delete failure on the broken URL.
// Steps:
// 1. Sets up a test repository with remote and initializes git-flow; enables keepremote
// 2. Creates a feature branch, commits, and pushes it
// 3. Points origin at a nonexistent path
// 4. Finishes with --force
// 5. Verifies the branch merged into develop
func TestFinishUnreachableRemoteForceCompletes(t *testing.T) {
	t.Parallel()
	dir, remoteDir := testutil.SetupTestRepoWithRemote(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer testutil.CleanupTestRepo(t, remoteDir)

	if _, err := testutil.RunGit(t, dir, "config", "gitflow.feature.finish.keepremote", "true"); err != nil {
		t.Fatalf("Failed to set keepremote config: %v", err)
	}

	if _, err := testutil.RunGitFlow(t, dir, "feature", "start", "test-unreachable-force"); err != nil {
		t.Fatalf("Failed to create feature branch: %v", err)
	}
	commitFeatureAndPush(t, dir, "feature.txt", "feature content", "Add feature file", "feature/test-unreachable-force")

	if _, err := testutil.RunGit(t, dir, "remote", "set-url", "origin", "./nonexistent-remote-repo.git"); err != nil {
		t.Fatalf("Failed to break remote: %v", err)
	}

	output, err := testutil.RunGitFlow(t, dir, "feature", "finish", "--force", "test-unreachable-force")
	if err != nil {
		t.Fatalf("Expected finish to complete with --force. Error: %v\nOutput: %s", err, output)
	}
	if testutil.BranchExists(t, dir, "feature/test-unreachable-force") {
		t.Error("Expected feature branch to be deleted")
	}
	if _, err := testutil.RunGit(t, dir, "checkout", "develop"); err != nil {
		t.Fatalf("Failed to checkout develop: %v", err)
	}
	if !testutil.FileExists(t, dir, "feature.txt") {
		t.Error("Expected feature.txt to exist in develop branch")
	}
}

// TestFinishAheadRemoteForceCompletes tests Scenario 11b: --force bypasses the sync check when the
// topic is ahead of the remote, so finish completes.
// Steps:
// 1. Sets up a test repository with remote and initializes git-flow
// 2. Creates a feature branch, commits, and pushes it
// 3. Adds a local commit without pushing (ahead by 1)
// 4. Finishes with --force
// 5. Verifies the branch merged into develop
func TestFinishAheadRemoteForceCompletes(t *testing.T) {
	t.Parallel()
	dir, remoteDir := testutil.SetupTestRepoWithRemote(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer testutil.CleanupTestRepo(t, remoteDir)

	if _, err := testutil.RunGitFlow(t, dir, "feature", "start", "test-ahead-force"); err != nil {
		t.Fatalf("Failed to create feature branch: %v", err)
	}
	commitFeatureAndPush(t, dir, "feature.txt", "feature content", "Add feature file", "feature/test-ahead-force")

	// Ahead by 1 (local-only commit)
	testutil.WriteFile(t, dir, "local.txt", "local content")
	if _, err := testutil.RunGit(t, dir, "add", "local.txt"); err != nil {
		t.Fatalf("Failed to add local.txt: %v", err)
	}
	if _, err := testutil.RunGit(t, dir, "commit", "-m", "Local commit"); err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	output, err := testutil.RunGitFlow(t, dir, "feature", "finish", "--force", "test-ahead-force")
	if err != nil {
		t.Fatalf("Expected finish with --force to succeed when ahead. Error: %v\nOutput: %s", err, output)
	}
	if testutil.BranchExists(t, dir, "feature/test-ahead-force") {
		t.Error("Expected feature branch to be deleted")
	}
	if _, err := testutil.RunGit(t, dir, "checkout", "develop"); err != nil {
		t.Fatalf("Failed to checkout develop: %v", err)
	}
	if !testutil.FileExists(t, dir, "local.txt") {
		t.Error("Expected local.txt to exist in develop branch")
	}
}

// TestFinishDivergedRemoteForceCompletes tests Scenario 11c: --force bypasses the sync check when
// the topic has diverged from the remote, so finish completes.
// Steps:
// 1. Sets up a test repository with remote and initializes git-flow
// 2. Creates a feature branch, commits, and pushes it
// 3. Creates divergence (remote-only commit via a second clone; local-only commit) and fetches
// 4. Finishes with --force
// 5. Verifies the branch merged into develop
func TestFinishDivergedRemoteForceCompletes(t *testing.T) {
	t.Parallel()
	dir, remoteDir := testutil.SetupTestRepoWithRemote(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer testutil.CleanupTestRepo(t, remoteDir)

	if _, err := testutil.RunGitFlow(t, dir, "feature", "start", "test-diverged-force"); err != nil {
		t.Fatalf("Failed to create feature branch: %v", err)
	}
	commitFeatureAndPush(t, dir, "feature.txt", "feature content", "Add feature file", "feature/test-diverged-force")

	// Remote-only commit via a second clone
	secondDir := t.TempDir()
	if _, err := testutil.RunGit(t, secondDir, "clone", remoteDir, "."); err != nil {
		t.Fatalf("Failed to clone: %v", err)
	}
	testutil.ConfigureGitIdentity(t, secondDir)
	if _, err := testutil.RunGit(t, secondDir, "checkout", "feature/test-diverged-force"); err != nil {
		t.Fatalf("Failed to checkout feature in second repo: %v", err)
	}
	testutil.WriteFile(t, secondDir, "remote-change.txt", "remote content")
	if _, err := testutil.RunGit(t, secondDir, "add", "remote-change.txt"); err != nil {
		t.Fatalf("Failed to add file in second repo: %v", err)
	}
	if _, err := testutil.RunGit(t, secondDir, "commit", "-m", "Remote commit"); err != nil {
		t.Fatalf("Failed to commit in second repo: %v", err)
	}
	if _, err := testutil.RunGit(t, secondDir, "push", "origin", "feature/test-diverged-force"); err != nil {
		t.Fatalf("Failed to push from second repo: %v", err)
	}

	// Local-only commit (creates divergence), then fetch to update the tracking ref
	testutil.WriteFile(t, dir, "local-change.txt", "local content")
	if _, err := testutil.RunGit(t, dir, "add", "local-change.txt"); err != nil {
		t.Fatalf("Failed to add local-change.txt: %v", err)
	}
	if _, err := testutil.RunGit(t, dir, "commit", "-m", "Local commit"); err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}
	if _, err := testutil.RunGit(t, dir, "fetch", "origin"); err != nil {
		t.Fatalf("Failed to fetch: %v", err)
	}

	output, err := testutil.RunGitFlow(t, dir, "feature", "finish", "--force", "test-diverged-force")
	if err != nil {
		t.Fatalf("Expected finish with --force to succeed when diverged. Error: %v\nOutput: %s", err, output)
	}
	if testutil.BranchExists(t, dir, "feature/test-diverged-force") {
		t.Error("Expected feature branch to be deleted")
	}
	if _, err := testutil.RunGit(t, dir, "checkout", "develop"); err != nil {
		t.Fatalf("Failed to checkout develop: %v", err)
	}
	if !testutil.FileExists(t, dir, "local-change.txt") {
		t.Error("Expected local-change.txt to exist in develop branch")
	}
}

// TestFinishParentAbsentOnRemoteIsBenign tests Scenario 12: the parent branch is absent on the
// remote (deleted there) while the topic is in sync. The parent fetch fails with ref-not-found,
// which is a non-fatal note; the topic fetch+sync pass and finish completes.
// Steps:
// 1. Sets up a test repository with remote and initializes git-flow
// 2. Creates a feature branch, commits, and pushes it (in sync)
// 3. Deletes develop directly in the bare remote
// 4. Finishes the feature branch
// 5. Verifies finish completes and the topic merged into local develop
func TestFinishParentAbsentOnRemoteIsBenign(t *testing.T) {
	t.Parallel()
	dir, remoteDir := testutil.SetupTestRepoWithRemote(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer testutil.CleanupTestRepo(t, remoteDir)

	if _, err := testutil.RunGitFlow(t, dir, "feature", "start", "test-parent-absent"); err != nil {
		t.Fatalf("Failed to create feature branch: %v", err)
	}
	commitFeatureAndPush(t, dir, "feature.txt", "feature content", "Add feature file", "feature/test-parent-absent")

	// Delete the parent (develop) directly in the bare remote; topic stays in sync
	if _, err := testutil.RunGit(t, remoteDir, "update-ref", "-d", "refs/heads/develop"); err != nil {
		t.Fatalf("Failed to delete develop in bare remote: %v", err)
	}

	output, err := testutil.RunGitFlow(t, dir, "feature", "finish", "test-parent-absent")
	if err != nil {
		t.Fatalf("Expected finish to complete when the parent is absent on the remote. Error: %v\nOutput: %s", err, output)
	}
	if testutil.BranchExists(t, dir, "feature/test-parent-absent") {
		t.Error("Expected feature branch to be deleted")
	}
	if _, err := testutil.RunGit(t, dir, "checkout", "develop"); err != nil {
		t.Fatalf("Failed to checkout develop: %v", err)
	}
	if !testutil.FileExists(t, dir, "feature.txt") {
		t.Error("Expected feature.txt to be merged into local develop")
	}
}

// TestFinishFeatureFetchConfigOnFetches tests that gitflow.feature.finish.fetch=on
// enables the fetch, matching git-config's truthy "on" spelling.
// Steps:
// 1. Sets up a test repository with a remote and initializes git-flow
// 2. Configures gitflow.feature.finish.fetch = on
// 3. Creates a feature branch and commits a test file
// 4. Finishes the feature branch without any fetch flag
// 5. Verifies "Fetching from remote" appears in output
// 6. Verifies the feature branch is deleted and the file exists on develop
func TestFinishFeatureFetchConfigOnFetches(t *testing.T) {
	t.Parallel()
	dir, remoteDir := testutil.SetupTestRepoWithRemote(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer testutil.CleanupTestRepo(t, remoteDir)

	if _, err := testutil.RunGit(t, dir, "config", "gitflow.feature.finish.fetch", "on"); err != nil {
		t.Fatalf("Failed to configure fetch option: %v", err)
	}

	if _, err := testutil.RunGitFlow(t, dir, "feature", "start", "test-config-on-fetch"); err != nil {
		t.Fatalf("Failed to create feature branch: %v", err)
	}

	testutil.WriteFile(t, dir, "config-on-fetch-test.txt", "test content")
	if _, err := testutil.RunGit(t, dir, "add", "config-on-fetch-test.txt"); err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	if _, err := testutil.RunGit(t, dir, "commit", "-m", "Add config on-fetch test file"); err != nil {
		t.Fatalf("Failed to commit file: %v", err)
	}

	output, err := testutil.RunGitFlow(t, dir, "feature", "finish", "test-config-on-fetch")
	if err != nil {
		t.Fatalf("Failed to finish feature branch: %v\nOutput: %s", err, output)
	}

	if !strings.Contains(output, "Fetching from remote") {
		t.Errorf("Expected fetch to occur with fetch=on. Output: %s", output)
	}
	if testutil.BranchExists(t, dir, "feature/test-config-on-fetch") {
		t.Error("Expected feature branch to be deleted")
	}
	if _, err := testutil.RunGit(t, dir, "checkout", "develop"); err != nil {
		t.Fatalf("Failed to checkout develop: %v", err)
	}
	if !testutil.FileExists(t, dir, "config-on-fetch-test.txt") {
		t.Error("Expected config-on-fetch-test.txt to exist in develop branch")
	}
}

// TestFinishFeatureFetchConfigNoSkipsFetch tests that gitflow.feature.finish.fetch=no
// skips the fetch, matching git-config's falsy "no" spelling. Distinguishes a falsy
// value from an unset key, which would fetch (the finish default is true).
// Steps:
// 1. Sets up a test repository with a remote and initializes git-flow
// 2. Configures gitflow.feature.finish.fetch = no
// 3. Creates a feature branch and commits a test file
// 4. Finishes the feature branch without any fetch flag
// 5. Verifies "Fetching from remote" does NOT appear in output
// 6. Verifies the feature branch is deleted and the file exists on develop
func TestFinishFeatureFetchConfigNoSkipsFetch(t *testing.T) {
	t.Parallel()
	dir, remoteDir := testutil.SetupTestRepoWithRemote(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer testutil.CleanupTestRepo(t, remoteDir)

	if _, err := testutil.RunGit(t, dir, "config", "gitflow.feature.finish.fetch", "no"); err != nil {
		t.Fatalf("Failed to configure fetch option: %v", err)
	}

	if _, err := testutil.RunGitFlow(t, dir, "feature", "start", "test-config-no-fetch-word"); err != nil {
		t.Fatalf("Failed to create feature branch: %v", err)
	}

	testutil.WriteFile(t, dir, "config-no-fetch-word-test.txt", "test content")
	if _, err := testutil.RunGit(t, dir, "add", "config-no-fetch-word-test.txt"); err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	if _, err := testutil.RunGit(t, dir, "commit", "-m", "Add config no-fetch word test file"); err != nil {
		t.Fatalf("Failed to commit file: %v", err)
	}

	output, err := testutil.RunGitFlow(t, dir, "feature", "finish", "test-config-no-fetch-word")
	if err != nil {
		t.Fatalf("Failed to finish feature branch: %v\nOutput: %s", err, output)
	}

	if strings.Contains(output, "Fetching from remote") {
		t.Errorf("Expected no fetch with fetch=no. Output: %s", output)
	}
	if testutil.BranchExists(t, dir, "feature/test-config-no-fetch-word") {
		t.Error("Expected feature branch to be deleted")
	}
	if _, err := testutil.RunGit(t, dir, "checkout", "develop"); err != nil {
		t.Fatalf("Failed to checkout develop: %v", err)
	}
	if !testutil.FileExists(t, dir, "config-no-fetch-word-test.txt") {
		t.Error("Expected config-no-fetch-word-test.txt to exist in develop branch")
	}
}
