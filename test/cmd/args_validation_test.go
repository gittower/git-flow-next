package cmd_test

import (
	"strings"
	"testing"

	"github.com/gittower/git-flow-next/test/testutil"
)

func assertCommandRejectsExtraArgs(t *testing.T, expected string, args ...string) {
	t.Helper()

	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	output, err := testutil.RunGitFlow(t, dir, args...)
	if err == nil {
		t.Fatalf("Expected command %q to reject extra arguments", strings.Join(args, " "))
	}
	if !strings.Contains(output, expected) {
		t.Fatalf("Expected output to contain %q, got: %s", expected, output)
	}
}

// TestShorthandUpdateRejectsExtraArgs verifies update accepts at most one base branch.
// Steps:
// 1. Runs the top-level update command with two positional arguments
// 2. Verifies Cobra rejects the second argument before command execution
func TestShorthandUpdateRejectsExtraArgs(t *testing.T) {
	t.Parallel()
	assertCommandRejectsExtraArgs(t, "accepts at most 1 arg(s), received 2", "update", "main", "develop")
}

// TestShorthandRebaseRejectsExtraArgs verifies rebase accepts at most one base branch.
// Steps:
// 1. Runs the top-level rebase command with two positional arguments
// 2. Verifies Cobra rejects the second argument before command execution
func TestShorthandRebaseRejectsExtraArgs(t *testing.T) {
	t.Parallel()
	assertCommandRejectsExtraArgs(t, "accepts at most 1 arg(s), received 2", "rebase", "main", "develop")
}

// TestShorthandPublishRejectsExtraArgs verifies publish accepts no positional arguments.
// Steps:
// 1. Runs the top-level publish command with an unexpected positional argument
// 2. Verifies Cobra rejects the argument before command execution
func TestShorthandPublishRejectsExtraArgs(t *testing.T) {
	t.Parallel()
	assertCommandRejectsExtraArgs(t, `unknown command "extra"`, "publish", "extra")
}

// TestShorthandFinishRejectsExtraArgs verifies finish accepts no positional arguments.
// Steps:
// 1. Runs the top-level finish command with an unexpected positional argument
// 2. Verifies Cobra rejects the argument before command execution
func TestShorthandFinishRejectsExtraArgs(t *testing.T) {
	t.Parallel()
	assertCommandRejectsExtraArgs(t, `unknown command "extra"`, "finish", "extra")
}

// TestInitRejectsExtraArgs verifies init accepts no positional arguments.
// Steps:
// 1. Runs the init command with an unexpected positional argument
// 2. Verifies Cobra rejects the argument before initialization
func TestInitRejectsExtraArgs(t *testing.T) {
	t.Parallel()
	assertCommandRejectsExtraArgs(t, `unknown command "extra"`, "init", "extra")
}

// TestOverviewRejectsExtraArgs verifies overview accepts no positional arguments.
// Steps:
// 1. Runs the overview command with an unexpected positional argument
// 2. Verifies Cobra rejects the argument before generating output
func TestOverviewRejectsExtraArgs(t *testing.T) {
	t.Parallel()
	assertCommandRejectsExtraArgs(t, `unknown command "extra"`, "overview", "extra")
}

// TestVersionRejectsExtraArgs verifies version accepts no positional arguments.
// Steps:
// 1. Runs the version command with an unexpected positional argument
// 2. Verifies Cobra rejects the argument before printing version information
func TestVersionRejectsExtraArgs(t *testing.T) {
	t.Parallel()
	assertCommandRejectsExtraArgs(t, `unknown command "extra"`, "version", "extra")
}

// TestConfigListRejectsExtraArgs verifies config list accepts no positional arguments.
// Steps:
// 1. Runs the config list command with an unexpected positional argument
// 2. Verifies Cobra rejects the argument before reading configuration
func TestConfigListRejectsExtraArgs(t *testing.T) {
	t.Parallel()
	assertCommandRejectsExtraArgs(t, `unknown command "extra"`, "config", "list", "extra")
}
