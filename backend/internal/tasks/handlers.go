package tasks

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/maicondmenezes/openjob/internal/aiusage"
	"github.com/maicondmenezes/openjob/internal/deterministic"
	"github.com/maicondmenezes/openjob/internal/discovery"
	kwengine "github.com/maicondmenezes/openjob/internal/keywords"
	"github.com/maicondmenezes/openjob/internal/llm"
	"github.com/maicondmenezes/openjob/internal/score"
	"github.com/maicondmenezes/openjob/internal/store"
	"golang.org/x/time/rate"
)

// ---- score_job ------------------------------------------------------------

// ScoreJobHandler is the migrated CP6 batch scoring: for each job it loads
// the base resume, decrypts the BYOK config and calls the same score.Score
// used by the synchronous endpoint — same prompt, same validation, same
// persisted columns (regression-tested against identical payloads).
type ScoreJobHandler struct {
	repo      store.Repository
	masterKey []byte
	llm       llm.Client
	logger    *slog.Logger
	// limiter is shared across every score_job task so the LLM provider's
	// global rate limit is respected even with max_concurrency > 1 (spec §6).
	// Rate limiting is the handler's job, not the pool's, letting free
	// handlers (heuristics) run in parallel.
	limiter *rate.Limiter

	deterministic        *deterministic.Service
	deterministicEnabled bool
}

// NewScoreJobHandler wires the handler. limiter may be nil in tests.
func NewScoreJobHandler(repo store.Repository, masterKey []byte, llmClient llm.Client, limiter *rate.Limiter, logger *slog.Logger) *ScoreJobHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &ScoreJobHandler{
		repo:      repo,
		masterKey: masterKey,
		llm:       llmClient,
		logger:    logger,
		limiter:   limiter,
	}
}

// SetDeterministic wires the deterministic scoring path. When enabled and the
// user has not opted into AI scoring, the handler never calls the LLM.
func (h *ScoreJobHandler) SetDeterministic(svc *deterministic.Service, enabled bool) {
	h.deterministic = svc
	h.deterministicEnabled = enabled
}

func (h *ScoreJobHandler) Type() string { return TypeScoreJob }

func (h *ScoreJobHandler) Run(ctx context.Context, payload json.RawMessage, progress func(int)) (json.RawMessage, error) {
	var p struct {
		JobID int64 `json:"job_id"`
	}
	if err := json.Unmarshal(payload, &p); err != nil || p.JobID <= 0 {
		return nil, fmt.Errorf("payload inválido: job_id obrigatório")
	}

	progress(10)
	job, err := h.repo.GetJob(ctx, p.JobID)
	if err != nil {
		return nil, fmt.Errorf("vaga %d não encontrada: %w", p.JobID, err)
	}

	curriculum, err := h.repo.LatestCurriculum(ctx)
	if err != nil {
		return nil, errors.New("nenhum currículo-base cadastrado para scoring")
	}

	if h.deterministicEnabled && h.deterministic != nil && !aiusage.Load(ctx, h.repo).Score {
		progress(40)
		res, err := h.deterministic.ScoreJob(ctx, curriculum.ContentMD, job.Title, job.Description)
		if err != nil {
			return nil, err
		}
		progress(90)
		if err := h.repo.SetJobScore(ctx, p.JobID, res.Score, res.Reason); err != nil {
			return nil, fmt.Errorf("falha ao persistir score da vaga %d: %w", p.JobID, err)
		}
		progress(100)
		return json.Marshal(map[string]any{
			"job_id":            p.JobID,
			"score_fit":         res.Score,
			"reason":            res.Reason,
			"raw_bm25":          res.RawBM25,
			"calibrated":        res.Calibrated,
			"mechanism_version": res.MechanismVersion,
			"resource_version":  res.ResourceVersion,
			"index_version":     res.IndexVersion,
			"job_label":         fmt.Sprintf("%s @ %s", job.Title, job.Company),
			"curriculum":        curriculum.ID,
		})
	}

	llmCfg, err := h.resolveLLMConfig(ctx)
	if err != nil {
		return nil, err
	}

	// Handlers must yield between steps: a cancelled task relies on ctx.
	if h.limiter != nil {
		if err := h.limiter.Wait(ctx); err != nil {
			return nil, fmt.Errorf("aguardando limite de requisições: %w", err)
		}
	}
	progress(40)

	res, err := score.Score(ctx, h.llm, llmCfg, curriculum.ContentMD, job.Description)
	if err != nil {
		return nil, err // retried with backoff by the pool
	}

	progress(90)
	if err := h.repo.SetJobScore(ctx, p.JobID, res.Score, res.Reason); err != nil {
		return nil, fmt.Errorf("falha ao persistir score da vaga %d: %w", p.JobID, err)
	}
	progress(100)

	return json.Marshal(map[string]any{
		"job_id":     p.JobID,
		"score_fit":  res.Score,
		"reason":     res.Reason,
		"job_label":  fmt.Sprintf("%s @ %s", job.Title, job.Company),
		"curriculum": curriculum.ID,
	})
}

