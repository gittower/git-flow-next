package cmd_test

import (
	"strings"
	"testing"

	"github.com/gittower/git-flow-next/test/testutil"
)

// Command-level coverage for verbatim topic branch prefixes (issue #189).
//
// Interactive `git flow init` used to append a `/` to every entered topic
// branch prefix, so `feature_` became `feature_/`. Every other write path
// (init prefix flags, `config add/edit topic --prefix`, git-flow-avh import,
// direct `git config`) already stored the prefix verbatim, and every runtime
// consumer treats the prefix as an opaque string. This file pins the whole
// "prefixes are stored and used verbatim" property in one place: the five
// interactive prompts, the other write paths, and the runtime commands that
// consume a flat prefix.
//
// `interactiveConfig()` asks eight questions in this order — production
// branch, development branch, feature/bugfix/release/hotfix/support prefix,
// version tag prefix — so every interactive scenario supplies eight stdin
// lines.

// TestInitInteractiveFlatFeaturePrefixEndToEnd covers spec scenario 1: the
// reported regression, end to end.
// Steps:
// 1. Sets up a test repository without git-flow configuration
// 2. Runs 'git flow init' answering only the feature prompt with 'feature_'
// 3. Verifies gitflow.branch.feature.prefix is stored as 'feature_'
// 4. Runs 'git flow feature start login'
// 5. Verifies branch feature_login exists and is checked out, and that no feature_/login branch was created
func TestInitInteractiveFlatFeaturePrefixEndToEnd(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	output, err := testutil.RunGitFlowWithInput(t, dir, "\n\nfeature_\n\n\n\n\n\n", "init")
	if err != nil {
		t.Fatalf("Failed to run git-flow init: %v\nOutput: %s", err, output)
	}

	if prefix := testutil.GitConfigValue(t, dir, "gitflow.branch.feature.prefix"); prefix != "feature_" {
		t.Errorf("Expected gitflow.branch.feature.prefix to be 'feature_', got: %s", prefix)
	}

	output, err = testutil.RunGitFlow(t, dir, "feature", "start", "login")
	if err != nil {
		t.Fatalf("Failed to start feature branch: %v\nOutput: %s", err, output)
	}

	if !testutil.BranchExists(t, dir, "feature_login") {
		t.Error("Expected branch 'feature_login' to exist")
	}
	if current := testutil.GetCurrentBranch(t, dir); current != "feature_login" {
		t.Errorf("Expected current branch to be 'feature_login', got: %s", current)
	}
	if testutil.BranchExists(t, dir, "feature_/login") {
		t.Error("Expected no branch named 'feature_/login' to exist")
	}
}

// TestInitInteractiveAllPrefixesStoredVerbatim covers spec scenario 2: all five
// prefix prompts accept flat input.
// Steps:
// 1. Sets up a test repository without git-flow configuration
// 2. Runs 'git flow init' answering all five prefix prompts with flat values
// 3. Verifies each gitflow.branch.<type>.prefix is stored exactly as entered
func TestInitInteractiveAllPrefixesStoredVerbatim(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	input := "\n\nfeature_\nbugfix_\nrelease-\nhotfix-\nsupport_\n\n"
	output, err := testutil.RunGitFlowWithInput(t, dir, input, "init")
	if err != nil {
		t.Fatalf("Failed to run git-flow init: %v\nOutput: %s", err, output)
	}

	if prefix := testutil.GitConfigValue(t, dir, "gitflow.branch.feature.prefix"); prefix != "feature_" {
		t.Errorf("Expected gitflow.branch.feature.prefix to be 'feature_', got: %s", prefix)
	}
	if prefix := testutil.GitConfigValue(t, dir, "gitflow.branch.bugfix.prefix"); prefix != "bugfix_" {
		t.Errorf("Expected gitflow.branch.bugfix.prefix to be 'bugfix_', got: %s", prefix)
	}
	if prefix := testutil.GitConfigValue(t, dir, "gitflow.branch.release.prefix"); prefix != "release-" {
		t.Errorf("Expected gitflow.branch.release.prefix to be 'release-', got: %s", prefix)
	}
	if prefix := testutil.GitConfigValue(t, dir, "gitflow.branch.hotfix.prefix"); prefix != "hotfix-" {
		t.Errorf("Expected gitflow.branch.hotfix.prefix to be 'hotfix-', got: %s", prefix)
	}
	if prefix := testutil.GitConfigValue(t, dir, "gitflow.branch.support.prefix"); prefix != "support_" {
		t.Errorf("Expected gitflow.branch.support.prefix to be 'support_', got: %s", prefix)
	}
}

