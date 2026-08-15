package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gittower/git-flow-next/internal/config"
	"github.com/gittower/git-flow-next/internal/errors"
	"github.com/gittower/git-flow-next/internal/git"
	"github.com/gittower/git-flow-next/internal/navigate"
	"github.com/gittower/git-flow-next/internal/worktree"
	"github.com/spf13/cobra"
)

// unmanagedTag marks a worktree git-flow did not create, the distinction that
// decides what the cleanup commands may remove.
const unmanagedTag = "(unmanaged)"

// detachedBranchLabel stands in for the branch column of a worktree whose HEAD is
// not on a branch.
const detachedBranchLabel = "(detached)"

var worktreeCmd = &cobra.Command{
	Use:   "worktree",
	Short: "Manage worktrees for branches",
	Long: `Manage git worktrees by branch name.

Worktrees are addressed by their full branch name (e.g. feature/user-auth), not
by topic type plus short name. Paths are computed from the gitflow.worktreePath
template unless --path overrides them.

git-flow records the worktrees it creates, so 'worktree list' can tell them apart
from worktrees created with plain 'git worktree add'.`,
	Example: `  git flow worktree add feature/user-auth
  git flow worktree list
  git flow worktree remove feature/user-auth
  git flow worktree prune
  git flow worktree path feature/user-auth`,
}

var worktreeAddCmd = &cobra.Command{
	Use:   "add <branch>",
	Short: "Create a worktree for an existing branch",
	Long: `Create a worktree for an existing branch at the computed path.

The branch must exist and must not be checked out in another worktree. Parent
directories of the computed path are created as needed.`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		path, _ := cmd.Flags().GetString("path")
		noCD, _ := cmd.Flags().GetBool("no-cd")
		quiet, _ := cmd.Flags().GetBool("quiet")
		WorktreeAddCommand(args[0], path, noCD, quiet)
	},
}

var worktreeRemoveCmd = &cobra.Command{
	Use:   "remove <branch>",
	Short: "Remove the worktree of a branch, keeping the branch",
	Long: `Remove the worktree that has the given branch checked out.

The branch itself is kept. Removal refuses a worktree with uncommitted or
untracked changes unless --force is given, and never removes the main worktree.
Because the command names its target explicitly, it removes the worktree whatever
its origin.`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		force, _ := cmd.Flags().GetBool("force")
		noCD, _ := cmd.Flags().GetBool("no-cd")
		WorktreeRemoveCommand(args[0], force, noCD)
	},
}

var worktreeListCmd = &cobra.Command{
	Use:   "list",
	Short: "List linked worktrees",
	Long: `List the linked worktrees of this repository with their branch and path.

The main worktree is excluded. Worktrees git-flow did not create are tagged
(unmanaged); a worktree whose HEAD is detached shows (detached) instead of a
branch name.`,
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		WorktreeListCommand()
	},
}

var worktreePruneCmd = &cobra.Command{
	Use:   "prune",
	Short: "Drop admin entries whose worktree directories are gone",
	Long: `Drop the administrative entries of worktrees whose directories no longer exist.

Provenance markers whose branch has no live worktree are dropped as well, so they
never outlive what they describe.`,
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		WorktreePruneCommand()
	},
}

var worktreePathCmd = &cobra.Command{
	Use:   "path <branch>",
	Short: "Print the computed worktree path for a branch",
	Long: `Print the path the gitflow.worktreePath template computes for a branch.

Nothing is created and nothing is changed.`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		WorktreePathCommand(args[0])
	},
}

// WorktreeAddCommand is the implementation of 'git flow worktree add'.
func WorktreeAddCommand(branch string, path string, noCD bool, quiet bool) {
	runWorktreeCommand(func(repo *git.Repo) error {
		return executeWorktreeAdd(repo, branch, path, noCD, quiet)
	})
}

// WorktreeRemoveCommand is the implementation of 'git flow worktree remove'.
func WorktreeRemoveCommand(branch string, force bool, noCD bool) {
	runWorktreeCommand(func(repo *git.Repo) error {
		return executeWorktreeRemove(repo, branch, force, noCD)
	})
}

