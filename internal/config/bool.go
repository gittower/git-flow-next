package config

import (
	"strconv"
	"strings"
)

// ParseBool interprets a raw git-config value using git-config(1) boolean
// semantics: "true"/"yes"/"on" and any non-zero integer are true; "false"/"no"/
// "off"/"0" and the empty string are false; matching is case-insensitive.
// Unlike git-config(1), which errors on an unparseable value, an unrecognized
// value resolves to false — the resolvers have no channel to surface the error.
//
// The value is not trimmed: git rejects a whitespace-padded boolean as a bad
// value, and strconv.Atoi rejects padded integers, so padded input follows the
// unrecognized-value rule.
func ParseBool(value string) bool {
	switch strings.ToLower(value) {
	case "true", "yes", "on":
		return true
	case "false", "no", "off", "0", "":
		return false
	}

	// Any integer is a valid git-config boolean: zero is false, non-zero true.
	n, err := strconv.Atoi(value)
	if err != nil {
		return false
	}
	return n != 0
}
