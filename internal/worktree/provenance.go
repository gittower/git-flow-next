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

// ListMarkers returns the branch names whose provenance marker says git-flow
// created the worktree, read from local config only — the scope markers are
// written to.
//
// The VALUE decides, not the key's presence, so every consumer agrees with
// IsManaged. MarkManaged only ever writes "true", but a hand-written
// gitflow.worktree.<branch>.managed=false must read unmanaged wherever it is
// read: over-claiming managed-ness is the direction that misleads about whether
// the cleanup commands remove a worktree or merely detach it.
func ListMarkers(repo *git.Repo) ([]string, error) {
	order, values, err := listMarkers(repo)
	if err != nil {
		return nil, err
	}

	var branches []string
	for _, branch := range order {
		if config.ParseBool(values[branch]) {
			branches = append(branches, branch)
		}
	}
	return branches, nil
}

// listMarkers returns every marked branch in the order git reported it, plus each
// branch's raw marker value.
//
// A branch is listed once however many values its key carries, and the LAST value
// wins, which is what `git config --get` returns and therefore what IsManaged
// sees.
func listMarkers(repo *git.Repo) ([]string, map[string]string, error) {
	lines, err := repo.GetConfigLocalRegexpLines(`^gitflow\.worktree\..*\.managed$`)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list worktree provenance markers: %w", err)
	}

	var order []string
	values := map[string]string{}
	for _, line := range lines {
		fields := strings.SplitN(strings.TrimSpace(line), " ", 2)
		match := markerKeyPattern.FindStringSubmatch(fields[0])
		if match == nil {
			continue
		}
		branch := match[1]
		if _, seen := values[branch]; !seen {
			order = append(order, branch)
		}
		values[branch] = ""
		if len(fields) > 1 {
			values[branch] = fields[1]
		}
	}
	return order, values, nil
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
//
// The sweep reads the marker KEYS, not ListMarkers' filtered view: a marker
// written false still has a key, and leaving it behind would let it outlive the
// worktree it describes.
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

	markers, _, err := listMarkers(repo)
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
