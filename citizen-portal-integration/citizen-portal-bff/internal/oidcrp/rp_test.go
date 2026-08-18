package oidcrp

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/coreos/go-oidc/v3/oidc/oidctest"
	josepkg "github.com/go-jose/go-jose/v4"
)

// testIDP is a minimal stand-in for WSO2 IS's discovery document and JWKS,
// used so oidcrp's verification logic (issuer/audience/expiry/nonce/
// signature/back-channel-logout checks) is exercised against a real HTTP
// round trip and real JWS signatures, without a live IS instance.
type testIDP struct {
	srv   *httptest.Server
	priv  *rsa.PrivateKey
	keyID string
}

func newTestIDP(t *testing.T) *testIDP {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generating RSA key: %v", err)
	}
	idp := &testIDP{priv: priv, keyID: "test-key-1"}

	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, r *http.Request) {
		issuer := idp.srv.URL
		disc := map[string]any{
			"issuer":                                issuer,
			"authorization_endpoint":                issuer + "/oauth2/authorize",
			"token_endpoint":                        issuer + "/oauth2/token",
			"jwks_uri":                              issuer + "/oauth2/jwks",
			"end_session_endpoint":                  issuer + "/oidc/logout",
			"userinfo_endpoint":                     issuer + "/oauth2/userinfo",
			"response_types_supported":              []string{"code"},
			"subject_types_supported":               []string{"public"},
			"id_token_signing_alg_values_supported": []string{"RS256"},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(disc)
	})
	mux.HandleFunc("/oauth2/jwks", func(w http.ResponseWriter, r *http.Request) {
		set := josepkg.JSONWebKeySet{Keys: []josepkg.JSONWebKey{
			{Key: idp.priv.Public(), KeyID: idp.keyID, Algorithm: "RS256", Use: "sig"},
		}}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(set)
	})

	idp.srv = httptest.NewServer(mux)
	t.Cleanup(idp.srv.Close)
	return idp
}

// sign builds a signed ID token (or back-channel logout token) with the
// given claim overrides layered on sane defaults.
func (idp *testIDP) sign(t *testing.T, overrides map[string]any) string {
	t.Helper()
	claims := map[string]any{
		"iss": idp.srv.URL,
		"aud": "test-client",
		"sub": "user-sub-123",
		"exp": time.Now().Add(time.Hour).Unix(),
		"iat": time.Now().Unix(),
	}
	for k, v := range overrides {
		if v == nil {
			delete(claims, k)
			continue
		}
		claims[k] = v
	}
	raw, err := json.Marshal(claims)
	if err != nil {
		t.Fatalf("marshaling claims: %v", err)
	}
	return oidctest.SignIDToken(idp.priv, idp.keyID, "RS256", string(raw))
}

func newTestRP(t *testing.T, idp *testIDP) *RP {
	t.Helper()
	httpClient := idp.srv.Client()
	provider, err := NewProvider(context.Background(), idp.srv.URL, httpClient)
	if err != nil {
		t.Fatalf("NewProvider: %v", err)
	}
	return NewRP(provider, httpClient, "test-client", "test-secret", "http://localhost:8090/bff/portal/callback", "openid profile email")
}

func TestNewProviderDiscoversEndSessionEndpoint(t *testing.T) {
	idp := newTestIDP(t)
	rp := newTestRP(t, idp)
	if got := rp.provider.EndSessionEndpoint(); got != idp.srv.URL+"/oidc/logout" {
		t.Errorf("EndSessionEndpoint() = %q", got)
	}
}

func TestAuthCodeURLIncludesPKCEAndNonce(t *testing.T) {
	idp := newTestIDP(t)
	rp := newTestRP(t, idp)

	u := rp.AuthCodeURL(AuthRequest{State: "state-1", Nonce: "nonce-1", CodeVerifier: "verifier-abc-1234567890123456789012345678901234"})
	parsed, err := parseQuery(u)
	if err != nil {
		t.Fatalf("parsing auth URL: %v", err)
	}
	if parsed.Get("state") != "state-1" {
		t.Errorf("state = %q", parsed.Get("state"))
	}
	if parsed.Get("nonce") != "nonce-1" {
		t.Errorf("nonce = %q", parsed.Get("nonce"))
	}
	if parsed.Get("code_challenge_method") != "S256" {
		t.Errorf("code_challenge_method = %q", parsed.Get("code_challenge_method"))
	}
	if parsed.Get("code_challenge") == "" {
		t.Error("code_challenge missing")
	}
	if parsed.Has("prompt") {
		t.Errorf("prompt = %q, want it absent unless the caller asked for it", parsed.Get("prompt"))
	}
}

