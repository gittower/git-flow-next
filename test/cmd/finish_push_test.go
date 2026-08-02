package cmd_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gittower/git-flow-next/test/testutil"
)

// Push output format asserted throughout these tests (from the spec examples):
//
//	Header line: "Pushing to remote 'origin'..."
//	Per branch:  "  <branch> -> origin/<branch>"
//	Tag:         "  <tag> (tag) -> origin"
const (
	pushHeader     = "Pushing to remote 'origin'..."
	pushMainLine   = "  main -> origin/main"
	pushDevelLine  = "  develop -> origin/develop"
	pushTag100Line = "  1.0.0 (tag) -> origin"
)

// setupSingleTrackWithRemote creates a GitHub-flow (single-track) repository with
// an origin remote whose main branch is tracked. This maps to the spec's
// "single-track" layout: main only, feature -> main, no develop, no tag.
// Returns the local repo dir and the bare remote dir.
func setupSingleTrackWithRemote(t *testing.T) (string, string) {
	t.Helper()
	dir := testutil.SetupTestRepo(t)

	if _, err := testutil.RunGitFlow(t, dir, "init", "--preset=github"); err != nil {
		testutil.CleanupTestRepo(t, dir)
		t.Fatalf("Failed to initialize git-flow github preset: %v", err)
	}

	remoteDir, err := testutil.AddRemote(t, dir, "origin", false)
	if err != nil {
		testutil.CleanupTestRepo(t, dir)
		t.Fatalf("Failed to add remote: %v", err)
	}

	if _, err := testutil.RunGit(t, dir, "push", "-u", "origin", "main"); err != nil {
		testutil.CleanupTestRepo(t, dir)
		testutil.CleanupTestRepo(t, remoteDir)
		t.Fatalf("Failed to push main: %v", err)
	}

	return dir, remoteDir
}

// setupSingleTrackNoRemote creates a GitHub-flow (single-track) repository with
// no remote configured.
func setupSingleTrackNoRemote(t *testing.T) string {
	t.Helper()
	dir := testutil.SetupTestRepo(t)

	if _, err := testutil.RunGitFlow(t, dir, "init", "--preset=github"); err != nil {
		testutil.CleanupTestRepo(t, dir)
		t.Fatalf("Failed to initialize git-flow github preset: %v", err)
	}

	return dir
}

// createTopicCommit starts a topic branch of the given type and commits one file
// to it, leaving that branch checked out.
func createTopicCommit(t *testing.T, dir, branchType, name, file, content string) {
	t.Helper()
	if _, err := testutil.RunGitFlow(t, dir, branchType, "start", name); err != nil {
		t.Fatalf("Failed to start %s branch %s: %v", branchType, name, err)
	}
	if err := testutil.WriteFile(t, dir, file, content); err != nil {
		t.Fatalf("Failed to write %s: %v", file, err)
	}
	if _, err := testutil.RunGit(t, dir, "add", file); err != nil {
		t.Fatalf("Failed to add %s: %v", file, err)
	}
	if _, err := testutil.RunGit(t, dir, "commit", "-m", "Add "+file); err != nil {
		t.Fatalf("Failed to commit %s: %v", file, err)
	}
}

