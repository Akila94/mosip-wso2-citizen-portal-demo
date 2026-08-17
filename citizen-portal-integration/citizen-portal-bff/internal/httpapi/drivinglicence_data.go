package httpapi

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

// mountDrivingLicenceDataRoutes registers the driving-licence app's
// /bff/driving-licence/api/* routes on r. Every route requires a session
// (the caller wraps r with s.requireSession(app) before calling this).
// app is threaded through only to the CSRF-protected write route, which
// needs it to check the double-submit token.
func (s *Server) mountDrivingLicenceDataRoutes(r chi.Router, app *AppRoute) {
	r.Get("/config", s.handleDrivingLicenceConfig)
	r.Get("/test-slots", s.handleDrivingLicenceTestSlots)
	r.Get("/identity", s.handleDrivingLicenceIdentity)
	r.Post("/applications", s.handleDrivingLicenceSubmitApplication(app))
}

// handleDrivingLicenceConfig proxies GET /driving-licence/config.
func (s *Server) handleDrivingLicenceConfig(w http.ResponseWriter, r *http.Request) {
	sess, ok := sessionFromContext(r.Context())
	if !ok {
		s.internalError(w, errMissingSessionInContext)
		return
	}
	resp, err := s.Upstream.DrivingLicenceConfig(r.Context(), sess.AccessToken)
	s.forwardUpstreamResponse(w, resp, err)
}

// handleDrivingLicenceTestSlots proxies GET /driving-licence/test-slots.
// The week query parameter is validated here — a non-integer week is
// rejected with 400 without ever calling gov-services-api — mirroring the
// same validation gov-services-api's own handler does, as defense in
// depth rather than blind passthrough of an unvalidated query string.
// A missing week defaults to 0, matching gov-services-api's own default.
func (s *Server) handleDrivingLicenceTestSlots(w http.ResponseWriter, r *http.Request) {
	sess, ok := sessionFromContext(r.Context())
	if !ok {
		s.internalError(w, errMissingSessionInContext)
		return
	}

	week := 0
	if weekParam := r.URL.Query().Get("week"); weekParam != "" {
		parsed, err := strconv.Atoi(weekParam)
		if err != nil {
			http.Error(w, "week must be an integer", http.StatusBadRequest)
			return
		}
		week = parsed
	}

	resp, err := s.Upstream.DrivingLicenceTestSlots(r.Context(), sess.AccessToken, week)
	s.forwardUpstreamResponse(w, resp, err)
}

// handleDrivingLicenceIdentity proxies GET /citizen/profile. This app's
// "identity" data endpoint is the shared, registry-backed profile — not a
// separate driving-licence-specific identity source; see
// PORTAL-INTEGRATION-PLAN.md's Component 2 table, which lists
// /citizen/profile as reachable by any of the three apps for exactly this
// purpose.
func (s *Server) handleDrivingLicenceIdentity(w http.ResponseWriter, r *http.Request) {
	sess, ok := sessionFromContext(r.Context())
	if !ok {
		s.internalError(w, errMissingSessionInContext)
		return
	}
	resp, err := s.Upstream.CitizenProfile(r.Context(), sess.AccessToken)
	s.forwardUpstreamResponse(w, resp, err)
}

// handleDrivingLicenceSubmitApplication proxies POST
// /driving-licence/applications. This is a state-changing request, so it
// is CSRF-protected using the same double-submit check handleLogout uses.
// The BFF's own maxRequestBodyBytes middleware (router.go), already
// applied globally, caps the request body before it reaches this handler
// — there is deliberately no second cap here.
func (s *Server) handleDrivingLicenceSubmitApplication(app *AppRoute) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !s.checkCSRF(r, app) {
			http.Error(w, "csrf token missing or invalid", http.StatusForbidden)
			return
		}

		sess, ok := sessionFromContext(r.Context())
		if !ok {
			s.internalError(w, errMissingSessionInContext)
			return
		}

		resp, err := s.Upstream.DrivingLicenceSubmitApplication(r.Context(), sess.AccessToken, r.Body)
		s.forwardUpstreamResponse(w, resp, err)
	}
}
