package cmd_test

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/gittower/git-flow-next/test/testutil"
)

// =============================================================================
// Shared helpers for the integrate command tests. These live here so all
// integrate_*_test.go files (package cmd_test) can use them.
// =============================================================================

// integRevParse returns the resolved commit hash for a ref.
func integRevParse(t *testing.T, dir, ref string) string {
	t.Helper()
	out, err := testutil.RunGit(t, dir, "rev-parse", ref)
	if err != nil {
		t.Fatalf("Failed to rev-parse %s: %v", ref, err)
	}
	return strings.TrimSpace(out)
}

// integMergeCount returns the number of merge commits reachable from ref.
func integMergeCount(t *testing.T, dir, ref string) int {
	t.Helper()
	out, err := testutil.RunGit(t, dir, "rev-list", "--merges", "--count", ref)
	if err != nil {
		t.Fatalf("Failed to count merges on %s: %v", ref, err)
	}
	n, err := strconv.Atoi(strings.TrimSpace(out))
	if err != nil {
		t.Fatalf("Failed to parse merge count %q: %v", out, err)
	}
	return n
}

// integIsAncestor reports whether ancestor is an ancestor of (or equal to)
// descendant. Returns false when either ref is unknown to the repository.
func integIsAncestor(t *testing.T, dir, ancestor, descendant string) bool {
	t.Helper()
	_, err := testutil.RunGit(t, dir, "merge-base", "--is-ancestor", ancestor, descendant)
	return err == nil
}

// integTagExists reports whether a tag with the given name exists.
func integTagExists(t *testing.T, dir, name string) bool {
	t.Helper()
	out, _ := testutil.RunGit(t, dir, "tag", "-l", name)
	return strings.TrimSpace(out) == name
}

// integTagIsAnnotated reports whether the named tag is an annotated (tag object)
// rather than a lightweight tag.
func integTagIsAnnotated(t *testing.T, dir, name string) bool {
	t.Helper()
	out, err := testutil.RunGit(t, dir, "cat-file", "-t", name)
	if err != nil {
		return false
	}
	return strings.TrimSpace(out) == "tag"
}

// integTagMessage returns the message body of an annotated tag.
func integTagMessage(t *testing.T, dir, name string) string {
	t.Helper()
	out, err := testutil.RunGit(t, dir, "tag", "-l", "--format=%(contents)", name)
	if err != nil {
		t.Fatalf("Failed to read tag message for %s: %v", name, err)
	}
	return out
}

// integAddCommit checks out branch, writes file with content, commits, and
// returns the resulting commit hash.
func integAddCommit(t *testing.T, dir, branch, file, content, msg string) string {
	t.Helper()
	if _, err := testutil.RunGit(t, dir, "checkout", branch); err != nil {
		t.Fatalf("Failed to checkout %s: %v", branch, err)
	}
	if err := testutil.WriteFile(t, dir, file, content); err != nil {
		t.Fatalf("Failed to write %s: %v", file, err)
	}
	if _, err := testutil.RunGit(t, dir, "add", file); err != nil {
		t.Fatalf("Failed to add %s: %v", file, err)
	}
	if _, err := testutil.RunGit(t, dir, "commit", "-m", msg); err != nil {
		t.Fatalf("Failed to commit on %s: %v", branch, err)
	}
	return integRevParse(t, dir, "HEAD")
}

// integGitDir returns the repository's .git directory (worktree-aware).
func integGitDir(t *testing.T, dir string) string {
	t.Helper()
	out, err := testutil.RunGit(t, dir, "rev-parse", "--git-dir")
	if err != nil {
		t.Fatalf("Failed to resolve git dir: %v", err)
	}
	gd := strings.TrimSpace(out)
	if !filepath.IsAbs(gd) {
		gd = filepath.Join(dir, gd)
	}
	return gd
}

