-- 0002_sources_discovery.sql — Fontes e orquestração de descoberta automática (CP6).

CREATE TABLE IF NOT EXISTS sources (
  id           TEXT PRIMARY KEY,
  name         TEXT NOT NULL,
  adapter      TEXT NOT NULL UNIQUE,
  enabled      INTEGER NOT NULL DEFAULT 1,
  priority     INTEGER NOT NULL DEFAULT 100,
  auth_type    TEXT NOT NULL DEFAULT 'none',
  auth_config  TEXT,
  capabilities TEXT NOT NULL,
  created_at   DATETIME NOT NULL,
  updated_at   DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS discovery_runs (
  id            INTEGER PRIMARY KEY AUTOINCREMENT,
  started_at    DATETIME NOT NULL,
  completed_at  DATETIME,
  status        TEXT NOT NULL, -- 'running' | 'completed' | 'failed'
  stats         TEXT NOT NULL, -- JSON com breakdown de contagens e erros
  error_message TEXT
);

ALTER TABLE jobs ADD COLUMN external_id TEXT;
ALTER TABLE jobs ADD COLUMN matched_keywords TEXT;
ALTER TABLE jobs ADD COLUMN matched_resume_terms TEXT;

CREATE INDEX IF NOT EXISTS idx_sources_enabled ON sources(enabled);
CREATE INDEX IF NOT EXISTS idx_discovery_runs_started ON discovery_runs(started_at);

-- Fontes padrão com APIs gratuitas disponíveis (sem webscraping)
INSERT OR IGNORE INTO sources (id, name, adapter, enabled, priority, auth_type, auth_config, capabilities, created_at, updated_at)
VALUES
(
  'himalayas',
  'Himalayas',
  'himalayas',
  1,
  10,
  'none',
  '',
  '{"supports_keyword_search":true,"supports_pagination":true,"max_results_per_run":50,"rate_limit":{"requests":60,"period":"minute"},"attribution_required":false,"experimental":false,"recent_window_hours":0}',
  '2026-01-01T00:00:00Z',
  '2026-01-01T00:00:00Z'
),
(
  'remotive',
  'Remotive',
  'remotive',
  1,
  20,
  'none',
  '',
  '{"supports_keyword_search":true,"supports_pagination":false,"max_results_per_run":50,"rate_limit":{"requests":4,"period":"day"},"attribution_required":true,"experimental":false,"recent_window_hours":24}',
  '2026-01-01T00:00:00Z',
  '2026-01-01T00:00:00Z'
),
(
  'jobicy',
  'Jobicy',
  'jobicy',
  1,
  30,
  'none',
  '',
  '{"supports_keyword_search":false,"supports_pagination":false,"max_results_per_run":50,"rate_limit":{"requests":60,"period":"minute"},"attribution_required":true,"experimental":false,"recent_window_hours":168}',
  '2026-01-01T00:00:00Z',
  '2026-01-01T00:00:00Z'
),
(
  'remoteok',
  'Remote OK',
  'remoteok',
  1,
  40,
  'none',
  '',
  '{"supports_keyword_search":false,"supports_pagination":false,"max_results_per_run":100,"rate_limit":{"requests":30,"period":"minute"},"attribution_required":true,"experimental":false,"recent_window_hours":24}',
  '2026-01-01T00:00:00Z',
  '2026-01-01T00:00:00Z'
);
