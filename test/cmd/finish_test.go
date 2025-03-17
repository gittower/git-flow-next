package cmd_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gittower/git-flow-next/test/testutil"
)

// Test functions

// TestFinishFeatureBranch tests the finish command for feature branches
func TestFinishFeatureBranch(t *testing.T) {
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults and create branches
	output, err := testutil.RunGitFlow(t, dir, "init", "-d", "-c")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Create a feature branch
	output, err = testutil.RunGitFlow(t, dir, "feature", "start", "my-feature")
	if err != nil {
		t.Fatalf("Failed to create feature branch: %v\nOutput: %s", err, output)
	}

	// Create a test file
	testutil.WriteFile(t, dir, "test.txt", "test content")

	// Commit the changes
	_, err = testutil.RunGit(t, dir, "add", "test.txt")
	if err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Add test file")
	if err != nil {
		t.Fatalf("Failed to commit file: %v", err)
	}

	// Finish the feature branch
	output, err = testutil.RunGitFlow(t, dir, "feature", "finish", "my-feature")
	if err != nil {
		t.Fatalf("Failed to finish feature branch: %v\nOutput: %s", err, output)
	}

	// Verify that feature branch is deleted
	if testutil.BranchExists(t, dir, "feature/my-feature") {
		t.Error("Expected feature branch to be deleted")
	}

	// Verify that changes are merged into develop
	_, err = testutil.RunGit(t, dir, "checkout", "develop")
	if err != nil {
		t.Fatalf("Failed to checkout develop: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, "test.txt")); os.IsNotExist(err) {
		t.Error("Expected test.txt to exist in develop branch")
	}
}

// TestFinishReleaseAndHotfixBranches tests the finish command for release and hotfix branches
func TestFinishReleaseAndHotfixBranches(t *testing.T) {
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults and create branches
	output, err := testutil.RunGitFlow(t, dir, "init", "-d", "-c")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Test release branch
	output, err = testutil.RunGitFlow(t, dir, "release", "start", "1.0.0")
	if err != nil {
		t.Fatalf("Failed to create release branch: %v\nOutput: %s", err, output)
	}

	// Create a test file
	testutil.WriteFile(t, dir, "release.txt", "release content")

	// Commit the changes
	_, err = testutil.RunGit(t, dir, "add", "release.txt")
	if err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Add release file")
	if err != nil {
		t.Fatalf("Failed to commit file: %v", err)
	}

	// Finish the release branch
	output, err = testutil.RunGitFlow(t, dir, "release", "finish", "1.0.0")
	if err != nil {
		t.Fatalf("Failed to finish release branch: %v\nOutput: %s", err, output)
	}

	// Test hotfix branch
	output, err = testutil.RunGitFlow(t, dir, "hotfix", "start", "1.0.1")
	if err != nil {
		t.Fatalf("Failed to create hotfix branch: %v\nOutput: %s", err, output)
	}

	// Create a test file
	testutil.WriteFile(t, dir, "hotfix.txt", "hotfix content")

	// Commit the changes
	_, err = testutil.RunGit(t, dir, "add", "hotfix.txt")
	if err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Add hotfix file")
	if err != nil {
		t.Fatalf("Failed to commit file: %v", err)
	}

	// Finish the hotfix branch
	output, err = testutil.RunGitFlow(t, dir, "hotfix", "finish", "1.0.1")
	if err != nil {
		t.Fatalf("Failed to finish hotfix branch: %v\nOutput: %s", err, output)
	}
}

// TestFinishWithCustomConfig tests the finish command with custom configuration
func TestFinishWithCustomConfig(t *testing.T) {
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with custom configuration and create branches
	input := "custom-main\ncustom-dev\nf/\nr/\nh/\ns/\ny\n"
	output, err := testutil.RunGitFlowWithInput(t, dir, input, "init", "-c")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Create a feature branch
	output, err = testutil.RunGitFlow(t, dir, "feature", "start", "custom-feature")
	if err != nil {
		t.Fatalf("Failed to create feature branch: %v\nOutput: %s", err, output)
	}

	// Create a test file
	testutil.WriteFile(t, dir, "test.txt", "test content")

	// Commit the changes
	_, err = testutil.RunGit(t, dir, "add", "test.txt")
	if err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Add test file")
	if err != nil {
		t.Fatalf("Failed to commit file: %v", err)
	}

	// Finish the feature branch
	output, err = testutil.RunGitFlow(t, dir, "feature", "finish", "custom-feature")
	if err != nil {
		t.Fatalf("Failed to finish feature branch: %v\nOutput: %s", err, output)
	}

	// Verify that feature branch is deleted
	if testutil.BranchExists(t, dir, "f/custom-feature") {
		t.Error("Expected feature branch to be deleted")
	}

	// Verify that changes are merged into custom-dev
	_, err = testutil.RunGit(t, dir, "checkout", "custom-dev")
	if err != nil {
		t.Fatalf("Failed to checkout custom-dev: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, "test.txt")); os.IsNotExist(err) {
		t.Error("Expected test.txt to exist in custom-dev branch")
	}
}

// TestFinishNonExistentBranch tests the finish command with a non-existent branch
func TestFinishNonExistentBranch(t *testing.T) {
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults and create branches
	output, err := testutil.RunGitFlow(t, dir, "init", "-d", "-c")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Try to finish a non-existent feature branch
	output, err = testutil.RunGitFlow(t, dir, "feature", "finish", "non-existent")
	if err == nil {
		t.Fatal("Expected error when finishing non-existent branch")
	}

	// Check if the error message is appropriate
	if !strings.Contains(output, "does not exist") {
		t.Errorf("Expected error message to contain 'does not exist', got: %s", output)
	}
}

// TestFinishWithMergeConflict tests that the finish command properly handles merge conflicts
func TestFinishWithMergeConflict(t *testing.T) {
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults and create branches
	output, err := testutil.RunGitFlow(t, dir, "init", "-d", "-c")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Set merge strategy to merge using Git config
	_, err = testutil.RunGit(t, dir, "config", "gitflow.branch.feature.merge-strategy", "merge")
	if err != nil {
		t.Fatalf("Failed to set merge strategy: %v", err)
	}

	// Create feature branch
	output, err = testutil.RunGitFlow(t, dir, "feature", "start", "conflict-test")
	if err != nil {
		t.Fatalf("Failed to create feature branch: %v\nOutput: %s", err, output)
	}

	// Create and modify file in feature branch
	testutil.WriteFile(t, dir, "conflict.txt", "content from feature branch")
	_, err = testutil.RunGit(t, dir, "add", "conflict.txt")
	if err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Add conflict.txt in feature")
	if err != nil {
		t.Fatalf("Failed to commit file: %v", err)
	}

	// Switch to develop and modify the same file
	_, err = testutil.RunGit(t, dir, "checkout", "develop")
	if err != nil {
		t.Fatalf("Failed to checkout develop: %v", err)
	}

	testutil.WriteFile(t, dir, "conflict.txt", "content from develop branch")
	_, err = testutil.RunGit(t, dir, "add", "conflict.txt")
	if err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Add conflict.txt in develop")
	if err != nil {
		t.Fatalf("Failed to commit file: %v", err)
	}

	// Switch back to feature branch
	_, err = testutil.RunGit(t, dir, "checkout", "feature/conflict-test")
	if err != nil {
		t.Fatalf("Failed to checkout feature branch: %v", err)
	}

	// Try to finish the feature branch
	output, err = testutil.RunGitFlow(t, dir, "feature", "finish", "conflict-test")
	if err == nil {
		t.Fatalf("Expected finish command to fail due to merge conflict, but it succeeded. Output: %s", output)
	}

	// Check if merge state file exists
	if _, err := os.Stat(filepath.Join(dir, ".git", "gitflow", "state", "merge.json")); os.IsNotExist(err) {
		t.Error("Expected merge state file to exist")
	}

	// Read merge state file and verify its contents
	stateFile := filepath.Join(dir, ".git", "gitflow", "state", "merge.json")
	data, err := os.ReadFile(stateFile)
	if err != nil {
		t.Fatalf("Failed to read merge state file: %v", err)
	}

	var state struct {
		Action         string `json:"action"`
		BranchType     string `json:"branchType"`
		BranchName     string `json:"branchName"`
		CurrentStep    string `json:"currentStep"`
		ParentBranch   string `json:"parentBranch"`
		MergeStrategy  string `json:"mergeStrategy"`
		FullBranchName string `json:"fullBranchName"`
	}

	if err := json.Unmarshal(data, &state); err != nil {
		t.Fatalf("Failed to parse merge state file: %v", err)
	}

	// Verify merge state contents
	if state.Action != "finish" {
		t.Errorf("Expected action to be 'finish', got '%s'", state.Action)
	}
	if state.BranchType != "feature" {
		t.Errorf("Expected branchType to be 'feature', got '%s'", state.BranchType)
	}
	if state.BranchName != "conflict-test" {
		t.Errorf("Expected branchName to be 'conflict-test', got '%s'", state.BranchName)
	}
	if state.CurrentStep != "merge" {
		t.Errorf("Expected currentStep to be 'merge', got '%s'", state.CurrentStep)
	}
	if state.ParentBranch != "develop" {
		t.Errorf("Expected parentBranch to be 'develop', got '%s'", state.ParentBranch)
	}
	if state.MergeStrategy != "merge" {
		t.Errorf("Expected mergeStrategy to be 'merge', got '%s'", state.MergeStrategy)
	}
	if state.FullBranchName != "feature/conflict-test" {
		t.Errorf("Expected fullBranchName to be 'feature/conflict-test', got '%s'", state.FullBranchName)
	}
}

func TestFinishWithMergeAbort(t *testing.T) {
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults and create branches
	output, err := testutil.RunGitFlow(t, dir, "init", "-d", "-c")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Create a file in develop
	testutil.WriteFile(t, dir, "test.txt", "develop content")

	// Commit the file in develop
	_, err = testutil.RunGit(t, dir, "add", "test.txt")
	if err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Add test.txt in develop")
	if err != nil {
		t.Fatalf("Failed to commit file: %v", err)
	}

	// Create and switch to feature branch
	output, err = testutil.RunGitFlow(t, dir, "feature", "start", "abort-feature")
	if err != nil {
		t.Fatalf("Failed to create feature branch: %v\nOutput: %s", err, output)
	}

	// Modify the same file in feature branch
	testutil.WriteFile(t, dir, "test.txt", "feature content")

	// Commit the changes in feature branch
	_, err = testutil.RunGit(t, dir, "add", "test.txt")
	if err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Modify test.txt in feature")
	if err != nil {
		t.Fatalf("Failed to commit file: %v", err)
	}

	// Try to finish the feature branch (should fail due to conflict)
	output, err = testutil.RunGitFlow(t, dir, "feature", "finish", "abort-feature")
	if err == nil {
		t.Fatal("Expected finish to fail due to merge conflict")
	}

	// Abort the merge
	output, err = testutil.RunGitFlow(t, dir, "feature", "finish", "--abort", "abort-feature")
	if err != nil {
		t.Fatalf("Failed to abort merge: %v\nOutput: %s", err, output)
	}

	// Verify we're back on the feature branch
	currentBranch := testutil.GetCurrentBranch(t, dir)
	if !strings.Contains(currentBranch, "abort-feature") {
		t.Errorf("Expected to be back on feature branch after abort, got %s", currentBranch)
	}
}

func TestFinishWithRebaseConflict(t *testing.T) {
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults and create branches
	output, err := testutil.RunGitFlow(t, dir, "init", "-d", "-c")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Create a file in develop
	testutil.WriteFile(t, dir, "test.txt", "develop content")

	// Commit the file in develop
	_, err = testutil.RunGit(t, dir, "add", "test.txt")
	if err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Add test.txt in develop")
	if err != nil {
		t.Fatalf("Failed to commit file: %v", err)
	}

	// Create and switch to feature branch
	output, err = testutil.RunGitFlow(t, dir, "feature", "start", "rebase-feature")
	if err != nil {
		t.Fatalf("Failed to create feature branch: %v\nOutput: %s", err, output)
	}

	// Modify the same file in feature branch
	testutil.WriteFile(t, dir, "test.txt", "feature content")

	// Commit the changes in feature branch
	_, err = testutil.RunGit(t, dir, "add", "test.txt")
	if err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Modify test.txt in feature")
	if err != nil {
		t.Fatalf("Failed to commit file: %v", err)
	}

	// Try to finish the feature branch with rebase
	output, err = testutil.RunGitFlow(t, dir, "feature", "finish", "--rebase", "rebase-feature")
	if err == nil {
		t.Fatal("Expected finish to fail due to rebase conflict")
	}

	// Verify that we're in a rebase conflict state
	currentBranch := testutil.GetCurrentBranch(t, dir)
	if !strings.Contains(currentBranch, "rebase-feature") {
		t.Errorf("Expected to be on feature branch during rebase conflict, got %s", currentBranch)
	}
}
