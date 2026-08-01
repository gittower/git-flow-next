package cmd_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gittower/git-flow-next/test/testutil"
)

// createRejectingHooks creates hooks that reject merge and commit operations.
// This installs both pre-merge-commit (for merge commits) and pre-commit (for regular commits).
// The --no-verify flag on git merge bypasses pre-merge-commit.
// The --no-verify flag on git commit bypasses pre-commit.
func createRejectingHooks(t *testing.T, dir string) {
	t.Helper()
	hooksDir := filepath.Join(dir, ".git", "hooks")
	if err := os.MkdirAll(hooksDir, 0755); err != nil {
		t.Fatalf("Failed to create hooks directory: %v", err)
	}

	script := `#!/bin/sh
echo "Hook: operation rejected by hook" >&2
exit 1
`
	// Install pre-merge-commit hook (runs during merge commits, Git 2.24+)
	preMergeCommitPath := filepath.Join(hooksDir, "pre-merge-commit")
	if err := os.WriteFile(preMergeCommitPath, []byte(script), 0755); err != nil {
		t.Fatalf("Failed to create pre-merge-commit hook: %v", err)
	}

	// Install pre-commit hook (runs during regular commits like squash merge final commit)
	preCommitPath := filepath.Join(hooksDir, "pre-commit")
	if err := os.WriteFile(preCommitPath, []byte(script), 0755); err != nil {
		t.Fatalf("Failed to create pre-commit hook: %v", err)
	}
}

// TestFinishFeatureFailsWithRejectingPreCommitHook tests that finish fails when a pre-merge-commit hook rejects.
// Steps:
// 1. Sets up a test repository and initializes git-flow
// 2. Creates a feature branch
// 3. Adds a test file and commits to the feature branch
// 4. Adds a different commit on develop to force a non-fast-forward merge
// 5. Installs pre-merge-commit and pre-commit hooks that reject (exit code 1)
// 6. Attempts to finish the feature branch without --no-verify
// 7. Verifies the finish operation fails (hook blocked the merge commit)
// 8. Verifies the feature branch still exists (not deleted)
func TestFinishFeatureFailsWithRejectingPreCommitHook(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow
	_, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v", err)
	}

	// Start a feature branch
	_, err = testutil.RunGitFlow(t, dir, "feature", "start", "test-hook")
	if err != nil {
		t.Fatalf("Failed to start feature: %v", err)
	}

	// Add a commit to the feature branch
	testutil.WriteFile(t, dir, "feature.txt", "feature content")
	_, _ = testutil.RunGit(t, dir, "add", "feature.txt")
	_, _ = testutil.RunGit(t, dir, "commit", "-m", "Add feature file")

	// Add a commit to develop to force a non-fast-forward merge
	// (Fast-forward merges don't create merge commits and don't trigger pre-merge-commit hooks)
	_, _ = testutil.RunGit(t, dir, "checkout", "develop")
	testutil.WriteFile(t, dir, "develop.txt", "develop content")
	_, _ = testutil.RunGit(t, dir, "add", "develop.txt")
	_, _ = testutil.RunGit(t, dir, "commit", "-m", "Add develop file")

	// Switch back to feature branch for finish command
	_, _ = testutil.RunGit(t, dir, "checkout", "feature/test-hook")

	// Install rejecting hooks AFTER setting up the branches
	createRejectingHooks(t, dir)

	// Try to finish without --no-verify - should fail due to hook
	output, err := testutil.RunGitFlow(t, dir, "feature", "finish", "--no-fetch", "test-hook")
	if err == nil {
		t.Fatal("Expected finish to fail due to pre-merge-commit hook, but it succeeded")
	}

	// Verify the error mentions the hook or commit failure
	outputLower := strings.ToLower(output)
	if !strings.Contains(outputLower, "hook") && !strings.Contains(outputLower, "rejected") {
		t.Logf("Output: %s", output)
	}

	// Verify feature branch still exists (finish was blocked)
	if !testutil.BranchExists(t, dir, "feature/test-hook") {
		t.Error("Feature branch should still exist when finish failed")
	}
}

