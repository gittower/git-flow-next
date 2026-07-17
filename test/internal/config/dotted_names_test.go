package config_test

import (
	"os/exec"
	"testing"

	"github.com/gittower/git-flow-next/internal/config"
)

// setDottedConfig writes the given git config key/value pairs and marks the
// repo as git-flow-initialized so LoadConfig actually parses the branch keys
// instead of returning DefaultConfig().
func setDottedConfig(t *testing.T, dir string, kv map[string]string) {
	t.Helper()
	for key, value := range kv {
		cmd := exec.Command("git", "config", key, value)
		cmd.Dir = dir
		if err := cmd.Run(); err != nil {
			t.Fatalf("Failed to set git config %s: %v", key, err)
		}
	}
	// Mark the repo as initialized so the branch keys are parsed.
	cmd := exec.Command("git", "config", "gitflow.version", "1.0")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to set gitflow version: %v", err)
	}
}

// TestLoadConfigDottedBranchName covers spec scenario 1: a base branch whose
// name contains a dot (custom.main) is parsed into a single canonical entry
// with its properties intact, and no phantom "custom" branch is created.
func TestLoadConfigDottedBranchName(t *testing.T) {
	dir := setupTestRepo(t)
	defer cleanupTestRepo(t, dir)

	setDottedConfig(t, dir, map[string]string{
		"gitflow.branch.custom.main.type":             "base",
		"gitflow.branch.custom.main.upstreamStrategy": "merge",
	})

	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	branch, ok := cfg.Branches["custom.main"]
	if !ok {
		t.Fatalf("Expected branch 'custom.main' to be loaded; branches: %v", keysOf(cfg.Branches))
	}
	if branch.Type != "base" {
		t.Errorf("Expected Type 'base', got '%s'", branch.Type)
	}
	if branch.UpstreamStrategy != "merge" {
		t.Errorf("Expected UpstreamStrategy 'merge', got '%s'", branch.UpstreamStrategy)
	}
	if _, phantom := cfg.Branches["custom"]; phantom {
		t.Errorf("Did not expect a phantom 'custom' branch entry; branches: %v", keysOf(cfg.Branches))
	}
}

// TestLoadConfigDottedTopicRoundTrip covers spec scenario 3: a dotted topic
// branch type (qa.release) round-trips every field through the config reader
// without any being dropped by the parse.
func TestLoadConfigDottedTopicRoundTrip(t *testing.T) {
	dir := setupTestRepo(t)
	defer cleanupTestRepo(t, dir)

	setDottedConfig(t, dir, map[string]string{
		"gitflow.branch.qa.release.type":               "topic",
		"gitflow.branch.qa.release.parent":             "custom.main",
		"gitflow.branch.qa.release.startPoint":         "custom.dev",
		"gitflow.branch.qa.release.prefix":             "qa/",
		"gitflow.branch.qa.release.upstreamStrategy":   "merge",
		"gitflow.branch.qa.release.downstreamStrategy": "rebase",
		"gitflow.branch.qa.release.tag":                "true",
		"gitflow.branch.qa.release.tagPrefix":          "qa-",
	})

	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	branch, ok := cfg.Branches["qa.release"]
	if !ok {
		t.Fatalf("Expected branch 'qa.release' to be loaded; branches: %v", keysOf(cfg.Branches))
	}

	expected := config.BranchConfig{
		Type:               "topic",
		Parent:             "custom.main",
		StartPoint:         "custom.dev",
		Prefix:             "qa/",
		UpstreamStrategy:   "merge",
		DownstreamStrategy: "rebase",
		Tag:                true,
		TagPrefix:          "qa-",
	}
	if branch.Type != expected.Type {
		t.Errorf("Expected Type '%s', got '%s'", expected.Type, branch.Type)
	}
	if branch.Parent != expected.Parent {
		t.Errorf("Expected Parent '%s', got '%s'", expected.Parent, branch.Parent)
	}
	if branch.StartPoint != expected.StartPoint {
		t.Errorf("Expected StartPoint '%s', got '%s'", expected.StartPoint, branch.StartPoint)
	}
	if branch.Prefix != expected.Prefix {
		t.Errorf("Expected Prefix '%s', got '%s'", expected.Prefix, branch.Prefix)
	}
	if branch.UpstreamStrategy != expected.UpstreamStrategy {
		t.Errorf("Expected UpstreamStrategy '%s', got '%s'", expected.UpstreamStrategy, branch.UpstreamStrategy)
	}
	if branch.DownstreamStrategy != expected.DownstreamStrategy {
		t.Errorf("Expected DownstreamStrategy '%s', got '%s'", expected.DownstreamStrategy, branch.DownstreamStrategy)
	}
	if branch.Tag != expected.Tag {
		t.Errorf("Expected Tag %v, got %v", expected.Tag, branch.Tag)
	}
	if branch.TagPrefix != expected.TagPrefix {
		t.Errorf("Expected TagPrefix '%s', got '%s'", expected.TagPrefix, branch.TagPrefix)
	}
}

