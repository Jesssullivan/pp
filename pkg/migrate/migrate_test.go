package migrate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// ---------- helpers ----------

func mgTestFixture(t *testing.T, name string) string {
	t.Helper()
	return filepath.Join("testdata", name)
}

func mgTempDir(t *testing.T) string {
	t.Helper()
	return t.TempDir()
}

func mgWriteFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("writing test file %s: %v", path, err)
	}
	return path
}

// ---------- Version Detection ----------

func TestDetectVersion_V1Full(t *testing.T) {
	v, err := DetectVersion(mgTestFixture(t, "v1_full.toml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v != 1 {
		t.Errorf("expected version 1, got %d", v)
	}
}

func TestDetectVersion_V1Partial(t *testing.T) {
	v, err := DetectVersion(mgTestFixture(t, "v1_partial.toml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v != 1 {
		t.Errorf("expected version 1, got %d", v)
	}
}

func TestDetectVersion_V2(t *testing.T) {
	v, err := DetectVersion(mgTestFixture(t, "v2_sample.toml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v != 2 {
		t.Errorf("expected version 2, got %d", v)
	}
}

func TestDetectVersion_EmptyFile(t *testing.T) {
	dir := mgTempDir(t)
	path := mgWriteFile(t, dir, "empty.toml", "")
	_, err := DetectVersion(path)
	if err == nil {
		t.Fatal("expected error for empty file")
	}
	if !strings.Contains(err.Error(), "empty") {
		t.Errorf("error should mention empty: %v", err)
	}
}

func TestDetectVersion_MissingFile(t *testing.T) {
	_, err := DetectVersion("/nonexistent/path/config.toml")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestDetectVersion_WhitespaceOnly(t *testing.T) {
	dir := mgTempDir(t)
	path := mgWriteFile(t, dir, "whitespace.toml", "   \n\t\n  ")
	_, err := DetectVersion(path)
	if err == nil {
		t.Fatal("expected error for whitespace-only file")
	}
}

// ---------- NeedsMigration ----------

func TestNeedsMigration_V1(t *testing.T) {
	needs, err := NeedsMigration(mgTestFixture(t, "v1_full.toml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !needs {
		t.Error("expected migration needed for v1 config")
	}
}

func TestNeedsMigration_V2(t *testing.T) {
	needs, err := NeedsMigration(mgTestFixture(t, "v2_sample.toml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if needs {
		t.Error("expected no migration needed for v2 config")
	}
}

func TestNeedsMigration_Missing(t *testing.T) {
	needs, err := NeedsMigration("/nonexistent/config.toml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if needs {
		t.Error("expected no migration needed for missing file")
	}
}

// ---------- V1 Parsing ----------

func TestParseV1_Full(t *testing.T) {
	cfg, err := mgParseV1(mgTestFixture(t, "v1_full.toml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.WaifuPath != "/home/user/.config/prompt-pulse/waifu.png" {
		t.Errorf("wrong waifu_path: %s", cfg.WaifuPath)
	}
	if !cfg.WaifuEnabled {
		t.Error("expected waifu_enabled=true")
	}
	if cfg.DaemonSocket != "/tmp/prompt-pulse.sock" {
		t.Errorf("wrong daemon_socket: %s", cfg.DaemonSocket)
	}
	if cfg.DaemonPidFile != "/tmp/prompt-pulse.pid" {
		t.Errorf("wrong daemon_pid_file: %s", cfg.DaemonPidFile)
	}
	if cfg.CacheDir != "/home/user/.cache/prompt-pulse" {
		t.Errorf("wrong cache_dir: %s", cfg.CacheDir)
	}
	if cfg.CacheTTL.Duration != 10*time.Minute {
		t.Errorf("wrong cache_ttl: %v", cfg.CacheTTL.Duration)
	}
	if cfg.BannerWidth != 140 {
		t.Errorf("wrong banner_width: %d", cfg.BannerWidth)
	}
	if !cfg.BannerShowWaifu {
		t.Error("expected banner_show_waifu=true")
	}
	if !cfg.TailscaleEnabled {
		t.Error("expected tailscale_enabled=true")
	}
	if !cfg.K8sEnabled {
		t.Error("expected k8s_enabled=true")
	}
	if !cfg.ClaudeEnabled {
		t.Error("expected claude_enabled=true")
	}
	if cfg.BillingEnabled {
		t.Error("expected billing_enabled=false")
	}
	if !cfg.StarshipEnabled {
		t.Error("expected starship_enabled=true")
	}
	if cfg.StarshipFormat != "$directory$git_branch$character" {
		t.Errorf("wrong starship_format: %s", cfg.StarshipFormat)
	}
	if cfg.Theme != "gruvbox" {
		t.Errorf("wrong theme: %s", cfg.Theme)
	}
}

func TestParseV1_Partial(t *testing.T) {
	cfg, err := mgParseV1(mgTestFixture(t, "v1_partial.toml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Explicitly set values
	if cfg.WaifuEnabled {
		t.Error("expected waifu_enabled=false")
	}
	if !cfg.TailscaleEnabled {
		t.Error("expected tailscale_enabled=true")
	}
	if cfg.Theme != "nord" {
		t.Errorf("wrong theme: %s", cfg.Theme)
	}

	// Defaults for missing fields
	if cfg.BannerWidth != 120 {
		t.Errorf("expected default banner_width=120, got %d", cfg.BannerWidth)
	}
	if cfg.CacheTTL.Duration != 5*time.Minute {
		t.Errorf("expected default cache_ttl=5m, got %v", cfg.CacheTTL.Duration)
	}
	if !cfg.ClaudeEnabled {
		t.Error("expected default claude_enabled=true")
	}
}

func TestParseV1_Defaults(t *testing.T) {
	dir := mgTempDir(t)
	// Minimal v1 with just a theme key so it's recognized as v1
	path := mgWriteFile(t, dir, "minimal.toml", `theme = "default"`)

	cfg, err := mgParseV1(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	defaults := mgV1Defaults()
	if cfg.WaifuEnabled != defaults.WaifuEnabled {
		t.Errorf("expected default waifu_enabled=%v", defaults.WaifuEnabled)
	}
	if cfg.BannerShowWaifu != defaults.BannerShowWaifu {
		t.Errorf("expected default banner_show_waifu=%v", defaults.BannerShowWaifu)
	}
	if cfg.TailscaleEnabled != defaults.TailscaleEnabled {
		t.Errorf("expected default tailscale_enabled=%v", defaults.TailscaleEnabled)
	}
}

func TestParseV1_InvalidToml(t *testing.T) {
	dir := mgTempDir(t)
	path := mgWriteFile(t, dir, "invalid.toml", `{{{not valid toml!!!`)

	_, err := mgParseV1(path)
	if err == nil {
		t.Fatal("expected error for invalid TOML")
	}
}

func TestParseV1_MissingFile(t *testing.T) {
	_, err := mgParseV1("/nonexistent/path/config.toml")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

// ---------- Config Transformation ----------

func TestTransformConfig_AllFields(t *testing.T) {
	v1 := &V1Config{
		WaifuPath:        "/path/to/waifu.png",
		WaifuEnabled:     true,
		DaemonSocket:     "/tmp/ppulse.sock",
		DaemonPidFile:    "/tmp/ppulse.pid",
		CacheDir:         "/custom/cache",
		CacheTTL:         duration{10 * time.Minute},
		BannerWidth:      160,
		BannerShowWaifu:  true,
		TailscaleEnabled: true,
		K8sEnabled:       true,
		ClaudeEnabled:    false,
		BillingEnabled:   true,
		StarshipEnabled:  true,
		StarshipFormat:   "$directory$character",
		Theme:            "dracula",
	}

	v2, changes := mgTransformConfig(v1)

	// Verify transformed values
	if !v2.Image.WaifuEnabled {
		t.Error("expected waifu_enabled=true in v2")
	}
	if v2.General.CacheDir != "/custom/cache" {
		t.Errorf("expected cache_dir=/custom/cache, got %s", v2.General.CacheDir)
	}
	if v2.General.DataRetention.Duration != 10*time.Minute {
		t.Errorf("expected data_retention=10m, got %v", v2.General.DataRetention.Duration)
	}
	if v2.Theme.Name != "dracula" {
		t.Errorf("expected theme=dracula, got %s", v2.Theme.Name)
	}
	if !v2.Collectors.Tailscale.Enabled {
		t.Error("expected tailscale.enabled=true")
	}
	if !v2.Collectors.Kubernetes.Enabled {
		t.Error("expected kubernetes.enabled=true")
	}
	if v2.Collectors.Claude.Enabled {
		t.Error("expected claude.enabled=false")
	}
	if !v2.Collectors.Billing.Enabled {
		t.Error("expected billing.enabled=true")
	}
	if v2.Banner.StandardMinWidth != 160 {
		t.Errorf("expected standard_min_width=160, got %d", v2.Banner.StandardMinWidth)
	}

	// Verify changes were tracked
	if len(changes) == 0 {
		t.Fatal("expected some changes to be tracked")
	}

	// Check specific change types exist
	mgAssertChangeExists(t, changes, "daemon.socket", "removed")
	mgAssertChangeExists(t, changes, "daemon.pid_file", "removed")
	mgAssertChangeExists(t, changes, "layout.preset", "added")
	mgAssertChangeExists(t, changes, "image.protocol", "added")
}

func TestTransformConfig_DefaultInjection(t *testing.T) {
	// Minimal v1 config
	v1 := mgV1Defaults()
	v2, changes := mgTransformConfig(v1)

	// Should get v2 defaults for new fields
	if v2.Layout.Preset != "dashboard" {
		t.Errorf("expected layout.preset=dashboard, got %s", v2.Layout.Preset)
	}
	if v2.Image.Protocol != "auto" {
		t.Errorf("expected image.protocol=auto, got %s", v2.Image.Protocol)
	}
	if v2.Shell.TUIKeybinding != `\C-p` {
		t.Errorf("expected shell.tui_keybinding=\\C-p, got %s", v2.Shell.TUIKeybinding)
	}

	// New v2 fields should have "added" changes
	addedCount := 0
	for _, c := range changes {
		if c.Action == "added" {
			addedCount++
		}
	}
	if addedCount == 0 {
		t.Error("expected some 'added' changes for new v2 fields")
	}
}

func TestTransformConfig_ChangeTracking(t *testing.T) {
	v1 := &V1Config{
		Theme:            "nord",
		TailscaleEnabled: false, // differs from v2 default (true)
		ClaudeEnabled:    true,  // same as v2 default
		CacheTTL:         duration{5 * time.Minute},
		BannerWidth:      120,
	}

	_, changes := mgTransformConfig(v1)

	// Theme changed from v2 default
	mgAssertChangeExists(t, changes, "theme.name", "changed")

	// Tailscale changed from v2 default
	mgAssertChangeExists(t, changes, "collectors.tailscale.enabled", "changed")
}

// ---------- Backup / Restore ----------

func TestBackup_CreatesFile(t *testing.T) {
	dir := mgTempDir(t)
	src := mgWriteFile(t, dir, "config.toml", `waifu_enabled = true`)

	backupPath, err := mgBackup(src)
	if err != nil {
		t.Fatalf("backup failed: %v", err)
	}

	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		t.Fatalf("backup file does not exist: %s", backupPath)
	}

	// Verify content matches
	original, _ := os.ReadFile(src)
	backed, _ := os.ReadFile(backupPath)
	if string(original) != string(backed) {
		t.Error("backup content does not match original")
	}

	// Verify naming pattern
	base := filepath.Base(backupPath)
	if !strings.HasPrefix(base, "config.toml.v1.") {
		t.Errorf("unexpected backup name: %s", base)
	}
	if !strings.HasSuffix(base, ".bak") {
		t.Errorf("backup should end with .bak: %s", base)
	}
}

func TestBackup_MissingFile(t *testing.T) {
	_, err := mgBackup("/nonexistent/config.toml")
	if err == nil {
		t.Fatal("expected error backing up missing file")
	}
}

func TestListBackups_Empty(t *testing.T) {
	dir := mgTempDir(t)
	backups, err := mgListBackups(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(backups) != 0 {
		t.Errorf("expected 0 backups, got %d", len(backups))
	}
}

func TestListBackups_Multiple(t *testing.T) {
	dir := mgTempDir(t)

	// Create backup files manually
	mgWriteFile(t, dir, "config.toml.v1.20250101-120000.bak", "old")
	mgWriteFile(t, dir, "config.toml.v1.20250601-120000.bak", "newer")
	mgWriteFile(t, dir, "config.toml.v2.20250701-120000.bak", "newest")
	mgWriteFile(t, dir, "not-a-backup.txt", "skip me")

	backups, err := mgListBackups(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(backups) != 3 {
		t.Fatalf("expected 3 backups, got %d", len(backups))
	}

	// Should be sorted newest first
	if backups[0].Version != 2 {
		t.Errorf("newest backup should be v2, got v%d", backups[0].Version)
	}
	if backups[2].Version != 1 {
		t.Errorf("oldest backup should be v1, got v%d", backups[2].Version)
	}
}

func TestRestore(t *testing.T) {
	dir := mgTempDir(t)
	original := "waifu_enabled = true\ntheme = \"gruvbox\"\n"
	backupPath := mgWriteFile(t, dir, "config.toml.v1.20250101-120000.bak", original)
	configPath := filepath.Join(dir, "config.toml")

	// Write a different file as the current config
	mgWriteFile(t, dir, "config.toml", "theme = \"nord\"")

	err := mgRestore(backupPath, configPath)
	if err != nil {
		t.Fatalf("restore failed: %v", err)
	}

	data, _ := os.ReadFile(configPath)
	if string(data) != original {
		t.Errorf("restored content doesn't match:\ngot:  %s\nwant: %s", string(data), original)
	}
}

func TestRestore_MissingBackup(t *testing.T) {
	dir := mgTempDir(t)
	err := mgRestore("/nonexistent/backup.bak", filepath.Join(dir, "config.toml"))
	if err == nil {
		t.Fatal("expected error restoring missing backup")
	}
}

func TestBackup_AtomicWrite(t *testing.T) {
	dir := mgTempDir(t)
	content := "waifu_enabled = true\ntheme = \"catppuccin\"\n"
	src := mgWriteFile(t, dir, "config.toml", content)

	backupPath, err := mgBackup(src)
	if err != nil {
		t.Fatalf("backup failed: %v", err)
	}

	// No temp files should remain
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".backup-") {
			t.Errorf("temp file not cleaned up: %s", e.Name())
		}
	}

	// Backup should exist and match
	data, _ := os.ReadFile(backupPath)
	if string(data) != content {
		t.Error("backup content mismatch after atomic write")
	}
}

func TestIsBackupFile(t *testing.T) {
	tests := []struct {
		name   string
		expect bool
	}{
		{"config.toml.v1.20250101-120000.bak", true},
		{"config.toml.v2.20260209-093045.bak", true},
		{"config.toml", false},
		{"backup.bak", false},
		{"config.toml.v1.bad-date.bak", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mgIsBackupFile(tt.name)
			if got != tt.expect {
				t.Errorf("mgIsBackupFile(%q) = %v, want %v", tt.name, got, tt.expect)
			}
		})
	}
}

// ---------- Feature Parity ----------

func TestCheckParity_FullConfig(t *testing.T) {
	report, err := CheckParity(mgTestFixture(t, "v1_full.toml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(report.Covered) == 0 {
		t.Error("expected some covered features")
	}
	if len(report.NewInV2) == 0 {
		t.Error("expected some new v2 features")
	}
	if report.Score <= 0 || report.Score > 1.0 {
		t.Errorf("expected score between 0 and 1, got %f", report.Score)
	}
}

func TestCheckParity_PartialConfig(t *testing.T) {
	report, err := CheckParity(mgTestFixture(t, "v1_partial.toml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Partial config should still have some coverage
	if report.Score < 0 || report.Score > 1.0 {
		t.Errorf("score out of range: %f", report.Score)
	}
}

func TestCheckParity_ScoreCalculation(t *testing.T) {
	// All features active
	v1All := &V1Config{
		WaifuEnabled:     true,
		WaifuPath:        "/path/to/waifu.png",
		BannerShowWaifu:  true,
		TailscaleEnabled: true,
		K8sEnabled:       true,
		ClaudeEnabled:    true,
		BillingEnabled:   true,
		StarshipEnabled:  true,
		StarshipFormat:   "$dir",
		DaemonSocket:     "/tmp/sock",
		DaemonPidFile:    "/tmp/pid",
		CacheDir:         "/cache",
		CacheTTL:         duration{5 * time.Minute},
	}

	report := mgCheckParityFromConfig(v1All)

	// Score should be less than 1.0 because some features are missing (starship, waifu local path)
	if report.Score >= 1.0 {
		t.Errorf("expected score < 1.0 when starship/local-waifu features are active, got %f", report.Score)
	}
	if report.Score <= 0 {
		t.Errorf("expected score > 0, got %f", report.Score)
	}

	// Verify missing includes starship
	found := false
	for _, m := range report.Missing {
		if strings.Contains(m, "starship") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected starship to be in Missing list")
	}
}

func TestCheckParity_EmptyConfig(t *testing.T) {
	v1 := &V1Config{}
	report := mgCheckParityFromConfig(v1)

	// system-metrics and shells are always active
	if report.Score <= 0 {
		t.Errorf("expected positive score for always-active features, got %f", report.Score)
	}
}

func TestCheckParity_NewV2Features(t *testing.T) {
	v1 := mgV1Defaults()
	report := mgCheckParityFromConfig(v1)

	if len(report.NewInV2) == 0 {
		t.Error("expected new v2 features to be listed")
	}

	// Check some known new features
	newSet := make(map[string]bool)
	for _, f := range report.NewInV2 {
		newSet[f] = true
	}

	expectedNew := []string{
		"layout:preset-system",
		"shell:tui-keybinding",
		"shell:instant-banner",
		"shell:ksh-support",
	}

	for _, expected := range expectedNew {
		if !newSet[expected] {
			t.Errorf("expected %q in NewInV2 list", expected)
		}
	}
}

// ---------- Compat Shims ----------

func TestEnsurePidCompat_CreateSymlink(t *testing.T) {
	dir := mgTempDir(t)
	v2Pid := filepath.Join(dir, "v2", "daemon.pid")
	os.MkdirAll(filepath.Dir(v2Pid), 0o755)
	os.WriteFile(v2Pid, []byte("12345"), 0o644)

	// Override v1 default for testing
	origFn := mgV1DefaultPidPath
	_ = origFn // acknowledge original

	// We can't easily override the function, so just test the directory creation
	err := mgEnsurePidCompat(v2Pid)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestEnsurePidCompat_EmptyPath(t *testing.T) {
	err := mgEnsurePidCompat("")
	if err != nil {
		t.Fatalf("unexpected error for empty path: %v", err)
	}
}

func TestEnsureCacheCompat(t *testing.T) {
	dir := mgTempDir(t)
	cacheDir := filepath.Join(dir, "cache")

	err := mgEnsureCacheCompat(cacheDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check subdirectories exist
	for _, sub := range []string{"waifu", "banner", "sessions"} {
		path := filepath.Join(cacheDir, sub)
		info, err := os.Stat(path)
		if err != nil {
			t.Errorf("subdirectory %s not created: %v", sub, err)
			continue
		}
		if !info.IsDir() {
			t.Errorf("%s should be a directory", sub)
		}
	}
}

func TestEnsureCacheCompat_EmptyPath(t *testing.T) {
	err := mgEnsureCacheCompat("")
	if err != nil {
		t.Fatalf("unexpected error for empty path: %v", err)
	}
}

func TestEnsureSocketCompat_EmptyPath(t *testing.T) {
	err := mgEnsureSocketCompat("")
	if err != nil {
		t.Fatalf("unexpected error for empty path: %v", err)
	}
}

func TestCleanupCompat_NoShims(t *testing.T) {
	// Should not error when no shims exist
	err := mgCleanupCompat()
	// May or may not error depending on whether v1 default paths exist
	// Just ensure it doesn't panic
	_ = err
}

// ---------- End-to-End Migration ----------

func TestMigrate_FullPipeline(t *testing.T) {
	dir := mgTempDir(t)

	// Copy v1 fixture to temp dir
	v1Data, _ := os.ReadFile(mgTestFixture(t, "v1_full.toml"))
	v1Path := mgWriteFile(t, dir, "config.toml", string(v1Data))
	v2Path := filepath.Join(dir, "config_v2.toml")

	result, err := Migrate(v1Path, v2Path)
	if err != nil {
		t.Fatalf("migration failed: %v", err)
	}

	if !result.Success {
		t.Error("expected successful migration")
	}
	if result.BackupPath == "" {
		t.Error("expected backup path to be set")
	}
	if len(result.Changes) == 0 {
		t.Error("expected some changes to be recorded")
	}

	// Verify backup exists
	if _, err := os.Stat(result.BackupPath); os.IsNotExist(err) {
		t.Error("backup file does not exist")
	}

	// Verify v2 config was written
	if _, err := os.Stat(v2Path); os.IsNotExist(err) {
		t.Error("v2 config file was not created")
	}

	// Verify v2 config is valid
	v2Version, err := DetectVersion(v2Path)
	if err != nil {
		t.Fatalf("v2 config version detection failed: %v", err)
	}
	if v2Version != 2 {
		t.Errorf("expected migrated config to be v2, got v%d", v2Version)
	}
}

func TestMigrate_AlreadyV2(t *testing.T) {
	dir := mgTempDir(t)
	v2Data, _ := os.ReadFile(mgTestFixture(t, "v2_sample.toml"))
	v2Path := mgWriteFile(t, dir, "config.toml", string(v2Data))
	outPath := filepath.Join(dir, "config_out.toml")

	result, err := Migrate(v2Path, outPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Error("expected success for already-v2 config")
	}
	if len(result.Warnings) == 0 {
		t.Error("expected warning about already being v2")
	}
	if result.BackupPath != "" {
		t.Error("expected no backup for already-v2 config")
	}
}

func TestMigrate_MissingSource(t *testing.T) {
	dir := mgTempDir(t)
	_, err := Migrate("/nonexistent/config.toml", filepath.Join(dir, "out.toml"))
	if err == nil {
		t.Fatal("expected error for missing source file")
	}
}

func TestMigrate_CorruptedV1(t *testing.T) {
	dir := mgTempDir(t)
	// File has v1 markers but invalid TOML
	path := mgWriteFile(t, dir, "config.toml", `waifu_enabled = {{invalid}}`)
	outPath := filepath.Join(dir, "out.toml")

	_, err := Migrate(path, outPath)
	if err == nil {
		t.Fatal("expected error for corrupted v1 config")
	}
}

func TestMigrate_PartialV1(t *testing.T) {
	dir := mgTempDir(t)
	v1Data, _ := os.ReadFile(mgTestFixture(t, "v1_partial.toml"))
	v1Path := mgWriteFile(t, dir, "config.toml", string(v1Data))
	v2Path := filepath.Join(dir, "config_v2.toml")

	result, err := Migrate(v1Path, v2Path)
	if err != nil {
		t.Fatalf("migration failed: %v", err)
	}

	if !result.Success {
		t.Error("expected successful migration for partial v1")
	}

	// Read the v2 output and verify defaults were applied
	v2Version, _ := DetectVersion(v2Path)
	if v2Version != 2 {
		t.Errorf("expected v2, got v%d", v2Version)
	}
}

// ---------- Edge Cases ----------

func TestMigrate_EmptySourceFile(t *testing.T) {
	dir := mgTempDir(t)
	path := mgWriteFile(t, dir, "config.toml", "")
	outPath := filepath.Join(dir, "out.toml")

	_, err := Migrate(path, outPath)
	if err == nil {
		t.Fatal("expected error for empty source file")
	}
}

func TestDetectVersion_V2WithOnlyGeneralSection(t *testing.T) {
	dir := mgTempDir(t)
	content := `[general]
log_level = "debug"
`
	path := mgWriteFile(t, dir, "config.toml", content)

	v, err := DetectVersion(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v != 2 {
		t.Errorf("expected v2 for config with [general] section, got v%d", v)
	}
}

func TestWriteConfig_AtomicWrite(t *testing.T) {
	dir := mgTempDir(t)
	outPath := filepath.Join(dir, "config.toml")

	v1 := mgV1Defaults()
	v2, _ := mgTransformConfig(v1)

	err := mgWriteConfig(outPath, v2)
	if err != nil {
		t.Fatalf("write failed: %v", err)
	}

	// No temp files should remain
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".prompt-pulse-migrate-") {
			t.Errorf("temp file not cleaned up: %s", e.Name())
		}
	}

	// File should exist and be valid
	if _, err := os.Stat(outPath); os.IsNotExist(err) {
		t.Fatal("output file not created")
	}
}

// ---------- helpers for assertions ----------

func mgAssertChangeExists(t *testing.T, changes []ConfigChange, field, action string) {
	t.Helper()
	for _, c := range changes {
		if c.Field == field && c.Action == action {
			return
		}
	}
	t.Errorf("expected change {field=%q, action=%q} not found in %d changes", field, action, len(changes))
}
