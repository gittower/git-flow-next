---
name: pr-summary
description: Generate a PR summary and write to .ai/ pr_summary.md
allowed-tools: Read, Grep, Glob, Write, Bash
---

# Create PR Summary

Generate a pull request summary based on branch changes.

## Instructions

1. **Gather Context**
   - Get current branch name
   - Find the corresponding `.ai/` workflow folder for this branch:
     - For issue branches (e.g., `feature/59-no-verify`): look for `.ai/issue-59-*`
     - For named branches (e.g., `feature/worktree-support`): look for `.ai/feature-worktree-support/` or `.ai/worktree-support/`
     - Use `ls .ai/` and match based on branch name patterns
   - Read these files if they exist (use them to understand the feature goals and implementation approach):
     - `concept.md` - Feature concept and design rationale
     - `analysis.md` - Issue analysis and requirements
     - `plan.md` - Implementation plan with specific changes
   - Get associated issue number from branch name, commits, or analysis.md

2. **Analyze Changes**
   ```bash
   # All commits on this branch
   git log main...HEAD --oneline

   # Full diff
   git diff main...HEAD --stat

   # Changed files
   git diff main...HEAD --name-only
   ```

3. **Generate Summary**

   Read `.github/PULL_REQUEST_TEMPLATE.md` for the format specification and example.

   Use context from the `.ai/` folder files to write a better summary:
   - **concept.md**: Understand the design goals and rationale
   - **analysis.md**: Reference the original issue requirements
   - **plan.md**: Verify all planned changes were implemented

   Write the summary to `.ai/<folder>/pr_summary.md`.

4. **Validate Format**

   Before presenting the create command, run `pr_summary.md` through the
   [posting checklist](../_shared/POSTING_CHECKLIST.md) (mechanical + tone),
   then check it against `.github/PULL_REQUEST_TEMPLATE.md`. A PR body is a
   **GitHub-only artifact** — it never enters git history, so unlike commit
   messages it is not caught by code review, and it posts via `gh` so no hook
   reminder fires. This step is that missing gate. Reject and rewrite the
   summary if any of these fail:

   - **First line** is a single-sentence TL;DR (what + why), not a heading
   - **Details paragraph** follows it, covering what changed and key areas
   - **Issue keyword** on its own line when an issue exists
     (`Resolves`/`Closes`/`Relates #N`)
   - **No leftover template scaffolding** — no `<!-- ... -->` comments,
     author checklist, or example text copied from the template
   - **No AI attribution** anywhere (e.g. "Generated with Claude Code");
     credit is via `Co-Authored-By` commit trailers only
   - Optional `## Remarks` and `**Review focus:**` sections, if present,
     match the template's shape

   If it fails, fix `pr_summary.md` and re-check before continuing. This same
   check applies to any skill that posts a PR body directly (see
   `/resolve-issue`, `/takeover-pr`).

5. **Create PR Command**

   Output the `gh pr create` command:

   ```bash
   gh pr create --title "<title>" --body "$(cat .ai/<folder>/pr_summary.md)"
   ```

6. **Report Completion**
   - Show path to pr_summary.md
   - Show the PR creation command
   - Remind to push branch first if not already pushed

## PR Title Guidelines

Follow the **Pull Request Titles** rules in
[COMMIT_GUIDELINES.md](../../../COMMIT_GUIDELINES.md) — same subject format as
commits: `type(scope): Subject`.

- Use the same type/scope vocabulary as commits (feat, fix, docs, refactor,
  …); pick the dominant type for a multi-purpose PR — don't stack prefixes
- Imperative mood, sentence case, no trailing period ("Add feature" not
  "Added feature")
- Be specific but concise
- Do NOT include the issue number in the title (it will be linked in the PR body)