// TestInitInteractiveSlashSuffixedPrefixUnchanged covers spec scenario 3: a
// slash-suffixed answer is stored as-is, not double-slashed.
// Steps:
// 1. Sets up a test repository without git-flow configuration
// 2. Runs 'git flow init' answering the feature prompt with 'feat/'
// 3. Verifies gitflow.branch.feature.prefix is 'feat/', not 'feat//'
func TestInitInteractiveSlashSuffixedPrefixUnchanged(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	output, err := testutil.RunGitFlowWithInput(t, dir, "\n\nfeat/\n\n\n\n\n\n", "init")
	if err != nil {
		t.Fatalf("Failed to run git-flow init: %v\nOutput: %s", err, output)
	}

	if prefix := testutil.GitConfigValue(t, dir, "gitflow.branch.feature.prefix"); prefix != "feat/" {
		t.Errorf("Expected gitflow.branch.feature.prefix to be 'feat/', got: %s", prefix)
	}
}

// TestInitInteractiveEmptyInputKeepsDefaultPrefixes covers spec scenario 4:
// pressing Enter at every prompt keeps the default prefixes.
// Steps:
// 1. Sets up a test repository without git-flow configuration
// 2. Runs 'git flow init' answering every prompt with an empty line
// 3. Verifies the five topic prefixes keep their hierarchical defaults
func TestInitInteractiveEmptyInputKeepsDefaultPrefixes(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	output, err := testutil.RunGitFlowWithInput(t, dir, "\n\n\n\n\n\n\n\n", "init")
	if err != nil {
		t.Fatalf("Failed to run git-flow init: %v\nOutput: %s", err, output)
	}

	if prefix := testutil.GitConfigValue(t, dir, "gitflow.branch.feature.prefix"); prefix != "feature/" {
		t.Errorf("Expected gitflow.branch.feature.prefix to be 'feature/', got: %s", prefix)
	}
	if prefix := testutil.GitConfigValue(t, dir, "gitflow.branch.bugfix.prefix"); prefix != "bugfix/" {
		t.Errorf("Expected gitflow.branch.bugfix.prefix to be 'bugfix/', got: %s", prefix)
	}
	if prefix := testutil.GitConfigValue(t, dir, "gitflow.branch.release.prefix"); prefix != "release/" {
		t.Errorf("Expected gitflow.branch.release.prefix to be 'release/', got: %s", prefix)
	}
	if prefix := testutil.GitConfigValue(t, dir, "gitflow.branch.hotfix.prefix"); prefix != "hotfix/" {
		t.Errorf("Expected gitflow.branch.hotfix.prefix to be 'hotfix/', got: %s", prefix)
	}
	if prefix := testutil.GitConfigValue(t, dir, "gitflow.branch.support.prefix"); prefix != "support/" {
		t.Errorf("Expected gitflow.branch.support.prefix to be 'support/', got: %s", prefix)
	}
}

// TestInitInteractiveTrimsWhitespaceAroundPrefix covers spec scenario 5:
// surrounding whitespace is trimmed from a prefix answer.
// Steps:
// 1. Sets up a test repository without git-flow configuration
// 2. Runs 'git flow init' answering the feature prompt with '  feature_  '
// 3. Reads gitflow.branch.feature.prefix raw (git preserves trailing spaces in a config value, and testutil.GitConfigValue would trim them away)
// 4. Verifies the stored value is exactly 'feature_'
func TestInitInteractiveTrimsWhitespaceAroundPrefix(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	output, err := testutil.RunGitFlowWithInput(t, dir, "\n\n  feature_  \n\n\n\n\n\n", "init")
	if err != nil {
		t.Fatalf("Failed to run git-flow init: %v\nOutput: %s", err, output)
	}

	raw, err := testutil.RunGit(t, dir, "config", "--local", "--get", "gitflow.branch.feature.prefix")
	if err != nil {
		t.Fatalf("Failed to read gitflow.branch.feature.prefix: %v\nOutput: %s", err, raw)
	}
	if prefix := strings.TrimRight(raw, "\n"); prefix != "feature_" {
		t.Errorf("Expected gitflow.branch.feature.prefix to be 'feature_', got: %q", prefix)
	}
}

