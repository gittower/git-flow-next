package cmd_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gittower/git-flow-next/internal/errors"
	"github.com/gittower/git-flow-next/test/testutil"
)

// baseKeyFor returns the git config key recording the start point a topic branch
// was created from.
func baseKeyFor(branch string) string {
	return "gitflow.branch." + branch + ".base"
}

// TestRenameMovesWorktreeMarker verifies the worktree provenance marker follows a
// branch through a rename, so a git-flow-created worktree is not demoted to
// (unmanaged) by the rename.
// Steps:
//  1. Sets up a test repository and initializes git-flow with defaults
//  2. Runs 'git flow feature start old --worktree'
//  3. Verifies gitflow.worktree.feature/old.managed is 'true'
//  4. Runs 'git flow feature rename old new' from the main worktree
//  5. Verifies gitflow.worktree.feature/new.managed is 'true' and the old key is gone
//  6. Runs 'git flow worktree list' and verifies the row for feature/new shows the
//     worktree path and is not tagged (unmanaged)
func TestRenameMovesWorktreeMarker(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer os.RemoveAll(worktreeRootFor(dir))

	if out, err := testutil.RunGitFlow(t, dir, "feature", "start", "old", "--worktree"); err != nil {
		t.Fatalf("Failed to start feature with a worktree: %v\nOutput: %s", err, out)
	}
	if got := testutil.GitConfigValue(t, dir, managedMarkerFor("feature/old")); got != "true" {
		t.Fatalf("Precondition failed: expected the marker for feature/old to be 'true', got %q", got)
	}
	wtPath := computedWorktreePath(t, dir, "feature/old")

	if out, err := testutil.RunGitFlow(t, dir, "feature", "rename", "old", "new"); err != nil {
		t.Fatalf("Failed to rename the feature branch: %v\nOutput: %s", err, out)
	}

	if got := testutil.GitConfigValue(t, dir, managedMarkerFor("feature/new")); got != "true" {
		t.Errorf("Expected the marker for feature/new to be 'true', got %q", got)
	}
	if testutil.GitConfigExists(t, dir, managedMarkerFor("feature/old")) {
		t.Error("Expected the marker for feature/old to be removed")
	}

	output, err := testutil.RunGitFlow(t, dir, "worktree", "list")
	if err != nil {
		t.Fatalf("worktree list failed: %v\nOutput: %s", err, output)
	}
	rows := worktreeRows(output)
	if len(rows) != 1 {
		t.Fatalf("Expected exactly one worktree row, got %d: %s", len(rows), output)
	}
	if !strings.Contains(rows[0], wtPath) {
		t.Errorf("Expected the row to show the worktree path %q, got %q", wtPath, rows[0])
	}
	if strings.Contains(rows[0], "(unmanaged)") {
		t.Errorf("Expected the renamed branch's worktree to stay managed, got %q", rows[0])
	}
}

// TestRenameKeepsFeatureListWorktreeManaged verifies the bulk marker read behind
// '<type> list --worktrees' also sees the marker under the new branch name.
// Steps:
//  1. Sets up a test repository and initializes git-flow with defaults
//  2. Runs 'git flow feature start old --worktree'
//  3. Runs 'git flow feature rename old new'
//  4. Runs 'git flow feature list --worktrees'
//  5. Verifies the row for 'new' shows the worktree path and carries no
//     (unmanaged) tag
func TestRenameKeepsFeatureListWorktreeManaged(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer os.RemoveAll(worktreeRootFor(dir))

	if out, err := testutil.RunGitFlow(t, dir, "feature", "start", "old", "--worktree"); err != nil {
		t.Fatalf("Failed to start feature with a worktree: %v\nOutput: %s", err, out)
	}
	wtPath := computedWorktreePath(t, dir, "feature/old")

	if out, err := testutil.RunGitFlow(t, dir, "feature", "rename", "old", "new"); err != nil {
		t.Fatalf("Failed to rename the feature branch: %v\nOutput: %s", err, out)
	}

	output, err := testutil.RunGitFlow(t, dir, "feature", "list", "--worktrees")
	if err != nil {
		t.Fatalf("feature list --worktrees failed: %v\nOutput: %s", err, output)
	}

	cell := listCell(t, output, "new")
	if want := relCell(t, dir, wtPath); !strings.Contains(cell, want) {
		t.Errorf("Expected the worktree cell for 'new' to show %q, got %q", want, cell)
	}
	if strings.Contains(cell, "(unmanaged)") {
		t.Errorf("Expected the worktree cell for 'new' to carry no (unmanaged) tag, got %q", cell)
	}
}

