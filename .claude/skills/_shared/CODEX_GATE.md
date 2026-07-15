# Codex Gate Convention

A Codex gate is an external review of an artifact (spec, test plan, code) by
the Codex agent, followed by a disciplined evaluation of its feedback. Skills
reference this document instead of describing the procedure themselves, so
the rules stay in one place.

## When a Skill Calls for a Codex Gate

The skill names the artifact under review and the gate step. Run the gate
after the artifact is complete but before it is used downstream (a spec
before acceptance, a test plan before implementation, code before PR).

## 1. Invoke Codex

Spawn the Codex agent via the Agent tool with `subagent_type:
"codex:codex-rescue"`. Codex has no conversation context — hand it
everything it needs:

- **The artifact**: the file path (and PR number or commit range for code)
- **The task**: what to review it for, and what a good outcome looks like
- **The source of truth**: the spec issue or analysis the artifact derives from
- **Relevant guidelines**: paths to the guideline files that apply
  (TESTING_GUIDELINES.md and GIT_TEST_SCENARIOS.md for test plans,
  CODING_GUIDELINES.md and REVIEW_CRITERIA.md for code, ISSUE_GUIDELINES.md
  for specs)
- **Project philosophy**: quote CLAUDE.md's pragmatic, anti-over-engineering
  stance so Codex calibrates its suggestions

Ask Codex for concrete, itemized findings — each with a location, the
problem, and a suggested resolution — not a prose essay.

## 2. Evaluate the Feedback

Judge every finding independently. Apply a finding only with **high
confidence that it is correct and improves the artifact**:

**Apply when the finding identifies:**
- A real gap: missing test scenario, unhandled edge case, wrong expected
  outcome, requirement from the source of truth not covered
- A genuine correctness problem in the artifact
- A concrete simplification consistent with project philosophy

**Reject when the finding:**
- Proposes over-engineering: hypothetical scenarios, premature abstraction,
  layering the project philosophy rejects
- Conflicts with an explicit decision in the source of truth or guidelines
- Is stylistic preference without clear value
- You cannot confidently verify it is correct — **when in doubt, leave it
  out**

Partially applying is fine: accept the underlying concern, implement a
simpler resolution than suggested.

## 3. Log the Gate

Write the outcome to `.ai/<folder>/codex-<step>.md` (e.g.,
`codex-test-plan.md`, `codex-code-review.md`, `codex-spec.md`). One gate run
per file section, newest on top if the gate runs multiple times:

```markdown
## Gate: <step> — <artifact> @ <date or revision>

| # | Finding | Verdict | Reason |
|---|---------|---------|--------|
| 1 | <one-line summary> | applied | <why> |
| 2 | <one-line summary> | rejected | <why — reference philosophy/guidelines> |
| 3 | <one-line summary> | partial | <concern valid, simpler fix: what was done> |
```

The log is the audit trail — every finding appears in it, including the
rejected ones. Never silently drop feedback.

## 4. Apply and Continue

Update the artifact with the applied findings, then continue the skill. A
Codex gate never blocks on the user; if a finding raises a question only the
user can answer (a genuine design decision, not a judgment call), record it
as an open question in the artifact and surface it at the skill's next user
gate.
