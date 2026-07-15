---
name: takeover-pr
description: Take over a stale PR — supersede it with requested changes applied, crediting the original author
argument-hint: <pr-number>
allowed-tools: Bash, Read, Write, Edit, Grep, Glob, mcp__github__get_pull_request, mcp__github__get_pull_request_reviews, mcp__github__get_pull_request_comments, mcp__github__create_pull_request, mcp__github__add_issue_comment, mcp__github__update_issue
---

# Takeover PR

Execute the review response window policy from CONTRIBUTING.md: when a
contributor has not responded to requested changes within 7 days, close
their PR and land a successor PR based on their work with the requested
changes applied on top — crediting them. All public actions (push, new PR,
closing comment) wait for user confirmation.

## Arguments

`/takeover-pr <pr-number>`

## Instructions

### 1. Verify Eligibility — Strictly

Fetch the PR, its reviews, comments, and commits. The takeover is only
legitimate if **all** of these hold:

- The latest maintainer review is `REQUEST_CHANGES` (or explicitly requests
  changes in its body)
- **7 or more days** have passed since that review was submitted
- The author has not responded since: no commits pushed, no comments, no
  review replies after the review's timestamp. Any author activity — even
  "I need more time" — resets the window per CONTRIBUTING.md

If any condition fails, **stop** and report why (including the exact dates).
Do not proceed on a technicality; when the situation is ambiguous (e.g., the
author reacted with an emoji, or pushed to a different branch), surface it
to the user instead of deciding alone.

### 2. Preserve the Contributor's Work

```bash
git fetch origin pull/<number>/head:takeover/pr-<number>
git checkout develop && git pull
git checkout -b feature/<slug> develop
```

Derive `<slug>` from the linked issue or PR title, following the usual
branch naming. Then bring in the contributor's commits:

- **Preferred**: `git merge` or `git rebase` their commits onto the new
  branch **unchanged** — original authorship is preserved automatically
- If their commits must be reworked (squashed, split, or amended to meet
  COMMIT_GUIDELINES.md): add `Co-authored-by: <name> <email>` trailers to
  every commit that contains their work. Get name/email from
  `git log --format='%an <%ae>'` on their commits

### 3. Apply the Requested Changes

Use the review-feedback machinery rather than improvising:

1. Evaluate the review comments exactly as `/address-review` does (read
   `.claude/skills/address-review/SKILL.md`, steps 4–6): verdict + severity
   per comment, written to `.ai/pr-<number>/review-plan-<sha>.md`
2. Implement the accepted items on the new branch
3. Run `go build ./...` and `go test ./...`
4. Commit per COMMIT_GUIDELINES.md — takeover fixes are your commits;
   the contributor's credit stays on their preserved commits

### 4. Local Review

Run `/code-review` (read the skill) on the new branch vs main. Fix any
must-fix findings before proceeding.

### 5. Draft the Public Actions

Prepare, but do not execute:

1. **Successor PR** title and body (per `/pr-summary` format), including a
   credit paragraph: `This PR supersedes #<number> by @<author>, whose
   commits are included <unchanged | with Co-authored-by attribution>. It
   adds the changes requested in review.` plus `Closes #<issue>` for the
   underlying issue
2. **Closing comment** for the original PR: courteous and factual — thank
   the author, link the successor PR, reference the CONTRIBUTING.md
   response window policy, invite them to review the successor
3. Whether CONTRIBUTORS.md needs the author added (check if they're
   already listed)

### 6. Confirmation Gate

Present to the user:

- The eligibility evidence (review date, days elapsed, no author activity)
- Commits on the new branch (`git log --oneline develop..HEAD`) showing
  preserved authorship
- The review-plan verdicts and what was implemented
- The full successor PR body and the closing comment

Ask: **"Push, open the successor PR, and close the original?"**

### 7. Execute

After confirmation:

1. Push the branch: `git push -u origin feature/<slug>`
2. Create the successor PR (`mcp__github__create_pull_request`)
3. Post the closing comment on the original PR
   (`mcp__github__add_issue_comment`), then close it:
   `gh pr close <number>`
4. Update CONTRIBUTORS.md if drafted in step 5 (commit on the new branch
   or main per project practice)

### 8. Report

- Successor PR URL, original PR closed
- How the contributor was credited
- Any review findings deferred to follow-up
