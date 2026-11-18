package discovery

import (
	"context"
	"fmt"
	"sync"
)

// Adapter is the common contract implemented by all automated job discovery
// sources (CP6). CP14 also registers respectful scraping adapters; the
// defensive controls live in Capabilities + the ratelimit package, so the
// orchestrator stays source-agnostic.
type Adapter interface {
	Slug() string
	Name() string
	DefaultCapabilities() Capabilities
	Fetch(ctx context.Context, cfg SourceConfig, keywords []string) ([]DiscoveredJob, error)
}

// AdapterInfo exposes metadata of a registered adapter for the API and frontend UI.
type AdapterInfo struct {
	Slug         string       `json:"slug"`
	Name         string       `json:"name"`
	Capabilities Capabilities `json:"capabilities"`
}

// Registry manages the thread-safe set of known discovery adapters.
type Registry struct {
	mu       sync.RWMutex
	adapters map[string]Adapter
}

// NewRegistry instantiates an empty adapter registry.
func NewRegistry() *Registry {
	return &Registry{
		adapters: make(map[string]Adapter),
	}
}

// NewDefaultRegistry instantiates a registry pre-loaded with every supported
// adapter: the CP6 free-API remote boards, the CP14 LATAM API sources and the
// CP14 respectful scraping sources.
func NewDefaultRegistry() *Registry {
	r := NewRegistry()
	// CP6 remote/global APIs.
	r.Register(NewHimalayasAdapter())
	r.Register(NewRemotiveAdapter())
	r.Register(NewJobicyAdapter())
	r.Register(NewRemoteOKAdapter())
	// CP14 LATAM APIs.
	r.Register(NewGetOnBoardAdapter())
	r.Register(NewTramposAdapter())
	r.Register(NewGupyAdapter())
	r.Register(NewManfredAdapter())
	// CP14 scraping (defensive rate limiting + robots constraints).
	r.Register(NewProgramathorAdapter())
	r.Register(NewComputrabajoAdapter())
	r.Register(NewBumeranAdapter())
	r.Register(NewVagasAdapter())
	r.Register(NewCathoAdapter())
	r.Register(NewInfoJobsAdapter())
	return r
}

// Register registers an adapter implementation.
func (r *Registry) Register(a Adapter) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.adapters[a.Slug()] = a
}

// Get retrieves an adapter by its unique slug.
func (r *Registry) Get(slug string) (Adapter, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	a, ok := r.adapters[slug]
	return a, ok
}

// List returns descriptors for all registered adapters.
func (r *Registry) List() []AdapterInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()
	infos := make([]AdapterInfo, 0, len(r.adapters))
	for _, a := range r.adapters {
		infos = append(infos, AdapterInfo{
			Slug:         a.Slug(),
			Name:         a.Name(),
			Capabilities: a.DefaultCapabilities(),
		})
	}
	return infos
}

// Validate validates that a source references an existing adapter in the registry.
func (r *Registry) Validate(slug string) error {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if _, ok := r.adapters[slug]; !ok {
		return fmt.Errorf("adapter %q não está registrado", slug)
	}
	return nil
}
