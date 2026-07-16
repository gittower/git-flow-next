<!-- PR SUMMARY FORMAT -->
<!-- First line: One sentence TL;DR - what this PR does and why -->
<!-- Details paragraph: What was added/modified, how it works, key areas touched -->
<!-- GitHub keywords: On their own line (Resolves/Relates/Closes #ISSUE) -->
<!-- Remarks section: Optional - design decisions, scope boundaries, caveats -->
<!-- Review focus: Optional - specific files/lines reviewers should inspect -->
<!-- No AI attribution: Never add lines like "Generated with Claude Code" - AI contributions are credited via Co-Authored-By commit trailers only -->

<!-- EXAMPLE:
Adds `--merge-message` flag to customize merge commit messages during finish operations.

The implementation supports both CLI flags and git config with placeholder expansion (`%b` for branch, `%p` for parent). Messages persist through conflict resolution. Changes touch `cmd/finish.go`, `internal/git/repo.go`, and `internal/mergestate/mergestate.go`.

Resolves #58

## Remarks

- Squash message remains CLI-only since content is branch-specific
- Both upstream and child merges support custom messages

**Review focus:**
- `cmd/finish.go:142-168` - placeholder expansion logic
- `internal/mergestate/mergestate.go` - new persistence fields
-->

<!-- AUTHOR CHECKLIST — Do not include in final PR, for your own verification only -->
<!-- - [ ] Commit messages follow COMMIT_GUIDELINES.md -->
<!-- - [ ] Tests added/updated and passing (go test ./...) -->
<!-- - [ ] Documentation updated in docs/ for changed commands or options -->
