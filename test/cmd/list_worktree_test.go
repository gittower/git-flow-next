package cmd_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gittower/git-flow-next/test/testutil"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// listCell returns the worktree column of the row for the given short branch
// name in '<type> list --worktrees' output.
//
// The row is "  <name><padding>  <cell>", so everything after the name and the
// padding is the cell. The name is matched with a trailing space so a branch
// whose name is a prefix of another ("b01" vs "b010") cannot be picked up by
// mistake.
func listCell(t *testing.T, output string, name string) string {
	t.Helper()
	prefix := "  " + name + " "
	for _, line := range strings.Split(output, "\n") {
		if strings.HasPrefix(line, prefix) {
			return strings.TrimLeft(strings.TrimPrefix(line, "  "+name), " ")
		}
	}
	t.Fatalf("Expected a row for branch %q in output:\n%s", name, output)
	return ""
}

// relCell returns the path the worktree column is expected to show for target:
// its location relative to the main worktree root of the repository at dir.
//
// target must already be symlink-resolved — computedWorktreePath returns that
// form, and an existing directory goes through testutil.EvalPath — because git
// reports resolved paths and both sides of the comparison have to agree.
func relCell(t *testing.T, dir string, target string) string {
	t.Helper()
	rel, err := filepath.Rel(testutil.EvalPath(t, dir), target)
	if err != nil {
		t.Fatalf("Failed to compute the relative path of %q: %v", target, err)
	}
	return rel
}

// handMadeWorktree creates a worktree for branch with plain 'git worktree add',
// bypassing git-flow entirely so no provenance marker is written, and returns its
// symlink-resolved path. It lives in the test's own temporary directory, which
// the test framework removes.
func handMadeWorktree(t *testing.T, dir string, branch string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "by-hand")
	if out, err := testutil.RunGit(t, dir, "worktree", "add", path, branch); err != nil {
		t.Fatalf("Failed to add a worktree by hand for %s: %v\nOutput: %s", branch, err, out)
	}
	return testutil.EvalPath(t, path)
}

// failingGitShim returns the subprocess environment that puts a fake 'git' ahead
// of the real one on PATH. The fake execs the real git for every invocation
// except the one whose full argument string equals argv, for which it exits 128.
//
// The exit code is 128 and not 1 on purpose: GetConfigLocalRegexpLines treats
// exit 1 as "no matching keys" and returns success, so a shim failing with 1
// would make the marker-read test assert nothing.
//
// The shim is scoped to the child process, so tests using it stay parallel-safe.
func failingGitShim(t *testing.T, argv string) []string {
	t.Helper()
	realGit, err := exec.LookPath("git")
	if err != nil {
		t.Fatalf("Failed to locate the real git: %v", err)
	}
	shimDir := t.TempDir()
	script := fmt.Sprintf("#!/bin/sh\nif [ \"$*\" = '%s' ]; then\n\techo 'fatal: simulated git failure' >&2\n\texit 128\nfi\nexec %s \"$@\"\n", argv, realGit)
	if err := os.WriteFile(filepath.Join(shimDir, "git"), []byte(script), 0755); err != nil {
		t.Fatalf("Failed to write the git shim: %v", err)
	}
	return []string{"PATH=" + shimDir + string(os.PathListSeparator) + os.Getenv("PATH")}
}

// countTraceLines returns how many lines of a GIT_TRACE file contain every one of
// the given substrings.
func countTraceLines(lines []string, substrings ...string) int {
	count := 0
	for _, line := range lines {
		matched := true
		for _, substring := range substrings {
			if !strings.Contains(line, substring) {
				matched = false
				break
			}
		}
		if matched {
			count++
		}
	}
	return count
}

// ---------------------------------------------------------------------------
// Group A — spec scenarios
// ---------------------------------------------------------------------------

// TestListWorktreesShowsPathAndDash covers spec scenario 1: a branch with a
// worktree shows its path, a branch without one shows a dash. The whole stdout is
// asserted byte-for-byte, which is also what pins SC-1 — no header row, no
// Remote or Parent column, and prefix-stripped names.
// Steps:
// 1. Initializes git-flow and creates feature/user-auth with a worktree
// 2. Creates feature/docs with no worktree
// 3. Runs 'feature list --worktrees'
// 4. Verifies stdout equals the expected two-row table exactly, with empty stderr
func TestListWorktreesShowsPathAndDash(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer os.RemoveAll(worktreeRootFor(dir))

	createFreeBranch(t, dir, "feature/user-auth")
	wtPath := addWorktree(t, dir, "feature/user-auth")
	createFreeBranch(t, dir, "feature/docs")

	stdout, stderr, err := testutil.RunGitFlowStreams(t, dir, "feature", "list", "--worktrees")
	if err != nil {
		t.Fatalf("Failed to list feature branches: %v\nStderr: %s", err, stderr)
	}
	if stderr != "" {
		t.Errorf("Expected empty stderr, got: %s", stderr)
	}

	width := len("user-auth")
	want := "Feature branches:\n" +
		fmt.Sprintf("  %-*s  %s\n", width, "docs", "-") +
		fmt.Sprintf("  %-*s  %s\n", width, "user-auth", relCell(t, dir, wtPath))
	if stdout != want {
		t.Errorf("Expected stdout:\n%q\nGot:\n%q", want, stdout)
	}
}

