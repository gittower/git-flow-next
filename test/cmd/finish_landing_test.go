package cmd_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gittower/git-flow-next/test/testutil"
)

// setupParentMergeConflict prepares a repository in which finishing release/1.0.0
// conflicts while merging into the parent branch 'main'.
// Steps:
// 1. Writes shared.txt with content "base" on main and commits it
// 2. Fast-forwards develop to main so both branches share the base commit
// 3. Starts release/1.0.0 and sets shared.txt to "release"
// 4. Sets shared.txt to "hotfix" on main so the merge into main conflicts
// 5. Leaves HEAD on release/1.0.0
func setupParentMergeConflict(t *testing.T, dir string) {
	t.Helper()

	if _, err := testutil.RunGit(t, dir, "checkout", "main"); err != nil {
		t.Fatalf("Failed to checkout main: %v", err)
	}
	writeAndCommit(t, dir, "shared.txt", "base", "Add shared file")

	if _, err := testutil.RunGit(t, dir, "checkout", "develop"); err != nil {
		t.Fatalf("Failed to checkout develop: %v", err)
	}
	if _, err := testutil.RunGit(t, dir, "merge", "--ff-only", "main"); err != nil {
		t.Fatalf("Failed to fast-forward develop to main: %v", err)
	}

	if output, err := testutil.RunGitFlow(t, dir, "release", "start", "1.0.0"); err != nil {
		t.Fatalf("Failed to start release: %v\nOutput: %s", err, output)
	}
	writeAndCommit(t, dir, "shared.txt", "release", "Release change")

	if _, err := testutil.RunGit(t, dir, "checkout", "main"); err != nil {
		t.Fatalf("Failed to checkout main: %v", err)
	}
	writeAndCommit(t, dir, "shared.txt", "hotfix", "Diverging change on main")

	if _, err := testutil.RunGit(t, dir, "checkout", "release/1.0.0"); err != nil {
		t.Fatalf("Failed to checkout release/1.0.0: %v", err)
	}
}

// setupChildUpdateConflict prepares a repository in which finishing release/1.0.0
// merges into 'main' cleanly and then conflicts while auto-updating 'develop'.
// Steps:
// 1. Writes shared.txt with content "base" on main and commits it
// 2. Fast-forwards develop to main so both branches share the base commit
// 3. Starts release/1.0.0 and sets shared.txt to "release"
// 4. Sets shared.txt to "develop" on develop so the child update conflicts
// 5. Leaves HEAD on release/1.0.0
func setupChildUpdateConflict(t *testing.T, dir string) {
	t.Helper()

	if _, err := testutil.RunGit(t, dir, "checkout", "main"); err != nil {
		t.Fatalf("Failed to checkout main: %v", err)
	}
	writeAndCommit(t, dir, "shared.txt", "base", "Add shared file")

	if _, err := testutil.RunGit(t, dir, "checkout", "develop"); err != nil {
		t.Fatalf("Failed to checkout develop: %v", err)
	}
	if _, err := testutil.RunGit(t, dir, "merge", "--ff-only", "main"); err != nil {
		t.Fatalf("Failed to fast-forward develop to main: %v", err)
	}

	if output, err := testutil.RunGitFlow(t, dir, "release", "start", "1.0.0"); err != nil {
		t.Fatalf("Failed to start release: %v\nOutput: %s", err, output)
	}
	writeAndCommit(t, dir, "shared.txt", "release", "Release change")

	if _, err := testutil.RunGit(t, dir, "checkout", "develop"); err != nil {
		t.Fatalf("Failed to checkout develop: %v", err)
	}
	writeAndCommit(t, dir, "shared.txt", "develop", "Diverging change on develop")

	if _, err := testutil.RunGit(t, dir, "checkout", "release/1.0.0"); err != nil {
		t.Fatalf("Failed to checkout release/1.0.0: %v", err)
	}
}

// writeAndCommit writes a file in the test repository and commits it.
func writeAndCommit(t *testing.T, dir string, name string, content string, message string) {
	t.Helper()

	if err := testutil.WriteFile(t, dir, name, content); err != nil {
		t.Fatalf("Failed to write %s: %v", name, err)
	}
	if _, err := testutil.RunGit(t, dir, "add", name); err != nil {
		t.Fatalf("Failed to stage %s: %v", name, err)
	}
	if _, err := testutil.RunGit(t, dir, "commit", "-m", message); err != nil {
		t.Fatalf("Failed to commit %s: %v", name, err)
	}
}

// resolveSharedConflict writes the agreed resolution for the shared.txt fixtures
// and stages it, so the finish can be continued.
func resolveSharedConflict(t *testing.T, dir string) {
	t.Helper()

	if err := testutil.WriteFile(t, dir, "shared.txt", "resolved"); err != nil {
		t.Fatalf("Failed to write resolution: %v", err)
	}
	if _, err := testutil.RunGit(t, dir, "add", "shared.txt"); err != nil {
		t.Fatalf("Failed to stage resolution: %v", err)
	}
}

