package migrate

import (
	"fmt"
	"os"
	"path/filepath"
)

// mgEnsurePidCompat ensures the v2 PID file location is compatible with v1 daemon
// management scripts by creating a symlink from the v1 default location if different.
func mgEnsurePidCompat(v2PidPath string) error {
	if v2PidPath == "" {
		return nil
	}

	// Ensure parent directory exists
	dir := filepath.Dir(v2PidPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating PID directory %s: %w", dir, err)
	}

	// v1 default PID location
	v1PidPath := mgV1DefaultPidPath()
	if v1PidPath == v2PidPath {
		return nil
	}

	// If v1 path exists as a regular file, remove it (stale PID)
	if info, err := os.Lstat(v1PidPath); err == nil {
		if info.Mode()&os.ModeSymlink != 0 || info.Mode().IsRegular() {
			os.Remove(v1PidPath)
		}
	}

	// Create parent of v1 path if needed
	v1Dir := filepath.Dir(v1PidPath)
	if err := os.MkdirAll(v1Dir, 0o755); err != nil {
		return fmt.Errorf("creating v1 PID directory %s: %w", v1Dir, err)
	}

	// Create symlink from v1 location -> v2 location
	if err := os.Symlink(v2PidPath, v1PidPath); err != nil {
		return fmt.Errorf("creating PID compat symlink: %w", err)
	}

	return nil
}

// mgEnsureCacheCompat ensures the v2 cache directory structure is compatible with
// v1 expectations (waifu cache, banner cache subdirectories).
func mgEnsureCacheCompat(v2CacheDir string) error {
	if v2CacheDir == "" {
		return nil
	}

	subdirs := []string{
		"waifu",
		"banner",
		"sessions",
	}

	for _, sub := range subdirs {
		path := filepath.Join(v2CacheDir, sub)
		if err := os.MkdirAll(path, 0o755); err != nil {
			return fmt.Errorf("creating cache subdirectory %s: %w", path, err)
		}
	}

	return nil
}

// mgEnsureSocketCompat creates a symlink from the v1 socket path to the v2 socket
// path if they differ, allowing v1 scripts to continue connecting.
func mgEnsureSocketCompat(v2SocketPath string) error {
	if v2SocketPath == "" {
		return nil
	}

	v1SocketPath := mgV1DefaultSocketPath()
	if v1SocketPath == v2SocketPath {
		return nil
	}

	// Ensure parent directory of v1 socket exists
	v1Dir := filepath.Dir(v1SocketPath)
	if err := os.MkdirAll(v1Dir, 0o755); err != nil {
		return fmt.Errorf("creating v1 socket directory %s: %w", v1Dir, err)
	}

	// Remove existing v1 socket/symlink
	if _, err := os.Lstat(v1SocketPath); err == nil {
		os.Remove(v1SocketPath)
	}

	// Symlink v1 -> v2
	if err := os.Symlink(v2SocketPath, v1SocketPath); err != nil {
		return fmt.Errorf("creating socket compat symlink: %w", err)
	}

	return nil
}

// mgCleanupCompat removes all compatibility shims created during migration.
// Call this after the transition period when v1 scripts are no longer in use.
func mgCleanupCompat() error {
	var errs []error

	// Remove PID symlink
	v1Pid := mgV1DefaultPidPath()
	if info, err := os.Lstat(v1Pid); err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			if err := os.Remove(v1Pid); err != nil {
				errs = append(errs, fmt.Errorf("removing PID symlink %s: %w", v1Pid, err))
			}
		}
	}

	// Remove socket symlink
	v1Socket := mgV1DefaultSocketPath()
	if info, err := os.Lstat(v1Socket); err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			if err := os.Remove(v1Socket); err != nil {
				errs = append(errs, fmt.Errorf("removing socket symlink %s: %w", v1Socket, err))
			}
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("compat cleanup had %d errors: %v", len(errs), errs)
	}

	return nil
}

// mgV1DefaultPidPath returns the v1 default PID file location.
func mgV1DefaultPidPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "run", "prompt-pulse", "daemon.pid")
}

// mgV1DefaultSocketPath returns the v1 default Unix socket location.
func mgV1DefaultSocketPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "run", "prompt-pulse", "daemon.sock")
}
