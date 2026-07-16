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

// TestGetTrackingBranch tests that the tracking branch is correctly identified.
// Steps:
// 1. Sets up a test repository with a remote
// 2. Creates a feature branch and pushes it with tracking
// 3. Calls GetTrackingBranch on the feature branch
// 4. Verifies the correct remote tracking branch is returned (origin/feature/test)
func TestGetTrackingBranch(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Add a remote
	remoteDir, err := testutil.AddRemote(t, dir, "origin", false)
	if err != nil {
		t.Fatalf("Failed to add remote: %v", err)
	}
	defer testutil.CleanupTestRepo(t, remoteDir)

	// Create and push a feature branch with tracking
	_, err = testutil.RunGit(t, dir, "checkout", "-b", "feature/test")
	if err != nil {
		t.Fatalf("Failed to create branch: %v", err)
	}

	testutil.WriteFile(t, dir, "test.txt", "test content")
	_, err = testutil.RunGit(t, dir, "add", "test.txt")
	if err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "test commit")
	if err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Push with --set-upstream to establish tracking
	_, err = testutil.RunGit(t, dir, "push", "--set-upstream", "origin", "feature/test")
	if err != nil {
		t.Fatalf("Failed to push branch: %v", err)
	}

	withGitRepo(t, dir, func() {
		trackingBranch, err := git.GetTrackingBranch("feature/test")
		if err != nil {
			t.Fatalf("GetTrackingBranch returned unexpected error: %v", err)
		}
		expected := "origin/feature/test"
		if trackingBranch != expected {
			t.Errorf("Expected tracking branch '%s', got '%s'", expected, trackingBranch)
		}
	})
}

// TestGetTrackingBranchNoTracking tests behavior when branch has no tracking branch.
// Steps:
// 1. Sets up a test repository
// 2. Creates a local-only branch without pushing
// 3. Calls GetTrackingBranch on the local branch
// 4. Verifies an appropriate error is returned
func TestGetTrackingBranchNoTracking(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Create a local-only branch (no remote, no tracking)
	_, err := testutil.RunGit(t, dir, "checkout", "-b", "feature/local-only")
	if err != nil {
		t.Fatalf("Failed to create branch: %v", err)
	}

	withGitRepo(t, dir, func() {
		_, err := git.GetTrackingBranch("feature/local-only")
		if err == nil {
			t.Error("Expected error for branch without tracking, got nil")
		}
	})
}

// TestCompareBranchWithRemoteEqual tests comparison when local equals remote.
// Steps:
// 1. Sets up a test repository with a remote
// 2. Creates and pushes a feature branch with tracking
// 3. Calls CompareBranchWithRemote (no changes made after push)
// 4. Verifies SyncStatusEqual is returned with 0 commits difference
func TestCompareBranchWithRemoteEqual(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	remoteDir, err := testutil.AddRemote(t, dir, "origin", false)
	if err != nil {
		t.Fatalf("Failed to add remote: %v", err)
	}
	defer testutil.CleanupTestRepo(t, remoteDir)

	// Create and push a feature branch
	_, err = testutil.RunGit(t, dir, "checkout", "-b", "feature/test")
	if err != nil {
		t.Fatalf("Failed to create branch: %v", err)
	}

	testutil.WriteFile(t, dir, "test.txt", "test content")
	_, err = testutil.RunGit(t, dir, "add", "test.txt")
	if err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "test commit")
	if err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	_, err = testutil.RunGit(t, dir, "push", "--set-upstream", "origin", "feature/test")
	if err != nil {
		t.Fatalf("Failed to push branch: %v", err)
	}

	withGitRepo(t, dir, func() {
		status, count, err := git.CompareBranchWithRemote("feature/test")
		if err != nil {
			t.Fatalf("CompareBranchWithRemote returned unexpected error: %v", err)
		}
		if status != git.SyncStatusEqual {
			t.Errorf("Expected status SyncStatusEqual, got %s", status)
		}
		if count != 0 {
			t.Errorf("Expected count 0, got %d", count)
		}
	})
}

