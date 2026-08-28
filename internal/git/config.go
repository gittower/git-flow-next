package git

import (
	"fmt"
	"os/exec"
	"strings"
)

// ConfigScope represents the scope for Git configuration operations
type ConfigScope string

const (
	// ConfigScopeDefault uses merged config for reads, local for writes (Git's default behavior)
	ConfigScopeDefault ConfigScope = ""
	// ConfigScopeLocal reads/writes only local repository config (.git/config)
	ConfigScopeLocal ConfigScope = "local"
	// ConfigScopeGlobal reads/writes only global user config (~/.gitconfig)
	ConfigScopeGlobal ConfigScope = "global"
	// ConfigScopeSystem reads/writes only system-wide config (/etc/gitconfig)
	ConfigScopeSystem ConfigScope = "system"
	// ConfigScopeFile reads/writes only the specified file (requires filePath parameter)
	ConfigScopeFile ConfigScope = "file"
)

// GetConfig gets a Git config value from the repository's merged config.
func (r *Repo) GetConfig(key string) (string, error) {
	output, err := r.gitCmd("config", "--get", key).Output()
	if err != nil {
		return "", fmt.Errorf("failed to get git config %s: %w", key, err)
	}
	return strings.TrimSpace(string(output)), nil
}

// GetConfigLocal gets a Git config value from the repository's LOCAL config only
// (.git/config), ignoring global and system scope.
//
// Use it for repository-local state git-flow writes itself rather than for
// settings users configure: the matching write (SetConfig) and removal
// (UnsetConfigIfPresent) are both local-scoped, so a merged read would report a
// global or system value that those two can never manage — see
// UnsetConfigIfPresent for the same asymmetry stated from the write side.
func (r *Repo) GetConfigLocal(key string) (string, error) {
	output, err := r.gitCmd("config", "--local", "--get", key).Output()
	if err != nil {
		return "", fmt.Errorf("failed to get local git config %s: %w", key, err)
	}
	return strings.TrimSpace(string(output)), nil
}

// GetConfigLocalIfSet reads a SINGLE-VALUED key from LOCAL config only,
// distinguishing an unset key from a failed read: `git config --local --get-all`
// exits 1 when the key is absent, and any other status (128 for unreadable
// config) is a real failure.
//
// GetConfigLocal cannot stand in for this, because it collapses both into an
// error. A caller that mirrors one key onto another (MoveConfigLocal) would then
// read a failed read as "source absent" and REMOVE the destination, turning a
// transient failure into data loss. readConfigValue makes the same distinction
// for MERGED scope; this is its local-scoped counterpart.
//
// A multi-value key is an ERROR, not a silent pick. `--get` would exit 0 and
// return the LAST value, so a mirroring caller would overwrite the destination
// with one arbitrarily chosen value and only then fail on the source unset —
// destroying data on the way to reporting a problem. --get-all makes the
// ambiguity visible before anything is written.
//
// Caveat: git also exits 1 for a malformed key NAME, which this reports as
// "absent" like any unset key. Every caller builds its key from constants and a
// branch name, and a ref can contain neither a space nor a newline, so a
// malformed key cannot arise here. UnsetConfigIfPresent and readConfigValue
// already conflate the two the same way.
func (r *Repo) GetConfigLocalIfSet(key string) (value string, found bool, err error) {
	output, runErr := r.gitCmd("config", "--local", "--get-all", key).Output()
	if runErr != nil {
		if exitErr, ok := runErr.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return "", false, nil
		}
		if detail := stderrOf(runErr); detail != "" {
			return "", false, fmt.Errorf("failed to get local git config %s: %s: %w", key, detail, runErr)
		}
		return "", false, fmt.Errorf("failed to get local git config %s: %w", key, runErr)
	}
	// One value per line, with a trailing newline. A single empty value is one
	// empty line, which is why the trailing newline is stripped rather than the
	// whole string trimmed.
	lines := strings.Split(strings.TrimSuffix(string(output), "\n"), "\n")
	if len(lines) > 1 {
		return "", false, fmt.Errorf("local git config %s has %d values; expected one", key, len(lines))
	}
	// Deliberately NOT trimmed: the caller mirrors this value onto another key,
	// and a padded value must survive the move byte for byte. Trimming here would
	// silently rewrite it. GetConfigLocal trims because it parses; this preserves.
	return lines[0], true, nil
}

