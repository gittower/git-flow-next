package cmd_test

import (
	"bytes"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gittower/git-flow-next/test/testutil"
)

// TestFinishFeatureBranchDefaultLocalDeletion tests that feature branches are deleted locally by default.
// Steps:
// 1. Sets up a test repository and initializes git-flow
// 2. Creates a feature branch
// 3. Adds changes to the feature branch
// 4. Finishes the feature branch
// 5. Verifies the local branch is deleted
func TestFinishFeatureBranchDefaultLocalDeletion(t *testing.T) {
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults and create branches
	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
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

	// Verify that local feature branch is deleted
	if testutil.BranchExists(t, dir, "feature/my-feature") {
		t.Error("Expected local feature branch to be deleted by default")
	}
}

// TestFinishFeatureBranchDefaultRemoteDeletion tests that feature branches are deleted both locally and remotely by default.
// Steps:
// 1. Sets up a test repository and initializes git-flow
// 2. Creates a feature branch
// 3. Adds changes to the feature branch
// 4. Adds a remote repository
// 5. Finishes the feature branch
// 6. Verifies both local and remote branches are deleted
func TestFinishFeatureBranchDefaultRemoteDeletion(t *testing.T) {
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults and create branches
	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
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

	// Add a remote repository
	remoteDir, err := testutil.AddRemote(t, dir, "origin", true)
	if err != nil {
		t.Fatalf("Failed to add remote: %v", err)
	}
	defer testutil.CleanupTestRepo(t, remoteDir)

	// Push the feature branch to remote
	_, err = testutil.RunGit(t, dir, "push", "origin", "feature/my-feature")
	if err != nil {
		t.Fatalf("Failed to push feature branch: %v", err)
	}

	// Finish the feature branch
	output, err = testutil.RunGitFlow(t, dir, "feature", "finish", "my-feature")
	if err != nil {
		t.Fatalf("Failed to finish feature branch: %v\nOutput: %s", err, output)
	}

	// Verify that local feature branch is deleted
	if testutil.BranchExists(t, dir, "feature/my-feature") {
		t.Error("Expected local feature branch to be deleted by default")
	}

	// Verify that remote feature branch is deleted
	_, err = testutil.RunGit(t, dir, "fetch", "origin")
	if err != nil {
		t.Fatalf("Failed to fetch from remote: %v", err)
	}
	if testutil.BranchExists(t, dir, "origin/feature/my-feature") {
		t.Error("Expected remote feature branch to be deleted by default")
	}
}

// TestFinishFeatureBranchKeepLocal tests that the keep-local option preserves the local branch when finishing.
// Steps:
// 1. Sets up a test repository and initializes git-flow
// 2. Creates a feature branch
// 3. Adds changes to the feature branch
// 4. Finishes the feature branch with the keeplocal option
// 5. Verifies the branch is merged into develop
// 6. Verifies the local feature branch is preserved
func TestFinishFeatureBranchKeepLocal(t *testing.T) {
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Create a feature branch
	output, err = testutil.RunGitFlow(t, dir, "feature", "start", "keep-local-test")
	if err != nil {
		t.Fatalf("Failed to create feature branch: %v\nOutput: %s", err, output)
	}

	// Create a test file
	testutil.WriteFile(t, dir, "test.txt", "feature content")

	// Commit the changes
	_, err = testutil.RunGit(t, dir, "add", "test.txt")
	if err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Add test file")
	if err != nil {
		t.Fatalf("Failed to commit file: %v", err)
	}

	// Get path to the git-flow binary
	gitFlowPath, err := filepath.Abs(filepath.Join("..", "..", "git-flow"))
	if err != nil {
		t.Fatalf("Failed to get absolute path to git-flow: %v", err)
	}

	// Finish the feature branch with keeplocal option
	cmd := exec.Command(gitFlowPath, "feature", "finish", "keep-local-test", "--keeplocal")
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()
	if err != nil {
		t.Fatalf("Failed to finish feature branch: %v\nOutput: %s", err, stdout.String()+stderr.String())
	}
	output = stdout.String() + stderr.String()

	// Verify that we're now on develop branch
	currentBranch := testutil.GetCurrentBranch(t, dir)
	if currentBranch != "develop" {
		t.Errorf("Expected to be on develop branch after finish, got %s", currentBranch)
	}

	// Verify that the changes are in develop
	if !testutil.FileExists(t, dir, "test.txt") {
		t.Error("Expected test.txt to exist in develop branch")
	}

	// Verify that the local branch still exists
	if !testutil.BranchExists(t, dir, "feature/keep-local-test") {
		t.Error("Expected feature branch to still exist with --keeplocal option")
	}

	// Checkout the feature branch and verify it still has the content
	_, err = testutil.RunGit(t, dir, "checkout", "feature/keep-local-test")
	if err != nil {
		t.Fatalf("Failed to checkout feature branch: %v", err)
	}

	// Verify the file content in the feature branch
	content := testutil.ReadFile(t, dir, "test.txt")
	if content != "feature content" {
		t.Errorf("Expected file content to be 'feature content', got '%s'", content)
	}
}

