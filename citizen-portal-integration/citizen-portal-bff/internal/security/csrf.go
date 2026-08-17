package security

import "crypto/subtle"

// ConstantTimeEqual compares two secret-bearing strings (CSRF tokens,
// session identifiers) in constant time, and treats an empty value as never
// matching — an empty cookie or header must never authorize a request.
func ConstantTimeEqual(a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	if len(a) != len(b) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
