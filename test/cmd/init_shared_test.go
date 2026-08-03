package cmd_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gittower/git-flow-next/internal/errors"
	"github.com/gittower/git-flow-next/test/testutil"
)

// TestInitSharedCreatesFileAndCopiesLocal covers scenario 1: init --shared
// --defaults creates .gitflow and copies the keys into local config.
// Steps:
// 1. Sets up a fresh repository with no git-flow config
// 2. Runs 'git flow init --shared --defaults'
// 3. Verifies .gitflow exists with version and default branch types
// 4. Verifies local .git/config carries the same gitflow.* keys
// 5. Verifies 'git flow overview' immediately lists the default topic types
func TestInitSharedCreatesFileAndCopiesLocal(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	if out, err := testutil.RunGitFlow(t, dir, "init", "--shared", "--defaults"); err != nil {
		t.Fatalf("init --shared failed: %v\n%s", err, out)
	}

	if _, err := os.Stat(testutil.SharedConfigPath(dir)); err != nil {
		t.Fatalf("expected .gitflow to exist: %v", err)
	}
	if v := testutil.SharedConfigValue(t, dir, "gitflow.version"); v == "" {
		t.Error("expected .gitflow to declare gitflow.version")
	}
	if v := testutil.SharedConfigValue(t, dir, "gitflow.branch.feature.type"); v != "topic" {
		t.Errorf("expected .gitflow feature.type=topic, got %q", v)
	}
	if v := testutil.SharedConfigValue(t, dir, "gitflow.branch.feature.prefix"); v != "feature/" {
		t.Errorf("expected .gitflow feature.prefix=feature/, got %q", v)
	}

	if v := testutil.GitConfigValue(t, dir, "gitflow.branch.feature.prefix"); v != "feature/" {
		t.Errorf("expected local feature.prefix=feature/, got %q", v)
	}

	out, err := testutil.RunGitFlow(t, dir, "overview")
	if err != nil {
		t.Fatalf("overview failed: %v\n%s", err, out)
	}
	for _, want := range []string{"feature", "release", "hotfix"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected overview to list %q, got: %s", want, out)
		}
	}
}

// TestInitSharedWritesAtToplevelFromSubdir covers scenario 2: init --shared from
// a subdirectory writes .gitflow at the repository toplevel.
// Steps:
// 1. Sets up a repository and creates a sub/ directory
// 2. Runs 'git flow init --shared --defaults' from sub/
// 3. Verifies <repo>/.gitflow exists and <repo>/sub/.gitflow does not
func TestInitSharedWritesAtToplevelFromSubdir(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	subDir := filepath.Join(dir, "sub")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("failed to create sub dir: %v", err)
	}

	if out, err := testutil.RunGitFlow(t, subDir, "init", "--shared", "--defaults"); err != nil {
		t.Fatalf("init --shared from subdir failed: %v\n%s", err, out)
	}

	if _, err := os.Stat(filepath.Join(dir, ".gitflow")); err != nil {
		t.Errorf("expected .gitflow at toplevel: %v", err)
	}
	if _, err := os.Stat(filepath.Join(subDir, ".gitflow")); err == nil {
		t.Error("expected no .gitflow in the subdirectory")
	}
}

// TestInitSharedRejectsOtherScopeFlags covers scenario 3: --shared is mutually
// exclusive with the other scope flags.
// Steps:
// 1. Sets up a fresh repository
// 2. Runs 'git flow init --shared --local --defaults'
// 3. Verifies exit code ExitCodeInvalidInput, an error naming --shared, and no .gitflow
func TestInitSharedRejectsOtherScopeFlags(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	out, err := testutil.RunGitFlow(t, dir, "init", "--shared", "--local", "--defaults")
	if err == nil {
		t.Fatalf("expected error for mutually exclusive scope flags, got success\n%s", out)
	}
	if ee, ok := err.(*testutil.ExitError); ok {
		if ee.ExitCode != int(errors.ExitCodeInvalidInput) {
			t.Errorf("expected exit code %d, got %d", errors.ExitCodeInvalidInput, ee.ExitCode)
		}
	} else {
		t.Fatalf("expected ExitError, got %T", err)
	}
	if !strings.Contains(out, "--shared") {
		t.Errorf("expected error to name --shared, got: %s", out)
	}
	if _, err := os.Stat(filepath.Join(dir, ".gitflow")); err == nil {
		t.Error("expected no .gitflow to be created on a rejected invocation")
	}
}

