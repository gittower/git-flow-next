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

	// Record develop's SHA before finishing
	developBefore, err := testutil.RunGit(t, dir, "rev-parse", "develop")
	if err != nil {
		t.Fatalf("Failed to get develop SHA: %v", err)
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

	// Verify error message suggests pulling the remote changes
	if !strings.Contains(output, "git pull") {
		t.Errorf("Expected error message to suggest 'git pull'. Output: %s", output)
	}

	// Verify error message suggests --force option
	if !strings.Contains(output, "--force") {
		t.Errorf("Expected error message to suggest --force option. Output: %s", output)
	}

	// Verify the merge did not happen: develop is unchanged
	developAfter, err := testutil.RunGit(t, dir, "rev-parse", "develop")
	if err != nil {
		t.Fatalf("Failed to get develop SHA after: %v", err)
	}
	if developBefore != developAfter {
		t.Errorf("Expected develop to be unchanged after aborted finish. Before: %s After: %s", developBefore, developAfter)
	}

	// Verify branch still exists (finish was aborted)
	if !testutil.BranchExists(t, dir, "feature/test-behind") {
		t.Error("Expected feature branch to still exist after aborted finish")
	}
}

// TestFinishFeatureBranchAheadOfRemote tests that finish aborts when local is ahead of remote.
// This is Scenario 9: being ahead now aborts (previously a silent note that proceeded and merged).
// Steps:
// 1. Sets up a test repository with remote and initializes git-flow
// 2. Creates a feature branch and pushes it with tracking
// 3. Adds a local commit without pushing (ahead by 1)
// 4. Records develop's SHA before finishing
// 5. Attempts to finish the feature branch
// 6. Verifies the operation fails with an "ahead" message including the commit count
// 7. Verifies the merge did not happen (develop unchanged, branch still exists)
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

	// Add another local commit without pushing (ahead by 1)
	testutil.WriteFile(t, dir, "local.txt", "local content")
	_, err = testutil.RunGit(t, dir, "add", "local.txt")
	if err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Local commit")
	if err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Record develop's SHA before finishing
	developBefore, err := testutil.RunGit(t, dir, "rev-parse", "develop")
	if err != nil {
		t.Fatalf("Failed to get develop SHA: %v", err)
	}

	// Attempt to finish the feature branch
	output, err := testutil.RunGitFlow(t, dir, "feature", "finish", "test-ahead")

	// Verify failure
	if err == nil {
		t.Errorf("Expected finish to fail when ahead of remote. Output: %s", output)
	}

	// Verify error message mentions being ahead with the commit count
	if !strings.Contains(output, "ahead") {
		t.Errorf("Expected error message to mention 'ahead'. Output: %s", output)
	}
	if !strings.Contains(output, "1 commit") {
		t.Errorf("Expected error message to include the commit count. Output: %s", output)
	}

	// Verify the merge did not happen: develop is unchanged
	developAfter, err := testutil.RunGit(t, dir, "rev-parse", "develop")
	if err != nil {
		t.Fatalf("Failed to get develop SHA after: %v", err)
	}
	if developBefore != developAfter {
		t.Errorf("Expected develop to be unchanged after aborted finish. Before: %s After: %s", developBefore, developAfter)
	}

	// Verify branch still exists (finish was aborted)
	if !testutil.BranchExists(t, dir, "feature/test-ahead") {
		t.Error("Expected feature branch to still exist after aborted finish")
	}
}

