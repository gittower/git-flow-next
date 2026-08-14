package cmd_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gittower/git-flow-next/test/testutil"
)

// The wrapper behaviours below are each written once and run against all three
// emitted scripts, so a failure names both the shell and the behaviour.

// installWrapper returns the line that installs the emitted wrapper in shell,
// exactly as the manpage tells users to write it.
func installWrapper(shell string) string {
	return testutil.ShellInstallLine(shell)
}

// captureStatus returns the statement that records the previous command's exit
// code in a variable. In fish it MUST be the statement immediately after the
// command: any test, if or echo in between replaces $status with its own.
func captureStatus(shell string, name string) string {
	if shell == "fish" {
		return fmt.Sprintf("set -l %s $status", name)
	}
	return fmt.Sprintf("%s=$?", name)
}

// lastLine returns the last non-empty line of output, which is where every
// wrapper script below leaves its final 'pwd -P'.
func lastLine(output string) string {
	lines := strings.Split(strings.TrimRight(output, "\n"), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		if strings.TrimSpace(lines[i]) != "" {
			return strings.TrimSpace(lines[i])
		}
	}
	return ""
}

// assertWrapperTempFiles asserts both halves of the cleanup contract: the
// wrapper left no navigation temp file behind, and it asked for exactly the
// expected number of temporary files.
//
// The call count is what makes several assertions non-vacuous — a wrapper that
// creates a file and removes it is indistinguishable from one that never
// creates it by the leftover check alone, and a wrapper reusing one fixed file
// passes the leftover check while handing stale destinations to later commands.
func assertWrapperTempFiles(t *testing.T, res testutil.ShellResult, wantMktempCalls int) {
	t.Helper()
	if leftovers := testutil.ShellTempFileLeftovers(t, res.TempDir); len(leftovers) > 0 {
		t.Errorf("Expected no navigation temp files left behind, got %v", leftovers)
	}
	if res.MktempCalls != wantMktempCalls {
		t.Errorf("Expected %d mktemp call(s), got %d", wantMktempCalls, res.MktempCalls)
	}
}

// wrapperRepoWithWorktree sets up a repository with feature/x and a worktree for
// it, the starting state most wrapper behaviours need. It returns the repository
// directory and the resolved worktree path.
func wrapperRepoWithWorktree(t *testing.T) (string, string) {
	t.Helper()
	dir := initWorktreeRepo(t)
	t.Cleanup(func() { os.RemoveAll(worktreeRootFor(dir)) })
	t.Cleanup(func() { testutil.CleanupTestRepo(t, dir) })
	createFreeBranch(t, dir, "feature/x")
	return dir, addWorktree(t, dir, "feature/x")
}

// ---------------------------------------------------------------------------
// W1 — navigation
// ---------------------------------------------------------------------------

// assertWrapperNavigatesToWorktree runs a wrapped checkout and verifies the
// shell itself ended up in the worktree.
func assertWrapperNavigatesToWorktree(t *testing.T, shell string) {
	t.Helper()
	dir, wtPath := wrapperRepoWithWorktree(t)

	res := testutil.RunShell(t, testutil.ShellRun{
		Shell:  shell,
		Dir:    dir,
		Script: installWrapper(shell) + "; git-flow feature checkout x; pwd -P",
	})

	if res.ExitCode != 0 {
		t.Fatalf("Expected exit code 0, got %d\nStderr: %s", res.ExitCode, res.Stderr)
	}
	if got := lastLine(res.Stdout); got != wtPath {
		t.Errorf("Expected the shell to end up in %q, got %q", wtPath, got)
	}
	assertWrapperTempFiles(t, res, 1)
}

// TestShellInitBashWrapperNavigatesToWorktree covers scenario 15 for bash.
// Steps:
// 1. Sets up a repository with feature/x and a worktree for it
// 2. Installs the emitted bash wrapper and runs 'git-flow feature checkout x'
// 3. Verifies the shell's own working directory is now the worktree
// 4. Verifies exactly one temporary file was created and none was left behind
func TestShellInitBashWrapperNavigatesToWorktree(t *testing.T) {
	t.Parallel()
	assertWrapperNavigatesToWorktree(t, "bash")
}

// TestShellInitZshWrapperNavigatesToWorktree covers scenario 15 for zsh.
// Steps:
// 1. Sets up a repository with feature/x and a worktree for it
// 2. Installs the emitted zsh wrapper and runs 'git-flow feature checkout x'
// 3. Verifies the shell's own working directory is now the worktree
// 4. Verifies exactly one temporary file was created and none was left behind
func TestShellInitZshWrapperNavigatesToWorktree(t *testing.T) {
	t.Parallel()
	assertWrapperNavigatesToWorktree(t, "zsh")
}

// TestShellInitFishWrapperNavigatesToWorktree covers scenario 15 for fish.
// Steps:
// 1. Sets up a repository with feature/x and a worktree for it
// 2. Installs the emitted fish wrapper and runs 'git-flow feature checkout x'
// 3. Verifies the shell's own working directory is now the worktree
// 4. Verifies exactly one temporary file was created and none was left behind
func TestShellInitFishWrapperNavigatesToWorktree(t *testing.T) {
	t.Parallel()
	assertWrapperNavigatesToWorktree(t, "fish")
}

// ---------------------------------------------------------------------------
// W2 — the message survives the navigation
// ---------------------------------------------------------------------------

