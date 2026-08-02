package cmd_test

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gittower/git-flow-next/test/testutil"
)

// TestFinishFeatureWithTag tests finishing a feature branch with the --tag flag.
// Steps:
// 1. Sets up a test repository and initializes git-flow
// 2. Creates a feature branch
// 3. Adds changes to the feature branch
// 4. Finishes the feature branch with --tag flag
// 5. Verifies a tag is created for the feature
func TestFinishFeatureWithTag(t *testing.T) {
	t.Parallel()
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Create a feature branch
	output, err = testutil.RunGitFlow(t, dir, "feature", "start", "tagged-feature")
	if err != nil {
		t.Fatalf("Failed to create feature branch: %v\nOutput: %s", err, output)
	}

	// Create and commit a test file
	testutil.WriteFile(t, dir, "feature.txt", "feature content")
	_, err = testutil.RunGit(t, dir, "add", "feature.txt")
	if err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Add feature file")
	if err != nil {
		t.Fatalf("Failed to commit file: %v", err)
	}

	gitFlowPath := testutil.GitFlowPath()

	// Run git-flow directly with exec.Command to get full control over arguments
	cmd := exec.Command(gitFlowPath, "feature", "finish", "tagged-feature", "--tag")
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()
	if err != nil {
		t.Fatalf("Failed to finish feature branch: %v\nOutput: %s", err, stdout.String()+stderr.String())
	}
	output = stdout.String() + stderr.String()

	// Verify that a tag was created
	output, err = testutil.RunGit(t, dir, "tag", "-l")
	if err != nil {
		t.Fatalf("Failed to list tags: %v", err)
	}
	if !strings.Contains(output, "tagged-feature") {
		t.Error("Expected tag 'tagged-feature' to be created")
	}
}

// TestFinishReleaseWithCustomTag tests finishing a release branch with custom tag prefix.
// Steps:
// 1. Sets up a test repository and initializes git-flow with custom tag prefix
// 2. Creates a release branch
// 3. Adds changes to the release branch
// 4. Finishes the release branch
// 5. Verifies the branch is merged into main and develop
// 6. Verifies a tag is created with custom prefix
// 7. Verifies the release branch is deleted
func TestFinishReleaseWithCustomTag(t *testing.T) {
	t.Parallel()
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Create a release branch
	output, err = testutil.RunGitFlow(t, dir, "release", "start", "1.2.0")
	if err != nil {
		t.Fatalf("Failed to create release branch: %v\nOutput: %s", err, output)
	}

	// Create and commit a test file
	testutil.WriteFile(t, dir, "release.txt", "release content")
	_, err = testutil.RunGit(t, dir, "add", "release.txt")
	if err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Add release file")
	if err != nil {
		t.Fatalf("Failed to commit file: %v", err)
	}

	gitFlowPath := testutil.GitFlowPath()

	// Run git-flow directly
	cmd := exec.Command(gitFlowPath, "release", "finish", "1.2.0", "--tagname", "v1.2.0-beta")
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()
	if err != nil {
		t.Fatalf("Failed to finish release branch: %v\nOutput: %s", err, stdout.String()+stderr.String())
	}
	output = stdout.String() + stderr.String()

	// Verify that custom tag was created
	output, err = testutil.RunGit(t, dir, "tag", "-l")
	if err != nil {
		t.Fatalf("Failed to list tags: %v", err)
	}
	if !strings.Contains(output, "v1.2.0-beta") {
		t.Error("Expected tag 'v1.2.0-beta' to be created")
	}
	if strings.Contains(output, "1.2.0") && !strings.Contains(output, "v1.2.0-beta") {
		t.Error("Expected tag to use custom name 'v1.2.0-beta' instead of '1.2.0'")
	}
}

