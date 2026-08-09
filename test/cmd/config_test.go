package cmd_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gittower/git-flow-next/internal/config"
	"github.com/gittower/git-flow-next/internal/git"
	"github.com/gittower/git-flow-next/test/testutil"
)

// TestConfigAddBase tests adding base branch configurations.
// Steps:
// 1. Sets up a test repository and initializes git-flow
// 2. Adds various base branches with different configurations
// 3. Verifies branches are created and configuration is saved correctly
// 4. Tests error conditions like duplicate branches and invalid parents
func TestConfigAddBase(t *testing.T) {
	t.Parallel()
	// Setup test repository
	tempDir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, tempDir)

	// Initialize git-flow with defaults
	var err error
	_, err = testutil.RunGitFlow(t, tempDir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v", err)
	}

	tests := []struct {
		name            string
		branchName      string
		parent          string
		upstreamStrat   string
		downstreamStrat string
		autoUpdate      bool
		expectError     bool
	}{
		{"Add staging branch", "staging", "main", "merge", "merge", false, false},
		{"Add production branch", "production", "", "none", "none", false, false},
		{"Add duplicate branch", "staging", "main", "", "", false, true},
		{"Add with invalid parent", "test", "nonexistent", "", "", false, true},
		{"Add with circular dependency", "circular", "staging", "", "", false, false}, // This should work
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stderr to check for errors
			err := captureConfigAddBase(t, tempDir, tt.branchName, tt.parent, tt.upstreamStrat, tt.downstreamStrat, tt.autoUpdate)

			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if !tt.expectError {
				// Verify configuration was saved
				cfg, err := config.Load(mustOpenRepo(t, tempDir))
				if err != nil {
					t.Fatalf("Failed to load config: %v", err)
				}

				branchConfig, exists := cfg.Branches[tt.branchName]
				if !exists {
					t.Errorf("Branch %s not found in configuration", tt.branchName)
				}

				if branchConfig.Type != string(config.BranchTypeBase) {
					t.Errorf("Expected branch type 'base', got '%s'", branchConfig.Type)
				}

				if branchConfig.Parent != tt.parent {
					t.Errorf("Expected parent '%s', got '%s'", tt.parent, branchConfig.Parent)
				}
			}
		})
	}
}

// TestConfigAddTopic tests adding topic branch type configurations.
// Steps:
// 1. Sets up a test repository and initializes git-flow
// 2. Adds various topic branch types with different prefixes and settings
// 3. Verifies configurations are saved correctly with proper defaults
// 4. Tests error conditions like invalid parents and duplicate types
func TestConfigAddTopic(t *testing.T) {
	t.Parallel()
	// Setup test repository
	tempDir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, tempDir)

	// Initialize git-flow with defaults
	var err error
	_, err = testutil.RunGitFlow(t, tempDir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v", err)
	}

	tests := []struct {
		name            string
		branchName      string
		parent          string
		prefix          string
		startingPoint   string
		upstreamStrat   string
		downstreamStrat string
		tag             bool
		expectError     bool
	}{
		{"Add epic branch type", "epic", "develop", "epic/", "develop", "squash", "merge", false, false},
		{"Add experiment branch type", "experiment", "main", "exp/", "main", "merge", "none", true, false},
		{"Add duplicate branch type", "feature", "develop", "", "", "", "", false, true},
		{"Add with invalid parent", "test", "nonexistent", "", "", "", "", false, true},
		{"Add with invalid starting point", "test2", "develop", "", "nonexistent", "", "", false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := captureConfigAddTopic(t, tempDir, tt.branchName, tt.parent, tt.prefix, tt.startingPoint, tt.upstreamStrat, tt.downstreamStrat, tt.tag)

			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if !tt.expectError {
				// Verify configuration was saved
				cfg, err := config.Load(mustOpenRepo(t, tempDir))
				if err != nil {
					t.Fatalf("Failed to load config: %v", err)
				}

				branchConfig, exists := cfg.Branches[tt.branchName]
				if !exists {
					t.Errorf("Branch %s not found in configuration", tt.branchName)
				}

				if branchConfig.Type != string(config.BranchTypeTopic) {
					t.Errorf("Expected branch type 'topic', got '%s'", branchConfig.Type)
				}

				expectedPrefix := tt.prefix
				if expectedPrefix == "" {
					expectedPrefix = tt.branchName + "/"
				}
				if branchConfig.Prefix != expectedPrefix {
					t.Errorf("Expected prefix '%s', got '%s'", expectedPrefix, branchConfig.Prefix)
				}
			}
		})
	}
}

// TestConfigRenameBase tests renaming base branch configurations.
// Steps:
// 1. Sets up a test repository and initializes git-flow
// 2. Renames the develop branch to integration
// 3. Verifies both Git branch and configuration are updated
// 4. Verifies child branch references are updated
// 5. Tests error conditions like renaming nonexistent branches
func TestConfigRenameBase(t *testing.T) {
	t.Parallel()
	// Setup test repository
	tempDir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, tempDir)

	// Initialize git-flow with defaults
	var err error
	_, err = testutil.RunGitFlow(t, tempDir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v", err)
	}

	// Rename develop to integration
	err = captureConfigRenameBase(t, tempDir, "develop", "integration")
	if err != nil {
		t.Fatalf("Failed to rename base branch: %v", err)
	}
	// Verify configuration was updated
	cfg, err := config.Load(mustOpenRepo(t, tempDir))
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Check that old name is gone and new name exists
	if _, exists := cfg.Branches["develop"]; exists {
		t.Errorf("Old branch name 'develop' still exists in configuration")
	}

	integrationConfig, exists := cfg.Branches["integration"]
	if !exists {
		t.Fatalf("New branch name 'integration' not found in configuration")
	}

	if integrationConfig.Type != string(config.BranchTypeBase) {
		t.Errorf("Expected branch type 'base', got '%s'", integrationConfig.Type)
	}

	// Check that child branches were updated
	featureConfig := cfg.Branches["feature"]
	if featureConfig.Parent != "integration" {
		t.Errorf("Expected feature parent to be updated to 'integration', got '%s'", featureConfig.Parent)
	}

	// Test error cases
	t.Run("Rename nonexistent branch", func(t *testing.T) {
		err := captureConfigRenameBase(t, tempDir, "nonexistent", "newname")
		if err == nil {
			t.Errorf("Expected error when renaming nonexistent branch")
		}
	})

	t.Run("Rename to existing name", func(t *testing.T) {
		err := captureConfigRenameBase(t, tempDir, "integration", "main")
		if err == nil {
			t.Errorf("Expected error when renaming to existing branch name")
		}
	})
}

// TestConfigDeleteBase tests deleting base branch configurations.
// Steps:
// 1. Sets up a test repository and initializes git-flow
// 2. Adds a staging branch without dependents
// 3. Tests deletion of branch with dependents (should fail)
// 4. Tests deletion of branch without dependents (should succeed)
// 5. Verifies configuration is removed but Git branch remains
func TestConfigDeleteBase(t *testing.T) {
	t.Parallel()
	// Setup test repository
	tempDir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, tempDir)

	// Initialize git-flow with defaults and add staging branch
	var err error
	_, err = testutil.RunGitFlow(t, tempDir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v", err)
	}
	captureConfigAddBase(t, tempDir, "staging", "main", "", "", false)

	// Try to delete main branch (should fail because develop depends on it)
	t.Run("Delete branch with dependents", func(t *testing.T) {
		err := captureConfigDeleteBase(t, tempDir, "main")
		if err == nil {
			t.Errorf("Expected error when deleting branch with dependents")
		}
	})

	// Delete staging branch (should succeed)
	t.Run("Delete branch without dependents", func(t *testing.T) {
		err := captureConfigDeleteBase(t, tempDir, "staging")
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		// Verify configuration was updated
		cfg, err := config.Load(mustOpenRepo(t, tempDir))
		if err != nil {
			t.Fatalf("Failed to load config: %v", err)
		}

		if _, exists := cfg.Branches["staging"]; exists {
			t.Errorf("Staging branch should have been removed from configuration")
		}
	})

	// Test error case
	t.Run("Delete nonexistent branch", func(t *testing.T) {
		err := captureConfigDeleteBase(t, tempDir, "nonexistent")
		if err == nil {
			t.Errorf("Expected error when deleting nonexistent branch")
		}
	})
}

