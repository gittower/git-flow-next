package cmd_test

import (
	"strings"
	"testing"

	"github.com/gittower/git-flow-next/test/testutil"
)

// TestOverviewWithDefaultConfig tests the overview command with default configuration.
// Steps:
// 1. Sets up a test repository and initializes git-flow with defaults
// 2. Runs the overview command
// 3. Verifies the output contains all expected sections:
//   - Base branches section
//   - Topic branch configurations
//   - Active topic branches
//
// 4. Verifies the base branch relationships (develop -> main -> root)
// 5. Verifies the merge strategy information for base branches
func TestOverviewWithDefaultConfig(t *testing.T) {
	t.Parallel()
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Run git-flow overview
	output, err = testutil.RunGitFlow(t, dir, "overview")
	if err != nil {
		t.Fatalf("Failed to run git-flow overview: %v\nOutput: %s", err, output)
	}

	// Check if the output contains the expected sections
	if !strings.Contains(output, "Base branches:") {
		t.Errorf("Expected output to contain 'Base branches:', got: %s", output)
	}

	if !strings.Contains(output, "Topic branch types:") {
		t.Errorf("Expected output to contain 'Topic branch types:', got: %s", output)
	}

	if !strings.Contains(output, "Active topic branches:") {
		t.Errorf("Expected output to contain 'Active topic branches:', got: %s", output)
	}

	// Check if the output contains the base branches with their relationships
	if !strings.Contains(output, "develop → main [auto-update]") {
		t.Errorf("Expected output to contain 'develop → main [auto-update]', got: %s", output)
	}

	if !strings.Contains(output, "main (root)") {
		t.Errorf("Expected output to contain 'main (root)', got: %s", output)
	}

	// Check if the output contains merge strategy descriptions
	if !strings.Contains(output, "Merges into") {
		t.Errorf("Expected output to contain 'Merges into' (merge strategy), got: %s", output)
	}

	// Check if support branch shows 'none' strategy
	if !strings.Contains(output, "none into") || !strings.Contains(output, "none from") {
		t.Errorf("Expected output to contain 'none into' and 'none from' for support branches, got: %s", output)
	}
}

// TestOverviewWithActiveBranches tests the overview command with active branches.
// Steps:
// 1. Sets up a test repository and initializes git-flow
// 2. Creates a feature branch and a release branch
// 3. Runs the overview command
// 4. Verifies the output contains both active branches:
//   - feature/my-feature
//   - release/1.0.0
//
// 5. Verifies the branch types and names are correctly displayed
func TestOverviewWithActiveBranches(t *testing.T) {
	t.Parallel()
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Create a feature branch
	output, err = testutil.RunGitFlow(t, dir, "feature", "start", "my-feature")
	if err != nil {
		t.Fatalf("Failed to create feature branch: %v\nOutput: %s", err, output)
	}

	// Create a release branch
	output, err = testutil.RunGitFlow(t, dir, "release", "start", "1.0.0")
	if err != nil {
		t.Fatalf("Failed to create release branch: %v\nOutput: %s", err, output)
	}

	// Run git-flow overview
	output, err = testutil.RunGitFlow(t, dir, "overview")
	if err != nil {
		t.Fatalf("Failed to run git-flow overview: %v\nOutput: %s", err, output)
	}

	// Check if the output contains the active branches
	if !strings.Contains(output, "feature/my-feature (feature)") {
		t.Errorf("Expected output to contain 'feature/my-feature (feature)', got: %s", output)
	}

	if !strings.Contains(output, "release/1.0.0 (release)") {
		t.Errorf("Expected output to contain 'release/1.0.0 (release)', got: %s", output)
	}
}

