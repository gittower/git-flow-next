package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/gittower/git-flow-next/internal/git"
)

// SharedConfigFileName is the name of the committable shared-config file that
// lives at the repository work-tree root.
const SharedConfigFileName = ".gitflow"

// SharedConfigPath returns the absolute path to the repository's .gitflow file.
func SharedConfigPath(repo *git.Repo) string {
	return filepath.Join(repo.WorkTree(), SharedConfigFileName)
}

// SharedConfigExists reports whether a .gitflow file is present at the repo root.
func SharedConfigExists(repo *git.Repo) bool {
	return SharedConfigExistsIn(repo.WorkTree())
}

// SharedConfigExistsIn reports whether a .gitflow file is present in dir. It
// serves callers that have no repository handle yet — notably `init --shared
// --init`, which must probe the invocation directory before creating the
// repository that would become its work-tree root.
func SharedConfigExistsIn(dir string) bool {
	info, err := os.Stat(filepath.Join(dir, SharedConfigFileName))
	return err == nil && !info.IsDir()
}

// configEntry is a single parsed git-config line (key + value), preserving
// multi-value order across a slice of entries.
type configEntry struct {
	key   string
	value string
}

// parseConfigEntries turns `git config --get-regexp` output lines into ordered
// (key, value) pairs, dropping blank lines. Git prints one line per value, so a
// multi-value key appears as several consecutive entries in file order.
func parseConfigEntries(lines []string) []configEntry {
	var entries []configEntry
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, " ", 2)
		value := ""
		if len(parts) == 2 {
			value = parts[1]
		}
		entries = append(entries, configEntry{key: parts[0], value: value})
	}
	return entries
}

// IsSharedManagedKey reports whether a gitflow.* key participates in the
// shared-managed set that is copied between .gitflow and local .git/config.
//
// A key is shared-managed iff it is under gitflow. AND is not a gitflow.shared.*
// control key AND is not per-branch runtime metadata (gitflow.branch.<name>.base,
// written by Repo.SetBaseBranch). gitflow.version and gitflow.initialized are
// shared-managed. Everything outside gitflow.* (core.*, alias.*, …) is not.
func IsSharedManagedKey(key string) bool {
	k := strings.ToLower(key)
	if !strings.HasPrefix(k, "gitflow.") {
		return false
	}
	if strings.HasPrefix(k, "gitflow.shared.") {
		return false
	}
	// Per-branch runtime metadata: gitflow.branch.<actual-branch>.base. The
	// branch name may contain dots, so match on the ".base" suffix — no real
	// branch *definition* uses a ".base" property (definitions use type/parent/
	// prefix/strategies).
	if strings.HasPrefix(k, "gitflow.branch.") && strings.HasSuffix(k, ".base") {
		return false
	}
	return true
}

// isHookPathKey reports whether a key points at an executable hook/filter path
// that must not be copied from a committed .gitflow unless explicitly trusted.
func isHookPathKey(key string) bool {
	return strings.EqualFold(key, "gitflow.path.hooks")
}

// SharedTrustHooks reports whether gitflow.shared.trustHooks is set true in the
// repository's merged git config.
func SharedTrustHooks(repo *git.Repo) bool {
	v, err := repo.GetConfig("gitflow.shared.trustHooks")
	return err == nil && v == "true"
}

// SharedAutoInit reports whether gitflow.shared.autoInit is set true in the
// repository's merged git config.
func SharedAutoInit(repo *git.Repo) bool {
	v, err := repo.GetConfig("gitflow.shared.autoInit")
	return err == nil && v == "true"
}

// SharedConfigValid parses the .gitflow file and reports whether it is a valid
// shared config: it must parse AND declare a non-empty gitflow.version. A parse
// failure is returned as an error naming the file.
func SharedConfigValid(repo *git.Repo) (bool, error) {
	entries, err := readSharedEntriesRaw(repo)
	if err != nil {
		return false, err
	}
	for _, e := range entries {
		if strings.EqualFold(e.key, "gitflow.version") && e.value != "" {
			return true, nil
		}
	}
	return false, nil
}

// readSharedEntriesRaw returns every gitflow.* entry in the .gitflow file, in
// order, or a parse error naming the file.
func readSharedEntriesRaw(repo *git.Repo) ([]configEntry, error) {
	path := SharedConfigPath(repo)
	lines, err := git.GetConfigFileRegexpLines(path, "^gitflow\\.")
	if err != nil {
		return nil, fmt.Errorf("shared config file %s is invalid or unparsable: %w", path, err)
	}
	return parseConfigEntries(lines), nil
}

