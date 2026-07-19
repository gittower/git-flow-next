---
name: follow-up-review
description: Re-review an incoming PR after the author responded — scoped to the delta since the last review, verifies prior requested changes, posts after confirmation
argument-hint: <pr-number>
allowed-tools: Bash, Read, Write, Grep, Glob, mcp__github__get_pull_request, mcp__github__get_pull_request_files, mcp__github__get_pull_request_reviews, mcp__github__get_pull_request_comments, mcp__github__create_pull_request_review
---

# Follow-up Review

Re-review an incoming pull request after the author has responded to a prior
maintainer review. Unlike `/handle-pr` (a fresh full review), this skill is
**round-aware**: it reconstructs what the last review requested, reads the
author's replies, reviews only the delta since the last-reviewed commit, and
reports each prior item as Resolved / Still open / Partially addressed — plus
anything new introduced in the delta. Posts only after user confirmation.

Use this for **every round after the first**. For the first review of an
incoming PR, use `/handle-pr`. For addressing feedback on a PR *you authored*,
use `/address-review`.

## Arguments

`/follow-up-review <pr-number>`

## Workflow

### Step 1: Fetch Prior Round State

Fetch in parallel (owner `gittower`, repo `git-flow-next`):

- PR details: `mcp__github__get_pull_request` — current HEAD SHA, head branch, base
- Reviews: `mcp__github__get_pull_request_reviews`
- Inline comments: `mcp__github__get_pull_request_comments`

Identify the **most recent maintainer review** (the last round we posted). Record:

- Its `event` (was it REQUEST_CHANGES / COMMENT / APPROVE) and `submitted_at`
- Its `commit_id` — this is the **last-reviewed SHA**. The delta to review is
  `lastReviewedSHA..HEAD`.

If no prior maintainer review exists, stop and tell the user to run `/handle-pr`
first — there is no round to follow up on.

### Step 2: Reconstruct the Prior Asks

Parse the last maintainer review (body + its inline comments) into a list of
**requested items**, each with:

- Severity (Must fix / Should fix / Nit)
- The `file:line` (or inline thread) it was anchored to
- The thread ID, if it was an inline diff comment (needed to check for replies)

Also pull the prior local artifact if present — check `.ai/<folder>/` for the
`pr-review-*.md` or `review-pr<number>-*.md` from the last round — and use it to
recover any nuance the posted review compressed.

### Step 3: Read the Author's Replies First

Before judging anything, read what the author said. For each prior inline thread,
check for reply comments (any author) and whether the thread is resolved. Read
general PR conversation comments since the last review too.

This matters: an author may have addressed a concern in code, or *explained why
they didn't* ("intentional because X"). Account for the reply before you flag an
item as still-open. If a reply makes a dismissal reasonable, note it rather than
re-raising the same point.

### Step 4: Check Out and Scope the Delta

```bash
git fetch origin <head-branch>
git checkout <head-branch>
```

Get the delta since the last review:

```bash
git diff <lastReviewedSHA>..<HEAD> --stat
git log <lastReviewedSHA>..<HEAD> --oneline
```

**Scope rule — delta-only.** Review the commits added since the last review, in
the context of the prior asks. Do **not** re-scan the whole PR for pre-existing
issues outside the delta that weren't flagged last round — re-surfacing untouched
code as new must-fixes reads as moving goalposts to the author. New findings are
limited to what the delta actually changed. (If you genuinely spot a serious
pre-existing defect outside the delta, raise it to the user at the gate as an
out-of-scope note, not as a blocking finding.)

Apply **[../code-review/REVIEW_CRITERIA.md](../code-review/REVIEW_CRITERIA.md)** to the delta.

### Step 5: Verdict Each Prior Ask

For every item from Step 2, assign a status backed by evidence:

- **Resolved** — the delta (or an author reply) fully addresses it. Cite the
  commit SHA or the line that fixes it.
- **Partially addressed** — some but not all of the concern is handled. Say
  what's left.
- **Still open** — not addressed, and the reply (if any) doesn't justify leaving
  it. Carry its original severity forward.
