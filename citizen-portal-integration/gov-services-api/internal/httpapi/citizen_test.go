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

func citizenTestRouter(v *authmw.Verifier, reg *registry.Registry) http.Handler {
	s := NewServer(reg, testLogger())
	r := chi.NewRouter()
	r.Route("/citizen", func(cr chi.Router) {
		cr.Use(authmw.RequireAudienceAndScope(v, []string{"portal-client", "dl-client", "vrl-client"}, ""))
		cr.Get("/profile", s.handleCitizenProfile)
	})
	return r
}

func requestProfile(t *testing.T, router http.Handler, token string) citizenProfile {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/citizen/profile", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body: %s", rr.Code, rr.Body.String())
	}
	var profile citizenProfile
	if err := json.Unmarshal(rr.Body.Bytes(), &profile); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	return profile
}

func TestHandleCitizenProfileNoScopesReturnsOnlySub(t *testing.T) {
	as := newTestAuthServer(t)
	v := as.verifier(t)
	reg := registry.New()
	router := citizenTestRouter(v, reg)

	tok := as.sign(t, "sub-1", "portal-client", "openid")
	profile := requestProfile(t, router, tok)

	if profile.Sub != "sub-1" {
		t.Errorf("Sub = %q", profile.Sub)
	}
	if profile.Name != "" || profile.Birthdate != "" || profile.NIC != "" || profile.Address != "" {
		t.Errorf("expected only sub with no matching scopes, got %+v", profile)
	}
}

func TestHandleCitizenProfileProfileScopeAddsNameAndBirthdate(t *testing.T) {
	as := newTestAuthServer(t)
	v := as.verifier(t)
	reg := registry.New()
	router := citizenTestRouter(v, reg)

	tok := as.sign(t, "sub-1", "portal-client", "openid profile")
	profile := requestProfile(t, router, tok)

	if profile.Name != "John Doe" || profile.Birthdate != "04 Mar 1996" {
		t.Errorf("profile scope did not add name/birthdate: %+v", profile)
	}
	if profile.NIC != "" || profile.Address != "" {
		t.Errorf("profile scope should not add nic/address: %+v", profile)
	}
}

func TestHandleCitizenProfileAddressScopeAddsNICAndAddress(t *testing.T) {
	as := newTestAuthServer(t)
	v := as.verifier(t)
	reg := registry.New()
	router := citizenTestRouter(v, reg)

	tok := as.sign(t, "sub-1", "dl-client", "openid address")
	profile := requestProfile(t, router, tok)

	if profile.NIC != "19•• ••• •••• 4471" || profile.Address != "14 Lake Road, Marolia City" {
		t.Errorf("address scope did not add nic/address: %+v", profile)
	}
	if profile.Name != "" || profile.Birthdate != "" {
		t.Errorf("address scope should not add name/birthdate: %+v", profile)
	}
}

func TestHandleCitizenProfileAllScopesTogether(t *testing.T) {
	as := newTestAuthServer(t)
	v := as.verifier(t)
	reg := registry.New()
	router := citizenTestRouter(v, reg)

	tok := as.sign(t, "sub-1", "portal-client", "openid profile address")
	profile := requestProfile(t, router, tok)

	if profile.Name != "John Doe" || profile.Birthdate != "04 Mar 1996" {
		t.Errorf("missing profile fields: %+v", profile)
	}
	if profile.NIC == "" || profile.Address == "" {
		t.Errorf("missing address fields: %+v", profile)
	}
}

func TestHandleCitizenProfileAcceptsAnyOfTheThreeAudiences(t *testing.T) {
	as := newTestAuthServer(t)
	v := as.verifier(t)
	reg := registry.New()
	router := citizenTestRouter(v, reg)

	for _, aud := range []string{"portal-client", "dl-client", "vrl-client"} {
		tok := as.sign(t, "sub-1", aud, "openid")
		req := httptest.NewRequest(http.MethodGet, "/citizen/profile", nil)
		req.Header.Set("Authorization", "Bearer "+tok)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("aud=%s: status = %d, want 200", aud, rr.Code)
		}
	}
}
