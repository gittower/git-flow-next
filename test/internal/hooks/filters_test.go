package hooks_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/gittower/git-flow-next/internal/hooks"
	"github.com/gittower/git-flow-next/test/testutil"
)

// createExecutableScript creates an executable script in the hooks directory.
func createExecutableScript(t *testing.T, dir, name, content string) {
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

// createNonExecutableScript creates a non-executable script in the hooks directory.
func createNonExecutableScript(t *testing.T, dir, name, content string) {
	t.Helper()
	hooksDir := filepath.Join(dir, ".git", "hooks")
	if err := os.MkdirAll(hooksDir, 0755); err != nil {
		t.Fatalf("Failed to create hooks directory: %v", err)
	}

	scriptPath := filepath.Join(hooksDir, name)
	if err := os.WriteFile(scriptPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create script %s: %v", name, err)
	}
}

// TestVersionFilterModifiesVersion tests that the version filter modifies the version correctly.
func TestVersionFilterModifiesVersion(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Create a version filter script that adds 'v' prefix
	script := `#!/bin/sh
VERSION="$1"
if [ "${VERSION#v}" = "$VERSION" ]; then
    echo "v$VERSION"
else
    echo "$VERSION"
fi
`
	createExecutableScript(t, dir, "filter-flow-release-start-version", script)

	gitDir := filepath.Join(dir, ".git")
	result, err := hooks.RunVersionFilter(gitDir, "release", "1.0.0")
	if err != nil {
		t.Fatalf("RunVersionFilter failed: %v", err)
	}

	if result != "v1.0.0" {
		t.Errorf("Expected 'v1.0.0', got '%s'", result)
	}
}

// TestVersionFilterPreservesExistingPrefix tests that version filter preserves existing prefix.
func TestVersionFilterPreservesExistingPrefix(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Create a version filter script that adds 'v' prefix only if not present
	script := `#!/bin/sh
VERSION="$1"
if [ "${VERSION#v}" = "$VERSION" ]; then
    echo "v$VERSION"
else
    echo "$VERSION"
fi
`
	createExecutableScript(t, dir, "filter-flow-release-start-version", script)

	gitDir := filepath.Join(dir, ".git")
	result, err := hooks.RunVersionFilter(gitDir, "release", "v1.0.0")
	if err != nil {
		t.Fatalf("RunVersionFilter failed: %v", err)
	}

	if result != "v1.0.0" {
		t.Errorf("Expected 'v1.0.0', got '%s'", result)
	}
}

// TestVersionFilterNonExistentReturnsOriginal tests that non-existent filter returns original version.
func TestVersionFilterNonExistentReturnsOriginal(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	gitDir := filepath.Join(dir, ".git")
	result, err := hooks.RunVersionFilter(gitDir, "release", "1.0.0")
	if err != nil {
		t.Fatalf("RunVersionFilter failed: %v", err)
	}

	if result != "1.0.0" {
		t.Errorf("Expected '1.0.0', got '%s'", result)
	}
}

// TestVersionFilterNonExecutableSkipped tests that non-executable filter is skipped.
func TestVersionFilterNonExecutableSkipped(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows does not distinguish executable permissions via file mode bits")
	}
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Create a non-executable version filter script
	script := `#!/bin/sh
echo "modified-$1"
`
	createNonExecutableScript(t, dir, "filter-flow-release-start-version", script)

	gitDir := filepath.Join(dir, ".git")
	result, err := hooks.RunVersionFilter(gitDir, "release", "1.0.0")
	if err != nil {
		t.Fatalf("RunVersionFilter failed: %v", err)
	}

	// Should return original since script is not executable
	if result != "1.0.0" {
		t.Errorf("Expected '1.0.0' (original), got '%s'", result)
	}
}

// TestVersionFilterNonZeroExitReturnsError tests that filter with non-zero exit returns error.
func TestVersionFilterNonZeroExitReturnsError(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Create a version filter script that exits with error
	script := `#!/bin/sh
echo "Error: invalid version" >&2
exit 1
`
	createExecutableScript(t, dir, "filter-flow-release-start-version", script)

	gitDir := filepath.Join(dir, ".git")
	_, err := hooks.RunVersionFilter(gitDir, "release", "1.0.0")
	if err == nil {
		t.Fatal("Expected error for non-zero exit code, got nil")
	}
}

