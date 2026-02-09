package platform

import (
	"runtime"
	"strings"
	"testing"
)

// --- Test 1: Current() returns correct platform ---

func TestCurrentReturnsPlatform(t *testing.T) {
	got := Current()
	want := Platform(runtime.GOOS)
	if got != want {
		t.Errorf("Current() = %q, want %q", got, want)
	}
	// On this machine it must be one of the known platforms
	if got != Darwin && got != Linux {
		t.Errorf("Current() = %q, want darwin or linux", got)
	}
}

// --- Test 2: PlTestFilterDarwinMounts removes devfs ---

func TestFilterDarwinMountsRemovesDevfs(t *testing.T) {
	mounts := []DiskInfo{
		{Path: "/dev", FSType: "devfs", Total: 100},
		{Path: "/", FSType: "apfs", Total: 500_000_000_000},
	}
	filtered := PlTestFilterDarwinMounts(mounts)
	for _, m := range filtered {
		if m.FSType == "devfs" {
			t.Error("devfs mount should have been filtered out")
		}
	}
	if len(filtered) != 1 {
		t.Errorf("expected 1 mount after filtering, got %d", len(filtered))
	}
}

// --- Test 3: PlTestFilterDarwinMounts removes map auto_home ---

func TestFilterDarwinMountsRemovesAutoHome(t *testing.T) {
	mounts := []DiskInfo{
		{Path: "/home", FSType: "map auto_home", Total: 100},
		{Path: "/System/Volumes/Data", FSType: "apfs", Total: 500_000_000_000},
	}
	filtered := PlTestFilterDarwinMounts(mounts)
	for _, m := range filtered {
		if strings.Contains(m.FSType, "auto_home") {
			t.Error("map auto_home mount should have been filtered out")
		}
	}
	if len(filtered) != 1 {
		t.Errorf("expected 1 mount after filtering, got %d", len(filtered))
	}
}

// --- Test 4: PlTestFilterLinuxMounts removes tmpfs ---

func TestFilterLinuxMountsRemovesTmpfs(t *testing.T) {
	mounts := []DiskInfo{
		{Path: "/tmp", FSType: "tmpfs", Total: 4_000_000_000},
		{Path: "/", FSType: "ext4", Total: 500_000_000_000},
	}
	filtered := PlTestFilterLinuxMounts(mounts)
	for _, m := range filtered {
		if m.FSType == "tmpfs" {
			t.Error("tmpfs mount should have been filtered out")
		}
	}
	if len(filtered) != 1 {
		t.Errorf("expected 1 mount after filtering, got %d", len(filtered))
	}
}

// --- Test 5: PlTestFilterLinuxMounts removes proc ---

func TestFilterLinuxMountsRemovesProc(t *testing.T) {
	mounts := []DiskInfo{
		{Path: "/proc", FSType: "proc", Total: 1},
		{Path: "/proc/bus/usb", FSType: "usbfs", Total: 1},
		{Path: "/", FSType: "xfs", Total: 500_000_000_000},
	}
	filtered := PlTestFilterLinuxMounts(mounts)
	for _, m := range filtered {
		if strings.HasPrefix(m.Path, "/proc") {
			t.Errorf("proc mount %q should have been filtered out", m.Path)
		}
	}
}

// --- Test 6: PlTestFilterLinuxMounts removes sys ---

func TestFilterLinuxMountsRemovesSys(t *testing.T) {
	mounts := []DiskInfo{
		{Path: "/sys", FSType: "sysfs", Total: 1},
		{Path: "/sys/kernel/debug", FSType: "debugfs", Total: 1},
		{Path: "/home", FSType: "ext4", Total: 1_000_000_000_000},
	}
	filtered := PlTestFilterLinuxMounts(mounts)
	for _, m := range filtered {
		if strings.HasPrefix(m.Path, "/sys") {
			t.Errorf("sysfs mount %q should have been filtered out", m.Path)
		}
	}
}

