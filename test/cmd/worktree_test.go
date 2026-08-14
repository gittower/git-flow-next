package cmd_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gittower/git-flow-next/test/testutil"
)

// worktreeExitCode returns the process exit code carried by a testutil.ExitError,
// 0 for a nil error, or -1 for any other error type.
func worktreeExitCode(err error) int {
	if err == nil {
		return 0
	}
	if ee, ok := err.(*testutil.ExitError); ok {
		return ee.ExitCode
	}
	return -1
}

// worktreeRootFor returns the directory the default path template puts worktrees
// in: a sibling of the repository named "<repo>-worktrees". It lives OUTSIDE the
// test repository, so testutil.CleanupTestRepo (a plain RemoveAll of the repo
// directory) never touches it — every test that creates a worktree at the
// computed path must remove this root itself.
func worktreeRootFor(dir string) string {
	return dir + "-worktrees"
}

// computedWorktreePath returns the path the default template
// (../{{ repo }}-worktrees/{{ branch }}) computes for branch in the repository at
// dir. The repository root is symlink-resolved because git reports the resolved
// form, and the (possibly not yet created) remainder is appended with Join —
// filepath.EvalSymlinks fails on a path that does not exist, so it must never be
// applied to the leaf.
func computedWorktreePath(t *testing.T, dir string, branch string) string {
	t.Helper()
	root := testutil.EvalPath(t, dir)
	return filepath.Join(filepath.Dir(root), filepath.Base(root)+"-worktrees", filepath.FromSlash(branch))
}

// initWorktreeRepo creates a temporary repository with git-flow initialized to
// its defaults.
func initWorktreeRepo(t *testing.T) string {
	t.Helper()
	dir := testutil.SetupTestRepo(t)
	if out, err := testutil.RunGitFlow(t, dir, "init", "--defaults"); err != nil {
		testutil.CleanupTestRepo(t, dir)
		t.Fatalf("Failed to initialize git-flow: %v\nOutput: %s", err, out)
	}
	return dir
}

// addWorktree runs 'git flow worktree add <branch>' and returns the computed path
// the worktree was created at.
func addWorktree(t *testing.T, dir string, branch string) string {
	t.Helper()
	if out, err := testutil.RunGitFlow(t, dir, "worktree", "add", branch); err != nil {
		t.Fatalf("Failed to add worktree for %s: %v\nOutput: %s", branch, err, out)
	}
	return computedWorktreePath(t, dir, branch)
}

// createFreeBranch creates branch off main without checking it out, so a worktree
// can be added for it.
func createFreeBranch(t *testing.T, dir string, branch string) {
	t.Helper()
	if out, err := testutil.RunGit(t, dir, "branch", branch, "main"); err != nil {
		t.Fatalf("Failed to create branch %s: %v\nOutput: %s", branch, err, out)
	}
}

// cdFilePath creates an empty file in a temporary directory and returns its path,
// for use as GIT_FLOW_CD_FILE. Zero length is the pre-state every navigation
// assertion compares against.
func cdFilePath(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "cd-destination")
	if err := os.WriteFile(path, nil, 0644); err != nil {
		t.Fatalf("Failed to create CD file: %v", err)
	}
	return path
}

// cdEnv builds the subprocess environment that points GIT_FLOW_CD_FILE at path.
func cdEnv(path string) []string {
	return []string{"GIT_FLOW_CD_FILE=" + path}
}

// readCDFile returns the trimmed content of the navigation destination file.
func readCDFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read CD file %s: %v", path, err)
	}
	return strings.TrimSpace(string(data))
}

// assertCDFileEmpty fails the test unless the navigation destination file is
// still zero-length.
func assertCDFileEmpty(t *testing.T, path string) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Failed to stat CD file %s: %v", path, err)
	}
	if info.Size() != 0 {
		t.Errorf("Expected CD file to stay empty, got %q", readCDFile(t, path))
	}
}

// worktreeRows splits 'git flow worktree list' output into its non-empty rows.
func worktreeRows(output string) []string {
	var rows []string
	for _, line := range strings.Split(output, "\n") {
		if strings.TrimSpace(line) != "" {
			rows = append(rows, strings.TrimSpace(line))
		}
	}
	return rows
}

// gitWorktreeList returns the raw output of 'git worktree list' for the repo.
func gitWorktreeList(t *testing.T, dir string) string {
	t.Helper()
	out, err := testutil.RunGit(t, dir, "worktree", "list")
	if err != nil {
		t.Fatalf("git worktree list failed: %v\nOutput: %s", err, out)
	}
	return out
}

// ---------------------------------------------------------------------------
// Group A — path computation (CLI level)
// ---------------------------------------------------------------------------

// TestWorktreePathDefaultTemplate covers scenario 1: the default template
// resolves to an absolute sibling directory of the repository.
// Steps:
// 1. Sets up a repository with git-flow defaults and a feature/user-auth branch
// 2. Leaves gitflow.worktreePath unset
// 3. Runs 'git flow worktree path feature/user-auth'
// 4. Verifies stdout is exactly <repoParent>/<repo>-worktrees/feature/user-auth
func TestWorktreePathDefaultTemplate(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	createFreeBranch(t, dir, "feature/user-auth")

	stdout, stderr, err := testutil.RunGitFlowStreams(t, dir, "worktree", "path", "feature/user-auth")
	if err != nil {
		t.Fatalf("worktree path failed: %v\nStderr: %s", err, stderr)
	}

	want := computedWorktreePath(t, dir, "feature/user-auth") + "\n"
	if stdout != want {
		t.Errorf("Expected stdout %q, got %q", want, stdout)
	}
}

// TestWorktreePathTildeTemplateWithBranchName covers scenario 2: a ~-rooted
// template expands to the home directory and {{ branchName }} drops the prefix.
// Steps:
// 1. Sets gitflow.worktreePath to '~/wt/{{ branchName }}' (camelCase key on purpose)
// 2. Creates branch feature/user-auth
// 3. Runs 'git flow worktree path feature/user-auth' with HOME overridden
// 4. Verifies the output is <HOME>/wt/user-auth, compared against the raw HOME value
func TestWorktreePathTildeTemplateWithBranchName(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	createFreeBranch(t, dir, "feature/user-auth")

	if out, err := testutil.RunGit(t, dir, "config", "gitflow.worktreePath", "~/wt/{{ branchName }}"); err != nil {
		t.Fatalf("Failed to set gitflow.worktreePath: %v\nOutput: %s", err, out)
	}

	home := t.TempDir()
	output, err := testutil.RunGitFlowWithEnv(t, dir, []string{"HOME=" + home}, "worktree", "path", "feature/user-auth")
	if err != nil {
		t.Fatalf("worktree path failed: %v\nOutput: %s", err, output)
	}

	// os.UserHomeDir returns the HOME string verbatim, so the expectation is
	// deliberately NOT symlink-resolved on either side.
	want := filepath.Join(home, "wt", "user-auth")
	if got := strings.TrimSpace(output); got != want {
		t.Errorf("Expected path %q, got %q", want, got)
	}
}

// TestWorktreePathTopicTypeAndUnspacedForms covers scenario 3: {{ topicType }}
// expands to the topic branch type and unspaced placeholders behave identically.
// Steps:
// 1. Sets gitflow.worktreePath to '../wt/{{topicType}}/{{ branch }}'
// 2. Runs 'git flow worktree path feature/user-auth'
// 3. Verifies the result is <repoParent>/wt/feature/feature/user-auth
// 4. Repeats with the spaced/unspaced forms swapped and requires identical output
func TestWorktreePathTopicTypeAndUnspacedForms(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	createFreeBranch(t, dir, "feature/user-auth")

	if out, err := testutil.RunGit(t, dir, "config", "gitflow.worktreePath", "../wt/{{topicType}}/{{ branch }}"); err != nil {
		t.Fatalf("Failed to set gitflow.worktreePath: %v\nOutput: %s", err, out)
	}

	unspaced, err := testutil.RunGitFlow(t, dir, "worktree", "path", "feature/user-auth")
	if err != nil {
		t.Fatalf("worktree path failed: %v\nOutput: %s", err, unspaced)
	}

	repoParent := filepath.Dir(testutil.EvalPath(t, dir))
	want := filepath.Join(repoParent, "wt", "feature", "feature", "user-auth")
	if got := strings.TrimSpace(unspaced); got != want {
		t.Errorf("Expected path %q, got %q", want, got)
	}

	if out, err := testutil.RunGit(t, dir, "config", "gitflow.worktreePath", "../wt/{{ topicType }}/{{branch}}"); err != nil {
		t.Fatalf("Failed to set gitflow.worktreePath: %v\nOutput: %s", err, out)
	}
	spaced, err := testutil.RunGitFlow(t, dir, "worktree", "path", "feature/user-auth")
	if err != nil {
		t.Fatalf("worktree path failed: %v\nOutput: %s", err, spaced)
	}
	if spaced != unspaced {
		t.Errorf("Expected spaced and unspaced templates to expand identically, got %q vs %q", spaced, unspaced)
	}
}