// TestRenameMovedMarkerClearedByWorktreeRemove verifies the marker moved to the
// new branch name is still reachable by the cleanup path, so no provenance marker
// outlives the worktree it describes.
// Steps:
//  1. Sets up a test repository and initializes git-flow with defaults
//  2. Runs 'git flow feature start old --worktree'
//  3. Runs 'git flow feature rename old new'
//  4. Runs 'git flow worktree remove feature/new'
//  5. Verifies the worktree directory is gone
//  6. Verifies no gitflow.worktree.* key survives under either branch name
func TestRenameMovedMarkerClearedByWorktreeRemove(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer os.RemoveAll(worktreeRootFor(dir))

	if out, err := testutil.RunGitFlow(t, dir, "feature", "start", "old", "--worktree"); err != nil {
		t.Fatalf("Failed to start feature with a worktree: %v\nOutput: %s", err, out)
	}
	wtPath := computedWorktreePath(t, dir, "feature/old")

	if out, err := testutil.RunGitFlow(t, dir, "feature", "rename", "old", "new"); err != nil {
		t.Fatalf("Failed to rename the feature branch: %v\nOutput: %s", err, out)
	}

	output, err := testutil.RunGitFlow(t, dir, "worktree", "remove", "feature/new")
	if err != nil {
		t.Fatalf("worktree remove failed: %v\nOutput: %s", err, output)
	}

	if _, statErr := os.Stat(wtPath); !os.IsNotExist(statErr) {
		t.Errorf("Expected the worktree directory %q to be removed", wtPath)
	}
	if testutil.GitConfigHasPrefix(t, dir, "gitflow.worktree.") {
		t.Error("Expected no worktree provenance marker to survive the removal")
	}
}

// TestRenameMovesBaseKey verifies the recorded start point follows a branch
// through a rename, so the key does not outlive the branch it describes.
// Steps:
//  1. Sets up a test repository and initializes git-flow with defaults
//  2. Runs 'git flow feature start old develop'
//  3. Verifies gitflow.branch.feature/old.base is 'develop'
//  4. Runs 'git flow feature rename old new'
//  5. Verifies gitflow.branch.feature/new.base is 'develop' and the old key is gone
func TestRenameMovesBaseKey(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	if out, err := testutil.RunGitFlow(t, dir, "init", "--defaults"); err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, out)
	}
	if out, err := testutil.RunGitFlow(t, dir, "feature", "start", "old", "develop"); err != nil {
		t.Fatalf("Failed to start the feature branch: %v\nOutput: %s", err, out)
	}
	if got := testutil.GitConfigValue(t, dir, baseKeyFor("feature/old")); got != "develop" {
		t.Fatalf("Precondition failed: expected the base key for feature/old to be 'develop', got %q", got)
	}

	if out, err := testutil.RunGitFlow(t, dir, "feature", "rename", "old", "new"); err != nil {
		t.Fatalf("Failed to rename the feature branch: %v\nOutput: %s", err, out)
	}

	if got := testutil.GitConfigValue(t, dir, baseKeyFor("feature/new")); got != "develop" {
		t.Errorf("Expected the base key for feature/new to be 'develop', got %q", got)
	}
	if testutil.GitConfigExists(t, dir, baseKeyFor("feature/old")) {
		t.Error("Expected the base key for feature/old to be removed")
	}
}

// TestRenameHandMadeBranchCreatesNoKeys verifies renaming a branch that carries
// neither git-flow key invents nothing under the new name.
// Steps:
//  1. Sets up a test repository and initializes git-flow with defaults
//  2. Creates feature/old with plain 'git branch', so neither key exists
//  3. Runs 'git flow feature rename old new'
//  4. Verifies feature/new exists and feature/old does not
//  5. Verifies no gitflow.worktree.* key exists and no base key exists under
//     either branch name
func TestRenameHandMadeBranchCreatesNoKeys(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	if out, err := testutil.RunGitFlow(t, dir, "init", "--defaults"); err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, out)
	}
	if out, err := testutil.RunGit(t, dir, "branch", "feature/old", "develop"); err != nil {
		t.Fatalf("Failed to create feature/old by hand: %v\nOutput: %s", err, out)
	}

	if out, err := testutil.RunGitFlow(t, dir, "feature", "rename", "old", "new"); err != nil {
		t.Fatalf("Failed to rename the feature branch: %v\nOutput: %s", err, out)
	}

	if !testutil.BranchExists(t, dir, "feature/new") {
		t.Error("Expected feature/new to exist after the rename")
	}
	if testutil.BranchExists(t, dir, "feature/old") {
		t.Error("Expected feature/old to be gone after the rename")
	}
	if testutil.GitConfigHasPrefix(t, dir, "gitflow.worktree.") {
		t.Error("Expected the rename to invent no worktree provenance marker")
	}
	if testutil.GitConfigExists(t, dir, baseKeyFor("feature/old")) {
		t.Error("Expected no base key for feature/old")
	}
	if testutil.GitConfigExists(t, dir, baseKeyFor("feature/new")) {
		t.Error("Expected the rename to invent no base key for feature/new")
	}
}

