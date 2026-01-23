package cmd_test

import (
	"strings"
	"testing"

	"github.com/gittower/git-flow-next/test/testutil"
)

// TestPublishFeatureBranchWithPushOption tests publishing a feature branch with push-option configured.
// Steps:
// 1. Sets up a test repository with a remote
// 2. Configures push-option for feature branch type
// 3. Creates a feature branch
// 4. Publishes the branch
// 5. Verifies the branch was published successfully
func TestPublishFeatureBranchWithPushOption(t *testing.T) {
	// Setup test repo with remote
	dir, remoteDir := testutil.SetupTestRepoWithRemote(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer testutil.CleanupTestRepo(t, remoteDir)

	// Configure push-option for feature branch type
	_, err := testutil.RunGit(t, dir, "config", "gitflow.branch.feature.pushOption", "ci.skip")
	if err != nil {
		t.Fatalf("Failed to set push-option config: %v", err)
	}

	// Create a feature branch
	output, err := testutil.RunGitFlow(t, dir, "feature", "start", "test-push-option")
	if err != nil {
		t.Fatalf("Failed to start feature: %v\nOutput: %s", err, output)
	}

	// Publish the feature
	output, err = testutil.RunGitFlow(t, dir, "feature", "publish", "test-push-option")
	if err != nil {
		t.Fatalf("Failed to publish feature: %v\nOutput: %s", err, output)
	}

	// Verify output contains success message
	if !strings.Contains(output, "Successfully published 'feature/test-push-option'") {
		t.Errorf("Expected success message in output, got: %s", output)
	}

	// Verify remote branch exists
	_, err = testutil.RunGit(t, dir, "fetch", "origin")
	if err != nil {
		t.Fatalf("Failed to fetch: %v", err)
	}

	if !testutil.RemoteBranchExists(t, dir, "origin", "feature/test-push-option") {
		t.Error("Expected remote branch 'origin/feature/test-push-option' to exist")
	}
}

// TestPublishWithConfigAddTopicPushOption tests adding a topic type with --push-option flag.
// Steps:
// 1. Sets up a test repository with a remote
// 2. Uses 'config add topic' command with --push-option flag
// 3. Verifies the configuration is saved correctly
// 4. Creates a branch of that type and publishes it
// 5. Verifies the branch was published successfully
func TestPublishWithConfigAddTopicPushOption(t *testing.T) {
	// Setup test repo with remote
	dir, remoteDir := testutil.SetupTestRepoWithRemote(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer testutil.CleanupTestRepo(t, remoteDir)

	// Add custom topic type with --push-option flag
	output, err := testutil.RunGitFlow(t, dir, "config", "add", "topic", "bugfix", "develop",
		"--prefix=bugfix/", "--push-option=ci.skip")
	if err != nil {
		t.Fatalf("Failed to add topic with --push-option: %v\nOutput: %s", err, output)
	}

	// Verify the push-option was saved to config
	configOutput, err := testutil.RunGit(t, dir, "config", "gitflow.branch.bugfix.pushOption")
	if err != nil {
		t.Fatalf("Failed to read push-option config: %v", err)
	}

	if !strings.Contains(configOutput, "ci.skip") {
		t.Errorf("Expected push-option 'ci.skip' in config, got: %s", configOutput)
	}

	// Create a bugfix branch
	output, err = testutil.RunGitFlow(t, dir, "bugfix", "start", "urgent-fix")
	if err != nil {
		t.Fatalf("Failed to start bugfix: %v\nOutput: %s", err, output)
	}

	// Publish the bugfix
	output, err = testutil.RunGitFlow(t, dir, "bugfix", "publish", "urgent-fix")
	if err != nil {
		t.Fatalf("Failed to publish bugfix: %v\nOutput: %s", err, output)
	}

	// Verify success
	if !strings.Contains(output, "Successfully published 'bugfix/urgent-fix'") {
		t.Errorf("Expected success message in output, got: %s", output)
	}

	// Verify remote branch exists
	if !testutil.RemoteBranchExists(t, dir, "origin", "bugfix/urgent-fix") {
		t.Error("Expected remote branch to exist")
	}
}

// TestPublishWithoutPushOption tests publishing without push-option configured.
// Steps:
// 1. Sets up a test repository with a remote
// 2. Ensures no push-option is configured (default behavior)
// 3. Creates and publishes a branch normally
// 4. Verifies it works without push-option
func TestPublishWithoutPushOption(t *testing.T) {
	// Setup test repo with remote
	dir, remoteDir := testutil.SetupTestRepoWithRemote(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer testutil.CleanupTestRepo(t, remoteDir)

	// Create a feature branch
	output, err := testutil.RunGitFlow(t, dir, "feature", "start", "no-push-option")
	if err != nil {
		t.Fatalf("Failed to start feature: %v\nOutput: %s", err, output)
	}

	// Publish without push-option configured (should work normally)
	output, err = testutil.RunGitFlow(t, dir, "feature", "publish", "no-push-option")
	if err != nil {
		t.Fatalf("Failed to publish feature: %v\nOutput: %s", err, output)
	}

	// Verify success
	if !strings.Contains(output, "Successfully published 'feature/no-push-option'") {
		t.Errorf("Expected success message in output, got: %s", output)
	}

	// Verify remote branch exists
	if !testutil.RemoteBranchExists(t, dir, "origin", "feature/no-push-option") {
		t.Error("Expected remote branch to exist")
	}
}

// TestPublishReleaseBranchWithPushOption tests publishing a release branch with push-option.
// Steps:
// 1. Sets up a test repository with a remote
// 2. Configures push-option for release branch type
// 3. Creates and publishes a release branch
// 4. Verifies success
func TestPublishReleaseBranchWithPushOption(t *testing.T) {
	// Setup test repo with remote
	dir, remoteDir := testutil.SetupTestRepoWithRemote(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer testutil.CleanupTestRepo(t, remoteDir)

	// Configure push-option for release branch type
	_, err := testutil.RunGit(t, dir, "config", "gitflow.branch.release.pushOption", "merge_request.create")
	if err != nil {
		t.Fatalf("Failed to set push-option config: %v", err)
	}

	// Create a release branch
	output, err := testutil.RunGitFlow(t, dir, "release", "start", "2.0.0")
	if err != nil {
		t.Fatalf("Failed to start release: %v\nOutput: %s", err, output)
	}

	// Publish the release
	output, err = testutil.RunGitFlow(t, dir, "release", "publish", "2.0.0")
	if err != nil {
		t.Fatalf("Failed to publish release: %v\nOutput: %s", err, output)
	}

	// Verify success message
	if !strings.Contains(output, "Successfully published 'release/2.0.0'") {
		t.Errorf("Expected success message in output, got: %s", output)
	}

	// Verify remote branch exists
	if !testutil.RemoteBranchExists(t, dir, "origin", "release/2.0.0") {
		t.Error("Expected remote branch to exist")
	}
}

// TestPublishHotfixBranchWithPushOption tests publishing a hotfix branch with push-option.
// Steps:
// 1. Sets up a test repository with a remote
// 2. Configures push-option for hotfix branch type
// 3. Creates and publishes a hotfix branch
// 4. Verifies success
func TestPublishHotfixBranchWithPushOption(t *testing.T) {
	// Setup test repo with remote
	dir, remoteDir := testutil.SetupTestRepoWithRemote(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer testutil.CleanupTestRepo(t, remoteDir)

	// Configure push-option for hotfix branch type
	_, err := testutil.RunGit(t, dir, "config", "gitflow.branch.hotfix.pushOption", "ci.variable=\"DEPLOY=production\"")
	if err != nil {
		t.Fatalf("Failed to set push-option config: %v", err)
	}

	// Create a hotfix branch
	output, err := testutil.RunGitFlow(t, dir, "hotfix", "start", "1.0.1")
	if err != nil {
		t.Fatalf("Failed to start hotfix: %v\nOutput: %s", err, output)
	}

	// Publish the hotfix
	output, err = testutil.RunGitFlow(t, dir, "hotfix", "publish", "1.0.1")
	if err != nil {
		t.Fatalf("Failed to publish hotfix: %v\nOutput: %s", err, output)
	}

	// Verify success message
	if !strings.Contains(output, "Successfully published 'hotfix/1.0.1'") {
		t.Errorf("Expected success message in output, got: %s", output)
	}

	// Verify remote branch exists
	if !testutil.RemoteBranchExists(t, dir, "origin", "hotfix/1.0.1") {
		t.Error("Expected remote branch to exist")
	}
}
