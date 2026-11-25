// Package aiusage stores the AI opt-in preference. Deterministic scoring is the
// default; the AI provider is only used when the user explicitly enables it.
// Keywords and filtering are always deterministic and have no AI toggle.
package aiusage

import (
	"context"
	"encoding/json"

	"github.com/maicondmenezes/openjob/internal/store"
)

// SettingKey is the settings-table key holding the JSON preferences.
const SettingKey = "settings.ai_usage"

// Settings selects whether scoring may call the configured AI provider.
// False means the deterministic engine is used.
type Settings struct {
	Score bool `json:"score"`
}

// Load reads the preference, returning the deterministic default when the
// setting is absent or malformed.
func Load(ctx context.Context, repo store.Repository) Settings {
	raw, err := repo.GetSetting(ctx, SettingKey)
	if err != nil || raw == "" {
		return Settings{}
	}
	var settings Settings
	if err := json.Unmarshal([]byte(raw), &settings); err != nil {
		return Settings{}
	}
	return settings
}

// Save persists the preferences as JSON.
func Save(ctx context.Context, repo store.Repository, settings Settings) error {
	raw, err := json.Marshal(settings)
	if err != nil {
		return err
	}
	return repo.SetSetting(ctx, SettingKey, string(raw))
}
