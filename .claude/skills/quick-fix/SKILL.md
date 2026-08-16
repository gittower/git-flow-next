---
name: quick-fix
description: Handle a small fix or maintenance task locally end-to-end — no PR, but full review gates and a confirmed local merge into main
argument-hint: <description | issue-number>
allowed-tools: Task, Bash, Read, Glob, Grep, Write, Edit, Agent, mcp__github__get_issue
---

# Quick Fix

Resolve a small fix or maintenance task entirely locally: lightweight task
note instead of a spec, test-first fix on a feature branch, the same review
rigor as the PR route (local review + mandatory Codex gate), then a local
merge into main after user confirmation. No PR is created.

This repo develops single-track on `main` — feature branches are cut from
and merged back into `main`; there is no `develop` branch (see
[DEV_WORKFLOW.md](../../../DEV_WORKFLOW.md) §Branching Model).

This is a shortcut past the *process overhead* (spec issue, PR, GitHub
review), never past the *quality gates*. With no PR there is no Copilot
review, so the Codex code-review gate is mandatory here — even for tiny
diffs.

## Arguments

`/quick-fix <description | issue-number>`

- **Description**: free-text description of the fix (the usual case)
- **Issue number**: an existing GitHub issue describing the task

## Step 0: Eligibility Gate

Classify the task before doing anything. This skill must not become a
loophole around the spec route.

**Eligible:**
- Documentation fixes (typos, corrections, clarifications)
- CI / build / tooling maintenance
- Test-only changes (fixing flaky tests, improving coverage of existing behavior)
- Chores: dependency bumps, dead code removal, comment cleanups
- Small bug fixes with an obvious root cause, an obviously correct fix,
  and no design decisions
- Scope of roughly a handful of files

**Not eligible — stop and redirect:**
- New features or functionality
- CLI surface changes: new commands, flags, or configuration options
- Behavior changes users would notice (beyond restoring the documented/
  intended behavior)
- Anything requiring a design decision or with multiple plausible approaches
- Untriaged external reports (run `/triage` first)

If not eligible, **stop** and tell the user which route applies:
`/create-spec` → `/resolve-issue` for features and non-trivial fixes,
`/triage <number>` for external reports.

**Escalation rule (applies for the rest of the workflow):** if the work
grows past these bounds mid-flight — the root cause is deeper, more files
are affected, a design question appears — stop, report what was learned,
keep the branch and `.ai/` artifacts, and recommend switching to the spec
route. Do not push on.

## Step 1: Task Note

If an issue number was given, fetch the issue (`mcp__github__get_issue`)
and derive the task from it.

Derive a short kebab-case slug and create `.ai/quick-<slug>/task.md`:

```markdown
# Quick Fix: <title>

## Source
<Issue #N (link), or "ad hoc: <user's description>">

## What & Why
<2-4 sentences: the problem and the fix>

## Affected Files
- `path/to/file.go` - <what changes>

## Test Plan
<2-3 scenarios: setup / action / expected outcome — the regression test(s)
that will be written first. For docs/CI/chore changes with nothing to
test, state explicitly: "No tests: <justification>".>
```

This replaces the spec, full plan, and `/validate-tests` gate. The test
plan section is still binding: for any code change, the tests listed here
are written first.

## Step 2: Create Branch + Worktree

Create the branch in its own worktree, using the sibling-root convention (see
[DEV_WORKFLOW.md](../../../DEV_WORKFLOW.md) §3):

```bash
git worktree add -b feature/quick-<slug> ../git-flow-next.worktrees/quick-<slug> main
cd ../git-flow-next.worktrees/quick-<slug>
```

**Verify**: current directory is the new worktree and the current branch is
`feature/quick-<slug>`. Never commit directly on main. The
`.ai/quick-<slug>/` task note stays in the main clone — reference it as
`../git-flow-next/.ai/quick-<slug>/` from inside the worktree.

## Step 3: Implement (Test-First)

For code changes:

1. Write the regression test(s) from the task note's test plan
2. Verify they fail for the right reason (`go test ./...`)
3. Commit the tests using the `/commit` skill
   (read `.claude/skills/commit/SKILL.md`)
