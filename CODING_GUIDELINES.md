# Coding Guidelines

This document outlines the coding standards and conventions used in the git-flow-next project. Following these guidelines ensures consistency and maintainability across the codebase.

## Documentation Requirements

**⚠️ CRITICAL**: Documentation in `docs/` must be updated whenever making changes to commands, options, or behavior.

### Documentation Update Rules

1. **New Commands**: Create new manpage documentation in `docs/`
2. **Modified Commands**: Update existing manpage files with current behavior  
3. **New Options**: Add to relevant command documentation with examples
4. **Changed Behavior**: Update descriptions, examples, and exit codes
5. **Configuration Changes**: Update `gitflow-config.5.md` with new keys/values

### Documentation Standards

- Follow Unix manpage conventions (see `docs/README.md`)
- Include practical examples for all new functionality
- Update cross-references in SEE ALSO sections
- Test all examples for accuracy
- Consider impact on different workflows (Classic, GitHub, GitLab, Custom)

### Before Committing

Always verify documentation is current:
```bash
# Check if any command help text changed
git diff --name-only | grep -E '(cmd/|internal/)' && echo "Review docs/ for updates"

# Ensure examples work
cd docs && grep -r "git flow" *.md | head -10  # Spot-check examples
```

## Development Philosophy

**CRITICAL**: We follow a **pragmatic, anti-over-engineering approach** to software development:

### Core Principles

1. **Pragmatism Over Patterns**
   - Use patterns wisely, but don't let them dictate your code
   - Solve real problems, not theoretical ones
   - Choose simplicity over cleverness

2. **Meaningful Encapsulation**
   - Encapsulate code in meaningful ways that reflect the problem domain
   - Group related functionality naturally, not artificially
   - Avoid unnecessary abstractions that don't add real value

3. **Function Parameters and Complexity**
   - **Complex functions will naturally receive many parameters** - this is acceptable
   - Many function calls are perfectly fine if that's what the logic requires
   - Don't artificially reduce parameter counts just for the sake of it

4. **Option Structs - When They Make Sense**
   - Combine function arguments into option structs when it **makes logical sense**
   - Example: `TagOptions` groups related tag configuration (good)
   - Don't create option structs just to reduce parameter counts (bad)

5. **Anti-Over-Engineering**
   - Reject unnecessary complexity in favor of straightforward solutions
   - Avoid premature abstractions and excessive layering
   - Write code that directly solves the problem at hand

### What This Means in Practice

```go
// GOOD: Complex function with many parameters that reflect the real problem
func FinishCommand(branchType string, name string, continueOp bool, abortOp bool, 
    force bool, tagOptions *TagOptions, retentionOptions *BranchRetentionOptions)

// GOOD: Option struct groups related concepts
type TagOptions struct {
    ShouldTag   *bool
    ShouldSign  *bool
    SigningKey  string
    Message     string
}

// BAD: Artificial abstraction that doesn't add value
type FinishContext struct {
    Everything interface{}
}
func FinishCommand(ctx *FinishContext)
```

### Guidelines for Refactoring

- **Don't refactor for the sake of following patterns**
- **Do refactor when it solves actual maintenance or clarity problems**
- **Prefer explicit over implicit**
- **Prefer readable over clever**

## Go Code Style

### Package Organization

**Package Naming:**
- Use single word, lowercase names: `config`, `git`, `errors`, `util`
- Avoid underscores or mixed case
- Keep package names short and descriptive

**File Organization:**
- One file per major command in `cmd/`
- Group related functionality within packages
- Use descriptive filenames: `config.go`, `repo.go`, `mergestate.go`

### Import Organization

Organize imports in three distinct groups with blank lines between:

```go
import (
    // 1. Standard library packages (alphabetical)
    "fmt"
    "os"
    "strings"

    // 2. Third-party packages (alphabetical)
    "github.com/spf13/cobra"

    // 3. Local packages (alphabetical)
    "github.com/gittower/git-flow-next/internal/config"
    "github.com/gittower/git-flow-next/internal/errors"
    "github.com/gittower/git-flow-next/internal/git"
)
```

### Naming Conventions

