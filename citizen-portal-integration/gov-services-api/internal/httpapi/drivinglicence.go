package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
)

// licenceApplicationType mirrors citizen-portal-demo-app's
// LicenceApplicationType (types.ts).
type licenceApplicationType struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	Description string `json:"description"`
}

// licenceClass mirrors citizen-portal-demo-app's LicenceClass (types.ts).
type licenceClass struct {
	ID               string `json:"id"`
	Label            string `json:"label"`
	Description      string `json:"description"`
	Eligible         bool   `json:"eligible"`
	IneligibleReason string `json:"ineligibleReason,omitempty"`
}

// editableFieldDef mirrors citizen-portal-demo-app's EditableFieldDef
// (types.ts).
type editableFieldDef struct {
	ID       string `json:"id"`
	Label    string `json:"label"`
	Required bool   `json:"required"`
	Value    string `json:"value"`
	Help     string `json:"help"`
}

// verifiedAttributeLine mirrors citizen-portal-demo-app's
// VerifiedAttributeLine (types.ts).
type verifiedAttributeLine struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

// verifiedIdentitySummary mirrors citizen-portal-demo-app's
// VerifiedIdentitySummary (types.ts).
type verifiedIdentitySummary struct {
	Name       string                  `json:"name"`
	BadgeLabel string                  `json:"badgeLabel"`
	Attributes []verifiedAttributeLine `json:"attributes"`
}

// selfAssertedAttribute mirrors citizen-portal-demo-app's
// SelfAssertedAttribute (types.ts).
type selfAssertedAttribute struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Value string `json:"value"`
}

// declarationQuestion mirrors citizen-portal-demo-app's
// DeclarationQuestion (types.ts).
type declarationQuestion struct {
	ID                string `json:"id"`
	Question          string `json:"question"`
	Help              string `json:"help"`
	YesTriggersReview bool   `json:"yesTriggersReview"`
}

// feeLine mirrors citizen-portal-demo-app's FeeLine (types.ts).
type feeLine struct {
	Label  string `json:"label"`
	Amount string `json:"amount"`
}

// reviewSection mirrors citizen-portal-demo-app's ReviewSection (types.ts).
type reviewSection struct {
	Step  string   `json:"step"`
	Lines []string `json:"lines"`
}

// applicationConfig mirrors citizen-portal-demo-app's ApplicationConfig
// (types.ts).
type applicationConfig struct {
	AppTypes            []licenceApplicationType `json:"appTypes"`
	LicenceClasses      []licenceClass           `json:"licenceClasses"`
	PermitNumber        string                   `json:"permitNumber"`
	PermitIssueDate     string                   `json:"permitIssueDate"`
	EditableFields      []editableFieldDef       `json:"editableFields"`
	VerifiedIdentity    verifiedIdentitySummary  `json:"verifiedIdentity"`
	SelfAsserted        []selfAssertedAttribute  `json:"selfAsserted"`
	Declarations        []declarationQuestion    `json:"declarations"`
	CollectionDistricts []string                 `json:"collectionDistricts"`
	PickupStations      []string                 `json:"pickupStations"`
	FeeBreakdown        []feeLine                `json:"feeBreakdown"`
	TotalFee            string                   `json:"totalFee"`
	Review              []reviewSection          `json:"review"`
}

// applicationConfirmation mirrors citizen-portal-demo-app's
// ApplicationConfirmation (types.ts).
type applicationConfirmation struct {
	Reference          string   `json:"reference"`
	PaymentReference   string   `json:"paymentReference"`
	AmountDue          string   `json:"amountDue"`
	Appointment        string   `json:"appointment"`
	Location           string   `json:"location"`
	ProcessingEstimate string   `json:"processingEstimate"`
	NextSteps          []string `json:"nextSteps"`
}

// testSlot mirrors citizen-portal-demo-app's TestSlot (types.ts).
type testSlot struct {
	Time  string `json:"time"`
	State string `json:"state"`
}

// testDay mirrors citizen-portal-demo-app's TestDay (types.ts).
type testDay struct {
	Day   string     `json:"day"`
	Slots []testSlot `json:"slots"`
}

