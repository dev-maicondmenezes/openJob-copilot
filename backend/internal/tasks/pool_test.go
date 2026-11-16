package tasks

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/maicondmenezes/openjob/internal/events"
	"github.com/maicondmenezes/openjob/internal/store"
	"go.uber.org/goleak"
)

type fakeHandler struct {
	typ       string
	mu        sync.Mutex
	runs      int
	failFirst int // fail the first N runs with a plain error
	sleep     time.Duration
	progress  []int
}

func (h *fakeHandler) Type() string { return h.typ }

func (h *fakeHandler) Run(ctx context.Context, payload json.RawMessage, progress func(int)) (json.RawMessage, error) {
	h.mu.Lock()
	h.runs++
	run := h.runs
	fail := run <= h.failFirst
	d := h.sleep
	h.mu.Unlock()

	if d > 0 {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(d):
		}
	}
	if fail {
		return nil, errors.New("falha simulada do handler")
	}
	h.mu.Lock()
	h.progress = append(h.progress, 100)
	h.mu.Unlock()
	progress(100)
	return json.Marshal(map[string]any{"run": run})
}

// openPoolStore opens a test store and registers a cleanup that closes the
// DB BEFORE running the goleak check (the sql connection-opener goroutine
// otherwise trips VerifyNone).
func openPoolStore(t *testing.T) *store.SQLiteStore {
	t.Helper()
	repo, err := store.Open(filepath.Join(t.TempDir(), "openjob.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() {
		_ = repo.Close()
		goleak.VerifyNone(t)
	})
	return repo
}

// waitFor polls cond until it returns true or the deadline expires.
func waitFor(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("condition not met within deadline")
}

func TestPoolSuccessCompletesTask(t *testing.T) {
	repo := openPoolStore(t)
	bus := events.New()
	defer bus.Close()
	sub := bus.Subscribe("test", 16)
	defer sub.Close()

	h := &fakeHandler{typ: TypeScoreJob}
	reg := NewRegistry()
	reg.Register(h)
	pool := NewPool(repo, reg, bus, Config{MaxConcurrency: 1, WorkerPollMS: 50}, nil)
	pool.Start()
	defer pool.Stop()

	task, err := pool.Enqueue(context.Background(), TypeScoreJob, []byte(`{}`), 10)
	if err != nil {
		t.Fatal(err)
	}

	waitFor(t, func() bool {
		got, err := repo.GetTask(context.Background(), task.ID)
		return err == nil && got.Status == "done"
	})
	got, _ := repo.GetTask(context.Background(), task.ID)
	if got.Progress != 100 || got.Result == "" {
		t.Fatalf("done task: %+v", got)
	}

	// completion event published on the bus (progress events may precede it)
	waitFor(t, func() bool { return len(sub.C()) > 0 })
	var e events.Event
	for {
		e = <-sub.C()
		if e.Kind != events.KindTaskProgress {
			break
		}
	}
	if e.Kind != events.KindTaskCompleted || e.Metadata["task_id"] != task.ID {
		t.Fatalf("event = %+v", e)
	}
}

func TestPoolRetryBackoffThenSuccess(t *testing.T) {
	repo := openPoolStore(t)
	h := &fakeHandler{typ: "flaky", failFirst: 2}
	reg := NewRegistry()
	reg.Register(h)
	pool := NewPool(repo, reg, nil, Config{MaxConcurrency: 1, WorkerPollMS: 30}, nil)
	pool.backoffFn = func(int) time.Duration { return 20 * time.Millisecond } // shrink retry delays
	pool.Start()
	defer pool.Stop()

	task, err := pool.Enqueue(context.Background(), "flaky", []byte(`{}`), 10)
	if err != nil {
		t.Fatal(err)
	}

	waitFor(t, func() bool {
		got, err := repo.GetTask(context.Background(), task.ID)
		return err == nil && got.Status == "done"
	})
	got, _ := repo.GetTask(context.Background(), task.ID)
	if got.Attempts != 3 {
		t.Fatalf("attempts = %d, want 3 (2 retries + 1 success)", got.Attempts)
	}
	if got.Error != "" {
		t.Errorf("successful task should clear error, got %q", got.Error)
	}
	h.mu.Lock()
	runs := h.runs
	h.mu.Unlock()
	if runs != 3 {
		t.Fatalf("handler ran %d times, want 3", runs)
	}
}

