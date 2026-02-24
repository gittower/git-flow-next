---
applyTo: "test/**/*.go,**/*_test.go"
---

# Test Review Instructions

## Test Structure

- MUST: One test case per function — no table-driven tests (exception: pure validation functions with simple input/output)
- MUST: Every test function has a structured comment — first line describes what the test validates, then `// Steps:` with numbered list of test actions
- MUST: Use descriptive test names that indicate what is being tested: `TestFinishFeatureBranchWithMergeConflict`
- MUST: Test both success and error conditions for the functionality under test

## Test Setup and Cleanup

- MUST: Use `testutil.SetupTestRepo(t)` and `defer testutil.CleanupTestRepo(t, dir)` for temporary Git repos
- MUST: Initialize with `testutil.RunGitFlow(t, dir, "init", "--defaults")` for most tests
- MUST: Check errors from setup commands with `t.Fatalf` — do not silently ignore setup failures
- SHOULD: Use feature branches for general topic branch testing — they are the most common type
- SHOULD: Only modify config with `git config` commands when the test specifically needs non-default behavior

## Command Execution

- MUST: Use `testutil.RunGitFlow(t, dir, ...)` and `testutil.RunGit(t, dir, ...)` — never `exec.Command("git", ...)` directly in test code
- MUST: Always ensure `cmd.Dir` is set to the test repo directory — never rely on the test runner's working directory
- MUST: Use `runGitFlowWithInput` for interactive commands — never bash piping with `exec.Command("bash", "-c", ...)`
- SHOULD: Use long flag variants (`--force` not `-f`, `--defaults` not `-d`) for readability

## Working Directory

- MUST: Use testutil helper functions that set `cmd.Dir` internally — this is the primary and preferred approach
- MUST: If `os.Chdir()` is unavoidable (calling internal functions without dir parameter), always save and restore the original directory with defer
- SHOULD: Document why `os.Chdir()` is necessary when used
- SHOULD: Avoid importing `internal/git` or `internal/config` in command tests — use testutil functions instead

## Assertions and Verification

- MUST: Include context in assertion error messages — `t.Errorf("Expected branch %s to exist", branchName)` not just `t.Error("branch missing")`
- SHOULD: Verify intermediate state, not just final outcomes — check branch exists before testing finish
- SHOULD: Include both output and error in failure messages: `t.Fatalf("Failed: %v\nOutput: %s", err, output)`
- SHOULD: Use `testutil.BranchExists(t, dir, branch)` for branch existence checks
- SHOULD: Check `.git/` directory contents for reliable conflict and rebase state detection

## Test Isolation and Organization

- MUST: Each test must be independent — no shared state between tests
- MUST: Do not rely on test execution order
- SHOULD: Match test setup to test goal — if testing behavior X, do not create dependencies on unrelated feature Y
- SHOULD: Group related tests in the same file — e.g., all finish-with-conflict tests together
- SHOULD: Test with remotes when the operation under test is remote-aware — many Git operations behave differently with remotes configured

## Scenario Setup

### Conflict Generation

- MUST: Create the topic branch first, then add conflicting content — sequence is: (1) start branch from clean base, (2) add and commit content on topic branch, (3) switch to base branch, (4) add and commit different content to the same file — only then will merge/finish produce a conflict
- MUST: Never add content to base before branching — the topic branch inherits the file and modifying it later is not a conflict, just a fast-forward

### Rebase State Verification

- MUST: Do not use `testutil.GetCurrentBranch()` to verify rebase conflicts — it returns `"HEAD"` during an active rebase, not the branch name
- MUST: Check `.git/rebase-merge/` directory existence to confirm active rebase state
- SHOULD: Read `.git/rebase-merge/head-name` to verify which branch is being rebased — value is a full ref like `refs/heads/feature/branch-name`

### Merge State Verification

- SHOULD: Check `.git/MERGE_HEAD` existence to confirm active merge conflict state
- SHOULD: Check `.git/SQUASH_MSG` existence to confirm squash merge state
- SHOULD: Use `testutil.LoadMergeState(t, dir)` to verify git-flow-next's own state (action, current step, branch info) for multi-step operations
- SHOULD: For multi-step operations, verify both Git state (`.git/MERGE_HEAD`, `.git/rebase-merge/`) and application state (`LoadMergeState`) — they track different things

### Remote Operations

- SHOULD: Use `testutil.AddRemote(t, dir, "origin", true)` to add a local bare repo as remote — remember to defer cleanup of the remote dir
- SHOULD: Use `testutil.SetupTestRepoWithRemote(t)` when the test needs a pre-configured remote from the start
- SHOULD: Simulate remote-ahead scenarios by cloning the bare remote to a second working copy, committing and pushing there, then fetching from the original repo

### Child Branch Update Conflicts

- SHOULD: To test conflicts during the auto-update phase of finish (e.g., develop update after release finish), add conflicting content to the child branch (develop) before finishing the topic branch into its parent (main)

## Anti-Patterns

- MUST: Do not trust placeholder helper functions — verify they actually execute the intended commands
- MUST: Create branches before adding conflicting content — wrong sequence produces incorrect conflict states
- SHOULD: Do not test both short and long flag variants — testing one (long) is sufficient since Cobra handles flag parsing
- NIT: Avoid overly complex test setups that obscure the test's purpose
