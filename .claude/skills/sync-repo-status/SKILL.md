---
name: sync-repo-status
description: Regenerate REPO_STATUS.md — full snapshot of open PRs, local branches, issues, and discussions with merge-readiness and triage assessment
allowed-tools: Bash, Read, Write, Glob, mcp__github__list_issues, mcp__github__list_pull_requests, mcp__github__get_pull_request_reviews, mcp__github__get_pull_request_comments
---

# Sync Repo Status

Regenerate `REPO_STATUS.md` in the project root: a complete point-in-time
snapshot of everything open — PRs (grouped by whose move it is), local
branches without PRs, issues (grouped by theme, with reply/duplicate/
linked-PR status), and open discussion questions — ending in a prioritized
action queue.

Read-only against GitHub: never posts, labels, or replies. Unlike
`/scan-repo` (a windowed activity delta written to `.ai/scans/`), this
produces the full current state, overwriting `REPO_STATUS.md` each run.

## Arguments

`/sync-repo-status`

## Instructions

### 1. Gather State

Run in parallel where possible:

**Local branches** (fetch first so ahead/behind counts are accurate):
```bash
git fetch --all --prune
git branch -vv --sort=-committerdate
```
For each local branch except main:
```bash
git rev-list --left-right --count main...<branch>   # behind/ahead
git log -1 --format='%cs %s' <branch>               # last commit date + subject
```
Note branches with no upstream (unpushed work) and branches checked out in
other worktrees (marked `+` in `git branch` output).

**Open PRs**:
```bash
gh pr list --state open --limit 50 --json number,title,author,headRefName,baseRefName,isDraft,createdAt,updatedAt,reviewDecision,mergeable,statusCheckRollup
```

**PR ↔ issue links** (`closingIssuesReferences` is GraphQL-only — not
available via `gh pr view --json`):
```bash
gh api graphql -f query='{ repository(owner: "gittower", name: "git-flow-next") {
  pullRequests(states: OPEN, first: 50) { nodes { number
    closingIssuesReferences(first: 5) { nodes { number } } } } } }'
```

**Per-PR review state** — for each open PR fetch reviews, comments, and
unresolved review threads:
```bash
gh pr view <n> --json author,reviews,comments
gh api graphql -f query='query($pr: Int!) { repository(owner: "gittower", name: "git-flow-next") {
  pullRequest(number: $pr) { reviewThreads(first: 50) { nodes { isResolved
    comments(first: 1) { nodes { author { login } body } } } } } } }' -F pr=<n>
```
Determine whose move it is: read the last few PR comments and the latest
review verdict. Copilot review threads are often left unresolved even after
being addressed — check the author's "addressed review" comments before
treating unresolved threads as open work.

**Open issues**:
```bash
gh issue list --state open --limit 100 --json number,title,labels,createdAt,updatedAt,author,comments
```
For unanswered external issues (no maintainer among commenters), read the
body (`gh issue view <n> --json title,body`) to classify and spot
duplicates.

**Discussions** (GraphQL):
```bash
gh api graphql -f query='{ repository(owner: "gittower", name: "git-flow-next") {
  discussions(first: 50, states: OPEN) { nodes { number title createdAt
    isAnswered category { name } author { login }
    comments(first: 10) { totalCount nodes { author { login } } } } } }'
```

Treat comments by repo collaborators and bots as "ours"; everyone else is
external.

### 2. Assess

**PRs** — sort into four groups:
- *Own PRs, merge candidates*: review feedback addressed, mergeable; note
  behind-main count and anything to verify before merging (e.g. missing
  docs for new config keys — cross-check `docs/gitflow-config.5.md`)
- *Contributor PRs waiting on maintainer*: no review yet, or resolved and
  awaiting approval; flag overlaps with own PRs or local branches
- *Contributor PRs waiting on them*: changes requested with no author
  activity since — note how long, suggest nudge/close when stale (>2 months)
- *Local branches without PRs*: assess by ahead/behind and age whether to
  revive, restart from main, or delete

**Issues** — group by theme (recurring clusters: merge-state/finish bugs,
Windows distribution, config/init bugs, AVH parity & feature requests, own
backlog/roadmap). For each issue note:
- replied or unanswered (bold the unanswered ones)
- duplicate or same-family of another issue
- linked to an open PR (from closingIssuesReferences) — likely resolved by
  a pending merge
- waiting on reporter → nudge/close candidate

**Discussions** — flag unanswered Q&A threads, replied-but-unmarked
answers, and idea threads that connect to open issues or branches.

### 3. Write REPO_STATUS.md

Overwrite `REPO_STATUS.md` in the project root. Structure:

```markdown
# Repo Status Overview

_Snapshot of open PRs, branches, issues, and discussions — generated <date>._

## Open Pull Requests (<n>)
### Own PRs — merge candidates          (table: PR, title, branch state, status)
### Contributor PRs — waiting on maintainer
### Contributor PRs — waiting on them (stale)

## Local branches without PRs           (table: branch, state, assessment)

## Open Issues (<n>), grouped           (one subsection per theme)

## Discussions

## TLDR action queue                    (numbered, most valuable first)
```

Every PR/issue reference as a markdown link. Keep assessments to one line
each — what it is, where it stands, what unblocks it.

`REPO_STATUS.md` is listed in `.gitignore` — verify it still is, and never
commit the generated file.

### 4. Report to User

Summarize in the conversation: counts per group, what changed since the
previous snapshot if one existed (newly ready PRs, new unanswered issues,
items that dropped off), and the top 3 items from the action queue.
