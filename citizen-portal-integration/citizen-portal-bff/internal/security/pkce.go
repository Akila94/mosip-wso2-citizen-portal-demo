package security

import (
	"crypto/sha256"
	"encoding/base64"
)

// pkceVerifierBytes yields a base64url string of length 43 once encoded
// (32 raw bytes -> 43 base64url chars with no padding), the minimum allowed
// under RFC 7636 and comfortably within its [43,128] range.
const pkceVerifierBytes = 32

// GenerateVerifier returns a cryptographically random PKCE code verifier
// (RFC 7636 §4.1).
func GenerateVerifier() (string, error) {
	return RandomToken(pkceVerifierBytes)
}

// Challenge computes the S256 PKCE code challenge for a verifier (RFC 7636
// §4.2): BASE64URL-ENCODE(SHA256(ASCII(verifier))), no padding. Plain-method
// PKCE is never used in this codebase.
func Challenge(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}
