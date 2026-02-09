// Package migrate provides v1-to-v2 configuration migration for prompt-pulse.
//
// The migration pipeline is: detect version -> backup -> parse v1 -> transform -> write v2 -> verify.
// All operations are safe: backups are created before any changes, and atomic file writes
// prevent partial updates.
package migrate

import (
	"fmt"
	"os"
	"strings"

	"github.com/BurntSushi/toml"

	"gitlab.com/tinyland/lab/prompt-pulse/pkg/config"
)

// MigrationResult holds the outcome of a v1-to-v2 migration.
type MigrationResult struct {
	// Success indicates whether the migration completed without errors.
	Success bool

	// Warnings contains non-fatal issues encountered during migration.
	Warnings []string

	// BackupPath is the path to the pre-migration backup file.
	BackupPath string

	// Changes lists all configuration field changes made during migration.
	Changes []ConfigChange
}

// ConfigChange describes a single field change during migration.
type ConfigChange struct {
	// Field is the dotted path of the configuration field (e.g. "collectors.tailscale.enabled").
	Field string

	// OldValue is the string representation of the v1 value.
	OldValue string

	// NewValue is the string representation of the v2 value.
	NewValue string

	// Action is one of "added", "removed", or "changed".
	Action string
}

// Migrate performs a full v1-to-v2 configuration migration.
// It detects the config version, creates a backup, parses the v1 config,
// transforms it to v2 format, and writes the result to v2ConfigPath.
func Migrate(v1ConfigPath, v2ConfigPath string) (*MigrationResult, error) {
	result := &MigrationResult{}

	// Detect version
	version, err := DetectVersion(v1ConfigPath)
	if err != nil {
		return nil, fmt.Errorf("version detection failed: %w", err)
	}
	if version == 2 {
		result.Success = true
		result.Warnings = append(result.Warnings, "config is already v2 format, no migration needed")
		return result, nil
	}

	// Backup the original
	backupPath, err := mgBackup(v1ConfigPath)
	if err != nil {
		return nil, fmt.Errorf("backup failed: %w", err)
	}
	result.BackupPath = backupPath

	// Parse v1 config
	v1, err := mgParseV1(v1ConfigPath)
	if err != nil {
		return nil, fmt.Errorf("v1 parsing failed: %w", err)
	}

	// Transform to v2
	v2Config, changes := mgTransformConfig(v1)
	result.Changes = changes

	// Write the v2 config atomically
	if err := mgWriteConfig(v2ConfigPath, v2Config); err != nil {
		return nil, fmt.Errorf("writing v2 config failed: %w", err)
	}

	result.Success = true
	return result, nil
}

// DetectVersion determines whether a config file is v1 (flat TOML) or v2 (nested TOML).
// Returns 1 for v1 format, 2 for v2 format.
func DetectVersion(configPath string) (int, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return 0, fmt.Errorf("reading config: %w", err)
	}

	content := string(data)
	if strings.TrimSpace(content) == "" {
		return 0, fmt.Errorf("config file is empty")
	}

	// v2 uses nested TOML sections like [general], [collectors.tailscale], [image], etc.
	// v1 uses flat keys like waifu_path, daemon_socket, tailscale_enabled, etc.
	v2Markers := []string{
		"[general]",
		"[collectors",
		"[image]",
		"[theme]",
		"[shell]",
		"[banner]",
		"[layout]",
	}

	lowerContent := strings.ToLower(content)
	for _, marker := range v2Markers {
		if strings.Contains(lowerContent, marker) {
			return 2, nil
		}
	}

	// Check for v1-style flat keys
	v1Markers := []string{
		"waifu_path",
		"waifu_enabled",
		"daemon_socket",
		"daemon_pid_file",
		"cache_dir",
		"cache_ttl",
		"banner_width",
		"banner_show_waifu",
		"tailscale_enabled",
		"k8s_enabled",
		"claude_enabled",
		"billing_enabled",
		"starship_enabled",
		"starship_format",
		"theme",
	}

	for _, marker := range v1Markers {
		if strings.Contains(lowerContent, marker) {
			return 1, nil
		}
	}

	// If we can successfully decode it as v2, treat it as v2
	var testCfg config.Config
	if _, err := toml.Decode(content, &testCfg); err == nil {
		return 2, nil
	}

	return 0, fmt.Errorf("unable to determine config version")
}

// NeedsMigration checks whether the config at the given path requires migration.
// Returns true if the config is v1 format, false if v2 or not present.
func NeedsMigration(configPath string) (bool, error) {
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return false, nil
	}

	version, err := DetectVersion(configPath)
	if err != nil {
		return false, err
	}
	return version == 1, nil
}

// mgWriteConfig writes a v2 Config to a file using atomic write (temp + rename).
func mgWriteConfig(path string, cfg *config.Config) error {
	tmpFile, err := os.CreateTemp(mgDir(path), ".prompt-pulse-migrate-*.toml")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	tmpPath := tmpFile.Name()

	encoder := toml.NewEncoder(tmpFile)
	if err := encoder.Encode(cfg); err != nil {
		tmpFile.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("encoding config: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("closing temp file: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("atomic rename: %w", err)
	}

	return nil
}
