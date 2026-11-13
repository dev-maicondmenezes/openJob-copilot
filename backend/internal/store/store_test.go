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
	if count != 1 {
		t.Errorf("applied migrations = %d, want 1", count)
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
