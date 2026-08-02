package cmd_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gittower/git-flow-next/internal/errors"
	"github.com/gittower/git-flow-next/test/testutil"
)

// createHookScript creates an executable hook script in the repository's hooks directory.
func createHookScript(t *testing.T, dir, name, content string) {
	t.Helper()
	hooksDir := filepath.Join(dir, ".git", "hooks")
	if err := os.MkdirAll(hooksDir, 0755); err != nil {
		t.Fatalf("Failed to create hooks directory: %v", err)
	}

	scriptPath := filepath.Join(hooksDir, name)
	if err := os.WriteFile(scriptPath, []byte(content), 0755); err != nil {
		t.Fatalf("Failed to create script %s: %v", name, err)
	}
}

// =============================================================================
// Pre-Hook Tests - Verify hooks can block operations
// =============================================================================

// TestStartPreHookBlocks tests that a failing pre-hook prevents branch creation.
func TestStartPreHookBlocks(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow
	_, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v", err)
	}

	// Create a pre-hook that fails
	script := `#!/bin/sh
echo "Pre-hook blocking operation" >&2
exit 1
`
	createHookScript(t, dir, "pre-flow-feature-start", script)

	// Try to start a feature - should fail
	output, err := testutil.RunGitFlow(t, dir, "feature", "start", "blocked-feature")
	if err == nil {
		t.Fatal("Expected feature start to fail due to pre-hook, but it succeeded")
	}

	// Verify the error mentions the hook
	if !strings.Contains(output, "pre-hook") {
		t.Errorf("Expected error to mention pre-hook, got: %s", output)
	}

	// Verify branch was NOT created
	if testutil.BranchExists(t, dir, "feature/blocked-feature") {
		t.Error("Branch should not have been created when pre-hook failed")
	}
}

// TestFinishPreHookBlocks tests that a failing pre-hook prevents finish operation.
func TestFinishPreHookBlocks(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow and create a feature branch
	_, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v", err)
	}

	_, err = testutil.RunGitFlow(t, dir, "feature", "start", "my-feature")
	if err != nil {
		t.Fatalf("Failed to start feature: %v", err)
	}

	// Add a commit to the feature branch
	testutil.WriteFile(t, dir, "feature.txt", "feature content")
	_, _ = testutil.RunGit(t, dir, "add", "feature.txt")
	_, _ = testutil.RunGit(t, dir, "commit", "-m", "Add feature file")

	// Create a pre-hook that fails
	script := `#!/bin/sh
echo "Pre-hook blocking finish" >&2
exit 1
`
	createHookScript(t, dir, "pre-flow-feature-finish", script)

	// Try to finish - should fail
	output, err := testutil.RunGitFlow(t, dir, "feature", "finish", "my-feature")
	if err == nil {
		t.Fatal("Expected feature finish to fail due to pre-hook, but it succeeded")
	}

	// Verify the error mentions the hook
	if !strings.Contains(output, "pre-hook") {
		t.Errorf("Expected error to mention pre-hook, got: %s", output)
	}

	// Verify branch still exists (finish was blocked)
	if !testutil.BranchExists(t, dir, "feature/my-feature") {
		t.Error("Feature branch should still exist when pre-hook blocked finish")
	}
}

// TestDeletePreHookBlocks tests that a failing pre-hook prevents delete operation.
func TestDeletePreHookBlocks(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow and create a feature branch
	_, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v", err)
	}

	_, err = testutil.RunGitFlow(t, dir, "feature", "start", "to-delete")
	if err != nil {
		t.Fatalf("Failed to start feature: %v", err)
	}

	// Switch back to develop so we can delete the feature branch
	_, _ = testutil.RunGit(t, dir, "checkout", "develop")

	// Create a pre-hook that fails
	script := `#!/bin/sh
echo "Pre-hook blocking delete" >&2
exit 1
`
	createHookScript(t, dir, "pre-flow-feature-delete", script)

	// Try to delete - should fail
	output, err := testutil.RunGitFlow(t, dir, "feature", "delete", "to-delete")
	if err == nil {
		t.Fatal("Expected feature delete to fail due to pre-hook, but it succeeded")
	}

	// Verify the error mentions the hook
	if !strings.Contains(output, "pre-hook") {
		t.Errorf("Expected error to mention pre-hook, got: %s", output)
	}

	// Verify branch still exists
	if !testutil.BranchExists(t, dir, "feature/to-delete") {
		t.Error("Feature branch should still exist when pre-hook blocked delete")
	}
}

// =============================================================================
// Post-Hook Tests - Verify hooks run after operations
// =============================================================================

// TestStartPostHookRuns tests that post-hook executes after successful branch creation.
func TestStartPostHookRuns(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow
	_, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v", err)
	}

	// Create a marker file path
	markerFile := filepath.Join(dir, "post-hook-executed.txt")

	// Create a post-hook that creates a marker file
	script := `#!/bin/sh
echo "BRANCH=$BRANCH" > "` + markerFile + `"
echo "BRANCH_NAME=$BRANCH_NAME" >> "` + markerFile + `"
echo "BRANCH_TYPE=$BRANCH_TYPE" >> "` + markerFile + `"
`
	createHookScript(t, dir, "post-flow-feature-start", script)

	// Start a feature
	_, err = testutil.RunGitFlow(t, dir, "feature", "start", "post-hook-test")
	if err != nil {
		t.Fatalf("Failed to start feature: %v", err)
	}

	// Verify post-hook ran by checking marker file
	content, err := os.ReadFile(markerFile)
	if err != nil {
		t.Fatalf("Post-hook did not run - marker file not found: %v", err)
	}

	// Verify environment variables were passed correctly
	contentStr := string(content)
	if !strings.Contains(contentStr, "BRANCH=feature/post-hook-test") {
		t.Errorf("Expected BRANCH=feature/post-hook-test in hook output, got: %s", contentStr)
	}
	if !strings.Contains(contentStr, "BRANCH_NAME=post-hook-test") {
		t.Errorf("Expected BRANCH_NAME=post-hook-test in hook output, got: %s", contentStr)
	}
	if !strings.Contains(contentStr, "BRANCH_TYPE=feature") {
		t.Errorf("Expected BRANCH_TYPE=feature in hook output, got: %s", contentStr)
	}
}

// TestFinishPostHookReceivesExitCode tests that post-hook receives correct exit code.
func TestFinishPostHookReceivesExitCode(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow and create a feature branch
	_, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v", err)
	}

	_, err = testutil.RunGitFlow(t, dir, "feature", "start", "exit-code-test")
	if err != nil {
		t.Fatalf("Failed to start feature: %v", err)
	}

	// Add a commit
	testutil.WriteFile(t, dir, "test.txt", "test content")
	_, _ = testutil.RunGit(t, dir, "add", "test.txt")
	_, _ = testutil.RunGit(t, dir, "commit", "-m", "Add test file")

	// Create a marker file path
	markerFile := filepath.Join(dir, "post-finish-executed.txt")

	// Create a post-hook that records the exit code
	script := `#!/bin/sh
echo "EXIT_CODE=$EXIT_CODE" > "` + markerFile + `"
`
	createHookScript(t, dir, "post-flow-feature-finish", script)

	// Finish the feature
	_, err = testutil.RunGitFlow(t, dir, "feature", "finish", "exit-code-test")
	if err != nil {
		t.Fatalf("Failed to finish feature: %v", err)
	}

	// Verify post-hook ran and received exit code 0
	content, err := os.ReadFile(markerFile)
	if err != nil {
		t.Fatalf("Post-hook did not run - marker file not found: %v", err)
	}

	if !strings.Contains(string(content), "EXIT_CODE=0") {
		t.Errorf("Expected EXIT_CODE=0 in hook output, got: %s", string(content))
	}
}

