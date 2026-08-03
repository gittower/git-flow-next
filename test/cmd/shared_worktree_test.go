package cmd_test

import (
	"os"
	"strings"
	"testing"

	"github.com/gittower/git-flow-next/test/testutil"
)

// TestSharedWorktreeInheritsConfig covers scenario 31: a linked worktree inherits
// the copied config from the common .git/config, with no separate activation.
// Steps:
// 1. init --shared --defaults in the main clone (local now carries the copy)
// 2. Adds a linked worktree on a new branch off main
// 3. Runs 'feature start x' from the worktree
// 4. Verifies no first-run prompt/notice and that feature/x exists from the worktree
func TestSharedWorktreeInheritsConfig(t *testing.T) {
	t.Parallel()
	dir := initSharedDefaults(t)
	defer testutil.CleanupTestRepo(t, dir)

	wtPath, err := os.MkdirTemp("", "git-flow-test-wt-*")
	if err != nil {
		t.Fatalf("failed to create worktree temp dir: %v", err)
	}
	// git worktree add requires the destination not to pre-exist as a populated dir.
	os.RemoveAll(wtPath)
	if out, err := testutil.RunGit(t, dir, "worktree", "add", "-b", "wt-test", wtPath, "main"); err != nil {
		t.Fatalf("git worktree add failed: %v\n%s", err, out)
	}
	t.Cleanup(func() {
		_, _ = testutil.RunGit(t, dir, "worktree", "remove", "--force", wtPath)
		os.RemoveAll(wtPath)
	})

	out, err := testutil.RunGitFlow(t, wtPath, "feature", "start", "x")
	if err != nil {
		t.Fatalf("feature start from worktree failed: %v\n%s", err, out)
	}
	if strings.Contains(out, ".gitflow") {
		t.Errorf("expected no first-run .gitflow prompt/notice from the worktree, got: %s", out)
	}
	if !testutil.BranchExists(t, wtPath, "feature/x") {
		t.Error("expected feature/x to exist when created from the worktree")
	}
}
