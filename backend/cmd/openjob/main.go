package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/maicondmenezes/openjob/internal/api"
	"github.com/maicondmenezes/openjob/internal/config"
)

// openJob — Checkpoint 1 (Foundation).
// Go backend with chi serving the embedded SPA and /api/health.
// Configuration comes from OPENJOB_* environment variables (spec §2).
func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg := config.Load()

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      api.New(cfg, logger),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		logger.Info("openjob starting",
			"port", cfg.Port,
			"db_path", cfg.DBPath,
			"hostname", cfg.Hostname,
			"url", "http://"+cfg.Hostname+":"+cfg.Port,
		)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errCh:
		logger.Error("server error", "err", err)
		os.Exit(1)
	case sig := <-sigCh:
		logger.Info("shutting down", "signal", sig.String())
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			logger.Error("graceful shutdown failed", "err", err)
			os.Exit(1)
		}
		logger.Info("shutdown complete")
	}
}
