package testutil

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// ShellRun describes one execution of an emitted shell-init script.
type ShellRun struct {
	// Shell is bash, zsh or fish.
	Shell string
	// Dir is the working directory the shell starts in. It is set as cmd.Dir;
	// tests never call os.Chdir.
	Dir string
	// Env holds extra KEY=VALUE pairs layered onto the from-scratch environment.
	Env []string
	// Script is the shell source executed with the shell's -c option.
	Script string
	// ShellArgs holds extra options passed to the shell BEFORE -c, for a test
	// that has to run the shell in a different mode (--posix, say).
	ShellArgs []string
	// StubGitFlow, when non-empty, is written into the PATH-prefix bin/ as an
	// executable git-flow INSTEAD of the symlink to the real binary. It drives
	// wrapper paths that a real git-flow cannot produce deterministically, such
	// as a destination that no longer exists.
	StubGitFlow string
}

// ShellResult carries everything a wrapper assertion needs, including the
// directories the run actually used — so no assertion can accidentally inspect a
// directory the shell never saw.
//
// TempDir is returned rather than fetched separately on purpose: t.TempDir()
// allocates a NEW directory on every call within the same test, so a
// free-standing helper would hand back a directory the shell never wrote to and
// every leftover assertion would pass unconditionally.
type ShellResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
	// TempDir is the TMPDIR this run's shell was given.
	TempDir string
	// MktempCalls counts the lines the mktemp shim logged during this run.
	MktempCalls int
}

// mktempShim logs one line per invocation and then execs the real mktemp, so a
// test can prove how many temporary files a wrapper asked for — including the
// zero a delegated `git status` must ask for. The shim removes its own directory
// from PATH before the exec so it never recurses into itself.
const mktempShim = `#!/bin/sh
printf 'mktemp\n' >> "$GIT_FLOW_TEST_MKTEMP_LOG"
PATH="$GIT_FLOW_TEST_REAL_PATH"
export PATH
exec mktemp "$@"
`

// RequireShell fails the test when the shell is not installed. It deliberately
// does NOT skip: a skip would leave the suite green while two of the three
// emitted scripts were never executed, which is the failure mode this whole
// group of tests exists to prevent.
func RequireShell(t *testing.T, shell string) {
	t.Helper()
	if _, err := exec.LookPath(shell); err != nil {
		t.Fatalf("Shell %q is required by the shell-init wrapper tests but is not installed.\n"+
			"Install it and re-run: macOS 'brew install %s', Debian/Ubuntu 'sudo apt-get install -y %s'.",
			shell, shell, shell)
	}
}