// assertMergeConflictAt verifies that a merge conflict is in progress and that
// the persisted merge state stopped in the expected step and child branch.
// An empty expectedChild means the child branch is not checked.
func assertMergeConflictAt(t *testing.T, dir string, expectedStep string, expectedChild string) {
	t.Helper()

	if !testutil.IsMergeInProgress(t, dir) {
		t.Fatal("Expected a merge to be in progress")
	}
	state, err := testutil.LoadMergeState(t, dir)
	if err != nil {
		t.Fatalf("Failed to load merge state: %v", err)
	}
	if state.CurrentStep != expectedStep {
		t.Fatalf("Expected merge state step %q, got %q", expectedStep, state.CurrentStep)
	}
	if expectedChild != "" && state.CurrentChildBranch != expectedChild {
		t.Fatalf("Expected merge state child branch %q, got %q", expectedChild, state.CurrentChildBranch)
	}
}

// TestFinishReleaseLandsOnIntegrationBranch tests that a default-config release finish
// leaves the user on the integration branch 'develop' rather than the parent 'main'.
// Steps:
// 1. Sets up a test repository and initializes git-flow with defaults
// 2. Starts release/1.0.0 and commits a file
// 3. Finishes the release branch
// 4. Verifies HEAD is on develop
// 5. Verifies the release commit is reachable from main and the tag points at main
// 6. Verifies the release branch was deleted
func TestFinishReleaseLandsOnIntegrationBranch(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	if output, err := testutil.RunGitFlow(t, dir, "init", "--defaults"); err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}
	if output, err := testutil.RunGitFlow(t, dir, "release", "start", "1.0.0"); err != nil {
		t.Fatalf("Failed to start release: %v\nOutput: %s", err, output)
	}
	writeAndCommit(t, dir, "version.txt", "1.0.0", "Add version file")

	releaseSha, err := testutil.RunGit(t, dir, "rev-parse", "HEAD")
	if err != nil {
		t.Fatalf("Failed to resolve release commit: %v", err)
	}
	releaseSha = strings.TrimSpace(releaseSha)

	output, err := testutil.RunGitFlow(t, dir, "release", "finish", "1.0.0")
	if err != nil {
		t.Fatalf("Failed to finish release: %v\nOutput: %s", err, output)
	}

	// Assert the landing branch before anything else moves HEAD.
	if current := testutil.GetCurrentBranch(t, dir); current != "develop" {
		t.Errorf("Expected to be on develop after release finish, got %s", current)
	}

	// The release commit must be part of main's history, not merely its content.
	if _, err := testutil.RunGit(t, dir, "merge-base", "--is-ancestor", releaseSha, "main"); err != nil {
		t.Errorf("Expected release commit %s to be an ancestor of main: %v", releaseSha, err)
	}

	tagSha, err := testutil.RunGit(t, dir, "rev-parse", "1.0.0^{commit}")
	if err != nil {
		t.Fatalf("Failed to resolve tag 1.0.0: %v", err)
	}
	mainSha, err := testutil.RunGit(t, dir, "rev-parse", "main")
	if err != nil {
		t.Fatalf("Failed to resolve main: %v", err)
	}
	if strings.TrimSpace(tagSha) != strings.TrimSpace(mainSha) {
		t.Errorf("Expected tag 1.0.0 to point at main (%s), got %s", strings.TrimSpace(mainSha), strings.TrimSpace(tagSha))
	}

	if testutil.BranchExists(t, dir, "release/1.0.0") {
		t.Error("Expected release/1.0.0 to be deleted after finish")
	}
}

// TestFinishHotfixLandsOnIntegrationBranch tests that a default-config hotfix finish
// leaves the user on the integration branch 'develop'.
// Steps:
// 1. Sets up a test repository and initializes git-flow with defaults
// 2. Starts hotfix/1.0.1 and commits a file
// 3. Finishes the hotfix branch
// 4. Verifies HEAD is on develop and the hotfix branch was deleted
func TestFinishHotfixLandsOnIntegrationBranch(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	if output, err := testutil.RunGitFlow(t, dir, "init", "--defaults"); err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}
	if output, err := testutil.RunGitFlow(t, dir, "hotfix", "start", "1.0.1"); err != nil {
		t.Fatalf("Failed to start hotfix: %v\nOutput: %s", err, output)
	}
	writeAndCommit(t, dir, "hotfix.txt", "fix", "Add hotfix file")

	if output, err := testutil.RunGitFlow(t, dir, "hotfix", "finish", "1.0.1"); err != nil {
		t.Fatalf("Failed to finish hotfix: %v\nOutput: %s", err, output)
	}

	if current := testutil.GetCurrentBranch(t, dir); current != "develop" {
		t.Errorf("Expected to be on develop after hotfix finish, got %s", current)
	}
	if testutil.BranchExists(t, dir, "hotfix/1.0.1") {
		t.Error("Expected hotfix/1.0.1 to be deleted after finish")
	}
}

