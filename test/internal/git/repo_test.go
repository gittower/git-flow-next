package git_test

import (
	goerrors "errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/gittower/git-flow-next/internal/git"
	"github.com/gittower/git-flow-next/test/testutil"
)

// openRepo opens a git.Repo handle for dir, failing the test on error. It is the
// CWD-independent replacement for the old withGitRepo(os.Chdir) helper.
func openRepo(t *testing.T, dir string) *git.Repo {
	t.Helper()
	repo, err := git.Open(dir)
	if err != nil {
		t.Fatalf("git.Open(%q) failed: %v", dir, err)
	}
	return repo
}

// setupConflictingBranches seeds conflict.txt on the base, then has a feature
// branch and the diverging main branch each modify it differently, producing a
// modify/modify (UU) conflict per GIT_TEST_SCENARIOS.md. It leaves the repo
// checked out on main.
func setupConflictingBranches(t *testing.T, dir string) {
	t.Helper()
	// Seed the file on the base (main) so both sides modify a common ancestor
	// version, yielding a UU (both-modified) conflict rather than AA (add/add).
	testutil.WriteFile(t, dir, "conflict.txt", "base content")
	if _, err := testutil.RunGit(t, dir, "add", "conflict.txt"); err != nil {
		t.Fatalf("Failed to add base file: %v", err)
	}
	if _, err := testutil.RunGit(t, dir, "commit", "-m", "Base conflict file"); err != nil {
		t.Fatalf("Failed to commit base file: %v", err)
	}
	if _, err := testutil.RunGit(t, dir, "checkout", "-b", "feature"); err != nil {
		t.Fatalf("Failed to create feature branch: %v", err)
	}
	testutil.WriteFile(t, dir, "conflict.txt", "feature content")
	if _, err := testutil.RunGit(t, dir, "add", "conflict.txt"); err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	if _, err := testutil.RunGit(t, dir, "commit", "-m", "Feature commit"); err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}
	if _, err := testutil.RunGit(t, dir, "checkout", "main"); err != nil {
		t.Fatalf("Failed to checkout main: %v", err)
	}
	testutil.WriteFile(t, dir, "conflict.txt", "main content")
	if _, err := testutil.RunGit(t, dir, "add", "conflict.txt"); err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	if _, err := testutil.RunGit(t, dir, "commit", "-m", "Main commit"); err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}
}

func TestRemoteBranchExists_ExistingBranch(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	remoteDir, err := testutil.AddRemote(t, dir, "origin", false)
	if err != nil {
		t.Fatalf("Failed to add remote: %v", err)
	}
	defer testutil.CleanupTestRepo(t, remoteDir)

	if _, err = testutil.RunGit(t, dir, "checkout", "-b", "feature/test"); err != nil {
		t.Fatalf("Failed to create branch: %v", err)
	}
	testutil.WriteFile(t, dir, "test.txt", "test content")
	if _, err = testutil.RunGit(t, dir, "add", "test.txt"); err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	if _, err = testutil.RunGit(t, dir, "commit", "-m", "test commit"); err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}
	if _, err = testutil.RunGit(t, dir, "push", "-u", "origin", "feature/test"); err != nil {
		t.Fatalf("Failed to push branch: %v", err)
	}
	if _, err = testutil.RunGit(t, dir, "fetch", "origin"); err != nil {
		t.Fatalf("Failed to fetch from remote: %v", err)
	}

	repo := openRepo(t, dir)
	if !repo.RemoteBranchExists("origin", "feature/test") {
		t.Error("Expected RemoteBranchExists to return true for existing branch")
	}
}

func TestRemoteBranchExists_NonExistentBranch(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	remoteDir, err := testutil.AddRemote(t, dir, "origin", true)
	if err != nil {
		t.Fatalf("Failed to add remote: %v", err)
	}
	defer testutil.CleanupTestRepo(t, remoteDir)

	if _, err = testutil.RunGit(t, dir, "fetch", "origin"); err != nil {
		t.Fatalf("Failed to fetch from remote: %v", err)
	}

	repo := openRepo(t, dir)
	if repo.RemoteBranchExists("origin", "feature/non-existent") {
		t.Error("Expected RemoteBranchExists to return false for non-existent branch")
	}
}

