package widgets

import (
	"fmt"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"gitlab.com/tinyland/lab/prompt-pulse/pkg/app"
	"gitlab.com/tinyland/lab/prompt-pulse/pkg/collectors/sysmetrics"
)

// --- helpers ---

// smTestMetrics returns a fully populated Metrics struct for testing.
func smTestMetrics() sysmetrics.Metrics {
	return sysmetrics.Metrics{
		CPU: sysmetrics.CPUMetrics{
			Cores: []float64{45.0, 62.0, 30.0, 88.0},
			Total: 56.25,
			Count: 4,
		},
		Memory: sysmetrics.MemoryMetrics{
			Total:           16 * 1024 * 1024 * 1024, // 16 GB
			Used:            8200 * 1024 * 1024,       // ~8.2 GB
			Available:       7800 * 1024 * 1024,
			SwapTotal:       8 * 1024 * 1024 * 1024, // 8 GB
			SwapUsed:        1200 * 1024 * 1024,     // ~1.2 GB
			UsedPercent:     51.25,
			SwapUsedPercent: 14.65,
		},
		Disks: []sysmetrics.DiskMetrics{
			{
				Path:        "/",
				FSType:      "apfs",
				Total:       500 * 1024 * 1024 * 1024,
				Used:        450 * 1024 * 1024 * 1024,
				Free:        50 * 1024 * 1024 * 1024,
				UsedPercent: 90.0,
			},
			{
				Path:        "/data",
				FSType:      "ext4",
				Total:       1000 * 1024 * 1024 * 1024,
				Used:        300 * 1024 * 1024 * 1024,
				Free:        700 * 1024 * 1024 * 1024,
				UsedPercent: 30.0,
			},
		},
		Load: sysmetrics.LoadMetrics{
			Load1:  1.23,
			Load5:  0.98,
			Load15: 0.76,
		},
		Uptime:    14*24*time.Hour + 6*time.Hour + 23*time.Minute,
		Timestamp: time.Now(),
	}
}

// smTestMetricsNoSwap returns metrics with zero swap configured.
func smTestMetricsNoSwap() sysmetrics.Metrics {
	m := smTestMetrics()
	m.Memory.SwapTotal = 0
	m.Memory.SwapUsed = 0
	m.Memory.SwapUsedPercent = 0
	return m
}

// smTestMetricsNoDisks returns metrics with no disk mounts.
func smTestMetricsNoDisks() sysmetrics.Metrics {
	m := smTestMetrics()
	m.Disks = nil
	return m
}

// --- tests ---

func TestSysMetricsWidgetID(t *testing.T) {
	w := NewSysMetricsWidget()
	if got := w.ID(); got != "sysmetrics" {
		t.Errorf("ID() = %q, want %q", got, "sysmetrics")
	}
}

func TestSysMetricsWidgetTitle(t *testing.T) {
	w := NewSysMetricsWidget()
	if got := w.Title(); got != "System" {
		t.Errorf("Title() = %q, want %q", got, "System")
	}
}

func TestSysMetricsWidgetMinSize(t *testing.T) {
	w := NewSysMetricsWidget()
	minW, minH := w.MinSize()
	if minW != 25 {
		t.Errorf("MinSize() width = %d, want 25", minW)
	}
	if minH != 4 {
		t.Errorf("MinSize() height = %d, want 4", minH)
	}
}

func TestSysMetricsWidgetViewNoData(t *testing.T) {
	w := NewSysMetricsWidget()
	view := w.View(40, 5)
	if !strings.Contains(view, "No data") {
		t.Errorf("View with no data should contain 'No data', got:\n%s", view)
	}
}

func TestSysMetricsWidgetViewZeroDimensions(t *testing.T) {
	w := NewSysMetricsWidget()
	if got := w.View(0, 10); got != "" {
		t.Errorf("View(0, 10) should return empty string, got: %q", got)
	}
	if got := w.View(10, 0); got != "" {
		t.Errorf("View(10, 0) should return empty string, got: %q", got)
	}
}

