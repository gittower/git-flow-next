package cmd_test

import (
	"strings"
	"testing"

	"github.com/gittower/git-flow-next/internal/errors"
	"github.com/gittower/git-flow-next/test/testutil"
)

// This file covers issue #210: `finish --ff-only` (and gitflow.<type>.finish.ff-only)
// as a precondition on the topic/parent relationship. The gate passes when the parent
// is an ancestor of the topic (equality included) and fails whenever the parent carries
// any commit the topic lacks — both true divergence and the parent-ahead-only case that
// `git merge --ff-only` would call "already up to date".
//
// Every topology below has each side touch a *different* file, so a plain merge would
// succeed and only the fast-forward condition is under test.

// =============================================================================
// Shared fixtures
// =============================================================================

// setupFFOnlyRepo creates a temporary repository initialized with git-flow defaults.
// Callers keep their own `defer testutil.CleanupTestRepo(t, dir)`.
func setupFFOnlyRepo(t *testing.T) string {
	t.Helper()
	dir := testutil.SetupTestRepo(t)
	if out, err := testutil.RunGitFlow(t, dir, "init", "--defaults"); err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, out)
	}
	return dir
}

// commitFile writes, stages and commits a file on the currently checked out branch.
func commitFile(t *testing.T, dir, name, content, message string) {
	t.Helper()
	if err := testutil.WriteFile(t, dir, name, content); err != nil {
		t.Fatalf("Failed to write %s: %v", name, err)
	}
	if _, err := testutil.RunGit(t, dir, "add", name); err != nil {
		t.Fatalf("Failed to add %s: %v", name, err)
	}
	if _, err := testutil.RunGit(t, dir, "commit", "-m", message); err != nil {
		t.Fatalf("Failed to commit %s: %v", name, err)
	}
}

// checkoutBranch checks out an existing branch, failing the test on error.
func checkoutBranch(t *testing.T, dir, branch string) {
	t.Helper()
	if out, err := testutil.RunGit(t, dir, "checkout", branch); err != nil {
		t.Fatalf("Failed to checkout %s: %v\nOutput: %s", branch, err, out)
	}
}

// ffCapableRelease creates release/1.0.0 with one commit and leaves main untouched, so
// main can fast-forward to the release tip. HEAD ends on release/1.0.0. Returns the tip.
func ffCapableRelease(t *testing.T, dir string) string {
	t.Helper()
	if out, err := testutil.RunGitFlow(t, dir, "release", "start", "1.0.0"); err != nil {
		t.Fatalf("Failed to start release: %v\nOutput: %s", err, out)
	}
	commitFile(t, dir, "release.txt", "release content", "Add release.txt")
	return revParse(t, dir, "release/1.0.0")
}

// divergedRelease builds the ff-capable topology and then advances main with a commit
// the release branch lacks, producing a true divergence. HEAD ends on release/1.0.0.
func divergedRelease(t *testing.T, dir string) string {
	t.Helper()
	tip := ffCapableRelease(t, dir)
	checkoutBranch(t, dir, "main")
	commitFile(t, dir, "main.txt", "main content", "Add main.txt")
	checkoutBranch(t, dir, "release/1.0.0")
	return tip
}

// parentAheadOnlyRelease creates an empty release branch and then advances main, so the
// parent is strictly ahead of the topic. `git merge --ff-only` would report "already up
// to date" here; the gate must still reject it. HEAD ends on release/1.0.0.
func parentAheadOnlyRelease(t *testing.T, dir string) string {
	t.Helper()
	if out, err := testutil.RunGitFlow(t, dir, "release", "start", "1.0.0"); err != nil {
		t.Fatalf("Failed to start release: %v\nOutput: %s", err, out)
	}
	checkoutBranch(t, dir, "main")
	commitFile(t, dir, "main.txt", "main content", "Add main.txt")
	checkoutBranch(t, dir, "release/1.0.0")
	return revParse(t, dir, "release/1.0.0")
}

// repoSnapshot captures the observable local state of a repository. It deliberately
// excludes refs/remotes: a configured fetch may legitimately move remote-tracking refs
// before the gate fires.
type repoSnapshot struct {
	currentBranch string
	heads         string
	tags          string
	status        string
}

// snapshotRepo captures the current branch, all local heads, the tag list and the
// working-tree status.
func snapshotRepo(t *testing.T, dir string) repoSnapshot {
	t.Helper()
	heads, err := testutil.RunGit(t, dir, "for-each-ref", "--format=%(refname) %(objectname)", "refs/heads")
	if err != nil {
		t.Fatalf("Failed to list local heads: %v", err)
	}
	tags, err := testutil.RunGit(t, dir, "tag", "-l")
	if err != nil {
		t.Fatalf("Failed to list tags: %v", err)
	}
	status, err := testutil.RunGit(t, dir, "status", "--porcelain")
	if err != nil {
		t.Fatalf("Failed to read status: %v", err)
	}
	return repoSnapshot{
		currentBranch: testutil.GetCurrentBranch(t, dir),
		heads:         heads,
		tags:          tags,
		status:        status,
	}
}

// assertRepoUnchanged re-snapshots the repository and asserts nothing was mutated,
// including that no git-flow merge state file was written.
func assertRepoUnchanged(t *testing.T, dir string, before repoSnapshot) {
	t.Helper()
	after := snapshotRepo(t, dir)
	if after.currentBranch != before.currentBranch {
		t.Errorf("Expected current branch unchanged. Before: %q After: %q", before.currentBranch, after.currentBranch)
	}
	if after.heads != before.heads {
		t.Errorf("Expected local branch heads unchanged.\nBefore:\n%s\nAfter:\n%s", before.heads, after.heads)
	}
	if after.tags != before.tags {
		t.Errorf("Expected tag list unchanged. Before: %q After: %q", before.tags, after.tags)
	}
	if after.status != before.status {
		t.Errorf("Expected working tree status unchanged. Before: %q After: %q", before.status, after.status)
	}
	if testutil.GitFlowMergeStateExists(t, dir) {
		t.Error("Expected no git-flow merge state file after a rejected finish")
	}
}

// commitParentCount returns the number of parents of the tip commit of ref. A
// fast-forwarded branch tip keeps its single parent; a merge commit has two.
func commitParentCount(t *testing.T, dir, ref string) int {
	t.Helper()
	out, err := testutil.RunGit(t, dir, "rev-list", "--parents", "-n", "1", ref)
	if err != nil {
		t.Fatalf("Failed to read parents of %s: %v", ref, err)
	}
	return len(strings.Fields(strings.TrimSpace(out))) - 1
}

// fileOnBranch reports whether path exists in the tree of ref, without checking it out.
// A missing path is the only failure git may report here: any other error (an unknown
// ref, a broken repository) would masquerade as "absent" and silently satisfy a negative
// assertion, so it fails the test instead.
func fileOnBranch(t *testing.T, dir, ref, path string) bool {
	t.Helper()
	out, err := testutil.RunGit(t, dir, "cat-file", "-e", ref+":"+path)
	if err == nil {
		return true
	}
	if !strings.Contains(out, "does not exist") && !strings.Contains(out, "exists on disk, but not in") {
		t.Fatalf("Failed to look up %s:%s: %v\nOutput: %s", ref, path, err, out)
	}
	return false
}

