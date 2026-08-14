package testutil

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

var gitFlowPath string

func init() {
	// Check for GIT_FLOW_PATH environment variable first
	if envPath := os.Getenv("GIT_FLOW_PATH"); envPath != "" {
		gitFlowPath = envPath
		return
	}

	// Get the absolute path to the git-flow binary
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	// If we're in a test subdirectory, go up to the project root
	if strings.HasSuffix(wd, "test/cmd") {
		wd = filepath.Join(wd, "..", "..")
	}
	gitFlowPath = filepath.Join(wd, "git-flow")
}

// BuildGitFlow compiles the git-flow binary into a temporary directory and
// points RunGitFlow at it, so tests never execute a stale or missing binary.
// If the GIT_FLOW_PATH environment variable is set, no build is performed and
// that binary is used instead. The returned cleanup function removes the
// temporary directory. Intended to be called from TestMain in packages that
// execute the git-flow binary.
func BuildGitFlow() (func(), error) {
	if os.Getenv("GIT_FLOW_PATH") != "" {
		return func() {}, nil
	}

	tmpDir, err := os.MkdirTemp("", "git-flow-test-bin-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temporary directory: %w", err)
	}

	binary := filepath.Join(tmpDir, "git-flow")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}

	// Build by module path so this works from any package directory
	cmd := exec.Command("go", "build", "-o", binary, "github.com/gittower/git-flow-next")
	output, err := cmd.CombinedOutput()
	if err != nil {
		os.RemoveAll(tmpDir)
		return nil, fmt.Errorf("failed to build git-flow: %w\nOutput: %s", err, output)
	}

	gitFlowPath = binary
	return func() { os.RemoveAll(tmpDir) }, nil
}

// GitFlowPath returns the path of the git-flow binary under test. Use this
// instead of resolving the binary path manually, so tests run the binary
// built by TestMain (or the GIT_FLOW_PATH override).
func GitFlowPath() string {
	return gitFlowPath
}

// ConfigureGitIdentity sets the git user identity in a repository so commits
// work on machines without a global git configuration (e.g. CI runners).
// SetupTestRepo does this automatically; call this for repositories created
// by other means, such as clones.
func ConfigureGitIdentity(t *testing.T, dir string) {
	t.Helper()
	_, err := RunGit(t, dir, "config", "user.name", "Test User")
	if err != nil {
		t.Fatalf("Failed to configure Git user name: %v", err)
	}
	_, err = RunGit(t, dir, "config", "user.email", "test@example.com")
	if err != nil {
		t.Fatalf("Failed to configure Git user email: %v", err)
	}
}

// RunGit runs a git command in the specified directory and returns its output
func RunGit(t *testing.T, dir string, args ...string) (string, error) {
	return RunGitWithEnv(t, dir, nil, args...)
}

// RunGitWithEnv runs a git command in the specified directory with extra
// environment variables appended to the child process env, returning its output.
// The extra env is scoped to the subprocess only — it never mutates the test
// process environment — so concurrent tests can isolate settings like
// GIT_CONFIG_GLOBAL without leaking into each other (see RunGit for the
// no-extra-env case).
func RunGitWithEnv(t *testing.T, dir string, env []string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), env...)
	// Set GIT_EDITOR to colon (:) to prevent interactive editor from opening.
	// The colon is a shell builtin that does nothing and returns success.
	// Appended last so it always wins over any caller-supplied GIT_EDITOR
	// (exec dedups env keeping the final value), keeping git non-interactive.
	cmd.Env = append(cmd.Env, "GIT_EDITOR=:")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("git command failed: %w\nOutput: %s", err, output)
	}
	return string(output), nil
}

// gitFlowEnv assembles the environment for a git-flow subprocess.
//
// Order matters, and it is: the test process environment, then an empty
// GIT_FLOW_CD_FILE, then the caller's extra env, then GIT_EDITOR=: last.
// exec dedups env keeping the FINAL value of a key, so:
//   - the ambient GIT_FLOW_CD_FILE of a developer who exported the variable is
//     neutralized for every test that does not opt in — otherwise each run would
//     write into that developer's file and the "variable unset" scenario would
//     silently stop testing anything;
//   - a caller that does pass GIT_FLOW_CD_FILE=<path> still wins, since its
//     value comes after the neutralizing one;
//   - GIT_EDITOR=: comes last so caller env can never make git interactive.
func gitFlowEnv(env []string) []string {
	full := append(os.Environ(), "GIT_FLOW_CD_FILE=")
	full = append(full, env...)
	return append(full, "GIT_EDITOR=:")
}

