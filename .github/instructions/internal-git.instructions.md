---
applyTo: "internal/git/**/*.go"
---

# Git Package Review Instructions

## Wrapper Pattern

- MUST: Each new Git operation gets a dedicated function in `repo.go` — callers never build `exec.Command("git", ...)` themselves
- MUST: Capture and return command output in error messages so callers have diagnostic context
- MUST: Wrap underlying `exec.ExitError` with `%w` so callers can inspect the cause
- SHOULD: Include relevant identifiers (branch name, remote name, ref) in error messages

## Error Handling

- MUST: Use `cmd.CombinedOutput()` or `cmd.Output()` and include trimmed output in error messages
- MUST: Distinguish between exit code errors (expected Git failures) and other errors (system-level failures)
- SHOULD: Return typed errors for well-known failure modes (conflict detected, branch not found, not a git repo)

## Platform Compatibility

- MUST: Use `filepath.Join()` for all path construction — never hardcode `/` separators
- MUST: Use `runtime.GOOS` guards for platform-specific behavior (file permissions, script execution)
- SHOULD: On Windows, execute shell scripts via `sh` (shipped with Git for Windows) since `exec.Command` cannot run shebangs directly
- SHOULD: Account for NTFS limitations — no Unix permission bits, case-insensitive paths

## Compatibility

- MUST: Preserve behavior compatibility with git-flow-avh — same config key names, same hook argument order, same branch naming defaults
- SHOULD: Handle both `main` and `master` as potential production branch names
