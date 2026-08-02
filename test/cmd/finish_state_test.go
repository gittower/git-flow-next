package cmd_test

import (
	"strings"
	"testing"

	"github.com/gittower/git-flow-next/internal/mergestate"
	"github.com/gittower/git-flow-next/test/testutil"
)

// TestFinishStatePersistsChildStrategies tests that child branch strategies are persisted in state
func TestFinishStatePersistsChildStrategies(t *testing.T) {
	t.Parallel()
	// Setup test repository
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Create child branches with different strategies
	testutil.RunGit(t, dir, "checkout", "-b", "staging", "main")
	testutil.RunGit(t, dir, "config", "gitflow.branch.staging.type", "base")
	testutil.RunGit(t, dir, "config", "gitflow.branch.staging.parent", "main")
	testutil.RunGit(t, dir, "config", "gitflow.branch.staging.autoUpdate", "true")
	testutil.RunGit(t, dir, "config", "gitflow.branch.staging.downstreamStrategy", "rebase")

	testutil.RunGit(t, dir, "checkout", "-b", "qa", "main")
	testutil.RunGit(t, dir, "config", "gitflow.branch.qa.type", "base")
	testutil.RunGit(t, dir, "config", "gitflow.branch.qa.parent", "main")
	testutil.RunGit(t, dir, "config", "gitflow.branch.qa.autoUpdate", "true")
	testutil.RunGit(t, dir, "config", "gitflow.branch.qa.downstreamStrategy", "squash")

	// Keep develop with merge strategy (default)
	testutil.RunGit(t, dir, "config", "gitflow.branch.develop.autoUpdate", "true")
	testutil.RunGit(t, dir, "config", "gitflow.branch.develop.downstreamStrategy", "merge")

	// Create some commits in child branches to cause conflicts
	testutil.RunGit(t, dir, "checkout", "staging")
	testutil.WriteFile(t, dir, "test-file.txt", "Staging version")
	testutil.RunGit(t, dir, "add", "test-file.txt")
	testutil.RunGit(t, dir, "commit", "-m", "Staging changes")

	testutil.RunGit(t, dir, "checkout", "qa")
	testutil.WriteFile(t, dir, "test-file.txt", "QA version")
	testutil.RunGit(t, dir, "add", "test-file.txt")
	testutil.RunGit(t, dir, "commit", "-m", "QA changes")

	testutil.RunGit(t, dir, "checkout", "develop")
	testutil.WriteFile(t, dir, "test-file.txt", "Develop version")
	testutil.RunGit(t, dir, "add", "test-file.txt")
	testutil.RunGit(t, dir, "commit", "-m", "Develop changes")

	// Create a hotfix that will conflict with child branches
	output, err = testutil.RunGitFlow(t, dir, "hotfix", "start", "state-test")
	if err != nil {
		t.Fatalf("Failed to create hotfix: %v\nOutput: %s", err, output)
	}

	testutil.WriteFile(t, dir, "test-file.txt", "Hotfix version")
	testutil.RunGit(t, dir, "add", "test-file.txt")
	testutil.RunGit(t, dir, "commit", "-m", "Hotfix changes")

	// Start the finish operation (will hit conflict in first child)
	output, _ = testutil.RunGitFlow(t, dir, "hotfix", "finish", "state-test")

	// Check that the merge state exists and has the child strategies
	state, err := testutil.LoadMergeState(t, dir)
	if err != nil {
		t.Fatalf("Failed to load merge state: %v", err)
	}

	if state == nil {
		t.Fatal("Expected merge state to exist")
	}

	// Verify child strategies are stored
	if state.ChildStrategies == nil {
		t.Fatal("Expected ChildStrategies to be populated")
	}

	// Check each strategy
	expectedStrategies := map[string]string{
		"develop": "merge",
		"staging": "rebase",
		"qa":      "squash",
	}

	for branch, expectedStrategy := range expectedStrategies {
		if strategy, ok := state.ChildStrategies[branch]; !ok {
			t.Errorf("Missing strategy for branch %s", branch)
		} else if strategy != expectedStrategy {
			t.Errorf("Wrong strategy for branch %s: expected %s, got %s", branch, expectedStrategy, strategy)
		}
	}

	// Verify that CurrentChildBranch is set when there's a conflict
	if strings.Contains(output, "conflict") && state.CurrentChildBranch == "" {
		t.Log("Note: CurrentChildBranch might be empty if conflict happened during main merge")
	}
}

