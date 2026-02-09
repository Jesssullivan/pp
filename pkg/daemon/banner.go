package daemon

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// BannerEntry holds a single pre-rendered banner for a specific terminal
// configuration. Inspired by powerlevel10k's instant prompt technique: the
// daemon pre-renders during idle time so shell startup can display instantly.
type BannerEntry struct {
	Rendered  string    `json:"rendered"`
	Width     int       `json:"width"`
	Height    int       `json:"height"`
	Protocol  string    `json:"protocol"`
	Timestamp time.Time `json:"timestamp"`
	Hash      string    `json:"hash"`
}

// bannerCacheFile is the on-disk representation: a map of cache keys to entries.
type bannerCacheFile struct {
	Entries map[string]*BannerEntry `json:"entries"`
}

// BannerCache manages pre-rendered banner entries on disk. Multiple entries
// can be stored (one per width/height/protocol combination). The cache file
// survives daemon restarts.
type BannerCache struct {
	path string
	mu   sync.Mutex
}

// NewBannerCache creates a BannerCache backed by the given file path.
func NewBannerCache(path string) *BannerCache {
	return &BannerCache{path: path}
}

// bannerKey returns the cache key for a given terminal configuration.
func bannerKey(width, height int, protocol string) string {
	return fmt.Sprintf("%dx%d/%s", width, height, protocol)
}

// Get retrieves a cached banner entry matching the given terminal dimensions
// and graphics protocol. Returns the entry and true if found, nil and false
// otherwise.
func (bc *BannerCache) Get(width, height int, protocol string) (*BannerEntry, bool) {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	cf, err := bc.load()
	if err != nil {
		return nil, false
	}

	key := bannerKey(width, height, protocol)
	entry, ok := cf.Entries[key]
	return entry, ok
}

// Put stores a pre-rendered banner entry in the cache. The entry is keyed
// by its Width, Height, and Protocol fields. A content hash is computed
// automatically if not already set.
func (bc *BannerCache) Put(entry *BannerEntry) error {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	if entry.Hash == "" {
		entry.Hash = computeHash(entry.Rendered)
	}

	cf, err := bc.load()
	if err != nil {
		// Start fresh if the file is missing or corrupt.
		cf = &bannerCacheFile{
			Entries: make(map[string]*BannerEntry),
		}
	}

	key := bannerKey(entry.Width, entry.Height, entry.Protocol)
	cf.Entries[key] = entry

	return bc.save(cf)
}

// Invalidate clears all entries from the cache by removing the file.
func (bc *BannerCache) Invalidate() error {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	if err := os.Remove(bc.path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("invalidate banner cache: %w", err)
	}
	return nil
}

// IsStale returns true if the cache file does not exist, has no entries, or
// the most recent entry is older than maxAge.
func (bc *BannerCache) IsStale(maxAge time.Duration) bool {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	cf, err := bc.load()
	if err != nil {
		return true
	}

	if len(cf.Entries) == 0 {
		return true
	}

	var newest time.Time
	for _, entry := range cf.Entries {
		if entry.Timestamp.After(newest) {
			newest = entry.Timestamp
		}
	}

	return time.Since(newest) > maxAge
}

// load reads and parses the cache file. Caller must hold bc.mu.
func (bc *BannerCache) load() (*bannerCacheFile, error) {
	data, err := os.ReadFile(bc.path)
	if err != nil {
		return nil, err
	}

	var cf bannerCacheFile
	if err := json.Unmarshal(data, &cf); err != nil {
		return nil, err
	}

	if cf.Entries == nil {
		cf.Entries = make(map[string]*BannerEntry)
	}

	return &cf, nil
}

// save writes the cache file atomically. Caller must hold bc.mu.
func (bc *BannerCache) save(cf *bannerCacheFile) error {
	dir := filepath.Dir(bc.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create banner cache directory: %w", err)
	}

	data, err := json.MarshalIndent(cf, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal banner cache: %w", err)
	}

	tmp := bc.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("write temp banner cache: %w", err)
	}

	if err := os.Rename(tmp, bc.path); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("rename banner cache: %w", err)
	}

	return nil
}

// computeHash returns a hex-encoded SHA-256 hash of the content.
func computeHash(content string) string {
	h := sha256.Sum256([]byte(content))
	return fmt.Sprintf("%x", h)
}

// bannerEntryToJSON serializes a BannerEntry to indented JSON string.
func bannerEntryToJSON(entry *BannerEntry) (string, error) {
	data, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal banner entry: %w", err)
	}
	return string(data), nil
}