// =============================================================================
// Version Filter Tests - Verify filters modify branch names
// =============================================================================

// TestStartVersionFilterModifiesBranchName tests that version filter modifies the branch name.
func TestStartVersionFilterModifiesBranchName(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow
	_, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v", err)
	}

	// Create a version filter that adds 'v' prefix
	script := `#!/bin/sh
VERSION="$1"
if [ "${VERSION#v}" = "$VERSION" ]; then
    echo "v$VERSION"
else
    echo "$VERSION"
fi
`
	createHookScript(t, dir, "filter-flow-release-start-version", script)

	// Start a release with version "1.0.0" - filter should change to "v1.0.0"
	output, err := testutil.RunGitFlow(t, dir, "release", "start", "1.0.0")
	if err != nil {
		t.Fatalf("Failed to start release: %v\nOutput: %s", err, output)
	}

	// Verify the filter message is shown
	if !strings.Contains(output, "Version filter changed") {
		t.Errorf("Expected output to mention version filter, got: %s", output)
	}

	// Verify the branch was created with the filtered name
	if !testutil.BranchExists(t, dir, "release/v1.0.0") {
		t.Error("Expected release/v1.0.0 branch to exist (filtered from 1.0.0)")
	}

	// Verify original name branch was NOT created
	if testutil.BranchExists(t, dir, "release/1.0.0") {
		t.Error("release/1.0.0 should not exist - filter should have changed it to v1.0.0")
	}
}

// TestStartDerivesVersionFromFilterWhenNameOmitted tests that a no-argument
// start runs the version filter (with an empty version argument) and uses its
// trimmed output as the branch name for a release branch.
// Steps:
// 1. Sets up a test repository and initializes git-flow with defaults
// 2. Installs filter-flow-release-start-version that echoes "1.4.0" only when its $1 is empty
// 3. Runs 'git flow release start' with no name and no base
// 4. Verifies exit 0 and output contains "Created branch 'release/1.4.0' from 'develop'"
// 5. Verifies branch release/1.4.0 exists (filter-derived name from empty $1)
func TestStartDerivesVersionFromFilterWhenNameOmitted(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	_, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v", err)
	}

	// Filter echoes 1.4.0 only when the version argument ($1) is empty,
	// proving it was invoked with an empty version by the no-arg start.
	script := `#!/bin/sh
VERSION="$1"
if [ -z "$VERSION" ]; then
    echo "1.4.0"
fi
`
	createHookScript(t, dir, "filter-flow-release-start-version", script)

	output, err := testutil.RunGitFlow(t, dir, "release", "start")
	if err != nil {
		t.Fatalf("Expected release start to succeed, got error: %v\nOutput: %s", err, output)
	}

	if !strings.Contains(output, "Created branch 'release/1.4.0' from 'develop'") {
		t.Errorf("Expected output to contain \"Created branch 'release/1.4.0' from 'develop'\", got: %s", output)
	}

	if !testutil.BranchExists(t, dir, "release/1.4.0") {
		t.Error("Expected release/1.4.0 branch to exist (derived from version filter)")
	}
}

// TestStartNoFilterNoNameReturnsEmptyNameError tests that a no-argument start
// with no version filter falls back to the business-layer empty-name error
// rather than the Cobra arg-count error.
// Steps:
// 1. Sets up a test repository and initializes git-flow with defaults (no filter installed)
// 2. Runs 'git flow release start' with no args
// 3. Verifies non-zero exit with code ExitCodeInvalidInput (2)
// 4. Verifies output contains "branch name cannot be empty" and NOT "accepts between"
// 5. Verifies no release branch was created (refs/heads/release/ is empty)
func TestStartNoFilterNoNameReturnsEmptyNameError(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	_, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v", err)
	}

	output, err := testutil.RunGitFlow(t, dir, "release", "start")
	if err == nil {
		t.Fatalf("Expected release start with no name to fail, but it succeeded\nOutput: %s", output)
	}

	if exitErr, ok := err.(*testutil.ExitError); ok {
		if exitErr.ExitCode != int(errors.ExitCodeInvalidInput) {
			t.Errorf("Expected exit code %d, got %d", errors.ExitCodeInvalidInput, exitErr.ExitCode)
		}
	} else {
		t.Errorf("Expected *testutil.ExitError, got %T", err)
	}

	if !strings.Contains(output, "branch name cannot be empty") {
		t.Errorf("Expected output to contain 'branch name cannot be empty', got: %s", output)
	}
	if strings.Contains(output, "accepts between") {
		t.Errorf("Expected business-layer error, not Cobra arg-count error, got: %s", output)
	}

	refs, err := testutil.RunGit(t, dir, "for-each-ref", "--format=%(refname:short)", "refs/heads/release/")
	if err != nil {
		t.Fatalf("for-each-ref failed: %v", err)
	}
	if strings.TrimSpace(refs) != "" {
		t.Errorf("Expected no release branch to be created, got refs: %s", refs)
	}
}

// TestStartFilterReturnsEmptyReturnsEmptyNameError tests that a no-argument
// start with a filter that produces no output falls back to the empty-name error.
// Steps:
// 1. Sets up a test repository and initializes git-flow with defaults
// 2. Installs filter-flow-release-start-version that prints nothing and exits 0
// 3. Runs 'git flow release start' with no args
// 4. Verifies non-zero exit with code ExitCodeInvalidInput (2)
// 5. Verifies output contains "branch name cannot be empty"
// 6. Verifies no release branch was created (refs/heads/release/ is empty)
func TestStartFilterReturnsEmptyReturnsEmptyNameError(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	_, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v", err)
	}

	script := `#!/bin/sh
exit 0
`
	createHookScript(t, dir, "filter-flow-release-start-version", script)

	output, err := testutil.RunGitFlow(t, dir, "release", "start")
	if err == nil {
		t.Fatalf("Expected release start with empty filter output to fail, but it succeeded\nOutput: %s", output)
	}

	if exitErr, ok := err.(*testutil.ExitError); ok {
		if exitErr.ExitCode != int(errors.ExitCodeInvalidInput) {
			t.Errorf("Expected exit code %d, got %d", errors.ExitCodeInvalidInput, exitErr.ExitCode)
		}
	} else {
		t.Errorf("Expected *testutil.ExitError, got %T", err)
	}

	if !strings.Contains(output, "branch name cannot be empty") {
		t.Errorf("Expected output to contain 'branch name cannot be empty', got: %s", output)
	}

	refs, err := testutil.RunGit(t, dir, "for-each-ref", "--format=%(refname:short)", "refs/heads/release/")
	if err != nil {
		t.Fatalf("for-each-ref failed: %v", err)
	}
	if strings.TrimSpace(refs) != "" {
		t.Errorf("Expected no release branch to be created, got refs: %s", refs)
	}
}