// TestListWorktreesShowsChangeCount covers spec scenario 2: a worktree with
// uncommitted work shows a change count. Reinterpreted by SC-6 — the count is of
// porcelain entries, not files — so this setup builds three top-level entries,
// where both readings agree.
// Steps:
// 1. Creates feature/user-auth with a worktree
// 2. Modifies the tracked README.md and adds two untracked top-level files inside it
// 3. Runs 'feature list --worktrees'
// 4. Verifies the user-auth cell equals exactly "<relpath> [3]"
func TestListWorktreesShowsChangeCount(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer os.RemoveAll(worktreeRootFor(dir))

	createFreeBranch(t, dir, "feature/user-auth")
	wtPath := addWorktree(t, dir, "feature/user-auth")
	for _, name := range []string{"README.md", "a.txt", "b.txt"} {
		if err := os.WriteFile(filepath.Join(wtPath, name), []byte("changed"), 0644); err != nil {
			t.Fatalf("Failed to write %s in the worktree: %v", name, err)
		}
	}

	stdout, stderr, err := testutil.RunGitFlowStreams(t, dir, "feature", "list", "--worktrees")
	if err != nil {
		t.Fatalf("Failed to list feature branches: %v\nStderr: %s", err, stderr)
	}

	want := relCell(t, dir, wtPath) + " [3]"
	if cell := listCell(t, stdout, "user-auth"); cell != want {
		t.Errorf("Expected cell %q, got %q\nOutput:\n%s", want, cell, stdout)
	}
}

// TestListWorktreesCountsEntriesNotFiles pins SC-6: the annotation counts
// 'git status --porcelain' entries under git's default untracked handling, so an
// untracked directory collapses into one entry and -uall is never used.
// Steps:
// 1. Creates feature/user-auth with a worktree
// 2. Creates an untracked directory holding two files, plus one untracked top-level file
// 3. Runs 'feature list --worktrees'
// 4. Verifies the cell equals exactly "<relpath> [2]", not [3]
func TestListWorktreesCountsEntriesNotFiles(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer os.RemoveAll(worktreeRootFor(dir))

	createFreeBranch(t, dir, "feature/user-auth")
	wtPath := addWorktree(t, dir, "feature/user-auth")
	buildDir := filepath.Join(wtPath, "build")
	if err := os.MkdirAll(buildDir, 0755); err != nil {
		t.Fatalf("Failed to create the untracked directory: %v", err)
	}
	for _, name := range []string{"one.o", "two.o"} {
		if err := os.WriteFile(filepath.Join(buildDir, name), []byte("x"), 0644); err != nil {
			t.Fatalf("Failed to write %s: %v", name, err)
		}
	}
	if err := os.WriteFile(filepath.Join(wtPath, "notes.txt"), []byte("notes"), 0644); err != nil {
		t.Fatalf("Failed to write the untracked file: %v", err)
	}

	stdout, stderr, err := testutil.RunGitFlowStreams(t, dir, "feature", "list", "--worktrees")
	if err != nil {
		t.Fatalf("Failed to list feature branches: %v\nStderr: %s", err, stderr)
	}

	want := relCell(t, dir, wtPath) + " [2]"
	if cell := listCell(t, stdout, "user-auth"); cell != want {
		t.Errorf("Expected cell %q, got %q\nOutput:\n%s", want, cell, stdout)
	}
}

// TestListWorktreesCountsUntrackedDespiteStatusConfig pins the other half of
// SC-6: the count is defined against NORMAL untracked handling, not against
// whatever the user configured. status.showUntrackedFiles overrides git's
// default, so a user who set it to "no" would see a worktree holding untracked
// work reported as clean — the annotation would be silently wrong exactly where
// it matters.
// Steps:
// 1. Creates feature/user-auth with a worktree
// 2. Sets status.showUntrackedFiles=no in the repository
// 3. Writes one untracked top-level file into the worktree
// 4. Runs 'feature list --worktrees'
// 5. Verifies the cell equals exactly "<relpath> [1]"
func TestListWorktreesCountsUntrackedDespiteStatusConfig(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer os.RemoveAll(worktreeRootFor(dir))

	createFreeBranch(t, dir, "feature/user-auth")
	wtPath := addWorktree(t, dir, "feature/user-auth")
	if out, err := testutil.RunGit(t, dir, "config", "status.showUntrackedFiles", "no"); err != nil {
		t.Fatalf("Failed to set status.showUntrackedFiles: %v\nOutput: %s", err, out)
	}
	if err := os.WriteFile(filepath.Join(wtPath, "notes.txt"), []byte("notes"), 0644); err != nil {
		t.Fatalf("Failed to write the untracked file: %v", err)
	}

	stdout, stderr, err := testutil.RunGitFlowStreams(t, dir, "feature", "list", "--worktrees")
	if err != nil {
		t.Fatalf("Failed to list feature branches: %v\nStderr: %s", err, stderr)
	}

	want := relCell(t, dir, wtPath) + " [1]"
	if cell := listCell(t, stdout, "user-auth"); cell != want {
		t.Errorf("Expected cell %q, got %q\nOutput:\n%s", want, cell, stdout)
	}
}

// TestListWorktreesCleanWorktreeHasNoCount covers spec scenario 3: a clean
// worktree shows its path and nothing else. The cell is asserted by equality, so
// a "(clean)" tag or any other annotation fails.
// Steps:
// 1. Creates feature/user-auth with a worktree and writes nothing into it
// 2. Runs 'feature list --worktrees'
// 3. Verifies the cell equals exactly the relative path
func TestListWorktreesCleanWorktreeHasNoCount(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer os.RemoveAll(worktreeRootFor(dir))

	createFreeBranch(t, dir, "feature/user-auth")
	wtPath := addWorktree(t, dir, "feature/user-auth")

	stdout, stderr, err := testutil.RunGitFlowStreams(t, dir, "feature", "list", "--worktrees")
	if err != nil {
		t.Fatalf("Failed to list feature branches: %v\nStderr: %s", err, stderr)
	}

	want := relCell(t, dir, wtPath)
	if cell := listCell(t, stdout, "user-auth"); cell != want {
		t.Errorf("Expected cell %q, got %q\nOutput:\n%s", want, cell, stdout)
	}
}