// TestWorktreePathHasNoSideEffects covers scenario 20: 'worktree path' prints
// only, and creates nothing.
// Steps:
// 1. Sets up a repository with a feature/x branch and no worktree
// 2. Runs 'git flow worktree path feature/x'
// 3. Verifies the printed path and its parent do not exist
// 4. Verifies git worktree list shows only the main worktree and no marker was written
func TestWorktreePathHasNoSideEffects(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer os.RemoveAll(worktreeRootFor(dir))
	createFreeBranch(t, dir, "feature/x")

	output, err := testutil.RunGitFlow(t, dir, "worktree", "path", "feature/x")
	if err != nil {
		t.Fatalf("worktree path failed: %v\nOutput: %s", err, output)
	}
	printed := strings.TrimSpace(output)

	if _, err := os.Stat(printed); !os.IsNotExist(err) {
		t.Errorf("Expected printed path %q not to exist", printed)
	}
	if _, err := os.Stat(filepath.Dir(printed)); !os.IsNotExist(err) {
		t.Errorf("Expected parent of printed path %q not to exist", filepath.Dir(printed))
	}
	if rows := strings.Count(strings.TrimSpace(gitWorktreeList(t, dir)), "\n"); rows != 0 {
		t.Errorf("Expected only the main worktree, got: %s", gitWorktreeList(t, dir))
	}
	if testutil.GitConfigExists(t, dir, "gitflow.worktree.feature/x.managed") {
		t.Error("Expected no managed marker after 'worktree path'")
	}
}

// TestWorktreePathEmptyTemplateFallsBackToDefault verifies an empty
// gitflow.worktreePath falls back to the default template rather than resolving
// to the main worktree root.
// Steps:
// 1. Sets gitflow.worktreePath to the empty string
// 2. Runs 'git flow worktree path feature/x'
// 3. Verifies the output equals the default-template result
func TestWorktreePathEmptyTemplateFallsBackToDefault(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	createFreeBranch(t, dir, "feature/x")

	if out, err := testutil.RunGit(t, dir, "config", "gitflow.worktreePath", ""); err != nil {
		t.Fatalf("Failed to set gitflow.worktreePath: %v\nOutput: %s", err, out)
	}

	output, err := testutil.RunGitFlow(t, dir, "worktree", "path", "feature/x")
	if err != nil {
		t.Fatalf("worktree path failed: %v\nOutput: %s", err, output)
	}

	want := computedWorktreePath(t, dir, "feature/x")
	if got := strings.TrimSpace(output); got != want {
		t.Errorf("Expected empty template to fall back to %q, got %q", want, got)
	}
}

// TestWorktreePathNonTopicBranchCollapsesSeparator verifies that an empty
// {{ topicType }} expansion collapses the separator it leaves behind.
// Steps:
// 1. Sets gitflow.worktreePath to '../wt/{{ topicType }}/{{ branch }}'
// 2. Runs 'git flow worktree path main' (main is not a topic branch)
// 3. Verifies the result is <repoParent>/wt/main with no doubled separator
func TestWorktreePathNonTopicBranchCollapsesSeparator(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	if out, err := testutil.RunGit(t, dir, "config", "gitflow.worktreePath", "../wt/{{ topicType }}/{{ branch }}"); err != nil {
		t.Fatalf("Failed to set gitflow.worktreePath: %v\nOutput: %s", err, out)
	}

	output, err := testutil.RunGitFlow(t, dir, "worktree", "path", "main")
	if err != nil {
		t.Fatalf("worktree path failed: %v\nOutput: %s", err, output)
	}

	want := filepath.Join(filepath.Dir(testutil.EvalPath(t, dir)), "wt", "main")
	if got := strings.TrimSpace(output); got != want {
		t.Errorf("Expected %q, got %q", want, got)
	}
}

// ---------------------------------------------------------------------------
// Group A — the initialization gate, on every subcommand
// ---------------------------------------------------------------------------

// TestWorktreePathRequiresInitialization verifies 'worktree path' refuses to run
// in a repository where git-flow was never initialized.
// Steps:
// 1. Sets up a plain git repository without running 'git flow init'
// 2. Runs 'git flow worktree path main'
// 3. Verifies exit code 1 and a not-initialized message
func TestWorktreePathRequiresInitialization(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	output, err := testutil.RunGitFlow(t, dir, "worktree", "path", "main")
	if got := worktreeExitCode(err); got != 1 {
		t.Fatalf("Expected exit code 1, got %d\nOutput: %s", got, output)
	}
	if !strings.Contains(output, "not initialized") {
		t.Errorf("Expected a not-initialized message, got: %s", output)
	}
}

// TestWorktreeAddRequiresInitialization verifies 'worktree add' refuses to run in
// an uninitialized repository and mutates nothing.
// Steps:
// 1. Sets up a plain git repository with a feature/x branch, no 'git flow init'
// 2. Runs 'git flow worktree add feature/x'
// 3. Verifies exit code 1
// 4. Verifies no worktree was created and no gitflow.worktree.* key was written
func TestWorktreeAddRequiresInitialization(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer os.RemoveAll(worktreeRootFor(dir))
	createFreeBranch(t, dir, "feature/x")

	output, err := testutil.RunGitFlow(t, dir, "worktree", "add", "feature/x")
	if got := worktreeExitCode(err); got != 1 {
		t.Fatalf("Expected exit code 1, got %d\nOutput: %s", got, output)
	}
	if _, err := os.Stat(worktreeRootFor(dir)); !os.IsNotExist(err) {
		t.Errorf("Expected no worktree root at %q", worktreeRootFor(dir))
	}
	if testutil.GitConfigHasPrefix(t, dir, "gitflow.worktree.") {
		t.Error("Expected no gitflow.worktree.* key to be written")
	}
}

// TestWorktreeRemoveRequiresInitialization verifies 'worktree remove' refuses to
// run in an uninitialized repository.
// Steps:
// 1. Sets up a plain git repository with a feature/x branch, no 'git flow init'
// 2. Runs 'git flow worktree remove feature/x'
// 3. Verifies exit code 1 and a not-initialized message
func TestWorktreeRemoveRequiresInitialization(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	createFreeBranch(t, dir, "feature/x")

	output, err := testutil.RunGitFlow(t, dir, "worktree", "remove", "feature/x")
	if got := worktreeExitCode(err); got != 1 {
		t.Fatalf("Expected exit code 1, got %d\nOutput: %s", got, output)
	}
	if !strings.Contains(output, "not initialized") {
		t.Errorf("Expected a not-initialized message, got: %s", output)
	}
}

// TestWorktreeListRequiresInitialization verifies 'worktree list' refuses to run
// in an uninitialized repository.
// Steps:
// 1. Sets up a plain git repository without running 'git flow init'
// 2. Runs 'git flow worktree list'
// 3. Verifies exit code 1 and a not-initialized message
func TestWorktreeListRequiresInitialization(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	output, err := testutil.RunGitFlow(t, dir, "worktree", "list")
	if got := worktreeExitCode(err); got != 1 {
		t.Fatalf("Expected exit code 1, got %d\nOutput: %s", got, output)
	}
	if !strings.Contains(output, "not initialized") {
		t.Errorf("Expected a not-initialized message, got: %s", output)
	}
}

// TestWorktreePruneRequiresInitialization verifies 'worktree prune' refuses to
// run in an uninitialized repository and leaves stale admin entries alone.
// Steps:
// 1. Sets up a plain git repository, adds a worktree by hand and deletes its directory
// 2. Runs 'git flow worktree prune'
// 3. Verifies exit code 1
// 4. Verifies the stale admin entry is still listed by git worktree list
func TestWorktreePruneRequiresInitialization(t *testing.T) {
	t.Parallel()
	dir := testutil.SetupTestRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	stale := filepath.Join(t.TempDir(), "stale-worktree")
	if out, err := testutil.RunGit(t, dir, "worktree", "add", stale, "-b", "stale-branch"); err != nil {
		t.Fatalf("Failed to create worktree: %v\nOutput: %s", err, out)
	}
	if err := os.RemoveAll(stale); err != nil {
		t.Fatalf("Failed to delete worktree directory: %v", err)
	}

	output, err := testutil.RunGitFlow(t, dir, "worktree", "prune")
	if got := worktreeExitCode(err); got != 1 {
		t.Fatalf("Expected exit code 1, got %d\nOutput: %s", got, output)
	}
	if !strings.Contains(gitWorktreeList(t, dir), "stale-branch") {
		t.Error("Expected the stale admin entry to survive a refused prune")
	}
}

// ---------------------------------------------------------------------------
// Group C — worktree add
// ---------------------------------------------------------------------------

