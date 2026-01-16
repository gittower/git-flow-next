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
	// ExitCodeValidationError indicates a validation error
	ExitCodeValidationError ExitCode = 6
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

// LocalBranchNotFoundError indicates a local branch does not exist
type LocalBranchNotFoundError struct {
	BranchName string
}

func (e *LocalBranchNotFoundError) Error() string {
	return fmt.Sprintf("local branch '%s' does not exist", e.BranchName)
}

func (e *LocalBranchNotFoundError) ExitCode() ExitCode {
	return ExitCodeBranchNotFound
}

// RemoteBranchExistsError indicates a branch already exists on the remote
type RemoteBranchExistsError struct {
	Remote     string
	BranchName string
}

func (e *RemoteBranchExistsError) Error() string {
	return fmt.Sprintf("branch '%s' already exists on remote '%s'", e.BranchName, e.Remote)
}

func (e *RemoteBranchExistsError) ExitCode() ExitCode {
	return ExitCodeBranchExists
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
	BranchName string
}

func (e *InvalidBranchNameError) Error() string {
	return fmt.Sprintf("invalid branch name: %s", e.BranchName)
}

func (e *InvalidBranchNameError) ExitCode() ExitCode {
	return ExitCodeInvalidInput
}

// InvalidMergeStrategyError indicates an invalid merge strategy
type InvalidMergeStrategyError struct {
	Strategy string
}

func (e *InvalidMergeStrategyError) Error() string {
	return fmt.Sprintf("invalid merge strategy: %s (valid options: merge, rebase, squash)", e.Strategy)
}

func (e *InvalidMergeStrategyError) ExitCode() ExitCode {
	return ExitCodeInvalidInput
}

// CircularDependencyError indicates a circular dependency in branch configuration
type CircularDependencyError struct {
	BranchName string
}

func (e *CircularDependencyError) Error() string {
	return fmt.Sprintf("circular dependency detected for branch '%s'", e.BranchName)
}

func (e *CircularDependencyError) ExitCode() ExitCode {
	return ExitCodeValidationError
}

// BranchHasDependentsError indicates a branch cannot be deleted because it has dependents
type BranchHasDependentsError struct {
	BranchName string
	Dependent  string
}

func (e *BranchHasDependentsError) Error() string {
	return fmt.Sprintf("cannot delete branch '%s': branch '%s' depends on it", e.BranchName, e.Dependent)
}

func (e *BranchHasDependentsError) ExitCode() ExitCode {
	return ExitCodeValidationError
}

// UnresolvedConflictsError represents an error when there are unresolved conflicts
type UnresolvedConflictsError struct{}

func (e *UnresolvedConflictsError) Error() string {
	return "there are still unresolved conflicts. Resolve them and try again"
}

func (e *UnresolvedConflictsError) ExitCode() uint8 {
	return 1
}

// RemoteBranchNotFoundError indicates the branch doesn't exist on the remote
type RemoteBranchNotFoundError struct {
	Remote     string
	BranchName string
}

func (e *RemoteBranchNotFoundError) Error() string {
	return fmt.Sprintf("branch '%s' not found on remote '%s'", e.BranchName, e.Remote)
}

func (e *RemoteBranchNotFoundError) ExitCode() ExitCode {
	return ExitCodeBranchNotFound
}

// BranchBehindRemoteError indicates the local branch is behind its remote tracking branch
type BranchBehindRemoteError struct {
	BranchName    string
	RemoteName    string
	CommitsBehind int
}

func (e *BranchBehindRemoteError) Error() string {
	return fmt.Sprintf("branch '%s' is %d commit(s) behind '%s/%s'. Pull the latest changes or use --no-remote-check to proceed anyway",
		e.BranchName, e.CommitsBehind, e.RemoteName, e.BranchName)
}

func (e *BranchBehindRemoteError) ExitCode() ExitCode {
	return ExitCodeInvalidInput
}