// TestFinishFeatureBranchKeepRemote tests that the keep-remote option preserves the remote branch when finishing.
// Steps:
// 1. Sets up a test repository and initializes git-flow
// 2. Creates a feature branch
// 3. Adds changes to the feature branch
// 4. Adds a remote and pushes the branch
// 5. Finishes the feature branch with the keepremote option
// 6. Verifies the branch is merged into develop
// 7. Verifies the local feature branch is deleted
// 8. Verifies the remote feature branch is preserved
func TestFinishFeatureBranchKeepRemote(t *testing.T) {
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Create a feature branch
	output, err = testutil.RunGitFlow(t, dir, "feature", "start", "keep-remote-test")
	if err != nil {
		t.Fatalf("Failed to create feature branch: %v\nOutput: %s", err, output)
	}

	// Create a test file
	testutil.WriteFile(t, dir, "test.txt", "feature content")

	// Commit the changes
	_, err = testutil.RunGit(t, dir, "add", "test.txt")
	if err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Add test file")
	if err != nil {
		t.Fatalf("Failed to commit file: %v", err)
	}

	// Add a remote repository
	remoteDir, err := testutil.AddRemote(t, dir, "origin", true)
	if err != nil {
		t.Fatalf("Failed to add remote: %v", err)
	}
	defer testutil.CleanupTestRepo(t, remoteDir)

	// Push the feature branch to remote
	_, err = testutil.RunGit(t, dir, "push", "origin", "feature/keep-remote-test")
	if err != nil {
		t.Fatalf("Failed to push feature branch: %v", err)
	}

	// Get path to the git-flow binary
	gitFlowPath, err := filepath.Abs(filepath.Join("..", "..", "git-flow"))
	if err != nil {
		t.Fatalf("Failed to get absolute path to git-flow: %v", err)
	}

	// Finish the feature branch with keepremote option
	cmd := exec.Command(gitFlowPath, "feature", "finish", "keep-remote-test", "--keepremote")
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()
	if err != nil {
		t.Fatalf("Failed to finish feature branch: %v\nOutput: %s", err, stdout.String()+stderr.String())
	}
	output = stdout.String() + stderr.String()

	// Verify that we're now on develop branch
	currentBranch := testutil.GetCurrentBranch(t, dir)
	if currentBranch != "develop" {
		t.Errorf("Expected to be on develop branch after finish, got %s", currentBranch)
	}

	// Verify that the changes are in develop
	if !testutil.FileExists(t, dir, "test.txt") {
		t.Error("Expected test.txt to exist in develop branch")
	}

	// Verify that the local branch is deleted
	if testutil.BranchExists(t, dir, "feature/keep-remote-test") {
		t.Error("Expected local feature branch to be deleted")
	}

	// Fetch from remote to update references
	_, err = testutil.RunGit(t, dir, "fetch", "origin")
	if err != nil {
		t.Fatalf("Failed to fetch from remote: %v", err)
	}

	// Verify that the remote branch still exists
	remoteExists := testutil.RemoteBranchExists(t, dir, "origin", "feature/keep-remote-test")
	if !remoteExists {
		t.Error("Expected remote feature branch to still exist with --keepremote option")
	}

	// Try to checkout the remote branch and verify it has the content
	_, err = testutil.RunGit(t, dir, "checkout", "-b", "verify-remote", "origin/feature/keep-remote-test")
	if err != nil {
		t.Fatalf("Failed to checkout remote branch: %v", err)
	}

	// Verify the file content from the remote branch
	content := testutil.ReadFile(t, dir, "test.txt")
	if content != "feature content" {
		t.Errorf("Expected file content to be 'feature content', got '%s'", content)
	}
}

