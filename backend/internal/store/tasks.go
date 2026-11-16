package store

import (
	"context"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"
)

// ---- tasks (CP10) ---------------------------------------------------------
//
// The queue is durable in SQLite (docs/adr/0010). The worker claim is a single
// UPDATE ... RETURNING: SQLite serializes writes (single connection), so even
// a concurrent worker pool can never claim the same row twice.

const taskCols = `id, type, status, COALESCE(payload, '{}'), COALESCE(result, ''),
	COALESCE(error, ''), progress, attempts, max_attempts, priority,
	COALESCE(next_run_at, ''), created_at, COALESCE(started_at, ''), COALESCE(finished_at, '')`

func scanTask(scan func(dest ...any) error) (Task, error) {
	var t Task
	var nextRun, started, finished, created any
	if err := scan(&t.ID, &t.Type, &t.Status, &t.Payload, &t.Result, &t.Error,
		&t.Progress, &t.Attempts, &t.MaxAttempts, &t.Priority,
		&nextRun, &created, &started, &finished); err != nil {
		return t, err
	}
	if ts, ok := readTime(nextRun); ok {
		t.NextRunAt = &ts
	}
	if ts, ok := readTime(started); ok {
		t.StartedAt = &ts
	}
	if ts, ok := readTime(finished); ok {
		t.FinishedAt = &ts
	}
	t.CreatedAt, _ = readTime(created)
	return t, nil
}

// InsertTask enqueues a task; the caller generates the ULID id and is
// responsible for type/payload validity (the tasks package owns that policy).
func (s *SQLiteStore) InsertTask(ctx context.Context, t Task) error {
	if t.ID == "" || t.Type == "" {
		return errors.New("store: task id and type are required")
	}
	if t.Status == "" {
		t.Status = "queued"
	}
	if t.MaxAttempts <= 0 {
		t.MaxAttempts = 3
	}
	if t.Priority == 0 {
		t.Priority = 100
	}
	now := nowUTC().Format(time.RFC3339Nano)
	var nextRun, started, finished any
	if t.NextRunAt != nil {
		nextRun = t.NextRunAt.UTC().Format(time.RFC3339Nano)
	}
	if t.StartedAt != nil {
		started = t.StartedAt.UTC().Format(time.RFC3339Nano)
	}
	if t.FinishedAt != nil {
		finished = t.FinishedAt.UTC().Format(time.RFC3339Nano)
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO tasks (id, type, status, payload, result, error, progress, attempts, max_attempts, priority, next_run_at, created_at, started_at, finished_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		t.ID, t.Type, t.Status, t.Payload, t.Result, t.Error, t.Progress,
		t.Attempts, t.MaxAttempts, t.Priority, nextRun, now, started, finished)
	return err
}

// ClaimNextTask atomically claims the oldest eligible queued task (priority
// first, then FIFO). The single UPDATE serializes concurrent workers: the row
// flips to 'running' and attempts are incremented in the same statement, so a
// task is never handed out twice. ErrNotFound when nothing is eligible.
func (s *SQLiteStore) ClaimNextTask(ctx context.Context) (Task, error) {
	now := nowUTC().Format(time.RFC3339Nano)
	row := s.db.QueryRowContext(ctx, `
		UPDATE tasks SET status = 'running', started_at = ?, attempts = attempts + 1, progress = 0
		WHERE id = (
			SELECT id FROM tasks
			WHERE status = 'queued' AND (next_run_at IS NULL OR next_run_at <= ?)
			ORDER BY priority ASC, created_at ASC, id ASC
			LIMIT 1
		)
		RETURNING `+taskCols, now, now)
	t, err := scanTask(row.Scan)
	if errors.Is(err, sql.ErrNoRows) {
		return t, ErrNotFound
	}
	if err != nil {
		return t, fmt.Errorf("store: claim task: %w", err)
	}
	return t, nil
}

func (s *SQLiteStore) GetTask(ctx context.Context, id string) (Task, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+taskCols+` FROM tasks WHERE id = ?`, id)
	t, err := scanTask(row.Scan)
	if errors.Is(err, sql.ErrNoRows) {
		return t, ErrNotFound
	}
	return t, err
}

