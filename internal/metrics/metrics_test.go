package metrics

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/neuland-ingolstadt/app-feedback/internal/feedback"
)

func TestHandlerExportsGaugesWithoutCommentText(t *testing.T) {
	stats := feedback.Aggregate(sampleRecords(), []string{"campaign-a"})
	srv := httptest.NewServer(Handler(func(context.Context) (feedback.Stats, error) {
		return stats, nil
	}))
	t.Cleanup(srv.Close)

	res, err := http.Get(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status %d", res.StatusCode)
	}
	body := readAll(t, res)
	for _, want := range []string{
		`app_feedback_submissions{campaign="campaign-a"} 1`,
		`app_feedback_rating_sum{campaign="campaign-a"} 4`,
		`app_feedback_rating_responses{campaign="campaign-a",rating="4"} 1`,
		`app_feedback_platform_responses{campaign="campaign-a",platform="ios"} 1`,
		`app_feedback_category_responses{campaign="campaign-a",category="timetable"} 1`,
		`app_feedback_feature_interest_responses{campaign="campaign-a",feature="digitalId",level="3"} 1`,
		`app_feedback_feature_interest_score_sum{campaign="campaign-a",feature="digitalId"} 3`,
		`app_feedback_comments_nonempty{campaign="campaign-a"} 1`,
		`app_feedback_missing_features_nonempty{campaign="campaign-a"} 0`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("missing %s\nbody:\n%s", want, body)
		}
	}
	if strings.Contains(body, "secret-comment-text") {
		t.Fatal("comment text leaked into metrics")
	}
}

func TestHandlerFailsScrapeWhenStatsFail(t *testing.T) {
	srv := httptest.NewServer(Handler(func(context.Context) (feedback.Stats, error) {
		return feedback.Stats{}, errors.New("db down")
	}))
	t.Cleanup(srv.Close)

	res, err := http.Get(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode == http.StatusOK {
		t.Fatal("expected scrape error")
	}
}

func sampleRecords() []feedback.Record {
	return []feedback.Record{{
		Campaign:        "campaign-a",
		Rating:          4,
		Categories:      []feedback.Category{"timetable"},
		Comment:         "secret-comment-text",
		FeatureInterest: feedback.FeatureInterestAnswers{"digitalId": 3},
		Platform:        "ios",
	}}
}

func readAll(t *testing.T, res *http.Response) string {
	t.Helper()
	buf, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	return string(buf)
}
