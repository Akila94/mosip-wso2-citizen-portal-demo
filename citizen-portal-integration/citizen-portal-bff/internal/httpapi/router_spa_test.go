package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/akila94/mosip-wso2-citizen-portal-demo/citizen-portal-bff/internal/devproxy"
	"github.com/akila94/mosip-wso2-citizen-portal-demo/citizen-portal-bff/internal/security"
	"github.com/akila94/mosip-wso2-citizen-portal-demo/citizen-portal-bff/internal/session"
)

// spaSpy stands in for the internal/devproxy handler: it records what the
// router handed to it, so these tests prove the *wiring* (which requests
// fall through to the SPA) without duplicating devproxy's own tests.
type spaSpy struct {
	servedPaths []string
}

func (s *spaSpy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.servedPaths = append(s.servedPaths, r.URL.Path)
	w.Header().Set("Content-Type", "text/html")
	if _, err := w.Write([]byte("<!doctype html>spa")); err != nil {
		panic(err)
	}
}

func TestRouterSendsUnmatchedPathsToTheSPA(t *testing.T) {
	s, _ := newDataRouteTestServer(t, &fakeUpstream{})
	spa := &spaSpy{}
	s.SPA = spa

	for _, path := range []string{"/", "/timeline", "/apps/driving-licence/step/2", "/assets/index-abc.js"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		s.Router().ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("GET %s: status = %d, want 200", path, rec.Code)
		}
	}
	if len(spa.servedPaths) != 4 {
		t.Fatalf("SPA served %v, want all four unmatched paths", spa.servedPaths)
	}
}

func TestRouterPrefersRegisteredBFFRoutesOverTheSPA(t *testing.T) {
	s, apps := newDataRouteTestServer(t, &fakeUpstream{})
	spa := &spaSpy{}
	s.SPA = spa

	app := apps["portal"]
	cookie := sessionCookieFor(t, s, app, session.AuthSession{AppKey: "portal", AccessToken: "at-1"})
	req := httptest.NewRequest(http.MethodGet, "/bff/portal/api/timeline", nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	s.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 from the real data route", rec.Code)
	}
	if len(spa.servedPaths) != 0 {
		t.Errorf("SPA handled %v — registered /bff routes must always win", spa.servedPaths)
	}
}

func TestRouterReturnsJSONNotFoundWhenNoSPAIsConfigured(t *testing.T) {
	s, _ := newDataRouteTestServer(t, &fakeUpstream{})

	req := httptest.NewRequest(http.MethodGet, "/bff/portal/nope", nil)
	rec := httptest.NewRecorder()
	s.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "application/json") {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("body %q is not JSON: %v", rec.Body.String(), err)
	}
}

// A mistyped path under an app's /api subrouter is only reachable past the
// requireSession middleware, so this is the case that would otherwise return
// index.html to a fetch() and be impossible to debug.
func TestRouterSendsAnUnknownAPIPathToTheSPAHandlerForItsJSON404(t *testing.T) {
	s, apps := newDataRouteTestServer(t, &fakeUpstream{})
	spa := &spaSpy{}
	s.SPA = spa

	app := apps["portal"]
	cookie := sessionCookieFor(t, s, app, session.AuthSession{AppKey: "portal", AccessToken: "at-1"})
	req := httptest.NewRequest(http.MethodGet, "/bff/portal/api/typo", nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	s.Router().ServeHTTP(rec, req)

	// internal/devproxy owns the /bff/ guard (and tests it); here the point
	// is only that the request reaches it rather than dying inside chi.
	if len(spa.servedPaths) != 1 || spa.servedPaths[0] != "/bff/portal/api/typo" {
		t.Fatalf("SPA handler saw %v, want the unmatched /bff path so its guard can answer with JSON", spa.servedPaths)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want the spy's own 200 — chi must not answer this itself", rec.Code)
	}
}

// TestRouterWithTheRealSPAHandler wires the actual internal/devproxy
// handler behind the actual router — the production composition — so the
// two seams the spy cannot show are covered: a deep link really returns
// index.html under the SPA policy, and an unmatched /bff path really comes
// back as JSON under the strict API policy.
func TestRouterWithTheRealSPAHandler(t *testing.T) {
	staticDir := t.TempDir()
	const indexBody = "<!doctype html>real spa"
	if err := os.WriteFile(filepath.Join(staticDir, "index.html"), []byte(indexBody), 0o600); err != nil {
		t.Fatalf("writing index.html: %v", err)
	}
	spa, err := devproxy.New(devproxy.Config{StaticDir: staticDir, Logger: discardLogger()})
	if err != nil {
		t.Fatalf("devproxy.New: %v", err)
	}

	s, _ := newDataRouteTestServer(t, &fakeUpstream{})
	s.SPA = spa

	deepLink := httptest.NewRequest(http.MethodGet, "/apps/driving-licence/step/2", nil)
	deepLinkRec := httptest.NewRecorder()
	s.Router().ServeHTTP(deepLinkRec, deepLink)

	if deepLinkRec.Code != http.StatusOK || deepLinkRec.Body.String() != indexBody {
		t.Fatalf("deep link -> %d %q, want 200 and the SPA index", deepLinkRec.Code, deepLinkRec.Body.String())
	}
	if csp := deepLinkRec.Header().Get("Content-Security-Policy"); csp != security.SPAContentSecurityPolicy(false) {
		t.Errorf("deep-link CSP = %q, want the static SPA policy", csp)
	}

	apiTypo := httptest.NewRequest(http.MethodGet, "/bff/portal/nope", nil)
	apiTypoRec := httptest.NewRecorder()
	s.Router().ServeHTTP(apiTypoRec, apiTypo)

	if apiTypoRec.Code != http.StatusNotFound {
		t.Fatalf("unmatched /bff path -> %d, want 404", apiTypoRec.Code)
	}
	if strings.Contains(apiTypoRec.Body.String(), "<!doctype html") {
		t.Errorf("unmatched /bff path returned the SPA: %s", apiTypoRec.Body.String())
	}
	if csp := apiTypoRec.Header().Get("Content-Security-Policy"); csp != security.APIContentSecurityPolicy {
		t.Errorf("unmatched /bff path CSP = %q, want the strict API policy", csp)
	}
}

func TestRouterAppliesTheSPAContentSecurityPolicyToSPAResponsesOnly(t *testing.T) {
	s, _ := newDataRouteTestServer(t, &fakeUpstream{})
	s.SPA = &spaSpy{}

	spaReq := httptest.NewRequest(http.MethodGet, "/timeline", nil)
	spaRec := httptest.NewRecorder()
	s.Router().ServeHTTP(spaRec, spaReq)

	apiReq := httptest.NewRequest(http.MethodGet, "/bff/portal/session", nil)
	apiRec := httptest.NewRecorder()
	s.Router().ServeHTTP(apiRec, apiReq)

	spaCSP := spaRec.Header().Get("Content-Security-Policy")
	apiCSP := apiRec.Header().Get("Content-Security-Policy")
	if spaCSP == apiCSP {
		t.Fatalf("SPA and API responses share the CSP %q — the API policy must stay strict", apiCSP)
	}
	if strings.Contains(apiCSP, "unsafe-inline") {
		t.Errorf("API CSP = %q, must not be relaxed", apiCSP)
	}
	if !strings.Contains(spaCSP, "unsafe-inline") {
		t.Errorf("SPA CSP = %q, must allow the inline style attributes the React screens use", spaCSP)
	}
}