// --- Test 7: PlTestGenerateLaunchdPlist produces valid XML ---

func TestGenerateLaunchdPlistValidXML(t *testing.T) {
	cfg := ServiceConfig{
		BinaryPath: "/usr/local/bin/prompt-pulse",
		Interval:   "30s",
		LogPath:    "/tmp/pp.log",
		ConfigPath: "/etc/prompt-pulse.toml",
	}
	plist := PlTestGenerateLaunchdPlist(cfg)
	if !strings.HasPrefix(plist, "<?xml version=") {
		t.Error("plist should start with XML declaration")
	}
	if !strings.Contains(plist, "<plist version=") {
		t.Error("plist should contain <plist> element")
	}
	if !strings.Contains(plist, "</plist>") {
		t.Error("plist should contain closing </plist>")
	}
}

// --- Test 8: PlTestGenerateLaunchdPlist contains binary path ---

func TestGenerateLaunchdPlistContainsBinaryPath(t *testing.T) {
	cfg := ServiceConfig{
		BinaryPath: "/opt/bin/prompt-pulse",
		Interval:   "1m",
		LogPath:    "/var/log/pp.log",
		ConfigPath: "/etc/pp.toml",
	}
	plist := PlTestGenerateLaunchdPlist(cfg)
	if !strings.Contains(plist, "/opt/bin/prompt-pulse") {
		t.Error("plist should contain the binary path")
	}
}

// --- Test 9: PlTestGenerateLaunchdPlist contains interval ---

func TestGenerateLaunchdPlistContainsInterval(t *testing.T) {
	cfg := ServiceConfig{
		BinaryPath: "/usr/local/bin/prompt-pulse",
		Interval:   "45s",
		LogPath:    "/tmp/pp.log",
		ConfigPath: "/etc/pp.toml",
	}
	plist := PlTestGenerateLaunchdPlist(cfg)
	if !strings.Contains(plist, "45s") {
		t.Error("plist should contain the interval value")
	}
	if !strings.Contains(plist, "--interval") {
		t.Error("plist should contain --interval argument")
	}
}

// --- Test 10: PlTestGenerateSystemdUnit produces valid unit structure ---

func TestGenerateSystemdUnitValidStructure(t *testing.T) {
	cfg := ServiceConfig{
		BinaryPath: "/usr/local/bin/prompt-pulse",
		Interval:   "30s",
		LogPath:    "/var/log/pp.log",
		ConfigPath: "/etc/pp.toml",
	}
	unit := PlTestGenerateSystemdUnit(cfg)
	if !strings.Contains(unit, "[Unit]") {
		t.Error("unit should contain [Unit] section")
	}
	if !strings.Contains(unit, "[Service]") {
		t.Error("unit should contain [Service] section")
	}
	if !strings.Contains(unit, "[Install]") {
		t.Error("unit should contain [Install] section")
	}
}

// --- Test 11: PlTestGenerateSystemdUnit contains ExecStart ---

func TestGenerateSystemdUnitContainsExecStart(t *testing.T) {
	cfg := ServiceConfig{
		BinaryPath: "/opt/prompt-pulse",
		Interval:   "60s",
		LogPath:    "/tmp/pp.log",
		ConfigPath: "/etc/pp.toml",
	}
	unit := PlTestGenerateSystemdUnit(cfg)
	if !strings.Contains(unit, "ExecStart=/opt/prompt-pulse") {
		t.Error("unit should contain ExecStart with binary path")
	}
}

// --- Test 12: PlTestGenerateSystemdUnit contains Restart=on-failure ---

func TestGenerateSystemdUnitContainsRestart(t *testing.T) {
	cfg := ServiceConfig{
		BinaryPath: "/usr/bin/prompt-pulse",
		Interval:   "30s",
		LogPath:    "/tmp/pp.log",
		ConfigPath: "/etc/pp.toml",
	}
	unit := PlTestGenerateSystemdUnit(cfg)
	if !strings.Contains(unit, "Restart=on-failure") {
		t.Error("unit should contain Restart=on-failure")
	}
}

