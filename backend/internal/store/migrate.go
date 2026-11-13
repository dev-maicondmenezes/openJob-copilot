package store

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strconv"
	"strings"
)

// Versioned migrations are embedded in the binary (spec §10.3): a fresh
// container bootstraps its own schema, and upgrades are additive — never
// "drop table". Files are named NNNN_label.sql; the numeric prefix is the
// version recorded in schema_migrations.
//
//go:embed migrations/*.sql
var migrationsFS embed.FS

type migration struct {
	version int
	name    string
	sql     string
}

func loadMigrations() ([]migration, error) {
	entries, err := fs.Glob(migrationsFS, "migrations/*.sql")
	if err != nil {
		return nil, fmt.Errorf("store: glob migrations: %w", err)
	}
	sort.Strings(entries)

	migs := make([]migration, 0, len(entries))
	for _, entry := range entries {
		data, err := migrationsFS.ReadFile(entry)
		if err != nil {
			return nil, fmt.Errorf("store: read %s: %w", entry, err)
		}
		base := strings.TrimSuffix(path.Base(entry), ".sql")
		versionStr, _, _ := strings.Cut(base, "_")
		version, err := strconv.Atoi(versionStr)
		if err != nil {
			return nil, fmt.Errorf("store: migration %s: bad version prefix: %w", entry, err)
		}
		migs = append(migs, migration{version: version, name: base, sql: string(data)})
	}
	return migs, nil
}

// migrate applies every migration newer than the recorded schema version,
// each inside its own transaction together with the version bump.
func migrate(ctx context.Context, db *sql.DB) error {
	if _, err := db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version    INTEGER PRIMARY KEY,
			name       TEXT NOT NULL,
			applied_at DATETIME NOT NULL
		)`); err != nil {
		return fmt.Errorf("store: create schema_migrations: %w", err)
	}

	var current int
	if err := db.QueryRowContext(ctx,
		`SELECT COALESCE(MAX(version), 0) FROM schema_migrations`).Scan(&current); err != nil {
		return fmt.Errorf("store: read schema version: %w", err)
	}

	migs, err := loadMigrations()
	if err != nil {
		return err
	}

	for _, m := range migs {
		if m.version <= current {
			continue
		}
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("store: begin migration %d: %w", m.version, err)
		}
		if _, err := tx.ExecContext(ctx, m.sql); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("store: apply migration %s: %w", m.name, err)
		}
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO schema_migrations (version, name, applied_at) VALUES (?, ?, ?)`,
			m.version, m.name, nowUTC()); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("store: record migration %s: %w", m.name, err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("store: commit migration %s: %w", m.name, err)
		}
	}
	return nil
}
