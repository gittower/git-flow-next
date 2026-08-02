package cmd_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gittower/git-flow-next/test/testutil"
)

// Command-level coverage for dotted base/topic branch names (issue #93).
//
// The direct LoadConfig root-cause scenarios (1, 3, 4, 5, 6) live in
// test/internal/config/dotted_names_test.go. The no-regression scenarios
// (9: defaults, 10: dot-free custom names) are already covered by
// TestOverviewWithDefaultConfig and TestOverviewWithCustomConfig
// (custom-main/custom-dev) in overview_test.go, so they are not duplicated
// here. This file covers the remaining command-level scenarios: overview (2),
// end-to-end parent resolution on finish (7), and config list (8).

// TestOverviewDottedBaseBranches covers spec scenario 2: after init with dotted
// base branch names, `overview` lists them under "Base branches:" with the
// correct relationship annotations, and topic types still resolve their
// dotted parents. Presence and relationship are asserted, not line order.
func TestOverviewDottedBaseBranches(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	output, err := testutil.RunGitFlow(t, dir, "init",
		"--preset=classic", "--main=custom.main", "--develop=custom.dev")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	output, err = testutil.RunGitFlow(t, dir, "overview")
	if err != nil {
		t.Fatalf("Failed to run git-flow overview: %v\nOutput: %s", err, output)
	}

	if !strings.Contains(output, "custom.main (root)") {
		t.Errorf("Expected output to contain 'custom.main (root)', got: %s", output)
	}
	if !strings.Contains(output, "custom.dev → custom.main [auto-update]") {
		t.Errorf("Expected output to contain 'custom.dev → custom.main [auto-update]', got: %s", output)
	}

	// Topic types still print their dotted parents.
	if !strings.Contains(output, "Parent: custom.main") {
		t.Errorf("Expected a topic type with 'Parent: custom.main', got: %s", output)
	}
	if !strings.Contains(output, "Parent: custom.dev") {
		t.Errorf("Expected a topic type with 'Parent: custom.dev', got: %s", output)
	}
}

// TestFinishFeatureIntoDottedDevelop covers spec scenario 7: a dotted base
// branch resolves as a topic parent end-to-end — a feature started and
// finished merges into the dotted develop branch without a missing-base or
// unknown-branch-type failure.
func TestFinishFeatureIntoDottedDevelop(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	output, err := testutil.RunGitFlow(t, dir, "init",
		"--preset=classic", "--main=custom.main", "--develop=custom.dev")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	output, err = testutil.RunGitFlow(t, dir, "feature", "start", "x")
	if err != nil {
		t.Fatalf("Failed to start feature: %v\nOutput: %s", err, output)
	}

	testutil.WriteFile(t, dir, "feature.txt", "feature content")
	if _, err := testutil.RunGit(t, dir, "add", "feature.txt"); err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	if _, err := testutil.RunGit(t, dir, "commit", "-m", "Add feature file"); err != nil {
		t.Fatalf("Failed to commit file: %v", err)
	}

	output, err = testutil.RunGitFlow(t, dir, "feature", "finish", "x")
	if err != nil {
		t.Fatalf("Failed to finish feature (dotted parent did not resolve): %v\nOutput: %s", err, output)
	}

	// The feature branch is gone and its change landed on the dotted develop.
	if testutil.BranchExists(t, dir, "feature/x") {
		t.Error("Expected feature branch to be deleted after finish")
	}
	if _, err := testutil.RunGit(t, dir, "checkout", "custom.dev"); err != nil {
		t.Fatalf("Failed to checkout custom.dev: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "feature.txt")); os.IsNotExist(err) {
		t.Error("Expected feature.txt to be merged into custom.dev")
	}
}

// TestConfigListDottedNames covers spec scenario 8: `config list` renders the
// dotted base branch names with their original dots and the correct
// relationship, with no phantom single-segment entry.
func TestConfigListDottedNames(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	output, err := testutil.RunGitFlow(t, dir, "init",
		"--preset=classic", "--main=custom.main", "--develop=custom.dev")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	output, err = testutil.RunGitFlow(t, dir, "config", "list")
	if err != nil {
		t.Fatalf("Failed to run git-flow config list: %v\nOutput: %s", err, output)
	}

	if !strings.Contains(output, "custom.main → (root)") {
		t.Errorf("Expected output to contain 'custom.main → (root)', got: %s", output)
	}
	if !strings.Contains(output, "custom.dev → custom.main") {
		t.Errorf("Expected output to contain 'custom.dev → custom.main', got: %s", output)
	}

	// No phantom single-segment 'custom' base branch entry.
	if strings.Contains(output, "custom → (root)") || strings.Contains(output, "custom → custom") {
		t.Errorf("Did not expect a phantom 'custom' branch entry, got: %s", output)
	}
}

// TestConfigAddAndRenameDottedName verifies that `config add base` and
// `config rename` accept dotted branch names — the write/validation path
// counterpart to the read-path fix. Branch-name validation delegates to
// `git check-ref-format`, so a mid-name dot is accepted while git-invalid
// names are still rejected.
func TestConfigAddAndRenameDottedName(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	// Add a dotted base branch — previously rejected as an "invalid branch name".
	output, err = testutil.RunGitFlow(t, dir, "config", "add", "base", "custom.main", "main")
	if err != nil {
		t.Fatalf("config add base custom.main failed: %v\nOutput: %s", err, output)
	}
	if got, _ := testutil.RunGit(t, dir, "config", "--get", "gitflow.branch.custom.main.type"); strings.TrimSpace(got) != "base" {
		t.Errorf("Expected gitflow.branch.custom.main.type=base, got %q", got)
	}
	if !refExists(t, dir, "custom.main") {
		t.Error("Expected refs/heads/custom.main to be created")
	}

	// A git-invalid name is still rejected.
	if _, err := testutil.RunGitFlow(t, dir, "config", "add", "base", "custom..main", "main"); err == nil {
		t.Error("Expected config add base custom..main to be rejected")
	}

	// Rename an existing base to a dotted name.
	output, err = testutil.RunGitFlow(t, dir, "config", "rename", "base", "custom.main", "custom.trunk")
	if err != nil {
		t.Fatalf("config rename base to dotted name failed: %v\nOutput: %s", err, output)
	}
	if got, _ := testutil.RunGit(t, dir, "config", "--get", "gitflow.branch.custom.trunk.type"); strings.TrimSpace(got) != "base" {
		t.Errorf("Expected gitflow.branch.custom.trunk.type=base after rename, got %q", got)
	}
}
