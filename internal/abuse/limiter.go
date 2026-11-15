package abuse

import (
	"sync"
	"time"
)

// TokenBucket is a tiny per-key rate limiter.
type TokenBucket struct {
	mu      sync.Mutex
	rate    float64
	burst   float64
	buckets map[string]*bucket
}

type bucket struct {
	tokens float64
	last   time.Time
}

func NewTokenBucket(rate float64, burst int) *TokenBucket {
	if rate <= 0 {
		rate = 1
	}
	if burst < 1 {
		burst = 1
	}
	return &TokenBucket{
		rate:    rate,
		burst:   float64(burst),
		buckets: make(map[string]*bucket),
	}
}

func (t *TokenBucket) Allow(key string, now time.Time) bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	b, ok := t.buckets[key]
	if !ok {
		t.buckets[key] = &bucket{tokens: t.burst - 1, last: now}
		return true
	}
	elapsed := now.Sub(b.last).Seconds()
	b.tokens = min(t.burst, b.tokens+elapsed*t.rate)
	b.last = now
	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

// GC drops idle buckets. Call occasionally from a background loop.
func (t *TokenBucket) GC(now time.Time, maxIdle time.Duration) {
	t.mu.Lock()
	defer t.mu.Unlock()
	for k, b := range t.buckets {
		if now.Sub(b.last) > maxIdle {
			delete(t.buckets, k)
		}
	}
}