// MoveConfigLocal makes newKey mirror oldKey in LOCAL config, then drops oldKey.
//
// Mirroring, not copying: when oldKey is absent, newKey is REMOVED rather than
// left in place, so a stale value under the destination name can never be
// silently inherited by whatever the caller is renaming.
//
// Both the read and the writes are local-scoped. A global or system value must
// never be read as the source and copied down into the repository's config.
//
// A multi-value source is refused rather than migrated: GetConfigLocalIfSet
// returns an error for one, so this returns BEFORE writing and both names are
// left exactly as they were. git-flow writes no key this primitive serves as
// multi-value, and UnsetConfigIfPresent already refuses multi-value keys for the
// same reason — refusing destroys nothing, while guessing at a merge order might.
func (r *Repo) MoveConfigLocal(oldKey string, newKey string) error {
	// Moving a key onto itself would otherwise write the value and then unset the
	// key it was just written to.
	if oldKey == newKey {
		return nil
	}

	value, found, err := r.GetConfigLocalIfSet(oldKey)
	if err != nil {
		return err
	}
	if !found {
		return r.UnsetConfigIfPresent(newKey)
	}

	// Destination first: a failure between the two writes leaves the value
	// readable under both names rather than under neither.
	if err := r.SetConfig(newKey, value); err != nil {
		return err
	}
	return r.UnsetConfigIfPresent(oldKey)
}

// HasUserIdentity reports whether both user.name and user.email are configured
// and non-empty in git's merged/effective config (local > global > system),
// matching what `git commit` would see. It returns false without error when a
// key is simply unset (git config --get exits with status 1), and propagates
// any other failure (e.g. not in a repository) to the caller.
func (r *Repo) HasUserIdentity() (bool, error) {
	for _, key := range []string{"user.name", "user.email"} {
		value, found, err := r.readConfigValue(key)
		if err != nil {
			return false, err
		}
		if !found || strings.TrimSpace(value) == "" {
			return false, nil
		}
	}
	return true, nil
}

// readConfigValue reads a single git config value from the merged/effective
// scope. It distinguishes an unset key (git config --get exits with status 1)
// from an unexpected failure: an unset key returns ("", false, nil), while any
// other error is returned to the caller.
func (r *Repo) readConfigValue(key string) (value string, found bool, err error) {
	output, runErr := r.gitCmd("config", "--get", key).Output()
	if runErr != nil {
		if exitErr, ok := runErr.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return "", false, nil
		}
		return "", false, fmt.Errorf("failed to get git config %s: %w", key, runErr)
	}
	return strings.TrimSpace(string(output)), true, nil
}

// GetConfigAllValues gets all values for a multi-value Git config key
func (r *Repo) GetConfigAllValues(key string) ([]string, error) {
	output, err := r.gitCmd("config", "--get-all", key).Output()
	if err != nil {
		// If no config values match, return empty slice (not an error)
		if strings.Contains(err.Error(), "exit status 1") {
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to get git config %s: %w", key, err)
	}

	var values []string
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if line != "" {
			values = append(values, line)
		}
	}
	return values, nil
}

// SetConfig sets a Git config value (local scope by git's default write behavior).
func (r *Repo) SetConfig(key string, value string) error {
	if _, err := r.gitCmd("config", key, value).Output(); err != nil {
		if detail := stderrOf(err); detail != "" {
			return fmt.Errorf("failed to set git config %s: %s: %w", key, detail, err)
		}
		return fmt.Errorf("failed to set git config %s: %w", key, err)
	}
	return nil
}

// UnsetConfigSection removes all Git config values matching a pattern
func (r *Repo) UnsetConfigSection(pattern string) error {
	_, err := r.gitCmd("config", "--remove-section", pattern).Output()
	if err != nil {
		// Don't treat "section not found" as an error
		if strings.Contains(err.Error(), "exit status 128") {
			return nil
		}
		return fmt.Errorf("failed to unset git config section %s: %w", pattern, err)
	}
	return nil
}

// GetAllConfig gets all Git config values matching a pattern, as a key->value map.
func (r *Repo) GetAllConfig(pattern string) (map[string]string, error) {
	output, err := r.gitCmd("config", "--get-regexp", pattern).Output()
	if err != nil {
		// If no config values match, don't treat it as an error
		if strings.Contains(err.Error(), "exit status 1") {
			return make(map[string]string), nil
		}
		return nil, fmt.Errorf("failed to get git config matching %s: %w", pattern, err)
	}

	config := make(map[string]string)
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, " ", 2)
		if len(parts) == 2 {
			config[parts[0]] = parts[1]
		}
	}

	return config, nil
}

