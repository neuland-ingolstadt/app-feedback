package feedback

import (
	"context"
	"fmt"
	"time"

	"github.com/neuland-ingolstadt/app-feedback/internal/abuse"
	"github.com/neuland-ingolstadt/app-feedback/internal/config"
)

type Store interface {
	Get(ctx context.Context, id string) (*Record, error)
	Upsert(ctx context.Context, r Record) error
	List(ctx context.Context) ([]Record, error)
	Ping(ctx context.Context) error
}

type Service struct {
	cfg     config.Config
	store   Store
	ipLimit *abuse.TokenBucket
	idLimit *abuse.TokenBucket
}

func NewService(cfg config.Config, store Store) *Service {
	return &Service{
		cfg:     cfg,
		store:   store,
		ipLimit: abuse.NewTokenBucket(cfg.IPRateLimit, cfg.IPRateBurst),
		idLimit: abuse.NewTokenBucket(cfg.IDRateLimit, cfg.IDRateBurst),
	}
}

func (s *Service) Submit(ctx context.Context, sub Submission, ip string, now time.Time) error {
	if !s.ipLimit.Allow(ip, now) {
		return ErrRateLimited
	}
	if !s.idLimit.Allow(sub.ID, now) {
		return ErrRateLimited
	}

	if err := ValidateSubmission(sub, s.cfg.MaxCommentBytes, s.cfg.AllowedCampaigns); err != nil {
		return fmt.Errorf("%w: %s", ErrInvalid, err.Error())
	}

	existing, err := s.store.Get(ctx, sub.ID)
	if err != nil {
		return err
	}

	upsertCount := 1
	if existing != nil {
		upsertCount = existing.UpsertCount + 1
		if upsertCount > s.cfg.MaxUpsertsPerID {
			return fmt.Errorf("%w: upsert budget exhausted", ErrInvalid)
		}
	}

	rec := Record{
		ID:              sub.ID,
		Campaign:        sub.Campaign,
		Rating:          sub.Rating,
		Categories:      sub.Categories,
		Comment:         deref(sub.Comment),
		FeatureInterest: sub.FeatureInterest,
		MissingFeatures: deref(sub.MissingFeatures),
		Platform:        sub.Platform,
		AppVersion:      sub.AppVersion,
		UpsertCount:     upsertCount,
		UpdatedAt:       now.UTC(),
	}
	if existing != nil {
		rec.CreatedAt = existing.CreatedAt
	} else {
		rec.CreatedAt = now.UTC()
	}

	return s.store.Upsert(ctx, rec)
}

func (s *Service) Export(ctx context.Context) ([]Record, error) {
	return s.store.List(ctx)
}

func (s *Service) Ping(ctx context.Context) error {
	return s.store.Ping(ctx)
}

func (s *Service) GC(now time.Time) {
	s.ipLimit.GC(now, time.Hour)
	s.idLimit.GC(now, time.Hour)
}

func deref(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}
