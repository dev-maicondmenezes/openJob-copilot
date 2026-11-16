package store

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

func openStore(t *testing.T) *SQLiteStore {
	t.Helper()
	repo, err := Open( //nolint:staticcheck // helper local ao teste
		t.TempDir() + "/openjob.db")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = repo.Close() })
	return repo
}

func TestNotificationsRoundtripAndFilters(t *testing.T) {
	repo := openStore(t)
	ctx := context.Background()

	id1, err := repo.InsertNotification(ctx, Notification{
		Type: "application_status", Severity: "success",
		Title: "t1", Body: "b1", Link: "/applications", Metadata: `{"application_id":1}`,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.InsertNotification(ctx, Notification{
		Type: "source_error", Severity: "warning", Title: "t2", Body: "b2",
	}); err != nil {
		t.Fatal(err)
	}
	_, err = repo.InsertNotification(ctx, Notification{
		Type: "task_failed", Severity: "error", Title: "t3", Body: "b3", ReadAt: timePtr(time.Now().UTC()),
	})
	if err != nil {
		t.Fatal(err)
	}

	if n, err := repo.CountUnreadNotifications(ctx); err != nil || n != 2 {
		t.Fatalf("unread = %d, err %v; want 2", n, err)
	}

	// Feed: newest first.
	items, next, err := repo.ListNotifications(ctx, NotificationFilter{Limit: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 || items[0].ID < items[1].ID {
		t.Fatalf("feed order wrong: %v", items)
	}
	if next == "" {
		t.Fatal("expected next cursor")
	}
	if items[0].ID != 3 {
		t.Errorf("newest first: got %d want 3", items[0].ID)
	}

	// Cursor pagination continues without overlap (page1 held ids 3 and 2).
	page2, next2, err := repo.ListNotifications(ctx, NotificationFilter{Limit: 10, Cursor: next})
	if err != nil {
		t.Fatal(err)
	}
	if len(page2) != 1 || page2[0].ID != 1 {
		t.Fatalf("page2 = %v, want oldest notification id=1", page2)
	}
	if next2 != "" {
		t.Errorf("page2 next cursor should be empty, got %q", next2)
	}

	// Filter by type.
	onlySource, _, err := repo.ListNotifications(ctx, NotificationFilter{Type: "source_error", Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(onlySource) != 1 || onlySource[0].Type != "source_error" {
		t.Fatalf("type filter failed: %v", onlySource)
	}

	// Filter unread only.
	unreadOnly, _, err := repo.ListNotifications(ctx, NotificationFilter{UnreadOnly: true, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(unreadOnly) != 2 {
		t.Fatalf("unread filter failed: %d items", len(unreadOnly))
	}

	// Mark one read → unread count drops, read_at set.
	if err := repo.MarkNotificationRead(ctx, id1); err != nil {
		t.Fatal(err)
	}
	if err := repo.MarkNotificationRead(ctx, id1); err != nil { // idempotent
		t.Fatal(err)
	}
	if n, _ := repo.CountUnreadNotifications(ctx); n != 1 {
		t.Fatalf("unread after mark = %d, want 1", n)
	}
	got, err := repo.GetNotification(ctx, id1)
	if err != nil || got.ReadAt == nil {
		t.Fatalf("read_at not stamped: %v %v", got, err)
	}

	// Mark all read.
	if n, err := repo.MarkAllNotificationsRead(ctx); err != nil || n != 1 {
		t.Fatalf("mark all = %d, err %v; want 1", n, err)
	}
	if n, _ := repo.CountUnreadNotifications(ctx); n != 0 {
		t.Fatalf("unread after read-all = %d, want 0", n)
	}

	// Delete one.
	if err := repo.DeleteNotification(ctx, id1); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.GetNotification(ctx, id1); err != ErrNotFound {
		t.Fatalf("get after delete = %v, want ErrNotFound", err)
	}

	// Clear read.
	if n, err := repo.DeleteReadNotifications(ctx); err != nil || n != 2 {
		t.Fatalf("clear read = %d, err %v; want 2", n, err)
	}
}

func TestNotificationsRetentionKeepsUnread(t *testing.T) {
	repo := openStore(t)
	ctx := context.Background()

	old := time.Now().UTC().AddDate(0, 0, -100)
	if _, err := repo.InsertNotification(ctx, Notification{Type: "test", Title: "old-read", ReadAt: &old}); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.InsertNotification(ctx, Notification{Type: "test", Title: "old-unread"}); err != nil {
		t.Fatal(err)
	}

	deleted, err := repo.DeleteReadNotificationsBefore(ctx, time.Now().UTC().AddDate(0, 0, -90))
	if err != nil {
		t.Fatal(err)
	}
	if deleted != 1 {
		t.Fatalf("deleted = %d, want 1", deleted)
	}
	items, _, err := repo.ListNotifications(ctx, NotificationFilter{Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Title != "old-unread" {
		t.Fatalf("unread notification must survive retention: %v", items)
	}
}

func timePtr(t time.Time) *time.Time { return &t }

func TestTaskClaimAtomicityUnderConcurrency(t *testing.T) {
	repo := openStore(t)
	ctx := context.Background()

	const total = 25
	for i := 0; i < total; i++ {
		if err := repo.InsertTask(ctx, Task{ID: fmt.Sprintf("01TEST%02d", i), Type: "score_job", Priority: 50}); err != nil {
			t.Fatal(err)
		}
	}

	const workers = 10
	var mu sync.Mutex
	claimed := map[string]bool{}
	var wg sync.WaitGroup
	errs := make(chan error, workers)
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				task, err := repo.ClaimNextTask(ctx)
				if err == ErrNotFound {
					return
				}
				if err != nil {
					errs <- err
					return
				}
				mu.Lock()
				if claimed[task.ID] {
					errs <- fmt.Errorf("task %s claimed twice", task.ID)
				}
				claimed[task.ID] = true
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatal(err)
	}
	if len(claimed) != total {
		t.Fatalf("claimed %d tasks, want %d", len(claimed), total)
	}
	// Every claim increments attempts exactly once.
	task, err := repo.GetTask(ctx, claimedFirstID(claimed))
	if err != nil {
		t.Fatal(err)
	}
	if task.Attempts != 1 || task.Status != "running" {
		t.Fatalf("task state after claim: %+v", task)
	}
}

func claimedFirstID(claimed map[string]bool) string {
	for id := range claimed {
		return id
	}
	return ""
}

func TestTaskLifecycleRetryBackoffAndFailure(t *testing.T) {
	repo := openStore(t)
	ctx := context.Background()

	if err := repo.InsertTask(ctx, Task{ID: "01TASK1", Type: "score_job", MaxAttempts: 2}); err != nil {
		t.Fatal(err)
	}

	// First attempt fails → requeued with backoff, attempts preserved.
	task, err := repo.ClaimNextTask(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if task.Attempts != 1 || task.Status != "running" {
		t.Fatalf("claim 1: %+v", task)
	}
	next := time.Now().UTC().Add(time.Minute)
	if err := repo.RequeueTaskWithBackoff(ctx, task.ID, next, "boom"); err != nil {
		t.Fatal(err)
	}
	task, _ = repo.GetTask(ctx, task.ID)
	if task.Status != "queued" || task.Error != "boom" || task.NextRunAt == nil {
		t.Fatalf("after requeue: %+v", task)
	}

	// Backoff not elapsed → not claimable.
	if _, err := repo.ClaimNextTask(ctx); err != ErrNotFound {
		t.Fatalf("task with future next_run_at claimed: %v", err)
	}

	// Expired backoff → claimable again.
	// force next_run_at into the past
	if _, err := repo.db.Exec(`UPDATE tasks SET next_run_at = ? WHERE id = ?`, time.Now().UTC().Add(-time.Second), task.ID); err != nil {
		t.Fatal(err)
	}
	task, err = repo.ClaimNextTask(ctx)
	if err != nil || task.Attempts != 2 {
		t.Fatalf("claim after backoff: %+v err %v", task, err)
	}

	// Exhausted attempts → failed.
	if err := repo.FinishTask(ctx, task.ID, "failed", "", "boom final"); err != nil {
		t.Fatal(err)
	}
	task, _ = repo.GetTask(ctx, task.ID)
	if task.Status != "failed" || task.Error != "boom final" || task.FinishedAt == nil {
		t.Fatalf("failed task: %+v", task)
	}

	// Manual retry resets everything.
	if err := repo.RetryTask(ctx, task.ID); err != nil {
		t.Fatal(err)
	}
	task, _ = repo.GetTask(ctx, task.ID)
	if task.Status != "queued" || task.Attempts != 0 || task.Error != "" || task.NextRunAt != nil {
		t.Fatalf("after manual retry: %+v", task)
	}
}

func TestTaskCancelAndBootRecovery(t *testing.T) {
	repo := openStore(t)
	ctx := context.Background()

	if err := repo.InsertTask(ctx, Task{ID: "01CANCL", Type: "score_job"}); err != nil {
		t.Fatal(err)
	}
	// Cancel queued directly.
	ok, err := repo.CancelQueuedTask(ctx, "01CANCL", "cancelado")
	if err != nil || !ok {
		t.Fatalf("cancel queued: ok=%v err=%v", ok, err)
	}
	task, _ := repo.GetTask(ctx, "01CANCL")
	if task.Status != "cancelled" {
		t.Fatalf("cancelled task: %+v", task)
	}

	// Boot recovery: running → queued.
	if err := repo.InsertTask(ctx, Task{ID: "01ORPHN", Type: "score_job"}); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.ClaimNextTask(ctx); err != nil {
		t.Fatal(err)
	}
	n, err := repo.RequeueRunningTasks(ctx)
	if err != nil || n != 1 {
		t.Fatalf("requeue running = %d, err %v; want 1", n, err)
	}
	task, _ = repo.GetTask(ctx, "01ORPHN")
	if task.Status != "queued" || task.StartedAt != nil {
		t.Fatalf("recovered task: %+v", task)
	}
}

func TestTaskCountsAndBulkActions(t *testing.T) {
	repo := openStore(t)
	ctx := context.Background()

	for i, status := range []string{"queued", "queued", "failed", "running"} {
		id := fmt.Sprintf("01CNT%02d", i)
		tk := Task{ID: id, Type: "score_job", Status: status}
		if status == "failed" { // finished_at is required by CountTasksFailedSince
			tk.FinishedAt = timePtr(time.Now().UTC())
		}
		if err := repo.InsertTask(ctx, tk); err != nil {
			t.Fatal(err)
		}
	}
	counts, err := repo.CountTasksByStatus(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if counts["queued"] != 2 || counts["failed"] != 1 || counts["running"] != 1 {
		t.Fatalf("counts = %v", counts)
	}
	if n, _ := repo.CountTasks(ctx, []string{"queued", "running"}, "score_job"); n != 3 {
		t.Fatalf("CountTasks = %d, want 3", n)
	}
	if n, _ := repo.CountTasksFailedSince(ctx, time.Now().UTC().Add(-time.Hour)); n != 1 {
		t.Fatalf("failed since = %d, want 1", n)
	}

	if n, err := repo.CancelAllQueuedTasks(ctx, "bulk"); err != nil || n != 2 {
		t.Fatalf("cancel all queued = %d, err %v; want 2", n, err)
	}
	if n, err := repo.RetryAllFailedTasks(ctx); err != nil || n != 1 {
		t.Fatalf("retry all failed = %d, err %v; want 1", n, err)
	}
	counts, _ = repo.CountTasksByStatus(ctx)
	if counts["queued"] != 1 || counts["cancelled"] != 2 || counts["running"] != 1 {
		t.Fatalf("counts after bulk = %v", counts)
	}
}

func TestTaskListPagination(t *testing.T) {
	repo := openStore(t)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		if err := repo.InsertTask(ctx, Task{ID: fmt.Sprintf("01LIST%02d", i), Type: "score_job"}); err != nil {
			t.Fatal(err)
		}
	}
	items, next, err := repo.ListTasks(ctx, []string{"queued"}, "", "", 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 3 || next == "" {
		t.Fatalf("page1 = %d items, next %q", len(items), next)
	}
	page2, next2, err := repo.ListTasks(ctx, nil, "score_job", next, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(page2) != 2 || next2 != "" {
		t.Fatalf("page2 = %d items, next %q", len(page2), next2)
	}
	// No overlap between pages.
	for _, a := range items {
		for _, b := range page2 {
			if a.ID == b.ID {
				t.Fatal("cursor pagination repeated a task")
			}
		}
	}
}
