package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/akila94/mosip-wso2-citizen-portal-demo/citizen-portal-bff/internal/security"
	"github.com/akila94/mosip-wso2-citizen-portal-demo/citizen-portal-bff/internal/upstream"
)

// The landing page renders the service catalogue to signed-out visitors, so
// this route has to work with no session cookie at all — that is the whole
// reason it exists next to the session-gated /api/catalogue.
func TestPublicCatalogueSucceedsWithNoSessionCookie(t *testing.T) {
	up := &fakeUpstream{response: upstream.Response{StatusCode: http.StatusOK, ContentType: "application/json", Body: []byte(`[{"id":"transport"}]`)}}
	s, _ := newDataRouteTestServer(t, up)

	req := httptest.NewRequest(http.MethodGet, "/bff/portal/public/catalogue", nil)
	rec := httptest.NewRecorder()
	s.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 with no session, body=%s", rec.Code, rec.Body.String())
	}
	if rec.Body.String() != `[{"id":"transport"}]` {
		t.Errorf("body = %q, want the upstream body forwarded verbatim", rec.Body.String())
	}
	if up.lastMethod != "PublicPortalCatalogue" {
		t.Errorf("upstream method = %q, want PublicPortalCatalogue", up.lastMethod)
	}
}

func TestPublicCatalogueCallsUpstreamWithNoAccessToken(t *testing.T) {
	up := &fakeUpstream{}
	s, _ := newDataRouteTestServer(t, up)

	req := httptest.NewRequest(http.MethodGet, "/bff/portal/public/catalogue", nil)
	rec := httptest.NewRecorder()
	s.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	// UpstreamClient.PublicPortalCatalogue takes no access token at all, so
	// there is structurally nothing to send; the fake records that no token
	// reached it.
	if up.lastAccessToken != "" {
		t.Errorf("upstream saw access token %q, want none on a public call", up.lastAccessToken)
	}
	if up.lastAssuranceLevel != "" {
		t.Errorf("upstream saw assuranceLevel %q, want none — there is no session to derive one from", up.lastAssuranceLevel)
	}
}

// Even when a session exists, the public route must stay public: it must not
// start injecting that session's token.
func TestPublicCatalogueSendsNoTokenEvenWhenASessionExists(t *testing.T) {
	up := &fakeUpstream{}
	s, apps := newDataRouteTestServer(t, up)

	cookie := sessionCookieFor(t, s, apps["portal"], testInspectorSession("portal"))
	req := httptest.NewRequest(http.MethodGet, "/bff/portal/public/catalogue", nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	s.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if up.lastAccessToken != "" {
		t.Errorf("upstream saw access token %q on a public route", up.lastAccessToken)
	}
}

// One named handler for one named upstream path — not the start of a
// generic unauthenticated proxy.
func TestPublicCatalogueExistsOnlyUnderThePortalPrefix(t *testing.T) {
	up := &fakeUpstream{}
	s, _ := newDataRouteTestServer(t, up)

	for _, path := range []string{
		"/bff/driving-licence/public/catalogue",
		"/bff/revenue-licence/public/catalogue",
		"/bff/portal/public/timeline",
		"/bff/portal/public/consents",
	} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		s.Router().ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("GET %s: status = %d, want 404 — only the portal catalogue is public", path, rec.Code)
		}
		if up.lastMethod != "" {
			t.Errorf("GET %s reached upstream method %q", path, up.lastMethod)
		}
	}
}

func TestPublicCatalogueCarriesTheStrictAPISecurityHeaders(t *testing.T) {
	up := &fakeUpstream{}
	s, _ := newDataRouteTestServer(t, up)
	s.SPA = &spaSpy{} // the SPA policy must not bleed onto this API response

	req := httptest.NewRequest(http.MethodGet, "/bff/portal/public/catalogue", nil)
	rec := httptest.NewRecorder()
	s.Router().ServeHTTP(rec, req)

	if csp := rec.Header().Get("Content-Security-Policy"); csp != security.APIContentSecurityPolicy {
		t.Errorf("Content-Security-Policy = %q, want the strict API policy — public is not unguarded", csp)
	}
	if xfo := rec.Header().Get("X-Frame-Options"); xfo != "DENY" {
		t.Errorf("X-Frame-Options = %q, want DENY", xfo)
	}
	if nosniff := rec.Header().Get("X-Content-Type-Options"); nosniff != "nosniff" {
		t.Errorf("X-Content-Type-Options = %q, want nosniff", nosniff)
	}
}

func TestPublicCatalogueRespondsWithBadGatewayOnUpstreamTransportError(t *testing.T) {
	up := &fakeUpstream{err: errUpstreamTransportFailureForTest}
	s, _ := newDataRouteTestServer(t, up)

	req := httptest.NewRequest(http.MethodGet, "/bff/portal/public/catalogue", nil)
	rec := httptest.NewRecorder()
	s.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, want 502", rec.Code)
	}
}
