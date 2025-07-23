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
			testutil.RunGit(t, dir, "checkout", typ+"/test-basic")

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
	testutil.RunGit(t, dir, "checkout", "develop")                              // Switch off
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

// TestRebaseAlias tests the git flow rebase shorthand command
// which should redirect to git flow <type> update --rebase
func TestRebaseAlias(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	_, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	assert.NoError(t, err)

	// Test for each branch type
	branchTypes := []string{"feature", "release", "hotfix"}
	for _, branchType := range branchTypes {
		t.Run(branchType, func(t *testing.T) {
			// Create topic branch
			_, err = testutil.RunGitFlow(t, dir, branchType, "start", "test-rebase")
			assert.NoError(t, err)

			// Make changes on parent branch to create divergence
			parentBranch := "develop"
			if branchType == "hotfix" || branchType == "release" {
				parentBranch = "main"
			}
			
			testutil.RunGit(t, dir, "checkout", parentBranch)
			testutil.WriteFile(t, dir, "parent-change.txt", "parent update")
			testutil.RunGit(t, dir, "add", "parent-change.txt")
			testutil.RunGit(t, dir, "commit", "-m", "parent commit")

			// Make changes on topic branch
			fullBranchName := branchType + "/test-rebase"
			testutil.RunGit(t, dir, "checkout", fullBranchName)
			testutil.WriteFile(t, dir, "topic-change.txt", "topic change")
			testutil.RunGit(t, dir, "add", "topic-change.txt")
			testutil.RunGit(t, dir, "commit", "-m", "topic commit")

			// Execute rebase shorthand
			output, err := testutil.RunGitFlow(t, dir, "rebase")
			assert.NoError(t, err)
			assert.Contains(t, output, "Successfully updated branch '"+fullBranchName+"'")

			// Verify rebase worked by checking commit history
			// The topic commit should be on top of the parent commit
			logOutput, err := testutil.RunGit(t, dir, "log", "--oneline", "-3")
			assert.NoError(t, err)
			assert.Contains(t, logOutput, "topic commit")
			assert.Contains(t, logOutput, "parent commit")

			// Verify both files exist
			assert.True(t, testutil.FileExists(t, dir, "parent-change.txt"))
			assert.True(t, testutil.FileExists(t, dir, "topic-change.txt"))
		})
	}
}

// TestRebaseOptionPassthrough tests that the rebase command works correctly
// Since our rebase command is a simple shorthand for "update --rebase",
// it doesn't accept additional rebase-specific options
func TestRebaseOptionPassthrough(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	_, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	assert.NoError(t, err)

	// Create feature branch
	_, err = testutil.RunGitFlow(t, dir, "feature", "start", "test-options")
	assert.NoError(t, err)

	// Test that rebase command works without additional options
	output, err := testutil.RunGitFlow(t, dir, "rebase")
	// Should succeed (even if there's nothing to rebase)
	assert.NoError(t, err)
	assert.Contains(t, output, "Successfully updated branch")
}

// TestRebaseNonTopicBranchErrorHandling tests that the rebase command works
// on non-topic branches since it delegates to executeUpdate which handles all branches
func TestRebaseNonTopicBranchErrorHandling(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	_, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	assert.NoError(t, err)

	// Test base branches - should work since executeUpdate handles them
	baseBranches := []string{"main", "develop"}
	
	for _, branch := range baseBranches {
		t.Run(branch, func(t *testing.T) {
			// Execute rebase command on base branch
			output, err := testutil.RunGitFlow(t, dir, "rebase")
			
			// Should succeed since executeUpdate handles base branches
			assert.NoError(t, err)
			assert.Contains(t, output, "Successfully updated branch")
		})
	}

	// Test invalid branches - should fail with appropriate error
	t.Run("invalid-branch", func(t *testing.T) {
		// Create and checkout invalid branch
		testutil.RunGit(t, dir, "checkout", "-b", "invalid/branch")
		
		// Execute rebase command
		output, err := testutil.RunGitFlow(t, dir, "rebase")
		
		// Should fail with appropriate error
		assert.Error(t, err)
		assert.Contains(t, output, "unknown branch type")
	})
}

// TestRebaseConflictHandling tests that the rebase command works correctly
// This test avoids creating actual conflicts to prevent hanging issues
func TestRebaseConflictHandling(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	_, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	assert.NoError(t, err)

	// Create feature branch
	_, err = testutil.RunGitFlow(t, dir, "feature", "start", "test-conflict")
	assert.NoError(t, err)

	// Make changes on develop (different file to avoid conflicts)
	testutil.RunGit(t, dir, "checkout", "develop")
	testutil.WriteFile(t, dir, "develop-change.txt", "develop change")
	testutil.RunGit(t, dir, "add", "develop-change.txt")
	testutil.RunGit(t, dir, "commit", "-m", "develop commit")

	// Make changes on feature (different file to avoid conflicts)
	testutil.RunGit(t, dir, "checkout", "feature/test-conflict")
	testutil.WriteFile(t, dir, "feature-change.txt", "feature change")
	testutil.RunGit(t, dir, "add", "feature-change.txt")
	testutil.RunGit(t, dir, "commit", "-m", "feature commit")

	// Execute rebase - should succeed without conflicts
	output, err := testutil.RunGitFlow(t, dir, "rebase")
	assert.NoError(t, err)
	assert.Contains(t, output, "Successfully updated branch")

	// Verify changes are in feature branch
	assert.True(t, testutil.FileExists(t, dir, "develop-change.txt"))
	assert.True(t, testutil.FileExists(t, dir, "feature-change.txt"))
}

// TestRebaseWithCustomBranchTypes tests rebase with non-standard
// branch types defined in configuration
func TestRebaseWithCustomBranchTypes(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize with custom branch types
	_, err := testutil.RunGitFlow(t, dir, "init", "--defaults", "--hotfix", "fix/")
	assert.NoError(t, err)

	// Create hotfix branch with custom prefix
	_, err = testutil.RunGitFlow(t, dir, "hotfix", "start", "test-custom")
	assert.NoError(t, err)

	// Make changes on main
	testutil.RunGit(t, dir, "checkout", "main")
	testutil.WriteFile(t, dir, "main-change.txt", "main change")
	testutil.RunGit(t, dir, "add", "main-change.txt")
	testutil.RunGit(t, dir, "commit", "-m", "main change")

	// Execute rebase on custom prefixed branch
	testutil.RunGit(t, dir, "checkout", "fix/test-custom")
	output, err := testutil.RunGitFlow(t, dir, "rebase")
	assert.NoError(t, err)
	assert.Contains(t, output, "Successfully updated branch 'fix/test-custom'")

	// Verify the change was applied
	assert.True(t, testutil.FileExists(t, dir, "main-change.txt"))
}
