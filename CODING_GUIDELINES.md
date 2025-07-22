# Coding Guidelines

This document outlines the coding standards and conventions used in the git-flow-next project. Following these guidelines ensures consistency and maintainability across the codebase.

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

### Command Implementation Pattern

Each command follows a consistent two-layer pattern:

```go
// 1. Command function handles error conversion and exit codes
func FinishCommand(params...) {
    if err := executeFinish(params...); err != nil {
        var exitCode errors.ExitCode
        if flowErr, ok := err.(errors.Error); ok {
            exitCode = flowErr.ExitCode()
        } else {
            exitCode = errors.ExitCodeGeneral
        }
        fmt.Fprintf(os.Stderr, "Error: %s\n", err)
        os.Exit(int(exitCode))
    }
}

// 2. Execute function contains actual business logic
func executeFinish(params...) error {
    // Implementation logic
    // Return structured errors
    return nil
}
```

### Configuration Handling

Load configuration once and pass to functions:

```go
// Load configuration early
cfg, err := config.LoadConfig()
if err != nil {
    return &errors.ConfigError{Err: err}
}

// Validate using configuration methods
branchConfig, exists := cfg.Branches[branchType]
if !exists {
    return &errors.InvalidBranchTypeError{BranchType: branchType}
}
```

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

### Code Reviews

All code changes must:
- Follow these coding guidelines
- Include appropriate tests
- Have clear commit messages
- Pass all existing tests
- Include documentation for new features

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