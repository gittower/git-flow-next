package cmd_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gittower/git-flow-next/test/testutil"
)

// managedMarkerFor returns the provenance marker key for a branch, the key that
// records git-flow as the creator of its worktree.
func managedMarkerFor(branch string) string {
	return "gitflow.worktree." + branch + ".managed"
}

// writeObstruction creates dir and a file inside it, the "occupied path" state
// every clobber scenario starts from. An EMPTY directory is deliberately not an
// obstruction (#172 decision 12), so the file is what makes the path occupied.
func writeObstruction(t *testing.T, dir string, name string, content string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("Failed to create obstruction directory %s: %v", dir, err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write obstruction file in %s: %v", dir, err)
	}
}

// assertFileContent fails unless path holds exactly want, proving a refused
// command left the user's data untouched.
func assertFileContent(t *testing.T, path string, want string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Expected %s to still exist: %v", path, err)
	}
	if string(data) != want {
		t.Errorf("Expected %s to hold %q, got %q", path, want, string(data))
	}
}

// ---------------------------------------------------------------------------
// Group A — checkout navigation (spec scenarios 1-9, 18)
// ---------------------------------------------------------------------------

// TestCheckoutNavigatesToExistingWorktree covers scenario 1: checkout navigates
// to the branch's worktree instead of switching the main worktree's branch.
// Steps:
// 1. Initializes git-flow, creates feature/x and a worktree for it
// 2. Records the branch checked out in the main worktree
// 3. Runs 'git flow feature checkout x' with GIT_FLOW_CD_FILE set to an empty file
// 4. Verifies the CD file holds the worktree path and the main worktree's branch is unchanged
// 5. Verifies stdout carries the worktree and cd lines and no 'Switched to branch' line
// 6. Verifies no shell-init tip is printed, since the channel is plainly in use
func TestCheckoutNavigatesToExistingWorktree(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer os.RemoveAll(worktreeRootFor(dir))
	createFreeBranch(t, dir, "feature/x")
	wtPath := addWorktree(t, dir, "feature/x")
	branchBefore := testutil.GetCurrentBranch(t, dir)
	cdFile := cdFilePath(t)

	stdout, stderr, err := testutil.RunGitFlowStreamsWithEnv(t, dir, cdEnv(cdFile), "feature", "checkout", "x")
	if err != nil {
		t.Fatalf("feature checkout failed: %v\nStderr: %s", err, stderr)
	}

	if got := readCDFile(t, cdFile); got != wtPath {
		t.Errorf("Expected CD file to hold %q, got %q", wtPath, got)
	}
	if got := testutil.GetCurrentBranch(t, dir); got != branchBefore {
		t.Errorf("Expected the main worktree to stay on %q, got %q", branchBefore, got)
	}
	if !strings.Contains(stdout, "Worktree for branch 'feature/x' at "+wtPath) {
		t.Errorf("Expected stdout to name the worktree, got %q", stdout)
	}
	if !strings.Contains(stdout, "To switch to it: cd "+wtPath) {
		t.Errorf("Expected stdout to carry the cd hint, got %q", stdout)
	}
	if strings.Contains(stdout, "Switched to branch") {
		t.Errorf("Expected no classic switch message, got %q", stdout)
	}
	if strings.Contains(stdout, "Tip:") {
		t.Errorf("Expected no shell-init tip while the channel is in use, got %q", stdout)
	}
}

// TestCheckoutWithoutWorktreeSwitchesBranch covers scenario 2: with no worktree
// and no --worktree, checkout behaves exactly as it does today.
// Steps:
// 1. Initializes git-flow and creates feature/x without checking it out
// 2. Runs 'git flow feature checkout x' with GIT_FLOW_CD_FILE set to an empty file
// 3. Verifies stdout is the classic 'Switched to branch' message
// 4. Verifies the main worktree is now on feature/x
// 5. Verifies the CD file is still zero-length, since a classic switch is not a navigation
// 6. Verifies no worktree was created
func TestCheckoutWithoutWorktreeSwitchesBranch(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	createFreeBranch(t, dir, "feature/x")
	cdFile := cdFilePath(t)

	stdout, stderr, err := testutil.RunGitFlowStreamsWithEnv(t, dir, cdEnv(cdFile), "feature", "checkout", "x")
	if err != nil {
		t.Fatalf("feature checkout failed: %v\nStderr: %s", err, stderr)
	}

	if want := "Switched to branch 'feature/x'\n"; stdout != want {
		t.Errorf("Expected stdout %q, got %q", want, stdout)
	}
	if got := testutil.GetCurrentBranch(t, dir); got != "feature/x" {
		t.Errorf("Expected to be on feature/x, got %q", got)
	}
	assertCDFileEmpty(t, cdFile)
	if _, err := os.Stat(computedWorktreePath(t, dir, "feature/x")); !os.IsNotExist(err) {
		t.Errorf("Expected no worktree to be created at the computed path")
	}
}