// TestConfigDeleteTopic tests deleting topic branch type configurations.
// Steps:
// 1. Sets up a test repository and initializes git-flow
// 2. Adds a custom topic branch type (e.g., support)
// 3. Verifies the topic branch type was added correctly
// 4. Deletes the topic branch type
// 5. Verifies configuration is removed from both in-memory and git config
func TestConfigDeleteTopic(t *testing.T) {
	t.Parallel()
	// Setup test repository
	tempDir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, tempDir)

	// Initialize git-flow with defaults
	_, err := testutil.RunGitFlow(t, tempDir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v", err)
	}

	// Add a custom topic branch type
	t.Run("Add experimental topic type", func(t *testing.T) {
		err := captureConfigAddTopic(t, tempDir, "experimental", "main", "experimental/", "", "", "", true)
		if err != nil {
			t.Fatalf("Failed to add experimental topic type: %v", err)
		}

		// Verify it was added
		cfg, err := config.Load(mustOpenRepo(t, tempDir))
		if err != nil {
			t.Fatalf("Failed to load config: %v", err)
		}

		if _, exists := cfg.Branches["experimental"]; !exists {
			t.Errorf("Experimental branch type was not added to configuration")
		}

		// Verify it was saved to git config
		output, err := testutil.RunGit(t, tempDir, "config", "--get-regexp", "^gitflow\\.branch\\.experimental")
		if err != nil {
			t.Errorf("Experimental branch type was not saved to git config")
		}
		if output == "" {
			t.Errorf("Expected git config entries for experimental branch type, got empty output")
		}
	})

	// Delete the experimental topic type
	t.Run("Delete experimental topic type", func(t *testing.T) {
		err := captureConfigDeleteTopic(t, tempDir, "experimental")
		if err != nil {
			t.Errorf("Unexpected error deleting topic type: %v", err)
		}

		// Verify configuration was removed from in-memory config
		cfg, err := config.Load(mustOpenRepo(t, tempDir))
		if err != nil {
			t.Fatalf("Failed to load config: %v", err)
		}

		if _, exists := cfg.Branches["experimental"]; exists {
			t.Errorf("Experimental branch type was not removed from configuration")
		}

		// Verify it was removed from git config
		output, err := testutil.RunGit(t, tempDir, "config", "--get-regexp", "^gitflow\\.branch\\.experimental")
		// Should return error because config doesn't exist
		if err == nil {
			t.Errorf("Expected error when getting deleted git config, but got none. Output: %s", output)
		}
	})

	// Test error case - delete nonexistent topic type
	t.Run("Delete nonexistent topic type", func(t *testing.T) {
		err := captureConfigDeleteTopic(t, tempDir, "nonexistent")
		if err == nil {
			t.Errorf("Expected error when deleting nonexistent topic type")
		}
	})

	// Test error case - try to delete a base branch as topic
	t.Run("Delete base branch as topic", func(t *testing.T) {
		err := captureConfigDeleteTopic(t, tempDir, "main")
		if err == nil {
			t.Errorf("Expected error when trying to delete base branch as topic")
		}
	})
}

// TestConfigList tests the configuration listing functionality.
// Steps:
// 1. Sets up a test repository without initialization
// 2. Tests listing configuration in uninitialized repository
// 3. Initializes git-flow and adds custom configuration
// 4. Tests listing configuration with branches and settings
// 5. Verifies all branch types and hierarchies are displayed correctly
func TestConfigList(t *testing.T) {
	t.Parallel()
	// Setup test repository
	tempDir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, tempDir)

	// Test list with no configuration
	t.Run("List uninitialized", func(t *testing.T) {
		_, err := captureConfigList(t, tempDir)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		// Should show message about not being initialized
	})

	// Initialize git-flow
	var err error
	_, err = testutil.RunGitFlow(t, tempDir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v", err)
	}

	// Add some custom configuration
	captureConfigAddBase(t, tempDir, "staging", "main", "", "", false)
	captureConfigAddTopic(t, tempDir, "epic", "develop", "epic/", "", "", "", false)

	// Test list with configuration
	t.Run("List with configuration", func(t *testing.T) {
		_, err := captureConfigList(t, tempDir)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		// Should show all branches and their configuration
	})
}

// TestPresetConfigurations tests the built-in workflow presets.
// Steps:
// 1. Sets up test repositories for each preset type
// 2. Initializes git-flow with each preset (classic, github, gitlab)
// 3. Verifies that expected branches are configured for each preset
// 4. Validates that preset-specific configurations are applied correctly
func TestPresetConfigurations(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name             string
		preset           string
		expectedBranches []string
	}{
		{"Classic GitFlow", "classic", []string{"main", "develop", "feature", "bugfix", "release", "hotfix", "support"}},
		{"GitHub Flow", "github", []string{"main", "feature"}},
		{"GitLab Flow", "gitlab", []string{"production", "staging", "main", "feature", "hotfix"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup test repository
			tempDir := testutil.SetupTestRepo(t)
			defer testutil.CleanupTestRepo(t, tempDir)

			// Initialize git-flow with preset
			var err error
			_, err = testutil.RunGitFlow(t, tempDir, "init", "--preset="+tt.preset)
			if err != nil {
				t.Fatalf("Failed to initialize git-flow with preset %s: %v", tt.preset, err)
			}

			// Verify configuration
			cfg, err := config.Load(mustOpenRepo(t, tempDir))
			if err != nil {
				t.Fatalf("Failed to load config: %v", err)
			}

			for _, expectedBranch := range tt.expectedBranches {
				if _, exists := cfg.Branches[expectedBranch]; !exists {
					t.Errorf("Expected branch '%s' not found in %s preset", expectedBranch, tt.name)
				}
			}
		})
	}
}

// TestCircularDependencyValidation tests validation of circular dependencies in branch configuration.
// Steps:
// 1. Sets up a test repository and initializes git-flow
// 2. Adds a staging branch with main as parent
// 3. Attempts to create circular dependency by making main depend on staging
// 4. Verifies that circular dependencies are detected and prevented
// 5. Ensures the system remains stable after validation failures
func TestCircularDependencyValidation(t *testing.T) {
	t.Parallel()
	// Setup test repository
	tempDir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, tempDir)

	// Initialize git-flow with defaults
	var err error
	_, err = testutil.RunGitFlow(t, tempDir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v", err)
	}

	// Add staging branch
	captureConfigAddBase(t, tempDir, "staging", "main", "", "", false)

	// Try to make main depend on staging (should fail)
	t.Run("Create circular dependency", func(t *testing.T) {
		err := captureConfigEditBase(t, tempDir, "main", "", "", false)
		// This test needs to be more specific about what constitutes a circular dependency
		// For now, just ensure the system doesn't crash
		if err != nil {
			t.Logf("Got expected error for circular dependency: %v", err)
		}
	})
}

// Helper functions to capture command execution without exiting

func captureConfigAddBase(t *testing.T, dir string, name, parent, upstreamStrategy, downstreamStrategy string, autoUpdate bool) error {
	// Build command arguments
	args := []string{"config", "add", "base", name}
	if parent != "" {
		args = append(args, parent)
	}
	if upstreamStrategy != "" {
		args = append(args, "--upstream-strategy="+upstreamStrategy)
	}
	if downstreamStrategy != "" {
		args = append(args, "--downstream-strategy="+downstreamStrategy)
	}
	if autoUpdate {
		args = append(args, "--auto-update")
	}

	// Run the command
	_, err := testutil.RunGitFlow(t, dir, args...)
	return err
}

func captureConfigAddTopic(t *testing.T, dir string, name, parent, prefix, startingPoint, upstreamStrategy, downstreamStrategy string, tag bool) error {
	// Build command arguments
	args := []string{"config", "add", "topic", name, parent}
	if prefix != "" {
		args = append(args, "--prefix="+prefix)
	}
	if startingPoint != "" {
		args = append(args, "--starting-point="+startingPoint)
	}
	if upstreamStrategy != "" {
		args = append(args, "--upstream-strategy="+upstreamStrategy)
	}
	if downstreamStrategy != "" {
		args = append(args, "--downstream-strategy="+downstreamStrategy)
	}
	if tag {
		args = append(args, "--tag")
	}

	// Run the command
	_, err := testutil.RunGitFlow(t, dir, args...)
	return err
}

func captureConfigRenameBase(t *testing.T, dir string, oldName, newName string) error {
	// Build command arguments
	args := []string{"config", "rename", "base", oldName, newName}

	// Run the command
	_, err := testutil.RunGitFlow(t, dir, args...)
	return err
}

func captureConfigDeleteBase(t *testing.T, dir string, name string) error {
	// Build command arguments
	args := []string{"config", "delete", "base", name}

	// Run the command
	_, err := testutil.RunGitFlow(t, dir, args...)
	return err
}

func captureConfigEditBase(t *testing.T, dir string, name, upstreamStrategy, downstreamStrategy string, autoUpdate bool) error {
	// Build command arguments
	args := []string{"config", "edit", "base", name}
	if upstreamStrategy != "" {
		args = append(args, "--upstream-strategy="+upstreamStrategy)
	}
	if downstreamStrategy != "" {
		args = append(args, "--downstream-strategy="+downstreamStrategy)
	}
	if autoUpdate {
		args = append(args, "--auto-update")
	}

	// Run the command
	_, err := testutil.RunGitFlow(t, dir, args...)
	return err
}

func captureConfigList(t *testing.T, dir string) (string, error) {
	// Build command arguments
	args := []string{"config", "list"}

	// Run the command
	return testutil.RunGitFlow(t, dir, args...)
}

func captureConfigDeleteTopic(t *testing.T, dir string, name string) error {
	// Build command arguments
	args := []string{"config", "delete", "topic", name}

	// Run the command
	_, err := testutil.RunGitFlow(t, dir, args...)
	return err
}

// gitflowBranchConfig returns the output of
// `git config --get-regexp '^gitflow\.branch\.'` for the test repo. On no
// matches git exits non-zero and returns an empty string; the tests treat an
// empty result as "no branch config lines" rather than a fatal error.
func gitflowBranchConfig(t *testing.T, dir string) string {
	output, _ := testutil.RunGit(t, dir, "config", "--get-regexp", "^gitflow\\.branch\\.")
	return output
}