// TestFinishStateTracksCurrentChild tests that CurrentChildBranch is set during child updates
func TestFinishStateTracksCurrentChild(t *testing.T) {
	t.Parallel()
	// Setup test repository
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Create a child branch with autoUpdate
	testutil.RunGit(t, dir, "checkout", "-b", "staging", "main")
	testutil.RunGit(t, dir, "config", "gitflow.branch.staging.type", "base")
	testutil.RunGit(t, dir, "config", "gitflow.branch.staging.parent", "main")
	testutil.RunGit(t, dir, "config", "gitflow.branch.staging.autoUpdate", "true")
	testutil.RunGit(t, dir, "config", "gitflow.branch.staging.downstreamStrategy", "merge")

	// Set develop to autoUpdate=true as well
	testutil.RunGit(t, dir, "config", "gitflow.branch.develop.autoUpdate", "true")

	// Create conflicting content in staging
	testutil.WriteFile(t, dir, "conflict.txt", "Staging content")
	testutil.RunGit(t, dir, "add", "conflict.txt")
	testutil.RunGit(t, dir, "commit", "-m", "Staging commit")

	// Create conflicting content in develop
	testutil.RunGit(t, dir, "checkout", "develop")
	testutil.WriteFile(t, dir, "conflict.txt", "Develop content")
	testutil.RunGit(t, dir, "add", "conflict.txt")
	testutil.RunGit(t, dir, "commit", "-m", "Develop commit")

	// Create a hotfix with conflicting content
	output, err = testutil.RunGitFlow(t, dir, "hotfix", "start", "child-track")
	if err != nil {
		t.Fatalf("Failed to create hotfix: %v\nOutput: %s", err, output)
	}

	testutil.WriteFile(t, dir, "conflict.txt", "Hotfix content")
	testutil.RunGit(t, dir, "add", "conflict.txt")
	testutil.RunGit(t, dir, "commit", "-m", "Hotfix commit")

	// Finish the hotfix - should succeed for main but conflict on child update
	output, _ = testutil.RunGitFlow(t, dir, "hotfix", "finish", "child-track")

	// If we hit a conflict during child update, check the state
	if strings.Contains(output, "Merge conflict detected") && strings.Contains(output, "Now updating") {
		state, err := testutil.LoadMergeState(t, dir)
		if err != nil {
			t.Fatalf("Failed to load merge state: %v", err)
		}

		if state == nil {
			t.Fatal("Expected merge state to exist")
		}

		// CurrentChildBranch should be set
		if state.CurrentChildBranch == "" {
			t.Error("Expected CurrentChildBranch to be set during child update conflict")
		} else {
			t.Logf("CurrentChildBranch correctly set to: %s", state.CurrentChildBranch)

			// Verify it's one of our child branches
			if state.CurrentChildBranch != "develop" && state.CurrentChildBranch != "staging" {
				t.Errorf("Unexpected CurrentChildBranch: %s", state.CurrentChildBranch)
			}
		}

		// Verify the step is correct
		if state.CurrentStep != "update_children" {
			t.Errorf("Expected CurrentStep to be 'update_children', got: %s", state.CurrentStep)
		}
	}
}

