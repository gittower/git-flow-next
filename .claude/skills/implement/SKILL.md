---
name: implement
description: Execute an implementation plan, making changes and committing properly
allowed-tools: Read, Grep, Glob, Edit, Write, Bash
---

# Implement

Execute an implementation plan or review fixes, making code changes and committing according to guidelines.

## Arguments

`/implement [target]`

### Target (optional)

Specifies what to implement. Can be:

- **Nothing** - auto-detect from branch name, check for review.md then plan.md
- **Directory name** - folder within `.ai/` (e.g., `issue-59-add-no-verify-option`)
- **Relative path** - path to `.ai/` folder (e.g., `.ai/my-feature`)
- **File path** - direct path to a specific file (e.g., `.ai/my-feature/review.md`)

### Source Priority

When given a directory (or auto-detecting):
1. If `review.md` exists → implement fixes from the review
2. Otherwise if `plan.md` exists → implement the plan
3. If neither exists → suggest running `/create-plan` or `/local-review` first

### Examples

```bash
# Auto-detect from branch, prefer review.md over plan.md
/implement

# Use specific folder in .ai/
/implement issue-59-add-no-verify-option

# Use relative path
/implement .ai/my-feature

# Use specific file directly
/implement .ai/my-feature/plan.md
/implement .ai/my-feature/review.md
```

## Instructions

1. **Parse Arguments and Find Source**

   If a file path was provided (ends with `.md`):
   - Use that file directly
   - Determine if it's a plan or review from filename

   If a directory/folder was provided:
   - If it starts with `.ai/`, use it directly
   - Otherwise, treat it as a folder name within `.ai/`
   - Check for `review.md` first, then `plan.md`

   If no argument provided:
   - Extract issue number from branch name (e.g., `feature/59-...` → `59`)
   - Look for existing `.ai/issue-<number>-*` folder
   - Check for `review.md` first, then `plan.md`

   If no source file found:
   - Suggest running `/create-plan` for new work
   - Suggest running `/local-review` if code exists but needs review

2. **Verify Prerequisites**
   - Confirm on correct feature branch
   - Check working directory is clean (`git status`)
   - Ensure tests pass before starting: `go test ./...`

3. **Load Guidelines**
   - Read CODING_GUIDELINES.md for implementation standards
   - Read COMMIT_GUIDELINES.md for commit format
   - Read GIT_TEST_SCENARIOS.md when implementing tests (required for Git scenario setup)

4. **Execute Work**

   ### For Plan Mode (implementing from plan.md)

   For each task in the plan:

   **Before Each Task:**
   - Review task requirements and identify all files to modify
   - Read CODING_GUIDELINES.md rules relevant to the change

   **Code Changes:**
   - Make focused, atomic changes
   - Follow existing patterns in the codebase
   - Keep changes minimal — don't over-engineer

   **After Each Task:**
   - Verify build: `go build ./...`
   - Run tests: `go test ./...`
   - Review changes: `git diff`

   ### For Review Mode (implementing from review.md)

   For each issue found in the review:

   **Must fix:**
   - Address all must-fix issues before continuing
   - These prevent the PR from being merged

   **Should fix:**
   - Address should-fix items that improve code quality
   - Document any intentionally skipped with rationale

   **Nit:**
   - Consider implementing if they improve the code
   - Skip if they add unnecessary complexity

   **After Fixes:**
   - Verify build: `go build ./...`
   - Run tests: `go test ./...`
   - Review changes: `git diff`

5. **Commit Strategy**

   ### When to Commit
   - After completing a logical unit of work
   - After each checkpoint in the plan (plan mode)
   - After fixing a category of issues (review mode)
   - Keep commits atomic and focused
   - Use `/commit` skill for proper formatting

6. **Checkpoint Verification**

   At each checkpoint:
   - [ ] Build succeeds: `go build ./...`
   - [ ] Tests pass: `go test ./...`
   - [ ] Expected behavior works
   - [ ] Changes committed

7. **Track Progress**

   For plan mode - update plan.md checkboxes:
   ```markdown
   - [x] Completed task
   - [ ] Pending task
   ```

   For review mode - update review.md checkboxes:
   ```markdown
   - [x] Fixed: <issue description>
   - [ ] Pending: <issue description>
   ```

8. **Handle Issues**

   If problems arise:
   - Document the issue
   - Check if it affects the plan/review
   - Adjust approach if needed
   - Ask for clarification if blocked

## Completion

**For Plan Mode:**
1. Verify all tests pass
2. Check all checkboxes in plan.md are complete
3. Suggest running `/local-review` before PR

**For Review Mode:**
1. Verify all tests pass
2. Check all blocking issues are resolved
3. Update review.md with fixes applied
4. Ready for PR if all blocking issues fixed