func TestSysMetricsWidgetUpdateWithMetrics(t *testing.T) {
	w := NewSysMetricsWidget()
	m := smTestMetrics()

	cmd := w.Update(app.DataUpdateEvent{
		Source: "sysmetrics",
		Data:   m,
	})

	if cmd != nil {
		t.Error("Update should return nil cmd")
	}
	if w.metrics == nil {
		t.Fatal("metrics should be set after Update")
	}
	if w.metrics.CPU.Total != 56.25 {
		t.Errorf("CPU.Total = %f, want 56.25", w.metrics.CPU.Total)
	}
	if len(w.cpuHistory) != 1 {
		t.Errorf("cpuHistory length = %d, want 1", len(w.cpuHistory))
	}
	if len(w.loadHistory) != 1 {
		t.Errorf("loadHistory length = %d, want 1", len(w.loadHistory))
	}
}

func TestSysMetricsWidgetUpdateWithPointerMetrics(t *testing.T) {
	w := NewSysMetricsWidget()
	m := smTestMetrics()

	w.Update(app.DataUpdateEvent{
		Source: "sysmetrics",
		Data:   &m,
	})

	if w.metrics == nil {
		t.Fatal("metrics should be set after Update with pointer")
	}
	if w.metrics.CPU.Total != 56.25 {
		t.Errorf("CPU.Total = %f, want 56.25", w.metrics.CPU.Total)
	}
}

func TestSysMetricsWidgetUpdateIgnoresOtherSources(t *testing.T) {
	w := NewSysMetricsWidget()
	m := smTestMetrics()

	w.Update(app.DataUpdateEvent{
		Source: "billing",
		Data:   m,
	})

	if w.metrics != nil {
		t.Error("metrics should be nil after Update from wrong source")
	}
}

func TestSysMetricsWidgetUpdateIgnoresErrors(t *testing.T) {
	w := NewSysMetricsWidget()

	w.Update(app.DataUpdateEvent{
		Source: "sysmetrics",
		Err:    fmt.Errorf("test error"),
	})

	if w.metrics != nil {
		t.Error("metrics should be nil after error Update")
	}
}

func TestSysMetricsWidgetHandleKeyE(t *testing.T) {
	w := NewSysMetricsWidget()

	if w.expanded {
		t.Error("expanded should start false")
	}

	w.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})

	if !w.expanded {
		t.Error("expanded should be true after pressing 'e'")
	}

	w.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})

	if w.expanded {
		t.Error("expanded should be false after pressing 'e' again")
	}
}

func TestSysMetricsWidgetHandleKeyC(t *testing.T) {
	w := NewSysMetricsWidget()

	if w.perCore {
		t.Error("perCore should start false")
	}

	w.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})

	if !w.perCore {
		t.Error("perCore should be true after pressing 'c'")
	}

	w.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})

	if w.perCore {
		t.Error("perCore should be false after pressing 'c' again")
	}
}

func TestSysMetricsWidgetHandleKeyUnknown(t *testing.T) {
	w := NewSysMetricsWidget()

	cmd := w.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'z'}})

	if cmd != nil {
		t.Error("HandleKey should return nil for unknown keys")
	}
}

func TestSysMetricsWidgetViewCompact(t *testing.T) {
	w := NewSysMetricsWidget()
	m := smTestMetrics()
	w.metrics = &m

	view := w.View(60, 15)
	lines := strings.Split(view, "\n")

	// Should have exactly 15 lines (height).
	if len(lines) != 15 {
		t.Errorf("compact view should have 15 lines, got %d", len(lines))
	}

	// Should contain CPU.
	found := false
	for _, line := range lines {
		if strings.Contains(line, "CPU") {
			found = true
			break
		}
	}
	if !found {
		t.Error("compact view should contain CPU gauge")
	}

	// Should contain RAM.
	found = false
	for _, line := range lines {
		if strings.Contains(line, "RAM") {
			found = true
			break
		}
	}
	if !found {
		t.Error("compact view should contain RAM gauge")
	}

	// Should contain Load.
	found = false
	for _, line := range lines {
		if strings.Contains(line, "Load") {
			found = true
			break
		}
	}
	if !found {
		t.Error("compact view should contain Load values")
	}

	// Should contain Uptime.
	found = false
	for _, line := range lines {
		if strings.Contains(line, "Uptime") {
			found = true
			break
		}
	}
	if !found {
		t.Error("compact view should contain Uptime")
	}
}

