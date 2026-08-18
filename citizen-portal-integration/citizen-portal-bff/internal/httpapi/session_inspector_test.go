package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/akila94/mosip-wso2-citizen-portal-demo/citizen-portal-bff/internal/session"
)

// inspect drives GET /bff/{app}/api/session-inspector through the real
// router (middleware chain included) for the given session.
func inspect(t *testing.T, s *Server, app *AppRoute, sess session.AuthSession) (*httptest.ResponseRecorder, sessionInspectorView) {
	t.Helper()

	cookie := sessionCookieFor(t, s, app, sess)
	req := httptest.NewRequest(http.MethodGet, app.RoutePrefix+"/api/session-inspector", nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	s.Router().ServeHTTP(rec, req)

	var body sessionInspectorView
	if rec.Code == http.StatusOK {
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatalf("decoding body %q: %v", rec.Body.String(), err)
		}
	}
	return rec, body
}

func testInspectorSession(appKey string) session.AuthSession {
	return session.AuthSession{
		AppKey:    appKey,
		User:      session.User{Sub: "psut-sub-42", Name: "Jane Citizen"},
		Sid:       "is-session-1",
		Acr:       "mosip:idp:acr:generated-code",
		Amr:       []string{"EsignetOIDCAuthenticator"},
		AuthTime:  time.Unix(1755000000, 0).UTC(),
		ExpiresAt: time.Unix(1755003600, 0).UTC(),
		IDTokenClaims: map[string]any{
			"sub": "psut-sub-42",
			"aud": appKey + "-client-id",
			"iss": "https://localhost:9443/oauth2/token",
		},
		RawIDToken:  "raw-id-token-value",
		AccessToken: "access-token-value",
	}
}

func TestSessionInspectorReturnsTheFactsBehindTheSession(t *testing.T) {
	up := &fakeUpstream{}
	s, apps := newDataRouteTestServer(t, up)
	app := apps["driving-licence"]

	rec, body := inspect(t, s, app, testInspectorSession("driving-licence"))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", rec.Code, rec.Body.String())
	}

	if body.AppKey != "driving-licence" {
		t.Errorf("appKey = %q", body.AppKey)
	}
	if body.ClientID != "dl-client-id" {
		t.Errorf("clientId = %q, want dl-client-id", body.ClientID)
	}
	if body.ClientLabel != "Driving Licence Service" {
		t.Errorf("clientLabel = %q", body.ClientLabel)
	}
	if body.Subject != "psut-sub-42" {
		t.Errorf("subject = %q", body.Subject)
	}
	if body.Sid != "is-session-1" {
		t.Errorf("sid = %q", body.Sid)
	}
	if body.Acr != "mosip:idp:acr:generated-code" {
		t.Errorf("acr = %q", body.Acr)
	}
	if len(body.Amr) != 1 || body.Amr[0] != "EsignetOIDCAuthenticator" {
		t.Errorf("amr = %v", body.Amr)
	}
	if body.AuthTime != 1755000000 || body.ExpiresAt != 1755003600 {
		t.Errorf("authTime/expiresAt = %d/%d", body.AuthTime, body.ExpiresAt)
	}
	if body.AssuranceLevel != string(session.AssuranceSubstantial) {
		t.Errorf("assuranceLevel = %q, want substantial", body.AssuranceLevel)
	}
	if body.IDP != session.IdentityProviderESignet {
		t.Errorf("idp = %q, want %q", body.IDP, session.IdentityProviderESignet)
	}
}

func TestSessionInspectorNeverCallsGovServicesAPI(t *testing.T) {
	up := &fakeUpstream{}
	s, apps := newDataRouteTestServer(t, up)

	if rec, _ := inspect(t, s, apps["portal"], testInspectorSession("portal")); rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if up.lastMethod != "" {
		t.Errorf("upstream method called = %q — the inspector is computed from BFF session state alone", up.lastMethod)
	}
}

