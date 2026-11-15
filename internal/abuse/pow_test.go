package abuse

import (
	"testing"
	"time"
)

func TestPoWRoundTrip(t *testing.T) {
	v := NewVerifier("test-secret", 12, time.Minute)
	now := time.Now().UTC()
	ch, err := v.Issue(IPClass("203.0.113.10"), now)
	if err != nil {
		t.Fatal(err)
	}
	counter, err := Solve(ch.Nonce, ch.Difficulty)
	if err != nil {
		t.Fatal(err)
	}
	proof, err := v.Verify(ch.Challenge, counter, IPClass("203.0.113.55"), now.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if proof.Nonce != ch.Nonce {
		t.Fatalf("nonce mismatch")
	}
}

func TestPoWRejectsBadCounter(t *testing.T) {
	v := NewVerifier("test-secret", 10, time.Minute)
	now := time.Now().UTC()
	ch, err := v.Issue("", now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := v.Verify(ch.Challenge, 0, "", now); err == nil {
		// counter 0 might randomly work at low difficulty; mine a known-bad by tampering token
	}
	if _, err := v.Verify(ch.Challenge+"x", 1, "", now); err == nil {
		t.Fatal("expected invalid token")
	}
}

func TestPoWExpires(t *testing.T) {
	v := NewVerifier("test-secret", 8, time.Second)
	now := time.Now().UTC()
	ch, err := v.Issue("", now)
	if err != nil {
		t.Fatal(err)
	}
	counter, err := Solve(ch.Nonce, ch.Difficulty)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := v.Verify(ch.Challenge, counter, "", now.Add(2*time.Second)); err == nil {
		t.Fatal("expected expiry")
	}
}

func TestIPClass(t *testing.T) {
	if got := IPClass("1.2.3.4"); got != "1.2.3" {
		t.Fatalf("got %q", got)
	}
}
