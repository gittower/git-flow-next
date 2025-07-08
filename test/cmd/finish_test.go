package cmd_test

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gittower/git-flow-next/test/testutil"
)

// Test functions

// TestFinishFeatureBranch tests the basic feature branch finishing functionality.
// Steps:
// 1. Sets up a test repository and initializes git-flow
// 2. Creates a feature branch
// 3. Adds changes to the feature branch
// 4. Finishes the feature branch
// 5. Verifies the branch is merged into develop
// 6. Verifies the feature branch is deleted
func TestFinishFeatureBranch(t *testing.T) {
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults and create branches
	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
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

// TestFinishReleaseBranch tests the basic release branch finishing functionality.
// Steps:
// 1. Sets up a test repository and initializes git-flow
// 2. Creates a release branch
// 3. Adds changes to the release branch
// 4. Finishes the release branch
// 5. Verifies the branch is merged into main and develop
// 6. Verifies a tag is created
// 7. Verifies the release branch is deleted
func TestFinishReleaseBranch(t *testing.T) {
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults and create branches
	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
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

// TestFinishHotfixBranch tests the basic hotfix branch finishing functionality.
// Steps:
// 1. Sets up a test repository and initializes git-flow
// 2. Creates a hotfix branch
// 3. Adds changes to the hotfix branch
// 4. Finishes the hotfix branch
// 5. Verifies the branch is merged into main and develop
// 6. Verifies a tag is created
// 7. Verifies the hotfix branch is deleted
func TestFinishHotfixBranch(t *testing.T) {
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults and create branches
	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
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

// TestFinishWithCustomConfig tests finishing branches with custom configuration.
// Steps:
// 1. Sets up a test repository and initializes git-flow with custom config
// 2. Creates a feature branch
// 3. Adds changes to the feature branch
// 4. Finishes the feature branch
// 5. Verifies the branch is merged into the custom develop branch
// 6. Verifies the feature branch is deleted
func TestFinishWithCustomConfig(t *testing.T) {
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with custom configuration
	output, err := testutil.RunGitFlow(t, dir, "init",
		"--main", "custom-main", // custom main branch name
		"--develop", "custom-dev", // custom develop branch name
		"--feature", "f/", // custom feature prefix
		"--bugfix", "b/", // custom bugfix prefix
		"--release", "r/", // custom release prefix
		"--hotfix", "h/", // custom hotfix prefix
		"--support", "s/", // custom support prefix
		"--tag", "v") // custom tag prefix
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Create a feature branch
	output, err = testutil.RunGitFlow(t, dir, "feature", "start", "custom-feature")
	if err != nil {
		t.Fatalf("Failed to start feature branch: %v\nOutput: %s", err, output)
	}

	// Make changes and commit
	err = testutil.WriteFile(t, dir, "custom-feature.txt", "Custom feature content")
	if err != nil {
		t.Fatalf("Failed to create feature file: %v", err)
	}

	// Add and commit the change
	_, err = testutil.RunGit(t, dir, "add", "custom-feature.txt")
	if err != nil {
		t.Fatalf("Failed to add feature file: %v", err)
	}

	_, err = testutil.RunGit(t, dir, "commit", "-m", "Add custom feature")
	if err != nil {
		t.Fatalf("Failed to commit feature change: %v", err)
	}

	// Finish the feature branch
	output, err = testutil.RunGitFlow(t, dir, "feature", "finish", "custom-feature")
	if err != nil {
		t.Fatalf("Failed to finish feature branch: %v\nOutput: %s", err, output)
	}

	// Check if the feature branch was deleted
	if testutil.BranchExists(t, dir, "f/custom-feature") {
		t.Errorf("Expected 'f/custom-feature' branch to be deleted")
	}

	// Checkout develop
	_, err = testutil.RunGit(t, dir, "checkout", "custom-dev")
	if err != nil {
		t.Fatalf("Failed to checkout custom-dev: %v", err)
	}

	// Check if the changes were merged
	if _, err := os.Stat(filepath.Join(dir, "custom-feature.txt")); os.IsNotExist(err) {
		t.Errorf("Expected custom-feature.txt to exist in custom-dev branch")
	}
}

// TestFinishNonExistentBranch tests the behavior when attempting to finish a non-existent branch.
// Steps:
// 1. Sets up a test repository and initializes git-flow
// 2. Attempts to finish a non-existent branch
// 3. Verifies the operation fails with appropriate error
func TestFinishNonExistentBranch(t *testing.T) {
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults and create branches
	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
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

// TestFinishWithMergeConflict tests the behavior when finishing a branch with merge conflicts.
// Steps:
// 1. Sets up a test repository and initializes git-flow
// 2. Creates a feature branch
// 3. Adds conflicting changes to both feature and develop branches
// 4. Attempts to finish the feature branch
// 5. Verifies the operation fails with merge conflict
func TestFinishWithMergeConflict(t *testing.T) {
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults and create branches
	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
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

// TestFinishWithMergeAbort tests aborting a merge during branch finishing.
// Steps:
// 1. Sets up a test repository and initializes git-flow
// 2. Creates a feature branch
// 3. Adds conflicting changes to both feature and develop branches
// 4. Attempts to finish the feature branch
// 5. Aborts the merge when conflict occurs
// 6. Verifies the branches are in their original state
func TestFinishWithMergeAbort(t *testing.T) {
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults and create branches
	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
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

// TestFinishWithRebaseConflict tests the behavior when finishing a branch with rebase conflicts.
// Steps:
// 1. Sets up a test repository and initializes git-flow
// 2. Creates a feature branch
// 3. Adds changes to develop branch
// 4. Adds conflicting changes to feature branch
// 5. Attempts to finish the feature branch with rebase
// 6. Verifies the operation fails with rebase conflict
func TestFinishWithRebaseConflict(t *testing.T) {
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults and create branches
	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
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

// TestFinishWithMergeContinue tests continuing a merge after resolving conflicts.
// Steps:
// 1. Sets up a test repository and initializes git-flow
// 2. Creates a feature branch
// 3. Adds conflicting changes to both feature and develop branches
// 4. Attempts to finish the feature branch
// 5. Resolves conflicts and continues the merge
// 6. Verifies the branch is successfully finished
func TestFinishWithMergeContinue(t *testing.T) {
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults and create branches
	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
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

// TestFinishWithChildBranchConflict tests the behavior when finishing a branch with child branches.
// Steps:
// 1. Sets up a test repository and initializes git-flow
// 2. Creates a feature branch
// 3. Creates a child branch from the feature branch
// 4. Attempts to finish the feature branch
// 5. Verifies the operation fails due to child branch
func TestFinishWithChildBranchConflict(t *testing.T) {
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
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

// TestFinishReleaseWithMergeContinue tests continuing a merge after resolving conflicts in a release branch.
// Steps:
// 1. Sets up a test repository and initializes git-flow
// 2. Creates a release branch
// 3. Adds conflicting changes to both release and develop branches
// 4. Attempts to finish the release branch
// 5. Resolves conflicts and continues the merge
// 6. Verifies the release is successfully finished
func TestFinishReleaseWithMergeContinue(t *testing.T) {
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults and create branches
	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
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

// TestFinishNonStandardBranchWithForce tests finishing a non-standard branch with force flag.
// Steps:
// 1. Sets up a test repository and initializes git-flow
// 2. Creates a non-standard branch
// 3. Adds changes to the branch
// 4. Finishes the branch with force flag
// 5. Verifies the branch is merged into develop
// 6. Verifies the branch is deleted
func TestFinishNonStandardBranchWithForce(t *testing.T) {
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
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

// TestFinishNonStandardBranchWithoutForce tests the behavior when finishing a non-standard branch without force flag.
// Steps:
// 1. Sets up a test repository and initializes git-flow
// 2. Creates a non-standard branch
// 3. Attempts to finish the branch without force flag
// 4. Verifies the operation fails with appropriate error
func TestFinishNonStandardBranchWithoutForce(t *testing.T) {
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
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

// TestFinishNonStandardBranchWithTag tests finishing a non-standard branch with tag creation.
// Steps:
// 1. Sets up a test repository and initializes git-flow
// 2. Creates a non-standard branch
// 3. Adds changes to the branch
// 4. Finishes the branch with tag flag
// 5. Verifies the branch is merged into develop
// 6. Verifies a tag is created
// 7. Verifies the branch is deleted
func TestFinishNonStandardBranchWithTag(t *testing.T) {
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults and tag configuration
	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
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

// TestFinishFeatureWithTag tests finishing a feature branch with tag creation.
// Steps:
// 1. Sets up a test repository and initializes git-flow
// 2. Creates a feature branch
// 3. Adds changes to the feature branch
// 4. Finishes the feature branch with tag flag
// 5. Verifies the branch is merged into develop
// 6. Verifies a tag is created
// 7. Verifies the feature branch is deleted
func TestFinishFeatureWithTag(t *testing.T) {
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Create a feature branch
	output, err = testutil.RunGitFlow(t, dir, "feature", "start", "tagged-feature")
	if err != nil {
		t.Fatalf("Failed to create feature branch: %v\nOutput: %s", err, output)
	}

	// Create and commit a test file
	testutil.WriteFile(t, dir, "feature.txt", "feature content")
	_, err = testutil.RunGit(t, dir, "add", "feature.txt")
	if err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Add feature file")
	if err != nil {
		t.Fatalf("Failed to commit file: %v", err)
	}

	// Get path to the git-flow binary
	gitFlowPath, err := filepath.Abs(filepath.Join("..", "..", "git-flow"))
	if err != nil {
		t.Fatalf("Failed to get absolute path to git-flow: %v", err)
	}

	// Run git-flow directly with exec.Command to get full control over arguments
	cmd := exec.Command(gitFlowPath, "feature", "finish", "tagged-feature", "--tag")
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()
	if err != nil {
		t.Fatalf("Failed to finish feature branch: %v\nOutput: %s", err, stdout.String()+stderr.String())
	}
	output = stdout.String() + stderr.String()

	// Verify that a tag was created
	output, err = testutil.RunGit(t, dir, "tag", "-l")
	if err != nil {
		t.Fatalf("Failed to list tags: %v", err)
	}
	if !strings.Contains(output, "tagged-feature") {
		t.Error("Expected tag 'tagged-feature' to be created")
	}
}

// TestFinishReleaseWithCustomTag tests finishing a release branch with custom tag prefix.
// Steps:
// 1. Sets up a test repository and initializes git-flow with custom tag prefix
// 2. Creates a release branch
// 3. Adds changes to the release branch
// 4. Finishes the release branch
// 5. Verifies the branch is merged into main and develop
// 6. Verifies a tag is created with custom prefix
// 7. Verifies the release branch is deleted
func TestFinishReleaseWithCustomTag(t *testing.T) {
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Create a release branch
	output, err = testutil.RunGitFlow(t, dir, "release", "start", "1.2.0")
	if err != nil {
		t.Fatalf("Failed to create release branch: %v\nOutput: %s", err, output)
	}

	// Create and commit a test file
	testutil.WriteFile(t, dir, "release.txt", "release content")
	_, err = testutil.RunGit(t, dir, "add", "release.txt")
	if err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Add release file")
	if err != nil {
		t.Fatalf("Failed to commit file: %v", err)
	}

	// Get path to the git-flow binary
	gitFlowPath, err := filepath.Abs(filepath.Join("..", "..", "git-flow"))
	if err != nil {
		t.Fatalf("Failed to get absolute path to git-flow: %v", err)
	}

	// Run git-flow directly
	cmd := exec.Command(gitFlowPath, "release", "finish", "1.2.0", "--tagname", "v1.2.0-beta")
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()
	if err != nil {
		t.Fatalf("Failed to finish release branch: %v\nOutput: %s", err, stdout.String()+stderr.String())
	}
	output = stdout.String() + stderr.String()

	// Verify that custom tag was created
	output, err = testutil.RunGit(t, dir, "tag", "-l")
	if err != nil {
		t.Fatalf("Failed to list tags: %v", err)
	}
	if !strings.Contains(output, "v1.2.0-beta") {
		t.Error("Expected tag 'v1.2.0-beta' to be created")
	}
	if strings.Contains(output, "1.2.0") && !strings.Contains(output, "v1.2.0-beta") {
		t.Error("Expected tag to use custom name 'v1.2.0-beta' instead of '1.2.0'")
	}
}

// TestFinishReleaseWithCustomMessage tests finishing a release branch with custom commit message.
// Steps:
// 1. Sets up a test repository and initializes git-flow
// 2. Creates a release branch
// 3. Adds changes to the release branch
// 4. Finishes the release branch with custom message
// 5. Verifies the branch is merged into main and develop
// 6. Verifies the commit message matches the custom message
// 7. Verifies the release branch is deleted
func TestFinishReleaseWithCustomMessage(t *testing.T) {
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Set tag prefix for release branches
	_, err = testutil.RunGit(t, dir, "config", "gitflow.branch.release.tagprefix", "v")
	if err != nil {
		t.Fatalf("Failed to set tag prefix: %v", err)
	}

	// Create a release branch
	output, err = testutil.RunGitFlow(t, dir, "release", "start", "1.3.0")
	if err != nil {
		t.Fatalf("Failed to create release branch: %v\nOutput: %s", err, output)
	}

	// Create and commit a test file
	testutil.WriteFile(t, dir, "release.txt", "release content")
	_, err = testutil.RunGit(t, dir, "add", "release.txt")
	if err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Add release file")
	if err != nil {
		t.Fatalf("Failed to commit file: %v", err)
	}

	// Custom message for the tag
	customMessage := "This is release 1.3.0"

	// Get path to the git-flow binary
	gitFlowPath, err := filepath.Abs(filepath.Join("..", "..", "git-flow"))
	if err != nil {
		t.Fatalf("Failed to get absolute path to git-flow: %v", err)
	}

	// Run git-flow directly
	cmd := exec.Command(gitFlowPath, "release", "finish", "1.3.0", "--message", customMessage)
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()
	if err != nil {
		t.Fatalf("Failed to finish release branch: %v\nOutput: %s", err, stdout.String()+stderr.String())
	}
	output = stdout.String() + stderr.String()

	// Verify that the tag was created with the custom message
	output, err = testutil.RunGit(t, dir, "tag", "-n", "-l", "v1.3.0")
	if err != nil {
		t.Fatalf("Failed to get tag message: %v", err)
	}
	if !strings.Contains(output, customMessage) {
		t.Errorf("Expected tag message to contain '%s', got: %s", customMessage, output)
	}
}

// TestFinishReleaseWithNoTag tests finishing a release branch without creating a tag.
// Steps:
// 1. Sets up a test repository and initializes git-flow
// 2. Creates a release branch
// 3. Adds changes to the release branch
// 4. Finishes the release branch with no-tag flag
// 5. Verifies the branch is merged into main and develop
// 6. Verifies no tag is created
// 7. Verifies the release branch is deleted
func TestFinishReleaseWithNoTag(t *testing.T) {
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Create a release branch
	output, err = testutil.RunGitFlow(t, dir, "release", "start", "1.4.0")
	if err != nil {
		t.Fatalf("Failed to create release branch: %v\nOutput: %s", err, output)
	}

	// Create and commit a test file
	testutil.WriteFile(t, dir, "release.txt", "release content")
	_, err = testutil.RunGit(t, dir, "add", "release.txt")
	if err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Add release file")
	if err != nil {
		t.Fatalf("Failed to commit file: %v", err)
	}

	// Get path to the git-flow binary
	gitFlowPath, err := filepath.Abs(filepath.Join("..", "..", "git-flow"))
	if err != nil {
		t.Fatalf("Failed to get absolute path to git-flow: %v", err)
	}

	// Run git-flow directly
	cmd := exec.Command(gitFlowPath, "release", "finish", "1.4.0", "--notag")
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()
	if err != nil {
		t.Fatalf("Failed to finish release branch: %v\nOutput: %s", err, stdout.String()+stderr.String())
	}
	output = stdout.String() + stderr.String()

	// Verify that no tag was created
	output, err = testutil.RunGit(t, dir, "tag", "-l")
	if err != nil {
		t.Fatalf("Failed to list tags: %v", err)
	}
	if strings.Contains(output, "1.4.0") {
		t.Error("Expected no tag to be created with --notag flag")
	}
}

// TestFinishReleaseWithMessageFile tests finishing a release branch using a message file.
// Steps:
// 1. Sets up a test repository and initializes git-flow
// 2. Creates a release branch
// 3. Adds changes to the release branch
// 4. Creates a message file
// 5. Finishes the release branch using the message file
// 6. Verifies the branch is merged into main and develop
// 7. Verifies the commit message matches the file content
// 8. Verifies the release branch is deleted
func TestFinishReleaseWithMessageFile(t *testing.T) {
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Set tag prefix for release branches
	_, err = testutil.RunGit(t, dir, "config", "gitflow.branch.release.tagprefix", "v")
	if err != nil {
		t.Fatalf("Failed to set tag prefix: %v", err)
	}

	// Create a release branch
	output, err = testutil.RunGitFlow(t, dir, "release", "start", "1.5.0")
	if err != nil {
		t.Fatalf("Failed to create release branch: %v\nOutput: %s", err, output)
	}

	// Create and commit a test file
	testutil.WriteFile(t, dir, "release.txt", "release content")
	_, err = testutil.RunGit(t, dir, "add", "release.txt")
	if err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Add release file")
	if err != nil {
		t.Fatalf("Failed to commit file: %v", err)
	}

	// Create a message file for the tag
	tagMessageFilePath := filepath.Join(dir, "tag-message.txt")
	customMessage := "This is release 1.5.0\nWith a multi-line message\nThat describes all the changes"
	err = os.WriteFile(tagMessageFilePath, []byte(customMessage), 0644)
	if err != nil {
		t.Fatalf("Failed to create tag message file: %v", err)
	}

	// Get path to the git-flow binary
	gitFlowPath, err := filepath.Abs(filepath.Join("..", "..", "git-flow"))
	if err != nil {
		t.Fatalf("Failed to get absolute path to git-flow: %v", err)
	}

	// Run git-flow directly
	cmd := exec.Command(gitFlowPath, "release", "finish", "1.5.0", "--messagefile", tagMessageFilePath)
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()
	if err != nil {
		t.Fatalf("Failed to finish release branch: %v\nOutput: %s", err, stdout.String()+stderr.String())
	}
	output = stdout.String() + stderr.String()

	// Verify that the tag was created
	output, err = testutil.RunGit(t, dir, "tag", "-l")
	if err != nil {
		t.Fatalf("Failed to list tags: %v", err)
	}
	if !strings.Contains(output, "v1.5.0") {
		t.Error("Expected tag 'v1.5.0' to be created")
	}

	// Verify that the tag message matches the file content
	output, err = testutil.RunGit(t, dir, "tag", "-n99", "-l", "v1.5.0")
	if err != nil {
		t.Fatalf("Failed to get tag message: %v", err)
	}

	// Just verify that the tag message contains key parts of our file content
	if !strings.Contains(output, "This is release 1.5.0") {
		t.Errorf("Expected tag message to contain 'This is release 1.5.0'. Got: %s", output)
	}

	if !strings.Contains(output, "With a multi-line message") {
		t.Errorf("Expected tag message to contain 'With a multi-line message'. Got: %s", output)
	}

	if !strings.Contains(output, "That describes all the changes") {
		t.Errorf("Expected tag message to contain 'That describes all the changes'. Got: %s", output)
	}
}

// TestFinishReleaseWithConfigMessageFile tests finishing a release branch using a message file from config.
// Steps:
// 1. Sets up a test repository and initializes git-flow
// 2. Configures a default message file
// 3. Creates a release branch
// 4. Adds changes to the release branch
// 5. Finishes the release branch
// 6. Verifies the branch is merged into main and develop
// 7. Verifies the commit message matches the config file content
// 8. Verifies the release branch is deleted
func TestFinishReleaseWithConfigMessageFile(t *testing.T) {
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Set tag prefix for release branches
	_, err = testutil.RunGit(t, dir, "config", "gitflow.branch.release.tagprefix", "v")
	if err != nil {
		t.Fatalf("Failed to set tag prefix: %v", err)
	}

	// Create a message file for the tag
	tagMessageFilePath := filepath.Join(dir, "config-tag-message.txt")
	customMessage := "This message comes from a config-specified file"
	err = os.WriteFile(tagMessageFilePath, []byte(customMessage), 0644)
	if err != nil {
		t.Fatalf("Failed to create tag message file: %v", err)
	}

	// Set the message file in git config
	_, err = testutil.RunGit(t, dir, "config", "gitflow.release.finish.messagefile", tagMessageFilePath)
	if err != nil {
		t.Fatalf("Failed to set message file config: %v", err)
	}

	// Create a release branch
	output, err = testutil.RunGitFlow(t, dir, "release", "start", "1.6.0")
	if err != nil {
		t.Fatalf("Failed to create release branch: %v\nOutput: %s", err, output)
	}

	// Create and commit a test file
	testutil.WriteFile(t, dir, "release.txt", "release content")
	_, err = testutil.RunGit(t, dir, "add", "release.txt")
	if err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Add release file")
	if err != nil {
		t.Fatalf("Failed to commit file: %v", err)
	}

	// Finish the release branch (should use message file from config)
	output, err = testutil.RunGitFlow(t, dir, "release", "finish", "1.6.0")
	if err != nil {
		t.Fatalf("Failed to finish release branch: %v\nOutput: %s", err, output)
	}

	// Verify that the tag was created
	output, err = testutil.RunGit(t, dir, "tag", "-l")
	if err != nil {
		t.Fatalf("Failed to list tags: %v", err)
	}
	if !strings.Contains(output, "v1.6.0") {
		t.Error("Expected tag 'v1.6.0' to be created")
	}

	// Verify that the tag message matches the file content
	output, err = testutil.RunGit(t, dir, "tag", "-n99", "-l", "v1.6.0")
	if err != nil {
		t.Fatalf("Failed to get tag message: %v", err)
	}

	if !strings.Contains(output, "This message comes from a config-specified file") {
		t.Errorf("Expected tag message to contain 'This message comes from a config-specified file'. Got: %s", output)
	}
}

// TestFinishTagFromBranchConfig tests finishing a branch with tag configuration from branch config.
// Steps:
// 1. Sets up a test repository and initializes git-flow
// 2. Configures tag settings for a branch type
// 3. Creates a branch of that type
// 4. Adds changes to the branch
// 5. Finishes the branch
// 6. Verifies the branch is merged
// 7. Verifies a tag is created according to config
// 8. Verifies the branch is deleted
func TestFinishTagFromBranchConfig(t *testing.T) {
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Verify that release branches create tags by default (branch config)
	configOutput, err := testutil.RunGit(t, dir, "config", "gitflow.branch.release.tag")
	if err == nil && configOutput == "false" {
		// If it's already set to false, reset it to true
		_, err = testutil.RunGit(t, dir, "config", "gitflow.branch.release.tag", "true")
		if err != nil {
			t.Fatalf("Failed to reset tag config: %v", err)
		}
	}

	// Create a release branch
	output, err = testutil.RunGitFlow(t, dir, "release", "start", "2.0.0")
	if err != nil {
		t.Fatalf("Failed to create release branch: %v\nOutput: %s", err, output)
	}

	// Create and commit a test file
	testutil.WriteFile(t, dir, "release.txt", "release content")
	_, err = testutil.RunGit(t, dir, "add", "release.txt")
	if err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Add release file")
	if err != nil {
		t.Fatalf("Failed to commit file: %v", err)
	}

	// Finish the release branch using default command (no options)
	// This should create a tag
	output, err = testutil.RunGitFlow(t, dir, "release", "finish", "2.0.0")
	if err != nil {
		t.Fatalf("Failed to finish release branch: %v\nOutput: %s", err, output)
	}

	// Check that tag was created
	tagList, err := testutil.RunGit(t, dir, "tag", "-l")
	if err != nil {
		t.Fatalf("Failed to list tags: %v", err)
	}
	t.Logf("Tags: %s", tagList)

	if !strings.Contains(tagList, "2.0.0") {
		t.Errorf("Expected tag '2.0.0' to be created (branch config should create tag by default)")
	}
}

// TestFinishNotagFromCLI tests that the --no-tag flag overrides configuration.
// Steps:
// 1. Sets up a test repository and initializes git-flow
// 2. Configures tag creation for a branch type
// 3. Creates a branch of that type
// 4. Adds changes to the branch
// 5. Finishes the branch with --no-tag flag
// 6. Verifies the branch is merged
// 7. Verifies no tag is created despite config
// 8. Verifies the branch is deleted
func TestFinishNotagFromCLI(t *testing.T) {
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Ensure release branches are configured to create tags (branch config)
	_, err = testutil.RunGit(t, dir, "config", "gitflow.branch.release.tag", "true")
	if err != nil {
		t.Fatalf("Failed to set tag config: %v", err)
	}

	// Verify tag configuration is enabled
	tagConfig, err := testutil.RunGit(t, dir, "config", "gitflow.branch.release.tag")
	if err != nil || strings.TrimSpace(tagConfig) != "true" {
		t.Fatalf("Failed to verify tag config is enabled: %v, got: %s", err, tagConfig)
	}

	// Create a release branch
	output, err = testutil.RunGitFlow(t, dir, "release", "start", "2.1.0")
	if err != nil {
		t.Fatalf("Failed to create release branch: %v\nOutput: %s", err, output)
	}

	// Create and commit a test file
	testutil.WriteFile(t, dir, "release.txt", "release content")
	_, err = testutil.RunGit(t, dir, "add", "release.txt")
	if err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Add release file")
	if err != nil {
		t.Fatalf("Failed to commit file: %v", err)
	}

	// Get path to the git-flow binary
	gitFlowPath, err := filepath.Abs(filepath.Join("..", "..", "git-flow"))
	if err != nil {
		t.Fatalf("Failed to get absolute path to git-flow: %v", err)
	}

	// Finish with --notag to override the config
	cmd := exec.Command(gitFlowPath, "release", "finish", "2.1.0", "--notag")
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()
	if err != nil {
		t.Fatalf("Failed to finish release: %v\nOutput: %s", err, stdout.String()+stderr.String())
	}
	t.Logf("Command output: %s", stdout.String()+stderr.String())

	// Check that no tag was created (CLI --notag should override config)
	tagList, err := testutil.RunGit(t, dir, "tag", "-l")
	if err != nil {
		t.Fatalf("Failed to list tags: %v", err)
	}
	t.Logf("Tags: %s", tagList)

	if strings.Contains(tagList, "2.1.0") {
		t.Errorf("No tag should have been created when --notag is specified (CLI should override config)")
	}
}

// TestFinishNotagFromConfig tests that tag configuration can be disabled.
// Steps:
// 1. Sets up a test repository and initializes git-flow
// 2. Configures tag creation to be disabled for a branch type
// 3. Creates a branch of that type
// 4. Adds changes to the branch
// 5. Finishes the branch
// 6. Verifies the branch is merged
// 7. Verifies no tag is created
// 8. Verifies the branch is deleted
func TestFinishNotagFromConfig(t *testing.T) {
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Ensure release branches are configured to create tags
	_, err = testutil.RunGit(t, dir, "config", "gitflow.branch.release.tag", "true")
	if err != nil {
		t.Fatalf("Failed to set tag config: %v", err)
	}

	// Set the finish.notag config to true
	_, err = testutil.RunGit(t, dir, "config", "gitflow.release.finish.notag", "true")
	if err != nil {
		t.Fatalf("Failed to set finish.notag config: %v", err)
	}

	// Verify configs are set correctly
	tagConfig, err := testutil.RunGit(t, dir, "config", "gitflow.branch.release.tag")
	if err != nil || strings.TrimSpace(tagConfig) != "true" {
		t.Fatalf("Failed to verify branch tag config is enabled: %v, got: %s", err, tagConfig)
	}

	notagConfig, err := testutil.RunGit(t, dir, "config", "gitflow.release.finish.notag")
	if err != nil || strings.TrimSpace(notagConfig) != "true" {
		t.Fatalf("Failed to verify finish.notag config is enabled: %v, got: %s", err, notagConfig)
	}

	// Create a release branch
	output, err = testutil.RunGitFlow(t, dir, "release", "start", "2.2.0")
	if err != nil {
		t.Fatalf("Failed to create release branch: %v\nOutput: %s", err, output)
	}

	// Create and commit a test file
	testutil.WriteFile(t, dir, "release.txt", "release content")
	_, err = testutil.RunGit(t, dir, "add", "release.txt")
	if err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Add release file")
	if err != nil {
		t.Fatalf("Failed to commit file: %v", err)
	}

	// Finish the release branch (should use notag from config)
	output, err = testutil.RunGitFlow(t, dir, "release", "finish", "2.2.0")
	if err != nil {
		t.Fatalf("Failed to finish release branch: %v\nOutput: %s", err, output)
	}

	// Check that no tag was created
	tagList, err := testutil.RunGit(t, dir, "tag", "-l")
	if err != nil {
		t.Fatalf("Failed to list tags: %v", err)
	}
	t.Logf("Tags: %s", tagList)

	if strings.Contains(tagList, "2.2.0") {
		t.Errorf("No tag should have been created when gitflow.release.finish.notag is true")
	}
}

// TestFinishFeatureBranchDefaultLocalDeletion tests that feature branches are deleted locally by default.
// Steps:
// 1. Sets up a test repository and initializes git-flow
// 2. Creates a feature branch
// 3. Adds changes to the feature branch
// 4. Finishes the feature branch
// 5. Verifies the local branch is deleted
func TestFinishFeatureBranchDefaultLocalDeletion(t *testing.T) {
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults and create branches
	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
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

	// Verify that local feature branch is deleted
	if testutil.BranchExists(t, dir, "feature/my-feature") {
		t.Error("Expected local feature branch to be deleted by default")
	}
}

// TestFinishFeatureBranchDefaultRemoteDeletion tests that feature branches are deleted both locally and remotely by default.
// Steps:
// 1. Sets up a test repository and initializes git-flow
// 2. Creates a feature branch
// 3. Adds changes to the feature branch
// 4. Adds a remote repository
// 5. Finishes the feature branch
// 6. Verifies both local and remote branches are deleted
func TestFinishFeatureBranchDefaultRemoteDeletion(t *testing.T) {
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults and create branches
	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
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

	// Add a remote repository
	remoteDir, err := testutil.AddRemote(t, dir, "origin", true)
	if err != nil {
		t.Fatalf("Failed to add remote: %v", err)
	}
	defer testutil.CleanupTestRepo(t, remoteDir)

	// Push the feature branch to remote
	_, err = testutil.RunGit(t, dir, "push", "origin", "feature/my-feature")
	if err != nil {
		t.Fatalf("Failed to push feature branch: %v", err)
	}

	// Finish the feature branch
	output, err = testutil.RunGitFlow(t, dir, "feature", "finish", "my-feature")
	if err != nil {
		t.Fatalf("Failed to finish feature branch: %v\nOutput: %s", err, output)
	}

	// Verify that local feature branch is deleted
	if testutil.BranchExists(t, dir, "feature/my-feature") {
		t.Error("Expected local feature branch to be deleted by default")
	}

	// Verify that remote feature branch is deleted
	_, err = testutil.RunGit(t, dir, "fetch", "origin")
	if err != nil {
		t.Fatalf("Failed to fetch from remote: %v", err)
	}
	if testutil.BranchExists(t, dir, "origin/feature/my-feature") {
		t.Error("Expected remote feature branch to be deleted by default")
	}
}

// TestFinishFeatureBranchKeepLocal tests that the keep-local option preserves the local branch when finishing.
// Steps:
// 1. Sets up a test repository and initializes git-flow
// 2. Creates a feature branch
// 3. Adds changes to the feature branch
// 4. Finishes the feature branch with the keeplocal option
// 5. Verifies the branch is merged into develop
// 6. Verifies the local feature branch is preserved
func TestFinishFeatureBranchKeepLocal(t *testing.T) {
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Create a feature branch
	output, err = testutil.RunGitFlow(t, dir, "feature", "start", "keep-local-test")
	if err != nil {
		t.Fatalf("Failed to create feature branch: %v\nOutput: %s", err, output)
	}

	// Create a test file
	testutil.WriteFile(t, dir, "test.txt", "feature content")

	// Commit the changes
	_, err = testutil.RunGit(t, dir, "add", "test.txt")
	if err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Add test file")
	if err != nil {
		t.Fatalf("Failed to commit file: %v", err)
	}

	// Get path to the git-flow binary
	gitFlowPath, err := filepath.Abs(filepath.Join("..", "..", "git-flow"))
	if err != nil {
		t.Fatalf("Failed to get absolute path to git-flow: %v", err)
	}

	// Finish the feature branch with keeplocal option
	cmd := exec.Command(gitFlowPath, "feature", "finish", "keep-local-test", "--keeplocal")
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()
	if err != nil {
		t.Fatalf("Failed to finish feature branch: %v\nOutput: %s", err, stdout.String()+stderr.String())
	}
	output = stdout.String() + stderr.String()

	// Verify that we're now on develop branch
	currentBranch := testutil.GetCurrentBranch(t, dir)
	if currentBranch != "develop" {
		t.Errorf("Expected to be on develop branch after finish, got %s", currentBranch)
	}

	// Verify that the changes are in develop
	if !testutil.FileExists(t, dir, "test.txt") {
		t.Error("Expected test.txt to exist in develop branch")
	}

	// Verify that the local branch still exists
	if !testutil.BranchExists(t, dir, "feature/keep-local-test") {
		t.Error("Expected feature branch to still exist with --keeplocal option")
	}

	// Checkout the feature branch and verify it still has the content
	_, err = testutil.RunGit(t, dir, "checkout", "feature/keep-local-test")
	if err != nil {
		t.Fatalf("Failed to checkout feature branch: %v", err)
	}

	// Verify the file content in the feature branch
	content := testutil.ReadFile(t, dir, "test.txt")
	if content != "feature content" {
		t.Errorf("Expected file content to be 'feature content', got '%s'", content)
	}
}

// TestFinishFeatureBranchKeepRemote tests that the keep-remote option preserves the remote branch when finishing.
// Steps:
// 1. Sets up a test repository and initializes git-flow
// 2. Creates a feature branch
// 3. Adds changes to the feature branch
// 4. Adds a remote and pushes the branch
// 5. Finishes the feature branch with the keepremote option
// 6. Verifies the branch is merged into develop
// 7. Verifies the local feature branch is deleted
// 8. Verifies the remote feature branch is preserved
func TestFinishFeatureBranchKeepRemote(t *testing.T) {
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Create a feature branch
	output, err = testutil.RunGitFlow(t, dir, "feature", "start", "keep-remote-test")
	if err != nil {
		t.Fatalf("Failed to create feature branch: %v\nOutput: %s", err, output)
	}

	// Create a test file
	testutil.WriteFile(t, dir, "test.txt", "feature content")

	// Commit the changes
	_, err = testutil.RunGit(t, dir, "add", "test.txt")
	if err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Add test file")
	if err != nil {
		t.Fatalf("Failed to commit file: %v", err)
	}

	// Add a remote repository
	remoteDir, err := testutil.AddRemote(t, dir, "origin", true)
	if err != nil {
		t.Fatalf("Failed to add remote: %v", err)
	}
	defer testutil.CleanupTestRepo(t, remoteDir)

	// Push the feature branch to remote
	_, err = testutil.RunGit(t, dir, "push", "origin", "feature/keep-remote-test")
	if err != nil {
		t.Fatalf("Failed to push feature branch: %v", err)
	}

	// Get path to the git-flow binary
	gitFlowPath, err := filepath.Abs(filepath.Join("..", "..", "git-flow"))
	if err != nil {
		t.Fatalf("Failed to get absolute path to git-flow: %v", err)
	}

	// Finish the feature branch with keepremote option
	cmd := exec.Command(gitFlowPath, "feature", "finish", "keep-remote-test", "--keepremote")
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()
	if err != nil {
		t.Fatalf("Failed to finish feature branch: %v\nOutput: %s", err, stdout.String()+stderr.String())
	}
	output = stdout.String() + stderr.String()

	// Verify that we're now on develop branch
	currentBranch := testutil.GetCurrentBranch(t, dir)
	if currentBranch != "develop" {
		t.Errorf("Expected to be on develop branch after finish, got %s", currentBranch)
	}

	// Verify that the changes are in develop
	if !testutil.FileExists(t, dir, "test.txt") {
		t.Error("Expected test.txt to exist in develop branch")
	}

	// Verify that the local branch is deleted
	if testutil.BranchExists(t, dir, "feature/keep-remote-test") {
		t.Error("Expected local feature branch to be deleted")
	}

	// Fetch from remote to update references
	_, err = testutil.RunGit(t, dir, "fetch", "origin")
	if err != nil {
		t.Fatalf("Failed to fetch from remote: %v", err)
	}

	// Verify that the remote branch still exists
	remoteExists := testutil.RemoteBranchExists(t, dir, "origin", "feature/keep-remote-test")
	if !remoteExists {
		t.Error("Expected remote feature branch to still exist with --keepremote option")
	}

	// Try to checkout the remote branch and verify it has the content
	_, err = testutil.RunGit(t, dir, "checkout", "-b", "verify-remote", "origin/feature/keep-remote-test")
	if err != nil {
		t.Fatalf("Failed to checkout remote branch: %v", err)
	}

	// Verify the file content from the remote branch
	content := testutil.ReadFile(t, dir, "test.txt")
	if content != "feature content" {
		t.Errorf("Expected file content to be 'feature content', got '%s'", content)
	}
}

// TestFinishFeatureBranchWithFetchFlag tests that the --fetch flag works when finishing a branch.
// Steps:
// 1. Sets up a test repository with a remote and initializes git-flow
// 2. Creates a feature branch and adds changes
// 3. Simulates remote changes by updating the target branch in the remote
// 4. Finishes the feature branch with --fetch flag
// 5. Verifies the fetch occurs and remote changes are incorporated
func TestFinishFeatureBranchWithFetchFlag(t *testing.T) {
	// Setup test repo
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Add a remote repository
	remoteDir, err := testutil.AddRemote(t, dir, "origin", true)
	if err != nil {
		t.Fatalf("Failed to add remote: %v", err)
	}
	defer testutil.CleanupTestRepo(t, remoteDir)

	// Create a feature branch
	output, err = testutil.RunGitFlow(t, dir, "feature", "start", "fetch-test")
	if err != nil {
		t.Fatalf("Failed to create feature branch: %v\nOutput: %s", err, output)
	}

	// Create and commit a change on the feature branch
	testutil.WriteFile(t, dir, "feature.txt", "feature content")
	_, err = testutil.RunGit(t, dir, "add", "feature.txt")
	if err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Add feature file")
	if err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Create a second clone to simulate a different developer making changes
	tempDir2, err := os.MkdirTemp("", "git-flow-test-clone-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory for clone: %v", err)
	}
	defer testutil.CleanupTestRepo(t, tempDir2)

	// Clone the remote to the second directory
	_, err = testutil.RunGit(t, tempDir2, "clone", remoteDir, ".")
	if err != nil {
		t.Fatalf("Failed to clone remote: %v", err)
	}

	// Configure Git user in the clone
	_, err = testutil.RunGit(t, tempDir2, "config", "user.name", "Test User")
	if err != nil {
		t.Fatalf("Failed to configure Git user name in clone: %v", err)
	}
	_, err = testutil.RunGit(t, tempDir2, "config", "user.email", "test@example.com")
	if err != nil {
		t.Fatalf("Failed to configure Git user email in clone: %v", err)
	}

	// In the clone, make a change to develop and push it
	_, err = testutil.RunGit(t, tempDir2, "checkout", "develop")
	if err != nil {
		t.Fatalf("Failed to checkout develop in clone: %v", err)
	}
	testutil.WriteFile(t, tempDir2, "remote-change.txt", "remote content")
	_, err = testutil.RunGit(t, tempDir2, "add", "remote-change.txt")
	if err != nil {
		t.Fatalf("Failed to add remote change file: %v", err)
	}
	_, err = testutil.RunGit(t, tempDir2, "commit", "-m", "Remote change on develop")
	if err != nil {
		t.Fatalf("Failed to commit remote change: %v", err)
	}
	_, err = testutil.RunGit(t, tempDir2, "push", "origin", "develop")
	if err != nil {
		t.Fatalf("Failed to push develop: %v", err)
	}

	// Finish the feature branch with --fetch flag
	output, err = testutil.RunGitFlow(t, dir, "feature", "finish", "fetch-test", "--fetch")
	if err != nil {
		t.Fatalf("Failed to finish feature branch: %v\nOutput: %s", err, output)
	}

	// Verify that fetch occurred by checking output
	if !strings.Contains(output, "Fetching from origin") {
		t.Errorf("Expected fetch operation to be mentioned in output, but got: %s", output)
	}

	// Verify both the feature content and remote change are present
	featureContent := testutil.ReadFile(t, dir, "feature.txt")
	if featureContent != "feature content" {
		t.Errorf("Expected feature content to be 'feature content', got '%s'", featureContent)
	}

	remoteContent := testutil.ReadFile(t, dir, "remote-change.txt")
	if remoteContent != "remote content" {
		t.Errorf("Expected remote content to be 'remote content', got '%s'", remoteContent)
	}

	// Verify we're on develop branch
	currentBranch := testutil.GetCurrentBranch(t, dir)
	if currentBranch != "develop" {
		t.Errorf("Expected to be on develop branch after finish, got %s", currentBranch)
	}
}

// TestFinishFeatureBranchWithNoFetchFlag tests that the --no-fetch flag prevents fetching.
// Steps:
// 1. Sets up a test repository with fetch config enabled
// 2. Creates a feature branch
// 3. Simulates remote changes
// 4. Finishes with --no-fetch flag
// 5. Verifies no fetch occurs despite config setting
func TestFinishFeatureBranchWithNoFetchFlag(t *testing.T) {
	// Setup test repo
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Set config to enable fetch by default
	_, err = testutil.RunGit(t, dir, "config", "gitflow.feature.finish.fetch", "true")
	if err != nil {
		t.Fatalf("Failed to set fetch config: %v", err)
	}

	// Add a remote repository
	remoteDir, err := testutil.AddRemote(t, dir, "origin", true)
	if err != nil {
		t.Fatalf("Failed to add remote: %v", err)
	}
	defer testutil.CleanupTestRepo(t, remoteDir)

	// Create a feature branch
	output, err = testutil.RunGitFlow(t, dir, "feature", "start", "no-fetch-test")
	if err != nil {
		t.Fatalf("Failed to create feature branch: %v\nOutput: %s", err, output)
	}

	// Create and commit a change on the feature branch
	testutil.WriteFile(t, dir, "feature.txt", "feature content")
	_, err = testutil.RunGit(t, dir, "add", "feature.txt")
	if err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Add feature file")
	if err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Finish the feature branch with --no-fetch flag (should override config)
	output, err = testutil.RunGitFlow(t, dir, "feature", "finish", "no-fetch-test", "--no-fetch")
	if err != nil {
		t.Fatalf("Failed to finish feature branch: %v\nOutput: %s", err, output)
	}

	// Verify that fetch did NOT occur by checking output
	if strings.Contains(output, "Fetching from origin") {
		t.Errorf("Expected no fetch operation due to --no-fetch flag, but fetch occurred: %s", output)
	}

	// Verify the feature was merged
	currentBranch := testutil.GetCurrentBranch(t, dir)
	if currentBranch != "develop" {
		t.Errorf("Expected to be on develop branch after finish, got %s", currentBranch)
	}
}

// TestFinishFeatureBranchWithFetchConfig tests that the config setting enables fetch.
// Steps:
// 1. Sets up a test repository with fetch config enabled
// 2. Creates a feature branch and adds changes
// 3. Simulates remote changes
// 4. Finishes without explicit fetch flag
// 5. Verifies fetch occurs due to config setting
func TestFinishFeatureBranchWithFetchConfig(t *testing.T) {
	// Setup test repo
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Set config to enable fetch
	_, err = testutil.RunGit(t, dir, "config", "gitflow.feature.finish.fetch", "true")
	if err != nil {
		t.Fatalf("Failed to set fetch config: %v", err)
	}

	// Add a remote repository
	remoteDir, err := testutil.AddRemote(t, dir, "origin", true)
	if err != nil {
		t.Fatalf("Failed to add remote: %v", err)
	}
	defer testutil.CleanupTestRepo(t, remoteDir)

	// Create a feature branch
	output, err = testutil.RunGitFlow(t, dir, "feature", "start", "config-fetch-test")
	if err != nil {
		t.Fatalf("Failed to create feature branch: %v\nOutput: %s", err, output)
	}

	// Create and commit a change on the feature branch
	testutil.WriteFile(t, dir, "feature.txt", "feature content")
	_, err = testutil.RunGit(t, dir, "add", "feature.txt")
	if err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Add feature file")
	if err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Finish the feature branch without explicit fetch flag (should use config)
	output, err = testutil.RunGitFlow(t, dir, "feature", "finish", "config-fetch-test")
	if err != nil {
		t.Fatalf("Failed to finish feature branch: %v\nOutput: %s", err, output)
	}

	// Verify that fetch occurred by checking output
	if !strings.Contains(output, "Fetching from origin") {
		t.Errorf("Expected fetch operation due to config setting, but fetch did not occur: %s", output)
	}

	// Verify the feature was merged
	currentBranch := testutil.GetCurrentBranch(t, dir)
	if currentBranch != "develop" {
		t.Errorf("Expected to be on develop branch after finish, got %s", currentBranch)
	}
}

// TestFinishFeatureBranchDefaultNoFetch tests that the default behavior is no fetch.
// Steps:
// 1. Sets up a test repository without fetch config
// 2. Creates a feature branch
// 3. Finishes without any fetch flags
// 4. Verifies no fetch occurs by default
func TestFinishFeatureBranchDefaultNoFetch(t *testing.T) {
	// Setup test repo
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Add a remote repository
	remoteDir, err := testutil.AddRemote(t, dir, "origin", true)
	if err != nil {
		t.Fatalf("Failed to add remote: %v", err)
	}
	defer testutil.CleanupTestRepo(t, remoteDir)

	// Create a feature branch
	output, err = testutil.RunGitFlow(t, dir, "feature", "start", "default-test")
	if err != nil {
		t.Fatalf("Failed to create feature branch: %v\nOutput: %s", err, output)
	}

	// Create and commit a change on the feature branch
	testutil.WriteFile(t, dir, "feature.txt", "feature content")
	_, err = testutil.RunGit(t, dir, "add", "feature.txt")
	if err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Add feature file")
	if err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Finish the feature branch without any fetch options
	output, err = testutil.RunGitFlow(t, dir, "feature", "finish", "default-test")
	if err != nil {
		t.Fatalf("Failed to finish feature branch: %v\nOutput: %s", err, output)
	}

	// Verify that fetch did NOT occur by checking output
	if strings.Contains(output, "Fetching from origin") {
		t.Errorf("Expected no fetch operation by default, but fetch occurred: %s", output)
	}

	// Verify the feature was merged
	currentBranch := testutil.GetCurrentBranch(t, dir)
	if currentBranch != "develop" {
		t.Errorf("Expected to be on develop branch after finish, got %s", currentBranch)
	}
}

// TestFinishReleaseBranchWithFetch tests that fetch works for release branches too.
func TestFinishReleaseBranchWithFetch(t *testing.T) {
	// Setup test repo
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Add a remote repository
	remoteDir, err := testutil.AddRemote(t, dir, "origin", true)
	if err != nil {
		t.Fatalf("Failed to add remote: %v", err)
	}
	defer testutil.CleanupTestRepo(t, remoteDir)

	// Create a release branch
	output, err = testutil.RunGitFlow(t, dir, "release", "start", "1.0.0")
	if err != nil {
		t.Fatalf("Failed to create release branch: %v\nOutput: %s", err, output)
	}

	// Create and commit a change on the release branch
	testutil.WriteFile(t, dir, "release.txt", "release content")
	_, err = testutil.RunGit(t, dir, "add", "release.txt")
	if err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Add release file")
	if err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Finish the release branch with --fetch flag
	output, err = testutil.RunGitFlow(t, dir, "release", "finish", "1.0.0", "--fetch")
	if err != nil {
		t.Fatalf("Failed to finish release branch: %v\nOutput: %s", err, output)
	}

	// Verify that fetch occurred by checking output
	if !strings.Contains(output, "Fetching from origin") {
		t.Errorf("Expected fetch operation for release branch, but got: %s", output)
	}

	// Verify we're on main branch (release branches merge to main)
	currentBranch := testutil.GetCurrentBranch(t, dir)
	if currentBranch != "main" {
		t.Errorf("Expected to be on main branch after release finish, got %s", currentBranch)
	}
}

// TestFinishHotfixBranchWithFetch tests that fetch works for hotfix branches too.
func TestFinishHotfixBranchWithFetch(t *testing.T) {
	// Setup test repo
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Add a remote repository
	remoteDir, err := testutil.AddRemote(t, dir, "origin", true)
	if err != nil {
		t.Fatalf("Failed to add remote: %v", err)
	}
	defer testutil.CleanupTestRepo(t, remoteDir)

	// Create a hotfix branch
	output, err = testutil.RunGitFlow(t, dir, "hotfix", "start", "1.0.1")
	if err != nil {
		t.Fatalf("Failed to create hotfix branch: %v\nOutput: %s", err, output)
	}

	// Create and commit a change on the hotfix branch
	testutil.WriteFile(t, dir, "hotfix.txt", "hotfix content")
	_, err = testutil.RunGit(t, dir, "add", "hotfix.txt")
	if err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Add hotfix file")
	if err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Finish the hotfix branch with --fetch flag
	output, err = testutil.RunGitFlow(t, dir, "hotfix", "finish", "1.0.1", "--fetch")
	if err != nil {
		t.Fatalf("Failed to finish hotfix branch: %v\nOutput: %s", err, output)
	}

	// Verify that fetch occurred by checking output
	if !strings.Contains(output, "Fetching from origin") {
		t.Errorf("Expected fetch operation for hotfix branch, but got: %s", output)
	}

	// Verify we're on main branch (hotfix branches merge to main)
	currentBranch := testutil.GetCurrentBranch(t, dir)
	if currentBranch != "main" {
		t.Errorf("Expected to be on main branch after hotfix finish, got %s", currentBranch)
	}
}
