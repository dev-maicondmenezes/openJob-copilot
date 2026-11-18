// Package ratelimit implements the CP14 three-layer defensive limiter used by
// the discovery orchestrator:
//
//  1. pace  — a token bucket per source (golang.org/x/time/rate) that
//     serializes every HTTP request, no matter how many times the user clicks
//     “Buscar agora”;
//  2. volume — enforced by the adapter via capabilities.max_results_per_run;
//  3. daily quota + circuit breaker — persistent state in SQLite
//     (source_daily_usage / source_health) so restarts never reset the
//     counter or forget a 429/403 backoff.
//
// The package deliberately does not import discovery (the orchestrator imports
// it) to keep the dependency direction one-way.
package ratelimit

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/maicondmenezes/openjob/internal/store"
	"golang.org/x/time/rate"
)

// Config mirrors the declarative rate_limit capability.
type Config struct {
	Requests int
	Period   string
	Strategy string
}

// Limiter owns one token bucket and the persistent health/quota helpers.
type Limiter struct {
	repo store.Repository
	now  func() time.Time

	mu      sync.Mutex
	buckets map[string]*bucket
}

type bucket struct {
	cfg     Config
	limiter *rate.Limiter
}

// New creates a limiter backed by repo (daily counters and health state).
func New(repo store.Repository) *Limiter {
	return &Limiter{repo: repo, now: time.Now, buckets: map[string]*bucket{}}
}

// NewWithClock is the test seam for deterministic day/backoff assertions.
func NewWithClock(repo store.Repository, now func() time.Time) *Limiter {
	l := New(repo)
	if now != nil {
		l.now = now
	}
	return l
}

// SourceWaiter is the per-source handle handed to adapters. It implements
// discovery.RequestWaiter (structurally, no import cycle).
type SourceWaiter struct {
	limiter *rate.Limiter
}

// Wait blocks until the source's token bucket allows one request.
func (w *SourceWaiter) Wait(ctx context.Context) error {
	if w == nil || w.limiter == nil {
		return nil
	}
	return w.limiter.Wait(ctx)
}

// Interval returns the configured spacing between requests (0 when
// rate limiting is disabled). Useful for tests and the UI.
func (w *SourceWaiter) Interval() time.Duration {
	if w == nil || w.limiter == nil {
		return 0
	}
	limit := float64(w.limiter.Limit())
	if limit <= 0 {
		return 0
	}
	return time.Duration(float64(time.Second) / limit)
}

// Waiter returns (creating when needed) the token bucket for a source.
// Conservative by design: burst is always 1, so a run never fires two
// requests back-to-back even when the configured rate allows it.
func (l *Limiter) Waiter(sourceID string, cfg Config) *SourceWaiter {
	normalized := normalize(cfg)
	l.mu.Lock()
	defer l.mu.Unlock()

	if b, ok := l.buckets[sourceID]; ok && b.cfg == normalized {
		return &SourceWaiter{limiter: b.limiter}
	}

	var lim *rate.Limiter
	if normalized.Requests > 0 {
		interval, ok := periodDuration(normalized.Period)
		if ok {
			lim = rate.NewLimiter(rate.Every(interval/time.Duration(normalized.Requests)), 1)
		}
	}
	l.buckets[sourceID] = &bucket{cfg: normalized, limiter: lim}
	return &SourceWaiter{limiter: lim}
}

func normalize(c Config) Config {
	c.Period = strings.ToLower(strings.TrimSpace(c.Period))
	c.Strategy = strings.ToLower(strings.TrimSpace(c.Strategy))
	if c.Strategy == "" {
		c.Strategy = "token_bucket"
	}
	return c
}

// periodDuration accepts the documented names and Go durations ("2s").
func periodDuration(period string) (time.Duration, bool) {
	switch strings.ToLower(strings.TrimSpace(period)) {
	case "second", "seconds", "s":
		return time.Second, true
	case "minute", "minutes", "m":
		return time.Minute, true
	case "hour", "hours", "h":
		return time.Hour, true
	case "day", "days", "d":
		return 24 * time.Hour, true
	}
	if d, err := time.ParseDuration(period); err == nil && d > 0 {
		return d, true
	}
	return 0, false
}

// Day returns the local calendar day used as the quota key.
func (l *Limiter) Day() string { return l.now().Format("2006-01-02") }

// DailyUsage returns today's counter for a source (zero when absent).
func (l *Limiter) DailyUsage(ctx context.Context, sourceID string) (store.SourceDailyUsage, error) {
	usage, err := l.repo.GetSourceDailyUsage(ctx, sourceID, l.Day())
	if err != nil {
		if err == store.ErrNotFound {
			return store.SourceDailyUsage{SourceID: sourceID, Day: l.Day()}, nil
		}
		return usage, err
	}
	return usage, nil
}

// Remaining returns how many quota units a source may still consume today.
// unit is "requests" (default) or "details". quota <= 0 means unlimited.
func (l *Limiter) Remaining(ctx context.Context, sourceID string, quota int, unit string) (int, error) {
	if quota <= 0 {
		return -1, nil
	}
	usage, err := l.DailyUsage(ctx, sourceID)
	if err != nil {
		return 0, err
	}
	used := usage.Used
	if unit == "details" {
		used = usage.DetailsUsed
	}
	remaining := quota - used
	if remaining < 0 {
		remaining = 0
	}
	return remaining, nil
}

// ConsumeDaily registers usage and returns the fresh counter.
func (l *Limiter) ConsumeDaily(ctx context.Context, sourceID string, usedDelta, detailsDelta int) (store.SourceDailyUsage, error) {
	if usedDelta < 0 {
		usedDelta = 0
	}
	if detailsDelta < 0 {
		detailsDelta = 0
	}
	if usedDelta == 0 && detailsDelta == 0 {
		return l.DailyUsage(ctx, sourceID)
	}
	return l.repo.AddSourceDailyUsage(ctx, sourceID, l.Day(), usedDelta, detailsDelta)
}

