package sysmetrics

import (
	"context"
	"testing"
	"time"
)

// --- Interface method tests ---

func TestName(t *testing.T) {
	c := New(DefaultConfig())
	if got := c.Name(); got != "sysmetrics" {
		t.Errorf("Name() = %q, want %q", got, "sysmetrics")
	}
}

func TestIntervalDefault(t *testing.T) {
	c := New(Config{})
	if got := c.Interval(); got != 2*time.Second {
		t.Errorf("Interval() with zero config = %v, want 2s", got)
	}
}

func TestIntervalCustom(t *testing.T) {
	c := New(Config{FastInterval: 5 * time.Second})
	if got := c.Interval(); got != 5*time.Second {
		t.Errorf("Interval() = %v, want 5s", got)
	}
}

func TestHealthyInitialState(t *testing.T) {
	c := New(DefaultConfig())
	if !c.Healthy() {
		t.Error("Healthy() should be true before any collection")
	}
}

func TestDefaultConfigValues(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.FastInterval != 2*time.Second {
		t.Errorf("DefaultConfig FastInterval = %v, want 2s", cfg.FastInterval)
	}
	if cfg.SlowInterval != 60*time.Second {
		t.Errorf("DefaultConfig SlowInterval = %v, want 60s", cfg.SlowInterval)
	}
	if len(cfg.MonitoredMounts) != 0 {
		t.Errorf("DefaultConfig MonitoredMounts = %v, want empty", cfg.MonitoredMounts)
	}
}

// --- Integration tests (run on actual host) ---

func TestCollectReturnsValidMetrics(t *testing.T) {
	c := New(DefaultConfig())
	result, err := c.Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect() error: %v", err)
	}

	m, ok := result.(Metrics)
	if !ok {
		t.Fatalf("Collect() returned %T, want Metrics", result)
	}

	// CPU cores must be present.
	if m.CPU.Count <= 0 {
		t.Errorf("CPU.Count = %d, want > 0", m.CPU.Count)
	}
	if len(m.CPU.Cores) != m.CPU.Count {
		t.Errorf("len(CPU.Cores) = %d, want %d", len(m.CPU.Cores), m.CPU.Count)
	}

	// CPU total must be in valid range.
	if m.CPU.Total < 0 || m.CPU.Total > 100 {
		t.Errorf("CPU.Total = %f, want 0-100", m.CPU.Total)
	}

	// Per-core values must be in valid range.
	for i, pct := range m.CPU.Cores {
		if pct < 0 || pct > 100 {
			t.Errorf("CPU.Cores[%d] = %f, want 0-100", i, pct)
		}
	}

	// Timestamp must be recent.
	if time.Since(m.Timestamp) > 5*time.Second {
		t.Errorf("Timestamp is too old: %v", m.Timestamp)
	}
}

func TestCollectMemoryValid(t *testing.T) {
	c := New(DefaultConfig())
	result, err := c.Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect() error: %v", err)
	}

	m := result.(Metrics)

	if m.Memory.Total == 0 {
		t.Error("Memory.Total should be > 0")
	}
	if m.Memory.Used > m.Memory.Total {
		t.Errorf("Memory.Used (%d) > Memory.Total (%d)", m.Memory.Used, m.Memory.Total)
	}
	if m.Memory.UsedPercent < 0 || m.Memory.UsedPercent > 100 {
		t.Errorf("Memory.UsedPercent = %f, want 0-100", m.Memory.UsedPercent)
	}
}

func TestCollectDiskValid(t *testing.T) {
	c := New(DefaultConfig())
	result, err := c.Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect() error: %v", err)
	}

	m := result.(Metrics)

	if len(m.Disks) == 0 {
		t.Fatal("expected at least one disk")
	}

	for i, d := range m.Disks {
		if d.Path == "" {
			t.Errorf("Disks[%d].Path is empty", i)
		}
		if d.Total == 0 {
			t.Errorf("Disks[%d].Total is 0", i)
		}
		if d.Used > d.Total {
			t.Errorf("Disks[%d].Used (%d) > Total (%d)", i, d.Used, d.Total)
		}
		if d.UsedPercent < 0 || d.UsedPercent > 100 {
			t.Errorf("Disks[%d].UsedPercent = %f, want 0-100", i, d.UsedPercent)
		}
	}
}

func TestCollectLoadValid(t *testing.T) {
	c := New(DefaultConfig())
	result, err := c.Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect() error: %v", err)
	}

	m := result.(Metrics)

	if m.Load.Load1 < 0 {
		t.Errorf("Load.Load1 = %f, want >= 0", m.Load.Load1)
	}
	if m.Load.Load5 < 0 {
		t.Errorf("Load.Load5 = %f, want >= 0", m.Load.Load5)
	}
	if m.Load.Load15 < 0 {
		t.Errorf("Load.Load15 = %f, want >= 0", m.Load.Load15)
	}
}

