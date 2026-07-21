package cmd_test

import (
	"strings"
	"testing"

	"github.com/gittower/git-flow-next/test/testutil"
)

// integSetupConflict creates a diverged modify/modify conflict on the given file
// between main and develop. It seeds a shared base version of the file, then
// edits it differently on each branch. Returns (xCommit on main, cCommit on
// develop) and leaves HEAD on develop.
func integSetupConflict(t *testing.T, dir, file string) (xCommit, cCommit string) {
	t.Helper()
	integAddCommit(t, dir, "main", file, "base\n", "Add "+file+" base on main")
	if _, err := testutil.RunGit(t, dir, "checkout", "develop"); err != nil {
		t.Fatalf("Failed to checkout develop: %v", err)
	}
	if _, err := testutil.RunGit(t, dir, "merge", "main"); err != nil {
		t.Fatalf("Failed to fast-forward develop from main: %v", err)
	}
	xCommit = integAddCommit(t, dir, "main", file, "main version\n", "Edit "+file+" on main")
	cCommit = integAddCommit(t, dir, "develop", file, "develop version\n", "Edit "+file+" on develop")
	return xCommit, cCommit
}

// TestIntegrateMergeConflictThenContinue verifies a merge conflict is resumable.
//
// Steps:
//  1. init --defaults; create a diverged conflict on conflict.txt.
//  2. Run: git flow integrate develop --tag v2.0.0 (conflicts).
//  3. Assert Git is in a real merge-conflict state and integrate state is saved.
//  4. Resolve + stage, then git flow integrate --continue.
//  5. Assert the merge completes, tag is created, children updated, state cleared.
func TestIntegrateMergeConflictThenContinue(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	if out, err := testutil.RunGitFlow(t, dir, "init", "--defaults"); err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, out)
	}
	integSetupConflict(t, dir, "conflict.txt")

	out, err := testutil.RunGitFlow(t, dir, "integrate", "develop", "--tag", "v2.0.0")
	if err == nil {
		t.Fatalf("Expected merge conflict, got success.\nOutput: %s", out)
	}
	if !strings.Contains(out, "conflict") {
		t.Errorf("Expected conflict message, got: %s", out)
	}
	if !integMergeHeadExists(t, dir) {
		t.Error("Expected .git/MERGE_HEAD to exist during merge conflict")
	}
	if status, _ := testutil.RunGit(t, dir, "status", "--porcelain"); !strings.Contains(status, "UU") {
		t.Errorf("Expected a UU conflict entry, got: %s", status)
	}
	state, err := testutil.LoadMergeState(t, dir)
	if err != nil || state == nil {
		t.Fatalf("Expected saved merge state, err=%v", err)
	}
	if state.Action != "integrate" {
		t.Errorf("Expected Action integrate, got %s", state.Action)
	}
	if state.CurrentStep != "merge" {
		t.Errorf("Expected CurrentStep merge, got %s", state.CurrentStep)
	}
	if state.MergeStrategy != "merge" {
		t.Errorf("Expected MergeStrategy merge, got %s", state.MergeStrategy)
	}

	testutil.WriteFile(t, dir, "conflict.txt", "resolved\n")
	testutil.RunGit(t, dir, "add", "conflict.txt")

	out, err = testutil.RunGitFlow(t, dir, "integrate", "--continue")
	if err != nil {
		t.Fatalf("integrate --continue failed: %v\nOutput: %s", err, out)
	}
	if integMergeHeadExists(t, dir) {
		t.Error("Expected .git/MERGE_HEAD to be gone after continue")
	}
	if !integTagExists(t, dir, "v2.0.0") || !integTagIsAnnotated(t, dir, "v2.0.0") {
		t.Error("Expected annotated tag v2.0.0 after continue")
	}
	if integRevParse(t, dir, "v2.0.0^{commit}") != integRevParse(t, dir, "main") {
		t.Error("Expected v2.0.0 on main's tip")
	}
	if !integIsAncestor(t, dir, integRevParse(t, dir, "main"), "develop") {
		t.Error("Expected develop to be auto-updated from main")
	}
	if integStateExists(t, dir) {
		t.Error("Expected merge state to be cleared after continue")
	}
	if got := testutil.GetCurrentBranch(t, dir); got != "main" {
		t.Errorf("Expected HEAD on main, got %s", got)
	}
}

