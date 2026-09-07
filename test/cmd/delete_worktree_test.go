package cmd_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gittower/git-flow-next/test/testutil"
)

// ---------------------------------------------------------------------------
// delete + worktree cleanup (spec #175)
// ---------------------------------------------------------------------------

// TestDeleteRemovesCleanManagedWorktree covers spec scenario 9: deleting a
// branch with a clean git-flow-created worktree removes the worktree and the
// branch.
// Steps:
// 1. Initializes git-flow, creates feature/x and a managed worktree for it
// 2. Runs 'git flow feature delete x' from the main worktree
// 3. Verifies exit 0, the worktree directory and admin entry are gone, the marker is cleared, and the branch is gone
func TestDeleteRemovesCleanManagedWorktree(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer os.RemoveAll(worktreeRootFor(dir))
	createFreeBranch(t, dir, "feature/x")
	wtPath := addWorktree(t, dir, "feature/x")

	output, err := testutil.RunGitFlow(t, dir, "feature", "delete", "x")
	if err != nil {
		t.Fatalf("feature delete failed: %v\nOutput: %s", err, output)
	}

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

// TestDeleteDetachesHandMadeWorktree covers spec scenario 10: a hand-made
// worktree is detached, not removed, when its branch is deleted.
// Steps:
// 1. Initializes git-flow and creates feature/x with a hand-made worktree
// 2. Runs 'git flow feature delete x'
// 3. Verifies exit 0, the branch is gone, and the worktree survives, detached
func TestDeleteDetachesHandMadeWorktree(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	createFreeBranch(t, dir, "feature/x")
	wtPath := filepath.Join(t.TempDir(), "handmade")
	if out, err := testutil.RunGit(t, dir, "worktree", "add", wtPath, "feature/x"); err != nil {
		t.Fatalf("git worktree add failed: %v\nOutput: %s", err, out)
	}

	output, err := testutil.RunGitFlow(t, dir, "feature", "delete", "x")
	if err != nil {
		t.Fatalf("feature delete failed: %v\nOutput: %s", err, output)
	}

	if testutil.BranchExists(t, dir, "feature/x") {
		t.Error("Expected feature/x to be deleted")
	}
	if !strings.Contains(gitWorktreeList(t, dir), wtPath) {
		t.Errorf("Expected git worktree list to still show %s", wtPath)
	}
	assertWorktreeHeadDetached(t, wtPath)
}

// TestDeleteKeepWorktreeDetachesManagedWorktree covers spec scenario 11:
// --keep-worktree detaches a git-flow-created worktree instead of removing it.
// Steps:
// 1. Initializes git-flow and creates feature/x with a managed worktree
// 2. Runs 'git flow feature delete x --keep-worktree'
// 3. Verifies exit 0, the branch is gone, the directory survives detached, and the marker is cleared
func TestDeleteKeepWorktreeDetachesManagedWorktree(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer os.RemoveAll(worktreeRootFor(dir))
	createFreeBranch(t, dir, "feature/x")
	wtPath := addWorktree(t, dir, "feature/x")

	output, err := testutil.RunGitFlow(t, dir, "feature", "delete", "x", "--keep-worktree")
	if err != nil {
		t.Fatalf("feature delete --keep-worktree failed: %v\nOutput: %s", err, output)
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

// TestDeleteRefusesUntrackedFilesWithoutForceWorktree covers spec scenario 12
// (first half): untracked files in a managed worktree refuse deletion without
// --force-worktree.
// Steps:
// 1. Initializes git-flow, creates feature/x with a managed worktree, and writes an untracked file inside it
// 2. Runs 'git flow feature delete x'
// 3. Verifies exit 6, a message naming --force-worktree, and that the branch and worktree survive
func TestDeleteRefusesUntrackedFilesWithoutForceWorktree(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer os.RemoveAll(worktreeRootFor(dir))
	createFreeBranch(t, dir, "feature/x")
	wtPath := addWorktree(t, dir, "feature/x")
	if err := os.WriteFile(filepath.Join(wtPath, "scratch.txt"), []byte("scratch"), 0644); err != nil {
		t.Fatalf("Failed to write untracked file: %v", err)
	}

	output, err := testutil.RunGitFlow(t, dir, "feature", "delete", "x")
	if got := worktreeExitCode(err); got != 6 {
		t.Fatalf("Expected exit code 6, got %d\nOutput: %s", got, output)
	}
	if !strings.Contains(output, "--force-worktree") {
		t.Errorf("Expected the refusal to name --force-worktree, got: %s", output)
	}
	if !testutil.BranchExists(t, dir, "feature/x") {
		t.Error("Expected feature/x to survive the refusal")
	}
	if _, err := os.Stat(filepath.Join(wtPath, "scratch.txt")); err != nil {
		t.Errorf("Expected the untracked file to survive: %v", err)
	}
}

// TestDeleteForceWorktreeRemovesUntrackedFiles covers spec scenario 12 (second
// half): --force-worktree removes a managed worktree that has untracked files.
// Steps:
// 1. Initializes git-flow, creates feature/x with a managed worktree, and writes an untracked file inside it
// 2. Runs 'git flow feature delete x --force-worktree'
// 3. Verifies exit 0 and that the worktree and branch are both gone
func TestDeleteForceWorktreeRemovesUntrackedFiles(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer os.RemoveAll(worktreeRootFor(dir))
	createFreeBranch(t, dir, "feature/x")
	wtPath := addWorktree(t, dir, "feature/x")
	if err := os.WriteFile(filepath.Join(wtPath, "scratch.txt"), []byte("scratch"), 0644); err != nil {
		t.Fatalf("Failed to write untracked file: %v", err)
	}

	output, err := testutil.RunGitFlow(t, dir, "feature", "delete", "x", "--force-worktree")
	if err != nil {
		t.Fatalf("feature delete --force-worktree failed: %v\nOutput: %s", err, output)
	}

	if _, err := os.Stat(wtPath); !os.IsNotExist(err) {
		t.Errorf("Expected worktree directory %q to be gone", wtPath)
	}
	if testutil.BranchExists(t, dir, "feature/x") {
		t.Error("Expected feature/x to be deleted")
	}
}

// TestDeleteFromInsideOwnWorktreeNavigatesToMainWorktree covers spec scenario
// 13: deleting a branch from inside its own git-flow-created worktree records
// the main worktree as the destination — delete always prefers main, unlike
// finish, since it has no merge target to prefer instead.
// Steps:
// 1. Initializes git-flow and creates feature/x with a managed worktree
// 2. Runs 'git flow feature delete x' with cwd inside that worktree and GIT_FLOW_CD_FILE set
// 3. Verifies exit 0, the CD file holds the main worktree path, and the worktree and branch are gone
func TestDeleteFromInsideOwnWorktreeNavigatesToMainWorktree(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer os.RemoveAll(worktreeRootFor(dir))
	createFreeBranch(t, dir, "feature/x")
	wtPath := addWorktree(t, dir, "feature/x")
	cdFile := cdFilePath(t)
	mainRoot := testutil.EvalPath(t, dir)

	output, err := testutil.RunGitFlowWithEnv(t, wtPath, cdEnv(cdFile), "feature", "delete", "x")
	if err != nil {
		t.Fatalf("feature delete from inside the worktree failed: %v\nOutput: %s", err, output)
	}

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

// TestDeleteForceAndForceWorktreeOnUnmergedDirtyBranch covers spec scenario
// 14: '-f -W' force-deletes an unmerged branch and its dirty managed worktree
// together. (The spec text says '-D', but this repo's delete command's
// unmerged-force flag is '-f'/'--force' — '-D' is finish's --force-delete
// retention flag.)
// Steps:
// 1. Initializes git-flow, creates feature/x with unmerged commits and a managed worktree with an untracked file
// 2. Runs 'git flow feature delete x --force --force-worktree'
// 3. Verifies exit 0 and that both the branch and the worktree are gone
func TestDeleteForceAndForceWorktreeOnUnmergedDirtyBranch(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer os.RemoveAll(worktreeRootFor(dir))
	createFreeBranch(t, dir, "feature/x")
	wtPath := addWorktree(t, dir, "feature/x")
	commitFileInWorktree(t, wtPath, "unmerged.txt", "unmerged work", "unmerged commit")
	if err := os.WriteFile(filepath.Join(wtPath, "scratch.txt"), []byte("scratch"), 0644); err != nil {
		t.Fatalf("Failed to write untracked file: %v", err)
	}

	output, err := testutil.RunGitFlow(t, dir, "feature", "delete", "x", "--force", "--force-worktree")
	if err != nil {
		t.Fatalf("feature delete --force --force-worktree failed: %v\nOutput: %s", err, output)
	}

	if testutil.BranchExists(t, dir, "feature/x") {
		t.Error("Expected feature/x to be force-deleted")
	}
	if _, err := os.Stat(wtPath); !os.IsNotExist(err) {
		t.Errorf("Expected worktree directory %q to be gone", wtPath)
	}
}

// TestDeleteRefusesUnmergedBranchWithoutFreeingWorktree guards against a
// regression: the worktree used to be freed before 'git branch -d' had any
// chance to refuse an unmerged branch, so a clean-but-unmerged delete without
// --force lost its worktree even though the branch itself correctly survived
// — a refusal that "worked" but still cost the user their worktree.
// Steps:
// 1. Initializes git-flow, creates feature/x with a managed worktree and an unmerged commit (the worktree itself is clean — no uncommitted changes)
// 2. Runs 'git flow feature delete x' without --force
// 3. Verifies a non-zero exit, and that BOTH the branch and its worktree (directory, admin entry, provenance marker) survive
func TestDeleteRefusesUnmergedBranchWithoutFreeingWorktree(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer os.RemoveAll(worktreeRootFor(dir))
	createFreeBranch(t, dir, "feature/x")
	wtPath := addWorktree(t, dir, "feature/x")
	commitFileInWorktree(t, wtPath, "unmerged.txt", "unmerged work", "unmerged commit")

	output, err := testutil.RunGitFlow(t, dir, "feature", "delete", "x")
	if err == nil {
		t.Fatalf("Expected delete to refuse an unmerged branch, got success: %s", output)
	}
	if !testutil.BranchExists(t, dir, "feature/x") {
		t.Error("Expected feature/x to survive the refusal")
	}
	if _, statErr := os.Stat(wtPath); statErr != nil {
		t.Errorf("Expected the worktree directory to survive the refusal, got: %v", statErr)
	}
	if !strings.Contains(gitWorktreeList(t, dir), wtPath) {
		t.Error("Expected the worktree's admin entry to survive the refusal")
	}
	if !testutil.GitConfigExists(t, dir, managedMarkerFor("feature/x")) {
		t.Error("Expected the managed marker to survive the refusal")
	}
}

// TestDeleteRefusesAgainstConfiguredUpstreamNotHead guards against a
// regression in the mergedness pre-check above: 'git branch -d' checks a
// branch's configured upstream when it has one, not HEAD — so a pre-check
// that only compared against HEAD could pass (branch merged into HEAD) while
// the real 'git branch -d' still refuses (branch not merged into its
// upstream), freeing the worktree for a deletion that then fails anyway.
// Steps:
// 1. Initializes git-flow, creates feature/x with a managed worktree and a commit
// 2. Merges feature/x into develop directly (so it IS an ancestor of HEAD/develop)
// 3. Points feature/x's upstream at 'main' instead — which never received that merge, so feature/x is NOT an ancestor of its upstream
// 4. Runs 'git flow feature delete x' without --force
// 5. Verifies a non-zero exit, and that both the branch and its worktree survive
func TestDeleteRefusesAgainstConfiguredUpstreamNotHead(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer os.RemoveAll(worktreeRootFor(dir))
	createFreeBranch(t, dir, "feature/x")
	wtPath := addWorktree(t, dir, "feature/x")
	commitFileInWorktree(t, wtPath, "feature-x.txt", "hello", "add feature-x.txt")

	if out, err := testutil.RunGit(t, dir, "merge", "feature/x"); err != nil {
		t.Fatalf("Failed to merge feature/x into develop: %v\nOutput: %s", err, out)
	}
	if out, err := testutil.RunGit(t, dir, "branch", "--set-upstream-to=main", "feature/x"); err != nil {
		t.Fatalf("Failed to point feature/x's upstream at main: %v\nOutput: %s", err, out)
	}

	output, err := testutil.RunGitFlow(t, dir, "feature", "delete", "x")
	if err == nil {
		t.Fatalf("Expected delete to refuse (not merged into its configured upstream), got success: %s", output)
	}
	if !testutil.BranchExists(t, dir, "feature/x") {
		t.Error("Expected feature/x to survive the refusal")
	}
	if _, statErr := os.Stat(wtPath); statErr != nil {
		t.Errorf("Expected the worktree directory to survive the refusal, got: %v", statErr)
	}
}

// TestDeleteFromInsideOwnWorktreeChecksMergednessAgainstParent guards against a
// regression the #175 redirect could otherwise introduce: when the parent has
// no dedicated worktree of its own, delete's redirect lands on the main
// worktree, which may be checked out on some OTHER branch entirely — not the
// parent. Without an explicit checkout of the parent there, a non-force
// 'git branch -d' checks mergedness against whatever the main worktree
// happens to have checked out, which can wrongly refuse a branch that is
// genuinely merged into its real parent.
// Steps:
// 1. Initializes git-flow, commits a change on develop so it diverges from main, then checks main out in the main worktree (so the two are no longer the same commit, and the main worktree sits on a branch other than the parent)
// 2. Creates feature/x from develop (inheriting the divergent commit) and a managed worktree for it
// 3. Runs 'git flow feature delete x' (no --force) with cwd inside that worktree
// 4. Verifies exit 0 and the branch is gone — proving mergedness was checked against develop, not against whatever the main worktree had checked out
func TestDeleteFromInsideOwnWorktreeChecksMergednessAgainstParent(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer os.RemoveAll(worktreeRootFor(dir))

	if err := testutil.WriteFile(t, dir, "develop-only.txt", "diverges from main"); err != nil {
		t.Fatalf("Failed to write divergent file: %v", err)
	}
	if out, err := testutil.RunGit(t, dir, "add", "develop-only.txt"); err != nil {
		t.Fatalf("Failed to stage divergent file: %v\nOutput: %s", err, out)
	}
	if out, err := testutil.RunGit(t, dir, "commit", "-m", "develop-only commit"); err != nil {
		t.Fatalf("Failed to commit on develop: %v\nOutput: %s", err, out)
	}
	if out, err := testutil.RunGit(t, dir, "checkout", "main"); err != nil {
		t.Fatalf("Failed to checkout main in the main worktree: %v\nOutput: %s", err, out)
	}

	if out, err := testutil.RunGit(t, dir, "branch", "feature/x", "develop"); err != nil {
		t.Fatalf("Failed to create feature/x from develop: %v\nOutput: %s", err, out)
	}
	wtPath := addWorktree(t, dir, "feature/x")

	output, err := testutil.RunGitFlow(t, wtPath, "feature", "delete", "x")
	if err != nil {
		t.Fatalf("feature delete from inside the worktree failed (mergedness likely checked against the wrong branch): %v\nOutput: %s", err, output)
	}
	if testutil.BranchExists(t, dir, "feature/x") {
		t.Error("Expected feature/x to be deleted")
	}
}

// TestDeletePostHookRunsInSurvivingWorktree guards against a regression the
// #175 redirect could otherwise introduce: if the redirect happened only
// inside performDelete, WithHooks (which wraps the whole operation, including
// the post-delete hook) would still hold the ORIGINAL, pre-redirect repo
// handle — so a delete that just removed the worktree the user was standing
// in would run its post-hook with a working directory that no longer exists,
// and the hook process would fail to even start.
// Steps:
// 1. Initializes git-flow, creates feature/x with a managed worktree
// 2. Installs a post-flow-feature-delete hook that writes a marker file at a fixed path outside any worktree
// 3. Runs 'git flow feature delete x' with cwd inside feature/x's own worktree
// 4. Verifies exit 0, the worktree is gone, and the marker file was written — proving the post-hook actually ran (a stale cmd.Dir would have prevented it from starting at all)
func TestDeletePostHookRunsInSurvivingWorktree(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer os.RemoveAll(worktreeRootFor(dir))
	createFreeBranch(t, dir, "feature/x")
	wtPath := addWorktree(t, dir, "feature/x")

	markerFile := filepath.Join(t.TempDir(), "post-delete-hook-ran.txt")
	postScript := "#!/bin/sh\npwd > \"" + markerFile + "\"\n"
	createHookScript(t, dir, "post-flow-feature-delete", postScript)

	output, err := testutil.RunGitFlow(t, wtPath, "feature", "delete", "x")
	if err != nil {
		t.Fatalf("feature delete from inside the worktree failed: %v\nOutput: %s", err, output)
	}
	if _, err := os.Stat(wtPath); !os.IsNotExist(err) {
		t.Errorf("Expected worktree directory %q to be gone", wtPath)
	}

	hookCwd, readErr := os.ReadFile(markerFile)
	if readErr != nil {
		t.Fatalf("Expected the post-delete hook to have run and written %s, got: %v", markerFile, readErr)
	}
	if strings.Contains(strings.TrimSpace(string(hookCwd)), wtPath) {
		t.Errorf("Expected the post-delete hook to run outside the removed worktree, got cwd %q", strings.TrimSpace(string(hookCwd)))
	}
}

// TestDeleteNoWorktreeFlagsAreNoOps covers spec scenario 15 for delete: with no
// worktree for the branch, the new flags change nothing.
// Steps:
// 1. Initializes git-flow and creates feature/x with no worktree anywhere
// 2. Runs 'git flow feature delete x --keep-worktree --force-worktree'
// 3. Verifies exit 0, the branch is gone, and no worktree-related output appears
func TestDeleteNoWorktreeFlagsAreNoOps(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	createFreeBranch(t, dir, "feature/x")

	output, err := testutil.RunGitFlow(t, dir, "feature", "delete", "x", "--keep-worktree", "--force-worktree")
	if err != nil {
		t.Fatalf("feature delete failed: %v\nOutput: %s", err, output)
	}
	if testutil.BranchExists(t, dir, "feature/x") {
		t.Error("Expected feature/x to be deleted")
	}
	if strings.Contains(strings.ToLower(output), "worktree") {
		t.Errorf("Expected no worktree-related output, got: %s", output)
	}
}

// TestDeleteMainWorktreeFlagsAreNoOps covers spec scenario 16 for delete: a
// branch checked out in the main worktree is unaffected by the new flags.
// feature/x is left checked out (the current branch) in the main worktree so
// WorktreeForBranch resolves an entry with Main == true, exercising that
// branch of preflightWorktreeCleanup/freeWorktreeForBranch — checking out
// develop first would leave feature/x checked out nowhere and collapse this
// into scenario 15 (no worktree at all) instead.
// Steps:
// 1. Initializes git-flow and runs 'feature start x', leaving feature/x checked out in the main worktree
// 2. Runs 'git flow feature delete x --keep-worktree --force-worktree' while feature/x is still checked out
// 3. Verifies exit 0, the branch is gone, and no worktree-related output appears
func TestDeleteMainWorktreeFlagsAreNoOps(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	if out, err := testutil.RunGitFlow(t, dir, "feature", "start", "x"); err != nil {
		t.Fatalf("feature start failed: %v\nOutput: %s", err, out)
	}

	output, err := testutil.RunGitFlow(t, dir, "feature", "delete", "x", "--keep-worktree", "--force-worktree")
	if err != nil {
		t.Fatalf("feature delete failed: %v\nOutput: %s", err, output)
	}
	if testutil.BranchExists(t, dir, "feature/x") {
		t.Error("Expected feature/x to be deleted")
	}
	if strings.Contains(strings.ToLower(output), "worktree") {
		t.Errorf("Expected no worktree-related output, got: %s", output)
	}
}