func TestSysMetricsWidgetViewExpanded(t *testing.T) {
	w := NewSysMetricsWidget()
	m := smTestMetrics()
	w.metrics = &m
	w.expanded = true

	view := w.View(60, 24)
	lines := strings.Split(view, "\n")

	// Should have exactly 24 lines (height).
	if len(lines) != 24 {
		t.Errorf("expanded view should have 24 lines, got %d", len(lines))
	}

	// Should contain Memory section.
	found := false
	for _, line := range lines {
		if strings.Contains(line, "Memory") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expanded view should contain Memory section header")
	}

	// Should contain Disk section.
	found = false
	for _, line := range lines {
		if strings.Contains(line, "Disk") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expanded view should contain Disk section header")
	}

	// Should contain Swap gauge since SwapTotal > 0.
	found = false
	for _, line := range lines {
		if strings.Contains(line, "Swap") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expanded view should contain Swap gauge when swap is configured")
	}

	// Should contain both disk mounts.
	foundRoot := false
	foundData := false
	for _, line := range lines {
		if strings.Contains(line, "/data") {
			foundData = true
		}
		// Check for "/" but make sure it is the root mount label, not
		// just part of another path. The root mount gauge starts with "/".
		if strings.HasPrefix(strings.TrimSpace(line), "/") && !strings.Contains(line, "/data") {
			foundRoot = true
		}
	}
	if !foundRoot {
		t.Error("expanded view should contain / disk mount")
	}
	if !foundData {
		t.Error("expanded view should contain /data disk mount")
	}
}

func TestSysMetricsWidgetViewExpandedPerCore(t *testing.T) {
	w := NewSysMetricsWidget()
	m := smTestMetrics()
	w.metrics = &m
	w.expanded = true
	w.perCore = true

	view := w.View(60, 30)

	// Should contain Core labels.
	for i := 0; i < 4; i++ {
		coreLabel := fmt.Sprintf("Core %d", i)
		if !strings.Contains(view, coreLabel) {
			t.Errorf("per-core expanded view should contain %q", coreLabel)
		}
	}
}

func TestSysMetricsWidgetNoSwap(t *testing.T) {
	w := NewSysMetricsWidget()
	m := smTestMetricsNoSwap()
	w.metrics = &m
	w.expanded = true

	view := w.View(60, 20)

	// Should NOT contain Swap gauge line.
	for _, line := range strings.Split(view, "\n") {
		if strings.Contains(line, "Swap") {
			t.Error("expanded view should not contain Swap gauge when swap is 0")
		}
	}
}

func TestSysMetricsWidgetNoDisks(t *testing.T) {
	w := NewSysMetricsWidget()
	m := smTestMetricsNoDisks()
	w.metrics = &m

	// Compact mode.
	view := w.View(40, 8)
	if !strings.Contains(view, "No disks") {
		t.Error("compact view with no disks should contain 'No disks'")
	}

	// Expanded mode.
	w.expanded = true
	view = w.View(40, 15)
	if !strings.Contains(view, "No disks") {
		t.Error("expanded view with no disks should contain 'No disks'")
	}
}