// TestRenameHandMadeWorktreeStaysUnmanaged verifies renaming a branch whose
// worktree the user created by hand does not fabricate a provenance marker.
// Steps:
//  1. Sets up a test repository and initializes git-flow with defaults
//  2. Creates feature/old with plain 'git branch' so it can take a linked worktree
//  3. Adds a worktree for it with plain 'git worktree add', writing no marker
//  4. Runs 'git flow feature rename old new'
//  5. Runs 'git flow worktree list' and verifies the row for feature/new is still
//     tagged (unmanaged)
//  6. Verifies no gitflow.worktree.* key was created
func TestRenameHandMadeWorktreeStaysUnmanaged(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer os.RemoveAll(worktreeRootFor(dir))

	createFreeBranch(t, dir, "feature/old")
	handmade := computedWorktreePath(t, dir, "feature/old")
	if out, err := testutil.RunGit(t, dir, "worktree", "add", handmade, "feature/old"); err != nil {
		t.Fatalf("Failed to add a worktree by hand: %v\nOutput: %s", err, out)
	}

	if out, err := testutil.RunGitFlow(t, dir, "feature", "rename", "old", "new"); err != nil {
		t.Fatalf("Failed to rename the feature branch: %v\nOutput: %s", err, out)
	}

	output, err := testutil.RunGitFlow(t, dir, "worktree", "list")
	if err != nil {
		t.Fatalf("worktree list failed: %v\nOutput: %s", err, output)
	}
	rows := worktreeRows(output)
	if len(rows) != 1 {
		t.Fatalf("Expected exactly one worktree row, got %d: %s", len(rows), output)
	}
	if !strings.HasPrefix(rows[0], "feature/new") {
		t.Errorf("Expected the row to be for feature/new, got %q", rows[0])
	}
	if !strings.HasSuffix(rows[0], "(unmanaged)") {
		t.Errorf("Expected a hand-made worktree to stay (unmanaged) after the rename, got %q", rows[0])
	}
	if testutil.GitConfigHasPrefix(t, dir, "gitflow.worktree.") {
		t.Error("Expected the rename to create no worktree provenance marker")
	}
}

// TestRenamePreservesHandWrittenFalseMarker verifies the marker's value is moved
// raw, so a hand-written 'false' is not normalized into a claim that git-flow
// created the worktree.
// Steps:
//  1. Sets up a test repository and initializes git-flow with defaults
//  2. Runs 'git flow feature start old --worktree'
//  3. Overwrites gitflow.worktree.feature/old.managed with 'false'
//  4. Runs 'git flow feature rename old new'
//  5. Verifies gitflow.worktree.feature/new.managed is 'false' and the old key is gone
//  6. Runs 'git flow worktree list' and verifies the row is still tagged (unmanaged)
func TestRenamePreservesHandWrittenFalseMarker(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer os.RemoveAll(worktreeRootFor(dir))

	if out, err := testutil.RunGitFlow(t, dir, "feature", "start", "old", "--worktree"); err != nil {
		t.Fatalf("Failed to start feature with a worktree: %v\nOutput: %s", err, out)
	}
	if out, err := testutil.RunGit(t, dir, "config", "--local", managedMarkerFor("feature/old"), "false"); err != nil {
		t.Fatalf("Failed to overwrite the marker: %v\nOutput: %s", err, out)
	}

	if out, err := testutil.RunGitFlow(t, dir, "feature", "rename", "old", "new"); err != nil {
		t.Fatalf("Failed to rename the feature branch: %v\nOutput: %s", err, out)
	}

	if got := testutil.GitConfigValue(t, dir, managedMarkerFor("feature/new")); got != "false" {
		t.Errorf("Expected the marker for feature/new to stay 'false', got %q", got)
	}
	if testutil.GitConfigExists(t, dir, managedMarkerFor("feature/old")) {
		t.Error("Expected the marker for feature/old to be removed")
	}

	output, err := testutil.RunGitFlow(t, dir, "worktree", "list")
	if err != nil {
		t.Fatalf("worktree list failed: %v\nOutput: %s", err, output)
	}
	rows := worktreeRows(output)
	if len(rows) != 1 {
		t.Fatalf("Expected exactly one worktree row, got %d: %s", len(rows), output)
	}
	if !strings.HasSuffix(rows[0], "(unmanaged)") {
		t.Errorf("Expected a 'false' marker to keep the row (unmanaged), got %q", rows[0])
	}
}

