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
// Steps:
// 1. Creates two independent test repositories A and B
// 2. Opens a handle for B with git.Open(B)
// 3. Creates branch feature/only-in-b through B's handle
// 4. Verifies feature/only-in-b exists in B
// 5. Verifies feature/only-in-b did not leak into A
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
// Steps:
// 1. Creates a test repository and a nested sub/dir inside it
// 2. Computes a relative path from the process CWD to the nested dir
// 3. Opens a handle with git.Open(relative nested path)
// 4. Verifies WorkTree(), GitDir(), and CommonGitDir() are all absolute
// 5. Verifies WorkTree() resolves to the repository root and GitDir() lives inside it
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
// Steps:
// 1. Creates a test repository and adds a linked worktree via git worktree add
// 2. Opens a handle for the linked worktree path
// 3. Verifies WorkTree(), GitDir(), and CommonGitDir() are all absolute
// 4. Verifies GitDir() points at the per-worktree dir under .git/worktrees
// 5. Verifies CommonGitDir() points at the main repository's .git
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
// Steps:
// 1. Calls git.Open("") with an empty directory argument
// 2. Verifies an error is returned
// 3. Verifies the returned repo handle is nil
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
// Steps:
// 1. Creates a plain temporary directory that is not a git repository
// 2. Calls git.Open on that directory
// 3. Verifies an error is returned
// 4. Verifies the returned repo handle is nil
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
