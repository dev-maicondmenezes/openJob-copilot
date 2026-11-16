package discovery

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"testing"

	"github.com/maicondmenezes/openjob/internal/store"
)

type mockAdapter struct {
	slug string
	jobs []DiscoveredJob
	err  error
}

func (m *mockAdapter) Slug() string { return m.slug }
func (m *mockAdapter) Name() string { return "Mock " + m.slug }
func (m *mockAdapter) DefaultCapabilities() Capabilities {
	return Capabilities{SupportsKeywordSearch: true}
}
func (m *mockAdapter) Fetch(ctx context.Context, cfg SourceConfig, keywords []string) ([]DiscoveredJob, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.jobs, nil
}

func openTestStore(t *testing.T) *store.SQLiteStore {
	t.Helper()
	repo, err := store.Open(filepath.Join(t.TempDir(), "openjob.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = repo.Close() })
	return repo
}

func TestOrchestratorConfigValidation(t *testing.T) {
	repo := openTestStore(t)
	orch := NewOrchestrator(repo, nil, nil, nil, nil)
	ctx := context.Background()

	// >5 keywords must fail
	err := orch.SaveConfig(ctx, Config{
		Keywords: []string{"go", "docker", "k8s", "aws", "gcp", "azure"},
	})
	if err == nil {
		t.Error("expected error for >5 keywords, got nil")
	}

	// <= 5 keywords passes
	err = orch.SaveConfig(ctx, Config{
		Keywords: []string{"go", "docker", "k8s"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	loaded, err := orch.LoadConfig(ctx)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if len(loaded.Keywords) != 3 {
		t.Errorf("expected 3 keywords, got %d", len(loaded.Keywords))
	}
}

func TestOrchestratorPipelineAndDedup(t *testing.T) {
	repo := openTestStore(t)
	ctx := context.Background()

	// Seed resume
	_, err := repo.SaveCurriculum(ctx, store.Curriculum{
		ContentMD: "Senior Developer with Go, Docker, Kubernetes, Microservices experience",
	})
	if err != nil {
		t.Fatal(err)
	}

	registry := NewRegistry()
	mock := &mockAdapter{
		slug: "test-src",
		jobs: []DiscoveredJob{
			{
				Title:       "Go Developer",
				Company:     "Corp A",
				URL:         "https://example.com/job/1",
				Description: "Build Go microservices with Docker and Kubernetes.",
				Source:      "test-src",
				ExternalID:  "1",
			},
			{
				Title:       "Ruby Developer",
				Company:     "Corp B",
				URL:         "https://example.com/job/2",
				Description: "Ruby on Rails with PostgreSQL and Redis.",
				Source:      "test-src",
				ExternalID:  "2",
			},
		},
	}
	registry.Register(mock)

	// Clean existing seeded sources so only mock runs
	existing, err := repo.ListSources(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range existing {
		_ = repo.DeleteSource(ctx, s.ID)
	}
	err = repo.CreateSource(ctx, store.Source{
		ID:           "test-src",
		Name:         "Test Source",
		Adapter:      "test-src",
		Enabled:      true,
		Priority:     1,
		Capabilities: `{"supports_keyword_search":true}`,
	})
	if err != nil {
		t.Fatal(err)
	}

	orch := NewOrchestrator(repo, registry, nil, nil, nil)

	// Configure keywords: only "go"
	err = orch.SaveConfig(ctx, Config{
		Keywords: []string{"go"},
		ResumeFilter: ResumeFilterSettings{
			Enabled:    true,
			MinMatches: 2,
			MatchMode:  "any",
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	// First Run: Job 1 passes (has "go" and resume matches), Job 2 is filtered out (no "go")
	runID, stats, err := orch.Discover(ctx)
	if err != nil {
		t.Fatalf("discover run 1: %v", err)
	}
	if runID <= 0 {
		t.Errorf("invalid runID: %d", runID)
	}
	if stats.Fetched != 2 {
		t.Errorf("fetched = %d, want 2", stats.Fetched)
	}
	if stats.NewJobs != 1 {
		t.Errorf("new_jobs = %d, want 1", stats.NewJobs)
	}
	if stats.FilteredKeywords != 1 {
		t.Errorf("filtered_keywords = %d, want 1", stats.FilteredKeywords)
	}

	// Second Run with same mock jobs: Job 1 must be deduplicated! (new_jobs = 0, duplicates = 1)
	_, stats2, err := orch.Discover(ctx)
	if err != nil {
		t.Fatalf("discover run 2: %v", err)
	}
	if stats2.NewJobs != 0 {
		t.Errorf("run 2 new_jobs = %d, want 0", stats2.NewJobs)
	}
	if stats2.Duplicates != 1 {
		t.Errorf("run 2 duplicates = %d, want 1", stats2.Duplicates)
	}
}

// fakeEnqueuer records EnqueueScoreJob calls (CP10: discovery → queue).
type fakeEnqueuer struct {
	mu     sync.Mutex
	jobIDs []int64
	err    error
}

func (f *fakeEnqueuer) EnqueueScoreJob(_ context.Context, jobID int64, _ int) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return f.err
	}
	f.jobIDs = append(f.jobIDs, jobID)
	return nil
}

// TestOrchestratorEnqueuesScoring verifies the CP10 migration: with an
// enqueuer wired, discovery enqueues the new jobs for async scoring (up to
// maxBatchScoringPerRun, prioritized by resume matches) instead of scoring
// inline.
func TestOrchestratorEnqueuesScoring(t *testing.T) {
	repo := openTestStore(t)
	ctx := context.Background()

	_, err := repo.SaveCurriculum(ctx, store.Curriculum{
		ContentMD: "Go, Docker, Kubernetes",
	})
	if err != nil {
		t.Fatal(err)
	}

	registry := NewRegistry()
	var jobs []DiscoveredJob
	for i := 1; i <= 3; i++ {
		jobs = append(jobs, DiscoveredJob{
			Title:       fmt.Sprintf("Go Developer %d", i),
			Company:     "Corp",
			URL:         fmt.Sprintf("https://example.com/job/%d", i),
			Description: "Go backend with Docker and Kubernetes.",
			Source:      "test-src",
		})
	}
	registry.Register(&mockAdapter{slug: "test-src", jobs: jobs})

	for _, s := range mustListSources(t, repo) {
		_ = repo.DeleteSource(ctx, s.ID)
	}
	if err := repo.CreateSource(ctx, store.Source{
		ID: "test-src", Name: "Test", Adapter: "test-src", Enabled: true, Priority: 1,
		Capabilities: `{"supports_keyword_search":true}`,
	}); err != nil {
		t.Fatal(err)
	}

	orch := NewOrchestrator(repo, registry, nil, nil, nil)
	_ = orch.SaveConfig(ctx, Config{Keywords: []string{"go"}, ResumeFilter: ResumeFilterSettings{Enabled: false}})

	enq := &fakeEnqueuer{}
	orch.SetScoreEnqueuer(enq)

	_, stats, err := orch.Discover(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if stats.NewJobs != 3 {
		t.Fatalf("new_jobs = %d, want 3", stats.NewJobs)
	}
	if stats.Scored != 3 {
		t.Fatalf("queued scoring = %d, want 3", stats.Scored)
	}
	enq.mu.Lock()
	ids := enq.jobIDs
	enq.mu.Unlock()
	if len(ids) != 3 {
		t.Fatalf("enqueued %d jobs, want 3", len(ids))
	}
}

func mustListSources(t *testing.T, repo store.Repository) []store.Source {
	t.Helper()
	sources, err := repo.ListSources(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	return sources
}

func TestOrchestratorFaultTolerance(t *testing.T) {
	repo := openTestStore(t)
	ctx := context.Background()

	registry := NewRegistry()
	failing := &mockAdapter{
		slug: "failing-src",
		err:  errors.New("connection reset by peer"),
	}
	working := &mockAdapter{
		slug: "working-src",
		jobs: []DiscoveredJob{
			{
				Title:       "Backend Engineer",
				Company:     "Corp A",
				URL:         "https://example.com/job/10",
				Description: "Go backend",
				Source:      "working-src",
			},
		},
	}
	registry.Register(failing)
	registry.Register(working)

	// Clean existing seeded sources so only mock sources run
	existing, err := repo.ListSources(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range existing {
		_ = repo.DeleteSource(ctx, s.ID)
	}

	_ = repo.CreateSource(ctx, store.Source{
		ID:           "failing-src",
		Name:         "Failing",
		Adapter:      "failing-src",
		Enabled:      true,
		Priority:     1,
		Capabilities: `{}`,
	})
	_ = repo.CreateSource(ctx, store.Source{
		ID:           "working-src",
		Name:         "Working",
		Adapter:      "working-src",
		Enabled:      true,
		Priority:     2,
		Capabilities: `{}`,
	})

	orch := NewOrchestrator(repo, registry, nil, nil, nil)
	_ = orch.SaveConfig(ctx, Config{
		ResumeFilter: ResumeFilterSettings{Enabled: false},
	})

	// Discovery should not abort; working source must succeed despite failing source
	_, stats, err := orch.Discover(ctx)
	if err != nil {
		t.Fatalf("discover should not return error when one source fails: %v", err)
	}
	if len(stats.Errors) != 1 {
		t.Errorf("expected 1 error logged in stats, got %d", len(stats.Errors))
	}
	if stats.NewJobs != 1 {
		t.Errorf("expected 1 new job from working source, got %d", stats.NewJobs)
	}
}
