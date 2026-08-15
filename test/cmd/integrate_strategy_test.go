package cmd_test

import (
	"strings"
	"testing"

	"github.com/gittower/git-flow-next/test/testutil"
)

// TestIntegrateSquashOverride verifies --squash lands multiple source commits as
// a single squashed commit on the parent.
//
// Steps:
//  1. init --defaults; add commits C1 and C2 to develop (not on main).
//  2. Run: git flow integrate develop --squash.
//  3. Assert main gains exactly one new commit (not C1/C2 by hash) carrying both
//     files, with no merge commit.
func TestIntegrateSquashOverride(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	if out, err := testutil.RunGitFlow(t, dir, "init", "--defaults"); err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, out)
	}

	c1 := integAddCommit(t, dir, "develop", "a.txt", "1", "Add a.txt on develop")
	c2 := integAddCommit(t, dir, "develop", "b.txt", "2", "Add b.txt on develop")
	preMerges := integMergeCount(t, dir, "main")

	out, err := testutil.RunGitFlow(t, dir, "integrate", "develop", "--squash")
	if err != nil {
		t.Fatalf("integrate develop --squash failed: %v\nOutput: %s", err, out)
	}

	mainTip := integRevParse(t, dir, "main")
	if mainTip == c1 || mainTip == c2 {
		t.Errorf("Expected a new squashed commit distinct from C1/C2, got %s", mainTip)
	}
	if !testutil.FileExists(t, dir, "a.txt") || !testutil.FileExists(t, dir, "b.txt") {
		t.Error("Expected both a.txt and b.txt to be present on main after squash")
	}
	if got := integMergeCount(t, dir, "main"); got != preMerges {
		t.Errorf("Expected no merge commit from squash, merge count changed from %d to %d", preMerges, got)
	}
}

// TestIntegrateRebaseOverrideRewritesSource verifies --rebase replays the source
// branch onto the parent, rewriting its history.
//
// Steps:
//  1. init --defaults; add X to main and C to develop (diverged).
//  2. Run: git flow integrate develop --rebase.
//  3. Assert develop's C sits on top of X (X ancestor of develop, C hash changed),
//     main advanced to include the rebased C, and no merge commit was created.
func TestIntegrateRebaseOverrideRewritesSource(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	if out, err := testutil.RunGitFlow(t, dir, "init", "--defaults"); err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, out)
	}

	xCommit := integAddCommit(t, dir, "main", "x.txt", "X", "Add X on main")
	originalC := integAddCommit(t, dir, "develop", "c.txt", "C", "Add C on develop")
	preMerges := integMergeCount(t, dir, "main")

	out, err := testutil.RunGitFlow(t, dir, "integrate", "develop", "--rebase")
	if err != nil {
		t.Fatalf("integrate develop --rebase failed: %v\nOutput: %s", err, out)
	}

	if !integIsAncestor(t, dir, xCommit, "develop") {
		t.Errorf("Expected X (%s) to be an ancestor of the rewritten develop", xCommit)
	}
	if integRevParse(t, dir, "develop") == originalC {
		t.Error("Expected develop's C commit to be rewritten (new hash) by rebase")
	}
	if !integIsAncestor(t, dir, integRevParse(t, dir, "develop"), "main") {
		t.Error("Expected main to include the rebased develop tip")
	}
	if got := integMergeCount(t, dir, "main"); got != preMerges {
		t.Errorf("Expected no merge commit from rebase, merge count changed from %d to %d", preMerges, got)
	}
}

// TestIntegrateConfiguredUpstreamStrategy verifies a Layer-1 branch-type
// upstream strategy is honored without a CLI flag.
//
// Steps:
//  1. init --defaults; set gitflow.branch.develop.upstreamStrategy squash.
//  2. Add commits C1, C2 to develop.
//  3. Run: git flow integrate develop (no strategy flag).
//  4. Assert a single squashed commit on main.
func TestIntegrateConfiguredUpstreamStrategy(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	if out, err := testutil.RunGitFlow(t, dir, "init", "--defaults"); err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, out)
	}
	if _, err := testutil.RunGit(t, dir, "config", "gitflow.branch.develop.upstreamStrategy", "squash"); err != nil {
		t.Fatalf("Failed to set upstreamStrategy: %v", err)
	}

	c1 := integAddCommit(t, dir, "develop", "a.txt", "1", "Add a.txt on develop")
	c2 := integAddCommit(t, dir, "develop", "b.txt", "2", "Add b.txt on develop")
	preMerges := integMergeCount(t, dir, "main")

	out, err := testutil.RunGitFlow(t, dir, "integrate", "develop")
	if err != nil {
		t.Fatalf("integrate develop failed: %v\nOutput: %s", err, out)
	}

	mainTip := integRevParse(t, dir, "main")
	if mainTip == c1 || mainTip == c2 {
		t.Errorf("Expected a squashed commit (Layer-1 config), got %s", mainTip)
	}
	if !testutil.FileExists(t, dir, "a.txt") || !testutil.FileExists(t, dir, "b.txt") {
		t.Error("Expected both files present on main after squash")
	}
	if got := integMergeCount(t, dir, "main"); got != preMerges {
		t.Errorf("Expected no merge commit, merge count changed from %d to %d", preMerges, got)
	}
}