// TestRenameSourceReplacesStaleDestinationKeys verifies the source branch's values
// overwrite keys left behind under the destination name by an earlier branch.
// Steps:
//  1. Sets up a test repository and initializes git-flow with defaults
//  2. Runs 'git flow feature start old --worktree'
//  3. Sets the source marker to 'false' so it differs from the stale destination
//  4. Plants stale destination keys: marker 'true' and base 'main' for feature/new
//  5. Runs 'git flow feature rename old new'
//  6. Verifies the marker for feature/new is 'false' and its base is 'develop'
//  7. Verifies both old-name keys are gone
func TestRenameSourceReplacesStaleDestinationKeys(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer os.RemoveAll(worktreeRootFor(dir))

	if out, err := testutil.RunGitFlow(t, dir, "feature", "start", "old", "--worktree"); err != nil {
		t.Fatalf("Failed to start feature with a worktree: %v\nOutput: %s", err, out)
	}
	if out, err := testutil.RunGit(t, dir, "config", "--local", managedMarkerFor("feature/old"), "false"); err != nil {
		t.Fatalf("Failed to set the source marker: %v\nOutput: %s", err, out)
	}
	if out, err := testutil.RunGit(t, dir, "config", "--local", managedMarkerFor("feature/new"), "true"); err != nil {
		t.Fatalf("Failed to plant the stale destination marker: %v\nOutput: %s", err, out)
	}
	if out, err := testutil.RunGit(t, dir, "config", "--local", baseKeyFor("feature/new"), "main"); err != nil {
		t.Fatalf("Failed to plant the stale destination base key: %v\nOutput: %s", err, out)
	}

	if out, err := testutil.RunGitFlow(t, dir, "feature", "rename", "old", "new"); err != nil {
		t.Fatalf("Failed to rename the feature branch: %v\nOutput: %s", err, out)
	}

	if got := testutil.GitConfigValue(t, dir, managedMarkerFor("feature/new")); got != "false" {
		t.Errorf("Expected the source marker 'false' to replace the stale 'true', got %q", got)
	}
	if got := testutil.GitConfigValue(t, dir, baseKeyFor("feature/new")); got != "develop" {
		t.Errorf("Expected the source base 'develop' to replace the stale 'main', got %q", got)
	}
	if testutil.GitConfigExists(t, dir, managedMarkerFor("feature/old")) {
		t.Error("Expected the marker for feature/old to be removed")
	}
	if testutil.GitConfigExists(t, dir, baseKeyFor("feature/old")) {
		t.Error("Expected the base key for feature/old to be removed")
	}
}

// TestRenameAbsentSourceRemovesStaleDestinationKeys verifies the migration mirrors
// rather than copies: with no key under the old name, a stale key under the new
// name is removed instead of being silently inherited by the renamed branch.
// Steps:
//  1. Sets up a test repository and initializes git-flow with defaults
//  2. Creates feature/old with plain 'git branch', so it carries no git-flow keys
//  3. Plants stale destination keys: marker 'true' and base 'main' for feature/new
//  4. Runs 'git flow feature rename old new'
//  5. Verifies neither key exists for feature/new
//  6. Verifies neither key exists for feature/old
func TestRenameAbsentSourceRemovesStaleDestinationKeys(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	if out, err := testutil.RunGitFlow(t, dir, "init", "--defaults"); err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, out)
	}
	if out, err := testutil.RunGit(t, dir, "branch", "feature/old", "develop"); err != nil {
		t.Fatalf("Failed to create feature/old by hand: %v\nOutput: %s", err, out)
	}
	if out, err := testutil.RunGit(t, dir, "config", "--local", managedMarkerFor("feature/new"), "true"); err != nil {
		t.Fatalf("Failed to plant the stale destination marker: %v\nOutput: %s", err, out)
	}
	if out, err := testutil.RunGit(t, dir, "config", "--local", baseKeyFor("feature/new"), "main"); err != nil {
		t.Fatalf("Failed to plant the stale destination base key: %v\nOutput: %s", err, out)
	}

	if out, err := testutil.RunGitFlow(t, dir, "feature", "rename", "old", "new"); err != nil {
		t.Fatalf("Failed to rename the feature branch: %v\nOutput: %s", err, out)
	}

	if testutil.GitConfigExists(t, dir, managedMarkerFor("feature/new")) {
		t.Error("Expected the stale marker for feature/new to be removed, not inherited")
	}
	if testutil.GitConfigExists(t, dir, baseKeyFor("feature/new")) {
		t.Error("Expected the stale base key for feature/new to be removed, not inherited")
	}
	if testutil.GitConfigExists(t, dir, managedMarkerFor("feature/old")) {
		t.Error("Expected no marker for feature/old")
	}
	if testutil.GitConfigExists(t, dir, baseKeyFor("feature/old")) {
		t.Error("Expected no base key for feature/old")
	}
}

