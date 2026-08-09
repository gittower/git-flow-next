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

// TestLoadConfigCaseInsensitive verifies that branch property keys are matched
// case-insensitively when loading configuration.
// Steps:
// 1. Sets up a test repository
// 2. Sets startPoint keys with mixed casing (startPoint, StartPoint, STARTPOINT)
// 3. Loads config through a git.Repo handle for the repository
// 4. Verifies each branch's start point resolves regardless of key case
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

// TestLoadConfigWithMixedCaseProperties verifies that all branch config
// properties are parsed regardless of the casing used in their config keys.
// Steps:
// 1. Sets up a test repository
// 2. Sets the full set of feature branch properties with mixed-case keys
// 3. Loads config through a git.Repo handle for the repository
// 4. Verifies each parsed feature property matches the expected value
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

// TestApplyOverrides_NoOverrides verifies that applying empty overrides leaves
// the default branch configuration intact.
// Steps:
// 1. Builds a default config
// 2. Applies an empty ConfigOverrides
// 3. Verifies all default branches retain their types, parents, and start points
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

// TestApplyOverrides_CustomBranchNames verifies that overriding the main and
// develop branch names rekeys the branches and updates dependent parents.
// Steps:
// 1. Builds a default config
// 2. Applies overrides setting custom main and develop branch names
// 3. Verifies the custom-named base branches exist with correct parents
// 4. Verifies feature/release/hotfix/support parents point at the custom names
// 5. Verifies the original "main" and "develop" keys no longer exist
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

// TestApplyOverrides_CustomPrefixes verifies that custom branch prefix overrides
// are applied without altering parents or start points.
// Steps:
// 1. Builds a default config
// 2. Applies overrides setting custom feature/release/hotfix/support prefixes
// 3. Verifies each branch adopts its custom prefix
// 4. Verifies each branch retains its default parent and start point
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

// TestApplyOverrides_CustomTagPrefix verifies that a custom tag prefix override
// is applied to the tagging branches.
// Steps:
// 1. Builds a default config
// 2. Applies an override setting a custom tag prefix
// 3. Verifies the release and hotfix branches adopt the custom tag prefix
// 4. Verifies their parents and start points are unchanged
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

// TestApplyOverrides_AllOverrides verifies that applying the full set of
// overrides together produces a consistent, fully customized config.
// Steps:
// 1. Builds a default config
// 2. Applies custom branch names, prefixes, and tag prefix together
// 3. Verifies the custom-named base branches exist with correct parents
// 4. Verifies feature/release/hotfix/support prefixes, parents, and start points
// 5. Verifies the custom tag prefix on release and hotfix branches
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

// TestDefaultRemoteConfiguration verifies that the remote defaults to "origin"
// when no remote is configured.
// Steps:
// 1. Sets up a test repository and marks it git-flow-initialized
// 2. Loads config through a git.Repo handle for the repository
// 3. Verifies the loaded remote is "origin"
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

// TestGitFlowAVHRemoteImport verifies that the remote is imported from the
// git-flow-avh gitflow.origin key.
// Steps:
// 1. Sets up a test repository and sets gitflow.origin to a custom remote
// 2. Imports git-flow-avh config through a git.Repo handle
// 3. Verifies the imported config's remote matches the avh value
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

// TestLoadConfigPreservesBranchNameCase verifies that the canonical casing of a
// branch name is preserved and remains case-insensitively resolvable.
// Steps:
// 1. Sets up a test repository with a mixed-case branch name (V9_Release)
// 2. Loads config through a git.Repo handle for the repository
// 3. Verifies the canonical key 'V9_Release' is preserved and not lowercased
// 4. Verifies ResolveBranchName resolves a lowercased query to the canonical key
// 5. Verifies the branch's parsed Type and UpstreamStrategy
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