// TestStartExplicitNameNoFilter tests that an explicit name with no filter
// installed creates the branch unchanged and prints no filter-change message.
// Steps:
// 1. Sets up a test repository and initializes git-flow with defaults (no filter installed)
// 2. Runs 'git flow release start 1.0.0'
// 3. Verifies exit 0 and output contains "Created branch 'release/1.0.0' from 'develop'"
// 4. Verifies branch release/1.0.0 exists
// 5. Verifies output does NOT contain "Version filter changed"
func TestStartExplicitNameNoFilter(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	_, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v", err)
	}

	output, err := testutil.RunGitFlow(t, dir, "release", "start", "1.0.0")
	if err != nil {
		t.Fatalf("Expected release start to succeed, got error: %v\nOutput: %s", err, output)
	}

	if !strings.Contains(output, "Created branch 'release/1.0.0' from 'develop'") {
		t.Errorf("Expected output to contain \"Created branch 'release/1.0.0' from 'develop'\", got: %s", output)
	}
	if !testutil.BranchExists(t, dir, "release/1.0.0") {
		t.Error("Expected release/1.0.0 branch to exist")
	}
	if strings.Contains(output, "Version filter changed") {
		t.Errorf("Expected no version filter message, got: %s", output)
	}
}

// TestStartDerivesVersionFromFilterHotfix tests that no-argument start version
// derivation is generic across branch types, using hotfix (start point main).
// Steps:
// 1. Sets up a test repository and initializes git-flow with defaults
// 2. Installs filter-flow-hotfix-start-version that echoes "2.0.1"
// 3. Runs 'git flow hotfix start' with no args
// 4. Verifies exit 0 and output contains "Created branch 'hotfix/2.0.1' from 'main'"
// 5. Verifies branch hotfix/2.0.1 exists (derived name, hotfix start point main)
func TestStartDerivesVersionFromFilterHotfix(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	_, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v", err)
	}

	script := `#!/bin/sh
echo "2.0.1"
`
	createHookScript(t, dir, "filter-flow-hotfix-start-version", script)

	output, err := testutil.RunGitFlow(t, dir, "hotfix", "start")
	if err != nil {
		t.Fatalf("Expected hotfix start to succeed, got error: %v\nOutput: %s", err, output)
	}

	if !strings.Contains(output, "Created branch 'hotfix/2.0.1' from 'main'") {
		t.Errorf("Expected output to contain \"Created branch 'hotfix/2.0.1' from 'main'\", got: %s", output)
	}
	if !testutil.BranchExists(t, dir, "hotfix/2.0.1") {
		t.Error("Expected hotfix/2.0.1 branch to exist (derived from version filter)")
	}
}

// TestStartFeatureNoFilterNoNameReturnsEmptyNameError tests that a feature type
// with no version filter and no argument still yields the empty-name error.
// Steps:
// 1. Sets up a test repository and initializes git-flow with defaults (no feature filter)
// 2. Runs 'git flow feature start' with no args
// 3. Verifies non-zero exit with code ExitCodeInvalidInput (2)
// 4. Verifies output contains "branch name cannot be empty"
// 5. Verifies no feature branch was created (refs/heads/feature/ is empty)
func TestStartFeatureNoFilterNoNameReturnsEmptyNameError(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	_, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v", err)
	}

	output, err := testutil.RunGitFlow(t, dir, "feature", "start")
	if err == nil {
		t.Fatalf("Expected feature start with no name to fail, but it succeeded\nOutput: %s", output)
	}

	if exitErr, ok := err.(*testutil.ExitError); ok {
		if exitErr.ExitCode != int(errors.ExitCodeInvalidInput) {
			t.Errorf("Expected exit code %d, got %d", errors.ExitCodeInvalidInput, exitErr.ExitCode)
		}
	} else {
		t.Errorf("Expected *testutil.ExitError, got %T", err)
	}

	if !strings.Contains(output, "branch name cannot be empty") {
		t.Errorf("Expected output to contain 'branch name cannot be empty', got: %s", output)
	}

	refs, err := testutil.RunGit(t, dir, "for-each-ref", "--format=%(refname:short)", "refs/heads/feature/")
	if err != nil {
		t.Fatalf("for-each-ref failed: %v", err)
	}
	if strings.TrimSpace(refs) != "" {
		t.Errorf("Expected no feature branch to be created, got refs: %s", refs)
	}
}

// TestStartTooManyArgsRejected tests that relaxing the lower arg bound to zero
// still caps the upper bound at two, rejecting three positional arguments.
// Steps:
// 1. Sets up a test repository and initializes git-flow with defaults
// 2. Runs 'git flow release start 1.0.0 develop extra' (three positional args)
// 3. Verifies non-zero exit and output contains "accepts between" (Cobra arg-count error)
// 4. Verifies no release branch was created (refs/heads/release/ is empty)
func TestStartTooManyArgsRejected(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	_, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v", err)
	}

	output, err := testutil.RunGitFlow(t, dir, "release", "start", "1.0.0", "develop", "extra")
	if err == nil {
		t.Fatalf("Expected release start with three args to fail, but it succeeded\nOutput: %s", output)
	}

	if !strings.Contains(output, "accepts between") {
		t.Errorf("Expected Cobra arg-count error containing 'accepts between', got: %s", output)
	}

	refs, err := testutil.RunGit(t, dir, "for-each-ref", "--format=%(refname:short)", "refs/heads/release/")
	if err != nil {
		t.Fatalf("for-each-ref failed: %v", err)
	}
	if strings.TrimSpace(refs) != "" {
		t.Errorf("Expected no release branch to be created, got refs: %s", refs)
	}
}

// TestStartFilterNonZeroExitReturnsGitError tests that a version filter which
// fails (non-zero exit) surfaces a Git error, not the empty-name fallback,
// proving the filter runs before the moved empty-name guard.
// Steps:
// 1. Sets up a test repository and initializes git-flow with defaults
// 2. Installs filter-flow-release-start-version that writes to stderr and exits 1
// 3. Runs 'git flow release start' with no args
// 4. Verifies non-zero exit with code ExitCodeGitError (3), not ExitCodeInvalidInput (2)
// 5. Verifies output contains "version filter" and NOT "branch name cannot be empty"
// 6. Verifies no release branch was created (refs/heads/release/ is empty)
func TestStartFilterNonZeroExitReturnsGitError(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	_, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v", err)
	}

	script := `#!/bin/sh
echo "filter boom" >&2
exit 1
`
	createHookScript(t, dir, "filter-flow-release-start-version", script)

	output, err := testutil.RunGitFlow(t, dir, "release", "start")
	if err == nil {
		t.Fatalf("Expected release start with failing filter to fail, but it succeeded\nOutput: %s", output)
	}

	if exitErr, ok := err.(*testutil.ExitError); ok {
		if exitErr.ExitCode != int(errors.ExitCodeGitError) {
			t.Errorf("Expected exit code %d, got %d", errors.ExitCodeGitError, exitErr.ExitCode)
		}
	} else {
		t.Errorf("Expected *testutil.ExitError, got %T", err)
	}

	if !strings.Contains(output, "version filter") {
		t.Errorf("Expected output to contain 'version filter', got: %s", output)
	}
	if strings.Contains(output, "branch name cannot be empty") {
		t.Errorf("Expected Git error, not empty-name fallback, got: %s", output)
	}

	refs, err := testutil.RunGit(t, dir, "for-each-ref", "--format=%(refname:short)", "refs/heads/release/")
	if err != nil {
		t.Fatalf("for-each-ref failed: %v", err)
	}
	if strings.TrimSpace(refs) != "" {
		t.Errorf("Expected no release branch to be created, got refs: %s", refs)
	}
}

