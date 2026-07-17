---
name: create-spec
description: Create a Codex-gated spec issue from a triaged report, concept, or idea
argument-hint: <issue-number | description>
allowed-tools: Read, Grep, Glob, Write, Bash, Agent, mcp__github__get_issue, mcp__github__search_issues, mcp__github__create_issue, mcp__github__update_issue, mcp__github__add_issue_comment
---

# Create Spec

Turn an accepted bug report, feature request, concept, or idea into an
implementation-ready **spec issue** per the Spec Issues section of
ISSUE_GUIDELINES.md. The spec is drafted locally, reviewed by Codex, and
only posted to GitHub after user acceptance. The spec issue is the source of
truth for implementation; test scenarios are its centerpiece.

## Arguments

`/create-spec <source>`

- **Issue number** — a triaged user report (reads `.ai/issue-<n>-*/triage.md`
  if present) or an existing thin issue to be specced
- **Description** — a feature/fix idea from the user, specced from scratch
- **Nothing** — look for a `concept.md` or `triage.md` in the workflow
  folder matching the current branch

## Instructions

### 1. Gather Input

- If an issue number: fetch it (`mcp__github__get_issue`), and read
  `.ai/issue-<n>-*/triage.md` and any existing `analysis.md`/`concept.md`
- If a description: use it directly; check `mcp__github__search_issues` for
  related or duplicate issues (open and closed) before speccing
- Create/reuse the workflow folder `.ai/issue-<n>-<slug>/` or
  `.ai/feature-<slug>/`

### 2. Research the Codebase

Understand what the spec touches: current behavior, affected components,
existing configuration options, prior art. Enough to write concrete expected
behavior and test scenarios — not an implementation plan. Consult
ARCHITECTURE.md and CONFIGURATION.md as needed.

### 3. Draft the Spec

Write `.ai/<folder>/spec.md` following the Spec Issues structure in
ISSUE_GUIDELINES.md exactly (it will be posted as the issue body):

- Brief summary, `Refs #<n>` to the originating report if one exists
- **Goal** — what to achieve
- **Expected Behavior** — concrete: example commands, expected output,
  before/after
- **Test Scenarios** — the heart of the spec. Happy paths, error
  conditions, edge cases; each with setup, action, expected outcome,
  concrete enough to become a test without guessing
- **Out of Scope** — deliberate exclusions
- **Technical Notes** — only what removes ambiguity

The spec stays at concept level: what to achieve and what changes. How to
implement it belongs to the implementation phase.

**Breakdown**: if the work can't land as one reviewable PR, split into
sub-specs — each a complete spec of its own (goal, behavior, test
scenarios), independently implementable. Draft them as
`spec-<part-slug>.md` files; the main spec keeps the overall goal and a
Breakdown task list.

### 4. Codex Gate

Run a Codex gate per
[../_shared/CODEX_GATE.md](../_shared/CODEX_GATE.md):

- **Artifact**: the spec draft(s)
- **Task for Codex**: find missing test scenarios, ambiguous expected
  behavior, contradictions with the originating report, and scope creep.
  Is this a readable concept a maintainer can verify and accept?
- **Source of truth**: the originating issue/triage/concept content
- **Guidelines**: ISSUE_GUIDELINES.md (Spec Issues section)

Evaluate per the convention — apply high-confidence findings, reject
over-engineering, when in doubt leave it out. Log all verdicts to
`.ai/<folder>/codex-spec.md`.

### 5. Acceptance Gate

Present to the user:

- The full spec draft (and sub-specs if any)
- The Codex gate summary (applied/rejected counts, log path)
- Open questions, if any — genuine design decisions surfaced by drafting
  or the Codex gate

Wait for acceptance. The user may edit, answer open questions, or reject.
Iterate until accepted.

### 6. Post to GitHub

After acceptance:

1. Create the spec issue via `mcp__github__create_issue`: title per
   ISSUE_GUIDELINES.md (imperative, specific), body from `spec.md`, labels:
   `spec` plus `bug` or `enhancement`
2. For breakdowns: create sub-issues first, then the parent with the
   Breakdown task list referencing their numbers
3. Cross-link and close the originating user report, if any:
   - The spec body already carries `Refs #<n>`
   - Comment on the user report via `mcp__github__add_issue_comment`,
     addressing the reporter: the work is now tracked in the spec issue
     (link it) and this report is being closed in its favor so there is a
     single source of truth
   - Close the originating report
     (`gh issue close <n> --reason "not planned"`) — it is superseded by
     the spec issue, which is where the fix will land. If a maintainer wants
     a specific report kept open, they can say so at the acceptance gate.
   - When the spec later ships, the fix's own `Refs`/`Resolves` trailers and
     a closing note on the spec issue record completion; the original report
     is already consolidated.

### 7. Report

- Spec issue URL(s)
- Next step: implementation via `/resolve-issue <spec-number>` (or the
  manual chain starting with `/analyze-issue`)
