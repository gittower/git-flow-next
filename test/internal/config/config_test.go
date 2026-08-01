package config_test

import (
	"os"
	"os/exec"
	"testing"

	"github.com/gittower/git-flow-next/internal/config"
	"github.com/gittower/git-flow-next/internal/git"
	"github.com/gittower/git-flow-next/test/testutil"
	"github.com/stretchr/testify/assert"
)

// setupTestRepo creates a temporary git repository and returns its path. Unlike
// the previous helper, it does not change the process working directory: tests
// open a git.Repo handle for the returned dir instead.
func setupTestRepo(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "git-flow-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if err := cmd.Run(); err != nil {
			t.Fatalf("git %v failed: %v", args, err)
		}
	}
	run("init")
	run("config", "user.name", "Test User")
	run("config", "user.email", "test@example.com")
	return dir
}

func cleanupTestRepo(t *testing.T, dir string) {
	if err := os.RemoveAll(dir); err != nil {
		t.Errorf("Failed to cleanup test repo: %v", err)
	}
}

// setConfig sets a git config value in dir.
func setConfig(t *testing.T, dir, key, value string) {
	t.Helper()
	cmd := exec.Command("git", "config", key, value)
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to set git config %s: %v", key, err)
	}
}

func openRepo(t *testing.T, dir string) *git.Repo {
	t.Helper()
	repo, err := git.Open(dir)
	if err != nil {
		t.Fatalf("git.Open(%q) failed: %v", dir, err)
	}
	return repo
}

func TestLoadConfigCaseInsensitive(t *testing.T) {
	t.Parallel()
	dir := setupTestRepo(t)
	defer cleanupTestRepo(t, dir)

	setConfig(t, dir, "gitflow.branch.feature.startPoint", "develop")
	setConfig(t, dir, "gitflow.branch.release.StartPoint", "develop")
	setConfig(t, dir, "gitflow.branch.hotfix.STARTPOINT", "main")
	setConfig(t, dir, "gitflow.version", "1.0")

	cfg, err := config.Load(openRepo(t, dir))
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

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
	t.Parallel()
	dir := setupTestRepo(t)
	defer cleanupTestRepo(t, dir)

	setConfig(t, dir, "gitflow.branch.feature.Type", "topic")
	setConfig(t, dir, "gitflow.branch.feature.parent", "develop")
	setConfig(t, dir, "gitflow.branch.feature.UpstreamStrategy", "rebase")
	setConfig(t, dir, "gitflow.branch.feature.downstreamStrategy", "squash")
	setConfig(t, dir, "gitflow.branch.feature.PREFIX", "feature/")
	setConfig(t, dir, "gitflow.branch.feature.AutoUpdate", "true")
	setConfig(t, dir, "gitflow.version", "1.0")

	cfg, err := config.Load(openRepo(t, dir))
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

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
	t.Parallel()
	cfg := config.DefaultConfig()
	cfg = config.ApplyOverrides(cfg, config.ConfigOverrides{})

	mainConfig, exists := cfg.Branches["main"]
	assert.True(t, exists)
	assert.Equal(t, string(config.BranchTypeBase), mainConfig.Type)
	assert.Equal(t, "", mainConfig.Parent)
	assert.Equal(t, "", mainConfig.StartPoint)

	developConfig, exists := cfg.Branches["develop"]
	assert.True(t, exists)
	assert.Equal(t, string(config.BranchTypeBase), developConfig.Type)
	assert.Equal(t, "main", developConfig.Parent)
	assert.Equal(t, "", developConfig.StartPoint)

	featureConfig, exists := cfg.Branches["feature"]
	assert.True(t, exists)
	assert.Equal(t, "feature/", featureConfig.Prefix)
	assert.Equal(t, "develop", featureConfig.Parent)
	assert.Equal(t, "develop", featureConfig.StartPoint)

	releaseConfig, exists := cfg.Branches["release"]
	assert.True(t, exists)
	assert.Equal(t, "release/", releaseConfig.Prefix)
	assert.Equal(t, "main", releaseConfig.Parent)
	assert.Equal(t, "develop", releaseConfig.StartPoint)

	hotfixConfig, exists := cfg.Branches["hotfix"]
	assert.True(t, exists)
	assert.Equal(t, "hotfix/", hotfixConfig.Prefix)
	assert.Equal(t, "main", hotfixConfig.Parent)
	assert.Equal(t, "main", hotfixConfig.StartPoint)

	supportConfig, exists := cfg.Branches["support"]
	assert.True(t, exists)
	assert.Equal(t, "support/", supportConfig.Prefix)
	assert.Equal(t, "main", supportConfig.Parent)
	assert.Equal(t, "main", supportConfig.StartPoint)
}