// TestFinishFeatureWithNoVerifyFlagBypassesHook tests that --no-verify bypasses the pre-merge-commit hook.
// Steps:
// 1. Sets up a test repository and initializes git-flow
// 2. Creates a feature branch
// 3. Adds a test file and commits to the feature branch
// 4. Adds a different commit on develop to force a non-fast-forward merge
// 5. Installs pre-merge-commit and pre-commit hooks that reject (exit code 1)
// 6. Finishes the feature branch with --no-verify flag
// 7. Verifies the finish operation succeeds (hook was bypassed)
// 8. Verifies the feature branch is deleted
// 9. Verifies changes are merged into develop
func TestFinishFeatureWithNoVerifyFlagBypassesHook(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow
	_, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v", err)
	}

	// Start a feature branch
	_, err = testutil.RunGitFlow(t, dir, "feature", "start", "noverify-test")
	if err != nil {
		t.Fatalf("Failed to start feature: %v", err)
	}

	// Add a commit to the feature branch
	testutil.WriteFile(t, dir, "feature.txt", "feature content")
	_, _ = testutil.RunGit(t, dir, "add", "feature.txt")
	_, _ = testutil.RunGit(t, dir, "commit", "-m", "Add feature file")

	// Add a commit to develop to force a non-fast-forward merge
	// (Fast-forward merges don't create merge commits and don't trigger pre-merge-commit hooks)
	_, _ = testutil.RunGit(t, dir, "checkout", "develop")
	testutil.WriteFile(t, dir, "develop.txt", "develop content")
	_, _ = testutil.RunGit(t, dir, "add", "develop.txt")
	_, _ = testutil.RunGit(t, dir, "commit", "-m", "Add develop file")

	// Switch back to feature branch for finish command
	_, _ = testutil.RunGit(t, dir, "checkout", "feature/noverify-test")

	// Install rejecting hooks AFTER setting up the branches
	createRejectingHooks(t, dir)

	// Finish with --no-verify - should succeed (hook bypassed)
	output, err := testutil.RunGitFlow(t, dir, "feature", "finish", "--no-fetch", "--no-verify", "noverify-test")
	if err != nil {
		t.Fatalf("Expected finish with --no-verify to succeed, but it failed: %v\nOutput: %s", err, output)
	}

	// Verify feature branch is deleted
	if testutil.BranchExists(t, dir, "feature/noverify-test") {
		t.Error("Feature branch should be deleted after successful finish")
	}

	// Verify we're on develop
	currentBranch := testutil.GetCurrentBranch(t, dir)
	if currentBranch != "develop" {
		t.Errorf("Expected to be on develop branch, got: %s", currentBranch)
	}

	// Verify the feature file exists on develop (changes merged)
	featureFile := filepath.Join(dir, "feature.txt")
	if _, err := os.Stat(featureFile); os.IsNotExist(err) {
		t.Error("Feature file should exist on develop after merge")
	}
}

