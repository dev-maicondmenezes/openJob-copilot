-- 0008_jobs_fts.sql — global job search index (Fase 5 / B11.3).
-- FTS5 is available in the pure-Go modernc.org/sqlite driver (proved by the
-- compatibility spike). The index stores normalized/stemmed tokens produced
-- by the NLP core, so the query and the documents share the same vocabulary.
-- It is a standalone (contentless-external) table keyed by job_id and can be
-- rebuilt from the jobs table at any time.

CREATE VIRTUAL TABLE IF NOT EXISTS jobs_fts USING fts5(
  job_id UNINDEXED,
  title,
  requirements,
  responsibilities,
  description,
  tokenize = 'unicode61 remove_diacritics 2'
);

-- Vocabulary view used to obtain corpus-level document frequencies for the
-- deterministic BM25 implementation.
CREATE VIRTUAL TABLE IF NOT EXISTS jobs_fts_vocab USING fts5vocab(jobs_fts, row);

CREATE INDEX IF NOT EXISTS idx_jobs_created_at ON jobs(created_at);
