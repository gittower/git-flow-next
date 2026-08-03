package config_test

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/gittower/git-flow-next/internal/config"
	"github.com/gittower/git-flow-next/test/testutil"
)

// setSharedConfig writes a key into the repository's .gitflow file via
// `git config --file`.
func setSharedConfig(t *testing.T, dir, key, value string) {
	t.Helper()
	cmd := exec.Command("git", "config", "--file", config.SharedConfigFileName, key, value)
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to set shared config %s: %v", key, err)
	}
}

// localConfigValue returns a single value from local git config, and whether the
// key is present.
func localConfigValue(t *testing.T, dir, key string) (string, bool) {
	t.Helper()
	cmd := exec.Command("git", "config", "--local", "--get", key)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "", false
	}
	return strings.TrimSpace(string(out)), true
}

// TestSharedManagedKeyPredicate covers scenario 38: the shared-managed key
// predicate classifies keys correctly (pure function, table-driven per the
// TESTING_GUIDELINES exception).
// Steps:
// 1. Enumerates representative keys with their expected managed classification
// 2. Calls config.IsSharedManagedKey on each
// 3. Verifies control keys and runtime .base keys are excluded, real config included
func TestSharedManagedKeyPredicate(t *testing.T) {
	t.Parallel()
	cases := []struct {
		key     string
		managed bool
	}{
		{"gitflow.branch.feature.prefix", true},
		{"gitflow.feature.publish.push-option", true},
		{"gitflow.version", true},
		{"gitflow.initialized", true},
		{"gitflow.shared.autoInit", false},
		{"gitflow.shared.trustHooks", false},
		{"gitflow.branch.feature/foo.base", false},
		{"core.sshCommand", false},
	}
	for _, tc := range cases {
		t.Run(tc.key, func(t *testing.T) {
			if got := config.IsSharedManagedKey(tc.key); got != tc.managed {
				t.Errorf("IsSharedManagedKey(%q) = %v, want %v", tc.key, got, tc.managed)
			}
		})
	}
}

// TestCopySharedToLocalStaleRemovalPreservesLocalOnly covers scenario 39:
// CopySharedToLocal removes stale managed keys but preserves local-only keys.
// Steps:
// 1. Seeds local with a stale managed key, a shared-control key, and a runtime base key
// 2. Writes a .gitflow with feature defaults (no "old" branch)
// 3. Calls config.CopySharedToLocal
// 4. Verifies old.* is gone, feature.* copied, and the local-only keys preserved
func TestCopySharedToLocalStaleRemovalPreservesLocalOnly(t *testing.T) {
	t.Parallel()
	dir := setupTestRepo(t)
	defer cleanupTestRepo(t, dir)

	setConfig(t, dir, "gitflow.branch.old.type", "topic")
	setConfig(t, dir, "gitflow.shared.autoInit", "true")
	setConfig(t, dir, "gitflow.branch.feature/foo.base", "develop")

	setSharedConfig(t, dir, "gitflow.version", "1.0")
	setSharedConfig(t, dir, "gitflow.branch.feature.type", "topic")
	setSharedConfig(t, dir, "gitflow.branch.feature.prefix", "feature/")

	if _, err := config.CopySharedToLocal(openRepo(t, dir)); err != nil {
		t.Fatalf("CopySharedToLocal failed: %v", err)
	}

	if _, ok := localConfigValue(t, dir, "gitflow.branch.old.type"); ok {
		t.Error("expected stale gitflow.branch.old.type to be removed from local")
	}
	if v, _ := localConfigValue(t, dir, "gitflow.branch.feature.prefix"); v != "feature/" {
		t.Errorf("expected local feature.prefix=feature/, got %q", v)
	}
	if v, _ := localConfigValue(t, dir, "gitflow.shared.autoInit"); v != "true" {
		t.Errorf("expected local gitflow.shared.autoInit preserved as true, got %q", v)
	}
	if v, _ := localConfigValue(t, dir, "gitflow.branch.feature/foo.base"); v != "develop" {
		t.Errorf("expected runtime gitflow.branch.feature/foo.base preserved as develop, got %q", v)
	}
}