// TestListWorktreesTagsOnlyUnmanagedWorktrees covers spec scenario 4: a worktree
// created with plain 'git worktree add' is tagged (unmanaged), one git-flow
// created is not. Both cells are asserted by equality, so an erroneous
// "(managed)" tag on git-flow's own worktree fails too.
// Steps:
// 1. Creates feature/by-flow with 'git flow worktree add'
// 2. Creates feature/by-hand with plain 'git worktree add'
// 3. Runs 'feature list --worktrees'
// 4. Verifies the by-hand cell carries the tag and the by-flow cell does not
func TestListWorktreesTagsOnlyUnmanagedWorktrees(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer os.RemoveAll(worktreeRootFor(dir))

	createFreeBranch(t, dir, "feature/by-flow")
	flowPath := addWorktree(t, dir, "feature/by-flow")
	createFreeBranch(t, dir, "feature/by-hand")
	handPath := handMadeWorktree(t, dir, "feature/by-hand")

	stdout, stderr, err := testutil.RunGitFlowStreams(t, dir, "feature", "list", "--worktrees")
	if err != nil {
		t.Fatalf("Failed to list feature branches: %v\nStderr: %s", err, stderr)
	}

	wantHand := relCell(t, dir, handPath) + " (unmanaged)"
	if cell := listCell(t, stdout, "by-hand"); cell != wantHand {
		t.Errorf("Expected cell %q, got %q\nOutput:\n%s", wantHand, cell, stdout)
	}
	wantFlow := relCell(t, dir, flowPath)
	if cell := listCell(t, stdout, "by-flow"); cell != wantFlow {
		t.Errorf("Expected cell %q, got %q\nOutput:\n%s", wantFlow, cell, stdout)
	}
}

// TestListWithoutWorktreesFlagOutputUnchanged covers spec scenario 5: without the
// flag the output is byte-identical to what list printed before this feature,
// even when a worktree exists.
// Steps:
// 1. Starts feature/api-v2 and creates feature/user-auth with a worktree
// 2. Runs 'feature list' without the flag
// 3. Verifies stdout is exactly the plain two-line listing, with empty stderr
func TestListWithoutWorktreesFlagOutputUnchanged(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer os.RemoveAll(worktreeRootFor(dir))

	if out, err := testutil.RunGitFlow(t, dir, "feature", "start", "api-v2"); err != nil {
		t.Fatalf("Failed to start feature: %v\nOutput: %s", err, out)
	}
	createFreeBranch(t, dir, "feature/user-auth")
	addWorktree(t, dir, "feature/user-auth")

	stdout, stderr, err := testutil.RunGitFlowStreams(t, dir, "feature", "list")
	if err != nil {
		t.Fatalf("Failed to list feature branches: %v\nStderr: %s", err, stderr)
	}
	if stderr != "" {
		t.Errorf("Expected empty stderr, got: %s", stderr)
	}

	want := "Feature branches:\n  api-v2\n  user-auth\n"
	if stdout != want {
		t.Errorf("Expected stdout:\n%q\nGot:\n%q", want, stdout)
	}
}

// TestListWorktreesPrintsNoCurrentBranchMarker replaces spec scenario 6, which is
// OVERRIDDEN by SC-2: list has never printed a '*' current-branch marker and none
// is added here, so the scenario as written is not constructible. Its inverse is
// asserted instead, which pins the decision rather than leaving it untested.
// Steps:
// 1. Starts feature/api-v2, leaving it checked out in the MAIN worktree
// 2. Creates feature/user-auth with a linked worktree
// 3. Runs 'feature list --worktrees'
// 4. Verifies '*' appears nowhere in stdout and the current branch's row reads exactly "  api-v2     -"
func TestListWorktreesPrintsNoCurrentBranchMarker(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer os.RemoveAll(worktreeRootFor(dir))

	if out, err := testutil.RunGitFlow(t, dir, "feature", "start", "api-v2"); err != nil {
		t.Fatalf("Failed to start feature: %v\nOutput: %s", err, out)
	}
	createFreeBranch(t, dir, "feature/user-auth")
	addWorktree(t, dir, "feature/user-auth")

	stdout, stderr, err := testutil.RunGitFlowStreams(t, dir, "feature", "list", "--worktrees")
	if err != nil {
		t.Fatalf("Failed to list feature branches: %v\nStderr: %s", err, stderr)
	}

	if strings.Contains(stdout, "*") {
		t.Errorf("Expected no '*' anywhere in stdout, got:\n%s", stdout)
	}
	wantRow := fmt.Sprintf("  %-*s  %s", len("user-auth"), "api-v2", "-")
	found := false
	for _, line := range strings.Split(stdout, "\n") {
		if line == wantRow {
			found = true
		}
	}
	if !found {
		t.Errorf("Expected a row %q, got:\n%s", wantRow, stdout)
	}
}

// TestListWorktreesShowsPathRelativeToMainWorktree covers spec scenario 7: the
// path is shown relative to the main worktree root (SC-7).
// Steps:
// 1. Creates feature/user-auth with a worktree at the computed sibling path
// 2. Runs 'feature list --worktrees'
// 3. Verifies the cell is the '..'-prefixed relative path and the absolute repository root never appears
func TestListWorktreesShowsPathRelativeToMainWorktree(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer os.RemoveAll(worktreeRootFor(dir))

	createFreeBranch(t, dir, "feature/user-auth")
	wtPath := addWorktree(t, dir, "feature/user-auth")

	stdout, stderr, err := testutil.RunGitFlowStreams(t, dir, "feature", "list", "--worktrees")
	if err != nil {
		t.Fatalf("Failed to list feature branches: %v\nStderr: %s", err, stderr)
	}

	want := relCell(t, dir, wtPath)
	if !strings.HasPrefix(want, "..") {
		t.Fatalf("Expected the computed worktree path to be a sibling of the repository, got %q", want)
	}
	if cell := listCell(t, stdout, "user-auth"); cell != want {
		t.Errorf("Expected cell %q, got %q\nOutput:\n%s", want, cell, stdout)
	}
	if root := testutil.EvalPath(t, dir); strings.Contains(stdout, root) {
		t.Errorf("Expected no absolute repository path %q in output:\n%s", root, stdout)
	}
}