// TestRenameLeavesOtherBranchKeysUntouched verifies the migration is keyed on the
// renamed branch alone and never sweeps by prefix.
// Steps:
//  1. Sets up a test repository and initializes git-flow with defaults
//  2. Runs 'git flow feature start old --worktree' and 'git flow feature start other --worktree'
//  3. Runs 'git flow feature rename old new'
//  4. Verifies feature/other keeps its marker 'true' and its base 'develop'
//  5. Runs 'git flow worktree list' and verifies the feature/other row is not
//     tagged (unmanaged)
//  6. Verifies feature/new carries the migrated keys and feature/old carries none
func TestRenameLeavesOtherBranchKeysUntouched(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer os.RemoveAll(worktreeRootFor(dir))

	if out, err := testutil.RunGitFlow(t, dir, "feature", "start", "old", "--worktree"); err != nil {
		t.Fatalf("Failed to start feature 'old' with a worktree: %v\nOutput: %s", err, out)
	}
	if out, err := testutil.RunGitFlow(t, dir, "feature", "start", "other", "--worktree"); err != nil {
		t.Fatalf("Failed to start feature 'other' with a worktree: %v\nOutput: %s", err, out)
	}

	if out, err := testutil.RunGitFlow(t, dir, "feature", "rename", "old", "new"); err != nil {
		t.Fatalf("Failed to rename the feature branch: %v\nOutput: %s", err, out)
	}

	if got := testutil.GitConfigValue(t, dir, managedMarkerFor("feature/other")); got != "true" {
		t.Errorf("Expected the marker for feature/other to stay 'true', got %q", got)
	}
	if got := testutil.GitConfigValue(t, dir, baseKeyFor("feature/other")); got != "develop" {
		t.Errorf("Expected the base key for feature/other to stay 'develop', got %q", got)
	}

	output, err := testutil.RunGitFlow(t, dir, "worktree", "list")
	if err != nil {
		t.Fatalf("worktree list failed: %v\nOutput: %s", err, output)
	}
	var otherRow string
	for _, row := range worktreeRows(output) {
		if strings.HasPrefix(row, "feature/other ") {
			otherRow = row
		}
	}
	if otherRow == "" {
		t.Fatalf("Expected a worktree row for feature/other, got: %s", output)
	}
	if strings.Contains(otherRow, "(unmanaged)") {
		t.Errorf("Expected feature/other's worktree to stay managed, got %q", otherRow)
	}

	if got := testutil.GitConfigValue(t, dir, managedMarkerFor("feature/new")); got != "true" {
		t.Errorf("Expected the marker for feature/new to be 'true', got %q", got)
	}
	if got := testutil.GitConfigValue(t, dir, baseKeyFor("feature/new")); got != "develop" {
		t.Errorf("Expected the base key for feature/new to be 'develop', got %q", got)
	}
	if testutil.GitConfigExists(t, dir, managedMarkerFor("feature/old")) {
		t.Error("Expected the marker for feature/old to be removed")
	}
	if testutil.GitConfigExists(t, dir, baseKeyFor("feature/old")) {
		t.Error("Expected the base key for feature/old to be removed")
	}
}