// integRebaseInProgress reports whether a rebase is in progress (rebase-merge dir).
func integRebaseInProgress(t *testing.T, dir string) bool {
	t.Helper()
	_, err := os.Stat(filepath.Join(integGitDir(t, dir), "rebase-merge"))
	return err == nil
}

// integMergeHeadExists reports whether .git/MERGE_HEAD is present.
func integMergeHeadExists(t *testing.T, dir string) bool {
	t.Helper()
	_, err := os.Stat(filepath.Join(integGitDir(t, dir), "MERGE_HEAD"))
	return err == nil
}

// integStateExists reports whether the git-flow merge state file exists.
func integStateExists(t *testing.T, dir string) bool {
	t.Helper()
	_, err := os.Stat(filepath.Join(integGitDir(t, dir), "gitflow", "state", "merge.json"))
	return err == nil
}

// =============================================================================
// Happy paths
// =============================================================================

// TestIntegrateFastForwardNoTag verifies a linear integrate fast-forwards the
// parent without a merge commit or tag, and the auto-update child is a no-op.
//
// Steps:
//  1. init --defaults; checkout develop and add commit C (develop ahead of main).
//  2. Run: git flow integrate develop.
//  3. Assert main fast-forwarded to C with no merge commit, no tag, develop
//     unchanged (== main), HEAD on main, both branches still present.
func TestIntegrateFastForwardNoTag(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	if out, err := testutil.RunGitFlow(t, dir, "init", "--defaults"); err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, out)
	}

	cCommit := integAddCommit(t, dir, "develop", "c.txt", "C", "Add C on develop")
	preMerges := integMergeCount(t, dir, "main")

	out, err := testutil.RunGitFlow(t, dir, "integrate", "develop")
	if err != nil {
		t.Fatalf("integrate develop failed: %v\nOutput: %s", err, out)
	}

	if got := integRevParse(t, dir, "main"); got != cCommit {
		t.Errorf("Expected main to fast-forward to C (%s), got %s", cCommit, got)
	}
	if got := integMergeCount(t, dir, "main"); got != preMerges {
		t.Errorf("Expected no new merge commit on main, merge count changed from %d to %d", preMerges, got)
	}
	if tags, _ := testutil.RunGit(t, dir, "tag", "-l"); strings.TrimSpace(tags) != "" {
		t.Errorf("Expected no tag to be created, got: %s", tags)
	}
	if integRevParse(t, dir, "develop") != integRevParse(t, dir, "main") {
		t.Error("Expected develop auto-update to be a no-op (develop == main)")
	}
	if got := testutil.GetCurrentBranch(t, dir); got != "main" {
		t.Errorf("Expected HEAD on main, got %s", got)
	}
	if !testutil.BranchExists(t, dir, "main") || !testutil.BranchExists(t, dir, "develop") {
		t.Error("Expected both main and develop to still exist")
	}
}

// TestIntegrateCurrentBranch verifies integrate with no argument resolves the
// target from the current branch's parent.
//
// Steps:
//  1. init --defaults; add commit C to develop and leave HEAD on develop.
//  2. Run: git flow integrate (no argument).
//  3. Assert main fast-forwarded to C, HEAD on main.
func TestIntegrateCurrentBranch(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	if out, err := testutil.RunGitFlow(t, dir, "init", "--defaults"); err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, out)
	}

	cCommit := integAddCommit(t, dir, "develop", "c.txt", "C", "Add C on develop")
	if got := testutil.GetCurrentBranch(t, dir); got != "develop" {
		t.Fatalf("Expected HEAD on develop before integrate, got %s", got)
	}

	out, err := testutil.RunGitFlow(t, dir, "integrate")
	if err != nil {
		t.Fatalf("integrate (no arg) failed: %v\nOutput: %s", err, out)
	}

	if got := integRevParse(t, dir, "main"); got != cCommit {
		t.Errorf("Expected main to fast-forward to C (%s), got %s", cCommit, got)
	}
	if got := testutil.GetCurrentBranch(t, dir); got != "main" {
		t.Errorf("Expected HEAD on main, got %s", got)
	}
}

