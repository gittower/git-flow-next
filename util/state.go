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

// State represents the state of an operation
type State struct {
	Action         string   `json:"action"`
	BranchType     string   `json:"branchType"`
	BranchName     string   `json:"branchName"`
	CurrentStep    string   `json:"currentStep"`
	ParentBranch   string   `json:"parentBranch"`
	MergeStrategy  string   `json:"mergeStrategy"`
	RemainingSteps []string `json:"remainingSteps"`
}

// SaveState saves the state to a file
func SaveState(state *State) error {
	// Create state directory if it doesn't exist
	stateDir := filepath.Join(".git", "gitflow", "state")
	err := os.MkdirAll(stateDir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create state directory: %w", err)
	}

	// Create state file
	stateFile := filepath.Join(stateDir, "state.json")
	file, err := os.Create(stateFile)
	if err != nil {
		return fmt.Errorf("failed to create state file: %w", err)
	}
	defer file.Close()

	// Write state to file
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	err = encoder.Encode(state)
	if err != nil {
		return fmt.Errorf("failed to write state to file: %w", err)
	}

	return nil
}

// LoadState loads the state from a file
func LoadState() (*State, error) {
	// Check if state file exists
	stateFile := filepath.Join(".git", "gitflow", "state", "state.json")
	if _, err := os.Stat(stateFile); os.IsNotExist(err) {
		return nil, fmt.Errorf("no state file found")
	}

	// Open state file
	file, err := os.Open(stateFile)
	if err != nil {
		return nil, fmt.Errorf("failed to open state file: %w", err)
	}
	defer file.Close()

	// Read state from file
	var state State
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&state)
	if err != nil {
		return nil, fmt.Errorf("failed to read state from file: %w", err)
	}

	return &state, nil
}

// ClearState clears the state
func ClearState() error {
	// Check if state file exists
	stateFile := filepath.Join(".git", "gitflow", "state", "state.json")
	if _, err := os.Stat(stateFile); os.IsNotExist(err) {
		return nil
	}

	// Remove state file
	err := os.Remove(stateFile)
	if err != nil {
		return fmt.Errorf("failed to remove state file: %w", err)
	}

	return nil
}

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
