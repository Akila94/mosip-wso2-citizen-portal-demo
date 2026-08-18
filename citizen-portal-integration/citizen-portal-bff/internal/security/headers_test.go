package security

import (
	"net/http"
	"net/http/httptest"
	"strings"
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

// parseCSP splits a Content-Security-Policy header into directive name ->
// source list, so the SPA CSP tests assert on directives rather than on one
// brittle string literal.
func parseCSP(t *testing.T, policy string) map[string][]string {
	t.Helper()
	directives := make(map[string][]string)
	for _, raw := range strings.Split(policy, ";") {
		fields := strings.Fields(raw)
		if len(fields) == 0 {
			continue
		}
		directives[fields[0]] = fields[1:]
	}
	return directives
}

func containsSource(sources []string, want string) bool {
	for _, s := range sources {
		if s == want {
			return true
		}
	}
	return false
}

// spaHeadersRecorder runs the SPA middleware chained after SecurityHeaders,
// exactly as the router does, so these tests also prove the SPA policy
// replaces (rather than appends to) the strict API policy.
func spaHeadersRecorder(t *testing.T, devMode bool) *httptest.ResponseRecorder {
	t.Helper()
	handler := SecurityHeaders(SPAHeaders(devMode)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})))
	req := httptest.NewRequest(http.MethodGet, "/timeline", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

func TestSPAHeadersReplaceTheStrictAPIPolicyWithExactlyOnePolicy(t *testing.T) {
	rec := spaHeadersRecorder(t, false)

	policies := rec.Header().Values("Content-Security-Policy")
	if len(policies) != 1 {
		t.Fatalf("Content-Security-Policy set %d times, want exactly 1: %q", len(policies), policies)
	}
	if policies[0] == APIContentSecurityPolicy {
		t.Error("SPA responses must not carry the strict API CSP — inline style attributes would be blocked")
	}
}

func TestSPAContentSecurityPolicyStaticModeAllowsInlineStylesAndDataImagesOnly(t *testing.T) {
	directives := parseCSP(t, SPAContentSecurityPolicy(false))

	// The React screens use element style={{...}} attributes pervasively
	// (650 occurrences across 80 files in citizen-portal-demo-app/src), so
	// style-src needs 'unsafe-inline'.
	if !containsSource(directives["style-src"], "'unsafe-inline'") {
		t.Errorf("style-src = %v, want 'unsafe-inline'", directives["style-src"])
	}
	// eSignet's `picture` claim is a data: URI JPEG.
	if !containsSource(directives["img-src"], "data:") {
		t.Errorf("img-src = %v, want data:", directives["img-src"])
	}
	// Production must never relax script-src.
	if containsSource(directives["script-src"], "'unsafe-inline'") || containsSource(directives["script-src"], "'unsafe-eval'") {
		t.Errorf("script-src = %v, want no inline/eval relaxation in static mode", directives["script-src"])
	}
	if !containsSource(directives["connect-src"], "'self'") || len(directives["connect-src"]) != 1 {
		t.Errorf("connect-src = %v, want exactly 'self' in static mode", directives["connect-src"])
	}
}

func TestSPAContentSecurityPolicyDevModeAllowsViteInlinePreambleAndHMRSocket(t *testing.T) {
	directives := parseCSP(t, SPAContentSecurityPolicy(true))

	if !containsSource(directives["script-src"], "'unsafe-inline'") {
		t.Errorf("script-src = %v, want 'unsafe-inline' in dev mode (Vite's injected preamble)", directives["script-src"])
	}
	if !containsSource(directives["script-src"], "'unsafe-eval'") {
		t.Errorf("script-src = %v, want 'unsafe-eval' in dev mode", directives["script-src"])
	}
	if !containsSource(directives["connect-src"], "ws:") && !containsSource(directives["connect-src"], "wss:") {
		t.Errorf("connect-src = %v, want a WebSocket source in dev mode (Vite HMR)", directives["connect-src"])
	}
}

func TestSPAContentSecurityPolicyKeepsHardInvariantsInBothModes(t *testing.T) {
	for _, devMode := range []bool{false, true} {
		directives := parseCSP(t, SPAContentSecurityPolicy(devMode))
		for directive, want := range map[string]string{
			"frame-ancestors": "'none'",
			"object-src":      "'none'",
			"base-uri":        "'self'",
			"form-action":     "'self'",
		} {
			if got := directives[directive]; len(got) != 1 || got[0] != want {
				t.Errorf("devMode=%v: %s = %v, want [%s]", devMode, directive, got, want)
			}
		}
	}
}

func TestSPAHeadersKeepXFrameOptionsDeny(t *testing.T) {
	for _, devMode := range []bool{false, true} {
		rec := spaHeadersRecorder(t, devMode)
		if got := rec.Header().Get("X-Frame-Options"); got != "DENY" {
			t.Errorf("devMode=%v: X-Frame-Options = %q, want DENY", devMode, got)
		}
	}
}
