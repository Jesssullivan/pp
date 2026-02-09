// Package platform provides platform-specific abstractions for Darwin (macOS)
// and Linux, handling OS-specific behavior for disk monitoring, service
// management, and container runtime detection.
package platform

import "runtime"

// Platform identifies the current OS platform.
type Platform string

const (
	// Darwin represents macOS.
	Darwin Platform = "darwin"
	// Linux represents Linux distributions.
	Linux Platform = "linux"
)

// Current returns the platform for the running OS.
func Current() Platform {
	return Platform(runtime.GOOS)
}

// DiskInfo represents a mounted filesystem with accurate usage statistics.
type DiskInfo struct {
	Path        string  // mount path
	FSType      string  // filesystem type (apfs, ext4, xfs, etc.)
	Total       uint64  // total bytes
	Used        uint64  // used bytes
	Free        uint64  // free bytes
	UsedPercent float64 // usage as a percentage 0-100
	Label       string  // user-friendly label
}

// ServiceConfig holds daemon service configuration for launchd or systemd.
type ServiceConfig struct {
	BinaryPath string // absolute path to the daemon binary
	Interval   string // polling interval (e.g. "30s", "1m")
	LogPath    string // path to log file
	ConfigPath string // path to config file
}

// ContainerRuntime describes a detected container runtime.
type ContainerRuntime struct {
	Name    string // "docker", "podman", "colima", "orbstack", "lima"
	Running bool   // whether the runtime is currently active
	Version string // version string
	Socket  string // path to the container socket
}
