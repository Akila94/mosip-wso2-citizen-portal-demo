package oidcrp

import (
	"context"
	"fmt"
	"net/http"

	"github.com/coreos/go-oidc/v3/oidc"
)

// Provider wraps a discovered OIDC provider plus the extra discovery fields
// go-oidc's Provider does not expose directly (end_session_endpoint).
type Provider struct {
	inner              *oidc.Provider
	endSessionEndpoint string
}

// NewProvider runs OIDC discovery against issuer using httpClient (which
// must already trust IS's certificate — see NewHTTPClient) and returns a
// Provider. issuer must be the exact string IS puts in its ID tokens' `iss`
// claim (verified: `https://localhost:9443/oauth2/token` for a default
// WSO2 IS 7.3.0 super-tenant install — see PORTAL-INTEGRATION-PLAN.md's
// appendix), since go-oidc validates the token's issuer against the
// discovery URL's origin.
func NewProvider(ctx context.Context, issuer string, httpClient *http.Client) (*Provider, error) {
	ctx = oidc.ClientContext(ctx, httpClient)
	inner, err := oidc.NewProvider(ctx, issuer)
	if err != nil {
		return nil, fmt.Errorf("oidcrp: discovery against %q failed: %w", issuer, err)
	}

	var extra struct {
		EndSessionEndpoint string `json:"end_session_endpoint"`
	}
	if err := inner.Claims(&extra); err != nil {
		return nil, fmt.Errorf("oidcrp: parsing discovery document: %w", err)
	}
	if extra.EndSessionEndpoint == "" {
		return nil, fmt.Errorf("oidcrp: discovery document has no end_session_endpoint")
	}

	return &Provider{inner: inner, endSessionEndpoint: extra.EndSessionEndpoint}, nil
}

// EndSessionEndpoint returns the RP-initiated logout endpoint (IS's
// `/oidc/logout`) discovered from the provider's metadata.
func (p *Provider) EndSessionEndpoint() string {
	return p.endSessionEndpoint
}
