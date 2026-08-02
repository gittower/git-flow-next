package cmd_test

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/gittower/git-flow-next/test/testutil"
)

// TestInitFreshRepo tests that init works on a fresh repository without --force.
// Steps:
// 1. Sets up a new test repository
// 2. Runs 'git flow init --defaults'
// 3. Verifies initialization succeeds and config is created
func TestInitFreshRepo(t *testing.T) {
	t.Parallel()
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Run git-flow init --defaults
	output, err := runGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to run git-flow init --defaults: %v\nOutput: %s", err, output)
	}

	// Check if the output contains the expected message
	if !strings.Contains(output, "Initializing git-flow with default settings") {
		t.Errorf("Expected output to contain 'Initializing git-flow with default settings', got: %s", output)
	}

	// Verify configuration was created
	version := getGitConfig(t, dir, "gitflow.version")
	if version != "1.0" {
		t.Errorf("Expected gitflow.version to be '1.0', got: %s", version)
	}

	initialized := getGitConfig(t, dir, "gitflow.initialized")
	if initialized != "true" {
		t.Errorf("Expected gitflow.initialized to be 'true', got: %s", initialized)
	}
}

// TestInitWithoutForceFailsWhenAlreadyInitialized tests that re-init without --force fails.
// Steps:
// 1. Sets up a test repository and initializes git-flow
// 2. Attempts to run 'git flow init --defaults' again without --force
// 3. Verifies the operation fails with "already initialized" error
func TestInitWithoutForceFailsWhenAlreadyInitialized(t *testing.T) {
	t.Parallel()
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// First initialization
	output, err := runGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed initial git-flow init: %v\nOutput: %s", err, output)
	}

	// Attempt second initialization without --force
	output, err = runGitFlow(t, dir, "init", "--defaults")
	if err == nil {
		t.Fatalf("Expected git-flow init to fail when already initialized, but it succeeded\nOutput: %s", output)
	}

	// Check error message
	if !strings.Contains(output, "already initialized") {
		t.Errorf("Expected error to contain 'already initialized', got: %s", output)
	}

	if !strings.Contains(output, "--force") {
		t.Errorf("Expected error to mention '--force' option, got: %s", output)
	}
}

// TestInitWithForceSucceedsWhenAlreadyInitialized tests that re-init with --force succeeds.
// Steps:
// 1. Sets up a test repository and initializes git-flow
// 2. Runs 'git flow init --defaults --force'
// 3. Verifies the operation succeeds and shows reconfiguration message
func TestInitWithForceSucceedsWhenAlreadyInitialized(t *testing.T) {
	t.Parallel()
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// First initialization
	output, err := runGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed initial git-flow init: %v\nOutput: %s", err, output)
	}

	// Second initialization with --force
	output, err = runGitFlow(t, dir, "init", "--defaults", "--force")
	if err != nil {
		t.Fatalf("Failed to run git-flow init --defaults --force: %v\nOutput: %s", err, output)
	}

	// Check for reconfiguration message
	if !strings.Contains(output, "Reconfiguring git-flow") {
		t.Errorf("Expected output to contain 'Reconfiguring git-flow', got: %s", output)
	}

	// Verify configuration still exists
	version := getGitConfig(t, dir, "gitflow.version")
	if version != "1.0" {
		t.Errorf("Expected gitflow.version to be '1.0', got: %s", version)
	}
}

// TestInitInteractiveReconfirmYes tests interactive re-init with 'y' confirmation.
// Steps:
// 1. Sets up a test repository and initializes git-flow
// 2. Runs 'git flow init' interactively with 'y' response to reconfigure prompt
// 3. Verifies the operation proceeds after confirmation
func TestInitInteractiveReconfirmYes(t *testing.T) {
	t.Parallel()
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// First initialization with defaults
	output, err := runGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed initial git-flow init: %v\nOutput: %s", err, output)
	}

	// Interactive second initialization - confirm with 'y' followed by standard interactive input
	// Input: y (confirm), then defaults for all prompts (empty lines use defaults)
	input := "y\n\n\n\n\n\n\n\n"
	output, err = runGitFlowWithInput(t, dir, input, "init")
	if err != nil {
		t.Fatalf("Failed to run interactive git-flow init with confirmation: %v\nOutput: %s", err, output)
	}

	// Check for confirmation prompt
	if !strings.Contains(output, "already configured") {
		t.Errorf("Expected output to contain 'already configured', got: %s", output)
	}

	if !strings.Contains(output, "reconfigure") {
		t.Errorf("Expected output to contain 'reconfigure', got: %s", output)
	}

	// Verify initialization completed
	if !strings.Contains(output, "Git flow has been initialized") {
		t.Errorf("Expected output to contain 'Git flow has been initialized', got: %s", output)
	}
}

