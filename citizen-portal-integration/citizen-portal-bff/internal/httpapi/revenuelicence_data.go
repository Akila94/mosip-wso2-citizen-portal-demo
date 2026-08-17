package httpapi

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// mountRevenueLicenceDataRoutes registers the revenue-licence app's
// /bff/revenue-licence/api/* routes on r. Every route requires a session
// (the caller wraps r with s.requireSession(app) before calling this).
// app is threaded through only to the CSRF-protected write route, which
// needs it to check the double-submit token.
func (s *Server) mountRevenueLicenceDataRoutes(r chi.Router, app *AppRoute) {
	r.Get("/vehicles", s.handleVehicleRegistryVehicles)
	r.Get("/identity", s.handleRevenueLicenceIdentity)
	r.Post("/vehicles/{id}/renew", s.handleVehicleRegistryRenew(app))
}

// handleVehicleRegistryVehicles proxies GET /vehicle-registry/vehicles.
func (s *Server) handleVehicleRegistryVehicles(w http.ResponseWriter, r *http.Request) {
	sess, ok := sessionFromContext(r.Context())
	if !ok {
		s.internalError(w, errMissingSessionInContext)
		return
	}
	resp, err := s.Upstream.VehicleRegistryVehicles(r.Context(), sess.AccessToken)
	s.forwardUpstreamResponse(w, resp, err)
}

// handleRevenueLicenceIdentity proxies GET /citizen/profile — the same
// shared, registry-backed profile the driving-licence app's /identity
// route reaches (see drivinglicence_data.go's handleDrivingLicenceIdentity
// doc comment).
func (s *Server) handleRevenueLicenceIdentity(w http.ResponseWriter, r *http.Request) {
	sess, ok := sessionFromContext(r.Context())
	if !ok {
		s.internalError(w, errMissingSessionInContext)
		return
	}
	resp, err := s.Upstream.CitizenProfile(r.Context(), sess.AccessToken)
	s.forwardUpstreamResponse(w, resp, err)
}

// handleVehicleRegistryRenew proxies POST
// /vehicle-registry/vehicles/{id}/renew. This is a state-changing request,
// so it is CSRF-protected using the same double-submit check handleLogout
// uses.
func (s *Server) handleVehicleRegistryRenew(app *AppRoute) http.HandlerFunc {
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

		vehicleID := chi.URLParam(r, "id")
		resp, err := s.Upstream.VehicleRegistryRenew(r.Context(), sess.AccessToken, vehicleID)
		s.forwardUpstreamResponse(w, resp, err)
	}
}