func TestDeleteNonExistentRemoteBranch(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	remoteDir, err := testutil.AddRemote(t, dir, "origin", true)
	if err != nil {
		t.Fatalf("Failed to add remote: %v", err)
	}
	defer testutil.CleanupTestRepo(t, remoteDir)

	repo := openRepo(t, dir)
	if err := repo.DeleteRemoteBranch("origin", "feature/non-existent"); err == nil {
		t.Error("Expected an error when deleting non-existent remote branch, got nil")
	}
}

func TestDeleteExistingRemoteBranch(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	remoteDir, err := testutil.AddRemote(t, dir, "origin", true)
	if err != nil {
		t.Fatalf("Failed to add remote: %v", err)
	}
	defer testutil.CleanupTestRepo(t, remoteDir)

	if _, err = testutil.RunGit(t, dir, "checkout", "-b", "feature/test"); err != nil {
		t.Fatalf("Failed to create branch: %v", err)
	}
	testutil.WriteFile(t, dir, "test.txt", "test content")
	if _, err = testutil.RunGit(t, dir, "add", "test.txt"); err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	if _, err = testutil.RunGit(t, dir, "commit", "-m", "test commit"); err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}
	if _, err = testutil.RunGit(t, dir, "push", "origin", "feature/test"); err != nil {
		t.Fatalf("Failed to push branch: %v", err)
	}
	if _, err = testutil.RunGit(t, dir, "fetch", "origin"); err != nil {
		t.Fatalf("Failed to fetch from remote: %v", err)
	}

	repo := openRepo(t, dir)
	if err := repo.DeleteRemoteBranch("origin", "feature/test"); err != nil {
		t.Errorf("Expected no error when deleting existing remote branch, got: %v", err)
	}

	if _, err = testutil.RunGit(t, dir, "fetch", "origin", "--prune"); err != nil {
		t.Fatalf("Failed to fetch from remote: %v", err)
	}
	if _, err = testutil.RunGit(t, dir, "rev-parse", "--verify", "refs/remotes/origin/feature/test"); err == nil {
		t.Error("Expected remote tracking branch to be deleted")
	}
}

func TestDeleteBranchFromNonExistentRemote(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	repo := openRepo(t, dir)
	if err := repo.DeleteRemoteBranch("non-existent-remote", "feature/test"); err == nil {
		t.Error("Expected an error when deleting from non-existent remote, got nil")
	}
}

func TestGetTrackingBranch(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	remoteDir, err := testutil.AddRemote(t, dir, "origin", false)
	if err != nil {
		t.Fatalf("Failed to add remote: %v", err)
	}
	defer testutil.CleanupTestRepo(t, remoteDir)

	if _, err = testutil.RunGit(t, dir, "checkout", "-b", "feature/test"); err != nil {
		t.Fatalf("Failed to create branch: %v", err)
	}
	testutil.WriteFile(t, dir, "test.txt", "test content")
	if _, err = testutil.RunGit(t, dir, "add", "test.txt"); err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	if _, err = testutil.RunGit(t, dir, "commit", "-m", "test commit"); err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}
	if _, err = testutil.RunGit(t, dir, "push", "--set-upstream", "origin", "feature/test"); err != nil {
		t.Fatalf("Failed to push branch: %v", err)
	}

	repo := openRepo(t, dir)
	trackingBranch, err := repo.GetTrackingBranch("feature/test")
	if err != nil {
		t.Fatalf("GetTrackingBranch returned unexpected error: %v", err)
	}
	if trackingBranch != "origin/feature/test" {
		t.Errorf("Expected tracking branch 'origin/feature/test', got '%s'", trackingBranch)
	}
}

func TestGetTrackingBranchNoTracking(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	if _, err := testutil.RunGit(t, dir, "checkout", "-b", "feature/local-only"); err != nil {
		t.Fatalf("Failed to create branch: %v", err)
	}

	repo := openRepo(t, dir)
	if _, err := repo.GetTrackingBranch("feature/local-only"); err == nil {
		t.Error("Expected error for branch without tracking, got nil")
	}
}

