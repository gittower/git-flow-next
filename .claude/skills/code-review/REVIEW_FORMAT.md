# PR Review Output Format

## Overview

The PR review produces a structured summary designed for human reviewers. It provides two quick-glance signals (verdict and impact), a brief assessment, categorized findings by severity, a detailed test coverage evaluation, and a collapsible AI fix prompt for automated remediation.

## Structure

### Header (no heading, starts the review)

Two inline status indicators followed by a 1-3 sentence assessment:

- **Verdict**: One of three values:
  - `✅ **Approved**` — Good to merge, no issues.
  - `✅ **Approved with notes**` — Good to merge, but review the notes.
  - `🚫 **Changes requested**` — Do not merge until issues are resolved.
- **Impact**: One of three values:
  - `🔴 **High impact**` — Touches critical paths, wide scope of impact, data loss or security implications if wrong.
  - `🟡 **Medium impact**` — Meaningful logic changes but scoped impact, moderate scope of impact.
  - `🟢 **Low impact**` — Config, docs, refactors, isolated additions, minimal scope of impact.

The assessment summary briefly explains why those two states were chosen. It should describe what the changeset does, how confident the review is, and what the scope of impact is if something goes wrong.

### Severity Sections (h3)

Findings are grouped by severity, not by category (security, code quality, etc.). Each item is a concise one-liner describing what is wrong without file or line references — inline diff comments carry that detail.

- `### Must fix` — Must fix before merge. Issues that would cause failures, data loss, security vulnerabilities, or violate critical project guidelines.
- `### Should fix` — Expected to be addressed but won't block merge. Convention violations, gaps in documentation, maintainability concerns.
- `### Nit` — Optional improvements at the author's discretion. Style, additional tests, minor optimizations.

If a severity section has no items, omit it entirely.

### Test Coverage Assessment (h3)

Always present. Starts with a summary sentence evaluating whether tests are sufficient to merge with confidence.

Contains up to three subsections (h4):

- `#### Test issues` — Problems with existing tests: guideline violations, broken setup, tautological assertions, missing isolation, shared state mutation, tests that can never fail, wrong assertion targets. Each item tagged with severity: `[must fix]`, `[should fix]`, or `[nit]`.
- `#### Existing tests` — Table of tests in the changeset with columns: Test file, What it covers (a brief explanation of the scenario being tested — what setup, action, and expected outcome the test verifies), Evaluation (✅ Solid, ⚠️ with note).
- `#### Missing tests` — Tests that should exist but don't. Each tagged with `[impact: high]`, `[impact: medium]`, or `[impact: low]`.

If a subsection has no items, omit it.

### AI Fix Prompt (collapsible, at the bottom)

A single collapsible `<details>` block at the very end, separated by a horizontal rule. Contains numbered fix instructions organized by severity category. Each item provides enough context for an AI agent to locate and fix the issue: file path, what is wrong, and what the fix should look like. Fix instructions should reference the project's conventions and patterns where applicable.

The numbering is continuous across categories so items are easy to reference. The user can copy the prompt, remove items they don't want addressed, and paste it into an AI coding agent.

## Rules

1. All sections use h3 (`###`). Subsections within Test Coverage use h4 (`####`).
2. The summary items (Must fix, Should fix, Nit) stay concise and do not include file/line details — inline diff comments provide that context.
3. The Test Coverage Assessment is always present and always includes the summary sentence.
4. If a severity section has no items, omit the section entirely.
5. If an area defined in the project's review guidelines was evaluated and had no findings, state what was evaluated and that no issues were found in the assessment summary. The reviewer should know which areas were checked.
6. The verdict and impact are independent dimensions: verdict reflects quality of the change, impact reflects consequence of getting it wrong.
   Verdict is determined by the highest severity present: **Must fix** → Changes requested, **Should fix** (with or without Nit) → Approved with notes, **Nit** only or no findings → Approved.
7. Severity definitions:
   - **Must fix**: Issues that would cause failures, data loss, security vulnerabilities, or violate critical project guidelines. The PR must not merge until these are resolved.
   - **Should fix**: Issues that affect maintainability, violate project conventions, or have gaps that should be addressed. Won't block merge but expected to be resolved.
   - **Nit**: Optional improvements. Style preferences, additional test ideas, minor optimizations. Author's discretion.

