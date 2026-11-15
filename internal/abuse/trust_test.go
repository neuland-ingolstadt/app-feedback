package abuse

import (
	"testing"
	"time"
)

func TestHoneypotShadowBans(t *testing.T) {
	eng := NewTrustEngine()
	d := eng.Evaluate(Signals{HoneypotTriggered: true}, 0.65, 0.9, time.Now())
	if !d.ShadowBan || d.Score != 1 {
		t.Fatalf("expected honeypot shadowban, got %+v", d)
	}
}

func TestSpammyTextRaisesScore(t *testing.T) {
	eng := NewTrustEngine()
	d := eng.Evaluate(Signals{
		Comment: "BUY NOW http://a.test http://b.test CLICK HERE!!!!!",
	}, 0.65, 0.9, time.Now())
	if d.Score < 0.3 {
		t.Fatalf("expected elevated score, got %v (%v)", d.Score, d.Reasons)
	}
}