// TestIntegrateLayer2OverridesLayer1Strategy verifies a Layer-2
// gitflow.<branch>.integrate.squash overrides the Layer-1 merge default.
//
// Steps:
//  1. init --defaults (Layer-1 develop.upstreamStrategy = merge).
//  2. Set gitflow.develop.integrate.squash true.
//  3. Add commits C1, C2 to develop; run integrate develop (no flag).
//  4. Assert a single squashed commit on main (Layer-2 beats Layer-1).
func TestIntegrateLayer2OverridesLayer1Strategy(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	if out, err := testutil.RunGitFlow(t, dir, "init", "--defaults"); err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, out)
	}
	if _, err := testutil.RunGit(t, dir, "config", "gitflow.develop.integrate.squash", "true"); err != nil {
		t.Fatalf("Failed to set integrate.squash config: %v", err)
	}

	c1 := integAddCommit(t, dir, "develop", "a.txt", "1", "Add a.txt on develop")
	c2 := integAddCommit(t, dir, "develop", "b.txt", "2", "Add b.txt on develop")
	preMerges := integMergeCount(t, dir, "main")

	out, err := testutil.RunGitFlow(t, dir, "integrate", "develop")
	if err != nil {
		t.Fatalf("integrate develop failed: %v\nOutput: %s", err, out)
	}

	mainTip := integRevParse(t, dir, "main")
	if mainTip == c1 || mainTip == c2 {
		t.Errorf("Expected squashed commit (Layer-2 over Layer-1), got %s", mainTip)
	}
	if got := integMergeCount(t, dir, "main"); got != preMerges {
		t.Errorf("Expected no merge commit, merge count changed from %d to %d", preMerges, got)
	}
}

// TestIntegrateLayer2SquashOverriddenByFlag verifies a CLI --no-squash beats a
// configured integrate.squash=true default.
//
// Steps:
//  1. init --defaults; set gitflow.develop.integrate.squash true.
//  2. Add commits C1, C2 to develop.
//  3. Run: git flow integrate develop --no-squash.
//  4. Assert a normal merge/fast-forward: both C1 and C2 reachable on main by
//     their original hashes (not squashed).
func TestIntegrateLayer2SquashOverriddenByFlag(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	if out, err := testutil.RunGitFlow(t, dir, "init", "--defaults"); err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, out)
	}
	if _, err := testutil.RunGit(t, dir, "config", "gitflow.develop.integrate.squash", "true"); err != nil {
		t.Fatalf("Failed to set integrate.squash config: %v", err)
	}

	c1 := integAddCommit(t, dir, "develop", "a.txt", "1", "Add a.txt on develop")
	c2 := integAddCommit(t, dir, "develop", "b.txt", "2", "Add b.txt on develop")

	out, err := testutil.RunGitFlow(t, dir, "integrate", "develop", "--no-squash")
	if err != nil {
		t.Fatalf("integrate develop --no-squash failed: %v\nOutput: %s", err, out)
	}

	if !integIsAncestor(t, dir, c1, "main") {
		t.Errorf("Expected C1 (%s) reachable on main by original hash", c1)
	}
	if !integIsAncestor(t, dir, c2, "main") {
		t.Errorf("Expected C2 (%s) reachable on main by original hash", c2)
	}
}

// =============================================================================
// Regression guards for issue #210: finish gains --ff-only, integrate does not.
// =============================================================================

