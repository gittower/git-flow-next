package worktree_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gittower/git-flow-next/internal/config"
	"github.com/gittower/git-flow-next/internal/git"
	"github.com/gittower/git-flow-next/internal/worktree"
	"github.com/gittower/git-flow-next/test/testutil"
)

// setupInitializedRepo creates a repository with git-flow initialized to its
// defaults, so the loaded configuration carries the topic branch types that
// {{ topicType }} and {{ branchName }} expand against.
func setupInitializedRepo(t *testing.T) string {
	t.Helper()
	dir := testutil.SetupTestRepo(t)
	if out, err := testutil.RunGitFlow(t, dir, "init", "--defaults"); err != nil {
		testutil.CleanupTestRepo(t, dir)
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, out)
	}
	return dir
}

// openConfigured opens a repo handle for dir and loads its git-flow config.
func openConfigured(t *testing.T, dir string) (*git.Repo, *config.Config) {
	t.Helper()
	repo, err := git.Open(dir)
	if err != nil {
		t.Fatalf("git.Open(%q) failed: %v", dir, err)
	}
	cfg, err := config.Load(repo)
	if err != nil {
		t.Fatalf("config.Load failed: %v", err)
	}
	return repo, cfg
}

// setTemplate writes gitflow.worktreePath into the repository's git config.
func setTemplate(t *testing.T, dir string, template string) {
	t.Helper()
	if out, err := testutil.RunGit(t, dir, "config", "gitflow.worktreePath", template); err != nil {
		t.Fatalf("Failed to set gitflow.worktreePath: %v\nOutput: %s", err, out)
	}
}

// TestComputePathDefaultTemplate verifies the default template resolves to an
// absolute sibling of the repository.
// Steps:
// 1. Creates a repository with git-flow defaults and no gitflow.worktreePath
// 2. Opens a repo handle and loads the configuration
// 3. Calls worktree.ComputePath for feature/user-auth
// 4. Verifies the result is <repoParent>/<repo>-worktrees/feature/user-auth
func TestComputePathDefaultTemplate(t *testing.T) {
	t.Parallel()
	dir := setupInitializedRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	repo, cfg := openConfigured(t, dir)

	got, err := worktree.ComputePath(cfg, repo, "feature/user-auth")
	if err != nil {
		t.Fatalf("ComputePath failed: %v", err)
	}

	root := testutil.EvalPath(t, dir)
	want := filepath.Join(filepath.Dir(root), filepath.Base(root)+"-worktrees", "feature", "user-auth")
	if got != want {
		t.Errorf("Expected %q, got %q", want, got)
	}
}

// TestComputePathExpandsTilde verifies a leading ~ expands to the home directory.
// Steps:
// 1. Creates a repository with git-flow defaults
// 2. Overrides HOME for the test process (so no t.Parallel)
// 3. Sets gitflow.worktreePath to '~/wt/{{ branch }}'
// 4. Verifies ComputePath returns <HOME>/wt/feature/x
func TestComputePathExpandsTilde(t *testing.T) {
	dir := setupInitializedRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	home := t.TempDir()
	t.Setenv("HOME", home)
	setTemplate(t, dir, "~/wt/{{ branch }}")
	repo, cfg := openConfigured(t, dir)

	got, err := worktree.ComputePath(cfg, repo, "feature/x")
	if err != nil {
		t.Fatalf("ComputePath failed: %v", err)
	}

	want := filepath.Join(home, "wt", "feature", "x")
	if got != want {
		t.Errorf("Expected %q, got %q", want, got)
	}
}

// TestComputePathUnspacedPlaceholdersMatchSpaced verifies both brace forms expand
// identically.
// Steps:
// 1. Creates a repository with git-flow defaults
// 2. Computes the path with the spaced template '../wt/{{ topicType }}/{{ branch }}'
// 3. Computes the path with the unspaced template '../wt/{{topicType}}/{{branch}}'
// 4. Verifies the two results are byte-identical
func TestComputePathUnspacedPlaceholdersMatchSpaced(t *testing.T) {
	t.Parallel()
	dir := setupInitializedRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	setTemplate(t, dir, "../wt/{{ topicType }}/{{ branch }}")
	repo, cfg := openConfigured(t, dir)
	spaced, err := worktree.ComputePath(cfg, repo, "feature/x")
	if err != nil {
		t.Fatalf("ComputePath failed: %v", err)
	}

	setTemplate(t, dir, "../wt/{{topicType}}/{{branch}}")
	repo, cfg = openConfigured(t, dir)
	unspaced, err := worktree.ComputePath(cfg, repo, "feature/x")
	if err != nil {
		t.Fatalf("ComputePath failed: %v", err)
	}

	if spaced != unspaced {
		t.Errorf("Expected identical expansion, got %q (spaced) vs %q (unspaced)", spaced, unspaced)
	}
}