// TestWorktreeAddCreatesWorktree covers scenario 4: add creates the worktree at
// the computed path.
// Steps:
// 1. Sets up a repository and creates feature/x without checking it out
// 2. Runs 'git flow worktree add feature/x'
// 3. Verifies the computed (nested) path exists and contains a .git file
// 4. Verifies git worktree list reports the path with [feature/x]
func TestWorktreeAddCreatesWorktree(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer os.RemoveAll(worktreeRootFor(dir))
	createFreeBranch(t, dir, "feature/x")

	output, err := testutil.RunGitFlow(t, dir, "worktree", "add", "feature/x")
	if err != nil {
		t.Fatalf("worktree add failed: %v\nOutput: %s", err, output)
	}

	wtPath := computedWorktreePath(t, dir, "feature/x")
	info, err := os.Stat(wtPath)
	if err != nil || !info.IsDir() {
		t.Fatalf("Expected worktree directory at %q: %v", wtPath, err)
	}
	if _, err := os.Stat(filepath.Join(wtPath, ".git")); err != nil {
		t.Errorf("Expected a .git file inside the worktree: %v", err)
	}

	list := gitWorktreeList(t, dir)
	if !strings.Contains(list, wtPath) || !strings.Contains(list, "[feature/x]") {
		t.Errorf("Expected git worktree list to show %q with [feature/x], got: %s", wtPath, list)
	}
}

// TestWorktreeAddWritesManagedMarker covers scenario 5: add records provenance.
// Steps:
// 1. Sets up a repository with a free feature/x branch
// 2. Runs 'git flow worktree add feature/x'
// 3. Verifies gitflow.worktree.feature/x.managed is true in local config
func TestWorktreeAddWritesManagedMarker(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer os.RemoveAll(worktreeRootFor(dir))
	createFreeBranch(t, dir, "feature/x")

	addWorktree(t, dir, "feature/x")

	if v := testutil.GitConfigValue(t, dir, "gitflow.worktree.feature/x.managed"); v != "true" {
		t.Errorf("Expected gitflow.worktree.feature/x.managed=true, got %q", v)
	}
}

// TestWorktreeAddWithCustomPath covers scenario 6: --path overrides the template.
// Steps:
// 1. Sets up a repository with a free feature/x branch
// 2. Runs 'git flow worktree add feature/x --path ./wt-custom-add'
// 3. Verifies the worktree exists at the custom path and not at the computed path
// 4. Verifies the provenance marker is written the same way
func TestWorktreeAddWithCustomPath(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer os.RemoveAll(worktreeRootFor(dir))
	createFreeBranch(t, dir, "feature/x")

	output, err := testutil.RunGitFlow(t, dir, "worktree", "add", "feature/x", "--path", "./wt-custom-add")
	if err != nil {
		t.Fatalf("worktree add --path failed: %v\nOutput: %s", err, output)
	}

	custom := filepath.Join(testutil.EvalPath(t, dir), "wt-custom-add")
	if info, err := os.Stat(custom); err != nil || !info.IsDir() {
		t.Fatalf("Expected worktree at %q: %v", custom, err)
	}
	if _, err := os.Stat(computedWorktreePath(t, dir, "feature/x")); !os.IsNotExist(err) {
		t.Error("Expected the computed template path not to be used when --path is given")
	}
	if v := testutil.GitConfigValue(t, dir, "gitflow.worktree.feature/x.managed"); v != "true" {
		t.Errorf("Expected gitflow.worktree.feature/x.managed=true, got %q", v)
	}
}

// TestWorktreeAddNonExistentBranch covers scenario 7: add refuses an unknown branch.
// Steps:
// 1. Sets up a repository with git-flow defaults
// 2. Runs 'git flow worktree add feature/nope'
// 3. Verifies exit code 5 and an error naming the branch
// 4. Verifies nothing was created and no marker was written
func TestWorktreeAddNonExistentBranch(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer os.RemoveAll(worktreeRootFor(dir))

	output, err := testutil.RunGitFlow(t, dir, "worktree", "add", "feature/nope")
	if got := worktreeExitCode(err); got != 5 {
		t.Fatalf("Expected exit code 5, got %d\nOutput: %s", got, output)
	}
	if !strings.Contains(output, "feature/nope") {
		t.Errorf("Expected the error to name the branch, got: %s", output)
	}
	if _, err := os.Stat(computedWorktreePath(t, dir, "feature/nope")); !os.IsNotExist(err) {
		t.Error("Expected no worktree directory to be created")
	}
	if testutil.GitConfigHasPrefix(t, dir, "gitflow.worktree.") {
		t.Error("Expected no gitflow.worktree.* key to be written")
	}
}

// TestWorktreeAddWhenWorktreeAlreadyExists covers scenario 8: add refuses when a
// worktree for the branch already exists.
// Steps:
// 1. Sets up a repository and adds a worktree for feature/x
// 2. Runs 'git flow worktree add feature/x' a second time
// 3. Verifies exit code 4 and a message naming the branch and the existing path
// 4. Verifies the first worktree is untouched and no worktree was added
func TestWorktreeAddWhenWorktreeAlreadyExists(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer os.RemoveAll(worktreeRootFor(dir))
	createFreeBranch(t, dir, "feature/x")

	wtPath := addWorktree(t, dir, "feature/x")
	before := gitWorktreeList(t, dir)

	output, err := testutil.RunGitFlow(t, dir, "worktree", "add", "feature/x")
	if got := worktreeExitCode(err); got != 4 {
		t.Fatalf("Expected exit code 4, got %d\nOutput: %s", got, output)
	}
	if !strings.Contains(output, "feature/x") || !strings.Contains(output, wtPath) {
		t.Errorf("Expected the error to name the branch and existing path, got: %s", output)
	}
	if info, err := os.Stat(wtPath); err != nil || !info.IsDir() {
		t.Errorf("Expected the existing worktree to be untouched: %v", err)
	}
	if after := gitWorktreeList(t, dir); after != before {
		t.Errorf("Expected the worktree list to be unchanged.\nBefore: %s\nAfter: %s", before, after)
	}
}

// TestWorktreeAddTargetPathOccupied covers scenario 9: add refuses when the
// target path is an unrelated directory.
// Steps:
// 1. Creates the computed path as a plain directory containing a file
// 2. Runs 'git flow worktree add feature/x'
// 3. Verifies exit code 6 and a message naming the occupied path
// 4. Verifies the directory contents are unchanged, no worktree registered, no marker
func TestWorktreeAddTargetPathOccupied(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer os.RemoveAll(worktreeRootFor(dir))
	createFreeBranch(t, dir, "feature/x")

	wtPath := computedWorktreePath(t, dir, "feature/x")
	if err := os.MkdirAll(wtPath, 0755); err != nil {
		t.Fatalf("Failed to create occupying directory: %v", err)
	}
	occupant := filepath.Join(wtPath, "occupant.txt")
	if err := os.WriteFile(occupant, []byte("mine"), 0644); err != nil {
		t.Fatalf("Failed to write occupying file: %v", err)
	}

	output, err := testutil.RunGitFlow(t, dir, "worktree", "add", "feature/x")
	if got := worktreeExitCode(err); got != 6 {
		t.Fatalf("Expected exit code 6, got %d\nOutput: %s", got, output)
	}
	if !strings.Contains(output, wtPath) {
		t.Errorf("Expected the error to name the occupied path, got: %s", output)
	}
	content, err := os.ReadFile(occupant)
	if err != nil || string(content) != "mine" {
		t.Errorf("Expected the occupying file to be untouched, got %q (%v)", string(content), err)
	}
	if strings.Contains(gitWorktreeList(t, dir), "feature/x") {
		t.Error("Expected no worktree to be registered for feature/x")
	}
	if testutil.GitConfigHasPrefix(t, dir, "gitflow.worktree.") {
		t.Error("Expected no gitflow.worktree.* key to be written")
	}
}

// TestWorktreeAddIntoExistingEmptyDirectorySucceeds pins the boundary of scenario
// 9 recorded in decisions.md §12: an existing EMPTY directory is not "occupied"
// and is accepted as the target, matching plain 'git worktree add'. Without this
// test a regression to refusing empty directories would pass the whole suite,
// since the occupancy test above uses a non-empty directory.
// Steps:
// 1. Pre-creates the computed path as an empty directory
// 2. Runs 'git flow worktree add feature/x'
// 3. Verifies exit code 0 and that the worktree was registered at that path
// 4. Verifies the provenance marker was written like any other add
func TestWorktreeAddIntoExistingEmptyDirectorySucceeds(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer os.RemoveAll(worktreeRootFor(dir))
	createFreeBranch(t, dir, "feature/x")

	wtPath := computedWorktreePath(t, dir, "feature/x")
	if err := os.MkdirAll(wtPath, 0755); err != nil {
		t.Fatalf("Failed to create the empty target directory: %v", err)
	}

	output, err := testutil.RunGitFlow(t, dir, "worktree", "add", "feature/x")
	if err != nil {
		t.Fatalf("Expected add into an empty directory to succeed: %v\nOutput: %s", err, output)
	}
	if !strings.Contains(gitWorktreeList(t, dir), wtPath) {
		t.Errorf("Expected a worktree registered at %s, got: %s", wtPath, gitWorktreeList(t, dir))
	}
	if _, err := os.Stat(filepath.Join(wtPath, ".git")); err != nil {
		t.Errorf("Expected the worktree to be checked out into the existing directory: %v", err)
	}
	if v := testutil.GitConfigValue(t, dir, "gitflow.worktree.feature/x.managed"); v != "true" {
		t.Errorf("Expected gitflow.worktree.feature/x.managed=true, got %q", v)
	}
}

