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

// TestSamePathFoldsCaseOnWindows verifies that two spellings of one location
// differing only in case denote the same path on Windows, where path names are
// case-insensitive, and different paths elsewhere.
func TestSamePathFoldsCaseOnWindows(t *testing.T) {
	upper := abs("Users", "Alex", "WT")
	lower := abs("users", "alex", "wt")

	defer setWindows(true)()
	if !SamePath(upper, lower) {
		t.Errorf("SamePath(%q, %q) = false on Windows, want true", upper, lower)
	}

	// Negative control: the two branches must genuinely differ, or the test
	// above would pass for the wrong reason.
	restore := setWindows(false)
	defer restore()
	if SamePath(upper, lower) {
		t.Errorf("SamePath(%q, %q) = true on Unix, want false", upper, lower)
	}
}

// TestSamePathUnaffectedByFoldingWhenCasingMatches verifies that folding does
// not disturb the ordinary case: identical paths match on both platforms, and
// genuinely different paths match on neither.
func TestSamePathUnaffectedByFoldingWhenCasingMatches(t *testing.T) {
	for _, windows := range []bool{true, false} {
		restore := setWindows(windows)

		same := abs("Users", "Alex", "WT")
		if !SamePath(same, same) {
			t.Errorf("SamePath(%q, %q) = false with isWindows=%v, want true", same, same, windows)
		}

		other := abs("Users", "Alex", "different")
		if SamePath(same, other) {
			t.Errorf("SamePath(%q, %q) = true with isWindows=%v, want false", same, other, windows)
		}

		restore()
	}
}

// TestIsWithinFoldsCaseOnWindows verifies that case folding applies to both
// comparisons IsWithin makes: the equality test for a child that IS the parent,
// and the prefix test for a child nested underneath it.
func TestIsWithinFoldsCaseOnWindows(t *testing.T) {
	parent := abs("Users", "Alex", "WT")
	sameCased := abs("users", "alex", "wt")
	nested := abs("users", "alex", "wt", "src", "pkg")

	defer setWindows(true)()
	if !IsWithin(sameCased, parent) {
		t.Errorf("IsWithin(%q, %q) = false on Windows, want true (equality test)", sameCased, parent)
	}
	if !IsWithin(nested, parent) {
		t.Errorf("IsWithin(%q, %q) = false on Windows, want true (prefix test)", nested, parent)
	}

	// Negative control for both comparisons.
	restore := setWindows(false)
	defer restore()
	if IsWithin(sameCased, parent) {
		t.Errorf("IsWithin(%q, %q) = true on Unix, want false", sameCased, parent)
	}
	if IsWithin(nested, parent) {
		t.Errorf("IsWithin(%q, %q) = true on Unix, want false", nested, parent)
	}
}

// TestIsWithinSeparatorBoundaryHoldsUnderFolding verifies that folding does not
// weaken the separator-boundary check that stops a sibling sharing a name prefix
// from being treated as nested. Case folding makes more pairs comparable; it
// must not make more pairs "within".
func TestIsWithinSeparatorBoundaryHoldsUnderFolding(t *testing.T) {
	defer setWindows(true)()

	parent := abs("a", "foo")
	sibling := abs("A", "FooBar")
	if IsWithin(sibling, parent) {
		t.Errorf("IsWithin(%q, %q) = true on Windows, want false: %q is a sibling of %q, not inside it",
			sibling, parent, sibling, parent)
	}

	// The genuine child of that same differently-cased parent still is within.
	child := abs("A", "FOO", "bar")
	if !IsWithin(child, parent) {
		t.Errorf("IsWithin(%q, %q) = false on Windows, want true", child, parent)
	}
}
