# GitHub Copilot Instructions for git-flow-next

This document provides context and guidelines for GitHub Copilot when working with the git-flow-next codebase.

## Project Overview

git-flow-next is a modern, flexible Go implementation of Git workflow management that builds upon the original git-flow concepts with extensive customization capabilities for modern development practices.

### Core Architecture

- **Branch Dependency Model**: Parent-child relationships between branches with automatic change propagation
- **Single Topic Branch Implementation**: Unified command structure for all branch types (feature, hotfix, release)
- **Configuration-Driven**: Branch types and behaviors defined via Git configuration (`gitflow.*` keys)
- **Step-Based State Machine**: Complex operations broken into resumable steps

### Key Branch Types & Default Configuration

**Base Branches** (long-living):
- **main**: Production branch (root branch)
- **develop**: Integration branch (auto-updates from main)

**Topic Branches** (short-living):
- **feature/***: New features (parent: develop, starts from develop)
- **release/***: Release preparation (parent: main, starts from develop)
- **hotfix/***: Emergency fixes (parent: main, starts from main)

**Default Merge Strategies:**
- Feature finish: `merge` into develop
- Release finish: `merge` into main (then auto-update develop)
- Hotfix finish: `merge` into main (then auto-update develop)
- Feature update: `rebase` from develop
- Release/Hotfix update: `merge`/`rebase` from main

**Default Tag Settings:**
- Feature: No tags created
- Release/Hotfix: Tags created on finish

## Coding Guidelines

### Go Code Style

**Package Structure:**
- `cmd/` - CLI command implementations using Cobra
- `internal/config/` - Git configuration management
- `internal/git/` - Git command wrappers
- `internal/errors/` - Custom error types with exit codes
- `internal/mergestate/` - Persistent state for multi-step operations

**Naming Conventions:**
- PascalCase for exported functions: `LoadConfig()`, `FinishCommand()`
- camelCase for private functions: `executeFinish()`, `handleContinue()`
- Use descriptive names: `branchName`, `parentBranch`, `mergeStrategy`

**Constants:**
```go
// Use grouped constants with descriptive prefixes
const (
    stepMerge          = "merge"
    stepCreateTag      = "create_tag"
    stepUpdateChildren = "update_children"
    stepDeleteBranch   = "delete_branch"
)
```

### Error Handling Patterns

**Custom Error Types:**
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

**Error Wrapping:**
```go
if err != nil {
    return &errors.GitError{
        Operation: fmt.Sprintf("checkout branch '%s'", branchName),
        Err:       err,
    }
}
```

### Command Implementation Pattern

**Two-Layer Structure:**
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

// 2. Execute function contains business logic
func executeFinish(params...) error {
    // Implementation logic
    return nil
}
```

### Configuration Patterns

**Loading Configuration:**
```go
cfg, err := config.LoadConfig()
if err != nil {
    return &errors.ConfigError{Err: err}
}

branchConfig, exists := cfg.Branches[branchType]
if !exists {
    return &errors.InvalidBranchTypeError{BranchType: branchType}
}
```

**Configuration Keys:**
- Base branches: `gitflow.branch.main`, `gitflow.branch.develop`
- Base branch relationships: `gitflow.branch.develop.parent`, `gitflow.branch.develop.autoUpdate`
- Topic branch types: `gitflow.branch.feature`, `gitflow.branch.hotfix`, `gitflow.branch.release`
- Merge strategies (upstream): `gitflow.feature.finish.merge`, `gitflow.release.finish.merge`
- Merge strategies (downstream): `gitflow.feature.downstreamStrategy`, `gitflow.release.downstreamStrategy`
- Branch-specific settings: `gitflow.feature.finish.notag`, `gitflow.release.finish.notag`

### Git Operations

**Always Use Wrappers:**
```go
// Never call git directly - always use internal/git package
if err := git.Checkout(branchName); err != nil {
    return &errors.GitError{
        Operation: fmt.Sprintf("checkout branch '%s'", branchName),
        Err:       err,
    }
}
```

### State Management

**Persistent State Structure:**
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

## Testing Patterns

### Test Structure

**Standard Test Setup:**
```go
func TestExample(t *testing.T) {
    // Setup
    dir := testutil.SetupTestRepo(t)
    defer testutil.CleanupTestRepo(t, dir)
    
    // Initialize git-flow with defaults (creates main, develop, configures feature/release/hotfix)
    output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
    if err != nil {
        t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
    }
    
    // Use feature branches for general topic branch testing
    output, err = testutil.RunGitFlow(t, dir, "feature", "start", "test-branch")
    // Test implementation...
}
```

**Testing Best Practices:**
- **Use git-flow defaults**: Initialize test repos with `git flow init --defaults`
- **Use feature branches**: Feature branches for general topic branch testing
- **Modify config when needed**: Use `git config` for specific test scenarios

**Test Naming:**
- Descriptive function names: `TestFinishFeatureBranchWithMergeConflict`
- Include step-by-step comments describing test state and expectations

### Test Utilities

**Common Operations:**
```go
// Repository setup
dir := testutil.SetupTestRepo(t)
defer testutil.CleanupTestRepo(t, dir)

// Git operations  
testutil.RunGitFlow(t, dir, "feature", "start", "test-branch")
testutil.WriteFile(t, dir, "file.txt", "content")
testutil.RunGit(t, dir, "add", "file.txt")

// Assertions
if !testutil.BranchExists(t, dir, "feature/test-branch") {
    t.Error("Expected branch to exist")
}
```

## Key Implementation Details

### The Finish Command (Most Complex)

**Step-Based Architecture:**
1. `stepMerge` - Merge topic branch using configured strategy
2. `stepCreateTag` - Create tags if configured
3. `stepUpdateChildren` - Update dependent child branches 
4. `stepDeleteBranch` - Clean up branches based on retention settings

**Merge Strategies:**
- `strategyMerge` - Standard Git merge
- `strategyRebase` - Rebase then fast-forward merge
- `strategySquash` - Squash all commits into one

**State Persistence:**
- Operations can be interrupted by conflicts
- State is saved to `.git/gitflow/state/merge.json`
- Resume with `--continue` or abort with `--abort`

### Configuration System

**Branch Configuration:**
```go
type BranchConfig struct {
    Type               string // "base" or "topic"
    Parent             string // Parent branch name
    StartPoint         string // Where to create from
    UpstreamStrategy   string // "merge", "rebase", "none"
    DownstreamStrategy string // "merge", "rebase", "squash", "none"
    Prefix             string // Branch name prefix
    AutoUpdate         bool   // Auto-update from parent
    Tag                bool   // Create tags on finish
    TagPrefix          string // Tag name prefix
}
```

### Dynamic Command Registration

**Pattern:**
- `cmd/topicbranch.go` reads configuration to determine available branch types
- Dynamically creates Cobra commands for each type
- All commands use the same underlying implementation with different parameters

## Common Patterns to Follow

1. **Always validate inputs** before performing Git operations
2. **Use structured errors** with specific types and exit codes  
3. **Wrap Git operations** through `internal/git` package
4. **Handle conflicts gracefully** with clear user instructions
5. **Persist state** for multi-step operations
6. **Test thoroughly** with realistic Git scenarios
7. **Document complex logic** with clear comments
8. **Follow Go conventions** for naming and structure

## Output Patterns

**User Communication:**
```go
// Progress messages
fmt.Printf("Merging using strategy: %s\n", strategy)
fmt.Printf("Created tag '%s'\n", tagName)

// Error messages (to stderr)
fmt.Fprintf(os.Stderr, "Error: %s\n", err)

// Conflict resolution instructions
msg := fmt.Sprintf("Merge conflicts detected. Resolve conflicts and run 'git flow %s finish --continue %s'\n", 
    branchType, branchName)
msg += fmt.Sprintf("To abort the merge, run 'git flow %s finish --abort %s'", branchType, branchName)
fmt.Println(msg)
```

## Commit Guidelines

Follow the project's commit message standards as defined in [COMMIT_GUIDELINES.md](../COMMIT_GUIDELINES.md). Key points:

- Use structured format: `<type>: <subject>` with optional body and footer
- Types: feat, fix, refactor, test, docs, style, etc.
- Subject ≤50 characters, imperative mood, no period
- Body explains what and why, uses flowing paragraphs
- Reference issues with "Fixes #123", "Closes #456"

## Integration Points

- **Tower**: Git Tower GUI integration using same configuration
- **CI/CD**: Webhook-triggered deployments based on branch patterns
- **git-flow-avh**: Compatibility with existing configurations

When suggesting code changes, ensure they align with these established patterns and maintain consistency with the existing codebase architecture. Always follow the commit guidelines when creating commits or suggesting commit messages.