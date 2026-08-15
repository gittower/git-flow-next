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

// TestIntegrateChildBranchReportOrderIsDeterministic verifies that integrate
// reports the auto-update child base branches in the same stable, sorted order
// it updates them in, rather than in Go's randomized map iteration order.
//
// Steps:
//  1. init --defaults; configure alpha and zulu as auto-update base children of
//     main, created in the order zulu, alpha so a sorted result cannot be
//     mistaken for insertion or configuration order. develop is already one.
//  2. Give develop a commit not on main so the integration has work to do.
//  3. Run: git flow integrate develop.
//  4. Assert the "Found child base branch" order is alpha, develop, zulu.
//  5. Assert the "Updating child base branch" order is the same, so a fix that
//     sorted only for display would not satisfy the test.
//  6. Repeat across five independent repositories, since with three children an
//     unsorted collection lands on sorted order by chance about one run in six.
func TestIntegrateChildBranchReportOrderIsDeterministic(t *testing.T) {
	t.Parallel()

	want := []string{"alpha", "develop", "zulu"}

	const iterations = 5
	for i := 0; i < iterations; i++ {
		// The iteration body is a closure so each repository is cleaned up as
		// its iteration ends rather than all five at the end of the test.
		func() {
			dir := testutil.SetupTestRepo(t)
			defer testutil.CleanupTestRepo(t, dir)

			if out, err := testutil.RunGitFlow(t, dir, "init", "--defaults"); err != nil {
				t.Fatalf("iteration %d: failed to initialize git-flow: %v\nOutput: %s", i, err, out)
			}

			// A silently dropped config key would change the configured child
			// set the assertions below are written against.
			setConfig := func(key string, value string) {
				if _, err := testutil.RunGit(t, dir, "config", key, value); err != nil {
					t.Fatalf("iteration %d: failed to set %s: %v", i, key, err)
				}
			}

			setConfig("gitflow.branch.develop.autoUpdate", "true")
			setConfig("gitflow.branch.develop.downstreamStrategy", "merge")
			for _, name := range []string{"zulu", "alpha"} {
				if _, err := testutil.RunGit(t, dir, "checkout", "-b", name, "main"); err != nil {
					t.Fatalf("iteration %d: failed to create %s branch: %v", i, name, err)
				}
				setConfig("gitflow.branch."+name+".type", "base")
				setConfig("gitflow.branch."+name+".parent", "main")
				setConfig("gitflow.branch."+name+".autoUpdate", "true")
				setConfig("gitflow.branch."+name+".downstreamStrategy", "merge")
			}

			// Give develop a commit main does not have, so integrating it is
			// not a no-op and every child has something to receive.
			integAddCommit(t, dir, "develop", "c.txt", "C", "Add C on develop")

			out, err := testutil.RunGitFlow(t, dir, "integrate", "develop")
			if err != nil {
				t.Fatalf("iteration %d: integrate develop failed: %v\nOutput: %s", i, err, out)
			}

			// The reported order and the order the children are actually
			// updated in must both be sorted: a fix that sorted only for
			// display would leave the reporting and the updating disagreeing.
			assertChildOrder(t, i, "collection", childOrder(out, "Found child base branch '"), want, out)
			assertChildOrder(t, i, "update", childOrder(out, "Updating child base branch '"), want, out)
		}()
	}
}