// TestCompareBranchWithRemoteAhead tests comparison when local is ahead of remote.
// Steps:
// 1. Sets up a test repository with a remote
// 2. Creates and pushes a feature branch with tracking
// 3. Adds a new commit locally without pushing
// 4. Calls CompareBranchWithRemote
// 5. Verifies SyncStatusAhead is returned with correct commit count
func TestCompareBranchWithRemoteAhead(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	remoteDir, err := testutil.AddRemote(t, dir, "origin", false)
	if err != nil {
		t.Fatalf("Failed to add remote: %v", err)
	}
	defer testutil.CleanupTestRepo(t, remoteDir)

	// Create and push a feature branch
	_, err = testutil.RunGit(t, dir, "checkout", "-b", "feature/test")
	if err != nil {
		t.Fatalf("Failed to create branch: %v", err)
	}

	testutil.WriteFile(t, dir, "test.txt", "test content")
	_, err = testutil.RunGit(t, dir, "add", "test.txt")
	if err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "test commit")
	if err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	_, err = testutil.RunGit(t, dir, "push", "--set-upstream", "origin", "feature/test")
	if err != nil {
		t.Fatalf("Failed to push branch: %v", err)
	}

	// Add another commit locally without pushing
	testutil.WriteFile(t, dir, "local.txt", "local content")
	_, err = testutil.RunGit(t, dir, "add", "local.txt")
	if err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "local commit")
	if err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	withGitRepo(t, dir, func() {
		status, count, err := git.CompareBranchWithRemote("feature/test")
		if err != nil {
			t.Fatalf("CompareBranchWithRemote returned unexpected error: %v", err)
		}
		if status != git.SyncStatusAhead {
			t.Errorf("Expected status SyncStatusAhead, got %s", status)
		}
		if count != 1 {
			t.Errorf("Expected count 1, got %d", count)
		}
	})
}

// TestCompareBranchWithRemoteBehind tests comparison when local is behind remote.
// Steps:
// 1. Sets up a test repository with a remote
// 2. Creates and pushes a feature branch with tracking
// 3. Clones to second repo, adds commit, pushes to remote
// 4. Fetches in original repo to update remote refs
// 5. Calls CompareBranchWithRemote on original repo
// 6. Verifies SyncStatusBehind is returned with correct commit count
func TestCompareBranchWithRemoteBehind(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	remoteDir, err := testutil.AddRemote(t, dir, "origin", false)
	if err != nil {
		t.Fatalf("Failed to add remote: %v", err)
	}
	defer testutil.CleanupTestRepo(t, remoteDir)

	// Create and push a feature branch
	_, err = testutil.RunGit(t, dir, "checkout", "-b", "feature/test")
	if err != nil {
		t.Fatalf("Failed to create branch: %v", err)
	}

	testutil.WriteFile(t, dir, "test.txt", "test content")
	_, err = testutil.RunGit(t, dir, "add", "test.txt")
	if err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "test commit")
	if err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	_, err = testutil.RunGit(t, dir, "push", "--set-upstream", "origin", "feature/test")
	if err != nil {
		t.Fatalf("Failed to push branch: %v", err)
	}

	// Clone to a second working copy and make changes
	secondDir := t.TempDir()
	_, err = testutil.RunGit(t, secondDir, "clone", remoteDir, ".")
	if err != nil {
		t.Fatalf("Failed to clone: %v", err)
	}
	testutil.ConfigureGitIdentity(t, secondDir)

	_, err = testutil.RunGit(t, secondDir, "checkout", "feature/test")
	if err != nil {
		t.Fatalf("Failed to checkout feature branch in second repo: %v", err)
	}

	testutil.WriteFile(t, secondDir, "remote-change.txt", "remote content")
	_, err = testutil.RunGit(t, secondDir, "add", "remote-change.txt")
	if err != nil {
		t.Fatalf("Failed to add file in second repo: %v", err)
	}
	_, err = testutil.RunGit(t, secondDir, "commit", "-m", "remote commit")
	if err != nil {
		t.Fatalf("Failed to commit in second repo: %v", err)
	}
	_, err = testutil.RunGit(t, secondDir, "push", "origin", "feature/test")
	if err != nil {
		t.Fatalf("Failed to push from second repo: %v", err)
	}

	// Fetch in original repo to update remote refs
	_, err = testutil.RunGit(t, dir, "fetch", "origin")
	if err != nil {
		t.Fatalf("Failed to fetch: %v", err)
	}

	withGitRepo(t, dir, func() {
		status, count, err := git.CompareBranchWithRemote("feature/test")
		if err != nil {
			t.Fatalf("CompareBranchWithRemote returned unexpected error: %v", err)
		}
		if status != git.SyncStatusBehind {
			t.Errorf("Expected status SyncStatusBehind, got %s", status)
		}
		if count != 1 {
			t.Errorf("Expected count 1, got %d", count)
		}
	})
}

