---
name: address-review
description: Fetch PR review feedback, evaluate it, implement accepted changes, and update the PR after confirmation
argument-hint: <pr-number> [--plan-only]
allowed-tools: Bash, Read, Write, Edit, Grep, Glob, mcp__github__get_pull_request, mcp__github__get_pull_request_reviews, mcp__github__get_pull_request_comments, mcp__github__add_issue_comment
---

# Address Review

Fetch review comments from a PR (Copilot, human, or any reviewer), evaluate each one against the project's development philosophy, write the evaluation as a plan artifact, and implement the accepted changes. All public actions (push, PR comment, description update) wait for user confirmation.

## Arguments

`/address-review <pr-number> [--plan-only]`

- `<pr-number>` — required, the PR whose review feedback to address
- `--plan-only` — evaluate and write the plan artifact, but do not implement. Use when the feedback is large enough to review the plan first; implement later with `/implement .ai/<folder>/review-plan-<sha>.md`.

## Instructions

### 1. Fetch PR Context

Fetch all review data in parallel:

- PR details: `mcp__github__get_pull_request` (owner: `gittower`, repo: `git-flow-next`)
- Reviews: `mcp__github__get_pull_request_reviews`
- Inline comments: `mcp__github__get_pull_request_comments`

From the PR details, extract the head branch name and head SHA (short form for filenames).

### 2. Identify New Comments

Compare reviews by `submitted_at` timestamp. Focus on the **most recent review round** — comments from earlier rounds that were already addressed (check for existing reply comments or commits after the review) should be noted as "previously addressed" and skipped.

Group comments by review round (same `pull_request_review_id`).

Also skip items that are:
- Explicitly marked as non-blocking ("nit:", "optional:", "minor:") unless trivially worth fixing
- Questions that were answered in reply threads without requiring changes
- Praise or approval statements

### 3. Checkout the Branch

Ensure the PR branch is checked out locally:

```bash
git fetch origin <head-branch>
git checkout <head-branch>
```

### 4. Read Relevant Source Files

For each comment, read the file and lines referenced in the `path` and `diff_hunk` fields. Understand the full context — don't evaluate comments in isolation.

### 5. Evaluate Each Comment

For every comment, assign a **verdict** and a **severity**.

