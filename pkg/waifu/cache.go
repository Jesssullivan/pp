package waifu

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"sync/atomic"
)

// CacheKey uniquely identifies a rendered image output by content hash,
// rendering protocol, and target dimensions.
type CacheKey struct {
	ContentHash string
	Protocol    string
	Width       int
	Height      int
}

// filename returns the on-disk filename for this cache entry.
func (k CacheKey) filename() string {
	return fmt.Sprintf("%s_%dx%d.render", k.Protocol, k.Width, k.Height)
}

// subdir returns the subdirectory (first 8 chars of content hash).
func (k CacheKey) subdir() string {
	if len(k.ContentHash) >= 8 {
		return k.ContentHash[:8]
	}
	return k.ContentHash
}

// CacheStats reports hit/miss/size statistics for observability.
type CacheStats struct {
	Hits      uint64
	Misses    uint64
	Entries   int
	SizeBytes int64
}

// ImageCache is a disk-backed cache for rendered terminal image strings.
// Entries are keyed by content hash plus render parameters, stored as flat
// files under the cache directory.
type ImageCache struct {
	mu        sync.RWMutex
	dir       string
	maxSize   int64
	usedBytes int64
	entries   int

	hits   atomic.Uint64
	misses atomic.Uint64
}

// NewImageCache creates a disk-backed image cache rooted at dir with a
// maximum total size of maxSize bytes. It scans the directory on startup
// to compute the current usage.
func NewImageCache(dir string, maxSize int64) *ImageCache {
	if maxSize <= 0 {
		maxSize = 100 * 1024 * 1024
	}

	c := &ImageCache{
		dir:     dir,
		maxSize: maxSize,
	}

	// Scan existing cache directory for current usage.
	c.scanUsage()

	return c
}

// path returns the full filesystem path for a cache key.
func (c *ImageCache) path(key CacheKey) string {
	return filepath.Join(c.dir, key.subdir(), key.filename())
}

// Get retrieves a rendered string from the disk cache. Returns the rendered
// string and true on hit, or empty string and false on miss.
func (c *ImageCache) Get(key CacheKey) (string, bool) {
	p := c.path(key)
	data, err := os.ReadFile(p)
	if err != nil {
		c.misses.Add(1)
		return "", false
	}

	c.hits.Add(1)
	return string(data), true
}

// Put stores a rendered string to the disk cache using atomic writes.
// It writes to a temporary file in the same directory, then renames to
// the final path. This prevents partial reads.
func (c *ImageCache) Put(key CacheKey, rendered string) error {
	p := c.path(key)
	dir := filepath.Dir(p)

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create cache dir: %w", err)
	}

	// Atomic write: temp file + rename.
	tmp, err := os.CreateTemp(dir, ".render-*.tmp")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpName := tmp.Name()

	if _, err := tmp.WriteString(rendered); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return fmt.Errorf("write temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("close temp file: %w", err)
	}

	if err := os.Rename(tmpName, p); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("rename temp to final: %w", err)
	}

	// Update size tracking.
	c.mu.Lock()
	c.usedBytes += int64(len(rendered))
	c.entries++
	c.mu.Unlock()

	return nil
}

// Has checks whether a cache entry exists without reading its content.
func (c *ImageCache) Has(key CacheKey) bool {
	_, err := os.Stat(c.path(key))
	return err == nil
}

// Evict removes a specific cache entry.
func (c *ImageCache) Evict(key CacheKey) {
	p := c.path(key)
	info, err := os.Stat(p)
	if err != nil {
		return
	}

	if err := os.Remove(p); err != nil {
		return
	}

	c.mu.Lock()
	c.usedBytes -= info.Size()
	if c.usedBytes < 0 {
		c.usedBytes = 0
	}
	c.entries--
	if c.entries < 0 {
		c.entries = 0
	}
	c.mu.Unlock()

	// Try to remove empty parent directory (best effort).
	dir := filepath.Dir(p)
	os.Remove(dir) // only succeeds if empty
}

// Prune removes the oldest cache entries (by mtime) until total size is
// under maxSize. Returns an error only if the directory walk fails.
func (c *ImageCache) Prune() error {
	type fileEntry struct {
		path  string
		size  int64
		mtime int64
	}

	var files []fileEntry
	err := filepath.Walk(c.dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip unreadable entries
		}
		if info.IsDir() {
			return nil
		}
		if filepath.Ext(path) == ".render" {
			files = append(files, fileEntry{
				path:  path,
				size:  info.Size(),
				mtime: info.ModTime().UnixNano(),
			})
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("walk cache dir: %w", err)
	}

	// Sort by mtime ascending (oldest first).
	sort.Slice(files, func(i, j int) bool {
		return files[i].mtime < files[j].mtime
	})

	// Compute total size.
	var totalSize int64
	for _, f := range files {
		totalSize += f.size
	}

	// Remove oldest until under limit.
	for _, f := range files {
		if totalSize <= c.maxSize {
			break
		}
		if err := os.Remove(f.path); err != nil {
			continue
		}
		totalSize -= f.size
		// Try to remove empty parent directory.
		os.Remove(filepath.Dir(f.path))
	}

	// Re-scan to get accurate counts.
	c.scanUsage()

	return nil
}

// Stats returns current cache statistics.
func (c *ImageCache) Stats() CacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return CacheStats{
		Hits:      c.hits.Load(),
		Misses:    c.misses.Load(),
		Entries:   c.entries,
		SizeBytes: c.usedBytes,
	}
}

// scanUsage walks the cache directory and computes total size and entry count.
func (c *ImageCache) scanUsage() {
	var size int64
	var count int

	filepath.Walk(c.dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() && filepath.Ext(path) == ".render" {
			size += info.Size()
			count++
		}
		return nil
	})

	c.mu.Lock()
	c.usedBytes = size
	c.entries = count
	c.mu.Unlock()
}
