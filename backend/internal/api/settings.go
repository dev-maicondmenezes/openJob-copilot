package api

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/maicondmenezes/openjob/internal/llm"
	"github.com/maicondmenezes/openjob/internal/store"
)

// Settings keys used by the BYOK flow (spec §3.2). The API key is stored
// encrypted (AES-GCM); base_url and model are non-secret and stored as-is.
const (
	settingLLMAPIKey  = "llm.api_key"
	settingLLMBaseURL = "llm.base_url"
	settingLLMModel   = "llm.model"
)

// maxSettingsBody bounds JSON bodies — BYOK payloads are tiny.
const maxSettingsBody = 1 << 20 // 1 MiB

// settingsAPI serves the /api/settings/* endpoints.
type settingsAPI struct {
	repo      store.Repository
	masterKey []byte
	llm       llm.Client
	logger    *slog.Logger
}

// llmKeyRequest is the payload of POST /api/settings/llm-key. The API key is
// write-only: it is never echoed back in any response.
type llmKeyRequest struct {
	BaseURL string `json:"base_url"`
	Model   string `json:"model"`
	APIKey  string `json:"api_key"`
}

// registerSettings mounts the settings routes.
func registerSettings(r chi.Router, s *settingsAPI) {
	r.Post("/settings/llm-key", s.saveLLMKey)
	r.Get("/settings/llm-key", s.llmKeyStatus)
	r.Get("/settings/llm", s.llmConfig) // non-secret values for UI hydration
	r.Post("/settings/llm-ping", s.pingLLM)
}

// saveLLMKey encrypts and persists the BYOK configuration. Response never
// contains the API key (spec §4: keys are never exposed to the frontend).
func (s *settingsAPI) saveLLMKey(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxSettingsBody)
	var req llmKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "JSON inválido"})
		return
	}

	req.APIKey = strings.TrimSpace(req.APIKey)
	if req.APIKey == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "api_key é obrigatória"})
		return
	}

	cfg := llm.Config{BaseURL: req.BaseURL, Model: req.Model}.Normalize()
	if !strings.HasPrefix(cfg.BaseURL, "http://") && !strings.HasPrefix(cfg.BaseURL, "https://") {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "base_url deve começar com http:// ou https://"})
		return
	}

	sealed, err := store.EncryptString(s.masterKey, req.APIKey)
	if err != nil {
		s.logger.Error("encrypt llm key failed", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "falha ao cifrar a chave"})
		return
	}

	ctx := r.Context()
	for key, value := range map[string]string{
		settingLLMAPIKey:  sealed,
		settingLLMBaseURL: cfg.BaseURL,
		settingLLMModel:   cfg.Model,
	} {
		if err := s.repo.SetSetting(ctx, key, value); err != nil {
			s.logger.Error("persist llm settings failed", "key", key, "err", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "falha ao salvar configuração"})
			return
		}
	}

	// 204: nada a devolver — e nunca a chave.
	w.WriteHeader(http.StatusNoContent)
}

// llmKeyStatus answers GET /api/settings/llm-key with configured: bool only.
func (s *settingsAPI) llmKeyStatus(w http.ResponseWriter, r *http.Request) {
	configured, err := s.hasAPIKey(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "falha ao consultar configuração"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"configured": configured})
}

// llmConfig returns the non-secret LLM settings so the UI can show what is
// already saved without ever touching the key.
func (s *settingsAPI) llmConfig(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	configured, err := s.hasAPIKey(ctx)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "falha ao consultar configuração"})
		return
	}
	baseURL, _ := s.repo.GetSetting(ctx, settingLLMBaseURL)
	model, _ := s.repo.GetSetting(ctx, settingLLMModel)
	if baseURL == "" {
		baseURL = llm.DefaultBaseURL
	}
	if model == "" {
		model = llm.DefaultModel
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"configured": configured,
		"base_url":   baseURL,
		"model":      model,
	})
}

// pingLLM runs a live connectivity/credentials test against the stored
// provider config and returns a legible success/failure (spec CP3).
func (s *settingsAPI) pingLLM(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	sealed, err := s.repo.GetSetting(ctx, settingLLMAPIKey)
	if errors.Is(err, store.ErrNotFound) {
		writeJSON(w, http.StatusOK, map[string]any{
			"ok":    false,
			"error": "Nenhuma chave configurada — salve a API key primeiro.",
		})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "falha ao consultar configuração"})
		return
	}

	apiKey, err := store.DecryptString(s.masterKey, sealed)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"ok":    false,
			"error": "A chave armazenada não pôde ser decifrada (master key mudou?). Reconfigure a API key.",
		})
		return
	}

	baseURL, _ := s.repo.GetSetting(ctx, settingLLMBaseURL)
	model, _ := s.repo.GetSetting(ctx, settingLLMModel)

	if err := s.llm.Ping(ctx, llm.Config{BaseURL: baseURL, APIKey: apiKey, Model: model}); err != nil {
		s.logger.Info("llm ping failed", "err", err)
		writeJSON(w, http.StatusOK, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":      true,
		"message": "Conexão estabelecida com o provedor.",
	})
}

func (s *settingsAPI) hasAPIKey(ctx context.Context) (bool, error) {
	sealed, err := s.repo.GetSetting(ctx, settingLLMAPIKey)
	if errors.Is(err, store.ErrNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return sealed != "", nil
}
