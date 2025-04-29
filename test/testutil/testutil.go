package testutil

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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

// LoadMergeState loads the merge state from the test repository
func LoadMergeState(t *testing.T, dir string) (*mergestate.MergeState, error) {
	stateFile := filepath.Join(dir, ".git", "gitflow", "state", "merge.json")
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
	// Check for .git/MERGE_HEAD which indicates a merge in progress
	_, err := os.Stat(filepath.Join(dir, ".git", "MERGE_HEAD"))
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
