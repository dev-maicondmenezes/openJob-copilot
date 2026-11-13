package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	// Pure-Go SQLite driver, no CGO (docs/adr/0002). Registered as "sqlite".
	_ "modernc.org/sqlite"
)

// nowUTC is the single timestamp source of the store. Everything is stored
// as UTC RFC3339 so reads parse deterministically across drivers.
func nowUTC() time.Time { return time.Now().UTC() }

// Open opens (creating when needed) the SQLite database at path, ensures its
// parent directory exists, applies pending migrations and returns the
// concrete Repository. Pragmas are set via DSN so they are per-connection.
func Open(path string) (*SQLiteStore, error) {
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("store: create db dir %s: %w", dir, err)
		}
	}

	dsn := "file:" + path +
		"?_pragma=busy_timeout(5000)" +
		"&_pragma=journal_mode(WAL)" + // concurrency: readers never block the writer
		"&_pragma=foreign_keys(ON)" +
		"&_pragma=synchronous(NORMAL)"

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("store: open %s: %w", path, err)
	}
	// A single connection serializes all access; SQLite writes are the
	// bottleneck anyway and this sidesteps SQLITE_BUSY entirely for the
	// single-user local-first workload.
	db.SetMaxOpenConns(1)

	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("store: ping %s: %w", path, err)
	}

	if err := migrate(context.Background(), db); err != nil {
		_ = db.Close()
		return nil, err
	}
	return &SQLiteStore{db: db}, nil
}

// SQLiteStore is the Repository implementation on top of modernc.org/sqlite
// (pure Go, no CGO — see docs/adr/0002).
type SQLiteStore struct {
	db *sql.DB
}

func (s *SQLiteStore) Close() error { return s.db.Close() }

// ---- settings -----------------------------------------------------------

func (s *SQLiteStore) GetSetting(ctx context.Context, key string) (string, error) {
	var value string
	err := s.db.QueryRowContext(ctx, `SELECT value FROM settings WHERE key = ?`, key).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrNotFound
	}
	return value, err
}

func (s *SQLiteStore) SetSetting(ctx context.Context, key, value string) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO settings (key, value, updated_at) VALUES (?, ?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at`,
		key, value, nowUTC())
	return err
}

func (s *SQLiteStore) DeleteSetting(ctx context.Context, key string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM settings WHERE key = ?`, key)
	return err
}

// ---- curriculum ---------------------------------------------------------

func (s *SQLiteStore) SaveCurriculum(ctx context.Context, c Curriculum) (int64, error) {
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO curriculum (content_md, source_filename, uploaded_at) VALUES (?, ?, ?)`,
		c.ContentMD, c.SourceFilename, nowUTC())
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *SQLiteStore) LatestCurriculum(ctx context.Context) (Curriculum, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, content_md, COALESCE(source_filename, ''), uploaded_at
		FROM curriculum ORDER BY id DESC LIMIT 1`)
	var c Curriculum
	var uploaded string
	err := row.Scan(&c.ID, &c.ContentMD, &c.SourceFilename, &uploaded)
	if errors.Is(err, sql.ErrNoRows) {
		return c, ErrNotFound
	}
	if err != nil {
		return c, err
	}
	c.UploadedAt, _ = parseUTC(uploaded)
	return c, nil
}

// ---- jobs ---------------------------------------------------------------

func (s *SQLiteStore) CreateJob(ctx context.Context, j Job) (int64, error) {
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO jobs (title, company, url, description, source, status, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		j.Title, j.Company, j.URL, j.Description, j.Source, j.Status, nowUTC())
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

const jobCols = `id, title, company, COALESCE(url, ''), description, source,
	score_fit, COALESCE(score_reason, ''), status, created_at`

func scanJob(scan func(dest ...any) error) (Job, error) {
	var j Job
	var score sql.NullInt64
	var created string
	if err := scan(&j.ID, &j.Title, &j.Company, &j.URL, &j.Description, &j.Source,
		&score, &j.ScoreReason, &j.Status, &created); err != nil {
		return j, err
	}
	if score.Valid {
		v := int(score.Int64)
		j.ScoreFit = &v
	}
	j.CreatedAt, _ = parseUTC(created)
	return j, nil
}

func (s *SQLiteStore) GetJob(ctx context.Context, id int64) (Job, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+jobCols+` FROM jobs WHERE id = ?`, id)
	j, err := scanJob(row.Scan)
	if errors.Is(err, sql.ErrNoRows) {
		return j, ErrNotFound
	}
	return j, err
}

func (s *SQLiteStore) ListJobs(ctx context.Context, status string) ([]Job, error) {
	query := `SELECT ` + jobCols + ` FROM jobs`
	args := []any{}
	if status != "" {
		query += ` WHERE status = ?`
		args = append(args, status)
	}
	query += ` ORDER BY id DESC`

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	jobs := []Job{}
	for rows.Next() {
		j, err := scanJob(rows.Scan)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, j)
	}
	return jobs, rows.Err()
}

