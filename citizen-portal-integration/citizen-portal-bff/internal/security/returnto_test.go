package security

import "testing"

func TestValidateReturnTo(t *testing.T) {
	const prefix = "/apps/driving-licence"

	valid := []string{
		"/apps/driving-licence",
		"/apps/driving-licence/",
		"/apps/driving-licence/step/1",
		"/apps/driving-licence/step/1?foo=bar",
	}
	for _, in := range valid {
		t.Run("valid: "+in, func(t *testing.T) {
			got, err := ValidateReturnTo(in, prefix)
			if err != nil {
				t.Fatalf("ValidateReturnTo(%q) unexpected error: %v", in, err)
			}
			if got != in {
				t.Errorf("ValidateReturnTo(%q) = %q, want unchanged", in, got)
			}
		})
	}

	invalid := []string{
		"",
		"relative/path",
		"//evil.com",
		"///evil.com",
		"http://evil.com",
		"https://evil.com/apps/driving-licence",
		"/\\evil.com",
		"/apps/revenue-licence",      // wrong app prefix
		"/apps/driving-licence-evil", // prefix-looks-like but escapes
		"/../etc/passwd",
		"/apps/driving-licence/../../etc/passwd",
		"/apps/driving-licence\r\nSet-Cookie: evil=1",
		"  /apps/driving-licence", // leading whitespace
		"/apps/driving-licence\t",
	}
	for _, in := range invalid {
		t.Run("invalid", func(t *testing.T) {
			if _, err := ValidateReturnTo(in, prefix); err == nil {
				t.Errorf("ValidateReturnTo(%q) = nil error, want rejection", in)
			}
		})
	}
}

func TestValidateReturnToRootPrefix(t *testing.T) {
	// The portal app's prefix is "/" — every in-app path should be accepted,
	// but scheme-relative and absolute URLs must still be rejected.
	got, err := ValidateReturnTo("/timeline", "/")
	if err != nil || got != "/timeline" {
		t.Fatalf("ValidateReturnTo(/timeline, /) = %q, %v", got, err)
	}
	if _, err := ValidateReturnTo("//evil.com", "/"); err == nil {
		t.Error("expected rejection of protocol-relative URL under root prefix")
	}
}
