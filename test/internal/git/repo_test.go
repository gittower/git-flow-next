package git_test

import (
	"os"
	"testing"

	"github.com/gittower/git-flow-next/internal/git"
	"github.com/gittower/git-flow-next/test/testutil"
)

// withGitRepo changes to the provided directory, runs the testFunc, and changes back to the original directory after the test function is done
func withGitRepo(t *testing.T, dir string, testFunc func()) {
	// Save current directory
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}

	// Change to the test directory
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Failed to change to test directory: %v", err)
	}

	// Ensure we change back to the original directory when done
	defer func() {
		if err := os.Chdir(oldDir); err != nil {
			t.Fatalf("Failed to change back to original directory: %v", err)
		}
	}()

	// Run the test function
	testFunc()
}

func TestRemoteBranchExists_ExistingBranch(t *testing.T) {
	// Setup test repo
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Add a remote
	remoteDir, err := testutil.AddRemote(t, dir, "origin", false) // Don't push all branches yet
	if err != nil {
		t.Fatalf("Failed to add remote: %v", err)
	}
	defer testutil.CleanupTestRepo(t, remoteDir)

	// Create and push a feature branch
	_, err = testutil.RunGit(t, dir, "checkout", "-b", "feature/test")
	if err != nil {
		t.Fatalf("Failed to create branch: %v", err)
	}

	// Create a test file and commit it
	testutil.WriteFile(t, dir, "test.txt", "test content")
	_, err = testutil.RunGit(t, dir, "add", "test.txt")
	if err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "test commit")
	if err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Push the branch with --set-upstream
	_, err = testutil.RunGit(t, dir, "push", "-u", "origin", "feature/test")
	if err != nil {
		t.Fatalf("Failed to push branch: %v", err)
	}

	// Fetch to update remote tracking branches
	_, err = testutil.RunGit(t, dir, "fetch", "origin")
	if err != nil {
		t.Fatalf("Failed to fetch from remote: %v", err)
	}

	// Use our helper to change to the test directory and run the test
	withGitRepo(t, dir, func() {
		// Check if the branch exists
		exists := git.RemoteBranchExists("origin", "feature/test")
		if !exists {
			t.Error("Expected RemoteBranchExists to return true for existing branch")
		}
	})
}

func TestRemoteBranchExists_NonExistentBranch(t *testing.T) {
	// Setup test repo
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Add a remote
	remoteDir, err := testutil.AddRemote(t, dir, "origin", true)
	if err != nil {
		t.Fatalf("Failed to add remote: %v", err)
	}
	defer testutil.CleanupTestRepo(t, remoteDir)

	// Fetch to update remote tracking branches
	_, err = testutil.RunGit(t, dir, "fetch", "origin")
	if err != nil {
		t.Fatalf("Failed to fetch from remote: %v", err)
	}

	// Use our helper to change to the test directory and run the test
	withGitRepo(t, dir, func() {
		// Check if a non-existent branch exists
		exists := git.RemoteBranchExists("origin", "feature/non-existent")
		if exists {
			t.Error("Expected RemoteBranchExists to return false for non-existent branch")
		}
	})
}

func TestDeleteNonExistentRemoteBranch(t *testing.T) {
	// Setup test repo
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Add a remote
	remoteDir, err := testutil.AddRemote(t, dir, "origin", true)
	if err != nil {
		t.Fatalf("Failed to add remote: %v", err)
	}
	defer testutil.CleanupTestRepo(t, remoteDir)

	// Use our helper to change to the test directory and run the test
	withGitRepo(t, dir, func() {
		// Try to delete a non-existent branch
		err = git.DeleteRemoteBranch("origin", "feature/non-existent")
		if err == nil {
			t.Error("Expected an error when deleting non-existent remote branch, got nil")
		}
	})
}

func TestDeleteExistingRemoteBranch(t *testing.T) {
	// Setup test repo
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Add a remote
	remoteDir, err := testutil.AddRemote(t, dir, "origin", true)
	if err != nil {
		t.Fatalf("Failed to add remote: %v", err)
	}
	defer testutil.CleanupTestRepo(t, remoteDir)

	// Create and push a feature branch
	_, err = testutil.RunGit(t, dir, "checkout", "-b", "feature/test")
	if err != nil {
		t.Fatalf("Failed to create branch: %v", err)
	}

	// Create a test file and commit it
	testutil.WriteFile(t, dir, "test.txt", "test content")
	_, err = testutil.RunGit(t, dir, "add", "test.txt")
	if err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "test commit")
	if err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Push the branch
	_, err = testutil.RunGit(t, dir, "push", "origin", "feature/test")
	if err != nil {
		t.Fatalf("Failed to push branch: %v", err)
	}

	// Fetch to update remote tracking branches
	_, err = testutil.RunGit(t, dir, "fetch", "origin")
	if err != nil {
		t.Fatalf("Failed to fetch from remote: %v", err)
	}

	// Use our helper to change to the test directory and run the test
	withGitRepo(t, dir, func() {
		// Delete the remote branch
		err = git.DeleteRemoteBranch("origin", "feature/test")
		if err != nil {
			t.Errorf("Expected no error when deleting existing remote branch, got: %v", err)
		}
	})

	// Fetch from remote to update refs
	_, err = testutil.RunGit(t, dir, "fetch", "origin", "--prune")
	if err != nil {
		t.Fatalf("Failed to fetch from remote: %v", err)
	}

	// Verify branch was deleted by checking the remote tracking branch
	_, err = testutil.RunGit(t, dir, "rev-parse", "--verify", "refs/remotes/origin/feature/test")
	if err == nil {
		t.Error("Expected remote tracking branch to be deleted")
	}
}

func TestDeleteBranchFromNonExistentRemote(t *testing.T) {
	// Setup test repo
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Use our helper to change to the test directory and run the test
	withGitRepo(t, dir, func() {
		// Try to delete a branch from a non-existent remote
		err := git.DeleteRemoteBranch("non-existent-remote", "feature/test")
		if err == nil {
			t.Error("Expected an error when deleting from non-existent remote, got nil")
		}
	})
}