// WorktreeListCommand is the implementation of 'git flow worktree list'.
func WorktreeListCommand() {
	runWorktreeCommand(executeWorktreeList)
}

// WorktreePruneCommand is the implementation of 'git flow worktree prune'.
func WorktreePruneCommand() {
	runWorktreeCommand(executeWorktreePrune)
}

// WorktreePathCommand is the implementation of 'git flow worktree path'.
func WorktreePathCommand(branch string) {
	runWorktreeCommand(func(repo *git.Repo) error {
		return executeWorktreePath(repo, branch)
	})
}

// runWorktreeCommand opens the invocation repository, runs one subcommand, and
// maps a typed git-flow error to its exit code. The five subcommands share it
// rather than repeating the wrapper five times.
func runWorktreeCommand(run func(repo *git.Repo) error) {
	repo := mustOpenRepo()
	if err := run(repo); err != nil {
		exitCode := errors.ExitCodeGitError
		if flowErr, ok := err.(errors.Error); ok {
			exitCode = flowErr.ExitCode()
		}
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(int(exitCode))
	}
}

// requireInitialized fails unless git-flow is initialized in the repository.
// Every worktree subcommand gates on it, including the read-only ones, so an
// uninitialized repository never sees half a command work.
func requireInitialized(repo *git.Repo) error {
	initialized, err := config.IsInitialized(repo)
	if err != nil {
		return &errors.GitError{Operation: "check if git-flow is initialized", Err: err}
	}
	if !initialized {
		return &errors.NotInitializedError{}
	}
	return nil
}

// loadWorktreeConfig gates on initialization and loads the configuration the
// path template expands against.
func loadWorktreeConfig(repo *git.Repo) (*config.Config, error) {
	if err := requireInitialized(repo); err != nil {
		return nil, err
	}
	cfg, err := config.Load(repo)
	if err != nil {
		return nil, &errors.GitError{Operation: "load configuration", Err: err}
	}
	return cfg, nil
}

// executeWorktreeAdd creates a worktree for an existing branch.
func executeWorktreeAdd(repo *git.Repo, branch string, pathFlag string, noCD bool, quiet bool) error {
	cfg, err := loadWorktreeConfig(repo)
	if err != nil {
		return err
	}

	if err := repo.BranchExists(branch); err != nil {
		return &errors.LocalBranchNotFoundError{BranchName: branch}
	}

	// A branch can be checked out in only one worktree, so refuse before git
	// does — and distinguish "it is in the main worktree" from "it already has a
	// worktree", which are different problems for the user.
	existing, err := repo.WorktreeForBranch(branch)
	if err != nil {
		return &errors.GitError{Operation: "look up worktree for branch", Err: err}
	}
	if existing != nil {
		if existing.Main {
			return &errors.BranchCheckedOutElsewhereError{Branch: branch, Path: existing.Path}
		}
		return &errors.WorktreeExistsError{Branch: branch, Path: existing.Path}
	}

	target, err := createWorktreeAt(cfg, repo, branch, pathFlag, false)
	if err != nil {
		return err
	}

	fmt.Printf("Created worktree for branch '%s' at %s\n", branch, target)
	fmt.Printf("To switch to it: cd %s\n", target)
	printShellInitTip(quiet)

	if !noCD {
		// Written only now, after the worktree exists.
		if err := navigate.WriteDestination(target); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
		}
	}
	return nil
}

