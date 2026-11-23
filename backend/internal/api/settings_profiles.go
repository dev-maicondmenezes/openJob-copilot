package api

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"
	"github.com/maicondmenezes/openjob/internal/llm"
	"github.com/maicondmenezes/openjob/internal/store"
)

const maxProfileNameRunes = 80

// profileDTO is the only shape in which a profile leaves the server. It has
// no key field at all, sealed or not: the key is write-only (spec §4).
type profileDTO struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	PresetID string `json:"preset_id,omitempty"`
	BaseURL  string `json:"base_url"`
	Model    string `json:"model"`
	Active   bool   `json:"active"`
	Local    bool   `json:"local"`
}

// profileRequest is the body of POST/PUT /api/settings/llm/profiles.
type profileRequest struct {
	Name     string `json:"name"`
	PresetID string `json:"preset_id"`
	BaseURL  string `json:"base_url"`
	Model    string `json:"model"`
	// APIKey is optional on update (empty keeps the stored key) and optional
	// on create only for local runtimes.
	APIKey   string `json:"api_key"`
	Activate bool   `json:"activate"`
}

func registerSettingsProfiles(r chi.Router, s *settingsAPI) {
	r.Get("/settings/llm/presets", s.presetsLLM)
	r.Get("/settings/llm/profiles", s.listProfiles)
	r.Post("/settings/llm/profiles", s.createProfile)
	r.Put("/settings/llm/profiles/{id}", s.updateProfile)
	r.Delete("/settings/llm/profiles/{id}", s.deleteProfile)
	r.Post("/settings/llm/profiles/{id}/test", s.testProfile)
	r.Post("/settings/llm/activate", s.activateProfile)
}

// presetsLLM serves the versioned catalog. It depends on nothing but the
// compiled-in list, so it answers the same with or without a provider set up.
func (s *settingsAPI) presetsLLM(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, llm.Presets())
}

func toProfileDTO(p store.Profile) (profileDTO, error) {
	cfg, err := p.DecodeConfig()
	if err != nil {
		return profileDTO{}, err
	}
	norm := llm.Config{BaseURL: cfg.BaseURL, Model: cfg.Model}.Normalize()
	return profileDTO{
		ID: p.ID, Name: p.Name, PresetID: cfg.PresetID,
		BaseURL: norm.BaseURL, Model: norm.Model,
		Active: p.IsDefault, Local: llm.IsLocalBaseURL(norm.BaseURL),
	}, nil
}

// listProfiles always answers 200 with a (possibly empty) array, so a fresh
// install renders the onboarding state instead of an error. Pre-profiles
// installations are migrated lazily here, once.
func (s *settingsAPI) listProfiles(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if err := s.repo.MigrateFromFlatConfig(ctx); err != nil {
		s.logger.Error("migrate flat llm config failed", "err", err)
		writeAPIError(w, http.StatusInternalServerError, "settings.load_failed", "failed to load profiles", nil)
		return
	}
	profiles, err := s.repo.ListProfiles(ctx)
	if err != nil {
		s.logger.Error("list llm profiles failed", "err", err)
		writeAPIError(w, http.StatusInternalServerError, "settings.load_failed", "failed to load profiles", nil)
		return
	}
	out := make([]profileDTO, 0, len(profiles))
	for _, p := range profiles {
		dto, err := toProfileDTO(p)
		if err != nil {
			s.logger.Error("skip unreadable llm profile", "id", p.ID, "err", err)
			continue
		}
		out = append(out, dto)
	}
	writeJSON(w, http.StatusOK, map[string]any{"profiles": out})
}

// resolveKey returns the sealed key to persist, or writes the 400 itself.
func (s *settingsAPI) resolveKey(w http.ResponseWriter, apiKey, baseURL, existingSealed string) (string, bool) {
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		if existingSealed != "" {
			return existingSealed, true
		}
		if !llm.IsLocalBaseURL(baseURL) {
			writeAPIError(w, http.StatusBadRequest, "api_key_required", "api_key is required", nil)
			return "", false
		}
		apiKey = llm.LocalKeyPlaceholder
	}
	sealed, err := store.EncryptString(s.masterKey, apiKey)
	if err != nil {
		s.logger.Error("encrypt llm key failed", "err", err)
		writeAPIError(w, http.StatusInternalServerError, "settings.encrypt_failed", "failed to encrypt the key", nil)
		return "", false
	}
	return sealed, true
}

