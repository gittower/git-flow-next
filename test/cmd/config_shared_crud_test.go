package cmd_test

import (
	"os"
	"strings"
	"testing"

	"github.com/gittower/git-flow-next/test/testutil"
)

// TestConfigEditTopicSharedUpdatesFileAndLocal covers scenario 26: config edit
// topic --shared updates .gitflow, re-syncs local, and the new prefix takes effect.
// Steps:
// 1. init --shared --defaults
// 2. Runs 'config edit topic feature --shared --prefix feat/', then 'feature start x'
// 3. Verifies .gitflow and local prefix are feat/ and feat/x is created
func TestConfigEditTopicSharedUpdatesFileAndLocal(t *testing.T) {
	t.Parallel()
	dir := initSharedDefaults(t)
	defer testutil.CleanupTestRepo(t, dir)

	if out, err := testutil.RunGitFlow(t, dir, "config", "edit", "topic", "feature", "--shared", "--prefix", "feat/"); err != nil {
		t.Fatalf("config edit topic --shared failed: %v\n%s", err, out)
	}
	if v := testutil.SharedConfigValue(t, dir, "gitflow.branch.feature.prefix"); v != "feat/" {
		t.Errorf("expected .gitflow feature.prefix=feat/, got %q", v)
	}
	if v := testutil.GitConfigValue(t, dir, "gitflow.branch.feature.prefix"); v != "feat/" {
		t.Errorf("expected local feature.prefix=feat/, got %q", v)
	}
	if out, err := testutil.RunGitFlow(t, dir, "feature", "start", "x"); err != nil {
		t.Fatalf("feature start failed: %v\n%s", err, out)
	}
	if !testutil.BranchExists(t, dir, "feat/x") {
		t.Error("expected feat/x to be created with the new prefix")
	}
}

// TestConfigAddTopicSharedAddsTypeAndSyncs covers scenario 27: config add topic
// --shared adds a type to .gitflow, re-syncs local, and the new type works.
// Steps:
// 1. init --shared --defaults
// 2. Runs 'config add topic qa develop --shared', then 'qa start x'
// 3. Verifies .gitflow and local carry gitflow.branch.qa.* and qa/x is created
func TestConfigAddTopicSharedAddsTypeAndSyncs(t *testing.T) {
	t.Parallel()
	dir := initSharedDefaults(t)
	defer testutil.CleanupTestRepo(t, dir)

	if out, err := testutil.RunGitFlow(t, dir, "config", "add", "topic", "qa", "develop", "--shared"); err != nil {
		t.Fatalf("config add topic --shared failed: %v\n%s", err, out)
	}
	if v := testutil.SharedConfigValue(t, dir, "gitflow.branch.qa.type"); v != "topic" {
		t.Errorf("expected .gitflow qa.type=topic, got %q", v)
	}
	if !testutil.GitConfigHasPrefix(t, dir, "gitflow.branch.qa.") {
		t.Error("expected local re-synced with gitflow.branch.qa.*")
	}
	if out, err := testutil.RunGitFlow(t, dir, "qa", "start", "x"); err != nil {
		t.Fatalf("qa start failed: %v\n%s", err, out)
	}
	if !testutil.BranchExists(t, dir, "qa/x") {
		t.Error("expected qa/x to be created")
	}
}

