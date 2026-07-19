---
name: handle-pr
description: Review an incoming PR end-to-end — strict local review, drafted GitHub review, post after confirmation
argument-hint: <pr-number>
allowed-tools: Task, Bash, Read, Glob, Grep, Write, mcp__github__get_pull_request, mcp__github__get_pull_request_files, mcp__github__create_pull_request_review
---

# Handle PR

Review an incoming pull request end-to-end: a strict local review against
REVIEW_CRITERIA.md (including scope and spec satisfaction), distilled into a
GitHub review draft, posted only after user confirmation. Internal and
external PRs are treated the same.

For later rounds — after the author responds to a posted review — use
`/follow-up-review <number>` instead (it is round-aware: it verifies the prior
requested changes against the delta and reads the author's replies).
`/address-review` is the author-side counterpart, for feedback on a PR *you*
authored — not for re-reviewing an incoming PR.

## Arguments

`/handle-pr <pr-number>`

## Workflow

### Step 1: Pre-Checks

Fetch the PR (`mcp__github__get_pull_request`). Check before any deep
review:

- **Scope**: does the PR address exactly one concern (CONTRIBUTING.md: one
  PR, one concern)? If it clearly bundles unrelated changes, the review can
  short-circuit: the primary finding is "split this PR", detailed findings
  are secondary
- **Linked spec/issue**: identify the linked issue from the PR body or
  branch name. If it links a `spec` issue, that spec is the verification
  target
- **CI status**: note failing checks (`gh pr checks <number>`)

### Step 2: Local Review

Spawn a subagent (Task tool, `subagent_type: "general-purpose"`):

- **description**: `Review PR locally`
- **prompt**: `Read the skill definition at .claude/skills/code-review/SKILL.md and execute it fully in PR mode for PR #<number>. Apply all eight review areas in REVIEW_CRITERIA.md strictly — including Scope (one PR, one concern) and Spec Satisfaction against issue #<linked-issue>. This is an incoming PR; hold it to the project's full standard: tests, guidelines, commit messages, documentation.`

**Verify**: the review file `review-pr<number>-*.md` exists in `.ai/`.

### Step 3: Draft the GitHub Review

Spawn a subagent:

- **description**: `Draft GitHub review`
- **prompt**: `Read the skill definition at .claude/skills/pr-review/SKILL.md and execute it fully for PR #<number>. Use the existing code review file in .ai/ as the basis. Write the review file for preview — do not post.`

**Verify**: the `pr-review-*.md` draft exists.

### Step 4: Confirmation Gate

Present to the user:

- Verdict (APPROVE / COMMENT / REQUEST_CHANGES) and why
- The full review body and inline comments from the draft
- Anything noteworthy from pre-checks (scope, CI, missing spec link)

Wait for confirmation. The user may edit findings, downgrade/upgrade the
event, or decide not to post.

### Step 5: Post

On confirmation, post via the `/post-review` skill pattern (read
`.claude/skills/post-review/SKILL.md`): parse the draft, map inline comments
to diff lines, post with `mcp__github__create_pull_request_review`.

### Step 6: Report

- Posted review URL and event type
- If REQUEST_CHANGES: note that the 7-day response window from
  CONTRIBUTING.md starts now — `/check-prs` tracks it, `/takeover-pr`
  applies after it lapses
