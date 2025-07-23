# git-flow-next Technical Overview

git-flow-next is a modern, flexible implementation of Git workflow management that builds upon the original git-flow concepts while providing extensive customization capabilities for modern development practices.

## Project Structure

The project follows Go best practices with clear separation of concerns:

```
git-flow-next/
├── cmd/                    # Command implementations
│   ├── root.go            # Root CLI command setup with Cobra
│   ├── init.go            # Repository initialization command
│   ├── start.go           # Branch starting logic  
│   ├── finish.go          # Branch finishing logic (most complex)
│   ├── topicbranch.go     # Dynamic command registration for branch types
│   ├── list.go            # Branch listing commands
│   ├── checkout.go        # Branch checkout functionality
│   ├── delete.go          # Branch deletion
│   ├── rename.go          # Branch renaming
│   ├── update.go          # Branch updating from parent
│   └── overview.go        # Repository overview/status
├── internal/              # Internal packages (not exported)
│   ├── config/           # Git configuration management
│   │   └── config.go     # Branch type definitions, config loading
│   ├── git/              # Git command wrapper
│   │   └── repo.go       # Git operations with error handling
│   ├── mergestate/       # Merge conflict state persistence
│   │   └── mergestate.go # State management for multi-step operations
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
git flow topic start feature my-feature
git flow topic start hotfix critical-fix  
git flow topic start release v1.0
```

All topic branches use the same `start` and `finish` commands, with behavior determined by configuration rather than branch type.

### Configurable Behavior

Topic branch behavior is defined through Git configuration, allowing complete customization of:

- **Parent branch**: Which base branch to branch from
- **Start point**: Where to create the branch (can differ from parent)
- **Merge strategies**: How changes flow upstream and downstream
- **Tag creation**: Whether to create tags when finishing
- **Child branch updates**: Automatic updating of child base branches after finish

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

#### Tag Configuration

| Branch Type | Default Tagging | Tag Prefix | When Tagged |
|-------------|-----------------|------------|-------------|
| Feature | ❌ Disabled | (none) | Never by default |
| Release | ✅ Enabled | (none) | On finish |
| Hotfix | ✅ Enabled | (none) | On finish |

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
git config gitflow.feature.finish.merge merge
git config gitflow.release.finish.merge merge
git config gitflow.hotfix.finish.merge merge

# Merge strategies (downstream - update operations)
git config gitflow.feature.downstreamStrategy rebase
git config gitflow.release.downstreamStrategy merge
git config gitflow.hotfix.downstreamStrategy rebase

# Tag settings
git config gitflow.feature.finish.notag true
git config gitflow.release.finish.notag false
git config gitflow.hotfix.finish.notag false
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
    downstreamStrategy = squash-merge
```

### Configurable Properties

#### For Base Branches:
- **parent**: The parent base branch for dependency tracking
- **upstreamStrategy**: How to merge changes to parent (`merge`, `rebase`, `none`)
- **downstreamStrategy**: How to receive changes from parent (`merge`, `rebase`, `squash`, `none`)

#### For Topic Branch Types (using gitflow.branch.* keys):
- **parent**: Default parent base branch
- **startPoint**: Branch to create from (can differ from parent)
- **upstreamStrategy**: How to merge back to parent
- **downstreamStrategy**: How to receive updates from parent
- **tag**: Whether to create tags when finishing
- **tagPrefix**: Prefix for created tags

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
git flow topic finish hotfix security-patch

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
git flow topic start <type> <name>
git flow topic finish <type> <name>
git flow topic list <type>

# Base branch operations  
git flow merge-upstream <branch>  # or: git flow up <branch>
git flow update <branch>

# Status and overview
git flow status
git flow overview
```

### Command Aliases

For compatibility, traditional commands are aliased:

```bash
git flow feature start <name>    # → git flow topic start feature <name>
git flow hotfix finish <name>    # → git flow topic finish hotfix <name>
git flow release list            # → git flow topic list release
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

#### Key Components

**State Management**: Uses `mergestate.MergeState` to persist operation state across command invocations. This enables the `--continue` and `--abort` functionality.

**Step Handlers**: Each step has a dedicated handler function:
- `handleContinue()`: Orchestrates step progression
- `handleCreateTagStep()`: Manages tag creation with full configuration support
- `handleUpdateChildrenStep()`: Updates dependent child branches
- `handleDeleteBranchStep()`: Handles branch cleanup based on retention settings

**Configuration Integration**: The command respects all branch-specific configuration settings for merge strategies, tag creation, and branch retention.

#### Main Execution Flow

1. **Initialization**: Load configuration, validate inputs, resolve branch names
2. **State Detection**: Determine if this is a new operation, continuation, or abort
3. **Step Execution**: Execute current step with conflict detection
4. **State Persistence**: Save progress and move to next step
5. **Child Updates**: Automatically update dependent branches
6. **Cleanup**: Remove branches and clear state

#### Conflict Handling

```go
if strings.Contains(mergeErr.Error(), "conflict") {
    state.CurrentStep = stepMerge
    mergestate.SaveMergeState(state)
    return &errors.UnresolvedConflictsError{}
}
```

When conflicts occur:
- Current state is saved with the problematic step
- User receives clear instructions for resolution
- Operation can be resumed with `--continue` or cancelled with `--abort`

#### Child Branch Cascading

One of the most powerful features is automatic child branch updating:

```bash
# Finishing a hotfix into main automatically updates develop
git flow topic finish hotfix security-patch

# The system will:
# 1. Merge hotfix/security-patch into main  
# 2. Automatically update develop from main (if configured)
# 3. Update any other child branches of main
```

This ensures consistency across the branch hierarchy without manual intervention.

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
# Import existing configuration
git flow init --import-avh

# Migrate to new configuration format
git flow config migrate
```

## Extensibility

The unified topic branch implementation and configuration-driven approach make git-flow-next highly extensible:

- Add custom branch types through configuration
- Define organization-specific workflow templates
- Create custom merge strategies through hooks
- Extend functionality through plugin architecture

This technical foundation enables teams to implement any branching strategy while maintaining the automation and convenience that made git-flow popular.