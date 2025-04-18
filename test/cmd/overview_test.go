package cmd_test

import (
	"strings"
	"testing"

	"github.com/gittower/git-flow-next/test/testutil"
)

// TestOverviewWithDefaultConfig tests the overview command with default configuration
func TestOverviewWithDefaultConfig(t *testing.T) {
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	output, err := testutil.RunGitFlow(t, dir, "init", "-d")
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

// TestOverviewWithActiveBranches tests the overview command with active branches
func TestOverviewWithActiveBranches(t *testing.T) {
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	output, err := testutil.RunGitFlow(t, dir, "init", "-d")
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

// TestOverviewWithCustomConfig tests the overview command with custom configuration
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

// TestOverviewWithCurrentBranch tests the overview command with the current branch highlighted
func TestOverviewWithCurrentBranch(t *testing.T) {
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Initialize git-flow with defaults
	output, err := testutil.RunGitFlow(t, dir, "init", "-d")
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
