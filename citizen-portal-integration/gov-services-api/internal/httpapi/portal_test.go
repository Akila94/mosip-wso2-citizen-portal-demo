package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandlePortalCatalogueNoneAssuranceKeepsFixtureState(t *testing.T) {
	s := NewServer(nil, testLogger())
	req := httptest.NewRequest(http.MethodGet, "/portal/catalogue?assuranceLevel=none", nil)
	rr := httptest.NewRecorder()
	s.handlePortalCatalogue(rr, req)

	var categories []serviceCategory
	if err := json.Unmarshal(rr.Body.Bytes(), &categories); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	transport := findCategory(categories, "transport")
	dl := findService(transport.Services, "svc-dl")
	if dl.State != "LIVE" {
		t.Errorf("svc-dl state with assuranceLevel=none = %q, want LIVE (fixture value)", dl.State)
	}
	transfer := findService(transport.Services, "svc-transfer")
	if transfer.State != "STUB" {
		t.Errorf("svc-transfer state with assuranceLevel=none = %q, want STUB (fixture value)", transfer.State)
	}
}

func TestHandlePortalCatalogueAbsentAssuranceKeepsFixtureState(t *testing.T) {
	s := NewServer(nil, testLogger())
	req := httptest.NewRequest(http.MethodGet, "/portal/catalogue", nil)
	rr := httptest.NewRecorder()
	s.handlePortalCatalogue(rr, req)

	var categories []serviceCategory
	if err := json.Unmarshal(rr.Body.Bytes(), &categories); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	transport := findCategory(categories, "transport")
	dl := findService(transport.Services, "svc-dl")
	if dl.State != "LIVE" {
		t.Errorf("svc-dl state with no assuranceLevel = %q, want LIVE", dl.State)
	}
}

func TestHandlePortalCatalogueAuthenticatedDerivesStepUpOrReady(t *testing.T) {
	s := NewServer(nil, testLogger())
	req := httptest.NewRequest(http.MethodGet, "/portal/catalogue?assuranceLevel=basic", nil)
	rr := httptest.NewRecorder()
	s.handlePortalCatalogue(rr, req)

	var categories []serviceCategory
	if err := json.Unmarshal(rr.Body.Bytes(), &categories); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	transport := findCategory(categories, "transport")

	// svc-transfer has stepUpRequired=true in the fixture -> STEP_UP.
	transfer := findService(transport.Services, "svc-transfer")
	if transfer.State != "STEP_UP" {
		t.Errorf("svc-transfer state = %q, want STEP_UP", transfer.State)
	}

	// svc-dl has stepUpRequired=false in the fixture -> READY.
	dl := findService(transport.Services, "svc-dl")
	if dl.State != "READY" {
		t.Errorf("svc-dl state = %q, want READY", dl.State)
	}
}

func TestHandlePortalTimelineReturnsFixture(t *testing.T) {
	s := NewServer(nil, testLogger())
	req := httptest.NewRequest(http.MethodGet, "/portal/timeline", nil)
	rr := httptest.NewRecorder()
	s.handlePortalTimeline(rr, req)

	var items []timelineItem
	if err := json.Unmarshal(rr.Body.Bytes(), &items); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(items) != 5 {
		t.Errorf("len(items) = %d, want 5", len(items))
	}
}

func TestHandlePortalAttributesReturnsFixture(t *testing.T) {
	s := NewServer(nil, testLogger())
	req := httptest.NewRequest(http.MethodGet, "/portal/attributes", nil)
	rr := httptest.NewRecorder()
	s.handlePortalAttributes(rr, req)

	var attrs []attributeRecord
	if err := json.Unmarshal(rr.Body.Bytes(), &attrs); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(attrs) != 8 {
		t.Errorf("len(attrs) = %d, want 8", len(attrs))
	}
}

func TestHandlePortalConsentsReturnsFixture(t *testing.T) {
	s := NewServer(nil, testLogger())
	req := httptest.NewRequest(http.MethodGet, "/portal/consents", nil)
	rr := httptest.NewRecorder()
	s.handlePortalConsents(rr, req)

	var consents []consentGrant
	if err := json.Unmarshal(rr.Body.Bytes(), &consents); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(consents) != 3 {
		t.Errorf("len(consents) = %d, want 3", len(consents))
	}
}

func TestHandlePortalDocumentsReturnsFixture(t *testing.T) {
	s := NewServer(nil, testLogger())
	req := httptest.NewRequest(http.MethodGet, "/portal/documents", nil)
	rr := httptest.NewRecorder()
	s.handlePortalDocuments(rr, req)

	var docs []walletDocument
	if err := json.Unmarshal(rr.Body.Bytes(), &docs); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(docs) != 3 {
		t.Errorf("len(docs) = %d, want 3", len(docs))
	}
}

func TestHandlePortalDepartmentRecordsReturnsFixture(t *testing.T) {
	s := NewServer(nil, testLogger())
	req := httptest.NewRequest(http.MethodGet, "/portal/department-records", nil)
	rr := httptest.NewRecorder()
	s.handlePortalDepartmentRecords(rr, req)

	var records []departmentRecord
	if err := json.Unmarshal(rr.Body.Bytes(), &records); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(records) != 3 {
		t.Errorf("len(records) = %d, want 3", len(records))
	}
}

func findCategory(categories []serviceCategory, id string) serviceCategory {
	for _, c := range categories {
		if c.ID == id {
			return c
		}
	}
	return serviceCategory{}
}

func findService(services []serviceItem, id string) serviceItem {
	for _, svc := range services {
		if svc.ID == id {
			return svc
		}
	}
	return serviceItem{}
}
