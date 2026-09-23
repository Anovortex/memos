package v1

import "strings"

// normalizeEmail trims and lowercases an address so lookups are
// case-insensitive. The empty string stays empty. Persisted addresses go
// through store.NormalizeEmail, which also validates; this is the lookup form.
func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
