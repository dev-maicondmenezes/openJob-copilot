package discovery

import (
	"context"
	"fmt"
	"sync"
)

// Adapter is the common contract implemented by all automated job discovery sources.
// Only sources offering public free APIs are registered. No webscraping is performed in this module.
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

// NewDefaultRegistry instantiates a registry pre-loaded with all supported free API adapters.
func NewDefaultRegistry() *Registry {
	r := NewRegistry()
	r.Register(NewHimalayasAdapter())
	r.Register(NewRemotiveAdapter())
	r.Register(NewJobicyAdapter())
	r.Register(NewRemoteOKAdapter())
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
