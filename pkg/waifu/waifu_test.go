package waifu

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// --- Helpers ---

// createTestImage writes a fake image file with the given extension and content.
func createTestImage(t *testing.T, dir, name string, content []byte) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("write test image: %v", err)
	}
	return path
}

// mockRenderer is a test double for ImageRenderer.
type mockRenderer struct {
	mu       sync.Mutex
	calls    int
	result   string
	err      error
	delay    time.Duration
	renderFn func(path string, width, height int) (string, error)
}

func (m *mockRenderer) RenderFile(path string, width, height int) (string, error) {
	m.mu.Lock()
	m.calls++
	fn := m.renderFn
	result := m.result
	err := m.err
	delay := m.delay
	m.mu.Unlock()

	if delay > 0 {
		time.Sleep(delay)
	}

	if fn != nil {
		return fn(path, width, height)
	}
	return result, err
}

func (m *mockRenderer) callCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.calls
}

// --- Session Tests ---

func TestSessionIDFormat(t *testing.T) {
	dir := t.TempDir()
	createTestImage(t, dir, "test.png", []byte("fake-png-data"))

	sm := NewSessionManager(SessionConfig{
		ImageDir: dir,
		CacheDir: t.TempDir(),
	})

	s, err := sm.GetOrCreate()
	if err != nil {
		t.Fatalf("GetOrCreate: %v", err)
	}

	expected := fmt.Sprintf("ppulse-%d", os.Getpid())
	if s.ID != expected {
		t.Errorf("session ID = %q, want %q", s.ID, expected)
	}
}

func TestSessionIDNeverContainsTimestamp(t *testing.T) {
	dir := t.TempDir()
	createTestImage(t, dir, "test.png", []byte("fake-png-data"))

	sm := NewSessionManager(SessionConfig{
		ImageDir: dir,
		CacheDir: t.TempDir(),
	})

	s, err := sm.GetOrCreate()
	if err != nil {
		t.Fatalf("GetOrCreate: %v", err)
	}

	// Session ID must be "ppulse-" + digits only.
	if !strings.HasPrefix(s.ID, "ppulse-") {
		t.Errorf("session ID %q does not start with ppulse-", s.ID)
	}

	// No colons, dashes beyond the first, or dots that would indicate a timestamp.
	pidPart := strings.TrimPrefix(s.ID, "ppulse-")
	for _, c := range pidPart {
		if c < '0' || c > '9' {
			t.Errorf("session ID PID part %q contains non-digit %q", pidPart, string(c))
		}
	}
}

func TestSessionStableSamePID(t *testing.T) {
	dir := t.TempDir()
	createTestImage(t, dir, "test.png", []byte("fake-png-data"))

	sm := NewSessionManager(SessionConfig{
		ImageDir: dir,
		CacheDir: t.TempDir(),
	})

	s1, err := sm.GetOrCreate()
	if err != nil {
		t.Fatalf("first GetOrCreate: %v", err)
	}

	s2, err := sm.GetOrCreate()
	if err != nil {
		t.Fatalf("second GetOrCreate: %v", err)
	}

	if s1 != s2 {
		t.Errorf("GetOrCreate returned different session pointers for same PID")
	}
	if s1.ID != s2.ID {
		t.Errorf("session IDs differ: %q vs %q", s1.ID, s2.ID)
	}
	if s1.ImagePath != s2.ImagePath {
		t.Errorf("image paths differ: %q vs %q", s1.ImagePath, s2.ImagePath)
	}
}

func TestSessionSelectsImage(t *testing.T) {
	dir := t.TempDir()
	createTestImage(t, dir, "waifu1.png", []byte("image-data-1"))
	createTestImage(t, dir, "waifu2.jpg", []byte("image-data-2"))

	sm := NewSessionManager(SessionConfig{
		ImageDir: dir,
		CacheDir: t.TempDir(),
	})

	s, err := sm.GetOrCreate()
	if err != nil {
		t.Fatalf("GetOrCreate: %v", err)
	}

	if s.ImagePath == "" {
		t.Fatal("session ImagePath is empty")
	}

	// The selected image must exist.
	if _, err := os.Stat(s.ImagePath); err != nil {
		t.Errorf("selected image does not exist: %v", err)
	}
}

