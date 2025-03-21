package cmd_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gittower/git-flow-next/test/testutil"
)

// Test functions

// TestFinishFeatureBranch tests the finish command for feature branches
func TestFinishFeatureBranch(t *testing.T) {
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults and create branches
	output, err := testutil.RunGitFlow(t, dir, "init", "-d", "-c")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Create a feature branch
	output, err = testutil.RunGitFlow(t, dir, "feature", "start", "my-feature")
	if err != nil {
		t.Fatalf("Failed to create feature branch: %v\nOutput: %s", err, output)
	}

	// Create a test file
	testutil.WriteFile(t, dir, "test.txt", "test content")

	// Commit the changes
	_, err = testutil.RunGit(t, dir, "add", "test.txt")
	if err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Add test file")
	if err != nil {
		t.Fatalf("Failed to commit file: %v", err)
	}

	// Finish the feature branch
	output, err = testutil.RunGitFlow(t, dir, "feature", "finish", "my-feature")
	if err != nil {
		t.Fatalf("Failed to finish feature branch: %v\nOutput: %s", err, output)
	}

	// Verify that feature branch is deleted
	if testutil.BranchExists(t, dir, "feature/my-feature") {
		t.Error("Expected feature branch to be deleted")
	}

	// Verify that changes are merged into develop
	_, err = testutil.RunGit(t, dir, "checkout", "develop")
	if err != nil {
		t.Fatalf("Failed to checkout develop: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, "test.txt")); os.IsNotExist(err) {
		t.Error("Expected test.txt to exist in develop branch")
	}

	// Verify that no tag was created (feature branches don't create tags)
	output, err = testutil.RunGit(t, dir, "tag", "-l")
	if err != nil {
		t.Fatalf("Failed to list tags: %v", err)
	}
	if output != "" {
		t.Error("Expected no tags to be created for feature branches")
	}
}

// TestFinishReleaseBranch tests the finish command for release branches
func TestFinishReleaseBranch(t *testing.T) {
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults and create branches
	output, err := testutil.RunGitFlow(t, dir, "init", "-d", "-c")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Set tag prefix for release branches
	_, err = testutil.RunGit(t, dir, "config", "gitflow.branch.release.tagprefix", "v")
	if err != nil {
		t.Fatalf("Failed to set tag prefix: %v", err)
	}

	// Test release branch
	output, err = testutil.RunGitFlow(t, dir, "release", "start", "1.0.0")
	if err != nil {
		t.Fatalf("Failed to create release branch: %v\nOutput: %s", err, output)
	}

	// Ensure we're on the release branch
	_, err = testutil.RunGit(t, dir, "checkout", "release/1.0.0")
	if err != nil {
		t.Fatalf("Failed to checkout release branch: %v", err)
	}

	// Create a test file
	testutil.WriteFile(t, dir, "release.txt", "release content")

	// Commit the changes
	_, err = testutil.RunGit(t, dir, "add", "release.txt")
	if err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Add release file")
	if err != nil {
		t.Fatalf("Failed to commit file: %v", err)
	}

	// Finish the release branch
	output, err = testutil.RunGitFlow(t, dir, "release", "finish", "1.0.0")
	if err != nil {
		t.Fatalf("Failed to finish release branch: %v\nOutput: %s", err, output)
	}

	// Verify that release branch is deleted
	if testutil.BranchExists(t, dir, "release/1.0.0") {
		t.Error("Expected release branch to be deleted")
	}

	// Verify changes are in main branch
	_, err = testutil.RunGit(t, dir, "checkout", "main")
	if err != nil {
		t.Fatalf("Failed to checkout main: %v", err)
	}

	content, err := testutil.RunGit(t, dir, "--no-pager", "show", "HEAD:release.txt")
	if err != nil {
		t.Fatalf("Failed to read file content from main: %v", err)
	}
	if content != "release content" {
		t.Errorf("Expected release.txt content in main to be 'release content', got '%s'", content)
	}

	// Verify changes are in develop branch (as it's a child base branch of main)
	_, err = testutil.RunGit(t, dir, "checkout", "develop")
	if err != nil {
		t.Fatalf("Failed to checkout develop: %v", err)
	}

	content, err = testutil.RunGit(t, dir, "--no-pager", "show", "HEAD:release.txt")
	if err != nil {
		t.Fatalf("Failed to read file content from develop: %v", err)
	}
	if content != "release content" {
		t.Errorf("Expected release.txt content in develop to be 'release content', got '%s'", content)
	}

	// Verify no merge state is left
	if testutil.IsMergeInProgress(t, dir) {
		t.Error("Expected no merge in progress after successful finish")
	}

	// Verify that a tag was created
	output, err = testutil.RunGit(t, dir, "tag", "-l")
	if err != nil {
		t.Fatalf("Failed to list tags: %v", err)
	}
	if !strings.Contains(output, "v1.0.0") {
		t.Error("Expected tag 'v1.0.0' to be created")
	}
}

// TestFinishHotfixBranch tests the finish command for hotfix branches
func TestFinishHotfixBranch(t *testing.T) {
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults and create branches
	output, err := testutil.RunGitFlow(t, dir, "init", "-d", "-c")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Set tag prefix for hotfix branches
	_, err = testutil.RunGit(t, dir, "config", "gitflow.branch.hotfix.tagprefix", "v")
	if err != nil {
		t.Fatalf("Failed to set tag prefix: %v", err)
	}

	// Test hotfix branch
	output, err = testutil.RunGitFlow(t, dir, "hotfix", "start", "1.0.1")
	if err != nil {
		t.Fatalf("Failed to create hotfix branch: %v\nOutput: %s", err, output)
	}

	// Create a test file
	testutil.WriteFile(t, dir, "hotfix.txt", "hotfix content")

	// Commit the changes
	_, err = testutil.RunGit(t, dir, "add", "hotfix.txt")
	if err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Add hotfix file")
	if err != nil {
		t.Fatalf("Failed to commit file: %v", err)
	}

	// Finish the hotfix branch
	output, err = testutil.RunGitFlow(t, dir, "hotfix", "finish", "1.0.1")
	if err != nil {
		t.Fatalf("Failed to finish hotfix branch: %v\nOutput: %s", err, output)
	}

	// Verify that hotfix branch is deleted
	if testutil.BranchExists(t, dir, "hotfix/1.0.1") {
		t.Error("Expected hotfix branch to be deleted")
	}

	// Verify changes are in main branch
	_, err = testutil.RunGit(t, dir, "checkout", "main")
	if err != nil {
		t.Fatalf("Failed to checkout main: %v", err)
	}

	content, err := testutil.RunGit(t, dir, "--no-pager", "show", "HEAD:hotfix.txt")
	if err != nil {
		t.Fatalf("Failed to read file content from main: %v", err)
	}
	if content != "hotfix content" {
		t.Errorf("Expected hotfix.txt content in main to be 'hotfix content', got '%s'", content)
	}

	// Verify changes are in develop branch
	_, err = testutil.RunGit(t, dir, "checkout", "develop")
	if err != nil {
		t.Fatalf("Failed to checkout develop: %v", err)
	}

	content, err = testutil.RunGit(t, dir, "--no-pager", "show", "HEAD:hotfix.txt")
	if err != nil {
		t.Fatalf("Failed to read file content from develop: %v", err)
	}
	if content != "hotfix content" {
		t.Errorf("Expected hotfix.txt content in develop to be 'hotfix content', got '%s'", content)
	}

	// Verify no merge state is left
	if testutil.IsMergeInProgress(t, dir) {
		t.Error("Expected no merge in progress after successful finish")
	}

	// Verify that a tag was created
	output, err = testutil.RunGit(t, dir, "tag", "-l")
	if err != nil {
		t.Fatalf("Failed to list tags: %v", err)
	}
	if !strings.Contains(output, "v1.0.1") {
		t.Error("Expected tag 'v1.0.1' to be created")
	}
}

// TestFinishWithCustomConfig tests the finish command with custom configuration
func TestFinishWithCustomConfig(t *testing.T) {
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with custom configuration and create branches
	input := "custom-main\ncustom-dev\nf/\nr/\nh/\ns/\ny\n"
	output, err := testutil.RunGitFlowWithInput(t, dir, input, "init", "-c")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Create a feature branch
	output, err = testutil.RunGitFlow(t, dir, "feature", "start", "custom-feature")
	if err != nil {
		t.Fatalf("Failed to create feature branch: %v\nOutput: %s", err, output)
	}

	// Create a test file
	testutil.WriteFile(t, dir, "test.txt", "test content")

	// Commit the changes
	_, err = testutil.RunGit(t, dir, "add", "test.txt")
	if err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Add test file")
	if err != nil {
		t.Fatalf("Failed to commit file: %v", err)
	}

	// Finish the feature branch
	output, err = testutil.RunGitFlow(t, dir, "feature", "finish", "custom-feature")
	if err != nil {
		t.Fatalf("Failed to finish feature branch: %v\nOutput: %s", err, output)
	}

	// Verify that feature branch is deleted
	if testutil.BranchExists(t, dir, "f/custom-feature") {
		t.Error("Expected feature branch to be deleted")
	}

	// Verify that changes are merged into custom-dev
	_, err = testutil.RunGit(t, dir, "checkout", "custom-dev")
	if err != nil {
		t.Fatalf("Failed to checkout custom-dev: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, "test.txt")); os.IsNotExist(err) {
		t.Error("Expected test.txt to exist in custom-dev branch")
	}
}

// TestFinishNonExistentBranch tests the finish command with a non-existent branch
func TestFinishNonExistentBranch(t *testing.T) {
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults and create branches
	output, err := testutil.RunGitFlow(t, dir, "init", "-d", "-c")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Try to finish a non-existent feature branch
	output, err = testutil.RunGitFlow(t, dir, "feature", "finish", "non-existent")
	if err == nil {
		t.Fatal("Expected error when finishing non-existent branch")
	}

	// Check if the error message is appropriate
	if !strings.Contains(output, "does not exist") {
		t.Errorf("Expected error message to contain 'does not exist', got: %s", output)
	}
}

// TestFinishWithMergeConflict tests that the finish command properly handles merge conflicts
func TestFinishWithMergeConflict(t *testing.T) {
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults and create branches
	output, err := testutil.RunGitFlow(t, dir, "init", "-d", "-c")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Set merge strategy to merge for feature branches
	_, err = testutil.RunGit(t, dir, "config", "gitflow.branch.feature.upstreamstrategy", "merge")
	if err != nil {
		t.Fatalf("Failed to set merge strategy: %v", err)
	}

	// Create and switch to feature branch
	output, err = testutil.RunGitFlow(t, dir, "feature", "start", "conflict-test")
	if err != nil {
		t.Fatalf("Failed to create feature branch: %v\nOutput: %s", err, output)
	}

	// Create file in feature branch
	testutil.WriteFile(t, dir, "test.txt", "feature content")
	_, err = testutil.RunGit(t, dir, "add", "test.txt")
	if err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Add test.txt in feature")
	if err != nil {
		t.Fatalf("Failed to commit file: %v", err)
	}

	// Switch to develop and create the same file with different content
	_, err = testutil.RunGit(t, dir, "checkout", "develop")
	if err != nil {
		t.Fatalf("Failed to checkout develop: %v", err)
	}

	testutil.WriteFile(t, dir, "test.txt", "develop content")
	_, err = testutil.RunGit(t, dir, "add", "test.txt")
	if err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Add test.txt in develop")
	if err != nil {
		t.Fatalf("Failed to commit file: %v", err)
	}

	// Try to finish the feature branch (should fail due to conflict)
	output, err = testutil.RunGitFlow(t, dir, "feature", "finish", "conflict-test")
	if err == nil {
		t.Fatal("Expected finish to fail due to merge conflict")
	}

	// Verify merge state
	state, err := testutil.LoadMergeState(t, dir)
	if err != nil {
		t.Fatalf("Failed to load merge state: %v", err)
	}

	if state.Action != "finish" {
		t.Errorf("Expected action to be 'finish', got '%s'", state.Action)
	}
	if state.BranchType != "feature" {
		t.Errorf("Expected branchType to be 'feature', got '%s'", state.BranchType)
	}
	if state.BranchName != "conflict-test" {
		t.Errorf("Expected branchName to be 'conflict-test', got '%s'", state.BranchName)
	}
	if state.CurrentStep != "merge" {
		t.Errorf("Expected currentStep to be 'merge', got '%s'", state.CurrentStep)
	}
	if state.ParentBranch != "develop" {
		t.Errorf("Expected parentBranch to be 'develop', got '%s'", state.ParentBranch)
	}
	if state.MergeStrategy != "merge" {
		t.Errorf("Expected mergeStrategy to be 'merge', got '%s'", state.MergeStrategy)
	}
	if state.FullBranchName != "feature/conflict-test" {
		t.Errorf("Expected fullBranchName to be 'feature/conflict-test', got '%s'", state.FullBranchName)
	}
}

func TestFinishWithMergeAbort(t *testing.T) {
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults and create branches
	output, err := testutil.RunGitFlow(t, dir, "init", "-d", "-c")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Set merge strategy to merge for feature branches
	_, err = testutil.RunGit(t, dir, "config", "gitflow.branch.feature.upstreamstrategy", "merge")
	if err != nil {
		t.Fatalf("Failed to set merge strategy: %v", err)
	}

	// Create and switch to feature branch
	output, err = testutil.RunGitFlow(t, dir, "feature", "start", "abort-feature")
	if err != nil {
		t.Fatalf("Failed to create feature branch: %v\nOutput: %s", err, output)
	}

	// Create file in feature branch
	testutil.WriteFile(t, dir, "test.txt", "feature content")
	_, err = testutil.RunGit(t, dir, "add", "test.txt")
	if err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Add test.txt in feature")
	if err != nil {
		t.Fatalf("Failed to commit file: %v", err)
	}

	// Switch to develop and create the same file with different content
	_, err = testutil.RunGit(t, dir, "checkout", "develop")
	if err != nil {
		t.Fatalf("Failed to checkout develop: %v", err)
	}

	testutil.WriteFile(t, dir, "test.txt", "develop content")
	_, err = testutil.RunGit(t, dir, "add", "test.txt")
	if err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Add test.txt in develop")
	if err != nil {
		t.Fatalf("Failed to commit file: %v", err)
	}

	// Try to finish the feature branch (should fail due to conflict)
	output, err = testutil.RunGitFlow(t, dir, "feature", "finish", "abort-feature")
	if err == nil {
		t.Fatal("Expected finish to fail due to merge conflict")
	}

	// Verify we're in a merge conflict state
	if !testutil.IsMergeInProgress(t, dir) {
		t.Error("Expected to be in merge conflict state")
	}

	// Abort the merge
	output, err = testutil.RunGitFlow(t, dir, "feature", "finish", "--abort", "abort-feature")
	if err != nil {
		t.Fatalf("Failed to abort merge: %v\nOutput: %s", err, output)
	}

	// Verify we're back on the feature branch
	currentBranch := testutil.GetCurrentBranch(t, dir)
	if !strings.Contains(currentBranch, "abort-feature") {
		t.Errorf("Expected to be back on feature branch after abort, got %s", currentBranch)
	}

	// Verify the merge was aborted (no merge in progress)
	if testutil.IsMergeInProgress(t, dir) {
		t.Error("Expected no merge in progress after abort")
	}

	// Verify the file content is back to the feature branch version
	content := testutil.ReadFile(t, dir, "test.txt")
	if content != "feature content" {
		t.Errorf("Expected file content to be 'feature content', got '%s'", content)
	}
}

func TestFinishWithRebaseConflict(t *testing.T) {
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults and create branches
	output, err := testutil.RunGitFlow(t, dir, "init", "-d", "-c")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Create a file in develop
	testutil.WriteFile(t, dir, "test.txt", "develop content")

	// Commit the file in develop
	_, err = testutil.RunGit(t, dir, "add", "test.txt")
	if err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Add test.txt in develop")
	if err != nil {
		t.Fatalf("Failed to commit file: %v", err)
	}

	// Create and switch to feature branch
	output, err = testutil.RunGitFlow(t, dir, "feature", "start", "rebase-feature")
	if err != nil {
		t.Fatalf("Failed to create feature branch: %v\nOutput: %s", err, output)
	}

	// Modify the same file in feature branch
	testutil.WriteFile(t, dir, "test.txt", "feature content")

	// Commit the changes in feature branch
	_, err = testutil.RunGit(t, dir, "add", "test.txt")
	if err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Modify test.txt in feature")
	if err != nil {
		t.Fatalf("Failed to commit file: %v", err)
	}

	// Try to finish the feature branch with rebase
	output, err = testutil.RunGitFlow(t, dir, "feature", "finish", "--rebase", "rebase-feature")
	if err == nil {
		t.Fatal("Expected finish to fail due to rebase conflict")
	}

	// Verify that we're in a rebase conflict state
	currentBranch := testutil.GetCurrentBranch(t, dir)
	if !strings.Contains(currentBranch, "rebase-feature") {
		t.Errorf("Expected to be on feature branch during rebase conflict, got %s", currentBranch)
	}
}

// TestFinishWithMergeContinue tests the continue functionality after resolving a merge conflict
func TestFinishWithMergeContinue(t *testing.T) {
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults and create branches
	output, err := testutil.RunGitFlow(t, dir, "init", "-d", "-c")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Set merge strategy to merge for feature branches
	_, err = testutil.RunGit(t, dir, "config", "gitflow.branch.feature.upstreamstrategy", "merge")
	if err != nil {
		t.Fatalf("Failed to set merge strategy: %v", err)
	}

	// Create and switch to feature branch
	output, err = testutil.RunGitFlow(t, dir, "feature", "start", "continue-test")
	if err != nil {
		t.Fatalf("Failed to create feature branch: %v\nOutput: %s", err, output)
	}

	// Create file in feature branch
	testutil.WriteFile(t, dir, "test.txt", "feature content")
	_, err = testutil.RunGit(t, dir, "add", "test.txt")
	if err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Add test.txt in feature")
	if err != nil {
		t.Fatalf("Failed to commit file: %v", err)
	}

	// Switch to develop and create the same file with different content
	_, err = testutil.RunGit(t, dir, "checkout", "develop")
	if err != nil {
		t.Fatalf("Failed to checkout develop: %v", err)
	}

	testutil.WriteFile(t, dir, "test.txt", "develop content")
	_, err = testutil.RunGit(t, dir, "add", "test.txt")
	if err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Add test.txt in develop")
	if err != nil {
		t.Fatalf("Failed to commit file: %v", err)
	}

	// Try to finish the feature branch (should fail due to conflict)
	output, err = testutil.RunGitFlow(t, dir, "feature", "finish", "continue-test")
	if err == nil {
		t.Fatal("Expected finish to fail due to merge conflict")
	}

	// Verify we're in a merge conflict state
	if !testutil.IsMergeInProgress(t, dir) {
		t.Error("Expected to be in merge conflict state")
	}

	// Resolve the conflict by choosing the feature branch version
	testutil.WriteFile(t, dir, "test.txt", "feature content")
	_, err = testutil.RunGit(t, dir, "add", "test.txt")
	if err != nil {
		t.Fatalf("Failed to add resolved file: %v", err)
	}

	// Commit the merge resolution
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Merge resolved")
	if err != nil {
		t.Fatalf("Failed to commit merge resolution: %v", err)
	}

	// Continue the finish operation
	output, err = testutil.RunGitFlow(t, dir, "feature", "finish", "--continue", "continue-test")
	if err != nil {
		t.Fatalf("Failed to continue finish operation: %v\nOutput: %s", err, output)
	}

	// Verify we're no longer in a merge state
	if testutil.IsMergeInProgress(t, dir) {
		t.Error("Expected no merge in progress after continue")
	}

	// Verify we're on the develop branch
	currentBranch := testutil.GetCurrentBranch(t, dir)
	if !strings.Contains(currentBranch, "develop") {
		t.Errorf("Expected to be on develop branch after continue, got %s", currentBranch)
	}

	// Verify the feature branch was deleted
	if testutil.BranchExists(t, dir, "feature/continue-test") {
		t.Error("Expected feature branch to be deleted after successful finish")
	}

	// Verify the file content matches our resolution
	content := testutil.ReadFile(t, dir, "test.txt")
	if content != "feature content" {
		t.Errorf("Expected file content to be 'feature content', got '%s'", content)
	}
}

// TestFinishWithChildBranchConflict tests that conflicts in child base branches are handled properly
func TestFinishWithChildBranchConflict(t *testing.T) {
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	output, err := testutil.RunGitFlow(t, dir, "init", "-d", "-c")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Create a release branch
	output, err = testutil.RunGitFlow(t, dir, "release", "start", "1.0.0")
	if err != nil {
		t.Fatalf("Failed to create release branch: %v\nOutput: %s", err, output)
	}

	// Add a file in release branch
	testutil.WriteFile(t, dir, "version.txt", "1.0.0")
	_, err = testutil.RunGit(t, dir, "add", "version.txt")
	if err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Add version file")
	if err != nil {
		t.Fatalf("Failed to commit file: %v", err)
	}

	// Switch to develop and create conflicting change
	_, err = testutil.RunGit(t, dir, "checkout", "develop")
	if err != nil {
		t.Fatalf("Failed to checkout develop: %v", err)
	}

	testutil.WriteFile(t, dir, "version.txt", "dev-version")
	_, err = testutil.RunGit(t, dir, "add", "version.txt")
	if err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Add dev version")
	if err != nil {
		t.Fatalf("Failed to commit file: %v", err)
	}

	// Try to finish the release branch (should succeed for main but fail for develop)
	output, err = testutil.RunGitFlow(t, dir, "release", "finish", "1.0.0")
	if err == nil {
		t.Fatal("Expected finish to fail due to conflict in develop branch")
	}

	// Verify merge state
	state, err := testutil.LoadMergeState(t, dir)
	if err != nil {
		t.Fatalf("Failed to load merge state: %v", err)
	}

	if state.Action != "finish" {
		t.Errorf("Expected action to be 'finish', got '%s'", state.Action)
	}
	if state.BranchType != "release" {
		t.Errorf("Expected branchType to be 'release', got '%s'", state.BranchType)
	}
	if state.BranchName != "1.0.0" {
		t.Errorf("Expected branchName to be '1.0.0', got '%s'", state.BranchName)
	}
	if state.CurrentStep != "update_children" {
		t.Errorf("Expected currentStep to be 'update_children', got '%s'", state.CurrentStep)
	}
	if state.ParentBranch != "main" {
		t.Errorf("Expected parentBranch to be 'main', got '%s'", state.ParentBranch)
	}
	if len(state.ChildBranches) != 1 || state.ChildBranches[0] != "develop" {
		t.Errorf("Expected ChildBranches to contain ['develop'], got %v", state.ChildBranches)
	}
	if len(state.UpdatedBranches) != 0 {
		t.Errorf("Expected UpdatedBranches to be empty, got %v", state.UpdatedBranches)
	}

	// Verify we're on develop branch with conflict
	currentBranch := testutil.GetCurrentBranch(t, dir)
	if !strings.Contains(currentBranch, "develop") {
		t.Errorf("Expected to be on develop branch, got %s", currentBranch)
	}

	// Verify the file contents from both branches during merge
	content, err := testutil.RunGit(t, dir, "--no-pager", "show", ":2:version.txt")
	if err != nil {
		t.Fatalf("Failed to read develop version of file: %v", err)
	}
	if content != "dev-version" {
		t.Errorf("Expected version.txt content in develop to be 'dev-version', got '%s'", content)
	}

	content, err = testutil.RunGit(t, dir, "--no-pager", "show", ":3:version.txt")
	if err != nil {
		t.Fatalf("Failed to read release version of file: %v", err)
	}
	if content != "1.0.0" {
		t.Errorf("Expected version.txt content in release to be '1.0.0', got '%s'", content)
	}

	// Resolve the conflict by taking the release version
	testutil.WriteFile(t, dir, "version.txt", "1.0.0")
	_, err = testutil.RunGit(t, dir, "add", "version.txt")
	if err != nil {
		t.Fatalf("Failed to add resolved file: %v", err)
	}

	// Commit the merge resolution
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Merge resolved")
	if err != nil {
		t.Fatalf("Failed to commit merge resolution: %v", err)
	}

	// Continue the finish operation
	output, err = testutil.RunGitFlow(t, dir, "release", "finish", "--continue", "1.0.0")
	if err != nil {
		t.Fatalf("Failed to continue finish operation: %v\nOutput: %s", err, output)
	}

	// Verify final state
	if testutil.IsMergeInProgress(t, dir) {
		t.Error("Expected no merge in progress after continue")
	}

	// Verify release branch was deleted
	if testutil.BranchExists(t, dir, "release/1.0.0") {
		t.Error("Expected release branch to be deleted")
	}

	// Verify content in both main and develop
	_, err = testutil.RunGit(t, dir, "checkout", "main")
	if err != nil {
		t.Fatalf("Failed to checkout main: %v", err)
	}
	content, err = testutil.RunGit(t, dir, "--no-pager", "show", "HEAD:version.txt")
	if err != nil {
		t.Fatalf("Failed to read file content from main: %v", err)
	}
	if content != "1.0.0" {
		t.Errorf("Expected version.txt content in main to be '1.0.0', got '%s'", content)
	}

	_, err = testutil.RunGit(t, dir, "checkout", "develop")
	if err != nil {
		t.Fatalf("Failed to checkout develop: %v", err)
	}
	content, err = testutil.RunGit(t, dir, "--no-pager", "show", "HEAD:version.txt")
	if err != nil {
		t.Fatalf("Failed to read file content from develop: %v", err)
	}
	if content != "1.0.0" {
		t.Errorf("Expected version.txt content in develop to be '1.0.0', got '%s'", content)
	}
}

// TestFinishReleaseWithMergeContinue tests the continue functionality after resolving a merge conflict for release branches
func TestFinishReleaseWithMergeContinue(t *testing.T) {
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults and create branches
	output, err := testutil.RunGitFlow(t, dir, "init", "-d", "-c")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Set tag prefix for release branches
	_, err = testutil.RunGit(t, dir, "config", "gitflow.branch.release.tagprefix", "v")
	if err != nil {
		t.Fatalf("Failed to set tag prefix: %v", err)
	}

	// Create and switch to release branch
	output, err = testutil.RunGitFlow(t, dir, "release", "start", "1.0.0")
	if err != nil {
		t.Fatalf("Failed to create release branch: %v\nOutput: %s", err, output)
	}

	// Create file in release branch
	testutil.WriteFile(t, dir, "version.txt", "1.0.0")
	_, err = testutil.RunGit(t, dir, "add", "version.txt")
	if err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Add version file")
	if err != nil {
		t.Fatalf("Failed to commit file: %v", err)
	}

	// Switch to main and create the same file with different content
	_, err = testutil.RunGit(t, dir, "checkout", "main")
	if err != nil {
		t.Fatalf("Failed to checkout main: %v", err)
	}

	testutil.WriteFile(t, dir, "version.txt", "main content")
	_, err = testutil.RunGit(t, dir, "add", "version.txt")
	if err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Add version file in main")
	if err != nil {
		t.Fatalf("Failed to commit file: %v", err)
	}

	// Try to finish the release branch (should fail due to conflict)
	output, err = testutil.RunGitFlow(t, dir, "release", "finish", "1.0.0")
	if err == nil {
		t.Fatal("Expected finish to fail due to merge conflict")
	}

	// Verify we're in a merge conflict state
	if !testutil.IsMergeInProgress(t, dir) {
		t.Error("Expected to be in merge conflict state")
	}

	// Resolve the conflict by choosing the release branch version
	testutil.WriteFile(t, dir, "version.txt", "1.0.0")
	_, err = testutil.RunGit(t, dir, "add", "version.txt")
	if err != nil {
		t.Fatalf("Failed to add resolved file: %v", err)
	}

	// Commit the merge resolution
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Merge resolved")
	if err != nil {
		t.Fatalf("Failed to commit merge resolution: %v", err)
	}

	// Continue the finish operation
	output, err = testutil.RunGitFlow(t, dir, "release", "finish", "--continue", "1.0.0")
	if err != nil {
		t.Fatalf("Failed to continue finish operation: %v\nOutput: %s", err, output)
	}

	// Verify we're no longer in a merge state
	if testutil.IsMergeInProgress(t, dir) {
		t.Error("Expected no merge in progress after continue")
	}

	// Verify we're on the main branch
	currentBranch := testutil.GetCurrentBranch(t, dir)
	if !strings.Contains(currentBranch, "main") {
		t.Errorf("Expected to be on main branch after continue, got %s", currentBranch)
	}

	// Verify the release branch was deleted
	if testutil.BranchExists(t, dir, "release/1.0.0") {
		t.Error("Expected release branch to be deleted after successful finish")
	}

	// Verify the file content matches our resolution
	content := testutil.ReadFile(t, dir, "version.txt")
	if content != "1.0.0" {
		t.Errorf("Expected file content to be '1.0.0', got '%s'", content)
	}

	// Verify that a tag was created
	output, err = testutil.RunGit(t, dir, "tag", "-l")
	if err != nil {
		t.Fatalf("Failed to list tags: %v", err)
	}
	if !strings.Contains(output, "v1.0.0") {
		t.Error("Expected tag 'v1.0.0' to be created")
	}
}

// TestFinishNonStandardBranchWithForce tests finishing a non-standard branch with force flag
func TestFinishNonStandardBranchWithForce(t *testing.T) {
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	output, err := testutil.RunGitFlow(t, dir, "init", "-d", "-c")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Create a non-standard branch from develop
	_, err = testutil.RunGit(t, dir, "checkout", "develop")
	if err != nil {
		t.Fatalf("Failed to checkout develop: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "checkout", "-b", "custom/my-branch")
	if err != nil {
		t.Fatalf("Failed to create custom branch: %v", err)
	}

	// Add some changes
	testutil.WriteFile(t, dir, "test.txt", "test content")
	_, err = testutil.RunGit(t, dir, "add", "test.txt")
	if err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Add test file")
	if err != nil {
		t.Fatalf("Failed to commit file: %v", err)
	}

	// Finish the branch using feature strategy with force flag
	output, err = testutil.RunGitFlow(t, dir, "feature", "finish", "-f", "custom/my-branch")
	if err != nil {
		t.Fatalf("Failed to finish custom branch: %v\nOutput: %s", err, output)
	}

	// Verify branch was merged to develop
	_, err = testutil.RunGit(t, dir, "checkout", "develop")
	if err != nil {
		t.Fatalf("Failed to checkout develop: %v", err)
	}

	// Check if test.txt exists in develop
	if !testutil.FileExists(t, dir, "test.txt") {
		t.Error("Expected test.txt to exist in develop branch")
	}

	// Verify custom branch was deleted
	if testutil.BranchExists(t, dir, "custom/my-branch") {
		t.Error("Expected custom branch to be deleted")
	}
}

// TestFinishNonStandardBranchWithoutForce tests finishing a non-standard branch without force flag
func TestFinishNonStandardBranchWithoutForce(t *testing.T) {
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	output, err := testutil.RunGitFlow(t, dir, "init", "-d", "-c")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Create a non-standard branch from develop
	_, err = testutil.RunGit(t, dir, "checkout", "develop")
	if err != nil {
		t.Fatalf("Failed to checkout develop: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "checkout", "-b", "custom/my-branch")
	if err != nil {
		t.Fatalf("Failed to create custom branch: %v", err)
	}

	// Add some changes
	testutil.WriteFile(t, dir, "test.txt", "test content")
	_, err = testutil.RunGit(t, dir, "add", "test.txt")
	if err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Add test file")
	if err != nil {
		t.Fatalf("Failed to commit file: %v", err)
	}

	// Try to finish the branch without force flag (should fail)
	output, err = testutil.RunGitFlow(t, dir, "feature", "finish", "custom/my-branch")
	if err == nil {
		t.Fatal("Expected finish to fail without force flag and user confirmation")
	}

	// Verify branch still exists
	if !testutil.BranchExists(t, dir, "custom/my-branch") {
		t.Error("Expected custom branch to still exist")
	}

	// Verify we're still on the custom branch
	currentBranch := testutil.GetCurrentBranch(t, dir)
	if currentBranch != "custom/my-branch" {
		t.Errorf("Expected to be on custom/my-branch, got %s", currentBranch)
	}
}

// TestFinishNonStandardBranchWithTag tests finishing a non-standard branch with tag creation
func TestFinishNonStandardBranchWithTag(t *testing.T) {
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults and tag configuration
	output, err := testutil.RunGitFlow(t, dir, "init", "-d", "-c")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Create a non-standard branch from develop
	_, err = testutil.RunGit(t, dir, "checkout", "develop")
	if err != nil {
		t.Fatalf("Failed to checkout develop: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "checkout", "-b", "custom/my-release")
	if err != nil {
		t.Fatalf("Failed to create custom branch: %v", err)
	}

	// Add some changes
	testutil.WriteFile(t, dir, "release.txt", "release content")
	_, err = testutil.RunGit(t, dir, "add", "release.txt")
	if err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Add release file")
	if err != nil {
		t.Fatalf("Failed to commit file: %v", err)
	}

	// Finish the branch using release strategy with force flag
	output, err = testutil.RunGitFlow(t, dir, "release", "finish", "-f", "custom/my-release")
	if err != nil {
		t.Fatalf("Failed to finish custom release branch: %v\nOutput: %s", err, output)
	}

	// Verify tag was created
	tagExists, err := testutil.RunGit(t, dir, "tag", "-l", "my-release")
	if err != nil {
		t.Fatalf("Failed to list tags: %v", err)
	}
	if tagExists == "" {
		t.Error("Expected tag 'my-release' to exist")
	}
}