// TestInitInteractiveReconfirmNo tests interactive re-init with 'n' cancellation.
// Steps:
// 1. Sets up a test repository and initializes git-flow
// 2. Runs 'git flow init' interactively with 'n' response to reconfigure prompt
// 3. Verifies the operation is cancelled
func TestInitInteractiveReconfirmNo(t *testing.T) {
	t.Parallel()
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// First initialization with defaults
	output, err := runGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed initial git-flow init: %v\nOutput: %s", err, output)
	}

	// Get initial config to compare later
	initialFeaturePrefix := getGitConfig(t, dir, "gitflow.branch.feature.prefix")

	// Interactive second initialization - cancel with 'n'
	input := "n\n"
	output, err = runGitFlowWithInput(t, dir, input, "init")
	if err != nil {
		t.Fatalf("Unexpected error when cancelling reconfiguration: %v\nOutput: %s", err, output)
	}

	// Check for cancellation message
	if !strings.Contains(output, "cancelled") {
		t.Errorf("Expected output to contain 'cancelled', got: %s", output)
	}

	// Verify configuration wasn't changed
	currentFeaturePrefix := getGitConfig(t, dir, "gitflow.branch.feature.prefix")
	if currentFeaturePrefix != initialFeaturePrefix {
		t.Errorf("Expected feature prefix to remain '%s', got: '%s'", initialFeaturePrefix, currentFeaturePrefix)
	}
}

// TestInitForceWithPresetChange tests force reconfiguration with a different preset.
// Steps:
// 1. Sets up a test repository and initializes git-flow with default preset
// 2. Runs 'git flow init --preset github --force'
// 3. Verifies the configuration is updated to GitHub preset
func TestInitForceWithPresetChange(t *testing.T) {
	t.Parallel()
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// First initialization with classic preset (defaults)
	output, err := runGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed initial git-flow init: %v\nOutput: %s", err, output)
	}

	// Verify develop branch exists (classic preset has develop)
	developType := getGitConfig(t, dir, "gitflow.branch.develop.type")
	if developType != "base" {
		t.Errorf("Expected develop branch to exist after classic init, got type: %s", developType)
	}

	// Reinitialize with github preset
	output, err = runGitFlow(t, dir, "init", "--preset", "github", "--force")
	if err != nil {
		t.Fatalf("Failed to run git-flow init --preset github --force: %v\nOutput: %s", err, output)
	}

	// Verify GitHub preset was applied (feature branches point to main)
	featureParent := getGitConfig(t, dir, "gitflow.branch.feature.parent")
	if featureParent != "main" {
		t.Errorf("Expected feature parent to be 'main' for GitHub preset, got: %s", featureParent)
	}
}

// TestInitForceWithCustomPrefix tests force reconfiguration with custom branch prefix.
// Steps:
// 1. Sets up a test repository and initializes git-flow with defaults
// 2. Runs 'git flow init --force --feature feat/'
// 3. Verifies the feature prefix is updated
func TestInitForceWithCustomPrefix(t *testing.T) {
	t.Parallel()
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// First initialization with defaults
	output, err := runGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed initial git-flow init: %v\nOutput: %s", err, output)
	}

	// Verify initial feature prefix
	initialPrefix := getGitConfig(t, dir, "gitflow.branch.feature.prefix")
	if initialPrefix != "feature/" {
		t.Errorf("Expected initial feature prefix to be 'feature/', got: %s", initialPrefix)
	}

	// Reinitialize with custom prefix
	output, err = runGitFlow(t, dir, "init", "--force", "--feature", "feat/")
	if err != nil {
		t.Fatalf("Failed to run git-flow init --force --feature feat/: %v\nOutput: %s", err, output)
	}

	// Verify feature prefix was updated
	newPrefix := getGitConfig(t, dir, "gitflow.branch.feature.prefix")
	if newPrefix != "feat/" {
		t.Errorf("Expected feature prefix to be 'feat/', got: %s", newPrefix)
	}
}

