package cmd_test

import (
	"os"
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

	// Create and switch to feature branch (from clean develop)
	output, err = testutil.RunGitFlow(t, dir, "feature", "start", "rebase-feature")
	if err != nil {
		t.Fatalf("Failed to create feature branch: %v\nOutput: %s", err, output)
	}

	// Create a file in feature branch
	testutil.WriteFile(t, dir, "test.txt", "feature content")

	// Commit the changes in feature branch
	_, err = testutil.RunGit(t, dir, "add", "test.txt")
	if err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Add test.txt in feature")
	if err != nil {
		t.Fatalf("Failed to commit file: %v", err)
	}

	// Switch back to develop and create conflicting content
	_, err = testutil.RunGit(t, dir, "checkout", "develop")
	if err != nil {
		t.Fatalf("Failed to checkout develop: %v", err)
	}

	// Create the same file with different content in develop
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

	// Try to finish the feature branch with rebase
	output, err = testutil.RunGitFlow(t, dir, "feature", "finish", "--rebase", "rebase-feature")
	if err == nil {
		t.Fatal("Expected finish to fail due to rebase conflict")
	}

	// Verify that we're in a rebase conflict state
	rebaseMergeDir := filepath.Join(dir, ".git", "rebase-merge")
	if _, err := os.Stat(rebaseMergeDir); os.IsNotExist(err) {
		t.Error("Expected to be in rebase conflict state, but rebase-merge directory does not exist")
	}

	// Check that the correct branch is being rebased
	headNameFile := filepath.Join(rebaseMergeDir, "head-name")
	if headNameBytes, err := os.ReadFile(headNameFile); err != nil {
		t.Errorf("Failed to read head-name file: %v", err)
	} else {
		headName := strings.TrimSpace(string(headNameBytes))
		expectedBranch := "refs/heads/feature/rebase-feature"
		if headName != expectedBranch {
			t.Errorf("Expected to be rebasing branch %s, got %s", expectedBranch, headName)
		}
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

	// Continue the finish operation - should complete the merge automatically
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

	// Continue the finish operation - should complete the merge automatically
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

	// Continue the finish operation - should complete the merge automatically
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

// TestFinishUsesConfiguredParent tests that finish command uses configured parent branch, not stored base.
// Steps:
// 1. Sets up a test repository and initializes git-flow with defaults
// 2. Creates a feature branch (which stores base as 'develop')
// 3. Changes the feature branch type configuration to point to main
// 4. Finishes the feature branch
// 5. Verifies the feature was merged into main (configured parent) not develop (stored base)
func TestFinishUsesConfiguredParent(t *testing.T) {
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Create feature branch (stores 'develop' as base)
	output, err = testutil.RunGitFlow(t, dir, "feature", "start", "stored-base-test")
	if err != nil {
		t.Fatalf("Failed to start feature branch: %v\nOutput: %s", err, output)
	}

	// Add a change to feature branch
	testutil.WriteFile(t, dir, "feature.txt", "feature content")
	_, err = testutil.RunGit(t, dir, "add", "feature.txt")
	if err != nil {
		t.Fatalf("Failed to add feature file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Add feature file")
	if err != nil {
		t.Fatalf("Failed to commit feature file: %v", err)
	}

	// Change feature branch type configuration to point to main (should be ignored)
	_, err = testutil.RunGit(t, dir, "config", "gitflow.branch.feature.parent", "main")
	if err != nil {
		t.Fatalf("Failed to change feature parent config: %v", err)
	}

	// Verify stored base is still develop
	storedBase, err := testutil.RunGit(t, dir, "config", "--get", "gitflow.branch.feature/stored-base-test.base")
	if err != nil {
		t.Fatalf("Failed to get stored base: %v", err)
	}
	if strings.TrimSpace(storedBase) != "develop" {
		t.Errorf("Expected stored base to be 'develop', got '%s'", strings.TrimSpace(storedBase))
	}

	// Get develop commit before merge
	developBefore, err := testutil.RunGit(t, dir, "rev-parse", "develop")
	if err != nil {
		t.Fatalf("Failed to get develop commit: %v", err)
	}

	// Get main commit before merge
	mainBefore, err := testutil.RunGit(t, dir, "rev-parse", "main")
	if err != nil {
		t.Fatalf("Failed to get main commit: %v", err)
	}

	// Finish the feature branch
	output, err = testutil.RunGitFlow(t, dir, "feature", "finish", "stored-base-test")
	if err != nil {
		t.Fatalf("Failed to finish feature branch: %v\nOutput: %s", err, output)
	}

	// Get develop commit after merge
	developAfter, err := testutil.RunGit(t, dir, "rev-parse", "develop")
	if err != nil {
		t.Fatalf("Failed to get develop commit: %v", err)
	}

	// Get main commit after merge
	mainAfter, err := testutil.RunGit(t, dir, "rev-parse", "main")
	if err != nil {
		t.Fatalf("Failed to get main commit: %v", err)
	}

	// Verify main changed (received the merge)
	if mainBefore == mainAfter {
		t.Error("Expected main branch to change after merge (should have received the feature)")
	}

	// Verify develop also changed (should have been auto-updated from main)
	if developBefore == developAfter {
		t.Error("Expected develop branch to change after auto-update from main")
	}

	// Verify the feature file exists in main
	_, err = testutil.RunGit(t, dir, "checkout", "main")
	if err != nil {
		t.Fatalf("Failed to checkout main: %v", err)
	}
	content := testutil.ReadFile(t, dir, "feature.txt")
	if content != "feature content" {
		t.Errorf("Expected feature.txt content in main to be 'feature content', got '%s'", content)
	}
}

// TestFinishCleansUpBaseBranch tests that finish command cleans up base branch config when deleting branch.
// Steps:
// 1. Sets up a test repository and initializes git-flow with defaults
// 2. Creates a feature branch (which stores base config)
// 3. Adds changes and finishes the feature branch
// 4. Verifies the base branch config is cleaned up after branch deletion
func TestFinishCleansUpBaseBranch(t *testing.T) {
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Create feature branch
	output, err = testutil.RunGitFlow(t, dir, "feature", "start", "cleanup-test")
	if err != nil {
		t.Fatalf("Failed to start feature branch: %v\nOutput: %s", err, output)
	}

	// Verify base config exists
	_, err = testutil.RunGit(t, dir, "config", "--get", "gitflow.branch.feature/cleanup-test.base")
	if err != nil {
		t.Fatalf("Expected base config to exist after start: %v", err)
	}

	// Add a change
	testutil.WriteFile(t, dir, "cleanup.txt", "test content")
	_, err = testutil.RunGit(t, dir, "add", "cleanup.txt")
	if err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Add cleanup test file")
	if err != nil {
		t.Fatalf("Failed to commit file: %v", err)
	}

	// Finish the feature branch (should delete branch and clean up config)
	output, err = testutil.RunGitFlow(t, dir, "feature", "finish", "cleanup-test")
	if err != nil {
		t.Fatalf("Failed to finish feature branch: %v\nOutput: %s", err, output)
	}

	// Verify base config is cleaned up
	_, err = testutil.RunGit(t, dir, "config", "--get", "gitflow.branch.feature/cleanup-test.base")
	if err == nil {
		t.Error("Expected base config to be cleaned up after finish, but it still exists")
	}

	// Verify branch is deleted
	if testutil.BranchExists(t, dir, "feature/cleanup-test") {
		t.Error("Expected feature branch to be deleted after finish")
	}
}

// TestFinishBranchBehindRemote tests that finish fails when local branch is behind remote
func TestFinishBranchBehindRemote(t *testing.T) {
	// Setup test repo with remote
	dir, remoteDir := testutil.SetupTestRepoWithRemote(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer testutil.CleanupTestRepo(t, remoteDir)

	// Create a feature branch
	output, err := testutil.RunGitFlow(t, dir, "feature", "start", "behind-test")
	if err != nil {
		t.Fatalf("Failed to create feature branch: %v\nOutput: %s", err, output)
	}

	// Add a commit to the feature branch
	testutil.WriteFile(t, dir, "local.txt", "local content")
	_, err = testutil.RunGit(t, dir, "add", "local.txt")
	if err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Local commit")
	if err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Push the feature branch to remote
	_, err = testutil.RunGit(t, dir, "push", "-u", "origin", "feature/behind-test")
	if err != nil {
		t.Fatalf("Failed to push feature branch: %v", err)
	}

	// Clone the remote to simulate a teammate
	teammateDir, err := os.MkdirTemp("", "git-flow-test-teammate-*")
	if err != nil {
		t.Fatalf("Failed to create teammate directory: %v", err)
	}
	defer os.RemoveAll(teammateDir)

	_, err = testutil.RunGit(t, teammateDir, "clone", remoteDir, ".")
	if err != nil {
		t.Fatalf("Failed to clone remote: %v", err)
	}
	_, err = testutil.RunGit(t, teammateDir, "config", "user.name", "Teammate")
	if err != nil {
		t.Fatalf("Failed to configure teammate user name: %v", err)
	}
	_, err = testutil.RunGit(t, teammateDir, "config", "user.email", "teammate@example.com")
	if err != nil {
		t.Fatalf("Failed to configure teammate user email: %v", err)
	}

	// Teammate checks out the feature branch and adds a commit
	_, err = testutil.RunGit(t, teammateDir, "checkout", "feature/behind-test")
	if err != nil {
		t.Fatalf("Failed to checkout feature branch in teammate repo: %v", err)
	}
	testutil.WriteFile(t, teammateDir, "teammate.txt", "teammate content")
	_, err = testutil.RunGit(t, teammateDir, "add", "teammate.txt")
	if err != nil {
		t.Fatalf("Failed to add teammate file: %v", err)
	}
	_, err = testutil.RunGit(t, teammateDir, "commit", "-m", "Teammate commit")
	if err != nil {
		t.Fatalf("Failed to commit teammate file: %v", err)
	}
	_, err = testutil.RunGit(t, teammateDir, "push")
	if err != nil {
		t.Fatalf("Failed to push teammate commit: %v", err)
	}

	// Now back to original repo - local is behind remote
	// Try to finish with fetch enabled - should fail
	output, err = testutil.RunGitFlow(t, dir, "feature", "finish", "behind-test", "--fetch")
	if err == nil {
		t.Fatalf("Expected finish to fail when branch is behind remote, but it succeeded\nOutput: %s", output)
	}

	if !strings.Contains(output, "behind") {
		t.Errorf("Expected error message to mention 'behind', got: %s", output)
	}

	// Verify branch still exists (not deleted since finish failed)
	if !testutil.BranchExists(t, dir, "feature/behind-test") {
		t.Error("Expected feature branch to still exist after failed finish")
	}
}

// TestFinishBranchBehindRemoteWithNoCheck tests that --no-remote-check bypasses the check
func TestFinishBranchBehindRemoteWithNoCheck(t *testing.T) {
	// Setup test repo with remote
	dir, remoteDir := testutil.SetupTestRepoWithRemote(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer testutil.CleanupTestRepo(t, remoteDir)

	// Create a feature branch
	output, err := testutil.RunGitFlow(t, dir, "feature", "start", "nocheck-test")
	if err != nil {
		t.Fatalf("Failed to create feature branch: %v\nOutput: %s", err, output)
	}

	// Add a commit to the feature branch
	testutil.WriteFile(t, dir, "local.txt", "local content")
	_, err = testutil.RunGit(t, dir, "add", "local.txt")
	if err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Local commit")
	if err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Push the feature branch to remote
	_, err = testutil.RunGit(t, dir, "push", "-u", "origin", "feature/nocheck-test")
	if err != nil {
		t.Fatalf("Failed to push feature branch: %v", err)
	}

	// Clone the remote to simulate a teammate
	teammateDir, err := os.MkdirTemp("", "git-flow-test-teammate-*")
	if err != nil {
		t.Fatalf("Failed to create teammate directory: %v", err)
	}
	defer os.RemoveAll(teammateDir)

	_, err = testutil.RunGit(t, teammateDir, "clone", remoteDir, ".")
	if err != nil {
		t.Fatalf("Failed to clone remote: %v", err)
	}
	_, err = testutil.RunGit(t, teammateDir, "config", "user.name", "Teammate")
	if err != nil {
		t.Fatalf("Failed to configure teammate user name: %v", err)
	}
	_, err = testutil.RunGit(t, teammateDir, "config", "user.email", "teammate@example.com")
	if err != nil {
		t.Fatalf("Failed to configure teammate user email: %v", err)
	}

	// Teammate checks out the feature branch and adds a commit
	_, err = testutil.RunGit(t, teammateDir, "checkout", "feature/nocheck-test")
	if err != nil {
		t.Fatalf("Failed to checkout feature branch in teammate repo: %v", err)
	}
	testutil.WriteFile(t, teammateDir, "teammate.txt", "teammate content")
	_, err = testutil.RunGit(t, teammateDir, "add", "teammate.txt")
	if err != nil {
		t.Fatalf("Failed to add teammate file: %v", err)
	}
	_, err = testutil.RunGit(t, teammateDir, "commit", "-m", "Teammate commit")
	if err != nil {
		t.Fatalf("Failed to commit teammate file: %v", err)
	}
	_, err = testutil.RunGit(t, teammateDir, "push")
	if err != nil {
		t.Fatalf("Failed to push teammate commit: %v", err)
	}

	// Now back to original repo - local is behind remote
	// Finish with --no-remote-check should succeed
	output, err = testutil.RunGitFlow(t, dir, "feature", "finish", "nocheck-test", "--fetch", "--no-remote-check")
	if err != nil {
		t.Fatalf("Expected finish with --no-remote-check to succeed, but it failed: %v\nOutput: %s", err, output)
	}

	// Verify branch is deleted (finish succeeded)
	if testutil.BranchExists(t, dir, "feature/nocheck-test") {
		t.Error("Expected feature branch to be deleted after successful finish")
	}
}

// TestFinishBranchNoRemoteTracking tests finish works when no remote tracking exists
func TestFinishBranchNoRemoteTracking(t *testing.T) {
	// Setup test repo with remote (but we won't push the feature branch)
	dir, remoteDir := testutil.SetupTestRepoWithRemote(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer testutil.CleanupTestRepo(t, remoteDir)

	// Create a feature branch (local only, not pushed)
	output, err := testutil.RunGitFlow(t, dir, "feature", "start", "local-only")
	if err != nil {
		t.Fatalf("Failed to create feature branch: %v\nOutput: %s", err, output)
	}

	// Add a commit
	testutil.WriteFile(t, dir, "local.txt", "local content")
	_, err = testutil.RunGit(t, dir, "add", "local.txt")
	if err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Local commit")
	if err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Finish with fetch enabled - should succeed because no remote tracking branch exists
	output, err = testutil.RunGitFlow(t, dir, "feature", "finish", "local-only", "--fetch")
	if err != nil {
		t.Fatalf("Expected finish to succeed for local-only branch, but it failed: %v\nOutput: %s", err, output)
	}

	// Verify branch is deleted
	if testutil.BranchExists(t, dir, "feature/local-only") {
		t.Error("Expected feature branch to be deleted after finish")
	}
}

// TestFinishBranchUpToDateWithRemote tests finish works when branch is up to date with remote
func TestFinishBranchUpToDateWithRemote(t *testing.T) {
	// Setup test repo with remote
	dir, remoteDir := testutil.SetupTestRepoWithRemote(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer testutil.CleanupTestRepo(t, remoteDir)

	// Create a feature branch
	output, err := testutil.RunGitFlow(t, dir, "feature", "start", "uptodate-test")
	if err != nil {
		t.Fatalf("Failed to create feature branch: %v\nOutput: %s", err, output)
	}

	// Add a commit
	testutil.WriteFile(t, dir, "local.txt", "local content")
	_, err = testutil.RunGit(t, dir, "add", "local.txt")
	if err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Local commit")
	if err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Push to remote (so we're up to date)
	_, err = testutil.RunGit(t, dir, "push", "-u", "origin", "feature/uptodate-test")
	if err != nil {
		t.Fatalf("Failed to push feature branch: %v", err)
	}

	// Finish with fetch enabled - should succeed because we're up to date
	output, err = testutil.RunGitFlow(t, dir, "feature", "finish", "uptodate-test", "--fetch")
	if err != nil {
		t.Fatalf("Expected finish to succeed when up to date with remote, but it failed: %v\nOutput: %s", err, output)
	}

	// Verify branch is deleted
	if testutil.BranchExists(t, dir, "feature/uptodate-test") {
		t.Error("Expected feature branch to be deleted after finish")
	}
}
