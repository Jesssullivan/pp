package widgets

import (
	"fmt"
	"math"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"gitlab.com/tinyland/lab/prompt-pulse/pkg/app"
	"gitlab.com/tinyland/lab/prompt-pulse/pkg/collectors/sysmetrics"
	"gitlab.com/tinyland/lab/prompt-pulse/pkg/components"
)

// System metrics color constants.
const (
	smColorGreen  = "#4CAF50"
	smColorYellow = "#FF9800"
	smColorRed    = "#F44336"
	smColorBlue   = "#64B5F6"

	// CPU thresholds (percentage 0-100).
	smCPUWarnThreshold = 50.0
	smCPUCritThreshold = 80.0

	// Memory thresholds (percentage 0-100).
	smMemWarnThreshold = 50.0
	smMemCritThreshold = 80.0

	// Disk thresholds (percentage 0-100).
	smDiskWarnThreshold = 70.0
	smDiskCritThreshold = 85.0

	// Maximum history length for sparkline rolling buffers.
	smMaxHistory = 60
)

// SysMetricsWidget displays system metrics including CPU, memory, disk,
// load averages, and uptime. It supports compact and expanded display modes.
type SysMetricsWidget struct {
	metrics     *sysmetrics.Metrics
	expanded    bool
	perCore     bool
	cpuHistory  []float64
	loadHistory []float64
}

// NewSysMetricsWidget creates a new SysMetricsWidget in compact mode.
func NewSysMetricsWidget() *SysMetricsWidget {
	return &SysMetricsWidget{}
}

// ID returns the unique identifier for this widget.
func (w *SysMetricsWidget) ID() string {
	return "sysmetrics"
}

// Title returns the display name for this widget.
func (w *SysMetricsWidget) Title() string {
	return "System"
}

// MinSize returns the minimum width and height this widget requires.
func (w *SysMetricsWidget) MinSize() (int, int) {
	return 25, 4
}

// Update handles messages directed at this widget. It processes
// DataUpdateEvent messages with Source "sysmetrics".
func (w *SysMetricsWidget) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case app.DataUpdateEvent:
		if msg.Source != "sysmetrics" {
			return nil
		}
		if msg.Err != nil {
			return nil
		}
		m, ok := msg.Data.(sysmetrics.Metrics)
		if !ok {
			// Also accept pointer form.
			mp, okp := msg.Data.(*sysmetrics.Metrics)
			if !okp || mp == nil {
				return nil
			}
			m = *mp
		}
		w.metrics = &m

		// Append CPU total to history.
		w.cpuHistory = append(w.cpuHistory, m.CPU.Total)
		if len(w.cpuHistory) > smMaxHistory {
			w.cpuHistory = w.cpuHistory[len(w.cpuHistory)-smMaxHistory:]
		}

		// Append load1 to history.
		w.loadHistory = append(w.loadHistory, m.Load.Load1)
		if len(w.loadHistory) > smMaxHistory {
			w.loadHistory = w.loadHistory[len(w.loadHistory)-smMaxHistory:]
		}
	}
	return nil
}

// HandleKey processes key events when this widget has focus.
// 'e' toggles expanded mode, 'c' toggles per-core CPU view.
func (w *SysMetricsWidget) HandleKey(key tea.KeyMsg) tea.Cmd {
	switch key.String() {
	case "e":
		w.expanded = !w.expanded
		return nil
	case "c":
		w.perCore = !w.perCore
		return nil
	}
	return nil
}

// View renders the widget content into the given area dimensions.
func (w *SysMetricsWidget) View(width, height int) string {
	if width <= 0 || height <= 0 {
		return ""
	}

	if w.metrics == nil {
		return smCenterMessage("No data", width, height)
	}

	var lines []string
	if w.expanded {
		lines = w.smViewExpanded(width)
	} else {
		lines = w.smViewCompact(width)
	}

	return smFitLines(lines, width, height)
}