// TestListWorktreesRelativePathIsSameFromLinkedWorktree pins SC-7's base: the
// paths are relative to the MAIN worktree root, not to the invocation directory,
// so standing inside a linked worktree does not change what is printed.
// Steps:
// 1. Creates feature/user-auth with a worktree
// 2. Runs 'feature list --worktrees' from the repository root
// 3. Runs it again from inside the linked worktree
// 4. Verifies both runs print the identical cell
func TestListWorktreesRelativePathIsSameFromLinkedWorktree(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer os.RemoveAll(worktreeRootFor(dir))

	createFreeBranch(t, dir, "feature/user-auth")
	wtPath := addWorktree(t, dir, "feature/user-auth")

	fromRoot, stderr, err := testutil.RunGitFlowStreams(t, dir, "feature", "list", "--worktrees")
	if err != nil {
		t.Fatalf("Failed to list from the repository root: %v\nStderr: %s", err, stderr)
	}
	fromWorktree, stderr, err := testutil.RunGitFlowStreams(t, wtPath, "feature", "list", "--worktrees")
	if err != nil {
		t.Fatalf("Failed to list from the linked worktree: %v\nStderr: %s", err, stderr)
	}

	rootCell := listCell(t, fromRoot, "user-auth")
	worktreeCell := listCell(t, fromWorktree, "user-auth")
	if rootCell != worktreeCell {
		t.Errorf("Expected the same cell from both directories, got %q from the root and %q from the worktree", rootCell, worktreeCell)
	}
	if want := relCell(t, dir, wtPath); rootCell != want {
		t.Errorf("Expected cell %q, got %q", want, rootCell)
	}
}

// TestListWorktreesMarksMissingWorktree covers spec scenario 8: a registered
// worktree whose directory was removed by hand is reported as stale, and that is
// visibly different from a branch that never had one (SC-5).
// Steps:
// 1. Creates feature/user-auth with a worktree and feature/docs without one
// 2. Removes the worktree directory, leaving git's admin entry behind
// 3. Runs 'feature list --worktrees'
// 4. Verifies exit 0, empty stderr, "<relpath> (missing)" for user-auth and "-" for docs
func TestListWorktreesMarksMissingWorktree(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer os.RemoveAll(worktreeRootFor(dir))

	createFreeBranch(t, dir, "feature/user-auth")
	wtPath := addWorktree(t, dir, "feature/user-auth")
	createFreeBranch(t, dir, "feature/docs")
	want := relCell(t, dir, wtPath) + " (missing)"
	if err := os.RemoveAll(wtPath); err != nil {
		t.Fatalf("Failed to remove the worktree directory: %v", err)
	}

	stdout, stderr, err := testutil.RunGitFlowStreams(t, dir, "feature", "list", "--worktrees")
	if code := worktreeExitCode(err); code != 0 {
		t.Fatalf("Expected exit code 0, got %d\nStderr: %s", code, stderr)
	}
	if stderr != "" {
		t.Errorf("Expected empty stderr for a stale row, got: %s", stderr)
	}
	if cell := listCell(t, stdout, "user-auth"); cell != want {
		t.Errorf("Expected cell %q, got %q\nOutput:\n%s", want, cell, stdout)
	}
	if cell := listCell(t, stdout, "docs"); cell != "-" {
		t.Errorf("Expected cell %q for a branch with no worktree, got %q", "-", cell)
	}
}

// TestListWorktreesDetectsMissingWithoutPrunableField pins that staleness is
// detected with os.Stat and never with the porcelain 'prunable' field, which is
// Git 2.36+ and deliberately dropped by the parser (#172).
// Steps:
// 1. Creates feature/user-auth with a worktree
// 2. Locks the worktree, so git reports 'locked' and never 'prunable' for it
// 3. Removes the worktree directory
// 4. Verifies the cell is still "<relpath> (missing)" and the command exits 0
func TestListWorktreesDetectsMissingWithoutPrunableField(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer os.RemoveAll(worktreeRootFor(dir))

	createFreeBranch(t, dir, "feature/user-auth")
	wtPath := addWorktree(t, dir, "feature/user-auth")
	want := relCell(t, dir, wtPath) + " (missing)"
	if out, err := testutil.RunGit(t, dir, "worktree", "lock", wtPath); err != nil {
		t.Fatalf("Failed to lock the worktree: %v\nOutput: %s", err, out)
	}
	if err := os.RemoveAll(wtPath); err != nil {
		t.Fatalf("Failed to remove the worktree directory: %v", err)
	}

	stdout, stderr, err := testutil.RunGitFlowStreams(t, dir, "feature", "list", "--worktrees")
	if code := worktreeExitCode(err); code != 0 {
		t.Fatalf("Expected exit code 0, got %d\nStderr: %s", code, stderr)
	}
	if cell := listCell(t, stdout, "user-auth"); cell != want {
		t.Errorf("Expected cell %q, got %q\nOutput:\n%s", want, cell, stdout)
	}
}

// ---------------------------------------------------------------------------
// Group B — decisions coverage
// ---------------------------------------------------------------------------

