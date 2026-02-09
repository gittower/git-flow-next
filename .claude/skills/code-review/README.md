# Code Review Skill

Automated PR code review via the GitHub API. Runs in CI (GitHub Actions) or locally (dry-run).

## How It Works

```
PR opened / new commits pushed / @claude mention
        │
        ▼
┌─────────────────┐
│ Intent Detection │
│                 │
│  pull_request ──────► review
│  @claude review ────► review
│  @claude <other> ───► respond
└────────┬────────┘
         ▼
┌─────────────────┐
│ Load Guidelines  │
│                 │
│  CLAUDE.md ──────► project instructions
│  REVIEW_         │
│  GUIDELINES.md ──► what to evaluate
│  REVIEW_         │
│  FORMAT.md ──────► how to present findings
└────────┬────────┘
         ▼
┌─────────────────┐     ┌──────────────────────┐
│ Pre-Review Check │────►│ Review Type Decision  │
│                 │     │                      │
│  No prior review ────► Full review           │
│  Same commit ────────► "No new changes"      │
│  New commits ────────► Follow-up review      │
│  Force-push ─────────► Full review (deduped) │
└────────┬────────┘     └──────────────────────┘
         ▼
┌─────────────────┐
│ Evaluate & Post  │
│                 │
│  Classify findings by severity
│  Format review body
│  Attach inline diff comments
│  Submit single review via API
└─────────────────┘
```

## Files

| File | Purpose |
|------|---------|
| `SKILL.md` | Execution logic: intent detection, pre-review checks, API submission, error handling |
| `REVIEW_FORMAT.md` | Output format: review structure, severity definitions, verdict rules, examples |
| `REVIEW_CRITERIA.md` | What to evaluate: checklists for tests, architecture, security, code quality |

## Severity Levels

Findings are classified into three severity levels:

| Severity | Meaning | Verdict Effect |
|----------|---------|----------------|
| **Must fix** | Bugs, security issues, data loss, critical guideline violations | Changes requested (`REQUEST_CHANGES`) |
| **Should fix** | Convention violations, documentation gaps, maintainability concerns | Approved with notes (`APPROVE`) |
| **Nit** | Style preferences, minor optimizations, optional improvements | Approved (`APPROVE`) |

The verdict is determined by the highest severity present:

- Any **Must fix** → `🚫 Changes requested`
- **Should fix** (with or without Nit) → `✅ Approved with notes`
- **Nit** only or no findings → `✅ Approved`

## Review Output Structure

Every review body follows this structure:

```
Header          → Verdict + Impact + 1-3 sentence assessment
Severity sections → Must fix / Should fix / Nit (omit if empty)
Test Coverage   → Always present; test issues, existing tests table, missing tests
AI fix prompt   → Collapsible <details> block with numbered fix instructions
```

Inline diff comments are attached to specific lines in the PR. Each inline comment includes a collapsible `🤖 AI fix prompt` with copy-pasteable fix instructions.

## Execution Modes

| Mode | Detection | Behavior |
|------|-----------|----------|
| **CI** | `GITHUB_ACTIONS=true` or `CI=true` | Posts review to GitHub, uses exit codes |
| **Local** | Neither set | Dry-run, outputs JSON to console |

## Review Types

### Initial Review

Full review of the entire PR diff against the base branch.

### Follow-up Review

When new commits are pushed after a previous review. Tracks resolution state:

- **Resolved from previous review** — strikethrough on fixed items
- **Still open from previous review** — listed without re-explanation
- **New findings** — new issues from the new commits, grouped by severity

The verdict reflects the current state of the entire PR, not just the new commits.

### Force-Push Review

Full review against the base branch, but deduplicates against previous inline comments to avoid re-posting the same feedback.

## Actions

### Review

Triggered by `pull_request` events or `@claude review` mentions. Produces a single GitHub review with a structured body and inline comments.

### Respond

Triggered by `@claude` mentions that aren't review requests. Posts a reply to the relevant comment thread (inline or general).

## Integration with Other Skills

| Skill | Relationship |
|-------|-------------|
| `local-review` | Same severity levels; runs locally before PR creation |
| `plan-from-review` | Extracts action items from review feedback into an implementation plan |
| `implement` | Executes fixes prioritized by severity: must fix first, then should fix, then nit |