// TestInitFlagFlatFeaturePrefixStoredVerbatim covers spec scenario 6: the init
// --feature flag stores a flat prefix verbatim.
// Steps:
// 1. Sets up a test repository without git-flow configuration
// 2. Runs 'git flow init --defaults --feature feature_'
// 3. Verifies gitflow.branch.feature.prefix is stored as 'feature_'
func TestInitFlagFlatFeaturePrefixStoredVerbatim(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults", "--feature", "feature_")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	if prefix := testutil.GitConfigValue(t, dir, "gitflow.branch.feature.prefix"); prefix != "feature_" {
		t.Errorf("Expected gitflow.branch.feature.prefix to be 'feature_', got: %s", prefix)
	}
}

// TestConfigAddTopicFlatPrefixStoredVerbatim covers spec scenario 7:
// 'config add topic --prefix' stores a flat prefix verbatim.
// Steps:
// 1. Sets up a test repository and initializes git-flow with defaults
// 2. Runs 'git flow config add topic epic develop --prefix epic_'
// 3. Verifies gitflow.branch.epic.prefix is stored as 'epic_'
func TestConfigAddTopicFlatPrefixStoredVerbatim(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}

	output, err = testutil.RunGitFlow(t, dir, "config", "add", "topic", "epic", "develop", "--prefix", "epic_")
	if err != nil {
		t.Fatalf("Failed to add topic branch type: %v\nOutput: %s", err, output)
	}

	if prefix := testutil.GitConfigValue(t, dir, "gitflow.branch.epic.prefix"); prefix != "epic_" {
		t.Errorf("Expected gitflow.branch.epic.prefix to be 'epic_', got: %s", prefix)
	}
}

// TestStartWithFlatFeaturePrefix covers spec scenario 8: start builds the
// branch name from a flat prefix.
// Steps:
// 1. Sets up a test repository and initializes git-flow with defaults
// 2. Sets gitflow.branch.feature.prefix to 'feature_'
// 3. Runs 'git flow feature start login'
// 4. Verifies feature_login exists and is checked out, and feature/login does not exist
func TestStartWithFlatFeaturePrefix(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}
	if out, err := testutil.RunGit(t, dir, "config", "gitflow.branch.feature.prefix", "feature_"); err != nil {
		t.Fatalf("Failed to set flat feature prefix: %v\nOutput: %s", err, out)
	}

	output, err = testutil.RunGitFlow(t, dir, "feature", "start", "login")
	if err != nil {
		t.Fatalf("Failed to start feature branch: %v\nOutput: %s", err, output)
	}

	if !testutil.BranchExists(t, dir, "feature_login") {
		t.Error("Expected branch 'feature_login' to exist")
	}
	if current := testutil.GetCurrentBranch(t, dir); current != "feature_login" {
		t.Errorf("Expected current branch to be 'feature_login', got: %s", current)
	}
	if testutil.BranchExists(t, dir, "feature/login") {
		t.Error("Expected no branch named 'feature/login' to exist")
	}
}

