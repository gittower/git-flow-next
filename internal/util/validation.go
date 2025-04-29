package util

import (
	"regexp"
	"strings"
)

// IsValidBranchName checks if a branch name is valid
func IsValidBranchName(name string) bool {
	// Git branch names cannot:
	// - Have a path component that begins with "."
	// - Have a double dot ".."
	// - Have a character that is not alphanumeric, underscore, or dash
	// - End with a "/"
	// - End with ".lock"
	// - Contain a space " "

	if strings.Contains(name, "..") {
		return false
	}

	if strings.HasSuffix(name, "/") {
		return false
	}

	if strings.HasSuffix(name, ".lock") {
		return false
	}

	if strings.Contains(name, " ") {
		return false
	}

	// Check for path components starting with "."
	parts := strings.Split(name, "/")
	for _, part := range parts {
		if strings.HasPrefix(part, ".") {
			return false
		}
	}

	// Check for invalid characters
	validChars := regexp.MustCompile(`^[a-zA-Z0-9_\-/]+$`)
	return validChars.MatchString(name)
}

// IsValidPrefix checks if a prefix is valid
func IsValidPrefix(prefix string) bool {
	// A prefix should end with a "/"
	if !strings.HasSuffix(prefix, "/") {
		return false
	}

	// Remove the trailing "/" and check if it's a valid branch name
	return IsValidBranchName(strings.TrimSuffix(prefix, "/"))
}
