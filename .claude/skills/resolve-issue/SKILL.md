---
name: resolve-issue
description: Resolve a spec issue end-to-end using sequential subagents
argument-hint: <issue-number>
allowed-tools: Task, Bash, Read, Glob, Grep, Write, Agent, mcp__github__get_issue
---

# Resolve Issue

Resolve a GitHub spec issue end-to-end by orchestrating the full workflow.
Each step runs in a fresh subagent context via the Task tool so that no
single context becomes overloaded. Runs autonomously; the only user gates
are abort conditions and the final publish (push + PR creation).

## Arguments

`/resolve-issue <issue-number>`

## Precondition: Spec Issue Required

Before anything else, fetch issue #$ARGUMENTS (`mcp__github__get_issue`) and
verify it is a **spec issue**: it carries the `spec` label, or its body has
the spec structure per ISSUE_GUIDELINES.md (Goal, Expected Behavior, Test
Scenarios sections).

If it is not a spec issue, **stop** and tell the user:
- For an untriaged external report: run `/triage <number>` first
- For an accepted report or idea without a spec: run `/create-spec <number>`

If the spec has a Breakdown section (sub-issues), do not resolve the parent
— tell the user to run `/resolve-issue` per sub-issue instead.

## Workflow

Execute these steps **sequentially**. Each step uses the **Task tool** with
`subagent_type: "general-purpose"` to spawn a fresh agent. Wait for each
step to complete and verify its output before proceeding.

Between steps, you (the orchestrator) handle verification and branch
management directly.

---

### Step 1: Analyze Spec Against Codebase

Spawn a subagent:

- **subagent_type**: `general-purpose`
- **description**: `Analyze spec issue`
- **prompt**: `Read the skill definition at .claude/skills/analyze-issue/SKILL.md and execute it fully. The issue number is $ARGUMENTS. The repository is gittower/git-flow-next. The issue is a spec issue — the analysis should map its Expected Behavior and Test Scenarios onto the codebase: affected components, current behavior, root cause location for bugs.`

**Verify**: Use Glob to confirm `.ai/issue-$ARGUMENTS-*/analysis.md` was created.

---

### Step 2: Create Feature Branch + Worktree

Run directly (no subagent needed):

1. Find the `.ai/` folder created in step 1:
   ```bash
   ls -d .ai/issue-$ARGUMENTS-*
   ```
2. Extract the slug from the folder name (e.g., `issue-42-squash-merge` → `42-squash-merge`)
3. Create the feature branch in its own worktree, using the sibling-root
   convention (see [DEV_WORKFLOW.md](../../../DEV_WORKFLOW.md) §3):
   ```bash
   git worktree add -b feature/<number>-<slug> ../git-flow-next.worktrees/<number>-<slug> develop
   cd ../git-flow-next.worktrees/<number>-<slug>
   ```

**Verify**: Confirm the current directory is the new worktree and the current
branch is the feature branch.

Save the worktree path and slug in variables for subsequent steps. The `.ai/`
folder stays in the main clone — reference it as
`../git-flow-next/.ai/issue-<number>-<slug>/` from inside the worktree.

---

### Step 3: Create Test-First Plan

Spawn a subagent:

- **subagent_type**: `general-purpose`
- **description**: `Create implementation plan`
- **prompt**: `Read the skill definition at .claude/skills/create-plan/SKILL.md and execute it fully. The current branch is feature/<slug> and the workflow folder is <ai-folder>. The source of truth is spec issue #$ARGUMENTS; the codebase analysis is at <ai-folder>/analysis.md. The Test Plan section comes first and derives from the spec's Test Scenarios.`

**Verify**: Use Glob to confirm `<ai-folder>/plan.md` was created and it contains a Test Plan section.

---

### Step 4: Codex-Gate the Test Plan

Spawn a subagent:

- **subagent_type**: `general-purpose`
- **description**: `Codex-gate test plan`
- **prompt**: `Read the skill definition at .claude/skills/validate-tests/SKILL.md and execute it fully. The workflow folder is <ai-folder>. The plan is at <ai-folder>/plan.md. The source of truth is spec issue #$ARGUMENTS.`

