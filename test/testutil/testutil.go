package testutil

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gittower/git-flow-next/internal/mergestate"
)

// ExitError represents an error with an exit code
type ExitError struct {
	ExitCode int
	Err      error
}

func (e *ExitError) Error() string {
	return e.Err.Error()
}

// getGitDirForRepo returns the git directory for the specified repository.
// This handles both regular repos (returns ".git") and worktrees (returns the actual git dir path).
func getGitDirForRepo(dir string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	cmd.Dir = dir
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get git directory: %w", err)
	}
	gitDir := strings.TrimSpace(string(output))
	// If the path is relative, make it absolute relative to dir
	if !filepath.IsAbs(gitDir) {
		gitDir = filepath.Join(dir, gitDir)
	}
	return gitDir, nil
}

// LoadMergeState loads the merge state from the test repository
func LoadMergeState(t *testing.T, dir string) (*mergestate.MergeState, error) {
	gitDir, err := getGitDirForRepo(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to determine git directory: %w", err)
	}
	stateFile := filepath.Join(gitDir, "gitflow", "state", "merge.json")
	data, err := os.ReadFile(stateFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read merge state file: %w", err)
	}

	var state mergestate.MergeState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to parse merge state file: %w", err)
	}

	return &state, nil
}

// IsMergeInProgress checks if a merge is in progress in the test repository
func IsMergeInProgress(t *testing.T, dir string) bool {
	// Check for MERGE_HEAD in git directory which indicates a merge in progress
	gitDir, err := getGitDirForRepo(dir)
	if err != nil {
		return false
	}
	_, err = os.Stat(filepath.Join(gitDir, "MERGE_HEAD"))
	return !os.IsNotExist(err)
}

// ReadFile reads a file from the test repository
func ReadFile(t *testing.T, dir string, name string) string {
	content, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		t.Fatalf("Failed to read file %s: %v", name, err)
	}
	return string(content)
}

// FileExists checks if a file exists in the repository
func FileExists(t *testing.T, dir string, path string) bool {
	t.Helper()
	fullPath := filepath.Join(dir, path)
	_, err := os.Stat(fullPath)
	return err == nil
}

// GitFlowMergeStateExists checks if the git-flow merge state file exists
func GitFlowMergeStateExists(t *testing.T, dir string) bool {
	t.Helper()
	gitDir, err := getGitDirForRepo(dir)
	if err != nil {
		t.Fatalf("Failed to determine git directory for repo %s: %v", dir, err)
	}
	stateFile := filepath.Join(gitDir, "gitflow", "state", "merge.json")
	_, err = os.Stat(stateFile)
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}
	t.Fatalf("Failed to stat git-flow merge state file %s: %v", stateFile, err)
	return false
}