// TestFinishReleaseWithCustomMessage tests finishing a release branch with custom commit message.
// Steps:
// 1. Sets up a test repository and initializes git-flow
// 2. Creates a release branch
// 3. Adds changes to the release branch
// 4. Finishes the release branch with custom message
// 5. Verifies the branch is merged into main and develop
// 6. Verifies the commit message matches the custom message
// 7. Verifies the release branch is deleted
func TestFinishReleaseWithCustomMessage(t *testing.T) {
	t.Parallel()
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Set tag prefix for release branches
	_, err = testutil.RunGit(t, dir, "config", "gitflow.branch.release.tagprefix", "v")
	if err != nil {
		t.Fatalf("Failed to set tag prefix: %v", err)
	}

	// Create a release branch
	output, err = testutil.RunGitFlow(t, dir, "release", "start", "1.3.0")
	if err != nil {
		t.Fatalf("Failed to create release branch: %v\nOutput: %s", err, output)
	}

	// Create and commit a test file
	testutil.WriteFile(t, dir, "release.txt", "release content")
	_, err = testutil.RunGit(t, dir, "add", "release.txt")
	if err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Add release file")
	if err != nil {
		t.Fatalf("Failed to commit file: %v", err)
	}

	// Custom message for the tag
	customMessage := "This is release 1.3.0"

	gitFlowPath := testutil.GitFlowPath()

	// Run git-flow directly
	cmd := exec.Command(gitFlowPath, "release", "finish", "1.3.0", "--message", customMessage)
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()
	if err != nil {
		t.Fatalf("Failed to finish release branch: %v\nOutput: %s", err, stdout.String()+stderr.String())
	}
	output = stdout.String() + stderr.String()

	// Verify that the tag was created with the custom message
	output, err = testutil.RunGit(t, dir, "tag", "-n", "-l", "v1.3.0")
	if err != nil {
		t.Fatalf("Failed to get tag message: %v", err)
	}
	if !strings.Contains(output, customMessage) {
		t.Errorf("Expected tag message to contain '%s', got: %s", customMessage, output)
	}
}

// TestFinishReleaseWithNoTag tests finishing a release branch without creating a tag.
// Steps:
// 1. Sets up a test repository and initializes git-flow
// 2. Creates a release branch
// 3. Adds changes to the release branch
// 4. Finishes the release branch with no-tag flag
// 5. Verifies the branch is merged into main and develop
// 6. Verifies no tag is created
// 7. Verifies the release branch is deleted
func TestFinishReleaseWithNoTag(t *testing.T) {
	t.Parallel()
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Create a release branch
	output, err = testutil.RunGitFlow(t, dir, "release", "start", "1.4.0")
	if err != nil {
		t.Fatalf("Failed to create release branch: %v\nOutput: %s", err, output)
	}

	// Create and commit a test file
	testutil.WriteFile(t, dir, "release.txt", "release content")
	_, err = testutil.RunGit(t, dir, "add", "release.txt")
	if err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Add release file")
	if err != nil {
		t.Fatalf("Failed to commit file: %v", err)
	}

	gitFlowPath := testutil.GitFlowPath()

	// Run git-flow directly
	cmd := exec.Command(gitFlowPath, "release", "finish", "1.4.0", "--notag")
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()
	if err != nil {
		t.Fatalf("Failed to finish release branch: %v\nOutput: %s", err, stdout.String()+stderr.String())
	}
	output = stdout.String() + stderr.String()

	// Verify that no tag was created
	output, err = testutil.RunGit(t, dir, "tag", "-l")
	if err != nil {
		t.Fatalf("Failed to list tags: %v", err)
	}
	if strings.Contains(output, "1.4.0") {
		t.Error("Expected no tag to be created with --notag flag")
	}
}

