package config_test

import (
	"testing"

	"github.com/gittower/git-flow-next/internal/config"
)

// TestParseBoolTruthyLiterals tests that git-config(1)'s truthy literals all parse as true.
// Steps:
// 1. Calls config.ParseBool with "true", "yes", "on" and "1"
// 2. Verifies every input resolves to true
func TestParseBoolTruthyLiterals(t *testing.T) {
	t.Parallel()

	for _, value := range []string{"true", "yes", "on", "1"} {
		if !config.ParseBool(value) {
			t.Errorf("ParseBool(%q) = false, want true", value)
		}
	}
}

// TestParseBoolFalsyLiterals tests that git-config(1)'s falsy literals all parse as false.
// Steps:
// 1. Calls config.ParseBool with "false", "no", "off" and "0"
// 2. Verifies every input resolves to false
func TestParseBoolFalsyLiterals(t *testing.T) {
	t.Parallel()

	for _, value := range []string{"false", "no", "off", "0"} {
		if config.ParseBool(value) {
			t.Errorf("ParseBool(%q) = true, want false", value)
		}
	}
}

// TestParseBoolEmptyString tests that an empty value resolves to false.
// Steps:
// 1. Calls config.ParseBool with ""
// 2. Verifies the result is false
func TestParseBoolEmptyString(t *testing.T) {
	t.Parallel()

	if config.ParseBool("") {
		t.Error("ParseBool(\"\") = true, want false")
	}
}

// TestParseBoolNonZeroIntegers tests that any non-zero integer of either sign is truthy.
// Steps:
// 1. Calls config.ParseBool with "5", "42" and "-1"
// 2. Verifies every input resolves to true
func TestParseBoolNonZeroIntegers(t *testing.T) {
	t.Parallel()

	for _, value := range []string{"5", "42", "-1"} {
		if !config.ParseBool(value) {
			t.Errorf("ParseBool(%q) = false, want true", value)
		}
	}
}

// TestParseBoolCaseInsensitive tests that boolean literals match case-insensitively.
// Steps:
// 1. Calls config.ParseBool with "Yes", "ON", "True" and "Off"
// 2. Verifies the first three resolve to true and "Off" resolves to false
func TestParseBoolCaseInsensitive(t *testing.T) {
	t.Parallel()

	cases := []struct {
		value string
		want  bool
	}{
		{"Yes", true},
		{"ON", true},
		{"True", true},
		{"Off", false},
	}

	for _, tc := range cases {
		if got := config.ParseBool(tc.value); got != tc.want {
			t.Errorf("ParseBool(%q) = %v, want %v", tc.value, got, tc.want)
		}
	}
}

// TestParseBoolUnrecognizedValue tests that an unparseable value degrades to false.
// Steps:
//  1. Calls config.ParseBool with "maybe" and "truue"
//  2. Verifies both resolve to false without panicking (there is no error return,
//     so an unrecognized value is treated as "off")
func TestParseBoolUnrecognizedValue(t *testing.T) {
	t.Parallel()

	for _, value := range []string{"maybe", "truue"} {
		if config.ParseBool(value) {
			t.Errorf("ParseBool(%q) = true, want false", value)
		}
	}
}

// TestParseBoolWhitespacePaddedValue tests that ParseBool itself does not trim its input.
// A padded value matches no spelling and strconv.Atoi rejects it, so the helper's
// contract is to treat it as unrecognized. This is a statement about the helper
// only: whether padded input can reach it differs by read path — repo.GetConfig
// trims, while the resolver and branch-property paths pass the raw value through.
// Steps:
// 1. Calls config.ParseBool with " yes", "true " and " 1"
// 2. Verifies all three resolve to false
func TestParseBoolWhitespacePaddedValue(t *testing.T) {
	t.Parallel()

	for _, value := range []string{" yes", "true ", " 1"} {
		if config.ParseBool(value) {
			t.Errorf("ParseBool(%q) = true, want false", value)
		}
	}
}

// TestParseBoolSignedAndPaddedIntegers tests signed and zero-padded integer forms.
// Steps:
//  1. Calls config.ParseBool with "+5", "007" and "-0"
//  2. Verifies "+5" and "007" resolve to true and "-0" resolves to false (it
//     parses to the integer 0)
func TestParseBoolSignedAndPaddedIntegers(t *testing.T) {
	t.Parallel()

	cases := []struct {
		value string
		want  bool
	}{
		{"+5", true},
		{"007", true},
		{"-0", false},
	}

	for _, tc := range cases {
		if got := config.ParseBool(tc.value); got != tc.want {
			t.Errorf("ParseBool(%q) = %v, want %v", tc.value, got, tc.want)
		}
	}
}

// TestParseBoolIntegerOverflow tests that an integer too large for int degrades to false.
// Steps:
// 1. Calls config.ParseBool with "99999999999999999999"
// 2. Verifies the result is false (unparseable, so the unrecognized-value rule applies)
func TestParseBoolIntegerOverflow(t *testing.T) {
	t.Parallel()

	if config.ParseBool("99999999999999999999") {
		t.Error("ParseBool(\"99999999999999999999\") = true, want false")
	}
}
