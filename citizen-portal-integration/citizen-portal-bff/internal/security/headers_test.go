package security

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSecurityHeadersMiddleware(t *testing.T) {
	handler := SecurityHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	want := map[string]string{
		"X-Frame-Options":            "DENY",
		"X-Content-Type-Options":     "nosniff",
		"Referrer-Policy":            "no-referrer",
		"Content-Security-Policy":    "default-src 'self'; frame-ancestors 'none'",
		"Cross-Origin-Opener-Policy": "same-origin",
	}
	for header, want := range want {
		if got := rec.Header().Get(header); got != want {
			t.Errorf("header %s = %q, want %q", header, got, want)
		}
	}
}
