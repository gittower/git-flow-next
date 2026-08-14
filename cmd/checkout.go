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
)

// CheckoutOptions carries the flags of 'git flow <type> checkout'. They are
// grouped because they travel together through one call; individually they are
// unrelated switches.
type CheckoutOptions struct {
	// Worktree creates the branch's worktree when it does not exist yet.
	Worktree bool
	// NoCD suppresses the navigation-destination write, and nothing else: the
	// branch switch and the printed path both still happen.
	NoCD bool
	// Clobber allows a plain directory in the way of a new worktree to be
	// removed. It only means anything alongside Worktree.
	Clobber bool
	// Quiet suppresses the shell-init tip.
	Quiet bool
	// ShowCommands echoes the git command before running it.
	ShowCommands bool
}

// CheckoutCommand handles checking out a topic branch
func CheckoutCommand(branchType string, nameOrPrefix string, opts CheckoutOptions) {
	repo := mustOpenRepo()
	if err := executeCheckout(repo, branchType, nameOrPrefix, opts); err != nil {
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

// executeCheckout performs the actual branch checkout logic and returns any errors
func executeCheckout(repo *git.Repo, branchType string, nameOrPrefix string, opts CheckoutOptions) error {
	// Load configuration
	cfg, err := config.Load(repo)
	if err != nil {
		return &errors.GitError{Operation: "load configuration", Err: err}
	}

	// Get branch configuration
	branchConfig, ok := cfg.Branches[branchType]
	if !ok {
		return &errors.InvalidBranchTypeError{BranchType: branchType}
	}

	// If no name/prefix provided, list available branches and return
	if nameOrPrefix == "" {
		branches, err := repo.ListBranches()
		if err != nil {
			return &errors.GitError{Operation: "list branches", Err: err}
		}

		prefix := branchConfig.Prefix
		found := false
		fmt.Printf("Available %s branches:\n", branchType)
		for _, branch := range branches {
			if strings.HasPrefix(branch, prefix) {
				found = true
				fmt.Printf("  %s\n", strings.TrimPrefix(branch, prefix))
			}
		}
		if !found {
			fmt.Printf("No %s branches exist.\n", branchType)
		}
		return nil
	}

	// Construct full branch name
	fullBranchName := nameOrPrefix
	if branchConfig.Prefix != "" {
		fullBranchName = branchConfig.Prefix + nameOrPrefix
	}

	// Check if branch exists
	err = repo.BranchExists(fullBranchName)
	if err != nil {
		// If exact match not found, try prefix match
		branches, err := repo.ListBranches()
		if err != nil {
			return &errors.GitError{Operation: "list branches", Err: err}
		}

		matches := []string{}
		prefix := branchConfig.Prefix + nameOrPrefix
		for _, branch := range branches {
			if strings.HasPrefix(branch, prefix) {
				matches = append(matches, branch)
			}
		}

		switch len(matches) {
		case 0:
			return &errors.BranchNotFoundError{BranchName: fullBranchName}
		case 1:
			fullBranchName = matches[0]
		default:
			return &errors.GitError{Operation: "checkout branch", Err: fmt.Errorf("ambiguous branch name '%s' matches multiple branches:\n  %s", nameOrPrefix, strings.Join(matches, "\n  "))}
		}
	}

	// The worktree lookup runs on the RESOLVED name, never on what the user
	// typed: 'checkout user' must find feature/user-auth's worktree.
	entry, err := repo.WorktreeForBranch(fullBranchName)
	if err != nil {
		return &errors.GitError{Operation: "look up worktree for branch", Err: err}
	}

	if entry != nil {
		// A registered worktree that is no longer there must never be handed to
		// the shell as a destination: git-flow would exit 0 while the shell
		// failed to enter it, or entered something unrelated.
		present, presentErr := worktreeIsPresent(entry.Path)
		if presentErr != nil {
			return &errors.GitError{Operation: "inspect the worktree directory", Err: presentErr}
		}
		if !present {
			return &errors.GitError{
				Operation: fmt.Sprintf("check out branch '%s'", fullBranchName),
				Err:       fmt.Errorf("its worktree at %s is gone; 'git flow worktree prune' drops stale entries", entry.Path),
			}
		}

		// Navigating to the worktree the command already runs in would replace
		// today's output with navigation lines and ask the shell to cd where it
		// already is. So the ordinary case — checking out the branch you are on
		// — stays exactly as it was, and only a genuine move navigates.
		if !git.SamePath(entry.Path, repo.WorkTree()) {
			return navigateToWorktree(fullBranchName, entry.Path, false, opts)
		}
	} else if opts.Worktree {
		// --clobber reaches createWorktreeAt only from here: with no creation
		// requested there is nothing to clobber, so the flag is ignored.
		target, err := createWorktreeAt(cfg, repo, fullBranchName, "", opts.Clobber)
		if err != nil {
			return err
		}
		return navigateToWorktree(fullBranchName, target, true, opts)
	}

	// Show git command if requested
	if opts.ShowCommands {
		fmt.Printf("$ git checkout %s\n", fullBranchName)
	}

	// Checkout the branch. This stays 'git checkout' rather than 'git switch',
	// which would raise the required Git version from 2.17 to 2.23.
	err = repo.Checkout(fullBranchName)
	if err != nil {
		return &errors.GitError{Operation: fmt.Sprintf("checkout branch '%s'", fullBranchName), Err: err}
	}

	fmt.Printf("Switched to branch '%s'\n", fullBranchName)
	return nil
}

// worktreeIsPresent reports whether the directory a worktree entry records is
// still that worktree.
//
// Existence alone is not enough. A user who deletes a worktree directory by hand
// and puts something else at that exact path leaves an entry pointing at a plain
// directory or a file, and handing either to the shell is the failure SC-7 exists
// to prevent: git-flow exits 0 while the shell either cannot enter the path at
// all or lands somewhere that is not the worktree.
//
// The .git entry settles it without a git subprocess: a linked worktree carries a
// .git FILE and the main worktree a .git DIRECTORY, so an Lstat that refuses only
// on absence accepts both — the same idiom clobberTarget uses.
func worktreeIsPresent(path string) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	// Checked before the join, so a regular file at path is answered here rather
	// than by an ENOTDIR that os.IsNotExist does not recognize.
	if !info.IsDir() {
		return false, nil
	}
	if _, err := os.Lstat(filepath.Join(path, ".git")); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// navigateToWorktree reports the branch's worktree and offers it to the calling
// shell. The branch is never checked out in the current worktree — git allows a
// branch in only one worktree at a time, and the destination already has it.
//
// The destination is written only after the worktree is known to exist, and a
// write failure warns rather than failing a command that otherwise succeeded:
// the user still has the path on screen.
func navigateToWorktree(branch string, path string, created bool, opts CheckoutOptions) error {
	if created {
		fmt.Printf("Created worktree for branch '%s' at %s\n", branch, path)
	} else {
		fmt.Printf("Worktree for branch '%s' at %s\n", branch, path)
	}
	fmt.Printf("To switch to it: cd %s\n", path)
	printShellInitTip(opts.Quiet)

	if !opts.NoCD {
		if err := navigate.WriteDestination(path); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
		}
	}
	return nil
}
