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

// promptLogin is the OIDC `prompt` value that forces WSO2 IS to
// re-authenticate the citizen even though it already has a session for them.
// IS 7.3.0 honours it (PORTAL-INTEGRATION-PLAN.md's appendix), which is what
// makes the SPA's "raise assurance" action a real re-authentication rather
// than a cosmetic one.
const promptLogin = "login"

// handleLogin starts an ordinary login. It sends no `prompt`, so an existing
// IS session is reused silently — that silent reuse is the SSO the demo
// exists to show.
func (s *Server) handleLogin(app *AppRoute) http.HandlerFunc {
	return s.startAuthorization(app, "")
}

// handleStepUp starts a re-authorization that forces a fresh authentication
// at IS. It is identical to handleLogin — same returnTo validation, same
// single-use state/nonce/PKCE transaction, same short-TTL HttpOnly cookie —
// except for the `prompt=login` parameter on the authorization request.
//
// It deliberately does not require an existing session: a citizen who
// arrives at a step-up link with no session should simply authenticate, and
// the resulting assurance level is derived from the ID token afterwards
// either way (session.DeriveAssuranceLevel), never from the request.
func (s *Server) handleStepUp(app *AppRoute) http.HandlerFunc {
	return s.startAuthorization(app, promptLogin)
}

// startAuthorization builds one authorization request for app. prompt is
// passed through to the authorization URL when non-empty; everything else is
// identical for both entry points, so the security-relevant steps
// (returnTo validation, CSPRNG state/nonce/verifier, transaction storage)
// exist exactly once.
func (s *Server) startAuthorization(app *AppRoute, prompt string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		returnTo := r.URL.Query().Get("returnTo")
		if returnTo == "" {
			returnTo = app.ReturnToPrefix
		}
		validated, err := security.ValidateReturnTo(returnTo, app.ReturnToPrefix)
		if err != nil {
			// The rejected value itself is never logged — only the reason —
			// so an open-redirect probe cannot write attacker text into the
			// log (guideline §1.7).
			s.Logger.Warn("rejected returnTo on authorization request", "app", app.Key, "prompt", prompt, "reason", err.Error())
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

		setCookie(w, app, s.CookieSecure, app.LoginTxnCookieName, txnKey, loginTxnTTL)
		w.Header().Set("Cache-Control", "no-store")

		authURL := app.Client.AuthCodeURL(oidcrp.AuthRequest{
			State:        state,
			Nonce:        nonce,
			CodeVerifier: verifier,
			Prompt:       prompt,
		})
		http.Redirect(w, r, authURL, http.StatusFound)
	}
}
