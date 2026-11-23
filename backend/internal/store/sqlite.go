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

	llm "github.com/maicondmenezes/openjob/internal/llm"
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

func (s *SQLiteStore) CreateProfile(ctx context.Context, id, name string, config []byte, isDefault bool) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO llm_profiles (id, name, config, is_default, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		id, name, config, nowUTC(), nowUTC())
	return err
}

func (s *SQLiteStore) GetProfile(ctx context.Context, id string) (profile Profile, err error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, name, config, is_default, created_at, updated_at
		FROM llm_profiles WHERE id = ?`, id)
	var p Profile
	var configRaw []byte
	err = row.Scan(&p.ID, &p.Name, &configRaw, &p.IsDefault, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return p, err
	}
	p.Config = configRaw
	return p, nil
}

func (s *SQLiteStore) ListProfiles(ctx context.Context) ([]Profile, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, config, is_default, created_at, updated_at
		FROM llm_profiles ORDER BY is_default DESC, created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var profiles []Profile
	for rows.Next() {
		var p Profile
		var configRaw []byte
		if err := rows.Scan(&p.ID, &p.Name, &configRaw, &p.IsDefault, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		p.Config = configRaw
		profiles = append(profiles, p)
	}
	return profiles, nil
}

func (s *SQLiteStore) UpdateProfile(ctx context.Context, id, name string, config []byte) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE llm_profiles SET name = ?, config = ?, updated_at = ?
		WHERE id = ?`,
		name, config, nowUTC(), id)
	return err
}

func (s *SQLiteStore) DeleteProfile(ctx context.Context, id string) error {
	// Prevent deleting the last/default profile if it would leave no profiles
	_, err := s.db.ExecContext(ctx, `DELETE FROM llm_profiles WHERE id = ?`, id)
	return err
}

func (s *SQLiteStore) SetDefaultProfile(ctx context.Context, id string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	// Reset all defaults
	_, err = tx.ExecContext(ctx, `UPDATE llm_profiles SET is_default = 0`)
	if err != nil {
		tx.Rollback()
		return err
	}
	// Set the new default
	_, err = tx.ExecContext(ctx, `UPDATE llm_profiles SET is_default = 1 WHERE id = ?`, id)
	if err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}

