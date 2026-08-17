package httpapi

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// serviceEffort mirrors citizen-portal-demo-app's EffortContract
// (types.ts).
type serviceEffort struct {
	SignInRequired bool   `json:"signInRequired"`
	Signature      string `json:"signature"`
	Steps          int    `json:"steps"`
	TimeEstimate   string `json:"timeEstimate"`
	Fee            string `json:"fee"`
}

// serviceItem mirrors citizen-portal-demo-app's ServiceItem (types.ts).
type serviceItem struct {
	ID             string        `json:"id"`
	Title          string        `json:"title"`
	Description    string        `json:"description"`
	State          string        `json:"state"`
	StepUpRequired bool          `json:"stepUpRequired"`
	Effort         serviceEffort `json:"effort"`
}

// serviceCategory mirrors citizen-portal-demo-app's ServiceCategory
// (types.ts).
type serviceCategory struct {
	ID       string        `json:"id"`
	Name     string        `json:"name"`
	Count    string        `json:"count"`
	Services []serviceItem `json:"services"`
}

// timelineItem mirrors citizen-portal-demo-app's TimelineItem (types.ts).
type timelineItem struct {
	ID          string `json:"id"`
	Date        string `json:"date"`
	Relative    string `json:"relative"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Chip        string `json:"chip"`
	ActionLabel string `json:"actionLabel"`
}

// attributeRecord mirrors citizen-portal-demo-app's AttributeRecord
// (types.ts).
type attributeRecord struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	Value       string `json:"value"`
	Source      string `json:"source"`
	SourceLabel string `json:"sourceLabel"`
	Editable    bool   `json:"editable"`
}

// consentGrant mirrors citizen-portal-demo-app's ConsentGrant (types.ts).
type consentGrant struct {
	ID          string   `json:"id"`
	AppName     string   `json:"appName"`
	Agency      string   `json:"agency"`
	Scopes      []string `json:"scopes"`
	GrantedDate string   `json:"grantedDate"`
}

// walletDocument mirrors citizen-portal-demo-app's WalletDocument
// (types.ts).
type walletDocument struct {
	ID              string `json:"id"`
	Title           string `json:"title"`
	Status          string `json:"status"`
	Number          string `json:"number"`
	IssuedDate      string `json:"issuedDate"`
	ExpiryDate      string `json:"expiryDate"`
	PrimaryAction   string `json:"primaryAction"`
	SecondaryAction string `json:"secondaryAction"`
}

// departmentRecordRow mirrors citizen-portal-demo-app's
// DepartmentRecordRow (types.ts).
type departmentRecordRow struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

// departmentRecord mirrors citizen-portal-demo-app's DepartmentRecord
// (types.ts).
type departmentRecord struct {
	ID         string                `json:"id"`
	Department string                `json:"department"`
	Rows       []departmentRecordRow `json:"rows"`
}

func eff(signInRequired bool, signature string, steps int, timeEstimate, fee string) serviceEffort {
	return serviceEffort{SignInRequired: signInRequired, Signature: signature, Steps: steps, TimeEstimate: timeEstimate, Fee: fee}
}

// serviceCategories is copied verbatim from
// citizen-portal-demo-app/src/services/portalService.ts's
// serviceCategories fixture.
var serviceCategories = []serviceCategory{
	{
		ID: "transport", Name: "Transport", Count: "3 services",
		Services: []serviceItem{
			{ID: "svc-dl", Title: "Apply for a driving licence", Description: "New, renewal or duplicate licence for all classes.", State: "LIVE", StepUpRequired: false, Effort: eff(true, "optional", 4, "12 min", "fee $18")},
			{ID: "svc-vrl", Title: "Renew vehicle revenue licence", Description: "Annual revenue licence for a registered vehicle.", State: "LIVE", StepUpRequired: false, Effort: eff(true, "none", 2, "4 min", "fee from $15")},
			{ID: "svc-transfer", Title: "Transfer vehicle ownership", Description: "Record a change of registered owner.", State: "STUB", StepUpRequired: true, Effort: eff(true, "required", 5, "20 min", "fee $35")},
		},
	},
	{
		ID: "health", Name: "Health", Count: "2 services",
		Services: []serviceItem{
			{ID: "svc-fitness", Title: "Request medical fitness certificate", Description: "Book a fitness assessment at a state clinic.", State: "STUB", StepUpRequired: false, Effort: eff(true, "none", 3, "8 min", "fee $8")},
			{ID: "svc-vaccine", Title: "Vaccination record", Description: "View and share your immunisation history.", State: "STUB", StepUpRequired: false, Effort: eff(true, "none", 1, "1 min", "free")},
		},
	},
	{
		ID: "revenue", Name: "Revenue", Count: "2 services",
		Services: []serviceItem{
			{ID: "svc-tax", Title: "File individual income tax", Description: "Annual return for salaried and self-employed filers.", State: "STUB", StepUpRequired: true, Effort: eff(true, "required", 6, "25 min", "free")},
			{ID: "svc-clearance", Title: "Request tax clearance", Description: "Clearance letter for visa or tender purposes.", State: "STUB", StepUpRequired: false, Effort: eff(true, "none", 2, "5 min", "fee $11")},
		},
	},
	{
		ID: "civil", Name: "Civil registration", Count: "1 service",
		Services: []serviceItem{
			{ID: "svc-birth", Title: "Order a birth certificate copy", Description: "Certified copy delivered or collected.", State: "STUB", StepUpRequired: false, Effort: eff(true, "none", 3, "6 min", "fee $5")},
		},
	},
	{
		ID: "land", Name: "Land", Count: "1 service",
		Services: []serviceItem{
			{ID: "svc-land", Title: "Search land title", Description: "Public title search by deed or parcel number.", State: "STUB", StepUpRequired: false, Effort: eff(false, "none", 1, "2 min", "free")},
		},
	},
}

// timelineFull is copied verbatim from portalService.ts's timelineFull
// fixture.
var timelineFull = []timelineItem{
	{ID: "t1", Date: "12 Sep 2026", Relative: "in 40 days", Title: "Driving licence expires", Description: "Class B · renewal opens 30 days before expiry", Chip: "Action needed", ActionLabel: "Renew"},
	{ID: "t2", Date: "30 Sep 2026", Relative: "next month", Title: "Vehicle revenue licence due — CAB-4471", Description: "Annual renewal · $15 + insurance check", Chip: "Payment due", ActionLabel: "Pay"},
	{ID: "t3", Date: "20 Aug 2026", Relative: "Thursday", Title: "Written test appointment", Description: "Colombo West licence office · 09:30 · bring NIC", Chip: "Appointment", ActionLabel: "View"},
	{ID: "t4", Date: "18 Aug 2026", Relative: "in 5 days", Title: "Driving licence application under review", Description: "Ref DL-2026-004871 · submitted 13 Aug", Chip: "Waiting on government", ActionLabel: "Track"},
	{ID: "t5", Date: "01 Nov 2026", Relative: "in 80 days", Title: "Passport expires", Description: "Renewal available 6 months before expiry", Chip: "For information", ActionLabel: "Remind me"},
}

// portalAttributes is copied verbatim from portalService.ts's attributes
// fixture.
var portalAttributes = []attributeRecord{
	{ID: "a1", Label: "Full name", Value: "John Doe", Source: "verified", SourceLabel: "Verified — National Digital ID", Editable: false},
	{ID: "a2", Label: "NIC number", Value: "19•• ••• •••• 4471", Source: "verified", SourceLabel: "Verified — National Digital ID", Editable: false},
	{ID: "a3", Label: "Date of birth", Value: "04 Mar 1996", Source: "verified", SourceLabel: "Verified — National Digital ID", Editable: false},
	{ID: "a4", Label: "Address", Value: "14 Lake Road, Marolia City", Source: "verified", SourceLabel: "Verified — National Digital ID", Editable: false},
	{ID: "a5", Label: "Photograph", Value: "[ image placeholder ]", Source: "verified", SourceLabel: "Verified — National Digital ID", Editable: false},
	{ID: "a6", Label: "Mobile number", Value: "+94 7•• ••• 220", Source: "self-asserted", SourceLabel: "Self-asserted — verified by OTP", Editable: true},
	{ID: "a7", Label: "Email", Value: "john.doe@example.mr", Source: "self-asserted", SourceLabel: "Self-asserted", Editable: true},
	{ID: "a8", Label: "Blood group", Value: "O+", Source: "self-asserted", SourceLabel: "Self-asserted", Editable: true},
}

// departmentRecords is copied verbatim from portalService.ts's
// departmentRecords fixture.
var departmentRecords = []departmentRecord{
	{ID: "r-transport", Department: "Transport", Rows: []departmentRecordRow{{Label: "Driving licence", Value: "none yet"}, {Label: "Vehicles registered", Value: "1 · CAB-4471"}, {Label: "Demerit points", Value: "0"}}},
	{ID: "r-revenue", Department: "Revenue", Rows: []departmentRecordRow{{Label: "Taxpayer ID", Value: "TIN-••• 8820"}, {Label: "Last return filed", Value: "2025/26"}, {Label: "Outstanding", Value: "none"}}},
	{ID: "r-civil", Department: "Civil registration", Rows: []departmentRecordRow{{Label: "Birth record", Value: "registered 1996"}, {Label: "Marital status", Value: "single"}, {Label: "Dependants", Value: "none"}}},
}

// portalConsents is copied verbatim from portalService.ts's consents
// fixture. Read-only for M3 — consent revoke is deferred to a later
// milestone's write-action integration.
var portalConsents = []consentGrant{
	{ID: "cons-dl", AppName: "Driving Licence Service", Agency: "Dept. of Motor Traffic", Scopes: []string{"openid", "profile", "nic", "dob", "address", "photograph"}, GrantedDate: "13 Aug 2026"},
	{ID: "cons-vrl", AppName: "Vehicle Revenue Licence", Agency: "Provincial Revenue Office", Scopes: []string{"openid", "profile", "nic", "vehicle_registry.read"}, GrantedDate: "13 Aug 2026"},
	{ID: "cons-birth", AppName: "Birth Certificate Service", Agency: "Registrar General", Scopes: []string{"openid", "profile", "nic"}, GrantedDate: "02 Feb 2026"},
}

// walletDocuments is copied verbatim from portalService.ts's
// walletDocuments fixture.
var walletDocuments = []walletDocument{
	{ID: "doc-nic", Title: "National identity card", Status: "VALID", Number: "19•• ••• •••• 4471", IssuedDate: "11 Jan 2014", ExpiryDate: "no expiry", PrimaryAction: "Download", SecondaryAction: "Share"},
	{ID: "doc-dl", Title: "Driving licence", Status: "NOT_ISSUED", Number: "—", IssuedDate: "—", ExpiryDate: "—", PrimaryAction: "Apply now", SecondaryAction: "Learn more"},
	{ID: "doc-vehicle", Title: "Vehicle registration — CAB-4471", Status: "VALID", Number: "CAB-4471", IssuedDate: "02 Jun 2023", ExpiryDate: "30 Sep 2026", PrimaryAction: "Download", SecondaryAction: "Share"},
}

// mountPortalRoutes registers the /portal/* handlers on pr.
func (s *Server) mountPortalRoutes(pr chi.Router) {
	pr.Get("/catalogue", s.handlePortalCatalogue)
	pr.Get("/timeline", s.handlePortalTimeline)
	pr.Get("/attributes", s.handlePortalAttributes)
	pr.Get("/consents", s.handlePortalConsents)
	pr.Get("/documents", s.handlePortalDocuments)
	pr.Get("/department-records", s.handlePortalDepartmentRecords)
}

// handlePortalCatalogue returns the service catalogue, projected by the
// caller-supplied assuranceLevel query parameter. This resource server
// cannot derive assurance level from the access token itself — WSO2 IS's
// JWT access token claims do not include `amr`/`acr` (verified against
// PORTAL-INTEGRATION-PLAN.md's appendix) — so the BFF passes it explicitly
// as a query parameter. This is a deliberate design decision, not an
// oversight: it keeps the token's claim set exactly what IS actually
// issues, rather than inventing a claim IS does not produce.
func (s *Server) handlePortalCatalogue(w http.ResponseWriter, r *http.Request) {
	assuranceLevel := r.URL.Query().Get("assuranceLevel")

	categories := make([]serviceCategory, len(serviceCategories))
	for i, category := range serviceCategories {
		services := make([]serviceItem, len(category.Services))
		for j, service := range category.Services {
			services[j] = service
			if assuranceLevel != "" && assuranceLevel != "none" {
				if service.StepUpRequired {
					services[j].State = "STEP_UP"
				} else {
					services[j].State = "READY"
				}
			}
		}
		categories[i] = category
		categories[i].Services = services
	}
	s.writeJSON(w, categories)
}

// handlePortalTimeline returns the citizen's timeline entries.
func (s *Server) handlePortalTimeline(w http.ResponseWriter, r *http.Request) {
	s.writeJSON(w, timelineFull)
}

// handlePortalAttributes returns the citizen's identity attribute records.
func (s *Server) handlePortalAttributes(w http.ResponseWriter, r *http.Request) {
	s.writeJSON(w, portalAttributes)
}

// handlePortalConsents returns the citizen's consent grants. Read-only for
// M3 — see portalConsents' doc comment.
func (s *Server) handlePortalConsents(w http.ResponseWriter, r *http.Request) {
	s.writeJSON(w, portalConsents)
}

// handlePortalDocuments returns the citizen's wallet documents.
func (s *Server) handlePortalDocuments(w http.ResponseWriter, r *http.Request) {
	s.writeJSON(w, walletDocuments)
}

// handlePortalDepartmentRecords returns the citizen's per-department
// summary records.
func (s *Server) handlePortalDepartmentRecords(w http.ResponseWriter, r *http.Request) {
	s.writeJSON(w, departmentRecords)
}