// refExists reports whether refs/heads/<branch> resolves in the test repo.
func refExists(t *testing.T, dir string, branch string) bool {
	_, err := testutil.RunGit(t, dir, "rev-parse", "--verify", "refs/heads/"+branch)
	return err == nil
}

// assertContainsLine fails if none of the lines in output contains substr.
func assertContainsLine(t *testing.T, output, substr, context string) {
	t.Helper()
	if !strings.Contains(output, substr) {
		t.Errorf("%s: expected config to contain %q, got:\n%s", context, substr, output)
	}
}

// assertNoLineContains fails if any line in output contains substr.
func assertNoLineContains(t *testing.T, output, substr, context string) {
	t.Helper()
	if strings.Contains(output, substr) {
		t.Errorf("%s: expected config to NOT contain %q, got:\n%s", context, substr, output)
	}
}

// TestConfigAddBaseUppercaseName tests adding a base branch with an uppercase name.
// Steps:
// 1. Sets up a test repository and initializes git-flow with defaults
// 2. Runs 'config add base V9_Release'
// 3. Verifies gitflow.branch.V9_Release.type base is written (exact case)
// 4. Verifies no lowercase gitflow.branch.v9_release.* variant section exists
// 5. Verifies refs/heads/V9_Release was created
func TestConfigAddBaseUppercaseName(t *testing.T) {
	t.Parallel()
	tempDir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, tempDir)

	if _, err := testutil.RunGitFlow(t, tempDir, "init", "--defaults"); err != nil {
		t.Fatalf("Failed to initialize git-flow: %v", err)
	}

	if _, err := testutil.RunGitFlow(t, tempDir, "config", "add", "base", "V9_Release"); err != nil {
		t.Fatalf("Failed to add base branch V9_Release: %v", err)
	}

	cfg := gitflowBranchConfig(t, tempDir)
	assertContainsLine(t, cfg, "gitflow.branch.V9_Release.type base", "add base uppercase")
	assertNoLineContains(t, cfg, "gitflow.branch.v9_release.", "add base uppercase")
	if !refExists(t, tempDir, "V9_Release") {
		t.Errorf("Expected refs/heads/V9_Release to exist")
	}
}

// TestConfigAddBaseExactCaseParent tests adding a base branch referencing an
// existing uppercase parent by its exact case (the core #122 regression).
// Steps:
// 1. Sets up a test repository and initializes git-flow with defaults
// 2. Adds base branch V9_Release
// 3. Runs 'config add base V10_Release V9_Release'
// 4. Verifies success (no "does not exist" error) and the stored parent is V9_Release
// 5. Verifies refs/heads/V10_Release was created
func TestConfigAddBaseExactCaseParent(t *testing.T) {
	t.Parallel()
	tempDir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, tempDir)

	if _, err := testutil.RunGitFlow(t, tempDir, "init", "--defaults"); err != nil {
		t.Fatalf("Failed to initialize git-flow: %v", err)
	}
	if _, err := testutil.RunGitFlow(t, tempDir, "config", "add", "base", "V9_Release"); err != nil {
		t.Fatalf("Failed to add base branch V9_Release: %v", err)
	}

	output, err := testutil.RunGitFlow(t, tempDir, "config", "add", "base", "V10_Release", "V9_Release")
	if err != nil {
		t.Fatalf("Expected exact-case parent reference to succeed, got error: %v\nOutput: %s", err, output)
	}

	cfg := gitflowBranchConfig(t, tempDir)
	assertContainsLine(t, cfg, "gitflow.branch.V10_Release.parent V9_Release", "exact-case parent")
	if !refExists(t, tempDir, "V10_Release") {
		t.Errorf("Expected refs/heads/V10_Release to exist")
	}
}

// TestConfigAddBaseDifferentCaseParentUsesCanonicalRef tests that a different-case
// parent reference resolves to the canonical name and the git op uses the
// canonical ref (guards case-sensitive filesystems / Linux CI).
// Steps:
// 1. Sets up a test repository and initializes git-flow with defaults
// 2. Adds base branch V9_Release
// 3. Runs 'config add base V11_Release v9_release' (lowercase parent)
// 4. Verifies stored parent is canonical V9_Release, no v9_release section
// 5. Verifies refs/heads/V11_Release was created (from the canonical ref)
func TestConfigAddBaseDifferentCaseParentUsesCanonicalRef(t *testing.T) {
	t.Parallel()
	tempDir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, tempDir)

	if _, err := testutil.RunGitFlow(t, tempDir, "init", "--defaults"); err != nil {
		t.Fatalf("Failed to initialize git-flow: %v", err)
	}
	if _, err := testutil.RunGitFlow(t, tempDir, "config", "add", "base", "V9_Release"); err != nil {
		t.Fatalf("Failed to add base branch V9_Release: %v", err)
	}

	output, err := testutil.RunGitFlow(t, tempDir, "config", "add", "base", "V11_Release", "v9_release")
	if err != nil {
		t.Fatalf("Expected different-case parent reference to succeed, got error: %v\nOutput: %s", err, output)
	}

	cfg := gitflowBranchConfig(t, tempDir)
	assertContainsLine(t, cfg, "gitflow.branch.V11_Release.parent V9_Release", "different-case parent")
	assertNoLineContains(t, cfg, "gitflow.branch.v9_release.", "different-case parent")
	if !refExists(t, tempDir, "V11_Release") {
		t.Errorf("Expected refs/heads/V11_Release to exist (created from canonical ref)")
	}
}

// TestConfigAddBaseMixedCaseParentResolves tests arbitrary mixed casing on a
// parent reference resolves to the canonical name.
// Steps:
// 1. Sets up a test repository and initializes git-flow with defaults
// 2. Adds base branch V9_Release
// 3. Runs 'config add base V12_Release v9_RELEASE'
// 4. Verifies stored parent is canonical V9_Release, no case-variant section
func TestConfigAddBaseMixedCaseParentResolves(t *testing.T) {
	t.Parallel()
	tempDir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, tempDir)

	if _, err := testutil.RunGitFlow(t, tempDir, "init", "--defaults"); err != nil {
		t.Fatalf("Failed to initialize git-flow: %v", err)
	}
	if _, err := testutil.RunGitFlow(t, tempDir, "config", "add", "base", "V9_Release"); err != nil {
		t.Fatalf("Failed to add base branch V9_Release: %v", err)
	}

	output, err := testutil.RunGitFlow(t, tempDir, "config", "add", "base", "V12_Release", "v9_RELEASE")
	if err != nil {
		t.Fatalf("Expected mixed-case parent reference to succeed, got error: %v\nOutput: %s", err, output)
	}

	cfg := gitflowBranchConfig(t, tempDir)
	assertContainsLine(t, cfg, "gitflow.branch.V12_Release.parent V9_Release", "mixed-case parent")
	assertNoLineContains(t, cfg, "gitflow.branch.v9_release.", "mixed-case parent")
	assertNoLineContains(t, cfg, "gitflow.branch.v9_RELEASE.", "mixed-case parent")
}

// TestConfigAddTopicStartingPointResolvesCaseInsensitively tests that a topic
// --starting-point resolves case-insensitively to the canonical name.
// Steps:
// 1. Sets up a test repository and initializes git-flow with defaults
// 2. Adds base branch V9_Release
// 3. Runs 'config add topic candidate develop --starting-point v9_release'
// 4. Verifies the stored start point is canonical V9_Release, no v9_release section
func TestConfigAddTopicStartingPointResolvesCaseInsensitively(t *testing.T) {
	t.Parallel()
	tempDir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, tempDir)

	if _, err := testutil.RunGitFlow(t, tempDir, "init", "--defaults"); err != nil {
		t.Fatalf("Failed to initialize git-flow: %v", err)
	}
	if _, err := testutil.RunGitFlow(t, tempDir, "config", "add", "base", "V9_Release"); err != nil {
		t.Fatalf("Failed to add base branch V9_Release: %v", err)
	}

	output, err := testutil.RunGitFlow(t, tempDir, "config", "add", "topic", "candidate", "develop", "--starting-point", "v9_release")
	if err != nil {
		t.Fatalf("Expected topic add with case-insensitive starting point to succeed, got error: %v\nOutput: %s", err, output)
	}

	cfg := gitflowBranchConfig(t, tempDir)
	assertContainsLine(t, cfg, "gitflow.branch.candidate.startpoint V9_Release", "topic starting point")
	assertNoLineContains(t, cfg, "gitflow.branch.v9_release.", "topic starting point")
}

// TestConfigAddTopicParentResolvesCaseInsensitively tests that a topic parent
// resolves case-insensitively and the defaulted start point also reads canonical.
// Steps:
// 1. Sets up a test repository and initializes git-flow with defaults
// 2. Adds base branch V9_Release
// 3. Runs 'config add topic hotpatch v9_release --prefix hp/'
// 4. Verifies parent is canonical V9_Release
// 5. Verifies defaulted start point (== parent) is also canonical V9_Release
// 6. Verifies no v9_release case-variant section exists
func TestConfigAddTopicParentResolvesCaseInsensitively(t *testing.T) {
	t.Parallel()
	tempDir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, tempDir)

	if _, err := testutil.RunGitFlow(t, tempDir, "init", "--defaults"); err != nil {
		t.Fatalf("Failed to initialize git-flow: %v", err)
	}
	if _, err := testutil.RunGitFlow(t, tempDir, "config", "add", "base", "V9_Release"); err != nil {
		t.Fatalf("Failed to add base branch V9_Release: %v", err)
	}

	output, err := testutil.RunGitFlow(t, tempDir, "config", "add", "topic", "hotpatch", "v9_release", "--prefix", "hp/")
	if err != nil {
		t.Fatalf("Expected topic add with case-insensitive parent to succeed, got error: %v\nOutput: %s", err, output)
	}

	cfg := gitflowBranchConfig(t, tempDir)
	assertContainsLine(t, cfg, "gitflow.branch.hotpatch.parent V9_Release", "topic parent")
	assertContainsLine(t, cfg, "gitflow.branch.hotpatch.startpoint V9_Release", "topic defaulted start point")
	assertNoLineContains(t, cfg, "gitflow.branch.v9_release.", "topic parent")
}

