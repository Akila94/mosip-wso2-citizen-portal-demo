package httpapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"log/slog"
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

// --- An upstream 401 is two different failures wearing one status code:
// an access token that really has expired (the citizen can fix that by
// signing in again) and a live token the resource server refused for a
// configuration reason (they cannot). Forwarding both as 401 sent a citizen
// into a re-login loop that could never work, so they are told apart here. ---

// recordingLogger returns a logger writing into the returned buffer, so a
// test can assert on what a handler logged. These log lines are not
// incidental: pointing the next reader at the real cause of an upstream 401
// is half of what this split exists for, so they are worth asserting on.
func recordingLogger() (*slog.Logger, *bytes.Buffer) {
	var logs bytes.Buffer
	return slog.New(slog.NewTextHandler(&logs, nil)), &logs
}

// upstreamRejectedResponse is what gov-services-api answers with when it
// refuses an access token — the body is deliberately distinctive so a test
// can prove the BFF did not pass it through.
func upstreamRejectedResponse(statusCode int) upstream.Response {
	return upstream.Response{
		StatusCode:  statusCode,
		ContentType: "application/json",
		Body:        []byte(`{"error":"invalid_token","error_description":"upstream detail that must not reach the browser"}`),
	}
}

func TestUpstreamUnauthorizedWithAnExpiredAccessTokenStaysUnauthorized(t *testing.T) {
	up := &fakeUpstream{response: upstreamRejectedResponse(http.StatusUnauthorized)}
	s, apps := newDataRouteTestServer(t, up)
	logger, logs := recordingLogger()
	s.Logger = logger
	app := apps["portal"]

	sess := session.AuthSession{
		AppKey:               "portal",
		AccessToken:          "at-expired",
		AccessTokenExpiresAt: time.Now().Add(-time.Minute),
	}
	cookie := sessionCookieFor(t, s, app, sess)
	req := httptest.NewRequest(http.MethodGet, "/bff/portal/api/timeline", nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	s.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401 — the token really had expired, so signing in again is the fix", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "upstream detail") {
		t.Errorf("body = %q, want our own message rather than the upstream error body", rec.Body.String())
	}
	if strings.Contains(logs.String(), "level=ERROR") {
		t.Errorf("logged at ERROR: %q — an expired token is an expected condition, not a fault", logs.String())
	}
	if !strings.Contains(logs.String(), "level=WARN") {
		t.Errorf("logs = %q, want a WARN line recording the expired token", logs.String())
	}
}

func TestUpstreamUnauthorizedWithALiveAccessTokenIsABadGateway(t *testing.T) {
	up := &fakeUpstream{response: upstreamRejectedResponse(http.StatusUnauthorized)}
	s, apps := newDataRouteTestServer(t, up)
	logger, logs := recordingLogger()
	s.Logger = logger
	app := apps["portal"]

	sess := session.AuthSession{
		AppKey:               "portal",
		AccessToken:          "at-live",
		AccessTokenExpiresAt: time.Now().Add(time.Hour),
	}
	cookie := sessionCookieFor(t, s, app, sess)
	req := httptest.NewRequest(http.MethodGet, "/bff/portal/api/timeline", nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	s.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, want 502 — the citizen's session is fine, so calling this a session expiry is a lie", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "upstream detail") {
		t.Errorf("body = %q, want our own message rather than the upstream error body", rec.Body.String())
	}

	logged := logs.String()
	if !strings.Contains(logged, "level=ERROR") {
		t.Errorf("logs = %q, want an ERROR line — a rejected live token is a server-side fault", logged)
	}
	if !strings.Contains(logged, "/bff/portal/api/timeline") {
		t.Errorf("logs = %q, want the request path so the failing call is identifiable", logged)
	}
	if !strings.Contains(logged, "401") {
		t.Errorf("logs = %q, want the upstream status", logged)
	}
	if !strings.Contains(logged, "opaque") || !strings.Contains(logged, "JWT") {
		t.Errorf("logs = %q, want the known cause named (an opaque, non-JWT access token on the Console registration)", logged)
	}
	if strings.Contains(logged, "at-live") {
		t.Errorf("logs = %q, want no access token in a log line ever", logged)
	}
}

func TestUpstreamUnauthorizedWithUnknownTokenExpiryIsABadGateway(t *testing.T) {
	up := &fakeUpstream{response: upstreamRejectedResponse(http.StatusUnauthorized)}
	s, apps := newDataRouteTestServer(t, up)
	app := apps["portal"]

	// No AccessTokenExpiresAt at all: IS did not tell us when the token
	// expires, so "expired" is unproven and blaming the citizen's session
	// would be a guess. Treated as the server-side fault it more likely is.
	cookie := sessionCookieFor(t, s, app, session.AuthSession{AppKey: "portal", AccessToken: "at-1"})
	req := httptest.NewRequest(http.MethodGet, "/bff/portal/api/timeline", nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	s.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, want 502 for an unknown token expiry", rec.Code)
	}
}