// TestIntegrateMergeConflictThenAbort verifies a merge conflict fully rolls back
// on abort.
//
// Steps:
//  1. init --defaults; create a diverged conflict; capture pre-integrate tips.
//  2. Run: git flow integrate develop --tag v2.0.0 (conflicts).
//  3. Run: git flow integrate --abort.
//  4. Assert MERGE_HEAD gone, main/develop restored, no tag, state cleared,
//     HEAD on develop, working tree clean.
func TestIntegrateMergeConflictThenAbort(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	if out, err := testutil.RunGitFlow(t, dir, "init", "--defaults"); err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, out)
	}
	integSetupConflict(t, dir, "conflict.txt")
	preMain := integRevParse(t, dir, "main")
	preDevelop := integRevParse(t, dir, "develop")

	if out, err := testutil.RunGitFlow(t, dir, "integrate", "develop", "--tag", "v2.0.0"); err == nil {
		t.Fatalf("Expected merge conflict, got success.\nOutput: %s", out)
	}

	out, err := testutil.RunGitFlow(t, dir, "integrate", "--abort")
	if err != nil {
		t.Fatalf("integrate --abort failed: %v\nOutput: %s", err, out)
	}
	if integMergeHeadExists(t, dir) {
		t.Error("Expected .git/MERGE_HEAD to be gone after abort")
	}
	if got := integRevParse(t, dir, "main"); got != preMain {
		t.Errorf("Expected main restored to %s, got %s", preMain, got)
	}
	if got := integRevParse(t, dir, "develop"); got != preDevelop {
		t.Errorf("Expected develop unchanged (%s), got %s", preDevelop, got)
	}
	if integTagExists(t, dir, "v2.0.0") {
		t.Error("Expected no tag v2.0.0 after abort")
	}
	if integStateExists(t, dir) {
		t.Error("Expected merge state cleared after abort")
	}
	if got := testutil.GetCurrentBranch(t, dir); got != "develop" {
		t.Errorf("Expected HEAD on develop after abort, got %s", got)
	}
	if status, _ := testutil.RunGit(t, dir, "status", "--porcelain"); strings.TrimSpace(status) != "" {
		t.Errorf("Expected clean working tree after abort, got: %s", status)
	}
}

// TestIntegrateRebaseConflictThenContinue verifies a rebase conflict is resumable.
//
// Steps:
//  1. init --defaults; create a diverged conflict on f.txt.
//  2. Run: git flow integrate develop --rebase (conflicts during replay).
//  3. Assert rebase-merge state present and integrate state saved (rebase).
//  4. Resolve + stage, then git flow integrate --continue.
//  5. Assert rebase-merge gone, develop rewritten onto X, main advanced, no merge
//     commit, state cleared, HEAD on main.
func TestIntegrateRebaseConflictThenContinue(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	if out, err := testutil.RunGitFlow(t, dir, "init", "--defaults"); err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, out)
	}
	xCommit, originalC := integSetupConflict(t, dir, "f.txt")
	preMerges := integMergeCount(t, dir, "main")

	out, err := testutil.RunGitFlow(t, dir, "integrate", "develop", "--rebase")
	if err == nil {
		t.Fatalf("Expected rebase conflict, got success.\nOutput: %s", out)
	}
	if !integRebaseInProgress(t, dir) {
		t.Error("Expected .git/rebase-merge to be present during rebase conflict")
	}
	state, err := testutil.LoadMergeState(t, dir)
	if err != nil || state == nil {
		t.Fatalf("Expected saved merge state, err=%v", err)
	}
	if state.Action != "integrate" {
		t.Errorf("Expected Action integrate, got %s", state.Action)
	}
	if state.MergeStrategy != "rebase" {
		t.Errorf("Expected MergeStrategy rebase, got %s", state.MergeStrategy)
	}

	testutil.WriteFile(t, dir, "f.txt", "resolved\n")
	testutil.RunGit(t, dir, "add", "f.txt")

	out, err = testutil.RunGitFlow(t, dir, "integrate", "--continue")
	if err != nil {
		t.Fatalf("integrate --continue failed: %v\nOutput: %s", err, out)
	}
	if integRebaseInProgress(t, dir) {
		t.Error("Expected .git/rebase-merge to be gone after continue")
	}
	if content := testutil.ReadFile(t, dir, "f.txt"); !strings.Contains(content, "resolved") {
		t.Errorf("Expected resolved content present, got: %s", content)
	}
	if !integIsAncestor(t, dir, xCommit, "develop") {
		t.Error("Expected develop rewritten on top of X")
	}
	if integRevParse(t, dir, "develop") == originalC {
		t.Error("Expected develop's C to be rewritten by rebase")
	}
	if !integIsAncestor(t, dir, integRevParse(t, dir, "develop"), "main") {
		t.Error("Expected main to include the rebased develop tip")
	}
	if got := integMergeCount(t, dir, "main"); got != preMerges {
		t.Errorf("Expected no merge commit, count went %d -> %d", preMerges, got)
	}
	if integStateExists(t, dir) {
		t.Error("Expected merge state cleared after continue")
	}
	if got := testutil.GetCurrentBranch(t, dir); got != "main" {
		t.Errorf("Expected HEAD on main, got %s", got)
	}
}

