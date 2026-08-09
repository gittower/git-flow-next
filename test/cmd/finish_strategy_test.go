package cmd_test

import (
	"strings"
	"testing"

	"github.com/gittower/git-flow-next/test/testutil"
)

// TestFinishWithRebaseFlag tests that the --rebase flag works correctly.
// Steps:
// 1. Sets up a test repository and initializes git-flow with defaults
// 2. Creates a feature branch with commits
// 3. Makes changes in develop branch
// 4. Finishes the feature branch with --rebase flag
// 5. Verifies the branch was rebased before merging
func TestFinishWithRebaseFlag(t *testing.T) {
	t.Parallel()
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Create and switch to feature branch
	output, err = testutil.RunGitFlow(t, dir, "feature", "start", "rebase-test")
	if err != nil {
		t.Fatalf("Failed to start feature branch: %v\nOutput: %s", err, output)
	}

	// Add a commit to feature branch
	testutil.WriteFile(t, dir, "feature.txt", "feature content")
	_, err = testutil.RunGit(t, dir, "add", "feature.txt")
	if err != nil {
		t.Fatalf("Failed to add feature file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Add feature file")
	if err != nil {
		t.Fatalf("Failed to commit feature file: %v", err)
	}

	// Switch to develop and add a commit
	_, err = testutil.RunGit(t, dir, "checkout", "develop")
	if err != nil {
		t.Fatalf("Failed to switch to develop: %v", err)
	}
	testutil.WriteFile(t, dir, "develop.txt", "develop content")
	_, err = testutil.RunGit(t, dir, "add", "develop.txt")
	if err != nil {
		t.Fatalf("Failed to add develop file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Add develop file")
	if err != nil {
		t.Fatalf("Failed to commit develop file: %v", err)
	}

	// Finish feature branch with --rebase flag
	output, err = testutil.RunGitFlow(t, dir, "feature", "finish", "rebase-test", "--rebase")
	if err != nil {
		t.Fatalf("Failed to finish feature branch: %v\nOutput: %s", err, output)
	}

	// Verify that the output indicates rebase strategy was used
	if !strings.Contains(output, "Merging using strategy: rebase") {
		t.Errorf("Expected output to indicate rebase strategy, got: %s", output)
	}

	// Verify that both files exist in develop
	_, err = testutil.RunGit(t, dir, "checkout", "develop")
	if err != nil {
		t.Fatalf("Failed to checkout develop: %v", err)
	}

	if !testutil.FileExists(t, dir, "feature.txt") {
		t.Error("Expected feature.txt to exist in develop branch")
	}
	if !testutil.FileExists(t, dir, "develop.txt") {
		t.Error("Expected develop.txt to exist in develop branch")
	}
}

// TestFinishWithSquashFlag tests that the --squash flag works correctly.
// Steps:
// 1. Sets up a test repository and initializes git-flow with defaults
// 2. Creates a feature branch with multiple commits
// 3. Finishes the feature branch with --squash flag
// 4. Verifies the commits were squashed into a single commit
func TestFinishWithSquashFlag(t *testing.T) {
	t.Parallel()
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Create and switch to feature branch
	output, err = testutil.RunGitFlow(t, dir, "feature", "start", "squash-test")
	if err != nil {
		t.Fatalf("Failed to start feature branch: %v\nOutput: %s", err, output)
	}

	// Add multiple commits to feature branch
	testutil.WriteFile(t, dir, "file1.txt", "first file")
	_, err = testutil.RunGit(t, dir, "add", "file1.txt")
	if err != nil {
		t.Fatalf("Failed to add first file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Add first file")
	if err != nil {
		t.Fatalf("Failed to commit first file: %v", err)
	}

	testutil.WriteFile(t, dir, "file2.txt", "second file")
	_, err = testutil.RunGit(t, dir, "add", "file2.txt")
	if err != nil {
		t.Fatalf("Failed to add second file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Add second file")
	if err != nil {
		t.Fatalf("Failed to commit second file: %v", err)
	}

	// Count commits in feature branch before finish
	beforeLog, err := testutil.RunGit(t, dir, "log", "--oneline", "develop..HEAD")
	if err != nil {
		t.Fatalf("Failed to get commit log: %v", err)
	}
	beforeCommitCount := len(strings.Split(strings.TrimSpace(beforeLog), "\n"))
	if beforeCommitCount != 2 {
		t.Fatalf("Expected 2 commits in feature branch, got %d", beforeCommitCount)
	}

	// Finish feature branch with --squash flag (keep branch to avoid deletion issues)
	output, err = testutil.RunGitFlow(t, dir, "feature", "finish", "squash-test", "--squash", "--keep")
	if err != nil {
		t.Fatalf("Failed to finish feature branch: %v\nOutput: %s", err, output)
	}

	// Verify that the output indicates squash strategy was used
	if !strings.Contains(output, "Merging using strategy: squash") {
		t.Errorf("Expected output to indicate squash strategy, got: %s", output)
	}

	// Verify that both files exist in develop
	_, err = testutil.RunGit(t, dir, "checkout", "develop")
	if err != nil {
		t.Fatalf("Failed to checkout develop: %v", err)
	}

	if !testutil.FileExists(t, dir, "file1.txt") {
		t.Error("Expected file1.txt to exist in develop branch")
	}
	if !testutil.FileExists(t, dir, "file2.txt") {
		t.Error("Expected file2.txt to exist in develop branch")
	}
}

// TestFinishWithNoFFFlag tests that the --no-ff flag works correctly.
// Steps:
// 1. Sets up a test repository and initializes git-flow with defaults
// 2. Creates a feature branch with commits
// 3. Finishes the feature branch with --no-ff flag
// 4. Verifies a merge commit was created even for fast-forward case
func TestFinishWithNoFFFlag(t *testing.T) {
	t.Parallel()
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Create and switch to feature branch
	output, err = testutil.RunGitFlow(t, dir, "feature", "start", "no-ff-test")
	if err != nil {
		t.Fatalf("Failed to start feature branch: %v\nOutput: %s", err, output)
	}

	// Add a commit to feature branch
	testutil.WriteFile(t, dir, "feature.txt", "feature content")
	_, err = testutil.RunGit(t, dir, "add", "feature.txt")
	if err != nil {
		t.Fatalf("Failed to add feature file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Add feature file")
	if err != nil {
		t.Fatalf("Failed to commit feature file: %v", err)
	}

	// Finish feature branch with --no-ff flag
	output, err = testutil.RunGitFlow(t, dir, "feature", "finish", "no-ff-test", "--no-ff")
	if err != nil {
		t.Fatalf("Failed to finish feature branch: %v\nOutput: %s", err, output)
	}

	// Switch to develop branch to check merge commit
	_, err = testutil.RunGit(t, dir, "checkout", "develop")
	if err != nil {
		t.Fatalf("Failed to checkout develop: %v", err)
	}

	// Check that the latest commit has multiple parents (indicating a merge commit)
	parents, err := testutil.RunGit(t, dir, "rev-list", "--parents", "-n", "1", "HEAD")
	if err != nil {
		t.Fatalf("Failed to get parent commits: %v", err)
	}

	// A merge commit should have at least 2 parents (so at least 3 hashes in the output)
	parentCount := len(strings.Fields(strings.TrimSpace(parents)))
	if parentCount < 3 {
		t.Errorf("Expected merge commit with multiple parents, got %d parents", parentCount-1)
	}

	// Verify that feature file exists in develop
	if !testutil.FileExists(t, dir, "feature.txt") {
		t.Error("Expected feature.txt to exist in develop branch")
	}
}

// TestMergeStrategyFlagOverridesConfig tests that command-line flags override configuration.
// Steps:
// 1. Sets up a test repository and initializes git-flow with defaults
// 2. Sets configuration to use merge strategy
// 3. Creates a feature branch
// 4. Finishes with --rebase flag to override config
// 5. Verifies rebase strategy was used instead of configured merge
func TestMergeStrategyFlagOverridesConfig(t *testing.T) {
	t.Parallel()
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Set configuration to use merge strategy for features
	_, err = testutil.RunGit(t, dir, "config", "gitflow.branch.feature.upstreamstrategy", "merge")
	if err != nil {
		t.Fatalf("Failed to set upstream strategy config: %v", err)
	}

	// Verify the config was set
	strategy, err := testutil.RunGit(t, dir, "config", "--get", "gitflow.branch.feature.upstreamstrategy")
	if err != nil {
		t.Fatalf("Failed to get upstream strategy config: %v", err)
	}
	if strings.TrimSpace(strategy) != "merge" {
		t.Fatalf("Expected merge strategy in config, got: %s", strings.TrimSpace(strategy))
	}

	// Create and switch to feature branch
	output, err = testutil.RunGitFlow(t, dir, "feature", "start", "config-override-test")
	if err != nil {
		t.Fatalf("Failed to start feature branch: %v\nOutput: %s", err, output)
	}

	// Add a commit to feature branch
	testutil.WriteFile(t, dir, "feature.txt", "feature content")
	_, err = testutil.RunGit(t, dir, "add", "feature.txt")
	if err != nil {
		t.Fatalf("Failed to add feature file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Add feature file")
	if err != nil {
		t.Fatalf("Failed to commit feature file: %v", err)
	}

	// Finish feature branch with --rebase flag (should override config)
	output, err = testutil.RunGitFlow(t, dir, "feature", "finish", "config-override-test", "--rebase")
	if err != nil {
		t.Fatalf("Failed to finish feature branch: %v\nOutput: %s", err, output)
	}

	// Verify that rebase strategy was used (override config)
	if !strings.Contains(output, "Merging using strategy: rebase") {
		t.Errorf("Expected rebase strategy to override config, got: %s", output)
	}

	// Verify that feature file exists in develop
	_, err = testutil.RunGit(t, dir, "checkout", "develop")
	if err != nil {
		t.Fatalf("Failed to checkout develop: %v", err)
	}

	if !testutil.FileExists(t, dir, "feature.txt") {
		t.Error("Expected feature.txt to exist in develop branch")
	}
}

// TestFinishWithSquashMessageFlag tests that the --squash-message flag customizes the commit message.
// Steps:
// 1. Sets up a test repository and initializes git-flow with defaults
// 2. Creates a feature branch with multiple commits
// 3. Finishes the feature branch with --squash and --squash-message flags
// 4. Verifies the custom commit message was used
func TestFinishWithSquashMessageFlag(t *testing.T) {
	t.Parallel()
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Create and switch to feature branch
	output, err = testutil.RunGitFlow(t, dir, "feature", "start", "squash-msg-test")
	if err != nil {
		t.Fatalf("Failed to start feature branch: %v\nOutput: %s", err, output)
	}

	// Add multiple commits to feature branch
	testutil.WriteFile(t, dir, "file1.txt", "first file")
	_, err = testutil.RunGit(t, dir, "add", "file1.txt")
	if err != nil {
		t.Fatalf("Failed to add first file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Add first file")
	if err != nil {
		t.Fatalf("Failed to commit first file: %v", err)
	}

	testutil.WriteFile(t, dir, "file2.txt", "second file")
	_, err = testutil.RunGit(t, dir, "add", "file2.txt")
	if err != nil {
		t.Fatalf("Failed to add second file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Add second file")
	if err != nil {
		t.Fatalf("Failed to commit second file: %v", err)
	}

	// Finish feature branch with custom squash message
	customMessage := "feat: add login functionality"
	output, err = testutil.RunGitFlow(t, dir, "feature", "finish", "squash-msg-test", "--squash", "--squash-message", customMessage, "--keep")
	if err != nil {
		t.Fatalf("Failed to finish feature branch: %v\nOutput: %s", err, output)
	}

	// Verify that squash strategy was used
	if !strings.Contains(output, "Merging using strategy: squash") {
		t.Errorf("Expected output to indicate squash strategy, got: %s", output)
	}

	// Verify the commit message in develop
	_, err = testutil.RunGit(t, dir, "checkout", "develop")
	if err != nil {
		t.Fatalf("Failed to checkout develop: %v", err)
	}

	// Get the last commit message
	commitMsg, err := testutil.RunGit(t, dir, "log", "-1", "--format=%s")
	if err != nil {
		t.Fatalf("Failed to get commit message: %v", err)
	}

	if strings.TrimSpace(commitMsg) != customMessage {
		t.Errorf("Expected commit message '%s', got '%s'", customMessage, strings.TrimSpace(commitMsg))
	}
}

// TestSquashMessageIgnoredWithoutSquashStrategy tests that --squash-message is ignored when not using squash strategy.
// Steps:
// 1. Sets up a test repository and initializes git-flow with defaults
// 2. Creates a feature branch with commits
// 3. Finishes with --squash-message but WITHOUT --squash flag
// 4. Verifies the commit message is NOT the custom squash message (uses merge commit message instead)
func TestSquashMessageIgnoredWithoutSquashStrategy(t *testing.T) {
	t.Parallel()
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Create and switch to feature branch
	output, err = testutil.RunGitFlow(t, dir, "feature", "start", "no-squash-test")
	if err != nil {
		t.Fatalf("Failed to start feature branch: %v\nOutput: %s", err, output)
	}

	// Add a commit to feature branch
	testutil.WriteFile(t, dir, "feature.txt", "feature content")
	_, err = testutil.RunGit(t, dir, "add", "feature.txt")
	if err != nil {
		t.Fatalf("Failed to add feature file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Add feature file")
	if err != nil {
		t.Fatalf("Failed to commit feature file: %v", err)
	}

	// Finish feature branch with squash-message but WITHOUT --squash flag (using default merge strategy)
	ignoredMessage := "this message should be ignored"
	output, err = testutil.RunGitFlow(t, dir, "feature", "finish", "no-squash-test", "--squash-message", ignoredMessage, "--no-ff", "--keep")
	if err != nil {
		t.Fatalf("Failed to finish feature branch: %v\nOutput: %s", err, output)
	}

	// Verify that merge strategy was used (not squash)
	if strings.Contains(output, "Merging using strategy: squash") {
		t.Errorf("Expected merge strategy, but squash was used: %s", output)
	}

	// Verify the commit message in develop is NOT the squash message
	_, err = testutil.RunGit(t, dir, "checkout", "develop")
	if err != nil {
		t.Fatalf("Failed to checkout develop: %v", err)
	}

	// Get the last commit message
	commitMsg, err := testutil.RunGit(t, dir, "log", "-1", "--format=%s")
	if err != nil {
		t.Fatalf("Failed to get commit message: %v", err)
	}

	// The commit message should be a merge commit message, NOT the squash message
	if strings.TrimSpace(commitMsg) == ignoredMessage {
		t.Errorf("Squash message should be ignored when not using squash strategy, but got: '%s'", commitMsg)
	}

	// It should be a merge commit message
	if !strings.Contains(commitMsg, "Merge branch") {
		t.Errorf("Expected merge commit message, got: '%s'", commitMsg)
	}
}

// TestSquashMessagePreservedAfterConflict tests that --squash-message is preserved in merge state after conflict.
// Steps:
// 1. Sets up a test repository and initializes git-flow with defaults
// 2. Creates a feature branch with conflicting changes
// 3. Attempts to finish with --squash --squash-message (causes conflict)
// 4. Resolves conflict and continues WITHOUT --squash-message
// 5. Verifies the original custom squash message was used (preserved from state)
func TestSquashMessagePreservedAfterConflict(t *testing.T) {
	t.Parallel()
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Create conflicting content in develop
	testutil.RunGit(t, dir, "checkout", "develop")
	testutil.WriteFile(t, dir, "conflict.txt", "Develop version\nLine 2\nLine 3")
	testutil.RunGit(t, dir, "add", "conflict.txt")
	testutil.RunGit(t, dir, "commit", "-m", "Develop changes")

	// Create feature with conflicting changes
	output, err = testutil.RunGitFlow(t, dir, "feature", "start", "squash-conflict")
	if err != nil {
		t.Fatalf("Failed to create feature: %v\nOutput: %s", err, output)
	}

	testutil.WriteFile(t, dir, "conflict.txt", "Feature version\nLine 2 modified\nLine 3")
	testutil.RunGit(t, dir, "add", "conflict.txt")
	testutil.RunGit(t, dir, "commit", "-m", "Feature changes")

	// Meanwhile, add more changes to develop to create conflict
	testutil.RunGit(t, dir, "checkout", "develop")
	testutil.WriteFile(t, dir, "conflict.txt", "Develop version updated\nLine 2\nLine 3 modified")
	testutil.RunGit(t, dir, "add", "conflict.txt")
	testutil.RunGit(t, dir, "commit", "-m", "More develop changes")

	// Try to finish feature with squash (will conflict)
	customMessage := "feat: squash merged after conflict resolution"
	testutil.RunGit(t, dir, "checkout", "feature/squash-conflict")
	output, _ = testutil.RunGitFlow(t, dir, "feature", "finish", "squash-conflict", "--squash", "--squash-message", customMessage)

	// Verify conflict detected
	if !strings.Contains(output, "conflict") {
		t.Fatal("Expected merge conflict to be detected")
	}

	// Verify merge state exists with squash strategy and message
	state, err := testutil.LoadMergeState(t, dir)
	if err != nil || state == nil {
		t.Fatal("Expected merge state to exist after conflict")
	}

	if state.MergeStrategy != "squash" {
		t.Errorf("Expected MergeStrategy to be 'squash', got: %s", state.MergeStrategy)
	}

	if state.SquashMessage != customMessage {
		t.Errorf("Expected SquashMessage in state to be '%s', got: '%s'", customMessage, state.SquashMessage)
	}

	// Resolve conflict
	testutil.WriteFile(t, dir, "conflict.txt", "Resolved version\nLine 2 resolved\nLine 3 resolved")
	testutil.RunGit(t, dir, "add", "conflict.txt")

	// Continue finish operation WITHOUT --squash-message (should use preserved message from state)
	output, err = testutil.RunGitFlow(t, dir, "feature", "finish", "--continue", "squash-conflict")
	if err != nil {
		t.Fatalf("Failed to continue finish: %v\nOutput: %s", err, output)
	}

	// Verify success
	if !strings.Contains(output, "Successfully finished") {
		t.Error("Expected successful finish message")
	}

	// Verify the commit message in develop uses the preserved message
	testutil.RunGit(t, dir, "checkout", "develop")
	commitMsg, err := testutil.RunGit(t, dir, "log", "-1", "--format=%s")
	if err != nil {
		t.Fatalf("Failed to get commit message: %v", err)
	}

	if strings.TrimSpace(commitMsg) != customMessage {
		t.Errorf("Expected commit message '%s', got '%s'", customMessage, strings.TrimSpace(commitMsg))
	}
}

// TestSquashMessageOverrideOnContinue tests that --squash-message can be overridden on continue.
// Steps:
// 1. Sets up a test repository and initializes git-flow with defaults
// 2. Creates a feature branch with conflicting changes
// 3. Attempts to finish with --squash --squash-message (causes conflict)
// 4. Resolves conflict and continues WITH a different --squash-message
// 5. Verifies the new message was used (overriding the preserved one)
func TestSquashMessageOverrideOnContinue(t *testing.T) {
	t.Parallel()
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Create conflicting content in develop
	testutil.RunGit(t, dir, "checkout", "develop")
	testutil.WriteFile(t, dir, "conflict.txt", "Develop version\nLine 2\nLine 3")
	testutil.RunGit(t, dir, "add", "conflict.txt")
	testutil.RunGit(t, dir, "commit", "-m", "Develop changes")

	// Create feature with conflicting changes
	output, err = testutil.RunGitFlow(t, dir, "feature", "start", "squash-override")
	if err != nil {
		t.Fatalf("Failed to create feature: %v\nOutput: %s", err, output)
	}

	testutil.WriteFile(t, dir, "conflict.txt", "Feature version\nLine 2 modified\nLine 3")
	testutil.RunGit(t, dir, "add", "conflict.txt")
	testutil.RunGit(t, dir, "commit", "-m", "Feature changes")

	// Meanwhile, add more changes to develop to create conflict
	testutil.RunGit(t, dir, "checkout", "develop")
	testutil.WriteFile(t, dir, "conflict.txt", "Develop version updated\nLine 2\nLine 3 modified")
	testutil.RunGit(t, dir, "add", "conflict.txt")
	testutil.RunGit(t, dir, "commit", "-m", "More develop changes")

	// Try to finish feature with squash (will conflict)
	originalMessage := "feat: original message"
	testutil.RunGit(t, dir, "checkout", "feature/squash-override")
	output, _ = testutil.RunGitFlow(t, dir, "feature", "finish", "squash-override", "--squash", "--squash-message", originalMessage)

	// Verify conflict detected
	if !strings.Contains(output, "conflict") {
		t.Fatal("Expected merge conflict to be detected")
	}

	// Resolve conflict
	testutil.WriteFile(t, dir, "conflict.txt", "Resolved version\nLine 2 resolved\nLine 3 resolved")
	testutil.RunGit(t, dir, "add", "conflict.txt")

	// Continue finish operation WITH a different --squash-message (should override)
	overrideMessage := "feat: overridden message after conflict"
	output, err = testutil.RunGitFlow(t, dir, "feature", "finish", "--continue", "--squash-message", overrideMessage, "squash-override")
	if err != nil {
		t.Fatalf("Failed to continue finish: %v\nOutput: %s", err, output)
	}

	// Verify success
	if !strings.Contains(output, "Successfully finished") {
		t.Error("Expected successful finish message")
	}

	// Verify the commit message in develop uses the override message
	testutil.RunGit(t, dir, "checkout", "develop")
	commitMsg, err := testutil.RunGit(t, dir, "log", "-1", "--format=%s")
	if err != nil {
		t.Fatalf("Failed to get commit message: %v", err)
	}

	if strings.TrimSpace(commitMsg) != overrideMessage {
		t.Errorf("Expected commit message '%s', got '%s'", overrideMessage, strings.TrimSpace(commitMsg))
	}

	// Make sure it's NOT the original message
	if strings.TrimSpace(commitMsg) == originalMessage {
		t.Errorf("Expected override message to be used, but got original message: '%s'", commitMsg)
	}
}

// TestFinishFeatureBranchWithMergeMessage tests that a custom merge message is used for the upstream merge.
// Steps:
// 1. Sets up a test repository and initializes git-flow with defaults
// 2. Creates a feature branch and adds a commit
// 3. Finishes the feature branch with --merge-message "feat: custom merge message"
// 4. Verifies the merge commit message matches the custom message
// 5. Verifies the feature branch is deleted
func TestFinishFeatureBranchWithMergeMessage(t *testing.T) {
	t.Parallel()
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Create and switch to feature branch
	output, err = testutil.RunGitFlow(t, dir, "feature", "start", "merge-msg-test")
	if err != nil {
		t.Fatalf("Failed to start feature branch: %v\nOutput: %s", err, output)
	}

	// Add a commit to feature branch
	testutil.WriteFile(t, dir, "feature.txt", "feature content")
	_, err = testutil.RunGit(t, dir, "add", "feature.txt")
	if err != nil {
		t.Fatalf("Failed to add feature file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Add feature file")
	if err != nil {
		t.Fatalf("Failed to commit feature file: %v", err)
	}

	// Switch to develop and add a commit to prevent fast-forward merge
	_, err = testutil.RunGit(t, dir, "checkout", "develop")
	if err != nil {
		t.Fatalf("Failed to switch to develop: %v", err)
	}
	testutil.WriteFile(t, dir, "develop.txt", "develop content")
	_, err = testutil.RunGit(t, dir, "add", "develop.txt")
	if err != nil {
		t.Fatalf("Failed to add develop file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Add develop file")
	if err != nil {
		t.Fatalf("Failed to commit develop file: %v", err)
	}

	// Finish feature branch with custom merge message
	customMessage := "feat: custom merge message for feature"
	output, err = testutil.RunGitFlow(t, dir, "feature", "finish", "merge-msg-test", "--merge-message", customMessage)
	if err != nil {
		t.Fatalf("Failed to finish feature branch: %v\nOutput: %s", err, output)
	}

	// Verify that the feature file exists in develop
	_, err = testutil.RunGit(t, dir, "checkout", "develop")
	if err != nil {
		t.Fatalf("Failed to checkout develop: %v", err)
	}

	if !testutil.FileExists(t, dir, "feature.txt") {
		t.Error("Expected feature.txt to exist in develop branch")
	}

	// Get the last commit message
	commitMsg, err := testutil.RunGit(t, dir, "log", "-1", "--format=%s")
	if err != nil {
		t.Fatalf("Failed to get commit message: %v", err)
	}

	if strings.TrimSpace(commitMsg) != customMessage {
		t.Errorf("Expected commit message '%s', got '%s'", customMessage, strings.TrimSpace(commitMsg))
	}

	// Verify feature branch was deleted
	if testutil.BranchExists(t, dir, "feature/merge-msg-test") {
		t.Error("Expected feature branch to be deleted")
	}
}

// TestFinishReleaseBranchWithUpdateMessage tests that a custom update message is used for child branch updates.
// Steps:
// 1. Sets up a test repository and initializes git-flow with defaults
// 2. Creates a release branch and adds a commit
// 3. Finishes the release branch with --update-message "chore: sync develop from main"
// 4. Verifies the release merges into main successfully
// 5. Verifies the develop branch update uses the custom message
func TestFinishReleaseBranchWithUpdateMessage(t *testing.T) {
	t.Parallel()
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Create and switch to release branch
	output, err = testutil.RunGitFlow(t, dir, "release", "start", "1.0.0")
	if err != nil {
		t.Fatalf("Failed to start release branch: %v\nOutput: %s", err, output)
	}

	// Add a commit to release branch
	testutil.WriteFile(t, dir, "release.txt", "release content")
	_, err = testutil.RunGit(t, dir, "add", "release.txt")
	if err != nil {
		t.Fatalf("Failed to add release file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Add release file")
	if err != nil {
		t.Fatalf("Failed to commit release file: %v", err)
	}

	// Finish release branch with custom update message
	customUpdateMessage := "chore: sync develop from main after release 1.0.0"
	output, err = testutil.RunGitFlow(t, dir, "release", "finish", "1.0.0", "--update-message", customUpdateMessage)
	if err != nil {
		t.Fatalf("Failed to finish release branch: %v\nOutput: %s", err, output)
	}

	// Verify that release file exists in main
	_, err = testutil.RunGit(t, dir, "checkout", "main")
	if err != nil {
		t.Fatalf("Failed to checkout main: %v", err)
	}

	if !testutil.FileExists(t, dir, "release.txt") {
		t.Error("Expected release.txt to exist in main branch")
	}

	// Checkout develop and verify the update commit message
	_, err = testutil.RunGit(t, dir, "checkout", "develop")
	if err != nil {
		t.Fatalf("Failed to checkout develop: %v", err)
	}

	if !testutil.FileExists(t, dir, "release.txt") {
		t.Error("Expected release.txt to exist in develop branch")
	}

	// Get the last commit message on develop (should be the update commit)
	commitMsg, err := testutil.RunGit(t, dir, "log", "-1", "--format=%s")
	if err != nil {
		t.Fatalf("Failed to get commit message: %v", err)
	}

	if strings.TrimSpace(commitMsg) != customUpdateMessage {
		t.Errorf("Expected update message '%s', got '%s'", customUpdateMessage, strings.TrimSpace(commitMsg))
	}
}

// TestFinishWithMergeMessagePersistsThroughConflict tests that custom message survives --continue.
// Steps:
// 1. Sets up a test repository and initializes git-flow
// 2. Creates a feature branch with changes
// 3. Creates conflicting changes on develop
// 4. Attempts to finish the feature with --merge-message "custom: merge"
// 5. Verifies merge conflict occurs
// 6. Resolves the conflict and stages files
// 7. Runs finish --continue
// 8. Verifies the final merge commit uses the custom message
func TestFinishWithMergeMessagePersistsThroughConflict(t *testing.T) {
	t.Parallel()
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Create conflicting content in develop
	testutil.RunGit(t, dir, "checkout", "develop")
	testutil.WriteFile(t, dir, "conflict.txt", "Develop version\nLine 2\nLine 3")
	testutil.RunGit(t, dir, "add", "conflict.txt")
	testutil.RunGit(t, dir, "commit", "-m", "Develop changes")

	// Create feature with conflicting changes
	output, err = testutil.RunGitFlow(t, dir, "feature", "start", "merge-msg-conflict")
	if err != nil {
		t.Fatalf("Failed to create feature: %v\nOutput: %s", err, output)
	}

	testutil.WriteFile(t, dir, "conflict.txt", "Feature version\nLine 2 modified\nLine 3")
	testutil.RunGit(t, dir, "add", "conflict.txt")
	testutil.RunGit(t, dir, "commit", "-m", "Feature changes")

	// Meanwhile, add more changes to develop to create conflict
	testutil.RunGit(t, dir, "checkout", "develop")
	testutil.WriteFile(t, dir, "conflict.txt", "Develop version updated\nLine 2\nLine 3 modified")
	testutil.RunGit(t, dir, "add", "conflict.txt")
	testutil.RunGit(t, dir, "commit", "-m", "More develop changes")

	// Try to finish feature with custom merge message (will conflict)
	customMessage := "feat: merged after conflict resolution"
	testutil.RunGit(t, dir, "checkout", "feature/merge-msg-conflict")
	output, _ = testutil.RunGitFlow(t, dir, "feature", "finish", "merge-msg-conflict", "--merge-message", customMessage)

	// Verify conflict detected
	if !strings.Contains(output, "conflict") {
		t.Fatal("Expected merge conflict to be detected")
	}

	// Verify merge state exists with merge message
	state, err := testutil.LoadMergeState(t, dir)
	if err != nil || state == nil {
		t.Fatal("Expected merge state to exist after conflict")
	}

	if state.MergeMessage != customMessage {
		t.Errorf("Expected MergeMessage in state to be '%s', got: '%s'", customMessage, state.MergeMessage)
	}

	// Resolve conflict
	testutil.WriteFile(t, dir, "conflict.txt", "Resolved version\nLine 2 resolved\nLine 3 resolved")
	testutil.RunGit(t, dir, "add", "conflict.txt")

	// Continue finish operation WITHOUT --merge-message (should use preserved message from state)
	output, err = testutil.RunGitFlow(t, dir, "feature", "finish", "--continue", "merge-msg-conflict")
	if err != nil {
		t.Fatalf("Failed to continue finish: %v\nOutput: %s", err, output)
	}

	// Verify success
	if !strings.Contains(output, "Successfully finished") {
		t.Error("Expected successful finish message")
	}

	// Verify the commit message in develop uses the preserved message
	testutil.RunGit(t, dir, "checkout", "develop")
	commitMsg, err := testutil.RunGit(t, dir, "log", "-1", "--format=%s")
	if err != nil {
		t.Fatalf("Failed to get commit message: %v", err)
	}

	if strings.TrimSpace(commitMsg) != customMessage {
		t.Errorf("Expected commit message '%s', got '%s'", customMessage, strings.TrimSpace(commitMsg))
	}
}

// TestFinishWithMergeMessageUsedAfterRebaseStrategy tests message is used for final merge after rebase.
// Steps:
// 1. Sets up a test repository and initializes git-flow
// 2. Creates a feature branch with commits
// 3. Adds commits to develop to make rebase meaningful
// 4. Finishes with --rebase --merge-message "custom: rebase merge"
// 5. Verifies the final merge commit (after rebase) uses the custom message
func TestFinishWithMergeMessageUsedAfterRebaseStrategy(t *testing.T) {
	t.Parallel()
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Create and switch to feature branch
	output, err = testutil.RunGitFlow(t, dir, "feature", "start", "rebase-msg-test")
	if err != nil {
		t.Fatalf("Failed to start feature branch: %v\nOutput: %s", err, output)
	}

	// Add a commit to feature branch
	testutil.WriteFile(t, dir, "feature.txt", "feature content")
	_, err = testutil.RunGit(t, dir, "add", "feature.txt")
	if err != nil {
		t.Fatalf("Failed to add feature file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Add feature file")
	if err != nil {
		t.Fatalf("Failed to commit feature file: %v", err)
	}

	// Switch to develop and add a commit
	_, err = testutil.RunGit(t, dir, "checkout", "develop")
	if err != nil {
		t.Fatalf("Failed to switch to develop: %v", err)
	}
	testutil.WriteFile(t, dir, "develop.txt", "develop content")
	_, err = testutil.RunGit(t, dir, "add", "develop.txt")
	if err != nil {
		t.Fatalf("Failed to add develop file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Add develop file")
	if err != nil {
		t.Fatalf("Failed to commit develop file: %v", err)
	}

	// Finish feature branch with rebase and custom merge message, using --no-ff to force merge commit
	customMessage := "feat: rebased and merged with custom message"
	output, err = testutil.RunGitFlow(t, dir, "feature", "finish", "rebase-msg-test", "--rebase", "--merge-message", customMessage, "--no-ff")
	if err != nil {
		t.Fatalf("Failed to finish feature branch: %v\nOutput: %s", err, output)
	}

	// Verify that rebase strategy was used
	if !strings.Contains(output, "Merging using strategy: rebase") {
		t.Errorf("Expected output to indicate rebase strategy, got: %s", output)
	}

	// Verify the commit message in develop
	_, err = testutil.RunGit(t, dir, "checkout", "develop")
	if err != nil {
		t.Fatalf("Failed to checkout develop: %v", err)
	}

	// Get the last commit message (should be the merge commit with custom message)
	commitMsg, err := testutil.RunGit(t, dir, "log", "-1", "--format=%s")
	if err != nil {
		t.Fatalf("Failed to get commit message: %v", err)
	}

	if strings.TrimSpace(commitMsg) != customMessage {
		t.Errorf("Expected commit message '%s', got '%s'", customMessage, strings.TrimSpace(commitMsg))
	}
}

// TestFinishWithoutMergeMessageUsesDefault tests that default message is used when flag is omitted.
// Steps:
// 1. Sets up a test repository and initializes git-flow
// 2. Creates and finishes a feature branch without --merge-message (using --no-ff to force merge commit)
// 3. Verifies the merge commit uses the default "Merge branch 'feature/...' into develop" format
func TestFinishWithoutMergeMessageUsesDefault(t *testing.T) {
	t.Parallel()
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Create and switch to feature branch
	output, err = testutil.RunGitFlow(t, dir, "feature", "start", "default-msg-test")
	if err != nil {
		t.Fatalf("Failed to start feature branch: %v\nOutput: %s", err, output)
	}

	// Add a commit to feature branch
	testutil.WriteFile(t, dir, "feature.txt", "feature content")
	_, err = testutil.RunGit(t, dir, "add", "feature.txt")
	if err != nil {
		t.Fatalf("Failed to add feature file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Add feature file")
	if err != nil {
		t.Fatalf("Failed to commit feature file: %v", err)
	}

	// Switch to develop and add a commit to prevent fast-forward merge
	_, err = testutil.RunGit(t, dir, "checkout", "develop")
	if err != nil {
		t.Fatalf("Failed to switch to develop: %v", err)
	}
	testutil.WriteFile(t, dir, "develop.txt", "develop content")
	_, err = testutil.RunGit(t, dir, "add", "develop.txt")
	if err != nil {
		t.Fatalf("Failed to add develop file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Add develop file")
	if err != nil {
		t.Fatalf("Failed to commit develop file: %v", err)
	}

	// Finish feature branch without custom merge message
	output, err = testutil.RunGitFlow(t, dir, "feature", "finish", "default-msg-test")
	if err != nil {
		t.Fatalf("Failed to finish feature branch: %v\nOutput: %s", err, output)
	}

	// Verify the commit message in develop
	_, err = testutil.RunGit(t, dir, "checkout", "develop")
	if err != nil {
		t.Fatalf("Failed to checkout develop: %v", err)
	}

	// Get the last commit message
	commitMsg, err := testutil.RunGit(t, dir, "log", "-1", "--format=%s")
	if err != nil {
		t.Fatalf("Failed to get commit message: %v", err)
	}

	// It should be a default merge commit message
	if !strings.Contains(commitMsg, "Merge branch") && !strings.Contains(commitMsg, "feature/default-msg-test") {
		t.Errorf("Expected default merge commit message with branch name, got: '%s'", commitMsg)
	}
}

// TestFinishMergeMessageStatePersistence tests that messages are correctly saved in state file.
// Steps:
// 1. Sets up a test repository and initializes git-flow
// 2. Creates a feature branch with conflicting changes
// 3. Runs finish with --merge-message and --update-message
// 4. Verifies the merge state file contains both messages
// 5. Aborts the finish operation to clean up
func TestFinishMergeMessageStatePersistence(t *testing.T) {
	t.Parallel()
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Create conflicting content in develop
	testutil.RunGit(t, dir, "checkout", "develop")
	testutil.WriteFile(t, dir, "conflict.txt", "Develop version")
	testutil.RunGit(t, dir, "add", "conflict.txt")
	testutil.RunGit(t, dir, "commit", "-m", "Develop changes")

	// Create feature with conflicting changes
	output, err = testutil.RunGitFlow(t, dir, "feature", "start", "state-persist")
	if err != nil {
		t.Fatalf("Failed to create feature: %v\nOutput: %s", err, output)
	}

	testutil.WriteFile(t, dir, "conflict.txt", "Feature version")
	testutil.RunGit(t, dir, "add", "conflict.txt")
	testutil.RunGit(t, dir, "commit", "-m", "Feature changes")

	// Meanwhile, add more changes to develop to create conflict
	testutil.RunGit(t, dir, "checkout", "develop")
	testutil.WriteFile(t, dir, "conflict.txt", "Develop version updated")
	testutil.RunGit(t, dir, "add", "conflict.txt")
	testutil.RunGit(t, dir, "commit", "-m", "More develop changes")

	// Try to finish feature with both custom messages (will conflict)
	mergeMessage := "feat: custom merge message"
	updateMessage := "chore: custom update message"
	testutil.RunGit(t, dir, "checkout", "feature/state-persist")
	output, _ = testutil.RunGitFlow(t, dir, "feature", "finish", "state-persist",
		"--merge-message", mergeMessage,
		"--update-message", updateMessage)

	// Verify conflict detected
	if !strings.Contains(output, "conflict") {
		t.Fatal("Expected merge conflict to be detected")
	}

	// Verify merge state exists with both messages
	state, err := testutil.LoadMergeState(t, dir)
	if err != nil || state == nil {
		t.Fatal("Expected merge state to exist after conflict")
	}

	if state.MergeMessage != mergeMessage {
		t.Errorf("Expected MergeMessage in state to be '%s', got: '%s'", mergeMessage, state.MergeMessage)
	}

	if state.UpdateMessage != updateMessage {
		t.Errorf("Expected UpdateMessage in state to be '%s', got: '%s'", updateMessage, state.UpdateMessage)
	}

	// Abort to clean up
	_, err = testutil.RunGitFlow(t, dir, "feature", "finish", "--abort", "state-persist")
	if err != nil {
		t.Logf("Abort may have partial failure (expected): %v", err)
	}
}

// TestMergeStrategyConfigUsedByDefault tests that branch configuration is used when no flags are provided.
// Steps:
// 1. Sets up a test repository and initializes git-flow with defaults
// 2. Sets configuration to use rebase strategy
// 3. Creates a feature branch
// 4. Finishes without any flags
// 5. Verifies rebase strategy from config was used
func TestMergeStrategyConfigUsedByDefault(t *testing.T) {
	t.Parallel()
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Set configuration to use rebase strategy for features
	_, err = testutil.RunGit(t, dir, "config", "gitflow.branch.feature.upstreamstrategy", "rebase")
	if err != nil {
		t.Fatalf("Failed to set upstream strategy config: %v", err)
	}

	// Create and switch to feature branch
	output, err = testutil.RunGitFlow(t, dir, "feature", "start", "config-default-test")
	if err != nil {
		t.Fatalf("Failed to start feature branch: %v\nOutput: %s", err, output)
	}

	// Add a commit to feature branch
	testutil.WriteFile(t, dir, "feature.txt", "feature content")
	_, err = testutil.RunGit(t, dir, "add", "feature.txt")
	if err != nil {
		t.Fatalf("Failed to add feature file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Add feature file")
	if err != nil {
		t.Fatalf("Failed to commit feature file: %v", err)
	}

	// Switch to develop and add a commit
	_, err = testutil.RunGit(t, dir, "checkout", "develop")
	if err != nil {
		t.Fatalf("Failed to switch to develop: %v", err)
	}
	testutil.WriteFile(t, dir, "develop.txt", "develop content")
	_, err = testutil.RunGit(t, dir, "add", "develop.txt")
	if err != nil {
		t.Fatalf("Failed to add develop file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Add develop file")
	if err != nil {
		t.Fatalf("Failed to commit develop file: %v", err)
	}

	// Finish feature branch without any flags (should use config)
	output, err = testutil.RunGitFlow(t, dir, "feature", "finish", "config-default-test")
	if err != nil {
		t.Fatalf("Failed to finish feature branch: %v\nOutput: %s", err, output)
	}

	// Verify that rebase strategy from config was used
	if !strings.Contains(output, "Merging using strategy: rebase") {
		t.Errorf("Expected rebase strategy from config to be used, got: %s", output)
	}

	// Verify that both files exist in develop
	_, err = testutil.RunGit(t, dir, "checkout", "develop")
	if err != nil {
		t.Fatalf("Failed to checkout develop: %v", err)
	}

	if !testutil.FileExists(t, dir, "feature.txt") {
		t.Error("Expected feature.txt to exist in develop branch")
	}
	if !testutil.FileExists(t, dir, "develop.txt") {
		t.Error("Expected develop.txt to exist in develop branch")
	}
}

// TestFinishWithMergeMessagePlaceholders tests that placeholders in --merge-message are expanded correctly.
// Steps:
// 1. Sets up a test repository and initializes git-flow with defaults
// 2. Creates a feature branch and adds a commit
// 3. Finishes with --merge-message "Merge %b into %p" (uses placeholders)
// 4. Verifies the merge commit message has expanded placeholders
func TestFinishWithMergeMessagePlaceholders(t *testing.T) {
	t.Parallel()
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Create and switch to feature branch
	output, err = testutil.RunGitFlow(t, dir, "feature", "start", "placeholder-test")
	if err != nil {
		t.Fatalf("Failed to start feature branch: %v\nOutput: %s", err, output)
	}

	// Add a commit to feature branch
	testutil.WriteFile(t, dir, "feature.txt", "feature content")
	_, err = testutil.RunGit(t, dir, "add", "feature.txt")
	if err != nil {
		t.Fatalf("Failed to add feature file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Add feature file")
	if err != nil {
		t.Fatalf("Failed to commit feature file: %v", err)
	}

	// Switch to develop and add a commit to prevent fast-forward
	_, err = testutil.RunGit(t, dir, "checkout", "develop")
	if err != nil {
		t.Fatalf("Failed to switch to develop: %v", err)
	}
	testutil.WriteFile(t, dir, "develop.txt", "develop content")
	_, err = testutil.RunGit(t, dir, "add", "develop.txt")
	if err != nil {
		t.Fatalf("Failed to add develop file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Add develop file")
	if err != nil {
		t.Fatalf("Failed to commit develop file: %v", err)
	}

	// Finish feature branch with merge message containing placeholders
	output, err = testutil.RunGitFlow(t, dir, "feature", "finish", "placeholder-test", "--merge-message", "feat: merge %b into %p")
	if err != nil {
		t.Fatalf("Failed to finish feature branch: %v\nOutput: %s", err, output)
	}

	// Verify the merge commit message in develop has expanded placeholders
	_, err = testutil.RunGit(t, dir, "checkout", "develop")
	if err != nil {
		t.Fatalf("Failed to checkout develop: %v", err)
	}

	commitMsg, err := testutil.RunGit(t, dir, "log", "-1", "--format=%s")
	if err != nil {
		t.Fatalf("Failed to get commit message: %v", err)
	}

	expectedMsg := "feat: merge feature/placeholder-test into develop"
	if strings.TrimSpace(commitMsg) != expectedMsg {
		t.Errorf("Expected commit message '%s', got '%s'", expectedMsg, strings.TrimSpace(commitMsg))
	}
}

// TestFinishWithUpdateMessagePlaceholders tests that placeholders in --update-message are expanded correctly.
// Steps:
// 1. Sets up a test repository and initializes git-flow with defaults
// 2. Creates a release branch and adds a commit
// 3. Finishes with --update-message "chore: sync %b from %p" (uses placeholders)
// 4. Verifies the develop update commit message has expanded placeholders
func TestFinishWithUpdateMessagePlaceholders(t *testing.T) {
	t.Parallel()
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Create and switch to release branch
	output, err = testutil.RunGitFlow(t, dir, "release", "start", "1.0.0")
	if err != nil {
		t.Fatalf("Failed to start release branch: %v\nOutput: %s", err, output)
	}

	// Add a commit to release branch
	testutil.WriteFile(t, dir, "release.txt", "release content")
	_, err = testutil.RunGit(t, dir, "add", "release.txt")
	if err != nil {
		t.Fatalf("Failed to add release file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Add release file")
	if err != nil {
		t.Fatalf("Failed to commit release file: %v", err)
	}

	// Finish release branch with update message containing placeholders
	// %b = child branch (develop), %p = parent branch (main)
	output, err = testutil.RunGitFlow(t, dir, "release", "finish", "1.0.0", "--update-message", "chore: sync %b from %p")
	if err != nil {
		t.Fatalf("Failed to finish release branch: %v\nOutput: %s", err, output)
	}

	// Verify the update commit message in develop has expanded placeholders
	_, err = testutil.RunGit(t, dir, "checkout", "develop")
	if err != nil {
		t.Fatalf("Failed to checkout develop: %v", err)
	}

	commitMsg, err := testutil.RunGit(t, dir, "log", "-1", "--format=%s")
	if err != nil {
		t.Fatalf("Failed to get commit message: %v", err)
	}

	// For child updates: %b = child branch (develop), %p = source branch (main)
	expectedMsg := "chore: sync develop from main"
	if strings.TrimSpace(commitMsg) != expectedMsg {
		t.Errorf("Expected commit message '%s', got '%s'", expectedMsg, strings.TrimSpace(commitMsg))
	}
}

// TestFinishMessagePlaceholderEscaping tests that %% produces a literal percent sign.
// Steps:
// 1. Sets up a test repository and initializes git-flow with defaults
// 2. Creates a feature branch and adds a commit
// 3. Finishes with --merge-message "100%% complete: %b" (escaped percent)
// 4. Verifies the merge commit message has literal % and expanded %b
func TestFinishMessagePlaceholderEscaping(t *testing.T) {
	t.Parallel()
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Create and switch to feature branch
	output, err = testutil.RunGitFlow(t, dir, "feature", "start", "escape-test")
	if err != nil {
		t.Fatalf("Failed to start feature branch: %v\nOutput: %s", err, output)
	}

	// Add a commit to feature branch
	testutil.WriteFile(t, dir, "feature.txt", "feature content")
	_, err = testutil.RunGit(t, dir, "add", "feature.txt")
	if err != nil {
		t.Fatalf("Failed to add feature file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Add feature file")
	if err != nil {
		t.Fatalf("Failed to commit feature file: %v", err)
	}

	// Switch to develop and add a commit to prevent fast-forward
	_, err = testutil.RunGit(t, dir, "checkout", "develop")
	if err != nil {
		t.Fatalf("Failed to switch to develop: %v", err)
	}
	testutil.WriteFile(t, dir, "develop.txt", "develop content")
	_, err = testutil.RunGit(t, dir, "add", "develop.txt")
	if err != nil {
		t.Fatalf("Failed to add develop file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Add develop file")
	if err != nil {
		t.Fatalf("Failed to commit develop file: %v", err)
	}

	// Finish feature branch with merge message containing escaped percent
	output, err = testutil.RunGitFlow(t, dir, "feature", "finish", "escape-test", "--merge-message", "100%% complete: %b")
	if err != nil {
		t.Fatalf("Failed to finish feature branch: %v\nOutput: %s", err, output)
	}

	// Verify the merge commit message in develop
	_, err = testutil.RunGit(t, dir, "checkout", "develop")
	if err != nil {
		t.Fatalf("Failed to checkout develop: %v", err)
	}

	commitMsg, err := testutil.RunGit(t, dir, "log", "-1", "--format=%s")
	if err != nil {
		t.Fatalf("Failed to get commit message: %v", err)
	}

	// %% should become %, %b should be expanded
	expectedMsg := "100% complete: feature/escape-test"
	if strings.TrimSpace(commitMsg) != expectedMsg {
		t.Errorf("Expected commit message '%s', got '%s'", expectedMsg, strings.TrimSpace(commitMsg))
	}
}

// TestFinishWithMergeMessageFromConfig tests that merge message from git config is used.
// Steps:
// 1. Sets up a test repository and initializes git-flow with defaults
// 2. Configures gitflow.feature.finish.mergeMessage with placeholder "chore: merge %b into %p"
// 3. Creates a feature branch and adds a commit
// 4. Adds a commit to develop to ensure non-fast-forward merge
// 5. Finishes the feature branch without --merge-message flag
// 6. Verifies the merge commit message is "chore: merge feature/config-test into develop"
// 7. Verifies the feature branch is deleted
func TestFinishWithMergeMessageFromConfig(t *testing.T) {
	t.Parallel()
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Configure merge message in git config (Layer 2)
	_, err = testutil.RunGit(t, dir, "config", "gitflow.feature.finish.mergemessage", "chore: merge %b into %p")
	if err != nil {
		t.Fatalf("Failed to configure mergemessage: %v", err)
	}

	// Create feature branch
	output, err = testutil.RunGitFlow(t, dir, "feature", "start", "config-test")
	if err != nil {
		t.Fatalf("Failed to start feature branch: %v\nOutput: %s", err, output)
	}

	// Add a commit to feature branch
	testutil.WriteFile(t, dir, "feature.txt", "feature content")
	_, err = testutil.RunGit(t, dir, "add", "feature.txt")
	if err != nil {
		t.Fatalf("Failed to add feature file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Add feature file")
	if err != nil {
		t.Fatalf("Failed to commit feature file: %v", err)
	}

	// Add a commit to develop to force non-fast-forward merge
	_, err = testutil.RunGit(t, dir, "checkout", "develop")
	if err != nil {
		t.Fatalf("Failed to checkout develop: %v", err)
	}
	testutil.WriteFile(t, dir, "develop.txt", "develop content")
	_, err = testutil.RunGit(t, dir, "add", "develop.txt")
	if err != nil {
		t.Fatalf("Failed to add develop file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Add develop file")
	if err != nil {
		t.Fatalf("Failed to commit develop file: %v", err)
	}

	// Finish feature branch WITHOUT --merge-message flag
	output, err = testutil.RunGitFlow(t, dir, "feature", "finish", "config-test")
	if err != nil {
		t.Fatalf("Failed to finish feature branch: %v\nOutput: %s", err, output)
	}

	// Verify the merge commit message
	commitMsg, err := testutil.RunGit(t, dir, "log", "-1", "--format=%s")
	if err != nil {
		t.Fatalf("Failed to get commit message: %v", err)
	}

	expectedMessage := "chore: merge feature/config-test into develop"
	if strings.TrimSpace(commitMsg) != expectedMessage {
		t.Errorf("Expected commit message '%s', got '%s'", expectedMessage, strings.TrimSpace(commitMsg))
	}

	// Verify feature branch is deleted
	if testutil.BranchExists(t, dir, "feature/config-test") {
		t.Error("Expected feature branch to be deleted")
	}
}

// TestFinishMergeMessageCLIOverridesConfig tests that CLI flag overrides git config.
// Steps:
// 1. Sets up a test repository and initializes git-flow with defaults
// 2. Configures gitflow.feature.finish.mergeMessage to "config: message"
// 3. Creates a feature branch and adds a commit
// 4. Adds a commit to develop to ensure non-fast-forward merge
// 5. Finishes the feature branch WITH --merge-message "cli: override"
// 6. Verifies the merge commit uses "cli: override", not the config message
func TestFinishMergeMessageCLIOverridesConfig(t *testing.T) {
	t.Parallel()
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Configure merge message in git config (Layer 2)
	_, err = testutil.RunGit(t, dir, "config", "gitflow.feature.finish.mergemessage", "config: message")
	if err != nil {
		t.Fatalf("Failed to configure mergemessage: %v", err)
	}

	// Create feature branch
	output, err = testutil.RunGitFlow(t, dir, "feature", "start", "override-test")
	if err != nil {
		t.Fatalf("Failed to start feature branch: %v\nOutput: %s", err, output)
	}

	// Add a commit to feature branch
	testutil.WriteFile(t, dir, "feature.txt", "feature content")
	_, err = testutil.RunGit(t, dir, "add", "feature.txt")
	if err != nil {
		t.Fatalf("Failed to add feature file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Add feature file")
	if err != nil {
		t.Fatalf("Failed to commit feature file: %v", err)
	}

	// Add a commit to develop to force non-fast-forward merge
	_, err = testutil.RunGit(t, dir, "checkout", "develop")
	if err != nil {
		t.Fatalf("Failed to checkout develop: %v", err)
	}
	testutil.WriteFile(t, dir, "develop.txt", "develop content")
	_, err = testutil.RunGit(t, dir, "add", "develop.txt")
	if err != nil {
		t.Fatalf("Failed to add develop file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Add develop file")
	if err != nil {
		t.Fatalf("Failed to commit develop file: %v", err)
	}

	// Finish feature branch WITH --merge-message flag (CLI override)
	cliMessage := "cli: override"
	output, err = testutil.RunGitFlow(t, dir, "feature", "finish", "override-test", "--merge-message", cliMessage)
	if err != nil {
		t.Fatalf("Failed to finish feature branch: %v\nOutput: %s", err, output)
	}

	// Verify the merge commit uses CLI message, not config
	commitMsg, err := testutil.RunGit(t, dir, "log", "-1", "--format=%s")
	if err != nil {
		t.Fatalf("Failed to get commit message: %v", err)
	}

	if strings.TrimSpace(commitMsg) != cliMessage {
		t.Errorf("Expected CLI message '%s' to override config, got '%s'", cliMessage, strings.TrimSpace(commitMsg))
	}
}

// TestFinishWithUpdateMessageFromConfig tests that update message from git config is used.
// Steps:
// 1. Sets up a test repository and initializes git-flow with defaults
// 2. Configures gitflow.release.finish.updateMessage with "chore: sync %b from main"
// 3. Creates a release branch and adds a commit
// 4. Finishes the release branch without --update-message flag
// 5. Verifies the develop update merge commit uses the configured message
func TestFinishWithUpdateMessageFromConfig(t *testing.T) {
	t.Parallel()
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Configure update message in git config (Layer 2)
	_, err = testutil.RunGit(t, dir, "config", "gitflow.release.finish.updatemessage", "chore: sync %b from main")
	if err != nil {
		t.Fatalf("Failed to configure updatemessage: %v", err)
	}

	// Create release branch
	output, err = testutil.RunGitFlow(t, dir, "release", "start", "1.0.0")
	if err != nil {
		t.Fatalf("Failed to start release branch: %v\nOutput: %s", err, output)
	}

	// Add a commit to release branch
	testutil.WriteFile(t, dir, "release.txt", "release content")
	_, err = testutil.RunGit(t, dir, "add", "release.txt")
	if err != nil {
		t.Fatalf("Failed to add release file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Release changes")
	if err != nil {
		t.Fatalf("Failed to commit release file: %v", err)
	}

	// Finish release branch WITHOUT --update-message flag
	output, err = testutil.RunGitFlow(t, dir, "release", "finish", "1.0.0")
	if err != nil {
		t.Fatalf("Failed to finish release branch: %v\nOutput: %s", err, output)
	}

	// Check the develop branch for the update merge commit
	_, err = testutil.RunGit(t, dir, "checkout", "develop")
	if err != nil {
		t.Fatalf("Failed to checkout develop: %v", err)
	}

	// Get the most recent merge commit on develop (should be the update from main)
	commitMsg, err := testutil.RunGit(t, dir, "log", "-1", "--format=%s")
	if err != nil {
		t.Fatalf("Failed to get commit message: %v", err)
	}

	// The update message should have placeholders expanded
	expectedMessage := "chore: sync develop from main"
	if strings.TrimSpace(commitMsg) != expectedMessage {
		t.Errorf("Expected update message '%s', got '%s'", expectedMessage, strings.TrimSpace(commitMsg))
	}
}

// TestFinishUpdateMessageCLIOverridesConfig tests that CLI flag overrides git config for update message.
// Steps:
// 1. Sets up a test repository and initializes git-flow with defaults
// 2. Configures gitflow.release.finish.updateMessage to "config: update"
// 3. Creates a release branch and adds a commit
// 4. Finishes the release branch WITH --update-message "cli: update override"
// 5. Verifies the develop update uses the CLI message, not config
func TestFinishUpdateMessageCLIOverridesConfig(t *testing.T) {
	t.Parallel()
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Configure update message in git config (Layer 2)
	_, err = testutil.RunGit(t, dir, "config", "gitflow.release.finish.updatemessage", "config: update")
	if err != nil {
		t.Fatalf("Failed to configure updatemessage: %v", err)
	}

	// Create release branch
	output, err = testutil.RunGitFlow(t, dir, "release", "start", "1.0.1")
	if err != nil {
		t.Fatalf("Failed to start release branch: %v\nOutput: %s", err, output)
	}

	// Add a commit to release branch
	testutil.WriteFile(t, dir, "release.txt", "release content")
	_, err = testutil.RunGit(t, dir, "add", "release.txt")
	if err != nil {
		t.Fatalf("Failed to add release file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Release changes")
	if err != nil {
		t.Fatalf("Failed to commit release file: %v", err)
	}

	// Finish release branch WITH --update-message flag (CLI override)
	cliUpdateMessage := "cli: update override"
	output, err = testutil.RunGitFlow(t, dir, "release", "finish", "1.0.1", "--update-message", cliUpdateMessage)
	if err != nil {
		t.Fatalf("Failed to finish release branch: %v\nOutput: %s", err, output)
	}

	// Check the develop branch for the update merge commit
	_, err = testutil.RunGit(t, dir, "checkout", "develop")
	if err != nil {
		t.Fatalf("Failed to checkout develop: %v", err)
	}

	// Get the most recent merge commit on develop
	commitMsg, err := testutil.RunGit(t, dir, "log", "-1", "--format=%s")
	if err != nil {
		t.Fatalf("Failed to get commit message: %v", err)
	}

	if strings.TrimSpace(commitMsg) != cliUpdateMessage {
		t.Errorf("Expected CLI update message '%s' to override config, got '%s'", cliUpdateMessage, strings.TrimSpace(commitMsg))
	}
}

// TestMergeStrategyConfigYesEnablesRebase tests that gitflow.feature.finish.rebase=yes
// selects the rebase strategy, matching git-config's truthy "yes" spelling.
// Steps:
// 1. Sets up a test repository and initializes git-flow with defaults
// 2. Sets gitflow.feature.finish.rebase=yes (Layer-1 default is merge)
// 3. Creates a feature branch with a commit
// 4. Adds a commit on develop
// 5. Finishes without any flags
// 6. Verifies the rebase strategy was used and both files exist on develop
func TestMergeStrategyConfigYesEnablesRebase(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	if _, err := testutil.RunGit(t, dir, "config", "gitflow.feature.finish.rebase", "yes"); err != nil {
		t.Fatalf("Failed to set rebase config: %v", err)
	}

	output, err = testutil.RunGitFlow(t, dir, "feature", "start", "rebase-yes-test")
	if err != nil {
		t.Fatalf("Failed to start feature branch: %v\nOutput: %s", err, output)
	}

	testutil.WriteFile(t, dir, "feature.txt", "feature content")
	if _, err := testutil.RunGit(t, dir, "add", "feature.txt"); err != nil {
		t.Fatalf("Failed to add feature file: %v", err)
	}
	if _, err := testutil.RunGit(t, dir, "commit", "-m", "Add feature file"); err != nil {
		t.Fatalf("Failed to commit feature file: %v", err)
	}

	if _, err := testutil.RunGit(t, dir, "checkout", "develop"); err != nil {
		t.Fatalf("Failed to switch to develop: %v", err)
	}
	testutil.WriteFile(t, dir, "develop.txt", "develop content")
	if _, err := testutil.RunGit(t, dir, "add", "develop.txt"); err != nil {
		t.Fatalf("Failed to add develop file: %v", err)
	}
	if _, err := testutil.RunGit(t, dir, "commit", "-m", "Add develop file"); err != nil {
		t.Fatalf("Failed to commit develop file: %v", err)
	}

	output, err = testutil.RunGitFlow(t, dir, "feature", "finish", "rebase-yes-test")
	if err != nil {
		t.Fatalf("Failed to finish feature branch: %v\nOutput: %s", err, output)
	}

	if !strings.Contains(output, "Merging using strategy: rebase") {
		t.Errorf("Expected rebase strategy from rebase=yes config, got: %s", output)
	}

	if _, err := testutil.RunGit(t, dir, "checkout", "develop"); err != nil {
		t.Fatalf("Failed to checkout develop: %v", err)
	}
	if !testutil.FileExists(t, dir, "feature.txt") {
		t.Error("Expected feature.txt to exist in develop branch")
	}
	if !testutil.FileExists(t, dir, "develop.txt") {
		t.Error("Expected develop.txt to exist in develop branch")
	}
}

// TestMergeStrategyFalsyPositiveKeyDoesNotDisableRebase tests that a falsy value on
// the positive rebase key does not flip a Layer-1 rebase strategy back to merge.
// The positive key is one-directional by design; no-rebase is the way to disable it.
// Steps:
// 1. Sets up a test repository and initializes git-flow with defaults
// 2. Sets gitflow.branch.feature.upstreamstrategy=rebase (Layer 1)
// 3. Sets gitflow.feature.finish.rebase=off (Layer 2, falsy positive key)
// 4. Creates a feature branch with a commit and adds a commit on develop
// 5. Finishes without any flags
// 6. Verifies the rebase strategy is still used
func TestMergeStrategyFalsyPositiveKeyDoesNotDisableRebase(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	if _, err := testutil.RunGit(t, dir, "config", "gitflow.branch.feature.upstreamstrategy", "rebase"); err != nil {
		t.Fatalf("Failed to set upstream strategy config: %v", err)
	}
	if _, err := testutil.RunGit(t, dir, "config", "gitflow.feature.finish.rebase", "off"); err != nil {
		t.Fatalf("Failed to set rebase config: %v", err)
	}

	output, err = testutil.RunGitFlow(t, dir, "feature", "start", "falsy-positive-test")
	if err != nil {
		t.Fatalf("Failed to start feature branch: %v\nOutput: %s", err, output)
	}

	testutil.WriteFile(t, dir, "feature.txt", "feature content")
	if _, err := testutil.RunGit(t, dir, "add", "feature.txt"); err != nil {
		t.Fatalf("Failed to add feature file: %v", err)
	}
	if _, err := testutil.RunGit(t, dir, "commit", "-m", "Add feature file"); err != nil {
		t.Fatalf("Failed to commit feature file: %v", err)
	}

	if _, err := testutil.RunGit(t, dir, "checkout", "develop"); err != nil {
		t.Fatalf("Failed to switch to develop: %v", err)
	}
	testutil.WriteFile(t, dir, "develop.txt", "develop content")
	if _, err := testutil.RunGit(t, dir, "add", "develop.txt"); err != nil {
		t.Fatalf("Failed to add develop file: %v", err)
	}
	if _, err := testutil.RunGit(t, dir, "commit", "-m", "Add develop file"); err != nil {
		t.Fatalf("Failed to commit develop file: %v", err)
	}

	output, err = testutil.RunGitFlow(t, dir, "feature", "finish", "falsy-positive-test")
	if err != nil {
		t.Fatalf("Failed to finish feature branch: %v\nOutput: %s", err, output)
	}

	if !strings.Contains(output, "Merging using strategy: rebase") {
		t.Errorf("Expected Layer-1 rebase to survive a falsy positive key, got: %s", output)
	}
}
