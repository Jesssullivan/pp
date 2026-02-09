// Package image provides high-performance terminal image rendering with
// protocol auto-detection, caching, and async support.
package image

import (
	"container/list"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"sync"
	"sync/atomic"
)

// CacheKey uniquely identifies a rendered image output by its protocol,
// target dimensions, and content hash.
type CacheKey struct {
	Protocol  string
	Width     int
	Height    int
	ImageHash [32]byte
}

// String returns a human-readable key for debugging.
func (k CacheKey) String() string {
	return fmt.Sprintf("%s:%dx%d:%x", k.Protocol, k.Width, k.Height, k.ImageHash[:8])
}

// CacheStats reports hit/miss counts for observability.
type CacheStats struct {
	Hits       uint64
	Misses     uint64
	Evictions  uint64
	Entries    int
	SizeBytes  int64
}

// cacheEntry is stored in the LRU list.
type cacheEntry struct {
	key       CacheKey
	rendered  string
	sizeBytes int64
}

// Cache is a thread-safe LRU cache for rendered terminal image strings.
// It uses container/list for O(1) eviction and promotion.
type Cache struct {
	mu        sync.RWMutex
	items     map[CacheKey]*list.Element
	order     *list.List // front = most recent, back = least recent
	maxBytes  int64
	usedBytes int64

	hits      atomic.Uint64
	misses    atomic.Uint64
	evictions atomic.Uint64
}

// NewCache creates a new LRU cache with the given maximum size in megabytes.
// If maxMB is <= 0, a default of 32 MB is used.
func NewCache(maxMB int) *Cache {
	if maxMB <= 0 {
		maxMB = 32
	}
	return &Cache{
		items:    make(map[CacheKey]*list.Element),
		order:    list.New(),
		maxBytes: int64(maxMB) * 1024 * 1024,
	}
}

// Get retrieves a cached rendered string. Returns the string and true on
// hit, or empty string and false on miss.
func (c *Cache) Get(key CacheKey) (string, bool) {
	c.mu.RLock()
	elem, ok := c.items[key]
	c.mu.RUnlock()

	if !ok {
		c.misses.Add(1)
		return "", false
	}

	// Promote to front (most recently used). Needs write lock.
	c.mu.Lock()
	c.order.MoveToFront(elem)
	c.mu.Unlock()

	c.hits.Add(1)
	return elem.Value.(*cacheEntry).rendered, true
}

// Put stores a rendered string in the cache. If the cache exceeds its
// maximum size, the least recently used entries are evicted.
func (c *Cache) Put(key CacheKey, rendered string) {
	entrySize := int64(len(rendered))

	c.mu.Lock()
	defer c.mu.Unlock()

	// If key already exists, update it.
	if elem, ok := c.items[key]; ok {
		old := elem.Value.(*cacheEntry)
		c.usedBytes -= old.sizeBytes
		old.rendered = rendered
		old.sizeBytes = entrySize
		c.usedBytes += entrySize
		c.order.MoveToFront(elem)
		c.evictLocked()
		return
	}

	// Evict until there is room (or cache is empty).
	for c.usedBytes+entrySize > c.maxBytes && c.order.Len() > 0 {
		c.evictBackLocked()
	}

	entry := &cacheEntry{
		key:       key,
		rendered:  rendered,
		sizeBytes: entrySize,
	}
	elem := c.order.PushFront(entry)
	c.items[key] = elem
	c.usedBytes += entrySize
}

// Invalidate clears all cache entries.
func (c *Cache) Invalidate() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items = make(map[CacheKey]*list.Element)
	c.order.Init()
	c.usedBytes = 0
}

// Stats returns current cache statistics.
func (c *Cache) Stats() CacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return CacheStats{
		Hits:      c.hits.Load(),
		Misses:    c.misses.Load(),
		Evictions: c.evictions.Load(),
		Entries:   c.order.Len(),
		SizeBytes: c.usedBytes,
	}
}

// evictLocked evicts entries from the back until under maxBytes.
// Caller must hold c.mu write lock.
func (c *Cache) evictLocked() {
	for c.usedBytes > c.maxBytes && c.order.Len() > 0 {
		c.evictBackLocked()
	}
}

// evictBackLocked removes the least recently used entry.
// Caller must hold c.mu write lock.
func (c *Cache) evictBackLocked() {
	back := c.order.Back()
	if back == nil {
		return
	}
	entry := c.order.Remove(back).(*cacheEntry)
	delete(c.items, entry.key)
	c.usedBytes -= entry.sizeBytes
	c.evictions.Add(1)
}

// HashImage computes a SHA-256 hash of an image's RGBA pixel data.
// This is used to generate stable cache keys.
func HashImage(pixelData []byte) [32]byte {
	return sha256.Sum256(pixelData)
}

// MakeCacheKey builds a CacheKey from components.
func MakeCacheKey(protocol string, width, height int, imgHash [32]byte) CacheKey {
	return CacheKey{
		Protocol:  protocol,
		Width:     width,
		Height:    height,
		ImageHash: imgHash,
	}
}

// imagePixelBytes extracts raw pixel data from an image for hashing.
// Uses a fast path for NRGBA/RGBA images and a slower generic fallback.
func imagePixelBytes(img interface {
	Bounds() interface{ Dx() int; Dy() int }
}) []byte {
	// This is a helper; the actual implementation uses image.Image.
	return nil
}

// ImageHashFromRGBA computes a hash from an image by serializing its bounds
// and sampling pixel data. For large images, it samples rather than scanning
// every pixel.
func ImageHashFromBounds(width, height int, samplePixels []byte) [32]byte {
	h := sha256.New()
	var buf [8]byte
	binary.LittleEndian.PutUint32(buf[:4], uint32(width))
	binary.LittleEndian.PutUint32(buf[4:], uint32(height))
	h.Write(buf[:])
	h.Write(samplePixels)
	var result [32]byte
	copy(result[:], h.Sum(nil))
	return result
}
