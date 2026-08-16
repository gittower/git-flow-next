# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- `checkout` is worktree-aware: when the branch has a worktree, it reports the path and offers it to the calling shell instead of switching the current worktree's branch. `--worktree` creates a missing worktree first and records git-flow as its creator, `--force` clears a plain directory out of the way, `--no-cd` suppresses only the shell handover, and `--quiet` drops the shell-init tip
- `shell-init {bash|zsh|fish}` prints a shell wrapper that turns that handover into an automatic `cd`. It defines both a `git` and a `git-flow` function, so the documented `git flow …` form navigates too, and it scopes the navigation variable to a single command rather than exporting it
- `worktree add` now points at `shell-init` when the navigation channel is unused, and gains `--quiet` to suppress that tip
- `<type> start --worktree` creates the new branch in its own worktree instead of checking it out, records git-flow as its creator, and offers the path to the calling shell. `--worktree-path` puts it somewhere else and implies `--worktree`, `--no-worktree` opts out, `--no-cd` suppresses only the shell handover, and `--quiet` drops the shell-init tip. Combining the three is not an error: the one that appears last on the command line wins, which is specific to these flags — every other `--x`/`--no-x` pair in git-flow still prefers the positive flag whatever the order
- `gitflow.branch.<type>.worktree` makes worktree creation the default for a topic branch type, so `start` needs no flag. `init` writes the key for topic branch types only
- `<type> list --worktrees` appends a column reporting each branch's linked worktree: its path relative to the main worktree root, `[n]` for the number of changed entries, `(unmanaged)` for a worktree git-flow did not create, `(missing)` for one whose directory is gone, and `-` for a branch with no linked worktree — including a branch checked out in the main worktree, since the column reports *linked* worktrees. The count is of `git status --porcelain` entries rather than files, so an untracked directory counts once however many files it holds. Without the flag the output is unchanged
- `finish --ff-only` (and `gitflow.<type>.finish.ff-only`) makes a fast-forward into the parent a precondition rather than a strategy: if the parent carries any commit the topic branch does not — a true divergence, or merely being ahead — finish aborts before touching any local branch, tag or the working tree, so what lands on the parent is exactly the tested topic tip. It is rejected in combination with `--ff`, `--no-ff`, or a squash strategy, it suppresses the rebase of a rebase strategy rather than rewriting the topic branch to make it land, and it constrains the upstream merge only, not the automatic child updates

### Changed

- `start` no longer checks the new branch out when it creates a worktree for it: Git allows a branch in only one worktree at a time, so the invocation worktree stays where it was. Without a worktree, `start` checks the branch out exactly as before

### Fixed

- The EXIT STATUS section of the `git flow <type> checkout` manpage documented codes 1/2/3/4 that the command has never returned; it now documents the actual 1/2/3/5/6
- `finish` collected the auto-update child base branches in Go's randomized map order, so with more than one auto-update child the order they were updated in, the order they were reported in, and the order recorded in merge state (the resume order after a conflict) all varied between identical runs; children are now processed in sorted order, matching `integrate`
- `finish` left you on the parent branch, so a `release` or `hotfix` finish ended on `main` instead of on `develop` as git-flow-avh does — even under `--keep`, where nothing is deleted and the checkout served no purpose. It now ends on the integration branch: the last auto-update child of the parent, or the parent itself when it has none. The branch is derived from the configured topology rather than a hardcoded name, so it works for custom branch names too, and finish reports the branch it leaves you on
- The `git flow <type> list` manpage documented a `[pattern]` argument with shell-style globbing, a `*` current-branch marker and remote tracking information that the command has never had, plus a trailing period on the empty-result message and exit codes it has never returned; it now documents what the command actually does and the codes 0/1/2/3 it actually returns
- Worktree path comparison ignores case on Windows, where two spellings of one location differing only in case were treated as different paths. The refusal to remove or detach the main worktree, the refusal to delete a registered worktree, and the detection that the shell is standing inside a worktree being removed all failed to fire on a case mismatch

## [2.0.0] - 2026-08-09

### Added

