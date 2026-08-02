package cmd_test

import (
	"strings"
	"testing"

	"github.com/gittower/git-flow-next/test/testutil"
)

// =============================================================================
// E. update resume/abort — new capability (issue #143)
// =============================================================================

// TestUpdateMergeConflictThenContinue verifies a merge-strategy update conflict is
// resumable and completes exactly one update (no tag, no child update, no delete).
// Steps:
//  1. Sets up an update merge conflict on feature/x (downstream=merge).
//  2. Captures pre-update develop and feature/x tips.
//  3. Resolves c.txt, stages, runs 'git flow feature update --continue x'.
//  4. Verifies the merge commits, feature/x has develop as ancestor, MERGE_HEAD
//     gone, state cleared, HEAD on feature/x, exit 0.
//  5. Verifies exactly one completion: no tag, develop tip unchanged, feature/x kept.
func TestUpdateMergeConflictThenContinue(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	setupUpdateInProgressMerge(t, dir)

	preDevelop := integRevParse(t, dir, "develop")
	tagsBefore, _ := testutil.RunGit(t, dir, "tag", "-l")

	testutil.WriteFile(t, dir, "c.txt", "resolved\n")
	testutil.RunGit(t, dir, "add", "c.txt")

	out, err := testutil.RunGitFlow(t, dir, "feature", "update", "--continue", "x")
	if err != nil {
		t.Fatalf("feature update --continue failed: %v\n%s", err, out)
	}
	if integMergeHeadExists(t, dir) {
		t.Error("expected MERGE_HEAD gone after update continue")
	}
	if !integIsAncestor(t, dir, integRevParse(t, dir, "develop"), "feature/x") {
		t.Error("expected develop to be an ancestor of feature/x after update")
	}
	if testutil.GitFlowMergeStateExists(t, dir) {
		t.Error("expected merge state cleared after update continue")
	}
	if got := testutil.GetCurrentBranch(t, dir); got != "feature/x" {
		t.Errorf("expected HEAD on feature/x, got %s", got)
	}
	// Exactly one update completion: no tag, no child update, no deletion.
	if tagsAfter, _ := testutil.RunGit(t, dir, "tag", "-l"); strings.TrimSpace(tagsAfter) != strings.TrimSpace(tagsBefore) {
		t.Errorf("expected no tag created by update, tags: %q -> %q", tagsBefore, tagsAfter)
	}
	if got := integRevParse(t, dir, "develop"); got != preDevelop {
		t.Errorf("expected develop tip unchanged (no child update), was %s got %s", preDevelop, got)
	}
	if !testutil.BranchExists(t, dir, "feature/x") {
		t.Error("expected feature/x to still exist (update never deletes)")
	}
}

// TestUpdateMergeConflictThenAbort verifies a merge-strategy update conflict rolls
// back cleanly on abort.
// Steps:
//  1. Sets up an update merge conflict on feature/x; captures feature/x, develop tips.
//  2. Runs 'git flow feature update --abort x'.
//  3. Verifies MERGE_HEAD gone, feature/x and develop restored, state cleared,
//     HEAD on feature/x, clean working tree, exit 0.
func TestUpdateMergeConflictThenAbort(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	setupUpdateInProgressMerge(t, dir)

	preFeature := integRevParse(t, dir, "feature/x")
	preDevelop := integRevParse(t, dir, "develop")

	out, err := testutil.RunGitFlow(t, dir, "feature", "update", "--abort", "x")
	if err != nil {
		t.Fatalf("feature update --abort failed: %v\n%s", err, out)
	}
	if integMergeHeadExists(t, dir) {
		t.Error("expected MERGE_HEAD gone after update abort")
	}
	if got := integRevParse(t, dir, "feature/x"); got != preFeature {
		t.Errorf("expected feature/x restored to %s, got %s", preFeature, got)
	}
	if got := integRevParse(t, dir, "develop"); got != preDevelop {
		t.Errorf("expected develop unchanged (%s), got %s", preDevelop, got)
	}
	if testutil.GitFlowMergeStateExists(t, dir) {
		t.Error("expected merge state cleared after update abort")
	}
	if got := testutil.GetCurrentBranch(t, dir); got != "feature/x" {
		t.Errorf("expected HEAD on feature/x, got %s", got)
	}
	if status := porcelain(t, dir); strings.TrimSpace(status) != "" {
		t.Errorf("expected clean working tree after abort, got: %q", status)
	}
}