// assertGateMessage asserts the output carries the ff-only gate error naming both
// branches, the condition, the guarantee and the remedy. Each substring is wrap-safe:
// none of them may span a line break in the rendered message.
func assertGateMessage(t *testing.T, output, parent, topic string) {
	t.Helper()
	heading := "cannot fast-forward '" + parent + "' to '" + topic + "'"
	// The condition must name which side carries the extra commits, not merely state
	// that some branch does — that association is what tells the user where to look.
	condition := "'" + parent + "' has commits that are not in '" + topic + "'"
	for _, want := range []string{
		heading,
		condition,
		"exactly the tested branch tip",
		"re-test",
	} {
		if !strings.Contains(output, want) {
			t.Errorf("Expected gate error to contain %q. Output: %s", want, output)
		}
	}
	if strings.Contains(output, "Error: Error:") {
		t.Errorf("Expected a single 'Error: ' prefix. Output: %s", output)
	}
}

// =============================================================================
// Scenario 1-24: spec scenarios
// =============================================================================

// TestFinishFFOnlyFlagFastForwardsParent tests that --ff-only fast-forwards the parent
// when the topic is a descendant of it (spec scenario 1).
// Steps:
// 1. Sets up a test repository and initializes git-flow with defaults
// 2. Creates release/1.0.0 with one commit, leaving main untouched
// 3. Runs 'git flow release finish 1.0.0 --ff-only'
// 4. Verifies the command succeeds and main equals the release tip
// 5. Verifies main's tip has a single parent (no merge commit was added to main)
// 6. Verifies the 1.0.0 tag points at that same commit
func TestFinishFFOnlyFlagFastForwardsParent(t *testing.T) {
	t.Parallel()
	dir := setupFFOnlyRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	tip := ffCapableRelease(t, dir)

	output, err := testutil.RunGitFlow(t, dir, "release", "finish", "1.0.0", "--ff-only")
	if err != nil {
		t.Fatalf("Expected finish to succeed: %v\nOutput: %s", err, output)
	}

	if got := revParse(t, dir, "main"); got != tip {
		t.Errorf("Expected main to equal the release tip %s, got %s", tip, got)
	}
	if got := commitParentCount(t, dir, "main"); got != 1 {
		t.Errorf("Expected main's tip to have exactly one parent (fast-forward), got %d", got)
	}
	if got := revParse(t, dir, "1.0.0^{commit}"); got != tip {
		t.Errorf("Expected tag 1.0.0 to point at %s, got %s", tip, got)
	}
}

// TestFinishFFOnlyFlagRejectsDivergedParent tests that --ff-only aborts when the parent
// has diverged from the topic (spec scenario 2).
// Steps:
// 1. Sets up a test repository and initializes git-flow with defaults
// 2. Creates release/1.0.0 with one commit and advances main with a different commit
// 3. Runs 'git flow release finish 1.0.0 --ff-only'
// 4. Verifies the command exits with the validation error code (6)
// 5. Verifies the error names both branches, the condition, the guarantee and the remedy
func TestFinishFFOnlyFlagRejectsDivergedParent(t *testing.T) {
	t.Parallel()
	dir := setupFFOnlyRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	divergedRelease(t, dir)

	output, err := testutil.RunGitFlow(t, dir, "release", "finish", "1.0.0", "--ff-only")
	if err == nil {
		t.Fatalf("Expected finish to fail on a diverged parent. Output: %s", output)
	}
	assertExitCode(t, err, errors.ExitCodeValidationError)
	assertGateMessage(t, output, "main", "release/1.0.0")
}

// TestFinishFFOnlyRejectsParentAheadOnly tests that --ff-only aborts when the parent is
// only ahead of the topic, the case `git merge --ff-only` would call "already up to
// date" (spec scenario 3).
// Steps:
// 1. Sets up a test repository and initializes git-flow with defaults
// 2. Creates an empty release/1.0.0 and then advances main with one commit
// 3. Runs 'git flow release finish 1.0.0 --ff-only'
// 4. Verifies the command exits with the validation error code (6)
// 5. Verifies the gate error names both branches
func TestFinishFFOnlyRejectsParentAheadOnly(t *testing.T) {
	t.Parallel()
	dir := setupFFOnlyRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	parentAheadOnlyRelease(t, dir)

	output, err := testutil.RunGitFlow(t, dir, "release", "finish", "1.0.0", "--ff-only")
	if err == nil {
		t.Fatalf("Expected finish to fail when the parent is ahead only. Output: %s", output)
	}
	assertExitCode(t, err, errors.ExitCodeValidationError)
	assertGateMessage(t, output, "main", "release/1.0.0")
}

// TestFinishFFOnlyFailureLeavesRepositoryUnchanged tests that a rejected --ff-only
// finish mutates nothing (spec scenario 4).
// Steps:
// 1. Sets up a test repository and initializes git-flow with defaults
// 2. Creates the diverged release topology and snapshots the repository
// 3. Runs 'git flow release finish 1.0.0 --ff-only'
// 4. Verifies exit code 6
// 5. Verifies branch heads, current branch, tags, status and merge state are unchanged
// 6. Verifies release/1.0.0 still exists
func TestFinishFFOnlyFailureLeavesRepositoryUnchanged(t *testing.T) {
	t.Parallel()
	dir := setupFFOnlyRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	divergedRelease(t, dir)
	before := snapshotRepo(t, dir)

	output, err := testutil.RunGitFlow(t, dir, "release", "finish", "1.0.0", "--ff-only")
	if err == nil {
		t.Fatalf("Expected finish to fail on a diverged parent. Output: %s", output)
	}
	assertExitCode(t, err, errors.ExitCodeValidationError)
	assertRepoUnchanged(t, dir, before)
	if !testutil.BranchExists(t, dir, "release/1.0.0") {
		t.Error("Expected release/1.0.0 to still exist after a rejected finish")
	}
}

// TestFinishFFOnlyFailureSkipsPreFinishHook tests that the gate fires before the
// pre-finish hook runs (spec scenario 5).
// Steps:
// 1. Sets up a test repository and initializes git-flow with defaults
// 2. Creates the diverged release topology
// 3. Installs a pre-flow-release-finish hook that touches a marker file
// 4. Runs 'git flow release finish 1.0.0 --ff-only'
// 5. Verifies exit code 6 and that the marker file was never created
func TestFinishFFOnlyFailureSkipsPreFinishHook(t *testing.T) {
	t.Parallel()
	dir := setupFFOnlyRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	divergedRelease(t, dir)
	createHookScript(t, dir, "pre-flow-release-finish", "#!/bin/sh\ntouch \"$(git rev-parse --show-toplevel)/hook-ran\"\nexit 0\n")

	output, err := testutil.RunGitFlow(t, dir, "release", "finish", "1.0.0", "--ff-only")
	if err == nil {
		t.Fatalf("Expected finish to fail on a diverged parent. Output: %s", output)
	}
	assertExitCode(t, err, errors.ExitCodeValidationError)
	if testutil.FileExists(t, dir, "hook-ran") {
		t.Error("Expected the pre-finish hook not to run when the ff-only gate rejects the finish")
	}
}

