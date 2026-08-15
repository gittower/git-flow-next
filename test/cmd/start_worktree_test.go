package cmd_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gittower/git-flow-next/test/testutil"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// registeredWorktreePath returns the path git records for the LINKED worktree that
// has branch checked out, or "" when no linked worktree holds it. It reads the
// porcelain form so an assertion never depends on the human column layout.
//
// The main worktree is skipped: git lists it first, and a classic start checks the
// new branch out there, which is not a worktree this feature created.
func registeredWorktreePath(t *testing.T, dir string, branch string) string {
	t.Helper()
	out, err := testutil.RunGit(t, dir, "worktree", "list", "--porcelain")
	if err != nil {
		t.Fatalf("git worktree list --porcelain failed: %v\nOutput: %s", err, out)
	}
	path := ""
	seen := 0
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "worktree ") {
			path = strings.TrimPrefix(line, "worktree ")
			seen++
			continue
		}
		if seen > 1 && line == "branch refs/heads/"+branch {
			return path
		}
	}
	return ""
}

// assertNothingCreated fails unless all three artifacts of a worktree start are
// absent for branch: the branch itself, a registered worktree, and the provenance
// marker. Every "nothing is created" assertion checks all three — a refusal that
// left any one of them behind would otherwise pass.
func assertNothingCreated(t *testing.T, dir string, branch string) {
	t.Helper()
	if testutil.BranchExists(t, dir, branch) {
		t.Errorf("Expected branch %s not to be created", branch)
	}
	if path := registeredWorktreePath(t, dir, branch); path != "" {
		t.Errorf("Expected no worktree registered for %s, found one at %s", branch, path)
	}
	if marker := testutil.GitConfigValue(t, dir, managedMarkerFor(branch)); marker != "" {
		t.Errorf("Expected no provenance marker for %s, got %q", branch, marker)
	}
}

// assertWorktreeCreatedAt fails unless branch has a worktree at want: the
// directory exists, git registers it for the branch, and the provenance marker
// records git-flow as its creator.
func assertWorktreeCreatedAt(t *testing.T, dir string, branch string, want string) {
	t.Helper()
	if info, err := os.Stat(want); err != nil || !info.IsDir() {
		t.Fatalf("Expected a worktree directory at %q: %v", want, err)
	}
	registered := registeredWorktreePath(t, dir, branch)
	if registered == "" {
		t.Fatalf("Expected git to register a worktree for %s", branch)
	}
	if resolved := testutil.EvalPath(t, want); registered != resolved {
		t.Errorf("Expected the registered worktree of %s at %q, got %q", branch, resolved, registered)
	}
	if marker := testutil.GitConfigValue(t, dir, managedMarkerFor(branch)); marker != "true" {
		t.Errorf("Expected %s=true, got %q", managedMarkerFor(branch), marker)
	}
}

// customWorktreeDest returns the relative --worktree-path value for a repository
// at dir and the absolute path it resolves to, and removes that path at the end
// of the test.
//
// The destination is derived from the repository name rather than being a fixed
// "../custom": SetupTestRepo creates every repository directly under the system
// temp directory, so a fixed relative path would be the SAME directory in every
// test that used it.
//
// The absolute form is built from the symlink-RESOLVED repository directory. A
// hand-typed relative path is made absolute with os.Getwd, which returns $PWD
// only when $PWD identifies the real working directory and otherwise falls back
// to the kernel's resolved path. These tests never override PWD, and the one they
// inherit points at the package directory, so the child always gets the resolved
// spelling. The repository root is resolved and the not-yet-existing sibling is
// appended with Join — EvalSymlinks fails on a path that does not exist.
func customWorktreeDest(t *testing.T, dir string) (relative string, absolute string) {
	t.Helper()
	name := filepath.Base(dir) + "-custom"
	absolute = filepath.Join(filepath.Dir(testutil.EvalPath(t, dir)), name)
	t.Cleanup(func() { os.RemoveAll(absolute) })
	return filepath.Join("..", name), absolute
}

// sentinelHook installs a pre-start hook that records it ran by creating a file,
// and returns that file's path. The path lives outside the repository so no
// worktree operation can disturb it.
func sentinelHook(t *testing.T, dir string, name string) string {
	t.Helper()
	sentinel := filepath.Join(t.TempDir(), "hook-fired")
	createHookScript(t, dir, name, fmt.Sprintf("#!/bin/sh\necho fired > %q\nexit 0\n", sentinel))
	return sentinel
}

// startVersionFilter installs a version filter for feature start that appends
// "-filtered" to the name it is given, so the branch the command ends up creating
// differs from the one the user typed.
func startVersionFilter(t *testing.T, dir string) {
	t.Helper()
	createHookScript(t, dir, "filter-flow-feature-start-version", "#!/bin/sh\necho \"$1-filtered\"\n")
}

// ---------------------------------------------------------------------------
// Spec scenarios
// ---------------------------------------------------------------------------

// TestStartWithoutWorktreeCreatesNoWorktree covers scenario 1: with no flag and no
// type default, start behaves exactly as it always has.
//
// The type default is UNSET first, which is required rather than cosmetic: init
// writes the key explicitly for every topic type, so without the unset this would
// duplicate scenario 12 and the absent-key path would never be exercised.
// Steps:
// 1. Sets up a repository with git-flow defaults
// 2. Unsets gitflow.branch.feature.worktree so the key is genuinely absent
// 3. Runs 'git flow feature start x'
// 4. Verifies exit code 0 and that feature/x exists
// 5. Verifies no linked worktree and no provenance marker were created
// 6. Verifies stdout carries the branch line and no worktree, cd or tip line
func TestStartWithoutWorktreeCreatesNoWorktree(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer os.RemoveAll(worktreeRootFor(dir))

	// Tolerates a missing key: before the writer exists there is nothing to unset.
	_, _ = testutil.RunGit(t, dir, "config", "--local", "--unset", "gitflow.branch.feature.worktree")

	output, err := testutil.RunGitFlow(t, dir, "feature", "start", "x")
	if err != nil {
		t.Fatalf("feature start failed: %v\nOutput: %s", err, output)
	}

	if !testutil.BranchExists(t, dir, "feature/x") {
		t.Error("Expected branch feature/x to be created")
	}
	if path := registeredWorktreePath(t, dir, "feature/x"); path != "" {
		t.Errorf("Expected no worktree for feature/x, found one at %s", path)
	}
	if marker := testutil.GitConfigValue(t, dir, managedMarkerFor("feature/x")); marker != "" {
		t.Errorf("Expected no provenance marker, got %q", marker)
	}

	list, err := testutil.RunGitFlow(t, dir, "worktree", "list")
	if err != nil {
		t.Fatalf("worktree list failed: %v\nOutput: %s", err, list)
	}
	if !strings.Contains(list, "No linked worktrees found") {
		t.Errorf("Expected no linked worktrees, got: %s", list)
	}

	if !strings.Contains(output, "Created branch 'feature/x' from 'develop'") {
		t.Errorf("Expected the branch line, got: %s", output)
	}
	for _, unwanted := range []string{"Created worktree", "To switch to it:", "Tip:"} {
		if strings.Contains(output, unwanted) {
			t.Errorf("Expected no %q line, got: %s", unwanted, output)
		}
	}
}

// TestStartWorktreeCreatesAtComputedPath covers scenario 2: --worktree creates the
// branch and a worktree at the computed path.
// Steps:
// 1. Sets up a repository with git-flow defaults
// 2. Runs 'git flow feature start x --worktree'
// 3. Verifies exit code 0, that feature/x exists and that the worktree is registered at the computed path
// 4. Verifies stdout carries the branch line, the worktree line and the cd hint in that order
// 5. Verifies the printed path is the absolute computed path
func TestStartWorktreeCreatesAtComputedPath(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer os.RemoveAll(worktreeRootFor(dir))

	output, err := testutil.RunGitFlow(t, dir, "feature", "start", "x", "--worktree")
	if err != nil {
		t.Fatalf("feature start --worktree failed: %v\nOutput: %s", err, output)
	}

	if !testutil.BranchExists(t, dir, "feature/x") {
		t.Error("Expected branch feature/x to be created")
	}
	wtPath := computedWorktreePath(t, dir, "feature/x")
	assertWorktreeCreatedAt(t, dir, "feature/x", wtPath)

	branchLine := strings.Index(output, "Created branch 'feature/x' from 'develop'")
	worktreeLine := strings.Index(output, "Created worktree for branch 'feature/x' at "+wtPath)
	cdLine := strings.Index(output, "To switch to it: cd "+wtPath)
	if branchLine < 0 || worktreeLine < 0 || cdLine < 0 {
		t.Fatalf("Expected the branch, worktree and cd lines with path %q, got: %s", wtPath, output)
	}
	if !(branchLine < worktreeLine && worktreeLine < cdLine) {
		t.Errorf("Expected the branch, worktree and cd lines in that order, got: %s", output)
	}
	if !filepath.IsAbs(wtPath) {
		t.Errorf("Expected an absolute computed path, got %q", wtPath)
	}
}