// TestCheckoutWorktreeFlagCreatesAndMarks covers scenario 3: --worktree creates
// the missing worktree, records provenance, and navigates to it.
// Steps:
// 1. Initializes git-flow and creates feature/x without checking it out
// 2. Runs 'git flow feature checkout x --worktree' with GIT_FLOW_CD_FILE set
// 3. Verifies the worktree exists at the computed path and git lists it
// 4. Verifies the provenance marker is set to true
// 5. Verifies the CD file holds the worktree path and stdout reports the creation
// 6. Verifies the main worktree's branch is unchanged
func TestCheckoutWorktreeFlagCreatesAndMarks(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer os.RemoveAll(worktreeRootFor(dir))
	createFreeBranch(t, dir, "feature/x")
	branchBefore := testutil.GetCurrentBranch(t, dir)
	cdFile := cdFilePath(t)

	stdout, stderr, err := testutil.RunGitFlowStreamsWithEnv(t, dir, cdEnv(cdFile), "feature", "checkout", "x", "--worktree")
	if err != nil {
		t.Fatalf("feature checkout --worktree failed: %v\nStderr: %s", err, stderr)
	}

	wtPath := computedWorktreePath(t, dir, "feature/x")
	if info, err := os.Stat(wtPath); err != nil || !info.IsDir() {
		t.Fatalf("Expected a worktree directory at %s: %v", wtPath, err)
	}
	if !strings.Contains(gitWorktreeList(t, dir), wtPath) {
		t.Errorf("Expected git worktree list to show %s", wtPath)
	}
	if got := testutil.GitConfigValue(t, dir, managedMarkerFor("feature/x")); got != "true" {
		t.Errorf("Expected the provenance marker to be true, got %q", got)
	}
	if got := readCDFile(t, cdFile); got != wtPath {
		t.Errorf("Expected CD file to hold %q, got %q", wtPath, got)
	}
	if !strings.Contains(stdout, "Created worktree for branch 'feature/x' at "+wtPath) {
		t.Errorf("Expected stdout to report the creation, got %q", stdout)
	}
	if got := testutil.GetCurrentBranch(t, dir); got != branchBefore {
		t.Errorf("Expected the main worktree to stay on %q, got %q", branchBefore, got)
	}
}

// TestCheckoutHandMadeWorktreeStaysUnmanaged covers scenario 4: arriving at a
// worktree git-flow did not create never writes a provenance marker.
// Steps:
// 1. Initializes git-flow and creates feature/x
// 2. Creates a worktree for it with plain 'git worktree add' at a path of the test's choosing
// 3. Runs 'git flow feature checkout x' with GIT_FLOW_CD_FILE set
// 4. Verifies the CD file holds the hand-made path
// 5. Verifies no provenance marker was written
// 6. Verifies 'git flow worktree list' still tags the row (unmanaged)
func TestCheckoutHandMadeWorktreeStaysUnmanaged(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	createFreeBranch(t, dir, "feature/x")
	handPath := filepath.Join(t.TempDir(), "hand")
	if out, err := testutil.RunGit(t, dir, "worktree", "add", handPath, "feature/x"); err != nil {
		t.Fatalf("Failed to create the hand-made worktree: %v\nOutput: %s", err, out)
	}
	cdFile := cdFilePath(t)

	stdout, stderr, err := testutil.RunGitFlowStreamsWithEnv(t, dir, cdEnv(cdFile), "feature", "checkout", "x")
	if err != nil {
		t.Fatalf("feature checkout failed: %v\nStderr: %s", err, stderr)
	}

	want := testutil.EvalPath(t, handPath)
	if got := readCDFile(t, cdFile); got != want {
		t.Errorf("Expected CD file to hold the hand-made path %q, got %q", want, got)
	}
	if got := testutil.GitConfigValue(t, dir, managedMarkerFor("feature/x")); got != "" {
		t.Errorf("Expected no provenance marker, got %q", got)
	}
	if !strings.Contains(stdout, "Worktree for branch 'feature/x'") {
		t.Errorf("Expected stdout to name the worktree, got %q", stdout)
	}

	list, err := testutil.RunGitFlow(t, dir, "worktree", "list")
	if err != nil {
		t.Fatalf("worktree list failed: %v\nOutput: %s", err, list)
	}
	if !strings.Contains(list, "(unmanaged)") {
		t.Errorf("Expected the hand-made worktree to stay unmanaged, got %q", list)
	}
}

// TestCheckoutWorktreeOccupiedPathFails covers scenario 5: --worktree refuses an
// occupied target path when --clobber is not given.
// Steps:
// 1. Initializes git-flow and creates feature/x
// 2. Creates the computed worktree path as a directory holding a file
// 3. Runs 'git flow feature checkout x --worktree' with GIT_FLOW_CD_FILE set
// 4. Verifies exit code 6 and that stderr names the occupied path
// 5. Verifies the CD file is still zero-length and the obstruction is untouched
// 6. Verifies no worktree was registered
func TestCheckoutWorktreeOccupiedPathFails(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer os.RemoveAll(worktreeRootFor(dir))
	createFreeBranch(t, dir, "feature/x")
	wtPath := computedWorktreePath(t, dir, "feature/x")
	writeObstruction(t, wtPath, "keep.txt", "precious")
	cdFile := cdFilePath(t)

	_, stderr, err := testutil.RunGitFlowStreamsWithEnv(t, dir, cdEnv(cdFile), "feature", "checkout", "x", "--worktree")
	if got := worktreeExitCode(err); got != 6 {
		t.Fatalf("Expected exit code 6, got %d\nStderr: %s", got, stderr)
	}
	if !strings.Contains(stderr, wtPath) {
		t.Errorf("Expected stderr to name %q, got %q", wtPath, stderr)
	}
	assertCDFileEmpty(t, cdFile)
	assertFileContent(t, filepath.Join(wtPath, "keep.txt"), "precious")
	if strings.Contains(gitWorktreeList(t, dir), wtPath) {
		t.Errorf("Expected no worktree to be registered at %s", wtPath)
	}
}