// TestVersionFilterPassedToHooks tests that filtered version is passed to hooks.
func TestVersionFilterPassedToHooks(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow
	_, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v", err)
	}

	markerFile := filepath.Join(dir, "hook-received-version.txt")

	// Create a version filter that adds 'v' prefix
	filterScript := `#!/bin/sh
VERSION="$1"
echo "v$VERSION"
`
	createHookScript(t, dir, "filter-flow-release-start-version", filterScript)

	// Create a post-hook that records the version it received
	hookScript := `#!/bin/sh
echo "VERSION=$VERSION" > "` + markerFile + `"
echo "BRANCH_NAME=$BRANCH_NAME" >> "` + markerFile + `"
`
	createHookScript(t, dir, "post-flow-release-start", hookScript)

	// Start a release
	_, err = testutil.RunGitFlow(t, dir, "release", "start", "2.0.0")
	if err != nil {
		t.Fatalf("Failed to start release: %v", err)
	}

	// Verify hook received filtered version
	content, err := os.ReadFile(markerFile)
	if err != nil {
		t.Fatalf("Post-hook did not run: %v", err)
	}

	contentStr := string(content)
	// The hook should receive the filtered version
	if !strings.Contains(contentStr, "VERSION=v2.0.0") {
		t.Errorf("Expected VERSION=v2.0.0, got: %s", contentStr)
	}
	if !strings.Contains(contentStr, "BRANCH_NAME=v2.0.0") {
		t.Errorf("Expected BRANCH_NAME=v2.0.0, got: %s", contentStr)
	}
}

// =============================================================================
// Tag Message Filter Tests - Verify filters modify tag messages
// =============================================================================

// TestFinishTagMessageFilterModifiesTag tests that tag message filter modifies the tag message.
func TestFinishTagMessageFilterModifiesTag(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow
	_, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v", err)
	}

	// Start a release
	_, err = testutil.RunGitFlow(t, dir, "release", "start", "3.0.0")
	if err != nil {
		t.Fatalf("Failed to start release: %v", err)
	}

	// Add a commit to the release branch
	testutil.WriteFile(t, dir, "release.txt", "release content")
	_, _ = testutil.RunGit(t, dir, "add", "release.txt")
	_, _ = testutil.RunGit(t, dir, "commit", "-m", "Add release file")

	// Create a tag message filter that adds custom prefix
	script := `#!/bin/sh
VERSION="$1"
echo "Custom Release: $VERSION - Modified by filter"
`
	createHookScript(t, dir, "filter-flow-release-finish-tag-message", script)

	// Finish the release
	_, err = testutil.RunGitFlow(t, dir, "release", "finish", "3.0.0")
	if err != nil {
		t.Fatalf("Failed to finish release: %v", err)
	}

	// Verify tag was created
	output, err := testutil.RunGit(t, dir, "tag", "-l")
	if err != nil {
		t.Fatalf("Failed to list tags: %v", err)
	}
	if !strings.Contains(output, "3.0.0") {
		t.Errorf("Expected tag 3.0.0 to exist, got: %s", output)
	}

	// Verify tag message was modified by filter
	output, err = testutil.RunGit(t, dir, "tag", "-l", "-n1", "3.0.0")
	if err != nil {
		t.Fatalf("Failed to get tag message: %v", err)
	}
	if !strings.Contains(output, "Custom Release") || !strings.Contains(output, "Modified by filter") {
		t.Errorf("Expected tag message to be modified by filter, got: %s", output)
	}
}

// =============================================================================
// Worktree Integration Test - Verify hooks work from worktree
// =============================================================================

// TestStartWithHooksInWorktree tests that hooks work when running commands from a worktree.
func TestStartWithHooksInWorktree(t *testing.T) {
	t.Parallel()
	// Setup main repository
	mainRepo := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, mainRepo)

	// Initialize git-flow in main repo
	_, err := testutil.RunGitFlow(t, mainRepo, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v", err)
	}

	// Create a worktree
	worktreePath, err := os.MkdirTemp("", "git-flow-cmd-worktree-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory for worktree: %v", err)
	}
	defer os.RemoveAll(worktreePath)
	os.RemoveAll(worktreePath)

	_, err = testutil.RunGit(t, mainRepo, "worktree", "add", worktreePath, "-b", "worktree-branch")
	if err != nil {
		t.Fatalf("Failed to create worktree: %v", err)
	}

	// Create marker file in worktree directory
	markerFile := filepath.Join(worktreePath, "worktree-hook-executed.txt")

	// Create hooks in main repo's .git/hooks (shared location)
	preScript := `#!/bin/sh
echo "pre-hook-ran" > "` + markerFile + `"
`
	createHookScript(t, mainRepo, "pre-flow-feature-start", preScript)

	postScript := `#!/bin/sh
echo "post-hook-ran" >> "` + markerFile + `"
echo "BRANCH=$BRANCH" >> "` + markerFile + `"
`
	createHookScript(t, mainRepo, "post-flow-feature-start", postScript)

	// Run git-flow command from worktree
	output, err := testutil.RunGitFlow(t, worktreePath, "feature", "start", "worktree-feature")
	if err != nil {
		t.Fatalf("Failed to start feature from worktree: %v\nOutput: %s", err, output)
	}

	// Verify hooks ran (marker file was created)
	content, err := os.ReadFile(markerFile)
	if err != nil {
		t.Fatalf("Hooks did not run from worktree - marker file not found: %v", err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, "pre-hook-ran") {
		t.Error("Pre-hook did not run from worktree")
	}
	if !strings.Contains(contentStr, "post-hook-ran") {
		t.Error("Post-hook did not run from worktree")
	}
	if !strings.Contains(contentStr, "BRANCH=feature/worktree-feature") {
		t.Errorf("Hook did not receive correct branch, got: %s", contentStr)
	}

	// Verify branch was created
	if !testutil.BranchExists(t, worktreePath, "feature/worktree-feature") {
		t.Error("Feature branch should have been created from worktree")
	}
}

// =============================================================================
// Publish Hook Tests - Verify hooks run for publish operations
// =============================================================================

