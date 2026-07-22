package cmd_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gittower/git-flow-next/internal/errors"
	"github.com/gittower/git-flow-next/test/testutil"
)

// =============================================================================
// Shared setup + assertion helpers for issue #143: --continue/--abort scope.
//
// These helpers set up the three "owner" in-progress states used repeatedly by
// the foreign-refusal scenarios, and implement the spec's non-destructive
// assertion set F. They live here (package cmd_test) so every scope/update test
// file can use them, alongside the integrate_test.go helpers (integRevParse,
// integMergeHeadExists, integRebaseInProgress, integStateExists, integAddCommit,
// integGitDir, integSetupConflict).
// =============================================================================

const (
	markerMergeHead = "MERGE_HEAD"
	markerRebaseDir = "rebase-merge"
)

// setupFinishInProgress leaves the repo in a finish merge conflict: Action=finish,
// .git/MERGE_HEAD present, HEAD on develop.
func setupFinishInProgress(t *testing.T, dir string) {
	t.Helper()
	if out, err := testutil.RunGitFlow(t, dir, "init", "--defaults"); err != nil {
		t.Fatalf("init failed: %v\n%s", err, out)
	}
	if out, err := testutil.RunGitFlow(t, dir, "feature", "start", "x"); err != nil {
		t.Fatalf("feature start failed: %v\n%s", err, out)
	}
	integAddCommit(t, dir, "feature/x", "c.txt", "feature\n", "feature c")
	integAddCommit(t, dir, "develop", "c.txt", "develop\n", "develop c")
	testutil.RunGit(t, dir, "checkout", "feature/x")
	if out, err := testutil.RunGitFlow(t, dir, "feature", "finish", "x"); err == nil {
		t.Fatalf("expected finish conflict, got success:\n%s", out)
	}
	if !integMergeHeadExists(t, dir) {
		t.Fatalf("expected MERGE_HEAD present after finish conflict")
	}
	if s, err := testutil.LoadMergeState(t, dir); err != nil || s == nil || s.Action != "finish" {
		t.Fatalf("expected saved finish state, got %+v (err=%v)", s, err)
	}
}

// setupUpdateInProgressMerge leaves the repo in an update merge conflict:
// Action=update, .git/MERGE_HEAD present, HEAD on feature/x (downstream=merge).
func setupUpdateInProgressMerge(t *testing.T, dir string) {
	t.Helper()
	if out, err := testutil.RunGitFlow(t, dir, "init", "--defaults"); err != nil {
		t.Fatalf("init failed: %v\n%s", err, out)
	}
	if _, err := testutil.RunGit(t, dir, "config", "gitflow.branch.feature.downstreamStrategy", "merge"); err != nil {
		t.Fatalf("failed to set downstream merge strategy: %v", err)
	}
	if out, err := testutil.RunGitFlow(t, dir, "feature", "start", "x"); err != nil {
		t.Fatalf("feature start failed: %v\n%s", err, out)
	}
	integAddCommit(t, dir, "feature/x", "c.txt", "feature\n", "feature c")
	integAddCommit(t, dir, "develop", "c.txt", "develop\n", "develop c")
	testutil.RunGit(t, dir, "checkout", "feature/x")
	if out, err := testutil.RunGitFlow(t, dir, "feature", "update", "x"); err == nil {
		t.Fatalf("expected update merge conflict, got success:\n%s", out)
	}
	if !integMergeHeadExists(t, dir) {
		t.Fatalf("expected MERGE_HEAD present after update merge conflict")
	}
	if s, err := testutil.LoadMergeState(t, dir); err != nil || s == nil || s.Action != "update" {
		t.Fatalf("expected saved update state, got %+v (err=%v)", s, err)
	}
}

// setupUpdateInProgressRebase leaves the repo in an update rebase conflict:
// Action=update, .git/rebase-merge present (default feature downstream=rebase).
func setupUpdateInProgressRebase(t *testing.T, dir string) {
	t.Helper()
	if out, err := testutil.RunGitFlow(t, dir, "init", "--defaults"); err != nil {
		t.Fatalf("init failed: %v\n%s", err, out)
	}
	if out, err := testutil.RunGitFlow(t, dir, "feature", "start", "x"); err != nil {
		t.Fatalf("feature start failed: %v\n%s", err, out)
	}
	integAddCommit(t, dir, "feature/x", "c.txt", "feature\n", "feature c")
	integAddCommit(t, dir, "develop", "c.txt", "develop\n", "develop c")
	testutil.RunGit(t, dir, "checkout", "feature/x")
	if out, err := testutil.RunGitFlow(t, dir, "feature", "update", "x"); err == nil {
		t.Fatalf("expected update rebase conflict, got success:\n%s", out)
	}
	if !integRebaseInProgress(t, dir) {
		t.Fatalf("expected .git/rebase-merge present after update rebase conflict")
	}
	if s, err := testutil.LoadMergeState(t, dir); err != nil || s == nil || s.Action != "update" {
		t.Fatalf("expected saved update state, got %+v (err=%v)", s, err)
	}
}