// TestFinishDeleteBranchUsesConfiguredRemote tests that branch deletion uses the configured remote name.
// Steps:
// 1. Sets up a test repository and initializes git-flow with defaults
// 2. Adds a remote named "upstream" (not "origin")
// 3. Configures gitflow.origin=upstream so git-flow uses the custom remote
// 4. Creates a feature branch and adds a commit
// 5. Publishes the feature branch to "upstream"
// 6. Runs 'git flow feature finish'
// 7. Verifies the remote branch is deleted from "upstream"
// 8. Verifies finish completes successfully
func TestFinishDeleteBranchUsesConfiguredRemote(t *testing.T) {
	// Setup test repository
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	_, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v", err)
	}

	// Add a remote named "upstream" (not "origin")
	remoteDir, err := testutil.AddRemote(t, dir, "upstream", true)
	if err != nil {
		t.Fatalf("Failed to add upstream remote: %v", err)
	}
	defer testutil.CleanupTestRepo(t, remoteDir)

	// Configure git-flow to use "upstream" as the remote
	_, err = testutil.RunGit(t, dir, "config", "gitflow.origin", "upstream")
	if err != nil {
		t.Fatalf("Failed to configure gitflow.origin: %v", err)
	}

	// Create a feature branch
	_, err = testutil.RunGitFlow(t, dir, "feature", "start", "custom-remote-test")
	if err != nil {
		t.Fatalf("Failed to create feature branch: %v", err)
	}

	// Create a test file and commit
	testutil.WriteFile(t, dir, "custom-remote-test.txt", "test content")
	_, err = testutil.RunGit(t, dir, "add", "custom-remote-test.txt")
	if err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Add custom remote test file")
	if err != nil {
		t.Fatalf("Failed to commit file: %v", err)
	}

	// Publish the feature branch to "upstream"
	_, err = testutil.RunGitFlow(t, dir, "feature", "publish", "custom-remote-test")
	if err != nil {
		t.Fatalf("Failed to publish feature branch: %v", err)
	}

	// Verify the remote branch exists on "upstream" before finish
	_, err = testutil.RunGit(t, dir, "fetch", "upstream")
	if err != nil {
		t.Fatalf("Failed to fetch from upstream: %v", err)
	}
	if !testutil.RemoteBranchExists(t, dir, "upstream", "feature/custom-remote-test") {
		t.Fatal("Expected remote branch to exist on upstream before finish")
	}

	// Finish the feature branch
	output, err := testutil.RunGitFlow(t, dir, "feature", "finish", "custom-remote-test")
	if err != nil {
		t.Fatalf("Failed to finish feature branch: %v\nOutput: %s", err, output)
	}

	// Verify finish completed successfully
	if !strings.Contains(output, "Successfully finished") {
		t.Error("Expected successful finish message")
	}

	// Verify local branch is deleted
	if testutil.BranchExists(t, dir, "feature/custom-remote-test") {
		t.Error("Expected local feature branch to be deleted")
	}

	// Fetch from upstream and verify remote branch is deleted
	_, err = testutil.RunGit(t, dir, "fetch", "upstream", "--prune")
	if err != nil {
		t.Fatalf("Failed to fetch from upstream: %v", err)
	}
	if testutil.RemoteBranchExists(t, dir, "upstream", "feature/custom-remote-test") {
		t.Error("Expected remote branch to be deleted from upstream")
	}
}

// TestFinishClearsMergeStateWhenBranchDeletionFails tests that merge state is cleared even when
// remote branch deletion fails during finish.
// Steps:
// 1. Sets up a test repository with a remote and initializes git-flow
// 2. Creates a feature branch with changes and pushes to remote
// 3. Configures the remote repository to reject branch deletions so deletion will fail
// 4. Finishes the feature branch (merge succeeds, remote deletion fails)
// 5. Verifies the merge state file does not exist (cleared despite deletion error)
// 6. Verifies the merge completed successfully (changes are on develop)
func TestFinishClearsMergeStateWhenBranchDeletionFails(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Add a remote repository
	remoteDir, err := testutil.AddRemote(t, dir, "origin", true)
	if err != nil {
		t.Fatalf("Failed to add remote: %v", err)
	}
	defer testutil.CleanupTestRepo(t, remoteDir)

	// Create a feature branch with changes
	output, err = testutil.RunGitFlow(t, dir, "feature", "start", "delete-fail-test")
	if err != nil {
		t.Fatalf("Failed to start feature branch: %v\nOutput: %s", err, output)
	}

	testutil.WriteFile(t, dir, "feature-file.txt", "feature content")
	_, err = testutil.RunGit(t, dir, "add", "feature-file.txt")
	if err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Add feature file")
	if err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Push feature branch to remote
	_, err = testutil.RunGit(t, dir, "push", "origin", "feature/delete-fail-test")
	if err != nil {
		t.Fatalf("Failed to push feature branch: %v", err)
	}

	// Configure the remote bare repo to reject branch deletions
	_, err = testutil.RunGit(t, remoteDir, "config", "receive.denyDeletes", "true")
	if err != nil {
		t.Fatalf("Failed to configure receive.denyDeletes: %v", err)
	}

	// Finish the feature branch — merge should succeed but remote deletion should fail
	output, err = testutil.RunGitFlow(t, dir, "feature", "finish", "delete-fail-test")
	_ = output // finish may or may not return an error depending on how deletion failure is handled

	// Verify remote branch still exists (confirms deletion was actually rejected)
	remoteRefs, err := testutil.RunGit(t, dir, "ls-remote", "--heads", "origin", "feature/delete-fail-test")
	if err != nil {
		t.Fatalf("Failed to list remote refs: %v", err)
	}
	if remoteRefs == "" {
		t.Fatal("Expected remote branch feature/delete-fail-test to still exist, but it was deleted")
	}

	// KEY ASSERTION: merge state must be cleared even though deletion failed
	if testutil.GitFlowMergeStateExists(t, dir) {
		t.Error("Expected merge state to be cleared after finish, but merge.json still exists")
	}

	// Verify the merge itself completed — feature changes should be on develop
	_, err = testutil.RunGit(t, dir, "checkout", "develop")
	if err != nil {
		t.Fatalf("Failed to checkout develop: %v", err)
	}
	if !testutil.FileExists(t, dir, "feature-file.txt") {
		t.Error("Expected feature-file.txt to exist on develop after merge")
	}
}