// TestPublishPreHookBlocks tests that a failing pre-hook prevents publish operation.
func TestPublishPreHookBlocks(t *testing.T) {
	t.Parallel()
	dir, remoteDir := testutil.SetupTestRepoWithRemote(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer testutil.CleanupTestRepo(t, remoteDir)

	// Start a feature branch
	_, err := testutil.RunGitFlow(t, dir, "feature", "start", "publish-test")
	if err != nil {
		t.Fatalf("Failed to start feature: %v", err)
	}

	// Create a pre-hook that fails
	script := `#!/bin/sh
echo "Pre-hook blocking publish" >&2
exit 1
`
	createHookScript(t, dir, "pre-flow-feature-publish", script)

	// Try to publish - should fail
	output, err := testutil.RunGitFlow(t, dir, "feature", "publish", "publish-test")
	if err == nil {
		t.Fatal("Expected feature publish to fail due to pre-hook, but it succeeded")
	}

	// Verify the error mentions the hook
	if !strings.Contains(output, "pre-hook") {
		t.Errorf("Expected error to mention pre-hook, got: %s", output)
	}

	// Verify branch was NOT pushed to remote
	if testutil.RemoteBranchExists(t, dir, "origin", "feature/publish-test") {
		t.Error("Branch should not have been pushed when pre-hook failed")
	}
}

// TestPublishPostHookRuns tests that post-hook executes after successful publish.
func TestPublishPostHookRuns(t *testing.T) {
	t.Parallel()
	dir, remoteDir := testutil.SetupTestRepoWithRemote(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer testutil.CleanupTestRepo(t, remoteDir)

	// Start a feature branch
	_, err := testutil.RunGitFlow(t, dir, "feature", "start", "publish-hook-test")
	if err != nil {
		t.Fatalf("Failed to start feature: %v", err)
	}

	// Create a marker file path
	markerFile := filepath.Join(dir, "publish-hook-executed.txt")

	// Create a post-hook that creates a marker file
	script := `#!/bin/sh
echo "BRANCH=$BRANCH" > "` + markerFile + `"
echo "BRANCH_NAME=$BRANCH_NAME" >> "` + markerFile + `"
echo "ORIGIN=$ORIGIN" >> "` + markerFile + `"
`
	createHookScript(t, dir, "post-flow-feature-publish", script)

	// Publish the feature
	_, err = testutil.RunGitFlow(t, dir, "feature", "publish", "publish-hook-test")
	if err != nil {
		t.Fatalf("Failed to publish feature: %v", err)
	}

	// Verify post-hook ran by checking marker file
	content, err := os.ReadFile(markerFile)
	if err != nil {
		t.Fatalf("Post-hook did not run - marker file not found: %v", err)
	}

	// Verify environment variables were passed correctly
	contentStr := string(content)
	if !strings.Contains(contentStr, "BRANCH=feature/publish-hook-test") {
		t.Errorf("Expected BRANCH=feature/publish-hook-test in hook output, got: %s", contentStr)
	}
	if !strings.Contains(contentStr, "BRANCH_NAME=publish-hook-test") {
		t.Errorf("Expected BRANCH_NAME=publish-hook-test in hook output, got: %s", contentStr)
	}
	if !strings.Contains(contentStr, "ORIGIN=origin") {
		t.Errorf("Expected ORIGIN=origin in hook output, got: %s", contentStr)
	}
}

// =============================================================================
// Track Hook Tests - Verify hooks run for track operations
// =============================================================================

// TestTrackPreHookBlocks tests that a failing pre-hook prevents track operation.
func TestTrackPreHookBlocks(t *testing.T) {
	t.Parallel()
	dir, remoteDir := testutil.SetupTestRepoWithRemote(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer testutil.CleanupTestRepo(t, remoteDir)

	// Start and publish a feature branch to create it on remote
	_, err := testutil.RunGitFlow(t, dir, "feature", "start", "track-test")
	if err != nil {
		t.Fatalf("Failed to start feature: %v", err)
	}
	_, err = testutil.RunGitFlow(t, dir, "feature", "publish", "track-test")
	if err != nil {
		t.Fatalf("Failed to publish feature: %v", err)
	}

	// Delete local branch and switch to develop
	_, _ = testutil.RunGit(t, dir, "checkout", "develop")
	_, _ = testutil.RunGit(t, dir, "branch", "-D", "feature/track-test")

	// Create a pre-hook that fails
	script := `#!/bin/sh
echo "Pre-hook blocking track" >&2
exit 1
`
	createHookScript(t, dir, "pre-flow-feature-track", script)

	// Try to track - should fail
	output, err := testutil.RunGitFlow(t, dir, "feature", "track", "track-test")
	if err == nil {
		t.Fatal("Expected feature track to fail due to pre-hook, but it succeeded")
	}

	// Verify the error mentions the hook
	if !strings.Contains(output, "pre-hook") {
		t.Errorf("Expected error to mention pre-hook, got: %s", output)
	}

	// Verify local branch was NOT created
	if testutil.BranchExists(t, dir, "feature/track-test") {
		t.Error("Local branch should not have been created when pre-hook failed")
	}
}

// TestTrackPostHookRuns tests that post-hook executes after successful track.
func TestTrackPostHookRuns(t *testing.T) {
	t.Parallel()
	dir, remoteDir := testutil.SetupTestRepoWithRemote(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer testutil.CleanupTestRepo(t, remoteDir)

	// Start and publish a feature branch to create it on remote
	_, err := testutil.RunGitFlow(t, dir, "feature", "start", "track-hook-test")
	if err != nil {
		t.Fatalf("Failed to start feature: %v", err)
	}
	_, err = testutil.RunGitFlow(t, dir, "feature", "publish", "track-hook-test")
	if err != nil {
		t.Fatalf("Failed to publish feature: %v", err)
	}

	// Delete local branch and switch to develop
	_, _ = testutil.RunGit(t, dir, "checkout", "develop")
	_, _ = testutil.RunGit(t, dir, "branch", "-D", "feature/track-hook-test")

	// Create a marker file path
	markerFile := filepath.Join(dir, "track-hook-executed.txt")

	// Create a post-hook that creates a marker file
	script := `#!/bin/sh
echo "BRANCH=$BRANCH" > "` + markerFile + `"
echo "BRANCH_NAME=$BRANCH_NAME" >> "` + markerFile + `"
echo "ORIGIN=$ORIGIN" >> "` + markerFile + `"
`
	createHookScript(t, dir, "post-flow-feature-track", script)

	// Track the feature
	_, err = testutil.RunGitFlow(t, dir, "feature", "track", "track-hook-test")
	if err != nil {
		t.Fatalf("Failed to track feature: %v", err)
	}

	// Verify post-hook ran by checking marker file
	content, err := os.ReadFile(markerFile)
	if err != nil {
		t.Fatalf("Post-hook did not run - marker file not found: %v", err)
	}

	// Verify environment variables were passed correctly
	contentStr := string(content)
	if !strings.Contains(contentStr, "BRANCH=feature/track-hook-test") {
		t.Errorf("Expected BRANCH=feature/track-hook-test in hook output, got: %s", contentStr)
	}
	if !strings.Contains(contentStr, "BRANCH_NAME=track-hook-test") {
		t.Errorf("Expected BRANCH_NAME=track-hook-test in hook output, got: %s", contentStr)
	}
}

// =============================================================================
// Delete Post-Hook Test - Verify post-hook runs after delete
// =============================================================================

// TestDeletePostHookRuns tests that post-hook executes after successful delete.
func TestDeletePostHookRuns(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow and create a feature branch
	_, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v", err)
	}

	_, err = testutil.RunGitFlow(t, dir, "feature", "start", "to-delete-hook")
	if err != nil {
		t.Fatalf("Failed to start feature: %v", err)
	}

	// Switch back to develop so we can delete the feature branch
	_, _ = testutil.RunGit(t, dir, "checkout", "develop")

	// Create a marker file path
	markerFile := filepath.Join(dir, "delete-hook-executed.txt")

	// Create a post-hook that creates a marker file
	script := `#!/bin/sh
echo "BRANCH=$BRANCH" > "` + markerFile + `"
echo "BRANCH_NAME=$BRANCH_NAME" >> "` + markerFile + `"
echo "EXIT_CODE=$EXIT_CODE" >> "` + markerFile + `"
`
	createHookScript(t, dir, "post-flow-feature-delete", script)

	// Delete the feature
	_, err = testutil.RunGitFlow(t, dir, "feature", "delete", "to-delete-hook")
	if err != nil {
		t.Fatalf("Failed to delete feature: %v", err)
	}

	// Verify post-hook ran by checking marker file
	content, err := os.ReadFile(markerFile)
	if err != nil {
		t.Fatalf("Post-hook did not run - marker file not found: %v", err)
	}

	// Verify environment variables were passed correctly
	contentStr := string(content)
	if !strings.Contains(contentStr, "BRANCH=feature/to-delete-hook") {
		t.Errorf("Expected BRANCH=feature/to-delete-hook in hook output, got: %s", contentStr)
	}
	if !strings.Contains(contentStr, "EXIT_CODE=0") {
		t.Errorf("Expected EXIT_CODE=0 in hook output, got: %s", contentStr)
	}
}

// =============================================================================
// Custom Branch Type Hook Tests - Verify hooks work with custom branch configs
// =============================================================================

// TestCustomBranchTypePreHookBlocks tests that hooks work with custom branch types.
func TestCustomBranchTypePreHookBlocks(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow
	_, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v", err)
	}

	// Configure a custom branch type "bugfix"
	_, _ = testutil.RunGit(t, dir, "config", "gitflow.branch.bugfix.prefix", "bugfix/")
	_, _ = testutil.RunGit(t, dir, "config", "gitflow.branch.bugfix.parent", "develop")

	// Create a pre-hook that fails for the custom branch type
	script := `#!/bin/sh
echo "Pre-hook blocking bugfix start" >&2
exit 1
`
	createHookScript(t, dir, "pre-flow-bugfix-start", script)

	// Try to start a bugfix - should fail
	output, err := testutil.RunGitFlow(t, dir, "bugfix", "start", "custom-blocked")
	if err == nil {
		t.Fatal("Expected bugfix start to fail due to pre-hook, but it succeeded")
	}

	// Verify the error mentions the hook
	if !strings.Contains(output, "pre-hook") {
		t.Errorf("Expected error to mention pre-hook, got: %s", output)
	}

	// Verify branch was NOT created
	if testutil.BranchExists(t, dir, "bugfix/custom-blocked") {
		t.Error("Branch should not have been created when pre-hook failed")
	}
}

// TestCustomBranchTypePostHookReceivesCorrectType tests that post-hooks receive the correct branch type.
// Uses "support" branch type which is a standard git-flow type but configured with custom prefix.
func TestCustomBranchTypePostHookReceivesCorrectType(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow
	_, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v", err)
	}

	// Configure support branch with a custom prefix
	_, _ = testutil.RunGit(t, dir, "config", "gitflow.branch.support.prefix", "sup/")
	_, _ = testutil.RunGit(t, dir, "config", "gitflow.branch.support.parent", "main")

	// Create a marker file path
	markerFile := filepath.Join(dir, "custom-hook-executed.txt")

	// Create a post-hook that records the branch type
	script := `#!/bin/sh
echo "BRANCH=$BRANCH" > "` + markerFile + `"
echo "BRANCH_NAME=$BRANCH_NAME" >> "` + markerFile + `"
echo "BRANCH_TYPE=$BRANCH_TYPE" >> "` + markerFile + `"
echo "BASE_BRANCH=$BASE_BRANCH" >> "` + markerFile + `"
`
	createHookScript(t, dir, "post-flow-support-start", script)

	// Start a support branch
	_, err = testutil.RunGitFlow(t, dir, "support", "start", "lts-1.0")
	if err != nil {
		t.Fatalf("Failed to start support: %v", err)
	}

	// Verify post-hook ran by checking marker file
	content, err := os.ReadFile(markerFile)
	if err != nil {
		t.Fatalf("Post-hook did not run - marker file not found: %v", err)
	}

	// Verify environment variables were passed correctly for support type
	contentStr := string(content)
	if !strings.Contains(contentStr, "BRANCH=sup/lts-1.0") {
		t.Errorf("Expected BRANCH=sup/lts-1.0 in hook output, got: %s", contentStr)
	}
	if !strings.Contains(contentStr, "BRANCH_NAME=lts-1.0") {
		t.Errorf("Expected BRANCH_NAME=lts-1.0 in hook output, got: %s", contentStr)
	}
	if !strings.Contains(contentStr, "BRANCH_TYPE=support") {
		t.Errorf("Expected BRANCH_TYPE=support in hook output, got: %s", contentStr)
	}
	if !strings.Contains(contentStr, "BASE_BRANCH=main") {
		t.Errorf("Expected BASE_BRANCH=main in hook output, got: %s", contentStr)
	}
}

// =============================================================================
// Update Hook Tests - Verify hooks run for update operations
// =============================================================================

// TestUpdatePreHookBlocks tests that a failing pre-hook prevents update operation.
func TestUpdatePreHookBlocks(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow and create a feature branch
	_, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v", err)
	}

	_, err = testutil.RunGitFlow(t, dir, "feature", "start", "update-test")
	if err != nil {
		t.Fatalf("Failed to start feature: %v", err)
	}

	// Add a commit to the feature branch
	testutil.WriteFile(t, dir, "feature.txt", "feature content")
	_, _ = testutil.RunGit(t, dir, "add", "feature.txt")
	_, _ = testutil.RunGit(t, dir, "commit", "-m", "Add feature file")

	// Add a commit to develop (so there's something to update from)
	_, _ = testutil.RunGit(t, dir, "checkout", "develop")
	testutil.WriteFile(t, dir, "develop.txt", "develop content")
	_, _ = testutil.RunGit(t, dir, "add", "develop.txt")
	_, _ = testutil.RunGit(t, dir, "commit", "-m", "Add develop file")

	// Switch back to feature branch
	_, _ = testutil.RunGit(t, dir, "checkout", "feature/update-test")

	// Create a pre-hook that fails
	script := `#!/bin/sh
echo "Pre-hook blocking update" >&2
exit 1
`
	createHookScript(t, dir, "pre-flow-feature-update", script)

	// Try to update - should fail
	output, err := testutil.RunGitFlow(t, dir, "feature", "update", "update-test")
	if err == nil {
		t.Fatal("Expected feature update to fail due to pre-hook, but it succeeded")
	}

	// Verify the error mentions the hook
	if !strings.Contains(output, "pre-hook") {
		t.Errorf("Expected error to mention pre-hook, got: %s", output)
	}
}