// TestFinishFeatureWithNoVerifyConfig tests that gitflow.feature.finish.noVerify config bypasses the hook.
// Steps:
// 1. Sets up a test repository and initializes git-flow
// 2. Sets gitflow.feature.finish.noVerify=true in git config
// 3. Creates a feature branch
// 4. Adds a test file and commits to the feature branch
// 5. Adds a different commit on develop to force a non-fast-forward merge
// 6. Installs pre-merge-commit and pre-commit hooks that reject (exit code 1)
// 7. Finishes the feature branch without any CLI flags
// 8. Verifies the finish operation succeeds (hook bypassed via config)
// 9. Verifies the feature branch is deleted
// 10. Verifies changes are merged into develop
func TestFinishFeatureWithNoVerifyConfig(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow
	_, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v", err)
	}

	// Set noVerify config before creating the branch
	_, err = testutil.RunGit(t, dir, "config", "gitflow.feature.finish.noVerify", "true")
	if err != nil {
		t.Fatalf("Failed to set noVerify config: %v", err)
	}

	// Start a feature branch
	_, err = testutil.RunGitFlow(t, dir, "feature", "start", "config-test")
	if err != nil {
		t.Fatalf("Failed to start feature: %v", err)
	}

	// Add a commit to the feature branch
	testutil.WriteFile(t, dir, "feature.txt", "feature content")
	_, _ = testutil.RunGit(t, dir, "add", "feature.txt")
	_, _ = testutil.RunGit(t, dir, "commit", "-m", "Add feature file")

	// Add a commit to develop to force a non-fast-forward merge
	_, _ = testutil.RunGit(t, dir, "checkout", "develop")
	testutil.WriteFile(t, dir, "develop.txt", "develop content")
	_, _ = testutil.RunGit(t, dir, "add", "develop.txt")
	_, _ = testutil.RunGit(t, dir, "commit", "-m", "Add develop file")

	// Switch back to feature branch for finish command
	_, _ = testutil.RunGit(t, dir, "checkout", "feature/config-test")

	// Install rejecting hooks
	createRejectingHooks(t, dir)

	// Finish WITHOUT --no-verify flag - config should enable it
	output, err := testutil.RunGitFlow(t, dir, "feature", "finish", "--no-fetch", "config-test")
	if err != nil {
		t.Fatalf("Expected finish to succeed with noVerify config, but it failed: %v\nOutput: %s", err, output)
	}

	// Verify feature branch is deleted
	if testutil.BranchExists(t, dir, "feature/config-test") {
		t.Error("Feature branch should be deleted after successful finish")
	}

	// Verify the feature file exists on develop
	featureFile := filepath.Join(dir, "feature.txt")
	if _, err := os.Stat(featureFile); os.IsNotExist(err) {
		t.Error("Feature file should exist on develop after merge")
	}
}

// TestFinishFeatureNoVerifyFlagOverridesConfig tests that CLI --no-verify overrides noVerify=false config.
// Steps:
// 1. Sets up a test repository and initializes git-flow
// 2. Sets gitflow.feature.finish.noVerify=false in git config
// 3. Creates a feature branch
// 4. Adds a test file and commits to the feature branch
// 5. Adds a different commit on develop to force a non-fast-forward merge
// 6. Installs pre-merge-commit and pre-commit hooks that reject (exit code 1)
// 7. Finishes the feature branch with --no-verify flag
// 8. Verifies the finish operation succeeds (CLI flag overrides config)
// 9. Verifies the feature branch is deleted
// 10. Verifies changes are merged into develop
func TestFinishFeatureNoVerifyFlagOverridesConfig(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow
	_, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v", err)
	}

	// Explicitly set noVerify=false in config
	_, err = testutil.RunGit(t, dir, "config", "gitflow.feature.finish.noVerify", "false")
	if err != nil {
		t.Fatalf("Failed to set noVerify config: %v", err)
	}

	// Start a feature branch
	_, err = testutil.RunGitFlow(t, dir, "feature", "start", "override-test")
	if err != nil {
		t.Fatalf("Failed to start feature: %v", err)
	}

	// Add a commit to the feature branch
	testutil.WriteFile(t, dir, "feature.txt", "feature content")
	_, _ = testutil.RunGit(t, dir, "add", "feature.txt")
	_, _ = testutil.RunGit(t, dir, "commit", "-m", "Add feature file")

	// Add a commit to develop to force a non-fast-forward merge
	_, _ = testutil.RunGit(t, dir, "checkout", "develop")
	testutil.WriteFile(t, dir, "develop.txt", "develop content")
	_, _ = testutil.RunGit(t, dir, "add", "develop.txt")
	_, _ = testutil.RunGit(t, dir, "commit", "-m", "Add develop file")

	// Switch back to feature branch for finish command
	_, _ = testutil.RunGit(t, dir, "checkout", "feature/override-test")

	// Install rejecting hooks
	createRejectingHooks(t, dir)

	// Finish WITH --no-verify flag - should override config=false
	output, err := testutil.RunGitFlow(t, dir, "feature", "finish", "--no-fetch", "--no-verify", "override-test")
	if err != nil {
		t.Fatalf("Expected finish with --no-verify to succeed, but it failed: %v\nOutput: %s", err, output)
	}

	// Verify feature branch is deleted
	if testutil.BranchExists(t, dir, "feature/override-test") {
		t.Error("Feature branch should be deleted after successful finish")
	}

	// Verify the feature file exists on develop
	featureFile := filepath.Join(dir, "feature.txt")
	if _, err := os.Stat(featureFile); os.IsNotExist(err) {
		t.Error("Feature file should exist on develop after merge")
	}
}

