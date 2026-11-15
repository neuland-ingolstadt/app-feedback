package feedback_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/neuland-ingolstadt/app-feedback/internal/abuse"
	"github.com/neuland-ingolstadt/app-feedback/internal/config"
	"github.com/neuland-ingolstadt/app-feedback/internal/feedback"
	"github.com/neuland-ingolstadt/app-feedback/internal/store"
)

func TestSubmitRoundTrip(t *testing.T) {
	dir := t.TempDir()
	db, err := store.OpenSQLite("file:" + filepath.Join(dir, "t.db") + "?_pragma=busy_timeout(5000)")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	cfg := config.Config{
		ChallengeSecret:    "test-secret",
		PoWDifficulty:      10,
		ChallengeTTL:       time.Minute,
		IPRateLimit:        100,
		IPRateBurst:        100,
		IDRateLimit:        100,
		IDRateBurst:        100,
		MaxUpsertsPerID:    8,
		MaxCommentBytes:    4000,
		ShadowBanThreshold: 0.65,
		RejectThreshold:    0.9,
		AllowedCampaigns:   []string{"campaign-a"},
	}
	svc := feedback.NewService(cfg, db)
	now := time.Now().UTC()
	ip := "198.51.100.9"

	ch, err := svc.IssueChallenge(ip, now)
	if err != nil {
		t.Fatal(err)
	}
	counter, err := abuse.Solve(ch.Nonce, ch.Difficulty)
	if err != nil {
		t.Fatal(err)
	}

	res, err := svc.Submit(context.Background(), feedback.Submission{
		ID:        "submission01",
		Campaign:  "campaign-a",
		Rating:    4,
		Challenge: ch.Challenge,
		Counter:   counter,
		Platform:  "ios",
	}, ip, now.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if !res.Accepted {
		t.Fatalf("expected accept, got %+v", res)
	}

	_, err = svc.Submit(context.Background(), feedback.Submission{
		ID:        "submission01",
		Campaign:  "campaign-a",
		Rating:    5,
		Challenge: ch.Challenge,
		Counter:   counter,
	}, ip, now.Add(3*time.Second))
	if err == nil {
		t.Fatal("expected replay error")
	}

	ch2, err := svc.IssueChallenge(ip, now.Add(2*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	counter2, err := abuse.Solve(ch2.Nonce, ch2.Difficulty)
	if err != nil {
		t.Fatal(err)
	}
	_, err = svc.Submit(context.Background(), feedback.Submission{
		ID:        "submission01",
		Campaign:  "campaign-a",
		Rating:    5,
		Challenge: ch2.Challenge,
		Counter:   counter2,
		Platform:  "ios",
	}, ip, now.Add(4*time.Second))
	if err != nil {
		t.Fatal(err)
	}
}
