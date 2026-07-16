# Testing Guidelines

This document outlines the testing conventions and patterns used in the git-flow-next project.

> **Important**: When implementing tests, you MUST read [GIT_TEST_SCENARIOS.md](GIT_TEST_SCENARIOS.md) for detailed instructions on setting up Git test scenarios (merge conflicts, remotes, state verification). That document contains essential patterns for creating realistic test conditions.

## Test Function Naming

Follow descriptive naming patterns that clearly indicate what is being tested:

```go
// Basic functionality
func TestStartFeatureBranch(t *testing.T)
func TestFinishFeatureBranch(t *testing.T)
func TestInitWithDefaults(t *testing.T)

// Configuration-specific tests
func TestStartWithCustomConfig(t *testing.T)
func TestInitWithAVHConfig(t *testing.T)

// Error conditions
func TestUpdateWithMergeConflict(t *testing.T)
func TestFinishWithMergeConflict(t *testing.T)
func TestDeleteNonExistentRemoteBranch(t *testing.T)

// Feature-specific tests
func TestFinishFeatureBranchWithFetchFlag(t *testing.T)
func TestStartWithFetchFlag(t *testing.T)
```

## Test Comments and State Documentation

**REQUIRED**: Every test function must have a structured comment that follows this exact pattern:

1. **First line**: Brief description of what the test validates
2. **Steps:** section with numbered list of test actions
3. **Expected outcomes** embedded in the steps

### Comment Pattern (MANDATORY)

```go
// TestFinishWithMergeConflict tests the behavior when finishing a branch with merge conflicts.
// Steps:
// 1. Sets up a test repository and initializes git-flow
// 2. Creates a feature branch
// 3. Adds conflicting changes to both feature and develop branches
// 4. Attempts to finish the feature branch
// 5. Verifies the operation fails with merge conflict
func TestFinishWithMergeConflict(t *testing.T) {
    // Test implementation...
}
```

### Additional Comment Guidelines

- **Be specific**: Include exact branch names, commands, and expected results
- **Number all steps**: Use sequential numbering (1, 2, 3...)  
- **Include verification**: Always specify what is being verified
- **Use active voice**: "Creates a branch" not "A branch is created"
- **Match test structure**: Comments should reflect actual test implementation

### Examples of Good Test Comments

```go
// TestConfigAddBase tests adding base branch configurations.
// Steps:
// 1. Sets up a test repository and initializes git-flow
// 2. Adds various base branches with different configurations  
// 3. Verifies branches are created and configuration is saved correctly
// 4. Tests error conditions like duplicate branches and invalid parents
func TestConfigAddBase(t *testing.T) { ... }

// TestStartFeatureBranch tests the start command for feature branches.
// Steps:
// 1. Sets up a test repository and initializes git-flow with defaults
// 2. Runs 'git flow feature start my-feature'
// 3. Verifies feature/my-feature branch is created
// 4. Verifies branch is based on develop branch
func TestStartFeatureBranch(t *testing.T) { ... }
```

## One Test Case Per Function Rule

**CRITICAL**: Each test function must test exactly one scenario or behavior. Never use table-driven tests with multiple test cases in a single function.

### ❌ BAD: Multiple Test Cases in One Function

```go
func TestMergeStrategyConfigHierarchy(t *testing.T) {
    testCases := []struct {
        name           string
        branchConfig   string
        commandConfig  string
        flag           string
        expectedResult string
    }{
        {"branch_default_merge", "merge", "", "", "merge"},
        {"command_overrides_branch", "merge", "rebase=true", "", "rebase"},
        {"flag_overrides_config", "merge", "rebase=true", "--no-rebase", "merge"},
    }
    
    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            // Test implementation...
        })
    }
}
```

### ✅ GOOD: One Test Case Per Function