// assertWrapperKeepsMessageAndNavigates verifies git-flow's own output reaches
// the terminal in its own order and the shell still moves.
func assertWrapperKeepsMessageAndNavigates(t *testing.T, shell string) {
	t.Helper()
	dir, wtPath := wrapperRepoWithWorktree(t)

	echoPWD := `echo "PWD=$(pwd -P)"`
	if shell == "fish" {
		echoPWD = `echo "PWD="(pwd -P)`
	}
	res := testutil.RunShell(t, testutil.ShellRun{
		Shell:  shell,
		Dir:    dir,
		Script: installWrapper(shell) + "; git-flow feature checkout x; " + echoPWD,
	})

	if res.ExitCode != 0 {
		t.Fatalf("Expected exit code 0, got %d\nStderr: %s", res.ExitCode, res.Stderr)
	}
	worktreeLine := strings.Index(res.Stdout, "Worktree for branch 'feature/x' at "+wtPath)
	cdLine := strings.Index(res.Stdout, "To switch to it: cd "+wtPath)
	pwdLine := strings.Index(res.Stdout, "PWD="+wtPath)
	if worktreeLine < 0 || cdLine < 0 || pwdLine < 0 {
		t.Fatalf("Expected both messages and the final directory in %q", res.Stdout)
	}
	if !(worktreeLine < cdLine && cdLine < pwdLine) {
		t.Errorf("Expected git-flow's messages in its own order before the shell's, got %q", res.Stdout)
	}
	if strings.Contains(res.Stdout, "Tip:") {
		t.Errorf("Expected no shell-init tip while the wrapper supplies the channel, got %q", res.Stdout)
	}
	assertWrapperTempFiles(t, res, 1)
}

// TestShellInitBashWrapperKeepsMessageAndNavigates covers scenario 14 for bash.
// Steps:
// 1. Sets up a repository with feature/x and a worktree for it
// 2. Installs the emitted bash wrapper, runs the checkout and echoes the resulting directory
// 3. Verifies git-flow's two messages appear in its own order, before the shell's echo
// 4. Verifies the shell moved to the worktree and no shell-init tip was printed
func TestShellInitBashWrapperKeepsMessageAndNavigates(t *testing.T) {
	t.Parallel()
	assertWrapperKeepsMessageAndNavigates(t, "bash")
}

// TestShellInitZshWrapperKeepsMessageAndNavigates covers scenario 14 for zsh.
// Steps:
// 1. Sets up a repository with feature/x and a worktree for it
// 2. Installs the emitted zsh wrapper, runs the checkout and echoes the resulting directory
// 3. Verifies git-flow's two messages appear in its own order, before the shell's echo
// 4. Verifies the shell moved to the worktree and no shell-init tip was printed
func TestShellInitZshWrapperKeepsMessageAndNavigates(t *testing.T) {
	t.Parallel()
	assertWrapperKeepsMessageAndNavigates(t, "zsh")
}

// TestShellInitFishWrapperKeepsMessageAndNavigates covers scenario 14 for fish.
// Steps:
// 1. Sets up a repository with feature/x and a worktree for it
// 2. Installs the emitted fish wrapper, runs the checkout and echoes the resulting directory
// 3. Verifies git-flow's two messages appear in its own order, before the shell's echo
// 4. Verifies the shell moved to the worktree and no shell-init tip was printed
func TestShellInitFishWrapperKeepsMessageAndNavigates(t *testing.T) {
	t.Parallel()
	assertWrapperKeepsMessageAndNavigates(t, "fish")
}

// ---------------------------------------------------------------------------
// W3 — a command that does not navigate leaves the directory alone
// ---------------------------------------------------------------------------

// assertWrapperWithoutNavigationKeepsDirectory primes the channel with a real
// navigation first, so a wrapper reusing one fixed temp file would move the
// shell on the second, non-navigating command.
func assertWrapperWithoutNavigationKeepsDirectory(t *testing.T, shell string) {
	t.Helper()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer os.RemoveAll(worktreeRootFor(dir))
	createFreeBranch(t, dir, "feature/x")
	start := testutil.EvalPath(t, dir)

	res := testutil.RunShell(t, testutil.ShellRun{
		Shell: shell,
		Dir:   dir,
		Script: installWrapper(shell) +
			"; git-flow worktree add feature/x" +
			fmt.Sprintf("; cd %q", start) +
			"; git-flow version; pwd -P",
	})

	if res.ExitCode != 0 {
		t.Fatalf("Expected exit code 0, got %d\nStderr: %s", res.ExitCode, res.Stderr)
	}
	if got := lastLine(res.Stdout); got != start {
		t.Errorf("Expected the shell to stay in %q, got %q", start, got)
	}
	if !strings.Contains(res.Stdout, "git-flow-next") {
		t.Errorf("Expected the version output to reach the terminal, got %q", res.Stdout)
	}
	assertWrapperTempFiles(t, res, 2)
}

// TestShellInitBashWrapperWithoutNavigationKeepsDirectory covers scenario 16 for bash.
// Steps:
// 1. Sets up a repository with feature/x
// 2. Installs the emitted bash wrapper, creates a worktree (which navigates), then returns to the repository
// 3. Runs 'git-flow version', a command that does not navigate
// 4. Verifies the working directory is unchanged, the version output is present and no temp file was left
func TestShellInitBashWrapperWithoutNavigationKeepsDirectory(t *testing.T) {
	t.Parallel()
	assertWrapperWithoutNavigationKeepsDirectory(t, "bash")
}

// TestShellInitZshWrapperWithoutNavigationKeepsDirectory covers scenario 16 for zsh.
// Steps:
// 1. Sets up a repository with feature/x
// 2. Installs the emitted zsh wrapper, creates a worktree (which navigates), then returns to the repository
// 3. Runs 'git-flow version', a command that does not navigate
// 4. Verifies the working directory is unchanged, the version output is present and no temp file was left
func TestShellInitZshWrapperWithoutNavigationKeepsDirectory(t *testing.T) {
	t.Parallel()
	assertWrapperWithoutNavigationKeepsDirectory(t, "zsh")
}