func TestSessionInspectorIsRegisteredForAllThreeApps(t *testing.T) {
	up := &fakeUpstream{}
	s, apps := newDataRouteTestServer(t, up)

	for key, app := range apps {
		rec, body := inspect(t, s, app, testInspectorSession(key))
		if rec.Code != http.StatusOK {
			t.Errorf("%s: status = %d, want 200, body=%s", key, rec.Code, rec.Body.String())
			continue
		}
		if body.AppKey != key {
			t.Errorf("%s: appKey = %q", key, body.AppKey)
		}
	}
}

func TestSessionInspectorRequiresASession(t *testing.T) {
	up := &fakeUpstream{}
	s, _ := newDataRouteTestServer(t, up)

	req := httptest.NewRequest(http.MethodGet, "/bff/portal/api/session-inspector", nil)
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
		t.Error("expected the shared authenticated=false 401 body")
	}
}

// --- clientsInSession: the SSO proof. ---

func TestSessionInspectorListsEveryAppSharingTheSameSid(t *testing.T) {
	up := &fakeUpstream{}
	s, apps := newDataRouteTestServer(t, up)

	// Application B logged in over the same IS session; the portal did not.
	if _, err := s.Sessions.CreateSession(session.AuthSession{AppKey: "revenue-licence", Sid: "is-session-1"}); err != nil {
		t.Fatalf("creating the second app's session: %v", err)
	}
	if _, err := s.Sessions.CreateSession(session.AuthSession{AppKey: "portal", Sid: "a-different-is-session"}); err != nil {
		t.Fatalf("creating the unrelated session: %v", err)
	}

	_, body := inspect(t, s, apps["driving-licence"], testInspectorSession("driving-licence"))

	want := []clientInSessionView{
		{AppKey: "driving-licence", AppName: "Driving Licence Service"},
		{AppKey: "revenue-licence", AppName: "Vehicle Revenue Licence"},
	}
	if len(body.ClientsInSession) != len(want) {
		t.Fatalf("clientsInSession = %+v, want %+v", body.ClientsInSession, want)
	}
	for i, w := range want {
		if body.ClientsInSession[i] != w {
			t.Errorf("clientsInSession[%d] = %+v, want %+v (ordered by app key so the UI does not flicker)", i, body.ClientsInSession[i], w)
		}
	}
}

func TestSessionInspectorDeduplicatesRepeatedSessionsForOneApp(t *testing.T) {
	up := &fakeUpstream{}
	s, apps := newDataRouteTestServer(t, up)

	// A second login for the same app over the same IS session leaves two
	// store entries; the inspector must still list the app once.
	if _, err := s.Sessions.CreateSession(session.AuthSession{AppKey: "driving-licence", Sid: "is-session-1"}); err != nil {
		t.Fatalf("creating the duplicate session: %v", err)
	}

	_, body := inspect(t, s, apps["driving-licence"], testInspectorSession("driving-licence"))
	if len(body.ClientsInSession) != 1 {
		t.Fatalf("clientsInSession = %+v, want exactly one entry", body.ClientsInSession)
	}
}

func TestSessionInspectorWithoutASidListsOnlyTheCallingApp(t *testing.T) {
	up := &fakeUpstream{}
	s, apps := newDataRouteTestServer(t, up)

	// Another app's session, also without a sid: an ID token with no sid
	// cannot be correlated, so these two must never be grouped.
	if _, err := s.Sessions.CreateSession(session.AuthSession{AppKey: "revenue-licence"}); err != nil {
		t.Fatalf("creating the sid-less session: %v", err)
	}

	sess := testInspectorSession("portal")
	sess.Sid = ""
	_, body := inspect(t, s, apps["portal"], sess)

	if len(body.ClientsInSession) != 1 || body.ClientsInSession[0].AppKey != "portal" {
		t.Fatalf("clientsInSession = %+v, want only the calling app", body.ClientsInSession)
	}
}

// --- releasedClaims: the per-client difference, minus the handshake noise. ---

