package tasks

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/maicondmenezes/openjob/internal/deterministic"
	"github.com/maicondmenezes/openjob/internal/discovery"
	"github.com/maicondmenezes/openjob/internal/llm"
	"github.com/maicondmenezes/openjob/internal/score"
	"github.com/maicondmenezes/openjob/internal/store"
	"golang.org/x/time/rate"
	"time"
)

// fixedLLM returns the same completion for every request — the fixed payload
// of the CP10 regression test ("mesmo score para o mesmo payload").
type fixedLLM struct{ response string }

func (f fixedLLM) Ping(ctx context.Context, cfg llm.Config) error { return nil }
func (f fixedLLM) Complete(ctx context.Context, cfg llm.Config, req llm.CompletionRequest) (string, error) {
	return f.response, nil
}

// TestScoreHandlerMatchesDirectScore is the CP6→CP10 regression: scoring a
// job through the queue handler persists exactly what score.Score returns
// for the same curriculum/JD payload with the same LLM client.
func TestScoreHandlerMatchesDirectScore(t *testing.T) {
	repo, err := store.Open(filepath.Join(t.TempDir(), "openjob.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = repo.Close() }()
	ctx := context.Background()

	// BYOK present (handler decrypts from settings like the CP5 endpoint).
	masterKey, err := store.LoadOrCreateMasterKey(filepath.Join(t.TempDir(), "master.key"))
	if err != nil {
		t.Fatal(err)
	}
	sealed, err := store.EncryptString(masterKey, "sk-test")
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.SetSetting(ctx, "llm.api_key", sealed); err != nil {
		t.Fatal(err)
	}

	const mockResp = "{\"score\": 77, \"reason\": \"Bom match em Go.\"}"
	if _, err := repo.SaveCurriculum(ctx, store.Curriculum{ContentMD: "Go, Docker, Kubernetes"}); err != nil {
		t.Fatal(err)
	}
	jobID, err := repo.CreateJob(ctx, store.Job{
		Title: "Backend Go", Company: "Acme", Description: "Go backend with Docker", Source: "manual", Status: "new",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Direct path (what the CP5 endpoint and CP6 batch used).
	res, err := score.Score(ctx, fixedLLM{mockResp}, llm.Config{}, "Go, Docker, Kubernetes", "Go backend with Docker")
	if err != nil {
		t.Fatal(err)
	}

	// Queue path (CP10): handler with the same LLM client + inputs.
	h := NewScoreJobHandler(repo, masterKey, fixedLLM{mockResp}, rate.NewLimiter(rate.Inf, 0), nil)
	payload, _ := json.Marshal(map[string]any{"job_id": jobID})
	out, err := h.Run(ctx, payload, func(int) {})
	if err != nil {
		t.Fatal(err)
	}
	var handlerResult struct {
		ScoreFit int    `json:"score_fit"`
		Reason   string `json:"reason"`
	}
	if err := json.Unmarshal(out, &handlerResult); err != nil {
		t.Fatal(err)
	}

	if handlerResult.ScoreFit != res.Score {
		t.Fatalf("handler score = %d, direct = %d", handlerResult.ScoreFit, res.Score)
	}
	if handlerResult.Reason != res.Reason {
		t.Fatalf("handler reason = %q, direct = %q", handlerResult.Reason, res.Reason)
	}

	// Persisted on the job exactly like the CP5 endpoint does.
	job, err := repo.GetJob(ctx, jobID)
	if err != nil {
		t.Fatal(err)
	}
	if job.ScoreFit == nil || *job.ScoreFit != res.Score || job.ScoreReason != res.Reason {
		t.Fatalf("persisted score wrong: %+v (want %d/%q)", job, res.Score, res.Reason)
	}
}

// TestScoreHandlerSharedLimiterSequencesGlobalRate shows that two handler
// instances sharing one rate.Limiter (the wiring used with
// max_concurrency > 1) serialize calls: the global provider limit is
// respected (spec §6 risk note).
func TestScoreHandlerSharedLimiter(t *testing.T) {
	lim := rate.NewLimiter(rate.Every(50*time.Millisecond), 1) // 20 req/s shared
	start := time.Now()
	for i := 0; i < 3; i++ {
		if err := lim.Wait(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	elapsed := time.Since(start)
	// 3 waits on a 50ms limiter need ≥ 100ms (first is immediate).
	if elapsed < 100*time.Millisecond {
		t.Fatalf("shared limiter did not serialize: %s", elapsed)
	}
}

// countingLLM fails the test if the deterministic path ever calls the LLM.
type countingLLM struct{ calls int }

func (c *countingLLM) Ping(context.Context, llm.Config) error { c.calls++; return nil }
func (c *countingLLM) Complete(context.Context, llm.Config, llm.CompletionRequest) (string, error) {
	c.calls++
	return "", nil
}

func TestScoreHandlerDeterministicSkipsLLM(t *testing.T) {
	repo, err := store.Open(filepath.Join(t.TempDir(), "openjob.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = repo.Close() }()
	ctx := context.Background()

	if _, err := repo.SaveCurriculum(ctx, store.Curriculum{ContentMD: "Go, Docker, Kubernetes, PostgreSQL"}); err != nil {
		t.Fatal(err)
	}
	jobID, err := repo.CreateJob(ctx, store.Job{
		Title: "Backend Go", Company: "Acme", Description: "Requisitos\n- Go\n- Docker\n- PostgreSQL", Source: "manual", Status: "new",
	})
	if err != nil {
		t.Fatal(err)
	}

	svc, err := deterministic.New(deterministic.DefaultOptions(repo))
	if err != nil {
		t.Fatal(err)
	}
	fake := &countingLLM{}
	h := NewScoreJobHandler(repo, nil, fake, rate.NewLimiter(rate.Inf, 0), nil)
	h.SetDeterministic(svc, true)

	out, err := h.Run(ctx, mustJSON(t, map[string]any{"job_id": jobID}), func(int) {})
	if err != nil {
		t.Fatal(err)
	}
	if fake.calls != 0 {
		t.Fatalf("LLM called %d times on deterministic path", fake.calls)
	}
	var result struct {
		ScoreFit         int    `json:"score_fit"`
		MechanismVersion string `json:"mechanism_version"`
		IndexVersion     string `json:"index_version"`
		Calibrated       bool   `json:"calibrated"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		t.Fatal(err)
	}
	if result.ScoreFit <= 0 || result.MechanismVersion == "" || result.IndexVersion == "" {
		t.Fatalf("unexpected deterministic result: %+v", result)
	}
	if result.Calibrated {
		t.Fatalf("calibrated = true, want false: %+v", result)
	}
	job, err := repo.GetJob(ctx, jobID)
	if err != nil {
		t.Fatal(err)
	}
	if job.ScoreFit == nil || *job.ScoreFit != result.ScoreFit {
		t.Fatalf("persisted score = %v, want %d", job.ScoreFit, result.ScoreFit)
	}
}

func TestExtractKeywordsHandlerDeterministic(t *testing.T) {
	repo, err := store.Open(filepath.Join(t.TempDir(), "openjob.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = repo.Close() }()
	ctx := context.Background()

	curriculum := "## Experiência\n\nGo, Docker e Kubernetes.\n\n## Habilidades\n\nPostgreSQL e Redis."
	if _, err := repo.SaveCurriculum(ctx, store.Curriculum{ContentMD: curriculum}); err != nil {
		t.Fatal(err)
	}
	svc, err := deterministic.New(deterministic.DefaultOptions(repo))
	if err != nil {
		t.Fatal(err)
	}
	h := NewExtractKeywordsHandler(repo)
	h.SetDeterministic(svc, true)

	out, err := h.Run(ctx, nil, func(int) {})
	if err != nil {
		t.Fatal(err)
	}
	var result struct {
		Count            int      `json:"count"`
		Keywords         []string `json:"keywords"`
		AlgorithmVersion string   `json:"algorithm_version"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		t.Fatal(err)
	}
	if result.Count == 0 || len(result.Keywords) == 0 {
		t.Fatalf("no keywords: %+v", result)
	}
	if result.AlgorithmVersion == "" {
		t.Fatal("algorithm_version missing")
	}
	raw, err := repo.GetSetting(ctx, discovery.SettingResumeKeywords)
	if err != nil || raw == "" {
		t.Fatalf("keywords not persisted: %v", err)
	}
}

func mustJSON(t *testing.T, v any) json.RawMessage {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return b
}
