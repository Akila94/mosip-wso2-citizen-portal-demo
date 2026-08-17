package httpapi

import (
	"context"
	"net/http"

	"github.com/akila94/mosip-wso2-citizen-portal-demo/citizen-portal-bff/internal/session"
)

// sessionContextKey is an unexported context key, per Go's own documented
// convention (avoids collisions with keys set by other packages).
type sessionContextKey struct{}

// requireSession returns middleware that reads app's session cookie, loads
// the session, and — if either is missing — writes the same
// {"authenticated":false} 401 body handleSession already returns,
// short-circuiting the chain. On success it stores the session in the
// request context for downstream handlers to read via sessionFromContext.
func (s *Server) requireSession(app *AppRoute) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Cache-Control", "no-store")

			key, ok := readCookie(r, app.SessionCookieName)
			if !ok {
				s.respondUnauthenticated(w)
				return
			}
			sess, ok := s.Sessions.GetSession(key)
			if !ok {
				s.respondUnauthenticated(w)
				return
			}

			ctx := context.WithValue(r.Context(), sessionContextKey{}, sess)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// respondUnauthenticated writes the exact same 401 body handleSession
// returns for a missing or unknown session (session_handler.go), reused
// here so every session-gated endpoint looks identical to the SPA
// regardless of which one rejected the request.
func (s *Server) respondUnauthenticated(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	s.writeJSON(w, sessionView{Authenticated: false})
}

// sessionFromContext retrieves the session requireSession stored, for use
// by data-route handlers registered behind it.
func sessionFromContext(ctx context.Context) (session.AuthSession, bool) {
	sess, ok := ctx.Value(sessionContextKey{}).(session.AuthSession)
	return sess, ok
}