// The split has to be uniform across every session-backed route, not just
// the one it was first noticed on.
func TestUpstreamUnauthorizedIsTranslatedOnEverySessionBackedApp(t *testing.T) {
	cases := []struct {
		name     string
		appKey   string
		method   string
		path     string
		withCSRF bool
	}{
		{name: "portal read", appKey: "portal", method: http.MethodGet, path: "/bff/portal/api/catalogue"},
		{name: "portal department records", appKey: "portal", method: http.MethodGet, path: "/bff/portal/api/department-records"},
		{name: "driving licence read", appKey: "driving-licence", method: http.MethodGet, path: "/bff/driving-licence/api/config"},
		{name: "driving licence test slots", appKey: "driving-licence", method: http.MethodGet, path: "/bff/driving-licence/api/test-slots?week=2"},
		{name: "driving licence identity", appKey: "driving-licence", method: http.MethodGet, path: "/bff/driving-licence/api/identity"},
		{name: "driving licence write", appKey: "driving-licence", method: http.MethodPost, path: "/bff/driving-licence/api/applications", withCSRF: true},
		{name: "revenue licence read", appKey: "revenue-licence", method: http.MethodGet, path: "/bff/revenue-licence/api/vehicles"},
		{name: "revenue licence write", appKey: "revenue-licence", method: http.MethodPost, path: "/bff/revenue-licence/api/vehicles/CAB-4471/renew", withCSRF: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			up := &fakeUpstream{response: upstreamRejectedResponse(http.StatusUnauthorized)}
			s, apps := newDataRouteTestServer(t, up)
			app := apps[tc.appKey]

			sess := session.AuthSession{
				AppKey:               tc.appKey,
				AccessToken:          "at-live",
				AccessTokenExpiresAt: time.Now().Add(time.Hour),
			}
			req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(`{}`))
			req.AddCookie(sessionCookieFor(t, s, app, sess))
			if tc.withCSRF {
				csrfCookie, csrfHeader := csrfCookieAndHeader(app)
				req.AddCookie(csrfCookie)
				req.Header.Set(csrfHeaderName, csrfHeader)
			}
			rec := httptest.NewRecorder()
			s.Router().ServeHTTP(rec, req)

			if rec.Code != http.StatusBadGateway {
				t.Fatalf("status = %d, want 502, body=%s", rec.Code, rec.Body.String())
			}
			if strings.Contains(rec.Body.String(), "upstream detail") {
				t.Errorf("body = %q leaked the upstream error body", rec.Body.String())
			}
		})
	}
}

// 403 is already an unambiguous answer — the token was understood and its
// audience or scope was refused — so it keeps being forwarded untouched.
func TestUpstreamForbiddenIsStillForwardedVerbatim(t *testing.T) {
	up := &fakeUpstream{response: upstream.Response{
		StatusCode:  http.StatusForbidden,
		ContentType: "application/json",
		Body:        []byte(`{"error":"insufficient_scope"}`),
	}}
	s, apps := newDataRouteTestServer(t, up)
	app := apps["portal"]

	cookie := sessionCookieFor(t, s, app, session.AuthSession{AppKey: "portal", AccessToken: "at-1"})
	req := httptest.NewRequest(http.MethodGet, "/bff/portal/api/timeline", nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	s.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403 forwarded unchanged", rec.Code)
	}
	if rec.Body.String() != `{"error":"insufficient_scope"}` {
		t.Errorf("body = %q, want the upstream body forwarded verbatim", rec.Body.String())
	}
}

// Any other upstream status still passes straight through: only 401 was ever
// ambiguous.
func TestUpstreamNotFoundIsStillForwardedVerbatim(t *testing.T) {
	up := &fakeUpstream{response: upstream.Response{
		StatusCode:  http.StatusNotFound,
		ContentType: "application/json",
		Body:        []byte(`{"error":"no such vehicle"}`),
	}}
	s, apps := newDataRouteTestServer(t, up)
	app := apps["revenue-licence"]

	cookie := sessionCookieFor(t, s, app, session.AuthSession{AppKey: "revenue-licence", AccessToken: "at-vrl"})
	req := httptest.NewRequest(http.MethodGet, "/bff/revenue-licence/api/vehicles", nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	s.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404 forwarded unchanged", rec.Code)
	}
	if rec.Body.String() != `{"error":"no such vehicle"}` {
		t.Errorf("body = %q, want the upstream body forwarded verbatim", rec.Body.String())
	}
}