func TestCompareBranchWithRemoteEqual(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	remoteDir, err := testutil.AddRemote(t, dir, "origin", false)
	if err != nil {
		t.Fatalf("Failed to add remote: %v", err)
	}
	defer testutil.CleanupTestRepo(t, remoteDir)

	if _, err = testutil.RunGit(t, dir, "checkout", "-b", "feature/test"); err != nil {
		t.Fatalf("Failed to create branch: %v", err)
	}
	testutil.WriteFile(t, dir, "test.txt", "test content")
	if _, err = testutil.RunGit(t, dir, "add", "test.txt"); err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	if _, err = testutil.RunGit(t, dir, "commit", "-m", "test commit"); err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}
	if _, err = testutil.RunGit(t, dir, "push", "--set-upstream", "origin", "feature/test"); err != nil {
		t.Fatalf("Failed to push branch: %v", err)
	}

	repo := openRepo(t, dir)
	status, count, err := repo.CompareBranchWithRemote("feature/test")
	if err != nil {
		t.Fatalf("CompareBranchWithRemote returned unexpected error: %v", err)
	}
	if status != git.SyncStatusEqual {
		t.Errorf("Expected status SyncStatusEqual, got %s", status)
	}
	if count != 0 {
		t.Errorf("Expected count 0, got %d", count)
	}
}

func TestCompareBranchWithRemoteAhead(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	remoteDir, err := testutil.AddRemote(t, dir, "origin", false)
	if err != nil {
		t.Fatalf("Failed to add remote: %v", err)
	}
	defer testutil.CleanupTestRepo(t, remoteDir)

	if _, err = testutil.RunGit(t, dir, "checkout", "-b", "feature/test"); err != nil {
		t.Fatalf("Failed to create branch: %v", err)
	}
	testutil.WriteFile(t, dir, "test.txt", "test content")
	if _, err = testutil.RunGit(t, dir, "add", "test.txt"); err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	if _, err = testutil.RunGit(t, dir, "commit", "-m", "test commit"); err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}
	if _, err = testutil.RunGit(t, dir, "push", "--set-upstream", "origin", "feature/test"); err != nil {
		t.Fatalf("Failed to push branch: %v", err)
	}
	testutil.WriteFile(t, dir, "local.txt", "local content")
	if _, err = testutil.RunGit(t, dir, "add", "local.txt"); err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	if _, err = testutil.RunGit(t, dir, "commit", "-m", "local commit"); err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	repo := openRepo(t, dir)
	status, count, err := repo.CompareBranchWithRemote("feature/test")
	if err != nil {
		t.Fatalf("CompareBranchWithRemote returned unexpected error: %v", err)
	}
	if status != git.SyncStatusAhead {
		t.Errorf("Expected status SyncStatusAhead, got %s", status)
	}
	if count != 1 {
		t.Errorf("Expected count 1, got %d", count)
	}
}

func TestCompareBranchWithRemoteBehind(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	remoteDir, err := testutil.AddRemote(t, dir, "origin", false)
	if err != nil {
		t.Fatalf("Failed to add remote: %v", err)
	}
	defer testutil.CleanupTestRepo(t, remoteDir)

	if _, err = testutil.RunGit(t, dir, "checkout", "-b", "feature/test"); err != nil {
		t.Fatalf("Failed to create branch: %v", err)
	}
	testutil.WriteFile(t, dir, "test.txt", "test content")
	if _, err = testutil.RunGit(t, dir, "add", "test.txt"); err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	if _, err = testutil.RunGit(t, dir, "commit", "-m", "test commit"); err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}
	if _, err = testutil.RunGit(t, dir, "push", "--set-upstream", "origin", "feature/test"); err != nil {
		t.Fatalf("Failed to push branch: %v", err)
	}

	secondDir := t.TempDir()
	if _, err = testutil.RunGit(t, secondDir, "clone", remoteDir, "."); err != nil {
		t.Fatalf("Failed to clone: %v", err)
	}
	testutil.ConfigureGitIdentity(t, secondDir)
	if _, err = testutil.RunGit(t, secondDir, "checkout", "feature/test"); err != nil {
		t.Fatalf("Failed to checkout feature branch in second repo: %v", err)
	}
	testutil.WriteFile(t, secondDir, "remote-change.txt", "remote content")
	if _, err = testutil.RunGit(t, secondDir, "add", "remote-change.txt"); err != nil {
		t.Fatalf("Failed to add file in second repo: %v", err)
	}
	if _, err = testutil.RunGit(t, secondDir, "commit", "-m", "remote commit"); err != nil {
		t.Fatalf("Failed to commit in second repo: %v", err)
	}
	if _, err = testutil.RunGit(t, secondDir, "push", "origin", "feature/test"); err != nil {
		t.Fatalf("Failed to push from second repo: %v", err)
	}
	if _, err = testutil.RunGit(t, dir, "fetch", "origin"); err != nil {
		t.Fatalf("Failed to fetch: %v", err)
	}

	repo := openRepo(t, dir)
	status, count, err := repo.CompareBranchWithRemote("feature/test")
	if err != nil {
		t.Fatalf("CompareBranchWithRemote returned unexpected error: %v", err)
	}
	if status != git.SyncStatusBehind {
		t.Errorf("Expected status SyncStatusBehind, got %s", status)
	}
	if count != 1 {
		t.Errorf("Expected count 1, got %d", count)
	}
}

