//go:build linux

package platform

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// plGenerateSystemdUnit generates a systemd user service unit file.
// This is a thin wrapper around the testable pure function.
func plGenerateSystemdUnit(cfg ServiceConfig) string {
	return PlGenerateSystemdUnitFunc(cfg)
}

// PlGenerateSystemdUnitFunc is the pure, testable implementation of systemd
// unit generation.
func PlGenerateSystemdUnitFunc(cfg ServiceConfig) string {
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

// plSystemdUnitPath returns the path to the systemd user service unit file.
func plSystemdUnitPath() string {
	return PlSystemdUnitPathFunc()
}

// PlSystemdUnitPathFunc is the pure, testable implementation.
func PlSystemdUnitPathFunc() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = os.Getenv("HOME")
	}
	return filepath.Join(home, ".config", "systemd", "user", "prompt-pulse.service")
}

// plInstallSystemdService writes the unit file and enables it.
func plInstallSystemdService(cfg ServiceConfig) error {
	unitPath := plSystemdUnitPath()

	// Ensure directory exists
	dir := filepath.Dir(unitPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating systemd user directory: %w", err)
	}

	// Write unit file
	content := plGenerateSystemdUnit(cfg)
	if err := os.WriteFile(unitPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("writing unit file: %w", err)
	}

	// Reload systemd user daemon
	if err := exec.Command("systemctl", "--user", "daemon-reload").Run(); err != nil {
		return fmt.Errorf("reloading systemd: %w", err)
	}

	// Enable and start
	if err := exec.Command("systemctl", "--user", "enable", "--now", "prompt-pulse.service").Run(); err != nil {
		return fmt.Errorf("enabling service: %w", err)
	}

	return nil
}

// plUninstallSystemdService stops, disables, and removes the service.
func plUninstallSystemdService() error {
	// Stop and disable (ignore errors if not running)
	_ = exec.Command("systemctl", "--user", "stop", "prompt-pulse.service").Run()
	_ = exec.Command("systemctl", "--user", "disable", "prompt-pulse.service").Run()

	unitPath := plSystemdUnitPath()
	if err := os.Remove(unitPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing unit file: %w", err)
	}

	// Reload after removal
	_ = exec.Command("systemctl", "--user", "daemon-reload").Run()

	return nil
}
