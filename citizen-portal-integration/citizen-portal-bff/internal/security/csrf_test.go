package security

import "testing"

func TestConstantTimeEqual(t *testing.T) {
	if !ConstantTimeEqual("abc123", "abc123") {
		t.Error("expected equal strings to match")
	}
	if ConstantTimeEqual("abc123", "abc124") {
		t.Error("expected different strings to not match")
	}
	if ConstantTimeEqual("abc", "abcd") {
		t.Error("expected different-length strings to not match")
	}
	if ConstantTimeEqual("", "") {
		t.Error("expected empty strings to be rejected, not treated as a match")
	}
}
