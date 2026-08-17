package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestComputeTestWeekMatchesHandComputedSeeds asserts specific
// (weekOffset, dayIndex, timeIndex) -> state triples against values
// hand-computed from applicationService.ts's getTestWeek formula:
// seed = (weekOffset*17 + dayIndex*5 + timeIndex*3) % 5; state is "full"
// when seed == 0, else "available".
//
// For weekOffset=0, dayIndex*5 is always a multiple of 5, so
// (dayIndex*5 + timeIndex*3) % 5 reduces to (timeIndex*3) % 5, which is 0
// only at timeIndex=0 — every day's 09:30 slot is "full" and the other two
// slots are "available" for every day of week 0.
//
// For weekOffset=1, the added 17 shifts every day's residue by 17%5=2, so
// timeIndex=0 becomes seed 2 ("available") and timeIndex=1 becomes seed 0
// ("full") for every day of week 1.
func TestComputeTestWeekMatchesHandComputedSeeds(t *testing.T) {
	tests := []struct {
		name       string
		weekOffset int
		dayIndex   int
		timeIndex  int
		wantState  string
	}{
		{"week0 Mon 09:30 is full", 0, 0, 0, "full"},
		{"week0 Mon 11:00 is available", 0, 0, 1, "available"},
		{"week0 Mon 14:00 is available", 0, 0, 2, "available"},
		{"week0 Fri 09:30 is full", 0, 4, 0, "full"},
		{"week0 Fri 11:00 is available", 0, 4, 1, "available"},
		{"week1 Mon 09:30 is available", 1, 0, 0, "available"},
		{"week1 Mon 11:00 is full", 1, 0, 1, "full"},
		{"week1 Mon 14:00 is available", 1, 0, 2, "available"},
		{"week1 Wed 09:30 is available", 1, 2, 0, "available"},
		{"week1 Wed 11:00 is full", 1, 2, 1, "full"},
		{"week1 Fri 11:00 is full", 1, 4, 1, "full"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			week := computeTestWeek(tc.weekOffset)
			got := week[tc.dayIndex].Slots[tc.timeIndex].State
			if got != tc.wantState {
				t.Errorf("computeTestWeek(%d)[%d].Slots[%d].State = %q, want %q", tc.weekOffset, tc.dayIndex, tc.timeIndex, got, tc.wantState)
			}
		})
	}
}

func TestComputeTestWeekDayLabelsWeek0(t *testing.T) {
	week := computeTestWeek(0)
	wantLabels := []string{"Mon 18", "Tue 19", "Wed 20", "Thu 21", "Fri 22"}
	for i, want := range wantLabels {
		if week[i].Day != want {
			t.Errorf("week[%d].Day = %q, want %q", i, week[i].Day, want)
		}
	}
}

func TestComputeTestWeekDayLabelsWeek1(t *testing.T) {
	week := computeTestWeek(1)
	wantLabels := []string{"Mon 25", "Tue 26", "Wed 27", "Thu 28", "Fri 29"}
	for i, want := range wantLabels {
		if week[i].Day != want {
			t.Errorf("week[%d].Day = %q, want %q", i, week[i].Day, want)
		}
	}
}

func TestHandleDrivingLicenceTestSlotsDefaultsToWeekZero(t *testing.T) {
	s := NewServer(nil, testLogger())
	req := httptest.NewRequest(http.MethodGet, "/driving-licence/test-slots", nil)
	rr := httptest.NewRecorder()
	s.handleDrivingLicenceTestSlots(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	var days []testDay
	if err := json.Unmarshal(rr.Body.Bytes(), &days); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(days) != 5 || days[0].Day != "Mon 18" {
		t.Errorf("unexpected week: %+v", days)
	}
}

func TestHandleDrivingLicenceTestSlotsRejectsNonIntegerWeek(t *testing.T) {
	s := NewServer(nil, testLogger())
	req := httptest.NewRequest(http.MethodGet, "/driving-licence/test-slots?week=not-a-number", nil)
	rr := httptest.NewRecorder()
	s.handleDrivingLicenceTestSlots(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rr.Code)
	}
}

func TestHandleDrivingLicenceApplicationSubmitIgnoresPayloadShape(t *testing.T) {
	s := NewServer(nil, testLogger())
	body := []byte(`{"anything": "goes", "nested": {"a": 1}}`)
	req := httptest.NewRequest(http.MethodPost, "/driving-licence/applications", newJSONBody(body))
	rr := httptest.NewRecorder()
	s.handleDrivingLicenceApplicationSubmit(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body: %s", rr.Code, rr.Body.String())
	}
	var confirmation applicationConfirmation
	if err := json.Unmarshal(rr.Body.Bytes(), &confirmation); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if confirmation.Reference != "DL-2026-004871" {
		t.Errorf("Reference = %q", confirmation.Reference)
	}
}

func TestHandleDrivingLicenceConfigReturnsFixture(t *testing.T) {
	s := NewServer(nil, testLogger())
	req := httptest.NewRequest(http.MethodGet, "/driving-licence/config", nil)
	rr := httptest.NewRecorder()
	s.handleDrivingLicenceConfig(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	var cfg applicationConfig
	if err := json.Unmarshal(rr.Body.Bytes(), &cfg); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if cfg.TotalFee != "$20" || cfg.VerifiedIdentity.Name != "John Doe" {
		t.Errorf("unexpected config: %+v", cfg)
	}
}