// TestShellInitFishWrapperWithoutNavigationKeepsDirectory covers scenario 16 for fish.
// Steps:
// 1. Sets up a repository with feature/x
// 2. Installs the emitted fish wrapper, creates a worktree (which navigates), then returns to the repository
// 3. Runs 'git-flow version', a command that does not navigate
// 4. Verifies the working directory is unchanged, the version output is present and no temp file was left
func TestShellInitFishWrapperWithoutNavigationKeepsDirectory(t *testing.T) {
	t.Parallel()
	assertWrapperWithoutNavigationKeepsDirectory(t, "fish")
}

// ---------------------------------------------------------------------------
// W4 — a failing command keeps its exit code and cleans up
// ---------------------------------------------------------------------------

// assertWrapperPreservesFailureExitCode uses TWO runs, because no single script
// can carry both halves: under a caller's `set -e` a wrapper that correctly
// returns git-flow's nonzero status terminates the script AT the call, so
// nothing after it runs (verified on bash 3.2.57 and zsh 5.9).
//
// Run A, without errexit, carries the observational assertions. Run B, with
// errexit, asserts the exit code and — the point of the run — that the wrapper
// still removed its temp file: the `command; status=$?; rm` shape aborts before
// the rm and leaks one, while the `|| __gf_status=$?` shape does not.
func assertWrapperPreservesFailureExitCode(t *testing.T, shell string) {
	t.Helper()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	start := testutil.EvalPath(t, dir)

	runA := testutil.RunShell(t, testutil.ShellRun{
		Shell: shell,
		Dir:   dir,
		Script: installWrapper(shell) +
			"; git-flow feature checkout nonexistent; " + captureStatus(shell, "rc") +
			"; echo RC=$rc; pwd -P; exit $rc",
	})

	if runA.ExitCode != 5 {
		t.Errorf("Expected the shell to exit 5, got %d\nStderr: %s", runA.ExitCode, runA.Stderr)
	}
	if !strings.Contains(runA.Stdout, "RC=5") {
		t.Errorf("Expected the wrapper to return git-flow's exit code, got %q", runA.Stdout)
	}
	if got := lastLine(runA.Stdout); got != start {
		t.Errorf("Expected the shell to stay in %q, got %q", start, got)
	}
	if !strings.Contains(runA.Stderr, "Error:") {
		t.Errorf("Expected git-flow's error text on stderr, got %q", runA.Stderr)
	}
	if strings.Contains(runA.Stdout, "Error:") {
		t.Errorf("Expected git-flow's error text to stay off stdout, got %q", runA.Stdout)
	}
	assertWrapperTempFiles(t, runA, 1)

	if shell == "fish" {
		// fish has no errexit, so there is no second run to make.
		return
	}

	runB := testutil.RunShell(t, testutil.ShellRun{
		Shell:  shell,
		Dir:    dir,
		Script: "set -e; " + installWrapper(shell) + "; git-flow feature checkout nonexistent",
	})

	if runB.ExitCode != 5 {
		t.Errorf("Expected the shell to exit 5 under set -e, got %d\nStderr: %s", runB.ExitCode, runB.Stderr)
	}
	assertWrapperTempFiles(t, runB, 1)
}

// TestShellInitBashWrapperPreservesFailureExitCode covers scenario 17 for bash.
// Steps:
// 1. Sets up a repository with git-flow initialized
// 2. Installs the emitted bash wrapper and runs a checkout of a branch that does not exist
// 3. Verifies the wrapper returns exit code 5, the shell exits 5 and the directory is unchanged
// 4. Verifies git-flow's error text went to stderr, not stdout
// 5. Repeats the failing command under 'set -e' and verifies the temp file was still removed
func TestShellInitBashWrapperPreservesFailureExitCode(t *testing.T) {
	t.Parallel()
	assertWrapperPreservesFailureExitCode(t, "bash")
}

// TestShellInitZshWrapperPreservesFailureExitCode covers scenario 17 for zsh.
// Steps:
// 1. Sets up a repository with git-flow initialized
// 2. Installs the emitted zsh wrapper and runs a checkout of a branch that does not exist
// 3. Verifies the wrapper returns exit code 5, the shell exits 5 and the directory is unchanged
// 4. Verifies git-flow's error text went to stderr, not stdout
// 5. Repeats the failing command under 'set -e' and verifies the temp file was still removed
func TestShellInitZshWrapperPreservesFailureExitCode(t *testing.T) {
	t.Parallel()
	assertWrapperPreservesFailureExitCode(t, "zsh")
}

// TestShellInitFishWrapperPreservesFailureExitCode covers scenario 17 for fish.
//
// The status capture is the statement immediately after the command on purpose:
// one statement in between would replace git-flow's status with its own, which
// is the mistake this assertion exists to catch.
// Steps:
// 1. Sets up a repository with git-flow initialized
// 2. Installs the emitted fish wrapper and runs a checkout of a branch that does not exist
// 3. Verifies the wrapper returns exit code 5, the shell exits 5 and the directory is unchanged
// 4. Verifies git-flow's error text went to stderr, not stdout, and no temp file was left
func TestShellInitFishWrapperPreservesFailureExitCode(t *testing.T) {
	t.Parallel()
	assertWrapperPreservesFailureExitCode(t, "fish")
}

// ---------------------------------------------------------------------------
// W5 — the caller's own channel is not clobbered
// ---------------------------------------------------------------------------