// TestFinishFeatureNoVerifyPersistedThroughContinue tests that --no-verify persists through --continue.
// Steps:
// 1. Sets up a test repository and initializes git-flow
// 2. Creates a feature branch with a test file
// 3. Switches to develop and adds conflicting content to the same file
// 4. Switches back to feature branch
// 5. Installs pre-merge-commit and pre-commit hooks that reject (exit code 1)
// 6. Runs finish with --no-verify and --no-fetch (conflict occurs, hook hasn't fired yet)
// 7. Verifies merge state exists with NoVerify=true persisted in state file
// 8. Resolves the conflict and stages the resolved file
// 9. Runs finish --continue WITHOUT --no-verify on the command line
// 10. Verifies the finish succeeds (hook bypassed via persisted MergeState.NoVerify)
// 11. Verifies the feature branch is deleted
// 12. Verifies the resolved changes are merged into develop
func TestFinishFeatureNoVerifyPersistedThroughContinue(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow
	_, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v", err)
	}

	// Start a feature branch
	_, err = testutil.RunGitFlow(t, dir, "feature", "start", "conflict-test")
	if err != nil {
		t.Fatalf("Failed to start feature: %v", err)
	}

	// Add a file on the feature branch
	testutil.WriteFile(t, dir, "conflict.txt", "feature version")
	_, _ = testutil.RunGit(t, dir, "add", "conflict.txt")
	_, _ = testutil.RunGit(t, dir, "commit", "-m", "Add feature version of file")

	// Switch to develop and create conflicting content
	_, _ = testutil.RunGit(t, dir, "checkout", "develop")
	testutil.WriteFile(t, dir, "conflict.txt", "develop version")
	_, _ = testutil.RunGit(t, dir, "add", "conflict.txt")
	_, _ = testutil.RunGit(t, dir, "commit", "-m", "Add develop version of file")

	// Switch back to feature
	_, _ = testutil.RunGit(t, dir, "checkout", "feature/conflict-test")

	// Install rejecting pre-commit hook BEFORE finish
	// The hook will fire during the commit after conflict resolution
	createRejectingHooks(t, dir)

	// Try to finish with --no-verify - will conflict
	output, err := testutil.RunGitFlow(t, dir, "feature", "finish", "--no-fetch", "--no-verify", "conflict-test")
	if err == nil {
		t.Fatalf("Expected finish to fail with merge conflict, but it succeeded\nOutput: %s", output)
	}

	// Verify we have a conflict state
	if !strings.Contains(output, "conflict") && !strings.Contains(strings.ToLower(output), "conflict") {
		t.Logf("Output: %s", output)
	}

	// Verify merge state file exists and contains NoVerify=true
	stateFile := filepath.Join(dir, ".git", "gitflow", "state", "merge.json")
	stateContent, err := os.ReadFile(stateFile)
	if err != nil {
		t.Fatalf("Failed to read merge state file: %v", err)
	}
	if !strings.Contains(string(stateContent), `"noVerify":true`) {
		t.Errorf("Expected merge state to contain noVerify:true, got: %s", string(stateContent))
	}

	// Resolve the conflict
	testutil.WriteFile(t, dir, "conflict.txt", "resolved content")
	_, _ = testutil.RunGit(t, dir, "add", "conflict.txt")

	// Continue WITHOUT --no-verify flag - should use persisted state
	output, err = testutil.RunGitFlow(t, dir, "feature", "finish", "--continue", "conflict-test")
	if err != nil {
		t.Fatalf("Expected finish --continue to succeed (noVerify persisted), but it failed: %v\nOutput: %s", err, output)
	}

	// Verify feature branch is deleted
	if testutil.BranchExists(t, dir, "feature/conflict-test") {
		t.Error("Feature branch should be deleted after successful finish --continue")
	}

	// Verify the resolved file exists on develop
	content, err := os.ReadFile(filepath.Join(dir, "conflict.txt"))
	if err != nil {
		t.Fatal("Resolved file should exist on develop")
	}
	if string(content) != "resolved content" {
		t.Errorf("Expected resolved content, got: %s", string(content))
	}
}