// resolveLLMConfig decrypts the BYOK settings — identical logic to the sync
// score endpoint (jobs.go resolveLLMConfig), kept local to the package to
// avoid an api import.
func (h *ScoreJobHandler) resolveLLMConfig(ctx context.Context) (llm.Config, error) {
	sealedKey, err := h.repo.GetSetting(ctx, "llm.api_key")
	if err != nil || sealedKey == "" {
		return llm.Config{}, errors.New("provedor de IA não configurado — adicione sua chave em Configurações")
	}
	apiKey, err := store.DecryptString(h.masterKey, sealedKey)
	if err != nil || apiKey == "" {
		return llm.Config{}, errors.New("a chave de IA armazenada não pôde ser decifrada — reconfigure em Configurações")
	}
	baseURL, _ := h.repo.GetSetting(ctx, "llm.base_url")
	model, _ := h.repo.GetSetting(ctx, "llm.model")
	return llm.Config{BaseURL: baseURL, Model: model, APIKey: apiKey}.Normalize(), nil
}

// ---- extract_resume_keywords ----------------------------------------------

// ExtractKeywordsHandler migrates the CP6 synchronous resume-keyword refresh
// into the queue: it extracts the top keywords from the latest curriculum
// (pure heuristics, no LLM cost) and caches them in settings.
type ExtractKeywordsHandler struct {
	repo store.Repository

	deterministic        *deterministic.Service
	deterministicEnabled bool
}

// NewExtractKeywordsHandler wires the handler.
func NewExtractKeywordsHandler(repo store.Repository) *ExtractKeywordsHandler {
	return &ExtractKeywordsHandler{repo: repo}
}

// SetDeterministic wires the deterministic keyword engine.
func (h *ExtractKeywordsHandler) SetDeterministic(svc *deterministic.Service, enabled bool) {
	h.deterministic = svc
	h.deterministicEnabled = enabled
}

func (h *ExtractKeywordsHandler) Type() string { return TypeExtractResumeKeywords }

func (h *ExtractKeywordsHandler) Run(ctx context.Context, _ json.RawMessage, progress func(int)) (json.RawMessage, error) {
	curriculum, err := h.repo.LatestCurriculum(ctx)
	if err != nil {
		return nil, errors.New("nenhum currículo-base cadastrado para extração de palavras-chave")
	}

	progress(30)
	// ctx check between steps (cancel propagation contract).
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	keywords := discovery.ExtractResumeKeywords(curriculum.ContentMD, 20)
	algorithmVersion := ""
	if h.deterministicEnabled && h.deterministic != nil {
		if extracted, err := h.deterministic.ExtractKeywordStrings(ctx, curriculum.ContentMD, 20); err == nil && len(extracted) > 0 {
			keywords = extracted
			algorithmVersion = kwengine.AlgorithmVersion
		}
	}

	progress(80)
	b, err := json.Marshal(keywords)
	if err != nil {
		return nil, err
	}
	if err := h.repo.SetSetting(ctx, discovery.SettingResumeKeywords, string(b)); err != nil {
		return nil, fmt.Errorf("falha ao salvar palavras-chave: %w", err)
	}
	progress(100)

	payload := map[string]any{
		"count":      len(keywords),
		"keywords":   keywords,
		"curriculum": curriculum.ID,
		"at":         time.Now().UTC().Format(time.RFC3339),
	}
	if algorithmVersion != "" {
		payload["algorithm_version"] = algorithmVersion
	}
	return json.Marshal(payload)
}