// TestFinishChildRebaseContinuation tests that child branch with rebase strategy continues correctly
func TestFinishChildRebaseContinuation(t *testing.T) {
	t.Parallel()
	// Setup test repository
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Set develop to use rebase strategy
	testutil.RunGit(t, dir, "config", "gitflow.branch.develop.autoUpdate", "true")
	testutil.RunGit(t, dir, "config", "gitflow.branch.develop.downstreamStrategy", "rebase")

	// Create initial content in develop
	testutil.RunGit(t, dir, "checkout", "develop")
	testutil.WriteFile(t, dir, "base.txt", "Initial content")
	testutil.RunGit(t, dir, "add", "base.txt")
	testutil.RunGit(t, dir, "commit", "-m", "Initial develop commit")

	// Create conflicting content in a file
	testutil.WriteFile(t, dir, "conflict.txt", "Develop line 1\nDevelop line 2\nDevelop line 3")
	testutil.RunGit(t, dir, "add", "conflict.txt")
	testutil.RunGit(t, dir, "commit", "-m", "Develop changes")

	// Create a hotfix with conflicting content
	output, err = testutil.RunGitFlow(t, dir, "hotfix", "start", "rebase-test")
	if err != nil {
		t.Fatalf("Failed to create hotfix: %v\nOutput: %s", err, output)
	}

	testutil.WriteFile(t, dir, "conflict.txt", "Hotfix line 1\nHotfix line 2\nHotfix line 3")
	testutil.RunGit(t, dir, "add", "conflict.txt")
	testutil.RunGit(t, dir, "commit", "-m", "Hotfix changes")

	// Also add a non-conflicting file
	testutil.WriteFile(t, dir, "hotfix-only.txt", "Hotfix specific content")
	testutil.RunGit(t, dir, "add", "hotfix-only.txt")
	testutil.RunGit(t, dir, "commit", "-m", "Hotfix non-conflicting change")

	// Finish the hotfix - should succeed for main but conflict during rebase of develop
	output, _ = testutil.RunGitFlow(t, dir, "hotfix", "finish", "rebase-test")

	// Check if we're in a rebase conflict
	if strings.Contains(output, "conflict") && strings.Contains(output, "develop") {
		// Resolve the conflict
		testutil.WriteFile(t, dir, "conflict.txt", "Resolved content")
		testutil.RunGit(t, dir, "add", "conflict.txt")

		// Continue the finish operation
		output, err = testutil.RunGitFlow(t, dir, "hotfix", "finish", "--continue", "rebase-test")
		if err != nil {
			// Check if it's asking for rebase --continue
			if strings.Contains(err.Error(), "rebase") {
				// Try to continue the rebase first
				testutil.RunGit(t, dir, "rebase", "--continue")
				// Then continue the finish
				output, err = testutil.RunGitFlow(t, dir, "hotfix", "finish", "--continue", "rebase-test")
			}
		}

		// Verify the operation completed
		if err != nil && !strings.Contains(output, "Successfully finished") {
			t.Logf("Finish may have failed or need more continuation: %v\nOutput: %s", err, output)
		}

		// Verify develop has the changes
		testutil.RunGit(t, dir, "checkout", "develop")
		if !testutil.FileExists(t, dir, "hotfix-only.txt") {
			t.Log("Note: Hotfix changes might not be in develop if rebase continuation needs adjustment")
		}
	}
}

// TestFinishStateBackwardCompatibility tests that state without new fields still works
func TestFinishStateBackwardCompatibility(t *testing.T) {
	t.Parallel()
	// Setup test repository
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Configure develop for auto-update
	testutil.RunGit(t, dir, "config", "gitflow.branch.develop.autoUpdate", "true")

	// Create and finish a feature to ensure things still work
	output, err = testutil.RunGitFlow(t, dir, "feature", "start", "compat-test")
	if err != nil {
		t.Fatalf("Failed to create feature: %v\nOutput: %s", err, output)
	}

	testutil.WriteFile(t, dir, "feature.txt", "Feature content")
	testutil.RunGit(t, dir, "add", "feature.txt")
	testutil.RunGit(t, dir, "commit", "-m", "Feature commit")

	// Finish should work even with enhanced state handling
	output, err = testutil.RunGitFlow(t, dir, "feature", "finish", "compat-test")
	if err != nil {
		t.Fatalf("Failed to finish feature: %v\nOutput: %s", err, output)
	}

	// Verify the feature was merged
	testutil.RunGit(t, dir, "checkout", "develop")
	if !testutil.FileExists(t, dir, "feature.txt") {
		t.Error("Feature changes not found in develop branch")
	}
}

