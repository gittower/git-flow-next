package cmd_test

import (
	"strings"
	"testing"

	"github.com/gittower/git-flow-next/test/testutil"
)

// TestIntegrateRejectsTopicBranch verifies integrate refuses a topic branch and
// points at finish.
//
// Steps:
//  1. init --defaults; git flow feature start foo.
//  2. Run: git flow integrate feature/foo.
//  3. Assert non-zero exit, message mentions base branches and finish, no mutation.
func TestIntegrateRejectsTopicBranch(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	if out, err := testutil.RunGitFlow(t, dir, "init", "--defaults"); err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, out)
	}
	if out, err := testutil.RunGitFlow(t, dir, "feature", "start", "foo"); err != nil {
		t.Fatalf("Failed to start feature: %v\nOutput: %s", err, out)
	}
	preMain := integRevParse(t, dir, "main")

	out, err := testutil.RunGitFlow(t, dir, "integrate", "feature/foo")
	if err == nil {
		t.Fatalf("Expected integrate on a topic branch to fail.\nOutput: %s", out)
	}
	if !strings.Contains(out, "base branch") || !strings.Contains(out, "finish") {
		t.Errorf("Expected error to mention base branches and finish, got: %s", out)
	}
	if got := integRevParse(t, dir, "main"); got != preMain {
		t.Errorf("Expected main unchanged, got %s", got)
	}
	if tags, _ := testutil.RunGit(t, dir, "tag", "-l"); strings.TrimSpace(tags) != "" {
		t.Errorf("Expected no tag, got: %s", tags)
	}
}

// TestIntegrateNoParent verifies integrating a branch with no configured parent
// errors.
//
// Steps:
//  1. init --defaults (main has no parent).
//  2. Run: git flow integrate main.
//  3. Assert non-zero exit, error mentions parent, nothing mutated.
func TestIntegrateNoParent(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	if out, err := testutil.RunGitFlow(t, dir, "init", "--defaults"); err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, out)
	}
	preMain := integRevParse(t, dir, "main")

	out, err := testutil.RunGitFlow(t, dir, "integrate", "main")
	if err == nil {
		t.Fatalf("Expected integrate main (no parent) to fail.\nOutput: %s", out)
	}
	if !strings.Contains(out, "parent") {
		t.Errorf("Expected error to mention parent, got: %s", out)
	}
	if got := integRevParse(t, dir, "main"); got != preMain {
		t.Errorf("Expected main unchanged, got %s", got)
	}
}

// TestIntegrateSelfParent verifies a branch configured as its own parent errors
// before any mutation.
//
// Steps:
//  1. init --defaults; set gitflow.branch.develop.parent develop.
//  2. Run: git flow integrate develop.
//  3. Assert non-zero exit, self-parent error, nothing mutated.
func TestIntegrateSelfParent(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	if out, err := testutil.RunGitFlow(t, dir, "init", "--defaults"); err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, out)
	}
	testutil.RunGit(t, dir, "config", "gitflow.branch.develop.parent", "develop")
	preDevelop := integRevParse(t, dir, "develop")

	out, err := testutil.RunGitFlow(t, dir, "integrate", "develop")
	if err == nil {
		t.Fatalf("Expected integrate with self-parent to fail.\nOutput: %s", out)
	}
	if !strings.Contains(out, "itself") {
		t.Errorf("Expected self-parent error, got: %s", out)
	}
	if got := integRevParse(t, dir, "develop"); got != preDevelop {
		t.Errorf("Expected develop unchanged, got %s", got)
	}
}

// TestIntegrateUpstreamStrategyNone verifies a base branch whose upstream
// strategy is none cannot be integrated.
//
// Steps:
//  1. init --defaults; set gitflow.branch.develop.upstreamStrategy none.
//  2. Run: git flow integrate develop.
//  3. Assert non-zero exit, strategy error, nothing merged.
func TestIntegrateUpstreamStrategyNone(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	if out, err := testutil.RunGitFlow(t, dir, "init", "--defaults"); err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, out)
	}
	testutil.RunGit(t, dir, "config", "gitflow.branch.develop.upstreamStrategy", "none")
	preMain := integRevParse(t, dir, "main")

	out, err := testutil.RunGitFlow(t, dir, "integrate", "develop")
	if err == nil {
		t.Fatalf("Expected integrate with upstream strategy none to fail.\nOutput: %s", out)
	}
	if !strings.Contains(out, "strategy") {
		t.Errorf("Expected upstream strategy error, got: %s", out)
	}
	if got := integRevParse(t, dir, "main"); got != preMain {
		t.Errorf("Expected main unchanged, got %s", got)
	}
}