// TestVersionFilterEmptyOutputReturnsOriginal tests that empty output returns original version.
func TestVersionFilterEmptyOutputReturnsOriginal(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Create a version filter script that outputs nothing
	script := `#!/bin/sh
# Do nothing
`
	createExecutableScript(t, dir, "filter-flow-release-start-version", script)

	gitDir := filepath.Join(dir, ".git")
	result, err := hooks.RunVersionFilter(gitDir, "release", "1.0.0")
	if err != nil {
		t.Fatalf("RunVersionFilter failed: %v", err)
	}

	// Should return original since output is empty
	if result != "1.0.0" {
		t.Errorf("Expected '1.0.0' (original), got '%s'", result)
	}
}

// TestVersionFilterHotfix tests version filter for hotfix branches.
func TestVersionFilterHotfix(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Create a hotfix version filter script
	script := `#!/bin/sh
echo "hotfix-$1"
`
	createExecutableScript(t, dir, "filter-flow-hotfix-start-version", script)

	gitDir := filepath.Join(dir, ".git")
	result, err := hooks.RunVersionFilter(gitDir, "hotfix", "1.0.1")
	if err != nil {
		t.Fatalf("RunVersionFilter failed: %v", err)
	}

	if result != "hotfix-1.0.1" {
		t.Errorf("Expected 'hotfix-1.0.1', got '%s'", result)
	}
}

// TestVersionFilterCustomBranchType tests version filter for custom branch types.
func TestVersionFilterCustomBranchType(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Create a custom branch type version filter script
	script := `#!/bin/sh
echo "custom-$1"
`
	createExecutableScript(t, dir, "filter-flow-epic-start-version", script)

	gitDir := filepath.Join(dir, ".git")
	result, err := hooks.RunVersionFilter(gitDir, "epic", "my-epic")
	if err != nil {
		t.Fatalf("RunVersionFilter failed: %v", err)
	}

	if result != "custom-my-epic" {
		t.Errorf("Expected 'custom-my-epic', got '%s'", result)
	}
}

// TestTagMessageFilterModifiesMessage tests that the tag message filter modifies the message correctly.
func TestTagMessageFilterModifiesMessage(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Create a tag message filter script that appends to the message
	script := `#!/bin/sh
VERSION="$1"
MESSAGE="$2"
echo "${MESSAGE}

Release ${VERSION} - Auto-generated"
`
	createExecutableScript(t, dir, "filter-flow-release-finish-tag-message", script)

	gitDir := filepath.Join(dir, ".git")
	ctx := hooks.FilterContext{
		BranchType: "release",
		BranchName: "1.0.0",
		Version:    "1.0.0",
		TagMessage: "Release version 1.0.0",
		BaseBranch: "main",
	}

	result, err := hooks.RunTagMessageFilter(gitDir, "release", ctx)
	if err != nil {
		t.Fatalf("RunTagMessageFilter failed: %v", err)
	}

	expected := "Release version 1.0.0\n\nRelease 1.0.0 - Auto-generated"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

// TestTagMessageFilterNonExistentReturnsOriginal tests that non-existent filter returns original message.
func TestTagMessageFilterNonExistentReturnsOriginal(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	gitDir := filepath.Join(dir, ".git")
	ctx := hooks.FilterContext{
		BranchType: "release",
		BranchName: "1.0.0",
		Version:    "1.0.0",
		TagMessage: "Release version 1.0.0",
		BaseBranch: "main",
	}

	result, err := hooks.RunTagMessageFilter(gitDir, "release", ctx)
	if err != nil {
		t.Fatalf("RunTagMessageFilter failed: %v", err)
	}

	if result != "Release version 1.0.0" {
		t.Errorf("Expected 'Release version 1.0.0', got '%s'", result)
	}
}

// TestTagMessageFilterNonExecutableSkipped tests that non-executable filter is skipped.
func TestTagMessageFilterNonExecutableSkipped(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows does not distinguish executable permissions via file mode bits")
	}
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Create a non-executable tag message filter script
	script := `#!/bin/sh
echo "modified message"
`
	createNonExecutableScript(t, dir, "filter-flow-release-finish-tag-message", script)

	gitDir := filepath.Join(dir, ".git")
	ctx := hooks.FilterContext{
		BranchType: "release",
		BranchName: "1.0.0",
		Version:    "1.0.0",
		TagMessage: "Original message",
		BaseBranch: "main",
	}

	result, err := hooks.RunTagMessageFilter(gitDir, "release", ctx)
	if err != nil {
		t.Fatalf("RunTagMessageFilter failed: %v", err)
	}

	// Should return original since script is not executable
	if result != "Original message" {
		t.Errorf("Expected 'Original message', got '%s'", result)
	}
}

