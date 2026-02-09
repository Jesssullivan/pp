// Package sysmetrics provides a cross-platform system metrics collector for
// prompt-pulse v2. It uses gopsutil to gather CPU, memory, disk, load, and
// uptime data on both Darwin and Linux without /proc dependencies.
package sysmetrics

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/load"
	"github.com/shirou/gopsutil/v4/mem"
)

// Config controls the SysMetrics collector behaviour.
type Config struct {
	// FastInterval is the polling rate for CPU and RAM (default 2s).
	FastInterval time.Duration

	// SlowInterval is the polling rate for disk enumeration (default 60s).
	SlowInterval time.Duration

	// MonitoredMounts restricts disk collection to these mount paths.
	// An empty slice means "collect all non-virtual partitions".
	MonitoredMounts []string
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		FastInterval: 2 * time.Second,
		SlowInterval: 60 * time.Second,
	}
}

// --- Metric data types ---

// CPUMetrics holds per-core and aggregate CPU utilisation.
type CPUMetrics struct {
	// Cores contains per-core usage percentages (0-100).
	Cores []float64 `json:"cores"`

	// Total is the overall CPU usage percentage (0-100).
	Total float64 `json:"total"`

	// Count is the number of logical CPUs.
	Count int `json:"count"`
}

// MemoryMetrics holds physical and swap memory statistics.
type MemoryMetrics struct {
	Total            uint64  `json:"total"`
	Used             uint64  `json:"used"`
	Available        uint64  `json:"available"`
	SwapTotal        uint64  `json:"swap_total"`
	SwapUsed         uint64  `json:"swap_used"`
	UsedPercent      float64 `json:"used_percent"`
	SwapUsedPercent  float64 `json:"swap_used_percent"`
}

// DiskMetrics holds usage data for a single mount point.
type DiskMetrics struct {
	Path        string  `json:"path"`
	FSType      string  `json:"fstype"`
	Total       uint64  `json:"total"`
	Used        uint64  `json:"used"`
	Free        uint64  `json:"free"`
	UsedPercent float64 `json:"used_percent"`
}

// LoadMetrics holds system load averages.
type LoadMetrics struct {
	Load1  float64 `json:"load1"`
	Load5  float64 `json:"load5"`
	Load15 float64 `json:"load15"`
}

// Metrics is the aggregate snapshot returned by Collect.
type Metrics struct {
	CPU       CPUMetrics    `json:"cpu"`
	Memory    MemoryMetrics `json:"memory"`
	Disks     []DiskMetrics `json:"disks"`
	Load      LoadMetrics   `json:"load"`
	Uptime    time.Duration `json:"uptime"`
	Timestamp time.Time     `json:"timestamp"`
}

// --- Collector implementation ---

// Collector gathers system metrics via gopsutil. It satisfies the
// pkg/collectors.Collector interface (Name, Collect, Interval, Healthy).
type Collector struct {
	cfg     Config
	mu      sync.Mutex
	healthy bool
}

// New creates a Collector with the given configuration. Zero-value fields
// in cfg are replaced with defaults.
func New(cfg Config) *Collector {
	if cfg.FastInterval <= 0 {
		cfg.FastInterval = DefaultConfig().FastInterval
	}
	if cfg.SlowInterval <= 0 {
		cfg.SlowInterval = DefaultConfig().SlowInterval
	}
	return &Collector{
		cfg:     cfg,
		healthy: true, // healthy until proven otherwise
	}
}

// Name returns the collector's unique identifier.
func (c *Collector) Name() string {
	return "sysmetrics"
}

// Interval returns the fast polling interval (CPU/RAM cadence).
func (c *Collector) Interval() time.Duration {
	return c.cfg.FastInterval
}

// Healthy reports whether the last collection succeeded.
func (c *Collector) Healthy() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.healthy
}

// setHealthy updates the health flag in a thread-safe manner.
func (c *Collector) setHealthy(h bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.healthy = h
}

// Collect gathers all system metrics. If individual sub-collectors fail the
// method still returns as much data as possible; errors are aggregated. A
// fully cancelled context returns immediately with an error.
func (c *Collector) Collect(ctx context.Context) (interface{}, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	m := Metrics{
		Timestamp: time.Now(),
	}

	var errs []string

	// --- CPU ---
	if err := c.collectCPU(ctx, &m); err != nil {
		errs = append(errs, fmt.Sprintf("cpu: %v", err))
	}

	// --- Memory ---
	if err := c.collectMemory(ctx, &m); err != nil {
		errs = append(errs, fmt.Sprintf("memory: %v", err))
	}

	// --- Disk ---
	if err := c.collectDisk(ctx, &m); err != nil {
		errs = append(errs, fmt.Sprintf("disk: %v", err))
	}

	// --- Load ---
	if err := c.collectLoad(ctx, &m); err != nil {
		errs = append(errs, fmt.Sprintf("load: %v", err))
	}

	// --- Uptime ---
	if err := c.collectUptime(ctx, &m); err != nil {
		errs = append(errs, fmt.Sprintf("uptime: %v", err))
	}

	// If everything failed, report unhealthy and return an aggregated error.
	if len(errs) == 5 {
		c.setHealthy(false)
		return nil, fmt.Errorf("sysmetrics: all sub-collectors failed: %s", strings.Join(errs, "; "))
	}

	// Partial failures are not fatal; the collector is still healthy as long
	// as at least one sub-collector produced data.
	c.setHealthy(true)

	if len(errs) > 0 {
		return m, fmt.Errorf("sysmetrics: partial errors: %s", strings.Join(errs, "; "))
	}

	return m, nil
}

