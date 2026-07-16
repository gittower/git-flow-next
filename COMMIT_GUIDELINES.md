# Commit Guidelines

This document establishes standards for commit messages in the git-flow-next project to ensure clear, consistent, and informative version history.

## Commit Message Format

Use the following structure for commit messages:

```
<type>[(<scope>)]: <subject>
                                        ← blank line
<body>
                                        ← blank line
<footer>
```

### Subject Line (Required)

- **Length**: Maximum 50 characters (including type and scope)
- **Case**: Use sentence case (first word capitalized)
- **Tense**: Use imperative mood ("Add feature" not "Added feature")
- **Punctuation**: No period at the end
- **Format**: `<type>: <description>` or `<type>(<scope>): <description>`

### Scope (Optional)

- **Purpose**: Narrow the area affected when the type alone is too broad
- **Values**: Use the primary area affected — `finish`, `start`, `config`, `init`, `publish`, etc.
- **Keep short**: One word, lowercase, no spaces
- **When to use**: When the change targets a specific command or subsystem and the subject doesn't already make it obvious
- **When to omit**: For cross-cutting changes, documentation-only changes, or when the subject is clear enough without it

### Body (Optional but Recommended)

- **Length**: Write body text as continuous flowing prose first, then hard wrap at 72 characters. Each line should fill close to the 72-character limit — don't wrap early at 55-65 characters
- **Purpose**: Explain the "what" and "why", not the "how"
- **Format**: Use paragraphs separated by blank lines
- **Lists**: Use bullet points for multiple related changes

### Footer (Optional)

- **References**: Link to issues, pull requests, or breaking changes
- **Format**: `Resolves #123`, `Closes #456`, `Refs #789`
- **AI co-authorship**: Commits created with AI agent support must include a `Co-Authored-By` trailer crediting the agent (see [AI-Assisted Commits](#ai-assisted-commits))

## Commit Types

### Primary Types

- **feat**: New feature or functionality
- **fix**: Bug fix or correction (user-facing issues only — not for internal tooling, CI, or infrastructure fixes; use the appropriate type instead)
- **refactor**: Code restructuring without changing functionality
- **perf**: Performance improvements
- **test**: Adding or modifying tests (including fixes to tests — use `test:` not `fix:` for test corrections)
- **docs**: Documentation changes
- **style**: Code formatting, whitespace, or style changes

### Secondary Types

- **build**: Build system or dependency changes
- **ci**: Continuous integration configuration, workflows, and internal automation (including fixes to CI scripts and skills)
- **chore**: Maintenance tasks, tool updates
- **revert**: Reverting previous commits

## Examples

### Feature Addition
```
feat: Add support for custom merge strategies in finish command

Implements configurable merge strategies per branch type allowing users to specify merge, rebase, or squash operations. The configuration follows the existing pattern of branch-specific settings and supports both command-line overrides and git config defaults.

- Add merge strategy validation in config loader
- Update finish command to respect strategy settings  
- Add comprehensive tests for all strategy combinations

Closes #234
```

### Bug Fix
```
fix: Resolve state file corruption during interrupted operations

Fixes issue where merge state file could become corrupted if the process was interrupted during JSON serialization. The fix adds atomic write operations using temporary files and proper error handling for filesystem issues.

Resolves #456
```

### Refactoring
```
refactor: Extract tag creation logic to git module

Moves createTagWithOptions and createTag functions from cmd/finish.go to internal/git/repo.go for better separation of concerns. Combines both functions into a single git.CreateTag function with optional parameters, reducing finish command complexity by 36 lines and improving reusability across commands.

- Consolidate tag creation into single function with options struct
- Remove duplicate code and os/exec dependency from finish command
- Add better error messages and validation for tag operations
```

### Test Addition
```
test: Add comprehensive test for consecutive conflicts in multi-step operations

Implements TestFinishWithConsecutiveConflicts which validates the system's ability to handle multiple conflicts during a single finish operation: first conflict between release branch and main during merge, second conflict between develop branch and main during auto-update. Tests state persistence across conflict resolutions, validates proper error messages and recovery workflow, and ensures complete cleanup after successful resolution.
```

### Documentation
```
docs: Update testing guidelines with default configuration details

Adds comprehensive documentation about git-flow default branches and settings to help developers write consistent tests. Includes branch relationships, merge strategies, and examples of proper test setup using git-flow defaults rather than custom configurations.
```

## Best Practices

### Do's

- **Be specific**: Describe exactly what changed and why it matters
- **Use active voice**: "Add feature" instead of "Feature was added"
- **Reference issues**: Always link to relevant issue numbers
- **Focus on impact**: Explain the user-facing or system-level benefits
- **Group related changes**: Combine logically related changes in single commits
- **Test before committing**: Ensure all tests pass and code compiles
- **Credit AI assistance**: Add a `Co-Authored-By` trailer when an AI agent contributed to the commit

### Don'ts

- **Don't use vague subjects**: Avoid "Fix bug" or "Update code"
- **Don't exceed line limits**: Keep subject under 50 chars, body under 72
- **Don't mix concerns**: Separate unrelated changes into different commits
- **Don't include file lists**: Git tracks files automatically
- **Don't exceed 72 characters per line**: Hard wrap the body at 72 columns
- **Don't commit broken code**: Each commit should represent a working state
- **Don't use free-form AI attribution**: Credit AI agents only via the structured `Co-Authored-By` trailer, not prose footers like "Generated with Claude Code"

## Special Cases

### AI-Assisted Commits

When an AI agent contributed to a commit, always credit it with a
`Co-Authored-By` trailer in the footer, using the agent's name and a
contact address identifying the tool:

```
test: Add stale merge state regression tests

Adds end-to-end tests for the stale state scenarios where a conflict
is resolved with a plain git commit instead of --continue.

Refs #110

Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>
```

The trailer is the only place AI involvement should appear — do not add
prose attribution to the subject or body.

### Breaking Changes

For breaking changes, add a footer explaining the impact:

```
feat: Change configuration file format to YAML

Migrates configuration from JSON to YAML format for better readability and comments support. Existing JSON configurations are automatically migrated on first run.

BREAKING CHANGE: Configuration files must be migrated from config.json to config.yaml format. Migration is automatic but requires manual review of settings.
```

### Work in Progress

For temporary commits during development:

```
wip: Implement basic tag creation logic

Partial implementation of tag creation functionality. Still needs error handling and testing.
```

Note: WIP commits should be squashed before merging to main.

### Revert Commits

```
revert: "feat: Add experimental batch processing"

This reverts commit abc1234 due to performance regression in large repositories. The feature will be reimplemented with better memory management.

Refs #567
```

## Validation

Before committing, verify your message:

1. **Subject is clear and specific** (≤50 characters)
2. **Body explains context and reasoning** (when needed)
3. **Type matches the actual change**
4. **All tests pass** and code compiles
5. **Related issues are referenced**

## Tools and Automation

Consider using:

- **Git hooks**: Validate commit message format automatically
- **Conventional commits**: Tools like commitizen for guided commits
- **Issue linking**: Automatic issue linking in GitHub/GitLab
- **Commit templates**: Set up `.gitmessage` template for consistency

## References

- [Conventional Commits](https://www.conventionalcommits.org/)
- [How to Write a Git Commit Message](https://chris.beams.io/posts/git-commit/)
- [Angular Commit Message Guidelines](https://github.com/angular/angular/blob/main/CONTRIBUTING.md#commit)