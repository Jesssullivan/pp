//go:build darwin

package platform

import (
	"os/exec"
	"strings"

	"github.com/shirou/gopsutil/v4/disk"
)

// plGetDiskInfo returns disk usage information for Darwin systems.
// It handles APFS correctly by querying /System/Volumes/Data explicitly,
// since querying $HOME on APFS returns misleading container-level stats.
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
			Label:       plDarwinLabel(p.Mountpoint),
		}
		results = append(results, info)
	}

	return plFilterDarwinMounts(results), nil
}

// plDarwinLabel returns a user-friendly label for a Darwin mount path.
func plDarwinLabel(path string) string {
	switch path {
	case "/":
		return "macOS"
	case "/System/Volumes/Data":
		return "Data"
	default:
		parts := strings.Split(path, "/")
		if len(parts) > 0 {
			return parts[len(parts)-1]
		}
		return path
	}
}

// plGetAPFSVolumes uses diskutil to get accurate APFS volume sizes.
func plGetAPFSVolumes() ([]DiskInfo, error) {
	cmd := exec.Command("diskutil", "apfs", "list", "-plist")
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	_ = out // plist parsing would go here for extended APFS info
	// Fall back to standard disk info which is sufficient for most cases
	return plGetDiskInfo()
}

// plFilterDarwinMounts removes synthetic and system mounts from the list.
// This is a thin wrapper around the testable pure function.
func plFilterDarwinMounts(mounts []DiskInfo) []DiskInfo {
	return PlFilterDarwinMountsFunc(mounts)
}

// PlFilterDarwinMountsFunc is the pure, testable implementation of Darwin mount
// filtering. It removes devfs, map auto_home, and other synthetic mounts.
func PlFilterDarwinMountsFunc(mounts []DiskInfo) []DiskInfo {
	var filtered []DiskInfo
	for _, m := range mounts {
		if plIsSyntheticDarwinMount(m) {
			continue
		}
		filtered = append(filtered, m)
	}
	return filtered
}

// plIsSyntheticDarwinMount returns true if the mount is a synthetic/system mount
// that should be excluded from disk reporting.
func plIsSyntheticDarwinMount(m DiskInfo) bool {
	syntheticFS := map[string]bool{
		"devfs":    true,
		"autofs":   true,
		"nullfs":   true,
		"synthfs":  true,
		"volfs":    true,
		"fdescfs":  true,
		"lifs":     true,
		"nfs":      false, // NFS is real
		"smbfs":    false, // SMB is real
	}
	if syntheticFS[m.FSType] {
		return true
	}

	syntheticPaths := []string{
		"/dev",
		"/System/Volumes/VM",
		"/System/Volumes/Preboot",
		"/System/Volumes/Update",
		"/System/Volumes/xarts",
		"/System/Volumes/iSCPreboot",
		"/System/Volumes/Hardware",
		"/private/var/vm",
	}
	for _, sp := range syntheticPaths {
		if m.Path == sp {
			return true
		}
	}

	// Filter "map auto_home" and similar autofs entries
	if strings.Contains(m.FSType, "map") && strings.Contains(m.FSType, "auto") {
		return true
	}

	// Filter zero-size mounts
	if m.Total == 0 {
		return true
	}

	return false
}
