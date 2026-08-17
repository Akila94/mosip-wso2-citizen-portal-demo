package security

import (
	"errors"
	"regexp"
	"strings"
)

// returnToPattern accepts only an absolute in-app path: a single leading
// slash, followed by unreserved/path characters. It rejects a second leading
// slash (protocol-relative URL), a scheme, backslashes (some browsers treat
// "/\" as "//" during URL normalisation), and any control character.
var returnToPattern = regexp.MustCompile(`^/[A-Za-z0-9._~!$&'()*+,;=:@%/?-]*$`)

var (
	// ErrEmptyReturnTo is returned for an empty or missing returnTo value.
	ErrEmptyReturnTo = errors.New("returnTo must not be empty")
	// ErrInvalidReturnTo is returned when returnTo is not a same-app, absolute
	// in-app path.
	ErrInvalidReturnTo = errors.New("returnTo must be an absolute path within this app")
)

// ValidateReturnTo enforces guideline §1.22 (unvalidated redirects): a
// returnTo value must be an absolute path (never an absolute URL, never
// protocol-relative), must contain no path traversal, and must fall within
// the calling app's own route prefix. appPrefix "/" matches any absolute
// path. On success it returns the value unchanged; on failure the caller
// must fall back to a fixed, safe default rather than surface the error
// value's raw content.
func ValidateReturnTo(raw string, appPrefix string) (string, error) {
	if raw == "" {
		return "", ErrEmptyReturnTo
	}
	if strings.ContainsAny(raw, "\r\n\t") {
		return "", ErrInvalidReturnTo
	}
	if strings.HasPrefix(raw, "//") || strings.HasPrefix(raw, "/\\") {
		return "", ErrInvalidReturnTo
	}
	if !returnToPattern.MatchString(raw) {
		return "", ErrInvalidReturnTo
	}
	// Reject path traversal outright — a leading "/.." or an embedded "/../"
	// segment, checked on the path only (strip a query string first).
	path := raw
	if i := strings.IndexByte(path, '?'); i >= 0 {
		path = path[:i]
	}
	for _, seg := range strings.Split(path, "/") {
		if seg == ".." {
			return "", ErrInvalidReturnTo
		}
	}

	if appPrefix == "/" || appPrefix == "" {
		return raw, nil
	}
	if path == appPrefix {
		return raw, nil
	}
	if strings.HasPrefix(path, appPrefix+"/") {
		return raw, nil
	}
	return "", ErrInvalidReturnTo
}