// TestLoadConfigReflectsTargetRepoOffCwd verifies config.Load reads from the
// target repository, not the process working directory.
// Steps:
// 1. Sets up a test repository B and initializes git-flow in it
// 2. Sets a custom feature prefix (feat/) in B
// 3. Loads config through a git.Repo handle for B
// 4. Verifies the loaded feature prefix is 'feat/' from B
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

// TestLoadConfigUninitializedReturnsDefaults verifies config.Load returns
// defaults for a repository that has not been git-flow-initialized.
// Steps:
// 1. Sets up a test repository B without initializing git-flow
// 2. Loads config through a git.Repo handle for B
// 3. Verifies no error is returned
// 4. Verifies the default feature prefix ('feature/') and default remote ('origin')
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

// TestLoadConfigImportsAvhOffCwd verifies config.Load imports git-flow-avh
// configuration from the target repository when no gitflow.version is present.
// Steps:
// 1. Sets up a test repository B with distinctive git-flow-avh keys and no version
// 2. Loads config through a git.Repo handle for B
// 3. Verifies the AVH master rename ('production') is imported and 'main' is gone
// 4. Verifies the imported feature prefix is 'feat/'
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

// TestConfigSetIsolatedToTargetRepo verifies that a config write through one
// repository's handle affects only that repository.
// Steps:
// 1. Sets up and initializes two test repositories A and B
// 2. Sets a custom feature prefix ('xb/') through B's handle
// 3. Verifies B's loaded config reflects the new prefix
// 4. Verifies A's loaded config retains the default 'feature/' prefix
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

// TestConfigClearIsolatedToTargetRepo verifies that clearing config through one
// repository's handle affects only that repository.
// Steps:
// 1. Sets up and initializes two test repositories A and B
// 2. Clears gitflow config through B's handle
// 3. Verifies B's gitflow.branch.feature.prefix is removed
// 4. Verifies A's gitflow.branch.feature.prefix remains 'feature/'
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

// TestTrunkBranchReturnsUnparentedBaseBranch tests that TrunkBranch reports the
// base branch with no parent for the shipped default configuration.
// Steps:
// 1. Builds config.DefaultConfig()
// 2. Calls TrunkBranch()
// 3. Verifies it returns "main"
func TestTrunkBranchReturnsUnparentedBaseBranch(t *testing.T) {
	t.Parallel()
	if got := config.DefaultConfig().TrunkBranch(); got != "main" {
		t.Errorf("Expected trunk 'main', got %q", got)
	}
}

// TestTrunkBranchTieBreakIsLexicographic tests that TrunkBranch is deterministic
// when several base branches have no parent, despite Go's randomized map
// iteration order.
// Steps:
// 1. Builds a Config with two unparented base branches, "zulu" and "alpha"
// 2. Calls TrunkBranch() at least 20 times
// 3. Verifies every call returns "alpha", the lexicographically smallest name
func TestTrunkBranchTieBreakIsLexicographic(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{
		Version: "1.0",
		Branches: map[string]config.BranchConfig{
			"zulu":  {Type: string(config.BranchTypeBase)},
			"alpha": {Type: string(config.BranchTypeBase)},
		},
	}
	for i := 0; i < 20; i++ {
		if got := cfg.TrunkBranch(); got != "alpha" {
			t.Fatalf("Iteration %d: expected trunk 'alpha', got %q", i, got)
		}
	}
}

// TestTrunkBranchWithoutTrunkReturnsEmpty tests that TrunkBranch returns an
// empty string for a configuration whose base branches all have a parent.
// Steps:
// 1. Builds a Config whose only base branch has a non-empty Parent
// 2. Calls TrunkBranch()
// 3. Verifies it returns ""
func TestTrunkBranchWithoutTrunkReturnsEmpty(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{
		Version: "1.0",
		Branches: map[string]config.BranchConfig{
			"develop": {Type: string(config.BranchTypeBase), Parent: "main"},
		},
	}
	if got := cfg.TrunkBranch(); got != "" {
		t.Errorf("Expected empty trunk, got %q", got)
	}
}
