package tasks

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/maicondmenezes/openjob/internal/events"
	"github.com/maicondmenezes/openjob/internal/store"
)

// SettingKey is the settings key holding the JSON worker config.
const SettingKey = "tasks.config"

// Config controls the worker pool (§3.3). Stored as JSON in settings.
type Config struct {
	MaxConcurrency int `json:"max_concurrency"` // 1..8
	WorkerPollMS   int `json:"worker_poll_ms"`  // 100..5000
}

// DefaultConfig mirrors §3.3: 2 workers polling every 500ms.
func DefaultConfig() Config {
	return Config{MaxConcurrency: 2, WorkerPollMS: 500}
}

// clamp normalizes a config to the documented ranges.
func clamp(cfg Config) Config {
	def := DefaultConfig()
	if cfg.MaxConcurrency < 1 {
		cfg.MaxConcurrency = def.MaxConcurrency
	}
	if cfg.MaxConcurrency > 8 {
		cfg.MaxConcurrency = 8
	}
	if cfg.WorkerPollMS < 100 {
		cfg.WorkerPollMS = 500
	}
	if cfg.WorkerPollMS > 5000 {
		cfg.WorkerPollMS = 5000
	}
	return cfg
}

// LoadConfig reads the config from settings with range validation.
func LoadConfig(ctx context.Context, repo store.Repository) (Config, error) {
	raw, err := repo.GetSetting(ctx, SettingKey)
	if err != nil {
		return DefaultConfig(), nil
	}
	var cfg Config
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		return DefaultConfig(), nil
	}
	return clamp(cfg), nil
}

// SaveConfig validates ranges (spec: 1–8 workers, 100–5000ms poll) and persists.
func SaveConfig(ctx context.Context, repo store.Repository, cfg Config) error {
	if cfg.MaxConcurrency < 1 || cfg.MaxConcurrency > 8 {
		return fmt.Errorf("max_concurrency deve estar entre 1 e 8")
	}
	if cfg.WorkerPollMS < 100 || cfg.WorkerPollMS > 5000 {
		return fmt.Errorf("worker_poll_ms deve estar entre 100 e 5000")
	}
	b, err := json.Marshal(clamp(cfg))
	if err != nil {
		return err
	}
	return repo.SetSetting(ctx, SettingKey, string(b))
}

// backoff returns the delay before attempt n+1 (1s, 4s, 16s… capped at 1min).
func backoff(attempts int) time.Duration {
	d := time.Second
	for i := 1; i < attempts; i++ {
		d *= 4
		if d >= time.Minute {
			return time.Minute
		}
	}
	return d
}

type runningTask struct {
	cancel context.CancelFunc
}

// Pool is the durable task queue worker pool. It is also the enqueue API used
// by producers (discovery orchestrator, settings endpoints).
type Pool struct {
	repo     store.Repository
	registry *Registry
	bus      *events.Bus // optional: nil in unit tests
	logger   *slog.Logger
	// backoffFn computes the delay before the next retry (injectable for tests).
	backoffFn func(attempts int) time.Duration

	stop chan struct{}
	wg   sync.WaitGroup

	mu      sync.Mutex
	cfg     Config
	workers int
	running map[string]*runningTask
}

// NewPool creates the pool; call Start to begin processing.
func NewPool(repo store.Repository, registry *Registry, bus *events.Bus, cfg Config, logger *slog.Logger) *Pool {
	if logger == nil {
		logger = slog.Default()
	}
	return &Pool{
		repo:      repo,
		registry:  registry,
		bus:       bus,
		logger:    logger,
		backoffFn: backoff,
		stop:      make(chan struct{}),
		cfg:       clamp(cfg),
		running:   map[string]*runningTask{},
	}
}

// Start recovers orphaned 'running' tasks (previous process died mid-run) and
// spawns the configured number of workers.
func (p *Pool) Start() {
	p.mu.Lock()
	target := p.cfg.MaxConcurrency
	p.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if n, err := p.repo.RequeueRunningTasks(ctx); err != nil {
		p.logger.Warn("tasks: requeue running failed", "err", err)
	} else if n > 0 {
		p.logger.Info("tasks: requeued orphaned running tasks", "count", n)
	}

	p.wg.Add(1)
	go p.supervise(target)
}

