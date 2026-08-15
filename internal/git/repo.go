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

// gitCommand is the single command factory for the git package: every
// repo-bound git operation runs through it, so it pins cmd.Dir to a
// deterministic working directory instead of inheriting the process CWD.
// (The repository-less refname check in internal/util calls git directly, as
// it needs no work tree.) Pass dir="" for the rare repository-less invocations
// (explicit --global/--system/--file config scopes) that must not bind to a
// work tree.
func gitCommand(dir string, args ...string) *exec.Cmd {
	cmd := exec.Command("git", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	return cmd
}

// Repo is a handle to a specific git repository resolved to absolute paths. All
// repo-bound operations are methods on *Repo and run with the work tree as their
// working directory, so they never depend on the process CWD.
type Repo struct {
	workTree     string // absolute path to the work-tree root
	gitDir       string // absolute path to this working tree's git dir
	commonGitDir string // absolute path to the shared (common) git dir
}

// WorkTree returns the absolute path to the repository's work-tree root.
func (r *Repo) WorkTree() string { return r.workTree }

// GitDir returns the absolute path to this working tree's git directory. For a
// linked worktree this is the per-worktree dir (<common>/worktrees/<name>).
func (r *Repo) GitDir() string { return r.gitDir }

// CommonGitDir returns the absolute path to the repository's common git
// directory (the main .git), shared across linked worktrees.
func (r *Repo) CommonGitDir() string { return r.commonGitDir }

// MainWorkTree returns the absolute path to the repository's MAIN work-tree
// root, which is not necessarily this handle's.
//
// WorkTree() reports the worktree the command was invoked from — a linked
// worktree when the user is standing in one. Several decisions must anchor on
// the main worktree instead: a relative gitflow.worktreePath template resolves
// against it, and removing the worktree the user is inside hands them back to
// it. Git always lists the main worktree first, so the first porcelain record
// answers this; a repository whose worktree list cannot be parsed falls back to
// the parent of the common git dir, which is the main work-tree root of any
// non-bare repository.
func (r *Repo) MainWorkTree() (string, error) {
	entries, err := r.ListWorktrees()
	if err != nil {
		return "", err
	}
	for _, entry := range entries {
		if entry.Main {
			return entry.Path, nil
		}
	}
	return filepath.Dir(r.commonGitDir), nil
}

// gitCmd builds a git command bound to this repository's work tree.
func (r *Repo) gitCmd(args ...string) *exec.Cmd {
	return gitCommand(r.workTree, args...)
}

// Init creates a new git repository in dir with initialBranch as its initial
// branch, overriding any ambient init.defaultBranch. dir must already exist and
// must not already be a repository (callers check that first). It returns git's
// own output ("Initialized empty Git repository in ...") so the caller can show
// it, since CombinedOutput would otherwise swallow it.
func Init(dir string, initialBranch string) (string, error) {
	if strings.TrimSpace(dir) == "" {
		return "", fmt.Errorf("git.Init: directory must not be empty")
	}
	if strings.TrimSpace(initialBranch) == "" {
		return "", fmt.Errorf("git.Init: initial branch must not be empty")
	}
	output, err := gitCommand(dir, "init", "--initial-branch="+initialBranch).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to initialize git repository in %s: %s: %w", dir, strings.TrimSpace(string(output)), err)
	}
	return strings.TrimSpace(string(output)), nil
}