// RunGitFlow runs a git-flow command in the specified directory and returns its output
func RunGitFlow(t *testing.T, dir string, args ...string) (string, error) {
	return RunGitFlowWithEnv(t, dir, nil, args...)
}

// RunGitFlowWithEnv runs a git-flow command in the specified directory with extra
// environment variables appended to the child process env, returning its combined
// output. The extra env is scoped to the subprocess only — it never mutates the
// test process environment — so concurrent tests can isolate settings like
// GIT_FLOW_CD_FILE without leaking into each other (see RunGitFlow for the
// no-extra-env case).
func RunGitFlowWithEnv(t *testing.T, dir string, env []string, args ...string) (string, error) {
	cmd := exec.Command(gitFlowPath, args...)
	cmd.Dir = dir
	cmd.Env = gitFlowEnv(env)
	output, err := cmd.CombinedOutput()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return string(output), &ExitError{
				ExitCode: exitErr.ExitCode(),
				Err:      fmt.Errorf("%s", output),
			}
		}
		return string(output), err
	}
	return string(output), nil
}

// RunGitFlowStreams runs a git-flow command and returns stdout and stderr
// separately. Use it when a test has to prove which stream carried the output;
// RunGitFlow merges the two and cannot distinguish them.
func RunGitFlowStreams(t *testing.T, dir string, args ...string) (string, string, error) {
	return RunGitFlowStreamsWithEnv(t, dir, nil, args...)
}

// RunGitFlowStreamsWithEnv runs a git-flow command with extra environment
// variables and returns stdout and stderr separately. It is the stream-separating
// counterpart of RunGitFlowWithEnv, for tests that need both an environment
// (e.g. GIT_FLOW_CD_FILE) and proof of which stream carried the output.
func RunGitFlowStreamsWithEnv(t *testing.T, dir string, env []string, args ...string) (string, string, error) {
	cmd := exec.Command(gitFlowPath, args...)
	cmd.Dir = dir
	cmd.Env = gitFlowEnv(env)
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			// Report whatever the command actually printed, on either stream,
			// so a failure is diagnosable even when it stays silent on stderr.
			detail := strings.TrimSpace(strings.Join([]string{stderr.String(), stdout.String()}, "\n"))
			if detail == "" {
				detail = err.Error()
			}
			return stdout.String(), stderr.String(), &ExitError{
				ExitCode: exitErr.ExitCode(),
				Err:      fmt.Errorf("%s", detail),
			}
		}
		return stdout.String(), stderr.String(), err
	}
	return stdout.String(), stderr.String(), nil
}

// RunGitFlowWithInput runs a git-flow command with the provided input and returns its output
func RunGitFlowWithInput(t *testing.T, dir string, input string, args ...string) (string, error) {
	cmd := exec.Command(gitFlowPath, args...)
	cmd.Dir = dir
	cmd.Stdin = strings.NewReader(input)
	cmd.Env = gitFlowEnv(nil)
	output, err := cmd.CombinedOutput()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return string(output), &ExitError{
				ExitCode: exitErr.ExitCode(),
				Err:      fmt.Errorf("%s", output),
			}
		}
		return string(output), err
	}
	return string(output), nil
}