// TestConfigRenameTopicSharedFileOnly covers scenario 28a: config rename topic
// --shared edits only .gitflow + re-syncs; a local-only control key is preserved
// and a local-only shared-managed key is stale-removed (not pushed up).
// Steps:
// 1. init --shared --defaults, set local-only gitflow.shared.autoInit and gitflow.feature.start.fetch
// 2. Runs 'config rename topic feature feat --shared'
// 3. Verifies .gitflow renamed feature->feat, local synced, autoInit preserved, start.fetch removed and not in .gitflow
func TestConfigRenameTopicSharedFileOnly(t *testing.T) {
	t.Parallel()
	dir := initSharedDefaults(t)
	defer testutil.CleanupTestRepo(t, dir)
	if _, err := testutil.RunGit(t, dir, "config", "--local", "gitflow.shared.autoInit", "true"); err != nil {
		t.Fatalf("set autoInit: %v", err)
	}
	if _, err := testutil.RunGit(t, dir, "config", "--local", "gitflow.feature.start.fetch", "true"); err != nil {
		t.Fatalf("set local start.fetch: %v", err)
	}

	if out, err := testutil.RunGitFlow(t, dir, "config", "rename", "topic", "feature", "feat", "--shared"); err != nil {
		t.Fatalf("config rename topic --shared failed: %v\n%s", err, out)
	}

	if !testutil.SharedConfigHasPrefix(t, dir, "gitflow.branch.feat.") {
		t.Error("expected .gitflow to have gitflow.branch.feat.*")
	}
	if testutil.SharedConfigHasPrefix(t, dir, "gitflow.branch.feature.") {
		t.Error("expected .gitflow to no longer have gitflow.branch.feature.*")
	}
	if !testutil.GitConfigHasPrefix(t, dir, "gitflow.branch.feat.") {
		t.Error("expected local re-synced with gitflow.branch.feat.*")
	}
	if testutil.GitConfigHasPrefix(t, dir, "gitflow.branch.feature.") {
		t.Error("expected local gitflow.branch.feature.* removed by re-sync")
	}
	// Control key: never shared-managed, preserved locally, not written to .gitflow.
	if v := testutil.GitConfigValue(t, dir, "gitflow.shared.autoInit"); v != "true" {
		t.Errorf("expected local autoInit preserved, got %q", v)
	}
	if testutil.SharedConfigValue(t, dir, "gitflow.shared.autoInit") != "" {
		t.Error("expected gitflow.shared.autoInit NOT written into .gitflow")
	}
	// Local-only shared-managed key: stale-removed and never pushed up.
	if testutil.GitConfigExists(t, dir, "gitflow.feature.start.fetch") {
		t.Error("expected local-only gitflow.feature.start.fetch removed by re-sync")
	}
	if testutil.SharedConfigValue(t, dir, "gitflow.feature.start.fetch") != "" {
		t.Error("expected gitflow.feature.start.fetch NOT pushed up into .gitflow")
	}
}

// TestConfigDeleteTopicSharedFileOnly covers scenario 28b: config delete topic
// --shared removes the section from .gitflow and re-syncs local.
// Steps:
// 1. init --shared --defaults, add a qa topic via --shared
// 2. Runs 'config delete topic qa --shared'
// 3. Verifies qa.* gone from both .gitflow and local, feature defaults remain in both
func TestConfigDeleteTopicSharedFileOnly(t *testing.T) {
	t.Parallel()
	dir := initSharedDefaults(t)
	defer testutil.CleanupTestRepo(t, dir)
	if out, err := testutil.RunGitFlow(t, dir, "config", "add", "topic", "qa", "develop", "--shared"); err != nil {
		t.Fatalf("config add topic qa --shared failed: %v\n%s", err, out)
	}

	if out, err := testutil.RunGitFlow(t, dir, "config", "delete", "topic", "qa", "--shared"); err != nil {
		t.Fatalf("config delete topic qa --shared failed: %v\n%s", err, out)
	}
	if testutil.SharedConfigHasPrefix(t, dir, "gitflow.branch.qa.") {
		t.Error("expected .gitflow to no longer have gitflow.branch.qa.*")
	}
	if testutil.GitConfigHasPrefix(t, dir, "gitflow.branch.qa.") {
		t.Error("expected local to no longer have gitflow.branch.qa.*")
	}
	if testutil.SharedConfigValue(t, dir, "gitflow.branch.feature.type") != "topic" {
		t.Error("expected feature defaults to remain in .gitflow")
	}
	if testutil.GitConfigValue(t, dir, "gitflow.branch.feature.type") != "topic" {
		t.Error("expected feature defaults to remain in local")
	}
}

// TestConfigEditLocalOnlyCausesDrift covers scenario 29: config edit WITHOUT
// --shared writes local only; status then reports drift.
// Steps:
// 1. init --shared --defaults
// 2. Runs 'config edit topic feature --prefix feat/' (no --shared), then 'config status'
// 3. Verifies .gitflow unchanged, local is feat/, and status reports drift with exit 6
func TestConfigEditLocalOnlyCausesDrift(t *testing.T) {
	t.Parallel()
	dir := initSharedDefaults(t)
	defer testutil.CleanupTestRepo(t, dir)

	if out, err := testutil.RunGitFlow(t, dir, "config", "edit", "topic", "feature", "--prefix", "feat/"); err != nil {
		t.Fatalf("config edit topic (local) failed: %v\n%s", err, out)
	}
	if v := testutil.SharedConfigValue(t, dir, "gitflow.branch.feature.prefix"); v != "feature/" {
		t.Errorf("expected .gitflow feature.prefix unchanged (feature/), got %q", v)
	}
	if v := testutil.GitConfigValue(t, dir, "gitflow.branch.feature.prefix"); v != "feat/" {
		t.Errorf("expected local feature.prefix=feat/, got %q", v)
	}

	out, err := testutil.RunGitFlow(t, dir, "config", "status")
	if err == nil {
		t.Fatalf("expected drift after a local-only edit, got success\n%s", out)
	}
	assertExitCode(t, err, 6)
}

