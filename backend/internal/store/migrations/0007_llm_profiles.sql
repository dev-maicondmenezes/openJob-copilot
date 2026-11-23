-- Checkpoint 4: Perfis LLM múltiplos (B2)

CREATE TABLE IF NOT EXISTS llm_profiles (
    id              TEXT PRIMARY KEY,
    name            TEXT NOT NULL,
    config          BLOB NOT NULL,
    is_default      INTEGER NOT NULL DEFAULT 0,
    created_at      DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at      DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);

-- Migrate existing flat LLM config into the first profile if no profiles exist
-- This runs after the migration; the Go code should check if any profiles exist
-- and if not, create one from the current llm.* settings.

CREATE UNIQUE INDEX IF NOT EXISTS idx_llm_profiles_default ON llm_profiles(is_default) WHERE is_default = 1;