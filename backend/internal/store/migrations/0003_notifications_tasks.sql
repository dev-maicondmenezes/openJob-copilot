-- 0003_notifications_tasks.sql — Checkpoint 10: central de notificações in-app
-- e fila de tarefas de IA assíncronas (docs/plan/10-notificacoes-in-app.md).
-- Aditivo, como todas as migrações (docs/adr: migrações nunca derrubam tabelas).
-- DATETIME continua sendo UTC RFC3339 gerado pela camada Go.

CREATE TABLE IF NOT EXISTS notifications (
  id         INTEGER PRIMARY KEY AUTOINCREMENT,
  type       TEXT NOT NULL,              -- application_status | discovery_result | task_completed | task_failed | source_error | test
  severity   TEXT NOT NULL DEFAULT 'info', -- info | success | warning | error
  title      TEXT NOT NULL,
  body       TEXT NOT NULL DEFAULT '',
  link       TEXT,
  metadata   TEXT,                       -- JSON livre para auditoria
  read_at    DATETIME,
  created_at DATETIME NOT NULL
);

-- Feed do sino (não lidas primeiro) e filtros por tipo na UI.
CREATE INDEX IF NOT EXISTS idx_notifications_feed ON notifications(read_at, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_notifications_type ON notifications(type, created_at DESC);

CREATE TABLE IF NOT EXISTS tasks (
  id           TEXT PRIMARY KEY,         -- ULID: ordenável por tempo
  type         TEXT NOT NULL,            -- score_job | extract_resume_keywords | (tailor_reserve no futuro)
  status       TEXT NOT NULL DEFAULT 'queued', -- queued | running | done | failed | cancelled
  payload      TEXT NOT NULL DEFAULT '{}',
  result       TEXT,                     -- JSON de saída persistido
  error        TEXT,                     -- último erro legível
  progress     INTEGER NOT NULL DEFAULT 0, -- 0-100
  attempts     INTEGER NOT NULL DEFAULT 0,
  max_attempts INTEGER NOT NULL DEFAULT 3,
  priority     INTEGER NOT NULL DEFAULT 100, -- menor = mais prioritário
  next_run_at  DATETIME,                 -- backoff exponencial do retry (NULL = elegível já)
  created_at   DATETIME NOT NULL,
  started_at   DATETIME,
  finished_at  DATETIME
);

-- Claim atômico do worker (UPDATE ... RETURNING) e métricas por tipo/status.
CREATE INDEX IF NOT EXISTS idx_tasks_claim      ON tasks(status, priority, created_at);
CREATE INDEX IF NOT EXISTS idx_tasks_type_status ON tasks(type, status);
