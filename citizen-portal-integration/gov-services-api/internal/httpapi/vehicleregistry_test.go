package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/akila94/mosip-wso2-citizen-portal-demo/gov-services-api/internal/authmw"
	"github.com/akila94/mosip-wso2-citizen-portal-demo/gov-services-api/internal/registry"
	"github.com/go-chi/chi/v5"
)

func TestLast6Digits(t *testing.T) {
	tests := []struct {
		name  string
		input int64
		want  string
	}{
		{"long number takes last 6", 1755423000123, "000123"},
		{"exactly 6 digits returned whole", 123456, "123456"},
		{"fewer than 6 digits returned whole", 42, "42"},
		{"zero returned whole", 0, "0"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := last6Digits(tc.input); got != tc.want {
				t.Errorf("last6Digits(%d) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

// vehicleRegistryTestRouter mounts only the /vehicle-registry/* routes
// behind the real authmw.RequireAudienceAndScope middleware, so requests
// pass through the same signature/audience/scope pipeline a live
// deployment uses.
func vehicleRegistryTestRouter(v *authmw.Verifier, reg *registry.Registry) http.Handler {
	s := NewServer(reg, testLogger())
	r := chi.NewRouter()
	r.Route("/vehicle-registry", func(vr chi.Router) {
		vr.Use(authmw.RequireAudienceAndScope(v, []string{"vrl-client"}, ""))
		s.mountVehicleRegistryRoutes(vr)
	})
	return r
}

func TestHandleVehicleRegistryRenewNotFoundVehicle(t *testing.T) {
	as := newTestAuthServer(t)
	v := as.verifier(t)
	reg := registry.New()
	router := vehicleRegistryTestRouter(v, reg)

	tok := as.sign(t, "sub-1", "vrl-client", "vehicle_registry.read")
	req := httptest.NewRequest(http.MethodPost, "/vehicle-registry/vehicles/no-such-id/renew", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404, body: %s", rr.Code, rr.Body.String())
	}
}

func TestHandleVehicleRegistryRenewHappyPath(t *testing.T) {
	as := newTestAuthServer(t)
	v := as.verifier(t)
	reg := registry.New()
	router := vehicleRegistryTestRouter(v, reg)

	// Deliberately does not call reg.GetOrSeed first — a citizen renewing
	// on their very first request must still find their seeded vehicle.
	tok := as.sign(t, "sub-1", "vrl-client", "vehicle_registry.read")
	req := httptest.NewRequest(http.MethodPost, "/vehicle-registry/vehicles/veh-cab4471/renew", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body: %s", rr.Code, rr.Body.String())
	}
	var receipt vehicleRenewalReceipt
	if err := json.Unmarshal(rr.Body.Bytes(), &receipt); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(receipt.ReceiptRef) < len("PAY-VRL-") {
		t.Errorf("ReceiptRef = %q", receipt.ReceiptRef)
	}
}

func TestHandleVehicleRegistryListReturnsSubsVehicles(t *testing.T) {
	as := newTestAuthServer(t)
	v := as.verifier(t)
	reg := registry.New()
	router := vehicleRegistryTestRouter(v, reg)

	tok := as.sign(t, "sub-1", "vrl-client", "vehicle_registry.read")
	req := httptest.NewRequest(http.MethodGet, "/vehicle-registry/vehicles", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	var vehicles []registry.Vehicle
	if err := json.Unmarshal(rr.Body.Bytes(), &vehicles); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(vehicles) != 1 || vehicles[0].ID != "veh-cab4471" {
		t.Errorf("vehicles = %+v", vehicles)
	}
}

// TestHandleVehicleRegistryListAcceptsTokenWithNoScope asserts the current,
// deliberate access model for /vehicle-registry/*: the audience check alone
// (the token's aud contains vrl-client) is sufficient, with no custom scope
// required at all — a token carrying no scope claim whatsoever must still
// be accepted.
func TestHandleVehicleRegistryListAcceptsTokenWithNoScope(t *testing.T) {
	as := newTestAuthServer(t)
	v := as.verifier(t)
	reg := registry.New()
	router := vehicleRegistryTestRouter(v, reg)

	tok := as.sign(t, "sub-1", "vrl-client", "") // no scope claim value at all
	req := httptest.NewRequest(http.MethodGet, "/vehicle-registry/vehicles", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body: %s", rr.Code, rr.Body.String())
	}
	var vehicles []registry.Vehicle
	if err := json.Unmarshal(rr.Body.Bytes(), &vehicles); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(vehicles) != 1 || vehicles[0].ID != "veh-cab4471" {
		t.Errorf("vehicles = %+v", vehicles)
	}
}

func TestHandleVehicleRegistryListRejectsWrongAudience(t *testing.T) {
	as := newTestAuthServer(t)
	v := as.verifier(t)
	reg := registry.New()
	router := vehicleRegistryTestRouter(v, reg)

	// A token minted for a different application's client ID must not
	// reach the vehicle registry router.
	tok := as.sign(t, "sub-1", "portal-client", "vehicle_registry.read")
	req := httptest.NewRequest(http.MethodGet, "/vehicle-registry/vehicles", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rr.Code)
	}
}