```go
// TestMergeStrategyBranchDefault tests that branch default strategy is used.
// Steps:
// 1. Sets up repository with branch configured for merge strategy
// 2. Runs finish command without overrides
// 3. Verifies merge strategy was used
func TestMergeStrategyBranchDefault(t *testing.T) {
    // Single test scenario implementation...
}

// TestMergeStrategyCommandConfigOverride tests command config overriding branch default.
// Steps:
// 1. Sets up repository with branch configured for merge strategy
// 2. Sets gitflow.feature.finish.rebase=true in config
// 3. Runs finish command without flags
// 4. Verifies rebase strategy was used instead of merge
func TestMergeStrategyCommandConfigOverride(t *testing.T) {
    // Single test scenario implementation...
}

// TestMergeStrategyFlagOverridesConfig tests command flag overriding config.
// Steps:
// 1. Sets up repository with rebase configured via command config
// 2. Runs finish command with --no-rebase flag
// 3. Verifies merge strategy was used instead of rebase
func TestMergeStrategyFlagOverridesConfig(t *testing.T) {
    // Single test scenario implementation...
}
```

### Why One Test Case Per Function?

1. **Clear failure identification** - When a test fails, you immediately know which specific scenario failed
2. **Focused debugging** - Each test tests one thing, making debugging straightforward
3. **Maintainable test suite** - Easy to modify, disable, or extend individual test scenarios
4. **Better test names** - Function names clearly describe what is being tested
5. **Proper test isolation** - Each test sets up exactly what it needs, nothing more
6. **Follows git-flow-next philosophy** - Explicit and readable over clever and compact

### Exception: Helper Functions

Table-driven tests are acceptable in helper functions that test pure functions with multiple input/output combinations:

```go
func TestValidateBranchName(t *testing.T) {
    // This is acceptable for pure validation functions
    testCases := []struct {
        name    string
        input   string
        isValid bool
    }{
        {"valid_name", "feature-branch", true},
        {"invalid_spaces", "feature branch", false},
        {"invalid_colon", "feature:branch", false},
    }
    
    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            result := util.ValidateBranchName(tc.input)
            assert.Equal(t, tc.isValid, result == nil)
        })
    }
}
```

**Rule**: Only use table-driven tests for testing pure functions with simple input/output validation, never for complex integration scenarios.

## Temporary Git Repository Testing

All tests use temporary Git repositories created through test utilities.

### Default Branch Configuration

When using `git flow init --defaults`, the following branches and settings are configured:

#### Base Branches
- **main**: Production branch (root branch)
- **develop**: Integration branch (auto-updates from main)

#### Topic Branch Types  
- **feature**: Prefix `feature/`, parent `develop`, starts from `develop`
- **release**: Prefix `release/`, parent `main`, starts from `develop` 
- **hotfix**: Prefix `hotfix/`, parent `main`, starts from `main`

#### Default Merge Strategies
- **Feature finish**: `merge` into `develop`
- **Release finish**: `merge` into `main` (then auto-update `develop`)
- **Hotfix finish**: `merge` into `main` (then auto-update `develop`)
- **Feature update**: `rebase` from `develop`
- **Release update**: `merge` from `main`
- **Hotfix update**: `rebase` from `main`

#### Default Tag Settings
- **Feature**: No tags created
- **Release**: Tags created on finish
- **Hotfix**: Tags created on finish

### Testing with Default Configuration

- **Use git-flow defaults by default**: Initialize repositories with `git flow init --defaults`
- **Use feature branches for topic branch testing**: Feature branches are the most common topic branch type and should be used for general topic branch functionality tests
- **Only create standalone branches when required**: If your test specifically needs a non-git-flow branch or custom configuration, create it explicitly
- **Adjust configuration when needed**: Use `git config` commands to modify behavior for specific test scenarios

