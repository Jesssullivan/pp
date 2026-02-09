//go:build linux

package sysinfo

import (
	"os"
)

// siKernelVersionPlatform returns the kernel version on Linux by reading
// /proc/version.
func siKernelVersionPlatform() string {
	data, err := os.ReadFile("/proc/version")
	if err != nil {
		return ""
	}
	return string(data)
}
