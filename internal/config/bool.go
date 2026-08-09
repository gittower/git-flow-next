package config

// ParseBool interprets a raw git-config value using git-config(1) boolean
// semantics: "true"/"yes"/"on" and any non-zero integer are true; "false"/"no"/
// "off"/"0" and the empty string are false; matching is case-insensitive.
// Unlike git-config(1), which errors on an unparseable value, an unrecognized
// value resolves to false — the resolvers have no channel to surface the error.
func ParseBool(value string) bool {
	// Placeholder preserving the pre-fix behavior so the new tests fail on
	// behavior rather than on a missing symbol. Replaced by the real
	// implementation in the follow-up commit.
	return value == "true"
}