**Functions:**
- PascalCase for exported functions: `LoadConfig()`, `FinishCommand()`
- camelCase for private functions: `executeFinish()`, `handleContinue()`
- Use descriptive verb names: `ValidateBranchName()`, `CreateTag()`

**Variables:**
- camelCase consistently: `branchName`, `mergeStrategy`, `configValue`
- Avoid abbreviations unless they're well-known: `cfg` for config, `err` for error
- Use descriptive names: `parentBranch` not `parent`

**Constants:**
- PascalCase for exported: `ExitCodeSuccess`, `DefaultTimeout`
- Use groups with descriptive prefixes:

```go
// Step constants for state machine
const (
    stepMerge          = "merge"
    stepCreateTag      = "create_tag"
    stepUpdateChildren = "update_children"
    stepDeleteBranch   = "delete_branch"
)
```

**Types:**
- PascalCase for exported structs: `BranchConfig`, `TagOptions`
- Use descriptive, unambiguous names
- Suffix with purpose when helpful: `BranchRetentionOptions`, `MergeState`

### Struct Definitions

Structure fields logically and provide clear documentation:

```go
// TagOptions contains options for tag creation when finishing a branch
type TagOptions struct {
    ShouldTag   *bool  // Whether to create a tag (nil means use config default)
    ShouldSign  *bool  // Whether to sign the tag (nil means use config default)
    SigningKey  string // GPG signing key to use
    Message     string // Custom tag message
    MessageFile string // Path to file containing tag message
    TagName     string // Custom tag name
}
```

**Guidelines:**
- Group related fields together
- Use pointer types for optional boolean values (`*bool`)
- Document when `nil` has special meaning
- Align field comments for readability

## Error Handling

### Custom Error Types

Define structured errors with specific types and exit codes:

```go
type BranchNotFoundError struct {
    BranchName string
}

func (e *BranchNotFoundError) Error() string {
    return fmt.Sprintf("branch '%s' not found", e.BranchName)
}

func (e *BranchNotFoundError) ExitCode() ExitCode {
    return ExitCodeBranchNotFound
}
```

### Error Handling Pattern

Always handle errors explicitly with appropriate context:

```go
output, err := git.GetConfig(configKey)
if err != nil {
    return &errors.GitError{
        Operation: fmt.Sprintf("get config '%s'", configKey),
        Err:       err,
    }
}
```

**Guidelines:**
- Never ignore errors (`_ = someFunction()` is prohibited)
- Wrap errors with context using structured error types
- Return specific error types for different failure conditions
- Use `fmt.Errorf()` with `%w` verb for error wrapping when appropriate

### Exit Codes

Define meaningful exit codes for different error conditions:

```go
const (
    ExitCodeSuccess               ExitCode = 0
    ExitCodeGeneral              ExitCode = 1
    ExitCodeNotInitialized       ExitCode = 2
    ExitCodeBranchNotFound       ExitCode = 3
    ExitCodeMergeConflict        ExitCode = 4
    ExitCodeInvalidBranchType    ExitCode = 5
    ExitCodeUncommittedChanges   ExitCode = 6
)
```

## Command Structure

### Universal Command Architecture

All commands must follow this exact three-layer pattern:

```go
// Layer 1: Cobra Command Handler (in init() or command creation)
RunE: func(cmd *cobra.Command, args []string) error {
    // Parse flags and arguments
    param1, _ := cmd.Flags().GetString("flag1")
    param2, _ := cmd.Flags().GetBool("flag2")
    
    // Call command wrapper
    CommandName(branchType, name, param1, param2)
    return nil
}

// Layer 2: Command Wrapper - Handles error conversion and exit codes
func CommandName(params...) {
    if err := executeCommand(params...); err != nil {
        var exitCode errors.ExitCode
        if flowErr, ok := err.(errors.Error); ok {
            exitCode = flowErr.ExitCode()
        } else {
            exitCode = errors.ExitCodeGitError // Default fallback
        }
        fmt.Fprintf(os.Stderr, "Error: %v\n", err)
        os.Exit(int(exitCode))
    }
}

// Layer 3: Execute Function - Contains actual business logic
func executeCommand(params...) error {
    // All business logic here
    // Return structured errors only
    return nil
}
```

