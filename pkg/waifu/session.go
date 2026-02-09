// Package waifu provides session management, caching, image selection, and
// background prefetching for the waifu image display subsystem.
//
// Key design decisions that fix v1 bugs:
//   - Session IDs are PID-based (no timestamp), so the cache persists across
//     shell sessions sharing the same terminal.
//   - Content hashing uses SHA-256 of file bytes (not apparent file size).
//   - Prefetcher goroutines are tracked via sync.WaitGroup.
//   - A single rendering code path through the ImageRenderer interface.
package waifu

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

// SessionConfig configures the SessionManager.
type SessionConfig struct {
	// ImageDir is the directory containing waifu images.
	ImageDir string

	// CacheDir is the directory for rendered cache files.
	CacheDir string

	// MaxCacheSize is the max cache size in bytes. Default: 100MB.
	MaxCacheSize int64
}

// Session represents an active waifu image session tied to a process.
type Session struct {
	// ID is the stable session identifier, format: "ppulse-{PID}".
	ID string

	// ImagePath is the absolute path to the selected image file.
	ImagePath string

	// ContentHash is the first 16 hex chars of the SHA-256 of the image content.
	ContentHash string

	// CreatedAt is when the session was created.
	CreatedAt time.Time
}

// SessionManager manages waifu image sessions. Sessions are keyed by a
// PID-based identifier so that the same terminal process always gets the
// same cached image.
type SessionManager struct {
	mu       sync.RWMutex
	sessions map[string]*Session
	cfg      SessionConfig
}

// NewSessionManager creates a SessionManager with the given configuration.
func NewSessionManager(cfg SessionConfig) *SessionManager {
	if cfg.MaxCacheSize <= 0 {
		cfg.MaxCacheSize = 100 * 1024 * 1024 // 100 MB
	}
	return &SessionManager{
		sessions: make(map[string]*Session),
		cfg:      cfg,
	}
}

// GetOrCreate returns an existing session for the current PID, or creates a
// new one by selecting a random image from ImageDir and computing its content
// hash.
func (sm *SessionManager) GetOrCreate() (*Session, error) {
	id := fmt.Sprintf("ppulse-%d", os.Getpid())

	sm.mu.RLock()
	if s, ok := sm.sessions[id]; ok {
		sm.mu.RUnlock()
		return s, nil
	}
	sm.mu.RUnlock()

	// Select a random image.
	imgPath, err := PickRandom(sm.cfg.ImageDir)
	if err != nil {
		return nil, fmt.Errorf("pick random image: %w", err)
	}

	// Compute content hash.
	hash, err := contentHash(imgPath)
	if err != nil {
		return nil, fmt.Errorf("compute content hash: %w", err)
	}

	s := &Session{
		ID:          id,
		ImagePath:   imgPath,
		ContentHash: hash,
		CreatedAt:   time.Now(),
	}

	sm.mu.Lock()
	// Double-check: another goroutine may have created it while we were hashing.
	if existing, ok := sm.sessions[id]; ok {
		sm.mu.Unlock()
		return existing, nil
	}
	sm.sessions[id] = s
	sm.mu.Unlock()

	return s, nil
}

// Get looks up a session by ID. Returns the session and true if found,
// or nil and false otherwise.
func (sm *SessionManager) Get(id string) (*Session, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	s, ok := sm.sessions[id]
	return s, ok
}

// Close removes a session by ID and cleans up its resources.
func (sm *SessionManager) Close(id string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	delete(sm.sessions, id)
}

// CleanStale removes all sessions older than maxAge.
func (sm *SessionManager) CleanStale(maxAge time.Duration) {
	cutoff := time.Now().Add(-maxAge)

	sm.mu.Lock()
	defer sm.mu.Unlock()

	for id, s := range sm.sessions {
		if s.CreatedAt.Before(cutoff) {
			delete(sm.sessions, id)
		}
	}
}

// ActiveCount returns the number of currently active sessions.
func (sm *SessionManager) ActiveCount() int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return len(sm.sessions)
}

// contentHash reads the first 64KB of a file and returns the first 16 hex
// characters of its SHA-256 hash. Reading only the head of the file keeps
// hashing fast for large images while still providing sufficient uniqueness.
func contentHash(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	buf := make([]byte, 64*1024)
	n, err := f.Read(buf)
	if err != nil && err != io.EOF {
		return "", err
	}
	h.Write(buf[:n])

	return fmt.Sprintf("%x", h.Sum(nil))[:16], nil
}
