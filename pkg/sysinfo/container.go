package sysinfo

import (
	"os"
	"runtime"
	"strings"
)

// siDetectContainer checks whether the current process is running inside a
// container. It returns true plus the container type string ("docker",
// "podman", "lxc") or false with an empty string.
func siDetectContainer() (bool, string) {
	// Podman sets CONTAINER=podman in its default environment.
	if v := os.Getenv("CONTAINER"); v != "" {
		return true, strings.ToLower(v)
	}

	// Docker creates /.dockerenv as a sentinel file.
	if siFileExists("/.dockerenv") {
		return true, "docker"
	}

	// Podman creates /run/.containerenv.
	if siFileExists("/run/.containerenv") {
		return true, "podman"
	}

	// On Linux, inspect /proc/1/cgroup for container signatures.
	if runtime.GOOS == "linux" {
		if ct := siDetectContainerFromCgroup(); ct != "" {
			return true, ct
		}
	}

	return false, ""
}

// siDetectContainerFromCgroup reads /proc/1/cgroup and looks for container
// runtime signatures. Returns the container type or empty string.
func siDetectContainerFromCgroup() string {
	data, err := os.ReadFile("/proc/1/cgroup")
	if err != nil {
		return ""
	}
	return siParseCgroup(string(data))
}

// siParseCgroup inspects cgroup content for container runtime signatures.
// This is a pure function suitable for unit testing.
func siParseCgroup(content string) string {
	lower := strings.ToLower(content)
	if strings.Contains(lower, "docker") || strings.Contains(lower, "containerd") {
		return "docker"
	}
	if strings.Contains(lower, "lxc") {
		return "lxc"
	}
	// Podman uses cgroup paths with libpod.
	if strings.Contains(lower, "libpod") {
		return "podman"
	}
	return ""
}

// siFileExists reports whether the given path exists and is not a directory.
func siFileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}
