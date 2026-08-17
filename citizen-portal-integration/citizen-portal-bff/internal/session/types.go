package session

import "time"

// LoginTxn holds everything a login redirect needs to remember between the
// 302 to IS's authorize endpoint and the callback: the OAuth "state" (bound
// 1:1 to this transaction), the OIDC nonce, and the PKCE verifier. It never
// leaves the server — only the transaction's store key travels to the
// browser, as a short-TTL HttpOnly cookie.
type LoginTxn struct {
	AppKey       string // "portal" | "driving-licence" | "revenue-licence"
	State        string
	Nonce        string
	CodeVerifier string
	ReturnTo     string
	CreatedAt    time.Time
}

// User is the subset of ID-token/UserInfo claims the BFF keeps. Field names
// follow the OIDC standard claim names (RFC 7519 / OIDC Core 5.1).
type User struct {
	Sub         string
	Name        string
	GivenName   string
	FamilyName  string
	Email       string
	PhoneNumber string
	Birthdate   string
	Picture     string
}

// AuthSession is an authenticated session for one app. RawIDToken is kept
// solely to build the RP-initiated logout URL's id_token_hint — it is never
// serialized to the browser; the HTTP layer projects a Session into a
// token-free JSON view before responding.
type AuthSession struct {
	AppKey     string
	User       User
	Sid        string // OIDC "sid" claim — links this session to its IdP session
	Acr        string
	Amr        []string
	AuthTime   time.Time
	ExpiresAt  time.Time
	RawIDToken string
	// AccessToken is the OAuth2 access token issued alongside the ID token.
	// Like RawIDToken, it is kept solely for server-to-server use — calling
	// gov-services-api on the citizen's behalf — and is never serialized to
	// the browser; the HTTP layer's sessionView has no field for it.
	AccessToken string
	// AccessTokenExpiresAt is the access token's own expiry (which can differ
	// from the ID token's), so a caller can tell a stale token from a live
	// one before spending a round trip to gov-services-api discovering it was
	// rejected. There is deliberately no refresh-token handling yet — if the
	// access token expires before the BFF's own session idle timeout, an
	// upstream call simply fails with whatever gov-services-api returns for
	// an expired token, and that failure surfaces to the SPA as a normal
	// upstream error. This is a known, documented gap, not an oversight.
	AccessTokenExpiresAt time.Time
}
