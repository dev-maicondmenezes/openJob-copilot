package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/maicondmenezes/openjob/internal/events"
	"github.com/maicondmenezes/openjob/internal/store"
	"github.com/maicondmenezes/openjob/internal/tasks"
)

// tasksAPI serves the task queue management endpoints (CP10).
type tasksAPI struct {
	repo   store.Repository
	pool   *tasks.Pool
	bus    *events.Bus
	logger *slog.Logger
}

// registerTasksAPI mounts the tasks routes + the worker config.
func registerTasksAPI(r chi.Router, t *tasksAPI) {
	r.Get("/tasks", t.list)
	r.Get("/tasks/stream", t.stream) // static route wins over {id} in chi
	r.Get("/tasks/{id}", t.get)
	r.Post("/tasks/{id}/cancel", t.cancel)
	r.Post("/tasks/{id}/retry", t.retry)
	r.Post("/tasks/cancel-all-queued", t.cancelAllQueued)
	r.Post("/tasks/retry-all-failed", t.retryAllFailed)

	r.Get("/settings/tasks", t.getConfig)
	r.Put("/settings/tasks", t.updateConfig)
}

// taskPayload renders a task row for the UI.
func taskPayload(t store.Task) map[string]any {
	var result, payload any
	_ = json.Unmarshal([]byte(t.Payload), &payload)
	if t.Result != "" {
		_ = json.Unmarshal([]byte(t.Result), &result)
	}
	item := map[string]any{
		"id":           t.ID,
		"type":         t.Type,
		"status":       t.Status,
		"payload":      payload,
		"result":       result,
		"error":        t.Error,
		"progress":     t.Progress,
		"attempts":     t.Attempts,
		"max_attempts": t.MaxAttempts,
		"priority":     t.Priority,
		"created_at":   t.CreatedAt,
		"started_at":   nil,
		"finished_at":  nil,
	}
	if t.StartedAt != nil {
		item["started_at"] = t.StartedAt
	}
	if t.FinishedAt != nil {
		item["finished_at"] = t.FinishedAt
	}
	return item
}

// ---- GET /api/tasks ----------------------------------------------------------

// list supports status=queued,running (comma list), type, limit and a
// keyset cursor — the polling fallback of the SSE stream.
func (t *tasksAPI) list(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	if limit <= 0 {
		limit = 50
	}
	var statuses []string
	if s := q.Get("status"); s != "" {
		for _, part := range strings.Split(s, ",") {
			if part = strings.TrimSpace(part); part != "" {
				statuses = append(statuses, part)
			}
		}
	}
	items, next, err := t.repo.ListTasks(r.Context(), statuses, q.Get("type"), q.Get("cursor"), limit)
	if err != nil {
		t.logger.Error("list tasks failed", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "falha ao listar tarefas"})
		return
	}
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		out = append(out, taskPayload(item))
	}
	counts, _ := t.repo.CountTasksByStatus(r.Context())
	writeJSON(w, http.StatusOK, map[string]any{
		"tasks":        out,
		"next_cursor":  next,
		"status_counts": counts,
	})
}

// ---- GET /api/tasks/{id} ------------------------------------------------------

func (t *tasksAPI) get(w http.ResponseWriter, r *http.Request) {
	task, err := t.repo.GetTask(r.Context(), chi.URLParam(r, "id"))
	if errors.Is(err, store.ErrNotFound) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "tarefa não encontrada"})
		return
	}
	if err != nil {
		t.logger.Error("get task failed", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "falha ao consultar tarefa"})
		return
	}
	writeJSON(w, http.StatusOK, taskPayload(task))
}

// ---- POST /api/tasks/{id}/cancel ----------------------------------------------

func (t *tasksAPI) cancel(w http.ResponseWriter, r *http.Request) {
	task, err := t.pool.CancelTask(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "tarefa não encontrada"})
			return
		}
		writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, taskPayload(task))
}

// ---- POST /api/tasks/{id}/retry ------------------------------------------------

func (t *tasksAPI) retry(w http.ResponseWriter, r *http.Request) {
	task, err := t.pool.RetryTask(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "tarefa não encontrada"})
			return
		}
		writeJSON(w, http.StatusConflict, map[string]string{"error": "só tarefas failed ou cancelled podem ser reenfileiradas"})
		return
	}
	writeJSON(w, http.StatusOK, taskPayload(task))
}