// TestConfigAddTopicCaseOnlyReAddRejected tests that re-adding a topic whose name
// differs only in case from an existing one is rejected.
// Steps:
// 1. Sets up a test repository and initializes git-flow with defaults
// 2. Adds topic QA_Feature (prefix qa/)
// 3. Runs 'config add topic qa_feature develop --prefix qa2/'
// 4. Verifies the command fails with an "already exists" error naming QA_Feature
// 5. Verifies no qa_feature section written and original QA_Feature prefix unchanged
func TestConfigAddTopicCaseOnlyReAddRejected(t *testing.T) {
	t.Parallel()
	tempDir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, tempDir)

	if _, err := testutil.RunGitFlow(t, tempDir, "init", "--defaults"); err != nil {
		t.Fatalf("Failed to initialize git-flow: %v", err)
	}
	if _, err := testutil.RunGitFlow(t, tempDir, "config", "add", "topic", "QA_Feature", "develop", "--prefix", "qa/"); err != nil {
		t.Fatalf("Failed to add topic QA_Feature: %v", err)
	}

	output, err := testutil.RunGitFlow(t, tempDir, "config", "add", "topic", "qa_feature", "develop", "--prefix", "qa2/")
	if err == nil {
		t.Fatalf("Expected case-only re-add to fail, got success\nOutput: %s", output)
	}
	if !strings.Contains(output, "QA_Feature") {
		t.Errorf("Expected error to name existing canonical 'QA_Feature', got:\n%s", output)
	}

	cfg := gitflowBranchConfig(t, tempDir)
	assertNoLineContains(t, cfg, "gitflow.branch.qa_feature.", "case-only re-add rejected")
	assertContainsLine(t, cfg, "gitflow.branch.QA_Feature.prefix qa/", "case-only re-add rejected (original unchanged)")
}

// TestConfigAddBaseCaseOnlyVariantRejected tests that adding a case-only variant
// of an existing base name is rejected without mutating the existing entry.
// Steps:
// 1. Sets up a test repository and initializes git-flow with defaults
// 2. Adds base branch V9_Release
// 3. Runs 'config add base v9_release main'
// 4. Verifies the command fails with an "already exists" error naming V9_Release
// 5. Verifies no v9_release section and V9_Release still type base with no main parent
func TestConfigAddBaseCaseOnlyVariantRejected(t *testing.T) {
	t.Parallel()
	tempDir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, tempDir)

	if _, err := testutil.RunGitFlow(t, tempDir, "init", "--defaults"); err != nil {
		t.Fatalf("Failed to initialize git-flow: %v", err)
	}
	if _, err := testutil.RunGitFlow(t, tempDir, "config", "add", "base", "V9_Release"); err != nil {
		t.Fatalf("Failed to add base branch V9_Release: %v", err)
	}

	output, err := testutil.RunGitFlow(t, tempDir, "config", "add", "base", "v9_release", "main")
	if err == nil {
		t.Fatalf("Expected case-only variant add to fail, got success\nOutput: %s", output)
	}
	if !strings.Contains(output, "V9_Release") {
		t.Errorf("Expected error to name existing canonical 'V9_Release', got:\n%s", output)
	}

	cfg := gitflowBranchConfig(t, tempDir)
	assertNoLineContains(t, cfg, "gitflow.branch.v9_release.", "case-only variant rejected")
	assertContainsLine(t, cfg, "gitflow.branch.V9_Release.type base", "case-only variant rejected (existing intact)")
	assertNoLineContains(t, cfg, "gitflow.branch.V9_Release.parent main", "case-only variant rejected (no partial mutation)")
}

// TestConfigEditBaseResolvesCaseInsensitively tests that editing a base branch
// resolves the name case-insensitively to the canonical section.
// Steps:
// 1. Sets up a test repository and initializes git-flow with defaults
// 2. Adds base branch V9_Release
// 3. Runs 'config edit base v9_release --upstream-strategy rebase'
// 4. Verifies gitflow.branch.V9_Release.upstreamStrategy reads rebase
// 5. Verifies no v9_release duplicate section
func TestConfigEditBaseResolvesCaseInsensitively(t *testing.T) {
	t.Parallel()
	tempDir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, tempDir)

	if _, err := testutil.RunGitFlow(t, tempDir, "init", "--defaults"); err != nil {
		t.Fatalf("Failed to initialize git-flow: %v", err)
	}
	if _, err := testutil.RunGitFlow(t, tempDir, "config", "add", "base", "V9_Release"); err != nil {
		t.Fatalf("Failed to add base branch V9_Release: %v", err)
	}

	output, err := testutil.RunGitFlow(t, tempDir, "config", "edit", "base", "v9_release", "--upstream-strategy", "rebase")
	if err != nil {
		t.Fatalf("Expected case-insensitive edit to succeed, got error: %v\nOutput: %s", err, output)
	}

	cfg := gitflowBranchConfig(t, tempDir)
	assertContainsLine(t, cfg, "gitflow.branch.V9_Release.upstreamstrategy rebase", "edit base case-insensitive")
	assertNoLineContains(t, cfg, "gitflow.branch.v9_release.", "edit base case-insensitive")
}

// TestConfigEditTopicResolvesCaseInsensitively tests that editing a topic type
// resolves the name case-insensitively.
// Steps:
// 1. Sets up a test repository and initializes git-flow with defaults
// 2. Adds topic QA_Feature (prefix qa/)
// 3. Runs 'config edit topic qa_feature --upstream-strategy rebase'
// 4. Verifies gitflow.branch.QA_Feature.upstreamStrategy reads rebase
// 5. Verifies no qa_feature case-variant section
func TestConfigEditTopicResolvesCaseInsensitively(t *testing.T) {
	t.Parallel()
	tempDir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, tempDir)

	if _, err := testutil.RunGitFlow(t, tempDir, "init", "--defaults"); err != nil {
		t.Fatalf("Failed to initialize git-flow: %v", err)
	}
	if _, err := testutil.RunGitFlow(t, tempDir, "config", "add", "topic", "QA_Feature", "develop", "--prefix", "qa/"); err != nil {
		t.Fatalf("Failed to add topic QA_Feature: %v", err)
	}

	output, err := testutil.RunGitFlow(t, tempDir, "config", "edit", "topic", "qa_feature", "--upstream-strategy", "rebase")
	if err != nil {
		t.Fatalf("Expected case-insensitive topic edit to succeed, got error: %v\nOutput: %s", err, output)
	}

	cfg := gitflowBranchConfig(t, tempDir)
	assertContainsLine(t, cfg, "gitflow.branch.QA_Feature.upstreamstrategy rebase", "edit topic case-insensitive")
	assertNoLineContains(t, cfg, "gitflow.branch.qa_feature.", "edit topic case-insensitive")
}

// TestConfigDeleteBaseResolvesCaseInsensitively tests deleting a base branch by a
// case-variant name removes the canonical section.
// Steps:
// 1. Sets up a test repository and initializes git-flow with defaults
// 2. Adds base branch V9_Release
// 3. Runs 'config delete base v9_release'
// 4. Verifies the V9_Release section is fully removed (no V9_Release or v9_release lines)
func TestConfigDeleteBaseResolvesCaseInsensitively(t *testing.T) {
	t.Parallel()
	tempDir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, tempDir)

	if _, err := testutil.RunGitFlow(t, tempDir, "init", "--defaults"); err != nil {
		t.Fatalf("Failed to initialize git-flow: %v", err)
	}
	if _, err := testutil.RunGitFlow(t, tempDir, "config", "add", "base", "V9_Release"); err != nil {
		t.Fatalf("Failed to add base branch V9_Release: %v", err)
	}

	output, err := testutil.RunGitFlow(t, tempDir, "config", "delete", "base", "v9_release")
	if err != nil {
		t.Fatalf("Expected case-insensitive delete to succeed, got error: %v\nOutput: %s", err, output)
	}

	cfg := gitflowBranchConfig(t, tempDir)
	assertNoLineContains(t, cfg, "gitflow.branch.V9_Release.", "delete base case-insensitive")
	assertNoLineContains(t, cfg, "gitflow.branch.v9_release.", "delete base case-insensitive")
}

