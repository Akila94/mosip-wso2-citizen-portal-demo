// Package security implements the log-forging protection this service's
// resource-server role requires: every externally supplied string that
// reaches a log line or an error response (a claim value, a header value)
// must be sanitized first, per the WSO2 Secure Engineering Guidelines this
// whole project follows (guideline §1.7, CRLF/log injection).
package security

import "strings"

const (
	maxLogFieldLength = 2048
	truncationMarker  = "...(truncated)"
)

// SanitizeForLog strips CR, LF and TAB from an externally supplied string and
// caps its length before it reaches a log line or an error message returned
// to a caller. Functionally identical to citizen-portal-bff's
// internal/security.SanitizeForLog (that package cannot be imported across
// module boundaries), so both services behave identically for the same
// input.
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
