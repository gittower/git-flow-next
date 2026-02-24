---
name: resolve-issue
description: Resolve a GitHub issue end-to-end using sequential subagents
argument-hint: <issue-number>
allowed-tools: Task, Bash, Read, Glob, Grep, Write
---

# Resolve Issue

Resolve a GitHub issue end-to-end by orchestrating the full workflow. Each step runs in a fresh subagent context via the Task tool so that no single context becomes overloaded.

## Arguments

`/resolve-issue <issue-number>`

## Workflow

Execute these steps **sequentially**. Each step uses the **Task tool** with `subagent_type: "general-purpose"` to spawn a fresh agent. Wait for each step to complete and verify its output before proceeding to the next.

Between steps, you (the orchestrator) handle verification and branch management directly.

---

### Step 1: Analyze Issue

Spawn a subagent:

- **subagent_type**: `general-purpose`
- **description**: `Analyze GitHub issue`
- **prompt**: `Read the skill definition at .claude/skills/analyze-issue/SKILL.md and execute it fully. The issue number is $ARGUMENTS. The repository is gittower/git-flow-next.`

**Verify**: Use Glob to confirm `.ai/issue-$ARGUMENTS-*/analysis.md` was created.

---

### Step 2: Create Feature Branch

Run directly (no subagent needed):

1. Find the `.ai/` folder created in step 1:
   ```bash
   ls -d .ai/issue-$ARGUMENTS-*
   ```
2. Extract the slug from the folder name (e.g., `issue-42-squash-merge` → `42-squash-merge`)
3. Create a feature branch:
   ```bash
   go run main.go feature start <number>-<slug>
   ```
   If that fails, fall back to:
   ```bash
   git checkout -b feature/<number>-<slug> develop
   ```

**Verify**: Confirm current branch is the feature branch.

Save the `.ai/` folder path and slug in variables for subsequent steps.

---

### Step 3: Create Plan

Spawn a subagent:

- **subagent_type**: `general-purpose`
- **description**: `Create implementation plan`
- **prompt**: `Read the skill definition at .claude/skills/create-plan/SKILL.md and execute it fully. The current branch is feature/<slug> and the workflow folder is <ai-folder>. Read the analysis at <ai-folder>/analysis.md as input.`

**Verify**: Use Glob to confirm `<ai-folder>/plan.md` was created.

---

### Step 4: Validate Tests

Spawn a subagent:

- **subagent_type**: `general-purpose`
- **description**: `Validate test approach`
- **prompt**: `Read the skill definition at .claude/skills/validate-tests/SKILL.md and execute it fully. The workflow folder is <ai-folder>. The plan is at <ai-folder>/plan.md.`

**Verify**: Read the plan.md to confirm the test section was reviewed or updated.

---

### Step 5: Implement

Spawn a subagent:

- **subagent_type**: `general-purpose`
- **description**: `Implement the plan`
- **prompt**: `Read the skill definition at .claude/skills/implement/SKILL.md and execute it fully. Implement from <ai-folder>/plan.md. The current branch is feature/<slug>. Use the /commit skill (read .claude/skills/commit/SKILL.md) for creating commits.`

**Verify** (run directly, not in subagent):
```bash
go build ./...
go test ./...
```

If either fails, report the error and ask user whether to retry or abort.

---

### Step 6: Local Review

Spawn a subagent:

- **subagent_type**: `general-purpose`
- **description**: `Review changes locally`
- **prompt**: `Read the skill definition at .claude/skills/code-review/SKILL.md and execute it fully. Review all new commits on the current branch vs main. Write the review to <ai-folder>/.`

**Verify**: Use Glob to confirm `<ai-folder>/review-*.md` was created.

---

### Step 7: Implement Review Fixes

Read the review file from step 6. Check for blocking issues or warnings.

**If issues found**, spawn a subagent:

- **subagent_type**: `general-purpose`
- **description**: `Implement review fixes`
- **prompt**: `Read the skill definition at .claude/skills/implement/SKILL.md and execute it fully. Implement fixes from the review at <ai-folder>/review-*.md. The current branch is feature/<slug>. Use the /commit skill (read .claude/skills/commit/SKILL.md) for creating commits.`

**If no issues found**, skip this step and report that the review was clean.

**Verify** (run directly):
```bash
go build ./...
go test ./...
```

---

### Step 8: PR Summary

Spawn a subagent:

- **subagent_type**: `general-purpose`
- **description**: `Generate PR summary`
- **prompt**: `Read the skill definition at .claude/skills/pr-summary/SKILL.md and execute it fully. The current branch is feature/<slug> and the workflow folder is <ai-folder>.`

**Verify**: Use Glob to confirm `<ai-folder>/pr_summary.md` was created.

---

## Progress Reporting

After each step, report to the user:
- Which step completed (e.g., "Step 3/8: Create Plan - Done")
- Key output or findings
- Path to any created files
- Whether proceeding to next step

## Error Handling

If any step fails:
1. Report the failure with details from the subagent
2. Ask the user whether to **retry**, **skip**, or **abort**
3. Do not automatically proceed past failures

## Completion

When all steps complete, report:
- Summary of all changes made (files modified, commits created)
- Path to the PR summary file
- The `gh pr create` command to create the PR
- Remind to push the branch first if not yet pushed
