package httpapi

import (
	"net/http"

	"github.com/akila94/mosip-wso2-citizen-portal-demo/citizen-portal-bff/internal/security"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// Router builds the BFF's HTTP handler: security headers on every response,
// then one route group per registered app under /bff/{key}/....
func (s *Server) Router() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(security.SecurityHeaders)
	r.Use(requestBodyLimit)

	for _, app := range s.Apps {
		app := app
		prefix := app.RoutePrefix
		r.Get(prefix+"/login", s.handleLogin(app))
		r.Get(prefix+"/callback", s.handleCallback(app))
		r.Get(prefix+"/session", s.handleSession(app))
		r.Post(prefix+"/logout", s.handleLogout(app))
		r.Post(prefix+"/backchannel-logout", s.handleBackchannelLogout(app))
	}

	return r
}

// maxRequestBodyBytes caps every request body — a back-channel logout POST
// is the only body-bearing route this app registers, and it is small.
// Mirrors esignet-bridge/server.js's 64 kB JSON limit.
const maxRequestBodyBytes = 64 * 1024

func requestBodyLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
		next.ServeHTTP(w, r)
	})
}
