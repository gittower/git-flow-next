package cmd

import (
	"os"
	"path/filepath"
	"sync"

	"github.com/gittower/git-flow-next/internal/git"
)

var (
	repoOnce   sync.Once
	cachedRepo *git.Repo
	cachedErr  error
)

// openRepo opens a git.Repo handle for the invocation directory — the process
// working directory where git-flow was run. All repo-bound operations run
// against this handle's absolute paths, so command behavior does not depend on
// any later change of working directory.
//
// The result is memoized: git-flow is a single-shot CLI whose invocation
// directory never changes within a process, so the (potentially several)
// callers per invocation share one resolution rather than re-running git
// rev-parse each time.
func openRepo() (*git.Repo, error) {
	repoOnce.Do(func() {
		dir, err := os.Getwd()
		if err != nil {
			cachedErr = err
			return
		}
		cachedRepo, cachedErr = git.Open(dir)
	})
	return cachedRepo, cachedErr
}

// normalizeInvocationPath resolves a possibly-relative path against the
// invocation directory (the process working directory), returning an absolute
// path. An empty path is returned unchanged. This preserves the historical,
// invocation-directory-relative meaning of relative path arguments (e.g. the tag
// message file) even though repo-bound git commands now run with the work tree
// as their working directory.
func normalizeInvocationPath(path string) string {
	if path == "" || filepath.IsAbs(path) {
		return path
	}
	dir, err := os.Getwd()
	if err != nil {
		return path
	}
	return filepath.Join(dir, path)
}