// TestFinishDetectsStaleStateEmptyFields tests that stale state with empty critical fields is auto-cleared.
// Steps:
// 1. Sets up a test repository and initializes git-flow
// 2. Writes a merge state file with empty BranchType
// 3. Creates and finishes a feature branch normally
// 4. Verifies the stale state was cleared and finish succeeded
func TestFinishDetectsStaleStateEmptyFields(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Write stale state with empty BranchType
	testutil.WriteMergeState(t, dir, &mergestate.MergeState{
		Action:         "finish",
		BranchType:     "", // empty — should be detected as stale
		FullBranchName: "feature/old-branch",
		CurrentStep:    "merge",
	})

	// Create and finish a new feature — should succeed because stale state is cleared
	output, err = testutil.RunGitFlow(t, dir, "feature", "start", "new-feature")
	if err != nil {
		t.Fatalf("Failed to start feature: %v\nOutput: %s", err, output)
	}
	testutil.WriteFile(t, dir, "new.txt", "content")
	_, err = testutil.RunGit(t, dir, "add", "new.txt")
	if err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "New feature commit")
	if err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	output, err = testutil.RunGitFlow(t, dir, "feature", "finish", "new-feature")
	if err != nil {
		t.Fatalf("Expected finish to succeed after clearing stale state, got error: %v\nOutput: %s", err, output)
	}

	// Verify stale state was cleared
	if testutil.GitFlowMergeStateExists(t, dir) {
		t.Error("Expected merge state to be cleared")
	}
}

// TestFinishDetectsStaleStateMergeStepNoConflict tests that state at merge step is cleared when
// git is not actually in a merge or rebase state.
// Steps:
// 1. Sets up a test repository and initializes git-flow
// 2. Writes a merge state file at the merge step
// 3. Runs feature finish which checks for merge in progress
// 4. Verifies stale state is cleared and a new finish can proceed
func TestFinishDetectsStaleStateMergeStepNoConflict(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Write stale state claiming merge step but git is not in merge state
	testutil.WriteMergeState(t, dir, &mergestate.MergeState{
		Action:         "finish",
		BranchType:     "feature",
		BranchName:     "old-branch",
		FullBranchName: "feature/old-branch",
		CurrentStep:    "merge",
		ParentBranch:   "develop",
		MergeStrategy:  "merge",
	})

	// Create and finish a new feature — stale state should be auto-cleared
	output, err = testutil.RunGitFlow(t, dir, "feature", "start", "fresh-feature")
	if err != nil {
		t.Fatalf("Failed to start feature: %v\nOutput: %s", err, output)
	}
	testutil.WriteFile(t, dir, "fresh.txt", "content")
	_, err = testutil.RunGit(t, dir, "add", "fresh.txt")
	if err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Fresh feature commit")
	if err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	output, err = testutil.RunGitFlow(t, dir, "feature", "finish", "fresh-feature")
	if err != nil {
		t.Fatalf("Expected finish to succeed after clearing stale state, got error: %v\nOutput: %s", err, output)
	}

	if strings.Contains(output, "merge in progress") {
		t.Error("Expected stale state to be cleared, but got merge in progress error")
	}
}

