package cache

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func newTestStore(t *testing.T, opts ...func(*StoreConfig)) *Store {
	t.Helper()
	cfg := StoreConfig{
		Dir:             t.TempDir(),
		MaxSizeMB:       50,
		DefaultTTL:      time.Hour,
		CleanupInterval: time.Hour, // long interval so tests control cleanup
	}
	for _, o := range opts {
		o(&cfg)
	}
	s, err := NewStore(cfg)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

// --- Basic Put/Get ---

func TestPutGetRoundTrip(t *testing.T) {
	s := newTestStore(t)

	data := []byte(`{"name":"test","count":42}`)
	if err := s.Put("mykey", data); err != nil {
		t.Fatalf("Put: %v", err)
	}

	got, ok := s.Get("mykey")
	if !ok {
		t.Fatal("expected hit")
	}
	if string(got) != string(data) {
		t.Errorf("round-trip mismatch: got %q, want %q", got, data)
	}
}

func TestPutStringGetStringRoundTrip(t *testing.T) {
	s := newTestStore(t)

	if err := s.PutString("greeting", "hello world"); err != nil {
		t.Fatalf("PutString: %v", err)
	}

	got, ok := s.GetString("greeting")
	if !ok {
		t.Fatal("expected hit")
	}
	if got != "hello world" {
		t.Errorf("got %q, want %q", got, "hello world")
	}
}

func TestGetMissingKeyReturnsFalse(t *testing.T) {
	s := newTestStore(t)

	_, ok := s.Get("nonexistent")
	if ok {
		t.Error("expected miss for nonexistent key")
	}
}

// --- TTL ---

func TestGetReturnsFalseForExpiredEntry(t *testing.T) {
	s := newTestStore(t, func(cfg *StoreConfig) {
		cfg.DefaultTTL = 50 * time.Millisecond
	})

	if err := s.Put("expiring", []byte("data")); err != nil {
		t.Fatalf("Put: %v", err)
	}

	// Wait for TTL to pass
	time.Sleep(100 * time.Millisecond)

	_, ok := s.Get("expiring")
	if ok {
		t.Error("expected miss for expired entry")
	}
}

func TestPutWithTTLRespectsCustomTTL(t *testing.T) {
	s := newTestStore(t, func(cfg *StoreConfig) {
		cfg.DefaultTTL = time.Hour // default is long
	})

	// Store with a very short custom TTL
	if err := s.PutWithTTL("short", []byte("temp"), 50*time.Millisecond); err != nil {
		t.Fatalf("PutWithTTL: %v", err)
	}

	// Should be readable immediately
	_, ok := s.Get("short")
	if !ok {
		t.Fatal("expected hit before TTL expires")
	}

	time.Sleep(100 * time.Millisecond)

	_, ok = s.Get("short")
	if ok {
		t.Error("expected miss after custom TTL expires")
	}
}

func TestZeroTTLNeverExpires(t *testing.T) {
	s := newTestStore(t, func(cfg *StoreConfig) {
		cfg.DefaultTTL = 0 // no expiry
	})

	if err := s.Put("forever", []byte("immortal")); err != nil {
		t.Fatalf("Put: %v", err)
	}

	// Even after some time, should still be present
	got, ok := s.Get("forever")
	if !ok {
		t.Fatal("expected hit for zero-TTL entry")
	}
	if string(got) != "immortal" {
		t.Errorf("got %q, want %q", got, "immortal")
	}
}

// --- Delete ---

func TestDeleteRemovesEntry(t *testing.T) {
	s := newTestStore(t)

	if err := s.Put("doomed", []byte("value")); err != nil {
		t.Fatalf("Put: %v", err)
	}

	if err := s.Delete("doomed"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	_, ok := s.Get("doomed")
	if ok {
		t.Error("expected miss after Delete")
	}
}

func TestDeleteNonexistentKeyIsNoOp(t *testing.T) {
	s := newTestStore(t)

	if err := s.Delete("ghost"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
}

// --- Has ---

func TestHasChecksExistence(t *testing.T) {
	s := newTestStore(t)

	if s.Has("missing") {
		t.Error("expected Has=false for missing key")
	}

	if err := s.Put("present", []byte("here")); err != nil {
		t.Fatalf("Put: %v", err)
	}

	if !s.Has("present") {
		t.Error("expected Has=true for present key")
	}
}

func TestHasReturnsFalseForExpired(t *testing.T) {
	s := newTestStore(t, func(cfg *StoreConfig) {
		cfg.DefaultTTL = 50 * time.Millisecond
	})

	if err := s.Put("ephemeral", []byte("gone")); err != nil {
		t.Fatalf("Put: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	if s.Has("ephemeral") {
		t.Error("expected Has=false for expired key")
	}
}

// --- Keys ---

func TestKeysListsNonExpiredEntries(t *testing.T) {
	s := newTestStore(t)

	for _, k := range []string{"alpha", "beta", "gamma"} {
		if err := s.PutString(k, k); err != nil {
			t.Fatalf("PutString %s: %v", k, err)
		}
	}

	keys := s.Keys()
	if len(keys) != 3 {
		t.Fatalf("expected 3 keys, got %d: %v", len(keys), keys)
	}

	want := map[string]bool{"alpha": true, "beta": true, "gamma": true}
	for _, k := range keys {
		if !want[k] {
			t.Errorf("unexpected key: %s", k)
		}
	}
}

func TestKeysExcludesExpired(t *testing.T) {
	s := newTestStore(t, func(cfg *StoreConfig) {
		cfg.DefaultTTL = time.Hour
	})

	if err := s.PutString("long-lived", "yes"); err != nil {
		t.Fatalf("PutString: %v", err)
	}
	if err := s.PutWithTTL("short-lived", []byte("no"), 50*time.Millisecond); err != nil {
		t.Fatalf("PutWithTTL: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	keys := s.Keys()
	if len(keys) != 1 {
		t.Fatalf("expected 1 key, got %d: %v", len(keys), keys)
	}
	if keys[0] != "long-lived" {
		t.Errorf("expected key 'long-lived', got %q", keys[0])
	}
}

// --- Clear ---

func TestClearRemovesEverything(t *testing.T) {
	s := newTestStore(t)

	for _, k := range []string{"a", "b", "c"} {
		if err := s.PutString(k, k); err != nil {
			t.Fatalf("PutString %s: %v", k, err)
		}
	}

	if err := s.Clear(); err != nil {
		t.Fatalf("Clear: %v", err)
	}

	if keys := s.Keys(); len(keys) != 0 {
		t.Errorf("expected no keys after Clear, got %v", keys)
	}
	if s.Size() != 0 {
		t.Errorf("expected size=0 after Clear, got %d", s.Size())
	}
}

// --- Size ---

func TestSizeTrackingAccurate(t *testing.T) {
	s := newTestStore(t)

	data1 := []byte("hello")
	data2 := []byte("world!!")
	expected := int64(len(data1) + len(data2))

	if err := s.Put("k1", data1); err != nil {
		t.Fatalf("Put: %v", err)
	}
	if err := s.Put("k2", data2); err != nil {
		t.Fatalf("Put: %v", err)
	}

	if got := s.Size(); got != expected {
		t.Errorf("expected size=%d, got %d", expected, got)
	}

	// Delete one entry
	if err := s.Delete("k1"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if got := s.Size(); got != int64(len(data2)) {
		t.Errorf("after delete: expected size=%d, got %d", len(data2), got)
	}
}

func TestSizeUpdatesOnOverwrite(t *testing.T) {
	s := newTestStore(t)

	if err := s.Put("key", []byte("short")); err != nil {
		t.Fatalf("Put: %v", err)
	}
	if err := s.Put("key", []byte("much longer value")); err != nil {
		t.Fatalf("Put: %v", err)
	}

	expected := int64(len("much longer value"))
	if got := s.Size(); got != expected {
		t.Errorf("expected size=%d after overwrite, got %d", expected, got)
	}
}

// --- Stats ---

func TestStatsHitMissCounting(t *testing.T) {
	s := newTestStore(t)

	if err := s.Put("hit", []byte("val")); err != nil {
		t.Fatalf("Put: %v", err)
	}

	// 1 hit
	s.Get("hit")
	// 2 misses
	s.Get("miss1")
	s.Get("miss2")

	st := s.Stats()
	if st.Hits != 1 {
		t.Errorf("expected 1 hit, got %d", st.Hits)
	}
	if st.Misses != 2 {
		t.Errorf("expected 2 misses, got %d", st.Misses)
	}
	if st.Entries != 1 {
		t.Errorf("expected 1 entry, got %d", st.Entries)
	}
}

// --- LRU Eviction ---

func TestLRUEvictionOldestEvicted(t *testing.T) {
	// 1 MB max, each entry ~500KB, so only 2 fit
	s := newTestStore(t, func(cfg *StoreConfig) {
		cfg.MaxSizeMB = 1
	})

	big := make([]byte, 500*1024) // 500 KB

	// Put 3 entries; first one should be evicted
	for _, k := range []string{"first", "second", "third"} {
		if err := s.Put(k, big); err != nil {
			t.Fatalf("Put %s: %v", k, err)
		}
	}

	// "first" should have been evicted (LRU)
	if _, ok := s.Get("first"); ok {
		t.Error("expected 'first' to be evicted")
	}

	// "second" and "third" should still exist
	if _, ok := s.Get("second"); !ok {
		t.Error("expected 'second' to still exist")
	}
	if _, ok := s.Get("third"); !ok {
		t.Error("expected 'third' to still exist")
	}

	st := s.Stats()
	if st.Evictions < 1 {
		t.Errorf("expected at least 1 eviction, got %d", st.Evictions)
	}
}

func TestLRUPromotionAccessedEntryNotEvicted(t *testing.T) {
	// 1 MB max, each entry ~400KB, so 2 fit (with some overhead margin)
	s := newTestStore(t, func(cfg *StoreConfig) {
		cfg.MaxSizeMB = 1
	})

	big := make([]byte, 400*1024) // 400 KB

	if err := s.Put("first", big); err != nil {
		t.Fatalf("Put: %v", err)
	}
	if err := s.Put("second", big); err != nil {
		t.Fatalf("Put: %v", err)
	}

	// Access "first" to promote it to the front of LRU
	s.Get("first")

	// Adding "third" should evict "second" (back of LRU), not "first"
	if err := s.Put("third", big); err != nil {
		t.Fatalf("Put: %v", err)
	}

	if _, ok := s.Get("first"); !ok {
		t.Error("expected 'first' to survive (was promoted via access)")
	}
	if _, ok := s.Get("second"); ok {
		t.Error("expected 'second' to be evicted (was at back of LRU)")
	}
}

// --- Atomic Writes ---

func TestAtomicWriteNoPartialReads(t *testing.T) {
	s := newTestStore(t)

	data := []byte(`{"complete":"yes","value":12345}`)
	if err := s.Put("atomic", data); err != nil {
		t.Fatalf("Put: %v", err)
	}

	// Read the raw file and verify it is complete
	h := hashKey("atomic")
	raw, err := os.ReadFile(filepath.Join(s.cfg.Dir, h+".cache"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	if string(raw) != string(data) {
		t.Errorf("file content mismatch: got %q", raw)
	}

	// Verify no temp files remain
	entries, err := os.ReadDir(s.cfg.Dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".tmp-") {
			t.Errorf("temp file left behind: %s", e.Name())
		}
	}
}

// --- Typed Access ---

func TestGetTypedPutTypedWithStruct(t *testing.T) {
	s := newTestStore(t)

	type Widget struct {
		Name  string  `json:"name"`
		Score float64 `json:"score"`
	}

	original := Widget{Name: "sprocket", Score: 99.5}

	if err := PutTyped(s, "widget", original); err != nil {
		t.Fatalf("PutTyped: %v", err)
	}

	got, ok := GetTyped[Widget](s, "widget")
	if !ok {
		t.Fatal("expected hit")
	}
	if got != original {
		t.Errorf("typed round-trip mismatch: got %+v, want %+v", got, original)
	}
}

func TestGetTypedInvalidJSONReturnsFalse(t *testing.T) {
	s := newTestStore(t)

	// Store raw bytes that are not valid JSON for the target type
	if err := s.Put("badjson", []byte("not json at all")); err != nil {
		t.Fatalf("Put: %v", err)
	}

	type Target struct {
		Field string `json:"field"`
	}

	_, ok := GetTyped[Target](s, "badjson")
	if ok {
		t.Error("expected false for invalid JSON")
	}
}

func TestPutTypedWithNonSerializableReturnsError(t *testing.T) {
	s := newTestStore(t)

	// math.NaN() causes json.Marshal to fail
	err := PutTyped(s, "bad", math.NaN())
	if err == nil {
		t.Error("expected error for non-serializable value (NaN)")
	}
}

func TestPutTypedWithTTL(t *testing.T) {
	s := newTestStore(t)

	type Item struct {
		Value int `json:"value"`
	}

	if err := PutTypedWithTTL(s, "ttl-typed", Item{Value: 42}, 50*time.Millisecond); err != nil {
		t.Fatalf("PutTypedWithTTL: %v", err)
	}

	got, ok := GetTyped[Item](s, "ttl-typed")
	if !ok {
		t.Fatal("expected hit before TTL")
	}
	if got.Value != 42 {
		t.Errorf("expected 42, got %d", got.Value)
	}

	time.Sleep(100 * time.Millisecond)

	_, ok = GetTyped[Item](s, "ttl-typed")
	if ok {
		t.Error("expected miss after TTL")
	}
}

// --- Concurrent Access ---

func TestConcurrentAccess(t *testing.T) {
	s := newTestStore(t)

	const goroutines = 20
	const iterations = 50

	var wg sync.WaitGroup
	wg.Add(goroutines * 2) // writers + readers

	// Writers
	for g := 0; g < goroutines; g++ {
		go func(id int) {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				key := fmt.Sprintf("key-%d", id%5) // overlap keys
				data := []byte(fmt.Sprintf("writer-%d-iter-%d", id, i))
				if err := s.Put(key, data); err != nil {
					t.Errorf("goroutine %d: Put: %v", id, err)
					return
				}
			}
		}(g)
	}

	// Readers
	for g := 0; g < goroutines; g++ {
		go func(id int) {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				key := fmt.Sprintf("key-%d", id%5)
				s.Get(key)
				s.Has(key)
			}
		}(g)
	}

	wg.Wait()

	// After all goroutines finish, every surviving key should be readable
	for _, k := range s.Keys() {
		data, ok := s.Get(k)
		if !ok {
			t.Errorf("key %q in Keys() but Get returned false", k)
		}
		if len(data) == 0 {
			t.Errorf("key %q returned empty data", k)
		}
	}
}

// --- Startup Scan ---

func TestStartupScanRebuildsFromExistingFiles(t *testing.T) {
	dir := t.TempDir()

	// Create first store, write data, close it
	s1, err := NewStore(StoreConfig{
		Dir:             dir,
		MaxSizeMB:       50,
		DefaultTTL:      time.Hour,
		CleanupInterval: time.Hour,
	})
	if err != nil {
		t.Fatalf("NewStore (first): %v", err)
	}

	if err := s1.PutString("persist-a", "value-a"); err != nil {
		t.Fatalf("PutString: %v", err)
	}
	if err := s1.PutString("persist-b", "value-b"); err != nil {
		t.Fatalf("PutString: %v", err)
	}

	_ = s1.Close()

	// Create second store from same directory
	s2, err := NewStore(StoreConfig{
		Dir:             dir,
		MaxSizeMB:       50,
		DefaultTTL:      time.Hour,
		CleanupInterval: time.Hour,
	})
	if err != nil {
		t.Fatalf("NewStore (second): %v", err)
	}
	defer func() { _ = s2.Close() }()

	// Data should be available
	got, ok := s2.GetString("persist-a")
	if !ok {
		t.Fatal("expected 'persist-a' to survive restart")
	}
	if got != "value-a" {
		t.Errorf("got %q, want %q", got, "value-a")
	}

	got, ok = s2.GetString("persist-b")
	if !ok {
		t.Fatal("expected 'persist-b' to survive restart")
	}
	if got != "value-b" {
		t.Errorf("got %q, want %q", got, "value-b")
	}

	// Size should reflect both entries
	expectedSize := int64(len("value-a") + len("value-b"))
	if s2.Size() != expectedSize {
		t.Errorf("size after scan: got %d, want %d", s2.Size(), expectedSize)
	}
}

// --- Hash ---

func TestHashKeyDeterministic(t *testing.T) {
	h1 := hashKey("test-key")
	h2 := hashKey("test-key")
	if h1 != h2 {
		t.Errorf("hash not deterministic: %q != %q", h1, h2)
	}

	h3 := hashKey("different-key")
	if h1 == h3 {
		t.Errorf("different keys produced same hash: %q", h1)
	}
}

func TestHashKeyHandlesSpecialCharacters(t *testing.T) {
	special := []string{
		"key with spaces",
		"key/with/slashes",
		"key\\with\\backslashes",
		"key:with:colons",
		"key?with=query&params",
		"key\twith\ttabs",
		"key\nwith\nnewlines",
		"../../../etc/passwd",
		"",
		strings.Repeat("x", 10000),
	}

	seen := make(map[string]string)
	for _, key := range special {
		h := hashKey(key)
		// Must be 16 hex chars
		if len(h) != 16 {
			t.Errorf("hash of %q has length %d, want 16", key, len(h))
		}
		// Must be valid hex
		for _, c := range h {
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
				t.Errorf("hash of %q contains non-hex char %q", key, string(c))
			}
		}
		// Must be unique (within this small set)
		if prev, dup := seen[h]; dup {
			t.Errorf("hash collision: %q and %q both hash to %q", prev, key, h)
		}
		seen[h] = key
	}
}

// --- Close ---

func TestCloseStopsCleanupGoroutine(t *testing.T) {
	s := newTestStore(t, func(cfg *StoreConfig) {
		cfg.CleanupInterval = 10 * time.Millisecond
	})

	// Let a few ticks happen
	time.Sleep(50 * time.Millisecond)

	// Close should return without hanging
	done := make(chan struct{})
	go func() {
		_ = s.Close()
		close(done)
	}()

	select {
	case <-done:
		// success
	case <-time.After(2 * time.Second):
		t.Fatal("Close did not return within 2 seconds")
	}
}

// --- Background Cleanup ---

func TestBackgroundCleanupSweepsExpired(t *testing.T) {
	s := newTestStore(t, func(cfg *StoreConfig) {
		cfg.DefaultTTL = 50 * time.Millisecond
		cfg.CleanupInterval = 100 * time.Millisecond
	})

	if err := s.Put("sweep-me", []byte("temporary")); err != nil {
		t.Fatalf("Put: %v", err)
	}

	// Wait for TTL + cleanup interval + buffer
	time.Sleep(300 * time.Millisecond)

	// The background sweep should have removed the entry
	if s.Has("sweep-me") {
		t.Error("expected expired entry to be swept by background cleanup")
	}

	// Verify files are actually removed from disk
	h := hashKey("sweep-me")
	if _, err := os.Stat(filepath.Join(s.cfg.Dir, h+".cache")); !os.IsNotExist(err) {
		t.Error("expected data file to be removed after sweep")
	}
	if _, err := os.Stat(filepath.Join(s.cfg.Dir, h+".meta")); !os.IsNotExist(err) {
		t.Error("expected meta file to be removed after sweep")
	}
}

// --- Edge Cases ---

func TestOverwriteExistingKey(t *testing.T) {
	s := newTestStore(t)

	if err := s.PutString("key", "original"); err != nil {
		t.Fatalf("PutString: %v", err)
	}
	if err := s.PutString("key", "updated"); err != nil {
		t.Fatalf("PutString: %v", err)
	}

	got, ok := s.GetString("key")
	if !ok {
		t.Fatal("expected hit")
	}
	if got != "updated" {
		t.Errorf("got %q, want %q", got, "updated")
	}
}

func TestEmptyValue(t *testing.T) {
	s := newTestStore(t)

	if err := s.Put("empty", []byte{}); err != nil {
		t.Fatalf("Put: %v", err)
	}

	got, ok := s.Get("empty")
	if !ok {
		t.Fatal("expected hit for empty value")
	}
	if len(got) != 0 {
		t.Errorf("expected empty slice, got %d bytes", len(got))
	}
}

func TestNewStoreCreatesDirectory(t *testing.T) {
	base := t.TempDir()
	nested := filepath.Join(base, "deep", "nested", "cache")

	s, err := NewStore(StoreConfig{
		Dir:             nested,
		CleanupInterval: time.Hour,
	})
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	defer func() { _ = s.Close() }()

	info, err := os.Stat(nested)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if !info.IsDir() {
		t.Error("expected directory to be created")
	}
}

func TestStatsEvictionCount(t *testing.T) {
	s := newTestStore(t, func(cfg *StoreConfig) {
		cfg.MaxSizeMB = 1
	})

	big := make([]byte, 600*1024) // 600KB, only 1 fits in 1MB

	if err := s.Put("a", big); err != nil {
		t.Fatalf("Put a: %v", err)
	}
	if err := s.Put("b", big); err != nil {
		t.Fatalf("Put b: %v", err)
	}

	st := s.Stats()
	if st.Evictions < 1 {
		t.Errorf("expected at least 1 eviction, got %d", st.Evictions)
	}
}

func TestScanSkipsExpiredOnStartup(t *testing.T) {
	dir := t.TempDir()

	// Create store and add an entry with very short TTL
	s1, err := NewStore(StoreConfig{
		Dir:             dir,
		MaxSizeMB:       50,
		DefaultTTL:      50 * time.Millisecond,
		CleanupInterval: time.Hour,
	})
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	if err := s1.PutString("expired-on-restart", "gone"); err != nil {
		t.Fatalf("PutString: %v", err)
	}
	_ = s1.Close()

	// Wait for TTL to pass
	time.Sleep(100 * time.Millisecond)

	// Open new store -- scan should skip the expired entry
	s2, err := NewStore(StoreConfig{
		Dir:             dir,
		MaxSizeMB:       50,
		DefaultTTL:      time.Hour,
		CleanupInterval: time.Hour,
	})
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	defer func() { _ = s2.Close() }()

	if s2.Has("expired-on-restart") {
		t.Error("expected expired entry to be skipped during scan")
	}
	if s2.Size() != 0 {
		t.Errorf("expected size=0 after scan skips expired, got %d", s2.Size())
	}
}

func TestMetaFileIntegrity(t *testing.T) {
	s := newTestStore(t)

	data := []byte("test-data-payload")
	if err := s.Put("meta-check", data); err != nil {
		t.Fatalf("Put: %v", err)
	}

	h := hashKey("meta-check")
	raw, err := os.ReadFile(filepath.Join(s.cfg.Dir, h+".meta"))
	if err != nil {
		t.Fatalf("ReadFile meta: %v", err)
	}

	var m entryMeta
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("Unmarshal meta: %v", err)
	}

	if m.Key != "meta-check" {
		t.Errorf("meta key: got %q, want %q", m.Key, "meta-check")
	}
	if m.Size != int64(len(data)) {
		t.Errorf("meta size: got %d, want %d", m.Size, len(data))
	}
	if m.Created == 0 {
		t.Error("meta created should not be zero")
	}
}
