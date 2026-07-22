package git

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// ErrRemoteRefNotFound is a sentinel error returned by FetchBranch when git reports that the
// requested branch does not exist on the remote (e.g. never pushed, or deleted after a remote
// merge). This is benign: callers treat it as "no remote ref to compare against" rather than a
// transport failure.
var ErrRemoteRefNotFound = errors.New("remote ref not found")

// BranchSyncStatus represents the sync status between a local branch and its remote tracking branch
type BranchSyncStatus string

const (
	// SyncStatusEqual indicates the local and remote branches are at the same commit
	SyncStatusEqual BranchSyncStatus = "equal"
	// SyncStatusAhead indicates the local branch has commits not on the remote
	SyncStatusAhead BranchSyncStatus = "ahead"
	// SyncStatusBehind indicates the remote branch has commits not on the local branch
	SyncStatusBehind BranchSyncStatus = "behind"
	// SyncStatusDiverged indicates both branches have commits the other doesn't have
	SyncStatusDiverged BranchSyncStatus = "diverged"
	// SyncStatusNoTracking indicates the branch has no remote tracking branch configured
	SyncStatusNoTracking BranchSyncStatus = "no_tracking"
)

// IsGitRepo checks if the current directory is a Git repository
func IsGitRepo() bool {
	cmd := exec.Command("git", "rev-parse", "--is-inside-work-tree")
	err := cmd.Run()
	return err == nil
}

// GetGitDir returns the path to the git directory for the current repository.
// For regular repositories, this returns ".git".
// For worktrees, this returns the actual git directory path (e.g., "/repo/.git/worktrees/work1").
func GetGitDir() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get git directory: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

// GetCurrentBranch returns the current Git branch
func GetCurrentBranch() (string, error) {
	// Check if we have any commits
	hasCommits, err := HasCommits()
	if err != nil {
		return "", fmt.Errorf("failed to check if repository has commits: %w", err)
	}

	if !hasCommits {
		// If no commits, there's no current branch
		return "", nil
	}

	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get current branch: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

// BranchExists checks if a branch exists
func BranchExists(branch string) error {
	cmd := exec.Command("git", "rev-parse", "--verify", "--quiet", "refs/heads/"+branch)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("branch '%s' does not exist", branch)
	}
	return nil
}

// BranchOrCommitExists checks if a branch, tag, or commit exists
func BranchOrCommitExists(ref string) error {
	cmd := exec.Command("git", "rev-parse", "--verify", "--quiet", ref)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("reference '%s' does not exist", ref)
	}
	return nil
}

// CreateBranch creates a new branch
func CreateBranch(name string, startPoint string) error {
	// Check if we have any commits
	hasCommits, err := HasCommits()
	if err != nil {
		return fmt.Errorf("failed to check if repository has commits: %w", err)
	}

	if !hasCommits {
		// If no commits, create an initial commit first
		err = CreateInitialCommit(name)
		if err != nil {
			return fmt.Errorf("failed to create initial commit: %w", err)
		}
		return nil
	}

	// If startPoint is empty, use the current branch
	if startPoint == "" {
		currentBranch, err := GetCurrentBranch()
		if err != nil {
			return fmt.Errorf("failed to get current branch: %w", err)
		}
		startPoint = currentBranch
	}

	cmd := exec.Command("git", "checkout", "-b", name, startPoint)
	_, err = cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to create branch: %w", err)
	}
	return nil
}

// Checkout checks out a branch
func Checkout(branch string) error {
	cmd := exec.Command("git", "checkout", branch)
	_, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to checkout branch: %w", err)
	}
	return nil
}

// DeleteBranch deletes a branch
func DeleteBranch(branch string, force bool) error {
	flag := "-d"
	if force {
		flag = "-D"
	}

	cmd := exec.Command("git", "branch", flag, branch)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to delete branch: %s", string(output))
	}
	return nil
}

// HasCommits checks if the repository has any commits
func HasCommits() (bool, error) {
	cmd := exec.Command("git", "rev-parse", "--verify", "HEAD")
	err := cmd.Run()
	if err != nil {
		// If error, there are no commits
		return false, nil
	}
	return true, nil
}

