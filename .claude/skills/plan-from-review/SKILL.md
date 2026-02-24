---
name: plan-from-review
description: Create an implementation plan from PR review feedback
allowed-tools: Bash, Read, Grep, Glob, Write
---

# Plan From Review Skill

Generate an implementation plan based on PR code review feedback. Extracts action items, issues, and suggestions from review comments and creates a structured plan to address them.

## Arguments

```
/plan-from-review <pr-number> [--dry-run]
```

- `<pr-number>`: Required. The pull request number to extract review feedback from.
- `--dry-run`: Optional. Output the plan to console without writing to file.

## Execution Flow

### 1. Detect Repository

```bash
# Get repo from git remote
REPO=$(gh repo view --json nameWithOwner --jq '.nameWithOwner')
```

### 2. Fetch PR Information

```bash
# Get PR details including current HEAD commit
gh pr view $PR_NUMBER --json title,body,headRefName,baseRefName,number,headRefOid

# Get the latest review (prefer REQUEST_CHANGES or COMMENT with content)
gh api repos/$REPO/pulls/$PR_NUMBER/reviews --jq '
  [.[] | select(.body != "" and .body != null)] |
  sort_by(.submitted_at) |
  last'

# Get inline review comments
gh api repos/$REPO/pulls/$PR_NUMBER/comments --paginate
```

### 3. Analyze Review Feedback

**Do NOT mechanically parse formatted sections.** Instead, read and understand the full review conversation:

1. **Read the review body** - Understand the overall assessment
2. **Read all inline comments** - Understand specific concerns at each location
3. **Read any reply threads** - Context may clarify or resolve issues
4. **Check the review verdict** - APPROVE, REQUEST_CHANGES, or COMMENT

**Intelligently determine what's actionable:**

- What problems were identified? (bugs, missing tests, security issues, style)
- What was suggested but not required?
- What was just informational or already resolved in discussion?
- Are there implicit issues? (e.g., "this looks complex" may imply refactoring needed)

**Categorize by your assessment of severity:**
- **Must fix**: Bugs, security issues, missing critical tests, architectural problems
- **Should fix**: Missing edge case tests, documentation gaps, code clarity issues
- **Nit**: Style preferences, optional improvements, nice-to-haves

**Skip items that are:**
- Already addressed in follow-up comments
- Explicitly marked as non-blocking ("nit:", "optional:", "minor:")
- Questions that were answered without requiring changes
- Praise or approval statements

### 4. Read Project Guidelines

Before generating the plan:
- Review CODING_GUIDELINES.md for implementation standards
- Review TESTING_GUIDELINES.md for test requirements
- Review COMMIT_GUIDELINES.md for commit structure

### 5. Determine Output Location

Plans are stored with a revision prefix (short commit SHA) to support multiple review cycles:

```bash
# Get current HEAD commit (short SHA)
HEAD_SHA=$(gh pr view $PR_NUMBER --json headRefOid --jq '.headRefOid' | cut -c1-7)

# Extract issue number from branch name if present
BRANCH=$(gh pr view $PR_NUMBER --json headRefName --jq '.headRefName')
# feature/42-something -> 42
ISSUE_NUM=$(echo "$BRANCH" | grep -oE '[0-9]+' | head -1)

# Determine .ai folder
if [ -n "$ISSUE_NUM" ]; then
  AI_FOLDER=$(ls -d .ai/issue-${ISSUE_NUM}-* 2>/dev/null | head -1)
fi
if [ -z "$AI_FOLDER" ]; then
  # Create folder from PR number
  AI_FOLDER=".ai/pr-${PR_NUMBER}"
fi

# Output file includes revision
OUTPUT_FILE="$AI_FOLDER/review-plan-${HEAD_SHA}.md"
```

**File naming**: `review-plan-<commit-sha>.md`
- Example: `review-plan-c9625f7.md`
- Allows tracking multiple review cycles
- Easy to correlate plan with specific PR revision

### 6. Generate Implementation Plan

Write to `$AI_FOLDER/review-plan-<sha>.md` following the create-plan template structure:

```markdown
# Implementation Plan: Review Feedback for PR #<number>

## Source
- PR: #<number> - <title> (<link>)
- Review: <reviewer> on <date>
- Revision: `<full-commit-sha>`
- Branch: `<branch-name>`

## Overview
<Brief summary of review feedback - what needs to be addressed>

## Prerequisites
- [ ] PR branch checked out: `<branch-name>`
- [ ] Review feedback understood
- [ ] No blocking questions remain

## Implementation Tasks

### Task 1: <Issue title derived from review comment>
**Files**: `<path/to/file.go>`

**Changes**:
- [ ] <Specific change from review feedback>
- [ ] <Additional change if needed>

**Details**:
<Fix prompt content if available, or detailed description of what to change>

**Severity**: Must fix | Should fix | Nit

### Task 2: <Next issue>
**Files**: `<path/to/file1.go>`, `<path/to/file2.go>`

**Changes**:
- [ ] <Specific change>

**Details**:
<Context from review comment>

**Depends on**: Task 1 (if applicable)

**Severity**: Should fix

### Task N: Add/Update Tests
**Files**: `test/cmd/<file>_test.go`

**Changes**:
- [ ] Add `Test<FunctionName>` - <what it tests>
- [ ] Update existing test for <scenario>

**Test Requirements**: Per TESTING_GUIDELINES.md

## Test Plan

### Tests to Add/Modify
| Test Name | Purpose | File |
|-----------|---------|------|
| `TestXxx` | <from review feedback> | `test/...` |

### Verification
- [ ] All new tests pass
- [ ] Existing tests still pass
- [ ] Edge cases covered

## Documentation Updates
- [ ] `docs/<command>.1.md` - <if mentioned in review>
- [ ] `docs/gitflow-config.5.md` - <if config changes needed>
- [ ] Code comments - <if clarity issues raised>

## Checkpoints

After each checkpoint, verify:
1. `go build ./...` succeeds
2. `go test ./...` passes
3. Review issue addressed

| Checkpoint | After Task | Verification |
|------------|------------|--------------|
| 1 | Task 1 | <must-fix issue resolved> |
| 2 | Task 2 | <should-fix item addressed> |
| 3 | Task N | All tests pass |

## Commit Strategy

Plan commits following COMMIT_GUIDELINES.md:
1. Fix must-fix issues first (group related fixes)
2. Address should-fix items (separate commits for distinct concerns)
3. Implement nits (optional, can be separate PR)

Example commit messages:
- `fix(<scope>): <must-fix issue summary>`
- `test(<scope>): <add missing test>`
- `refactor(<scope>): <address should-fix item>`

## Estimated Scope
- Files to modify: <count>
- New files: <count>
- Tests to add/modify: <count>
- Must fix: <count>
- Should fix: <count>
- Nit: <count>
```

### 7. Handle Empty Reviews

If no actionable feedback found:
```markdown
# Implementation Plan: Review Feedback for PR #<number>

## Source
- PR: #<number> - <title>
- Revision: `<commit-sha>`

## Overview
No action items found in review.

## Status
The review may have been:
- An approval with no requested changes
- Comments without specific action items

**Next steps**:
- Check PR comments directly for context
- Request clarification from reviewer if needed
- Proceed with merge if approved
```

---

## Dry Run Mode

When `--dry-run` is passed:
- Fetch and parse review data normally
- Output the generated plan to console
- Do NOT write any files

````
DRY RUN - Plan from PR #123 Review

Would write to: .ai/issue-42-feature-name/review-plan-c9625f7.md

```markdown
# Implementation Plan: Review Feedback for PR #123
...
```
````

---

## Example Output

For a PR with review feedback at revision `c9625f7`:

```markdown
# Implementation Plan: Review Feedback for PR #60

## Source
- PR: #60 - Add --no-verify option to finish command
- Review: claude-bot on 2024-02-05
- Revision: `c9625f7a481f09e246e13f5f4307dd3487d0327d`
- Branch: `feature/59-add-no-verify-option-to-finish-command`

## Overview
Implementation is solid with good test coverage. One missing test case identified for squash merge strategy.

## Prerequisites
- [ ] PR branch checked out: `feature/59-add-no-verify-option-to-finish-command`
- [ ] Review feedback understood
- [ ] No blocking questions remain

## Implementation Tasks

### Task 1: Add squash merge test with --no-verify
**Files**: `test/cmd/finish_noverify_test.go`

**Changes**:
- [ ] Add `TestFinishFeatureSquashWithNoVerify` test function
- [ ] Follow TESTING_GUIDELINES.md comment format

**Details**:
Add a test that:
1. Sets up a feature branch with commits
2. Configures squash merge strategy via git config
3. Installs rejecting pre-commit hook
4. Finishes with --no-verify flag
5. Verifies the squash commit succeeds (hook bypassed)

**Severity**: Should fix

## Test Plan

### Tests to Add/Modify
| Test Name | Purpose | File |
|-----------|---------|------|
| `TestFinishFeatureSquashWithNoVerify` | Verify --no-verify works with squash strategy | `test/cmd/finish_noverify_test.go` |

### Verification
- [ ] New test passes
- [ ] Existing 7 tests still pass
- [ ] Covers the `Commit` call after squash merge

## Documentation Updates
- [ ] None required (feature already documented)

## Checkpoints

| Checkpoint | After Task | Verification |
|------------|------------|--------------|
| 1 | Task 1 | Test passes, `go test ./...` succeeds |

## Commit Strategy

```
test(finish): Add squash merge test for --no-verify flag

Adds TestFinishFeatureSquashWithNoVerify to verify that the --no-verify
flag correctly bypasses hooks during squash merge commit operations.
```

## Estimated Scope
- Files to modify: 1
- New files: 0
- Tests to add: 1
- Must fix: 0
- Should fix: 1
- Nit: 0
```

---

## Multiple Review Cycles

When a PR goes through multiple review cycles:

```
.ai/issue-59-add-no-verify/
├── analysis.md
├── plan.md                    # Original implementation plan
├── review-plan-c9625f7.md     # First review (revision c9625f7)
├── review-plan-a1b2c3d.md     # Second review after fixes
└── pr_summary.md
```

Each review plan captures the state at that revision, allowing:
- Historical tracking of review feedback
- Understanding what was fixed between revisions
- Reference for similar future issues

---

## Integration with Other Skills

After creating the plan:
- Use `/implement .ai/<folder>/review-plan-<sha>.md` to execute the fixes
- Use `/code-review` after implementing to verify changes
- Use `/commit` to create properly formatted commits