// TestWorktreeAddBranchCheckedOutInMainWorktree covers scenario 10: add refuses a
// branch that is checked out in the main worktree.
// Steps:
// 1. Runs 'git flow feature start x', leaving feature/x checked out in the main worktree
// 2. Runs 'git flow worktree add feature/x'
// 3. Verifies exit code 6 and a message naming the main worktree
// 4. Verifies no directory was left behind and no marker was written
func TestWorktreeAddBranchCheckedOutInMainWorktree(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer os.RemoveAll(worktreeRootFor(dir))

	if out, err := testutil.RunGitFlow(t, dir, "feature", "start", "x"); err != nil {
		t.Fatalf("feature start failed: %v\nOutput: %s", err, out)
	}

	output, err := testutil.RunGitFlow(t, dir, "worktree", "add", "feature/x")
	if got := worktreeExitCode(err); got != 6 {
		t.Fatalf("Expected exit code 6, got %d\nOutput: %s", got, output)
	}
	if !strings.Contains(output, testutil.EvalPath(t, dir)) {
		t.Errorf("Expected the error to name the main worktree, got: %s", output)
	}
	if _, err := os.Stat(computedWorktreePath(t, dir, "feature/x")); !os.IsNotExist(err) {
		t.Error("Expected no partial worktree directory to be left behind")
	}
	if testutil.GitConfigHasPrefix(t, dir, "gitflow.worktree.") {
		t.Error("Expected no gitflow.worktree.* key to be written")
	}
}

// TestWorktreeAddRelativePathResolvesAgainstInvocationDir verifies a relative
// --path resolves against the invocation directory, not the main worktree root.
// Steps:
// 1. Creates subdirectory <repo>/sub and a free feature/x branch
// 2. Runs 'git flow worktree add feature/x --path ../custom' from <repo>/sub
// 3. Verifies the worktree lands at <repo>/custom
// 4. Verifies it did NOT land at <repoParent>/custom
func TestWorktreeAddRelativePathResolvesAgainstInvocationDir(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	createFreeBranch(t, dir, "feature/x")

	sub := filepath.Join(dir, "sub")
	if err := os.MkdirAll(sub, 0755); err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}

	output, err := testutil.RunGitFlow(t, sub, "worktree", "add", "feature/x", "--path", "../custom")
	if err != nil {
		t.Fatalf("worktree add --path failed: %v\nOutput: %s", err, output)
	}

	repoRoot := testutil.EvalPath(t, dir)
	inRepo := filepath.Join(repoRoot, "custom")
	outsideRepo := filepath.Join(filepath.Dir(repoRoot), "custom")
	defer os.RemoveAll(outsideRepo)

	if info, err := os.Stat(inRepo); err != nil || !info.IsDir() {
		t.Errorf("Expected worktree at the invocation-relative path %q: %v", inRepo, err)
	}
	if _, err := os.Stat(outsideRepo); !os.IsNotExist(err) {
		t.Errorf("Expected no worktree at the main-worktree-relative path %q", outsideRepo)
	}
}

// ---------------------------------------------------------------------------
// Group D — worktree remove
// ---------------------------------------------------------------------------

// TestWorktreeRemoveCleanWorktree covers scenario 11: remove drops a clean worktree.
// Steps:
// 1. Adds a worktree for feature/x through git-flow (marker present)
// 2. Runs 'git flow worktree remove feature/x'
// 3. Verifies the directory and admin entry are gone and the marker is cleared
// 4. Verifies the branch feature/x still exists
func TestWorktreeRemoveCleanWorktree(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer os.RemoveAll(worktreeRootFor(dir))
	createFreeBranch(t, dir, "feature/x")
	wtPath := addWorktree(t, dir, "feature/x")

	output, err := testutil.RunGitFlow(t, dir, "worktree", "remove", "feature/x")
	if err != nil {
		t.Fatalf("worktree remove failed: %v\nOutput: %s", err, output)
	}

	if _, err := os.Stat(wtPath); !os.IsNotExist(err) {
		t.Errorf("Expected worktree directory %q to be gone", wtPath)
	}
	if strings.Contains(gitWorktreeList(t, dir), wtPath) {
		t.Error("Expected the admin entry to be gone")
	}
	if testutil.GitConfigExists(t, dir, "gitflow.worktree.feature/x.managed") {
		t.Error("Expected the managed marker to be cleared")
	}
	if !testutil.BranchExists(t, dir, "feature/x") {
		t.Error("Expected branch feature/x to survive worktree removal")
	}
}

// TestWorktreeRemoveRefusesUncommittedChanges covers scenario 12a: remove refuses
// a worktree with modified tracked files.
// Steps:
// 1. Adds a worktree for feature/x and modifies a tracked file inside it
// 2. Runs 'git flow worktree remove feature/x'
// 3. Verifies exit code 6 and a message mentioning --force
// 4. Verifies the directory, the modification, and the marker all survive
func TestWorktreeRemoveRefusesUncommittedChanges(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer os.RemoveAll(worktreeRootFor(dir))
	createFreeBranch(t, dir, "feature/x")
	wtPath := addWorktree(t, dir, "feature/x")

	if err := os.WriteFile(filepath.Join(wtPath, "README.md"), []byte("modified"), 0644); err != nil {
		t.Fatalf("Failed to modify tracked file: %v", err)
	}

	output, err := testutil.RunGitFlow(t, dir, "worktree", "remove", "feature/x")
	if got := worktreeExitCode(err); got != 6 {
		t.Fatalf("Expected exit code 6, got %d\nOutput: %s", got, output)
	}
	if !strings.Contains(output, "--force") {
		t.Errorf("Expected the refusal to mention --force, got: %s", output)
	}
	content, err := os.ReadFile(filepath.Join(wtPath, "README.md"))
	if err != nil || string(content) != "modified" {
		t.Errorf("Expected the modification to survive, got %q (%v)", string(content), err)
	}
	if v := testutil.GitConfigValue(t, dir, "gitflow.worktree.feature/x.managed"); v != "true" {
		t.Errorf("Expected the marker to survive a refusal, got %q", v)
	}
}

// TestWorktreeRemoveForceWithUncommittedChanges covers scenario 12b: --force
// removes a worktree with modified tracked files.
// Steps:
// 1. Adds a worktree for feature/x and modifies a tracked file inside it
// 2. Runs 'git flow worktree remove feature/x --force'
// 3. Verifies exit code 0 and that the directory is gone
// 4. Verifies the marker is cleared and the branch is retained
func TestWorktreeRemoveForceWithUncommittedChanges(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer os.RemoveAll(worktreeRootFor(dir))
	createFreeBranch(t, dir, "feature/x")
	wtPath := addWorktree(t, dir, "feature/x")

	if err := os.WriteFile(filepath.Join(wtPath, "README.md"), []byte("modified"), 0644); err != nil {
		t.Fatalf("Failed to modify tracked file: %v", err)
	}

	output, err := testutil.RunGitFlow(t, dir, "worktree", "remove", "feature/x", "--force")
	if err != nil {
		t.Fatalf("worktree remove --force failed: %v\nOutput: %s", err, output)
	}
	if _, err := os.Stat(wtPath); !os.IsNotExist(err) {
		t.Errorf("Expected worktree directory %q to be gone", wtPath)
	}
	if testutil.GitConfigExists(t, dir, "gitflow.worktree.feature/x.managed") {
		t.Error("Expected the managed marker to be cleared")
	}
	if !testutil.BranchExists(t, dir, "feature/x") {
		t.Error("Expected branch feature/x to survive worktree removal")
	}
}

// TestWorktreeRemoveRefusesUntrackedFiles covers scenario 13a: the dirty check
// covers untracked files, not just tracked modifications.
// Steps:
// 1. Adds a worktree for feature/x and writes a new untracked file inside it
// 2. Runs 'git flow worktree remove feature/x'
// 3. Verifies exit code 6 and a refusal message
// 4. Verifies the worktree and the untracked file survive
func TestWorktreeRemoveRefusesUntrackedFiles(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer os.RemoveAll(worktreeRootFor(dir))
	createFreeBranch(t, dir, "feature/x")
	wtPath := addWorktree(t, dir, "feature/x")

	untracked := filepath.Join(wtPath, "scratch.txt")
	if err := os.WriteFile(untracked, []byte("scratch"), 0644); err != nil {
		t.Fatalf("Failed to write untracked file: %v", err)
	}

	output, err := testutil.RunGitFlow(t, dir, "worktree", "remove", "feature/x")
	if got := worktreeExitCode(err); got != 6 {
		t.Fatalf("Expected exit code 6, got %d\nOutput: %s", got, output)
	}
	if !strings.Contains(output, "--force") {
		t.Errorf("Expected the refusal to mention --force, got: %s", output)
	}
	if _, err := os.Stat(untracked); err != nil {
		t.Errorf("Expected the untracked file to survive: %v", err)
	}
}

