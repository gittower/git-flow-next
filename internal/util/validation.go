package util

import (
	"fmt"
	"os/exec"
	"strings"
)

// IsValidBranchName reports whether name is a valid git-flow branch name.
//
// Validation delegates to `git check-ref-format`, so accepted names follow
// Git's own reference-name rules — notably, dots are allowed inside a name
// (e.g. "custom.main", "V10.5"). git-flow adds one restriction beyond Git: a
// name may not begin with "-" (see below).
func IsValidBranchName(name string) bool {
	if name == "" {
		return false
	}
	// git-flow creates and renames branches with commands like
	// `git checkout -b <name>` and `git branch -m <old> <new>` that pass the
	// name as a bare operand (no "--" terminator), so a name beginning with
	// "-" would be misparsed as an option and could leave config and refs
	// inconsistent. Reject it even though Git itself accepts such a refname.
	if strings.HasPrefix(name, "-") {
		return false
	}
	// The name must form a valid refname under refs/heads/. check-ref-format
	// needs no repository and exits non-zero for an invalid name.
	return exec.Command("git", "check-ref-format", "refs/heads/"+name).Run() == nil
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

// ValidateBranchName validates a branch name and returns an error if invalid
func ValidateBranchName(name string) error {
	if name == "" {
		return fmt.Errorf("branch name cannot be empty")
	}

	if !IsValidBranchName(name) {
		return fmt.Errorf("invalid branch name: %s", name)
	}

	return nil
}
