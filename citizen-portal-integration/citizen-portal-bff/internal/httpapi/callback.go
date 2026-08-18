package httpapi

import (
	"net/http"

	"github.com/akila94/mosip-wso2-citizen-portal-demo/citizen-portal-bff/internal/security"
	"github.com/akila94/mosip-wso2-citizen-portal-demo/citizen-portal-bff/internal/session"
)

func (s *Server) handleCallback(app *AppRoute) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")

		if errParam := r.URL.Query().Get("error"); errParam != "" {
			s.Logger.Warn("IS returned an authorization error",
				"app", app.Key,
				"error", security.SanitizeForLog(errParam),
				"error_description", security.SanitizeForLog(r.URL.Query().Get("error_description")))
			http.Error(w, "authentication failed", http.StatusBadRequest)
			return
		}

		txnKey, ok := readCookie(r, app.LoginTxnCookieName)
		if !ok {
			http.Error(w, "missing or expired login transaction", http.StatusBadRequest)
			return
		}
		clearCookie(w, app, s.CookieSecure, app.LoginTxnCookieName)

		txn, ok := s.Sessions.ConsumeLoginTxn(txnKey)
		if !ok {
			http.Error(w, "missing or expired login transaction", http.StatusBadRequest)
			return
		}

		state := r.URL.Query().Get("state")
		if !security.ConstantTimeEqual(state, txn.State) {
			s.Logger.Warn("state mismatch on callback", "app", app.Key)
			http.Error(w, "state mismatch", http.StatusBadRequest)
			return
		}

		code := r.URL.Query().Get("code")
		if code == "" {
			http.Error(w, "missing authorization code", http.StatusBadRequest)
			return
		}

		token, err := app.Client.Exchange(r.Context(), code, txn.CodeVerifier)
		if err != nil {
			s.Logger.Error("token exchange failed", "app", app.Key, "error", security.SanitizeForLog(err.Error()))
			http.Error(w, "authentication failed", http.StatusBadGateway)
			return
		}

		rawIDToken, ok := token.Extra("id_token").(string)
		if !ok || rawIDToken == "" {
			s.Logger.Error("token response had no id_token", "app", app.Key)
			http.Error(w, "authentication failed", http.StatusBadGateway)
			return
		}

		claims, err := app.Client.VerifyIDToken(r.Context(), rawIDToken, txn.Nonce)
		if err != nil {
			s.Logger.Error("id token verification failed", "app", app.Key, "error", security.SanitizeForLog(err.Error()))
			http.Error(w, "authentication failed", http.StatusUnauthorized)
			return
		}

		sess := session.AuthSession{
			AppKey: app.Key,
			User: session.User{
				Sub:         claims.Sub,
				Name:        claims.Name,
				GivenName:   claims.GivenName,
				FamilyName:  claims.FamilyName,
				Email:       claims.Email,
				PhoneNumber: claims.PhoneNumber,
				Birthdate:   claims.Birthdate,
				Picture:     claims.Picture,
			},
			Sid:                  claims.Sid,
			Acr:                  claims.Acr,
			Amr:                  claims.Amr,
			AuthTime:             claims.AuthTime,
			ExpiresAt:            claims.Expiry,
			RawIDToken:           rawIDToken,
			AccessToken:          token.AccessToken,
			AccessTokenExpiresAt: token.Expiry,
			// The full claim set is kept for the session inspector, which
			// shows what each of the three clients was actually released.
			// releasedClaims bounds it and drops the nonce first.
			IDTokenClaims: releasedClaims(claims.Raw),
		}

		sessionKey, err := s.Sessions.CreateSession(sess)
		if err != nil {
			s.internalError(w, err)
			return
		}
		setCookie(w, app, s.CookieSecure, app.SessionCookieName, sessionKey, s.SessionIdleTimeout)

		// The CSRF cookie is HttpOnly like every other cookie here: the SPA
		// reads the token from GET /bff/{app}/session's JSON body instead of
		// document.cookie (see handleSession), because a cookie scoped to
		// Path=/bff/{app} is not visible to a document served from / or
		// /apps/... under RFC 6265 §5.4 path-matching.
		csrfToken, err := security.RandomToken(32)
		if err != nil {
			s.internalError(w, err)
			return
		}
		setCookie(w, app, s.CookieSecure, app.CSRFCookieName, csrfToken, s.SessionIdleTimeout)

		http.Redirect(w, r, txn.ReturnTo, http.StatusFound)
	}
}
