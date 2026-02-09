package layout

import (
	"fmt"
	"sync"
)

// cacheKey uniquely identifies a layout computation.
type cacheKey struct {
	direction   Direction
	constraints string // deterministic serialization of constraints
	areaW       int
	areaH       int
	areaX       int
	areaY       int
	flex        Flex
	spacing     int
	margin      int
}

// LayoutCache stores previously computed layout results to avoid
// recomputing identical layouts every render frame.
// It is safe for concurrent use.
type LayoutCache struct {
	mu      sync.RWMutex
	entries map[cacheKey][]Rect
}

// NewLayoutCache creates an empty cache.
func NewLayoutCache() *LayoutCache {
	return &LayoutCache{
		entries: make(map[cacheKey][]Rect),
	}
}

// Get looks up a cached result. Returns nil if not found.
func (c *LayoutCache) Get(l *Layout, area Rect) []Rect {
	key := makeKey(l, area)
	c.mu.RLock()
	defer c.mu.RUnlock()
	rects, ok := c.entries[key]
	if !ok {
		return nil
	}
	// Return a copy so callers cannot mutate the cache.
	cp := make([]Rect, len(rects))
	copy(cp, rects)
	return cp
}

// Put stores a layout result in the cache.
func (c *LayoutCache) Put(l *Layout, area Rect, rects []Rect) {
	key := makeKey(l, area)
	cp := make([]Rect, len(rects))
	copy(cp, rects)
	c.mu.Lock()
	c.entries[key] = cp
	c.mu.Unlock()
}

// Invalidate clears all cached entries. Call this on terminal resize.
func (c *LayoutCache) Invalidate() {
	c.mu.Lock()
	c.entries = make(map[cacheKey][]Rect)
	c.mu.Unlock()
}

// Len returns the number of cached entries.
func (c *LayoutCache) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.entries)
}

// SplitCached performs l.Split(area) with caching. If a cached result
// exists it is returned directly; otherwise the result is computed,
// cached, and returned.
func (c *LayoutCache) SplitCached(l *Layout, area Rect) []Rect {
	if cached := c.Get(l, area); cached != nil {
		return cached
	}
	result := l.Split(area)
	c.Put(l, area, result)
	return result
}

// makeKey builds a deterministic cache key from a Layout and area.
func makeKey(l *Layout, area Rect) cacheKey {
	return cacheKey{
		direction:   l.direction,
		constraints: hashConstraints(l.constraints),
		areaW:       area.Width,
		areaH:       area.Height,
		areaX:       area.X,
		areaY:       area.Y,
		flex:        l.flex,
		spacing:     l.spacing,
		margin:      l.margin,
	}
}

// hashConstraints produces a deterministic string representation of
// a constraint slice, suitable for use as a map key.
func hashConstraints(cs []Constraint) string {
	if len(cs) == 0 {
		return ""
	}
	// Pre-allocate a reasonable buffer.
	buf := make([]byte, 0, len(cs)*16)
	for i, c := range cs {
		if i > 0 {
			buf = append(buf, '|')
		}
		switch v := c.(type) {
		case Length:
			buf = append(buf, fmt.Sprintf("L%d", v.Value)...)
		case Percentage:
			buf = append(buf, fmt.Sprintf("P%d", v.Value)...)
		case Min:
			buf = append(buf, fmt.Sprintf("m%d", v.Value)...)
		case Max:
			buf = append(buf, fmt.Sprintf("M%d", v.Value)...)
		case Fill:
			buf = append(buf, fmt.Sprintf("F%d", v.Weight)...)
		case Ratio:
			buf = append(buf, fmt.Sprintf("R%d/%d", v.Num, v.Den)...)
		default:
			buf = append(buf, '?')
		}
	}
	return string(buf)
}