// TestFinishByNameWithFlatFeaturePrefix covers spec scenario 9: finishing by
// short name resolves a flat-prefixed branch.
// Steps:
// 1. Sets up a test repository and initializes git-flow with defaults
// 2. Sets gitflow.branch.feature.prefix to 'feature_'
// 3. Runs 'git flow feature start login' and commits feature.txt
// 4. Runs 'git flow feature finish login'
// 5. Verifies the command succeeds without the missing-prefix warning, feature_login is deleted, and feature.txt is present on develop
func TestFinishByNameWithFlatFeaturePrefix(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}
	if out, err := testutil.RunGit(t, dir, "config", "gitflow.branch.feature.prefix", "feature_"); err != nil {
		t.Fatalf("Failed to set flat feature prefix: %v\nOutput: %s", err, out)
	}

	output, err = testutil.RunGitFlow(t, dir, "feature", "start", "login")
	if err != nil {
		t.Fatalf("Failed to start feature branch: %v\nOutput: %s", err, output)
	}
	if err := testutil.WriteFile(t, dir, "feature.txt", "feature content"); err != nil {
		t.Fatalf("Failed to write feature.txt: %v", err)
	}
	if out, err := testutil.RunGit(t, dir, "add", "feature.txt"); err != nil {
		t.Fatalf("Failed to add feature.txt: %v\nOutput: %s", err, out)
	}
	if out, err := testutil.RunGit(t, dir, "commit", "-m", "Add feature file"); err != nil {
		t.Fatalf("Failed to commit feature file: %v\nOutput: %s", err, out)
	}

	output, err = testutil.RunGitFlow(t, dir, "feature", "finish", "login")
	if err != nil {
		t.Fatalf("Failed to finish feature branch: %v\nOutput: %s", err, output)
	}
	if strings.Contains(output, "not a standard") {
		t.Errorf("Expected no missing-prefix warning, got: %s", output)
	}

	if testutil.BranchExists(t, dir, "feature_login") {
		t.Error("Expected branch 'feature_login' to be deleted")
	}
	if out, err := testutil.RunGit(t, dir, "checkout", "develop"); err != nil {
		t.Fatalf("Failed to checkout develop: %v\nOutput: %s", err, out)
	}
	if !testutil.FileExists(t, dir, "feature.txt") {
		t.Error("Expected feature.txt to exist on develop")
	}
}

// TestFinishCurrentBranchWithFlatFeaturePrefix covers spec scenario 10:
// finishing without a name derives the short name from a flat-prefixed branch.
// Steps:
// 1. Sets up a test repository and initializes git-flow with defaults
// 2. Sets gitflow.branch.feature.prefix to 'feature_'
// 3. Runs 'git flow feature start login' and commits feature.txt
// 4. Runs 'git flow feature finish' while still on feature_login
// 5. Verifies feature_login is deleted and feature.txt is present on develop
func TestFinishCurrentBranchWithFlatFeaturePrefix(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}
	if out, err := testutil.RunGit(t, dir, "config", "gitflow.branch.feature.prefix", "feature_"); err != nil {
		t.Fatalf("Failed to set flat feature prefix: %v\nOutput: %s", err, out)
	}

	output, err = testutil.RunGitFlow(t, dir, "feature", "start", "login")
	if err != nil {
		t.Fatalf("Failed to start feature branch: %v\nOutput: %s", err, output)
	}
	if err := testutil.WriteFile(t, dir, "feature.txt", "feature content"); err != nil {
		t.Fatalf("Failed to write feature.txt: %v", err)
	}
	if out, err := testutil.RunGit(t, dir, "add", "feature.txt"); err != nil {
		t.Fatalf("Failed to add feature.txt: %v\nOutput: %s", err, out)
	}
	if out, err := testutil.RunGit(t, dir, "commit", "-m", "Add feature file"); err != nil {
		t.Fatalf("Failed to commit feature file: %v\nOutput: %s", err, out)
	}

	output, err = testutil.RunGitFlow(t, dir, "feature", "finish")
	if err != nil {
		t.Fatalf("Failed to finish current feature branch: %v\nOutput: %s", err, output)
	}
	if strings.Contains(output, "not a standard") {
		t.Errorf("Expected no missing-prefix warning, got: %s", output)
	}

	if testutil.BranchExists(t, dir, "feature_login") {
		t.Error("Expected branch 'feature_login' to be deleted")
	}
	if out, err := testutil.RunGit(t, dir, "checkout", "develop"); err != nil {
		t.Fatalf("Failed to checkout develop: %v\nOutput: %s", err, out)
	}
	if !testutil.FileExists(t, dir, "feature.txt") {
		t.Error("Expected feature.txt to exist on develop")
	}
}

