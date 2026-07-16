package cmd_test

import (
	"strings"
	"testing"

	"github.com/gittower/git-flow-next/test/testutil"
)

// TestFinishFeatureBranchBehindRemote tests that finish fails when local branch is behind remote.
// Steps:
// 1. Sets up a test repository with remote and initializes git-flow
// 2. Creates a feature branch and pushes it with tracking
// 3. Simulates remote changes by cloning, committing, and pushing
// 4. Fetches in original repo to update remote refs
// 5. Attempts to finish the feature branch
// 6. Verifies the operation fails with BranchBehindRemoteError
// 7. Verifies the error message contains resolution suggestions
func TestFinishFeatureBranchBehindRemote(t *testing.T) {
	dir, remoteDir := testutil.SetupTestRepoWithRemote(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer testutil.CleanupTestRepo(t, remoteDir)

	// Create feature branch
	_, err := testutil.RunGitFlow(t, dir, "feature", "start", "test-behind")
	if err != nil {
		t.Fatalf("Failed to create feature branch: %v", err)
	}

	// Add a commit to the feature branch
	testutil.WriteFile(t, dir, "feature.txt", "feature content")
	_, err = testutil.RunGit(t, dir, "add", "feature.txt")
	if err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Add feature file")
	if err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Push with tracking
	_, err = testutil.RunGit(t, dir, "push", "--set-upstream", "origin", "feature/test-behind")
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

	_, err = testutil.RunGit(t, secondDir, "checkout", "feature/test-behind")
	if err != nil {
		t.Fatalf("Failed to checkout feature branch in second repo: %v", err)
	}

	testutil.WriteFile(t, secondDir, "remote-change.txt", "remote content")
	_, err = testutil.RunGit(t, secondDir, "add", "remote-change.txt")
	if err != nil {
		t.Fatalf("Failed to add file in second repo: %v", err)
	}
	_, err = testutil.RunGit(t, secondDir, "commit", "-m", "Remote commit")
	if err != nil {
		t.Fatalf("Failed to commit in second repo: %v", err)
	}
	_, err = testutil.RunGit(t, secondDir, "push", "origin", "feature/test-behind")
	if err != nil {
		t.Fatalf("Failed to push from second repo: %v", err)
	}

	// Fetch in original repo to update remote refs
	_, err = testutil.RunGit(t, dir, "fetch", "origin")
	if err != nil {
		t.Fatalf("Failed to fetch: %v", err)
	}

	// Attempt to finish the feature branch
	output, err := testutil.RunGitFlow(t, dir, "feature", "finish", "test-behind")

	// Verify failure
	if err == nil {
		t.Error("Expected finish to fail when behind remote")
	}

	// Verify error message mentions being behind
	if !strings.Contains(output, "behind") {
		t.Errorf("Expected error message to mention 'behind'. Output: %s", output)
	}

	// Verify error message suggests update command
	if !strings.Contains(output, "git flow feature update") {
		t.Errorf("Expected error message to suggest update command. Output: %s", output)
	}

	// Verify error message suggests --force option
	if !strings.Contains(output, "--force") {
		t.Errorf("Expected error message to suggest --force option. Output: %s", output)
	}

	// Verify branch still exists (finish was aborted)
	if !testutil.BranchExists(t, dir, "feature/test-behind") {
		t.Error("Expected feature branch to still exist after aborted finish")
	}
}

// TestFinishFeatureBranchAheadOfRemote tests that finish succeeds when local is ahead of remote.
// Steps:
// 1. Sets up a test repository with remote and initializes git-flow
// 2. Creates a feature branch and pushes it with tracking
// 3. Adds a local commit without pushing
// 4. Finishes the feature branch
// 5. Verifies the operation succeeds
// 6. Verifies the note about being ahead is shown
func TestFinishFeatureBranchAheadOfRemote(t *testing.T) {
	dir, remoteDir := testutil.SetupTestRepoWithRemote(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer testutil.CleanupTestRepo(t, remoteDir)

	// Create feature branch
	_, err := testutil.RunGitFlow(t, dir, "feature", "start", "test-ahead")
	if err != nil {
		t.Fatalf("Failed to create feature branch: %v", err)
	}

	// Add a commit to the feature branch
	testutil.WriteFile(t, dir, "feature.txt", "feature content")
	_, err = testutil.RunGit(t, dir, "add", "feature.txt")
	if err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Add feature file")
	if err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Push with tracking
	_, err = testutil.RunGit(t, dir, "push", "--set-upstream", "origin", "feature/test-ahead")
	if err != nil {
		t.Fatalf("Failed to push branch: %v", err)
	}

	// Add another local commit without pushing
	testutil.WriteFile(t, dir, "local.txt", "local content")
	_, err = testutil.RunGit(t, dir, "add", "local.txt")
	if err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Local commit")
	if err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Finish the feature branch
	output, err := testutil.RunGitFlow(t, dir, "feature", "finish", "test-ahead")
	if err != nil {
		t.Fatalf("Expected finish to succeed when ahead of remote. Error: %v\nOutput: %s", err, output)
	}

	// Verify note about being ahead is shown
	if !strings.Contains(output, "ahead") {
		t.Errorf("Expected note about being ahead of remote. Output: %s", output)
	}

	// Verify branch is deleted
	if testutil.BranchExists(t, dir, "feature/test-ahead") {
		t.Error("Expected feature branch to be deleted")
	}

	// Verify changes are merged into develop
	_, err = testutil.RunGit(t, dir, "checkout", "develop")
	if err != nil {
		t.Fatalf("Failed to checkout develop: %v", err)
	}

	if !testutil.FileExists(t, dir, "local.txt") {
		t.Error("Expected local.txt to exist in develop branch")
	}
}

// TestFinishFeatureBranchDivergedFromRemote tests that finish fails when branches have diverged.
// Steps:
// 1. Sets up a test repository with remote and initializes git-flow
// 2. Creates a feature branch and pushes it with tracking
// 3. Simulates divergence: remote gets commit, local gets different commit
// 4. Fetches to update remote refs
// 5. Attempts to finish the feature branch
// 6. Verifies the operation fails with appropriate error
func TestFinishFeatureBranchDivergedFromRemote(t *testing.T) {
	dir, remoteDir := testutil.SetupTestRepoWithRemote(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer testutil.CleanupTestRepo(t, remoteDir)

	// Create feature branch
	_, err := testutil.RunGitFlow(t, dir, "feature", "start", "test-diverged")
	if err != nil {
		t.Fatalf("Failed to create feature branch: %v", err)
	}

	// Add a commit to the feature branch
	testutil.WriteFile(t, dir, "feature.txt", "feature content")
	_, err = testutil.RunGit(t, dir, "add", "feature.txt")
	if err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Add feature file")
	if err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Push with tracking
	_, err = testutil.RunGit(t, dir, "push", "--set-upstream", "origin", "feature/test-diverged")
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

	_, err = testutil.RunGit(t, secondDir, "checkout", "feature/test-diverged")
	if err != nil {
		t.Fatalf("Failed to checkout feature branch in second repo: %v", err)
	}

	testutil.WriteFile(t, secondDir, "remote-change.txt", "remote content")
	_, err = testutil.RunGit(t, secondDir, "add", "remote-change.txt")
	if err != nil {
		t.Fatalf("Failed to add file in second repo: %v", err)
	}
	_, err = testutil.RunGit(t, secondDir, "commit", "-m", "Remote commit")
	if err != nil {
		t.Fatalf("Failed to commit in second repo: %v", err)
	}
	_, err = testutil.RunGit(t, secondDir, "push", "origin", "feature/test-diverged")
	if err != nil {
		t.Fatalf("Failed to push from second repo: %v", err)
	}

	// Add local commit (before fetching - creates divergence)
	testutil.WriteFile(t, dir, "local-change.txt", "local content")
	_, err = testutil.RunGit(t, dir, "add", "local-change.txt")
	if err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Local commit")
	if err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Now fetch to update remote refs (creates divergence)
	_, err = testutil.RunGit(t, dir, "fetch", "origin")
	if err != nil {
		t.Fatalf("Failed to fetch: %v", err)
	}

	// Attempt to finish the feature branch
	output, err := testutil.RunGitFlow(t, dir, "feature", "finish", "test-diverged")

	// Verify failure
	if err == nil {
		t.Error("Expected finish to fail when diverged from remote")
	}

	// Verify error message mentions being behind (diverged branches are also behind)
	if !strings.Contains(output, "behind") {
		t.Errorf("Expected error message to mention 'behind'. Output: %s", output)
	}

	// Verify branch still exists (finish was aborted)
	if !testutil.BranchExists(t, dir, "feature/test-diverged") {
		t.Error("Expected feature branch to still exist after aborted finish")
	}
}

// TestFinishFeatureBranchNoTrackingBranch tests that finish succeeds without tracking branch.
// Steps:
// 1. Sets up a test repository and initializes git-flow (no remote)
// 2. Creates a feature branch (local only, no tracking)
// 3. Adds a commit to the feature branch
// 4. Finishes the feature branch
// 5. Verifies the operation succeeds (no remote check needed)
func TestFinishFeatureBranchNoTrackingBranch(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	_, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v", err)
	}

	// Create feature branch (local only, no remote)
	_, err = testutil.RunGitFlow(t, dir, "feature", "start", "test-no-tracking")
	if err != nil {
		t.Fatalf("Failed to create feature branch: %v", err)
	}

	// Add a commit to the feature branch
	testutil.WriteFile(t, dir, "feature.txt", "feature content")
	_, err = testutil.RunGit(t, dir, "add", "feature.txt")
	if err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Add feature file")
	if err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Finish the feature branch
	output, err := testutil.RunGitFlow(t, dir, "feature", "finish", "test-no-tracking")
	if err != nil {
		t.Fatalf("Expected finish to succeed without tracking branch. Error: %v\nOutput: %s", err, output)
	}

	// Verify branch is deleted
	if testutil.BranchExists(t, dir, "feature/test-no-tracking") {
		t.Error("Expected feature branch to be deleted")
	}

	// Verify changes are merged into develop
	_, err = testutil.RunGit(t, dir, "checkout", "develop")
	if err != nil {
		t.Fatalf("Failed to checkout develop: %v", err)
	}

	if !testutil.FileExists(t, dir, "feature.txt") {
		t.Error("Expected feature.txt to exist in develop branch")
	}
}

// TestFinishFeatureBranchForceBypassesRemoteCheck tests that --force bypasses remote check.
// Steps:
// 1. Sets up a test repository with remote and initializes git-flow
// 2. Creates a feature branch and pushes it with tracking
// 3. Simulates being behind remote (clone, commit, push from other repo)
// 4. Fetches to update remote refs
// 5. Finishes the feature branch with --force flag
// 6. Verifies the operation succeeds despite being behind
// 7. Verifies the branch is deleted and changes merged
func TestFinishFeatureBranchForceBypassesRemoteCheck(t *testing.T) {
	dir, remoteDir := testutil.SetupTestRepoWithRemote(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer testutil.CleanupTestRepo(t, remoteDir)

	// Create feature branch
	_, err := testutil.RunGitFlow(t, dir, "feature", "start", "test-force")
	if err != nil {
		t.Fatalf("Failed to create feature branch: %v", err)
	}

	// Add a commit to the feature branch
	testutil.WriteFile(t, dir, "feature.txt", "feature content")
	_, err = testutil.RunGit(t, dir, "add", "feature.txt")
	if err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Add feature file")
	if err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Push with tracking
	_, err = testutil.RunGit(t, dir, "push", "--set-upstream", "origin", "feature/test-force")
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

	_, err = testutil.RunGit(t, secondDir, "checkout", "feature/test-force")
	if err != nil {
		t.Fatalf("Failed to checkout feature branch in second repo: %v", err)
	}

	testutil.WriteFile(t, secondDir, "remote-change.txt", "remote content")
	_, err = testutil.RunGit(t, secondDir, "add", "remote-change.txt")
	if err != nil {
		t.Fatalf("Failed to add file in second repo: %v", err)
	}
	_, err = testutil.RunGit(t, secondDir, "commit", "-m", "Remote commit")
	if err != nil {
		t.Fatalf("Failed to commit in second repo: %v", err)
	}
	_, err = testutil.RunGit(t, secondDir, "push", "origin", "feature/test-force")
	if err != nil {
		t.Fatalf("Failed to push from second repo: %v", err)
	}

	// Fetch in original repo to update remote refs
	_, err = testutil.RunGit(t, dir, "fetch", "origin")
	if err != nil {
		t.Fatalf("Failed to fetch: %v", err)
	}

	// Finish the feature branch with --force
	output, err := testutil.RunGitFlow(t, dir, "feature", "finish", "--force", "test-force")
	if err != nil {
		t.Fatalf("Expected finish with --force to succeed. Error: %v\nOutput: %s", err, output)
	}

	// Verify branch is deleted
	if testutil.BranchExists(t, dir, "feature/test-force") {
		t.Error("Expected feature branch to be deleted")
	}

	// Verify changes are merged into develop
	_, err = testutil.RunGit(t, dir, "checkout", "develop")
	if err != nil {
		t.Fatalf("Failed to checkout develop: %v", err)
	}

	if !testutil.FileExists(t, dir, "feature.txt") {
		t.Error("Expected feature.txt to exist in develop branch")
	}
}

// TestFinishContinueSkipsRemoteCheck tests that --continue doesn't re-check remote status.
// Steps:
// 1. Sets up a test repository with remote and initializes git-flow
// 2. Creates merge conflict scenario
// 3. Attempts finish (will conflict)
// 4. Simulates remote getting ahead while conflict is being resolved
// 5. Resolves conflict and continues with --continue
// 6. Verifies continue succeeds (no remote re-check)
func TestFinishContinueSkipsRemoteCheck(t *testing.T) {
	dir, remoteDir := testutil.SetupTestRepoWithRemote(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer testutil.CleanupTestRepo(t, remoteDir)

	// Create conflicting content in develop
	_, err := testutil.RunGit(t, dir, "checkout", "develop")
	if err != nil {
		t.Fatalf("Failed to checkout develop: %v", err)
	}
	testutil.WriteFile(t, dir, "conflict.txt", "Develop version\nLine 2\nLine 3")
	_, err = testutil.RunGit(t, dir, "add", "conflict.txt")
	if err != nil {
		t.Fatalf("Failed to add conflict.txt: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Develop changes")
	if err != nil {
		t.Fatalf("Failed to commit develop changes: %v", err)
	}

	// Create feature branch with conflicting changes
	_, err = testutil.RunGitFlow(t, dir, "feature", "start", "test-continue-remote")
	if err != nil {
		t.Fatalf("Failed to create feature branch: %v", err)
	}

	testutil.WriteFile(t, dir, "conflict.txt", "Feature version\nLine 2 modified\nLine 3")
	_, err = testutil.RunGit(t, dir, "add", "conflict.txt")
	if err != nil {
		t.Fatalf("Failed to add conflict.txt: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Feature changes")
	if err != nil {
		t.Fatalf("Failed to commit feature changes: %v", err)
	}

	// Push the feature branch with tracking
	_, err = testutil.RunGit(t, dir, "push", "--set-upstream", "origin", "feature/test-continue-remote")
	if err != nil {
		t.Fatalf("Failed to push branch: %v", err)
	}

	// Add more conflicting changes to develop
	_, err = testutil.RunGit(t, dir, "checkout", "develop")
	if err != nil {
		t.Fatalf("Failed to checkout develop: %v", err)
	}
	testutil.WriteFile(t, dir, "conflict.txt", "Develop version updated\nLine 2\nLine 3 modified")
	_, err = testutil.RunGit(t, dir, "add", "conflict.txt")
	if err != nil {
		t.Fatalf("Failed to add conflict.txt: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "More develop changes")
	if err != nil {
		t.Fatalf("Failed to commit more develop changes: %v", err)
	}

	// Switch back to feature branch
	_, err = testutil.RunGit(t, dir, "checkout", "feature/test-continue-remote")
	if err != nil {
		t.Fatalf("Failed to checkout feature branch: %v", err)
	}

	// Try to finish (will conflict)
	output, _ := testutil.RunGitFlow(t, dir, "feature", "finish", "test-continue-remote")

	// Verify conflict was detected
	if !strings.Contains(output, "conflict") && !strings.Contains(output, "CONFLICT") {
		t.Fatalf("Expected merge conflict to be detected. Output: %s", output)
	}

	// Simulate remote getting ahead while conflict is being resolved
	secondDir := t.TempDir()
	_, err = testutil.RunGit(t, secondDir, "clone", remoteDir, ".")
	if err != nil {
		t.Fatalf("Failed to clone: %v", err)
	}
	testutil.ConfigureGitIdentity(t, secondDir)

	_, err = testutil.RunGit(t, secondDir, "checkout", "feature/test-continue-remote")
	if err != nil {
		t.Fatalf("Failed to checkout feature branch in second repo: %v", err)
	}

	testutil.WriteFile(t, secondDir, "remote-while-resolving.txt", "remote content")
	_, err = testutil.RunGit(t, secondDir, "add", "remote-while-resolving.txt")
	if err != nil {
		t.Fatalf("Failed to add file in second repo: %v", err)
	}
	_, err = testutil.RunGit(t, secondDir, "commit", "-m", "Remote commit while resolving")
	if err != nil {
		t.Fatalf("Failed to commit in second repo: %v", err)
	}
	_, err = testutil.RunGit(t, secondDir, "push", "origin", "feature/test-continue-remote")
	if err != nil {
		t.Fatalf("Failed to push from second repo: %v", err)
	}

	// Fetch in original repo to update remote refs (now local is behind)
	_, err = testutil.RunGit(t, dir, "fetch", "origin")
	if err != nil {
		t.Fatalf("Failed to fetch: %v", err)
	}

	// Resolve conflict
	testutil.WriteFile(t, dir, "conflict.txt", "Resolved version\nLine 2 resolved\nLine 3 resolved")
	_, err = testutil.RunGit(t, dir, "add", "conflict.txt")
	if err != nil {
		t.Fatalf("Failed to resolve conflict: %v", err)
	}

	// Continue finish operation (should succeed despite being behind remote now)
	continueOutput, err := testutil.RunGitFlow(t, dir, "feature", "finish", "--continue", "test-continue-remote")
	if err != nil {
		t.Fatalf("Expected continue to succeed without re-checking remote. Error: %v\nOutput: %s", err, continueOutput)
	}

	// Verify successful completion
	if !strings.Contains(continueOutput, "Successfully finished") {
		t.Errorf("Expected successful finish message after continue. Output: %s", continueOutput)
	}

	// Verify branch deleted
	if testutil.BranchExists(t, dir, "feature/test-continue-remote") {
		t.Error("Expected feature branch to be deleted after successful finish")
	}
}

// TestFinishFeatureBranchEqualToRemote tests that finish succeeds when equal to remote.
// Steps:
// 1. Sets up a test repository with remote and initializes git-flow
// 2. Creates a feature branch, adds commit, and pushes with tracking
// 3. Finishes the feature branch (local equals remote)
// 4. Verifies the operation succeeds
// 5. Verifies no warning about being ahead/behind
func TestFinishFeatureBranchEqualToRemote(t *testing.T) {
	dir, remoteDir := testutil.SetupTestRepoWithRemote(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer testutil.CleanupTestRepo(t, remoteDir)

	// Create feature branch
	_, err := testutil.RunGitFlow(t, dir, "feature", "start", "test-equal")
	if err != nil {
		t.Fatalf("Failed to create feature branch: %v", err)
	}

	// Add a commit to the feature branch
	testutil.WriteFile(t, dir, "feature.txt", "feature content")
	_, err = testutil.RunGit(t, dir, "add", "feature.txt")
	if err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Add feature file")
	if err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Push with tracking
	_, err = testutil.RunGit(t, dir, "push", "--set-upstream", "origin", "feature/test-equal")
	if err != nil {
		t.Fatalf("Failed to push branch: %v", err)
	}

	// Finish the feature branch (local equals remote)
	output, err := testutil.RunGitFlow(t, dir, "feature", "finish", "test-equal")
	if err != nil {
		t.Fatalf("Expected finish to succeed when equal to remote. Error: %v\nOutput: %s", err, output)
	}

	// Verify no warning about being ahead or behind
	if strings.Contains(output, "ahead") || strings.Contains(output, "behind") {
		t.Errorf("Expected no warning about ahead/behind when equal to remote. Output: %s", output)
	}

	// Verify branch is deleted
	if testutil.BranchExists(t, dir, "feature/test-equal") {
		t.Error("Expected feature branch to be deleted")
	}

	// Verify changes are merged into develop
	_, err = testutil.RunGit(t, dir, "checkout", "develop")
	if err != nil {
		t.Fatalf("Failed to checkout develop: %v", err)
	}

	if !testutil.FileExists(t, dir, "feature.txt") {
		t.Error("Expected feature.txt to exist in develop branch")
	}
}

// TestFinishFetchFailsButSyncCheckStillRuns tests that when fetch fails, the sync check still runs.
// Steps:
// 1. Sets up a test repository with remote and initializes git-flow
// 2. Creates a feature branch and pushes it with tracking
// 3. Simulates remote changes by cloning, committing, and pushing
// 4. Fetches in original repo to update remote refs (so sync check has data)
// 5. Breaks the remote URL so subsequent fetch fails
// 6. Attempts to finish with --fetch flag (fetch is disabled by default)
// 7. Verifies fetch failure is non-fatal (note is printed)
// 8. Verifies sync check still runs and catches "behind" state
func TestFinishFetchFailsButSyncCheckStillRuns(t *testing.T) {
	dir, remoteDir := testutil.SetupTestRepoWithRemote(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer testutil.CleanupTestRepo(t, remoteDir)

	// Create feature branch
	_, err := testutil.RunGitFlow(t, dir, "feature", "start", "test-fetch-fail")
	if err != nil {
		t.Fatalf("Failed to create feature branch: %v", err)
	}

	// Add a commit to the feature branch
	testutil.WriteFile(t, dir, "feature.txt", "feature content")
	_, err = testutil.RunGit(t, dir, "add", "feature.txt")
	if err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Add feature file")
	if err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Push with tracking
	_, err = testutil.RunGit(t, dir, "push", "--set-upstream", "origin", "feature/test-fetch-fail")
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

	_, err = testutil.RunGit(t, secondDir, "checkout", "feature/test-fetch-fail")
	if err != nil {
		t.Fatalf("Failed to checkout feature branch in second repo: %v", err)
	}

	testutil.WriteFile(t, secondDir, "remote-change.txt", "remote content")
	_, err = testutil.RunGit(t, secondDir, "add", "remote-change.txt")
	if err != nil {
		t.Fatalf("Failed to add file in second repo: %v", err)
	}
	_, err = testutil.RunGit(t, secondDir, "commit", "-m", "Remote commit")
	if err != nil {
		t.Fatalf("Failed to commit in second repo: %v", err)
	}
	_, err = testutil.RunGit(t, secondDir, "push", "origin", "feature/test-fetch-fail")
	if err != nil {
		t.Fatalf("Failed to push from second repo: %v", err)
	}

	// Fetch in original repo to update remote refs (so sync check has data to work with)
	_, err = testutil.RunGit(t, dir, "fetch", "origin")
	if err != nil {
		t.Fatalf("Failed to fetch: %v", err)
	}

	// Break the remote URL so fetch will fail
	_, err = testutil.RunGit(t, dir, "remote", "set-url", "origin", "/nonexistent/path/that/does/not/exist")
	if err != nil {
		t.Fatalf("Failed to change remote URL: %v", err)
	}

	// Attempt to finish with fetch explicitly enabled
	// Fetch should fail but sync check should still run with previously fetched refs
	output, err := testutil.RunGitFlow(t, dir, "feature", "finish", "--fetch", "test-fetch-fail")

	// Verify failure due to being behind (not due to fetch failure)
	if err == nil {
		t.Error("Expected finish to fail when behind remote")
	}

	// Verify fetch failure note is printed (non-fatal)
	if !strings.Contains(output, "Could not fetch") {
		t.Errorf("Expected output to mention fetch failure. Output: %s", output)
	}

	// Verify sync check still ran and caught the "behind" state
	if !strings.Contains(output, "behind") {
		t.Errorf("Expected error message to mention 'behind' (sync check should still run). Output: %s", output)
	}

	// Verify branch still exists (finish was aborted)
	if !testutil.BranchExists(t, dir, "feature/test-fetch-fail") {
		t.Error("Expected feature branch to still exist after aborted finish")
	}
}