// Gap is the state returned by Check: when Blocked, the source must not run
// until Until (or forever when Until is nil, e.g. manual block).
type Gap struct {
	Blocked bool
	Status  string
	Reason  string
	Until   *time.Time
}

// Check inspects the persisted circuit breaker/backoff state. Expired
// backoffs are normalized back to "ok" and persisted best-effort.
func (l *Limiter) Check(ctx context.Context, sourceID string) (Gap, error) {
	h, err := l.repo.GetSourceHealth(ctx, sourceID)
	if err != nil {
		if err == store.ErrNotFound {
			return Gap{Status: "ok"}, nil
		}
		return Gap{}, err
	}

	now := l.now()
	if h.DegradedUntil != nil && h.DegradedUntil.After(now) {
		return Gap{
			Blocked: true,
			Status:  "degraded",
			Reason:  fmt.Sprintf("fonte degradada após %d falhas; nova tentativa em %s", h.ConsecutiveFailures, h.DegradedUntil.Local().Format("02/01 15:04")),
			Until:   h.DegradedUntil,
		}, nil
	}
	if h.BackoffUntil != nil && h.BackoffUntil.After(now) {
		return Gap{
			Blocked: true,
			Status:  "blocked",
			Reason:  fmt.Sprintf("backoff ativo após bloqueio; nova tentativa em %s", h.BackoffUntil.Local().Format("02/01 15:04")),
			Until:   h.BackoffUntil,
		}, nil
	}

	// Expired states: heal lazily so long-running installs do not stay
	// degraded forever after the penalty window. A healed circuit breaker
	// starts a fresh failure streak; an expired 429 backoff keeps the streak
	// so a repeated block still escalates to 4h/24h.
	if h.Status != "ok" || h.BackoffUntil != nil || h.DegradedUntil != nil {
		if h.Status == "degraded" {
			h.ConsecutiveFailures = 0
		}
		h.Status = "ok"
		h.BackoffUntil = nil
		h.DegradedUntil = nil
		_ = l.repo.UpsertSourceHealth(ctx, h)
	}
	return Gap{Status: "ok"}, nil
}

// FailureResult reports the transition caused by a failure.
type FailureResult struct {
	Health     store.SourceHealth
	DidDegrade bool
}

// RecordFailure increments the consecutive failure counter and applies the
// CP14 policy:
//   - 429/403 (blocked): retry after 1h, 4h, then 24h;
//   - any third consecutive failure: degraded for 24h (circuit breaker).
func (l *Limiter) RecordFailure(ctx context.Context, sourceID, errMsg string, blocked bool) (FailureResult, error) {
	h, err := l.repo.GetSourceHealth(ctx, sourceID)
	if err != nil && err != store.ErrNotFound {
		return FailureResult{}, err
	}
	if err == store.ErrNotFound {
		h = store.SourceHealth{SourceID: sourceID, Status: "ok"}
	}

	h.ConsecutiveFailures++
	h.LastError = errMsg
	// Persist absolute instants in UTC: the store parses them back with RFC3339
	// and a local-zone string would lose the offset in some code paths.
	now := l.now().UTC()

	if blocked {
		h.Status = "backoff"
		var delay time.Duration
		switch {
		case h.ConsecutiveFailures <= 1:
			delay = time.Hour
		case h.ConsecutiveFailures == 2:
			delay = 4 * time.Hour
		default:
			delay = 24 * time.Hour
		}
		until := now.Add(delay)
		h.BackoffUntil = &until
	} else {
		// Non-blocking failures only trip the circuit breaker on the third
		// consecutive error; before that the source stays runnable.
		h.Status = "ok"
		h.BackoffUntil = nil
	}

	didDegrade := false
	if h.ConsecutiveFailures >= 3 {
		until := now.Add(24 * time.Hour)
		h.Status = "degraded"
		h.DegradedUntil = &until
		// Keep the backoff shorter than the degraded window, mirroring the
		// documented retry schedule (degraded wins while it lasts).
		didDegrade = true
	}

	if err := l.repo.UpsertSourceHealth(ctx, h); err != nil {
		return FailureResult{}, err
	}
	return FailureResult{Health: h, DidDegrade: didDegrade}, nil
}

// RecordSuccess clears the failure counter and closes any backoff/circuit.
func (l *Limiter) RecordSuccess(ctx context.Context, sourceID string) error {
	now := l.now().UTC()
	h := store.SourceHealth{
		SourceID:            sourceID,
		Status:              "ok",
		ConsecutiveFailures: 0,
		LastSuccessAt:       &now,
	}
	return l.repo.UpsertSourceHealth(ctx, h)
}

// HealthMap returns the persisted health of every source with a row.
func (l *Limiter) HealthMap(ctx context.Context) (map[string]store.SourceHealth, error) {
	rows, err := l.repo.ListSourceHealth(ctx)
	if err != nil {
		return nil, err
	}
	out := make(map[string]store.SourceHealth, len(rows))
	for _, h := range rows {
		out[h.SourceID] = h
	}
	return out, nil
}

// UsageMap returns today's counters keyed by source id.
func (l *Limiter) UsageMap(ctx context.Context) (map[string]store.SourceDailyUsage, error) {
	rows, err := l.repo.ListSourceDailyUsage(ctx, l.Day())
	if err != nil {
		return nil, err
	}
	out := make(map[string]store.SourceDailyUsage, len(rows))
	for _, u := range rows {
		out[u.SourceID] = u
	}
	return out, nil
}
