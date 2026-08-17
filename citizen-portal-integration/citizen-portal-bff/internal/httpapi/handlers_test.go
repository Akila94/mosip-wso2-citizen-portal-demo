package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/akila94/mosip-wso2-citizen-portal-demo/citizen-portal-bff/internal/oidcrp"
	"github.com/akila94/mosip-wso2-citizen-portal-demo/citizen-portal-bff/internal/session"
	"github.com/coreos/go-oidc/v3/oidc"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func newTestServer(t *testing.T, client OIDCClient) (*Server, *AppRoute) {
	t.Helper()
	app := &AppRoute{
		Key:                   "portal",
		RoutePrefix:           "/bff/portal",
		ReturnToPrefix:        "/",
		Client:                client,
		SessionCookieName:     "cp_sid",
		LoginTxnCookieName:    "cp_txn",
		CSRFCookieName:        "cp_csrf",
		PostLogoutRedirectURI: "http://localhost:8090/",
		ClientID:              "portal-client-id",
		AppName:               "Citizen Portal",
	}
	mgr := session.NewManager(session.Config{
		MaxSessions: 100,
		LoginTxnTTL: time.Minute,
		IdleTimeout: time.Minute,
	})
	t.Cleanup(mgr.Close)

	s := NewServer(map[string]*AppRoute{"portal": app}, mgr, false, time.Minute, discardLogger(), &fakeUpstream{})
	return s, app
}

func TestHandleLoginRedirectsWithCookieAndPKCE(t *testing.T) {
	s, app := newTestServer(t, &fakeClient{})

	req := httptest.NewRequest(http.MethodGet, "/bff/portal/login?returnTo=/timeline", nil)
	rec := httptest.NewRecorder()
	s.handleLogin(app)(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d, want 302", rec.Code)
	}
	loc := rec.Header().Get("Location")
	if loc == "" {
		t.Fatal("no Location header set")
	}

	var txnCookie *http.Cookie
	for _, c := range rec.Result().Cookies() {
		if c.Name == app.LoginTxnCookieName {
			txnCookie = c
		}
	}
	if txnCookie == nil {
		t.Fatal("login transaction cookie not set")
	}
	if !txnCookie.HttpOnly {
		t.Error("login transaction cookie must be HttpOnly")
	}
	if txnCookie.Path != app.RoutePrefix {
		t.Errorf("login transaction cookie Path = %q, want %q", txnCookie.Path, app.RoutePrefix)
	}
}

