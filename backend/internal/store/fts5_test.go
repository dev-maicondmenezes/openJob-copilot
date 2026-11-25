package store

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/maicondmenezes/openjob/internal/search"
)

// TestFTS5Compatibility is the executable compatibility audit required by
// B11.3: the pure-Go SQLite driver must support FTS5 tables, MATCH, bm25()
// and fts5vocab with the configured tokenizer.
func TestFTS5Compatibility(t *testing.T) {
	repo, err := Open(filepath.Join(t.TempDir(), "openjob.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = repo.Close() }()
	ctx := context.Background()

	var version string
	if err := repo.db.QueryRowContext(ctx, `SELECT sqlite_version()`).Scan(&version); err != nil {
		t.Fatalf("sqlite_version: %v", err)
	}
	if version == "" {
		t.Fatal("empty sqlite version")
	}

	docs := []search.Document{
		{JobID: 1, Title: "backend go", Requirements: "go docker", Description: "postgresql"},
		{JobID: 2, Title: "frontend react", Requirements: "javascript css", Description: "typescript"},
	}
	if err := repo.Rebuild(ctx, docs); err != nil {
		t.Fatalf("rebuild FTS5 index: %v", err)
	}

	hits, err := repo.Search(ctx, "go", search.DefaultWeights(), 10)
	if err != nil {
		t.Fatalf("MATCH/bm25: %v", err)
	}
	if len(hits) != 1 || hits[0].JobID != 1 {
		t.Fatalf("hits = %+v, want job 1", hits)
	}

	df, err := repo.TermDocFreq(ctx, "go")
	if err != nil {
		t.Fatalf("fts5vocab: %v", err)
	}
	if df != 1 {
		t.Fatalf("doc freq = %d, want 1", df)
	}

	stats, err := repo.Stats(ctx)
	if err != nil {
		t.Fatalf("stats: %v", err)
	}
	if stats.Documents != 2 {
		t.Fatalf("documents = %d, want 2", stats.Documents)
	}

	if err := repo.Delete(ctx, 1); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if n, err := repo.Count(ctx); err != nil || n != 1 {
		t.Fatalf("count after delete = %d, %v", n, err)
	}
}
