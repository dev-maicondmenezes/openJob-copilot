package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/maicondmenezes/openjob/internal/discovery"
	"github.com/maicondmenezes/openjob/internal/store"
)

type sourcesAPI struct {
	repo         store.Repository
	orchestrator *discovery.Orchestrator
	masterKey    []byte
	logger       *slog.Logger
}

func registerSources(r chi.Router, s *sourcesAPI) {
	r.Get("/sources", s.listSources)
	r.Get("/sources/adapters", s.listAdapters)
	r.Post("/sources", s.createSource)
	r.Patch("/sources/{id}", s.updateSource)
	r.Delete("/sources/{id}", s.deleteSource)
	r.Post("/sources/{id}/test", s.testSource)
}

// sourceHealthResponse is the CP14 persisted circuit-breaker state surfaced
// in /settings/sources.
type sourceHealthResponse struct {
	Status              string `json:"status"` // ok | backoff | degraded | blocked
	ConsecutiveFailures int    `json:"consecutive_failures"`
	BackoffUntil        string `json:"backoff_until,omitempty"`
	DegradedUntil       string `json:"degraded_until,omitempty"`
	LastError           string `json:"last_error,omitempty"`
}

type sourceResponse struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	Adapter      string                 `json:"adapter"`
	Enabled      bool                   `json:"enabled"`
	Priority     int                    `json:"priority"`
	AuthType     string                 `json:"auth_type"`
	HasAPIKey    bool                   `json:"has_api_key"`
	Capabilities discovery.Capabilities `json:"capabilities"`
	CreatedAt    string                 `json:"created_at"`
	UpdatedAt    string                 `json:"updated_at"`

	// CP14 runtime state: today's quota consumption + health.
	UsageToday  int                  `json:"usage_today"`
	UsageQuota  int                  `json:"usage_quota"`
	UsageUnit   string               `json:"usage_unit"`
	DetailsUsed int                  `json:"details_used"`
	Health      sourceHealthResponse `json:"health"`
}

func (s *sourcesAPI) toSourceResponse(src store.Source) sourceResponse {
	var caps discovery.Capabilities
	if err := json.Unmarshal([]byte(src.Capabilities), &caps); err != nil {
		if adapter, ok := s.orchestrator.Registry().Get(src.Adapter); ok {
			caps = adapter.DefaultCapabilities()
		}
	}
	unit := caps.QuotaUnit
	if unit == "" {
		unit = "requests"
	}
	return sourceResponse{
		ID:           src.ID,
		Name:         src.Name,
		Adapter:      src.Adapter,
		Enabled:      src.Enabled,
		Priority:     src.Priority,
		AuthType:     src.AuthType,
		HasAPIKey:    src.AuthType == "api_key" && src.AuthConfig != "",
		Capabilities: caps,
		CreatedAt:    src.CreatedAt.Format(time.RFC3339),
		UpdatedAt:    src.UpdatedAt.Format(time.RFC3339),
		UsageQuota:   caps.DailyQuota,
		UsageUnit:    unit,
		Health:       sourceHealthResponse{Status: "ok"},
	}
}

// applyRuntime decorates a response with the persistent usage/health state.
func (s *sourcesAPI) applyRuntime(resp *sourceResponse, usage store.SourceDailyUsage, health store.SourceHealth) {
	resp.DetailsUsed = usage.DetailsUsed
	if resp.UsageUnit == "details" {
		resp.UsageToday = usage.DetailsUsed
	} else {
		resp.UsageToday = usage.Used
	}
	if health.Status == "" {
		health.Status = "ok"
	}
	resp.Health = sourceHealthResponse{
		Status:              health.Status,
		ConsecutiveFailures: health.ConsecutiveFailures,
		LastError:           health.LastError,
	}
	if health.BackoffUntil != nil {
		resp.Health.BackoffUntil = health.BackoffUntil.Format(time.RFC3339)
	}
	if health.DegradedUntil != nil {
		resp.Health.DegradedUntil = health.DegradedUntil.Format(time.RFC3339)
	}
}

func (s *sourcesAPI) runtimeMaps(ctx context.Context) (map[string]store.SourceDailyUsage, map[string]store.SourceHealth) {
	usage, _ := s.orchestrator.Limiter().UsageMap(ctx)
	health, _ := s.orchestrator.Limiter().HealthMap(ctx)
	if usage == nil {
		usage = map[string]store.SourceDailyUsage{}
	}
	if health == nil {
		health = map[string]store.SourceHealth{}
	}
	return usage, health
}