func TestSessionContentHashComputed(t *testing.T) {
	dir := t.TempDir()
	content := []byte("deterministic-image-content-for-hashing")
	createTestImage(t, dir, "only.png", content)

	sm := NewSessionManager(SessionConfig{
		ImageDir: dir,
		CacheDir: t.TempDir(),
	})

	s, err := sm.GetOrCreate()
	if err != nil {
		t.Fatalf("GetOrCreate: %v", err)
	}

	// Verify the hash matches what we expect.
	h := sha256.Sum256(content)
	expected := fmt.Sprintf("%x", h[:])[:16]

	if s.ContentHash != expected {
		t.Errorf("ContentHash = %q, want %q", s.ContentHash, expected)
	}
}

func TestSessionContentHashLength(t *testing.T) {
	dir := t.TempDir()
	createTestImage(t, dir, "test.png", []byte("some-image-bytes"))

	sm := NewSessionManager(SessionConfig{
		ImageDir: dir,
		CacheDir: t.TempDir(),
	})

	s, err := sm.GetOrCreate()
	if err != nil {
		t.Fatalf("GetOrCreate: %v", err)
	}

	if len(s.ContentHash) != 16 {
		t.Errorf("ContentHash length = %d, want 16", len(s.ContentHash))
	}
}

func TestSessionGet(t *testing.T) {
	dir := t.TempDir()
	createTestImage(t, dir, "test.png", []byte("data"))

	sm := NewSessionManager(SessionConfig{
		ImageDir: dir,
		CacheDir: t.TempDir(),
	})

	s, err := sm.GetOrCreate()
	if err != nil {
		t.Fatalf("GetOrCreate: %v", err)
	}

	got, ok := sm.Get(s.ID)
	if !ok {
		t.Fatal("Get returned false for existing session")
	}
	if got.ID != s.ID {
		t.Errorf("Get returned different session: %q vs %q", got.ID, s.ID)
	}

	_, ok = sm.Get("nonexistent-id")
	if ok {
		t.Error("Get returned true for nonexistent session")
	}
}

func TestSessionClose(t *testing.T) {
	dir := t.TempDir()
	createTestImage(t, dir, "test.png", []byte("data"))

	sm := NewSessionManager(SessionConfig{
		ImageDir: dir,
		CacheDir: t.TempDir(),
	})

	s, err := sm.GetOrCreate()
	if err != nil {
		t.Fatalf("GetOrCreate: %v", err)
	}

	sm.Close(s.ID)

	_, ok := sm.Get(s.ID)
	if ok {
		t.Error("session still exists after Close")
	}
}

func TestCleanStale(t *testing.T) {
	sm := NewSessionManager(SessionConfig{
		ImageDir: t.TempDir(),
		CacheDir: t.TempDir(),
	})

	// Manually insert sessions with old timestamps.
	sm.mu.Lock()
	sm.sessions["old-1"] = &Session{
		ID:        "old-1",
		CreatedAt: time.Now().Add(-2 * time.Hour),
	}
	sm.sessions["old-2"] = &Session{
		ID:        "old-2",
		CreatedAt: time.Now().Add(-3 * time.Hour),
	}
	sm.sessions["recent"] = &Session{
		ID:        "recent",
		CreatedAt: time.Now().Add(-5 * time.Minute),
	}
	sm.mu.Unlock()

	sm.CleanStale(1 * time.Hour)

	if sm.ActiveCount() != 1 {
		t.Errorf("ActiveCount = %d after CleanStale, want 1", sm.ActiveCount())
	}

	_, ok := sm.Get("recent")
	if !ok {
		t.Error("recent session was incorrectly cleaned")
	}
}