func TestCompareBranchWithRemoteDiverged(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	remoteDir, err := testutil.AddRemote(t, dir, "origin", false)
	if err != nil {
		t.Fatalf("Failed to add remote: %v", err)
	}
	defer testutil.CleanupTestRepo(t, remoteDir)

	if _, err = testutil.RunGit(t, dir, "checkout", "-b", "feature/test"); err != nil {
		t.Fatalf("Failed to create branch: %v", err)
	}
	testutil.WriteFile(t, dir, "test.txt", "test content")
	if _, err = testutil.RunGit(t, dir, "add", "test.txt"); err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	if _, err = testutil.RunGit(t, dir, "commit", "-m", "test commit"); err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}
	if _, err = testutil.RunGit(t, dir, "push", "--set-upstream", "origin", "feature/test"); err != nil {
		t.Fatalf("Failed to push branch: %v", err)
	}

	secondDir := t.TempDir()
	if _, err = testutil.RunGit(t, secondDir, "clone", remoteDir, "."); err != nil {
		t.Fatalf("Failed to clone: %v", err)
	}
	testutil.ConfigureGitIdentity(t, secondDir)
	if _, err = testutil.RunGit(t, secondDir, "checkout", "feature/test"); err != nil {
		t.Fatalf("Failed to checkout feature branch in second repo: %v", err)
	}
	testutil.WriteFile(t, secondDir, "remote-change.txt", "remote content")
	if _, err = testutil.RunGit(t, secondDir, "add", "remote-change.txt"); err != nil {
		t.Fatalf("Failed to add file in second repo: %v", err)
	}
	if _, err = testutil.RunGit(t, secondDir, "commit", "-m", "remote commit"); err != nil {
		t.Fatalf("Failed to commit in second repo: %v", err)
	}
	if _, err = testutil.RunGit(t, secondDir, "push", "origin", "feature/test"); err != nil {
		t.Fatalf("Failed to push from second repo: %v", err)
	}

	testutil.WriteFile(t, dir, "local-change.txt", "local content")
	if _, err = testutil.RunGit(t, dir, "add", "local-change.txt"); err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	if _, err = testutil.RunGit(t, dir, "commit", "-m", "local commit"); err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}
	if _, err = testutil.RunGit(t, dir, "fetch", "origin"); err != nil {
		t.Fatalf("Failed to fetch: %v", err)
	}

	repo := openRepo(t, dir)
	status, count, err := repo.CompareBranchWithRemote("feature/test")
	if err != nil {
		t.Fatalf("CompareBranchWithRemote returned unexpected error: %v", err)
	}
	if status != git.SyncStatusDiverged {
		t.Errorf("Expected status SyncStatusDiverged, got %s", status)
	}
	if count != 2 {
		t.Errorf("Expected count 2 (1 ahead + 1 behind), got %d", count)
	}
}

func TestFetchBranch(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	remoteDir, err := testutil.AddRemote(t, dir, "origin", true)
	if err != nil {
		t.Fatalf("Failed to add remote: %v", err)
	}
	defer testutil.CleanupTestRepo(t, remoteDir)

	initialHash, err := testutil.RunGit(t, dir, "rev-parse", "origin/main")
	if err != nil {
		t.Fatalf("Failed to get initial hash: %v", err)
	}

	secondDir := t.TempDir()
	if _, err = testutil.RunGit(t, secondDir, "clone", remoteDir, "."); err != nil {
		t.Fatalf("Failed to clone: %v", err)
	}
	testutil.ConfigureGitIdentity(t, secondDir)
	testutil.WriteFile(t, secondDir, "remote-change.txt", "remote content")
	if _, err = testutil.RunGit(t, secondDir, "add", "remote-change.txt"); err != nil {
		t.Fatalf("Failed to add file in second repo: %v", err)
	}
	if _, err = testutil.RunGit(t, secondDir, "commit", "-m", "remote commit"); err != nil {
		t.Fatalf("Failed to commit in second repo: %v", err)
	}
	if _, err = testutil.RunGit(t, secondDir, "push", "origin", "main"); err != nil {
		t.Fatalf("Failed to push from second repo: %v", err)
	}

	repo := openRepo(t, dir)
	if err := repo.FetchBranch("origin", "main"); err != nil {
		t.Fatalf("FetchBranch returned unexpected error: %v", err)
	}

	newHash, err := testutil.RunGit(t, dir, "rev-parse", "origin/main")
	if err != nil {
		t.Fatalf("Failed to get new hash: %v", err)
	}
	if initialHash == newHash {
		t.Error("Expected origin/main to be updated after fetch, but hash unchanged")
	}
}

