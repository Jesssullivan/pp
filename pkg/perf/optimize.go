// Package perf provides performance benchmarks, optimization utilities, and
// regression detection for the prompt-pulse dashboard's critical rendering
// paths. All private helpers are prefixed with "pf" to avoid naming conflicts.
package perf

import (
	"fmt"
	"strings"
	"sync"
)

// StringPool provides a sync.Pool for strings.Builder to reduce allocations
// in hot rendering paths. Builders are Reset before being returned to the
// pool, so callers always receive an empty builder.
type StringPool struct {
	pool sync.Pool
}

// NewStringPool creates a StringPool with a factory that creates new
// strings.Builder instances.
func NewStringPool() *StringPool {
	return &StringPool{
		pool: sync.Pool{
			New: func() interface{} {
				return &strings.Builder{}
			},
		},
	}
}

// Get retrieves a strings.Builder from the pool. The builder is guaranteed
// to be empty (Reset has been called). Callers must call Put when done to
// return the builder to the pool.
func (p *StringPool) Get() *strings.Builder {
	return p.pool.Get().(*strings.Builder)
}

// Put returns a strings.Builder to the pool after resetting it. The builder
// must not be used after calling Put.
func (p *StringPool) Put(b *strings.Builder) {
	b.Reset()
	p.pool.Put(b)
}

// PreallocBuilder returns a pointer to a strings.Builder pre-allocated to the
// given capacity. This avoids repeated growth when the approximate output size
// is known in advance. A pointer is returned because strings.Builder cannot be
// copied after its first write (including Grow).
func PreallocBuilder(capacity int) *strings.Builder {
	if capacity < 0 {
		capacity = 0
	}
	b := &strings.Builder{}
	b.Grow(capacity)
	return b
}

// WidgetRenderTask describes a single widget rendering job for BatchRender.
type WidgetRenderTask struct {
	// Render is the function that produces the widget's display string.
	// It receives the allocated width and height in character cells.
	Render func(w, h int) string

	// Width is the allocated width in character cells.
	Width int

	// Height is the allocated height in character cells.
	Height int
}

// BatchRender renders multiple widgets in parallel using a worker pool of up
// to maxWorkers goroutines. Results are returned in the same order as the
// input tasks. If a render function panics, the corresponding result slot
// contains a panic message string instead of crashing the entire batch.
//
// If maxWorkers <= 0, it defaults to 1 (serial execution).
// If tasks is empty, an empty slice is returned.
func BatchRender(tasks []WidgetRenderTask, maxWorkers int) []string {
	if len(tasks) == 0 {
		return []string{}
	}
	if maxWorkers <= 0 {
		maxWorkers = 1
	}

	results := make([]string, len(tasks))

	if maxWorkers == 1 {
		// Fast path: no goroutine overhead for serial execution.
		for i, t := range tasks {
			results[i] = pfSafeRender(t)
		}
		return results
	}

	// Parallel path with bounded worker pool.
	var wg sync.WaitGroup
	sem := make(chan struct{}, maxWorkers)

	for i, t := range tasks {
		wg.Add(1)
		sem <- struct{}{} // acquire semaphore slot
		go func(idx int, task WidgetRenderTask) {
			defer wg.Done()
			defer func() { <-sem }() // release semaphore slot
			results[idx] = pfSafeRender(task)
		}(i, t)
	}

	wg.Wait()
	return results
}

// pfSafeRender calls a WidgetRenderTask's Render function, recovering from
// panics and returning an error message instead.
func pfSafeRender(t WidgetRenderTask) (result string) {
	defer func() {
		if r := recover(); r != nil {
			result = fmt.Sprintf("[render panic: %v]", r)
		}
	}()

	if t.Render == nil {
		return "[nil render func]"
	}
	return t.Render(t.Width, t.Height)
}