// TestCompareBranchWithRemoteDiverged tests comparison when local and remote have diverged.
// Steps:
// 1. Sets up a test repository with a remote
// 2. Creates and pushes a feature branch with tracking
// 3. Clones to second repo, adds commit, pushes to remote
// 4. Adds different commit locally (without fetching first)
// 5. Fetches in original repo to update remote refs
// 6. Calls CompareBranchWithRemote
// 7. Verifies SyncStatusDiverged is returned
func TestCompareBranchWithRemoteDiverged(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	remoteDir, err := testutil.AddRemote(t, dir, "origin", false)
	if err != nil {
		t.Fatalf("Failed to add remote: %v", err)
	}
	defer testutil.CleanupTestRepo(t, remoteDir)

	// Create and push a feature branch
	_, err = testutil.RunGit(t, dir, "checkout", "-b", "feature/test")
	if err != nil {
		t.Fatalf("Failed to create branch: %v", err)
	}

	testutil.WriteFile(t, dir, "test.txt", "test content")
	_, err = testutil.RunGit(t, dir, "add", "test.txt")
	if err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "test commit")
	if err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	_, err = testutil.RunGit(t, dir, "push", "--set-upstream", "origin", "feature/test")
	if err != nil {
		t.Fatalf("Failed to push branch: %v", err)
	}

	// Clone to a second working copy and make changes
	secondDir := t.TempDir()
	_, err = testutil.RunGit(t, secondDir, "clone", remoteDir, ".")
	if err != nil {
		t.Fatalf("Failed to clone: %v", err)
	}
	testutil.ConfigureGitIdentity(t, secondDir)

	_, err = testutil.RunGit(t, secondDir, "checkout", "feature/test")
	if err != nil {
		t.Fatalf("Failed to checkout feature branch in second repo: %v", err)
	}

	testutil.WriteFile(t, secondDir, "remote-change.txt", "remote content")
	_, err = testutil.RunGit(t, secondDir, "add", "remote-change.txt")
	if err != nil {
		t.Fatalf("Failed to add file in second repo: %v", err)
	}
	_, err = testutil.RunGit(t, secondDir, "commit", "-m", "remote commit")
	if err != nil {
		t.Fatalf("Failed to commit in second repo: %v", err)
	}
	_, err = testutil.RunGit(t, secondDir, "push", "origin", "feature/test")
	if err != nil {
		t.Fatalf("Failed to push from second repo: %v", err)
	}

	// Add local commit (before fetching - creates divergence)
	testutil.WriteFile(t, dir, "local-change.txt", "local content")
	_, err = testutil.RunGit(t, dir, "add", "local-change.txt")
	if err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "local commit")
	if err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Now fetch to update remote refs (creates divergence)
	_, err = testutil.RunGit(t, dir, "fetch", "origin")
	if err != nil {
		t.Fatalf("Failed to fetch: %v", err)
	}

	withGitRepo(t, dir, func() {
		status, count, err := git.CompareBranchWithRemote("feature/test")
		if err != nil {
			t.Fatalf("CompareBranchWithRemote returned unexpected error: %v", err)
		}
		if status != git.SyncStatusDiverged {
			t.Errorf("Expected status SyncStatusDiverged, got %s", status)
		}
		if count != 2 { // 1 ahead + 1 behind = 2 total
			t.Errorf("Expected count 2 (1 ahead + 1 behind), got %d", count)
		}
	})
}

