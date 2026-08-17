package httpapi

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/akila94/mosip-wso2-citizen-portal-demo/gov-services-api/internal/authmw"
	"github.com/coreos/go-oidc/v3/oidc/oidctest"
	josepkg "github.com/go-jose/go-jose/v4"
)

// testAuthServer is a minimal stand-in for WSO2 IS's discovery document and
// JWKS, used so httpapi's routes can be exercised behind the real
// authmw.RequireAudienceAndScope middleware — never a hand-rolled context
// value — matching how a live deployment actually populates the subject
// and scopes.
type testAuthServer struct {
	srv   *httptest.Server
	priv  *rsa.PrivateKey
	keyID string
}

func newTestAuthServer(t *testing.T) *testAuthServer {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generating RSA key: %v", err)
	}
	as := &testAuthServer{priv: priv, keyID: "test-key-1"}

	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, r *http.Request) {
		issuer := as.srv.URL
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
			{Key: as.priv.Public(), KeyID: as.keyID, Algorithm: "RS256", Use: "sig"},
		}}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(set)
	})

	as.srv = httptest.NewServer(mux)
	t.Cleanup(as.srv.Close)
	return as
}

// verifier builds an authmw.Verifier backed by this stub server's own JWKS.
func (as *testAuthServer) verifier(t *testing.T) *authmw.Verifier {
	t.Helper()
	keySet, err := authmw.NewKeySet(context.Background(), as.srv.URL, as.srv.Client())
	if err != nil {
		t.Fatalf("authmw.NewKeySet: %v", err)
	}
	return authmw.NewVerifier(keySet, as.srv.URL)
}

// sign builds a signed access token for sub, aud and scope, using this
// server's own signing key.
func (as *testAuthServer) sign(t *testing.T, sub, aud, scope string) string {
	t.Helper()
	claims := map[string]any{
		"iss":       as.srv.URL,
		"aud":       aud,
		"sub":       sub,
		"client_id": aud,
		"scope":     scope,
		"exp":       time.Now().Add(time.Hour).Unix(),
	}
	raw, err := json.Marshal(claims)
	if err != nil {
		t.Fatalf("marshaling claims: %v", err)
	}
	return oidctest.SignIDToken(as.priv, as.keyID, "RS256", string(raw))
}
