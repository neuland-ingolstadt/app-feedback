package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/neuland-ingolstadt/app-feedback/internal/feedback"

	_ "modernc.org/sqlite"
)

type SQLite struct {
	db *sql.DB
}

func OpenSQLite(dsn string) (*SQLite, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	s := &SQLite{db: db}
	if err := s.migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

func (s *SQLite) Close() error {
	return s.db.Close()
}

func (s *SQLite) migrate() error {
	_, err := s.db.Exec(`
CREATE TABLE IF NOT EXISTS feedback (
  id TEXT PRIMARY KEY,
  campaign TEXT NOT NULL,
  rating INTEGER NOT NULL,
  categories_json TEXT NOT NULL DEFAULT '[]',
  comment TEXT NOT NULL DEFAULT '',
  feature_interest_json TEXT NOT NULL DEFAULT '{}',
  missing_features TEXT NOT NULL DEFAULT '',
  platform TEXT NOT NULL DEFAULT '',
  app_version TEXT NOT NULL DEFAULT '',
  client_ip TEXT NOT NULL DEFAULT '',
  upsert_count INTEGER NOT NULL DEFAULT 1,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_feedback_campaign ON feedback(campaign);
`)
	return err
}

func (s *SQLite) Get(ctx context.Context, id string) (*feedback.Record, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT id, campaign, rating, categories_json, comment, feature_interest_json,
       missing_features, platform, app_version, client_ip, upsert_count,
       created_at, updated_at
FROM feedback WHERE id = ?`, id)

	var r feedback.Record
	var cats, feats string
	var created, updated string
	err := row.Scan(
		&r.ID, &r.Campaign, &r.Rating, &cats, &r.Comment, &feats,
		&r.MissingFeatures, &r.Platform, &r.AppVersion, &r.ClientIP, &r.UpsertCount,
		&created, &updated,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	_ = json.Unmarshal([]byte(cats), &r.Categories)
	_ = json.Unmarshal([]byte(feats), &r.FeatureInterest)
	r.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
	r.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updated)
	return &r, nil
}

func (s *SQLite) Upsert(ctx context.Context, r feedback.Record) error {
	cats, err := json.Marshal(r.Categories)
	if err != nil {
		return err
	}
	if r.Categories == nil {
		cats = []byte("[]")
	}
	feats, err := json.Marshal(r.FeatureInterest)
	if err != nil {
		return err
	}
	if r.FeatureInterest == nil {
		feats = []byte("{}")
	}
	_, err = s.db.ExecContext(ctx, `
INSERT INTO feedback (
  id, campaign, rating, categories_json, comment, feature_interest_json,
  missing_features, platform, app_version, client_ip, upsert_count,
  created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
  campaign=excluded.campaign,
  rating=excluded.rating,
  categories_json=excluded.categories_json,
  comment=excluded.comment,
  feature_interest_json=excluded.feature_interest_json,
  missing_features=excluded.missing_features,
  platform=excluded.platform,
  app_version=excluded.app_version,
  client_ip=excluded.client_ip,
  upsert_count=excluded.upsert_count,
  updated_at=excluded.updated_at
`, r.ID, r.Campaign, r.Rating, string(cats), r.Comment, string(feats),
		r.MissingFeatures, r.Platform, r.AppVersion, r.ClientIP, r.UpsertCount,
		r.CreatedAt.UTC().Format(time.RFC3339Nano),
		r.UpdatedAt.UTC().Format(time.RFC3339Nano))
	return err
}

func (s *SQLite) List(ctx context.Context) ([]feedback.Record, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT id, campaign, rating, categories_json, comment, feature_interest_json,
       missing_features, platform, app_version, client_ip, upsert_count,
       created_at, updated_at
FROM feedback
ORDER BY created_at ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []feedback.Record
	for rows.Next() {
		var r feedback.Record
		var cats, feats string
		var created, updated string
		if err := rows.Scan(
			&r.ID, &r.Campaign, &r.Rating, &cats, &r.Comment, &feats,
			&r.MissingFeatures, &r.Platform, &r.AppVersion, &r.ClientIP, &r.UpsertCount,
			&created, &updated,
		); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(cats), &r.Categories)
		_ = json.Unmarshal([]byte(feats), &r.FeatureInterest)
		r.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
		r.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updated)
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *SQLite) Ping(ctx context.Context) error {
	if err := s.db.PingContext(ctx); err != nil {
		return err
	}
	var one int
	if err := s.db.QueryRowContext(ctx, `SELECT 1`).Scan(&one); err != nil {
		return fmt.Errorf("sqlite ping query: %w", err)
	}
	return nil
}