// TestFinishReleaseWithNoVerifyStillCreatesTag tests that --no-verify bypasses hooks but tag is still created.
// Note: Release finish merges to main (not develop), and then updates develop from main.
// The child branch update (develop from main) does NOT use --no-verify (by design).
// To isolate the release merge test, we disable auto-update on develop.
// Steps:
// 1. Sets up a test repository and initializes git-flow
// 2. Disables auto-update on develop branch
// 3. Creates a release branch (release/1.0.0)
// 4. Adds a test file and commits to the release branch
// 5. Adds a commit on main to force a non-fast-forward merge
// 6. Installs pre-merge-commit and pre-commit hooks that reject (exit code 1)
// 7. Finishes the release branch with --no-verify flag
// 8. Verifies the finish operation succeeds (hook was bypassed)
// 9. Verifies tag '1.0.0' was created (tag creation is unaffected by --no-verify)
// 10. Verifies the release branch is deleted
func TestFinishReleaseWithNoVerifyStillCreatesTag(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow
	_, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v", err)
	}

	// Disable auto-update on develop to avoid child branch update triggering hooks
	// (Child branch updates don't use --no-verify by design)
	_, err = testutil.RunGit(t, dir, "config", "gitflow.branch.develop.autoUpdate", "false")
	if err != nil {
		t.Fatalf("Failed to disable auto-update: %v", err)
	}

	// Start a release branch
	_, err = testutil.RunGitFlow(t, dir, "release", "start", "1.0.0")
	if err != nil {
		t.Fatalf("Failed to start release: %v", err)
	}

	// Add a commit to the release branch
	testutil.WriteFile(t, dir, "release.txt", "release content")
	_, _ = testutil.RunGit(t, dir, "add", "release.txt")
	_, _ = testutil.RunGit(t, dir, "commit", "-m", "Add release file")

	// Add a commit to main to force a non-fast-forward merge
	_, _ = testutil.RunGit(t, dir, "checkout", "main")
	testutil.WriteFile(t, dir, "main.txt", "main content")
	_, _ = testutil.RunGit(t, dir, "add", "main.txt")
	_, _ = testutil.RunGit(t, dir, "commit", "-m", "Add main file")

	// Switch back to release branch for finish command
	_, _ = testutil.RunGit(t, dir, "checkout", "release/1.0.0")

	// Install rejecting hooks
	createRejectingHooks(t, dir)

	// Finish with --no-verify
	output, err := testutil.RunGitFlow(t, dir, "release", "finish", "--no-fetch", "--no-verify", "1.0.0")
	if err != nil {
		t.Fatalf("Expected release finish with --no-verify to succeed, but it failed: %v\nOutput: %s", err, output)
	}

	// Verify tag was created
	tagOutput, err := testutil.RunGit(t, dir, "tag", "-l", "1.0.0")
	if err != nil {
		t.Fatalf("Failed to list tags: %v", err)
	}
	if !strings.Contains(tagOutput, "1.0.0") {
		t.Errorf("Expected tag 1.0.0 to be created, got: %s", tagOutput)
	}

	// Verify release branch is deleted
	if testutil.BranchExists(t, dir, "release/1.0.0") {
		t.Error("Release branch should be deleted after successful finish")
	}
}