// TestFinishFeatureLandsOnDevelop tests that a default-config feature finish still
// leaves the user on develop, its parent, which has no auto-update children.
// Steps:
// 1. Sets up a test repository and initializes git-flow with defaults
// 2. Starts feature/f1 and commits a file
// 3. Finishes the feature branch
// 4. Verifies HEAD is on develop and the feature branch was deleted
func TestFinishFeatureLandsOnDevelop(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	if output, err := testutil.RunGitFlow(t, dir, "init", "--defaults"); err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}
	if output, err := testutil.RunGitFlow(t, dir, "feature", "start", "f1"); err != nil {
		t.Fatalf("Failed to start feature: %v\nOutput: %s", err, output)
	}
	writeAndCommit(t, dir, "f.txt", "feature", "Add feature file")

	if output, err := testutil.RunGitFlow(t, dir, "feature", "finish", "f1"); err != nil {
		t.Fatalf("Failed to finish feature: %v\nOutput: %s", err, output)
	}

	if current := testutil.GetCurrentBranch(t, dir); current != "develop" {
		t.Errorf("Expected to be on develop after feature finish, got %s", current)
	}
	if testutil.BranchExists(t, dir, "feature/f1") {
		t.Error("Expected feature/f1 to be deleted after finish")
	}
}

// TestFinishBugfixLandsOnDevelop tests that a default-config bugfix finish still
// leaves the user on develop.
// Steps:
// 1. Sets up a test repository and initializes git-flow with defaults
// 2. Starts bugfix/b1 and commits a file
// 3. Finishes the bugfix branch
// 4. Verifies HEAD is on develop and the bugfix branch was deleted
func TestFinishBugfixLandsOnDevelop(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	if output, err := testutil.RunGitFlow(t, dir, "init", "--defaults"); err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}
	if output, err := testutil.RunGitFlow(t, dir, "bugfix", "start", "b1"); err != nil {
		t.Fatalf("Failed to start bugfix: %v\nOutput: %s", err, output)
	}
	writeAndCommit(t, dir, "b.txt", "bugfix", "Add bugfix file")

	if output, err := testutil.RunGitFlow(t, dir, "bugfix", "finish", "b1"); err != nil {
		t.Fatalf("Failed to finish bugfix: %v\nOutput: %s", err, output)
	}

	if current := testutil.GetCurrentBranch(t, dir); current != "develop" {
		t.Errorf("Expected to be on develop after bugfix finish, got %s", current)
	}
	if testutil.BranchExists(t, dir, "bugfix/b1") {
		t.Error("Expected bugfix/b1 to be deleted after finish")
	}
}

// TestFinishLandsOnParentWithoutAutoUpdateChildren tests that a finish whose parent has
// no auto-update children leaves the user on the parent branch.
// Steps:
// 1. Sets up a test repository and initializes git-flow with the github preset
// 2. Starts feature/f1 and commits a file
// 3. Finishes the feature branch
// 4. Verifies HEAD is on main, the feature branch is gone and no develop branch exists
func TestFinishLandsOnParentWithoutAutoUpdateChildren(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	if output, err := testutil.RunGitFlow(t, dir, "init", "--preset=github"); err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}
	if output, err := testutil.RunGitFlow(t, dir, "feature", "start", "f1"); err != nil {
		t.Fatalf("Failed to start feature: %v\nOutput: %s", err, output)
	}
	writeAndCommit(t, dir, "f.txt", "feature", "Add feature file")

	if output, err := testutil.RunGitFlow(t, dir, "feature", "finish", "f1"); err != nil {
		t.Fatalf("Failed to finish feature: %v\nOutput: %s", err, output)
	}

	if current := testutil.GetCurrentBranch(t, dir); current != "main" {
		t.Errorf("Expected to be on main after feature finish, got %s", current)
	}
	if testutil.BranchExists(t, dir, "feature/f1") {
		t.Error("Expected feature/f1 to be deleted after finish")
	}
	if testutil.BranchExists(t, dir, "develop") {
		t.Error("Expected no develop branch in a github preset repository")
	}
}