// TestIntegrateNothingToIntegrate verifies an already-up-to-date integrate is a
// clean no-op.
//
// Steps:
//  1. init --defaults (develop == main, nothing ahead).
//  2. Run: git flow integrate develop.
//  3. Assert exit 0, main unchanged, no tag, both branches present, HEAD on main.
func TestIntegrateNothingToIntegrate(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	if out, err := testutil.RunGitFlow(t, dir, "init", "--defaults"); err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, out)
	}

	preMain := integRevParse(t, dir, "main")

	out, err := testutil.RunGitFlow(t, dir, "integrate", "develop")
	if err != nil {
		t.Fatalf("integrate develop (no-op) failed: %v\nOutput: %s", err, out)
	}

	if got := integRevParse(t, dir, "main"); got != preMain {
		t.Errorf("Expected main unchanged (%s), got %s", preMain, got)
	}
	if tags, _ := testutil.RunGit(t, dir, "tag", "-l"); strings.TrimSpace(tags) != "" {
		t.Errorf("Expected no tag, got: %s", tags)
	}
	if !testutil.BranchExists(t, dir, "develop") {
		t.Error("Expected develop to still exist")
	}
	if got := testutil.GetCurrentBranch(t, dir); got != "main" {
		t.Errorf("Expected HEAD on main, got %s", got)
	}
}

// TestIntegrateNothingToIntegrateWithTag verifies the tag is still created even
// when the merge itself is a no-op.
//
// Steps:
//  1. init --defaults (develop == main).
//  2. Run: git flow integrate develop --tag v1.0.0.
//  3. Assert exit 0 and annotated tag v1.0.0 exists on main.
func TestIntegrateNothingToIntegrateWithTag(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	if out, err := testutil.RunGitFlow(t, dir, "init", "--defaults"); err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, out)
	}

	out, err := testutil.RunGitFlow(t, dir, "integrate", "develop", "--tag", "v1.0.0")
	if err != nil {
		t.Fatalf("integrate develop --tag failed: %v\nOutput: %s", err, out)
	}

	if !integTagExists(t, dir, "v1.0.0") {
		t.Error("Expected tag v1.0.0 to be created even for a no-op merge")
	}
	if !integTagIsAnnotated(t, dir, "v1.0.0") {
		t.Error("Expected v1.0.0 to be an annotated tag")
	}
	if integRevParse(t, dir, "v1.0.0^{commit}") != integRevParse(t, dir, "main") {
		t.Error("Expected v1.0.0 to point at main's tip")
	}
}