// CreateInitialCommit creates an initial commit and branch
func CreateInitialCommit(branch string) error {
	// Create an empty initial commit
	cmd := exec.Command("git", "commit", "--allow-empty", "-m", "Initial commit")
	_, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to create initial commit: %w", err)
	}

	// Rename the default branch to the target name
	cmd = exec.Command("git", "branch", "-m", branch)
	_, err = cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to rename branch to %s: %w", branch, err)
	}

	return nil
}

// Merge merges a branch into the current branch
func Merge(branch string, noVerify bool) error {
	args := []string{"merge", "--no-ff"}
	if noVerify {
		args = append(args, "--no-verify")
	}
	args = append(args, branch)

	cmd := exec.Command("git", args...)
	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	// Check for merge conflicts - Git returns exit code 1 and specific output patterns
	if err != nil {
		// Check if there are unmerged paths (conflicts)
		conflictCmd := exec.Command("git", "ls-files", "--unmerged")
		conflictOutput, _ := conflictCmd.Output()

		if len(conflictOutput) > 0 ||
			strings.Contains(outputStr, "Automatic merge failed") ||
			strings.Contains(outputStr, "CONFLICT") ||
			strings.Contains(outputStr, "merge failed") ||
			strings.Contains(outputStr, "needs merge") {
			return fmt.Errorf("merge conflict: %s", outputStr)
		}
		return fmt.Errorf("failed to merge branch: %s", outputStr)
	}

	return nil
}

// Rebase rebases the current branch onto another branch
func Rebase(branch string) error {
	cmd := exec.Command("git", "rebase", branch)
	output, err := cmd.CombinedOutput()
	if err != nil {
		if strings.Contains(string(output), "conflict") {
			return fmt.Errorf("rebase conflict: %s", string(output))
		}
		return fmt.Errorf("failed to rebase branch: %s", string(output))
	}
	return nil
}

// SquashMerge performs a squash merge of a branch into the current branch
func SquashMerge(branch string, noVerify bool) error {
	args := []string{"merge", "--squash"}
	if noVerify {
		args = append(args, "--no-verify")
	}
	args = append(args, branch)

	cmd := exec.Command("git", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		if strings.Contains(string(output), "conflict") {
			return fmt.Errorf("squash merge conflict: %s", string(output))
		}
		return fmt.Errorf("failed to squash merge branch: %s", string(output))
	}

	// Commit the squashed changes
	commitArgs := []string{"commit", "-m", fmt.Sprintf("Squashed commit of branch '%s'", branch)}
	if noVerify {
		commitArgs = append(commitArgs, "--no-verify")
	}
	cmd = exec.Command("git", commitArgs...)
	output, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to commit squashed changes: %s", string(output))
	}

	return nil
}

// ListBranches returns a list of all branches in the repository
func ListBranches() ([]string, error) {
	cmd := exec.Command("git", "branch", "--format=%(refname:short)")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list branches: %w", err)
	}

	// Split the output by newlines and remove empty lines
	branches := []string{}
	for _, branch := range strings.Split(string(output), "\n") {
		if branch != "" {
			branches = append(branches, strings.TrimSpace(branch))
		}
	}

	return branches, nil
}

// HasConflicts checks if there are unresolved conflicts
func HasConflicts() bool {
	// Check for unmerged paths
	cmd := exec.Command("git", "diff", "--name-only", "--diff-filter=U")
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return len(output) > 0
}

// IsGitMergeInProgress checks if git is in a merge state by looking for MERGE_HEAD
func IsGitMergeInProgress() bool {
	gitDir, err := GetGitDir()
	if err != nil {
		return false
	}
	_, err = os.Stat(filepath.Join(gitDir, "MERGE_HEAD"))
	return err == nil
}

// IsGitRebaseInProgress checks if git is in a rebase state by looking for
// rebase-merge/ (merge backend, including interactive) or rebase-apply/ (legacy apply backend)
func IsGitRebaseInProgress() bool {
	gitDir, err := GetGitDir()
	if err != nil {
		return false
	}
	if _, err := os.Stat(filepath.Join(gitDir, "rebase-merge")); err == nil {
		return true
	}
	if _, err := os.Stat(filepath.Join(gitDir, "rebase-apply")); err == nil {
		return true
	}
	return false
}

