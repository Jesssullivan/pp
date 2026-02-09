package image

import (
	"bytes"
	"compress/zlib"
	"container/list"
	"encoding/base64"
	"image"
	"image/color"
	"image/draw"
	"io"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
	"unicode/utf8"

	"gitlab.com/tinyland/lab/prompt-pulse/pkg/config"
	"gitlab.com/tinyland/lab/prompt-pulse/pkg/terminal"
)

// --- helpers ---------------------------------------------------------------

// makeImage creates a solid-colored NRGBA test image.
func makeImage(w, h int, c color.Color) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	draw.Draw(img, img.Bounds(), &image.Uniform{c}, image.Point{}, draw.Src)
	return img
}

// makeGradientImage creates a gradient image for testing uniqueness.
func makeGradientImage(w, h int) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetNRGBA(x, y, color.NRGBA{
				R: uint8(x * 255 / max(w-1, 1)),
				G: uint8(y * 255 / max(h-1, 1)),
				B: 128,
				A: 255,
			})
		}
	}
	return img
}

// makeCaps builds terminal.Capabilities for testing.
func makeCaps(proto terminal.GraphicsProtocol) terminal.Capabilities {
	return terminal.Capabilities{
		Term:      terminal.TermGhostty,
		Protocol:  proto,
		TrueColor: true,
		Size: terminal.Size{
			Cols:   80,
			Rows:   24,
			PixelW: 640,
			PixelH: 384,
			CellW:  8,
			CellH:  16,
		},
	}
}

// makeCfg builds a minimal ImageConfig.
func makeCfg() config.ImageConfig {
	return config.ImageConfig{
		MaxCacheSizeMB: 8,
	}
}

// --- Protocol auto-detection tests -----------------------------------------

func TestProtocolAutoDetectKitty(t *testing.T) {
	caps := makeCaps(terminal.ProtocolKitty)
	r := NewRenderer(caps, makeCfg())
	if r.Protocol() != terminal.ProtocolKitty {
		t.Errorf("expected ProtocolKitty, got %s", r.Protocol())
	}
}

func TestProtocolAutoDetectITerm2(t *testing.T) {
	caps := makeCaps(terminal.ProtocolITerm2)
	r := NewRenderer(caps, makeCfg())
	if r.Protocol() != terminal.ProtocolITerm2 {
		t.Errorf("expected ProtocolITerm2, got %s", r.Protocol())
	}
}

func TestProtocolAutoDetectSixel(t *testing.T) {
	caps := makeCaps(terminal.ProtocolSixel)
	r := NewRenderer(caps, makeCfg())
	if r.Protocol() != terminal.ProtocolSixel {
		t.Errorf("expected ProtocolSixel, got %s", r.Protocol())
	}
}

func TestProtocolAutoDetectHalfblocks(t *testing.T) {
	caps := makeCaps(terminal.ProtocolHalfblocks)
	r := NewRenderer(caps, makeCfg())
	if r.Protocol() != terminal.ProtocolHalfblocks {
		t.Errorf("expected ProtocolHalfblocks, got %s", r.Protocol())
	}
}

func TestProtocolConfigOverride(t *testing.T) {
	caps := makeCaps(terminal.ProtocolKitty) // would auto-detect kitty

	cfg := makeCfg()
	cfg.Protocol = "halfblocks"
	r := NewRenderer(caps, cfg)

	if r.Protocol() != terminal.ProtocolHalfblocks {
		t.Errorf("config override failed: expected ProtocolHalfblocks, got %s", r.Protocol())
	}
}

func TestProtocolAutoConfigDoesNotOverride(t *testing.T) {
	caps := makeCaps(terminal.ProtocolKitty)

	cfg := makeCfg()
	cfg.Protocol = "auto"
	r := NewRenderer(caps, cfg)

	if r.Protocol() != terminal.ProtocolKitty {
		t.Errorf("auto should not override: expected ProtocolKitty, got %s", r.Protocol())
	}
}

func TestProtocolNoneDisablesRendering(t *testing.T) {
	cfg := makeCfg()
	cfg.Protocol = "none"
	r := NewRenderer(makeCaps(terminal.ProtocolKitty), cfg)

	_, err := r.Render(makeImage(4, 4, color.White), 10, 10)
	if err == nil {
		t.Error("expected error for ProtocolNone, got nil")
	}
}

// --- Cache tests -----------------------------------------------------------

func TestCacheGetMiss(t *testing.T) {
	c := NewCache(1)
	key := MakeCacheKey("halfblocks", 10, 10, [32]byte{1})
	_, ok := c.Get(key)
	if ok {
		t.Error("expected cache miss for empty cache")
	}
}

func TestCachePutAndGet(t *testing.T) {
	c := NewCache(1)
	key := MakeCacheKey("halfblocks", 10, 10, [32]byte{1})
	c.Put(key, "rendered-output")

	val, ok := c.Get(key)
	if !ok {
		t.Fatal("expected cache hit after put")
	}
	if val != "rendered-output" {
		t.Errorf("expected 'rendered-output', got %q", val)
	}
}

func TestCacheHitMissCounters(t *testing.T) {
	c := NewCache(1)
	key := MakeCacheKey("halfblocks", 10, 10, [32]byte{1})
	missKey := MakeCacheKey("halfblocks", 20, 20, [32]byte{2})

	c.Put(key, "data")
	c.Get(key)      // hit
	c.Get(key)      // hit
	c.Get(missKey)  // miss

	stats := c.Stats()
	if stats.Hits != 2 {
		t.Errorf("expected 2 hits, got %d", stats.Hits)
	}
	if stats.Misses != 1 {
		t.Errorf("expected 1 miss, got %d", stats.Misses)
	}
	if stats.Entries != 1 {
		t.Errorf("expected 1 entry, got %d", stats.Entries)
	}
}