// TestConfigRenameBaseResolvesAndUpdatesChildren tests renaming a base branch by a
// case-variant name resolves to canonical, updates children, and renames the ref.
// Steps:
// 1. Sets up a test repository and initializes git-flow with defaults
// 2. Adds base V9_Release and base V10_Release with parent V9_Release
// 3. Runs 'config rename base v9_release V9_Stable'
// 4. Verifies V9_Release section removed, V9_Stable section present
// 5. Verifies V10_Release.parent now reads V9_Stable
// 6. Verifies refs/heads/V9_Stable exists and refs/heads/V9_Release does not
func TestConfigRenameBaseResolvesAndUpdatesChildren(t *testing.T) {
	t.Parallel()
	tempDir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, tempDir)

	if _, err := testutil.RunGitFlow(t, tempDir, "init", "--defaults"); err != nil {
		t.Fatalf("Failed to initialize git-flow: %v", err)
	}
	if _, err := testutil.RunGitFlow(t, tempDir, "config", "add", "base", "V9_Release"); err != nil {
		t.Fatalf("Failed to add base branch V9_Release: %v", err)
	}
	if _, err := testutil.RunGitFlow(t, tempDir, "config", "add", "base", "V10_Release", "V9_Release"); err != nil {
		t.Fatalf("Failed to add base branch V10_Release: %v", err)
	}

	output, err := testutil.RunGitFlow(t, tempDir, "config", "rename", "base", "v9_release", "V9_Stable")
	if err != nil {
		t.Fatalf("Expected case-insensitive rename to succeed, got error: %v\nOutput: %s", err, output)
	}

	cfg := gitflowBranchConfig(t, tempDir)
	assertNoLineContains(t, cfg, "gitflow.branch.V9_Release.", "rename base resolves")
	assertContainsLine(t, cfg, "gitflow.branch.V9_Stable.type base", "rename base resolves")
	assertContainsLine(t, cfg, "gitflow.branch.V10_Release.parent V9_Stable", "rename base updates children")
	if !refExists(t, tempDir, "V9_Stable") {
		t.Errorf("Expected refs/heads/V9_Stable to exist after rename")
	}
	if refExists(t, tempDir, "V9_Release") {
		t.Errorf("Expected refs/heads/V9_Release to be gone after rename")
	}
}

// TestConfigRenameBaseCaseCollisionRejected tests renaming a base to a case-variant
// of a *different* existing branch is rejected without touching config or refs.
// Steps:
// 1. Sets up a test repository and initializes git-flow with defaults
// 2. Adds base V9_Release and base Preprod
// 3. Runs 'config rename base Preprod v9_release'
// 4. Verifies the command fails with an "already exists" error naming V9_Release
// 5. Verifies both original sections and both refs are unchanged, no v9_release section
func TestConfigRenameBaseCaseCollisionRejected(t *testing.T) {
	t.Parallel()
	tempDir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, tempDir)

	if _, err := testutil.RunGitFlow(t, tempDir, "init", "--defaults"); err != nil {
		t.Fatalf("Failed to initialize git-flow: %v", err)
	}
	if _, err := testutil.RunGitFlow(t, tempDir, "config", "add", "base", "V9_Release"); err != nil {
		t.Fatalf("Failed to add base branch V9_Release: %v", err)
	}
	if _, err := testutil.RunGitFlow(t, tempDir, "config", "add", "base", "Preprod"); err != nil {
		t.Fatalf("Failed to add base branch Preprod: %v", err)
	}

	output, err := testutil.RunGitFlow(t, tempDir, "config", "rename", "base", "Preprod", "v9_release")
	if err == nil {
		t.Fatalf("Expected case-collision rename to fail, got success\nOutput: %s", output)
	}
	if !strings.Contains(output, "V9_Release") {
		t.Errorf("Expected error to name existing canonical 'V9_Release', got:\n%s", output)
	}

	cfg := gitflowBranchConfig(t, tempDir)
	assertContainsLine(t, cfg, "gitflow.branch.V9_Release.type base", "rename collision rejected (V9 intact)")
	assertContainsLine(t, cfg, "gitflow.branch.Preprod.type base", "rename collision rejected (Preprod intact)")
	assertNoLineContains(t, cfg, "gitflow.branch.v9_release.", "rename collision rejected (no variant)")
	if !refExists(t, tempDir, "V9_Release") {
		t.Errorf("Expected refs/heads/V9_Release to still exist")
	}
	if !refExists(t, tempDir, "Preprod") {
		t.Errorf("Expected refs/heads/Preprod to still exist")
	}
}

// TestConfigRenameBaseCaseOnlySelfRename tests a case-only self-rename
// re-canonicalizes the entry in place.
// Steps:
// 1. Sets up a test repository and initializes git-flow with defaults
// 2. Adds base V9_Release
// 3. Runs 'config rename base V9_Release v9_release'
// 4. Verifies exactly one section, now v9_release, and no V9_Release section
func TestConfigRenameBaseCaseOnlySelfRename(t *testing.T) {
	t.Parallel()
	tempDir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, tempDir)

	if _, err := testutil.RunGitFlow(t, tempDir, "init", "--defaults"); err != nil {
		t.Fatalf("Failed to initialize git-flow: %v", err)
	}
	if _, err := testutil.RunGitFlow(t, tempDir, "config", "add", "base", "V9_Release"); err != nil {
		t.Fatalf("Failed to add base branch V9_Release: %v", err)
	}

	output, err := testutil.RunGitFlow(t, tempDir, "config", "rename", "base", "V9_Release", "v9_release")
	if err != nil {
		t.Fatalf("Expected case-only self-rename to succeed, got error: %v\nOutput: %s", err, output)
	}

	cfg := gitflowBranchConfig(t, tempDir)
	assertContainsLine(t, cfg, "gitflow.branch.v9_release.type base", "case-only self-rename")
	assertNoLineContains(t, cfg, "gitflow.branch.V9_Release.", "case-only self-rename")

	// The Git ref must be re-cased too (the RenameBranchForce -M path). On a
	// case-sensitive filesystem the old-case ref must be gone and the new-case
	// ref present; this is the behavior the force fallback exists to provide.
	if !refExists(t, tempDir, "v9_release") {
		t.Errorf("Expected refs/heads/v9_release to exist after case-only self-rename")
	}
}

// TestConfigAddBaseAbsentParentErrors tests a genuinely-absent reference still errors
// without mutating the unrelated existing entry.
// Steps:
// 1. Sets up a test repository and initializes git-flow with defaults
// 2. Adds base V9_Release
// 3. Runs 'config add base X NoSuchBranch'
// 4. Verifies the command fails with a "does not exist" error for NoSuchBranch
// 5. Verifies no X section and existing V9_Release section intact
func TestConfigAddBaseAbsentParentErrors(t *testing.T) {
	t.Parallel()
	tempDir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, tempDir)

	if _, err := testutil.RunGitFlow(t, tempDir, "init", "--defaults"); err != nil {
		t.Fatalf("Failed to initialize git-flow: %v", err)
	}
	if _, err := testutil.RunGitFlow(t, tempDir, "config", "add", "base", "V9_Release"); err != nil {
		t.Fatalf("Failed to add base branch V9_Release: %v", err)
	}

	output, err := testutil.RunGitFlow(t, tempDir, "config", "add", "base", "X", "NoSuchBranch")
	if err == nil {
		t.Fatalf("Expected absent-parent add to fail, got success\nOutput: %s", output)
	}
	if !strings.Contains(output, "NoSuchBranch") {
		t.Errorf("Expected error to name missing 'NoSuchBranch', got:\n%s", output)
	}

	cfg := gitflowBranchConfig(t, tempDir)
	assertNoLineContains(t, cfg, "gitflow.branch.X.", "absent parent errors (no X section)")
	assertContainsLine(t, cfg, "gitflow.branch.V9_Release.type base", "absent parent errors (V9 intact)")
}

// TestConfigNoDuplicateFromReloadRoundTrip tests that a load+save round trip does
// not produce the #117 duplicate lowercase section.
// Steps:
// 1. Sets up a test repository and initializes git-flow with defaults
// 2. Adds base V9_Release
// 3. Runs 'config add base V10_Release v9_release' (triggers load+save)
// 4. Verifies success and V10_Release created with canonical parent V9_Release
// 5. Verifies exactly one V9 section and no duplicate v9_release section
func TestConfigNoDuplicateFromReloadRoundTrip(t *testing.T) {
	t.Parallel()
	tempDir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, tempDir)

	if _, err := testutil.RunGitFlow(t, tempDir, "init", "--defaults"); err != nil {
		t.Fatalf("Failed to initialize git-flow: %v", err)
	}
	if _, err := testutil.RunGitFlow(t, tempDir, "config", "add", "base", "V9_Release"); err != nil {
		t.Fatalf("Failed to add base branch V9_Release: %v", err)
	}

	output, err := testutil.RunGitFlow(t, tempDir, "config", "add", "base", "V10_Release", "v9_release")
	if err != nil {
		t.Fatalf("Expected reload round-trip add to succeed, got error: %v\nOutput: %s", err, output)
	}

	cfg := gitflowBranchConfig(t, tempDir)
	assertContainsLine(t, cfg, "gitflow.branch.V10_Release.parent V9_Release", "reload round-trip")
	if !refExists(t, tempDir, "V10_Release") {
		t.Errorf("Expected refs/heads/V10_Release to exist")
	}
	assertContainsLine(t, cfg, "gitflow.branch.V9_Release.type base", "reload round-trip (V9 present)")
	assertNoLineContains(t, cfg, "gitflow.branch.v9_release.", "reload round-trip (#117 no duplicate)")
}

