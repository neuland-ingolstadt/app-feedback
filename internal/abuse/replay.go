package abuse

import (
	"sync"
	"time"
)

// ReplayCache remembers spent challenge ids until they expire.
type ReplayCache struct {
	mu   sync.Mutex
	seen map[string]time.Time
}

func NewReplayCache() *ReplayCache {
	return &ReplayCache{seen: make(map[string]time.Time)}
}

// Spend returns false if the challenge was already used.
func (r *ReplayCache) Spend(id string, expiresAt, now time.Time) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.gcLocked(now)
	if _, ok := r.seen[id]; ok {
		return false
	}
	r.seen[id] = expiresAt
	return true
}

func (r *ReplayCache) gcLocked(now time.Time) {
	for id, exp := range r.seen {
		if now.After(exp) {
			delete(r.seen, id)
		}
	}
}