func TestCacheEviction(t *testing.T) {
	// Very small cache: 1 byte max.
	c := &Cache{
		items:    make(map[CacheKey]*list.Element),
		order:    list.New(),
		maxBytes: 100,
	}

	key1 := MakeCacheKey("hb", 1, 1, [32]byte{1})
	key2 := MakeCacheKey("hb", 2, 2, [32]byte{2})

	// Put 60 bytes.
	c.Put(key1, strings.Repeat("a", 60))
	// Put 60 more bytes - should evict key1.
	c.Put(key2, strings.Repeat("b", 60))

	stats := c.Stats()
	if stats.Evictions != 1 {
		t.Errorf("expected 1 eviction, got %d", stats.Evictions)
	}

	_, ok := c.Get(key1)
	if ok {
		t.Error("key1 should have been evicted")
	}

	val, ok := c.Get(key2)
	if !ok {
		t.Error("key2 should still be in cache")
	}
	if len(val) != 60 {
		t.Errorf("expected 60 bytes, got %d", len(val))
	}
}

func TestCacheLRUOrder(t *testing.T) {
	c := &Cache{
		items:    make(map[CacheKey]*list.Element),
		order:    list.New(),
		maxBytes: 150,
	}

	key1 := MakeCacheKey("hb", 1, 1, [32]byte{1})
	key2 := MakeCacheKey("hb", 2, 2, [32]byte{2})
	key3 := MakeCacheKey("hb", 3, 3, [32]byte{3})

	c.Put(key1, strings.Repeat("a", 50))
	c.Put(key2, strings.Repeat("b", 50))
	c.Put(key3, strings.Repeat("c", 50))

	// All three fit. Now access key1 to make it most recent.
	c.Get(key1)

	// Add key4 which requires eviction.
	key4 := MakeCacheKey("hb", 4, 4, [32]byte{4})
	c.Put(key4, strings.Repeat("d", 50))

	// key2 was LRU (key1 was promoted), so key2 should be evicted.
	_, ok := c.Get(key2)
	if ok {
		t.Error("key2 should have been evicted (LRU)")
	}

	// key1 should still be present (was promoted).
	_, ok = c.Get(key1)
	if !ok {
		t.Error("key1 should still be in cache (recently accessed)")
	}
}

func TestCacheInvalidate(t *testing.T) {
	c := NewCache(1)
	key := MakeCacheKey("hb", 1, 1, [32]byte{1})
	c.Put(key, "data")
	c.Invalidate()

	_, ok := c.Get(key)
	if ok {
		t.Error("expected miss after invalidation")
	}

	stats := c.Stats()
	if stats.Entries != 0 {
		t.Errorf("expected 0 entries after invalidation, got %d", stats.Entries)
	}
}

func TestCacheUpdateExistingKey(t *testing.T) {
	c := NewCache(1)
	key := MakeCacheKey("hb", 1, 1, [32]byte{1})

	c.Put(key, "old")
	c.Put(key, "new")

	val, ok := c.Get(key)
	if !ok {
		t.Fatal("expected hit")
	}
	if val != "new" {
		t.Errorf("expected 'new', got %q", val)
	}

	stats := c.Stats()
	if stats.Entries != 1 {
		t.Errorf("expected 1 entry after update, got %d", stats.Entries)
	}
}

func TestCacheKeyStability(t *testing.T) {
	hash := [32]byte{0xde, 0xad, 0xbe, 0xef}
	k1 := MakeCacheKey("kitty", 40, 20, hash)
	k2 := MakeCacheKey("kitty", 40, 20, hash)

	if k1 != k2 {
		t.Error("identical inputs should produce identical cache keys")
	}

	k3 := MakeCacheKey("kitty", 40, 21, hash)
	if k1 == k3 {
		t.Error("different heights should produce different cache keys")
	}
}

func TestCacheConcurrency(t *testing.T) {
	c := NewCache(4)
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			key := MakeCacheKey("hb", n%10, n%5, [32]byte{byte(n % 256)})
			c.Put(key, strings.Repeat("x", n%100))
			c.Get(key)
		}(i)
	}

	wg.Wait()

	stats := c.Stats()
	if stats.Entries < 0 {
		t.Error("negative entry count after concurrent access")
	}
}

// --- Resize tests ----------------------------------------------------------

func TestResizeToFitMaintainsAspectRatio(t *testing.T) {
	// 200x100 image into 10x10 cells (80x160 pixels).
	img := makeImage(200, 100, color.White)
	resized := ResizeToFit(img, 10, 10, 8, 16)

	bounds := resized.Bounds()
	w, h := bounds.Dx(), bounds.Dy()

	// Should be constrained by width: 80px wide.
	// Height at aspect ratio 2:1 -> 40px.
	if w > 80 {
		t.Errorf("width %d exceeds max pixel width 80", w)
	}
	if h > 160 {
		t.Errorf("height %d exceeds max pixel height 160", h)
	}

	// Aspect ratio should be approximately 2:1.
	ratio := float64(w) / float64(h)
	if ratio < 1.5 || ratio > 2.5 {
		t.Errorf("aspect ratio %f is too far from expected 2.0 (w=%d, h=%d)", ratio, w, h)
	}
}

