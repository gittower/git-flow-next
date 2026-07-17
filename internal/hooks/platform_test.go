package hooks

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"
)

// setWindows overrides the isWindows platform seam for the duration of a test
// and returns a function that restores the previous value. This lets tests
// exercise the Windows-specific branches on any host (CI runs Linux only).
func setWindows(v bool) func() {
	old := isWindows
	isWindows = v
	return func() { isWindows = old }
}

// TestScriptCommandWindows verifies that on Windows scripts are routed through
// "sh" with the script path as the first argument. Assertions are on cmd.Args
// (the logical argv), not cmd.Path, so the test does not require sh on PATH.
func TestScriptCommandWindows(t *testing.T) {
	defer setWindows(true)()

	cmd := scriptCommand("/hooks/foo.sh", "1.0.0", "message")

	want := []string{"sh", "/hooks/foo.sh", "1.0.0", "message"}
	if !reflect.DeepEqual(cmd.Args, want) {
		t.Errorf("scriptCommand args = %v, want %v", cmd.Args, want)
	}
}

// TestScriptCommandUnix verifies that on Unix the script is invoked directly.
func TestScriptCommandUnix(t *testing.T) {
	defer setWindows(false)()

	cmd := scriptCommand("/hooks/foo.sh", "1.0.0", "message")

	want := []string{"/hooks/foo.sh", "1.0.0", "message"}
	if !reflect.DeepEqual(cmd.Args, want) {
		t.Errorf("scriptCommand args = %v, want %v", cmd.Args, want)
	}
}

// TestIsExecutableFileInfoWindows verifies that on Windows any non-directory
// file is treated as executable (NTFS has no Unix permission bits), while
// directories are not.
func TestIsExecutableFileInfoWindows(t *testing.T) {
	defer setWindows(true)()

	dir := t.TempDir()
	file := filepath.Join(dir, "hook")
	if err := os.WriteFile(file, []byte("#!/bin/sh\n"), 0644); err != nil {
		t.Fatal(err)
	}

	fileInfo, err := os.Stat(file)
	if err != nil {
		t.Fatal(err)
	}
	if !isExecutableFileInfo(fileInfo) {
		t.Error("non-executable (0644) file should be executable on Windows")
	}

	dirInfo, err := os.Stat(dir)
	if err != nil {
		t.Fatal(err)
	}
	if isExecutableFileInfo(dirInfo) {
		t.Error("directory should never be executable on Windows")
	}
}

// TestIsExecutableFileInfoUnix verifies that on Unix executability is decided
// by the file mode's execute bits.
func TestIsExecutableFileInfoUnix(t *testing.T) {
	defer setWindows(false)()

	dir := t.TempDir()
	nonExec := filepath.Join(dir, "plain")
	exec := filepath.Join(dir, "runnable")
	if err := os.WriteFile(nonExec, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(exec, []byte("x"), 0755); err != nil {
		t.Fatal(err)
	}

	nonExecInfo, err := os.Stat(nonExec)
	if err != nil {
		t.Fatal(err)
	}
	if isExecutableFileInfo(nonExecInfo) {
		t.Error("0644 file should not be executable on Unix")
	}

	execInfo, err := os.Stat(exec)
	if err != nil {
		t.Fatal(err)
	}
	if !isExecutableFileInfo(execInfo) {
		t.Error("0755 file should be executable on Unix")
	}
}

// TestScriptCommandWindowsRoundTrip runs a non-executable (0644) script through
// the Windows code path on the host to prove the two PR changes work together:
// the file is eligible under Windows executability rules, and routing it through
// "sh" actually executes it. Because the file lacks the execute bit, a direct
// exec would fail with a permission error on Unix — so success here confirms the
// script ran via sh, not by direct execution. This gives real end-to-end
// confidence for the Windows path while running on the Linux CI runner.
func TestScriptCommandWindowsRoundTrip(t *testing.T) {
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("sh not available on PATH")
	}
	defer setWindows(true)()

	dir := t.TempDir()
	script := filepath.Join(dir, "filter")
	// 0644: readable but NOT executable — can only run via `sh <script>`.
	if err := os.WriteFile(script, []byte("#!/bin/sh\necho \"filtered-$1\"\n"), 0644); err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(script)
	if err != nil {
		t.Fatal(err)
	}
	if !isExecutableFileInfo(info) {
		t.Fatal("script should be considered executable under Windows rules")
	}

	out, err := scriptCommand(script, "1.2.3").Output()
	if err != nil {
		t.Fatalf("running non-executable script via sh failed: %v", err)
	}
	if got, want := string(out), "filtered-1.2.3\n"; got != want {
		t.Errorf("script output = %q, want %q", got, want)
	}

	// Negative control: on the Unix path the same 0644 file cannot be executed
	// directly, confirming the two branches genuinely differ in behavior.
	restore := setWindows(false)
	defer restore()
	if _, err := scriptCommand(script, "1.2.3").Output(); err == nil {
		t.Error("expected direct execution of a non-executable file to fail on Unix path")
	}
}
