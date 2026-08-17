package httpapi

import (
	"net/http"

	"github.com/akila94/mosip-wso2-citizen-portal-demo/gov-services-api/internal/authmw"
)

// citizenProfile is the /citizen/profile response shape, projected by the
// caller's granted scopes. omitempty on every optional field means an
// absent field is simply not present, never present-but-null.
type citizenProfile struct {
	Sub       string `json:"sub"`
	Name      string `json:"name,omitempty"`
	Birthdate string `json:"birthdate,omitempty"`
	NIC       string `json:"nic,omitempty"`
	Address   string `json:"address,omitempty"`
}

// handleCitizenProfile returns the registry record for the caller's `sub`,
// projected by which scopes the presented token actually carries. This is
// the one endpoint any of the three applications' tokens can reach
// (PORTAL-INTEGRATION-PLAN.md Component 2: "any of the three | the
// registry record, projected by the caller's scopes").
//
// The scope-to-field mapping below is a documented design decision, not a
// literal requirement from the plan (which leaves it unspecified):
//   - `sub` is always included — it is the opaque pairwise identifier, never
//     a real NIC number, and identifies "whose record this is" regardless of
//     scope.
//   - `profile` unlocks name/birthdate — the conventional OIDC `profile`
//     scope's subject matter.
//   - `address` unlocks nic/address, grouped together — NIC is exactly the
//     kind of sensitive government-issued identifier the `address` scope is
//     the closest real match for among this project's actually-configured
//     scopes; there is no dedicated `nic` scope in
//     PORTAL-INTEGRATION-PLAN.md's Component 4 scope tables, so this reuses
//     `address` rather than inventing one.
//
// Vehicle data is deliberately not exposed here: the custom
// `vehicle_registry.read` scope that used to gate it has been removed
// project-wide (audience alone now gates /vehicle-registry/*), and the
// authoritative, non-duplicated source for a citizen's vehicles is
// GET /vehicle-registry/vehicles.
func (s *Server) handleCitizenProfile(w http.ResponseWriter, r *http.Request) {
	sub, ok := authmw.SubjectFromContext(r.Context())
	if !ok {
		s.internalError(w, errMissingSubject)
		return
	}
	scopes, _ := authmw.ScopesFromContext(r.Context())
	grantedScopes := make(map[string]bool, len(scopes))
	for _, scope := range scopes {
		grantedScopes[scope] = true
	}

	record := s.Registry.GetOrSeed(sub)
	profile := citizenProfile{Sub: sub}

	if grantedScopes["profile"] {
		profile.Name = record.Name
		profile.Birthdate = record.Birthdate
	}
	if grantedScopes["address"] {
		profile.NIC = record.NIC
		profile.Address = record.Address
	}

	s.writeJSON(w, profile)
}
