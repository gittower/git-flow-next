package cmd_test

import (
	"os"
	"testing"

	"github.com/gittower/git-flow-next/config"
	"github.com/gittower/git-flow-next/git"
	"github.com/gittower/git-flow-next/test/testutil"
	"github.com/stretchr/testify/assert"
)

func TestUpdateFeatureBranch(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	// Initialize git-flow with branch creation
	if _, err := testutil.RunGitFlow(t, dir, "init", "-d", "-c"); err != nil {
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

func TestUpdateWithMergeConflict(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	// Initialize git-flow with branch creation
	if _, err := testutil.RunGitFlow(t, dir, "init", "-d", "-c"); err != nil {
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
	assert.Contains(t, output, "merge conflict")
}

func TestUpdateNonExistentBranch(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	// Initialize git-flow with branch creation
	if _, err := testutil.RunGitFlow(t, dir, "init", "-d", "-c"); err != nil {
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
	assert.Contains(t, output, "branch not found")
}

func TestUpdateCurrentBranch(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	// Initialize git-flow with branch creation
	if _, err := testutil.RunGitFlow(t, dir, "init", "-d", "-c"); err != nil {
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
