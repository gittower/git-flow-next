---
applyTo: "**/*.go"
---

# Go Code Review Instructions

## Error Handling

- MUST: Never ignore errors — no `_ = someFunction()` patterns
- MUST: Use custom error types from `internal/errors` with specific exit codes
- MUST: Wrap errors with context using structured error types — include operation description
- MUST: Return specific error types for different failure conditions (e.g., `BranchNotFoundError`, `MergeConflictError`, `NotInitializedError`)
- SHOULD: Use `fmt.Errorf()` with `%w` verb for error wrapping when structured types are not needed
- SHOULD: Write clear, actionable error messages that tell the user how to recover

## Git Operations

- MUST: All Git operations go through `internal/git/repo.go` wrappers — never `exec.Command("git", ...)` directly in command code
- MUST: Handle merge conflicts gracefully with proper state persistence
- MUST: Check for uncommitted changes before destructive operations
- SHOULD: Clean up old git config entries before saving new ones during rename/delete operations
- SHOULD: Handle missing config sections gracefully — `git config --remove-section` returns exit status 128 if section does not exist

## Configuration

- MUST: Load configuration once at command start with `config.LoadConfig()`, pass `cfg` to all functions — never call `LoadConfig()` multiple times within a command
- MUST: Use pointer types (`*bool`, `*string`) for optional command-line config values — nil means "not set, use default"
- MUST: Implement three-layer config precedence — Layer 1 branch type defaults -> Layer 2 command-specific git config -> Layer 3 CLI flags (always win)
- SHOULD: Document when `nil` has special meaning in struct fields

## Import Organization

- SHOULD: Three import groups separated by blank lines: (1) standard library, (2) third-party, (3) local packages
- SHOULD: Alphabetical order within each group

## Naming Conventions

- SHOULD: PascalCase for exported functions and types, camelCase for private
- SHOULD: Use descriptive verb names for functions: `ValidateBranchName()`, `CreateTag()`
- SHOULD: Use descriptive variable names: `parentBranch` not `parent`, `branchName` not `name`
- SHOULD: Suffix types with purpose when helpful: `BranchRetentionOptions`, `MergeState`
- NIT: Avoid abbreviations except well-known ones (`cfg`, `err`)

## Struct Definitions

- SHOULD: Group related fields together
- SHOULD: Use `*bool` pointer types for optional boolean config values where nil means "use default"
- SHOULD: Document exported struct fields with comments
- NIT: Align struct field comments for readability

## Package Organization

- SHOULD: Use single-word, lowercase package names: `config`, `git`, `errors`, `util`
- SHOULD: One file per major command in `cmd/`
- SHOULD: Group related functionality within packages

## State Management

- MUST: Use JSON-serialized state (`mergestate.MergeState`) for multi-step operations that can be interrupted
- MUST: Include all information needed to resume operations in state structs
- SHOULD: Use JSON tags for serialization on all state struct fields

## Output

- MUST: Write error output to stderr (`fmt.Fprintf(os.Stderr, ...)`)
- SHOULD: Write progress and success messages to stdout
- SHOULD: Include context in success messages (branch name, count of updated branches, etc.)

## Code Style

- MUST: Run `go fmt` — code must be properly formatted
- SHOULD: Run `go vet` for static analysis in addition to `go fmt`
- SHOULD: Keep functions focused on a single responsibility
- SHOULD: Validate inputs early in execute functions before performing operations
- NIT: Constants should use groups with descriptive prefixes for related values
- NIT: Use PascalCase for exported constants, camelCase for unexported

## Documentation in Code

- SHOULD: Document all exported functions with Go doc comments describing purpose and behavior
- SHOULD: Provide package-level doc comments at the top of main package files (e.g., `// Package config provides git-flow configuration management.`)
- SHOULD: Document complex algorithms and state machines with step-by-step comments describing the sequence and resumability

## Performance

- SHOULD: Minimize Git subprocess invocations — cache configuration lookups within a command execution
- SHOULD: Avoid unnecessary string allocations in hot paths
- NIT: Prefer efficient data structures for branch operations
