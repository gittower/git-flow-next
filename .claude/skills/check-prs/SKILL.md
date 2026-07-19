---
name: check-prs
description: Scan open PRs for stale review requests and report which need action
allowed-tools: Bash, Read, mcp__github__list_pull_requests, mcp__github__get_pull_request_reviews, mcp__github__add_issue_comment
---

# Check PRs

Scan all open pull requests and report where each stands against the review
response window policy in CONTRIBUTING.md. Read-only by default — run
manually whenever you want an overview. Posting reminders is optional and
gated.

## Arguments

`/check-prs`

## Instructions

### 1. Gather Open PRs

```bash
gh pr list --state open --json number,title,author,updatedAt,reviewDecision,isDraft
```

Skip drafts.

### 2. Assess Each PR

For each PR, fetch reviews and comments and determine:

- **Latest maintainer review** and its verdict + date
- **Author activity since**: commits pushed, comments, or review replies
  after that review (any activity resets the window)
- **Days waiting**: since the review if unanswered, else since last
  maintainer activity

Classify:

| Status | Meaning |
|--------|---------|
| `awaiting review` | No maintainer review yet — action is on us |
| `changes requested — in window` | Author has < 7 days left to respond |
| `changes requested — window lapsed` | ≥ 7 days, no author response — takeover eligible |
| `author responded` | Author replied/pushed — action is on us to re-review |
| `approved` | Ready to merge |

### 3. Report

Output a table sorted by urgency (lapsed first, then author-responded, then
awaiting review):

```
| PR | Title | Author | Status | Days | Next action |
|----|-------|--------|--------|------|-------------|
| #93 | ... | @user | window lapsed | 9 | /takeover-pr 93 |
| #95 | ... | @user | author responded | 2 | /follow-up-review 95 |
| #96 | ... | @user | awaiting review | 4 | /handle-pr 96 |
```

Include concrete evidence for lapsed PRs (review date, last author
activity) so the takeover decision is verifiable.

### 4. Optional: Reminders

For PRs in-window but close to lapsing (5–6 days), offer to post a friendly
reminder comment. Draft the comment, show it, and post only after user
confirmation (`mcp__github__add_issue_comment`). Never post reminders
unprompted.