// TestListWithFlatFeaturePrefix covers spec scenario 11: list filters branches
// on a flat prefix and strips it from the printed names.
// Steps:
// 1. Sets up a test repository and initializes git-flow with defaults
// 2. Sets gitflow.branch.feature.prefix to 'feature_'
// 3. Runs 'git flow feature start login'
// 4. Creates decoy branches other_branch and feature/legacy from develop
// 5. Runs 'git flow feature list'
// 6. Verifies the output lists 'login' with the prefix stripped and lists neither decoy
func TestListWithFlatFeaturePrefix(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}
	if out, err := testutil.RunGit(t, dir, "config", "gitflow.branch.feature.prefix", "feature_"); err != nil {
		t.Fatalf("Failed to set flat feature prefix: %v\nOutput: %s", err, out)
	}

	output, err = testutil.RunGitFlow(t, dir, "feature", "start", "login")
	if err != nil {
		t.Fatalf("Failed to start feature branch: %v\nOutput: %s", err, output)
	}

	if out, err := testutil.RunGit(t, dir, "checkout", "develop"); err != nil {
		t.Fatalf("Failed to checkout develop: %v\nOutput: %s", err, out)
	}
	if out, err := testutil.RunGit(t, dir, "branch", "other_branch"); err != nil {
		t.Fatalf("Failed to create decoy branch 'other_branch': %v\nOutput: %s", err, out)
	}
	if out, err := testutil.RunGit(t, dir, "branch", "feature/legacy"); err != nil {
		t.Fatalf("Failed to create decoy branch 'feature/legacy': %v\nOutput: %s", err, out)
	}

	output, err = testutil.RunGitFlow(t, dir, "feature", "list")
	if err != nil {
		t.Fatalf("Failed to list feature branches: %v\nOutput: %s", err, output)
	}

	if !strings.Contains(output, "login") {
		t.Errorf("Expected output to list 'login', got: %s", output)
	}
	if strings.Contains(output, "feature_login") {
		t.Errorf("Expected output to strip the prefix, got: %s", output)
	}
	if strings.Contains(output, "other_branch") {
		t.Errorf("Expected output not to list 'other_branch', got: %s", output)
	}
	if strings.Contains(output, "legacy") {
		t.Errorf("Expected output not to list 'legacy', got: %s", output)
	}
}

// TestShorthandFinishWithFlatFeaturePrefix covers spec scenario 12: the
// shorthand finish detects the branch type from a flat prefix.
// Steps:
// 1. Sets up a test repository and initializes git-flow with defaults
// 2. Sets gitflow.branch.feature.prefix to 'feature_'
// 3. Runs 'git flow feature start login' and commits feature.txt
// 4. Runs 'git flow finish' with no type and no name
// 5. Verifies the type was detected unambiguously, feature_login is deleted, and feature.txt is present on develop
func TestShorthandFinishWithFlatFeaturePrefix(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}
	if out, err := testutil.RunGit(t, dir, "config", "gitflow.branch.feature.prefix", "feature_"); err != nil {
		t.Fatalf("Failed to set flat feature prefix: %v\nOutput: %s", err, out)
	}

	output, err = testutil.RunGitFlow(t, dir, "feature", "start", "login")
	if err != nil {
		t.Fatalf("Failed to start feature branch: %v\nOutput: %s", err, output)
	}
	if err := testutil.WriteFile(t, dir, "feature.txt", "feature content"); err != nil {
		t.Fatalf("Failed to write feature.txt: %v", err)
	}
	if out, err := testutil.RunGit(t, dir, "add", "feature.txt"); err != nil {
		t.Fatalf("Failed to add feature.txt: %v\nOutput: %s", err, out)
	}
	if out, err := testutil.RunGit(t, dir, "commit", "-m", "Add feature file"); err != nil {
		t.Fatalf("Failed to commit feature file: %v\nOutput: %s", err, out)
	}

	output, err = testutil.RunGitFlow(t, dir, "finish")
	if err != nil {
		t.Fatalf("Failed to run shorthand finish: %v\nOutput: %s", err, output)
	}
	if strings.Contains(output, "Ambiguous branch") {
		t.Errorf("Expected unambiguous branch type detection, got: %s", output)
	}

	if testutil.BranchExists(t, dir, "feature_login") {
		t.Error("Expected branch 'feature_login' to be deleted")
	}
	if out, err := testutil.RunGit(t, dir, "checkout", "develop"); err != nil {
		t.Fatalf("Failed to checkout develop: %v\nOutput: %s", err, out)
	}
	if !testutil.FileExists(t, dir, "feature.txt") {
		t.Error("Expected feature.txt to exist on develop")
	}
}

