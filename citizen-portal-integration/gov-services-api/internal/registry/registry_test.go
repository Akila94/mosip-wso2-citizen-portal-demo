package registry

import (
	"fmt"
	"reflect"
	"sync"
	"testing"
)

func TestGetOrSeedCreatesSeededRecordForUnseenSub(t *testing.T) {
	r := New()
	record := r.GetOrSeed("sub-1")

	if record.Sub != "sub-1" {
		t.Errorf("Sub = %q, want sub-1", record.Sub)
	}
	if record.Name != "John Doe" {
		t.Errorf("Name = %q, want John Doe", record.Name)
	}
	if record.NIC != "19•• ••• •••• 4471" {
		t.Errorf("NIC = %q", record.NIC)
	}
	if record.Birthdate != "04 Mar 1996" {
		t.Errorf("Birthdate = %q", record.Birthdate)
	}
	if record.Address != "14 Lake Road, Marolia City" {
		t.Errorf("Address = %q", record.Address)
	}
	if len(record.Vehicles) != 1 {
		t.Fatalf("len(Vehicles) = %d, want 1", len(record.Vehicles))
	}
	v := record.Vehicles[0]
	if v.ID != "veh-cab4471" || v.Plate != "CAB-4471" || v.DueDate != "DUE 30 SEP" {
		t.Errorf("unexpected seeded vehicle: %+v", v)
	}
}

func TestGetOrSeedReturnsSameRecordOnSecondCall(t *testing.T) {
	r := New()
	first := r.GetOrSeed("sub-1")
	second := r.GetOrSeed("sub-1")

	if !reflect.DeepEqual(first, second) {
		t.Errorf("expected identical records for repeated GetOrSeed calls on the same sub: %+v vs %+v", first, second)
	}
}

func TestGetOrSeedGivesIndependentRecordsPerSub(t *testing.T) {
	r := New()
	subA := r.GetOrSeed("sub-a")
	subB := r.GetOrSeed("sub-b")

	if subA.Sub == subB.Sub {
		t.Fatal("expected different subs to produce records with different Sub fields")
	}

	// Mutating one sub's vehicle via RenewVehicle must never affect the
	// other sub's independently-seeded backing storage.
	if _, ok := r.RenewVehicle("sub-a", "veh-cab4471"); !ok {
		t.Fatal("expected RenewVehicle to find veh-cab4471 for sub-a")
	}
	stillOriginal := r.GetOrSeed("sub-b")
	if stillOriginal.Vehicles[0].DueDate != "DUE 30 SEP" {
		t.Errorf("sub-b's vehicle was mutated by a renewal on sub-a: %+v", stillOriginal.Vehicles[0])
	}
}

func TestGetOrSeedConcurrentAccess(t *testing.T) {
	r := New()
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			sub := fmt.Sprintf("sub-%d", i%10)
			r.GetOrSeed(sub)
		}()
	}
	wg.Wait()
}

func TestRenewVehicleUnknownSub(t *testing.T) {
	r := New()
	_, ok := r.RenewVehicle("no-such-sub", "veh-cab4471")
	if ok {
		t.Fatal("expected RenewVehicle to report not-found for an unseen sub")
	}
}

func TestRenewVehicleUnknownVehicleID(t *testing.T) {
	r := New()
	r.GetOrSeed("sub-1")
	_, ok := r.RenewVehicle("sub-1", "no-such-vehicle")
	if ok {
		t.Fatal("expected RenewVehicle to report not-found for an unknown vehicle ID")
	}
}

func TestRenewVehicleHappyPath(t *testing.T) {
	r := New()
	r.GetOrSeed("sub-1")

	updated, ok := r.RenewVehicle("sub-1", "veh-cab4471")
	if !ok {
		t.Fatal("expected RenewVehicle to find veh-cab4471")
	}
	if updated.DueDate != "RENEWED · valid to 30 Sep 2027" {
		t.Errorf("DueDate = %q", updated.DueDate)
	}

	// The renewal must be persisted, not just returned.
	persisted := r.GetOrSeed("sub-1")
	if persisted.Vehicles[0].DueDate != "RENEWED · valid to 30 Sep 2027" {
		t.Errorf("persisted DueDate = %q", persisted.Vehicles[0].DueDate)
	}
}
