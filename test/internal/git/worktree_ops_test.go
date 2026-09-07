package git_test

import (
	"fmt"
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
	// Registered as a defer, not t.Cleanup: defers run LIFO before any t.Cleanup,
	// so this one runs while the repository still exists, whereas CleanupTestRepo
	// above (registered first, so it runs last) has by then removed it. That
	// ordering is what lets the unlock failure be reported rather than discarded.
	defer func() {
		if out, err := testutil.RunGit(t, dir, "worktree", "unlock", wtPath); err != nil {
			t.Errorf("Failed to unlock the worktree: %v\nOutput: %s", err, out)
		}
	}()

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

// TestWorktreeOperationInProgressDetectsMerge verifies a conflicted merge
// inside a worktree is reported as "merge" in progress.
// Steps:
// 1. Creates a repository with feature/x and a linked worktree for it
// 2. Commits conflicting content to the same file on main and on the worktree
// 3. Merges main into the worktree by hand, producing a conflict
// 4. Calls repo.WorktreeOperationInProgress on the worktree path
// 5. Verifies it reports "merge" in progress with no error
func TestWorktreeOperationInProgressDetectsMerge(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	repo, wtPath := setupWorktreeRepo(t, dir, "feature/x")

	if err := os.WriteFile(filepath.Join(wtPath, "README.md"), []byte("from feature"), 0644); err != nil {
		t.Fatalf("Failed to write conflicting content on the worktree: %v", err)
	}
	if out, err := testutil.RunGit(t, wtPath, "commit", "-am", "feature change"); err != nil {
		t.Fatalf("Failed to commit on the worktree: %v\nOutput: %s", err, out)
	}
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("from main"), 0644); err != nil {
		t.Fatalf("Failed to write conflicting content on main: %v", err)
	}
	if out, err := testutil.RunGit(t, dir, "commit", "-am", "main change"); err != nil {
		t.Fatalf("Failed to commit on main: %v\nOutput: %s", err, out)
	}

	if out, err := testutil.RunGit(t, wtPath, "merge", "main"); err == nil {
		t.Fatalf("Expected the merge to conflict, but it succeeded: %s", out)
	}

	label, inProgress, err := repo.WorktreeOperationInProgress(wtPath)
	if err != nil {
		t.Fatalf("WorktreeOperationInProgress failed: %v", err)
	}
	if !inProgress {
		t.Fatal("Expected an operation to be reported in progress")
	}
	if label != "merge" {
		t.Errorf("Expected label 'merge', got %q", label)
	}
}

// TestWorktreeOperationInProgressDetectsRebase verifies a conflicted rebase
// inside a worktree is reported as "rebase" in progress.
// Steps:
// 1. Creates a repository with feature/x (one commit ahead) and a linked worktree for it
// 2. Commits conflicting content to the same file on main
// 3. Starts 'git rebase main' by hand inside the worktree, producing a conflict
// 4. Calls repo.WorktreeOperationInProgress on the worktree path
// 5. Verifies it reports "rebase" in progress with no error
func TestWorktreeOperationInProgressDetectsRebase(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	repo, wtPath := setupWorktreeRepo(t, dir, "feature/x")

	if err := os.WriteFile(filepath.Join(wtPath, "README.md"), []byte("from feature"), 0644); err != nil {
		t.Fatalf("Failed to write conflicting content on the worktree: %v", err)
	}
	if out, err := testutil.RunGit(t, wtPath, "commit", "-am", "feature change"); err != nil {
		t.Fatalf("Failed to commit on the worktree: %v\nOutput: %s", err, out)
	}
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("from main"), 0644); err != nil {
		t.Fatalf("Failed to write conflicting content on main: %v", err)
	}
	if out, err := testutil.RunGit(t, dir, "commit", "-am", "main change"); err != nil {
		t.Fatalf("Failed to commit on main: %v\nOutput: %s", err, out)
	}

	if out, err := testutil.RunGit(t, wtPath, "rebase", "main"); err == nil {
		t.Fatalf("Expected the rebase to conflict, but it succeeded: %s", out)
	}

	label, inProgress, err := repo.WorktreeOperationInProgress(wtPath)
	if err != nil {
		t.Fatalf("WorktreeOperationInProgress failed: %v", err)
	}
	if !inProgress {
		t.Fatal("Expected an operation to be reported in progress")
	}
	if label != "rebase" {
		t.Errorf("Expected label 'rebase', got %q", label)
	}
}