func TestPoolMaxAttemptsExhaustedFailsWithEvent(t *testing.T) {
	repo := openPoolStore(t)
	bus := events.New()
	defer bus.Close()
	sub := bus.Subscribe("test", 16)
	defer sub.Close()

	h := &fakeHandler{typ: "always_fail", failFirst: 999}
	reg := NewRegistry()
	reg.Register(h)
	pool := NewPool(repo, reg, bus, Config{MaxConcurrency: 1, WorkerPollMS: 30}, nil)
	// Shrink the retry backoff so max_attempts is reached in test time.
	pool.backoffFn = func(int) time.Duration { return 10 * time.Millisecond }

	task, err := pool.Enqueue(context.Background(), "always_fail", []byte(`{}`), 10)
	if err != nil {
		t.Fatal(err)
	}

	pool.Start()
	defer pool.Stop()

	waitFor(t, func() bool {
		got, err := repo.GetTask(context.Background(), task.ID)
		return err == nil && got.Status == "failed"
	})
	got, _ := repo.GetTask(context.Background(), task.ID)
	if got.Attempts != 3 {
		t.Fatalf("attempts = %d, want 3", got.Attempts)
	}
	if got.Error != "falha simulada do handler" {
		t.Fatalf("error = %q", got.Error)
	}

	// task.failed notification event (retry notices may precede it)
	waitFor(t, func() bool { return len(sub.C()) > 0 })
	var e events.Event
	for {
		e = <-sub.C()
		if e.Kind == events.KindTaskFailed {
			break
		}
	}
	if e.Severity != events.SeverityError {
		t.Fatalf("event = %+v", e)
	}
}


func TestPoolCancelRunningPropagatesContext(t *testing.T) {
	repo := openPoolStore(t)
	h := &fakeHandler{typ: "slow", sleep: 10 * time.Second}
	reg := NewRegistry()
	reg.Register(h)
	pool := NewPool(repo, reg, nil, Config{MaxConcurrency: 1, WorkerPollMS: 30}, nil)
	pool.Start()
	defer pool.Stop()

	task, err := pool.Enqueue(context.Background(), "slow", []byte(`{}`), 10)
	if err != nil {
		t.Fatal(err)
	}

	waitFor(t, func() bool {
		got, err := repo.GetTask(context.Background(), task.ID)
		return err == nil && got.Status == "running"
	})

	if _, err := pool.CancelTask(context.Background(), task.ID); err != nil {
		t.Fatal(err)
	}

	waitFor(t, func() bool {
		got, err := repo.GetTask(context.Background(), task.ID)
		return err == nil && got.Status == "cancelled"
	})
	got, _ := repo.GetTask(context.Background(), task.ID)
	if got.Error != "Cancelada." {
		t.Errorf("cancelled task error = %q", got.Error)
	}
}

func TestPoolCancelQueuedDirectly(t *testing.T) {
	repo := openPoolStore(t)
	reg := NewRegistry()
	pool := NewPool(repo, reg, nil, Config{MaxConcurrency: 1, WorkerPollMS: 250}, nil)
	pool.Start()
	defer pool.Stop()

	// Fill the single worker slot with a slow task, then cancel the queued one.
	slow := &fakeHandler{typ: "slow", sleep: 500 * time.Millisecond}
	reg.Register(slow)
	first, _ := pool.Enqueue(context.Background(), "slow", []byte(`{}`), 1)
	second, err := pool.Enqueue(context.Background(), "slow", []byte(`{}`), 2)
	if err != nil {
		t.Fatal(err)
	}

	waitFor(t, func() bool {
		got, _ := repo.GetTask(context.Background(), first.ID)
		return got.Status == "running"
	})

	if _, err := pool.CancelTask(context.Background(), second.ID); err != nil {
		t.Fatal(err)
	}
	got, _ := repo.GetTask(context.Background(), second.ID)
	if got.Status != "cancelled" {
		t.Fatalf("queued task status = %s, want cancelled", got.Status)
	}
}

func TestPoolBootRecoveryRequeuesRunning(t *testing.T) {
	repo := openPoolStore(t)
	ctx := context.Background()

	// Simulate a crash: a task left 'running' by a previous process.
	orphan := store.Task{ID: newULID(), Type: "rec", Status: "running", Payload: "{}", MaxAttempts: 3}
	if err := repo.InsertTask(ctx, orphan); err != nil {
		t.Fatal(err)
	}

	h := &fakeHandler{typ: "rec"}
	reg := NewRegistry()
	reg.Register(h)
	pool := NewPool(repo, reg, nil, Config{MaxConcurrency: 1, WorkerPollMS: 30}, nil)
	pool.Start() // recovery happens here
	defer pool.Stop()

	waitFor(t, func() bool {
		got, err := repo.GetTask(ctx, orphan.ID)
		return err == nil && got.Status == "done"
	})
}

func TestPoolConcurrencyRespected(t *testing.T) {
	repo := openPoolStore(t)

	const concurrency = 2
	var mu sync.Mutex
	current, peak := 0, 0
	h := handlerFunc(TypeScoreJob, func(ctx context.Context, payload json.RawMessage, progress func(int)) (json.RawMessage, error) {
		mu.Lock()
		current++
		if current > peak {
			peak = current
		}
		mu.Unlock()
		time.Sleep(150 * time.Millisecond)
		mu.Lock()
		current--
		mu.Unlock()
		return json.Marshal(map[string]any{})
	})
	reg := NewRegistry()
	reg.Register(h)
	pool := NewPool(repo, reg, nil, Config{MaxConcurrency: concurrency, WorkerPollMS: 20}, nil)
	pool.Start()
	defer pool.Stop()

	for i := 0; i < 4; i++ {
		if _, err := pool.Enqueue(context.Background(), TypeScoreJob, []byte(`{}`), 10); err != nil {
			t.Fatal(err)
		}
	}

	waitFor(t, func() bool {
		n, _ := repo.CountTasksByStatus(context.Background())
		return n["done"] == 4
	})
	mu.Lock()
	defer mu.Unlock()
	if peak > concurrency {
		t.Fatalf("peak concurrency = %d, want <= %d", peak, concurrency)
	}
}

