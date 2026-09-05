package entra

import (
	"crypto/rand"
	"encoding/base64"
	"sync"
	"time"
)

// StateStore guards the OIDC authorization-code flow against CSRF: the state
// issued by /entra/login must be presented back by /entra/callback, and it is
// single-use. Entries are kept in memory with a TTL and evicted lazily.
type StateStore struct {
	mu    sync.Mutex
	ttl   time.Duration
	store map[string]time.Time
}

func NewStateStore(ttl time.Duration) *StateStore {
	return &StateStore{ttl: ttl, store: make(map[string]time.Time)}
}

// New returns a fresh random state and records it with the store TTL.
func (s *StateStore) New() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	state := base64.RawURLEncoding.EncodeToString(b)

	s.mu.Lock()
	defer s.mu.Unlock()
	s.evictLocked()
	s.store[state] = time.Now().Add(s.ttl)
	return state
}

// Verify consumes the state if it is known and not expired. Because a state is
// single-use, a second replay of the same callback fails.
func (s *StateStore) Verify(state string) bool {
	if state == "" {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	exp, ok := s.store[state]
	delete(s.store, state) // single-use regardless of outcome
	if !ok || time.Now().After(exp) {
		return false
	}
	return true
}

func (s *StateStore) evictLocked() {
	now := time.Now()
	for k, exp := range s.store {
		if now.After(exp) {
			delete(s.store, k)
		}
	}
}