// TestOverviewWithCustomConfig tests the overview command with custom configuration.
// Steps:
// 1. Sets up a test repository and initializes git-flow with custom config:
//   - Custom main and develop branch names
//   - Custom prefixes for all branch types
//   - Custom tag prefix
//
// 2. Runs the overview command
// 3. Verifies the output contains custom base branch names and relationships
// 4. Verifies the output contains all custom topic branch prefixes
// 5. Verifies the branch relationships reflect the custom configuration
func TestOverviewWithCustomConfig(t *testing.T) {
	t.Parallel()
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with custom configuration
	output, err := testutil.RunGitFlow(t, dir, "init",
		"--main", "custom-main", // custom main branch name
		"--develop", "custom-dev", // custom develop branch name
		"--feature", "f/", // custom feature prefix
		"--bugfix", "b/", // custom bugfix prefix
		"--release", "r/", // custom release prefix
		"--hotfix", "h/", // custom hotfix prefix
		"--support", "s/", // custom support prefix
		"--tag", "v") // custom tag prefix
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Run git-flow overview
	output, err = testutil.RunGitFlow(t, dir, "overview")
	if err != nil {
		t.Fatalf("Failed to run git-flow overview: %v\nOutput: %s", err, output)
	}

	// Check if the output contains the custom base branches
	if !strings.Contains(output, "custom-dev → custom-main [auto-update]") {
		t.Errorf("Expected output to contain 'custom-dev → custom-main [auto-update]', got: %s", output)
	}

	if !strings.Contains(output, "custom-main (root)") {
		t.Errorf("Expected output to contain 'custom-main (root)', got: %s", output)
	}

	// Check if the output contains the custom topic branch prefixes as section headers
	if !strings.Contains(output, "f/*:") {
		t.Errorf("Expected output to contain 'f/*:' section header, got: %s", output)
	}

	if !strings.Contains(output, "r/*:") {
		t.Errorf("Expected output to contain 'r/*:' section header, got: %s", output)
	}

	if !strings.Contains(output, "h/*:") {
		t.Errorf("Expected output to contain 'h/*:' section header, got: %s", output)
	}

	if !strings.Contains(output, "s/*:") {
		t.Errorf("Expected output to contain 's/*:' section header, got: %s", output)
	}
}

// TestOverviewWithCurrentBranch tests the overview command with the current branch highlighted.
// Steps:
// 1. Sets up a test repository and initializes git-flow
// 2. Creates and checks out a feature branch
// 3. Runs the overview command
// 4. Verifies the current branch is marked with an asterisk (*)
// 5. Verifies the branch information is correct and complete
func TestOverviewWithCurrentBranch(t *testing.T) {
	t.Parallel()
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Create a feature branch and stay on it
	output, err = testutil.RunGitFlow(t, dir, "feature", "start", "my-feature")
	if err != nil {
		t.Fatalf("Failed to create feature branch: %v\nOutput: %s", err, output)
	}

	// Run git-flow overview
	output, err = testutil.RunGitFlow(t, dir, "overview")
	if err != nil {
		t.Fatalf("Failed to run git-flow overview: %v\nOutput: %s", err, output)
	}

	// Check if the output shows the current branch with an asterisk
	if !strings.Contains(output, "* feature/my-feature (feature)") {
		t.Errorf("Expected output to contain '* feature/my-feature (feature)', got: %s", output)
	}
}

