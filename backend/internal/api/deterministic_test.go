package api

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/maicondmenezes/openjob/internal/aiusage"
	"github.com/maicondmenezes/openjob/internal/config"
	"github.com/maicondmenezes/openjob/internal/store"
)

// TestDeterministicScoreEndpoint proves the API path uses the deterministic
// engine (and therefore never calls the LLM) when the master flag is on.
func TestDeterministicScoreEndpoint(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "openjob.db")
	repo, err := store.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = repo.Close() })
	ctx := context.Background()

	if _, err := repo.SaveCurriculum(ctx, store.Curriculum{ContentMD: "Go, PostgreSQL, Docker, Kubernetes"}); err != nil {
		t.Fatal(err)
	}
	jobID, err := repo.CreateJob(ctx, store.Job{
		Title: "Backend Go", Company: "Acme",
		Description: "Requisitos\n- Go\n- PostgreSQL\n- Docker",
		Source:      "manual", Status: "new",
	})
	if err != nil {
		t.Fatal(err)
	}

	masterKey, err := store.LoadOrCreateMasterKey(store.MasterKeyPath(dbPath))
	if err != nil {
		t.Fatal(err)
	}
	cfg := config.Config{
		Port: "0", DBPath: dbPath, Hostname: "openjob.copilot",
		Features: config.Features{ScoreDeterministic: true, KeywordsDeterministic: true, FilterDeterministic: true},
	}
	// Complete returns an error: reaching the LLM path would fail the request.
	handler, runtime := New(cfg, slog.Default(), repo, masterKey, &pingLLMStub{})
	t.Cleanup(runtime.Stop)
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	status, body, decoded := call(t, srv, http.MethodPost, "/api/jobs/"+strconv.FormatInt(jobID, 10)+"/score", nil)
	if status != http.StatusOK {
		t.Fatalf("status = %d, body = %s", status, body)
	}
	if score, _ := decoded["score_fit"].(float64); score <= 0 {
		t.Fatalf("score_fit = %v, body = %s", decoded["score_fit"], body)
	}
	if decoded["mechanism_version"] == "" || decoded["mechanism_version"] == nil {
		t.Fatalf("mechanism_version missing: %s", body)
	}
	if calibrated, ok := decoded["calibrated"].(bool); !ok || calibrated {
		t.Fatalf("calibrated = %v, want false (body %s)", decoded["calibrated"], body)
	}
	if decoded["index_version"] == "" || decoded["index_version"] == nil {
		t.Fatalf("index_version missing: %s", body)
	}

	job, err := repo.GetJob(ctx, jobID)
	if err != nil {
		t.Fatal(err)
	}
	if job.ScoreFit == nil || *job.ScoreFit <= 0 {
		t.Fatalf("score not persisted: %+v", job)
	}

	status, body, decoded = call(t, srv, http.MethodGet, "/api/features", nil)
	if status != http.StatusOK {
		t.Fatalf("features status = %d, body = %s", status, body)
	}
	for _, key := range []string{"score.deterministic", "keywords.deterministic", "filter.deterministic"} {
		if decoded[key] != true {
			t.Errorf("feature %q = %v, want true (body %s)", key, decoded[key], body)
		}
	}
}

// TestManualJobAutoScoresDeterministically proves a manually inserted job gets
// its fit calculated on creation, without any manual "score" action.
func TestManualJobAutoScoresDeterministically(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "openjob.db")
	repo, err := store.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = repo.Close() })
	ctx := context.Background()

	if _, err := repo.SaveCurriculum(ctx, store.Curriculum{ContentMD: "Go, PostgreSQL, Docker, Kubernetes"}); err != nil {
		t.Fatal(err)
	}

	masterKey, err := store.LoadOrCreateMasterKey(store.MasterKeyPath(dbPath))
	if err != nil {
		t.Fatal(err)
	}
	cfg := config.Config{
		Port: "0", DBPath: dbPath, Hostname: "openjob.copilot",
		Features: config.Features{ScoreDeterministic: true, KeywordsDeterministic: true, FilterDeterministic: true},
	}
	handler, runtime := New(cfg, slog.Default(), repo, masterKey, &pingLLMStub{})
	t.Cleanup(runtime.Stop)
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	status, body, decoded := call(t, srv, http.MethodPost, "/api/jobs/manual", map[string]string{
		"title": "Backend Go", "company": "Acme",
		"description": "Requisitos\n- Go\n- PostgreSQL\n- Docker",
	})
	if status != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", status, body)
	}
	score, ok := decoded["score_fit"].(float64)
	if !ok || score <= 0 {
		t.Fatalf("score_fit = %v, want > 0 (body %s)", decoded["score_fit"], body)
	}

	jobID := int64(decoded["id"].(float64))
	job, err := repo.GetJob(ctx, jobID)
	if err != nil {
		t.Fatal(err)
	}
	if job.ScoreFit == nil || *job.ScoreFit <= 0 {
		t.Fatalf("score not persisted on creation: %+v", job)
	}
}

// TestAIOptInSkipsDeterministicScore proves the runtime toggle wins over the
// deterministic engine: with AI scoring enabled and no provider configured,
// the request takes the LLM path and fails instead of silently scoring locally.
func TestAIOptInSkipsDeterministicScore(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "openjob.db")
	repo, err := store.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = repo.Close() })
	ctx := context.Background()

	if _, err := repo.SaveCurriculum(ctx, store.Curriculum{ContentMD: "Go, PostgreSQL, Docker"}); err != nil {
		t.Fatal(err)
	}
	if err := aiusage.Save(ctx, repo, aiusage.Settings{Score: true}); err != nil {
		t.Fatal(err)
	}
	jobID, err := repo.CreateJob(ctx, store.Job{
		Title: "Backend Go", Company: "Acme",
		Description: "Requisitos\n- Go\n- PostgreSQL",
		Source:      "manual", Status: "new",
	})
	if err != nil {
		t.Fatal(err)
	}

	masterKey, err := store.LoadOrCreateMasterKey(store.MasterKeyPath(dbPath))
	if err != nil {
		t.Fatal(err)
	}
	cfg := config.Config{
		Port: "0", DBPath: dbPath, Hostname: "openjob.copilot",
		Features: config.Features{ScoreDeterministic: true, KeywordsDeterministic: true, FilterDeterministic: true},
	}
	handler, runtime := New(cfg, slog.Default(), repo, masterKey, &pingLLMStub{})
	t.Cleanup(runtime.Stop)
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	status, body, _ := call(t, srv, http.MethodPost, "/api/jobs/"+strconv.FormatInt(jobID, 10)+"/score", nil)
	if status == http.StatusOK {
		t.Fatalf("status = 200, want AI path failure (body %s)", body)
	}

	job, err := repo.GetJob(ctx, jobID)
	if err != nil {
		t.Fatal(err)
	}
	if job.ScoreFit != nil {
		t.Fatalf("score persisted through deterministic path: %+v", job)
	}
}