- Shared, committable configuration: `init --shared` writes the git-flow setup to a `.gitflow` file at the repository root and copies it into local config, so a team's branch model can live in the repository instead of every clone
- `--shared` on every `config add`, `edit`, `rename`, and `delete` verb edits the shared `.gitflow` file and re-syncs local config
- `config sync` copies `.gitflow` into local config, and `config status` reports drift between the two (exit code 6 when they differ)
- A fresh clone carrying a `.gitflow` is offered activation on first run — prompted when interactive, applied automatically with `gitflow.shared.autoInit`, and topic branch types declared only in the shared file get working commands before activation
- Shell tab completion for both the `git-flow` and `git flow` invocations, covering bash, zsh, and fish
- `init --init` creates the repository when run outside one, with an interactive prompt offering the same when the flag is absent; the new repository's initial branch is the resolved git-flow trunk rather than `init.defaultBranch`

### Changed

- **Breaking**: `git flow version` now prints `2.0.0 (git-flow-next)` instead of `git-flow-next version 2.0.0`, so the first whitespace-separated token is a bare version number as tooling written against git-flow-avh expects. Scripts matching the old string must be updated
- Release archives now contain a plain `git-flow` / `git-flow.exe` binary instead of a version- and platform-suffixed filename, so it works as the `git flow` subcommand without renaming after extraction

### Fixed

- Boolean git-config values now follow git-config(1) rules: `yes`, `on`, and non-zero integers are truthy and matching is case-insensitive. Previously only the literal `true` counted, so a setting like `git config gitflow.release.finish.push yes` was silently a no-op
- `config edit` no longer resets boolean settings that were not passed, so editing one field can no longer silently clear an unrelated one such as `autoUpdate`
- Interactive `init` now stores entered branch prefixes verbatim instead of appending a slash, so answering `feature_` produces `feature_login` rather than `feature_/login`
- Hooks and filters run from a linked worktree now resolve a relative `gitflow.path.hooks` or `core.hooksPath` against that worktree and execute with it as their working directory, instead of pointing at the main checkout
- Commands run outside a git repository now report a git error suggesting `git init` (exit code 3) instead of a git-flow-not-initialized error steering to `git flow init`
- Adding a base branch now rolls back its saved configuration when creating the git branch fails, leaving no orphaned config and keeping the command safe to retry
- The `config` command group is no longer preempted by first-run shared-config activation, including its nested `add`, `edit`, `rename`, and `delete` subcommands
- `--shared` edits now warn when an untrusted `gitflow.path.hooks` is withheld from local config, matching what `config sync` already reported
- Declining the "create a repository?" prompt now reports a decline instead of an internal probe failure
- git-flow-next now builds on every Go target OS; terminal detection previously had no implementation for aix, solaris, illumos, and plan9

## [1.2.0] - 2026-08-01

### Added

- `git flow integrate` command to merge changes from a parent branch into a topic branch, with options resolved from an `integrate` config namespace
- `finish` now pushes the finished branches and release tag to the remote after merging, honoring push options
- `update --continue` and `update --abort` to resume or cancel an update interrupted by conflicts
- `finish` now detects and auto-clears stale merge state left by a previous interrupted operation
- `delete` now fetches and runs a remote sync-check before deleting a branch (`--fetch`, `gitflow.<type>.delete.fetch`)
- `start` can derive the branch name from a configured version filter
- Windows arm64 build target

### Changed

- `finish` now runs a stricter pre-merge sync check: it aborts when the parent (merge-target) branch or the topic branch is behind or has diverged from its remote (with a diverged-specific message), while tolerating either being ahead; `--force` skips the check
- `finish` now treats a fetch failure against a reachable-but-failing remote as fatal (was a silent note); the error names the cause and suggests `--no-fetch` / `--force`
- `start` now fetches by default before creating a branch (skipped silently when no remote is configured); disable with `--no-fetch` or `gitflow.<type>.start.fetch false`. Shares the unified fetch resolution (default → config → flag) with `finish`
- Commands now refuse to act on merge state owned by a foreign operation and on structurally-incomplete state, returning exit code 3 instead of proceeding destructively

