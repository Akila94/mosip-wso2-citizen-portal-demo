package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/akila94/mosip-wso2-citizen-portal-demo/citizen-portal-bff/internal/session"
	"github.com/akila94/mosip-wso2-citizen-portal-demo/citizen-portal-bff/internal/upstream"
)

// errUpstreamTransportFailureForTest stands in for a genuine transport
// failure from internal/upstream (connection refused, timeout, etc.),
// distinct from gov-services-api returning a non-2xx status.
var errUpstreamTransportFailureForTest = errors.New("fake upstream transport failure")

// newDataRouteTestServer builds a Server with all three apps registered
// (mirroring cmd/bff/main.go's wiring) and up as the upstream client, so
// the /bff/{app}/api/... data routes' full router — including the
// requireSession middleware chain — can be exercised end to end via
// s.Router().ServeHTTP, the same way a real request would reach them.
func newDataRouteTestServer(t *testing.T, up UpstreamClient) (*Server, map[string]*AppRoute) {
	t.Helper()

	apps := map[string]*AppRoute{
		"portal": {
			Key: "portal", RoutePrefix: "/bff/portal", ReturnToPrefix: "/",
			SessionCookieName: "cp_sid", LoginTxnCookieName: "cp_txn", CSRFCookieName: "cp_csrf",
			ClientID: "portal-client-id", AppName: "Citizen Portal", Client: &fakeClient{},
		},
		"driving-licence": {
			Key: "driving-licence", RoutePrefix: "/bff/driving-licence", ReturnToPrefix: "/apps/driving-licence",
			SessionCookieName: "dl_sid", LoginTxnCookieName: "dl_txn", CSRFCookieName: "dl_csrf",
			ClientID: "dl-client-id", AppName: "Driving Licence Service", Client: &fakeClient{},
		},
		"revenue-licence": {
			Key: "revenue-licence", RoutePrefix: "/bff/revenue-licence", ReturnToPrefix: "/apps/revenue-licence",
			SessionCookieName: "vrl_sid", LoginTxnCookieName: "vrl_txn", CSRFCookieName: "vrl_csrf",
			ClientID: "vrl-client-id", AppName: "Vehicle Revenue Licence", Client: &fakeClient{},
		},
	}

	mgr := session.NewManager(session.Config{
		MaxSessions: 100,
		LoginTxnTTL: time.Minute,
		IdleTimeout: time.Minute,
	})
	t.Cleanup(mgr.Close)

	s := NewServer(apps, mgr, false, time.Minute, discardLogger(), up)
	return s, apps
}

// sessionCookieFor creates sess in s's session store and returns the cookie
// a browser would present for app to authenticate as that session.
func sessionCookieFor(t *testing.T, s *Server, app *AppRoute, sess session.AuthSession) *http.Cookie {
	t.Helper()
	key, err := s.Sessions.CreateSession(sess)
	if err != nil {
		t.Fatalf("creating session: %v", err)
	}
	return &http.Cookie{Name: app.SessionCookieName, Value: key}
}

// csrfCookieAndHeader returns a matching CSRF cookie/header pair for app,
// standing in for the double-submit token a real login would have set.
func csrfCookieAndHeader(app *AppRoute) (*http.Cookie, string) {
	const token = "csrf-token-for-test"
	return &http.Cookie{Name: app.CSRFCookieName, Value: token}, token
}

// --- Task 3b: assuranceLevel must be derived server-side, never taken
// from the request. ---

func TestPortalCatalogueDerivesAssuranceLevelFromSessionNotRequest(t *testing.T) {
	up := &fakeUpstream{}
	s, apps := newDataRouteTestServer(t, up)
	app := apps["portal"]

	// Amr derives to "basic" (no eSignet authenticator present), but the
	// caller tries to self-report "substantial" via the query string.
	sess := session.AuthSession{AppKey: "portal", AccessToken: "at-basic", Amr: []string{"BasicAuthenticator"}}
	cookie := sessionCookieFor(t, s, app, sess)

	req := httptest.NewRequest(http.MethodGet, "/bff/portal/api/catalogue?assuranceLevel=substantial", nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	s.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", rec.Code, rec.Body.String())
	}
	if up.lastMethod != "PortalCatalogue" {
		t.Fatalf("upstream method called = %q, want PortalCatalogue", up.lastMethod)
	}
	if up.lastAssuranceLevel != string(session.AssuranceBasic) {
		t.Errorf("assuranceLevel passed upstream = %q, want %q (derived from session, not the query string)", up.lastAssuranceLevel, session.AssuranceBasic)
	}
	if up.lastAccessToken != "at-basic" {
		t.Errorf("accessToken passed upstream = %q, want %q", up.lastAccessToken, "at-basic")
	}
}