// TestStartWorktreeWritesProvenanceMarker covers scenario 3: a worktree created by
// start is recorded as git-flow's own.
// Steps:
// 1. Sets up a repository with git-flow defaults
// 2. Runs 'git flow feature start x --worktree'
// 3. Verifies gitflow.worktree.feature/x.managed is true
// 4. Verifies 'git flow worktree list' shows the worktree without the (unmanaged) tag
func TestStartWorktreeWritesProvenanceMarker(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer os.RemoveAll(worktreeRootFor(dir))

	output, err := testutil.RunGitFlow(t, dir, "feature", "start", "x", "--worktree")
	if err != nil {
		t.Fatalf("feature start --worktree failed: %v\nOutput: %s", err, output)
	}

	if marker := testutil.GitConfigValue(t, dir, managedMarkerFor("feature/x")); marker != "true" {
		t.Errorf("Expected gitflow.worktree.feature/x.managed=true, got %q", marker)
	}

	list, err := testutil.RunGitFlow(t, dir, "worktree", "list")
	if err != nil {
		t.Fatalf("worktree list failed: %v\nOutput: %s", err, list)
	}
	if !strings.Contains(list, "feature/x") {
		t.Errorf("Expected worktree list to show feature/x, got: %s", list)
	}
	if strings.Contains(list, "(unmanaged)") {
		t.Errorf("Expected the worktree to be listed as managed, got: %s", list)
	}
}

// TestStartTypeDefaultCreatesWorktree covers scenario 4: the Layer-1 type default
// makes worktree creation automatic.
// Steps:
// 1. Sets up a repository with git-flow defaults
// 2. Sets gitflow.branch.feature.worktree true
// 3. Runs 'git flow feature start x' with no flags
// 4. Verifies the worktree exists at the computed path, is registered and is marked
// 5. Verifies stdout carries the worktree line
func TestStartTypeDefaultCreatesWorktree(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer os.RemoveAll(worktreeRootFor(dir))

	if out, err := testutil.RunGit(t, dir, "config", "gitflow.branch.feature.worktree", "true"); err != nil {
		t.Fatalf("Failed to set the type default: %v\nOutput: %s", err, out)
	}

	output, err := testutil.RunGitFlow(t, dir, "feature", "start", "x")
	if err != nil {
		t.Fatalf("feature start failed: %v\nOutput: %s", err, output)
	}

	assertWorktreeCreatedAt(t, dir, "feature/x", computedWorktreePath(t, dir, "feature/x"))
	if !strings.Contains(output, "Created worktree for branch 'feature/x' at ") {
		t.Errorf("Expected the worktree line, got: %s", output)
	}
}

// TestStartNoWorktreeOverridesTypeDefault covers scenario 5: --no-worktree beats a
// Layer-1 default of true.
// Steps:
// 1. Sets up a repository with git-flow defaults and gitflow.branch.feature.worktree true
// 2. Runs 'git flow feature start x --no-worktree'
// 3. Verifies exit code 0 and that feature/x exists
// 4. Verifies no linked worktree, no marker and no worktree line
func TestStartNoWorktreeOverridesTypeDefault(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer os.RemoveAll(worktreeRootFor(dir))

	if out, err := testutil.RunGit(t, dir, "config", "gitflow.branch.feature.worktree", "true"); err != nil {
		t.Fatalf("Failed to set the type default: %v\nOutput: %s", err, out)
	}

	output, err := testutil.RunGitFlow(t, dir, "feature", "start", "x", "--no-worktree")
	if err != nil {
		t.Fatalf("feature start --no-worktree failed: %v\nOutput: %s", err, output)
	}

	if !testutil.BranchExists(t, dir, "feature/x") {
		t.Error("Expected branch feature/x to be created")
	}
	if path := registeredWorktreePath(t, dir, "feature/x"); path != "" {
		t.Errorf("Expected no worktree for feature/x, found one at %s", path)
	}
	if marker := testutil.GitConfigValue(t, dir, managedMarkerFor("feature/x")); marker != "" {
		t.Errorf("Expected no provenance marker, got %q", marker)
	}

	list, err := testutil.RunGitFlow(t, dir, "worktree", "list")
	if err != nil {
		t.Fatalf("worktree list failed: %v\nOutput: %s", err, list)
	}
	if !strings.Contains(list, "No linked worktrees found") {
		t.Errorf("Expected no linked worktrees, got: %s", list)
	}
	if strings.Contains(output, "Created worktree") {
		t.Errorf("Expected no worktree line, got: %s", output)
	}
}

// TestStartWorktreePathImpliesCreationAndMarks covers scenario 6: --worktree-path
// implies creation and puts the worktree where it says.
// Steps:
// 1. Sets up a repository with git-flow defaults
// 2. Runs 'git flow feature start x --worktree-path ../<repo>-custom' without --worktree
// 3. Verifies exit code 0 and that the worktree is registered for feature/x at that path
// 4. Verifies the printed path is the hand-typed destination, made absolute against the resolved invocation directory
// 5. Verifies nothing was created under the computed worktree root
func TestStartWorktreePathImpliesCreationAndMarks(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	relative, absolute := customWorktreeDest(t, dir)

	output, err := testutil.RunGitFlow(t, dir, "feature", "start", "x", "--worktree-path", relative)
	if err != nil {
		t.Fatalf("feature start --worktree-path failed: %v\nOutput: %s", err, output)
	}

	assertWorktreeCreatedAt(t, dir, "feature/x", absolute)
	if !strings.Contains(output, "Created worktree for branch 'feature/x' at "+absolute) {
		t.Errorf("Expected the worktree line to name %q, got: %s", absolute, output)
	}
	if _, err := os.Stat(worktreeRootFor(dir)); !os.IsNotExist(err) {
		t.Errorf("Expected nothing under the computed worktree root %q", worktreeRootFor(dir))
	}
}

// TestStartWorktreePathBeatsNoWorktree covers scenario 6 Test B: --worktree-path
// implies creation, so it is the POSITIVE side of the --worktree/--no-worktree
// pair and beats --no-worktree (decisions.md SC-11).
// Steps:
// 1. Sets up a repository with git-flow defaults
// 2. Runs 'git flow feature start x --worktree-path ../<repo>-custom --no-worktree'
// 3. Verifies exit code 0 and that feature/x exists
// 4. Verifies the worktree was still created at the hand-typed path and is marked
func TestStartWorktreePathBeatsNoWorktree(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	relative, absolute := customWorktreeDest(t, dir)

	output, err := testutil.RunGitFlow(t, dir, "feature", "start", "x", "--worktree-path", relative, "--no-worktree")
	if err != nil {
		t.Fatalf("feature start --worktree-path --no-worktree failed: %v\nOutput: %s", err, output)
	}

	if !testutil.BranchExists(t, dir, "feature/x") {
		t.Error("Expected branch feature/x to be created")
	}
	assertWorktreeCreatedAt(t, dir, "feature/x", absolute)
}

// TestStartWorktreeQuietSuppressesShellInitTip covers scenario 7: --quiet drops the
// shell-init tip and nothing else.
// Steps:
// 1. Sets up a repository with git-flow defaults and GIT_FLOW_CD_FILE unset, the only state where the tip would print
// 2. Runs 'git flow feature start x --worktree --quiet'
// 3. Verifies exit code 0 and that the worktree exists
// 4. Verifies stdout still carries the worktree line and the cd hint
// 5. Verifies stdout carries no Tip: line
func TestStartWorktreeQuietSuppressesShellInitTip(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer os.RemoveAll(worktreeRootFor(dir))

	stdout, stderr, err := testutil.RunGitFlowStreamsWithEnv(t, dir, nil, "feature", "start", "x", "--worktree", "--quiet")
	if err != nil {
		t.Fatalf("feature start --worktree --quiet failed: %v\nStderr: %s", err, stderr)
	}

	wtPath := computedWorktreePath(t, dir, "feature/x")
	assertWorktreeCreatedAt(t, dir, "feature/x", wtPath)
	if !strings.Contains(stdout, "Created worktree for branch 'feature/x' at "+wtPath) {
		t.Errorf("Expected the worktree line, got: %s", stdout)
	}
	if !strings.Contains(stdout, "To switch to it: cd "+wtPath) {
		t.Errorf("Expected the cd hint, got: %s", stdout)
	}
	if strings.Contains(stdout, "Tip:") {
		t.Errorf("Expected no shell-init tip, got: %s", stdout)
	}
}