func TestResizeToFitLandscape(t *testing.T) {
	// Wide landscape image: 400x100.
	img := makeImage(400, 100, color.NRGBA{R: 255, A: 255})
	resized := ResizeToFit(img, 20, 10, 8, 16)

	bounds := resized.Bounds()
	if bounds.Dx() > 160 { // 20*8
		t.Errorf("landscape width %d exceeds max 160", bounds.Dx())
	}
	if bounds.Dy() > 160 { // 10*16
		t.Errorf("landscape height %d exceeds max 160", bounds.Dy())
	}
}

func TestResizeToFitPortrait(t *testing.T) {
	// Tall portrait image: 100x400.
	img := makeImage(100, 400, color.NRGBA{G: 255, A: 255})
	resized := ResizeToFit(img, 20, 10, 8, 16)

	bounds := resized.Bounds()
	if bounds.Dx() > 160 {
		t.Errorf("portrait width %d exceeds max 160", bounds.Dx())
	}
	if bounds.Dy() > 160 {
		t.Errorf("portrait height %d exceeds max 160", bounds.Dy())
	}
}

func TestResizeToFitSquare(t *testing.T) {
	img := makeImage(300, 300, color.NRGBA{B: 255, A: 255})
	resized := ResizeToFit(img, 10, 10, 8, 16)

	bounds := resized.Bounds()
	if bounds.Dx() > 80 {
		t.Errorf("square width %d exceeds max 80", bounds.Dx())
	}
	if bounds.Dy() > 160 {
		t.Errorf("square height %d exceeds max 160", bounds.Dy())
	}
}

func TestResizeToFitSmallImageNoUpscale(t *testing.T) {
	// 4x4 image into 100x100 cells - should NOT upscale.
	img := makeImage(4, 4, color.White)
	resized := ResizeToFit(img, 100, 100, 8, 16)

	bounds := resized.Bounds()
	if bounds.Dx() != 4 || bounds.Dy() != 4 {
		t.Errorf("small image should not be upscaled: got %dx%d", bounds.Dx(), bounds.Dy())
	}
}

func TestResizeToFitNilImage(t *testing.T) {
	result := ResizeToFit(nil, 10, 10, 8, 16)
	if result != nil {
		t.Error("nil input should return nil")
	}
}

func TestResizeToFitZeroDimensions(t *testing.T) {
	img := makeImage(100, 100, color.White)

	// Zero width cells should clamp to 1.
	r1 := ResizeToFit(img, 0, 10, 8, 16)
	if r1.Bounds().Dx() > 8 { // 1*8
		t.Errorf("zero width cells: got width %d, expected <= 8", r1.Bounds().Dx())
	}

	// Zero height cells should clamp to 1.
	r2 := ResizeToFit(img, 10, 0, 8, 16)
	if r2.Bounds().Dy() > 16 { // 1*16
		t.Errorf("zero height cells: got height %d, expected <= 16", r2.Bounds().Dy())
	}
}

func TestResizeToFitZeroCellSize(t *testing.T) {
	img := makeImage(100, 100, color.White)

	// Zero cellW/cellH should use defaults (8, 16).
	resized := ResizeToFit(img, 10, 5, 0, 0)
	bounds := resized.Bounds()

	if bounds.Dx() > 80 { // 10*8 default
		t.Errorf("zero cellW: width %d exceeds default max 80", bounds.Dx())
	}
	if bounds.Dy() > 80 { // 5*16 default
		t.Errorf("zero cellH: height %d exceeds default max 80", bounds.Dy())
	}
}

func TestResizeToFitExactFit(t *testing.T) {
	// Image that exactly fits the pixel budget should be returned as-is.
	img := makeImage(80, 160, color.White)
	resized := ResizeToFit(img, 10, 10, 8, 16)

	if resized != img {
		t.Error("image that fits exactly should be returned unmodified")
	}
}

// --- Halfblocks renderer tests (works without real terminal) ---------------

func TestRenderHalfblocksSolidColor(t *testing.T) {
	caps := makeCaps(terminal.ProtocolHalfblocks)
	r := NewRenderer(caps, makeCfg())

	// 4x4 red image.
	img := makeImage(4, 4, color.NRGBA{R: 255, G: 0, B: 0, A: 255})
	output, err := r.Render(img, 10, 10)
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}

	if output == "" {
		t.Error("expected non-empty output")
	}

	// Should contain ANSI escape sequences.
	if !strings.Contains(output, "\x1b[") {
		t.Error("output should contain ANSI escape sequences")
	}

	// Should contain the upper half block character.
	if !strings.Contains(output, "\u2580") {
		t.Error("output should contain upper half block character U+2580")
	}

	// Should reset at the end.
	if !strings.HasSuffix(output, "\x1b[0m") {
		t.Error("output should end with ANSI reset")
	}
}

func TestRenderHalfblocksTransparent(t *testing.T) {
	caps := makeCaps(terminal.ProtocolHalfblocks)
	r := NewRenderer(caps, makeCfg())

	// Fully transparent 4x4 image.
	img := makeImage(4, 4, color.NRGBA{R: 0, G: 0, B: 0, A: 0})
	output, err := r.Render(img, 10, 10)
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}

	// Should use spaces for transparent pixels.
	if !strings.Contains(output, " ") {
		t.Error("transparent pixels should render as spaces")
	}
}

func TestRenderHalfblocksOddHeight(t *testing.T) {
	caps := makeCaps(terminal.ProtocolHalfblocks)
	r := NewRenderer(caps, makeCfg())

	// 4x3 image (odd height - last row has no bottom pixel).
	img := makeImage(4, 3, color.NRGBA{R: 0, G: 255, B: 0, A: 255})
	output, err := r.Render(img, 10, 10)
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}

	if output == "" {
		t.Error("expected non-empty output for odd-height image")
	}
}