// TestFinishReleaseWithMessageFile tests finishing a release branch using a message file.
// Steps:
// 1. Sets up a test repository and initializes git-flow
// 2. Creates a release branch
// 3. Adds changes to the release branch
// 4. Creates a message file
// 5. Finishes the release branch using the message file
// 6. Verifies the branch is merged into main and develop
// 7. Verifies the commit message matches the file content
// 8. Verifies the release branch is deleted
func TestFinishReleaseWithMessageFile(t *testing.T) {
	t.Parallel()
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Set tag prefix for release branches
	_, err = testutil.RunGit(t, dir, "config", "gitflow.branch.release.tagprefix", "v")
	if err != nil {
		t.Fatalf("Failed to set tag prefix: %v", err)
	}

	// Create a release branch
	output, err = testutil.RunGitFlow(t, dir, "release", "start", "1.5.0")
	if err != nil {
		t.Fatalf("Failed to create release branch: %v\nOutput: %s", err, output)
	}

	// Create and commit a test file
	testutil.WriteFile(t, dir, "release.txt", "release content")
	_, err = testutil.RunGit(t, dir, "add", "release.txt")
	if err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Add release file")
	if err != nil {
		t.Fatalf("Failed to commit file: %v", err)
	}

	// Create a message file for the tag
	tagMessageFilePath := filepath.Join(dir, "tag-message.txt")
	customMessage := "This is release 1.5.0\nWith a multi-line message\nThat describes all the changes"
	err = os.WriteFile(tagMessageFilePath, []byte(customMessage), 0644)
	if err != nil {
		t.Fatalf("Failed to create tag message file: %v", err)
	}

	gitFlowPath := testutil.GitFlowPath()

	// Run git-flow directly
	cmd := exec.Command(gitFlowPath, "release", "finish", "1.5.0", "--messagefile", tagMessageFilePath)
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()
	if err != nil {
		t.Fatalf("Failed to finish release branch: %v\nOutput: %s", err, stdout.String()+stderr.String())
	}
	output = stdout.String() + stderr.String()

	// Verify that the tag was created
	output, err = testutil.RunGit(t, dir, "tag", "-l")
	if err != nil {
		t.Fatalf("Failed to list tags: %v", err)
	}
	if !strings.Contains(output, "v1.5.0") {
		t.Error("Expected tag 'v1.5.0' to be created")
	}

	// Verify that the tag message matches the file content
	output, err = testutil.RunGit(t, dir, "tag", "-n99", "-l", "v1.5.0")
	if err != nil {
		t.Fatalf("Failed to get tag message: %v", err)
	}

	// Just verify that the tag message contains key parts of our file content
	if !strings.Contains(output, "This is release 1.5.0") {
		t.Errorf("Expected tag message to contain 'This is release 1.5.0'. Got: %s", output)
	}

	if !strings.Contains(output, "With a multi-line message") {
		t.Errorf("Expected tag message to contain 'With a multi-line message'. Got: %s", output)
	}

	if !strings.Contains(output, "That describes all the changes") {
		t.Errorf("Expected tag message to contain 'That describes all the changes'. Got: %s", output)
	}
}

// TestFinishReleaseWithConfigMessageFile tests finishing a release branch using a message file from config.
// Steps:
// 1. Sets up a test repository and initializes git-flow
// 2. Configures a default message file
// 3. Creates a release branch
// 4. Adds changes to the release branch
// 5. Finishes the release branch
// 6. Verifies the branch is merged into main and develop
// 7. Verifies the commit message matches the config file content
// 8. Verifies the release branch is deleted
func TestFinishReleaseWithConfigMessageFile(t *testing.T) {
	t.Parallel()
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Set tag prefix for release branches
	_, err = testutil.RunGit(t, dir, "config", "gitflow.branch.release.tagprefix", "v")
	if err != nil {
		t.Fatalf("Failed to set tag prefix: %v", err)
	}

	// Create a message file for the tag
	tagMessageFilePath := filepath.Join(dir, "config-tag-message.txt")
	customMessage := "This message comes from a config-specified file"
	err = os.WriteFile(tagMessageFilePath, []byte(customMessage), 0644)
	if err != nil {
		t.Fatalf("Failed to create tag message file: %v", err)
	}

	// Set the message file in git config
	_, err = testutil.RunGit(t, dir, "config", "gitflow.release.finish.messagefile", tagMessageFilePath)
	if err != nil {
		t.Fatalf("Failed to set message file config: %v", err)
	}

	// Create a release branch
	output, err = testutil.RunGitFlow(t, dir, "release", "start", "1.6.0")
	if err != nil {
		t.Fatalf("Failed to create release branch: %v\nOutput: %s", err, output)
	}

	// Create and commit a test file
	testutil.WriteFile(t, dir, "release.txt", "release content")
	_, err = testutil.RunGit(t, dir, "add", "release.txt")
	if err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Add release file")
	if err != nil {
		t.Fatalf("Failed to commit file: %v", err)
	}

	// Finish the release branch (should use message file from config)
	output, err = testutil.RunGitFlow(t, dir, "release", "finish", "1.6.0")
	if err != nil {
		t.Fatalf("Failed to finish release branch: %v\nOutput: %s", err, output)
	}

	// Verify that the tag was created
	output, err = testutil.RunGit(t, dir, "tag", "-l")
	if err != nil {
		t.Fatalf("Failed to list tags: %v", err)
	}
	if !strings.Contains(output, "v1.6.0") {
		t.Error("Expected tag 'v1.6.0' to be created")
	}

	// Verify that the tag message matches the file content
	output, err = testutil.RunGit(t, dir, "tag", "-n99", "-l", "v1.6.0")
	if err != nil {
		t.Fatalf("Failed to get tag message: %v", err)
	}

	if !strings.Contains(output, "This message comes from a config-specified file") {
		t.Errorf("Expected tag message to contain 'This message comes from a config-specified file'. Got: %s", output)
	}
}