// TestFetchBranch tests targeted fetch of a specific branch.
// Steps:
// 1. Sets up a test repository with a remote
// 2. Pushes main branch to remote
// 3. Clones to second repo, adds commit to main, pushes
// 4. Calls FetchBranch for main in original repo
// 5. Verifies the remote ref is updated
func TestFetchBranch(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	remoteDir, err := testutil.AddRemote(t, dir, "origin", true)
	if err != nil {
		t.Fatalf("Failed to add remote: %v", err)
	}
	defer testutil.CleanupTestRepo(t, remoteDir)

	// Get initial commit hash on origin/main
	initialHash, err := testutil.RunGit(t, dir, "rev-parse", "origin/main")
	if err != nil {
		t.Fatalf("Failed to get initial hash: %v", err)
	}

	// Clone to a second working copy and make changes
	secondDir := t.TempDir()
	_, err = testutil.RunGit(t, secondDir, "clone", remoteDir, ".")
	if err != nil {
		t.Fatalf("Failed to clone: %v", err)
	}
	testutil.ConfigureGitIdentity(t, secondDir)

	testutil.WriteFile(t, secondDir, "remote-change.txt", "remote content")
	_, err = testutil.RunGit(t, secondDir, "add", "remote-change.txt")
	if err != nil {
		t.Fatalf("Failed to add file in second repo: %v", err)
	}
	_, err = testutil.RunGit(t, secondDir, "commit", "-m", "remote commit")
	if err != nil {
		t.Fatalf("Failed to commit in second repo: %v", err)
	}
	_, err = testutil.RunGit(t, secondDir, "push", "origin", "main")
	if err != nil {
		t.Fatalf("Failed to push from second repo: %v", err)
	}

	// Fetch the branch using FetchBranch
	withGitRepo(t, dir, func() {
		err := git.FetchBranch("origin", "main")
		if err != nil {
			t.Fatalf("FetchBranch returned unexpected error: %v", err)
		}
	})

	// Verify the remote ref was updated
	newHash, err := testutil.RunGit(t, dir, "rev-parse", "origin/main")
	if err != nil {
		t.Fatalf("Failed to get new hash: %v", err)
	}

	if initialHash == newHash {
		t.Error("Expected origin/main to be updated after fetch, but hash unchanged")
	}
}

// TestFetchBranchNonExistent tests fetch of a non-existent branch.
// Steps:
// 1. Sets up a test repository with a remote
// 2. Calls FetchBranch for a branch that doesn't exist on remote
// 3. Verifies an error is returned
func TestFetchBranchNonExistent(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	remoteDir, err := testutil.AddRemote(t, dir, "origin", true)
	if err != nil {
		t.Fatalf("Failed to add remote: %v", err)
	}
	defer testutil.CleanupTestRepo(t, remoteDir)

	withGitRepo(t, dir, func() {
		err := git.FetchBranch("origin", "non-existent-branch")
		if err == nil {
			t.Error("Expected error when fetching non-existent branch, got nil")
		}
	})
}

// TestIsGitMergeInProgressTrue tests detection of an active git merge conflict.
// Steps:
// 1. Sets up a test repository with two branches containing conflicting changes
// 2. Starts a merge that produces a conflict
// 3. Verifies IsGitMergeInProgress returns true
func TestIsGitMergeInProgressTrue(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Create a branch with content
	_, err := testutil.RunGit(t, dir, "checkout", "-b", "feature")
	if err != nil {
		t.Fatalf("Failed to create branch: %v", err)
	}
	testutil.WriteFile(t, dir, "conflict.txt", "feature content")
	_, err = testutil.RunGit(t, dir, "add", "conflict.txt")
	if err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Feature commit")
	if err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Add conflicting content on main
	_, err = testutil.RunGit(t, dir, "checkout", "main")
	if err != nil {
		t.Fatalf("Failed to checkout main: %v", err)
	}
	testutil.WriteFile(t, dir, "conflict.txt", "main content")
	_, err = testutil.RunGit(t, dir, "add", "conflict.txt")
	if err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Main commit")
	if err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Start merge that will conflict
	_, _ = testutil.RunGit(t, dir, "merge", "feature")

	withGitRepo(t, dir, func() {
		if !git.IsGitMergeInProgress() {
			t.Error("Expected IsGitMergeInProgress to return true during merge conflict")
		}
	})
}