// handlerFunc builds a Handler from a bare function (test helper).
func handlerFunc(typ string, fn func(context.Context, json.RawMessage, func(int)) (json.RawMessage, error)) Handler {
	return &fnHandler{typ: typ, fn: fn}
}

type fnHandler struct {
	typ string
	fn  func(context.Context, json.RawMessage, func(int)) (json.RawMessage, error)
}

func (f *fnHandler) Type() string { return f.typ }
func (f *fnHandler) Run(ctx context.Context, payload json.RawMessage, progress func(int)) (json.RawMessage, error) {
	return f.fn(ctx, payload, progress)
}

func TestBackoffSchedule(t *testing.T) {
	for _, tc := range []struct {
		attempts int
		want     time.Duration
	}{
		{1, time.Second},
		{2, 4 * time.Second},
		{3, 16 * time.Second},
		{7, time.Minute},
	} {
		if got := backoff(tc.attempts); got != tc.want {
			t.Errorf("backoff(%d) = %s, want %s", tc.attempts, got, tc.want)
		}
	}
}

func TestULIDSortableAndUnique(t *testing.T) {
	ids := make([]string, 1000)
	for i := range ids {
		ids[i] = newULID()
	}
	seen := map[string]bool{}
	for i, id := range ids {
		if seen[id] {
			t.Fatalf("duplicate ULID at %d", i)
		}
		seen[id] = true
		if len(id) != 26 {
			t.Fatalf("ULID length = %d, want 26", len(id))
		}
		if i > 0 && ids[i-1] > id {
			t.Fatalf("ULIDs not monotonic: %s > %s", ids[i-1], id)
		}
	}
}

func TestPoolScoreHandlerEndToEnd(t *testing.T) {
	repo := openPoolStore(t)
	ctx := context.Background()

	// Seed curriculum + BYOK + a job (the encrypted-key path is exercised).
	if _, err := repo.SaveCurriculum(ctx, store.Curriculum{ContentMD: "Go developer"}); err != nil {
		t.Fatal(err)
	}
	repo.SetSetting(ctx, "llm.base_url", "http://localhost:1")
	repo.SetSetting(ctx, "llm.model", "test")

	h := NewScoreJobHandler(repo, []byte("k"), nil, nil, nil)
	reg := NewRegistry()
	reg.Register(h)
	pool := NewPool(repo, reg, nil, Config{MaxConcurrency: 1, WorkerPollMS: 30}, nil)
	pool.backoffFn = func(int) time.Duration { return 10 * time.Millisecond } // shrink retries
	pool.Start()
	defer pool.Stop()

	jobID, err := repo.CreateJob(ctx, store.Job{Title: "Dev", Company: "Acme", Description: "Go backend", Source: "manual", Status: "new"})
	if err != nil {
		t.Fatal(err)
	}
	task, err := pool.EnqueueScoreJob(ctx, jobID, 10)
	if err != nil {
		t.Fatal(err)
	}

	// No LLM configured → legible error, retried to 'failed' eventually.
	waitFor(t, func() bool {
		got, err := repo.GetTask(ctx, task.ID)
		return err == nil && got.Status == "failed"
	})
	got, _ := repo.GetTask(ctx, task.ID)
	if got.Error == "" {
		t.Fatal("expected a legible error for missing LLM config")
	}
	t.Log("score handler error:", got.Error)

	// Keywords handler runs without LLM (pure heuristics).
	kh := NewExtractKeywordsHandler(repo)
	reg2 := NewRegistry()
	reg2.Register(kh)
	pool2 := NewPool(repo, reg2, nil, Config{MaxConcurrency: 1, WorkerPollMS: 30}, nil)
	pool2.Start()
	defer pool2.Stop()

	ktask, err := pool2.Enqueue(ctx, TypeExtractResumeKeywords, []byte(`{}`), 1)
	if err != nil {
		t.Fatal(err)
	}
	waitFor(t, func() bool {
		got, err := repo.GetTask(ctx, ktask.ID)
		return err == nil && got.Status == "done"
	})
	kres, _ := repo.GetTask(ctx, ktask.ID)
	var out struct {
		Count    int      `json:"count"`
		Keywords []string `json:"keywords"`
	}
	if err := json.Unmarshal([]byte(kres.Result), &out); err != nil {
		t.Fatalf("keywords result: %v", err)
	}
	if out.Count == 0 || len(out.Keywords) != out.Count {
		t.Fatalf("keywords result = %+v", out)
	}
	// Cached in settings (CP6 contract preserved).
	raw, err := repo.GetSetting(ctx, "resume_keywords")
	if err != nil || raw == "" {
		t.Fatalf("resume_keywords not cached: %q %v", raw, err)
	}
}