func TestSessionInspectorReturnsTheFullReleasedClaimSet(t *testing.T) {
	up := &fakeUpstream{}
	s, apps := newDataRouteTestServer(t, up)

	sess := testInspectorSession("revenue-licence")
	sess.IDTokenClaims = map[string]any{
		"sub":       "psut-sub-42",
		"aud":       "vrl-client-id",
		"iss":       "https://localhost:9443/oauth2/token",
		"name":      "Jane Citizen",
		"exp":       float64(1755003600),
		"amr":       []any{"EsignetOIDCAuthenticator"},
		"auth_time": float64(1755000000),
	}
	_, body := inspect(t, s, apps["revenue-licence"], sess)

	for _, claim := range []string{"sub", "aud", "iss", "name", "exp", "amr", "auth_time"} {
		if _, ok := body.ReleasedClaims[claim]; !ok {
			t.Errorf("releasedClaims is missing %q: %+v", claim, body.ReleasedClaims)
		}
	}
	if body.ReleasedClaims["aud"] != "vrl-client-id" {
		t.Errorf("releasedClaims[aud] = %v, want this client's own id", body.ReleasedClaims["aud"])
	}
}

func TestSessionInspectorNeverReturnsTokensOrTheNonce(t *testing.T) {
	up := &fakeUpstream{}
	s, apps := newDataRouteTestServer(t, up)

	sess := testInspectorSession("portal")
	sess.IDTokenClaims["nonce"] = "per-transaction-handshake-value"
	rec, body := inspect(t, s, apps["portal"], sess)

	if _, ok := body.ReleasedClaims["nonce"]; ok {
		t.Error("nonce must be stripped — it is a per-transaction handshake value with no business meaning")
	}
	for _, secret := range []string{"raw-id-token-value", "access-token-value", "per-transaction-handshake-value", "id_token", "access_token", "refresh_token"} {
		if strings.Contains(rec.Body.String(), secret) {
			t.Errorf("inspector response leaked %q: %s", secret, rec.Body.String())
		}
	}
}

func TestSessionInspectorAlwaysReturnsAnObjectForReleasedClaims(t *testing.T) {
	up := &fakeUpstream{}
	s, apps := newDataRouteTestServer(t, up)

	sess := testInspectorSession("portal")
	sess.IDTokenClaims = nil
	rec, _ := inspect(t, s, apps["portal"], sess)

	// A null would force the SPA to null-check on every render.
	if strings.Contains(rec.Body.String(), `"releasedClaims":null`) {
		t.Errorf("releasedClaims must serialize as {} when unknown: %s", rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), `"clientsInSession":null`) {
		t.Errorf("clientsInSession must serialize as an array: %s", rec.Body.String())
	}
}

func TestSessionInspectorOmitsAcrWhenISDidNotResolveOne(t *testing.T) {
	up := &fakeUpstream{}
	s, apps := newDataRouteTestServer(t, up)

	sess := testInspectorSession("portal")
	sess.Acr = ""
	rec, _ := inspect(t, s, apps["portal"], sess)

	if strings.Contains(rec.Body.String(), `"acr"`) {
		t.Errorf("acr must be omitted when empty: %s", rec.Body.String())
	}
}

// --- The stored claim map is bounded. ---

func TestCallbackStoresABoundedNonceFreeClaimMap(t *testing.T) {
	raw := map[string]any{"nonce": "handshake", "sub": "sub-1"}
	for i := 0; i < maxReleasedClaims*2; i++ {
		raw[string(rune('a'+i%26))+string(rune('a'+i/26))] = i
	}

	stored := releasedClaims(raw)
	if len(stored) > maxReleasedClaims {
		t.Fatalf("stored %d claims, want at most %d", len(stored), maxReleasedClaims)
	}
	if _, ok := stored["nonce"]; ok {
		t.Error("nonce must never be stored")
	}
}

func TestReleasedClaimsIsADefensiveCopy(t *testing.T) {
	raw := map[string]any{"sub": "sub-1"}
	stored := releasedClaims(raw)
	raw["injected-later"] = "value"

	if _, ok := stored["injected-later"]; ok {
		t.Error("releasedClaims must copy, not alias, the caller's map")
	}
}