// TestStartWorktreePrintsShellInitTipWithoutQuiet covers scenario 7 Test B, the
// non-vacuity control for --quiet: without it the tip does print, so a build that
// never printed the tip at all cannot pass the suppression test.
// Steps:
// 1. Sets up a repository with git-flow defaults and GIT_FLOW_CD_FILE unset
// 2. Runs 'git flow feature start x --worktree'
// 3. Verifies stdout carries the shell-init tip naming the <shell> placeholder
func TestStartWorktreePrintsShellInitTipWithoutQuiet(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer os.RemoveAll(worktreeRootFor(dir))

	stdout, stderr, err := testutil.RunGitFlowStreamsWithEnv(t, dir, nil, "feature", "start", "x", "--worktree")
	if err != nil {
		t.Fatalf("feature start --worktree failed: %v\nStderr: %s", err, stderr)
	}

	if !strings.Contains(stdout, "Tip: run 'git flow shell-init <shell>' for automatic directory switching") {
		t.Errorf("Expected the shell-init tip, got: %s", stdout)
	}
}

// TestStartWorktreeOccupiedPathCreatesNothing covers scenario 8: an occupied target
// path is refused before anything is created.
// Steps:
// 1. Sets up a repository with git-flow defaults and records the checked-out branch
// 2. Creates a non-empty directory at the computed path for feature/x
// 3. Runs 'git flow feature start x --worktree'
// 4. Verifies exit code 6 and that stderr names the occupied path
// 5. Verifies no branch, no registered worktree and no provenance marker
// 6. Verifies the obstruction is untouched and HEAD is still the pre-command branch
func TestStartWorktreeOccupiedPathCreatesNothing(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer os.RemoveAll(worktreeRootFor(dir))

	branchBefore := testutil.GetCurrentBranch(t, dir)
	wtPath := computedWorktreePath(t, dir, "feature/x")
	writeObstruction(t, wtPath, "keep.txt", "precious")

	stdout, stderr, err := testutil.RunGitFlowStreamsWithEnv(t, dir, nil, "feature", "start", "x", "--worktree")
	if got := worktreeExitCode(err); got != 6 {
		t.Fatalf("Expected exit code 6, got %d\nStdout: %s\nStderr: %s", got, stdout, stderr)
	}
	if !strings.Contains(stderr, wtPath) {
		t.Errorf("Expected stderr to name the occupied path %q, got: %s", wtPath, stderr)
	}

	assertNothingCreated(t, dir, "feature/x")
	assertFileContent(t, filepath.Join(wtPath, "keep.txt"), "precious")
	if branchAfter := testutil.GetCurrentBranch(t, dir); branchAfter != branchBefore {
		t.Errorf("Expected HEAD to stay on %q, got %q", branchBefore, branchAfter)
	}
}

// TestStartWorktreeBeatsNoWorktree covers scenario 9 with the assertion
// deliberately INVERTED per decisions.md SC-5/SC-6: the positive flag wins, as it
// does for all ~38 other --x/--no-x pairs in this CLI, so the combination creates
// the branch AND the worktree instead of erroring.
// Steps:
// 1. Sets up a repository with git-flow defaults
// 2. Runs 'git flow feature start x --worktree --no-worktree'
// 3. Verifies exit code 0 and that feature/x exists
// 4. Verifies the worktree exists at the computed path, is registered and is marked
// 5. Verifies stdout carries the worktree line
func TestStartWorktreeBeatsNoWorktree(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer os.RemoveAll(worktreeRootFor(dir))

	output, err := testutil.RunGitFlow(t, dir, "feature", "start", "x", "--worktree", "--no-worktree")
	if err != nil {
		t.Fatalf("feature start --worktree --no-worktree failed: %v\nOutput: %s", err, output)
	}

	if !testutil.BranchExists(t, dir, "feature/x") {
		t.Error("Expected branch feature/x to be created")
	}
	assertWorktreeCreatedAt(t, dir, "feature/x", computedWorktreePath(t, dir, "feature/x"))
	if !strings.Contains(output, "Created worktree for branch 'feature/x' at ") {
		t.Errorf("Expected the worktree line, got: %s", output)
	}
}

// TestStartNoWorktreeBeforeWorktreeStillCreatesWorktree covers scenario 9 Test B:
// the outcome is order-independent, because pflag records only whether each flag
// was set, never their relative order (decisions.md SC-5/SC-6).
// Steps:
// 1. Sets up a repository with git-flow defaults
// 2. Runs 'git flow feature start x --no-worktree --worktree' with the flags reversed
// 3. Verifies exit code 0 and that feature/x exists
// 4. Verifies the worktree exists at the computed path, is registered and is marked
func TestStartNoWorktreeBeforeWorktreeStillCreatesWorktree(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer os.RemoveAll(worktreeRootFor(dir))

	output, err := testutil.RunGitFlow(t, dir, "feature", "start", "x", "--no-worktree", "--worktree")
	if err != nil {
		t.Fatalf("feature start --no-worktree --worktree failed: %v\nOutput: %s", err, output)
	}

	if !testutil.BranchExists(t, dir, "feature/x") {
		t.Error("Expected branch feature/x to be created")
	}
	assertWorktreeCreatedAt(t, dir, "feature/x", computedWorktreePath(t, dir, "feature/x"))
}

// TestStartWorktreeWithCustomStartPoint covers scenario 10: an explicit start point
// is honored when a worktree is created.
// Steps:
// 1. Sets up a repository with git-flow defaults and an extra commit on main
// 2. Returns to develop, so main and develop point at different commits
// 3. Runs 'git flow feature start x main --worktree'
// 4. Verifies feature/x points at main's tip
// 5. Verifies the new worktree's HEAD is that same commit and is on feature/x
// 6. Verifies gitflow.branch.feature/x.base is main
func TestStartWorktreeWithCustomStartPoint(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer os.RemoveAll(worktreeRootFor(dir))

	if out, err := testutil.RunGit(t, dir, "checkout", "main"); err != nil {
		t.Fatalf("Failed to check out main: %v\nOutput: %s", err, out)
	}
	if err := testutil.WriteFile(t, dir, "main-only.txt", "main only"); err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}
	if out, err := testutil.RunGit(t, dir, "add", "main-only.txt"); err != nil {
		t.Fatalf("Failed to stage file: %v\nOutput: %s", err, out)
	}
	if out, err := testutil.RunGit(t, dir, "commit", "-m", "Main only commit"); err != nil {
		t.Fatalf("Failed to commit: %v\nOutput: %s", err, out)
	}
	if out, err := testutil.RunGit(t, dir, "checkout", "develop"); err != nil {
		t.Fatalf("Failed to check out develop: %v\nOutput: %s", err, out)
	}

	mainTip, err := testutil.RunGit(t, dir, "rev-parse", "main")
	if err != nil {
		t.Fatalf("Failed to resolve main: %v", err)
	}
	mainTip = strings.TrimSpace(mainTip)

	output, err := testutil.RunGitFlow(t, dir, "feature", "start", "x", "main", "--worktree")
	if err != nil {
		t.Fatalf("feature start with a start point failed: %v\nOutput: %s", err, output)
	}

	branchTip, err := testutil.RunGit(t, dir, "rev-parse", "feature/x")
	if err != nil {
		t.Fatalf("Failed to resolve feature/x: %v", err)
	}
	if strings.TrimSpace(branchTip) != mainTip {
		t.Errorf("Expected feature/x at main's tip %s, got %s", mainTip, strings.TrimSpace(branchTip))
	}

	wtPath := computedWorktreePath(t, dir, "feature/x")
	assertWorktreeCreatedAt(t, dir, "feature/x", wtPath)

	wtHead, err := testutil.RunGit(t, wtPath, "rev-parse", "HEAD")
	if err != nil {
		t.Fatalf("Failed to resolve the worktree HEAD: %v", err)
	}
	if strings.TrimSpace(wtHead) != mainTip {
		t.Errorf("Expected the worktree HEAD at %s, got %s", mainTip, strings.TrimSpace(wtHead))
	}
	if branch := testutil.GetCurrentBranch(t, wtPath); branch != "feature/x" {
		t.Errorf("Expected the worktree to be on feature/x, got %q", branch)
	}

	if base := testutil.GitConfigValue(t, dir, "gitflow.branch.feature/x.base"); base != "main" {
		t.Errorf("Expected gitflow.branch.feature/x.base=main, got %q", base)
	}
}