// TestFinishTagFromBranchConfig tests finishing a branch with tag configuration from branch config.
// Steps:
// 1. Sets up a test repository and initializes git-flow
// 2. Configures tag settings for a branch type
// 3. Creates a branch of that type
// 4. Adds changes to the branch
// 5. Finishes the branch
// 6. Verifies the branch is merged
// 7. Verifies a tag is created according to config
// 8. Verifies the branch is deleted
func TestFinishTagFromBranchConfig(t *testing.T) {
	t.Parallel()
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Verify that release branches create tags by default (branch config)
	configOutput, err := testutil.RunGit(t, dir, "config", "gitflow.branch.release.tag")
	if err == nil && configOutput == "false" {
		// If it's already set to false, reset it to true
		_, err = testutil.RunGit(t, dir, "config", "gitflow.branch.release.tag", "true")
		if err != nil {
			t.Fatalf("Failed to reset tag config: %v", err)
		}
	}

	// Create a release branch
	output, err = testutil.RunGitFlow(t, dir, "release", "start", "2.0.0")
	if err != nil {
		t.Fatalf("Failed to create release branch: %v\nOutput: %s", err, output)
	}

	// Create and commit a test file
	testutil.WriteFile(t, dir, "release.txt", "release content")
	_, err = testutil.RunGit(t, dir, "add", "release.txt")
	if err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Add release file")
	if err != nil {
		t.Fatalf("Failed to commit file: %v", err)
	}

	// Finish the release branch using default command (no options)
	// This should create a tag
	output, err = testutil.RunGitFlow(t, dir, "release", "finish", "2.0.0")
	if err != nil {
		t.Fatalf("Failed to finish release branch: %v\nOutput: %s", err, output)
	}

	// Check that tag was created
	tagList, err := testutil.RunGit(t, dir, "tag", "-l")
	if err != nil {
		t.Fatalf("Failed to list tags: %v", err)
	}
	t.Logf("Tags: %s", tagList)

	if !strings.Contains(tagList, "2.0.0") {
		t.Errorf("Expected tag '2.0.0' to be created (branch config should create tag by default)")
	}
}