// IsGitSquashMergeInProgress checks if git is in a squash merge state.
// Squash merges create SQUASH_MSG but not MERGE_HEAD. However, SQUASH_MSG also
// appears during interactive rebase squash steps, so we exclude that case.
func IsGitSquashMergeInProgress() bool {
	gitDir, err := GetGitDir()
	if err != nil {
		return false
	}
	if _, err := os.Stat(filepath.Join(gitDir, "SQUASH_MSG")); err != nil {
		return false
	}
	// Exclude interactive rebase squash steps
	if _, err := os.Stat(filepath.Join(gitDir, "rebase-merge")); err == nil {
		return false
	}
	return true
}

// MergeAbort aborts the current merge
func MergeAbort() error {
	cmd := exec.Command("git", "merge", "--abort")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to abort merge: %w", err)
	}
	return nil
}

// RebaseAbort aborts the current rebase
func RebaseAbort() error {
	cmd := exec.Command("git", "rebase", "--abort")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to abort rebase: %s", string(output))
	}
	return nil
}

// RenameBranch renames a branch. If oldBranch is provided, it renames that branch to newBranch.
// If oldBranch is not provided, it renames the current branch to newBranch.
func RenameBranch(oldBranch, newBranch string) error {
	return renameBranch(oldBranch, newBranch, false)
}

// RenameBranchForce renames a Git branch using the force flag (-M). This is
// required for a case-only rename on case-insensitive filesystems, where the
// destination name folds to the same existing ref and the non-forcing -m
// refuses with "a branch named '…' already exists". Callers must confirm the
// rename does not clobber a genuinely different branch before forcing.
func RenameBranchForce(oldBranch, newBranch string) error {
	return renameBranch(oldBranch, newBranch, true)
}

func renameBranch(oldBranch, newBranch string, force bool) error {
	flag := "-m"
	if force {
		flag = "-M"
	}
	args := []string{"branch", flag, oldBranch, newBranch}

	cmd := exec.Command("git", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to rename branch: %s", strings.TrimSpace(string(output)))
	}
	return nil
}

// Fetch performs a git fetch from the specified remote
func Fetch(remote string) error {
	cmd := exec.Command("git", "fetch", remote)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to fetch from remote '%s': %s", remote, string(output))
	}
	return nil
}

// DeleteRemoteBranch deletes a branch from a remote repository
func DeleteRemoteBranch(remote, branch string) error {
	cmd := exec.Command("git", "push", remote, ":"+branch)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to delete remote branch: %s", string(output))
	}
	return nil
}

// RemoteExists checks if a remote is configured in the local git config.
// This is a local-only check (no network call).
func RemoteExists(remote string) bool {
	cmd := exec.Command("git", "remote", "get-url", remote)
	return cmd.Run() == nil
}

// RemoteBranchExists checks if a remote branch exists
func RemoteBranchExists(remote, branch string) bool {
	// Check if the remote tracking branch exists
	ref := fmt.Sprintf("refs/remotes/%s/%s", remote, branch)
	cmd := exec.Command("git", "rev-parse", "--verify", "--quiet", ref)
	return cmd.Run() == nil
}

// TagOptions contains options for tag creation
type TagOptions struct {
	Message     string // Tag message (required for annotated tags)
	MessageFile string // File containing the message (optional, overrides Message)
	Sign        bool   // Whether to sign the tag (optional)
	SigningKey  string // Key to use for signing (optional, implies Sign=true)
}

// CreateTag creates a Git tag with the specified options
func CreateTag(tagName string, options *TagOptions) error {
	// Check if tag already exists
	cmd := exec.Command("git", "show-ref", "--tags", tagName)
	if err := cmd.Run(); err == nil {
		// Tag already exists, skip creation
		return nil
	}

	// Build command arguments
	args := []string{"tag"}

	// Use annotated tag
	args = append(args, "-a")

	// Apply signing if requested
	shouldSign := options.Sign || options.SigningKey != ""
	if shouldSign {
		args = append(args, "-s")

		// Apply signing key if specified
		if options.SigningKey != "" {
			args = append(args, "-u", options.SigningKey)
		}
	}

	// Apply tag name
	args = append(args, tagName)

	// Apply message
	if options.MessageFile != "" {
		args = append(args, "-F", options.MessageFile)
	} else if options.Message != "" {
		args = append(args, "-m", options.Message)
	} else {
		return fmt.Errorf("tag message is required for annotated tags")
	}

	// Execute tag command
	cmd = exec.Command("git", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to create tag '%s': %w (output: %s)", tagName, err, string(output))
	}

	return nil
}

