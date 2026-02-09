package deploy

import (
	"fmt"
	"os"
	"path/filepath"
)

// Check represents a single deployment verification check.
type Check struct {
	// Name identifies the check.
	Name string

	// Run executes the check and returns (passed, message).
	Run func() (bool, string)

	// Required marks the check as mandatory; a failure here means the
	// overall verification fails.
	Required bool
}

// dpCheckBinary returns a check that verifies the prompt-pulse binary
// exists and is executable.
func dpCheckBinary(profile *HostProfile) Check {
	return Check{
		Name:     "binary",
		Required: true,
		Run: func() (bool, string) {
			path := profile.BinaryPath
			if path == "" {
				path = dpDefaultBinaryPath()
			}
			info, err := os.Stat(path)
			if err != nil {
				return false, fmt.Sprintf("binary not found: %s", path)
			}
			if info.IsDir() {
				return false, fmt.Sprintf("path is a directory: %s", path)
			}
			// Check executable bit (Unix).
			if info.Mode()&0o111 == 0 {
				return false, fmt.Sprintf("binary not executable: %s", path)
			}
			return true, fmt.Sprintf("binary ok: %s", path)
		},
	}
}

// dpCheckConfig returns a check that verifies the config file exists
// and is a regular file.
func dpCheckConfig(profile *HostProfile) Check {
	return Check{
		Name:     "config",
		Required: true,
		Run: func() (bool, string) {
			path := profile.ConfigPath
			if path == "" {
				path = dpDefaultConfigPath()
			}
			info, err := os.Stat(path)
			if err != nil {
				return false, fmt.Sprintf("config not found: %s", path)
			}
			if info.IsDir() {
				return false, fmt.Sprintf("config path is a directory: %s", path)
			}
			return true, fmt.Sprintf("config ok: %s", path)
		},
	}
}

// dpCheckDaemon returns a check that verifies the daemon socket or PID
// file exists, indicating the daemon is running or was recently active.
func dpCheckDaemon(profile *HostProfile) Check {
	return Check{
		Name:     "daemon",
		Required: false,
		Run: func() (bool, string) {
			sock := profile.SocketPath
			if sock == "" {
				sock = dpDefaultSocketPath()
			}
			if _, err := os.Stat(sock); err == nil {
				return true, fmt.Sprintf("daemon socket ok: %s", sock)
			}

			pid := profile.PIDFile
			if pid == "" {
				pid = dpDefaultPIDPath()
			}
			if _, err := os.Stat(pid); err == nil {
				return true, fmt.Sprintf("daemon PID file ok: %s", pid)
			}

			return false, "daemon not detected (no socket or PID file)"
		},
	}
}

// dpCheckCache returns a check that verifies the cache directory exists
// with the expected subdirectories.
func dpCheckCache(profile *HostProfile) Check {
	return Check{
		Name:     "cache",
		Required: true,
		Run: func() (bool, string) {
			dir := profile.CacheDir
			if dir == "" {
				dir = dpDefaultCacheDir()
			}
			info, err := os.Stat(dir)
			if err != nil {
				return false, fmt.Sprintf("cache dir not found: %s", dir)
			}
			if !info.IsDir() {
				return false, fmt.Sprintf("cache path is not a directory: %s", dir)
			}

			subdirs := []string{"waifu", "banner", "sessions"}
			var missing []string
			for _, sub := range subdirs {
				p := filepath.Join(dir, sub)
				if _, err := os.Stat(p); err != nil {
					missing = append(missing, sub)
				}
			}
			if len(missing) > 0 {
				return false, fmt.Sprintf("cache missing subdirs: %v", missing)
			}
			return true, fmt.Sprintf("cache ok: %s", dir)
		},
	}
}

// dpCheckShell returns a check that verifies shell integration for the
// given shell name. It checks that a shell-integration marker file exists.
func dpCheckShell(profile *HostProfile, shell string) Check {
	return Check{
		Name:     fmt.Sprintf("shell-%s", shell),
		Required: false,
		Run: func() (bool, string) {
			dir := profile.CacheDir
			if dir == "" {
				dir = dpDefaultCacheDir()
			}
			marker := filepath.Join(dir, "shells", shell+".sh")
			if _, err := os.Stat(marker); err != nil {
				return false, fmt.Sprintf("shell integration missing for %s: %s", shell, marker)
			}
			return true, fmt.Sprintf("shell %s ok", shell)
		},
	}
}

