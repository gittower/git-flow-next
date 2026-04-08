package hooks

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// scriptCommand creates an exec.Cmd for running a hook/filter script.
// On Windows, scripts are executed via "sh" (shipped with Git for Windows),
// since exec.Command cannot run shell scripts directly on Windows.
func scriptCommand(path string, args ...string) *exec.Cmd {
	if runtime.GOOS == "windows" {
		return exec.Command("sh", append([]string{path}, args...)...)
	}
	return exec.Command(path, args...)
}

// RunVersionFilter executes a version filter for the given branch type and returns the modified version.
// The filter script name is: filter-flow-{branchType}-start-version
// If the filter does not exist or is not executable, the original version is returned.
// If the filter exits with a non-zero status, an error is returned.
func RunVersionFilter(gitDir string, branchType string, version string) (string, error) {
	filterName := GetFilterName(branchType, "start", FilterTargetVersion)
	hooksDir := getHooksDir(gitDir)
	scriptPath := filepath.Join(hooksDir, filterName)

	// Check if filter exists and is executable
	if !isExecutable(scriptPath) {
		return version, nil
	}

	// Execute the filter with version as argument
	repoRoot := filepath.Dir(getCommonGitDir(gitDir))
	result, err := runFilter(scriptPath, version, nil, repoRoot)
	if err != nil {
		return "", fmt.Errorf("version filter '%s' failed: %w", filterName, err)
	}

	// If the filter returned empty output, use the original version
	if result == "" {
		return version, nil
	}

	return result, nil
}

// RunTagMessageFilter executes a tag message filter for the given branch type and returns the modified message.
// The filter script name is: filter-flow-{branchType}-finish-tag-message
// If the filter does not exist or is not executable, the original message is returned.
// If the filter exits with a non-zero status, an error is returned.
func RunTagMessageFilter(gitDir string, branchType string, ctx FilterContext) (string, error) {
	filterName := GetFilterName(branchType, "finish", FilterTargetTagMessage)
	hooksDir := getHooksDir(gitDir)
	scriptPath := filepath.Join(hooksDir, filterName)

	// Check if filter exists and is executable
	if !isExecutable(scriptPath) {
		return ctx.TagMessage, nil
	}

	// Build environment variables for the filter
	env := buildFilterEnv(ctx)

	// Execute the filter with version and message as arguments
	// The filter receives: $1 = version, $2 = message
	repoRoot := filepath.Dir(getCommonGitDir(gitDir))
	result, err := runFilterWithArgs(scriptPath, []string{ctx.Version, ctx.TagMessage}, env, repoRoot)
	if err != nil {
		return "", fmt.Errorf("tag message filter '%s' failed: %w", filterName, err)
	}

	// If the filter returned empty output, use the original message
	if result == "" {
		return ctx.TagMessage, nil
	}

	return result, nil
}

// isExecutable checks if a file exists and is executable.
// On Windows, NTFS doesn't store Unix permission bits, so any existing
// non-directory file is considered executable (matching Git for Windows behavior).
func isExecutable(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return isExecutableFileInfo(info)
}

// isExecutableFileInfo checks if a file is executable given its FileInfo.
// On Windows, any non-directory file is considered executable.
func isExecutableFileInfo(info os.FileInfo) bool {
	if runtime.GOOS == "windows" {
		return !info.IsDir()
	}
	return info.Mode()&0111 != 0
}

// buildFilterEnv builds environment variables for filter execution.
func buildFilterEnv(ctx FilterContext) []string {
	env := os.Environ()
	env = append(env,
		fmt.Sprintf("BRANCH_TYPE=%s", ctx.BranchType),
		fmt.Sprintf("BRANCH_NAME=%s", ctx.BranchName),
		fmt.Sprintf("BASE_BRANCH=%s", ctx.BaseBranch),
		fmt.Sprintf("ORIGIN=%s", ctx.Origin),
	)
	if ctx.FullBranch != "" {
		env = append(env, fmt.Sprintf("BRANCH=%s", ctx.FullBranch))
	}
	if ctx.Version != "" {
		env = append(env, fmt.Sprintf("VERSION=%s", ctx.Version))
	}
	return env
}

// runFilter executes a filter script with input as argument.
func runFilter(scriptPath string, input string, env []string, repoRoot string) (string, error) {
	cmd := scriptCommand(scriptPath, input)

	if env != nil {
		cmd.Env = env
	}

	// Set working directory to repository root
	cmd.Dir = repoRoot

	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("exit code %d: %s", exitErr.ExitCode(), string(exitErr.Stderr))
		}
		return "", err
	}

	return strings.TrimSpace(string(output)), nil
}

// runFilterWithArgs executes a filter script with arguments.
func runFilterWithArgs(scriptPath string, args []string, env []string, repoRoot string) (string, error) {
	cmd := scriptCommand(scriptPath, args...)

	if env != nil {
		cmd.Env = env
	}

	// Set working directory to repository root
	cmd.Dir = repoRoot

	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("exit code %d: %s", exitErr.ExitCode(), string(exitErr.Stderr))
		}
		return "", err
	}

	return strings.TrimSpace(string(output)), nil
}