// TestFinishFFOnlyConfigFastForwardsParent tests that gitflow.release.finish.ff-only
// drives the gate without any flag (spec scenario 6).
// Steps:
// 1. Sets up a test repository and initializes git-flow with defaults
// 2. Sets gitflow.release.finish.ff-only true
// 3. Creates release/1.0.0 with one commit, leaving main untouched
// 4. Runs 'git flow release finish 1.0.0' with no flags
// 5. Verifies success, main equal to the release tip, and a single-parent tip
func TestFinishFFOnlyConfigFastForwardsParent(t *testing.T) {
	t.Parallel()
	dir := setupFFOnlyRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	if _, err := testutil.RunGit(t, dir, "config", "gitflow.release.finish.ff-only", "true"); err != nil {
		t.Fatalf("Failed to set ff-only config: %v", err)
	}
	tip := ffCapableRelease(t, dir)

	output, err := testutil.RunGitFlow(t, dir, "release", "finish", "1.0.0")
	if err != nil {
		t.Fatalf("Expected finish to succeed: %v\nOutput: %s", err, output)
	}
	if got := revParse(t, dir, "main"); got != tip {
		t.Errorf("Expected main to equal the release tip %s, got %s", tip, got)
	}
	if got := commitParentCount(t, dir, "main"); got != 1 {
		t.Errorf("Expected main's tip to have exactly one parent (fast-forward), got %d", got)
	}
}

// TestFinishFFOnlyConfigRejectsDivergedParent tests that the configured ff-only key
// aborts a finish whose parent has moved (spec scenario 7).
// Steps:
// 1. Sets up a test repository and initializes git-flow with defaults
// 2. Sets gitflow.release.finish.ff-only true
// 3. Creates the diverged release topology
// 4. Runs 'git flow release finish 1.0.0' with no flags
// 5. Verifies exit code 6
func TestFinishFFOnlyConfigRejectsDivergedParent(t *testing.T) {
	t.Parallel()
	dir := setupFFOnlyRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	if _, err := testutil.RunGit(t, dir, "config", "gitflow.release.finish.ff-only", "true"); err != nil {
		t.Fatalf("Failed to set ff-only config: %v", err)
	}
	divergedRelease(t, dir)

	output, err := testutil.RunGitFlow(t, dir, "release", "finish", "1.0.0")
	if err == nil {
		t.Fatalf("Expected finish to fail on a diverged parent. Output: %s", output)
	}
	assertExitCode(t, err, errors.ExitCodeValidationError)
}

// TestFinishFFFlagOverridesConfiguredFFOnly tests that a CLI --ff beats a configured
// ff-only (spec scenario 8).
// Steps:
// 1. Sets up a test repository and initializes git-flow with defaults
// 2. Sets gitflow.release.finish.ff-only true
// 3. Creates the diverged release topology
// 4. Runs 'git flow release finish 1.0.0 --ff'
// 5. Verifies success and that main's tip is a merge commit carrying both files
func TestFinishFFFlagOverridesConfiguredFFOnly(t *testing.T) {
	t.Parallel()
	dir := setupFFOnlyRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	if _, err := testutil.RunGit(t, dir, "config", "gitflow.release.finish.ff-only", "true"); err != nil {
		t.Fatalf("Failed to set ff-only config: %v", err)
	}
	divergedRelease(t, dir)

	output, err := testutil.RunGitFlow(t, dir, "release", "finish", "1.0.0", "--ff")
	if err != nil {
		t.Fatalf("Expected --ff to override the configured ff-only: %v\nOutput: %s", err, output)
	}
	if got := commitParentCount(t, dir, "main"); got != 2 {
		t.Errorf("Expected main's tip to be a merge commit (2 parents), got %d", got)
	}
	if !fileOnBranch(t, dir, "main", "release.txt") || !fileOnBranch(t, dir, "main", "main.txt") {
		t.Error("Expected main to carry both release.txt and main.txt after the merge")
	}
}

// TestFinishNoFFFlagOverridesConfiguredFFOnly tests that a CLI --no-ff beats a
// configured ff-only (spec scenario 9).
// Steps:
// 1. Sets up a test repository and initializes git-flow with defaults
// 2. Sets gitflow.release.finish.ff-only true
// 3. Creates the ff-capable release topology
// 4. Runs 'git flow release finish 1.0.0 --no-ff'
// 5. Verifies success and that main's tip is a merge commit distinct from the tip
func TestFinishNoFFFlagOverridesConfiguredFFOnly(t *testing.T) {
	t.Parallel()
	dir := setupFFOnlyRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	if _, err := testutil.RunGit(t, dir, "config", "gitflow.release.finish.ff-only", "true"); err != nil {
		t.Fatalf("Failed to set ff-only config: %v", err)
	}
	tip := ffCapableRelease(t, dir)

	output, err := testutil.RunGitFlow(t, dir, "release", "finish", "1.0.0", "--no-ff")
	if err != nil {
		t.Fatalf("Expected --no-ff to override the configured ff-only: %v\nOutput: %s", err, output)
	}
	if got := commitParentCount(t, dir, "main"); got != 2 {
		t.Errorf("Expected main's tip to be a merge commit (2 parents), got %d", got)
	}
	if got := revParse(t, dir, "main"); got == tip {
		t.Errorf("Expected main to advance past the release tip %s via a merge commit", tip)
	}
}

// TestFinishFFOnlyFlagOverridesConfiguredNoFF tests that a CLI --ff-only beats a
// configured no-ff (spec scenario 10).
// Steps:
// 1. Sets up a test repository and initializes git-flow with defaults
// 2. Sets gitflow.release.finish.no-ff true
// 3. Creates the ff-capable release topology
// 4. Runs 'git flow release finish 1.0.0 --ff-only'
// 5. Verifies success, main equal to the release tip, and a single-parent tip
func TestFinishFFOnlyFlagOverridesConfiguredNoFF(t *testing.T) {
	t.Parallel()
	dir := setupFFOnlyRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	if _, err := testutil.RunGit(t, dir, "config", "gitflow.release.finish.no-ff", "true"); err != nil {
		t.Fatalf("Failed to set no-ff config: %v", err)
	}
	tip := ffCapableRelease(t, dir)

	output, err := testutil.RunGitFlow(t, dir, "release", "finish", "1.0.0", "--ff-only")
	if err != nil {
		t.Fatalf("Expected --ff-only to override the configured no-ff: %v\nOutput: %s", err, output)
	}
	if got := revParse(t, dir, "main"); got != tip {
		t.Errorf("Expected main to equal the release tip %s, got %s", tip, got)
	}
	if got := commitParentCount(t, dir, "main"); got != 1 {
		t.Errorf("Expected main's tip to have exactly one parent (fast-forward), got %d", got)
	}
}