// TestListWorktreesShowsDashForMainWorktreeBranch covers SC-4: a branch checked
// out in the MAIN worktree shows '-', because this column reports LINKED
// worktrees. It diverges deliberately from #174's SC-6, where navigation does
// treat the main worktree as the branch's worktree — navigating somewhere and
// listing linked worktrees are different questions.
// Steps:
// 1. Starts feature/api-v2, which checks it out in the main worktree
// 2. Runs 'feature list --worktrees'
// 3. Verifies the cell is exactly "-" and neither the repository root nor "(unmanaged)" appears
func TestListWorktreesShowsDashForMainWorktreeBranch(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	if out, err := testutil.RunGitFlow(t, dir, "feature", "start", "api-v2"); err != nil {
		t.Fatalf("Failed to start feature: %v\nOutput: %s", err, out)
	}

	stdout, stderr, err := testutil.RunGitFlowStreams(t, dir, "feature", "list", "--worktrees")
	if err != nil {
		t.Fatalf("Failed to list feature branches: %v\nStderr: %s", err, stderr)
	}

	if cell := listCell(t, stdout, "api-v2"); cell != "-" {
		t.Errorf("Expected cell %q for a branch in the main worktree, got %q", "-", cell)
	}
	if root := testutil.EvalPath(t, dir); strings.Contains(stdout, root) {
		t.Errorf("Expected no repository root path %q in output:\n%s", root, stdout)
	}
	if strings.Contains(stdout, "(unmanaged)") {
		t.Errorf("Expected no '(unmanaged)' tag in output:\n%s", stdout)
	}
}

// TestListWorktreesMissingWorktreeKeepsUnmanagedTag covers SC-5: a stale row
// keeps its provenance tag and never carries a change count, which is the only
// place both annotations appear together.
// Steps:
// 1. Creates feature/by-hand with plain 'git worktree add', so no marker exists
// 2. Removes its directory
// 3. Runs 'feature list --worktrees'
// 4. Verifies the cell is exactly "<relpath> (missing) (unmanaged)" and the command exits 0
func TestListWorktreesMissingWorktreeKeepsUnmanagedTag(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	createFreeBranch(t, dir, "feature/by-hand")
	handPath := handMadeWorktree(t, dir, "feature/by-hand")
	want := relCell(t, dir, handPath) + " (missing) (unmanaged)"
	if err := os.RemoveAll(handPath); err != nil {
		t.Fatalf("Failed to remove the worktree directory: %v", err)
	}

	stdout, stderr, err := testutil.RunGitFlowStreams(t, dir, "feature", "list", "--worktrees")
	if code := worktreeExitCode(err); code != 0 {
		t.Fatalf("Expected exit code 0, got %d\nStderr: %s", code, stderr)
	}
	if cell := listCell(t, stdout, "by-hand"); cell != want {
		t.Errorf("Expected cell %q, got %q\nOutput:\n%s", want, cell, stdout)
	}
}

// TestListWorktreesDegradesWhenStatusFails covers SC-9: a per-worktree
// 'git status' failure degrades that one row and warns, rather than aborting the
// listing. The degraded cell is asserted by equality so it cannot be confused
// with SC-5's "(missing)" rendering, which describes a different state.
// Steps:
// 1. Creates feature/broken and feature/healthy, each with a worktree
// 2. Points the broken worktree's .git file at a directory that does not exist
// 3. Runs 'feature list --worktrees'
// 4. Verifies exit 0, the broken cell is the bare path, the healthy row is intact, and exactly one warning names the absolute path
func TestListWorktreesDegradesWhenStatusFails(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer os.RemoveAll(worktreeRootFor(dir))

	createFreeBranch(t, dir, "feature/broken")
	brokenPath := addWorktree(t, dir, "feature/broken")
	createFreeBranch(t, dir, "feature/healthy")
	healthyPath := addWorktree(t, dir, "feature/healthy")
	if err := os.WriteFile(filepath.Join(brokenPath, ".git"), []byte("gitdir: /definitely/not/here\n"), 0644); err != nil {
		t.Fatalf("Failed to break the worktree: %v", err)
	}

	stdout, stderr, err := testutil.RunGitFlowStreams(t, dir, "feature", "list", "--worktrees")
	if code := worktreeExitCode(err); code != 0 {
		t.Fatalf("Expected exit code 0, got %d\nStderr: %s", code, stderr)
	}

	if cell := listCell(t, stdout, "broken"); cell != relCell(t, dir, brokenPath) {
		t.Errorf("Expected the degraded cell %q, got %q\nOutput:\n%s", relCell(t, dir, brokenPath), cell, stdout)
	}
	if cell := listCell(t, stdout, "healthy"); cell != relCell(t, dir, healthyPath) {
		t.Errorf("Expected the healthy cell %q, got %q\nOutput:\n%s", relCell(t, dir, healthyPath), cell, stdout)
	}

	wantPrefix := fmt.Sprintf("Warning: failed to check status of worktree at %s:", brokenPath)
	warnings := 0
	for _, line := range strings.Split(strings.TrimRight(stderr, "\n"), "\n") {
		if line == "" {
			continue
		}
		if !strings.HasPrefix(line, wantPrefix) {
			t.Errorf("Expected every stderr line to start with %q, got %q", wantPrefix, line)
		}
		warnings++
	}
	if warnings != 1 {
		t.Errorf("Expected exactly one warning on stderr, got %d:\n%s", warnings, stderr)
	}
}

