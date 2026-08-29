package cmd_test

import (
	"strings"
	"testing"

	"github.com/gittower/git-flow-next/test/testutil"
)

// runConfigListRepeatedly runs 'config list' the given number of times and
// returns each run's output. Go randomizes map iteration per run, so a single
// run cannot distinguish a stable order from a lucky one.
func runConfigListRepeatedly(t *testing.T, dir string, runs int) []string {
	t.Helper()

	outputs := make([]string, 0, runs)
	for i := 0; i < runs; i++ {
		output, err := testutil.RunGitFlow(t, dir, "config", "list")
		if err != nil {
			t.Fatalf("Failed to run git-flow config list (run %d): %v\nOutput: %s", i+1, err, output)
		}
		outputs = append(outputs, output)
	}
	return outputs
}

// configListTopicTypeHeaders returns the section headers of the topic branch
// types block, in the order they appear. Headers sit at column 0 and end in a
// colon; the properties below them are indented, which is what separates the
// two. Scanning stops at the horizontal rule that closes the block.
func configListTopicTypeHeaders(output string) []string {
	var headers []string
	inSection := false
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimRight(line, "\r")
		if strings.HasPrefix(line, "Topic branch types:") {
			inSection = true
			continue
		}
		if !inSection {
			continue
		}
		if strings.HasPrefix(line, "─") {
			break
		}
		if line == "" || strings.HasPrefix(line, " ") || strings.HasPrefix(line, "-") {
			continue
		}
		if strings.HasSuffix(line, ":") {
			headers = append(headers, line)
		}
	}
	return headers
}

// configListBaseBranchLines returns the base branch relationship lines of the
// output, in the order they appear. Trunks render as "name → (root)" and child
// base branches as "name → parent", so trunk selects the former and the
// inverse selects the latter.
func configListBaseBranchLines(output string, trunk bool) []string {
	var lines []string
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimRight(line, "\r")
		if !strings.Contains(line, "→") {
			continue
		}
		if strings.HasSuffix(line, "(root)") == trunk {
			lines = append(lines, line)
		}
	}
	return lines
}

// setConfigListKey sets a git config key, failing the test if the git config
// command fails. The assertions are written against a specific configured
// branch set, so a failed write has to surface here rather than as a confusing
// assertion mismatch later.
func setConfigListKey(t *testing.T, dir string, key string, value string) {
	t.Helper()

	if _, err := testutil.RunGit(t, dir, "config", key, value); err != nil {
		t.Fatalf("Failed to set %s: %v", key, err)
	}
}

// TestConfigListTopicTypeOrderIsDeterministic tests that the topic branch type
// sections are listed in a stable, alphabetical order across identical runs.
// Steps:
// 1. Sets up a test repository and initializes git-flow with defaults
// 2. Runs 'config list' five times
// 3. Verifies every run lists the topic type sections sorted by branch type name
func TestConfigListTopicTypeOrderIsDeterministic(t *testing.T) {
	t.Parallel()
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// The default topic types, sorted by branch type name
	expected := []string{"bugfix:", "feature:", "hotfix:", "release:", "support:"}

	for i, out := range runConfigListRepeatedly(t, dir, 5) {
		headers := configListTopicTypeHeaders(out)
		if strings.Join(headers, " ") != strings.Join(expected, " ") {
			t.Errorf("Run %d: expected topic type sections %v, got %v\nOutput: %s", i+1, expected, headers, out)
		}
	}
}

// TestConfigListBaseBranchOrderIsDeterministic tests that the base branch lists
// are stable across identical runs. The default configuration has a single
// trunk and a single child base branch, so a second one of each is configured.
// Steps:
// 1. Sets up a test repository and initializes git-flow with defaults
// 2. Adds a second trunk branch and a second child base branch
// 3. Runs 'config list' five times
// 4. Verifies every run lists both trunks and both children in sorted order
func TestConfigListBaseBranchOrderIsDeterministic(t *testing.T) {
	t.Parallel()
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// A second trunk (no parent), sorting before the default main
	setConfigListKey(t, dir, "gitflow.branch.archive.type", "base")

	// A second child base branch, sorting after the default develop
	setConfigListKey(t, dir, "gitflow.branch.staging.type", "base")
	setConfigListKey(t, dir, "gitflow.branch.staging.parent", "main")

	expectedTrunks := []string{"  archive → (root)", "  main → (root)"}
	expectedChildren := []string{"  develop → main", "  staging → main"}

	for i, out := range runConfigListRepeatedly(t, dir, 5) {
		trunks := configListBaseBranchLines(out, true)
		if strings.Join(trunks, " ") != strings.Join(expectedTrunks, " ") {
			t.Errorf("Run %d: expected trunk branches %v, got %v\nOutput: %s", i+1, expectedTrunks, trunks, out)
		}

		children := configListBaseBranchLines(out, false)
		if strings.Join(children, " ") != strings.Join(expectedChildren, " ") {
			t.Errorf("Run %d: expected child base branches %v, got %v\nOutput: %s", i+1, expectedChildren, children, out)
		}
	}
}

// TestConfigListOmitsActiveTopicBranches tests that a started topic branch is
// not listed as a branch type. Its runtime gitflow.branch.<branch>.base key
// parses into a branch entry with no type, which is state rather than
// configuration.
// Steps:
// 1. Sets up a test repository and initializes git-flow with defaults
// 2. Runs 'git flow feature start plain', which writes the runtime base key
// 3. Runs 'config list'
// 4. Verifies feature/plain is not listed as a topic branch type
// 5. Verifies the five configured topic types are still listed
func TestConfigListOmitsActiveTopicBranches(t *testing.T) {
	t.Parallel()
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Start a topic branch, which records gitflow.branch.feature/plain.base
	output, err = testutil.RunGitFlow(t, dir, "feature", "start", "plain")
	if err != nil {
		t.Fatalf("Failed to start feature branch: %v\nOutput: %s", err, output)
	}

	output, err = testutil.RunGitFlow(t, dir, "config", "list")
	if err != nil {
		t.Fatalf("Failed to run git-flow config list: %v\nOutput: %s", err, output)
	}

	headers := configListTopicTypeHeaders(output)
	for _, header := range headers {
		if header == "feature/plain:" {
			t.Errorf("Expected the active branch feature/plain not to be listed as a topic type, got %v\nOutput: %s", headers, output)
		}
	}

	for _, expected := range []string{"bugfix:", "feature:", "hotfix:", "release:", "support:"} {
		found := false
		for _, header := range headers {
			if header == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected topic type %q to be listed, got %v\nOutput: %s", expected, headers, output)
		}
	}
}
