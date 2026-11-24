package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
)

// Flat settings keys read by tailoring, scoring, e-mail classification and
// discovery. Activating a profile mirrors it into these keys, so every
// consumer picks the change up immediately and no restart is needed.
const (
	flatLLMAPIKey  = "llm.api_key"
	flatLLMBaseURL = "llm.base_url"
	flatLLMModel   = "llm.model"
)

// ProfileConfig is the JSON document kept in llm_profiles.config. The API key
// is never stored in clear: SealedKey is AES-GCM output (store.EncryptString).
type ProfileConfig struct {
	BaseURL   string `json:"base_url"`
	Model     string `json:"model"`
	SealedKey string `json:"sealed_key"`
	PresetID  string `json:"preset_id,omitempty"`
}

// EncodeProfileConfig serialises cfg for Profile.Config.
func EncodeProfileConfig(cfg ProfileConfig) ([]byte, error) { return json.Marshal(cfg) }

// DecodeConfig parses Profile.Config.
func (p Profile) DecodeConfig() (ProfileConfig, error) {
	var cfg ProfileConfig
	if err := json.Unmarshal(p.Config, &cfg); err != nil {
		return ProfileConfig{}, fmt.Errorf("store: decode profile %q: %w", p.ID, err)
	}
	return cfg, nil
}

const profileCols = `id, name, config, is_default, created_at, updated_at`

func (s *SQLiteStore) CreateProfile(ctx context.Context, id, name string, config []byte, isDefault bool) error {
	def := 0
	if isDefault {
		def = 1
	}
	now := nowUTC()
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO llm_profiles (`+profileCols+`)
		VALUES (?, ?, ?, ?, ?, ?)`,
		id, name, config, def, now, now)
	return err
}

func (s *SQLiteStore) GetProfile(ctx context.Context, id string) (Profile, error) {
	var p Profile
	err := s.db.QueryRowContext(ctx, `SELECT `+profileCols+` FROM llm_profiles WHERE id = ?`, id).
		Scan(&p.ID, &p.Name, &p.Config, &p.IsDefault, &p.CreatedAt, &p.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Profile{}, ErrNotFound
	}
	return p, err
}

// ListProfiles returns the active profile first, then by creation order.
func (s *SQLiteStore) ListProfiles(ctx context.Context) ([]Profile, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT `+profileCols+` FROM llm_profiles ORDER BY is_default DESC, created_at, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	profiles := []Profile{}
	for rows.Next() {
		var p Profile
		if err := rows.Scan(&p.ID, &p.Name, &p.Config, &p.IsDefault, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		profiles = append(profiles, p)
	}
	return profiles, rows.Err()
}

func (s *SQLiteStore) UpdateProfile(ctx context.Context, id, name string, config []byte) error {
	res, err := s.db.ExecContext(ctx, `
		UPDATE llm_profiles SET name = ?, config = ?, updated_at = ? WHERE id = ?`,
		name, config, nowUTC(), id)
	if err != nil {
		return err
	}
	return requireAffected(res)
}

func (s *SQLiteStore) DeleteProfile(ctx context.Context, id string) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM llm_profiles WHERE id = ?`, id)
	if err != nil {
		return err
	}
	return requireAffected(res)
}

// ActivateProfile flips the active flag and copies the profile into the flat
// llm.* settings inside one transaction: either both change or neither does.
func (s *SQLiteStore) ActivateProfile(ctx context.Context, id string) error {
	p, err := s.GetProfile(ctx, id)
	if err != nil {
		return err
	}
	cfg, err := p.DecodeConfig()
	if err != nil {
		return err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck // no-op after Commit

	if _, err := tx.ExecContext(ctx, `UPDATE llm_profiles SET is_default = 0 WHERE is_default = 1`); err != nil {
		return err
	}
	res, err := tx.ExecContext(ctx, `UPDATE llm_profiles SET is_default = 1, updated_at = ? WHERE id = ?`, nowUTC(), id)
	if err != nil {
		return err
	}
	if err := requireAffected(res); err != nil {
		return err
	}
	for key, value := range map[string]string{
		flatLLMAPIKey:  cfg.SealedKey,
		flatLLMBaseURL: cfg.BaseURL,
		flatLLMModel:   cfg.Model,
	} {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO settings (key, value, updated_at) VALUES (?, ?, ?)
			ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at`,
			key, value, nowUTC()); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// MigrateFromFlatConfig turns a pre-profiles installation (llm.* keys) into a
// first, active profile. It is idempotent and a no-op when profiles already
// exist or when nothing was ever configured: a fresh install keeps zero
// profiles, which the UI renders as an onboarding empty state.
func (s *SQLiteStore) MigrateFromFlatConfig(ctx context.Context) error {
	var count int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM llm_profiles`).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	sealed, err := s.GetSetting(ctx, flatLLMAPIKey)
	if errors.Is(err, ErrNotFound) || (err == nil && sealed == "") {
		return nil
	}
	if err != nil {
		return err
	}
	baseURL, _ := s.GetSetting(ctx, flatLLMBaseURL)
	model, _ := s.GetSetting(ctx, flatLLMModel)
	raw, err := EncodeProfileConfig(ProfileConfig{BaseURL: baseURL, Model: model, SealedKey: sealed})
	if err != nil {
		return err
	}
	return s.CreateProfile(ctx, "default", "Perfil padrão", raw, true)
}

func requireAffected(res sql.Result) error {
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}
