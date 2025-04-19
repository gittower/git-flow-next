package git

import (
	"fmt"
	"os/exec"
	"strings"
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
