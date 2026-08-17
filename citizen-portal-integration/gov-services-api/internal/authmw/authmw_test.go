package authmw

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/coreos/go-oidc/v3/oidc/oidctest"
	josepkg "github.com/go-jose/go-jose/v4"
)

// testIS is a minimal stand-in for WSO2 IS's discovery document and JWKS,
// used so authmw's verification logic (signature/issuer/expiry/audience/
// scope checks) is exercised against a real HTTP round trip and real JWS
// signatures, without a live IS instance. Mirrors
// citizen-portal-bff/internal/oidcrp/rp_test.go's testIDP, adapted for
// access-token claims instead of ID-token claims.
type testIS struct {
	srv   *httptest.Server
	priv  *rsa.PrivateKey
	keyID string
}

func newTestIS(t *testing.T) *testIS {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generating RSA key: %v", err)
	}
	is := &testIS{priv: priv, keyID: "test-key-1"}

	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, r *http.Request) {
		issuer := is.srv.URL
		disc := map[string]any{
			"issuer":                                issuer,
			"authorization_endpoint":                issuer + "/oauth2/authorize",
			"token_endpoint":                        issuer + "/oauth2/token",
			"jwks_uri":                              issuer + "/oauth2/jwks",
			"response_types_supported":              []string{"code"},
			"subject_types_supported":               []string{"public"},
			"id_token_signing_alg_values_supported": []string{"RS256"},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(disc)
	})
	mux.HandleFunc("/oauth2/jwks", func(w http.ResponseWriter, r *http.Request) {
		set := josepkg.JSONWebKeySet{Keys: []josepkg.JSONWebKey{
			{Key: is.priv.Public(), KeyID: is.keyID, Algorithm: "RS256", Use: "sig"},
		}}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(set)
	})

	is.srv = httptest.NewServer(mux)
	t.Cleanup(is.srv.Close)
	return is
}

// sign builds a signed access token with the given claim overrides layered
// on sane defaults, using is's own signing key.
func (is *testIS) sign(t *testing.T, overrides map[string]any) string {
	t.Helper()
	return signWith(t, is.priv, is.keyID, overrides, is.srv.URL)
}

// signWithOtherKey builds a token signed by an unrelated key, to exercise
// the signature-mismatch rejection path.
func signWithOtherKey(t *testing.T, issuer string, overrides map[string]any) string {
	t.Helper()
	otherPriv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generating RSA key: %v", err)
	}
	return signWith(t, otherPriv, "other-key", overrides, issuer)
}

func signWith(t *testing.T, priv *rsa.PrivateKey, keyID string, overrides map[string]any, issuer string) string {
	t.Helper()
	claims := map[string]any{
		"iss":       issuer,
		"aud":       "test-client",
		"sub":       "sub-abc-123",
		"client_id": "test-client",
		"scope":     "openid profile",
		"exp":       time.Now().Add(time.Hour).Unix(),
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
	return oidctest.SignIDToken(priv, keyID, "RS256", string(raw))
}

func newTestVerifier(t *testing.T, is *testIS) *Verifier {
	t.Helper()
	httpClient := is.srv.Client()
	keySet, err := NewKeySet(context.Background(), is.srv.URL, httpClient)
	if err != nil {
		t.Fatalf("NewKeySet: %v", err)
	}
	return NewVerifier(keySet, is.srv.URL)
}

func TestVerifyAcceptsValidToken(t *testing.T) {
	is := newTestIS(t)
	v := newTestVerifier(t, is)

	tok := is.sign(t, nil)
	claims, err := v.Verify(context.Background(), tok)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if claims.Subject != "sub-abc-123" {
		t.Errorf("Subject = %q", claims.Subject)
	}
}

func TestVerifyRejectsWrongSignature(t *testing.T) {
	is := newTestIS(t)
	v := newTestVerifier(t, is)

	tok := signWithOtherKey(t, is.srv.URL, nil)
	if _, err := v.Verify(context.Background(), tok); err == nil {
		t.Fatal("expected rejection of a token signed by an unrelated key")
	}
}

func TestVerifyRejectsWrongIssuer(t *testing.T) {
	is := newTestIS(t)
	v := newTestVerifier(t, is)

	tok := is.sign(t, map[string]any{"iss": "https://not-the-real-is.example"})
	if _, err := v.Verify(context.Background(), tok); err == nil {
		t.Fatal("expected rejection of a token with a mismatched issuer")
	}
}

func TestVerifyRejectsExpiredToken(t *testing.T) {
	is := newTestIS(t)
	v := newTestVerifier(t, is)

	tok := is.sign(t, map[string]any{"exp": time.Now().Add(-time.Hour).Unix()})
	if _, err := v.Verify(context.Background(), tok); err == nil {
		t.Fatal("expected rejection of an expired token")
	}
}

func TestVerifyRejectsNotYetValidToken(t *testing.T) {
	is := newTestIS(t)
	v := newTestVerifier(t, is)

	tok := is.sign(t, map[string]any{"nbf": time.Now().Add(time.Hour).Unix()})
	if _, err := v.Verify(context.Background(), tok); err == nil {
		t.Fatal("expected rejection of a not-yet-valid token")
	}
}

func TestVerifyAcceptsArrayAudience(t *testing.T) {
	is := newTestIS(t)
	v := newTestVerifier(t, is)

	tok := is.sign(t, map[string]any{"aud": []string{"test-client", "another-aud"}})
	claims, err := v.Verify(context.Background(), tok)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if len(claims.Audience) != 2 {
		t.Errorf("Audience = %+v", claims.Audience)
	}
}

// TestRequireAudienceAndScopeRejectsWrongAudience is the single most
// important test in this package — it's the literal audience-separation
// the whole milestone is about: a token minted for Application A's client
// ID must be rejected by Application B's router.
func TestRequireAudienceAndScopeRejectsWrongAudience(t *testing.T) {
	is := newTestIS(t)
	v := newTestVerifier(t, is)

	tok := is.sign(t, map[string]any{"aud": "application-a-client-id"})
	rr := doRequest(t, v, []string{"application-b-client-id"}, "", tok)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rr.Code)
	}
}

