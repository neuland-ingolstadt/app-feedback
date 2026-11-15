package feedback_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

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
		IPRateLimit:      100,
		IPRateBurst:      100,
		IDRateLimit:      100,
		IDRateBurst:      100,
		MaxUpsertsPerID:  8,
		MaxCommentBytes:  4000,
		AllowedCampaigns: []string{"campaign-a"},
	}
	svc := feedback.NewService(cfg, db)
	now := time.Now().UTC()
	ip := "198.51.100.9"

	err = svc.Submit(context.Background(), feedback.Submission{
		ID:       "submission01",
		Campaign: "campaign-a",
		Rating:   4,
		Platform: "ios",
	}, ip, now)
	if err != nil {
		t.Fatal(err)
	}

	err = svc.Submit(context.Background(), feedback.Submission{
		ID:       "submission01",
		Campaign: "campaign-a",
		Rating:   5,
		Platform: "ios",
	}, ip, now.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}

	rows, err := svc.Export(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].Rating != 5 || rows[0].UpsertCount != 2 {
		t.Fatalf("unexpected rows: %+v", rows)
	}
}

func TestUpsertBudget(t *testing.T) {
	dir := t.TempDir()
	db, err := store.OpenSQLite("file:" + filepath.Join(dir, "t.db") + "?_pragma=busy_timeout(5000)")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	cfg := config.Config{
		IPRateLimit:      100,
		IPRateBurst:      100,
		IDRateLimit:      100,
		IDRateBurst:      100,
		MaxUpsertsPerID:  2,
		MaxCommentBytes:  4000,
		AllowedCampaigns: nil,
	}
	svc := feedback.NewService(cfg, db)
	now := time.Now().UTC()

	sub := feedback.Submission{ID: "submission01", Campaign: "camp-a", Rating: 3}
	if err := svc.Submit(context.Background(), sub, "1.1.1.1", now); err != nil {
		t.Fatal(err)
	}
	if err := svc.Submit(context.Background(), sub, "1.1.1.1", now.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	if err := svc.Submit(context.Background(), sub, "1.1.1.1", now.Add(2*time.Second)); err == nil {
		t.Fatal("expected upsert budget error")
	}
}