// TestStartWorktreeWithFetchStillFetches covers scenario 11 Test A: --fetch is
// unaffected by worktree creation. It is also the control for E8 Test C, which
// asserts the fetch is skipped when the path is occupied.
// Steps:
// 1. Sets up a repository with git-flow defaults and a configured remote
// 2. Runs 'git flow feature start x --worktree --fetch'
// 3. Verifies exit code 0 and that stdout carries the fetch line
// 4. Verifies the worktree was created and registered
func TestStartWorktreeWithFetchStillFetches(t *testing.T) {
	t.Parallel()
	dir, remoteDir := testutil.SetupTestRepoWithRemote(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer testutil.CleanupTestRepo(t, remoteDir)
	defer os.RemoveAll(worktreeRootFor(dir))

	output, err := testutil.RunGitFlow(t, dir, "feature", "start", "x", "--worktree", "--fetch")
	if err != nil {
		t.Fatalf("feature start --worktree --fetch failed: %v\nOutput: %s", err, output)
	}

	if !strings.Contains(output, "Fetching from") {
		t.Errorf("Expected a fetch line, got: %s", output)
	}
	assertWorktreeCreatedAt(t, dir, "feature/x", computedWorktreePath(t, dir, "feature/x"))
}

// TestStartWorktreeWithNoFetchDoesNotFetch covers scenario 11 Test B: --no-fetch
// still suppresses the fetch when a worktree is created.
// Steps:
// 1. Sets up a repository with git-flow defaults and a configured remote
// 2. Runs 'git flow feature start x --worktree --no-fetch'
// 3. Verifies exit code 0 and that stdout carries no fetch line
// 4. Verifies the worktree was still created
func TestStartWorktreeWithNoFetchDoesNotFetch(t *testing.T) {
	t.Parallel()
	dir, remoteDir := testutil.SetupTestRepoWithRemote(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer testutil.CleanupTestRepo(t, remoteDir)
	defer os.RemoveAll(worktreeRootFor(dir))

	output, err := testutil.RunGitFlow(t, dir, "feature", "start", "x", "--worktree", "--no-fetch")
	if err != nil {
		t.Fatalf("feature start --worktree --no-fetch failed: %v\nOutput: %s", err, output)
	}

	if strings.Contains(output, "Fetching from") {
		t.Errorf("Expected no fetch line, got: %s", output)
	}
	assertWorktreeCreatedAt(t, dir, "feature/x", computedWorktreePath(t, dir, "feature/x"))
}

// TestStartWorktreeFetchFlagOverridesConfig covers scenario 11 Test C: worktree
// creation does not disturb the existing fetch resolution, so the flag still beats
// a config false.
// Steps:
// 1. Sets up a repository with git-flow defaults and a configured remote
// 2. Sets gitflow.feature.start.fetch false
// 3. Runs 'git flow feature start x --worktree --fetch'
// 4. Verifies stdout carries the fetch line, so the flag won
// 5. Verifies the worktree was created
func TestStartWorktreeFetchFlagOverridesConfig(t *testing.T) {
	t.Parallel()
	dir, remoteDir := testutil.SetupTestRepoWithRemote(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer testutil.CleanupTestRepo(t, remoteDir)
	defer os.RemoveAll(worktreeRootFor(dir))

	if out, err := testutil.RunGit(t, dir, "config", "gitflow.feature.start.fetch", "false"); err != nil {
		t.Fatalf("Failed to set the fetch config: %v\nOutput: %s", err, out)
	}

	output, err := testutil.RunGitFlow(t, dir, "feature", "start", "x", "--worktree", "--fetch")
	if err != nil {
		t.Fatalf("feature start --worktree --fetch failed: %v\nOutput: %s", err, output)
	}

	if !strings.Contains(output, "Fetching from") {
		t.Errorf("Expected a fetch line, got: %s", output)
	}
	assertWorktreeCreatedAt(t, dir, "feature/x", computedWorktreePath(t, dir, "feature/x"))
}

// TestStartExplicitFalseTypeDefaultCreatesNoWorktree covers scenario 12: an
// explicit false type default behaves exactly like an absent one.
// Steps:
// 1. Sets up a repository with git-flow defaults
// 2. Sets gitflow.branch.feature.worktree false
// 3. Runs 'git flow feature start x'
// 4. Verifies feature/x exists with no worktree and no provenance marker
func TestStartExplicitFalseTypeDefaultCreatesNoWorktree(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer os.RemoveAll(worktreeRootFor(dir))

	if out, err := testutil.RunGit(t, dir, "config", "gitflow.branch.feature.worktree", "false"); err != nil {
		t.Fatalf("Failed to set the type default: %v\nOutput: %s", err, out)
	}

	output, err := testutil.RunGitFlow(t, dir, "feature", "start", "x")
	if err != nil {
		t.Fatalf("feature start failed: %v\nOutput: %s", err, output)
	}

	if !testutil.BranchExists(t, dir, "feature/x") {
		t.Error("Expected branch feature/x to be created")
	}
	if path := registeredWorktreePath(t, dir, "feature/x"); path != "" {
		t.Errorf("Expected no worktree for feature/x, found one at %s", path)
	}
	if marker := testutil.GitConfigValue(t, dir, managedMarkerFor("feature/x")); marker != "" {
		t.Errorf("Expected no provenance marker, got %q", marker)
	}
}

// TestStartWorktreeWritesNavigationDestination covers scenario 13: with
// GIT_FLOW_CD_FILE set, the new worktree's absolute path is handed to the shell.
// Steps:
// 1. Sets up a repository with git-flow defaults and an empty GIT_FLOW_CD_FILE
// 2. Runs 'git flow feature start x --worktree' with the variable set
// 3. Verifies exit code 0 and that the worktree exists
// 4. Verifies the file holds the absolute computed worktree path
// 5. Verifies no shell-init tip prints, since the channel is plainly in use
func TestStartWorktreeWritesNavigationDestination(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer os.RemoveAll(worktreeRootFor(dir))
	cdFile := cdFilePath(t)

	output, err := testutil.RunGitFlowWithEnv(t, dir, cdEnv(cdFile), "feature", "start", "x", "--worktree")
	if err != nil {
		t.Fatalf("feature start --worktree failed: %v\nOutput: %s", err, output)
	}

	wtPath := computedWorktreePath(t, dir, "feature/x")
	assertWorktreeCreatedAt(t, dir, "feature/x", wtPath)
	if got := readCDFile(t, cdFile); got != wtPath {
		t.Errorf("Expected CD file to hold %q, got %q", wtPath, got)
	}
	if strings.Contains(output, "Tip:") {
		t.Errorf("Expected no shell-init tip while the channel is in use, got: %s", output)
	}
}

// TestStartWorktreeWithoutCDFilePrintsHintAndTip covers scenario 14: with the
// variable unset, start prints exactly the four contract lines and writes nothing.
//
// This is the exact-stdout pin for #173. A test repository with no remote prints
// no fetch line, so the four lines are the whole of stdout.
// Steps:
// 1. Sets up a repository with git-flow defaults and creates a CD file that is NOT exported
// 2. Runs 'git flow feature start x --worktree' with GIT_FLOW_CD_FILE unset
// 3. Verifies the worktree was created
// 4. Verifies stdout is exactly the branch, worktree, cd and tip lines in that order
// 5. Verifies stderr is empty and the unexported CD file was never written
func TestStartWorktreeWithoutCDFilePrintsHintAndTip(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer os.RemoveAll(worktreeRootFor(dir))
	cdFile := cdFilePath(t)

	stdout, stderr, err := testutil.RunGitFlowStreamsWithEnv(t, dir, nil, "feature", "start", "x", "--worktree")
	if err != nil {
		t.Fatalf("feature start --worktree failed: %v\nStderr: %s", err, stderr)
	}

	wtPath := computedWorktreePath(t, dir, "feature/x")
	assertWorktreeCreatedAt(t, dir, "feature/x", wtPath)

	want := "Created branch 'feature/x' from 'develop'\n" +
		"Created worktree for branch 'feature/x' at " + wtPath + "\n" +
		"To switch to it: cd " + wtPath + "\n" +
		"Tip: run 'git flow shell-init <shell>' for automatic directory switching\n"
	if stdout != want {
		t.Errorf("Expected stdout %q, got %q", want, stdout)
	}
	if stderr != "" {
		t.Errorf("Expected no output on stderr, got %q", stderr)
	}
	assertCDFileEmpty(t, cdFile)
}

// TestStartWorktreeNoCDLeavesFileEmpty covers scenario 15: --no-cd suppresses the
// channel write and nothing else.
// Steps:
// 1. Sets up a repository with git-flow defaults and an empty GIT_FLOW_CD_FILE
// 2. Runs 'git flow feature start x --worktree --no-cd' with the variable set
// 3. Verifies the branch, worktree and provenance marker were all created
// 4. Verifies the CD file is still empty
// 5. Verifies the cd hint still prints, since --no-cd suppresses only the channel write
func TestStartWorktreeNoCDLeavesFileEmpty(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer os.RemoveAll(worktreeRootFor(dir))
	cdFile := cdFilePath(t)

	output, err := testutil.RunGitFlowWithEnv(t, dir, cdEnv(cdFile), "feature", "start", "x", "--worktree", "--no-cd")
	if err != nil {
		t.Fatalf("feature start --worktree --no-cd failed: %v\nOutput: %s", err, output)
	}

	if !testutil.BranchExists(t, dir, "feature/x") {
		t.Error("Expected branch feature/x to be created")
	}
	wtPath := computedWorktreePath(t, dir, "feature/x")
	assertWorktreeCreatedAt(t, dir, "feature/x", wtPath)
	assertCDFileEmpty(t, cdFile)
	if !strings.Contains(output, "To switch to it: cd "+wtPath) {
		t.Errorf("Expected the cd hint to still print, got: %s", output)
	}
}

// TestStartWithoutWorktreeLeavesCDFileEmpty covers scenario 16: a start that
// creates no worktree writes nothing to the channel and prints what it always did.
// Steps:
// 1. Sets up a repository with git-flow defaults and an empty GIT_FLOW_CD_FILE
// 2. Runs 'git flow feature start x' with the variable set
// 3. Verifies the CD file is still empty
// 4. Verifies stdout is exactly the branch line, with no worktree or navigation lines
func TestStartWithoutWorktreeLeavesCDFileEmpty(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer os.RemoveAll(worktreeRootFor(dir))
	cdFile := cdFilePath(t)

	stdout, stderr, err := testutil.RunGitFlowStreamsWithEnv(t, dir, cdEnv(cdFile), "feature", "start", "x")
	if err != nil {
		t.Fatalf("feature start failed: %v\nStderr: %s", err, stderr)
	}

	assertCDFileEmpty(t, cdFile)
	want := "Created branch 'feature/x' from 'develop'\n"
	if stdout != want {
		t.Errorf("Expected stdout %q, got %q", want, stdout)
	}
}

// TestStartWorktreePathWritesAbsoluteDestination covers scenario 17: a relative
// --worktree-path reaches the channel in absolute form.
//
// The expectation is built from the symlink-resolved repository directory: this
// test does not override PWD, so os.Getwd falls back to the kernel's resolved
// working directory. See customWorktreeDest.
// Steps:
// 1. Sets up a repository with git-flow defaults and an empty GIT_FLOW_CD_FILE
// 2. Runs 'git flow feature start x --worktree-path ../<repo>-custom' with the variable set
// 3. Verifies the CD file holds the absolute destination, never the literal relative form
func TestStartWorktreePathWritesAbsoluteDestination(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	relative, absolute := customWorktreeDest(t, dir)
	cdFile := cdFilePath(t)

	output, err := testutil.RunGitFlowWithEnv(t, dir, cdEnv(cdFile), "feature", "start", "x", "--worktree-path", relative)
	if err != nil {
		t.Fatalf("feature start --worktree-path failed: %v\nOutput: %s", err, output)
	}

	if got := readCDFile(t, cdFile); got != absolute {
		t.Errorf("Expected CD file to hold %q, got %q", absolute, got)
	}
}

// TestStartWorktreeOccupiedPathLeavesCDFileEmpty covers scenario 18: a refusal
// writes nothing to the navigation channel.
// Steps:
// 1. Sets up a repository with git-flow defaults and an empty GIT_FLOW_CD_FILE
// 2. Creates a non-empty directory at the computed path for feature/x
// 3. Runs 'git flow feature start x --worktree' with the variable set
// 4. Verifies exit code 6
// 5. Verifies no branch, no registered worktree and no provenance marker
// 6. Verifies the CD file is still empty
func TestStartWorktreeOccupiedPathLeavesCDFileEmpty(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer os.RemoveAll(worktreeRootFor(dir))
	cdFile := cdFilePath(t)
	writeObstruction(t, computedWorktreePath(t, dir, "feature/x"), "keep.txt", "precious")

	output, err := testutil.RunGitFlowWithEnv(t, dir, cdEnv(cdFile), "feature", "start", "x", "--worktree")
	if got := worktreeExitCode(err); got != 6 {
		t.Fatalf("Expected exit code 6, got %d\nOutput: %s", got, output)
	}

	assertNothingCreated(t, dir, "feature/x")
	assertCDFileEmpty(t, cdFile)
}

// ---------------------------------------------------------------------------
// Additional scenarios from decisions.md
// ---------------------------------------------------------------------------

// TestStartWorktreeLeavesHeadUnchanged covers E1 (decisions.md SC-2): creating a
// worktree must not check the branch out in the invocation worktree, because git
// allows a branch in only one worktree at a time.
//
// This is the regression guard for the 'git checkout -b' to 'git branch' change.
// Without it a reversion would fail only with git's own "already used by worktree"
// message, which is easy to misdiagnose.
// Steps:
// 1. Sets up a repository with git-flow defaults, checked out on develop
// 2. Records the current branch and commit
// 3. Runs 'git flow feature start x --worktree'
// 4. Verifies the invocation repository is still on develop at the same commit
// 5. Verifies the new worktree directory is the one on feature/x
func TestStartWorktreeLeavesHeadUnchanged(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer os.RemoveAll(worktreeRootFor(dir))

	branchBefore := testutil.GetCurrentBranch(t, dir)
	if branchBefore != "develop" {
		t.Fatalf("Expected the repository to start on develop, got %q", branchBefore)
	}
	headBefore, err := testutil.RunGit(t, dir, "rev-parse", "HEAD")
	if err != nil {
		t.Fatalf("Failed to resolve HEAD: %v", err)
	}

	output, err := testutil.RunGitFlow(t, dir, "feature", "start", "x", "--worktree")
	if err != nil {
		t.Fatalf("feature start --worktree failed: %v\nOutput: %s", err, output)
	}

	if branchAfter := testutil.GetCurrentBranch(t, dir); branchAfter != "develop" {
		t.Errorf("Expected the invocation worktree to stay on develop, got %q", branchAfter)
	}
	headAfter, err := testutil.RunGit(t, dir, "rev-parse", "HEAD")
	if err != nil {
		t.Fatalf("Failed to resolve HEAD: %v", err)
	}
	if strings.TrimSpace(headAfter) != strings.TrimSpace(headBefore) {
		t.Errorf("Expected HEAD to stay at %s, got %s", strings.TrimSpace(headBefore), strings.TrimSpace(headAfter))
	}

	wtPath := computedWorktreePath(t, dir, "feature/x")
	if branch := testutil.GetCurrentBranch(t, wtPath); branch != "feature/x" {
		t.Errorf("Expected the new worktree to be on feature/x, got %q", branch)
	}
}

// TestStartWithoutWorktreeChecksOutNewBranch covers E4: the classic path still
// leaves the user on the new branch, so the no-checkout creation never becomes the
// default.
// Steps:
// 1. Sets up a repository with git-flow defaults, checked out on develop
// 2. Runs 'git flow feature start x' with no worktree flags
// 3. Verifies the current branch is feature/x
func TestStartWithoutWorktreeChecksOutNewBranch(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	output, err := testutil.RunGitFlow(t, dir, "feature", "start", "x")
	if err != nil {
		t.Fatalf("feature start failed: %v\nOutput: %s", err, output)
	}

	if branch := testutil.GetCurrentBranch(t, dir); branch != "feature/x" {
		t.Errorf("Expected the repository to be on feature/x, got %q", branch)
	}
}

// TestStartWorktreeFromLinkedWorktree covers E5: run from a linked worktree, the
// new worktree still lands at the path computed from the MAIN worktree root.
// Steps:
// 1. Sets up a repository with git-flow defaults and adds a linked worktree for a free branch
// 2. Runs 'git flow feature start x --worktree' with the linked worktree as the working directory
// 3. Verifies exit code 0 and that the worktree is at the main-worktree-relative computed path
// 4. Verifies the linked worktree's HEAD is unchanged
// 5. Verifies the marker reached the shared config, so both directories list the new worktree as managed
func TestStartWorktreeFromLinkedWorktree(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer os.RemoveAll(worktreeRootFor(dir))
	createFreeBranch(t, dir, "wt-test")

	wtPath, err := os.MkdirTemp("", "git-flow-test-wt-*")
	if err != nil {
		t.Fatalf("Failed to create the linked worktree directory: %v", err)
	}
	// git worktree add refuses a destination that already exists as a populated
	// directory, so the temp directory is removed and recreated by git.
	os.RemoveAll(wtPath)
	if out, err := testutil.RunGit(t, dir, "worktree", "add", wtPath, "wt-test"); err != nil {
		t.Fatalf("git worktree add failed: %v\nOutput: %s", err, out)
	}
	t.Cleanup(func() {
		_, _ = testutil.RunGit(t, dir, "worktree", "remove", "--force", wtPath)
		os.RemoveAll(wtPath)
	})

	output, err := testutil.RunGitFlow(t, wtPath, "feature", "start", "x", "--worktree")
	if err != nil {
		t.Fatalf("feature start --worktree from a linked worktree failed: %v\nOutput: %s", err, output)
	}

	assertWorktreeCreatedAt(t, dir, "feature/x", computedWorktreePath(t, dir, "feature/x"))
	if branch := testutil.GetCurrentBranch(t, wtPath); branch != "wt-test" {
		t.Errorf("Expected the linked worktree to stay on wt-test, got %q", branch)
	}

	for _, from := range []string{dir, wtPath} {
		list, err := testutil.RunGitFlow(t, from, "worktree", "list")
		if err != nil {
			t.Fatalf("worktree list from %s failed: %v\nOutput: %s", from, err, list)
		}
		for _, row := range worktreeRows(list) {
			if strings.HasPrefix(row, "feature/x") && strings.Contains(row, "(unmanaged)") {
				t.Errorf("Expected feature/x to be listed as managed from %s, got: %s", from, row)
			}
		}
	}
}

// TestStartWorktreeAcceptsEmptyTargetDirectory covers E6 (#172 decision 12): an
// existing EMPTY directory is a valid target, matching plain 'git worktree add'.
// Steps:
// 1. Sets up a repository with git-flow defaults
// 2. Creates the computed path for feature/x as an empty directory
// 3. Runs 'git flow feature start x --worktree'
// 4. Verifies exit code 0, that the worktree was created in that directory and that the marker was written
func TestStartWorktreeAcceptsEmptyTargetDirectory(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer os.RemoveAll(worktreeRootFor(dir))

	wtPath := computedWorktreePath(t, dir, "feature/x")
	if err := os.MkdirAll(wtPath, 0755); err != nil {
		t.Fatalf("Failed to create the empty target directory: %v", err)
	}

	output, err := testutil.RunGitFlow(t, dir, "feature", "start", "x", "--worktree")
	if err != nil {
		t.Fatalf("feature start --worktree into an empty directory failed: %v\nOutput: %s", err, output)
	}

	assertWorktreeCreatedAt(t, dir, "feature/x", wtPath)
}

// TestStartWorktreeInCommitlessRepositoryRefused covers E7 (decisions.md SC-9): a
// repository with no commits cannot back a second worktree, so --worktree is
// refused before anything happens.
// Steps:
// 1. Sets up a commit-less repository and runs 'git flow init --defaults --no-create-branches'
// 2. Asserts the precondition: HEAD is unborn and develop does not exist
// 3. Runs 'git flow feature start x --worktree'
// 4. Verifies exit code 2 and an explanation on stderr
// 5. Verifies no branch, no worktree, no marker and still no commit
func TestStartWorktreeInCommitlessRepositoryRefused(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupEmptyTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	if out, err := testutil.RunGitFlow(t, dir, "init", "--defaults", "--no-create-branches"); err != nil {
		t.Fatalf("init --defaults --no-create-branches failed: %v\nOutput: %s", err, out)
	}
	if _, err := testutil.RunGit(t, dir, "rev-parse", "--verify", "HEAD"); err == nil {
		t.Fatal("Expected the repository to have no commits")
	}
	if testutil.BranchExists(t, dir, "develop") {
		t.Fatal("Expected no develop branch in the commit-less repository")
	}

	stdout, stderr, err := testutil.RunGitFlowStreamsWithEnv(t, dir, nil, "feature", "start", "x", "--worktree")
	if got := worktreeExitCode(err); got != 2 {
		t.Fatalf("Expected exit code 2, got %d\nStdout: %s\nStderr: %s", got, stdout, stderr)
	}
	if !strings.Contains(stderr, "commit") {
		t.Errorf("Expected stderr to explain the missing commit, got: %s", stderr)
	}

	assertNothingCreated(t, dir, "feature/x")
	if _, err := testutil.RunGit(t, dir, "rev-parse", "--verify", "HEAD"); err == nil {
		t.Error("Expected no commit to be created")
	}
}

// TestStartWithoutWorktreeInCommitlessRepositoryFailsOnStartPoint covers E7 Test B,
// the non-vacuity control: without --worktree the same repository fails later and
// differently, so the exit-2 refusal is a genuinely new, earlier failure.
// Steps:
// 1. Sets up a commit-less repository and runs 'git flow init --defaults --no-create-branches'
// 2. Runs 'git flow feature start x' with no worktree flag
// 3. Verifies exit code 5 and the missing start point message
func TestStartWithoutWorktreeInCommitlessRepositoryFailsOnStartPoint(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupEmptyTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	if out, err := testutil.RunGitFlow(t, dir, "init", "--defaults", "--no-create-branches"); err != nil {
		t.Fatalf("init --defaults --no-create-branches failed: %v\nOutput: %s", err, out)
	}

	stdout, stderr, err := testutil.RunGitFlowStreamsWithEnv(t, dir, nil, "feature", "start", "x")
	if got := worktreeExitCode(err); got != 5 {
		t.Fatalf("Expected exit code 5, got %d\nStdout: %s\nStderr: %s", got, stdout, stderr)
	}
	if !strings.Contains(stderr, "start point branch 'develop' does not exist") {
		t.Errorf("Expected the missing start point message, got: %s", stderr)
	}
}

// TestStartWorktreeOccupiedPathSkipsPreStartHook covers E8 (decisions.md SC-7): a
// command that cannot succeed runs no pre-start hook, so "nothing is created" also
// means nothing ran.
// Steps:
// 1. Sets up a repository with git-flow defaults and a pre-flow-feature-start hook that writes a sentinel
// 2. Creates a non-empty directory at the computed path for feature/x
// 3. Runs 'git flow feature start x --worktree'
// 4. Verifies exit code 6 and that the sentinel file was never written
// 5. Verifies no branch, no registered worktree and no provenance marker
func TestStartWorktreeOccupiedPathSkipsPreStartHook(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer os.RemoveAll(worktreeRootFor(dir))

	sentinel := sentinelHook(t, dir, "pre-flow-feature-start")
	writeObstruction(t, computedWorktreePath(t, dir, "feature/x"), "keep.txt", "precious")

	output, err := testutil.RunGitFlow(t, dir, "feature", "start", "x", "--worktree")
	if got := worktreeExitCode(err); got != 6 {
		t.Fatalf("Expected exit code 6, got %d\nOutput: %s", got, output)
	}
	if _, err := os.Stat(sentinel); !os.IsNotExist(err) {
		t.Errorf("Expected the pre-start hook not to run, sentinel %q exists", sentinel)
	}

	assertNothingCreated(t, dir, "feature/x")
}

// TestStartWorktreeRunsPreStartHookOnViablePath covers E8 Test B, the non-vacuity
// control: on a viable path the pre-start hook DOES run, so an implementation
// whose worktree path never entered the hook wrapper cannot pass the suppression
// test.
// Steps:
// 1. Sets up a repository with git-flow defaults and a pre-flow-feature-start hook that writes a sentinel
// 2. Leaves the computed path free
// 3. Runs 'git flow feature start x --worktree'
// 4. Verifies exit code 0, that the sentinel exists and that the worktree was created
func TestStartWorktreeRunsPreStartHookOnViablePath(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer os.RemoveAll(worktreeRootFor(dir))

	sentinel := sentinelHook(t, dir, "pre-flow-feature-start")

	output, err := testutil.RunGitFlow(t, dir, "feature", "start", "x", "--worktree")
	if err != nil {
		t.Fatalf("feature start --worktree failed: %v\nOutput: %s", err, output)
	}
	if _, err := os.Stat(sentinel); err != nil {
		t.Errorf("Expected the pre-start hook to run and write %q: %v", sentinel, err)
	}

	assertWorktreeCreatedAt(t, dir, "feature/x", computedWorktreePath(t, dir, "feature/x"))
}

// TestStartWorktreeOccupiedPathSkipsFetch covers the other half of decisions.md
// SC-7: a doomed command runs no fetch either. Scenario 11 Test A is the paired
// control proving the fetch line does print on a viable path.
// Steps:
// 1. Sets up a repository with git-flow defaults and a configured remote
// 2. Creates a non-empty directory at the computed path for feature/x
// 3. Runs 'git flow feature start x --worktree --fetch'
// 4. Verifies exit code 6 and that stdout carries no fetch line
// 5. Verifies no branch, no registered worktree and no provenance marker
func TestStartWorktreeOccupiedPathSkipsFetch(t *testing.T) {
	t.Parallel()
	dir, remoteDir := testutil.SetupTestRepoWithRemote(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer testutil.CleanupTestRepo(t, remoteDir)
	defer os.RemoveAll(worktreeRootFor(dir))

	writeObstruction(t, computedWorktreePath(t, dir, "feature/x"), "keep.txt", "precious")

	output, err := testutil.RunGitFlow(t, dir, "feature", "start", "x", "--worktree", "--fetch")
	if got := worktreeExitCode(err); got != 6 {
		t.Fatalf("Expected exit code 6, got %d\nOutput: %s", got, output)
	}
	if strings.Contains(output, "Fetching from") {
		t.Errorf("Expected no fetch line on a refused command, got: %s", output)
	}

	assertNothingCreated(t, dir, "feature/x")
}

// TestStartWorktreeUsesFilteredBranchName covers E9 (decisions.md SC-7): the
// version filter runs before the worktree path is computed, so the worktree
// follows the filtered name.
// Steps:
// 1. Sets up a repository with git-flow defaults and a version filter rewriting x to x-filtered
// 2. Runs 'git flow feature start x --worktree'
// 3. Verifies the branch is feature/x-filtered
// 4. Verifies the worktree is at the computed path for feature/x-filtered
func TestStartWorktreeUsesFilteredBranchName(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer os.RemoveAll(worktreeRootFor(dir))

	startVersionFilter(t, dir)

	output, err := testutil.RunGitFlow(t, dir, "feature", "start", "x", "--worktree")
	if err != nil {
		t.Fatalf("feature start --worktree failed: %v\nOutput: %s", err, output)
	}

	if !testutil.BranchExists(t, dir, "feature/x-filtered") {
		t.Error("Expected branch feature/x-filtered to be created")
	}
	assertWorktreeCreatedAt(t, dir, "feature/x-filtered", computedWorktreePath(t, dir, "feature/x-filtered"))
}

// TestStartWorktreeChecksFilteredPathForOccupancy covers E9 Test B: the occupancy
// check runs against the FILTERED path, and refuses before anything is created
// under either name.
// Steps:
// 1. Sets up a repository with git-flow defaults and a version filter rewriting x to x-filtered
// 2. Creates a non-empty directory at the computed path for feature/x-filtered
// 3. Runs 'git flow feature start x --worktree'
// 4. Verifies exit code 6
// 5. Verifies all three negatives against BOTH the filtered and the unfiltered branch names
func TestStartWorktreeChecksFilteredPathForOccupancy(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer os.RemoveAll(worktreeRootFor(dir))

	startVersionFilter(t, dir)
	writeObstruction(t, computedWorktreePath(t, dir, "feature/x-filtered"), "keep.txt", "precious")

	output, err := testutil.RunGitFlow(t, dir, "feature", "start", "x", "--worktree")
	if got := worktreeExitCode(err); got != 6 {
		t.Fatalf("Expected exit code 6, got %d\nOutput: %s", got, output)
	}

	assertNothingCreated(t, dir, "feature/x-filtered")
	assertNothingCreated(t, dir, "feature/x")
}

// TestStartWorktreeIgnoresUnfilteredPathOccupancy covers E9 Test C: an obstruction
// at the PRE-filter path is irrelevant. This is the assertion that fails if the
// path validation is moved above the version filter.
// Steps:
// 1. Sets up a repository with git-flow defaults and a version filter rewriting x to x-filtered
// 2. Creates a non-empty directory at the computed path for the pre-filter name feature/x
// 3. Runs 'git flow feature start x --worktree'
// 4. Verifies exit code 0 and that the worktree landed at the filtered path
func TestStartWorktreeIgnoresUnfilteredPathOccupancy(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer os.RemoveAll(worktreeRootFor(dir))

	startVersionFilter(t, dir)
	writeObstruction(t, computedWorktreePath(t, dir, "feature/x"), "keep.txt", "precious")

	output, err := testutil.RunGitFlow(t, dir, "feature", "start", "x", "--worktree")
	if err != nil {
		t.Fatalf("feature start --worktree failed: %v\nOutput: %s", err, output)
	}

	assertWorktreeCreatedAt(t, dir, "feature/x-filtered", computedWorktreePath(t, dir, "feature/x-filtered"))
}

// TestStartWorktreeFailureKeepsBranch covers E10 (decisions.md SC-8): when worktree
// creation fails after the branch exists, the branch is kept and the error is
// reported. Deleting a branch on an error path risks destroying more than it
// repairs.
//
// The failure is injected with a READ-ONLY parent directory: the pre-check sees an
// absent leaf and passes, then MkdirAll fails with EACCES once the branch already
// exists. A regular-file ancestor would NOT work — os.Stat returns ENOTDIR there,
// which os.IsNotExist does not match, so the command would refuse in
// pre-validation and nothing would be created at all.
// Steps:
// 1. Sets up a repository with git-flow defaults and an empty GIT_FLOW_CD_FILE
// 2. Creates a read-only directory outside the repository and points gitflow.worktreePath inside it
// 3. Probes whether the filesystem enforces the write bit and skips if it does not
// 4. Runs 'git flow feature start x --worktree'
// 5. Verifies exit code 3 and that stderr reports the worktree failure
// 6. Verifies feature/x still exists with its base recorded, and that nothing else was left behind
func TestStartWorktreeFailureKeepsBranch(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	cdFile := cdFilePath(t)

	blocked := filepath.Join(t.TempDir(), "blocked")
	if err := os.Mkdir(blocked, 0755); err != nil {
		t.Fatalf("Failed to create the blocked directory: %v", err)
	}
	if err := os.Chmod(blocked, 0555); err != nil {
		t.Fatalf("Failed to make the blocked directory read-only: %v", err)
	}
	// Restored before TempDir's own cleanup, which would otherwise fail to
	// remove a read-only directory. Cleanups run last-registered-first.
	t.Cleanup(func() { os.Chmod(blocked, 0755) })

	// Guard by capability, not by UID: on a filesystem that ignores write-mode
	// bits a CORRECT implementation would succeed and fail this test.
	probe := filepath.Join(blocked, "probe")
	if err := os.Mkdir(probe, 0755); err == nil {
		os.Remove(probe)
		t.Skip("filesystem does not enforce write permissions")
	}

	template := filepath.Join(blocked, "{{ branch }}")
	if out, err := testutil.RunGit(t, dir, "config", "gitflow.worktreePath", template); err != nil {
		t.Fatalf("Failed to set the path template: %v\nOutput: %s", err, out)
	}

	stdout, stderr, err := testutil.RunGitFlowStreamsWithEnv(t, dir, cdEnv(cdFile), "feature", "start", "x", "--worktree")
	if got := worktreeExitCode(err); got != 3 {
		t.Fatalf("Expected exit code 3, got %d\nStdout: %s\nStderr: %s", got, stdout, stderr)
	}
	if !strings.Contains(stderr, "worktree") {
		t.Errorf("Expected stderr to report the worktree failure, got: %s", stderr)
	}

	if !testutil.BranchExists(t, dir, "feature/x") {
		t.Error("Expected feature/x to survive the failed worktree creation")
	}
	if base := testutil.GitConfigValue(t, dir, "gitflow.branch.feature/x.base"); base != "develop" {
		t.Errorf("Expected gitflow.branch.feature/x.base=develop, got %q", base)
	}
	if marker := testutil.GitConfigValue(t, dir, managedMarkerFor("feature/x")); marker != "" {
		t.Errorf("Expected no provenance marker after a failed creation, got %q", marker)
	}
	if path := registeredWorktreePath(t, dir, "feature/x"); path != "" {
		t.Errorf("Expected no worktree registered for feature/x, found one at %s", path)
	}
	if _, err := os.Stat(filepath.Join(blocked, "feature", "x")); !os.IsNotExist(err) {
		t.Errorf("Expected the target leaf not to exist under %q", blocked)
	}
	assertCDFileEmpty(t, cdFile)
}

// TestStartWorktreeCreatesNestedParentDirectories covers E11: a branch name with
// slashes computes a nested path whose intermediate directories are created.
// Steps:
// 1. Sets up a repository with git-flow defaults
// 2. Runs 'git flow feature start deep/nested/name --worktree'
// 3. Verifies exit code 0 and that the worktree exists at <root>/feature/deep/nested/name
// 4. Verifies the provenance marker round-trips the full branch name through the git subsection
func TestStartWorktreeCreatesNestedParentDirectories(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer os.RemoveAll(worktreeRootFor(dir))

	output, err := testutil.RunGitFlow(t, dir, "feature", "start", "deep/nested/name", "--worktree")
	if err != nil {
		t.Fatalf("feature start --worktree with a nested name failed: %v\nOutput: %s", err, output)
	}

	branch := "feature/deep/nested/name"
	assertWorktreeCreatedAt(t, dir, branch, computedWorktreePath(t, dir, branch))
	if marker := testutil.GitConfigValue(t, dir, "gitflow.worktree."+branch+".managed"); marker != "true" {
		t.Errorf("Expected gitflow.worktree.%s.managed=true, got %q", branch, marker)
	}
}

// TestStartWorktreeHonorsCustomPathTemplate covers E12: a custom
// gitflow.worktreePath template decides where the worktree goes.
// Steps:
// 1. Sets up a repository with git-flow defaults
// 2. Sets gitflow.worktreePath to '../{{ repo }}-wt/{{ topicType }}/{{ branchName }}'
// 3. Runs 'git flow feature start x --worktree'
// 4. Verifies the worktree is at <parent>/<repo>-wt/feature/x and registered for feature/x
// 5. Verifies the printed path matches
func TestStartWorktreeHonorsCustomPathTemplate(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	root := testutil.EvalPath(t, dir)
	customRoot := filepath.Join(filepath.Dir(root), filepath.Base(root)+"-wt")
	t.Cleanup(func() { os.RemoveAll(customRoot) })

	if out, err := testutil.RunGit(t, dir, "config", "gitflow.worktreePath", "../{{ repo }}-wt/{{ topicType }}/{{ branchName }}"); err != nil {
		t.Fatalf("Failed to set the path template: %v\nOutput: %s", err, out)
	}

	output, err := testutil.RunGitFlow(t, dir, "feature", "start", "x", "--worktree")
	if err != nil {
		t.Fatalf("feature start --worktree failed: %v\nOutput: %s", err, output)
	}

	want := filepath.Join(customRoot, "feature", "x")
	assertWorktreeCreatedAt(t, dir, "feature/x", want)
	if !strings.Contains(output, "Created worktree for branch 'feature/x' at "+want) {
		t.Errorf("Expected the worktree line to name %q, got: %s", want, output)
	}
}

// TestStartWorktreeHonorsLowercasedPathTemplateKey covers E12 Test B: the template
// is read case-insensitively, because git lowercases variable names in
// --get-regexp output and the key arrives as gitflow.worktreepath.
// Steps:
// 1. Sets up a repository with git-flow defaults
// 2. Sets the template under the lowercased key gitflow.worktreepath
// 3. Runs 'git flow feature start x --worktree'
// 4. Verifies the worktree is at <parent>/<repo>-wt/feature/x and registered for feature/x
func TestStartWorktreeHonorsLowercasedPathTemplateKey(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	root := testutil.EvalPath(t, dir)
	customRoot := filepath.Join(filepath.Dir(root), filepath.Base(root)+"-wt")
	t.Cleanup(func() { os.RemoveAll(customRoot) })

	if out, err := testutil.RunGit(t, dir, "config", "gitflow.worktreepath", "../{{ repo }}-wt/{{ topicType }}/{{ branchName }}"); err != nil {
		t.Fatalf("Failed to set the path template: %v\nOutput: %s", err, out)
	}

	output, err := testutil.RunGitFlow(t, dir, "feature", "start", "x", "--worktree")
	if err != nil {
		t.Fatalf("feature start --worktree failed: %v\nOutput: %s", err, output)
	}

	assertWorktreeCreatedAt(t, dir, "feature/x", filepath.Join(customRoot, "feature", "x"))
}

// TestStartWorktreeOccupiedPathWinsOverExistingBranch covers E14 (decisions.md
// SC-14): when the path is occupied AND the branch already exists, the occupied
// path is what the user is told about, because the validation runs before the
// branch-exists check.
//
// It is also the sharpest ordering probe in the suite: it fails the moment the
// pre-validation moves below executeStart's branch-exists check.
// Steps:
// 1. Sets up a repository with git-flow defaults and creates feature/x with a plain start
// 2. Returns to develop and records feature/x's tip
// 3. Creates a non-empty directory at the computed path for feature/x
// 4. Runs 'git flow feature start x --worktree'
// 5. Verifies exit code 6, not 4, and that stderr names the occupied path
// 6. Verifies no worktree and no marker, and that both the branch and the obstruction are untouched
func TestStartWorktreeOccupiedPathWinsOverExistingBranch(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer os.RemoveAll(worktreeRootFor(dir))

	if out, err := testutil.RunGitFlow(t, dir, "feature", "start", "x"); err != nil {
		t.Fatalf("feature start failed: %v\nOutput: %s", err, out)
	}
	if out, err := testutil.RunGit(t, dir, "checkout", "develop"); err != nil {
		t.Fatalf("Failed to check out develop: %v\nOutput: %s", err, out)
	}
	tipBefore, err := testutil.RunGit(t, dir, "rev-parse", "feature/x")
	if err != nil {
		t.Fatalf("Failed to resolve feature/x: %v", err)
	}

	wtPath := computedWorktreePath(t, dir, "feature/x")
	writeObstruction(t, wtPath, "keep.txt", "precious")

	stdout, stderr, err := testutil.RunGitFlowStreamsWithEnv(t, dir, nil, "feature", "start", "x", "--worktree")
	if got := worktreeExitCode(err); got != 6 {
		t.Fatalf("Expected exit code 6, got %d\nStdout: %s\nStderr: %s", got, stdout, stderr)
	}
	if !strings.Contains(stderr, wtPath) {
		t.Errorf("Expected stderr to name the occupied path %q, got: %s", wtPath, stderr)
	}

	if path := registeredWorktreePath(t, dir, "feature/x"); path != "" {
		t.Errorf("Expected no worktree registered for feature/x, found one at %s", path)
	}
	if marker := testutil.GitConfigValue(t, dir, managedMarkerFor("feature/x")); marker != "" {
		t.Errorf("Expected no provenance marker, got %q", marker)
	}
	tipAfter, err := testutil.RunGit(t, dir, "rev-parse", "feature/x")
	if err != nil {
		t.Fatalf("Failed to resolve feature/x: %v", err)
	}
	if strings.TrimSpace(tipAfter) != strings.TrimSpace(tipBefore) {
		t.Errorf("Expected feature/x to stay at %s, got %s", strings.TrimSpace(tipBefore), strings.TrimSpace(tipAfter))
	}
	assertFileContent(t, filepath.Join(wtPath, "keep.txt"), "precious")
}
