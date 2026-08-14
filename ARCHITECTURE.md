# git-flow-next Technical Overview

git-flow-next is a modern, flexible implementation of Git workflow management that builds upon the original git-flow concepts while providing extensive customization capabilities for modern development practices.

> **See also:** [CODE_REFERENCE.md](CODE_REFERENCE.md) for a quick codebase navigation guide with file paths, struct definitions, and implementation details.

## Project Structure

The project follows Go best practices with clear separation of concerns:

```
git-flow-next/
├── cmd/                    # Command implementations
│   ├── root.go            # Root CLI command setup with Cobra
│   ├── init.go            # Repository initialization command
│   ├── config.go          # config subcommands (add/edit/rename/delete/list)
│   ├── start.go           # Branch starting logic
│   ├── finish.go          # Branch finishing logic (most complex)
│   ├── update.go          # Branch updating from parent
│   ├── integrate.go       # Integrate a base branch into its parent
│   ├── topicbranch.go     # Dynamic command registration for branch types
│   ├── shorthand.go       # Shorthand commands (auto-detect branch type)
│   ├── continue_abort.go  # Continue/abort in-progress operations; foreign-op guard
│   ├── preflight.go       # Shared pre-merge fetch/sync preflight guard
│   ├── publish.go         # Publish a topic branch to the remote
│   ├── track.go           # Track a remote topic branch
│   ├── list.go            # Branch listing commands
│   ├── checkout.go        # Branch checkout functionality
│   ├── delete.go          # Branch deletion
│   ├── rename.go          # Branch renaming
│   ├── worktree.go        # Worktree management by branch name
│   └── overview.go        # Repository overview
├── internal/              # Internal packages (not exported)
│   ├── config/           # Git configuration management
│   │   └── config.go     # Branch type definitions, config loading
│   ├── git/              # Git command wrapper
│   │   ├── repo.go       # Git operations with error handling
│   │   └── worktree.go   # Worktree add/remove/list/prune/detach
│   ├── worktree/         # Worktree path templates and provenance markers
│   │   ├── path.go       # gitflow.worktreePath expansion
│   │   └── provenance.go # Markers recording which worktrees git-flow created
│   ├── navigate/         # GIT_FLOW_CD_FILE destination channel
│   │   └── cdfile.go     # Hand a destination directory to the calling shell
│   ├── mergestate/       # Merge conflict state persistence
│   │   └── mergestate.go # State management for multi-step operations
│   ├── hooks/            # Client-side hook execution
│   │   ├── hooks.go      # Hook runner and context construction
│   │   ├── filters.go    # Hook matching/filtering
│   │   └── types.go      # Hook type definitions
│   ├── errors/           # Custom error types and exit codes
│   │   └── errors.go     # Structured error handling
│   ├── util/             # Validation and utility functions
│   │   └── validation.go # Input validation helpers
│   └── update/           # Branch updating logic
│       └── update.go     # Shared update functionality
├── test/                  # Test files (mirrors source structure)
│   ├── cmd/              # Command-level integration tests
│   ├── internal/         # Internal package unit tests
│   └── testutil/         # Test utilities and Git repo helpers
├── scripts/              # Build and deployment scripts
│   └── build.sh          # Multi-platform build script
├── main.go               # Application entry point
└── [documentation files] # README.md, ARCHITECTURE.md, etc.
```

### Key Organizational Principles

