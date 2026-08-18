package httpapi

import (
	"context"
	"io"

	"github.com/akila94/mosip-wso2-citizen-portal-demo/citizen-portal-bff/internal/upstream"
)

// fakeUpstream is a test double for UpstreamClient — no network, so
// data-route handler tests exercise the HTTP-layer logic (session
// resolution, assurance-level derivation, CSRF, response passthrough) in
// isolation from internal/upstream, which has its own test suite against
// a real httptest.Server. Every method's return value is settable per test
// via the response/err fields, mirroring fakeClient's exchangeErr/
// verifyErr style; every call's arguments are recorded for assertions.
type fakeUpstream struct {
	response upstream.Response
	err      error

	lastAccessToken    string
	lastAssuranceLevel string
	lastWeek           int
	lastVehicleID      string
	lastMethod         string
}

func (f *fakeUpstream) result() (upstream.Response, error) {
	if f.err != nil {
		return upstream.Response{}, f.err
	}
	if f.response.StatusCode == 0 {
		return upstream.Response{StatusCode: 200, ContentType: "application/json", Body: []byte(`{"ok":true}`)}, nil
	}
	return f.response, nil
}

// PublicPortalCatalogue records only that it was called: it deliberately
// has no access-token parameter, so a test asserting lastAccessToken == ""
// is asserting a structural fact, not a convention.
func (f *fakeUpstream) PublicPortalCatalogue(_ context.Context) (upstream.Response, error) {
	f.lastMethod = "PublicPortalCatalogue"
	return f.result()
}

func (f *fakeUpstream) PortalCatalogue(_ context.Context, accessToken, assuranceLevel string) (upstream.Response, error) {
	f.lastMethod = "PortalCatalogue"
	f.lastAccessToken = accessToken
	f.lastAssuranceLevel = assuranceLevel
	return f.result()
}

func (f *fakeUpstream) PortalTimeline(_ context.Context, accessToken string) (upstream.Response, error) {
	f.lastMethod = "PortalTimeline"
	f.lastAccessToken = accessToken
	return f.result()
}

func (f *fakeUpstream) PortalAttributes(_ context.Context, accessToken string) (upstream.Response, error) {
	f.lastMethod = "PortalAttributes"
	f.lastAccessToken = accessToken
	return f.result()
}

func (f *fakeUpstream) PortalConsents(_ context.Context, accessToken string) (upstream.Response, error) {
	f.lastMethod = "PortalConsents"
	f.lastAccessToken = accessToken
	return f.result()
}

func (f *fakeUpstream) PortalDocuments(_ context.Context, accessToken string) (upstream.Response, error) {
	f.lastMethod = "PortalDocuments"
	f.lastAccessToken = accessToken
	return f.result()
}

func (f *fakeUpstream) PortalDepartmentRecords(_ context.Context, accessToken string) (upstream.Response, error) {
	f.lastMethod = "PortalDepartmentRecords"
	f.lastAccessToken = accessToken
	return f.result()
}

func (f *fakeUpstream) DrivingLicenceConfig(_ context.Context, accessToken string) (upstream.Response, error) {
	f.lastMethod = "DrivingLicenceConfig"
	f.lastAccessToken = accessToken
	return f.result()
}

func (f *fakeUpstream) DrivingLicenceTestSlots(_ context.Context, accessToken string, week int) (upstream.Response, error) {
	f.lastMethod = "DrivingLicenceTestSlots"
	f.lastAccessToken = accessToken
	f.lastWeek = week
	return f.result()
}

func (f *fakeUpstream) DrivingLicenceSubmitApplication(_ context.Context, accessToken string, _ io.Reader) (upstream.Response, error) {
	f.lastMethod = "DrivingLicenceSubmitApplication"
	f.lastAccessToken = accessToken
	return f.result()
}

func (f *fakeUpstream) VehicleRegistryVehicles(_ context.Context, accessToken string) (upstream.Response, error) {
	f.lastMethod = "VehicleRegistryVehicles"
	f.lastAccessToken = accessToken
	return f.result()
}

func (f *fakeUpstream) VehicleRegistryRenew(_ context.Context, accessToken, vehicleID string) (upstream.Response, error) {
	f.lastMethod = "VehicleRegistryRenew"
	f.lastAccessToken = accessToken
	f.lastVehicleID = vehicleID
	return f.result()
}

func (f *fakeUpstream) CitizenProfile(_ context.Context, accessToken string) (upstream.Response, error) {
	f.lastMethod = "CitizenProfile"
	f.lastAccessToken = accessToken
	return f.result()
}

var _ UpstreamClient = (*fakeUpstream)(nil)