// TestComputePathAbsoluteTemplateUsedAsIs verifies an absolute template is not
// re-rooted against the main worktree.
// Steps:
// 1. Creates a repository with git-flow defaults
// 2. Sets gitflow.worktreePath to an absolute template
// 3. Calls ComputePath for feature/x
// 4. Verifies the result is the template's own root, not a path under the repository
func TestComputePathAbsoluteTemplateUsedAsIs(t *testing.T) {
	t.Parallel()
	dir := setupInitializedRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	absRoot := t.TempDir()
	setTemplate(t, dir, filepath.ToSlash(absRoot)+"/{{ branch }}")
	repo, cfg := openConfigured(t, dir)

	got, err := worktree.ComputePath(cfg, repo, "feature/x")
	if err != nil {
		t.Fatalf("ComputePath failed: %v", err)
	}

	want := filepath.Join(absRoot, "feature", "x")
	if got != want {
		t.Errorf("Expected %q, got %q", want, got)
	}
}

// TestComputePathRelativeTemplateResolvesAgainstMainWorkTree verifies a relative
// template resolves against the MAIN worktree root even when the command runs
// inside a linked worktree.
// Steps:
// 1. Creates a repository with git-flow defaults and a linked worktree on wt-branch
// 2. Sets gitflow.worktreePath to the relative template '../wt/{{ branch }}'
// 3. Opens a repo handle INSIDE the linked worktree and calls ComputePath
// 4. Verifies the result is relative to the main worktree root, not the linked one
func TestComputePathRelativeTemplateResolvesAgainstMainWorkTree(t *testing.T) {
	t.Parallel()
	dir := setupInitializedRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	linked := filepath.Join(t.TempDir(), "linked-worktree")
	if out, err := testutil.RunGit(t, dir, "worktree", "add", linked, "-b", "wt-branch"); err != nil {
		t.Fatalf("git worktree add failed: %v\nOutput: %s", err, out)
	}
	linked = testutil.EvalPath(t, linked)

	setTemplate(t, dir, "../wt/{{ branch }}")
	repo, cfg := openConfigured(t, linked)

	got, err := worktree.ComputePath(cfg, repo, "feature/x")
	if err != nil {
		t.Fatalf("ComputePath failed: %v", err)
	}

	mainRoot := testutil.EvalPath(t, dir)
	want := filepath.Join(filepath.Dir(mainRoot), "wt", "feature", "x")
	if got != want {
		t.Errorf("Expected the path to resolve against the main worktree (%q), got %q", want, got)
	}
	if got == filepath.Join(filepath.Dir(linked), "wt", "feature", "x") {
		t.Error("Expected the relative template NOT to resolve against the linked worktree")
	}
}

// TestComputePathNestedBranchProducesNestedDirs verifies a slashed branch name
// yields nested directories using host separators.
// Steps:
// 1. Creates a repository with git-flow defaults
// 2. Calls ComputePath for feature/user-auth with the default template
// 3. Verifies the result ends with the host-separated "feature/user-auth" pair
// 4. Verifies the result is absolute
func TestComputePathNestedBranchProducesNestedDirs(t *testing.T) {
	t.Parallel()
	dir := setupInitializedRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	repo, cfg := openConfigured(t, dir)

	got, err := worktree.ComputePath(cfg, repo, "feature/user-auth")
	if err != nil {
		t.Fatalf("ComputePath failed: %v", err)
	}

	if !filepath.IsAbs(got) {
		t.Errorf("Expected an absolute path, got %q", got)
	}
	nested := filepath.Join("feature", "user-auth")
	if filepath.Join(filepath.Base(filepath.Dir(got)), filepath.Base(got)) != nested {
		t.Errorf("Expected %q to end in the nested pair %q", got, nested)
	}
}

// TestComputePathCreatesNothing verifies path computation has no side effects.
// Steps:
// 1. Creates a repository with git-flow defaults
// 2. Calls ComputePath for feature/x
// 3. Verifies the returned path does not exist
// 4. Verifies its parent directory does not exist either
func TestComputePathCreatesNothing(t *testing.T) {
	t.Parallel()
	dir := setupInitializedRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer os.RemoveAll(dir + "-worktrees")
	repo, cfg := openConfigured(t, dir)

	got, err := worktree.ComputePath(cfg, repo, "feature/x")
	if err != nil {
		t.Fatalf("ComputePath failed: %v", err)
	}

	if _, err := os.Stat(got); !os.IsNotExist(err) {
		t.Errorf("Expected %q not to exist", got)
	}
	if _, err := os.Stat(filepath.Dir(got)); !os.IsNotExist(err) {
		t.Errorf("Expected %q not to exist", filepath.Dir(got))
	}
}