// TestConfigAddTopicSharedWithoutFileErrors covers scenario 30: config add topic
// --shared without a .gitflow errors, suggests init --shared, and creates no file.
// Steps:
// 1. init --defaults (local only, no .gitflow)
// 2. Runs 'config add topic qa develop --shared'
// 3. Verifies non-zero exit, a message suggesting init --shared, and no .gitflow created
func TestConfigAddTopicSharedWithoutFileErrors(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	if out, err := testutil.RunGitFlow(t, dir, "init", "--defaults"); err != nil {
		t.Fatalf("init --defaults failed: %v\n%s", err, out)
	}

	out, err := testutil.RunGitFlow(t, dir, "config", "add", "topic", "qa", "develop", "--shared")
	if err == nil {
		t.Fatalf("expected --shared without .gitflow to fail, got success\n%s", out)
	}
	if !strings.Contains(out, "init --shared") {
		t.Errorf("expected a message suggesting 'git flow init --shared', got: %s", out)
	}
	if _, statErr := os.Stat(testutil.SharedConfigPath(dir)); statErr == nil {
		t.Error("expected no .gitflow to be created")
	}
}

// TestConfigEditTopicSharedFreshCloneNoLocalInit covers Imp-4: shared CRUD must
// not be gated on local git-flow init. On a FRESH CLONE (committed .gitflow with a
// feature type, NO local init), 'config edit topic feature --shared --prefix=feat/'
// must succeed, update .gitflow, and re-sync local.
// Steps:
// 1. Fresh clone carrying a committed .gitflow, no local gitflow.* keys
// 2. Runs 'config edit topic feature --shared --prefix=feat/'
// 3. Verifies success, .gitflow feature.prefix=feat/, and local re-synced to feat/
func TestConfigEditTopicSharedFreshCloneNoLocalInit(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupFreshCloneWithShared(t, testutil.AuthorSharedConfig(t))
	defer testutil.CleanupTestRepo(t, dir)

	if out, err := testutil.RunGitFlow(t, dir, "config", "edit", "topic", "feature", "--shared", "--prefix=feat/"); err != nil {
		t.Fatalf("expected shared edit to succeed without local init, got: %v\n%s", err, out)
	}
	if v := testutil.SharedConfigValue(t, dir, "gitflow.branch.feature.prefix"); v != "feat/" {
		t.Errorf("expected .gitflow feature.prefix=feat/, got %q", v)
	}
	if v := testutil.GitConfigValue(t, dir, "gitflow.branch.feature.prefix"); v != "feat/" {
		t.Errorf("expected local re-synced feature.prefix=feat/, got %q", v)
	}
}

// TestConfigAddBaseSharedAddsBaseAndSyncs covers scenario 30base-a: config add
// base --shared adds a base branch to .gitflow, re-syncs local, and creates the branch.
// Steps:
// 1. init --shared --defaults
// 2. Runs 'config add base staging main --shared'
// 3. Verifies .gitflow and local carry gitflow.branch.staging.* (parent=main) and staging exists
func TestConfigAddBaseSharedAddsBaseAndSyncs(t *testing.T) {
	t.Parallel()
	dir := initSharedDefaults(t)
	defer testutil.CleanupTestRepo(t, dir)

	if out, err := testutil.RunGitFlow(t, dir, "config", "add", "base", "staging", "main", "--shared"); err != nil {
		t.Fatalf("config add base --shared failed: %v\n%s", err, out)
	}
	if v := testutil.SharedConfigValue(t, dir, "gitflow.branch.staging.type"); v != "base" {
		t.Errorf("expected .gitflow staging.type=base, got %q", v)
	}
	if v := testutil.SharedConfigValue(t, dir, "gitflow.branch.staging.parent"); v != "main" {
		t.Errorf("expected .gitflow staging.parent=main, got %q", v)
	}
	if v := testutil.GitConfigValue(t, dir, "gitflow.branch.staging.parent"); v != "main" {
		t.Errorf("expected local staging.parent=main, got %q", v)
	}
	if !testutil.BranchExists(t, dir, "staging") {
		t.Error("expected the staging branch to be created")
	}
}

