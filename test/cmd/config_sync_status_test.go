package cmd_test

import (
	"os"
	"strings"
	"testing"

	"github.com/gittower/git-flow-next/internal/errors"
	"github.com/gittower/git-flow-next/test/testutil"
)

// initSharedDefaults sets up a repo initialized via `git flow init --shared
// --defaults` (local config already matches .gitflow) and returns its directory.
func initSharedDefaults(t *testing.T) string {
	t.Helper()
	dir := testutil.SetupTestRepo(t)
	if out, err := testutil.RunGitFlow(t, dir, "init", "--shared", "--defaults"); err != nil {
		testutil.CleanupTestRepo(t, dir)
		t.Fatalf("init --shared failed: %v\n%s", err, out)
	}
	return dir
}

// TestConfigSyncAddsMissingLocalValue covers scenario 19: sync pulls a value
// present in .gitflow but absent locally.
// Steps:
// 1. init --shared --defaults, then add gitflow.feature.start.fetch=true only to .gitflow
// 2. Runs 'git flow config sync'
// 3. Verifies local gitflow.feature.start.fetch=true
func TestConfigSyncAddsMissingLocalValue(t *testing.T) {
	t.Parallel()
	dir := initSharedDefaults(t)
	defer testutil.CleanupTestRepo(t, dir)
	testutil.SharedConfigSet(t, dir, "gitflow.feature.start.fetch", "true")

	if out, err := testutil.RunGitFlow(t, dir, "config", "sync"); err != nil {
		t.Fatalf("config sync failed: %v\n%s", err, out)
	}
	if v := testutil.GitConfigValue(t, dir, "gitflow.feature.start.fetch"); v != "true" {
		t.Errorf("expected local start.fetch=true after sync, got %q", v)
	}
}

// TestConfigSyncRemovesStaleKeys covers scenario 20: sync removes a stale branch
// type and a dropped multi-value entry.
// Steps:
// 1. init --shared --defaults, add a push-option [a,b] and a qa type to both file and local
// 2. Edits .gitflow to remove the qa type and drop the b push-option value
// 3. Runs 'git flow config sync' and verifies qa.* is gone and push-option is [a] only
func TestConfigSyncRemovesStaleKeys(t *testing.T) {
	t.Parallel()
	dir := initSharedDefaults(t)
	defer testutil.CleanupTestRepo(t, dir)

	sharedPath := testutil.SharedConfigPath(dir)
	if _, err := testutil.RunGit(t, dir, "config", "--file", sharedPath, "--add", "gitflow.feature.publish.push-option", "a"); err != nil {
		t.Fatalf("add push-option a: %v", err)
	}
	if _, err := testutil.RunGit(t, dir, "config", "--file", sharedPath, "--add", "gitflow.feature.publish.push-option", "b"); err != nil {
		t.Fatalf("add push-option b: %v", err)
	}
	if out, err := testutil.RunGitFlow(t, dir, "config", "add", "topic", "qa", "develop", "--shared"); err != nil {
		t.Fatalf("config add topic qa --shared failed: %v\n%s", err, out)
	}
	if out, err := testutil.RunGitFlow(t, dir, "config", "sync"); err != nil {
		t.Fatalf("initial sync failed: %v\n%s", err, out)
	}

	// Shrink .gitflow: drop the qa type and the b push-option value.
	if _, err := testutil.RunGit(t, dir, "config", "--file", sharedPath, "--remove-section", "gitflow.branch.qa"); err != nil {
		t.Fatalf("remove qa section: %v", err)
	}
	if _, err := testutil.RunGit(t, dir, "config", "--file", sharedPath, "--unset-all", "gitflow.feature.publish.push-option"); err != nil {
		t.Fatalf("unset push-option: %v", err)
	}
	if _, err := testutil.RunGit(t, dir, "config", "--file", sharedPath, "--add", "gitflow.feature.publish.push-option", "a"); err != nil {
		t.Fatalf("re-add push-option a: %v", err)
	}

	if out, err := testutil.RunGitFlow(t, dir, "config", "sync"); err != nil {
		t.Fatalf("config sync failed: %v\n%s", err, out)
	}
	if testutil.GitConfigHasPrefix(t, dir, "gitflow.branch.qa.") {
		t.Error("expected stale local gitflow.branch.qa.* to be removed")
	}
	local := testutil.GitConfigAll(t, dir, "gitflow.feature.publish.push-option")
	if len(local) != 1 || local[0] != "a" {
		t.Errorf("expected local push-option [a] after shrink, got %v", local)
	}
}

