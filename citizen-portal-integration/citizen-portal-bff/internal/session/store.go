// Package session implements the BFF's server-side state: short-lived login
// transactions (state/nonce/PKCE, held only between the redirect to IS and
// the callback) and longer-lived authenticated sessions. Nothing in this
// package is ever serialized to the browser — only an opaque, random
// identifier is, as a cookie value.
package session

import (
	"container/list"
	"sync"
	"time"
)

const sweepInterval = 30 * time.Second

type entry[T any] struct {
	key   string
	value T
	exp   time.Time
}

// Store is a generic, size-bounded, TTL-expiring key/value store, safe for
// concurrent use. On reaching capacity it evicts the oldest-inserted entry
// first — the same bounded-cache discipline as esignet-bridge/server.js's
// TtlMap, so the two codebases fail the same way under load rather than one
// growing unbounded.
type Store[T any] struct {
	mu        sync.Mutex
	byKey     map[string]*list.Element // -> *entry[T]
	order     *list.List               // insertion order, oldest at Front
	maxSize   int
	stopSweep chan struct{}
}

// NewStore creates a Store bounded to maxSize live entries and starts a
// background sweeper that reclaims expired entries every sweepInterval, so
// memory is bounded even for keys nobody ever calls Get on again. Call
// Close when the store is no longer needed to stop that goroutine.
func NewStore[T any](maxSize int) *Store[T] {
	s := &Store[T]{
		byKey:     make(map[string]*list.Element),
		order:     list.New(),
		maxSize:   maxSize,
		stopSweep: make(chan struct{}),
	}
	go s.sweepLoop()
	return s
}

// Put stores value under key with the given time-to-live, evicting the
// oldest entry first if the store is at capacity. A Put of an existing key
// replaces its value and moves it to the back of the eviction order.
func (s *Store[T]) Put(key string, value T, ttl time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if el, ok := s.byKey[key]; ok {
		s.order.Remove(el)
		delete(s.byKey, key)
	}

	for s.order.Len() >= s.maxSize {
		oldest := s.order.Front()
		if oldest == nil {
			break
		}
		s.order.Remove(oldest)
		delete(s.byKey, oldest.Value.(*entry[T]).key)
	}

	el := s.order.PushBack(&entry[T]{key: key, value: value, exp: time.Now().Add(ttl)})
	s.byKey[key] = el
}

// Get returns the value for key if present and not expired. An expired
// entry is removed lazily on lookup, in addition to the background sweep.
func (s *Store[T]) Get(key string) (T, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	el, ok := s.byKey[key]
	if !ok {
		var zero T
		return zero, false
	}
	e := el.Value.(*entry[T])
	if time.Now().After(e.exp) {
		s.order.Remove(el)
		delete(s.byKey, key)
		var zero T
		return zero, false
	}
	return e.value, true
}

// Delete removes key unconditionally, whether or not it exists.
func (s *Store[T]) Delete(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.deleteLocked(key)
}

func (s *Store[T]) deleteLocked(key string) {
	if el, ok := s.byKey[key]; ok {
		s.order.Remove(el)
		delete(s.byKey, key)
	}
}

// DeleteWhere removes every non-expired entry for which match returns true,
// and returns the count removed. Used for back-channel logout: destroy
// every session sharing the IdP's sid, across every app's own store lookup
// key.
func (s *Store[T]) DeleteWhere(match func(T) bool) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	var toRemove []string
	for el := s.order.Front(); el != nil; el = el.Next() {
		e := el.Value.(*entry[T])
		if now.After(e.exp) {
			continue
		}
		if match(e.value) {
			toRemove = append(toRemove, e.key)
		}
	}
	for _, k := range toRemove {
		s.deleteLocked(k)
	}
	return len(toRemove)
}

// Len returns the number of live (non-expired) entries.
func (s *Store[T]) Len() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.order.Len()
}

// Close stops the background sweeper. Safe to call once; the store must not
// be used afterward.
func (s *Store[T]) Close() {
	close(s.stopSweep)
}

func (s *Store[T]) sweepLoop() {
	ticker := time.NewTicker(sweepInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			s.sweep()
		case <-s.stopSweep:
			return
		}
	}
}

func (s *Store[T]) sweep() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	var expired []string
	for el := s.order.Front(); el != nil; el = el.Next() {
		e := el.Value.(*entry[T])
		if now.After(e.exp) {
			expired = append(expired, e.key)
		}
	}
	for _, k := range expired {
		s.deleteLocked(k)
	}
}
