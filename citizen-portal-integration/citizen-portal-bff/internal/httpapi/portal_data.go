package httpapi

import (
	"net/http"

	"github.com/akila94/mosip-wso2-citizen-portal-demo/citizen-portal-bff/internal/session"
	"github.com/go-chi/chi/v5"
)

// mountPortalDataRoutes registers the portal app's /bff/portal/api/*
// routes on r. Every route requires a session (the caller wraps r with
// s.requireSession(app) before calling this) and forwards the call to
// gov-services-api via s.Upstream, passing the response through verbatim.
func (s *Server) mountPortalDataRoutes(r chi.Router) {
	r.Get("/catalogue", s.handlePortalCatalogue)
	r.Get("/timeline", s.handlePortalTimeline)
	r.Get("/attributes", s.handlePortalAttributes)
	r.Get("/consents", s.handlePortalConsents)
	r.Get("/documents", s.handlePortalDocuments)
	r.Get("/department-records", s.handlePortalDepartmentRecords)
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
	s.forwardUpstreamResponse(w, resp, err)
}

// handlePortalTimeline proxies GET /portal/timeline.
func (s *Server) handlePortalTimeline(w http.ResponseWriter, r *http.Request) {
	sess, ok := sessionFromContext(r.Context())
	if !ok {
		s.internalError(w, errMissingSessionInContext)
		return
	}
	resp, err := s.Upstream.PortalTimeline(r.Context(), sess.AccessToken)
	s.forwardUpstreamResponse(w, resp, err)
}

// handlePortalAttributes proxies GET /portal/attributes.
func (s *Server) handlePortalAttributes(w http.ResponseWriter, r *http.Request) {
	sess, ok := sessionFromContext(r.Context())
	if !ok {
		s.internalError(w, errMissingSessionInContext)
		return
	}
	resp, err := s.Upstream.PortalAttributes(r.Context(), sess.AccessToken)
	s.forwardUpstreamResponse(w, resp, err)
}

// handlePortalConsents proxies GET /portal/consents.
func (s *Server) handlePortalConsents(w http.ResponseWriter, r *http.Request) {
	sess, ok := sessionFromContext(r.Context())
	if !ok {
		s.internalError(w, errMissingSessionInContext)
		return
	}
	resp, err := s.Upstream.PortalConsents(r.Context(), sess.AccessToken)
	s.forwardUpstreamResponse(w, resp, err)
}

// handlePortalDocuments proxies GET /portal/documents.
func (s *Server) handlePortalDocuments(w http.ResponseWriter, r *http.Request) {
	sess, ok := sessionFromContext(r.Context())
	if !ok {
		s.internalError(w, errMissingSessionInContext)
		return
	}
	resp, err := s.Upstream.PortalDocuments(r.Context(), sess.AccessToken)
	s.forwardUpstreamResponse(w, resp, err)
}

// handlePortalDepartmentRecords proxies GET /portal/department-records.
func (s *Server) handlePortalDepartmentRecords(w http.ResponseWriter, r *http.Request) {
	sess, ok := sessionFromContext(r.Context())
	if !ok {
		s.internalError(w, errMissingSessionInContext)
		return
	}
	resp, err := s.Upstream.PortalDepartmentRecords(r.Context(), sess.AccessToken)
	s.forwardUpstreamResponse(w, resp, err)
}
