package cmd_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gittower/git-flow-next/test/testutil"
)

// commitFileInWorktree writes name with content into the worktree at wtPath,
// stages it and commits it there, giving a topic branch a distinguishing
// change that finish's merge should carry onto the parent.
func commitFileInWorktree(t *testing.T, wtPath string, name string, content string, message string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(wtPath, name), []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write %s: %v", name, err)
	}
	if out, err := testutil.RunGit(t, wtPath, "add", name); err != nil {
		t.Fatalf("Failed to stage %s: %v\nOutput: %s", name, err, out)
	}
	if out, err := testutil.RunGit(t, wtPath, "commit", "-m", message); err != nil {
		t.Fatalf("Failed to commit %s: %v\nOutput: %s", name, err, out)
	}
}

// assertFileOnBranch fails unless name exists (and is committed) on branch.
func assertFileOnBranch(t *testing.T, dir string, branch string, name string) {
	t.Helper()
	if out, err := testutil.RunGit(t, dir, "show", branch+":"+name); err != nil {
		t.Errorf("Expected %s to exist on %s: %v\nOutput: %s", name, branch, err, out)
	}
}

// assertWorktreeHeadDetached fails unless the worktree at path has a detached
// HEAD (symbolic-ref fails).
func assertWorktreeHeadDetached(t *testing.T, path string) {
	t.Helper()
	if out, err := testutil.RunGit(t, path, "symbolic-ref", "-q", "HEAD"); err == nil {
		t.Errorf("Expected a detached HEAD at %s, got a symbolic ref: %s", path, out)
	}
}

// ---------------------------------------------------------------------------
// finish + worktree cleanup (spec #175)
// ---------------------------------------------------------------------------

// TestFinishRemovesCleanManagedWorktree covers spec scenario 1: finishing a
// branch with a clean git-flow-created worktree merges, removes the worktree,
// clears its marker, and deletes the branch.
// Steps:
// 1. Initializes git-flow, creates feature/x and a managed worktree for it with a distinguishing commit
// 2. Runs 'git flow feature finish x' from the main worktree
// 3. Verifies exit 0, the merge landed on develop, the worktree directory and admin entry are gone
// 4. Verifies the provenance marker is cleared and the branch is gone
func TestFinishRemovesCleanManagedWorktree(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer os.RemoveAll(worktreeRootFor(dir))
	createFreeBranch(t, dir, "feature/x")
	wtPath := addWorktree(t, dir, "feature/x")
	commitFileInWorktree(t, wtPath, "feature-x.txt", "hello", "add feature-x.txt")

	output, err := testutil.RunGitFlow(t, dir, "feature", "finish", "x")
	if err != nil {
		t.Fatalf("feature finish failed: %v\nOutput: %s", err, output)
	}

	assertFileOnBranch(t, dir, "develop", "feature-x.txt")
	if _, err := os.Stat(wtPath); !os.IsNotExist(err) {
		t.Errorf("Expected worktree directory %q to be gone", wtPath)
	}
	if strings.Contains(gitWorktreeList(t, dir), wtPath) {
		t.Error("Expected the admin entry to be gone")
	}
	if testutil.GitConfigExists(t, dir, managedMarkerFor("feature/x")) {
		t.Error("Expected the managed marker to be cleared")
	}
	if testutil.BranchExists(t, dir, "feature/x") {
		t.Error("Expected feature/x to be deleted")
	}
}

