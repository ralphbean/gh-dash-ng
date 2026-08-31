package config

import (
	"path"
	"strings"
)

// IsCheckIgnored reports whether identity matches an ignored-check glob.
// Patterns have already been validated while loading configuration.
func IsCheckIgnored(patterns []string, identity string) bool {
	// path.Match gives us familiar glob syntax, but treats '/' specially.
	// Checks are slash-delimited identities and users expect '*' to span the
	// whole visible value, so replace separators before matching.
	const separator = "\x00"
	identity = strings.ReplaceAll(identity, "/", separator)
	for _, pattern := range patterns {
		pattern = strings.ReplaceAll(pattern, "/", separator)
		matched, _ := path.Match(pattern, identity)
		if matched {
			return true
		}
	}
	return false
}