func TestRenderHalfblocksCachedOnSecondCall(t *testing.T) {
	caps := makeCaps(terminal.ProtocolHalfblocks)
	r := NewRenderer(caps, makeCfg())
	img := makeImage(4, 4, color.NRGBA{R: 128, G: 64, B: 32, A: 255})

	// First render.
	out1, err := r.Render(img, 5, 5)
	if err != nil {
		t.Fatalf("first render: %v", err)
	}

	// Second render should hit cache.
	out2, err := r.Render(img, 5, 5)
	if err != nil {
		t.Fatalf("second render: %v", err)
	}

	if out1 != out2 {
		t.Error("cached output should match original")
	}

	stats := r.Cache().Stats()
	if stats.Hits != 1 {
		t.Errorf("expected 1 cache hit, got %d", stats.Hits)
	}
}

func TestRenderNilImage(t *testing.T) {
	caps := makeCaps(terminal.ProtocolHalfblocks)
	r := NewRenderer(caps, makeCfg())

	_, err := r.Render(nil, 10, 10)
	if err == nil {
		t.Error("expected error for nil image")
	}
}

func TestRenderDifferentSizesProduceDifferentCacheKeys(t *testing.T) {
	caps := makeCaps(terminal.ProtocolHalfblocks)
	r := NewRenderer(caps, makeCfg())
	img := makeImage(100, 100, color.White)

	_, err1 := r.Render(img, 10, 10)
	if err1 != nil {
		t.Fatalf("render 10x10: %v", err1)
	}

	_, err2 := r.Render(img, 20, 20)
	if err2 != nil {
		t.Fatalf("render 20x20: %v", err2)
	}

	stats := r.Cache().Stats()
	// Two misses (different sizes), zero hits.
	if stats.Misses != 2 {
		t.Errorf("expected 2 misses for different sizes, got %d", stats.Misses)
	}
	if stats.Hits != 0 {
		t.Errorf("expected 0 hits for different sizes, got %d", stats.Hits)
	}
}

// --- Image hash stability tests --------------------------------------------

func TestImageHashStability(t *testing.T) {
	caps := makeCaps(terminal.ProtocolHalfblocks)
	r := NewRenderer(caps, makeCfg())

	img := makeGradientImage(16, 16)

	h1 := r.hashImage(img)
	h2 := r.hashImage(img)

	if h1 != h2 {
		t.Error("hash of same image should be stable across calls")
	}
}

func TestImageHashDifferentImages(t *testing.T) {
	caps := makeCaps(terminal.ProtocolHalfblocks)
	r := NewRenderer(caps, makeCfg())

	img1 := makeImage(16, 16, color.White)
	img2 := makeImage(16, 16, color.Black)

	h1 := r.hashImage(img1)
	h2 := r.hashImage(img2)

	if h1 == h2 {
		t.Error("different images should produce different hashes")
	}
}

func TestImageHashDifferentSizes(t *testing.T) {
	caps := makeCaps(terminal.ProtocolHalfblocks)
	r := NewRenderer(caps, makeCfg())

	img1 := makeImage(16, 16, color.White)
	img2 := makeImage(32, 32, color.White)

	h1 := r.hashImage(img1)
	h2 := r.hashImage(img2)

	if h1 == h2 {
		t.Error("images with different dimensions should produce different hashes")
	}
}

func TestImageHashLargeImageSampling(t *testing.T) {
	caps := makeCaps(terminal.ProtocolHalfblocks)
	r := NewRenderer(caps, makeCfg())

	// Large image that triggers sampling path (> 256x256 = 65536 pixels).
	img := makeGradientImage(300, 300)
	h1 := r.hashImage(img)
	h2 := r.hashImage(img)

	if h1 != h2 {
		t.Error("large image hash should be stable")
	}
}

// --- Async rendering tests -------------------------------------------------