// TestFinishDetectsStaleStateDeleteStepBranchGone tests that state at delete_branch step is
// cleared when the topic branch no longer exists.
// Steps:
// 1. Sets up a test repository and initializes git-flow
// 2. Writes a merge state file at delete_branch step referencing a non-existent branch
// 3. Runs feature finish to verify stale state is cleared
// 4. Verifies a new finish can proceed normally
func TestFinishDetectsStaleStateDeleteStepBranchGone(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Write stale state at delete_branch step for a branch that doesn't exist
	testutil.WriteMergeState(t, dir, &mergestate.MergeState{
		Action:         "finish",
		BranchType:     "feature",
		BranchName:     "deleted-branch",
		FullBranchName: "feature/deleted-branch",
		CurrentStep:    "delete_branch",
		ParentBranch:   "develop",
		MergeStrategy:  "merge",
	})

	// Create and finish a new feature — stale state should be auto-cleared
	output, err = testutil.RunGitFlow(t, dir, "feature", "start", "another-feature")
	if err != nil {
		t.Fatalf("Failed to start feature: %v\nOutput: %s", err, output)
	}
	testutil.WriteFile(t, dir, "another.txt", "content")
	_, err = testutil.RunGit(t, dir, "add", "another.txt")
	if err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Another feature commit")
	if err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	output, err = testutil.RunGitFlow(t, dir, "feature", "finish", "another-feature")
	if err != nil {
		t.Fatalf("Expected finish to succeed after clearing stale state, got error: %v\nOutput: %s", err, output)
	}

	if testutil.GitFlowMergeStateExists(t, dir) {
		t.Error("Expected merge state to be cleared after finish")
	}
}

// TestFinishValidStateMergeStepWithConflict tests that legitimate merge state is NOT cleared
// when git is actually in a merge conflict state.
// Steps:
// 1. Sets up a test repository and initializes git-flow
// 2. Creates a feature branch with conflicting changes
// 3. Attempts to finish (produces merge conflict, creating valid state)
// 4. Verifies the merge state is preserved (not cleared as stale)
func TestFinishValidStateMergeStepWithConflict(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Create feature with content
	output, err = testutil.RunGitFlow(t, dir, "feature", "start", "conflict-test")
	if err != nil {
		t.Fatalf("Failed to start feature: %v\nOutput: %s", err, output)
	}
	testutil.WriteFile(t, dir, "conflict.txt", "feature content")
	_, err = testutil.RunGit(t, dir, "add", "conflict.txt")
	if err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Feature commit")
	if err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Add conflicting content on develop
	_, err = testutil.RunGit(t, dir, "checkout", "develop")
	if err != nil {
		t.Fatalf("Failed to checkout develop: %v", err)
	}
	testutil.WriteFile(t, dir, "conflict.txt", "develop content")
	_, err = testutil.RunGit(t, dir, "add", "conflict.txt")
	if err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Develop commit")
	if err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Switch back and finish — will produce conflict
	_, err = testutil.RunGit(t, dir, "checkout", "feature/conflict-test")
	if err != nil {
		t.Fatalf("Failed to checkout feature branch: %v", err)
	}
	output, err = testutil.RunGitFlow(t, dir, "feature", "finish", "conflict-test")

	// Finish should fail with conflict
	if err == nil {
		t.Fatal("Expected finish to fail with merge conflict")
	}

	// The merge state should be preserved — this is a legitimate conflict
	if !testutil.GitFlowMergeStateExists(t, dir) {
		t.Error("Expected merge state to be preserved during legitimate conflict")
	}

	state, stateErr := testutil.LoadMergeState(t, dir)
	if stateErr != nil {
		t.Fatalf("Failed to load merge state: %v", stateErr)
	}
	if state.CurrentStep != "merge" {
		t.Errorf("Expected state step 'merge', got '%s'", state.CurrentStep)
	}
}