// TestConfigSyncOverwritesLocalValue covers scenario 21: sync overwrites a
// locally-differing shared-managed value.
// Steps:
// 1. init --shared --defaults, then set local feature.prefix=local/ (file still feature/)
// 2. Runs 'git flow config sync'
// 3. Verifies local feature.prefix is overwritten back to feature/
func TestConfigSyncOverwritesLocalValue(t *testing.T) {
	t.Parallel()
	dir := initSharedDefaults(t)
	defer testutil.CleanupTestRepo(t, dir)
	if _, err := testutil.RunGit(t, dir, "config", "--local", "gitflow.branch.feature.prefix", "local/"); err != nil {
		t.Fatalf("set local prefix: %v", err)
	}

	if out, err := testutil.RunGitFlow(t, dir, "config", "sync"); err != nil {
		t.Fatalf("config sync failed: %v\n%s", err, out)
	}
	if v := testutil.GitConfigValue(t, dir, "gitflow.branch.feature.prefix"); v != "feature/" {
		t.Errorf("expected local feature.prefix overwritten to feature/, got %q", v)
	}
}

// TestConfigSyncPreservesLocalOnlyKeys covers scenario 22: sync preserves
// local-only keys (autoInit control key and runtime branch metadata).
// Steps:
// 1. init --shared --defaults, set local gitflow.shared.autoInit=true, start feature/foo (writes a .base key)
// 2. Runs 'git flow config sync'
// 3. Verifies autoInit and the runtime feature/foo.base key both survive
func TestConfigSyncPreservesLocalOnlyKeys(t *testing.T) {
	t.Parallel()
	dir := initSharedDefaults(t)
	defer testutil.CleanupTestRepo(t, dir)
	if _, err := testutil.RunGit(t, dir, "config", "--local", "gitflow.shared.autoInit", "true"); err != nil {
		t.Fatalf("set autoInit: %v", err)
	}
	if out, err := testutil.RunGitFlow(t, dir, "feature", "start", "foo"); err != nil {
		t.Fatalf("feature start foo failed: %v\n%s", err, out)
	}

	if out, err := testutil.RunGitFlow(t, dir, "config", "sync"); err != nil {
		t.Fatalf("config sync failed: %v\n%s", err, out)
	}
	if v := testutil.GitConfigValue(t, dir, "gitflow.shared.autoInit"); v != "true" {
		t.Errorf("expected local autoInit preserved, got %q", v)
	}
	if v := testutil.GitConfigValue(t, dir, "gitflow.branch.feature/foo.base"); v == "" {
		t.Error("expected runtime gitflow.branch.feature/foo.base preserved after sync")
	}
}

// TestConfigStatusInSyncExitsZero covers scenario 23: status when in sync exits 0.
// Steps:
// 1. init --shared --defaults (local already matches .gitflow)
// 2. Runs 'git flow config status'
// 3. Verifies it reports in sync and exits 0
func TestConfigStatusInSyncExitsZero(t *testing.T) {
	t.Parallel()
	dir := initSharedDefaults(t)
	defer testutil.CleanupTestRepo(t, dir)

	out, err := testutil.RunGitFlow(t, dir, "config", "status")
	if err != nil {
		t.Fatalf("expected config status to exit 0 when in sync, got err: %v\n%s", err, out)
	}
	if !strings.Contains(out, "in sync") {
		t.Errorf("expected 'in sync' in output, got: %s", out)
	}
}

// TestConfigStatusDriftExitsNonZero covers scenario 24: status when out of sync
// names the differing key, exits 6, and does not flag local-only keys as drift.
// Steps:
// 1. init --shared --defaults, set .gitflow feature.prefix=feat/, add local-only autoInit and a runtime base key
// 2. Runs 'git flow config status'
// 3. Verifies out-of-sync, names feature.prefix, exit 6, and local-only keys not reported
func TestConfigStatusDriftExitsNonZero(t *testing.T) {
	t.Parallel()
	dir := initSharedDefaults(t)
	defer testutil.CleanupTestRepo(t, dir)
	testutil.SharedConfigSet(t, dir, "gitflow.branch.feature.prefix", "feat/")
	if _, err := testutil.RunGit(t, dir, "config", "--local", "gitflow.shared.autoInit", "true"); err != nil {
		t.Fatalf("set autoInit: %v", err)
	}
	if out, err := testutil.RunGitFlow(t, dir, "feature", "start", "foo"); err != nil {
		t.Fatalf("feature start foo failed: %v\n%s", err, out)
	}

	out, err := testutil.RunGitFlow(t, dir, "config", "status")
	if err == nil {
		t.Fatalf("expected config status to fail on drift, got success\n%s", out)
	}
	assertExitCode(t, err, errors.ExitCodeValidationError)
	if !strings.Contains(out, "gitflow.branch.feature.prefix") {
		t.Errorf("expected drift to name gitflow.branch.feature.prefix, got: %s", out)
	}
	if strings.Contains(out, "autoInit") || strings.Contains(out, "feature/foo.base") {
		t.Errorf("expected local-only keys not reported as drift, got: %s", out)
	}
}

