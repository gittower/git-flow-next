package git_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gittower/git-flow-next/internal/git"
	"github.com/gittower/git-flow-next/test/testutil"
)

// setupWorktreeRepo creates a repository with a free branch and a linked
// worktree for it, created through the repo handle. It returns the handle and
// the SYMLINK-RESOLVED worktree path, so comparisons against the paths git
// reports hold on macOS (/var vs /private/var).
func setupWorktreeRepo(t *testing.T, dir string, branch string) (*git.Repo, string) {
	t.Helper()
	if out, err := testutil.RunGit(t, dir, "branch", branch); err != nil {
		t.Fatalf("Failed to create branch %s: %v\nOutput: %s", branch, err, out)
	}
	repo := openRepo(t, dir)
	wtPath := filepath.Join(t.TempDir(), "linked-worktree")
	if err := repo.AddWorktree(wtPath, branch); err != nil {
		t.Fatalf("AddWorktree failed: %v", err)
	}
	return repo, testutil.EvalPath(t, wtPath)
}

// findWorktreeEntry returns the entry whose path matches wtPath, or nil.
func findWorktreeEntry(entries []git.WorktreeEntry, wtPath string) *git.WorktreeEntry {
	for i := range entries {
		if entries[i].Path == wtPath {
			return &entries[i]
		}
	}
	return nil
}

