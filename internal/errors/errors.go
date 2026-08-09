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

// InvalidInputError indicates invalid user input with a custom message. It is a
// typed error so callers get ExitCodeInvalidInput (2) instead of the default
// git-error code.
type InvalidInputError struct {
	Message string
}

func (e *InvalidInputError) Error() string {
	return e.Message
}

func (e *InvalidInputError) ExitCode() ExitCode {
	return ExitCodeInvalidInput
}

// SharedConfigDriftError indicates that local git-flow config has drifted from
// the committed .gitflow file. It carries the drift exit code (validation error)
// so `config status` can signal drift distinctly from "not initialized".
type SharedConfigDriftError struct{}

func (e *SharedConfigDriftError) Error() string {
	return "local git-flow configuration is out of sync with .gitflow (run 'git flow config sync')"
}

func (e *SharedConfigDriftError) ExitCode() ExitCode {
	return ExitCodeValidationError
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

// MergeInProgressError represents an error when a git-flow operation is already
// in progress. It is owner-aware: Action names the owning command ("finish",
// "update", "integrate") and BranchType (when the owner is a topic type) lets
// the message print the exact resume/abort commands for that owner.
type MergeInProgressError struct {
	Action     string // owning command: "finish", "update", or "integrate"
	BranchName string // full branch name the operation is running on
	BranchType string // topic branch type when applicable; empty for base/top-level
}

// recoveryCommand returns the base git-flow command that owns the in-progress
// operation (without the --continue/--abort suffix), tailored to the owner:
//   - finish:    git flow <type> finish
//   - integrate: git flow integrate
//   - update:    git flow <type> update (topic) or git flow update (base/top-level)
func (e *MergeInProgressError) recoveryCommand() string {
	switch e.Action {
	case "finish":
		if e.BranchType != "" {
			return fmt.Sprintf("git flow %s finish", e.BranchType)
		}
		return "git flow finish"
	case "integrate":
		return "git flow integrate"
	case "update":
		if e.BranchType != "" {
			return fmt.Sprintf("git flow %s update", e.BranchType)
		}
		return "git flow update"
	default:
		return "git flow"
	}
}

func (e *MergeInProgressError) Error() string {
	base := e.recoveryCommand()
	return fmt.Sprintf(`a %s operation is already in progress for '%s'.
It must be resolved before running another git-flow operation.
  To resume: %s --continue
  To abort:  %s --abort`,
		e.Action, e.BranchName, base, base)
}

func (e *MergeInProgressError) ExitCode() ExitCode {
	return ExitCodeGitError
}

// NoMergeInProgressError represents an error when no merge is in progress
type NoMergeInProgressError struct{}

func (e *NoMergeInProgressError) Error() string {
	return "no merge in progress. Nothing to continue or abort"
}

func (e *NoMergeInProgressError) ExitCode() ExitCode {
	return ExitCodeGitError
}

// UnrecognizedOperationError represents an in-progress git-flow state whose
// Action is empty or unknown. It is never auto-cleared; the user must resolve it
// manually or remove the state file.
type UnrecognizedOperationError struct {
	BranchName string
}

func (e *UnrecognizedOperationError) Error() string {
	// BranchName is best-effort: the loadErr/unparseable path cannot recover a
	// name and passes "". Omit the "for '<name>'" clause entirely in that case
	// rather than printing an unhelpful "for ''".
	if e.BranchName == "" {
		return "an unrecognized git-flow operation is in progress; resolve it manually or remove the state file (.git/gitflow/state/merge.json)"
	}
	return fmt.Sprintf("an unrecognized git-flow operation is in progress for '%s'; resolve it manually or remove the state file (.git/gitflow/state/merge.json)", e.BranchName)
}

func (e *UnrecognizedOperationError) ExitCode() ExitCode {
	return ExitCodeGitError
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

func (e *UnresolvedConflictsError) ExitCode() ExitCode {
	return ExitCodeGitError
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

// Sync status values for BranchNotInSyncError. These mirror the git.BranchSyncStatus string
// values but are duplicated here to avoid an import cycle (internal/git imports would be circular).
const (
	SyncStatusAhead    = "ahead"
	SyncStatusBehind   = "behind"
	SyncStatusDiverged = "diverged"
)

// BranchNotInSyncError indicates the local topic branch is not in sync with its remote tracking
// branch. Behind and diverged risk losing remote work, so they abort unless --force is given; the
// current callers (finish, delete) tolerate ahead, so in practice only behind/diverged reach this
// error, but the ahead message is retained for any caller that does not set tolerateAhead. The
// message is tailored to the specific Status and Operation.
type BranchNotInSyncError struct {
	BranchName   string
	ShortName    string // topic name without the branch-type prefix, used to build the suggested command
	RemoteBranch string
	Status       string // "ahead", "behind", or "diverged"
	CommitCount  int
	BranchType   string
	// Operation names the command that tripped the sync gate ("finish" or "delete"). It tailors
	// the action verb and the suggested `git flow <type> <op> --force` command to the caller.
	// Empty means "finish".
	Operation string
}

func (e *BranchNotInSyncError) Error() string {
	// Use the caller-supplied short name for the suggested command. Deriving it here from the
	// full branch name would mishandle nested topic names (e.g. feature/ui/login). Fall back to
	// the full name only if the caller left it unset.
	shortName := e.ShortName
	if shortName == "" {
		shortName = e.BranchName
	}

	// delete reuses this error via the shared preflight, but its consequences differ from
	// finish's: deleting a local branch never rewrites the remote, so the wording and the
	// suggested command are tailored here. delete tolerates an ahead branch, so it only ever
	// reaches the behind/diverged cases.
	if e.Operation == "delete" {
		if e.Status == SyncStatusDiverged {
			return fmt.Sprintf(`local branch '%s' has diverged from '%s' by %d commit(s).

The local and remote branches each have commits the other does not.
Deleting now would drop the local-only commits.

To resolve:
  git pull                       # reconcile with the remote first

To delete anyway (dropping local-only commits):
  git flow %s delete --force %s`,
				e.BranchName, e.RemoteBranch, e.CommitCount, e.BranchType, shortName)
		}
		return fmt.Sprintf(`local branch '%s' is behind '%s' by %d commit(s).

The remote branch has commits not present locally. Deleting now would
drop the local branch before those commits are integrated.

To resolve:
  git pull                       # bring in the remote commits first

To delete anyway:
  git flow %s delete --force %s`,
			e.BranchName, e.RemoteBranch, e.CommitCount, e.BranchType, shortName)
	}

	switch e.Status {
	case SyncStatusAhead:
		return fmt.Sprintf(`local branch '%s' is ahead of '%s' by %d commit(s).

The local branch has commits that have not been published to the remote.
Finishing now would complete the merge without pushing those commits.

To resolve:
  git push                       # publish your local commits first

To finish anyway (without pushing):
  git flow %s finish --force %s`,
			e.BranchName, e.RemoteBranch, e.CommitCount,
			e.BranchType, shortName)
	case SyncStatusDiverged:
		return fmt.Sprintf(`local branch '%s' has diverged from '%s' by %d commit(s).

The local and remote branches each have commits the other does not.
Finishing now would discard the remote-only commits.

To resolve:
  git pull                       # merge/rebase the remote changes first

To finish anyway (discarding remote changes):
  git flow %s finish --force %s`,
			e.BranchName, e.RemoteBranch, e.CommitCount,
			e.BranchType, shortName)
	default: // SyncStatusBehind
		return fmt.Sprintf(`local branch '%s' is behind '%s' by %d commit(s).

The remote branch has commits not present locally. Finishing now
would discard those changes.

To resolve:
  git pull                       # merge/rebase the remote changes first

To finish anyway (discarding remote changes):
  git flow %s finish --force %s`,
			e.BranchName, e.RemoteBranch, e.CommitCount,
			e.BranchType, shortName)
	}
}

func (e *BranchNotInSyncError) ExitCode() ExitCode {
	return ExitCodeValidationError
}

// FetchFailedError indicates a targeted fetch of a branch failed with a transport/auth error
// (as opposed to a benign missing remote ref). It is fatal so the user does not unknowingly
// finish against stale data; the message names the cause and offers escape hatches.
type FetchFailedError struct {
	Remote string
	Branch string
	Detail string
}

func (e *FetchFailedError) Error() string {
	return fmt.Sprintf(`failed to fetch '%s' from remote '%s': %s

The remote could not be reached, so the sync check cannot run against
fresh data. To proceed anyway:
  --no-fetch    skip the fetch and use local tracking data
  --force       ignore fetch failures and skip the sync check`,
		e.Branch, e.Remote, e.Detail)
}

func (e *FetchFailedError) ExitCode() ExitCode {
	return ExitCodeGitError
}

// BaseBranchNotInSyncError indicates the parent (merge-target) branch is behind or diverged from
// its remote, so finishing would merge onto a stale base. Unlike BranchNotInSyncError (which is
// about the local *topic* branch and tells the user to push/pull the topic), the remedy here is to
// update the parent branch itself. Ahead is acceptable for a parent and never reaches this error.
// Only behind and diverged are represented.
type BaseBranchNotInSyncError struct {
	BranchName   string
	RemoteBranch string
	Status       string // "behind" or "diverged"
	CommitCount  int
}

func (e *BaseBranchNotInSyncError) Error() string {
	if e.Status == SyncStatusDiverged {
		return fmt.Sprintf(`branch '%s' has diverged from '%s' by %d commit(s).

The base branch and its remote each have commits the other does not.
Finishing now would merge onto a stale base.

Reconcile it (e.g. git checkout %s && git pull) or pass --force to finish anyway.`,
			e.BranchName, e.RemoteBranch, e.CommitCount, e.BranchName)
	}
	return fmt.Sprintf(`branch '%s' is %d commit(s) behind '%s'.

The base branch has commits on its remote that are not present locally.
Finishing now would merge onto a stale base.

Update it (e.g. git checkout %s && git pull) or pass --force to finish anyway.`,
		e.BranchName, e.CommitCount, e.RemoteBranch, e.BranchName)
}

func (e *BaseBranchNotInSyncError) ExitCode() ExitCode {
	return ExitCodeValidationError
}

// RemoteNotConfiguredError indicates a required remote is not configured
type RemoteNotConfiguredError struct {
	Remote    string
	Operation string
}

func (e *RemoteNotConfiguredError) Error() string {
	return fmt.Sprintf("No remote '%s' configured. Cannot %s.", e.Remote, e.Operation)
}

func (e *RemoteNotConfiguredError) ExitCode() ExitCode {
	return ExitCodeGitError
}

// NotBaseBranchError indicates that integrate was invoked on a branch that is
// not a base branch (integrate only applies to base branches).
type NotBaseBranchError struct {
	BranchName string
}

func (e *NotBaseBranchError) Error() string {
	return fmt.Sprintf("branch '%s' is not a base branch; 'git flow integrate' applies only to base branches (use 'git flow finish' for topic branches)", e.BranchName)
}

func (e *NotBaseBranchError) ExitCode() ExitCode {
	return ExitCodeValidationError
}

// NoParentBranchError indicates a base branch has no configured parent to
// integrate into.
type NoParentBranchError struct {
	BranchName string
}

func (e *NoParentBranchError) Error() string {
	return fmt.Sprintf("branch '%s' has no configured parent branch to integrate into", e.BranchName)
}

func (e *NoParentBranchError) ExitCode() ExitCode {
	return ExitCodeValidationError
}

// SelfParentError indicates a base branch is configured as its own parent.
type SelfParentError struct {
	BranchName string
}

func (e *SelfParentError) Error() string {
	return fmt.Sprintf("cannot integrate branch '%s' into itself", e.BranchName)
}

func (e *SelfParentError) ExitCode() ExitCode {
	return ExitCodeValidationError
}

// NoUpstreamStrategyError indicates a base branch has no upstream merge strategy
// configured, so there is nothing to integrate.
type NoUpstreamStrategyError struct {
	BranchName string
}

func (e *NoUpstreamStrategyError) Error() string {
	return fmt.Sprintf("branch '%s' has no upstream merge strategy configured to integrate with", e.BranchName)
}

func (e *NoUpstreamStrategyError) ExitCode() ExitCode {
	return ExitCodeValidationError
}

// TagEnabledNoNameError indicates tagging was enabled but no tag name could be
// resolved (base branches have no version-derived default name).
type TagEnabledNoNameError struct{}

func (e *TagEnabledNoNameError) Error() string {
	return "tagging is enabled but no tag name was provided; supply a name with --tag <name> or configure integrate.tagname"
}

func (e *TagEnabledNoNameError) ExitCode() ExitCode {
	return ExitCodeInvalidInput
}

// AlreadyInitializedError indicates git-flow is already configured
type AlreadyInitializedError struct{}

func (e *AlreadyInitializedError) Error() string {
	return "git-flow is already initialized in this repository. Use --force to reconfigure"
}

func (e *AlreadyInitializedError) ExitCode() ExitCode {
	return ExitCodeValidationError
}

// RepositoryCreationDeclinedError indicates the user answered no to the prompt
// offering to create a git repository. Nothing failed here — the user declined —
// so the message is printed verbatim instead of being wrapped in GitError's
// "failed to <operation>" phrasing. The exit code matches the non-interactive
// "not a git repository" path so callers see one code for "no repository to
// work in", however that conclusion was reached.
type RepositoryCreationDeclinedError struct{}

func (e *RepositoryCreationDeclinedError) Error() string {
	return "no git repository. Run 'git init' first, or re-run with --init"
}

func (e *RepositoryCreationDeclinedError) ExitCode() ExitCode {
	return ExitCodeGitError
}

// MissingUserIdentityError indicates git user.name/user.email are not configured
type MissingUserIdentityError struct{}

func (e *MissingUserIdentityError) Error() string {
	return "git user identity is not configured. Set it before running git flow init:\n" +
		"  git config --global user.name \"Your Name\"\n" +
		"  git config --global user.email \"you@example.com\""
}

func (e *MissingUserIdentityError) ExitCode() ExitCode {
	return ExitCodeValidationError
}
