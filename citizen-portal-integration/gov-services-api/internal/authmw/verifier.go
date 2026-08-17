package authmw

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
)

// Verifier validates a raw JWT access token's signature (against keySet),
// issuer, and expiry, and returns its parsed claims. It does not check
// audience or scope — those are per-router concerns, applied by
// RequireAudienceAndScope.
type Verifier struct {
	keySet oidc.KeySet
	issuer string
}

// NewVerifier constructs a Verifier that checks signatures against keySet
// and rejects any token whose `iss` claim does not equal issuer exactly.
func NewVerifier(keySet oidc.KeySet, issuer string) *Verifier {
	return &Verifier{keySet: keySet, issuer: issuer}
}

// Verify checks rawToken's signature against v.keySet (the same primitive
// go-oidc's own IDTokenVerifier uses internally, exposed as public API on
// oidc.KeySet — no extra JWT library is needed for signature verification),
// then validates its issuer and time-based claims, returning the parsed
// claims on success. Every rejection reason is a distinct, wrapped error so
// callers can distinguish why a token was rejected.
func (v *Verifier) Verify(ctx context.Context, rawToken string) (accessTokenClaims, error) {
	payload, err := v.keySet.VerifySignature(ctx, rawToken)
	if err != nil {
		return accessTokenClaims{}, fmt.Errorf("authmw: signature verification failed: %w", err)
	}

	var claims accessTokenClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return accessTokenClaims{}, fmt.Errorf("authmw: parsing access token claims: %w", err)
	}

	if claims.Issuer != v.issuer {
		return accessTokenClaims{}, fmt.Errorf("authmw: unexpected issuer")
	}

	now := time.Now().Unix()
	if now >= claims.Expiry {
		return accessTokenClaims{}, fmt.Errorf("authmw: token expired")
	}
	if claims.NotBefore != 0 && now < claims.NotBefore {
		return accessTokenClaims{}, fmt.Errorf("authmw: token not yet valid")
	}

	return claims, nil
}
