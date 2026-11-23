package tasks

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/maicondmenezes/openjob/internal/store"
)

func TestParseEmailSyncInterval(t *testing.T) {
	cases := []struct {
		in   string
		want time.Duration
		ok   bool
	}{
		{"", 0, false},
		{"off", 0, false},
		{"5m", 5 * time.Minute, true},
		{"15m", 15 * time.Minute, true},
		{"30m", 30 * time.Minute, true},
		{"1h", time.Hour, true},
		{"2h", 2 * time.Hour, true},
		{"30s", 0, false},
		{"bogus", 0, false},
	}
	for _, c := range cases {
		got, ok := parseEmailSyncInterval(c.in)
		if got != c.want || ok != c.ok {
			t.Errorf("parseEmailSyncInterval(%q) = (%v, %v), want (%v, %v)", c.in, got, ok, c.want, c.ok)
		}
	}
}

func TestEmailSyncSchedulerEnqueuesOnlyWhenEnabled(t *testing.T) {
	repo, err := store.Open(filepath.Join(t.TempDir(), "openjob.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = repo.Close() }()

	ctx := context.Background()
	registry := NewRegistry()
	pool := NewPool(repo, registry, nil, DefaultConfig(), nil)
	scheduler := NewEmailSyncScheduler(repo, pool, nil)

	count := func() int {
		n, err := repo.CountTasks(ctx, []string{"queued", "running"}, TypeSyncEmail)
		if err != nil {
			t.Fatalf("CountTasks: %v", err)
		}
		return n
	}

	// No accounts yet: the ticker stays silent.
	scheduler.enqueueIfIdle()
	if n := count(); n != 0 {
		t.Fatalf("queued %d sync tasks without accounts, want 0", n)
	}

	// Disabled account: still silent.
	if err := repo.SaveEmailAccount(ctx, store.EmailAccount{
		ID: "imap", Provider: "imap", Address: "me@example.com", AuthConfig: []byte("x"), Folder: "INBOX", Enabled: false,
	}); err != nil {
		t.Fatal(err)
	}
	scheduler.enqueueIfIdle()
	if n := count(); n != 0 {
		t.Fatalf("queued %d sync tasks with a disabled account, want 0", n)
	}

	// Enabled account: one task, and a second tick does not stack another.
	if err := repo.SaveEmailAccount(ctx, store.EmailAccount{
		ID: "imap", Provider: "imap", Address: "me@example.com", AuthConfig: []byte("x"), Folder: "INBOX", Enabled: true,
	}); err != nil {
		t.Fatal(err)
	}
	scheduler.enqueueIfIdle()
	if n := count(); n != 1 {
		t.Fatalf("queued %d sync tasks with an enabled account, want 1", n)
	}
	scheduler.enqueueIfIdle()
	if n := count(); n != 1 {
		t.Fatalf("queued %d sync tasks after a second tick, want 1 (deduplicated)", n)
	}

	scheduler.Stop()
}