// TestWorktreeRemoveForceWithUntrackedFiles covers scenario 13b: --force removes
// a worktree that only holds untracked files.
// Steps:
// 1. Adds a worktree for feature/x and writes a new untracked file inside it
// 2. Runs 'git flow worktree remove feature/x --force'
// 3. Verifies exit code 0 and that the directory is gone
// 4. Verifies the marker is cleared
func TestWorktreeRemoveForceWithUntrackedFiles(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer os.RemoveAll(worktreeRootFor(dir))
	createFreeBranch(t, dir, "feature/x")
	wtPath := addWorktree(t, dir, "feature/x")

	if err := os.WriteFile(filepath.Join(wtPath, "scratch.txt"), []byte("scratch"), 0644); err != nil {
		t.Fatalf("Failed to write untracked file: %v", err)
	}

	output, err := testutil.RunGitFlow(t, dir, "worktree", "remove", "feature/x", "--force")
	if err != nil {
		t.Fatalf("worktree remove --force failed: %v\nOutput: %s", err, output)
	}
	if _, err := os.Stat(wtPath); !os.IsNotExist(err) {
		t.Errorf("Expected worktree directory %q to be gone", wtPath)
	}
	if testutil.GitConfigExists(t, dir, "gitflow.worktree.feature/x.managed") {
		t.Error("Expected the managed marker to be cleared")
	}
}

// TestWorktreeRemoveUnmanagedWorktree covers scenario 14: remove ignores
// provenance for an explicitly named target.
// Steps:
// 1. Creates a worktree for feature/x with plain 'git worktree add' (no marker)
// 2. Runs 'git flow worktree remove feature/x'
// 3. Verifies exit code 0 and that the worktree is gone
// 4. Verifies the branch is retained
func TestWorktreeRemoveUnmanagedWorktree(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	createFreeBranch(t, dir, "feature/x")

	wtPath := filepath.Join(t.TempDir(), "handmade")
	if out, err := testutil.RunGit(t, dir, "worktree", "add", wtPath, "feature/x"); err != nil {
		t.Fatalf("git worktree add failed: %v\nOutput: %s", err, out)
	}

	output, err := testutil.RunGitFlow(t, dir, "worktree", "remove", "feature/x")
	if err != nil {
		t.Fatalf("worktree remove failed: %v\nOutput: %s", err, output)
	}
	if _, err := os.Stat(wtPath); !os.IsNotExist(err) {
		t.Errorf("Expected the hand-made worktree %q to be removed", wtPath)
	}
	if !testutil.BranchExists(t, dir, "feature/x") {
		t.Error("Expected branch feature/x to survive worktree removal")
	}
}

// TestWorktreeRemoveWithoutWorktree covers scenario 15: remove errors when the
// branch has no worktree.
// Steps:
// 1. Creates branch feature/x with no worktree anywhere
// 2. Runs 'git flow worktree remove feature/x'
// 3. Verifies exit code 5 and a "no worktree" message
// 4. Verifies the branch is untouched
func TestWorktreeRemoveWithoutWorktree(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	createFreeBranch(t, dir, "feature/x")

	output, err := testutil.RunGitFlow(t, dir, "worktree", "remove", "feature/x")
	if got := worktreeExitCode(err); got != 5 {
		t.Fatalf("Expected exit code 5, got %d\nOutput: %s", got, output)
	}
	if !strings.Contains(output, "no worktree") {
		t.Errorf("Expected a 'no worktree' message, got: %s", output)
	}
	if !testutil.BranchExists(t, dir, "feature/x") {
		t.Error("Expected branch feature/x to be untouched")
	}
}

// TestWorktreeRemoveRefusesMainWorktree covers scenario 16: the main worktree is
// never removed, and that refusal beats the "nothing to remove" path.
// Steps:
//  1. Sets up a repository and checks out main in the main worktree
//     ('git flow init' leaves develop checked out, so this is explicit)
//  2. Runs 'git flow worktree remove main'
//  3. Verifies exit code 6 and a message about the main worktree
//  4. Verifies the repository directory still exists and is a worktree
func TestWorktreeRemoveRefusesMainWorktree(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	if out, err := testutil.RunGit(t, dir, "checkout", "main"); err != nil {
		t.Fatalf("Failed to check out main: %v\nOutput: %s", err, out)
	}

	output, err := testutil.RunGitFlow(t, dir, "worktree", "remove", "main")
	if got := worktreeExitCode(err); got != 6 {
		t.Fatalf("Expected exit code 6, got %d\nOutput: %s", got, output)
	}
	if !strings.Contains(output, "main worktree") {
		t.Errorf("Expected a main-worktree refusal, got: %s", output)
	}
	if info, err := os.Stat(dir); err != nil || !info.IsDir() {
		t.Fatalf("Expected the repository directory to survive: %v", err)
	}
	if !strings.Contains(gitWorktreeList(t, dir), testutil.EvalPath(t, dir)) {
		t.Error("Expected the main worktree to still be listed")
	}
}

// TestWorktreeRemoveStaleEntry verifies remove succeeds against a stale admin
// entry whose directory was deleted by hand — a missing directory counts as clean.
// Steps:
// 1. Adds a worktree for feature/x, then deletes its directory with os.RemoveAll
// 2. Runs 'git flow worktree remove feature/x' without --force
// 3. Verifies exit code 0 and that the admin entry is gone
// 4. Verifies the marker is cleared and the branch is retained
func TestWorktreeRemoveStaleEntry(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer os.RemoveAll(worktreeRootFor(dir))
	createFreeBranch(t, dir, "feature/x")
	wtPath := addWorktree(t, dir, "feature/x")

	if err := os.RemoveAll(wtPath); err != nil {
		t.Fatalf("Failed to delete worktree directory: %v", err)
	}

	output, err := testutil.RunGitFlow(t, dir, "worktree", "remove", "feature/x")
	if err != nil {
		t.Fatalf("worktree remove on a stale entry failed: %v\nOutput: %s", err, output)
	}
	if strings.Contains(gitWorktreeList(t, dir), wtPath) {
		t.Error("Expected the stale admin entry to be gone")
	}
	if testutil.GitConfigExists(t, dir, "gitflow.worktree.feature/x.managed") {
		t.Error("Expected the managed marker to be cleared")
	}
	if !testutil.BranchExists(t, dir, "feature/x") {
		t.Error("Expected branch feature/x to survive")
	}
}

// TestWorktreeRemoveNoCDFlagLeavesFileEmpty verifies --no-cd suppresses the
// navigation write even when removing the worktree the command runs in.
// Steps:
// 1. Adds a worktree for feature/x and creates an empty GIT_FLOW_CD_FILE
// 2. Runs 'git flow worktree remove feature/x --no-cd' from inside that worktree
// 3. Verifies exit code 0 and that the worktree is removed
// 4. Verifies the CD file is still zero-length
func TestWorktreeRemoveNoCDFlagLeavesFileEmpty(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer os.RemoveAll(worktreeRootFor(dir))
	createFreeBranch(t, dir, "feature/x")
	wtPath := addWorktree(t, dir, "feature/x")
	cdFile := cdFilePath(t)

	output, err := testutil.RunGitFlowWithEnv(t, wtPath, cdEnv(cdFile), "worktree", "remove", "feature/x", "--no-cd")
	if err != nil {
		t.Fatalf("worktree remove --no-cd failed: %v\nOutput: %s", err, output)
	}
	if _, err := os.Stat(wtPath); !os.IsNotExist(err) {
		t.Errorf("Expected worktree directory %q to be gone", wtPath)
	}
	assertCDFileEmpty(t, cdFile)
}

// ---------------------------------------------------------------------------
// Group E — worktree list
// ---------------------------------------------------------------------------

// TestWorktreeListShowsLinkedWorktrees covers scenario 17: list shows every
// linked worktree and excludes the main one.
// Steps:
// 1. Adds worktrees for feature/a and feature/b through git-flow
// 2. Runs 'git flow worktree list'
// 3. Verifies exactly two rows, each carrying its branch and absolute path
// 4. Verifies no row is the main worktree (neither its path nor its branch)
func TestWorktreeListShowsLinkedWorktrees(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer os.RemoveAll(worktreeRootFor(dir))
	createFreeBranch(t, dir, "feature/a")
	createFreeBranch(t, dir, "feature/b")
	pathA := addWorktree(t, dir, "feature/a")
	pathB := addWorktree(t, dir, "feature/b")

	output, err := testutil.RunGitFlow(t, dir, "worktree", "list")
	if err != nil {
		t.Fatalf("worktree list failed: %v\nOutput: %s", err, output)
	}

	rows := worktreeRows(output)
	if len(rows) != 2 {
		t.Fatalf("Expected exactly two rows, got %d: %s", len(rows), output)
	}

	mainRoot := testutil.EvalPath(t, dir)
	seen := map[string]string{}
	for _, row := range rows {
		fields := strings.Fields(row)
		if len(fields) < 2 {
			t.Fatalf("Expected a branch and a path in row %q", row)
		}
		branch, path := fields[0], fields[1]
		// Compare the path FIELD for equality: the default template nests
		// worktrees under a sibling directory whose name has the main worktree
		// root as a literal prefix, so a substring check is unsatisfiable.
		if path == mainRoot {
			t.Errorf("Expected the main worktree to be excluded, got row %q", row)
		}
		if branch == "main" {
			t.Errorf("Expected the main worktree's branch to be excluded, got row %q", row)
		}
		seen[branch] = path
	}
	if seen["feature/a"] != pathA {
		t.Errorf("Expected feature/a at %q, got %q", pathA, seen["feature/a"])
	}
	if seen["feature/b"] != pathB {
		t.Errorf("Expected feature/b at %q, got %q", pathB, seen["feature/b"])
	}
}