### Fixed

- `init` now fails fast when the git user identity is missing
- Branch names are validated with `git check-ref-format`, and dots are now allowed in base branch names
- Branch names are resolved case-insensitively while preserving their original case
- Leaf commands now reject unexpected positional arguments
- `delete`, `rename`, and `finish` are gated when git-flow is uninitialized
- Base-branch config cleanup is scoped to local config and treats a missing key as a no-op
- `finish --abort` is a no-op when no merge is in progress
- Hooks and filters now run on Windows (executed via `sh`)

## [1.1.0] - 2026-04-06

### Added

- Push-option support for publish command (`--push-option` / `-o`)
- Support for configurable hooks directory via `gitflow.path.hooks` and `core.hooksPath`

### Fixed

- Validate remote exists before any state-changing delete operations
- Validate remote before remote operations (publish, track, finish sync)
- Check if remote exists before attempting to delete remote branch
- Skip fetch when no remote is configured
- Use empty commit instead of README.md when initializing empty repositories
- Default to empty version tag prefix during init
- Thread config correctly into tag creation step during finish

## [1.0.0] - 2026-02-08

### Added

- `--no-verify` option for finish command to skip git hooks during merge
- `--merge-message` and `--update-message` options for finish command with placeholder support
- Git config support for merge message options (`gitflow.<type>.finish.mergeMessage`, `gitflow.<type>.finish.updateMessage`)
- Configuration scope flags for init command (`--local`, `--global`, `--system`, `--file`)
- `--force` option for init command to allow reconfiguration
- Remote sync check before finish to ensure local branch is up-to-date with remote

### Changed

- Default fetch behavior changed to `true` for finish command

### Fixed

- Hooks now receive correct positional arguments

## [0.3.0] - 2026-01-14

### Added

- Add hooks and filters system for customizing git-flow operations
  - Pre/post hooks for start, finish, publish, track, delete, and update actions
  - Version filters for topic branches on start
  - Tag message filters for topic branches that create tags on finish
- Add `publish` command for topic branches to push branches to remote
- Add `--squash-message` option for custom squash commit messages on finish
- Add config support for force deletion (`gitflow.branch.<type>.forceDelete`)

### Fixed

- Fix worktree support for hooks, filters, and merge state
- Fix shorthand `git flow publish` command (was returning "not implemented")
- Fix `git flow <type> finish` to allow optional branch name (uses current branch)

## [0.2.0] - 2026-01-11

### Added

- Add `track` command for topic branches to track existing remote branches
- Add `--fetch` option for topic branch finish command

### Fixed

- Create base branches in dependency order during init
- Fix docs link in README.md

### Changed

- Optimize binary size (~50% reduction)

## [0.1.1] - 2025-09-24

### Fixed

- Minor bug fixes and improvements

## [0.1.0] - 2025-09-16

### Added

- Initial release of git-flow-next
- Support for feature, release, hotfix, and support branch workflows
- Fully customizable base and topic branches with configurable prefixes and relationships
- Configurable merge strategies: merge, rebase, or squash when finishing branches
- Flexible configuration via git config or command-line flags
- Conflict recovery: resolve conflicts and continue where you left off
- Automatic updates to child branches (e.g., develop syncs from main)
- Compatibility with existing git-flow-avh repositories

[Unreleased]: https://github.com/gittower/git-flow-next/compare/v2.0.0...HEAD
[2.0.0]: https://github.com/gittower/git-flow-next/compare/v1.2.0...v2.0.0
[1.2.0]: https://github.com/gittower/git-flow-next/compare/v1.1.0...v1.2.0
[1.1.0]: https://github.com/gittower/git-flow-next/compare/v1.0.0...v1.1.0
[1.0.0]: https://github.com/gittower/git-flow-next/compare/v0.3.0...v1.0.0
[0.3.0]: https://github.com/gittower/git-flow-next/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/gittower/git-flow-next/compare/v0.1.1...v0.2.0
[0.1.1]: https://github.com/gittower/git-flow-next/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/gittower/git-flow-next/releases/tag/v0.1.0
