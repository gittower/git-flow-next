---
name: commit
description: Create a commit following COMMIT_GUIDELINES.md
argument-hint: [optional message hint]
allowed-tools: Bash, Read, Grep
---

# Commit Changes

Create a well-formatted commit following the project's COMMIT_GUIDELINES.md.

## Instructions

1. **Read COMMIT_GUIDELINES.md** for all formatting rules, commit types, and conventions

2. **Check Status**
   - Run `git status` to see staged and unstaged changes
   - Run `git diff --cached` to see what will be committed
   - If nothing staged, ask user what to stage or stage all with `git add -A`

3. **Analyze Changes**
   - Review the diff to understand what changed
   - Identify the appropriate commit type and scope per COMMIT_GUIDELINES.md
   - Choose the type based on **what the change does**, not what file extension it has — use the decision guide below

4. **Write Commit Message** per COMMIT_GUIDELINES.md rules

5. **Create Commit** using HEREDOC format for proper formatting:
   ```bash
   git commit -m "$(cat <<'EOF'
   <type>(<scope>): <subject>

   <body>

   <footer>
   EOF
   )"
   ```

6. **Verify** — run `git log -1` to check the message

## Commit Type Decision Guide

Choose the type based on the **purpose** of the change, not the file type:

| Type | Use when... | Example |
|------|------------|---------|
| **feat** | Adding new user-facing functionality | New CLI command, new flag, new config option |
| **fix** | Fixing a user-facing bug | Incorrect merge behavior, wrong exit code |
| **refactor** | Restructuring code without behavior change | Extract helper, rename internal function |
| **test** | Adding/fixing tests (even if fixing a bug *in* a test) | New test case, fix flaky test |
| **docs** | Changing **user-facing documentation** content | Update README, manpage, ARCHITECTURE.md |
| **ci** | Changing CI/CD, automation, review tooling, GitHub config | Workflows, Copilot/Claude instructions, skills, `.github/` config |
| **build** | Changing build system or dependencies | go.mod, build scripts, Makefile |
| **style** | Code formatting only, no logic change | `go fmt`, whitespace fixes |
| **chore** | Maintenance that doesn't fit above | Update .gitignore, tool config |

### Common pitfalls

- **`.md` files are not always `docs`** — a Markdown file that configures tool behavior (e.g., Copilot instructions, Claude skills, review criteria) is `ci`, not `docs`
- **`fix` is for user-facing bugs only** — fixing a CI workflow is `ci`, fixing a test is `test`, fixing a build script is `build`
- **`test` covers test fixes too** — use `test:` not `fix:` when correcting test code

## Hard Wrapping Technique

When a guideline specifies a hard line wrap limit, write the text as continuous flowing prose first, then hard wrap at the specified character limit, filling each line as close to the limit as possible.

## Example

```bash
git commit -m "$(cat <<'EOF'
feat(finish): Add squash merge strategy

Implements --squash flag for finish command that performs a squash merge
instead of a regular merge, creating a single commit with all branch
changes on the target branch.

- Add squash option to merge strategy enum
- Update finish command to handle squash flag
- Add configuration support for default squash behavior

Resolves #42
EOF
)"
```
