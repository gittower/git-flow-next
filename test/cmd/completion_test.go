package cmd_test

import (
	"strings"
	"testing"

	"github.com/gittower/git-flow-next/test/testutil"
)

// TestCompletionBash verifies that "completion bash" generates valid output
// including Cobra's standard completion and the _git_flow bridge function.
func TestCompletionBash(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	output, err := testutil.RunGitFlow(t, dir, "completion", "bash")
	if err != nil {
		t.Fatalf("completion bash failed: %v\nOutput: %s", err, output)
	}

	// Cobra's standard completion function
	if !strings.Contains(output, "__start_git-flow") {
		t.Error("expected bash completion to contain __start_git-flow")
	}
	// Standard registration for "git-flow" direct invocation
	if !strings.Contains(output, "complete -o default -F __start_git-flow git-flow") {
		t.Error("expected bash completion to register complete for git-flow")
	}
	// Bridge function for "git flow" subcommand
	if !strings.Contains(output, "_git_flow()") {
		t.Error("expected bash completion to contain _git_flow bridge function")
	}
}

// TestCompletionZsh verifies that "completion zsh" generates valid output
// with the _git-flow function that zsh's _git auto-discovers.
func TestCompletionZsh(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	output, err := testutil.RunGitFlow(t, dir, "completion", "zsh")
	if err != nil {
		t.Fatalf("completion zsh failed: %v\nOutput: %s", err, output)
	}

	if !strings.Contains(output, "_git-flow") {
		t.Error("expected zsh completion to contain _git-flow function")
	}
}

// TestCompletionFish verifies that "completion fish" generates valid output
// including the git subcommand bridge.
func TestCompletionFish(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	output, err := testutil.RunGitFlow(t, dir, "completion", "fish")
	if err != nil {
		t.Fatalf("completion fish failed: %v\nOutput: %s", err, output)
	}

	// Cobra's standard fish completion
	if !strings.Contains(output, "complete -c git-flow") {
		t.Error("expected fish completion to contain registrations for git-flow")
	}
	// Bridge: register "flow" as a git subcommand
	if !strings.Contains(output, "__fish_git_flow_complete") {
		t.Error("expected fish completion to contain __fish_git_flow_complete bridge")
	}
	if !strings.Contains(output, `complete -c git -f -n '__fish_use_subcommand' -a flow`) {
		t.Error("expected fish completion to register 'flow' as a git subcommand")
	}
}

// TestCompletionPowerShell verifies that "completion powershell" generates valid output.
func TestCompletionPowerShell(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	output, err := testutil.RunGitFlow(t, dir, "completion", "powershell")
	if err != nil {
		t.Fatalf("completion powershell failed: %v\nOutput: %s", err, output)
	}

	if !strings.Contains(output, "git-flow") {
		t.Error("expected powershell completion to reference git-flow")
	}
}

// TestCompletionInvalidShell verifies that an invalid shell argument is rejected.
func TestCompletionInvalidShell(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	_, err := testutil.RunGitFlow(t, dir, "completion", "invalid")
	if err == nil {
		t.Error("expected completion with invalid shell to fail")
	}
}

// TestCompletionNoArgs verifies that running completion without arguments fails.
func TestCompletionNoArgs(t *testing.T) {
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	_, err := testutil.RunGitFlow(t, dir, "completion")
	if err == nil {
		t.Error("expected completion without arguments to fail")
	}
}