// TestFinishFeatureDefaultDoesNotSkipVerify tests that without --no-verify, hooks run normally.
// Steps:
// 1. Sets up a test repository and initializes git-flow
// 2. Creates a feature branch
// 3. Adds a test file and commits to the feature branch
// 4. Adds a different commit on develop to force a non-fast-forward merge
// 5. Installs pre-merge-commit and pre-commit hooks that reject (exit code 1)
// 6. Attempts to finish the feature branch without --no-verify flag and without config
// 7. Verifies the finish operation fails (hook is active by default)
// 8. Verifies the feature branch still exists
func TestFinishFeatureDefaultDoesNotSkipVerify(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow
	_, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v", err)
	}

	// Start a feature branch
	_, err = testutil.RunGitFlow(t, dir, "feature", "start", "default-test")
	if err != nil {
		t.Fatalf("Failed to start feature: %v", err)
	}

	// Add a commit to the feature branch
	testutil.WriteFile(t, dir, "feature.txt", "feature content")
	_, _ = testutil.RunGit(t, dir, "add", "feature.txt")
	_, _ = testutil.RunGit(t, dir, "commit", "-m", "Add feature file")

	// Add a commit to develop to force a non-fast-forward merge
	// (Fast-forward merges don't create merge commits and don't trigger pre-merge-commit hooks)
	_, _ = testutil.RunGit(t, dir, "checkout", "develop")
	testutil.WriteFile(t, dir, "develop.txt", "develop content")
	_, _ = testutil.RunGit(t, dir, "add", "develop.txt")
	_, _ = testutil.RunGit(t, dir, "commit", "-m", "Add develop file")

	// Switch back to feature branch for finish command
	_, _ = testutil.RunGit(t, dir, "checkout", "feature/default-test")

	// Install rejecting hooks
	createRejectingHooks(t, dir)

	// Try to finish WITHOUT any noVerify flags or config - should fail
	_, err = testutil.RunGitFlow(t, dir, "feature", "finish", "--no-fetch", "default-test")
	if err == nil {
		t.Fatal("Expected finish to fail due to pre-merge-commit hook (default behavior), but it succeeded")
	}

	// Verify feature branch still exists (finish was blocked)
	if !testutil.BranchExists(t, dir, "feature/default-test") {
		t.Error("Feature branch should still exist when finish failed due to hook")
	}
}

