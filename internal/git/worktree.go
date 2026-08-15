package git

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// isWindows selects case-insensitive path comparison. It is a variable rather
// than a runtime.GOOS test at each use so tests can exercise both branches on
// any host, matching the seam of the same name in internal/hooks.
var isWindows = runtime.GOOS == "windows"

// WorktreeEntry is one record of `git worktree list --porcelain`: a worktree
// registered with the repository, whether it is the main one or a linked one.
type WorktreeEntry struct {
	// Path is the absolute path of the worktree as git records it.
	Path string
	// Branch is the short branch name checked out there, empty when detached.
	Branch string
	// Head is the commit the worktree points at.
	Head string
	// Detached reports a HEAD that is not on a branch.
	Detached bool
	// Bare reports a bare main repository (which has no work tree).
	Bare bool
	// Main reports the main worktree, which git always lists first.
	Main bool
}

// ListWorktrees returns every worktree registered with the repository, the main
// one first.
//
// It parses `git worktree list --porcelain`: blank-line-separated records whose
// lines are `worktree <abs-path>`, `HEAD <sha>`, and then either
// `branch refs/heads/<name>` or `detached`, plus `bare` for a bare main
// repository. Any other line is ignored on purpose — git adds annotations over
// time (`locked`, `prunable`, …) and depending on them would tie git-flow to a
// git newer than the documented 2.17+ floor. Stale entries are therefore
// reported like any other until `git worktree prune` runs.
func (r *Repo) ListWorktrees() ([]WorktreeEntry, error) {
	output, err := r.gitCmd("worktree", "list", "--porcelain").Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list worktrees: %w", err)
	}
	return parseWorktreePorcelain(string(output)), nil
}

// parseWorktreePorcelain turns porcelain output into entries. A record starts at
// each `worktree ` line, so no blank-line bookkeeping is needed.
func parseWorktreePorcelain(output string) []WorktreeEntry {
	var entries []WorktreeEntry
	var current *WorktreeEntry

	flush := func() {
		if current != nil {
			entries = append(entries, *current)
			current = nil
		}
	}

	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimRight(line, "\r")
		switch {
		case strings.HasPrefix(line, "worktree "):
			flush()
			current = &WorktreeEntry{
				Path: filepath.Clean(strings.TrimPrefix(line, "worktree ")),
				Main: len(entries) == 0,
			}
		case current == nil:
			// Anything before the first record header is not ours to interpret.
			continue
		case strings.HasPrefix(line, "HEAD "):
			current.Head = strings.TrimPrefix(line, "HEAD ")
		case strings.HasPrefix(line, "branch "):
			current.Branch = strings.TrimPrefix(strings.TrimPrefix(line, "branch "), "refs/heads/")
		case line == "detached":
			current.Detached = true
		case line == "bare":
			current.Bare = true
		}
	}
	flush()

	return entries
}

// WorktreeForBranch returns the worktree that has branch checked out, or
// (nil, nil) when no worktree holds it. The branch match is exact and
// case-sensitive; the returned entry may be the main worktree, which callers
// distinguish through its Main field.
func (r *Repo) WorktreeForBranch(branch string) (*WorktreeEntry, error) {
	entries, err := r.ListWorktrees()
	if err != nil {
		return nil, err
	}
	for i := range entries {
		if entries[i].Branch == branch {
			return &entries[i], nil
		}
	}
	return nil, nil
}

// AddWorktree creates a linked worktree at path with branch checked out. The
// branch must exist and must not be checked out anywhere else; callers are
// expected to have verified both, so a failure here is reported with git's own
// output (git writes progress to stderr even when it fails).
func (r *Repo) AddWorktree(path string, branch string) error {
	output, err := r.gitCmd("worktree", "add", path, branch).CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to add worktree at %s: %s: %w", path, strings.TrimSpace(string(output)), err)
	}
	return nil
}

// RemoveWorktree removes the linked worktree at path, refusing the main worktree
// before invoking git at all. force passes --force, which git requires for a
// worktree with uncommitted or untracked changes.
func (r *Repo) RemoveWorktree(path string, force bool) error {
	if err := r.refuseMainWorktree(path, "remove"); err != nil {
		return err
	}

	args := []string{"worktree", "remove"}
	if force {
		args = append(args, "--force")
	}
	args = append(args, path)

	output, err := r.gitCmd(args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to remove worktree at %s: %s: %w", path, strings.TrimSpace(string(output)), err)
	}
	return nil
}

// PruneWorktrees drops the admin entries of worktrees whose directories are gone.
func (r *Repo) PruneWorktrees() error {
	output, err := r.gitCmd("worktree", "prune").CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to prune worktrees: %s: %w", strings.TrimSpace(string(output)), err)
	}
	return nil
}

// DetachWorktree detaches the HEAD of the worktree at path from its branch,
// leaving the tree and any uncommitted work untouched and freeing the branch for
// checkout elsewhere. It refuses the main worktree. `checkout --detach` is used
// rather than `switch --detach`, which needs git 2.23.
func (r *Repo) DetachWorktree(path string) error {
	if err := r.refuseMainWorktree(path, "detach"); err != nil {
		return err
	}

	output, err := gitCommand(path, "checkout", "--detach").CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to detach worktree at %s: %s: %w", path, strings.TrimSpace(string(output)), err)
	}
	return nil
}