func validBaseURL(u string) bool {
	return strings.HasPrefix(u, "http://") || strings.HasPrefix(u, "https://")
}

func (s *settingsAPI) readProfileRequest(w http.ResponseWriter, r *http.Request) (profileRequest, bool) {
	r.Body = http.MaxBytesReader(w, r.Body, maxSettingsBody)
	var req profileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_json", "invalid JSON", nil)
		return req, false
	}
	req.Name = strings.TrimSpace(req.Name)
	if utf8.RuneCountInString(req.Name) > maxProfileNameRunes {
		writeAPIError(w, http.StatusBadRequest, "profile_name_invalid", "profile name is too long", nil)
		return req, false
	}
	return req, true
}

func (s *settingsAPI) createProfile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	req, ok := s.readProfileRequest(w, r)
	if !ok {
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

	if err := s.repo.MigrateFromFlatConfig(ctx); err != nil {
		s.logger.Error("migrate flat llm config failed", "err", err)
		writeAPIError(w, http.StatusInternalServerError, "settings.save_failed", "failed to save profile", nil)
		return
	}
	existing, err := s.repo.ListProfiles(ctx)
	if err != nil {
		s.logger.Error("list llm profiles failed", "err", err)
		writeAPIError(w, http.StatusInternalServerError, "settings.save_failed", "failed to save profile", nil)
		return
	}

	name := req.Name
	if name == "" {
		if p, found := llm.PresetByID(req.PresetID); found {
			name = p.Name
		} else {
			name = cfg.Model
		}
	}
	id, err := newProfileID()
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "settings.save_failed", "failed to save profile", nil)
		return
	}
	raw, err := store.EncodeProfileConfig(store.ProfileConfig{
		BaseURL: cfg.BaseURL, Model: cfg.Model, SealedKey: sealed, PresetID: req.PresetID,
	})
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "settings.save_failed", "failed to save profile", nil)
		return
	}
	if err := s.repo.CreateProfile(ctx, id, name, raw, false); err != nil {
		s.logger.Error("create llm profile failed", "err", err)
		writeAPIError(w, http.StatusInternalServerError, "settings.save_failed", "failed to save profile", nil)
		return
	}
	// The very first profile becomes active on its own: there is nothing to
	// switch away from, and tailoring/scoring read the active one.
	if req.Activate || len(existing) == 0 {
		if err := s.repo.ActivateProfile(ctx, id); err != nil {
			s.logger.Error("activate new llm profile failed", "err", err)
			writeAPIError(w, http.StatusInternalServerError, "settings.save_failed", "failed to save profile", nil)
			return
		}
	}
	s.respondProfile(w, r, id, http.StatusCreated)
}

func (s *settingsAPI) updateProfile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := chi.URLParam(r, "id")
	req, ok := s.readProfileRequest(w, r)
	if !ok {
		return
	}
	p, err := s.repo.GetProfile(ctx, id)
	if errors.Is(err, store.ErrNotFound) {
		writeAPIError(w, http.StatusNotFound, "profile_not_found", "profile not found", nil)
		return
	}
	if err != nil {
		s.logger.Error("get llm profile failed", "err", err)
		writeAPIError(w, http.StatusInternalServerError, "settings.save_failed", "failed to save profile", nil)
		return
	}
	cur, err := p.DecodeConfig()
	if err != nil {
		s.logger.Error("decode llm profile failed", "id", id, "err", err)
		writeAPIError(w, http.StatusInternalServerError, "settings.save_failed", "failed to save profile", nil)
		return
	}

	cfg := llm.Config{BaseURL: req.BaseURL, Model: req.Model}.Normalize()
	if !validBaseURL(cfg.BaseURL) {
		writeAPIError(w, http.StatusBadRequest, "base_url_invalid", "base_url must start with http:// or https://", nil)
		return
	}
	sealed, ok := s.resolveKey(w, req.APIKey, cfg.BaseURL, cur.SealedKey)
	if !ok {
		return
	}
	name := req.Name
	if name == "" {
		name = p.Name
	}
	presetID := req.PresetID
	if presetID == "" {
		presetID = cur.PresetID
	}
	raw, err := store.EncodeProfileConfig(store.ProfileConfig{
		BaseURL: cfg.BaseURL, Model: cfg.Model, SealedKey: sealed, PresetID: presetID,
	})
	if err == nil {
		err = s.repo.UpdateProfile(ctx, id, name, raw)
	}
	if err == nil && p.IsDefault {
		// Editing the active profile must take effect immediately.
		err = s.repo.ActivateProfile(ctx, id)
	}
	if err != nil {
		s.logger.Error("update llm profile failed", "id", id, "err", err)
		writeAPIError(w, http.StatusInternalServerError, "settings.save_failed", "failed to save profile", nil)
		return
	}
	s.respondProfile(w, r, id, http.StatusOK)
}

