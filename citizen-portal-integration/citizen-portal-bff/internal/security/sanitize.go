// Package security implements the cross-cutting protections the WSO2 secure
// engineering guidelines require of this BFF: log-forging prevention,
// open-redirect prevention, CSRF, and constant-time comparisons.
package security

import "strings"

const (
	maxLogFieldLength = 2048
	truncationMarker  = "...(truncated)"
)

// SanitizeForLog strips CR, LF and TAB from an externally supplied string and
// caps its length before it reaches a log line or an error message returned
// to a caller. Mirrors LogSanitizer.clean() in the Java authenticator and
// clean() in esignet-bridge/server.js — every externally supplied string in
// this codebase goes through the equivalent of this function first, because
// an unsanitized value can forge log lines (CRLF injection, guideline §1.7).
func SanitizeForLog(s string) string {
	if len(s) > maxLogFieldLength {
		s = s[:maxLogFieldLength-len(truncationMarker)] + truncationMarker
	}
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch r {
		case '\r', '\n', '\t':
			continue
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}
