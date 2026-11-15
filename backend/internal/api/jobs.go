package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/maicondmenezes/openjob/internal/jobfetch"
	"github.com/maicondmenezes/openjob/internal/llm"
	"github.com/maicondmenezes/openjob/internal/score"
	"github.com/maicondmenezes/openjob/internal/store"
)

// jobsAPI serves the manual job entry + scoring + approval endpoints (CP5).
type jobsAPI struct {
	repo      store.Repository
	masterKey []byte
	llm       llm.Client
	logger    *slog.Logger
}

// registerJobs mounts the jobs and applications-list routes.
func registerJobs(r chi.Router, j *jobsAPI) {
	r.Post("/jobs/manual", j.createManual)
	r.Post("/jobs/fetch", j.fetchJob)
	r.Get("/jobs", j.listJobs)
	r.Post("/jobs/{id}/score", j.scoreJob)
	r.Post("/jobs/{id}/approve", j.approveJob)
	r.Post("/jobs/{id}/discard", j.discardJob)
	r.Get("/applications", j.listApplications)
}

// ---- POST /api/jobs/manual -------------------------------------------------

type manualJobRequest struct {
	Title       string `json:"title"`
	Company     string `json:"company"`
	URL         string `json:"url"`
	Description string `json:"description"`
}

// createManual persists a manually pasted vacancy (source='manual',
// status='new'). The URL is optional (many vacancies carry no link) but,
// when present, must be a valid http(s) URL and is deduplicated.
func (j *jobsAPI) createManual(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxSettingsBody)
	var req manualJobRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "JSON inválido"})
		return
	}

	req.Title = strings.TrimSpace(req.Title)
	req.Company = strings.TrimSpace(req.Company)
	req.URL = strings.TrimSpace(req.URL)
	req.Description = strings.TrimSpace(req.Description)

	if req.Title == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "título é obrigatório"})
		return
	}
	if req.Company == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "empresa é obrigatória"})
		return
	}
	if req.Description == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "descrição da vaga (JD) é obrigatória"})
		return
	}

	if req.URL != "" {
		u, err := url.Parse(req.URL)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error": "URL inválida — deve começar com http:// ou https://",
			})
			return
		}

		ctx := r.Context()
		exists, err := j.repo.JobURLExists(ctx, req.URL)
		if err != nil {
			j.logger.Error("job url dedup check failed", "err", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "falha ao verificar duplicidade de vaga"})
			return
		}
		if exists {
			writeJSON(w, http.StatusConflict, map[string]string{
				"error": "Já existe uma vaga cadastrada com essa URL.",
			})
			return
		}
	}

	id, err := j.repo.CreateJob(r.Context(), store.Job{
		Title:       req.Title,
		Company:     req.Company,
		URL:         req.URL,
		Description: req.Description,
		Source:      "manual",
		Status:      "new",
	})
	if err != nil {
		j.logger.Error("create manual job failed", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "falha ao salvar vaga"})
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"id":          id,
		"title":       req.Title,
		"company":     req.Company,
		"url":         req.URL,
		"description": req.Description,
		"source":      "manual",
		"status":      "new",
	})
}

// ---- POST /api/jobs/fetch ----------------------------------------------------

type fetchJobRequest struct {
	URL string `json:"url"`
}

// fetchJob downloads the page at the pasted URL and extracts a draft
// (title, company, description) so the UI can prefill the manual form.
// Nothing is persisted — the user reviews and saves via /jobs/manual.
func (j *jobsAPI) fetchJob(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxSettingsBody)
	var req fetchJobRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "JSON inválido"})
		return
	}
	req.URL = strings.TrimSpace(req.URL)
	if req.URL == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "url é obrigatória"})
		return
	}

	draft, err := jobfetch.Fetch(r.Context(), req.URL)
	switch {
	case errors.Is(err, jobfetch.ErrInvalidURL):
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "URL inválida — deve começar com http:// ou https://",
		})
		return
	case errors.Is(err, jobfetch.ErrNotHTML):
		writeJSON(w, http.StatusBadGateway, map[string]string{
			"error": "A página não é HTML (download/PDF?) — preencha os dados manualmente.",
		})
		return
	case err != nil:
		j.logger.Error("job fetch failed", "url", req.URL, "err", err)
		writeJSON(w, http.StatusBadGateway, map[string]string{
			"error": fmt.Sprintf("Falha ao buscar a página da vaga: %v", err),
		})
		return
	}

	exists, err := j.repo.JobURLExists(r.Context(), draft.URL)
	if err != nil {
		j.logger.Error("job url dedup check failed", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "falha ao verificar duplicidade de vaga"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"url":         draft.URL,
		"title":       draft.Title,
		"company":     draft.Company,
		"description": draft.Description,
		"exists":      exists,
	})
}

// ---- GET /api/jobs ----------------------------------------------------------