// TestFinishPushDefaultDoesNotPush tests that finishing without any push flag or
// config does not push to the remote.
// Steps:
// 1. Sets up a single-track (github preset) repo with origin tracking main
// 2. Creates feature branch 'login' with one commit
// 3. Runs 'git flow feature finish login --no-fetch' (no push flag, no config)
// 4. Verifies exit 0 and that main is ahead of origin (not pushed)
// 5. Verifies output contains no "Pushing to remote" header
func TestFinishPushDefaultDoesNotPush(t *testing.T) {
	dir, remoteDir := setupSingleTrackWithRemote(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer testutil.CleanupTestRepo(t, remoteDir)

	createTopicCommit(t, dir, "feature", "login", "login.txt", "login content")

	output, err := testutil.RunGitFlow(t, dir, "feature", "finish", "login", "--no-fetch")
	if err != nil {
		t.Fatalf("Failed to finish feature branch: %v\nOutput: %s", err, output)
	}

	if got := testutil.CommitsAhead(t, dir, "origin/main", "main"); got == 0 {
		t.Errorf("Expected main to be ahead of origin (not pushed), got %d commits ahead", got)
	}
	if strings.Contains(output, pushHeader) {
		t.Errorf("Expected no push to occur by default. Output: %s", output)
	}
}

// TestFinishPushSingleTrackFeature tests that --push pushes the target branch on a
// single-track feature finish (no children, no tag).
// Steps:
// 1. Sets up a single-track repo with origin tracking main
// 2. Creates feature branch 'login' with one commit
// 3. Runs 'git flow feature finish login --push --no-fetch'
// 4. Verifies exit 0 and main up to date with origin (pushed)
// 5. Verifies output shows the push header and the main push line
// 6. Verifies the finished topic branch is NOT pushed to origin
func TestFinishPushSingleTrackFeature(t *testing.T) {
	dir, remoteDir := setupSingleTrackWithRemote(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer testutil.CleanupTestRepo(t, remoteDir)

	createTopicCommit(t, dir, "feature", "login", "login.txt", "login content")

	output, err := testutil.RunGitFlow(t, dir, "feature", "finish", "login", "--push", "--no-fetch")
	if err != nil {
		t.Fatalf("Failed to finish feature branch with --push: %v\nOutput: %s", err, output)
	}

	if got := testutil.CommitsAhead(t, dir, "origin/main", "main"); got != 0 {
		t.Errorf("Expected main up to date with origin (pushed), got %d commits ahead", got)
	}
	if !strings.Contains(output, pushHeader) {
		t.Errorf("Expected push header in output. Output: %s", output)
	}
	if !strings.Contains(output, pushMainLine) {
		t.Errorf("Expected main push line in output. Output: %s", output)
	}

	// The topic branch itself is never pushed by these flags.
	heads, err := testutil.RunGit(t, dir, "ls-remote", "--heads", "origin")
	if err != nil {
		t.Fatalf("Failed to list remote heads: %v", err)
	}
	if strings.Contains(heads, "refs/heads/feature/login") {
		t.Errorf("Expected feature/login NOT to be pushed to origin. Heads: %s", heads)
	}
}

// TestFinishPushClassicReleaseBranchesAndTag tests that --push on a classic release
// pushes the target branch, the auto-updated child branch, and the created tag.
// Steps:
// 1. Sets up a classic repo (main + develop tracked) with origin
// 2. Creates release branch '1.0.0' with one commit
// 3. Runs 'git flow release finish 1.0.0 --push --no-fetch'
// 4. Verifies exit 0 and that main and develop are up to date with origin
// 5. Verifies the release tag 1.0.0 exists on origin
// 6. Verifies output shows main, develop, and tag push lines
func TestFinishPushClassicReleaseBranchesAndTag(t *testing.T) {
	dir, remoteDir := testutil.SetupTestRepoWithRemote(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer testutil.CleanupTestRepo(t, remoteDir)

	createTopicCommit(t, dir, "release", "1.0.0", "release.txt", "release content")

	output, err := testutil.RunGitFlow(t, dir, "release", "finish", "1.0.0", "--push", "--no-fetch")
	if err != nil {
		t.Fatalf("Failed to finish release with --push: %v\nOutput: %s", err, output)
	}

	if got := testutil.CommitsAhead(t, dir, "origin/main", "main"); got != 0 {
		t.Errorf("Expected main up to date with origin, got %d ahead", got)
	}
	if got := testutil.CommitsAhead(t, dir, "origin/develop", "develop"); got != 0 {
		t.Errorf("Expected develop up to date with origin, got %d ahead", got)
	}
	if !testutil.RemoteTagExists(t, dir, "origin", "1.0.0") {
		t.Error("Expected tag 1.0.0 to exist on origin")
	}
	if !strings.Contains(output, pushMainLine) {
		t.Errorf("Expected main push line. Output: %s", output)
	}
	if !strings.Contains(output, pushDevelLine) {
		t.Errorf("Expected develop push line. Output: %s", output)
	}
	if !strings.Contains(output, pushTag100Line) {
		t.Errorf("Expected tag push line. Output: %s", output)
	}
}

// TestFinishPushConfigEnablesPush tests that gitflow.feature.finish.push=true
// triggers a push without any CLI flag.
// Steps:
// 1. Sets up a single-track repo with origin tracking main
// 2. Sets gitflow.feature.finish.push=true
// 3. Creates feature branch 'login' with one commit
// 4. Runs 'git flow feature finish login --no-fetch' (no push flag)
// 5. Verifies exit 0 and main up to date with origin (config alone triggers push)
// 6. Verifies output contains the main push line
func TestFinishPushConfigEnablesPush(t *testing.T) {
	dir, remoteDir := setupSingleTrackWithRemote(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer testutil.CleanupTestRepo(t, remoteDir)

	if _, err := testutil.RunGit(t, dir, "config", "gitflow.feature.finish.push", "true"); err != nil {
		t.Fatalf("Failed to configure push: %v", err)
	}

	createTopicCommit(t, dir, "feature", "login", "login.txt", "login content")

	output, err := testutil.RunGitFlow(t, dir, "feature", "finish", "login", "--no-fetch")
	if err != nil {
		t.Fatalf("Failed to finish feature branch: %v\nOutput: %s", err, output)
	}

	if got := testutil.CommitsAhead(t, dir, "origin/main", "main"); got != 0 {
		t.Errorf("Expected main up to date with origin (config push), got %d ahead", got)
	}
	if !strings.Contains(output, pushMainLine) {
		t.Errorf("Expected main push line. Output: %s", output)
	}
}

// TestFinishPushNoPushFlagOverridesConfig tests that --no-push overrides
// gitflow.feature.finish.push=true.
// Steps:
// 1. Sets up a single-track repo with origin tracking main
// 2. Sets gitflow.feature.finish.push=true
// 3. Creates feature branch 'login' with one commit
// 4. Runs 'git flow feature finish login --no-push --no-fetch'
// 5. Verifies exit 0 and main ahead of origin (nothing pushed)
// 6. Verifies output contains no "Pushing to remote" header
func TestFinishPushNoPushFlagOverridesConfig(t *testing.T) {
	dir, remoteDir := setupSingleTrackWithRemote(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer testutil.CleanupTestRepo(t, remoteDir)

	if _, err := testutil.RunGit(t, dir, "config", "gitflow.feature.finish.push", "true"); err != nil {
		t.Fatalf("Failed to configure push: %v", err)
	}

	createTopicCommit(t, dir, "feature", "login", "login.txt", "login content")

	output, err := testutil.RunGitFlow(t, dir, "feature", "finish", "login", "--no-push", "--no-fetch")
	if err != nil {
		t.Fatalf("Failed to finish feature branch: %v\nOutput: %s", err, output)
	}

	if got := testutil.CommitsAhead(t, dir, "origin/main", "main"); got == 0 {
		t.Errorf("Expected main ahead of origin (--no-push overrides config), got %d ahead", got)
	}
	if strings.Contains(output, pushHeader) {
		t.Errorf("Expected no push with --no-push. Output: %s", output)
	}
}

// TestFinishPushBranchesNotTag tests that --push --no-pushtag pushes branches but
// not the tag on a classic release.
// Steps:
// 1. Sets up a classic repo with origin
// 2. Creates release branch '1.0.0' with one commit
// 3. Runs 'git flow release finish 1.0.0 --push --no-pushtag --no-fetch'
// 4. Verifies exit 0 and main and develop up to date with origin
// 5. Verifies tag 1.0.0 is NOT on origin
// 6. Verifies output shows branch lines but no (tag) line
func TestFinishPushBranchesNotTag(t *testing.T) {
	dir, remoteDir := testutil.SetupTestRepoWithRemote(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer testutil.CleanupTestRepo(t, remoteDir)

	createTopicCommit(t, dir, "release", "1.0.0", "release.txt", "release content")

	output, err := testutil.RunGitFlow(t, dir, "release", "finish", "1.0.0", "--push", "--no-pushtag", "--no-fetch")
	if err != nil {
		t.Fatalf("Failed to finish release: %v\nOutput: %s", err, output)
	}

	if got := testutil.CommitsAhead(t, dir, "origin/main", "main"); got != 0 {
		t.Errorf("Expected main up to date with origin, got %d ahead", got)
	}
	if got := testutil.CommitsAhead(t, dir, "origin/develop", "develop"); got != 0 {
		t.Errorf("Expected develop up to date with origin, got %d ahead", got)
	}
	if testutil.RemoteTagExists(t, dir, "origin", "1.0.0") {
		t.Error("Expected tag 1.0.0 NOT to be on origin with --no-pushtag")
	}
	if !strings.Contains(output, pushMainLine) {
		t.Errorf("Expected main push line. Output: %s", output)
	}
	if strings.Contains(output, "(tag)") {
		t.Errorf("Expected no tag push line. Output: %s", output)
	}
}

// TestFinishPushTagOnlyViaFlag tests that --pushtag alone pushes only the tag, not
// branches, on a classic release.
// Steps:
// 1. Sets up a classic repo with origin
// 2. Creates release branch '1.0.0' with one commit
// 3. Runs 'git flow release finish 1.0.0 --pushtag --no-fetch' (no --push)
// 4. Verifies exit 0 and tag 1.0.0 exists on origin
// 5. Verifies main and develop are ahead of origin (branches not pushed)
// 6. Verifies output shows the tag line but no branch push lines
func TestFinishPushTagOnlyViaFlag(t *testing.T) {
	dir, remoteDir := testutil.SetupTestRepoWithRemote(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer testutil.CleanupTestRepo(t, remoteDir)

	createTopicCommit(t, dir, "release", "1.0.0", "release.txt", "release content")

	output, err := testutil.RunGitFlow(t, dir, "release", "finish", "1.0.0", "--pushtag", "--no-fetch")
	if err != nil {
		t.Fatalf("Failed to finish release: %v\nOutput: %s", err, output)
	}

	if !testutil.RemoteTagExists(t, dir, "origin", "1.0.0") {
		t.Error("Expected tag 1.0.0 to exist on origin")
	}
	if got := testutil.CommitsAhead(t, dir, "origin/main", "main"); got == 0 {
		t.Errorf("Expected main ahead of origin (branches not pushed), got %d ahead", got)
	}
	if got := testutil.CommitsAhead(t, dir, "origin/develop", "develop"); got == 0 {
		t.Errorf("Expected develop ahead of origin (branches not pushed), got %d ahead", got)
	}
	if !strings.Contains(output, pushTag100Line) {
		t.Errorf("Expected tag push line. Output: %s", output)
	}
	if strings.Contains(output, pushMainLine) || strings.Contains(output, pushDevelLine) {
		t.Errorf("Expected no branch push lines. Output: %s", output)
	}
}

// TestFinishPushConfigBranchesTagSuppressed tests per-key config where push=true
// but pushtag=false on a classic release.
// Steps:
// 1. Sets up a classic repo with origin
// 2. Sets gitflow.release.finish.push=true and gitflow.release.finish.pushtag=false
// 3. Creates release branch '1.0.0' with one commit
// 4. Runs 'git flow release finish 1.0.0 --no-fetch' (no flags)
// 5. Verifies exit 0 and main and develop up to date with origin
// 6. Verifies tag 1.0.0 is NOT on origin (tag suppressed)
func TestFinishPushConfigBranchesTagSuppressed(t *testing.T) {
	dir, remoteDir := testutil.SetupTestRepoWithRemote(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer testutil.CleanupTestRepo(t, remoteDir)

	if _, err := testutil.RunGit(t, dir, "config", "gitflow.release.finish.push", "true"); err != nil {
		t.Fatalf("Failed to configure push: %v", err)
	}
	if _, err := testutil.RunGit(t, dir, "config", "gitflow.release.finish.pushtag", "false"); err != nil {
		t.Fatalf("Failed to configure pushtag: %v", err)
	}

	createTopicCommit(t, dir, "release", "1.0.0", "release.txt", "release content")

	output, err := testutil.RunGitFlow(t, dir, "release", "finish", "1.0.0", "--no-fetch")
	if err != nil {
		t.Fatalf("Failed to finish release: %v\nOutput: %s", err, output)
	}

	if got := testutil.CommitsAhead(t, dir, "origin/main", "main"); got != 0 {
		t.Errorf("Expected main up to date with origin, got %d ahead", got)
	}
	if got := testutil.CommitsAhead(t, dir, "origin/develop", "develop"); got != 0 {
		t.Errorf("Expected develop up to date with origin, got %d ahead", got)
	}
	if testutil.RemoteTagExists(t, dir, "origin", "1.0.0") {
		t.Error("Expected tag 1.0.0 NOT to be on origin (pushtag=false)")
	}
}

// TestFinishPushConfigTagOnly tests that gitflow.release.finish.pushtag=true alone
// pushes the tag but not branches.
// Steps:
// 1. Sets up a classic repo with origin
// 2. Sets gitflow.release.finish.pushtag=true (no push config, no flags)
// 3. Creates release branch '1.0.0' with one commit
// 4. Runs 'git flow release finish 1.0.0 --no-fetch'
// 5. Verifies exit 0 and tag 1.0.0 exists on origin
// 6. Verifies main and develop are ahead of origin (branches not pushed)
func TestFinishPushConfigTagOnly(t *testing.T) {
	dir, remoteDir := testutil.SetupTestRepoWithRemote(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer testutil.CleanupTestRepo(t, remoteDir)

	if _, err := testutil.RunGit(t, dir, "config", "gitflow.release.finish.pushtag", "true"); err != nil {
		t.Fatalf("Failed to configure pushtag: %v", err)
	}

	createTopicCommit(t, dir, "release", "1.0.0", "release.txt", "release content")

	output, err := testutil.RunGitFlow(t, dir, "release", "finish", "1.0.0", "--no-fetch")
	if err != nil {
		t.Fatalf("Failed to finish release: %v\nOutput: %s", err, output)
	}

	if !testutil.RemoteTagExists(t, dir, "origin", "1.0.0") {
		t.Error("Expected tag 1.0.0 to exist on origin (pushtag config)")
	}
	if got := testutil.CommitsAhead(t, dir, "origin/main", "main"); got == 0 {
		t.Errorf("Expected main ahead of origin (branches not pushed), got %d ahead", got)
	}
	if got := testutil.CommitsAhead(t, dir, "origin/develop", "develop"); got == 0 {
		t.Errorf("Expected develop ahead of origin (branches not pushed), got %d ahead", got)
	}
}

// TestFinishPushTagFlagOverridesConfigFalse tests that --pushtag overrides
// gitflow.release.finish.pushtag=false without implying --push.
// Steps:
// 1. Sets up a classic repo with origin
// 2. Sets gitflow.release.finish.pushtag=false
// 3. Creates release branch '1.0.0' with one commit
// 4. Runs 'git flow release finish 1.0.0 --pushtag --no-fetch'
// 5. Verifies exit 0 and tag 1.0.0 exists on origin (flag overrides config)
// 6. Verifies main and develop are ahead of origin (--pushtag alone does not imply --push)
func TestFinishPushTagFlagOverridesConfigFalse(t *testing.T) {
	dir, remoteDir := testutil.SetupTestRepoWithRemote(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer testutil.CleanupTestRepo(t, remoteDir)

	if _, err := testutil.RunGit(t, dir, "config", "gitflow.release.finish.pushtag", "false"); err != nil {
		t.Fatalf("Failed to configure pushtag: %v", err)
	}

	createTopicCommit(t, dir, "release", "1.0.0", "release.txt", "release content")

	output, err := testutil.RunGitFlow(t, dir, "release", "finish", "1.0.0", "--pushtag", "--no-fetch")
	if err != nil {
		t.Fatalf("Failed to finish release: %v\nOutput: %s", err, output)
	}

	if !testutil.RemoteTagExists(t, dir, "origin", "1.0.0") {
		t.Error("Expected tag 1.0.0 to exist on origin (--pushtag overrides config)")
	}
	if got := testutil.CommitsAhead(t, dir, "origin/main", "main"); got == 0 {
		t.Errorf("Expected main ahead of origin (--pushtag must not imply --push), got %d ahead", got)
	}
	if got := testutil.CommitsAhead(t, dir, "origin/develop", "develop"); got == 0 {
		t.Errorf("Expected develop ahead of origin (--pushtag must not imply --push), got %d ahead", got)
	}
}

// TestFinishPushNoPushSuppressesInheritedTag tests that --no-push suppresses the
// inherited tag push even when gitflow.release.finish.push=true.
// Steps:
// 1. Sets up a classic repo with origin
// 2. Sets gitflow.release.finish.push=true
// 3. Creates release branch '1.0.0' with one commit
// 4. Runs 'git flow release finish 1.0.0 --no-push --no-fetch'
// 5. Verifies exit 0, main and develop ahead of origin, and tag 1.0.0 NOT on origin
func TestFinishPushNoPushSuppressesInheritedTag(t *testing.T) {
	dir, remoteDir := testutil.SetupTestRepoWithRemote(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer testutil.CleanupTestRepo(t, remoteDir)

	if _, err := testutil.RunGit(t, dir, "config", "gitflow.release.finish.push", "true"); err != nil {
		t.Fatalf("Failed to configure push: %v", err)
	}

	createTopicCommit(t, dir, "release", "1.0.0", "release.txt", "release content")

	output, err := testutil.RunGitFlow(t, dir, "release", "finish", "1.0.0", "--no-push", "--no-fetch")
	if err != nil {
		t.Fatalf("Failed to finish release: %v\nOutput: %s", err, output)
	}

	if got := testutil.CommitsAhead(t, dir, "origin/main", "main"); got == 0 {
		t.Errorf("Expected main ahead of origin (--no-push), got %d ahead", got)
	}
	if got := testutil.CommitsAhead(t, dir, "origin/develop", "develop"); got == 0 {
		t.Errorf("Expected develop ahead of origin (--no-push), got %d ahead", got)
	}
	if testutil.RemoteTagExists(t, dir, "origin", "1.0.0") {
		t.Error("Expected tag 1.0.0 NOT to be on origin (--no-push suppresses inherited tag)")
	}
}

// TestFinishPushTagNoopWhenNoTag tests that --push --pushtag on a single-track
// feature (no tag created) pushes the branch and no-ops the tag.
// Steps:
// 1. Sets up a single-track repo with origin tracking main
// 2. Creates feature branch 'login' with one commit
// 3. Runs 'git flow feature finish login --push --pushtag --no-fetch'
// 4. Verifies exit 0 (no error over absent tag) and main up to date with origin
// 5. Verifies output shows the main push line and no (tag) line
func TestFinishPushTagNoopWhenNoTag(t *testing.T) {
	dir, remoteDir := setupSingleTrackWithRemote(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer testutil.CleanupTestRepo(t, remoteDir)

	createTopicCommit(t, dir, "feature", "login", "login.txt", "login content")

	output, err := testutil.RunGitFlow(t, dir, "feature", "finish", "login", "--push", "--pushtag", "--no-fetch")
	if err != nil {
		t.Fatalf("Failed to finish feature branch: %v\nOutput: %s", err, output)
	}

	if got := testutil.CommitsAhead(t, dir, "origin/main", "main"); got != 0 {
		t.Errorf("Expected main up to date with origin, got %d ahead", got)
	}
	if !strings.Contains(output, pushMainLine) {
		t.Errorf("Expected main push line. Output: %s", output)
	}
	if strings.Contains(output, "(tag)") {
		t.Errorf("Expected no tag push line when no tag created. Output: %s", output)
	}
}

// TestFinishPushClassicHotfix tests that --push works on a classic hotfix, pushing
// the target branch, the auto-updated develop, and the hotfix tag.
// Steps:
// 1. Sets up a classic repo with origin
// 2. Creates hotfix branch '1.0.1' with one commit
// 3. Runs 'git flow hotfix finish 1.0.1 --push --no-fetch'
// 4. Verifies exit 0 and main and develop up to date with origin
// 5. Verifies hotfix tag 1.0.1 exists on origin
func TestFinishPushClassicHotfix(t *testing.T) {
	dir, remoteDir := testutil.SetupTestRepoWithRemote(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer testutil.CleanupTestRepo(t, remoteDir)

	createTopicCommit(t, dir, "hotfix", "1.0.1", "hotfix.txt", "hotfix content")

	output, err := testutil.RunGitFlow(t, dir, "hotfix", "finish", "1.0.1", "--push", "--no-fetch")
	if err != nil {
		t.Fatalf("Failed to finish hotfix with --push: %v\nOutput: %s", err, output)
	}

	if got := testutil.CommitsAhead(t, dir, "origin/main", "main"); got != 0 {
		t.Errorf("Expected main up to date with origin, got %d ahead", got)
	}
	if got := testutil.CommitsAhead(t, dir, "origin/develop", "develop"); got != 0 {
		t.Errorf("Expected develop up to date with origin, got %d ahead", got)
	}
	if !testutil.RemoteTagExists(t, dir, "origin", "1.0.1") {
		t.Error("Expected hotfix tag 1.0.1 to exist on origin")
	}
}

// TestFinishPushPerBranchTypeIndependence tests that push resolution keys off the
// branch type: a feature configured to push does, while a hotfix configured not to
// does not.
// Steps:
// 1. Sets up a classic repo with origin
// 2. Sets gitflow.feature.finish.push=true and gitflow.hotfix.finish.push=false
// 3. Creates feature branch 'x' (-> develop) with one commit and finishes it
// 4. Verifies develop up to date with origin (feature target pushed)
// 5. Creates hotfix branch '1.0.1' (-> main) with one commit and finishes it
// 6. Verifies main and develop are ahead of origin (hotfix pushes nothing)
// 7. Verifies hotfix tag 1.0.1 is NOT on origin and no push header in hotfix output
func TestFinishPushPerBranchTypeIndependence(t *testing.T) {
	dir, remoteDir := testutil.SetupTestRepoWithRemote(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer testutil.CleanupTestRepo(t, remoteDir)

	if _, err := testutil.RunGit(t, dir, "config", "gitflow.feature.finish.push", "true"); err != nil {
		t.Fatalf("Failed to configure feature push: %v", err)
	}
	if _, err := testutil.RunGit(t, dir, "config", "gitflow.hotfix.finish.push", "false"); err != nil {
		t.Fatalf("Failed to configure hotfix push: %v", err)
	}

	// Finish the feature (target develop should be pushed).
	createTopicCommit(t, dir, "feature", "x", "feature-x.txt", "feature x content")
	featureOutput, err := testutil.RunGitFlow(t, dir, "feature", "finish", "x", "--no-fetch")
	if err != nil {
		t.Fatalf("Failed to finish feature: %v\nOutput: %s", err, featureOutput)
	}
	if got := testutil.CommitsAhead(t, dir, "origin/develop", "develop"); got != 0 {
		t.Errorf("Expected develop up to date with origin after feature finish, got %d ahead", got)
	}

	// Finish the hotfix (nothing should be pushed).
	createTopicCommit(t, dir, "hotfix", "1.0.1", "hotfix.txt", "hotfix content")
	hotfixOutput, err := testutil.RunGitFlow(t, dir, "hotfix", "finish", "1.0.1", "--no-fetch")
	if err != nil {
		t.Fatalf("Failed to finish hotfix: %v\nOutput: %s", err, hotfixOutput)
	}
	if got := testutil.CommitsAhead(t, dir, "origin/main", "main"); got == 0 {
		t.Errorf("Expected main ahead of origin after hotfix finish (no push), got %d ahead", got)
	}
	if got := testutil.CommitsAhead(t, dir, "origin/develop", "develop"); got == 0 {
		t.Errorf("Expected develop ahead of origin after hotfix finish (no push), got %d ahead", got)
	}
	if testutil.RemoteTagExists(t, dir, "origin", "1.0.1") {
		t.Error("Expected hotfix tag 1.0.1 NOT to be on origin (hotfix push disabled)")
	}
	if strings.Contains(hotfixOutput, pushHeader) {
		t.Errorf("Expected no push header in hotfix output. Output: %s", hotfixOutput)
	}
}

// TestFinishPushNoRemoteSkips tests that --push with no remote configured skips the
// push with a note rather than erroring.
// Steps:
// 1. Sets up a single-track repo with NO remote
// 2. Creates feature branch 'login' with one commit
// 3. Runs 'git flow feature finish login --push'
// 4. Verifies exit 0 and the finish completes (feature branch deleted, main has merge)
// 5. Verifies output notes the push was skipped and contains no hard error
func TestFinishPushNoRemoteSkips(t *testing.T) {
	dir := setupSingleTrackNoRemote(t)
	defer testutil.CleanupTestRepo(t, dir)

	createTopicCommit(t, dir, "feature", "login", "login.txt", "login content")

	output, err := testutil.RunGitFlow(t, dir, "feature", "finish", "login", "--push")
	if err != nil {
		t.Fatalf("Expected finish to succeed with no remote, got error: %v\nOutput: %s", err, output)
	}

	if testutil.BranchExists(t, dir, "feature/login") {
		t.Error("Expected feature/login to be deleted")
	}
	if _, err := testutil.RunGit(t, dir, "checkout", "main"); err != nil {
		t.Fatalf("Failed to checkout main: %v", err)
	}
	if !testutil.FileExists(t, dir, "login.txt") {
		t.Error("Expected login.txt to exist in main")
	}

	lower := strings.ToLower(output)
	if !strings.Contains(lower, "push") || !strings.Contains(lower, "skip") {
		t.Errorf("Expected a skip note mentioning push. Output: %s", output)
	}
}

// TestFinishPushRunsAfterContinue tests that push (enabled via config) runs at the
// end of a resumed finish and does not run during the conflicted initial stage.
// Steps:
// 1. Sets up a classic repo with origin and gitflow.release.finish.push=true
// 2. Creates release branch '1.0.0' with conflicting content against main
// 3. Runs 'git flow release finish 1.0.0 --no-fetch' which conflicts
// 4. Verifies conflict state, that nothing was pushed, and no push header
// 5. Resolves the conflict and runs 'git flow release finish --continue 1.0.0'
// 6. Verifies exit 0, main and develop up to date with origin, and tag 1.0.0 on origin
func TestFinishPushRunsAfterContinue(t *testing.T) {
	dir, remoteDir := testutil.SetupTestRepoWithRemote(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer testutil.CleanupTestRepo(t, remoteDir)

	if _, err := testutil.RunGit(t, dir, "config", "gitflow.release.finish.push", "true"); err != nil {
		t.Fatalf("Failed to configure push: %v", err)
	}

	// Create release branch with conflicting content, then add conflicting
	// content to main so the merge into main conflicts.
	if _, err := testutil.RunGitFlow(t, dir, "release", "start", "1.0.0"); err != nil {
		t.Fatalf("Failed to start release: %v", err)
	}
	if err := testutil.WriteFile(t, dir, "conflict.txt", "release version"); err != nil {
		t.Fatalf("Failed to write conflict.txt: %v", err)
	}
	if _, err := testutil.RunGit(t, dir, "add", "conflict.txt"); err != nil {
		t.Fatalf("Failed to add conflict.txt: %v", err)
	}
	if _, err := testutil.RunGit(t, dir, "commit", "-m", "Release conflict"); err != nil {
		t.Fatalf("Failed to commit release conflict: %v", err)
	}

	if _, err := testutil.RunGit(t, dir, "checkout", "main"); err != nil {
		t.Fatalf("Failed to checkout main: %v", err)
	}
	if err := testutil.WriteFile(t, dir, "conflict.txt", "main version"); err != nil {
		t.Fatalf("Failed to write conflict.txt on main: %v", err)
	}
	if _, err := testutil.RunGit(t, dir, "add", "conflict.txt"); err != nil {
		t.Fatalf("Failed to add conflict.txt on main: %v", err)
	}
	if _, err := testutil.RunGit(t, dir, "commit", "-m", "Main conflict"); err != nil {
		t.Fatalf("Failed to commit main conflict: %v", err)
	}
	if _, err := testutil.RunGit(t, dir, "checkout", "release/1.0.0"); err != nil {
		t.Fatalf("Failed to checkout release branch: %v", err)
	}

	// Initial finish conflicts.
	conflictOutput, _ := testutil.RunGitFlow(t, dir, "release", "finish", "1.0.0", "--no-fetch")
	if !strings.Contains(conflictOutput, "conflict") && !strings.Contains(conflictOutput, "CONFLICT") {
		t.Errorf("Expected merge conflict. Output: %s", conflictOutput)
	}
	// Conflict stage must not have pushed anything.
	if _, err := os.Stat(filepath.Join(dir, ".git", "MERGE_HEAD")); os.IsNotExist(err) {
		t.Error("Expected .git/MERGE_HEAD to exist during conflict")
	}
	state, err := testutil.LoadMergeState(t, dir)
	if err != nil {
		t.Fatalf("Expected in-progress merge state: %v", err)
	}
	if state.Action != "finish" {
		t.Errorf("Expected in-progress finish, got action %q", state.Action)
	}
	if testutil.RemoteTagExists(t, dir, "origin", "1.0.0") {
		t.Error("Expected tag 1.0.0 absent from origin during conflict")
	}
	if strings.Contains(conflictOutput, pushHeader) {
		t.Errorf("Expected no push header during conflict. Output: %s", conflictOutput)
	}

	// Resolve and continue.
	if err := testutil.WriteFile(t, dir, "conflict.txt", "resolved"); err != nil {
		t.Fatalf("Failed to write conflict resolution: %v", err)
	}
	if _, err := testutil.RunGit(t, dir, "add", "conflict.txt"); err != nil {
		t.Fatalf("Failed to stage resolution: %v", err)
	}
	continueOutput, err := testutil.RunGitFlow(t, dir, "release", "finish", "--continue", "1.0.0")
	if err != nil {
		t.Fatalf("Failed to continue finish: %v\nOutput: %s", err, continueOutput)
	}

	if got := testutil.CommitsAhead(t, dir, "origin/main", "main"); got != 0 {
		t.Errorf("Expected main up to date with origin after continue, got %d ahead", got)
	}
	if got := testutil.CommitsAhead(t, dir, "origin/develop", "develop"); got != 0 {
		t.Errorf("Expected develop up to date with origin after continue, got %d ahead", got)
	}
	if !testutil.RemoteTagExists(t, dir, "origin", "1.0.0") {
		t.Error("Expected tag 1.0.0 on origin after continue")
	}
}

// TestFinishPushRejectedNonFastForward tests that a rejected (non-ff) push surfaces
// as an error while leaving the completed local finish intact.
//
// Ordering matters here because of two later features: start fetches by default (#98)
// and finish runs a parent-sync-check preflight (#99). The remote must be diverged
// *after* the topic branch is created and *without* refreshing the local origin/main
// tracking ref, so the preflight still sees main in sync and finish proceeds to the
// push. If the divergence were visible to the tracking ref, the #99 preflight would
// abort with "branch 'main' is behind 'origin/main'" before the push path is ever
// reached (the failure that this ordering deliberately avoids).
//
// Steps:
// 1. Sets up a single-track repo with origin tracking main
// 2. Creates feature branch 'login' with one commit locally, before any divergence (so start's default fetch does not refresh origin/main)
// 3. Diverges origin/main by pushing a commit from a second clone, leaving the local tracking ref stale (so the parent-sync-check sees main in sync)
// 4. Runs 'git flow feature finish login --push --no-fetch' (the stale tracking ref lets finish merge locally so only the real push is rejected)
// 5. Verifies non-zero exit and the error reports the rejected push (not the preflight)
// 6. Verifies the local finish is complete (feature deleted, main has merge, no merge state)
func TestFinishPushRejectedNonFastForward(t *testing.T) {
	dir, remoteDir := setupSingleTrackWithRemote(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer testutil.CleanupTestRepo(t, remoteDir)

	// Create the feature locally first, before diverging the remote, so the default
	// fetch on 'feature start' (#98) does not pull the divergence into origin/main.
	createTopicCommit(t, dir, "feature", "login", "login.txt", "login content")

	// Diverge origin/main via a second clone. dir does not fetch afterwards, so its
	// origin/main tracking ref stays at the base and the #99 parent-sync-check sees
	// main as in sync; only the actual push below is rejected as non-fast-forward.
	secondDir := t.TempDir()
	if _, err := testutil.RunGit(t, secondDir, "clone", remoteDir, "."); err != nil {
		t.Fatalf("Failed to clone remote: %v", err)
	}
	testutil.ConfigureGitIdentity(t, secondDir)
	if err := testutil.WriteFile(t, secondDir, "remote.txt", "remote change"); err != nil {
		t.Fatalf("Failed to write remote.txt: %v", err)
	}
	if _, err := testutil.RunGit(t, secondDir, "add", "remote.txt"); err != nil {
		t.Fatalf("Failed to add remote.txt: %v", err)
	}
	if _, err := testutil.RunGit(t, secondDir, "commit", "-m", "Remote change"); err != nil {
		t.Fatalf("Failed to commit remote change: %v", err)
	}
	if _, err := testutil.RunGit(t, secondDir, "push", "origin", "main"); err != nil {
		t.Fatalf("Failed to push remote change: %v", err)
	}

	output, err := testutil.RunGitFlow(t, dir, "feature", "finish", "login", "--push", "--no-fetch")
	if err == nil {
		t.Fatalf("Expected finish to fail on non-ff push. Output: %s", output)
	}

	lower := strings.ToLower(output)
	if !strings.Contains(lower, "reject") && !strings.Contains(lower, "non-fast-forward") && !strings.Contains(lower, "fast-forward") {
		t.Errorf("Expected error to report rejected/non-fast-forward push. Output: %s", output)
	}
	// Guard against silently passing on the #99 parent-sync-check preflight instead of
	// the push rejection: that error path never reaches the push and would leave the
	// local finish incomplete, defeating the point of this test.
	if strings.Contains(lower, "behind") && strings.Contains(lower, "stale base") {
		t.Errorf("Expected the rejected push, not the parent-sync-check preflight. Output: %s", output)
	}

	// Local finish is already complete; nothing rolled back.
	if testutil.BranchExists(t, dir, "feature/login") {
		t.Error("Expected feature/login to be deleted (local finish complete)")
	}
	if _, statErr := os.Stat(filepath.Join(dir, ".git", "MERGE_HEAD")); !os.IsNotExist(statErr) {
		t.Error("Expected no .git/MERGE_HEAD (finish complete)")
	}
	if _, loadErr := testutil.LoadMergeState(t, dir); loadErr == nil {
		t.Error("Expected no in-progress merge state (finish complete)")
	}
	if _, err := testutil.RunGit(t, dir, "checkout", "main"); err != nil {
		t.Fatalf("Failed to checkout main: %v", err)
	}
	if !testutil.FileExists(t, dir, "login.txt") {
		t.Error("Expected login.txt in main (merge completed locally)")
	}
}

// TestFinishPushConfigPushInheritsTag tests that a bare gitflow.release.finish.push=true
// (no pushtag key, no flags) pushes the created tag via the default-derivation path.
// Steps:
// 1. Sets up a classic repo with origin
// 2. Sets gitflow.release.finish.push=true (no pushtag key)
// 3. Creates release branch '1.0.0' with one commit
// 4. Runs 'git flow release finish 1.0.0 --no-fetch'
// 5. Verifies exit 0, main and develop up to date with origin, and tag 1.0.0 on origin
func TestFinishPushConfigPushInheritsTag(t *testing.T) {
	dir, remoteDir := testutil.SetupTestRepoWithRemote(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer testutil.CleanupTestRepo(t, remoteDir)

	if _, err := testutil.RunGit(t, dir, "config", "gitflow.release.finish.push", "true"); err != nil {
		t.Fatalf("Failed to configure push: %v", err)
	}

	createTopicCommit(t, dir, "release", "1.0.0", "release.txt", "release content")

	output, err := testutil.RunGitFlow(t, dir, "release", "finish", "1.0.0", "--no-fetch")
	if err != nil {
		t.Fatalf("Failed to finish release: %v\nOutput: %s", err, output)
	}

	if got := testutil.CommitsAhead(t, dir, "origin/main", "main"); got != 0 {
		t.Errorf("Expected main up to date with origin, got %d ahead", got)
	}
	if got := testutil.CommitsAhead(t, dir, "origin/develop", "develop"); got != 0 {
		t.Errorf("Expected develop up to date with origin, got %d ahead", got)
	}
	if !testutil.RemoteTagExists(t, dir, "origin", "1.0.0") {
		t.Error("Expected tag 1.0.0 on origin (pushTag inherits config push value)")
	}
}

// TestFinishPushOrderingParentThenChildren tests that --push output lists the parent
// branch before child branches, the tag after both, and dedupes the parent push line.
// Steps:
// 1. Sets up a classic repo with origin
// 2. Creates release branch '1.0.0' with one commit
// 3. Runs 'git flow release finish 1.0.0 --push --no-fetch' and captures output
// 4. Verifies main push line precedes develop push line, which precedes the tag line
// 5. Verifies the exact main push line appears exactly once (dedupe)
func TestFinishPushOrderingParentThenChildren(t *testing.T) {
	dir, remoteDir := testutil.SetupTestRepoWithRemote(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer testutil.CleanupTestRepo(t, remoteDir)

	createTopicCommit(t, dir, "release", "1.0.0", "release.txt", "release content")

	output, err := testutil.RunGitFlow(t, dir, "release", "finish", "1.0.0", "--push", "--no-fetch")
	if err != nil {
		t.Fatalf("Failed to finish release: %v\nOutput: %s", err, output)
	}

	mainIdx := strings.Index(output, pushMainLine)
	develIdx := strings.Index(output, pushDevelLine)
	tagIdx := strings.Index(output, pushTag100Line)
	if mainIdx < 0 || develIdx < 0 || tagIdx < 0 {
		t.Fatalf("Expected all three push lines present. Output: %s", output)
	}
	if !(mainIdx < develIdx) {
		t.Errorf("Expected main push line before develop push line. Output: %s", output)
	}
	if !(develIdx < tagIdx) {
		t.Errorf("Expected develop push line before tag line. Output: %s", output)
	}
	if count := strings.Count(output, pushMainLine); count != 1 {
		t.Errorf("Expected exact main push line exactly once (dedupe), got %d. Output: %s", count, output)
	}
}

// TestFinishPushShorthandForwardsFlag tests that the shorthand `git flow finish`
// (operating on the current topic branch) honors the --push flag. Regression
// guard: the shorthand originally registered the push flags but dropped them
// when calling FinishCommand, so `git flow finish --push` silently did nothing.
// Steps:
// 1. Sets up a single-track repo with origin tracking main
// 2. Creates feature branch 'login' with one commit (leaves it checked out)
// 3. Runs the shorthand 'git flow finish --push --no-fetch' (no type/name)
// 4. Verifies main is up to date with origin (the flag was forwarded)
// 5. Verifies output shows the push header and the main push line
func TestFinishPushShorthandForwardsFlag(t *testing.T) {
	dir, remoteDir := setupSingleTrackWithRemote(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer testutil.CleanupTestRepo(t, remoteDir)

	createTopicCommit(t, dir, "feature", "login", "login.txt", "login content")

	output, err := testutil.RunGitFlow(t, dir, "finish", "--push", "--no-fetch")
	if err != nil {
		t.Fatalf("Failed to finish via shorthand with --push: %v\nOutput: %s", err, output)
	}

	if got := testutil.CommitsAhead(t, dir, "origin/main", "main"); got != 0 {
		t.Errorf("Expected main up to date with origin (shorthand --push forwarded), got %d commits ahead. Output: %s", got, output)
	}
	if !strings.Contains(output, pushHeader) || !strings.Contains(output, pushMainLine) {
		t.Errorf("Expected push header and main push line via shorthand. Output: %s", output)
	}
}

// TestFinishFetchShorthandForwardsFlag tests that the shorthand `git flow finish`
// (operating on the current topic branch) honors the --fetch flag. Regression
// guard: the shorthand registered the fetch flags but dropped them when calling
// FinishCommand (passing nil), so `git flow finish --fetch` silently fell back to
// config/defaults instead of honoring the flag.
// Steps:
//  1. Sets up a single-track repo with origin tracking main
//  2. Disables fetch via config so the default alone would not fetch
//  3. Creates feature branch 'login' with one commit (leaves it checked out)
//  4. Runs the shorthand 'git flow finish --fetch' (no type/name)
//  5. Verifies the fetch message appears, proving the flag was forwarded and
//     overrode the config
func TestFinishFetchShorthandForwardsFlag(t *testing.T) {
	dir, remoteDir := setupSingleTrackWithRemote(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer testutil.CleanupTestRepo(t, remoteDir)

	// Disable fetch via config so a dropped --fetch flag would result in no fetch.
	// The flag must override this config to prove it is forwarded.
	if _, err := testutil.RunGit(t, dir, "config", "gitflow.feature.finish.fetch", "false"); err != nil {
		t.Fatalf("Failed to configure fetch option: %v", err)
	}

	createTopicCommit(t, dir, "feature", "login", "login.txt", "login content")

	output, err := testutil.RunGitFlow(t, dir, "finish", "--fetch")
	if err != nil {
		t.Fatalf("Failed to finish via shorthand with --fetch: %v\nOutput: %s", err, output)
	}

	if !strings.Contains(output, "Fetching from remote") {
		t.Errorf("Expected fetch to occur via shorthand --fetch (flag forwarded, overriding config). Output: %s", output)
	}
}
