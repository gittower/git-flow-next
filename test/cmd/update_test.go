package cmd_test

import (
	"os"
	"testing"

	"github.com/gittower/git-flow-next/internal/config"
	"github.com/gittower/git-flow-next/internal/git"
	"github.com/gittower/git-flow-next/test/testutil"
	"github.com/stretchr/testify/assert"
)

// TestUpdateFeatureBranch tests the basic feature branch update functionality.
// Steps:
// 1. Sets up a test repository and initializes git-flow
// 2. Creates a feature branch
// 3. Makes changes in the develop branch
// 4. Updates the feature branch
// 5. Verifies the changes from develop are in the feature branch
func TestUpdateFeatureBranch(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	// Initialize git-flow with branch creation
	if _, err := testutil.RunGitFlow(t, dir, "init", "--defaults"); err != nil {
		t.Fatal(err)
	}

	// Verify git-flow is initialized
	initialized, err := config.IsInitialized()
	if err != nil {
		t.Fatal(err)
	}
	if !initialized {
		t.Fatal("git-flow is not initialized")
	}

	// Create a feature branch
	if _, err := testutil.RunGitFlow(t, dir, "feature", "start", "test-feature"); err != nil {
		t.Fatal(err)
	}

	// Make changes in develop branch
	if err := git.Checkout("develop"); err != nil {
		t.Fatal(err)
	}
	if err := testutil.WriteFile(t, dir, "develop-change.txt", "develop change"); err != nil {
		t.Fatal(err)
	}
	if _, err := testutil.RunGit(t, dir, "add", "develop-change.txt"); err != nil {
		t.Fatal(err)
	}
	if _, err := testutil.RunGit(t, dir, "commit", "-m", "Add develop change"); err != nil {
		t.Fatal(err)
	}

	// Update feature branch
	if _, err := testutil.RunGitFlow(t, dir, "update", "feature/test-feature"); err != nil {
		t.Fatal(err)
	}

	// Verify changes are in feature branch
	if err := git.Checkout("feature/test-feature"); err != nil {
		t.Fatal(err)
	}
	_, err = os.Stat("develop-change.txt")
	assert.NoError(t, err, "develop changes should be in feature branch")
}

// TestUpdateWithMergeConflict tests the behavior when updating a branch with merge conflicts.
// Steps:
// 1. Sets up a test repository and initializes git-flow
// 2. Creates a feature branch
// 3. Makes conflicting changes in both feature and develop branches
// 4. Attempts to update the feature branch
// 5. Verifies the operation fails with merge conflict
func TestUpdateWithMergeConflict(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	// Initialize git-flow with branch creation
	if _, err := testutil.RunGitFlow(t, dir, "init", "--defaults"); err != nil {
		t.Fatal(err)
	}

	// Verify git-flow is initialized
	initialized, err := config.IsInitialized()
	if err != nil {
		t.Fatal(err)
	}
	if !initialized {
		t.Fatal("git-flow is not initialized")
	}

	// Create a feature branch
	if _, err := testutil.RunGitFlow(t, dir, "feature", "start", "test-feature"); err != nil {
		t.Fatal(err)
	}

	// Make conflicting changes in both branches
	if err := git.Checkout("develop"); err != nil {
		t.Fatal(err)
	}
	if err := testutil.WriteFile(t, dir, "conflict.txt", "develop version"); err != nil {
		t.Fatal(err)
	}
	if _, err := testutil.RunGit(t, dir, "add", "conflict.txt"); err != nil {
		t.Fatal(err)
	}
	if _, err := testutil.RunGit(t, dir, "commit", "-m", "Add develop version"); err != nil {
		t.Fatal(err)
	}

	if err := git.Checkout("feature/test-feature"); err != nil {
		t.Fatal(err)
	}
	if err := testutil.WriteFile(t, dir, "conflict.txt", "feature version"); err != nil {
		t.Fatal(err)
	}
	if _, err := testutil.RunGit(t, dir, "add", "conflict.txt"); err != nil {
		t.Fatal(err)
	}
	if _, err := testutil.RunGit(t, dir, "commit", "-m", "Add feature version"); err != nil {
		t.Fatal(err)
	}

	// Attempt to update feature branch
	output, err := testutil.RunGitFlow(t, dir, "update", "feature/test-feature")
	assert.Error(t, err, "should fail due to merge conflict")
	assert.Contains(t, output, "unresolved conflicts")
}

