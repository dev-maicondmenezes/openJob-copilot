package tasks

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/maicondmenezes/openjob/internal/store"
)

// SettingEmailSyncInterval is the settings key holding the email polling
// interval ("off", "5m", "15m", "30m", "1h"). It mirrors the key used by the
// settings API so both read the same value.
const SettingEmailSyncInterval = "email.sync_interval"

// DefaultEmailSyncInterval is the polling cadence suggested by CP17.
const DefaultEmailSyncInterval = "5m"

// EmailSyncScheduler periodically enqueues a sync_email task. The ticker is
// idle unless at least one email account is enabled (off by default), which
// keeps the local-first promise: no background network until the user opts in.
type EmailSyncScheduler struct {
	repo   store.Repository
	pool   *Pool
	logger *slog.Logger

	mu     sync.Mutex
	ticker *time.Ticker
	stop   chan struct{}
}

// NewEmailSyncScheduler creates a scheduler; call Configure to start it.
func NewEmailSyncScheduler(repo store.Repository, pool *Pool, logger *slog.Logger) *EmailSyncScheduler {
	if logger == nil {
		logger = slog.Default()
	}
	return &EmailSyncScheduler{repo: repo, pool: pool, logger: logger}
}

// Start configures the ticker from the persisted interval, falling back to the
// CP17 default when the setting is missing.
func (s *EmailSyncScheduler) Start(ctx context.Context) {
	interval := DefaultEmailSyncInterval
	if raw, err := s.repo.GetSetting(ctx, SettingEmailSyncInterval); err == nil && strings.TrimSpace(raw) != "" {
		interval = raw
	}
	s.Configure(interval)
}

// Configure starts, stops or adjusts the polling ticker. "off" disables it.
func (s *EmailSyncScheduler) Configure(interval string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.ticker != nil {
		s.ticker.Stop()
		s.ticker = nil
	}
	if s.stop != nil {
		close(s.stop)
		s.stop = nil
	}

	d, ok := parseEmailSyncInterval(interval)
	if !ok {
		return
	}

	ticker := time.NewTicker(d)
	stop := make(chan struct{})
	s.ticker = ticker
	s.stop = stop

	go func() {
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				s.enqueueIfIdle()
			}
		}
	}()
}

// Stop halts the ticker.
func (s *EmailSyncScheduler) Stop() {
	s.Configure("off")
}

func (s *EmailSyncScheduler) enqueueIfIdle() {
	if s.pool == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	accounts, err := s.repo.ListEmailAccounts(ctx)
	if err != nil {
		s.logger.Warn("email scheduler: list accounts failed", "err", err)
		return
	}
	enabled := false
	for _, acc := range accounts {
		if acc.Enabled {
			enabled = true
			break
		}
	}
	if !enabled {
		return
	}

	// Avoid stacking syncs: skip when one is already queued or running.
	if n, err := s.repo.CountTasks(ctx, []string{"queued", "running"}, TypeSyncEmail); err == nil && n > 0 {
		return
	}

	if _, err := s.pool.Enqueue(ctx, TypeSyncEmail, json.RawMessage("{}"), 50); err != nil {
		s.logger.Warn("email scheduler: enqueue sync failed", "err", err)
		return
	}
	s.logger.Info("email scheduler: sync_email enqueued")
}

// parseEmailSyncInterval accepts the values exposed by /settings/email plus any
// duration >= 1 minute. Empty and "off" disable the ticker.
func parseEmailSyncInterval(interval string) (time.Duration, bool) {
	switch strings.ToLower(strings.TrimSpace(interval)) {
	case "", "off":
		return 0, false
	case "5m":
		return 5 * time.Minute, true
	case "15m":
		return 15 * time.Minute, true
	case "30m":
		return 30 * time.Minute, true
	case "1h":
		return time.Hour, true
	}

	parsed, err := time.ParseDuration(interval)
	if err != nil || parsed < time.Minute {
		return 0, false
	}
	return parsed, true
}