func TestAsyncRenderCompletes(t *testing.T) {
	caps := makeCaps(terminal.ProtocolHalfblocks)
	r := NewRenderer(caps, makeCfg())
	ar := NewAsyncRenderer(r)
	defer ar.Close()

	img := makeImage(4, 4, color.NRGBA{R: 255, A: 255})
	done := make(chan struct{})
	var result string
	var renderErr error

	ar.RenderAsync(img, 10, 10, func(s string, err error) {
		result = s
		renderErr = err
		close(done)
	})

	select {
	case <-done:
		if renderErr != nil {
			t.Fatalf("async render error: %v", renderErr)
		}
		if result == "" {
			t.Error("async render produced empty output")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("async render timed out")
	}
}

func TestAsyncRenderCancel(t *testing.T) {
	caps := makeCaps(terminal.ProtocolHalfblocks)
	r := NewRenderer(caps, makeCfg())
	ar := NewAsyncRendererWithWorkers(r, 1)
	defer ar.Close()

	// Fill the worker with a job.
	blocker := make(chan struct{})
	img := makeImage(4, 4, color.White)

	ar.RenderAsync(img, 10, 10, func(string, error) {
		<-blocker // Block the worker.
	})

	// Submit a second job and immediately cancel it.
	var called atomic.Bool
	cancel := ar.RenderAsync(img, 20, 20, func(string, error) {
		called.Store(true)
	})
	cancel()

	// Unblock the first job.
	close(blocker)

	// Give a moment for the second job to process.
	time.Sleep(100 * time.Millisecond)

	// The callback may or may not be called depending on timing, but if
	// cancel was effective, called should be false. This is a best-effort test.
	// We mainly verify no panics or deadlocks occur.
}

func TestAsyncRenderMultipleJobs(t *testing.T) {
	caps := makeCaps(terminal.ProtocolHalfblocks)
	r := NewRenderer(caps, makeCfg())
	ar := NewAsyncRendererWithWorkers(r, 2)
	defer ar.Close()

	const n = 10
	var completed atomic.Int32
	done := make(chan struct{})

	for i := 0; i < n; i++ {
		img := makeImage(4+i, 4+i, color.NRGBA{R: uint8(i * 25), A: 255})
		ar.RenderAsync(img, 10, 10, func(s string, err error) {
			if err != nil {
				t.Errorf("job failed: %v", err)
			}
			if completed.Add(1) == n {
				close(done)
			}
		})
	}

	select {
	case <-done:
		// All completed.
	case <-time.After(10 * time.Second):
		t.Fatalf("timed out: only %d/%d completed", completed.Load(), n)
	}
}

func TestAsyncRendererClose(t *testing.T) {
	caps := makeCaps(terminal.ProtocolHalfblocks)
	r := NewRenderer(caps, makeCfg())
	ar := NewAsyncRenderer(r)

	// Close should not panic and should be idempotent.
	ar.Close()
	ar.Close()
}

// --- ImageToNRGBA tests ----------------------------------------------------

func TestImageToNRGBA(t *testing.T) {
	// RGBA source.
	rgba := image.NewRGBA(image.Rect(0, 0, 10, 10))
	draw.Draw(rgba, rgba.Bounds(), &image.Uniform{color.White}, image.Point{}, draw.Src)
	result := ImageToNRGBA(rgba)

	if result.Bounds() != rgba.Bounds() {
		t.Error("bounds should match")
	}

	// Already NRGBA.
	nrgba := image.NewNRGBA(image.Rect(0, 0, 5, 5))
	same := ImageToNRGBA(nrgba)
	if same != nrgba {
		t.Error("NRGBA input should return same pointer")
	}
}

// --- Integration: full pipeline halfblocks ---------------------------------

func TestFullPipelineHalfblocks(t *testing.T) {
	caps := makeCaps(terminal.ProtocolHalfblocks)
	r := NewRenderer(caps, makeCfg())

	// A moderately sized gradient image.
	img := makeGradientImage(64, 48)
	output, err := r.Render(img, 20, 15)
	if err != nil {
		t.Fatalf("full pipeline render: %v", err)
	}

	if output == "" {
		t.Error("full pipeline produced empty output")
	}

	// Should produce multiple lines.
	lines := strings.Split(output, "\n")
	if len(lines) < 2 {
		t.Errorf("expected multiple lines, got %d", len(lines))
	}

	// Should contain color codes.
	if !strings.Contains(output, "\x1b[38;2;") {
		t.Error("output should contain 24-bit foreground color codes")
	}
}

// --- Cache default size test -----------------------------------------------

func TestCacheDefaultSize(t *testing.T) {
	c := NewCache(0) // Should default to 32 MB.
	if c.maxBytes != 32*1024*1024 {
		t.Errorf("expected default 32 MB, got %d bytes", c.maxBytes)
	}
}

func TestCacheNegativeSize(t *testing.T) {
	c := NewCache(-5)
	if c.maxBytes != 32*1024*1024 {
		t.Errorf("expected default 32 MB for negative size, got %d bytes", c.maxBytes)
	}
}

// --- NewRenderer default cache size ----------------------------------------

func TestNewRendererDefaultCacheSize(t *testing.T) {
	cfg := config.ImageConfig{} // MaxCacheSizeMB = 0
	r := NewRenderer(makeCaps(terminal.ProtocolHalfblocks), cfg)

	if r.cache.maxBytes != 32*1024*1024 {
		t.Errorf("renderer should use default cache size, got %d", r.cache.maxBytes)
	}
}

// --- MakeCacheKey / CacheKey.String ----------------------------------------

func TestCacheKeyString(t *testing.T) {
	key := MakeCacheKey("kitty", 40, 20, [32]byte{0xab, 0xcd, 0xef, 0x01, 0x23, 0x45, 0x67, 0x89})
	s := key.String()
	if !strings.Contains(s, "kitty") {
		t.Error("key string should contain protocol")
	}
	if !strings.Contains(s, "40x20") {
		t.Error("key string should contain dimensions")
	}
}

// --- Kitty Unicode placeholder tests ---------------------------------------

func TestImgKittyUnicodePlaceholderLength(t *testing.T) {
	rows, cols := 3, 4
	placeholder := imgKittyUnicodePlaceholder(1, rows, cols)

	// Split into lines and count runes per line.
	lines := strings.Split(placeholder, "\n")
	if len(lines) != rows {
		t.Fatalf("expected %d lines, got %d", rows, len(lines))
	}

	for i, line := range lines {
		// Each cell contributes 3 runes: base + row diacritic + col diacritic.
		runeCount := utf8.RuneCountInString(line)
		expected := cols * 3
		if runeCount != expected {
			t.Errorf("line %d: expected %d runes, got %d", i, expected, runeCount)
		}
	}
}

func TestImgKittyUnicodePlaceholderUsesBaseChar(t *testing.T) {
	placeholder := imgKittyUnicodePlaceholder(1, 2, 2)

	// U+10EEEE should appear in the placeholder.
	const baseChar = '\U0010EEEE'
	if !strings.ContainsRune(placeholder, baseChar) {
		t.Error("placeholder should contain U+10EEEE base character")
	}

	// Count occurrences: should be rows * cols = 4.
	count := strings.Count(placeholder, string(baseChar))
	if count != 4 {
		t.Errorf("expected 4 base characters, got %d", count)
	}
}

func TestImgKittyUnicodePlaceholderEmpty(t *testing.T) {
	if imgKittyUnicodePlaceholder(1, 0, 5) != "" {
		t.Error("zero rows should return empty string")
	}
	if imgKittyUnicodePlaceholder(1, 5, 0) != "" {
		t.Error("zero cols should return empty string")
	}
}

// --- Kitty transmit tests --------------------------------------------------

func TestImgKittyTransmitChunking(t *testing.T) {
	// Create data larger than one chunk (4096 bytes base64 = ~3072 raw bytes).
	data := bytes.Repeat([]byte{0xAB}, 5000)
	result := imgKittyTransmit(data, 42, false)

	// Should contain multiple APC sequences.
	chunks := strings.Count(result, imgKittyESC)
	if chunks < 2 {
		t.Errorf("expected multiple chunks for 5000 bytes, got %d APC sequences", chunks)
	}

	// First chunk should contain the header fields.
	if !strings.Contains(result, "a=t") {
		t.Error("first chunk should contain action=t")
	}
	if !strings.Contains(result, "i=42") {
		t.Error("first chunk should contain image id")
	}
	if !strings.Contains(result, "f=32") {
		t.Error("first chunk should contain format=32")
	}

	// Last chunk should have m=0.
	lastST := strings.LastIndex(result, imgKittyST)
	lastESC := strings.LastIndex(result[:lastST], imgKittyESC)
	lastChunk := result[lastESC:lastST]
	if !strings.Contains(lastChunk, "m=0") {
		t.Error("last chunk should have m=0 (final)")
	}
}

func TestImgKittyTransmitCompressed(t *testing.T) {
	// Highly compressible data.
	data := bytes.Repeat([]byte{0x00}, 10000)

	uncompressed := imgKittyTransmit(data, 1, false)
	compressed := imgKittyTransmit(data, 1, true)

	if len(compressed) >= len(uncompressed) {
		t.Errorf("compressed output (%d bytes) should be smaller than uncompressed (%d bytes)",
			len(compressed), len(uncompressed))
	}

	// Compressed version should have the compression flag.
	if !strings.Contains(compressed, "o=z") {
		t.Error("compressed transmit should include o=z flag")
	}
}

func TestImgKittyTransmitEmpty(t *testing.T) {
	result := imgKittyTransmit(nil, 1, false)

	if !strings.Contains(result, imgKittyESC) {
		t.Error("empty transmit should still produce an APC sequence")
	}
	if !strings.Contains(result, "m=0") {
		t.Error("empty transmit should have m=0 (no continuation)")
	}
	if !strings.Contains(result, "i=1") {
		t.Error("empty transmit should contain image id")
	}
}

// --- Kitty display tests ---------------------------------------------------

func TestImgKittyDisplayIncludesIDAndDimensions(t *testing.T) {
	result := imgKittyDisplay(7, 3, 5, 2)

	if !strings.Contains(result, "i=7") {
		t.Error("display should contain image id")
	}
	if !strings.Contains(result, "r=3") {
		t.Error("display should contain row count")
	}
	if !strings.Contains(result, "c=5") {
		t.Error("display should contain column count")
	}
	if !strings.Contains(result, "z=2") {
		t.Error("display should contain z-index")
	}

	// Should also contain the placeholder.
	const baseChar = '\U0010EEEE'
	if !strings.ContainsRune(result, baseChar) {
		t.Error("display should contain Unicode placeholder")
	}
}

// --- ZLIB compression tests ------------------------------------------------

func TestImgZlibCompressRoundtrip(t *testing.T) {
	original := []byte("The quick brown fox jumps over the lazy dog. " +
		"Pack my box with five dozen liquor jugs.")

	compressed, err := imgZlibCompress(original)
	if err != nil {
		t.Fatalf("compress failed: %v", err)
	}

	// Decompress.
	r, err := zlib.NewReader(bytes.NewReader(compressed))
	if err != nil {
		t.Fatalf("new zlib reader: %v", err)
	}
	defer r.Close()

	decompressed, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("decompress failed: %v", err)
	}

	if !bytes.Equal(original, decompressed) {
		t.Errorf("roundtrip failed: got %q, want %q", decompressed, original)
	}
}