// TestIsGitMergeInProgressFalse tests that clean repos are not detected as merging.
// Steps:
// 1. Sets up a clean test repository
// 2. Verifies IsGitMergeInProgress returns false
func TestIsGitMergeInProgressFalse(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	withGitRepo(t, dir, func() {
		if git.IsGitMergeInProgress() {
			t.Error("Expected IsGitMergeInProgress to return false on clean repo")
		}
	})
}

// TestIsGitRebaseInProgressTrue tests detection of an active git rebase conflict.
// Steps:
// 1. Sets up a test repository with two branches containing conflicting changes
// 2. Starts a rebase that produces a conflict
// 3. Verifies IsGitRebaseInProgress returns true
func TestIsGitRebaseInProgressTrue(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Create a branch with content
	_, err := testutil.RunGit(t, dir, "checkout", "-b", "feature")
	if err != nil {
		t.Fatalf("Failed to create branch: %v", err)
	}
	testutil.WriteFile(t, dir, "conflict.txt", "feature content")
	_, err = testutil.RunGit(t, dir, "add", "conflict.txt")
	if err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Feature commit")
	if err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Add conflicting content on main
	_, err = testutil.RunGit(t, dir, "checkout", "main")
	if err != nil {
		t.Fatalf("Failed to checkout main: %v", err)
	}
	testutil.WriteFile(t, dir, "conflict.txt", "main content")
	_, err = testutil.RunGit(t, dir, "add", "conflict.txt")
	if err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Main commit")
	if err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Switch to feature and rebase onto main (will conflict)
	_, err = testutil.RunGit(t, dir, "checkout", "feature")
	if err != nil {
		t.Fatalf("Failed to checkout feature: %v", err)
	}
	_, _ = testutil.RunGit(t, dir, "rebase", "main")

	withGitRepo(t, dir, func() {
		if !git.IsGitRebaseInProgress() {
			t.Error("Expected IsGitRebaseInProgress to return true during rebase conflict")
		}
	})
}

// TestIsGitRebaseInProgressFalse tests that clean repos are not detected as rebasing.
// Steps:
// 1. Sets up a clean test repository
// 2. Verifies IsGitRebaseInProgress returns false
func TestIsGitRebaseInProgressFalse(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	withGitRepo(t, dir, func() {
		if git.IsGitRebaseInProgress() {
			t.Error("Expected IsGitRebaseInProgress to return false on clean repo")
		}
	})
}

// TestIsGitSquashMergeInProgressTrue tests detection of an active squash merge conflict.
// Steps:
// 1. Sets up a test repository with two branches containing conflicting changes
// 2. Starts a squash merge that produces a conflict
// 3. Verifies IsGitSquashMergeInProgress returns true
func TestIsGitSquashMergeInProgressTrue(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Create a branch with content
	_, err := testutil.RunGit(t, dir, "checkout", "-b", "feature")
	if err != nil {
		t.Fatalf("Failed to create branch: %v", err)
	}
	testutil.WriteFile(t, dir, "conflict.txt", "feature content")
	_, err = testutil.RunGit(t, dir, "add", "conflict.txt")
	if err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Feature commit")
	if err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Add conflicting content on main
	_, err = testutil.RunGit(t, dir, "checkout", "main")
	if err != nil {
		t.Fatalf("Failed to checkout main: %v", err)
	}
	testutil.WriteFile(t, dir, "conflict.txt", "main content")
	_, err = testutil.RunGit(t, dir, "add", "conflict.txt")
	if err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Main commit")
	if err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Start squash merge that will conflict
	_, _ = testutil.RunGit(t, dir, "merge", "--squash", "feature")

	withGitRepo(t, dir, func() {
		if !git.IsGitSquashMergeInProgress() {
			t.Error("Expected IsGitSquashMergeInProgress to return true during squash merge conflict")
		}
	})
}

// TestIsGitSquashMergeInProgressFalse tests that clean repos are not detected as squash merging.
// Steps:
// 1. Sets up a clean test repository
// 2. Verifies IsGitSquashMergeInProgress returns false
func TestIsGitSquashMergeInProgressFalse(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	withGitRepo(t, dir, func() {
		if git.IsGitSquashMergeInProgress() {
			t.Error("Expected IsGitSquashMergeInProgress to return false on clean repo")
		}
	})
}
