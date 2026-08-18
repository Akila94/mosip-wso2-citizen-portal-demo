package httpapi

import (
	"net/http"

	"github.com/akila94/mosip-wso2-citizen-portal-demo/citizen-portal-bff/internal/session"
	"github.com/go-chi/chi/v5"
)

// mountPortalDataRoutes registers the portal app's /bff/portal/api/*
// routes on r. Every route requires a session (the caller wraps r with
// s.requireSession(app) before calling this) and forwards the call to
// gov-services-api via s.Upstream, passing the response through verbatim
// except for an upstream 401 — see forwardSessionUpstreamResponse.
func (s *Server) mountPortalDataRoutes(r chi.Router) {
	r.Get("/catalogue", s.handlePortalCatalogue)
	r.Get("/timeline", s.handlePortalTimeline)
	r.Get("/attributes", s.handlePortalAttributes)
	r.Get("/consents", s.handlePortalConsents)
	r.Get("/documents", s.handlePortalDocuments)
	r.Get("/department-records", s.handlePortalDepartmentRecords)
}

// handlePublicPortalCatalogue proxies GET /public/catalogue — the only
// route on this BFF that is served without a session, and the only one that
// calls gov-services-api with no access token at all.
//
// It exists because the portal's landing page shows the service catalogue to
// signed-out visitors (LandingScreen renders ServiceCatalogue in both the
// signed-in and signed-out branches): a government service catalogue is
// public information, so requiring a citizen session to read it would be
// wrong on the merits as well as broken on the screen.
//
// Being public is not the same as being unguarded. This route still runs the
// whole middleware chain — security headers (including the strict API
// Content-Security-Policy, since this is a JSON API response and not the
// SPA), the 64 kB request-body cap — and the upstream client still bounds
// the response body it reads. What it does not do is authenticate, and it
// takes nothing at all from the caller: no assurance level, no path segment,
// no query. It is one named handler for one named upstream path, which is
// what keeps it from becoming the generic unauthenticated proxy this project
// deliberately does not have.
func (s *Server) handlePublicPortalCatalogue(w http.ResponseWriter, r *http.Request) {
	// no-store matches every other data route here. The payload is identical
	// for every visitor, so this is consistency rather than a privacy need.
	w.Header().Set("Cache-Control", "no-store")
	resp, err := s.Upstream.PublicPortalCatalogue(r.Context())
	// The only caller of the verbatim forwarder: with no session behind it
	// there is no access token whose rejection could need explaining.
	s.forwardPublicUpstreamResponse(w, resp, err)
}

// handlePortalCatalogue proxies GET /portal/catalogue. assuranceLevel is
// deliberately derived server-side from the verified session's `amr`
// claim via session.DeriveAssuranceLevel, never taken from the incoming
// request: accepting a caller-supplied assuranceLevel would let a client
// self-report a higher assurance level than it actually authenticated
// with, unlocking STEP_UP-gated catalogue entries without ever stepping
// up. Any assuranceLevel query parameter on the incoming BFF request is
// therefore ignored.
func (s *Server) handlePortalCatalogue(w http.ResponseWriter, r *http.Request) {
	sess, ok := sessionFromContext(r.Context())
	if !ok {
		s.internalError(w, errMissingSessionInContext)
		return
	}
	assuranceLevel := string(session.DeriveAssuranceLevel(sess.Amr))
	resp, err := s.Upstream.PortalCatalogue(r.Context(), sess.AccessToken, assuranceLevel)
	s.forwardSessionUpstreamResponse(w, r, sess, resp, err)
}

// handlePortalTimeline proxies GET /portal/timeline.
func (s *Server) handlePortalTimeline(w http.ResponseWriter, r *http.Request) {
	sess, ok := sessionFromContext(r.Context())
	if !ok {
		s.internalError(w, errMissingSessionInContext)
		return
	}
	resp, err := s.Upstream.PortalTimeline(r.Context(), sess.AccessToken)
	s.forwardSessionUpstreamResponse(w, r, sess, resp, err)
}

// handlePortalAttributes proxies GET /portal/attributes.
func (s *Server) handlePortalAttributes(w http.ResponseWriter, r *http.Request) {
	sess, ok := sessionFromContext(r.Context())
	if !ok {
		s.internalError(w, errMissingSessionInContext)
		return
	}
	resp, err := s.Upstream.PortalAttributes(r.Context(), sess.AccessToken)
	s.forwardSessionUpstreamResponse(w, r, sess, resp, err)
}

// handlePortalConsents proxies GET /portal/consents.
func (s *Server) handlePortalConsents(w http.ResponseWriter, r *http.Request) {
	sess, ok := sessionFromContext(r.Context())
	if !ok {
		s.internalError(w, errMissingSessionInContext)
		return
	}
	resp, err := s.Upstream.PortalConsents(r.Context(), sess.AccessToken)
	s.forwardSessionUpstreamResponse(w, r, sess, resp, err)
}

// handlePortalDocuments proxies GET /portal/documents.
func (s *Server) handlePortalDocuments(w http.ResponseWriter, r *http.Request) {
	sess, ok := sessionFromContext(r.Context())
	if !ok {
		s.internalError(w, errMissingSessionInContext)
		return
	}
	resp, err := s.Upstream.PortalDocuments(r.Context(), sess.AccessToken)
	s.forwardSessionUpstreamResponse(w, r, sess, resp, err)
}

// handlePortalDepartmentRecords proxies GET /portal/department-records.
func (s *Server) handlePortalDepartmentRecords(w http.ResponseWriter, r *http.Request) {
	sess, ok := sessionFromContext(r.Context())
	if !ok {
		s.internalError(w, errMissingSessionInContext)
		return
	}
	resp, err := s.Upstream.PortalDepartmentRecords(r.Context(), sess.AccessToken)
	s.forwardSessionUpstreamResponse(w, r, sess, resp, err)
}