func TestHandleLoginRejectsInvalidReturnTo(t *testing.T) {
	s, app := newTestServer(t, &fakeClient{})

	req := httptest.NewRequest(http.MethodGet, "/bff/portal/login?returnTo=https://evil.example.com", nil)
	rec := httptest.NewRecorder()
	s.handleLogin(app)(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

// loginThenCallback drives handleLogin then handleCallback with the state
// and login-transaction cookie handleLogin actually produced, so callback
// tests exercise the real cookie/state binding rather than a hand-built
// transaction.
func loginThenCallback(t *testing.T, s *Server, app *AppRoute, mutateQuery func(v url.Values)) *httptest.ResponseRecorder {
	t.Helper()

	loginReq := httptest.NewRequest(http.MethodGet, "/bff/portal/login?returnTo=/timeline", nil)
	loginRec := httptest.NewRecorder()
	s.handleLogin(app)(loginRec, loginReq)

	loc, err := url.Parse(loginRec.Header().Get("Location"))
	if err != nil {
		t.Fatalf("parsing login redirect: %v", err)
	}
	state := loc.Query().Get("state")

	var txnCookie *http.Cookie
	for _, c := range loginRec.Result().Cookies() {
		if c.Name == app.LoginTxnCookieName {
			txnCookie = c
		}
	}
	if txnCookie == nil {
		t.Fatal("login transaction cookie not set by handleLogin")
	}

	q := url.Values{}
	q.Set("code", "auth-code-1")
	q.Set("state", state)
	if mutateQuery != nil {
		mutateQuery(q)
	}

	cbReq := httptest.NewRequest(http.MethodGet, "/bff/portal/callback?"+q.Encode(), nil)
	cbReq.AddCookie(txnCookie)
	cbRec := httptest.NewRecorder()
	s.handleCallback(app)(cbRec, cbReq)
	return cbRec
}

func TestHandleCallbackHappyPathSetsSessionCookie(t *testing.T) {
	client := &fakeClient{}
	s, app := newTestServer(t, client)

	rec := loginThenCallback(t, s, app, nil)

	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d, want 302, body=%s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Location"); got != "/timeline" {
		t.Errorf("redirect Location = %q, want /timeline", got)
	}

	var sessionCookie, csrfCookie *http.Cookie
	for _, c := range rec.Result().Cookies() {
		switch c.Name {
		case app.SessionCookieName:
			sessionCookie = c
		case app.CSRFCookieName:
			csrfCookie = c
		}
	}
	if sessionCookie == nil || !sessionCookie.HttpOnly {
		t.Fatal("expected an HttpOnly session cookie")
	}
	if csrfCookie == nil || csrfCookie.HttpOnly {
		t.Fatal("expected a non-HttpOnly CSRF cookie")
	}
	if client.lastExchangeCode != "auth-code-1" {
		t.Errorf("Exchange code = %q", client.lastExchangeCode)
	}
	if client.lastVerifyRawIDToken != "raw-id-token" {
		t.Errorf("VerifyIDToken rawIDToken = %q", client.lastVerifyRawIDToken)
	}

	if sessionCookie == nil {
		t.Fatal("expected a session cookie")
	}
	sess, ok := s.Sessions.GetSession(sessionCookie.Value)
	if !ok {
		t.Fatal("expected the created session to be retrievable")
	}
	if sess.AccessToken != "at-1" {
		t.Errorf("AuthSession.AccessToken = %q, want %q", sess.AccessToken, "at-1")
	}
	if !sess.AccessTokenExpiresAt.Equal(testAccessTokenExpiry) {
		t.Errorf("AuthSession.AccessTokenExpiresAt = %v, want %v", sess.AccessTokenExpiresAt, testAccessTokenExpiry)
	}
}

func TestHandleCallbackRejectsStateMismatch(t *testing.T) {
	s, app := newTestServer(t, &fakeClient{})

	rec := loginThenCallback(t, s, app, func(v url.Values) { v.Set("state", "tampered-state") })
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestHandleCallbackRejectsMissingTxnCookie(t *testing.T) {
	s, app := newTestServer(t, &fakeClient{})

	req := httptest.NewRequest(http.MethodGet, "/bff/portal/callback?code=abc&state=xyz", nil)
	rec := httptest.NewRecorder()
	s.handleCallback(app)(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestHandleCallbackRejectsReplayedTxn(t *testing.T) {
	client := &fakeClient{}
	s, app := newTestServer(t, client)

	loginReq := httptest.NewRequest(http.MethodGet, "/bff/portal/login?returnTo=/timeline", nil)
	loginRec := httptest.NewRecorder()
	s.handleLogin(app)(loginRec, loginReq)
	loc, _ := url.Parse(loginRec.Header().Get("Location"))
	state := loc.Query().Get("state")
	var txnCookie *http.Cookie
	for _, c := range loginRec.Result().Cookies() {
		if c.Name == app.LoginTxnCookieName {
			txnCookie = c
		}
	}

	makeReq := func() *http.Request {
		q := url.Values{}
		q.Set("code", "auth-code-1")
		q.Set("state", state)
		r := httptest.NewRequest(http.MethodGet, "/bff/portal/callback?"+q.Encode(), nil)
		r.AddCookie(txnCookie)
		return r
	}

	first := httptest.NewRecorder()
	s.handleCallback(app)(first, makeReq())
	if first.Code != http.StatusFound {
		t.Fatalf("first callback status = %d, want 302", first.Code)
	}

	second := httptest.NewRecorder()
	s.handleCallback(app)(second, makeReq())
	if second.Code != http.StatusBadRequest {
		t.Fatalf("replayed callback status = %d, want 400 (transaction must be single-use)", second.Code)
	}
}

func TestHandleCallbackPropagatesIdentityProviderError(t *testing.T) {
	s, app := newTestServer(t, &fakeClient{})

	req := httptest.NewRequest(http.MethodGet, "/bff/portal/callback?error=access_denied&error_description=user+cancelled", nil)
	rec := httptest.NewRecorder()
	s.handleCallback(app)(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestHandleCallbackRejectsExchangeFailure(t *testing.T) {
	s, app := newTestServer(t, &fakeClient{exchangeErr: errFakeExchange})

	rec := loginThenCallback(t, s, app, nil)
	if rec.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, want 502", rec.Code)
	}
}

func TestHandleCallbackRejectsIDTokenVerificationFailure(t *testing.T) {
	s, app := newTestServer(t, &fakeClient{verifyErr: errors.New("bad signature")})

	rec := loginThenCallback(t, s, app, nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestHandleSessionUnauthenticatedWithoutCookie(t *testing.T) {
	s, app := newTestServer(t, &fakeClient{})

	req := httptest.NewRequest(http.MethodGet, "/bff/portal/session", nil)
	rec := httptest.NewRecorder()
	s.handleSession(app)(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
	var body sessionView
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decoding body: %v", err)
	}
	if body.Authenticated {
		t.Error("expected authenticated=false")
	}
}

func TestHandleSessionReturnsProjectionAfterLogin(t *testing.T) {
	client := &fakeClient{verifyClaims: &oidcrp.Claims{
		Sub: "sub-42", Name: "Jane Citizen", Sid: "sid-42",
		Amr: []string{"EsignetOIDCAuthenticator"},
	}}
	s, app := newTestServer(t, client)

	cbRec := loginThenCallback(t, s, app, nil)
	var sessionCookie *http.Cookie
	for _, c := range cbRec.Result().Cookies() {
		if c.Name == app.SessionCookieName {
			sessionCookie = c
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/bff/portal/session", nil)
	req.AddCookie(sessionCookie)
	rec := httptest.NewRecorder()
	s.handleSession(app)(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", rec.Code, rec.Body.String())
	}
	var body sessionView
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decoding body: %v", err)
	}
	if !body.Authenticated || body.User.Sub != "sub-42" || body.Sid != "sid-42" {
		t.Errorf("unexpected session view: %+v", body)
	}
	if body.ClientID != "portal-client-id" || body.AppName != "Citizen Portal" {
		t.Errorf("ClientID/AppName = %q/%q, want portal-client-id/Citizen Portal", body.ClientID, body.AppName)
	}
	if body.AssuranceLevel != string(session.AssuranceSubstantial) {
		t.Errorf("AssuranceLevel = %q, want substantial", body.AssuranceLevel)
	}
	if raw := rec.Body.String(); containsAny(raw, "raw-id-token", "access_token", "refresh_token") {
		t.Errorf("session response leaked token material: %s", raw)
	}
}

func containsAny(s string, subs ...string) bool {
	for _, sub := range subs {
		if len(sub) > 0 && indexOf(s, sub) >= 0 {
			return true
		}
	}
	return false
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func TestHandleLogoutRejectsWithoutCSRFToken(t *testing.T) {
	s, app := newTestServer(t, &fakeClient{})

	req := httptest.NewRequest(http.MethodPost, "/bff/portal/logout", nil)
	rec := httptest.NewRecorder()
	s.handleLogout(app)(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rec.Code)
	}
}

func TestHandleLogoutSucceedsWithMatchingCSRFToken(t *testing.T) {
	client := &fakeClient{}
	s, app := newTestServer(t, client)

	cbRec := loginThenCallback(t, s, app, nil)
	var sessionCookie, csrfCookie *http.Cookie
	for _, c := range cbRec.Result().Cookies() {
		switch c.Name {
		case app.SessionCookieName:
			sessionCookie = c
		case app.CSRFCookieName:
			csrfCookie = c
		}
	}

	req := httptest.NewRequest(http.MethodPost, "/bff/portal/logout", nil)
	req.AddCookie(sessionCookie)
	req.AddCookie(csrfCookie)
	req.Header.Set(csrfHeaderName, csrfCookie.Value)
	rec := httptest.NewRecorder()
	s.handleLogout(app)(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", rec.Code, rec.Body.String())
	}

	sessionReq := httptest.NewRequest(http.MethodGet, "/bff/portal/session", nil)
	sessionReq.AddCookie(sessionCookie)
	sessionRec := httptest.NewRecorder()
	s.handleSession(app)(sessionRec, sessionReq)
	if sessionRec.Code != http.StatusUnauthorized {
		t.Fatal("expected the session to be destroyed after logout")
	}
}

func TestHandleLogoutRejectsMismatchedCSRFToken(t *testing.T) {
	client := &fakeClient{}
	s, app := newTestServer(t, client)

	cbRec := loginThenCallback(t, s, app, nil)
	var sessionCookie, csrfCookie *http.Cookie
	for _, c := range cbRec.Result().Cookies() {
		switch c.Name {
		case app.SessionCookieName:
			sessionCookie = c
		case app.CSRFCookieName:
			csrfCookie = c
		}
	}

	req := httptest.NewRequest(http.MethodPost, "/bff/portal/logout", nil)
	req.AddCookie(sessionCookie)
	req.AddCookie(csrfCookie)
	req.Header.Set(csrfHeaderName, "wrong-token")
	rec := httptest.NewRecorder()
	s.handleLogout(app)(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rec.Code)
	}
}

func TestHandleBackchannelLogoutDestroysSessionsBySid(t *testing.T) {
	client := &fakeClient{}
	s, app := newTestServer(t, client)

	// Create two sessions sharing sid "shared-sid" directly via the
	// manager, standing in for two apps sharing one IdP session.
	k1, _ := s.Sessions.CreateSession(session.AuthSession{AppKey: "portal", Sid: "shared-sid"})
	k2, _ := s.Sessions.CreateSession(session.AuthSession{AppKey: "portal", Sid: "shared-sid"})

	client.logoutToken = &oidc.LogoutToken{SessionID: "shared-sid", TokenID: "jti-1"}

	form := url.Values{}
	form.Set("logout_token", "irrelevant-because-fake-verifier")
	req := httptest.NewRequest(http.MethodPost, "/bff/portal/backchannel-logout", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rec := httptest.NewRecorder()
	s.handleBackchannelLogout(app)(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", rec.Code, rec.Body.String())
	}
	if _, ok := s.Sessions.GetSession(k1); ok {
		t.Error("session k1 should have been destroyed")
	}
	if _, ok := s.Sessions.GetSession(k2); ok {
		t.Error("session k2 should have been destroyed")
	}
}

func TestHandleBackchannelLogoutRejectsReplayedToken(t *testing.T) {
	client := &fakeClient{}
	s, app := newTestServer(t, client)
	client.logoutToken = &oidc.LogoutToken{SessionID: "shared-sid", TokenID: "jti-dup"}

	makeReq := func() *http.Request {
		form := url.Values{}
		form.Set("logout_token", "irrelevant")
		req := httptest.NewRequest(http.MethodPost, "/bff/portal/backchannel-logout", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		return req
	}

	first := httptest.NewRecorder()
	s.handleBackchannelLogout(app)(first, makeReq())
	if first.Code != http.StatusOK {
		t.Fatalf("first call status = %d, want 200", first.Code)
	}

	second := httptest.NewRecorder()
	s.handleBackchannelLogout(app)(second, makeReq())
	if second.Code != http.StatusBadRequest {
		t.Fatalf("replayed call status = %d, want 400", second.Code)
	}
}

func TestHandleBackchannelLogoutRejectsMissingSid(t *testing.T) {
	client := &fakeClient{logoutToken: &oidc.LogoutToken{TokenID: "jti-no-sid"}}
	s, app := newTestServer(t, client)

	form := url.Values{}
	form.Set("logout_token", "irrelevant")
	req := httptest.NewRequest(http.MethodPost, "/bff/portal/backchannel-logout", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	s.handleBackchannelLogout(app)(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestHandleBackchannelLogoutRejectsInvalidToken(t *testing.T) {
	client := &fakeClient{logoutErr: errors.New("bad signature")}
	s, app := newTestServer(t, client)

	form := url.Values{}
	form.Set("logout_token", "garbage")
	req := httptest.NewRequest(http.MethodPost, "/bff/portal/backchannel-logout", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	s.handleBackchannelLogout(app)(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}
