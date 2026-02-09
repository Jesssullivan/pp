package cache

import (
	"encoding/json"
	"fmt"
	"time"
)

// GetTyped deserializes a cached JSON value into the given type T.
// Returns the zero value of T and false if the key is missing, expired,
// or the stored data is not valid JSON for type T.
func GetTyped[T any](s *Store, key string) (T, bool) {
	data, ok := s.Get(key)
	if !ok {
		var zero T
		return zero, false
	}
	var v T
	if err := json.Unmarshal(data, &v); err != nil {
		var zero T
		return zero, false
	}
	return v, true
}

// PutTyped serializes value as JSON and stores it with the default TTL.
func PutTyped[T any](s *Store, key string, value T) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("cache: marshal typed value for %q: %w", key, err)
	}
	return s.Put(key, data)
}

// PutTypedWithTTL serializes value as JSON and stores it with a custom TTL.
func PutTypedWithTTL[T any](s *Store, key string, value T, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("cache: marshal typed value for %q: %w", key, err)
	}
	return s.PutWithTTL(key, data, ttl)
}
