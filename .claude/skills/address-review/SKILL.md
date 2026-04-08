---
name: address-review
description: Check a PR for review comments, evaluate them, and address valid ones
argument-hint: <pr-number>
allowed-tools: Bash, Read, Write, Grep, Glob, mcp__github__get_pull_request, mcp__github__get_pull_request_reviews, mcp__github__get_pull_request_comments, mcp__github__add_issue_comment
---

# Address Review

Fetch review comments from a PR (Copilot, human, or any reviewer), evaluate each one against the project's development philosophy, and address valid ones — dismissing over-engineering or noise.

## Arguments

`/address-review <pr-number>`

## Instructions

### 1. Fetch PR Context

Fetch all review data in parallel:

- PR details: `mcp__github__get_pull_request` (owner: `gittower`, repo: `git-flow-next`)
- Reviews: `mcp__github__get_pull_request_reviews`
- Inline comments: `mcp__github__get_pull_request_comments`

From the PR details, extract the head branch name.

### 2. Identify New Comments

Compare reviews by `submitted_at` timestamp. Focus on the **most recent review round** — comments from earlier rounds that were already addressed (check for existing reply comments or commits after the review) should be noted as "previously addressed" and skipped.

Group comments by review round (same `pull_request_review_id`).

### 3. Checkout the Branch

Ensure the PR branch is checked out locally:

```bash
git fetch origin <head-branch>
git checkout <head-branch>
```

### 4. Read Relevant Source Files

For each comment, read the file and lines referenced in the `path` and `diff_hunk` fields. Understand the full context — don't evaluate comments in isolation.

### 5. Evaluate Each Comment

For every comment, assess it against these criteria:

**Accept when the comment identifies:**
- A real bug or logic error
- A genuine correctness issue (e.g., code doesn't achieve what it claims)
- A missing piece the reviewer correctly identified (e.g., missing doc sync, untested path)
- A simple improvement with clear value (e.g., reuse existing helper, better error message)

**Dismiss when the comment:**
- Proposes over-engineering that violates the project's pragmatic philosophy (CLAUDE.md: "Reject unnecessary complexity, premature abstractions, and excessive layering")
- Suggests adding code for hypothetical scenarios that aren't realistic
- Raises a concern already handled by existing behavior (e.g., an error message that's already actionable)
- Is a stale comment from a previous review round that was already addressed
- Suggests changes outside the scope of the PR

**Partially accept when:**
- The concern is valid but the suggested fix is over-engineered — accept the concern, implement a simpler fix
- A docs improvement is needed but the suggested text is too verbose — accept the intent, write concisely

For each comment, classify as: **accept**, **dismiss**, or **partial** (accept concern, simpler fix).

### 6. Present Analysis to User

Before making any changes, present the evaluation as a summary table:

```
## Review Analysis — PR #<number> (round <N>)

| # | File | Verdict | Summary |
|---|------|---------|---------|
| 1 | path/to/file.go | accept | <one-line reason> |
| 2 | path/to/other.go | dismiss | <one-line reason> |
| 3 | docs/file.md | partial | <concern valid, simpler fix> |
```

For dismissed comments, briefly explain why (reference project philosophy if relevant).
For partial accepts, explain what you'd do differently.

**Wait for user confirmation before proceeding.** The user may override any verdict.

### 7. Implement Accepted Changes

After user confirms:

1. Make the code changes for all accepted and partially-accepted comments
2. Run `go build ./...` to verify compilation
3. Run `go test ./...` (or the relevant test subset) to verify tests pass

Keep changes minimal and focused — fix what the review identified, don't refactor surrounding code.

### 8. Commit and Push

Use the `/commit` skill pattern (read `.claude/skills/commit/SKILL.md`):

- Stage only the changed files
- Write a commit message following COMMIT_GUIDELINES.md
- The commit message should reference the review, e.g., `fix(hooks): Address review feedback on Windows support`
- Push to the remote branch

### 9. Update PR

After pushing:

1. **Update the local PR summary** if one exists in `.ai/`:
   - Find the relevant `.ai/issue-*` or `.ai/pr-*` folder
   - Update `pr_summary.md` to reflect the new changes

2. **Update the PR description on GitHub** if the summary changed materially:
   ```bash
   gh api repos/gittower/git-flow-next/pulls/<number> -X PATCH -f body="<updated body>"
   ```

3. **Post a comment on the PR** summarizing what was addressed:
   - Group by accepted/dismissed
   - For accepted items, reference the commit SHA
   - For dismissed items, briefly explain why
   - Use `mcp__github__add_issue_comment`

### 10. Report

Output a final summary:
- How many comments were addressed, dismissed, or partially accepted
- What commit(s) were created
- Any remaining items that need manual attention

## Notes

- This skill is designed for automated reviewer comments (Copilot, bots) but works for human reviews too
- The project philosophy explicitly rejects over-engineering — use this as a filter
- When in doubt about a comment's validity, present it to the user rather than silently dismissing
- If the PR branch has diverged from main, do NOT rebase — just work on the current HEAD