// TestUpdatePostHookRuns tests that post-hook executes after successful update.
func TestUpdatePostHookRuns(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow and create a feature branch
	_, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v", err)
	}

	_, err = testutil.RunGitFlow(t, dir, "feature", "start", "update-hook-test")
	if err != nil {
		t.Fatalf("Failed to start feature: %v", err)
	}

	// Add a commit to the feature branch
	testutil.WriteFile(t, dir, "feature.txt", "feature content")
	_, _ = testutil.RunGit(t, dir, "add", "feature.txt")
	_, _ = testutil.RunGit(t, dir, "commit", "-m", "Add feature file")

	// Add a commit to develop (so there's something to update from)
	_, _ = testutil.RunGit(t, dir, "checkout", "develop")
	testutil.WriteFile(t, dir, "develop.txt", "develop content")
	_, _ = testutil.RunGit(t, dir, "add", "develop.txt")
	_, _ = testutil.RunGit(t, dir, "commit", "-m", "Add develop file")

	// Switch back to feature branch
	_, _ = testutil.RunGit(t, dir, "checkout", "feature/update-hook-test")

	// Create a marker file path
	markerFile := filepath.Join(dir, "update-hook-executed.txt")

	// Create a post-hook that creates a marker file
	script := `#!/bin/sh
echo "BRANCH=$BRANCH" > "` + markerFile + `"
echo "BRANCH_NAME=$BRANCH_NAME" >> "` + markerFile + `"
echo "BRANCH_TYPE=$BRANCH_TYPE" >> "` + markerFile + `"
echo "BASE_BRANCH=$BASE_BRANCH" >> "` + markerFile + `"
echo "EXIT_CODE=$EXIT_CODE" >> "` + markerFile + `"
`
	createHookScript(t, dir, "post-flow-feature-update", script)

	// Update the feature
	_, err = testutil.RunGitFlow(t, dir, "feature", "update", "update-hook-test")
	if err != nil {
		t.Fatalf("Failed to update feature: %v", err)
	}

	// Verify post-hook ran by checking marker file
	content, err := os.ReadFile(markerFile)
	if err != nil {
		t.Fatalf("Post-hook did not run - marker file not found: %v", err)
	}

	// Verify environment variables were passed correctly
	contentStr := string(content)
	if !strings.Contains(contentStr, "BRANCH=feature/update-hook-test") {
		t.Errorf("Expected BRANCH=feature/update-hook-test in hook output, got: %s", contentStr)
	}
	if !strings.Contains(contentStr, "BRANCH_NAME=update-hook-test") {
		t.Errorf("Expected BRANCH_NAME=update-hook-test in hook output, got: %s", contentStr)
	}
	if !strings.Contains(contentStr, "BRANCH_TYPE=feature") {
		t.Errorf("Expected BRANCH_TYPE=feature in hook output, got: %s", contentStr)
	}
	if !strings.Contains(contentStr, "BASE_BRANCH=develop") {
		t.Errorf("Expected BASE_BRANCH=develop in hook output, got: %s", contentStr)
	}
	if !strings.Contains(contentStr, "EXIT_CODE=0") {
		t.Errorf("Expected EXIT_CODE=0 in hook output, got: %s", contentStr)
	}
}