// TestFinishConfiguredFFOnlyWinsOverConfiguredNoFF tests that within Layer 2 the
// ff-only key beats the no-ff key (spec scenario 11).
// Steps:
// 1. Sets up a test repository and initializes git-flow with defaults
// 2. Sets both gitflow.release.finish.ff-only and gitflow.release.finish.no-ff to true
// 3. Creates the diverged release topology
// 4. Runs 'git flow release finish 1.0.0' with no flags
// 5. Verifies exit code 6 and that main never received release.txt
func TestFinishConfiguredFFOnlyWinsOverConfiguredNoFF(t *testing.T) {
	t.Parallel()
	dir := setupFFOnlyRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	if _, err := testutil.RunGit(t, dir, "config", "gitflow.release.finish.ff-only", "true"); err != nil {
		t.Fatalf("Failed to set ff-only config: %v", err)
	}
	if _, err := testutil.RunGit(t, dir, "config", "gitflow.release.finish.no-ff", "true"); err != nil {
		t.Fatalf("Failed to set no-ff config: %v", err)
	}
	divergedRelease(t, dir)

	output, err := testutil.RunGitFlow(t, dir, "release", "finish", "1.0.0")
	if err == nil {
		t.Fatalf("Expected the configured ff-only gate to fire. Output: %s", output)
	}
	assertExitCode(t, err, errors.ExitCodeValidationError)
	if fileOnBranch(t, dir, "main", "release.txt") {
		t.Error("Expected main not to have received release.txt")
	}
}

// TestFinishFFOnlyWithSquashFlagRejected tests that --ff-only and --squash conflict
// (spec scenario 12).
// Steps:
// 1. Sets up a test repository and initializes git-flow with defaults
// 2. Creates the ff-capable release topology and snapshots the repository
// 3. Runs 'git flow release finish 1.0.0 --ff-only --squash'
// 4. Verifies exit code 2 and that the error names both --ff-only and squash
// 5. Verifies the repository is unchanged
func TestFinishFFOnlyWithSquashFlagRejected(t *testing.T) {
	t.Parallel()
	dir := setupFFOnlyRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	ffCapableRelease(t, dir)
	before := snapshotRepo(t, dir)

	output, err := testutil.RunGitFlow(t, dir, "release", "finish", "1.0.0", "--ff-only", "--squash")
	if err == nil {
		t.Fatalf("Expected --ff-only --squash to be rejected. Output: %s", output)
	}
	assertExitCode(t, err, errors.ExitCodeInvalidInput)
	if !strings.Contains(output, "--ff-only") || !strings.Contains(output, "squash") {
		t.Errorf("Expected the error to name both --ff-only and squash. Output: %s", output)
	}
	assertRepoUnchanged(t, dir, before)
}

// TestFinishFFOnlyWithConfiguredSquashRejected tests that the conflict check reads the
// resolved strategy, not just the flags (spec scenario 13).
// Steps:
// 1. Sets up a test repository and initializes git-flow with defaults
// 2. Sets gitflow.release.finish.squash true
// 3. Creates the ff-capable release topology and snapshots the repository
// 4. Runs 'git flow release finish 1.0.0 --ff-only'
// 5. Verifies exit code 2, the message naming both options, and an unchanged repository
func TestFinishFFOnlyWithConfiguredSquashRejected(t *testing.T) {
	t.Parallel()
	dir := setupFFOnlyRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	if _, err := testutil.RunGit(t, dir, "config", "gitflow.release.finish.squash", "true"); err != nil {
		t.Fatalf("Failed to set squash config: %v", err)
	}
	ffCapableRelease(t, dir)
	before := snapshotRepo(t, dir)

	output, err := testutil.RunGitFlow(t, dir, "release", "finish", "1.0.0", "--ff-only")
	if err == nil {
		t.Fatalf("Expected --ff-only with a configured squash to be rejected. Output: %s", output)
	}
	assertExitCode(t, err, errors.ExitCodeInvalidInput)
	if !strings.Contains(output, "--ff-only") || !strings.Contains(output, "squash") {
		t.Errorf("Expected the error to name both --ff-only and squash. Output: %s", output)
	}
	assertRepoUnchanged(t, dir, before)
}

// TestFinishFFOnlyWithLayer1SquashStrategyRejected tests that the conflict check covers
// a squash strategy originating at Layer 1 (spec scenario 14).
// Steps:
// 1. Sets up a test repository and initializes git-flow with defaults
// 2. Sets gitflow.branch.release.upstreamStrategy squash
// 3. Creates the ff-capable release topology and snapshots the repository
// 4. Runs 'git flow release finish 1.0.0 --ff-only'
// 5. Verifies exit code 2, the message naming both options, and an unchanged repository
func TestFinishFFOnlyWithLayer1SquashStrategyRejected(t *testing.T) {
	t.Parallel()
	dir := setupFFOnlyRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	if _, err := testutil.RunGit(t, dir, "config", "gitflow.branch.release.upstreamStrategy", "squash"); err != nil {
		t.Fatalf("Failed to set upstreamStrategy config: %v", err)
	}
	ffCapableRelease(t, dir)
	before := snapshotRepo(t, dir)

	output, err := testutil.RunGitFlow(t, dir, "release", "finish", "1.0.0", "--ff-only")
	if err == nil {
		t.Fatalf("Expected --ff-only with a Layer 1 squash strategy to be rejected. Output: %s", output)
	}
	assertExitCode(t, err, errors.ExitCodeInvalidInput)
	if !strings.Contains(output, "--ff-only") || !strings.Contains(output, "squash") {
		t.Errorf("Expected the error to name both --ff-only and squash. Output: %s", output)
	}
	assertRepoUnchanged(t, dir, before)
}

// TestFinishFFOnlyWithNoFFFlagRejected tests that --no-ff and --ff-only conflict with
// exit code 2, not cobra's exit code 1 (spec scenario 15).
// Steps:
// 1. Sets up a test repository and initializes git-flow with defaults
// 2. Creates the ff-capable release topology and snapshots the repository
// 3. Runs 'git flow release finish 1.0.0 --no-ff --ff-only'
// 4. Verifies exit code 2 and that the error names both --no-ff and --ff-only
// 5. Verifies the repository is unchanged
func TestFinishFFOnlyWithNoFFFlagRejected(t *testing.T) {
	t.Parallel()
	dir := setupFFOnlyRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	ffCapableRelease(t, dir)
	before := snapshotRepo(t, dir)

	output, err := testutil.RunGitFlow(t, dir, "release", "finish", "1.0.0", "--no-ff", "--ff-only")
	if err == nil {
		t.Fatalf("Expected --no-ff --ff-only to be rejected. Output: %s", output)
	}
	assertExitCode(t, err, errors.ExitCodeInvalidInput)
	if !strings.Contains(output, "--no-ff") || !strings.Contains(output, "--ff-only") {
		t.Errorf("Expected the error to name both --no-ff and --ff-only. Output: %s", output)
	}
	assertRepoUnchanged(t, dir, before)
}

// TestFinishFFOnlyWithFFFlagRejected tests that --ff and --ff-only conflict (spec
// scenario 16).
// Steps:
// 1. Sets up a test repository and initializes git-flow with defaults
// 2. Creates the ff-capable release topology and snapshots the repository
// 3. Runs 'git flow release finish 1.0.0 --ff --ff-only'
// 4. Verifies exit code 2 and the full conflict message naming --ff-only and --ff
// 5. Verifies the repository is unchanged
func TestFinishFFOnlyWithFFFlagRejected(t *testing.T) {
	t.Parallel()
	dir := setupFFOnlyRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	ffCapableRelease(t, dir)
	before := snapshotRepo(t, dir)

	output, err := testutil.RunGitFlow(t, dir, "release", "finish", "1.0.0", "--ff", "--ff-only")
	if err == nil {
		t.Fatalf("Expected --ff --ff-only to be rejected. Output: %s", output)
	}
	assertExitCode(t, err, errors.ExitCodeInvalidInput)
	// Asserting on "--ff" alone would be satisfied by the "--ff-only" token itself, so
	// the whole conflict message is required.
	wantMessage := "cannot combine --ff-only with --ff: they are conflicting values of the same fast-forward setting"
	if !strings.Contains(output, wantMessage) {
		t.Errorf("Expected the error to be %q. Output: %s", wantMessage, output)
	}
	assertRepoUnchanged(t, dir, before)
}

