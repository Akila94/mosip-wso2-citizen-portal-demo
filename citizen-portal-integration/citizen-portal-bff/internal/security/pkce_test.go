package security

import (
	"crypto/sha256"
	"encoding/base64"
	"testing"
)

func TestGenerateVerifierLength(t *testing.T) {
	v, err := GenerateVerifier()
	if err != nil {
		t.Fatalf("GenerateVerifier: %v", err)
	}
	// RFC 7636: 43-128 characters from [A-Za-z0-9-._~].
	if len(v) < 43 || len(v) > 128 {
		t.Fatalf("verifier length %d out of RFC 7636 range [43,128]", len(v))
	}
}

func TestChallengeIsS256OfVerifier(t *testing.T) {
	v, err := GenerateVerifier()
	if err != nil {
		t.Fatalf("GenerateVerifier: %v", err)
	}
	got := Challenge(v)

	sum := sha256.Sum256([]byte(v))
	want := base64.RawURLEncoding.EncodeToString(sum[:])
	if got != want {
		t.Errorf("Challenge(%q) = %q, want %q", v, got, want)
	}
}

func TestChallengeIsDeterministic(t *testing.T) {
	v := "fixed-verifier-value-for-determinism-check-1234567890"
	if Challenge(v) != Challenge(v) {
		t.Error("Challenge is not deterministic for the same verifier")
	}
}
