package collectors

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

// MockCollector implements Collector for testing. All fields are configurable
// and it tracks how many times Collect has been called.
type MockCollector struct {
	name     string
	interval time.Duration
	data     interface{}
	err      error
	healthy  bool

	mu        sync.RWMutex
	callCount atomic.Int64

	// CollectFunc, if set, overrides the default Collect behavior.
	// This allows tests to inject dynamic behavior (e.g., return different
	// data on each call, or block until a signal).
	CollectFunc func(ctx context.Context) (interface{}, error)
}

// MockCollectorOption configures a MockCollector.
type MockCollectorOption func(*MockCollector)

// WithData sets the data returned by Collect.
func WithData(data interface{}) MockCollectorOption {
	return func(m *MockCollector) { m.data = data }
}

// WithError sets the error returned by Collect.
func WithError(err error) MockCollectorOption {
	return func(m *MockCollector) { m.err = err }
}

// WithHealthy sets the Healthy() return value.
func WithHealthy(healthy bool) MockCollectorOption {
	return func(m *MockCollector) { m.healthy = healthy }
}

// WithCollectFunc sets a custom function for Collect.
func WithCollectFunc(fn func(ctx context.Context) (interface{}, error)) MockCollectorOption {
	return func(m *MockCollector) { m.CollectFunc = fn }
}

// NewMockCollector creates a mock collector with the given name, interval,
// and options.
func NewMockCollector(name string, interval time.Duration, opts ...MockCollectorOption) *MockCollector {
	m := &MockCollector{
		name:     name,
		interval: interval,
		healthy:  true,
	}
	for _, opt := range opts {
		opt(m)
	}
	return m
}

// Name returns the collector name.
func (m *MockCollector) Name() string { return m.name }

// Interval returns the configured collection interval.
func (m *MockCollector) Interval() time.Duration { return m.interval }

// Healthy returns the configured health status.
func (m *MockCollector) Healthy() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.healthy
}

// SetHealthy updates the health status (thread-safe).
func (m *MockCollector) SetHealthy(h bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.healthy = h
}

// SetData updates the returned data (thread-safe).
func (m *MockCollector) SetData(data interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data = data
}

// SetError updates the returned error (thread-safe).
func (m *MockCollector) SetError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.err = err
}

// Collect performs a mock collection. It increments the call counter and
// returns the configured data and error, or delegates to CollectFunc if set.
func (m *MockCollector) Collect(ctx context.Context) (interface{}, error) {
	m.callCount.Add(1)

	if m.CollectFunc != nil {
		return m.CollectFunc(ctx)
	}

	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.data, m.err
}

// CallCount returns how many times Collect has been called.
func (m *MockCollector) CallCount() int64 {
	return m.callCount.Load()
}
