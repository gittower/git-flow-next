---
name: health-audit
description: Internal codebase health audit — checks docs, skills, duplication, dead code, coding-guidelines, correctness, and tests for drift and real defects. Runs all areas by default, or one named area. Writes per-area reports to .ai/health-audit-<date>/.
argument-hint: "[area: docs | skills | duplication | dead-code | guidelines | correctness | tests] (omit for all)"
allowed-tools: Bash, Read, Write, Glob, Grep, Task
---

# Health Audit

Sanity-check the codebase for the drift that accumulates in an AI-maintained
repo: docs falling out of sync with commands, skills referencing moved files,
copy-paste helpers, guideline violations, test-convention drift. Each area is
audited by an independent subagent grounded in its **truth source** (the code,
or a specific guideline doc), so findings are concrete and low-noise.

Read-only against the codebase: the audit never edits source, posts, or opens
issues. It only reports. Findings are meant to feed the normal pipelines
(`/gh-issue` → `/resolve-issue`, or direct doc fixes).

## Arguments

`/health-audit [area]`

- No argument → run **all seven areas** in parallel, then write a `SUMMARY.md`.
- One `area` → run only that area. Valid slugs (also the report filenames):
  `docs`, `skills`, `duplication`, `dead-code`, `guidelines`, `correctness`,
  `tests`.

## Areas and truth sources

| Slug | Audits | Truth source |
|------|--------|--------------|
| `docs` | manpages vs actual flags/config/commands | `cmd/*.go`, config keys, `version/` |
| `skills` | stale refs, overlap, doc-consistency of `.claude/skills/` | the skill files + root guideline docs |
| `duplication` | copy-paste logic, parallel implementations | source + `CODING_GUIDELINES.md` |
| `dead-code` | unused exports/files, unreachable or orphan code | source |
| `guidelines` | coding-guideline compliance, layering, config precedence | `CODING_GUIDELINES.md`, CLAUDE.md "Code Conventions" |
| `correctness` | real defects: unhandled errors, wrong exit codes, boundary/state bugs | source |
| `tests` | build/test pass, coverage gaps, convention drift | `TESTING_GUIDELINES.md`, CLAUDE.md testing sections |

## Instructions

### 1. Resolve scope and output dir

Get today's date and pick the areas to run:

```bash
date +%Y-%m-%d
```

- If an argument was given, validate it against the five slugs (reject anything
  else with the valid list) and run only that one.
- Otherwise run all seven.

The output directory is `.ai/health-audit-<today>/`. Create it. `.ai/` is
gitignored, so nothing here gets committed. If the dir already exists (a re-run
same day), overwrite the area files being regenerated; leave others intact.

### 2. Run each area as a subagent

Launch one `Task`/subagent **per selected area, in parallel** (all in one
message when running several). Each subagent gets the matching prompt below,
**prefixed** with this shared preamble:

> You are auditing the git-flow-next repo (Go CLI implementing the git-flow
> branching model) at the repository root. This is an AI-maintained repo; the
> risk you are hunting is **drift**, not catastrophic breakage. Respect the
> project's stated pragmatic, anti-over-engineering philosophy (CLAUDE.md): do
> NOT flag long functions, many-parameter signatures, or "missing" abstraction
> layers — those are explicitly acceptable. Only report REAL issues with
> concrete `file:line` evidence. When in doubt, lean toward NOT flagging and
> mark it low severity. Read the relevant guideline doc FIRST so you check
> against the project's actual rules, not generic opinions.
>
> Write your findings to `.ai/health-audit-<today>/<slug>.md` (create nothing
> else). Format: a top `# <Area> — <date>` heading, then findings as a bullet
> list where each is `**[high|med|low]**` + one-line description + `file:line`
> references + the concrete mismatch. Group trivially-clean sub-checks into a
> single "Clean:" bullet. End with a one-sentence **Verdict**. Return to me
> only a 3–6 line digest: the count by severity and your highest-severity
> findings — not the full file.

Then the per-area body:

**`docs`** — Check manpages in `docs/*.md` against the code:
1. Every command in `cmd/*.go`: do its flags (long AND short) match
   `docs/git-flow-<cmd>.1.md`? Find flags in code missing from docs, and
   documented flags that no longer exist.
2. Config keys: every `gitflow.*` key read/written in `internal/config` and
   `cmd/` — documented in `docs/gitflow-config.5.md` (or a per-command
   manpage)? Any documented key the code no longer uses?
3. Commands with no manpage at all, or manpages for removed commands, and
   broken `git-flow-*(1)` cross-references.
4. Recently-added commands (publish, rename, track, checkout) — verify doc
   coverage (CLAUDE.md makes doc updates MANDATORY for command changes).
5. `version/version.go` vs `cmd/version.go` — do the version constants match?
   Skip stylistic/wording nitpicks.

**`skills`** — Check `.claude/skills/` (and any `.claude/commands/`):
1. STALE REFERENCES: grep each skill for referenced files, paths, skill names,
   and doc names; verify each target exists. Report dangling references.
2. OVERLAP: genuinely adjacent skills (e.g. the review family, `gh-issue` vs
   any global `create-github-issue`, `scan-repo` vs `sync-repo-status`) — is
   the boundary documented, or ambiguous? Don't flag skills that legitimately
   differ.
3. CONSISTENCY: any skill instructing something that contradicts
   `GITHUB_GUIDELINES.md`, `COMMIT_GUIDELINES.md`, `ISSUE_GUIDELINES.md`, or
   `DEV_WORKFLOW.md`.
4. Skills referenced in `DEV_WORKFLOW.md`/`CLAUDE.md` that don't exist, and
   skills that exist but are referenced nowhere (orphans / undocumented
   relative to peers).