### Configuration Loading Pattern

**CRITICAL**: Load configuration once at the beginning and pass through all calls:

```go
func executeCommand(params...) error {
    // Load config ONCE at the start
    cfg, err := config.LoadConfig()
    if err != nil {
        return &errors.GitError{Operation: "load configuration", Err: err}
    }
    
    // Pass cfg to all subsequent function calls
    return doWork(cfg, params...)
}
```

**Guidelines:**
- Never call `config.LoadConfig()` multiple times within a command
- Pass `cfg` parameter to all helper functions that need configuration
- Cache configuration lookups within the command execution

### Standard Command Initialization

Every command (except `init`) must start with this exact pattern:

```go
// Validate that git-flow is initialized
initialized, err := config.IsInitialized()
if err != nil {
    return &errors.GitError{Operation: "check if git-flow is initialized", Err: err}
}
if !initialized {
    return &errors.NotInitializedError{}
}
```

### Branch Type Validation

For topic branch commands:

```go
// Get branch configuration
branchConfig, ok := cfg.Branches[branchType]
if !ok {
    return &errors.InvalidBranchTypeError{BranchType: branchType}
}
```

### Branch Name Resolution

For commands accepting branch names:

```go
// Construct full branch name if needed
fullBranchName := name
if branchConfig.Prefix != "" && !strings.HasPrefix(name, branchConfig.Prefix) {
    fullBranchName = branchConfig.Prefix + name
}

// Check if branch exists
if err := git.BranchExists(fullBranchName); err != nil {
    return &errors.BranchNotFoundError{BranchName: fullBranchName}
}
```

### Current Branch Detection

When commands can work on current branch:

```go
var branchName string
if name == "" {
    currentBranch, err := git.GetCurrentBranch()
    if err != nil {
        return &errors.GitError{Operation: "get current branch", Err: err}
    }
    branchName = currentBranch
} else {
    branchName = name
}
```

### Configuration Override Pattern

**CRITICAL**: All commands must implement a **three-layer precedence hierarchy** where command-line arguments always win. Branch defaults are reserved for essential branch-type configuration; some options intentionally skip Layer 1 and use only Layer 2 + Layer 3 (e.g., publish push-options).

#### **Layer 1: Branch Configuration Defaults**
Default values from branch type configuration in Git config under `gitflow.branch.*`

#### **Layer 2: Command-Specific Git Config Overrides**
Branch-specific overrides in Git config under `gitflow.<branchtype>.*`

#### **Layer 3: Command-Line Arguments (HIGHEST PRIORITY)**
Explicit flags passed to the command - **THESE ALWAYS WIN**

#### **Implementation Pattern:**

```go
// 1. Start with branch configuration default (Layer 1)
shouldTag := branchConfig.Tag

// 2. Check for branch-specific config override (Layer 2)
branchSpecificConfig, err := git.GetConfig(fmt.Sprintf("gitflow.%s.finish.notag", branchType))
if err == nil && branchSpecificConfig == "true" {
    shouldTag = false
}

// 3. Command-line flags override everything (Layer 3 - WINS)
if tagOptions != nil && tagOptions.ShouldTag != nil {
    shouldTag = *tagOptions.ShouldTag
}
```

#### **Examples of Configuration Hierarchy:**

**Tag Creation Example:**
```bash
# Layer 1: Branch config default
git config gitflow.branch.release.tag true

# Layer 2: Command-specific override
git config gitflow.release.finish.notag true  # Disables tags

# Layer 3: Command-line override (WINS)
git flow release finish v1.0 --tag  # Forces tag creation despite config
```

**Merge Strategy Example:**
```bash
# Layer 1: Branch config default
git config gitflow.branch.feature.upstreamStrategy merge

# Layer 2: Command-specific override
git config gitflow.feature.finish.rebase true

# Layer 3: Command-line override (WINS)
git flow feature finish my-feature --squash  # Forces squash merge
```

#### **Required Implementation:**

1. **Check the applicable layers** in the correct order (skip Layer 1 if the option has no branch default)
2. **Use pointer types** (`*bool`, `*string`) for command-line options to distinguish between "not set" and "set to false/empty"
3. **Command-line flags must always win** - no exceptions
4. **Document the precedence** in command help text when relevant

