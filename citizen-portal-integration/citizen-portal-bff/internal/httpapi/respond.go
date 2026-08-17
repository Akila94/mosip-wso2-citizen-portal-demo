package httpapi

import (
	"encoding/json"
	"net/http"
)

// writeJSON encodes v to w, logging (not panicking on) an encode failure —
// which can only happen if the client disconnects mid-write, at which point
// there is nothing further to send the caller.
func (s *Server) writeJSON(w http.ResponseWriter, v any) {
	if err := json.NewEncoder(w).Encode(v); err != nil {
		s.Logger.Warn("failed writing JSON response", "error", err.Error())
	}
}
