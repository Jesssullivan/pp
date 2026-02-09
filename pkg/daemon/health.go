package daemon

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// WriteHealthFile writes the health status as indented JSON to path.
// The write is atomic: content goes to a temporary file first, then is
// renamed into place to prevent partial reads.
func WriteHealthFile(path string, status *HealthStatus) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create health directory: %w", err)
	}

	data, err := json.MarshalIndent(status, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal health status: %w", err)
	}

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("write temp health file: %w", err)
	}

	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("rename health file: %w", err)
	}

	return nil
}

// ReadHealthFile reads and parses the health status JSON from path.
func ReadHealthFile(path string) (*HealthStatus, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read health file: %w", err)
	}

	var status HealthStatus
	if err := json.Unmarshal(data, &status); err != nil {
		return nil, fmt.Errorf("unmarshal health file: %w", err)
	}

	return &status, nil
}

// healthStatusToJSON serializes a HealthStatus to indented JSON string.
func healthStatusToJSON(status *HealthStatus) (string, error) {
	data, err := json.MarshalIndent(status, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal health status: %w", err)
	}
	return string(data), nil
}
