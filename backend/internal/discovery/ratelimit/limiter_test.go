package ratelimit

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/maicondmenezes/openjob/internal/store"
)

func openStore(t *testing.T) *store.SQLiteStore {
	t.Helper()
	repo, err := store.Open(filepath.Join(t.TempDir(), "openjob.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = repo.Close() })
	return repo
}

func TestTokenBucketIsConservative(t *testing.T) {
	repo := openStore(t)
	l := New(repo)

	// 2 requests/second ⇒ 500ms spacing, burst 1.
	waiter := l.Waiter("scraper", Config{Requests: 2, Period: "second"})
	if got := waiter.Interval(); got != 500*time.Millisecond {
		t.Fatalf("interval = %s, want 500ms", got)
	}
	ctx := context.Background()
	if err := waiter.Wait(ctx); err != nil {
		t.Fatalf("first wait: %v", err)
	}
	// The second request must not fire immediately: a short deadline proves
	// the bucket serializes repeated clicks.
	short, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if err := waiter.Wait(short); err == nil {
		t.Fatal("second wait succeeded immediately, want rate limit")
	}

	// Go duration syntax is also accepted for slower sources (1 req/2s).
	slow := l.Waiter("slow", Config{Requests: 1, Period: "2s"})
	if got := slow.Interval(); got != 2*time.Second {
		t.Fatalf("slow interval = %s, want 2s", got)
	}
}

func TestDailyQuotaPersistsAndResetsAtLocalMidnight(t *testing.T) {
	repo := openStore(t)
	now := time.Date(2026, 5, 10, 23, 0, 0, 0, time.Local)
	l := NewWithClock(repo, func() time.Time { return now })
	ctx := context.Background()

	if _, err := l.ConsumeDaily(ctx, "vagas", 5, 5); err != nil {
		t.Fatal(err)
	}
	remaining, err := l.Remaining(ctx, "vagas", 20, "details")
	if err != nil {
		t.Fatal(err)
	}
	if remaining != 15 {
		t.Fatalf("remaining = %d, want 15", remaining)
	}

	// A fresh limiter (e.g. process restart) sees the persisted counter.
	l2 := NewWithClock(repo, func() time.Time { return now })
	remaining, _ = l2.Remaining(ctx, "vagas", 20, "details")
	if remaining != 15 {
		t.Fatalf("after restart remaining = %d, want 15", remaining)
	}
	usage, _ := l2.DailyUsage(ctx, "vagas")
	if usage.DetailsUsed != 5 || usage.Used != 5 {
		t.Fatalf("usage = %+v, want 5/5", usage)
	}

	// Next local day → new row → quota fully available.
	now = now.Add(2 * time.Hour)
	remaining, _ = l2.Remaining(ctx, "vagas", 20, "details")
	if remaining != 20 {
		t.Fatalf("next day remaining = %d, want 20", remaining)
	}
}

func TestBackoffAndCircuitBreaker(t *testing.T) {
	repo := openStore(t)
	now := time.Date(2026, 5, 10, 8, 0, 0, 0, time.Local)
	l := NewWithClock(repo, func() time.Time { return now })
	ctx := context.Background()

	res, err := l.RecordFailure(ctx, "computrabajo", "HTTP 429", true)
	if err != nil {
		t.Fatal(err)
	}
	if res.Health.Status != "backoff" || res.Health.BackoffUntil == nil {
		t.Fatalf("health = %+v, want backoff", res.Health)
	}
	gap, _ := l.Check(ctx, "computrabajo")
	if !gap.Blocked || gap.Status != "blocked" {
		t.Fatalf("gap = %+v, want blocked", gap)
	}

	// Expired backoff heals lazily.
	now = now.Add(61 * time.Minute)
	gap, _ = l.Check(ctx, "computrabajo")
	if gap.Blocked {
		t.Fatalf("gap after 1h = %+v, want allowed", gap)
	}

	// Three consecutive failures → degraded for 24h.
	for i := 0; i < 3; i++ {
		res, err = l.RecordFailure(ctx, "catho", "parse error", false)
		if err != nil {
			t.Fatal(err)
		}
	}
	if res.Health.Status != "degraded" || res.Health.DegradedUntil == nil {
		t.Fatalf("health = %+v, want degraded", res.Health)
	}
	gap, _ = l.Check(ctx, "catho")
	if !gap.Blocked || gap.Status != "degraded" {
		t.Fatalf("gap = %+v, want degraded", gap)
	}
	now = now.Add(25 * time.Hour)
	gap, _ = l.Check(ctx, "catho")
	if gap.Blocked {
		t.Fatalf("gap after 24h = %+v, want allowed", gap)
	}

	// Success clears the counter.
	if err := l.RecordSuccess(ctx, "catho"); err != nil {
		t.Fatal(err)
	}
	h, err := repo.GetSourceHealth(ctx, "catho")
	if err != nil {
		t.Fatal(err)
	}
	if h.Status != "ok" || h.ConsecutiveFailures != 0 {
		t.Fatalf("health after success = %+v", h)
	}
}

func TestCheckMissingHealthIsOK(t *testing.T) {
	repo := openStore(t)
	l := New(repo)
	gap, err := l.Check(context.Background(), "never-seen")
	if err != nil {
		t.Fatal(err)
	}
	if gap.Blocked || gap.Status != "ok" {
		t.Fatalf("gap = %+v, want ok", gap)
	}
	if _, err := repo.GetSourceHealth(context.Background(), "never-seen"); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}
