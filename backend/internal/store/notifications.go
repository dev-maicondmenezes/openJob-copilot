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

// ---- notifications (CP10) ------------------------------------------------

const notificationCols = `id, type, severity, title, body,
	COALESCE(i18n_key, ''), COALESCE(i18n_params, ''), COALESCE(link, ''),
	COALESCE(metadata, ''), read_at, created_at`

func scanNotification(scan func(dest ...any) error) (Notification, error) {
	var n Notification
	var read, created any
	if err := scan(&n.ID, &n.Type, &n.Severity, &n.Title, &n.Body, &n.I18nKey, &n.I18nParams, &n.Link, &n.Metadata, &read, &created); err != nil {
		return n, err
	}
	if s, ok := read.(string); ok && s != "" {
		t, _ := parseUTC(s)
		n.ReadAt = &t
	} else if tt, ok := read.(time.Time); ok {
		t := tt.UTC()
		n.ReadAt = &t
	}
	createdStr, _ := readTime(created)
	n.CreatedAt = createdStr
	return n, nil
}

// readTime normalizes the driver's time column values: modernc may return
// either a raw TEXT string (RFC3339Nano or Go's String() layout) or an
// already-converted time.Time.
func readTime(v any) (time.Time, bool) {
	switch t := v.(type) {
	case string:
		if t == "" {
			return time.Time{}, false
		}
		ts, err := parseUTC(t)
		if err != nil {
			return time.Time{}, false
		}
		return ts.UTC(), true
	case time.Time:
		return t.UTC(), true
	default: // nil or unexpected type
		return time.Time{}, false
	}
}

func (s *SQLiteStore) InsertNotification(ctx context.Context, n Notification) (int64, error) {
	if n.Severity == "" {
		n.Severity = "info"
	}
	var link, meta, i18nKey, i18nParams any
	if n.I18nKey != "" {
		i18nKey = n.I18nKey
	}
	if n.I18nParams != "" {
		i18nParams = n.I18nParams
	}
	if n.Link != "" {
		link = n.Link
	}
	if n.Metadata != "" {
		meta = n.Metadata
	}
	var readAt any
	if n.ReadAt != nil {
		readAt = n.ReadAt.UTC().Format(time.RFC3339Nano)
	}
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO notifications (type, severity, title, body, i18n_key, i18n_params, link, metadata, read_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		n.Type, n.Severity, n.Title, n.Body, i18nKey, i18nParams, link, meta, readAt, nowUTC().Format(time.RFC3339Nano))
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *SQLiteStore) GetNotification(ctx context.Context, id int64) (Notification, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT `+notificationCols+` FROM notifications WHERE id = ?`, id)
	n, err := scanNotification(row.Scan)
	if errors.Is(err, sql.ErrNoRows) {
		return n, ErrNotFound
	}
	return n, err
}

// notificationCursor encodes (created_at, id) pagination state in base64 so
// the HTTP layer never needs to understand it.
func encodeNotificationCursor(n Notification) string {
	raw := fmt.Sprintf("%s|%d", n.CreatedAt.UTC().Format(time.RFC3339Nano), n.ID)
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

func decodeNotificationCursor(cur string) (time.Time, int64, error) {
	raw, err := base64.RawURLEncoding.DecodeString(cur)
	if err != nil {
		return time.Time{}, 0, fmt.Errorf("store: invalid cursor: %w", err)
	}
	parts := strings.SplitN(string(raw), "|", 2)
	if len(parts) != 2 {
		return time.Time{}, 0, errors.New("store: invalid cursor")
	}
	ts, err := time.Parse(time.RFC3339Nano, parts[0])
	if err != nil {
		return time.Time{}, 0, fmt.Errorf("store: invalid cursor: %w", err)
	}
	var id int64
	if _, err := fmt.Sscanf(parts[1], "%d", &id); err != nil {
		return time.Time{}, 0, fmt.Errorf("store: invalid cursor: %w", err)
	}
	return ts, id, nil
}

// ListNotifications returns the feed newest-first. Unread entries always sort
// before read ones so the top of the feed is actionable (badge parity), and
// each group is ordered by created_at DESC. Pagination is keyset-based on
// (created_at, id) to stay stable while new entries are inserted.
func (s *SQLiteStore) ListNotifications(ctx context.Context, f NotificationFilter) ([]Notification, string, error) {
	limit := f.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}

	query := `SELECT ` + notificationCols + ` FROM notifications`
	var where []string
	var args []any
	if f.UnreadOnly {
		where = append(where, `read_at IS NULL`)
	}
	if f.Type != "" {
		where = append(where, `type = ?`)
		args = append(args, f.Type)
	}
	if f.Severity != "" {
		where = append(where, `severity = ?`)
		args = append(args, f.Severity)
	}
	if f.Cursor != "" {
		ts, id, err := decodeNotificationCursor(f.Cursor)
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
	args = append(args, limit+1) // one extra row to detect the next page

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()

	items := []Notification{}
	for rows.Next() {
		n, err := scanNotification(rows.Scan)
		if err != nil {
			return nil, "", err
		}
		items = append(items, n)
	}
	if err := rows.Err(); err != nil {
		return nil, "", err
	}

	next := ""
	if len(items) > limit {
		items = items[:limit]
		next = encodeNotificationCursor(items[len(items)-1])
	}
	return items, next, nil
}

// MarkNotificationRead stamps read_at once (idempotent — re-marking keeps the
// original timestamp so the badge count is the only thing that changes).
func (s *SQLiteStore) MarkNotificationRead(ctx context.Context, id int64) error {
	res, err := s.db.ExecContext(ctx, `
		UPDATE notifications SET read_at = COALESCE(read_at, ?) WHERE id = ?`,
		nowUTC().Format(time.RFC3339Nano), id)
	if err != nil {
		return err
	}
	return ensureRow(res)
}

func (s *SQLiteStore) MarkAllNotificationsRead(ctx context.Context) (int64, error) {
	res, err := s.db.ExecContext(ctx, `
		UPDATE notifications SET read_at = ? WHERE read_at IS NULL`,
		nowUTC().Format(time.RFC3339Nano))
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (s *SQLiteStore) DeleteNotification(ctx context.Context, id int64) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM notifications WHERE id = ?`, id)
	if err != nil {
		return err
	}
	return ensureRow(res)
}

// DeleteReadNotifications removes every already-read entry ("limpar lidas").
func (s *SQLiteStore) DeleteReadNotifications(ctx context.Context) (int64, error) {
	res, err := s.db.ExecContext(ctx, `DELETE FROM notifications WHERE read_at IS NOT NULL`)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// DeleteReadNotificationsBefore is the retention housekeeping: read entries
// older than the cutoff are purged; unread ones are always kept (spec CP10).
func (s *SQLiteStore) DeleteReadNotificationsBefore(ctx context.Context, cutoff time.Time) (int64, error) {
	res, err := s.db.ExecContext(ctx, `
		DELETE FROM notifications WHERE read_at IS NOT NULL AND read_at < ?`,
		cutoff.UTC().Format(time.RFC3339))
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (s *SQLiteStore) CountUnreadNotifications(ctx context.Context) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM notifications WHERE read_at IS NULL`).Scan(&count)
	return count, err
}