// TestRenameWithFlatFeaturePrefix covers spec scenario 13: rename rebuilds both
// branch names from a flat prefix.
// Steps:
// 1. Sets up a test repository and initializes git-flow with defaults
// 2. Sets gitflow.branch.feature.prefix to 'feature_'
// 3. Runs 'git flow feature start login'
// 4. Runs 'git flow feature rename login signup'
// 5. Verifies feature_signup exists and is checked out, and feature_login is gone
func TestRenameWithFlatFeaturePrefix(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}
	if out, err := testutil.RunGit(t, dir, "config", "gitflow.branch.feature.prefix", "feature_"); err != nil {
		t.Fatalf("Failed to set flat feature prefix: %v\nOutput: %s", err, out)
	}

	output, err = testutil.RunGitFlow(t, dir, "feature", "start", "login")
	if err != nil {
		t.Fatalf("Failed to start feature branch: %v\nOutput: %s", err, output)
	}

	output, err = testutil.RunGitFlow(t, dir, "feature", "rename", "login", "signup")
	if err != nil {
		t.Fatalf("Failed to rename feature branch: %v\nOutput: %s", err, output)
	}

	if !testutil.BranchExists(t, dir, "feature_signup") {
		t.Error("Expected branch 'feature_signup' to exist")
	}
	if testutil.BranchExists(t, dir, "feature_login") {
		t.Error("Expected branch 'feature_login' to be gone")
	}
	if current := testutil.GetCurrentBranch(t, dir); current != "feature_signup" {
		t.Errorf("Expected current branch to be 'feature_signup', got: %s", current)
	}
}

// TestInitImportsFlatAVHFeaturePrefix covers spec scenario 14: importing a
// git-flow-avh configuration keeps a flat prefix verbatim.
// Steps:
// 1. Sets up a test repository without git-flow-next configuration
// 2. Writes avh-style config: gitflow.branch.master, gitflow.branch.develop, and a flat gitflow.prefix.feature of 'feature_'
// 3. Runs 'git flow init' with no flags, which takes the import short-circuit
// 4. Verifies the import ran and gitflow.branch.feature.prefix is 'feature_'
// 5. Runs 'git flow feature start x' and verifies feature_x is checked out
func TestInitImportsFlatAVHFeaturePrefix(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	if out, err := testutil.RunGit(t, dir, "config", "gitflow.branch.master", "main"); err != nil {
		t.Fatalf("Failed to set gitflow.branch.master: %v\nOutput: %s", err, out)
	}
	if out, err := testutil.RunGit(t, dir, "config", "gitflow.branch.develop", "develop"); err != nil {
		t.Fatalf("Failed to set gitflow.branch.develop: %v\nOutput: %s", err, out)
	}
	if out, err := testutil.RunGit(t, dir, "config", "gitflow.prefix.feature", "feature_"); err != nil {
		t.Fatalf("Failed to set gitflow.prefix.feature: %v\nOutput: %s", err, out)
	}

	output, err := testutil.RunGitFlow(t, dir, "init")
	if err != nil {
		t.Fatalf("Failed to run git-flow init: %v\nOutput: %s", err, output)
	}
	if !strings.Contains(output, "Found existing git-flow-avh configuration, importing") {
		t.Errorf("Expected the avh import path to run, got: %s", output)
	}

	if prefix := testutil.GitConfigValue(t, dir, "gitflow.branch.feature.prefix"); prefix != "feature_" {
		t.Errorf("Expected gitflow.branch.feature.prefix to be 'feature_', got: %s", prefix)
	}

	output, err = testutil.RunGitFlow(t, dir, "feature", "start", "x")
	if err != nil {
		t.Fatalf("Failed to start feature branch: %v\nOutput: %s", err, output)
	}
	if !testutil.BranchExists(t, dir, "feature_x") {
		t.Error("Expected branch 'feature_x' to exist")
	}
	if current := testutil.GetCurrentBranch(t, dir); current != "feature_x" {
		t.Errorf("Expected current branch to be 'feature_x', got: %s", current)
	}
}

