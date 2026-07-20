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

// GetConfig gets a Git config value
func GetConfig(key string) (string, error) {
	cmd := exec.Command("git", "config", "--get", key)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get git config %s: %w", key, err)
	}
	return strings.TrimSpace(string(output)), nil
}

// GetConfigAllValues gets all values for a multi-value Git config key
func GetConfigAllValues(key string) ([]string, error) {
	return GetConfigAllValuesInDir("", key)
}

// GetConfigAllValuesInDir gets all values for a multi-value Git config key in the specified directory
func GetConfigAllValuesInDir(dir, key string) ([]string, error) {
	cmd := exec.Command("git", "config", "--get-all", key)
	if dir != "" {
		cmd.Dir = dir
	}
	output, err := cmd.Output()
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

// GetConfigInDir gets a Git config value in the specified directory
func GetConfigInDir(dir, key string) (string, error) {
	cmd := exec.Command("git", "config", "--get", key)
	cmd.Dir = dir
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get git config %s in dir %s: %w", key, dir, err)
	}
	return strings.TrimSpace(string(output)), nil
}

// SetConfig sets a Git config value
func SetConfig(key string, value string) error {
	cmd := exec.Command("git", "config", key, value)
	_, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to set git config %s: %w", key, err)
	}
	return nil
}

// UnsetConfigSection removes all Git config values matching a pattern
func UnsetConfigSection(pattern string) error {
	cmd := exec.Command("git", "config", "--remove-section", pattern)
	_, err := cmd.Output()
	if err != nil {
		// Don't treat "section not found" as an error
		if strings.Contains(err.Error(), "exit status 128") {
			return nil
		}
		return fmt.Errorf("failed to unset git config section %s: %w", pattern, err)
	}
	return nil
}

// GetAllConfig gets all Git config values matching a pattern
func GetAllConfig(pattern string) (map[string]string, error) {
	cmd := exec.Command("git", "config", "--get-regexp", pattern)
	output, err := cmd.Output()
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

// UnsetConfig unsets a Git config value
func UnsetConfig(key string) error {
	cmd := exec.Command("git", "config", "--unset", key)
	_, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to unset git config %s: %w", key, err)
	}
	return nil
}

// UnsetConfigIfPresent unsets a Git config value, treating a confirmed-absent
// key as a no-op. A key that is not set (git config --get exits 1) returns nil
// silently. Any other failure — including a multi-value key that plain --unset
// refuses (exit 5) or a genuine read/write error — is surfaced as an error.
func UnsetConfigIfPresent(key string) error {
	// Detect absence precisely: `git config --get` exits 1 when the key is
	// absent. Do NOT match on --unset exit 5, which also means "multi-value".
	err := exec.Command("git", "config", "--get", key).Run()
	if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
		return nil // key not present: nothing to clean up
	}
	// Present, multi-value, or read error: attempt the unset so real
	// failures (multi-value, read-only) still surface to the caller.
	return UnsetConfig(key)
}

// GetConfigWithScope gets a Git config value at a specific scope.
// For ConfigScopeDefault, reads merged config (git's standard behavior).
// For specific scopes, reads only from that scope.
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
	cmd := exec.Command("git", args...)
	output, err := cmd.Output()
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
	cmd := exec.Command("git", args...)
	_, err := cmd.Output()
	if err != nil {
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
	cmd := exec.Command("git", args...)
	_, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to unset git config %s: %w", key, err)
	}
	return nil
}

// GetBaseBranch returns the stored base branch for a topic branch
func GetBaseBranch(branchName string) (string, error) {
	configKey := fmt.Sprintf("gitflow.branch.%s.base", branchName)
	return GetConfig(configKey)
}

// SetBaseBranch stores the base branch for a topic branch
func SetBaseBranch(branchName, baseBranch string) error {
	configKey := fmt.Sprintf("gitflow.branch.%s.base", branchName)
	return SetConfig(configKey, baseBranch)
}
