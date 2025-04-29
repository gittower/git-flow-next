package model

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const (
	stateDir  = ".git/gitflow/state"
	stateFile = "merge.json"
)

// MergeState represents the state of a merge operation
type MergeState struct {
	Action          string   `json:"action"`          // "finish"
	BranchType      string   `json:"branchType"`      // feature, release, hotfix, etc.
	BranchName      string   `json:"branchName"`      // name of the branch being merged
	CurrentStep     string   `json:"currentStep"`     // current step in the process (merge, update_children, delete_branch)
	ParentBranch    string   `json:"parentBranch"`    // target branch for the merge
	MergeStrategy   string   `json:"mergeStrategy"`   // merge strategy being used
	FullBranchName  string   `json:"fullBranchName"`  // full name of the branch (with prefix)
	ChildBranches   []string `json:"childBranches"`   // child branches that need to be updated
	UpdatedBranches []string `json:"updatedBranches"` // child branches that have been updated
}

// SaveMergeState saves the current merge state to a file
func SaveMergeState(state *MergeState) error {
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
func LoadMergeState() (*MergeState, error) {
	statePath := filepath.Join(stateDir, stateFile)
	data, err := os.ReadFile(statePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read state file: %w", err)
	}

	var state MergeState
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
