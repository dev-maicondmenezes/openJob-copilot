package api

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/maicondmenezes/openjob/internal/discovery"
	"github.com/maicondmenezes/openjob/internal/store"
	"github.com/maicondmenezes/openjob/internal/tasks"
)

type discoveryAPI struct {
	orchestrator *discovery.Orchestrator
	repo         store.Repository
	logger       *slog.Logger
	pool         *tasks.Pool // CP10: keyword refresh enqueued as a task
}

func registerDiscovery(r chi.Router, d *discoveryAPI) {
	// Settings
	r.Get("/settings/discovery", d.getConfig)
	r.Put("/settings/discovery", d.updateConfig)
	r.Post("/settings/discovery/resume-keywords/refresh", d.refreshResumeKeywords)

	// Discovery runs
	r.Post("/jobs/discover", d.triggerDiscover)
	r.Get("/jobs/discover/status", d.getDiscoveryStatus)
	r.Get("/jobs/discover/{id}", d.getDiscoveryRun)
}

func (d *discoveryAPI) getConfig(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	cfg, err := d.orchestrator.LoadConfig(ctx)
	if err != nil {
		d.logger.Error("load discovery config failed", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "falha ao carregar configuração de busca"})
		return
	}

	resumeKws, _ := d.orchestrator.GetResumeKeywords(ctx)

	writeJSON(w, http.StatusOK, map[string]any{
		"keywords":           cfg.Keywords,
		"resume_filter":      cfg.ResumeFilter,
		"discovery_interval": cfg.DiscoveryInterval,
		"resume_keywords":    resumeKws,
	})
}

func (d *discoveryAPI) updateConfig(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxSettingsBody)
	var req discovery.Config
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_json", "invalid JSON", nil)
		return
	}

	ctx := r.Context()
	if err := d.orchestrator.SaveConfig(ctx, req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	cfg, _ := d.orchestrator.LoadConfig(ctx)
	resumeKws, _ := d.orchestrator.GetResumeKeywords(ctx)

	writeJSON(w, http.StatusOK, map[string]any{
		"keywords":           cfg.Keywords,
		"resume_filter":      cfg.ResumeFilter,
		"discovery_interval": cfg.DiscoveryInterval,
		"resume_keywords":    resumeKws,
	})
}

// refreshResumeKeywords re-extracts the resume keywords. Default flow (CP10):
// enqueue an extract_resume_keywords task — the UI tracks it via /tasks (SSE).
// `?sync=true` keeps the synchronous fallback so tests and impatient users
// don't wait for the queue.
func (d *discoveryAPI) refreshResumeKeywords(w http.ResponseWriter, r *http.Request) {
	if r.URL.Query().Get("sync") != "true" && d.pool != nil {
		task, err := d.pool.Enqueue(r.Context(), tasks.TypeExtractResumeKeywords, []byte(`{}`), 1)
		if err != nil {
			d.logger.Error("enqueue keyword refresh failed", "err", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "falha ao enfileirar extração de palavras-chave"})
			return
		}
		writeJSON(w, http.StatusAccepted, map[string]any{
			"task_id": task.ID,
			"status":  task.Status,
		})
		return
	}

	keywords, err := d.orchestrator.RefreshResumeKeywords(r.Context())
	if err != nil {
		d.logger.Error("refresh resume keywords failed", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "falha ao extrair palavras-chave do currículo"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"resume_keywords": keywords,
	})
}

func (d *discoveryAPI) triggerDiscover(w http.ResponseWriter, r *http.Request) {
	if d.orchestrator.IsRunning() {
		writeJSON(w, http.StatusConflict, map[string]any{
			"error":  "Já existe uma descoberta em andamento.",
			"run_id": d.orchestrator.CurrentRunID(),
		})
		return
	}

	// Run in background goroutine so HTTP client receives immediate run ID
	// and can poll / stream status
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		_, _, err := d.orchestrator.Discover(ctx)
		if err != nil {
			d.logger.Error("background discovery run failed", "err", err)
		}
	}()

	// Give a moment for run record creation
	for i := 0; i < 20; i++ {
		if id := d.orchestrator.CurrentRunID(); id > 0 {
			writeJSON(w, http.StatusAccepted, map[string]any{
				"run_id": id,
				"status": "running",
			})
			return
		}
		time.Sleep(20 * time.Millisecond)
	}

	writeJSON(w, http.StatusAccepted, map[string]any{
		"status": "running",
	})
}

func (d *discoveryAPI) getDiscoveryStatus(w http.ResponseWriter, r *http.Request) {
	running := d.orchestrator.IsRunning()
	runID := d.orchestrator.CurrentRunID()

	ctx := r.Context()
	latest, err := d.repo.LatestDiscoveryRun(ctx)
	var latestData any
	if err == nil {
		var stats any
		_ = json.Unmarshal([]byte(latest.Stats), &stats)
		completedAt := ""
		if latest.CompletedAt != nil {
			completedAt = latest.CompletedAt.Format(time.RFC3339)
		}
		latestData = map[string]any{
			"id":            latest.ID,
			"started_at":    latest.StartedAt.Format(time.RFC3339),
			"completed_at":  completedAt,
			"status":        latest.Status,
			"stats":         stats,
			"error_message": latest.ErrorMessage,
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"running":    running,
		"run_id":     runID,
		"latest_run": latestData,
	})
}

func (d *discoveryAPI) getDiscoveryRun(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_id", "invalid id", nil)
		return
	}

	run, err := d.repo.GetDiscoveryRun(r.Context(), id)
	if errors.Is(err, store.ErrNotFound) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "execução não encontrada"})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "falha ao buscar execução"})
		return
	}

	var stats any
	_ = json.Unmarshal([]byte(run.Stats), &stats)
	completedAt := ""
	if run.CompletedAt != nil {
		completedAt = run.CompletedAt.Format(time.RFC3339)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"id":            run.ID,
		"started_at":    run.StartedAt.Format(time.RFC3339),
		"completed_at":  completedAt,
		"status":        run.Status,
		"stats":         stats,
		"error_message": run.ErrorMessage,
	})
}