// --- Test 13: DiskInfo struct fields ---

func TestDiskInfoStructFields(t *testing.T) {
	d := DiskInfo{
		Path:        "/data",
		FSType:      "ext4",
		Total:       1_000_000_000,
		Used:        500_000_000,
		Free:        500_000_000,
		UsedPercent: 50.0,
		Label:       "data",
	}
	if d.Path != "/data" {
		t.Errorf("Path = %q, want /data", d.Path)
	}
	if d.FSType != "ext4" {
		t.Errorf("FSType = %q, want ext4", d.FSType)
	}
	if d.Total != 1_000_000_000 {
		t.Errorf("Total = %d, want 1000000000", d.Total)
	}
	if d.Label != "data" {
		t.Errorf("Label = %q, want data", d.Label)
	}
}

// --- Test 14: ServiceConfig struct fields ---

func TestServiceConfigStructFields(t *testing.T) {
	cfg := ServiceConfig{
		BinaryPath: "/usr/local/bin/pp",
		Interval:   "30s",
		LogPath:    "/var/log/pp.log",
		ConfigPath: "/etc/pp.toml",
	}
	if cfg.BinaryPath != "/usr/local/bin/pp" {
		t.Errorf("BinaryPath = %q, want /usr/local/bin/pp", cfg.BinaryPath)
	}
	if cfg.Interval != "30s" {
		t.Errorf("Interval = %q, want 30s", cfg.Interval)
	}
	if cfg.LogPath != "/var/log/pp.log" {
		t.Errorf("LogPath = %q, want /var/log/pp.log", cfg.LogPath)
	}
	if cfg.ConfigPath != "/etc/pp.toml" {
		t.Errorf("ConfigPath = %q, want /etc/pp.toml", cfg.ConfigPath)
	}
}

// --- Test 15: ContainerRuntime struct fields ---

func TestContainerRuntimeStructFields(t *testing.T) {
	cr := ContainerRuntime{
		Name:    "docker",
		Running: true,
		Version: "24.0.7",
		Socket:  "/var/run/docker.sock",
	}
	if cr.Name != "docker" {
		t.Errorf("Name = %q, want docker", cr.Name)
	}
	if !cr.Running {
		t.Error("Running should be true")
	}
	if cr.Version != "24.0.7" {
		t.Errorf("Version = %q, want 24.0.7", cr.Version)
	}
	if cr.Socket != "/var/run/docker.sock" {
		t.Errorf("Socket = %q, want /var/run/docker.sock", cr.Socket)
	}
}

// --- Test 16: DetectContainerRuntimes returns a list ---

func TestDetectContainerRuntimesReturnsList(t *testing.T) {
	runtimes := DetectContainerRuntimes()
	// May be empty on CI, but should not panic
	if runtimes == nil {
		// nil is acceptable when nothing is detected
		runtimes = []ContainerRuntime{}
	}
	_ = len(runtimes) // ensure it is a usable slice
}

// --- Test 17: PlTestLaunchdPlistPath contains LaunchAgents ---

func TestLaunchdPlistPathContainsLaunchAgents(t *testing.T) {
	path := PlTestLaunchdPlistPath("/Users/testuser")
	if !strings.Contains(path, "LaunchAgents") {
		t.Errorf("plist path %q should contain LaunchAgents", path)
	}
	if !strings.Contains(path, "com.tinyland.prompt-pulse.plist") {
		t.Errorf("plist path %q should contain the plist filename", path)
	}
}

// --- Test 18: PlTestSystemdUnitPath contains systemd/user ---

func TestSystemdUnitPathContainsSystemdUser(t *testing.T) {
	path := PlTestSystemdUnitPath("/home/testuser")
	if !strings.Contains(path, "systemd/user") {
		t.Errorf("unit path %q should contain systemd/user", path)
	}
	if !strings.Contains(path, "prompt-pulse.service") {
		t.Errorf("unit path %q should contain the service filename", path)
	}
}

