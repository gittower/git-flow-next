# Issue Guidelines

Standards for writing clear, useful GitHub issues in the git-flow-next project.

## General Principles

Tone, formatting, and the general posting rules — lead with a summary, be concise, no emojis, minimal formatting, `###` sections, no hard-wrapping — live in [GITHUB_GUIDELINES.md](GITHUB_GUIDELINES.md), the single source of truth for all GitHub content. This document covers only the conventions specific to issues: titles, labels, body structure, and spec issues.

## Title

An issue describes a problem or request — a state, not a change — so titles
read differently from commit and PR titles.

- Clear, specific, and searchable — understandable without opening the issue
- **No type prefix** — the [label](#labels) carries the type, so
  `bug:`, `Feature request:`, or `Question:` prefixes are redundant noise
- An **area prefix** is fine when it aids scanning, e.g.
  `release finish: cannot finish after resolving a conflict`
- **Bug reports**: describe the observed problem, not a presumed fix — you may
  not know the fix yet (`release finish fails after resolving a conflict`,
  not `Fix release finish conflict`)
- **Feature requests**: imperative reads naturally
  (`Add --dry-run option to finish command`)

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

### Breakdown into Sub-Specs

Break a spec into sub-specs when it can't be delivered as a single
reviewable PR. Each sub-spec is itself a complete spec (own goal, behavior,
test scenarios) and independently implementable. Attach each as a native
GitHub sub-issue of the parent spec, and keep a human-readable task list in
the parent linking them:

```
### Breakdown
- [ ] #<n> — <sub-spec title>
- [ ] #<n> — <sub-spec title>
```

### Relation to User Reports

A user report (bug or feature, often from someone else) stays open as the
durable, reporter-facing record. When a report is accepted, the spec is
created as a **separate issue** and attached to the report as a native GitHub
sub-issue — the report is the parent, the spec is the child — with the spec
body also carrying "Refs #N". The report stays the place to follow progress;
the spec is the implementation source of truth beneath it.

The report closes when the spec ships — when the fix merges to `main`, not when
it is released. The fix PR carries "Resolves #<spec>", which closes the spec
automatically, but GitHub does **not** cascade that to the parent: the report
has to be closed explicitly, with a short note saying what shipped and pointing
at the PR and merge commit (name the version too, once there is one). Nesting is
expected — a report parents a spec, which in turn parents its sub-specs, and the
same rule applies at every level.

This is easy to miss precisely because the spec closes itself and the parent
silently does not, leaving shipped work sitting in the open-issue list. Any
workflow that merges a spec's PR owns closing the report as part of that merge —
see the merge steps in [/ship-issue](.claude/skills/ship-issue/SKILL.md) and
[/quick-fix](.claude/skills/quick-fix/SKILL.md).

## What to Avoid

The general list (emojis, h1/h2 headings, hard-wrapping, walls of text, restating the title) is in [GITHUB_GUIDELINES.md](GITHUB_GUIDELINES.md). Issue-specific:

- Overly templated issues where most sections are "N/A" — use only the sections that are relevant
