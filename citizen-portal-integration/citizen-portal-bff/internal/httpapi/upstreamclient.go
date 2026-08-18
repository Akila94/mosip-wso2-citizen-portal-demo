package httpapi

import (
	"context"
	"io"

	"github.com/akila94/mosip-wso2-citizen-portal-demo/citizen-portal-bff/internal/upstream"
)

// UpstreamClient is the subset of *upstream.Client the data-route handlers
// depend on. *upstream.Client satisfies this interface structurally;
// declaring it here lets handler tests substitute a fake without a running
// gov-services-api, while production code always passes the real thing —
// the same pattern OIDCClient already establishes for oidcrp.RP.
type UpstreamClient interface {
	// PublicPortalCatalogue takes no access token by construction: it is the
	// one call in this interface made with no citizen session behind it, and
	// having no token parameter is what guarantees none can be sent.
	PublicPortalCatalogue(ctx context.Context) (upstream.Response, error)

	PortalCatalogue(ctx context.Context, accessToken, assuranceLevel string) (upstream.Response, error)
	PortalTimeline(ctx context.Context, accessToken string) (upstream.Response, error)
	PortalAttributes(ctx context.Context, accessToken string) (upstream.Response, error)
	PortalConsents(ctx context.Context, accessToken string) (upstream.Response, error)
	PortalDocuments(ctx context.Context, accessToken string) (upstream.Response, error)
	PortalDepartmentRecords(ctx context.Context, accessToken string) (upstream.Response, error)

	DrivingLicenceConfig(ctx context.Context, accessToken string) (upstream.Response, error)
	DrivingLicenceTestSlots(ctx context.Context, accessToken string, week int) (upstream.Response, error)
	DrivingLicenceSubmitApplication(ctx context.Context, accessToken string, body io.Reader) (upstream.Response, error)

	VehicleRegistryVehicles(ctx context.Context, accessToken string) (upstream.Response, error)
	VehicleRegistryRenew(ctx context.Context, accessToken, vehicleID string) (upstream.Response, error)

	CitizenProfile(ctx context.Context, accessToken string) (upstream.Response, error)
}

var _ UpstreamClient = (*upstream.Client)(nil)
