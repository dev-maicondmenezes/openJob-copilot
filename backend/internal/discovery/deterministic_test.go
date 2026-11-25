package discovery

import (
	"context"
	"log/slog"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/maicondmenezes/openjob/internal/filtering"
	"github.com/maicondmenezes/openjob/internal/store"
)

func newDiscoveryTestRepo(t *testing.T) *store.SQLiteStore {
	t.Helper()
	repo, err := store.Open(filepath.Join(t.TempDir(), "openjob.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = repo.Close() })
	return repo
}

type stubFilterer struct {
	called bool
	result filtering.Result
	err    error
}

func (s *stubFilterer) FilterJob(context.Context, int64, []string, string, string, string, filtering.Policy) (filtering.Result, error) {
	s.called = true
	return s.result, s.err
}

func TestFilterResumeDecisionUsesDeterministicEngine(t *testing.T) {
	repo := newDiscoveryTestRepo(t)
	o := NewOrchestrator(repo, nil, nil, nil, slog.Default())
	stub := &stubFilterer{result: filtering.Result{Accepted: true, MatchedTerms: []string{"go", "docker"}}}
	o.SetResumeFilterer(stub)

	cfg := Config{ResumeFilter: ResumeFilterSettings{Enabled: true, MinMatches: 3, MatchMode: "any"}}
	accepted, matched := o.filterResumeDecision(context.Background(), "Backend Go", "Go e Docker", []string{"go", "docker"}, "curriculum", cfg)
	if !stub.called {
		t.Fatal("deterministic filterer was not used")
	}
	if !accepted || !reflect.DeepEqual(matched, []string{"go", "docker"}) {
		t.Fatalf("accepted = %v, matched = %v", accepted, matched)
	}
}

func TestFilterResumeDecisionFallsBackToLegacy(t *testing.T) {
	repo := newDiscoveryTestRepo(t)
	o := NewOrchestrator(repo, nil, nil, nil, slog.Default())
	stub := &stubFilterer{result: filtering.Result{Accepted: false}}
	o.SetResumeFilterer(stub)

	cfg := Config{ResumeFilter: ResumeFilterSettings{Enabled: true, MinMatches: 1, MatchMode: "any"}}
	accepted, matched := o.filterResumeDecision(context.Background(), "Backend Go", "Go e Docker", []string{"go"}, "", cfg)
	if stub.called {
		t.Fatal("deterministic filterer should not run without curriculum text")
	}
	if !accepted || !reflect.DeepEqual(matched, []string{"go"}) {
		t.Fatalf("accepted = %v, matched = %v", accepted, matched)
	}
}

func TestBatchScoreUsesDeterministicScorer(t *testing.T) {
	repo := newDiscoveryTestRepo(t)
	ctx := context.Background()
	if _, err := repo.SaveCurriculum(ctx, store.Curriculum{ContentMD: "Go, Docker"}); err != nil {
		t.Fatal(err)
	}
	jobID, err := repo.CreateJob(ctx, store.Job{Title: "Backend Go", Company: "Acme", Description: "Go e Docker", Source: "manual", Status: "new"})
	if err != nil {
		t.Fatal(err)
	}

	o := NewOrchestrator(repo, nil, nil, nil, slog.Default())
	o.SetDeterministicScorer(func(context.Context, string, string, string) (int, string, error) {
		return 88, "deterministic match", nil
	})

	scored := o.batchScore(ctx, []jobToScore{{ID: jobID, Title: "Backend Go", Description: "Go e Docker"}})
	if scored != 1 {
		t.Fatalf("scored = %d, want 1", scored)
	}
	job, err := repo.GetJob(ctx, jobID)
	if err != nil {
		t.Fatal(err)
	}
	if job.ScoreFit == nil || *job.ScoreFit != 88 || job.ScoreReason != "deterministic match" {
		t.Fatalf("persisted score = %+v", job)
	}
}

func TestRefreshResumeKeywordsUsesDeterministicExtractor(t *testing.T) {
	repo := newDiscoveryTestRepo(t)
	ctx := context.Background()
	if _, err := repo.SaveCurriculum(ctx, store.Curriculum{ContentMD: "Go, Docker"}); err != nil {
		t.Fatal(err)
	}

	o := NewOrchestrator(repo, nil, nil, nil, slog.Default())
	o.SetKeywordExtractor(func(string, int) []string { return []string{"go", "docker"} })

	got, err := o.RefreshResumeKeywords(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, []string{"go", "docker"}) {
		t.Fatalf("keywords = %v", got)
	}
	cached, err := o.GetResumeKeywords(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(cached, got) {
		t.Fatalf("cached = %v, want %v", cached, got)
	}
}

// TestDiscoverScoresDeterministicallyInline proves deterministic discovery
// scoring is synchronous: the new jobs already have a fit when Discover
// returns, and no queue task is required (so the UI does not show unscored
// jobs right after a search).
func TestDiscoverScoresDeterministicallyInline(t *testing.T) {
	repo := openTestStore(t)
	ctx := context.Background()

	if _, err := repo.SaveCurriculum(ctx, store.Curriculum{ContentMD: "Go, Docker, Kubernetes"}); err != nil {
		t.Fatal(err)
	}

	registry := NewRegistry()
	registry.Register(&mockAdapter{slug: "det-src", jobs: []DiscoveredJob{{
		Title:       "Go Developer",
		Company:     "Corp",
		URL:         "https://example.com/job/det-1",
		Description: "Go backend with Docker and Kubernetes.",
		Source:      "det-src",
	}}})
	for _, s := range mustListSources(t, repo) {
		_ = repo.DeleteSource(ctx, s.ID)
	}
	if err := repo.CreateSource(ctx, store.Source{
		ID: "det-src", Name: "Det", Adapter: "det-src", Enabled: true, Priority: 1,
		Capabilities: `{"supports_keyword_search":true}`,
	}); err != nil {
		t.Fatal(err)
	}

	orch := NewOrchestrator(repo, registry, nil, nil, slog.Default())
	if err := orch.SaveConfig(ctx, Config{Keywords: []string{"go"}, ResumeFilter: ResumeFilterSettings{Enabled: false}}); err != nil {
		t.Fatal(err)
	}
	orch.SetDeterministicScorer(func(context.Context, string, string, string) (int, string, error) {
		return 91, "inline deterministic", nil
	})
	enq := &fakeEnqueuer{}
	orch.SetScoreEnqueuer(enq)

	_, stats, err := orch.Discover(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if stats.NewJobs != 1 || stats.Scored != 1 {
		t.Fatalf("stats = %+v, want 1 new and 1 scored", stats)
	}
	enq.mu.Lock()
	enqueued := len(enq.jobIDs)
	enq.mu.Unlock()
	if enqueued != 0 {
		t.Fatalf("enqueued %d jobs, want 0 for deterministic inline scoring", enqueued)
	}

	jobs, err := repo.ListJobs(ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(jobs) != 1 || jobs[0].ScoreFit == nil || *jobs[0].ScoreFit != 91 {
		t.Fatalf("jobs = %+v, want one job scored 91", jobs)
	}
}