func TestPortalCatalogueRejectsWithoutSession(t *testing.T) {
	up := &fakeUpstream{}
	s, _ := newDataRouteTestServer(t, up)

	req := httptest.NewRequest(http.MethodGet, "/bff/portal/api/catalogue", nil)
	rec := httptest.NewRecorder()
	s.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
	var body sessionView
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decoding body: %v", err)
	}
	if body.Authenticated {
		t.Errorf("body = %+v, want authenticated=false (the same shape handleSession returns)", body)
	}
	if up.lastMethod != "" {
		t.Errorf("upstream should never be called without a session, got method %q", up.lastMethod)
	}
}

// --- Portal: one happy-path proving verbatim response passthrough. ---

func TestPortalTimelineHappyPathForwardsUpstreamResponseVerbatim(t *testing.T) {
	up := &fakeUpstream{response: upstream.Response{StatusCode: http.StatusOK, ContentType: "application/json", Body: []byte(`[{"id":"t1"}]`)}}
	s, apps := newDataRouteTestServer(t, up)
	app := apps["portal"]

	cookie := sessionCookieFor(t, s, app, session.AuthSession{AppKey: "portal", AccessToken: "at-1"})
	req := httptest.NewRequest(http.MethodGet, "/bff/portal/api/timeline", nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	s.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if rec.Body.String() != `[{"id":"t1"}]` {
		t.Errorf("body = %q, want the upstream body forwarded verbatim", rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
	if up.lastMethod != "PortalTimeline" || up.lastAccessToken != "at-1" {
		t.Errorf("unexpected upstream call: method=%q accessToken=%q", up.lastMethod, up.lastAccessToken)
	}
}

// --- Driving licence: happy path, week validation, and the CSRF gate on
// the write route. ---

func TestDrivingLicenceConfigHappyPath(t *testing.T) {
	up := &fakeUpstream{response: upstream.Response{StatusCode: http.StatusOK, ContentType: "application/json", Body: []byte(`{"permitNumber":"LP-1"}`)}}
	s, apps := newDataRouteTestServer(t, up)
	app := apps["driving-licence"]

	cookie := sessionCookieFor(t, s, app, session.AuthSession{AppKey: "driving-licence", AccessToken: "at-dl"})
	req := httptest.NewRequest(http.MethodGet, "/bff/driving-licence/api/config", nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	s.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", rec.Code, rec.Body.String())
	}
	if rec.Body.String() != `{"permitNumber":"LP-1"}` {
		t.Errorf("body = %q, want the upstream body forwarded verbatim", rec.Body.String())
	}
	if up.lastMethod != "DrivingLicenceConfig" || up.lastAccessToken != "at-dl" {
		t.Errorf("unexpected upstream call: method=%q accessToken=%q", up.lastMethod, up.lastAccessToken)
	}
}

func TestDrivingLicenceTestSlotsRejectsNonIntegerWeekWithoutCallingUpstream(t *testing.T) {
	up := &fakeUpstream{}
	s, apps := newDataRouteTestServer(t, up)
	app := apps["driving-licence"]

	cookie := sessionCookieFor(t, s, app, session.AuthSession{AppKey: "driving-licence", AccessToken: "at-dl"})
	req := httptest.NewRequest(http.MethodGet, "/bff/driving-licence/api/test-slots?week=not-a-number", nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	s.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	if up.lastMethod != "" {
		t.Errorf("upstream should never be called for an invalid week, got method %q", up.lastMethod)
	}
}

func TestDrivingLicenceTestSlotsPassesParsedWeekUpstream(t *testing.T) {
	up := &fakeUpstream{}
	s, apps := newDataRouteTestServer(t, up)
	app := apps["driving-licence"]

	cookie := sessionCookieFor(t, s, app, session.AuthSession{AppKey: "driving-licence", AccessToken: "at-dl"})
	req := httptest.NewRequest(http.MethodGet, "/bff/driving-licence/api/test-slots?week=3", nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	s.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", rec.Code, rec.Body.String())
	}
	if up.lastWeek != 3 {
		t.Errorf("week passed upstream = %d, want 3", up.lastWeek)
	}
}

func TestDrivingLicenceApplicationsRejectsWithoutCSRFToken(t *testing.T) {
	up := &fakeUpstream{}
	s, apps := newDataRouteTestServer(t, up)
	app := apps["driving-licence"]

	cookie := sessionCookieFor(t, s, app, session.AuthSession{AppKey: "driving-licence", AccessToken: "at-dl"})
	req := httptest.NewRequest(http.MethodPost, "/bff/driving-licence/api/applications", nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	s.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rec.Code)
	}
	if up.lastMethod != "" {
		t.Errorf("upstream should never be called without a valid CSRF token, got method %q", up.lastMethod)
	}
}

func TestDrivingLicenceApplicationsSucceedsWithMatchingCSRFToken(t *testing.T) {
	up := &fakeUpstream{response: upstream.Response{StatusCode: http.StatusOK, ContentType: "application/json", Body: []byte(`{"reference":"DL-1"}`)}}
	s, apps := newDataRouteTestServer(t, up)
	app := apps["driving-licence"]

	sessionCookie := sessionCookieFor(t, s, app, session.AuthSession{AppKey: "driving-licence", AccessToken: "at-dl"})
	csrfCookie, csrfHeader := csrfCookieAndHeader(app)

	req := httptest.NewRequest(http.MethodPost, "/bff/driving-licence/api/applications", strings.NewReader(`{}`))
	req.AddCookie(sessionCookie)
	req.AddCookie(csrfCookie)
	req.Header.Set(csrfHeaderName, csrfHeader)
	rec := httptest.NewRecorder()
	s.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", rec.Code, rec.Body.String())
	}
	if rec.Body.String() != `{"reference":"DL-1"}` {
		t.Errorf("body = %q, want the upstream body forwarded verbatim", rec.Body.String())
	}
	if up.lastMethod != "DrivingLicenceSubmitApplication" {
		t.Errorf("upstream method = %q, want DrivingLicenceSubmitApplication", up.lastMethod)
	}
}

// --- Revenue licence: happy path and the CSRF gate on the renew route. ---

func TestVehicleRegistryVehiclesHappyPath(t *testing.T) {
	up := &fakeUpstream{response: upstream.Response{StatusCode: http.StatusOK, ContentType: "application/json", Body: []byte(`[{"id":"CAB-4471"}]`)}}
	s, apps := newDataRouteTestServer(t, up)
	app := apps["revenue-licence"]

	cookie := sessionCookieFor(t, s, app, session.AuthSession{AppKey: "revenue-licence", AccessToken: "at-vrl"})
	req := httptest.NewRequest(http.MethodGet, "/bff/revenue-licence/api/vehicles", nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	s.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", rec.Code, rec.Body.String())
	}
	if rec.Body.String() != `[{"id":"CAB-4471"}]` {
		t.Errorf("body = %q, want the upstream body forwarded verbatim", rec.Body.String())
	}
	if up.lastMethod != "VehicleRegistryVehicles" || up.lastAccessToken != "at-vrl" {
		t.Errorf("unexpected upstream call: method=%q accessToken=%q", up.lastMethod, up.lastAccessToken)
	}
}

func TestVehicleRegistryRenewRejectsWithoutCSRFToken(t *testing.T) {
	up := &fakeUpstream{}
	s, apps := newDataRouteTestServer(t, up)
	app := apps["revenue-licence"]

	cookie := sessionCookieFor(t, s, app, session.AuthSession{AppKey: "revenue-licence", AccessToken: "at-vrl"})
	req := httptest.NewRequest(http.MethodPost, "/bff/revenue-licence/api/vehicles/CAB-4471/renew", nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	s.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rec.Code)
	}
	if up.lastMethod != "" {
		t.Errorf("upstream should never be called without a valid CSRF token, got method %q", up.lastMethod)
	}
}

func TestVehicleRegistryRenewSucceedsWithMatchingCSRFTokenAndForwardsVehicleID(t *testing.T) {
	up := &fakeUpstream{response: upstream.Response{StatusCode: http.StatusOK, ContentType: "application/json", Body: []byte(`{"receiptRef":"PAY-1"}`)}}
	s, apps := newDataRouteTestServer(t, up)
	app := apps["revenue-licence"]

	sessionCookie := sessionCookieFor(t, s, app, session.AuthSession{AppKey: "revenue-licence", AccessToken: "at-vrl"})
	csrfCookie, csrfHeader := csrfCookieAndHeader(app)

	req := httptest.NewRequest(http.MethodPost, "/bff/revenue-licence/api/vehicles/CAB-4471/renew", nil)
	req.AddCookie(sessionCookie)
	req.AddCookie(csrfCookie)
	req.Header.Set(csrfHeaderName, csrfHeader)
	rec := httptest.NewRecorder()
	s.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", rec.Code, rec.Body.String())
	}
	if rec.Body.String() != `{"receiptRef":"PAY-1"}` {
		t.Errorf("body = %q, want the upstream body forwarded verbatim", rec.Body.String())
	}
	if up.lastMethod != "VehicleRegistryRenew" || up.lastVehicleID != "CAB-4471" {
		t.Errorf("unexpected upstream call: method=%q vehicleID=%q", up.lastMethod, up.lastVehicleID)
	}
}