// TestCheckoutWorktreeClobberReplacesStaleDirectory covers scenario 6: --clobber
// replaces a plain directory in the way and creates the worktree there.
// Steps:
// 1. Initializes git-flow and creates feature/x
// 2. Creates the computed worktree path as a directory holding a file
// 3. Runs 'git flow feature checkout x --worktree --clobber' with GIT_FLOW_CD_FILE set
// 4. Verifies exit code 0 and that the stale file is gone
// 5. Verifies a worktree exists at the path, the marker is written and the CD file holds the path
// 6. Verifies stdout reports the creation
func TestCheckoutWorktreeClobberReplacesStaleDirectory(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer os.RemoveAll(worktreeRootFor(dir))
	createFreeBranch(t, dir, "feature/x")
	wtPath := computedWorktreePath(t, dir, "feature/x")
	writeObstruction(t, wtPath, "stale.txt", "old")
	cdFile := cdFilePath(t)

	stdout, stderr, err := testutil.RunGitFlowStreamsWithEnv(t, dir, cdEnv(cdFile), "feature", "checkout", "x", "--worktree", "--clobber")
	if err != nil {
		t.Fatalf("feature checkout --worktree --clobber failed: %v\nStderr: %s", err, stderr)
	}

	if _, err := os.Stat(filepath.Join(wtPath, "stale.txt")); !os.IsNotExist(err) {
		t.Errorf("Expected the stale file to be removed")
	}
	if !strings.Contains(gitWorktreeList(t, dir), wtPath) {
		t.Errorf("Expected git worktree list to show %s", wtPath)
	}
	if got := testutil.GitConfigValue(t, dir, managedMarkerFor("feature/x")); got != "true" {
		t.Errorf("Expected the provenance marker to be true, got %q", got)
	}
	if got := readCDFile(t, cdFile); got != wtPath {
		t.Errorf("Expected CD file to hold %q, got %q", wtPath, got)
	}
	if !strings.Contains(stdout, "Created worktree for branch 'feature/x' at "+wtPath) {
		t.Errorf("Expected stdout to report the creation, got %q", stdout)
	}
}

// TestCheckoutNoCDLeavesFileEmptyAndKeepsBranch covers scenario 7: --no-cd
// suppresses only the channel write.
// Steps:
// 1. Initializes git-flow, creates feature/x and a worktree for it
// 2. Records the branch checked out in the main worktree
// 3. Runs 'git flow feature checkout x --no-cd' with GIT_FLOW_CD_FILE set
// 4. Verifies the CD file is still zero-length
// 5. Verifies the main worktree's branch is unchanged
// 6. Verifies stdout still carries the human-readable cd hint
func TestCheckoutNoCDLeavesFileEmptyAndKeepsBranch(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer os.RemoveAll(worktreeRootFor(dir))
	createFreeBranch(t, dir, "feature/x")
	wtPath := addWorktree(t, dir, "feature/x")
	branchBefore := testutil.GetCurrentBranch(t, dir)
	cdFile := cdFilePath(t)

	stdout, stderr, err := testutil.RunGitFlowStreamsWithEnv(t, dir, cdEnv(cdFile), "feature", "checkout", "x", "--no-cd")
	if err != nil {
		t.Fatalf("feature checkout --no-cd failed: %v\nStderr: %s", err, stderr)
	}

	assertCDFileEmpty(t, cdFile)
	if got := testutil.GetCurrentBranch(t, dir); got != branchBefore {
		t.Errorf("Expected the main worktree to stay on %q, got %q", branchBefore, got)
	}
	if !strings.Contains(stdout, "To switch to it: cd "+wtPath) {
		t.Errorf("Expected stdout to keep the cd hint, got %q", stdout)
	}
}

// TestCheckoutWithoutCDFileEnvPrintsPath covers scenario 8: with the channel
// unused the path is printed for manual use, with the shell-init tip.
// Steps:
// 1. Initializes git-flow, creates feature/x and a worktree for it
// 2. Records the branch checked out in the main worktree
// 3. Runs 'git flow feature checkout x' with GIT_FLOW_CD_FILE unset
// 4. Verifies stdout equals exactly the two navigation lines plus the tip
// 5. Verifies stderr is empty
// 6. Verifies the main worktree's branch is unchanged, as in scenario 1
func TestCheckoutWithoutCDFileEnvPrintsPath(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer os.RemoveAll(worktreeRootFor(dir))
	createFreeBranch(t, dir, "feature/x")
	wtPath := addWorktree(t, dir, "feature/x")
	branchBefore := testutil.GetCurrentBranch(t, dir)

	stdout, stderr, err := testutil.RunGitFlowStreamsWithEnv(t, dir, nil, "feature", "checkout", "x")
	if err != nil {
		t.Fatalf("feature checkout failed: %v\nStderr: %s", err, stderr)
	}

	want := "Worktree for branch 'feature/x' at " + wtPath + "\n" +
		"To switch to it: cd " + wtPath + "\n" +
		"Tip: run 'git flow shell-init <shell>' for automatic directory switching\n"
	if stdout != want {
		t.Errorf("Expected stdout %q, got %q", want, stdout)
	}
	if stderr != "" {
		t.Errorf("Expected no output on stderr, got %q", stderr)
	}
	if got := testutil.GetCurrentBranch(t, dir); got != branchBefore {
		t.Errorf("Expected the main worktree to stay on %q, got %q", branchBefore, got)
	}
}