// TestInitWithPresetFailsWithoutForce tests that re-init with preset fails without --force.
// Steps:
// 1. Sets up a test repository and initializes git-flow
// 2. Attempts to run 'git flow init --preset github' without --force
// 3. Verifies the operation fails
func TestInitWithPresetFailsWithoutForce(t *testing.T) {
	t.Parallel()
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// First initialization
	output, err := runGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed initial git-flow init: %v\nOutput: %s", err, output)
	}

	// Attempt second initialization with preset but without --force
	output, err = runGitFlow(t, dir, "init", "--preset", "github")
	if err == nil {
		t.Fatalf("Expected git-flow init --preset to fail when already initialized, but it succeeded\nOutput: %s", output)
	}

	// Check error message
	if !strings.Contains(output, "already initialized") {
		t.Errorf("Expected error to contain 'already initialized', got: %s", output)
	}
}

// TestInitWithCustomFlagFailsWithoutForce tests that re-init with custom flags fails without --force.
// Steps:
// 1. Sets up a test repository and initializes git-flow
// 2. Attempts to run 'git flow init --feature feat/' without --force
// 3. Verifies the operation fails
func TestInitWithCustomFlagFailsWithoutForce(t *testing.T) {
	t.Parallel()
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// First initialization
	output, err := runGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed initial git-flow init: %v\nOutput: %s", err, output)
	}

	// Attempt second initialization with custom flag but without --force
	output, err = runGitFlow(t, dir, "init", "--feature", "feat/")
	if err == nil {
		t.Fatalf("Expected git-flow init with custom flag to fail when already initialized, but it succeeded\nOutput: %s", output)
	}

	// Check error message
	if !strings.Contains(output, "already initialized") {
		t.Errorf("Expected error to contain 'already initialized', got: %s", output)
	}
}

// TestAVHConfigDoesNotRequireForce verifies that importing AVH config works without --force.
// AVH-only configuration (without gitflow.version) should be importable without --force,
// since it's not considered a git-flow-next re-initialization.
// Steps:
// 1. Sets up a test repository with AVH configuration only
// 2. Runs 'git flow init' without --force
// 3. Verifies the operation succeeds and imports AVH config
func TestAVHConfigDoesNotRequireForce(t *testing.T) {
	t.Parallel()
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Set up AVH configuration (without git-flow-next's gitflow.version)
	avhConfigs := [][]string{
		{"gitflow.branch.master", "main"},
		{"gitflow.branch.develop", "dev"},
		{"gitflow.prefix.feature", "feat/"},
		{"gitflow.prefix.release", "rel/"},
		{"gitflow.prefix.hotfix", "fix/"},
	}

	for _, cfg := range avhConfigs {
		cmd := exec.Command("git", "config", cfg[0], cfg[1])
		cmd.Dir = dir
		if err := cmd.Run(); err != nil {
			t.Fatalf("Failed to set %s: %v", cfg[0], err)
		}
	}

	// Run git-flow init without --force - should work since only AVH config exists
	output, err := runGitFlow(t, dir, "init")
	if err != nil {
		t.Fatalf("Expected git-flow init to succeed with AVH-only config, but it failed: %v\nOutput: %s", err, output)
	}

	// Check that AVH import message was shown
	if !strings.Contains(output, "Found existing git-flow-avh configuration, importing") {
		t.Errorf("Expected output to contain AVH import message, got: %s", output)
	}

	// Verify that imported config has the AVH prefixes
	featurePrefix := getGitConfig(t, dir, "gitflow.branch.feature.prefix")
	if featurePrefix != "feat/" {
		t.Errorf("Expected feature prefix to be 'feat/' from AVH import, got: %s", featurePrefix)
	}
}

// TestInitForceOnFreshRepo verifies that --force on an uninitialized repo works normally.
// Steps:
// 1. Sets up a new test repository
// 2. Runs 'git flow init --defaults --force'
// 3. Verifies the operation succeeds
func TestInitForceOnFreshRepo(t *testing.T) {
	t.Parallel()
	// Setup
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// Run git-flow init --defaults --force on fresh repo
	output, err := runGitFlow(t, dir, "init", "--defaults", "--force")
	if err != nil {
		t.Fatalf("Expected git-flow init --force to succeed on fresh repo, but it failed: %v\nOutput: %s", err, output)
	}

	// Should show normal initialization message (not reconfiguration)
	if !strings.Contains(output, "Initializing git-flow with default settings") {
		t.Errorf("Expected output to contain 'Initializing git-flow with default settings', got: %s", output)
	}

	// Should NOT show reconfiguration message since repo wasn't initialized before
	if strings.Contains(output, "Reconfiguring git-flow") {
		t.Errorf("Did not expect reconfiguration message on fresh repo, got: %s", output)
	}

	// Verify configuration was created
	version := getGitConfig(t, dir, "gitflow.version")
	if version != "1.0" {
		t.Errorf("Expected gitflow.version to be '1.0', got: %s", version)
	}
}