// TestFinishDetachesHandMadeWorktree covers spec scenario 2: a worktree created
// by hand is detached, not removed, and its files stay exactly as they were.
// Steps:
// 1. Initializes git-flow, creates feature/x and a hand-made worktree via plain 'git worktree add'
// 2. Writes an extra file inside it (proof the directory is untouched afterward)
// 3. Runs 'git flow feature finish x'
// 4. Verifies exit 0, the merge landed, the branch is gone
// 5. Verifies 'git worktree list' still shows the path, HEAD is detached, and both files survive
func TestFinishDetachesHandMadeWorktree(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	createFreeBranch(t, dir, "feature/x")
	wtPath := filepath.Join(t.TempDir(), "handmade")
	if out, err := testutil.RunGit(t, dir, "worktree", "add", wtPath, "feature/x"); err != nil {
		t.Fatalf("git worktree add failed: %v\nOutput: %s", err, out)
	}
	commitFileInWorktree(t, wtPath, "feature-x.txt", "hello", "add feature-x.txt")
	if err := os.WriteFile(filepath.Join(wtPath, "untouched.txt"), []byte("still here"), 0644); err != nil {
		t.Fatalf("Failed to write extra file: %v", err)
	}

	output, err := testutil.RunGitFlow(t, dir, "feature", "finish", "x")
	if err != nil {
		t.Fatalf("feature finish failed: %v\nOutput: %s", err, output)
	}

	assertFileOnBranch(t, dir, "develop", "feature-x.txt")
	if testutil.BranchExists(t, dir, "feature/x") {
		t.Error("Expected feature/x to be deleted")
	}
	if !strings.Contains(gitWorktreeList(t, dir), wtPath) {
		t.Errorf("Expected git worktree list to still show %s", wtPath)
	}
	assertWorktreeHeadDetached(t, wtPath)
	assertFileContent(t, filepath.Join(wtPath, "untouched.txt"), "still here")
	assertFileContent(t, filepath.Join(wtPath, "feature-x.txt"), "hello")
}

// TestFinishKeepWorktreeDetachesManagedWorktree covers spec scenario 3:
// --keep-worktree applies the detach path to a git-flow-created worktree too.
// Steps:
// 1. Initializes git-flow and creates feature/x with a managed worktree
// 2. Runs 'git flow feature finish x --keep-worktree'
// 3. Verifies exit 0, the merge landed, the branch is deleted
// 4. Verifies the directory survives, detached, and its provenance marker is cleared
func TestFinishKeepWorktreeDetachesManagedWorktree(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer os.RemoveAll(worktreeRootFor(dir))
	createFreeBranch(t, dir, "feature/x")
	wtPath := addWorktree(t, dir, "feature/x")

	output, err := testutil.RunGitFlow(t, dir, "feature", "finish", "x", "--keep-worktree")
	if err != nil {
		t.Fatalf("feature finish --keep-worktree failed: %v\nOutput: %s", err, output)
	}

	if testutil.BranchExists(t, dir, "feature/x") {
		t.Error("Expected feature/x to be deleted")
	}
	if info, err := os.Stat(wtPath); err != nil || !info.IsDir() {
		t.Fatalf("Expected the worktree directory to survive: %v", err)
	}
	assertWorktreeHeadDetached(t, wtPath)
	if testutil.GitConfigExists(t, dir, managedMarkerFor("feature/x")) {
		t.Error("Expected the managed marker to be cleared")
	}
}

// TestFinishRefusesDirtyManagedWorktreeWithoutForce covers spec scenario 4
// (first half): a dirty git-flow-created worktree without --force-worktree
// aborts before the merge starts.
// Steps:
// 1. Initializes git-flow, creates feature/x with a managed worktree, and modifies a tracked file inside it
// 2. Runs 'git flow feature finish x'
// 3. Verifies exit 6 and a message naming --force-worktree
// 4. Verifies develop never received the merge, feature/x still exists, and the worktree/modification/marker survive
func TestFinishRefusesDirtyManagedWorktreeWithoutForce(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer os.RemoveAll(worktreeRootFor(dir))
	createFreeBranch(t, dir, "feature/x")
	wtPath := addWorktree(t, dir, "feature/x")
	if err := os.WriteFile(filepath.Join(wtPath, "README.md"), []byte("modified"), 0644); err != nil {
		t.Fatalf("Failed to modify tracked file: %v", err)
	}

	output, err := testutil.RunGitFlow(t, dir, "feature", "finish", "x")
	if got := worktreeExitCode(err); got != 6 {
		t.Fatalf("Expected exit code 6, got %d\nOutput: %s", got, output)
	}
	if !strings.Contains(output, "--force-worktree") {
		t.Errorf("Expected the refusal to name --force-worktree, got: %s", output)
	}
	if !testutil.BranchExists(t, dir, "feature/x") {
		t.Error("Expected feature/x to survive the refusal")
	}
	content, err := os.ReadFile(filepath.Join(wtPath, "README.md"))
	if err != nil || string(content) != "modified" {
		t.Errorf("Expected the modification to survive, got %q (%v)", string(content), err)
	}
	if v := testutil.GitConfigValue(t, dir, managedMarkerFor("feature/x")); v != "true" {
		t.Errorf("Expected the marker to survive, got %q", v)
	}
	if testutil.IsMergeInProgress(t, dir) {
		t.Error("Expected no merge state to have been written")
	}
}