// TestCheckoutIgnoresBranchTypeWorktreeDefault covers scenario 9 as a FORWARD
// GUARD: checkout never auto-creates a worktree from a branch-type default.
//
// gitflow.branch.<type>.worktree does not exist until #173, so this test passes
// VACUOUSLY today. It is kept so #173 cannot accidentally teach checkout to
// honour the key; do not read it as live coverage.
// Steps:
// 1. Initializes git-flow and creates feature/x
// 2. Sets gitflow.branch.feature.worktree to true
// 3. Runs 'git flow feature checkout x' with GIT_FLOW_CD_FILE set
// 4. Verifies stdout is the classic 'Switched to branch' message
// 5. Verifies no worktree was created and the CD file is untouched
func TestCheckoutIgnoresBranchTypeWorktreeDefault(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	createFreeBranch(t, dir, "feature/x")
	if out, err := testutil.RunGit(t, dir, "config", "gitflow.branch.feature.worktree", "true"); err != nil {
		t.Fatalf("Failed to set the branch type worktree default: %v\nOutput: %s", err, out)
	}
	cdFile := cdFilePath(t)

	stdout, stderr, err := testutil.RunGitFlowStreamsWithEnv(t, dir, cdEnv(cdFile), "feature", "checkout", "x")
	if err != nil {
		t.Fatalf("feature checkout failed: %v\nStderr: %s", err, stderr)
	}

	if want := "Switched to branch 'feature/x'\n"; stdout != want {
		t.Errorf("Expected stdout %q, got %q", want, stdout)
	}
	if _, err := os.Stat(computedWorktreePath(t, dir, "feature/x")); !os.IsNotExist(err) {
		t.Errorf("Expected no worktree to be created")
	}
	assertCDFileEmpty(t, cdFile)
}

// TestCheckoutQuietSuppressesShellInitTip covers scenario 18: --quiet suppresses
// the shell-init tip.
// Steps:
// 1. Initializes git-flow, creates feature/x and a worktree for it
// 2. Runs 'git flow feature checkout x --quiet' with GIT_FLOW_CD_FILE unset, the only state where the tip would print
// 3. Verifies stdout equals exactly the two navigation lines
// 4. Verifies no Tip line is present and stderr is empty
func TestCheckoutQuietSuppressesShellInitTip(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer os.RemoveAll(worktreeRootFor(dir))
	createFreeBranch(t, dir, "feature/x")
	wtPath := addWorktree(t, dir, "feature/x")

	stdout, stderr, err := testutil.RunGitFlowStreamsWithEnv(t, dir, nil, "feature", "checkout", "x", "--quiet")
	if err != nil {
		t.Fatalf("feature checkout --quiet failed: %v\nStderr: %s", err, stderr)
	}

	want := "Worktree for branch 'feature/x' at " + wtPath + "\n" +
		"To switch to it: cd " + wtPath + "\n"
	if stdout != want {
		t.Errorf("Expected stdout %q, got %q", want, stdout)
	}
	if stderr != "" {
		t.Errorf("Expected no output on stderr, got %q", stderr)
	}
}

// ---------------------------------------------------------------------------
// Group B — checkout decisions with no spec scenario
// ---------------------------------------------------------------------------

// TestCheckoutBranchInMainWorktreeNavigatesToMainWorktree covers SC-6: a branch
// checked out in the main worktree is navigated to like any other worktree.
// Steps:
// 1. Initializes git-flow and creates feature/x and feature/other
// 2. Adds a worktree for feature/other so the command can run from a linked worktree
// 3. Checks feature/x out in the main worktree
// 4. Runs 'git flow feature checkout x' from the linked worktree with GIT_FLOW_CD_FILE set
// 5. Verifies the CD file holds the main worktree root and stdout names it
// 6. Verifies git's 'already used by worktree' failure never appears, because no classic switch is attempted
func TestCheckoutBranchInMainWorktreeNavigatesToMainWorktree(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer os.RemoveAll(worktreeRootFor(dir))
	createFreeBranch(t, dir, "feature/x")
	createFreeBranch(t, dir, "feature/other")
	linkedPath := addWorktree(t, dir, "feature/other")
	if out, err := testutil.RunGit(t, dir, "checkout", "feature/x"); err != nil {
		t.Fatalf("Failed to check feature/x out in the main worktree: %v\nOutput: %s", err, out)
	}
	cdFile := cdFilePath(t)

	stdout, stderr, err := testutil.RunGitFlowStreamsWithEnv(t, linkedPath, cdEnv(cdFile), "feature", "checkout", "x")
	if err != nil {
		t.Fatalf("feature checkout from a linked worktree failed: %v\nStderr: %s", err, stderr)
	}

	mainRoot := testutil.EvalPath(t, dir)
	if got := readCDFile(t, cdFile); got != mainRoot {
		t.Errorf("Expected CD file to hold the main worktree %q, got %q", mainRoot, got)
	}
	if !strings.Contains(stdout, "Worktree for branch 'feature/x' at "+mainRoot) {
		t.Errorf("Expected stdout to name the main worktree, got %q", stdout)
	}
	if strings.Contains(stderr, "already used by worktree") {
		t.Errorf("Expected no classic switch to be attempted, got %q", stderr)
	}
}

