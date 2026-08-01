package git_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gittower/git-flow-next/internal/git"
	"github.com/gittower/git-flow-next/test/testutil"
)

// evalSymlinks normalizes a path through filepath.EvalSymlinks so comparisons
// survive macOS's /var -> /private/var TMPDIR symlink. It falls back to the
// input on error (e.g. the path does not exist yet).
func evalSymlinks(t *testing.T, path string) string {
	t.Helper()
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return path
	}
	return resolved
}

// relFromCwd computes a genuinely relative path from the process working
// directory to target, so tests can exercise git.Open with a relative argument.
func relFromCwd(t *testing.T, target string) string {
	t.Helper()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	rel, err := filepath.Rel(cwd, target)
	if err != nil {
		t.Fatalf("Failed to compute relative path: %v", err)
	}
	if filepath.IsAbs(rel) {
		t.Fatalf("Expected relative path, got absolute: %s", rel)
	}
	return rel
}

// TestOpenTargetsGivenRepoNotCwd verifies the core CWD-independence property:
// git.Open(B) operates on repository B even though the process working directory
// is neither B nor the second repo A, and a mutation on B's handle does not touch A.
func TestOpenTargetsGivenRepoNotCwd(t *testing.T) {
	t.Parallel()

	repoA := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, repoA)
	repoB := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, repoB)

	repo, err := git.Open(repoB)
	if err != nil {
		t.Fatalf("git.Open(B) failed: %v", err)
	}

	if err := repo.CreateBranch("feature/only-in-b", "main"); err != nil {
		t.Fatalf("CreateBranch on B failed: %v", err)
	}

	// The branch must exist in B.
	if _, err := testutil.RunGit(t, repoB, "rev-parse", "--verify", "refs/heads/feature/only-in-b"); err != nil {
		t.Errorf("Expected feature/only-in-b to exist in B: %v", err)
	}

	// The branch must NOT exist in A (Open(B) targeted only B).
	if _, err := testutil.RunGit(t, repoA, "rev-parse", "--verify", "refs/heads/feature/only-in-b"); err == nil {
		t.Error("feature/only-in-b leaked into repo A; Open(B) affected the wrong repository")
	}
}

// TestOpenAccessorsAreAbsoluteFromNestedSubdir verifies that opening a repo via a
// genuinely relative path to a nested subdirectory yields absolute accessors that
// resolve to the target repository's root and git dir.
func TestOpenAccessorsAreAbsoluteFromNestedSubdir(t *testing.T) {
	t.Parallel()

	repoB := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, repoB)

	nested := filepath.Join(repoB, "sub", "dir")
	if err := os.MkdirAll(nested, 0755); err != nil {
		t.Fatalf("Failed to create nested subdir: %v", err)
	}

	relNested := relFromCwd(t, nested)

	repo, err := git.Open(relNested)
	if err != nil {
		t.Fatalf("git.Open(relative nested) failed: %v", err)
	}

	if !filepath.IsAbs(repo.WorkTree()) {
		t.Errorf("WorkTree() is not absolute: %s", repo.WorkTree())
	}
	if !filepath.IsAbs(repo.GitDir()) {
		t.Errorf("GitDir() is not absolute: %s", repo.GitDir())
	}
	if !filepath.IsAbs(repo.CommonGitDir()) {
		t.Errorf("CommonGitDir() is not absolute: %s", repo.CommonGitDir())
	}
	if filepath.Base(repo.GitDir()) == ".git" && repo.GitDir() == ".git" {
		t.Errorf("GitDir() must not be the literal '.git', got %q", repo.GitDir())
	}

	wantRoot := evalSymlinks(t, repoB)
	gotRoot := evalSymlinks(t, repo.WorkTree())
	if gotRoot != wantRoot {
		t.Errorf("WorkTree() = %q, want %q", gotRoot, wantRoot)
	}

	if _, err := os.Stat(repo.GitDir()); err != nil {
		t.Errorf("GitDir() does not exist: %v", err)
	}
	gitDir := evalSymlinks(t, repo.GitDir())
	if filepath.Dir(gitDir) != wantRoot {
		t.Errorf("GitDir() %q is not inside B root %q", gitDir, wantRoot)
	}
}

// TestOpenAccessorsAbsoluteForLinkedWorktree verifies accessors for a linked
// worktree: GitDir() points at the per-worktree git dir, CommonGitDir() at the
// main .git, both absolute.
func TestOpenAccessorsAbsoluteForLinkedWorktree(t *testing.T) {
	t.Parallel()

	repoB := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, repoB)

	wtParent, err := os.MkdirTemp("", "git-flow-linked-wt-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(wtParent)
	wtPath := filepath.Join(wtParent, "wt")

	if _, err := testutil.RunGit(t, repoB, "worktree", "add", wtPath, "-b", "wt-feature"); err != nil {
		t.Fatalf("Failed to add linked worktree: %v", err)
	}

	repo, err := git.Open(wtPath)
	if err != nil {
		t.Fatalf("git.Open(linked worktree) failed: %v", err)
	}

	if !filepath.IsAbs(repo.WorkTree()) || !filepath.IsAbs(repo.GitDir()) || !filepath.IsAbs(repo.CommonGitDir()) {
		t.Errorf("Expected all accessors absolute: worktree=%q gitDir=%q commonGitDir=%q",
			repo.WorkTree(), repo.GitDir(), repo.CommonGitDir())
	}

	if _, err := os.Stat(repo.GitDir()); err != nil {
		t.Errorf("per-worktree GitDir() does not exist: %v", err)
	}

	gitDir := evalSymlinks(t, repo.GitDir())
	// Per-worktree git dir is <main .git>/worktrees/<name>.
	if filepath.Base(filepath.Dir(gitDir)) != "worktrees" {
		t.Errorf("Expected GitDir() to sit under .git/worktrees, got %q", gitDir)
	}

	wantCommon := evalSymlinks(t, filepath.Join(repoB, ".git"))
	gotCommon := evalSymlinks(t, repo.CommonGitDir())
	if gotCommon != wantCommon {
		t.Errorf("CommonGitDir() = %q, want main .git %q", gotCommon, wantCommon)
	}
}

// TestOpenEmptyStringReturnsError verifies git.Open("") errors without falling
// back to the process working directory (which is itself a valid git repo).
func TestOpenEmptyStringReturnsError(t *testing.T) {
	t.Parallel()

	repo, err := git.Open("")
	if err == nil {
		t.Fatal("Expected error for git.Open(\"\"), got nil")
	}
	if repo != nil {
		t.Errorf("Expected nil repo for git.Open(\"\"), got %+v", repo)
	}
}

// TestOpenNonRepoDirReturnsError verifies git.Open on a plain non-repo directory
// errors rather than resolving an ancestor repo or the process CWD.
func TestOpenNonRepoDirReturnsError(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	repo, err := git.Open(dir)
	if err == nil {
		t.Fatal("Expected error for git.Open(non-repo dir), got nil")
	}
	if repo != nil {
		t.Errorf("Expected nil repo for non-repo dir, got %+v", repo)
	}
}
