package session

import (
	"testing"
	"time"
)

func TestStorePutAndGet(t *testing.T) {
	s := NewStore[string](100)
	defer s.Close()

	s.Put("k1", "v1", time.Minute)
	got, ok := s.Get("k1")
	if !ok || got != "v1" {
		t.Fatalf("Get(k1) = %q, %v; want v1, true", got, ok)
	}
}

func TestStoreGetMissing(t *testing.T) {
	s := NewStore[string](100)
	defer s.Close()

	if _, ok := s.Get("missing"); ok {
		t.Fatal("Get(missing) should return ok=false")
	}
}

func TestStoreExpiry(t *testing.T) {
	s := NewStore[string](100)
	defer s.Close()

	s.Put("k1", "v1", 10*time.Millisecond)
	time.Sleep(30 * time.Millisecond)

	if _, ok := s.Get("k1"); ok {
		t.Fatal("Get should not return an expired entry")
	}
}

func TestStoreDelete(t *testing.T) {
	s := NewStore[string](100)
	defer s.Close()

	s.Put("k1", "v1", time.Minute)
	s.Delete("k1")
	if _, ok := s.Get("k1"); ok {
		t.Fatal("Get after Delete should return ok=false")
	}
}

func TestStoreEvictsOldestOnCapacity(t *testing.T) {
	s := NewStore[string](3)
	defer s.Close()

	s.Put("k1", "v1", time.Minute)
	s.Put("k2", "v2", time.Minute)
	s.Put("k3", "v3", time.Minute)
	s.Put("k4", "v4", time.Minute) // pushes out the oldest entry (k1)

	if _, ok := s.Get("k1"); ok {
		t.Fatal("expected k1 to have been evicted once capacity was exceeded")
	}
	if got, ok := s.Get("k4"); !ok || got != "v4" {
		t.Fatalf("expected k4 present, got %q, %v", got, ok)
	}
}

func TestStoreLenReflectsLiveEntries(t *testing.T) {
	s := NewStore[int](100)
	defer s.Close()

	s.Put("a", 1, time.Minute)
	s.Put("b", 2, time.Minute)
	if got := s.Len(); got != 2 {
		t.Fatalf("Len() = %d, want 2", got)
	}
	s.Delete("a")
	if got := s.Len(); got != 1 {
		t.Fatalf("Len() after Delete = %d, want 1", got)
	}
}

func TestStoreDeleteWhereMatchingDestroysMultipleEntries(t *testing.T) {
	type entry struct {
		Sid string
	}
	s := NewStore[entry](100)
	defer s.Close()

	s.Put("session-a", entry{Sid: "shared-sid"}, time.Minute)
	s.Put("session-b", entry{Sid: "shared-sid"}, time.Minute)
	s.Put("session-c", entry{Sid: "other-sid"}, time.Minute)

	n := s.DeleteWhere(func(v entry) bool { return v.Sid == "shared-sid" })
	if n != 2 {
		t.Fatalf("DeleteWhere destroyed %d entries, want 2", n)
	}
	if _, ok := s.Get("session-a"); ok {
		t.Error("session-a should have been destroyed")
	}
	if _, ok := s.Get("session-b"); ok {
		t.Error("session-b should have been destroyed")
	}
	if _, ok := s.Get("session-c"); !ok {
		t.Error("session-c should survive — different sid")
	}
}

func TestStoreConcurrentAccess(t *testing.T) {
	s := NewStore[int](1000)
	defer s.Close()

	done := make(chan struct{})
	for i := 0; i < 50; i++ {
		go func(i int) {
			defer func() { done <- struct{}{} }()
			key := string(rune('a' + i%26))
			s.Put(key, i, time.Minute)
			s.Get(key)
			s.Len()
		}(i)
	}
	for i := 0; i < 50; i++ {
		<-done
	}
}
