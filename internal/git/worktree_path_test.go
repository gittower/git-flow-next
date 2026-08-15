package git

import (
	"path/filepath"
	"testing"
)

// setWindows overrides the isWindows platform seam for the duration of a test
// and returns a function that restores the previous value. This lets tests
// exercise the Windows-specific branches on any host (CI runs Linux only), and
// mirrors the same seam in internal/hooks.
func setWindows(v bool) func() {
	old := isWindows
	isWindows = v
	return func() { isWindows = old }
}

// abs builds an absolute path under a root that does not exist on any host, so
// resolvePath cannot reach a real directory entry and hand back the casing the
// filesystem records. That keeps these tests measuring the comparison itself
// rather than the case sensitivity of the volume the tests happen to run on —
// which matters on macOS, whose default volume is already case-insensitive.
func abs(elem ...string) string {
	return filepath.Join(append([]string{string(filepath.Separator), "git-flow-case-test"}, elem...)...)
}

// TestSamePathFoldsCaseOnWindows tests that two spellings of one location
// differing only in case denote the same path on Windows.
// Steps:
//  1. Builds two absolute paths that differ only in case, under a root that
//     exists on no host
//  2. Forces the isWindows seam on and calls SamePath
//  3. Verifies the paths compare equal, as Windows' case-insensitive path
//     names require
func TestSamePathFoldsCaseOnWindows(t *testing.T) {
	defer setWindows(true)()

	upper := abs("Users", "Alex", "WT")
	lower := abs("users", "alex", "wt")

	if !SamePath(upper, lower) {
		t.Errorf("SamePath(%q, %q) = false on Windows, want true", upper, lower)
	}
}

// TestSamePathKeepsCaseSensitivityOnUnix tests that the same two spellings stay
// distinct off Windows. This is the negative control for
// TestSamePathFoldsCaseOnWindows: without it a SamePath that folded
// unconditionally would pass that test for the wrong reason.
// Steps:
//  1. Builds the same two case-differing absolute paths
//  2. Forces the isWindows seam off and calls SamePath
//  3. Verifies the paths compare unequal, as Linux and a case-sensitive macOS
//     volume both require
func TestSamePathKeepsCaseSensitivityOnUnix(t *testing.T) {
	defer setWindows(false)()

	upper := abs("Users", "Alex", "WT")
	lower := abs("users", "alex", "wt")

	if SamePath(upper, lower) {
		t.Errorf("SamePath(%q, %q) = true on Unix, want false", upper, lower)
	}
}

// TestSamePathFoldsToUpperCase tests that the fold matches the direction
// Windows folds in, which is upper case: NTFS and exFAT carry an upcase table
// and ordinal case-insensitive comparison upper-cases both sides.
// Steps:
//  1. Builds two paths differing only in Greek small sigma versus final sigma,
//     whose lowercase forms stay distinct but whose uppercase form is a shared
//     capital sigma
//  2. Forces the isWindows seam on and calls SamePath
//  3. Verifies the paths compare equal, which holds for an upper-case fold and
//     fails for a lower-case one
func TestSamePathFoldsToUpperCase(t *testing.T) {
	defer setWindows(true)()

	sigma := abs("repo", "σ")      // σ GREEK SMALL LETTER SIGMA
	finalSigma := abs("repo", "ς") // ς GREEK SMALL LETTER FINAL SIGMA

	if !SamePath(sigma, finalSigma) {
		t.Errorf("SamePath(%q, %q) = false on Windows, want true: both upper-case to capital sigma, "+
			"so Windows denotes one location", sigma, finalSigma)
	}
}

// TestSamePathMatchesIdenticalPaths tests that folding does not disturb the
// ordinary case of a path compared against itself.
// Steps:
//  1. Builds one absolute path
//  2. Compares it against itself with the isWindows seam forced on, then off
//  3. Verifies it matches itself on both platforms
func TestSamePathMatchesIdenticalPaths(t *testing.T) {
	path := abs("Users", "Alex", "WT")

	restore := setWindows(true)
	if !SamePath(path, path) {
		t.Errorf("SamePath(%q, %q) = false on Windows, want true", path, path)
	}
	restore()

	defer setWindows(false)()
	if !SamePath(path, path) {
		t.Errorf("SamePath(%q, %q) = false on Unix, want true", path, path)
	}
}