func TestApplyOverrides_CustomBranchNames(t *testing.T) {
	t.Parallel()
	cfg := config.DefaultConfig()
	cfg = config.ApplyOverrides(cfg, config.ConfigOverrides{
		MainBranch:    "custom-main",
		DevelopBranch: "custom-dev",
	})

	mainConfig, exists := cfg.Branches["custom-main"]
	assert.True(t, exists)
	assert.Equal(t, string(config.BranchTypeBase), mainConfig.Type)
	assert.Equal(t, "", mainConfig.Parent)
	assert.Equal(t, "", mainConfig.StartPoint)

	developConfig, exists := cfg.Branches["custom-dev"]
	assert.True(t, exists)
	assert.Equal(t, string(config.BranchTypeBase), developConfig.Type)
	assert.Equal(t, "custom-main", developConfig.Parent)
	assert.Equal(t, "", developConfig.StartPoint)

	featureConfig, exists := cfg.Branches["feature"]
	assert.True(t, exists)
	assert.Equal(t, "custom-dev", featureConfig.Parent)
	assert.Equal(t, "custom-dev", featureConfig.StartPoint)

	releaseConfig, exists := cfg.Branches["release"]
	assert.True(t, exists)
	assert.Equal(t, "custom-main", releaseConfig.Parent)
	assert.Equal(t, "custom-dev", releaseConfig.StartPoint)

	hotfixConfig, exists := cfg.Branches["hotfix"]
	assert.True(t, exists)
	assert.Equal(t, "custom-main", hotfixConfig.Parent)
	assert.Equal(t, "custom-main", hotfixConfig.StartPoint)

	supportConfig, exists := cfg.Branches["support"]
	assert.True(t, exists)
	assert.Equal(t, "custom-main", supportConfig.Parent)
	assert.Equal(t, "custom-main", supportConfig.StartPoint)

	_, exists = cfg.Branches["main"]
	assert.False(t, exists)
	_, exists = cfg.Branches["develop"]
	assert.False(t, exists)
}

func TestApplyOverrides_CustomPrefixes(t *testing.T) {
	t.Parallel()
	cfg := config.DefaultConfig()
	cfg = config.ApplyOverrides(cfg, config.ConfigOverrides{
		FeaturePrefix: "f/",
		ReleasePrefix: "r/",
		HotfixPrefix:  "h/",
		SupportPrefix: "s/",
	})

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
	t.Parallel()
	cfg := config.DefaultConfig()
	cfg = config.ApplyOverrides(cfg, config.ConfigOverrides{
		TagPrefix: "v",
	})

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
	t.Parallel()
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

	mainConfig, exists := cfg.Branches["custom-main"]
	assert.True(t, exists)
	assert.Equal(t, string(config.BranchTypeBase), mainConfig.Type)
	assert.Equal(t, "", mainConfig.Parent)
	assert.Equal(t, "", mainConfig.StartPoint)

	developConfig, exists := cfg.Branches["custom-dev"]
	assert.True(t, exists)
	assert.Equal(t, string(config.BranchTypeBase), developConfig.Type)
	assert.Equal(t, "custom-main", developConfig.Parent)
	assert.Equal(t, "", developConfig.StartPoint)

	featureConfig := cfg.Branches["feature"]
	assert.Equal(t, "f/", featureConfig.Prefix)
	assert.Equal(t, "custom-dev", featureConfig.Parent)
	assert.Equal(t, "custom-dev", featureConfig.StartPoint)

	releaseConfig := cfg.Branches["release"]
	assert.Equal(t, "r/", releaseConfig.Prefix)
	assert.Equal(t, "custom-main", releaseConfig.Parent)
	assert.Equal(t, "custom-dev", releaseConfig.StartPoint)
	assert.Equal(t, "v", releaseConfig.TagPrefix)

	hotfixConfig := cfg.Branches["hotfix"]
	assert.Equal(t, "h/", hotfixConfig.Prefix)
	assert.Equal(t, "custom-main", hotfixConfig.Parent)
	assert.Equal(t, "custom-main", hotfixConfig.StartPoint)
	assert.Equal(t, "v", hotfixConfig.TagPrefix)

	supportConfig := cfg.Branches["support"]
	assert.Equal(t, "s/", supportConfig.Prefix)
	assert.Equal(t, "custom-main", supportConfig.Parent)
	assert.Equal(t, "custom-main", supportConfig.StartPoint)
}

