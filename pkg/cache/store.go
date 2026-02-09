package cache

import (
	"container/list"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// StoreConfig holds configuration for a cache Store.
type StoreConfig struct {
	// Dir is the directory path where cache files are stored.
	Dir string

	// MaxSizeMB is the maximum total cache size in megabytes. Default: 50.
	MaxSizeMB int

	// DefaultTTL is the default time-to-live for cache entries. Default: 1 hour.
	// A value of 0 means entries never expire by TTL.
	DefaultTTL time.Duration

	// CleanupInterval is how often the background goroutine sweeps for
	// expired entries. Default: 5 minutes.
	CleanupInterval time.Duration
}

// CacheStats holds runtime statistics for a cache Store.
type CacheStats struct {
	Hits      int64
	Misses    int64
	Evictions int64
	Size      int64
	Entries   int
}

// entryMeta is the JSON structure persisted alongside each cache entry.
type entryMeta struct {
	Key     string `json:"key"`
	Created int64  `json:"created"` // UnixNano
	TTLNS   int64  `json:"ttl_ns"`  // 0 = no TTL
	Size    int64  `json:"size"`    // data file size in bytes
}

// lruEntry is the value stored in each list.Element.
type lruEntry struct {
	hash string
	key  string
	size int64
}

// Store is a disk-backed key-value cache with LRU eviction and TTL-based
// expiry. Each entry is stored as two files: {hash}.cache (data) and
// {hash}.meta (JSON metadata). Writes are atomic via temp-file-then-rename.
type Store struct {
	cfg StoreConfig

	mu       sync.RWMutex
	lru      *list.List               // front = most recently used
	items    map[string]*list.Element // hash -> *list.Element (value is *lruEntry)
	curSize  int64                    // total bytes of data files
	hits     int64
	misses   int64
	evictions int64

	done      chan struct{} // signals cleanup goroutine to stop
	closeOnce sync.Once
	wg        sync.WaitGroup
}

// NewStore creates a new cache Store. The cache directory is created with
// 0755 permissions if it does not exist. Existing entries are loaded from
// disk to rebuild the LRU state.
func NewStore(cfg StoreConfig) (*Store, error) {
	if cfg.MaxSizeMB <= 0 {
		cfg.MaxSizeMB = 50
	}
	if cfg.DefaultTTL < 0 {
		cfg.DefaultTTL = time.Hour
	}
	if cfg.DefaultTTL == 0 {
		// 0 means no expiry, leave as 0
	}
	if cfg.CleanupInterval <= 0 {
		cfg.CleanupInterval = 5 * time.Minute
	}

	if err := os.MkdirAll(cfg.Dir, 0755); err != nil {
		return nil, fmt.Errorf("cache: create directory %s: %w", cfg.Dir, err)
	}

	s := &Store{
		cfg:   cfg,
		lru:   list.New(),
		items: make(map[string]*list.Element),
		done:  make(chan struct{}),
	}

	if err := s.scanDir(); err != nil {
		return nil, fmt.Errorf("cache: scan directory: %w", err)
	}

	s.wg.Add(1)
	go s.cleanupLoop()

	return s, nil
}

// Get retrieves the raw bytes for key. Returns (nil, false) if the key is
// missing or expired. On a hit, the entry is promoted to the front of the LRU.
func (s *Store) Get(key string) ([]byte, bool) {
	h := hashKey(key)

	s.mu.Lock()
	defer s.mu.Unlock()

	elem, ok := s.items[h]
	if !ok {
		s.misses++
		return nil, false
	}

	// Check TTL
	meta, err := s.readMeta(h)
	if err != nil {
		s.misses++
		return nil, false
	}
	if s.isExpired(meta) {
		s.removeLocked(h, elem)
		s.misses++
		return nil, false
	}

	data, err := os.ReadFile(s.dataPath(h))
	if err != nil {
		s.misses++
		return nil, false
	}

	// Promote in LRU
	s.lru.MoveToFront(elem)
	s.hits++
	return data, true
}

// GetString is a convenience method that returns the cached value as a string.
func (s *Store) GetString(key string) (string, bool) {
	data, ok := s.Get(key)
	if !ok {
		return "", false
	}
	return string(data), true
}

// Put stores value under key with the store's default TTL.
func (s *Store) Put(key string, value []byte) error {
	return s.PutWithTTL(key, value, s.cfg.DefaultTTL)
}

// PutWithTTL stores value under key with a custom TTL. A TTL of 0 means the
// entry never expires by time (only by LRU eviction or explicit deletion).
func (s *Store) PutWithTTL(key string, value []byte, ttl time.Duration) error {
	h := hashKey(key)
	size := int64(len(value))

	meta := entryMeta{
		Key:     key,
		Created: time.Now().UnixNano(),
		TTLNS:   int64(ttl),
		Size:    size,
	}

	metaBytes, err := json.Marshal(meta)
	if err != nil {
		return fmt.Errorf("cache: marshal meta for %q: %w", key, err)
	}

	// Atomic write: data file
	if err := atomicWrite(s.dataPath(h), value, s.cfg.Dir); err != nil {
		return fmt.Errorf("cache: write data for %q: %w", key, err)
	}

	// Atomic write: meta file
	if err := atomicWrite(s.metaPath(h), metaBytes, s.cfg.Dir); err != nil {
		// Best effort: remove the data file we just wrote
		_ = os.Remove(s.dataPath(h))
		return fmt.Errorf("cache: write meta for %q: %w", key, err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// If key already exists, update in place
	if elem, ok := s.items[h]; ok {
		entry := elem.Value.(*lruEntry)
		s.curSize -= entry.size
		entry.size = size
		s.curSize += size
		s.lru.MoveToFront(elem)
	} else {
		entry := &lruEntry{hash: h, key: key, size: size}
		elem := s.lru.PushFront(entry)
		s.items[h] = elem
		s.curSize += size
	}

	// Evict if over max size
	s.evictLocked()

	return nil
}

// PutString is a convenience method that stores a string value with the default TTL.
func (s *Store) PutString(key, value string) error {
	return s.Put(key, []byte(value))
}

// Delete removes a specific entry from the cache.
func (s *Store) Delete(key string) error {
	h := hashKey(key)

	s.mu.Lock()
	defer s.mu.Unlock()

	elem, ok := s.items[h]
	if !ok {
		return nil
	}

	s.removeLocked(h, elem)
	return nil
}

// Has reports whether key exists and is not expired.
func (s *Store) Has(key string) bool {
	h := hashKey(key)

	s.mu.RLock()
	defer s.mu.RUnlock()

	elem, ok := s.items[h]
	if !ok {
		return false
	}

	meta, err := s.readMeta(h)
	if err != nil {
		return false
	}
	if s.isExpired(meta) {
		// Do not modify the LRU under a read lock; the entry will be
		// cleaned up by the background goroutine or the next write.
		// But we can upgrade to a write lock to clean it now.
		s.mu.RUnlock()
		s.mu.Lock()
		// Re-check after lock upgrade
		if elem2, ok2 := s.items[h]; ok2 {
			s.removeLocked(h, elem2)
		}
		s.mu.Unlock()
		s.mu.RLock()
		_ = elem // suppress unused warning
		return false
	}

	return true
}

// Keys returns a list of all non-expired keys in the cache.
func (s *Store) Keys() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var keys []string
	for h, elem := range s.items {
		meta, err := s.readMeta(h)
		if err != nil {
			continue
		}
		if s.isExpired(meta) {
			continue
		}
		entry := elem.Value.(*lruEntry)
		keys = append(keys, entry.key)
	}
	return keys
}

// Clear removes all entries from the cache.
func (s *Store) Clear() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	entries, err := os.ReadDir(s.cfg.Dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("cache: clear read dir: %w", err)
	}

	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasSuffix(name, ".cache") || strings.HasSuffix(name, ".meta") || strings.HasPrefix(name, ".tmp-") {
			_ = os.Remove(filepath.Join(s.cfg.Dir, name))
		}
	}

	s.lru.Init()
	s.items = make(map[string]*list.Element)
	s.curSize = 0

	return nil
}