func TestImgZlibCompressEmptyInput(t *testing.T) {
	compressed, err := imgZlibCompress([]byte{})
	if err != nil {
		t.Fatalf("compress empty: %v", err)
	}

	// Should produce valid zlib output (just the header + empty stream).
	r, err := zlib.NewReader(bytes.NewReader(compressed))
	if err != nil {
		t.Fatalf("new zlib reader for empty: %v", err)
	}
	defer r.Close()

	decompressed, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("decompress empty: %v", err)
	}
	if len(decompressed) != 0 {
		t.Errorf("expected empty decompressed output, got %d bytes", len(decompressed))
	}
}

// --- Pixel perfect size tests ----------------------------------------------

func TestImgPixelPerfectSizeMaintainsAspectRatio(t *testing.T) {
	// 160x80 image (2:1 aspect ratio), cell 8x16, max 40x20 cells.
	cols, rows := imgPixelPerfectSize(160, 80, 8, 16, 40, 20)

	// At native resolution: 160/8=20 cols, 80/16=5 rows. Fits within max.
	if cols != 20 {
		t.Errorf("expected 20 cols, got %d", cols)
	}
	if rows != 5 {
		t.Errorf("expected 5 rows, got %d", rows)
	}
}

func TestImgPixelPerfectSizeRespectsMaxDimensions(t *testing.T) {
	// 800x600 image, cell 8x16, max 10x5 cells (80x80 pixels).
	cols, rows := imgPixelPerfectSize(800, 600, 8, 16, 10, 5)

	if cols > 10 {
		t.Errorf("cols %d exceeds max 10", cols)
	}
	if rows > 5 {
		t.Errorf("rows %d exceeds max 5", rows)
	}
	if cols <= 0 || rows <= 0 {
		t.Errorf("cols (%d) and rows (%d) must be positive", cols, rows)
	}
}