// TestUpdateNonExistentBranch tests the behavior when attempting to update a non-existent branch.
// Steps:
// 1. Sets up a test repository and initializes git-flow
// 2. Attempts to update a non-existent branch
// 3. Verifies the operation fails with appropriate error
func TestUpdateNonExistentBranch(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	// Initialize git-flow with branch creation
	if _, err := testutil.RunGitFlow(t, dir, "init", "--defaults"); err != nil {
		t.Fatal(err)
	}

	// Verify git-flow is initialized
	initialized, err := config.IsInitialized()
	if err != nil {
		t.Fatal(err)
	}
	if !initialized {
		t.Fatal("git-flow is not initialized")
	}

	// Try to update non-existent branch
	output, err := testutil.RunGitFlow(t, dir, "update", "feature/non-existent")
	assert.Error(t, err)
	assert.Contains(t, output, "does not exist")
}

// TestUpdateCurrentBranch tests updating the current branch without specifying its name.
// Steps:
// 1. Sets up a test repository and initializes git-flow
// 2. Creates a feature branch
// 3. Makes changes in the develop branch
// 4. Switches to the feature branch
// 5. Updates the branch without specifying its name
// 6. Verifies the changes from develop are in the feature branch
func TestUpdateCurrentBranch(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	// Initialize git-flow with branch creation
	if _, err := testutil.RunGitFlow(t, dir, "init", "--defaults"); err != nil {
		t.Fatal(err)
	}

	// Verify git-flow is initialized
	initialized, err := config.IsInitialized()
	if err != nil {
		t.Fatal(err)
	}
	if !initialized {
		t.Fatal("git-flow is not initialized")
	}

	// Create a feature branch
	if _, err := testutil.RunGitFlow(t, dir, "feature", "start", "test-feature"); err != nil {
		t.Fatal(err)
	}

	// Make changes in develop branch
	if err := git.Checkout("develop"); err != nil {
		t.Fatal(err)
	}
	if err := testutil.WriteFile(t, dir, "develop-change.txt", "develop change"); err != nil {
		t.Fatal(err)
	}
	if _, err := testutil.RunGit(t, dir, "add", "develop-change.txt"); err != nil {
		t.Fatal(err)
	}
	if _, err := testutil.RunGit(t, dir, "commit", "-m", "Add develop change"); err != nil {
		t.Fatal(err)
	}

	// Switch to feature branch and update without specifying branch name
	if err := git.Checkout("feature/test-feature"); err != nil {
		t.Fatal(err)
	}
	if _, err := testutil.RunGitFlow(t, dir, "update"); err != nil {
		t.Fatal(err)
	}

	// Verify changes are in feature branch
	_, err = os.Stat("develop-change.txt")
	assert.NoError(t, err, "develop changes should be in feature branch")
}

// TestUpdateBaseBranch tests updating a base branch (develop) with changes from main.
// Steps:
// 1. Sets up a test repository and initializes git-flow
// 2. Makes changes in the main branch
// 3. Updates the develop branch with changes from main
// 4. Verifies the changes from main are in develop
// 5. Verifies we're still on the develop branch
func TestUpdateBaseBranch(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	// Create initial commit and rename master to main
	if err := testutil.WriteFile(t, dir, "initial.txt", "initial content"); err != nil {
		t.Fatal(err)
	}
	if _, err := testutil.RunGit(t, dir, "add", "initial.txt"); err != nil {
		t.Fatal(err)
	}
	if _, err := testutil.RunGit(t, dir, "commit", "-m", "Initial commit"); err != nil {
		t.Fatal(err)
	}
	if _, err := testutil.RunGit(t, dir, "branch", "-M", "main"); err != nil {
		t.Fatal(err)
	}

	// Initialize git-flow with default configuration and create branches
	if _, err := testutil.RunGitFlow(t, dir, "init", "--defaults"); err != nil {
		t.Fatal(err)
	}

	// Verify git-flow is initialized
	initialized, err := config.IsInitialized()
	if err != nil {
		t.Fatal(err)
	}
	if !initialized {
		t.Fatal("git-flow is not initialized")
	}

	// Make changes in main branch
	if err := git.Checkout("main"); err != nil {
		t.Fatal(err)
	}
	if err := testutil.WriteFile(t, dir, "main-change.txt", "main branch change"); err != nil {
		t.Fatal(err)
	}
	if _, err := testutil.RunGit(t, dir, "add", "main-change.txt"); err != nil {
		t.Fatal(err)
	}
	if _, err := testutil.RunGit(t, dir, "commit", "-m", "Add main branch change"); err != nil {
		t.Fatal(err)
	}

	// Switch to develop branch
	if err := git.Checkout("develop"); err != nil {
		t.Fatal(err)
	}

	// Update develop branch with changes from main
	if _, err := testutil.RunGitFlow(t, dir, "update", "develop"); err != nil {
		t.Fatal(err)
	}

	// Verify changes from main are in develop
	_, err = os.Stat("main-change.txt")
	assert.NoError(t, err, "main branch changes should be in develop branch")

	// Verify we're still on develop branch
	currentBranch, err := git.GetCurrentBranch()
	assert.NoError(t, err)
	assert.Equal(t, "develop", currentBranch, "should still be on develop branch")
}
