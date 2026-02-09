// Package config provides TOML-based configuration for prompt-pulse v2.
package config

import (
	"fmt"
	"time"
)

// Duration wraps time.Duration with TOML-friendly string parsing.
// Supports standard Go duration strings: "1s", "30s", "5m", "1h", "15m", etc.
type Duration struct {
	time.Duration
}

// UnmarshalText implements encoding.TextUnmarshaler for TOML parsing.
func (d *Duration) UnmarshalText(text []byte) error {
	s := string(text)
	if s == "" {
		d.Duration = 0
		return nil
	}
	parsed, err := time.ParseDuration(s)
	if err != nil {
		return fmt.Errorf("invalid duration %q: %w", s, err)
	}
	if parsed < 0 {
		return fmt.Errorf("negative duration %q not allowed", s)
	}
	d.Duration = parsed
	return nil
}

// MarshalText implements encoding.TextMarshaler for TOML serialization.
func (d Duration) MarshalText() ([]byte, error) {
	return []byte(d.Duration.String()), nil
}
