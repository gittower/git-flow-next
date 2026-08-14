package cmd_test

import (
	"strings"
	"testing"

	"github.com/gittower/git-flow-next/test/testutil"
)

// shellInitScript returns the script 'git flow shell-init <shell>' emits,
// failing the test when the command itself fails.
func shellInitScript(t *testing.T, shell string) string {
	t.Helper()
	stdout, stderr, err := testutil.RunGitFlowStreams(t, t.TempDir(), "shell-init", shell)
	if err != nil {
		t.Fatalf("shell-init %s failed: %v\nStderr: %s", shell, err, stderr)
	}
	if stderr != "" {
		t.Errorf("Expected no output on stderr, got %q", stderr)
	}
	return stdout
}

// assertContainsAll fails for every wanted substring the script is missing.
func assertContainsAll(t *testing.T, script string, wanted []string) {
	t.Helper()
	for _, want := range wanted {
		if !strings.Contains(script, want) {
			t.Errorf("Expected the script to contain %q", want)
		}
	}
}

// assertContainsNone fails for every forbidden substring the script carries.
func assertContainsNone(t *testing.T, script string, forbidden []string) {
	t.Helper()
	for _, bad := range forbidden {
		if strings.Contains(script, bad) {
			t.Errorf("Expected the script NOT to contain %q", bad)
		}
	}
}

// TestShellInitBashScriptContent covers scenario 10: the bash script defines the
// wrapper, supplies the navigation file per invocation, and cleans up.
// Steps:
// 1. Runs 'git flow shell-init bash'
// 2. Verifies the script defines the shared navigation function and both the git and git-flow wrappers
// 3. Verifies it sets GIT_FLOW_CD_FILE, calls git-flow and delegates ordinary git verbatim
// 4. Verifies it creates a temporary file with mktemp, removes it and returns a status
// 5. Verifies it never exports GIT_FLOW_CD_FILE, which would leak into other programs
// 6. Verifies no command substitution wraps the git-flow invocation, which would capture the output the spec requires to stream
func TestShellInitBashScriptContent(t *testing.T) {
	t.Parallel()
	script := shellInitScript(t, "bash")

	assertContainsAll(t, script, []string{
		"__git_flow_nav",
		"git()",
		"git-flow()",
		"GIT_FLOW_CD_FILE=",
		"command git-flow",
		// Asserted with its arguments: "command git" alone is a substring of
		// "command git-flow" and would pass for a script that never delegates.
		`command git "$@"`,
		"mktemp",
		"rm -f",
		"return",
	})
	assertContainsNone(t, script, []string{
		"export GIT_FLOW_CD_FILE",
		"$(command git-flow",
		"$(git-flow",
	})
}

// TestShellInitZshScriptContent covers scenario 11: the zsh script carries the
// same wrapper contract as the bash one.
// Steps:
// 1. Runs 'git flow shell-init zsh'
// 2. Verifies the script defines the shared navigation function and both the git and git-flow wrappers
// 3. Verifies it sets GIT_FLOW_CD_FILE, calls git-flow and delegates ordinary git verbatim
// 4. Verifies it creates a temporary file with mktemp, removes it and returns a status
// 5. Verifies it never exports GIT_FLOW_CD_FILE and never captures git-flow's output in a command substitution
func TestShellInitZshScriptContent(t *testing.T) {
	t.Parallel()
	script := shellInitScript(t, "zsh")

	assertContainsAll(t, script, []string{
		"__git_flow_nav",
		"git()",
		"git-flow()",
		"GIT_FLOW_CD_FILE=",
		"command git-flow",
		`command git "$@"`,
		"mktemp",
		"rm -f",
		"return",
	})
	assertContainsNone(t, script, []string{
		"export GIT_FLOW_CD_FILE",
		"$(command git-flow",
		"$(git-flow",
	})
}

// TestShellInitFishScriptContent covers scenario 11: the fish script carries the
// same wrapper contract in fish syntax.
// Steps:
// 1. Runs 'git flow shell-init fish'
// 2. Verifies the script defines the shared navigation function and both the git and git-flow wrappers
// 3. Verifies it scopes GIT_FLOW_CD_FILE to one command with env, captures $status and delegates ordinary git verbatim
// 4. Verifies it creates a temporary file with mktemp and removes it
// 5. Verifies it never exports the variable with set -x or set -gx, fish's actual export forms
// 6. Verifies no command substitution wraps the git-flow invocation
func TestShellInitFishScriptContent(t *testing.T) {
	t.Parallel()
	script := shellInitScript(t, "fish")

	assertContainsAll(t, script, []string{
		"function __git_flow_nav",
		"function git-flow",
		"function git",
		"env GIT_FLOW_CD_FILE=",
		"$status",
		"mktemp",
		"rm -f",
		"command git $argv",
	})
	// Not asserted: "contains no export". Fish has no export builtin, so that
	// would be true of every fish script ever written, correct or broken.
	assertContainsNone(t, script, []string{
		"set -x GIT_FLOW_CD_FILE",
		"set -gx GIT_FLOW_CD_FILE",
		"(command git-flow",
		"(git-flow ",
	})
}

