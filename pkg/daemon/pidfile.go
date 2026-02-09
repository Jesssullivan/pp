package daemon

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

// AcquirePID creates a PID file at path with the current process PID.
// It fails if another live process already holds the lock. If the existing
// PID file points to a dead process, it is removed and re-acquired.
//
// The write is atomic: content is written to a temporary file in the same
// directory, then renamed into place.
func AcquirePID(path string) error {
	// Ensure the directory exists.
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create PID directory: %w", err)
	}

	// Check for existing PID file.
	existingPID, err := ReadPID(path)
	if err == nil {
		// PID file exists and is readable. Check if the process is alive.
		if IsProcessAlive(existingPID) {
			return fmt.Errorf("daemon already running (PID %d)", existingPID)
		}
		// Stale PID file -- remove it.
		os.Remove(path)
	}

	// Atomic write: write to temp file, then rename.
	pid := os.Getpid()
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(strconv.Itoa(pid)), 0o644); err != nil {
		return fmt.Errorf("write temp PID file: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("rename PID file: %w", err)
	}

	return nil
}

// ReleasePID removes the PID file at the given path.
func ReleasePID(path string) error {
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove PID file: %w", err)
	}
	return nil
}

// ReadPID reads and parses the PID from the given file.
func ReadPID(path string) (int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, fmt.Errorf("read PID file: %w", err)
	}

	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0, fmt.Errorf("parse PID file: %w", err)
	}

	return pid, nil
}

// IsProcessAlive checks whether a process with the given PID exists by
// sending signal 0. On Unix, this returns nil if the process exists and
// the caller has permission to signal it, or ESRCH if it does not exist.
func IsProcessAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	err = process.Signal(syscall.Signal(0))
	return err == nil
}
