package worktree

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/gittower/git-flow-next/internal/config"
	"github.com/gittower/git-flow-next/internal/git"
)

// markerKeyPrefix and markerKeySuffix bracket the branch name in a provenance
// marker key. Git treats everything between the first and last dot as the
// subsection, so a branch name containing slashes or dots survives round-tripping
// and keeps its case.
const (
	markerKeyPrefix = "gitflow.worktree."
	markerKeySuffix = ".managed"
)

// markerKeyPattern matches a marker key and captures its branch name. Git
// lowercases the section and the final key in --get-regexp output but preserves
// the subsection, so only the branch name carries the author's casing.
var markerKeyPattern = regexp.MustCompile(`^gitflow\.worktree\.(.+)\.managed$`)

// MarkerKey returns the git config key that records git-flow as the creator of
// branch's worktree.
func MarkerKey(branch string) string {
	return markerKeyPrefix + branch + markerKeySuffix
}

// MarkManaged records that git-flow created the worktree for branch. The marker
// is repository-local state, never a user setting: it is excluded from the
// shared-config set so it can never reach a committed .gitflow.
func MarkManaged(repo *git.Repo, branch string) error {
	if err := repo.SetConfig(MarkerKey(branch), "true"); err != nil {
		return fmt.Errorf("failed to record worktree provenance for '%s': %w", branch, err)
	}
	return nil
}

// IsManaged reports whether git-flow created the worktree for branch.
//
// Provenance is never inferred by comparing a worktree's path against the
// template: --path and any later change to gitflow.worktreePath both break that
// correspondence.
//
// The read is LOCAL-scoped, matching MarkManaged, ClearMarker and ListMarkers.
// Reading merged config would let a stray global or system
// gitflow.worktree.<branch>.managed make a hand-made worktree report as
// git-flow's — and since clearing only touches local scope, that marker could
// never be cleared, so the cleanup commands would delete the user's own worktree.
func IsManaged(repo *git.Repo, branch string) bool {
	value, err := repo.GetConfigLocal(MarkerKey(branch))
	return err == nil && config.ParseBool(value)
}

// ClearMarker drops branch's provenance marker, tolerating an absent key.
func ClearMarker(repo *git.Repo, branch string) error {
	if err := repo.UnsetConfigIfPresent(MarkerKey(branch)); err != nil {
		return fmt.Errorf("failed to clear worktree provenance for '%s': %w", branch, err)
	}
	return nil
}

// ListMarkers returns the branch names that carry a provenance marker, read from
// local config only — the scope markers are written to.
func ListMarkers(repo *git.Repo) ([]string, error) {
	lines, err := repo.GetConfigLocalRegexpLines(`^gitflow\.worktree\..*\.managed$`)
	if err != nil {
		return nil, fmt.Errorf("failed to list worktree provenance markers: %w", err)
	}

	var branches []string
	seen := map[string]bool{}
	for _, line := range lines {
		key := strings.SplitN(strings.TrimSpace(line), " ", 2)[0]
		match := markerKeyPattern.FindStringSubmatch(key)
		if match == nil || seen[match[1]] {
			continue
		}
		seen[match[1]] = true
		branches = append(branches, match[1])
	}
	return branches, nil
}

// SweepMarkers drops every marker whose branch has no live worktree, so markers
// never outlive what they describe. Call it after PruneWorktrees, once the stale
// admin entries are gone.
//
// The sweep is keyed on the BRANCH, not the worktree path, so a worktree that
// was detached from its branch also loses its marker even though its directory
// survives. That fails safe: a branch-keyed marker that outlived the detach
// could later be inherited by a worktree the user created by hand for the same
// branch, and the cleanup commands would then delete their work. The worst case
// this way round is a git-flow-created worktree being left alone.
func SweepMarkers(repo *git.Repo) error {
	entries, err := repo.ListWorktrees()
	if err != nil {
		return err
	}
	live := make(map[string]bool, len(entries))
	for _, entry := range entries {
		if entry.Branch != "" {
			live[entry.Branch] = true
		}
	}

	markers, err := ListMarkers(repo)
	if err != nil {
		return err
	}
	for _, branch := range markers {
		if live[branch] {
			continue
		}
		if err := ClearMarker(repo, branch); err != nil {
			return err
		}
	}
	return nil
}
