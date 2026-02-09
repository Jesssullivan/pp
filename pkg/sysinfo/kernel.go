package sysinfo

import (
	"strings"
)

// siKernelVersion returns the kernel version string. Platform-specific
// retrieval is handled by siKernelVersionPlatform.
func siKernelVersion() string {
	raw := siKernelVersionPlatform()
	return siParseKernelVersion(raw)
}

// siParseKernelVersion cleans a raw kernel version string by trimming
// whitespace and stripping common prefixes like "Linux version ".
func siParseKernelVersion(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return ""
	}

	// /proc/version on Linux returns something like:
	// "Linux version 6.1.0-27-amd64 (debian-kernel@...) (gcc ...) ..."
	// We want just the version number.
	if strings.HasPrefix(s, "Linux version ") {
		s = strings.TrimPrefix(s, "Linux version ")
		// Take only the version token (first field).
		if idx := strings.IndexByte(s, ' '); idx >= 0 {
			s = s[:idx]
		}
	}

	return s
}
