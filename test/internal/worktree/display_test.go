package worktree_test

import (
	"path/filepath"
	"testing"

	"github.com/gittower/git-flow-next/internal/worktree"
)

// TestRelativeDisplayPathIsRelativeToBase covers SC-7: a worktree path is
// rendered relative to the main worktree root, which is what makes the list
// column readable.
// Steps:
// 1. Builds a base and a sibling worktree path
// 2. Calls worktree.RelativeDisplayPath with them
// 3. Verifies the result is the '..'-prefixed relative form
func TestRelativeDisplayPathIsRelativeToBase(t *testing.T) {
	t.Parallel()

	base := filepath.FromSlash("/repo")
	path := filepath.FromSlash("/repo-worktrees/feature/x")

	want := filepath.Join("..", "repo-worktrees", "feature", "x")
	if got := worktree.RelativeDisplayPath(base, path); got != want {
		t.Errorf("Expected %q, got %q", want, got)
	}
}

// TestRelativeDisplayPathFallsBackToAbsolute covers SC-7's fallback: when the two
// paths have no relative form — different Windows volumes in practice — the path
// is returned unchanged rather than empty.
// Steps:
// 1. Passes a RELATIVE base with an absolute path, the portable way to make filepath.Rel fail
// 2. Calls worktree.RelativeDisplayPath with them
// 3. Verifies the original path comes back unchanged
func TestRelativeDisplayPathFallsBackToAbsolute(t *testing.T) {
	t.Parallel()

	base := filepath.FromSlash("relative/base")
	path := filepath.FromSlash("/abs/path/x")

	if got := worktree.RelativeDisplayPath(base, path); got != path {
		t.Errorf("Expected the path to be returned unchanged as %q, got %q", path, got)
	}
}
