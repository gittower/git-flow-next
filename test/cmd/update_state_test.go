package cmd_test

import (
	"strings"
	"testing"

	"github.com/gittower/git-flow-next/test/testutil"
)

// =============================================================================
// F. Nothing in progress: update --abort no-op; update --continue no-merge
// =============================================================================

// TestUpdateAbortNoOpWhenNothingInProgress verifies feature update --abort with
// nothing in progress is a forgiving no-op.
// Steps:
// 1. init --defaults; feature start x (no conflict, no in-progress op).
// 2. Runs 'git flow feature update --abort x'.
// 3. Verifies exit 0, no error, and no merge-state file created.
func TestUpdateAbortNoOpWhenNothingInProgress(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	if out, err := testutil.RunGitFlow(t, dir, "init", "--defaults"); err != nil {
		t.Fatalf("init failed: %v\n%s", err, out)
	}
	if out, err := testutil.RunGitFlow(t, dir, "feature", "start", "x"); err != nil {
		t.Fatalf("feature start failed: %v\n%s", err, out)
	}

	out, err := testutil.RunGitFlow(t, dir, "feature", "update", "--abort", "x")
	if err != nil {
		t.Fatalf("expected update --abort no-op to succeed: %v\n%s", err, out)
	}
	if testutil.GitFlowMergeStateExists(t, dir) {
		t.Error("expected no merge-state file after no-op abort")
	}
}

// TestUpdateContinueNoMergeInProgress verifies feature update --continue with
// nothing in progress errors with "no merge in progress" and exit 3.
// Steps:
// 1. init --defaults; feature start x (no in-progress op).
// 2. Runs 'git flow feature update --continue x'.
// 3. Verifies "no merge in progress" and exit 3.
func TestUpdateContinueNoMergeInProgress(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	if out, err := testutil.RunGitFlow(t, dir, "init", "--defaults"); err != nil {
		t.Fatalf("init failed: %v\n%s", err, out)
	}
	if out, err := testutil.RunGitFlow(t, dir, "feature", "start", "x"); err != nil {
		t.Fatalf("feature start failed: %v\n%s", err, out)
	}

	out, err := testutil.RunGitFlow(t, dir, "feature", "update", "--continue", "x")
	if code := scopeExitCode(err); code != 3 {
		t.Fatalf("expected exit 3, got %d (err=%v)\n%s", code, err, out)
	}
	if !strings.Contains(out, "no merge in progress") {
		t.Errorf("expected 'no merge in progress', got:\n%s", out)
	}
}

// =============================================================================
// G. update state is self-describing; both-flags precedence; own-op in progress
// =============================================================================

// TestUpdateStateIsSelfDescribing verifies a topic update conflict saves a
// resolvable, self-describing state.
// Steps:
//  1. Sets up an update merge conflict on feature/x (downstream=merge).
//  2. Loads .git/gitflow/state/merge.json.
//  3. Verifies Action=update, BranchType=feature, FullBranchName=feature/x,
//     ParentBranch=develop, MergeStrategy=merge, CurrentStep=merge.
func TestUpdateStateIsSelfDescribing(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	setupUpdateInProgressMerge(t, dir)

	state, err := testutil.LoadMergeState(t, dir)
	if err != nil || state == nil {
		t.Fatalf("expected saved update state, err=%v", err)
	}
	if state.Action != "update" {
		t.Errorf("expected Action update, got %s", state.Action)
	}
	if state.BranchType != "feature" {
		t.Errorf("expected BranchType feature (resolvable), got %q", state.BranchType)
	}
	if state.FullBranchName != "feature/x" {
		t.Errorf("expected FullBranchName feature/x, got %s", state.FullBranchName)
	}
	if state.ParentBranch != "develop" {
		t.Errorf("expected ParentBranch develop, got %s", state.ParentBranch)
	}
	if state.MergeStrategy != "merge" {
		t.Errorf("expected MergeStrategy merge, got %s", state.MergeStrategy)
	}
	if state.CurrentStep != "merge" {
		t.Errorf("expected CurrentStep merge, got %s", state.CurrentStep)
	}
}

