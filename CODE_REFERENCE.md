# Git-Flow-Next Code Reference

Quick navigation guide to the codebase: file paths, key structs, configuration keys, and implementation flows. For conceptual design and architectural rationale, see [ARCHITECTURE.md](ARCHITECTURE.md).

## Command Invocation Flow

```
main.go → cmd.Execute() → Cobra Framework
    ↓
cmd/root.go (rootCmd)
    ↓
cmd/topicbranch.go (Dynamic Registration)
    ├─> Registers: feature, release, hotfix, support, etc.
    └─> Each gets: start, finish, list, update, delete, rename, checkout, publish, track
```

## Command Execution Steps

### 1. Entry & CLI Setup (`main.go:11` → `cmd/root.go:28`)
- Execute Cobra root command
- Handle global flags (e.g., `--verbose`)

### 2. Dynamic Command Registration (`cmd/topicbranch.go:14-42`)
- Load git-flow configuration on init
- Discover topic branch types from config (feature, release, hotfix, etc.)
- Dynamically register commands for each branch type

### 3. Command Parsing (`cmd/topicbranch.go:69-98`)
- Parse subcommand (start, finish, list, update, delete, etc.)
- Extract flags and arguments
- Convert flag pairs (e.g., `--tag`/`--notag`) into option structs

### 4. Configuration Resolution - 3-Layer Precedence (`internal/config/resolver.go`)
```
Layer 1: Branch Type Definition    ← Layer 2: Command Config  ← Layer 3: CLI Flags
gitflow.branch.<type>.*               gitflow.<type>.<cmd>.*     --flag overrides
(identity & process characteristics)  (operational settings)     (highest priority)
```
Layer 1 defines *what the branch type is* (structural role + process characteristics like tagging).
Layer 2 controls *how commands execute* (operational knobs). Not all options have a Layer 1 equivalent.

### 5. Validation (`cmd/start.go:31-54`)
- Check git-flow initialization: `config.IsInitialized()`
- Validate inputs (branch name, branch type exists)
- Load full configuration: `config.LoadConfig()`

### 6. Git Operations (`cmd/start.go:68-112`)
- Optional: Fetch from remote (based on config/flags)
- Check branch existence/conflicts
- Execute git commands via `internal/git/repo.go`
- Store metadata (e.g., base branch in git config)

### 7. Error Handling (`cmd/start.go:16-25`)
- Custom error types from `internal/errors`
- Map to specific exit codes
- Return to user with context

**Key Pattern**: All commands follow this same flow—only the git operations in step 6 differ.

---

## Key Internal Modules

### `internal/config/` - Configuration Management
**Purpose**: Manages git-flow configuration (branch types, prefixes, merge strategies)

**Key files:**
- `config.go` - Main Config struct, Load/Save, branch types, preset definitions (Classic, GitHub, GitLab), AVH import
- `resolver.go` - 3-layer option resolution for finish command

**Key responsibilities:**
- Load/save configuration from Git config (`gitflow.*` keys)
- Define branch types (base vs. topic) and their relationships
- Import git-flow-avh compatibility
- Provide default configurations and presets
- **Three-layer precedence resolver**: Branch type definition (identity & process characteristics) → Command config (operational settings) → CLI flags; many options intentionally skip Layer 1 because they don't describe a branch type characteristic

**Used by**: All commands during initialization and option resolution

---

### `internal/git/` - Git Operations Wrapper
**Purpose**: Centralized Git command execution with error handling

**Key files:**
- `repo.go` - All Git commands (CreateBranch, Merge, Rebase, etc.)
- `config.go` - Direct git config access (GetConfig, SetConfig)

**Key operations:**
- Branch management: Create, delete, rename, checkout, list
- Merge operations: Merge, rebase, squash (with conflict detection)
- State checks: Branch existence, conflict detection, commit history
- Remote operations: Fetch, remote branch management
- Tag creation with signing support

**Used by**: All commands that interact with Git (executed after validation)

---

### `internal/errors/` - Custom Error Types
**Purpose**: Structured error handling with specific exit codes

**Error types:**
- `NotInitializedError` (exit 1) - git-flow not initialized
- `InvalidBranchTypeError` (exit 2) - Unknown branch type
- `GitError` (exit 3) - Generic Git operation failure
- `BranchExistsError` (exit 4) - Branch already exists
- `BranchNotFoundError` (exit 5) - Required branch missing
- Plus conflict-specific errors for merge/rebase states

**Used by**: All commands to provide consistent error reporting

---

### `internal/mergestate/` - Merge State Persistence
**Purpose**: Track multi-step operations (especially finish with conflicts)

**State tracked:**
- Current operation (finish) and step (merge, update_children, delete_branch)
- Branch being operated on and its parent
- Merge strategy in use
- Child branches pending updates
- Current child branch being processed

**Stored in**: `.git/gitflow/state/merge.json`

**Used by**: Finish command for `--continue`/`--abort` operations across conflicts

---

### `internal/hooks/` - Client-Side Hooks
**Purpose**: Run client-side hooks around git-flow operations (e.g. start/finish)

**Key files:**
- `hooks.go` - Hook runner and context construction
- `filters.go` - Match/filter which hooks apply to an operation
- `types.go` - Hook type definitions

**Used by**: Commands that fire hooks at defined points in their lifecycle

---

### `internal/util/` - Validation Utilities
**Purpose**: Input validation and sanitization

**Key functions:**
- Branch name validation (Git naming rules)
- Prefix validation (must end with `/`)
- Character restrictions (no `..`, spaces, `.lock`, etc.)

