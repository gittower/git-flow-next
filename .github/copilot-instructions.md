# GitHub Copilot Instructions for git-flow-next

git-flow-next is a modern Go CLI implementing the git-flow branching model. It manages feature, release, and hotfix branches through a unified, configuration-driven command structure.

## Development Philosophy

- Follow a pragmatic, anti-over-engineering approach — solve real problems, not theoretical ones
- Prefer explicit over implicit, readable over clever
- Complex functions with many parameters are acceptable when they reflect real problems
- Use option structs when they group related concepts (e.g., `TagOptions`), not just to reduce parameter counts
- Reject unnecessary complexity, premature abstractions, and excessive layering

## Architecture

- All commands follow a three-layer pattern: Cobra handler (parse flags) -> Command wrapper (error/exit) -> Execute function (business logic)
- Branch types are configuration-driven — behavior defined via Git config under `gitflow.*`
- Three-layer configuration precedence: branch type defaults -> git config overrides -> CLI flags (always win)
- All Git operations go through `internal/git/repo.go` wrappers — never call `exec.Command("git", ...)` directly in command code
- Maintain backward compatibility with git-flow-avh repositories — import AVH configuration and preserve default branch naming conventions

## Commit Messages

- Format: `<type>[(<scope>)]: <subject>`
- Use imperative mood, sentence case, no trailing period
- Subject line maximum 50 characters
- Types: feat, fix, refactor, perf, test, docs, style, build, ci, chore, revert
- Scope: single lowercase word (e.g., finish, config, init) — omit when the subject already makes the area obvious
- Body wrapped at 72 characters, lines filled close to the limit — explain "what" and "why", not "how"
- Reference issues in footers: `Resolves #123`, `Closes #456`, `Refs #789`
- Include `BREAKING CHANGE:` footer when introducing breaking changes
- Never include AI attribution footers (e.g., "Generated with Claude Code")
- Separate unrelated changes into different commits — each commit should represent a working state

## Security

- Never pass unsanitized user input to shell commands
- Never hardcode credentials or secrets
- Validate all input at system boundaries

## Mandatory Change Requirements

- Create or adjust tests for new functionality
- Run `go test ./...` before committing
- Update documentation in `docs/` for any command, option, or behavior changes

## Version Updates

When updating version information, ensure both locations are updated:
- `version/version.go` — Core version constant
- `cmd/version.go` — Version variable used by the version command
