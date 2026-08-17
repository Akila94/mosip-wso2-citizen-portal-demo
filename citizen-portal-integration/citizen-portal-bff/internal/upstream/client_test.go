package upstream

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"
)

// echoServer returns an httptest.Server that echoes back the Authorization
// header it received and the raw query string, as a small JSON-ish body,
// so tests can assert both without gov-services-api's real response shapes.
func echoServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"authorization":"` + r.Header.Get("Authorization") + `","query":"` + r.URL.RawQuery + `","method":"` + r.Method + `","path":"` + r.URL.Path + `"}`))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func newTestClient(t *testing.T, baseURL string) *Client {
	t.Helper()
	return New(baseURL, &http.Client{Timeout: 5 * time.Second})
}

func TestPortalCatalogueSendsBearerTokenAndAssuranceLevelQuery(t *testing.T) {
	srv := echoServer(t)
	c := newTestClient(t, srv.URL)

	resp, err := c.PortalCatalogue(context.Background(), "token-abc", "substantial")
	if err != nil {
		t.Fatalf("PortalCatalogue: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	body := string(resp.Body)
	if !strings.Contains(body, `"authorization":"Bearer token-abc"`) {
		t.Errorf("body = %s, want Authorization header echoed", body)
	}
	if !strings.Contains(body, `"query":"assuranceLevel=substantial"`) {
		t.Errorf("body = %s, want assuranceLevel query echoed", body)
	}
}

func TestDrivingLicenceTestSlotsSendsWeekQuery(t *testing.T) {
	srv := echoServer(t)
	c := newTestClient(t, srv.URL)

	resp, err := c.DrivingLicenceTestSlots(context.Background(), "token-xyz", 3)
	if err != nil {
		t.Fatalf("DrivingLicenceTestSlots: %v", err)
	}
	body := string(resp.Body)
	if !strings.Contains(body, `"query":"week=`+strconv.Itoa(3)+`"`) {
		t.Errorf("body = %s, want week=3 query echoed", body)
	}
}

func TestVehicleRegistryRenewPostsToCorrectPath(t *testing.T) {
	srv := echoServer(t)
	c := newTestClient(t, srv.URL)

	resp, err := c.VehicleRegistryRenew(context.Background(), "token-1", "CAB-4471")
	if err != nil {
		t.Fatalf("VehicleRegistryRenew: %v", err)
	}
	body := string(resp.Body)
	if !strings.Contains(body, `"method":"POST"`) {
		t.Errorf("body = %s, want POST method echoed", body)
	}
	if !strings.Contains(body, `"path":"/vehicle-registry/vehicles/CAB-4471/renew"`) {
		t.Errorf("body = %s, want the vehicle ID in the path", body)
	}
}

func TestDoReturnsResponseNotErrorForNonSuccessStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "vehicle not found", http.StatusNotFound)
	}))
	t.Cleanup(srv.Close)
	c := newTestClient(t, srv.URL)

	resp, err := c.VehicleRegistryRenew(context.Background(), "token-1", "does-not-exist")
	if err != nil {
		t.Fatalf("expected no Go error for a non-2xx upstream status, got %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("StatusCode = %d, want 404", resp.StatusCode)
	}
}

func TestDoReturnsErrorOnTransportFailure(t *testing.T) {
	// A closed server's address refuses connections, giving a genuine
	// transport failure rather than an HTTP response.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	closedAddr := srv.URL
	srv.Close()

	c := newTestClient(t, closedAddr)
	_, err := c.CitizenProfile(context.Background(), "token-1")
	if err == nil {
		t.Fatal("expected a transport error calling a closed server, got nil")
	}
}

func TestPortalCatalogueOmitsQueryWhenAssuranceLevelEmpty(t *testing.T) {
	srv := echoServer(t)
	c := newTestClient(t, srv.URL)

	resp, err := c.PortalCatalogue(context.Background(), "token-abc", "")
	if err != nil {
		t.Fatalf("PortalCatalogue: %v", err)
	}
	body := string(resp.Body)
	if !strings.Contains(body, `"query":""`) {
		t.Errorf("body = %s, want empty query when assuranceLevel is empty", body)
	}
}