func TestFetchBranchNonExistent(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	remoteDir, err := testutil.AddRemote(t, dir, "origin", true)
	if err != nil {
		t.Fatalf("Failed to add remote: %v", err)
	}
	defer testutil.CleanupTestRepo(t, remoteDir)

	repo := openRepo(t, dir)
	err = repo.FetchBranch("origin", "non-existent-branch")
	if err == nil {
		t.Fatal("Expected error when fetching non-existent branch, got nil")
	}
	if !goerrors.Is(err, git.ErrRemoteRefNotFound) {
		t.Errorf("Expected error to wrap ErrRemoteRefNotFound, got: %v", err)
	}
}

func TestFetchBranchTransportFailure(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	bogusRemote := filepath.Join(dir, "does-not-exist-remote.git")

	repo := openRepo(t, dir)
	err := repo.FetchBranch(bogusRemote, "main")
	if err == nil {
		t.Fatal("Expected error when fetching from a non-existent remote, got nil")
	}
	if goerrors.Is(err, git.ErrRemoteRefNotFound) {
		t.Errorf("Expected transport failure NOT to wrap ErrRemoteRefNotFound, got: %v", err)
	}
}

func TestIsGitMergeInProgressTrue(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	setupConflictingBranches(t, dir)
	_, _ = testutil.RunGit(t, dir, "merge", "feature")

	repo := openRepo(t, dir)
	if !repo.IsGitMergeInProgress() {
		t.Error("Expected IsGitMergeInProgress to return true during merge conflict")
	}
}

func TestIsGitMergeInProgressFalse(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	repo := openRepo(t, dir)
	if repo.IsGitMergeInProgress() {
		t.Error("Expected IsGitMergeInProgress to return false on clean repo")
	}
}

func TestIsGitRebaseInProgressTrue(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	setupConflictingBranches(t, dir)
	if _, err := testutil.RunGit(t, dir, "checkout", "feature"); err != nil {
		t.Fatalf("Failed to checkout feature: %v", err)
	}
	_, _ = testutil.RunGit(t, dir, "rebase", "main")

	repo := openRepo(t, dir)
	if !repo.IsGitRebaseInProgress() {
		t.Error("Expected IsGitRebaseInProgress to return true during rebase conflict")
	}
}

func TestIsGitRebaseInProgressFalse(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	repo := openRepo(t, dir)
	if repo.IsGitRebaseInProgress() {
		t.Error("Expected IsGitRebaseInProgress to return false on clean repo")
	}
}

func TestIsGitSquashMergeInProgressTrue(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	setupConflictingBranches(t, dir)
	_, _ = testutil.RunGit(t, dir, "merge", "--squash", "feature")

	repo := openRepo(t, dir)
	if !repo.IsGitSquashMergeInProgress() {
		t.Error("Expected IsGitSquashMergeInProgress to return true during squash merge conflict")
	}
}

func TestIsGitSquashMergeInProgressFalse(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	repo := openRepo(t, dir)
	if repo.IsGitSquashMergeInProgress() {
		t.Error("Expected IsGitSquashMergeInProgress to return false on clean repo")
	}
}

// --- Scenario 4: in-progress detection off-CWD, with setup guards ---

