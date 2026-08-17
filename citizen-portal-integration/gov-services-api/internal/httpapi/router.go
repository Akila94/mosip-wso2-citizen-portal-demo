package httpapi

import (
	"log/slog"
	"net/http"

	"github.com/akila94/mosip-wso2-citizen-portal-demo/gov-services-api/internal/authmw"
	"github.com/akila94/mosip-wso2-citizen-portal-demo/gov-services-api/internal/registry"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// maxRequestBodyBytes caps every request body. Mirrors
// citizen-portal-bff/internal/httpapi/router.go's own 64 kB cap
// (esignet-bridge/server.js's convention).
const maxRequestBodyBytes = 64 * 1024

// NewRouter builds gov-services-api's HTTP handler: one route group per
// resource domain, each gated by authmw.RequireAudienceAndScope with its
// own required audience (the client ID of the application allowed to call
// it). No router in this project requires a custom scope on top of the
// audience check — a citizen only ever holds an access token whose
// audience is the specific app they authenticated to, so the audience
// check alone already proves "this citizen has a validly-authenticated
// session with this app".
func NewRouter(v *authmw.Verifier, portalClientID, dlClientID, vrlClientID string, reg *registry.Registry, logger *slog.Logger) http.Handler {
	s := NewServer(reg, logger)

	r := chi.NewRouter()
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(requestBodyLimit)

	r.Route("/portal", func(pr chi.Router) {
		pr.Use(authmw.RequireAudienceAndScope(v, []string{portalClientID}, ""))
		s.mountPortalRoutes(pr)
	})

	r.Route("/driving-licence", func(dr chi.Router) {
		dr.Use(authmw.RequireAudienceAndScope(v, []string{dlClientID}, ""))
		s.mountDrivingLicenceRoutes(dr)
	})

	r.Route("/vehicle-registry", func(vr chi.Router) {
		vr.Use(authmw.RequireAudienceAndScope(v, []string{vrlClientID}, ""))
		s.mountVehicleRegistryRoutes(vr)
	})

	r.Route("/citizen", func(cr chi.Router) {
		cr.Use(authmw.RequireAudienceAndScope(v, []string{portalClientID, dlClientID, vrlClientID}, ""))
		cr.Get("/profile", s.handleCitizenProfile)
	})

	return r
}

// requestBodyLimit caps every request body at maxRequestBodyBytes, matching
// the BFF's own convention.
func requestBodyLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
		next.ServeHTTP(w, r)
	})
}