4. Implement the fix; tests are never edited to make the fix pass — if the
   test plan turns out wrong, update `task.md` explicitly and say so
5. Commit via `/commit`. If the task came from an issue, reference it
   (`Resolves #N`)

For docs/CI/chore changes: make the change, commit via `/commit`.

**Verify**:

```bash
go build ./...
go test ./...
```

Follow CODING_GUIDELINES.md, and the Documentation Requirements in
CLAUDE.md (manpage updates if any command behavior or configuration is
touched — though note that substantial doc-impacting changes are usually a
sign the task is not quick-fix eligible).

## Step 4: Local Review

Spawn a fresh subagent so the review is independent of the implementation
context:

- **subagent_type**: `general-purpose`
- **description**: `Review quick fix`
- **prompt**: `Read the skill definition at .claude/skills/code-review/SKILL.md and execute it fully. Review all new commits on the current branch vs main. The task definition is .ai/quick-<slug>/task.md — verify the change does exactly that and nothing more. Write the review with --output quick-<slug>.`

**Verify**: `.ai/quick-<slug>/review-*.md` was created.

## Step 5: Codex Code Review (Mandatory)

Run a Codex gate per `.claude/skills/_shared/CODEX_GATE.md`:

- **Artifact**: the branch diff vs main (`git diff main...HEAD`) plus the
  commit list
- **Task for Codex**: independent code review — correctness, bugs, missing
  edge cases, guideline violations. Provide `task.md` as the source of
  truth
- **Guidelines**: CODING_GUIDELINES.md, TESTING_GUIDELINES.md,
  REVIEW_CRITERIA.md

This gate is **not skippable**, regardless of diff size: it is the only
external review this change will get.

Evaluate findings per the convention (high-confidence only). Log all
verdicts to `.ai/quick-<slug>/codex-code-review.md`.

## Step 6: Apply Review Findings

Combine accepted findings from steps 4 and 5. If fixes are needed, apply
them (Test Immutability Rule applies), commit via `/commit`, and re-verify:

```bash
go build ./...
go test ./...
```

If both reviews are clean, report that and continue.

## Step 7: Merge Gate

The local equivalent of the publish gate. Present to the user:

- The task note summary and the diff stat (`git diff main...HEAD --stat`)
- The commit list
- Review outcomes (local + Codex) and any fixes applied
- The exact commands to run:
  ```bash
  go run main.go feature finish quick-<slug>
  git push origin main
  ```

Ask: **"Merge into main and push?"** (offer merge-without-push as an
alternative).

On confirmation (run from the main clone, since `main` lives there):

1. Remove the worktree first so the branch is no longer checked out:
   `cd ../git-flow-next && git worktree remove ../git-flow-next.worktrees/quick-<slug>`
2. `go run main.go feature finish quick-<slug>` — merges into main and
   deletes the branch. Fall back to
   `git checkout main && git merge --no-ff feature/quick-<slug> && git branch -d feature/quick-<slug>` if needed
3. Push main if confirmed
4. If the task came from an issue: closing it is a public action — a local
   merge fires no `Resolves #N` reference at all, so nothing closes by
   itself. Ask whether to close it now with a short comment noting the fix
   is on main and naming the merge commit. If that issue is itself a spec
   with a parent user report ("Refs #N" in its body), ask about the parent
   too — GitHub never cascades a close upwards, so a shipped report is left
   sitting open. See [ISSUE_GUIDELINES.md](../../../ISSUE_GUIDELINES.md)
   §Relation to User Reports. Leave the parent open if this fix only
   partly completes it.

**Verify**: main contains the merge, the feature branch is gone, and
`go test ./...` passes on main.

## Progress Reporting

After each step, report which step completed, key findings, and paths to
created files.

## Error Handling

If any step fails:
1. Report the failure with details
2. Ask the user whether to **retry**, **skip**, or **abort**
3. Do not automatically proceed past failures

An abort leaves the feature branch and `.ai/quick-<slug>/` in place for
inspection or escalation to the spec route.