// TestIntegrateRebaseConflictThenAbort verifies a rebase conflict rolls back on
// abort.
//
// Steps:
//  1. init --defaults; create a diverged conflict; capture pre-integrate tips.
//  2. Run: git flow integrate develop --rebase (conflicts).
//  3. Run: git flow integrate --abort.
//  4. Assert rebase aborted, main/develop restored, no tag, state cleared,
//     HEAD on develop.
func TestIntegrateRebaseConflictThenAbort(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	if out, err := testutil.RunGitFlow(t, dir, "init", "--defaults"); err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, out)
	}
	integSetupConflict(t, dir, "f.txt")
	preMain := integRevParse(t, dir, "main")
	preDevelop := integRevParse(t, dir, "develop")

	if out, err := testutil.RunGitFlow(t, dir, "integrate", "develop", "--rebase"); err == nil {
		t.Fatalf("Expected rebase conflict, got success.\nOutput: %s", out)
	}

	out, err := testutil.RunGitFlow(t, dir, "integrate", "--abort")
	if err != nil {
		t.Fatalf("integrate --abort failed: %v\nOutput: %s", err, out)
	}
	if integRebaseInProgress(t, dir) {
		t.Error("Expected .git/rebase-merge to be gone after abort")
	}
	if got := integRevParse(t, dir, "main"); got != preMain {
		t.Errorf("Expected main restored to %s, got %s", preMain, got)
	}
	if got := integRevParse(t, dir, "develop"); got != preDevelop {
		t.Errorf("Expected develop restored to %s, got %s", preDevelop, got)
	}
	if integTagExists(t, dir, "v2.0.0") {
		t.Error("Expected no tag after abort")
	}
	if integStateExists(t, dir) {
		t.Error("Expected merge state cleared after abort")
	}
	if got := testutil.GetCurrentBranch(t, dir); got != "develop" {
		t.Errorf("Expected HEAD on develop after abort, got %s", got)
	}
}

