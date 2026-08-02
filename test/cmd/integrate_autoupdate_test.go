package cmd_test

import (
	"testing"

	"github.com/gittower/git-flow-next/test/testutil"
)

// TestIntegrateDivergedParentCreatesMergeAndUpdatesChild verifies a diverged
// parent yields a merge commit and the auto-update child receives the changes.
//
// Steps:
//  1. init --defaults; add X to main and C to develop (diverged).
//  2. Run: git flow integrate develop.
//  3. Assert main has a merge commit containing both C and X, develop is
//     auto-updated to contain X, and HEAD is on main.
func TestIntegrateDivergedParentCreatesMergeAndUpdatesChild(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	if out, err := testutil.RunGitFlow(t, dir, "init", "--defaults"); err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, out)
	}

	xCommit := integAddCommit(t, dir, "main", "x.txt", "X", "Add X on main")
	cCommit := integAddCommit(t, dir, "develop", "c.txt", "C", "Add C on develop")
	preMerges := integMergeCount(t, dir, "main")

	out, err := testutil.RunGitFlow(t, dir, "integrate", "develop")
	if err != nil {
		t.Fatalf("integrate develop failed: %v\nOutput: %s", err, out)
	}

	if !integIsAncestor(t, dir, cCommit, "main") {
		t.Errorf("Expected C (%s) to be reachable on main", cCommit)
	}
	if !integIsAncestor(t, dir, xCommit, "main") {
		t.Errorf("Expected X (%s) to be reachable on main", xCommit)
	}
	if got := integMergeCount(t, dir, "main"); got != preMerges+1 {
		t.Errorf("Expected exactly one new merge commit on main, count went %d -> %d", preMerges, got)
	}
	if !integIsAncestor(t, dir, xCommit, "develop") {
		t.Errorf("Expected develop to be auto-updated to contain X (%s)", xCommit)
	}
	if got := testutil.GetCurrentBranch(t, dir); got != "main" {
		t.Errorf("Expected HEAD on main, got %s", got)
	}
}

// TestIntegrateMultipleChildrenMixedStrategies verifies distinct child update
// strategies and auto-update filtering across several base branches.
//
// Steps:
//  1. init --defaults; add base branch staging (autoUpdate, rebase) with its own
//     commit S, and base branch legacy (autoUpdate=false). develop keeps merge.
//  2. Give develop commit C not on main; capture staging/legacy/develop tips.
//  3. Run: git flow integrate develop.
//  4. Assert main gains C; staging is rebased (S replayed, hash changed, no merge
//     commit); develop merge is a no-op; legacy is untouched; nothing deleted.
func TestIntegrateMultipleChildrenMixedStrategies(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	if out, err := testutil.RunGitFlow(t, dir, "init", "--defaults"); err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, out)
	}

	// staging: base child of main, auto-update via rebase, with its own commit S.
	if _, err := testutil.RunGit(t, dir, "checkout", "-b", "staging", "main"); err != nil {
		t.Fatalf("Failed to create staging: %v", err)
	}
	testutil.RunGit(t, dir, "config", "gitflow.branch.staging.type", "base")
	testutil.RunGit(t, dir, "config", "gitflow.branch.staging.parent", "main")
	testutil.RunGit(t, dir, "config", "gitflow.branch.staging.autoUpdate", "true")
	testutil.RunGit(t, dir, "config", "gitflow.branch.staging.downstreamStrategy", "rebase")
	originalStaging := integAddCommit(t, dir, "staging", "staging.txt", "S", "Add S on staging")

	// legacy: base child of main with auto-update disabled.
	if _, err := testutil.RunGit(t, dir, "checkout", "-b", "legacy", "main"); err != nil {
		t.Fatalf("Failed to create legacy: %v", err)
	}
	testutil.RunGit(t, dir, "config", "gitflow.branch.legacy.type", "base")
	testutil.RunGit(t, dir, "config", "gitflow.branch.legacy.parent", "main")
	testutil.RunGit(t, dir, "config", "gitflow.branch.legacy.autoUpdate", "false")
	originalLegacy := integRevParse(t, dir, "legacy")

	// develop keeps its merge downstream strategy and auto-update.
	testutil.RunGit(t, dir, "config", "gitflow.branch.develop.autoUpdate", "true")
	testutil.RunGit(t, dir, "config", "gitflow.branch.develop.downstreamStrategy", "merge")
	cCommit := integAddCommit(t, dir, "develop", "develop.txt", "C", "Add C on develop")
	originalDevelop := integRevParse(t, dir, "develop")

	preStagingMerges := integMergeCount(t, dir, "staging")

	out, err := testutil.RunGitFlow(t, dir, "integrate", "develop")
	if err != nil {
		t.Fatalf("integrate develop failed: %v\nOutput: %s", err, out)
	}

	// main advanced to include C.
	if !integIsAncestor(t, dir, cCommit, "main") {
		t.Errorf("Expected C (%s) reachable on main", cCommit)
	}

	// staging rebased: main's tip is now an ancestor of staging, S was rewritten,
	// and no merge commit was introduced.
	mainTip := integRevParse(t, dir, "main")
	if !integIsAncestor(t, dir, mainTip, "staging") {
		t.Error("Expected staging to be rebased onto main's new tip")
	}
	if integRevParse(t, dir, "staging") == originalStaging {
		t.Error("Expected staging's S commit to be rewritten by rebase")
	}
	if got := integMergeCount(t, dir, "staging"); got != preStagingMerges {
		t.Errorf("Expected no merge commit on staging (rebase), count went %d -> %d", preStagingMerges, got)
	}

	// develop merge is a no-op (already == main after fast-forward).
	if got := integRevParse(t, dir, "develop"); got != originalDevelop {
		t.Errorf("Expected develop unchanged (no-op merge), was %s got %s", originalDevelop, got)
	}

	// legacy is not moved.
	if got := integRevParse(t, dir, "legacy"); got != originalLegacy {
		t.Errorf("Expected legacy untouched (%s), got %s", originalLegacy, got)
	}

	// Nothing deleted.
	for _, b := range []string{"main", "develop", "staging", "legacy"} {
		if !testutil.BranchExists(t, dir, b) {
			t.Errorf("Expected branch %s to still exist", b)
		}
	}
}