// GetConfigRegexpLines returns the raw output lines of `git config --get-regexp
// <pattern>` for the repository, or an empty slice when nothing matches. It is
// used by the config package, which needs the raw lines (preserving empty values
// and dotted subsection names) rather than a pre-parsed map.
func (r *Repo) GetConfigRegexpLines(pattern string) ([]string, error) {
	output, err := r.gitCmd("config", "--get-regexp", pattern).Output()
	if err != nil {
		if strings.Contains(err.Error(), "exit status 1") {
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to get git config matching %s: %w", pattern, err)
	}
	return strings.Split(string(output), "\n"), nil
}

// GetConfigLocalRegexpLines returns the raw output lines of `git config --local
// --get-regexp <pattern>` for the repository's local config only (.git/config),
// or an empty slice when nothing matches. Multi-value keys yield one line per
// value, in file order. Used by the shared-config copy/status logic, which must
// distinguish local keys from merged (global/system) ones.
func (r *Repo) GetConfigLocalRegexpLines(pattern string) ([]string, error) {
	output, err := r.gitCmd("config", "--local", "--get-regexp", pattern).Output()
	if err != nil {
		// exit 1 = no matching keys (a valid, possibly empty result).
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to get local git config matching %s: %w", pattern, err)
	}
	return strings.Split(string(output), "\n"), nil
}

// AddConfig adds a value to a (possibly multi-value) Git config key in local
// scope with `git config --add`. Unlike SetConfig it never replaces existing
// values, so callers can reproduce an ordered multi-value list by unsetting then
// adding each value in order.
func (r *Repo) AddConfig(key, value string) error {
	if _, err := r.gitCmd("config", "--local", "--add", key, value).Output(); err != nil {
		return fmt.Errorf("failed to add git config %s: %w", key, err)
	}
	return nil
}

// UnsetAllConfigIfPresent removes all values of a key from local config with
// `git config --local --unset-all`, treating an absent key (exit 5) as a no-op.
// This is the multi-value-safe counterpart to UnsetConfigIfPresent.
func (r *Repo) UnsetAllConfigIfPresent(key string) error {
	_, err := r.gitCmd("config", "--local", "--unset-all", key).Output()
	if err != nil {
		// exit 5 = key does not exist in the given scope: nothing to remove.
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 5 {
			return nil
		}
		return fmt.Errorf("failed to unset-all git config %s: %w", key, err)
	}
	return nil
}

// GetConfigFileRegexpLines returns the raw output lines of `git config --file
// <filePath> --get-regexp <pattern>`, or an empty slice when nothing matches.
// A non-"no match" failure (e.g. an unparsable file) is returned as an error so
// callers can surface a clear message naming the file. Multi-value keys yield one
// line per value, in file order.
func GetConfigFileRegexpLines(filePath, pattern string) ([]string, error) {
	output, err := gitCommand("", "config", "--file", filePath, "--get-regexp", pattern).Output()
	if err != nil {
		// exit 1 = no matching keys (a valid, possibly empty result).
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to read git config from %s: %w", filePath, err)
	}
	return strings.Split(string(output), "\n"), nil
}

// GetConfigFileAllValues returns all values for a key from a file-scoped config,
// in order, or an empty slice when unset.
func GetConfigFileAllValues(filePath, key string) ([]string, error) {
	output, err := gitCommand("", "config", "--file", filePath, "--get-all", key).Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to read git config %s from %s: %w", key, filePath, err)
	}
	var values []string
	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		if line != "" {
			values = append(values, line)
		}
	}
	return values, nil
}

// UnsetConfigSectionFile removes an entire section from a file-scoped config with
// `git config --file <filePath> --remove-section <section>`, treating a missing
// section (exit 128) as a no-op.
func UnsetConfigSectionFile(filePath, section string) error {
	_, err := gitCommand("", "config", "--file", filePath, "--remove-section", section).Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 128 {
			return nil
		}
		return fmt.Errorf("failed to remove section %s from %s: %w", section, filePath, err)
	}
	return nil
}

// UnsetConfig unsets a Git config value
func (r *Repo) UnsetConfig(key string) error {
	if _, err := r.gitCmd("config", "--unset", key).Output(); err != nil {
		return fmt.Errorf("failed to unset git config %s: %w", key, err)
	}
	return nil
}