// TestIntegrateChildUpdateConflictThenContinue verifies a conflict during child
// auto-update is resumable without redoing the parent merge or tag.
//
// Steps:
//  1. init --defaults; add base child staging (auto-update, merge) with commit S.
//  2. Add commit C to develop creating conflict.txt (clean into main).
//  3. Run: git flow integrate develop --tag v2.0.0. Parent merge + tag succeed;
//     updating staging from main conflicts on conflict.txt.
//  4. Capture main tip and tag object; resolve + stage; git flow integrate
//     --continue.
//  5. Assert main tip and tag object unchanged, no duplicate merge on main,
//     staging carries main's content, state cleared.
func TestIntegrateChildUpdateConflictThenContinue(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	if out, err := testutil.RunGitFlow(t, dir, "init", "--defaults"); err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, out)
	}

	if _, err := testutil.RunGit(t, dir, "checkout", "-b", "staging", "main"); err != nil {
		t.Fatalf("Failed to create staging: %v", err)
	}
	testutil.RunGit(t, dir, "config", "gitflow.branch.staging.type", "base")
	testutil.RunGit(t, dir, "config", "gitflow.branch.staging.parent", "main")
	testutil.RunGit(t, dir, "config", "gitflow.branch.staging.autoUpdate", "true")
	testutil.RunGit(t, dir, "config", "gitflow.branch.staging.downstreamStrategy", "merge")
	integAddCommit(t, dir, "staging", "conflict.txt", "staging content\n", "Add S on staging")

	integAddCommit(t, dir, "develop", "conflict.txt", "develop content\n", "Add C on develop")

	out, err := testutil.RunGitFlow(t, dir, "integrate", "develop", "--tag", "v2.0.0")
	if err == nil {
		t.Fatalf("Expected child update conflict, got success.\nOutput: %s", out)
	}
	if !integMergeHeadExists(t, dir) {
		t.Error("Expected .git/MERGE_HEAD during child merge conflict")
	}
	state, err := testutil.LoadMergeState(t, dir)
	if err != nil || state == nil {
		t.Fatalf("Expected saved merge state, err=%v", err)
	}
	if state.Action != "integrate" {
		t.Errorf("Expected Action integrate, got %s", state.Action)
	}
	if state.CurrentStep != "update_children" {
		t.Errorf("Expected CurrentStep update_children, got %s", state.CurrentStep)
	}

	mainAtConflict := integRevParse(t, dir, "main")
	tagAtConflict := integRevParse(t, dir, "v2.0.0")
	mergesAtConflict := integMergeCount(t, dir, "main")

	testutil.WriteFile(t, dir, "conflict.txt", "resolved\n")
	testutil.RunGit(t, dir, "add", "conflict.txt")

	out, err = testutil.RunGitFlow(t, dir, "integrate", "--continue")
	if err != nil {
		t.Fatalf("integrate --continue failed: %v\nOutput: %s", err, out)
	}
	if got := integRevParse(t, dir, "main"); got != mainAtConflict {
		t.Errorf("Expected main tip unchanged on continue (%s), got %s", mainAtConflict, got)
	}
	if got := integRevParse(t, dir, "v2.0.0"); got != tagAtConflict {
		t.Errorf("Expected tag object unchanged on continue (%s), got %s", tagAtConflict, got)
	}
	if got := integMergeCount(t, dir, "main"); got != mergesAtConflict {
		t.Errorf("Expected no duplicate merge on main, count went %d -> %d", mergesAtConflict, got)
	}
	if !integIsAncestor(t, dir, mainAtConflict, "staging") {
		t.Error("Expected staging to carry main's content after continue")
	}
	if integStateExists(t, dir) {
		t.Error("Expected merge state cleared after continue")
	}
}

// TestIntegrateChildUpdateConflictThenAbort verifies aborting during a child
// update uses the child's strategy and preserves the completed parent merge/tag.
//
// Steps:
//  1. init --defaults (parent integrate strategy = merge). Add base children
//     early (auto-update, merge) and staging (auto-update, rebase).
//  2. Add clean commit C to develop; staging carries a commit that conflicts with
//     main after the integrate (so its rebase replay conflicts).
//  3. Run: git flow integrate develop --tag v2.0.0. Parent merge + tag succeed;
//     staging's rebase update conflicts.
//  4. Capture main tip, tag object, early tip; confirm rebase-merge present.
//  5. Run: git flow integrate --abort.
//  6. Assert merge + tag preserved, early untouched, rebase-merge gone (proving
//     child-strategy-aware abort), state cleared.
func TestIntegrateChildUpdateConflictThenAbort(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	if out, err := testutil.RunGitFlow(t, dir, "init", "--defaults"); err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, out)
	}

	// early: clean auto-update via merge, with its own history so its update is
	// a real, observable change.
	if _, err := testutil.RunGit(t, dir, "checkout", "-b", "early", "main"); err != nil {
		t.Fatalf("Failed to create early: %v", err)
	}
	testutil.RunGit(t, dir, "config", "gitflow.branch.early.type", "base")
	testutil.RunGit(t, dir, "config", "gitflow.branch.early.parent", "main")
	testutil.RunGit(t, dir, "config", "gitflow.branch.early.autoUpdate", "true")
	testutil.RunGit(t, dir, "config", "gitflow.branch.early.downstreamStrategy", "merge")
	integAddCommit(t, dir, "early", "early.txt", "early\n", "Add early.txt on early")

	// staging: auto-update via rebase, carries a commit that conflicts with main.
	if _, err := testutil.RunGit(t, dir, "checkout", "-b", "staging", "main"); err != nil {
		t.Fatalf("Failed to create staging: %v", err)
	}
	testutil.RunGit(t, dir, "config", "gitflow.branch.staging.type", "base")
	testutil.RunGit(t, dir, "config", "gitflow.branch.staging.parent", "main")
	testutil.RunGit(t, dir, "config", "gitflow.branch.staging.autoUpdate", "true")
	testutil.RunGit(t, dir, "config", "gitflow.branch.staging.downstreamStrategy", "rebase")
	integAddCommit(t, dir, "staging", "conflict.txt", "staging content\n", "Add S on staging")

	// develop: clean commit C that creates conflict.txt (integrates into main
	// cleanly, but makes staging's rebase replay conflict).
	integAddCommit(t, dir, "develop", "conflict.txt", "develop content\n", "Add C on develop")

	out, err := testutil.RunGitFlow(t, dir, "integrate", "develop", "--tag", "v2.0.0")
	if err == nil {
		t.Fatalf("Expected staging rebase conflict, got success.\nOutput: %s", out)
	}
	if !integRebaseInProgress(t, dir) {
		t.Error("Expected .git/rebase-merge present at staging rebase conflict")
	}
	state, err := testutil.LoadMergeState(t, dir)
	if err != nil || state == nil {
		t.Fatalf("Expected saved merge state, err=%v", err)
	}
	if state.CurrentStep != "update_children" {
		t.Errorf("Expected CurrentStep update_children, got %s", state.CurrentStep)
	}
	if state.MergeStrategy != "merge" {
		t.Errorf("Expected parent MergeStrategy merge, got %s", state.MergeStrategy)
	}

	mainAtConflict := integRevParse(t, dir, "main")
	tagAtConflict := integRevParse(t, dir, "v2.0.0")
	earlyAtConflict := integRevParse(t, dir, "early")

	out, err = testutil.RunGitFlow(t, dir, "integrate", "--abort")
	if err != nil {
		t.Fatalf("integrate --abort failed: %v\nOutput: %s", err, out)
	}
	// The bug this catches: aborting with state.MergeStrategy (merge) instead of
	// the child's strategy (rebase) would leave a stray rebase-merge behind.
	if integRebaseInProgress(t, dir) {
		t.Error("Expected .git/rebase-merge gone after child-strategy-aware abort")
	}
	if got := integRevParse(t, dir, "main"); got != mainAtConflict {
		t.Errorf("Expected completed parent merge preserved (%s), got %s", mainAtConflict, got)
	}
	if got := integRevParse(t, dir, "v2.0.0"); got != tagAtConflict {
		t.Errorf("Expected tag preserved (%s), got %s", tagAtConflict, got)
	}
	if got := integRevParse(t, dir, "early"); got != earlyAtConflict {
		t.Errorf("Expected early child untouched by abort (%s), got %s", earlyAtConflict, got)
	}
	if integStateExists(t, dir) {
		t.Error("Expected merge state cleared after abort")
	}
}

