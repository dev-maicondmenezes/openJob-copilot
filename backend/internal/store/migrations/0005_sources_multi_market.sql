-- 0005_sources_multi_market.sql — CP14: multi-market sources, persistent
-- daily quotas and circuit-breaker/backoff state.
--
-- Scraping sources are first-class citizens: their capabilities declare the
-- defensive pace (rate_limit), the per-run ceiling (max_results_per_run), the
-- optional daily quota and the robots.txt paths the adapter must never use.

CREATE TABLE IF NOT EXISTS source_daily_usage (
  source_id    TEXT NOT NULL,
  day          TEXT NOT NULL, -- local date YYYY-MM-DD
  used         INTEGER NOT NULL DEFAULT 0,
  details_used INTEGER NOT NULL DEFAULT 0,
  updated_at   DATETIME NOT NULL,
  PRIMARY KEY (source_id, day)
);

CREATE TABLE IF NOT EXISTS source_health (
  source_id            TEXT PRIMARY KEY,
  status               TEXT NOT NULL DEFAULT 'ok', -- ok | backoff | degraded | blocked
  consecutive_failures INTEGER NOT NULL DEFAULT 0,
  backoff_until        DATETIME,
  degraded_until       DATETIME,
  last_error           TEXT,
  last_success_at      DATETIME,
  updated_at           DATETIME NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_source_daily_usage_day ON source_daily_usage(day);
CREATE INDEX IF NOT EXISTS idx_source_health_status ON source_health(status);

-- CP14 API sources (zero-auth/public or optional bearer key).
INSERT OR IGNORE INTO sources (id, name, adapter, enabled, priority, auth_type, auth_config, capabilities, created_at, updated_at)
VALUES
(
  'getonboard', 'Get on Board', 'getonboard', 1, 50, 'none', '',
  '{"access_type":"api","supports_keyword_search":true,"supports_pagination":true,"max_results_per_run":100,"rate_limit":{"requests":1,"period":"second","strategy":"token_bucket"},"attribution_required":false,"experimental":false,"recent_window_hours":0}',
  '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z'
),
(
  'trampos', 'Trampos', 'trampos', 1, 51, 'none', '',
  '{"access_type":"api","supports_keyword_search":false,"supports_pagination":true,"max_results_per_run":100,"rate_limit":{"requests":1,"period":"second","strategy":"token_bucket"},"attribution_required":false,"experimental":false,"recent_window_hours":0}',
  '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z'
),
(
  'gupy', 'Gupy', 'gupy', 1, 52, 'none', '',
  '{"access_type":"api","supports_keyword_search":true,"supports_pagination":true,"max_results_per_run":100,"rate_limit":{"requests":1,"period":"second","strategy":"token_bucket"},"attribution_required":false,"experimental":false,"recent_window_hours":0}',
  '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z'
),
(
  'getmanfred', 'getManfred', 'getmanfred', 1, 53, 'none', '',
  '{"access_type":"api","supports_keyword_search":false,"supports_pagination":false,"max_results_per_run":100,"rate_limit":{"requests":1,"period":"second","strategy":"token_bucket"},"attribution_required":false,"experimental":false,"recent_window_hours":0}',
  '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z'
),
(
  'programathor', 'Programathor', 'programathor', 1, 60, 'none', '',
  '{"access_type":"scraping","supports_keyword_search":true,"supports_pagination":true,"max_results_per_run":200,"rate_limit":{"requests":1,"period":"2s","strategy":"token_bucket"},"attribution_required":false,"experimental":false,"recent_window_hours":0}',
  '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z'
),
(
  'computrabajo', 'Computrabajo', 'computrabajo', 1, 61, 'none', '',
  '{"access_type":"scraping","supports_keyword_search":true,"supports_pagination":true,"max_results_per_run":500,"rate_limit":{"requests":1,"period":"second","strategy":"token_bucket"},"attribution_required":false,"experimental":false,"recent_window_hours":0,"robots_disallowed_paths":["by=","pubdate=","sal="]}',
  '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z'
),
(
  'bumeran', 'Bumeran', 'bumeran', 1, 62, 'none', '',
  '{"access_type":"scraping","supports_keyword_search":true,"supports_pagination":true,"max_results_per_run":1200,"rate_limit":{"requests":1,"period":"second","strategy":"token_bucket"},"attribution_required":false,"experimental":false,"recent_window_hours":0,"robots_disallowed_paths":["recientes=","relevantes="]}',
  '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z'
),
(
  'vagas', 'Vagas.com.br', 'vagas', 1, 63, 'none', '',
  '{"access_type":"scraping","supports_keyword_search":true,"supports_pagination":true,"max_results_per_run":200,"rate_limit":{"requests":1,"period":"2s","strategy":"token_bucket"},"attribution_required":false,"experimental":false,"recent_window_hours":0,"requires_daily_quota":true,"daily_quota":20,"quota_unit":"details","max_details_per_run":2,"robots_disallowed_paths":["/vagas/pesquisas"]}',
  '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z'
),
(
  'catho', 'Catho', 'catho', 1, 64, 'none', '',
  '{"access_type":"scraping","supports_keyword_search":true,"supports_pagination":true,"max_results_per_run":300,"rate_limit":{"requests":1,"period":"second","strategy":"token_bucket"},"attribution_required":false,"experimental":false,"recent_window_hours":0}',
  '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z'
),
(
  'infojobs', 'InfoJobs', 'infojobs', 1, 65, 'none', '',
  '{"access_type":"scraping","supports_keyword_search":true,"supports_pagination":true,"max_results_per_run":200,"rate_limit":{"requests":1,"period":"2s","strategy":"token_bucket"},"attribution_required":false,"experimental":false,"recent_window_hours":0,"requires_daily_quota":true,"daily_quota":75,"quota_unit":"requests","robots_disallowed_paths":[]}',
  '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z'
);