// RebaseWithOptions rebases the current branch onto another branch with optional preserve-merges
func RebaseWithOptions(targetBranch string, preserveMerges bool) error {
	args := []string{"rebase"}
	if preserveMerges {
		args = append(args, "--preserve-merges")
	}
	args = append(args, targetBranch)

	cmd := exec.Command("git", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		if strings.Contains(string(output), "conflict") {
			return fmt.Errorf("rebase conflict: %s", string(output))
		}
		return fmt.Errorf("failed to rebase branch: %s", string(output))
	}
	return nil
}

// MergeWithOptions merges a branch into current branch with optional no-fast-forward
func MergeWithOptions(branchName string, noFF bool, noVerify bool) error {
	args := []string{"merge"}
	if noFF {
		args = append(args, "--no-ff")
	}
	if noVerify {
		args = append(args, "--no-verify")
	}
	args = append(args, branchName)

	cmd := exec.Command("git", args...)
	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	// Check for merge conflicts - Git returns exit code 1 and specific output patterns
	if err != nil {
		// Check if there are unmerged paths (conflicts)
		conflictCmd := exec.Command("git", "ls-files", "--unmerged")
		conflictOutput, _ := conflictCmd.Output()

		if len(conflictOutput) > 0 ||
			strings.Contains(outputStr, "Automatic merge failed") ||
			strings.Contains(outputStr, "CONFLICT") ||
			strings.Contains(outputStr, "merge failed") ||
			strings.Contains(outputStr, "needs merge") {
			return fmt.Errorf("merge conflict: %s", outputStr)
		}
		return fmt.Errorf("failed to merge branch: %s", outputStr)
	}

	return nil
}

// MergeWithMessage merges a branch into current branch with a custom commit message
func MergeWithMessage(branchName string, message string, noFF bool, noVerify bool) error {
	args := []string{"merge"}
	if noFF {
		args = append(args, "--no-ff")
	}
	if noVerify {
		args = append(args, "--no-verify")
	}
	args = append(args, "-m", message, branchName)

	cmd := exec.Command("git", args...)
	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	// Check for merge conflicts - Git returns exit code 1 and specific output patterns
	if err != nil {
		// Check if there are unmerged paths (conflicts)
		conflictCmd := exec.Command("git", "ls-files", "--unmerged")
		conflictOutput, _ := conflictCmd.Output()

		if len(conflictOutput) > 0 ||
			strings.Contains(outputStr, "Automatic merge failed") ||
			strings.Contains(outputStr, "CONFLICT") ||
			strings.Contains(outputStr, "merge failed") ||
			strings.Contains(outputStr, "needs merge") {
			return fmt.Errorf("merge conflict: %s", outputStr)
		}
		return fmt.Errorf("failed to merge branch: %s", outputStr)
	}

	return nil
}

// Commit creates a commit with the given message
func Commit(message string, noVerify bool) error {
	args := []string{"commit", "-m", message}
	if noVerify {
		args = append(args, "--no-verify")
	}
	cmd := exec.Command("git", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to commit: %s", string(output))
	}
	return nil
}

// RebaseContinue continues an ongoing rebase operation after conflicts are resolved
func RebaseContinue() error {
	cmd := exec.Command("git", "rebase", "--continue")
	output, err := cmd.CombinedOutput()
	outputStr := string(output)
	if err != nil {
		if strings.Contains(outputStr, "No rebase in progress") {
			// Not an error - rebase is already complete
			return nil
		}
		if strings.Contains(outputStr, "conflict") || strings.Contains(outputStr, "CONFLICT") {
			return fmt.Errorf("rebase conflict: %s", outputStr)
		}
		return fmt.Errorf("failed to continue rebase: %s", outputStr)
	}
	return nil
}

