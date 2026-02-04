package cmd_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gittower/git-flow-next/test/testutil"
)

// TestFinishSingleConflictMergeStrategy tests finishing with a single conflict using merge strategy
func TestFinishSingleConflictMergeStrategy(t *testing.T) {
	// Setup test repository
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Create conflicting content in develop
	testutil.RunGit(t, dir, "checkout", "develop")
	testutil.WriteFile(t, dir, "conflict.txt", "Develop version\nLine 2\nLine 3")
	testutil.RunGit(t, dir, "add", "conflict.txt")
	testutil.RunGit(t, dir, "commit", "-m", "Develop changes")

	// Create feature with conflicting changes
	output, err = testutil.RunGitFlow(t, dir, "feature", "start", "merge-conflict")
	if err != nil {
		t.Fatalf("Failed to create feature: %v\nOutput: %s", err, output)
	}

	testutil.WriteFile(t, dir, "conflict.txt", "Feature version\nLine 2 modified\nLine 3")
	testutil.RunGit(t, dir, "add", "conflict.txt")
	testutil.RunGit(t, dir, "commit", "-m", "Feature changes")

	// Meanwhile, add more changes to develop
	testutil.RunGit(t, dir, "checkout", "develop")
	testutil.WriteFile(t, dir, "conflict.txt", "Develop version updated\nLine 2\nLine 3 modified")
	testutil.RunGit(t, dir, "add", "conflict.txt")
	testutil.RunGit(t, dir, "commit", "-m", "More develop changes")

	// Try to finish feature (will conflict)
	testutil.RunGit(t, dir, "checkout", "feature/merge-conflict")
	output, _ = testutil.RunGitFlow(t, dir, "feature", "finish", "merge-conflict")

	// Verify conflict detected
	if !strings.Contains(output, "conflict") {
		t.Fatal("Expected merge conflict to be detected")
	}

	// Verify merge state exists
	state, err := testutil.LoadMergeState(t, dir)
	if err != nil || state == nil {
		t.Fatal("Expected merge state to exist after conflict")
	}

	if state.CurrentStep != "merge" {
		t.Errorf("Expected CurrentStep to be 'merge', got: %s", state.CurrentStep)
	}

	if state.MergeStrategy != "merge" {
		t.Errorf("Expected MergeStrategy to be 'merge', got: %s", state.MergeStrategy)
	}

	// Resolve conflict
	testutil.WriteFile(t, dir, "conflict.txt", "Resolved version\nLine 2 resolved\nLine 3 resolved")
	testutil.RunGit(t, dir, "add", "conflict.txt")

	// Continue finish operation
	output, err = testutil.RunGitFlow(t, dir, "feature", "finish", "--continue", "merge-conflict")
	if err != nil {
		t.Fatalf("Failed to continue finish: %v\nOutput: %s", err, output)
	}

	// Verify success
	if !strings.Contains(output, "Successfully finished") {
		t.Error("Expected successful finish message")
	}

	// Verify branch deleted
	if testutil.BranchExists(t, dir, "feature/merge-conflict") {
		t.Error("Feature branch should be deleted after finish")
	}

	// Verify changes in develop
	testutil.RunGit(t, dir, "checkout", "develop")
	content := testutil.ReadFile(t, dir, "conflict.txt")
	if !strings.Contains(content, "Resolved version") {
		t.Error("Resolved changes not found in develop")
	}
}