// TestWorktreeListTagsUnmanagedWorktree covers scenario 18: provenance comes from
// the marker only, never from the path shape.
// Steps:
// 1. Adds feature/a through git-flow (marker written)
// 2. Adds feature/b by hand at the path the template WOULD compute (no marker)
// 3. Runs 'git flow worktree list'
// 4. Verifies the feature/b row is tagged (unmanaged) and the feature/a row is not
func TestWorktreeListTagsUnmanagedWorktree(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer os.RemoveAll(worktreeRootFor(dir))
	createFreeBranch(t, dir, "feature/a")
	createFreeBranch(t, dir, "feature/b")
	addWorktree(t, dir, "feature/a")

	// Deliberately template-shaped, so only the missing marker can make it read
	// as unmanaged.
	handmade := computedWorktreePath(t, dir, "feature/b")
	if out, err := testutil.RunGit(t, dir, "worktree", "add", handmade, "feature/b"); err != nil {
		t.Fatalf("git worktree add failed: %v\nOutput: %s", err, out)
	}

	output, err := testutil.RunGitFlow(t, dir, "worktree", "list")
	if err != nil {
		t.Fatalf("worktree list failed: %v\nOutput: %s", err, output)
	}

	// Assert the row count before inspecting rows: without it a regression that
	// dropped the unmanaged row entirely would leave the loop with nothing to
	// contradict and the test would pass having asserted nothing.
	rows := worktreeRows(output)
	if len(rows) != 2 {
		t.Fatalf("Expected exactly two rows, got %d: %s", len(rows), output)
	}

	for _, row := range rows {
		fields := strings.Fields(row)
		switch fields[0] {
		case "feature/a":
			if strings.Contains(row, "(unmanaged)") {
				t.Errorf("Expected the git-flow-created worktree to carry no tag, got %q", row)
			}
		case "feature/b":
			if !strings.HasSuffix(row, "(unmanaged)") {
				t.Errorf("Expected the hand-made worktree to be tagged (unmanaged), got %q", row)
			}
		default:
			t.Errorf("Unexpected row %q", row)
		}
	}
}

// TestWorktreeListIgnoresGlobalScopedMarker verifies the provenance marker is
// read from LOCAL config only, the scope it is written to and cleared in. A
// marker that exists only in global config must not make a hand-made worktree
// report as git-flow's: clearing only touches local scope, so such a marker could
// never be removed and the cleanup commands would delete the user's own worktree.
// Steps:
// 1. Isolates the global config to a temp file so the real global config is untouched
// 2. Creates a worktree for feature/b by hand, so no local marker exists
// 3. Writes gitflow.worktree.feature/b.managed=true in the GLOBAL scope only
// 4. Verifies no gitflow.worktree.* key exists in local config
// 5. Runs 'git flow worktree list' and verifies the row is still tagged (unmanaged)
func TestWorktreeListIgnoresGlobalScopedMarker(t *testing.T) {
	t.Parallel()
	// The override is passed through each subprocess env (not the test process
	// env) so it stays scoped to this test and is safe under parallel execution.
	env := []string{"GIT_CONFIG_GLOBAL=" + filepath.Join(t.TempDir(), "global-gitconfig")}

	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer os.RemoveAll(worktreeRootFor(dir))
	createFreeBranch(t, dir, "feature/b")

	handmade := computedWorktreePath(t, dir, "feature/b")
	if out, err := testutil.RunGit(t, dir, "worktree", "add", handmade, "feature/b"); err != nil {
		t.Fatalf("git worktree add failed: %v\nOutput: %s", err, out)
	}

	if out, err := testutil.RunGitWithEnv(t, dir, env, "config", "--global", "gitflow.worktree.feature/b.managed", "true"); err != nil {
		t.Fatalf("Failed to write the global marker: %v\nOutput: %s", err, out)
	}
	if testutil.GitConfigHasPrefix(t, dir, "gitflow.worktree.") {
		t.Fatal("Expected no gitflow.worktree.* key in local config")
	}

	output, err := testutil.RunGitFlowWithEnv(t, dir, env, "worktree", "list")
	if err != nil {
		t.Fatalf("worktree list failed: %v\nOutput: %s", err, output)
	}

	rows := worktreeRows(output)
	if len(rows) != 1 {
		t.Fatalf("Expected exactly one row, got %d: %s", len(rows), output)
	}
	if !strings.HasSuffix(rows[0], "(unmanaged)") {
		t.Errorf("Expected a global-only marker to be ignored and the row tagged (unmanaged), got %q", rows[0])
	}
}

// TestWorktreeListShowsDetachedWorktree verifies a detached worktree is still
// listed, with (detached) in the branch column.
// Steps:
// 1. Adds a worktree for feature/x and detaches its HEAD
// 2. Runs 'git flow worktree list'
// 3. Verifies the worktree is still listed with its path unchanged
// 4. Verifies its branch column reads (detached)
func TestWorktreeListShowsDetachedWorktree(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer os.RemoveAll(worktreeRootFor(dir))
	createFreeBranch(t, dir, "feature/x")
	wtPath := addWorktree(t, dir, "feature/x")

	if out, err := testutil.RunGit(t, wtPath, "checkout", "--detach"); err != nil {
		t.Fatalf("Failed to detach worktree HEAD: %v\nOutput: %s", err, out)
	}

	output, err := testutil.RunGitFlow(t, dir, "worktree", "list")
	if err != nil {
		t.Fatalf("worktree list failed: %v\nOutput: %s", err, output)
	}

	rows := worktreeRows(output)
	if len(rows) != 1 {
		t.Fatalf("Expected exactly one row, got %d: %s", len(rows), output)
	}
	fields := strings.Fields(rows[0])
	if fields[0] != "(detached)" {
		t.Errorf("Expected the branch column to read (detached), got %q", rows[0])
	}
	if fields[1] != wtPath {
		t.Errorf("Expected the path column to stay %q, got %q", wtPath, fields[1])
	}
}

// TestWorktreeListShowsStaleWorktreeUntilPrune verifies a stale entry is listed
// as-is until prune runs.
// Steps:
// 1. Adds a worktree for feature/x, then deletes its directory
// 2. Runs 'git flow worktree list' and verifies the entry is still shown
// 3. Runs 'git flow worktree prune'
// 4. Runs 'git flow worktree list' again and verifies the entry is gone
func TestWorktreeListShowsStaleWorktreeUntilPrune(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer os.RemoveAll(worktreeRootFor(dir))
	createFreeBranch(t, dir, "feature/x")
	wtPath := addWorktree(t, dir, "feature/x")

	if err := os.RemoveAll(wtPath); err != nil {
		t.Fatalf("Failed to delete worktree directory: %v", err)
	}

	before, err := testutil.RunGitFlow(t, dir, "worktree", "list")
	if err != nil {
		t.Fatalf("worktree list failed: %v\nOutput: %s", err, before)
	}
	if !strings.Contains(before, wtPath) || !strings.Contains(before, "feature/x") {
		t.Errorf("Expected the stale entry to still be listed, got: %s", before)
	}

	if out, err := testutil.RunGitFlow(t, dir, "worktree", "prune"); err != nil {
		t.Fatalf("worktree prune failed: %v\nOutput: %s", err, out)
	}

	after, err := testutil.RunGitFlow(t, dir, "worktree", "list")
	if err != nil {
		t.Fatalf("worktree list failed: %v\nOutput: %s", err, after)
	}
	if strings.Contains(after, wtPath) {
		t.Errorf("Expected the stale entry to be gone after prune, got: %s", after)
	}
}

// TestWorktreeListWithNoLinkedWorktrees verifies the empty listing message.
// Steps:
// 1. Sets up a repository with git-flow defaults and no linked worktrees
// 2. Runs 'git flow worktree list'
// 3. Verifies exit code 0
// 4. Verifies the single line 'No linked worktrees found'
func TestWorktreeListWithNoLinkedWorktrees(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)

	output, err := testutil.RunGitFlow(t, dir, "worktree", "list")
	if err != nil {
		t.Fatalf("worktree list failed: %v\nOutput: %s", err, output)
	}
	if strings.TrimSpace(output) != "No linked worktrees found" {
		t.Errorf("Expected 'No linked worktrees found', got: %s", output)
	}
}

