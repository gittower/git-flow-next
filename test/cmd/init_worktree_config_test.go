package cmd_test

import (
	"testing"

	"github.com/gittower/git-flow-next/test/testutil"
)

// worktreeDefaultTopicTypes lists every topic branch type in DefaultConfig. The
// worktree default is written for all of them — bugfix included, which a
// hand-rolled writer is easy to forget.
var worktreeDefaultTopicTypes = []string{"feature", "bugfix", "release", "hotfix", "support"}

// worktreeDefaultBaseBranches lists the base branch types in DefaultConfig, where
// a worktree default is meaningless: start never creates a worktree for them.
var worktreeDefaultBaseBranches = []string{"main", "develop"}

// TestInitDoesNotWriteWorktreeForBaseBranches covers E2: init writes the Layer-1
// worktree key for topic branch types only.
//
// Config is read from LOCAL scope throughout. A merged 'git config --get' would
// let an ambient global key fail a correct implementation, and would let an
// ambient topic false hide a missing local write.
// Steps:
// 1. Sets up a fresh repository
// 2. Runs 'git flow init --defaults'
// 3. Verifies gitflow.branch.main.worktree and gitflow.branch.develop.worktree are absent from local config
// 4. Verifies gitflow.branch.<type>.worktree is explicitly false for every topic type, bugfix included
func TestInitDoesNotWriteWorktreeForBaseBranches(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	if out, err := testutil.RunGitFlow(t, dir, "init", "--defaults"); err != nil {
		t.Fatalf("init --defaults failed: %v\nOutput: %s", err, out)
	}

	for _, base := range worktreeDefaultBaseBranches {
		key := "gitflow.branch." + base + ".worktree"
		if testutil.GitConfigExists(t, dir, key) {
			t.Errorf("Expected %s to be absent, got %q", key, testutil.GitConfigValue(t, dir, key))
		}
	}

	for _, topic := range worktreeDefaultTopicTypes {
		key := "gitflow.branch." + topic + ".worktree"
		if got := testutil.GitConfigValue(t, dir, key); got != "false" {
			t.Errorf("Expected %s=false, got %q", key, got)
		}
	}
}

// TestInitSharedWritesWorktreeForTopicTypesOnly covers E3: the shared writer makes
// the same topic-only distinction, so no worktree noise reaches the committed
// .gitflow for base branches.
// Steps:
// 1. Sets up a fresh repository
// 2. Runs 'git flow init --shared --defaults'
// 3. Verifies .gitflow carries worktree=false for every topic type, bugfix included
// 4. Verifies .gitflow carries no worktree key under branch.main or branch.develop
// 5. Verifies 'git flow config status' reports in sync, proving the local and shared writers agree on the new key
func TestInitSharedWritesWorktreeForTopicTypesOnly(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	if out, err := testutil.RunGitFlow(t, dir, "init", "--shared", "--defaults"); err != nil {
		t.Fatalf("init --shared --defaults failed: %v\nOutput: %s", err, out)
	}

	for _, topic := range worktreeDefaultTopicTypes {
		key := "gitflow.branch." + topic + ".worktree"
		if got := testutil.SharedConfigValue(t, dir, key); got != "false" {
			t.Errorf("Expected .gitflow %s=false, got %q", key, got)
		}
	}

	for _, base := range worktreeDefaultBaseBranches {
		key := "gitflow.branch." + base + ".worktree"
		if got := testutil.SharedConfigValue(t, dir, key); got != "" {
			t.Errorf("Expected .gitflow to carry no %s, got %q", key, got)
		}
	}

	if out, err := testutil.RunGitFlow(t, dir, "config", "status"); err != nil {
		t.Errorf("Expected config status to report in sync: %v\nOutput: %s", err, out)
	}
}
