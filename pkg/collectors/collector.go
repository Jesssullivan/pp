// Package collectors defines the interfaces, registry, and runner for
// prompt-pulse data collectors. Each collector (sysmetrics, tailscale, k8s,
// claude, billing) implements the Collector interface and is orchestrated by
// a Runner that fans results into a single updates channel consumed by the TUI.
package collectors

import (
	"context"
	"time"
)

// Collector is the interface all data sources implement. Implementations live
// in sub-packages (e.g., pkg/collectors/sysmetrics) and are registered with
// the Registry at startup.
type Collector interface {
	// Name returns a unique identifier for this collector (e.g., "sysmetrics").
	Name() string

	// Collect performs one collection cycle and returns the data. The returned
	// value is opaque here; consumers type-assert based on the collector name.
	Collect(ctx context.Context) (interface{}, error)

	// Interval returns how often this collector should run. The runner uses
	// this to configure a per-collector ticker.
	Interval() time.Duration

	// Healthy returns whether the collector is functioning. A collector that
	// has never run or whose last run succeeded is considered healthy.
	Healthy() bool
}

// CollectorStatus tracks the runtime state of a single collector. The runner
// updates this after every collection cycle.
type CollectorStatus struct {
	Name        string
	Healthy     bool
	LastRun     time.Time
	LastError   error
	RunCount    int64
	ErrorCount  int64
	LastLatency time.Duration
}

// Update carries the result of a single collection cycle from a collector
// goroutine to the consumer (typically the TUI event loop).
type Update struct {
	Source    string
	Data     interface{}
	Timestamp time.Time
	Error    error
}