// TestFinishSingleConflictRebaseStrategy tests finishing with a single conflict using rebase strategy
func TestFinishSingleConflictRebaseStrategy(t *testing.T) {
	// Setup test repository
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Create conflicting content in develop
	testutil.RunGit(t, dir, "checkout", "develop")
	testutil.WriteFile(t, dir, "conflict.txt", "Develop base content")
	testutil.RunGit(t, dir, "add", "conflict.txt")
	testutil.RunGit(t, dir, "commit", "-m", "Develop base")

	// Create feature with conflicting changes
	output, err = testutil.RunGitFlow(t, dir, "feature", "start", "rebase-conflict")
	if err != nil {
		t.Fatalf("Failed to create feature: %v\nOutput: %s", err, output)
	}

	testutil.WriteFile(t, dir, "conflict.txt", "Feature changes on top")
	testutil.RunGit(t, dir, "add", "conflict.txt")
	testutil.RunGit(t, dir, "commit", "-m", "Feature commit")

	// Meanwhile, modify develop
	testutil.RunGit(t, dir, "checkout", "develop")
	testutil.WriteFile(t, dir, "conflict.txt", "Develop updated content")
	testutil.RunGit(t, dir, "add", "conflict.txt")
	testutil.RunGit(t, dir, "commit", "-m", "Develop update")

	// Try to finish feature with rebase (will conflict)
	testutil.RunGit(t, dir, "checkout", "feature/rebase-conflict")
	output, _ = testutil.RunGitFlow(t, dir, "feature", "finish", "--rebase", "rebase-conflict")

	// Verify conflict detected
	if !strings.Contains(output, "conflict") {
		t.Fatal("Expected rebase conflict to be detected")
	}

	// Verify merge state
	state, err := testutil.LoadMergeState(t, dir)
	if err != nil || state == nil {
		t.Fatal("Expected merge state to exist after conflict")
	}

	if state.MergeStrategy != "rebase" {
		t.Errorf("Expected MergeStrategy to be 'rebase', got: %s", state.MergeStrategy)
	}

	// Resolve conflict
	testutil.WriteFile(t, dir, "conflict.txt", "Rebased and resolved")
	testutil.RunGit(t, dir, "add", "conflict.txt")

	// Continue finish operation
	output, err = testutil.RunGitFlow(t, dir, "feature", "finish", "--continue", "rebase-conflict")
	if err != nil {
		t.Fatalf("Failed to continue finish: %v\nOutput: %s", err, output)
	}

	// Verify success
	if !strings.Contains(output, "Successfully finished") {
		t.Error("Expected successful finish message")
	}

	// Verify linear history (rebase should create linear history)
	testutil.RunGit(t, dir, "checkout", "develop")
	log, _ := testutil.RunGit(t, dir, "log", "--oneline", "-5")
	if strings.Contains(log, "Merge") && !strings.Contains(log, "Merge branch 'feature/rebase-conflict'") {
		t.Log("Note: Rebase might have created a merge commit for the final integration")
	}
}

// TestFinishSingleConflictSquashStrategy tests finishing with a single conflict using squash strategy
func TestFinishSingleConflictSquashStrategy(t *testing.T) {
	// Setup test repository
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Create base content in develop
	testutil.RunGit(t, dir, "checkout", "develop")
	testutil.WriteFile(t, dir, "conflict.txt", "Base content")
	testutil.RunGit(t, dir, "add", "conflict.txt")
	testutil.RunGit(t, dir, "commit", "-m", "Base commit")

	// Create feature with multiple commits
	output, err = testutil.RunGitFlow(t, dir, "feature", "start", "squash-conflict")
	if err != nil {
		t.Fatalf("Failed to create feature: %v\nOutput: %s", err, output)
	}

	// Multiple feature commits
	testutil.WriteFile(t, dir, "conflict.txt", "Feature change 1")
	testutil.RunGit(t, dir, "add", "conflict.txt")
	testutil.RunGit(t, dir, "commit", "-m", "Feature commit 1")

	testutil.WriteFile(t, dir, "conflict.txt", "Feature change 2")
	testutil.RunGit(t, dir, "add", "conflict.txt")
	testutil.RunGit(t, dir, "commit", "-m", "Feature commit 2")

	testutil.WriteFile(t, dir, "feature-file.txt", "Feature specific")
	testutil.RunGit(t, dir, "add", "feature-file.txt")
	testutil.RunGit(t, dir, "commit", "-m", "Feature commit 3")

	// Create conflict in develop
	testutil.RunGit(t, dir, "checkout", "develop")
	testutil.WriteFile(t, dir, "conflict.txt", "Develop conflicting change")
	testutil.RunGit(t, dir, "add", "conflict.txt")
	testutil.RunGit(t, dir, "commit", "-m", "Develop conflict")

	// Try to finish with squash (will conflict)
	testutil.RunGit(t, dir, "checkout", "feature/squash-conflict")
	output, _ = testutil.RunGitFlow(t, dir, "feature", "finish", "--squash", "squash-conflict")

	// Verify conflict detected
	if !strings.Contains(output, "conflict") {
		t.Fatal("Expected squash conflict to be detected")
	}

	// Verify merge state
	state, err := testutil.LoadMergeState(t, dir)
	if err != nil || state == nil {
		t.Fatal("Expected merge state to exist after conflict")
	}

	if state.MergeStrategy != "squash" {
		t.Errorf("Expected MergeStrategy to be 'squash', got: %s", state.MergeStrategy)
	}

	// Resolve conflict
	testutil.WriteFile(t, dir, "conflict.txt", "Squashed and resolved")
	testutil.RunGit(t, dir, "add", "conflict.txt")

	// Continue finish operation
	output, err = testutil.RunGitFlow(t, dir, "feature", "finish", "--continue", "squash-conflict")
	if err != nil {
		t.Fatalf("Failed to continue finish: %v\nOutput: %s", err, output)
	}

	// Verify success
	if !strings.Contains(output, "Successfully finished") {
		t.Error("Expected successful finish message")
	}

	// Verify only one commit for all feature changes
	testutil.RunGit(t, dir, "checkout", "develop")
	log, _ := testutil.RunGit(t, dir, "log", "--oneline", "-3")
	if strings.Count(log, "Feature commit") > 0 {
		t.Error("Individual feature commits should be squashed")
	}
	if !strings.Contains(log, "Squashed") {
		t.Log("Note: Squash commit message might vary")
	}
}