// TestConfigEditBaseSharedUpdatesFileAndLocal covers scenario 30base-b: config
// edit base --shared updates .gitflow and re-syncs local.
// Steps:
// 1. init --shared --defaults (develop has autoUpdate)
// 2. Runs 'config edit base develop --auto-update=false --shared'
// 3. Verifies develop.autoUpdate=false in both .gitflow and local
func TestConfigEditBaseSharedUpdatesFileAndLocal(t *testing.T) {
	t.Parallel()
	dir := initSharedDefaults(t)
	defer testutil.CleanupTestRepo(t, dir)

	if out, err := testutil.RunGitFlow(t, dir, "config", "edit", "base", "develop", "--auto-update=false", "--shared"); err != nil {
		t.Fatalf("config edit base --shared failed: %v\n%s", err, out)
	}
	if v := testutil.SharedConfigValue(t, dir, "gitflow.branch.develop.autoUpdate"); v != "false" {
		t.Errorf("expected .gitflow develop.autoUpdate=false, got %q", v)
	}
	if v := testutil.GitConfigValue(t, dir, "gitflow.branch.develop.autoupdate"); v != "false" {
		t.Errorf("expected local develop.autoUpdate=false, got %q", v)
	}
}

// TestConfigRenameBaseSharedFileOnly covers scenario 30base-c: config rename base
// --shared renames the section in .gitflow and re-syncs local.
// Steps:
// 1. init --shared --defaults, add a staging base via --shared
// 2. Runs 'config rename base staging stage --shared'
// 3. Verifies .gitflow and local have stage.* and not staging.*
func TestConfigRenameBaseSharedFileOnly(t *testing.T) {
	t.Parallel()
	dir := initSharedDefaults(t)
	defer testutil.CleanupTestRepo(t, dir)
	if out, err := testutil.RunGitFlow(t, dir, "config", "add", "base", "staging", "main", "--shared"); err != nil {
		t.Fatalf("config add base staging --shared failed: %v\n%s", err, out)
	}

	if out, err := testutil.RunGitFlow(t, dir, "config", "rename", "base", "staging", "stage", "--shared"); err != nil {
		t.Fatalf("config rename base --shared failed: %v\n%s", err, out)
	}
	if !testutil.SharedConfigHasPrefix(t, dir, "gitflow.branch.stage.") {
		t.Error("expected .gitflow to have gitflow.branch.stage.*")
	}
	if testutil.SharedConfigHasPrefix(t, dir, "gitflow.branch.staging.") {
		t.Error("expected .gitflow to no longer have gitflow.branch.staging.*")
	}
	if !testutil.GitConfigHasPrefix(t, dir, "gitflow.branch.stage.") {
		t.Error("expected local to have gitflow.branch.stage.*")
	}
	if testutil.GitConfigHasPrefix(t, dir, "gitflow.branch.staging.") {
		t.Error("expected local to no longer have gitflow.branch.staging.*")
	}
}

// TestConfigDeleteBaseSharedFileOnly covers scenario 30base-d: config delete base
// --shared removes the section from .gitflow and re-syncs local.
// Steps:
// 1. init --shared --defaults, add a staging base via --shared
// 2. Runs 'config delete base staging --shared'
// 3. Verifies staging.* gone from both, and core bases main/develop remain in both
func TestConfigDeleteBaseSharedFileOnly(t *testing.T) {
	t.Parallel()
	dir := initSharedDefaults(t)
	defer testutil.CleanupTestRepo(t, dir)
	if out, err := testutil.RunGitFlow(t, dir, "config", "add", "base", "staging", "main", "--shared"); err != nil {
		t.Fatalf("config add base staging --shared failed: %v\n%s", err, out)
	}

	if out, err := testutil.RunGitFlow(t, dir, "config", "delete", "base", "staging", "--shared"); err != nil {
		t.Fatalf("config delete base --shared failed: %v\n%s", err, out)
	}
	if testutil.SharedConfigHasPrefix(t, dir, "gitflow.branch.staging.") {
		t.Error("expected .gitflow to no longer have gitflow.branch.staging.*")
	}
	if testutil.GitConfigHasPrefix(t, dir, "gitflow.branch.staging.") {
		t.Error("expected local to no longer have gitflow.branch.staging.*")
	}
	for _, base := range []string{"main", "develop"} {
		if testutil.SharedConfigValue(t, dir, "gitflow.branch."+base+".type") != "base" {
			t.Errorf("expected core base %q to remain in .gitflow", base)
		}
		if testutil.GitConfigValue(t, dir, "gitflow.branch."+base+".type") != "base" {
			t.Errorf("expected core base %q to remain in local", base)
		}
	}
}