func TestActiveCount(t *testing.T) {
	sm := NewSessionManager(SessionConfig{
		ImageDir: t.TempDir(),
		CacheDir: t.TempDir(),
	})

	if sm.ActiveCount() != 0 {
		t.Errorf("initial ActiveCount = %d, want 0", sm.ActiveCount())
	}

	sm.mu.Lock()
	sm.sessions["a"] = &Session{ID: "a", CreatedAt: time.Now()}
	sm.sessions["b"] = &Session{ID: "b", CreatedAt: time.Now()}
	sm.mu.Unlock()

	if sm.ActiveCount() != 2 {
		t.Errorf("ActiveCount = %d, want 2", sm.ActiveCount())
	}

	sm.Close("a")
	if sm.ActiveCount() != 1 {
		t.Errorf("ActiveCount after Close = %d, want 1", sm.ActiveCount())
	}
}

func TestSessionErrorOnEmptyDir(t *testing.T) {
	sm := NewSessionManager(SessionConfig{
		ImageDir: t.TempDir(), // empty
		CacheDir: t.TempDir(),
	})

	_, err := sm.GetOrCreate()
	if err == nil {
		t.Fatal("expected error for empty image directory")
	}
}

// --- Cache Tests ---

func TestCachePutGet(t *testing.T) {
	cache := NewImageCache(t.TempDir(), 10*1024*1024)

	key := CacheKey{
		ContentHash: "abcdef0123456789",
		Protocol:    "kitty",
		Width:       80,
		Height:      24,
	}
	rendered := "\x1b[38;2;255;0;0m\u2580"

	if err := cache.Put(key, rendered); err != nil {
		t.Fatalf("Put: %v", err)
	}

	got, ok := cache.Get(key)
	if !ok {
		t.Fatal("Get returned false for existing key")
	}
	if got != rendered {
		t.Errorf("Get = %q, want %q", got, rendered)
	}
}

func TestCacheHas(t *testing.T) {
	cache := NewImageCache(t.TempDir(), 10*1024*1024)

	key := CacheKey{
		ContentHash: "abcdef0123456789",
		Protocol:    "halfblocks",
		Width:       40,
		Height:      12,
	}

	if cache.Has(key) {
		t.Error("Has returned true for missing key")
	}

	if err := cache.Put(key, "rendered-data"); err != nil {
		t.Fatalf("Put: %v", err)
	}

	if !cache.Has(key) {
		t.Error("Has returned false for existing key")
	}
}

func TestCacheEvict(t *testing.T) {
	cache := NewImageCache(t.TempDir(), 10*1024*1024)

	key := CacheKey{
		ContentHash: "abcdef0123456789",
		Protocol:    "kitty",
		Width:       80,
		Height:      24,
	}

	if err := cache.Put(key, "data-to-evict"); err != nil {
		t.Fatalf("Put: %v", err)
	}

	cache.Evict(key)

	if cache.Has(key) {
		t.Error("key still exists after Evict")
	}

	_, ok := cache.Get(key)
	if ok {
		t.Error("Get returned true after Evict")
	}
}

func TestCacheAtomicWrite(t *testing.T) {
	cacheDir := t.TempDir()
	cache := NewImageCache(cacheDir, 10*1024*1024)

	key := CacheKey{
		ContentHash: "aabbccdd11223344",
		Protocol:    "sixel",
		Width:       60,
		Height:      20,
	}

	rendered := "atomic-write-test-content"
	if err := cache.Put(key, rendered); err != nil {
		t.Fatalf("Put: %v", err)
	}

	// The final file should exist with the correct content.
	p := cache.path(key)
	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("read cached file: %v", err)
	}
	if string(data) != rendered {
		t.Errorf("cached file content = %q, want %q", string(data), rendered)
	}

	// No temp files should remain.
	dir := filepath.Dir(p)
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".tmp") {
			t.Errorf("leftover temp file: %s", e.Name())
		}
	}
}

func TestCacheKeyStability(t *testing.T) {
	cache := NewImageCache(t.TempDir(), 10*1024*1024)

	key1 := CacheKey{ContentHash: "abcdef0123456789", Protocol: "kitty", Width: 80, Height: 24}
	key2 := CacheKey{ContentHash: "abcdef0123456789", Protocol: "kitty", Width: 80, Height: 24}

	if err := cache.Put(key1, "content-v1"); err != nil {
		t.Fatalf("Put key1: %v", err)
	}

	got, ok := cache.Get(key2)
	if !ok {
		t.Fatal("identical cache key not found")
	}
	if got != "content-v1" {
		t.Errorf("Get = %q, want %q", got, "content-v1")
	}
}