- **Withdrawn** — on reflection (often after the author's explanation) the item
  no longer applies. Note why.

### Step 6: Determine the Event

- `APPROVE` — every prior Must/Should item is Resolved or Withdrawn, and the
  delta introduces no new Must-fix. (Outstanding Nits alone don't block.)
- `REQUEST_CHANGES` — any prior Must-fix is Still open / Partially addressed, or
  the delta introduces a new Must-fix.
- `COMMENT` — no blocking items, but Should-fix items remain open or the delta
  adds non-blocking findings.

### Step 7: Write the Follow-up Draft

Determine the output folder (reuse the existing one from prior rounds):

```bash
# Extract issue number from branch name (feature/42-x -> 42); prefer the
# existing .ai/issue-<number>-* folder, else .ai/pr-<number>/
HEAD_SHA=$(gh pr view <number> --json headRefOid --jq '.headRefOid' | cut -c1-7)
```

Write to `.ai/<folder>/followup-review-<HEAD_SHA>.md` using this format — the
frontmatter matches `/pr-review` so `/post-review` can consume it directly:

````markdown
---
pr: <number>
event: <APPROVE|COMMENT|REQUEST_CHANGES>
round: <N>
since: <lastReviewedSHA-short>
---

<Opening: one or two sentences. State the round and that this reviews the delta
since <since>. Lead with what still blocks merge, or say plainly that the prior
items are addressed and nothing new blocks. No generic praise. See GITHUB_GUIDELINES.md.>

**Since last round**

| Prior item | Severity | Status | Evidence |
|------------|----------|--------|----------|
| <short desc> — `file:line` | must fix | Resolved | `<sha>` |
| <short desc> — `file:line` | should fix | Still open | not addressed |
| <short desc> — `file:line` | nit | Withdrawn | intentional per author reply |

**Still Open**
- <item> — `file:line` — <what remains>

**New in this round**
- <finding introduced by the delta> — `file:line` — <brief explanation>

## Inline Comments

### `<file>:<line>`
<Comment body — concise, actionable. Only on lines visible in the delta diff.>
````

**Format rules:**
- Only include sections that have content. If everything is resolved, the
  "Still Open" and "New in this round" sections are omitted — the table carries it.
- For an **APPROVE**, the opening must explicitly confirm the prior items landed
  (e.g. "All three round-1 must-fix items addressed — `<sha>`, `<sha>`, `<sha>`;
  nothing new blocks merge."), not a blind clean-approval.
- Tone/style per [GITHUB_GUIDELINES.md](../../../GITHUB_GUIDELINES.md) — concise,
  no emojis, no checkboxes, no hard-wrapping. This is posted publicly.
- Inline comment headers use the exact `### \`file:line\`` form for parseability.

### Step 8: Confirmation Gate

Present to the user in one block:

- Event (APPROVE / COMMENT / REQUEST_CHANGES) and why
- The "Since last round" table
- The full draft body and inline comments
- Any out-of-scope pre-existing issues noticed (as notes, not findings)

Wait for confirmation. The user may re-verdict items, downgrade/upgrade the
event, or decide not to post.

### Step 9: Post

On confirmation, post via the `/post-review` pattern (read
`.claude/skills/post-review/SKILL.md`), passing the draft path explicitly:
parse the frontmatter for `pr` and `event`, use the body up to `## Inline
Comments` as the review body, map inline comments to delta diff lines, and post
with `mcp__github__create_pull_request_review` (include `commit_id` = HEAD SHA).
Verify the body against the [posting checklist](../_shared/POSTING_CHECKLIST.md) first.

### Step 10: Report

- Posted review URL and event type
- Counts: prior items resolved / still open / new findings
- If REQUEST_CHANGES: the CONTRIBUTING.md response window restarts — `/check-prs`
  tracks it; the next author response is another `/follow-up-review <number>`

## Notes

- Delta-only by design: each round reviews `lastReviewedSHA..HEAD`, so successive
  `followup-review-<sha>.md` artifacts accumulate side by side and stay correlated
  with the PR state they reviewed — mirroring `/address-review`'s per-SHA plans.
- Read author replies before flagging (Step 3) — this is what makes the round feel
  like a conversation rather than a re-run.
- Reviewer role only. `/address-review` is the author-side counterpart (implement
  feedback on your own PR); do not confuse the two.