// TestFinishForceWorktreeRemovesDirtyManagedWorktree covers spec scenario 4
// (second half): --force-worktree lets a dirty git-flow-created worktree be
// removed and the finish proceed.
// Steps:
// 1. Initializes git-flow, creates feature/x with a managed worktree, and modifies a tracked file inside it
// 2. Runs 'git flow feature finish x --force-worktree'
// 3. Verifies exit 0, the merge landed, and the worktree and branch are gone
func TestFinishForceWorktreeRemovesDirtyManagedWorktree(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer os.RemoveAll(worktreeRootFor(dir))
	createFreeBranch(t, dir, "feature/x")
	wtPath := addWorktree(t, dir, "feature/x")
	if err := os.WriteFile(filepath.Join(wtPath, "README.md"), []byte("modified"), 0644); err != nil {
		t.Fatalf("Failed to modify tracked file: %v", err)
	}

	output, err := testutil.RunGitFlow(t, dir, "feature", "finish", "x", "--force-worktree")
	if err != nil {
		t.Fatalf("feature finish --force-worktree failed: %v\nOutput: %s", err, output)
	}

	if _, err := os.Stat(wtPath); !os.IsNotExist(err) {
		t.Errorf("Expected worktree directory %q to be gone", wtPath)
	}
	if testutil.BranchExists(t, dir, "feature/x") {
		t.Error("Expected feature/x to be deleted")
	}
}

