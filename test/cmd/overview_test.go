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

	if !strings.Contains(output, "Topic branch configurations:") {
		t.Errorf("Expected output to contain 'Topic branch configurations:', got: %s", output)
	}

	if !strings.Contains(output, "Active topic branches:") {
		t.Errorf("Expected output to contain 'Active topic branches:', got: %s", output)
	}

	// Check if the output contains the base branches with their relationships
	if !strings.Contains(output, "develop -> main") {
		t.Errorf("Expected output to contain 'develop -> main', got: %s", output)
	}

	if !strings.Contains(output, "main -> (root)") {
		t.Errorf("Expected output to contain 'main -> (root)', got: %s", output)
	}

	// Check if the output contains the merge strategy information for base branches
	if !strings.Contains(output, "Upstream: merge, Downstream: merge") {
		t.Errorf("Expected output to contain 'Upstream: merge, Downstream: merge', got: %s", output)
	}

	if !strings.Contains(output, "Upstream: none, Downstream: none") {
		t.Errorf("Expected output to contain 'Upstream: none, Downstream: none', got: %s", output)
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
	if !strings.Contains(output, "custom-dev -> custom-main") {
		t.Errorf("Expected output to contain 'custom-dev -> custom-main', got: %s", output)
	}

	if !strings.Contains(output, "custom-main -> (root)") {
		t.Errorf("Expected output to contain 'custom-main -> (root)', got: %s", output)
	}

	// Check if the output contains the custom topic branch prefixes
	if !strings.Contains(output, "Prefix: f/") {
		t.Errorf("Expected output to contain 'Prefix: f/', got: %s", output)
	}

	if !strings.Contains(output, "Prefix: r/") {
		t.Errorf("Expected output to contain 'Prefix: r/', got: %s", output)
	}

	if !strings.Contains(output, "Prefix: h/") {
		t.Errorf("Expected output to contain 'Prefix: h/', got: %s", output)
	}

	if !strings.Contains(output, "Prefix: s/") {
		t.Errorf("Expected output to contain 'Prefix: s/', got: %s", output)
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

	if !strings.Contains(output, "Topic branch configurations:") {
		t.Errorf("Expected output to contain 'Topic branch configurations:', got: %s", output)
	}

	// Check if the output contains the base branches with correct relationship
	if !strings.Contains(output, "develop -> main") {
		t.Errorf("Expected output to contain 'develop -> main', got: %s", output)
	}

	// Check if the output contains all the branch types with correct prefixes
	if !strings.Contains(output, "feature") || !strings.Contains(output, "Prefix: feature/") {
		t.Errorf("Expected output to contain feature branch type with prefix 'feature/', got: %s", output)
	}

	if !strings.Contains(output, "bugfix") || !strings.Contains(output, "Prefix: bugfix/") {
		t.Errorf("Expected output to contain bugfix branch type with prefix 'bugfix/', got: %s", output)
	}

	if !strings.Contains(output, "release") || !strings.Contains(output, "Prefix: release/") {
		t.Errorf("Expected output to contain release branch type with prefix 'release/', got: %s", output)
	}

	if !strings.Contains(output, "hotfix") || !strings.Contains(output, "Prefix: hotfix/") {
		t.Errorf("Expected output to contain hotfix branch type with prefix 'hotfix/', got: %s", output)
	}

	if !strings.Contains(output, "support") || !strings.Contains(output, "Prefix: support/") {
		t.Errorf("Expected output to contain support branch type with prefix 'support/', got: %s", output)
	}

	// Check if tag prefix is correctly displayed
	if !strings.Contains(output, "Tag prefix: v") {
		t.Errorf("Expected output to contain 'Tag prefix: v', got: %s", output)
	}

	// Check if active feature branch is displayed
	if !strings.Contains(output, "* feature/test-feature (feature)") {
		t.Errorf("Expected output to contain '* feature/test-feature (feature)', got: %s", output)
	}
}