// TestFinishFFOnlyWithRebaseFastForwardsParent tests that --ff-only combines with
// --rebase when the gate passes (spec scenario 17).
// Steps:
// 1. Sets up a test repository and initializes git-flow with defaults
// 2. Creates the ff-capable release topology
// 3. Runs 'git flow release finish 1.0.0 --ff-only --rebase'
// 4. Verifies success, main equal to the release tip, and a single-parent tip
func TestFinishFFOnlyWithRebaseFastForwardsParent(t *testing.T) {
	t.Parallel()
	dir := setupFFOnlyRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	tip := ffCapableRelease(t, dir)

	output, err := testutil.RunGitFlow(t, dir, "release", "finish", "1.0.0", "--ff-only", "--rebase")
	if err != nil {
		t.Fatalf("Expected finish to succeed: %v\nOutput: %s", err, output)
	}
	if got := revParse(t, dir, "main"); got != tip {
		t.Errorf("Expected main to equal the release tip %s, got %s", tip, got)
	}
	if got := commitParentCount(t, dir, "main"); got != 1 {
		t.Errorf("Expected main's tip to have exactly one parent (fast-forward), got %d", got)
	}
}

// TestFinishFFOnlyWithRebaseFailureLeavesTopicHistoryIntact tests that a rejected
// --ff-only --rebase finish never rewrites the topic branch (spec scenario 18).
// Steps:
// 1. Sets up a test repository and initializes git-flow with defaults
// 2. Creates the diverged release topology and captures 'git rev-list release/1.0.0'
// 3. Runs 'git flow release finish 1.0.0 --ff-only --rebase'
// 4. Verifies exit code 6 and an identical revision list afterwards
// 5. Verifies no rebase is in progress
func TestFinishFFOnlyWithRebaseFailureLeavesTopicHistoryIntact(t *testing.T) {
	t.Parallel()
	dir := setupFFOnlyRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	divergedRelease(t, dir)
	revsBefore, err := testutil.RunGit(t, dir, "rev-list", "release/1.0.0")
	if err != nil {
		t.Fatalf("Failed to capture release revision list: %v", err)
	}

	output, err := testutil.RunGitFlow(t, dir, "release", "finish", "1.0.0", "--ff-only", "--rebase")
	if err == nil {
		t.Fatalf("Expected finish to fail on a diverged parent. Output: %s", output)
	}
	assertExitCode(t, err, errors.ExitCodeValidationError)

	revsAfter, err := testutil.RunGit(t, dir, "rev-list", "release/1.0.0")
	if err != nil {
		t.Fatalf("Failed to re-read release revision list: %v", err)
	}
	if revsAfter != revsBefore {
		t.Errorf("Expected release/1.0.0 history untouched.\nBefore:\n%s\nAfter:\n%s", revsBefore, revsAfter)
	}
	if testutil.FileExists(t, dir, ".git/rebase-merge") || testutil.FileExists(t, dir, ".git/rebase-apply") {
		t.Error("Expected no rebase to be in progress after a rejected finish")
	}
}

// TestFinishFFOnlyDoesNotConstrainChildAutoUpdate tests that the gate applies to the
// upstream merge only, not the child back-merge (spec scenario 19).
// Steps:
// 1. Sets up a test repository and initializes git-flow with defaults
// 2. Creates the ff-capable release topology and adds a commit on develop
// 3. Runs 'git flow release finish 1.0.0 --ff-only'
// 4. Verifies success and main equal to the release tip
// 5. Verifies develop's tip is a merge commit carrying both release.txt and develop.txt
func TestFinishFFOnlyDoesNotConstrainChildAutoUpdate(t *testing.T) {
	t.Parallel()
	dir := setupFFOnlyRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	tip := ffCapableRelease(t, dir)
	checkoutBranch(t, dir, "develop")
	commitFile(t, dir, "develop.txt", "develop content", "Add develop.txt")
	checkoutBranch(t, dir, "release/1.0.0")

	output, err := testutil.RunGitFlow(t, dir, "release", "finish", "1.0.0", "--ff-only")
	if err != nil {
		t.Fatalf("Expected finish to succeed: %v\nOutput: %s", err, output)
	}
	if got := revParse(t, dir, "main"); got != tip {
		t.Errorf("Expected main to equal the release tip %s, got %s", tip, got)
	}
	if got := commitParentCount(t, dir, "develop"); got != 2 {
		t.Errorf("Expected develop's tip to be a merge commit (2 parents), got %d", got)
	}
	if !fileOnBranch(t, dir, "develop", "release.txt") || !fileOnBranch(t, dir, "develop", "develop.txt") {
		t.Error("Expected develop to carry both release.txt and develop.txt after the back-merge")
	}
}

// TestFinishFFOnlyRejectsDivergedFeatureParent tests that --ff-only is not
// release-specific (spec scenario 20).
// Steps:
// 1. Sets up a test repository and initializes git-flow with defaults
// 2. Creates feature/f1 with one commit and advances develop with a different commit
// 3. Runs 'git flow feature finish f1 --ff-only'
// 4. Verifies exit code 6 and a gate error naming develop and feature/f1
func TestFinishFFOnlyRejectsDivergedFeatureParent(t *testing.T) {
	t.Parallel()
	dir := setupFFOnlyRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	if out, err := testutil.RunGitFlow(t, dir, "feature", "start", "f1"); err != nil {
		t.Fatalf("Failed to start feature: %v\nOutput: %s", err, out)
	}
	commitFile(t, dir, "feature.txt", "feature content", "Add feature.txt")
	checkoutBranch(t, dir, "develop")
	commitFile(t, dir, "develop.txt", "develop content", "Add develop.txt")
	checkoutBranch(t, dir, "feature/f1")

	output, err := testutil.RunGitFlow(t, dir, "feature", "finish", "f1", "--ff-only")
	if err == nil {
		t.Fatalf("Expected finish to fail on a diverged develop. Output: %s", output)
	}
	assertExitCode(t, err, errors.ExitCodeValidationError)
	assertGateMessage(t, output, "develop", "feature/f1")
}

