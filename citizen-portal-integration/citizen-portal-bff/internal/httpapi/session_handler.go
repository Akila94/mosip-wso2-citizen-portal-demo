package httpapi

import (
	"net/http"
	"time"

	"github.com/akila94/mosip-wso2-citizen-portal-demo/citizen-portal-bff/internal/session"
)

// sessionView is the token-free projection returned to the SPA. There is no
// field here that could hold an ID token, access token or refresh token —
// that is a structural guarantee, not a redaction step: nothing in
// session.AuthSession beyond RawIDToken is sensitive, and RawIDToken is
// deliberately never copied into this struct.
//
// There is deliberately no releasedClaims field either, even though
// PORTAL-INTEGRATION-PLAN.md's session-projection sketch shows one. This
// endpoint is polled on every page load by every app, while only the session
// inspector renders the claim set — so the full ID-token claim map is served
// exclusively by GET /bff/{app}/api/session-inspector. That is data
// minimisation, not an oversight: please do not "fix" the inconsistency by
// widening this struct.
type sessionView struct {
	Authenticated  bool     `json:"authenticated"`
	ClientID       string   `json:"clientId,omitempty"`
	AppName        string   `json:"appName,omitempty"`
	AppKey         string   `json:"appKey"`
	User           userView `json:"user"`
	AssuranceLevel string   `json:"assuranceLevel"`
	// IDP is derived from `amr`, not read from a claim — IS emits no `idp`
	// claim. See session.DeriveIdentityProvider.
	IDP       string   `json:"idp,omitempty"`
	Sid       string   `json:"sid"`
	Acr       string   `json:"acr,omitempty"`
	Amr       []string `json:"amr,omitempty"`
	AuthTime  int64    `json:"authTime,omitempty"`
	ExpiresAt int64    `json:"expiresAt,omitempty"`
	// CSRFToken is this session's double-submit token, returned only to a
	// caller that already presented a valid session cookie.
	//
	// It is not a downgrade from a JS-readable cookie — it is what makes the
	// double-submit work at all. The token's secrecy rests on the same-origin
	// policy either way: a cross-origin attacker can make the browser *send*
	// the CSRF cookie but cannot read it, and equally cannot read this
	// response body, because CORS denies a cross-origin reader access to it.
	// Learning the token still requires same-origin script execution. Making
	// the cookie HttpOnly (see cookies.go) strictly improves on the previous
	// arrangement, and is now possible because the cookie's Path=/bff/{app}
	// scope meant the SPA could never read it from document.cookie anyway.
	CSRFToken string `json:"csrfToken,omitempty"`
}

type userView struct {
	Sub         string `json:"sub"`
	Name        string `json:"name,omitempty"`
	GivenName   string `json:"givenName,omitempty"`
	FamilyName  string `json:"familyName,omitempty"`
	Email       string `json:"email,omitempty"`
	PhoneNumber string `json:"phoneNumber,omitempty"`
	Birthdate   string `json:"birthdate,omitempty"`
	Picture     string `json:"picture,omitempty"`
}

// projectSession builds the SPA-facing view of sess. csrfToken is the value
// of the caller's own CSRF cookie, echoed back so the SPA can present it as
// the X-CSRF-Token header on state-changing requests; it is empty when the
// caller presented no CSRF cookie, and the view then simply omits it.
func projectSession(app *AppRoute, sess session.AuthSession, csrfToken string) sessionView {
	return sessionView{
		Authenticated: true,
		ClientID:      app.ClientID,
		AppName:       app.AppName,
		AppKey:        app.Key,
		User: userView{
			Sub:         sess.User.Sub,
			Name:        sess.User.Name,
			GivenName:   sess.User.GivenName,
			FamilyName:  sess.User.FamilyName,
			Email:       sess.User.Email,
			PhoneNumber: sess.User.PhoneNumber,
			Birthdate:   sess.User.Birthdate,
			Picture:     sess.User.Picture,
		},
		AssuranceLevel: string(session.DeriveAssuranceLevel(sess.Amr)),
		IDP:            session.DeriveIdentityProvider(sess.Amr),
		Sid:            sess.Sid,
		Acr:            sess.Acr,
		Amr:            sess.Amr,
		AuthTime:       unixOrZero(sess.AuthTime),
		ExpiresAt:      unixOrZero(sess.ExpiresAt),
		CSRFToken:      csrfToken,
	}
}

func unixOrZero(t time.Time) int64 {
	if t.IsZero() {
		return 0
	}
	return t.Unix()
}

func (s *Server) handleSession(app *AppRoute) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Type", "application/json")

		key, ok := readCookie(r, app.SessionCookieName)
		if !ok {
			w.WriteHeader(http.StatusUnauthorized)
			s.writeJSON(w, sessionView{Authenticated: false})
			return
		}
		sess, ok := s.Sessions.GetSession(key)
		if !ok {
			w.WriteHeader(http.StatusUnauthorized)
			s.writeJSON(w, sessionView{Authenticated: false})
			return
		}

		// The CSRF token is read from the caller's own cookie and echoed
		// back, so it is only ever disclosed to a request that already
		// carried both cookies. It is never returned on the 401 paths above.
		csrfToken, _ := readCookie(r, app.CSRFCookieName)
		s.writeJSON(w, projectSession(app, sess, csrfToken))
	}
}
