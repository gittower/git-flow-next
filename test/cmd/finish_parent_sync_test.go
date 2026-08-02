package cmd_test

import (
	"strings"
	"testing"

	"github.com/gittower/git-flow-next/test/testutil"
)

// This file covers issue #99: before finish merges a topic branch into its parent (merge-target)
// branch, the parent must be in sync with its remote. The parent invariant differs from the topic
// invariant checked in finish_sync_test.go: the parent must not be *behind* or *diverged* from its
// remote (merging onto a stale base), but being *ahead* is acceptable (the normal state right after
// a previous unpushed finish). --force skips the check; the check is gated the same way as the topic
// check (skipped with no remote / no tracking branch / a benign missing remote ref).
//
// The standard fixture parent is develop (git-flow defaults: feature parent = develop), and
// SetupTestRepoWithRemote pushes develop with upstream tracking.

// advanceRemoteDevelop clones the bare remote into a throwaway working copy, adds a commit on
// develop, and pushes it — advancing origin/develop without touching the branch under test. It
// returns the new remote develop SHA.
func advanceRemoteDevelop(t *testing.T, remoteDir, file, content, msg string) string {
	t.Helper()
	second := t.TempDir()
	if _, err := testutil.RunGit(t, second, "clone", remoteDir, "."); err != nil {
		t.Fatalf("Failed to clone remote: %v", err)
	}
	testutil.ConfigureGitIdentity(t, second)
	if _, err := testutil.RunGit(t, second, "checkout", "develop"); err != nil {
		t.Fatalf("Failed to checkout develop in clone: %v", err)
	}
	testutil.WriteFile(t, second, file, content)
	if _, err := testutil.RunGit(t, second, "add", file); err != nil {
		t.Fatalf("Failed to add %s in clone: %v", file, err)
	}
	if _, err := testutil.RunGit(t, second, "commit", "-m", msg); err != nil {
		t.Fatalf("Failed to commit in clone: %v", err)
	}
	if _, err := testutil.RunGit(t, second, "push", "origin", "develop"); err != nil {
		t.Fatalf("Failed to push develop from clone: %v", err)
	}
	sha, err := testutil.RunGit(t, second, "rev-parse", "develop")
	if err != nil {
		t.Fatalf("Failed to get develop SHA from clone: %v", err)
	}
	return strings.TrimSpace(sha)
}

// commitOnLocalDevelop adds a local commit to develop and returns to the given branch. It is used to
// put local develop ahead of (or diverged from) its remote without pushing.
func commitOnLocalDevelop(t *testing.T, dir, returnBranch, file, content, msg string) {
	t.Helper()
	if _, err := testutil.RunGit(t, dir, "checkout", "develop"); err != nil {
		t.Fatalf("Failed to checkout develop: %v", err)
	}
	testutil.WriteFile(t, dir, file, content)
	if _, err := testutil.RunGit(t, dir, "add", file); err != nil {
		t.Fatalf("Failed to add %s: %v", file, err)
	}
	if _, err := testutil.RunGit(t, dir, "commit", "-m", msg); err != nil {
		t.Fatalf("Failed to commit on develop: %v", err)
	}
	if _, err := testutil.RunGit(t, dir, "checkout", returnBranch); err != nil {
		t.Fatalf("Failed to checkout %s: %v", returnBranch, err)
	}
}

// revParse returns the SHA of a ref, failing the test on error.
func revParse(t *testing.T, dir, ref string) string {
	t.Helper()
	sha, err := testutil.RunGit(t, dir, "rev-parse", ref)
	if err != nil {
		t.Fatalf("Failed to rev-parse %s: %v", ref, err)
	}
	return strings.TrimSpace(sha)
}

