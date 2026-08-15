package cmd

import (
	"fmt"
	"os"

	"github.com/gittower/git-flow-next/internal/config"
	"github.com/gittower/git-flow-next/internal/errors"
	"github.com/gittower/git-flow-next/internal/git"
	"github.com/gittower/git-flow-next/internal/hooks"
	"github.com/gittower/git-flow-next/internal/navigate"
)

// StartOptions carries the flags of 'git flow <type> start'.
type StartOptions struct {
	// ShouldFetch overrides the resolved fetch behavior; nil means "use config".
	ShouldFetch *bool
	// Worktree overrides the branch type's worktree default; nil means "use config".
	Worktree *bool
	// WorktreePath overrides the computed worktree path. A non-empty value
	// implies creation, which the caller folds into Worktree.
	WorktreePath string
	// NoCD suppresses the navigation-destination write, and nothing else.
	NoCD bool
	// Quiet suppresses the shell-init tip.
	Quiet bool
}

// StartCommand is the implementation of the start command for topic branches
// If opts.ShouldFetch is nil, the function will check config for fetch preference
// If base is empty, the function will use the configured starting point
func StartCommand(branchType string, name string, base string, opts StartOptions) {
	repo := mustOpenRepo()
	if err := start(repo, branchType, name, base, opts); err != nil {
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

// start performs the actual branch creation logic with optional fetch and returns any errors
func start(repo *git.Repo, branchType string, name string, base string, opts StartOptions) error {
	// Validate that git-flow is initialized
	initialized, err := config.IsInitialized(repo)
	if err != nil {
		return &errors.GitError{Operation: "check if git-flow is initialized", Err: err}
	}
	if !initialized {
		return &errors.NotInitializedError{}
	}

	// Apply version filter for any branch type. The filter script
	// (filter-flow-{branchType}-start-version) decides what to do; when no name
	// was provided, it runs with an empty version argument and may supply one.
	filteredName, err := hooks.RunVersionFilter(repo, branchType, name)
	if err != nil {
		return &errors.GitError{Operation: "run version filter", Err: err}
	}
	if filteredName != name {
		if name == "" {
			fmt.Printf("Version filter derived name '%s'\n", filteredName)
		} else {
			fmt.Printf("Version filter changed '%s' to '%s'\n", name, filteredName)
		}
		name = filteredName
	}

	// Fall back to the empty-name error only when neither an explicit name nor a
	// version filter supplied one.
	if name == "" {
		return &errors.EmptyBranchNameError{}
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

	// Get full branch name
	fullBranchName := branchConfig.Prefix + name

	// Get start point
	startPoint := branchConfig.Parent
	if branchConfig.StartPoint != "" {
		// If start point is specified in config, use it instead of parent
		startPoint = branchConfig.StartPoint
	}
	if base != "" {
		// If base argument is provided, it overrides the configured starting point
		startPoint = base
	}

	// Validated here, before the hook wrapper: a command that cannot succeed must
	// run neither the fetch nor the pre-start hook, so "nothing is created" also
	// means nothing ran. It has to stay after the version filter above, which can
	// change the branch name and therefore the computed path.
	createWorktree := config.ResolveStartShouldCreateWorktree(cfg, branchType, opts.Worktree)
	if createWorktree {
		if err := validateWorktreeTarget(cfg, repo, fullBranchName, opts.WorktreePath); err != nil {
			return err
		}
	}

	// Build hook context
	hookCtx := hooks.HookContext{
		BranchType: branchType,
		BranchName: name,
		FullBranch: fullBranchName,
		BaseBranch: startPoint,
		Origin:     cfg.Remote,
	}
	// Set version for branches configured with tagging
	if branchConfig.Tag {
		hookCtx.Version = name
	}

	// Run start operation wrapped with hooks
	return hooks.WithHooks(repo, branchType, hooks.HookActionStart, hookCtx, func() error {
		return executeStart(repo, branchType, name, base, opts, createWorktree, cfg, branchConfig, fullBranchName, startPoint)
	})
}

// validateWorktreeTarget refuses, before anything is created, the two conditions
// that would otherwise leave a half-created state behind.
func validateWorktreeTarget(cfg *config.Config, repo *git.Repo, branch string, pathFlag string) error {
	// A repository with no commits has exactly one branch and it is necessarily
	// checked out, so there is nothing a second worktree could hold. CreateBranch
	// would also RENAME that branch rather than create one, which no worktree
	// could follow.
	// HasCommits reports an unverifiable HEAD as (false, nil) and has no other
	// failure mode, so there is no error path to handle here.
	hasCommits, _ := repo.HasCommits()
	if !hasCommits {
		return &errors.InvalidInputError{Message: "cannot create a worktree in a repository with no commits; commit first, then start the branch"}
	}

	target, err := resolveWorktreeTarget(cfg, repo, branch, pathFlag)
	if err != nil {
		return err
	}
	occupied, err := pathIsOccupied(target)
	if err != nil {
		return &errors.GitError{Operation: "inspect target path", Err: err}
	}
	if occupied {
		return &errors.WorktreePathOccupiedError{Path: target}
	}
	return nil
}

// executeStart performs the actual start operation (called within hooks wrapper)
func executeStart(repo *git.Repo, branchType string, name string, base string, opts StartOptions, createWorktree bool, cfg *config.Config, branchConfig config.BranchConfig, fullBranchName string, startPoint string) error {
	// Determine if we should fetch using the shared resolver (default true for start):
	// Layer 1 default -> gitflow.<type>.start.fetch config -> CLI flag.
	remoteName := cfg.Remote
	if config.ResolveStartShouldFetch(cfg, branchType, opts.ShouldFetch) {
		// Skip silently when no remote is configured (no "Fetching" line, no error).
		if repo.RemoteExists(remoteName) {
			fmt.Printf("Fetching from %s...\n", remoteName)
			// Non-fatal: a failed fetch is a warning; start has no sync gate.
			if err := repo.Fetch(remoteName); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
			}
		}
	}

	// Check if branch already exists
	if err := repo.BranchExists(fullBranchName); err == nil {
		return &errors.BranchExistsError{BranchName: fullBranchName}
	}

	// Check if start point exists (can be branch, tag, or commit)
	if err := repo.BranchOrCommitExists(startPoint); err != nil {
		return &errors.BranchNotFoundError{BranchName: startPoint}
	}

	// Create branch. With a worktree the branch must NOT be checked out here —
	// git allows a branch in only one worktree, and the new worktree is about to
	// take it.
	var err error
	if createWorktree {
		err = repo.CreateBranchNoCheckout(fullBranchName, startPoint)
	} else {
		err = repo.CreateBranch(fullBranchName, startPoint)
	}
	if err != nil {
		return &errors.GitError{Operation: "create branch", Err: err}
	}

	// Store the start point in Git config
	if err := repo.SetBaseBranch(fullBranchName, startPoint); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to store base branch: %v\n", err)
	}

	fmt.Printf("Created branch '%s' from '%s'\n", fullBranchName, startPoint)

	if createWorktree {
		target, err := createWorktreeAt(cfg, repo, fullBranchName, opts.WorktreePath, false)
		if err != nil {
			// The branch and its base config already exist. They are left alone:
			// deleting a branch on an error path risks destroying more than it
			// repairs, and the user can retry with 'git flow worktree add'.
			return err
		}
		fmt.Printf("Created worktree for branch '%s' at %s\n", fullBranchName, target)
		fmt.Printf("To switch to it: cd %s\n", target)
		printShellInitTip(opts.Quiet)

		if !opts.NoCD {
			// Written only now, after the worktree exists.
			if err := navigate.WriteDestination(target); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
			}
		}
	}
	return nil
}