// readSharedManagedEntries returns the ordered shared-managed entries from the
// .gitflow file (filtered by IsSharedManagedKey), or a parse error naming the
// file.
func readSharedManagedEntries(repo *git.Repo) ([]configEntry, error) {
	entries, err := readSharedEntriesRaw(repo)
	if err != nil {
		return nil, err
	}
	var out []configEntry
	for _, e := range entries {
		if IsSharedManagedKey(e.key) {
			out = append(out, e)
		}
	}
	return out, nil
}

// SharedConfigHookPathKeys returns the distinct hook/filter path keys present in
// the .gitflow file (e.g. gitflow.path.hooks). Used to refuse non-interactive
// auto-init when an untrusted shared hook path is present.
func SharedConfigHookPathKeys(repo *git.Repo) ([]string, error) {
	entries, err := readSharedManagedEntries(repo)
	if err != nil {
		return nil, err
	}
	var keys []string
	seen := map[string]bool{}
	for _, e := range entries {
		if isHookPathKey(e.key) && !seen[e.key] {
			seen[e.key] = true
			keys = append(keys, e.key)
		}
	}
	return keys, nil
}

// CopyResult reports the outcome of a shared -> local copy.
type CopyResult struct {
	// SkippedHookKeys lists hook/filter path keys present in .gitflow that were
	// not copied because gitflow.shared.trustHooks was not enabled.
	SkippedHookKeys []string
}

// CopySharedToLocal copies the shared-managed gitflow.* keys from the repo's
// .gitflow file into its local .git/config: it overwrites differing values,
// reproduces multi-value order, and removes local shared-managed keys that are
// absent from the effective shared set (stale removal). Hook/filter path keys are
// copied only when gitflow.shared.trustHooks is enabled; otherwise they are
// skipped (and stale-removed from local if previously copied) and reported in
// CopyResult.SkippedHookKeys. Non-shared-managed keys (gitflow.shared.*, runtime
// .base keys, and anything outside gitflow.*) are never touched.
func CopySharedToLocal(repo *git.Repo) (CopyResult, error) {
	var result CopyResult

	entries, err := readSharedManagedEntries(repo)
	if err != nil {
		return result, err
	}
	trust := SharedTrustHooks(repo)

	// Partition into effective (to copy) vs skipped hook keys, preserving order.
	var effectiveOrder []string
	effectiveValues := map[string][]string{}
	skippedSeen := map[string]bool{}
	for _, e := range entries {
		if isHookPathKey(e.key) && !trust {
			if !skippedSeen[e.key] {
				skippedSeen[e.key] = true
				result.SkippedHookKeys = append(result.SkippedHookKeys, e.key)
			}
			continue
		}
		if _, ok := effectiveValues[e.key]; !ok {
			effectiveOrder = append(effectiveOrder, e.key)
		}
		effectiveValues[e.key] = append(effectiveValues[e.key], e.value)
	}

	// Apply each effective key: unset-all then add the values in order. This
	// overwrites a differing local value and reproduces multi-value order.
	for _, key := range effectiveOrder {
		if err := repo.UnsetAllConfigIfPresent(key); err != nil {
			return result, err
		}
		for _, v := range effectiveValues[key] {
			if err := repo.AddConfig(key, v); err != nil {
				return result, err
			}
		}
	}

	// Stale removal: drop local shared-managed keys not in the effective set.
	localLines, err := repo.GetConfigLocalRegexpLines("^gitflow\\.")
	if err != nil {
		return result, err
	}
	removedSeen := map[string]bool{}
	for _, e := range parseConfigEntries(localLines) {
		if !IsSharedManagedKey(e.key) {
			continue
		}
		if _, keep := effectiveValues[e.key]; keep {
			continue
		}
		if removedSeen[e.key] {
			continue
		}
		removedSeen[e.key] = true
		if err := repo.UnsetAllConfigIfPresent(e.key); err != nil {
			return result, err
		}
	}

	return result, nil
}

