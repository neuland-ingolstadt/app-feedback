package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/neuland-ingolstadt/app-feedback/internal/config"
	"github.com/neuland-ingolstadt/app-feedback/internal/feedback"
	"github.com/neuland-ingolstadt/app-feedback/internal/metrics"
)

type Server struct {
	cfg     config.Config
	svc     *feedback.Service
	log     *slog.Logger
	version string
	commit  string
	mux     *http.ServeMux
}

func New(cfg config.Config, svc *feedback.Service, log *slog.Logger, version, commit string) *Server {
	s := &Server{cfg: cfg, svc: svc, log: log, version: version, commit: commit, mux: http.NewServeMux()}
	s.routes()
	return s
}

func (s *Server) Handler() http.Handler {
	return s.withMiddleware(s.mux)
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /healthz", s.handleHealth)
	s.mux.HandleFunc("PUT /v1/feedback", s.handleSubmit)
	s.mux.HandleFunc("GET /v1/admin/export", s.handleExport)
	// Cluster-internal scrape only. The public ingress must not route /metrics.
	s.mux.Handle("GET /metrics", metrics.Handler(s.svc.Stats))
}

func (s *Server) withMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := &statusWriter{ResponseWriter: w, status: 200}
		defer func() {
			s.log.Info("request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", ww.status,
				"duration_ms", time.Since(start).Milliseconds(),
			)
		}()

		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		if r.Method == http.MethodOptions {
			s.writeCORS(w, r)
			w.WriteHeader(http.StatusNoContent)
			return
		}
		s.writeCORS(w, r)
		next.ServeHTTP(ww, r)
	})
}

func (s *Server) writeCORS(w http.ResponseWriter, r *http.Request) {
	origin := r.Header.Get("Origin")
	if origin != "" {
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Vary", "Origin")
		w.Header().Set("Access-Control-Allow-Methods", "GET, PUT, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Feedback-Key, Authorization")
		w.Header().Set("Access-Control-Max-Age", "600")
	}
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if err := s.svc.Ping(ctx); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"ok": false, "error": "db"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":      true,
		"version": s.version,
		"commit":  s.commit,
	})
}

func (s *Server) handleSubmit(w http.ResponseWriter, r *http.Request) {
	if !s.checkWriteKey(w, r) {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
	var sub feedback.Submission
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&sub); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}

	ip := clientIP(r, s.cfg.TrustedProxyCount)
	err := s.svc.Submit(r.Context(), sub, ip, time.Now().UTC())
	if err != nil {
		switch {
		case errors.Is(err, feedback.ErrRateLimited):
			w.Header().Set("Retry-After", "30")
			writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": "rate_limited"})
		case errors.Is(err, feedback.ErrInvalid):
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_submission"})
		default:
			s.log.Error("submit failed", "err", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleExport(w http.ResponseWriter, r *http.Request) {
	if s.cfg.AdminToken == "" {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not_found"})
		return
	}
	auth := r.Header.Get("Authorization")
	if !strings.HasPrefix(auth, "Bearer ") || !subtleConstantTimeEq(strings.TrimPrefix(auth, "Bearer "), s.cfg.AdminToken) {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	rows, err := s.svc.Export(r.Context())
	if err != nil {
		s.log.Error("export failed", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"count": len(rows), "items": rows})
}

func (s *Server) checkWriteKey(w http.ResponseWriter, r *http.Request) bool {
	if s.cfg.WriteKey == "" {
		return true
	}
	if subtleConstantTimeEq(r.Header.Get("X-Feedback-Key"), s.cfg.WriteKey) {
		return true
	}
	writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	return false
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func clientIP(r *http.Request, trustedProxies int) string {
	if trustedProxies > 0 {
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			parts := strings.Split(xff, ",")
			idx := len(parts) - trustedProxies
			if idx < 0 {
				idx = 0
			}
			ip := strings.TrimSpace(parts[idx])
			if ip != "" {
				return ip
			}
		}
		if xri := strings.TrimSpace(r.Header.Get("X-Real-IP")); xri != "" {
			return xri
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func subtleConstantTimeEq(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	var v byte
	for i := 0; i < len(a); i++ {
		v |= a[i] ^ b[i]
	}
	return v == 0
}
