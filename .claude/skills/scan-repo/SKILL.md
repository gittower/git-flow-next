---
name: scan-repo
description: Weekly repo activity scan — new discussions, issues, and PRs, ordered by effort into an action list
argument-hint: [since, e.g. 14d or 2026-07-01]
allowed-tools: Bash, Read, Write, Glob, mcp__github__list_issues, mcp__github__search_issues, mcp__github__list_pull_requests, mcp__github__get_pull_request_reviews
---

# Scan Repo

Scan all activity on gittower/git-flow-next since the last scan and produce
a prioritized action list: what came in (discussions, issues, PRs), what
needs a reaction from us, and in which order to tackle it — quick wins
first, deeper work later, each item with the command that handles it.

Read-only: the scan never posts, labels, or replies. It only reports. Meant
to be run manually about once per week.

## Arguments

`/scan-repo [since]`

- `since` — optional lookback: a duration (`7d`, `14d`) or a date
  (`2026-07-01`). Default: the date of the most recent scan report in
  `.ai/scans/`, or 7 days if none exists.

## Instructions

### 1. Determine the Window

```bash
ls .ai/scans/scan-*.md 2>/dev/null | sort | tail -1
```

Use the date in the newest filename as `SINCE` unless an argument was
given. Fall back to 7 days ago. Get today's date with `date +%Y-%m-%d`.

### 2. Gather Activity

Collect in parallel where possible:

**Discussions** (GraphQL):
```bash
gh api graphql -f query='query { repository(owner: "gittower", name: "git-flow-next") {
  discussions(first: 30, orderBy: {field: UPDATED_AT, direction: DESC}) {
    nodes { number title category { name } isAnswered updatedAt createdAt
      author { login } comments(last: 3) { nodes { author { login } createdAt } } } } } }'
```
Flag: new discussions since `SINCE`; unanswered questions; threads where
the last comment is not ours.

**Issues**:
- New since the window: `gh issue list --state open --search "created:>SINCE" --json number,title,labels,author,createdAt`
- Updated with external activity: `gh issue list --state open --search "updated:>SINCE" --json number,title,labels,comments,updatedAt` — for each, check whether the last comment is from an external user (awaiting our reply) or from us (awaiting theirs; no action)
- Open `needs info`-type situations where the reporter has now responded

**Pull requests**:
- Assess every open PR exactly as `/check-prs` does (read
  `.claude/skills/check-prs/SKILL.md`, steps 1–2) — status vs the review
  response window, days waiting, whose move it is
- Additionally flag PRs opened since `SINCE` as new

**Housekeeping signals** (cheap checks, include only if noteworthy):
- CI status on main: `gh run list --branch main --limit 5 --json conclusion,name`
- Open spec issues (`label:spec`) with no linked PR — accepted work nobody
  started

Treat comments by repo collaborators and bots as "ours"; everyone else is
external.

### 3. Classify by Effort

Sort every actionable item into one of three buckets:

**Quick wins** — minutes each, mostly judgment-free:
- Questions we can answer directly (answer known from code/docs)
- Obvious duplicates to link and close
- Reporters who answered a needs-info request (unblock them)
- Reminder-worthy PRs approaching the response window
- Small PRs (single file, few lines, clear purpose)

**One command** — the autonomous pipelines do the heavy lifting; our cost
is reviewing their output at the gates:
- Untriaged bug reports and feature requests → `/triage`
- Typical incoming PRs → `/handle-pr`
- Lapsed response windows → `/takeover-pr`
- Accepted-but-unstarted specs → `/resolve-issue`

**Needs thinking** — human design or policy judgment up front:
- Feature requests touching architecture, config semantics, or
  compatibility (triage will end in a design discussion)
- Bug reports that contradict the spec or expose unclear intended behavior
- Large or multi-concern PRs (scope conversation with the author)
- Anything where two pipelines conflict (e.g., PR submitted for an issue
  that was never accepted)

Within each bucket, order by age (oldest first) — external people are
waiting.

### 4. Write the Report

Write `.ai/scans/scan-<today>.md`:

```markdown
# Repo Scan <today> (since <SINCE>)

## Summary
<2-3 sentences: volume, anything urgent, overall state>

## Action List

### Quick wins
| # | Item | Waiting | Why quick | Command |
|---|------|---------|-----------|---------|
| 1 | discussion #12 — <title> | 6d | answer is in docs/gitflow-config.5.md | reply directly |
| 2 | issue #101 — <title> | 3d | duplicate of #87 | /triage 101 |

### One command
| # | Item | Waiting | Pipeline | Command |
|---|------|---------|----------|---------|
| 3 | issue #103 — <title> | 2d | bug triage | /triage 103 |
| 4 | PR #99 — <title> | 4d | review | /handle-pr 99 |

### Needs thinking
| # | Item | Waiting | What to think about |
|---|------|---------|---------------------|
| 5 | issue #104 — <title> | 5d | proposes new branch type semantics — design call needed |

## No Action Needed
<Items with recent activity that are waiting on the other side, in-window
PRs, answered discussions — one line each so nothing looks forgotten.>

## Housekeeping
<CI on main, unstarted specs, anything stale — or "all green".>
```

### 5. Report to User

Show the summary and the full action list in the conversation, in tackle
order. End with the single most valuable next command (usually action #1).