// drivingLicenceConfig is copied verbatim from
// citizen-portal-demo-app/src/services/applicationService.ts's config
// fixture. It is a static per-app fixture, including the static
// verifiedIdentity block — deliberately not wired to internal/registry,
// which backs the separate, dynamic /citizen/profile endpoint instead (see
// citizen.go). Keeping these as two distinct data sources avoids
// conflating "static per-app fixture" with "registry-backed per-citizen
// record".
var drivingLicenceConfig = applicationConfig{
	AppTypes: []licenceApplicationType{
		{ID: "new", Label: "New licence", Description: "first licence for this class"},
		{ID: "renewal", Label: "Renewal", Description: "within 6 months of expiry"},
		{ID: "duplicate", Label: "Duplicate", Description: "lost or damaged"},
	},
	LicenceClasses: []licenceClass{
		{ID: "a1", Label: "A1 — motorcycle ≤125cc", Description: "from age 16", Eligible: true},
		{ID: "a", Label: "A — motorcycle", Description: "from age 18", Eligible: true},
		{ID: "b1", Label: "B1 — light tricycle", Description: "from age 18", Eligible: true},
		{ID: "b", Label: "B — motor car", Description: "from age 18", Eligible: true},
		{ID: "c1", Label: "C1 — light lorry", Description: "class B held 2 years", Eligible: false, IneligibleReason: "C1 requires a class B licence held for 2 years. You can apply for C1 from 13 Aug 2028, or continue with class B now."},
		{ID: "d1", Label: "D1 — light bus", Description: "class B held 3 years", Eligible: true},
	},
	PermitNumber:    "LP-2026-88214",
	PermitIssueDate: "02 Apr 2026",
	EditableFields: []editableFieldDef{
		{ID: "contact", Label: "Contact number", Required: true, Value: "+94 7•• ••• 220", Help: "Used for test reminders and OTP. Verified by code when changed."},
		{ID: "email", Label: "Email address", Required: true, Value: "john.doe@example.mr", Help: "Where the acknowledgement PDF is sent."},
		{ID: "bloodGroup", Label: "Blood group", Required: true, Value: "O+", Help: "Printed on the licence. Ask your clinic if unsure."},
		{ID: "emergencyContact", Label: "Emergency contact", Required: true, Value: "M. Doe · +94 7•• ••• 118", Help: "Name and number. Not shared with any other service."},
	},
	VerifiedIdentity: verifiedIdentitySummary{
		Name:       "John Doe",
		BadgeLabel: "VERIFIED — NATIONAL DIGITAL ID",
		Attributes: []verifiedAttributeLine{
			{Label: "NIC number", Value: "19•• ••• •••• 4471"},
			{Label: "Date of birth", Value: "04 Mar 1996"},
			{Label: "Address", Value: "14 Lake Road, Marolia City"},
			{Label: "Released to", Value: "Driving Licence Service"},
		},
	},
	SelfAsserted: []selfAssertedAttribute{
		{ID: "contact", Label: "Mobile number", Value: "+94 7•• ••• 220"},
		{ID: "email", Label: "Email", Value: "john.doe@example.mr"},
		{ID: "bloodGroup", Label: "Blood group", Value: "O+"},
	},
	Declarations: []declarationQuestion{
		{ID: "vision", Question: "Do you need corrected vision to drive?", Help: "Glasses or contact lenses. Adds a condition code to your licence.", YesTriggersReview: false},
		{ID: "blackouts", Question: "Have you had epilepsy, seizures or blackouts?", Help: "In the last 5 years, treated or untreated.", YesTriggersReview: true},
		{ID: "colourBlind", Question: "Do you have any colour blindness?", Help: "Red-green deficiency is assessed, not automatically disqualifying.", YesTriggersReview: false},
	},
	CollectionDistricts: []string{"Marolia Central", "Marolia West", "Marolia North", "Marolia South"},
	PickupStations:      []string{"Marolia West licence office", "Marolia Central licence office"},
	FeeBreakdown: []feeLine{
		{Label: "Licence issue fee — class B", Amount: "$14"},
		{Label: "Written test fee", Amount: "$4"},
		{Label: "Card production", Amount: "$2"},
		{Label: "Postal delivery", Amount: "not selected"},
	},
	TotalFee: "$20",
	Review: []reviewSection{
		{Step: "Step 1", Lines: []string{"New licence · class B (motor car)", "Learner permit LP-2026-88214, issued 02 Apr 2026", "Eligibility: passed"}},
		{Step: "Step 2", Lines: []string{"+94 7•• ••• 220 · john.doe@example.mr · O+", "Emergency contact: M. Doe", "Collect at Marolia West licence office, Marolia Central"}},
		{Step: "Step 3", Lines: []string{"Corrected vision: yes → condition code 01", "Epilepsy or blackouts: no · Colour blindness: no", "Medical certificate and learner permit uploaded"}},
		{Step: "Appointment", Lines: []string{"Written test · Thu 20 Aug 2026, 09:30", "Marolia West licence office"}},
	},
}