// RunGitFlowInteractive runs a git-flow command with the given stdin input and
// forces the first-run interactivity seam ON via GIT_FLOW_ASSUME_INTERACTIVE=1.
// The subprocess test harness cannot allocate a PTY, so git-flow's real
// term.IsTerminal check would always report non-interactive; this test-only env
// var makes the first-run activation decision treat stdin as interactive and
// read the answer from the piped input. Use it for the prompt scenarios (accept
// "y\n" / decline "n\n"); use RunGitFlow (env unset) to exercise the
// non-interactive hint path.
func RunGitFlowInteractive(t *testing.T, dir string, input string, args ...string) (string, error) {
	cmd := exec.Command(gitFlowPath, args...)
	cmd.Dir = dir
	cmd.Stdin = strings.NewReader(input)
	cmd.Env = append(gitFlowEnv(nil), "GIT_FLOW_ASSUME_INTERACTIVE=1")
	output, err := cmd.CombinedOutput()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return string(output), &ExitError{
				ExitCode: exitErr.ExitCode(),
				Err:      fmt.Errorf("%s", output),
			}
		}
		return string(output), err
	}
	return string(output), nil
}

// EvalPath returns path with all symlinks resolved, failing the test if it
// cannot be resolved.
//
// It exists because SetupTestRepo returns the raw os.MkdirTemp path (on macOS
// /var/folders/…) while git reports the resolved one (/private/var/folders/…),
// so any comparison between a test-computed path and a git-reported path must
// resolve both sides.
//
// IMPORTANT: call this only on a path that EXISTS — typically the repository
// root — and build expectations by appending the remaining components with
// filepath.Join. filepath.EvalSymlinks fails on a path that does not exist, so it
// can never be applied to a computed-but-not-yet-created path (e.g. the worktree
// path `worktree path` prints).
func EvalPath(t *testing.T, path string) string {
	t.Helper()
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		t.Fatalf("Failed to resolve symlinks for %q: %v", path, err)
	}
	return resolved
}

// SharedConfigPath returns the path to the committable .gitflow file at the root
// of the given repository directory.
func SharedConfigPath(dir string) string {
	return filepath.Join(dir, ".gitflow")
}

// GitConfigValue returns a single git config value from the repository's LOCAL
// scope (.git/config), or the empty string when the key is unset. Reading from
// local scope (not merged) is what the shared-copy tests need: they verify keys
// actually landed in .git/config.
func GitConfigValue(t *testing.T, dir, key string) string {
	out, err := RunGit(t, dir, "config", "--local", "--get", key)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(out)
}

// GitConfigExists reports whether a key is present in the repository's LOCAL
// git config.
func GitConfigExists(t *testing.T, dir, key string) bool {
	_, err := RunGit(t, dir, "config", "--local", "--get", key)
	return err == nil
}

// GitConfigAll returns all values for a (possibly multi-value) key from the
// repository's LOCAL git config, in file order. Returns nil when unset.
func GitConfigAll(t *testing.T, dir, key string) []string {
	out, err := RunGit(t, dir, "config", "--local", "--get-all", key)
	if err != nil {
		return nil
	}
	var vals []string
	for _, l := range strings.Split(strings.TrimSpace(out), "\n") {
		if l != "" {
			vals = append(vals, l)
		}
	}
	return vals
}

// GitConfigHasPrefix reports whether the repository's LOCAL git config contains
// any key whose name starts with the given prefix (e.g. "gitflow.branch.qa.").
func GitConfigHasPrefix(t *testing.T, dir, prefix string) bool {
	out, err := RunGit(t, dir, "config", "--local", "--get-regexp", "^"+prefix)
	if err != nil {
		return false
	}
	return strings.TrimSpace(out) != ""
}

// SharedConfigValue returns a single value from the .gitflow file, or empty when
// unset.
func SharedConfigValue(t *testing.T, dir, key string) string {
	out, err := RunGit(t, dir, "config", "--file", SharedConfigPath(dir), "--get", key)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(out)
}

// SharedConfigAll returns all values for a key from the .gitflow file, in order.
func SharedConfigAll(t *testing.T, dir, key string) []string {
	out, err := RunGit(t, dir, "config", "--file", SharedConfigPath(dir), "--get-all", key)
	if err != nil {
		return nil
	}
	var vals []string
	for _, l := range strings.Split(strings.TrimSpace(out), "\n") {
		if l != "" {
			vals = append(vals, l)
		}
	}
	return vals
}

// SharedConfigSet sets a key in the .gitflow file via `git config --file`.
func SharedConfigSet(t *testing.T, dir, key, value string) {
	t.Helper()
	if _, err := RunGit(t, dir, "config", "--file", SharedConfigPath(dir), key, value); err != nil {
		t.Fatalf("Failed to set %s in .gitflow: %v", key, err)
	}
}

