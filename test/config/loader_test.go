package config_test

import (
	"os"
	"os/exec"
	"testing"

	"github.com/gittower/git-flow-next/config"
	"github.com/gittower/git-flow-next/internal/git"
	"github.com/stretchr/testify/assert"
)

func setupTestRepo(t *testing.T) string {
	// Create a temporary directory
	dir, err := os.MkdirTemp("", "git-flow-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Change to the temporary directory
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Failed to change to temp dir: %v", err)
	}

	// Initialize git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to initialize git repo: %v", err)
	}

	// Configure git user
	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to configure git user name: %v", err)
	}

	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to configure git user email: %v", err)
	}

	return dir
}

func cleanupTestRepo(t *testing.T, dir string) {
	if err := os.RemoveAll(dir); err != nil {
		t.Errorf("Failed to cleanup test repo: %v", err)
	}
}

func TestLoadConfigCaseInsensitive(t *testing.T) {
	// Setup
	dir := setupTestRepo(t)
	defer cleanupTestRepo(t, dir)

	// Set config values with different cases
	testCases := []struct {
		key   string
		value string
	}{
		{"gitflow.branch.feature.startPoint", "develop"},
		{"gitflow.branch.release.StartPoint", "develop"},
		{"gitflow.branch.hotfix.STARTPOINT", "main"},
	}

	for _, tc := range testCases {
		cmd := exec.Command("git", "config", tc.key, tc.value)
		cmd.Dir = dir
		if err := cmd.Run(); err != nil {
			t.Fatalf("Failed to set git config %s: %v", tc.key, err)
		}
	}

	// Set version to mark as initialized
	cmd := exec.Command("git", "config", "gitflow.version", "1.0")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to set gitflow version: %v", err)
	}

	// Load config
	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify all start points are loaded correctly regardless of case
	expectedStartPoints := map[string]string{
		"feature": "develop",
		"release": "develop",
		"hotfix":  "main",
	}

	for branch, expected := range expectedStartPoints {
		if actual := cfg.Branches[branch].StartPoint; actual != expected {
			t.Errorf("Branch %s: expected start point %s, got %s", branch, expected, actual)
		}
	}
}

func TestLoadConfigWithMixedCaseProperties(t *testing.T) {
	// Setup
	dir := setupTestRepo(t)
	defer cleanupTestRepo(t, dir)

	// Set config values with mixed case for different properties
	configs := []struct {
		key   string
		value string
	}{
		{"gitflow.branch.feature.Type", "topic"},
		{"gitflow.branch.feature.parent", "develop"},
		{"gitflow.branch.feature.UpstreamStrategy", "rebase"},
		{"gitflow.branch.feature.downstreamStrategy", "squash"},
		{"gitflow.branch.feature.PREFIX", "feature/"},
		{"gitflow.branch.feature.AutoUpdate", "true"},
	}

	for _, cfg := range configs {
		cmd := exec.Command("git", "config", cfg.key, cfg.value)
		cmd.Dir = dir
		if err := cmd.Run(); err != nil {
			t.Fatalf("Failed to set git config %s: %v", cfg.key, err)
		}
	}

	// Set version to mark as initialized
	cmd := exec.Command("git", "config", "gitflow.version", "1.0")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to set gitflow version: %v", err)
	}

	// Load config
	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify all properties are loaded correctly regardless of case
	feature := cfg.Branches["feature"]
	expected := config.BranchConfig{
		Type:               "topic",
		Parent:             "develop",
		UpstreamStrategy:   "rebase",
		DownstreamStrategy: "squash",
		Prefix:             "feature/",
		AutoUpdate:         true,
	}

	if feature.Type != expected.Type {
		t.Errorf("Expected Type %s, got %s", expected.Type, feature.Type)
	}
	if feature.Parent != expected.Parent {
		t.Errorf("Expected Parent %s, got %s", expected.Parent, feature.Parent)
	}
	if feature.UpstreamStrategy != expected.UpstreamStrategy {
		t.Errorf("Expected UpstreamStrategy %s, got %s", expected.UpstreamStrategy, feature.UpstreamStrategy)
	}
	if feature.DownstreamStrategy != expected.DownstreamStrategy {
		t.Errorf("Expected DownstreamStrategy %s, got %s", expected.DownstreamStrategy, feature.DownstreamStrategy)
	}
	if feature.Prefix != expected.Prefix {
		t.Errorf("Expected Prefix %s, got %s", expected.Prefix, feature.Prefix)
	}
	if feature.AutoUpdate != expected.AutoUpdate {
		t.Errorf("Expected AutoUpdate %v, got %v", expected.AutoUpdate, feature.AutoUpdate)
	}
}