// assertWrapperDoesNotClobberCallerChannel seeds GIT_FLOW_CD_FILE before the
// wrapper is installed. Starting from an absent variable would prove too little:
// a wrapper that overwrote it globally and then unset it would pass while
// destroying whatever the caller had set.
//
// The command it runs must be one that WRITES a destination. With a
// non-navigating command the sentinel stays empty even for a wrapper that hands
// the caller's own file straight to the child, and the second assertion would
// prove nothing.
func assertWrapperDoesNotClobberCallerChannel(t *testing.T, shell string) {
	t.Helper()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer os.RemoveAll(worktreeRootFor(dir))
	createFreeBranch(t, dir, "feature/x")

	sentinel := filepath.Join(t.TempDir(), "caller-channel")
	if err := os.WriteFile(sentinel, nil, 0644); err != nil {
		t.Fatalf("Failed to create the sentinel channel file: %v", err)
	}

	echoLeak := `echo "LEAK=[${GIT_FLOW_CD_FILE-unset}]"`
	if shell == "fish" {
		echoLeak = `echo "LEAK=[$GIT_FLOW_CD_FILE]"`
	}
	res := testutil.RunShell(t, testutil.ShellRun{
		Shell:  shell,
		Dir:    dir,
		Env:    []string{"GIT_FLOW_CD_FILE=" + sentinel},
		Script: installWrapper(shell) + "; git-flow worktree add feature/x; " + echoLeak,
	})

	if res.ExitCode != 0 {
		t.Fatalf("Expected exit code 0, got %d\nStderr: %s", res.ExitCode, res.Stderr)
	}
	if want := "LEAK=[" + sentinel + "]"; !strings.Contains(res.Stdout, want) {
		t.Errorf("Expected the caller's channel to be restored as %q, got %q", want, res.Stdout)
	}
	info, err := os.Stat(sentinel)
	if err != nil {
		t.Fatalf("Failed to stat the sentinel channel file: %v", err)
	}
	if info.Size() != 0 {
		t.Errorf("Expected git-flow to write to the wrapper's temp file, not the caller's channel")
	}
	assertWrapperTempFiles(t, res, 1)
}

// TestShellInitBashWrapperDoesNotClobberCallerChannel covers the wrapper contract for bash.
// Steps:
// 1. Creates an empty sentinel file and exports GIT_FLOW_CD_FILE at it before the wrapper is installed
// 2. Installs the emitted bash wrapper and runs 'git-flow worktree add feature/x', a command that writes a destination
// 3. Verifies the caller's value is restored byte-for-byte after the command
// 4. Verifies the sentinel file is still zero-length, so git-flow wrote to the wrapper's temp file rather than the caller's
func TestShellInitBashWrapperDoesNotClobberCallerChannel(t *testing.T) {
	t.Parallel()
	assertWrapperDoesNotClobberCallerChannel(t, "bash")
}

// TestShellInitZshWrapperDoesNotClobberCallerChannel covers the wrapper contract for zsh.
// Steps:
// 1. Creates an empty sentinel file and exports GIT_FLOW_CD_FILE at it before the wrapper is installed
// 2. Installs the emitted zsh wrapper and runs 'git-flow worktree add feature/x', a command that writes a destination
// 3. Verifies the caller's value is restored byte-for-byte after the command
// 4. Verifies the sentinel file is still zero-length, so git-flow wrote to the wrapper's temp file rather than the caller's
func TestShellInitZshWrapperDoesNotClobberCallerChannel(t *testing.T) {
	t.Parallel()
	assertWrapperDoesNotClobberCallerChannel(t, "zsh")
}

// TestShellInitFishWrapperDoesNotClobberCallerChannel covers the wrapper contract for fish.
// Steps:
// 1. Creates an empty sentinel file and exports GIT_FLOW_CD_FILE at it before the wrapper is installed
// 2. Installs the emitted fish wrapper and runs 'git-flow worktree add feature/x', a command that writes a destination
// 3. Verifies the caller's value is restored byte-for-byte after the command
// 4. Verifies the sentinel file is still zero-length, so git-flow wrote to the wrapper's temp file rather than the caller's
func TestShellInitFishWrapperDoesNotClobberCallerChannel(t *testing.T) {
	t.Parallel()
	assertWrapperDoesNotClobberCallerChannel(t, "fish")
}

// ---------------------------------------------------------------------------
// W6 — a destination containing spaces
// ---------------------------------------------------------------------------

// assertWrapperHandlesPathWithSpaces navigates to a worktree whose path contains
// spaces, the classic quoting failure.
func assertWrapperHandlesPathWithSpaces(t *testing.T, shell string) {
	t.Helper()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	// This root is NOT worktreeRootFor(dir): the template below computes a
	// different one, which nothing else would ever remove.
	spacedRoot := filepath.Join(filepath.Dir(testutil.EvalPath(t, dir)), filepath.Base(dir)+" work trees")
	defer os.RemoveAll(spacedRoot)
	if out, err := testutil.RunGit(t, dir, "config", "gitflow.worktreePath", "../{{ repo }} work trees/{{ branch }}"); err != nil {
		t.Fatalf("Failed to set gitflow.worktreePath: %v\nOutput: %s", err, out)
	}
	createFreeBranch(t, dir, "feature/x")

	res := testutil.RunShell(t, testutil.ShellRun{
		Shell:  shell,
		Dir:    dir,
		Script: installWrapper(shell) + "; git-flow feature checkout x --worktree; pwd -P",
	})

	if res.ExitCode != 0 {
		t.Fatalf("Expected exit code 0, got %d\nStderr: %s", res.ExitCode, res.Stderr)
	}
	want := filepath.Join(spacedRoot, "feature", "x")
	if got := lastLine(res.Stdout); got != want {
		t.Errorf("Expected the shell to end up in %q, got %q", want, got)
	}
	assertWrapperTempFiles(t, res, 1)
}

