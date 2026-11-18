package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/maicondmenezes/openjob/internal/store"
)

func mustCapsJSON(t *testing.T, caps Capabilities) string {
	t.Helper()
	b, err := json.Marshal(caps)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

type cp14Adapter struct {
	slug  string
	caps  Capabilities
	jobs  []DiscoveredJob
	err   error
	calls int
	onRun func(cfg SourceConfig)
}

func (a *cp14Adapter) Slug() string                      { return a.slug }
func (a *cp14Adapter) Name() string                      { return "CP14 " + a.slug }
func (a *cp14Adapter) DefaultCapabilities() Capabilities { return a.caps }
func (a *cp14Adapter) Fetch(_ context.Context, cfg SourceConfig, _ []string) ([]DiscoveredJob, error) {
	a.calls++
	if a.onRun != nil {
		a.onRun(cfg)
	}
	return a.jobs, a.err
}

func setupCP14Orchestrator(t *testing.T, adapters ...Adapter) (*Orchestrator, *store.SQLiteStore) {
	t.Helper()
	repo := openTestStore(t)
	ctx := context.Background()
	for _, s := range mustListSources(t, repo) {
		_ = repo.DeleteSource(ctx, s.ID)
	}
	registry := NewRegistry()
	for _, a := range adapters {
		registry.Register(a)
		capsJSON := mustCapsJSON(t, a.DefaultCapabilities())
		if err := repo.CreateSource(ctx, store.Source{
			ID:           a.Slug(),
			Name:         a.Name(),
			Adapter:      a.Slug(),
			Enabled:      true,
			Priority:     10,
			Capabilities: capsJSON,
		}); err != nil {
			t.Fatal(err)
		}
	}
	orch := NewOrchestrator(repo, registry, nil, nil, nil)
	if err := orch.SaveConfig(ctx, Config{ResumeFilter: ResumeFilterSettings{Enabled: false}}); err != nil {
		t.Fatal(err)
	}
	return orch, repo
}

func TestOrchestratorPersistsDailyRequestQuotaAndSkipsExhaustedSource(t *testing.T) {
	adapter := &cp14Adapter{
		slug: "quota-src",
		caps: Capabilities{
			AccessType:         AccessScraping,
			MaxResultsPerRun:   10,
			RequiresDailyQuota: true,
			DailyQuota:         2,
			QuotaUnit:          "requests",
		},
		onRun: func(cfg SourceConfig) {
			if cfg.Usage != nil {
				cfg.Usage.Requests = 2
			}
		},
	}
	orch, repo := setupCP14Orchestrator(t, adapter)
	ctx := context.Background()

	_, stats, err := orch.Discover(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if adapter.calls != 1 || stats.Sources[0].DailyUsed != 2 {
		t.Fatalf("calls=%d stat=%+v, want 1 call and 2 used", adapter.calls, stats.Sources[0])
	}
	usage, err := repo.GetSourceDailyUsage(ctx, "quota-src", orch.Limiter().Day())
	if err != nil || usage.Used != 2 {
		t.Fatalf("usage = %+v err=%v, want 2 used", usage, err)
	}

	// Quota exhausted: the adapter is not called; the stat explains why.
	_, stats2, err := orch.Discover(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if adapter.calls != 1 {
		t.Fatalf("adapter calls = %d, want still 1", adapter.calls)
	}
	if !stats2.Sources[0].Skipped || stats2.Sources[0].Status != "quota" || stats2.Sources[0].Note == "" {
		t.Fatalf("stat = %+v, want skipped/quota with note", stats2.Sources[0])
	}
}

func TestOrchestratorKeepsListingWhenDetailQuotaIsExhausted(t *testing.T) {
	var adapter *cp14Adapter
	adapter = &cp14Adapter{
		slug: "detail-quota-src",
		caps: Capabilities{
			AccessType:         AccessScraping,
			MaxResultsPerRun:   10,
			RequiresDailyQuota: true,
			DailyQuota:         2,
			QuotaUnit:          "details",
		},
		onRun: func(cfg SourceConfig) {
			adapter.jobs = []DiscoveredJob{{
				Title:  "Go Dev",
				URL:    fmt.Sprintf("https://example.com/q/%d", adapter.calls),
				Source: "detail-quota-src",
			}}
			if cfg.Usage != nil && cfg.DailyRemaining > 0 {
				cfg.Usage.DetailRequests = 2
			}
		},
	}
	orch, repo := setupCP14Orchestrator(t, adapter)
	ctx := context.Background()

	_, stats, err := orch.Discover(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if adapter.calls != 1 || stats.Sources[0].DetailsUsed != 2 {
		t.Fatalf("calls=%d stat=%+v, want 1 call and 2 details", adapter.calls, stats.Sources[0])
	}

	// Details are exhausted, but the listing is free: the source still runs,
	// receives DailyRemaining=0 and the run reports the quota note.
	_, stats2, err := orch.Discover(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if adapter.calls != 2 {
		t.Fatalf("adapter calls = %d, want 2 (listing must keep running)", adapter.calls)
	}
	stat := stats2.Sources[0]
	if stat.Skipped || stat.Status != "ok" || stat.DetailsUsed != 2 || stat.Note == "" {
		t.Fatalf("stat = %+v, want running/ok with details note", stat)
	}
	usage, err := repo.GetSourceDailyUsage(ctx, "detail-quota-src", orch.Limiter().Day())
	if err != nil || usage.DetailsUsed != 2 {
		t.Fatalf("usage = %+v err=%v, want still 2 details", usage, err)
	}
}

func TestOrchestratorCircuitBreakerDegradesAfterThreeFailures(t *testing.T) {
	adapter := &cp14Adapter{slug: "broken-src", caps: Capabilities{AccessType: AccessAPI, MaxResultsPerRun: 1}, err: context.DeadlineExceeded}
	orch, repo := setupCP14Orchestrator(t, adapter)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		if _, _, err := orch.Discover(ctx); err != nil {
			t.Fatalf("run %d: %v", i+1, err)
		}
	}
	if adapter.calls != 3 {
		t.Fatalf("calls = %d, want 3", adapter.calls)
	}
	health, err := repo.GetSourceHealth(ctx, "broken-src")
	if err != nil {
		t.Fatal(err)
	}
	if health.Status != "degraded" || health.DegradedUntil == nil {
		t.Fatalf("health = %+v, want degraded", health)
	}

	// Fourth run: skipped without touching the adapter.
	_, stats, err := orch.Discover(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if adapter.calls != 3 {
		t.Fatalf("calls after degrader = %d, want 3", adapter.calls)
	}
	if !stats.Sources[0].Skipped || stats.Sources[0].Status != "degraded" {
		t.Fatalf("stat = %+v, want skipped/degraded", stats.Sources[0])
	}
}

func TestOrchestratorBlockedErrorSchedulesBackoff(t *testing.T) {
	adapter := &cp14Adapter{
		slug: "blocked-src",
		caps: Capabilities{AccessType: AccessScraping, MaxResultsPerRun: 1, RateLimit: RateLimitConfig{Requests: 1, Period: "second"}},
		err:  &BlockedError{StatusCode: 429},
	}
	orch, repo := setupCP14Orchestrator(t, adapter)
	ctx := context.Background()

	_, stats, err := orch.Discover(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if stats.Sources[0].Status != "backoff" || stats.Sources[0].BackoffUntil == "" {
		t.Fatalf("stat = %+v, want backoff with until", stats.Sources[0])
	}
	health, _ := repo.GetSourceHealth(ctx, "blocked-src")
	if health.BackoffUntil == nil || !health.BackoffUntil.After(time.Now().Add(-time.Minute)) {
		t.Fatalf("health = %+v, want future backoff", health)
	}
	if gap, _ := orch.Limiter().Check(ctx, "blocked-src"); !gap.Blocked {
		t.Fatalf("gap = %+v, want blocked (health=%+v)", gap, health)
	}

	_, stats2, err := orch.Discover(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if adapter.calls != 1 {
		t.Fatalf("calls = %d, want 1 (backoff must skip the source)", adapter.calls)
	}
	if !stats2.Sources[0].Skipped || stats2.Sources[0].Status != "blocked" {
		t.Fatalf("stat = %+v, want skipped/blocked", stats2.Sources[0])
	}
}

func TestOrchestratorRunsAPISourcesBeforeScraping(t *testing.T) {
	var order []string
	api := &cp14Adapter{
		slug:  "api-src",
		caps:  Capabilities{AccessType: AccessAPI, MaxResultsPerRun: 1},
		onRun: func(SourceConfig) { order = append(order, "api") },
	}
	scrape := &cp14Adapter{
		slug:  "scrape-src",
		caps:  Capabilities{AccessType: AccessScraping, MaxResultsPerRun: 1},
		onRun: func(SourceConfig) { order = append(order, "scraping") },
	}
	// Register in inverse order and give scraping a better priority; access
	// type must still win.
	orch, repo := setupCP14Orchestrator(t, scrape, api)
	ctx := context.Background()
	_ = repo.UpdateSource(ctx, store.Source{
		ID: "scrape-src", Name: "scrape", Adapter: "scrape-src", Enabled: true, Priority: 1,
		Capabilities: mustCapsJSON(t, scrape.caps),
	})
	if _, _, err := orch.Discover(ctx); err != nil {
		t.Fatal(err)
	}
	if len(order) != 2 || order[0] != "api" || order[1] != "scraping" {
		t.Fatalf("order = %v, want [api scraping]", order)
	}
}