func TestDefaultRemoteConfiguration(t *testing.T) {
	t.Parallel()
	dir := setupTestRepo(t)
	defer cleanupTestRepo(t, dir)

	setConfig(t, dir, "gitflow.version", "1.0")

	cfg, err := config.Load(openRepo(t, dir))
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}
	assert.Equal(t, "origin", cfg.Remote, "Default remote should be 'origin'")
}

func TestGitFlowAVHRemoteImport(t *testing.T) {
	t.Parallel()
	dir := setupTestRepo(t)
	defer cleanupTestRepo(t, dir)

	setConfig(t, dir, "gitflow.origin", "avh-remote")

	cfg, err := config.ImportGitFlowAVHConfig(openRepo(t, dir))
	if err != nil {
		t.Fatalf("Failed to import git-flow-avh config: %v", err)
	}
	assert.Equal(t, "avh-remote", cfg.Remote, "git-flow-avh remote should be imported")
}

func TestLoadConfigPreservesBranchNameCase(t *testing.T) {
	t.Parallel()
	dir := setupTestRepo(t)
	defer cleanupTestRepo(t, dir)

	setConfig(t, dir, "gitflow.branch.V9_Release.type", "base")
	setConfig(t, dir, "gitflow.branch.V9_Release.upstreamStrategy", "merge")
	setConfig(t, dir, "gitflow.version", "1.0")

	cfg, err := config.Load(openRepo(t, dir))
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if _, exists := cfg.Branches["V9_Release"]; !exists {
		t.Errorf("Expected canonical branch key 'V9_Release' to be preserved; branches: %v", keysOf(cfg.Branches))
	}
	if _, exists := cfg.Branches["v9_release"]; exists {
		t.Errorf("Did not expect a lowercased branch key 'v9_release'; branches: %v", keysOf(cfg.Branches))
	}

	canonical, found := cfg.ResolveBranchName("v9_release")
	if !found {
		t.Fatalf("Expected ResolveBranchName(\"v9_release\") to resolve, got not found")
	}
	if canonical != "V9_Release" {
		t.Errorf("Expected resolved canonical name 'V9_Release', got '%s'", canonical)
	}

	branch := cfg.Branches[canonical]
	if branch.Type != "base" {
		t.Errorf("Expected Type 'base', got '%s'", branch.Type)
	}
	if branch.UpstreamStrategy != "merge" {
		t.Errorf("Expected UpstreamStrategy 'merge', got '%s'", branch.UpstreamStrategy)
	}
}

// --- Scenario 6: config read off-CWD ---

func TestLoadConfigReflectsTargetRepoOffCwd(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	if _, err := testutil.RunGitFlow(t, dir, "init", "--defaults"); err != nil {
		t.Fatalf("Failed to init git-flow: %v", err)
	}
	if _, err := testutil.RunGit(t, dir, "config", "gitflow.branch.feature.prefix", "feat/"); err != nil {
		t.Fatalf("Failed to set feature prefix: %v", err)
	}

	cfg, err := config.Load(openRepo(t, dir))
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}
	if got := cfg.Branches["feature"].Prefix; got != "feat/" {
		t.Errorf("Expected feature prefix 'feat/' from B, got %q", got)
	}
}