func (s *SQLiteStore) MigrateFromFlatConfig(ctx context.Context) error {
	// Check if profiles already exist
	var count int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM llm_profiles`).Scan(&count)
	if err != nil {
		return err
	}
	if count > 0 {
		// Already migrated
		return nil
	}
	// Migrate current LLM settings into the first profile
	config := llm.Config{
		BaseURL: llm.DefaultBaseURL,
		Model:   llm.DefaultModel,
	}.Normalize()
	configBytes, err := json.Marshal(config)
	if err != nil {
		return err
	}
	// Create first profile as default
	return s.CreateProfile(ctx, "default", "Perfil padrão", configBytes, true)
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
	if a.StageGroup == "" {
		if a.Status == "sent" {
			a.StageGroup = "in_process"
		} else {
			a.StageGroup = a.Status
		}
	}
	if a.StageSource == "" {
		a.StageSource = "manual"
	}
	now := nowUTC()
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO applications (job_id, tailored_curriculum_md, status, sent_at, updated_at, stage_group, stage_label, stage_updated_at, stage_source)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		a.JobID, a.TailoredCurriculumMD, a.Status, nullableTime(a.SentAt), now, a.StageGroup, a.StageLabel, now, a.StageSource)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

const appCols = `id, job_id, COALESCE(tailored_curriculum_md, ''), status, COALESCE(sent_at, ''), updated_at, COALESCE(stage_group, 'pending'), COALESCE(stage_label, ''), COALESCE(stage_updated_at, updated_at), COALESCE(stage_source, 'manual')`

func scanApplication(scan func(dest ...any) error) (Application, error) {
	var a Application
	var sent, updated, stageUpdated string
	if err := scan(&a.ID, &a.JobID, &a.TailoredCurriculumMD, &a.Status, &sent, &updated, &a.StageGroup, &a.StageLabel, &stageUpdated, &a.StageSource); err != nil {
		return a, err
	}
	if sent != "" {
		t, _ := parseUTC(sent)
		a.SentAt = &t
	}
	a.UpdatedAt, _ = parseUTC(updated)
	if stageUpdated != "" {
		t, _ := parseUTC(stageUpdated)
		a.StageUpdatedAt = t
	} else {
		a.StageUpdatedAt = a.UpdatedAt
	}
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

func (s *SQLiteStore) updateApplicationPipeline(ctx context.Context, id int64, status, stageGroup, stageLabel, stageSource string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	var currentStatus, currentStage string
	var sentAt sql.NullString
	if err := tx.QueryRowContext(ctx,
		`SELECT status, COALESCE(stage_group, 'pending'), COALESCE(sent_at, '') FROM applications WHERE id = ?`, id).
		Scan(&currentStatus, &currentStage, &sentAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}

	now := nowUTC()
	shouldStampSent := (status == "sent" || stageGroup == "in_process") && (!sentAt.Valid || sentAt.String == "")

	var query string
	var args []any
	if shouldStampSent {
		query = `
			UPDATE applications
			SET status = ?, stage_group = ?, stage_label = ?, stage_updated_at = ?, stage_source = ?, sent_at = ?, updated_at = ?
			WHERE id = ?`
		args = []any{status, stageGroup, stageLabel, now, stageSource, now, now, id}
	} else {
		query = `
			UPDATE applications
			SET status = ?, stage_group = ?, stage_label = ?, stage_updated_at = ?, stage_source = ?, updated_at = ?
			WHERE id = ?`
		args = []any{status, stageGroup, stageLabel, now, stageSource, now, id}
	}

	if _, err := tx.ExecContext(ctx, query, args...); err != nil {
		return err
	}
	return tx.Commit()
}

// UpdateApplicationStatus changes the pipeline status; the first transition
// to 'sent' stamps sent_at (spec §7).
func (s *SQLiteStore) UpdateApplicationStatus(ctx context.Context, id int64, status string) error {
	stageGroup := status
	if status == "sent" {
		stageGroup = "in_process"
	}
	return s.updateApplicationPipeline(ctx, id, status, stageGroup, "", "manual")
}

// UpdateApplicationStage changes the pipeline stage_group and stage_label (CP17).
func (s *SQLiteStore) UpdateApplicationStage(ctx context.Context, id int64, stageGroup, stageLabel, stageSource string) error {
	legacyStatus := stageGroup
	if stageGroup == "in_process" {
		legacyStatus = "sent"
	}
	return s.updateApplicationPipeline(ctx, id, legacyStatus, stageGroup, stageLabel, stageSource)
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

// ---- source_daily_usage / source_health (CP14) --------------------------

func (s *SQLiteStore) GetSourceDailyUsage(ctx context.Context, sourceID, day string) (SourceDailyUsage, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT source_id, day, used, details_used, updated_at
		FROM source_daily_usage WHERE source_id = ? AND day = ?`, sourceID, day)
	var u SourceDailyUsage
	var updated string
	if err := row.Scan(&u.SourceID, &u.Day, &u.Used, &u.DetailsUsed, &updated); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return SourceDailyUsage{SourceID: sourceID, Day: day}, ErrNotFound
		}
		return u, err
	}
	u.UpdatedAt, _ = parseUTC(updated)
	return u, nil
}

// AddSourceDailyUsage atomically increments (or creates) the counter for a
// source/day and returns the fresh row. A reset at midnight requires no job:
// the new day simply has no row yet.
func (s *SQLiteStore) AddSourceDailyUsage(ctx context.Context, sourceID, day string, usedDelta, detailsDelta int) (SourceDailyUsage, error) {
	now := nowUTC()
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO source_daily_usage (source_id, day, used, details_used, updated_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(source_id, day) DO UPDATE SET
			used = used + excluded.used,
			details_used = details_used + excluded.details_used,
			updated_at = excluded.updated_at`,
		sourceID, day, usedDelta, detailsDelta, now)
	if err != nil {
		return SourceDailyUsage{}, err
	}
	return s.GetSourceDailyUsage(ctx, sourceID, day)
}

func (s *SQLiteStore) ListSourceDailyUsage(ctx context.Context, day string) ([]SourceDailyUsage, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT source_id, day, used, details_used, updated_at
		FROM source_daily_usage WHERE day = ? ORDER BY source_id`, day)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []SourceDailyUsage
	for rows.Next() {
		var u SourceDailyUsage
		var updated string
		if err := rows.Scan(&u.SourceID, &u.Day, &u.Used, &u.DetailsUsed, &updated); err != nil {
			return nil, err
		}
		u.UpdatedAt, _ = parseUTC(updated)
		out = append(out, u)
	}
	return out, rows.Err()
}

const sourceHealthCols = `source_id, status, consecutive_failures, COALESCE(backoff_until, ''), COALESCE(degraded_until, ''), COALESCE(last_error, ''), COALESCE(last_success_at, ''), updated_at`