// TestRenameToInvalidRefNameLeavesKeysUnderOldName verifies the migration runs only
// after the ref rename succeeds: a name git rejects leaves both keys where they were.
// Steps:
//  1. Sets up a test repository and initializes git-flow with defaults
//  2. Runs 'git flow feature start old --worktree'
//  3. Runs 'git flow feature rename old "bad name"', which git branch -m rejects
//  4. Verifies the command fails with the git-error exit code
//  5. Verifies the marker and base key are still under feature/old
//  6. Verifies no key was written under the rejected name
func TestRenameToInvalidRefNameLeavesKeysUnderOldName(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer os.RemoveAll(worktreeRootFor(dir))

	if out, err := testutil.RunGitFlow(t, dir, "feature", "start", "old", "--worktree"); err != nil {
		t.Fatalf("Failed to start feature with a worktree: %v\nOutput: %s", err, out)
	}

	output, err := testutil.RunGitFlow(t, dir, "feature", "rename", "old", "bad name")
	if err == nil {
		t.Fatalf("Expected the rename to fail for an invalid ref name, got: %s", output)
	}
	if got := worktreeExitCode(err); got != int(errors.ExitCodeGitError) {
		t.Errorf("Expected exit code %d, got %d\nOutput: %s", errors.ExitCodeGitError, got, output)
	}

	if got := testutil.GitConfigValue(t, dir, managedMarkerFor("feature/old")); got != "true" {
		t.Errorf("Expected the marker for feature/old to stay 'true', got %q", got)
	}
	if got := testutil.GitConfigValue(t, dir, baseKeyFor("feature/old")); got != "develop" {
		t.Errorf("Expected the base key for feature/old to stay 'develop', got %q", got)
	}
	if testutil.GitConfigExists(t, dir, managedMarkerFor("feature/bad name")) {
		t.Error("Expected no marker under the rejected branch name")
	}
	if testutil.GitConfigExists(t, dir, baseKeyFor("feature/bad name")) {
		t.Error("Expected no base key under the rejected branch name")
	}
}

// TestRenameFromInsideLinkedWorktree verifies a rename invoked from the branch's
// own linked worktree migrates the keys into the repository's local config.
// Steps:
//  1. Sets up a test repository and initializes git-flow with defaults
//  2. Runs 'git flow feature start old --worktree'
//  3. Runs 'git flow feature rename old new' with the worktree as working directory
//  4. Verifies the worktree's HEAD followed the branch to feature/new
//  5. Verifies the main repository's local config carries the marker and base key
//     under feature/new, with both old-name keys gone
func TestRenameFromInsideLinkedWorktree(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer os.RemoveAll(worktreeRootFor(dir))

	if out, err := testutil.RunGitFlow(t, dir, "feature", "start", "old", "--worktree"); err != nil {
		t.Fatalf("Failed to start feature with a worktree: %v\nOutput: %s", err, out)
	}
	wtPath := computedWorktreePath(t, dir, "feature/old")

	if out, err := testutil.RunGitFlow(t, wtPath, "feature", "rename", "old", "new"); err != nil {
		t.Fatalf("Failed to rename from inside the worktree: %v\nOutput: %s", err, out)
	}

	if got := testutil.GetCurrentBranch(t, wtPath); got != "feature/new" {
		t.Errorf("Expected the worktree to follow the branch to feature/new, got %q", got)
	}
	if got := testutil.GitConfigValue(t, dir, managedMarkerFor("feature/new")); got != "true" {
		t.Errorf("Expected the marker for feature/new to be 'true', got %q", got)
	}
	if got := testutil.GitConfigValue(t, dir, baseKeyFor("feature/new")); got != "develop" {
		t.Errorf("Expected the base key for feature/new to be 'develop', got %q", got)
	}
	if testutil.GitConfigExists(t, dir, managedMarkerFor("feature/old")) {
		t.Error("Expected the marker for feature/old to be removed")
	}
	if testutil.GitConfigExists(t, dir, baseKeyFor("feature/old")) {
		t.Error("Expected the base key for feature/old to be removed")
	}
}

// TestRenamePreservesDottedMixedCaseBranchNames verifies branch names carrying
// dots, slashes and mixed case round-trip through the migration at exact case.
// Git preserves a config subsection's case, so the exact-case hit paired with the
// lowercase miss proves the case was carried rather than folded.
// Steps:
//  1. Sets up a test repository and initializes git-flow with defaults
//  2. Runs 'git flow feature start A.b/Cd --worktree'
//  3. Verifies gitflow.branch.feature/A.b/Cd.base is 'develop' at exact case
//  4. Runs 'git flow feature rename A.b/Cd E.f/Gh'
//  5. Verifies both keys exist under feature/E.f/Gh at exact case
//  6. Verifies both exact-case old-name keys and both lowercased new-name keys are absent
func TestRenamePreservesDottedMixedCaseBranchNames(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer os.RemoveAll(worktreeRootFor(dir))

	if out, err := testutil.RunGitFlow(t, dir, "feature", "start", "A.b/Cd", "--worktree"); err != nil {
		t.Fatalf("Failed to start feature with a worktree: %v\nOutput: %s", err, out)
	}
	if got := testutil.GitConfigValue(t, dir, baseKeyFor("feature/A.b/Cd")); got != "develop" {
		t.Fatalf("Precondition failed: expected the base key for feature/A.b/Cd to be 'develop', got %q", got)
	}

	if out, err := testutil.RunGitFlow(t, dir, "feature", "rename", "A.b/Cd", "E.f/Gh"); err != nil {
		t.Fatalf("Failed to rename the feature branch: %v\nOutput: %s", err, out)
	}

	if got := testutil.GitConfigValue(t, dir, managedMarkerFor("feature/E.f/Gh")); got != "true" {
		t.Errorf("Expected the marker for feature/E.f/Gh to be 'true', got %q", got)
	}
	if got := testutil.GitConfigValue(t, dir, baseKeyFor("feature/E.f/Gh")); got != "develop" {
		t.Errorf("Expected the base key for feature/E.f/Gh to be 'develop', got %q", got)
	}
	if testutil.GitConfigExists(t, dir, managedMarkerFor("feature/A.b/Cd")) {
		t.Error("Expected the marker for feature/A.b/Cd to be removed")
	}
	if testutil.GitConfigExists(t, dir, baseKeyFor("feature/A.b/Cd")) {
		t.Error("Expected the base key for feature/A.b/Cd to be removed")
	}
	if testutil.GitConfigExists(t, dir, managedMarkerFor("feature/e.f/gh")) {
		t.Error("Expected no lowercased marker key: the branch name's case must be carried, not folded")
	}
	if testutil.GitConfigExists(t, dir, baseKeyFor("feature/e.f/gh")) {
		t.Error("Expected no lowercased base key: the branch name's case must be carried, not folded")
	}
}