func TestImgPixelPerfectSizeSquareImage(t *testing.T) {
	// 100x100 square image, cell 10x10 (square cells), max 20x20 cells.
	cols, rows := imgPixelPerfectSize(100, 100, 10, 10, 20, 20)

	if cols != 10 {
		t.Errorf("expected 10 cols for square image, got %d", cols)
	}
	if rows != 10 {
		t.Errorf("expected 10 rows for square image, got %d", rows)
	}
}

// --- Sharpen filter tests --------------------------------------------------

func TestImgSharpenFilterSameDimensions(t *testing.T) {
	img := makeImage(50, 50, color.NRGBA{R: 128, G: 64, B: 32, A: 255})
	sharpened := imgSharpenFilter(img, 0.3)

	if sharpened == nil {
		t.Fatal("sharpened image should not be nil")
	}

	bounds := sharpened.Bounds()
	if bounds.Dx() != 50 || bounds.Dy() != 50 {
		t.Errorf("expected 50x50, got %dx%d", bounds.Dx(), bounds.Dy())
	}
}

func TestImgSharpenFilterZeroAmount(t *testing.T) {
	img := makeImage(20, 20, color.NRGBA{R: 100, G: 100, B: 100, A: 255})
	result := imgSharpenFilter(img, 0)

	// Should return the image unchanged (same pointer via ImageToNRGBA shortcut).
	if result == nil {
		t.Fatal("result should not be nil")
	}

	// With zero amount, no sharpening is applied so result should be
	// the converted NRGBA (or original if already NRGBA).
	bounds := result.Bounds()
	if bounds.Dx() != 20 || bounds.Dy() != 20 {
		t.Errorf("expected 20x20, got %dx%d", bounds.Dx(), bounds.Dy())
	}
}

func TestImgSharpenFilterNil(t *testing.T) {
	result := imgSharpenFilter(nil, 0.5)
	if result != nil {
		t.Error("nil input should return nil")
	}
}

// --- Lanczos resize tests --------------------------------------------------

func TestImgLanczosResizeProducesCorrectDimensions(t *testing.T) {
	img := makeImage(200, 100, color.NRGBA{R: 255, G: 0, B: 0, A: 255})
	resized := imgLanczosResize(img, 80, 40)

	if resized == nil {
		t.Fatal("resized image should not be nil")
	}

	bounds := resized.Bounds()
	if bounds.Dx() != 80 || bounds.Dy() != 40 {
		t.Errorf("expected 80x40, got %dx%d", bounds.Dx(), bounds.Dy())
	}
}

func TestImgLanczosResizeDownscale(t *testing.T) {
	img := makeGradientImage(256, 256)
	resized := imgLanczosResize(img, 64, 64)

	if resized == nil {
		t.Fatal("resized image should not be nil")
	}

	bounds := resized.Bounds()
	if bounds.Dx() != 64 || bounds.Dy() != 64 {
		t.Errorf("expected 64x64, got %dx%d", bounds.Dx(), bounds.Dy())
	}

	// Verify it's actually a different image (not just the original).
	if bounds.Dx() == img.Bounds().Dx() {
		t.Error("downscaled image should have different dimensions than original")
	}
}

func TestImgLanczosResizeNil(t *testing.T) {
	result := imgLanczosResize(nil, 100, 100)
	if result != nil {
		t.Error("nil input should return nil")
	}
}

func TestImgLanczosResizeZeroDimensions(t *testing.T) {
	img := makeImage(100, 100, color.White)
	result := imgLanczosResize(img, 0, 50)
	// Should return original unchanged.
	if result.Bounds().Dx() != 100 {
		t.Error("zero width should return original image")
	}
}

// --- Terminal pipeline tests -----------------------------------------------

func TestImgTerminalPipelineCorrectPixelDimensions(t *testing.T) {
	img := makeImage(400, 300, color.NRGBA{R: 0, G: 128, B: 255, A: 255})

	// Target: 10 cols x 5 rows, cell 8x16 = 80x80 pixel output.
	result := imgTerminalPipeline(img, 10, 5, 8, 16)

	if result == nil {
		t.Fatal("pipeline result should not be nil")
	}

	bounds := result.Bounds()
	expectedW := 10 * 8 // 80
	expectedH := 5 * 16 // 80

	if bounds.Dx() != expectedW {
		t.Errorf("expected width %d, got %d", expectedW, bounds.Dx())
	}
	if bounds.Dy() != expectedH {
		t.Errorf("expected height %d, got %d", expectedH, bounds.Dy())
	}
}

func TestImgTerminalPipelineNil(t *testing.T) {
	result := imgTerminalPipeline(nil, 10, 5, 8, 16)
	if result != nil {
		t.Error("nil input should return nil")
	}
}

// --- Multi-placement tests -------------------------------------------------