// Size returns the current total size of cached data in bytes.
func (s *Store) Size() int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.curSize
}

// Stats returns a snapshot of cache statistics.
func (s *Store) Stats() CacheStats {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return CacheStats{
		Hits:      s.hits,
		Misses:    s.misses,
		Evictions: s.evictions,
		Size:      s.curSize,
		Entries:   s.lru.Len(),
	}
}

// Close stops the background cleanup goroutine and waits for it to finish.
// It is safe to call Close multiple times.
func (s *Store) Close() error {
	s.closeOnce.Do(func() {
		close(s.done)
	})
	s.wg.Wait()
	return nil
}

// --- internal helpers ---

func (s *Store) dataPath(hash string) string {
	return filepath.Join(s.cfg.Dir, hash+".cache")
}

func (s *Store) metaPath(hash string) string {
	return filepath.Join(s.cfg.Dir, hash+".meta")
}

func (s *Store) maxBytes() int64 {
	return int64(s.cfg.MaxSizeMB) * 1024 * 1024
}

func (s *Store) readMeta(hash string) (entryMeta, error) {
	var m entryMeta
	data, err := os.ReadFile(s.metaPath(hash))
	if err != nil {
		return m, err
	}
	if err := json.Unmarshal(data, &m); err != nil {
		return m, err
	}
	return m, nil
}