#### **Adding New Configuration Options:**

When adding a new option, the key question is: **does this describe what the branch type *is*, or how a command *executes*?**

**Most new options need only Layer 2 + Layer 3.** Operational settings like `fetch`, `sign`, `keep`, `push-options`, or `force-delete` control command behavior — they don't define a branch type characteristic. These belong in `gitflow.<branchtype>.<command>.<option>` (Layer 2) with a corresponding CLI flag (Layer 3).

**Add Layer 1 support only if the option defines a branch type's identity or process characteristic** — something inherent to what the branch type *is*. Ask yourself: "Would this make sense as part of the branch type's definition when someone sets up a new workflow?" Examples:
- `tag` — "releases produce tags" describes the release process → Layer 1
- `upstreamStrategy` — "features merge into develop" describes the feature workflow → Layer 1
- `sign` — "sign this tag with GPG" is an operational detail → Layer 2 + 3 only
- `keep` — "keep the branch after finishing" is a command behavior → Layer 2 + 3 only

**Checklist for new options:**

1. **Always** add Layer 3 (CLI flag) — users must be able to override per-invocation
2. **Always** add Layer 2 (`gitflow.<type>.<command>.<option>`) — users must be able to set persistent command behavior
3. **Only if it's a branch type characteristic** add Layer 1 (`gitflow.branch.<type>.<property>`) — and add it to the `BranchConfig` struct
4. **Update documentation** — add to `docs/gitflow-config.5.md` and `CONFIGURATION.md` in the appropriate layer sections

**Narrow exception to rule 2 — a pure Layer-1 identity property.** When an option belongs to Layer 1 *and* a Layer-2 twin would mean exactly the same thing for exactly one command, ship Layer 1 + Layer 3 only. A second key with identical meaning is a resolution trap: users cannot tell which one wins, and the resolver has to invent a precedence that carries no information. The shipped example is `gitflow.branch.<type>.worktree` (`git flow <type> start --worktree`/`--no-worktree`), where `gitflow.<type>.start.worktree` would have been that twin.

This is the exception, not an escape hatch. It does **not** apply when Layer 2 would say something Layer 1 cannot — a different value per command (`fetch` on `start` vs `finish`), or a command behavior that is not part of the branch type's identity. If in doubt, add Layer 2. When the exception is taken, say why in the resolver's godoc so the omission reads as deliberate.

### Input Validation

Always validate inputs early in the execute function:

```go
// Validate required inputs
if name == "" {
    return &errors.EmptyBranchNameError{}
}

// Validate branch exists before operations
if err := git.BranchExists(branchName); err != nil {
    return &errors.BranchNotFoundError{BranchName: branchName}
}
```

### Cobra Positional Argument Validation

Every leaf command must declare an explicit Cobra `Args` validator. Use
`cobra.NoArgs` when the command takes no positional arguments, or a bounded
validator such as `cobra.ExactArgs`, `cobra.MaximumNArgs`, or
`cobra.RangeArgs` that matches exactly what the command consumes. Parent and
dispatch commands that only delegate to subcommands are exempt.

### Options Struct Pattern

For commands with multiple flags, use structured options:

```go
type TagOptions struct {
    ShouldTag   *bool  // nil means use config default
    ShouldSign  *bool  // nil means use config default
    SigningKey  string
    Message     string
    TagName     string
}

type BranchRetentionOptions struct {
    Keep        *bool // Whether to keep the branch
    KeepRemote  *bool // Whether to keep remote branch
    KeepLocal   *bool // Whether to keep local branch
    ForceDelete *bool // Whether to force delete
}
```

### Command Function Signatures

Use consistent signatures:
- Command wrapper: `func CommandName(branchType, name string, options...)`
- Execute function: `func executeCommand(branchType, name string, options...) error`

## Git Operations

### Wrapper Functions

All Git operations must go through wrapper functions in `internal/git/`:

```go
// internal/git/repo.go
func Checkout(branch string) error {
    cmd := exec.Command("git", "checkout", branch)
    if err := cmd.Run(); err != nil {
        return fmt.Errorf("failed to checkout branch '%s': %w", branch, err)
    }
    return nil
}
```