// TestIntegrateNoFFFlagStillCreatesMergeCommit verifies integrate --no-ff is unchanged
// by the finish-side tri-state refactor.
//
// Steps:
//  1. init --defaults; commit develop.txt on develop only (main can fast-forward).
//  2. Run: git flow integrate develop --no-ff.
//  3. Assert main's tip is a merge commit despite a fast-forward being possible.
func TestIntegrateNoFFFlagStillCreatesMergeCommit(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	if out, err := testutil.RunGitFlow(t, dir, "init", "--defaults"); err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, out)
	}
	integAddCommit(t, dir, "develop", "develop.txt", "D", "Add develop.txt on develop")

	out, err := testutil.RunGitFlow(t, dir, "integrate", "develop", "--no-ff")
	if err != nil {
		t.Fatalf("integrate develop --no-ff failed: %v\nOutput: %s", err, out)
	}
	if got := commitParentCount(t, dir, "main"); got != 2 {
		t.Errorf("Expected main's tip to be a merge commit (2 parents), got %d", got)
	}
}

// TestIntegrateFFFlagStillFastForwards verifies integrate --ff is unchanged by the
// finish-side tri-state refactor.
//
// Steps:
//  1. init --defaults; commit develop.txt on develop only and capture its tip.
//  2. Run: git flow integrate develop --ff.
//  3. Assert main equals the captured develop tip with a single-parent tip commit.
func TestIntegrateFFFlagStillFastForwards(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	if out, err := testutil.RunGitFlow(t, dir, "init", "--defaults"); err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, out)
	}
	developTip := integAddCommit(t, dir, "develop", "develop.txt", "D", "Add develop.txt on develop")

	out, err := testutil.RunGitFlow(t, dir, "integrate", "develop", "--ff")
	if err != nil {
		t.Fatalf("integrate develop --ff failed: %v\nOutput: %s", err, out)
	}
	if got := integRevParse(t, dir, "main"); got != developTip {
		t.Errorf("Expected main to equal the develop tip %s, got %s", developTip, got)
	}
	if got := commitParentCount(t, dir, "main"); got != 1 {
		t.Errorf("Expected main's tip to have exactly one parent (fast-forward), got %d", got)
	}
}

// TestIntegrateRejectsFFOnlyFlag verifies integrate never registers --ff-only, so cobra
// rejects it as an unknown flag.
//
// Steps:
//  1. init --defaults.
//  2. Run: git flow integrate develop --ff-only.
//  3. Assert the command fails and the output reports an unknown flag. The exit code is
//     cobra's own (1 via main.go), not a git-flow usage code, so it is not asserted.
func TestIntegrateRejectsFFOnlyFlag(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	if out, err := testutil.RunGitFlow(t, dir, "init", "--defaults"); err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, out)
	}

	out, err := testutil.RunGitFlow(t, dir, "integrate", "develop", "--ff-only")
	if err == nil {
		t.Fatalf("Expected integrate --ff-only to be rejected. Output: %s", out)
	}
	if !strings.Contains(out, "unknown flag: --ff-only") {
		t.Errorf("Expected an 'unknown flag: --ff-only' error. Output: %s", out)
	}
}

// TestIntegrateFFOnlyConfigHasNoEffect verifies the ff-only Layer-2 key is inert outside
// the finish namespace.
//
// Steps:
//  1. init --defaults; set gitflow.develop.integrate.ff-only true.
//  2. Commit develop.txt on develop and main.txt on main (diverged).
//  3. Run: git flow integrate develop.
//  4. Assert it succeeds and main's tip is a merge commit — no gate fired.
func TestIntegrateFFOnlyConfigHasNoEffect(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	if out, err := testutil.RunGitFlow(t, dir, "init", "--defaults"); err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, out)
	}
	if _, err := testutil.RunGit(t, dir, "config", "gitflow.develop.integrate.ff-only", "true"); err != nil {
		t.Fatalf("Failed to set integrate.ff-only config: %v", err)
	}
	integAddCommit(t, dir, "develop", "develop.txt", "D", "Add develop.txt on develop")
	integAddCommit(t, dir, "main", "main.txt", "M", "Add main.txt on main")

	out, err := testutil.RunGitFlow(t, dir, "integrate", "develop")
	if err != nil {
		t.Fatalf("integrate develop failed: %v\nOutput: %s", err, out)
	}
	if got := commitParentCount(t, dir, "main"); got != 2 {
		t.Errorf("Expected main's tip to be a merge commit (2 parents), got %d", got)
	}
}
