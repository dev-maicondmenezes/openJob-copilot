package store

import (
	"context"
	"database/sql"
	"encoding/json"
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
	// url is UNIQUE in the schema; store NULL (not '') for link-less jobs so
	// several manual jobs without a URL never collide on the constraint.
	var url any
	if j.URL != "" {
		url = j.URL
	}
	var extID, kwJSON, resJSON any
	if j.ExternalID != "" {
		extID = j.ExternalID
	}
	if len(j.MatchedKeywords) > 0 {
		if b, err := json.Marshal(j.MatchedKeywords); err == nil {
			kwJSON = string(b)
		}
	}
	if len(j.MatchedResumeTerms) > 0 {
		if b, err := json.Marshal(j.MatchedResumeTerms); err == nil {
			resJSON = string(b)
		}
	}

	res, err := s.db.ExecContext(ctx, `
		INSERT INTO jobs (title, company, url, description, source, status, external_id, matched_keywords, matched_resume_terms, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		j.Title, j.Company, url, j.Description, j.Source, j.Status, extID, kwJSON, resJSON, nowUTC())
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// InsertDiscoveredJob persists a discovered vacancy using INSERT OR IGNORE
// on the UNIQUE url constraint, deduplicating atomically without race conditions.
// Returns (id, true, nil) on insert, or (0, false, nil) if duplicate.
func (s *SQLiteStore) InsertDiscoveredJob(ctx context.Context, j Job) (int64, bool, error) {
	if j.URL == "" {
		return 0, false, errors.New("store: discovered job url cannot be empty")
	}
	var extID, kwJSON, resJSON any
	if j.ExternalID != "" {
		extID = j.ExternalID
	}
	if len(j.MatchedKeywords) > 0 {
		if b, err := json.Marshal(j.MatchedKeywords); err == nil {
			kwJSON = string(b)
		}
	}
	if len(j.MatchedResumeTerms) > 0 {
		if b, err := json.Marshal(j.MatchedResumeTerms); err == nil {
			resJSON = string(b)
		}
	}

	res, err := s.db.ExecContext(ctx, `
		INSERT OR IGNORE INTO jobs (title, company, url, description, source, status, external_id, matched_keywords, matched_resume_terms, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		j.Title, j.Company, j.URL, j.Description, j.Source, j.Status, extID, kwJSON, resJSON, nowUTC())
	if err != nil {
		return 0, false, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return 0, false, err
	}
	if n == 0 {
		return 0, false, nil
	}
	id, err := res.LastInsertId()
	return id, true, err
}

const jobCols = `id, title, company, COALESCE(url, ''), description, source,
	score_fit, COALESCE(score_reason, ''), status, COALESCE(external_id, ''),
	COALESCE(matched_keywords, ''), COALESCE(matched_resume_terms, ''), created_at`

func scanJob(scan func(dest ...any) error) (Job, error) {
	var j Job
	var score sql.NullInt64
	var extID, kwJSON, resJSON, created string
	if err := scan(&j.ID, &j.Title, &j.Company, &j.URL, &j.Description, &j.Source,
		&score, &j.ScoreReason, &j.Status, &extID, &kwJSON, &resJSON, &created); err != nil {
		return j, err
	}
	if score.Valid {
		v := int(score.Int64)
		j.ScoreFit = &v
	}
	j.ExternalID = extID
	if kwJSON != "" {
		_ = json.Unmarshal([]byte(kwJSON), &j.MatchedKeywords)
	}
	if resJSON != "" {
		_ = json.Unmarshal([]byte(resJSON), &j.MatchedResumeTerms)
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

// ActiveApplicationForJob returns the open candidacy (pending, sent,
// interview or offer) linked to a job, or ErrNotFound when none exists —
// the guard against duplicated approvals (CP5).
func (s *SQLiteStore) ActiveApplicationForJob(ctx context.Context, jobID int64) (Application, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT `+appCols+` FROM applications
		WHERE job_id = ? AND status IN ('pending', 'sent', 'interview', 'offer')
		ORDER BY id DESC LIMIT 1`, jobID)
	a, err := scanApplication(row.Scan)
	if errors.Is(err, sql.ErrNoRows) {
		return a, ErrNotFound
	}
	return a, err
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

const appCols = `id, job_id, COALESCE(tailored_curriculum_md, ''), status, COALESCE(sent_at, ''), updated_at`

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

// ---- sources ------------------------------------------------------------

const sourceCols = `id, name, adapter, enabled, priority, auth_type,
	COALESCE(auth_config, ''), capabilities, created_at, updated_at`

func scanSource(scan func(dest ...any) error) (Source, error) {
	var s Source
	var enabled int
	var created, updated string
	if err := scan(&s.ID, &s.Name, &s.Adapter, &enabled, &s.Priority, &s.AuthType,
		&s.AuthConfig, &s.Capabilities, &created, &updated); err != nil {
		return s, err
	}
	s.Enabled = enabled != 0
	s.CreatedAt, _ = parseUTC(created)
	s.UpdatedAt, _ = parseUTC(updated)
	return s, nil
}

func (s *SQLiteStore) ListSources(ctx context.Context) ([]Source, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT `+sourceCols+` FROM sources ORDER BY priority ASC, id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sources []Source
	for rows.Next() {
		src, err := scanSource(rows.Scan)
		if err != nil {
			return nil, err
		}
		sources = append(sources, src)
	}
	return sources, rows.Err()
}

func (s *SQLiteStore) GetSource(ctx context.Context, id string) (Source, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+sourceCols+` FROM sources WHERE id = ?`, id)
	src, err := scanSource(row.Scan)
	if errors.Is(err, sql.ErrNoRows) {
		return src, ErrNotFound
	}
	return src, err
}

func (s *SQLiteStore) CreateSource(ctx context.Context, src Source) error {
	enabled := 0
	if src.Enabled {
		enabled = 1
	}
	now := nowUTC()
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO sources (id, name, adapter, enabled, priority, auth_type, auth_config, capabilities, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		src.ID, src.Name, src.Adapter, enabled, src.Priority, src.AuthType, src.AuthConfig, src.Capabilities, now, now)
	return err
}

func (s *SQLiteStore) UpdateSource(ctx context.Context, src Source) error {
	enabled := 0
	if src.Enabled {
		enabled = 1
	}
	res, err := s.db.ExecContext(ctx, `
		UPDATE sources
		SET name = ?, adapter = ?, enabled = ?, priority = ?, auth_type = ?, auth_config = ?, capabilities = ?, updated_at = ?
		WHERE id = ?`,
		src.Name, src.Adapter, enabled, src.Priority, src.AuthType, src.AuthConfig, src.Capabilities, nowUTC(), src.ID)
	if err != nil {
		return err
	}
	return ensureRow(res)
}

func (s *SQLiteStore) DeleteSource(ctx context.Context, id string) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM sources WHERE id = ?`, id)
	if err != nil {
		return err
	}
	return ensureRow(res)
}

// ---- discovery_runs -----------------------------------------------------

const discoveryRunCols = `id, started_at, COALESCE(completed_at, ''), status, stats, COALESCE(error_message, '')`

func scanDiscoveryRun(scan func(dest ...any) error) (DiscoveryRun, error) {
	var r DiscoveryRun
	var started, completed string
	if err := scan(&r.ID, &started, &completed, &r.Status, &r.Stats, &r.ErrorMessage); err != nil {
		return r, err
	}
	r.StartedAt, _ = parseUTC(started)
	if completed != "" {
		t, _ := parseUTC(completed)
		r.CompletedAt = &t
	}
	return r, nil
}

func (s *SQLiteStore) CreateDiscoveryRun(ctx context.Context, r DiscoveryRun) (int64, error) {
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO discovery_runs (started_at, completed_at, status, stats, error_message)
		VALUES (?, ?, ?, ?, ?)`,
		nowUTC(), nullableTime(r.CompletedAt), r.Status, r.Stats, r.ErrorMessage)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *SQLiteStore) UpdateDiscoveryRun(ctx context.Context, r DiscoveryRun) error {
	res, err := s.db.ExecContext(ctx, `
		UPDATE discovery_runs
		SET completed_at = ?, status = ?, stats = ?, error_message = ?
		WHERE id = ?`,
		nullableTime(r.CompletedAt), r.Status, r.Stats, r.ErrorMessage, r.ID)
	if err != nil {
		return err
	}
	return ensureRow(res)
}

func (s *SQLiteStore) GetDiscoveryRun(ctx context.Context, id int64) (DiscoveryRun, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+discoveryRunCols+` FROM discovery_runs WHERE id = ?`, id)
	r, err := scanDiscoveryRun(row.Scan)
	if errors.Is(err, sql.ErrNoRows) {
		return r, ErrNotFound
	}
	return r, err
}

func (s *SQLiteStore) LatestDiscoveryRun(ctx context.Context) (DiscoveryRun, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+discoveryRunCols+` FROM discovery_runs ORDER BY id DESC LIMIT 1`)
	r, err := scanDiscoveryRun(row.Scan)
	if errors.Is(err, sql.ErrNoRows) {
		return r, ErrNotFound
	}
	return r, err
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
