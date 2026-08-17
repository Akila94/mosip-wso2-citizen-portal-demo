// This file holds the shared plumbing the three apps' /bff/{app}/api/...
// data-route handlers (portal_data.go, drivinglicence_data.go,
// revenuelicence_data.go) all use: reading the session requireSession
// stored, and forwarding an upstream.Response through to the caller.
package httpapi

import (
	"errors"
	"net/http"

	"github.com/akila94/mosip-wso2-citizen-portal-demo/citizen-portal-bff/internal/security"
	"github.com/akila94/mosip-wso2-citizen-portal-demo/citizen-portal-bff/internal/upstream"
)

// errMissingSessionInContext indicates a data-route handler ran without
// requireSession middleware having populated the request context first —
// a wiring bug, never a caller-triggerable condition, hence the 500 rather
// than a 401.
var errMissingSessionInContext = errors.New("httpapi: session missing from request context (requireSession middleware not applied)")

// defaultUpstreamContentType is used when gov-services-api's response
// carries no Content-Type header, matching this project's convention of
// never leaving a response's Content-Type unset.
const defaultUpstreamContentType = "application/json"

// forwardUpstreamResponse writes resp's status code and body through to w
// verbatim when err is nil. A non-nil err means the call to
// gov-services-api itself failed (a genuine transport failure — see
// upstream.Client.do's doc comment) rather than gov-services-api returning
// a non-2xx status, so it is logged (sanitized) and answered with 502; it
// is never gov-services-api's own error response, which is instead
// forwarded as-is by the resp branch below.
func (s *Server) forwardUpstreamResponse(w http.ResponseWriter, resp upstream.Response, err error) {
	if err != nil {
		s.Logger.Error("upstream call to gov-services-api failed", "error", security.SanitizeForLog(err.Error()))
		http.Error(w, "upstream service unavailable", http.StatusBadGateway)
		return
	}

	contentType := resp.ContentType
	if contentType == "" {
		contentType = defaultUpstreamContentType
	}
	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(resp.StatusCode)
	if _, writeErr := w.Write(resp.Body); writeErr != nil {
		s.Logger.Warn("failed writing upstream response body to client", "error", writeErr.Error())
	}
}