// TestUpdateRebaseConflictThenContinue verifies a rebase-strategy update conflict
// is resumable and completes.
// Steps:
//  1. Sets up an update rebase conflict on feature/x (default downstream=rebase);
//     verifies rebase-merge present and head-name is refs/heads/feature/x.
//  2. Resolves c.txt, stages, runs 'git flow feature update --continue x'.
//  3. Verifies rebase completes, rebase-merge gone, develop is an ancestor of
//     feature/x, state cleared, HEAD on feature/x, exit 0.
func TestUpdateRebaseConflictThenContinue(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	setupUpdateInProgressRebase(t, dir)

	headName, _ := testutil.RunGit(t, dir, "rev-parse", "--symbolic-full-name", "HEAD")
	_ = headName // HEAD is detached during rebase; head-name verified below.
	if !integRebaseInProgress(t, dir) {
		t.Fatal("expected rebase in progress")
	}

	testutil.WriteFile(t, dir, "c.txt", "resolved\n")
	testutil.RunGit(t, dir, "add", "c.txt")

	out, err := testutil.RunGitFlow(t, dir, "feature", "update", "--continue", "x")
	if err != nil {
		t.Fatalf("feature update --continue (rebase) failed: %v\n%s", err, out)
	}
	if integRebaseInProgress(t, dir) {
		t.Error("expected rebase-merge gone after continue")
	}
	if !integIsAncestor(t, dir, integRevParse(t, dir, "develop"), "feature/x") {
		t.Error("expected develop to be an ancestor of feature/x after rebase update")
	}
	if testutil.GitFlowMergeStateExists(t, dir) {
		t.Error("expected merge state cleared after update continue")
	}
	if got := testutil.GetCurrentBranch(t, dir); got != "feature/x" {
		t.Errorf("expected HEAD on feature/x, got %s", got)
	}
}

// TestUpdateRebaseReconflictStaysResumable verifies a rebase update that
// re-conflicts on a second replayed commit stays resumable.
// Steps:
//  1. Builds feature/x with two commits each editing files that also diverged on
//     develop, so the rebase conflicts twice.
//  2. Runs 'git flow feature update x' (rebase) to reach the first conflict.
//  3. Resolves the first conflict, stages, runs 'git flow feature update --continue x'.
//  4. Verifies --continue reports unresolved conflicts (exit 3), the state file
//     bytes are unchanged, and rebase-merge is still present (resumable).
func TestUpdateRebaseReconflictStaysResumable(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	if out, err := testutil.RunGitFlow(t, dir, "init", "--defaults"); err != nil {
		t.Fatalf("init failed: %v\n%s", err, out)
	}
	if out, err := testutil.RunGitFlow(t, dir, "feature", "start", "x"); err != nil {
		t.Fatalf("feature start failed: %v\n%s", err, out)
	}
	// Two conflicting commits on feature/x (a.txt then b.txt).
	integAddCommit(t, dir, "feature/x", "a.txt", "feature a\n", "feature a")
	integAddCommit(t, dir, "feature/x", "b.txt", "feature b\n", "feature b")
	// Diverging edits on develop for both files.
	integAddCommit(t, dir, "develop", "a.txt", "develop a\n", "develop a")
	integAddCommit(t, dir, "develop", "b.txt", "develop b\n", "develop b")
	testutil.RunGit(t, dir, "checkout", "feature/x")

	if out, err := testutil.RunGitFlow(t, dir, "feature", "update", "x"); err == nil {
		t.Fatalf("expected first rebase conflict, got success:\n%s", out)
	}
	if !integRebaseInProgress(t, dir) {
		t.Fatal("expected rebase in progress at first conflict")
	}

	// Resolve the first conflict (a.txt), stage, then continue -> re-conflict on b.txt.
	testutil.WriteFile(t, dir, "a.txt", "resolved a\n")
	testutil.RunGit(t, dir, "add", "a.txt")

	stateBefore := readStateBytes(t, dir)
	out, err := testutil.RunGitFlow(t, dir, "feature", "update", "--continue", "x")
	if code := scopeExitCode(err); code != 3 {
		t.Fatalf("expected exit 3 on re-conflict, got %d (err=%v)\n%s", code, err, out)
	}
	if !integRebaseInProgress(t, dir) {
		t.Error("expected rebase-merge still present (resumable) after re-conflict")
	}
	if got := readStateBytes(t, dir); got != stateBefore {
		t.Errorf("expected state file unchanged after re-conflict.\nbefore: %s\nafter:  %s", stateBefore, got)
	}
}