// Open resolves dir to a git repository and returns a handle carrying absolute
// work-tree, git-dir, and common-git-dir paths. It errors on an empty dir or a
// directory that is not inside a git work tree; it never falls back to the
// process working directory.
func Open(dir string) (*Repo, error) {
	if strings.TrimSpace(dir) == "" {
		return nil, fmt.Errorf("git.Open: directory must not be empty")
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("git.Open: resolve %q: %w", dir, err)
	}
	info, err := os.Stat(abs)
	if err != nil {
		return nil, fmt.Errorf("git.Open: stat %q: %w", abs, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("git.Open: %q is not a directory", abs)
	}

	run := func(args ...string) (string, error) {
		out, err := gitCommand(abs, args...).Output()
		return strings.TrimSpace(string(out)), err
	}

	workTree, err := run("rev-parse", "--show-toplevel")
	if err != nil {
		return nil, fmt.Errorf("git.Open: %q is not inside a git work tree: %w", dir, err)
	}
	gitDir, err := run("rev-parse", "--absolute-git-dir")
	if err != nil {
		return nil, fmt.Errorf("git.Open: resolve git dir for %q: %w", dir, err)
	}
	commonRaw, err := run("rev-parse", "--git-common-dir")
	if err != nil {
		return nil, fmt.Errorf("git.Open: resolve common git dir for %q: %w", dir, err)
	}

	commonGitDir := commonRaw
	if !filepath.IsAbs(commonGitDir) {
		// --git-common-dir may be relative to the directory git ran in (abs).
		commonGitDir = filepath.Join(abs, commonGitDir)
	}

	return &Repo{
		workTree:     filepath.Clean(workTree),
		gitDir:       filepath.Clean(gitDir),
		commonGitDir: filepath.Clean(commonGitDir),
	}, nil
}

// GetCurrentBranch returns the current Git branch
func (r *Repo) GetCurrentBranch() (string, error) {
	// Check if we have any commits
	hasCommits, err := r.HasCommits()
	if err != nil {
		return "", fmt.Errorf("failed to check if repository has commits: %w", err)
	}

	if !hasCommits {
		// If no commits, there's no current branch
		return "", nil
	}

	output, err := r.gitCmd("rev-parse", "--abbrev-ref", "HEAD").Output()
	if err != nil {
		return "", fmt.Errorf("failed to get current branch: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

// BranchExists checks if a branch exists
func (r *Repo) BranchExists(branch string) error {
	if err := r.gitCmd("rev-parse", "--verify", "--quiet", "refs/heads/"+branch).Run(); err != nil {
		return fmt.Errorf("branch '%s' does not exist", branch)
	}
	return nil
}

// BranchOrCommitExists checks if a branch, tag, or commit exists
func (r *Repo) BranchOrCommitExists(ref string) error {
	if err := r.gitCmd("rev-parse", "--verify", "--quiet", ref).Run(); err != nil {
		return fmt.Errorf("reference '%s' does not exist", ref)
	}
	return nil
}

// IsAncestor reports whether ancestor is an ancestor of (or identical to)
// descendant. Identical refs yield true, which is exactly the "already
// fast-forwarded" rule the finish --ff-only precondition relies on.
//
// The query is read-only: it inspects refs and never touches HEAD, the index, or
// the work tree. MergeFFOnly is not a substitute — it checks out and mutates the
// target branch, and it reports success when the target is merely ahead.
//
// git merge-base --is-ancestor exits 0 for true and 1 for false. Any other exit
// status (notably 128 for an unknown ref or a broken repository) is returned as
// an error rather than collapsed into the boolean.
func (r *Repo) IsAncestor(ancestor string, descendant string) (bool, error) {
	output, err := r.gitCmd("merge-base", "--is-ancestor", ancestor, descendant).CombinedOutput()
	if err == nil {
		return true, nil
	}
	if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
		return false, nil
	}
	return false, fmt.Errorf("failed to check whether '%s' is an ancestor of '%s': %w: %s", ancestor, descendant, err, strings.TrimSpace(string(output)))
}

// CreateBranch creates a new branch
func (r *Repo) CreateBranch(name string, startPoint string) error {
	// Check if we have any commits
	hasCommits, err := r.HasCommits()
	if err != nil {
		return fmt.Errorf("failed to check if repository has commits: %w", err)
	}

	if !hasCommits {
		// If no commits, create an initial commit first
		if err := r.CreateInitialCommit(name); err != nil {
			return fmt.Errorf("failed to create initial commit: %w", err)
		}
		return nil
	}

	// If startPoint is empty, use the current branch
	if startPoint == "" {
		currentBranch, err := r.GetCurrentBranch()
		if err != nil {
			return fmt.Errorf("failed to get current branch: %w", err)
		}
		startPoint = currentBranch
	}

	if _, err := r.gitCmd("checkout", "-b", name, startPoint).Output(); err != nil {
		return fmt.Errorf("failed to create branch: %w", err)
	}
	return nil
}

// Checkout checks out a branch
func (r *Repo) Checkout(branch string) error {
	if _, err := r.gitCmd("checkout", branch).Output(); err != nil {
		return fmt.Errorf("failed to checkout branch: %w", err)
	}
	return nil
}

// DeleteBranch deletes a branch
func (r *Repo) DeleteBranch(branch string, force bool) error {
	flag := "-d"
	if force {
		flag = "-D"
	}

	output, err := r.gitCmd("branch", flag, branch).CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to delete branch: %s", string(output))
	}
	return nil
}

// HasCommits checks if the repository has any commits
func (r *Repo) HasCommits() (bool, error) {
	if err := r.gitCmd("rev-parse", "--verify", "HEAD").Run(); err != nil {
		// If error, there are no commits
		return false, nil
	}
	return true, nil
}

// CreateInitialCommit creates an initial commit and branch
func (r *Repo) CreateInitialCommit(branch string) error {
	// Create an empty initial commit
	if _, err := r.gitCmd("commit", "--allow-empty", "-m", "Initial commit").Output(); err != nil {
		return fmt.Errorf("failed to create initial commit: %w", err)
	}

	// Rename the default branch to the target name
	if _, err := r.gitCmd("branch", "-m", branch).Output(); err != nil {
		return fmt.Errorf("failed to rename branch to %s: %w", branch, err)
	}

	return nil
}

// Merge merges a branch into the current branch
func (r *Repo) Merge(branch string, noVerify bool) error {
	args := []string{"merge", "--no-ff"}
	if noVerify {
		args = append(args, "--no-verify")
	}
	args = append(args, branch)

	output, err := r.gitCmd(args...).CombinedOutput()
	outputStr := string(output)

	// Check for merge conflicts - Git returns exit code 1 and specific output patterns
	if err != nil {
		// Check if there are unmerged paths (conflicts)
		conflictOutput, _ := r.gitCmd("ls-files", "--unmerged").Output()

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
func (r *Repo) Rebase(branch string) error {
	output, err := r.gitCmd("rebase", branch).CombinedOutput()
	if err != nil {
		if strings.Contains(string(output), "conflict") {
			return fmt.Errorf("rebase conflict: %s", string(output))
		}
		return fmt.Errorf("failed to rebase branch: %s", string(output))
	}
	return nil
}

// SquashMerge performs a squash merge of a branch into the current branch
func (r *Repo) SquashMerge(branch string, noVerify bool) error {
	args := []string{"merge", "--squash"}
	if noVerify {
		args = append(args, "--no-verify")
	}
	args = append(args, branch)

	output, err := r.gitCmd(args...).CombinedOutput()
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
	output, err = r.gitCmd(commitArgs...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to commit squashed changes: %s", string(output))
	}

	return nil
}

// ListBranches returns a list of all branches in the repository
func (r *Repo) ListBranches() ([]string, error) {
	output, err := r.gitCmd("branch", "--format=%(refname:short)").Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list branches: %w", err)
	}

	branches := []string{}
	for _, branch := range strings.Split(string(output), "\n") {
		if branch != "" {
			branches = append(branches, strings.TrimSpace(branch))
		}
	}

	return branches, nil
}

// HasConflicts checks if there are unresolved conflicts
func (r *Repo) HasConflicts() bool {
	output, err := r.gitCmd("diff", "--name-only", "--diff-filter=U").Output()
	if err != nil {
		return false
	}
	return len(output) > 0
}

// IsGitMergeInProgress checks if git is in a merge state by looking for MERGE_HEAD
func (r *Repo) IsGitMergeInProgress() bool {
	_, err := os.Stat(filepath.Join(r.gitDir, "MERGE_HEAD"))
	return err == nil
}

// IsGitRebaseInProgress checks if git is in a rebase state by looking for
// rebase-merge/ (merge backend, including interactive) or rebase-apply/ (legacy apply backend)
func (r *Repo) IsGitRebaseInProgress() bool {
	if _, err := os.Stat(filepath.Join(r.gitDir, "rebase-merge")); err == nil {
		return true
	}
	if _, err := os.Stat(filepath.Join(r.gitDir, "rebase-apply")); err == nil {
		return true
	}
	return false
}

// IsGitSquashMergeInProgress checks if git is in a squash merge state.
// Squash merges create SQUASH_MSG but not MERGE_HEAD. However, SQUASH_MSG also
// appears during interactive rebase squash steps, so we exclude that case.
func (r *Repo) IsGitSquashMergeInProgress() bool {
	if _, err := os.Stat(filepath.Join(r.gitDir, "SQUASH_MSG")); err != nil {
		return false
	}
	// Exclude interactive rebase squash steps
	if _, err := os.Stat(filepath.Join(r.gitDir, "rebase-merge")); err == nil {
		return false
	}
	return true
}

// MergeAbort aborts the current merge
func (r *Repo) MergeAbort() error {
	if err := r.gitCmd("merge", "--abort").Run(); err != nil {
		return fmt.Errorf("failed to abort merge: %w", err)
	}
	return nil
}

// RebaseAbort aborts the current rebase
func (r *Repo) RebaseAbort() error {
	output, err := r.gitCmd("rebase", "--abort").CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to abort rebase: %s", string(output))
	}
	return nil
}

// RenameBranch renames oldBranch to newBranch. Both names are required; there is
// no rename-current-branch shorthand — pass the resolved current branch name if
// that is the intent.
func (r *Repo) RenameBranch(oldBranch, newBranch string) error {
	return r.renameBranch(oldBranch, newBranch, false)
}

// RenameBranchForce renames a Git branch using the force flag (-M). This is
// required for a case-only rename on case-insensitive filesystems, where the
// destination name folds to the same existing ref and the non-forcing -m
// refuses with "a branch named '…' already exists". Callers must confirm the
// rename does not clobber a genuinely different branch before forcing.
func (r *Repo) RenameBranchForce(oldBranch, newBranch string) error {
	return r.renameBranch(oldBranch, newBranch, true)
}

func (r *Repo) renameBranch(oldBranch, newBranch string, force bool) error {
	flag := "-m"
	if force {
		flag = "-M"
	}

	output, err := r.gitCmd("branch", flag, oldBranch, newBranch).CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to rename branch: %s", strings.TrimSpace(string(output)))
	}
	return nil
}

// Fetch performs a git fetch from the specified remote
func (r *Repo) Fetch(remote string) error {
	output, err := r.gitCmd("fetch", remote).CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to fetch from remote '%s': %s", remote, string(output))
	}
	return nil
}

