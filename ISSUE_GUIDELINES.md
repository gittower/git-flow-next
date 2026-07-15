# Issue Guidelines

Standards for writing clear, useful GitHub issues in the git-flow-next project.

## General Principles

- **Be brief and concise** — get to the point quickly
- **Lead with a summary** — open with 1-2 sentences explaining what this issue is about before any sections
- **No emojis** — keep the tone professional and scannable
- **Minimal formatting** — use plain text where possible; don't over-format with bold, blockquotes, or horizontal rules
- **Use h3 (`###`) for sections** — not h1 or h2, which compete with the issue title
- **Tables and code blocks are fine** — when they make things clearer, not for decoration

## Title

- Clear and descriptive
- Imperative mood ("Add...", "Fix...", "Update...")
- Specific enough to understand without opening the issue

Good: `Add --dry-run option to finish command`
Bad: `Feature request: dry run`

## Labels

Apply one of these labels when creating an issue:

- `bug` — something isn't working as expected
- `enhancement` — improvement to existing functionality or new feature
- `spec` — an accepted, implementation-ready specification (see [Spec Issues](#spec-issues))

## Body Structure

Start with a brief summary paragraph, then use `###` sections as needed. Not every issue needs all sections — use only what's relevant.

### Bug Reports

```
<Brief summary of what's broken and when it happens>

### Steps to Reproduce
1. ...
2. ...

### Expected Behavior
<What should happen>

### Actual Behavior
<What happens instead>

### Environment
- git-flow-next version:
- OS:
- Git version:
```

### Enhancement / Feature Requests

```
<Brief summary of what you want and why>

### <Sections as needed>
Use sections to break down details: proposed behavior, examples,
commands to support, edge cases, etc. Keep it scannable.
```

Describe the expected behavior as concretely as possible — example commands, expected output, or before/after comparisons help reviewers understand the proposal without ambiguity.

## Example

A well-structured enhancement issue — brief intro, h3 sections, a priority table, and code examples showing proposed usage and output:

> **Title:** Add `--dry-run` option to commands that perform more complex actions
>
> **Body:**
>
> Add a `--dry-run` / `-n` flag to commands that perform complex or destructive actions. When enabled, the command displays all operations that would be performed without actually executing them.
>
> ### Commands to Support
>
> | Priority | Command | Reason |
> |----------|---------|--------|
> | Critical | `finish` | Multi-stage: merge, tag, update children, delete |
> | High | `delete` | Permanently removes local/remote branches |
> | Medium | `publish` | Network operation, pushes to remote |
> | Medium | `update` | Merge/rebase operations, modifies history |
> | Low | `start` | Safe operation, but useful for preview |
>
> ### Example Usage
>
> ```bash
> # Preview what finish will do
> git flow feature finish my-feature --dry-run
>
> # Short flag (follows Git conventions)
> git flow release finish v1.0 -n
> ```
>
> ### Example Output
>
> ```
> Dry-run: git flow feature finish my-feature
> Target: feature/my-feature
> --------------------------------------------------
>
> Actions that would be performed:
>
>   1. Checkout parent branch 'develop'
>       $ git checkout develop
>   2. Merge 'feature/my-feature' into 'develop' using merge strategy
>       $ git merge feature/my-feature
>   3. [!] Delete local branch 'feature/my-feature'
>       $ git branch -d feature/my-feature
>
> No changes were made (dry-run mode).
> ```

## Spec Issues

A spec issue is the accepted, implementation-ready form of a feature or bug
fix. It is the single source of truth for what will be built: implementation
starts from a spec issue, and the resulting PR is verified against it. Specs
live in GitHub issues so they are public, linkable, and reviewable; working
artifacts (analysis, plans, reports) stay local in `.ai/`.

A spec describes the change at **concept level** — what to achieve and what
changes, with technical detail only where it's required to remove ambiguity.
How to implement it belongs to the implementation phase, not the spec. The
goal is a readable concept that a maintainer can verify and accept as good.

**Test scenarios are the centerpiece.** Every spec must spell out the
scenarios that exercise the new or changed behavior and the expected outcome
of each. If the test scenarios are complete and correct, the rest of the spec
follows from them.

### Structure

Start with a brief summary, then use these `###` sections. Goal, expected
behavior, and test scenarios are required; the rest only when relevant.

```
<Brief summary: what this spec covers and why. Link the originating
user report with "Refs #N" if one exists.>

### Goal
<What we want to achieve, in one short paragraph>

### Expected Behavior
<Concrete description of the behavior after the change — example
commands, expected output, before/after comparisons>

### Test Scenarios
<The heart of the spec. One entry per scenario:>

1. <Setup / precondition> — <action> — <expected outcome>
2. ...

<Cover the happy path, error conditions, and edge cases. Each scenario
must be concrete enough to be turned into a test without guessing.>

### Out of Scope
<What this spec deliberately does not cover>

### Technical Notes
<Only what's needed to remove ambiguity: affected components, config
keys, compatibility constraints. Not an implementation plan.>
```

### Sub-Issues

Break a spec into sub-issues when it can't be delivered as a single
reviewable PR. Each sub-issue is itself a complete spec (own goal, behavior,
test scenarios) and independently implementable. The parent spec keeps the
overall goal and a task list linking the sub-issues:

```
### Breakdown
- [ ] #<n> — <sub-issue title>
- [ ] #<n> — <sub-issue title>
```

### Relation to User Reports

User-reported issues stay open as the conversation with the reporter. When a
report is accepted, the spec is created as a **separate issue** with the
full detail, cross-linked in both directions ("Refs #N"). The user issue is
closed when the spec ships.

## What to Avoid

- Wall-of-text descriptions with no structure
- Overly templated issues where most sections are "N/A"
- Emoji-heavy formatting
- h1/h2 headings in the body
- Restating the title in the first line
