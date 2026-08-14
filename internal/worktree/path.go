// Package worktree computes where a branch's worktree belongs and records which
// worktrees git-flow created.
package worktree

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/gittower/git-flow-next/internal/config"
	"github.com/gittower/git-flow-next/internal/git"
)

// DefaultTemplate is the path template used when gitflow.worktreePath is unset
// or empty: a sibling directory of the repository, one subdirectory per branch.
const DefaultTemplate = "../{{ repo }}-worktrees/{{ branch }}"

// TemplateConfigKey is the git config key holding the path template.
const TemplateConfigKey = "gitflow.worktreePath"

// placeholderPattern matches {{ name }} and {{name}} alike, so both brace forms
// expand identically. This is deliberately not internal/util's %b/%p placeholder
// syntax, which serves hook and tag-message expansion.
var placeholderPattern = regexp.MustCompile(`\{\{\s*(\w+)\s*\}\}`)

// ComputePath returns the absolute path where branch's worktree belongs,
// according to the gitflow.worktreePath template.
//
// The template supports {{ repo }} (the main worktree's directory name),
// {{ branch }} (the full branch name), {{ branchName }} (the branch without its
// topic prefix), and {{ topicType }} (the topic branch type, empty for a
// non-topic branch). A leading ~ expands to the user's home directory; a
// relative template resolves against the MAIN worktree root, not the current
// one. The result is always absolute and uses host path separators.
//
// It has no side effects: nothing is created, not even parent directories.
func ComputePath(cfg *config.Config, repo *git.Repo, branch string) (string, error) {
	if strings.TrimSpace(branch) == "" {
		return "", fmt.Errorf("branch name must not be empty")
	}

	mainWorkTree, err := repo.MainWorkTree()
	if err != nil {
		return "", fmt.Errorf("failed to resolve the main worktree: %w", err)
	}

	topicType, branchName := splitTopicBranch(cfg, branch)
	expanded := placeholderPattern.ReplaceAllStringFunc(resolveTemplate(cfg, repo), func(match string) string {
		switch placeholderPattern.FindStringSubmatch(match)[1] {
		case "repo":
			return filepath.Base(mainWorkTree)
		case "branch":
			return branch
		case "branchName":
			return branchName
		case "topicType":
			return topicType
		default:
			// Unknown variables are left verbatim rather than silently dropped,
			// so a typo shows up in the path instead of vanishing.
			return match
		}
	})

	if rest, ok := tildeRest(expanded); ok {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to expand '~' in path template: %w", err)
		}
		return filepath.Join(home, filepath.FromSlash(rest)), nil
	}

	// Each of the three returns below normalizes the expanded template: an empty
	// {{ topicType }} leaves a doubled or trailing separator behind, and
	// filepath.Join and filepath.Clean both collapse it.
	path := filepath.FromSlash(expanded)
	if filepath.IsAbs(path) {
		return filepath.Clean(path), nil
	}
	return filepath.Join(mainWorkTree, path), nil
}

// resolveTemplate returns the configured path template, or DefaultTemplate when
// it is unset or empty.
//
// The key is read case-insensitively on purpose: a direct `git config --get`
// honors the camelCase spelling, but git lowercases variable names in
// --get-regexp output, so the same key arrives in cfg.CommandConfig as
// "gitflow.worktreepath". Falling back to a case-insensitive scan of the loaded
// config keeps both spellings working.
func resolveTemplate(cfg *config.Config, repo *git.Repo) string {
	if value, err := repo.GetConfig(TemplateConfigKey); err == nil && strings.TrimSpace(value) != "" {
		return strings.TrimSpace(value)
	}
	if cfg != nil {
		for key, value := range cfg.CommandConfig {
			if strings.EqualFold(key, TemplateConfigKey) && strings.TrimSpace(value) != "" {
				return strings.TrimSpace(value)
			}
		}
	}
	return DefaultTemplate
}

// splitTopicBranch reports the topic branch type whose prefix matches branch and
// the branch name with that prefix removed. A non-topic branch yields an empty
// type and the branch name unchanged. The longest matching prefix wins (with the
// type name as a tie-break) so nested prefixes resolve deterministically.
func splitTopicBranch(cfg *config.Config, branch string) (topicType string, branchName string) {
	branchName = branch
	if cfg == nil {
		return "", branchName
	}

	longest := 0
	for name, branchConfig := range cfg.Branches {
		if branchConfig.Type != string(config.BranchTypeTopic) || branchConfig.Prefix == "" {
			continue
		}
		if !strings.HasPrefix(branch, branchConfig.Prefix) {
			continue
		}
		length := len(branchConfig.Prefix)
		if length < longest || (length == longest && name >= topicType) {
			continue
		}
		longest = length
		topicType = name
		branchName = strings.TrimPrefix(branch, branchConfig.Prefix)
	}
	return topicType, branchName
}

// tildeRest reports whether path is home-rooted ("~" or "~/…") and returns the
// part after the tilde. A "~user" form is deliberately not expanded: git-flow
// has no portable way to resolve another user's home, and leaving it alone is
// better than silently rooting it in the current user's home.
func tildeRest(path string) (rest string, ok bool) {
	if path != "~" && !strings.HasPrefix(path, "~/") {
		return "", false
	}
	return strings.TrimPrefix(strings.TrimPrefix(path, "~"), "/"), true
}