// SharedConfigHasPrefix reports whether the .gitflow file contains any key whose
// name starts with the given prefix.
func SharedConfigHasPrefix(t *testing.T, dir, prefix string) bool {
	out, err := RunGit(t, dir, "config", "--file", SharedConfigPath(dir), "--get-regexp", "^"+prefix)
	if err != nil {
		return false
	}
	return strings.TrimSpace(out) != ""
}

// AuthorSharedConfig runs `git flow init --shared --defaults` (plus any extra
// args) in a throwaway repository and returns the raw bytes of the resulting
// .gitflow file. Callers copy this into a fresh clone via SetupFreshCloneWithShared.
func AuthorSharedConfig(t *testing.T, extraArgs ...string) []byte {
	t.Helper()
	src := SetupTestRepo(t)
	defer CleanupTestRepo(t, src)
	args := append([]string{"init", "--shared", "--defaults"}, extraArgs...)
	if out, err := RunGitFlow(t, src, args...); err != nil {
		t.Fatalf("Failed to author shared config: %v\nOutput: %s", err, out)
	}
	data, err := os.ReadFile(SharedConfigPath(src))
	if err != nil {
		t.Fatalf("Failed to read authored .gitflow: %v", err)
	}
	return data
}

// SetupFreshCloneWithShared creates a fresh repository that contains a committed
// .gitflow file (with the given content) but NO local gitflow.* keys, simulating
// a fresh clone of a repository that carries a committed .gitflow before its
// first-run activation. Like a real clone of a defaults repo, it also has a
// develop branch (the non-trunk base of the default configuration) so topic
// branches that start from develop can be created after activation.
func SetupFreshCloneWithShared(t *testing.T, gitflowContent []byte) string {
	t.Helper()
	dir := SetupTestRepo(t)
	if err := os.WriteFile(SharedConfigPath(dir), gitflowContent, 0644); err != nil {
		t.Fatalf("Failed to write .gitflow: %v", err)
	}
	if _, err := RunGit(t, dir, "add", ".gitflow"); err != nil {
		t.Fatalf("Failed to stage .gitflow: %v", err)
	}
	if _, err := RunGit(t, dir, "commit", "-m", "Add committed .gitflow"); err != nil {
		t.Fatalf("Failed to commit .gitflow: %v", err)
	}
	// A real clone of a git-flow-defaults repository has develop as well as main.
	// Branch develop AFTER committing .gitflow so both branches carry the file;
	// this keeps the work tree switchable when a test edits the (tracked) .gitflow.
	if _, err := RunGit(t, dir, "branch", "develop"); err != nil {
		t.Fatalf("Failed to create develop branch: %v", err)
	}
	return dir
}

// ClearLocalGitflowConfig removes every gitflow.* key from the repository's LOCAL
// git config, leaving other scopes and any .gitflow file untouched. It is used to
// return a repository that ran `git flow init` to a real "fresh clone" state
// (branches present, but no local git-flow configuration).
func ClearLocalGitflowConfig(t *testing.T, dir string) {
	t.Helper()
	out, err := RunGit(t, dir, "config", "--local", "--get-regexp", "^gitflow\\.")
	if err != nil {
		// No gitflow.* keys present: nothing to clear.
		return
	}
	seen := map[string]bool{}
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		if line == "" {
			continue
		}
		key := strings.SplitN(line, " ", 2)[0]
		if seen[key] {
			continue
		}
		seen[key] = true
		// --unset-all removes multi-value keys too. The key was just enumerated
		// from --get-regexp, so it exists; only exit 5 ("key not found", e.g. a
		// concurrent unset) is benign — any other failure is unexpected and would
		// leave the config partially cleared, so fail the test loudly.
		if _, err := RunGit(t, dir, "config", "--local", "--unset-all", key); err != nil {
			var exitErr *exec.ExitError
			if errors.As(err, &exitErr) && exitErr.ExitCode() == 5 {
				continue
			}
			t.Fatalf("Failed to unset local gitflow config %s: %v", key, err)
		}
	}
}