func (s *sourcesAPI) sourceWithRuntime(ctx context.Context, src store.Source) sourceResponse {
	resp := s.toSourceResponse(src)
	usage, health := s.runtimeMaps(ctx)
	s.applyRuntime(&resp, usage[src.ID], health[src.ID])
	return resp
}

func (s *sourcesAPI) listSources(w http.ResponseWriter, r *http.Request) {
	sources, err := s.repo.ListSources(r.Context())
	if err != nil {
		s.logger.Error("list sources failed", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "falha ao listar fontes"})
		return
	}

	usageMap, healthMap := s.runtimeMaps(r.Context())
	res := make([]sourceResponse, 0, len(sources))
	for _, src := range sources {
		resp := s.toSourceResponse(src)
		s.applyRuntime(&resp, usageMap[src.ID], healthMap[src.ID])
		res = append(res, resp)
	}
	writeJSON(w, http.StatusOK, res)
}

func (s *sourcesAPI) listAdapters(w http.ResponseWriter, r *http.Request) {
	adapters := s.orchestrator.Registry().List()
	writeJSON(w, http.StatusOK, adapters)
}

type createSourceRequest struct {
	ID           string                  `json:"id"`
	Name         string                  `json:"name"`
	Adapter      string                  `json:"adapter"`
	Enabled      *bool                   `json:"enabled"`
	Priority     *int                    `json:"priority"`
	AuthType     string                  `json:"auth_type"`
	APIKey       string                  `json:"api_key"`
	Capabilities *discovery.Capabilities `json:"capabilities"`
}

func (s *sourcesAPI) createSource(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxSettingsBody)
	var req createSourceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_json", "invalid JSON", nil)
		return
	}

	req.ID = strings.ToLower(strings.TrimSpace(req.ID))
	req.Name = strings.TrimSpace(req.Name)
	req.Adapter = strings.ToLower(strings.TrimSpace(req.Adapter))
	req.AuthType = strings.ToLower(strings.TrimSpace(req.AuthType))
	if req.AuthType == "" {
		req.AuthType = "none"
	}

	if req.ID == "" {
		writeAPIError(w, http.StatusBadRequest, "source.id_required", "id is required", nil)
		return
	}
	if req.Name == "" {
		writeAPIError(w, http.StatusBadRequest, "source.name_required", "name is required", nil)
		return
	}
	if err := s.orchestrator.Registry().Validate(req.Adapter); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	adapter, _ := s.orchestrator.Registry().Get(req.Adapter)
	caps := adapter.DefaultCapabilities()
	if req.Capabilities != nil {
		caps = *req.Capabilities
	}
	capsBytes, _ := json.Marshal(caps)

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	priority := 100
	if req.Priority != nil {
		priority = *req.Priority
	}

	var authConfig string
	if req.AuthType == "api_key" {
		req.APIKey = strings.TrimSpace(req.APIKey)
		if req.APIKey != "" {
			sealed, err := store.EncryptString(s.masterKey, req.APIKey)
			if err != nil {
				s.logger.Error("encrypt source api key failed", "err", err)
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "falha ao cifrar credencial"})
				return
			}
			authConfig = sealed
		}
	}

	src := store.Source{
		ID:           req.ID,
		Name:         req.Name,
		Adapter:      req.Adapter,
		Enabled:      enabled,
		Priority:     priority,
		AuthType:     req.AuthType,
		AuthConfig:   authConfig,
		Capabilities: string(capsBytes),
	}

	if err := s.repo.CreateSource(r.Context(), src); err != nil {
		s.logger.Error("create source failed", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "falha ao criar fonte"})
		return
	}

	saved, err := s.repo.GetSource(r.Context(), req.ID)
	if err != nil {
		writeJSON(w, http.StatusCreated, s.sourceWithRuntime(r.Context(), src))
		return
	}
	writeJSON(w, http.StatusCreated, s.sourceWithRuntime(r.Context(), saved))
}

type updateSourceRequest struct {
	Name         *string                 `json:"name"`
	Adapter      *string                 `json:"adapter"`
	Enabled      *bool                   `json:"enabled"`
	Priority     *int                    `json:"priority"`
	AuthType     *string                 `json:"auth_type"`
	APIKey       *string                 `json:"api_key"`
	Capabilities *discovery.Capabilities `json:"capabilities"`
}