**Used by**: Commands during input validation phase

---

### `internal/update/` - Shared Update Logic
**Purpose**: Common branch update functionality

**Used by**: Update command and finish command (for child branch updates)

---

## Command Implementation

### Dynamic Commands (via `cmd/topicbranch.go`)
| File | Purpose |
|------|---------|
| `start.go` | Create new branch from parent/startPoint |
| `finish.go` | Merge branch back (state machine, conflict handling) |
| `list.go` | List branches of type |
| `update.go` | Pull changes from parent |
| `delete.go` | Remove branch |
| `rename.go` | Rename branch |
| `checkout.go` | Switch to branch |
| `publish.go` | Push branch to remote and set up tracking |
| `track.go` | Create a local branch tracking a remote topic branch |

### Static Commands
| File | Purpose |
|------|---------|
| `init.go` | Initialize git-flow |
| `config.go` | Manage configuration |
| `overview.go` | Show git-flow status |
| `worktree.go` | Manage worktrees by branch name (add/remove/list/prune/path) |
| `version.go` | Display version |
| `shorthand.go` | Quick commands (work on current branch) |

---

## State Machine: Finish Command

The finish command uses a step-based state machine for complex multi-step operations:

```
Steps: MERGE → CREATE_TAG → UPDATE_CHILDREN → DELETE_BRANCH
```

### Flow with Conflict Handling
```
git flow feature finish my-feature --rebase
    ↓
1. Resolve options (3-layer precedence)
2. STATE: MERGE
   → Checkout develop
   → Rebase feature/my-feature
   → On conflict: save state to .git/gitflow/state/merge.json, exit
3. STATE: CREATE_TAG (if configured)
   → Create tag with prefix
4. STATE: UPDATE_CHILDREN
   → Update branches with autoUpdate=true
   → On conflict: save state (including current child), exit
5. STATE: DELETE_BRANCH
   → Checkout the landing branch (last auto-update child of the parent, else the parent)
   → Delete local/remote (unless --keep)
   → Report the branch you are left on
```

### Continue After Conflict
```
git flow feature finish --continue
    ↓
1. Load state from .git/gitflow/state/merge.json
2. Resume from saved step
3. Continue state machine
```

---

## Git Configuration Keys

```ini
gitflow.version                           = "1.0"
gitflow.branch.<name>.type                = "base|topic"
gitflow.branch.<name>.parent              = parent branch
gitflow.branch.<name>.prefix              = "feature/"
gitflow.branch.<name>.upstreamStrategy    = "merge|rebase|squash"
gitflow.branch.<name>.downstreamStrategy  = "merge|rebase"
gitflow.branch.<name>.tag                 = "true|false"
gitflow.branch.<name>.autoUpdate          = "true|false"

gitflow.<type>.start.fetch                = "true|false"
gitflow.<type>.finish.keep                = "true|false"
gitflow.<type>.finish.mergemessage        = "<message template>"
```

### Branch Types
- **Base branches**: main, develop (persistent)
- **Topic branches**: feature, release, hotfix, support, bugfix (temporary)
- **Custom types**: Any via configuration

---

## Module Interaction Flow

```
Commands → config.ResolveFinishOptions() → Three-layer precedence resolution (Layer 1 = branch type identity/process; Layer 2 = command behavior)
        ↓                                        ↓
   git operations                         util.ValidateBranchName()
        ↓                                        ↓
   Conflict? → mergestate.SaveMergeState()     errors.*Error types
        ↓
   Success/Fail → Exit with appropriate code
```

---

## Finding Code

### To modify command behavior:
1. Check if it's a topic command → `cmd/topicbranch.go`
2. Find the actual implementation → `cmd/<command>.go`
3. Check configuration resolution → `internal/config/resolver.go`
4. Git operations → `internal/git/repo.go`

### To add new functionality:
1. New command → Add to `cmd/` and register in `root.go` or `topicbranch.go`
2. New config option → Update `internal/config/config.go` and `resolver.go`
3. New Git operation → Add to `internal/git/repo.go`
4. New branch type → Configure via `gitflow.branch.<name>.*`

### To debug issues:
1. State problems → Check `.git/gitflow/state/merge.json`
2. Config issues → `git config --get-regexp gitflow`
3. Git operations → Trace through `internal/git/repo.go`
4. Command flow → Start at `cmd/<command>.go`

---

## Key Design Patterns

1. **Dynamic Command Generation**: Commands created at runtime based on config
2. **Three-Layer Configuration**: Predictable precedence system
3. **State Machine with Persistence**: Resume complex operations after conflicts
4. **Centralized Git Operations**: All Git commands through single module
5. **Configuration-Driven**: Behavior determined by git config
6. **Error Codes**: Specific exit codes for different failure modes

---

## Testing Structure

```
test/
├─ cmd/           - Command-level tests
├─ internal/      - Unit tests for internal packages
└─ testutil/      - Test helpers (CreateTestRepo, InitGitFlow)
```

Use `testutil.CreateTestRepo()` for integration tests with real Git operations.

---

## Command Examples

### Start Command Flow
```
git flow feature start my-feature
    ↓
1. Load config → get feature.parent (develop)
2. Determine startPoint → feature.startPoint or parent
3. Fetch if configured → gitflow.feature.start.fetch
4. Create branch → git checkout -b feature/my-feature develop
```

### Finish Command Flow
```
git flow feature finish my-feature
    ↓
1. Resolve options (defaults → config → flags)
2. Merge feature/my-feature into develop
3. Create tag if configured
4. Update child branches with autoUpdate=true
5. Delete branch (unless --keep)
```
