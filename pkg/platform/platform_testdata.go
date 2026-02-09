package platform

// This file contains pure functions that are testable cross-platform.
// Platform-specific files (disk_darwin.go, disk_linux.go, etc.) are thin
// wrappers around these functions. Tests use these directly with test data
// rather than reading the OS, so they compile on any platform.

import (
	"fmt"
	"strings"
)

// --- Darwin mount filtering (cross-platform testable) ---

// PlTestFilterDarwinMounts filters Darwin mounts using the same logic as the
// build-tagged plFilterDarwinMounts. Takes test data as input.
func PlTestFilterDarwinMounts(mounts []DiskInfo) []DiskInfo {
	var filtered []DiskInfo
	for _, m := range mounts {
		if plTestIsSyntheticDarwinMount(m) {
			continue
		}
		filtered = append(filtered, m)
	}
	return filtered
}

// plTestIsSyntheticDarwinMount is the cross-platform version of
// plIsSyntheticDarwinMount.
func plTestIsSyntheticDarwinMount(m DiskInfo) bool {
	syntheticFS := map[string]bool{
		"devfs":   true,
		"autofs":  true,
		"nullfs":  true,
		"synthfs": true,
		"volfs":   true,
		"fdescfs": true,
		"lifs":    true,
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

	if strings.Contains(m.FSType, "map") && strings.Contains(m.FSType, "auto") {
		return true
	}

	if m.Total == 0 {
		return true
	}

	return false
}

// --- Linux mount filtering (cross-platform testable) ---

// PlTestFilterLinuxMounts filters Linux mounts using the same logic as the
// build-tagged plFilterLinuxMounts. Takes test data as input.
func PlTestFilterLinuxMounts(mounts []DiskInfo) []DiskInfo {
	var filtered []DiskInfo
	for _, m := range mounts {
		if plTestIsVirtualLinuxMount(m) {
			continue
		}
		filtered = append(filtered, m)
	}
	return filtered
}

// plTestIsVirtualLinuxMount is the cross-platform version of
// plIsVirtualLinuxMount.
func plTestIsVirtualLinuxMount(m DiskInfo) bool {
	virtualFS := map[string]bool{
		"tmpfs":      true,
		"devtmpfs":   true,
		"proc":       true,
		"sysfs":      true,
		"cgroup":     true,
		"cgroup2":    true,
		"securityfs": true,
		"debugfs":    true,
		"tracefs":    true,
		"configfs":   true,
		"fusectl":    true,
		"hugetlbfs":  true,
		"mqueue":     true,
		"pstore":     true,
		"binfmt_misc": true,
		"devpts":     true,
		"efivarfs":   true,
		"ramfs":      true,
		"rpc_pipefs": true,
		"nfsd":       true,
		"overlay":    true,
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

	if m.Total == 0 {
		return true
	}

	return false
}

// --- Launchd plist generation (cross-platform testable) ---

// PlTestGenerateLaunchdPlist generates a launchd plist from ServiceConfig.
// This is the cross-platform testable version.
func PlTestGenerateLaunchdPlist(cfg ServiceConfig) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Label</key>
	<string>com.tinyland.prompt-pulse</string>
	<key>ProgramArguments</key>
	<array>
		<string>%s</string>
		<string>daemon</string>
		<string>--interval</string>
		<string>%s</string>
		<string>--config</string>
		<string>%s</string>
	</array>
	<key>RunAtLoad</key>
	<true/>
	<key>KeepAlive</key>
	<true/>
	<key>StandardOutPath</key>
	<string>%s</string>
	<key>StandardErrorPath</key>
	<string>%s</string>
	<key>EnvironmentVariables</key>
	<dict>
		<key>PATH</key>
		<string>/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin</string>
	</dict>
</dict>
</plist>
`, cfg.BinaryPath, cfg.Interval, cfg.ConfigPath, cfg.LogPath, cfg.LogPath)
}

// --- Systemd unit generation (cross-platform testable) ---

// PlTestGenerateSystemdUnit generates a systemd unit from ServiceConfig.
// This is the cross-platform testable version.
func PlTestGenerateSystemdUnit(cfg ServiceConfig) string {
	return fmt.Sprintf(`[Unit]
Description=prompt-pulse system metrics daemon
After=default.target

[Service]
Type=simple
ExecStart=%s daemon --interval %s --config %s
Restart=on-failure
RestartSec=5
StandardOutput=append:%s
StandardError=append:%s
Environment=PATH=/usr/local/bin:/usr/bin:/bin

[Install]
WantedBy=default.target
`, cfg.BinaryPath, cfg.Interval, cfg.ConfigPath, cfg.LogPath, cfg.LogPath)
}

// --- Path helpers (cross-platform testable) ---

// PlTestLaunchdPlistPath returns the expected launchd plist path for a given
// home directory.
func PlTestLaunchdPlistPath(home string) string {
	return home + "/Library/LaunchAgents/com.tinyland.prompt-pulse.plist"
}

// PlTestSystemdUnitPath returns the expected systemd unit path for a given
// home directory.
func PlTestSystemdUnitPath(home string) string {
	return home + "/.config/systemd/user/prompt-pulse.service"
}
