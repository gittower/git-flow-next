---
name: post-review
description: Post a written review file to GitHub as a proper PR review
argument-hint: <PR number or review file path> [additional instructions]
allowed-tools: Read, Glob, Grep, Bash, mcp__github__get_pull_request, mcp__github__get_pull_request_files, mcp__github__create_pull_request_review
---

# Post Review

Post a previously written review file to GitHub as a proper PR review with inline comments. Supports both `/pr-review` and `/code-review` output formats.

## Arguments

`/post-review <target> [additional instructions]`

- `<target>` — **required**, one of:
  - A **PR number** (e.g., `92`, `#92`) — finds the most recent review file for that PR
  - A **file path** to a review file (e.g., `.ai/pr-92/review-pr92-980cb92.md`)
- Everything after the target is treated as **additional instructions** — e.g., "only post the must-fix items", "skip the nits", "post as COMMENT instead of REQUEST_CHANGES", "don't include the commit message findings"

## Instructions

### 1. Parse Arguments

Extract the target from `$ARGUMENTS`:
- If it looks like a number (with optional `#` prefix): treat as PR number
- If it contains `/` or ends in `.md`: treat as a file path
- Everything after the target (that isn't the target itself) is the **additional instructions** string — preserve it for step 5

### 2. Find the Review File

**If target is a PR number:**
- Search for review files in this order:
  1. `.ai/pr-<number>/pr-review-*.md` (pr-review format with frontmatter)
  2. `.ai/pr-<number>/review-pr<number>-*.md` (code-review format)
  3. `.ai/issue-*-*/pr-review-*.md` matching the PR number in frontmatter
  4. `.ai/issue-*-*/review-pr<number>-*.md`
- If multiple files match, use the most recently modified one
- If no file is found, tell the user and stop

**If target is a file path:**
- Read the file directly
- If it doesn't exist, tell the user and stop

### 3. Detect Review Format and Parse

The skill handles two review file formats:

**Format A: `/pr-review` output** (has YAML frontmatter)

```markdown
---
pr: <number>
event: <APPROVE|COMMENT|REQUEST_CHANGES>
---

<summary text>

**Must Fix**
- ...

**Should Fix**
- ...

**Nit**
- ...

## Inline Comments

### `<file>:<line>`
<comment body>
```

Parse:
- `pr` and `event` from frontmatter
- **Review body** = everything between the frontmatter closing `---` and the `## Inline Comments` heading (or end of file if no inline comments section)
- **Inline comments** = each `### \`<file>:<line>\`` block under `## Inline Comments`

**Format B: `/code-review` output** (no frontmatter)

```markdown
# Code Review: PR #<number> - <title>

## Summary
- PR: #<number> - <title>
...

## Checklist Results

### Passed
- ...

### Must Fix
- [ ] **<title>** - `<file>:<line>` - <description>

### Should Fix
- [ ] **<title>** - `<file>:<line>` - <description>

### Nits
- ...

## Recommendations
1. ...
```

Parse:
- **PR number** from the `# Code Review: PR #<number>` heading or `- PR: #<number>` in Summary
- **Event**: determine from findings:
  - `REQUEST_CHANGES` if any "Must Fix" items exist
  - `COMMENT` if "Should Fix" items exist but no "Must Fix"
  - `APPROVE` if only "Passed" and optionally "Nit" items
- **Review body**: compose from the file content (see step 5)
- **Inline comments**: extract from Must Fix / Should Fix / Nit items that reference `file:line`

### 4. Get PR Diff for Line Mapping

Fetch the PR diff to map file:line references to actual diff positions:

```bash
gh pr diff <number>
```

Also fetch the PR metadata:
```bash
gh pr view <number> --json headRefOid,number,baseRefName --jq '{headRefOid: .headRefOid, number: .number, baseRefName: .baseRefName}'
```

For each inline comment, verify the referenced line is visible in the diff. If a line is not in the diff, move that finding into the review body summary instead.

### 5. Apply Additional Instructions

If the user provided additional instructions, apply them now. Common patterns:

- **"only must-fix"** / **"only the must fix items"** — drop Should Fix, Nit sections and their inline comments
- **"skip nits"** / **"no nits"** — remove Nit section and nit-related inline comments
- **"skip commit message"** / **"no commit findings"** — remove commit-message-related findings
- **"post as COMMENT"** — override the event to `COMMENT` regardless of findings
- **"post as APPROVE"** — override event to `APPROVE`
- **"only inline comments"** — minimize the body, post only inline comments
- **"only items 1 and 3"** — post only the specified numbered findings
- Any other filtering or modification the user describes

Use judgement to interpret the instructions. The goal is to give the user control over exactly what gets posted.

### 6. Compose the GitHub Review

**Review body** (the `body` parameter):

For **Format A** (pr-review): use the parsed review body text directly (between frontmatter and inline comments), after applying any filtering from step 5.

For **Format B** (code-review): compose the review body per [GITHUB_GUIDELINES.md](../../../GITHUB_GUIDELINES.md) (concise, no emojis, no checkboxes — it's a public GitHub review, not the full local file):
- Opening: a 1-3 sentence summary/verdict (derive from the review's overall assessment or recommendations)
- Include severity sections (Must Fix, Should Fix, Nit) that survived filtering, as clean bullet lists
- Do NOT include the "Passed", "Summary", "Recommendations", or "Ready for PR?" sections

Do not append any AI-attribution line to the review body (no "Generated with Claude" line, no emoji) — AI credit is via `Co-Authored-By` commit trailers only. See [GITHUB_GUIDELINES.md](../../../GITHUB_GUIDELINES.md).

**Inline comments** (the `comments` array):

For each inline comment that maps to a valid diff line:
- `path`: the file path (relative to repo root)
- `line`: the line number in the **new version** of the file
- `body`: the comment text

**Important**: Only include inline comments for lines that appear in the diff. The GitHub API will reject comments on lines not visible in the diff.

### 7. Preview Before Posting

Before posting, show the user a preview:

```
**PR #<number> — Review Preview**

Event: <APPROVE|COMMENT|REQUEST_CHANGES>

Body:
<first 5 lines of body>...

Inline comments: <count>
<list each as "- `file:line` — first 50 chars of comment...">
```

Then ask: **"Post this review?"**

Wait for the user's confirmation before proceeding. If they say no or want changes, adjust accordingly.

### 8. Post the Review

Determine repo owner and name:
```bash
gh repo view --json owner,name --jq '"\(.owner.login) \(.name)"'
```

Post using `mcp__github__create_pull_request_review` with:
- `owner`: repo owner
- `repo`: repo name
- `pull_number`: PR number
- `body`: composed review body
- `event`: determined event type
- `commit_id`: the head SHA of the PR
- `comments`: array of inline comments (may be empty)

### 9. Report Result

After posting:
- Confirm the review was posted
- Show the PR URL: `https://github.com/<owner>/<repo>/pull/<number>`
- If any inline comments were skipped (not in diff), mention which ones