func TestCollectUptimePositive(t *testing.T) {
	c := New(DefaultConfig())
	result, err := c.Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect() error: %v", err)
	}

	m := result.(Metrics)

	if m.Uptime <= 0 {
		t.Errorf("Uptime = %v, want > 0", m.Uptime)
	}
}

func TestHealthyAfterSuccessfulCollect(t *testing.T) {
	c := New(DefaultConfig())
	_, err := c.Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect() error: %v", err)
	}

	if !c.Healthy() {
		t.Error("Healthy() should be true after successful collect")
	}
}

func TestCollectWithCancelledContext(t *testing.T) {
	c := New(DefaultConfig())

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	_, err := c.Collect(ctx)
	if err == nil {
		t.Error("Collect with cancelled context should return error")
	}
}

func TestMountFilteringRootOnly(t *testing.T) {
	c := New(Config{
		MonitoredMounts: []string{"/"},
	})

	result, err := c.Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect() error: %v", err)
	}

	m := result.(Metrics)

	if len(m.Disks) != 1 {
		t.Fatalf("expected 1 disk with MonitoredMounts=[\"/\"], got %d", len(m.Disks))
	}
	if m.Disks[0].Path != "/" {
		t.Errorf("Disks[0].Path = %q, want %q", m.Disks[0].Path, "/")
	}
}

func TestEmptyMonitoredMountsReturnsAll(t *testing.T) {
	c := New(Config{
		MonitoredMounts: nil,
	})

	result, err := c.Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect() error: %v", err)
	}

	m := result.(Metrics)

	if len(m.Disks) == 0 {
		t.Error("expected at least one disk with empty MonitoredMounts")
	}
}

func TestMountFilteringBogusMount(t *testing.T) {
	c := New(Config{
		MonitoredMounts: []string{"/nonexistent-mount-path-12345"},
	})

	result, err := c.Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect() error: %v", err)
	}

	m := result.(Metrics)

	// Bogus mount should be silently skipped.
	for _, d := range m.Disks {
		if d.Path == "/nonexistent-mount-path-12345" {
			t.Error("bogus mount should not appear in results")
		}
	}
}

func TestCollectMetricsStruct(t *testing.T) {
	c := New(DefaultConfig())
	result, err := c.Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect() error: %v", err)
	}

	m := result.(Metrics)

	// Verify the struct has valid shapes: CPU cores > 0, memory used <= total,
	// disk used <= total.
	if m.CPU.Count == 0 {
		t.Error("CPU.Count must be > 0")
	}
	if m.Memory.Total == 0 {
		t.Error("Memory.Total must be > 0")
	}
	if m.Memory.Used > m.Memory.Total {
		t.Errorf("Memory.Used (%d) must be <= Memory.Total (%d)", m.Memory.Used, m.Memory.Total)
	}
	for i, d := range m.Disks {
		if d.Used > d.Total {
			t.Errorf("Disk[%d].Used (%d) must be <= Total (%d)", i, d.Used, d.Total)
		}
	}
}

func TestNewWithZeroConfig(t *testing.T) {
	c := New(Config{})
	if c.cfg.FastInterval != 2*time.Second {
		t.Errorf("zero FastInterval should default to 2s, got %v", c.cfg.FastInterval)
	}
	if c.cfg.SlowInterval != 60*time.Second {
		t.Errorf("zero SlowInterval should default to 60s, got %v", c.cfg.SlowInterval)
	}
}

// --- Virtual FS filtering ---

func TestIsVirtualFS(t *testing.T) {
	virtuals := []string{"devfs", "tmpfs", "sysfs", "proc", "devtmpfs", "autofs", "map"}
	for _, fs := range virtuals {
		if !isVirtualFS(fs) {
			t.Errorf("isVirtualFS(%q) = false, want true", fs)
		}
	}

	reals := []string{"apfs", "ext4", "xfs", "hfs", "btrfs", "zfs", "ntfs"}
	for _, fs := range reals {
		if isVirtualFS(fs) {
			t.Errorf("isVirtualFS(%q) = true, want false", fs)
		}
	}
}

// --- Concurrency safety ---

func TestHealthyConcurrency(t *testing.T) {
	c := New(DefaultConfig())

	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 100; i++ {
			c.setHealthy(i%2 == 0)
		}
	}()

	for i := 0; i < 100; i++ {
		_ = c.Healthy()
	}
	<-done
}