// TestLoadConfigDottedBooleanAndPrefix covers spec scenario 4: boolean and
// string properties survive on a dotted base branch (custom.dev).
func TestLoadConfigDottedBooleanAndPrefix(t *testing.T) {
	dir := setupTestRepo(t)
	defer cleanupTestRepo(t, dir)

	setDottedConfig(t, dir, map[string]string{
		"gitflow.branch.custom.dev.type":       "base",
		"gitflow.branch.custom.dev.parent":     "custom.main",
		"gitflow.branch.custom.dev.autoUpdate": "true",
	})

	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	branch, ok := cfg.Branches["custom.dev"]
	if !ok {
		t.Fatalf("Expected branch 'custom.dev' to be loaded; branches: %v", keysOf(cfg.Branches))
	}
	if branch.Parent != "custom.main" {
		t.Errorf("Expected Parent 'custom.main', got '%s'", branch.Parent)
	}
	if !branch.AutoUpdate {
		t.Errorf("Expected AutoUpdate true, got false")
	}
}

// TestLoadConfigMultiDotName covers spec scenario 5: a name with multiple dots
// (release.2.0) parses correctly, with the property read from the final
// segment regardless of how many dots precede it.
func TestLoadConfigMultiDotName(t *testing.T) {
	dir := setupTestRepo(t)
	defer cleanupTestRepo(t, dir)

	setDottedConfig(t, dir, map[string]string{
		"gitflow.branch.release.2.0.type": "base",
	})

	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	branch, ok := cfg.Branches["release.2.0"]
	if !ok {
		t.Fatalf("Expected branch 'release.2.0' to be loaded; branches: %v", keysOf(cfg.Branches))
	}
	if branch.Type != "base" {
		t.Errorf("Expected Type 'base', got '%s'", branch.Type)
	}
	if _, phantom := cfg.Branches["release"]; phantom {
		t.Errorf("Did not expect a phantom 'release' branch entry; branches: %v", keysOf(cfg.Branches))
	}
}

// TestLoadConfigMixedCaseDottedName covers spec scenario 6 (#123 regression
// guard): a mixed-case dotted name folds together with a differently-cased
// variant into a single canonical entry keyed by the first-seen case, with no
// phantom single-segment entries.
func TestLoadConfigMixedCaseDottedName(t *testing.T) {
	dir := setupTestRepo(t)
	defer cleanupTestRepo(t, dir)

	// Two keys with the same fold key but different case. Git stores subsections
	// case-sensitively, so these are two distinct config entries; the reader
	// must fold them into one canonical branch keyed by the first-seen case.
	// Write in a fixed order (mixed-case first) so the canonical is deterministic
	// — a map would iterate in random order and make first-seen flaky.
	for _, kv := range []struct{ key, value string }{
		{"gitflow.branch.Custom.Main.type", "base"},
		{"gitflow.branch.custom.main.parent", "production"},
	} {
		cmd := exec.Command("git", "config", kv.key, kv.value)
		cmd.Dir = dir
		if err := cmd.Run(); err != nil {
			t.Fatalf("Failed to set git config %s: %v", kv.key, err)
		}
	}
	cmd := exec.Command("git", "config", "gitflow.version", "1.0")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to set gitflow version: %v", err)
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	branch, ok := cfg.Branches["Custom.Main"]
	if !ok {
		t.Fatalf("Expected canonical branch 'Custom.Main' to be loaded; branches: %v", keysOf(cfg.Branches))
	}
	if branch.Type != "base" {
		t.Errorf("Expected Type 'base', got '%s'", branch.Type)
	}
	if branch.Parent != "production" {
		t.Errorf("Expected Parent 'production' to fold into the canonical entry, got '%s'", branch.Parent)
	}

	// No lowercased canonical duplicate, and no phantom single-segment entries.
	if _, dup := cfg.Branches["custom.main"]; dup {
		t.Errorf("Did not expect a lowercased duplicate 'custom.main'; branches: %v", keysOf(cfg.Branches))
	}
	for _, phantom := range []string{"Custom", "custom"} {
		if _, exists := cfg.Branches[phantom]; exists {
			t.Errorf("Did not expect a phantom '%s' branch entry; branches: %v", phantom, keysOf(cfg.Branches))
		}
	}

	// The #123 case-insensitive resolution still works for the dotted name.
	canonical, found := cfg.ResolveBranchName("custom.main")
	if !found {
		t.Fatalf("Expected ResolveBranchName(\"custom.main\") to resolve, got not found")
	}
	if canonical != "Custom.Main" {
		t.Errorf("Expected resolved canonical name 'Custom.Main', got '%s'", canonical)
	}
}