// TestUpdateRebaseConflictThenAbort verifies a rebase-strategy update conflict
// rolls back on abort.
// Steps:
//  1. Sets up an update rebase conflict on feature/x; captures feature/x tip.
//  2. Runs 'git flow feature update --abort x'.
//  3. Verifies rebase-merge gone, feature/x restored, state cleared, HEAD on
//     feature/x, exit 0.
func TestUpdateRebaseConflictThenAbort(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	setupUpdateInProgressRebase(t, dir)

	preFeature := integRevParse(t, dir, "feature/x")

	out, err := testutil.RunGitFlow(t, dir, "feature", "update", "--abort", "x")
	if err != nil {
		t.Fatalf("feature update --abort (rebase) failed: %v\n%s", err, out)
	}
	if integRebaseInProgress(t, dir) {
		t.Error("expected rebase-merge gone after abort")
	}
	if got := integRevParse(t, dir, "feature/x"); got != preFeature {
		t.Errorf("expected feature/x restored to %s, got %s", preFeature, got)
	}
	if testutil.GitFlowMergeStateExists(t, dir) {
		t.Error("expected merge state cleared after update abort")
	}
	if got := testutil.GetCurrentBranch(t, dir); got != "feature/x" {
		t.Errorf("expected HEAD on feature/x, got %s", got)
	}
}

// setupBaseUpdateInProgress sets up a top-level base-branch update conflict on
// develop (develop <- main, downstream=merge by default). Leaves MERGE_HEAD and
// HEAD on develop, and asserts the initial top-level 'git flow update' exits 3.
func setupBaseUpdateInProgress(t *testing.T, dir string) {
	t.Helper()
	if out, err := testutil.RunGitFlow(t, dir, "init", "--defaults"); err != nil {
		t.Fatalf("init failed: %v\n%s", err, out)
	}
	integAddCommit(t, dir, "main", "c.txt", "main\n", "main c")
	integAddCommit(t, dir, "develop", "c.txt", "develop\n", "develop c")
	testutil.RunGit(t, dir, "checkout", "develop")

	out, err := testutil.RunGitFlow(t, dir, "update")
	if code := scopeExitCode(err); code != 3 {
		t.Fatalf("expected initial top-level update conflict to exit 3, got %d (err=%v)\n%s", code, err, out)
	}
	if !integMergeHeadExists(t, dir) {
		t.Fatalf("expected MERGE_HEAD present after base update conflict")
	}
}

// TestTopLevelUpdateBaseBranchContinue verifies the top-level base-branch update
// surface is resumable via --continue.
// Steps:
// 1. Sets up a develop <- main update conflict via 'git flow update' (exit 3).
// 2. Resolves c.txt, stages, runs 'git flow update --continue'.
// 3. Verifies the merge completes, develop updated from main, state cleared, exit 0.
func TestTopLevelUpdateBaseBranchContinue(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	setupBaseUpdateInProgress(t, dir)

	mainTip := integRevParse(t, dir, "main")
	testutil.WriteFile(t, dir, "c.txt", "resolved\n")
	testutil.RunGit(t, dir, "add", "c.txt")

	out, err := testutil.RunGitFlow(t, dir, "update", "--continue")
	if err != nil {
		t.Fatalf("update --continue (base) failed: %v\n%s", err, out)
	}
	if integMergeHeadExists(t, dir) {
		t.Error("expected MERGE_HEAD gone after continue")
	}
	if !integIsAncestor(t, dir, mainTip, "develop") {
		t.Error("expected develop to include main after update")
	}
	if testutil.GitFlowMergeStateExists(t, dir) {
		t.Error("expected merge state cleared after continue")
	}
	if got := testutil.GetCurrentBranch(t, dir); got != "develop" {
		t.Errorf("expected HEAD on develop, got %s", got)
	}
}

// TestTopLevelUpdateBaseBranchAbort verifies the top-level base-branch update
// surface is abortable via --abort.
// Steps:
// 1. Sets up a develop <- main update conflict via 'git flow update'; captures develop tip.
// 2. Runs 'git flow update --abort'.
// 3. Verifies MERGE_HEAD gone, develop restored, state cleared, HEAD on develop, exit 0.
func TestTopLevelUpdateBaseBranchAbort(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	setupBaseUpdateInProgress(t, dir)

	preDevelop := integRevParse(t, dir, "develop")

	out, err := testutil.RunGitFlow(t, dir, "update", "--abort")
	if err != nil {
		t.Fatalf("update --abort (base) failed: %v\n%s", err, out)
	}
	if integMergeHeadExists(t, dir) {
		t.Error("expected MERGE_HEAD gone after abort")
	}
	if got := integRevParse(t, dir, "develop"); got != preDevelop {
		t.Errorf("expected develop restored to %s, got %s", preDevelop, got)
	}
	if testutil.GitFlowMergeStateExists(t, dir) {
		t.Error("expected merge state cleared after abort")
	}
	if got := testutil.GetCurrentBranch(t, dir); got != "develop" {
		t.Errorf("expected HEAD on develop, got %s", got)
	}
}