// TestConfigListShowsCanonicalCase tests that list shows canonical case and
// renders relationships in canonical case.
// Steps:
// 1. Sets up a test repository and initializes git-flow with defaults
// 2. Adds base V9_Release and base V10_Release with parent V9_Release
// 3. Runs 'config list'
// 4. Verifies output lists both names in original case
// 5. Verifies V10's relationship renders as 'V10_Release → V9_Release'
func TestConfigListShowsCanonicalCase(t *testing.T) {
	t.Parallel()
	tempDir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, tempDir)

	if _, err := testutil.RunGitFlow(t, tempDir, "init", "--defaults"); err != nil {
		t.Fatalf("Failed to initialize git-flow: %v", err)
	}
	if _, err := testutil.RunGitFlow(t, tempDir, "config", "add", "base", "V9_Release"); err != nil {
		t.Fatalf("Failed to add base branch V9_Release: %v", err)
	}
	if _, err := testutil.RunGitFlow(t, tempDir, "config", "add", "base", "V10_Release", "V9_Release"); err != nil {
		t.Fatalf("Failed to add base branch V10_Release: %v", err)
	}

	output, err := testutil.RunGitFlow(t, tempDir, "config", "list")
	if err != nil {
		t.Fatalf("Expected config list to succeed, got error: %v\nOutput: %s", err, output)
	}

	if !strings.Contains(output, "V9_Release") {
		t.Errorf("Expected list output to contain 'V9_Release', got:\n%s", output)
	}
	if !strings.Contains(output, "V10_Release → V9_Release") {
		t.Errorf("Expected list output to render 'V10_Release → V9_Release', got:\n%s", output)
	}
}

// TestConfigLowercaseNamesNoRegression tests that all-lowercase names still work
// end-to-end (add, edit, delete).
// Steps:
// 1. Sets up a test repository and initializes git-flow with defaults
// 2. Runs 'config add base staging main'
// 3. Runs 'config edit base staging --upstream-strategy rebase'
// 4. Runs 'config delete base staging'
// 5. Verifies each step succeeds; edit sets rebase; delete removes the section
func TestConfigLowercaseNamesNoRegression(t *testing.T) {
	t.Parallel()
	tempDir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, tempDir)

	if _, err := testutil.RunGitFlow(t, tempDir, "init", "--defaults"); err != nil {
		t.Fatalf("Failed to initialize git-flow: %v", err)
	}

	if _, err := testutil.RunGitFlow(t, tempDir, "config", "add", "base", "staging", "main"); err != nil {
		t.Fatalf("Expected add base staging to succeed, got error: %v", err)
	}

	if _, err := testutil.RunGitFlow(t, tempDir, "config", "edit", "base", "staging", "--upstream-strategy", "rebase"); err != nil {
		t.Fatalf("Expected edit base staging to succeed, got error: %v", err)
	}
	cfg := gitflowBranchConfig(t, tempDir)
	assertContainsLine(t, cfg, "gitflow.branch.staging.upstreamstrategy rebase", "lowercase no regression (edit)")

	if _, err := testutil.RunGitFlow(t, tempDir, "config", "delete", "base", "staging"); err != nil {
		t.Fatalf("Expected delete base staging to succeed, got error: %v", err)
	}
	cfg = gitflowBranchConfig(t, tempDir)
	assertNoLineContains(t, cfg, "gitflow.branch.staging.", "lowercase no regression (delete)")
}

// TestConfigInitDefaultsUnaffected tests that default init is unaffected by the
// case-insensitive changes.
// Steps:
// 1. Sets up a test repository
// 2. Runs 'git flow init --defaults'
// 3. Verifies standard main/develop/feature/release/hotfix config is produced
// 4. Verifies no lowercase-variant duplicate sections appear
func TestConfigInitDefaultsUnaffected(t *testing.T) {
	t.Parallel()
	tempDir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, tempDir)

	if _, err := testutil.RunGitFlow(t, tempDir, "init", "--defaults"); err != nil {
		t.Fatalf("Failed to initialize git-flow: %v", err)
	}

	cfg := gitflowBranchConfig(t, tempDir)
	assertContainsLine(t, cfg, "gitflow.branch.main.type base", "init defaults (main)")
	assertContainsLine(t, cfg, "gitflow.branch.develop.type base", "init defaults (develop)")
	assertContainsLine(t, cfg, "gitflow.branch.feature.type topic", "init defaults (feature)")
	assertContainsLine(t, cfg, "gitflow.branch.release.type topic", "init defaults (release)")
	assertContainsLine(t, cfg, "gitflow.branch.hotfix.type topic", "init defaults (hotfix)")
}

// TestConfigRenameTopicResolvesCaseInsensitively tests renaming a topic type by a
// case-variant name resolves to canonical and carries over settings.
// Steps:
// 1. Sets up a test repository and initializes git-flow with defaults
// 2. Adds topic QA_Feature (prefix qa/)
// 3. Runs 'config rename topic qa_feature QA_Regression'
// 4. Verifies QA_Feature section removed and QA_Regression present with prefix qa/
// 5. Verifies no qa_feature case-variant section remains
func TestConfigRenameTopicResolvesCaseInsensitively(t *testing.T) {
	t.Parallel()
	tempDir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, tempDir)

	if _, err := testutil.RunGitFlow(t, tempDir, "init", "--defaults"); err != nil {
		t.Fatalf("Failed to initialize git-flow: %v", err)
	}
	if _, err := testutil.RunGitFlow(t, tempDir, "config", "add", "topic", "QA_Feature", "develop", "--prefix", "qa/"); err != nil {
		t.Fatalf("Failed to add topic QA_Feature: %v", err)
	}

	output, err := testutil.RunGitFlow(t, tempDir, "config", "rename", "topic", "qa_feature", "QA_Regression")
	if err != nil {
		t.Fatalf("Expected case-insensitive topic rename to succeed, got error: %v\nOutput: %s", err, output)
	}

	cfg := gitflowBranchConfig(t, tempDir)
	assertNoLineContains(t, cfg, "gitflow.branch.QA_Feature.", "rename topic resolves")
	assertNoLineContains(t, cfg, "gitflow.branch.qa_feature.", "rename topic resolves (no variant)")
	assertContainsLine(t, cfg, "gitflow.branch.QA_Regression.prefix qa/", "rename topic carries prefix")
}

// TestConfigDeleteTopicResolvesCaseInsensitively tests deleting a topic type by a
// case-variant name removes the canonical section.
// Steps:
// 1. Sets up a test repository and initializes git-flow with defaults
// 2. Adds topic QA_Feature (prefix qa/)
// 3. Runs 'config delete topic qa_feature'
// 4. Verifies the QA_Feature section is fully removed (no QA_Feature or qa_feature lines)
func TestConfigDeleteTopicResolvesCaseInsensitively(t *testing.T) {
	t.Parallel()
	tempDir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, tempDir)

	if _, err := testutil.RunGitFlow(t, tempDir, "init", "--defaults"); err != nil {
		t.Fatalf("Failed to initialize git-flow: %v", err)
	}
	if _, err := testutil.RunGitFlow(t, tempDir, "config", "add", "topic", "QA_Feature", "develop", "--prefix", "qa/"); err != nil {
		t.Fatalf("Failed to add topic QA_Feature: %v", err)
	}

	output, err := testutil.RunGitFlow(t, tempDir, "config", "delete", "topic", "qa_feature")
	if err != nil {
		t.Fatalf("Expected case-insensitive topic delete to succeed, got error: %v\nOutput: %s", err, output)
	}

	cfg := gitflowBranchConfig(t, tempDir)
	assertNoLineContains(t, cfg, "gitflow.branch.QA_Feature.", "delete topic resolves")
	assertNoLineContains(t, cfg, "gitflow.branch.qa_feature.", "delete topic resolves")
}

// TestConfigEditTopicStartingPointResolvesCaseInsensitively tests that editing a
// topic --starting-point resolves the value case-insensitively to canonical.
// Steps:
// 1. Sets up a test repository and initializes git-flow with defaults
// 2. Adds base V9_Release and topic QA_Feature (prefix qa/)
// 3. Runs 'config edit topic QA_Feature --starting-point v9_release'
// 4. Verifies stored start point is canonical V9_Release, no v9_release section
func TestConfigEditTopicStartingPointResolvesCaseInsensitively(t *testing.T) {
	t.Parallel()
	tempDir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, tempDir)

	if _, err := testutil.RunGitFlow(t, tempDir, "init", "--defaults"); err != nil {
		t.Fatalf("Failed to initialize git-flow: %v", err)
	}
	if _, err := testutil.RunGitFlow(t, tempDir, "config", "add", "base", "V9_Release"); err != nil {
		t.Fatalf("Failed to add base branch V9_Release: %v", err)
	}
	if _, err := testutil.RunGitFlow(t, tempDir, "config", "add", "topic", "QA_Feature", "develop", "--prefix", "qa/"); err != nil {
		t.Fatalf("Failed to add topic QA_Feature: %v", err)
	}

	output, err := testutil.RunGitFlow(t, tempDir, "config", "edit", "topic", "QA_Feature", "--starting-point", "v9_release")
	if err != nil {
		t.Fatalf("Expected case-insensitive edit topic starting-point to succeed, got error: %v\nOutput: %s", err, output)
	}

	cfg := gitflowBranchConfig(t, tempDir)
	assertContainsLine(t, cfg, "gitflow.branch.QA_Feature.startpoint V9_Release", "edit topic starting point")
	assertNoLineContains(t, cfg, "gitflow.branch.v9_release.", "edit topic starting point")
}