// TestFinishLandsOnCustomIntegrationBranch tests that the landing branch is resolved from
// configuration rather than from a branch named 'develop'.
// Steps:
// 1. Sets up a test repository and initializes git-flow with defaults
// 2. Configures a custom topology: base trunk, auto-update base integration, topic rel
// 3. Starts rel/1.0.0 and commits a file
// 4. Finishes the rel branch
// 5. Verifies HEAD is on integration, not trunk and not develop
func TestFinishLandsOnCustomIntegrationBranch(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	if output, err := testutil.RunGitFlow(t, dir, "init", "--defaults"); err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}
	if output, err := testutil.RunGitFlow(t, dir, "config", "add", "base", "trunk"); err != nil {
		t.Fatalf("Failed to add base trunk: %v\nOutput: %s", err, output)
	}
	if output, err := testutil.RunGitFlow(t, dir, "config", "add", "base", "integration", "trunk", "--auto-update"); err != nil {
		t.Fatalf("Failed to add base integration: %v\nOutput: %s", err, output)
	}
	if output, err := testutil.RunGitFlow(t, dir, "config", "add", "topic", "rel", "trunk", "--starting-point", "integration", "--prefix", "rel/"); err != nil {
		t.Fatalf("Failed to add topic rel: %v\nOutput: %s", err, output)
	}
	if output, err := testutil.RunGitFlow(t, dir, "rel", "start", "1.0.0"); err != nil {
		t.Fatalf("Failed to start rel: %v\nOutput: %s", err, output)
	}
	writeAndCommit(t, dir, "r.txt", "rel", "Add rel file")

	if output, err := testutil.RunGitFlow(t, dir, "rel", "finish", "1.0.0"); err != nil {
		t.Fatalf("Failed to finish rel: %v\nOutput: %s", err, output)
	}

	if current := testutil.GetCurrentBranch(t, dir); current != "integration" {
		t.Errorf("Expected to be on integration after rel finish, got %s", current)
	}
	if testutil.BranchExists(t, dir, "rel/1.0.0") {
		t.Error("Expected rel/1.0.0 to be deleted after finish")
	}
}

// TestFinishWithKeepLandsOnIntegrationBranch tests that the landing rule is independent
// of whether the topic branch is deleted.
// Steps:
// 1. Sets up a test repository and initializes git-flow with defaults
// 2. Starts release/1.0.0 and commits a file
// 3. Finishes the release branch with --keep
// 4. Verifies HEAD is on develop and the release branch still exists
func TestFinishWithKeepLandsOnIntegrationBranch(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	if output, err := testutil.RunGitFlow(t, dir, "init", "--defaults"); err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}
	if output, err := testutil.RunGitFlow(t, dir, "release", "start", "1.0.0"); err != nil {
		t.Fatalf("Failed to start release: %v\nOutput: %s", err, output)
	}
	writeAndCommit(t, dir, "version.txt", "1.0.0", "Add version file")

	if output, err := testutil.RunGitFlow(t, dir, "release", "finish", "1.0.0", "--keep"); err != nil {
		t.Fatalf("Failed to finish release: %v\nOutput: %s", err, output)
	}

	if current := testutil.GetCurrentBranch(t, dir); current != "develop" {
		t.Errorf("Expected to be on develop after release finish --keep, got %s", current)
	}
	if !testutil.BranchExists(t, dir, "release/1.0.0") {
		t.Error("Expected release/1.0.0 to be kept with --keep")
	}
}

// TestFinishContinueAfterParentConflictLandsOnIntegrationBranch tests that continuing a
// finish after a parent-merge conflict lands on the integration branch.
// Steps:
// 1. Sets up a test repository with a conflicting merge into main
// 2. Runs release finish and verifies it stops in the merge step
// 3. Resolves and stages the conflict
// 4. Runs release finish --continue
// 5. Verifies HEAD is on develop and the release branch was deleted
func TestFinishContinueAfterParentConflictLandsOnIntegrationBranch(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	if output, err := testutil.RunGitFlow(t, dir, "init", "--defaults"); err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}
	setupParentMergeConflict(t, dir)

	if _, err := testutil.RunGitFlow(t, dir, "release", "finish", "1.0.0"); err == nil {
		t.Fatal("Expected release finish to fail with a merge conflict")
	}
	assertMergeConflictAt(t, dir, "merge", "")

	resolveSharedConflict(t, dir)

	if output, err := testutil.RunGitFlow(t, dir, "release", "finish", "1.0.0", "--continue"); err != nil {
		t.Fatalf("Failed to continue release finish: %v\nOutput: %s", err, output)
	}

	if testutil.IsMergeInProgress(t, dir) {
		t.Error("Expected no merge in progress after continue")
	}
	if current := testutil.GetCurrentBranch(t, dir); current != "develop" {
		t.Errorf("Expected to be on develop after continue, got %s", current)
	}
	if testutil.BranchExists(t, dir, "release/1.0.0") {
		t.Error("Expected release/1.0.0 to be deleted after finish")
	}
}