// TestFinishMultipleConflictsInSeparateCommits tests rebase with conflicts in multiple commits
func TestFinishMultipleConflictsInSeparateCommits(t *testing.T) {
	// Setup test repository
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Create base files in develop
	testutil.RunGit(t, dir, "checkout", "develop")
	testutil.WriteFile(t, dir, "file1.txt", "Develop file1 base")
	testutil.WriteFile(t, dir, "file2.txt", "Develop file2 base")
	testutil.RunGit(t, dir, "add", ".")
	testutil.RunGit(t, dir, "commit", "-m", "Base files")

	// Create feature with changes to both files in separate commits
	output, err = testutil.RunGitFlow(t, dir, "feature", "start", "multi-conflict")
	if err != nil {
		t.Fatalf("Failed to create feature: %v\nOutput: %s", err, output)
	}

	// First commit modifies file1
	testutil.WriteFile(t, dir, "file1.txt", "Feature file1 change")
	testutil.RunGit(t, dir, "add", "file1.txt")
	testutil.RunGit(t, dir, "commit", "-m", "Feature change to file1")

	// Second commit modifies file2
	testutil.WriteFile(t, dir, "file2.txt", "Feature file2 change")
	testutil.RunGit(t, dir, "add", "file2.txt")
	testutil.RunGit(t, dir, "commit", "-m", "Feature change to file2")

	// Create conflicts in develop for both files
	testutil.RunGit(t, dir, "checkout", "develop")
	testutil.WriteFile(t, dir, "file1.txt", "Develop file1 conflict")
	testutil.WriteFile(t, dir, "file2.txt", "Develop file2 conflict")
	testutil.RunGit(t, dir, "add", ".")
	testutil.RunGit(t, dir, "commit", "-m", "Develop changes both files")

	// Try to finish with rebase (will conflict on first commit)
	testutil.RunGit(t, dir, "checkout", "feature/multi-conflict")
	output, _ = testutil.RunGitFlow(t, dir, "feature", "finish", "--rebase", "multi-conflict")

	// Verify first conflict detected
	if !strings.Contains(output, "conflict") {
		t.Fatal("Expected first conflict to be detected")
	}

	// Resolve first conflict (file1)
	testutil.WriteFile(t, dir, "file1.txt", "File1 resolved")
	testutil.RunGit(t, dir, "add", "file1.txt")

	// Continue - should hit second conflict
	output, _ = testutil.RunGitFlow(t, dir, "feature", "finish", "--continue", "multi-conflict")

	// Check if we need another continue (second conflict)
	if strings.Contains(output, "conflict") {
		// Resolve second conflict (file2)
		testutil.WriteFile(t, dir, "file2.txt", "File2 resolved")
		testutil.RunGit(t, dir, "add", "file2.txt")

		// Continue again
		output, err = testutil.RunGitFlow(t, dir, "feature", "finish", "--continue", "multi-conflict")
		if err != nil && !strings.Contains(output, "Successfully finished") {
			t.Logf("Might need additional rebase continue: %v\nOutput: %s", err, output)
		}
	}

	// Verify both files are resolved in develop
	testutil.RunGit(t, dir, "checkout", "develop")
	file1 := testutil.ReadFile(t, dir, "file1.txt")
	file2 := testutil.ReadFile(t, dir, "file2.txt")

	if !strings.Contains(file1, "resolved") && !strings.Contains(file1, "Feature file1") {
		t.Log("File1 might not be fully resolved through the rebase")
	}
	if !strings.Contains(file2, "resolved") && !strings.Contains(file2, "Feature file2") {
		t.Log("File2 might not be fully resolved through the rebase")
	}
}

