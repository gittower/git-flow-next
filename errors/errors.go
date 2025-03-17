package errors

import "fmt"

// ExitCode represents the process exit code
type ExitCode int

const (
	// ExitCodeSuccess indicates successful execution
	ExitCodeSuccess ExitCode = 0
	// ExitCodeNotInitialized indicates git-flow is not initialized
	ExitCodeNotInitialized ExitCode = 1
	// ExitCodeInvalidInput indicates invalid user input
	ExitCodeInvalidInput ExitCode = 2
	// ExitCodeGitError indicates a Git operation failed
	ExitCodeGitError ExitCode = 3
	// ExitCodeBranchExists indicates a branch already exists
	ExitCodeBranchExists ExitCode = 4
	// ExitCodeBranchNotFound indicates a required branch does not exist
	ExitCodeBranchNotFound ExitCode = 5
)

// Error is the base interface for all git-flow errors
type Error interface {
	error
	ExitCode() ExitCode
}

// ExitCoder is an interface for errors that can provide an exit code
type ExitCoder interface {
	ExitCode() uint8
}

// NotInitializedError indicates that git-flow is not initialized
type NotInitializedError struct{}

func (e *NotInitializedError) Error() string {
	return "git flow is not initialized (run 'git flow init' first)"
}

func (e *NotInitializedError) ExitCode() ExitCode {
	return ExitCodeNotInitialized
}

// EmptyBranchNameError indicates that a branch name was not provided
type EmptyBranchNameError struct{}

func (e *EmptyBranchNameError) Error() string {
	return "branch name cannot be empty"
}

func (e *EmptyBranchNameError) ExitCode() ExitCode {
	return ExitCodeInvalidInput
}

// InvalidBranchTypeError indicates an unknown branch type
type InvalidBranchTypeError struct {
	BranchType string
}

func (e *InvalidBranchTypeError) Error() string {
	return fmt.Sprintf("unknown branch type: %s", e.BranchType)
}

func (e *InvalidBranchTypeError) ExitCode() ExitCode {
	return ExitCodeInvalidInput
}

// BranchExistsError indicates a branch already exists
type BranchExistsError struct {
	BranchName string
}

func (e *BranchExistsError) Error() string {
	return fmt.Sprintf("branch '%s' already exists", e.BranchName)
}

func (e *BranchExistsError) ExitCode() ExitCode {
	return ExitCodeBranchExists
}

// BranchNotFoundError indicates a required branch does not exist
type BranchNotFoundError struct {
	BranchName string
}

func (e *BranchNotFoundError) Error() string {
	return fmt.Sprintf("start point branch '%s' does not exist", e.BranchName)
}

func (e *BranchNotFoundError) ExitCode() ExitCode {
	return ExitCodeBranchNotFound
}

// GitError indicates a Git operation failed
type GitError struct {
	Operation string
	Err       error
}

func (e *GitError) Error() string {
	return fmt.Sprintf("failed to %s: %v", e.Operation, e.Err)
}

func (e *GitError) ExitCode() ExitCode {
	return ExitCodeGitError
}

func (e *GitError) Unwrap() error {
	return e.Err
}

// MergeInProgressError represents an error when a merge is already in progress
type MergeInProgressError struct {
	BranchName string
}

func (e *MergeInProgressError) Error() string {
	return fmt.Sprintf("a merge is already in progress for branch '%s'. Use --continue or --abort", e.BranchName)
}

func (e *MergeInProgressError) ExitCode() uint8 {
	return 1
}

// NoMergeInProgressError represents an error when no merge is in progress
type NoMergeInProgressError struct{}

func (e *NoMergeInProgressError) Error() string {
	return "no merge in progress. Nothing to continue or abort"
}

func (e *NoMergeInProgressError) ExitCode() uint8 {
	return 1
}

// InvalidBranchNameError represents an error when an invalid branch name is provided
type InvalidBranchNameError struct {
	Name string
}

func (e *InvalidBranchNameError) Error() string {
	return fmt.Sprintf("invalid branch name: %s", e.Name)
}

func (e *InvalidBranchNameError) ExitCode() uint8 {
	return 1
}

// UnresolvedConflictsError represents an error when there are unresolved conflicts
type UnresolvedConflictsError struct{}

func (e *UnresolvedConflictsError) Error() string {
	return "there are still unresolved conflicts. Resolve them and try again"
}

func (e *UnresolvedConflictsError) ExitCode() uint8 {
	return 1
}
