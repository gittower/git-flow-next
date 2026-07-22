package cmd_test

import (
	"strings"
	"testing"

	"github.com/gittower/git-flow-next/internal/errors"
	"github.com/gittower/git-flow-next/test/testutil"
)

// TestRenameFeature tests the basic functionality of renaming a feature branch.
// Steps:
// 1. Sets up a test repository and initializes git-flow
// 2. Creates a feature branch and adds some changes
// 3. Renames the feature branch
// 4. Verifies the old branch is deleted
// 5. Verifies the new branch exists with the same content
func TestRenameFeature(t *testing.T) {
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Create a feature branch
	output, err = testutil.RunGitFlow(t, dir, "feature", "start", "old-feature")
	if err != nil {
		t.Fatalf("Failed to create feature branch: %v\nOutput: %s", err, output)
	}

	// Add some changes
	testutil.WriteFile(t, dir, "feature.txt", "feature content")
	_, err = testutil.RunGit(t, dir, "add", "feature.txt")
	if err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Add feature file")
	if err != nil {
		t.Fatalf("Failed to commit file: %v", err)
	}

	// Rename the feature branch
	output, err = testutil.RunGitFlow(t, dir, "feature", "rename", "old-feature", "new-feature")
	if err != nil {
		t.Fatalf("Failed to rename feature branch: %v\nOutput: %s", err, output)
	}

	// Verify old branch doesn't exist
	if testutil.BranchExists(t, dir, "feature/old-feature") {
		t.Error("Expected old feature branch to be deleted")
	}

	// Verify new branch exists
	if !testutil.BranchExists(t, dir, "feature/new-feature") {
		t.Error("Expected new feature branch to exist")
	}

	// Verify the changes are in the new branch
	_, err = testutil.RunGit(t, dir, "checkout", "feature/new-feature")
	if err != nil {
		t.Fatalf("Failed to checkout new feature branch: %v", err)
	}

	content, err := testutil.RunGit(t, dir, "--no-pager", "show", "HEAD:feature.txt")
	if err != nil {
		t.Fatalf("Failed to read file content: %v", err)
	}
	if content != "feature content" {
		t.Errorf("Expected feature.txt content to be 'feature content', got '%s'", content)
	}
}

// TestRenameCurrentFeature tests renaming the currently checked out feature branch.
// Steps:
// 1. Sets up a test repository and initializes git-flow
// 2. Creates and checks out a feature branch
// 3. Renames the current feature branch
// 4. Verifies we're still on the renamed branch
func TestRenameCurrentFeature(t *testing.T) {
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Create and checkout a feature branch
	output, err = testutil.RunGitFlow(t, dir, "feature", "start", "current-feature")
	if err != nil {
		t.Fatalf("Failed to create feature branch: %v\nOutput: %s", err, output)
	}

	// Rename the current feature branch
	output, err = testutil.RunGitFlow(t, dir, "feature", "rename", "current-feature", "renamed-feature")
	if err != nil {
		t.Fatalf("Failed to rename current feature branch: %v\nOutput: %s", err, output)
	}

	// Verify we're on the renamed branch
	currentBranch := testutil.GetCurrentBranch(t, dir)
	if currentBranch != "feature/renamed-feature" {
		t.Errorf("Expected to be on feature/renamed-feature branch, got %s", currentBranch)
	}
}

// TestRenameNonExistentFeature tests the behavior when attempting to rename a non-existent feature branch.
// Steps:
// 1. Sets up a test repository and initializes git-flow
// 2. Attempts to rename a non-existent feature branch
// 3. Verifies the operation fails with appropriate error
func TestRenameNonExistentFeature(t *testing.T) {
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Try to rename non-existent branch
	output, err = testutil.RunGitFlow(t, dir, "feature", "rename", "nonexistent", "new-name")
	if err == nil {
		t.Fatal("Expected rename to fail for non-existent branch")
	}
}

// TestRenameToExistingFeature tests the behavior when attempting to rename a feature branch to a name that already exists.
// Steps:
// 1. Sets up a test repository and initializes git-flow
// 2. Creates two feature branches
// 3. Attempts to rename the first branch to the name of the second branch
// 4. Verifies the operation fails with appropriate error
func TestRenameToExistingFeature(t *testing.T) {
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Create first feature branch
	output, err = testutil.RunGitFlow(t, dir, "feature", "start", "first-feature")
	if err != nil {
		t.Fatalf("Failed to create first feature branch: %v\nOutput: %s", err, output)
	}

	// Create second feature branch
	output, err = testutil.RunGitFlow(t, dir, "feature", "start", "second-feature")
	if err != nil {
		t.Fatalf("Failed to create second feature branch: %v\nOutput: %s", err, output)
	}

	// Try to rename first branch to second branch name
	output, err = testutil.RunGitFlow(t, dir, "feature", "rename", "first-feature", "second-feature")
	if err == nil {
		t.Fatal("Expected rename to fail when target name already exists")
	}
}

// TestRenameWithInvalidBranchType tests the behavior when attempting to rename a branch with an invalid type.
// Steps:
// 1. Sets up a test repository and initializes git-flow
// 2. Attempts to rename a branch with an invalid type
// 3. Verifies the operation fails with appropriate error
func TestRenameWithInvalidBranchType(t *testing.T) {
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Try to rename with invalid branch type
	output, err = testutil.RunGitFlow(t, dir, "invalid", "rename", "old-name", "new-name")
	if err == nil {
		t.Fatal("Expected rename to fail with invalid branch type")
	}
}

// TestRenameWithoutInitialization tests the rename command in a repository where
// git-flow has not been initialized.
// Steps:
//  1. Sets up a plain Git repository without running git flow init
//  2. Attempts to rename a feature branch
//  3. Verifies the command fails with the "not initialized" error and exit code,
//     rather than a misleading "branch does not exist" error
func TestRenameWithoutInitialization(t *testing.T) {
	// Setup: plain repo, git-flow NOT initialized
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Attempt to rename a feature branch without initialization
	output, err := testutil.RunGitFlow(t, dir, "feature", "rename", "foo", "bar")
	if err == nil {
		t.Fatal("Expected rename to fail without git-flow initialization, but it succeeded")
	}

	// Check exit code
	if exitErr, ok := err.(*testutil.ExitError); ok {
		if exitErr.ExitCode != int(errors.ExitCodeNotInitialized) {
			t.Errorf("Expected exit code %d, got %d", errors.ExitCodeNotInitialized, exitErr.ExitCode)
		}
	} else {
		t.Error("Expected ExitError")
	}

	// Verify error message is the "not initialized" message, not "does not exist"
	if !strings.Contains(output, "Error: git flow is not initialized") {
		t.Errorf("Expected 'not initialized' error, got: %s", output)
	}
	if strings.Contains(output, "does not exist") {
		t.Errorf("Expected no misleading 'does not exist' error, got: %s", output)
	}

	// Verify neither branch was created and git-flow is still not initialized
	if testutil.BranchExists(t, dir, "feature/foo") {
		t.Error("Expected no branch to be created, but 'feature/foo' exists")
	}
	if testutil.BranchExists(t, dir, "feature/bar") {
		t.Error("Expected no branch to be created, but 'feature/bar' exists")
	}
	if _, err := testutil.RunGit(t, dir, "config", "--get", "gitflow.version"); err == nil {
		t.Error("Expected git-flow to still not be initialized after failed command")
	}
}