// listJobs returns every job ordered by score_fit (NULLs last, newest first
// inside each group) for the /jobs screen.
func (j *jobsAPI) listJobs(w http.ResponseWriter, r *http.Request) {
	jobs, err := j.repo.ListJobs(r.Context(), "")
	if err != nil {
		j.logger.Error("list jobs failed", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "falha ao listar vagas"})
		return
	}

	sort.SliceStable(jobs, func(i, k int) bool {
		si, sk := jobs[i].ScoreFit, jobs[k].ScoreFit
		switch {
		case si != nil && sk != nil:
			if *si != *sk {
				return *si > *sk
			}
			return jobs[i].ID > jobs[k].ID
		case si != nil: // scored before unscored
			return true
		case sk != nil:
			return false
		}
		return jobs[i].ID > jobs[k].ID
	})

	items := make([]map[string]any, 0, len(jobs))
	for _, job := range jobs {
		items = append(items, jobPayload(job))
	}
	writeJSON(w, http.StatusOK, map[string]any{"jobs": items})
}

func jobPayload(job store.Job) map[string]any {
	payload := map[string]any{
		"id":                   job.ID,
		"title":                job.Title,
		"company":              job.Company,
		"url":                  job.URL,
		"description":          job.Description,
		"source":               job.Source,
		"score_reason":         job.ScoreReason,
		"status":               job.Status,
		"external_id":          job.ExternalID,
		"matched_keywords":     job.MatchedKeywords,
		"matched_resume_terms": job.MatchedResumeTerms,
		"created_at":           job.CreatedAt,
	}
	if job.ScoreFit != nil {
		payload["score_fit"] = *job.ScoreFit
	} else {
		payload["score_fit"] = nil
	}
	return payload
}

// ---- POST /api/jobs/:id/score ------------------------------------------------

// scoreJob sends JD + base curriculum to the LLM, persists score_fit and
// score_reason, and returns them. On any LLM failure the score stays NULL.
func (j *jobsAPI) scoreJob(w http.ResponseWriter, r *http.Request) {
	jobID, ok := parseIDParam(w, r)
	if !ok {
		return
	}
	ctx := r.Context()

	job, err := j.repo.GetJob(ctx, jobID)
	if errors.Is(err, store.ErrNotFound) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "vaga não encontrada"})
		return
	}
	if err != nil {
		j.logger.Error("fetch job failed", "id", jobID, "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "falha ao consultar vaga"})
		return
	}

	cur, err := j.repo.LatestCurriculum(ctx)
	if errors.Is(err, store.ErrNotFound) {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "Nenhum currículo-base cadastrado. Faça upload do seu currículo primeiro.",
		})
		return
	}
	if err != nil {
		j.logger.Error("fetch curriculum failed", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "falha ao consultar currículo"})
		return
	}

	cfg, ok := j.resolveLLMConfig(w, ctx)
	if !ok {
		return
	}

	res, err := score.Score(ctx, j.llm, cfg, cur.ContentMD, job.Description)
	if err != nil {
		j.logger.Error("job scoring failed", "id", jobID, "err", err)
		switch {
		case errors.Is(err, score.ErrEmptyCurriculum), errors.Is(err, score.ErrEmptyJobDescription):
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		case errors.Is(err, llm.ErrNotConfigured):
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error": "Provedor de IA não configurado. Adicione sua chave em Configurações.",
			})
		case score.AsErrParse(err):
			writeJSON(w, http.StatusBadGateway, map[string]string{
				"error": fmt.Sprintf("A IA respondeu em um formato inesperado. Tente novamente. (%s)", score.ParseErrorDetail(err)),
			})
		default:
			writeJSON(w, http.StatusBadGateway, map[string]string{
				"error": fmt.Sprintf("Falha ao calcular score pelo provedor de IA: %v", err),
			})
		}
		// score_fit permanece NULL — nada foi persistido.
		return
	}

	if err := j.repo.SetJobScore(ctx, jobID, res.Score, res.Reason); err != nil {
		j.logger.Error("persist job score failed", "id", jobID, "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "falha ao salvar score da vaga"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"id":             jobID,
		"score_fit":      res.Score,
		"score_reason":   res.Reason,
		"prompt_version": score.PromptVersion,
	})
}

// ---- POST /api/jobs/:id/approve ----------------------------------------------

