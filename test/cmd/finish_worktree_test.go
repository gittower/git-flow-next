package cmd_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gittower/git-flow-next/test/testutil"
)

// createFeatureWithWorktree seeds a committed feature branch and returns a
// linked worktree holding that branch. The main checkout is switched back to
// develop first so the branch can be assigned to the linked worktree, mirroring
// the workflow where a feature is developed in its own worktree and finished
// from the main checkout.
func createFeatureWithWorktree(t *testing.T, dir, feature string) string {
	t.Helper()

	if _, err := testutil.RunGitFlow(t, dir, "init", "--defaults"); err != nil {
		t.Fatalf("Failed to initialize git-flow: %v", err)
	}
	if _, err := testutil.RunGitFlow(t, dir, "feature", "start", feature); err != nil {
		t.Fatalf("Failed to start feature branch: %v", err)
	}
	if err := testutil.WriteFile(t, dir, "feature.txt", "feature content"); err != nil {
		t.Fatalf("Failed to write feature file: %v", err)
	}
	if _, err := testutil.RunGit(t, dir, "add", "feature.txt"); err != nil {
		t.Fatalf("Failed to add feature file: %v", err)
	}
	if _, err := testutil.RunGit(t, dir, "commit", "-m", "Add feature file"); err != nil {
		t.Fatalf("Failed to commit feature file: %v", err)
	}

	// Switch the main checkout off the feature branch so it can live in a
	// linked worktree.
	if _, err := testutil.RunGit(t, dir, "checkout", "develop"); err != nil {
		t.Fatalf("Failed to checkout develop: %v", err)
	}

	wtDir := filepath.Join(t.TempDir(), "wt")
	if _, err := testutil.RunGit(t, dir, "worktree", "add", wtDir, "feature/"+feature); err != nil {
		t.Fatalf("Failed to add linked worktree: %v", err)
	}
	return wtDir
}

// TestFinishFeatureWithLinkedWorktreeAutoRemoved verifies that when
// gitflow.feature.finish.remove-worktree is enabled, finishing a feature whose
// branch lives in a linked worktree removes the worktree and deletes the branch.
func TestFinishFeatureWithLinkedWorktreeAutoRemoved(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	wtDir := createFeatureWithWorktree(t, dir, "worktree-feature")

	// Opt in via configuration.
	if _, err := testutil.RunGit(t, dir, "config", "gitflow.feature.finish.remove-worktree", "true"); err != nil {
		t.Fatalf("Failed to set remove-worktree config: %v", err)
	}

	output, err := testutil.RunGitFlow(t, dir, "feature", "finish", "worktree-feature")
	if err != nil {
		t.Fatalf("Failed to finish feature with linked worktree: %v\nOutput: %s", err, output)
	}

	if _, statErr := os.Stat(wtDir); !os.IsNotExist(statErr) {
		t.Errorf("Expected linked worktree directory to be removed, stat error: %v", statErr)
	}
	if testutil.BranchExists(t, dir, "feature/worktree-feature") {
		t.Error("Expected feature branch to be deleted")
	}
}

// TestFinishFeatureWithLinkedWorktreeFailsWithoutOptIn verifies the default
// behavior is unchanged: without remove-worktree enabled, finishing fails with
// git's own "used by worktree" error and leaves the worktree and branch intact.
func TestFinishFeatureWithLinkedWorktreeFailsWithoutOptIn(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	wtDir := createFeatureWithWorktree(t, dir, "no-optin-feature")

	output, err := testutil.RunGitFlow(t, dir, "feature", "finish", "no-optin-feature")
	if err == nil {
		t.Fatal("Expected finish to fail when a linked worktree holds the branch")
	}
	if !strings.Contains(output, "used by worktree") {
		t.Errorf("Expected git 'used by worktree' error, got:\n%s", output)
	}
	if _, statErr := os.Stat(wtDir); statErr != nil {
		t.Errorf("Expected linked worktree directory to remain, stat error: %v", statErr)
	}
	if !testutil.BranchExists(t, dir, "feature/no-optin-feature") {
		t.Error("Expected feature branch to remain")
	}
}

// TestFinishFeatureNoRemoveWorktreeFlagOverridesConfig verifies that the
// --no-remove-worktree flag wins over a configuration that enables removal.
func TestFinishFeatureNoRemoveWorktreeFlagOverridesConfig(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	createFeatureWithWorktree(t, dir, "flag-override-feature")

	if _, err := testutil.RunGit(t, dir, "config", "gitflow.feature.finish.remove-worktree", "true"); err != nil {
		t.Fatalf("Failed to set remove-worktree config: %v", err)
	}

	output, err := testutil.RunGitFlow(t, dir, "feature", "finish", "flag-override-feature", "--no-remove-worktree")
	if err == nil {
		t.Fatal("Expected finish to fail when removal is disabled via flag")
	}
	if !strings.Contains(output, "used by worktree") {
		t.Errorf("Expected git 'used by worktree' error, got:\n%s", output)
	}
}