// TestListWorktreesFalseMarkerReadsUnmanagedEverywhere pins that the provenance
// marker's VALUE decides, not the key's presence, and that both commands that
// report provenance agree about it. 'worktree list' parses the value per row,
// this column reads the markers in bulk, and a hand-written 'false' is the case
// where key-presence matching and value parsing disagree — in the unsafe
// direction, since claiming a worktree is managed misleads about whether cleanup
// removes it or merely detaches it.
// Steps:
// 1. Creates feature/by-flow with 'git flow worktree add', which writes the marker as true
// 2. Rewrites the marker to false by hand, which is the only way that value arises
// 3. Runs 'feature list --worktrees' and verifies the cell carries "(unmanaged)"
// 4. Runs 'worktree list' and verifies its row for the same worktree carries the same tag
func TestListWorktreesFalseMarkerReadsUnmanagedEverywhere(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer os.RemoveAll(worktreeRootFor(dir))

	createFreeBranch(t, dir, "feature/by-flow")
	flowPath := addWorktree(t, dir, "feature/by-flow")
	if out, err := testutil.RunGit(t, dir, "config", "gitflow.worktree.feature/by-flow.managed", "false"); err != nil {
		t.Fatalf("Failed to rewrite the provenance marker: %v\nOutput: %s", err, out)
	}

	stdout, stderr, err := testutil.RunGitFlowStreams(t, dir, "feature", "list", "--worktrees")
	if err != nil {
		t.Fatalf("Failed to list feature branches: %v\nStderr: %s", err, stderr)
	}
	want := relCell(t, dir, flowPath) + " (unmanaged)"
	if cell := listCell(t, stdout, "by-flow"); cell != want {
		t.Errorf("Expected cell %q, got %q\nOutput:\n%s", want, cell, stdout)
	}

	output, err := testutil.RunGitFlow(t, dir, "worktree", "list")
	if err != nil {
		t.Fatalf("Failed to list worktrees: %v\nOutput: %s", err, output)
	}
	wantRow := fmt.Sprintf("feature/by-flow  %s  (unmanaged)", flowPath)
	if !strings.Contains(output, wantRow) {
		t.Errorf("Expected row %q in 'worktree list' output:\n%s", wantRow, output)
	}
}

// TestListWorktreesPaddedMarkerReadsManagedEverywhere pins the other direction of
// the same agreement: a marker stored with surrounding whitespace must read
// managed for both readers. 'worktree list' reads the value per row through a
// getter that trims, while this column reads the markers in bulk, where git puts
// its separator before the raw value — so an untrimmed bulk read would report
// (unmanaged) for a worktree git-flow created and still owns.
// Steps:
// 1. Creates feature/by-flow with 'git flow worktree add', which writes the marker as true
// 2. Rewrites the marker to " true " by hand, padding it with spaces
// 3. Runs 'feature list --worktrees' and verifies the cell carries no tag
// 4. Runs 'worktree list' and verifies its row for the same worktree carries none either
func TestListWorktreesPaddedMarkerReadsManagedEverywhere(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer os.RemoveAll(worktreeRootFor(dir))

	createFreeBranch(t, dir, "feature/by-flow")
	flowPath := addWorktree(t, dir, "feature/by-flow")
	if out, err := testutil.RunGit(t, dir, "config", "gitflow.worktree.feature/by-flow.managed", " true "); err != nil {
		t.Fatalf("Failed to pad the provenance marker: %v\nOutput: %s", err, out)
	}

	stdout, stderr, err := testutil.RunGitFlowStreams(t, dir, "feature", "list", "--worktrees")
	if err != nil {
		t.Fatalf("Failed to list feature branches: %v\nStderr: %s", err, stderr)
	}
	want := relCell(t, dir, flowPath)
	if cell := listCell(t, stdout, "by-flow"); cell != want {
		t.Errorf("Expected cell %q, got %q\nOutput:\n%s", want, cell, stdout)
	}

	output, err := testutil.RunGitFlow(t, dir, "worktree", "list")
	if err != nil {
		t.Fatalf("Failed to list worktrees: %v\nOutput: %s", err, output)
	}
	wantRow := fmt.Sprintf("feature/by-flow  %s", flowPath)
	if !strings.Contains(output, wantRow) || strings.Contains(output, "(unmanaged)") {
		t.Errorf("Expected an untagged row %q in 'worktree list' output:\n%s", wantRow, output)
	}
}

// TestListWorktreesMarksUnusableWorktreeMissing pins that "(missing)" means the
// worktree is not present as a worktree, not merely that os.Stat failed: a
// regular file at the recorded path raises ENOTDIR rather than ENOENT, which
// os.IsNotExist does not recognize, and it is exactly as unusable as an absent
// directory. The row must not degrade to a bare path, and must not warn.
// Steps:
// 1. Creates feature/by-flow with a worktree
// 2. Replaces its directory with a regular file of the same name
// 3. Runs 'feature list --worktrees'
// 4. Verifies exit 0, the cell equals exactly "<relpath> (missing)", and stderr is empty
func TestListWorktreesMarksUnusableWorktreeMissing(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer os.RemoveAll(worktreeRootFor(dir))

	createFreeBranch(t, dir, "feature/by-flow")
	flowPath := addWorktree(t, dir, "feature/by-flow")
	want := relCell(t, dir, flowPath) + " (missing)"
	if err := os.RemoveAll(flowPath); err != nil {
		t.Fatalf("Failed to remove the worktree directory: %v", err)
	}
	if err := os.WriteFile(flowPath, []byte("not a worktree\n"), 0644); err != nil {
		t.Fatalf("Failed to write a file at the worktree path: %v", err)
	}

	stdout, stderr, err := testutil.RunGitFlowStreams(t, dir, "feature", "list", "--worktrees")
	if code := worktreeExitCode(err); code != 0 {
		t.Fatalf("Expected exit code 0, got %d\nStderr: %s", code, stderr)
	}
	if stderr != "" {
		t.Errorf("Expected empty stderr, got: %s", stderr)
	}
	if cell := listCell(t, stdout, "by-flow"); cell != want {
		t.Errorf("Expected cell %q, got %q\nOutput:\n%s", want, cell, stdout)
	}
}

// TestListWorktreesRendersCountBeforeUnmanagedTag covers SC-11/SC-12: the
// annotation order is path, then change count, then provenance tag.
// Steps:
// 1. Creates feature/by-hand with plain 'git worktree add', so it is unmanaged
// 2. Writes two untracked top-level files inside it
// 3. Runs 'feature list --worktrees'
// 4. Verifies the cell is exactly "<relpath> [2] (unmanaged)"
func TestListWorktreesRendersCountBeforeUnmanagedTag(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	createFreeBranch(t, dir, "feature/by-hand")
	handPath := handMadeWorktree(t, dir, "feature/by-hand")
	for _, name := range []string{"a.txt", "b.txt"} {
		if err := os.WriteFile(filepath.Join(handPath, name), []byte("x"), 0644); err != nil {
			t.Fatalf("Failed to write %s: %v", name, err)
		}
	}

	stdout, stderr, err := testutil.RunGitFlowStreams(t, dir, "feature", "list", "--worktrees")
	if err != nil {
		t.Fatalf("Failed to list feature branches: %v\nStderr: %s", err, stderr)
	}

	want := relCell(t, dir, handPath) + " [2] (unmanaged)"
	if cell := listCell(t, stdout, "by-hand"); cell != want {
		t.Errorf("Expected cell %q, got %q\nOutput:\n%s", want, cell, stdout)
	}
}