// supervise keeps the worker count in sync with cfg.MaxConcurrency so
// PUT /api/settings/tasks takes effect without a restart.
func (p *Pool) supervise(initial int) {
	defer p.wg.Done()
	p.mu.Lock()
	p.workers = initial
	p.mu.Unlock()
	for i := 0; i < initial; i++ {
		p.wg.Add(1)
		go p.worker()
	}

	t := time.NewTicker(time.Second)
	defer t.Stop()
	for {
		select {
		case <-p.stop:
			return
		case <-t.C:
			p.mu.Lock()
			current, target := p.workers, p.cfg.MaxConcurrency
			p.mu.Unlock()
			if target > current {
				for i := current; i < target; i++ {
					p.mu.Lock()
					p.workers++
					p.mu.Unlock()
					p.wg.Add(1)
					go p.worker()
				}
			}
			// Shrinking is handled by workers themselves: a worker exits when
			// it observes workers > target between tasks (see worker()).
		}
	}
}

func (p *Pool) currentConfig() Config {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.cfg
}

// SetConfig adjusts concurrency/poll interval at runtime.
func (p *Pool) SetConfig(cfg Config) {
	p.mu.Lock()
	p.cfg = clamp(cfg)
	p.mu.Unlock()
}

// Stop terminates the workers, waiting for the in-flight handler call to
// finish (handlers honor ctx cancellation, so Stop also cancels running ctxs).
func (p *Pool) Stop() {
	close(p.stop)

	// Cancel any running task contexts so handlers return promptly.
	p.mu.Lock()
	for id, rt := range p.running {
		p.logger.Info("tasks: cancelling running task on shutdown", "task_id", id)
		rt.cancel()
	}
	p.mu.Unlock()

	p.wg.Wait()
}

func (p *Pool) worker() {
	defer p.wg.Done()
	for {
		select {
		case <-p.stop:
			return
		default:
		}

		// Dynamic shrink: exit when there are more workers than configured.
		p.mu.Lock()
		if p.workers > p.cfg.MaxConcurrency {
			p.workers--
			p.mu.Unlock()
			return
		}
		p.mu.Unlock()

		task, err := p.repo.ClaimNextTask(context.Background())
		switch {
		case errors.Is(err, store.ErrNotFound):
			cfg := p.currentConfig()
			t := time.NewTimer(time.Duration(cfg.WorkerPollMS) * time.Millisecond)
			select {
			case <-p.stop:
				t.Stop()
				return
			case <-t.C:
			}
			continue
		case err != nil:
			p.logger.Error("tasks: claim failed", "err", err)
			t := time.NewTimer(time.Second)
			select {
			case <-p.stop:
				t.Stop()
				return
			case <-t.C:
			}
			continue
		}

		p.run(task)
	}
}

// Enqueue adds a task to the queue. Priority: lower runs first (scoring of
// high resume-matched jobs uses a lower value).
func (p *Pool) Enqueue(ctx context.Context, typ string, payload json.RawMessage, priority int) (store.Task, error) {
	t := store.Task{
		ID:          newULID(),
		Type:        typ,
		Status:      "queued",
		Payload:     string(payload),
		MaxAttempts: 3,
		Priority:    priority,
	}
	if t.Priority == 0 {
		t.Priority = 100
	}
	if err := p.repo.InsertTask(ctx, t); err != nil {
		return store.Task{}, fmt.Errorf("tasks: enqueue: %w", err)
	}
	if got, err := p.repo.GetTask(ctx, t.ID); err == nil {
		return got, nil
	}
	return t, nil
}

// CancelTask cancels a queued task directly or a running one by propagating
// ctx cancellation to its handler.
func (p *Pool) CancelTask(ctx context.Context, id string) (store.Task, error) {
	ok, err := p.repo.CancelQueuedTask(ctx, id, "Cancelada pelo usuário.")
	if err != nil {
		return store.Task{}, err
	}
	if ok {
		t, gerr := p.repo.GetTask(ctx, id)
		if gerr != nil {
			return store.Task{}, gerr
		}
		p.publish(events.Event{
			Kind:     events.KindTaskCancelled,
			Severity: events.SeverityWarning,
			Title:    "Tarefa cancelada",
			Body:     humanType(t.Type) + " cancelada.",
			Metadata: map[string]any{"task_id": id, "task_type": t.Type},
		})
		return t, nil
	}

	p.mu.Lock()
	rt, running := p.running[id]
	p.mu.Unlock()
	if !running {
		return store.Task{}, fmt.Errorf("tarefa %s não está mais em execução", id)
	}
	rt.cancel()
	t, gerr := p.repo.GetTask(ctx, id)
	if gerr != nil {
		return store.Task{}, gerr
	}
	return t, nil
}