// TestRenameThenDeleteClearsBaseKey verifies delete's base-key cleanup reaches the
// migrated key, so no base key outlives the branch after a rename.
// Steps:
//  1. Sets up a test repository and initializes git-flow with defaults
//  2. Runs 'git flow feature start old', leaving HEAD on feature/old
//  3. Runs 'git flow feature rename old new'
//  4. Checks out develop and runs 'git flow feature delete new'
//  5. Verifies no base key survives under either branch name
func TestRenameThenDeleteClearsBaseKey(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	if out, err := testutil.RunGitFlow(t, dir, "init", "--defaults"); err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, out)
	}
	if out, err := testutil.RunGitFlow(t, dir, "feature", "start", "old"); err != nil {
		t.Fatalf("Failed to start the feature branch: %v\nOutput: %s", err, out)
	}

	if out, err := testutil.RunGitFlow(t, dir, "feature", "rename", "old", "new"); err != nil {
		t.Fatalf("Failed to rename the feature branch: %v\nOutput: %s", err, out)
	}
	if out, err := testutil.RunGit(t, dir, "checkout", "develop"); err != nil {
		t.Fatalf("Failed to check out develop: %v\nOutput: %s", err, out)
	}
	if out, err := testutil.RunGitFlow(t, dir, "feature", "delete", "new"); err != nil {
		t.Fatalf("Failed to delete the renamed feature branch: %v\nOutput: %s", err, out)
	}

	if testutil.GitConfigExists(t, dir, baseKeyFor("feature/new")) {
		t.Error("Expected the base key for feature/new to be cleaned up by delete")
	}
	if testutil.GitConfigExists(t, dir, baseKeyFor("feature/old")) {
		t.Error("Expected no base key to survive under feature/old")
	}
}

// TestRenameIgnoresGlobalScopedBaseKey verifies the migration reads the base key
// from LOCAL scope only, so a global value is never copied down into the
// repository's config under the new branch name.
// Steps:
//  1. Isolates the global config to a temp file so the real global config is untouched
//  2. Sets up a test repository, initializes git-flow and starts feature/old
//  3. Removes the local base key for feature/old
//  4. Writes gitflow.branch.feature/old.base=main in the GLOBAL scope only
//  5. Verifies no local base key exists for feature/old
//  6. Runs 'git flow feature rename old new' with the isolated global config
//  7. Verifies no local base key exists for either branch name
func TestRenameIgnoresGlobalScopedBaseKey(t *testing.T) {
	t.Parallel()
	// The override is passed through each subprocess env (not the test process
	// env) so it stays scoped to this test and is safe under parallel execution.
	env := []string{"GIT_CONFIG_GLOBAL=" + filepath.Join(t.TempDir(), "global-gitconfig")}

	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	if out, err := testutil.RunGitFlowWithEnv(t, dir, env, "init", "--defaults"); err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, out)
	}
	if out, err := testutil.RunGitFlowWithEnv(t, dir, env, "feature", "start", "old"); err != nil {
		t.Fatalf("Failed to start the feature branch: %v\nOutput: %s", err, out)
	}
	if out, err := testutil.RunGit(t, dir, "config", "--local", "--unset", baseKeyFor("feature/old")); err != nil {
		t.Fatalf("Failed to remove the local base key: %v\nOutput: %s", err, out)
	}
	if out, err := testutil.RunGitWithEnv(t, dir, env, "config", "--global", baseKeyFor("feature/old"), "main"); err != nil {
		t.Fatalf("Failed to write the global base key: %v\nOutput: %s", err, out)
	}
	if testutil.GitConfigExists(t, dir, baseKeyFor("feature/old")) {
		t.Fatal("Precondition failed: expected no local base key for feature/old")
	}

	if out, err := testutil.RunGitFlowWithEnv(t, dir, env, "feature", "rename", "old", "new"); err != nil {
		t.Fatalf("Failed to rename the feature branch: %v\nOutput: %s", err, out)
	}

	if testutil.GitConfigExists(t, dir, baseKeyFor("feature/new")) {
		t.Errorf("Expected the global base value not to be copied into local config, got %q",
			testutil.GitConfigValue(t, dir, baseKeyFor("feature/new")))
	}
	if testutil.GitConfigExists(t, dir, baseKeyFor("feature/old")) {
		t.Error("Expected no local base key for feature/old")
	}
}