// TestBaseUpdateStateIsSelfDescribing verifies a base-branch update conflict saves
// a resolvable, self-describing state keyed on the base branch.
// Steps:
//  1. Sets up a develop <- main top-level update conflict.
//  2. Loads .git/gitflow/state/merge.json.
//  3. Verifies Action=update, BranchType=develop, FullBranchName=develop,
//     ParentBranch=main, CurrentStep=merge.
func TestBaseUpdateStateIsSelfDescribing(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	setupBaseUpdateInProgress(t, dir)

	state, err := testutil.LoadMergeState(t, dir)
	if err != nil || state == nil {
		t.Fatalf("expected saved update state, err=%v", err)
	}
	if state.Action != "update" {
		t.Errorf("expected Action update, got %s", state.Action)
	}
	if state.BranchType != "develop" {
		t.Errorf("expected BranchType develop (base key), got %q", state.BranchType)
	}
	if state.FullBranchName != "develop" {
		t.Errorf("expected FullBranchName develop, got %s", state.FullBranchName)
	}
	if state.ParentBranch != "main" {
		t.Errorf("expected ParentBranch main, got %s", state.ParentBranch)
	}
	if state.CurrentStep != "merge" {
		t.Errorf("expected CurrentStep merge, got %s", state.CurrentStep)
	}
}

// TestUpdateContinueAndAbortAbortWins verifies --continue and --abort together
// gives abort precedence.
// Steps:
// 1. Sets up an update merge conflict on feature/x; captures feature/x tip.
// 2. Runs 'git flow feature update --continue --abort x'.
// 3. Verifies the merge is aborted, feature/x restored, state cleared, exit 0.
func TestUpdateContinueAndAbortAbortWins(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	setupUpdateInProgressMerge(t, dir)

	preFeature := integRevParse(t, dir, "feature/x")

	out, err := testutil.RunGitFlow(t, dir, "feature", "update", "--continue", "--abort", "x")
	if err != nil {
		t.Fatalf("expected abort precedence to succeed: %v\n%s", err, out)
	}
	if integMergeHeadExists(t, dir) {
		t.Error("expected MERGE_HEAD gone after abort")
	}
	if got := integRevParse(t, dir, "feature/x"); got != preFeature {
		t.Errorf("expected feature/x restored to %s, got %s", preFeature, got)
	}
	if testutil.GitFlowMergeStateExists(t, dir) {
		t.Error("expected merge state cleared after abort")
	}
}

// TestPlainUpdateDuringOwnUpdateReportsInProgress verifies a plain feature update
// while its OWN update conflict is in progress reports a merge in progress rather
// than silently restarting.
// Steps:
// 1. Sets up an update merge conflict on feature/x (Action=update).
// 2. Runs 'git flow feature update x' (no continue/abort flags).
// 3. Verifies exit 3, the message names update, and the merge is untouched.
func TestPlainUpdateDuringOwnUpdateReportsInProgress(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	setupUpdateInProgressMerge(t, dir)

	stateBefore := readStateBytes(t, dir)
	out, err := testutil.RunGitFlow(t, dir, "feature", "update", "x")
	if code := scopeExitCode(err); code != 3 {
		t.Fatalf("expected exit 3, got %d (err=%v)\n%s", code, err, out)
	}
	if !strings.Contains(out, "update") {
		t.Errorf("expected message to name update, got:\n%s", out)
	}
	if !integMergeHeadExists(t, dir) {
		t.Error("expected MERGE_HEAD still present (not restarted)")
	}
	if got := readStateBytes(t, dir); got != stateBefore {
		t.Errorf("expected state unchanged.\nbefore: %s\nafter:  %s", stateBefore, got)
	}
}