// TestFinishContinueAfterChildConflictLandsOnIntegrationBranch tests that continuing a
// finish after a child-update conflict lands on the integration branch.
// Steps:
// 1. Sets up a test repository where the develop auto-update conflicts
// 2. Runs release finish and verifies it stops while updating develop
// 3. Resolves and stages the conflict
// 4. Runs release finish --continue
// 5. Verifies HEAD is on develop, the release branch is gone and the tag exists
func TestFinishContinueAfterChildConflictLandsOnIntegrationBranch(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	if output, err := testutil.RunGitFlow(t, dir, "init", "--defaults"); err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}
	setupChildUpdateConflict(t, dir)

	if _, err := testutil.RunGitFlow(t, dir, "release", "finish", "1.0.0"); err == nil {
		t.Fatal("Expected release finish to fail while updating develop")
	}
	assertMergeConflictAt(t, dir, "update_children", "develop")

	resolveSharedConflict(t, dir)

	if output, err := testutil.RunGitFlow(t, dir, "release", "finish", "1.0.0", "--continue"); err != nil {
		t.Fatalf("Failed to continue release finish: %v\nOutput: %s", err, output)
	}

	if current := testutil.GetCurrentBranch(t, dir); current != "develop" {
		t.Errorf("Expected to be on develop after continue, got %s", current)
	}
	if testutil.BranchExists(t, dir, "release/1.0.0") {
		t.Error("Expected release/1.0.0 to be deleted after finish")
	}
	tags, err := testutil.RunGit(t, dir, "tag", "-l")
	if err != nil {
		t.Fatalf("Failed to list tags: %v", err)
	}
	if !strings.Contains(tags, "1.0.0") {
		t.Errorf("Expected tag 1.0.0 to exist, got tags: %s", tags)
	}
}

// TestFinishContinueWithRebaseLandsOnIntegrationBranch tests that continuing a finish that
// used the rebase strategy lands on the integration branch.
// Steps:
// 1. Sets up a test repository and configures the release type to rebase upstream
// 2. Provokes a parent-merge conflict and verifies the rebase state via .git/rebase-merge
// 3. Resolves and stages the conflict
// 4. Runs release finish --continue
// 5. Verifies HEAD is on develop and the release branch was deleted
func TestFinishContinueWithRebaseLandsOnIntegrationBranch(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	if output, err := testutil.RunGitFlow(t, dir, "init", "--defaults"); err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}
	if _, err := testutil.RunGit(t, dir, "config", "gitflow.branch.release.upstreamstrategy", "rebase"); err != nil {
		t.Fatalf("Failed to configure rebase strategy: %v", err)
	}
	setupParentMergeConflict(t, dir)

	if _, err := testutil.RunGitFlow(t, dir, "release", "finish", "1.0.0"); err == nil {
		t.Fatal("Expected release finish to fail with a rebase conflict")
	}

	// During a rebase conflict the branch name is unreliable; verify Git's internal state.
	rebaseMergeDir := filepath.Join(dir, ".git", "rebase-merge")
	if _, err := os.Stat(rebaseMergeDir); os.IsNotExist(err) {
		t.Fatal("Expected to be in a rebase conflict state")
	}
	headName, err := os.ReadFile(filepath.Join(rebaseMergeDir, "head-name"))
	if err != nil {
		t.Fatalf("Failed to read rebase head-name: %v", err)
	}
	if strings.TrimSpace(string(headName)) != "refs/heads/release/1.0.0" {
		t.Fatalf("Expected release/1.0.0 to be rebased, got %s", strings.TrimSpace(string(headName)))
	}
	state, err := testutil.LoadMergeState(t, dir)
	if err != nil {
		t.Fatalf("Failed to load merge state: %v", err)
	}
	if state.CurrentStep != "merge" {
		t.Fatalf("Expected merge state step \"merge\", got %q", state.CurrentStep)
	}

	resolveSharedConflict(t, dir)

	if output, err := testutil.RunGitFlow(t, dir, "release", "finish", "1.0.0", "--continue"); err != nil {
		t.Fatalf("Failed to continue release finish: %v\nOutput: %s", err, output)
	}

	if current := testutil.GetCurrentBranch(t, dir); current != "develop" {
		t.Errorf("Expected to be on develop after continue, got %s", current)
	}
	if testutil.BranchExists(t, dir, "release/1.0.0") {
		t.Error("Expected release/1.0.0 to be deleted after finish")
	}
}