func encodeTaskCursor(t Task) string {
	raw := fmt.Sprintf("%s|%s", t.CreatedAt.UTC().Format(time.RFC3339Nano), t.ID)
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

func decodeTaskCursor(cur string) (time.Time, string, error) {
	raw, err := base64.RawURLEncoding.DecodeString(cur)
	if err != nil {
		return time.Time{}, "", fmt.Errorf("store: invalid cursor: %w", err)
	}
	parts := strings.SplitN(string(raw), "|", 2)
	if len(parts) != 2 {
		return time.Time{}, "", errors.New("store: invalid cursor")
	}
	ts, err := time.Parse(time.RFC3339Nano, parts[0])
	if err != nil {
		return time.Time{}, "", fmt.Errorf("store: invalid cursor: %w", err)
	}
	return ts, parts[1], nil
}

// ListTasks returns tasks newest-first with keyset pagination. statuses empty
// = all; typ empty = all types.
func (s *SQLiteStore) ListTasks(ctx context.Context, statuses []string, typ, cursor string, limit int) ([]Task, string, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}

	query := `SELECT ` + taskCols + ` FROM tasks`
	var where []string
	var args []any
	if len(statuses) > 0 {
		ph := make([]string, 0, len(statuses))
		for _, st := range statuses {
			ph = append(ph, "?")
			args = append(args, st)
		}
		where = append(where, `status IN (`+strings.Join(ph, `,`)+`)`)
	}
	if typ != "" {
		where = append(where, `type = ?`)
		args = append(args, typ)
	}
	if cursor != "" {
		ts, id, err := decodeTaskCursor(cursor)
		if err != nil {
			return nil, "", err
		}
		where = append(where, `(CAST(created_at AS TEXT) < ? OR (CAST(created_at AS TEXT) = ? AND id < ?))`)
		args = append(args, ts.UTC().Format(time.RFC3339Nano), ts.UTC().Format(time.RFC3339Nano), id)
	}
	if len(where) > 0 {
		query += ` WHERE ` + strings.Join(where, ` AND `)
	}
	query += ` ORDER BY created_at DESC, id DESC LIMIT ?`
	args = append(args, limit+1)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()

	items := []Task{}
	for rows.Next() {
		t, err := scanTask(rows.Scan)
		if err != nil {
			return nil, "", err
		}
		items = append(items, t)
	}
	if err := rows.Err(); err != nil {
		return nil, "", err
	}

	next := ""
	if len(items) > limit {
		items = items[:limit]
		next = encodeTaskCursor(items[len(items)-1])
	}
	return items, next, nil
}

func (s *SQLiteStore) UpdateTaskProgress(ctx context.Context, id string, progress int) error {
	if progress < 0 {
		progress = 0
	}
	if progress > 100 {
		progress = 100
	}
	res, err := s.db.ExecContext(ctx,
		`UPDATE tasks SET progress = ? WHERE id = ? AND status = 'running'`, progress, id)
	if err != nil {
		return err
	}
	return ensureRow(res)
}

// FinishTask closes out a task with a terminal status (done | failed |
// cancelled). Done tasks report 100% progress.
func (s *SQLiteStore) FinishTask(ctx context.Context, id, status, result, errMsg string) error {
	if status != "done" && status != "failed" && status != "cancelled" {
		return fmt.Errorf("store: invalid terminal task status %q", status)
	}
	progress := 0
	if status == "done" {
		progress = 100
	}
	var res any
	if result != "" {
		res = result
	}
	var errMsgVal any
	if errMsg != "" {
		errMsgVal = errMsg
	}
	_, err := s.db.ExecContext(ctx, `
		UPDATE tasks SET status = ?, result = ?, error = ?, progress = ?, finished_at = ?
		WHERE id = ?`, status, res, errMsgVal, progress, nowUTC().Format(time.RFC3339Nano), id)
	return err
}

// RequeueTaskWithBackoff puts a failed task back in the queue with a delayed
// next_run_at (exponential backoff is computed by the worker). attempts is
// already incremented by the claim, so the natural backoff also protects
// against endless retry loops of a broken handler.
func (s *SQLiteStore) RequeueTaskWithBackoff(ctx context.Context, id string, nextRunAt time.Time, errMsg string) error {
	var errMsgVal any
	if errMsg != "" {
		errMsgVal = errMsg
	}
	res, err := s.db.ExecContext(ctx, `
		UPDATE tasks SET status = 'queued', error = ?, next_run_at = ?,
			started_at = NULL, finished_at = NULL
		WHERE id = ? AND status = 'running'`, errMsgVal, nextRunAt.UTC().Format(time.RFC3339Nano), id)
	if err != nil {
		return err
	}
	return ensureRow(res)
}