func (s *Store) isExpired(m entryMeta) bool {
	if m.TTLNS <= 0 {
		return false
	}
	created := time.Unix(0, m.Created)
	return time.Since(created) > time.Duration(m.TTLNS)
}

// removeLocked removes an entry from the LRU and deletes its files.
// Caller must hold s.mu write lock.
func (s *Store) removeLocked(hash string, elem *list.Element) {
	entry := elem.Value.(*lruEntry)
	s.curSize -= entry.size
	s.lru.Remove(elem)
	delete(s.items, hash)
	_ = os.Remove(s.dataPath(hash))
	_ = os.Remove(s.metaPath(hash))
}

// evictLocked removes entries until curSize is within maxBytes.
// Expired entries are evicted first, then LRU from the back.
// Caller must hold s.mu write lock.
func (s *Store) evictLocked() {
	maxB := s.maxBytes()

	// First pass: remove expired entries
	if s.curSize > maxB {
		var toRemove []struct {
			hash string
			elem *list.Element
		}
		for h, elem := range s.items {
			meta, err := s.readMeta(h)
			if err != nil || s.isExpired(meta) {
				toRemove = append(toRemove, struct {
					hash string
					elem *list.Element
				}{h, elem})
			}
		}
		for _, r := range toRemove {
			s.removeLocked(r.hash, r.elem)
			s.evictions++
			if s.curSize <= maxB {
				return
			}
		}
	}

	// Second pass: evict LRU from back
	for s.curSize > maxB && s.lru.Len() > 0 {
		back := s.lru.Back()
		if back == nil {
			break
		}
		entry := back.Value.(*lruEntry)
		s.removeLocked(entry.hash, back)
		s.evictions++
	}
}

// scanDir reads existing .meta files from the cache directory and rebuilds
// the in-memory LRU index. Entries are added in no particular order; this
// is acceptable because the LRU will self-correct as entries are accessed.
func (s *Store) scanDir() error {
	entries, err := os.ReadDir(s.cfg.Dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".meta") {
			continue
		}

		hash := strings.TrimSuffix(name, ".meta")

		// Verify the companion data file exists
		if _, err := os.Stat(s.dataPath(hash)); err != nil {
			// Orphaned meta file, remove it
			_ = os.Remove(s.metaPath(hash))
			continue
		}

		meta, err := s.readMeta(hash)
		if err != nil {
			// Corrupted meta file, remove both files
			_ = os.Remove(s.metaPath(hash))
			_ = os.Remove(s.dataPath(hash))
			continue
		}

		// Skip expired entries during scan
		if s.isExpired(meta) {
			_ = os.Remove(s.metaPath(hash))
			_ = os.Remove(s.dataPath(hash))
			continue
		}

		entry := &lruEntry{
			hash: hash,
			key:  meta.Key,
			size: meta.Size,
		}
		elem := s.lru.PushBack(entry)
		s.items[hash] = elem
		s.curSize += meta.Size
	}

	return nil
}

// cleanupLoop periodically sweeps for expired entries.
func (s *Store) cleanupLoop() {
	defer s.wg.Done()
	ticker := time.NewTicker(s.cfg.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.done:
			return
		case <-ticker.C:
			s.sweepExpired()
		}
	}
}

// sweepExpired removes all expired entries.
func (s *Store) sweepExpired() {
	s.mu.Lock()
	defer s.mu.Unlock()

	var toRemove []struct {
		hash string
		elem *list.Element
	}

	for h, elem := range s.items {
		meta, err := s.readMeta(h)
		if err != nil || s.isExpired(meta) {
			toRemove = append(toRemove, struct {
				hash string
				elem *list.Element
			}{h, elem})
		}
	}

	for _, r := range toRemove {
		s.removeLocked(r.hash, r.elem)
		s.evictions++
	}
}

// atomicWrite writes data to path via a temporary file and rename.
func atomicWrite(path string, data []byte, tmpDir string) error {
	tmp, err := os.CreateTemp(tmpDir, ".tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()

	success := false
	defer func() {
		if !success {
			_ = os.Remove(tmpName)
		}
	}()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}

	if err := tmp.Close(); err != nil {
		return err
	}

	if err := os.Rename(tmpName, path); err != nil {
		return err
	}

	success = true
	return nil
}
