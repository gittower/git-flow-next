package cmd_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gittower/git-flow-next/test/testutil"
)

// createFeatureBranchWithWorktree seeds a committed feature branch held in a
// linked worktree and returns the worktree path. The main checkout is left on
// develop so the feature is available to worktrees.
func createFeatureBranchWithWorktree(t *testing.T, dir, feature string) string {
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

	if _, err := testutil.RunGit(t, dir, "checkout", "develop"); err != nil {
		t.Fatalf("Failed to checkout develop: %v", err)
	}

	wtDir := filepath.Join(t.TempDir(), "wt")
	if _, err := testutil.RunGit(t, dir, "worktree", "add", wtDir, "feature/"+feature); err != nil {
		t.Fatalf("Failed to add linked worktree: %v", err)
	}
	return wtDir
}

// TestDeleteFeatureWithLinkedWorktreeAutoRemoved verifies that delete removes a
// linked worktree holding the branch when remove-worktree is enabled.
func TestDeleteFeatureWithLinkedWorktreeAutoRemoved(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	wtDir := createFeatureBranchWithWorktree(t, dir, "delete-wt-feature")

	if _, err := testutil.RunGit(t, dir, "config", "gitflow.feature.delete.remove-worktree", "true"); err != nil {
		t.Fatalf("Failed to set remove-worktree config: %v", err)
	}

	// The branch is unmerged (its commit is not in develop), so force is needed.
	output, err := testutil.RunGitFlow(t, dir, "feature", "delete", "-f", "delete-wt-feature")
	if err != nil {
		t.Fatalf("Failed to delete feature with linked worktree: %v\nOutput: %s", err, output)
	}

	if _, statErr := os.Stat(wtDir); !os.IsNotExist(statErr) {
		t.Errorf("Expected linked worktree directory to be removed, stat error: %v", statErr)
	}
	if testutil.BranchExists(t, dir, "feature/delete-wt-feature") {
		t.Error("Expected feature branch to be deleted")
	}
}

// TestDeleteFeatureWithLinkedWorktreeFailsWithoutOptIn verifies that delete
// fails with git's own error when removal is not enabled.
func TestDeleteFeatureWithLinkedWorktreeFailsWithoutOptIn(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	wtDir := createFeatureBranchWithWorktree(t, dir, "delete-no-optin")

	output, err := testutil.RunGitFlow(t, dir, "feature", "delete", "-f", "delete-no-optin")
	if err == nil {
		t.Fatal("Expected delete to fail when a linked worktree holds the branch")
	}
	if !strings.Contains(output, "used by worktree") {
		t.Errorf("Expected git 'used by worktree' error, got:\n%s", output)
	}
	if _, statErr := os.Stat(wtDir); statErr != nil {
		t.Errorf("Expected linked worktree directory to remain, stat error: %v", statErr)
	}
	if !testutil.BranchExists(t, dir, "feature/delete-no-optin") {
		t.Error("Expected feature branch to remain")
	}
}

// TestDeleteFeatureWithDirtyWorktreeErrors verifies delete does not silently
// discard a dirty worktree.
func TestDeleteFeatureWithDirtyWorktreeErrors(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	wtDir := createFeatureBranchWithWorktree(t, dir, "delete-dirty")

	if err := testutil.WriteFile(t, wtDir, "untracked.txt", "dirty"); err != nil {
		t.Fatalf("Failed to write untracked file in worktree: %v", err)
	}
	if _, err := testutil.RunGit(t, dir, "config", "gitflow.feature.delete.remove-worktree", "true"); err != nil {
		t.Fatalf("Failed to set remove-worktree config: %v", err)
	}

	output, err := testutil.RunGitFlow(t, dir, "feature", "delete", "-f", "delete-dirty")
	if err == nil {
		t.Fatal("Expected delete to fail on a dirty worktree")
	}
	if !strings.Contains(output, "uncommitted or untracked changes") {
		t.Errorf("Expected actionable dirty-worktree error, got:\n%s", output)
	}
	if _, statErr := os.Stat(wtDir); statErr != nil {
		t.Errorf("Expected dirty worktree directory to remain, stat error: %v", statErr)
	}
	if !testutil.BranchExists(t, dir, "feature/delete-dirty") {
		t.Error("Expected feature branch to remain")
	}
}