**Guidelines:**
- Never call `git` commands directly in command functions
- Provide clear error messages with context
- Handle common Git errors appropriately
- Use consistent parameter validation

### Git Configuration Management

When modifying Git configuration, always clean up old entries to prevent stale configuration:

```go
// ❌ BAD: Only adds new config, leaves old entries
func RenameBaseBranch(oldName, newName string) error {
    cfg.Branches[newName] = cfg.Branches[oldName]
    delete(cfg.Branches, oldName)
    return config.SaveConfig(cfg)
}

// ✅ GOOD: Removes old config before saving new config
func RenameBaseBranch(oldName, newName string) error {
    // Remove old git config section first
    if err := git.UnsetConfigSection(fmt.Sprintf("gitflow.branch.%s", oldName)); err != nil {
        return fmt.Errorf("failed to remove old branch config: %w", err)
    }
    
    // Update in-memory configuration
    cfg.Branches[newName] = cfg.Branches[oldName]
    delete(cfg.Branches, oldName)
    
    // Save new configuration
    return config.SaveConfig(cfg)
}
```

#### UnsetConfigSection Implementation

```go
// UnsetConfigSection removes all Git config values matching a pattern
func UnsetConfigSection(pattern string) error {
    cmd := exec.Command("git", "config", "--remove-section", pattern)
    _, err := cmd.Output()
    if err != nil {
        // Don't treat "section not found" as an error  
        if strings.Contains(err.Error(), "exit status 128") {
            return nil
        }
        return fmt.Errorf("failed to unset git config section %s: %w", pattern, err)
    }
    return nil
}
```

**Configuration Management Guidelines:**
- **Always remove old config** before saving new config during rename/delete operations
- **Handle missing sections gracefully** - `git config --remove-section` returns exit status 128 if section doesn't exist
- **Use complete section patterns** - `gitflow.branch.branchname` removes all keys under that branch
- **Clean up atomically** - Remove old config before saving new to avoid inconsistent state

### Command-Line Interface Design

When implementing CLI commands that can operate in different modes:

```go
// Detect configuration flags to determine command mode
hasConfigFlags := mainBranch != "" || developBranch != "" || featurePrefix != "" || 
                  releasePrefix != "" || hotfixPrefix != "" || supportPrefix != ""

if hasConfigFlags {
    // Non-interactive mode with provided configuration
    cfg = config.DefaultConfig()
    // Apply flag overrides...
} else if useDefaults {
    // Non-interactive mode with defaults
    cfg = config.DefaultConfig()
} else {
    // Interactive mode
    cfg = interactiveConfig()
}
```

**CLI Design Guidelines:**
- **Detect explicit configuration** - Any provided config flags should prevent interactive mode
- **Three-layer precedence** - Defaults (when applicable) → Git config → Command flags (highest priority)
- **Clear mode determination** - Make it obvious when interactive vs non-interactive mode is used
- **Consistent flag handling** - Similar patterns across all commands that accept configuration

## State Management

### Persistent State

For complex multi-step operations, use persistent state:

```go
type MergeState struct {
    Action          string   `json:"action"`
    BranchType      string   `json:"branch_type"`
    BranchName      string   `json:"branch_name"`
    CurrentStep     string   `json:"current_step"`
    ParentBranch    string   `json:"parent_branch"`
    MergeStrategy   string   `json:"merge_strategy"`
    FullBranchName  string   `json:"full_branch_name"`
    ChildBranches   []string `json:"child_branches"`
    UpdatedBranches []string `json:"updated_branches"`
}
```

**Guidelines:**
- Use JSON tags for serialization
- Include all information needed to resume operations
- Provide clear field names and types
- Document state transitions

## Output and User Communication

### Output Patterns

Use consistent patterns for user communication:

```go
// Regular progress output
fmt.Printf("Merging using strategy: %s\n", strategy)

// Error output to stderr
fmt.Fprintf(os.Stderr, "Error: %s\n", err)

// Success messages with context
fmt.Printf("Successfully finished branch '%s' and updated %d child branches\n", 
    branchName, len(updatedBranches))
```