// TestTagMessageFilterNonZeroExitReturnsError tests that filter with non-zero exit returns error.
func TestTagMessageFilterNonZeroExitReturnsError(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Create a tag message filter script that exits with error
	script := `#!/bin/sh
exit 1
`
	createExecutableScript(t, dir, "filter-flow-release-finish-tag-message", script)

	gitDir := filepath.Join(dir, ".git")
	ctx := hooks.FilterContext{
		BranchType: "release",
		BranchName: "1.0.0",
		Version:    "1.0.0",
		TagMessage: "Original message",
		BaseBranch: "main",
	}

	_, err := hooks.RunTagMessageFilter(gitDir, "release", ctx)
	if err == nil {
		t.Fatal("Expected error for non-zero exit code, got nil")
	}
}

// TestTagMessageFilterHotfix tests tag message filter for hotfix branches.
func TestTagMessageFilterHotfix(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Create a hotfix tag message filter script
	script := `#!/bin/sh
VERSION="$1"
echo "Hotfix ${VERSION}"
`
	createExecutableScript(t, dir, "filter-flow-hotfix-finish-tag-message", script)

	gitDir := filepath.Join(dir, ".git")
	ctx := hooks.FilterContext{
		BranchType: "hotfix",
		BranchName: "1.0.1",
		Version:    "1.0.1",
		TagMessage: "Original message",
		BaseBranch: "main",
	}

	result, err := hooks.RunTagMessageFilter(gitDir, "hotfix", ctx)
	if err != nil {
		t.Fatalf("RunTagMessageFilter failed: %v", err)
	}

	if result != "Hotfix 1.0.1" {
		t.Errorf("Expected 'Hotfix 1.0.1', got '%s'", result)
	}
}

// TestTagMessageFilterCustomBranchType tests tag message filter for custom branch types.
func TestTagMessageFilterCustomBranchType(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Create a custom branch type tag message filter script
	script := `#!/bin/sh
VERSION="$1"
echo "Epic ${VERSION} completed"
`
	createExecutableScript(t, dir, "filter-flow-epic-finish-tag-message", script)

	gitDir := filepath.Join(dir, ".git")
	ctx := hooks.FilterContext{
		BranchType: "epic",
		BranchName: "my-epic",
		Version:    "my-epic",
		TagMessage: "Original message",
		BaseBranch: "main",
	}

	result, err := hooks.RunTagMessageFilter(gitDir, "epic", ctx)
	if err != nil {
		t.Fatalf("RunTagMessageFilter failed: %v", err)
	}

	if result != "Epic my-epic completed" {
		t.Errorf("Expected 'Epic my-epic completed', got '%s'", result)
	}
}

// TestTagMessageFilterReceivesEnvironmentVariables tests that filter receives environment variables.
func TestTagMessageFilterReceivesEnvironmentVariables(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Create a tag message filter script that uses environment variables
	script := `#!/bin/sh
echo "Type: ${BRANCH_TYPE}, Name: ${BRANCH_NAME}, Base: ${BASE_BRANCH}, Version: ${VERSION}"
`
	createExecutableScript(t, dir, "filter-flow-release-finish-tag-message", script)

	gitDir := filepath.Join(dir, ".git")
	ctx := hooks.FilterContext{
		BranchType: "release",
		BranchName: "my-release",
		Version:    "2.0.0",
		TagMessage: "Original",
		BaseBranch: "main",
	}

	result, err := hooks.RunTagMessageFilter(gitDir, "release", ctx)
	if err != nil {
		t.Fatalf("RunTagMessageFilter failed: %v", err)
	}

	expected := "Type: release, Name: my-release, Base: main, Version: 2.0.0"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

// TestGetFilterName tests the dynamic filter name generation.
func TestGetFilterName(t *testing.T) {
	tests := []struct {
		branchType string
		action     string
		target     hooks.FilterTarget
		expected   string
	}{
		{"release", "start", hooks.FilterTargetVersion, "filter-flow-release-start-version"},
		{"hotfix", "start", hooks.FilterTargetVersion, "filter-flow-hotfix-start-version"},
		{"feature", "start", hooks.FilterTargetVersion, "filter-flow-feature-start-version"},
		{"epic", "start", hooks.FilterTargetVersion, "filter-flow-epic-start-version"},
		{"release", "finish", hooks.FilterTargetTagMessage, "filter-flow-release-finish-tag-message"},
		{"hotfix", "finish", hooks.FilterTargetTagMessage, "filter-flow-hotfix-finish-tag-message"},
		{"custom", "finish", hooks.FilterTargetTagMessage, "filter-flow-custom-finish-tag-message"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := hooks.GetFilterName(tt.branchType, tt.action, tt.target)
			if result != tt.expected {
				t.Errorf("GetFilterName(%q, %q, %q) = %q, want %q",
					tt.branchType, tt.action, tt.target, result, tt.expected)
			}
		})
	}
}