**`duplication`** — Read `CODING_GUIDELINES.md` first, then audit `cmd/` and
`internal/` for logic that has diverged or is about to:
1. TRUE DUPLICATION: near-identical logic blocks in 2+ places that could share
   a helper (flag parsing, config-precedence resolution, conflict/merge-state
   handling, validation, hook-context construction). Give every copy's
   `file:line`.
2. PARALLEL IMPLEMENTATIONS: two functions doing the same job under different
   names, or a newer path that superseded an older one but left it in place.
   Respect the anti-over-engineering philosophy: some repetition is fine. Call
   it out only when the copies have already diverged, or are likely to, causing
   real bugs. A near-duplicate that would only save a few lines is NOT a
   finding.

**`dead-code`** — Audit `cmd/`, `internal/`, and `testutil/` for code nothing
reaches:
1. Exported symbols with no non-test caller; unexported symbols with no caller
   at all; unused files.
2. Unreachable branches and orphan helpers (including dead or duplicated
   `testutil` helpers).
   Before flagging, confirm there is truly no caller across `cmd/`, `internal/`,
   and tests — registration/reflection patterns (e.g. Cobra command
   registration, `init()` side effects) can hide callers. When unsure whether a
   symbol is intentional public API, mark it low severity.

**`guidelines`** — Read `CODING_GUIDELINES.md` + CLAUDE.md "Code Conventions"
first; check compliance with the project's stated MUST/ALWAYS/NEVER rules:
1. ERROR HANDLING: custom typed errors from `internal/errors` used; specific
   exit codes per condition; typed errors actually satisfy the `errors.Error`
   interface. Find bare `fmt.Errorf`/`errors.New` where a typed error is
   expected, and exit codes that diverge from the documented set.
2. GIT INTEGRATION & LAYERING: git through `internal/git`; find
   `exec.Command("git", ...)` calls that bypass the wrapper (note if in `cmd/`
   vs read-only `internal/` — the former is worse); graceful conflict handling;
   **uncommitted-changes check before mutating operations**.
3. COMMAND IMPLEMENTATION: consistent Cobra structure; the "validate git-flow
   is initialized" preamble on every command except `init`; long AND short
   flags; input validation; examples/usage present.
4. CONFIG PRECEDENCE: three-layer resolution (branch-type def → command config
   → CLI flags, flags always win) implemented consistently — does any command
   resolve differently?
5. Any other explicit rule in `CODING_GUIDELINES.md`, checked against code.

**`correctness`** — The one area that steps beyond drift into real defects.
Stay disciplined: report a bug ONLY with a concrete failure scenario (specific
inputs/state → wrong result or crash). Respect the pragmatic philosophy — do
NOT flag style, long functions, or theoretical "could be cleaner" issues.
Audit `cmd/` and `internal/`:
1. UNHANDLED ERRORS: returned errors ignored or swallowed; `err` checked but
   the wrong branch taken; deferred `Close`/cleanup errors dropped where the
   result matters.
2. EXIT CODES / ERROR TYPES: a condition that returns the wrong exit code, or a
   failure path that returns success (nil error) and lets execution continue.
3. STATE & BOUNDARY: merge-state read/written inconsistently across
   continue/abort; off-by-one, empty-slice, or nil-map access; branch-name /
   prefix edge cases; ordering assumptions that don't hold.
4. GIT-OP CORRECTNESS: operations run in the wrong order, against the wrong
   ref, or without a required precondition (fetch, uncommitted-changes check)
   such that work could be lost or a branch left in a bad state.
   Report each as `file:line` + the trigger + the wrong outcome. Grade severity
   by blast radius: data loss or branch corruption = high; a bad message = low.

**`tests`** — Read `TESTING_GUIDELINES.md` first, then:
1. Run `go build ./...` and `go test ./...` (allow several minutes;
   `test/cmd` alone can take ~6 min). Report the exact result — pass, or the
   failing package + short error. State what you ran.
2. COVERAGE: each `cmd/*.go` command has a `test/cmd/*_test.go`? List missing
   or obviously thin ones (trivial commands like `version` are acceptable).
3. CONVENTIONS: init repos with `git flow init --defaults`; use `testutil`
   helpers (`SetupTestRepo`, `RunGitFlow`) rather than rolling their own;
   "one test case per function" for integration scenarios (table-driven is
   allowed only for pure functions); error-checked `os.Chdir` restore.
4. Flaky/skipped: `t.Skip` (legit platform guards vs real skips), timing/
   network/ordering dependence, TODO stubs.
5. Dead or duplicated `testutil` helpers.

### 3. Write SUMMARY.md (full runs only)

When all seven areas ran, after the subagents finish, read the seven area files
and write `.ai/health-audit-<today>/SUMMARY.md`:

```markdown
# Repo Health Audit — <date>

<2–3 sentences: overall state, is the build/tests green, how much is drift vs real bugs.>

## Top priorities
<The handful of high-severity items across all areas, each one line with file:line and which area file has detail.>

## By area
| Area | high | med | low | Verdict |
|------|------|-----|-----|---------|
| docs | .. | .. | .. | <one line> |
| ... |

## Suggested next actions
<Group into: ship-before-release fixes, quick doc syncs, opportunistic consolidation, housekeeping. Each item → the pipeline that handles it (/gh-issue, direct edit, etc.).>
```

On a single-area run, skip `SUMMARY.md`.

### 4. Report to user

In the conversation: state the output dir, give the severity counts per area
(or the one area), and list the top priorities in fix order. Do NOT dump the
full reports — point to the files. Close by offering the obvious next step
(file the high-severity items as issues, or fix the cheap doc drift directly).
