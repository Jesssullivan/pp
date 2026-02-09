package waifu

import (
	"context"
	"sync"
	"time"
)

// ImageRenderer is the interface for rendering an image file to a terminal
// escape string. It matches the signature of pkg/image.Renderer.RenderFile.
type ImageRenderer interface {
	RenderFile(path string, width, height int) (string, error)
}

// PrefetchResult holds the outcome of a background prefetch operation.
type PrefetchResult struct {
	// Rendered is the terminal escape string for the image.
	Rendered string

	// Error is non-nil if rendering failed.
	Error error

	// FromCache is true if the result was served from disk cache.
	FromCache bool

	// Duration is how long the operation took.
	Duration time.Duration
}

// Prefetcher manages background image rendering with WaitGroup-tracked
// goroutines. This fixes the v1 bug where prefetch goroutines were orphaned
// because they were not added to a WaitGroup.
type Prefetcher struct {
	cache    *ImageCache
	renderer ImageRenderer
	wg       sync.WaitGroup
	cancel   context.CancelFunc
	ctx      context.Context
}

// NewPrefetcher creates a Prefetcher that uses the given cache and renderer.
func NewPrefetcher(cache *ImageCache, renderer ImageRenderer) *Prefetcher {
	ctx, cancel := context.WithCancel(context.Background())
	return &Prefetcher{
		cache:    cache,
		renderer: renderer,
		cancel:   cancel,
		ctx:      ctx,
	}
}

// Prefetch starts a background render for the given session's image at the
// specified dimensions. It returns a channel that receives exactly one
// PrefetchResult when the operation completes.
//
// If the result is already in the disk cache, it is returned immediately
// (synchronously, before the goroutine would start). Otherwise, a tracked
// goroutine is spawned to render and cache the result.
func (p *Prefetcher) Prefetch(session *Session, width, height int) <-chan PrefetchResult {
	ch := make(chan PrefetchResult, 1)

	key := CacheKey{
		ContentHash: session.ContentHash,
		Protocol:    "default",
		Width:       width,
		Height:      height,
	}

	// Fast path: cache hit.
	if rendered, ok := p.cache.Get(key); ok {
		ch <- PrefetchResult{
			Rendered:  rendered,
			FromCache: true,
			Duration:  0,
		}
		return ch
	}

	// Slow path: render in background goroutine tracked by WaitGroup.
	p.wg.Add(1)
	go func() {
		defer p.wg.Done()

		start := time.Now()

		// Check for cancellation before rendering.
		select {
		case <-p.ctx.Done():
			ch <- PrefetchResult{
				Error:    p.ctx.Err(),
				Duration: time.Since(start),
			}
			return
		default:
		}

		rendered, err := p.renderer.RenderFile(session.ImagePath, width, height)
		dur := time.Since(start)

		if err != nil {
			ch <- PrefetchResult{
				Error:    err,
				Duration: dur,
			}
			return
		}

		// Store in cache (best effort).
		_ = p.cache.Put(key, rendered)

		ch <- PrefetchResult{
			Rendered:  rendered,
			FromCache: false,
			Duration:  dur,
		}
	}()

	return ch
}

// PrefetchWithProtocol starts a background render with an explicit protocol
// name for the cache key. This is used when the caller knows which terminal
// protocol is active.
func (p *Prefetcher) PrefetchWithProtocol(session *Session, protocol string, width, height int) <-chan PrefetchResult {
	ch := make(chan PrefetchResult, 1)

	key := CacheKey{
		ContentHash: session.ContentHash,
		Protocol:    protocol,
		Width:       width,
		Height:      height,
	}

	if rendered, ok := p.cache.Get(key); ok {
		ch <- PrefetchResult{
			Rendered:  rendered,
			FromCache: true,
			Duration:  0,
		}
		return ch
	}

	p.wg.Add(1)
	go func() {
		defer p.wg.Done()

		start := time.Now()

		select {
		case <-p.ctx.Done():
			ch <- PrefetchResult{
				Error:    p.ctx.Err(),
				Duration: time.Since(start),
			}
			return
		default:
		}

		rendered, err := p.renderer.RenderFile(session.ImagePath, width, height)
		dur := time.Since(start)

		if err != nil {
			ch <- PrefetchResult{
				Error:    err,
				Duration: dur,
			}
			return
		}

		_ = p.cache.Put(key, rendered)

		ch <- PrefetchResult{
			Rendered:  rendered,
			FromCache: false,
			Duration:  dur,
		}
	}()

	return ch
}

// Wait blocks until all in-flight prefetch goroutines have completed.
func (p *Prefetcher) Wait() {
	p.wg.Wait()
}

// Close cancels all pending prefetch work and waits for in-flight goroutines
// to finish.
func (p *Prefetcher) Close() {
	p.cancel()
	p.wg.Wait()
}