// DeleteRemoteBranch deletes a branch from a remote repository
func (r *Repo) DeleteRemoteBranch(remote, branch string) error {
	output, err := r.gitCmd("push", remote, ":"+branch).CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to delete remote branch: %s", string(output))
	}
	return nil
}

// RemoteExists checks if a remote is configured in the local git config.
// This is a local-only check (no network call).
func (r *Repo) RemoteExists(remote string) bool {
	return r.gitCmd("remote", "get-url", remote).Run() == nil
}

// RemoteBranchExists checks if a remote branch exists
func (r *Repo) RemoteBranchExists(remote, branch string) bool {
	ref := fmt.Sprintf("refs/remotes/%s/%s", remote, branch)
	return r.gitCmd("rev-parse", "--verify", "--quiet", ref).Run() == nil
}

// TagOptions contains options for tag creation
type TagOptions struct {
	Message     string // Tag message (required for annotated tags)
	MessageFile string // File containing the message (optional, overrides Message)
	Sign        bool   // Whether to sign the tag (optional)
	SigningKey  string // Key to use for signing (optional, implies Sign=true)
}

// CreateTag creates a Git tag with the specified options
func (r *Repo) CreateTag(tagName string, options *TagOptions) error {
	// Check if tag already exists
	if err := r.gitCmd("show-ref", "--tags", tagName).Run(); err == nil {
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

	output, err := r.gitCmd(args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to create tag '%s': %w (output: %s)", tagName, err, string(output))
	}

	return nil
}

// RebaseWithOptions rebases the current branch onto another branch with optional preserve-merges
func (r *Repo) RebaseWithOptions(targetBranch string, preserveMerges bool) error {
	args := []string{"rebase"}
	if preserveMerges {
		args = append(args, "--preserve-merges")
	}
	args = append(args, targetBranch)

	output, err := r.gitCmd(args...).CombinedOutput()
	if err != nil {
		if strings.Contains(string(output), "conflict") {
			return fmt.Errorf("rebase conflict: %s", string(output))
		}
		return fmt.Errorf("failed to rebase branch: %s", string(output))
	}
	return nil
}

// MergeFFOnly attempts a fast-forward-only merge of the given branch into the current branch.
// Returns an error if the merge cannot be fast-forwarded.
func (r *Repo) MergeFFOnly(branch string) error {
	output, err := r.gitCmd("merge", "--ff-only", branch).CombinedOutput()
	if err != nil {
		return fmt.Errorf("fast-forward merge of %q failed: %w: %s", branch, err, strings.TrimSpace(string(output)))
	}
	return nil
}

// MergeWithOptions merges a branch into current branch with optional no-fast-forward
func (r *Repo) MergeWithOptions(branchName string, noFF bool, noVerify bool) error {
	args := []string{"merge"}
	if noFF {
		args = append(args, "--no-ff")
	}
	if noVerify {
		args = append(args, "--no-verify")
	}
	args = append(args, branchName)

	output, err := r.gitCmd(args...).CombinedOutput()
	outputStr := string(output)

	if err != nil {
		conflictOutput, _ := r.gitCmd("ls-files", "--unmerged").Output()

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
func (r *Repo) MergeWithMessage(branchName string, message string, noFF bool, noVerify bool) error {
	args := []string{"merge"}
	if noFF {
		args = append(args, "--no-ff")
	}
	if noVerify {
		args = append(args, "--no-verify")
	}
	args = append(args, "-m", message, branchName)

	output, err := r.gitCmd(args...).CombinedOutput()
	outputStr := string(output)

	if err != nil {
		conflictOutput, _ := r.gitCmd("ls-files", "--unmerged").Output()

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
func (r *Repo) Commit(message string, noVerify bool) error {
	args := []string{"commit", "-m", message}
	if noVerify {
		args = append(args, "--no-verify")
	}
	output, err := r.gitCmd(args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to commit: %s", string(output))
	}
	return nil
}

// RebaseContinue continues an ongoing rebase operation after conflicts are resolved
func (r *Repo) RebaseContinue() error {
	output, err := r.gitCmd("rebase", "--continue").CombinedOutput()
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
func (r *Repo) MergeSquashWithMessage(branchName string, message string, noVerify bool) error {
	args := []string{"merge", "--squash"}
	if noVerify {
		args = append(args, "--no-verify")
	}
	args = append(args, branchName)

	output, err := r.gitCmd(args...).CombinedOutput()
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
	output, err = r.gitCmd(commitArgs...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to commit squashed changes: %s", string(output))
	}

	return nil
}

// PushBranch pushes a local branch to a remote and sets up tracking
func (r *Repo) PushBranch(remote, branch string, pushOptions []string) error {
	args := []string{"push", "-u", remote}

	for _, opt := range pushOptions {
		args = append(args, "-o", opt)
	}

	args = append(args, branch)

	output, err := r.gitCmd(args...).CombinedOutput()
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
func (r *Repo) PushRef(remote, branch string) error {
	output, err := r.gitCmd("push", remote, branch).CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to push branch '%s' to '%s': %w (output: %s)", branch, remote, err, strings.TrimSpace(string(output)))
	}
	return nil
}

// PushTag pushes a single tag to a remote with `git push <remote> tag <tag>`.
// On failure it returns a wrapped error that embeds the underlying error and the
// trimmed combined output.
func (r *Repo) PushTag(remote, tag string) error {
	output, err := r.gitCmd("push", remote, "tag", tag).CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to push tag '%s' to '%s': %w (output: %s)", tag, remote, err, strings.TrimSpace(string(output)))
	}
	return nil
}

// CreateTrackingBranch creates a local branch that tracks a remote branch
func (r *Repo) CreateTrackingBranch(localBranch, remote, remoteBranch string) error {
	output, err := r.gitCmd("checkout", "-b", localBranch, "--track",
		fmt.Sprintf("%s/%s", remote, remoteBranch)).CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to create tracking branch: %s", string(output))
	}
	return nil
}

// GetTrackingBranch returns the remote tracking branch for a local branch.
// Returns the full tracking reference (e.g., "origin/feature/foo") or an error
// if no tracking branch is configured.
func (r *Repo) GetTrackingBranch(branch string) (string, error) {
	output, err := r.gitCmd("rev-parse", "--abbrev-ref", branch+"@{upstream}").CombinedOutput()
	if err != nil {
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
func (r *Repo) CompareBranchWithRemote(branch string) (BranchSyncStatus, int, error) {
	trackingBranch, err := r.GetTrackingBranch(branch)
	if err != nil {
		return SyncStatusNoTracking, 0, err
	}

	output, err := r.gitCmd("rev-list", "--left-right", "--count", branch+"..."+trackingBranch).CombinedOutput()
	if err != nil {
		return "", 0, fmt.Errorf("failed to compare branches: %s", string(output))
	}

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
func (r *Repo) FetchBranch(remote, branch string) error {
	cmd := r.gitCmd("fetch", remote, branch)
	// Pin the locale so isRemoteRefNotFound classifies stderr against git's English phrasing,
	// regardless of the user's LANG/LC_* settings. A localized "missing ref" message would
	// otherwise be misclassified as a fatal transport failure.
	cmd.Env = append(os.Environ(), "LC_ALL=C")
	output, err := cmd.CombinedOutput()
	if err != nil {
		stderr := strings.TrimSpace(string(output))
		if isRemoteRefNotFound(stderr) {
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
func (r *Repo) HasTrackingBranch(branch string) bool {
	_, err := r.GetTrackingBranch(branch)
	return err == nil
}

// DeleteRemoteTrackingRef removes a stale remote-tracking ref (refs/remotes/<remote>/<branch>)
// from the local repository. It is used to prune a tracking ref once the corresponding remote
// branch is found to be gone. Errors are returned so callers may log them, but they are safe to
// ignore (the ref may already be absent).
func (r *Repo) DeleteRemoteTrackingRef(remote, branch string) error {
	ref := fmt.Sprintf("refs/remotes/%s/%s", remote, branch)
	if output, err := r.gitCmd("update-ref", "-d", ref).CombinedOutput(); err != nil {
		return fmt.Errorf("failed to delete remote tracking ref '%s': %s: %w", ref, strings.TrimSpace(string(output)), err)
	}
	return nil
}