### User-Friendly Messages

Provide clear, actionable error messages:

```go
func (e *MergeConflictError) Error() string {
    return fmt.Sprintf("Merge conflicts detected. Resolve conflicts and run 'git flow %s finish --continue %s'", 
        e.BranchType, e.BranchName)
}
```

## Testing

### Test Organization

- Tests mirror source structure in `test/` directory
- Use descriptive test names: `TestFinishFeatureBranchWithMergeConflict`
- Group related tests in the same file

### Test Structure

Follow consistent test structure:

```go
func TestExample(t *testing.T) {
    // Setup
    dir := testutil.SetupTestRepo(t)
    defer testutil.CleanupTestRepo(t, dir)
    
    // Execute
    output, err := testutil.RunGitFlow(t, dir, "feature", "start", "test")
    
    // Assert
    if err != nil {
        t.Fatalf("Expected success, got error: %v\nOutput: %s", err, output)
    }
    
    // Verify state
    if !testutil.BranchExists(t, dir, "feature/test") {
        t.Error("Expected feature branch to be created")
    }
}
```

### Test Utilities

Use shared test utilities for common operations:

```go
// Setup and cleanup
dir := testutil.SetupTestRepo(t)
defer testutil.CleanupTestRepo(t, dir)

// Git operations
_, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
testutil.WriteFile(t, dir, "file.txt", "content")

// Assertions
if !testutil.BranchExists(t, dir, "feature/test") {
    t.Error("Expected branch to exist")
}
```

## Documentation

### Function Documentation

Document all exported functions with clear descriptions:

```go
// LoadConfig loads the git-flow configuration from git config.
// It reads all gitflow.branch.* configuration keys and constructs
// a Config struct with branch definitions and settings.
func LoadConfig() (*Config, error) {
    // Implementation
}
```

### Package Documentation

Provide package-level documentation at the top of main package files:

```go
// Package config provides git-flow configuration management.
// It handles loading branch type definitions and workflow settings
// from Git configuration files.
package config
```

### Complex Logic Documentation

Document complex algorithms and state machines:

```go
// The finish operation progresses through sequential steps:
// 1. merge: Merge topic branch into parent
// 2. create_tag: Create tag if configured  
// 3. update_children: Update dependent child branches
// 4. delete_branch: Clean up topic branch
//
// Each step can be interrupted by conflicts and resumed with --continue.
func finish(state *mergestate.MergeState) error {
    // Implementation
}
```

## Quality Standards

### Mandatory Change Requirements

**CRITICAL**: When changing code, you MUST always complete all three steps:

1. **Create/Adjust Tests**
   - Add new tests for new functionality
   - Update existing tests when behavior changes
   - Follow test naming conventions: `TestFunctionNameWithScenario`
   - Use test utilities from `testutil/` package
   - Ensure tests cover both success and error cases

2. **Run All Tests**
   - Execute `go test ./...` to run complete test suite
   - Fix any test failures before committing
   - Ensure no regressions in existing functionality
   - Verify new tests pass consistently

3. **Update Documentation**
   - Update relevant `.md` files for user-facing changes
   - Update function/package documentation for API changes
   - Update CLAUDE.md for development workflow changes
   - Update configuration examples if config changes

**Test Command Reference:**
```bash
# Run all tests
go test ./...

# Run tests for specific package
go test ./test/cmd/
go test ./test/internal/

# Run specific test file with verbose output
go test -v ./test/cmd/init_test.go
```

### Code Reviews

All code changes must:
- Follow these coding guidelines
- **Complete all three mandatory requirements above**
- Have clear commit messages following COMMIT_GUIDELINES.md
- Pass all existing tests
- Include comprehensive test coverage

### Static Analysis

The project uses:
- `go fmt` for consistent formatting
- `go vet` for basic static analysis
- `golint` for style checking (when available)

### Performance Considerations

- Minimize Git operations in large repositories
- Cache configuration lookups where appropriate
- Use efficient data structures for branch operations
- Avoid unnecessary string allocations in hot paths

These guidelines help maintain code quality and ensure consistency across the git-flow-next project. When in doubt, follow the patterns established in existing code and prioritize clarity and maintainability.