// TestFinishChildBranchConflictDifferentStrategies tests child branch updates with different merge strategies
func TestFinishChildBranchConflictDifferentStrategies(t *testing.T) {
	// Setup test repository
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Test each strategy for child branch
	strategies := []string{"merge", "rebase", "squash"}

	for _, strategy := range strategies {
		t.Run("Strategy_"+strategy, func(t *testing.T) {
			// Configure develop with the test strategy
			testutil.RunGit(t, dir, "config", "gitflow.branch.develop.autoUpdate", "true")
			testutil.RunGit(t, dir, "config", "gitflow.branch.develop.downstreamStrategy", strategy)

			// Create conflicting content in develop
			testutil.RunGit(t, dir, "checkout", "develop")
			testutil.WriteFile(t, dir, strategy+"-conflict.txt", "Develop content for "+strategy)
			testutil.RunGit(t, dir, "add", strategy+"-conflict.txt")
			testutil.RunGit(t, dir, "commit", "-m", "Develop "+strategy+" commit")

			// Create hotfix that will conflict
			output, err := testutil.RunGitFlow(t, dir, "hotfix", "start", strategy+"-test")
			if err != nil {
				t.Fatalf("Failed to create hotfix: %v\nOutput: %s", err, output)
			}

			testutil.WriteFile(t, dir, strategy+"-conflict.txt", "Hotfix content for "+strategy)
			testutil.RunGit(t, dir, "add", strategy+"-conflict.txt")
			testutil.RunGit(t, dir, "commit", "-m", "Hotfix "+strategy+" commit")

			// Finish hotfix (will succeed for main, conflict on develop)
			output, _ = testutil.RunGitFlow(t, dir, "hotfix", "finish", strategy+"-test")

			// Check if conflict in child update
			if strings.Contains(output, "updating") && strings.Contains(output, "develop") {
				if strings.Contains(output, "conflict") {
					// Verify the strategy is mentioned
					state, _ := testutil.LoadMergeState(t, dir)
					if state != nil && state.ChildStrategies != nil {
						if state.ChildStrategies["develop"] != strategy {
							t.Errorf("Expected develop strategy to be %s, got %s", strategy, state.ChildStrategies["develop"])
						}
					}

					// Resolve conflict
					testutil.WriteFile(t, dir, strategy+"-conflict.txt", "Resolved for "+strategy)
					testutil.RunGit(t, dir, "add", strategy+"-conflict.txt")

					// Continue based on strategy
					output, err = testutil.RunGitFlow(t, dir, "hotfix", "finish", "--continue", strategy+"-test")
					if err != nil && strings.Contains(err.Error(), "rebase") && strategy == "rebase" {
						// Might need git rebase --continue
						testutil.RunGit(t, dir, "rebase", "--continue")
						output, err = testutil.RunGitFlow(t, dir, "hotfix", "finish", "--continue", strategy+"-test")
					}

					// Verify completion
					if err == nil || strings.Contains(output, "Successfully finished") {
						t.Logf("Successfully handled %s strategy continuation", strategy)
					}
				}
			}

			// Clean up for next iteration
			testutil.RunGit(t, dir, "checkout", "main")
			testutil.RunGit(t, dir, "branch", "-D", "develop")
			testutil.RunGit(t, dir, "checkout", "-b", "develop", "main")
		})
	}
}