// executeWorktreeRemove removes the worktree holding a branch, keeping the branch.
func executeWorktreeRemove(repo *git.Repo, branch string, force bool, noCD bool) error {
	if err := requireInitialized(repo); err != nil {
		return err
	}

	mainWorkTree, err := repo.MainWorkTree()
	if err != nil {
		return &errors.GitError{Operation: "resolve the main worktree", Err: err}
	}

	entry, err := repo.WorktreeForBranch(branch)
	if err != nil {
		return &errors.GitError{Operation: "look up worktree for branch", Err: err}
	}
	// The main-worktree refusal comes first: naming the branch checked out there
	// is a mistake worth its own message, not a "nothing to remove".
	if entry != nil && entry.Main {
		return &errors.MainWorktreeError{Operation: "remove", Path: entry.Path}
	}
	if entry == nil {
		return &errors.WorktreeNotFoundError{Branch: branch}
	}

	// Anything other than a missing directory — an unreadable parent, most likely —
	// is surfaced here rather than being read as "the directory is there", which
	// would only fail further down with a message about something else.
	directoryExists := true
	if _, statErr := os.Stat(entry.Path); statErr != nil {
		if !os.IsNotExist(statErr) {
			return &errors.GitError{Operation: "inspect the worktree directory", Err: statErr}
		}
		directoryExists = false
	}

	// A directory that is already gone has nothing to lose, so it counts as clean
	// and the dirty check is skipped rather than failing on a missing worktree.
	if !force && directoryExists {
		dirty, err := repo.WorktreeHasChanges(entry.Path)
		if err != nil {
			return &errors.GitError{Operation: "check worktree for changes", Err: err}
		}
		if dirty {
			return &errors.WorktreeDirtyError{Branch: branch, Path: entry.Path}
		}
	}

	// Decide everything that depends on the current directory BEFORE the removal
	// deletes it: whether the user is standing inside the target, and where the
	// destination file is.
	// A failing Getwd means the current directory is already unreadable or gone,
	// so treat the user as not standing inside the target: writing a destination
	// on a guess would move a shell that never asked to be moved, while not
	// writing one costs at most a manual cd.
	strandedUser := false
	if cwd, err := os.Getwd(); err == nil {
		strandedUser = git.IsWithin(cwd, entry.Path)
	}
	destinationFile := ""
	if !noCD && strandedUser {
		destinationFile = navigate.DestinationFile()
	}

	// Once the worktree is gone, every git command bound to it fails, so switch
	// to a handle on the main worktree first.
	opRepo := repo
	if !git.SamePath(repo.WorkTree(), mainWorkTree) {
		opRepo, err = git.Open(mainWorkTree)
		if err != nil {
			return &errors.GitError{Operation: "open the main worktree", Err: err}
		}
	}

	if err := opRepo.RemoveWorktree(entry.Path, force); err != nil {
		if !directoryExists {
			return &errors.GitError{
				Operation: "remove worktree",
				Err:       fmt.Errorf("%w (the directory is already gone; 'git flow worktree prune' drops stale entries)", err),
			}
		}
		return &errors.GitError{Operation: "remove worktree", Err: err}
	}

	if err := worktree.ClearMarker(opRepo, branch); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
	}

	// Written only now, after the removal succeeded, and only when the user would
	// otherwise be left in a deleted directory.
	if destinationFile != "" {
		if err := navigate.WriteDestinationTo(destinationFile, mainWorkTree); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
		}
	}

	fmt.Printf("Removed worktree for branch '%s' at %s\n", branch, entry.Path)
	return nil
}

// executeWorktreeList prints one row per linked worktree.
func executeWorktreeList(repo *git.Repo) error {
	if err := requireInitialized(repo); err != nil {
		return err
	}

	entries, err := repo.ListWorktrees()
	if err != nil {
		return &errors.GitError{Operation: "list worktrees", Err: err}
	}

	type row struct {
		branch string
		path   string
		tag    string
	}
	var rows []row
	width := 0
	for _, entry := range entries {
		if entry.Main {
			continue
		}
		r := row{branch: entry.Branch, path: entry.Path}
		if entry.Branch == "" {
			// A detached worktree has no branch to key provenance on, so it can
			// only be reported as unmanaged.
			r.branch = detachedBranchLabel
			r.tag = unmanagedTag
		} else if !worktree.IsManaged(repo, entry.Branch) {
			r.tag = unmanagedTag
		}
		if len(r.branch) > width {
			width = len(r.branch)
		}
		rows = append(rows, r)
	}

	if len(rows) == 0 {
		fmt.Println("No linked worktrees found")
		return nil
	}

	for _, r := range rows {
		line := fmt.Sprintf("%-*s  %s", width, r.branch, r.path)
		if r.tag != "" {
			line += "  " + r.tag
		}
		fmt.Println(line)
	}
	return nil
}