// --- sub-collectors ---

func (c *Collector) collectCPU(ctx context.Context, m *Metrics) error {
	// Per-core percentages (interval=0 means instantaneous snapshot).
	perCore, err := cpu.PercentWithContext(ctx, 0, true)
	if err != nil {
		return err
	}

	// Total (all cores aggregated).
	total, err := cpu.PercentWithContext(ctx, 0, false)
	if err != nil {
		return err
	}

	m.CPU.Cores = perCore
	m.CPU.Count = len(perCore)
	if len(total) > 0 {
		m.CPU.Total = total[0]
	}
	return nil
}

func (c *Collector) collectMemory(ctx context.Context, m *Metrics) error {
	vm, err := mem.VirtualMemoryWithContext(ctx)
	if err != nil {
		return err
	}
	m.Memory.Total = vm.Total
	m.Memory.Used = vm.Used
	m.Memory.Available = vm.Available
	m.Memory.UsedPercent = vm.UsedPercent

	sw, err := mem.SwapMemoryWithContext(ctx)
	if err != nil {
		// Swap might not be available; treat as non-fatal within memory.
		m.Memory.SwapTotal = 0
		m.Memory.SwapUsed = 0
		m.Memory.SwapUsedPercent = 0
		return nil
	}
	m.Memory.SwapTotal = sw.Total
	m.Memory.SwapUsed = sw.Used
	if sw.Total > 0 {
		m.Memory.SwapUsedPercent = sw.UsedPercent
	}
	return nil
}

func (c *Collector) collectDisk(ctx context.Context, m *Metrics) error {
	// If specific mounts were requested, collect only those.
	if len(c.cfg.MonitoredMounts) > 0 {
		for _, mp := range c.cfg.MonitoredMounts {
			usage, err := disk.UsageWithContext(ctx, mp)
			if err != nil {
				continue // skip mounts that fail
			}
			m.Disks = append(m.Disks, DiskMetrics{
				Path:        usage.Path,
				FSType:      usage.Fstype,
				Total:       usage.Total,
				Used:        usage.Used,
				Free:        usage.Free,
				UsedPercent: usage.UsedPercent,
			})
		}
		return nil
	}

	// Otherwise enumerate all real partitions.
	parts, err := disk.PartitionsWithContext(ctx, false)
	if err != nil {
		return err
	}

	for _, p := range parts {
		if isVirtualFS(p.Fstype) {
			continue
		}
		usage, err := disk.UsageWithContext(ctx, p.Mountpoint)
		if err != nil {
			continue // skip partitions that fail
		}
		m.Disks = append(m.Disks, DiskMetrics{
			Path:        usage.Path,
			FSType:      usage.Fstype,
			Total:       usage.Total,
			Used:        usage.Used,
			Free:        usage.Free,
			UsedPercent: usage.UsedPercent,
		})
	}
	return nil
}

func (c *Collector) collectLoad(ctx context.Context, m *Metrics) error {
	avg, err := load.AvgWithContext(ctx)
	if err != nil {
		return err
	}
	m.Load.Load1 = avg.Load1
	m.Load.Load5 = avg.Load5
	m.Load.Load15 = avg.Load15
	return nil
}

func (c *Collector) collectUptime(ctx context.Context, m *Metrics) error {
	secs, err := host.UptimeWithContext(ctx)
	if err != nil {
		return err
	}
	m.Uptime = time.Duration(secs) * time.Second
	return nil
}

// isVirtualFS returns true for filesystem types that do not represent real
// storage and should be skipped during enumeration.
func isVirtualFS(fstype string) bool {
	switch fstype {
	case "devfs", "devtmpfs", "tmpfs", "sysfs", "proc", "cgroup", "cgroup2",
		"autofs", "mqueue", "hugetlbfs", "debugfs", "tracefs", "securityfs",
		"pstore", "bpf", "fusectl", "configfs", "ramfs", "rpc_pipefs",
		"nfsd", "map", "devpts", "squashfs":
		return true
	}
	return false
}
