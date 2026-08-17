package httpapi

import (
	"net/http"

	"github.com/akila94/mosip-wso2-citizen-portal-demo/citizen-portal-bff/internal/security"
)

const csrfHeaderName = "X-CSRF-Token"

// checkCSRF implements the double-submit pattern: the CSRF cookie's value
// must equal the header value on every state-changing request. A same-site
// script can read its own app's non-HttpOnly CSRF cookie and echo it back;
// a cross-site attacker cannot, since browsers do not let JavaScript on one
// origin read another origin's cookies.
func (s *Server) checkCSRF(r *http.Request, app *AppRoute) bool {
	cookieVal, ok := readCookie(r, app.CSRFCookieName)
	if !ok {
		return false
	}
	return security.ConstantTimeEqual(cookieVal, r.Header.Get(csrfHeaderName))
}

func (s *Server) handleLogout(app *AppRoute) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Type", "application/json")

		if !s.checkCSRF(r, app) {
			w.WriteHeader(http.StatusForbidden)
			s.writeJSON(w, map[string]string{"error": "csrf token missing or invalid"})
			return
		}

		var rawIDToken string
		if key, ok := readCookie(r, app.SessionCookieName); ok {
			if sess, ok := s.Sessions.GetSession(key); ok {
				rawIDToken = sess.RawIDToken
			}
			s.Sessions.DestroySession(key)
		}

		clearCookie(w, app, s.CookieSecure, app.SessionCookieName)
		clearCookie(w, app, s.CookieSecure, app.CSRFCookieName)

		logoutURL := app.Client.LogoutURL(rawIDToken, app.PostLogoutRedirectURI, "")
		s.writeJSON(w, map[string]string{"logoutUrl": logoutURL})
	}
}
