// Package starship generates a single-line formatted string for use as a
// starship custom module. It reads cached collector data and renders a
// compact status line with ANSI colors.
package starship

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// ssMaxCacheAge is the maximum age of a cache file before it is considered
// stale and ignored. Collectors are expected to refresh more frequently.
const ssMaxCacheAge = 5 * time.Minute

// ssReadCachedData reads a JSON cache file for the given collector key from
// cacheDir. Returns nil if the file does not exist, cannot be parsed, or is
// older than ssMaxCacheAge.
func ssReadCachedData[T any](cacheDir, key string) (*T, error) {
	path := filepath.Join(cacheDir, key+".json")

	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	// Reject stale data.
	if time.Since(info.ModTime()) > ssMaxCacheAge {
		return nil, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var v T
	if err := json.Unmarshal(data, &v); err != nil {
		return nil, err
	}

	return &v, nil
}