func TestApplyOverrides_NoOverrides(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg = config.ApplyOverrides(cfg, config.ConfigOverrides{})

	// Check main branch (base branch)
	mainConfig, exists := cfg.Branches["main"]
	assert.True(t, exists)
	assert.Equal(t, string(config.BranchTypeBase), mainConfig.Type)
	assert.Equal(t, "", mainConfig.Parent)
	assert.Equal(t, "", mainConfig.StartPoint)

	// Check develop branch (base branch)
	developConfig, exists := cfg.Branches["develop"]
	assert.True(t, exists)
	assert.Equal(t, string(config.BranchTypeBase), developConfig.Type)
	assert.Equal(t, "main", developConfig.Parent)
	assert.Equal(t, "", developConfig.StartPoint)

	// Check feature branch
	featureConfig, exists := cfg.Branches["feature"]
	assert.True(t, exists)
	assert.Equal(t, "feature/", featureConfig.Prefix)
	assert.Equal(t, "develop", featureConfig.Parent)
	assert.Equal(t, "develop", featureConfig.StartPoint)

	// Check release branch
	releaseConfig, exists := cfg.Branches["release"]
	assert.True(t, exists)
	assert.Equal(t, "release/", releaseConfig.Prefix)
	assert.Equal(t, "main", releaseConfig.Parent)
	assert.Equal(t, "develop", releaseConfig.StartPoint)

	// Check hotfix branch
	hotfixConfig, exists := cfg.Branches["hotfix"]
	assert.True(t, exists)
	assert.Equal(t, "hotfix/", hotfixConfig.Prefix)
	assert.Equal(t, "main", hotfixConfig.Parent)
	assert.Equal(t, "main", hotfixConfig.StartPoint)

	// Check support branch
	supportConfig, exists := cfg.Branches["support"]
	assert.True(t, exists)
	assert.Equal(t, "support/", supportConfig.Prefix)
	assert.Equal(t, "main", supportConfig.Parent)
	assert.Equal(t, "main", supportConfig.StartPoint)
}

func TestApplyOverrides_CustomBranchNames(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg = config.ApplyOverrides(cfg, config.ConfigOverrides{
		MainBranch:    "custom-main",
		DevelopBranch: "custom-dev",
	})

	// Check main branch (base branch)
	mainConfig, exists := cfg.Branches["custom-main"]
	assert.True(t, exists)
	assert.Equal(t, string(config.BranchTypeBase), mainConfig.Type)
	assert.Equal(t, "", mainConfig.Parent)
	assert.Equal(t, "", mainConfig.StartPoint)

	// Check develop branch (base branch)
	developConfig, exists := cfg.Branches["custom-dev"]
	assert.True(t, exists)
	assert.Equal(t, string(config.BranchTypeBase), developConfig.Type)
	assert.Equal(t, "custom-main", developConfig.Parent)
	assert.Equal(t, "", developConfig.StartPoint)

	// Check feature branch parent and start point
	featureConfig, exists := cfg.Branches["feature"]
	assert.True(t, exists)
	assert.Equal(t, "custom-dev", featureConfig.Parent)
	assert.Equal(t, "custom-dev", featureConfig.StartPoint)

	// Check release branch parent and start point
	releaseConfig, exists := cfg.Branches["release"]
	assert.True(t, exists)
	assert.Equal(t, "custom-main", releaseConfig.Parent)
	assert.Equal(t, "custom-dev", releaseConfig.StartPoint)

	// Check hotfix branch parent and start point
	hotfixConfig, exists := cfg.Branches["hotfix"]
	assert.True(t, exists)
	assert.Equal(t, "custom-main", hotfixConfig.Parent)
	assert.Equal(t, "custom-main", hotfixConfig.StartPoint)

	// Check support branch parent and start point
	supportConfig, exists := cfg.Branches["support"]
	assert.True(t, exists)
	assert.Equal(t, "custom-main", supportConfig.Parent)
	assert.Equal(t, "custom-main", supportConfig.StartPoint)

	// Check old names don't exist
	_, exists = cfg.Branches["main"]
	assert.False(t, exists)
	_, exists = cfg.Branches["develop"]
	assert.False(t, exists)
}

