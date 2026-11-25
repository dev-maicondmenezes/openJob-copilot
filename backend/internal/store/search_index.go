package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/maicondmenezes/openjob/internal/search"
)

// Search-index methods implement search.Index on top of SQLite FTS5. SQL is
// confined to the store package; the ranking engine only sees the interface.

func (s *SQLiteStore) Upsert(ctx context.Context, doc search.Document) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `DELETE FROM jobs_fts WHERE job_id = ?`, doc.JobID); err != nil {
		return fmt.Errorf("store: delete indexed job %d: %w", doc.JobID, err)
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO jobs_fts (job_id, title, requirements, responsibilities, description)
		VALUES (?, ?, ?, ?, ?)`,
		doc.JobID, doc.Title, doc.Requirements, doc.Responsibilities, doc.Description); err != nil {
		return fmt.Errorf("store: index job %d: %w", doc.JobID, err)
	}
	return tx.Commit()
}

func (s *SQLiteStore) Delete(ctx context.Context, jobID int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM jobs_fts WHERE job_id = ?`, jobID)
	return err
}

func (s *SQLiteStore) Rebuild(ctx context.Context, docs []search.Document) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `DELETE FROM jobs_fts`); err != nil {
		return fmt.Errorf("store: clear job index: %w", err)
	}
	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO jobs_fts (job_id, title, requirements, responsibilities, description)
		VALUES (?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()
	for _, doc := range docs {
		if _, err := stmt.ExecContext(ctx, doc.JobID, doc.Title, doc.Requirements, doc.Responsibilities, doc.Description); err != nil {
			return fmt.Errorf("store: index job %d: %w", doc.JobID, err)
		}
	}
	return tx.Commit()
}

func (s *SQLiteStore) Search(ctx context.Context, match string, weights search.Weights, limit int) ([]search.Hit, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT job_id, -bm25(jobs_fts, 0, ?, ?, ?, ?) AS score
		FROM jobs_fts
		WHERE jobs_fts MATCH ?
		ORDER BY score DESC, job_id ASC
		LIMIT ?`,
		weights.Title, weights.Requirements, weights.Responsibilities, weights.Description, match, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	hits := make([]search.Hit, 0, limit)
	for rows.Next() {
		var hit search.Hit
		if err := rows.Scan(&hit.JobID, &hit.RawBM25); err != nil {
			return nil, err
		}
		hits = append(hits, hit)
	}
	return hits, rows.Err()
}

func (s *SQLiteStore) TermDocFreq(ctx context.Context, term string) (int, error) {
	var doc int
	err := s.db.QueryRowContext(ctx, `SELECT doc FROM jobs_fts_vocab WHERE term = ?`, term).Scan(&doc)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	return doc, err
}

func (s *SQLiteStore) Stats(ctx context.Context) (search.Stats, error) {
	var stats search.Stats
	err := s.db.QueryRowContext(ctx, `
		SELECT
			COUNT(*),
			COALESCE(AVG(CASE WHEN title = '' THEN 0 ELSE length(title) - length(replace(title, ' ', '')) + 1 END), 0),
			COALESCE(AVG(CASE WHEN requirements = '' THEN 0 ELSE length(requirements) - length(replace(requirements, ' ', '')) + 1 END), 0),
			COALESCE(AVG(CASE WHEN responsibilities = '' THEN 0 ELSE length(responsibilities) - length(replace(responsibilities, ' ', '')) + 1 END), 0),
			COALESCE(AVG(CASE WHEN description = '' THEN 0 ELSE length(description) - length(replace(description, ' ', '')) + 1 END), 0)
		FROM jobs_fts`).Scan(
		&stats.Documents,
		&stats.AvgTitleLen,
		&stats.AvgRequirementsLen,
		&stats.AvgResponsibilitiesLen,
		&stats.AvgDescriptionLen,
	)
	return stats, err
}

func (s *SQLiteStore) Count(ctx context.Context) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM jobs_fts`).Scan(&n)
	return n, err
}