// WorktreeHasChanges reports whether the worktree at path has uncommitted or
// untracked changes. It runs `git status --porcelain` with path as the working
// directory — not this handle's work tree — so it answers for the worktree being
// asked about rather than the one the command was invoked from.
//
// Only stdout is parsed. CombinedOutput would fold a stray git warning into the
// porcelain the dirty check reads, so a clean worktree could report changes; git's
// stderr is instead recovered from the failure for the error message.
func (r *Repo) WorktreeHasChanges(path string) (bool, error) {
	output, err := gitCommand(path, "status", "--porcelain").Output()
	if err != nil {
		if detail := stderrOf(err); detail != "" {
			return false, fmt.Errorf("failed to check status of worktree at %s: %s: %w", path, detail, err)
		}
		return false, fmt.Errorf("failed to check status of worktree at %s: %w", path, err)
	}
	return strings.TrimSpace(string(output)) != "", nil
}

// stderrOf returns the trimmed stderr a failed command wrote, which Output()
// records on the *exec.ExitError. It is empty for any other failure.
func stderrOf(err error) string {
	if exitErr, ok := err.(*exec.ExitError); ok {
		return strings.TrimSpace(string(exitErr.Stderr))
	}
	return ""
}

// refuseMainWorktree returns an error when path is the repository's main
// worktree. The main worktree holds the repository itself, so removing or
// detaching it is never what the user meant.
func (r *Repo) refuseMainWorktree(path string, operation string) error {
	mainWorkTree, err := r.MainWorkTree()
	if err != nil {
		return err
	}
	if SamePath(path, mainWorkTree) {
		return fmt.Errorf("refusing to %s the main worktree at %s", operation, mainWorkTree)
	}
	return nil
}

// SamePath reports whether two paths denote the same location, resolving
// symlinks on BOTH sides. This matters on macOS, where a temporary directory is
// reached through /var while git records the real /private/var path.
//
// Case is folded on Windows, where the casing git records for a worktree need
// not match what the user typed or what os.Getwd returns.
func SamePath(a, b string) bool {
	return foldPath(resolvePath(a)) == foldPath(resolvePath(b))
}

// IsWithin reports whether child is parent or lives underneath it, comparing on
// a separator boundary so /a/foobar is not treated as being inside /a/foo. Both
// sides are symlink-resolved, and case is folded on Windows as in SamePath.
func IsWithin(child, parent string) bool {
	c, p := foldPath(resolvePath(child)), foldPath(resolvePath(parent))
	if c == p {
		return true
	}
	return strings.HasPrefix(c, p+string(filepath.Separator))
}

// foldPath returns p in the form used for comparison: unchanged where path
// names are case-sensitive, upper-cased where they are not.
//
// The fold is to UPPER case because that is the direction Windows folds in:
// NTFS and exFAT carry an upcase table and ordinal case-insensitive comparison
// upper-cases both sides. The directions disagree on characters whose lowercase
// forms are distinct but whose uppercase form is shared — Greek final sigma is
// the reachable example, where lowercasing leaves "ς" and "σ" different while
// Windows upper-cases both to "Σ" and treats the paths as one location. Neither
// direction reproduces the volume's table exactly; upper is the one that fails
// on rarer characters and in the direction Windows itself works.
//
// The gate is Windows-only, which leaves a residual gap: a macOS volume is
// case-insensitive by default and has the same defect, but case sensitivity
// there is a per-volume property that runtime.GOOS cannot decide. Closing it
// needs per-volume detection and is deliberately left alone here.
//
// Folding still matters even though EvalSymlinks rewrites existing components
// to their on-disk casing on Windows: that canonicalization only reaches paths
// that exist. The comparisons here routinely see paths that do not — a computed
// worktree path, a stale entry whose directory is gone, an EvalSymlinks failure
// on a network share — and there resolvePath's fallback preserves the caller's
// raw casing all the way to the comparison.
//
// Only comparisons see the folded form. Paths stored, passed to git or reported
// to the user keep the casing they arrived with.
func foldPath(p string) string {
	if isWindows {
		return strings.ToUpper(p)
	}
	return p
}

// resolvePath returns an absolute, symlink-resolved form of p, tolerating a path
// that does not exist: filepath.EvalSymlinks fails outright on a missing path, so
// the deepest existing ancestor is resolved and the remainder appended. Paths
// under comparison here routinely do not exist yet (a computed worktree path) or
// no longer exist (a stale worktree entry).
func resolvePath(p string) string {
	abs, err := filepath.Abs(p)
	if err != nil {
		abs = filepath.Clean(p)
	}

	current := abs
	remainder := ""
	for {
		resolved, err := filepath.EvalSymlinks(current)
		if err == nil {
			if remainder == "" {
				return filepath.Clean(resolved)
			}
			return filepath.Join(resolved, remainder)
		}
		parent := filepath.Dir(current)
		if parent == current {
			return abs
		}
		remainder = filepath.Join(filepath.Base(current), remainder)
		current = parent
	}
}