// drivingLicenceConfirmation is copied verbatim from
// applicationService.ts's submitApplication fixture.
var drivingLicenceConfirmation = applicationConfirmation{
	Reference:          "DL-2026-004871",
	PaymentReference:   "PAY-DL-771204 · valid until 20 Aug 2026",
	AmountDue:          "$20 — unpaid",
	Appointment:        "Written test · Thu 20 Aug 2026, 09:30",
	Location:           "Marolia West licence office",
	ProcessingEstimate: "10 working days after a passed test",
	NextSteps: []string{
		"1 — Pay $20 now or with the reference above at any bank, post office, agent or by USSD *363#.",
		"2 — Bring your national identity card to the test. Nothing printed is required.",
		"3 — Results are recorded against this reference; watch My Timeline.",
		"4 — Your licence appears in My Documents and at your chosen pick-up station.",
	},
}

// testWeekBaseMonday is week 0's Monday, fixed to match the wireframe —
// copied verbatim from applicationService.ts's getTestWeek.
var testWeekBaseMonday = time.Date(2026, time.August, 18, 0, 0, 0, 0, time.UTC)

var testWeekDayLabels = []string{"Mon", "Tue", "Wed", "Thu", "Fri"}
var testWeekTimes = []string{"09:30", "11:00", "14:00"}

// computeTestWeek ports applicationService.ts's getTestWeek deterministic
// algorithm exactly: for weekOffset weeks from testWeekBaseMonday, each of
// the 5 weekdays gets 3 fixed time slots, and a slot's state is "full" when
// (weekOffset*17 + dayIndex*5 + timeIndex*3) % 5 == 0, else "available".
func computeTestWeek(weekOffset int) []testDay {
	week := make([]testDay, len(testWeekDayLabels))
	for dayIdx, label := range testWeekDayLabels {
		date := testWeekBaseMonday.AddDate(0, 0, weekOffset*7+dayIdx)
		slots := make([]testSlot, len(testWeekTimes))
		for timeIdx, t := range testWeekTimes {
			seed := (weekOffset*17 + dayIdx*5 + timeIdx*3) % 5
			state := "available"
			if seed == 0 {
				state = "full"
			}
			slots[timeIdx] = testSlot{Time: t, State: state}
		}
		week[dayIdx] = testDay{Day: label + " " + strconv.Itoa(date.Day()), Slots: slots}
	}
	return week
}

// mountDrivingLicenceRoutes registers the /driving-licence/* handlers on dr.
func (s *Server) mountDrivingLicenceRoutes(dr chi.Router) {
	dr.Get("/config", s.handleDrivingLicenceConfig)
	dr.Get("/test-slots", s.handleDrivingLicenceTestSlots)
	dr.Post("/applications", s.handleDrivingLicenceApplicationSubmit)
}

// handleDrivingLicenceConfig returns the static application configuration
// fixture.
func (s *Server) handleDrivingLicenceConfig(w http.ResponseWriter, r *http.Request) {
	s.writeJSON(w, drivingLicenceConfig)
}

// handleDrivingLicenceTestSlots returns the deterministic mock week of test
// slots for the week query parameter (default 0). Rejects a non-integer
// week with 400.
func (s *Server) handleDrivingLicenceTestSlots(w http.ResponseWriter, r *http.Request) {
	weekParam := r.URL.Query().Get("week")
	weekOffset := 0
	if weekParam != "" {
		parsed, err := strconv.Atoi(weekParam)
		if err != nil {
			http.Error(w, "week must be an integer", http.StatusBadRequest)
			return
		}
		weekOffset = parsed
	}
	s.writeJSON(w, computeTestWeek(weekOffset))
}

// handleDrivingLicenceApplicationSubmit accepts any JSON body without
// validating its shape — matching applicationService.ts's submitApplication
// mock, which also ignores its payload — and returns the fixed application
// confirmation. An empty body is accepted too: the mock never inspects the
// payload, so there is nothing to require of it.
func (s *Server) handleDrivingLicenceApplicationSubmit(w http.ResponseWriter, r *http.Request) {
	var payload any
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil && !errors.Is(err, io.EOF) {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}
	s.writeJSON(w, drivingLicenceConfirmation)
}