func TestAuthCodeURLCarriesPromptOnlyWhenRequested(t *testing.T) {
	idp := newTestIDP(t)
	rp := newTestRP(t, idp)

	u := rp.AuthCodeURL(AuthRequest{
		State:        "state-1",
		Nonce:        "nonce-1",
		CodeVerifier: "verifier-abc-1234567890123456789012345678901234",
		Prompt:       "login",
	})
	parsed, err := parseQuery(u)
	if err != nil {
		t.Fatalf("parsing auth URL: %v", err)
	}
	if parsed.Get("prompt") != "login" {
		t.Errorf("prompt = %q, want login", parsed.Get("prompt"))
	}
	// The step-up request is otherwise an ordinary authorization request.
	if parsed.Get("code_challenge_method") != "S256" || parsed.Get("nonce") != "nonce-1" {
		t.Errorf("step-up lost PKCE or nonce: %v", parsed)
	}
}

func TestVerifyIDTokenAcceptsValidToken(t *testing.T) {
	idp := newTestIDP(t)
	rp := newTestRP(t, idp)

	tok := idp.sign(t, map[string]any{
		"nonce":     "expected-nonce",
		"amr":       []string{"EsignetOIDCAuthenticator"},
		"sid":       "session-xyz",
		"name":      "Jane Citizen",
		"auth_time": time.Now().Unix(),
	})

	claims, err := rp.VerifyIDToken(context.Background(), tok, "expected-nonce")
	if err != nil {
		t.Fatalf("VerifyIDToken: %v", err)
	}
	if claims.Sub != "user-sub-123" || claims.Name != "Jane Citizen" || claims.Sid != "session-xyz" {
		t.Errorf("unexpected claims: %+v", claims)
	}
	if len(claims.Amr) != 1 || claims.Amr[0] != "EsignetOIDCAuthenticator" {
		t.Errorf("unexpected amr: %+v", claims.Amr)
	}
}

func TestVerifyIDTokenExposesTheCompleteClaimSet(t *testing.T) {
	idp := newTestIDP(t)
	rp := newTestRP(t, idp)

	tok := idp.sign(t, map[string]any{
		"nonce":            "expected-nonce",
		"name":             "Jane Citizen",
		"custom_gov_claim": "issued-by-a-per-client-scope",
	})

	claims, err := rp.VerifyIDToken(context.Background(), tok, "expected-nonce")
	if err != nil {
		t.Fatalf("VerifyIDToken: %v", err)
	}
	// The session inspector shows what each client actually received, so
	// Raw must carry claims Claims has no typed field for.
	if claims.Raw["custom_gov_claim"] != "issued-by-a-per-client-scope" {
		t.Errorf("Raw = %+v, want the client-specific claim", claims.Raw)
	}
	if claims.Raw["aud"] != "test-client" {
		t.Errorf("Raw[aud] = %v, want test-client", claims.Raw["aud"])
	}
	if strings.Contains(fmt.Sprint(claims.Raw), tok) {
		t.Error("Raw must hold decoded claims only, never the token itself")
	}
}

func TestVerifyIDTokenRejectsNonceMismatch(t *testing.T) {
	idp := newTestIDP(t)
	rp := newTestRP(t, idp)

	tok := idp.sign(t, map[string]any{"nonce": "actual-nonce"})
	if _, err := rp.VerifyIDToken(context.Background(), tok, "different-nonce"); err == nil {
		t.Fatal("expected a nonce-mismatch error")
	}
}

func TestVerifyIDTokenRejectsMissingNonce(t *testing.T) {
	idp := newTestIDP(t)
	rp := newTestRP(t, idp)

	tok := idp.sign(t, nil) // no nonce claim at all
	if _, err := rp.VerifyIDToken(context.Background(), tok, "expected-nonce"); err == nil {
		t.Fatal("expected rejection of a token with no nonce claim")
	}
}

func TestVerifyIDTokenRejectsWrongAudience(t *testing.T) {
	idp := newTestIDP(t)
	rp := newTestRP(t, idp)

	tok := idp.sign(t, map[string]any{"nonce": "n1", "aud": "someone-elses-client"})
	if _, err := rp.VerifyIDToken(context.Background(), tok, "n1"); err == nil {
		t.Fatal("expected rejection of a token issued for a different audience")
	}
}