func scanSourceHealth(scan func(dest ...any) error) (SourceHealth, error) {
	var h SourceHealth
	var backoff, degraded, lastSuccess, updated string
	if err := scan(&h.SourceID, &h.Status, &h.ConsecutiveFailures, &backoff, &degraded, &h.LastError, &lastSuccess, &updated); err != nil {
		return h, err
	}
	if backoff != "" {
		t, _ := parseUTC(backoff)
		h.BackoffUntil = &t
	}
	if degraded != "" {
		t, _ := parseUTC(degraded)
		h.DegradedUntil = &t
	}
	if lastSuccess != "" {
		t, _ := parseUTC(lastSuccess)
		h.LastSuccessAt = &t
	}
	h.UpdatedAt, _ = parseUTC(updated)
	return h, nil
}

func (s *SQLiteStore) GetSourceHealth(ctx context.Context, sourceID string) (SourceHealth, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+sourceHealthCols+` FROM source_health WHERE source_id = ?`, sourceID)
	h, err := scanSourceHealth(row.Scan)
	if errors.Is(err, sql.ErrNoRows) {
		return SourceHealth{SourceID: sourceID, Status: "ok"}, ErrNotFound
	}
	return h, err
}

func (s *SQLiteStore) UpsertSourceHealth(ctx context.Context, h SourceHealth) error {
	if h.Status == "" {
		h.Status = "ok"
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO source_health (source_id, status, consecutive_failures, backoff_until, degraded_until, last_error, last_success_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(source_id) DO UPDATE SET
			status = excluded.status,
			consecutive_failures = excluded.consecutive_failures,
			backoff_until = excluded.backoff_until,
			degraded_until = excluded.degraded_until,
			last_error = excluded.last_error,
			last_success_at = excluded.last_success_at,
			updated_at = excluded.updated_at`,
		h.SourceID, h.Status, h.ConsecutiveFailures, nullableTime(h.BackoffUntil),
		nullableTime(h.DegradedUntil), h.LastError, nullableTime(h.LastSuccessAt), nowUTC())
	return err
}

func (s *SQLiteStore) ListSourceHealth(ctx context.Context) ([]SourceHealth, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT `+sourceHealthCols+` FROM source_health ORDER BY source_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []SourceHealth
	for rows.Next() {
		h, err := scanSourceHealth(rows.Scan)
		if err != nil {
			return nil, err
		}
		out = append(out, h)
	}
	return out, rows.Err()
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

// ---- email_accounts (CP17) ---------------------------------------------

const emailAccountCols = `id, provider, address, auth_config, folder, COALESCE(last_uid, ''), COALESCE(last_history_id, ''), enabled, COALESCE(last_sync_at, ''), COALESCE(last_error, '')`

func scanEmailAccount(scan func(dest ...any) error) (EmailAccount, error) {
	var a EmailAccount
	var lastSync string
	var enabledInt int
	if err := scan(&a.ID, &a.Provider, &a.Address, &a.AuthConfig, &a.Folder, &a.LastUID, &a.LastHistoryID, &enabledInt, &lastSync, &a.LastError); err != nil {
		return a, err
	}
	a.Enabled = enabledInt != 0
	if lastSync != "" {
		if t, err := parseUTC(lastSync); err == nil {
			a.LastSyncAt = &t
		}
	}
	return a, nil
}

func (s *SQLiteStore) GetEmailAccount(ctx context.Context, id string) (*EmailAccount, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+emailAccountCols+` FROM email_accounts WHERE id = ?`, id)
	a, err := scanEmailAccount(row.Scan)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (s *SQLiteStore) SaveEmailAccount(ctx context.Context, a EmailAccount) error {
	enabledInt := 0
	if a.Enabled {
		enabledInt = 1
	}
	if a.Folder == "" {
		a.Folder = "INBOX"
	}
	if a.AuthConfig == nil {
		a.AuthConfig = []byte{}
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO email_accounts (id, provider, address, auth_config, folder, last_uid, last_history_id, enabled, last_sync_at, last_error)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			provider = excluded.provider,
			address = excluded.address,
			auth_config = CASE WHEN length(excluded.auth_config) > 0 THEN excluded.auth_config ELSE email_accounts.auth_config END,
			folder = excluded.folder,
			enabled = excluded.enabled,
			last_error = excluded.last_error`,
		a.ID, a.Provider, a.Address, a.AuthConfig, a.Folder, a.LastUID, a.LastHistoryID, enabledInt, nullableTime(a.LastSyncAt), a.LastError)
	return err
}

func (s *SQLiteStore) ListEmailAccounts(ctx context.Context) ([]EmailAccount, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT `+emailAccountCols+` FROM email_accounts ORDER BY id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var accounts []EmailAccount
	for rows.Next() {
		a, err := scanEmailAccount(rows.Scan)
		if err != nil {
			return nil, err
		}
		accounts = append(accounts, a)
	}
	return accounts, rows.Err()
}

func (s *SQLiteStore) UpdateEmailAccountCursor(ctx context.Context, id, lastUID, lastHistoryID string, lastSyncAt time.Time, lastError string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE email_accounts
		SET last_uid = CASE WHEN ? != '' THEN ? ELSE last_uid END,
		    last_history_id = CASE WHEN ? != '' THEN ? ELSE last_history_id END,
		    last_sync_at = ?,
		    last_error = ?
		WHERE id = ?`,
		lastUID, lastUID, lastHistoryID, lastHistoryID, lastSyncAt, lastError, id)
	return err
}

// ---- email_messages (CP17) ---------------------------------------------

const emailMessageCols = `id, account_id, message_id, COALESCE(thread_id, ''), from_address, COALESCE(from_name, ''), COALESCE(subject, ''), received_at, COALESCE(body_text, ''), matched_application_id, match_confidence, COALESCE(stage_signal, ''), COALESCE(classified_at, ''), COALESCE(link_source, '')`

func scanEmailMessage(scan func(dest ...any) error) (EmailMessage, error) {
	var m EmailMessage
	var received, classified string
	var appID sql.NullInt64
	if err := scan(&m.ID, &m.AccountID, &m.MessageID, &m.ThreadID, &m.FromAddress, &m.FromName, &m.Subject, &received, &m.BodyText, &appID, &m.MatchConfidence, &m.StageSignal, &classified, &m.LinkSource); err != nil {
		return m, err
	}
	if appID.Valid {
		v := appID.Int64
		m.MatchedApplicationID = &v
	}
	if received != "" {
		t, _ := parseUTC(received)
		m.ReceivedAt = t
	}
	if classified != "" {
		if t, err := parseUTC(classified); err == nil {
			m.ClassifiedAt = &t
		}
	}
	return m, nil
}

func (s *SQLiteStore) SaveEmailMessage(ctx context.Context, msg EmailMessage) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO email_messages (
			id, account_id, message_id, thread_id, from_address, from_name, subject,
			received_at, body_text, matched_application_id, match_confidence, stage_signal, classified_at, link_source
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(message_id) DO UPDATE SET
			thread_id = excluded.thread_id,
			matched_application_id = excluded.matched_application_id,
			match_confidence = excluded.match_confidence,
			stage_signal = excluded.stage_signal,
			classified_at = excluded.classified_at,
			link_source = excluded.link_source`,
		msg.ID, msg.AccountID, msg.MessageID, msg.ThreadID, msg.FromAddress, msg.FromName, msg.Subject,
		msg.ReceivedAt, msg.BodyText, msg.MatchedApplicationID, msg.MatchConfidence, msg.StageSignal,
		nullableTime(msg.ClassifiedAt), msg.LinkSource)
	return err
}

