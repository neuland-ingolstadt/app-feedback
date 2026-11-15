package httpapi

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/neuland-ingolstadt/app-feedback/internal/config"
	"github.com/neuland-ingolstadt/app-feedback/internal/feedback"
	"github.com/neuland-ingolstadt/app-feedback/internal/store"
)

func TestPublicSubmitMetricsAndAdminExport(t *testing.T) {
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
		AdminToken:       "test-admin-token",
	}
	svc := feedback.NewService(cfg, db)
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	srv := httptest.NewServer(New(cfg, svc, log, "test", "abc").Handler())
	t.Cleanup(srv.Close)

	submit, err := http.NewRequest(http.MethodPut, srv.URL+"/v1/feedback", strings.NewReader(
		`{"id":"submission01","campaign":"campaign-a","rating":4,"platform":"ios","comment":"secret-comment-text","categories":["timetable"],"featureInterest":{"digitalId":3}}`,
	))
	if err != nil {
		t.Fatal(err)
	}
	submit.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(submit)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("submit status %d", res.StatusCode)
	}

	metricsRes, err := http.Get(srv.URL + "/metrics")
	if err != nil {
		t.Fatal(err)
	}
	defer metricsRes.Body.Close()
	body, err := io.ReadAll(metricsRes.Body)
	if err != nil {
		t.Fatal(err)
	}
	if metricsRes.StatusCode != http.StatusOK {
		t.Fatalf("metrics status %d\n%s", metricsRes.StatusCode, body)
	}
	text := string(body)
	if !strings.Contains(text, `app_feedback_submissions{campaign="campaign-a"} 1`) {
		t.Fatalf("missing submissions gauge:\n%s", text)
	}
	if strings.Contains(text, "secret-comment-text") {
		t.Fatal("metrics included comment text")
	}

	denied, err := http.Get(srv.URL + "/v1/admin/export")
	if err != nil {
		t.Fatal(err)
	}
	denied.Body.Close()
	if denied.StatusCode != http.StatusUnauthorized {
		t.Fatalf("export without token: %d", denied.StatusCode)
	}

	exportReq, err := http.NewRequest(http.MethodGet, srv.URL+"/v1/admin/export", nil)
	if err != nil {
		t.Fatal(err)
	}
	exportReq.Header.Set("Authorization", "Bearer test-admin-token")
	exportRes, err := http.DefaultClient.Do(exportReq)
	if err != nil {
		t.Fatal(err)
	}
	defer exportRes.Body.Close()
	exportBody, err := io.ReadAll(exportRes.Body)
	if err != nil {
		t.Fatal(err)
	}
	if exportRes.StatusCode != http.StatusOK || !strings.Contains(string(exportBody), "secret-comment-text") {
		t.Fatalf("export status %d body %s", exportRes.StatusCode, exportBody)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := svc.Ping(ctx); err != nil {
		t.Fatal(err)
	}
}
