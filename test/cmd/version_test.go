package cmd_test

import (
	"regexp"
	"testing"

	"github.com/gittower/git-flow-next/test/testutil"
	"github.com/gittower/git-flow-next/version"
)

// expectedVersionLine is the exact stdout 'git flow version' must produce.
// It is built from version.Version rather than a literal so the tests stay
// correct across releases, and so they fail loudly if the version value in
// cmd/version.go ever drifts from version/version.go, which CLAUDE.md requires
// to stay in sync.
var expectedVersionLine = version.Version + " (git-flow-next)\n"

// bareSemverPattern matches a version number with no tool-name prefix and no
// leading 'v' - the shape AVH-compatible parsers read from the first token.
var bareSemverPattern = regexp.MustCompile(`^\d+\.\d+\.\d+$`)

// assertVersionOutput fails unless out is exactly the expected version line.
func assertVersionOutput(t *testing.T, out string) {
	t.Helper()

	if out != expectedVersionLine {
		t.Errorf("Expected output %q, got %q", expectedVersionLine, out)
	}
}

// TestVersionOutputFormat verifies 'git flow version' writes the version number
// before the tool name, on stdout, in an initialized repository.
// Steps:
// 1. Sets up a test repository and initializes git-flow with defaults
// 2. Runs 'git flow version' capturing stdout and stderr separately
// 3. Verifies the command exits 0
// 4. Verifies stdout is exactly '<version> (git-flow-next)' with a trailing newline and nothing else
// 5. Verifies stderr is empty
func TestVersionOutputFormat(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	stdout, stderr, err := testutil.RunGitFlowStreams(t, dir, "version")
	if err != nil {
		t.Fatalf("Failed to run version: %v\nStderr: %s", err, stderr)
	}

	assertVersionOutput(t, stdout)
	if stderr != "" {
		t.Errorf("Expected empty stderr, got %q", stderr)
	}
}

// TestVersionNumberIsBareSemver verifies the version number carries no prefix,
// which is the property AVH-compatible parsers rely on when they read the first
// whitespace-separated token of the output.
// Steps:
// 1. Reads the canonical version constant
// 2. Verifies it matches a bare semantic version with no tool name and no leading 'v'
func TestVersionNumberIsBareSemver(t *testing.T) {
	t.Parallel()

	if !bareSemverPattern.MatchString(version.Version) {
		t.Errorf("Expected version %q to be a bare semantic version", version.Version)
	}
}

// TestVersionInUninitializedRepo verifies 'git flow version' prints the same
// line in a repository without git-flow configuration, without printing a
// first-run activation prompt or hint.
// Steps:
// 1. Sets up a test repository without running 'git flow init'
// 2. Runs 'git flow version' non-interactively
// 3. Verifies the command exits 0
// 4. Verifies the output is exactly the expected version line, which proves no prompt or hint was appended
func TestVersionInUninitializedRepo(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	out, err := testutil.RunGitFlow(t, dir, "version")
	if err != nil {
		t.Fatalf("Failed to run version in uninitialized repo: %v\nOutput: %s", err, out)
	}

	assertVersionOutput(t, out)
}

// TestVersionOutsideGitRepository verifies 'git flow version' prints the same
// line outside any Git repository, without printing a Git error.
// Steps:
// 1. Creates a temporary directory that is not a Git repository
// 2. Runs 'git flow version' in it
// 3. Verifies the command exits 0
// 4. Verifies the output is exactly the expected version line, which proves no Git error was appended
func TestVersionOutsideGitRepository(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	out, err := testutil.RunGitFlow(t, dir, "version")
	if err != nil {
		t.Fatalf("Failed to run version outside a repository: %v\nOutput: %s", err, out)
	}

	assertVersionOutput(t, out)
}