// TestVersionFilterWorksInGitWorktree tests that version filters work correctly in a git worktree.
func TestVersionFilterWorksInGitWorktree(t *testing.T) {
	// Setup main repository
	mainRepo := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, mainRepo)

	// Create a worktree
	worktreePath, err := os.MkdirTemp("", "git-flow-worktree-filter-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory for worktree: %v", err)
	}
	defer os.RemoveAll(worktreePath)
	os.RemoveAll(worktreePath)

	_, err = testutil.RunGit(t, mainRepo, "worktree", "add", worktreePath, "-b", "filter-test-branch")
	if err != nil {
		t.Fatalf("Failed to create worktree: %v", err)
	}

	// Create a version filter in the main repository's hooks directory
	script := `#!/bin/sh
VERSION="$1"
echo "v${VERSION}"
`
	createExecutableScript(t, mainRepo, "filter-flow-release-start-version", script)

	// Get the worktree's git directory
	oldDir, _ := os.Getwd()
	os.Chdir(worktreePath)
	worktreeGitDirOutput, err := testutil.RunGit(t, worktreePath, "rev-parse", "--git-dir")
	os.Chdir(oldDir)
	if err != nil {
		t.Fatalf("Failed to get worktree git directory: %v", err)
	}
	worktreeGitDir := strings.TrimSpace(worktreeGitDirOutput)

	// Run version filter from worktree context
	result, err := hooks.RunVersionFilter(worktreeGitDir, "release", "1.0.0")
	if err != nil {
		t.Fatalf("RunVersionFilter failed in worktree: %v", err)
	}

	if result != "v1.0.0" {
		t.Errorf("Expected 'v1.0.0', got '%s'", result)
	}
}

// TestTagMessageFilterWorksInGitWorktree tests that tag message filters work correctly in a git worktree.
func TestTagMessageFilterWorksInGitWorktree(t *testing.T) {
	// Setup main repository
	mainRepo := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, mainRepo)

	// Create a worktree
	worktreePath, err := os.MkdirTemp("", "git-flow-worktree-tagfilter-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory for worktree: %v", err)
	}
	defer os.RemoveAll(worktreePath)
	os.RemoveAll(worktreePath)

	_, err = testutil.RunGit(t, mainRepo, "worktree", "add", worktreePath, "-b", "tagfilter-test-branch")
	if err != nil {
		t.Fatalf("Failed to create worktree: %v", err)
	}

	// Create a tag message filter in the main repository's hooks directory
	script := `#!/bin/sh
VERSION="$1"
echo "Release ${VERSION} from worktree"
`
	createExecutableScript(t, mainRepo, "filter-flow-release-finish-tag-message", script)

	// Get the worktree's git directory
	oldDir, _ := os.Getwd()
	os.Chdir(worktreePath)
	worktreeGitDirOutput, err := testutil.RunGit(t, worktreePath, "rev-parse", "--git-dir")
	os.Chdir(oldDir)
	if err != nil {
		t.Fatalf("Failed to get worktree git directory: %v", err)
	}
	worktreeGitDir := strings.TrimSpace(worktreeGitDirOutput)

	ctx := hooks.FilterContext{
		BranchType: "release",
		BranchName: "2.0.0",
		Version:    "2.0.0",
		TagMessage: "Original message",
		BaseBranch: "main",
	}

	// Run tag message filter from worktree context
	result, err := hooks.RunTagMessageFilter(worktreeGitDir, "release", ctx)
	if err != nil {
		t.Fatalf("RunTagMessageFilter failed in worktree: %v", err)
	}

	if result != "Release 2.0.0 from worktree" {
		t.Errorf("Expected 'Release 2.0.0 from worktree', got '%s'", result)
	}
}
