// Package tasks implements the durable asynchronous task queue for AI work
// (CP10, docs/adr/0010): a SQLite-backed queue with an atomic claim
// (UPDATE ... RETURNING), a configurable worker pool, exponential-backoff
// retries, cancellation through context propagation and an in-process event
// feed for the UI (SSE). Handlers register by type, mirroring the adapter
// registry of the discovery package.
package tasks

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
)

// Task types (extensible; tailor_resume lands with the CP4 integration).
const (
	TypeScoreJob              = "score_job"
	TypeExtractResumeKeywords = "extract_resume_keywords"
	TypeTailorResume          = "tailor_resume"
	TypeDiscoverSources       = "discover_sources" // CP14: queued discovery runs
)

// Handler runs one task type. Handlers MUST check ctx between steps: a
// cancelled running task relies on ctx propagation (POST /api/tasks/{id}/cancel).
type Handler interface {
	Type() string
	// Run executes the task and reports multi-step progress (0-100). The
	// returned JSON is persisted as the task result.
	Run(ctx context.Context, payload json.RawMessage, progress func(int)) (json.RawMessage, error)
}

// ErrUnknownType is returned by Registry.Get for unregistered task types.
var ErrUnknownType = errors.New("tasks: unknown handler type")

// Registry maps task types to handlers.
type Registry struct {
	mu       sync.RWMutex
	handlers map[string]Handler
}

// NewRegistry creates an empty registry.
func NewRegistry() *Registry {
	return &Registry{handlers: map[string]Handler{}}
}

// Register adds a handler; registering the same type twice replaces it.
func (r *Registry) Register(h Handler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.handlers[h.Type()] = h
}

// Get returns the handler for a task type.
func (r *Registry) Get(typ string) (Handler, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	h, ok := r.handlers[typ]
	if !ok {
		return nil, fmt.Errorf("%w: %q", ErrUnknownType, typ)
	}
	return h, nil
}