// TestListRejectsSingularWorktreeFlag covers SC-8: the flag is the plural
// --worktrees, and the singular spelling is not silently accepted as an alias.
// Steps:
// 1. Initializes git-flow
// 2. Runs 'feature list --worktree'
// 3. Verifies exit 1, empty stdout, and an unknown-flag message on stderr
func TestListRejectsSingularWorktreeFlag(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	stdout, stderr, err := testutil.RunGitFlowStreams(t, dir, "feature", "list", "--worktree")
	if code := worktreeExitCode(err); code != 1 {
		t.Fatalf("Expected exit code 1, got %d\nStderr: %s", code, stderr)
	}
	if stdout != "" {
		t.Errorf("Expected empty stdout, got: %s", stdout)
	}
	if !strings.Contains(stderr, "unknown flag: --worktree") {
		t.Errorf("Expected stderr to report the unknown flag, got: %s", stderr)
	}
}

// TestListWorktreesFlagHasNoShorthand covers SC-8: no -w shorthand is registered.
// It is pinned through --help rather than by invoking the shorthand, since tests
// use long flag variants only.
// Steps:
// 1. Initializes git-flow
// 2. Runs 'feature list --help'
// 3. Verifies the flags block offers --worktrees and no "-w," shorthand
func TestListWorktreesFlagHasNoShorthand(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	output, err := testutil.RunGitFlow(t, dir, "feature", "list", "--help")
	if err != nil {
		t.Fatalf("Failed to show list help: %v\nOutput: %s", err, output)
	}
	if !strings.Contains(output, "--worktrees") {
		t.Errorf("Expected --worktrees in the help output, got:\n%s", output)
	}
	if strings.Contains(output, "-w,") {
		t.Errorf("Expected no -w shorthand in the help output, got:\n%s", output)
	}
}

// TestListWorktreesFlagWorksForCustomTopicType proves the flag comes from the
// single generic registration site, so every topic branch type gets it —
// including a user-defined one.
// Steps:
// 1. Initializes git-flow and adds a custom 'spike' topic type
// 2. Creates spike/x with a worktree
// 3. Runs 'spike list --worktrees'
// 4. Verifies exit 0 and that the cell is the worktree's relative path
func TestListWorktreesFlagWorksForCustomTopicType(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer os.RemoveAll(worktreeRootFor(dir))

	if out, err := testutil.RunGitFlow(t, dir, "config", "add", "topic", "spike", "develop", "--prefix=spike/"); err != nil {
		t.Fatalf("Failed to add the spike topic type: %v\nOutput: %s", err, out)
	}
	createFreeBranch(t, dir, "spike/x")
	wtPath := addWorktree(t, dir, "spike/x")

	stdout, stderr, err := testutil.RunGitFlowStreams(t, dir, "spike", "list", "--worktrees")
	if code := worktreeExitCode(err); code != 0 {
		t.Fatalf("Expected exit code 0, got %d\nStderr: %s", code, stderr)
	}
	want := relCell(t, dir, wtPath)
	if cell := listCell(t, stdout, "x"); cell != want {
		t.Errorf("Expected cell %q, got %q\nOutput:\n%s", want, cell, stdout)
	}
}

// TestListWorktreesAllDashesWithoutWorktrees covers the flag on a repository with
// no linked worktrees at all: every cell is a dash.
// Steps:
// 1. Starts feature/api-v2 and creates feature/user-auth, creating no worktrees
// 2. Runs 'feature list --worktrees'
// 3. Verifies exit 0 and that both cells are exactly "-"
func TestListWorktreesAllDashesWithoutWorktrees(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	if out, err := testutil.RunGitFlow(t, dir, "feature", "start", "api-v2"); err != nil {
		t.Fatalf("Failed to start feature: %v\nOutput: %s", err, out)
	}
	createFreeBranch(t, dir, "feature/user-auth")

	stdout, stderr, err := testutil.RunGitFlowStreams(t, dir, "feature", "list", "--worktrees")
	if code := worktreeExitCode(err); code != 0 {
		t.Fatalf("Expected exit code 0, got %d\nStderr: %s", code, stderr)
	}
	for _, name := range []string{"api-v2", "user-auth"} {
		if cell := listCell(t, stdout, name); cell != "-" {
			t.Errorf("Expected cell %q for %s, got %q", "-", name, cell)
		}
	}
}

// TestListWorktreesEmptyStillReportsNoBranches covers the flag on a repository
// with no branches of the type: the empty-state message is unchanged, and in
// particular carries no trailing period.
// Steps:
// 1. Initializes git-flow and creates no feature branches
// 2. Runs 'feature list --worktrees'
// 3. Verifies stdout is exactly "No feature branches found\n" and the command exits 0
func TestListWorktreesEmptyStillReportsNoBranches(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	stdout, stderr, err := testutil.RunGitFlowStreams(t, dir, "feature", "list", "--worktrees")
	if code := worktreeExitCode(err); code != 0 {
		t.Fatalf("Expected exit code 0, got %d\nStderr: %s", code, stderr)
	}
	if stdout != "No feature branches found\n" {
		t.Errorf("Expected stdout %q, got %q", "No feature branches found\n", stdout)
	}
}

