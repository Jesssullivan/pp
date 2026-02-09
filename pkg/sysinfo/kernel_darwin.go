//go:build darwin

package sysinfo

import (
	"golang.org/x/sys/unix"
)

// siKernelVersionPlatform returns the kernel version on macOS via sysctl.
func siKernelVersionPlatform() string {
	ver, err := unix.Sysctl("kern.osrelease")
	if err != nil {
		return ""
	}
	return ver
}
