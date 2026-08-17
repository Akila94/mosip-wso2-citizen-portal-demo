package authmw

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"github.com/akila94/mosip-wso2-citizen-portal-demo/gov-services-api/internal/security"
)

// contextKey is an unexported type for context.Context keys, per Go's own
// documented convention (avoids collisions with keys from other packages).
type contextKey string

const (
	subjectContextKey contextKey = "authmw.subject"
	scopesContextKey  contextKey = "authmw.scopes"
)

// SubjectFromContext returns the verified token's `sub` claim, for handlers
// that need to key the citizen registry (internal/registry) by it.
func SubjectFromContext(ctx context.Context) (string, bool) {
	sub, ok := ctx.Value(subjectContextKey).(string)
	return sub, ok
}

// ScopesFromContext returns the verified token's scopes, for handlers that
// need to project a response by which scopes were actually granted (the
// /citizen/profile endpoint).
func ScopesFromContext(ctx context.Context) ([]string, bool) {
	scopes, ok := ctx.Value(scopesContextKey).([]string)
	return scopes, ok
}

// bearerToken extracts the raw token from an "Authorization: Bearer <token>"
// header, returning false if the header is missing or malformed.
func bearerToken(r *http.Request) (string, bool) {
	header := r.Header.Get("Authorization")
	const prefix = "Bearer "
	if header == "" || !strings.HasPrefix(header, prefix) {
		return "", false
	}
	token := strings.TrimSpace(header[len(prefix):])
	if token == "" {
		return "", false
	}
	return token, true
}

// RequireAudienceAndScope returns chi-compatible middleware that: extracts
// the bearer token (400 if missing/malformed, never logging the token
// itself); verifies it via v.Verify (401 on any verification failure,
// logged with the specific reason but never the raw token); rejects if none
// of anyOfAudience appears in the token's aud (403 — this is what makes
// Application A's token genuinely rejected by Application B's router);
// rejects if requiredScope is non-empty and absent from the token's scopes
// (403). On success, stores the subject and scopes into the request context
// and calls next.
func RequireAudienceAndScope(v *Verifier, anyOfAudience []string, requiredScope string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, ok := bearerToken(r)
			if !ok {
				http.Error(w, "missing or malformed Authorization header", http.StatusBadRequest)
				return
			}

			claims, err := v.Verify(r.Context(), token)
			if err != nil {
				slog.Warn("access token rejected", "reason", security.SanitizeForLog(err.Error()), "path", security.SanitizeForLog(r.URL.Path))
				http.Error(w, "invalid access token", http.StatusUnauthorized)
				return
			}

			if !claims.hasAnyAudience(anyOfAudience) {
				slog.Warn("access token rejected: audience mismatch", "path", security.SanitizeForLog(r.URL.Path), "clientID", security.SanitizeForLog(claims.ClientID))
				http.Error(w, "token audience not accepted by this router", http.StatusForbidden)
				return
			}

			if requiredScope != "" && !claims.hasScope(requiredScope) {
				slog.Warn("access token rejected: missing required scope", "path", security.SanitizeForLog(r.URL.Path), "requiredScope", security.SanitizeForLog(requiredScope))
				http.Error(w, "token missing required scope", http.StatusForbidden)
				return
			}

			ctx := context.WithValue(r.Context(), subjectContextKey, claims.Subject)
			ctx = context.WithValue(ctx, scopesContextKey, claims.scopes())
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
