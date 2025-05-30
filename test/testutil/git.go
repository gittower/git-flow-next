package testutil

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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

// RunGit runs a git command in the specified directory and returns its output
func RunGit(t *testing.T, dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("git command failed: %w\nOutput: %s", err, output)
	}
	return string(output), nil
}

// RunGitFlow runs a git-flow command in the specified directory and returns its output
func RunGitFlow(t *testing.T, dir string, args ...string) (string, error) {
	cmd := exec.Command(gitFlowPath, args...)
	cmd.Dir = dir
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

// RunGitFlowWithInput runs a git-flow command with the provided input and returns its output
func RunGitFlowWithInput(t *testing.T, dir string, input string, args ...string) (string, error) {
	cmd := exec.Command(gitFlowPath, args...)
	cmd.Dir = dir
	cmd.Stdin = strings.NewReader(input)
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

	// Initialize bare repository
	_, err = RunGit(t, bareDir, "init", "--bare")
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
