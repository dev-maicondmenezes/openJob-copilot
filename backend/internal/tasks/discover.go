package tasks

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"

	"github.com/maicondmenezes/openjob/internal/discovery"
)

// DiscoverHandler runs one full discovery execution from the durable queue
// (CP14). The mutex serializes executions even when the pool has more than one
// worker, so ten quick "Buscar agora" clicks queue up and run one after the
// other — never in parallel against the same rate-limited sources.
type DiscoverHandler struct {
	orch   *discovery.Orchestrator
	logger *slog.Logger
	mu     sync.Mutex
}

func NewDiscoverHandler(orch *discovery.Orchestrator, logger *slog.Logger) *DiscoverHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &DiscoverHandler{orch: orch, logger: logger}
}

func (h *DiscoverHandler) Type() string { return TypeDiscoverSources }

func (h *DiscoverHandler) Run(ctx context.Context, _ json.RawMessage, progress func(int)) (json.RawMessage, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	progress(5)
	runID, stats, err := h.orch.Discover(ctx)
	if err != nil {
		return nil, fmt.Errorf("discovery run %d: %w", runID, err)
	}
	progress(100)

	result, _ := json.Marshal(map[string]any{
		"run_id":   runID,
		"fetched":  stats.Fetched,
		"new_jobs": stats.NewJobs,
		"scored":   stats.Scored,
		"errors":   stats.Errors,
	})
	return result, nil
}