// TestFinishAbortDuringParentMergeReturnsToTopicBranch tests that aborting during the parent
// merge still returns to the topic branch.
// Steps:
// 1. Sets up a test repository with a conflicting merge into main
// 2. Runs release finish and verifies it stops in the merge step
// 3. Runs release finish --abort
// 4. Verifies HEAD is on release/1.0.0, the branch still exists and no merge is in progress
func TestFinishAbortDuringParentMergeReturnsToTopicBranch(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	if output, err := testutil.RunGitFlow(t, dir, "init", "--defaults"); err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}
	setupParentMergeConflict(t, dir)

	if _, err := testutil.RunGitFlow(t, dir, "release", "finish", "1.0.0"); err == nil {
		t.Fatal("Expected release finish to fail with a merge conflict")
	}
	assertMergeConflictAt(t, dir, "merge", "")

	if output, err := testutil.RunGitFlow(t, dir, "release", "finish", "1.0.0", "--abort"); err != nil {
		t.Fatalf("Failed to abort release finish: %v\nOutput: %s", err, output)
	}

	if current := testutil.GetCurrentBranch(t, dir); current != "release/1.0.0" {
		t.Errorf("Expected to be on release/1.0.0 after abort, got %s", current)
	}
	if !testutil.BranchExists(t, dir, "release/1.0.0") {
		t.Error("Expected release/1.0.0 to still exist after abort")
	}
	if testutil.IsMergeInProgress(t, dir) {
		t.Error("Expected no merge in progress after abort")
	}
}

// TestFinishAbortDuringChildUpdateReturnsToTopicBranch tests that aborting once the parent
// merge already succeeded still returns to the topic branch.
// Steps:
// 1. Sets up a test repository where the develop auto-update conflicts
// 2. Runs release finish and verifies it stops while updating develop
// 3. Runs release finish --abort
// 4. Verifies HEAD is on release/1.0.0, the branch still exists and no merge is in progress
func TestFinishAbortDuringChildUpdateReturnsToTopicBranch(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	if output, err := testutil.RunGitFlow(t, dir, "init", "--defaults"); err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}
	setupChildUpdateConflict(t, dir)

	if _, err := testutil.RunGitFlow(t, dir, "release", "finish", "1.0.0"); err == nil {
		t.Fatal("Expected release finish to fail while updating develop")
	}
	assertMergeConflictAt(t, dir, "update_children", "develop")

	if output, err := testutil.RunGitFlow(t, dir, "release", "finish", "1.0.0", "--abort"); err != nil {
		t.Fatalf("Failed to abort release finish: %v\nOutput: %s", err, output)
	}

	if current := testutil.GetCurrentBranch(t, dir); current != "release/1.0.0" {
		t.Errorf("Expected to be on release/1.0.0 after abort, got %s", current)
	}
	if !testutil.BranchExists(t, dir, "release/1.0.0") {
		t.Error("Expected release/1.0.0 to still exist after abort")
	}
	if testutil.IsMergeInProgress(t, dir) {
		t.Error("Expected no merge in progress after abort")
	}
}

// TestFinishReportsLandingBranch tests that finish reports the branch it leaves the user on.
// Steps:
// 1. Sets up a test repository and initializes git-flow with defaults
// 2. Starts release/1.0.0 and commits a file
// 3. Finishes the release branch, capturing output
// 4. Verifies HEAD is on develop and the output names that same branch
func TestFinishReportsLandingBranch(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	if output, err := testutil.RunGitFlow(t, dir, "init", "--defaults"); err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}
	if output, err := testutil.RunGitFlow(t, dir, "release", "start", "1.0.0"); err != nil {
		t.Fatalf("Failed to start release: %v\nOutput: %s", err, output)
	}
	writeAndCommit(t, dir, "version.txt", "1.0.0", "Add version file")

	output, err := testutil.RunGitFlow(t, dir, "release", "finish", "1.0.0")
	if err != nil {
		t.Fatalf("Failed to finish release: %v\nOutput: %s", err, output)
	}

	current := testutil.GetCurrentBranch(t, dir)
	if current != "develop" {
		t.Errorf("Expected to be on develop after release finish, got %s", current)
	}
	expected := fmt.Sprintf("You are now on branch '%s'", current)
	if !strings.Contains(output, expected) {
		t.Errorf("Expected output to contain %q, got: %s", expected, output)
	}
}

// TestFinishReportsLandingBranchWithoutChildren tests that finish reports the landing branch
// even when HEAD did not move.
// Steps:
// 1. Sets up a test repository and initializes git-flow with the github preset
// 2. Starts feature/f1 and commits a file
// 3. Finishes the feature branch, capturing output
// 4. Verifies HEAD is on main and the output names that same branch
func TestFinishReportsLandingBranchWithoutChildren(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	if output, err := testutil.RunGitFlow(t, dir, "init", "--preset=github"); err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}
	if output, err := testutil.RunGitFlow(t, dir, "feature", "start", "f1"); err != nil {
		t.Fatalf("Failed to start feature: %v\nOutput: %s", err, output)
	}
	writeAndCommit(t, dir, "f.txt", "feature", "Add feature file")

	output, err := testutil.RunGitFlow(t, dir, "feature", "finish", "f1")
	if err != nil {
		t.Fatalf("Failed to finish feature: %v\nOutput: %s", err, output)
	}

	current := testutil.GetCurrentBranch(t, dir)
	if current != "main" {
		t.Errorf("Expected to be on main after feature finish, got %s", current)
	}
	expected := fmt.Sprintf("You are now on branch '%s'", current)
	if !strings.Contains(output, expected) {
		t.Errorf("Expected output to contain %q, got: %s", expected, output)
	}
}