// TestUpdateShorthandHookRuns tests that hooks work with the shorthand update command.
func TestUpdateShorthandHookRuns(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow and create a feature branch
	_, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v", err)
	}

	_, err = testutil.RunGitFlow(t, dir, "feature", "start", "shorthand-update")
	if err != nil {
		t.Fatalf("Failed to start feature: %v", err)
	}

	// Add a commit to the feature branch
	testutil.WriteFile(t, dir, "feature.txt", "feature content")
	_, _ = testutil.RunGit(t, dir, "add", "feature.txt")
	_, _ = testutil.RunGit(t, dir, "commit", "-m", "Add feature file")

	// Add a commit to develop
	_, _ = testutil.RunGit(t, dir, "checkout", "develop")
	testutil.WriteFile(t, dir, "develop.txt", "develop content")
	_, _ = testutil.RunGit(t, dir, "add", "develop.txt")
	_, _ = testutil.RunGit(t, dir, "commit", "-m", "Add develop file")

	// Switch back to feature branch
	_, _ = testutil.RunGit(t, dir, "checkout", "feature/shorthand-update")

	// Create a marker file path
	markerFile := filepath.Join(dir, "shorthand-update-hook.txt")

	// Create a post-hook
	script := `#!/bin/sh
echo "BRANCH_TYPE=$BRANCH_TYPE" > "` + markerFile + `"
echo "BRANCH_NAME=$BRANCH_NAME" >> "` + markerFile + `"
`
	createHookScript(t, dir, "post-flow-feature-update", script)

	// Use shorthand update command (git flow update)
	_, err = testutil.RunGitFlow(t, dir, "update")
	if err != nil {
		t.Fatalf("Failed to run shorthand update: %v", err)
	}

	// Verify post-hook ran
	content, err := os.ReadFile(markerFile)
	if err != nil {
		t.Fatalf("Post-hook did not run with shorthand command - marker file not found: %v", err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, "BRANCH_TYPE=feature") {
		t.Errorf("Expected BRANCH_TYPE=feature, got: %s", contentStr)
	}
	if !strings.Contains(contentStr, "BRANCH_NAME=shorthand-update") {
		t.Errorf("Expected BRANCH_NAME=shorthand-update, got: %s", contentStr)
	}
}

// =============================================================================
// Positional Arguments Tests - Verify git-flow-avh compatibility
// =============================================================================

// TestStartHookReceivesPositionalArguments tests that start hooks receive positional arguments
// matching git-flow-avh convention: $1=name, $2=origin, $3=branch, $4=base.
func TestStartHookReceivesPositionalArguments(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow
	_, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v", err)
	}

	// Create a marker file path
	markerFile := filepath.Join(dir, "start-args.txt")

	// Create a pre-hook that records positional arguments
	script := `#!/bin/sh
echo "ARG1=$1" > "` + markerFile + `"
echo "ARG2=$2" >> "` + markerFile + `"
echo "ARG3=$3" >> "` + markerFile + `"
echo "ARG4=$4" >> "` + markerFile + `"
echo "ARG_COUNT=$#" >> "` + markerFile + `"
`
	createHookScript(t, dir, "pre-flow-feature-start", script)

	// Start a feature
	_, err = testutil.RunGitFlow(t, dir, "feature", "start", "arg-test")
	if err != nil {
		t.Fatalf("Failed to start feature: %v", err)
	}

	// Verify hook received correct positional arguments
	content, err := os.ReadFile(markerFile)
	if err != nil {
		t.Fatalf("Hook did not run - marker file not found: %v", err)
	}

	contentStr := string(content)
	// $1 = name (short branch name)
	if !strings.Contains(contentStr, "ARG1=arg-test") {
		t.Errorf("Expected ARG1=arg-test, got: %s", contentStr)
	}
	// $2 = origin
	if !strings.Contains(contentStr, "ARG2=origin") {
		t.Errorf("Expected ARG2=origin, got: %s", contentStr)
	}
	// $3 = full branch name
	if !strings.Contains(contentStr, "ARG3=feature/arg-test") {
		t.Errorf("Expected ARG3=feature/arg-test, got: %s", contentStr)
	}
	// $4 = base branch (for start action)
	if !strings.Contains(contentStr, "ARG4=develop") {
		t.Errorf("Expected ARG4=develop, got: %s", contentStr)
	}
	// Start hook should receive 4 arguments
	if !strings.Contains(contentStr, "ARG_COUNT=4") {
		t.Errorf("Expected ARG_COUNT=4, got: %s", contentStr)
	}
}

// TestFinishHookReceivesPositionalArguments tests that finish hooks receive positional arguments
// matching git-flow-avh convention: $1=name, $2=origin, $3=branch.
func TestFinishHookReceivesPositionalArguments(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow and create a feature branch
	_, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v", err)
	}

	_, err = testutil.RunGitFlow(t, dir, "feature", "start", "finish-arg-test")
	if err != nil {
		t.Fatalf("Failed to start feature: %v", err)
	}

	// Add a commit
	testutil.WriteFile(t, dir, "test.txt", "test content")
	_, _ = testutil.RunGit(t, dir, "add", "test.txt")
	_, _ = testutil.RunGit(t, dir, "commit", "-m", "Add test file")

	// Create a marker file path
	markerFile := filepath.Join(dir, "finish-args.txt")

	// Create a pre-hook that records positional arguments
	script := `#!/bin/sh
echo "ARG1=$1" > "` + markerFile + `"
echo "ARG2=$2" >> "` + markerFile + `"
echo "ARG3=$3" >> "` + markerFile + `"
echo "ARG_COUNT=$#" >> "` + markerFile + `"
`
	createHookScript(t, dir, "pre-flow-feature-finish", script)

	// Finish the feature
	_, err = testutil.RunGitFlow(t, dir, "feature", "finish", "finish-arg-test")
	if err != nil {
		t.Fatalf("Failed to finish feature: %v", err)
	}

	// Verify hook received correct positional arguments
	content, err := os.ReadFile(markerFile)
	if err != nil {
		t.Fatalf("Hook did not run - marker file not found: %v", err)
	}

	contentStr := string(content)
	// $1 = name (short branch name)
	if !strings.Contains(contentStr, "ARG1=finish-arg-test") {
		t.Errorf("Expected ARG1=finish-arg-test, got: %s", contentStr)
	}
	// $2 = origin
	if !strings.Contains(contentStr, "ARG2=origin") {
		t.Errorf("Expected ARG2=origin, got: %s", contentStr)
	}
	// $3 = full branch name
	if !strings.Contains(contentStr, "ARG3=feature/finish-arg-test") {
		t.Errorf("Expected ARG3=feature/finish-arg-test, got: %s", contentStr)
	}
	// Finish hook should receive 3 arguments
	if !strings.Contains(contentStr, "ARG_COUNT=3") {
		t.Errorf("Expected ARG_COUNT=3, got: %s", contentStr)
	}
}

