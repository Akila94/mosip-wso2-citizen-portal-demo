package httpapi

import (
	"net/http"

	"github.com/akila94/mosip-wso2-citizen-portal-demo/citizen-portal-bff/internal/security"
)

// internalError logs the real error (sanitized) but never exposes internal
// detail to the caller — a generic 500 body only.
func (s *Server) internalError(w http.ResponseWriter, err error) {
	s.Logger.Error("internal error", "error", security.SanitizeForLog(err.Error()))
	http.Error(w, "internal error", http.StatusInternalServerError)
}