// ---------------------------------------------------------------------------
// Group F — worktree prune
// ---------------------------------------------------------------------------

// TestWorktreePruneRemovesStaleEntryAndMarker covers scenario 19: prune drops the
// stale admin entry and its marker, and only its marker.
// Steps:
// 1. Adds worktrees for feature/x and feature/keep, then deletes feature/x's directory
// 2. Runs 'git flow worktree prune'
// 3. Verifies the stale entry and gitflow.worktree.feature/x.managed are gone
// 4. Verifies the live worktree's marker is still present
func TestWorktreePruneRemovesStaleEntryAndMarker(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer os.RemoveAll(worktreeRootFor(dir))
	createFreeBranch(t, dir, "feature/x")
	createFreeBranch(t, dir, "feature/keep")
	wtPath := addWorktree(t, dir, "feature/x")
	addWorktree(t, dir, "feature/keep")

	if err := os.RemoveAll(wtPath); err != nil {
		t.Fatalf("Failed to delete worktree directory: %v", err)
	}

	output, err := testutil.RunGitFlow(t, dir, "worktree", "prune")
	if err != nil {
		t.Fatalf("worktree prune failed: %v\nOutput: %s", err, output)
	}

	if strings.Contains(gitWorktreeList(t, dir), wtPath) {
		t.Error("Expected the stale admin entry to be pruned")
	}
	if testutil.GitConfigExists(t, dir, "gitflow.worktree.feature/x.managed") {
		t.Error("Expected the stale marker to be swept")
	}
	if v := testutil.GitConfigValue(t, dir, "gitflow.worktree.feature/keep.managed"); v != "true" {
		t.Errorf("Expected the live worktree's marker to survive, got %q", v)
	}
}

// TestWorktreePruneDropsMarkerAfterDetach verifies the sweep is keyed on the
// branch: a live-but-detached worktree keeps its directory and loses its marker.
// Steps:
// 1. Adds a worktree for feature/x and detaches its HEAD
// 2. Runs 'git flow worktree prune'
// 3. Verifies the directory and admin entry survive but the marker is dropped
// 4. Verifies 'worktree list' now reports the row as (detached) … (unmanaged)
func TestWorktreePruneDropsMarkerAfterDetach(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer os.RemoveAll(worktreeRootFor(dir))
	createFreeBranch(t, dir, "feature/x")
	wtPath := addWorktree(t, dir, "feature/x")

	if out, err := testutil.RunGit(t, wtPath, "checkout", "--detach"); err != nil {
		t.Fatalf("Failed to detach worktree HEAD: %v\nOutput: %s", err, out)
	}

	if out, err := testutil.RunGitFlow(t, dir, "worktree", "prune"); err != nil {
		t.Fatalf("worktree prune failed: %v\nOutput: %s", err, out)
	}

	if info, err := os.Stat(wtPath); err != nil || !info.IsDir() {
		t.Fatalf("Expected the live worktree directory to survive prune: %v", err)
	}
	if !strings.Contains(gitWorktreeList(t, dir), wtPath) {
		t.Error("Expected the live admin entry to survive prune")
	}
	if testutil.GitConfigExists(t, dir, "gitflow.worktree.feature/x.managed") {
		t.Error("Expected the marker of a branch with no live worktree to be dropped")
	}

	output, err := testutil.RunGitFlow(t, dir, "worktree", "list")
	if err != nil {
		t.Fatalf("worktree list failed: %v\nOutput: %s", err, output)
	}
	rows := worktreeRows(output)
	if len(rows) != 1 {
		t.Fatalf("Expected exactly one row, got %d: %s", len(rows), output)
	}
	if !strings.HasPrefix(rows[0], "(detached)") || !strings.HasSuffix(rows[0], "(unmanaged)") {
		t.Errorf("Expected a '(detached) … (unmanaged)' row, got %q", rows[0])
	}
}

// ---------------------------------------------------------------------------
// Group H — the GIT_FLOW_CD_FILE navigation channel
// ---------------------------------------------------------------------------

// TestWorktreeAddWritesCDFile covers scenario 23: add hands back the new path.
// Steps:
// 1. Creates an empty GIT_FLOW_CD_FILE and a free feature/x branch
// 2. Runs 'git flow worktree add feature/x' with the variable set
// 3. Verifies exit code 0
// 4. Verifies the file holds the absolute path of the created worktree
func TestWorktreeAddWritesCDFile(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer os.RemoveAll(worktreeRootFor(dir))
	createFreeBranch(t, dir, "feature/x")
	cdFile := cdFilePath(t)

	output, err := testutil.RunGitFlowWithEnv(t, dir, cdEnv(cdFile), "worktree", "add", "feature/x")
	if err != nil {
		t.Fatalf("worktree add failed: %v\nOutput: %s", err, output)
	}

	want := computedWorktreePath(t, dir, "feature/x")
	if got := readCDFile(t, cdFile); got != want {
		t.Errorf("Expected CD file to hold %q, got %q", want, got)
	}
}

// TestWorktreeAddWithoutCDFileEnv covers scenario 24: with the variable unset the
// command prints exactly its confirmation and cd hint, and nothing else.
// Steps:
// 1. Runs 'git flow worktree add feature/x' with GIT_FLOW_CD_FILE explicitly empty
// 2. Verifies stdout equals exactly the confirmation and cd hint lines, proving no machine-readable protocol line rides along
// 3. Verifies stderr is empty, so nothing was diverted there either
func TestWorktreeAddWithoutCDFileEnv(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer os.RemoveAll(worktreeRootFor(dir))
	createFreeBranch(t, dir, "feature/x")

	stdout, stderr, err := testutil.RunGitFlowStreamsWithEnv(t, dir, nil, "worktree", "add", "feature/x")
	if err != nil {
		t.Fatalf("worktree add failed: %v\nStderr: %s", err, stderr)
	}

	wtPath := computedWorktreePath(t, dir, "feature/x")
	want := "Created worktree for branch 'feature/x' at " + wtPath + "\n" +
		"To switch to it: cd " + wtPath + "\n"
	if stdout != want {
		t.Errorf("Expected stdout %q, got %q", want, stdout)
	}
	if stderr != "" {
		t.Errorf("Expected no output on stderr, got %q", stderr)
	}
}

// TestWorktreeAddNoCDFlagLeavesFileEmpty covers scenario 25: --no-cd suppresses
// the write even when the variable is set.
// Steps:
// 1. Creates an empty GIT_FLOW_CD_FILE and a free feature/x branch
// 2. Runs 'git flow worktree add feature/x --no-cd' with the variable set
// 3. Verifies exit code 0 and that the worktree was created
// 4. Verifies the CD file is still zero-length
func TestWorktreeAddNoCDFlagLeavesFileEmpty(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer os.RemoveAll(worktreeRootFor(dir))
	createFreeBranch(t, dir, "feature/x")
	cdFile := cdFilePath(t)

	output, err := testutil.RunGitFlowWithEnv(t, dir, cdEnv(cdFile), "worktree", "add", "feature/x", "--no-cd")
	if err != nil {
		t.Fatalf("worktree add --no-cd failed: %v\nOutput: %s", err, output)
	}
	if _, err := os.Stat(computedWorktreePath(t, dir, "feature/x")); err != nil {
		t.Errorf("Expected the worktree to be created: %v", err)
	}
	assertCDFileEmpty(t, cdFile)
}

// TestWorktreeAddCustomPathWritesCDFile covers scenario 26: --path wins in the
// navigation channel too.
// Steps:
// 1. Creates an empty GIT_FLOW_CD_FILE and a free feature/x branch
// 2. Runs 'git flow worktree add feature/x --path ./wt-custom-cd' with the variable set
// 3. Verifies the file holds the absolute form of the custom path
// 4. Verifies it does not hold the computed template path
func TestWorktreeAddCustomPathWritesCDFile(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer os.RemoveAll(worktreeRootFor(dir))
	createFreeBranch(t, dir, "feature/x")
	cdFile := cdFilePath(t)

	output, err := testutil.RunGitFlowWithEnv(t, dir, cdEnv(cdFile), "worktree", "add", "feature/x", "--path", "./wt-custom-cd")
	if err != nil {
		t.Fatalf("worktree add --path failed: %v\nOutput: %s", err, output)
	}

	want := filepath.Join(testutil.EvalPath(t, dir), "wt-custom-cd")
	got := readCDFile(t, cdFile)
	if got != want {
		t.Errorf("Expected CD file to hold %q, got %q", want, got)
	}
	if got == computedWorktreePath(t, dir, "feature/x") {
		t.Error("Expected the CD file to hold the --path destination, not the computed path")
	}
}