// smViewCompact renders the compact view: aggregate CPU gauge, RAM gauge,
// primary disk gauge, load values, and uptime.
func (w *SysMetricsWidget) smViewCompact(width int) []string {
	var lines []string
	m := w.metrics

	// CPU gauge.
	cpuGauge := smRenderGauge("CPU", m.CPU.Total, 100.0, width,
		smCPUWarnThreshold, smCPUCritThreshold)
	lines = append(lines, smTruncLine(cpuGauge, width))

	// RAM gauge with value.
	ramLabel := fmt.Sprintf("RAM: %s/%s",
		smFormatBytes(m.Memory.Used), smFormatBytes(m.Memory.Total))
	ramGauge := smRenderGauge("RAM", m.Memory.UsedPercent, 100.0, width,
		smMemWarnThreshold, smMemCritThreshold)
	_ = ramLabel
	lines = append(lines, smTruncLine(ramGauge, width))

	// Primary disk (first mount, typically /).
	if len(m.Disks) > 0 {
		d := m.Disks[0]
		diskGauge := smRenderDiskGauge(d, width)
		lines = append(lines, smTruncLine(diskGauge, width))
	} else {
		lines = append(lines, smTruncLine(components.Dim("No disks"), width))
	}

	// Load averages.
	loadLine := fmt.Sprintf("Load: %.2f / %.2f / %.2f",
		m.Load.Load1, m.Load.Load5, m.Load.Load15)
	lines = append(lines, smTruncLine(loadLine, width))

	// Uptime.
	uptimeLine := "Uptime: " + smFormatUptime(m.Uptime)
	lines = append(lines, smTruncLine(uptimeLine, width))

	return lines
}

// smViewExpanded renders the expanded view: per-core CPU sparklines (or
// aggregate sparkline), RAM+Swap gauges with values, all disk mounts,
// load sparkline, and uptime.
func (w *SysMetricsWidget) smViewExpanded(width int) []string {
	var lines []string
	m := w.metrics

	// CPU section header.
	lines = append(lines, components.Bold("CPU"))

	if w.perCore && len(m.CPU.Cores) > 0 {
		// Per-core display.
		for i, pct := range m.CPU.Cores {
			label := fmt.Sprintf("Core %d", i)
			coreGauge := smRenderGauge(label, pct, 100.0, width,
				smCPUWarnThreshold, smCPUCritThreshold)
			lines = append(lines, smTruncLine(coreGauge, width))
		}
	} else {
		// Aggregate CPU gauge.
		cpuGauge := smRenderGauge("CPU", m.CPU.Total, 100.0, width,
			smCPUWarnThreshold, smCPUCritThreshold)
		lines = append(lines, smTruncLine(cpuGauge, width))

		// CPU sparkline history.
		if len(w.cpuHistory) > 0 {
			sparkWidth := width - 6
			if sparkWidth < 5 {
				sparkWidth = 5
			}
			maxY := 100.0
			spark := components.NewSparkline(components.SparklineStyle{
				Width: sparkWidth,
				Color: smColorBlue,
				MaxY:  &maxY,
			})
			sparkLine := "  " + spark.Render(w.cpuHistory, sparkWidth)
			lines = append(lines, smTruncLine(sparkLine, width))
		}
	}

	// Memory section.
	lines = append(lines, "")
	lines = append(lines, components.Bold("Memory"))

	// RAM gauge with values.
	ramSuffix := fmt.Sprintf(" %s/%s",
		smFormatBytes(m.Memory.Used), smFormatBytes(m.Memory.Total))
	ramGauge := smRenderGaugeWithSuffix("RAM", m.Memory.UsedPercent, 100.0,
		width, smMemWarnThreshold, smMemCritThreshold, ramSuffix)
	lines = append(lines, smTruncLine(ramGauge, width))

	// Swap gauge (only if swap is configured).
	if m.Memory.SwapTotal > 0 {
		swapSuffix := fmt.Sprintf(" %s/%s",
			smFormatBytes(m.Memory.SwapUsed), smFormatBytes(m.Memory.SwapTotal))
		swapGauge := smRenderGaugeWithSuffix("Swap", m.Memory.SwapUsedPercent, 100.0,
			width, smMemWarnThreshold, smMemCritThreshold, swapSuffix)
		lines = append(lines, smTruncLine(swapGauge, width))
	}

	// Disk section.
	lines = append(lines, "")
	lines = append(lines, components.Bold("Disk"))

	if len(m.Disks) > 0 {
		for _, d := range m.Disks {
			diskGauge := smRenderDiskGauge(d, width)
			lines = append(lines, smTruncLine(diskGauge, width))
		}
	} else {
		lines = append(lines, smTruncLine(components.Dim("No disks"), width))
	}

	// Load section.
	lines = append(lines, "")
	lines = append(lines, components.Bold("Load"))

	loadLine := fmt.Sprintf("1m: %.2f  5m: %.2f  15m: %.2f",
		m.Load.Load1, m.Load.Load5, m.Load.Load15)
	lines = append(lines, smTruncLine(loadLine, width))

	// Load sparkline history.
	if len(w.loadHistory) > 0 {
		sparkWidth := width - 6
		if sparkWidth < 5 {
			sparkWidth = 5
		}
		spark := components.NewSparkline(components.SparklineStyle{
			Width: sparkWidth,
			Color: smColorBlue,
		})
		sparkLine := "  " + spark.Render(w.loadHistory, sparkWidth)
		lines = append(lines, smTruncLine(sparkLine, width))
	}

	// Uptime.
	lines = append(lines, "")
	uptimeLine := "Uptime: " + smFormatUptime(m.Uptime)
	lines = append(lines, smTruncLine(uptimeLine, width))

	return lines
}

