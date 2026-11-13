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
	"github.com/maicondmenezes/openjob/internal/llm"
	"github.com/maicondmenezes/openjob/internal/store"
)

// openJob — Checkpoint 3 (SQLite + BYOK).
// Go backend with chi serving the embedded SPA, the /api surface and a local
// SQLite store with encrypted BYOK keys (AES-GCM + data/.master.key).
// Configuration comes from OPENJOB_* environment variables (spec §2).
func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg := config.Load()

	// Store: SQLite (modernc, no CGO) with embedded migrations (spec §7/§10.1).
	repo, err := store.Open(cfg.DBPath)
	if err != nil {
		logger.Error("opening store", "err", err)
		os.Exit(1)
	}

	// BYOK master key: generated on first run, mode 0600, next to the DB so a
	// single volume backup carries both (spec §3.2, docs/adr/0004).
	masterKeyPath := store.MasterKeyPath(cfg.DBPath)
	masterKey, err := store.LoadOrCreateMasterKey(masterKeyPath)
	if err != nil {
		logger.Error("loading master key", "err", err)
		os.Exit(1)
	}
	logger.Info("master key ready", "path", masterKeyPath)

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      api.New(cfg, logger, repo, masterKey, llm.NewOpenAIClient()),
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
		if err := repo.Close(); err != nil {
			logger.Warn("store close", "err", err)
		}
		logger.Info("shutdown complete")
	}
}
