-- 0001_init.sql — schema inicial (spec §7).
-- Portabilidade futura para PostgreSQL: apenas TEXT/INTEGER/DATETIME aqui;
-- AUTOINCREMENT fica confinado ao store (nada de SQL vazando para handlers).
-- DATETIME é gravado como UTC RFC3339 (formato gerado pela camada Go), nunca
-- dependendo de CURRENT_TIMESTAMP, para manter o parse determinístico.

CREATE TABLE IF NOT EXISTS settings (
  key        TEXT PRIMARY KEY,
  value      TEXT NOT NULL,
  updated_at DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS curriculum (
  id              INTEGER PRIMARY KEY AUTOINCREMENT,
  content_md      TEXT NOT NULL,
  source_filename TEXT,
  uploaded_at     DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS jobs (
  id          INTEGER PRIMARY KEY AUTOINCREMENT,
  title       TEXT NOT NULL,
  company     TEXT NOT NULL,
  url         TEXT UNIQUE,
  description TEXT NOT NULL,
  source      TEXT NOT NULL,               -- 'remotive' | 'wwr' | 'remoteok' | 'manual'
  score_fit   INTEGER,                     -- 0-100, NULL até ser calculado
  score_reason TEXT,
  status      TEXT NOT NULL DEFAULT 'new', -- new | approved | discarded
  created_at  DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS applications (
  id                    INTEGER PRIMARY KEY AUTOINCREMENT,
  job_id                INTEGER NOT NULL REFERENCES jobs(id),
  tailored_curriculum_md TEXT,
  status                TEXT NOT NULL DEFAULT 'pending', -- pending | sent | interview | rejected | offer
  sent_at               DATETIME,
  updated_at            DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS notify_log (
  id      INTEGER PRIMARY KEY AUTOINCREMENT,
  type    TEXT NOT NULL, -- 'daily_report' | 'status_change' | 'test'
  payload TEXT,
  sent_at DATETIME NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_jobs_status    ON jobs(status);
CREATE INDEX IF NOT EXISTS idx_app_status     ON applications(status);
CREATE INDEX IF NOT EXISTS idx_notify_sent_at ON notify_log(sent_at);