// TestFinishByFullBranchNameWithFlatFeaturePrefix covers the added scenario 15:
// finishing by full branch name does not trip the missing-prefix confirmation
// prompt when the prefix is flat.
// Steps:
// 1. Sets up a test repository and initializes git-flow with defaults
// 2. Sets gitflow.branch.feature.prefix to 'feature_'
// 3. Runs 'git flow feature start login' and commits feature.txt
// 4. Runs 'git flow feature finish feature_login' using the full branch name
// 5. Verifies the command succeeds without the non-standard-branch warning, feature_login is deleted, and feature.txt is present on develop
func TestFinishByFullBranchNameWithFlatFeaturePrefix(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	output, err := testutil.RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, output)
	}
	if out, err := testutil.RunGit(t, dir, "config", "gitflow.branch.feature.prefix", "feature_"); err != nil {
		t.Fatalf("Failed to set flat feature prefix: %v\nOutput: %s", err, out)
	}

	output, err = testutil.RunGitFlow(t, dir, "feature", "start", "login")
	if err != nil {
		t.Fatalf("Failed to start feature branch: %v\nOutput: %s", err, output)
	}
	if err := testutil.WriteFile(t, dir, "feature.txt", "feature content"); err != nil {
		t.Fatalf("Failed to write feature.txt: %v", err)
	}
	if out, err := testutil.RunGit(t, dir, "add", "feature.txt"); err != nil {
		t.Fatalf("Failed to add feature.txt: %v\nOutput: %s", err, out)
	}
	if out, err := testutil.RunGit(t, dir, "commit", "-m", "Add feature file"); err != nil {
		t.Fatalf("Failed to commit feature file: %v\nOutput: %s", err, out)
	}

	output, err = testutil.RunGitFlow(t, dir, "feature", "finish", "feature_login")
	if err != nil {
		t.Fatalf("Failed to finish feature branch by full name: %v\nOutput: %s", err, output)
	}
	if strings.Contains(output, "Warning: Branch 'feature_login' is not a standard feature branch") {
		t.Errorf("Expected no non-standard-branch warning, got: %s", output)
	}

	if testutil.BranchExists(t, dir, "feature_login") {
		t.Error("Expected branch 'feature_login' to be deleted")
	}
	if out, err := testutil.RunGit(t, dir, "checkout", "develop"); err != nil {
		t.Fatalf("Failed to checkout develop: %v\nOutput: %s", err, out)
	}
	if !testutil.FileExists(t, dir, "feature.txt") {
		t.Error("Expected feature.txt to exist on develop")
	}
}

// TestInitInteractiveWhitespaceOnlyPrefixKeepsDefault covers the added scenario
// 16: a whitespace-only prefix answer is trimmed to empty and keeps the default.
// Steps:
// 1. Sets up a test repository without git-flow configuration
// 2. Runs 'git flow init' answering the feature prompt with three spaces
// 3. Verifies gitflow.branch.feature.prefix keeps the default 'feature/'
func TestInitInteractiveWhitespaceOnlyPrefixKeepsDefault(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	output, err := testutil.RunGitFlowWithInput(t, dir, "\n\n   \n\n\n\n\n\n", "init")
	if err != nil {
		t.Fatalf("Failed to run git-flow init: %v\nOutput: %s", err, output)
	}

	if prefix := testutil.GitConfigValue(t, dir, "gitflow.branch.feature.prefix"); prefix != "feature/" {
		t.Errorf("Expected gitflow.branch.feature.prefix to be 'feature/', got: %s", prefix)
	}
}