// TestCheckoutStaleWorktreeEntryFails covers SC-7: a worktree entry whose
// directory is gone is never handed to the shell as a destination.
// Steps:
// 1. Initializes git-flow, creates feature/x and a worktree for it
// 2. Deletes the worktree directory by hand, leaving the admin entry behind
// 3. Runs 'git flow feature checkout x' with GIT_FLOW_CD_FILE set
// 4. Verifies exit code 3 and that stderr names the recorded path and 'git flow worktree prune'
// 5. Verifies the CD file is still zero-length
// 6. Verifies the main worktree's branch was not switched
func TestCheckoutStaleWorktreeEntryFails(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer os.RemoveAll(worktreeRootFor(dir))
	createFreeBranch(t, dir, "feature/x")
	wtPath := addWorktree(t, dir, "feature/x")
	branchBefore := testutil.GetCurrentBranch(t, dir)
	if err := os.RemoveAll(wtPath); err != nil {
		t.Fatalf("Failed to remove the worktree directory: %v", err)
	}
	cdFile := cdFilePath(t)

	_, stderr, err := testutil.RunGitFlowStreamsWithEnv(t, dir, cdEnv(cdFile), "feature", "checkout", "x")
	if got := worktreeExitCode(err); got != 3 {
		t.Fatalf("Expected exit code 3, got %d\nStderr: %s", got, stderr)
	}
	if !strings.Contains(stderr, wtPath) {
		t.Errorf("Expected stderr to name the recorded path %q, got %q", wtPath, stderr)
	}
	if !strings.Contains(stderr, "git flow worktree prune") {
		t.Errorf("Expected stderr to point at 'git flow worktree prune', got %q", stderr)
	}
	assertCDFileEmpty(t, cdFile)
	if got := testutil.GetCurrentBranch(t, dir); got != branchBefore {
		t.Errorf("Expected the main worktree to stay on %q, got %q", branchBefore, got)
	}
}

// TestCheckoutClobberRefusesFile covers SC-4: --clobber refuses a file at the
// target path.
// Steps:
// 1. Initializes git-flow and creates feature/x
// 2. Creates the computed worktree path as a file with known content
// 3. Runs 'git flow feature checkout x --worktree --clobber' with GIT_FLOW_CD_FILE set
// 4. Verifies exit code 6 and that stderr refuses because the target is a file
// 5. Verifies the file still exists with its content intact
// 6. Verifies the CD file is still zero-length
func TestCheckoutClobberRefusesFile(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer os.RemoveAll(worktreeRootFor(dir))
	createFreeBranch(t, dir, "feature/x")
	wtPath := computedWorktreePath(t, dir, "feature/x")
	if err := os.MkdirAll(filepath.Dir(wtPath), 0755); err != nil {
		t.Fatalf("Failed to create the target's parent directory: %v", err)
	}
	if err := os.WriteFile(wtPath, []byte("not a worktree"), 0644); err != nil {
		t.Fatalf("Failed to create the obstructing file: %v", err)
	}
	cdFile := cdFilePath(t)

	_, stderr, err := testutil.RunGitFlowStreamsWithEnv(t, dir, cdEnv(cdFile), "feature", "checkout", "x", "--worktree", "--clobber")
	if got := worktreeExitCode(err); got != 6 {
		t.Fatalf("Expected exit code 6, got %d\nStderr: %s", got, stderr)
	}
	if !strings.Contains(stderr, "it is a file, not a directory") {
		t.Errorf("Expected stderr to refuse a file, got %q", stderr)
	}
	assertFileContent(t, wtPath, "not a worktree")
	assertCDFileEmpty(t, cdFile)
}

// TestCheckoutClobberRefusesRegisteredWorktree covers SC-4: --clobber refuses a
// registered worktree of this repository.
// Steps:
// 1. Initializes git-flow and sets a branch-independent path template so two branches compute the same target
// 2. Creates feature/x and feature/y and adds a worktree for feature/y at that shared target
// 3. Runs 'git flow feature checkout x --worktree --clobber' with GIT_FLOW_CD_FILE set
// 4. Verifies exit code 6 and that stderr points at 'git flow worktree remove'
// 5. Verifies the other worktree still exists and is still listed by git
// 6. Verifies the CD file is still zero-length
func TestCheckoutClobberRefusesRegisteredWorktree(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	// The template is repository-unique on purpose: filepath.Dir(dir) is the
	// shared system temp directory, so a fixed name would collide with every
	// other parallel test and concurrent 'go test' run.
	sharedRoot := filepath.Join(filepath.Dir(testutil.EvalPath(t, dir)), filepath.Base(dir)+"-shared-wt")
	defer os.RemoveAll(sharedRoot)
	if out, err := testutil.RunGit(t, dir, "config", "gitflow.worktreePath", "../{{ repo }}-shared-wt"); err != nil {
		t.Fatalf("Failed to set gitflow.worktreePath: %v\nOutput: %s", err, out)
	}
	createFreeBranch(t, dir, "feature/x")
	createFreeBranch(t, dir, "feature/y")
	if out, err := testutil.RunGitFlow(t, dir, "worktree", "add", "feature/y"); err != nil {
		t.Fatalf("Failed to add the shared worktree: %v\nOutput: %s", err, out)
	}
	cdFile := cdFilePath(t)

	_, stderr, err := testutil.RunGitFlowStreamsWithEnv(t, dir, cdEnv(cdFile), "feature", "checkout", "x", "--worktree", "--clobber")
	if got := worktreeExitCode(err); got != 6 {
		t.Fatalf("Expected exit code 6, got %d\nStderr: %s", got, stderr)
	}
	if !strings.Contains(stderr, "git flow worktree remove") {
		t.Errorf("Expected stderr to point at 'git flow worktree remove', got %q", stderr)
	}
	if _, err := os.Stat(sharedRoot); err != nil {
		t.Errorf("Expected the other worktree to survive: %v", err)
	}
	if !strings.Contains(gitWorktreeList(t, dir), sharedRoot) {
		t.Errorf("Expected git worktree list to still show %s", sharedRoot)
	}
	assertCDFileEmpty(t, cdFile)
}