- **cmd/**: Contains all CLI command implementations using the Cobra framework
- **internal/**: Private packages that handle core functionality (config, git operations, state management)
- **test/**: Mirrors the source structure with comprehensive test coverage
- **testutil/**: Shared testing utilities, especially Git repository helpers

## Core Architecture

### Branch Dependency Model

The foundation of git-flow-next is a **branch dependency model** that formalizes the parent-child relationship between branches. This model enables:

- Automatic tracking of dependencies between branches
- Intelligent synchronization of changes across branch hierarchies  
- Consistent propagation of changes through the dependency tree
- Visualization of branch relationships

Every branch (except root branches) has a parent, and branches can have multiple children. This simple paradigm enables powerful workflow customization.

### Branch Types

git-flow-next defines two fundamental branch types:

#### Base Branches (Long-living)
- Exist throughout the project lifecycle
- Serve as integration points for features and releases
- Examples: `main`, `develop`, `staging`, `production`
- Configured with parent-child relationships for change propagation

#### Topic Branches (Short-living)
- Created for specific purposes (features, hotfixes, releases)
- Always have a defined parent base branch
- Automatically cleaned up after completion
- Examples: `feature/login`, `hotfix/security-fix`, `release/v1.0`

## Single Topic Branch Implementation

### Unified Command Structure

Instead of separate commands for `feature`, `hotfix`, and `release` branches, git-flow-next implements a **single topic branch mechanism**:

```bash
# Traditional git-flow
git flow feature start my-feature
git flow hotfix start critical-fix
git flow release start v1.0

# git-flow-next unified approach
git flow feature start my-feature
git flow hotfix start critical-fix
git flow release start v1.0
```

All topic branches use the same `start` and `finish` commands, with behavior determined by configuration rather than branch type.

### Configurable Behavior

Topic branch behavior is defined through Git configuration at two levels. **Branch type configuration** (Layer 1) defines the branch type's identity and process characteristics:

- **Parent branch**: Which base branch to branch from (structural)
- **Start point**: Where to create the branch (structural)
- **Merge strategies**: How changes flow upstream and downstream (process)
- **Tag creation**: Whether the branch type produces tags on finish (process)
- **Child branch updates**: Automatic updating of child base branches after finish (process)

**Command-specific configuration** (Layer 2) then controls operational details like fetch behavior, tag signing, and branch retention. CLI flags (Layer 3) override everything for one-off situations.

## Configuration System

### Default Configuration Overview

git-flow-next provides sensible defaults that work for most teams while remaining fully customizable.

#### Branch Structure
```
main/master     ← Production releases
    ↓
develop         ← Integration branch (auto-updated from main)
    ↓
feature/        ← New features
release/        ← Release preparation  
hotfix/         ← Emergency fixes
```

#### Base Branches

| Branch | Type | Parent | Config Key | Auto-Update from Parent |
|--------|------|--------|------------|------------------------|
| `main` | base | (root) | `gitflow.branch.main` | None |
| `develop` | base | `main` | `gitflow.branch.develop` | ✅ Yes |

#### Topic Branches

| Branch Type | Prefix | Parent | Start Point | Config Key | Created by Default |
|-------------|--------|--------|-------------|------------|-------------------|
| Feature | `feature/` | `develop` | `develop` | `gitflow.branch.feature` | ✅ Yes |
| Release | `release/` | `main` | `develop` | `gitflow.branch.release` | ✅ Yes |
| Hotfix | `hotfix/` | `main` | `main` | `gitflow.branch.hotfix` | ✅ Yes |

#### Merge Strategies

**Upstream Strategy (Finish Operations)** - How topic branches merge into their parent:

| Branch Type | Default | Options | Target Branch |
|-------------|---------|---------|---------------|
| Feature | `merge` | `merge`, `rebase`, `squash` | → `develop` |
| Release | `merge` | `merge`, `rebase`, `squash` | → `main` |
| Hotfix | `merge` | `merge`, `rebase`, `squash` | → `main` |

**Downstream Strategy (Update Operations)** - How parent updates are pulled into topic branches:

| Branch Type | Default | Options | Source Branch |
|-------------|---------|---------|---------------|
| Feature | `rebase` | `merge`, `rebase` | ← `develop` |
| Release | `merge` | `merge`, `rebase` | ← `main` |
| Hotfix | `rebase` | `merge`, `rebase` | ← `main` |

**Note**: The `--rebase` flag can be used with the `update` command to override the configured strategy and force rebase behavior.

#### Tag Configuration

| Branch Type | Default Tagging | Tag Prefix | When Tagged |
|-------------|-----------------|------------|-------------|
| Feature | ❌ Disabled | (configurable) | Never by default |
| Bugfix | ❌ Disabled | (configurable) | Never by default |
| Release | ✅ Enabled | v (during init) | On finish |
| Hotfix | ✅ Enabled | v (during init) | On finish |
| Support | ❌ Disabled | (configurable) | Never by default |

#### Branch Retention (After Finish)

| Setting | Default | Description |
|---------|---------|-------------|
| Delete Local | ✅ Yes | Remove local branch after successful merge |
| Delete Remote | ✅ Yes | Remove remote branch after successful merge |
| Force Delete | ❌ No | Use safe delete (checks for unmerged commits) |

#### Core Configuration Commands

```bash
# Base branch names
git config gitflow.branch.main main
git config gitflow.branch.develop develop

# Base branch relationships
git config gitflow.branch.develop.parent main
git config gitflow.branch.develop.upstreamStrategy merge
git config gitflow.branch.develop.downstreamStrategy merge
git config gitflow.branch.develop.autoUpdate true

# Topic branch prefixes
git config gitflow.branch.feature.prefix feature/
git config gitflow.branch.release.prefix release/
git config gitflow.branch.hotfix.prefix hotfix/

# Branch relationships
git config gitflow.branch.feature.parent develop
git config gitflow.branch.release.parent main
git config gitflow.branch.hotfix.parent main

# Merge strategies (upstream - finish operations)
git config gitflow.branch.feature.upstreamStrategy merge
git config gitflow.branch.release.upstreamStrategy merge
git config gitflow.branch.hotfix.upstreamStrategy merge

# Merge strategies (downstream - update operations)
git config gitflow.branch.feature.downstreamStrategy rebase
git config gitflow.branch.release.downstreamStrategy merge
git config gitflow.branch.hotfix.downstreamStrategy rebase

# Tag settings
git config gitflow.feature.finish.notag true
git config gitflow.release.finish.notag false
git config gitflow.hotfix.finish.notag false

# Usage examples
git flow update feature/my-feature              # Uses configured strategy
git flow update feature/my-feature --rebase     # Forces rebase strategy
git flow rebase                                 # Shorthand for update --rebase
```

**Note**: Release and hotfix branches merge only into `main`, then `develop` is automatically updated from `main` to stay synchronized.

### Branch Configuration Structure

Base branches are configured with dependency relationships:

```ini
[gitflow "branch.main"]
    type = base
    parent = 
    upstreamStrategy = none
    downstreamStrategy = none

[gitflow "branch.develop"] 
    type = base
    parent = main
    upstreamStrategy = merge
    downstreamStrategy = merge
```

Topic branch types are configured with the same key format:

```ini
[gitflow "branch.feature"]
    type = topic
    parent = develop
    startPoint = develop
    upstreamStrategy = rebase
    downstreamStrategy = merge
```

### Configurable Properties (Layer 1 — Branch Type Definition)

These properties define the branch type's identity and process characteristics. They describe *what the branch type is*, not how individual commands behave.

#### For Base Branches:
- **parent**: The parent base branch for dependency tracking (structural)
- **upstreamStrategy**: How changes flow to parent (process)
- **downstreamStrategy**: How updates flow from parent (process)
- **autoUpdate**: Whether the branch receives updates automatically on finish (process)

#### For Topic Branch Types (using gitflow.branch.* keys):
- **parent**: Default parent base branch (structural)
- **startPoint**: Branch to create from — can differ from parent (structural)
- **prefix**: Branch name prefix (structural)
- **upstreamStrategy**: How to merge back to parent on finish (process)
- **downstreamStrategy**: How to receive updates from parent (process)
- **tag**: Whether the branch type produces tags on finish (process)
- **tagPrefix**: Prefix for created tags (process)

### Merge Strategies

git-flow-next supports multiple merge strategies:

- **merge**: Standard Git merge with merge commit
- **rebase**: Rebase changes onto target branch
- **squash**: Squash all commits into single commit
- **none**: No automatic merging

## Example Workflow Configurations

### 1. Simple GitHub Flow

Perfect for continuous deployment with hotfix capability:

```ini
[gitflow "branch.main"]
    type = base
    parent = 
    upstreamStrategy = none
    downstreamStrategy = none

[gitflow "branch.feature"]
    type = topic
    parent = main
    startPoint = main
    upstreamStrategy = rebase
    downstreamStrategy = rebase

[gitflow "branch.hotfix"]
    type = topic
    parent = main
    startPoint = main
    upstreamStrategy = merge
    downstreamStrategy = none
```

**Branch Structure:**
```
main
├── feature/user-interface
├── feature/api-integration
└── hotfix/security-patch
```

### 2. Traditional Git-Flow

Classic git-flow with develop branch and release management:

```ini
[gitflow "branch.main"]
    type = base
    parent = 
    upstreamStrategy = none
    downstreamStrategy = none

[gitflow "branch.develop"]
    type = base
    parent = main
    upstreamStrategy = merge
    downstreamStrategy = merge

[gitflow "branch.feature"]
    type = topic
    parent = develop
    startPoint = develop
    upstreamStrategy = rebase
    downstreamStrategy = merge

[gitflow "branch.release"]
    type = topic
    parent = main
    startPoint = develop
    upstreamStrategy = merge
    downstreamStrategy = none
    tag = true

[gitflow "branch.hotfix"]
    type = topic
    parent = main
    startPoint = main
    upstreamStrategy = merge
    downstreamStrategy = none
    tag = true
```

**Branch Structure:**
```
main
├── hotfix/critical-fix
├── release/v1.0
└── develop
     ├── feature/payment-gateway
     │    └── feature/card-processing
     ├── feature/user-authentication
     │    └── feature/two-factor-auth
```

### 3. Web Application Flow

Multi-environment deployment with staging and production:

```ini
[gitflow "branch.production"]
    type = base
    parent = 
    upstreamStrategy = none
    downstreamStrategy = none

[gitflow "branch.staging"]
    type = base
    parent = production
    upstreamStrategy = merge
    downstreamStrategy = merge

[gitflow "branch.main"]
    type = base
    parent = staging
    upstreamStrategy = merge
    downstreamStrategy = merge

[gitflow "branch.feature"]
    type = topic
    parent = main
    startPoint = main
    upstreamStrategy = rebase
    downstreamStrategy = rebase

[gitflow "branch.hotfix"]
    type = topic
    parent = production
    startPoint = production
    upstreamStrategy = merge
    downstreamStrategy = none
```

**Branch Structure:**
```
production
├── hotfix/urgent-fix
└── staging
    └── main
        ├── feature/new-feature
        └── feature/ui-improvement
```

## Advanced Features

### Automatic Branch Updates

Configure branches to automatically receive updates from their parent:

```ini
[gitflow "branch.develop"]
    parent = main
    autoUpdate = true
    downstreamStrategy = merge
```

When `autoUpdate` is enabled, finishing a topic branch into `main` automatically propagates changes to `develop`.

### Cascade Updates

Changes can cascade through multiple levels of the dependency tree:

1. Finish `hotfix/security-patch` into `production`
2. Changes automatically flow to `staging` (if configured)
3. Changes then flow to `main` (if configured)
4. Finally cascade to `develop` (if configured)

### Tag Creation

Automatic tag creation with configurable naming:

```ini
[gitflow "branch.release"]
    tag = true
    tagPrefix = "v"
```

### Child Branch Updates

When finishing a topic branch, git-flow-next automatically updates child base branches that depend on the target parent branch. This ensures consistency across the branch hierarchy:

```bash
# Finishing a hotfix into main automatically updates develop
git flow hotfix finish security-patch

# The system will:
# 1. Merge hotfix/security-patch into main  
# 2. Automatically update develop from main (if configured)
# 3. Update any other child branches of main
```

Configure automatic updates in base branch settings:

```ini
[gitflow "branch.develop"]
    parent = main
    autoUpdate = true
    downstreamStrategy = merge
```

## Command Structure

### Core Commands

```bash
# Initialize git-flow configuration
git flow init

# Topic branch operations
git flow <type> start <name>
git flow <type> finish <name>
git flow <type> list

# Base branch operations
git flow integrate <branch>        # Integrate a base branch into its parent
git flow update <branch>
git flow update <branch> --rebase  # Force rebase strategy

# Shorthand commands (auto-detect branch type)
git flow rebase                    # Shorthand for: git flow <type> update --rebase
git flow update                    # Shorthand for: git flow <type> update
git flow finish                    # Shorthand for: git flow <type> finish

# Worktree operations (addressed by full branch name)
git flow worktree add <branch>
git flow worktree remove <branch>
git flow worktree list
git flow worktree prune
git flow worktree path <branch>

# Overview
git flow overview
```

### Dynamically Generated Commands

Topic-branch commands are generated per branch type from configuration. Each
configured type (feature, release, hotfix, support, and any custom type) gets
the same set of subcommands:

```bash
git flow feature start <name>    # start a feature branch
git flow hotfix finish <name>    # finish a hotfix branch
git flow release list            # list release branches
```

### Shorthand Commands

git-flow-next provides convenient shorthand commands that automatically detect your current topic branch:

```bash
git flow rebase                   # → git flow <type> update --rebase
git flow update                   # → git flow <type> update
git flow finish                   # → git flow <type> finish
git flow delete                   # → git flow <type> delete
git flow rename <name>            # → git flow <type> rename <name>
```

## Command Implementation

### Command Structure Overview

Commands in git-flow-next are implemented using the Cobra CLI framework with a clear architectural pattern:

1. **Root Command** (`cmd/root.go`): Sets up the main CLI structure and global flags
2. **Dynamic Registration** (`cmd/topicbranch.go`): Automatically registers branch type commands based on configuration
3. **Individual Commands** (`cmd/*.go`): Each major operation has its own file with specific logic

All commands follow a consistent pattern: validate inputs, load configuration, execute Git operations, and handle errors gracefully.

### The Finish Command: A Deep Dive

The finish command (`cmd/finish.go`) is the most complex command in the system, demonstrating the sophisticated architecture used throughout git-flow-next.

#### Step-Based State Machine

The finish command uses a **step-based state machine** approach to handle complex multi-step operations that can be interrupted by merge conflicts:

```
Steps: merge → create_tag → update_children → delete_branch
```

This architecture allows the command to:
- Resume operations after conflict resolution
- Provide clear progress feedback
- Handle complex branching scenarios
- Maintain consistency across interruptions

State is persisted to disk (`mergestate.MergeState`) so the operation can resume after conflict resolution via `--continue` or be cancelled with `--abort`. Each step has a dedicated handler, and child branch updates respect individual downstream strategies (see [Advanced Features > Child Branch Updates](#child-branch-updates) above).

For implementation details—struct definitions, handler functions, and code examples—see [CODE_REFERENCE.md](CODE_REFERENCE.md#state-machine-finish-command).

## Integration Points

### Tower Integration

git-flow-next integrates seamlessly with [Tower](https://www.git-tower.com), providing graphical workflow management while using the same configuration system.

### CI/CD Integration

The flexible configuration system enables easy integration with modern CI/CD pipelines by supporting:

- Webhook-triggered deployments based on branch patterns
- Environment-specific deployment strategies
- Automatic tag-based releases

## Migration from git-flow-avh

git-flow-next maintains compatibility with existing git-flow-avh configurations while providing migration tools for enhanced features:

```bash
# Existing git-flow-avh configuration is imported automatically on init
# (when AVH config is present and no explicit options are given)
git flow init
```

## Extensibility

The unified topic branch implementation and configuration-driven approach make git-flow-next highly extensible:

- Add custom branch types through configuration
- Define organization-specific workflow templates
- Create custom merge strategies through hooks
- Extend functionality through plugin architecture

This technical foundation enables teams to implement any branching strategy while maintaining the automation and convenience that made git-flow popular.