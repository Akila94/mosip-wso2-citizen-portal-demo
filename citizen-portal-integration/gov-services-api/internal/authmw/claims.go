package authmw

import (
	"encoding/json"
	"fmt"
	"strings"
)

// audience unmarshals either a single JSON string or an array of strings
// into a []string, since RFC 7519 §4.1.3 permits both and different IdPs
// (and different token shapes from the same IdP) use either form. WSO2 IS's
// JWT access tokens carry `aud` as [client_id] + any configured audiences,
// which can render as either shape depending on how many audiences are
// configured.
type audience []string

// UnmarshalJSON implements json.Unmarshaler, accepting a bare JSON string or
// a JSON array of strings and normalizing both into the same []string.
func (a *audience) UnmarshalJSON(data []byte) error {
	var single string
	if err := json.Unmarshal(data, &single); err == nil {
		*a = audience{single}
		return nil
	}

	var multiple []string
	if err := json.Unmarshal(data, &multiple); err == nil {
		*a = audience(multiple)
		return nil
	}

	return fmt.Errorf("authmw: aud claim is neither a string nor an array of strings: %s", string(data))
}

// accessTokenClaims is the subset of a WSO2 IS JWT access token's claims
// this service relies on (verified against IS 7.3.0 source, per
// PORTAL-INTEGRATION-PLAN.md's appendix "JWT access token claims" table).
type accessTokenClaims struct {
	Issuer    string   `json:"iss"`
	Audience  audience `json:"aud"`
	Expiry    int64    `json:"exp"`
	NotBefore int64    `json:"nbf"`
	Subject   string   `json:"sub"`
	ClientID  string   `json:"client_id"`
	Scope     string   `json:"scope"`
}

// scopes splits the claims' space-separated `scope` string into individual
// scope names, per RFC 6749 §3.3.
func (c accessTokenClaims) scopes() []string {
	return strings.Fields(c.Scope)
}

// hasAnyAudience reports whether any of candidates appears in the claims'
// aud list.
func (c accessTokenClaims) hasAnyAudience(candidates []string) bool {
	for _, want := range candidates {
		for _, got := range c.Audience {
			if got == want {
				return true
			}
		}
	}
	return false
}

// hasScope reports whether scope appears among the claims' granted scopes.
func (c accessTokenClaims) hasScope(scope string) bool {
	for _, got := range c.scopes() {
		if got == scope {
			return true
		}
	}
	return false
}