// executeWorktreePrune drops stale admin entries and the markers left behind.
func executeWorktreePrune(repo *git.Repo) error {
	if err := requireInitialized(repo); err != nil {
		return err
	}

	before, err := repo.ListWorktrees()
	if err != nil {
		return &errors.GitError{Operation: "list worktrees", Err: err}
	}
	if err := repo.PruneWorktrees(); err != nil {
		return &errors.GitError{Operation: "prune worktrees", Err: err}
	}
	after, err := repo.ListWorktrees()
	if err != nil {
		return &errors.GitError{Operation: "list worktrees", Err: err}
	}
	if err := worktree.SweepMarkers(repo); err != nil {
		return &errors.GitError{Operation: "sweep worktree provenance markers", Err: err}
	}

	remaining := make(map[string]bool, len(after))
	for _, entry := range after {
		remaining[entry.Path] = true
	}
	pruned := 0
	for _, entry := range before {
		if !remaining[entry.Path] {
			fmt.Printf("Pruned worktree entry at %s\n", entry.Path)
			pruned++
		}
	}
	if pruned == 0 {
		fmt.Println("No stale worktree entries found")
	}
	return nil
}

// executeWorktreePath prints the computed path for a branch and nothing else.
func executeWorktreePath(repo *git.Repo, branch string) error {
	cfg, err := loadWorktreeConfig(repo)
	if err != nil {
		return err
	}
	path, err := worktree.ComputePath(cfg, repo, branch)
	if err != nil {
		return &errors.GitError{Operation: "compute worktree path", Err: err}
	}
	fmt.Println(path)
	return nil
}

// resolveWorktreeTarget returns the absolute path the worktree belongs at: the
// --path value when given, otherwise the computed one.
//
// A relative --path resolves against the INVOCATION directory, which is what a
// hand-typed path means, while a relative gitflow.worktreePath template resolves
// against the main worktree root. The asymmetry is deliberate: a template is a
// repository-wide setting, a typed path is one command's argument.
func resolveWorktreeTarget(cfg *config.Config, repo *git.Repo, branch string, pathFlag string) (string, error) {
	// Use the TRIMMED value, the same one the emptiness test looks at: surrounding
	// whitespace in a hand-typed path is a typo, not a directory name.
	if trimmed := strings.TrimSpace(pathFlag); trimmed != "" {
		return normalizeInvocationPath(trimmed), nil
	}
	path, err := worktree.ComputePath(cfg, repo, branch)
	if err != nil {
		return "", &errors.GitError{Operation: "compute worktree path", Err: err}
	}
	return path, nil
}

// printShellInitTip points the user at 'git flow shell-init' after a navigation
// they now have to perform by hand.
//
// It prints only when the navigation channel is unused: a caller that set
// GIT_FLOW_CD_FILE has the wrapper installed already, and telling them to
// install it would be noise on every navigating command. The decision is made on
// the resolved destination file rather than on whether the variable exists,
// because an empty value means the same thing as an absent one.
//
// <shell> is printed verbatim: 'git flow shell-init' with no argument errors
// with usage, so naming a placeholder is what keeps the tip honest.
func printShellInitTip(quiet bool) {
	if quiet || navigate.DestinationFile() != "" {
		return
	}
	fmt.Println("Tip: run 'git flow shell-init <shell>' for automatic directory switching")
}