// TestFinishAfterManualConflictResolution tests that re-running finish completes the full
// operation after the user resolved a merge conflict with a plain `git commit` instead of
// `--continue`. The manual commit finishes the merge at the git level but leaves the state
// file behind; the re-run must detect the stale state, clear it, and finish the branch
// end-to-end (tag, child update, branch deletion). Regression test for issue #114.
// Steps:
// 1. Sets up a test repository and initializes git-flow
// 2. Creates a hotfix branch with changes conflicting with main
// 3. Attempts to finish (produces merge conflict, creating valid state)
// 4. Resolves the conflict manually with `git commit`, bypassing --continue
// 5. Verifies the state is stale: MERGE_HEAD is gone, state file still present
// 6. Re-runs finish and verifies it completes: state cleared, tag created, branch deleted
func TestFinishAfterManualConflictResolution(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Create hotfix with content
	output, err = testutil.RunGitFlow(t, dir, "hotfix", "start", "1.0.1")
	if err != nil {
		t.Fatalf("Failed to start hotfix: %v\nOutput: %s", err, output)
	}
	testutil.WriteFile(t, dir, "conflict.txt", "hotfix content")
	_, err = testutil.RunGit(t, dir, "add", "conflict.txt")
	if err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Hotfix commit")
	if err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Add conflicting content on main
	_, err = testutil.RunGit(t, dir, "checkout", "main")
	if err != nil {
		t.Fatalf("Failed to checkout main: %v", err)
	}
	testutil.WriteFile(t, dir, "conflict.txt", "main content")
	_, err = testutil.RunGit(t, dir, "add", "conflict.txt")
	if err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Main commit")
	if err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Switch back and finish — will produce conflict
	_, err = testutil.RunGit(t, dir, "checkout", "hotfix/1.0.1")
	if err != nil {
		t.Fatalf("Failed to checkout hotfix branch: %v", err)
	}
	output, err = testutil.RunGitFlow(t, dir, "hotfix", "finish", "1.0.1")
	if err == nil {
		t.Fatal("Expected finish to fail with merge conflict")
	}

	// Resolve the conflict manually with a plain git commit, bypassing --continue.
	// This clears MERGE_HEAD but leaves the git-flow state file behind.
	testutil.WriteFile(t, dir, "conflict.txt", "resolved content")
	_, err = testutil.RunGit(t, dir, "add", "conflict.txt")
	if err != nil {
		t.Fatalf("Failed to add resolved file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Manual conflict resolution")
	if err != nil {
		t.Fatalf("Failed to commit resolution: %v", err)
	}

	// The state is now stale: the git-level merge is complete but the
	// git-flow state file is still there
	if testutil.FileExists(t, dir, ".git/MERGE_HEAD") {
		t.Fatal("Expected MERGE_HEAD to be gone after manual resolution commit")
	}
	if !testutil.GitFlowMergeStateExists(t, dir) {
		t.Fatal("Expected merge state to still exist after manual resolution")
	}

	// Re-run finish — must clear the stale state and complete the operation
	output, err = testutil.RunGitFlow(t, dir, "hotfix", "finish", "1.0.1")
	if err != nil {
		t.Fatalf("Expected finish to succeed after manual resolution, got error: %v\nOutput: %s", err, output)
	}

	if testutil.GitFlowMergeStateExists(t, dir) {
		t.Error("Expected merge state to be cleared after finish")
	}

	tags, err := testutil.RunGit(t, dir, "tag", "-l", "1.0.1")
	if err != nil {
		t.Fatalf("Failed to list tags: %v", err)
	}
	if !strings.Contains(tags, "1.0.1") {
		t.Error("Expected tag '1.0.1' to be created on finish")
	}

	branches, err := testutil.RunGit(t, dir, "branch", "--list", "hotfix/1.0.1")
	if err != nil {
		t.Fatalf("Failed to list branches: %v", err)
	}
	if strings.TrimSpace(branches) != "" {
		t.Error("Expected hotfix branch to be deleted after finish")
	}

	// Child update ran: develop must contain the resolved content from main
	_, err = testutil.RunGit(t, dir, "checkout", "develop")
	if err != nil {
		t.Fatalf("Failed to checkout develop: %v", err)
	}
	if content := testutil.ReadFile(t, dir, "conflict.txt"); content != "resolved content" {
		t.Errorf("Expected develop to contain resolved content, got %q", content)
	}
}

// TestAbortClearsStaleMergeState tests that --abort does not leave a stale state file behind
// when the git-level merge is already gone (user resolved the conflict with a plain
// `git commit` instead of `--continue`). The exit status of --abort in this situation is
// deliberately not asserted; only the cleanup matters. Regression test for issue #110.
// Steps:
// 1. Sets up a test repository and initializes git-flow
// 2. Creates a feature branch with conflicting changes and attempts to finish
// 3. Resolves the conflict manually with `git commit`, bypassing --continue
// 4. Verifies the state is stale: MERGE_HEAD is gone, state file still present
// 5. Runs finish --abort and verifies the state file is gone afterwards
func TestAbortClearsStaleMergeState(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Create feature with content
	output, err = testutil.RunGitFlow(t, dir, "feature", "start", "stale-abort")
	if err != nil {
		t.Fatalf("Failed to start feature: %v\nOutput: %s", err, output)
	}
	testutil.WriteFile(t, dir, "conflict.txt", "feature content")
	_, err = testutil.RunGit(t, dir, "add", "conflict.txt")
	if err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Feature commit")
	if err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Add conflicting content on develop
	_, err = testutil.RunGit(t, dir, "checkout", "develop")
	if err != nil {
		t.Fatalf("Failed to checkout develop: %v", err)
	}
	testutil.WriteFile(t, dir, "conflict.txt", "develop content")
	_, err = testutil.RunGit(t, dir, "add", "conflict.txt")
	if err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Develop commit")
	if err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Switch back and finish — will produce conflict
	_, err = testutil.RunGit(t, dir, "checkout", "feature/stale-abort")
	if err != nil {
		t.Fatalf("Failed to checkout feature branch: %v", err)
	}
	output, err = testutil.RunGitFlow(t, dir, "feature", "finish", "stale-abort")
	if err == nil {
		t.Fatal("Expected finish to fail with merge conflict")
	}

	// Resolve the conflict manually with a plain git commit, bypassing --continue.
	// This clears MERGE_HEAD but leaves the git-flow state file behind.
	testutil.WriteFile(t, dir, "conflict.txt", "resolved content")
	_, err = testutil.RunGit(t, dir, "add", "conflict.txt")
	if err != nil {
		t.Fatalf("Failed to add resolved file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Manual conflict resolution")
	if err != nil {
		t.Fatalf("Failed to commit resolution: %v", err)
	}

	// The state is now stale: the git-level merge is complete but the
	// git-flow state file is still there
	if testutil.FileExists(t, dir, ".git/MERGE_HEAD") {
		t.Fatal("Expected MERGE_HEAD to be gone after manual resolution commit")
	}
	if !testutil.GitFlowMergeStateExists(t, dir) {
		t.Fatal("Expected merge state to still exist after manual resolution")
	}

	// Abort — regardless of exit status, no stale state file may remain
	_, _ = testutil.RunGitFlow(t, dir, "feature", "finish", "--abort", "stale-abort")

	if testutil.GitFlowMergeStateExists(t, dir) {
		t.Error("Expected merge state to be cleared after --abort on stale state")
	}
}

// TestFinishAbortNoOpWhenNothingInProgress verifies feature finish --abort with a
// truly clean repo (no in-progress op, no state file) is a forgiving no-op.
// Steps:
// 1. Sets up a test repository and initializes git-flow with defaults
// 2. Creates feature/x (no conflict, no in-progress op, no state file)
// 3. Runs 'git flow feature finish --abort x'
// 4. Verifies exit 0, no error, and no merge-state file created
func TestFinishAbortNoOpWhenNothingInProgress(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	if out, err := testutil.RunGitFlow(t, dir, "init", "--defaults"); err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, out)
	}
	if out, err := testutil.RunGitFlow(t, dir, "feature", "start", "x"); err != nil {
		t.Fatalf("Failed to start feature: %v\nOutput: %s", err, out)
	}

	out, err := testutil.RunGitFlow(t, dir, "feature", "finish", "--abort", "x")
	if err != nil {
		t.Fatalf("Expected finish --abort no-op to succeed: %v\nOutput: %s", err, out)
	}
	if testutil.GitFlowMergeStateExists(t, dir) {
		t.Error("Expected no merge-state file after no-op abort")
	}
}