// TestIntegrateFetchPullsRemoteChanges verifies --fetch incorporates remote
// parent commits before integrating.
//
// Steps:
//  1. SetupTestRepoWithRemote; clone the remote and push commit R to main.
//  2. Give the original develop commit C not on local main; leave HEAD on develop.
//  3. Run: git flow integrate develop --fetch.
//  4. Assert main contains both R (fetched) and C.
func TestIntegrateFetchPullsRemoteChanges(t *testing.T) {
	t.Parallel()
	dir, remoteDir := testutil.SetupTestRepoWithRemote(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer testutil.CleanupTestRepo(t, remoteDir)

	// Second working copy that pushes R to main on the remote.
	secondDir := t.TempDir()
	if _, err := testutil.RunGit(t, secondDir, "clone", remoteDir, "."); err != nil {
		t.Fatalf("Failed to clone remote: %v", err)
	}
	testutil.ConfigureGitIdentity(t, secondDir)
	rCommit := integAddCommit(t, secondDir, "main", "remote.txt", "R", "Add R on remote main")
	if _, err := testutil.RunGit(t, secondDir, "push", "origin", "main"); err != nil {
		t.Fatalf("Failed to push R to remote: %v", err)
	}

	// Local develop gets commit C not on local main.
	cCommit := integAddCommit(t, dir, "develop", "c.txt", "C", "Add C on develop")

	out, err := testutil.RunGitFlow(t, dir, "integrate", "develop", "--fetch")
	if err != nil {
		t.Fatalf("integrate develop --fetch failed: %v\nOutput: %s", err, out)
	}

	if !integIsAncestor(t, dir, rCommit, "main") {
		t.Errorf("Expected fetched commit R (%s) to be present on main", rCommit)
	}
	if !integIsAncestor(t, dir, cCommit, "main") {
		t.Errorf("Expected commit C (%s) to be present on main", cCommit)
	}
}

// TestIntegrateNoFetchOverridesConfig verifies --no-fetch overrides a configured
// integrate.fetch=true default.
//
// Steps:
//  1. SetupTestRepoWithRemote; set gitflow.develop.integrate.fetch true.
//  2. Push commit R to main via a second clone; give local develop commit C.
//  3. Run: git flow integrate develop --no-fetch.
//  4. Assert R is absent from main; main advanced only by C.
func TestIntegrateNoFetchOverridesConfig(t *testing.T) {
	t.Parallel()
	dir, remoteDir := testutil.SetupTestRepoWithRemote(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer testutil.CleanupTestRepo(t, remoteDir)

	if _, err := testutil.RunGit(t, dir, "config", "gitflow.develop.integrate.fetch", "true"); err != nil {
		t.Fatalf("Failed to set integrate.fetch config: %v", err)
	}

	secondDir := t.TempDir()
	if _, err := testutil.RunGit(t, secondDir, "clone", remoteDir, "."); err != nil {
		t.Fatalf("Failed to clone remote: %v", err)
	}
	testutil.ConfigureGitIdentity(t, secondDir)
	rCommit := integAddCommit(t, secondDir, "main", "remote.txt", "R", "Add R on remote main")
	if _, err := testutil.RunGit(t, secondDir, "push", "origin", "main"); err != nil {
		t.Fatalf("Failed to push R to remote: %v", err)
	}

	cCommit := integAddCommit(t, dir, "develop", "c.txt", "C", "Add C on develop")

	out, err := testutil.RunGitFlow(t, dir, "integrate", "develop", "--no-fetch")
	if err != nil {
		t.Fatalf("integrate develop --no-fetch failed: %v\nOutput: %s", err, out)
	}

	if integIsAncestor(t, dir, rCommit, "main") {
		t.Errorf("Expected fetched commit R (%s) to be ABSENT from main with --no-fetch", rCommit)
	}
	if !integIsAncestor(t, dir, cCommit, "main") {
		t.Errorf("Expected commit C (%s) to be present on main", cCommit)
	}
}

// TestIntegrateFetchParentCheckedOutSurfacesFailure verifies that when the
// parent branch is checked out, the parent:parent fetch refspec git refuses is
// surfaced honestly: integrate warns and does NOT print "Fetch completed", so a
// stale local parent is not integrated silently. The operation stays non-fatal.
//
// Steps:
//  1. SetupTestRepoWithRemote; push commit R to main via a second clone.
//  2. Give local develop commit C; check out main locally so the parent is HEAD.
//  3. Run: git flow integrate develop --fetch.
//  4. Assert exit 0, a Warning is printed, "Fetch completed" is NOT printed,
//     C is integrated into main, and R (unfetchable) is absent from local main.
func TestIntegrateFetchParentCheckedOutSurfacesFailure(t *testing.T) {
	t.Parallel()
	dir, remoteDir := testutil.SetupTestRepoWithRemote(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer testutil.CleanupTestRepo(t, remoteDir)

	secondDir := t.TempDir()
	if _, err := testutil.RunGit(t, secondDir, "clone", remoteDir, "."); err != nil {
		t.Fatalf("Failed to clone remote: %v", err)
	}
	testutil.ConfigureGitIdentity(t, secondDir)
	rCommit := integAddCommit(t, secondDir, "main", "remote.txt", "R", "Add R on remote main")
	if _, err := testutil.RunGit(t, secondDir, "push", "origin", "main"); err != nil {
		t.Fatalf("Failed to push R to remote: %v", err)
	}

	cCommit := integAddCommit(t, dir, "develop", "c.txt", "C", "Add C on develop")

	// Check out the parent so the parent:parent refspec is refused by git.
	if _, err := testutil.RunGit(t, dir, "checkout", "main"); err != nil {
		t.Fatalf("Failed to checkout main: %v", err)
	}

	out, err := testutil.RunGitFlow(t, dir, "integrate", "develop", "--fetch")
	if err != nil {
		t.Fatalf("integrate develop --fetch should stay non-fatal: %v\nOutput: %s", err, out)
	}

	if !strings.Contains(out, "Warning") {
		t.Errorf("Expected a warning surfacing the failed parent fetch, got:\n%s", out)
	}
	if strings.Contains(out, "Fetch completed") {
		t.Errorf("Expected NO \"Fetch completed\" when the parent fast-forward failed, got:\n%s", out)
	}
	// C still integrates into local main; R could not be fetched into the
	// checked-out parent, so it is honestly absent rather than silently claimed.
	if !integIsAncestor(t, dir, cCommit, "main") {
		t.Errorf("Expected commit C (%s) integrated into main", cCommit)
	}
	if integIsAncestor(t, dir, rCommit, "main") {
		t.Errorf("Expected R (%s) absent from local main after the refused fetch", rCommit)
	}
}

// TestIntegrateFetchConfigYesPullsRemoteChanges verifies that
// gitflow.<base>.integrate.fetch=yes incorporates remote parent commits before
// integrating, matching git-config's truthy "yes" spelling. The Layer-1 default
// is false, so only a truthy config can pull R in.
//
// Steps:
//  1. SetupTestRepoWithRemote; set gitflow.develop.integrate.fetch yes.
//  2. Clone the remote and push commit R to main.
//  3. Give local develop commit C not on local main; leave HEAD on develop.
//  4. Run: git flow integrate develop (no fetch flag).
//  5. Assert main contains both R (fetched) and C.
func TestIntegrateFetchConfigYesPullsRemoteChanges(t *testing.T) {
	t.Parallel()
	dir, remoteDir := testutil.SetupTestRepoWithRemote(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer testutil.CleanupTestRepo(t, remoteDir)

	if _, err := testutil.RunGit(t, dir, "config", "gitflow.develop.integrate.fetch", "yes"); err != nil {
		t.Fatalf("Failed to set integrate.fetch config: %v", err)
	}

	// Second working copy that pushes R to main on the remote.
	secondDir := t.TempDir()
	if _, err := testutil.RunGit(t, secondDir, "clone", remoteDir, "."); err != nil {
		t.Fatalf("Failed to clone remote: %v", err)
	}
	testutil.ConfigureGitIdentity(t, secondDir)
	rCommit := integAddCommit(t, secondDir, "main", "remote.txt", "R", "Add R on remote main")
	if _, err := testutil.RunGit(t, secondDir, "push", "origin", "main"); err != nil {
		t.Fatalf("Failed to push R to remote: %v", err)
	}

	// Local develop gets commit C not on local main.
	cCommit := integAddCommit(t, dir, "develop", "c.txt", "C", "Add C on develop")

	out, err := testutil.RunGitFlow(t, dir, "integrate", "develop")
	if err != nil {
		t.Fatalf("integrate develop failed: %v\nOutput: %s", err, out)
	}

	if !integIsAncestor(t, dir, rCommit, "main") {
		t.Errorf("Expected fetched commit R (%s) to be present on main with integrate.fetch=yes", rCommit)
	}
	if !integIsAncestor(t, dir, cCommit, "main") {
		t.Errorf("Expected commit C (%s) to be present on main", cCommit)
	}
}