// --- Test 19: PlTestFilterDarwinMounts keeps /System/Volumes/Data ---

func TestFilterDarwinMountsKeepsDataVolume(t *testing.T) {
	mounts := []DiskInfo{
		{Path: "/", FSType: "apfs", Total: 500_000_000_000},
		{Path: "/System/Volumes/Data", FSType: "apfs", Total: 500_000_000_000},
		{Path: "/System/Volumes/VM", FSType: "apfs", Total: 500_000_000_000},
		{Path: "/dev", FSType: "devfs", Total: 100},
	}
	filtered := PlTestFilterDarwinMounts(mounts)
	foundData := false
	for _, m := range filtered {
		if m.Path == "/System/Volumes/Data" {
			foundData = true
		}
	}
	if !foundData {
		t.Error("/System/Volumes/Data should be kept after filtering")
	}
}

// --- Test 20: PlTestFilterLinuxMounts keeps root and /home ---

func TestFilterLinuxMountsKeepsRootAndHome(t *testing.T) {
	mounts := []DiskInfo{
		{Path: "/", FSType: "ext4", Total: 500_000_000_000},
		{Path: "/home", FSType: "ext4", Total: 1_000_000_000_000},
		{Path: "/proc", FSType: "proc", Total: 1},
		{Path: "/sys", FSType: "sysfs", Total: 1},
		{Path: "/tmp", FSType: "tmpfs", Total: 4_000_000_000},
	}
	filtered := PlTestFilterLinuxMounts(mounts)
	foundRoot := false
	foundHome := false
	for _, m := range filtered {
		if m.Path == "/" {
			foundRoot = true
		}
		if m.Path == "/home" {
			foundHome = true
		}
	}
	if !foundRoot {
		t.Error("/ should be kept after filtering")
	}
	if !foundHome {
		t.Error("/home should be kept after filtering")
	}
}

// --- Test 21: Launchd plist has KeepAlive key ---

func TestLaunchdPlistHasKeepAlive(t *testing.T) {
	cfg := ServiceConfig{
		BinaryPath: "/usr/local/bin/prompt-pulse",
		Interval:   "30s",
		LogPath:    "/tmp/pp.log",
		ConfigPath: "/etc/pp.toml",
	}
	plist := PlTestGenerateLaunchdPlist(cfg)
	if !strings.Contains(plist, "<key>KeepAlive</key>") {
		t.Error("plist should contain KeepAlive key")
	}
	if !strings.Contains(plist, "<true/>") {
		t.Error("plist KeepAlive should be true")
	}
}

// --- Test 22: Systemd unit has [Service] section ---

func TestSystemdUnitHasServiceSection(t *testing.T) {
	cfg := ServiceConfig{
		BinaryPath: "/usr/bin/prompt-pulse",
		Interval:   "30s",
		LogPath:    "/tmp/pp.log",
		ConfigPath: "/etc/pp.toml",
	}
	unit := PlTestGenerateSystemdUnit(cfg)
	if !strings.Contains(unit, "[Service]") {
		t.Error("unit should contain [Service] section")
	}
	if !strings.Contains(unit, "Type=simple") {
		t.Error("unit should contain Type=simple")
	}
}

// --- Test 23: Container socket paths are reasonable ---

func TestContainerSocketPathsReasonable(t *testing.T) {
	runtimes := DetectContainerRuntimes()
	for _, rt := range runtimes {
		if rt.Socket == "" {
			t.Errorf("runtime %q has empty socket path", rt.Name)
		}
		// Socket paths should be absolute
		if !strings.HasPrefix(rt.Socket, "/") {
			t.Errorf("runtime %q socket path %q should be absolute", rt.Name, rt.Socket)
		}
	}
}

// --- Test 24: Filter removes zero-size mounts (Darwin) ---