// --- /identity is shared across the two write apps. ---

func TestDrivingLicenceIdentityCallsCitizenProfile(t *testing.T) {
	up := &fakeUpstream{}
	s, apps := newDataRouteTestServer(t, up)
	app := apps["driving-licence"]

	cookie := sessionCookieFor(t, s, app, session.AuthSession{AppKey: "driving-licence", AccessToken: "at-dl"})
	req := httptest.NewRequest(http.MethodGet, "/bff/driving-licence/api/identity", nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	s.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", rec.Code, rec.Body.String())
	}
	if up.lastMethod != "CitizenProfile" {
		t.Errorf("upstream method = %q, want CitizenProfile", up.lastMethod)
	}
}

func TestRevenueLicenceIdentityCallsCitizenProfile(t *testing.T) {
	up := &fakeUpstream{}
	s, apps := newDataRouteTestServer(t, up)
	app := apps["revenue-licence"]

	cookie := sessionCookieFor(t, s, app, session.AuthSession{AppKey: "revenue-licence", AccessToken: "at-vrl"})
	req := httptest.NewRequest(http.MethodGet, "/bff/revenue-licence/api/identity", nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	s.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", rec.Code, rec.Body.String())
	}
	if up.lastMethod != "CitizenProfile" {
		t.Errorf("upstream method = %q, want CitizenProfile", up.lastMethod)
	}
}

