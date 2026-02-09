package collectors

import (
	"fmt"
	"sort"
	"sync"
)

// Registry manages a set of named collectors. It is safe for concurrent use.
type Registry struct {
	mu         sync.RWMutex
	collectors map[string]Collector
	statuses   map[string]*CollectorStatus
}

// NewRegistry returns an empty registry ready for collector registration.
func NewRegistry() *Registry {
	return &Registry{
		collectors: make(map[string]Collector),
		statuses:   make(map[string]*CollectorStatus),
	}
}

// Register adds a collector to the registry. It returns an error if a
// collector with the same name is already registered.
func (r *Registry) Register(c Collector) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := c.Name()
	if _, exists := r.collectors[name]; exists {
		return fmt.Errorf("collector %q already registered", name)
	}

	r.collectors[name] = c
	r.statuses[name] = &CollectorStatus{
		Name:    name,
		Healthy: true,
	}
	return nil
}

// Unregister removes a collector by name. It is a no-op if the name is not
// found.
func (r *Registry) Unregister(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.collectors, name)
	delete(r.statuses, name)
}

// Get returns the collector with the given name, or false if not found.
func (r *Registry) Get(name string) (Collector, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	c, ok := r.collectors[name]
	return c, ok
}

// List returns a sorted slice of all registered collector names.
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.collectors))
	for name := range r.collectors {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// Status returns a copy of the runtime status for the named collector, or
// false if the collector is not registered.
func (r *Registry) Status(name string) (CollectorStatus, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	s, ok := r.statuses[name]
	if !ok {
		return CollectorStatus{}, false
	}
	return *s, true
}

// AllStatus returns a copy of all collector statuses, sorted by name.
func (r *Registry) AllStatus() []CollectorStatus {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]CollectorStatus, 0, len(r.statuses))
	for _, s := range r.statuses {
		result = append(result, *s)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})
	return result
}

// updateStatus updates the status entry for the named collector. Caller must
// NOT hold the lock; this method acquires it.
func (r *Registry) updateStatus(name string, fn func(s *CollectorStatus)) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if s, ok := r.statuses[name]; ok {
		fn(s)
	}
}
