// Package httpapi is gov-services-api's HTTP layer: chi routers for the
// four resource groups (/portal, /driving-licence, /vehicle-registry,
// /citizen), each gated by internal/authmw's per-router required audience
// and scope, mirroring citizen-portal-bff/internal/httpapi's
// one-package-many-files style.
package httpapi

import (
	"log/slog"

	"github.com/akila94/mosip-wso2-citizen-portal-demo/gov-services-api/internal/registry"
)

// Server holds everything the HTTP handlers need: the citizen registry and
// a structured logger.
type Server struct {
	Registry *registry.Registry
	Logger   *slog.Logger
}

// NewServer constructs a Server ready to build a router via NewRouter.
func NewServer(reg *registry.Registry, logger *slog.Logger) *Server {
	return &Server{Registry: reg, Logger: logger}
}