// TestListWorktreesAbortsWhenWorktreeListFails covers SC-13: a failed BULK read
// aborts with a git error instead of degrading. Degrading here would print every
// branch as "-", asserting that no branch has a worktree — an active lie.
// Steps:
// 1. Creates feature/user-auth
// 2. Puts a git shim on PATH that fails 'worktree list --porcelain' with exit 128
// 3. Runs 'feature list --worktrees'
// 4. Verifies exit 3, an error on stderr, and no branch row on stdout
func TestListWorktreesAbortsWhenWorktreeListFails(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	createFreeBranch(t, dir, "feature/user-auth")
	env := failingGitShim(t, "worktree list --porcelain")

	stdout, stderr, err := testutil.RunGitFlowStreamsWithEnv(t, dir, env, "feature", "list", "--worktrees")
	if code := worktreeExitCode(err); code != 3 {
		t.Fatalf("Expected exit code 3, got %d\nStdout: %s\nStderr: %s", code, stdout, stderr)
	}
	if !strings.HasPrefix(stderr, "Error:") {
		t.Errorf("Expected stderr to start with 'Error:', got: %s", stderr)
	}
	if strings.Contains(stdout, "user-auth") {
		t.Errorf("Expected no branch row on stdout, got:\n%s", stdout)
	}
}

// TestListWorktreesAbortsWhenMarkerListFails covers SC-13 for the second bulk
// read. Degrading here would tag a git-flow-created worktree "(unmanaged)" — a
// lie in the direction that decides what the cleanup commands may delete.
// Steps:
// 1. Creates feature/user-auth with a worktree, so a provenance marker genuinely exists
// 2. Puts a git shim on PATH that fails the marker read with exit 128
// 3. Runs 'feature list --worktrees'
// 4. Verifies exit 3, no "(unmanaged)" tag and no branch row on stdout
func TestListWorktreesAbortsWhenMarkerListFails(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer os.RemoveAll(worktreeRootFor(dir))

	createFreeBranch(t, dir, "feature/user-auth")
	addWorktree(t, dir, "feature/user-auth")
	env := failingGitShim(t, `config --local --get-regexp ^gitflow\.worktree\..*\.managed$`)

	stdout, stderr, err := testutil.RunGitFlowStreamsWithEnv(t, dir, env, "feature", "list", "--worktrees")
	if code := worktreeExitCode(err); code != 3 {
		t.Fatalf("Expected exit code 3, got %d\nStdout: %s\nStderr: %s", code, stdout, stderr)
	}
	if strings.Contains(stdout, "(unmanaged)") {
		t.Errorf("Expected no '(unmanaged)' tag on stdout, got:\n%s", stdout)
	}
	if strings.Contains(stdout, "user-auth") {
		t.Errorf("Expected no branch row on stdout, got:\n%s", stdout)
	}
}

// TestListRejectsPatternArgument covers SC-14: list takes no arguments, so the
// pattern filtering its manpage used to document has never existed. This is the
// test that keeps the removed claim from quietly becoming true again.
// Steps:
// 1. Initializes git-flow
// 2. Runs 'feature list user-*'
// 3. Verifies exit 1 and an unknown-command message on stderr
func TestListRejectsPatternArgument(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	_, stderr, err := testutil.RunGitFlowStreams(t, dir, "feature", "list", "user-*")
	if code := worktreeExitCode(err); code != 1 {
		t.Fatalf("Expected exit code 1, got %d\nStderr: %s", code, stderr)
	}
	if !strings.Contains(stderr, `unknown command "user-*"`) {
		t.Errorf("Expected stderr to reject the argument, got: %s", stderr)
	}
}

// ---------------------------------------------------------------------------
// Group C — the performance guard
// ---------------------------------------------------------------------------

// TestListWorktreesUsesBulkReads pins the performance requirement: every row is
// resolved from bulk reads, so the column costs one 'worktree list', one marker
// read and one 'git status' per LIVE linked worktree — never a per-row
// WorktreeForBranch or IsManaged.
//
// GIT_TRACE is pointed at a FILE, never set to 1: the value 1 writes to stderr,
// which would fold trace lines into the output of git-flow's CombinedOutput call
// sites.
// Steps:
// 1. Creates ten feature branches and exactly one linked worktree
// 2. Runs 'feature list --worktrees' with GIT_TRACE pointed at a file
// 3. Counts the traced git invocations by kind
// 4. Verifies one branch listing, one worktree listing, one bulk marker read, no per-branch marker read and one status call
func TestListWorktreesUsesBulkReads(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer os.RemoveAll(worktreeRootFor(dir))

	for i := 1; i <= 10; i++ {
		createFreeBranch(t, dir, fmt.Sprintf("feature/b%02d", i))
	}
	addWorktree(t, dir, "feature/b01")

	tracePath := filepath.Join(t.TempDir(), "git-trace.log")
	output, err := testutil.RunGitFlowWithEnv(t, dir, []string{"GIT_TRACE=" + tracePath}, "feature", "list", "--worktrees")
	if err != nil {
		t.Fatalf("Failed to list feature branches: %v\nOutput: %s", err, output)
	}

	data, err := os.ReadFile(tracePath)
	if err != nil {
		t.Fatalf("Failed to read the trace file: %v", err)
	}
	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) == "" {
		t.Fatal("GIT_TRACE produced no output")
	}

	checks := []struct {
		what       string
		substrings []string
		want       int
	}{
		{"branch listings", []string{"git branch '--format="}, 1},
		{"worktree listings", []string{"git worktree list"}, 1},
		{"bulk marker reads", []string{"--get-regexp", "managed"}, 1},
		{"per-branch marker reads", []string{"gitflow.worktree."}, 0},
		{"status calls", []string{"git status --porcelain"}, 1},
	}
	for _, check := range checks {
		if got := countTraceLines(lines, check.substrings...); got != check.want {
			t.Errorf("Expected %d %s, got %d\nTrace:\n%s", check.want, check.what, got, string(data))
		}
	}
}