func TestImgRenderPlacementsSingle(t *testing.T) {
	placements := []imgPlacement{
		{
			ID:     1,
			Data:   []byte{0xFF, 0x00, 0x00, 0xFF}, // 1 RGBA pixel
			Row:    0,
			Col:    0,
			Rows:   2,
			Cols:   3,
			ZIndex: 0,
		},
	}

	result := imgRenderPlacements(placements, terminal.ProtocolKitty)

	if result == "" {
		t.Fatal("expected non-empty output for single placement")
	}

	// Should contain Kitty transmit sequence.
	if !strings.Contains(result, imgKittyESC) {
		t.Error("Kitty placement should contain APC escape")
	}

	// Should contain the image id.
	if !strings.Contains(result, "i=1") {
		t.Error("should contain image id 1")
	}
}

func TestImgRenderPlacementsMultiple(t *testing.T) {
	placements := []imgPlacement{
		{
			ID:     10,
			Data:   []byte{0xFF, 0x00, 0x00, 0xFF},
			Row:    0,
			Col:    0,
			Rows:   2,
			Cols:   3,
			ZIndex: 0,
		},
		{
			ID:     20,
			Data:   []byte{0x00, 0xFF, 0x00, 0xFF},
			Row:    5,
			Col:    10,
			Rows:   4,
			Cols:   6,
			ZIndex: 1,
		},
	}

	result := imgRenderPlacements(placements, terminal.ProtocolKitty)

	if result == "" {
		t.Fatal("expected non-empty output for multiple placements")
	}

	// Should contain both image ids.
	if !strings.Contains(result, "i=10") {
		t.Error("should contain image id 10")
	}
	if !strings.Contains(result, "i=20") {
		t.Error("should contain image id 20")
	}

	// Should contain cursor positioning for second placement.
	if !strings.Contains(result, "\x1b[6;11H") {
		t.Error("should contain cursor positioning for second placement (row 5+1, col 10+1)")
	}
}

func TestImgRenderPlacementsEmpty(t *testing.T) {
	result := imgRenderPlacements(nil, terminal.ProtocolKitty)
	if result != "" {
		t.Error("empty placements should return empty string")
	}
}

func TestImgRenderPlacementsAutoID(t *testing.T) {
	placements := []imgPlacement{
		{
			ID:     0, // Auto-assign
			Data:   []byte{0xFF, 0x00, 0x00, 0xFF},
			Row:    0,
			Col:    0,
			Rows:   1,
			Cols:   1,
			ZIndex: 0,
		},
	}

	result := imgRenderPlacements(placements, terminal.ProtocolKitty)

	// Auto-assigned ID should be 1 (index + 1).
	if !strings.Contains(result, "i=1") {
		t.Error("auto-assigned ID should be 1")
	}
	if placements[0].ID != 1 {
		t.Errorf("placement ID should be updated to 1, got %d", placements[0].ID)
	}
}

func TestImgRenderPlacementsSequentialProtocol(t *testing.T) {
	placements := []imgPlacement{
		{
			ID:     1,
			Data:   []byte{0xFF, 0x00, 0x00, 0xFF},
			Row:    2,
			Col:    3,
			Rows:   1,
			Cols:   1,
			ZIndex: 0,
		},
	}

	result := imgRenderPlacements(placements, terminal.ProtocolHalfblocks)

	if result == "" {
		t.Fatal("sequential placement should produce output")
	}

	// Should contain cursor positioning.
	if !strings.Contains(result, "\x1b[3;4H") {
		t.Error("should contain cursor positioning for halfblocks placement")
	}
}

// --- Kitty transmit base64 validity ----------------------------------------

func TestImgKittyTransmitBase64Valid(t *testing.T) {
	data := []byte{0xDE, 0xAD, 0xBE, 0xEF, 0xCA, 0xFE}
	result := imgKittyTransmit(data, 1, false)

	// Extract the base64 payload from between ';' and imgKittyST.
	start := strings.Index(result, ";") + 1
	end := strings.Index(result, imgKittyST)
	if start <= 0 || end <= start {
		t.Fatal("could not extract base64 payload")
	}
	payload := result[start:end]

	_, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		t.Errorf("base64 payload is invalid: %v", err)
	}
}

// --- Detect cell size fallback test ----------------------------------------

func TestImgDetectCellSizeFallback(t *testing.T) {
	// In a test environment (no TTY), should fall back to defaults.
	w, h, err := imgDetectCellSize()
	if err != nil {
		t.Fatalf("detect cell size should not error: %v", err)
	}
	if w <= 0 || h <= 0 {
		t.Errorf("cell size should be positive, got %dx%d", w, h)
	}
}

// --- Lanczos3 kernel function test -----------------------------------------

func TestImgLanczos3AtKernelValues(t *testing.T) {
	// At t=0, kernel should be 1.
	if v := imgLanczos3At(0); v != 1.0 {
		t.Errorf("Lanczos3(0) = %f, want 1.0", v)
	}

	// At t>=3, kernel should be 0.
	if v := imgLanczos3At(3.0); v != 0.0 {
		t.Errorf("Lanczos3(3.0) = %f, want 0.0", v)
	}
	if v := imgLanczos3At(5.0); v != 0.0 {
		t.Errorf("Lanczos3(5.0) = %f, want 0.0", v)
	}

	// Negative values should behave the same as positive (symmetric).
	v1 := imgLanczos3At(1.5)
	v2 := imgLanczos3At(-1.5)
	if v1 != v2 {
		t.Errorf("Lanczos3 should be symmetric: f(1.5)=%f, f(-1.5)=%f", v1, v2)
	}
}