// TestFinishFFOnlyHotfixFastForwardsParent tests --ff-only on a hotfix branch (spec
// scenario 21).
// Steps:
// 1. Sets up a test repository and initializes git-flow with defaults
// 2. Creates hotfix/1.0.1 with one commit, leaving main untouched
// 3. Runs 'git flow hotfix finish 1.0.1 --ff-only'
// 4. Verifies success, main equal to the hotfix tip, and a single-parent tip
func TestFinishFFOnlyHotfixFastForwardsParent(t *testing.T) {
	t.Parallel()
	dir := setupFFOnlyRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	if out, err := testutil.RunGitFlow(t, dir, "hotfix", "start", "1.0.1"); err != nil {
		t.Fatalf("Failed to start hotfix: %v\nOutput: %s", err, out)
	}
	commitFile(t, dir, "hotfix.txt", "hotfix content", "Add hotfix.txt")
	tip := revParse(t, dir, "hotfix/1.0.1")

	output, err := testutil.RunGitFlow(t, dir, "hotfix", "finish", "1.0.1", "--ff-only")
	if err != nil {
		t.Fatalf("Expected finish to succeed: %v\nOutput: %s", err, output)
	}
	if got := revParse(t, dir, "main"); got != tip {
		t.Errorf("Expected main to equal the hotfix tip %s, got %s", tip, got)
	}
	if got := commitParentCount(t, dir, "main"); got != 1 {
		t.Errorf("Expected main's tip to have exactly one parent (fast-forward), got %d", got)
	}
}

// TestFinishFFOnlyConfigAppliesOnlyToItsBranchType tests that a release-scoped ff-only
// key does not leak into a feature finish (spec scenario 22).
// Steps:
// 1. Sets up a test repository and initializes git-flow with defaults
// 2. Sets gitflow.release.finish.ff-only true only
// 3. Creates feature/f1 with one commit and advances develop with a different commit
// 4. Runs 'git flow feature finish f1' with no flags
// 5. Verifies success and that develop's tip is a merge commit
func TestFinishFFOnlyConfigAppliesOnlyToItsBranchType(t *testing.T) {
	t.Parallel()
	dir := setupFFOnlyRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	if _, err := testutil.RunGit(t, dir, "config", "gitflow.release.finish.ff-only", "true"); err != nil {
		t.Fatalf("Failed to set ff-only config: %v", err)
	}
	if out, err := testutil.RunGitFlow(t, dir, "feature", "start", "f1"); err != nil {
		t.Fatalf("Failed to start feature: %v\nOutput: %s", err, out)
	}
	commitFile(t, dir, "feature.txt", "feature content", "Add feature.txt")
	checkoutBranch(t, dir, "develop")
	commitFile(t, dir, "develop.txt", "develop content", "Add develop.txt")
	checkoutBranch(t, dir, "feature/f1")

	output, err := testutil.RunGitFlow(t, dir, "feature", "finish", "f1")
	if err != nil {
		t.Fatalf("Expected the release ff-only key not to affect a feature finish: %v\nOutput: %s", err, output)
	}
	if got := commitParentCount(t, dir, "develop"); got != 2 {
		t.Errorf("Expected develop's tip to be a merge commit (2 parents), got %d", got)
	}
}

// TestFinishFFOnlyWithEqualBranchesSucceeds tests that the gate is satisfied when the
// topic and parent are the same commit (spec scenario 23).
// Steps:
// 1. Sets up a test repository and initializes git-flow with defaults
// 2. Creates an empty release/1.0.0 and leaves main untouched
// 3. Runs 'git flow release finish 1.0.0 --ff-only'
// 4. Verifies success and that main is unchanged from its pre-finish SHA
// 5. Verifies release/1.0.0 was deleted as usual
func TestFinishFFOnlyWithEqualBranchesSucceeds(t *testing.T) {
	t.Parallel()
	dir := setupFFOnlyRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	if out, err := testutil.RunGitFlow(t, dir, "release", "start", "1.0.0"); err != nil {
		t.Fatalf("Failed to start release: %v\nOutput: %s", err, out)
	}
	mainBefore := revParse(t, dir, "main")

	output, err := testutil.RunGitFlow(t, dir, "release", "finish", "1.0.0", "--ff-only")
	if err != nil {
		t.Fatalf("Expected finish to succeed with equal branches: %v\nOutput: %s", err, output)
	}
	if got := revParse(t, dir, "main"); got != mainBefore {
		t.Errorf("Expected main unchanged at %s, got %s", mainBefore, got)
	}
	if testutil.BranchExists(t, dir, "release/1.0.0") {
		t.Error("Expected release/1.0.0 to be deleted after finish")
	}
}

// TestShorthandFinishFFOnlyRejectsDivergedParent tests that the shorthand finish also
// builds the tri-state option (spec scenario 24).
// Steps:
// 1. Sets up a test repository and initializes git-flow with defaults
// 2. Creates the diverged release topology, leaving HEAD on release/1.0.0
// 3. Runs 'git flow finish --ff-only'
// 4. Verifies exit code 6 and the same gate error as the per-type command
func TestShorthandFinishFFOnlyRejectsDivergedParent(t *testing.T) {
	t.Parallel()
	dir := setupFFOnlyRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	divergedRelease(t, dir)

	output, err := testutil.RunGitFlow(t, dir, "finish", "--ff-only")
	if err == nil {
		t.Fatalf("Expected the shorthand finish to fail on a diverged parent. Output: %s", output)
	}
	assertExitCode(t, err, errors.ExitCodeValidationError)
	assertGateMessage(t, output, "main", "release/1.0.0")
}

// =============================================================================
// Additional scenarios (E1-E12)
// =============================================================================

// TestFinishFFOnlyContinueAfterChildConflictSucceeds tests that the gate does not
// re-fire from the --continue resume path (E1).
// Steps:
// 1. Sets up a test repository and initializes git-flow with defaults
// 2. Sets gitflow.release.finish.ff-only true
// 3. Creates release/1.0.0 with shared.txt and a conflicting shared.txt on develop
// 4. Runs 'git flow release finish 1.0.0', expecting a child-update conflict (exit 3)
// 5. Resolves the conflict and runs 'git flow release finish --continue 1.0.0'
// 6. Verifies the resume succeeds and develop's tip is a merge of main into develop
func TestFinishFFOnlyContinueAfterChildConflictSucceeds(t *testing.T) {
	t.Parallel()
	dir := setupFFOnlyRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	if _, err := testutil.RunGit(t, dir, "config", "gitflow.release.finish.ff-only", "true"); err != nil {
		t.Fatalf("Failed to set ff-only config: %v", err)
	}
	if out, err := testutil.RunGitFlow(t, dir, "release", "start", "1.0.0"); err != nil {
		t.Fatalf("Failed to start release: %v\nOutput: %s", err, out)
	}
	commitFile(t, dir, "shared.txt", "release content\n", "Add shared.txt on release")
	checkoutBranch(t, dir, "develop")
	commitFile(t, dir, "shared.txt", "develop content\n", "Add shared.txt on develop")
	checkoutBranch(t, dir, "release/1.0.0")

	output, err := testutil.RunGitFlow(t, dir, "release", "finish", "1.0.0")
	if err == nil {
		t.Fatalf("Expected the child auto-update to conflict. Output: %s", output)
	}
	assertExitCode(t, err, errors.ExitCodeGitError)
	if !strings.Contains(output, "Merge conflict detected") {
		t.Errorf("Expected a merge conflict error, not the ff-only gate. Output: %s", output)
	}
	if strings.Contains(output, "cannot fast-forward") {
		t.Errorf("Expected the ff-only gate to pass (main was untouched). Output: %s", output)
	}

	if err := testutil.WriteFile(t, dir, "shared.txt", "resolved content\n"); err != nil {
		t.Fatalf("Failed to resolve shared.txt: %v", err)
	}
	if _, err := testutil.RunGit(t, dir, "add", "shared.txt"); err != nil {
		t.Fatalf("Failed to stage the resolution: %v", err)
	}

	output, err = testutil.RunGitFlow(t, dir, "release", "finish", "--continue", "1.0.0")
	if err != nil {
		t.Fatalf("Expected the resume to succeed: %v\nOutput: %s", err, output)
	}
	if got := commitParentCount(t, dir, "develop"); got != 2 {
		t.Errorf("Expected develop's tip to be a merge commit (2 parents), got %d", got)
	}
	subject, err := testutil.RunGit(t, dir, "log", "-1", "--format=%s", "develop")
	if err != nil {
		t.Fatalf("Failed to read develop's tip subject: %v", err)
	}
	if !strings.Contains(subject, "Merge branch 'main'") {
		t.Errorf("Expected develop's tip to merge main, got subject %q", strings.TrimSpace(subject))
	}
}