## Review Scope

The review evaluates the changeset against the project's review guidelines. The project must provide its own review guidelines that define what to check and what standards to enforce. This format does not prescribe what to look for — it prescribes how to present findings.

Typical review areas include security, code quality & architecture, test coverage, performance, and formal checks (documentation, commit messages, style). The project's review guidelines define the specifics for each area.

All findings from all areas are merged into the severity sections. If an area was evaluated and had no findings, mention it in the assessment summary so the reviewer knows it was checked.

## Concrete Examples

### Example 1: Changes requested with must-fix issues

```markdown
🚫 **Changes requested** · 🔴 **High impact**

Modifies the core data processing pipeline and introduces a new
caching layer that multiple modules depend on. Two must-fix issues
prevent merge: a silently ignored error in a critical path and a
flaky test. The scope of impact is high — failures here could cause
silent data loss. Security and performance were reviewed with no
concerns.

### Must fix
- Silently ignored error in directory creation could fail without indication on restricted systems.
- Test mutates shared state without cleanup, causing ordering-dependent flaky failures.

### Should fix
- Business logic mixed into the request handler instead of the service layer, violating project architecture guidelines.
- No documentation update for the new `--force` flag.

### Nit
- Configuration value used directly instead of going through the resolution chain defined in project guidelines.
- Imports not grouped according to project style conventions.

### Test Coverage Assessment
Tests cover the happy path well but lack edge case and error path
coverage for the new processing logic.

#### Test issues
- **[must fix]** `processor_test.go::TestProcess` — Mutates shared state
  without cleanup, will cause ordering-dependent flaky failures.

#### Existing tests
| Test file | What it covers | Evaluation |
|-----------|---------------|------------|
| `processor_test.go::TestProcessBasic` | Processes a standard input with default config and verifies the output matches expected transformation | ✅ Solid |
| `processor_test.go::TestProcessEmpty` | Processes an empty input and checks that the output is empty without errors | ⚠️ Shallow — only asserts no error, doesn't verify output state |

#### Missing tests
- **[impact: high]** No test for concurrent access to the shared cache
- **[impact: medium]** Error path when upstream service returns 429 is untested

---

<details><summary>🤖 AI fix prompt</summary>

## Must fix

1. In `internal/processor/setup.go`, the error returned by `os.MkdirAll`
   is not checked. Wrap the call in proper error handling and return a
   descriptive error using the project's error handling conventions.

2. In `processor_test.go::TestProcess`, the test modifies shared state
   but has no cleanup. Add cleanup that resets the state after each test,
   or refactor to use an isolated fixture per test case.

## Should fix

3. In `handlers/process.go`, move the business logic out of the HTTP
   handler into the service layer, following the project's layered
   architecture pattern.

4. Add documentation for the new `--force` flag. Include a description
   of its behavior, default value, and any interaction with existing flags.

## Nit

5. In `internal/processor/config.go`, update the configuration resolution
   to follow the project's config precedence chain instead of reading the
   flag value directly.

6. Reorganize imports in affected files to follow the project's import
   grouping conventions.

</details>
```

### Example 2: Approved with notes, no must-fix issues

```markdown
✅ **Approved with notes** · 🟡 **Medium impact**

Adds a new machine-readable output mode for the status endpoint. The
implementation follows existing patterns well and test coverage is
solid. A few minor items to consider around documentation and code
structure. Security, performance, and architecture were reviewed with
no concerns.

### Should fix
- No documentation for the new `--format` flag.

### Nit
- The output formatter could share an interface with the existing pretty-printer to reduce duplication.
- Commit message on `a1b2c3d` exceeds the project's subject line length limit.

### Test Coverage Assessment
Good coverage across output formats and edge cases. Tests are well
isolated and assertions are meaningful.

#### Existing tests
| Test file | What it covers | Evaluation |
|-----------|---------------|------------|
| `status_test.go::TestMachineOutput` | Runs status with mixed entity states and verifies the machine-readable output matches the expected format | ✅ Solid |
| `status_test.go::TestMachineOutputEmpty` | Runs status with no entities present and checks for correct empty output | ✅ Solid |
| `status_test.go::TestMachineOutputNested` | Creates nested entity structures and verifies paths appear with correct prefixes | ✅ Solid |

#### Missing tests
- **[impact: low]** No test for extremely long entity names in machine-readable output

---

<details><summary>🤖 AI fix prompt</summary>

## Should fix

1. Add documentation for the new `--format` flag. Include a description
   of the output format, its intended use in automation, and how it
   differs from the default human-readable output.

## Nit

2. Extract a shared formatter interface that both the pretty-printer
   and machine-readable formatter implement, reducing duplication in
   header and entry rendering logic.

3. Amend commit `a1b2c3d` to shorten the subject line per the project's
   commit message conventions.

</details>
```

