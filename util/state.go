package util

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/gittower/git-flow-next/model"
)

const (
	stateDir  = ".git/gitflow/state"
	stateFile = "merge.json"
)

// SaveMergeState saves the current merge state to a file
func SaveMergeState(state *model.MergeState) error {
	// Create state directory if it doesn't exist
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		return fmt.Errorf("failed to create state directory: %w", err)
	}

	// Marshal state to JSON
	data, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	// Write state to file
	statePath := filepath.Join(stateDir, stateFile)
	if err := os.WriteFile(statePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write state file: %w", err)
	}

	return nil
}

// LoadMergeState loads the current merge state from file
func LoadMergeState() (*model.MergeState, error) {
	statePath := filepath.Join(stateDir, stateFile)
	data, err := os.ReadFile(statePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read state file: %w", err)
	}

	var state model.MergeState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to unmarshal state: %w", err)
	}

	return &state, nil
}

// ClearMergeState removes the merge state file
func ClearMergeState() error {
	statePath := filepath.Join(stateDir, stateFile)
	err := os.Remove(statePath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove state file: %w", err)
	}
	return nil
}

// IsMergeInProgress checks if there's a merge in progress
func IsMergeInProgress() bool {
	state, err := LoadMergeState()
	return err == nil && state != nil
}