// TestFinishFFOnlyWithMissingParentReportsBranchNotFound tests that a misconfigured
// parent keeps today's error rather than a raw merge-base failure (E2).
// Steps:
// 1. Sets up a test repository and initializes git-flow with defaults
// 2. Creates the ff-capable release topology
// 3. Sets gitflow.branch.release.parent to a nonexistent branch
// 4. Runs 'git flow release finish 1.0.0 --ff-only'
// 5. Verifies exit code 5 and an error naming the missing branch
func TestFinishFFOnlyWithMissingParentReportsBranchNotFound(t *testing.T) {
	t.Parallel()
	dir := setupFFOnlyRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	ffCapableRelease(t, dir)
	if _, err := testutil.RunGit(t, dir, "config", "gitflow.branch.release.parent", "nonexistent"); err != nil {
		t.Fatalf("Failed to set parent config: %v", err)
	}

	output, err := testutil.RunGitFlow(t, dir, "release", "finish", "1.0.0", "--ff-only")
	if err == nil {
		t.Fatalf("Expected finish to fail with a missing parent branch. Output: %s", output)
	}
	assertExitCode(t, err, errors.ExitCodeBranchNotFound)
	if !strings.Contains(output, "nonexistent") {
		t.Errorf("Expected the error to name the missing branch 'nonexistent'. Output: %s", output)
	}
}

// TestFinishFFOnlyWithForceStillEnforcesGate tests that --force does not bypass the
// gate (E3).
// Steps:
// 1. Sets up a test repository and initializes git-flow with defaults
// 2. Creates the diverged release topology
// 3. Runs 'git flow release finish 1.0.0 --ff-only --force'
// 4. Verifies exit code 6 — --force relaxes the sync preflight, not the gate
func TestFinishFFOnlyWithForceStillEnforcesGate(t *testing.T) {
	t.Parallel()
	dir := setupFFOnlyRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	divergedRelease(t, dir)

	output, err := testutil.RunGitFlow(t, dir, "release", "finish", "1.0.0", "--ff-only", "--force")
	if err == nil {
		t.Fatalf("Expected --force not to bypass the ff-only gate. Output: %s", output)
	}
	assertExitCode(t, err, errors.ExitCodeValidationError)
}

// TestFinishFFOnlyConfigFalseDoesNotGate tests that an explicit ff-only=false leaves the
// existing ff/no-ff resolution untouched (E4).
// Steps:
// 1. Sets up a test repository and initializes git-flow with defaults
// 2. Sets gitflow.release.finish.no-ff true and gitflow.release.finish.ff-only false
// 3. Creates the ff-capable release topology
// 4. Runs 'git flow release finish 1.0.0' with no flags
// 5. Verifies success and that the configured no-ff still produced a merge commit
func TestFinishFFOnlyConfigFalseDoesNotGate(t *testing.T) {
	t.Parallel()
	dir := setupFFOnlyRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	if _, err := testutil.RunGit(t, dir, "config", "gitflow.release.finish.no-ff", "true"); err != nil {
		t.Fatalf("Failed to set no-ff config: %v", err)
	}
	if _, err := testutil.RunGit(t, dir, "config", "gitflow.release.finish.ff-only", "false"); err != nil {
		t.Fatalf("Failed to set ff-only config: %v", err)
	}
	tip := ffCapableRelease(t, dir)

	output, err := testutil.RunGitFlow(t, dir, "release", "finish", "1.0.0")
	if err != nil {
		t.Fatalf("Expected finish to succeed: %v\nOutput: %s", err, output)
	}
	if got := commitParentCount(t, dir, "main"); got != 2 {
		t.Errorf("Expected the configured no-ff to still win (merge commit), got %d parent(s)", got)
	}
	if got := revParse(t, dir, "main"); got == tip {
		t.Errorf("Expected main to advance past the release tip %s via a merge commit", tip)
	}
}

// TestFinishFFOnlyWithNoSquashFlagAccepted tests that --no-squash clears a configured
// squash so --ff-only is accepted (E5).
// Steps:
// 1. Sets up a test repository and initializes git-flow with defaults
// 2. Sets gitflow.release.finish.squash true
// 3. Creates the ff-capable release topology
// 4. Runs 'git flow release finish 1.0.0 --ff-only --no-squash'
// 5. Verifies success, main equal to the release tip, and a single-parent tip
func TestFinishFFOnlyWithNoSquashFlagAccepted(t *testing.T) {
	t.Parallel()
	dir := setupFFOnlyRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	if _, err := testutil.RunGit(t, dir, "config", "gitflow.release.finish.squash", "true"); err != nil {
		t.Fatalf("Failed to set squash config: %v", err)
	}
	tip := ffCapableRelease(t, dir)

	output, err := testutil.RunGitFlow(t, dir, "release", "finish", "1.0.0", "--ff-only", "--no-squash")
	if err != nil {
		t.Fatalf("Expected --no-squash to clear the configured squash: %v\nOutput: %s", err, output)
	}
	if got := revParse(t, dir, "main"); got != tip {
		t.Errorf("Expected main to equal the release tip %s, got %s", tip, got)
	}
	if got := commitParentCount(t, dir, "main"); got != 1 {
		t.Errorf("Expected main's tip to have exactly one parent (fast-forward), got %d", got)
	}
}

// TestShorthandFinishFFOnlyFastForwardsParent tests the shorthand's success path (E6).
// Steps:
// 1. Sets up a test repository and initializes git-flow with defaults
// 2. Creates the ff-capable release topology, leaving HEAD on release/1.0.0
// 3. Runs 'git flow finish --ff-only'
// 4. Verifies success and main equal to the release tip
func TestShorthandFinishFFOnlyFastForwardsParent(t *testing.T) {
	t.Parallel()
	dir := setupFFOnlyRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	tip := ffCapableRelease(t, dir)

	output, err := testutil.RunGitFlow(t, dir, "finish", "--ff-only")
	if err != nil {
		t.Fatalf("Expected the shorthand finish to succeed: %v\nOutput: %s", err, output)
	}
	if got := revParse(t, dir, "main"); got != tip {
		t.Errorf("Expected main to equal the release tip %s, got %s", tip, got)
	}
}

