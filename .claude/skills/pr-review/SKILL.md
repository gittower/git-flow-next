---
name: pr-review
description: Create a GitHub PR review with inline comments, preview before posting
argument-hint: <PR number>
allowed-tools: Read, Write, Grep, Glob, Bash, mcp__github__get_pull_request, mcp__github__get_pull_request_files, mcp__github__create_pull_request_review
---

# PR Review

Create a GitHub PR review with summary and inline diff comments. Writes to a file for preview — never posts automatically.

## Arguments

`/pr-review <PR number> [--output path]`

- `<PR number>` — required, the PR to review (e.g., `74`, `#74`)
- `--output path` or `-o path` — optional, where to write the review file

## Instructions

1. **Parse Arguments**
   - Extract PR number from `$ARGUMENTS` (strip `#` prefix if present)
   - Check for `--output` / `-o` flag

2. **Gather PR Information**
   - Fetch PR metadata:
     ```bash
     gh pr view <number> --json title,author,baseRefName,headRefName,headRefOid,number
     ```
   - Get the full diff: `gh pr diff <number>`
   - Get commit list:
     ```bash
     gh pr view <number> --json commits --jq '.commits[] | "\(.oid[:7]) \(.messageHeadline)"'
     ```

3. **Look for Existing Code Review**

   Check if a `/code-review` has already been written for this PR:
   - Extract issue number from the PR branch name (e.g., `feature/69-...` → `69`)
   - Look for `.ai/issue-<number>-*/review-pr<number>-*.md`
   - Also check `.ai/pr-<number>/review-*.md`

   If found, read it and use it as the basis for the GitHub review — distill the findings into the concise posting format. You still need the diff to map inline comments to correct line numbers.

   If not found, perform a fresh review:
   - Read the full diff and changed files
   - Review against **[../code-review/REVIEW_CRITERIA.md](../code-review/REVIEW_CRITERIA.md)**
   - Read project guidelines as needed (CODING_GUIDELINES.md, TESTING_GUIDELINES.md, COMMIT_GUIDELINES.md)

4. **Determine Review Event**

   Based on findings:
   - `APPROVE` — no issues, or nits only
   - `COMMENT` — only "should fix" or informational items
   - `REQUEST_CHANGES` — any "must fix" items

5. **Map Inline Comments to Diff Lines**

   For each finding that warrants an inline comment:
   - Get the PR diff for the target file
   - Identify the correct line number in the **new version** of the file
   - Only comment on lines visible in the diff (added lines, or context lines within diff hunks)
   - If a finding references code outside the diff, include it in the review summary instead

   Keep inline comments concise — one finding per comment. The summary provides the overview; inline comments provide the specifics.

6. **Determine Output Location**

   ```bash
   HEAD_SHA=$(gh pr view <number> --json headRefOid --jq '.headRefOid' | cut -c1-7)
   FILENAME="pr-review-${HEAD_SHA}.md"
   ```

   If `--output` / `-o` was provided:
   - If it starts with `.ai/`, use it directly as the folder
   - Otherwise, treat it as a folder name within `.ai/`
   - Write to `<folder>/<filename>`

   If no `--output`:
   - Extract issue number from PR branch name if possible
   - Look for existing `.ai/issue-<number>-*` folder
   - Fall back to `.ai/pr-<number>/`
   - Write to `<folder>/<filename>`

7. **Write Review File**

   Use this exact format:

   ````markdown
   ---
   pr: <number>
   event: <APPROVE|COMMENT|REQUEST_CHANGES>
   ---

   <Opening paragraph: 1-3 sentences. Overall verdict — what's good,
   what the PR does well. Professional, concise tone.>

   ### Must Fix
   - <finding> — `file:line` — <brief explanation>

   ### Should Fix
   - <finding> — `file:line` — <brief explanation>

   ### Nit
   - <finding> — `file:line` — <brief explanation>

   ## Inline Comments

   ### `<file>:<line>`
   <Comment body — concise, actionable. Can use markdown.>

   ### `<file>:<line>`
   <Comment body>
   ````

   **Format rules:**
   - Only include severity sections that have items
   - No checkboxes, no emoji
   - Opening paragraph has no heading — it IS the top-level content
   - Inline comment headers use the exact format `### \`file:line\`` for parseability
   - Keep everything concise — this gets posted publicly

8. **Report to User**
   - Show the file path
   - Show a brief summary of findings (e.g., "1 must-fix, 1 nit, 2 inline comments")
   - Remind: "Review the file, then tell me to post it."

## Posting

When the user asks to post the review (e.g., "post it", "submit the review"):

1. Read the review file
2. Parse frontmatter for `pr` number and `event` type
3. Extract the review body (everything between frontmatter and `## Inline Comments`)
4. Parse inline comments from `## Inline Comments` section:
   - Each `### \`file:line\`` header defines a comment
   - The body below it (until the next `###` or end) is the comment text
5. Post using `mcp__github__create_pull_request_review` with:
   - `pull_number` from frontmatter
   - `event` from frontmatter
   - `body` = the review summary text
   - `comments` = array of `{path, line, body}` from parsed inline comments
6. Report the posted review URL