// TestWorktreeOperationInProgressDetectsBisect covers the bisect marker,
// which the merge/rebase tests above don't exercise. Four commits are used
// (not two) so bisect has a midpoint left to test after 'good'/'bad' are
// given, rather than immediately concluding and cleaning up BISECT_LOG on its
// own.
// Steps:
// 1. Creates a worktree with four commits
// 2. Starts a bisect there, marking the tip bad and the oldest commit good
// 3. Verifies WorktreeOperationInProgress reports ("bisect", true, nil)
func TestWorktreeOperationInProgressDetectsBisect(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	repo, wtPath := setupWorktreeRepo(t, dir, "feature/x")

	if err := os.WriteFile(filepath.Join(wtPath, "README.md"), []byte("commit 1"), 0644); err != nil {
		t.Fatalf("Failed to write commit 1 content: %v", err)
	}
	if out, err := testutil.RunGit(t, wtPath, "commit", "-am", "commit 1"); err != nil {
		t.Fatalf("Failed to create commit 1: %v\nOutput: %s", err, out)
	}
	goodRev, err := testutil.RunGit(t, wtPath, "rev-parse", "HEAD")
	if err != nil {
		t.Fatalf("Failed to resolve commit 1: %v", err)
	}
	goodRev = strings.TrimSpace(goodRev)
	for i := 2; i <= 4; i++ {
		content := fmt.Sprintf("commit %d", i)
		if err := os.WriteFile(filepath.Join(wtPath, "README.md"), []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write %s content: %v", content, err)
		}
		if out, err := testutil.RunGit(t, wtPath, "commit", "-am", content); err != nil {
			t.Fatalf("Failed to create %s: %v\nOutput: %s", content, err, out)
		}
	}

	if out, err := testutil.RunGit(t, wtPath, "bisect", "start"); err != nil {
		t.Fatalf("Failed to start bisect: %v\nOutput: %s", err, out)
	}
	if out, err := testutil.RunGit(t, wtPath, "bisect", "bad", "HEAD"); err != nil {
		t.Fatalf("Failed to mark HEAD bad: %v\nOutput: %s", err, out)
	}
	if out, err := testutil.RunGit(t, wtPath, "bisect", "good", goodRev); err != nil {
		t.Fatalf("Failed to mark commit 1 good: %v\nOutput: %s", err, out)
	}

	label, inProgress, err := repo.WorktreeOperationInProgress(wtPath)
	if err != nil {
		t.Fatalf("WorktreeOperationInProgress failed: %v", err)
	}
	if !inProgress {
		t.Fatal("Expected an operation to be reported in progress")
	}
	if label != "bisect" {
		t.Errorf("Expected label 'bisect', got %q", label)
	}
}

// TestWorktreeOperationInProgressDetectsCherryPick covers the cherry-pick
// marker, added alongside merge/rebase/bisect for #175.
// Steps:
// 1. Creates a worktree and a diverging, conflicting commit on main
// 2. Starts a conflicting 'git cherry-pick' of main's commit into the worktree, leaving it unresolved
// 3. Verifies WorktreeOperationInProgress reports ("cherry-pick", true, nil)
func TestWorktreeOperationInProgressDetectsCherryPick(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	repo, wtPath := setupWorktreeRepo(t, dir, "feature/x")

	if err := os.WriteFile(filepath.Join(wtPath, "README.md"), []byte("from feature"), 0644); err != nil {
		t.Fatalf("Failed to write conflicting content on the worktree: %v", err)
	}
	if out, err := testutil.RunGit(t, wtPath, "commit", "-am", "feature change"); err != nil {
		t.Fatalf("Failed to commit on the worktree: %v\nOutput: %s", err, out)
	}
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("from main"), 0644); err != nil {
		t.Fatalf("Failed to write conflicting content on main: %v", err)
	}
	if out, err := testutil.RunGit(t, dir, "commit", "-am", "main change"); err != nil {
		t.Fatalf("Failed to commit on main: %v\nOutput: %s", err, out)
	}
	mainRev, err := testutil.RunGit(t, dir, "rev-parse", "HEAD")
	if err != nil {
		t.Fatalf("Failed to resolve main's tip: %v", err)
	}

	if out, err := testutil.RunGit(t, wtPath, "cherry-pick", strings.TrimSpace(mainRev)); err == nil {
		t.Fatalf("Expected the cherry-pick to conflict, but it succeeded: %s", out)
	}

	label, inProgress, err := repo.WorktreeOperationInProgress(wtPath)
	if err != nil {
		t.Fatalf("WorktreeOperationInProgress failed: %v", err)
	}
	if !inProgress {
		t.Fatal("Expected an operation to be reported in progress")
	}
	if label != "cherry-pick" {
		t.Errorf("Expected label 'cherry-pick', got %q", label)
	}
}