// TestIntegrateContinueRejectsFinishState verifies integrate --continue refuses
// to resume a foreign finish operation.
//
// Steps:
//  1. init --defaults; start a feature finish that conflicts (Action = finish).
//  2. Run: git flow integrate --continue.
//  3. Assert non-zero exit and the finish state remains (Action still finish).
func TestIntegrateContinueRejectsFinishState(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	if out, err := testutil.RunGitFlow(t, dir, "init", "--defaults"); err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, out)
	}

	// Conflicting feature finish (feature vs develop merge).
	integAddCommit(t, dir, "develop", "conflict.txt", "develop\n", "develop base")
	if out, err := testutil.RunGitFlow(t, dir, "feature", "start", "foo"); err != nil {
		t.Fatalf("Failed to start feature: %v\nOutput: %s", err, out)
	}
	integAddCommit(t, dir, "feature/foo", "conflict.txt", "feature\n", "feature change")
	integAddCommit(t, dir, "develop", "conflict.txt", "develop updated\n", "develop change")
	testutil.RunGit(t, dir, "checkout", "feature/foo")
	if out, err := testutil.RunGitFlow(t, dir, "feature", "finish", "foo"); err == nil {
		t.Fatalf("Expected feature finish to conflict.\nOutput: %s", out)
	}

	out, err := testutil.RunGitFlow(t, dir, "integrate", "--continue")
	if err == nil {
		t.Fatalf("Expected integrate --continue to reject a finish state.\nOutput: %s", out)
	}
	state, err := testutil.LoadMergeState(t, dir)
	if err != nil || state == nil {
		t.Fatalf("Expected finish state to remain, err=%v", err)
	}
	if state.Action != "finish" {
		t.Errorf("Expected finish state left intact, got Action %s", state.Action)
	}
}