// RunShell executes run against the real interpreter and returns its result.
//
// The shell is started non-interactively and without user rc files, in a
// from-scratch environment: only PATH, HOME, TMPDIR and the handful of variables
// below are set, so nothing from the developer's shell or git configuration can
// change the outcome. GIT_FLOW_CD_FILE is deliberately absent unless the caller
// supplies it.
func RunShell(t *testing.T, run ShellRun) ShellResult {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("The emitted shell-init scripts target POSIX shells and fish")
	}
	RequireShell(t, run.Shell)

	tempDir := t.TempDir()
	binDir := filepath.Join(t.TempDir(), "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatalf("Failed to create shell PATH directory: %v", err)
	}

	// Both `git-flow …` and the wrapper's `command git-flow` have to resolve to
	// the binary under test; the real git is found further down the inherited
	// PATH.
	gitFlow := filepath.Join(binDir, "git-flow")
	if run.StubGitFlow != "" {
		if err := os.WriteFile(gitFlow, []byte(run.StubGitFlow), 0755); err != nil {
			t.Fatalf("Failed to write the git-flow stub: %v", err)
		}
	} else if err := os.Symlink(GitFlowPath(), gitFlow); err != nil {
		t.Fatalf("Failed to link the git-flow binary into the shell PATH: %v", err)
	}

	mktempLog := filepath.Join(tempDir, "mktemp.log")
	realPath := os.Getenv("PATH")
	if err := os.WriteFile(filepath.Join(binDir, "mktemp"), []byte(mktempShim), 0755); err != nil {
		t.Fatalf("Failed to write the mktemp shim: %v", err)
	}

	var args []string
	switch run.Shell {
	case "bash":
		args = []string{"--norc", "--noprofile"}
	case "zsh":
		args = []string{"-f"}
	case "fish":
		args = []string{"--no-config"}
	default:
		t.Fatalf("Unsupported shell %q", run.Shell)
	}
	args = append(args, run.ShellArgs...)
	args = append(args, "-c", run.Script)

	cmd := exec.Command(run.Shell, args...)
	cmd.Dir = run.Dir
	cmd.Env = append([]string{
		"PATH=" + binDir + string(os.PathListSeparator) + realPath,
		"HOME=" + t.TempDir(),
		"TMPDIR=" + tempDir,
		"GIT_FLOW_TEST_MKTEMP_LOG=" + mktempLog,
		"GIT_FLOW_TEST_REAL_PATH=" + realPath,
		"GIT_EDITOR=:",
		"LANG=C",
	}, run.Env...)

	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	exitCode := 0
	if err := cmd.Run(); err != nil {
		exitErr, ok := err.(*exec.ExitError)
		if !ok {
			t.Fatalf("Failed to run %s: %v\nStderr: %s", run.Shell, err, stderr.String())
		}
		exitCode = exitErr.ExitCode()
	}

	return ShellResult{
		Stdout:      stdout.String(),
		Stderr:      stderr.String(),
		ExitCode:    exitCode,
		TempDir:     tempDir,
		MktempCalls: countLines(t, mktempLog),
	}
}

// countLines returns the number of non-empty lines in path, treating a missing
// file as zero lines: the shim never runs when the wrapper never calls mktemp,
// which is itself an assertion some tests make.
func countLines(t *testing.T, path string) int {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return 0
		}
		t.Fatalf("Failed to read the mktemp log %s: %v", path, err)
	}
	count := 0
	for _, line := range strings.Split(string(data), "\n") {
		if strings.TrimSpace(line) != "" {
			count++
		}
	}
	return count
}

// ShellTempFileLeftovers returns the navigation temp files the wrapper left
// behind in the TMPDIR of a finished run. The wrapper must remove its file on
// every path, so a non-empty result is always a defect.
//
// It is a plain function rather than an assertion so a test can prove the check
// itself works by pointing it at a directory that deliberately leaks.
func ShellTempFileLeftovers(t *testing.T, tempDir string) []string {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(tempDir, "git-flow-cd.*"))
	if err != nil {
		t.Fatalf("Failed to glob %s for temp files: %v", tempDir, err)
	}
	return matches
}

// CheckShellSyntax parses script with the shell's parse-only mode and returns
// what the shell wrote to stderr plus whether it accepted the script. It catches
// what content assertions cannot: a syntax error in an emitted script.
func CheckShellSyntax(t *testing.T, shell string, script string) (string, bool) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("The emitted shell-init scripts target POSIX shells and fish")
	}
	RequireShell(t, shell)

	dir := t.TempDir()
	path := filepath.Join(dir, "shell-init-script")
	if err := os.WriteFile(path, []byte(script), 0644); err != nil {
		t.Fatalf("Failed to write the emitted script: %v", err)
	}

	flag := "-n"
	if shell == "fish" {
		flag = "--no-execute"
	}
	cmd := exec.Command(shell, flag, path)
	cmd.Dir = dir
	var stderr strings.Builder
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stderr.String(), err == nil
}

// WriteExecutable writes an executable script, for tests that need a stub on
// PATH of their own.
func WriteExecutable(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0755); err != nil {
		t.Fatalf("Failed to write executable %s: %v", path, err)
	}
}

// ShellInstallLine returns the line that installs the emitted wrapper in shell,
// as documented for users: an eval of the command substitution for bash and zsh,
// a pipe into source for fish.
func ShellInstallLine(shell string) string {
	if shell == "fish" {
		return "git-flow shell-init fish | source"
	}
	return fmt.Sprintf("eval \"$(git-flow shell-init %s)\"", shell)
}