// TestFinishFeatureSquashWithNoVerify tests that --no-verify works with squash merge strategy.
// When using squash merge, a separate Commit call is made after the squash operation.
// This test verifies that --no-verify is correctly passed to both the merge and the final commit.
// Steps:
// 1. Sets up a test repository and initializes git-flow
// 2. Configures squash merge strategy for feature finish
// 3. Creates a feature branch and adds a test file
// 4. Adds a commit on develop to force non-fast-forward merge
// 5. Installs pre-merge-commit and pre-commit hooks that reject (exit code 1)
// 6. Finishes the feature branch with --no-verify flag
// 7. Verifies the finish operation succeeds (hooks bypassed for both merge and commit)
// 8. Verifies the feature branch is deleted
// 9. Verifies changes are squashed into develop
func TestFinishFeatureSquashWithNoVerify(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow
	_, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v", err)
	}

	// Configure squash merge strategy for feature finish
	_, err = testutil.RunGit(t, dir, "config", "gitflow.feature.finish.squash", "true")
	if err != nil {
		t.Fatalf("Failed to set squash merge strategy: %v", err)
	}

	// Start a feature branch
	_, err = testutil.RunGitFlow(t, dir, "feature", "start", "squash-test")
	if err != nil {
		t.Fatalf("Failed to start feature: %v", err)
	}

	// Add a commit to the feature branch
	testutil.WriteFile(t, dir, "feature.txt", "feature content")
	_, _ = testutil.RunGit(t, dir, "add", "feature.txt")
	_, _ = testutil.RunGit(t, dir, "commit", "-m", "Add feature file")

	// Add a commit to develop to ensure we're actually doing a squash merge
	_, _ = testutil.RunGit(t, dir, "checkout", "develop")
	testutil.WriteFile(t, dir, "develop.txt", "develop content")
	_, _ = testutil.RunGit(t, dir, "add", "develop.txt")
	_, _ = testutil.RunGit(t, dir, "commit", "-m", "Add develop file")

	// Switch back to feature branch for finish command
	_, _ = testutil.RunGit(t, dir, "checkout", "feature/squash-test")

	// Install rejecting hooks AFTER setting up the branches
	createRejectingHooks(t, dir)

	// Finish with --no-verify and squash strategy - should succeed (hooks bypassed)
	output, err := testutil.RunGitFlow(t, dir, "feature", "finish", "--no-fetch", "--no-verify", "squash-test")
	if err != nil {
		t.Fatalf("Expected squash finish with --no-verify to succeed, but it failed: %v\nOutput: %s", err, output)
	}

	// Verify feature branch is deleted
	if testutil.BranchExists(t, dir, "feature/squash-test") {
		t.Error("Feature branch should be deleted after successful finish")
	}

	// Verify we're on develop
	currentBranch := testutil.GetCurrentBranch(t, dir)
	if currentBranch != "develop" {
		t.Errorf("Expected to be on develop branch, got: %s", currentBranch)
	}

	// Verify squash strategy was actually used, not the default merge
	if !strings.Contains(output, "Merging using strategy: squash") {
		t.Errorf("Expected output to indicate squash strategy, got: %s", output)
	}

	// Verify the develop tip is a squash commit, not a merge commit: a squash
	// produces a single-parent commit, whereas a merge would have two parents.
	parents, err := testutil.RunGit(t, dir, "rev-list", "--parents", "-n", "1", "develop")
	if err != nil {
		t.Fatalf("Failed to get parent commits: %v", err)
	}
	// Output is the commit hash followed by its parent hashes; a squash commit
	// yields exactly two fields (commit + single parent).
	if fields := len(strings.Fields(strings.TrimSpace(parents))); fields != 2 {
		t.Errorf("Expected a squash (single-parent) commit on develop, got %d parent(s)", fields-1)
	}

	// Verify the feature file exists on develop (changes squashed in)
	featureFile := filepath.Join(dir, "feature.txt")
	if _, err := os.Stat(featureFile); os.IsNotExist(err) {
		t.Error("Feature file should exist on develop after squash merge")
	}
}

