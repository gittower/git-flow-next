package cmd_test

import (
	"os"
	"strings"
	"testing"

	"github.com/gittower/git-flow-next/internal/errors"
	"github.com/gittower/git-flow-next/test/testutil"
)

// TestFirstRunPromptAcceptActivates covers scenario 6: an interactive first-run
// prompt that is accepted activates the shared config and proceeds.
// Steps:
// 1. Builds a fresh-clone fixture that carries a committed .gitflow but no local config
// 2. Runs 'feature start my-thing' with interactive stdin answering "y"
// 3. Verifies the prompt mentions .gitflow, local config was copied, and the branch exists
func TestFirstRunPromptAcceptActivates(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupFreshCloneWithShared(t, testutil.AuthorSharedConfig(t))
	defer testutil.CleanupTestRepo(t, dir)

	out, err := testutil.RunGitFlowInteractive(t, dir, "y\n", "feature", "start", "my-thing")
	if err != nil {
		t.Fatalf("feature start after accept failed: %v\n%s", err, out)
	}
	if !strings.Contains(out, ".gitflow") {
		t.Errorf("expected prompt to mention .gitflow, got: %s", out)
	}
	if v := testutil.GitConfigValue(t, dir, "gitflow.branch.feature.prefix"); v != "feature/" {
		t.Errorf("expected local feature.prefix copied to feature/, got %q", v)
	}
	if !testutil.BranchExists(t, dir, "feature/my-thing") {
		t.Error("expected feature/my-thing to be created")
	}
}

// TestFirstRunPromptDeclineNoCopy covers scenario 7: an interactive first-run
// prompt that is declined performs no copy and the command fails as uninitialized.
// Steps:
// 1. Builds a fresh-clone fixture with a committed .gitflow
// 2. Runs 'feature start my-thing' with interactive stdin answering "n"
// 3. Verifies no gitflow.* keys were written, no branch created, exit code 1
func TestFirstRunPromptDeclineNoCopy(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupFreshCloneWithShared(t, testutil.AuthorSharedConfig(t))
	defer testutil.CleanupTestRepo(t, dir)

	out, err := testutil.RunGitFlowInteractive(t, dir, "n\n", "feature", "start", "my-thing")
	if err == nil {
		t.Fatalf("expected declined first-run to fail, got success\n%s", out)
	}
	assertExitCode(t, err, errors.ExitCodeNotInitialized)
	if testutil.GitConfigHasPrefix(t, dir, "gitflow.") {
		t.Error("expected no gitflow.* keys in local config after decline")
	}
	if testutil.BranchExists(t, dir, "feature/my-thing") {
		t.Error("expected no branch to be created after decline")
	}
}

// TestFirstRunNonInteractiveHintNoMutation covers scenario 8: a non-interactive
// run with autoInit unset only hints and mutates nothing.
// Steps:
// 1. Builds a fresh-clone fixture with a committed .gitflow, autoInit unset
// 2. Runs 'feature start my-thing' non-interactively (stdin not a TTY)
// 3. Verifies a hint naming .gitflow, no gitflow.* keys, no branch, exit code 1
func TestFirstRunNonInteractiveHintNoMutation(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupFreshCloneWithShared(t, testutil.AuthorSharedConfig(t))
	defer testutil.CleanupTestRepo(t, dir)

	out, err := testutil.RunGitFlow(t, dir, "feature", "start", "my-thing")
	if err == nil {
		t.Fatalf("expected non-interactive uninitialized run to fail, got success\n%s", out)
	}
	assertExitCode(t, err, errors.ExitCodeNotInitialized)
	if !strings.Contains(out, ".gitflow") {
		t.Errorf("expected a hint naming .gitflow, got: %s", out)
	}
	if testutil.GitConfigHasPrefix(t, dir, "gitflow.") {
		t.Error("expected no gitflow.* keys in local config after a hint-only run")
	}
	if testutil.BranchExists(t, dir, "feature/my-thing") {
		t.Error("expected no branch to be created after a hint-only run")
	}
}

