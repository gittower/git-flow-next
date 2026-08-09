package cmd_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gittower/git-flow-next/test/testutil"
)

// advanceRemoteMain clones the given bare remote into a throwaway working copy,
// adds a commit on main, pushes it, and returns the new commit SHA. Used to make
// the primary repo's origin/main tracking ref provably stale before a fetch.
func advanceRemoteMain(t *testing.T, remoteDir string) string {
	t.Helper()
	parent, err := os.MkdirTemp("", "git-flow-test-advance-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(parent) })
	if _, err := testutil.RunGit(t, parent, "clone", remoteDir, "work"); err != nil {
		t.Fatalf("failed to clone remote: %v", err)
	}
	work := filepath.Join(parent, "work")
	testutil.ConfigureGitIdentity(t, work)
	if err := testutil.WriteFile(t, work, "advance.txt", "content"); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	if _, err := testutil.RunGit(t, work, "add", "advance.txt"); err != nil {
		t.Fatalf("failed to add file: %v", err)
	}
	if _, err := testutil.RunGit(t, work, "commit", "-m", "advance main"); err != nil {
		t.Fatalf("failed to commit: %v", err)
	}
	sha, err := testutil.RunGit(t, work, "rev-parse", "HEAD")
	if err != nil {
		t.Fatalf("failed to resolve HEAD: %v", err)
	}
	if _, err := testutil.RunGit(t, work, "push", "origin", "main"); err != nil {
		t.Fatalf("failed to push main: %v", err)
	}
	return strings.TrimSpace(sha)
}

// TestSharedBranchTypeRunnableAfterActivation covers scenario 12: a
// .gitflow-defined branch type is registered and runnable after activation.
// Steps:
// 1. Builds a fresh-clone fixture with autoInit=true
// 2. Runs 'feature start x'
// 3. Verifies feature/x was created and local feature.prefix=feature/
func TestSharedBranchTypeRunnableAfterActivation(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupFreshCloneWithShared(t, testutil.AuthorSharedConfig(t))
	defer testutil.CleanupTestRepo(t, dir)
	if _, err := testutil.RunGit(t, dir, "config", "--local", "gitflow.shared.autoInit", "true"); err != nil {
		t.Fatalf("failed to set autoInit: %v", err)
	}

	if out, err := testutil.RunGitFlow(t, dir, "feature", "start", "x"); err != nil {
		t.Fatalf("feature start failed: %v\n%s", err, out)
	}
	if !testutil.BranchExists(t, dir, "feature/x") {
		t.Error("expected feature/x to be created")
	}
	if v := testutil.GitConfigValue(t, dir, "gitflow.branch.feature.prefix"); v != "feature/" {
		t.Errorf("expected local feature.prefix=feature/, got %q", v)
	}
}

// TestSharedStartFetchDirectReadSite covers scenario 13: gitflow.feature.start.fetch
// declared only in .gitflow triggers a fetch (direct read site in cmd/start.go).
// Steps:
// 1. Builds a repo+remote, authors a .gitflow with start.fetch=true, clears local config, autoInit=true
// 2. Advances origin/main from a second clone to a known SHA the primary has not fetched
// 3. Runs 'feature start x' and verifies origin/main advanced to that SHA and a fetch notice printed
func TestSharedStartFetchDirectReadSite(t *testing.T) {
	t.Parallel()
	dir, remoteDir := testutil.SetupTestRepoWithRemote(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer testutil.CleanupTestRepo(t, remoteDir)

	if err := os.WriteFile(testutil.SharedConfigPath(dir), testutil.AuthorSharedConfig(t), 0644); err != nil {
		t.Fatalf("failed to write .gitflow: %v", err)
	}
	testutil.SharedConfigSet(t, dir, "gitflow.feature.start.fetch", "true")
	testutil.ClearLocalGitflowConfig(t, dir)
	if _, err := testutil.RunGit(t, dir, "config", "--local", "gitflow.shared.autoInit", "true"); err != nil {
		t.Fatalf("failed to set autoInit: %v", err)
	}

	wantSHA := advanceRemoteMain(t, remoteDir)
	before, _ := testutil.RunGit(t, dir, "rev-parse", "refs/remotes/origin/main")
	if strings.TrimSpace(before) == wantSHA {
		t.Fatalf("precondition failed: origin/main already at wantSHA before fetch")
	}

	out, err := testutil.RunGitFlow(t, dir, "feature", "start", "x")
	if err != nil {
		t.Fatalf("feature start failed: %v\n%s", err, out)
	}
	after, err := testutil.RunGit(t, dir, "rev-parse", "refs/remotes/origin/main")
	if err != nil {
		t.Fatalf("failed to resolve origin/main: %v", err)
	}
	if strings.TrimSpace(after) != wantSHA {
		t.Errorf("expected origin/main to advance to %s after fetch, got %s", wantSHA, strings.TrimSpace(after))
	}
	if !strings.Contains(out, "Fetching") {
		t.Errorf("expected a fetch notice in output, got: %s", out)
	}
}

// TestSharedDeleteForceDirectReadSite covers scenario 14a:
// gitflow.feature.delete.force declared only in .gitflow is honored (direct read
// in cmd/delete.go), so an unmerged feature branch is force-deleted even without
// --force on the CLI. (One behavior: force honored from shared config.)
// Steps:
// 1. Builds a fresh-clone fixture whose .gitflow sets delete.force=true (no deleteRemote), autoInit=true
// 2. Starts feature/x and commits a unique (provably unmerged) change on it
// 3. Runs 'feature delete x' (no --force) and verifies the local branch is gone
func TestSharedDeleteForceDirectReadSite(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupFreshCloneWithShared(t, testutil.AuthorSharedConfig(t))
	defer testutil.CleanupTestRepo(t, dir)
	testutil.SharedConfigSet(t, dir, "gitflow.feature.delete.force", "true")
	if _, err := testutil.RunGit(t, dir, "config", "--local", "gitflow.shared.autoInit", "true"); err != nil {
		t.Fatalf("failed to set autoInit: %v", err)
	}

	if out, err := testutil.RunGitFlow(t, dir, "feature", "start", "x"); err != nil {
		t.Fatalf("feature start failed: %v\n%s", err, out)
	}
	if err := testutil.WriteFile(t, dir, "unmerged.txt", "unique"); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	if _, err := testutil.RunGit(t, dir, "add", "unmerged.txt"); err != nil {
		t.Fatalf("failed to add file: %v", err)
	}
	if _, err := testutil.RunGit(t, dir, "commit", "-m", "unique change"); err != nil {
		t.Fatalf("failed to commit: %v", err)
	}

	if out, err := testutil.RunGitFlow(t, dir, "feature", "delete", "x"); err != nil {
		t.Fatalf("feature delete failed: %v\n%s", err, out)
	}
	if testutil.BranchExists(t, dir, "feature/x") {
		t.Error("expected local feature/x to be deleted (force honored from shared config)")
	}
}

// TestSharedDeleteRemoteDirectReadSite covers scenario 14b:
// gitflow.branch.feature.deleteRemote declared only in .gitflow is honored
// (direct read in cmd/delete.go). delete.force is present only so the unmerged
// branch delete proceeds; the asserted behavior is REMOTE deletion.
// Steps:
// 1. Builds a repo+remote, authors a .gitflow with deleteRemote=true and delete.force=true, autoInit=true
// 2. Starts feature/x, commits a unique change, and publishes it so refs/heads/feature/x exists on origin
// 3. Runs 'feature delete x' (no flags) and verifies the remote branch is gone
func TestSharedDeleteRemoteDirectReadSite(t *testing.T) {
	t.Parallel()
	dir, remoteDir := testutil.SetupTestRepoWithRemote(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer testutil.CleanupTestRepo(t, remoteDir)

	if err := os.WriteFile(testutil.SharedConfigPath(dir), testutil.AuthorSharedConfig(t), 0644); err != nil {
		t.Fatalf("failed to write .gitflow: %v", err)
	}
	testutil.SharedConfigSet(t, dir, "gitflow.branch.feature.deleteRemote", "true")
	testutil.SharedConfigSet(t, dir, "gitflow.feature.delete.force", "true")
	testutil.ClearLocalGitflowConfig(t, dir)
	if _, err := testutil.RunGit(t, dir, "config", "--local", "gitflow.shared.autoInit", "true"); err != nil {
		t.Fatalf("failed to set autoInit: %v", err)
	}

	if out, err := testutil.RunGitFlow(t, dir, "feature", "start", "x"); err != nil {
		t.Fatalf("feature start failed: %v\n%s", err, out)
	}
	if err := testutil.WriteFile(t, dir, "unmerged.txt", "unique"); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	if _, err := testutil.RunGit(t, dir, "add", "unmerged.txt"); err != nil {
		t.Fatalf("failed to add file: %v", err)
	}
	if _, err := testutil.RunGit(t, dir, "commit", "-m", "unique change"); err != nil {
		t.Fatalf("failed to commit: %v", err)
	}
	if out, err := testutil.RunGitFlow(t, dir, "feature", "publish", "x"); err != nil {
		t.Fatalf("feature publish failed: %v\n%s", err, out)
	}

	if out, err := testutil.RunGitFlow(t, dir, "feature", "delete", "x"); err != nil {
		t.Fatalf("feature delete failed: %v\n%s", err, out)
	}
	lsRemote, err := testutil.RunGit(t, dir, "ls-remote", "--heads", "origin", "refs/heads/feature/x")
	if err != nil {
		t.Fatalf("ls-remote failed: %v", err)
	}
	if strings.TrimSpace(lsRemote) != "" {
		t.Errorf("expected remote feature/x to be deleted, got: %s", lsRemote)
	}
}

// TestSharedPublishPushOptionMultiValueDirectReadSite covers scenario 15: a
// multi-value gitflow.feature.publish.push-option in .gitflow survives the copy
// and is transmitted in order (direct read in cmd/publish.go).
// Steps:
// 1. Builds a repo+remote with a push-option-capturing hook, authors .gitflow with two options, autoInit=true
// 2. Starts feature/x (activating) and publishes it
// 3. Verifies both options arrived in order and local get-all preserves order
func TestSharedPublishPushOptionMultiValueDirectReadSite(t *testing.T) {
	t.Parallel()
	dir, remoteDir := testutil.SetupTestRepoWithRemote(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer testutil.CleanupTestRepo(t, remoteDir)

	readOptions := installPushOptionHook(t, remoteDir)

	if err := os.WriteFile(testutil.SharedConfigPath(dir), testutil.AuthorSharedConfig(t), 0644); err != nil {
		t.Fatalf("failed to write .gitflow: %v", err)
	}
	if _, err := testutil.RunGit(t, dir, "config", "--file", testutil.SharedConfigPath(dir), "--add", "gitflow.feature.publish.push-option", "opt-a"); err != nil {
		t.Fatalf("failed to add push-option opt-a: %v", err)
	}
	if _, err := testutil.RunGit(t, dir, "config", "--file", testutil.SharedConfigPath(dir), "--add", "gitflow.feature.publish.push-option", "opt-b"); err != nil {
		t.Fatalf("failed to add push-option opt-b: %v", err)
	}
	testutil.ClearLocalGitflowConfig(t, dir)
	if _, err := testutil.RunGit(t, dir, "config", "--local", "gitflow.shared.autoInit", "true"); err != nil {
		t.Fatalf("failed to set autoInit: %v", err)
	}

	if out, err := testutil.RunGitFlow(t, dir, "feature", "start", "x"); err != nil {
		t.Fatalf("feature start failed: %v\n%s", err, out)
	}
	if out, err := testutil.RunGitFlow(t, dir, "feature", "publish", "x"); err != nil {
		t.Fatalf("feature publish failed: %v\n%s", err, out)
	}

	count, options := readOptions()
	if count != 2 || len(options) != 2 || options[0] != "opt-a" || options[1] != "opt-b" {
		t.Errorf("expected push options [opt-a opt-b] in order, got count=%d options=%v", count, options)
	}
	local := testutil.GitConfigAll(t, dir, "gitflow.feature.publish.push-option")
	if len(local) != 2 || local[0] != "opt-a" || local[1] != "opt-b" {
		t.Errorf("expected local push-option [opt-a opt-b] in order, got %v", local)
	}
}

// TestSharedNonGitflowKeysNotCopied covers scenario 16: non-gitflow keys in
// .gitflow are never copied into local config.
// Steps:
// 1. Builds a fresh-clone fixture whose .gitflow also declares core.sshCommand and alias.x, autoInit=true
// 2. Runs 'feature start x' (activates)
// 3. Verifies local config has neither core.sshCommand nor alias.x, but gitflow.* is present
func TestSharedNonGitflowKeysNotCopied(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupFreshCloneWithShared(t, testutil.AuthorSharedConfig(t))
	defer testutil.CleanupTestRepo(t, dir)
	testutil.SharedConfigSet(t, dir, "core.sshCommand", "echo pwned")
	testutil.SharedConfigSet(t, dir, "alias.x", "!sh -c evil")
	if _, err := testutil.RunGit(t, dir, "config", "--local", "gitflow.shared.autoInit", "true"); err != nil {
		t.Fatalf("failed to set autoInit: %v", err)
	}

	if out, err := testutil.RunGitFlow(t, dir, "feature", "start", "x"); err != nil {
		t.Fatalf("feature start failed: %v\n%s", err, out)
	}
	if testutil.GitConfigExists(t, dir, "core.sshCommand") {
		t.Error("expected core.sshCommand NOT copied into local config")
	}
	if testutil.GitConfigExists(t, dir, "alias.x") {
		t.Error("expected alias.x NOT copied into local config")
	}
	if v := testutil.GitConfigValue(t, dir, "gitflow.branch.feature.type"); v != "topic" {
		t.Errorf("expected gitflow.* copied, got feature.type=%q", v)
	}
}

// TestSharedHookPathSkippedWhenUntrustedInteractive covers scenario 17a: with an
// untrusted hook path, an interactive accept skips that one key with a warning
// but copies the rest and proceeds.
// Steps:
// 1. Builds a fresh-clone fixture whose .gitflow sets gitflow.path.hooks, trust/autoInit unset
// 2. Runs 'feature start x' interactively answering "y"
// 3. Verifies the hook path was not copied, the rest was, a warning naming trustHooks printed, branch created
func TestSharedHookPathSkippedWhenUntrustedInteractive(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupFreshCloneWithShared(t, testutil.AuthorSharedConfig(t))
	defer testutil.CleanupTestRepo(t, dir)
	testutil.SharedConfigSet(t, dir, "gitflow.path.hooks", "/some/hooks")

	out, err := testutil.RunGitFlowInteractive(t, dir, "y\n", "feature", "start", "x")
	if err != nil {
		t.Fatalf("feature start after accept failed: %v\n%s", err, out)
	}
	if testutil.GitConfigExists(t, dir, "gitflow.path.hooks") {
		t.Error("expected gitflow.path.hooks NOT copied when untrusted")
	}
	if v := testutil.GitConfigValue(t, dir, "gitflow.branch.feature.prefix"); v != "feature/" {
		t.Errorf("expected the rest of gitflow.* copied, got feature.prefix=%q", v)
	}
	if !strings.Contains(out, "gitflow.shared.trustHooks") {
		t.Errorf("expected a warning naming gitflow.shared.trustHooks, got: %s", out)
	}
	if !testutil.BranchExists(t, dir, "feature/x") {
		t.Error("expected feature/x to be created despite the skipped hook path")
	}
}

// TestSharedHookPathCopiedWhenTrusted covers scenario 17b: with
// gitflow.shared.trustHooks=true the hook path is copied.
// Steps:
// 1. Builds a fresh-clone fixture whose .gitflow sets gitflow.path.hooks, trustHooks=true, autoInit=true
// 2. Runs 'feature start x' (activates)
// 3. Verifies local config contains gitflow.path.hooks=/some/hooks
func TestSharedHookPathCopiedWhenTrusted(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupFreshCloneWithShared(t, testutil.AuthorSharedConfig(t))
	defer testutil.CleanupTestRepo(t, dir)
	testutil.SharedConfigSet(t, dir, "gitflow.path.hooks", "/some/hooks")
	if _, err := testutil.RunGit(t, dir, "config", "--local", "gitflow.shared.trustHooks", "true"); err != nil {
		t.Fatalf("failed to set trustHooks: %v", err)
	}
	if _, err := testutil.RunGit(t, dir, "config", "--local", "gitflow.shared.autoInit", "true"); err != nil {
		t.Fatalf("failed to set autoInit: %v", err)
	}

	if out, err := testutil.RunGitFlow(t, dir, "feature", "start", "x"); err != nil {
		t.Fatalf("feature start failed: %v\n%s", err, out)
	}
	if v := testutil.GitConfigValue(t, dir, "gitflow.path.hooks"); v != "/some/hooks" {
		t.Errorf("expected local gitflow.path.hooks=/some/hooks when trusted, got %q", v)
	}
}

// TestSharedAutoInitRefusesUntrustedHookPath covers scenario 17c: non-interactive
// auto-init with an untrusted hook path refuses wholesale.
// Steps:
// 1. Builds a fresh-clone fixture whose .gitflow sets gitflow.path.hooks, autoInit=true, trust unset
// 2. Runs 'feature start x' non-interactively
// 3. Verifies failure naming trustHooks, no gitflow.branch.* copied, and the autoInit control key untouched
func TestSharedAutoInitRefusesUntrustedHookPath(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupFreshCloneWithShared(t, testutil.AuthorSharedConfig(t))
	defer testutil.CleanupTestRepo(t, dir)
	testutil.SharedConfigSet(t, dir, "gitflow.path.hooks", "/some/hooks")
	if _, err := testutil.RunGit(t, dir, "config", "--local", "gitflow.shared.autoInit", "true"); err != nil {
		t.Fatalf("failed to set autoInit: %v", err)
	}

	out, err := testutil.RunGitFlow(t, dir, "feature", "start", "x")
	if err == nil {
		t.Fatalf("expected auto-init to refuse an untrusted hook path, got success\n%s", out)
	}
	if !strings.Contains(out, "gitflow.shared.trustHooks") {
		t.Errorf("expected the error to name gitflow.shared.trustHooks, got: %s", out)
	}
	if testutil.GitConfigHasPrefix(t, dir, "gitflow.branch.") {
		t.Error("expected no gitflow.branch.* copied when auto-init refuses")
	}
	if v := testutil.GitConfigValue(t, dir, "gitflow.shared.autoInit"); v != "true" {
		t.Errorf("expected the pre-existing autoInit control key preserved, got %q", v)
	}
}

// TestSharedCustomDottedTypeAvailableBeforeActivation covers scenario 18: a custom
// dotted-name topic type from .gitflow is usable on a fresh clone before activation.
// Steps:
// 1. Builds a fresh-clone fixture whose .gitflow defines a custom topic type qa.Release, autoInit=true
// 2. Runs 'git flow qa.Release start x' as the first command
// 3. Verifies the command is recognized and the dotted-name branch is created
func TestSharedCustomDottedTypeAvailableBeforeActivation(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupFreshCloneWithShared(t, testutil.AuthorSharedConfig(t))
	defer testutil.CleanupTestRepo(t, dir)
	testutil.SharedConfigSet(t, dir, "gitflow.branch.qa.Release.type", "topic")
	testutil.SharedConfigSet(t, dir, "gitflow.branch.qa.Release.parent", "develop")
	testutil.SharedConfigSet(t, dir, "gitflow.branch.qa.Release.prefix", "qa.Release/")
	if _, err := testutil.RunGit(t, dir, "config", "--local", "gitflow.shared.autoInit", "true"); err != nil {
		t.Fatalf("failed to set autoInit: %v", err)
	}

	out, err := testutil.RunGitFlow(t, dir, "qa.Release", "start", "x")
	if err != nil {
		t.Fatalf("qa.Release start failed: %v\n%s", err, out)
	}
	if strings.Contains(out, "unknown command") {
		t.Errorf("expected qa.Release command to be recognized, got: %s", out)
	}
	if !testutil.BranchExists(t, dir, "qa.Release/x") {
		t.Error("expected qa.Release/x branch to be created with its dotted name preserved")
	}
}

// TestSharedHookPathCopiedWhenTrustHooksYes covers the git-config truthy "yes"
// spelling of gitflow.shared.trustHooks: the hook path is copied on activation.
// Steps:
// 1. Builds a fresh-clone fixture whose .gitflow sets gitflow.path.hooks
// 2. Sets local gitflow.shared.trustHooks=yes and gitflow.shared.autoInit=true
// 3. Runs 'feature start x' (activates)
// 4. Verifies local config contains gitflow.path.hooks=/some/hooks
func TestSharedHookPathCopiedWhenTrustHooksYes(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupFreshCloneWithShared(t, testutil.AuthorSharedConfig(t))
	defer testutil.CleanupTestRepo(t, dir)
	testutil.SharedConfigSet(t, dir, "gitflow.path.hooks", "/some/hooks")
	if _, err := testutil.RunGit(t, dir, "config", "--local", "gitflow.shared.trustHooks", "yes"); err != nil {
		t.Fatalf("failed to set trustHooks: %v", err)
	}
	if _, err := testutil.RunGit(t, dir, "config", "--local", "gitflow.shared.autoInit", "true"); err != nil {
		t.Fatalf("failed to set autoInit: %v", err)
	}

	if out, err := testutil.RunGitFlow(t, dir, "feature", "start", "x"); err != nil {
		t.Fatalf("feature start failed: %v\n%s", err, out)
	}
	if v := testutil.GitConfigValue(t, dir, "gitflow.path.hooks"); v != "/some/hooks" {
		t.Errorf("expected local gitflow.path.hooks=/some/hooks when trusted, got %q", v)
	}
}
