//go:build linux

package platform

import (
	"os"
	"strconv"
	"strings"

	"github.com/shirou/gopsutil/v4/disk"
)

// plGetDiskInfo returns disk usage information for Linux systems.
func plGetDiskInfo() ([]DiskInfo, error) {
	partitions, err := disk.Partitions(false)
	if err != nil {
		return nil, err
	}

	var results []DiskInfo
	seen := make(map[string]bool)

	for _, p := range partitions {
		if seen[p.Mountpoint] {
			continue
		}
		seen[p.Mountpoint] = true

		usage, err := disk.Usage(p.Mountpoint)
		if err != nil {
			continue
		}

		info := DiskInfo{
			Path:        p.Mountpoint,
			FSType:      p.Fstype,
			Total:       usage.Total,
			Used:        usage.Used,
			Free:        usage.Free,
			UsedPercent: usage.UsedPercent,
			Label:       plLinuxLabel(p.Mountpoint),
		}
		results = append(results, info)
	}

	return plFilterLinuxMounts(results), nil
}

// plLinuxLabel returns a user-friendly label for a Linux mount path.
func plLinuxLabel(path string) string {
	switch path {
	case "/":
		return "root"
	case "/home":
		return "home"
	case "/boot":
		return "boot"
	case "/boot/efi":
		return "efi"
	default:
		parts := strings.Split(path, "/")
		if len(parts) > 0 {
			return parts[len(parts)-1]
		}
		return path
	}
}

// plFilterLinuxMounts removes virtual/system mounts from the list.
// This is a thin wrapper around the testable pure function.
func plFilterLinuxMounts(mounts []DiskInfo) []DiskInfo {
	return PlFilterLinuxMountsFunc(mounts)
}

// PlFilterLinuxMountsFunc is the pure, testable implementation of Linux mount
// filtering. It removes tmpfs, devtmpfs, proc, sys, cgroup, and overlay mounts
// (unless explicitly monitored).
func PlFilterLinuxMountsFunc(mounts []DiskInfo) []DiskInfo {
	var filtered []DiskInfo
	for _, m := range mounts {
		if plIsVirtualLinuxMount(m) {
			continue
		}
		filtered = append(filtered, m)
	}
	return filtered
}

// plIsVirtualLinuxMount returns true if the mount is a virtual/pseudo filesystem.
func plIsVirtualLinuxMount(m DiskInfo) bool {
	virtualFS := map[string]bool{
		"tmpfs":     true,
		"devtmpfs":  true,
		"proc":      true,
		"sysfs":     true,
		"cgroup":    true,
		"cgroup2":   true,
		"securityfs": true,
		"debugfs":   true,
		"tracefs":   true,
		"configfs":  true,
		"fusectl":   true,
		"hugetlbfs": true,
		"mqueue":    true,
		"pstore":    true,
		"binfmt_misc": true,
		"devpts":    true,
		"efivarfs":  true,
		"ramfs":     true,
		"rpc_pipefs": true,
		"nfsd":      true,
		"overlay":   true,
	}
	if virtualFS[m.FSType] {
		return true
	}

	virtualPaths := []string{
		"/proc",
		"/sys",
		"/dev",
		"/dev/shm",
		"/dev/pts",
		"/run",
		"/run/lock",
		"/run/user",
	}
	for _, vp := range virtualPaths {
		if m.Path == vp || strings.HasPrefix(m.Path, vp+"/") {
			return true
		}
	}

	// Filter zero-size mounts
	if m.Total == 0 {
		return true
	}

	return false
}

// plGetCgroupDiskLimits reads cgroup v2 disk limits if running in a container.
// Returns the limit in bytes, or 0 if not in a cgroup or no limit is set.
func plGetCgroupDiskLimits() (uint64, error) {
	data, err := os.ReadFile("/sys/fs/cgroup/io.max")
	if err != nil {
		// Not in a cgroup v2 environment or no io limits
		return 0, nil
	}

	content := strings.TrimSpace(string(data))
	if content == "" || content == "max" {
		return 0, nil
	}

	// Parse first line: "major:minor rbps=N wbps=N riops=N wiops=N"
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		for _, f := range fields {
			if strings.HasPrefix(f, "rbps=") {
				val := strings.TrimPrefix(f, "rbps=")
				if val == "max" {
					continue
				}
				limit, err := strconv.ParseUint(val, 10, 64)
				if err != nil {
					continue
				}
				return limit, nil
			}
		}
	}

	return 0, nil
}