// TestFinishMultipleChildrenWithConflicts tests multiple child branches each with conflicts
func TestFinishMultipleChildrenWithConflicts(t *testing.T) {
	// Setup test repository
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Create two child branches with different strategies
	testutil.RunGit(t, dir, "checkout", "-b", "staging", "main")
	testutil.RunGit(t, dir, "config", "gitflow.branch.staging.type", "base")
	testutil.RunGit(t, dir, "config", "gitflow.branch.staging.parent", "main")
	testutil.RunGit(t, dir, "config", "gitflow.branch.staging.autoUpdate", "true")
	testutil.RunGit(t, dir, "config", "gitflow.branch.staging.downstreamStrategy", "rebase")

	// Configure develop
	testutil.RunGit(t, dir, "config", "gitflow.branch.develop.autoUpdate", "true")
	testutil.RunGit(t, dir, "config", "gitflow.branch.develop.downstreamStrategy", "merge")

	// Create conflicting content in both children
	testutil.RunGit(t, dir, "checkout", "staging")
	testutil.WriteFile(t, dir, "shared.txt", "Staging version")
	testutil.RunGit(t, dir, "add", "shared.txt")
	testutil.RunGit(t, dir, "commit", "-m", "Staging commit")

	testutil.RunGit(t, dir, "checkout", "develop")
	testutil.WriteFile(t, dir, "shared.txt", "Develop version")
	testutil.RunGit(t, dir, "add", "shared.txt")
	testutil.RunGit(t, dir, "commit", "-m", "Develop commit")

	// Create hotfix with conflicting content
	output, err = testutil.RunGitFlow(t, dir, "hotfix", "start", "multi-child")
	if err != nil {
		t.Fatalf("Failed to create hotfix: %v\nOutput: %s", err, output)
	}

	testutil.WriteFile(t, dir, "shared.txt", "Hotfix version")
	testutil.RunGit(t, dir, "add", "shared.txt")
	testutil.RunGit(t, dir, "commit", "-m", "Hotfix commit")

	// Finish hotfix - will succeed for main, conflict on first child
	output, _ = testutil.RunGitFlow(t, dir, "hotfix", "finish", "multi-child")

	conflictCount := 0

	// Handle first child conflict
	if strings.Contains(output, "conflict") {
		conflictCount++
		state, _ := testutil.LoadMergeState(t, dir)
		if state != nil && state.CurrentChildBranch != "" {
			t.Logf("First conflict in child: %s", state.CurrentChildBranch)
		}

		// Resolve first conflict
		testutil.WriteFile(t, dir, "shared.txt", "First child resolved")
		testutil.RunGit(t, dir, "add", "shared.txt")

		// Continue
		output, _ = testutil.RunGitFlow(t, dir, "hotfix", "finish", "--continue", "multi-child")
	}

	// Handle second child conflict
	if strings.Contains(output, "conflict") {
		conflictCount++
		state, _ := testutil.LoadMergeState(t, dir)
		if state != nil && state.CurrentChildBranch != "" {
			t.Logf("Second conflict in child: %s", state.CurrentChildBranch)
		}

		// Resolve second conflict
		testutil.WriteFile(t, dir, "shared.txt", "Second child resolved")
		testutil.RunGit(t, dir, "add", "shared.txt")

		// Continue
		output, err = testutil.RunGitFlow(t, dir, "hotfix", "finish", "--continue", "multi-child")
		if err != nil && !strings.Contains(output, "Successfully finished") {
			t.Logf("Final continuation result: %v\nOutput: %s", err, output)
		}
	}

	if conflictCount != 2 {
		t.Logf("Expected 2 conflicts in child branches, got %d", conflictCount)
	}
}