// TestRenameWarnsWhenStateMigrationFails verifies a key the migration cannot move
// is reported as a warning on stderr without failing the command or undoing the
// rename, and that one failing key does not stop the others from migrating.
// Steps:
//  1. Sets up a test repository and initializes git-flow with defaults
//  2. Runs 'git flow feature start old --worktree'
//  3. Adds a second value to gitflow.worktree.feature/old.managed, which the
//     migration refuses rather than guessing which value to move
//  4. Runs 'git flow feature rename old new', capturing stdout and stderr separately
//  5. Verifies the command exits 0 and stdout reports the rename
//  6. Verifies feature/new exists and feature/old does not
//  7. Verifies stderr carries a warning naming both marker keys and stdout does not
//  8. Verifies the refused key still has both values under feature/old and none
//     under feature/new
//  9. Verifies the base key still migrated to feature/new
func TestRenameWarnsWhenStateMigrationFails(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer os.RemoveAll(worktreeRootFor(dir))

	if out, err := testutil.RunGitFlow(t, dir, "feature", "start", "old", "--worktree"); err != nil {
		t.Fatalf("Failed to start feature with a worktree: %v\nOutput: %s", err, out)
	}
	if out, err := testutil.RunGit(t, dir, "config", "--local", "--add", managedMarkerFor("feature/old"), "true"); err != nil {
		t.Fatalf("Failed to make the marker multi-valued: %v\nOutput: %s", err, out)
	}

	stdout, stderr, err := testutil.RunGitFlowStreams(t, dir, "feature", "rename", "old", "new")
	if err != nil {
		t.Fatalf("Expected a failed key migration to be a warning, not a failure: %v\nStderr: %s", err, stderr)
	}

	if !strings.Contains(stdout, "Renamed branch 'feature/old' to 'feature/new'") {
		t.Errorf("Expected stdout to report the rename, got %q", stdout)
	}
	if !testutil.BranchExists(t, dir, "feature/new") {
		t.Error("Expected feature/new to exist: a failed migration must not roll back the rename")
	}
	if testutil.BranchExists(t, dir, "feature/old") {
		t.Error("Expected feature/old to be gone: a failed migration must not roll back the rename")
	}

	if !strings.Contains(stderr, "Warning") {
		t.Errorf("Expected a warning on stderr, got %q", stderr)
	}
	if !strings.Contains(stderr, managedMarkerFor("feature/old")) {
		t.Errorf("Expected the warning to name the source key %q, got %q", managedMarkerFor("feature/old"), stderr)
	}
	if !strings.Contains(stderr, managedMarkerFor("feature/new")) {
		t.Errorf("Expected the warning to name the destination key %q, got %q", managedMarkerFor("feature/new"), stderr)
	}
	if strings.Contains(stdout, "Warning") {
		t.Errorf("Expected the warning on stderr only, but stdout carried it: %q", stdout)
	}

	if values := testutil.GitConfigAll(t, dir, managedMarkerFor("feature/old")); len(values) != 2 {
		t.Errorf("Expected the refused key to keep both values under feature/old, got %v", values)
	}
	if testutil.GitConfigExists(t, dir, managedMarkerFor("feature/new")) {
		t.Error("Expected the refused key to write nothing under feature/new")
	}

	if got := testutil.GitConfigValue(t, dir, baseKeyFor("feature/new")); got != "develop" {
		t.Errorf("Expected the base key to migrate despite the marker failure, got %q", got)
	}
	if testutil.GitConfigExists(t, dir, baseKeyFor("feature/old")) {
		t.Error("Expected the base key for feature/old to be removed")
	}
}