// RetryTask re-enqueues a failed/cancelled task resetting attempts.
func (p *Pool) RetryTask(ctx context.Context, id string) (store.Task, error) {
	if err := p.repo.RetryTask(ctx, id); err != nil {
		return store.Task{}, err
	}
	return p.repo.GetTask(ctx, id)
}

func (p *Pool) run(task store.Task) {
	ctx, cancel := context.WithCancel(context.Background())
	p.mu.Lock()
	p.running[task.ID] = &runningTask{cancel: cancel}
	p.mu.Unlock()
	defer func() {
		p.mu.Lock()
		delete(p.running, task.ID)
		p.mu.Unlock()
		cancel()
		if r := recover(); r != nil {
			p.logger.Error("tasks: handler panic", "task_id", task.ID, "type", task.Type, "panic", r)
			p.finishFailed(task, fmt.Errorf("panic interno do handler"))
		}
	}()

	started := time.Now()
	handler, err := p.registry.Get(task.Type)
	if err != nil {
		p.finishFailed(task, err)
		return
	}

	progressFn := func(n int) {
		p.reportProgress(task, n)
	}
	result, runErr := handler.Run(ctx, json.RawMessage(task.Payload), progressFn)
	duration := time.Since(started)

	if runErr == nil {
		p.finishDone(task, result, duration)
		return
	}
	if errors.Is(runErr, context.Canceled) || ctx.Err() != nil {
		p.finishCancelled(task, duration)
		return
	}
	p.handleFailure(task, runErr, duration)
}

func (p *Pool) reportProgress(task store.Task, n int) {
	if n < 0 {
		n = 0
	}
	if n > 100 {
		n = 100
	}
	if err := p.repo.UpdateTaskProgress(context.Background(), task.ID, n); err != nil {
		// The task may have been cancelled concurrently; progress writes on a
		// non-running row are expected to fail — warn only on real errors.
		p.logger.Debug("tasks: progress update failed", "task_id", task.ID, "err", err)
		return
	}
	p.publish(events.Event{
		Kind:     events.KindTaskProgress,
		Metadata: map[string]any{"task_id": task.ID, "task_type": task.Type, "progress": n},
	})
}

func (p *Pool) finishDone(task store.Task, result json.RawMessage, d time.Duration) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	resultStr := string(result)
	if err := p.repo.FinishTask(ctx, task.ID, "done", resultStr, ""); err != nil {
		p.logger.Error("tasks: finish done failed", "task_id", task.ID, "err", err)
	}
	p.logTask(task, "done", 0, d, "")

	meta := map[string]any{"task_id": task.ID, "task_type": task.Type}
	if resultStr != "" {
		var parsed map[string]any
		if json.Unmarshal(result, &parsed) == nil {
			for _, k := range []string{"job_id", "score_fit", "count", "job_label"} {
				if v, ok := parsed[k]; ok {
					meta[k] = v
				}
			}
		}
	}
	p.publish(events.Event{
		Kind:     events.KindTaskCompleted,
		Severity: events.SeveritySuccess,
		Title:    "Tarefa concluída",
		Body:     completedBody(task.Type, meta),
		Link:     "/tasks",
		Metadata: meta,
	})
}

func (p *Pool) finishCancelled(task store.Task, d time.Duration) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := p.repo.FinishTask(ctx, task.ID, "cancelled", "", "Cancelada."); err != nil {
		p.logger.Error("tasks: finish cancelled failed", "task_id", task.ID, "err", err)
	}
	p.logTask(task, "cancelled", task.Attempts, d, "cancelled by user")
	p.publish(events.Event{
		Kind:     events.KindTaskCancelled,
		Severity: events.SeverityWarning,
		Title:    "Tarefa cancelada",
		Body:     humanType(task.Type) + " cancelada.",
		Metadata: map[string]any{"task_id": task.ID, "task_type": task.Type},
	})
}

// handleFailure applies the retry policy: attempts < max_attempts re-enqueues
// with exponential backoff; exhausted attempts move the task to 'failed' and
// emit a task.failed event (which becomes a task_failed notification).
func (p *Pool) handleFailure(task store.Task, runErr error, d time.Duration) {
	errMsg := legibleError(runErr)
	if task.Attempts < task.MaxAttempts {
		next := time.Now().UTC().Add(p.backoffFn(task.Attempts))
		ctx2, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := p.repo.RequeueTaskWithBackoff(ctx2, task.ID, next, errMsg); err != nil {
			p.logger.Error("tasks: requeue with backoff failed", "task_id", task.ID, "err", err)
			p.finishFailed(task, runErr)
			return
		}
		p.logTask(task, "retry", task.Attempts, d, errMsg)
		p.publish(events.Event{
			Kind:     events.KindTaskProgress, // reuses the SSE channel for state sync
			Metadata: map[string]any{"task_id": task.ID, "task_type": task.Type, "status": "queued", "attempts": task.Attempts},
		})
		return
	}
	p.finishFailed(task, runErr)
}