// setupIntegrateInProgress leaves the repo in an integrate merge conflict:
// Action=integrate, .git/MERGE_HEAD present, HEAD on main.
func setupIntegrateInProgress(t *testing.T, dir string) {
	t.Helper()
	if out, err := testutil.RunGitFlow(t, dir, "init", "--defaults"); err != nil {
		t.Fatalf("init failed: %v\n%s", err, out)
	}
	integSetupConflict(t, dir, "conflict.txt")
	if out, err := testutil.RunGitFlow(t, dir, "integrate", "develop", "--tag", "v2.0.0"); err == nil {
		t.Fatalf("expected integrate conflict, got success:\n%s", out)
	}
	if !integMergeHeadExists(t, dir) {
		t.Fatalf("expected MERGE_HEAD present after integrate conflict")
	}
	if s, err := testutil.LoadMergeState(t, dir); err != nil || s == nil || s.Action != "integrate" {
		t.Fatalf("expected saved integrate state, got %+v (err=%v)", s, err)
	}
}

// exitCodeOf returns the process exit code carried by a testutil.ExitError,
// 0 for a nil error, or -1 for any other error type.
func scopeExitCode(err error) int {
	if err == nil {
		return 0
	}
	if ee, ok := err.(*testutil.ExitError); ok {
		return ee.ExitCode
	}
	return -1
}

// readStateBytes returns the raw bytes of the merge-state file (fatal if absent).
func readStateBytes(t *testing.T, dir string) string {
	t.Helper()
	p := filepath.Join(integGitDir(t, dir), "gitflow", "state", "merge.json")
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("failed to read merge state file: %v", err)
	}
	return string(b)
}

// symbolicHead returns refs/heads/<branch> for HEAD, or "" when HEAD is detached
// (e.g. mid-rebase). It never fails the test.
func symbolicHead(t *testing.T, dir string) string {
	out, _ := testutil.RunGit(t, dir, "symbolic-ref", "-q", "HEAD")
	return strings.TrimSpace(out)
}

// showRef returns the full ref listing (branches + tags), used to prove no ref
// moved during a refused operation.
func showRef(t *testing.T, dir string) string {
	out, _ := testutil.RunGit(t, dir, "show-ref")
	return out
}

// porcelain returns `git status --porcelain` output.
func porcelain(t *testing.T, dir string) string {
	out, _ := testutil.RunGit(t, dir, "status", "--porcelain")
	return out
}

// markerPresent reports whether the given in-progress marker is present.
func markerPresent(t *testing.T, dir, marker string) bool {
	switch marker {
	case markerRebaseDir:
		return integRebaseInProgress(t, dir)
	default:
		return integMergeHeadExists(t, dir)
	}
}

// assertNonDestructiveRefusal runs cmdArgs against an in-progress foreign
// operation and asserts the spec's non-destructive set F: exit 3; output names
// wantOwner and mentions both --continue and --abort; the merge-state file bytes,
// current branch, HEAD, all refs, and `git status --porcelain` are unchanged; and
// the in-progress marker (wantMarker) is still present. Returns the command output.
func assertNonDestructiveRefusal(t *testing.T, dir string, cmdArgs []string, wantOwner, wantMarker string) string {
	t.Helper()

	stateBefore := readStateBytes(t, dir)
	branchBefore := symbolicHead(t, dir)
	headBefore := integRevParse(t, dir, "HEAD")
	refsBefore := showRef(t, dir)
	statusBefore := porcelain(t, dir)
	if !markerPresent(t, dir, wantMarker) {
		t.Fatalf("precondition failed: marker %s not present before refusal", wantMarker)
	}

	out, err := testutil.RunGitFlow(t, dir, cmdArgs...)

	if code := scopeExitCode(err); code != 3 {
		t.Fatalf("expected exit code 3, got %d (err=%v)\nOutput: %s", code, err, out)
	}
	if !strings.Contains(out, wantOwner) {
		t.Errorf("expected output to name owner %q, got:\n%s", wantOwner, out)
	}
	if !strings.Contains(out, "--continue") || !strings.Contains(out, "--abort") {
		t.Errorf("expected output to mention --continue and --abort, got:\n%s", out)
	}
	if got := readStateBytes(t, dir); got != stateBefore {
		t.Errorf("merge state file changed.\nbefore: %s\nafter:  %s", stateBefore, got)
	}
	if got := symbolicHead(t, dir); got != branchBefore {
		t.Errorf("current branch changed: before %q, after %q", branchBefore, got)
	}
	if got := integRevParse(t, dir, "HEAD"); got != headBefore {
		t.Errorf("HEAD moved: before %s, after %s", headBefore, got)
	}
	if got := showRef(t, dir); got != refsBefore {
		t.Errorf("refs changed.\nbefore:\n%s\nafter:\n%s", refsBefore, got)
	}
	if got := porcelain(t, dir); got != statusBefore {
		t.Errorf("working tree status changed.\nbefore: %q\nafter: %q", statusBefore, got)
	}
	if !markerPresent(t, dir, wantMarker) {
		t.Errorf("in-progress marker %s missing after refusal", wantMarker)
	}
	return out
}

