package cmd_test

import (
	"testing"

	"github.com/gittower/git-flow-next/test/testutil"
	"github.com/stretchr/testify/assert"
)

// TestBasicCommandDetection verifies shorthand redirects correctly
func TestBasicCommandDetection(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Init with defaults
	_, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	assert.NoError(t, err)

	// Test for each type
	types := []string{"feature", "release", "hotfix"}
	for _, typ := range types {
		t.Run(typ, func(t *testing.T) {
			// Create, commit
			_, err = testutil.RunGitFlow(t, dir, typ, "start", "test-basic")
			assert.NoError(t, err)

			testutil.WriteFile(t, dir, "file.txt", "test")
			testutil.RunGit(t, dir, "add", "file.txt")
			testutil.RunGit(t, dir, "commit", "-m", "test commit")

			// Ensure on the branch for shorthand
			testutil.RunGit(t, dir, "checkout", typ + "/test-basic")

			output, err := testutil.RunGitFlow(t, dir, "finish", "--notag", "--no-keep")
			assert.NoError(t, err)
			assert.Contains(t, output, "Successfully finished branch '"+typ+"/test-basic'")
		})
	}
}

// TestBranchNameParsing checks standard and non-standard names
func TestBranchNameParsing(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	testutil.RunGitFlow(t, dir, "init", "--defaults")

	// Standard
	testutil.RunGitFlow(t, dir, "feature", "start", "standard")
	testutil.RunGit(t, dir, "checkout", "feature/standard") // Ensure on branch
	output, err := testutil.RunGitFlow(t, dir, "rename", "new-standard")
	assert.NoError(t, err)
	assert.Contains(t, output, "Renamed branch 'feature/standard' to 'feature/new-standard'")

	// Non-standard (slashes/dashes)
	testutil.RunGit(t, dir, "checkout", "develop")
	testutil.RunGitFlow(t, dir, "feature", "start", "with/slashes-and-dashes")
	testutil.RunGit(t, dir, "checkout", "feature/with/slashes-and-dashes")
	output, err = testutil.RunGitFlow(t, dir, "rename", "new/with-slashes")
	assert.NoError(t, err)
	assert.Contains(t, output, "Renamed branch 'feature/with/slashes-and-dashes' to 'feature/new/with-slashes'")
}

// TestCommandOptionsPassthrough verifies flags are passed
func TestCommandOptionsPassthrough(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	testutil.RunGitFlow(t, dir, "init", "--defaults")

	testutil.RunGitFlow(t, dir, "feature", "start", "test-options")
	testutil.RunGit(t, dir, "checkout", "feature/test-options") // Ensure on branch
	output, err := testutil.RunGitFlow(t, dir, "finish", "--keep", "--notag")
	assert.NoError(t, err)
	assert.Contains(t, output, "Successfully finished branch 'feature/test-options'")
	assert.True(t, testutil.BranchExists(t, dir, "feature/test-options")) // Verify --keep worked
}

// TestNonTopicBranchErrorHandling checks errors on non-topic
func TestNonTopicBranchErrorHandling(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	testutil.RunGitFlow(t, dir, "init", "--defaults")

	nonTopics := []string{"main", "develop", "custom/branch"}
	for _, branch := range nonTopics {
		testutil.RunGit(t, dir, "checkout", "-b", branch)
		output, err := testutil.RunGitFlow(t, dir, "finish")
		assert.Error(t, err)
		assert.Contains(t, output, "not a valid topic branch")
		if exitErr, ok := err.(*testutil.ExitError); ok {
			assert.NotZero(t, exitErr.ExitCode)
		} else {
			t.Errorf("Expected ExitError for %s", branch)
		}
	}
}

// TestAmbiguousBranchDetection checks prompt/error for ambiguity
func TestAmbiguousBranchDetection(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	testutil.RunGitFlow(t, dir, "init", "--defaults", "--feature", "feat/", "--hotfix", "feat/") // Force overlap

	testutil.RunGit(t, dir, "checkout", "-b", "feat/ambiguous")
	output, err := testutil.RunGitFlowWithInput(t, dir, "n\n", "finish") // Simulate 'n'
	assert.Error(t, err)
	assert.Contains(t, output, "Ambiguous branch")
	assert.Contains(t, output, "operation cancelled")
}

