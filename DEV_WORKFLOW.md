# Development Workflow

This project serves as an experiment in how much of the development and maintenance lifecycle can be automated with AI-assisted tooling, while also making it easy for contributors to follow a consistent, standardized workflow.

This document describes our structured approach to development tasks, from issue creation through to merged pull requests.

## Overview

Our workflow follows a consistent pattern that ensures quality, traceability, and thorough documentation at each stage:

```
Issue/Concept → Planning → Implementation → Review → PR → Merge
```

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

## 1. Issues (Bugs, Improvements, Smaller Features)

For bug fixes, improvements, and smaller features that don't require extensive upfront design.

### Process

1. **Create Issue**
   - `/gh-issue` - Create GitHub issue following our issue guidelines
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

3. **Proceed to Planning**
   - Once concept is approved, move to [Implementation](#3-implementing)

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

1. **Create Feature Branch**
   - For issues: `feature/<issue-number>-<short-description>`
     - Example: `feature/42-add-squash-merge`
   - For larger features: `feature/<feature-name>`
     - Example: `feature/custom-branch-types`
   - Use `git flow feature start` for branch creation

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
   - Public actions (push, PR comment, description update) wait for user confirmation
   - Use `--plan-only` to stop after the evaluation and implement later via
     `/implement .ai/<folder>/review-plan-<sha>.md`

7. **Merge**
   - Squash and merge (preferred) or merge commit
   - Delete feature branch after merge
   - Close related issues

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
│   ├── analysis.md                     # Issue analysis
│   ├── plan.md                         # Implementation plan
│   └── pr_summary.md                   # PR summary
├── issue-57-branch-validation/
│   ├── analysis.md
│   ├── plan.md
│   └── pr_summary.md
└── feature-custom-branch-types/        # Feature-based work
    ├── concept.md                      # Feature concept/design
    ├── plan.md                         # Implementation plan
    └── pr_summary.md                   # PR summary
```

### Naming Convention

- **Issues**: `issue-<number>-<slug>/` (matches branch `feature/<number>-<slug>`)
- **Features**: `feature-<name>/` (matches branch `feature/<name>`)

---

## Skills Reference

The following skills are used throughout this workflow:

| Skill | Purpose | Output |
|-------|---------|--------|
| `/gh-issue` | Create GitHub issue following guidelines | GitHub issue |
| `/analyze-issue` | Analyze issue, create workflow folder | `.ai/issue-*/analysis.md` |
| `/create-plan` | Generate test-first implementation plan | `.ai/*/plan.md` |
| `/validate-tests` | Codex-gate the test plan (required before implementing) | Updates `plan.md` + `codex-test-plan.md` |
| `/implement` | Execute plan, commit properly | Code + commits |
| `/code-review` | Review code against guidelines | Review notes |
| `/commit` | Commit following guidelines | Git commit |
| `/pr-summary` | Generate PR summary | `.ai/*/pr_summary.md` |
| `/address-review` | Evaluate PR review feedback, implement valid items | `.ai/*/review-plan-<sha>.md` + commits |

---

## Quick Reference

### Starting Work on an Issue

```bash
# 1. Create GitHub issue (optional, if not exists)
/gh-issue

# 2. Analyze the issue (creates .ai/issue-42-squash-merge/)
/analyze-issue 42

# 3. Create feature branch
git flow feature start 42-squash-merge

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

# 2. Create feature branch
git flow feature start my-feature

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
