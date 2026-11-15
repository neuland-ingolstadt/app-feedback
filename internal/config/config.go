package config

import (
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	ListenAddr  string
	DatabaseURL string

	// Optional shared write key. When set, clients must send it as
	// X-Feedback-Key.
	WriteKey string

	// Bearer token for /v1/admin/*. Empty disables admin routes.
	AdminToken string

	// Comma-separated allowlisted campaign ids. Empty allows any non-empty id.
	AllowedCampaigns []string

	IPRateLimit       float64 // tokens per second
	IPRateBurst       int
	IDRateLimit       float64
	IDRateBurst       int
	MaxUpsertsPerID   int
	MaxCommentBytes   int
	TrustedProxyCount int
}

func Load() (Config, error) {
	_ = godotenv.Load()

	return Config{
		ListenAddr:        envOr("LISTEN_ADDR", ":8080"),
		DatabaseURL:       envOr("DATABASE_URL", "file:feedback.db?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)"),
		WriteKey:          os.Getenv("WRITE_KEY"),
		AdminToken:        os.Getenv("ADMIN_TOKEN"),
		AllowedCampaigns:  splitCSV(os.Getenv("ALLOWED_CAMPAIGNS")),
		IPRateLimit:       envFloat("IP_RATE_LIMIT", 0.2), // ~12/min
		IPRateBurst:       envInt("IP_RATE_BURST", 5),
		IDRateLimit:       envFloat("ID_RATE_LIMIT", 0.1),
		IDRateBurst:       envInt("ID_RATE_BURST", 3),
		MaxUpsertsPerID:   envInt("MAX_UPSERTS_PER_ID", 8),
		MaxCommentBytes:   envInt("MAX_COMMENT_BYTES", 4000),
		TrustedProxyCount: envInt("TRUSTED_PROXY_COUNT", 1),
	}, nil
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

func envFloat(key string, fallback float64) float64 {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return fallback
	}
	return n
}

func splitCSV(v string) []string {
	if v == "" {
		return nil
	}
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