// approveJob creates a pending application for the job and moves it to
// 'approved'. Duplicate approvals are rejected: one ACTIVE application per
// job (spec CP5) — a previously rejected application does not block a new one.
func (j *jobsAPI) approveJob(w http.ResponseWriter, r *http.Request) {
	jobID, ok := parseIDParam(w, r)
	if !ok {
		return
	}
	ctx := r.Context()

	job, err := j.repo.GetJob(ctx, jobID)
	if errors.Is(err, store.ErrNotFound) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "vaga não encontrada"})
		return
	}
	if err != nil {
		j.logger.Error("fetch job failed", "id", jobID, "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "falha ao consultar vaga"})
		return
	}

	if _, err := j.repo.ActiveApplicationForJob(ctx, jobID); err == nil {
		writeJSON(w, http.StatusConflict, map[string]string{
			"error": "Esta vaga já possui uma candidatura ativa no pipeline.",
		})
		return
	} else if !errors.Is(err, store.ErrNotFound) {
		j.logger.Error("active application lookup failed", "id", jobID, "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "falha ao verificar candidaturas"})
		return
	}

	// Status first, reverted if the application insert fails — keeps the
	// invariant "approved ⇔ has active application" in practice.
	if job.Status != "approved" {
		if err := j.repo.UpdateJobStatus(ctx, jobID, "approved"); err != nil {
			j.logger.Error("update job status failed", "id", jobID, "err", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "falha ao aprovar vaga"})
			return
		}
	}

	appID, err := j.repo.CreateApplication(ctx, store.Application{JobID: jobID, Status: "pending"})
	if err != nil {
		if revertErr := j.repo.UpdateJobStatus(ctx, jobID, job.Status); revertErr != nil {
			j.logger.Error("revert job status failed", "id", jobID, "err", revertErr)
		}
		j.logger.Error("create application failed", "id", jobID, "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "falha ao criar candidatura"})
		return
	}

	app, err := j.repo.GetApplication(ctx, appID)
	if err != nil {
		// Created but unreadable: still report success with the essentials.
		j.logger.Error("fetch created application failed", "id", appID, "err", err)
		writeJSON(w, http.StatusCreated, map[string]any{
			"application_id": appID,
			"job_id":         jobID,
			"status":         "pending",
		})
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"application": map[string]any{
			"id":         app.ID,
			"job_id":     app.JobID,
			"status":     app.Status,
			"updated_at": app.UpdatedAt,
		},
	})
}

// ---- POST /api/jobs/:id/discard ----------------------------------------------

// discardJob marks the vacancy as discarded (no application is created).
func (j *jobsAPI) discardJob(w http.ResponseWriter, r *http.Request) {
	jobID, ok := parseIDParam(w, r)
	if !ok {
		return
	}
	if err := j.repo.UpdateJobStatus(r.Context(), jobID, "discarded"); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "vaga não encontrada"})
			return
		}
		j.logger.Error("discard job failed", "id", jobID, "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "falha ao descartar vaga"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"id": jobID, "status": "discarded"})
}

// ---- GET /api/applications ----------------------------------------------------

// listApplications returns the pipeline, each entry enriched with its job
// data (title/company/score) so the UI can render cards directly.
func (j *jobsAPI) listApplications(w http.ResponseWriter, r *http.Request) {
	apps, err := j.repo.ListApplications(r.Context())
	if err != nil {
		j.logger.Error("list applications failed", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "falha ao listar candidaturas"})
		return
	}

	items := make([]map[string]any, 0, len(apps))
	for _, app := range apps {
		item := map[string]any{
			"id":           app.ID,
			"job_id":       app.JobID,
			"status":       app.Status,
			"sent_at":      nil,
			"updated_at":   app.UpdatedAt,
			"has_tailored": app.TailoredCurriculumMD != "",
		}
		if app.SentAt != nil {
			item["sent_at"] = app.SentAt
		}
		if job, err := j.repo.GetJob(r.Context(), app.JobID); err == nil {
			item["job"] = map[string]any{
				"title":     job.Title,
				"company":   job.Company,
				"url":       job.URL,
				"score_fit": job.ScoreFit,
			}
		}
		items = append(items, item)
	}
	writeJSON(w, http.StatusOK, map[string]any{"applications": items})
}

// ---- helpers -------------------------------------------------------------------

// parseIDParam extracts and validates the {id} path parameter.
func parseIDParam(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "id inválido"})
		return 0, false
	}
	return id, true
}

// resolveLLMConfig decrypts the BYOK configuration, mirroring the tailor
// handler. A missing or undecipherable key answers with a clear 400.
func (j *jobsAPI) resolveLLMConfig(w http.ResponseWriter, ctx context.Context) (llm.Config, bool) {
	sealedKey, err := j.repo.GetSetting(ctx, settingLLMAPIKey)
	if errors.Is(err, store.ErrNotFound) || sealedKey == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "Provedor de IA não configurado. Adicione sua chave em Configurações.",
		})
		return llm.Config{}, false
	}
	if err != nil {
		j.logger.Error("fetch llm key setting failed", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "falha ao consultar chave de IA"})
		return llm.Config{}, false
	}

	apiKey, err := store.DecryptString(j.masterKey, sealedKey)
	if err != nil {
		j.logger.Error("decrypt llm key failed", "err", err)
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "A chave armazenada não pôde ser decifrada. Reconfigure em Configurações.",
		})
		return llm.Config{}, false
	}

	baseURL, _ := j.repo.GetSetting(ctx, settingLLMBaseURL)
	model, _ := j.repo.GetSetting(ctx, settingLLMModel)
	return llm.Config{BaseURL: baseURL, APIKey: apiKey, Model: model}.Normalize(), true
}
