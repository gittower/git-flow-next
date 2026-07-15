---
name: triage
description: Triage an external issue or discussion — classify, check duplicates, analyze, propose a verdict, then reply
argument-hint: <issue-number | discussion-number>
allowed-tools: Read, Grep, Glob, Write, Bash, mcp__github__get_issue, mcp__github__search_issues, mcp__github__list_issues, mcp__github__update_issue, mcp__github__add_issue_comment
---

# Triage

Triage an external issue or discussion for gittower/git-flow-next: determine
what it is (bug report, feature request, question), check for duplicates,
analyze it against the codebase, and propose a verdict. Everything up to the
verdict is autonomous; the public reply and any issue changes wait for user
confirmation.

## Arguments

`/triage <issue-number>` or `/triage discussion <number>`

## Instructions

### 1. Fetch the Report

For an **issue**: `mcp__github__get_issue` — title, body, labels, author,
existing comments.

For a **discussion**: fetch via GraphQL:
```bash
gh api graphql -f query='query { repository(owner: "gittower", name: "git-flow-next") {
  discussion(number: <N>) { id title body author { login }
    comments(first: 50) { nodes { body author { login } } } } } }'
```
Keep the discussion `id` — it's needed for replying.

Create the workflow folder `.ai/issue-<number>-<slug>/` (or
`.ai/discussion-<number>-<slug>/`), slug from the title as in
`/analyze-issue`.

### 2. Classify

Determine the type from the content, not the labels:

- **bug** — describes broken or unexpected behavior
- **feature** — requests new or changed functionality
- **question** — asks how to do something; nothing is broken or requested
- **support** — environment/setup problem specific to the reporter

A report can be misfiled (a "bug" that is actually expected behavior is a
question or a docs gap). Classify by what it actually is.

### 3. Check for Duplicates and Prior Art

Search **open and closed** issues via `mcp__github__search_issues` with
several term variations (feature area, command name, error message
fragments). Look for:

- The same bug or request (duplicate)
- An existing `spec` issue that already covers it
- Related issues that partially overlap or provide context
- Previously rejected requests for the same thing (note the rejection
  reasoning)

### 4. Analyze Against the Codebase

- **Bugs**: try to confirm plausibility — find the code path, check whether
  the described behavior can occur, identify the likely root cause with
  file:line references. If reproduction info is missing, note exactly what's
  missing
- **Features**: assess fit with the project's scope and philosophy
  (CLAUDE.md), affected components, and whether existing configuration
  already covers the need
- **Questions**: find the actual answer — in the code, docs/ manpages,
  CONFIGURATION.md. If docs don't cover it, note the docs gap

### 5. Write the Triage Document

Write `.ai/<folder>/triage.md`:

```markdown
# Triage: #<number> — <title>

## Classification
<bug | feature | question | support> — <one line why>

## Duplicate Check
- Searched: <terms used>
- <#N — same/related/spec, or "no duplicates found">

## Analysis
<Bug: plausibility, root cause, affected code. Feature: fit, scope,
affected components. Question: the answer.>

## Verdict (proposed)
<one of:>
- duplicate of #<N>
- accept as bug — next: /create-spec
- accept as feature — next: /create-spec
- reject — <reason>
- answer — <question answered in reply>
- needs info — <what's missing from the reporter>

## Draft Reply
<The complete reply to post, written for the reporter: friendly,
concise, no emojis. Link related/duplicate issues. For rejections,
explain the reasoning honestly. For answers, give the full answer.>

## Actions on Confirmation
- <post reply>
- <apply labels: bug/enhancement>
- <close as duplicate of #N / close as answered / leave open>
```

### 6. Confirmation Gate

Present to the user: classification, duplicate findings, verdict, the full
draft reply, and the list of actions. Wait for confirmation — the user may
change the verdict or edit the reply.

### 7. Execute

After confirmation:

- Post the reply (`mcp__github__add_issue_comment`; for discussions, the
  `addDiscussionComment` GraphQL mutation with the discussion `id`)
- Apply labels / close via `mcp__github__update_issue` as decided
  (duplicates: close with `state_reason: "not_planned"` after the reply
  links the original)

### 8. Report

Summarize the outcome and, for accepted bugs/features, point to the next
step: `/create-spec <number>` — the triage document feeds directly into it.
