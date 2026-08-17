// Package authmw is gov-services-api's authentication/authorization
// middleware: JWT access-token signature verification against WSO2 IS's
// live JWKS, plus the per-router required-audience and required-scope
// checks that make this a genuine resource server rather than a second
// relying party. It never talks to a browser and presents no credentials of
// its own — it only validates tokens the citizen-portal-bff injects on the
// citizen's behalf.
package authmw

import (
	"context"
	"fmt"
	"net/http"

	"github.com/coreos/go-oidc/v3/oidc"
)

// NewKeySet runs OIDC discovery against issuer using httpClient (which must
// already trust IS's certificate — see internal/httpclient.NewHTTPClient)
// and returns an oidc.KeySet backed by IS's live JWKS. The returned KeySet
// caches keys and handles rotation itself (oidc.RemoteKeySet's own
// behavior), so no separate caching layer is needed here.
func NewKeySet(ctx context.Context, issuer string, httpClient *http.Client) (oidc.KeySet, error) {
	discoveryCtx := oidc.ClientContext(ctx, httpClient)
	provider, err := oidc.NewProvider(discoveryCtx, issuer)
	if err != nil {
		return nil, fmt.Errorf("authmw: discovery against %q failed: %w", issuer, err)
	}

	var extra struct {
		JWKSURI string `json:"jwks_uri"`
	}
	if err := provider.Claims(&extra); err != nil {
		return nil, fmt.Errorf("authmw: parsing discovery document: %w", err)
	}
	if extra.JWKSURI == "" {
		return nil, fmt.Errorf("authmw: discovery document has no jwks_uri")
	}

	return oidc.NewRemoteKeySet(oidc.ClientContext(ctx, httpClient), extra.JWKSURI), nil
}