// ---- bulk actions (optional per spec §4.4) ---------------------------------------

func (t *tasksAPI) cancelAllQueued(w http.ResponseWriter, r *http.Request) {
	affected, err := t.repo.CancelAllQueuedTasks(r.Context(), "Cancelada pelo usuário.")
	if err != nil {
		t.logger.Error("cancel all queued failed", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "falha ao cancelar tarefas"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"updated": affected})
}

func (t *tasksAPI) retryAllFailed(w http.ResponseWriter, r *http.Request) {
	affected, err := t.repo.RetryAllFailedTasks(r.Context())
	if err != nil {
		t.logger.Error("retry all failed failed", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "falha ao reenfileirar tarefas"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"updated": affected})
}

// ---- GET /api/tasks/stream (SSE) ---------------------------------------------------

// stream pushes task state changes (queued/running/done/failed/cancelled) and
// progress updates over Server-Sent Events. The subscription is best-effort:
// when the subscriber buffer is full the event is dropped and UIs fall back
// to polling GET /api/tasks (spec §4.3). Heartbeats keep intermediaries from
// closing the idle connection.
func (t *tasksAPI) stream(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "streaming não suportado"})
		return
	}

	w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
	_, _ = fmt.Fprint(w, ": conectado\n\n")
	flusher.Flush()

	sub := t.bus.Subscribe("api.tasks.stream."+newClientID(), 64)
	defer sub.Close()

	heartbeat := time.NewTicker(15 * time.Second)
	defer heartbeat.Stop()

	writeEvent := func(evType string, data map[string]any) bool {
		b, err := json.Marshal(data)
		if err != nil {
			return true
		}
		if _, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", evType, b); err != nil {
			return false
		}
		flusher.Flush()
		return true
	}

	for {
		select {
		case <-r.Context().Done():
			return
		case <-heartbeat.C:
			if _, err := fmt.Fprint(w, ": ping\n\n"); err != nil {
				return
			}
			flusher.Flush()
		case e, ok := <-sub.C():
			if !ok {
				return
			}
			data := map[string]any{
				"kind":     e.Kind,
				"severity": e.Severity,
			}
			for _, key := range []string{"task_id", "task_type", "progress", "status", "attempts"} {
				if v, exists := e.Metadata[key]; exists {
					data[key] = v
				}
			}
			if !writeEvent("task", data) {
				return
			}
		}
	}
}

// newClientID gives each SSE subscriber a unique subscription name.
func newClientID() string {
	return strconv.FormatInt(time.Now().UnixNano(), 36)
}

// ---- GET/PUT /api/settings/tasks ---------------------------------------------------

type taskSettingsRequest struct {
	MaxConcurrency *int `json:"max_concurrency"`
	WorkerPollMS   *int `json:"worker_poll_ms"`
}

func (t *tasksAPI) getConfig(w http.ResponseWriter, r *http.Request) {
	cfg, err := tasks.LoadConfig(r.Context(), t.repo)
	if err != nil {
		t.logger.Error("load tasks config failed", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "falha ao carregar configuração"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"max_concurrency": cfg.MaxConcurrency,
		"worker_poll_ms":  cfg.WorkerPollMS,
	})
}

func (t *tasksAPI) updateConfig(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxSettingsBody)
	var req taskSettingsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "JSON inválido"})
		return
	}

	cfg, err := tasks.LoadConfig(r.Context(), t.repo)
	if err != nil {
		t.logger.Error("load tasks config failed", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "falha ao carregar configuração"})
		return
	}
	if req.MaxConcurrency != nil {
		cfg.MaxConcurrency = *req.MaxConcurrency
	}
	if req.WorkerPollMS != nil {
		cfg.WorkerPollMS = *req.WorkerPollMS
	}
	if err := tasks.SaveConfig(r.Context(), t.repo, cfg); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	t.pool.SetConfig(cfg) // workers adjust at runtime, no restart needed
	writeJSON(w, http.StatusOK, map[string]any{
		"max_concurrency": cfg.MaxConcurrency,
		"worker_poll_ms":  cfg.WorkerPollMS,
	})
}
