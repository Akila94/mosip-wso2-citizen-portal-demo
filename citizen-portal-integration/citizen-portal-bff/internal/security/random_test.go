package security

import (
	"strings"
	"testing"
)

func TestRandomTokenLengthAndCharset(t *testing.T) {
	tok, err := RandomToken(32)
	if err != nil {
		t.Fatalf("RandomToken: %v", err)
	}
	if len(tok) == 0 {
		t.Fatal("RandomToken returned empty string")
	}
	// base64url, no padding: only these characters are legal.
	const allowed = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"
	for _, r := range tok {
		if !strings.ContainsRune(allowed, r) {
			t.Fatalf("RandomToken produced disallowed character %q in %q", r, tok)
		}
	}
}

func TestRandomTokenIsUnique(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 1000; i++ {
		tok, err := RandomToken(32)
		if err != nil {
			t.Fatalf("RandomToken: %v", err)
		}
		if seen[tok] {
			t.Fatalf("RandomToken produced a duplicate: %q", tok)
		}
		seen[tok] = true
	}
}

func TestRandomTokenRejectsNonPositiveLength(t *testing.T) {
	if _, err := RandomToken(0); err == nil {
		t.Error("RandomToken(0) should error")
	}
	if _, err := RandomToken(-1); err == nil {
		t.Error("RandomToken(-1) should error")
	}
}