func TestCacheDifferentKeys(t *testing.T) {
	cache := NewImageCache(t.TempDir(), 10*1024*1024)

	key1 := CacheKey{ContentHash: "aaaa000011112222", Protocol: "kitty", Width: 80, Height: 24}
	key2 := CacheKey{ContentHash: "aaaa000011112222", Protocol: "kitty", Width: 40, Height: 12}
	key3 := CacheKey{ContentHash: "bbbb333344445555", Protocol: "kitty", Width: 80, Height: 24}

	cache.Put(key1, "content-1")
	cache.Put(key2, "content-2")
	cache.Put(key3, "content-3")

	got1, _ := cache.Get(key1)
	got2, _ := cache.Get(key2)
	got3, _ := cache.Get(key3)

	if got1 != "content-1" {
		t.Errorf("key1: got %q, want %q", got1, "content-1")
	}
	if got2 != "content-2" {
		t.Errorf("key2: got %q, want %q", got2, "content-2")
	}
	if got3 != "content-3" {
		t.Errorf("key3: got %q, want %q", got3, "content-3")
	}
}

func TestCachePrune(t *testing.T) {
	cacheDir := t.TempDir()
	// Very small max size: 50 bytes.
	cache := NewImageCache(cacheDir, 50)

	// Insert several entries totaling well over 50 bytes.
	for i := 0; i < 5; i++ {
		key := CacheKey{
			ContentHash: fmt.Sprintf("hash%d___________", i), // 16 chars
			Protocol:    "kitty",
			Width:       80,
			Height:      24,
		}
		content := strings.Repeat(fmt.Sprintf("x%d", i), 20) // ~40 bytes each
		if err := cache.Put(key, content); err != nil {
			t.Fatalf("Put %d: %v", i, err)
		}
		// Stagger mtimes so oldest is deterministic.
		time.Sleep(10 * time.Millisecond)
	}

	if err := cache.Prune(); err != nil {
		t.Fatalf("Prune: %v", err)
	}

	stats := cache.Stats()
	if stats.SizeBytes > 50 {
		t.Errorf("after Prune, SizeBytes = %d, want <= 50", stats.SizeBytes)
	}
}

func TestCacheStats(t *testing.T) {
	cache := NewImageCache(t.TempDir(), 10*1024*1024)

	key := CacheKey{ContentHash: "abcdef0123456789", Protocol: "kitty", Width: 80, Height: 24}

	// Miss.
	cache.Get(key)
	stats := cache.Stats()
	if stats.Misses != 1 {
		t.Errorf("Misses = %d, want 1", stats.Misses)
	}

	// Put + Hit.
	cache.Put(key, "data")
	cache.Get(key)
	stats = cache.Stats()
	if stats.Hits != 1 {
		t.Errorf("Hits = %d, want 1", stats.Hits)
	}
	if stats.Entries != 1 {
		t.Errorf("Entries = %d, want 1", stats.Entries)
	}
	if stats.SizeBytes != 4 { // len("data")
		t.Errorf("SizeBytes = %d, want 4", stats.SizeBytes)
	}
}

func TestCacheMissReturnsEmpty(t *testing.T) {
	cache := NewImageCache(t.TempDir(), 10*1024*1024)

	key := CacheKey{ContentHash: "nonexistent_hash", Protocol: "kitty", Width: 80, Height: 24}
	got, ok := cache.Get(key)
	if ok {
		t.Error("Get returned true for missing key")
	}
	if got != "" {
		t.Errorf("Get returned %q for missing key, want empty", got)
	}
}

// --- Picker Tests ---

func TestPickRandomReturnsImage(t *testing.T) {
	dir := t.TempDir()
	createTestImage(t, dir, "a.png", []byte("img-a"))
	createTestImage(t, dir, "b.jpg", []byte("img-b"))
	createTestImage(t, dir, "c.jpeg", []byte("img-c"))

	path, err := PickRandom(dir)
	if err != nil {
		t.Fatalf("PickRandom: %v", err)
	}

	if !strings.HasPrefix(path, dir) {
		t.Errorf("returned path %q not under directory %q", path, dir)
	}

	ext := strings.ToLower(filepath.Ext(path))
	if !supportedExtensions[ext] {
		t.Errorf("returned file has unsupported extension: %s", ext)
	}
}