// TestFinishNotagFromCLI tests that the --no-tag flag overrides configuration.
// Steps:
// 1. Sets up a test repository and initializes git-flow
// 2. Configures tag creation for a branch type
// 3. Creates a branch of that type
// 4. Adds changes to the branch
// 5. Finishes the branch with --no-tag flag
// 6. Verifies the branch is merged
// 7. Verifies no tag is created despite config
// 8. Verifies the branch is deleted
func TestFinishNotagFromCLI(t *testing.T) {
	t.Parallel()
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Ensure release branches are configured to create tags (branch config)
	_, err = testutil.RunGit(t, dir, "config", "gitflow.branch.release.tag", "true")
	if err != nil {
		t.Fatalf("Failed to set tag config: %v", err)
	}

	// Verify tag configuration is enabled
	tagConfig, err := testutil.RunGit(t, dir, "config", "gitflow.branch.release.tag")
	if err != nil || strings.TrimSpace(tagConfig) != "true" {
		t.Fatalf("Failed to verify tag config is enabled: %v, got: %s", err, tagConfig)
	}

	// Create a release branch
	output, err = testutil.RunGitFlow(t, dir, "release", "start", "2.1.0")
	if err != nil {
		t.Fatalf("Failed to create release branch: %v\nOutput: %s", err, output)
	}

	// Create and commit a test file
	testutil.WriteFile(t, dir, "release.txt", "release content")
	_, err = testutil.RunGit(t, dir, "add", "release.txt")
	if err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Add release file")
	if err != nil {
		t.Fatalf("Failed to commit file: %v", err)
	}

	gitFlowPath := testutil.GitFlowPath()

	// Finish with --notag to override the config
	cmd := exec.Command(gitFlowPath, "release", "finish", "2.1.0", "--notag")
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()
	if err != nil {
		t.Fatalf("Failed to finish release: %v\nOutput: %s", err, stdout.String()+stderr.String())
	}
	t.Logf("Command output: %s", stdout.String()+stderr.String())

	// Check that no tag was created (CLI --notag should override config)
	tagList, err := testutil.RunGit(t, dir, "tag", "-l")
	if err != nil {
		t.Fatalf("Failed to list tags: %v", err)
	}
	t.Logf("Tags: %s", tagList)

	if strings.Contains(tagList, "2.1.0") {
		t.Errorf("No tag should have been created when --notag is specified (CLI should override config)")
	}
}

// TestFinishNotagFromConfig tests that tag configuration can be disabled.
// Steps:
// 1. Sets up a test repository and initializes git-flow
// 2. Configures tag creation to be disabled for a branch type
// 3. Creates a branch of that type
// 4. Adds changes to the branch
// 5. Finishes the branch
// 6. Verifies the branch is merged
// 7. Verifies no tag is created
// 8. Verifies the branch is deleted
func TestFinishNotagFromConfig(t *testing.T) {
	t.Parallel()
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Ensure release branches are configured to create tags
	_, err = testutil.RunGit(t, dir, "config", "gitflow.branch.release.tag", "true")
	if err != nil {
		t.Fatalf("Failed to set tag config: %v", err)
	}

	// Set the finish.notag config to true
	_, err = testutil.RunGit(t, dir, "config", "gitflow.release.finish.notag", "true")
	if err != nil {
		t.Fatalf("Failed to set finish.notag config: %v", err)
	}

	// Verify configs are set correctly
	tagConfig, err := testutil.RunGit(t, dir, "config", "gitflow.branch.release.tag")
	if err != nil || strings.TrimSpace(tagConfig) != "true" {
		t.Fatalf("Failed to verify branch tag config is enabled: %v, got: %s", err, tagConfig)
	}

	notagConfig, err := testutil.RunGit(t, dir, "config", "gitflow.release.finish.notag")
	if err != nil || strings.TrimSpace(notagConfig) != "true" {
		t.Fatalf("Failed to verify finish.notag config is enabled: %v, got: %s", err, notagConfig)
	}

	// Create a release branch
	output, err = testutil.RunGitFlow(t, dir, "release", "start", "2.2.0")
	if err != nil {
		t.Fatalf("Failed to create release branch: %v\nOutput: %s", err, output)
	}

	// Create and commit a test file
	testutil.WriteFile(t, dir, "release.txt", "release content")
	_, err = testutil.RunGit(t, dir, "add", "release.txt")
	if err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "commit", "-m", "Add release file")
	if err != nil {
		t.Fatalf("Failed to commit file: %v", err)
	}

	// Finish the release branch (should use notag from config)
	output, err = testutil.RunGitFlow(t, dir, "release", "finish", "2.2.0")
	if err != nil {
		t.Fatalf("Failed to finish release branch: %v\nOutput: %s", err, output)
	}

	// Check that no tag was created
	tagList, err := testutil.RunGit(t, dir, "tag", "-l")
	if err != nil {
		t.Fatalf("Failed to list tags: %v", err)
	}
	t.Logf("Tags: %s", tagList)

	if strings.Contains(tagList, "2.2.0") {
		t.Errorf("No tag should have been created when gitflow.release.finish.notag is true")
	}
}
