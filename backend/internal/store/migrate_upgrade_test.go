package store

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
)

// applyMigrationsUpTo bootstraps a database the way an older openJob release
// did: schema_migrations plus every migration up to maxVersion. It returns the
// raw handle so the test can drive repository methods before the upgrade.
func applyMigrationsUpTo(t *testing.T, path string, maxVersion int) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite", "file:"+path)
	if err != nil {
		t.Fatalf("open raw db: %v", err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })

	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version    INTEGER PRIMARY KEY,
			name       TEXT NOT NULL,
			applied_at DATETIME NOT NULL
		)`); err != nil {
		t.Fatalf("create schema_migrations: %v", err)
	}

	migs, err := loadMigrations()
	if err != nil {
		t.Fatalf("load migrations: %v", err)
	}
	applied := 0
	for _, m := range migs {
		if m.version > maxVersion {
			break
		}
		if _, err := db.Exec(m.sql); err != nil {
			t.Fatalf("apply %s: %v", m.name, err)
		}
		if _, err := db.Exec(
			`INSERT INTO schema_migrations (version, name, applied_at) VALUES (?, ?, ?)`,
			m.version, m.name, nowUTC()); err != nil {
			t.Fatalf("record %s: %v", m.name, err)
		}
		applied++
	}
	if applied == 0 {
		t.Fatalf("no migrations applied up to %d", maxVersion)
	}
	return db
}

func TestUpgradeFromPreviousSchemaPreservesData(t *testing.T) {
	path := filepath.Join(t.TempDir(), "openjob.db")

	migs, err := loadMigrations()
	if err != nil {
		t.Fatalf("load migrations: %v", err)
	}
	previous := migs[len(migs)-1].version - 1

	db := applyMigrationsUpTo(t, path, previous)
	ctx := context.Background()
	old := &SQLiteStore{db: db}

	if err := old.SetSetting(ctx, "locale", "pt-BR"); err != nil {
		t.Fatalf("seed setting: %v", err)
	}
	if _, err := old.SaveCurriculum(ctx, Curriculum{ContentMD: "# Currículo antigo", SourceFilename: "cv.md"}); err != nil {
		t.Fatalf("seed curriculum: %v", err)
	}
	jobID, err := old.CreateJob(ctx, Job{
		Title: "Senior Go Developer", Company: "Acme", Description: "Go e SQLite",
		Source: "manual", Status: "new",
	})
	if err != nil {
		t.Fatalf("seed job: %v", err)
	}

	if err := db.Close(); err != nil {
		t.Fatalf("close old db: %v", err)
	}

	repo, err := Open(path)
	if err != nil {
		t.Fatalf("upgrade open: %v", err)
	}
	defer repo.Close()

	locale, err := repo.GetSetting(ctx, "locale")
	if err != nil || locale != "pt-BR" {
		t.Fatalf("setting after upgrade = %q, %v", locale, err)
	}
	cv, err := repo.LatestCurriculum(ctx)
	if err != nil || cv.ContentMD != "# Currículo antigo" {
		t.Fatalf("curriculum after upgrade = %+v, %v", cv, err)
	}
	job, err := repo.GetJob(ctx, jobID)
	if err != nil || job.Title != "Senior Go Developer" {
		t.Fatalf("job after upgrade = %+v, %v", job, err)
	}

	var version int
	if err := repo.db.QueryRow(`SELECT MAX(version) FROM schema_migrations`).Scan(&version); err != nil {
		t.Fatal(err)
	}
	if version != migs[len(migs)-1].version {
		t.Errorf("schema version after upgrade = %d, want %d", version, migs[len(migs)-1].version)
	}

	var fts int
	if err := repo.db.QueryRow(
		`SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = 'jobs_fts'`).Scan(&fts); err != nil {
		t.Fatal(err)
	}
	if fts != 1 {
		t.Error("jobs_fts was not created by the upgrade")
	}
}

func TestFreshDatabaseAppliesEveryMigration(t *testing.T) {
	repo := openTestStore(t)

	migs, err := loadMigrations()
	if err != nil {
		t.Fatalf("load migrations: %v", err)
	}
	var count int
	if err := repo.db.QueryRow(`SELECT COUNT(*) FROM schema_migrations`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != len(migs) {
		t.Errorf("applied migrations = %d, want %d", count, len(migs))
	}

	ctx := context.Background()
	jobID, err := repo.CreateJob(ctx, Job{
		Title: "Backend Engineer", Company: "Beta", Description: "Go",
		Source: "manual", Status: "new",
	})
	if err != nil {
		t.Fatalf("create job on fresh db: %v", err)
	}
	if jobID == 0 {
		t.Fatal("create job returned id 0")
	}
}
