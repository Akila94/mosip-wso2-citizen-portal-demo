package security

import (
	"strings"
	"testing"
)

func TestSanitizeForLog(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "plain string is unchanged",
			input: "hello world",
			want:  "hello world",
		},
		{
			name:  "strips carriage returns",
			input: "line1\rline2",
			want:  "line1line2",
		},
		{
			name:  "strips newlines",
			input: "line1\nline2",
			want:  "line1line2",
		},
		{
			name:  "strips tabs",
			input: "col1\tcol2",
			want:  "col1col2",
		},
		{
			name:  "strips CRLF injection attempt",
			input: "value\r\nInjected-Header: evil",
			want:  "valueInjected-Header: evil",
		},
		{
			name:  "empty string stays empty",
			input: "",
			want:  "",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := SanitizeForLog(tc.input); got != tc.want {
				t.Errorf("SanitizeForLog(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestSanitizeForLogCapsLength(t *testing.T) {
	input := strings.Repeat("a", maxLogFieldLength+500)
	got := SanitizeForLog(input)
	if len(got) != maxLogFieldLength {
		t.Errorf("len(got) = %d, want %d", len(got), maxLogFieldLength)
	}
	if !strings.HasSuffix(got, truncationMarker) {
		t.Errorf("got does not end with truncation marker: %q", got[len(got)-30:])
	}
}

func TestSanitizeForLogUnderCapUnaffected(t *testing.T) {
	input := strings.Repeat("b", maxLogFieldLength-1)
	got := SanitizeForLog(input)
	if got != input {
		t.Error("string under the cap should be returned unmodified")
	}
}