// MergeSquashWithMessage performs a squash merge with a custom commit message
func MergeSquashWithMessage(branchName string, message string, noVerify bool) error {
	args := []string{"merge", "--squash"}
	if noVerify {
		args = append(args, "--no-verify")
	}
	args = append(args, branchName)

	cmd := exec.Command("git", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		if strings.Contains(string(output), "conflict") {
			return fmt.Errorf("squash merge conflict: %s", string(output))
		}
		return fmt.Errorf("failed to squash merge branch: %s", string(output))
	}

	// Commit the squashed changes with custom message
	commitArgs := []string{"commit", "-m", message}
	if noVerify {
		commitArgs = append(commitArgs, "--no-verify")
	}
	cmd = exec.Command("git", commitArgs...)
	output, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to commit squashed changes: %s", string(output))
	}

	return nil
}

// PushBranch pushes a local branch to a remote and sets up tracking
func PushBranch(remote, branch string, pushOptions []string) error {
	args := []string{"push", "-u", remote}

	for _, opt := range pushOptions {
		args = append(args, "-o", opt)
	}

	args = append(args, branch)

	cmd := exec.Command("git", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to push branch '%s' to '%s': %s", branch, remote, strings.TrimSpace(string(output)))
	}
	return nil
}

// PushRef pushes a branch to a remote with a plain `git push <remote> <branch>`
// (no -u). Finish pushes base branches that already track their remote, so this
// avoids rewriting tracking config. On failure it returns a wrapped error that
// embeds the underlying error and the trimmed combined output, so a rejected
// (non-fast-forward) push surfaces to the caller.
func PushRef(remote, branch string) error {
	cmd := exec.Command("git", "push", remote, branch)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to push branch '%s' to '%s': %w (output: %s)", branch, remote, err, strings.TrimSpace(string(output)))
	}
	return nil
}

// PushTag pushes a single tag to a remote with `git push <remote> tag <tag>`.
// On failure it returns a wrapped error that embeds the underlying error and the
// trimmed combined output.
func PushTag(remote, tag string) error {
	cmd := exec.Command("git", "push", remote, "tag", tag)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to push tag '%s' to '%s': %w (output: %s)", tag, remote, err, strings.TrimSpace(string(output)))
	}
	return nil
}

// CreateTrackingBranch creates a local branch that tracks a remote branch
func CreateTrackingBranch(localBranch, remote, remoteBranch string) error {
	// git checkout -b <local> --track <remote>/<branch>
	cmd := exec.Command("git", "checkout", "-b", localBranch, "--track",
		fmt.Sprintf("%s/%s", remote, remoteBranch))
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to create tracking branch: %s", string(output))
	}
	return nil
}

// GetTrackingBranch returns the remote tracking branch for a local branch.
// Returns the full tracking reference (e.g., "origin/feature/foo") or an error
// if no tracking branch is configured.
func GetTrackingBranch(branch string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", branch+"@{upstream}")
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Check if the error is because there's no tracking branch
		outputStr := string(output)
		if strings.Contains(outputStr, "no upstream") || strings.Contains(outputStr, "does not track") {
			return "", fmt.Errorf("branch '%s' has no upstream tracking branch", branch)
		}
		return "", fmt.Errorf("failed to get tracking branch for '%s': %s", branch, outputStr)
	}
	return strings.TrimSpace(string(output)), nil
}