// =============================================================================
// A. Foreign --continue is refused (owner != caller)
// =============================================================================

// TestFinishConflictRefusesIntegrateContinue verifies integrate --continue is
// refused over a finish conflict and the finish owner remains resumable.
// Steps:
// 1. Sets up a finish merge conflict (Action=finish, MERGE_HEAD present).
// 2. Runs 'git flow integrate --continue'.
// 3. Verifies non-destructive refusal naming the finish owner (set F, exit 3).
// 4. Resolves + stages, runs 'git flow feature finish --continue x'.
// 5. Verifies the finish still completes (branch deleted, exit 0).
func TestFinishConflictRefusesIntegrateContinue(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	setupFinishInProgress(t, dir)

	assertNonDestructiveRefusal(t, dir, []string{"integrate", "--continue"}, "finish", markerMergeHead)

	// Owner still resumable after the refusal (set F item 6).
	testutil.WriteFile(t, dir, "c.txt", "resolved\n")
	testutil.RunGit(t, dir, "add", "c.txt")
	out, err := testutil.RunGitFlow(t, dir, "feature", "finish", "--continue", "x")
	if err != nil {
		t.Fatalf("finish --continue after refusal failed: %v\n%s", err, out)
	}
	if testutil.BranchExists(t, dir, "feature/x") {
		t.Error("expected feature/x deleted after finish continue")
	}
	if testutil.GitFlowMergeStateExists(t, dir) {
		t.Error("expected merge state cleared after finish continue")
	}
}

// TestFinishConflictRefusesUpdateContinue verifies feature update --continue is
// refused over a finish conflict.
// Steps:
// 1. Sets up a finish merge conflict (Action=finish).
// 2. Runs 'git flow feature update --continue x'.
// 3. Verifies non-destructive refusal naming the finish owner (set F, exit 3).
func TestFinishConflictRefusesUpdateContinue(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	setupFinishInProgress(t, dir)

	assertNonDestructiveRefusal(t, dir, []string{"feature", "update", "--continue", "x"}, "finish", markerMergeHead)
}

// TestUpdateConflictRefusesFinishContinue verifies feature finish --continue is
// refused over an update conflict with an owner-aware message (not the confusing
// "unknown branch type" error).
// Steps:
// 1. Sets up an update merge conflict (Action=update).
// 2. Runs 'git flow feature finish --continue x'.
// 3. Verifies non-destructive refusal naming the update owner (set F, exit 3).
// 4. Verifies the message is NOT an InvalidBranchTypeError.
func TestUpdateConflictRefusesFinishContinue(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	setupUpdateInProgressMerge(t, dir)

	out := assertNonDestructiveRefusal(t, dir, []string{"feature", "finish", "--continue", "x"}, "update", markerMergeHead)
	if strings.Contains(out, "unknown branch type") {
		t.Errorf("expected owner-aware refusal, got InvalidBranchTypeError:\n%s", out)
	}
}

// TestUpdateConflictRefusesIntegrateContinue verifies integrate --continue is
// refused over an update conflict.
// Steps:
// 1. Sets up an update merge conflict (Action=update).
// 2. Runs 'git flow integrate --continue'.
// 3. Verifies non-destructive refusal naming the update owner (set F, exit 3).
func TestUpdateConflictRefusesIntegrateContinue(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	setupUpdateInProgressMerge(t, dir)

	assertNonDestructiveRefusal(t, dir, []string{"integrate", "--continue"}, "update", markerMergeHead)
}

