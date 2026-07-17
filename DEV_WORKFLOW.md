# Development Workflow

This project serves as an experiment in how much of the development and maintenance lifecycle can be automated with AI-assisted tooling, while also making it easy for contributors to follow a consistent, standardized workflow.

This document describes our structured approach to development tasks, from issue creation through to merged pull requests.

## Overview

Our workflow follows a consistent pattern that ensures quality, traceability, and thorough documentation at each stage:

```
Triage → Spec → Planning (test-first) → Implementation → Review → PR → Merge
```

There are four entry points, each driven by a single high-level skill (see
[Skills Reference](#skills-reference)):

| Incoming | Entry point | Pipeline |
|----------|------------|----------|
| Weekly check-in (nothing specific) | `/scan-repo` | gather activity → classify by effort → ordered action list |
| External issue / discussion | `/triage` | classify → duplicate check → verdict → reply |
| Accepted report / own idea / concept | `/create-spec` | draft → Codex gate → acceptance → spec issue |
| Spec issue ready to build | `/resolve-issue` | analyze → test-first plan → gate → implement → review → PR |
| Small fix / maintenance (no PR needed) | `/quick-fix` | eligibility → task note → test-first fix → review gates → local merge |
| Incoming pull request | `/handle-pr` | strict review → drafted GitHub review → post |

Key artifacts are stored in `.ai/`, organized by issue or feature:

```
.ai/
├── issue-42-squash-merge/
│   ├── analysis.md
│   ├── plan.md
│   └── pr_summary.md
└── feature-custom-branch-types/
    ├── concept.md
    ├── plan.md
    └── pr_summary.md
```

> **Note**: The `.ai/` directory is not committed to git. These are working artifacts.

---

## 1. Issues & Incoming Reports

### External Reports (Issues & Discussions)

Reports from users — bug reports, feature requests, questions — go through
triage before any work starts:

1. **Triage**
   - `/triage <number>` - Classify (bug / feature / question / support),
     search open and closed issues for duplicates, analyze against the
     codebase, and propose a verdict with a draft reply
   - Verdicts: duplicate of #N, accept as bug/feature, reject, answer,
     needs info
   - The reply and any labeling/closing wait for user confirmation

2. **Spec** (for accepted bugs and features)
   - `/create-spec <number>` - Create the implementation-ready spec issue
     per [ISSUE_GUIDELINES.md](ISSUE_GUIDELINES.md): concept level, test
     scenarios as the centerpiece, sub-issues for larger work
   - Drafted locally, Codex-gated, posted after user acceptance
   - Cross-linked with the user report; the report stays open until the
     spec ships

3. **Implement**
   - `/resolve-issue <spec-number>` end-to-end, or the manual chain in
     [Implementation](#3-implementing)

### Internal Issues

For our own bug fixes and smaller improvements that don't need triage:

1. **Create Issue**
   - `/gh-issue` - Create GitHub issue following our issue guidelines
   - For anything that warrants a spec (features, non-trivial fixes), use
     `/create-spec` instead — it creates the issue in spec form
   - Use appropriate labels (bug, enhancement, etc.)
   - Reference related issues if applicable

2. **Analyze & Document**
   - `/analyze-issue <number>` - Analyze the issue
   - Creates folder `.ai/issue-<number>-<slug>/`
   - Writes analysis to `analysis.md`
   - Include:
     - Root cause analysis (for bugs)
     - Impact assessment
     - Affected files/components
     - Proposed approach
     - Edge cases to consider

3. **Add Context Comments**
   - Propose inline comments in relevant source files
   - Mark areas that need attention: `// TODO(#<issue>): <description>`
   - This helps track issue context directly in code

4. **Proceed to Planning**
   - Move to the [Implementation](#3-implementing) phase

### Small Fixes & Maintenance (Local Route)

For small, unambiguous work — docs fixes, CI/build maintenance, test-only
changes, chores, small bug fixes with an obvious root cause — the full
spec → PR → review cycle is overhead without benefit. These are handled
entirely locally:

- `/quick-fix <description | issue-number>` - Eligibility check → lightweight
  task note (`.ai/quick-<slug>/task.md`, replacing spec and plan) → feature
  branch → test-first fix → local review + **mandatory Codex gate** → local
  merge into develop after user confirmation. No PR is created.

The shortcut is past the *process* (spec issue, PR, GitHub review), not the
*gates*: with no PR there is no Copilot review, so the Codex code-review
gate cannot be skipped, and the merge into develop is user-confirmed like
any publish. Not eligible — and redirected to the normal route: new
features, CLI surface changes, user-visible behavior changes, anything
needing a design decision. If work grows past those bounds mid-flight, the
skill stops and escalates to `/create-spec` → `/resolve-issue`.

### Issue Analysis Template

```markdown
# Issue #<number>: <title>

## Summary
<Brief description of the issue>

## Analysis

### Root Cause (for bugs)
<What's causing this behavior>

### Affected Components
- `path/to/file.go` - <why>
- `path/to/other.go` - <why>

### Proposed Solution
<High-level approach>

### Edge Cases
- <Case 1>
- <Case 2>

### Testing Considerations
<What needs to be tested>
```

---

## 2. Larger Features

For significant new functionality that requires upfront design and planning.

### Process

1. **Create Concept Document**
   - Create folder `.ai/feature-<name>/`
   - Write concept to `concept.md`
   - Include:
     - Problem statement / motivation
     - Proposed solution
     - Alternative approaches considered
     - Architecture impact
     - API/CLI changes
     - Migration considerations (if applicable)
     - Open questions

2. **Concept Review**
   - Share concept with team for feedback
   - Iterate on design based on feedback
   - Resolve open questions

3. **Create Spec Issue**
   - `/create-spec` - Turn the approved concept into a public spec issue
     (Codex-gated, user-accepted; sub-issues for work that can't land as
     one PR)
   - The spec issue is the source of truth for implementation and review

4. **Proceed to Planning**
   - Move to [Implementation](#3-implementing), or run
     `/resolve-issue <spec-number>` end-to-end

### Concept Template

```markdown
# Feature: <name>

## Problem Statement
<What problem does this solve? Why is it needed?>

## Proposed Solution
<Detailed description of the approach>

## Architecture

### Components Affected
- <Component 1>: <changes>
- <Component 2>: <changes>

### New Components
- <New component>: <purpose>

## API/CLI Changes
<New commands, flags, configuration options>

## Alternative Approaches

### Option A: <name>
<Description, pros, cons>

### Option B: <name>
<Description, pros, cons>

### Decision
<Which approach and why>

## Migration / Compatibility
<Any breaking changes or migration steps>

## Open Questions
- [ ] <Question 1>
- [ ] <Question 2>
```

---

## 3. Implementing

The implementation phase transforms issues or concepts into working code.

### Process

1. **Create Feature Branch + Worktree**
   - Branch names:
     - For issues: `feature/<issue-number>-<short-description>`
       (e.g. `feature/42-add-squash-merge`)
     - For larger features: `feature/<feature-name>`
       (e.g. `feature/custom-branch-types`)
   - Every branch gets its **own git worktree**, kept in a dot-suffixed sibling
     root next to the clone so it never pollutes the main working copy and is
     never traversed by `go test ./...`, editors, or `git status`. Convention:
     `../<repo>.worktrees/<branch-leaf>/`, where `<branch-leaf>` is the part of
     the branch name after the last `/`.
     ```bash
     # from the main clone; <slug> is e.g. 42-add-squash-merge
     git worktree add -b feature/<slug> ../git-flow-next.worktrees/<slug> develop
     cd ../git-flow-next.worktrees/<slug>
     ```
   - Do the code work (planning-derived changes, tests, commits) from inside the
     worktree. **`.ai/` working artifacts stay in the main clone** — they are
     gitignored, not branch content, and some (scans) span branches; write them
     to the main clone's `.ai/` (e.g. `../git-flow-next/.ai/...`) when operating
     from a worktree.
   - When the branch is merged or abandoned, remove the worktree:
     `git worktree remove ../git-flow-next.worktrees/<slug>`

2. **Create Implementation Plan (Test-First)**
   - `/create-plan` - Generate a test-first plan from the spec issue, analysis, or concept
   - Writes to `plan.md` in the workflow folder
   - **Phase 1 is the test plan**: detailed scenarios (setup, action,
     expected outcome) exercising the code that does not yet exist — in the
     spirit of TDD, this is where the design and edge cases are found
   - Phase 2 outlines the implementation tasks that make those tests pass

3. **Gate the Test Plan**
   - `/validate-tests` - Required gate between planning and implementation
   - Local check against [TESTING_GUIDELINES.md](TESTING_GUIDELINES.md) and
     [GIT_TEST_SCENARIOS.md](GIT_TEST_SCENARIOS.md)
   - External Codex review of the test plan, per the shared convention in
     `.claude/skills/_shared/CODEX_GATE.md` — findings are applied only with
     high confidence, and all verdicts are logged to
     `.ai/<folder>/codex-test-plan.md`
   - After this gate, the test plan is authoritative

4. **Implement**
   - Execute the implementation plan with `/implement`
   - Tests are written first, verified to fail for the right reason, and
     committed before production code
   - **Tests are never changed to make the implementation pass.** If plan
     and implementation conflict, the test plan is revised explicitly and
     re-gated via `/validate-tests`; after 3 such revisions the workflow
     aborts for human review
   - Follow [CODING_GUIDELINES.md](CODING_GUIDELINES.md)
   - Commit incrementally using `/commit`
   - Run tests frequently: `go test ./...`

5. **Local Review**
   - `/code-review` - Self-review against project review criteria
   - Generates review notes

6. **Address Review Findings**
   - Fix any issues identified in local review
   - Update tests if needed
   - Ensure all tests pass

### Implementation Plan Structure

The plan leads with the test plan — the authoritative section — followed by
the implementation tasks derived from it. See the full template in
[`.claude/skills/create-plan/SKILL.md`](.claude/skills/create-plan/SKILL.md).

```markdown
# Implementation Plan: <branch-name>

## Source
- Spec: #<number> (link) — or analysis/concept document

## Test Plan                    ← authoritative, written first

### Scenario 1: <name>
- Test: `TestXxx` in `test/cmd/<file>_test.go`
- Setup / Action / Expected outcome

## Implementation Tasks         ← derived from the test plan

### Task 1: Write failing tests
### Task 2..N: <changes that make them pass>

## Documentation Updates
## Checkpoints
## Test Plan Revisions          ← appended by /implement when the plan changes
```

---

## 4. Committing

All commits must follow our commit message standards.

### Process

- `/commit` - Commit changes according to [COMMIT_GUIDELINES.md](COMMIT_GUIDELINES.md)
- Uses conventional commit format:
  ```
  <type>(<scope>): <subject>

  <body>

  <footer>
  ```
- Types: `feat`, `fix`, `refactor`, `test`, `docs`, `chore`
- Keep commits atomic and focused
- Reference issues: `Resolves #<number>` or `Relates to #<number>`

### Examples

```
feat(finish): add squash merge strategy

Add --squash flag to finish command that performs a squash merge
instead of a regular merge. This creates a single commit containing
all changes from the topic branch.

Resolves #42
```

```
fix(start): validate branch name before creation

Check that the branch name doesn't contain invalid characters
before attempting to create it. Previously, git would fail with
a cryptic error message.

Resolves #57
```

---

## 5. Pull Requests

The final stage before code reaches the main branch.

### Process

1. **Publish Branch**
   - Push feature branch to remote
   - `git flow feature publish <name>` or `git push -u origin <branch>`

2. **Create PR Summary**
   - `/pr-summary` - Generate PR summary following the [PR template](.github/PULL_REQUEST_TEMPLATE.md)
   - Writes to `pr_summary.md` in the workflow folder
   - Includes:
     - Summary of changes
     - Test plan / verification steps
     - Screenshots (if UI changes)
     - Breaking changes (if any)

3. **Create Pull Request**
   - Open PR against appropriate base branch
   - Link related issues
   - Add appropriate reviewers
   - Apply relevant labels

4. **External Review (Service)**
   - AI-assisted review for additional perspective
   - Automated checks (CI, linting, tests)

5. **Human Review**
   - Team member reviews the PR
   - Address feedback
   - Iterate until approved

6. **Address Review Feedback**
   - `/address-review <pr-number>` - Evaluate reviewer comments, implement valid ones
   - Writes the evaluation to `.ai/<folder>/review-plan-<sha>.md` (one per PR revision)
   - Implements accepted changes and commits locally without confirmation
   - Replies on each inline diff thread with its verdict, then resolves the
     threads it handled (accepted, dismissed, or partial)
   - Public actions (push, PR comment/reply, thread resolution, description
     update) wait for user confirmation
   - Use `--plan-only` to stop after the evaluation and implement later via
     `/implement .ai/<folder>/review-plan-<sha>.md`

7. **Merge**
   - Squash and merge (preferred) or merge commit
   - Delete feature branch after merge
   - Close related issues

### Incoming Pull Requests

PRs from contributors (external or team) are reviewed strictly and treated
the same:

1. **Review**
   - `/handle-pr <number>` - Full review pipeline: strict local review
     against all REVIEW_CRITERIA.md areas (including scope — one PR, one
     concern — and spec satisfaction), distilled into a GitHub review
     draft, posted after user confirmation

2. **Follow-up rounds**
   - `/address-review <number>` - When we authored the PR and received
     feedback
   - Re-review via `/handle-pr` when the contributor pushes updates

3. **Stale PRs**
   - `/check-prs` - Run manually to see where every open PR stands against
     the review response window (CONTRIBUTING.md: 7 days)
   - `/takeover-pr <number>` - When the window has lapsed: supersede the PR
     with the requested changes applied on top, crediting the original
     author. Eligibility is verified strictly; all public actions are
     confirmed first

### PR Summary Format

Follow the format defined in [`.github/PULL_REQUEST_TEMPLATE.md`](.github/PULL_REQUEST_TEMPLATE.md):

```markdown
<Summary prose — no header. Describe what changed and why in 1-3 sentences.
Link to resolved issues with "Resolves #ISSUE".>

## Notes

<Optional. Call out risks, edge cases, breaking changes, or scope clarifications.
Remove this section if not applicable.>
```

Keep it concise. The checklist in the template is for author verification only — do not include it in the final PR summary.

---

## Directory Structure

```
.ai/                              # Not committed to git
├── issue-42-squash-merge/              # Issue-based work
│   ├── triage.md                       # Triage verdict (external reports)
│   ├── spec.md                         # Spec draft (posted as spec issue)
│   ├── codex-spec.md                   # Codex gate log: spec
│   ├── analysis.md                     # Codebase analysis
│   ├── plan.md                         # Test-first implementation plan
│   ├── codex-test-plan.md              # Codex gate log: test plan
│   ├── review-<sha>.md                 # Local code review
│   ├── codex-code-review.md            # Codex gate log: code
│   ├── review-plan-<sha>.md            # Review feedback evaluation (per revision)
│   └── pr_summary.md                   # PR summary
├── pr-91/                              # Incoming PR reviews
│   ├── review-pr91-<sha>.md
│   └── pr-review-<sha>.md
├── scans/                              # Weekly activity scans
│   └── scan-2026-07-15.md              # Also anchors the next scan's window
├── quick-typo-in-finish-docs/          # Local quick fixes (no PR)
│   ├── task.md                         # Lightweight task note (replaces spec + plan)
│   ├── review-<sha>.md                 # Local code review
│   └── codex-code-review.md            # Codex gate log (mandatory)
└── feature-custom-branch-types/        # Feature-based work
    ├── concept.md                      # Feature concept/design
    ├── plan.md
    └── pr_summary.md
```

Not every folder has every file — external reports start with `triage.md`,
internal features with `concept.md`; Codex gate logs appear when the
corresponding gate runs. Specs are published as GitHub issues (the public
source of truth); these local files are working artifacts and audit trail.

### Naming Convention

- **Issues**: `issue-<number>-<slug>/` (matches branch `feature/<number>-<slug>`)
- **Features**: `feature-<name>/` (matches branch `feature/<name>`)
- **Quick fixes**: `quick-<slug>/` (matches branch `feature/quick-<slug>`)
- **Worktrees**: `../<repo>.worktrees/<branch-leaf>/` — a sibling of the clone,
  one worktree per branch (e.g. `../git-flow-next.worktrees/42-squash-merge/`
  for branch `feature/42-squash-merge`). See
  [§3 Implementing](#3-implementing).

---

## Autonomy & Human Gates

The workflows are designed to run as autonomously as possible with a small,
predictable set of decision points.

**Autonomous** (no confirmation needed):
- All analysis, triage research, planning, and spec drafting
- Writing code and tests, running builds and tests
- Local reviews and Codex gates (findings applied per
  `.claude/skills/_shared/CODEX_GATE.md`)
- Commits on feature branches

**Gated** (preview + user confirmation required):
- Anything public: issue comments and replies, creating issues, posting PR
  reviews and comments, pushing branches, creating/closing PRs
- Local merge into develop (the `/quick-fix` merge gate — the no-PR
  equivalent of publishing)
- Spec acceptance before implementation starts
- Abort conditions: the 3-revision test plan abort, failed verification
  steps, genuine design questions the AI cannot decide alone

---

## Skills Reference

Skills come in two tiers:

- **High-level skills** run an entire workflow from a single command. They
  orchestrate plumbing skills (often in fresh subagent contexts), proceed
  autonomously, and stop only at the gates above. These are the everyday
  entry points.
- **Plumbing skills** perform one step and produce one artifact. High-level
  skills compose them; they remain individually invocable for manual,
  step-by-step work or reruns of a single stage.

### High-Level Skills

| Skill | Purpose | Gate(s) |
|-------|---------|---------|
| `/scan-repo` | Weekly activity scan — new discussions/issues/PRs as an ordered action list | None (read-only) |
| `/triage` | Classify + analyze an external issue/discussion, propose verdict | Reply & labels |
| `/create-spec` | Draft, Codex-gate, and publish a spec issue | Acceptance, posting |
| `/resolve-issue` | Resolve a spec issue end-to-end (plan → gate → implement → review) | Publish (push + PR) |
| `/quick-fix` | Small fix/maintenance locally, no PR (task note → fix → review gates) | Merge into develop |
| `/handle-pr` | Strictly review an incoming PR and post a GitHub review | Posting |
| `/address-review` | Evaluate review feedback on our PR, implement valid items | Push, PR reply & thread resolve |
| `/check-prs` | Report all open PRs vs the review response window | Reminders (optional) |
| `/takeover-pr` | Supersede a stale PR with requested changes, crediting the author | Push, PR, closing |
| `/full-release` | Release end-to-end (prep → tag → CI → Homebrew tap → website sync) | Push & tag |

### Plumbing Skills

| Skill | Purpose | Output |
|-------|---------|--------|
| `/gh-issue` | Create GitHub issue following guidelines | GitHub issue |
| `/analyze-issue` | Analyze issue against the codebase | `.ai/issue-*/analysis.md` |
| `/create-plan` | Generate test-first implementation plan | `.ai/*/plan.md` |
| `/validate-tests` | Codex-gate the test plan (required before implementing) | Updates `plan.md` + `codex-test-plan.md` |
| `/implement` | Execute plan tests-first, commit properly | Code + commits |
| `/code-review` | Review code against REVIEW_CRITERIA.md | `.ai/*/review-*.md` |
| `/pr-review` | Draft a GitHub review file from findings | `.ai/*/pr-review-*.md` |
| `/post-review` | Post a written review file to GitHub | GitHub review |
| `/pr-summary` | Generate PR summary | `.ai/*/pr_summary.md` |
| `/commit` | Commit following guidelines | Git commit |
| `/release` | Update changelog and version from git history | Release commit |

The shared Codex gate procedure used by several skills is defined once in
[`.claude/skills/_shared/CODEX_GATE.md`](.claude/skills/_shared/CODEX_GATE.md).

---

## Quick Reference

### End-to-End (High-Level Skills)

```bash
# Weekly: what happened, where to act, in which order
/scan-repo                # quick wins → one-command pipelines → needs thinking

# External report arrives
/triage 42                # classify, dedup, verdict, reply

# Accepted → create the spec issue
/create-spec 42           # drafts, Codex-gates, posts as #43

# Build it
/resolve-issue 43         # analyze → plan → gate → implement → review → PR

# Small fix or maintenance — handled locally, no PR
/quick-fix "fix typo in finish manpage"   # task note → fix → review gates → merge

# Incoming PR from a contributor
/handle-pr 44             # strict review, posted after confirmation

# Periodic PR hygiene
/check-prs                # who's waiting on whom
/takeover-pr 45           # if the response window lapsed

# Ship a release
/full-release             # prep → confirm → tag → CI → Homebrew → website
```

### Starting Work on an Issue (Manual Chain)

```bash
# 1. Create GitHub issue (optional, if not exists)
/gh-issue

# 2. Analyze the issue (creates .ai/issue-42-squash-merge/)
/analyze-issue 42

# 3. Create feature branch + worktree (sibling root, see §3)
git worktree add -b feature/42-squash-merge ../git-flow-next.worktrees/42-squash-merge develop
cd ../git-flow-next.worktrees/42-squash-merge

# 4. Create and validate implementation plan
/create-plan
/validate-tests

# 5. Implement and commit
/implement          # or work with Claude directly
/commit             # for each logical change

# 6. Review before PR
/code-review

# 7. Publish and create PR summary
git flow feature publish 42-squash-merge
/pr-summary
```

### Starting Work on a Feature

```bash
# 1. Create workflow folder and write concept
mkdir -p .ai/feature-my-feature
# Write concept.md manually or with Claude's help

# 2. Create feature branch + worktree (sibling root, see §3)
git worktree add -b feature/my-feature ../git-flow-next.worktrees/my-feature develop
cd ../git-flow-next.worktrees/my-feature

# 3. Create and validate implementation plan
/create-plan
/validate-tests

# 4. Implement and commit
/implement
/commit

# 5. Review and publish
/code-review
git flow feature publish my-feature
/pr-summary
```