// TestIntegrateUnknownBranch verifies integrating an unknown branch errors.
//
// Steps:
//  1. init --defaults.
//  2. Run: git flow integrate nonexistent.
//  3. Assert non-zero exit.
func TestIntegrateUnknownBranch(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	if out, err := testutil.RunGitFlow(t, dir, "init", "--defaults"); err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, out)
	}

	out, err := testutil.RunGitFlow(t, dir, "integrate", "nonexistent")
	if err == nil {
		t.Fatalf("Expected integrate of unknown branch to fail.\nOutput: %s", out)
	}
}

// TestIntegrateCurrentBranchNotBase verifies integrate with no arg on a non-base
// current branch errors.
//
// Steps:
//  1. init --defaults; git flow feature start foo (HEAD on feature/foo).
//  2. Run: git flow integrate (no arg).
//  3. Assert non-zero exit, error mentions base branch.
func TestIntegrateCurrentBranchNotBase(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	if out, err := testutil.RunGitFlow(t, dir, "init", "--defaults"); err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, out)
	}
	if out, err := testutil.RunGitFlow(t, dir, "feature", "start", "foo"); err != nil {
		t.Fatalf("Failed to start feature: %v\nOutput: %s", err, out)
	}

	out, err := testutil.RunGitFlow(t, dir, "integrate")
	if err == nil {
		t.Fatalf("Expected integrate on non-base current branch to fail.\nOutput: %s", out)
	}
	if !strings.Contains(out, "base branch") {
		t.Errorf("Expected error to mention base branch, got: %s", out)
	}
}

// TestIntegrateParentMissing verifies a missing parent branch errors.
//
// Steps:
//  1. init --defaults; checkout develop; delete main locally.
//  2. Run: git flow integrate develop.
//  3. Assert non-zero exit, nothing mutated.
func TestIntegrateParentMissing(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	if out, err := testutil.RunGitFlow(t, dir, "init", "--defaults"); err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, out)
	}
	if _, err := testutil.RunGit(t, dir, "checkout", "develop"); err != nil {
		t.Fatalf("Failed to checkout develop: %v", err)
	}
	if _, err := testutil.RunGit(t, dir, "branch", "-D", "main"); err != nil {
		t.Fatalf("Failed to delete main: %v", err)
	}
	preDevelop := integRevParse(t, dir, "develop")

	out, err := testutil.RunGitFlow(t, dir, "integrate", "develop")
	if err == nil {
		t.Fatalf("Expected integrate with missing parent to fail.\nOutput: %s", out)
	}
	if got := integRevParse(t, dir, "develop"); got != preDevelop {
		t.Errorf("Expected develop unchanged, got %s", got)
	}
}

// TestIntegrateSourceBranchDeleted verifies a configured-but-deleted base branch
// errors cleanly before any state mutation, rather than saving state and checking
// out the parent before the merge fails.
//
// Steps:
//  1. init --defaults; checkout main; delete the configured develop branch.
//  2. Run: git flow integrate develop.
//  3. Assert non-zero exit, error mentions the branch does not exist, no merge
//     state was written, and main is unchanged.
func TestIntegrateSourceBranchDeleted(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	if out, err := testutil.RunGitFlow(t, dir, "init", "--defaults"); err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, out)
	}
	if _, err := testutil.RunGit(t, dir, "checkout", "main"); err != nil {
		t.Fatalf("Failed to checkout main: %v", err)
	}
	// develop remains configured as a base branch, but the git branch is gone.
	if _, err := testutil.RunGit(t, dir, "branch", "-D", "develop"); err != nil {
		t.Fatalf("Failed to delete develop: %v", err)
	}
	preMain := integRevParse(t, dir, "main")

	out, err := testutil.RunGitFlow(t, dir, "integrate", "develop")
	if err == nil {
		t.Fatalf("Expected integrate of deleted source branch to fail.\nOutput: %s", out)
	}
	if !strings.Contains(out, "does not exist") {
		t.Errorf("Expected error to report the branch does not exist, got: %s", out)
	}
	if integStateExists(t, dir) {
		t.Error("Expected no merge state written when the source branch is missing")
	}
	if got := integRevParse(t, dir, "main"); got != preMain {
		t.Errorf("Expected main unchanged (%s), got %s", preMain, got)
	}
}

// TestIntegrateNotInitialized verifies integrate errors when git-flow is not
// initialized.
//
// Steps:
//  1. SetupTestRepo without git flow init.
//  2. Run: git flow integrate develop.
//  3. Assert non-zero exit with a not-initialized error.
func TestIntegrateNotInitialized(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	out, err := testutil.RunGitFlow(t, dir, "integrate", "develop")
	if err == nil {
		t.Fatalf("Expected integrate in uninitialized repo to fail.\nOutput: %s", out)
	}
	if !strings.Contains(out, "not initialized") {
		t.Errorf("Expected not-initialized error, got: %s", out)
	}
}