func (p *Pool) finishFailed(task store.Task, runErr error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	errMsg := legibleError(runErr)
	if err := p.repo.FinishTask(ctx, task.ID, "failed", "", errMsg); err != nil {
		p.logger.Error("tasks: finish failed failed", "task_id", task.ID, "err", err)
	}
	p.logTask(task, "failed", task.Attempts, 0, errMsg)
	p.publish(events.Event{
		Kind:     events.KindTaskFailed,
		Severity: events.SeverityError,
		Title:    "Tarefa de IA falhou",
		Body:     humanType(task.Type) + ": " + errMsg,
		Link:     "/tasks",
		Metadata: map[string]any{"task_id": task.ID, "task_type": task.Type, "error": errMsg},
	})
}

func (p *Pool) publish(e events.Event) {
	if p.bus != nil {
		p.bus.Publish(e)
	}
}

func (p *Pool) logTask(task store.Task, outcome string, attempts int, d time.Duration, errMsg string) {
	attrs := []any{
		"task_id", task.ID,
		"type", task.Type,
		"outcome", outcome,
		"attempts", attempts,
		"duration", d.String(),
	}
	if errMsg != "" {
		attrs = append(attrs, "error", errMsg)
	}
	switch outcome {
	case "failed":
		p.logger.Error("tasks: task failed", attrs...)
	case "retry":
		p.logger.Warn("tasks: task scheduled for retry", attrs...)
	case "cancelled":
		p.logger.Info("tasks: task cancelled", attrs...)
	default:
		p.logger.Info("tasks: task done", attrs...)
	}
}

// humanType maps a task type to a short human label (PT-BR).
func humanType(typ string) string {
	switch typ {
	case TypeScoreJob:
		return "Score de vaga"
	case TypeExtractResumeKeywords:
		return "Extração de palavras-chave"
	case TypeTailorResume:
		return "Adaptação de currículo"
	case TypeDiscoverSources:
		return "Busca de vagas"
	}
	return typ
}

func completedBody(typ string, meta map[string]any) string {
	label, _ := meta["job_label"].(string)
	switch typ {
	case TypeScoreJob:
		score := 0
		if v, ok := meta["score_fit"].(float64); ok {
			score = int(v)
		} else if v, ok := meta["score_fit"].(int); ok {
			score = v
		}
		if label != "" {
			return fmt.Sprintf("%s — score %d/100.", label, score)
		}
		return fmt.Sprintf("Score calculado: %d/100.", score)
	case TypeExtractResumeKeywords:
		count := 0
		if v, ok := meta["count"].(float64); ok {
			count = int(v)
		} else if v, ok := meta["count"].(int); ok {
			count = v
		}
		if count > 0 {
			return fmt.Sprintf("%d palavras-chave extraídas do currículo.", count)
		}
	case TypeDiscoverSources:
		return "Busca de vagas concluída."
	}
	return humanType(typ) + " concluída."
}

// legibleError strips stack noise and bounds the message stored on the task.
func legibleError(err error) string {
	msg := strings.TrimSpace(err.Error())
	const max = 500
	if len(msg) > max {
		msg = msg[:max-3] + "..."
	}
	return msg
}

// EnqueueScoreJob enqueues one vacancy for async scoring — the discovery
// orchestrator's migration path from inline batch scoring (CP6 → CP10).
func (p *Pool) EnqueueScoreJob(ctx context.Context, jobID int64, priority int) (store.Task, error) {
	payload, err := json.Marshal(map[string]any{"job_id": jobID})
	if err != nil {
		return store.Task{}, err
	}
	return p.Enqueue(ctx, TypeScoreJob, payload, priority)
}

// DiscoveryEnqueuer adapts the Pool to discovery.ScoreEnqueuer (which does
// not care about the generated task ID).
type DiscoveryEnqueuer struct{ Pool *Pool }

// EnqueueScoreJob implements discovery.ScoreEnqueuer.
func (e DiscoveryEnqueuer) EnqueueScoreJob(ctx context.Context, jobID int64, priority int) error {
	_, err := e.Pool.EnqueueScoreJob(ctx, jobID, priority)
	return err
}