// TestShellInitBashWrapperHandlesPathWithSpaces covers the wrapper contract for bash.
// Steps:
// 1. Sets a gitflow.worktreePath template whose directory names contain spaces
// 2. Installs the emitted bash wrapper and runs 'git-flow feature checkout x --worktree'
// 3. Verifies the shell's working directory is the spaced path, unsplit
// 4. Verifies exactly one temporary file was created and none was left behind
func TestShellInitBashWrapperHandlesPathWithSpaces(t *testing.T) {
	t.Parallel()
	assertWrapperHandlesPathWithSpaces(t, "bash")
}

// TestShellInitZshWrapperHandlesPathWithSpaces covers the wrapper contract for zsh.
// Steps:
// 1. Sets a gitflow.worktreePath template whose directory names contain spaces
// 2. Installs the emitted zsh wrapper and runs 'git-flow feature checkout x --worktree'
// 3. Verifies the shell's working directory is the spaced path, unsplit
// 4. Verifies exactly one temporary file was created and none was left behind
func TestShellInitZshWrapperHandlesPathWithSpaces(t *testing.T) {
	t.Parallel()
	assertWrapperHandlesPathWithSpaces(t, "zsh")
}

// TestShellInitFishWrapperHandlesPathWithSpaces covers the wrapper contract for fish.
// Steps:
// 1. Sets a gitflow.worktreePath template whose directory names contain spaces
// 2. Installs the emitted fish wrapper and runs 'git-flow feature checkout x --worktree'
// 3. Verifies the shell's working directory is the spaced path, unsplit
// 4. Verifies exactly one temporary file was created and none was left behind
func TestShellInitFishWrapperHandlesPathWithSpaces(t *testing.T) {
	t.Parallel()
	assertWrapperHandlesPathWithSpaces(t, "fish")
}

// ---------------------------------------------------------------------------
// W7 — the git wrapper intercepts 'git flow …'
// ---------------------------------------------------------------------------

// assertGitWrapperNavigates is the decisive test for SC-1, and the directory
// assertion is what makes it decisive: 'git flow …' succeeds with NO wrapper
// installed at all, because git dispatches an unknown subcommand to a git-flow
// binary on PATH. Only the shell's own working directory can prove the git()
// function ran — git-flow as a subprocess cannot change it. Do not simplify this
// to an output check.
func assertGitWrapperNavigates(t *testing.T, shell string) {
	t.Helper()
	dir, wtPath := wrapperRepoWithWorktree(t)

	res := testutil.RunShell(t, testutil.ShellRun{
		Shell:  shell,
		Dir:    dir,
		Script: installWrapper(shell) + "; git flow feature checkout x; pwd -P",
	})

	if res.ExitCode != 0 {
		t.Fatalf("Expected exit code 0, got %d\nStderr: %s", res.ExitCode, res.Stderr)
	}
	if got := lastLine(res.Stdout); got != wtPath {
		t.Errorf("Expected 'git flow …' to move the shell to %q, got %q", wtPath, got)
	}
	assertWrapperTempFiles(t, res, 1)
}

// TestShellInitBashWrapperGitFormNavigates covers SC-1 for bash.
// Steps:
// 1. Sets up a repository with feature/x and a worktree for it
// 2. Installs the emitted bash wrapper and runs 'git flow feature checkout x'
// 3. Verifies the shell's own working directory is now the worktree, which only the git function can achieve
// 4. Verifies exactly one temporary file was created and none was left behind
func TestShellInitBashWrapperGitFormNavigates(t *testing.T) {
	t.Parallel()
	assertGitWrapperNavigates(t, "bash")
}

// TestShellInitZshWrapperGitFormNavigates covers SC-1 for zsh.
// Steps:
// 1. Sets up a repository with feature/x and a worktree for it
// 2. Installs the emitted zsh wrapper and runs 'git flow feature checkout x'
// 3. Verifies the shell's own working directory is now the worktree, which only the git function can achieve
// 4. Verifies exactly one temporary file was created and none was left behind
func TestShellInitZshWrapperGitFormNavigates(t *testing.T) {
	t.Parallel()
	assertGitWrapperNavigates(t, "zsh")
}

// TestShellInitFishWrapperGitFormNavigates covers SC-1 for fish.
// Steps:
// 1. Sets up a repository with feature/x and a worktree for it
// 2. Installs the emitted fish wrapper and runs 'git flow feature checkout x'
// 3. Verifies the shell's own working directory is now the worktree, which only the git function can achieve
// 4. Verifies exactly one temporary file was created and none was left behind
func TestShellInitFishWrapperGitFormNavigates(t *testing.T) {
	t.Parallel()
	assertGitWrapperNavigates(t, "fish")
}

// ---------------------------------------------------------------------------
// W8 — ordinary git is delegated verbatim
// ---------------------------------------------------------------------------