```go
// Standard test setup - use this for most tests
func TestExample(t *testing.T) {
    dir := testutil.SetupTestRepo(t)
    defer testutil.CleanupTestRepo(t, dir)
    
    // Initialize with defaults - creates main, develop, and configures feature/release/hotfix
    output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
    if err != nil {
        t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
    }
    
    // Use feature branches for topic branch testing
    output, err = testutil.RunGitFlow(t, dir, "feature", "start", "test-branch")
    // ... test implementation
}

// Example: Testing with modified merge strategy
func TestFeatureFinishWithRebase(t *testing.T) {
    dir := testutil.SetupTestRepo(t)
    defer testutil.CleanupTestRepo(t, dir)
    
    // Initialize with defaults
    output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
    if err != nil {
        t.Fatalf("Failed to initialize git-flow: %v", err)
    }
    
    // Modify merge strategy for this test
    _, err = testutil.RunGit(t, dir, "config", "gitflow.feature.finish.merge", "rebase")
    if err != nil {
        t.Fatalf("Failed to configure rebase strategy: %v", err)
    }
    
    // Now test with the modified configuration
    output, err = testutil.RunGitFlow(t, dir, "feature", "start", "rebase-test")
    // ... test implementation
}
```

### Basic Test Setup

```go
func TestExample(t *testing.T) {
    // Setup temporary repository
    dir := testutil.SetupTestRepo(t)
    defer testutil.CleanupTestRepo(t, dir)
    
    // Initialize git-flow with defaults
    output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
    if err != nil {
        t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
    }
    
    // Test implementation...
}
```

### Git Test Helper Functions

Available in `test/testutil/git.go`:

- `SetupTestRepo(t *testing.T) string` - Creates temporary Git repository
- `CleanupTestRepo(t *testing.T, dir string)` - Removes temporary repository
- `RunGit(t *testing.T, dir string, args ...string)` - Executes Git commands
- `RunGitFlow(t *testing.T, dir string, args ...string)` - Executes git-flow commands
- `WriteFile(t *testing.T, dir string, name string, content string)` - Creates files
- `BranchExists(t *testing.T, dir string, branch string) bool` - Checks branch existence
- `GetCurrentBranch(t *testing.T, dir string) string` - Gets current branch name
- `AddRemote(t *testing.T, dir, name string, bare bool) (string, error)` - Adds a remote repository
- `SetupTestRepoWithRemote(t *testing.T) (string, string)` - Creates repo with local remote

For detailed examples of creating merge conflicts, setting up remotes, and verifying Git states, see [GIT_TEST_SCENARIOS.md](GIT_TEST_SCENARIOS.md).

## Test Organization

### Directory Structure

```
test/
├── cmd/              # Command-level integration tests
├── internal/         # Internal package unit tests
└── testutil/         # Test utilities and helpers
```

### Error Handling in Tests

Always include comprehensive error checking with detailed failure messages:

```go
output, err := testutil.RunGitFlow(t, dir, "feature", "start", "test-branch")
if err != nil {
    t.Fatalf("Failed to start feature branch: %v\nOutput: %s", err, output)
}
```

### State Verification

Verify both Git state and application-specific state:

```go
// Check Git state
if !testutil.BranchExists(t, dir, "feature/test-branch") {
    t.Error("Expected feature branch to exist")
}

// Check application state
state, err := testutil.LoadMergeState(t, dir)
if state.Action != "finish" {
    t.Errorf("Expected action to be 'finish', got '%s'", state.Action)
}
```

## Best Practices

1. **Always use testutil helpers** - Never execute Git commands directly
2. **Avoid internal package imports in cmd tests** - Don't import `internal/git` or `internal/config` in command tests; use testutil functions instead (see [Working Directory Management](#working-directory-management-critical))
3. **Use long flag variants only** - Don't test both `-f` and `--force`; testing one variant is sufficient since Cobra handles flag parsing. Always use the long form (`--force`, `--defaults`) in tests for readability
4. **Include setup/cleanup** - Use defer to ensure cleanup happens
5. **Test error conditions** - Verify failures behave correctly
6. **Check intermediate state** - Don't just test final outcomes
7. **Use descriptive assertions** - Include context in error messages
8. **Match test setup to test goal** - If testing behavior X, don't create dependencies on unrelated feature Y
9. **Test with remotes when relevant** - Many Git operations behave differently with remotes
10. **Verify test helper implementations** - Don't trust placeholder functions that may not actually call commands
11. **Create conflicts correctly** - Branch first, then add conflicting content (see [GIT_TEST_SCENARIOS.md](GIT_TEST_SCENARIOS.md))
12. **Use Git internal state for verification** - Check `.git/` directory contents for reliable conflict state detection