// SharedStatus compares the shared-managed keys in the repo's local config
// against its .gitflow file, applying the same hook-trust filter as the copy.
// It returns whether the two are in sync, a sorted list of drifting keys, an
// optional security note describing an intentionally-skipped untrusted hook path
// (which is NOT counted as drift), and any parse error naming the file.
func SharedStatus(repo *git.Repo) (inSync bool, diffs []string, note string, err error) {
	fileEntries, err := readSharedManagedEntries(repo)
	if err != nil {
		return false, nil, "", err
	}
	trust := SharedTrustHooks(repo)

	fileValues := map[string][]string{}
	for _, e := range fileEntries {
		if isHookPathKey(e.key) && !trust {
			if note == "" {
				note = fmt.Sprintf("shared hook path %q is present in .gitflow but skipped (set gitflow.shared.trustHooks=true to apply it)", e.key)
			}
			continue
		}
		fileValues[e.key] = append(fileValues[e.key], e.value)
	}

	localLines, err := repo.GetConfigLocalRegexpLines("^gitflow\\.")
	if err != nil {
		return false, nil, "", err
	}
	localValues := map[string][]string{}
	for _, e := range parseConfigEntries(localLines) {
		if !IsSharedManagedKey(e.key) {
			continue
		}
		// Do NOT filter an untrusted hook path out of the local map: the file
		// (expected) map is filtered, so a hook path lingering in local config
		// after trust was revoked surfaces here as drift — matching what the next
		// sync would remove. The never-copied case (no local hook) stays in sync.
		localValues[e.key] = append(localValues[e.key], e.value)
	}

	keySet := map[string]bool{}
	for k := range fileValues {
		keySet[k] = true
	}
	for k := range localValues {
		keySet[k] = true
	}
	var keys []string
	for k := range keySet {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if !equalStringSlice(fileValues[k], localValues[k]) {
			diffs = append(diffs, k)
		}
	}

	return len(diffs) == 0, diffs, note, nil
}

// equalStringSlice reports whether two string slices are equal element-for-
// element and in the same order (order-sensitive, for multi-value comparison).
func equalStringSlice(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// IsUnconfiguredIgnoringShared reports whether the repository has no effective
// git-flow configuration, deliberately ignoring gitflow.shared.* control keys.
//
// The repo counts as configured (returns false) if merged git config has a
// non-empty gitflow.version, valid git-flow-avh config, or any git-flow-next
// branch *definition* (a gitflow.branch.<name>.type key). A stray command key
// (e.g. gitflow.feature.start.fetch) or a shared-control key alone does not count
// as configuration, so the repo still reads as unconfigured (returns true).
func IsUnconfiguredIgnoringShared(repo *git.Repo) (bool, error) {
	if v, err := repo.GetConfig("gitflow.version"); err == nil && v != "" {
		return false, nil
	}
	if CheckGitFlowAVHConfig(repo) {
		return false, nil
	}
	lines, err := repo.GetConfigRegexpLines("^gitflow\\.branch\\.")
	if err != nil {
		return false, err
	}
	for _, e := range parseConfigEntries(lines) {
		if strings.HasSuffix(strings.ToLower(e.key), ".type") {
			return false, nil
		}
	}
	return true, nil
}

// LoadSharedConfig loads the structured git-flow configuration from the .gitflow
// file itself (not from git config). Used by the `config … --shared` CRUD verbs,
// which edit the shared file and then re-sync it into local.
func LoadSharedConfig(repo *git.Repo) (*Config, error) {
	path := SharedConfigPath(repo)
	allLines, err := git.GetConfigFileRegexpLines(path, "^gitflow\\.")
	if err != nil {
		return nil, fmt.Errorf("shared config file %s is invalid or unparsable: %w", path, err)
	}

	cfg := &Config{
		Version:       "1.0",
		Remote:        "origin",
		CommandConfig: make(map[string]string),
		Branches:      make(map[string]BranchConfig),
	}
	for _, e := range parseConfigEntries(allLines) {
		cfg.CommandConfig[e.key] = e.value
		if strings.EqualFold(e.key, "gitflow.version") && e.value != "" {
			cfg.Version = e.value
		}
		if strings.EqualFold(e.key, "gitflow.origin") && e.value != "" {
			cfg.Remote = e.value
		}
	}

	branchLines, err := git.GetConfigFileRegexpLines(path, "^gitflow\\.branch\\.")
	if err != nil {
		return nil, fmt.Errorf("shared config file %s is invalid or unparsable: %w", path, err)
	}
	cfg.Branches = ParseBranchConfigLines(branchLines)
	return cfg, nil
}

// SharedTopicTypes returns the names of topic branch types declared in the
// repository's .gitflow file, or nil when the file is absent, unparsable, or
// declares none. It is tolerant of errors so it can be called during command
// registration on a fresh clone before activation.
func SharedTopicTypes(repo *git.Repo) []string {
	if !SharedConfigExists(repo) {
		return nil
	}
	cfg, err := LoadSharedConfig(repo)
	if err != nil {
		return nil
	}
	var types []string
	for name, branch := range cfg.Branches {
		if branch.Type == string(BranchTypeTopic) {
			types = append(types, name)
		}
	}
	return types
}
