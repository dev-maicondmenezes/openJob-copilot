package store

import (
	"context"
	"crypto/rand"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// osrand fills b with cryptographically random bytes (test helper).
func osrand(b []byte) (int, error) { return rand.Read(b) }

func openTestStore(t *testing.T) *SQLiteStore {
	t.Helper()
	repo, err := Open(filepath.Join(t.TempDir(), "openjob.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = repo.Close() })
	return repo
}

func TestOpenIsIdempotent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "openjob.db")

	first, err := Open(path)
	if err != nil {
		t.Fatalf("first open: %v", err)
	}
	if err := first.Close(); err != nil {
		t.Fatal(err)
	}

	// Reopen: migrations must not re-apply or fail (WAL side files exist).
	second, err := Open(path)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer second.Close()

	var count int
	if err := second.db.QueryRow(`SELECT COUNT(*) FROM schema_migrations`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	migs, err := loadMigrations()
	if err != nil {
		t.Fatal(err)
	}
	if count != len(migs) {
		t.Errorf("applied migrations = %d, want %d", count, len(migs))
	}
}

func TestSettingsRoundtrip(t *testing.T) {
	repo := openTestStore(t)
	ctx := context.Background()

	if _, err := repo.GetSetting(ctx, "missing"); !errors.Is(err, ErrNotFound) {
		t.Errorf("missing key err = %v, want ErrNotFound", err)
	}

	if err := repo.SetSetting(ctx, "llm.base_url", "https://api.openai.com/v1"); err != nil {
		t.Fatal(err)
	}
	if err := repo.SetSetting(ctx, "llm.base_url", "https://api.groq.com/openai/v1"); err != nil {
		t.Fatal(err)
	}
	got, err := repo.GetSetting(ctx, "llm.base_url")
	if err != nil {
		t.Fatal(err)
	}
	if got != "https://api.groq.com/openai/v1" {
		t.Errorf("value = %q, want upserted value", got)
	}

	if err := repo.DeleteSetting(ctx, "llm.base_url"); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.GetSetting(ctx, "llm.base_url"); !errors.Is(err, ErrNotFound) {
		t.Errorf("after delete err = %v, want ErrNotFound", err)
	}
}

func TestJobsAndApplications(t *testing.T) {
	repo := openTestStore(t)
	ctx := context.Background()

	id, err := repo.CreateJob(ctx, Job{
		Title: "Go Dev", Company: "Acme", URL: "https://jobs.example/1",
		Description: "desc", Source: "manual", Status: "new",
	})
	if err != nil {
		t.Fatal(err)
	}

	exists, err := repo.JobURLExists(ctx, "https://jobs.example/1")
	if err != nil || !exists {
		t.Errorf("JobURLExists = %v, %v; want true, nil", exists, err)
	}
	if exists, _ := repo.JobURLExists(ctx, "https://jobs.example/none"); exists {
		t.Error("JobURLExists(none) = true, want false")
	}

	if err := repo.SetJobScore(ctx, id, 87, "forte match Go/cloud"); err != nil {
		t.Fatal(err)
	}
	job, err := repo.GetJob(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if job.ScoreFit == nil || *job.ScoreFit != 87 {
		t.Errorf("ScoreFit = %v, want 87", job.ScoreFit)
	}
	if job.ScoreReason != "forte match Go/cloud" {
		t.Errorf("ScoreReason = %q", job.ScoreReason)
	}

	if err := repo.UpdateJobStatus(ctx, id, "approved"); err != nil {
		t.Fatal(err)
	}
	if err := repo.UpdateJobStatus(ctx, 999, "approved"); !errors.Is(err, ErrNotFound) {
		t.Errorf("UpdateJobStatus(999) err = %v, want ErrNotFound", err)
	}

	appID, err := repo.CreateApplication(ctx, Application{JobID: id})
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.UpdateApplicationStatus(ctx, appID, "sent"); err != nil {
		t.Fatal(err)
	}
	app, err := repo.GetApplication(ctx, appID)
	if err != nil {
		t.Fatal(err)
	}
	if app.Status != "sent" || app.SentAt == nil {
		t.Errorf("status = %q sent_at = %v; want sent + stamped", app.Status, app.SentAt)
	}
}

func TestCurriculumAndNotifyLog(t *testing.T) {
	repo := openTestStore(t)
	ctx := context.Background()

	if _, err := repo.LatestCurriculum(ctx); !errors.Is(err, ErrNotFound) {
		t.Errorf("empty curriculum err = %v, want ErrNotFound", err)
	}

	if _, err := repo.SaveCurriculum(ctx, Curriculum{ContentMD: "# CV", SourceFilename: "cv.pdf"}); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.SaveCurriculum(ctx, Curriculum{ContentMD: "# CV v2"}); err != nil {
		t.Fatal(err)
	}
	cv, err := repo.LatestCurriculum(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if cv.ContentMD != "# CV v2" {
		t.Errorf("latest = %q, want '# CV v2'", cv.ContentMD)
	}

	if _, err := repo.InsertNotifyLog(ctx, NotifyLog{Type: "test", Payload: `{"a":1}`}); err != nil {
		t.Fatal(err)
	}
	logs, err := repo.ListNotifyLog(ctx, 10)
	if err != nil || len(logs) != 1 {
		t.Fatalf("ListNotifyLog = %v, %v", logs, err)
	}
	if logs[0].Type != "test" {
		t.Errorf("type = %q, want test", logs[0].Type)
	}
}

// ---- crypto (critical tests — spec §10.5) --------------------------------

func TestMasterKeyCreatedWith0600(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".master.key")

	key, err := LoadOrCreateMasterKey(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(key) != 32 {
		t.Fatalf("key length = %d, want 32", len(key))
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("file mode = %o, want 600", perm)
	}

	// Second load returns the same key.
	again, err := LoadOrCreateMasterKey(path)
	if err != nil || string(again) != string(key) {
		t.Errorf("reload changed key (equal=%v, err=%v)", string(again) == string(key), err)
	}
}

func TestEncryptDecryptRoundtrip(t *testing.T) {
	key := make([]byte, 32)
	if _, err := osrand(key); err != nil {
		t.Fatal(err)
	}

	secret := "sk-personal-api-key-😀 with spaces & unicode"
	sealed, err := EncryptString(key, secret)
	if err != nil {
		t.Fatal(err)
	}

	// Ciphertext must not contain the plaintext.
	if sealed == secret || len(sealed) < 24 {
		t.Fatalf("ciphertext looks like plaintext: %q", sealed)
	}

	opened, err := DecryptString(key, sealed)
	if err != nil {
		t.Fatal(err)
	}
	if opened != secret {
		t.Errorf("roundtrip = %q, want %q", opened, secret)
	}

	// Distinct calls produce distinct ciphertexts (random nonce).
	sealed2, _ := EncryptString(key, secret)
	if sealed2 == sealed {
		t.Error("two encryptions of the same plaintext produced identical ciphertext")
	}
}

func TestDecryptWithWrongKeyFailsExplicitly(t *testing.T) {
	keyA := make([]byte, 32)
	keyB := make([]byte, 32)
	if _, err := osrand(keyA); err != nil {
		t.Fatal(err)
	}
	if _, err := osrand(keyB); err != nil {
		t.Fatal(err)
	}

	sealed, err := EncryptString(keyA, "sk-secret")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := DecryptString(keyB, sealed); !errors.Is(err, ErrKeyMismatch) {
		t.Errorf("wrong key err = %v, want ErrKeyMismatch", err)
	}
}

// Documented decision (docs/adr/0004): a corrupt/mis-sized master key file is
// an explicit hard error — never a silent regeneration, which would orphan
// every stored ciphertext.
func TestMasterKeyWrongSizeFailsLoudly(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".master.key")
	if err := os.WriteFile(path, []byte("too short"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadOrCreateMasterKey(path); err == nil {
		t.Error("short key file accepted, want explicit error")
	}
}

func TestSourcesCRUD(t *testing.T) {
	repo := openTestStore(t)
	ctx := context.Background()

	// Default sources seeded by migration 0002
	sources, err := repo.ListSources(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(sources) < 4 {
		t.Fatalf("expected at least 4 seeded sources, got %d", len(sources))
	}

	// Create custom source
	s := Source{
		ID:           "custom-src",
		Name:         "Custom Source",
		Adapter:      "custom_adapter",
		Enabled:      true,
		Priority:     50,
		AuthType:     "none",
		Capabilities: `{"supports_keyword_search":true}`,
	}
	if err := repo.CreateSource(ctx, s); err != nil {
		t.Fatalf("create source: %v", err)
	}

	got, err := repo.GetSource(ctx, "custom-src")
	if err != nil {
		t.Fatalf("get source: %v", err)
	}
	if got.Name != "Custom Source" || !got.Enabled || got.Priority != 50 {
		t.Errorf("got unexpected source: %+v", got)
	}

	// Update source
	got.Name = "Updated Source"
	got.Enabled = false
	got.Priority = 80
	if err := repo.UpdateSource(ctx, got); err != nil {
		t.Fatalf("update source: %v", err)
	}
	updated, err := repo.GetSource(ctx, "custom-src")
	if err != nil {
		t.Fatalf("get source after update: %v", err)
	}
	if updated.Name != "Updated Source" || updated.Enabled {
		t.Errorf("got source after update: %+v", updated)
	}

	// Delete source
	if err := repo.DeleteSource(ctx, "custom-src"); err != nil {
		t.Fatalf("delete source: %v", err)
	}
	if _, err := repo.GetSource(ctx, "custom-src"); !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestDiscoveryRuns(t *testing.T) {
	repo := openTestStore(t)
	ctx := context.Background()

	runID, err := repo.CreateDiscoveryRun(ctx, DiscoveryRun{
		Status: "running",
		Stats:  `{"fetched":0}`,
	})
	if err != nil {
		t.Fatalf("create run: %v", err)
	}
	if runID <= 0 {
		t.Fatalf("invalid runID: %d", runID)
	}

	run, err := repo.GetDiscoveryRun(ctx, runID)
	if err != nil {
		t.Fatalf("get run: %v", err)
	}
	if run.Status != "running" {
		t.Errorf("status = %q, want running", run.Status)
	}

	// Update run
	now := nowUTC()
	run.CompletedAt = &now
	run.Status = "completed"
	run.Stats = `{"fetched":10,"new":5}`
	if err := repo.UpdateDiscoveryRun(ctx, run); err != nil {
		t.Fatalf("update run: %v", err)
	}

	latest, err := repo.LatestDiscoveryRun(ctx)
	if err != nil {
		t.Fatalf("latest run: %v", err)
	}
	if latest.ID != runID || latest.Status != "completed" || latest.CompletedAt == nil {
		t.Errorf("latest run mismatch: %+v", latest)
	}
}

func TestInsertDiscoveredJobDedup(t *testing.T) {
	repo := openTestStore(t)
	ctx := context.Background()

	job := Job{
		Title:              "Backend Engineer",
		Company:            "TechCorp",
		URL:                "https://techcorp.example/jobs/42",
		Description:        "Go backend role",
		Source:             "himalayas",
		Status:             "new",
		ExternalID:         "42",
		MatchedKeywords:    []string{"backend", "go"},
		MatchedResumeTerms: []string{"go", "api"},
	}

	// First insert: succeeds and returns inserted = true
	id1, inserted, err := repo.InsertDiscoveredJob(ctx, job)
	if err != nil {
		t.Fatalf("first insert: %v", err)
	}
	if !inserted || id1 <= 0 {
		t.Fatalf("expected inserted=true with valid id, got %v, id=%d", inserted, id1)
	}

	// Second insert with the same URL: deduplicated! (inserted = false, id = 0)
	id2, inserted2, err := repo.InsertDiscoveredJob(ctx, job)
	if err != nil {
		t.Fatalf("second insert: %v", err)
	}
	if inserted2 || id2 != 0 {
		t.Fatalf("expected inserted=false and id=0, got inserted=%v, id=%d", inserted2, id2)
	}

	// Verify job retrieved has matched terms and external ID
	fetched, err := repo.GetJob(ctx, id1)
	if err != nil {
		t.Fatalf("get job: %v", err)
	}
	if fetched.ExternalID != "42" {
		t.Errorf("externalID = %q, want 42", fetched.ExternalID)
	}
	if len(fetched.MatchedKeywords) != 2 || fetched.MatchedKeywords[0] != "backend" {
		t.Errorf("matchedKeywords = %+v", fetched.MatchedKeywords)
	}
	if len(fetched.MatchedResumeTerms) != 2 || fetched.MatchedResumeTerms[0] != "go" {
		t.Errorf("matchedResumeTerms = %+v", fetched.MatchedResumeTerms)
	}
}