// TestConfigStatusDriftMissingLocalKey covers scenario 24b: status drift for a
// shared-managed key present in .gitflow but missing locally.
// Steps:
// 1. init --shared --defaults, add gitflow.feature.start.fetch=true only to .gitflow
// 2. Runs 'git flow config status'
// 3. Verifies out-of-sync, names start.fetch, exit 6
func TestConfigStatusDriftMissingLocalKey(t *testing.T) {
	t.Parallel()
	dir := initSharedDefaults(t)
	defer testutil.CleanupTestRepo(t, dir)
	testutil.SharedConfigSet(t, dir, "gitflow.feature.start.fetch", "true")

	out, err := testutil.RunGitFlow(t, dir, "config", "status")
	if err == nil {
		t.Fatalf("expected drift for shared-only key, got success\n%s", out)
	}
	assertExitCode(t, err, errors.ExitCodeValidationError)
	if !strings.Contains(out, "gitflow.feature.start.fetch") {
		t.Errorf("expected drift to name gitflow.feature.start.fetch, got: %s", out)
	}
}

// TestConfigStatusDriftStaleLocalKey covers scenario 24c: status drift for a
// stale shared-managed key present locally but absent from .gitflow.
// Steps:
// 1. init --shared --defaults, set a managed key locally that is not in .gitflow
// 2. Runs 'git flow config status'
// 3. Verifies out-of-sync, names the stale key, exit 6
func TestConfigStatusDriftStaleLocalKey(t *testing.T) {
	t.Parallel()
	dir := initSharedDefaults(t)
	defer testutil.CleanupTestRepo(t, dir)
	if _, err := testutil.RunGit(t, dir, "config", "--local", "gitflow.branch.oldtype.type", "topic"); err != nil {
		t.Fatalf("set stale local key: %v", err)
	}

	out, err := testutil.RunGitFlow(t, dir, "config", "status")
	if err == nil {
		t.Fatalf("expected drift for stale local key, got success\n%s", out)
	}
	assertExitCode(t, err, errors.ExitCodeValidationError)
	if !strings.Contains(out, "gitflow.branch.oldtype.type") {
		t.Errorf("expected drift to name gitflow.branch.oldtype.type, got: %s", out)
	}
}

// TestConfigStatusDriftReorderedMultiValue covers scenario 24d: status drift for
// a reordered multi-value publish.push-option.
// Steps:
// 1. init --shared --defaults, set .gitflow push-option [a,b] and sync, then set local to [b,a]
// 2. Runs 'git flow config status'
// 3. Verifies out-of-sync (order-sensitive), names push-option, exit 6
func TestConfigStatusDriftReorderedMultiValue(t *testing.T) {
	t.Parallel()
	dir := initSharedDefaults(t)
	defer testutil.CleanupTestRepo(t, dir)
	sharedPath := testutil.SharedConfigPath(dir)
	if _, err := testutil.RunGit(t, dir, "config", "--file", sharedPath, "--add", "gitflow.feature.publish.push-option", "a"); err != nil {
		t.Fatalf("add file push-option a: %v", err)
	}
	if _, err := testutil.RunGit(t, dir, "config", "--file", sharedPath, "--add", "gitflow.feature.publish.push-option", "b"); err != nil {
		t.Fatalf("add file push-option b: %v", err)
	}
	if out, err := testutil.RunGitFlow(t, dir, "config", "sync"); err != nil {
		t.Fatalf("sync failed: %v\n%s", err, out)
	}
	// Reverse the local order to [b, a].
	if _, err := testutil.RunGit(t, dir, "config", "--local", "--unset-all", "gitflow.feature.publish.push-option"); err != nil {
		t.Fatalf("unset local push-option: %v", err)
	}
	if _, err := testutil.RunGit(t, dir, "config", "--local", "--add", "gitflow.feature.publish.push-option", "b"); err != nil {
		t.Fatalf("add local push-option b: %v", err)
	}
	if _, err := testutil.RunGit(t, dir, "config", "--local", "--add", "gitflow.feature.publish.push-option", "a"); err != nil {
		t.Fatalf("add local push-option a: %v", err)
	}

	out, err := testutil.RunGitFlow(t, dir, "config", "status")
	if err == nil {
		t.Fatalf("expected drift for reordered multi-value, got success\n%s", out)
	}
	assertExitCode(t, err, errors.ExitCodeValidationError)
	if !strings.Contains(out, "gitflow.feature.publish.push-option") {
		t.Errorf("expected drift to name gitflow.feature.publish.push-option, got: %s", out)
	}
}