// createWorktreeAt creates the worktree for branch and records git-flow as its
// creator, returning the path it was created at.
//
// It is shared by 'worktree add' and 'checkout --worktree' so the two cannot
// drift: the target resolution, the occupancy rules and the provenance marker
// are decided in exactly one place. The caller is expected to have established
// that the branch exists and has no worktree yet, and prints its own
// confirmation — the wording differs between the two commands.
//
// force removes a plain directory that is in the way. It never removes
// anything else; see removeObstruction.
func createWorktreeAt(cfg *config.Config, repo *git.Repo, branch string, pathFlag string, force bool) (string, error) {
	target, err := resolveWorktreeTarget(cfg, repo, branch, pathFlag)
	if err != nil {
		return "", err
	}

	occupied, err := pathIsOccupied(target)
	if err != nil {
		return "", &errors.GitError{Operation: "inspect target path", Err: err}
	}
	if occupied {
		if !force {
			return "", &errors.WorktreePathOccupiedError{Path: target}
		}
		if err := removeObstruction(repo, target); err != nil {
			return "", err
		}
	}

	// A branch name with slashes computes a nested path, whose intermediate
	// directories have to exist first.
	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		return "", &errors.GitError{Operation: "create worktree parent directory", Err: err}
	}
	if err := repo.AddWorktree(target, branch); err != nil {
		return "", &errors.GitError{Operation: "add worktree", Err: err}
	}

	// A missing marker only means later cleanup leaves this worktree alone, so a
	// failure here warns rather than failing an operation that already succeeded.
	if err := worktree.MarkManaged(repo, branch); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
	}

	return target, nil
}

// removeObstruction removes an occupied target path, but only when what is
// there is a plain directory git-flow can afford to lose.
//
// The three refusals bound the blast radius of a mistyped path template. The
// .git test uses Lstat and refuses on ANY result — the linked-worktree form is a
// .git file and an ordinary clone is a .git directory, so a check that handled
// only one of them would remove the other.
//
// When target is a SYMLINK the guards below follow it while RemoveAll acts on the
// link itself, so the object inspected and the object removed differ. That is not
// destructive — Go's RemoveAll never descends through a symlink — and the blast
// radius is exactly one dangling link, so the asymmetry is left alone rather than
// traded for an Lstat that would refuse a symlinked worktree root outright.
func removeObstruction(repo *git.Repo, target string) error {
	info, err := os.Stat(target)
	if err != nil {
		return &errors.GitError{Operation: "inspect target path", Err: err}
	}
	if !info.IsDir() {
		return &errors.RemovalRefusedError{Path: target, Reason: "it is a file, not a directory"}
	}

	entries, err := repo.ListWorktrees()
	if err != nil {
		return &errors.GitError{Operation: "list worktrees", Err: err}
	}
	for _, entry := range entries {
		if git.SamePath(entry.Path, target) {
			return &errors.RemovalRefusedError{
				Path:   target,
				Reason: "it is a registered worktree of this repository; 'git flow worktree remove <branch>' removes it safely",
			}
		}
	}

	if _, err := os.Lstat(filepath.Join(target, ".git")); err == nil {
		return &errors.RemovalRefusedError{
			Path:   target,
			Reason: "it contains a .git entry, so it looks like a repository",
		}
	} else if !os.IsNotExist(err) {
		return &errors.GitError{Operation: "inspect target path", Err: err}
	}

	if err := os.RemoveAll(target); err != nil {
		return &errors.GitError{Operation: "remove the directory in the way", Err: err}
	}
	fmt.Printf("Removed %s to make room for the worktree\n", target)
	return nil
}

// pathIsOccupied reports whether something is already in the way at path. An
// empty directory is not in the way — git itself accepts one — but a file or a
// directory with content is.
func pathIsOccupied(path string) (bool, error) {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if !info.IsDir() {
		return true, nil
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		return false, err
	}
	return len(entries) > 0, nil
}

func init() {
	worktreeAddCmd.Flags().String("path", "", "Create the worktree at this path instead of the computed one")
	worktreeAddCmd.Flags().Bool("no-cd", false, "Do not write a navigation destination for the calling shell")
	worktreeAddCmd.Flags().BoolP("quiet", "q", false, "Do not print the shell-init tip")

	worktreeRemoveCmd.Flags().Bool("force", false, "Remove even with uncommitted or untracked changes")
	worktreeRemoveCmd.Flags().Bool("no-cd", false, "Do not write a navigation destination for the calling shell")

	worktreeCmd.AddCommand(worktreeAddCmd)
	worktreeCmd.AddCommand(worktreeRemoveCmd)
	worktreeCmd.AddCommand(worktreeListCmd)
	worktreeCmd.AddCommand(worktreePruneCmd)
	worktreeCmd.AddCommand(worktreePathCmd)

	rootCmd.AddCommand(worktreeCmd)
}
