package httpapi

import (
	"context"
	"errors"
	"fmt"

	"github.com/akila94/mosip-wso2-citizen-portal-demo/citizen-portal-bff/internal/oidcrp"
	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

// fakeClient is a test double for OIDCClient — no network, no signature
// verification, so handler tests exercise the HTTP-layer logic (cookies,
// CSRF, returnTo validation, replay rejection) in isolation from oidcrp,
// which has its own test suite against a real signed-token round trip.
type fakeClient struct {
	authCodeURL string

	exchangeToken *oauth2.Token
	exchangeErr   error

	verifyClaims *oidcrp.Claims
	verifyErr    error

	logoutToken *oidc.LogoutToken
	logoutErr   error

	logoutURL string

	lastExchangeCode     string
	lastExchangeVerifier string
	lastVerifyRawIDToken string
	lastVerifyNonce      string
}

func (f *fakeClient) AuthCodeURL(req oidcrp.AuthRequest) string {
	if f.authCodeURL != "" {
		return f.authCodeURL
	}
	return fmt.Sprintf("https://idp.example/authorize?state=%s&nonce=%s", req.State, req.Nonce)
}

func (f *fakeClient) Exchange(ctx context.Context, code, codeVerifier string) (*oauth2.Token, error) {
	f.lastExchangeCode = code
	f.lastExchangeVerifier = codeVerifier
	if f.exchangeErr != nil {
		return nil, f.exchangeErr
	}
	if f.exchangeToken != nil {
		return f.exchangeToken, nil
	}
	return (&oauth2.Token{AccessToken: "at-1"}).WithExtra(map[string]any{"id_token": "raw-id-token"}), nil
}

func (f *fakeClient) VerifyIDToken(ctx context.Context, rawIDToken, expectedNonce string) (*oidcrp.Claims, error) {
	f.lastVerifyRawIDToken = rawIDToken
	f.lastVerifyNonce = expectedNonce
	if f.verifyErr != nil {
		return nil, f.verifyErr
	}
	if f.verifyClaims != nil {
		return f.verifyClaims, nil
	}
	return &oidcrp.Claims{Sub: "sub-1", Name: "Test User", Sid: "sid-1", Amr: []string{"BasicAuthenticator"}}, nil
}

func (f *fakeClient) VerifyLogoutToken(ctx context.Context, rawLogoutToken string) (*oidc.LogoutToken, error) {
	if f.logoutErr != nil {
		return nil, f.logoutErr
	}
	return f.logoutToken, nil
}

func (f *fakeClient) LogoutURL(idTokenHint, postLogoutRedirectURI, state string) string {
	if f.logoutURL != "" {
		return f.logoutURL
	}
	return "https://idp.example/oidc/logout?id_token_hint=" + idTokenHint
}

var errFakeExchange = errors.New("fake exchange failure")
