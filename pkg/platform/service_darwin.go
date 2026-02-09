//go:build darwin

package platform

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// plGenerateLaunchdPlist generates a launchd plist XML string for the daemon.
// This is a thin wrapper around the testable pure function.
func plGenerateLaunchdPlist(cfg ServiceConfig) string {
	return PlGenerateLaunchdPlistFunc(cfg)
}

// PlGenerateLaunchdPlistFunc is the pure, testable implementation of launchd
// plist generation.
func PlGenerateLaunchdPlistFunc(cfg ServiceConfig) string {
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

// plLaunchdPlistPath returns the path to the launchd plist file.
func plLaunchdPlistPath() string {
	return PlLaunchdPlistPathFunc()
}

// PlLaunchdPlistPathFunc is the pure, testable implementation.
func PlLaunchdPlistPathFunc() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = os.Getenv("HOME")
	}
	return filepath.Join(home, "Library", "LaunchAgents", "com.tinyland.prompt-pulse.plist")
}

// plInstallLaunchdService writes the plist and loads it via launchctl.
func plInstallLaunchdService(cfg ServiceConfig) error {
	plistPath := plLaunchdPlistPath()

	// Ensure directory exists
	dir := filepath.Dir(plistPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating LaunchAgents directory: %w", err)
	}

	// Write plist
	content := plGenerateLaunchdPlist(cfg)
	if err := os.WriteFile(plistPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("writing plist: %w", err)
	}

	// Unload first if already loaded (ignore errors)
	_ = exec.Command("launchctl", "unload", plistPath).Run()

	// Load the service
	if err := exec.Command("launchctl", "load", plistPath).Run(); err != nil {
		return fmt.Errorf("loading launchd service: %w", err)
	}

	return nil
}

// plUninstallLaunchdService unloads and removes the plist.
func plUninstallLaunchdService() error {
	plistPath := plLaunchdPlistPath()

	// Unload (ignore errors if not loaded)
	_ = exec.Command("launchctl", "unload", plistPath).Run()

	// Remove the plist file
	if err := os.Remove(plistPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing plist: %w", err)
	}

	return nil
}
