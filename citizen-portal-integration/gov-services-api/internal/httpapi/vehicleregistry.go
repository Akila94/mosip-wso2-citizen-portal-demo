package httpapi

import (
	"net/http"
	"strconv"
	"time"

	"github.com/akila94/mosip-wso2-citizen-portal-demo/gov-services-api/internal/authmw"
	"github.com/akila94/mosip-wso2-citizen-portal-demo/gov-services-api/internal/security"
	"github.com/go-chi/chi/v5"
)

// vehicleRenewalReceipt is the response body for a successful vehicle
// renewal.
type vehicleRenewalReceipt struct {
	ReceiptRef string `json:"receiptRef"`
}

// mountVehicleRegistryRoutes registers the /vehicle-registry/* handlers on
// vr. Unlike /portal and /driving-licence, this router is registry-backed —
// vehicles are pulled from internal/registry against the citizen's `sub`,
// per PORTAL-INTEGRATION-PLAN.md's note that these are "pulled from the
// Transport registry against the citizen's verified NIC".
func (s *Server) mountVehicleRegistryRoutes(vr chi.Router) {
	vr.Get("/vehicles", s.handleVehicleRegistryList)
	vr.Post("/vehicles/{id}/renew", s.handleVehicleRegistryRenew)
}

// handleVehicleRegistryList returns the caller's registered vehicles,
// looked up by the verified token's `sub` — never by any value asserted in
// the token itself beyond that opaque identifier.
func (s *Server) handleVehicleRegistryList(w http.ResponseWriter, r *http.Request) {
	sub, ok := authmw.SubjectFromContext(r.Context())
	if !ok {
		s.internalError(w, errMissingSubject)
		return
	}
	record := s.Registry.GetOrSeed(sub)
	s.writeJSON(w, record.Vehicles)
}

// handleVehicleRegistryRenew renews the named vehicle for the caller's
// sub, returning 404 if it does not exist for that sub's record, else 200
// with a receipt reference derived from the current time — mirroring
// revenueLicenceService.ts's renewLicence.
func (s *Server) handleVehicleRegistryRenew(w http.ResponseWriter, r *http.Request) {
	sub, ok := authmw.SubjectFromContext(r.Context())
	if !ok {
		s.internalError(w, errMissingSubject)
		return
	}

	// Ensure sub's record exists before attempting the mutation — a
	// citizen renewing on their very first request (never having called
	// GET /vehicles first) must still find their seeded vehicle.
	s.Registry.GetOrSeed(sub)

	vehicleID := chi.URLParam(r, "id")
	if _, ok := s.Registry.RenewVehicle(sub, vehicleID); !ok {
		s.Logger.Warn("vehicle renewal requested for unknown vehicle", "vehicleID", security.SanitizeForLog(vehicleID))
		http.Error(w, "vehicle not found", http.StatusNotFound)
		return
	}

	s.writeJSON(w, vehicleRenewalReceipt{ReceiptRef: "PAY-VRL-" + last6Digits(time.Now().UnixMilli())})
}

// last6Digits formats n as a decimal string and returns its last 6
// characters, or the whole string if it has fewer than 6 — mirroring
// revenueLicenceService.ts's `Date.now().toString().slice(-6)` without
// panicking on a short input.
func last6Digits(n int64) string {
	s := strconv.FormatInt(n, 10)
	if len(s) <= 6 {
		return s
	}
	return s[len(s)-6:]
}
