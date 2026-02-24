---
applyTo: "cmd/**/*.go"
---

# Command Implementation Review Instructions

## Three-Layer Command Pattern

- MUST: Follow the three-layer pattern — Layer 1: Cobra RunE handler (parse flags, call wrapper) -> Layer 2: Command wrapper (convert errors to exit codes, print to stderr, os.Exit) -> Layer 3: Execute function (all business logic, returns error)
- MUST: Keep all business logic in the execute function (Layer 3) — Cobra handlers should only parse flags and call the wrapper
- MUST: Return structured error types from execute functions — never call `os.Exit()` or `fmt.Fprintf(os.Stderr, ...)` directly from Layer 3

## Initialization and Validation

- MUST: Check `config.IsInitialized()` at the start of every command except `init`
- MUST: Return `errors.NotInitializedError{}` when git-flow is not initialized
- MUST: Validate branch type exists in config before operations — check `cfg.Branches[branchType]`
- MUST: Validate required inputs early — check for empty branch names, invalid types before performing git operations

## Configuration Precedence

- MUST: Implement three-layer config precedence — branch defaults (Layer 1) -> git config overrides (Layer 2) -> CLI flags (Layer 3, always win)
- MUST: Use pointer types (`*bool`) for CLI flag values to distinguish "not set" from "set to false"
- MUST: Check layers in order — start with branch config default, apply command-specific config, then apply CLI flags
- MUST: When adding a new config option, always add Layer 3 (CLI flag) and Layer 2 (`gitflow.<type>.<command>.<option>`) — only add Layer 1 (`gitflow.branch.<type>.<property>` in `BranchConfig`) if it defines a branch type identity or process characteristic
- SHOULD: Only add Layer 1 support for options that define a branch type's identity or process characteristic — operational settings like `fetch`, `sign`, `keep` belong in Layer 2 + Layer 3 only

## Branch Name Handling

- SHOULD: Construct full branch name using prefix when the provided name is not already prefixed
- SHOULD: Detect current branch when no name argument is provided — use `git.GetCurrentBranch()`
- SHOULD: Verify branch exists before performing operations on it

## Flag Design

- SHOULD: Group related flags into option structs (e.g., `TagOptions`, `BranchRetentionOptions`)
- SHOULD: Use consistent command function signatures: `func CommandName(branchType, name string, options...)`
- NIT: Include both long and short flag variants for frequently used flags
- NIT: Include `Example` and `Use` fields in Cobra command definitions for `--help` output

## Multi-Step Operations

- SHOULD: Implement multi-step commands (like finish) as step-based state machines with defined step sequences — persist state via `mergestate.MergeState` so operations can resume with `--continue` or cancel with `--abort` after conflicts

## Dynamic Command Registration

- SHOULD: Register branch-type commands dynamically through `cmd/topicbranch.go` based on configuration — do not hardcode commands for individual branch types

## CLI Mode Detection

- SHOULD: Detect explicit configuration flags to determine interactive vs non-interactive mode
- SHOULD: Apply mode precedence: explicit config flags (non-interactive with overrides) > `--defaults` (non-interactive with defaults) > interactive prompting
