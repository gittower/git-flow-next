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

// GetBaseBranch returns the stored base branch for a topic branch
func (r *Repo) GetBaseBranch(branchName string) (string, error) {
	configKey := fmt.Sprintf("gitflow.branch.%s.base", branchName)
	return r.GetConfig(configKey)
}

// SetBaseBranch stores the base branch for a topic branch
func (r *Repo) SetBaseBranch(branchName, baseBranch string) error {
	configKey := fmt.Sprintf("gitflow.branch.%s.base", branchName)
	return r.SetConfig(configKey, baseBranch)
}