// TestFinishFromUnrelatedBranchLandsOnIntegrationBranch tests that the branch finish was
// invoked from does not affect the landing branch.
// Steps:
// 1. Sets up a test repository and initializes git-flow with defaults
// 2. Starts release/1.0.0, commits a file and checks out an unrelated branch
// 3. Finishes the release branch by name
// 4. Verifies HEAD is on develop and the unrelated branch still exists
func TestFinishFromUnrelatedBranchLandsOnIntegrationBranch(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	if output, err := testutil.RunGitFlow(t, dir, "init", "--defaults"); err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}
	if output, err := testutil.RunGitFlow(t, dir, "release", "start", "1.0.0"); err != nil {
		t.Fatalf("Failed to start release: %v\nOutput: %s", err, output)
	}
	writeAndCommit(t, dir, "version.txt", "1.0.0", "Add version file")

	if _, err := testutil.RunGit(t, dir, "checkout", "-b", "unrelated", "main"); err != nil {
		t.Fatalf("Failed to create unrelated branch: %v", err)
	}

	if output, err := testutil.RunGitFlow(t, dir, "release", "finish", "1.0.0"); err != nil {
		t.Fatalf("Failed to finish release: %v\nOutput: %s", err, output)
	}

	if current := testutil.GetCurrentBranch(t, dir); current != "develop" {
		t.Errorf("Expected to be on develop after release finish, got %s", current)
	}
	if !testutil.BranchExists(t, dir, "unrelated") {
		t.Error("Expected the unrelated branch to still exist")
	}
}

// TestFinishShorthandLandsOnIntegrationBranch tests that the shorthand 'git flow finish'
// lands on the same branch as the explicit form.
// Steps:
// 1. Sets up a test repository and initializes git-flow with defaults
// 2. Starts release/1.0.0 and commits a file, staying on the release branch
// 3. Runs git flow finish without a type or name
// 4. Verifies HEAD is on develop and the release branch was deleted
func TestFinishShorthandLandsOnIntegrationBranch(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	if output, err := testutil.RunGitFlow(t, dir, "init", "--defaults"); err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}
	if output, err := testutil.RunGitFlow(t, dir, "release", "start", "1.0.0"); err != nil {
		t.Fatalf("Failed to start release: %v\nOutput: %s", err, output)
	}
	writeAndCommit(t, dir, "version.txt", "1.0.0", "Add version file")

	if output, err := testutil.RunGitFlow(t, dir, "finish"); err != nil {
		t.Fatalf("Failed to finish via shorthand: %v\nOutput: %s", err, output)
	}

	if current := testutil.GetCurrentBranch(t, dir); current != "develop" {
		t.Errorf("Expected to be on develop after shorthand finish, got %s", current)
	}
	if testutil.BranchExists(t, dir, "release/1.0.0") {
		t.Error("Expected release/1.0.0 to be deleted after finish")
	}
}

// TestFinishSquashStrategyLandsOnIntegrationBranch tests that the squash strategy lands on
// the integration branch.
// Steps:
// 1. Sets up a test repository and configures the release type to squash upstream
// 2. Starts release/1.0.0 and commits a file
// 3. Finishes the release branch
// 4. Verifies HEAD is on develop and the release branch was deleted
func TestFinishSquashStrategyLandsOnIntegrationBranch(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	if output, err := testutil.RunGitFlow(t, dir, "init", "--defaults"); err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}
	if _, err := testutil.RunGit(t, dir, "config", "gitflow.branch.release.upstreamstrategy", "squash"); err != nil {
		t.Fatalf("Failed to configure squash strategy: %v", err)
	}
	if output, err := testutil.RunGitFlow(t, dir, "release", "start", "1.0.0"); err != nil {
		t.Fatalf("Failed to start release: %v\nOutput: %s", err, output)
	}
	writeAndCommit(t, dir, "version.txt", "1.0.0", "Add version file")

	if output, err := testutil.RunGitFlow(t, dir, "release", "finish", "1.0.0"); err != nil {
		t.Fatalf("Failed to finish release: %v\nOutput: %s", err, output)
	}

	if current := testutil.GetCurrentBranch(t, dir); current != "develop" {
		t.Errorf("Expected to be on develop after squash finish, got %s", current)
	}
	if testutil.BranchExists(t, dir, "release/1.0.0") {
		t.Error("Expected release/1.0.0 to be deleted after finish")
	}
}

