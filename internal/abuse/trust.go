package abuse

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"sync"
	"time"
	"unicode"
)

// TrustEngine accumulates cheap heuristics into a 0..1 risk score.
// Higher = more suspicious. Deliberately a bit overengineered for a survey API.
type TrustEngine struct {
	mu sync.Mutex

	commentHits map[string]int
	ipCreates   map[string][]time.Time
	idMutations map[string]int
}

func NewTrustEngine() *TrustEngine {
	return &TrustEngine{
		commentHits: make(map[string]int),
		ipCreates:   make(map[string][]time.Time),
		idMutations: make(map[string]int),
	}
}

type Signals struct {
	IsCreate          bool
	SolveDuration     time.Duration
	ExpectedSolveMin  time.Duration
	Comment           string
	MissingFeatures   string
	IP                string
	ID                string
	RatingFlip        bool
	UpsertCount       int
	MaxUpserts        int
	HoneypotTriggered bool
}

type Decision struct {
	Score     float64
	ShadowBan bool
	Reject    bool
	Reasons   []string
}

func (t *TrustEngine) Evaluate(s Signals, shadowThreshold, rejectThreshold float64, now time.Time) Decision {
	score := 0.0
	var reasons []string

	if s.HoneypotTriggered {
		return Decision{Score: 1, ShadowBan: true, Reasons: []string{"honeypot"}}
	}

	if s.SolveDuration > 0 && s.ExpectedSolveMin > 0 && s.SolveDuration < s.ExpectedSolveMin/4 {
		score += 0.35
		reasons = append(reasons, "pow_too_fast")
	}

	if looksLikeSpam(s.Comment) || looksLikeSpam(s.MissingFeatures) {
		score += 0.3
		reasons = append(reasons, "spammy_text")
	}

	if h := textHash(s.Comment); h != "" {
		t.mu.Lock()
		t.commentHits[h]++
		hits := t.commentHits[h]
		t.mu.Unlock()
		if hits >= 5 {
			score += 0.25
			reasons = append(reasons, "duplicate_comment_cluster")
		}
	}

	if s.IsCreate {
		t.mu.Lock()
		recent := filterRecent(t.ipCreates[s.IP], now, 10*time.Minute)
		recent = append(recent, now)
		t.ipCreates[s.IP] = recent
		n := len(recent)
		t.mu.Unlock()
		if n >= 8 {
			score += 0.35
			reasons = append(reasons, "ip_create_velocity")
		} else if n >= 4 {
			score += 0.15
			reasons = append(reasons, "ip_create_burst")
		}
	} else {
		t.mu.Lock()
		t.idMutations[s.ID]++
		muts := t.idMutations[s.ID]
		t.mu.Unlock()
		if muts >= 5 {
			score += 0.2
			reasons = append(reasons, "mutation_churn")
		}
		if s.RatingFlip {
			score += 0.1
			reasons = append(reasons, "rating_flip")
		}
	}

	if s.MaxUpserts > 0 && s.UpsertCount >= s.MaxUpserts {
		score += 0.4
		reasons = append(reasons, "upsert_budget_exhausted")
	}

	if score > 1 {
		score = 1
	}

	return Decision{
		Score:     score,
		ShadowBan: score >= shadowThreshold && score < rejectThreshold,
		Reject:    score >= rejectThreshold,
		Reasons:   reasons,
	}
}

func textHash(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	if len(s) < 12 {
		return ""
	}
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:8])
}

func looksLikeSpam(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}
	if len(s) > 20 && strings.Count(s, "http://")+strings.Count(s, "https://") >= 2 {
		return true
	}
	letters := 0
	upper := 0
	for _, r := range s {
		if unicode.IsLetter(r) {
			letters++
			if unicode.IsUpper(r) {
				upper++
			}
		}
	}
	if letters > 30 && float64(upper)/float64(letters) > 0.7 {
		return true
	}
	// repeated character runs
	if len(s) >= 16 {
		same := 1
		for i := 1; i < len(s); i++ {
			if s[i] == s[i-1] {
				same++
				if same >= 12 {
					return true
				}
			} else {
				same = 1
			}
		}
	}
	return false
}

func filterRecent(in []time.Time, now time.Time, window time.Duration) []time.Time {
	out := make([]time.Time, 0, len(in))
	for _, ts := range in {
		if now.Sub(ts) <= window {
			out = append(out, ts)
		}
	}
	return out
}
