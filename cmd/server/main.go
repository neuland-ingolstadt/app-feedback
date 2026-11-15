package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/neuland-ingolstadt/app-feedback/internal/config"
	"github.com/neuland-ingolstadt/app-feedback/internal/feedback"
	"github.com/neuland-ingolstadt/app-feedback/internal/httpapi"
	"github.com/neuland-ingolstadt/app-feedback/internal/store"
)

var (
	version = "dev"
	commit  = "unknown"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	cfg, err := config.Load()
	if err != nil {
		log.Error("config", "err", err)
		os.Exit(1)
	}

	db, err := store.OpenSQLite(cfg.DatabaseURL)
	if err != nil {
		log.Error("database", "err", err)
		os.Exit(1)
	}
	defer db.Close()

	svc := feedback.NewService(cfg, db)
	srv := httpapi.New(cfg, svc, log, version, commit)

	httpServer := &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           srv.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		ticker := time.NewTicker(10 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case now := <-ticker.C:
				svc.GC(now.UTC())
			}
		}
	}()

	go func() {
		log.Info("listening", "addr", cfg.ListenAddr, "version", version, "commit", commit)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("server", "err", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = httpServer.Shutdown(shutdownCtx)
	log.Info("shutdown complete")
}
