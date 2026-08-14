package worktree_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/gittower/git-flow-next/test/testutil"
)

// TestMain builds the git-flow binary before running the worktree path tests,
// which initialize git-flow via the binary to obtain a realistic configuration.
// Set GIT_FLOW_PATH to test a specific prebuilt binary instead.
func TestMain(m *testing.M) {
	cleanup, err := testutil.BuildGitFlow()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to build git-flow binary: %v\n", err)
		os.Exit(1)
	}
	code := m.Run()
	cleanup()
	os.Exit(code)
}