func TestRepoIsMergeInProgressTrueOffCwd(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	setupConflictingBranches(t, dir)
	_, _ = testutil.RunGit(t, dir, "merge", "feature")

	// Setup guard: confirm the conflict is genuinely in progress before asserting.
	if _, err := os.Stat(filepath.Join(dir, ".git", "MERGE_HEAD")); err != nil {
		t.Fatalf("Setup guard failed: MERGE_HEAD not present: %v", err)
	}
	status, err := testutil.RunGit(t, dir, "status", "--porcelain")
	if err != nil {
		t.Fatalf("Failed to read status: %v", err)
	}
	if !hasUnmergedEntry(status) {
		t.Fatalf("Setup guard failed: expected a UU entry in status, got:\n%s", status)
	}

	repo := openRepo(t, dir)
	if !repo.IsGitMergeInProgress() {
		t.Error("Expected IsGitMergeInProgress to return true off-CWD")
	}
}

func TestRepoIsRebaseInProgressTrueOffCwd(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	setupConflictingBranches(t, dir)
	if _, err := testutil.RunGit(t, dir, "checkout", "feature"); err != nil {
		t.Fatalf("Failed to checkout feature: %v", err)
	}
	_, _ = testutil.RunGit(t, dir, "rebase", "main")

	// Setup guard: a rebase state dir must exist.
	_, mergeErr := os.Stat(filepath.Join(dir, ".git", "rebase-merge"))
	_, applyErr := os.Stat(filepath.Join(dir, ".git", "rebase-apply"))
	if mergeErr != nil && applyErr != nil {
		t.Fatalf("Setup guard failed: neither rebase-merge nor rebase-apply present")
	}

	repo := openRepo(t, dir)
	if !repo.IsGitRebaseInProgress() {
		t.Error("Expected IsGitRebaseInProgress to return true off-CWD")
	}
}

func TestRepoIsSquashMergeInProgressTrueOffCwd(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	setupConflictingBranches(t, dir)
	_, _ = testutil.RunGit(t, dir, "merge", "--squash", "feature")

	// Setup guard: SQUASH_MSG present and no rebase-merge (squash, not rebase).
	if _, err := os.Stat(filepath.Join(dir, ".git", "SQUASH_MSG")); err != nil {
		t.Fatalf("Setup guard failed: SQUASH_MSG not present: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, ".git", "rebase-merge")); err == nil {
		t.Fatalf("Setup guard failed: unexpected rebase-merge present for squash")
	}

	repo := openRepo(t, dir)
	if !repo.IsGitSquashMergeInProgress() {
		t.Error("Expected IsGitSquashMergeInProgress to return true off-CWD")
	}
}

func TestRepoInProgressChecksIsolatedAcrossRepos(t *testing.T) {
	t.Parallel()
	repoA := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, repoA)
	repoB := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, repoB)

	// Repo A has a merge in progress; repo B is clean.
	setupConflictingBranches(t, repoA)
	_, _ = testutil.RunGit(t, repoA, "merge", "feature")
	if _, err := os.Stat(filepath.Join(repoA, ".git", "MERGE_HEAD")); err != nil {
		t.Fatalf("Setup guard failed: A has no MERGE_HEAD: %v", err)
	}

	repo := openRepo(t, repoB)
	if repo.IsGitMergeInProgress() {
		t.Error("B reported a merge in progress; A's state leaked")
	}
	if repo.IsGitRebaseInProgress() {
		t.Error("B reported a rebase in progress; A's state leaked")
	}
	if repo.IsGitSquashMergeInProgress() {
		t.Error("B reported a squash merge in progress; A's state leaked")
	}
}

