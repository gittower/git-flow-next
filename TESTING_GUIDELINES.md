# Testing Guidelines

This document outlines the testing conventions and patterns used in the git-flow-next project.

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

Add a comment above each test function that:
1. Describes what the test validates
2. Lists the steps the test will perform
3. Outlines the expected outcome

### Comment Pattern

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

## Temporary Git Repository Testing

All tests use temporary Git repositories created through test utilities.

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

## Creating Git Scenarios for Tests

### Merge Conflicts

To create merge conflicts for testing:

1. **Both Modified Conflict (Preferred)**:
   - Create a file with initial content on the base branch
   - Create a new branch and modify the file
   - Switch back to base branch and modify the same file differently
   - Attempting to merge will create a conflict

```go
// Create feature branch
output, err := testutil.RunGitFlow(t, dir, "feature", "start", "conflict-test")

// Create file in feature branch
testutil.WriteFile(t, dir, "test.txt", "feature content")
_, err = testutil.RunGit(t, dir, "add", "test.txt")
_, err = testutil.RunGit(t, dir, "commit", "-m", "Add test.txt in feature")

// Switch to develop and create conflicting content
_, err = testutil.RunGit(t, dir, "checkout", "develop")
testutil.WriteFile(t, dir, "test.txt", "develop content")
_, err = testutil.RunGit(t, dir, "add", "test.txt")
_, err = testutil.RunGit(t, dir, "commit", "-m", "Add test.txt in develop")

// Attempt to finish - should fail with conflict
output, err = testutil.RunGitFlow(t, dir, "feature", "finish", "conflict-test")
```

2. **Multiple Rebase Conflicts**:
   - Repeat the above scenario with additional commits and files
   - Each conflicting commit will create a separate rebase step

### Using Remotes

To test remote functionality:

1. **Create Bare Remote Repository**:
   - Use `testutil.AddRemote()` to create a local bare repository
   - Add it as a remote using the file path
   - Push branches to establish tracking

```go
// Add a remote repository
remoteDir, err := testutil.AddRemote(t, dir, "origin", true)
if err != nil {
    t.Fatalf("Failed to add remote: %v", err)
}
defer testutil.CleanupTestRepo(t, remoteDir)

// Create feature branch and make changes
output, err = testutil.RunGitFlow(t, dir, "feature", "start", "fetch-test")
testutil.WriteFile(t, dir, "feature.txt", "feature content")
_, err = testutil.RunGit(t, dir, "add", "feature.txt")
_, err = testutil.RunGit(t, dir, "commit", "-m", "Add feature file")

// Test operations with remote (fetch, push, etc.)
output, err = testutil.RunGitFlow(t, dir, "feature", "finish", "fetch-test", "--fetch")
```

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
2. **Include setup/cleanup** - Use defer to ensure cleanup happens
3. **Test error conditions** - Verify failures behave correctly
4. **Check intermediate state** - Don't just test final outcomes
5. **Use descriptive assertions** - Include context in error messages
6. **Test with remotes when relevant** - Many Git operations behave differently with remotes