func (s *SQLiteStore) GetEmailMessage(ctx context.Context, id string) (*EmailMessage, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+emailMessageCols+` FROM email_messages WHERE id = ?`, id)
	m, err := scanEmailMessage(row.Scan)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (s *SQLiteStore) GetEmailMessageByMessageID(ctx context.Context, messageID string) (*EmailMessage, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+emailMessageCols+` FROM email_messages WHERE message_id = ?`, messageID)
	m, err := scanEmailMessage(row.Scan)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (s *SQLiteStore) ListEmailMessagesByApplication(ctx context.Context, appID int64) ([]EmailMessage, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT `+emailMessageCols+` FROM email_messages WHERE matched_application_id = ? ORDER BY received_at DESC`, appID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var msgs []EmailMessage
	for rows.Next() {
		m, err := scanEmailMessage(rows.Scan)
		if err != nil {
			return nil, err
		}
		msgs = append(msgs, m)
	}
	return msgs, rows.Err()
}

func (s *SQLiteStore) ListSuggestedEmailMessages(ctx context.Context) ([]EmailMessage, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT `+emailMessageCols+` FROM email_messages WHERE matched_application_id IS NOT NULL AND link_source = 'suggested' ORDER BY received_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var msgs []EmailMessage
	for rows.Next() {
		m, err := scanEmailMessage(rows.Scan)
		if err != nil {
			return nil, err
		}
		msgs = append(msgs, m)
	}
	return msgs, rows.Err()
}

func (s *SQLiteStore) LinkEmailMessage(ctx context.Context, messageID string, appID int64, linkSource string) error {
	res, err := s.db.ExecContext(ctx, `
		UPDATE email_messages
		SET matched_application_id = ?, link_source = ?
		WHERE message_id = ? OR id = ?`, appID, linkSource, messageID, messageID)
	if err != nil {
		return err
	}
	return ensureRow(res)
}

func (s *SQLiteStore) UnlinkEmailMessage(ctx context.Context, messageID string) error {
	res, err := s.db.ExecContext(ctx, `
		UPDATE email_messages
		SET matched_application_id = NULL, link_source = NULL
		WHERE message_id = ? OR id = ?`, messageID, messageID)
	if err != nil {
		return err
	}
	return ensureRow(res)
}

func (s *SQLiteStore) FindApplicationByThreadID(ctx context.Context, threadID string) (*Application, error) {
	if threadID == "" {
		return nil, ErrNotFound
	}
	var appID int64
	err := s.db.QueryRowContext(ctx, `
		SELECT matched_application_id FROM email_messages
		WHERE thread_id = ? AND matched_application_id IS NOT NULL
		LIMIT 1`, threadID).Scan(&appID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	app, err := s.GetApplication(ctx, appID)
	if err != nil {
		return nil, err
	}
	return &app, nil
}

// ---- tags (CP17) -------------------------------------------------------

func (s *SQLiteStore) GetOrCreateTag(ctx context.Context, namespace, value, label string) (Tag, error) {
	var tag Tag
	err := s.db.QueryRowContext(ctx, `
		INSERT INTO tags (namespace, value, label)
		VALUES (?, ?, ?)
		ON CONFLICT(namespace, value) DO UPDATE SET label = excluded.label
		RETURNING id, namespace, value, label`, namespace, value, label).
		Scan(&tag.ID, &tag.Namespace, &tag.Value, &tag.Label)
	return tag, err
}

func (s *SQLiteStore) ListTags(ctx context.Context, namespace string) ([]Tag, error) {
	var rows *sql.Rows
	var err error
	if namespace != "" {
		rows, err = s.db.QueryContext(ctx, `SELECT id, namespace, value, label FROM tags WHERE namespace = ? ORDER BY value ASC`, namespace)
	} else {
		rows, err = s.db.QueryContext(ctx, `SELECT id, namespace, value, label FROM tags ORDER BY namespace ASC, value ASC`)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []Tag
	for rows.Next() {
		var t Tag
		if err := rows.Scan(&t.ID, &t.Namespace, &t.Value, &t.Label); err != nil {
			return nil, err
		}
		tags = append(tags, t)
	}
	return tags, rows.Err()
}

func (s *SQLiteStore) GetApplicationTags(ctx context.Context, appID int64) ([]Tag, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT t.id, t.namespace, t.value, t.label
		FROM tags t
		JOIN application_tags at ON t.id = at.tag_id
		WHERE at.application_id = ?
		ORDER BY t.namespace ASC, t.value ASC`, appID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []Tag
	for rows.Next() {
		var t Tag
		if err := rows.Scan(&t.ID, &t.Namespace, &t.Value, &t.Label); err != nil {
			return nil, err
		}
		tags = append(tags, t)
	}
	return tags, rows.Err()
}

