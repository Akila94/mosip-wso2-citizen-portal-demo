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
		r.Get(prefix+"/step-up", s.handleStepUp(app))
		r.Get(prefix+"/callback", s.handleCallback(app))
		r.Get(prefix+"/session", s.handleSession(app))
		r.Post(prefix+"/logout", s.handleLogout(app))
		r.Post(prefix+"/backchannel-logout", s.handleBackchannelLogout(app))
	}

	// Each app's /api/* data routes call gov-services-api on the citizen's
	// behalf via s.Upstream. These are genuinely different route sets per
	// app (not a generic loop), so they are registered explicitly by app
	// key — matching this project's "no generic proxy, named handlers"
	// philosophy applied one level up, at the routing layer too.
	// /session-inspector is mounted on all three apps: it is the only data
	// route they share, and holding two of them side by side is what makes
	// one IS session visible as SSO.
	if app, ok := s.Apps["portal"]; ok {
		// The one route registered outside requireSession: the service
		// catalogue is public information and the landing page shows it to
		// signed-out visitors. It lives under its own /public/ segment
		// rather than beside the session-gated /api/ routes, so "which
		// routes need a session" stays readable at a glance here. Portal
		// only — the two micro apps have no signed-out surface.
		r.Get(app.RoutePrefix+"/public/catalogue", s.handlePublicPortalCatalogue)

		r.Route(app.RoutePrefix+"/api", func(pr chi.Router) {
			pr.Use(s.requireSession(app))
			s.mountPortalDataRoutes(pr)
			s.mountSessionInspectorRoute(pr, app)
		})
	}
	if app, ok := s.Apps["driving-licence"]; ok {
		r.Route(app.RoutePrefix+"/api", func(dr chi.Router) {
			dr.Use(s.requireSession(app))
			s.mountDrivingLicenceDataRoutes(dr, app)
			s.mountSessionInspectorRoute(dr, app)
		})
	}
	if app, ok := s.Apps["revenue-licence"]; ok {
		r.Route(app.RoutePrefix+"/api", func(vr chi.Router) {
			vr.Use(s.requireSession(app))
			s.mountRevenueLicenceDataRoutes(vr, app)
			s.mountSessionInspectorRoute(vr, app)
		})
	}

	// Everything not matched above is the SPA — the BFF is the browser's
	// only origin (PORTAL-INTEGRATION-PLAN.md's port map). Registering this
	// last, as chi's NotFound handler, guarantees every /bff/... route above
	// wins first; internal/devproxy then refuses to answer an *unmatched*
	// /bff/... path with HTML, so a typo'd API path stays a JSON 404.
	//
	// The SPA policy is applied here rather than inside devproxy because the
	// router owns this response's header chain: security.SecurityHeaders has
	// already set the strict API CSP by the time a request reaches the
	// NotFound handler, and SPAHeaders replaces it for these responses only.
	if s.SPA != nil {
		r.NotFound(security.SPAHeaders(s.SPADevMode)(s.SPA).ServeHTTP)
	} else {
		r.NotFound(s.handleNotFound)
	}

	return r
}

// handleNotFound answers an unmatched path when no SPA handler is
// configured (unit tests, and any API-only deployment) with the same JSON
// error shape the rest of the API uses, rather than chi's plain-text
// default.
func (s *Server) handleNotFound(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusNotFound)
	s.writeJSON(w, map[string]string{"error": "not found"})
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
