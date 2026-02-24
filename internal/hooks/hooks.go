package hooks

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/gittower/git-flow-next/internal/git"
)

// RunPreHook executes a pre-hook script. Returns an error if the hook fails (non-zero exit).
// If the hook does not exist or is not executable, it returns nil (no error).
func RunPreHook(gitDir string, branchType string, action HookAction, ctx HookContext) error {
	result := runHook(gitDir, HookPre, branchType, action, ctx)
	if result.Error != nil {
		return result.Error
	}
	if result.Executed && result.ExitCode != 0 {
		if result.Output != "" {
			return fmt.Errorf("pre-hook '%s-flow-%s-%s' failed with exit code %d:\n%s",
				HookPre, branchType, action, result.ExitCode, result.Output)
		}
		return fmt.Errorf("pre-hook '%s-flow-%s-%s' failed with exit code %d",
			HookPre, branchType, action, result.ExitCode)
	}
	return nil
}

// RunPostHook executes a post-hook script. The result is returned but errors do not
// cause the operation to fail. If the hook does not exist or is not executable,
// it returns a result with Executed=false.
func RunPostHook(gitDir string, branchType string, action HookAction, ctx HookContext) HookResult {
	return runHook(gitDir, HookPost, branchType, action, ctx)
}

// configLookup is a package-level function variable for testability.
var configLookup = git.GetConfigInDir

// resolveHooksPath resolves a hooks path that may be relative to the repository root.
func resolveHooksPath(hooksPath, repoRoot string) string {
	if filepath.IsAbs(hooksPath) {
		return hooksPath
	}
	return filepath.Join(repoRoot, hooksPath)
}

// getHooksDir returns the directory where hooks are stored.
// Resolution follows three-level precedence:
//  1. gitflow.path.hooks - git-flow-specific override (avh compatibility)
//  2. core.hooksPath - Git's native hooks path configuration
//  3. .git/hooks - default location (worktree-aware)
func getHooksDir(gitDir string) string {
	commonDir := getCommonGitDir(gitDir)
	repoRoot := filepath.Dir(commonDir)

	// 1. git-flow-specific override (avh compatibility)
	if hooksPath, err := configLookup(repoRoot, "gitflow.path.hooks"); err == nil && hooksPath != "" {
		return resolveHooksPath(hooksPath, repoRoot)
	}

	// 2. Git's core.hooksPath
	if hooksPath, err := configLookup(repoRoot, "core.hooksPath"); err == nil && hooksPath != "" {
		return resolveHooksPath(hooksPath, repoRoot)
	}

	// 3. Default: .git/hooks (worktree-aware)
	return filepath.Join(commonDir, "hooks")
}

// getCommonGitDir returns the common git directory from a git directory path.
// For regular repositories, this returns the gitDir unchanged.
// For worktrees (where gitDir is like /repo/.git/worktrees/<name>), this returns /repo/.git
func getCommonGitDir(gitDir string) string {
	// Normalize the path
	gitDir = filepath.Clean(gitDir)

	// Check if this looks like a worktree git dir
	// Pattern: .../worktrees/<worktree-name>
	parent := filepath.Dir(gitDir)
	parentBase := filepath.Base(parent)

	if parentBase == "worktrees" {
		// This is a worktree git directory
		// Go up two levels: worktrees/<name> -> .git
		return filepath.Dir(parent)
	}

	// Not a worktree, return as-is
	return gitDir
}