// TestIntegrateConflictRefusesFinishContinue verifies feature finish --continue
// is refused over an integrate conflict and the integrate owner remains resumable.
// Steps:
// 1. Sets up an integrate merge conflict (Action=integrate, MERGE_HEAD present).
// 2. Runs 'git flow feature finish --continue x'.
// 3. Verifies non-destructive refusal naming the integrate owner (set F, exit 3).
// 4. Runs 'git flow integrate --abort' and verifies it cleanly aborts (exit 0).
func TestIntegrateConflictRefusesFinishContinue(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	setupIntegrateInProgress(t, dir)
	preMain := integRevParse(t, dir, "main")
	preDevelop := integRevParse(t, dir, "develop")

	out := assertNonDestructiveRefusal(t, dir, []string{"feature", "finish", "--continue", "x"}, "integrate", markerMergeHead)
	if !strings.Contains(out, "git flow integrate --continue") {
		t.Errorf("expected integrate recovery command in message, got:\n%s", out)
	}

	// Integrate owner still resumable after the refusal (set F item 6).
	if out, err := testutil.RunGitFlow(t, dir, "integrate", "--abort"); err != nil {
		t.Fatalf("integrate --abort after refusal failed: %v\n%s", err, out)
	}
	if integMergeHeadExists(t, dir) {
		t.Error("expected MERGE_HEAD gone after integrate abort")
	}
	if got := integRevParse(t, dir, "main"); got != preMain {
		t.Errorf("expected main restored to %s, got %s", preMain, got)
	}
	if got := integRevParse(t, dir, "develop"); got != preDevelop {
		t.Errorf("expected develop restored to %s, got %s", preDevelop, got)
	}
	if testutil.GitFlowMergeStateExists(t, dir) {
		t.Error("expected merge state cleared after integrate abort")
	}
}

// TestIntegrateConflictRefusesUpdateContinue verifies feature update --continue
// is refused over an integrate conflict.
// Steps:
// 1. Sets up an integrate merge conflict (Action=integrate).
// 2. Runs 'git flow feature update --continue x'.
// 3. Verifies non-destructive refusal naming the integrate owner (set F, exit 3).
func TestIntegrateConflictRefusesUpdateContinue(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	setupIntegrateInProgress(t, dir)

	assertNonDestructiveRefusal(t, dir, []string{"feature", "update", "--continue", "x"}, "integrate", markerMergeHead)
}

// =============================================================================
// B. Foreign --abort is refused (owner != caller)
// =============================================================================

// TestFinishConflictRefusesIntegrateAbort verifies integrate --abort is refused
// over a finish conflict and does not abort the finish merge.
// Steps:
// 1. Sets up a finish merge conflict (Action=finish, MERGE_HEAD present).
// 2. Runs 'git flow integrate --abort'.
// 3. Verifies non-destructive refusal (set F, exit 3) and MERGE_HEAD still present.
func TestFinishConflictRefusesIntegrateAbort(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	setupFinishInProgress(t, dir)

	assertNonDestructiveRefusal(t, dir, []string{"integrate", "--abort"}, "finish", markerMergeHead)
}

// TestFinishConflictRefusesUpdateAbort verifies feature update --abort is refused
// over a finish conflict.
// Steps:
// 1. Sets up a finish merge conflict (Action=finish).
// 2. Runs 'git flow feature update --abort x'.
// 3. Verifies non-destructive refusal naming the finish owner (set F, exit 3).
func TestFinishConflictRefusesUpdateAbort(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	setupFinishInProgress(t, dir)

	assertNonDestructiveRefusal(t, dir, []string{"feature", "update", "--abort", "x"}, "finish", markerMergeHead)
}

// TestUpdateConflictRefusesFinishAbort verifies feature finish --abort is refused
// over an update conflict (the core bug fix) and the update owner stays abortable.
// Steps:
//  1. Sets up an update merge conflict (Action=update, MERGE_HEAD present).
//  2. Runs 'git flow feature finish --abort x'.
//  3. Verifies non-destructive refusal naming the update owner (set F, exit 3);
//     the update merge is NOT aborted.
//  4. Runs 'git flow feature update --abort x' and verifies it cleanly aborts.
func TestUpdateConflictRefusesFinishAbort(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	setupUpdateInProgressMerge(t, dir)

	assertNonDestructiveRefusal(t, dir, []string{"feature", "finish", "--abort", "x"}, "update", markerMergeHead)

	// Update owner still abortable after the refusal (set F item 6).
	if out, err := testutil.RunGitFlow(t, dir, "feature", "update", "--abort", "x"); err != nil {
		t.Fatalf("feature update --abort after refusal failed: %v\n%s", err, out)
	}
	if integMergeHeadExists(t, dir) {
		t.Error("expected MERGE_HEAD gone after update abort")
	}
	if testutil.GitFlowMergeStateExists(t, dir) {
		t.Error("expected merge state cleared after update abort")
	}
	if got := testutil.GetCurrentBranch(t, dir); got != "feature/x" {
		t.Errorf("expected HEAD on feature/x after update abort, got %s", got)
	}
}

