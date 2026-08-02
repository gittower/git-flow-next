package config_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/gittower/git-flow-next/test/testutil"
)

// TestMain builds the git-flow binary before running the config tests that
// initialize git-flow via the binary. Set GIT_FLOW_PATH to test a specific
// prebuilt binary instead.
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
