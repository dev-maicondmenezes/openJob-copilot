package store

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestSourceDailyUsageUpsertAndList(t *testing.T) {
	repo := openTestStore(t)
	ctx := context.Background()

	if _, err := repo.GetSourceDailyUsage(ctx, "vagas", "2026-05-10"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing usage err = %v, want ErrNotFound", err)
	}

	u, err := repo.AddSourceDailyUsage(ctx, "vagas", "2026-05-10", 1, 1)
	if err != nil {
		t.Fatalf("add usage: %v", err)
	}
	if u.Used != 1 || u.DetailsUsed != 1 {
		t.Fatalf("usage = %+v, want 1/1", u)
	}
	u, err = repo.AddSourceDailyUsage(ctx, "vagas", "2026-05-10", 2, 2)
	if err != nil {
		t.Fatal(err)
	}
	if u.Used != 3 || u.DetailsUsed != 3 {
		t.Fatalf("usage after increment = %+v, want 3/3", u)
	}

	other, err := repo.AddSourceDailyUsage(ctx, "infojobs", "2026-05-10", 4, 0)
	if err != nil {
		t.Fatal(err)
	}
	if other.Used != 4 {
		t.Fatalf("other usage = %+v", other)
	}

	rows, err := repo.ListSourceDailyUsage(ctx, "2026-05-10")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatalf("rows = %d, want 2", len(rows))
	}
	// New day starts empty (natural midnight reset).
	nextDay, err := repo.AddSourceDailyUsage(ctx, "vagas", "2026-05-11", 5, 0)
	if err != nil {
		t.Fatal(err)
	}
	if nextDay.Used != 5 || nextDay.DetailsUsed != 0 {
		t.Fatalf("next day = %+v", nextDay)
	}
}

func TestSourceHealthUpsertAndList(t *testing.T) {
	repo := openTestStore(t)
	ctx := context.Background()

	if _, err := repo.GetSourceHealth(ctx, "catho"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing health err = %v, want ErrNotFound", err)
	}

	until := time.Now().UTC().Add(time.Hour)
	h := SourceHealth{
		SourceID:            "catho",
		Status:              "backoff",
		ConsecutiveFailures: 1,
		BackoffUntil:        &until,
		LastError:           "HTTP 429",
	}
	if err := repo.UpsertSourceHealth(ctx, h); err != nil {
		t.Fatal(err)
	}
	got, err := repo.GetSourceHealth(ctx, "catho")
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != "backoff" || got.ConsecutiveFailures != 1 || got.BackoffUntil == nil {
		t.Fatalf("health = %+v", got)
	}

	h.Status = "ok"
	h.ConsecutiveFailures = 0
	h.BackoffUntil = nil
	if err := repo.UpsertSourceHealth(ctx, h); err != nil {
		t.Fatal(err)
	}
	got, _ = repo.GetSourceHealth(ctx, "catho")
	if got.Status != "ok" || got.BackoffUntil != nil {
		t.Fatalf("health after reset = %+v", got)
	}

	rows, err := repo.ListSourceHealth(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].SourceID != "catho" {
		t.Fatalf("rows = %+v", rows)
	}
}

func TestMigrationSeedsCP14Sources(t *testing.T) {
	repo := openTestStore(t)
	ctx := context.Background()
	sources, err := repo.ListSources(ctx)
	if err != nil {
		t.Fatal(err)
	}
	byID := map[string]Source{}
	for _, s := range sources {
		byID[s.ID] = s
	}
	for _, id := range []string{
		"getonboard", "trampos", "gupy", "getmanfred",
		"programathor", "computrabajo", "bumeran", "vagas", "catho", "infojobs",
	} {
		if _, ok := byID[id]; !ok {
			t.Errorf("source %q not seeded by migration", id)
		}
	}
	if got := byID["vagas"].Capabilities; !strings.Contains(got, `"daily_quota":20`) {
		t.Errorf("vagas capabilities missing daily quota: %s", got)
	}
}