// TestFinishRebaseStrategyLandsOnIntegrationBranch tests that the rebase strategy lands on
// the integration branch.
// Steps:
// 1. Sets up a test repository and configures the release type to rebase upstream
// 2. Starts release/1.0.0 and commits a file
// 3. Finishes the release branch
// 4. Verifies HEAD is on develop and the release branch was deleted
func TestFinishRebaseStrategyLandsOnIntegrationBranch(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	if output, err := testutil.RunGitFlow(t, dir, "init", "--defaults"); err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}
	if _, err := testutil.RunGit(t, dir, "config", "gitflow.branch.release.upstreamstrategy", "rebase"); err != nil {
		t.Fatalf("Failed to configure rebase strategy: %v", err)
	}
	if output, err := testutil.RunGitFlow(t, dir, "release", "start", "1.0.0"); err != nil {
		t.Fatalf("Failed to start release: %v\nOutput: %s", err, output)
	}
	writeAndCommit(t, dir, "version.txt", "1.0.0", "Add version file")

	if output, err := testutil.RunGitFlow(t, dir, "release", "finish", "1.0.0"); err != nil {
		t.Fatalf("Failed to finish release: %v\nOutput: %s", err, output)
	}

	if current := testutil.GetCurrentBranch(t, dir); current != "develop" {
		t.Errorf("Expected to be on develop after rebase finish, got %s", current)
	}
	if testutil.BranchExists(t, dir, "release/1.0.0") {
		t.Error("Expected release/1.0.0 to be deleted after finish")
	}
}

// TestFinishSupportLandsOnIntegrationBranch tests that a support finish follows the same
// landing rule, without any special-casing of release and hotfix.
// Steps:
// 1. Sets up a test repository and initializes git-flow with defaults
// 2. Starts support/1.x and commits a file
// 3. Finishes the support branch
// 4. Verifies HEAD is on develop, the auto-update child of main
func TestFinishSupportLandsOnIntegrationBranch(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	if output, err := testutil.RunGitFlow(t, dir, "init", "--defaults"); err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}
	if output, err := testutil.RunGitFlow(t, dir, "support", "start", "1.x"); err != nil {
		t.Fatalf("Failed to start support: %v\nOutput: %s", err, output)
	}
	writeAndCommit(t, dir, "s.txt", "support", "Add support file")

	if output, err := testutil.RunGitFlow(t, dir, "support", "finish", "1.x"); err != nil {
		t.Fatalf("Failed to finish support: %v\nOutput: %s", err, output)
	}

	if current := testutil.GetCurrentBranch(t, dir); current != "develop" {
		t.Errorf("Expected to be on develop after support finish, got %s", current)
	}
}

// TestFinishLandsOnLastAutoUpdateChild tests that with several auto-update children the
// landing branch is the last one the integration sequence processed.
// Steps:
// 1. Sets up a test repository and initializes git-flow with defaults
// 2. Adds 'staging' as a second auto-update base child of main
// 3. Starts release/1.0.0 and commits a file
// 4. Finishes the release branch, capturing output
// 5. Verifies HEAD is on staging and the last child-update line names staging
func TestFinishLandsOnLastAutoUpdateChild(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	if output, err := testutil.RunGitFlow(t, dir, "init", "--defaults"); err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}
	if _, err := testutil.RunGit(t, dir, "checkout", "-b", "staging", "main"); err != nil {
		t.Fatalf("Failed to create staging branch: %v", err)
	}
	for key, value := range map[string]string{
		"gitflow.branch.staging.type":               "base",
		"gitflow.branch.staging.parent":             "main",
		"gitflow.branch.staging.autoUpdate":         "true",
		"gitflow.branch.staging.downstreamStrategy": "merge",
	} {
		if _, err := testutil.RunGit(t, dir, "config", key, value); err != nil {
			t.Fatalf("Failed to set %s: %v", key, err)
		}
	}

	if output, err := testutil.RunGitFlow(t, dir, "release", "start", "1.0.0"); err != nil {
		t.Fatalf("Failed to start release: %v\nOutput: %s", err, output)
	}
	writeAndCommit(t, dir, "version.txt", "1.0.0", "Add version file")

	output, err := testutil.RunGitFlow(t, dir, "release", "finish", "1.0.0")
	if err != nil {
		t.Fatalf("Failed to finish release: %v\nOutput: %s", err, output)
	}

	if current := testutil.GetCurrentBranch(t, dir); current != "staging" {
		t.Errorf("Expected to be on staging, the last processed auto-update child, got %s", current)
	}

	// The last child actually processed is named by the last update line, which is
	// printed as each child is integrated (unlike the collection-time report).
	lastUpdated := ""
	for _, line := range strings.Split(output, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "Successfully updated branch '") {
			lastUpdated = strings.TrimSpace(line)
		}
	}
	if !strings.Contains(lastUpdated, "'staging'") {
		t.Errorf("Expected the last child update line to name staging, got: %q\nOutput: %s", lastUpdated, output)
	}
}