// --- The CSRF round trip the SPA actually performs: read the token from
// GET /session's body (it cannot read the HttpOnly, path-scoped cookie),
// then echo it as X-CSRF-Token on a write. ---

// csrfTokenFromSessionEndpoint performs the SPA's own first step: call
// /bff/{app}/session with the session and CSRF cookies and read the token
// out of the response body.
func csrfTokenFromSessionEndpoint(t *testing.T, s *Server, app *AppRoute, sessionCookie, csrfCookie *http.Cookie) string {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, app.RoutePrefix+"/session", nil)
	req.AddCookie(sessionCookie)
	req.AddCookie(csrfCookie)
	rec := httptest.NewRecorder()
	s.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("/session status = %d, want 200, body=%s", rec.Code, rec.Body.String())
	}
	var body sessionView
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decoding /session body: %v", err)
	}
	if body.CSRFToken == "" {
		t.Fatal("/session returned no csrfToken — the SPA would have no way to perform a write")
	}
	return body.CSRFToken
}

func TestWriteRouteAcceptsTheCSRFTokenTheSessionEndpointHandedOut(t *testing.T) {
	up := &fakeUpstream{response: upstream.Response{StatusCode: http.StatusOK, ContentType: "application/json", Body: []byte(`{"reference":"DL-1"}`)}}
	s, apps := newDataRouteTestServer(t, up)
	app := apps["driving-licence"]

	sessionCookie := sessionCookieFor(t, s, app, session.AuthSession{AppKey: "driving-licence", AccessToken: "at-dl"})
	csrfCookie, _ := csrfCookieAndHeader(app)
	token := csrfTokenFromSessionEndpoint(t, s, app, sessionCookie, csrfCookie)

	req := httptest.NewRequest(http.MethodPost, "/bff/driving-licence/api/applications", strings.NewReader(`{}`))
	req.AddCookie(sessionCookie)
	req.AddCookie(csrfCookie)
	req.Header.Set(csrfHeaderName, token)
	rec := httptest.NewRecorder()
	s.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", rec.Code, rec.Body.String())
	}
	if up.lastMethod != "DrivingLicenceSubmitApplication" {
		t.Errorf("upstream method = %q", up.lastMethod)
	}
}

func TestWriteRouteRejectsAHeaderThatDoesNotMatchTheCookie(t *testing.T) {
	up := &fakeUpstream{}
	s, apps := newDataRouteTestServer(t, up)
	app := apps["driving-licence"]

	sessionCookie := sessionCookieFor(t, s, app, session.AuthSession{AppKey: "driving-licence", AccessToken: "at-dl"})
	csrfCookie, _ := csrfCookieAndHeader(app)

	req := httptest.NewRequest(http.MethodPost, "/bff/driving-licence/api/applications", strings.NewReader(`{}`))
	req.AddCookie(sessionCookie)
	req.AddCookie(csrfCookie)
	req.Header.Set(csrfHeaderName, csrfCookie.Value+"-tampered")
	rec := httptest.NewRecorder()
	s.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rec.Code)
	}
	if up.lastMethod != "" {
		t.Errorf("upstream must not be called for a mismatched CSRF token, got %q", up.lastMethod)
	}
}

// --- A genuine upstream transport failure surfaces as 502. ---

func TestPortalTimelineRespondsWithBadGatewayOnUpstreamTransportError(t *testing.T) {
	up := &fakeUpstream{err: errUpstreamTransportFailureForTest}
	s, apps := newDataRouteTestServer(t, up)
	app := apps["portal"]

	cookie := sessionCookieFor(t, s, app, session.AuthSession{AppKey: "portal", AccessToken: "at-1"})
	req := httptest.NewRequest(http.MethodGet, "/bff/portal/api/timeline", nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	s.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, want 502", rec.Code)
	}
}
