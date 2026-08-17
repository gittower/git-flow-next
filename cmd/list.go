package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/gittower/git-flow-next/internal/config"
	"github.com/gittower/git-flow-next/internal/errors"
	"github.com/gittower/git-flow-next/internal/git"
	"github.com/gittower/git-flow-next/internal/worktree"
)

// ListCommand is the implementation of the list command for topic branches
func ListCommand(branchType string, showWorktrees bool) {
	repo := mustOpenRepo()
	if err := list(repo, branchType, showWorktrees); err != nil {
		var exitCode errors.ExitCode
		if flowErr, ok := err.(errors.Error); ok {
			exitCode = flowErr.ExitCode()
		} else {
			exitCode = errors.ExitCodeGitError
		}
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(int(exitCode))
	}
}

// list performs the actual branch listing logic and returns any errors
func list(repo *git.Repo, branchType string, showWorktrees bool) error {
	// Validate that git-flow is initialized
	initialized, err := config.IsInitialized(repo)
	if err != nil {
		return &errors.GitError{Operation: "check if git-flow is initialized", Err: err}
	}
	if !initialized {
		return &errors.NotInitializedError{}
	}

	// Get configuration
	cfg, err := config.Load(repo)
	if err != nil {
		return &errors.GitError{Operation: "load configuration", Err: err}
	}

	// Get branch configuration
	branchConfig, ok := cfg.Branches[branchType]
	if !ok {
		return &errors.InvalidBranchTypeError{BranchType: branchType}
	}

	// Get the prefix for this branch type
	prefix := branchConfig.Prefix

	// Get all branches
	branches, err := repo.ListBranches()
	if err != nil {
		return &errors.GitError{Operation: "list branches", Err: err}
	}

	// Filter branches by prefix
	var topicBranches []string
	for _, branch := range branches {
		if strings.HasPrefix(branch, prefix) {
			// Remove the prefix to get the branch name
			name := strings.TrimPrefix(branch, prefix)
			topicBranches = append(topicBranches, name)
		}
	}

	// Print the branches
	if len(topicBranches) == 0 {
		fmt.Printf("No %s branches found\n", branchType)
		return nil
	}

	// Capitalize the first letter of the branch type
	branchTypeCapitalized := branchType
	if len(branchType) > 0 {
		branchTypeCapitalized = strings.ToUpper(branchType[:1]) + branchType[1:]
	}

	if !showWorktrees {
		fmt.Printf("%s branches:\n", branchTypeCapitalized)
		for _, branch := range topicBranches {
			fmt.Printf("  %s\n", branch)
		}
		return nil
	}

	// Everything the column needs is gathered BEFORE the first line is printed,
	// so a failed bulk read aborts instead of leaving half a table on screen.
	cells, err := worktreeCells(repo, prefix, topicBranches)
	if err != nil {
		return err
	}

	width := 0
	for _, branch := range topicBranches {
		if len(branch) > width {
			width = len(branch)
		}
	}

	fmt.Printf("%s branches:\n", branchTypeCapitalized)
	for _, branch := range topicBranches {
		fmt.Printf("  %-*s  %s\n", width, branch, cells[branch])
	}

	return nil
}

// worktreeCells returns the worktree column keyed by short branch name, "-" for
// every branch with no linked worktree.
//
// It resolves every row from BULK reads — one `git worktree list`, one marker
// read, and one `git status` per LIVE linked worktree of a listed branch. Neither
// repo.WorktreeForBranch nor worktree.IsManaged may be called here: each
// re-queries git, which would make listing cost one git process per branch.
//
// A failed bulk read aborts (SC-13). Degrading would render an active lie —
// "no branch has a worktree", or "every worktree is unmanaged", the second in the
// direction that decides what the cleanup commands may delete. Only the
// per-worktree status degrades, because there every other row is still true.
func worktreeCells(repo *git.Repo, prefix string, names []string) (map[string]string, error) {
	cells := make(map[string]string, len(names))
	for _, name := range names {
		cells[name] = "-"
	}

	entries, err := repo.ListWorktrees()
	if err != nil {
		return nil, &errors.GitError{Operation: "list worktrees", Err: err}
	}

	mainRoot := ""
	byBranch := make(map[string]git.WorktreeEntry, len(entries))
	for _, entry := range entries {
		if entry.Main {
			// The main worktree's porcelain record does carry its branch, but
			// this column reports LINKED worktrees, so a branch checked out
			// there keeps its dash (SC-4). Its path is the base every other path
			// is rendered against, taken from this same output so the two sides
			// cannot disagree about symlinks.
			mainRoot = entry.Path
			continue
		}
		if entry.Branch == "" {
			// A detached worktree has no branch, so no row can be keyed on it.
			continue
		}
		byBranch[entry.Branch] = entry
	}

	// Match the listed branches before the second bulk read, so a repository
	// whose worktrees all belong to other branch types costs nothing extra.
	matched := make(map[string]git.WorktreeEntry, len(names))
	for _, name := range names {
		if entry, ok := byBranch[prefix+name]; ok {
			matched[name] = entry
		}
	}
	if len(matched) == 0 {
		return cells, nil
	}

	markers, err := worktree.ListMarkers(repo)
	if err != nil {
		return nil, &errors.GitError{Operation: "list worktree provenance markers", Err: err}
	}
	managed := make(map[string]bool, len(markers))
	for _, branch := range markers {
		managed[branch] = true
	}

	for name, entry := range matched {
		cell := worktree.RelativeDisplayPath(mainRoot, entry.Path)
		if present, statErr := worktreeIsPresent(entry.Path); statErr == nil && !present {
			// The row reports the recorded path as stale rather than guessing at
			// a count (SC-5). Staleness is decided here and never from the
			// porcelain 'prunable' field, which needs Git 2.36+.
			//
			// worktreeIsPresent rather than a bare stat: it also answers "a file
			// sits at that path" and "the directory has no .git", both of which
			// are just as unusable as an absent directory, and it keeps a stat
			// failure that is NOT absence — permission denied, I/O — off this
			// branch. Such a failure falls through to the status attempt below,
			// which fails on the same condition and degrades with a warning
			// instead of claiming the directory is gone (SC-9).
			cell += " " + missingTag
		} else if count, err := repo.WorktreeChangeCount(entry.Path); err != nil {
			// One broken worktree must not make the whole listing useless, so
			// the row degrades to its bare path and warns (SC-9).
			fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
		} else if count > 0 {
			cell += fmt.Sprintf(" [%d]", count)
		}
		if !managed[entry.Branch] {
			cell += " " + unmanagedTag
		}
		cells[name] = cell
	}

	return cells, nil
}