func TestFilterDarwinMountsRemovesZeroSize(t *testing.T) {
	mounts := []DiskInfo{
		{Path: "/some/mount", FSType: "apfs", Total: 0},
		{Path: "/", FSType: "apfs", Total: 500_000_000_000},
	}
	filtered := PlTestFilterDarwinMounts(mounts)
	if len(filtered) != 1 {
		t.Errorf("expected 1 mount, got %d", len(filtered))
	}
	if filtered[0].Path != "/" {
		t.Errorf("expected / to survive, got %q", filtered[0].Path)
	}
}

// --- Test 25: Filter removes zero-size mounts (Linux) ---

func TestFilterLinuxMountsRemovesZeroSize(t *testing.T) {
	mounts := []DiskInfo{
		{Path: "/mnt/empty", FSType: "ext4", Total: 0},
		{Path: "/", FSType: "ext4", Total: 500_000_000_000},
	}
	filtered := PlTestFilterLinuxMounts(mounts)
	if len(filtered) != 1 {
		t.Errorf("expected 1 mount, got %d", len(filtered))
	}
}

// --- Test 26: Filter removes overlay mounts (Linux) ---

func TestFilterLinuxMountsRemovesOverlay(t *testing.T) {
	mounts := []DiskInfo{
		{Path: "/var/lib/docker/overlay2/abc", FSType: "overlay", Total: 100_000_000},
		{Path: "/", FSType: "xfs", Total: 500_000_000_000},
	}
	filtered := PlTestFilterLinuxMounts(mounts)
	for _, m := range filtered {
		if m.FSType == "overlay" {
			t.Error("overlay mount should have been filtered out")
		}
	}
}

// --- Test 27: Plist contains config path ---

func TestGenerateLaunchdPlistContainsConfigPath(t *testing.T) {
	cfg := ServiceConfig{
		BinaryPath: "/usr/local/bin/prompt-pulse",
		Interval:   "30s",
		LogPath:    "/tmp/pp.log",
		ConfigPath: "/home/user/.config/prompt-pulse/config.toml",
	}
	plist := PlTestGenerateLaunchdPlist(cfg)
	if !strings.Contains(plist, cfg.ConfigPath) {
		t.Error("plist should contain the config path")
	}
	if !strings.Contains(plist, "--config") {
		t.Error("plist should contain --config argument")
	}
}

// --- Test 28: Systemd unit contains log path ---

func TestGenerateSystemdUnitContainsLogPath(t *testing.T) {
	cfg := ServiceConfig{
		BinaryPath: "/usr/bin/prompt-pulse",
		Interval:   "30s",
		LogPath:    "/var/log/prompt-pulse/daemon.log",
		ConfigPath: "/etc/pp.toml",
	}
	unit := PlTestGenerateSystemdUnit(cfg)
	if !strings.Contains(unit, cfg.LogPath) {
		t.Error("unit should contain the log path")
	}
	if !strings.Contains(unit, "StandardOutput=append:") {
		t.Error("unit should have StandardOutput directive")
	}
}

// --- Test 29: Darwin filter removes Preboot and Update volumes ---

func TestFilterDarwinMountsRemovesSystemVolumes(t *testing.T) {
	mounts := []DiskInfo{
		{Path: "/System/Volumes/Preboot", FSType: "apfs", Total: 500_000_000},
		{Path: "/System/Volumes/Update", FSType: "apfs", Total: 500_000_000},
		{Path: "/", FSType: "apfs", Total: 500_000_000_000},
	}
	filtered := PlTestFilterDarwinMounts(mounts)
	if len(filtered) != 1 {
		t.Errorf("expected 1 mount, got %d", len(filtered))
	}
	if filtered[0].Path != "/" {
		t.Errorf("expected / to survive, got %q", filtered[0].Path)
	}
}

// --- Test 30: Platform constants have expected values ---

func TestPlatformConstants(t *testing.T) {
	if Darwin != "darwin" {
		t.Errorf("Darwin = %q, want darwin", Darwin)
	}
	if Linux != "linux" {
		t.Errorf("Linux = %q, want linux", Linux)
	}
}
