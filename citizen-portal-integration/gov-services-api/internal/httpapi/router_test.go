package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/akila94/mosip-wso2-citizen-portal-demo/gov-services-api/internal/registry"
)

// TestNewRouterEnforcesPerRouterAudience is the M3 acceptance scenario: a
// token minted for Application A (the driving-licence client) must be
// genuinely rejected by Application B's router (vehicle-registry), and
// vice versa, even though both tokens are otherwise valid, signed by the
// same IS.
func TestNewRouterEnforcesPerRouterAudience(t *testing.T) {
	as := newTestAuthServer(t)
	v := as.verifier(t)
	reg := registry.New()
	router := NewRouter(v, "portal-client", "dl-client", "vrl-client", reg, testLogger())

	dlToken := as.sign(t, "sub-1", "dl-client", "driving_licence.write")
	vrlToken := as.sign(t, "sub-1", "vrl-client", "vehicle_registry.read")

	tests := []struct {
		name       string
		method     string
		path       string
		token      string
		wantStatus int
	}{
		{"dl token allowed on driving-licence router", http.MethodGet, "/driving-licence/config", dlToken, http.StatusOK},
		{"dl token rejected by vehicle-registry router", http.MethodGet, "/vehicle-registry/vehicles", dlToken, http.StatusForbidden},
		{"vrl token allowed on vehicle-registry router", http.MethodGet, "/vehicle-registry/vehicles", vrlToken, http.StatusOK},
		{"vrl token rejected by driving-licence router", http.MethodGet, "/driving-licence/config", vrlToken, http.StatusForbidden},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			req.Header.Set("Authorization", "Bearer "+tc.token)
			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)
			if rr.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d, body: %s", rr.Code, tc.wantStatus, rr.Body.String())
			}
		})
	}
}

func TestNewRouterPortalRequiresNoSpecificScope(t *testing.T) {
	as := newTestAuthServer(t)
	v := as.verifier(t)
	reg := registry.New()
	router := NewRouter(v, "portal-client", "dl-client", "vrl-client", reg, testLogger())

	tok := as.sign(t, "sub-1", "portal-client", "openid")
	req := httptest.NewRequest(http.MethodGet, "/portal/timeline", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body: %s", rr.Code, rr.Body.String())
	}
}

// TestNewRouterDrivingLicenceAcceptsAudienceOnly asserts the current,
// deliberate access model for /driving-licence/*: the audience check alone
// (the token's aud contains dl-client) is sufficient. There is no custom
// scope requirement on top of it — a citizen only ever gets an access token
// whose audience is the specific app they authenticated to, so the audience
// check already proves "this citizen has a validly-authenticated session
// with the Driving Licence Service".
func TestNewRouterDrivingLicenceAcceptsAudienceOnly(t *testing.T) {
	as := newTestAuthServer(t)
	v := as.verifier(t)
	reg := registry.New()
	router := NewRouter(v, "portal-client", "dl-client", "vrl-client", reg, testLogger())

	tok := as.sign(t, "sub-1", "dl-client", "openid") // no custom scope at all
	req := httptest.NewRequest(http.MethodGet, "/driving-licence/config", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body: %s", rr.Code, rr.Body.String())
	}
}