func (s *SQLiteStore) SetApplicationTags(ctx context.Context, appID int64, tagIDs []int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `DELETE FROM application_tags WHERE application_id = ?`, appID); err != nil {
		return err
	}
	for _, tid := range tagIDs {
		if _, err := tx.ExecContext(ctx, `INSERT INTO application_tags (application_id, tag_id) VALUES (?, ?)`, appID, tid); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *SQLiteStore) AddApplicationTag(ctx context.Context, appID int64, tagID int64) error {
	_, err := s.db.ExecContext(ctx, `INSERT OR IGNORE INTO application_tags (application_id, tag_id) VALUES (?, ?)`, appID, tagID)
	return err
}

func (s *SQLiteStore) RemoveApplicationTag(ctx context.Context, appID int64, tagID int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM application_tags WHERE application_id = ? AND tag_id = ?`, appID, tagID)
	return err
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
	if t, err := time.Parse(time.RFC3339, v); err == nil {
		return t, nil
	}
	// modernc binds time.Time values using Go's String() layout; accept it so
	// rows written that way still parse deterministically. Two variants cover
	// named zones ("-03 -03") and numeric fixed zones.
	for _, layout := range []string{
		"2006-01-02 15:04:05.999999999 -0700 MST",
		"2006-01-02 15:04:05.999999999 -0700",
	} {
		if t, err := time.Parse(layout, v); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("store: cannot parse datetime %q", v)
}