func TestPickRandomErrorOnEmptyDir(t *testing.T) {
	dir := t.TempDir()

	_, err := PickRandom(dir)
	if err == nil {
		t.Fatal("expected error for empty directory")
	}
	if !strings.Contains(err.Error(), "no image files") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestPickRandomErrorOnMissingDir(t *testing.T) {
	_, err := PickRandom("/nonexistent/directory/path")
	if err == nil {
		t.Fatal("expected error for nonexistent directory")
	}
}

func TestPickRandomFiltersNonImages(t *testing.T) {
	dir := t.TempDir()
	// Only non-image files.
	createTestImage(t, dir, "readme.txt", []byte("text"))
	createTestImage(t, dir, "data.json", []byte("{}"))
	createTestImage(t, dir, "script.sh", []byte("#!/bin/bash"))

	_, err := PickRandom(dir)
	if err == nil {
		t.Fatal("expected error when no image files present")
	}
}

func TestPickRandomFiltersHiddenFiles(t *testing.T) {
	dir := t.TempDir()
	createTestImage(t, dir, ".hidden.png", []byte("hidden-image"))

	_, err := PickRandom(dir)
	if err == nil {
		t.Fatal("expected error: hidden images should be filtered out")
	}
}

func TestListImagesReturnsOnlyImages(t *testing.T) {
	dir := t.TempDir()
	createTestImage(t, dir, "a.png", []byte("img"))
	createTestImage(t, dir, "b.jpg", []byte("img"))
	createTestImage(t, dir, "c.gif", []byte("img"))
	createTestImage(t, dir, "d.webp", []byte("img"))
	createTestImage(t, dir, "e.bmp", []byte("img"))
	createTestImage(t, dir, "f.txt", []byte("text"))
	createTestImage(t, dir, "g.json", []byte("{}"))
	createTestImage(t, dir, ".hidden.png", []byte("img"))

	// Create a subdirectory (should be excluded).
	os.Mkdir(filepath.Join(dir, "subdir"), 0o755)

	images, err := ListImages(dir)
	if err != nil {
		t.Fatalf("ListImages: %v", err)
	}

	if len(images) != 5 {
		t.Errorf("ListImages returned %d images, want 5: %v", len(images), images)
	}

	for _, img := range images {
		ext := strings.ToLower(filepath.Ext(img))
		if !supportedExtensions[ext] {
			t.Errorf("ListImages returned non-image: %s", img)
		}
		if strings.HasPrefix(filepath.Base(img), ".") {
			t.Errorf("ListImages returned hidden file: %s", img)
		}
	}
}

func TestListImagesAllExtensions(t *testing.T) {
	dir := t.TempDir()
	exts := []string{".png", ".jpg", ".jpeg", ".gif", ".webp", ".bmp"}
	for i, ext := range exts {
		createTestImage(t, dir, fmt.Sprintf("img%d%s", i, ext), []byte("data"))
	}

	images, err := ListImages(dir)
	if err != nil {
		t.Fatalf("ListImages: %v", err)
	}

	if len(images) != len(exts) {
		t.Errorf("ListImages returned %d, want %d", len(images), len(exts))
	}
}

// --- Prefetcher Tests ---

func TestPrefetchCacheHit(t *testing.T) {
	cacheDir := t.TempDir()
	cache := NewImageCache(cacheDir, 10*1024*1024)

	session := &Session{
		ID:          "ppulse-99999",
		ImagePath:   "/fake/image.png",
		ContentHash: "abcdef0123456789",
	}

	key := CacheKey{
		ContentHash: session.ContentHash,
		Protocol:    "default",
		Width:       80,
		Height:      24,
	}
	cache.Put(key, "pre-cached-render")

	renderer := &mockRenderer{result: "should-not-be-called"}
	pf := NewPrefetcher(cache, renderer)
	defer pf.Close()

	ch := pf.Prefetch(session, 80, 24)
	result := <-ch

	if result.Error != nil {
		t.Fatalf("unexpected error: %v", result.Error)
	}
	if !result.FromCache {
		t.Error("expected FromCache = true")
	}
	if result.Rendered != "pre-cached-render" {
		t.Errorf("Rendered = %q, want %q", result.Rendered, "pre-cached-render")
	}
	if renderer.callCount() != 0 {
		t.Errorf("renderer was called %d times, expected 0", renderer.callCount())
	}
}

func TestPrefetchCacheMiss(t *testing.T) {
	cacheDir := t.TempDir()
	cache := NewImageCache(cacheDir, 10*1024*1024)

	session := &Session{
		ID:          "ppulse-99999",
		ImagePath:   "/fake/image.png",
		ContentHash: "abcdef0123456789",
	}

	renderer := &mockRenderer{result: "freshly-rendered"}
	pf := NewPrefetcher(cache, renderer)
	defer pf.Close()

	ch := pf.Prefetch(session, 80, 24)
	result := <-ch

	if result.Error != nil {
		t.Fatalf("unexpected error: %v", result.Error)
	}
	if result.FromCache {
		t.Error("expected FromCache = false")
	}
	if result.Rendered != "freshly-rendered" {
		t.Errorf("Rendered = %q, want %q", result.Rendered, "freshly-rendered")
	}
	if renderer.callCount() != 1 {
		t.Errorf("renderer was called %d times, expected 1", renderer.callCount())
	}

	// Verify the result was cached for future hits.
	key := CacheKey{
		ContentHash: session.ContentHash,
		Protocol:    "default",
		Width:       80,
		Height:      24,
	}
	if !cache.Has(key) {
		t.Error("result was not cached after prefetch")
	}
}

func TestPrefetchWaitBlocks(t *testing.T) {
	cacheDir := t.TempDir()
	cache := NewImageCache(cacheDir, 10*1024*1024)

	session := &Session{
		ID:          "ppulse-99999",
		ImagePath:   "/fake/image.png",
		ContentHash: "abcdef0123456789",
	}

	renderer := &mockRenderer{
		result: "slow-render",
		delay:  50 * time.Millisecond,
	}
	pf := NewPrefetcher(cache, renderer)

	ch := pf.Prefetch(session, 80, 24)

	// Wait should block until the goroutine completes.
	pf.Wait()

	// After Wait returns, the result should be available on the channel.
	select {
	case result := <-ch:
		if result.Error != nil {
			t.Fatalf("unexpected error: %v", result.Error)
		}
		if result.Rendered != "slow-render" {
			t.Errorf("Rendered = %q, want %q", result.Rendered, "slow-render")
		}
	default:
		t.Fatal("result not available after Wait()")
	}
}

func TestPrefetchCloseStopsWork(t *testing.T) {
	cacheDir := t.TempDir()
	cache := NewImageCache(cacheDir, 10*1024*1024)

	renderer := &mockRenderer{
		result: "should-complete",
		delay:  10 * time.Millisecond,
	}
	pf := NewPrefetcher(cache, renderer)

	// Launch several prefetches.
	for i := 0; i < 5; i++ {
		session := &Session{
			ID:          fmt.Sprintf("ppulse-%d", i),
			ImagePath:   "/fake/image.png",
			ContentHash: fmt.Sprintf("hash%d___________", i),
		}
		pf.Prefetch(session, 80, 24)
	}

	// Close should not hang; it cancels pending work and waits.
	done := make(chan struct{})
	go func() {
		pf.Close()
		close(done)
	}()

	select {
	case <-done:
		// Success: Close returned.
	case <-time.After(5 * time.Second):
		t.Fatal("Close() did not return within timeout")
	}
}

func TestPrefetchRenderError(t *testing.T) {
	cacheDir := t.TempDir()
	cache := NewImageCache(cacheDir, 10*1024*1024)

	session := &Session{
		ID:          "ppulse-99999",
		ImagePath:   "/fake/image.png",
		ContentHash: "abcdef0123456789",
	}

	renderer := &mockRenderer{err: fmt.Errorf("render failed: codec error")}
	pf := NewPrefetcher(cache, renderer)
	defer pf.Close()

	ch := pf.Prefetch(session, 80, 24)
	result := <-ch

	if result.Error == nil {
		t.Fatal("expected error from failed render")
	}
	if !strings.Contains(result.Error.Error(), "codec error") {
		t.Errorf("unexpected error: %v", result.Error)
	}
	if result.Rendered != "" {
		t.Errorf("Rendered should be empty on error, got %q", result.Rendered)
	}
}

func TestPrefetchDurationTracked(t *testing.T) {
	cacheDir := t.TempDir()
	cache := NewImageCache(cacheDir, 10*1024*1024)

	session := &Session{
		ID:          "ppulse-99999",
		ImagePath:   "/fake/image.png",
		ContentHash: "abcdef0123456789",
	}

	renderer := &mockRenderer{
		result: "rendered",
		delay:  20 * time.Millisecond,
	}
	pf := NewPrefetcher(cache, renderer)
	defer pf.Close()

	ch := pf.Prefetch(session, 80, 24)
	result := <-ch

	if result.Duration < 15*time.Millisecond {
		t.Errorf("Duration = %v, expected >= 15ms", result.Duration)
	}
}

func TestPrefetchCacheHitDurationZero(t *testing.T) {
	cacheDir := t.TempDir()
	cache := NewImageCache(cacheDir, 10*1024*1024)

	session := &Session{
		ID:          "ppulse-99999",
		ImagePath:   "/fake/image.png",
		ContentHash: "abcdef0123456789",
	}

	key := CacheKey{
		ContentHash: session.ContentHash,
		Protocol:    "default",
		Width:       80,
		Height:      24,
	}
	cache.Put(key, "cached")

	renderer := &mockRenderer{}
	pf := NewPrefetcher(cache, renderer)
	defer pf.Close()

	ch := pf.Prefetch(session, 80, 24)
	result := <-ch

	if result.Duration != 0 {
		t.Errorf("cache hit Duration = %v, want 0", result.Duration)
	}
}

func TestPrefetchMultipleConcurrent(t *testing.T) {
	cacheDir := t.TempDir()
	cache := NewImageCache(cacheDir, 10*1024*1024)

	renderer := &mockRenderer{
		renderFn: func(path string, width, height int) (string, error) {
			time.Sleep(10 * time.Millisecond)
			return fmt.Sprintf("rendered:%s:%dx%d", path, width, height), nil
		},
	}
	pf := NewPrefetcher(cache, renderer)

	var channels []<-chan PrefetchResult
	for i := 0; i < 10; i++ {
		session := &Session{
			ID:          fmt.Sprintf("ppulse-%d", i),
			ImagePath:   fmt.Sprintf("/fake/image%d.png", i),
			ContentHash: fmt.Sprintf("hash%04d________", i),
		}
		ch := pf.Prefetch(session, 80, 24)
		channels = append(channels, ch)
	}

	// Wait for all to complete.
	pf.Wait()

	for i, ch := range channels {
		select {
		case result := <-ch:
			if result.Error != nil {
				t.Errorf("prefetch %d error: %v", i, result.Error)
			}
		default:
			t.Errorf("prefetch %d result not ready after Wait()", i)
		}
	}
}

// --- Content Hash Tests ---

func TestContentHashDeterministic(t *testing.T) {
	dir := t.TempDir()
	content := []byte("identical-content-for-determinism-test")
	path := createTestImage(t, dir, "img.png", content)

	h1, err := contentHash(path)
	if err != nil {
		t.Fatalf("first hash: %v", err)
	}

	h2, err := contentHash(path)
	if err != nil {
		t.Fatalf("second hash: %v", err)
	}

	if h1 != h2 {
		t.Errorf("contentHash not deterministic: %q != %q", h1, h2)
	}
}

func TestContentHashDiffersForDifferentContent(t *testing.T) {
	dir := t.TempDir()
	path1 := createTestImage(t, dir, "a.png", []byte("content-a"))
	path2 := createTestImage(t, dir, "b.png", []byte("content-b"))

	h1, _ := contentHash(path1)
	h2, _ := contentHash(path2)

	if h1 == h2 {
		t.Error("different file content produced same hash")
	}
}