// assertGitWrapperDelegatesOtherCommands verifies the delegation path: git's own
// output and exit code, arguments passed verbatim (a pathspec containing a space
// resolves only if they are), and — provable only through the mktemp shim — no
// temporary file for a command that never enters the flow path.
//
// The verbatim-argument probe is a tracked FILE rather than a branch: git
// forbids a space in a ref name outright, so a branch named 'feature/with space'
// cannot be created at all.
func assertGitWrapperDelegatesOtherCommands(t *testing.T, shell string) {
	t.Helper()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	spacedFile := "path with space.txt"
	if err := testutil.WriteFile(t, dir, spacedFile, "tracked\n"); err != nil {
		t.Fatalf("Failed to create the spaced file: %v", err)
	}
	mustRunGit(t, dir, "add", spacedFile)
	mustRunGit(t, dir, "commit", "-m", "Add a path containing spaces")
	start := testutil.EvalPath(t, dir)

	res := testutil.RunShell(t, testutil.ShellRun{
		Shell: shell,
		Dir:   dir,
		Script: installWrapper(shell) +
			"; git --version" +
			"; git rev-parse --abbrev-ref HEAD" +
			`; git ls-files --error-unmatch -- "path with space.txt"` +
			"; git rev-parse --verify no-such-ref-here; echo RC=" + statusRef(shell) +
			"; pwd -P",
	})

	if !strings.Contains(res.Stdout, "git version") {
		t.Errorf("Expected git's own version output, got %q", res.Stdout)
	}
	if !strings.Contains(res.Stdout, testutil.GetCurrentBranch(t, dir)) {
		t.Errorf("Expected git to report the current branch, got %q", res.Stdout)
	}
	if !strings.Contains(res.Stdout, spacedFile) {
		t.Errorf("Expected the space-containing pathspec to resolve, got %q\nStderr: %s", res.Stdout, res.Stderr)
	}
	if !strings.Contains(res.Stdout, "RC=128") {
		t.Errorf("Expected git's own exit code for a bad ref, got %q", res.Stdout)
	}
	if got := lastLine(res.Stdout); got != start {
		t.Errorf("Expected the shell to stay in %q, got %q", start, got)
	}
	assertWrapperTempFiles(t, res, 0)
}

// statusRef returns the shell's expression for the previous command's exit code.
func statusRef(shell string) string {
	if shell == "fish" {
		return "$status"
	}
	return "$?"
}

// mustRunGit runs a git command and fails the test when it does.
func mustRunGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	out, err := testutil.RunGit(t, dir, args...)
	if err != nil {
		t.Fatalf("git %s failed: %v\nOutput: %s", strings.Join(args, " "), err, out)
	}
	return out
}

// TestShellInitBashWrapperDelegatesOtherGitCommands covers SC-1's delegation for bash.
// Steps:
// 1. Sets up a repository with a tracked file literally named 'path with space.txt'
// 2. Installs the emittedbash wrapper and runs four ordinary git commands through it
// 3. Verifies git's version, the current branch and the spaced pathspec all reach the terminal
// 4. Verifies git's own exit code for a bad ref survives the wrapper and the directory is unchanged
// 5. Verifies zero temporary files were created, since the delegation path must not pay for one
func TestShellInitBashWrapperDelegatesOtherGitCommands(t *testing.T) {
	t.Parallel()
	assertGitWrapperDelegatesOtherCommands(t, "bash")
}

// TestShellInitZshWrapperDelegatesOtherGitCommands covers SC-1's delegation for zsh.
// Steps:
// 1. Sets up a repository with a tracked file literally named 'path with space.txt'
// 2. Installs the emittedzsh wrapper and runs four ordinary git commands through it
// 3. Verifies git's version, the current branch and the spaced pathspec all reach the terminal
// 4. Verifies git's own exit code for a bad ref survives the wrapper and the directory is unchanged
// 5. Verifies zero temporary files were created, since the delegation path must not pay for one
func TestShellInitZshWrapperDelegatesOtherGitCommands(t *testing.T) {
	t.Parallel()
	assertGitWrapperDelegatesOtherCommands(t, "zsh")
}

// TestShellInitFishWrapperDelegatesOtherGitCommands covers SC-1's delegation for fish.
// Steps:
// 1. Sets up a repository with a tracked file literally named 'path with space.txt'
// 2. Installs the emittedfish wrapper and runs four ordinary git commands through it
// 3. Verifies git's version, the current branch and the spaced pathspec all reach the terminal
// 4. Verifies git's own exit code for a bad ref survives the wrapper and the directory is unchanged
// 5. Verifies zero temporary files were created, since the delegation path must not pay for one
func TestShellInitFishWrapperDelegatesOtherGitCommands(t *testing.T) {
	t.Parallel()
	assertGitWrapperDelegatesOtherCommands(t, "fish")
}

// ---------------------------------------------------------------------------
// W9 — 'git flow …' preserves a failure exit code
// ---------------------------------------------------------------------------

// assertGitWrapperPreservesFlowFailureExitCode covers the git() route, which
// reaches the shared navigation function differently and returns through its own
// status expression — in fish a second, independent place a delayed capture
// turns 5 into 0.
func assertGitWrapperPreservesFlowFailureExitCode(t *testing.T, shell string) {
	t.Helper()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	res := testutil.RunShell(t, testutil.ShellRun{
		Shell: shell,
		Dir:   dir,
		Script: installWrapper(shell) +
			"; git flow feature checkout nonexistent; " + captureStatus(shell, "rc") +
			"; echo RC=$rc; exit $rc",
	})

	if res.ExitCode != 5 {
		t.Errorf("Expected the shell to exit 5, got %d\nStderr: %s", res.ExitCode, res.Stderr)
	}
	if !strings.Contains(res.Stdout, "RC=5") {
		t.Errorf("Expected 'git flow …' to return git-flow's exit code, got %q", res.Stdout)
	}
	assertWrapperTempFiles(t, res, 1)
}

// TestShellInitBashWrapperGitFormPreservesFailureExitCode covers SC-1 and scenario 17 for bash.
// Steps:
// 1. Sets up a repository with git-flow initialized
// 2. Installs the emitted bash wrapper and runs 'git flow feature checkout nonexistent'
// 3. Verifies the recorded status is 5 and the shell exits 5
// 4. Verifies exactly one temporary file was created and none was left behind
func TestShellInitBashWrapperGitFormPreservesFailureExitCode(t *testing.T) {
	t.Parallel()
	assertGitWrapperPreservesFlowFailureExitCode(t, "bash")
}

