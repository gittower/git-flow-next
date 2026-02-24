---
applyTo: "docs/**/*.md"
---

# Documentation Review Instructions

## Update Requirements

- MUST: Update manpages in `docs/` when commands, options, or behavior change
- MUST: Update both `docs/gitflow-config.5.md` and `CONFIGURATION.md` when configuration keys are added, changed, or removed
- MUST: Create new manpage files for new commands
- MUST: Update exit codes and error descriptions when error handling changes

## Manpage Standards

- MUST: Follow Unix manpage section structure: NAME, SYNOPSIS, DESCRIPTION, OPTIONS, EXAMPLES, EXIT STATUS, SEE ALSO, NOTES
- MUST: Include practical examples for all new functionality
- SHOULD: Use bold for command names, options, and important terms
- SHOULD: Use italics for parameters, placeholders, and emphasis
- SHOULD: Use code blocks for configuration examples and commands

## Naming and Structure

- SHOULD: Name command manpages as `git-flow-<command>.1.md` (section 1) and configuration manpages as `gitflow-<topic>.5.md` (section 5)
- SHOULD: Update `docs/README.md` command listing when adding or renaming manpage files

## Cross-References

- SHOULD: Update SEE ALSO sections in related manpages when adding or renaming commands
- SHOULD: Keep cross-references bidirectional — if A references B, B should reference A

## Examples

- MUST: Provide working examples that match the current implementation
- SHOULD: Test all examples for accuracy before committing
- SHOULD: Include examples for common use cases and edge cases

## Workflow Considerations

- SHOULD: Consider impact on all supported workflows (Classic GitFlow, GitHub Flow, GitLab Flow, Custom)
- NIT: Include tips and notes for workflow-specific behavior differences
