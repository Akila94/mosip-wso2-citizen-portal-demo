// Package registry is the sub-keyed citizen registry: PORTAL-INTEGRATION-PLAN.md's
// Component 2 requires that NIC, address and vehicle records come from
// here, never from the token — eSignet's `sub` is a pairwise pseudonymous
// identifier and the registry is what makes that safe: it is looked up by
// `sub`, and a government-facing NIC/address/vehicle value is attached to
// that opaque key, not asserted by the identity provider.
package registry

import "sync"

// RegistryCheck is one automated check recorded against a vehicle (an
// insurance or emissions verification, for example).
type RegistryCheck struct {
	Label  string `json:"label"`
	Status string `json:"status"`
}

// Vehicle is one registered vehicle record.
type Vehicle struct {
	ID          string          `json:"id"`
	Plate       string          `json:"plate"`
	Description string          `json:"description"`
	DueDate     string          `json:"dueDate"`
	Fee         string          `json:"fee"`
	Owner       string          `json:"owner"`
	Province    string          `json:"province"`
	Registry    []RegistryCheck `json:"registryChecks"`
}

// Record is one citizen's registry entry, keyed by their IS `sub`.
type Record struct {
	Sub       string
	Name      string
	NIC       string
	Birthdate string
	Address   string
	Vehicles  []Vehicle
}

// Registry is a thread-safe, in-memory, sub-keyed store. Unlike the BFF's
// session store it has no TTL/eviction — a citizen's registry record is a
// persistent government record for the lifetime of this demo process, not
// session state.
type Registry struct {
	mu      sync.RWMutex
	records map[string]Record
}

// New constructs an empty Registry.
func New() *Registry {
	return &Registry{records: make(map[string]Record)}
}

// GetOrSeed returns sub's record, creating and storing a freshly seeded one
// on first sight of an unrecognized sub — this is what lets the demo work
// with whatever pairwise `sub` eSignet mints for a given login, without any
// pre-registration step. Every newly seeded record gets identical demo
// fixture data — this is a demo registry, not a real one; the point is
// proving the sub-keyed-not-token-derived architecture, not simulating
// distinct citizens.
func (r *Registry) GetOrSeed(sub string) Record {
	r.mu.RLock()
	record, ok := r.records[sub]
	r.mu.RUnlock()
	if ok {
		return record
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	// Re-check under the write lock in case another goroutine seeded this
	// sub between the RUnlock above and acquiring the write lock.
	if record, ok := r.records[sub]; ok {
		return record
	}
	record = seedRecord(sub)
	r.records[sub] = record
	return record
}

// RenewVehicle finds vehicleID within sub's record and marks it renewed,
// returning the updated Vehicle and whether it was found. It reports false
// if sub has no record yet or the vehicle ID does not exist within it.
func (r *Registry) RenewVehicle(sub, vehicleID string) (Vehicle, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	record, ok := r.records[sub]
	if !ok {
		return Vehicle{}, false
	}

	for i, vehicle := range record.Vehicles {
		if vehicle.ID == vehicleID {
			record.Vehicles[i].DueDate = "RENEWED · valid to 30 Sep 2027"
			r.records[sub] = record
			return record.Vehicles[i], true
		}
	}
	return Vehicle{}, false
}

// seedRecord builds the fixed demo fixture data for a newly seen sub,
// copied verbatim from citizen-portal-demo-app/src/services/portalService.ts's
// `attributes` array and revenueLicenceService.ts's `vehicles` array.
func seedRecord(sub string) Record {
	return Record{
		Sub:       sub,
		Name:      "John Doe",
		NIC:       "19•• ••• •••• 4471",
		Birthdate: "04 Mar 1996",
		Address:   "14 Lake Road, Marolia City",
		Vehicles: []Vehicle{
			{
				ID:          "veh-cab4471",
				Plate:       "CAB-4471",
				Description: "Motor car · 1,300cc · petrol",
				DueDate:     "DUE 30 SEP",
				Fee:         "$15",
				Owner:       "J. Doe (you)",
				Province:    "Marolia Central",
				Registry: []RegistryCheck{
					{Label: "Insurance check", Status: "Valid to 12 Nov 2026 · confirmed with your insurer automatically"},
					{Label: "Emissions test", Status: "Passed 04 Jun 2026 · valid"},
				},
			},
		},
	}
}
