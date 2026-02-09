---
name: code-review
description: PR code review mechanics via GitHub API
allowed-tools: Bash, Read, Write, Grep, Glob
---

# Code Review Skill

PR code review mechanics: create reviews via GitHub API, post inline comments, respond to mentions.

## Input Context

The workflow provides:
- `EVENT`: GitHub event type (`pull_request`, `issue_comment`, `pull_request_review_comment`)
- `REPO`: Repository in `owner/repo` format
- `PR_NUMBER`: Pull request number
- `COMMENT`: Comment body (empty for `pull_request` events)

## Execution Context Detection

Run these two commands (in parallel) to check for CI environment:

```bash
echo $GITHUB_ACTIONS
```

```bash
echo $CI
```

If either prints `true`, you are in **CI mode**. If both are empty, you are in **local mode**.

**Execution modes:**
- **CI mode**: `GITHUB_ACTIONS=true` or `CI=true` — Posts reviews to GitHub
- **Local mode**: Neither set — Auto-enables dry-run, outputs review to console

When running locally, inform the user:
> "Running in local mode — dry-run enabled automatically. Review output will be displayed but not posted to GitHub."

---

## Intent Detection

```
If EVENT == "pull_request":
  → INTENT = "review"

If EVENT == "issue_comment" or "pull_request_review_comment":
  If COMMENT contains "@claude review" or "@claude re-review":
    → INTENT = "review"
  Else:
    → INTENT = "respond"
```

---

## Loading Guidelines

Before reviewing or responding, load project-specific guidelines:

1. **Read CLAUDE.md** (if exists) - may contain review instructions
2. **Follow references** - if CLAUDE.md mentions other guideline files, read them
3. **Read `REVIEW_CRITERIA.md`** (in this skill's directory) - defines what to evaluate (review criteria, checklists, severity definitions)
4. **Read `REVIEW_FORMAT.md`** (in this skill's directory) - defines the output structure for all review output

**Two concerns, two documents:**
- `REVIEW_CRITERIA.md` defines **what to evaluate** (review criteria, checklists, severity definitions)
- `REVIEW_FORMAT.md` defines **how to present findings** (output structure, sections, formatting rules)

---

## Action: Review

### Pre-Review Check

Determine what needs reviewing:

```bash
# Get Claude's previous reviews
LAST_REVIEW=$(gh api repos/$REPO/pulls/$PR_NUMBER/reviews \
  --jq '[.[] | select(.user.login == "github-actions[bot]" or .user.login == "claude[bot]")] | sort_by(.submitted_at) | last')

# Get current HEAD commit
CURRENT_HEAD=$(gh pr view $PR_NUMBER --json headRefOid --jq '.headRefOid')

# Get commit SHA from last review
LAST_REVIEW_COMMIT=$(echo "$LAST_REVIEW" | jq -r '.commit_id // empty')
```

**Decision:**
- No previous review → Full review (`gh pr diff $PR_NUMBER`)
- Same commit → Complete successfully with no action (do not post anything)
- New commits in history → Incremental review
- Last commit not in history → Force-push handling

### Incremental Review

```bash
git diff $LAST_REVIEW_COMMIT..$CURRENT_HEAD
```

Only comment on lines changed in new commits. Use the **Follow-up Reviews** format from `REVIEW_FORMAT.md` — track resolved/still-open items from the previous review and present new findings separately.

### Force-Push Handling

```bash
git merge-base --is-ancestor $LAST_REVIEW_COMMIT $CURRENT_HEAD
# Exit 0 = normal push, non-zero = force-push
```

When force-push detected:
1. Full review against base branch
2. Fetch previous comments: `gh api repos/$REPO/pulls/$PR_NUMBER/comments`
3. Skip re-commenting on same file+line+issue

### Submitting the Review

Use the `Write` tool to create a JSON payload, then submit via `gh api --input`:

1. **Build the JSON payload** using the `Write` tool (avoids pipe permission issues in CI):

```json
// Write to review_payload.json using the Write tool
{
  "body": "Review summary here",
  "event": "COMMENT",
  "commit_id": "<HEAD commit SHA>",
  "comments": [
    {"path": "src/users.js", "line": 42, "body": "**Issue:** Missing input validation...\n\n<details>\n<summary>🤖 AI fix prompt</summary>\n\n```prompt\nFix instructions here...\n```\n\n</details>"},
    {"path": "src/api.js", "line": 15, "body": "**Issue:** Ignored error return value.\n\n<details>\n<summary>🤖 AI fix prompt</summary>\n\n```prompt\nFix instructions here...\n```\n\n</details>"}
  ]
}
```

2. **Get the HEAD commit SHA:**

```bash
COMMIT_SHA=$(gh pr view $PR_NUMBER --json headRefOid --jq '.headRefOid')
```

3. **Submit the review:**

```bash
gh api repos/$REPO/pulls/$PR_NUMBER/reviews --input review_payload.json
```

**IMPORTANT:** Do NOT use `jq ... | gh api ...` pipes — piped commands trigger separate permission checks in CI that cause failures. Always write the JSON payload to a file first using the `Write` tool, then pass it via `--input`.

**Review structure:**
- `body`: The review summary, formatted per `REVIEW_FORMAT.md`. Contains the header (verdict + impact + assessment), severity sections, test coverage assessment, and AI fix prompt. Severity section items are concise one-liners **without** file/line references — inline diff comments carry that detail. If some findings cannot be attached as inline comments (line number uncertain), include them in the relevant severity section with file path context as an exception.
- `comments`: File-specific findings with confident line numbers. Each comment targets a file and line number so it appears directly on the diff. Every finding that can be mapped to a specific line MUST be an inline comment.
- `event`: Map verdict to event — `"APPROVE"` for "Approved" or "Approved with notes", `"REQUEST_CHANGES"` for "Changes requested"

**CRITICAL: Single review per trigger.** Submit exactly ONE review. NEVER post separate issue comments (`gh pr comment`) for initial reviews — all feedback goes in a single review submission. The only exception is responding to re-review requests or @claude mentions, which use issue comments to reply.

**CRITICAL: No fallback comments.** If you cannot determine the correct line number for a finding, include it in the review body with the file path — do NOT post it as a separate issue comment.

### Review Output Format

The review body MUST follow the structure defined in `REVIEW_FORMAT.md`. The workflow is:

1. **Evaluate** the changeset using `REVIEW_CRITERIA.md` criteria (test coverage, coding guidelines & architecture, code quality, security, documentation, commit messages)
2. **Classify** each finding by severity: Must fix, Should fix, or Nit
3. **Format** all findings into the `REVIEW_FORMAT.md` structure:
   - Header with verdict, impact, and 1-3 sentence assessment (mention areas evaluated with no findings)
   - Severity sections with concise one-liners (omit empty sections)
   - Test Coverage Assessment (always present) with subsections as applicable
   - AI fix prompt in a collapsible `<details>` block at the bottom
4. **Attach** file-specific findings as inline diff comments — the severity sections in the body stay concise and free of file/line details

### Determining Line Numbers

The `line` field must be the line number in the NEW file version (at the PR's HEAD commit).

**Preferred method — fetch PR branch and read files directly:**

Fetch the PR branch and use standard file/git operations. Always use `origin/` prefixed refs — bare branch names like `main` may not exist as local refs in CI:

```bash
# Fetch both branches with proper remote refs
git fetch origin <branch-name>:refs/remotes/origin/<branch-name>
git fetch origin main:refs/remotes/origin/main

# View diff stats (always use origin/ prefix)
git diff origin/main...origin/<branch-name> --stat

# Read file content directly
git show origin/<branch-name>:path/to/file.go

# Or use the Read tool on fetched files
```

**IMPORTANT:** Do NOT use `FETCH_HEAD` — it changes with each `git fetch` call. Always use explicit `origin/<branch-name>` refs instead.

**Alternative — GitHub API (CI mode or when git fetch unavailable):**

```bash
# Fetch file content at PR HEAD
gh api repos/$REPO/contents/PATH?ref=$COMMIT_SHA --jq '.content' | base64 -d
```

Then search for the specific code pattern to find its line number. This is more reliable than parsing diff hunks.

**Fallback — parse from diff:**

1. Find the `@@` hunk header: `@@ -196,12 +196,41 @@` — the `+196` is the starting line in the new file
2. Count lines from hunk start: only count `+` lines and context lines (no prefix), skip `-` lines
3. For deleted lines, use `"side": "LEFT"` with the old file line number

**When uncertain:** If you cannot confidently determine the line number, do NOT guess. Include the finding in the review body instead:
> **src/users.js**: Missing input validation in `processUser()` function.

---

## Error Handling (CI Mode)

In CI mode, the skill MUST fail explicitly rather than complete silently when it cannot perform its review. Use `exit 1` to indicate failure.

### Permission Errors

After any `gh api` call, check for permission failures:

```bash
# Check if API call failed
if ! RESULT=$(gh api repos/$REPO/pulls/$PR_NUMBER/reviews --input - 2>&1); then
  echo "ERROR: Failed to submit review - $RESULT"
  exit 1
fi
```

Common permission errors to catch:
- `Resource not accessible by integration` — missing workflow permissions
- `Not Found` — repo access denied or PR doesn't exist
- `Validation Failed` — invalid review data (e.g., bad line numbers)

### Review Quality Failures

The skill MUST fail if it cannot produce a meaningful review:

1. **No findings with valid line numbers**: If line detection fails for ALL findings and none can be attached as inline comments, the review adds no value. **Exit with error** rather than posting only a summary.

2. **API submission failure**: If the review submission itself fails for any reason, **exit with error**.

```bash
# Track successful line detections
INLINE_COMMENT_COUNT=<number of comments with confirmed line numbers>

if [ "$INLINE_COMMENT_COUNT" -eq 0 ] && [ "$TOTAL_FINDINGS" -gt 0 ]; then
  echo "ERROR: Found $TOTAL_FINDINGS issues but could not determine line numbers for any. Review cannot proceed."
  exit 1
fi
```

### Success Criteria

A review is considered successful when:
- At least one inline comment has a verified line number, OR
- There are genuinely no findings (clean PR)
- The review was successfully submitted to GitHub

---

## Action: Respond

Handle @claude mentions that aren't review requests.

**IMPORTANT:** In CI mode, you MUST post your response to GitHub. Do not just produce text output — the user cannot see your output, only GitHub comments. Your response is invisible unless you post it.

### Fetch Context

```bash
# Get review comments (inline on code)
gh api repos/$REPO/pulls/$PR_NUMBER/comments --paginate

# Get issue comments (general PR discussion)
gh api repos/$REPO/issues/$PR_NUMBER/comments --paginate
```

### Posting the Response

After formulating your response, you MUST post it to GitHub:

```bash
# Reply to an inline review comment thread
gh api repos/$REPO/pulls/$PR_NUMBER/comments/$COMMENT_ID/replies \
  -f body="Response here"

# Reply to a general PR/issue comment
gh pr comment $PR_NUMBER --body "Response here"
```

**Choose the right method:**
- If the @claude mention is on an inline review comment → use the replies API with the comment ID
- If the @claude mention is on a general PR comment → use `gh pr comment`

Respond based on the question/request. If project guidelines specify a response format, use it.

---

## Dry Run Mode

When `--dry-run` is passed OR running in local mode (no CI environment detected):
- Analyze normally but DO NOT post to GitHub
- Output the exact JSON that would be submitted

````
DRY RUN - Review for PR #123

```json
{
  "body": "<Review body formatted per REVIEW_FORMAT.md>",
  "event": "REQUEST_CHANGES",
  "commit_id": "abc123...",
  "comments": [
    {"path": "src/users.js", "line": 42, "body": "**Issue:** Missing input validation...\n\n<details>\n<summary>🤖 AI fix prompt</summary>\n\n```prompt\nFix instructions here...\n```\n\n</details>"},
    {"path": "src/api.js", "line": 15, "body": "**Issue:** Ignored error return value.\n\n<details>\n<summary>🤖 AI fix prompt</summary>\n\n```prompt\nFix instructions here...\n```\n\n</details>"}
  ]
}
```
````

For responses, output the comment body:

````
DRY RUN - Response for PR #123

```markdown
Your response text here exactly as it would be posted...
```
````

---

## Exit Codes (CI Mode)

In CI mode, use exit codes to signal workflow success or failure:

| Exit Code | Meaning |
|-----------|---------|
| `0` | Review completed successfully |
| `1` | Review failed (permissions, line detection, API error) |

**Local mode** does not use exit codes — it outputs dry-run results for the user to review.