// TestUpdateConflictRefusesIntegrateAbort verifies integrate --abort is refused
// over an update conflict.
// Steps:
// 1. Sets up an update merge conflict (Action=update).
// 2. Runs 'git flow integrate --abort'.
// 3. Verifies non-destructive refusal naming the update owner (set F, exit 3).
func TestUpdateConflictRefusesIntegrateAbort(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	setupUpdateInProgressMerge(t, dir)

	assertNonDestructiveRefusal(t, dir, []string{"integrate", "--abort"}, "update", markerMergeHead)
}

// TestIntegrateConflictRefusesFinishAbort verifies feature finish --abort is
// refused over an integrate conflict.
// Steps:
// 1. Sets up an integrate merge conflict (Action=integrate).
// 2. Runs 'git flow feature finish --abort x'.
// 3. Verifies non-destructive refusal naming the integrate owner (set F, exit 3).
func TestIntegrateConflictRefusesFinishAbort(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	setupIntegrateInProgress(t, dir)

	assertNonDestructiveRefusal(t, dir, []string{"feature", "finish", "--abort", "x"}, "integrate", markerMergeHead)
}

// TestIntegrateConflictRefusesUpdateAbort verifies feature update --abort is
// refused over an integrate conflict.
// Steps:
// 1. Sets up an integrate merge conflict (Action=integrate).
// 2. Runs 'git flow feature update --abort x'.
// 3. Verifies non-destructive refusal naming the integrate owner (set F, exit 3).
func TestIntegrateConflictRefusesUpdateAbort(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	setupIntegrateInProgress(t, dir)

	assertNonDestructiveRefusal(t, dir, []string{"feature", "update", "--abort", "x"}, "integrate", markerMergeHead)
}

// =============================================================================
// C. Ordinary invocation (no flags) while a foreign op is in progress is refused
// =============================================================================

// TestPlainFinishRefusedDuringIntegrate verifies a plain feature finish is
// refused while an integrate op is in progress (guard fires before a new finish).
// Steps:
// 1. Sets up an integrate merge conflict (Action=integrate).
// 2. Runs 'git flow feature finish x' (no continue/abort flags).
// 3. Verifies non-destructive refusal naming the integrate owner (set F, exit 3).
func TestPlainFinishRefusedDuringIntegrate(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	setupIntegrateInProgress(t, dir)

	assertNonDestructiveRefusal(t, dir, []string{"feature", "finish", "x"}, "integrate", markerMergeHead)
}

// TestPlainUpdateRefusedDuringFinish verifies a plain feature update is refused
// while a finish op is in progress (the previously unguarded path).
// Steps:
// 1. Sets up a finish merge conflict (Action=finish).
// 2. Runs 'git flow feature update x' (no continue/abort flags).
// 3. Verifies non-destructive refusal naming the finish owner (set F, exit 3).
func TestPlainUpdateRefusedDuringFinish(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	setupFinishInProgress(t, dir)

	assertNonDestructiveRefusal(t, dir, []string{"feature", "update", "x"}, "finish", markerMergeHead)
}

// =============================================================================
// F. Nothing in progress: --continue reports "no merge in progress" (exit 3)
// =============================================================================

// TestFinishContinueNoMergeInProgress verifies finish --continue with nothing in
// progress errors with "no merge in progress" and exit 3.
// Steps:
// 1. init --defaults; feature start x (no conflict, no in-progress op).
// 2. Runs 'git flow feature finish --continue x'.
// 3. Verifies "no merge in progress" and exit 3.
func TestFinishContinueNoMergeInProgress(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	if out, err := testutil.RunGitFlow(t, dir, "init", "--defaults"); err != nil {
		t.Fatalf("init failed: %v\n%s", err, out)
	}
	if out, err := testutil.RunGitFlow(t, dir, "feature", "start", "x"); err != nil {
		t.Fatalf("feature start failed: %v\n%s", err, out)
	}

	out, err := testutil.RunGitFlow(t, dir, "feature", "finish", "--continue", "x")
	if code := scopeExitCode(err); code != 3 {
		t.Fatalf("expected exit 3, got %d (err=%v)\n%s", code, err, out)
	}
	if !strings.Contains(out, "no merge in progress") {
		t.Errorf("expected 'no merge in progress', got:\n%s", out)
	}
}