// TestFinishFeatureRemoveWorktreeFlagEnablesRemoval verifies that the
// --remove-worktree flag enables removal without any configuration.
func TestFinishFeatureRemoveWorktreeFlagEnablesRemoval(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	wtDir := createFeatureWithWorktree(t, dir, "flag-enable-feature")

	output, err := testutil.RunGitFlow(t, dir, "feature", "finish", "flag-enable-feature", "--remove-worktree")
	if err != nil {
		t.Fatalf("Failed to finish feature with --remove-worktree: %v\nOutput: %s", err, output)
	}

	if _, statErr := os.Stat(wtDir); !os.IsNotExist(statErr) {
		t.Errorf("Expected linked worktree directory to be removed, stat error: %v", statErr)
	}
	if testutil.BranchExists(t, dir, "feature/flag-enable-feature") {
		t.Error("Expected feature branch to be deleted")
	}
}

// TestFinishFeatureWithDirtyWorktreeErrors verifies that a linked worktree with
// uncommitted or untracked changes is not silently discarded: finish errors
// with actionable guidance and leaves the worktree and branch intact.
func TestFinishFeatureWithDirtyWorktreeErrors(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	wtDir := createFeatureWithWorktree(t, dir, "dirty-feature")

	// Make the worktree dirty with an untracked file.
	if err := testutil.WriteFile(t, wtDir, "untracked.txt", "dirty"); err != nil {
		t.Fatalf("Failed to write untracked file in worktree: %v", err)
	}

	if _, err := testutil.RunGit(t, dir, "config", "gitflow.feature.finish.remove-worktree", "true"); err != nil {
		t.Fatalf("Failed to set remove-worktree config: %v", err)
	}

	output, err := testutil.RunGitFlow(t, dir, "feature", "finish", "dirty-feature")
	if err == nil {
		t.Fatal("Expected finish to fail on a dirty worktree")
	}
	if !strings.Contains(output, "uncommitted or untracked changes") {
		t.Errorf("Expected actionable dirty-worktree error, got:\n%s", output)
	}
	if !strings.Contains(output, "--force-remove-worktree") {
		t.Errorf("Expected error to suggest --force-remove-worktree, got:\n%s", output)
	}
	if _, statErr := os.Stat(wtDir); statErr != nil {
		t.Errorf("Expected dirty worktree directory to remain, stat error: %v", statErr)
	}
	if !testutil.BranchExists(t, dir, "feature/dirty-feature") {
		t.Error("Expected feature branch to remain")
	}
	if _, statErr := os.Stat(filepath.Join(dir, "feature.txt")); !os.IsNotExist(statErr) {
		t.Errorf("Expected finish to abort before merging the feature, stat error: %v", statErr)
	}
}

// TestFinishFeatureWithDirtyWorktreeForceRemoved verifies that enabling
// force-remove-worktree discards the dirty worktree and completes the finish.
func TestFinishFeatureWithDirtyWorktreeForceRemoved(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	wtDir := createFeatureWithWorktree(t, dir, "force-dirty-feature")

	if err := testutil.WriteFile(t, wtDir, "untracked.txt", "dirty"); err != nil {
		t.Fatalf("Failed to write untracked file in worktree: %v", err)
	}

	if _, err := testutil.RunGit(t, dir, "config", "gitflow.feature.finish.remove-worktree", "true"); err != nil {
		t.Fatalf("Failed to set remove-worktree config: %v", err)
	}
	if _, err := testutil.RunGit(t, dir, "config", "gitflow.feature.finish.force-remove-worktree", "true"); err != nil {
		t.Fatalf("Failed to set force-remove-worktree config: %v", err)
	}

	output, err := testutil.RunGitFlow(t, dir, "feature", "finish", "force-dirty-feature")
	if err != nil {
		t.Fatalf("Failed to finish feature with dirty worktree and force: %v\nOutput: %s", err, output)
	}

	if _, statErr := os.Stat(wtDir); !os.IsNotExist(statErr) {
		t.Errorf("Expected dirty linked worktree directory to be removed, stat error: %v", statErr)
	}
	if testutil.BranchExists(t, dir, "feature/force-dirty-feature") {
		t.Error("Expected feature branch to be deleted")
	}
}