// TestConfigStatusNoSharedConfig covers scenario 25a: with no .gitflow, status
// reports "no shared config", exits 0, and does not mutate anything.
// Steps:
// 1. init --defaults (local only, no .gitflow)
// 2. Runs 'git flow config status'
// 3. Verifies "no shared config", exit 0, no .gitflow created
func TestConfigStatusNoSharedConfig(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	if out, err := testutil.RunGitFlow(t, dir, "init", "--defaults"); err != nil {
		t.Fatalf("init --defaults failed: %v\n%s", err, out)
	}

	out, err := testutil.RunGitFlow(t, dir, "config", "status")
	if err != nil {
		t.Fatalf("expected config status to exit 0 with no shared config, got: %v\n%s", err, out)
	}
	if !strings.Contains(out, "No shared") {
		t.Errorf("expected a 'no shared config' message, got: %s", out)
	}
	if _, statErr := os.Stat(testutil.SharedConfigPath(dir)); statErr == nil {
		t.Error("expected no .gitflow to be created by config status")
	}
}

// TestConfigSyncNoSharedConfig covers scenario 25b: with no .gitflow, sync reports
// "no shared config", exits 0, and does not mutate anything.
// Steps:
// 1. init --defaults (local only, no .gitflow)
// 2. Runs 'git flow config sync'
// 3. Verifies "no shared config", exit 0, no .gitflow created
func TestConfigSyncNoSharedConfig(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	if out, err := testutil.RunGitFlow(t, dir, "init", "--defaults"); err != nil {
		t.Fatalf("init --defaults failed: %v\n%s", err, out)
	}

	out, err := testutil.RunGitFlow(t, dir, "config", "sync")
	if err != nil {
		t.Fatalf("expected config sync to exit 0 with no shared config, got: %v\n%s", err, out)
	}
	if !strings.Contains(out, "No shared") {
		t.Errorf("expected a 'no shared config' message, got: %s", out)
	}
	if _, statErr := os.Stat(testutil.SharedConfigPath(dir)); statErr == nil {
		t.Error("expected no .gitflow to be created by config sync")
	}
}

// TestSharedFileMalformedClearErrorSync covers scenario 35b: a malformed .gitflow
// fails at config sync with a clear error naming the file.
// Steps:
// 1. Writes an unparsable .gitflow in a fresh repo
// 2. Runs 'git flow config sync'
// 3. Verifies non-zero exit, an error naming .gitflow, and no partial local mutation
func TestSharedFileMalformedClearErrorSync(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	if err := os.WriteFile(testutil.SharedConfigPath(dir), []byte("[gitflow\nbroken = = =\n"), 0644); err != nil {
		t.Fatalf("write malformed .gitflow: %v", err)
	}

	out, err := testutil.RunGitFlow(t, dir, "config", "sync")
	if err == nil {
		t.Fatalf("expected config sync to fail on malformed .gitflow, got success\n%s", out)
	}
	if !strings.Contains(out, ".gitflow") {
		t.Errorf("expected error naming .gitflow, got: %s", out)
	}
	if testutil.GitConfigHasPrefix(t, dir, "gitflow.branch.") {
		t.Error("expected no partial gitflow.branch.* mutation on malformed-file sync")
	}
}