// TestSamePathRejectsDifferentPathsUnderFolding tests that folding widens only
// what counts as the same casing, never what counts as the same location.
// Steps:
//  1. Builds two paths that differ in their final element, not merely in case
//  2. Forces the isWindows seam on and calls SamePath
//  3. Verifies they compare unequal
func TestSamePathRejectsDifferentPathsUnderFolding(t *testing.T) {
	defer setWindows(true)()

	path := abs("Users", "Alex", "WT")
	other := abs("Users", "Alex", "different")

	if SamePath(path, other) {
		t.Errorf("SamePath(%q, %q) = true on Windows, want false", path, other)
	}
}

// TestIsWithinFoldsCaseOnWindows tests that case folding reaches both
// comparisons IsWithin makes: the equality test for a child that IS the parent,
// and the prefix test for a child nested underneath it.
// Steps:
//  1. Builds a parent path, a differently cased spelling of that same parent,
//     and a differently cased path nested under it
//  2. Forces the isWindows seam on and calls IsWithin for each
//  3. Verifies the equal-but-differently-cased path is within the parent
//  4. Verifies the nested differently-cased path is within the parent
func TestIsWithinFoldsCaseOnWindows(t *testing.T) {
	defer setWindows(true)()

	parent := abs("Users", "Alex", "WT")
	sameCased := abs("users", "alex", "wt")
	nested := abs("users", "alex", "wt", "src", "pkg")

	if !IsWithin(sameCased, parent) {
		t.Errorf("IsWithin(%q, %q) = false on Windows, want true (equality test)", sameCased, parent)
	}
	if !IsWithin(nested, parent) {
		t.Errorf("IsWithin(%q, %q) = false on Windows, want true (prefix test)", nested, parent)
	}
}

// TestIsWithinKeepsCaseSensitivityOnUnix tests that neither of IsWithin's
// comparisons folds off Windows. This is the negative control for
// TestIsWithinFoldsCaseOnWindows.
// Steps:
//  1. Builds the same parent, differently cased parent, and nested paths
//  2. Forces the isWindows seam off and calls IsWithin for each
//  3. Verifies neither is reported as within the parent
func TestIsWithinKeepsCaseSensitivityOnUnix(t *testing.T) {
	defer setWindows(false)()

	parent := abs("Users", "Alex", "WT")
	sameCased := abs("users", "alex", "wt")
	nested := abs("users", "alex", "wt", "src", "pkg")

	if IsWithin(sameCased, parent) {
		t.Errorf("IsWithin(%q, %q) = true on Unix, want false", sameCased, parent)
	}
	if IsWithin(nested, parent) {
		t.Errorf("IsWithin(%q, %q) = true on Unix, want false", nested, parent)
	}
}

// TestIsWithinSeparatorBoundaryHoldsUnderFolding tests that folding does not
// weaken the separator-boundary check that stops a sibling sharing a name
// prefix from being treated as nested. Folding must make more pairs comparable
// without making more pairs "within".
// Steps:
//  1. Builds a parent and a differently cased sibling whose name extends the
//     parent's final element
//  2. Forces the isWindows seam on and verifies the sibling is NOT within the
//     parent, so the boundary still holds once case stops distinguishing them
//  3. Builds a genuine child of that same differently cased parent and verifies
//     it IS within, so step 2 cannot pass merely by folding being absent
func TestIsWithinSeparatorBoundaryHoldsUnderFolding(t *testing.T) {
	defer setWindows(true)()

	parent := abs("a", "foo")
	sibling := abs("A", "FooBar")
	if IsWithin(sibling, parent) {
		t.Errorf("IsWithin(%q, %q) = true on Windows, want false: %q is a sibling of %q, not inside it",
			sibling, parent, sibling, parent)
	}

	child := abs("A", "FOO", "bar")
	if !IsWithin(child, parent) {
		t.Errorf("IsWithin(%q, %q) = false on Windows, want true", child, parent)
	}
}
