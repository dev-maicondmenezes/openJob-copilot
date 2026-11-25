package api

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/maicondmenezes/openjob/internal/aiusage"
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

// registerSettings mounts the settings routes.
func registerSettings(r chi.Router, s *settingsAPI) {
	r.Get("/settings", s.getSettings)
	r.Put("/settings/appearance", s.updateAppearance)
	r.Post("/settings/llm-key", s.saveLLMKey)
	r.Get("/settings/llm-key", s.llmKeyStatus)
	r.Get("/settings/llm", s.llmConfig) // non-secret values for UI hydration
	r.Post("/settings/llm-ping", s.pingLLM)
	r.Get("/settings/ai", s.getAIUsage)
	r.Put("/settings/ai", s.updateAIUsage)
	registerSettingsProfiles(r, s)
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

// getAIUsage reports whether scoring may call the AI provider.
func (s *settingsAPI) getAIUsage(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, aiusage.Load(r.Context(), s.repo))
}

// updateAIUsage persists the scoring AI opt-in. False keeps the deterministic
// engine, which is the default.
func (s *settingsAPI) updateAIUsage(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxAppearanceBody)
	var req aiusage.Settings
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_json", "invalid JSON", nil)
		return
	}
	if err := aiusage.Save(r.Context(), s.repo, req); err != nil {
		s.logger.Error("persist ai usage failed", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to save AI settings"})
		return
	}
	writeJSON(w, http.StatusOK, req)
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
//
// It is the pre-profiles endpoint and stays supported: besides the flat llm.*
// keys it keeps the active profile in sync, so the two views never diverge.
func (s *settingsAPI) saveLLMKey(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxSettingsBody)
	var req llmKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_json", "invalid JSON", nil)
		return
	}

	cfg := llm.Config{BaseURL: req.BaseURL, Model: req.Model}.Normalize()
	if !validBaseURL(cfg.BaseURL) {
		writeAPIError(w, http.StatusBadRequest, "base_url_invalid", "base_url must start with http:// or https://", nil)
		return
	}
	sealed, ok := s.resolveKey(w, req.APIKey, cfg.BaseURL, "")
	if !ok {
		return
	}

	ctx := r.Context()
	// Fold any pre-profiles flat config into a profile before overwriting it.
	if err := s.repo.MigrateFromFlatConfig(ctx); err != nil {
		s.logger.Error("migrate flat llm config failed", "err", err)
		writeAPIError(w, http.StatusInternalServerError, "settings.save_failed", "failed to save configuration", nil)
		return
	}
	for key, value := range map[string]string{
		settingLLMAPIKey:  sealed,
		settingLLMBaseURL: cfg.BaseURL,
		settingLLMModel:   cfg.Model,
	} {
		if err := s.repo.SetSetting(ctx, key, value); err != nil {
			s.logger.Error("persist llm settings failed", "key", key, "err", err)
			writeAPIError(w, http.StatusInternalServerError, "settings.save_failed", "failed to save configuration", nil)
			return
		}
	}
	if err := s.syncActiveProfile(ctx, cfg, sealed); err != nil {
		s.logger.Error("sync active llm profile failed", "err", err)
		writeAPIError(w, http.StatusInternalServerError, "settings.save_failed", "failed to save configuration", nil)
		return
	}

	// 204: nada a devolver — e nunca a chave.
	w.WriteHeader(http.StatusNoContent)
}

// syncActiveProfile mirrors the flat settings just written into the active
// profile, creating the first profile when none exists yet.
func (s *settingsAPI) syncActiveProfile(ctx context.Context, cfg llm.Config, sealed string) error {
	profiles, err := s.repo.ListProfiles(ctx)
	if err != nil {
		return err
	}
	for _, p := range profiles {
		if !p.IsDefault {
			continue
		}
		old, err := p.DecodeConfig()
		if err != nil {
			return err
		}
		raw, err := store.EncodeProfileConfig(store.ProfileConfig{
			BaseURL: cfg.BaseURL, Model: cfg.Model, SealedKey: sealed, PresetID: old.PresetID,
		})
		if err != nil {
			return err
		}
		return s.repo.UpdateProfile(ctx, p.ID, p.Name, raw)
	}
	id, err := newProfileID()
	if err != nil {
		return err
	}
	raw, err := store.EncodeProfileConfig(store.ProfileConfig{BaseURL: cfg.BaseURL, Model: cfg.Model, SealedKey: sealed})
	if err != nil {
		return err
	}
	if err := s.repo.CreateProfile(ctx, id, cfg.Model, raw, false); err != nil {
		return err
	}
	return s.repo.ActivateProfile(ctx, id)
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

// pingLLM runs a live connectivity/credentials test against the active
// provider config and returns a legible success/failure (spec CP3).
func (s *settingsAPI) pingLLM(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sealed, err := s.repo.GetSetting(ctx, settingLLMAPIKey)
	if err != nil && !errors.Is(err, store.ErrNotFound) {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "falha ao consultar configuração"})
		return
	}
	baseURL, _ := s.repo.GetSetting(ctx, settingLLMBaseURL)
	model, _ := s.repo.GetSetting(ctx, settingLLMModel)
	s.pingSealed(w, r, sealed, baseURL, model)
}

// pingSealed decrypts sealed, pings the provider and writes the result. An
// empty sealed key is a normal, friendly answer ("configure a key"), never a
// 5xx: the settings page must load and be usable before any provider exists.
func (s *settingsAPI) pingSealed(w http.ResponseWriter, r *http.Request, sealed, baseURL, model string) {
	ctx := r.Context()
	if sealed == "" {
		fe := llm.ClassifyError(llm.ErrNotConfigured, "")
		writeJSON(w, http.StatusOK, map[string]any{
			"ok":         false,
			"error":      "Nenhuma chave configurada — salve a API key primeiro.",
			"code":       fe.Code,
			"message":    fe.Message,
			"suggestion": fe.Suggestion,
		})
		return
	}

	apiKey, err := store.DecryptString(s.masterKey, sealed)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"ok":    false,
			"error": "A chave armazenada não pôde ser decifrada (master key mudou?). Reconfigure a API key.",
			"code":  llm.CodeDecryptError,
		})
		return
	}

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
