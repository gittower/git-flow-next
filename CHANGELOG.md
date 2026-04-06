# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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

[Unreleased]: https://github.com/gittower/git-flow-next/compare/v1.1.0...HEAD
[1.1.0]: https://github.com/gittower/git-flow-next/compare/v1.0.0...v1.1.0
[1.0.0]: https://github.com/gittower/git-flow-next/compare/v0.3.0...v1.0.0
[0.3.0]: https://github.com/gittower/git-flow-next/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/gittower/git-flow-next/compare/v0.1.1...v0.2.0
[0.1.1]: https://github.com/gittower/git-flow-next/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/gittower/git-flow-next/releases/tag/v0.1.0