// SetupTestRepo creates a temporary Git repository for testing
func SetupTestRepo(t *testing.T) string {
	// Create temporary directory
	dir, err := os.MkdirTemp("", "git-flow-test-*")
	if err != nil {
		t.Fatalf("Failed to create temporary directory: %v", err)
	}

	// Initialize Git repository
	_, err = RunGit(t, dir, "init", "--initial-branch=main")
	if err != nil {
		t.Fatalf("Failed to initialize Git repository: %v", err)
	}

	// Configure Git user
	_, err = RunGit(t, dir, "config", "user.name", "Test User")
	if err != nil {
		t.Fatalf("Failed to configure Git user name: %v", err)
	}
	_, err = RunGit(t, dir, "config", "user.email", "test@example.com")
	if err != nil {
		t.Fatalf("Failed to configure Git user email: %v", err)
	}

	// Create initial commit
	err = WriteFile(t, dir, "README.md", "# Test Repository")
	if err != nil {
		t.Fatalf("Failed to create README.md: %v", err)
	}
	_, err = RunGit(t, dir, "add", "README.md")
	if err != nil {
		t.Fatalf("Failed to add README.md: %v", err)
	}
	_, err = RunGit(t, dir, "commit", "-m", "Initial commit")
	if err != nil {
		t.Fatalf("Failed to create initial commit: %v", err)
	}

	return dir
}

// SetupEmptyTestRepo creates a temporary Git repository with no commits.
// This is useful for testing behavior when git-flow init encounters an empty repo.
func SetupEmptyTestRepo(t *testing.T) string {
	dir, err := os.MkdirTemp("", "git-flow-test-*")
	if err != nil {
		t.Fatalf("Failed to create temporary directory: %v", err)
	}

	_, err = RunGit(t, dir, "init", "--initial-branch=main")
	if err != nil {
		t.Fatalf("Failed to initialize Git repository: %v", err)
	}

	_, err = RunGit(t, dir, "config", "user.name", "Test User")
	if err != nil {
		t.Fatalf("Failed to configure Git user name: %v", err)
	}
	_, err = RunGit(t, dir, "config", "user.email", "test@example.com")
	if err != nil {
		t.Fatalf("Failed to configure Git user email: %v", err)
	}

	return dir
}

// CleanupTestRepo removes the temporary test repository
func CleanupTestRepo(t *testing.T, dir string) {
	err := os.RemoveAll(dir)
	if err != nil {
		t.Errorf("Failed to cleanup test repository: %v", err)
	}
}

// WriteFile writes content to a file in the test repository
func WriteFile(t *testing.T, dir string, name string, content string) error {
	path := filepath.Join(dir, name)
	return os.WriteFile(path, []byte(content), 0644)
}

// BranchExists checks if a branch exists in the repository
func BranchExists(t *testing.T, dir string, branch string) bool {
	_, err := RunGit(t, dir, "rev-parse", "--verify", branch)
	return err == nil
}

// GetCurrentBranch returns the name of the current Git branch
func GetCurrentBranch(t *testing.T, dir string) string {
	output, err := RunGit(t, dir, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		t.Fatalf("Failed to get current branch: %v", err)
	}
	return strings.TrimSpace(output)
}

// AddRemote creates a bare repository and adds it as a remote to the test repository
// remoteName defaults to "origin" if empty
// pushAll determines whether to push all branches to the remote
func AddRemote(t *testing.T, dir string, remoteName string, pushAll bool) (string, error) {
	if remoteName == "" {
		remoteName = "origin"
	}

	// Create a temporary directory for the bare repository
	bareDir, err := os.MkdirTemp("", "git-flow-test-remote-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary directory for remote: %w", err)
	}

	// Initialize bare repository with explicit initial branch to avoid
	// depending on user's init.defaultBranch setting
	_, err = RunGit(t, bareDir, "init", "--bare", "--initial-branch=main")
	if err != nil {
		os.RemoveAll(bareDir)
		return "", fmt.Errorf("failed to initialize bare repository: %w", err)
	}

	// Add the bare repository as a remote
	_, err = RunGit(t, dir, "remote", "add", remoteName, bareDir)
	if err != nil {
		os.RemoveAll(bareDir)
		return "", fmt.Errorf("failed to add remote: %w", err)
	}

	// If pushAll is true, push all branches to the remote
	if pushAll {
		_, err = RunGit(t, dir, "push", "--all", remoteName)
		if err != nil {
			os.RemoveAll(bareDir)
			return "", fmt.Errorf("failed to push all branches to remote: %w", err)
		}
	}

	return bareDir, nil
}