// TestCheckoutClobberRefusesDirectoryWithGitEntry covers SC-4: --clobber refuses
// a directory holding a .git FILE, the linked-worktree form.
// Steps:
// 1. Initializes git-flow and creates feature/x
// 2. Creates the computed path as a directory holding a .git file and a decoy file
// 3. Runs 'git flow feature checkout x --worktree --clobber' with GIT_FLOW_CD_FILE set
// 4. Verifies exit code 6 and that stderr refuses because the directory looks like a repository
// 5. Verifies the directory and both files are intact
// 6. Verifies the CD file is still zero-length
func TestCheckoutClobberRefusesDirectoryWithGitEntry(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer os.RemoveAll(worktreeRootFor(dir))
	createFreeBranch(t, dir, "feature/x")
	wtPath := computedWorktreePath(t, dir, "feature/x")
	writeObstruction(t, wtPath, "decoy.txt", "work in progress")
	if err := os.WriteFile(filepath.Join(wtPath, ".git"), []byte("gitdir: /elsewhere\n"), 0644); err != nil {
		t.Fatalf("Failed to create the .git file: %v", err)
	}
	cdFile := cdFilePath(t)

	_, stderr, err := testutil.RunGitFlowStreamsWithEnv(t, dir, cdEnv(cdFile), "feature", "checkout", "x", "--worktree", "--clobber")
	if got := worktreeExitCode(err); got != 6 {
		t.Fatalf("Expected exit code 6, got %d\nStderr: %s", got, stderr)
	}
	if !strings.Contains(stderr, "it contains a .git entry") {
		t.Errorf("Expected stderr to refuse a repository-looking directory, got %q", stderr)
	}
	assertFileContent(t, filepath.Join(wtPath, "decoy.txt"), "work in progress")
	assertFileContent(t, filepath.Join(wtPath, ".git"), "gitdir: /elsewhere\n")
	assertCDFileEmpty(t, cdFile)
}

// TestCheckoutClobberRefusesDirectoryWithGitDirectory covers SC-4: --clobber
// refuses a directory holding a .git DIRECTORY, the ordinary-clone form.
//
// It is separate from the .git-file case because an implementation that stats
// the entry and checks !IsDir() passes that one and then removes somebody's
// clone.
// Steps:
// 1. Initializes git-flow and creates feature/x
// 2. Creates the computed path as a directory holding a .git/ directory with a file inside it, plus a sentinel file
// 3. Runs 'git flow feature checkout x --worktree --clobber' with GIT_FLOW_CD_FILE set
// 4. Verifies exit code 6 and that stderr refuses because the directory looks like a repository
// 5. Verifies the directory, the .git directory's content and the sentinel are all intact
// 6. Verifies the CD file is still zero-length
func TestCheckoutClobberRefusesDirectoryWithGitDirectory(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer os.RemoveAll(worktreeRootFor(dir))
	createFreeBranch(t, dir, "feature/x")
	wtPath := computedWorktreePath(t, dir, "feature/x")
	writeObstruction(t, wtPath, "sentinel.txt", "somebody's clone")
	writeObstruction(t, filepath.Join(wtPath, ".git"), "HEAD", "ref: refs/heads/main\n")
	cdFile := cdFilePath(t)

	_, stderr, err := testutil.RunGitFlowStreamsWithEnv(t, dir, cdEnv(cdFile), "feature", "checkout", "x", "--worktree", "--clobber")
	if got := worktreeExitCode(err); got != 6 {
		t.Fatalf("Expected exit code 6, got %d\nStderr: %s", got, stderr)
	}
	if !strings.Contains(stderr, "it contains a .git entry") {
		t.Errorf("Expected stderr to refuse a repository-looking directory, got %q", stderr)
	}
	assertFileContent(t, filepath.Join(wtPath, "sentinel.txt"), "somebody's clone")
	assertFileContent(t, filepath.Join(wtPath, ".git", "HEAD"), "ref: refs/heads/main\n")
	assertCDFileEmpty(t, cdFile)
}

// TestCheckoutClobberWithNothingToClobberSucceeds covers SC-4: --clobber with an
// empty target path is an ordinary creation.
// Steps:
// 1. Initializes git-flow and creates feature/x with nothing at the computed path
// 2. Runs 'git flow feature checkout x --worktree --clobber' with GIT_FLOW_CD_FILE set
// 3. Verifies exit code 0 and that the worktree was created and marked
// 4. Verifies stdout reports the creation and says nothing about clobbering
// 5. Verifies stderr is empty
func TestCheckoutClobberWithNothingToClobberSucceeds(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer os.RemoveAll(worktreeRootFor(dir))
	createFreeBranch(t, dir, "feature/x")
	cdFile := cdFilePath(t)

	stdout, stderr, err := testutil.RunGitFlowStreamsWithEnv(t, dir, cdEnv(cdFile), "feature", "checkout", "x", "--worktree", "--clobber")
	if err != nil {
		t.Fatalf("feature checkout --worktree --clobber failed: %v\nStderr: %s", err, stderr)
	}

	wtPath := computedWorktreePath(t, dir, "feature/x")
	if !strings.Contains(gitWorktreeList(t, dir), wtPath) {
		t.Errorf("Expected git worktree list to show %s", wtPath)
	}
	if got := testutil.GitConfigValue(t, dir, managedMarkerFor("feature/x")); got != "true" {
		t.Errorf("Expected the provenance marker to be true, got %q", got)
	}
	if !strings.Contains(stdout, "Created worktree for branch 'feature/x' at "+wtPath) {
		t.Errorf("Expected stdout to report the creation, got %q", stdout)
	}
	if strings.Contains(strings.ToLower(stdout), "removed") {
		t.Errorf("Expected no removal message when there was nothing to clobber, got %q", stdout)
	}
	if stderr != "" {
		t.Errorf("Expected no output on stderr, got %q", stderr)
	}
}