## Working Directory Management (CRITICAL)

**⚠️ IMPORTANT**: Test commands must run in the test repository directory, not the directory where tests are executed from.

### The Problem

When executing Git or git-flow commands in tests, the commands run in the **current working directory** by default. If you don't explicitly set the working directory, commands will execute in the test runner's directory (e.g., `test/cmd/`) instead of your temporary test repository. This causes:

- Commands operating on the wrong repository
- Flaky tests that depend on global state
- Tests that pass locally but fail in CI
- Hard-to-debug failures

### The Solution: Always Use `cmd.Dir`

The testutil helper functions (`RunGit`, `RunGitFlow`, `RunGitFlowWithInput`) properly set `cmd.Dir` to ensure commands execute in the correct directory:

```go
// From test/testutil/git.go - CORRECT pattern
func RunGit(t *testing.T, dir string, args ...string) (string, error) {
    cmd := exec.Command("git", args...)
    cmd.Dir = dir  // ✓ Ensures command runs in test repo
    // ...
}
```

### ✅ CORRECT: Use testutil Functions

```go
func TestExample(t *testing.T) {
    dir := testutil.SetupTestRepo(t)
    defer testutil.CleanupTestRepo(t, dir)

    // CORRECT: testutil.RunGit sets cmd.Dir internally
    output, err := testutil.RunGit(t, dir, "checkout", "-b", "feature/test")
    if err != nil {
        t.Fatalf("Failed to create branch: %v", err)
    }

    // CORRECT: testutil.RunGitFlow also sets cmd.Dir
    output, err = testutil.RunGitFlow(t, dir, "feature", "finish", "test")
    // ...
}
```

### ❌ AVOID: Using `os.Chdir()` with Internal Functions

Some tests use `os.Chdir()` to change the working directory, then call internal git functions that don't accept directory parameters. This pattern is fragile:

```go
// PROBLEMATIC: Relies on global state
func TestExample(t *testing.T) {
    dir := testutil.SetupTestRepo(t)
    defer testutil.CleanupTestRepo(t, dir)

    originalDir, _ := os.Getwd()
    defer os.Chdir(originalDir)  // Must restore!

    if err := os.Chdir(dir); err != nil {
        t.Fatal(err)
    }

    // These internal functions have no dir parameter
    // They rely on os.Chdir() having been called
    if err := git.Checkout("develop"); err != nil {  // ⚠️ Uses current dir
        t.Fatal(err)
    }
}
```

**Why this is problematic:**
- If `os.Chdir()` fails or defer doesn't run, subsequent tests break
- Test parallelization becomes impossible (shared global state)
- Harder to understand which directory commands execute in

### When `os.Chdir()` Is Acceptable

In some cases, `os.Chdir()` is necessary because internal functions (`internal/git/repo.go`, `internal/git/config.go`) don't accept directory parameters. If you must use this pattern:

1. **Always save and restore** the original directory with defer
2. **Place defer immediately** after the `os.Chdir()` call
3. **Document why** the pattern is necessary
4. **Consider adding `*InDir` variants** to internal functions if this pattern repeats

```go
// If os.Chdir() is unavoidable, do it safely:
originalDir, err := os.Getwd()
if err != nil {
    t.Fatalf("Failed to get working directory: %v", err)
}
if err := os.Chdir(dir); err != nil {
    t.Fatalf("Failed to change to test directory: %v", err)
}
defer func() {
    if err := os.Chdir(originalDir); err != nil {
        t.Errorf("Failed to restore working directory: %v", err)
    }
}()
```

### Known Areas Using `os.Chdir()`

The following test files currently use `os.Chdir()` because they call internal functions without directory parameters:

- `test/cmd/config_test.go` - Calls `config.LoadConfig()`, etc.
- `test/cmd/init_test.go` - Some edge case tests

**Note**: Functions in `internal/git/repo.go` and `internal/git/config.go` don't accept directory parameters. If you need to call these in tests, you must use `os.Chdir()`. Consider whether a testutil helper or `*InDir` variant would be better.

**Refactored**: `test/cmd/update_test.go` was refactored to use only testutil functions, eliminating the need for `os.Chdir()` and internal package imports.

## Test Implementation Anti-Patterns

Common testing mistakes to avoid:

1. **Trusting placeholder functions** - Ensure helper functions actually call the commands being tested
2. **Wrong conflict generation sequence** - Create branches before adding conflicting content
3. **Incorrect rebase state verification** - Check `.git/rebase-merge/` instead of branch name
4. **Testing multiple behaviors with fragile dependencies** - Keep tests focused on their stated purpose

For detailed examples of these anti-patterns and their solutions, see [GIT_TEST_SCENARIOS.md](GIT_TEST_SCENARIOS.md#common-mistakes-to-avoid).

## Test Execution Methods

### Test Binary

The `test/cmd` integration tests execute the git-flow binary as a subprocess. `TestMain` (in `test/cmd/main_test.go`) builds a fresh binary into a temporary directory via `testutil.BuildGitFlow()` before any test runs, so `go test ./...` is self-sufficient — no manual build step is required, and tests can never silently run against a stale binary.

To test a specific prebuilt binary instead (e.g. a release artifact), set the `GIT_FLOW_PATH` environment variable:

```bash
GIT_FLOW_PATH=/path/to/git-flow go test ./test/cmd/
```

### CRITICAL: Use Test Helpers, Not Bash Piping

**IMPORTANT**: When running git-flow commands that require interactive input, always use the `runGitFlowWithInput` helper function instead of bash piping.

#### ❌ BAD: Using Bash Piping

```go
// BAD: Bash piping can cause flaky tests due to process isolation issues
cmd := exec.Command("bash", "-c", fmt.Sprintf("cat %s | %s init", scriptPath, gitFlowPath))
cmd.Dir = dir
err = cmd.Run()
```

**Problems with bash piping:**
1. Creates an extra shell process layer that can affect working directory inheritance
2. May cause intermittent failures that are hard to reproduce
3. Harder to debug when failures occur

#### ✅ GOOD: Using Test Helper

```go
// GOOD: Direct stdin piping through the test helper
input := "custom-main\ncustom-dev\nfeature/\nbugfix/\nrelease/\nhotfix/\nsupport/\nv\n"
output, err := runGitFlowWithInput(t, dir, input, "init")
```

**Benefits:**
1. Direct stdin connection to the git-flow process
2. Consistent working directory handling via `cmd.Dir`
3. Reliable and deterministic test execution

### Working Directory Handling

All test helper functions must set `cmd.Dir` to the test repository directory:

```go
func RunGitFlow(t *testing.T, dir string, args ...string) (string, error) {
    cmd := exec.Command(gitFlowPath, args...)  // Binary built by TestMain
    cmd.Dir = dir  // CRITICAL: Always set working directory
    // ...
}
```

**Why this matters:**
- Git commands run by the git-flow binary inherit the working directory
- Without proper `cmd.Dir` setting, commands may run in the wrong repository
- This can cause tests to pass locally but fail in CI, or vice versa

## Debugging Test Failures

When tests fail, follow this systematic approach:

1. **Check Helper Function Implementations** - Ensure they actually call the intended commands
2. **Verify Conflict Setup** - Confirm branches and files are created in correct sequence
3. **Inspect Git State** - Use `.git/` directory contents to understand actual repository state
4. **Test Configuration Persistence** - Verify that config changes are actually saved and old entries removed
5. **Check Command Flag Logic** - Ensure flag detection properly influences command behavior
6. **Verify Test Execution Method** - Ensure tests use `runGitFlowWithInput` instead of bash piping for interactive commands