package web

import (
	"strings"
)

// Sluggify converts a string to a URL/filesystem-safe slug.
// Truncates to 100 chars, lowercases, replaces spaces/underscores with hyphens,
// and strips non-alphanumeric characters (except hyphens).
func Sluggify(s string) string {
	s = strings.ToLower(s)
	var result strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			result.WriteRune(r)
		} else if r == ' ' || r == '_' {
			result.WriteRune('-')
		}
	}
	out := result.String()
	if len(out) > 100 {
		out = out[:100]
	}
	return strings.Trim(out, "-")
}