**Verify**: Confirm `<ai-folder>/codex-test-plan.md` was created. After this
step the test plan is authoritative.

---

### Step 5: Implement

Spawn a subagent:

- **subagent_type**: `general-purpose`
- **description**: `Implement the plan`
- **prompt**: `Read the skill definition at .claude/skills/implement/SKILL.md and execute it fully. Implement from <ai-folder>/plan.md. The current branch is feature/<slug>. Use the /commit skill (read .claude/skills/commit/SKILL.md) for creating commits. Observe the Test Immutability Rule strictly: tests first, never edited to make the implementation pass; plan revisions go back through /validate-tests; abort after 3 revisions.`

**Verify** (run directly, not in subagent):
```bash
go build ./...
go test ./...
```

If the subagent reports a **3-revision abort**, stop the workflow entirely
and report to the user: the spec/plan and implementation are fundamentally
at odds and need human review. Do not continue to later steps.

If build or tests fail, report the error and ask the user whether to retry
or abort.

---

### Step 6: Local Review

Spawn a subagent:

- **subagent_type**: `general-purpose`
- **description**: `Review changes locally`
- **prompt**: `Read the skill definition at .claude/skills/code-review/SKILL.md and execute it fully. Review all new commits on the current branch vs main. The linked spec is issue #$ARGUMENTS — verify spec satisfaction per REVIEW_CRITERIA.md. Write the review to <ai-folder>/.`

**Verify**: Use Glob to confirm `<ai-folder>/review-*.md` was created.

---

### Step 7: Codex Code Review

Run a Codex gate per `.claude/skills/_shared/CODEX_GATE.md` (directly or in
a subagent):

- **Artifact**: the branch diff vs main (`git diff main...HEAD`) plus the
  commit list
- **Task for Codex**: independent code review — correctness, bugs, missing
  edge cases, guideline violations. Provide the spec (#$ARGUMENTS) as the
  source of truth
- **Guidelines**: CODING_GUIDELINES.md, TESTING_GUIDELINES.md,
  REVIEW_CRITERIA.md

Evaluate findings per the convention (high-confidence only; the local
review from step 6 is context for judging them). Log all verdicts to
`<ai-folder>/codex-code-review.md`.

---

### Step 8: Implement Review Fixes

Combine accepted findings from steps 6 and 7. **If fixes are needed**,
spawn a subagent:

- **subagent_type**: `general-purpose`
- **description**: `Implement review fixes`
- **prompt**: `Read the skill definition at .claude/skills/implement/SKILL.md and execute it fully. Implement fixes from <ai-folder>/review-*.md and the applied findings in <ai-folder>/codex-code-review.md. The current branch is feature/<slug>. The Test Immutability Rule applies. Use the /commit skill for commits.`

**If both reviews are clean**, skip this step and report that.

**Verify** (run directly):
```bash
go build ./...
go test ./...
```

---

### Step 9: PR Summary

Spawn a subagent:

- **subagent_type**: `general-purpose`
- **description**: `Generate PR summary`
- **prompt**: `Read the skill definition at .claude/skills/pr-summary/SKILL.md and execute it fully. The current branch is feature/<slug> and the workflow folder is <ai-folder>. The PR resolves spec issue #$ARGUMENTS.`

**Verify**: Use Glob to confirm `<ai-folder>/pr_summary.md` was created.

---

### Step 10: Publish Gate

Public actions wait for the user. Present:

- Summary of all changes (files modified, commits created)
- Review outcomes (local, Codex) and fixes applied
- The PR summary content
- The exact commands: `git push -u origin feature/<slug>` and
  `gh pr create --title "<title>" --body "$(cat <ai-folder>/pr_summary.md)"`

Ask: **"Publish the branch and create the PR?"**

On confirmation, push and create the PR, then report the PR URL.

---

## Progress Reporting

After each step, report to the user:
- Which step completed (e.g., "Step 4/10: Codex-Gate Test Plan - Done")
- Key output or findings
- Path to any created files

## Error Handling

If any step fails:
1. Report the failure with details from the subagent
2. Ask the user whether to **retry**, **skip**, or **abort**
3. Do not automatically proceed past failures

The 3-revision abort in step 5 is terminal — never retry it automatically.