// TestShellInitUnknownShellFails covers scenario 12: an unsupported shell is
// rejected.
// Steps:
// 1. Runs 'git flow shell-init tcsh'
// 2. Verifies exit code 1, the code main.go returns for every error Execute reports
// 3. Verifies stderr carries Cobra's invalid-argument message naming the command
func TestShellInitUnknownShellFails(t *testing.T) {
	t.Parallel()
	_, stderr, err := testutil.RunGitFlowStreams(t, t.TempDir(), "shell-init", "tcsh")
	if got := worktreeExitCode(err); got != 1 {
		t.Fatalf("Expected exit code 1, got %d\nStderr: %s", got, stderr)
	}
	if !strings.Contains(stderr, `invalid argument "tcsh" for "git-flow shell-init"`) {
		t.Errorf("Expected an invalid-argument message, got %q", stderr)
	}
}

// TestShellInitNoArgumentFails covers scenario 13: the shell argument is
// mandatory, which is why the tip names a <shell> placeholder.
// Steps:
// 1. Runs 'git flow shell-init' with no argument
// 2. Verifies exit code 1
// 3. Verifies stderr carries Cobra's arity message and the usage block
func TestShellInitNoArgumentFails(t *testing.T) {
	t.Parallel()
	_, stderr, err := testutil.RunGitFlowStreams(t, t.TempDir(), "shell-init")
	if got := worktreeExitCode(err); got != 1 {
		t.Fatalf("Expected exit code 1, got %d\nStderr: %s", got, stderr)
	}
	if !strings.Contains(stderr, "accepts 1 arg(s), received 0") {
		t.Errorf("Expected an arity message, got %q", stderr)
	}
	if !strings.Contains(stderr, "Usage:") || !strings.Contains(stderr, "shell-init") {
		t.Errorf("Expected the usage block, got %q", stderr)
	}
}

// TestShellInitBashScriptSyntaxValid checks the emitted bash script parses.
// Steps:
// 1. Runs 'git flow shell-init bash'
// 2. Parses the script with 'bash -n'
// 3. Verifies bash accepts it and writes nothing to stderr
func TestShellInitBashScriptSyntaxValid(t *testing.T) {
	t.Parallel()
	stderr, ok := testutil.CheckShellSyntax(t, "bash", shellInitScript(t, "bash"))
	if !ok || stderr != "" {
		t.Errorf("Expected bash to accept the script, got %q", stderr)
	}
}

// TestShellInitZshScriptSyntaxValid checks the emitted zsh script parses.
// Steps:
// 1. Runs 'git flow shell-init zsh'
// 2. Parses the script with 'zsh -n'
// 3. Verifies zsh accepts it and writes nothing to stderr
func TestShellInitZshScriptSyntaxValid(t *testing.T) {
	t.Parallel()
	stderr, ok := testutil.CheckShellSyntax(t, "zsh", shellInitScript(t, "zsh"))
	if !ok || stderr != "" {
		t.Errorf("Expected zsh to accept the script, got %q", stderr)
	}
}

// TestShellInitFishScriptSyntaxValid checks the emitted fish script parses.
// Steps:
// 1. Runs 'git flow shell-init fish'
// 2. Parses the script with 'fish --no-execute'
// 3. Verifies fish accepts it and writes nothing to stderr
func TestShellInitFishScriptSyntaxValid(t *testing.T) {
	t.Parallel()
	stderr, ok := testutil.CheckShellSyntax(t, "fish", shellInitScript(t, "fish"))
	if !ok || stderr != "" {
		t.Errorf("Expected fish to accept the script, got %q", stderr)
	}
}

// TestShellInitBashScriptSourcesUnderPosixMode covers SC-1's trap: bash in POSIX
// mode cannot define a function named git-flow, and sourcing must survive it.
//
// The git-flow function is expected to be ABSENT here — that is the documented
// degradation. The SOURCED assertion is load-bearing: it proves the shell
// survived the definition rather than dying on it, which is what happens when
// the invalid identifier is evaluated without the POSIX guard.
// Steps:
// 1. Starts bash with --posix, rc files disabled
// 2. Evaluates the output of 'git flow shell-init bash'
// 3. Echoes SOURCED and checks that a git function is installed
// 4. Verifies exit code 0 and that both markers are present
func TestShellInitBashScriptSourcesUnderPosixMode(t *testing.T) {
	t.Parallel()
	res := testutil.RunShell(t, testutil.ShellRun{
		Shell:     "bash",
		Dir:       t.TempDir(),
		ShellArgs: []string{"--posix"},
		// The command TYPE is what is asserted: a bare 'type git' succeeds for
		// the ordinary git executable on PATH, so it would pass for a script
		// that skips the git() function entirely in POSIX mode.
		Script: `eval "$(git flow shell-init bash)"; echo SOURCED; test "$(type -t git)" = function && echo HAVE-GIT-FN`,
	})

	if res.ExitCode != 0 {
		t.Fatalf("Expected exit code 0, got %d\nStderr: %s", res.ExitCode, res.Stderr)
	}
	if !strings.Contains(res.Stdout, "SOURCED") {
		t.Errorf("Expected sourcing to survive POSIX mode, got %q", res.Stdout)
	}
	if !strings.Contains(res.Stdout, "HAVE-GIT-FN") {
		t.Errorf("Expected the git wrapper to be installed in POSIX mode, got %q", res.Stdout)
	}
}
