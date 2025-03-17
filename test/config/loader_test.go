package config_test

import (
	"os"
	"os/exec"
	"testing"

	"github.com/gittower/git-flow-next/config"
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