// TestCopySharedToLocalIgnoresExcludedKeysInFile covers scenario 39b:
// CopySharedToLocal never copies excluded keys that appear inside .gitflow and
// never overwrites their local counterparts.
// Steps:
// 1. Seeds local counterparts for each excluded key with differing values
// 2. Writes a .gitflow that (wrongly) contains excluded keys plus feature defaults
// 3. Calls config.CopySharedToLocal
// 4. Verifies excluded keys are inert (not copied / not overwritten) and defaults copied
func TestCopySharedToLocalIgnoresExcludedKeysInFile(t *testing.T) {
	t.Parallel()
	dir := setupTestRepo(t)
	defer cleanupTestRepo(t, dir)

	setConfig(t, dir, "gitflow.shared.autoInit", "false")
	setConfig(t, dir, "gitflow.branch.feature/foo.base", "develop")

	setSharedConfig(t, dir, "gitflow.version", "1.0")
	setSharedConfig(t, dir, "gitflow.shared.autoInit", "true")
	setSharedConfig(t, dir, "gitflow.shared.trustHooks", "true")
	setSharedConfig(t, dir, "gitflow.branch.feature/foo.base", "main")
	setSharedConfig(t, dir, "gitflow.branch.feature.type", "topic")
	setSharedConfig(t, dir, "gitflow.branch.feature.prefix", "feature/")

	if _, err := config.CopySharedToLocal(openRepo(t, dir)); err != nil {
		t.Fatalf("CopySharedToLocal failed: %v", err)
	}

	if v, _ := localConfigValue(t, dir, "gitflow.shared.autoInit"); v != "false" {
		t.Errorf("expected local gitflow.shared.autoInit untouched (false), got %q", v)
	}
	if v, _ := localConfigValue(t, dir, "gitflow.branch.feature/foo.base"); v != "develop" {
		t.Errorf("expected local runtime base untouched (develop), got %q", v)
	}
	if _, ok := localConfigValue(t, dir, "gitflow.shared.trustHooks"); ok {
		t.Error("expected local gitflow.shared.trustHooks NOT created from .gitflow")
	}
	if v, _ := localConfigValue(t, dir, "gitflow.branch.feature.prefix"); v != "feature/" {
		t.Errorf("expected local feature.prefix copied as feature/, got %q", v)
	}
}

// TestIsUnconfiguredFreshRepo covers scenario 40a: a fresh repo with no gitflow.*
// reads as unconfigured.
// Steps:
// 1. Creates a fresh repository with no git-flow config
// 2. Calls config.IsUnconfiguredIgnoringShared
// 3. Verifies it reports unconfigured (true)
func TestIsUnconfiguredFreshRepo(t *testing.T) {
	t.Parallel()
	dir := setupTestRepo(t)
	defer cleanupTestRepo(t, dir)

	unconfigured, err := config.IsUnconfiguredIgnoringShared(openRepo(t, dir))
	if err != nil {
		t.Fatalf("IsUnconfiguredIgnoringShared failed: %v", err)
	}
	if !unconfigured {
		t.Error("expected fresh repo to read as unconfigured")
	}
}

// TestIsUnconfiguredSharedControlOnly covers scenario 40b: a repo carrying only
// gitflow.shared.* control keys still reads as unconfigured.
// Steps:
// 1. Sets only gitflow.shared.autoInit and gitflow.shared.trustHooks
// 2. Calls config.IsUnconfiguredIgnoringShared
// 3. Verifies it reports unconfigured (shared-control keys are ignored)
func TestIsUnconfiguredSharedControlOnly(t *testing.T) {
	t.Parallel()
	dir := setupTestRepo(t)
	defer cleanupTestRepo(t, dir)

	setConfig(t, dir, "gitflow.shared.autoInit", "true")
	setConfig(t, dir, "gitflow.shared.trustHooks", "true")

	unconfigured, err := config.IsUnconfiguredIgnoringShared(openRepo(t, dir))
	if err != nil {
		t.Fatalf("IsUnconfiguredIgnoringShared failed: %v", err)
	}
	if !unconfigured {
		t.Error("expected shared-control-only repo to read as unconfigured")
	}
}