// RetryTask is the manual retry endpoint: failed/cancelled tasks go back to
// the queue with attempts and error/result reset.
func (s *SQLiteStore) RetryTask(ctx context.Context, id string) error {
	res, err := s.db.ExecContext(ctx, `
		UPDATE tasks SET status = 'queued', attempts = 0, error = NULL, result = NULL,
			progress = 0, next_run_at = NULL, started_at = NULL, finished_at = NULL
		WHERE id = ? AND status IN ('failed', 'cancelled')`, id)
	if err != nil {
		return err
	}
	return ensureRow(res)
}

// CancelQueuedTask cancels a task that has not started yet. Returns false
// when the task is not in the 'queued' state anymore (the caller then tries
// to cancel a running context).
func (s *SQLiteStore) CancelQueuedTask(ctx context.Context, id, errMsg string) (bool, error) {
	var errMsgVal any
	if errMsg != "" {
		errMsgVal = errMsg
	}
	res, err := s.db.ExecContext(ctx, `
		UPDATE tasks SET status = 'cancelled', error = ?, finished_at = ?
		WHERE id = ? AND status = 'queued'`, errMsgVal, nowUTC().Format(time.RFC3339Nano), id)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n > 0, err
}

// RequeueRunningTasks is the boot-recovery: tasks left in 'running' by a
// previous process (orphaned) return to the queue. attempts was already
// incremented at claim time, so backoff still applies.
func (s *SQLiteStore) RequeueRunningTasks(ctx context.Context) (int64, error) {
	res, err := s.db.ExecContext(ctx, `
		UPDATE tasks SET status = 'queued', started_at = NULL, next_run_at = NULL
		WHERE status = 'running'`)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (s *SQLiteStore) CountTasksByStatus(ctx context.Context) (map[string]int, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT status, COUNT(*) FROM tasks GROUP BY status`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	counts := map[string]int{}
	for rows.Next() {
		var st string
		var n int
		if err := rows.Scan(&st, &n); err != nil {
			return nil, err
		}
		counts[st] = n
	}
	return counts, rows.Err()
}

func (s *SQLiteStore) CountTasks(ctx context.Context, statuses []string, typ string) (int, error) {
	query := `SELECT COUNT(*) FROM tasks`
	var where []string
	var args []any
	if len(statuses) > 0 {
		ph := make([]string, 0, len(statuses))
		for _, st := range statuses {
			ph = append(ph, "?")
			args = append(args, st)
		}
		where = append(where, `status IN (`+strings.Join(ph, `,`)+`)`)
	}
	if typ != "" {
		where = append(where, `type = ?`)
		args = append(args, typ)
	}
	if len(where) > 0 {
		query += ` WHERE ` + strings.Join(where, ` AND `)
	}
	var n int
	err := s.db.QueryRowContext(ctx, query, args...).Scan(&n)
	return n, err
}

func (s *SQLiteStore) CountTasksFailedSince(ctx context.Context, since time.Time) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM tasks
		WHERE status = 'failed' AND finished_at IS NOT NULL AND finished_at >= ?`,
		since.UTC().Format(time.RFC3339)).Scan(&n)
	return n, err
}

// CancelAllQueuedTasks is the bulk "Cancelar todas as queued" action. Tasks
// already claimed by a worker (running) are untouched — the per-task cancel
// endpoint handles those via context propagation.
func (s *SQLiteStore) CancelAllQueuedTasks(ctx context.Context, errMsg string) (int64, error) {
	res, err := s.db.ExecContext(ctx, `
		UPDATE tasks SET status = 'cancelled', error = ?, finished_at = ?
		WHERE status = 'queued'`, errMsg, nowUTC().Format(time.RFC3339Nano))
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// RetryAllFailedTasks is the bulk "Retry todas as failed" action: resets
// attempts and clears the error so the workers pick them up again.
func (s *SQLiteStore) RetryAllFailedTasks(ctx context.Context) (int64, error) {
	res, err := s.db.ExecContext(ctx, `
		UPDATE tasks SET status = 'queued', attempts = 0, error = NULL, result = NULL,
			progress = 0, next_run_at = NULL, started_at = NULL, finished_at = NULL
		WHERE status = 'failed'`)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}