// TestInitSharedExistingFileWithoutForceNoOp covers scenario 4: with an existing
// .gitflow, init --shared without --force is a no-op.
// Steps:
// 1. Runs init --shared --defaults, then edits .gitflow feature.prefix to a sentinel
// 2. Captures the current local feature.prefix
// 3. Runs 'git flow init --shared --defaults' again (no --force)
// 4. Verifies non-zero exit, .gitflow still has the sentinel, local prefix unchanged
func TestInitSharedExistingFileWithoutForceNoOp(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	if out, err := testutil.RunGitFlow(t, dir, "init", "--shared", "--defaults"); err != nil {
		t.Fatalf("init --shared failed: %v\n%s", err, out)
	}
	testutil.SharedConfigSet(t, dir, "gitflow.branch.feature.prefix", "sentinel/")
	localBefore := testutil.GitConfigValue(t, dir, "gitflow.branch.feature.prefix")

	out, err := testutil.RunGitFlow(t, dir, "init", "--shared", "--defaults")
	if err == nil {
		t.Fatalf("expected error for existing .gitflow without --force, got success\n%s", out)
	}

	if v := testutil.SharedConfigValue(t, dir, "gitflow.branch.feature.prefix"); v != "sentinel/" {
		t.Errorf("expected .gitflow feature.prefix to stay sentinel/, got %q", v)
	}
	if v := testutil.GitConfigValue(t, dir, "gitflow.branch.feature.prefix"); v != localBefore {
		t.Errorf("expected local feature.prefix unchanged (%q), got %q", localBefore, v)
	}
}

// TestInitSharedForceRewritesAndRecopies covers scenario 5: init --shared --force
// rewrites .gitflow, re-copies into local, and stale-removes an old managed key.
// Steps:
// 1. Runs init --shared --defaults, sets a sentinel prefix in .gitflow, adds a stale local key
// 2. Runs 'git flow init --shared --defaults --force'
// 3. Verifies .gitflow feature.prefix is back to feature/, local matches, stale key removed
func TestInitSharedForceRewritesAndRecopies(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	if out, err := testutil.RunGitFlow(t, dir, "init", "--shared", "--defaults"); err != nil {
		t.Fatalf("init --shared failed: %v\n%s", err, out)
	}
	testutil.SharedConfigSet(t, dir, "gitflow.branch.feature.prefix", "sentinel/")
	if _, err := testutil.RunGit(t, dir, "config", "--local", "gitflow.branch.oldtype.type", "topic"); err != nil {
		t.Fatalf("failed to seed stale local key: %v", err)
	}

	if out, err := testutil.RunGitFlow(t, dir, "init", "--shared", "--defaults", "--force"); err != nil {
		t.Fatalf("init --shared --force failed: %v\n%s", err, out)
	}

	if v := testutil.SharedConfigValue(t, dir, "gitflow.branch.feature.prefix"); v != "feature/" {
		t.Errorf("expected .gitflow feature.prefix rewritten to feature/, got %q", v)
	}
	if v := testutil.GitConfigValue(t, dir, "gitflow.branch.feature.prefix"); v != "feature/" {
		t.Errorf("expected local feature.prefix re-copied to feature/, got %q", v)
	}
	if testutil.GitConfigExists(t, dir, "gitflow.branch.oldtype.type") {
		t.Error("expected stale local gitflow.branch.oldtype.type to be removed on force re-copy")
	}
}