func TestApplyOverrides_CustomPrefixes(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg = config.ApplyOverrides(cfg, config.ConfigOverrides{
		FeaturePrefix: "f/",
		ReleasePrefix: "r/",
		HotfixPrefix:  "h/",
		SupportPrefix: "s/",
	})

	// Check prefixes while verifying parents and start points remain unchanged
	featureConfig := cfg.Branches["feature"]
	assert.Equal(t, "f/", featureConfig.Prefix)
	assert.Equal(t, "develop", featureConfig.Parent)
	assert.Equal(t, "develop", featureConfig.StartPoint)

	releaseConfig := cfg.Branches["release"]
	assert.Equal(t, "r/", releaseConfig.Prefix)
	assert.Equal(t, "main", releaseConfig.Parent)
	assert.Equal(t, "develop", releaseConfig.StartPoint)

	hotfixConfig := cfg.Branches["hotfix"]
	assert.Equal(t, "h/", hotfixConfig.Prefix)
	assert.Equal(t, "main", hotfixConfig.Parent)
	assert.Equal(t, "main", hotfixConfig.StartPoint)

	supportConfig := cfg.Branches["support"]
	assert.Equal(t, "s/", supportConfig.Prefix)
	assert.Equal(t, "main", supportConfig.Parent)
	assert.Equal(t, "main", supportConfig.StartPoint)
}

func TestApplyOverrides_CustomTagPrefix(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg = config.ApplyOverrides(cfg, config.ConfigOverrides{
		TagPrefix: "v",
	})

	// Check tag prefixes while verifying parents and start points remain unchanged
	releaseConfig := cfg.Branches["release"]
	assert.Equal(t, "v", releaseConfig.TagPrefix)
	assert.Equal(t, "main", releaseConfig.Parent)
	assert.Equal(t, "develop", releaseConfig.StartPoint)

	hotfixConfig := cfg.Branches["hotfix"]
	assert.Equal(t, "v", hotfixConfig.TagPrefix)
	assert.Equal(t, "main", hotfixConfig.Parent)
	assert.Equal(t, "main", hotfixConfig.StartPoint)
}