**Verdict — accept when the comment identifies:**
- A real bug or logic error
- A genuine correctness issue (e.g., code doesn't achieve what it claims)
- A missing piece the reviewer correctly identified (e.g., missing doc sync, untested path)
- A simple improvement with clear value (e.g., reuse existing helper, better error message)

**Verdict — dismiss when the comment:**
- Proposes over-engineering that violates the project's pragmatic philosophy (CLAUDE.md: "Reject unnecessary complexity, premature abstractions, and excessive layering")
- Suggests adding code for hypothetical scenarios that aren't realistic
- Raises a concern already handled by existing behavior (e.g., an error message that's already actionable)
- Is a stale comment from a previous review round that was already addressed
- Suggests changes outside the scope of the PR

**Verdict — partial when:**
- The concern is valid but the suggested fix is over-engineered — accept the concern, implement a simpler fix
- A docs improvement is needed but the suggested text is too verbose — accept the intent, write concisely

**Severity** (for accepted and partial items, per REVIEW_CRITERIA.md categories):
- **Must fix** — bugs, security issues, missing critical tests, correctness problems
- **Should fix** — missing edge case tests, documentation gaps, code clarity issues
- **Nit** — style preferences, optional improvements

### 6. Write the Plan Artifact

Always write the evaluation before doing anything else — this is the audit trail for the review round.

Determine the output folder:

```bash
# Extract issue number from branch name if present (feature/42-something -> 42)
# Look for existing .ai/issue-<number>-* folder, fall back to .ai/pr-<number>/
```

Write to `.ai/<folder>/review-plan-<head-sha>.md`:

```markdown
# Review Plan: PR #<number> (round <N>)

## Source
- PR: #<number> - <title> (<link>)
- Review: <reviewer> on <date>
- Revision: `<full-head-sha>`
- Branch: `<branch-name>`

## Evaluation

| # | File | Verdict | Severity | Summary |
|---|------|---------|----------|---------|
| 1 | path/to/file.go | accept | must fix | <one-line reason> |
| 2 | path/to/other.go | dismiss | — | <one-line reason> |
| 3 | docs/file.md | partial | should fix | <concern valid, simpler fix> |

## Tasks

### Task 1: <derived from accepted comment>
**Files**: `<path/to/file.go>`
**Changes**:
- [ ] <specific change>

**Details**: <what to change and why; for partial accepts, describe the simpler fix>

## Dismissed

- #2 — <why, referencing project philosophy where relevant>
```

If no actionable feedback was found, write the plan with an empty Tasks section noting the review was clean (approval or comments without action items), report that, and stop.

### 7. Plan-Only Stop

If `--plan-only` was given: report the path to the plan artifact, summarize the verdicts, and stop. Suggest `/implement .ai/<folder>/review-plan-<sha>.md` as the next step.

### 8. Implement Accepted Changes

Without `--plan-only`, proceed directly — no confirmation needed for local work:

1. Make the code changes for all accepted and partially-accepted tasks
2. Run `go build ./...` to verify compilation
3. Run `go test ./...` (or the relevant test subset) to verify tests pass
4. Update the checkboxes in the plan artifact as tasks complete

Keep changes minimal and focused — fix what the review identified, don't refactor surrounding code.

### 9. Commit Locally

Use the `/commit` skill pattern (read `.claude/skills/commit/SKILL.md`):

- Stage only the changed files
- Write a commit message following COMMIT_GUIDELINES.md
- The commit message should reference the review, e.g., `fix(hooks): Address review feedback on Windows support`

**Do NOT push yet.**

### 10. Prepare Public Actions

Draft everything that will touch GitHub, but do not execute yet.

**Route each reply by where the feedback lives:**

- **Inline diff comments** (a review comment with a `path` + `diff_hunk`, i.e. anchored to a line) → reply **inline on that comment's thread**, one reply per thread, so it resolves in context. Each reply states the verdict for that specific comment (accepted + commit SHA, dismissed + reason, or partial + what differed). After replying, **resolve that thread** (see step 12) — every thread we handled (accept, dismiss, or partial) gets resolved once its reply is posted. Leave a thread unresolved only if it stays genuinely open (e.g. deferred to a follow-up or awaiting the reviewer's decision); call those out at the gate.
- **General review comments** (the review body, or a top-level PR conversation comment not anchored to a diff line) → collect them into **one combined PR comment** that **quotes each piece of feedback** (Markdown `>` blockquote) followed by the response. Do not open a separate comment per general item.

So a review round may produce inline replies, a single combined comment, or both — depending on which kinds of feedback it contained. If there are no general comments, post no combined comment; if there are no inline comments, post no inline replies.

Also draft:
- **PR description update** — if a `pr_summary.md` exists in the `.ai/` folder and the changes materially alter the summary, draft the updated body
- The **push** of the new commit(s)

### 11. Confirmation Gate

Present to the user in one block:

- The verdict table (from step 6, updated with commit SHAs)
- Commits created (`git log` oneline of the new commits)
- The full draft replies — each inline thread reply (with the file it targets) and the combined general-comment comment, whichever apply
- Which inline threads will be **resolved** after their reply posts (and any that will be left open, with why)
- Whether the PR description will be updated

Then ask: **"Push and post?"**

Wait for confirmation. The user may adjust verdicts, edit the comment, or ask for changes — apply them and re-present. Nothing is pushed or posted until confirmed.

### 12. Execute Public Actions

After confirmation:

1. Push to the remote branch
2. Update the PR description if drafted:
   ```bash
   gh api repos/gittower/git-flow-next/pulls/<number> -X PATCH -f body="<updated body>"
   ```
3. Post the replies:
   - **Inline thread replies** — reply to each inline diff comment on its own thread:
     ```bash
     gh api repos/gittower/git-flow-next/pulls/<number>/comments/<comment-id>/replies -f body="<reply>"
     ```
   - **Combined general comment** — if any general feedback was collected, post the single quoted-and-answered comment via `mcp__github__add_issue_comment`
4. **Resolve the handled inline threads.** A REST reply does *not* resolve the thread — resolution is a GraphQL mutation keyed by the thread's node ID (not the comment ID). Map each handled comment to its thread, then resolve it.

   Fetch the thread IDs once (map `databaseId` of the first comment → thread `id`):
   ```bash
   gh api graphql -f query='
   query($owner:String!,$repo:String!,$pr:Int!){
     repository(owner:$owner,name:$repo){
       pullRequest(number:$pr){
         reviewThreads(first:100){
           nodes{ id isResolved comments(first:1){ nodes{ databaseId path } } }
         }
       }
     }
   }' -f owner=gittower -f repo=git-flow-next -F pr=<number>
   ```
   Then resolve each thread we replied to with a verdict (accept, dismiss, or partial):
   ```bash
   gh api graphql -f query='
   mutation($threadId:ID!){
     resolveReviewThread(input:{threadId:$threadId}){ thread{ isResolved } }
   }' -f threadId=<thread-node-id>
   ```
   Skip threads flagged as genuinely open at the gate, and any already `isResolved`.
5. Update `pr_summary.md` in `.ai/` if it exists

### 13. Report

Output a final summary:
- How many comments were addressed, dismissed, or partially accepted
- What commit(s) were created and pushed
- Links to the posted replies (inline threads and/or the combined comment)
- Which inline threads were resolved, and any left open (with why)
- Any remaining items that need manual attention

## Notes

- This skill replaces the former `/plan-from-review` — its output format lives on as the plan artifact in step 6, and `--plan-only` covers the plan-first workflow
- The plan artifact is written per revision (`review-plan-<sha>.md`), so multiple review cycles accumulate side by side and stay correlated with the PR state they reviewed
- The project philosophy explicitly rejects over-engineering — use this as a filter
- When in doubt about a comment's validity, present it to the user at the gate rather than silently dismissing
- If the PR branch has diverged from main, do NOT rebase — just work on the current HEAD