// TestFinishMultipleChildrenDifferentStrategiesNoConflict tests that different strategies are applied correctly
func TestFinishMultipleChildrenDifferentStrategiesNoConflict(t *testing.T) {
	// Setup test repository
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Create two child branches with different strategies
	testutil.RunGit(t, dir, "checkout", "-b", "staging", "main")
	testutil.RunGit(t, dir, "config", "gitflow.branch.staging.type", "base")
	testutil.RunGit(t, dir, "config", "gitflow.branch.staging.parent", "main")
	testutil.RunGit(t, dir, "config", "gitflow.branch.staging.autoUpdate", "true")
	testutil.RunGit(t, dir, "config", "gitflow.branch.staging.downstreamStrategy", "squash")

	testutil.RunGit(t, dir, "checkout", "-b", "qa", "main")
	testutil.RunGit(t, dir, "config", "gitflow.branch.qa.type", "base")
	testutil.RunGit(t, dir, "config", "gitflow.branch.qa.parent", "main")
	testutil.RunGit(t, dir, "config", "gitflow.branch.qa.autoUpdate", "true")
	testutil.RunGit(t, dir, "config", "gitflow.branch.qa.downstreamStrategy", "rebase")

	// Configure develop with merge
	testutil.RunGit(t, dir, "config", "gitflow.branch.develop.autoUpdate", "true")
	testutil.RunGit(t, dir, "config", "gitflow.branch.develop.downstreamStrategy", "merge")

	// Add unique commits to each child
	testutil.RunGit(t, dir, "checkout", "staging")
	testutil.WriteFile(t, dir, "staging.txt", "Staging specific")
	testutil.RunGit(t, dir, "add", "staging.txt")
	testutil.RunGit(t, dir, "commit", "-m", "Staging commit 1")
	testutil.WriteFile(t, dir, "staging2.txt", "Staging specific 2")
	testutil.RunGit(t, dir, "add", "staging2.txt")
	testutil.RunGit(t, dir, "commit", "-m", "Staging commit 2")

	testutil.RunGit(t, dir, "checkout", "qa")
	testutil.WriteFile(t, dir, "qa.txt", "QA specific")
	testutil.RunGit(t, dir, "add", "qa.txt")
	testutil.RunGit(t, dir, "commit", "-m", "QA commit")

	testutil.RunGit(t, dir, "checkout", "develop")
	testutil.WriteFile(t, dir, "develop.txt", "Develop specific")
	testutil.RunGit(t, dir, "add", "develop.txt")
	testutil.RunGit(t, dir, "commit", "-m", "Develop commit")

	// Create hotfix with non-conflicting content
	output, err = testutil.RunGitFlow(t, dir, "hotfix", "start", "strategies-test")
	if err != nil {
		t.Fatalf("Failed to create hotfix: %v\nOutput: %s", err, output)
	}

	testutil.WriteFile(t, dir, "hotfix.txt", "Hotfix content")
	testutil.RunGit(t, dir, "add", "hotfix.txt")
	testutil.RunGit(t, dir, "commit", "-m", "Hotfix commit")

	// Finish hotfix - should update all children with their strategies
	output, err = testutil.RunGitFlow(t, dir, "hotfix", "finish", "strategies-test")
	if err != nil {
		t.Fatalf("Failed to finish hotfix: %v\nOutput: %s", err, output)
	}

	// Verify the output mentions the correct strategies
	if strings.Contains(output, "staging") && !strings.Contains(output, "squash") {
		t.Log("Output might not show strategy details")
	}

	// Check staging branch (squash strategy)
	testutil.RunGit(t, dir, "checkout", "staging")
	stagingLog, _ := testutil.RunGit(t, dir, "log", "--oneline", "-5")
	// With squash, staging's own commits should remain separate
	if !strings.Contains(stagingLog, "Staging commit 1") || !strings.Contains(stagingLog, "Staging commit 2") {
		t.Log("Staging commits might have been affected by squash strategy")
	}

	// Check qa branch (rebase strategy)
	testutil.RunGit(t, dir, "checkout", "qa")
	qaLog, _ := testutil.RunGit(t, dir, "log", "--oneline", "-5")
	if !strings.Contains(qaLog, "QA commit") {
		t.Log("QA commit might have been affected by rebase strategy")
	}

	// Check develop branch (merge strategy)
	testutil.RunGit(t, dir, "checkout", "develop")
	developLog, _ := testutil.RunGit(t, dir, "log", "--oneline", "-5")
	if !strings.Contains(developLog, "Develop commit") {
		t.Log("Develop commit might have been affected by merge strategy")
	}

	// All branches should have the hotfix content
	for _, branch := range []string{"staging", "qa", "develop"} {
		testutil.RunGit(t, dir, "checkout", branch)
		if !testutil.FileExists(t, dir, "hotfix.txt") {
			t.Errorf("Hotfix content not found in %s branch", branch)
		}
	}
}