// TestFinishDetachesDirtyHandMadeWorktreeWithoutForce covers spec scenario 5: a
// hand-made worktree with uncommitted changes needs no force at all — it is
// detached, and the changes survive.
// Steps:
// 1. Initializes git-flow, creates feature/x with a hand-made worktree
// 2. Modifies a tracked file and adds an untracked file inside it
// 3. Runs 'git flow feature finish x' with no worktree flags
// 4. Verifies exit 0, the merge landed, the branch is gone, and both changes are still present
func TestFinishDetachesDirtyHandMadeWorktreeWithoutForce(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	createFreeBranch(t, dir, "feature/x")
	wtPath := filepath.Join(t.TempDir(), "handmade")
	if out, err := testutil.RunGit(t, dir, "worktree", "add", wtPath, "feature/x"); err != nil {
		t.Fatalf("git worktree add failed: %v\nOutput: %s", err, out)
	}
	if err := os.WriteFile(filepath.Join(wtPath, "README.md"), []byte("modified"), 0644); err != nil {
		t.Fatalf("Failed to modify tracked file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(wtPath, "scratch.txt"), []byte("scratch"), 0644); err != nil {
		t.Fatalf("Failed to write untracked file: %v", err)
	}

	output, err := testutil.RunGitFlow(t, dir, "feature", "finish", "x")
	if err != nil {
		t.Fatalf("feature finish failed: %v\nOutput: %s", err, output)
	}

	if testutil.BranchExists(t, dir, "feature/x") {
		t.Error("Expected feature/x to be deleted")
	}
	assertWorktreeHeadDetached(t, wtPath)
	assertFileContent(t, filepath.Join(wtPath, "README.md"), "modified")
	if _, err := os.Stat(filepath.Join(wtPath, "scratch.txt")); err != nil {
		t.Errorf("Expected the untracked file to survive: %v", err)
	}
}

// TestFinishRefusesWorktreeWithOperationInProgress covers spec scenario 6: a
// worktree with a merge underway cannot be freed, so finish aborts before
// touching anything.
// Steps:
// 1. Initializes git-flow, creates feature/x with a managed worktree
// 2. Creates a diverging branch and, inside the worktree, starts a conflicting 'git merge' by hand, leaving it unresolved
// 3. Runs 'git flow feature finish x'
// 4. Verifies exit 6 and a message naming the worktree and 'merge'
// 5. Verifies nothing was removed or detached, the in-progress merge survives, and feature/x still exists
func TestFinishRefusesWorktreeWithOperationInProgress(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer os.RemoveAll(worktreeRootFor(dir))
	createFreeBranch(t, dir, "feature/x")
	wtPath := addWorktree(t, dir, "feature/x")
	commitFileInWorktree(t, wtPath, "conflict.txt", "from feature", "feature change")

	// 'git flow init' leaves develop checked out in the main worktree, so
	// committing conflicting content there (via the main worktree directly,
	// with plain git — no git-flow operation involved) needs no extra branch
	// or worktree of its own.
	if err := testutil.WriteFile(t, dir, "conflict.txt", "from develop"); err != nil {
		t.Fatalf("Failed to write conflicting content on develop: %v", err)
	}
	if out, err := testutil.RunGit(t, dir, "add", "conflict.txt"); err != nil {
		t.Fatalf("Failed to stage conflicting content: %v\nOutput: %s", err, out)
	}
	if out, err := testutil.RunGit(t, dir, "commit", "-m", "develop change"); err != nil {
		t.Fatalf("Failed to commit conflicting content: %v\nOutput: %s", err, out)
	}

	if out, err := testutil.RunGit(t, wtPath, "merge", "develop"); err == nil {
		t.Fatalf("Expected the merge to conflict, but it succeeded: %s", out)
	}
	if _, err := os.Stat(filepath.Join(wtPath, ".git")); err != nil {
		t.Fatalf("Worktree lost after inducing the conflict: %v", err)
	}

	output, err := testutil.RunGitFlow(t, dir, "feature", "finish", "x")
	if got := worktreeExitCode(err); got != 6 {
		t.Fatalf("Expected exit code 6, got %d: %s", got, output)
	}
	if !strings.Contains(output, "merge") {
		t.Errorf("Expected the refusal to name 'merge', got: %s", output)
	}
	if !strings.Contains(output, wtPath) {
		t.Errorf("Expected the refusal to name the worktree path, got: %s", output)
	}
	if !testutil.BranchExists(t, dir, "feature/x") {
		t.Error("Expected feature/x to survive the refusal")
	}
	// wtPath's .git is a FILE pointing at its private git-dir (the linked
	// worktree form), so MERGE_HEAD is not reachable by joining wtPath/.git
	// directly; 'rev-parse --verify' resolves it the same way git itself would.
	if out, err := testutil.RunGit(t, wtPath, "rev-parse", "--verify", "-q", "MERGE_HEAD"); err != nil {
		t.Errorf("Expected the in-progress merge to survive untouched: %v\nOutput: %s", err, out)
	}
}

// TestFinishFromInsideOwnWorktreeNavigatesToMainWorktree covers spec scenario
// 7: finish run from inside the branch's own git-flow-created worktree merges
// successfully (redirected away from that worktree so the merge's own
// checkout cannot repurpose it), then removes it and records the main
// worktree as the destination, since the parent branch (develop) has no
// worktree of its own here.
// Steps:
// 1. Initializes git-flow and moves the main worktree onto 'main' (so 'develop' is free for the merge to check out)
// 2. Creates feature/x with a managed worktree
// 3. Runs 'git flow feature finish x' with cwd inside that worktree and GIT_FLOW_CD_FILE set
// 4. Verifies exit 0, the merge landed on develop, and the worktree is gone
// 5. Verifies the CD file holds the main worktree path and the branch is deleted
func TestFinishFromInsideOwnWorktreeNavigatesToMainWorktree(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer os.RemoveAll(worktreeRootFor(dir))
	if out, err := testutil.RunGit(t, dir, "checkout", "main"); err != nil {
		t.Fatalf("Failed to move the main worktree onto main: %v\nOutput: %s", err, out)
	}
	createFreeBranch(t, dir, "feature/x")
	wtPath := addWorktree(t, dir, "feature/x")
	commitFileInWorktree(t, wtPath, "feature-x.txt", "hello", "add feature-x.txt")
	cdFile := cdFilePath(t)
	mainRoot := testutil.EvalPath(t, dir)

	output, err := testutil.RunGitFlowWithEnv(t, wtPath, cdEnv(cdFile), "feature", "finish", "x")
	if err != nil {
		t.Fatalf("feature finish from inside the worktree failed: %v\nOutput: %s", err, output)
	}

	assertFileOnBranch(t, dir, "develop", "feature-x.txt")
	if got := readCDFile(t, cdFile); got != mainRoot {
		t.Errorf("Expected CD file to hold the main worktree %q, got %q", mainRoot, got)
	}
	if _, err := os.Stat(wtPath); !os.IsNotExist(err) {
		t.Errorf("Expected worktree directory %q to be gone", wtPath)
	}
	if testutil.BranchExists(t, dir, "feature/x") {
		t.Error("Expected feature/x to be deleted")
	}
}

// TestFinishFromInsideOwnWorktreeNavigatesToParentWorktree covers spec
// scenario 7's other half: when the parent branch DOES have its own worktree,
// that is the destination, not the main-worktree fallback.
// Steps:
// 1. Initializes git-flow, moves the main worktree onto 'main', and gives 'develop' its own managed worktree
// 2. Creates feature/x with a managed worktree
// 3. Runs 'git flow feature finish x' with cwd inside the feature worktree and GIT_FLOW_CD_FILE set
// 4. Verifies exit 0 and that the CD file holds develop's worktree path, not the main worktree
func TestFinishFromInsideOwnWorktreeNavigatesToParentWorktree(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer os.RemoveAll(worktreeRootFor(dir))
	if out, err := testutil.RunGit(t, dir, "checkout", "main"); err != nil {
		t.Fatalf("Failed to move the main worktree onto main: %v\nOutput: %s", err, out)
	}
	developWtPath := addWorktree(t, dir, "develop")
	createFreeBranch(t, dir, "feature/x")
	wtPath := addWorktree(t, dir, "feature/x")
	cdFile := cdFilePath(t)

	output, err := testutil.RunGitFlowWithEnv(t, wtPath, cdEnv(cdFile), "feature", "finish", "x")
	if err != nil {
		t.Fatalf("feature finish from inside the worktree failed: %v\nOutput: %s", err, output)
	}

	if got := readCDFile(t, cdFile); got != developWtPath {
		t.Errorf("Expected CD file to hold develop's worktree %q, got %q", developWtPath, got)
	}
	if testutil.BranchExists(t, dir, "feature/x") {
		t.Error("Expected feature/x to be deleted")
	}
}

// TestFinishFromInsideHandMadeWorktreeWritesNothing covers spec scenario 8:
// finishing from inside a hand-made worktree never navigates, because the
// directory is detached in place rather than removed.
// Steps:
// 1. Initializes git-flow and creates feature/x with a hand-made worktree
// 2. Runs 'git flow feature finish x' with cwd inside that worktree and GIT_FLOW_CD_FILE set
// 3. Verifies exit 0 and that the CD file stays empty
// 4. Verifies the directory survives, detached, at the same path
func TestFinishFromInsideHandMadeWorktreeWritesNothing(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	createFreeBranch(t, dir, "feature/x")
	wtPath := filepath.Join(t.TempDir(), "handmade")
	if out, err := testutil.RunGit(t, dir, "worktree", "add", wtPath, "feature/x"); err != nil {
		t.Fatalf("git worktree add failed: %v\nOutput: %s", err, out)
	}
	cdFile := cdFilePath(t)

	output, err := testutil.RunGitFlowWithEnv(t, wtPath, cdEnv(cdFile), "feature", "finish", "x")
	if err != nil {
		t.Fatalf("feature finish from inside the worktree failed: %v\nOutput: %s", err, output)
	}

	assertCDFileEmpty(t, cdFile)
	if info, err := os.Stat(wtPath); err != nil || !info.IsDir() {
		t.Fatalf("Expected the worktree directory to survive: %v", err)
	}
	assertWorktreeHeadDetached(t, wtPath)
}

// TestFinishKeepBranchLeavesWorktreeUntouched pins a judgment call: freeing a
// worktree is only ever done because the branch is about to disappear, so
// --keep (retain the branch) leaves a managed worktree completely alone, even
// though --force-worktree/--keep-worktree were also given.
// Steps:
// 1. Initializes git-flow and creates feature/x with a managed worktree
// 2. Runs 'git flow feature finish x --keep --force-worktree'
// 3. Verifies exit 0, the merge landed, and feature/x still exists
// 4. Verifies the worktree is untouched: still present, still attached to feature/x, marker still true
func TestFinishKeepBranchLeavesWorktreeUntouched(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer os.RemoveAll(worktreeRootFor(dir))
	createFreeBranch(t, dir, "feature/x")
	wtPath := addWorktree(t, dir, "feature/x")

	output, err := testutil.RunGitFlow(t, dir, "feature", "finish", "x", "--keep", "--force-worktree")
	if err != nil {
		t.Fatalf("feature finish --keep failed: %v\nOutput: %s", err, output)
	}

	if !testutil.BranchExists(t, dir, "feature/x") {
		t.Error("Expected feature/x to survive --keep")
	}
	if !strings.Contains(gitWorktreeList(t, dir), wtPath) {
		t.Errorf("Expected the worktree to still be listed at %s", wtPath)
	}
	if got := testutil.GetCurrentBranch(t, wtPath); got != "feature/x" {
		t.Errorf("Expected the worktree to still be on feature/x, got %q", got)
	}
	if v := testutil.GitConfigValue(t, dir, managedMarkerFor("feature/x")); v != "true" {
		t.Errorf("Expected the marker to survive untouched, got %q", v)
	}
}

// TestFinishNoWorktreeFlagsAreNoOps covers spec scenario 15: with no worktree
// for the branch, the new flags change nothing.
// Steps:
// 1. Initializes git-flow and creates feature/x with no worktree anywhere
// 2. Runs 'git flow feature finish x --keep-worktree --force-worktree'
// 3. Verifies exit 0, the merge landed, the branch is gone, and no worktree-related output appears
func TestFinishNoWorktreeFlagsAreNoOps(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	createFreeBranch(t, dir, "feature/x")

	output, err := testutil.RunGitFlow(t, dir, "feature", "finish", "x", "--keep-worktree", "--force-worktree")
	if err != nil {
		t.Fatalf("feature finish failed: %v\nOutput: %s", err, output)
	}
	if testutil.BranchExists(t, dir, "feature/x") {
		t.Error("Expected feature/x to be deleted")
	}
	if strings.Contains(strings.ToLower(output), "worktree") {
		t.Errorf("Expected no worktree-related output, got: %s", output)
	}
}

// TestFinishMainWorktreeFlagsAreNoOps covers spec scenario 16: a branch checked
// out in the main worktree is unaffected by the new flags.
// Steps:
// 1. Initializes git-flow and runs 'feature start x', leaving feature/x checked out in the main worktree
// 2. Runs 'git flow feature finish x --keep-worktree --force-worktree'
// 3. Verifies exit 0, the merge landed, the branch is gone, and no worktree-related output appears
func TestFinishMainWorktreeFlagsAreNoOps(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	if out, err := testutil.RunGitFlow(t, dir, "feature", "start", "x"); err != nil {
		t.Fatalf("feature start failed: %v\nOutput: %s", err, out)
	}

	output, err := testutil.RunGitFlow(t, dir, "feature", "finish", "x", "--keep-worktree", "--force-worktree")
	if err != nil {
		t.Fatalf("feature finish failed: %v\nOutput: %s", err, output)
	}
	if testutil.BranchExists(t, dir, "feature/x") {
		t.Error("Expected feature/x to be deleted")
	}
	if strings.Contains(strings.ToLower(output), "worktree") {
		t.Errorf("Expected no worktree-related output, got: %s", output)
	}
}

// TestFinishContinuePreflightRefusesDirtyManagedWorktree verifies the
// technical note that --continue must run the same worktree pre-flight: a
// finish that conflicts, gets resolved, and then hits a dirtied managed
// worktree on --continue must refuse there too, leaving the merge state
// intact for a later retry.
// Steps:
// 1. Initializes git-flow, creates feature/x with a managed worktree, and sets up a merge conflict on finish
// 2. Runs 'git flow feature finish x', which stops with unresolved conflicts
// 3. Resolves the conflict and stages it, then dirties the worktree with an unrelated untracked file
// 4. Runs 'git flow feature finish x --continue'
// 5. Verifies exit 6, a message naming --force-worktree, and that the merge state file still exists
func TestFinishContinuePreflightRefusesDirtyManagedWorktree(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer os.RemoveAll(worktreeRootFor(dir))
	createFreeBranch(t, dir, "feature/x")
	wtPath := addWorktree(t, dir, "feature/x")
	commitFileInWorktree(t, wtPath, "conflict.txt", "from feature", "feature change")
	if out, err := testutil.RunGit(t, dir, "checkout", "develop"); err != nil {
		t.Fatalf("Failed to checkout develop: %v\nOutput: %s", err, out)
	}
	if err := testutil.WriteFile(t, dir, "conflict.txt", "from develop"); err != nil {
		t.Fatalf("Failed to write conflicting content on develop: %v", err)
	}
	if out, err := testutil.RunGit(t, dir, "add", "conflict.txt"); err != nil {
		t.Fatalf("Failed to stage conflicting content: %v\nOutput: %s", err, out)
	}
	if out, err := testutil.RunGit(t, dir, "commit", "-m", "develop change"); err != nil {
		t.Fatalf("Failed to commit conflicting content: %v\nOutput: %s", err, out)
	}

	output, err := testutil.RunGitFlow(t, dir, "feature", "finish", "x")
	if err == nil {
		t.Fatalf("Expected the finish to conflict, got success: %s", output)
	}
	if !testutil.IsMergeInProgress(t, dir) {
		t.Fatalf("Expected a merge to be in progress after the conflict")
	}

	if err := testutil.WriteFile(t, dir, "conflict.txt", "resolved"); err != nil {
		t.Fatalf("Failed to resolve conflict: %v", err)
	}
	if out, err := testutil.RunGit(t, dir, "add", "conflict.txt"); err != nil {
		t.Fatalf("Failed to stage resolution: %v\nOutput: %s", err, out)
	}

	if err := os.WriteFile(filepath.Join(wtPath, "scratch.txt"), []byte("scratch"), 0644); err != nil {
		t.Fatalf("Failed to dirty the worktree: %v", err)
	}

	output, err = testutil.RunGitFlow(t, dir, "feature", "finish", "x", "--continue")
	if got := worktreeExitCode(err); got != 6 {
		t.Fatalf("Expected exit code 6, got %d\nOutput: %s", got, output)
	}
	if !strings.Contains(output, "--force-worktree") {
		t.Errorf("Expected the refusal to name --force-worktree, got: %s", output)
	}
	if !testutil.GitFlowMergeStateExists(t, dir) {
		t.Error("Expected the merge state to survive the refused continue")
	}
}

// TestFinishWorktreeFlagsHaveNoConfigEquivalent pins the "Layer 3 only, no
// config key" scoping decision for #175: unlike finish's other options
// (--keep, --rebase, --tag, ...), --keep-worktree and --force-worktree have
// no gitflow.<type>.finish.* counterpart. A config key shaped like one must be
// silently ignored rather than "fixed" by a future config-hierarchy sweep.
// Steps:
// 1. Initializes git-flow, creates feature/x with a clean, managed worktree
// 2. Sets gitflow.feature.finish.keep-worktree=true in git config — a key that does not exist as a real option
// 3. Runs 'git flow feature finish x' with no worktree flags at all
// 4. Verifies the worktree was REMOVED, not detached, proving the config key was never consulted
func TestFinishWorktreeFlagsHaveNoConfigEquivalent(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer os.RemoveAll(worktreeRootFor(dir))
	createFreeBranch(t, dir, "feature/x")
	wtPath := addWorktree(t, dir, "feature/x")

	if out, err := testutil.RunGit(t, dir, "config", "gitflow.feature.finish.keep-worktree", "true"); err != nil {
		t.Fatalf("Failed to set config: %v\nOutput: %s", err, out)
	}

	output, err := testutil.RunGitFlow(t, dir, "feature", "finish", "x")
	if err != nil {
		t.Fatalf("feature finish failed: %v\nOutput: %s", err, output)
	}
	if _, statErr := os.Stat(wtPath); !os.IsNotExist(statErr) {
		t.Errorf("Expected the worktree directory to be removed despite the config key, got: %v", statErr)
	}
	if strings.Contains(gitWorktreeList(t, dir), wtPath) {
		t.Error("Expected the admin entry to be gone despite the config key")
	}
	if testutil.BranchExists(t, dir, "feature/x") {
		t.Error("Expected feature/x to be deleted")
	}
}
