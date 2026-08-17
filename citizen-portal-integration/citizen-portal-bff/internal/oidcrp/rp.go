package oidcrp

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/akila94/mosip-wso2-citizen-portal-demo/citizen-portal-bff/internal/security"
	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

// RP is one app's registered OIDC relying-party configuration: its client
// credentials, redirect URI and scopes, bound to a shared discovered
// Provider. Each of the three apps (portal, driving-licence, revenue-licence)
// gets its own RP over the same Provider, since they all authenticate
// against the same WSO2 IS instance.
type RP struct {
	provider   *Provider
	httpClient *http.Client
	oauth2Cfg  oauth2.Config
	verifier   *oidc.IDTokenVerifier
	logoutVer  *oidc.IDTokenVerifier
}

// NewRP builds an RP for one app's registered client.
func NewRP(provider *Provider, httpClient *http.Client, clientID, clientSecret, redirectURL, scopes string) *RP {
	oauth2Cfg := oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Endpoint:     provider.inner.Endpoint(),
		Scopes:       strings.Fields(scopes),
	}
	verifier := provider.inner.Verifier(&oidc.Config{ClientID: clientID})
	// Back-channel logout tokens are audience-checked against the same
	// client ID, but must never carry a nonce (enforced by VerifyLogout
	// itself, per the OIDC Back-Channel Logout 1.0 spec).
	logoutVer := provider.inner.Verifier(&oidc.Config{ClientID: clientID})

	return &RP{
		provider:   provider,
		httpClient: httpClient,
		oauth2Cfg:  oauth2Cfg,
		verifier:   verifier,
		logoutVer:  logoutVer,
	}
}

// AuthRequest is everything needed to build an authorization redirect and
// later verify its response.
type AuthRequest struct {
	State        string
	Nonce        string
	CodeVerifier string
}

// AuthCodeURL builds the `/authorize` redirect URL for req, always with an
// S256 PKCE challenge — mandatory PKCE must also be turned on for the
// application in the IS Console (see MANUAL-STEPS.md); this ensures the
// BFF's own requests are compliant regardless of that Console setting.
func (rp *RP) AuthCodeURL(req AuthRequest) string {
	challenge := security.Challenge(req.CodeVerifier)
	return rp.oauth2Cfg.AuthCodeURL(
		req.State,
		oidc.Nonce(req.Nonce),
		oauth2.SetAuthURLParam("code_challenge", challenge),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
	)
}

// Exchange redeems an authorization code for tokens, presenting the PKCE
// verifier that matches the challenge sent in AuthCodeURL.
func (rp *RP) Exchange(ctx context.Context, code, codeVerifier string) (*oauth2.Token, error) {
	ctx = context.WithValue(ctx, oauth2.HTTPClient, rp.httpClient)
	return rp.oauth2Cfg.Exchange(ctx, code, oauth2.SetAuthURLParam("code_verifier", codeVerifier))
}

// VerifyIDToken verifies the token's signature (against IS's JWKS),
// issuer, audience and expiry, then additionally checks the nonce matches
// the one issued for this login transaction — go-oidc validates
// everything except the nonce itself (by design; see IDToken.Nonce's
// doc comment), so that check belongs to the caller.
func (rp *RP) VerifyIDToken(ctx context.Context, rawIDToken, expectedNonce string) (*Claims, error) {
	ctx = oidc.ClientContext(ctx, rp.httpClient)
	idToken, err := rp.verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return nil, fmt.Errorf("oidcrp: ID token verification failed: %w", err)
	}
	if idToken.Nonce == "" || idToken.Nonce != expectedNonce {
		return nil, fmt.Errorf("oidcrp: ID token nonce mismatch")
	}
	return claimsFromIDToken(idToken)
}

// VerifyLogoutToken verifies a back-channel logout token's signature,
// issuer, audience, expiry, the required backchannel-logout event, and the
// absence of a nonce (all enforced by go-oidc's VerifyLogout). Replay
// protection (rejecting an already-seen `jti`) is the caller's
// responsibility, per the library's own documentation.
func (rp *RP) VerifyLogoutToken(ctx context.Context, rawLogoutToken string) (*oidc.LogoutToken, error) {
	ctx = oidc.ClientContext(ctx, rp.httpClient)
	return rp.logoutVer.VerifyLogout(ctx, rawLogoutToken)
}

// LogoutURL builds an RP-initiated logout URL against IS's
// end_session_endpoint. postLogoutRedirectURI must be one of the app's own
// registered Authorized redirect URLs — IS validates it against that list,
// there is no separate allow-list field (verified against IS 7.3.0 docs).
func (rp *RP) LogoutURL(idTokenHint, postLogoutRedirectURI, state string) string {
	v := url.Values{}
	if idTokenHint != "" {
		v.Set("id_token_hint", idTokenHint)
	}
	v.Set("post_logout_redirect_uri", postLogoutRedirectURI)
	v.Set("client_id", rp.oauth2Cfg.ClientID)
	if state != "" {
		v.Set("state", state)
	}
	return rp.provider.EndSessionEndpoint() + "?" + v.Encode()
}
