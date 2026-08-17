package security

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
)

// ErrNonPositiveLength is returned by RandomToken for n <= 0.
var ErrNonPositiveLength = errors.New("security: token byte length must be positive")

// RandomToken returns a base64url (no padding), crypto/rand-backed random
// token built from n raw bytes. Used for session IDs, CSRF tokens and OAuth
// "state" values — guideline §1.25 requires a CSPRNG, never math/rand.
func RandomToken(n int) (string, error) {
	if n <= 0 {
		return "", ErrNonPositiveLength
	}
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
