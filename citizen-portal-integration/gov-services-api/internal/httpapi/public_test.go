package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/akila94/mosip-wso2-citizen-portal-demo/gov-services-api/internal/registry"
)

// publicRouter builds the real router, so these tests prove the public
// catalogue really is mounted outside every authmw group rather than merely
// that its handler function works when called directly.
func publicRouter(t *testing.T) http.Handler {
	t.Helper()
	as := newTestAuthServer(t)
	return NewRouter(as.verifier(t), "portal-client", "dl-client", "vrl-client", registry.New(), testLogger())
}

func getPublicCatalogue(t *testing.T, router http.Handler, target string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, target, nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	return rr
}

func TestPublicCatalogueNeedsNoAuthorizationHeader(t *testing.T) {
	rr := getPublicCatalogue(t, publicRouter(t), "/public/catalogue")

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 with no Authorization header at all, body: %s", rr.Code, rr.Body.String())
	}
	if ct := rr.Header().Get("Content-Type"); !strings.Contains(ct, "application/json") {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}

	var categories []serviceCategory
	if err := json.Unmarshal(rr.Body.Bytes(), &categories); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(categories) != len(serviceCategories) {
		t.Errorf("len(categories) = %d, want %d — the same fixture the authenticated route serves", len(categories), len(serviceCategories))
	}
}

func TestPublicCatalogueReturnsTheFixtureStatesWithNoPromotion(t *testing.T) {
	rr := getPublicCatalogue(t, publicRouter(t), "/public/catalogue")

	var categories []serviceCategory
	if err := json.Unmarshal(rr.Body.Bytes(), &categories); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	for _, category := range categories {
		for _, service := range category.Services {
			if service.State == "READY" || service.State == "STEP_UP" {
				t.Errorf("service %s state = %q — the signed-out catalogue must not promote anything", service.ID, service.State)
			}
		}
	}

	transport := findCategory(categories, "transport")
	if dl := findService(transport.Services, "svc-dl"); dl.State != "LIVE" {
		t.Errorf("svc-dl state = %q, want the fixture's own LIVE", dl.State)
	}
	if transfer := findService(transport.Services, "svc-transfer"); transfer.State != "STUB" {
		t.Errorf("svc-transfer state = %q, want the fixture's own STUB", transfer.State)
	}
}

// A caller-supplied assurance level is exactly the privilege escalation the
// authenticated route's design note warns about; on a route with no session
// to derive one from, it must have no effect whatsoever.
func TestPublicCatalogueIgnoresACallerSuppliedAssuranceLevel(t *testing.T) {
	router := publicRouter(t)
	plain := getPublicCatalogue(t, router, "/public/catalogue").Body.String()

	for _, target := range []string{
		"/public/catalogue?assuranceLevel=substantial",
		"/public/catalogue?assuranceLevel=basic",
		"/public/catalogue?assuranceLevel=none",
	} {
		rr := getPublicCatalogue(t, router, target)
		if rr.Code != http.StatusOK {
			t.Errorf("GET %s: status = %d, want 200", target, rr.Code)
			continue
		}
		if rr.Body.String() != plain {
			t.Errorf("GET %s changed the response — assuranceLevel must not be honoured here", target)
		}
	}
}

func TestPublicCatalogueExposesNothingCitizenSpecific(t *testing.T) {
	body := getPublicCatalogue(t, publicRouter(t), "/public/catalogue").Body.String()

	// Values drawn from every citizen-bearing fixture in this service: the
	// attribute records, the department records, the timeline and the
	// registry seed. None of them may appear on an unauthenticated route.
	for _, citizenValue := range []string{
		"John Doe",
		"john.doe@example.mr",
		"NIC",
		"TIN-",
		"CAB-4471",
		"sub",
		"Taxpayer",
		"Demerit",
	} {
		if strings.Contains(body, citizenValue) {
			t.Errorf("public catalogue leaked citizen-specific content %q: %s", citizenValue, body)
		}
	}
}

// The public route must not have loosened anything next to it. authmw
// answers a request with no Authorization header at all with 400 ("missing
// or malformed Authorization header") and reserves 401 for a header that is
// present but carries an invalid token; either way the catalogue behind it
// must not be served.
func TestPortalCatalogueStillRequiresAToken(t *testing.T) {
	rr := getPublicCatalogue(t, publicRouter(t), "/portal/catalogue")

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("unauthenticated /portal/catalogue status = %d, want 400 from authmw", rr.Code)
	}
	if strings.Contains(rr.Body.String(), "svc-dl") {
		t.Errorf("unauthenticated /portal/catalogue served catalogue content: %s", rr.Body.String())
	}
}

func TestPortalCatalogueStillRejectsAnInvalidToken(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/portal/catalogue", nil)
	req.Header.Set("Authorization", "Bearer not-a-real-token")
	rr := httptest.NewRecorder()
	publicRouter(t).ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401 for an invalid token", rr.Code)
	}
}

func TestPortalCatalogueStillPromotesForAnAuthenticatedCaller(t *testing.T) {
	as := newTestAuthServer(t)
	router := NewRouter(as.verifier(t), "portal-client", "dl-client", "vrl-client", registry.New(), testLogger())

	req := httptest.NewRequest(http.MethodGet, "/portal/catalogue?assuranceLevel=basic", nil)
	req.Header.Set("Authorization", "Bearer "+as.sign(t, "sub-1", "portal-client", "openid"))
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body: %s", rr.Code, rr.Body.String())
	}
	var categories []serviceCategory
	if err := json.Unmarshal(rr.Body.Bytes(), &categories); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	transport := findCategory(categories, "transport")
	if dl := findService(transport.Services, "svc-dl"); dl.State != "READY" {
		t.Errorf("authenticated svc-dl state = %q, want READY — the authenticated route must be unchanged", dl.State)
	}
}

// The public handler must never mutate the shared fixture the authenticated
// handler also serves: they are one source of truth.
func TestPublicCatalogueLeavesTheSharedFixtureUntouched(t *testing.T) {
	before, err := json.Marshal(serviceCategories)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	router := publicRouter(t)
	getPublicCatalogue(t, router, "/public/catalogue")
	getPublicCatalogue(t, router, "/public/catalogue?assuranceLevel=substantial")

	after, err := json.Marshal(serviceCategories)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(before) != string(after) {
		t.Error("the public handler mutated the shared serviceCategories fixture")
	}
}