## Follow-up Reviews

When new commits are pushed to a branch that has already been reviewed, the AI reviews only the new commits and produces a follow-up review. The follow-up review must not repeat findings from the previous review — it builds on top of it.

### Follow-up Header

Same verdict and impact format as an initial review, but the assessment summary must state:

- The commit range reviewed (e.g. `a1b2c3d`..`f4e5d6a`)
- How many commits are covered
- A link or reference to the previous review it builds on
- An updated overall assessment reflecting the current state of the PR

The verdict and impact reflect the **current state of the entire PR**, not just the new commits. If the previous review had must-fix issues and the new commits resolve them, the verdict should update accordingly.

### Follow-up Sections

The follow-up review uses three sections to track state changes, followed by an incremental test coverage assessment:

- `### Resolved from previous review` — Items from the prior review that have been addressed. Use strikethrough on the original finding with a brief note on how it was resolved.
- `### Still open from previous review` — Items from the prior review that remain unaddressed. Listed without re-explaining — the previous review has the detail.
- `### New findings` — New issues introduced by the new commits. Uses the same severity subsections (`#### Must fix`, `#### Should fix`, `#### Nit`). If a severity level has no new findings, omit it.

If a section would be empty, omit it entirely. For example, if all previous items are resolved and there are no new findings, only the "Resolved" section appears.

### Follow-up Test Coverage Assessment

Incremental — only covers tests added or changed in the new commits. Updates the missing tests list from the prior review (noting what was addressed and what remains). Does not repeat the full test table from the previous review.

### Follow-up AI Fix Prompt

Combines still-open items from the previous review and new findings into a single actionable prompt, organized by category (`## Still open`, `## New findings` with severity subsections).

### Example 3: Follow-up review after fixes

```markdown
✅ **Approved** · 🔴 **High impact**

Follow-up review covering commits `a1b2c3d`..`f4e5d6a` (3 commits),
building on [previous review](#link-to-previous-review).

The must-fix issues from the prior review have been resolved. Error
handling is now in place and the flaky test has been fixed. One new
nit introduced by the new commits. All areas reviewed with no
remaining concerns.

### Resolved from previous review
- ~~Silently ignored error in directory creation~~ — Fixed, proper error handling added.
- ~~Test mutates shared state without cleanup~~ — Fixed, uses isolated fixtures now.
- ~~No documentation update for the new `--force` flag~~ — Fixed, documentation added.

### Still open from previous review
- Business logic mixed into the request handler instead of the service layer.

### New findings

#### Nit
- New helper function in `utils/parse.go` duplicates existing logic in `utils/format.go`.

### Test Coverage Assessment
Previous gaps have been partially addressed. Concurrent access test
added. Error path for 429 responses still untested.

#### Existing tests
| Test file | What it covers | Evaluation |
|-----------|---------------|------------|
| `processor_test.go::TestProcessConcurrent` | Spawns multiple goroutines accessing the shared cache simultaneously and verifies no data races or corruption | ✅ Solid |

#### Missing tests
- **[impact: medium]** Error path when upstream service returns 429 is still untested

---

<details><summary>🤖 AI fix prompt</summary>

## Still open

1. In `handlers/process.go`, move the business logic out of the HTTP
   handler into the service layer, following the project's layered
   architecture pattern.

## New nit

2. In `utils/parse.go`, the new `extractFields` function duplicates
   logic already present in `utils/format.go::parseFields`. Reuse the
   existing function or extract a shared helper.

</details>
```
