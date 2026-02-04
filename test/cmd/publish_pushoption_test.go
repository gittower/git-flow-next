package cmd_test

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/gittower/git-flow-next/test/testutil"
)

// installPushOptionHook installs a pre-receive hook on the bare remote that captures
// push options to a file. Returns a function to read the captured options.
func installPushOptionHook(t *testing.T, remoteDir string) func() (int, []string) {
	t.Helper()

	// Enable push options on the bare remote
	_, err := testutil.RunGit(t, remoteDir, "config", "receive.advertisePushOptions", "true")
	if err != nil {
		t.Fatalf("Failed to enable push options on remote: %v", err)
	}

	// Create hooks directory if it doesn't exist
	hooksDir := filepath.Join(remoteDir, "hooks")
	if err := os.MkdirAll(hooksDir, 0755); err != nil {
		t.Fatalf("Failed to create hooks directory: %v", err)
	}

	// Install pre-receive hook that captures push options
	hookScript := `#!/bin/sh
echo "$GIT_PUSH_OPTION_COUNT" > "$GIT_DIR/push-options-received"
i=0
while [ "$i" -lt "${GIT_PUSH_OPTION_COUNT:-0}" ]; do
    eval "echo \$GIT_PUSH_OPTION_$i" >> "$GIT_DIR/push-options-received"
    i=$((i + 1))
done
cat  # drain stdin so push doesn't fail
`
	hookPath := filepath.Join(hooksDir, "pre-receive")
	if err := os.WriteFile(hookPath, []byte(hookScript), 0755); err != nil {
		t.Fatalf("Failed to write pre-receive hook: %v", err)
	}

	// Return function to read captured options
	return func() (int, []string) {
		optionsFile := filepath.Join(remoteDir, "push-options-received")
		data, err := os.ReadFile(optionsFile)
		if err != nil {
			// File doesn't exist means no push options were sent
			return 0, nil
		}
		lines := strings.Split(strings.TrimSpace(string(data)), "\n")
		if len(lines) == 0 {
			return 0, nil
		}
		count, err := strconv.Atoi(lines[0])
		if err != nil {
			return 0, nil
		}
		var options []string
		if len(lines) > 1 {
			options = lines[1:]
		}
		return count, options
	}
}