// TestIntegrateContinueNoMergeInProgress verifies integrate --continue with
// nothing in progress errors with "no merge in progress" and exit 3.
// Steps:
// 1. init --defaults (nothing in progress).
// 2. Runs 'git flow integrate --continue'.
// 3. Verifies "no merge in progress" and exit 3.
func TestIntegrateContinueNoMergeInProgress(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	if out, err := testutil.RunGitFlow(t, dir, "init", "--defaults"); err != nil {
		t.Fatalf("init failed: %v\n%s", err, out)
	}

	out, err := testutil.RunGitFlow(t, dir, "integrate", "--continue")
	if code := scopeExitCode(err); code != 3 {
		t.Fatalf("expected exit 3, got %d (err=%v)\n%s", code, err, out)
	}
	if !strings.Contains(out, "no merge in progress") {
		t.Errorf("expected 'no merge in progress', got:\n%s", out)
	}
}

// =============================================================================
// G. State integrity & edges
// =============================================================================

// setupBogusActionState sets up a real finish merge conflict, then overwrites the
// saved state's Action with the given value while leaving MERGE_HEAD in place.
func setupBogusActionState(t *testing.T, dir, action string) {
	t.Helper()
	setupFinishInProgress(t, dir)
	state, err := testutil.LoadMergeState(t, dir)
	if err != nil || state == nil {
		t.Fatalf("expected saved state, err=%v", err)
	}
	state.Action = action
	testutil.WriteMergeState(t, dir, state)
	if !integMergeHeadExists(t, dir) {
		t.Fatalf("expected MERGE_HEAD still present after rewriting state")
	}
}

// assertUnrecognizedRefusal runs cmdArgs and asserts an unrecognized-operation
// refusal: exit 3, a generic "unrecognized git-flow operation" message, the
// state-file bytes unchanged, and MERGE_HEAD still present. Nothing is cleared.
func assertUnrecognizedRefusal(t *testing.T, dir string, cmdArgs []string) {
	t.Helper()
	stateBefore := readStateBytes(t, dir)

	out, err := testutil.RunGitFlow(t, dir, cmdArgs...)
	if code := scopeExitCode(err); code != 3 {
		t.Fatalf("expected exit 3, got %d (err=%v)\n%s", code, err, out)
	}
	if !strings.Contains(out, "unrecognized git-flow operation") {
		t.Errorf("expected 'unrecognized git-flow operation' message, got:\n%s", out)
	}
	if got := readStateBytes(t, dir); got != stateBefore {
		t.Errorf("state file changed.\nbefore: %s\nafter:  %s", stateBefore, got)
	}
	if !integMergeHeadExists(t, dir) {
		t.Errorf("expected MERGE_HEAD still present after refusal")
	}
}

// TestEmptyActionRefusedNonDestructively verifies an empty-Action state is refused
// non-destructively (never auto-cleared).
// Steps:
// 1. Sets up a finish conflict, then overwrites Action="" (MERGE_HEAD kept).
// 2. Runs 'git flow feature finish --continue x'.
// 3. Verifies a generic unrecognized-operation refusal (exit 3), state kept.
func TestEmptyActionRefusedNonDestructively(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	setupBogusActionState(t, dir, "")

	assertUnrecognizedRefusal(t, dir, []string{"feature", "finish", "--continue", "x"})
}

// TestBogusActionRefusedNonDestructively verifies a bogus-Action state is refused
// non-destructively (never auto-cleared).
// Steps:
// 1. Sets up a finish conflict, then overwrites Action="bogus" (MERGE_HEAD kept).
// 2. Runs 'git flow integrate --continue'.
// 3. Verifies a generic unrecognized-operation refusal (exit 3), state kept.
func TestBogusActionRefusedNonDestructively(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	setupBogusActionState(t, dir, "bogus")

	assertUnrecognizedRefusal(t, dir, []string{"integrate", "--continue"})
}

// writeRawState overwrites the merge-state file with raw bytes, simulating a
// truncated JSON write from a crash mid-save. It bypasses the normal marshaller so
// the file exists but fails to parse.
func writeRawState(t *testing.T, dir, content string) {
	t.Helper()
	p := filepath.Join(integGitDir(t, dir), "gitflow", "state", "merge.json")
	if err := os.WriteFile(p, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write raw state file: %v", err)
	}
}

