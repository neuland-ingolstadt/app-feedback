package abuse

import (
	"testing"
	"time"
)

func TestTokenBucket(t *testing.T) {
	b := NewTokenBucket(1, 2)
	now := time.Now()
	if !b.Allow("a", now) || !b.Allow("a", now) {
		t.Fatal("expected burst of 2")
	}
	if b.Allow("a", now) {
		t.Fatal("expected deny after burst")
	}
	if !b.Allow("a", now.Add(time.Second)) {
		t.Fatal("expected refill")
	}
}