func TestSysmetricsFormatBytes(t *testing.T) {
	tests := []struct {
		bytes uint64
		want  string
	}{
		{0, "0 B"},
		{500, "500 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{1572864, "1.5 MB"},
		{1073741824, "1.0 GB"},
		{1610612736, "1.5 GB"},
		{1099511627776, "1.0 TB"},
		{1649267441664, "1.5 TB"},
	}

	for _, tt := range tests {
		got := smFormatBytes(tt.bytes)
		if got != tt.want {
			t.Errorf("smFormatBytes(%d) = %q, want %q", tt.bytes, got, tt.want)
		}
	}
}

func TestSysmetricsFormatUptime(t *testing.T) {
	tests := []struct {
		duration time.Duration
		want     string
	}{
		{0, "0m"},
		{-5 * time.Minute, "0m"},
		{45 * time.Minute, "45m"},
		{2*time.Hour + 15*time.Minute, "2h 15m"},
		{14*24*time.Hour + 6*time.Hour + 23*time.Minute, "14d 6h 23m"},
		{1*24*time.Hour + 0*time.Hour + 0*time.Minute, "1d 0h 0m"},
		{time.Hour, "1h 0m"},
	}

	for _, tt := range tests {
		got := smFormatUptime(tt.duration)
		if got != tt.want {
			t.Errorf("smFormatUptime(%v) = %q, want %q", tt.duration, got, tt.want)
		}
	}
}

func TestSysmetricsFormatPercent(t *testing.T) {
	tests := []struct {
		pct  float64
		want string
	}{
		{0.0, "0%"},
		{73.0, "73%"},
		{73.4, "73%"},
		{73.5, "74%"},
		{100.0, "100%"},
	}

	for _, tt := range tests {
		got := smFormatPercent(tt.pct)
		if got != tt.want {
			t.Errorf("smFormatPercent(%f) = %q, want %q", tt.pct, got, tt.want)
		}
	}
}

func TestSysMetricsWidgetViewMinimumSize(t *testing.T) {
	w := NewSysMetricsWidget()
	m := smTestMetrics()
	w.metrics = &m

	// 25x4 is the minimum size.
	view := w.View(25, 4)
	lines := strings.Split(view, "\n")
	if len(lines) != 4 {
		t.Errorf("View(25,4) should produce 4 lines, got %d", len(lines))
	}
}

func TestSysMetricsWidgetViewMediumSize(t *testing.T) {
	w := NewSysMetricsWidget()
	m := smTestMetrics()
	w.metrics = &m

	view := w.View(60, 15)
	lines := strings.Split(view, "\n")
	if len(lines) != 15 {
		t.Errorf("View(60,15) should produce 15 lines, got %d", len(lines))
	}
}

func TestSysMetricsWidgetViewLargeSize(t *testing.T) {
	w := NewSysMetricsWidget()
	m := smTestMetrics()
	w.metrics = &m
	w.expanded = true

	view := w.View(80, 24)
	lines := strings.Split(view, "\n")
	if len(lines) != 24 {
		t.Errorf("View(80,24) should produce 24 lines, got %d", len(lines))
	}
}

func TestSysMetricsWidgetDiskGaugeColorHighUsage(t *testing.T) {
	// Disk at 90% should be red (>= 85% critical threshold).
	color := smDiskColor(90.0)
	if color != smColorRed {
		t.Errorf("smDiskColor(90.0) = %q, want %q (red)", color, smColorRed)
	}

	// Disk at 75% should be yellow (>= 70% warning threshold).
	color = smDiskColor(75.0)
	if color != smColorYellow {
		t.Errorf("smDiskColor(75.0) = %q, want %q (yellow)", color, smColorYellow)
	}

	// Disk at 50% should be green (< 70%).
	color = smDiskColor(50.0)
	if color != smColorGreen {
		t.Errorf("smDiskColor(50.0) = %q, want %q (green)", color, smColorGreen)
	}
}

func TestSysMetricsWidgetCPUHistoryRolling(t *testing.T) {
	w := NewSysMetricsWidget()

	// Fill beyond the max history limit.
	for i := 0; i < smMaxHistory+10; i++ {
		m := sysmetrics.Metrics{
			CPU: sysmetrics.CPUMetrics{Total: float64(i)},
		}
		w.Update(app.DataUpdateEvent{
			Source: "sysmetrics",
			Data:   m,
		})
	}

	if len(w.cpuHistory) != smMaxHistory {
		t.Errorf("cpuHistory length = %d, want %d", len(w.cpuHistory), smMaxHistory)
	}

	// The oldest value should be 10 (smMaxHistory+10 - smMaxHistory).
	if w.cpuHistory[0] != 10.0 {
		t.Errorf("cpuHistory[0] = %f, want 10.0", w.cpuHistory[0])
	}
}

func TestSysMetricsWidgetCompileTimeInterface(t *testing.T) {
	// Verify the compile-time interface assertion is in place.
	var _ app.Widget = (*SysMetricsWidget)(nil)
}