// TestIntegrateContinueRejectsUpdateState verifies integrate --continue refuses
// to resume a foreign update operation.
//
// Steps:
//  1. init --defaults; start a feature update that conflicts (Action = update).
//  2. Run: git flow integrate --continue.
//  3. Assert non-zero exit and the update state remains (Action still update).
func TestIntegrateContinueRejectsUpdateState(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	if out, err := testutil.RunGitFlow(t, dir, "init", "--defaults"); err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, out)
	}

	// Conflicting feature update (develop into feature).
	if out, err := testutil.RunGitFlow(t, dir, "feature", "start", "foo"); err != nil {
		t.Fatalf("Failed to start feature: %v\nOutput: %s", err, out)
	}
	integAddCommit(t, dir, "feature/foo", "conflict.txt", "feature\n", "feature change")
	integAddCommit(t, dir, "develop", "conflict.txt", "develop\n", "develop change")
	testutil.RunGit(t, dir, "checkout", "feature/foo")
	if out, err := testutil.RunGitFlow(t, dir, "feature", "update", "foo"); err == nil {
		t.Fatalf("Expected feature update to conflict.\nOutput: %s", out)
	}
	state, err := testutil.LoadMergeState(t, dir)
	if err != nil || state == nil || state.Action != "update" {
		t.Fatalf("Expected update state to exist, got %+v (err=%v)", state, err)
	}

	out, err := testutil.RunGitFlow(t, dir, "integrate", "--continue")
	if err == nil {
		t.Fatalf("Expected integrate --continue to reject an update state.\nOutput: %s", out)
	}
	state, err = testutil.LoadMergeState(t, dir)
	if err != nil || state == nil {
		t.Fatalf("Expected update state to remain, err=%v", err)
	}
	if state.Action != "update" {
		t.Errorf("Expected update state left intact, got Action %s", state.Action)
	}
}

// TestIntegrateAbortRejectsFinishState verifies integrate --abort refuses to
// abort a foreign finish operation.
//
// Steps:
//  1. init --defaults; start a feature finish that conflicts (Action = finish).
//  2. Run: git flow integrate --abort.
//  3. Assert non-zero exit, finish MERGE_HEAD and state remain.
func TestIntegrateAbortRejectsFinishState(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	if out, err := testutil.RunGitFlow(t, dir, "init", "--defaults"); err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, out)
	}

	integAddCommit(t, dir, "develop", "conflict.txt", "develop\n", "develop base")
	if out, err := testutil.RunGitFlow(t, dir, "feature", "start", "foo"); err != nil {
		t.Fatalf("Failed to start feature: %v\nOutput: %s", err, out)
	}
	integAddCommit(t, dir, "feature/foo", "conflict.txt", "feature\n", "feature change")
	integAddCommit(t, dir, "develop", "conflict.txt", "develop updated\n", "develop change")
	testutil.RunGit(t, dir, "checkout", "feature/foo")
	if out, err := testutil.RunGitFlow(t, dir, "feature", "finish", "foo"); err == nil {
		t.Fatalf("Expected feature finish to conflict.\nOutput: %s", out)
	}

	out, err := testutil.RunGitFlow(t, dir, "integrate", "--abort")
	if err == nil {
		t.Fatalf("Expected integrate --abort to reject a finish state.\nOutput: %s", out)
	}
	if !integMergeHeadExists(t, dir) {
		t.Error("Expected finish MERGE_HEAD to remain after rejected abort")
	}
	state, err := testutil.LoadMergeState(t, dir)
	if err != nil || state == nil {
		t.Fatalf("Expected finish state to remain, err=%v", err)
	}
	if state.Action != "finish" {
		t.Errorf("Expected finish state left intact, got Action %s", state.Action)
	}
}

// TestIntegrateAbortNoOp verifies integrate --abort with nothing in progress is
// a forgiving no-op.
//
// Steps:
//  1. init --defaults (nothing in progress).
//  2. Run: git flow integrate --abort.
//  3. Assert exit 0 and nothing mutated.
func TestIntegrateAbortNoOp(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	if out, err := testutil.RunGitFlow(t, dir, "init", "--defaults"); err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, out)
	}
	preMain := integRevParse(t, dir, "main")
	preDevelop := integRevParse(t, dir, "develop")

	out, err := testutil.RunGitFlow(t, dir, "integrate", "--abort")
	if err != nil {
		t.Fatalf("Expected integrate --abort no-op to succeed: %v\nOutput: %s", err, out)
	}
	if got := integRevParse(t, dir, "main"); got != preMain {
		t.Errorf("Expected main unchanged, got %s", got)
	}
	if got := integRevParse(t, dir, "develop"); got != preDevelop {
		t.Errorf("Expected develop unchanged, got %s", got)
	}
}