// TestFirstRunAutoInitActivatesWithNotice covers scenario 9: with
// gitflow.shared.autoInit=true, a non-interactive run auto-copies with a notice.
// Steps:
// 1. Builds a fresh-clone fixture and sets gitflow.shared.autoInit=true locally
// 2. Runs 'feature start my-thing' non-interactively
// 3. Verifies a one-line auto-init notice, local config copied, and the branch exists
func TestFirstRunAutoInitActivatesWithNotice(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupFreshCloneWithShared(t, testutil.AuthorSharedConfig(t))
	defer testutil.CleanupTestRepo(t, dir)
	if _, err := testutil.RunGit(t, dir, "config", "--local", "gitflow.shared.autoInit", "true"); err != nil {
		t.Fatalf("failed to set autoInit: %v", err)
	}

	out, err := testutil.RunGitFlow(t, dir, "feature", "start", "my-thing")
	if err != nil {
		t.Fatalf("auto-init feature start failed: %v\n%s", err, out)
	}
	if !strings.Contains(out, "auto-initialized") || !strings.Contains(out, ".gitflow") {
		t.Errorf("expected an auto-init notice naming .gitflow, got: %s", out)
	}
	if v := testutil.GitConfigValue(t, dir, "gitflow.branch.feature.prefix"); v != "feature/" {
		t.Errorf("expected local feature.prefix copied, got %q", v)
	}
	if !testutil.BranchExists(t, dir, "feature/my-thing") {
		t.Error("expected feature/my-thing to be created")
	}
}

// TestFirstRunLocalConfigPresentNoPrompt covers scenario 10: an already
// locally-configured repo with a .gitflow present does not prompt; local wins.
// Steps:
// 1. Runs 'init --defaults' (local config, no .gitflow), then drops in a .gitflow with feat/
// 2. Runs 'feature start my-thing'
// 3. Verifies no prompt/notice, the LOCAL prefix feature/ is used, and local prefix unchanged
func TestFirstRunLocalConfigPresentNoPrompt(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	if out, err := testutil.RunGitFlow(t, dir, "init", "--defaults"); err != nil {
		t.Fatalf("init --defaults failed: %v\n%s", err, out)
	}
	// A .gitflow that disagrees with local (feat/ vs feature/), left unsynced.
	testutil.SharedConfigSet(t, dir, "gitflow.version", "1.0")
	testutil.SharedConfigSet(t, dir, "gitflow.branch.feature.type", "topic")
	testutil.SharedConfigSet(t, dir, "gitflow.branch.feature.prefix", "feat/")

	out, err := testutil.RunGitFlow(t, dir, "feature", "start", "my-thing")
	if err != nil {
		t.Fatalf("feature start failed: %v\n%s", err, out)
	}
	if strings.Contains(out, "not yet active") || strings.Contains(out, "auto-initialized") {
		t.Errorf("expected no first-run prompt/notice, got: %s", out)
	}
	if !testutil.BranchExists(t, dir, "feature/my-thing") {
		t.Error("expected feature/my-thing (local prefix) to be created")
	}
	if v := testutil.GitConfigValue(t, dir, "gitflow.branch.feature.prefix"); v != "feature/" {
		t.Errorf("expected local feature.prefix to stay feature/, got %q", v)
	}
}