// TestShellInitZshWrapperGitFormPreservesFailureExitCode covers SC-1 and scenario 17 for zsh.
// Steps:
// 1. Sets up a repository with git-flow initialized
// 2. Installs the emitted zsh wrapper and runs 'git flow feature checkout nonexistent'
// 3. Verifies the recorded status is 5 and the shell exits 5
// 4. Verifies exactly one temporary file was created and none was left behind
func TestShellInitZshWrapperGitFormPreservesFailureExitCode(t *testing.T) {
	t.Parallel()
	assertGitWrapperPreservesFlowFailureExitCode(t, "zsh")
}

// TestShellInitFishWrapperGitFormPreservesFailureExitCode covers SC-1 and scenario 17 for fish.
// Steps:
// 1. Sets up a repository with git-flow initialized
// 2. Installs the emitted fish wrapper and runs 'git flow feature checkout nonexistent'
// 3. Verifies the recorded status is 5 and the shell exits 5, proving the git function's own return preserved it
// 4. Verifies exactly one temporary file was created and none was left behind
func TestShellInitFishWrapperGitFormPreservesFailureExitCode(t *testing.T) {
	t.Parallel()
	assertGitWrapperPreservesFlowFailureExitCode(t, "fish")
}

// ---------------------------------------------------------------------------
// W10 — a destination that cannot be entered is reported, not masked
// ---------------------------------------------------------------------------

// failedCDStub is a git-flow stand-in that writes a destination which does not
// exist and exits 0. The wrapper is the unit under test here, and a real
// destination that vanishes between the write and the cd cannot be produced
// deterministically. shell-init is delegated to the real binary so the wrapper
// under test is still the emitted one.
func failedCDStub(t *testing.T) string {
	t.Helper()
	return fmt.Sprintf(`#!/bin/sh
if [ "$1" = "shell-init" ]; then
    exec %q "$@"
fi
printf '%%s\n' "/no/such/git-flow/destination" > "$GIT_FLOW_CD_FILE"
exit 0
`, testutil.GitFlowPath())
}

// assertWrapperReportsFailedCDAndKeepsStatus covers SC-13: a failed cd is
// reported on stderr while git-flow's own exit code survives.
func assertWrapperReportsFailedCDAndKeepsStatus(t *testing.T, shell string) {
	t.Helper()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	start := testutil.EvalPath(t, dir)

	res := testutil.RunShell(t, testutil.ShellRun{
		Shell:       shell,
		Dir:         dir,
		StubGitFlow: failedCDStub(t),
		Script: installWrapper(shell) +
			"; git-flow anything; " + captureStatus(shell, "rc") +
			"; echo RC=$rc; pwd -P; exit $rc",
	})

	if res.ExitCode != 0 {
		t.Errorf("Expected git-flow's own exit code 0 to survive a failed cd, got %d\nStderr: %s", res.ExitCode, res.Stderr)
	}
	if !strings.Contains(res.Stdout, "RC=0") {
		t.Errorf("Expected the wrapper not to mask git-flow's status with cd's, got %q", res.Stdout)
	}
	if got := lastLine(res.Stdout); got != start {
		t.Errorf("Expected the shell to stay in %q, got %q", start, got)
	}
	if !strings.Contains(res.Stderr, "/no/such/git-flow/destination") {
		t.Errorf("Expected stderr to name the destination it could not enter, got %q", res.Stderr)
	}
	assertWrapperTempFiles(t, res, 1)
}

// TestShellInitBashWrapperReportsFailedCD covers SC-13 for bash.
// Steps:
// 1. Sets up a repository and a git-flow stub that writes a destination which does not exist
// 2. Installs the emitted bash wrapper and runs a command through it
// 3. Verifies the recorded status and the shell's exit code are both 0, git-flow's own
// 4. Verifies the working directory is unchanged and stderr names the destination
// 5. Verifies no temporary file was left behind
func TestShellInitBashWrapperReportsFailedCD(t *testing.T) {
	t.Parallel()
	assertWrapperReportsFailedCDAndKeepsStatus(t, "bash")
}

// TestShellInitZshWrapperReportsFailedCD covers SC-13 for zsh.
// Steps:
// 1. Sets up a repository and a git-flow stub that writes a destination which does not exist
// 2. Installs the emitted zsh wrapper and runs a command through it
// 3. Verifies the recorded status and the shell's exit code are both 0, git-flow's own
// 4. Verifies the working directory is unchanged and stderr names the destination
// 5. Verifies no temporary file was left behind
func TestShellInitZshWrapperReportsFailedCD(t *testing.T) {
	t.Parallel()
	assertWrapperReportsFailedCDAndKeepsStatus(t, "zsh")
}

// TestShellInitFishWrapperReportsFailedCD covers SC-13 for fish.
// Steps:
// 1. Sets up a repository and a git-flow stub that writes a destination which does not exist
// 2. Installs the emitted fish wrapper and runs a command through it
// 3. Verifies the recorded status and the shell's exit code are both 0, git-flow's own
// 4. Verifies the working directory is unchanged and stderr names the destination
// 5. Verifies no temporary file was left behind
func TestShellInitFishWrapperReportsFailedCD(t *testing.T) {
	t.Parallel()
	assertWrapperReportsFailedCDAndKeepsStatus(t, "fish")
}

// ---------------------------------------------------------------------------
// W11 — a set-but-empty TMPDIR still navigates
// ---------------------------------------------------------------------------

