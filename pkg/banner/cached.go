package banner

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// bnCacheTTL is the maximum age of a cached banner file before it is
// considered stale and re-rendered.
const bnCacheTTL = 30 * time.Second

// RenderCached renders the banner, using a disk cache to avoid redundant
// work. If a cached file exists for the given data+preset combination and
// is younger than 30 seconds, its contents are returned directly (the
// fast <1ms path). Otherwise the banner is rendered fresh, written to the
// cache atomically (temp file + rename), and the result is returned.
func RenderCached(cacheDir string, data BannerData, preset Preset) (string, error) {
	key := bnCacheKey(data, preset)
	path := filepath.Join(cacheDir, "banner-"+key+".cache")

	// Check for a fresh cache hit.
	if info, err := os.Stat(path); err == nil {
		age := time.Since(info.ModTime())
		if age < bnCacheTTL {
			content, err := os.ReadFile(path)
			if err == nil {
				return string(content), nil
			}
			// Fall through on read error.
		}
	}

	// Render fresh.
	result := Render(data, preset)

	// Write to cache atomically.
	if err := bnAtomicWriteCache(cacheDir, path, result); err != nil {
		// Cache write failure is non-fatal; return the rendered result.
		return result, nil
	}

	return result, nil
}

// bnCacheKey produces a deterministic cache key by hashing all widget data
// content and the preset name. Any change to widget content or preset
// produces a different key.
func bnCacheKey(data BannerData, preset Preset) string {
	h := sha256.New()
	h.Write([]byte(preset.Name))
	h.Write([]byte{0}) // separator
	fmt.Fprintf(h, "%d:%d", preset.Width, preset.Height)
	h.Write([]byte{0})
	for _, w := range data.Widgets {
		h.Write([]byte(w.ID))
		h.Write([]byte{0})
		h.Write([]byte(w.Title))
		h.Write([]byte{0})
		h.Write([]byte(w.Content))
		h.Write([]byte{0})
		fmt.Fprintf(h, "%d:%d", w.MinW, w.MinH)
		h.Write([]byte{0})
	}
	sum := h.Sum(nil)
	return hex.EncodeToString(sum[:12]) // 24 hex chars
}

// bnAtomicWriteCache writes content to path via a temporary file and rename,
// ensuring readers never see a partial file.
func bnAtomicWriteCache(dir, path, content string) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("banner cache: mkdir %s: %w", dir, err)
	}

	tmp, err := os.CreateTemp(dir, ".banner-tmp-*")
	if err != nil {
		return fmt.Errorf("banner cache: create temp: %w", err)
	}
	tmpName := tmp.Name()

	success := false
	defer func() {
		if !success {
			_ = os.Remove(tmpName)
		}
	}()

	if _, err := tmp.WriteString(content); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("banner cache: write temp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("banner cache: close temp: %w", err)
	}

	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("banner cache: rename: %w", err)
	}

	success = true
	return nil
}

// bnCacheKeyExported is an exported wrapper for testing. Tests in the same
// package can call bnCacheKey directly, but this provides a public entry
// point if needed from external test packages.
func bnCacheKeyExported(data BannerData, preset Preset) string {
	return bnCacheKey(data, preset)
}

// bnCleanStaleCacheFiles removes banner cache files older than maxAge from dir.
// This is not called automatically but can be invoked by callers that want to
// manage cache size.
func bnCleanStaleCacheFiles(dir string, maxAge time.Duration) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasPrefix(name, "banner-") || !strings.HasSuffix(name, ".cache") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		if time.Since(info.ModTime()) > maxAge {
			_ = os.Remove(filepath.Join(dir, name))
		}
	}
	return nil
}
