// Package httpapi wires the BFF's HTTP routes: the OIDC login/callback/
// session/logout/back-channel-logout flow for one app (M1: the portal app
// only), the security middleware chain, and the session cookie management.
package httpapi

import (
	"context"

	"github.com/akila94/mosip-wso2-citizen-portal-demo/citizen-portal-bff/internal/oidcrp"
	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

// OIDCClient is the subset of *oidcrp.RP the HTTP handlers depend on.
// *oidcrp.RP satisfies this interface structurally; declaring it here lets
// handler tests substitute a fake without spinning up a stub IdP for every
// test, while production code always passes the real thing.
type OIDCClient interface {
	AuthCodeURL(req oidcrp.AuthRequest) string
	Exchange(ctx context.Context, code, codeVerifier string) (*oauth2.Token, error)
	VerifyIDToken(ctx context.Context, rawIDToken, expectedNonce string) (*oidcrp.Claims, error)
	VerifyLogoutToken(ctx context.Context, rawLogoutToken string) (*oidc.LogoutToken, error)
	LogoutURL(idTokenHint, postLogoutRedirectURI, state string) string
}

var _ OIDCClient = (*oidcrp.RP)(nil)