func TestRequireAudienceAndScopeAcceptsMatchingAudience(t *testing.T) {
	is := newTestIS(t)
	v := newTestVerifier(t, is)

	tok := is.sign(t, map[string]any{"aud": "application-b-client-id"})
	rr := doRequest(t, v, []string{"application-b-client-id"}, "", tok)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body: %s", rr.Code, rr.Body.String())
	}
}

func TestRequireAudienceAndScopeRejectsMissingScope(t *testing.T) {
	is := newTestIS(t)
	v := newTestVerifier(t, is)

	tok := is.sign(t, map[string]any{"scope": "openid profile"})
	rr := doRequest(t, v, []string{"test-client"}, "vehicle_registry.read", tok)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rr.Code)
	}
}

func TestRequireAudienceAndScopeAcceptsPresentScopeAmongSeveral(t *testing.T) {
	is := newTestIS(t)
	v := newTestVerifier(t, is)

	tok := is.sign(t, map[string]any{"scope": "openid profile vehicle_registry.read"})
	rr := doRequest(t, v, []string{"test-client"}, "vehicle_registry.read", tok)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body: %s", rr.Code, rr.Body.String())
	}
}

func TestRequireAudienceAndScopeRejectsMissingAuthorizationHeader(t *testing.T) {
	is := newTestIS(t)
	v := newTestVerifier(t, is)

	req := httptest.NewRequest(http.MethodGet, "/citizen/profile", nil)
	rr := httptest.NewRecorder()
	handler := RequireAudienceAndScope(v, []string{"test-client"}, "")(okHandler())
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rr.Code)
	}
}

func TestRequireAudienceAndScopeRejectsMalformedHeader(t *testing.T) {
	is := newTestIS(t)
	v := newTestVerifier(t, is)

	req := httptest.NewRequest(http.MethodGet, "/citizen/profile", nil)
	req.Header.Set("Authorization", "Basic dXNlcjpwYXNz")
	rr := httptest.NewRecorder()
	handler := RequireAudienceAndScope(v, []string{"test-client"}, "")(okHandler())
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rr.Code)
	}
}

func TestRequireAudienceAndScopeRejectsInvalidToken(t *testing.T) {
	is := newTestIS(t)
	v := newTestVerifier(t, is)

	rr := doRequest(t, v, []string{"test-client"}, "", "not-a-real-jwt")
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rr.Code)
	}
}

func TestRequireAudienceAndScopeStoresSubjectAndScopes(t *testing.T) {
	is := newTestIS(t)
	v := newTestVerifier(t, is)

	tok := is.sign(t, map[string]any{"sub": "sub-xyz-789", "scope": "openid profile address"})
	var gotSub string
	var gotScopes []string
	handler := RequireAudienceAndScope(v, []string{"test-client"}, "")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotSub, _ = SubjectFromContext(r.Context())
		gotScopes, _ = ScopesFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/citizen/profile", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if gotSub != "sub-xyz-789" {
		t.Errorf("subject = %q", gotSub)
	}
	if !containsString(gotScopes, "address") {
		t.Errorf("scopes = %+v, want to contain address", gotScopes)
	}
}

func doRequest(t *testing.T, v *Verifier, anyOfAudience []string, requiredScope, token string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/some/route", nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rr := httptest.NewRecorder()
	handler := RequireAudienceAndScope(v, anyOfAudience, requiredScope)(okHandler())
	handler.ServeHTTP(rr, req)
	return rr
}

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

func containsString(haystack []string, needle string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}
	return false
}

func TestAudienceUnmarshalString(t *testing.T) {
	var a audience
	if err := json.Unmarshal([]byte(`"abc"`), &a); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if len(a) != 1 || a[0] != "abc" {
		t.Errorf("audience = %+v", a)
	}
}

func TestAudienceUnmarshalArray(t *testing.T) {
	var a audience
	if err := json.Unmarshal([]byte(`["abc","def"]`), &a); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if len(a) != 2 || a[0] != "abc" || a[1] != "def" {
		t.Errorf("audience = %+v", a)
	}
}

func TestAudienceUnmarshalInvalidShape(t *testing.T) {
	var a audience
	if err := json.Unmarshal([]byte(`42`), &a); err == nil {
		t.Fatal("expected an error for a numeric aud claim")
	}
}