// TestPublishWithPushOption tests that a single --push-option flag is transmitted to the remote.
// Steps:
// 1. Sets up a test repository with a remote and installs a pre-receive hook to capture push options
// 2. Initializes git-flow and creates a feature branch
// 3. Runs 'git flow feature publish my-feature --push-option ci.skip'
// 4. Reads captured push options from the remote hook output file
// 5. Verifies exactly one push option "ci.skip" was received
func TestPublishWithPushOption(t *testing.T) {
	dir, remoteDir := testutil.SetupTestRepoWithRemote(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer testutil.CleanupTestRepo(t, remoteDir)

	readOptions := installPushOptionHook(t, remoteDir)

	_, err := testutil.RunGitFlow(t, dir, "feature", "start", "my-feature")
	if err != nil {
		t.Fatalf("Failed to start feature: %v", err)
	}

	output, err := testutil.RunGitFlow(t, dir, "feature", "publish", "my-feature", "--push-option", "ci.skip")
	if err != nil {
		t.Fatalf("Failed to publish feature: %v\nOutput: %s", err, output)
	}

	count, options := readOptions()
	if count != 1 {
		t.Errorf("Expected 1 push option, got %d", count)
	}
	if len(options) != 1 || options[0] != "ci.skip" {
		t.Errorf("Expected push option 'ci.skip', got %v", options)
	}
}

// TestPublishWithMultiplePushOptions tests that multiple -o flags are all transmitted to the remote.
// Steps:
// 1. Sets up a test repository with a remote and installs a pre-receive hook
// 2. Initializes git-flow and creates a feature branch
// 3. Runs 'git flow feature publish my-feature -o ci.skip -o merge_request.create'
// 4. Reads captured push options from the remote hook output file
// 5. Verifies both "ci.skip" and "merge_request.create" were received in order
func TestPublishWithMultiplePushOptions(t *testing.T) {
	dir, remoteDir := testutil.SetupTestRepoWithRemote(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer testutil.CleanupTestRepo(t, remoteDir)

	readOptions := installPushOptionHook(t, remoteDir)

	_, err := testutil.RunGitFlow(t, dir, "feature", "start", "my-feature")
	if err != nil {
		t.Fatalf("Failed to start feature: %v", err)
	}

	output, err := testutil.RunGitFlow(t, dir, "feature", "publish", "my-feature", "-o", "ci.skip", "-o", "merge_request.create")
	if err != nil {
		t.Fatalf("Failed to publish feature: %v\nOutput: %s", err, output)
	}

	count, options := readOptions()
	if count != 2 {
		t.Errorf("Expected 2 push options, got %d", count)
	}
	if len(options) != 2 || options[0] != "ci.skip" || options[1] != "merge_request.create" {
		t.Errorf("Expected push options ['ci.skip', 'merge_request.create'], got %v", options)
	}
}

// TestPublishWithConfigPushOptions tests that push options from git config are transmitted to the remote.
// Steps:
// 1. Sets up a test repository with a remote and installs a pre-receive hook
// 2. Initializes git-flow and sets gitflow.feature.publish.push-option config values
// 3. Creates a feature branch
// 4. Runs 'git flow feature publish my-feature' (no CLI push options)
// 5. Reads captured push options and verifies config defaults were sent
func TestPublishWithConfigPushOptions(t *testing.T) {
	dir, remoteDir := testutil.SetupTestRepoWithRemote(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer testutil.CleanupTestRepo(t, remoteDir)

	readOptions := installPushOptionHook(t, remoteDir)

	// Configure push options for feature branches
	_, err := testutil.RunGit(t, dir, "config", "gitflow.feature.publish.push-option", "ci.skip")
	if err != nil {
		t.Fatalf("Failed to set push-option config: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "config", "--add", "gitflow.feature.publish.push-option", "merge_request.create")
	if err != nil {
		t.Fatalf("Failed to add push-option config: %v", err)
	}

	_, err = testutil.RunGitFlow(t, dir, "feature", "start", "my-feature")
	if err != nil {
		t.Fatalf("Failed to start feature: %v", err)
	}

	output, err := testutil.RunGitFlow(t, dir, "feature", "publish", "my-feature")
	if err != nil {
		t.Fatalf("Failed to publish feature: %v\nOutput: %s", err, output)
	}

	count, options := readOptions()
	if count != 2 {
		t.Errorf("Expected 2 push options from config, got %d", count)
	}
	if len(options) != 2 || options[0] != "ci.skip" || options[1] != "merge_request.create" {
		t.Errorf("Expected push options ['ci.skip', 'merge_request.create'], got %v", options)
	}
}

// TestPublishWithNoPushOption tests that --no-push-option suppresses config defaults.
// Steps:
// 1. Sets up a test repository with a remote and installs a pre-receive hook
// 2. Initializes git-flow and sets gitflow.feature.publish.push-option config values
// 3. Creates a feature branch
// 4. Runs 'git flow feature publish my-feature --no-push-option'
// 5. Reads captured push options and verifies none were sent (count is 0)
func TestPublishWithNoPushOption(t *testing.T) {
	dir, remoteDir := testutil.SetupTestRepoWithRemote(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer testutil.CleanupTestRepo(t, remoteDir)

	readOptions := installPushOptionHook(t, remoteDir)

	// Configure push options for feature branches
	_, err := testutil.RunGit(t, dir, "config", "gitflow.feature.publish.push-option", "ci.skip")
	if err != nil {
		t.Fatalf("Failed to set push-option config: %v", err)
	}

	_, err = testutil.RunGitFlow(t, dir, "feature", "start", "my-feature")
	if err != nil {
		t.Fatalf("Failed to start feature: %v", err)
	}

	output, err := testutil.RunGitFlow(t, dir, "feature", "publish", "my-feature", "--no-push-option")
	if err != nil {
		t.Fatalf("Failed to publish feature: %v\nOutput: %s", err, output)
	}

	count, options := readOptions()
	if count != 0 {
		t.Errorf("Expected 0 push options with --no-push-option, got %d: %v", count, options)
	}
}

// TestPublishWithConfigAndCliPushOptions tests that CLI options combine with config defaults.
// Steps:
// 1. Sets up a test repository with a remote and installs a pre-receive hook
// 2. Initializes git-flow and sets gitflow.feature.publish.push-option to "ci.skip"
// 3. Creates a feature branch
// 4. Runs 'git flow feature publish my-feature --push-option merge_request.create'
// 5. Reads captured push options and verifies both "ci.skip" and "merge_request.create" received
func TestPublishWithConfigAndCliPushOptions(t *testing.T) {
	dir, remoteDir := testutil.SetupTestRepoWithRemote(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer testutil.CleanupTestRepo(t, remoteDir)

	readOptions := installPushOptionHook(t, remoteDir)

	// Configure push option for feature branches
	_, err := testutil.RunGit(t, dir, "config", "gitflow.feature.publish.push-option", "ci.skip")
	if err != nil {
		t.Fatalf("Failed to set push-option config: %v", err)
	}

	_, err = testutil.RunGitFlow(t, dir, "feature", "start", "my-feature")
	if err != nil {
		t.Fatalf("Failed to start feature: %v", err)
	}

	output, err := testutil.RunGitFlow(t, dir, "feature", "publish", "my-feature", "-o", "merge_request.create")
	if err != nil {
		t.Fatalf("Failed to publish feature: %v\nOutput: %s", err, output)
	}

	count, options := readOptions()
	if count != 2 {
		t.Errorf("Expected 2 push options (config + CLI), got %d", count)
	}
	// Config options come first, then CLI options
	if len(options) != 2 || options[0] != "ci.skip" || options[1] != "merge_request.create" {
		t.Errorf("Expected push options ['ci.skip', 'merge_request.create'], got %v", options)
	}
}

// TestPublishWithoutPushOptions tests that no push options are sent when none are configured or provided.
// Steps:
// 1. Sets up a test repository with a remote and installs a pre-receive hook
// 2. Initializes git-flow and creates a feature branch (no push option config)
// 3. Runs 'git flow feature publish my-feature'
// 4. Reads captured push options and verifies count is 0 or hook file shows no options
func TestPublishWithoutPushOptions(t *testing.T) {
	dir, remoteDir := testutil.SetupTestRepoWithRemote(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer testutil.CleanupTestRepo(t, remoteDir)

	readOptions := installPushOptionHook(t, remoteDir)

	_, err := testutil.RunGitFlow(t, dir, "feature", "start", "my-feature")
	if err != nil {
		t.Fatalf("Failed to start feature: %v", err)
	}

	output, err := testutil.RunGitFlow(t, dir, "feature", "publish", "my-feature")
	if err != nil {
		t.Fatalf("Failed to publish feature: %v\nOutput: %s", err, output)
	}

	count, options := readOptions()
	if count != 0 {
		t.Errorf("Expected 0 push options when none configured, got %d: %v", count, options)
	}
}
