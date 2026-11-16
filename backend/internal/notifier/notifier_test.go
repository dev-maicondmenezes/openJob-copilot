package notifier

import (
	"context"
	"log/slog"
	"path/filepath"
	"testing"
	"time"

	"github.com/maicondmenezes/openjob/internal/events"
	"github.com/maicondmenezes/openjob/internal/store"
)

func openNotifierStore(t *testing.T) *store.SQLiteStore {
	t.Helper()
	repo, err := store.Open(filepath.Join(t.TempDir(), "openjob.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = repo.Close() })
	return repo
}

func TestMapEventApplicationStatus(t *testing.T) {
	cases := []struct {
		to       string
		wantSev  string
		wantType string
	}{
		{"interview", "info", "application_status"},
		{"rejected", "warning", "application_status"},
		{"offer", "success", "application_status"},
		{"sent", "info", "application_status"},
	}
	for _, tc := range cases {
		nb, ok := mapEvent(events.Event{
			Kind: events.KindApplicationStatusChanged,
			Metadata: map[string]any{
				"job_label": "Dev @ Acme", "from": "pending", "to": tc.to,
			},
		})
		if !ok {
			t.Fatalf("to=%s: not mapped", tc.to)
		}
		if nb.severity != tc.wantSev || nb.notificationType != tc.wantType {
			t.Errorf("to=%s: got %s/%s, want %s/%s", tc.to, nb.severity, nb.notificationType, tc.wantSev, tc.wantType)
		}
		if nb.body == "" || nb.title == "" {
			t.Errorf("to=%s: empty title/body", tc.to)
		}
	}
}

func TestInAppPersistEventsAndDedup(t *testing.T) {
	repo := openNotifierStore(t)
	bus := events.New()
	defer bus.Close()

	n := NewInApp(repo, slog.Default())
	n.Start(bus)
	defer n.Stop()

	// source_error with dedup key: repeated publishes within the window
	// produce a single notification (spec §6 — dedup por type+id).
	for i := 0; i < 3; i++ {
		bus.Publish(events.Event{
			Kind:     events.KindSourceFetchFailed,
			Severity: events.SeverityWarning,
			Title:    "Falha em fonte",
			Body:     "remoteok: timeout",
			DedupKey: "source_error:remoteok",
			Metadata: map[string]any{"source": "remoteok"},
		})
	}
	// Different source is NOT deduplicated.
	bus.Publish(events.Event{
		Kind:     events.KindSourceFetchFailed,
		Severity: events.SeverityWarning,
		DedupKey: "source_error:remotive",
		Body:     "remotive: 500",
	})

	deadline := time.Now().Add(3 * time.Second)
	var count int
	for time.Now().Before(deadline) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		items, _, err := repo.ListNotifications(ctx, store.NotificationFilter{Type: "source_error", Limit: 50})
		cancel()
		count = len(items)
		if err == nil && count == 2 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if count != 2 {
		t.Fatalf("source_error notifications = %d, want 2 (dedup collapsed repeats)", count)
	}

	// Unread count reflects the two source errors.
	if unread, _ := repo.CountUnreadNotifications(context.Background()); unread < 2 {
		t.Fatalf("unread = %d, want >= 2", unread)
	}
}

func TestInAppDisabledDoesNotPersist(t *testing.T) {
	repo := openNotifierStore(t)
	ctx := context.Background()

	// Disable in-app channel.
	cfg := DefaultConfig()
	cfg.InAppEnabled = false
	if err := SaveConfig(ctx, repo, cfg); err != nil {
		t.Fatal(err)
	}

	bus := events.New()
	defer bus.Close()
	n := NewInApp(repo, slog.Default())
	n.Start(bus)
	defer n.Stop()

	bus.Publish(events.Event{Kind: events.KindTest, Title: "ping"})
	bus.Publish(events.Event{
		Kind:     events.KindApplicationStatusChanged,
		Metadata: map[string]any{"job_label": "x", "from": "pending", "to": "interview"},
	})

	time.Sleep(200 * time.Millisecond)
	items, _, _ := repo.ListNotifications(ctx, store.NotificationFilter{Limit: 10})
	if len(items) != 0 {
		t.Fatalf("in_app_enabled=false must not persist, got %v", items)
	}
}

func TestConfigRoundtrip(t *testing.T) {
	repo := openNotifierStore(t)
	ctx := context.Background()

	cfg, err := LoadConfig(ctx, repo)
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.InAppEnabled || cfg.RetentionDays != 90 || cfg.BrowserNotifications {
		t.Fatalf("defaults wrong: %+v", cfg)
	}

	cfg.RetentionDays = 30
	cfg.BrowserNotifications = true
	if err := SaveConfig(ctx, repo, cfg); err != nil {
		t.Fatal(err)
	}
	reloaded, _ := LoadConfig(ctx, repo)
	if reloaded.RetentionDays != 30 || !reloaded.BrowserNotifications {
		t.Fatalf("roundtrip failed: %+v", reloaded)
	}

	cfg.RetentionDays = 0
	if err := SaveConfig(ctx, repo, cfg); err == nil {
		t.Fatal("retention_days=0 must be rejected")
	}
}

func TestTaskEventsMapped(t *testing.T) {
	nb, ok := mapEvent(events.Event{
		Kind: events.KindTaskCompleted,
		Metadata: map[string]any{
			"task_type": "score_job", "job_label": "Dev @ Acme", "score_fit": 82,
		},
	})
	if !ok || nb.notificationType != "task_completed" || nb.severity != "success" {
		t.Fatalf("task.completed mapping: %+v ok=%v", nb, ok)
	}

	nb, ok = mapEvent(events.Event{
		Kind:     events.KindTaskFailed,
		Metadata: map[string]any{"task_type": "extract_resume_keywords"},
		Body:     "nenhum currículo",
	})
	if !ok || nb.notificationType != "task_failed" || nb.severity != "error" {
		t.Fatalf("task.failed mapping: %+v ok=%v", nb, ok)
	}

	// task.progress has no in-app representation.
	if _, ok := mapEvent(events.Event{Kind: events.KindTaskProgress}); ok {
		t.Error("task.progress must not map to a notification")
	}
	// task.cancelled has no notification type either.
	if _, ok := mapEvent(events.Event{Kind: events.KindTaskCancelled}); ok {
		t.Error("task.cancelled must not map to a notification")
	}
}