// dpCheckCollector returns a check that verifies a collector's data
// file exists in the cache directory.
func dpCheckCollector(profile *HostProfile, name string) Check {
	return Check{
		Name:     fmt.Sprintf("collector-%s", name),
		Required: false,
		Run: func() (bool, string) {
			dir := profile.CacheDir
			if dir == "" {
				dir = dpDefaultCacheDir()
			}
			dataFile := filepath.Join(dir, "collectors", name+".json")
			if _, err := os.Stat(dataFile); err != nil {
				return false, fmt.Sprintf("collector data missing: %s", dataFile)
			}
			return true, fmt.Sprintf("collector %s ok", name)
		},
	}
}

// dpCheckTheme returns a check that verifies the configured theme directory
// or file exists.
func dpCheckTheme(profile *HostProfile) Check {
	return Check{
		Name:     "theme",
		Required: false,
		Run: func() (bool, string) {
			dir := profile.CacheDir
			if dir == "" {
				dir = dpDefaultCacheDir()
			}
			themePath := filepath.Join(dir, "theme.json")
			if _, err := os.Stat(themePath); err != nil {
				return false, fmt.Sprintf("theme file missing: %s", themePath)
			}
			return true, "theme ok"
		},
	}
}

// dpCheckTerminal returns a check that verifies a TERM variable is set
// and non-empty, indicating a valid terminal environment.
func dpCheckTerminal() Check {
	return Check{
		Name:     "terminal",
		Required: false,
		Run: func() (bool, string) {
			term := os.Getenv("TERM")
			if term == "" {
				return false, "TERM environment variable not set"
			}
			return true, fmt.Sprintf("terminal ok: %s", term)
		},
	}
}

// dpCheckImage returns a check that verifies the image protocol can be
// detected. In a non-terminal environment this is expected to fail gracefully.
func dpCheckImage() Check {
	return Check{
		Name:     "image-protocol",
		Required: false,
		Run: func() (bool, string) {
			// Detection requires a real terminal; report available info.
			termProg := os.Getenv("TERM_PROGRAM")
			if termProg != "" {
				return true, fmt.Sprintf("terminal program: %s", termProg)
			}
			term := os.Getenv("TERM")
			if term != "" {
				return true, fmt.Sprintf("image detection available via TERM=%s", term)
			}
			return false, "no terminal detected for image protocol"
		},
	}
}

// dpCheckPermissions returns a check that verifies the config and cache
// directories have appropriate permissions (owner read/write, not world-writable).
func dpCheckPermissions(profile *HostProfile) Check {
	return Check{
		Name:     "permissions",
		Required: true,
		Run: func() (bool, string) {
			dirs := map[string]string{}

			configPath := profile.ConfigPath
			if configPath == "" {
				configPath = dpDefaultConfigPath()
			}
			dirs["config"] = filepath.Dir(configPath)

			cacheDir := profile.CacheDir
			if cacheDir == "" {
				cacheDir = dpDefaultCacheDir()
			}
			dirs["cache"] = cacheDir

			for label, dir := range dirs {
				info, err := os.Stat(dir)
				if err != nil {
					return false, fmt.Sprintf("%s dir not found: %s", label, dir)
				}
				mode := info.Mode().Perm()
				// Fail if world-writable.
				if mode&0o002 != 0 {
					return false, fmt.Sprintf("%s dir is world-writable: %s (mode %o)", label, dir, mode)
				}
			}
			return true, "permissions ok"
		},
	}
}

// dpDefaultBinaryPath returns the conventional binary location.
func dpDefaultBinaryPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "bin", "prompt-pulse")
}

// dpDefaultConfigPath returns the conventional config file location.
func dpDefaultConfigPath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		home, _ := os.UserHomeDir()
		dir = filepath.Join(home, ".config")
	}
	return filepath.Join(dir, "prompt-pulse", "config.toml")
}

// dpDefaultCacheDir returns the conventional cache directory.
func dpDefaultCacheDir() string {
	dir, err := os.UserCacheDir()
	if err != nil {
		home, _ := os.UserHomeDir()
		dir = filepath.Join(home, ".cache")
	}
	return filepath.Join(dir, "prompt-pulse")
}

// dpDefaultSocketPath returns the conventional daemon socket location.
func dpDefaultSocketPath() string {
	return filepath.Join(os.TempDir(), "prompt-pulse.sock")
}

// dpDefaultPIDPath returns the conventional PID file location.
func dpDefaultPIDPath() string {
	return filepath.Join(os.TempDir(), "prompt-pulse.pid")
}
