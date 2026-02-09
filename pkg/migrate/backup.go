package migrate

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// BackupInfo describes an existing backup file.
type BackupInfo struct {
	// Path is the absolute path to the backup file.
	Path string

	// Timestamp is when the backup was created.
	Timestamp time.Time

	// Version is the config version that was backed up (1 or 2).
	Version int
}

// mgBackup creates a timestamped backup of the given config file.
// The backup is named config.toml.v1.20060102-150405.bak.
// Returns the path to the backup file.
func mgBackup(configPath string) (string, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return "", fmt.Errorf("reading config for backup: %w", err)
	}

	version, _ := DetectVersion(configPath)
	if version == 0 {
		version = 1
	}

	ts := time.Now().Format("20060102-150405")
	backupName := fmt.Sprintf("%s.v%d.%s.bak", filepath.Base(configPath), version, ts)
	backupPath := filepath.Join(filepath.Dir(configPath), backupName)

	// Atomic write: temp file + rename
	tmpFile, err := os.CreateTemp(filepath.Dir(configPath), ".backup-*.tmp")
	if err != nil {
		return "", fmt.Errorf("creating temp file for backup: %w", err)
	}
	tmpPath := tmpFile.Name()

	if _, err := tmpFile.Write(data); err != nil {
		tmpFile.Close()
		os.Remove(tmpPath)
		return "", fmt.Errorf("writing backup data: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		os.Remove(tmpPath)
		return "", fmt.Errorf("closing backup temp file: %w", err)
	}

	if err := os.Rename(tmpPath, backupPath); err != nil {
		os.Remove(tmpPath)
		return "", fmt.Errorf("atomic rename for backup: %w", err)
	}

	return backupPath, nil
}

// backupPattern matches backup filenames like config.toml.v1.20060102-150405.bak
var backupPattern = regexp.MustCompile(`^(.+)\.v(\d+)\.(\d{8}-\d{6})\.bak$`)

// mgListBackups returns all backup files in the given directory, sorted by timestamp descending.
func mgListBackups(dir string) ([]BackupInfo, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("reading backup directory: %w", err)
	}

	var backups []BackupInfo
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		matches := backupPattern.FindStringSubmatch(entry.Name())
		if matches == nil {
			continue
		}

		ts, err := time.Parse("20060102-150405", matches[3])
		if err != nil {
			continue
		}

		version := 1
		if matches[2] == "2" {
			version = 2
		}

		backups = append(backups, BackupInfo{
			Path:      filepath.Join(dir, entry.Name()),
			Timestamp: ts,
			Version:   version,
		})
	}

	// Sort by timestamp descending (newest first)
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].Timestamp.After(backups[j].Timestamp)
	})

	return backups, nil
}

// mgRestore copies a backup file back to the config path using atomic write.
func mgRestore(backupPath, configPath string) error {
	data, err := os.ReadFile(backupPath)
	if err != nil {
		return fmt.Errorf("reading backup: %w", err)
	}

	tmpFile, err := os.CreateTemp(mgDir(configPath), ".restore-*.tmp")
	if err != nil {
		return fmt.Errorf("creating temp file for restore: %w", err)
	}
	tmpPath := tmpFile.Name()

	if _, err := tmpFile.Write(data); err != nil {
		tmpFile.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("writing restore data: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("closing restore temp file: %w", err)
	}

	if err := os.Rename(tmpPath, configPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("atomic rename for restore: %w", err)
	}

	return nil
}

// mgDir returns the directory portion of a path. Helper to avoid repeating filepath.Dir.
func mgDir(path string) string {
	d := filepath.Dir(path)
	if d == "" {
		return "."
	}
	return d
}

// mgIsBackupFile checks whether a filename looks like a backup file.
func mgIsBackupFile(name string) bool {
	return strings.HasSuffix(name, ".bak") && backupPattern.MatchString(name)
}