// TestFinishFeatureBranchDivergedFromRemote tests that finish aborts with a diverged-specific
// message when branches have diverged. This is Scenario 10: diverged now renders its own message
// rather than reusing the plain "behind" wording.
// Steps:
// 1. Sets up a test repository with remote and initializes git-flow
// 2. Creates a feature branch and pushes it with tracking
// 3. Simulates divergence: remote gets commit, local gets different commit
// 4. Fetches to update remote refs
// 5. Records develop's SHA before finishing
// 6. Attempts to finish the feature branch
// 7. Verifies the operation fails with a diverged-specific message (not "behind")
// 8. Verifies the merge did not happen (develop unchanged, branch still exists)
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

	// Record develop's SHA before finishing
	developBefore, err := testutil.RunGit(t, dir, "rev-parse", "develop")
	if err != nil {
		t.Fatalf("Failed to get develop SHA: %v", err)
	}

	// Attempt to finish the feature branch
	output, err := testutil.RunGitFlow(t, dir, "feature", "finish", "test-diverged")

	// Verify failure
	if err == nil {
		t.Error("Expected finish to fail when diverged from remote")
	}

	// Verify error message describes divergence and is NOT the plain "behind" wording
	if !strings.Contains(output, "diverged") {
		t.Errorf("Expected error message to mention 'diverged'. Output: %s", output)
	}
	if strings.Contains(output, "behind") {
		t.Errorf("Expected a diverged-specific message, not the 'behind' wording. Output: %s", output)
	}

	// Verify the merge did not happen: develop is unchanged
	developAfter, err := testutil.RunGit(t, dir, "rev-parse", "develop")
	if err != nil {
		t.Fatalf("Failed to get develop SHA after: %v", err)
	}
	if developBefore != developAfter {
		t.Errorf("Expected develop to be unchanged after aborted finish. Before: %s After: %s", developBefore, developAfter)
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

// TestFinishContinueSkipsRemoteCheck tests that --continue does not re-run the fetch/sync preflight.
// This is Scenario 23: the preflight is gated to the initial finish only. The topic is pushed in
// sync so the initial finish passes the preflight and can then reach a merge conflict. The remote is
// then broken; if --continue re-ran the preflight it would fatally abort on the unreachable remote.
// keepremote is enabled so the unrelated remote-branch deletion (which would also fail on the broken
// remote) does not mask the behavior under test.
// Steps:
// 1. Sets up a test repository with remote and initializes git-flow; enables keepremote
// 2. Creates conflicting develop + feature changes; pushes the feature in sync
// 3. Runs the initial finish, which passes the preflight then hits a merge conflict
// 4. Verifies the conflict via .git/MERGE_HEAD
// 5. Breaks the remote so any fetch would fatally fail
// 6. Resolves the conflict and runs --continue
// 7. Verifies continue completes without fetching or a transport abort, and clears the merge state
func TestFinishContinueSkipsRemoteCheck(t *testing.T) {
	dir, remoteDir := testutil.SetupTestRepoWithRemote(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer testutil.CleanupTestRepo(t, remoteDir)

	// Keep the remote branch so the delete step does not touch the (soon-broken) remote —
	// this isolates the test to the preflight-skip behavior.
	_, err := testutil.RunGit(t, dir, "config", "gitflow.feature.finish.keepremote", "true")
	if err != nil {
		t.Fatalf("Failed to set keepremote config: %v", err)
	}

	// Create conflicting content in develop
	_, err = testutil.RunGit(t, dir, "checkout", "develop")
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

	// Push the feature branch with tracking (in sync — the preflight will pass)
	_, err = testutil.RunGit(t, dir, "push", "--set-upstream", "origin", "feature/test-continue-remote")
	if err != nil {
		t.Fatalf("Failed to push branch: %v", err)
	}

	// Add more conflicting changes to develop so the merge will conflict
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

	// Try to finish (default fetch=true) — passes the preflight, then hits the conflict
	output, _ := testutil.RunGitFlow(t, dir, "feature", "finish", "test-continue-remote")

	// Verify conflict was detected
	if !strings.Contains(output, "conflict") && !strings.Contains(output, "CONFLICT") {
		t.Fatalf("Expected merge conflict to be detected. Output: %s", output)
	}

	// Verify the conflict via .git/MERGE_HEAD (not just output text)
	if !testutil.FileExists(t, dir, ".git/MERGE_HEAD") {
		t.Fatalf("Expected .git/MERGE_HEAD to exist during the merge conflict")
	}

	// Break the remote so any fetch during continue would fatally fail
	_, err = testutil.RunGit(t, dir, "remote", "set-url", "origin", "./nonexistent-remote-repo.git")
	if err != nil {
		t.Fatalf("Failed to break remote: %v", err)
	}

	// Resolve conflict
	testutil.WriteFile(t, dir, "conflict.txt", "Resolved version\nLine 2 resolved\nLine 3 resolved")
	_, err = testutil.RunGit(t, dir, "add", "conflict.txt")
	if err != nil {
		t.Fatalf("Failed to resolve conflict: %v", err)
	}

	// Continue finish operation (must not fetch or abort on the broken remote)
	continueOutput, err := testutil.RunGitFlow(t, dir, "feature", "finish", "--continue", "test-continue-remote")
	if err != nil {
		t.Fatalf("Expected continue to succeed without re-running the preflight. Error: %v\nOutput: %s", err, continueOutput)
	}

	// Verify no fetch occurred during continue
	if strings.Contains(continueOutput, "Fetching from remote") {
		t.Errorf("Expected no fetch during --continue. Output: %s", continueOutput)
	}

	// Verify successful completion
	if !strings.Contains(continueOutput, "Successfully finished") {
		t.Errorf("Expected successful finish message after continue. Output: %s", continueOutput)
	}

	// Verify the merge state is cleared
	if testutil.FileExists(t, dir, ".git/MERGE_HEAD") {
		t.Errorf("Expected .git/MERGE_HEAD to be cleared after continue")
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

	// Verify fetch ran (default fetch=true with a reachable remote) — Scenario 3
	if !strings.Contains(output, "Fetching from remote") {
		t.Errorf("Expected fetch to occur before the in-sync finish. Output: %s", output)
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

// TestFinishNoFetchStillRunsSyncCheck tests that --no-fetch skips the fetch but still runs
// the sync check (Scenario 22). This replaces the old TestFinishFetchFailsButSyncCheckStillRuns,
// which asserted a broken-URL fetch failure was non-fatal — a transport fetch failure is now
// fatal (see TestFinishUnreachableRemoteAborts, Scenario 5). Here --no-fetch skips the fetch, so
// the sync check runs against existing (stale) tracking data and still aborts on "behind".
// Steps:
// 1. Sets up a test repository with remote and initializes git-flow
// 2. Creates a feature branch, commits, and pushes it with tracking
// 3. Resets the local branch back one commit so it is behind the remote by 1
// 4. Records develop's SHA before finishing
// 5. Runs finish with --no-fetch
// 6. Verifies no fetch occurred but the sync check still aborts with "behind"
// 7. Verifies the merge did not happen (develop unchanged, branch still exists)
func TestFinishNoFetchStillRunsSyncCheck(t *testing.T) {
	dir, remoteDir := testutil.SetupTestRepoWithRemote(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer testutil.CleanupTestRepo(t, remoteDir)

	// Create feature branch
	_, err := testutil.RunGitFlow(t, dir, "feature", "start", "test-no-fetch-sync")
	if err != nil {
		t.Fatalf("Failed to create feature branch: %v", err)
	}

	// Add a commit and push with tracking
	testutil.WriteFile(t, dir, "feature.txt", "feature content")
	if _, err = testutil.RunGit(t, dir, "add", "feature.txt"); err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	if _, err = testutil.RunGit(t, dir, "commit", "-m", "Add feature file"); err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}
	if _, err = testutil.RunGit(t, dir, "push", "--set-upstream", "origin", "feature/test-no-fetch-sync"); err != nil {
		t.Fatalf("Failed to push branch: %v", err)
	}

	// Reset local branch back one commit so it is behind the remote by 1
	if _, err = testutil.RunGit(t, dir, "reset", "--hard", "HEAD~1"); err != nil {
		t.Fatalf("Failed to reset branch: %v", err)
	}

	// Record develop's SHA before finishing
	developBefore, err := testutil.RunGit(t, dir, "rev-parse", "develop")
	if err != nil {
		t.Fatalf("Failed to get develop SHA: %v", err)
	}

	// Finish with --no-fetch: the fetch is skipped but the sync check still runs
	output, err := testutil.RunGitFlow(t, dir, "feature", "finish", "--no-fetch", "test-no-fetch-sync")

	// Verify failure due to being behind
	if err == nil {
		t.Errorf("Expected finish to fail when behind remote. Output: %s", output)
	}

	// Verify no fetch occurred
	if strings.Contains(output, "Fetching from remote") {
		t.Errorf("Expected no fetch with --no-fetch. Output: %s", output)
	}

	// Verify the sync check still ran and caught the "behind" state
	if !strings.Contains(output, "behind") {
		t.Errorf("Expected error message to mention 'behind' (sync check should still run). Output: %s", output)
	}

	// Verify the merge did not happen: develop is unchanged
	developAfter, err := testutil.RunGit(t, dir, "rev-parse", "develop")
	if err != nil {
		t.Fatalf("Failed to get develop SHA after: %v", err)
	}
	if developBefore != developAfter {
		t.Errorf("Expected develop to be unchanged after aborted finish. Before: %s After: %s", developBefore, developAfter)
	}

	// Verify branch still exists (finish was aborted)
	if !testutil.BranchExists(t, dir, "feature/test-no-fetch-sync") {
		t.Error("Expected feature branch to still exist after aborted finish")
	}
}

// TestFinishCompareErrorAborts verifies that when a tracking branch exists but the sync comparison
// itself fails (e.g. a corrupt/dangling tracking ref), finish fails closed rather than merging
// against an undetermined sync status. This exercises the compare-error branch of the preflight,
// which HasTrackingBranch gates entry into, so reaching a compare error is unexpected and fatal.
// Steps:
//  1. Sets up a test repository with remote and initializes git-flow
//  2. Creates a feature branch, commits, and pushes it (establishes a tracking ref)
//  3. Corrupts the loose remote-tracking ref to point at a nonexistent object, so @{upstream} still
//     resolves by name (HasTrackingBranch stays true) but rev-list can no longer compare
//  4. Finishes with --no-fetch (so the corrupt ref is not repaired by a fetch)
//  5. Verifies finish fails, names the sync-status determination, and does not merge
func TestFinishCompareErrorAborts(t *testing.T) {
	dir, remoteDir := testutil.SetupTestRepoWithRemote(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer testutil.CleanupTestRepo(t, remoteDir)

	if _, err := testutil.RunGitFlow(t, dir, "feature", "start", "test-compare-error"); err != nil {
		t.Fatalf("Failed to create feature branch: %v", err)
	}
	commitFeatureAndPush(t, dir, "feature.txt", "feature content", "Add feature file", "feature/test-compare-error")

	// Corrupt the loose remote-tracking ref: point it at a nonexistent object. @{upstream} still
	// resolves the tracking name (so HasTrackingBranch stays true), but rev-list can no longer walk
	// the ref, forcing CompareBranchWithRemote to fail.
	trackingRef := ".git/refs/remotes/origin/feature/test-compare-error"
	if _, err := testutil.RunGit(t, dir, "rev-parse", "--verify", "origin/feature/test-compare-error"); err != nil {
		t.Fatalf("Precondition failed: expected a loose tracking ref to exist: %v", err)
	}
	if err := testutil.WriteFile(t, dir, trackingRef, "deadbeefdeadbeefdeadbeefdeadbeefdeadbeef\n"); err != nil {
		t.Fatalf("Failed to corrupt tracking ref: %v", err)
	}

	developBefore, err := testutil.RunGit(t, dir, "rev-parse", "develop")
	if err != nil {
		t.Fatalf("Failed to get develop SHA: %v", err)
	}

	// --no-fetch so the corrupt ref is not repaired by a fetch before the sync check
	output, err := testutil.RunGitFlow(t, dir, "feature", "finish", "--no-fetch", "test-compare-error")
	if err == nil {
		t.Errorf("Expected finish to fail when the sync status cannot be determined. Output: %s", output)
	}

	// Verify the failure is the sync-status determination (not a merge or unrelated error)
	if !strings.Contains(output, "determine sync status") {
		t.Errorf("Expected the error to name the sync-status determination. Output: %s", output)
	}

	// Verify no merge happened: develop unchanged and the branch still exists
	developAfter, err := testutil.RunGit(t, dir, "rev-parse", "develop")
	if err != nil {
		t.Fatalf("Failed to get develop SHA after: %v", err)
	}
	if developBefore != developAfter {
		t.Errorf("Expected develop unchanged after aborted finish. Before: %s After: %s", developBefore, developAfter)
	}
	if !testutil.BranchExists(t, dir, "feature/test-compare-error") {
		t.Error("Expected feature branch to still exist after aborted finish")
	}
}
