package httpapi

import (
	"errors"
	"net/http"

	"github.com/akila94/mosip-wso2-citizen-portal-demo/gov-services-api/internal/security"
)

// errMissingSubject indicates a handler ran without authmw having stored a
// subject in the request context — a programming error (a route mounted
// without RequireAudienceAndScope), never something a caller can trigger.
var errMissingSubject = errors.New("httpapi: no subject in request context")

// internalError logs the real error (sanitized) but never exposes internal
// detail to the caller — a generic 500 body only. Mirrors
// citizen-portal-bff/internal/httpapi/errors.go's internalError.
func (s *Server) internalError(w http.ResponseWriter, err error) {
	s.Logger.Error("internal error", "error", security.SanitizeForLog(err.Error()))
	http.Error(w, "internal error", http.StatusInternalServerError)
}
