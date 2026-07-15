---
name: validate-tests
description: Codex-gate the test plan in an implementation plan before implementation starts
allowed-tools: Read, Grep, Glob, Edit, Write, Bash, Agent
---

# Validate Tests (Codex Gate)

Gate the test plan of an implementation plan before any implementation
starts. Combines a local check against the testing guidelines with an
external Codex review, following the shared convention in
[../_shared/CODEX_GATE.md](../_shared/CODEX_GATE.md).

This is a **required step** between `/create-plan` and `/implement`. Once
the plan passes this gate, the test plan is authoritative — `/implement`
treats it as immutable.

## Instructions

1. **Find the Plan**
   - Detect current workflow folder from git branch
   - Read `.ai/<folder>/plan.md`
   - If no plan exists, suggest running `/create-plan` first

2. **Local Validation**

   Load TESTING_GUIDELINES.md and GIT_TEST_SCENARIOS.md, then check the
   Test Plan section:
   - Each scenario follows the guidelines (setup patterns, assertions that
     verify behavior rather than just "no error", proper Git scenario setup)
   - Pay special attention to rules marked CRITICAL in the guidelines
   - All code paths, error conditions, and edge cases have scenarios
   - Scenarios are concrete enough to implement without guessing

   Fix what you find directly in the plan.

3. **Codex Gate**

   Run a Codex gate per [../_shared/CODEX_GATE.md](../_shared/CODEX_GATE.md):

   - **Artifact**: the Test Plan section of `.ai/<folder>/plan.md`
   - **Task for Codex**: find missing scenarios, wrong or underspecified
     expected outcomes, and scenarios that don't match the source of truth.
     TDD framing: these tests define the design — gaps found now are cheap,
     gaps found during implementation are expensive
   - **Source of truth**: the spec issue or `analysis.md`/`concept.md` the
     plan was created from (pass its content or path)
   - **Guidelines**: TESTING_GUIDELINES.md, GIT_TEST_SCENARIOS.md

4. **Evaluate and Apply**

   Evaluate every Codex finding per the convention: apply only with high
   confidence, reject over-engineering and hypothetical scenarios, when in
   doubt leave it out. Update the plan's Test Plan section with applied
   findings.

   Log all findings — applied, partial, and rejected with reasons — to
   `.ai/<folder>/codex-test-plan.md`.

5. **Report Findings**

   Summarize:
   - Local validation: issues found and fixed
   - Codex gate: findings applied / rejected (with the log path)
   - Open questions that need the user (genuine design decisions only)
   - Confirmation that the plan is gated and ready for `/implement`
