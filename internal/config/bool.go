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
// The integer case is bounded by strconv.Atoi: an integer outside the range
// it can parse is not recognized and follows the unrecognized-value rule
// above. git-config(1) rejects out-of-range integers as well, and its own
// boolean range is narrower still (32-bit int), so the limit is unobservable
// on values git accepts as booleans.
//
// The value is deliberately not trimmed: a whitespace-padded value matches no
// spelling and strconv.Atoi rejects it, so it follows the unrecognized-value
// rule above. Whether padded input can reach here at all differs by read path
// — repo.GetConfig trims, while the resolver and branch-property paths pass
// the raw value through.
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
