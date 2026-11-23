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
	// settingLocale é a fonte da verdade do idioma (CP13 §3.1). Themef is
	// per-device (localStorage) e nunca vai para o backend nesta CP.
	settingLocale     = "settings.locale"
	defaultLocale     = "en-US"
	maxAppearanceBody = 1 << 10 // 1 KiB
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

// Preset is a versioned LLM configuration template with a free-tier flag.
// The catalog is versioned so the backend can evolve presets without breaking
// existing client contracts; new versions are added, never removed.
type Preset struct {
	Version    int    `json:"version"`
	BaseURL    string `json:"base_url"`
	Model      string `json:"model"`
	FreeTier   bool   `json:"free_tier"`
	Deprecated bool   `json:"deprecated,omitempty"`
}

// PresetsResponse is the payload for GET /api/settings/llm/presets.
type PresetsResponse struct {
	Presets []Preset `json:"presets"`
	Default Preset   `json:"default"`
}

// registerSettings mounts the settings routes.
func registerSettings(r chi.Router, s *settingsAPI) {
	r.Get("/settings", s.getSettings)
	r.Put("/settings/appearance", s.updateAppearance)
	r.Post("/settings/llm-key", s.saveLLMKey)
	r.Get("/settings/llm-key", s.llmKeyStatus)
	r.Get("/settings/llm", s.llmConfig) // non-secret values for UI hydration
	r.Post("/settings/llm-ping", s.pingLLM)
	r.Get("/settings/llm/presets", s.presetsLLM)
}

// getSettings returns the non-secret, user-facing preferences (CP13 §4.4).
// Theme is intentionally absent: it is stored per-device in localStorage.
func (s *settingsAPI) getSettings(w http.ResponseWriter, r *http.Request) {
	locale, err := s.repo.GetSetting(r.Context(), settingLocale)
	if err != nil && !errors.Is(err, store.ErrNotFound) {
		s.logger.Error("load locale setting failed", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load settings"})
		return
	}
	if !isSupportedLocale(locale) {
		locale = defaultLocale
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"settings": map[string]any{"locale": locale},
		"locale":   locale,
	})
}

type appearanceRequest struct {
	Locale string `json:"locale"`
}

func isSupportedLocale(locale string) bool {
	switch locale {
	case "pt-BR", "en-US", "es-419":
		return true
	default:
		return false
	}
}

// updateAppearance persists the server-side locale preference. Invalid
// locales are rejected instead of silently falling back so the UI can show
// an actionable error.
func (s *settingsAPI) updateAppearance(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxAppearanceBody)
	var req appearanceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_json", "invalid JSON", nil)
		return
	}
	if !isSupportedLocale(req.Locale) {
		writeAPIError(w, http.StatusBadRequest, "settings.invalid_locale", "unsupported locale", nil)
		return
	}
	if err := s.repo.SetSetting(r.Context(), settingLocale, req.Locale); err != nil {
		s.logger.Error("persist locale failed", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to save locale"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"settings": map[string]any{"locale": req.Locale},
		"locale":   req.Locale,
	})
}

// saveLLMKey encrypts and persists the BYOK configuration. Response never
// contains the API key (spec §4: keys are never exposed to the frontend).
func (s *settingsAPI) saveLLMKey(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxSettingsBody)
	var req llmKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_json", "invalid JSON", nil)
		return
	}

	req.APIKey = strings.TrimSpace(req.APIKey)
	if req.APIKey == "" {
		writeAPIError(w, http.StatusBadRequest, "api_key_required", "api_key is required", nil)
		return
	}

	cfg := llm.Config{BaseURL: req.BaseURL, Model: req.Model}.Normalize()
	if !strings.HasPrefix(cfg.BaseURL, "http://") && !strings.HasPrefix(cfg.BaseURL, "https://") {
		writeAPIError(w, http.StatusBadRequest, "base_url_invalid", "base_url must start with http:// or https://", nil)
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

// presetsLLM returns the versioned preset catalog with free-tier flags and a
// default model suggestion. The key never leaves the server; only base_url and
// model are exposed (spec CP4 §3.1).
func (s *settingsAPI) presetsLLM(w http.ResponseWriter, r *http.Request) {
	_ = r.Context()

	presets := []Preset{
		{
			Version:    1,
			BaseURL:    "https://api.openai.com/v1",
			Model:      "gpt-4o-mini",
			FreeTier:   true,
			Deprecated: false,
		},
		{
			Version:    1,
			BaseURL:    "https://api.groq.com/opensai/v1",
			Model:      "llama-3.3-70b-versatile",
			FreeTier:   true,
			Deprecated: false,
		},
		{
			Version:    1,
			BaseURL:    "https://api.together.xyz/v1",
			Model:      "mistral-7b-instruct",
			FreeTier:   false,
			Deprecated: false,
		},
		{
			Version:    1,
			BaseURL:    "https://api.perplexity.ai/v1",
			Model:      "pplx-7b-online",
			FreeTier:   false,
			Deprecated: false,
		},
	}

	// Select first free-tier preset as default
	var defaultPreset Preset
	for i := range presets {
		if presets[i].FreeTier {
			defaultPreset = presets[i]
			break
		}
	}
	// If no free-tier preset found, use first one
	if defaultPreset.Version == 0 {
		defaultPreset = presets[0]
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"presets": presets,
		"default": defaultPreset,
	})
}

// writeJSON is defined in server.go; reused here by the same package.

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
		fe := llm.ClassifyError(err, apiKey)
		writeJSON(w, http.StatusOK, map[string]any{
			"ok":         false,
			"error":      fe.Error(),
			"code":       fe.Code,
			"message":    fe.Message,
			"suggestion": fe.Suggestion,
		})
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