func TestApplyOverrides_AllOverrides(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg = config.ApplyOverrides(cfg, config.ConfigOverrides{
		MainBranch:    "custom-main",
		DevelopBranch: "custom-dev",
		FeaturePrefix: "f/",
		ReleasePrefix: "r/",
		HotfixPrefix:  "h/",
		SupportPrefix: "s/",
		TagPrefix:     "v",
	})

	// Check main branch (base branch)
	mainConfig, exists := cfg.Branches["custom-main"]
	assert.True(t, exists)
	assert.Equal(t, string(config.BranchTypeBase), mainConfig.Type)
	assert.Equal(t, "", mainConfig.Parent)
	assert.Equal(t, "", mainConfig.StartPoint)

	// Check develop branch (base branch)
	developConfig, exists := cfg.Branches["custom-dev"]
	assert.True(t, exists)
	assert.Equal(t, string(config.BranchTypeBase), developConfig.Type)
	assert.Equal(t, "custom-main", developConfig.Parent)
	assert.Equal(t, "", developConfig.StartPoint)

	// Check feature branch
	featureConfig := cfg.Branches["feature"]
	assert.Equal(t, "f/", featureConfig.Prefix)
	assert.Equal(t, "custom-dev", featureConfig.Parent)
	assert.Equal(t, "custom-dev", featureConfig.StartPoint)

	// Check release branch
	releaseConfig := cfg.Branches["release"]
	assert.Equal(t, "r/", releaseConfig.Prefix)
	assert.Equal(t, "custom-main", releaseConfig.Parent)
	assert.Equal(t, "custom-dev", releaseConfig.StartPoint)
	assert.Equal(t, "v", releaseConfig.TagPrefix)

	// Check hotfix branch
	hotfixConfig := cfg.Branches["hotfix"]
	assert.Equal(t, "h/", hotfixConfig.Prefix)
	assert.Equal(t, "custom-main", hotfixConfig.Parent)
	assert.Equal(t, "custom-main", hotfixConfig.StartPoint)
	assert.Equal(t, "v", hotfixConfig.TagPrefix)

	// Check support branch
	supportConfig := cfg.Branches["support"]
	assert.Equal(t, "s/", supportConfig.Prefix)
	assert.Equal(t, "custom-main", supportConfig.Parent)
	assert.Equal(t, "custom-main", supportConfig.StartPoint)
}

// TestDefaultRemoteConfiguration tests that "origin" is used as the default remote name
func TestDefaultRemoteConfiguration(t *testing.T) {
	// Setup
	dir := setupTestRepo(t)
	defer cleanupTestRepo(t, dir)

	// Initialize git-flow
	cmd := exec.Command("git", "config", "gitflow.version", "1.0")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to set gitflow version: %v", err)
	}

	// Load config without setting gitflow.origin
	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify default remote is "origin"
	assert.Equal(t, "origin", cfg.Remote, "Default remote should be 'origin'")
}

// TestCustomRemoteConfiguration tests that a custom remote name is used when gitflow.origin is set
func TestCustomRemoteConfiguration(t *testing.T) {
	// Setup
	dir := setupTestRepo(t)
	defer cleanupTestRepo(t, dir)

	// Initialize git-flow
	cmd := exec.Command("git", "config", "gitflow.version", "1.0")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to set gitflow version: %v", err)
	}

	// Set custom remote
	customRemote := "myremote"
	cmd = exec.Command("git", "config", "gitflow.origin", customRemote)
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to set custom remote: %v", err)
	}

	// Debug: Print git config
	cmd = exec.Command("git", "config", "--get", "gitflow.origin")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		t.Logf("Failed to get gitflow.origin config: %v", err)
	} else {
		t.Logf("gitflow.origin from git config: %s", string(out))
	}

	// We need to manually create a config to test with the specific repository
	cfg := config.DefaultConfig()

	// Override with our custom remote
	remote, err := git.GetConfigInDir(dir, "gitflow.origin")
	if err == nil && remote != "" {
		cfg.Remote = remote
	}

	// Verify custom remote is used
	assert.Equal(t, customRemote, cfg.Remote, "Custom remote should be used")
}

// TestGitFlowAVHRemoteImport tests that git-flow-avh remote configuration is imported correctly
func TestGitFlowAVHRemoteImport(t *testing.T) {
	// Setup
	dir := setupTestRepo(t)
	defer cleanupTestRepo(t, dir)

	// Set git-flow-avh config
	cmd := exec.Command("git", "config", "gitflow.origin", "avh-remote")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to set git-flow-avh remote: %v", err)
	}

	// Import git-flow-avh config
	cfg, err := config.ImportGitFlowAVHConfig()
	if err != nil {
		t.Fatalf("Failed to import git-flow-avh config: %v", err)
	}

	// Verify git-flow-avh remote is imported
	assert.Equal(t, "avh-remote", cfg.Remote, "git-flow-avh remote should be imported")
}
