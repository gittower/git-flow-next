---
name: create-plan
description: Create a test-first implementation plan from a spec issue, analysis, or concept
allowed-tools: Read, Grep, Glob, Write, Bash, mcp__github__get_issue
---

# Create Implementation Plan

Generate a test-first implementation plan. The plan is built in two phases,
in the spirit of TDD: **the test plan comes first** — detailed scenarios
exercising code that does not yet exist or will change — and the
implementation outline second, serving those tests. TDD is mostly about
finding the right design and all edge cases upfront; the test plan is where
that happens.

## Instructions

1. **Detect Current Context**
   - Check current git branch to determine workflow folder
   - Branch `feature/42-something` → look for `.ai/issue-42-*/`
   - Branch `feature/my-feature` → look for `.ai/feature-my-feature/`

2. **Find the Source of Truth**

   In order of preference:
   - **Spec issue**: if the branch references a GitHub issue, fetch it. If it
     carries the `spec` label (or contains Test Scenarios / Expected Behavior
     sections per ISSUE_GUIDELINES.md), it is the source of truth
   - `analysis.md` (from /analyze-issue) or `concept.md` (for features) in
     the workflow folder
   - If neither exists, ask the user to run `/analyze-issue` first or
     provide context

3. **Read Project Guidelines**
   - Review TESTING_GUIDELINES.md and GIT_TEST_SCENARIOS.md for test rules
   - Review CODING_GUIDELINES.md for implementation standards
   - Review COMMIT_GUIDELINES.md for commit structure

4. **Phase 1: Build the Test Plan**

   This phase comes first and gets the most effort:

   - Start from the source's test scenarios (spec issues list them
     explicitly; analyses have Testing Considerations)
   - **Double-check the source**: walk the expected behavior and hunt for
     scenarios the source missed — error conditions, edge cases, interaction
     with existing config options and merge strategies, conflicting state
   - Write each scenario concretely: setup, action, expected outcome, test
     name, test file. A scenario must be implementable as a test without
     guessing
   - Scenarios exercise the *target* behavior — code that does not exist
     yet. Do not water down expected outcomes to match current behavior

5. **Phase 2: Outline the Implementation**

   Derive the implementation tasks from the test plan — what has to change
   for those tests to pass:
   - Break down into specific, actionable tasks with file paths
   - Order tasks so tests can be written and failing first (see /implement)
   - Note dependencies between tasks

6. **Write the Plan**

   Write `plan.md` in the workflow folder using the template below.

## Plan Template

Write to `.ai/<folder>/plan.md`:

```markdown
# Implementation Plan: <branch-name>

## Source
- Spec: #<number> (<link>) — or Analysis: `.ai/<folder>/analysis.md`

## Overview
<Brief summary of what will be implemented>

## Test Plan

<This section is authoritative. Tests are written from it before the
implementation, and must not be changed to make the implementation pass —
see /implement.>

### Scenario 1: <name>
- **Test**: `Test<Name>` in `test/cmd/<file>_test.go`
- **Setup**: <repository/config state before the action>
- **Action**: <command or call being exercised>
- **Expected**: <concrete outcome — output, exit code, branch/config state>

### Scenario 2: <name>
...

### Coverage Check
- [ ] Happy path(s) covered
- [ ] Error conditions covered
- [ ] Edge cases from spec/analysis covered
- [ ] Additional edge cases found while double-checking: <list or "none">

## Implementation Tasks

### Task 1: Write failing tests
**Files**: `test/cmd/<file>_test.go`
- [ ] Implement all scenarios from the Test Plan as tests
- [ ] Verify they fail for the right reason (missing behavior, not setup bugs)

### Task 2: <Name>
**Files**: `path/to/file.go`
- [ ] <Specific change>

**Details**: <context or code snippets>

### Task 3: <Name>
**Files**: `path/to/file.go`
- [ ] <Specific change>

**Depends on**: Task 2

## Documentation Updates
- [ ] `docs/<command>.1.md` - <changes needed>
- [ ] `docs/gitflow-config.5.md` - <if config changes>
- [ ] Command help text updates

## Checkpoints

| Checkpoint | After Task | Verification |
|------------|------------|--------------|
| 1 | Task 1 | Tests compile and fail on missing behavior |
| 2 | Task 2 | <which scenarios now pass> |
| 3 | Task N | `go build ./...` and `go test ./...` fully pass |

## Commit Strategy

Plan commits following COMMIT_GUIDELINES.md. Tests are committed first.

## Estimated Scope
- Files to modify: <count>
- New files: <count>
- Tests to add: <count>

## Test Plan Revisions

<Empty initially. /implement appends an entry here whenever the test plan
has to change during implementation — see the revision rule in /implement.>
```

7. **Validate Plan**
   - Ensure all affected files from the source are covered
   - Verify every spec test scenario appears in the Test Plan
   - Check documentation requirements

8. **Report Completion**
   - Show path to created plan
   - **Required next step**: `/validate-tests` — the Codex gate on the test
     plan. Implementation must not start before the gate has run.