// TestFinishFeatureShorthandWithNoVerify tests that --no-verify works with shorthand command.
// The shorthand 'git flow finish' auto-detects the branch type from the current branch.
// Steps:
// 1. Sets up a test repository and initializes git-flow
// 2. Creates a feature branch and adds a test file
// 3. Adds a commit on develop to force non-fast-forward merge
// 4. Switches back to feature branch
// 5. Installs pre-merge-commit and pre-commit hooks that reject (exit code 1)
// 6. Finishes using shorthand: finish --no-verify (auto-detects current branch)
// 7. Verifies the finish operation succeeds (hook bypassed)
// 8. Verifies the feature branch is deleted
func TestFinishFeatureShorthandWithNoVerify(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow
	_, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v", err)
	}

	// Start a feature branch
	_, err = testutil.RunGitFlow(t, dir, "feature", "start", "shorthand-test")
	if err != nil {
		t.Fatalf("Failed to start feature: %v", err)
	}

	// Add a commit to the feature branch
	testutil.WriteFile(t, dir, "feature.txt", "feature content")
	_, _ = testutil.RunGit(t, dir, "add", "feature.txt")
	_, _ = testutil.RunGit(t, dir, "commit", "-m", "Add feature file")

	// Add a commit to develop to force a non-fast-forward merge
	_, _ = testutil.RunGit(t, dir, "checkout", "develop")
	testutil.WriteFile(t, dir, "develop.txt", "develop content")
	_, _ = testutil.RunGit(t, dir, "add", "develop.txt")
	_, _ = testutil.RunGit(t, dir, "commit", "-m", "Add develop file")

	// Switch back to feature branch for finish command
	_, _ = testutil.RunGit(t, dir, "checkout", "feature/shorthand-test")

	// Install rejecting hooks
	createRejectingHooks(t, dir)

	// Finish using shorthand 'finish' (auto-detects branch type from current branch)
	// This tests the shorthand command path in shorthand.go
	output, err := testutil.RunGitFlow(t, dir, "finish", "--no-fetch", "--no-verify")
	if err != nil {
		t.Fatalf("Expected shorthand finish with --no-verify to succeed, but it failed: %v\nOutput: %s", err, output)
	}

	// Verify feature branch is deleted
	if testutil.BranchExists(t, dir, "feature/shorthand-test") {
		t.Error("Feature branch should be deleted after successful finish")
	}

	// Verify we're on develop
	currentBranch := testutil.GetCurrentBranch(t, dir)
	if currentBranch != "develop" {
		t.Errorf("Expected to be on develop branch, got: %s", currentBranch)
	}
}

// TestFinishFeatureInvalidNoVerifyConfig tests behavior with invalid noVerify config value.
// Git's config system treats non-boolean values specially - we verify the behavior is predictable.
// Steps:
// 1. Sets up a test repository and initializes git-flow
// 2. Creates a feature branch and adds a test file
// 3. Adds a commit on develop to force non-fast-forward merge
// 4. Sets invalid config: gitflow.feature.finish.noVerify=invalid
// 5. Installs pre-merge-commit and pre-commit hooks that reject (exit code 1)
// 6. Attempts to finish without --no-verify flag
// 7. Verifies behavior (invalid config is treated as false, hook blocks merge)
func TestFinishFeatureInvalidNoVerifyConfig(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow
	_, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v", err)
	}

	// Set invalid noVerify config value (not a valid boolean)
	_, err = testutil.RunGit(t, dir, "config", "gitflow.feature.finish.noVerify", "invalid")
	if err != nil {
		t.Fatalf("Failed to set noVerify config: %v", err)
	}

	// Start a feature branch
	_, err = testutil.RunGitFlow(t, dir, "feature", "start", "invalid-config-test")
	if err != nil {
		t.Fatalf("Failed to start feature: %v", err)
	}

	// Add a commit to the feature branch
	testutil.WriteFile(t, dir, "feature.txt", "feature content")
	_, _ = testutil.RunGit(t, dir, "add", "feature.txt")
	_, _ = testutil.RunGit(t, dir, "commit", "-m", "Add feature file")

	// Add a commit to develop to force a non-fast-forward merge
	_, _ = testutil.RunGit(t, dir, "checkout", "develop")
	testutil.WriteFile(t, dir, "develop.txt", "develop content")
	_, _ = testutil.RunGit(t, dir, "add", "develop.txt")
	_, _ = testutil.RunGit(t, dir, "commit", "-m", "Add develop file")

	// Switch back to feature branch for finish command
	_, _ = testutil.RunGit(t, dir, "checkout", "feature/invalid-config-test")

	// Install rejecting hooks
	createRejectingHooks(t, dir)

	// Try to finish WITHOUT --no-verify flag - invalid config should be treated as false
	_, err = testutil.RunGitFlow(t, dir, "feature", "finish", "--no-fetch", "invalid-config-test")
	if err == nil {
		t.Fatal("Expected finish to fail due to pre-merge-commit hook (invalid config treated as false), but it succeeded")
	}

	// Verify feature branch still exists (finish was blocked)
	if !testutil.BranchExists(t, dir, "feature/invalid-config-test") {
		t.Error("Feature branch should still exist when finish failed due to hook")
	}
}
