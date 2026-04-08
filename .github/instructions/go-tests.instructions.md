---
applyTo: "test/**/*.go,**/*_test.go"
---

# Test Review Instructions

## Anti-Patterns

- MUST: Do not trust placeholder helper functions — verify they actually execute the intended commands
- MUST: Create branches before adding conflicting content — wrong sequence produces incorrect conflict states
- MUST: Do not use `testutil.GetCurrentBranch()` to verify rebase conflicts — it returns `"HEAD"` during an active rebase
- SHOULD: Do not test both short and long flag variants — testing one (long) is sufficient since Cobra handles flag parsing

## Test Structure

- MUST: One test case per function — no table-driven tests (exception: pure validation functions with simple input/output)
- MUST: Every test function has a structured comment — first line describes what the test validates, then `// Steps:` with numbered list of test actions
- MUST: Use descriptive test names: `TestFinishFeatureBranchWithMergeConflict`
- MUST: Test both success and error conditions for the functionality under test

## Test Setup and Cleanup

- MUST: Use `testutil.SetupTestRepo(t)` and `defer testutil.CleanupTestRepo(t, dir)` for temporary Git repos
- MUST: Initialize with `testutil.RunGitFlow(t, dir, "init", "--defaults")` for most tests
- MUST: Check errors from setup commands with `t.Fatalf` — do not silently ignore setup failures
- SHOULD: Use feature branches for general topic branch testing
- SHOULD: Only modify config when the test specifically needs non-default behavior

## Command Execution

- MUST: Use `testutil.RunGitFlow(t, dir, ...)` and `testutil.RunGit(t, dir, ...)` — never `exec.Command` directly
- MUST: Use `runGitFlowWithInput` for interactive commands — never bash piping
- MUST: Use testutil helpers that set `cmd.Dir` internally — never rely on the test runner's working directory
- MUST: If `os.Chdir()` is unavoidable, save and restore the original directory with defer

## Assertions

- MUST: Include context in error messages — `t.Errorf("Expected branch %s to exist", name)` not just `t.Error("missing")`
- SHOULD: Verify intermediate state, not just final outcomes — check branch exists before testing finish
- SHOULD: Use `testutil.BranchExists(t, dir, branch)` for branch existence checks

## Scenario Setup

- MUST: For conflicts: (1) start branch from clean base, (2) commit on topic branch, (3) switch to base, (4) commit different content to same file
- SHOULD: Check `.git/rebase-merge/` for active rebase state, `.git/MERGE_HEAD` for merge conflicts
- SHOULD: Use `testutil.LoadMergeState(t, dir)` for git-flow-next's own state — verify both Git state and application state
- SHOULD: Use `testutil.SetupTestRepoWithRemote(t)` when testing remote-aware operations
- SHOULD: Simulate remote-ahead by cloning bare remote to second copy, committing and pushing there, then fetching

## Test Isolation

- MUST: Each test must be independent — no shared state between tests, no reliance on execution order
- SHOULD: Group related tests in the same file
- SHOULD: Avoid importing `internal/git` or `internal/config` in command tests — use testutil instead