// TestWorktreeAddFailureLeavesCDFileEmpty covers scenario 27: a failed add writes
// nothing and reports on stderr.
// Steps:
// 1. Creates an empty GIT_FLOW_CD_FILE
// 2. Runs 'git flow worktree add feature/nope' for a branch that does not exist
// 3. Verifies exit code 5, an error on stderr, and empty stdout
// 4. Verifies the CD file is still zero-length
func TestWorktreeAddFailureLeavesCDFileEmpty(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer os.RemoveAll(worktreeRootFor(dir))
	cdFile := cdFilePath(t)

	stdout, stderr, err := testutil.RunGitFlowStreamsWithEnv(t, dir, cdEnv(cdFile), "worktree", "add", "feature/nope")
	if got := worktreeExitCode(err); got != 5 {
		t.Fatalf("Expected exit code 5, got %d\nStdout: %s\nStderr: %s", got, stdout, stderr)
	}
	if !strings.Contains(stderr, "feature/nope") {
		t.Errorf("Expected the error on stderr, got: %s", stderr)
	}
	if stdout != "" {
		t.Errorf("Expected empty stdout, got %q", stdout)
	}
	assertCDFileEmpty(t, cdFile)
}

// TestWorktreeRemoveFromInsideWritesMainWorktreePath covers scenario 28: removing
// the worktree the user is standing in hands back the main worktree.
// Steps:
// 1. Adds a worktree for feature/x and creates an empty GIT_FLOW_CD_FILE
// 2. Runs 'git flow worktree remove feature/x' from inside that worktree
// 3. Verifies exit code 0 and that the worktree is gone
// 4. Verifies the CD file holds the main worktree root
func TestWorktreeRemoveFromInsideWritesMainWorktreePath(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer os.RemoveAll(worktreeRootFor(dir))
	createFreeBranch(t, dir, "feature/x")
	wtPath := addWorktree(t, dir, "feature/x")
	cdFile := cdFilePath(t)

	output, err := testutil.RunGitFlowWithEnv(t, wtPath, cdEnv(cdFile), "worktree", "remove", "feature/x")
	if err != nil {
		t.Fatalf("worktree remove failed: %v\nOutput: %s", err, output)
	}
	if _, err := os.Stat(wtPath); !os.IsNotExist(err) {
		t.Errorf("Expected worktree directory %q to be gone", wtPath)
	}
	// Both sides symlink-resolved: git reports /private/var on macOS while the
	// temp dir is /var.
	want := testutil.EvalPath(t, dir)
	if got := readCDFile(t, cdFile); got != want {
		t.Errorf("Expected CD file to hold the main worktree %q, got %q", want, got)
	}
}

// TestWorktreeRemoveFromOutsideLeavesCDFileEmpty covers scenario 29: removing a
// worktree the user is not inside writes nothing.
// Steps:
// 1. Adds a worktree for feature/x and creates an empty GIT_FLOW_CD_FILE
// 2. Runs 'git flow worktree remove feature/x' from the main worktree
// 3. Verifies exit code 0 and that the worktree is removed
// 4. Verifies the CD file is still zero-length
func TestWorktreeRemoveFromOutsideLeavesCDFileEmpty(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer os.RemoveAll(worktreeRootFor(dir))
	createFreeBranch(t, dir, "feature/x")
	wtPath := addWorktree(t, dir, "feature/x")
	cdFile := cdFilePath(t)

	output, err := testutil.RunGitFlowWithEnv(t, dir, cdEnv(cdFile), "worktree", "remove", "feature/x")
	if err != nil {
		t.Fatalf("worktree remove failed: %v\nOutput: %s", err, output)
	}
	if _, err := os.Stat(wtPath); !os.IsNotExist(err) {
		t.Errorf("Expected worktree directory %q to be gone", wtPath)
	}
	assertCDFileEmpty(t, cdFile)
}

// TestWorktreeRemoveRefusedLeavesCDFileEmpty covers scenario 30: a refused
// removal writes no destination, even from inside the target.
// Steps:
// 1. Adds a worktree for feature/x and modifies a tracked file inside it
// 2. Runs 'git flow worktree remove feature/x' from inside it, variable set
// 3. Verifies exit code 6 and that the worktree still exists
// 4. Verifies the CD file is still zero-length
func TestWorktreeRemoveRefusedLeavesCDFileEmpty(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer os.RemoveAll(worktreeRootFor(dir))
	createFreeBranch(t, dir, "feature/x")
	wtPath := addWorktree(t, dir, "feature/x")
	cdFile := cdFilePath(t)

	if err := os.WriteFile(filepath.Join(wtPath, "README.md"), []byte("modified"), 0644); err != nil {
		t.Fatalf("Failed to modify tracked file: %v", err)
	}

	output, err := testutil.RunGitFlowWithEnv(t, wtPath, cdEnv(cdFile), "worktree", "remove", "feature/x")
	if got := worktreeExitCode(err); got != 6 {
		t.Fatalf("Expected exit code 6, got %d\nOutput: %s", got, output)
	}
	if _, err := os.Stat(wtPath); err != nil {
		t.Errorf("Expected the worktree to survive a refusal: %v", err)
	}
	assertCDFileEmpty(t, cdFile)
}

// TestWorktreeAddWithUnwritableCDFileWarns covers scenario 31: an unwritable
// destination warns but does not fail the operation.
// Steps:
// 1. Points GIT_FLOW_CD_FILE at an existing DIRECTORY, which can never be written as a file
// 2. Runs 'git flow worktree add feature/x'
// 3. Verifies exit code 0, the worktree exists, and the marker is written
// 4. Verifies stderr carries a Warning naming the destination and stdout still carries the hint
func TestWorktreeAddWithUnwritableCDFileWarns(t *testing.T) {
	t.Parallel()
	dir := initWorktreeRepo(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer os.RemoveAll(worktreeRootFor(dir))
	createFreeBranch(t, dir, "feature/x")

	// A directory destination fails with EISDIR on every platform and for every
	// user, unlike a 0500 directory (root-exempt, ACL-dependent, Windows-skipped).
	destination := t.TempDir()

	stdout, stderr, err := testutil.RunGitFlowStreamsWithEnv(t, dir, cdEnv(destination), "worktree", "add", "feature/x")
	if err != nil {
		t.Fatalf("Expected worktree add to succeed, got %v\nStderr: %s", err, stderr)
	}

	wtPath := computedWorktreePath(t, dir, "feature/x")
	if info, err := os.Stat(wtPath); err != nil || !info.IsDir() {
		t.Fatalf("Expected the worktree to be created: %v", err)
	}
	if v := testutil.GitConfigValue(t, dir, "gitflow.worktree.feature/x.managed"); v != "true" {
		t.Errorf("Expected the marker to be written, got %q", v)
	}
	if !strings.Contains(stderr, "Warning:") || !strings.Contains(stderr, destination) {
		t.Errorf("Expected a Warning naming %q on stderr, got: %s", destination, stderr)
	}
	if !strings.Contains(stdout, "Created worktree for branch 'feature/x'") ||
		!strings.Contains(stdout, "cd "+wtPath) {
		t.Errorf("Expected the confirmation and cd hint on stdout, got: %s", stdout)
	}
}

// ---------------------------------------------------------------------------
// Group I — shared-config regression
// ---------------------------------------------------------------------------

// TestWorktreeMarkerNotCopiedToSharedConfig verifies the provenance marker is
// machine-local: it never reaches .gitflow, never reads as drift, and is never
// deleted by a sync.
// Steps:
// 1. Runs 'git flow init --shared --defaults' and adds a worktree for feature/x
// 2. Verifies .gitflow carries no worktree. line and 'config status' exits 0
// 3. Runs 'git flow config sync'
// 4. Verifies the local marker survived and 'config status' still exits 0
func TestWorktreeMarkerNotCopiedToSharedConfig(t *testing.T) {
	t.Parallel()
	dir := initSharedDefaults(t)
	defer testutil.CleanupTestRepo(t, dir)
	defer os.RemoveAll(worktreeRootFor(dir))
	createFreeBranch(t, dir, "feature/x")
	addWorktree(t, dir, "feature/x")

	shared, err := os.ReadFile(testutil.SharedConfigPath(dir))
	if err != nil {
		t.Fatalf("Failed to read .gitflow: %v", err)
	}
	if strings.Contains(string(shared), "worktree.") {
		t.Errorf("Expected no worktree marker in .gitflow, got:\n%s", string(shared))
	}

	// A wrongly shared-managed marker present locally but absent from .gitflow
	// reads as drift — this is what catches a missing carve-out.
	if out, err := testutil.RunGitFlow(t, dir, "config", "status"); err != nil {
		t.Fatalf("Expected 'config status' to report no drift, got %v\nOutput: %s", err, out)
	}

	if out, err := testutil.RunGitFlow(t, dir, "config", "sync"); err != nil {
		t.Fatalf("config sync failed: %v\nOutput: %s", err, out)
	}

	// sync removes local shared-managed keys absent from .gitflow, so a broken
	// carve-out destroys the marker here rather than publishing it.
	if v := testutil.GitConfigValue(t, dir, "gitflow.worktree.feature/x.managed"); v != "true" {
		t.Errorf("Expected the marker to survive 'config sync', got %q", v)
	}
	if out, err := testutil.RunGitFlow(t, dir, "config", "status"); err != nil {
		t.Fatalf("Expected 'config status' to report no drift after sync, got %v\nOutput: %s", err, out)
	}
}
