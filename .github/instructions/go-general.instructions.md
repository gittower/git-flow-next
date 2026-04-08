---
applyTo: "**/*.go"
---

# Go Code Review Instructions

## Error Handling

- MUST: Never ignore errors — no `_ = someFunction()` patterns
- MUST: Use custom error types from `internal/errors` with specific exit codes
- MUST: Wrap errors with context — include operation description and relevant identifiers (branch name, config key, etc.)
- MUST: Return specific error types for different failure conditions (e.g., `BranchNotFoundError`, `MergeConflictError`, `NotInitializedError`)
- SHOULD: Use `fmt.Errorf()` with `%w` verb for error wrapping when structured types are not needed

## Git Operations

- MUST: All Git operations go through `internal/git/repo.go` wrappers — never `exec.Command("git", ...)` directly in command code
- MUST: Handle merge conflicts gracefully with proper state persistence
- MUST: Check for uncommitted changes before destructive operations
- SHOULD: Handle missing config sections gracefully — `git config --remove-section` returns exit 128 if section does not exist

## Configuration

- MUST: Load configuration once at command start with `config.LoadConfig()`, pass `cfg` to functions — never call `LoadConfig()` multiple times
- MUST: Use pointer types (`*bool`, `*string`) for optional config values — nil means "not set, use default"
- MUST: Implement three-layer config precedence — Layer 1 branch type defaults -> Layer 2 command-specific git config -> Layer 3 CLI flags (always win)

## Platform Awareness

- MUST: Use `runtime.GOOS` checks for platform-specific behavior — never assume Unix
- MUST: Use `filepath.Join()` for path construction — never hardcode path separators
- SHOULD: Test platform-sensitive code paths on Windows (file permissions, script execution, path handling)

## State Management

- MUST: Use JSON-serialized state (`mergestate.MergeState`) for multi-step operations that can be interrupted
- MUST: Include all information needed to resume operations in state structs

## Naming and Style

- SHOULD: PascalCase for exported, camelCase for private; descriptive verb names for functions
- SHOULD: Three import groups: (1) standard library, (2) third-party, (3) local packages
- MUST: Write error output to stderr, progress and success messages to stdout
- MUST: Run `go fmt` — code must be properly formatted
- SHOULD: Validate inputs early before performing operations

## Compatibility

- MUST: Preserve backward compatibility with git-flow-avh config keys, hook arguments, and branch naming
- SHOULD: Reuse existing helpers (e.g., `getBoolFlag`) instead of duplicating flag/config parsing logic