// assertUnparseableStateRefusal runs cmdArgs against a genuine git operation whose
// git-flow state file is unparseable (truncated). It asserts the spec's set F for
// the unrecognized case: exit 3, a generic "unrecognized git-flow operation"
// message, the state-file bytes byte-identical, MERGE_HEAD still present, and the
// current branch, HEAD, and all refs unchanged. Nothing is auto-cleared.
func assertUnparseableStateRefusal(t *testing.T, dir string, cmdArgs []string) {
	t.Helper()
	stateBefore := readStateBytes(t, dir)
	branchBefore := symbolicHead(t, dir)
	headBefore := integRevParse(t, dir, "HEAD")
	refsBefore := showRef(t, dir)
	if !integMergeHeadExists(t, dir) {
		t.Fatalf("precondition failed: MERGE_HEAD not present before refusal")
	}

	out, err := testutil.RunGitFlow(t, dir, cmdArgs...)
	if code := scopeExitCode(err); code != 3 {
		t.Fatalf("expected exit 3, got %d (err=%v)\n%s", code, err, out)
	}
	if !strings.Contains(out, "unrecognized git-flow operation") {
		t.Errorf("expected 'unrecognized git-flow operation' message, got:\n%s", out)
	}
	if got := readStateBytes(t, dir); got != stateBefore {
		t.Errorf("state file changed (should be byte-identical).\nbefore: %s\nafter:  %s", stateBefore, got)
	}
	if !integMergeHeadExists(t, dir) {
		t.Errorf("MERGE_HEAD missing after refusal (state was destructively cleared)")
	}
	if got := symbolicHead(t, dir); got != branchBefore {
		t.Errorf("current branch changed: before %q, after %q", branchBefore, got)
	}
	if got := integRevParse(t, dir, "HEAD"); got != headBefore {
		t.Errorf("HEAD moved: before %s, after %s", headBefore, got)
	}
	if got := showRef(t, dir); got != refsBefore {
		t.Errorf("refs changed.\nbefore:\n%s\nafter:\n%s", refsBefore, got)
	}
}

// TestTruncatedStateRefusesContinueNonDestructively verifies that --continue over a
// genuine git operation whose state file is truncated/unparseable is refused
// non-destructively, rather than falling through to IsMergeInProgress and deleting
// the state file while the git marker is still present (destructive refusal).
// Steps:
//  1. Sets up a finish merge conflict (MERGE_HEAD present), then overwrites the
//     state file with truncated JSON.
//  2. Runs 'git flow feature finish --continue x'.
//  3. Verifies exit 3, a generic unrecognized-operation refusal, state bytes
//     byte-identical, MERGE_HEAD kept, HEAD/branch/refs unchanged.
func TestTruncatedStateRefusesContinueNonDestructively(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	setupFinishInProgress(t, dir)
	writeRawState(t, dir, `{"action":"finish","branchType":"fea`)

	assertUnparseableStateRefusal(t, dir, []string{"feature", "finish", "--continue", "x"})
}

// TestTruncatedStateRefusesAbortNonDestructively verifies the same non-destructive
// refusal for --abort: a truncated state file over an active git merge must not be
// auto-cleared, and the git operation is left untouched (exit 3, not a silent no-op).
// Steps:
//  1. Sets up a finish merge conflict (MERGE_HEAD present), then overwrites the
//     state file with truncated JSON.
//  2. Runs 'git flow feature finish --abort x'.
//  3. Verifies exit 3, a generic unrecognized-operation refusal, state bytes
//     byte-identical, MERGE_HEAD kept, HEAD/branch/refs unchanged.
func TestTruncatedStateRefusesAbortNonDestructively(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	setupFinishInProgress(t, dir)
	writeRawState(t, dir, `{"action":"finish","branchType":"fea`)

	assertUnparseableStateRefusal(t, dir, []string{"feature", "finish", "--abort", "x"})
}

// TestForeignRefusalMessageContent verifies the foreign refusal message names the
// owner and prints the exact recovery commands.
// Steps:
//  1. Sets up an integrate merge conflict (Action=integrate).
//  2. Runs 'git flow feature finish --continue x'.
//  3. Verifies the message contains 'integrate', 'git flow integrate --continue',
//     and 'git flow integrate --abort'.
func TestForeignRefusalMessageContent(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	setupIntegrateInProgress(t, dir)

	out, err := testutil.RunGitFlow(t, dir, "feature", "finish", "--continue", "x")
	if code := scopeExitCode(err); code != 3 {
		t.Fatalf("expected exit 3, got %d (err=%v)\n%s", code, err, out)
	}
	for _, want := range []string{"integrate", "git flow integrate --continue", "git flow integrate --abort"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected message to contain %q, got:\n%s", want, out)
		}
	}
}