// hasUnmergedEntry reports whether git status --porcelain output contains an
// unmerged (UU) entry.
func hasUnmergedEntry(status string) bool {
	for _, line := range splitLines(status) {
		if len(line) >= 2 && line[0] == 'U' && line[1] == 'U' {
			return true
		}
	}
	return false
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

// =============================================================================
// IsAncestor (issue #210): the read-only ancestry query behind the finish
// --ff-only gate.
// =============================================================================

// setupAncestryRepo seeds commits c1 and c2 on the current branch and a second
// branch "side" pointing at c1. It leaves the repository checked out on its
// initial branch with a clean work tree.
func setupAncestryRepo(t *testing.T, dir string) {
	t.Helper()
	testutil.WriteFile(t, dir, "c1.txt", "c1")
	if _, err := testutil.RunGit(t, dir, "add", "c1.txt"); err != nil {
		t.Fatalf("Failed to add c1.txt: %v", err)
	}
	if _, err := testutil.RunGit(t, dir, "commit", "-m", "c1"); err != nil {
		t.Fatalf("Failed to commit c1: %v", err)
	}
	if _, err := testutil.RunGit(t, dir, "branch", "side"); err != nil {
		t.Fatalf("Failed to create side branch: %v", err)
	}
	testutil.WriteFile(t, dir, "c2.txt", "c2")
	if _, err := testutil.RunGit(t, dir, "add", "c2.txt"); err != nil {
		t.Fatalf("Failed to add c2.txt: %v", err)
	}
	if _, err := testutil.RunGit(t, dir, "commit", "-m", "c2"); err != nil {
		t.Fatalf("Failed to commit c2: %v", err)
	}
}

// TestIsAncestorTrueForEqualRefs verifies IsAncestor reports true when both refs
// name the same commit — the "gate passes when the branches are equal" rule the
// finish --ff-only feature rests on.
// Steps:
// 1. Creates a repository with commits c1 and c2 plus a side branch at c1
// 2. Captures HEAD and the porcelain status
// 3. Calls IsAncestor with the same ref on both sides
// 4. Verifies it returns (true, nil)
// 5. Verifies HEAD and the work-tree status are unchanged (the helper is read-only)
func TestIsAncestorTrueForEqualRefs(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	setupAncestryRepo(t, dir)

	repo := openRepo(t, dir)
	headBefore, err := testutil.RunGit(t, dir, "rev-parse", "HEAD")
	if err != nil {
		t.Fatalf("Failed to read HEAD: %v", err)
	}
	statusBefore, err := testutil.RunGit(t, dir, "status", "--porcelain")
	if err != nil {
		t.Fatalf("Failed to read status: %v", err)
	}

	ok, err := repo.IsAncestor("side", "side")
	if err != nil {
		t.Fatalf("IsAncestor returned an error for identical refs: %v", err)
	}
	if !ok {
		t.Error("Expected IsAncestor to report true for identical refs")
	}

	headAfter, err := testutil.RunGit(t, dir, "rev-parse", "HEAD")
	if err != nil {
		t.Fatalf("Failed to re-read HEAD: %v", err)
	}
	if headAfter != headBefore {
		t.Errorf("Expected HEAD unchanged. Before: %s After: %s", headBefore, headAfter)
	}
	statusAfter, err := testutil.RunGit(t, dir, "status", "--porcelain")
	if err != nil {
		t.Fatalf("Failed to re-read status: %v", err)
	}
	if statusAfter != statusBefore {
		t.Errorf("Expected work-tree status unchanged. Before: %q After: %q", statusBefore, statusAfter)
	}
}

// TestIsAncestorErrorsOnUnknownRef verifies an unknown ref surfaces as an error
// rather than being collapsed into the boolean. git exits 128 here, which must
// never be read as "not an ancestor".
// Steps:
// 1. Creates a repository with commits c1 and c2 plus a side branch at c1
// 2. Captures HEAD and the porcelain status
// 3. Calls IsAncestor with a ref that does not exist
// 4. Verifies a non-nil error is returned (and not a bare false)
// 5. Verifies HEAD and the work-tree status are unchanged
func TestIsAncestorErrorsOnUnknownRef(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	setupAncestryRepo(t, dir)

	repo := openRepo(t, dir)
	headBefore, err := testutil.RunGit(t, dir, "rev-parse", "HEAD")
	if err != nil {
		t.Fatalf("Failed to read HEAD: %v", err)
	}
	statusBefore, err := testutil.RunGit(t, dir, "status", "--porcelain")
	if err != nil {
		t.Fatalf("Failed to read status: %v", err)
	}

	ok, err := repo.IsAncestor("no-such-ref", "side")
	if err == nil {
		t.Fatalf("Expected an error for an unknown ref, got (%v, nil)", ok)
	}

	headAfter, err := testutil.RunGit(t, dir, "rev-parse", "HEAD")
	if err != nil {
		t.Fatalf("Failed to re-read HEAD: %v", err)
	}
	if headAfter != headBefore {
		t.Errorf("Expected HEAD unchanged. Before: %s After: %s", headBefore, headAfter)
	}
	statusAfter, err := testutil.RunGit(t, dir, "status", "--porcelain")
	if err != nil {
		t.Fatalf("Failed to re-read status: %v", err)
	}
	if statusAfter != statusBefore {
		t.Errorf("Expected work-tree status unchanged. Before: %q After: %q", statusBefore, statusAfter)
	}
}