// Command-Specific Tests
func TestDeleteAlias(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	testutil.RunGitFlow(t, dir, "init", "--defaults")
	testutil.RunGitFlow(t, dir, "feature", "start", "test-delete")
	testutil.RunGit(t, dir, "checkout", "develop") // Switch off
	output, err := testutil.RunGitFlow(t, dir, "delete", "feature/test-delete") // Use full prefixed name
	assert.NoError(t, err)
	assert.Contains(t, output, "Deleted branch feature/test-delete")
	assert.False(t, testutil.BranchExists(t, dir, "feature/test-delete"))
}

func TestUpdateAlias(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	testutil.RunGitFlow(t, dir, "init", "--defaults")
	testutil.RunGitFlow(t, dir, "feature", "start", "test-update")

	// Make parent changes
	testutil.RunGit(t, dir, "checkout", "develop")
	testutil.WriteFile(t, dir, "update.txt", "parent update")
	testutil.RunGit(t, dir, "add", "update.txt")
	testutil.RunGit(t, dir, "commit", "-m", "parent commit")

	// Update via shorthand (from topic branch)
	testutil.RunGit(t, dir, "checkout", "feature/test-update")
	output, err := testutil.RunGitFlow(t, dir, "update")
	assert.NoError(t, err)
	assert.Contains(t, output, "Successfully updated branch 'feature/test-update'")
	contents := testutil.ReadFile(t, dir, "update.txt")
	assert.Equal(t, "parent update", contents)
}

func TestRenameAlias(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	testutil.RunGitFlow(t, dir, "init", "--defaults")
	testutil.RunGitFlow(t, dir, "feature", "start", "test-rename")
	testutil.RunGit(t, dir, "checkout", "feature/test-rename")
	output, err := testutil.RunGitFlow(t, dir, "rename", "renamed-feature")
	assert.NoError(t, err)
	assert.Contains(t, output, "Renamed branch 'feature/test-rename' to 'feature/renamed-feature'")
	assert.True(t, testutil.BranchExists(t, dir, "feature/renamed-feature"))
}

func TestFinishAlias(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	testutil.RunGitFlow(t, dir, "init", "--defaults")
	testutil.RunGitFlow(t, dir, "feature", "start", "test-finish")
	testutil.WriteFile(t, dir, "finish.txt", "test")
	testutil.RunGit(t, dir, "add", "finish.txt")
	testutil.RunGit(t, dir, "commit", "-m", "finish commit")
	testutil.RunGit(t, dir, "checkout", "feature/test-finish")
	output, err := testutil.RunGitFlow(t, dir, "finish", "--notag", "--no-keep")
	assert.NoError(t, err)
	assert.Contains(t, output, "Successfully finished branch 'feature/test-finish'")
	assert.False(t, testutil.BranchExists(t, dir, "feature/test-finish"))
	testutil.RunGit(t, dir, "checkout", "develop")
	contents := testutil.ReadFile(t, dir, "finish.txt")
	assert.Equal(t, "test", contents)
}

// TestIntegrationWithExistingConfig checks custom config
func TestIntegrationWithExistingConfig(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	testutil.RunGitFlow(t, dir, "init", "--defaults", "--feature", "feat/")
	testutil.RunGitFlow(t, dir, "feature", "start", "test-config")
	testutil.RunGit(t, dir, "checkout", "feat/test-config")
	output, err := testutil.RunGitFlow(t, dir, "finish", "--notag", "--no-keep")
	assert.NoError(t, err)
	assert.Contains(t, output, "Successfully finished branch 'feat/test-config'")
}

// TestExitCodePropagation checks failure codes
func TestExitCodePropagation(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	testutil.RunGitFlow(t, dir, "init", "--defaults")
	testutil.RunGitFlow(t, dir, "feature", "start", "test-fail")

	// Force conflict
	testutil.RunGit(t, dir, "checkout", "develop")
	testutil.WriteFile(t, dir, "conflict.txt", "develop")
	testutil.RunGit(t, dir, "add", "conflict.txt")
	testutil.RunGit(t, dir, "commit", "-m", "conflict")

	testutil.RunGit(t, dir, "checkout", "feature/test-fail")
	testutil.WriteFile(t, dir, "conflict.txt", "feature")
	testutil.RunGit(t, dir, "add", "conflict.txt")
	testutil.RunGit(t, dir, "commit", "-m", "conflict")

	output, err := testutil.RunGitFlow(t, dir, "finish")
	assert.Error(t, err)
	assert.Contains(t, output, "conflict") // Assumes conflict message
	if exitErr, ok := err.(*testutil.ExitError); ok {
		assert.NotZero(t, exitErr.ExitCode)
	} else {
		t.Error("Expected ExitError")
	}
}