// --- private helpers (prefixed with "sm" to avoid conflicts) ---

// smFormatBytes formats a byte count into a human-readable string with
// appropriate units (B, KB, MB, GB, TB).
func smFormatBytes(bytes uint64) string {
	const (
		kb = 1024
		mb = 1024 * kb
		gb = 1024 * mb
		tb = 1024 * gb
	)

	switch {
	case bytes >= tb:
		return fmt.Sprintf("%.1f TB", float64(bytes)/float64(tb))
	case bytes >= gb:
		return fmt.Sprintf("%.1f GB", float64(bytes)/float64(gb))
	case bytes >= mb:
		return fmt.Sprintf("%.1f MB", float64(bytes)/float64(mb))
	case bytes >= kb:
		return fmt.Sprintf("%.1f KB", float64(bytes)/float64(kb))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

// smFormatUptime formats a duration into a human-readable string like
// "14d 6h 23m" or "2h 15m" or "45m".
func smFormatUptime(d time.Duration) string {
	if d <= 0 {
		return "0m"
	}

	totalMinutes := int(d.Minutes())
	days := totalMinutes / (60 * 24)
	hours := (totalMinutes % (60 * 24)) / 60
	minutes := totalMinutes % 60

	var parts []string
	if days > 0 {
		parts = append(parts, fmt.Sprintf("%dd", days))
	}
	if hours > 0 || days > 0 {
		parts = append(parts, fmt.Sprintf("%dh", hours))
	}
	parts = append(parts, fmt.Sprintf("%dm", minutes))

	return strings.Join(parts, " ")
}

// smFormatPercent formats a float64 percentage value into a string like "73%".
func smFormatPercent(pct float64) string {
	return fmt.Sprintf("%d%%", int(math.Round(pct)))
}

// smRenderGauge renders a labeled gauge bar for a system metric.
func smRenderGauge(label string, value, maxValue float64, width int, warnThresh, critThresh float64) string {
	barWidth := width - len(label) - 7 // label + ": " + " NNN%"
	if barWidth < 5 {
		barWidth = 5
	}

	g := components.NewGauge(components.GaugeStyle{
		Width:             barWidth,
		ShowPercent:       true,
		FilledColor:       smColorGreen,
		EmptyColor:        "#333333",
		WarningThreshold:  warnThresh / 100.0,
		CriticalThreshold: critThresh / 100.0,
		WarningColor:      smColorYellow,
		CriticalColor:     smColorRed,
		Label:             label,
		LabelWidth:        len(label) + 1,
	})

	return g.Render(value, maxValue, barWidth)
}

// smRenderGaugeWithSuffix renders a labeled gauge bar with an additional
// suffix string appended after the percent label.
func smRenderGaugeWithSuffix(label string, value, maxValue float64, width int, warnThresh, critThresh float64, suffix string) string {
	// Account for the suffix length in bar width calculation.
	barWidth := width - len(label) - 7 - len(suffix)
	if barWidth < 5 {
		barWidth = 5
	}

	g := components.NewGauge(components.GaugeStyle{
		Width:             barWidth,
		ShowPercent:       true,
		FilledColor:       smColorGreen,
		EmptyColor:        "#333333",
		WarningThreshold:  warnThresh / 100.0,
		CriticalThreshold: critThresh / 100.0,
		WarningColor:      smColorYellow,
		CriticalColor:     smColorRed,
		Label:             label,
		LabelWidth:        len(label) + 1,
	})

	return g.Render(value, maxValue, barWidth) + suffix
}

// smRenderDiskGauge renders a gauge for a disk mount point with used/total
// values. Uses disk-specific thresholds (green<70%, yellow<85%, red>=85%).
func smRenderDiskGauge(d sysmetrics.DiskMetrics, width int) string {
	label := d.Path
	if len(label) > 12 {
		label = label[:12]
	}

	suffix := fmt.Sprintf(" %s/%s",
		smFormatBytes(d.Used), smFormatBytes(d.Total))

	barWidth := width - len(label) - 7 - len(suffix)
	if barWidth < 5 {
		barWidth = 5
	}

	g := components.NewGauge(components.GaugeStyle{
		Width:             barWidth,
		ShowPercent:       true,
		FilledColor:       smColorGreen,
		EmptyColor:        "#333333",
		WarningThreshold:  smDiskWarnThreshold / 100.0,
		CriticalThreshold: smDiskCritThreshold / 100.0,
		WarningColor:      smColorYellow,
		CriticalColor:     smColorRed,
		Label:             label,
		LabelWidth:        len(label) + 1,
	})

	return g.Render(d.UsedPercent, 100.0, barWidth) + suffix
}

// smCenterMessage renders a centered message in the given area.
func smCenterMessage(msg string, width, height int) string {
	lines := make([]string, height)
	midY := height / 2
	for i := range lines {
		if i == midY {
			vis := components.VisibleLen(msg)
			pad := (width - vis) / 2
			if pad < 0 {
				pad = 0
			}
			lines[i] = strings.Repeat(" ", pad) + msg
		} else {
			lines[i] = ""
		}
	}
	return strings.Join(lines, "\n")
}

// smFitLines pads or truncates a slice of lines to fit exactly height
// lines, each no wider than width visible characters.
func smFitLines(lines []string, width, height int) string {
	if len(lines) > height {
		lines = lines[:height]
	}
	for len(lines) < height {
		lines = append(lines, "")
	}
	for i, line := range lines {
		if components.VisibleLen(line) > width {
			lines[i] = components.Truncate(line, width)
		}
	}
	return strings.Join(lines, "\n")
}

// smTruncLine truncates a single line to at most width visible characters.
func smTruncLine(line string, width int) string {
	if components.VisibleLen(line) > width {
		return components.Truncate(line, width)
	}
	return line
}

// smDiskColor returns the appropriate color based on disk usage percentage.
func smDiskColor(pct float64) string {
	switch {
	case pct >= smDiskCritThreshold:
		return smColorRed
	case pct >= smDiskWarnThreshold:
		return smColorYellow
	default:
		return smColorGreen
	}
}

// compile-time check that SysMetricsWidget implements app.Widget.
var _ app.Widget = (*SysMetricsWidget)(nil)
