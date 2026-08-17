package httpapi

import (
	"net/http"
	"time"

	"github.com/akila94/mosip-wso2-citizen-portal-demo/citizen-portal-bff/internal/oidcrp"
	"github.com/akila94/mosip-wso2-citizen-portal-demo/citizen-portal-bff/internal/security"
	"github.com/akila94/mosip-wso2-citizen-portal-demo/citizen-portal-bff/internal/session"
)

// loginTxnTTL bounds how long a login transaction cookie lives — the
// window between redirecting to IS and the browser coming back to
// /callback. Kept short: this is not a session, just a PKCE/state/nonce
// handshake in flight.
const loginTxnTTL = 5 * time.Minute

func (s *Server) handleLogin(app *AppRoute) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		returnTo := r.URL.Query().Get("returnTo")
		if returnTo == "" {
			returnTo = app.ReturnToPrefix
		}
		validated, err := security.ValidateReturnTo(returnTo, app.ReturnToPrefix)
		if err != nil {
			s.Logger.Warn("rejected returnTo on login", "app", app.Key, "reason", err.Error())
			http.Error(w, "invalid returnTo", http.StatusBadRequest)
			return
		}

		state, err := security.RandomToken(32)
		if err != nil {
			s.internalError(w, err)
			return
		}
		nonce, err := security.RandomToken(32)
		if err != nil {
			s.internalError(w, err)
			return
		}
		verifier, err := security.GenerateVerifier()
		if err != nil {
			s.internalError(w, err)
			return
		}

		txnKey, err := s.Sessions.CreateLoginTxn(session.LoginTxn{
			AppKey:       app.Key,
			State:        state,
			Nonce:        nonce,
			CodeVerifier: verifier,
			ReturnTo:     validated,
		})
		if err != nil {
			s.internalError(w, err)
			return
		}

		setCookie(w, app, s.CookieSecure, app.LoginTxnCookieName, txnKey, loginTxnTTL, true)
		w.Header().Set("Cache-Control", "no-store")

		authURL := app.Client.AuthCodeURL(oidcrp.AuthRequest{State: state, Nonce: nonce, CodeVerifier: verifier})
		http.Redirect(w, r, authURL, http.StatusFound)
	}
}