// CompareBranchWithRemote compares a local branch with its remote tracking branch.
// Returns the sync status and the number of commits different.
// For SyncStatusAhead, the count is commits ahead.
// For SyncStatusBehind, the count is commits behind.
// For SyncStatusDiverged, the count is total commits different (ahead + behind).
func CompareBranchWithRemote(branch string) (BranchSyncStatus, int, error) {
	// First get the tracking branch
	trackingBranch, err := GetTrackingBranch(branch)
	if err != nil {
		return SyncStatusNoTracking, 0, err
	}

	// Use git rev-list to count commits ahead and behind
	// Format: <ahead>\t<behind>
	cmd := exec.Command("git", "rev-list", "--left-right", "--count", branch+"..."+trackingBranch)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", 0, fmt.Errorf("failed to compare branches: %s", string(output))
	}

	// Parse the output (format: "ahead\tbehind")
	parts := strings.Fields(strings.TrimSpace(string(output)))
	if len(parts) != 2 {
		return "", 0, fmt.Errorf("unexpected output format from rev-list: %s", string(output))
	}

	ahead, err := strconv.Atoi(parts[0])
	if err != nil {
		return "", 0, fmt.Errorf("failed to parse ahead count: %w", err)
	}

	behind, err := strconv.Atoi(parts[1])
	if err != nil {
		return "", 0, fmt.Errorf("failed to parse behind count: %w", err)
	}

	// Determine status based on ahead/behind counts
	switch {
	case ahead == 0 && behind == 0:
		return SyncStatusEqual, 0, nil
	case ahead > 0 && behind == 0:
		return SyncStatusAhead, ahead, nil
	case ahead == 0 && behind > 0:
		return SyncStatusBehind, behind, nil
	default:
		return SyncStatusDiverged, ahead + behind, nil
	}
}

// FetchBranch fetches a specific branch from a remote.
// This is a targeted fetch that only updates the specified branch reference.
//
// On failure it classifies the git error:
//   - If stderr indicates the branch does not exist on the remote, it wraps the benign
//     sentinel ErrRemoteRefNotFound (check with errors.Is).
//   - Any other non-zero exit (transport/auth failure) returns a fatal error carrying the
//     trimmed stderr.
func FetchBranch(remote, branch string) error {
	cmd := exec.Command("git", "fetch", remote, branch)
	// Pin the locale so isRemoteRefNotFound classifies stderr against git's English phrasing,
	// regardless of the user's LANG/LC_* settings. A localized "missing ref" message would
	// otherwise be misclassified as a fatal transport failure.
	cmd.Env = append(os.Environ(), "LC_ALL=C")
	output, err := cmd.CombinedOutput()
	if err != nil {
		stderr := strings.TrimSpace(string(output))
		if isRemoteRefNotFound(stderr) {
			// Join the benign sentinel with the underlying exec error so callers can classify via
			// errors.Is(ErrRemoteRefNotFound) while the original error stays available for diagnostics.
			return fmt.Errorf("fetch branch '%s' from '%s': %s: %w", branch, remote, stderr, errors.Join(ErrRemoteRefNotFound, err))
		}
		return fmt.Errorf("failed to fetch branch '%s' from '%s': %s: %w", branch, remote, stderr, err)
	}
	return nil
}

// isRemoteRefNotFound reports whether git stderr indicates the requested ref does not exist on
// the remote (as opposed to a transport/auth failure). Matching is case-insensitive.
func isRemoteRefNotFound(stderr string) bool {
	lowered := strings.ToLower(stderr)
	return strings.Contains(lowered, "couldn't find remote ref") ||
		strings.Contains(lowered, "could not find remote ref") ||
		strings.Contains(lowered, "no such ref") ||
		strings.Contains(lowered, "not our ref")
}

// HasTrackingBranch reports whether the given local branch has a remote tracking branch
// configured. It is a thin wrapper over GetTrackingBranch that returns false on any error.
func HasTrackingBranch(branch string) bool {
	_, err := GetTrackingBranch(branch)
	return err == nil
}

// DeleteRemoteTrackingRef removes a stale remote-tracking ref (refs/remotes/<remote>/<branch>)
// from the local repository. It is used to prune a tracking ref once the corresponding remote
// branch is found to be gone. Errors are returned so callers may log them, but they are safe to
// ignore (the ref may already be absent).
func DeleteRemoteTrackingRef(remote, branch string) error {
	ref := fmt.Sprintf("refs/remotes/%s/%s", remote, branch)
	cmd := exec.Command("git", "update-ref", "-d", ref)
	if output, err := cmd.CombinedOutput(); err != nil {
		// Wrap the underlying exec error with %w so the exit status stays available for diagnostics,
		// mirroring FetchBranch's error wrapping, while still surfacing the trimmed git output.
		return fmt.Errorf("failed to delete remote tracking ref '%s': %s: %w", ref, strings.TrimSpace(string(output)), err)
	}
	return nil
}
