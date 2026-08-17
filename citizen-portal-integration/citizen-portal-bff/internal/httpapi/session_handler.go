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
type sessionView struct {
	Authenticated  bool     `json:"authenticated"`
	ClientID       string   `json:"-"` // reserved: populated once multi-app client metadata exists (M2)
	AppKey         string   `json:"appKey"`
	User           userView `json:"user"`
	AssuranceLevel string   `json:"assuranceLevel"`
	Sid            string   `json:"sid"`
	Acr            string   `json:"acr,omitempty"`
	Amr            []string `json:"amr,omitempty"`
	AuthTime       int64    `json:"authTime,omitempty"`
	ExpiresAt      int64    `json:"expiresAt,omitempty"`
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

func projectSession(app *AppRoute, sess session.AuthSession) sessionView {
	return sessionView{
		Authenticated: true,
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
		Sid:            sess.Sid,
		Acr:            sess.Acr,
		Amr:            sess.Amr,
		AuthTime:       unixOrZero(sess.AuthTime),
		ExpiresAt:      unixOrZero(sess.ExpiresAt),
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

		s.writeJSON(w, projectSession(app, sess))
	}
}