func setupConfigAddBaseMissingParentRef(t *testing.T) string {
	t.Helper()
	tempDir := testutil.SetupTestRepo(t)
	t.Cleanup(func() { testutil.CleanupTestRepo(t, tempDir) })

	if _, err := testutil.RunGitFlow(t, tempDir, "init", "--defaults"); err != nil {
		t.Fatalf("Failed to initialize git-flow: %v", err)
	}
	if _, err := testutil.RunGit(t, tempDir, "checkout", "main"); err != nil {
		t.Fatalf("Failed to check out main: %v", err)
	}
	if _, err := testutil.RunGit(t, tempDir, "branch", "-D", "develop"); err != nil {
		t.Fatalf("Failed to delete develop: %v", err)
	}

	return tempDir
}

// TestConfigAddBaseRollsBackConfigWhenBranchCreationFails tests that a failed
// branch creation removes the saved base configuration.
// Steps:
// 1. Initializes git-flow and removes the configured develop ref
// 2. Attempts to add qa with the missing develop ref
// 3. Verifies branch creation fails and the qa configuration is absent
// 4. Verifies config list does not show qa
func TestConfigAddBaseRollsBackConfigWhenBranchCreationFails(t *testing.T) {
	t.Parallel()
	tempDir := setupConfigAddBaseMissingParentRef(t)

	output, err := testutil.RunGitFlow(t, tempDir, "config", "add", "base", "qa", "develop")
	if err == nil {
		t.Fatalf("Expected branch creation to fail, got success\nOutput: %s", output)
	}
	if !strings.Contains(output, "create branch 'qa'") {
		t.Errorf("Expected branch creation error, got:\n%s", output)
	}

	cfg := gitflowBranchConfig(t, tempDir)
	assertNoLineContains(t, cfg, "gitflow.branch.qa.", "failed add rollback")

	listOutput, err := testutil.RunGitFlow(t, tempDir, "config", "list")
	if err != nil {
		t.Fatalf("Expected config list to succeed, got error: %v\nOutput: %s", err, listOutput)
	}
	assertNoLineContains(t, listOutput, "  qa ", "failed add rollback")
}

// TestConfigAddBaseCanRetryAfterBranchCreationFails tests that a failed base
// addition can be retried after restoring the missing parent ref.
// Steps:
// 1. Initializes git-flow with the configured develop ref removed
// 2. Verifies adding qa fails while develop is missing
// 3. Restores develop and retries the add
// 4. Verifies the qa configuration and ref are created
func TestConfigAddBaseCanRetryAfterBranchCreationFails(t *testing.T) {
	t.Parallel()
	tempDir := setupConfigAddBaseMissingParentRef(t)

	if output, err := testutil.RunGitFlow(t, tempDir, "config", "add", "base", "qa", "develop"); err == nil {
		t.Fatalf("Expected initial branch creation to fail, got success\nOutput: %s", output)
	}
	if _, err := testutil.RunGit(t, tempDir, "branch", "develop", "main"); err != nil {
		t.Fatalf("Failed to recreate develop: %v", err)
	}

	output, err := testutil.RunGitFlow(t, tempDir, "config", "add", "base", "qa", "develop")
	if err != nil {
		t.Fatalf("Expected retry to succeed, got error: %v\nOutput: %s", err, output)
	}

	cfg := gitflowBranchConfig(t, tempDir)
	assertContainsLine(t, cfg, "gitflow.branch.qa.type base", "retry after failed add")
	assertContainsLine(t, cfg, "gitflow.branch.qa.parent develop", "retry after failed add")
	if !refExists(t, tempDir, "qa") {
		t.Error("Expected refs/heads/qa to exist after retry")
	}
}

// TestConfigAddBaseUsesExistingRef tests that an existing branch bypasses
// branch creation while saving the base configuration.
// Steps:
// 1. Sets up a test repository and initializes git-flow with defaults
// 2. Creates a plain qa branch from main
// 3. Adds qa as a base branch with main as its parent
// 4. Verifies no branch is created and the base configuration is saved
func TestConfigAddBaseUsesExistingRef(t *testing.T) {
	t.Parallel()
	tempDir := testutil.SetupTestRepo(t)
	t.Cleanup(func() { testutil.CleanupTestRepo(t, tempDir) })

	if _, err := testutil.RunGitFlow(t, tempDir, "init", "--defaults"); err != nil {
		t.Fatalf("Failed to initialize git-flow: %v", err)
	}
	if _, err := testutil.RunGit(t, tempDir, "branch", "qa", "main"); err != nil {
		t.Fatalf("Failed to create qa branch: %v", err)
	}

	output, err := testutil.RunGitFlow(t, tempDir, "config", "add", "base", "qa", "main")
	if err != nil {
		t.Fatalf("Expected add with existing ref to succeed, got error: %v\nOutput: %s", err, output)
	}
	if strings.Contains(output, "Created branch 'qa'") {
		t.Errorf("Expected existing qa ref to skip branch creation, got:\n%s", output)
	}

	cfg := gitflowBranchConfig(t, tempDir)
	assertContainsLine(t, cfg, "gitflow.branch.qa.type base", "existing ref add")
}

// TestConfigEditTopicTagFalsePersists covers scenario 1: an explicit --tag=false
// on 'config edit topic' must be persisted, clearing a stored true.
// Steps:
// 1. Sets up a test repository and initializes git-flow with defaults (release.tag=true)
// 2. Runs 'config edit topic release --tag=false'
// 3. Verifies local gitflow.branch.release.tag is "false"
func TestConfigEditTopicTagFalsePersists(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	if out, err := testutil.RunGitFlow(t, dir, "init", "--defaults"); err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, out)
	}

	out, err := testutil.RunGitFlow(t, dir, "config", "edit", "topic", "release", "--tag=false")
	if err != nil {
		t.Fatalf("config edit topic --tag=false failed: %v\nOutput: %s", err, out)
	}

	if v := testutil.GitConfigValue(t, dir, "gitflow.branch.release.tag"); v != "false" {
		t.Errorf("Expected local release.tag=false, got %q\nOutput: %s", v, out)
	}
}

// TestConfigEditTopicTagTruePersists covers scenario 2: an explicit --tag=true on
// 'config edit topic' must be persisted, setting a stored false.
// Steps:
// 1. Sets up a test repository and initializes git-flow with defaults (feature does not tag)
// 2. Runs 'config edit topic feature --tag=true'
// 3. Verifies local gitflow.branch.feature.tag is "true"
func TestConfigEditTopicTagTruePersists(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	if out, err := testutil.RunGitFlow(t, dir, "init", "--defaults"); err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, out)
	}

	out, err := testutil.RunGitFlow(t, dir, "config", "edit", "topic", "feature", "--tag=true")
	if err != nil {
		t.Fatalf("config edit topic --tag=true failed: %v\nOutput: %s", err, out)
	}

	if v := testutil.GitConfigValue(t, dir, "gitflow.branch.feature.tag"); v != "true" {
		t.Errorf("Expected local feature.tag=true, got %q\nOutput: %s", v, out)
	}
}

// TestConfigEditTopicPreservesTagWhenFlagOmitted covers scenario 3: an omitted
// --tag on 'config edit topic' must preserve the stored value.
// Steps:
// 1. Sets up a test repository and initializes git-flow with defaults (release.tag=true)
// 2. Runs 'config edit topic release --prefix=rel/' without --tag
// 3. Verifies local release.prefix is "rel/" and release.tag is still "true"
func TestConfigEditTopicPreservesTagWhenFlagOmitted(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	if out, err := testutil.RunGitFlow(t, dir, "init", "--defaults"); err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, out)
	}

	out, err := testutil.RunGitFlow(t, dir, "config", "edit", "topic", "release", "--prefix=rel/")
	if err != nil {
		t.Fatalf("config edit topic --prefix=rel/ failed: %v\nOutput: %s", err, out)
	}

	if v := testutil.GitConfigValue(t, dir, "gitflow.branch.release.prefix"); v != "rel/" {
		t.Errorf("Expected local release.prefix=rel/, got %q\nOutput: %s", v, out)
	}
	if v := testutil.GitConfigValue(t, dir, "gitflow.branch.release.tag"); v != "true" {
		t.Errorf("Expected omitted --tag to preserve release.tag=true, got %q\nOutput: %s", v, out)
	}
}

// TestConfigEditBasePreservesAutoUpdateWhenFlagOmitted covers scenario 4: an
// omitted --auto-update on 'config edit base' must preserve the stored value.
// Steps:
// 1. Sets up a test repository and initializes git-flow with defaults (develop.autoUpdate=true)
// 2. Runs 'config edit base develop --upstream-strategy=merge' without --auto-update
// 3. Verifies local develop.upstreamStrategy is "merge" and develop.autoUpdate is still "true"
func TestConfigEditBasePreservesAutoUpdateWhenFlagOmitted(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	if out, err := testutil.RunGitFlow(t, dir, "init", "--defaults"); err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, out)
	}

	out, err := testutil.RunGitFlow(t, dir, "config", "edit", "base", "develop", "--upstream-strategy=merge")
	if err != nil {
		t.Fatalf("config edit base --upstream-strategy=merge failed: %v\nOutput: %s", err, out)
	}

	if v := testutil.GitConfigValue(t, dir, "gitflow.branch.develop.upstreamStrategy"); v != "merge" {
		t.Errorf("Expected local develop.upstreamStrategy=merge, got %q\nOutput: %s", v, out)
	}
	if v := testutil.GitConfigValue(t, dir, "gitflow.branch.develop.autoUpdate"); v != "true" {
		t.Errorf("Expected omitted --auto-update to preserve develop.autoUpdate=true, got %q\nOutput: %s", v, out)
	}
}