func (s *settingsAPI) deleteProfile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := chi.URLParam(r, "id")
	p, err := s.repo.GetProfile(ctx, id)
	if errors.Is(err, store.ErrNotFound) {
		writeAPIError(w, http.StatusNotFound, "profile_not_found", "profile not found", nil)
		return
	}
	if err != nil {
		s.logger.Error("get llm profile failed", "err", err)
		writeAPIError(w, http.StatusInternalServerError, "settings.save_failed", "failed to delete profile", nil)
		return
	}
	// Deleting the active profile would silently leave tailoring/scoring on a
	// provider the user no longer sees in the list. Make them switch first.
	if p.IsDefault {
		writeAPIError(w, http.StatusConflict, "profile_active_delete",
			"switch to another profile before deleting the active one", nil)
		return
	}
	if err := s.repo.DeleteProfile(ctx, id); err != nil {
		s.logger.Error("delete llm profile failed", "id", id, "err", err)
		writeAPIError(w, http.StatusInternalServerError, "settings.save_failed", "failed to delete profile", nil)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *settingsAPI) activateProfile(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxAppearanceBody)
	var req struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.ID) == "" {
		writeAPIError(w, http.StatusBadRequest, "invalid_json", "invalid JSON", nil)
		return
	}
	err := s.repo.ActivateProfile(r.Context(), req.ID)
	if errors.Is(err, store.ErrNotFound) {
		writeAPIError(w, http.StatusNotFound, "profile_not_found", "profile not found", nil)
		return
	}
	if err != nil {
		s.logger.Error("activate llm profile failed", "id", req.ID, "err", err)
		writeAPIError(w, http.StatusInternalServerError, "settings.save_failed", "failed to activate profile", nil)
		return
	}
	s.respondProfile(w, r, req.ID, http.StatusOK)
}

// testProfile pings one profile without activating it. Like llm-ping it always
// answers 200: a failing provider is a result, not a server error.
func (s *settingsAPI) testProfile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	p, err := s.repo.GetProfile(ctx, chi.URLParam(r, "id"))
	if errors.Is(err, store.ErrNotFound) {
		writeAPIError(w, http.StatusNotFound, "profile_not_found", "profile not found", nil)
		return
	}
	if err != nil {
		s.logger.Error("get llm profile failed", "err", err)
		writeAPIError(w, http.StatusInternalServerError, "settings.load_failed", "failed to load profile", nil)
		return
	}
	cfg, err := p.DecodeConfig()
	if err != nil {
		s.logger.Error("decode llm profile failed", "id", p.ID, "err", err)
		writeAPIError(w, http.StatusInternalServerError, "settings.load_failed", "failed to load profile", nil)
		return
	}
	s.pingSealed(w, r, cfg.SealedKey, cfg.BaseURL, cfg.Model)
}

func (s *settingsAPI) respondProfile(w http.ResponseWriter, r *http.Request, id string, status int) {
	p, err := s.repo.GetProfile(r.Context(), id)
	if err == nil {
		var dto profileDTO
		if dto, err = toProfileDTO(p); err == nil {
			writeJSON(w, status, map[string]any{"profile": dto})
			return
		}
	}
	s.logger.Error("reload llm profile failed", "id", id, "err", err)
	writeAPIError(w, http.StatusInternalServerError, "settings.load_failed", "failed to load profile", nil)
}

func newProfileID() (string, error) {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return "p_" + hex.EncodeToString(b[:]), nil
}
