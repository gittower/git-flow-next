package worktree

import "path/filepath"

// RelativeDisplayPath renders path for reading, relative to base, falling back to
// path unchanged when no relative form exists — two different Windows volumes are
// the reachable case, and an empty cell would be worse than a long one.
//
// It lives in an importable package rather than beside its caller so the fallback
// can be unit-tested: filepath.Rel cannot be made to fail from an integration
// test on Linux or macOS.
//
// It deliberately does NOT symlink-resolve. Both inputs are expected to come from
// the same `git worktree list --porcelain` output and are therefore already in
// git's resolved form, so resolving again would cost a stat per row and change
// nothing. A caller that ever passes a path from os.Getwd or a config template
// must symlink-resolve it itself first — with filepath.EvalSymlinks, or whatever
// tolerates a path that does not exist yet. git.SamePath is not that tool: it
// resolves internally but only answers whether two paths match, so it cannot
// hand back the resolved form this function needs.
//
// The case-folding half of the same rule is satisfied by the standard library:
// filepath.Rel compares path elements with sameWord, which is strings.EqualFold
// on Windows, so two spellings of one location differing only in case still
// produce a relative path rather than an absolute fallback.
func RelativeDisplayPath(base string, path string) string {
	rel, err := filepath.Rel(base, path)
	if err != nil {
		return path
	}
	return rel
}
