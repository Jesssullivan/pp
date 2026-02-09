package image

import (
	"image"
	"sync"
)

// defaultWorkers is the number of concurrent render goroutines.
const defaultWorkers = 2

// renderJob is an internal unit of work for the async pool.
type renderJob struct {
	img      image.Image
	width    int
	height   int
	callback func(string, error)
}

// AsyncRenderer manages a bounded goroutine pool for non-blocking image
// rendering. It is designed for TUI event loops where rendering must not
// block the main thread.
type AsyncRenderer struct {
	renderer *Renderer
	jobs     chan renderJob
	wg       sync.WaitGroup
	stopOnce sync.Once
	stop     chan struct{}
}

// NewAsyncRenderer creates an async wrapper around a Renderer with a
// bounded goroutine pool. The pool starts immediately.
func NewAsyncRenderer(r *Renderer) *AsyncRenderer {
	return NewAsyncRendererWithWorkers(r, defaultWorkers)
}

// NewAsyncRendererWithWorkers creates an async renderer with a specific
// number of workers.
func NewAsyncRendererWithWorkers(r *Renderer, workers int) *AsyncRenderer {
	if workers <= 0 {
		workers = defaultWorkers
	}

	ar := &AsyncRenderer{
		renderer: r,
		jobs:     make(chan renderJob, workers*4),
		stop:     make(chan struct{}),
	}

	for i := 0; i < workers; i++ {
		ar.wg.Add(1)
		go ar.worker()
	}

	return ar
}

// RenderAsync submits an image for asynchronous rendering. The callback is
// invoked from a worker goroutine when rendering completes or fails. Returns
// a cancel function that prevents the callback from being called (best-effort;
// if the render has already started, it will complete but the callback will
// still fire).
//
// This method never blocks the caller beyond channel send.
func (ar *AsyncRenderer) RenderAsync(img image.Image, width, height int, callback func(string, error)) func() {
	cancelled := make(chan struct{})

	wrappedCallback := func(result string, err error) {
		select {
		case <-cancelled:
			// Cancelled; do not invoke user callback.
			return
		default:
			callback(result, err)
		}
	}

	job := renderJob{
		img:      img,
		width:    width,
		height:   height,
		callback: wrappedCallback,
	}

	// Non-blocking send: if the job queue is full, run synchronously in a
	// new goroutine to avoid blocking the TUI loop.
	select {
	case ar.jobs <- job:
	default:
		go func() {
			result, err := ar.renderer.Render(img, width, height)
			wrappedCallback(result, err)
		}()
	}

	return func() {
		close(cancelled)
	}
}

// Close shuts down the worker pool. It signals all workers to stop and
// waits for in-flight jobs to complete.
func (ar *AsyncRenderer) Close() {
	ar.stopOnce.Do(func() {
		close(ar.stop)
		close(ar.jobs)
		ar.wg.Wait()
	})
}

// worker processes jobs from the queue until the pool is closed.
func (ar *AsyncRenderer) worker() {
	defer ar.wg.Done()

	for {
		select {
		case <-ar.stop:
			// Drain remaining jobs before exiting.
			for job := range ar.jobs {
				result, err := ar.renderer.Render(job.img, job.width, job.height)
				job.callback(result, err)
			}
			return
		case job, ok := <-ar.jobs:
			if !ok {
				return
			}
			result, err := ar.renderer.Render(job.img, job.width, job.height)
			job.callback(result, err)
		}
	}
}