// BuildHookArgs constructs the positional arguments for a hook based on the action.
// This matches git-flow-avh's argument passing convention for compatibility.
//
// Arguments by action:
//   - start:   [name, origin, branch, base]
//   - finish:  [name, origin, branch]
//   - publish: [name, origin, branch]
//   - track:   [name, origin, branch]
//   - delete:  [name, origin, branch]
//   - update:  [name, origin, branch, base] (git-flow-next extension)
func BuildHookArgs(action HookAction, ctx HookContext) []string {
	switch action {
	case HookActionStart, HookActionUpdate:
		// start/update: $1=name, $2=origin, $3=branch, $4=base
		return []string{ctx.BranchName, ctx.Origin, ctx.FullBranch, ctx.BaseBranch}
	case HookActionFinish, HookActionPublish, HookActionTrack, HookActionDelete:
		// finish/publish/track/delete: $1=name, $2=origin, $3=branch
		return []string{ctx.BranchName, ctx.Origin, ctx.FullBranch}
	default:
		return []string{ctx.BranchName, ctx.Origin, ctx.FullBranch}
	}
}

// runHook executes a hook script and returns the result.
func runHook(gitDir string, phase HookPhase, branchType string, action HookAction, ctx HookContext) HookResult {
	hookName := fmt.Sprintf("%s-flow-%s-%s", phase, branchType, action)
	hooksDir := getHooksDir(gitDir)
	hookPath := filepath.Join(hooksDir, hookName)

	// Check if hook exists
	info, err := os.Stat(hookPath)
	if os.IsNotExist(err) {
		return HookResult{Executed: false}
	}
	if err != nil {
		return HookResult{Executed: false, Error: err}
	}

	// Check if executable
	if info.Mode()&0111 == 0 {
		// Not executable, skip silently
		return HookResult{Executed: false}
	}

	// Build environment variables
	env := buildHookEnv(ctx, phase)

	// Build positional arguments for git-flow-avh compatibility
	args := BuildHookArgs(action, ctx)

	// Execute hook with arguments
	cmd := exec.Command(hookPath, args...)
	cmd.Env = env
	cmd.Dir = filepath.Dir(gitDir) // Repository root

	output, err := cmd.CombinedOutput()

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			return HookResult{
				Executed: true,
				ExitCode: 1,
				Output:   string(output),
				Error:    err,
			}
		}
	}

	return HookResult{
		Executed: true,
		ExitCode: exitCode,
		Output:   string(output),
		Error:    nil,
	}
}

// buildHookEnv builds environment variables for hook execution.
func buildHookEnv(ctx HookContext, phase HookPhase) []string {
	env := os.Environ()
	env = append(env,
		fmt.Sprintf("BRANCH=%s", ctx.FullBranch),
		fmt.Sprintf("BRANCH_NAME=%s", ctx.BranchName),
		fmt.Sprintf("BRANCH_TYPE=%s", ctx.BranchType),
		fmt.Sprintf("BASE_BRANCH=%s", ctx.BaseBranch),
		fmt.Sprintf("ORIGIN=%s", ctx.Origin),
	)

	if ctx.Version != "" {
		env = append(env, fmt.Sprintf("VERSION=%s", ctx.Version))
	}

	// For post-hooks, include the exit code of the operation
	if phase == HookPost {
		env = append(env, fmt.Sprintf("EXIT_CODE=%d", ctx.ExitCode))
	}

	return env
}

// WithHooks wraps an operation with pre and post hooks.
// The pre-hook is run before the operation. If it fails, the operation is not executed.
// The post-hook is run after the operation, regardless of success or failure.
// The context's ExitCode is set based on the operation result before running the post-hook.
func WithHooks(gitDir string, branchType string, action HookAction, ctx HookContext, operation func() error) error {
	// Run pre-hook
	if err := RunPreHook(gitDir, branchType, action, ctx); err != nil {
		return err
	}

	// Run operation
	opErr := operation()

	// Set exit code for post-hook
	if opErr != nil {
		ctx.ExitCode = 1
	} else {
		ctx.ExitCode = 0
	}

	// Run post-hook (ignore errors from post-hook)
	result := RunPostHook(gitDir, branchType, action, ctx)
	if result.Executed && result.Output != "" {
		// Print post-hook output for visibility
		fmt.Print(result.Output)
	}

	return opErr
}