// TestCheckoutWorktreeFlagWithExistingWorktreeNavigates covers --worktree on a
// branch that already has one: it navigates and leaves provenance alone.
// Steps:
// 1. Initializes git-flow, creates feature/x and a worktree for it
// 2. Records the provenance marker value
// 3. Runs 'git flow feature checkout x --worktree' with GIT_FLOW_CD_FILE set
// 4. Verifies no second worktree was created and the CD file holds the existing path
// 5. Verifies stdout says 'Worktree for branch', not 'Created worktree'
// 6. Verifies the provenance marker is unchanged
func TestCheckoutWorktreeFlagWithExistingWorktreeNavigates(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer os.RemoveAll(worktreeRootFor(dir))
	createFreeBranch(t, dir, "feature/x")
	wtPath := addWorktree(t, dir, "feature/x")
	markerBefore := testutil.GitConfigValue(t, dir, managedMarkerFor("feature/x"))
	cdFile := cdFilePath(t)

	stdout, stderr, err := testutil.RunGitFlowStreamsWithEnv(t, dir, cdEnv(cdFile), "feature", "checkout", "x", "--worktree")
	if err != nil {
		t.Fatalf("feature checkout --worktree failed: %v\nStderr: %s", err, stderr)
	}

	if got := strings.Count(gitWorktreeList(t, dir), "\n"); got != 2 {
		t.Errorf("Expected exactly the main worktree and one linked worktree, got %q", gitWorktreeList(t, dir))
	}
	if got := readCDFile(t, cdFile); got != wtPath {
		t.Errorf("Expected CD file to hold %q, got %q", wtPath, got)
	}
	if !strings.Contains(stdout, "Worktree for branch 'feature/x' at "+wtPath) {
		t.Errorf("Expected stdout to name the existing worktree, got %q", stdout)
	}
	if strings.Contains(stdout, "Created worktree") {
		t.Errorf("Expected no creation message, got %q", stdout)
	}
	if got := testutil.GitConfigValue(t, dir, managedMarkerFor("feature/x")); got != markerBefore {
		t.Errorf("Expected provenance %q to be unchanged, got %q", markerBefore, got)
	}
}

// TestCheckoutNoCDStillSwitchesBranchWithoutWorktree covers SC-9: --no-cd never
// blocks the classic branch switch.
// Steps:
// 1. Initializes git-flow and creates feature/x without checking it out
// 2. Runs 'git flow feature checkout x --no-cd' with GIT_FLOW_CD_FILE set
// 3. Verifies the main worktree is now on feature/x
// 4. Verifies stdout is the classic 'Switched to branch' message
// 5. Verifies the CD file is still zero-length
func TestCheckoutNoCDStillSwitchesBranchWithoutWorktree(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	createFreeBranch(t, dir, "feature/x")
	cdFile := cdFilePath(t)

	stdout, stderr, err := testutil.RunGitFlowStreamsWithEnv(t, dir, cdEnv(cdFile), "feature", "checkout", "x", "--no-cd")
	if err != nil {
		t.Fatalf("feature checkout --no-cd failed: %v\nStderr: %s", err, stderr)
	}

	if got := testutil.GetCurrentBranch(t, dir); got != "feature/x" {
		t.Errorf("Expected to be on feature/x, got %q", got)
	}
	if want := "Switched to branch 'feature/x'\n"; stdout != want {
		t.Errorf("Expected stdout %q, got %q", want, stdout)
	}
	assertCDFileEmpty(t, cdFile)
}

// TestCheckoutPrefixMatchNavigatesToWorktree covers the worktree lookup running
// on the RESOLVED branch name, never on the user's raw argument.
// Steps:
// 1. Initializes git-flow, creates feature/user-auth and a worktree for it
// 2. Runs 'git flow feature checkout user' with the unique prefix and GIT_FLOW_CD_FILE set
// 3. Verifies the CD file holds feature/user-auth's worktree path
// 4. Verifies stdout names the resolved branch
func TestCheckoutPrefixMatchNavigatesToWorktree(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer os.RemoveAll(worktreeRootFor(dir))
	createFreeBranch(t, dir, "feature/user-auth")
	wtPath := addWorktree(t, dir, "feature/user-auth")
	cdFile := cdFilePath(t)

	stdout, stderr, err := testutil.RunGitFlowStreamsWithEnv(t, dir, cdEnv(cdFile), "feature", "checkout", "user")
	if err != nil {
		t.Fatalf("feature checkout by prefix failed: %v\nStderr: %s", err, stderr)
	}

	if got := readCDFile(t, cdFile); got != wtPath {
		t.Errorf("Expected CD file to hold %q, got %q", wtPath, got)
	}
	if !strings.Contains(stdout, "Worktree for branch 'feature/user-auth' at "+wtPath) {
		t.Errorf("Expected stdout to name the resolved branch, got %q", stdout)
	}
}