// assertWrapperNavigatesWithEmptyTMPDIR verifies the /tmp fallback fires for a
// TMPDIR that is set to the empty string, not only for one that is unset.
//
// This is a cross-shell parity test. bash and zsh get it from "${TMPDIR:-/tmp}";
// fish had "set -q TMPDIR", which is TRUE for a set-but-empty variable, so it
// took the branch and ran mktemp against "/git-flow-cd.XXXXXX". That fails, the
// wrapper takes its no-temp-file path, and navigation is silently LOST while the
// command itself still runs and still returns the right exit code — a failure no
// exit-code or output assertion elsewhere would catch.
//
// assertWrapperTempFiles is deliberately not used: the fallback puts the temp
// file in the real /tmp, which the run's own TMPDIR-scoped leftover check cannot
// see and which parallel tests make unsafe to glob. The mktemp count is still
// meaningful, and it is what proves the wrapper reached its normal path at all.
func assertWrapperNavigatesWithEmptyTMPDIR(t *testing.T, shell string) {
	t.Helper()
	dir, wtPath := wrapperRepoWithWorktree(t)

	res := testutil.RunShell(t, testutil.ShellRun{
		Shell:  shell,
		Dir:    dir,
		Env:    []string{"TMPDIR="},
		Script: installWrapper(shell) + "; git-flow feature checkout x; pwd -P",
	})

	if res.ExitCode != 0 {
		t.Fatalf("Expected exit code 0, got %d\nStderr: %s", res.ExitCode, res.Stderr)
	}
	if got := lastLine(res.Stdout); got != wtPath {
		t.Errorf("Expected an empty TMPDIR to fall back to /tmp and still navigate to %q, got %q", wtPath, got)
	}
	if strings.Contains(res.Stderr, "mkstemp") || strings.Contains(res.Stderr, "mktemp") {
		t.Errorf("Expected no raw mktemp diagnostic on stderr, got %q", res.Stderr)
	}
	if res.MktempCalls != 1 {
		t.Errorf("Expected 1 mktemp call, got %d", res.MktempCalls)
	}
}

// TestShellInitBashWrapperNavigatesWithEmptyTMPDIR confirms bash already handles
// a set-but-empty TMPDIR, the behaviour fish is held to.
// Steps:
// 1. Sets up a repository with feature/x and a worktree for it
// 2. Installs the emitted bash wrapper with TMPDIR set to the empty string
// 3. Runs 'git-flow feature checkout x' and verifies the shell reached the worktree
// 4. Verifies stderr carries no mktemp diagnostic and exactly one temp file was requested
func TestShellInitBashWrapperNavigatesWithEmptyTMPDIR(t *testing.T) {
	t.Parallel()
	assertWrapperNavigatesWithEmptyTMPDIR(t, "bash")
}

// TestShellInitZshWrapperNavigatesWithEmptyTMPDIR confirms zsh already handles a
// set-but-empty TMPDIR, the behaviour fish is held to.
// Steps:
// 1. Sets up a repository with feature/x and a worktree for it
// 2. Installs the emitted zsh wrapper with TMPDIR set to the empty string
// 3. Runs 'git-flow feature checkout x' and verifies the shell reached the worktree
// 4. Verifies stderr carries no mktemp diagnostic and exactly one temp file was requested
func TestShellInitZshWrapperNavigatesWithEmptyTMPDIR(t *testing.T) {
	t.Parallel()
	assertWrapperNavigatesWithEmptyTMPDIR(t, "zsh")
}

// TestShellInitFishWrapperNavigatesWithEmptyTMPDIR is the regression test for
// fish's 'set -q TMPDIR', which was true for a set-but-empty variable and lost
// navigation.
// Steps:
// 1. Sets up a repository with feature/x and a worktree for it
// 2. Installs the emitted fish wrapper with TMPDIR set to the empty string
// 3. Runs 'git-flow feature checkout x' and verifies the shell reached the worktree
// 4. Verifies stderr carries no mktemp diagnostic and exactly one temp file was requested
func TestShellInitFishWrapperNavigatesWithEmptyTMPDIR(t *testing.T) {
	t.Parallel()
	assertWrapperNavigatesWithEmptyTMPDIR(t, "fish")
}

// ---------------------------------------------------------------------------
// Helper self-check
// ---------------------------------------------------------------------------

// TestShellRunLeakDetectionCatchesLeftovers proves the leftover check can fail.
//
// A leak assertion that has never been seen to fail is not evidence of anything:
// if RunShell and the check looked at different directories, every wrapper test
// would pass no matter how badly the wrapper leaked.
// Steps:
// 1. Runs a throwaway bash script that deliberately creates a git-flow-cd.XXXX file in its TMPDIR
// 2. Reads the leftovers of that same run
// 3. Verifies exactly the deliberately leaked file is reported
func TestShellRunLeakDetectionCatchesLeftovers(t *testing.T) {
	t.Parallel()
	res := testutil.RunShell(t, testutil.ShellRun{
		Shell:  "bash",
		Dir:    t.TempDir(),
		Script: `: > "$TMPDIR/git-flow-cd.LEAKED"`,
	})
	if res.ExitCode != 0 {
		t.Fatalf("Expected the throwaway script to succeed, got %d\nStderr: %s", res.ExitCode, res.Stderr)
	}

	leftovers := testutil.ShellTempFileLeftovers(t, res.TempDir)
	if len(leftovers) != 1 || filepath.Base(leftovers[0]) != "git-flow-cd.LEAKED" {
		t.Errorf("Expected the deliberate leak to be detected, got %v", leftovers)
	}
}