// TestWorktreeOperationInProgressDetectsRevert covers the revert marker,
// added alongside merge/rebase/bisect for #175.
// Steps:
// 1. Creates a worktree, commits a change, then commits a second change touching the same content
// 2. Starts a conflicting 'git revert' of the first commit, leaving it unresolved
// 3. Verifies WorktreeOperationInProgress reports ("revert", true, nil)
func TestWorktreeOperationInProgressDetectsRevert(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	repo, wtPath := setupWorktreeRepo(t, dir, "feature/x")

	if err := os.WriteFile(filepath.Join(wtPath, "README.md"), []byte("first change"), 0644); err != nil {
		t.Fatalf("Failed to write the first change: %v", err)
	}
	if out, err := testutil.RunGit(t, wtPath, "commit", "-am", "first change"); err != nil {
		t.Fatalf("Failed to commit the first change: %v\nOutput: %s", err, out)
	}
	revertTarget, err := testutil.RunGit(t, wtPath, "rev-parse", "HEAD")
	if err != nil {
		t.Fatalf("Failed to resolve the commit to revert: %v", err)
	}
	if err := os.WriteFile(filepath.Join(wtPath, "README.md"), []byte("second change"), 0644); err != nil {
		t.Fatalf("Failed to write the second change: %v", err)
	}
	if out, err := testutil.RunGit(t, wtPath, "commit", "-am", "second change"); err != nil {
		t.Fatalf("Failed to commit the second change: %v\nOutput: %s", err, out)
	}

	if out, err := testutil.RunGit(t, wtPath, "revert", "--no-edit", strings.TrimSpace(revertTarget)); err == nil {
		t.Fatalf("Expected the revert to conflict, but it succeeded: %s", out)
	}

	label, inProgress, err := repo.WorktreeOperationInProgress(wtPath)
	if err != nil {
		t.Fatalf("WorktreeOperationInProgress failed: %v", err)
	}
	if !inProgress {
		t.Fatal("Expected an operation to be reported in progress")
	}
	if label != "revert" {
		t.Errorf("Expected label 'revert', got %q", label)
	}
}

// TestWorktreeOperationInProgressReportsCleanWorktree verifies a worktree with
// no merge, rebase, or bisect underway reports nothing in progress.
// Steps:
// 1. Creates a repository with feature/x and a linked worktree for it
// 2. Calls repo.WorktreeOperationInProgress on the worktree path
// 3. Verifies it reports no operation in progress and no error
func TestWorktreeOperationInProgressReportsCleanWorktree(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	repo, wtPath := setupWorktreeRepo(t, dir, "feature/x")

	label, inProgress, err := repo.WorktreeOperationInProgress(wtPath)
	if err != nil {
		t.Fatalf("WorktreeOperationInProgress failed: %v", err)
	}
	if inProgress {
		t.Errorf("Expected no operation in progress, got %q", label)
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

// TestWorktreeChangeCountCountsPorcelainEntries verifies the change count is the
// number of 'git status --porcelain' entries in a worktree.
// Steps:
// 1. Creates a repository with feature/x and a linked worktree for it
// 2. Modifies a tracked file and adds one untracked top-level file inside it
// 3. Calls repo.WorktreeChangeCount on the worktree path
// 4. Verifies it returns 2 without an error
func TestWorktreeChangeCountCountsPorcelainEntries(t *testing.T) {
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

	count, err := repo.WorktreeChangeCount(wtPath)
	if err != nil {
		t.Fatalf("WorktreeChangeCount failed: %v", err)
	}
	if count != 2 {
		t.Errorf("Expected 2 changed entries, got %d", count)
	}
}

// TestWorktreeChangeCountIsZeroForCleanWorktree verifies a clean worktree counts
// zero entries, and specifically not one from a trailing newline.
// Steps:
// 1. Creates a repository with feature/x and a linked worktree for it
// 2. Writes nothing into the worktree
// 3. Calls repo.WorktreeChangeCount on the worktree path
// 4. Verifies it returns 0 without an error
func TestWorktreeChangeCountIsZeroForCleanWorktree(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	repo, wtPath := setupWorktreeRepo(t, dir, "feature/x")

	count, err := repo.WorktreeChangeCount(wtPath)
	if err != nil {
		t.Fatalf("WorktreeChangeCount failed: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected 0 changed entries, got %d", count)
	}
}

// TestWorktreeChangeCountFailsForMissingDirectory verifies the count fails when
// the worktree directory is gone, which is why callers stat the directory first
// rather than reading a missing worktree as clean.
// Steps:
// 1. Creates a repository with feature/x and a linked worktree for it
// 2. Removes the worktree directory
// 3. Calls repo.WorktreeChangeCount on the vanished path
// 4. Verifies it returns an error naming the path
func TestWorktreeChangeCountFailsForMissingDirectory(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	repo, wtPath := setupWorktreeRepo(t, dir, "feature/x")

	if err := os.RemoveAll(wtPath); err != nil {
		t.Fatalf("Failed to remove the worktree directory: %v", err)
	}

	if _, err := repo.WorktreeChangeCount(wtPath); err == nil {
		t.Fatal("Expected WorktreeChangeCount to fail for a missing directory")
	} else if !strings.Contains(err.Error(), wtPath) {
		t.Errorf("Expected the error to name %q, got: %v", wtPath, err)
	}
}
