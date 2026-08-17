package httpapi

import (
	"net/http"
	"time"

	"github.com/akila94/mosip-wso2-citizen-portal-demo/citizen-portal-bff/internal/security"
)

// replaySeenTTL bounds how long a logout token's jti is remembered for
// replay rejection. A logout token's own exp is normally only a few
// minutes out, but a minimum floor keeps a very-short-lived token from
// being immediately forgettable and replayable.
const replaySeenTTL = 10 * time.Minute

// handleBackchannelLogout implements the BFF side of OIDC Back-Channel
// Logout 1.0: IS calls this endpoint directly (no browser involved) with a
// `logout_token` form parameter. On a valid token this destroys every
// session sharing the token's `sid` — across every app registered on this
// Server, not just the one that received the call — which is what turns
// one IS sign-out into single logout for all three apps.
//
// This relies on IS always including `sid` in the logout token, verified
// against the exact IS 7.3.0 source (see PORTAL-INTEGRATION-PLAN.md's
// appendix: "sid in ID token ... present by default for /authorize and the
// authorization_code/refresh_token grants"). A token with no `sid` is
// rejected rather than falling back to a `sub`-only match, since a `sub`
// match alone cannot distinguish which of a citizen's several IS sessions
// is being ended.
func (s *Server) handleBackchannelLogout(app *AppRoute) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")

		if err := r.ParseForm(); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		rawToken := r.PostFormValue("logout_token")
		if rawToken == "" {
			http.Error(w, "missing logout_token", http.StatusBadRequest)
			return
		}

		logoutToken, err := app.Client.VerifyLogoutToken(r.Context(), rawToken)
		if err != nil {
			s.Logger.Warn("rejected back-channel logout token", "app", app.Key, "error", security.SanitizeForLog(err.Error()))
			http.Error(w, "invalid logout token", http.StatusBadRequest)
			return
		}

		if logoutToken.SessionID == "" {
			s.Logger.Warn("back-channel logout token has no sid", "app", app.Key)
			http.Error(w, "logout token missing sid", http.StatusBadRequest)
			return
		}

		if _, seen := s.replaySeen.Get(logoutToken.TokenID); seen {
			s.Logger.Warn("rejected replayed logout token", "app", app.Key)
			http.Error(w, "logout token already used", http.StatusBadRequest)
			return
		}
		s.replaySeen.Put(logoutToken.TokenID, struct{}{}, replaySeenTTL)

		n := s.Sessions.DestroyBySid(logoutToken.SessionID)
		s.Logger.Info("back-channel logout destroyed sessions", "app", app.Key, "count", n)

		w.WriteHeader(http.StatusOK)
	}
}