func (s *sourcesAPI) updateSource(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	src, err := s.repo.GetSource(r.Context(), id)
	if errors.Is(err, store.ErrNotFound) {
		writeAPIError(w, http.StatusNotFound, "source.not_found", "source not found", map[string]any{"id": id})
		return
	}
	if err != nil {
		s.logger.Error("get source failed", "id", id, "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "falha ao buscar fonte"})
		return
	}

	var req updateSourceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_json", "invalid JSON", nil)
		return
	}

	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name != "" {
			src.Name = name
		}
	}
	if req.Adapter != nil {
		adapterSlug := strings.ToLower(strings.TrimSpace(*req.Adapter))
		if err := s.orchestrator.Registry().Validate(adapterSlug); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		src.Adapter = adapterSlug
	}
	if req.Enabled != nil {
		src.Enabled = *req.Enabled
	}
	if req.Priority != nil {
		src.Priority = *req.Priority
	}
	if req.AuthType != nil {
		src.AuthType = strings.ToLower(strings.TrimSpace(*req.AuthType))
	}
	if req.APIKey != nil {
		apiKey := strings.TrimSpace(*req.APIKey)
		if apiKey != "" {
			sealed, err := store.EncryptString(s.masterKey, apiKey)
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "falha ao cifrar credencial"})
				return
			}
			src.AuthConfig = sealed
		} else if src.AuthType == "none" {
			src.AuthConfig = ""
		}
	}
	if req.Capabilities != nil {
		capsBytes, _ := json.Marshal(*req.Capabilities)
		src.Capabilities = string(capsBytes)
	}

	if err := s.repo.UpdateSource(r.Context(), src); err != nil {
		s.logger.Error("update source failed", "id", id, "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "falha ao atualizar fonte"})
		return
	}

	writeJSON(w, http.StatusOK, s.sourceWithRuntime(r.Context(), src))
}

func (s *sourcesAPI) deleteSource(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := s.repo.DeleteSource(r.Context(), id); errors.Is(err, store.ErrNotFound) {
		writeAPIError(w, http.StatusNotFound, "source.not_found", "source not found", map[string]any{"id": id})
		return
	} else if err != nil {
		s.logger.Error("delete source failed", "id", id, "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "falha ao excluir fonte"})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *sourcesAPI) testSource(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	src, err := s.repo.GetSource(r.Context(), id)
	if errors.Is(err, store.ErrNotFound) {
		writeAPIError(w, http.StatusNotFound, "source.not_found", "source not found", map[string]any{"id": id})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "falha ao buscar fonte"})
		return
	}

	adapter, ok := s.orchestrator.Registry().Get(src.Adapter)
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"ok":    false,
			"error": fmt.Sprintf("adapter %q não está registrado no sistema", src.Adapter),
		})
		return
	}

	apiKey := ""
	if src.AuthType == "api_key" && src.AuthConfig != "" {
		if decrypted, err := store.DecryptString(s.masterKey, src.AuthConfig); err == nil {
			apiKey = decrypted
		}
	}

	var caps discovery.Capabilities
	if err := json.Unmarshal([]byte(src.Capabilities), &caps); err != nil {
		caps = adapter.DefaultCapabilities()
	}
	caps.MaxResultsPerRun = 1 // fetch minimal 1 item for testing
	caps.MaxDetailsPerRun = 0 // never spend the daily detail quota on a test

	// One request, no meaningful quota consumption: details stay disabled and
	// the source token bucket still applies (first request is immediate).
	dailyRemaining := 0
	if caps.QuotaUnit != "details" {
		dailyRemaining = 1
	}
	srcCfg := discovery.SourceConfig{
		ID:             src.ID,
		Name:           src.Name,
		Adapter:        src.Adapter,
		Enabled:        src.Enabled,
		Priority:       src.Priority,
		AuthType:       src.AuthType,
		APIKey:         apiKey,
		Capabilities:   caps,
		RateLimiter:    s.orchestrator.WaiterFor(src.ID, caps),
		DailyRemaining: dailyRemaining,
		Usage:          &discovery.UsageTracker{},
	}

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	start := time.Now()
	jobs, err := adapter.Fetch(ctx, srcCfg, nil)
	latency := time.Since(start).Milliseconds()

	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"ok":         false,
			"latency_ms": latency,
			"error":      err.Error(),
		})
		return
	}

	var sample any
	if len(jobs) > 0 {
		sample = map[string]string{
			"title":   jobs[0].Title,
			"company": jobs[0].Company,
			"url":     jobs[0].URL,
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"ok":         true,
		"latency_ms": latency,
		"sample":     sample,
		"count":      len(jobs),
	})
}