// RemoteBranchExists checks if a remote branch exists in the repository
func RemoteBranchExists(t *testing.T, dir string, remote string, branch string) bool {
	ref := fmt.Sprintf("refs/remotes/%s/%s", remote, branch)
	_, err := RunGit(t, dir, "rev-parse", "--verify", "--quiet", ref)
	return err == nil
}

// RemoteTagExists checks whether a tag exists on the given remote by querying
// the remote directly with `git ls-remote --tags`. This reflects what was
// actually pushed to the remote, not the local tag database.
func RemoteTagExists(t *testing.T, dir, remote, tag string) bool {
	t.Helper()
	output, err := RunGit(t, dir, "ls-remote", "--tags", remote)
	if err != nil {
		t.Fatalf("Failed to list remote tags: %v\nOutput: %s", err, output)
	}
	// ls-remote lines are "<sha>\t<ref>"; annotated tags add a peeled
	// "<sha>\trefs/tags/<tag>^{}" line. Compare the ref field exactly so a
	// searched tag is not matched as a prefix of a longer tag (e.g. "1.0"
	// must not match "refs/tags/1.0.0").
	wantRef := fmt.Sprintf("refs/tags/%s", tag)
	for _, line := range strings.Split(output, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		ref := strings.TrimSuffix(fields[1], "^{}")
		if ref == wantRef {
			return true
		}
	}
	return false
}

// CommitsAhead returns the number of commits that branch has that base does not,
// computed via `git rev-list --count <base>..<branch>`. Use it with a remote
// tracking ref as base (e.g. CommitsAhead(t, dir, "origin/main", "main")) to
// determine whether a local branch has been pushed: a result of 0 means the
// branch is up to date with the remote, > 0 means it is ahead (not pushed).
func CommitsAhead(t *testing.T, dir, base, branch string) int {
	t.Helper()
	output, err := RunGit(t, dir, "rev-list", "--count", fmt.Sprintf("%s..%s", base, branch))
	if err != nil {
		t.Fatalf("Failed to count commits ahead (%s..%s): %v\nOutput: %s", base, branch, err, output)
	}
	count, err := strconv.Atoi(strings.TrimSpace(output))
	if err != nil {
		t.Fatalf("Failed to parse commit count %q: %v", output, err)
	}
	return count
}

// SetupTestRepoWithRemote creates a test repository with git-flow initialized and a local remote.
// It pushes main and develop branches to the remote with tracking enabled.
// Returns the local repository directory and the remote repository directory.
// Both should be cleaned up with CleanupTestRepo.
func SetupTestRepoWithRemote(t *testing.T) (string, string) {
	// Create test repository
	dir := SetupTestRepo(t)

	// Initialize git-flow with defaults
	_, err := RunGitFlow(t, dir, "init", "--defaults")
	if err != nil {
		CleanupTestRepo(t, dir)
		t.Fatalf("Failed to initialize git-flow: %v", err)
	}

	// Create a local bare remote repository
	remoteDir, err := AddRemote(t, dir, "origin", false)
	if err != nil {
		CleanupTestRepo(t, dir)
		t.Fatalf("Failed to add remote: %v", err)
	}

	// Push main and develop to remote with tracking
	_, err = RunGit(t, dir, "push", "-u", "origin", "main")
	if err != nil {
		CleanupTestRepo(t, dir)
		CleanupTestRepo(t, remoteDir)
		t.Fatalf("Failed to push main: %v", err)
	}
	_, err = RunGit(t, dir, "push", "-u", "origin", "develop")
	if err != nil {
		CleanupTestRepo(t, dir)
		CleanupTestRepo(t, remoteDir)
		t.Fatalf("Failed to push develop: %v", err)
	}

	return dir, remoteDir
}
