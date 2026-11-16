// Package events is the in-process event bus (CP10, docs/adr/0009): producers
// (application pipeline, discovery orchestrator, task workers) publish domain
// events; consumers (in-app notifier, future Telegram/e-mail channels) react.
// The bus is intentionally simple — no external queue — because openjob is a
// local-first single-process app. Publishing is best-effort and non-blocking:
// a slow or full subscriber never breaks the producer's main flow.
package events

import (
	"sync"
)

// Severity drives icon/color in the notification UI.
const (
	SeverityInfo    = "info"
	SeveritySuccess = "success"
	SeverityWarning = "warning"
	SeverityError   = "error"
)

// Event kinds published by the producers.
const (
	KindApplicationStatusChanged = "application.status_changed"
	KindDiscoveryCompleted       = "discovery.completed"
	KindDiscoveryFailed          = "discovery.failed"
	KindSourceFetchFailed        = "source.fetch_failed"
	KindTaskCompleted            = "task.completed"
	KindTaskFailed               = "task.failed"
	KindTaskCancelled            = "task.cancelled"
	KindTaskProgress             = "task.progress"
	KindTest                     = "test"
)

// Event is a domain event on the bus.
type Event struct {
	Kind     string
	Severity string // info | success | warning | error
	Title    string
	Body     string
	Link     string // optional internal SPA deep link
	// DedupKey, when non-empty, lets consumers coalesce repeated events (e.g.
	// source_error for the same source on every discovery tick) within a
	// short window instead of spamming the feed.
	DedupKey string
	Metadata map[string]any // optional audit payload
}

// Subscription receives events published after Subscribe was called.
type Subscription struct {
	name string
	ch   chan Event
	bus  *Bus
}

// C returns the event channel. It is closed by Close (or when the bus is
// closed); consumers must range/select until the channel is closed.
func (s *Subscription) C() <-chan Event { return s.ch }

// Close unsubscribes and closes the channel.
func (s *Subscription) Close() {
	s.bus.mu.Lock()
	defer s.bus.mu.Unlock()
	if _, ok := s.bus.subs[s.name]; ok {
		delete(s.bus.subs, s.name)
		close(s.ch)
	}
}

// Bus is the fan-out point of the app. A zero-value Bus is not usable; use
// New.
type Bus struct {
	mu   sync.Mutex
	subs map[string]*Subscription
	seq  int
}

// New creates an empty bus.
func New() *Bus {
	return &Bus{subs: map[string]*Subscription{}}
}

// Subscribe registers a consumer. buf is the per-subscriber channel size;
// Publish drops events for a subscriber whose buffer is full (the in-app
// notifier persists asynchronously anyway, and SSE clients fall back to
// polling — losing a transient event is preferable to blocking producers).
func (b *Bus) Subscribe(name string, buf int) *Subscription {
	if buf <= 0 {
		buf = 32
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	// Replace a previous subscription with the same name (idempotent start).
	if old, ok := b.subs[name]; ok {
		delete(b.subs, name)
		close(old.ch)
	}
	b.seq++
	sub := &Subscription{name: name, ch: make(chan Event, buf), bus: b}
	b.subs[name] = sub
	return sub
}

// Publish delivers the event to every subscriber without ever blocking:
// full subscriber buffers drop the event (best-effort semantics, spec CP10).
func (b *Bus) Publish(e Event) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for _, sub := range b.subs {
		select {
		case sub.ch <- e:
		default: // subscriber slow: drop instead of blocking producers
		}
	}
}

// Close unsubscribes everyone (used on shutdown).
func (b *Bus) Close() {
	b.mu.Lock()
	defer b.mu.Unlock()
	for name, sub := range b.subs {
		delete(b.subs, name)
		close(sub.ch)
	}
}