// TestUpdateHookReceivesPositionalArguments tests that update hooks receive positional arguments.
// Update is a git-flow-next extension: $1=name, $2=origin, $3=branch, $4=base.
func TestUpdateHookReceivesPositionalArguments(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow and create a feature branch
	_, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v", err)
	}

	_, err = testutil.RunGitFlow(t, dir, "feature", "start", "update-arg-test")
	if err != nil {
		t.Fatalf("Failed to start feature: %v", err)
	}

	// Add a commit to the feature branch
	testutil.WriteFile(t, dir, "feature.txt", "feature content")
	_, _ = testutil.RunGit(t, dir, "add", "feature.txt")
	_, _ = testutil.RunGit(t, dir, "commit", "-m", "Add feature file")

	// Add a commit to develop (so there's something to update from)
	_, _ = testutil.RunGit(t, dir, "checkout", "develop")
	testutil.WriteFile(t, dir, "develop.txt", "develop content")
	_, _ = testutil.RunGit(t, dir, "add", "develop.txt")
	_, _ = testutil.RunGit(t, dir, "commit", "-m", "Add develop file")

	// Switch back to feature branch
	_, _ = testutil.RunGit(t, dir, "checkout", "feature/update-arg-test")

	// Create a marker file path
	markerFile := filepath.Join(dir, "update-args.txt")

	// Create a post-hook that records positional arguments
	script := `#!/bin/sh
echo "ARG1=$1" > "` + markerFile + `"
echo "ARG2=$2" >> "` + markerFile + `"
echo "ARG3=$3" >> "` + markerFile + `"
echo "ARG4=$4" >> "` + markerFile + `"
echo "ARG_COUNT=$#" >> "` + markerFile + `"
`
	createHookScript(t, dir, "post-flow-feature-update", script)

	// Update the feature
	_, err = testutil.RunGitFlow(t, dir, "feature", "update", "update-arg-test")
	if err != nil {
		t.Fatalf("Failed to update feature: %v", err)
	}

	// Verify hook received correct positional arguments
	content, err := os.ReadFile(markerFile)
	if err != nil {
		t.Fatalf("Hook did not run - marker file not found: %v", err)
	}

	contentStr := string(content)
	// $1 = name (short branch name)
	if !strings.Contains(contentStr, "ARG1=update-arg-test") {
		t.Errorf("Expected ARG1=update-arg-test, got: %s", contentStr)
	}
	// $2 = origin
	if !strings.Contains(contentStr, "ARG2=origin") {
		t.Errorf("Expected ARG2=origin, got: %s", contentStr)
	}
	// $3 = full branch name
	if !strings.Contains(contentStr, "ARG3=feature/update-arg-test") {
		t.Errorf("Expected ARG3=feature/update-arg-test, got: %s", contentStr)
	}
	// $4 = base branch
	if !strings.Contains(contentStr, "ARG4=develop") {
		t.Errorf("Expected ARG4=develop, got: %s", contentStr)
	}
	// Update hook should receive 4 arguments
	if !strings.Contains(contentStr, "ARG_COUNT=4") {
		t.Errorf("Expected ARG_COUNT=4, got: %s", contentStr)
	}
}

// TestHookPositionalArgsMatchEnvVars tests that positional arguments match environment variables.
// This is critical for git-flow-avh compatibility.
func TestHookPositionalArgsMatchEnvVars(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow
	_, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v", err)
	}

	// Create a hook that compares positional args with env vars
	script := `#!/bin/sh
# Verify $1 equals $BRANCH_NAME
if [ "$1" != "$BRANCH_NAME" ]; then
    echo "Mismatch: \$1='$1' vs BRANCH_NAME='$BRANCH_NAME'" >&2
    exit 1
fi

# Verify $2 equals $ORIGIN
if [ "$2" != "$ORIGIN" ]; then
    echo "Mismatch: \$2='$2' vs ORIGIN='$ORIGIN'" >&2
    exit 1
fi

# Verify $3 equals $BRANCH
if [ "$3" != "$BRANCH" ]; then
    echo "Mismatch: \$3='$3' vs BRANCH='$BRANCH'" >&2
    exit 1
fi

# Verify $4 equals $BASE_BRANCH (for start action)
if [ "$4" != "$BASE_BRANCH" ]; then
    echo "Mismatch: \$4='$4' vs BASE_BRANCH='$BASE_BRANCH'" >&2
    exit 1
fi

exit 0
`
	createHookScript(t, dir, "pre-flow-feature-start", script)

	// Start a feature - hook will fail if args don't match env vars
	output, err := testutil.RunGitFlow(t, dir, "feature", "start", "consistency-test")
	if err != nil {
		t.Fatalf("Hook failed - positional args don't match env vars: %v\nOutput: %s", err, output)
	}

	// Verify branch was created (operation succeeded)
	if !testutil.BranchExists(t, dir, "feature/consistency-test") {
		t.Error("Branch should have been created when hook passed")
	}
}

// TestHotfixVersionFilter tests that version filters work with hotfix branch type.
// This verifies filters work with branch types beyond just release.
func TestHotfixVersionFilter(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow
	_, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v", err)
	}

	// Create a version filter that adds 'hotfix-' prefix for hotfixes
	script := `#!/bin/sh
VERSION="$1"
if [ "${VERSION#hotfix-}" = "$VERSION" ]; then
    echo "hotfix-$VERSION"
else
    echo "$VERSION"
fi
`
	createHookScript(t, dir, "filter-flow-hotfix-start-version", script)

	// Start a hotfix - filter should change "1.0.1" to "hotfix-1.0.1"
	output, err := testutil.RunGitFlow(t, dir, "hotfix", "start", "1.0.1")
	if err != nil {
		t.Fatalf("Failed to start hotfix: %v\nOutput: %s", err, output)
	}

	// Verify the filter message is shown
	if !strings.Contains(output, "Version filter changed") {
		t.Errorf("Expected output to mention version filter, got: %s", output)
	}

	// Verify the branch was created with the filtered name
	if !testutil.BranchExists(t, dir, "hotfix/hotfix-1.0.1") {
		t.Error("Expected hotfix/hotfix-1.0.1 branch to exist (filtered from 1.0.1)")
	}

	// Verify original name branch was NOT created
	if testutil.BranchExists(t, dir, "hotfix/1.0.1") {
		t.Error("hotfix/1.0.1 should not exist - filter should have changed it to hotfix-1.0.1")
	}
}