// TestTopLevelUpdateForeignRefusalExitsThree verifies the top-level update surface
// exits 3 (not 1) when refusing a foreign integrate op.
// Steps:
// 1. Sets up an integrate merge conflict (Action=integrate, HEAD on main).
// 2. Runs 'git flow update --continue' (top-level surface).
// 3. Verifies non-destructive refusal naming the integrate owner (set F, exit 3).
func TestTopLevelUpdateForeignRefusalExitsThree(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	setupIntegrateInProgress(t, dir)

	assertNonDestructiveRefusal(t, dir, []string{"update", "--continue"}, "integrate", markerMergeHead)
}

// setupLegacyUpdateState leaves the repo in a genuine update merge conflict, then
// overwrites the saved state with a legacy pre-#143 update state: valid JSON with a
// recognized Action="update" but an empty BranchType (one of the critical fields
// #143 now populates), keeping a real FullBranchName and CurrentStep="merge".
// MERGE_HEAD stays present, so the state is parseable and owner-matched yet
// structurally incomplete — the case that previously fell through to the owner
// path and triggered IsMergeInProgress's destructive auto-clear.
func setupLegacyUpdateState(t *testing.T, dir string) {
	t.Helper()
	setupUpdateInProgressMerge(t, dir)
	state, err := testutil.LoadMergeState(t, dir)
	if err != nil || state == nil {
		t.Fatalf("expected saved update state, err=%v", err)
	}
	state.BranchType = ""
	state.FullBranchName = "feature/x"
	state.CurrentStep = "merge"
	testutil.WriteMergeState(t, dir, state)
	if !integMergeHeadExists(t, dir) {
		t.Fatalf("expected MERGE_HEAD still present after rewriting state")
	}
}

// TestLegacyUpdateStateRefusesOwnerContinueNonDestructively verifies that the OWNER
// (feature update --continue) is refused non-destructively when the state is a legacy
// pre-#143 update state (recognized Action="update" but empty BranchType). Without the
// structural-completeness guard the owner check would match, the caller's
// IsMergeInProgress would reject the empty BranchType and DELETE the state file while
// MERGE_HEAD is still present (a destructive refusal).
// Steps:
//  1. Sets up a genuine update merge conflict, then overwrites the state with a
//     legacy update state (Action=update, BranchType="", CurrentStep=merge).
//  2. Runs 'git flow feature update --continue x' (the owner surface).
//  3. Verifies exit 3, a generic unrecognized-operation refusal, state bytes
//     byte-identical, MERGE_HEAD kept, HEAD/branch/refs unchanged.
func TestLegacyUpdateStateRefusesOwnerContinueNonDestructively(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	setupLegacyUpdateState(t, dir)

	assertUnparseableStateRefusal(t, dir, []string{"feature", "update", "--continue", "x"})
}

// TestLegacyUpdateStateRefusesOwnerAbortNonDestructively verifies the same
// non-destructive refusal for the owner --abort surface: a structurally-incomplete
// legacy update state over an active git merge must not be auto-cleared, and the git
// operation is left untouched (exit 3, not a silent no-op).
// Steps:
//  1. Sets up a genuine update merge conflict, then overwrites the state with a
//     legacy update state (Action=update, BranchType="", CurrentStep=merge).
//  2. Runs 'git flow feature update --abort x' (the owner surface).
//  3. Verifies exit 3, a generic unrecognized-operation refusal, state bytes
//     byte-identical, MERGE_HEAD kept, HEAD/branch/refs unchanged.
func TestLegacyUpdateStateRefusesOwnerAbortNonDestructively(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	setupLegacyUpdateState(t, dir)

	assertUnparseableStateRefusal(t, dir, []string{"feature", "update", "--abort", "x"})
}

// TestUnrecognizedOperationErrorOmitsEmptyBranchName verifies the cosmetic fix: when
// BranchName is empty (the loadErr/unparseable path constructs UnrecognizedOperationError{}),
// the message omits the "for '<name>'" clause entirely rather than printing "for ”".
// When BranchName is set, the message still names the branch.
func TestUnrecognizedOperationErrorOmitsEmptyBranchName(t *testing.T) {
	empty := (&errors.UnrecognizedOperationError{}).Error()
	if strings.Contains(empty, "for ''") {
		t.Errorf("empty-BranchName message must not contain \"for ''\", got:\n%s", empty)
	}
	if !strings.Contains(empty, "unrecognized git-flow operation") {
		t.Errorf("expected unrecognized-operation message, got:\n%s", empty)
	}

	named := (&errors.UnrecognizedOperationError{BranchName: "feature/x"}).Error()
	if !strings.Contains(named, "for 'feature/x'") {
		t.Errorf("expected named clause \"for 'feature/x'\", got:\n%s", named)
	}
}
