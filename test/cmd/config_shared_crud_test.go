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

// TestConfigNestedLeafFreshCloneNoActivation guards the first-run exemption for
// NESTED config leaves: `config add topic` sits two levels below `config`, so a
// direct-parent-only exemption would let first-run activation fire for it. On a
// fresh clone that has autoInit=true set locally, a wrongly-fired activation
// would auto-copy the shared config and print an "auto-initialized" notice
// before the command runs. The config verb must stay exempt: no notice, and the
// edit targets .gitflow directly.
// Steps:
// 1. Fresh clone carrying a committed .gitflow, no local gitflow.* keys, autoInit=true local
// 2. Runs 'config add topic qa develop --shared'
// 3. Verifies no auto-init notice, success, and gitflow.branch.qa lands in .gitflow
func TestConfigNestedLeafFreshCloneNoActivation(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupFreshCloneWithShared(t, testutil.AuthorSharedConfig(t))
	defer testutil.CleanupTestRepo(t, dir)

	// autoInit is a local control key, so it survives ClearLocalGitflowConfig's
	// job here: if activation wrongly fired it would take the auto-init branch.
	if _, err := testutil.RunGit(t, dir, "config", "--local", "gitflow.shared.autoInit", "true"); err != nil {
		t.Fatalf("failed to set autoInit: %v", err)
	}

	out, err := testutil.RunGitFlow(t, dir, "config", "add", "topic", "qa", "develop", "--shared")
	if err != nil {
		t.Fatalf("config add topic --shared on fresh clone failed: %v\n%s", err, out)
	}
	if strings.Contains(out, "auto-initialized") {
		t.Errorf("expected no first-run activation for a nested config command, got: %s", out)
	}
	if v := testutil.SharedConfigValue(t, dir, "gitflow.branch.qa.type"); v != "topic" {
		t.Errorf("expected .gitflow qa.type=topic, got %q", v)
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

// TestConfigEditTopicSharedTagFalsePersists covers scenario 8: an explicit
// --tag=false with --shared must clear the stored true in .gitflow and land in
// local through the shared→local re-sync.
// Steps:
// 1. init --shared --defaults (.gitflow has release.tag=true)
// 2. Runs 'config edit topic release --tag=false --shared'
// 3. Verifies release.tag is "false" in both .gitflow and local
func TestConfigEditTopicSharedTagFalsePersists(t *testing.T) {
	t.Parallel()
	dir := initSharedDefaults(t)
	defer testutil.CleanupTestRepo(t, dir)

	out, err := testutil.RunGitFlow(t, dir, "config", "edit", "topic", "release", "--tag=false", "--shared")
	if err != nil {
		t.Fatalf("config edit topic --tag=false --shared failed: %v\n%s", err, out)
	}

	if v := testutil.SharedConfigValue(t, dir, "gitflow.branch.release.tag"); v != "false" {
		t.Errorf("expected .gitflow release.tag=false, got %q\n%s", v, out)
	}
	if v := testutil.GitConfigValue(t, dir, "gitflow.branch.release.tag"); v != "false" {
		t.Errorf("expected local release.tag=false after re-sync, got %q\n%s", v, out)
	}
}

// TestConfigEditTopicSharedPreservesTagWhenFlagOmitted covers scenario 9: an
// omitted --tag with --shared must preserve the stored value in both stores.
// Steps:
// 1. init --shared --defaults (.gitflow has release.tag=true)
// 2. Runs 'config edit topic release --prefix=rel/ --shared' without --tag
// 3. Verifies release.prefix is "rel/" and release.tag is still "true" in both stores
func TestConfigEditTopicSharedPreservesTagWhenFlagOmitted(t *testing.T) {
	t.Parallel()
	dir := initSharedDefaults(t)
	defer testutil.CleanupTestRepo(t, dir)

	out, err := testutil.RunGitFlow(t, dir, "config", "edit", "topic", "release", "--prefix=rel/", "--shared")
	if err != nil {
		t.Fatalf("config edit topic --prefix=rel/ --shared failed: %v\n%s", err, out)
	}

	if v := testutil.SharedConfigValue(t, dir, "gitflow.branch.release.prefix"); v != "rel/" {
		t.Errorf("expected .gitflow release.prefix=rel/, got %q\n%s", v, out)
	}
	if v := testutil.GitConfigValue(t, dir, "gitflow.branch.release.prefix"); v != "rel/" {
		t.Errorf("expected local release.prefix=rel/, got %q\n%s", v, out)
	}
	if v := testutil.SharedConfigValue(t, dir, "gitflow.branch.release.tag"); v != "true" {
		t.Errorf("expected omitted --tag to preserve .gitflow release.tag=true, got %q\n%s", v, out)
	}
	if v := testutil.GitConfigValue(t, dir, "gitflow.branch.release.tag"); v != "true" {
		t.Errorf("expected omitted --tag to preserve local release.tag=true, got %q\n%s", v, out)
	}
}

// TestConfigEditBaseSharedPreservesAutoUpdateWhenFlagOmitted covers scenario 10:
// an omitted --auto-update with --shared must preserve the stored value in both
// stores, and non-tagging types gain an explicit tag=false in both.
// Steps:
// 1. init --shared --defaults (develop.autoUpdate=true)
// 2. Runs 'config edit base develop --upstream-strategy=merge --shared' without --auto-update
// 3. Verifies develop.upstreamStrategy is "merge" and develop.autoUpdate is still "true" in both stores
// 4. Verifies feature.tag is the explicit "false" in both stores
func TestConfigEditBaseSharedPreservesAutoUpdateWhenFlagOmitted(t *testing.T) {
	t.Parallel()
	dir := initSharedDefaults(t)
	defer testutil.CleanupTestRepo(t, dir)

	out, err := testutil.RunGitFlow(t, dir, "config", "edit", "base", "develop", "--upstream-strategy=merge", "--shared")
	if err != nil {
		t.Fatalf("config edit base --upstream-strategy=merge --shared failed: %v\n%s", err, out)
	}

	if v := testutil.SharedConfigValue(t, dir, "gitflow.branch.develop.upstreamStrategy"); v != "merge" {
		t.Errorf("expected .gitflow develop.upstreamStrategy=merge, got %q\n%s", v, out)
	}
	if v := testutil.GitConfigValue(t, dir, "gitflow.branch.develop.upstreamStrategy"); v != "merge" {
		t.Errorf("expected local develop.upstreamStrategy=merge, got %q\n%s", v, out)
	}
	if v := testutil.SharedConfigValue(t, dir, "gitflow.branch.develop.autoUpdate"); v != "true" {
		t.Errorf("expected omitted --auto-update to preserve .gitflow develop.autoUpdate=true, got %q\n%s", v, out)
	}
	if v := testutil.GitConfigValue(t, dir, "gitflow.branch.develop.autoUpdate"); v != "true" {
		t.Errorf("expected omitted --auto-update to preserve local develop.autoUpdate=true, got %q\n%s", v, out)
	}
	if v := testutil.SharedConfigValue(t, dir, "gitflow.branch.feature.tag"); v != "false" {
		t.Errorf("expected .gitflow feature.tag=false written explicitly, got %q\n%s", v, out)
	}
	if v := testutil.GitConfigValue(t, dir, "gitflow.branch.feature.tag"); v != "false" {
		t.Errorf("expected local feature.tag=false written explicitly, got %q\n%s", v, out)
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