// TestShorthandFinishFFOnlyFalseDoesNotGate tests that the shorthand reads --ff-only by
// value, so an explicit --ff-only=false leaves the gate inert just as the per-type
// surface does.
// Steps:
// 1. Sets up a test repository and initializes git-flow with defaults
// 2. Creates the diverged release topology, leaving HEAD on release/1.0.0
// 3. Runs 'git flow finish --ff-only=false'
// 4. Verifies success and that the ordinary merge produced a merge commit on main
func TestShorthandFinishFFOnlyFalseDoesNotGate(t *testing.T) {
	t.Parallel()
	dir := setupFFOnlyRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	tip := divergedRelease(t, dir)

	output, err := testutil.RunGitFlow(t, dir, "finish", "--ff-only=false")
	if err != nil {
		t.Fatalf("Expected the shorthand finish to succeed with --ff-only=false: %v\nOutput: %s", err, output)
	}
	if got := commitParentCount(t, dir, "main"); got != 2 {
		t.Errorf("Expected an ordinary merge commit on main, got %d parent(s)", got)
	}
	if got := revParse(t, dir, "main"); got == tip {
		t.Errorf("Expected main to advance past the release tip %s via a merge commit", tip)
	}
}

// TestFinishConfiguredFFOnlyWinsOverConfiguredFF tests that within Layer 2 the ff-only
// key beats the ff key (E9).
// Steps:
// 1. Sets up a test repository and initializes git-flow with defaults
// 2. Sets both gitflow.release.finish.ff-only and gitflow.release.finish.ff to true
// 3. Creates the diverged release topology
// 4. Runs 'git flow release finish 1.0.0' with no flags
// 5. Verifies exit code 6 and that main never received release.txt
func TestFinishConfiguredFFOnlyWinsOverConfiguredFF(t *testing.T) {
	t.Parallel()
	dir := setupFFOnlyRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	if _, err := testutil.RunGit(t, dir, "config", "gitflow.release.finish.ff-only", "true"); err != nil {
		t.Fatalf("Failed to set ff-only config: %v", err)
	}
	if _, err := testutil.RunGit(t, dir, "config", "gitflow.release.finish.ff", "true"); err != nil {
		t.Fatalf("Failed to set ff config: %v", err)
	}
	divergedRelease(t, dir)

	output, err := testutil.RunGitFlow(t, dir, "release", "finish", "1.0.0")
	if err == nil {
		t.Fatalf("Expected the configured ff-only gate to fire. Output: %s", output)
	}
	assertExitCode(t, err, errors.ExitCodeValidationError)
	if fileOnBranch(t, dir, "main", "release.txt") {
		t.Error("Expected main not to have received release.txt")
	}
}

// TestFinishFFOnlyRunsAfterParentSyncPreflight tests that the gate runs after the
// fetch/sync preflight, so the preflight failure is what the user sees (E10).
// Steps:
// 1. Sets up a git-flow-initialized test repository with a remote
// 2. Creates feature/f1, commits and pushes it
// 3. Advances origin/develop and local develop differently, diverging the parent
// 4. Runs 'git flow feature finish f1 --ff-only'
// 5. Verifies the parent-sync error (naming develop, mentioning diverged and --force)
// 6. Verifies the gate error is absent and develop and feature/f1 are untouched
func TestFinishFFOnlyRunsAfterParentSyncPreflight(t *testing.T) {
	t.Parallel()
	dir, remoteDir := testutil.SetupTestRepoWithRemote(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer testutil.CleanupTestRepo(t, remoteDir)

	if out, err := testutil.RunGitFlow(t, dir, "feature", "start", "f1"); err != nil {
		t.Fatalf("Failed to start feature: %v\nOutput: %s", err, out)
	}
	commitFeatureAndPush(t, dir, "feature.txt", "feature content", "Add feature file", "feature/f1")

	advanceRemoteDevelop(t, remoteDir, "remote1.txt", "r1", "Remote develop commit")
	commitOnLocalDevelop(t, dir, "feature/f1", "local-dev.txt", "l1", "Local develop commit")

	developBefore := revParse(t, dir, "develop")

	output, err := testutil.RunGitFlow(t, dir, "feature", "finish", "f1", "--ff-only")
	if err == nil {
		t.Fatalf("Expected finish to fail on a diverged parent remote. Output: %s", output)
	}
	if !strings.Contains(output, "diverged") || !strings.Contains(output, "develop") || !strings.Contains(output, "--force") {
		t.Errorf("Expected the parent-sync error naming develop, 'diverged' and --force. Output: %s", output)
	}
	if strings.Contains(output, "cannot fast-forward") {
		t.Errorf("Expected the sync preflight to abort before the ff-only gate. Output: %s", output)
	}
	if after := revParse(t, dir, "develop"); after != developBefore {
		t.Errorf("Expected develop unchanged. Before: %s After: %s", developBefore, after)
	}
	if !testutil.BranchExists(t, dir, "feature/f1") {
		t.Error("Expected feature/f1 to still exist after the aborted finish")
	}
}

// TestShorthandFinishFFOnlyWithNoFFRejected tests that the shorthand rejects conflicting
// fast-forward options too (E11).
// Steps:
// 1. Sets up a test repository and initializes git-flow with defaults
// 2. Creates the ff-capable release topology and snapshots the repository
// 3. Runs 'git flow finish --ff-only --no-ff'
// 4. Verifies exit code 2 and that the error names both options
// 5. Verifies the repository is unchanged
func TestShorthandFinishFFOnlyWithNoFFRejected(t *testing.T) {
	t.Parallel()
	dir := setupFFOnlyRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	ffCapableRelease(t, dir)
	before := snapshotRepo(t, dir)

	output, err := testutil.RunGitFlow(t, dir, "finish", "--ff-only", "--no-ff")
	if err == nil {
		t.Fatalf("Expected the shorthand to reject --ff-only --no-ff. Output: %s", output)
	}
	assertExitCode(t, err, errors.ExitCodeInvalidInput)
	if !strings.Contains(output, "--ff-only") || !strings.Contains(output, "--no-ff") {
		t.Errorf("Expected the error to name both --ff-only and --no-ff. Output: %s", output)
	}
	assertRepoUnchanged(t, dir, before)
}

// TestFinishFFAndNoFFFlagsStillResolveToNoFF tests that the pre-existing --ff --no-ff
// behavior is unchanged: only combinations involving --ff-only are newly rejected (E12).
// Steps:
// 1. Sets up a test repository and initializes git-flow with defaults
// 2. Creates the ff-capable release topology
// 3. Runs 'git flow release finish 1.0.0 --ff --no-ff'
// 4. Verifies success (not a usage error) and that no-ff still won
func TestFinishFFAndNoFFFlagsStillResolveToNoFF(t *testing.T) {
	t.Parallel()
	dir := setupFFOnlyRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	tip := ffCapableRelease(t, dir)

	output, err := testutil.RunGitFlow(t, dir, "release", "finish", "1.0.0", "--ff", "--no-ff")
	if err != nil {
		t.Fatalf("Expected --ff --no-ff to remain accepted: %v\nOutput: %s", err, output)
	}
	if got := commitParentCount(t, dir, "main"); got != 2 {
		t.Errorf("Expected no-ff to win (merge commit), got %d parent(s)", got)
	}
	if got := revParse(t, dir, "main"); got == tip {
		t.Errorf("Expected main to advance past the release tip %s via a merge commit", tip)
	}
}