func TestVerifyIDTokenRejectsExpiredToken(t *testing.T) {
	idp := newTestIDP(t)
	rp := newTestRP(t, idp)

	tok := idp.sign(t, map[string]any{"nonce": "n1", "exp": time.Now().Add(-time.Hour).Unix()})
	if _, err := rp.VerifyIDToken(context.Background(), tok, "n1"); err == nil {
		t.Fatal("expected rejection of an expired token")
	}
}

func TestVerifyIDTokenRejectsBadSignature(t *testing.T) {
	idp := newTestIDP(t)
	rp := newTestRP(t, idp)

	tok := idp.sign(t, map[string]any{"nonce": "n1"})
	tampered := tamperMiddleChar(tok)
	if _, err := rp.VerifyIDToken(context.Background(), tampered, "n1"); err == nil {
		t.Fatal("expected rejection of a tampered signature")
	}
}

func TestVerifyLogoutTokenAcceptsValid(t *testing.T) {
	idp := newTestIDP(t)
	rp := newTestRP(t, idp)

	tok := idp.sign(t, map[string]any{
		"sid": "session-xyz",
		"jti": "jti-1",
		"events": map[string]any{
			"http://schemas.openid.net/event/backchannel-logout": map[string]any{},
		},
	})
	lt, err := rp.VerifyLogoutToken(context.Background(), tok)
	if err != nil {
		t.Fatalf("VerifyLogoutToken: %v", err)
	}
	if lt.SessionID != "session-xyz" {
		t.Errorf("SessionID = %q", lt.SessionID)
	}
	if lt.TokenID != "jti-1" {
		t.Errorf("TokenID = %q", lt.TokenID)
	}
}

func TestVerifyLogoutTokenRejectsMissingEvent(t *testing.T) {
	idp := newTestIDP(t)
	rp := newTestRP(t, idp)

	tok := idp.sign(t, map[string]any{"sid": "session-xyz", "jti": "jti-2"})
	if _, err := rp.VerifyLogoutToken(context.Background(), tok); err == nil {
		t.Fatal("expected rejection of a logout token with no backchannel-logout event")
	}
}

func TestVerifyLogoutTokenRejectsNoncePresent(t *testing.T) {
	idp := newTestIDP(t)
	rp := newTestRP(t, idp)

	// A logout token carrying a nonce is a replayed ID token per spec.
	tok := idp.sign(t, map[string]any{
		"sid":   "session-xyz",
		"jti":   "jti-3",
		"nonce": "should-not-be-here",
		"events": map[string]any{
			"http://schemas.openid.net/event/backchannel-logout": map[string]any{},
		},
	})
	if _, err := rp.VerifyLogoutToken(context.Background(), tok); err == nil {
		t.Fatal("expected rejection of a logout token carrying a nonce")
	}
}

func TestLogoutURLBuildsExpectedQuery(t *testing.T) {
	idp := newTestIDP(t)
	rp := newTestRP(t, idp)

	u := rp.LogoutURL("id-token-hint-value", "http://localhost:8090/", "state-1")
	parsed, err := parseQuery(u)
	if err != nil {
		t.Fatalf("parsing logout URL: %v", err)
	}
	if parsed.Get("id_token_hint") != "id-token-hint-value" {
		t.Errorf("id_token_hint = %q", parsed.Get("id_token_hint"))
	}
	if parsed.Get("post_logout_redirect_uri") != "http://localhost:8090/" {
		t.Errorf("post_logout_redirect_uri = %q", parsed.Get("post_logout_redirect_uri"))
	}
	if parsed.Get("client_id") != "test-client" {
		t.Errorf("client_id = %q", parsed.Get("client_id"))
	}
	if parsed.Get("state") != "state-1" {
		t.Errorf("state = %q", parsed.Get("state"))
	}
}

func parseQuery(rawURL string) (url.Values, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}
	return u.Query(), nil
}

// tamperMiddleChar corrupts a JWS by flipping one character in the middle
// of the signature segment (the last dot-separated part) — a full base64
// character there always encodes 6 significant bits, unlike the final
// character, whose top bits may be padding a signature-verification library
// legitimately ignores.
func tamperMiddleChar(jws string) string {
	dot := strings.LastIndexByte(jws, '.')
	sig := jws[dot+1:]
	mid := len(sig) / 2
	var repl byte = 'A'
	if sig[mid] == 'A' {
		repl = 'B'
	}
	return jws[:dot+1] + sig[:mid] + string(repl) + sig[mid+1:]
}