// TestOverviewWithAVHConfig tests the overview command with existing git-flow-avh configuration.
// Steps:
// 1. Sets up a test repository
// 2. Configures the repository with git-flow-avh configuration values without running init
// 3. Runs the overview command
// 4. Verifies the output correctly interprets the git-flow-avh config
// 5. Verifies all branch types and prefixes are correctly displayed
func TestOverviewWithAVHConfig(t *testing.T) {
	t.Parallel()
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Set git-flow-avh config values directly instead of running init
	// Set branch structure
	_, err := testutil.RunGit(t, dir, "config", "gitflow.branch.master", "main")
	if err != nil {
		t.Fatalf("Failed to set gitflow.branch.master config: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "config", "gitflow.branch.develop", "develop")
	if err != nil {
		t.Fatalf("Failed to set gitflow.branch.develop config: %v", err)
	}

	// Set branch prefixes
	_, err = testutil.RunGit(t, dir, "config", "gitflow.prefix.feature", "feature/")
	if err != nil {
		t.Fatalf("Failed to set gitflow.prefix.feature config: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "config", "gitflow.prefix.bugfix", "bugfix/")
	if err != nil {
		t.Fatalf("Failed to set gitflow.prefix.bugfix config: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "config", "gitflow.prefix.release", "release/")
	if err != nil {
		t.Fatalf("Failed to set gitflow.prefix.release config: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "config", "gitflow.prefix.hotfix", "hotfix/")
	if err != nil {
		t.Fatalf("Failed to set gitflow.prefix.hotfix config: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "config", "gitflow.prefix.support", "support/")
	if err != nil {
		t.Fatalf("Failed to set gitflow.prefix.support config: %v", err)
	}
	_, err = testutil.RunGit(t, dir, "config", "gitflow.prefix.versiontag", "v")
	if err != nil {
		t.Fatalf("Failed to set gitflow.prefix.versiontag config: %v", err)
	}

	// Create the develop branch
	_, err = testutil.RunGit(t, dir, "checkout", "-b", "develop")
	if err != nil {
		t.Fatalf("Failed to create develop branch: %v", err)
	}

	// Create a feature branch to test active branch detection
	_, err = testutil.RunGit(t, dir, "checkout", "-b", "feature/test-feature", "develop")
	if err != nil {
		t.Fatalf("Failed to create feature branch: %v", err)
	}

	// Run git-flow overview
	output, err := testutil.RunGitFlow(t, dir, "overview")
	if err != nil {
		t.Fatalf("Failed to run git-flow overview: %v\nOutput: %s", err, output)
	}

	// Check if the output contains the expected sections
	if !strings.Contains(output, "Base branches:") {
		t.Errorf("Expected output to contain 'Base branches:', got: %s", output)
	}

	if !strings.Contains(output, "Topic branch types:") {
		t.Errorf("Expected output to contain 'Topic branch types:', got: %s", output)
	}

	// Check if the output contains the base branches with correct relationship
	if !strings.Contains(output, "develop → main [auto-update]") {
		t.Errorf("Expected output to contain 'develop → main [auto-update]', got: %s", output)
	}

	// Check if the output contains all the branch types with correct prefixes as section headers
	if !strings.Contains(output, "feature/*:") {
		t.Errorf("Expected output to contain 'feature/*:' section header, got: %s", output)
	}

	if !strings.Contains(output, "bugfix/*:") {
		t.Errorf("Expected output to contain 'bugfix/*:' section header, got: %s", output)
	}

	if !strings.Contains(output, "release/*:") {
		t.Errorf("Expected output to contain 'release/*:' section header, got: %s", output)
	}

	if !strings.Contains(output, "hotfix/*:") {
		t.Errorf("Expected output to contain 'hotfix/*:' section header, got: %s", output)
	}

	if !strings.Contains(output, "support/*:") {
		t.Errorf("Expected output to contain 'support/*:' section header, got: %s", output)
	}

	// Check if active feature branch is displayed
	if !strings.Contains(output, "* feature/test-feature (feature)") {
		t.Errorf("Expected output to contain '* feature/test-feature (feature)', got: %s", output)
	}
}

// runOverviewRepeatedly runs the overview command the given number of times and
// returns each run's output. Go randomizes map iteration per run, so a single
// run cannot distinguish a stable order from a lucky one.
func runOverviewRepeatedly(t *testing.T, dir string, runs int) []string {
	t.Helper()

	outputs := make([]string, 0, runs)
	for i := 0; i < runs; i++ {
		output, err := testutil.RunGitFlow(t, dir, "overview")
		if err != nil {
			t.Fatalf("Failed to run git-flow overview (run %d): %v\nOutput: %s", i+1, err, output)
		}
		outputs = append(outputs, output)
	}
	return outputs
}

// overviewLinesMatching returns the lines of overview output matching the given
// predicate, in the order they appear.
func overviewLinesMatching(output string, match func(line string) bool) []string {
	var lines []string
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimRight(line, "\r")
		if match(line) {
			lines = append(lines, line)
		}
	}
	return lines
}

// setOverviewConfig sets a git config key, failing the test if it does not take.
// A silently dropped key would change the configured branch set the assertions
// are written against.
func setOverviewConfig(t *testing.T, dir string, key string, value string) {
	t.Helper()

	if _, err := testutil.RunGit(t, dir, "config", key, value); err != nil {
		t.Fatalf("Failed to set %s: %v", key, err)
	}
}

// createOverviewBranch creates a branch off the given start point, failing the
// test if it does not come into existence.
func createOverviewBranch(t *testing.T, dir string, name string, startPoint string) {
	t.Helper()

	if _, err := testutil.RunGit(t, dir, "checkout", "-b", name, startPoint); err != nil {
		t.Fatalf("Failed to create branch %s: %v", name, err)
	}
}

// TestOverviewTopicTypeOrderIsDeterministic tests that the topic branch type
// sections are listed in a stable, alphabetical order across identical runs.
// Steps:
// 1. Sets up a test repository and initializes git-flow with defaults
// 2. Runs the overview command several times
// 3. Verifies every run lists the topic type sections sorted by branch type name
func TestOverviewTopicTypeOrderIsDeterministic(t *testing.T) {
	t.Parallel()
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// The default topic types, sorted by branch type name. Their prefixes carry
	// the same order, so the section headers spell it out directly.
	expected := []string{"bugfix/*:", "feature/*:", "hotfix/*:", "release/*:", "support/*:"}

	for i, out := range runOverviewRepeatedly(t, dir, 5) {
		headers := overviewLinesMatching(out, func(line string) bool {
			return strings.HasSuffix(line, "*:")
		})
		if strings.Join(headers, " ") != strings.Join(expected, " ") {
			t.Errorf("Run %d: expected topic type sections %v, got %v\nOutput: %s", i+1, expected, headers, out)
		}
	}
}

// TestOverviewBaseBranchOrderIsDeterministic tests that the base branch lists
// are stable across identical runs. The default configuration has a single
// trunk and a single child base branch, so a second one of each is configured.
// Steps:
// 1. Sets up a test repository and initializes git-flow with defaults
// 2. Adds a second trunk branch and a second child base branch
// 3. Runs the overview command several times
// 4. Verifies every run lists both trunks and both children in sorted order
func TestOverviewBaseBranchOrderIsDeterministic(t *testing.T) {
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
	createOverviewBranch(t, dir, "archive", "main")
	setOverviewConfig(t, dir, "gitflow.branch.archive.type", "base")

	// A second child base branch, sorting after the default develop
	createOverviewBranch(t, dir, "staging", "main")
	setOverviewConfig(t, dir, "gitflow.branch.staging.type", "base")
	setOverviewConfig(t, dir, "gitflow.branch.staging.parent", "main")

	expectedTrunks := []string{"  archive (root)", "  main (root)"}
	expectedChildren := []string{"  develop → main [auto-update]", "  staging → main"}

	for i, out := range runOverviewRepeatedly(t, dir, 5) {
		trunks := overviewLinesMatching(out, func(line string) bool {
			return strings.HasSuffix(line, "(root)")
		})
		if strings.Join(trunks, " ") != strings.Join(expectedTrunks, " ") {
			t.Errorf("Run %d: expected trunk branches %v, got %v\nOutput: %s", i+1, expectedTrunks, trunks, out)
		}

		children := overviewLinesMatching(out, func(line string) bool {
			return strings.Contains(line, "→")
		})
		if strings.Join(children, " ") != strings.Join(expectedChildren, " ") {
			t.Errorf("Run %d: expected child base branches %v, got %v\nOutput: %s", i+1, expectedChildren, children, out)
		}
	}
}

// TestOverviewActiveBranchTypeIsDeterministic tests that the branch type
// reported for an active topic branch is stable when more than one configured
// prefix matches it. The first match in sorted branch type order wins.
// Steps:
// 1. Sets up a test repository and initializes git-flow with defaults
// 2. Adds a topic type whose prefix extends the default feature prefix
// 3. Creates a branch matching both prefixes
// 4. Runs the overview command several times
// 5. Verifies every run reports the same branch type for that branch
func TestOverviewActiveBranchTypeIsDeterministic(t *testing.T) {
	t.Parallel()
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// A topic type whose prefix extends the default 'feature/' prefix, so a
	// branch below it matches two configured types
	setOverviewConfig(t, dir, "gitflow.branch.exp.type", "topic")
	setOverviewConfig(t, dir, "gitflow.branch.exp.parent", "develop")
	setOverviewConfig(t, dir, "gitflow.branch.exp.prefix", "feature/exp/")

	createOverviewBranch(t, dir, "feature/exp/trial", "develop")

	// 'exp' sorts before 'feature', so it is the type reported for the branch
	expected := "* feature/exp/trial (exp)"

	for i, out := range runOverviewRepeatedly(t, dir, 5) {
		if !strings.Contains(out, expected) {
			t.Errorf("Run %d: expected output to contain %q, got: %s", i+1, expected, out)
		}
	}
}
