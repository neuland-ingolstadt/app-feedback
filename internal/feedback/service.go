package feedback

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/neuland-ingolstadt/app-feedback/internal/abuse"
	"github.com/neuland-ingolstadt/app-feedback/internal/config"
)

type Store interface {
	Get(ctx context.Context, id string) (*Record, error)
	Upsert(ctx context.Context, r Record) error
	List(ctx context.Context, includeShadow bool) ([]Record, error)
	SpendChallenge(ctx context.Context, id string, expiresAt time.Time) (bool, error)
	Ping(ctx context.Context) error
}

type Service struct {
	cfg     config.Config
	store   Store
	pow     *abuse.Verifier
	ipLimit *abuse.TokenBucket
	idLimit *abuse.TokenBucket
	replay  *abuse.ReplayCache
	trust   *abuse.TrustEngine
}

func NewService(cfg config.Config, store Store) *Service {
	return &Service{
		cfg:     cfg,
		store:   store,
		pow:     abuse.NewVerifier(cfg.ChallengeSecret, cfg.PoWDifficulty, cfg.ChallengeTTL),
		ipLimit: abuse.NewTokenBucket(cfg.IPRateLimit, cfg.IPRateBurst),
		idLimit: abuse.NewTokenBucket(cfg.IDRateLimit, cfg.IDRateBurst),
		replay:  abuse.NewReplayCache(),
		trust:   abuse.NewTrustEngine(),
	}
}

func (s *Service) IssueChallenge(ip string, now time.Time) (ChallengeResponse, error) {
	ch, err := s.pow.Issue(abuse.IPClass(ip), now)
	if err != nil {
		return ChallengeResponse{}, err
	}
	return ChallengeResponse{
		Challenge:  ch.Challenge,
		Nonce:      ch.Nonce,
		Difficulty: ch.Difficulty,
		ExpiresAt:  ch.ExpiresAt,
		Algorithm:  ch.Algorithm,
	}, nil
}

type SubmitResult struct {
	Accepted     bool
	ShadowBanned bool
	TrustScore   float64
	Reasons      []string
}

func (s *Service) Submit(ctx context.Context, sub Submission, ip string, now time.Time) (SubmitResult, error) {
	if !s.ipLimit.Allow(ip, now) {
		return SubmitResult{}, ErrRateLimited
	}
	if !s.idLimit.Allow(sub.ID, now) {
		return SubmitResult{}, ErrRateLimited
	}

	honeypot := strings.TrimSpace(sub.Website) != ""

	if err := ValidateSubmission(sub, s.cfg.MaxCommentBytes, s.cfg.AllowedCampaigns); err != nil && !honeypot {
		return SubmitResult{}, fmt.Errorf("%w: %s", ErrInvalid, err.Error())
	}

	proof, err := s.pow.Verify(sub.Challenge, sub.Counter, abuse.IPClass(ip), now)
	if err != nil && !honeypot {
		return SubmitResult{}, fmt.Errorf("%w: %s", ErrInvalid, err.Error())
	}

	if !honeypot {
		// In-memory fast path + durable sqlite so restarts don't reset replays.
		if !s.replay.Spend(proof.ID, proof.ExpiresAt, now) {
			return SubmitResult{}, fmt.Errorf("%w: challenge replay", ErrInvalid)
		}
		ok, err := s.store.SpendChallenge(ctx, proof.ID, proof.ExpiresAt)
		if err != nil {
			return SubmitResult{}, err
		}
		if !ok {
			return SubmitResult{}, fmt.Errorf("%w: challenge replay", ErrInvalid)
		}
	}

	existing, err := s.store.Get(ctx, sub.ID)
	if err != nil {
		return SubmitResult{}, err
	}

	upsertCount := 1
	ratingFlip := false
	isCreate := existing == nil
	if existing != nil {
		upsertCount = existing.UpsertCount + 1
		ratingFlip = existing.Rating != sub.Rating
		if upsertCount > s.cfg.MaxUpsertsPerID {
			return SubmitResult{}, fmt.Errorf("%w: upsert budget exhausted", ErrInvalid)
		}
	}

	// Rough lower bound: difficulty 18 ≈ tens of ms on modern CPUs; scale linearly.
	expectedMin := time.Duration(float64(proof.Difficulty)/18.0*30) * time.Millisecond
	solveDur := time.Duration(0)
	if !proof.ExpiresAt.IsZero() {
		// We don't know issue time exactly; approximate from TTL remaining.
		issuedAt := proof.ExpiresAt.Add(-s.cfg.ChallengeTTL)
		if now.After(issuedAt) {
			solveDur = now.Sub(issuedAt)
		}
	}

	decision := s.trust.Evaluate(abuse.Signals{
		IsCreate:          isCreate,
		SolveDuration:     solveDur,
		ExpectedSolveMin:  expectedMin,
		Comment:           deref(sub.Comment),
		MissingFeatures:   deref(sub.MissingFeatures),
		IP:                ip,
		ID:                sub.ID,
		RatingFlip:        ratingFlip,
		UpsertCount:       upsertCount,
		MaxUpserts:        s.cfg.MaxUpsertsPerID,
		HoneypotTriggered: honeypot,
	}, s.cfg.ShadowBanThreshold, s.cfg.RejectThreshold, now)

	if decision.Reject {
		return SubmitResult{Accepted: false, TrustScore: decision.Score, Reasons: decision.Reasons}, ErrRejected
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
		ClientIP:        ip,
		TrustScore:      decision.Score,
		ShadowBanned:    decision.ShadowBan,
		UpsertCount:     upsertCount,
		UpdatedAt:       now.UTC(),
	}
	if existing != nil {
		rec.CreatedAt = existing.CreatedAt
		if existing.ShadowBanned {
			rec.ShadowBanned = true
		}
	} else {
		rec.CreatedAt = now.UTC()
	}

	if err := s.store.Upsert(ctx, rec); err != nil {
		return SubmitResult{}, err
	}

	return SubmitResult{
		Accepted:     true,
		ShadowBanned: rec.ShadowBanned,
		TrustScore:   decision.Score,
		Reasons:      decision.Reasons,
	}, nil
}

func (s *Service) Export(ctx context.Context, includeShadow bool) ([]Record, error) {
	return s.store.List(ctx, includeShadow)
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