// TestFirstRunAutoInitOnlyStillUnconfigured covers scenario 11: a repo whose only
// git-flow key is gitflow.shared.autoInit still reads as unconfigured and activates.
// Steps:
// 1. Builds a fresh-clone fixture and sets only gitflow.shared.autoInit=true locally
// 2. Runs 'feature start my-thing'
// 3. Verifies activation occurred (config copied) and the branch was created
func TestFirstRunAutoInitOnlyStillUnconfigured(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupFreshCloneWithShared(t, testutil.AuthorSharedConfig(t))
	defer testutil.CleanupTestRepo(t, dir)
	if _, err := testutil.RunGit(t, dir, "config", "--local", "gitflow.shared.autoInit", "true"); err != nil {
		t.Fatalf("failed to set autoInit: %v", err)
	}

	out, err := testutil.RunGitFlow(t, dir, "feature", "start", "my-thing")
	if err != nil {
		t.Fatalf("feature start failed: %v\n%s", err, out)
	}
	if v := testutil.GitConfigValue(t, dir, "gitflow.branch.feature.type"); v != "topic" {
		t.Errorf("expected activation to copy feature.type, got %q", v)
	}
	if !testutil.BranchExists(t, dir, "feature/my-thing") {
		t.Error("expected feature/my-thing to be created")
	}
}

// TestNoSharedFileBehavesAsToday covers scenario 32: a repo with no .gitflow that
// was init'd --local behaves exactly as before, with no first-run side effects.
// Steps:
// 1. Runs 'init --local --defaults' (no .gitflow)
// 2. Runs 'feature start x'
// 3. Verifies no prompt/hint/notice about .gitflow and the branch is created
func TestNoSharedFileBehavesAsToday(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	if out, err := testutil.RunGitFlow(t, dir, "init", "--local", "--defaults"); err != nil {
		t.Fatalf("init --local failed: %v\n%s", err, out)
	}

	out, err := testutil.RunGitFlow(t, dir, "feature", "start", "x")
	if err != nil {
		t.Fatalf("feature start failed: %v\n%s", err, out)
	}
	if strings.Contains(out, ".gitflow") {
		t.Errorf("expected no .gitflow text in output, got: %s", out)
	}
	if !testutil.BranchExists(t, dir, "feature/x") {
		t.Error("expected feature/x to be created")
	}
}

