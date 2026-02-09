package deploy

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// ---------- helpers ----------

// testProfile returns a HostProfile pointing at the given temp directory
// with a fake binary, config, cache, and socket laid out.
func testProfile(t *testing.T, dir string) *HostProfile {
	t.Helper()

	binPath := filepath.Join(dir, "bin", "prompt-pulse")
	confDir := filepath.Join(dir, "config")
	confPath := filepath.Join(confDir, "config.toml")
	cacheDir := filepath.Join(dir, "cache")
	sockPath := filepath.Join(dir, "prompt-pulse.sock")

	for _, d := range []string{
		filepath.Join(dir, "bin"),
		confDir,
		cacheDir,
		filepath.Join(cacheDir, "waifu"),
		filepath.Join(cacheDir, "banner"),
		filepath.Join(cacheDir, "sessions"),
		filepath.Join(cacheDir, "shells"),
		filepath.Join(cacheDir, "collectors"),
	} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	// Executable binary stub.
	if err := os.WriteFile(binPath, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	// Config stub.
	if err := os.WriteFile(confPath, []byte("[general]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Socket stub.
	if err := os.WriteFile(sockPath, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	// Theme file.
	if err := os.WriteFile(filepath.Join(cacheDir, "theme.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	return &HostProfile{
		Name:               "test-host",
		OS:                 "darwin",
		Arch:               "aarch64",
		Features:           []string{"sysmetrics"},
		Shells:             []string{"bash"},
		ExpectedCollectors: []string{"sysmetrics"},
		BinaryPath:         binPath,
		ConfigPath:         confPath,
		CacheDir:           cacheDir,
		SocketPath:         sockPath,
	}
}

// ---------- Host profile tests ----------

func TestXoxdBatesProfile(t *testing.T) {
	p := XoxdBates()
	if p.Name != "xoxd-bates" {
		t.Errorf("name = %q, want xoxd-bates", p.Name)
	}
	if p.OS != "darwin" {
		t.Errorf("os = %q, want darwin", p.OS)
	}
	if p.Arch != "aarch64" {
		t.Errorf("arch = %q, want aarch64", p.Arch)
	}
	if len(p.Features) != 5 {
		t.Errorf("features count = %d, want 5", len(p.Features))
	}
	if len(p.Shells) != 3 {
		t.Errorf("shells count = %d, want 3", len(p.Shells))
	}
}

func TestHoneyProfile(t *testing.T) {
	p := Honey()
	if p.Name != "honey" {
		t.Errorf("name = %q, want honey", p.Name)
	}
	if p.OS != "linux" {
		t.Errorf("os = %q, want linux", p.OS)
	}
	if p.Arch != "x86_64" {
		t.Errorf("arch = %q, want x86_64", p.Arch)
	}
	if len(p.Features) != 5 {
		t.Errorf("features count = %d, want 5", len(p.Features))
	}
	if len(p.Shells) != 2 {
		t.Errorf("shells count = %d, want 2", len(p.Shells))
	}
}

func TestPettingZooMiniProfile(t *testing.T) {
	p := PettingZooMini()
	if p.Name != "petting-zoo-mini" {
		t.Errorf("name = %q, want petting-zoo-mini", p.Name)
	}
	if p.OS != "darwin" {
		t.Errorf("os = %q, want darwin", p.OS)
	}
	if len(p.Features) != 4 {
		t.Errorf("features count = %d, want 4", len(p.Features))
	}
}

func TestNewHostProfile(t *testing.T) {
	p := NewHostProfile("custom", "linux", "x86_64")
	if p.Name != "custom" || p.OS != "linux" || p.Arch != "x86_64" {
		t.Errorf("unexpected profile: %+v", p)
	}
	if len(p.Features) != 0 {
		t.Errorf("new profile should have no features")
	}
}

func TestProfileExpectedCollectors(t *testing.T) {
	p := XoxdBates()
	want := map[string]bool{
		"waifu": true, "tailscale": true, "claude": true,
		"billing": true, "sysmetrics": true,
	}
	for _, c := range p.ExpectedCollectors {
		if !want[c] {
			t.Errorf("unexpected collector %q", c)
		}
	}
}

// ---------- Individual check tests ----------

func TestCheckBinary_Exists(t *testing.T) {
	dir := t.TempDir()
	p := testProfile(t, dir)

	c := dpCheckBinary(p)
	passed, msg := c.Run()
	if !passed {
		t.Errorf("binary check failed: %s", msg)
	}
}

func TestCheckBinary_Missing(t *testing.T) {
	p := &HostProfile{BinaryPath: "/nonexistent/prompt-pulse"}
	c := dpCheckBinary(p)
	passed, _ := c.Run()
	if passed {
		t.Error("binary check should fail for missing binary")
	}
}

func TestCheckBinary_NotExecutable(t *testing.T) {
	dir := t.TempDir()
	binPath := filepath.Join(dir, "prompt-pulse")
	if err := os.WriteFile(binPath, []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}
	p := &HostProfile{BinaryPath: binPath}
	c := dpCheckBinary(p)
	passed, msg := c.Run()
	if passed {
		t.Errorf("binary check should fail for non-executable; msg=%s", msg)
	}
}

func TestCheckConfig_Exists(t *testing.T) {
	dir := t.TempDir()
	p := testProfile(t, dir)

	c := dpCheckConfig(p)
	passed, msg := c.Run()
	if !passed {
		t.Errorf("config check failed: %s", msg)
	}
}

func TestCheckConfig_Missing(t *testing.T) {
	p := &HostProfile{ConfigPath: "/nonexistent/config.toml"}
	c := dpCheckConfig(p)
	passed, _ := c.Run()
	if passed {
		t.Error("config check should fail for missing config")
	}
}

func TestCheckCache_Complete(t *testing.T) {
	dir := t.TempDir()
	p := testProfile(t, dir)

	c := dpCheckCache(p)
	passed, msg := c.Run()
	if !passed {
		t.Errorf("cache check failed: %s", msg)
	}
}

func TestCheckCache_MissingSubdir(t *testing.T) {
	dir := t.TempDir()
	cacheDir := filepath.Join(dir, "cache")
	os.MkdirAll(cacheDir, 0o755)
	// Only create waifu, skip banner and sessions.
	os.MkdirAll(filepath.Join(cacheDir, "waifu"), 0o755)

	p := &HostProfile{CacheDir: cacheDir}
	c := dpCheckCache(p)
	passed, msg := c.Run()
	if passed {
		t.Error("cache check should fail with missing subdirs")
	}
	if !strings.Contains(msg, "banner") {
		t.Errorf("message should mention missing 'banner': %s", msg)
	}
}

func TestCheckShell_Exists(t *testing.T) {
	dir := t.TempDir()
	p := testProfile(t, dir)

	shellDir := filepath.Join(p.CacheDir, "shells")
	os.MkdirAll(shellDir, 0o755)
	os.WriteFile(filepath.Join(shellDir, "bash.sh"), []byte("# integration"), 0o644)

	c := dpCheckShell(p, "bash")
	passed, msg := c.Run()
	if !passed {
		t.Errorf("shell check failed: %s", msg)
	}
}

func TestCheckShell_Missing(t *testing.T) {
	p := &HostProfile{CacheDir: t.TempDir()}
	c := dpCheckShell(p, "fish")
	passed, _ := c.Run()
	if passed {
		t.Error("shell check should fail when integration file missing")
	}
}

func TestCheckPermissions_OK(t *testing.T) {
	dir := t.TempDir()
	p := testProfile(t, dir)

	c := dpCheckPermissions(p)
	passed, msg := c.Run()
	if !passed {
		t.Errorf("permissions check failed: %s", msg)
	}
}

func TestCheckPermissions_WorldWritable(t *testing.T) {
	dir := t.TempDir()
	p := testProfile(t, dir)

	// Make cache dir world-writable.
	os.Chmod(p.CacheDir, 0o777)

	c := dpCheckPermissions(p)
	passed, msg := c.Run()
	if passed {
		t.Errorf("permissions check should fail for world-writable dir; msg=%s", msg)
	}
}

func TestCheckDaemon_SocketExists(t *testing.T) {
	dir := t.TempDir()
	p := testProfile(t, dir)

	c := dpCheckDaemon(p)
	passed, msg := c.Run()
	if !passed {
		t.Errorf("daemon check failed: %s", msg)
	}
}

func TestCheckDaemon_NeitherExists(t *testing.T) {
	p := &HostProfile{
		SocketPath: "/nonexistent/socket",
		PIDFile:    "/nonexistent/pid",
	}
	c := dpCheckDaemon(p)
	passed, _ := c.Run()
	if passed {
		t.Error("daemon check should fail when no socket or pid file")
	}
}

// ---------- Verify tests ----------

func TestVerify_PassesWithFullProfile(t *testing.T) {
	dir := t.TempDir()
	p := testProfile(t, dir)

	// Create collector data.
	colDir := filepath.Join(p.CacheDir, "collectors")
	os.WriteFile(filepath.Join(colDir, "sysmetrics.json"), []byte(`{}`), 0o644)

	// Create shell integration.
	shellDir := filepath.Join(p.CacheDir, "shells")
	os.WriteFile(filepath.Join(shellDir, "bash.sh"), []byte("# ok"), 0o644)

	result, err := Verify(p)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Passed {
		for _, c := range result.Checks {
			if !c.Passed {
				t.Logf("failed check: %s: %s", c.Name, c.Message)
			}
		}
		t.Error("verify should pass with complete profile")
	}
}

func TestVerify_NilProfile(t *testing.T) {
	_, err := Verify(nil)
	if err == nil {
		t.Error("verify with nil profile should error")
	}
}

func TestVerify_FailsOnMissingBinary(t *testing.T) {
	p := &HostProfile{
		Name:       "broken",
		BinaryPath: "/nonexistent/binary",
		ConfigPath: "/nonexistent/config",
		CacheDir:   "/nonexistent/cache",
	}
	result, err := Verify(p)
	if err != nil {
		t.Fatal(err)
	}
	if result.Passed {
		t.Error("verify should fail when binary is missing")
	}
}

// ---------- Rollout plan tests ----------

func TestDefaultRolloutPlan(t *testing.T) {
	plan := DefaultRolloutPlan()
	if plan.Strategy != "serial" {
		t.Errorf("strategy = %q, want serial", plan.Strategy)
	}
	if len(plan.Hosts) != 3 {
		t.Fatalf("hosts = %d, want 3", len(plan.Hosts))
	}
	// Should be ordered by risk.
	if plan.Hosts[0].Profile.Name != "xoxd-bates" {
		t.Errorf("first host = %q, want xoxd-bates", plan.Hosts[0].Profile.Name)
	}
	if plan.Hosts[1].Profile.Name != "petting-zoo-mini" {
		t.Errorf("second host = %q, want petting-zoo-mini", plan.Hosts[1].Profile.Name)
	}
	if plan.Hosts[2].Profile.Name != "honey" {
		t.Errorf("third host = %q, want honey", plan.Hosts[2].Profile.Name)
	}
}

func TestRolloutPlan_Validate_Valid(t *testing.T) {
	plan := DefaultRolloutPlan()
	problems := plan.Validate()
	if len(problems) != 0 {
		t.Errorf("default plan should be valid, got: %v", problems)
	}
}

func TestRolloutPlan_Validate_InvalidStrategy(t *testing.T) {
	plan := NewRolloutPlan("random")
	plan.AddHost(XoxdBates(), 1)
	problems := plan.Validate()
	if len(problems) == 0 {
		t.Error("plan with invalid strategy should have problems")
	}
}

func TestRolloutPlan_Validate_NoHosts(t *testing.T) {
	plan := NewRolloutPlan("serial")
	problems := plan.Validate()
	found := false
	for _, p := range problems {
		if strings.Contains(p, "no hosts") {
			found = true
		}
	}
	if !found {
		t.Errorf("plan with no hosts should report that problem, got: %v", problems)
	}
}

func TestRolloutPlan_Validate_DuplicateOrders(t *testing.T) {
	plan := NewRolloutPlan("serial")
	plan.AddHost(XoxdBates(), 1)
	// Manually force a duplicate order.
	plan.Hosts = append(plan.Hosts, HostRollout{
		Profile: Honey(),
		Order:   1,
	})
	problems := plan.Validate()
	found := false
	for _, p := range problems {
		if strings.Contains(p, "duplicate order") {
			found = true
		}
	}
	if !found {
		t.Errorf("duplicate orders should be reported, got: %v", problems)
	}
}

func TestRolloutPlan_CustomPlan(t *testing.T) {
	plan := NewRolloutPlan("parallel")
	plan.AddHost(Honey(), 1)
	plan.AddHost(XoxdBates(), 2)
	problems := plan.Validate()
	if len(problems) != 0 {
		t.Errorf("custom parallel plan should be valid, got: %v", problems)
	}
	if plan.Strategy != "parallel" {
		t.Errorf("strategy = %q, want parallel", plan.Strategy)
	}
}

func TestRolloutPlan_AddHostSorts(t *testing.T) {
	plan := NewRolloutPlan("serial")
	plan.AddHost(Honey(), 3)
	plan.AddHost(XoxdBates(), 1)
	plan.AddHost(PettingZooMini(), 2)

	if plan.Hosts[0].Order != 1 || plan.Hosts[1].Order != 2 || plan.Hosts[2].Order != 3 {
		t.Errorf("hosts should be sorted by order: %d, %d, %d",
			plan.Hosts[0].Order, plan.Hosts[1].Order, plan.Hosts[2].Order)
	}
}

// ---------- Report tests ----------

func TestReport_TextFormat(t *testing.T) {
	r := NewReport(
		VerifyResult{Host: "host-a", Passed: true, Checks: []CheckResult{
			{Name: "binary", Passed: true, Message: "ok", Duration: time.Millisecond},
		}},
		VerifyResult{Host: "host-b", Passed: false, Checks: []CheckResult{
			{Name: "binary", Passed: false, Message: "missing", Duration: 2 * time.Millisecond},
		}},
	)
	text := r.RenderText()
	if !strings.Contains(text, "host-a [PASS]") {
		t.Error("text should contain host-a PASS")
	}
	if !strings.Contains(text, "host-b [FAIL]") {
		t.Error("text should contain host-b FAIL")
	}
	if !strings.Contains(text, "2 total") {
		t.Error("text should contain host total")
	}
}

func TestReport_MarkdownFormat(t *testing.T) {
	r := NewReport(
		VerifyResult{Host: "host-a", Passed: true, Checks: []CheckResult{
			{Name: "config", Passed: true, Message: "ok"},
		}},
	)
	md := r.RenderMarkdown()
	if !strings.Contains(md, "# Deployment Verification Report") {
		t.Error("markdown should have title")
	}
	if !strings.Contains(md, "| Check |") {
		t.Error("markdown should have table header")
	}
	if !strings.Contains(md, "host-a") {
		t.Error("markdown should contain hostname")
	}
}

func TestReport_JSONFormat(t *testing.T) {
	r := NewReport(
		VerifyResult{
			Host: "host-a", Passed: true,
			Checks:    []CheckResult{{Name: "bin", Passed: true, Message: "ok", Duration: time.Millisecond}},
			Timestamp: time.Date(2026, 2, 9, 12, 0, 0, 0, time.UTC),
		},
	)
	js, err := r.RenderJSON()
	if err != nil {
		t.Fatal(err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(js), &parsed); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	results, ok := parsed["results"].([]interface{})
	if !ok || len(results) != 1 {
		t.Errorf("expected 1 result in JSON, got %v", parsed["results"])
	}
}

func TestReport_SummaryCalculation(t *testing.T) {
	s := dpComputeSummary([]VerifyResult{
		{Passed: true, Checks: []CheckResult{
			{Passed: true}, {Passed: true}, {Passed: false},
		}},
		{Passed: false, Checks: []CheckResult{
			{Passed: false}, {Passed: false},
		}},
	})
	if s.TotalHosts != 2 {
		t.Errorf("total hosts = %d, want 2", s.TotalHosts)
	}
	if s.PassedHosts != 1 {
		t.Errorf("passed hosts = %d, want 1", s.PassedHosts)
	}
	if s.FailedHosts != 1 {
		t.Errorf("failed hosts = %d, want 1", s.FailedHosts)
	}
	if s.TotalChecks != 5 {
		t.Errorf("total checks = %d, want 5", s.TotalChecks)
	}
	if s.PassedChecks != 2 {
		t.Errorf("passed checks = %d, want 2", s.PassedChecks)
	}
	if s.FailedChecks != 3 {
		t.Errorf("failed checks = %d, want 3", s.FailedChecks)
	}
}

func TestReport_EmptyResults(t *testing.T) {
	r := NewReport()
	if r.Summary.TotalHosts != 0 {
		t.Error("empty report should have 0 hosts")
	}
	text := r.RenderText()
	if !strings.Contains(text, "0 total") {
		t.Errorf("empty report text should show 0 total; got:\n%s", text)
	}
}

// ---------- Rollback tests ----------

func TestDetectPreviousVersion(t *testing.T) {
	dir := t.TempDir()
	vFile := filepath.Join(dir, "version")
	os.WriteFile(vFile, []byte("v2.1.0\n"), 0o644)

	v, err := dpDetectPreviousVersion(dir)
	if err != nil {
		t.Fatal(err)
	}
	if v != "v2.1.0" {
		t.Errorf("version = %q, want v2.1.0", v)
	}
}

func TestDetectPreviousVersion_Missing(t *testing.T) {
	_, err := dpDetectPreviousVersion(t.TempDir())
	if err == nil {
		t.Error("should error when version file missing")
	}
}

func TestDetectPreviousVersion_Empty(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "version"), []byte("  \n"), 0o644)
	_, err := dpDetectPreviousVersion(dir)
	if err == nil {
		t.Error("should error when version file is empty")
	}
}

func TestCreateRollbackPlan(t *testing.T) {
	cfg := &RollbackConfig{
		BackupDir:       "/backups/v2.0.0",
		PreviousVersion: "v2.0.0",
		Host:            "honey",
	}
	steps, err := dpCreateRollbackPlan(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(steps) < 5 {
		t.Errorf("expected at least 5 steps, got %d", len(steps))
	}
	if !strings.Contains(steps[0], "honey") {
		t.Errorf("first step should mention host: %s", steps[0])
	}
}

func TestCreateRollbackPlan_NilConfig(t *testing.T) {
	_, err := dpCreateRollbackPlan(nil)
	if err == nil {
		t.Error("nil config should error")
	}
}

func TestCreateRollbackPlan_MissingFields(t *testing.T) {
	cases := []struct {
		name string
		cfg  *RollbackConfig
	}{
		{"no backup dir", &RollbackConfig{PreviousVersion: "v1", Host: "h"}},
		{"no version", &RollbackConfig{BackupDir: "/b", Host: "h"}},
		{"no host", &RollbackConfig{BackupDir: "/b", PreviousVersion: "v1"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := dpCreateRollbackPlan(tc.cfg)
			if err == nil {
				t.Error("should error")
			}
		})
	}
}

func TestValidateBackup(t *testing.T) {
	dir := t.TempDir()
	for _, sub := range []string{"binary", "config", "cache"} {
		os.MkdirAll(filepath.Join(dir, sub), 0o755)
	}
	if err := dpValidateBackup(dir); err != nil {
		t.Errorf("valid backup should pass: %v", err)
	}
}

func TestValidateBackup_Missing(t *testing.T) {
	err := dpValidateBackup("/nonexistent/backup")
	if err == nil {
		t.Error("missing backup should error")
	}
}

func TestValidateBackup_IncompleteStructure(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "binary"), 0o755)
	// Missing config and cache.
	err := dpValidateBackup(dir)
	if err == nil {
		t.Error("incomplete backup should error")
	}
	if !strings.Contains(err.Error(), "config") {
		t.Errorf("error should mention missing component: %v", err)
	}
}

func TestGenerateRollbackScript(t *testing.T) {
	cfg := &RollbackConfig{
		BackupDir:       "/backups/v2.0.0",
		PreviousVersion: "v2.0.0",
		Host:            "honey",
	}
	script, err := dpGenerateRollbackScript(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(script, "#!/usr/bin/env bash") {
		t.Error("script should have bash shebang")
	}
	if !strings.Contains(script, "v2.0.0") {
		t.Error("script should contain version")
	}
	if !strings.Contains(script, "honey") {
		t.Error("script should contain hostname")
	}
	if !strings.Contains(script, "set -euo pipefail") {
		t.Error("script should use strict mode")
	}
}

func TestGenerateRollbackScript_NilConfig(t *testing.T) {
	_, err := dpGenerateRollbackScript(nil)
	if err == nil {
		t.Error("nil config should error")
	}
}

// ---------- Health check tests ----------

func TestHealthCheck_AllHealthy(t *testing.T) {
	dir := t.TempDir()
	cacheDir := filepath.Join(dir, "cache")
	os.MkdirAll(filepath.Join(cacheDir, "collectors"), 0o755)

	sockPath := filepath.Join(dir, "sock")
	os.WriteFile(sockPath, nil, 0o600)

	// Fresh collector data.
	now := time.Now()
	colData := `{"updated_at":"` + now.Format(time.RFC3339) + `"}`
	os.WriteFile(filepath.Join(cacheDir, "collectors", "sysmetrics.json"), []byte(colData), 0o644)

	cfg := &HealthConfig{
		SocketPath:    sockPath,
		CacheDir:      cacheDir,
		Collectors:    []string{"sysmetrics"},
		StaleDuration: time.Hour,
		Now:           func() time.Time { return now },
	}

	status, err := dpCheckHealth(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if !status.Healthy {
		for _, c := range status.Components {
			t.Logf("component %s: %s (%s)", c.Name, c.Status, c.Message)
		}
		t.Error("all components should be healthy")
	}
}

func TestHealthCheck_DaemonUnhealthy(t *testing.T) {
	cfg := &HealthConfig{
		SocketPath: "/nonexistent/sock",
		CacheDir:   t.TempDir(),
	}
	status, err := dpCheckHealth(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if status.Healthy {
		t.Error("should be unhealthy when daemon socket missing")
	}
}

func TestHealthCheck_CacheStale(t *testing.T) {
	dir := t.TempDir()
	cacheDir := filepath.Join(dir, "cache")
	os.MkdirAll(cacheDir, 0o755)

	// Set mod time to 48 hours ago.
	past := time.Now().Add(-48 * time.Hour)
	os.Chtimes(cacheDir, past, past)

	sockPath := filepath.Join(dir, "sock")
	os.WriteFile(sockPath, nil, 0o600)

	cfg := &HealthConfig{
		SocketPath:    sockPath,
		CacheDir:      cacheDir,
		StaleDuration: 24 * time.Hour,
	}

	status, err := dpCheckHealth(cfg)
	if err != nil {
		t.Fatal(err)
	}
	// Cache should be degraded but overall might still report based on daemon.
	var cacheComp *ComponentHealth
	for i, c := range status.Components {
		if c.Name == "cache" {
			cacheComp = &status.Components[i]
			break
		}
	}
	if cacheComp == nil {
		t.Fatal("no cache component found")
	}
	if cacheComp.Status != "degraded" {
		t.Errorf("cache status = %q, want degraded", cacheComp.Status)
	}
}

func TestHealthCheck_CollectorOutdated(t *testing.T) {
	dir := t.TempDir()
	cacheDir := filepath.Join(dir, "cache")
	os.MkdirAll(filepath.Join(cacheDir, "collectors"), 0o755)

	sockPath := filepath.Join(dir, "sock")
	os.WriteFile(sockPath, nil, 0o600)

	// Stale collector timestamp.
	now := time.Now()
	staleTime := now.Add(-48 * time.Hour)
	colData := `{"updated_at":"` + staleTime.Format(time.RFC3339) + `"}`
	os.WriteFile(filepath.Join(cacheDir, "collectors", "gpu.json"), []byte(colData), 0o644)

	cfg := &HealthConfig{
		SocketPath:    sockPath,
		CacheDir:      cacheDir,
		Collectors:    []string{"gpu"},
		StaleDuration: time.Hour,
		Now:           func() time.Time { return now },
	}

	status, err := dpCheckHealth(cfg)
	if err != nil {
		t.Fatal(err)
	}
	var colComp *ComponentHealth
	for i, c := range status.Components {
		if c.Name == "collector-gpu" {
			colComp = &status.Components[i]
			break
		}
	}
	if colComp == nil {
		t.Fatal("no gpu collector component found")
	}
	if colComp.Status != "degraded" {
		t.Errorf("collector status = %q, want degraded", colComp.Status)
	}
}

func TestHealthCheck_DiskSpaceMissing(t *testing.T) {
	cfg := &HealthConfig{
		SocketPath: filepath.Join(t.TempDir(), "sock"),
		CacheDir:   "/nonexistent/cache",
	}
	status, err := dpCheckHealth(cfg)
	if err != nil {
		t.Fatal(err)
	}
	var diskComp *ComponentHealth
	for i, c := range status.Components {
		if c.Name == "disk-space" {
			diskComp = &status.Components[i]
			break
		}
	}
	if diskComp == nil {
		t.Fatal("no disk-space component found")
	}
	if diskComp.Status != "unhealthy" {
		t.Errorf("disk space status = %q, want unhealthy", diskComp.Status)
	}
}

func TestHealthCheck_NilConfig(t *testing.T) {
	_, err := dpCheckHealth(nil)
	if err == nil {
		t.Error("nil config should error")
	}
}

// ---------- Edge case tests ----------

func TestVerify_EmptyProfile(t *testing.T) {
	p := &HostProfile{Name: "empty"}
	result, err := Verify(p)
	if err != nil {
		t.Fatal(err)
	}
	// With no binary/config paths it will use defaults that likely don't exist.
	// Should not panic.
	if result.Host != "empty" {
		t.Errorf("host = %q, want empty", result.Host)
	}
}

func TestVerify_NoFeatures(t *testing.T) {
	dir := t.TempDir()
	p := testProfile(t, dir)
	p.Features = nil
	p.Shells = nil
	p.ExpectedCollectors = nil

	result, err := Verify(p)
	if err != nil {
		t.Fatal(err)
	}
	// Should have the base checks but no shell/collector checks.
	for _, c := range result.Checks {
		if strings.HasPrefix(c.Name, "shell-") || strings.HasPrefix(c.Name, "collector-") {
			t.Errorf("should not have %s check with empty shells/collectors", c.Name)
		}
	}
}

func TestVerify_UnknownOS(t *testing.T) {
	p := NewHostProfile("test", "plan9", "mips")
	p.BinaryPath = "/nonexistent/bin"
	result, err := Verify(p)
	if err != nil {
		t.Fatal(err)
	}
	// Should still produce a result without panicking.
	if result.Host != "test" {
		t.Errorf("host = %q, want test", result.Host)
	}
}

func TestCheckResult_Duration(t *testing.T) {
	dir := t.TempDir()
	p := testProfile(t, dir)

	result, err := Verify(p)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range result.Checks {
		if c.Duration < 0 {
			t.Errorf("check %s has negative duration: %v", c.Name, c.Duration)
		}
	}
}

func TestVerify_TimestampSet(t *testing.T) {
	before := time.Now()
	p := &HostProfile{Name: "ts-test"}
	result, err := Verify(p)
	if err != nil {
		t.Fatal(err)
	}
	if result.Timestamp.Before(before) {
		t.Error("timestamp should be after test start")
	}
}

// ---------- End-to-end pipeline test ----------

func TestEndToEnd_VerifyAndReport(t *testing.T) {
	dir := t.TempDir()
	p := testProfile(t, dir)

	// Set up complete environment.
	colDir := filepath.Join(p.CacheDir, "collectors")
	os.WriteFile(filepath.Join(colDir, "sysmetrics.json"), []byte(`{}`), 0o644)
	shellDir := filepath.Join(p.CacheDir, "shells")
	os.WriteFile(filepath.Join(shellDir, "bash.sh"), []byte("# ok"), 0o644)

	result, err := Verify(p)
	if err != nil {
		t.Fatal(err)
	}

	report := NewReport(*result)
	text := report.RenderText()
	if !strings.Contains(text, "test-host") {
		t.Error("report should contain hostname")
	}

	md := report.RenderMarkdown()
	if !strings.Contains(md, "test-host") {
		t.Error("markdown should contain hostname")
	}

	js, err := report.RenderJSON()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(js, "test-host") {
		t.Error("JSON should contain hostname")
	}
}

func TestEndToEnd_FullRolloutValidation(t *testing.T) {
	plan := DefaultRolloutPlan()
	problems := plan.Validate()
	if len(problems) != 0 {
		t.Errorf("default rollout should be valid: %v", problems)
	}

	// Generate a rollback script for each host.
	for _, hr := range plan.Hosts {
		cfg := &RollbackConfig{
			BackupDir:       "/tmp/backup-" + hr.Profile.Name,
			PreviousVersion: "v2.0.0",
			Host:            hr.Profile.Name,
		}
		script, err := dpGenerateRollbackScript(cfg)
		if err != nil {
			t.Errorf("rollback script for %s failed: %v", hr.Profile.Name, err)
			continue
		}
		if !strings.Contains(script, hr.Profile.Name) {
			t.Errorf("script for %s should mention the host", hr.Profile.Name)
		}
	}
}