// TestIsUnconfiguredCommandOnlyKey covers scenario 40c: a repo with only a
// command-scoped gitflow.* key (no version, no branch definitions) still reads as
// unconfigured.
// Steps:
// 1. Sets only gitflow.feature.start.fetch=true
// 2. Calls config.IsUnconfiguredIgnoringShared
// 3. Verifies it reports unconfigured (a stray command key is not branch config)
func TestIsUnconfiguredCommandOnlyKey(t *testing.T) {
	t.Parallel()
	dir := setupTestRepo(t)
	defer cleanupTestRepo(t, dir)

	setConfig(t, dir, "gitflow.feature.start.fetch", "true")

	unconfigured, err := config.IsUnconfiguredIgnoringShared(openRepo(t, dir))
	if err != nil {
		t.Fatalf("IsUnconfiguredIgnoringShared failed: %v", err)
	}
	if !unconfigured {
		t.Error("expected command-only-key repo to read as unconfigured")
	}
}

// TestIsUnconfiguredAfterGitFlowInit covers scenario 40d: a repo initialized by
// git-flow-next reads as configured.
// Steps:
// 1. Initializes git-flow with defaults via the binary
// 2. Calls config.IsUnconfiguredIgnoringShared
// 3. Verifies it reports configured (false)
func TestIsUnconfiguredAfterGitFlowInit(t *testing.T) {
	t.Parallel()
	dir := setupTestRepo(t)
	defer cleanupTestRepo(t, dir)

	cmd := exec.Command(testutil.GitFlowPath(), "init", "--defaults")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git flow init failed: %v\n%s", err, out)
	}

	unconfigured, err := config.IsUnconfiguredIgnoringShared(openRepo(t, dir))
	if err != nil {
		t.Fatalf("IsUnconfiguredIgnoringShared failed: %v", err)
	}
	if unconfigured {
		t.Error("expected git-flow-initialized repo to read as configured")
	}
}

// TestIsUnconfiguredAvhOnlyNoVersion covers scenario 40e: AVH-only config with no
// gitflow.version reads as configured.
// Steps:
// 1. Sets git-flow-avh style keys (gitflow.branch.master, gitflow.prefix.feature) and no version
// 2. Calls config.IsUnconfiguredIgnoringShared
// 3. Verifies it reports configured (AVH branch config counts as configuration)
func TestIsUnconfiguredAvhOnlyNoVersion(t *testing.T) {
	t.Parallel()
	dir := setupTestRepo(t)
	defer cleanupTestRepo(t, dir)

	setConfig(t, dir, "gitflow.branch.master", "master")
	setConfig(t, dir, "gitflow.branch.develop", "develop")
	setConfig(t, dir, "gitflow.prefix.feature", "feature/")

	unconfigured, err := config.IsUnconfiguredIgnoringShared(openRepo(t, dir))
	if err != nil {
		t.Fatalf("IsUnconfiguredIgnoringShared failed: %v", err)
	}
	if unconfigured {
		t.Error("expected AVH-only repo to read as configured")
	}
}

// TestIsUnconfiguredBranchDefsNoVersion covers scenario 40f: valid
// git-flow-next branch definitions without gitflow.version read as configured.
// Steps:
// 1. Sets gitflow.branch.feature.type/parent/prefix but no gitflow.version
// 2. Calls config.IsUnconfiguredIgnoringShared
// 3. Verifies it reports configured (branch definitions alone are enough)
func TestIsUnconfiguredBranchDefsNoVersion(t *testing.T) {
	t.Parallel()
	dir := setupTestRepo(t)
	defer cleanupTestRepo(t, dir)

	setConfig(t, dir, "gitflow.branch.feature.type", "topic")
	setConfig(t, dir, "gitflow.branch.feature.parent", "develop")
	setConfig(t, dir, "gitflow.branch.feature.prefix", "feature/")

	unconfigured, err := config.IsUnconfiguredIgnoringShared(openRepo(t, dir))
	if err != nil {
		t.Fatalf("IsUnconfiguredIgnoringShared failed: %v", err)
	}
	if unconfigured {
		t.Error("expected repo with branch definitions to read as configured")
	}
}