func (s *SQLiteStore) UpdateJobStatus(ctx context.Context, id int64, status string) error {
	res, err := s.db.ExecContext(ctx, `UPDATE jobs SET status = ? WHERE id = ?`, status, id)
	if err != nil {
		return err
	}
	return ensureRow(res)
}

func (s *SQLiteStore) SetJobScore(ctx context.Context, id int64, score int, reason string) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE jobs SET score_fit = ?, score_reason = ? WHERE id = ?`, score, reason, id)
	if err != nil {
		return err
	}
	return ensureRow(res)
}

func (s *SQLiteStore) JobURLExists(ctx context.Context, urlStr string) (bool, error) {
	var one int
	err := s.db.QueryRowContext(ctx, `SELECT 1 FROM jobs WHERE url = ? LIMIT 1`, urlStr).Scan(&one)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}

// ---- applications -------------------------------------------------------

func (s *SQLiteStore) CreateApplication(ctx context.Context, a Application) (int64, error) {
	if a.Status == "" {
		a.Status = "pending"
	}
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO applications (job_id, tailored_curriculum_md, status, sent_at, updated_at)
		VALUES (?, ?, ?, ?, ?)`,
		a.JobID, a.TailoredCurriculumMD, a.Status, nullableTime(a.SentAt), nowUTC())
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

const appCols = `id, job_id, COALESCE(tailored_curriculum_md, ''), status, sent_at, updated_at`

func scanApplication(scan func(dest ...any) error) (Application, error) {
	var a Application
	var sent, updated string
	if err := scan(&a.ID, &a.JobID, &a.TailoredCurriculumMD, &a.Status, &sent, &updated); err != nil {
		return a, err
	}
	if sent != "" {
		t, _ := parseUTC(sent)
		a.SentAt = &t
	}
	a.UpdatedAt, _ = parseUTC(updated)
	return a, nil
}

func (s *SQLiteStore) GetApplication(ctx context.Context, id int64) (Application, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+appCols+` FROM applications WHERE id = ?`, id)
	a, err := scanApplication(row.Scan)
	if errors.Is(err, sql.ErrNoRows) {
		return a, ErrNotFound
	}
	return a, err
}

func (s *SQLiteStore) ListApplications(ctx context.Context) ([]Application, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT `+appCols+` FROM applications ORDER BY id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	apps := []Application{}
	for rows.Next() {
		a, err := scanApplication(rows.Scan)
		if err != nil {
			return nil, err
		}
		apps = append(apps, a)
	}
	return apps, rows.Err()
}

// UpdateApplicationStatus changes the pipeline status; the first transition
// to 'sent' stamps sent_at (spec §7).
func (s *SQLiteStore) UpdateApplicationStatus(ctx context.Context, id int64, status string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	var current string
	if err := tx.QueryRowContext(ctx,
		`SELECT status FROM applications WHERE id = ?`, id).Scan(&current); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}

	if status == "sent" && current != "sent" {
		if _, err := tx.ExecContext(ctx, `
			UPDATE applications
			SET status = ?, sent_at = COALESCE(sent_at, ?), updated_at = ?
			WHERE id = ?`, status, nowUTC(), nowUTC(), id); err != nil {
			return err
		}
	} else {
		if _, err := tx.ExecContext(ctx,
			`UPDATE applications SET status = ?, updated_at = ? WHERE id = ?`,
			status, nowUTC(), id); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// ---- notify_log ---------------------------------------------------------

func (s *SQLiteStore) InsertNotifyLog(ctx context.Context, n NotifyLog) (int64, error) {
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO notify_log (type, payload, sent_at) VALUES (?, ?, ?)`,
		n.Type, n.Payload, nowUTC())
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *SQLiteStore) ListNotifyLog(ctx context.Context, limit int) ([]NotifyLog, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, type, COALESCE(payload, ''), sent_at
		FROM notify_log ORDER BY id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	logs := []NotifyLog{}
	for rows.Next() {
		var n NotifyLog
		var sent string
		if err := rows.Scan(&n.ID, &n.Type, &n.Payload, &sent); err != nil {
			return nil, err
		}
		n.SentAt, _ = parseUTC(sent)
		logs = append(logs, n)
	}
	return logs, rows.Err()
}

// ---- helpers ------------------------------------------------------------

func ensureRow(res sql.Result) error {
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func nullableTime(t *time.Time) any {
	if t == nil {
		return nil
	}
	return *t
}

func parseUTC(v string) (time.Time, error) {
	return time.Parse(time.RFC3339, v)
}