// TestConfigSyncSkipsUntrustedHookPath covers scenario 37: config sync applies the
// hook-trust filter, not only first-run.
// Steps:
// 1. init --shared --defaults, add gitflow.path.hooks only to .gitflow, trust unset
// 2. Runs 'git flow config sync'
// 3. Verifies local did not gain the hook path, a warning printed, other keys synced
func TestConfigSyncSkipsUntrustedHookPath(t *testing.T) {
	t.Parallel()
	dir := initSharedDefaults(t)
	defer testutil.CleanupTestRepo(t, dir)
	testutil.SharedConfigSet(t, dir, "gitflow.path.hooks", "/some/hooks")
	testutil.SharedConfigSet(t, dir, "gitflow.feature.start.fetch", "true")

	out, err := testutil.RunGitFlow(t, dir, "config", "sync")
	if err != nil {
		t.Fatalf("config sync failed: %v\n%s", err, out)
	}
	if testutil.GitConfigExists(t, dir, "gitflow.path.hooks") {
		t.Error("expected local to NOT gain gitflow.path.hooks when untrusted")
	}
	if !strings.Contains(out, "gitflow.shared.trustHooks") {
		t.Errorf("expected a warning naming gitflow.shared.trustHooks, got: %s", out)
	}
	if v := testutil.GitConfigValue(t, dir, "gitflow.feature.start.fetch"); v != "true" {
		t.Errorf("expected other keys synced (start.fetch=true), got %q", v)
	}
}

// TestConfigStatusUntrustedHookNotDrift covers scenario 37b: status does not flag
// an intentionally-skipped untrusted hook path as drift.
// Steps:
// 1. init --shared --defaults, add gitflow.path.hooks only to .gitflow, trust unset, sync (skips it)
// 2. Runs 'git flow config status'
// 3. Verifies IN SYNC, exit 0, and a security note (not a drift key)
func TestConfigStatusUntrustedHookNotDrift(t *testing.T) {
	t.Parallel()
	dir := initSharedDefaults(t)
	defer testutil.CleanupTestRepo(t, dir)
	testutil.SharedConfigSet(t, dir, "gitflow.path.hooks", "/some/hooks")
	if out, err := testutil.RunGitFlow(t, dir, "config", "sync"); err != nil {
		t.Fatalf("config sync failed: %v\n%s", err, out)
	}

	out, err := testutil.RunGitFlow(t, dir, "config", "status")
	if err != nil {
		t.Fatalf("expected status to exit 0 (skipped hook is not drift), got: %v\n%s", err, out)
	}
	if !strings.Contains(out, "in sync") {
		t.Errorf("expected 'in sync', got: %s", out)
	}
	if !strings.Contains(out, "gitflow.path.hooks") {
		t.Errorf("expected a security note mentioning gitflow.path.hooks, got: %s", out)
	}
}

// TestSharedSyncRemovesHookPathAfterTrustRevoked covers scenario 17d: revoking
// trustHooks removes a previously-copied hook path on the next sync.
// Steps:
// 1. init --shared --defaults, add gitflow.path.hooks, set trustHooks=true, sync (copies it)
// 2. Unset trustHooks and run 'git flow config sync' again
// 3. Verifies the hook path is removed locally, other keys remain, a warning printed
func TestSharedSyncRemovesHookPathAfterTrustRevoked(t *testing.T) {
	t.Parallel()
	dir := initSharedDefaults(t)
	defer testutil.CleanupTestRepo(t, dir)
	testutil.SharedConfigSet(t, dir, "gitflow.path.hooks", "/some/hooks")
	if _, err := testutil.RunGit(t, dir, "config", "--local", "gitflow.shared.trustHooks", "true"); err != nil {
		t.Fatalf("set trustHooks: %v", err)
	}
	if out, err := testutil.RunGitFlow(t, dir, "config", "sync"); err != nil {
		t.Fatalf("first sync failed: %v\n%s", err, out)
	}
	if v := testutil.GitConfigValue(t, dir, "gitflow.path.hooks"); v != "/some/hooks" {
		t.Fatalf("precondition: expected hook path copied when trusted, got %q", v)
	}

	if _, err := testutil.RunGit(t, dir, "config", "--local", "--unset", "gitflow.shared.trustHooks"); err != nil {
		t.Fatalf("unset trustHooks: %v", err)
	}
	out, err := testutil.RunGitFlow(t, dir, "config", "sync")
	if err != nil {
		t.Fatalf("second sync failed: %v\n%s", err, out)
	}
	if testutil.GitConfigExists(t, dir, "gitflow.path.hooks") {
		t.Error("expected local gitflow.path.hooks removed after trust revoked")
	}
	if v := testutil.GitConfigValue(t, dir, "gitflow.branch.feature.prefix"); v != "feature/" {
		t.Errorf("expected other keys to remain, got feature.prefix=%q", v)
	}
	if !strings.Contains(out, "gitflow.shared.trustHooks") {
		t.Errorf("expected a warning naming gitflow.shared.trustHooks, got: %s", out)
	}
}
