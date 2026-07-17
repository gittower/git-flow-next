package util_test

import (
	"testing"

	"github.com/gittower/git-flow-next/internal/util"
)

// TestIsValidBranchName checks that branch-name validation matches Git's own
// reference-name rules, including accepting dots inside a name (issue #93)
// while still rejecting the names Git refuses.
func TestIsValidBranchName(t *testing.T) {
	cases := []struct {
		name  string
		valid bool
	}{
		// Dotted names are valid (the #93 fix).
		{"custom.main", true},
		{"custom.dev", true},
		{"release.2.0", true},
		{"V10.5", true},
		{"foo.main.dev", true},
		{"custom.lock.x", true}, // ".lock" only matters as a suffix
		// Plain names remain valid.
		{"main", true},
		{"develop", true},
		{"feature/foo", true},
		{"v9_release", true},
		// Names Git rejects stay invalid.
		{"", false},
		{"custom..main", false}, // consecutive dots
		{".custom", false},      // component begins with a dot
		{"foo.lock", false},     // trailing .lock
		{"foo/", false},         // trailing slash
		{"with space", false},   // whitespace
		{"foo^bar", false},      // special character
		{"foo~bar", false},      // special character
		{"-foo", false},         // leading dash: option-like, unsafe as a git operand
		{"-f", false},           // leading dash
	}

	for _, c := range cases {
		if got := util.IsValidBranchName(c.name); got != c.valid {
			t.Errorf("IsValidBranchName(%q) = %v, want %v", c.name, got, c.valid)
		}
	}
}

// TestValidateBranchName checks the error-returning wrapper accepts dotted
// names and rejects empty and malformed ones.
func TestValidateBranchName(t *testing.T) {
	if err := util.ValidateBranchName("custom.main"); err != nil {
		t.Errorf("ValidateBranchName(\"custom.main\") returned error: %v", err)
	}
	if err := util.ValidateBranchName(""); err == nil {
		t.Error("ValidateBranchName(\"\") expected an error, got nil")
	}
	if err := util.ValidateBranchName("custom..main"); err == nil {
		t.Error("ValidateBranchName(\"custom..main\") expected an error, got nil")
	}
}

// TestIsValidPrefix checks prefix validation, including dotted prefixes.
func TestIsValidPrefix(t *testing.T) {
	cases := []struct {
		prefix string
		valid  bool
	}{
		{"feature/", true},
		{"qa.release/", true}, // dotted prefix is valid
		{"feature", false},    // must end with a slash
		{"foo..bar/", false},  // invalid branch name under the slash
	}

	for _, c := range cases {
		if got := util.IsValidPrefix(c.prefix); got != c.valid {
			t.Errorf("IsValidPrefix(%q) = %v, want %v", c.prefix, got, c.valid)
		}
	}
}