// TestConfigEditBaseAutoUpdateFalsePersists covers scenario 5: an explicit
// --auto-update=false on 'config edit base' must still win over the stored true.
// Steps:
// 1. Sets up a test repository and initializes git-flow with defaults (develop.autoUpdate=true)
// 2. Runs 'config edit base develop --auto-update=false'
// 3. Verifies local develop.autoUpdate is "false"
func TestConfigEditBaseAutoUpdateFalsePersists(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	if out, err := testutil.RunGitFlow(t, dir, "init", "--defaults"); err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, out)
	}

	out, err := testutil.RunGitFlow(t, dir, "config", "edit", "base", "develop", "--auto-update=false")
	if err != nil {
		t.Fatalf("config edit base --auto-update=false failed: %v\nOutput: %s", err, out)
	}

	if v := testutil.GitConfigValue(t, dir, "gitflow.branch.develop.autoUpdate"); v != "false" {
		t.Errorf("Expected local develop.autoUpdate=false, got %q\nOutput: %s", v, out)
	}
}

// TestConfigEditBaseAutoUpdateTrueOverridesAddedFalse covers scenario 6: an edit
// can flip a boolean added as false back to true.
// Steps:
// 1. Sets up a test repository and initializes git-flow with defaults
// 2. Runs 'config add base staging main --auto-update=false'
// 3. Runs 'config edit base staging --auto-update=true'
// 4. Verifies local staging.autoUpdate is "true"
func TestConfigEditBaseAutoUpdateTrueOverridesAddedFalse(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	if out, err := testutil.RunGitFlow(t, dir, "init", "--defaults"); err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, out)
	}
	if out, err := testutil.RunGitFlow(t, dir, "config", "add", "base", "staging", "main", "--auto-update=false"); err != nil {
		t.Fatalf("config add base staging failed: %v\nOutput: %s", err, out)
	}

	out, err := testutil.RunGitFlow(t, dir, "config", "edit", "base", "staging", "--auto-update=true")
	if err != nil {
		t.Fatalf("config edit base --auto-update=true failed: %v\nOutput: %s", err, out)
	}

	if v := testutil.GitConfigValue(t, dir, "gitflow.branch.staging.autoUpdate"); v != "true" {
		t.Errorf("Expected local staging.autoUpdate=true, got %q\nOutput: %s", v, out)
	}
}

// TestConfigEditTopicTagFalseShadowsGlobalTrue covers scenario 7: an explicit
// local false must shadow a true inherited from the global config scope.
// Steps:
// 1. Isolates the global and system git config through the subprocess env
// 2. Sets up a test repository and initializes git-flow with defaults
// 3. Unsets the local release.tag key and sets gitflow.branch.release.tag=true globally
// 4. Runs 'config edit topic release --tag=false'
// 5. Verifies the local read is "false" and the merged read is "false" too
func TestConfigEditTopicTagFalseShadowsGlobalTrue(t *testing.T) {
	t.Parallel()
	// Isolate global AND system config through the subprocess env so no ambient
	// gitflow.* key can leak in, and nothing touches the developer's real config.
	env := []string{
		"GIT_CONFIG_GLOBAL=" + filepath.Join(t.TempDir(), "global-gitconfig"),
		"GIT_CONFIG_SYSTEM=" + os.DevNull,
	}

	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	if out, err := runGitFlowWithEnv(t, dir, env, "init", "--defaults"); err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, out)
	}
	if out, err := testutil.RunGitWithEnv(t, dir, env, "config", "--local", "--unset", "gitflow.branch.release.tag"); err != nil {
		t.Fatalf("Failed to unset local release.tag: %v\nOutput: %s", err, out)
	}
	if out, err := testutil.RunGitWithEnv(t, dir, env, "config", "--global", "gitflow.branch.release.tag", "true"); err != nil {
		t.Fatalf("Failed to set global release.tag: %v\nOutput: %s", err, out)
	}

	out, err := runGitFlowWithEnv(t, dir, env, "config", "edit", "topic", "release", "--tag=false")
	if err != nil {
		t.Fatalf("config edit topic --tag=false failed: %v\nOutput: %s", err, out)
	}

	local, err := testutil.RunGitWithEnv(t, dir, env, "config", "--local", "--get", "gitflow.branch.release.tag")
	if err != nil {
		t.Fatalf("Failed to read local release.tag: %v\nOutput: %s", err, local)
	}
	if v := strings.TrimSpace(local); v != "false" {
		t.Errorf("Expected local release.tag=false, got %q\nOutput: %s", v, out)
	}

	merged, err := testutil.RunGitWithEnv(t, dir, env, "config", "--get", "gitflow.branch.release.tag")
	if err != nil {
		t.Fatalf("Failed to read merged release.tag: %v\nOutput: %s", err, merged)
	}
	if v := strings.TrimSpace(merged); v != "false" {
		t.Errorf("Expected the local false to shadow the global true, got merged %q\nOutput: %s", v, out)
	}
}

// TestConfigAddTopicWritesExplicitTagFalse covers scenario 12: 'config add' still
// treats an omitted boolean as false, and now writes that false explicitly.
// Steps:
// 1. Sets up a test repository and initializes git-flow with defaults
// 2. Runs 'config add topic docs develop' without --tag
// 3. Verifies local gitflow.branch.docs.tag is the explicit string "false"
func TestConfigAddTopicWritesExplicitTagFalse(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	if out, err := testutil.RunGitFlow(t, dir, "init", "--defaults"); err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, out)
	}

	out, err := testutil.RunGitFlow(t, dir, "config", "add", "topic", "docs", "develop")
	if err != nil {
		t.Fatalf("config add topic docs failed: %v\nOutput: %s", err, out)
	}

	if v := testutil.GitConfigValue(t, dir, "gitflow.branch.docs.tag"); v != "false" {
		t.Errorf("Expected local docs.tag=false written explicitly, got %q\nOutput: %s", v, out)
	}
}

// TestConfigEditTopicBareTagFlagSetsTrue covers scenario 13: the bare --tag form
// (no =value) counts as a supplied true.
// Steps:
// 1. Sets up a test repository and initializes git-flow with defaults (feature does not tag)
// 2. Runs 'config edit topic feature --tag'
// 3. Verifies local gitflow.branch.feature.tag is "true"
func TestConfigEditTopicBareTagFlagSetsTrue(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	if out, err := testutil.RunGitFlow(t, dir, "init", "--defaults"); err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, out)
	}

	out, err := testutil.RunGitFlow(t, dir, "config", "edit", "topic", "feature", "--tag")
	if err != nil {
		t.Fatalf("config edit topic --tag failed: %v\nOutput: %s", err, out)
	}

	if v := testutil.GitConfigValue(t, dir, "gitflow.branch.feature.tag"); v != "true" {
		t.Errorf("Expected bare --tag to set feature.tag=true, got %q\nOutput: %s", v, out)
	}
}

// TestConfigEditTopicPreservesOtherBranchTypeTags covers scenario 14: an edit
// targeting one branch type must leave every other branch type's tag intact.
// The writer rewrites all branches on every save, so the blast radius reaches
// well beyond the edited branch.
// Steps:
// 1. Sets up a test repository and initializes git-flow with defaults
// 2. Runs 'config edit topic feature --prefix=feat/'
// 3. Verifies local release.tag and hotfix.tag are "true"
// 4. Verifies local feature.tag, main.tag and develop.tag are "false"
func TestConfigEditTopicPreservesOtherBranchTypeTags(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	if out, err := testutil.RunGitFlow(t, dir, "init", "--defaults"); err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, out)
	}

	out, err := testutil.RunGitFlow(t, dir, "config", "edit", "topic", "feature", "--prefix=feat/")
	if err != nil {
		t.Fatalf("config edit topic --prefix=feat/ failed: %v\nOutput: %s", err, out)
	}

	for _, branch := range []string{"release", "hotfix"} {
		if v := testutil.GitConfigValue(t, dir, "gitflow.branch."+branch+".tag"); v != "true" {
			t.Errorf("Expected local %s.tag=true after an unrelated edit, got %q\nOutput: %s", branch, v, out)
		}
	}
	for _, branch := range []string{"feature", "main", "develop"} {
		if v := testutil.GitConfigValue(t, dir, "gitflow.branch."+branch+".tag"); v != "false" {
			t.Errorf("Expected local %s.tag=false after an unrelated edit, got %q\nOutput: %s", branch, v, out)
		}
	}
}

// mustOpenRepo opens a git.Repo handle for dir, failing the test on error.
func mustOpenRepo(t *testing.T, dir string) *git.Repo {
	t.Helper()
	repo, err := git.Open(dir)
	if err != nil {
		t.Fatalf("git.Open(%q) failed: %v", dir, err)
	}
	return repo
}