func TestLoadConfigUninitializedReturnsDefaults(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	cfg, err := config.Load(openRepo(t, dir))
	if err != nil {
		t.Fatalf("Expected no error for uninitialized repo, got %v", err)
	}
	// Falls back to defaults: default feature prefix, default remote.
	if got := cfg.Branches["feature"].Prefix; got != "feature/" {
		t.Errorf("Expected default feature prefix 'feature/', got %q", got)
	}
	if cfg.Remote != "origin" {
		t.Errorf("Expected default remote 'origin', got %q", cfg.Remote)
	}
}

func TestLoadConfigImportsAvhOffCwd(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	// git-flow-avh keys with distinctive non-default values, no gitflow.version.
	setConfig(t, dir, "gitflow.branch.master", "production")
	setConfig(t, dir, "gitflow.prefix.feature", "feat/")
	setConfig(t, dir, "gitflow.prefix.hotfix", "hf/")

	cfg, err := config.Load(openRepo(t, dir))
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if _, exists := cfg.Branches["production"]; !exists {
		t.Errorf("Expected AVH master 'production' imported; branches: %v", keysOf(cfg.Branches))
	}
	if _, exists := cfg.Branches["main"]; exists {
		t.Errorf("Did not expect default 'main' after AVH import renamed it to 'production'")
	}
	if got := cfg.Branches["feature"].Prefix; got != "feat/" {
		t.Errorf("Expected imported feature prefix 'feat/', got %q", got)
	}
}

// --- Scenario 7: config mutation off-CWD ---

func TestConfigSetIsolatedToTargetRepo(t *testing.T) {
	t.Parallel()
	dirA := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dirA)
	dirB := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dirB)

	if _, err := testutil.RunGitFlow(t, dirA, "init", "--defaults"); err != nil {
		t.Fatalf("Failed to init A: %v", err)
	}
	if _, err := testutil.RunGitFlow(t, dirB, "init", "--defaults"); err != nil {
		t.Fatalf("Failed to init B: %v", err)
	}

	repoB := openRepo(t, dirB)
	if err := repoB.SetConfig("gitflow.branch.feature.prefix", "xb/"); err != nil {
		t.Fatalf("SetConfig on B failed: %v", err)
	}

	cfgB, err := config.Load(repoB)
	if err != nil {
		t.Fatalf("Failed to load B config: %v", err)
	}
	if got := cfgB.Branches["feature"].Prefix; got != "xb/" {
		t.Errorf("Expected B feature prefix 'xb/', got %q", got)
	}

	cfgA, err := config.Load(openRepo(t, dirA))
	if err != nil {
		t.Fatalf("Failed to load A config: %v", err)
	}
	if got := cfgA.Branches["feature"].Prefix; got != "feature/" {
		t.Errorf("Expected A feature prefix to retain default 'feature/', got %q", got)
	}
}

func TestConfigClearIsolatedToTargetRepo(t *testing.T) {
	t.Parallel()
	dirA := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dirA)
	dirB := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dirB)

	if _, err := testutil.RunGitFlow(t, dirA, "init", "--defaults"); err != nil {
		t.Fatalf("Failed to init A: %v", err)
	}
	if _, err := testutil.RunGitFlow(t, dirB, "init", "--defaults"); err != nil {
		t.Fatalf("Failed to init B: %v", err)
	}

	repoB := openRepo(t, dirB)
	if err := config.ClearConfig(repoB); err != nil {
		t.Fatalf("ClearConfig on B failed: %v", err)
	}

	// B's gitflow config is gone.
	if _, err := repoB.GetConfig("gitflow.branch.feature.prefix"); err == nil {
		t.Error("Expected B's gitflow.branch.feature.prefix to be removed")
	}

	// A's gitflow config is intact.
	repoA := openRepo(t, dirA)
	got, err := repoA.GetConfig("gitflow.branch.feature.prefix")
	if err != nil {
		t.Fatalf("Expected A's gitflow.branch.feature.prefix intact, got err: %v", err)
	}
	if got != "feature/" {
		t.Errorf("Expected A feature prefix 'feature/', got %q", got)
	}
}

func keysOf(m map[string]config.BranchConfig) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