// TestBareRepoNoSharedDetection covers scenario 33: in a bare repository the
// first-run hook is a no-op and version still works.
// Steps:
// 1. Creates a bare repository
// 2. Runs 'git flow version' in it
// 3. Verifies exit 0, output exactly '<version> (git-flow-next)', and no .gitflow text or panic
func TestBareRepoNoSharedDetection(t *testing.T) {
	t.Parallel()
	bareDir, err := os.MkdirTemp("", "git-flow-test-bare-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer testutil.CleanupTestRepo(t, bareDir)
	if _, err := testutil.RunGit(t, bareDir, "init", "--bare", "--initial-branch=main"); err != nil {
		t.Fatalf("failed to init bare repo: %v", err)
	}

	out, err := testutil.RunGitFlow(t, bareDir, "version")
	if err != nil {
		t.Fatalf("version in bare repo failed: %v\n%s", err, out)
	}
	assertVersionOutput(t, out)
	if strings.Contains(out, ".gitflow") {
		t.Errorf("expected no .gitflow text in bare-repo version output, got: %s", out)
	}
}

// TestSharedFileMissingVersionNotOffered covers scenario 34: a .gitflow missing
// gitflow.version is not a valid shared config and is not activated.
// Steps:
// 1. Creates a repo with a .gitflow that has branch keys but no gitflow.version
// 2. Runs 'feature start x' non-interactively with autoInit unset
// 3. Verifies no activation, no branch, exit 1, and a hint that version is missing
func TestSharedFileMissingVersionNotOffered(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	if _, err := testutil.RunGit(t, dir, "branch", "develop"); err != nil {
		t.Fatalf("failed to create develop: %v", err)
	}
	testutil.SharedConfigSet(t, dir, "gitflow.branch.feature.type", "topic")
	testutil.SharedConfigSet(t, dir, "gitflow.branch.feature.prefix", "feature/")

	out, err := testutil.RunGitFlow(t, dir, "feature", "start", "x")
	if err == nil {
		t.Fatalf("expected failure for version-less .gitflow, got success\n%s", out)
	}
	assertExitCode(t, err, errors.ExitCodeNotInitialized)
	if testutil.GitConfigHasPrefix(t, dir, "gitflow.") {
		t.Error("expected no gitflow.* keys copied for a version-less .gitflow")
	}
	if testutil.BranchExists(t, dir, "feature/x") {
		t.Error("expected no branch created for a version-less .gitflow")
	}
	if !strings.Contains(out, "gitflow.version") {
		t.Errorf("expected a hint mentioning gitflow.version, got: %s", out)
	}
}

// TestSharedFileMalformedClearErrorFirstRun covers scenario 35: a malformed
// .gitflow fails first-run activation with a clear error naming the file.
// Steps:
// 1. Writes an unparsable .gitflow and sets gitflow.shared.autoInit=true locally
// 2. Runs 'feature start x' (drives the first-run activation path)
// 3. Verifies non-zero exit, an error naming .gitflow, no branch, and no partial copy
func TestSharedFileMalformedClearErrorFirstRun(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	if err := os.WriteFile(testutil.SharedConfigPath(dir), []byte("[gitflow\nbroken = = =\n"), 0644); err != nil {
		t.Fatalf("failed to write malformed .gitflow: %v", err)
	}
	if _, err := testutil.RunGit(t, dir, "config", "--local", "gitflow.shared.autoInit", "true"); err != nil {
		t.Fatalf("failed to set autoInit: %v", err)
	}

	out, err := testutil.RunGitFlow(t, dir, "feature", "start", "x")
	if err == nil {
		t.Fatalf("expected failure for malformed .gitflow, got success\n%s", out)
	}
	if !strings.Contains(out, ".gitflow") {
		t.Errorf("expected error naming .gitflow, got: %s", out)
	}
	if testutil.BranchExists(t, dir, "feature/x") {
		t.Error("expected no branch created on malformed-file failure")
	}
	if testutil.GitConfigHasPrefix(t, dir, "gitflow.branch.") {
		t.Error("expected no partial gitflow.branch.* keys copied on malformed-file failure")
	}
}

// TestFirstRunSkipsInitCommand covers scenario 36: the first-run hook does not
// fire for 'git flow init' itself.
// Steps:
// 1. Creates a repo with a present valid .gitflow, autoInit unset, non-interactive
// 2. Runs 'git flow init --defaults' (plain, no --shared)
// 3. Verifies init writes local config normally and .gitflow is untouched
func TestFirstRunSkipsInitCommand(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	if _, err := testutil.RunGit(t, dir, "branch", "develop"); err != nil {
		t.Fatalf("failed to create develop: %v", err)
	}
	testutil.SharedConfigSet(t, dir, "gitflow.version", "1.0")
	testutil.SharedConfigSet(t, dir, "gitflow.branch.feature.type", "topic")
	testutil.SharedConfigSet(t, dir, "gitflow.branch.feature.prefix", "shared/")

	out, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("init --defaults failed: %v\n%s", err, out)
	}
	// Local config came from init --defaults, not from the shared file.
	if v := testutil.GitConfigValue(t, dir, "gitflow.version"); v == "" {
		t.Error("expected init --defaults to write local gitflow.version")
	}
	if v := testutil.GitConfigValue(t, dir, "gitflow.branch.feature.prefix"); v != "feature/" {
		t.Errorf("expected local feature.prefix from init defaults (feature/), got %q", v)
	}
	// The shared file was not rewritten by the plain init.
	if v := testutil.SharedConfigValue(t, dir, "gitflow.branch.feature.prefix"); v != "shared/" {
		t.Errorf("expected .gitflow feature.prefix untouched (shared/), got %q", v)
	}
}

// assertExitCode fails the test unless err is a testutil.ExitError carrying the
// expected exit code.
func assertExitCode(t *testing.T, err error, want errors.ExitCode) {
	t.Helper()
	ee, ok := err.(*testutil.ExitError)
	if !ok {
		t.Fatalf("expected *testutil.ExitError, got %T (%v)", err, err)
	}
	if ee.ExitCode != int(want) {
		t.Errorf("expected exit code %d, got %d", int(want), ee.ExitCode)
	}
}
