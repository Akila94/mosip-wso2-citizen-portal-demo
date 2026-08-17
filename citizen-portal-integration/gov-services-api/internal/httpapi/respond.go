package httpapi

import (
	"encoding/json"
	"net/http"
)

// writeJSON encodes v to w as JSON, logging (not panicking on) an encode
// failure — which can only happen if the client disconnects mid-write, at
// which point there is nothing further to send the caller. Mirrors
// citizen-portal-bff/internal/httpapi/respond.go's writeJSON.
func (s *Server) writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		s.Logger.Warn("failed writing JSON response", "error", err.Error())
	}
}
