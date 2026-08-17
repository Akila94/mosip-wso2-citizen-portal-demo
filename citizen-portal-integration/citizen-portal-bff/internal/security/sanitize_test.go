package security

import "testing"

func TestSanitizeForLog(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"plain", "hello", "hello"},
		{"strips CR", "line1\rline2", "line1line2"},
		{"strips LF", "line1\nline2", "line1line2"},
		{"strips CRLF injection", "user\r\nX-Injected: evil", "userX-Injected: evil"},
		{"strips tab", "a\tb", "ab"},
		{"strips multiple control chars", "a\r\n\t\r\nb", "ab"},
		{"empty stays empty", "", ""},
		{"unicode preserved", "café ☃", "café ☃"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := SanitizeForLog(tc.in); got != tc.want {
				t.Errorf("SanitizeForLog(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestSanitizeForLogCapsLength(t *testing.T) {
	long := make([]byte, 5000)
	for i := range long {
		long[i] = 'a'
	}
	got := SanitizeForLog(string(long))
	if len(got) > maxLogFieldLength {
		t.Errorf("SanitizeForLog did not cap length: got %d bytes, want <= %d", len(got), maxLogFieldLength)
	}
}

func TestSanitizeForLogCapsLengthWithTruncationMarker(t *testing.T) {
	long := make([]byte, maxLogFieldLength+100)
	for i := range long {
		long[i] = 'x'
	}
	got := SanitizeForLog(string(long))
	if len(got) > maxLogFieldLength+len(truncationMarker) {
		t.Errorf("unexpected length %d", len(got))
	}
}