// TestDetachWorktreeKeepsChanges covers scenario 21: detaching a worktree's HEAD
// leaves the tree and its uncommitted work untouched and frees the branch.
// Steps:
// 1. Creates a repository with feature/x and a linked worktree for it
// 2. Modifies a tracked file and adds an untracked file inside the worktree
// 3. Calls repo.DetachWorktree on the worktree path
// 4. Verifies HEAD is unchanged and detached, both changes survive, and feature/x can be checked out again
func TestDetachWorktreeKeepsChanges(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	repo, wtPath := setupWorktreeRepo(t, dir, "feature/x")

	if err := os.WriteFile(filepath.Join(wtPath, "README.md"), []byte("modified"), 0644); err != nil {
		t.Fatalf("Failed to modify tracked file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(wtPath, "scratch.txt"), []byte("scratch"), 0644); err != nil {
		t.Fatalf("Failed to write untracked file: %v", err)
	}

	headBefore, err := testutil.RunGit(t, wtPath, "rev-parse", "HEAD")
	if err != nil {
		t.Fatalf("Failed to read HEAD: %v", err)
	}

	if err := repo.DetachWorktree(wtPath); err != nil {
		t.Fatalf("DetachWorktree failed: %v", err)
	}

	headAfter, err := testutil.RunGit(t, wtPath, "rev-parse", "HEAD")
	if err != nil {
		t.Fatalf("Failed to read HEAD after detach: %v", err)
	}
	if strings.TrimSpace(headAfter) != strings.TrimSpace(headBefore) {
		t.Errorf("Expected HEAD to stay at %q, got %q", strings.TrimSpace(headBefore), strings.TrimSpace(headAfter))
	}
	if _, err := testutil.RunGit(t, wtPath, "symbolic-ref", "-q", "HEAD"); err == nil {
		t.Error("Expected HEAD to be detached (symbolic-ref should fail)")
	}
	content, err := os.ReadFile(filepath.Join(wtPath, "README.md"))
	if err != nil || string(content) != "modified" {
		t.Errorf("Expected the modification to survive, got %q (%v)", string(content), err)
	}
	if _, err := os.Stat(filepath.Join(wtPath, "scratch.txt")); err != nil {
		t.Errorf("Expected the untracked file to survive: %v", err)
	}
	if out, err := testutil.RunGit(t, dir, "checkout", "feature/x"); err != nil {
		t.Errorf("Expected feature/x to be checked out again after detach: %v\nOutput: %s", err, out)
	}
}

// TestDetachWorktreeRefusesMainWorktree covers scenario 22: detach never acts on
// the main worktree.
// Steps:
// 1. Creates a repository and resolves its main worktree root
// 2. Calls repo.DetachWorktree on the main worktree
// 3. Verifies a non-nil error identifying the main worktree
// 4. Verifies the main worktree's HEAD is still a symbolic ref on main
func TestDetachWorktreeRefusesMainWorktree(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	repo := openRepo(t, dir)

	mainWorkTree, err := repo.MainWorkTree()
	if err != nil {
		t.Fatalf("MainWorkTree failed: %v", err)
	}

	err = repo.DetachWorktree(mainWorkTree)
	if err == nil {
		t.Fatal("Expected DetachWorktree to refuse the main worktree")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "main worktree") {
		t.Errorf("Expected the error to identify the main worktree, got: %v", err)
	}

	ref, err := testutil.RunGit(t, dir, "symbolic-ref", "HEAD")
	if err != nil {
		t.Fatalf("Expected the main worktree HEAD to stay symbolic: %v", err)
	}
	if strings.TrimSpace(ref) != "refs/heads/main" {
		t.Errorf("Expected HEAD to stay on refs/heads/main, got %q", strings.TrimSpace(ref))
	}
}

// TestListWorktreesParsesPorcelainRecords verifies the porcelain parser returns
// the main worktree first and a fully populated linked entry.
// Steps:
// 1. Creates a repository with feature/x and a linked worktree for it
// 2. Calls repo.ListWorktrees
// 3. Verifies the first entry is the main worktree, flagged Main
// 4. Verifies the linked entry carries Branch feature/x and an absolute path
func TestListWorktreesParsesPorcelainRecords(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	repo, wtPath := setupWorktreeRepo(t, dir, "feature/x")

	entries, err := repo.ListWorktrees()
	if err != nil {
		t.Fatalf("ListWorktrees failed: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("Expected 2 entries, got %d: %+v", len(entries), entries)
	}
	if !entries[0].Main {
		t.Errorf("Expected the first entry to be the main worktree, got %+v", entries[0])
	}
	if entries[0].Path != testutil.EvalPath(t, dir) {
		t.Errorf("Expected the main entry path %q, got %q", testutil.EvalPath(t, dir), entries[0].Path)
	}

	linked := findWorktreeEntry(entries, wtPath)
	if linked == nil {
		t.Fatalf("Expected an entry for %q, got %+v", wtPath, entries)
	}
	if linked.Branch != "feature/x" {
		t.Errorf("Expected Branch feature/x, got %q", linked.Branch)
	}
	if !filepath.IsAbs(linked.Path) {
		t.Errorf("Expected an absolute path, got %q", linked.Path)
	}
	if linked.Main {
		t.Error("Expected the linked entry not to be flagged Main")
	}
}

// TestListWorktreesIgnoresUnknownPorcelainLines verifies the parser tolerates
// annotations it does not know, using a real 'locked' record.
// Steps:
// 1. Creates a repository with feature/x and a linked worktree for it
// 2. Runs 'git worktree lock' so git emits a locked annotation in the porcelain record
// 3. Calls repo.ListWorktrees
// 4. Verifies no error and that the entry still parses with its branch and path
func TestListWorktreesIgnoresUnknownPorcelainLines(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	repo, wtPath := setupWorktreeRepo(t, dir, "feature/x")

	if out, err := testutil.RunGit(t, dir, "worktree", "lock", wtPath); err != nil {
		t.Fatalf("git worktree lock failed: %v\nOutput: %s", err, out)
	}
	t.Cleanup(func() { _, _ = testutil.RunGit(t, dir, "worktree", "unlock", wtPath) })

	entries, err := repo.ListWorktrees()
	if err != nil {
		t.Fatalf("ListWorktrees failed on a record with an unknown annotation: %v", err)
	}
	linked := findWorktreeEntry(entries, wtPath)
	if linked == nil {
		t.Fatalf("Expected an entry for %q, got %+v", wtPath, entries)
	}
	if linked.Branch != "feature/x" {
		t.Errorf("Expected Branch feature/x, got %q", linked.Branch)
	}
}

// TestListWorktreesReportsDetachedEntry verifies a detached worktree is reported
// as detached with no branch.
// Steps:
// 1. Creates a repository with feature/x and a linked worktree for it
// 2. Detaches the worktree's HEAD
// 3. Calls repo.ListWorktrees
// 4. Verifies the entry has Detached true and an empty Branch
func TestListWorktreesReportsDetachedEntry(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	repo, wtPath := setupWorktreeRepo(t, dir, "feature/x")

	if out, err := testutil.RunGit(t, wtPath, "checkout", "--detach"); err != nil {
		t.Fatalf("Failed to detach worktree HEAD: %v\nOutput: %s", err, out)
	}

	entries, err := repo.ListWorktrees()
	if err != nil {
		t.Fatalf("ListWorktrees failed: %v", err)
	}
	linked := findWorktreeEntry(entries, wtPath)
	if linked == nil {
		t.Fatalf("Expected an entry for %q, got %+v", wtPath, entries)
	}
	if !linked.Detached {
		t.Error("Expected the entry to be flagged Detached")
	}
	if linked.Branch != "" {
		t.Errorf("Expected an empty Branch for a detached entry, got %q", linked.Branch)
	}
}

// TestWorktreeForBranchFindsLinkedWorktree verifies the branch lookup returns the
// linked worktree holding the branch.
// Steps:
// 1. Creates a repository with feature/x and a linked worktree for it
// 2. Calls repo.WorktreeForBranch("feature/x")
// 3. Verifies a non-nil entry is returned
// 4. Verifies its path is the linked worktree and it is not the main worktree
func TestWorktreeForBranchFindsLinkedWorktree(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	repo, wtPath := setupWorktreeRepo(t, dir, "feature/x")

	entry, err := repo.WorktreeForBranch("feature/x")
	if err != nil {
		t.Fatalf("WorktreeForBranch failed: %v", err)
	}
	if entry == nil {
		t.Fatal("Expected an entry for feature/x")
	}
	if entry.Path != wtPath {
		t.Errorf("Expected path %q, got %q", wtPath, entry.Path)
	}
	if entry.Main {
		t.Error("Expected the linked worktree, not the main one")
	}
}

// TestWorktreeForBranchReturnsNilForBranchWithoutWorktree verifies that absence
// is not an error.
// Steps:
// 1. Creates a repository with a branch feature/lonely and no worktree for it
// 2. Calls repo.WorktreeForBranch("feature/lonely")
// 3. Verifies a nil error is returned
// 4. Verifies a nil entry is returned
func TestWorktreeForBranchReturnsNilForBranchWithoutWorktree(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	if out, err := testutil.RunGit(t, dir, "branch", "feature/lonely"); err != nil {
		t.Fatalf("Failed to create branch: %v\nOutput: %s", err, out)
	}
	repo := openRepo(t, dir)

	entry, err := repo.WorktreeForBranch("feature/lonely")
	if err != nil {
		t.Fatalf("Expected absence to be reported without an error, got: %v", err)
	}
	if entry != nil {
		t.Errorf("Expected a nil entry, got %+v", entry)
	}
}

// TestWorktreeHasChangesDetectsUntrackedFile verifies the dirty check runs
// against the worktree path and counts untracked files.
// Steps:
// 1. Creates a repository with feature/x and a clean linked worktree for it
// 2. Calls repo.WorktreeHasChanges and expects false
// 3. Writes an untracked file inside the worktree
// 4. Calls repo.WorktreeHasChanges again and expects true
func TestWorktreeHasChangesDetectsUntrackedFile(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	repo, wtPath := setupWorktreeRepo(t, dir, "feature/x")

	dirty, err := repo.WorktreeHasChanges(wtPath)
	if err != nil {
		t.Fatalf("WorktreeHasChanges failed: %v", err)
	}
	if dirty {
		t.Error("Expected a freshly created worktree to be clean")
	}

	if err := os.WriteFile(filepath.Join(wtPath, "scratch.txt"), []byte("scratch"), 0644); err != nil {
		t.Fatalf("Failed to write untracked file: %v", err)
	}

	dirty, err = repo.WorktreeHasChanges(wtPath)
	if err != nil {
		t.Fatalf("WorktreeHasChanges failed: %v", err)
	}
	if !dirty {
		t.Error("Expected an untracked file to make the worktree dirty")
	}
}

// TestRemoveWorktreeRefusesMainWorktree verifies removal refuses the main
// worktree before invoking git.
// Steps:
// 1. Creates a repository and resolves its main worktree root
// 2. Calls repo.RemoveWorktree on the main worktree without force
// 3. Verifies a non-nil error identifying the main worktree
// 4. Verifies the repository directory survives
func TestRemoveWorktreeRefusesMainWorktree(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	repo := openRepo(t, dir)

	mainWorkTree, err := repo.MainWorkTree()
	if err != nil {
		t.Fatalf("MainWorkTree failed: %v", err)
	}

	if err := repo.RemoveWorktree(mainWorkTree, false); err == nil {
		t.Fatal("Expected RemoveWorktree to refuse the main worktree")
	} else if !strings.Contains(strings.ToLower(err.Error()), "main worktree") {
		t.Errorf("Expected the error to identify the main worktree, got: %v", err)
	}

	if info, err := os.Stat(dir); err != nil || !info.IsDir() {
		t.Errorf("Expected the repository directory to survive: %v", err)
	}
}

// TestMainWorkTreeFromLinkedWorktree verifies a handle opened inside a linked
// worktree still reports the main worktree root.
// Steps:
// 1. Creates a repository with feature/x and a linked worktree for it
// 2. Opens a repo handle inside the linked worktree
// 3. Verifies WorkTree() reports the linked worktree
// 4. Verifies MainWorkTree() reports the main worktree root
func TestMainWorkTreeFromLinkedWorktree(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	_, wtPath := setupWorktreeRepo(t, dir, "feature/x")

	linkedRepo := openRepo(t, wtPath)

	if got := testutil.EvalPath(t, linkedRepo.WorkTree()); got != testutil.EvalPath(t, wtPath) {
		t.Errorf("Expected WorkTree() %q, got %q", testutil.EvalPath(t, wtPath), got)
	}

	mainWorkTree, err := linkedRepo.MainWorkTree()
	if err != nil {
		t.Fatalf("MainWorkTree failed: %v", err)
	}
	if got := testutil.EvalPath(t, mainWorkTree); got != testutil.EvalPath(t, dir) {
		t.Errorf("Expected MainWorkTree() %q, got %q", testutil.EvalPath(t, dir), got)
	}
}