// TestCheckoutUnwritableCDFileWarnsAndSucceeds covers a destination file that
// cannot be written: the navigation still happens and the command still succeeds.
// Steps:
// 1. Initializes git-flow, creates feature/x and a worktree for it
// 2. Points GIT_FLOW_CD_FILE at a path inside a directory that does not exist
// 3. Runs 'git flow feature checkout x'
// 4. Verifies exit code 0 and that stdout still carries the navigation lines
// 5. Verifies stderr carries a Warning naming the destination file
func TestCheckoutUnwritableCDFileWarnsAndSucceeds(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer os.RemoveAll(worktreeRootFor(dir))
	createFreeBranch(t, dir, "feature/x")
	wtPath := addWorktree(t, dir, "feature/x")
	unwritable := filepath.Join(t.TempDir(), "missing-dir", "cd-destination")

	stdout, stderr, err := testutil.RunGitFlowStreamsWithEnv(t, dir, cdEnv(unwritable), "feature", "checkout", "x")
	if err != nil {
		t.Fatalf("Expected exit code 0, got %v\nStderr: %s", err, stderr)
	}

	if !strings.Contains(stdout, "Worktree for branch 'feature/x' at "+wtPath) {
		t.Errorf("Expected stdout to carry the navigation lines, got %q", stdout)
	}
	if !strings.Contains(stderr, "Warning:") || !strings.Contains(stderr, unwritable) {
		t.Errorf("Expected a warning naming %q, got %q", unwritable, stderr)
	}
}

// TestCheckoutCurrentMainWorktreeUsesClassicBehavior covers SC-14: when the
// destination is the worktree the command already runs in, nothing changes.
//
// Without this an implementation that navigates whenever the branch has ANY
// worktree — including the one it is standing in — would change the output of
// the most common checkout invocation and make the installed wrapper perform a
// no-op cd to the repository root.
// Steps:
// 1. Initializes git-flow and runs 'feature start x' so feature/x is checked out in the main worktree
// 2. Runs 'git flow feature checkout x' from the main worktree with GIT_FLOW_CD_FILE set
// 3. Verifies stdout equals exactly the classic 'Switched to branch' message
// 4. Verifies stderr is empty and no navigation or tip lines were printed
// 5. Verifies the CD file is still zero-length and the branch is still feature/x
func TestCheckoutCurrentMainWorktreeUsesClassicBehavior(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	if out, err := testutil.RunGitFlow(t, dir, "feature", "start", "x"); err != nil {
		t.Fatalf("feature start failed: %v\nOutput: %s", err, out)
	}
	cdFile := cdFilePath(t)

	stdout, stderr, err := testutil.RunGitFlowStreamsWithEnv(t, dir, cdEnv(cdFile), "feature", "checkout", "x")
	if err != nil {
		t.Fatalf("feature checkout failed: %v\nStderr: %s", err, stderr)
	}

	if want := "Switched to branch 'feature/x'\n"; stdout != want {
		t.Errorf("Expected stdout %q, got %q", want, stdout)
	}
	if stderr != "" {
		t.Errorf("Expected no output on stderr, got %q", stderr)
	}
	assertCDFileEmpty(t, cdFile)
	if got := testutil.GetCurrentBranch(t, dir); got != "feature/x" {
		t.Errorf("Expected to be on feature/x, got %q", got)
	}
}

// TestCheckoutClobberWithoutWorktreeIsSilentNoOp covers SC-15: --clobber without
// --worktree changes nothing and says nothing.
// Steps:
// 1. Initializes git-flow and creates feature/x
// 2. Creates the computed worktree path as a directory holding a sentinel file
// 3. Runs 'git flow feature checkout x --clobber' without --worktree, with GIT_FLOW_CD_FILE set
// 4. Verifies stdout equals exactly the classic 'Switched to branch' message and stderr is empty
// 5. Verifies the sentinel directory and its file are byte-identical
// 6. Verifies no worktree was created and the CD file is still zero-length
func TestCheckoutClobberWithoutWorktreeIsSilentNoOp(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer os.RemoveAll(worktreeRootFor(dir))
	createFreeBranch(t, dir, "feature/x")
	wtPath := computedWorktreePath(t, dir, "feature/x")
	writeObstruction(t, wtPath, "sentinel.txt", "do not delete")
	cdFile := cdFilePath(t)

	stdout, stderr, err := testutil.RunGitFlowStreamsWithEnv(t, dir, cdEnv(cdFile), "feature", "checkout", "x", "--clobber")
	if err != nil {
		t.Fatalf("feature checkout --clobber failed: %v\nStderr: %s", err, stderr)
	}

	if want := "Switched to branch 'feature/x'\n"; stdout != want {
		t.Errorf("Expected stdout %q, got %q", want, stdout)
	}
	if stderr != "" {
		t.Errorf("Expected no output on stderr, got %q", stderr)
	}
	if got := testutil.GetCurrentBranch(t, dir); got != "feature/x" {
		t.Errorf("Expected to be on feature/x, got %q", got)
	}
	assertFileContent(t, filepath.Join(wtPath, "sentinel.txt"), "do not delete")
	if strings.Contains(gitWorktreeList(t, dir), wtPath) {
		t.Errorf("Expected no worktree at %s", wtPath)
	}
	assertCDFileEmpty(t, cdFile)
}