// assertFeatureMerged asserts finish completed: develop advanced past developBefore and the feature
// commit is now an ancestor of develop, and the local feature branch was deleted.
func assertFeatureMerged(t *testing.T, dir, branch, featureSha, developBefore string) {
	t.Helper()
	developAfter := revParse(t, dir, "develop")
	if developAfter == developBefore {
		t.Errorf("Expected develop to advance after finish; it stayed at %s", developBefore)
	}
	if _, err := testutil.RunGit(t, dir, "merge-base", "--is-ancestor", featureSha, "develop"); err != nil {
		t.Errorf("Expected feature commit %s to be an ancestor of develop after finish", featureSha)
	}
	if testutil.BranchExists(t, dir, branch) {
		t.Errorf("Expected branch %s to be deleted after finish", branch)
	}
}

// Scenario 1: Parent in sync — finish proceeds and merges.
func TestFinishParentInSyncProceeds(t *testing.T) {
	t.Parallel()
	dir, remoteDir := testutil.SetupTestRepoWithRemote(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer testutil.CleanupTestRepo(t, remoteDir)

	if _, err := testutil.RunGitFlow(t, dir, "feature", "start", "in-sync"); err != nil {
		t.Fatalf("Failed to create feature branch: %v", err)
	}
	featureSha := commitFeatureAndPush(t, dir, "feature.txt", "feature content", "Add feature file", "feature/in-sync")
	developBefore := revParse(t, dir, "develop")

	output, err := testutil.RunGitFlow(t, dir, "feature", "finish", "in-sync")
	if err != nil {
		t.Fatalf("Expected finish to succeed when parent is in sync. Output: %s, err: %v", output, err)
	}
	assertFeatureMerged(t, dir, "feature/in-sync", featureSha, developBefore)
}

// Scenario 2: Parent behind its remote — abort; merge does not happen.
func TestFinishParentBehindRemoteAborts(t *testing.T) {
	t.Parallel()
	dir, remoteDir := testutil.SetupTestRepoWithRemote(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer testutil.CleanupTestRepo(t, remoteDir)

	if _, err := testutil.RunGitFlow(t, dir, "feature", "start", "parent-behind"); err != nil {
		t.Fatalf("Failed to create feature branch: %v", err)
	}
	commitFeatureAndPush(t, dir, "feature.txt", "feature content", "Add feature file", "feature/parent-behind")

	// Advance origin/develop so local develop will be behind after the preflight's parent fetch.
	advanceRemoteDevelop(t, remoteDir, "remote1.txt", "r1", "Remote develop commit")

	developBefore := revParse(t, dir, "develop")

	output, err := testutil.RunGitFlow(t, dir, "feature", "finish", "parent-behind")
	if err == nil {
		t.Errorf("Expected finish to fail when parent is behind remote. Output: %s", output)
	}
	if !strings.Contains(output, "develop") {
		t.Errorf("Expected error to name the parent branch 'develop'. Output: %s", output)
	}
	if !strings.Contains(output, "behind") {
		t.Errorf("Expected error to mention 'behind'. Output: %s", output)
	}
	if !strings.Contains(output, "--force") {
		t.Errorf("Expected error to suggest '--force'. Output: %s", output)
	}
	if after := revParse(t, dir, "develop"); after != developBefore {
		t.Errorf("Expected develop unchanged after aborted finish. Before: %s After: %s", developBefore, after)
	}
	if !testutil.BranchExists(t, dir, "feature/parent-behind") {
		t.Error("Expected feature branch to still exist after aborted finish")
	}
}

// Scenario 3: Parent diverged from its remote — abort; merge does not happen.
func TestFinishParentDivergedAborts(t *testing.T) {
	t.Parallel()
	dir, remoteDir := testutil.SetupTestRepoWithRemote(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer testutil.CleanupTestRepo(t, remoteDir)

	if _, err := testutil.RunGitFlow(t, dir, "feature", "start", "parent-diverged"); err != nil {
		t.Fatalf("Failed to create feature branch: %v", err)
	}
	commitFeatureAndPush(t, dir, "feature.txt", "feature content", "Add feature file", "feature/parent-diverged")

	// Remote develop gains one commit; local develop gains a different one → diverged after fetch.
	advanceRemoteDevelop(t, remoteDir, "remote1.txt", "r1", "Remote develop commit")
	commitOnLocalDevelop(t, dir, "feature/parent-diverged", "local-dev.txt", "l1", "Local develop commit")

	developBefore := revParse(t, dir, "develop")

	output, err := testutil.RunGitFlow(t, dir, "feature", "finish", "parent-diverged")
	if err == nil {
		t.Errorf("Expected finish to fail when parent is diverged. Output: %s", output)
	}
	if !strings.Contains(output, "develop") {
		t.Errorf("Expected error to name the parent branch 'develop'. Output: %s", output)
	}
	if !strings.Contains(output, "diverged") {
		t.Errorf("Expected error to mention 'diverged'. Output: %s", output)
	}
	if after := revParse(t, dir, "develop"); after != developBefore {
		t.Errorf("Expected develop unchanged after aborted finish. Before: %s After: %s", developBefore, after)
	}
	if !testutil.BranchExists(t, dir, "feature/parent-diverged") {
		t.Error("Expected feature branch to still exist after aborted finish")
	}
}

// Scenario 4: Parent ahead of its remote — proceeds (the key divergence from the topic rule).
func TestFinishParentAheadProceeds(t *testing.T) {
	t.Parallel()
	dir, remoteDir := testutil.SetupTestRepoWithRemote(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer testutil.CleanupTestRepo(t, remoteDir)

	if _, err := testutil.RunGitFlow(t, dir, "feature", "start", "parent-ahead"); err != nil {
		t.Fatalf("Failed to create feature branch: %v", err)
	}
	featureSha := commitFeatureAndPush(t, dir, "feature.txt", "feature content", "Add feature file", "feature/parent-ahead")

	// Local develop gains a commit that is not pushed → local develop ahead of origin/develop.
	commitOnLocalDevelop(t, dir, "feature/parent-ahead", "local-dev.txt", "l1", "Local develop commit")

	developBefore := revParse(t, dir, "develop")

	output, err := testutil.RunGitFlow(t, dir, "feature", "finish", "parent-ahead")
	if err != nil {
		t.Fatalf("Expected finish to succeed when parent is ahead of remote. Output: %s, err: %v", output, err)
	}
	assertFeatureMerged(t, dir, "feature/parent-ahead", featureSha, developBefore)
}

// Scenario 5: Topic in sync but parent behind — aborts on the parent check (runs independently of
// the topic check).
func TestFinishTopicInSyncParentBehindAborts(t *testing.T) {
	t.Parallel()
	dir, remoteDir := testutil.SetupTestRepoWithRemote(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer testutil.CleanupTestRepo(t, remoteDir)

	if _, err := testutil.RunGitFlow(t, dir, "feature", "start", "topic-ok-parent-behind"); err != nil {
		t.Fatalf("Failed to create feature branch: %v", err)
	}
	// Topic pushed and equal → topic check passes, so any abort must come from the parent check.
	commitFeatureAndPush(t, dir, "feature.txt", "feature content", "Add feature file", "feature/topic-ok-parent-behind")

	advanceRemoteDevelop(t, remoteDir, "remote1.txt", "r1", "Remote develop commit")

	developBefore := revParse(t, dir, "develop")

	output, err := testutil.RunGitFlow(t, dir, "feature", "finish", "topic-ok-parent-behind")
	if err == nil {
		t.Errorf("Expected finish to fail on the parent check even though the topic is in sync. Output: %s", output)
	}
	if !strings.Contains(output, "develop") {
		t.Errorf("Expected the abort to name the parent branch 'develop', not the topic. Output: %s", output)
	}
	if !strings.Contains(output, "behind") {
		t.Errorf("Expected error to mention 'behind'. Output: %s", output)
	}
	if after := revParse(t, dir, "develop"); after != developBefore {
		t.Errorf("Expected develop unchanged after aborted finish. Before: %s After: %s", developBefore, after)
	}
	if !testutil.BranchExists(t, dir, "feature/topic-ok-parent-behind") {
		t.Error("Expected feature branch to still exist after aborted finish")
	}
}

// Scenario 6: Parent behind, --force — the parent check is skipped; finish completes.
func TestFinishParentBehindForceSkipsCheck(t *testing.T) {
	t.Parallel()
	dir, remoteDir := testutil.SetupTestRepoWithRemote(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer testutil.CleanupTestRepo(t, remoteDir)

	if _, err := testutil.RunGitFlow(t, dir, "feature", "start", "parent-behind-force"); err != nil {
		t.Fatalf("Failed to create feature branch: %v", err)
	}
	featureSha := commitFeatureAndPush(t, dir, "feature.txt", "feature content", "Add feature file", "feature/parent-behind-force")

	advanceRemoteDevelop(t, remoteDir, "remote1.txt", "r1", "Remote develop commit")

	developBefore := revParse(t, dir, "develop")

	output, err := testutil.RunGitFlow(t, dir, "feature", "finish", "--force", "parent-behind-force")
	if err != nil {
		t.Fatalf("Expected finish --force to bypass the parent check. Output: %s, err: %v", output, err)
	}
	assertFeatureMerged(t, dir, "feature/parent-behind-force", featureSha, developBefore)
}

// Scenario 7: Parent has no tracking branch — the check is skipped; finish completes. origin/develop
// still exists locally, but develop's upstream is unset so HasTrackingBranch(develop) is false. This
// isolates the no-tracking gate from the benign missing-ref path (Scenario 9).
func TestFinishParentNoTrackingBranchSkips(t *testing.T) {
	t.Parallel()
	dir, remoteDir := testutil.SetupTestRepoWithRemote(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer testutil.CleanupTestRepo(t, remoteDir)

	// Unset develop's upstream so it has no tracking branch, while origin/develop stays present.
	if _, err := testutil.RunGit(t, dir, "branch", "--unset-upstream", "develop"); err != nil {
		t.Fatalf("Failed to unset develop upstream: %v", err)
	}
	if _, err := testutil.RunGit(t, dir, "rev-parse", "--verify", "develop@{upstream}"); err == nil {
		t.Fatal("Precondition failed: expected develop to have no upstream after unset")
	}

	if _, err := testutil.RunGitFlow(t, dir, "feature", "start", "parent-no-tracking"); err != nil {
		t.Fatalf("Failed to create feature branch: %v", err)
	}
	featureSha := commitFeatureAndPush(t, dir, "feature.txt", "feature content", "Add feature file", "feature/parent-no-tracking")
	developBefore := revParse(t, dir, "develop")

	output, err := testutil.RunGitFlow(t, dir, "feature", "finish", "parent-no-tracking")
	if err != nil {
		t.Fatalf("Expected finish to succeed when parent has no tracking branch. Output: %s, err: %v", output, err)
	}
	if strings.Contains(output, "behind") || strings.Contains(output, "diverged") {
		t.Errorf("Expected no parent sync abort when parent has no tracking branch. Output: %s", output)
	}
	assertFeatureMerged(t, dir, "feature/parent-no-tracking", featureSha, developBefore)
}

// Scenario 8: No remote configured — no fetch and no parent check; finish completes.
func TestFinishNoRemoteParentCheckSkipped(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	if _, err := testutil.RunGitFlow(t, dir, "init", "--defaults"); err != nil {
		t.Fatalf("Failed to init git-flow: %v", err)
	}

	if _, err := testutil.RunGitFlow(t, dir, "feature", "start", "no-remote"); err != nil {
		t.Fatalf("Failed to create feature branch: %v", err)
	}
	testutil.WriteFile(t, dir, "feature.txt", "feature content")
	if _, err := testutil.RunGit(t, dir, "add", "feature.txt"); err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	if _, err := testutil.RunGit(t, dir, "commit", "-m", "Add feature file"); err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}
	featureSha := revParse(t, dir, "feature/no-remote")
	developBefore := revParse(t, dir, "develop")

	output, err := testutil.RunGitFlow(t, dir, "feature", "finish", "no-remote")
	if err != nil {
		t.Fatalf("Expected finish to succeed with no remote. Output: %s, err: %v", output, err)
	}
	if strings.Contains(output, "Fetching") {
		t.Errorf("Expected no fetch with no remote configured. Output: %s", output)
	}
	if strings.Contains(output, "behind") || strings.Contains(output, "diverged") {
		t.Errorf("Expected no parent sync abort with no remote. Output: %s", output)
	}
	assertFeatureMerged(t, dir, "feature/no-remote", featureSha, developBefore)
}

// Scenario 9: Stale origin/develop after the remote branch was deleted — the parent fetch reports a
// benign missing ref, the parent check is skipped against the stale ref, and finish completes
// without --force. The stale ref deliberately points at a *different* commit than local develop, so
// a broken implementation that compared it would abort.
func TestFinishParentStaleTrackingRefProceeds(t *testing.T) {
	t.Parallel()
	dir, remoteDir := testutil.SetupTestRepoWithRemote(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer testutil.CleanupTestRepo(t, remoteDir)

	if _, err := testutil.RunGitFlow(t, dir, "feature", "start", "parent-stale"); err != nil {
		t.Fatalf("Failed to create feature branch: %v", err)
	}
	featureSha := commitFeatureAndPush(t, dir, "feature.txt", "feature content", "Add feature file", "feature/parent-stale")

	// Advance origin/develop, then fetch so local origin/develop points at the advanced commit while
	// local develop stays behind — the stale ref now differs from local develop.
	advanceRemoteDevelop(t, remoteDir, "remote1.txt", "r1", "Remote develop commit")
	if _, err := testutil.RunGit(t, dir, "fetch", "origin", "develop"); err != nil {
		t.Fatalf("Failed to fetch develop: %v", err)
	}
	if revParse(t, dir, "refs/remotes/origin/develop") == revParse(t, dir, "develop") {
		t.Fatal("Precondition failed: expected stale origin/develop to differ from local develop")
	}

	// Delete develop on the remote from a throwaway clone, leaving the original's stale tracking ref.
	second := t.TempDir()
	if _, err := testutil.RunGit(t, second, "clone", remoteDir, "."); err != nil {
		t.Fatalf("Failed to clone remote: %v", err)
	}
	if _, err := testutil.RunGit(t, second, "push", "origin", "--delete", "develop"); err != nil {
		t.Fatalf("Failed to delete remote develop: %v", err)
	}
	if _, err := testutil.RunGit(t, dir, "rev-parse", "--verify", "refs/remotes/origin/develop"); err != nil {
		t.Fatalf("Precondition failed: expected the original clone to keep its stale origin/develop: %v", err)
	}

	developBefore := revParse(t, dir, "develop")

	output, err := testutil.RunGitFlow(t, dir, "feature", "finish", "parent-stale")
	if err != nil {
		t.Fatalf("Expected finish to succeed against a stale parent tracking ref. Output: %s, err: %v", output, err)
	}
	if strings.Contains(output, "behind") || strings.Contains(output, "diverged") {
		t.Errorf("Expected no parent sync abort against a stale (missing-remote) ref. Output: %s", output)
	}
	assertFeatureMerged(t, dir, "feature/parent-stale", featureSha, developBefore)
}

// Scenario 10: Parent behind in the cached tracking ref, run with --no-fetch — the fetch is skipped
// but the parent check still runs against cached data and aborts (mirrors the topic --no-fetch rule).
func TestFinishParentBehindNoFetchStillChecks(t *testing.T) {
	t.Parallel()
	dir, remoteDir := testutil.SetupTestRepoWithRemote(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer testutil.CleanupTestRepo(t, remoteDir)

	if _, err := testutil.RunGitFlow(t, dir, "feature", "start", "parent-behind-nofetch"); err != nil {
		t.Fatalf("Failed to create feature branch: %v", err)
	}
	commitFeatureAndPush(t, dir, "feature.txt", "feature content", "Add feature file", "feature/parent-behind-nofetch")

	// Advance origin/develop and fetch once so the cached tracking ref shows develop behind.
	advanceRemoteDevelop(t, remoteDir, "remote1.txt", "r1", "Remote develop commit")
	if _, err := testutil.RunGit(t, dir, "fetch", "origin", "develop"); err != nil {
		t.Fatalf("Failed to fetch develop: %v", err)
	}

	developBefore := revParse(t, dir, "develop")

	output, err := testutil.RunGitFlow(t, dir, "feature", "finish", "--no-fetch", "parent-behind-nofetch")
	if err == nil {
		t.Errorf("Expected finish --no-fetch to still abort on the cached behind parent. Output: %s", output)
	}
	if strings.Contains(output, "Fetching") {
		t.Errorf("Expected no fetch with --no-fetch. Output: %s", output)
	}
	if !strings.Contains(output, "develop") || !strings.Contains(output, "behind") {
		t.Errorf("Expected a 'develop behind' abort against cached data. Output: %s", output)
	}
	if after := revParse(t, dir, "develop"); after != developBefore {
		t.Errorf("Expected develop unchanged after aborted finish. Before: %s After: %s", developBefore, after)
	}
	if !testutil.BranchExists(t, dir, "feature/parent-behind-nofetch") {
		t.Error("Expected feature branch to still exist after aborted finish")
	}
}

// Scenario 11: Parent compare-failure — the parent block fails closed. The loose origin/develop ref
// is corrupted so develop@{upstream} still resolves (HasTrackingBranch stays true) but the sync
// comparison cannot walk it. --no-fetch prevents a fetch from repairing the ref first.
func TestFinishParentCompareErrorAborts(t *testing.T) {
	t.Parallel()
	dir, remoteDir := testutil.SetupTestRepoWithRemote(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer testutil.CleanupTestRepo(t, remoteDir)

	if _, err := testutil.RunGitFlow(t, dir, "feature", "start", "parent-compare-error"); err != nil {
		t.Fatalf("Failed to create feature branch: %v", err)
	}
	// Topic pushed and equal so the topic check passes and we reach the parent block.
	commitFeatureAndPush(t, dir, "feature.txt", "feature content", "Add feature file", "feature/parent-compare-error")

	// Corrupt the loose parent tracking ref: point it at a nonexistent object.
	if _, err := testutil.RunGit(t, dir, "rev-parse", "--verify", "origin/develop"); err != nil {
		t.Fatalf("Precondition failed: expected a loose origin/develop ref: %v", err)
	}
	if err := testutil.WriteFile(t, dir, ".git/refs/remotes/origin/develop", "deadbeefdeadbeefdeadbeefdeadbeefdeadbeef\n"); err != nil {
		t.Fatalf("Failed to corrupt parent tracking ref: %v", err)
	}

	developBefore := revParse(t, dir, "develop")

	output, err := testutil.RunGitFlow(t, dir, "feature", "finish", "--no-fetch", "parent-compare-error")
	if err == nil {
		t.Errorf("Expected finish to fail when the parent sync status cannot be determined. Output: %s", output)
	}
	if !strings.Contains(output, "determine sync status") || !strings.Contains(output, "develop") {
		t.Errorf("Expected the error to name the sync-status determination for 'develop'. Output: %s", output)
	}
	if after := revParse(t, dir, "develop"); after != developBefore {
		t.Errorf("Expected develop unchanged after aborted finish. Before: %s After: %s", developBefore, after)
	}
	if !testutil.BranchExists(t, dir, "feature/parent-compare-error") {
		t.Error("Expected feature branch to still exist after aborted finish")
	}
}