// UnsetConfigIfPresent unsets a Git config value in local config, treating a
// confirmed-absent key as a no-op. A key that is not set in local config (git
// config --local --get exits 1) returns nil silently. Any other failure —
// including a multi-value key that --unset refuses (exit 5) or a genuine
// read/write error — is surfaced as an error.
func (r *Repo) UnsetConfigIfPresent(key string) error {
	// Detect absence precisely and in the scope that gets cleaned: `git config
	// --local --get` exits 1 when the key is absent from local config. The unset
	// below is local-scoped too, so probing merged config (local+global+system)
	// would wrongly treat a global/system-only value as present and fall through
	// to a local --unset that finds nothing and errors. Do NOT match on --unset
	// exit 5, which also means "multi-value".
	err := r.gitCmd("config", "--local", "--get", key).Run()
	if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
		return nil // key not present in local config: nothing to clean up
	}
	// Present, multi-value, or read error: attempt the unset in the same local
	// scope we probed so real failures (multi-value, read-only) still surface.
	if _, err := r.gitCmd("config", "--local", "--unset", key).Output(); err != nil {
		if detail := stderrOf(err); detail != "" {
			return fmt.Errorf("failed to unset git config %s: %s: %w", key, detail, err)
		}
		return fmt.Errorf("failed to unset git config %s: %w", key, err)
	}
	return nil
}

// GetConfigWithScope gets a Git config value at a specific scope. It is
// repository-less by design: global/system/file scopes are not bound to a work
// tree, and a relative --file path resolves against the process working
// directory (the invocation directory).
// For ConfigScopeDefault, reads merged config (git's standard behavior).
func GetConfigWithScope(key string, scope ConfigScope, filePath string) (string, error) {
	args := []string{"config"}
	switch scope {
	case ConfigScopeLocal:
		args = append(args, "--local")
	case ConfigScopeGlobal:
		args = append(args, "--global")
	case ConfigScopeSystem:
		args = append(args, "--system")
	case ConfigScopeFile:
		args = append(args, "--file", filePath)
		// ConfigScopeDefault: no flag = merged config
	}
	args = append(args, "--get", key)
	output, err := gitCommand("", args...).Output()
	if err != nil {
		return "", fmt.Errorf("failed to get git config %s: %w", key, err)
	}
	return strings.TrimSpace(string(output)), nil
}

// SetConfigWithScope sets a Git config value at a specific scope.
// For ConfigScopeDefault, writes to local (git's standard behavior).
func SetConfigWithScope(key, value string, scope ConfigScope, filePath string) error {
	args := []string{"config"}
	switch scope {
	case ConfigScopeLocal:
		args = append(args, "--local")
	case ConfigScopeGlobal:
		args = append(args, "--global")
	case ConfigScopeSystem:
		args = append(args, "--system")
	case ConfigScopeFile:
		args = append(args, "--file", filePath)
		// ConfigScopeDefault: no flag = local (git's default for writes)
	}
	args = append(args, key, value)
	if _, err := gitCommand("", args...).Output(); err != nil {
		return fmt.Errorf("failed to set git config %s: %w", key, err)
	}
	return nil
}

// UnsetConfigWithScope unsets a Git config value at a specific scope.
func UnsetConfigWithScope(key string, scope ConfigScope, filePath string) error {
	args := []string{"config"}
	switch scope {
	case ConfigScopeLocal:
		args = append(args, "--local")
	case ConfigScopeGlobal:
		args = append(args, "--global")
	case ConfigScopeSystem:
		args = append(args, "--system")
	case ConfigScopeFile:
		args = append(args, "--file", filePath)
		// ConfigScopeDefault: no flag = local (git's default for writes)
	}
	args = append(args, "--unset", key)
	if _, err := gitCommand("", args...).Output(); err != nil {
		return fmt.Errorf("failed to unset git config %s: %w", key, err)
	}
	return nil
}

// BaseBranchKey returns the git config key recording the start point a topic
// branch was created from. It is the counterpart to worktree.MarkerKey: both name
// a per-branch key that has to be spelled identically wherever it is read,
// written, cleaned up or migrated.
func BaseBranchKey(branch string) string {
	return fmt.Sprintf("gitflow.branch.%s.base", branch)
}

// GetBaseBranch returns the stored base branch for a topic branch.
//
// It reads MERGED config, so it is not usable for migrating per-branch state:
// a global or system value would be copied down into local config. Use
// GetConfigLocalIfSet for that. See #237.
func (r *Repo) GetBaseBranch(branchName string) (string, error) {
	return r.GetConfig(BaseBranchKey(branchName))
}

// SetBaseBranch stores the base branch for a topic branch
func (r *Repo) SetBaseBranch(branchName, baseBranch string) error {
	return r.SetConfig(BaseBranchKey(branchName), baseBranch)
}
