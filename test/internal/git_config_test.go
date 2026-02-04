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
// 3. Calls GetConfigAllValues for that key
// 4. Verifies all values are returned in order
// 5. Calls GetConfigAllValues for a non-existent key
// 6. Verifies empty result with no error
func TestGetConfigAllValues(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Change to the test directory so git commands work
	// (GetConfigAllValues uses git config which operates in current directory)
	originalDir := t.TempDir() // dummy to satisfy the compiler
	_ = originalDir

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

	// We need to run the test from within the test directory
	// since GetConfigAllValues uses git config which operates in the current directory
	// Use a subprocess approach via testutil
	values, err := git.GetConfigAllValuesInDir(dir, "test.multivalue")
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
	values, err = git.GetConfigAllValuesInDir(dir, "test.nonexistent")
	if err != nil {
		t.Errorf("GetConfigAllValues for non-existent key returned error: %v", err)
	}
	if len(values) != 0 {
		t.Errorf("Expected empty slice for non-existent key, got %v", values)
	}
}