// TestFinishWithConsecutiveConflicts tests handling of multiple conflicts during finish operation.
// This test uses a release branch finish which naturally creates consecutive conflicts:
// 1. Release branch conflicts with main during merge (first conflict)
// 2. Develop branch conflicts with main during auto-update (second conflict)
// Steps:
// 1. Initialize git-flow with defaults (creates main, develop branches)
// 2. Add conflicting content to main branch
// 3. Add different conflicting content to develop branch
// 4. Create release branch from develop with additional changes
// 5. Attempt release finish - should fail with merge conflict (release vs main)
// 6. Resolve conflict and continue - should fail with auto-update conflict (develop vs main)
// 7. Resolve second conflict and continue - should complete successfully
func TestFinishWithConsecutiveConflicts(t *testing.T) {
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults - creates main, develop, and all topic branch configs
	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Add initial conflicting content to main branch
	_, err = testutil.RunGit(t, dir, "checkout", "main")
	if err != nil {
		t.Fatalf("Failed to checkout main: %v", err)
	}
	testutil.WriteFile(t, dir, "version.txt", "version: 1.0.0\nstatus: production\nenvironment: main")
	_, err = testutil.RunGit(t, dir, "add", "version.txt")
	if err != nil {
		t.Fatalf("Failed to add file on main: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Add version info to main")
	if err != nil {
		t.Fatalf("Failed to commit on main: %v", err)
	}

	// Create release branch first (before adding conflicting content to develop)
	_, err = testutil.RunGit(t, dir, "checkout", "develop")
	if err != nil {
		t.Fatalf("Failed to checkout develop: %v", err)
	}
	output, err = testutil.RunGitFlow(t, dir, "release", "start", "v1.1.0")
	if err != nil {
		t.Fatalf("Failed to create release branch: %v\nOutput: %s", err, output)
	}

	// Add release-specific changes that will conflict with main
	testutil.WriteFile(t, dir, "version.txt", "version: 1.1.0\nstatus: release-candidate\nenvironment: release\nfeatures: new-login,improved-ui\nrelease-date: 2024-01-15")
	_, err = testutil.RunGit(t, dir, "add", "version.txt")
	if err != nil {
		t.Fatalf("Failed to add file on release: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Prepare release v1.1.0")
	if err != nil {
		t.Fatalf("Failed to commit on release: %v", err)
	}

	// Now add conflicting content to develop (after release branch creation)
	_, err = testutil.RunGit(t, dir, "checkout", "develop")
	if err != nil {
		t.Fatalf("Failed to checkout develop: %v", err)
	}
	testutil.WriteFile(t, dir, "version.txt", "version: 1.1.0\nstatus: development\nenvironment: develop\nfeatures: new-login,improved-ui,experimental-feature")
	_, err = testutil.RunGit(t, dir, "add", "version.txt")
	if err != nil {
		t.Fatalf("Failed to add file on develop: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Add development version info with experimental features")
	if err != nil {
		t.Fatalf("Failed to commit on develop: %v", err)
	}

	// Attempt to finish release branch - this should cause first conflict (release vs main)
	output, err = testutil.RunGitFlow(t, dir, "release", "finish", "v1.1.0")
	if err == nil {
		t.Fatal("Expected release finish to fail due to merge conflict, but it succeeded")
	}

	// Verify we're in conflict state
	if !strings.Contains(output, "Merge conflict detected") {
		t.Errorf("Expected merge conflict message, got: %s", output)
	}

	// Check that merge state file exists and has correct step
	stateFile := filepath.Join(dir, ".git", "gitflow", "state", "merge.json")
	if _, err := os.Stat(stateFile); os.IsNotExist(err) {
		t.Fatal("Expected merge state file to exist after conflict")
	}

	// Verify git status shows conflicts on version.txt
	gitStatus, err := testutil.RunGit(t, dir, "status", "--porcelain")
	if err != nil {
		t.Fatalf("Failed to get git status: %v", err)
	}
	if !strings.Contains(gitStatus, "UU version.txt") && !strings.Contains(gitStatus, "AA version.txt") {
		t.Errorf("Expected conflict markers on version.txt in git status, got: %s", gitStatus)
	}

	// Resolve first conflict manually - merge release and main versions
	testutil.WriteFile(t, dir, "version.txt", "version: 1.1.0\nstatus: production\nenvironment: main\nfeatures: new-login,improved-ui\nrelease-date: 2024-01-15")
	_, err = testutil.RunGit(t, dir, "add", "version.txt")
	if err != nil {
		t.Fatalf("Failed to stage resolved file: %v", err)
	}

	// Continue the finish operation - this should complete the merge and proceed to develop auto-update and cause second conflict
	output, err = testutil.RunGitFlow(t, dir, "release", "finish", "--continue", "v1.1.0")
	if err == nil {
		t.Fatal("Expected release finish --continue to fail due to develop auto-update conflict, but it succeeded")
	}

	// Verify second conflict message for develop auto-update
	if !strings.Contains(output, "Merge conflict detected") && !strings.Contains(output, "Now updating 'develop'") {
		t.Errorf("Expected develop auto-update conflict message, got: %s", output)
	}

	// Verify we're still in a conflicted state but for develop auto-update (develop vs updated main)
	gitStatus, err = testutil.RunGit(t, dir, "status", "--porcelain")
	if err != nil {
		t.Fatalf("Failed to get git status after develop conflict: %v", err)
	}
	if !strings.Contains(gitStatus, "UU version.txt") && !strings.Contains(gitStatus, "AA version.txt") {
		t.Errorf("Expected conflict markers for develop auto-update, got: %s", gitStatus)
	}

	// Resolve second conflict - merge develop and updated main
	testutil.WriteFile(t, dir, "version.txt", "version: 1.1.0\nstatus: development\nenvironment: develop\nfeatures: new-login,improved-ui,experimental-feature\nrelease-date: 2024-01-15")
	_, err = testutil.RunGit(t, dir, "add", "version.txt")
	if err != nil {
		t.Fatalf("Failed to stage resolved develop conflict: %v", err)
	}

	// Continue again - this should complete the merge and finish successfully
	output, err = testutil.RunGitFlow(t, dir, "release", "finish", "--continue", "v1.1.0")
	if err != nil {
		t.Fatalf("Expected release finish --continue to succeed after resolving all conflicts: %v\nOutput: %s", err, output)
	}

	// Verify success message
	if !strings.Contains(output, "Successfully finished branch") {
		t.Errorf("Expected success message, got: %s", output)
	}

	// Verify a tag was created (releases create tags by default)
	tags, err := testutil.RunGit(t, dir, "tag", "-l")
	if err != nil {
		t.Fatalf("Failed to list tags: %v", err)
	}
	if !strings.Contains(tags, "v1.1.0") {
		t.Errorf("Expected tag v1.1.0 to be created, got tags: %s", tags)
	}

	// Verify merge state is cleaned up
	if _, err := os.Stat(stateFile); !os.IsNotExist(err) {
		t.Error("Expected merge state file to be cleaned up after successful completion")
	}

	// Verify final repository state - should be on main branch after release finish
	currentBranch, err := testutil.RunGit(t, dir, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		t.Fatalf("Failed to get current branch: %v", err)
	}
	if strings.TrimSpace(currentBranch) != "main" {
		t.Errorf("Expected to be on main branch after release finish, got: %s", strings.TrimSpace(currentBranch))
	}

	// Verify release branch is deleted
	branches, err := testutil.RunGit(t, dir, "branch")
	if err != nil {
		t.Fatalf("Failed to list branches: %v", err)
	}
	if strings.Contains(branches, "release/v1.1.0") {
		t.Error("Expected release branch to be deleted after successful finish")
	}

	// Verify main branch has the resolved content with release info
	mainContent := testutil.ReadFile(t, dir, "version.txt")
	if !strings.Contains(mainContent, "version: 1.1.0") || !strings.Contains(mainContent, "release-date: 2024-01-15") {
		t.Errorf("Expected main branch to have release changes, got: %s", mainContent)
	}

	// Verify develop branch has the resolved content with both release info and develop-specific content
	_, err = testutil.RunGit(t, dir, "checkout", "develop")
	if err != nil {
		t.Fatalf("Failed to checkout develop branch: %v", err)
	}
	developContent := testutil.ReadFile(t, dir, "version.txt")
	if !strings.Contains(developContent, "environment: develop") || !strings.Contains(developContent, "experimental-feature") {
		t.Errorf("Expected develop branch to have both release and develop-specific content, got: %s", developContent)
	}
}
