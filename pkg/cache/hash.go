package cache

import (
	"crypto/sha256"
	"encoding/hex"
)

// hashKey returns the first 16 hex characters of the SHA-256 hash of key.
// This produces a deterministic, filesystem-safe identifier for any key
// regardless of length, special characters, or path separators.
func hashKey(key string) string {
	h := sha256.Sum256([]byte(key))
	return hex.EncodeToString(h[:8]) // 8 bytes = 16 hex chars
}
