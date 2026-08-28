package internal_test

import (
	"testing"

	"github.com/gittower/git-flow-next/internal/git"
	"github.com/gittower/git-flow-next/test/testutil"
)

// TestGetConfigAllValues tests the GetConfigAllValues helper reads multi-value git config keys.
// Steps:
// 1. Sets up a test repository
// 2. Adds multiple values for a single git config key using 'git config --add'
// 3. Calls repo.GetConfigAllValues for that key
// 4. Verifies all values are returned in order
// 5. Calls repo.GetConfigAllValues for a non-existent key
// 6. Verifies empty result with no error
func TestGetConfigAllValues(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Set multiple values for a single key
	_, err := testutil.RunGit(t, dir, "config", "test.multivalue", "first")
	if err != nil {
		t.Fatalf("Failed to set first config value: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "config", "--add", "test.multivalue", "second")
	if err != nil {
		t.Fatalf("Failed to add second config value: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "config", "--add", "test.multivalue", "third")
	if err != nil {
		t.Fatalf("Failed to add third config value: %v", err)
	}

	repo, err := git.Open(dir)
	if err != nil {
		t.Fatalf("git.Open failed: %v", err)
	}

	values, err := repo.GetConfigAllValues("test.multivalue")
	if err != nil {
		t.Fatalf("GetConfigAllValues returned error: %v", err)
	}

	if len(values) != 3 {
		t.Errorf("Expected 3 values, got %d: %v", len(values), values)
	}
	if len(values) >= 3 {
		if values[0] != "first" || values[1] != "second" || values[2] != "third" {
			t.Errorf("Expected ['first', 'second', 'third'], got %v", values)
		}
	}

	// Test non-existent key
	values, err = repo.GetConfigAllValues("test.nonexistent")
	if err != nil {
		t.Errorf("GetConfigAllValues for non-existent key returned error: %v", err)
	}
	if len(values) != 0 {
		t.Errorf("Expected empty slice for non-existent key, got %v", values)
	}
}

// TestGetConfigLocalIfSetDistinguishesUnsetFromEmpty tests that GetConfigLocalIfSet
// reports a present-but-empty key as found, which a caller checking for a
// non-empty string would misread as absent.
// Steps:
// 1. Sets up a test repository
// 2. Writes gitflow.test.empty with an empty value in local config
// 3. Calls repo.GetConfigLocalIfSet for that key
// 4. Verifies it returns an empty value, found=true and no error
// 5. Calls repo.GetConfigLocalIfSet for an unset key
// 6. Verifies it returns an empty value, found=false and no error
func TestGetConfigLocalIfSetDistinguishesUnsetFromEmpty(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	if out, err := testutil.RunGit(t, dir, "config", "--local", "gitflow.test.empty", ""); err != nil {
		t.Fatalf("Failed to write the empty-valued key: %v\nOutput: %s", err, out)
	}

	repo, err := git.Open(dir)
	if err != nil {
		t.Fatalf("git.Open failed: %v", err)
	}

	value, found, err := repo.GetConfigLocalIfSet("gitflow.test.empty")
	if err != nil {
		t.Fatalf("GetConfigLocalIfSet returned error for a present key: %v", err)
	}
	if !found {
		t.Error("Expected a present-but-empty key to be reported as found")
	}
	if value != "" {
		t.Errorf("Expected an empty value, got %q", value)
	}

	value, found, err = repo.GetConfigLocalIfSet("gitflow.test.missing")
	if err != nil {
		t.Fatalf("GetConfigLocalIfSet returned error for an unset key: %v", err)
	}
	if found {
		t.Error("Expected an unset key to be reported as not found")
	}
	if value != "" {
		t.Errorf("Expected an empty value for an unset key, got %q", value)
	}
}

// TestMoveConfigLocalMovesValue tests that MoveConfigLocal writes the source value
// under the destination key and drops the source.
// Steps:
// 1. Sets up a test repository
// 2. Writes gitflow.test.src=value in local config
// 3. Calls repo.MoveConfigLocal("gitflow.test.src", "gitflow.test.dst")
// 4. Verifies no error, gitflow.test.dst is 'value' and gitflow.test.src is gone
func TestMoveConfigLocalMovesValue(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	if out, err := testutil.RunGit(t, dir, "config", "--local", "gitflow.test.src", "value"); err != nil {
		t.Fatalf("Failed to write the source key: %v\nOutput: %s", err, out)
	}

	repo, err := git.Open(dir)
	if err != nil {
		t.Fatalf("git.Open failed: %v", err)
	}

	if err := repo.MoveConfigLocal("gitflow.test.src", "gitflow.test.dst"); err != nil {
		t.Fatalf("MoveConfigLocal returned error: %v", err)
	}

	if got := testutil.GitConfigValue(t, dir, "gitflow.test.dst"); got != "value" {
		t.Errorf("Expected the destination to be 'value', got %q", got)
	}
	if testutil.GitConfigExists(t, dir, "gitflow.test.src") {
		t.Error("Expected the source key to be removed")
	}
}

// TestMoveConfigLocalAbsentSourceClearsDestination tests that MoveConfigLocal
// mirrors rather than copies: an absent source removes the destination instead of
// leaving a stale value behind.
// Steps:
// 1. Sets up a test repository
// 2. Writes gitflow.test.dst=stale in local config, with no gitflow.test.src
// 3. Calls repo.MoveConfigLocal("gitflow.test.src", "gitflow.test.dst")
// 4. Verifies no error and that gitflow.test.dst is gone
// 5. Verifies gitflow.test.src is still absent
func TestMoveConfigLocalAbsentSourceClearsDestination(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	if out, err := testutil.RunGit(t, dir, "config", "--local", "gitflow.test.dst", "stale"); err != nil {
		t.Fatalf("Failed to write the destination key: %v\nOutput: %s", err, out)
	}

	repo, err := git.Open(dir)
	if err != nil {
		t.Fatalf("git.Open failed: %v", err)
	}

	if err := repo.MoveConfigLocal("gitflow.test.src", "gitflow.test.dst"); err != nil {
		t.Fatalf("MoveConfigLocal returned error: %v", err)
	}

	if testutil.GitConfigExists(t, dir, "gitflow.test.dst") {
		t.Errorf("Expected the stale destination to be removed, got %q", testutil.GitConfigValue(t, dir, "gitflow.test.dst"))
	}
	if testutil.GitConfigExists(t, dir, "gitflow.test.src") {
		t.Error("Expected the source key to stay absent")
	}
}

// TestMoveConfigLocalPreservesEmptySourceValue tests that an empty source VALUE is
// moved as an empty value rather than being treated as an absent key, which would
// remove the destination instead of overwriting it.
// Steps:
// 1. Sets up a test repository
// 2. Writes gitflow.test.src with an empty value and gitflow.test.dst=stale
// 3. Calls repo.MoveConfigLocal("gitflow.test.src", "gitflow.test.dst")
// 4. Verifies no error and that gitflow.test.dst exists with an empty value
// 5. Verifies gitflow.test.src is gone
func TestMoveConfigLocalPreservesEmptySourceValue(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	if out, err := testutil.RunGit(t, dir, "config", "--local", "gitflow.test.src", ""); err != nil {
		t.Fatalf("Failed to write the empty-valued source key: %v\nOutput: %s", err, out)
	}
	if out, err := testutil.RunGit(t, dir, "config", "--local", "gitflow.test.dst", "stale"); err != nil {
		t.Fatalf("Failed to write the destination key: %v\nOutput: %s", err, out)
	}

	repo, err := git.Open(dir)
	if err != nil {
		t.Fatalf("git.Open failed: %v", err)
	}

	if err := repo.MoveConfigLocal("gitflow.test.src", "gitflow.test.dst"); err != nil {
		t.Fatalf("MoveConfigLocal returned error: %v", err)
	}

	if !testutil.GitConfigExists(t, dir, "gitflow.test.dst") {
		t.Error("Expected the destination to hold an empty value, not to be removed")
	}
	if got := testutil.GitConfigValue(t, dir, "gitflow.test.dst"); got != "" {
		t.Errorf("Expected the destination value to be empty, got %q", got)
	}
	if testutil.GitConfigExists(t, dir, "gitflow.test.src") {
		t.Error("Expected the source key to be removed")
	}
}

// TestMoveConfigLocalRefusesMultiValueSource tests that a multi-value source is
// refused before anything is written, so neither name is changed. The destination
// assertion is the load-bearing one: a --get-based read would return the last
// source value and overwrite the destination on the way to reporting the error.
// Steps:
// 1. Sets up a test repository
// 2. Writes two values for gitflow.test.src and gitflow.test.dst=keep
// 3. Calls repo.MoveConfigLocal("gitflow.test.src", "gitflow.test.dst")
// 4. Verifies a non-nil error is returned
// 5. Verifies gitflow.test.src still carries both values in order
// 6. Verifies gitflow.test.dst is still 'keep'
func TestMoveConfigLocalRefusesMultiValueSource(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	if out, err := testutil.RunGit(t, dir, "config", "--local", "--add", "gitflow.test.src", "first"); err != nil {
		t.Fatalf("Failed to add the first source value: %v\nOutput: %s", err, out)
	}
	if out, err := testutil.RunGit(t, dir, "config", "--local", "--add", "gitflow.test.src", "second"); err != nil {
		t.Fatalf("Failed to add the second source value: %v\nOutput: %s", err, out)
	}
	if out, err := testutil.RunGit(t, dir, "config", "--local", "gitflow.test.dst", "keep"); err != nil {
		t.Fatalf("Failed to write the destination key: %v\nOutput: %s", err, out)
	}

	repo, err := git.Open(dir)
	if err != nil {
		t.Fatalf("git.Open failed: %v", err)
	}

	if err := repo.MoveConfigLocal("gitflow.test.src", "gitflow.test.dst"); err == nil {
		t.Error("Expected MoveConfigLocal to refuse a multi-value source")
	}

	values := testutil.GitConfigAll(t, dir, "gitflow.test.src")
	if len(values) != 2 || values[0] != "first" || values[1] != "second" {
		t.Errorf("Expected the source to keep ['first', 'second'], got %v", values)
	}
	if got := testutil.GitConfigValue(t, dir, "gitflow.test.dst"); got != "keep" {
		t.Errorf("Expected the destination to be untouched at 'keep', got %q", got)
	}
}

// TestMoveConfigLocalSameKeyIsNoOp tests that moving a key onto itself keeps the
// value. Without the guard, the write-then-unset sequence would delete the value
// it had just written.
// Steps:
// 1. Sets up a test repository
// 2. Writes gitflow.test.src=value in local config
// 3. Calls repo.MoveConfigLocal("gitflow.test.src", "gitflow.test.src")
// 4. Verifies no error and that gitflow.test.src is still 'value'
func TestMoveConfigLocalSameKeyIsNoOp(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	if out, err := testutil.RunGit(t, dir, "config", "--local", "gitflow.test.src", "value"); err != nil {
		t.Fatalf("Failed to write the source key: %v\nOutput: %s", err, out)
	}

	repo, err := git.Open(dir)
	if err != nil {
		t.Fatalf("git.Open failed: %v", err)
	}

	if err := repo.MoveConfigLocal("gitflow.test.src", "gitflow.test.src"); err != nil {
		t.Fatalf("MoveConfigLocal returned error: %v", err)
	}

	if got := testutil.GitConfigValue(t, dir, "gitflow.test.src"); got != "value" {
		t.Errorf("Expected the value to survive a same-key move, got %q", got)
	}
}
