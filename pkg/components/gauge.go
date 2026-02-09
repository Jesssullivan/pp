package components

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// Block characters for sub-cell precision (8 levels per cell).
var gaugeBlocks = [9]rune{
	' ',    // 0/8 empty
	'\u258F', // 1/8 ▏
	'\u258E', // 2/8 ▎
	'\u258D', // 3/8 ▍
	'\u258C', // 4/8 ▌
	'\u258B', // 5/8 ▋
	'\u258A', // 6/8 ▊
	'\u2589', // 7/8 ▉
	'\u2588', // 8/8 █
}

// GaugeStyle configures the appearance of a horizontal bar gauge.
type GaugeStyle struct {
	Width             int     // total width in cells for the bar portion
	ShowPercent       bool    // show "73%" label after bar
	ShowValue         bool    // show "7.3/10.0" label after bar
	Label             string  // optional left label (e.g., "CPU")
	LabelWidth        int     // fixed width for label area (0 = no label)
	FilledColor       string  // hex color for filled portion (default "#4CAF50")
	EmptyColor        string  // hex color for empty portion (default "#333333")
	WarningThreshold  float64 // threshold (0-1) where color changes to warning
	CriticalThreshold float64 // threshold (0-1) where color changes to critical
	WarningColor      string  // hex color for warning (default "#FF9800")
	CriticalColor     string  // hex color for critical (default "#F44336")
}

// GaugeData holds data for a single gauge in a multi-gauge render.
type GaugeData struct {
	Label    string
	Value    float64
	MaxValue float64
}

// Gauge renders horizontal bar gauges with sub-cell precision.
type Gauge struct {
	style GaugeStyle
}

// DefaultGaugeStyle returns a GaugeStyle with sensible defaults.
func DefaultGaugeStyle() GaugeStyle {
	return GaugeStyle{
		Width:             20,
		ShowPercent:       true,
		ShowValue:         false,
		FilledColor:       "#4CAF50",
		EmptyColor:        "#333333",
		WarningThreshold:  0.7,
		CriticalThreshold: 0.9,
		WarningColor:      "#FF9800",
		CriticalColor:     "#F44336",
	}
}

// NewGauge creates a new Gauge with the given style.
func NewGauge(style GaugeStyle) *Gauge {
	return &Gauge{style: style}
}

// Render renders a gauge bar at the given width. The width parameter overrides
// the style width for this call. value and maxValue define the fill ratio.
func (g *Gauge) Render(value, maxValue float64, width int) string {
	if width <= 0 {
		width = g.style.Width
	}
	if width <= 0 {
		width = 20
	}

	// Clamp ratio to [0, 1].
	ratio := 0.0
	if maxValue > 0 {
		ratio = value / maxValue
	}
	if ratio < 0 {
		ratio = 0
	}
	if ratio > 1 {
		ratio = 1
	}

	// Select fill color based on thresholds.
	fillColor := g.style.FilledColor
	if fillColor == "" {
		fillColor = "#4CAF50"
	}
	if g.style.WarningThreshold > 0 && ratio >= g.style.WarningThreshold {
		fillColor = g.style.WarningColor
		if fillColor == "" {
			fillColor = "#FF9800"
		}
	}
	if g.style.CriticalThreshold > 0 && ratio >= g.style.CriticalThreshold {
		fillColor = g.style.CriticalColor
		if fillColor == "" {
			fillColor = "#F44336"
		}
	}

	// Build the bar with sub-cell precision.
	bar := gaugeRenderBar(ratio, width, fillColor, g.style.EmptyColor)

	var b strings.Builder

	// Prepend label if set.
	if g.style.Label != "" {
		labelW := g.style.LabelWidth
		if labelW <= 0 {
			labelW = len(g.style.Label) + 1
		}
		padded := gaugePadRight(g.style.Label, labelW)
		b.WriteString(padded)
	}

	b.WriteString(bar)

	// Append percent label.
	if g.style.ShowPercent {
		pct := ratio * 100
		b.WriteString(fmt.Sprintf(" %d%%", int(math.Round(pct))))
	}

	// Append value label.
	if g.style.ShowValue {
		b.WriteString(fmt.Sprintf(" %.1f/%.1f", value, maxValue))
	}

	return b.String()
}

// RenderMulti renders multiple gauges stacked vertically with aligned labels.
func (g *Gauge) RenderMulti(gauges []GaugeData, width int) string {
	if len(gauges) == 0 {
		return ""
	}

	// Find the maximum label width for alignment.
	maxLabelLen := 0
	for _, gd := range gauges {
		if len(gd.Label) > maxLabelLen {
			maxLabelLen = len(gd.Label)
		}
	}

	var lines []string
	for _, gd := range gauges {
		// Create a copy of the gauge with aligned label settings.
		sg := *g
		sg.style.Label = gd.Label
		sg.style.LabelWidth = maxLabelLen + 1 // +1 for spacing
		lines = append(lines, sg.Render(gd.Value, gd.MaxValue, width))
	}

	return strings.Join(lines, "\n")
}

// gaugeRenderBar builds the ANSI-colored bar string with sub-cell precision.
func gaugeRenderBar(ratio float64, width int, fillColor, emptyColor string) string {
	// Total sub-cell units available.
	totalUnits := width * 8
	filledUnits := int(math.Round(ratio * float64(totalUnits)))
	if filledUnits < 0 {
		filledUnits = 0
	}
	if filledUnits > totalUnits {
		filledUnits = totalUnits
	}

	fullCells := filledUnits / 8
	partialEighths := filledUnits % 8
	emptyCells := width - fullCells
	if partialEighths > 0 {
		emptyCells--
	}
	if emptyCells < 0 {
		emptyCells = 0
	}

	fgFill := gaugeColorFg(fillColor)
	bgEmpty := gaugeColorBg(emptyColor)
	fgEmpty := gaugeColorFg(emptyColor)
	reset := "\x1b[0m"

	var b strings.Builder

	// Full filled cells: filled-color foreground, empty-color background.
	if fullCells > 0 {
		b.WriteString(fgFill)
		b.WriteString(bgEmpty)
		b.WriteString(strings.Repeat(string(gaugeBlocks[8]), fullCells))
		b.WriteString(reset)
	}

	// Partial cell at the boundary.
	if partialEighths > 0 {
		b.WriteString(fgFill)
		b.WriteString(bgEmpty)
		b.WriteRune(gaugeBlocks[partialEighths])
		b.WriteString(reset)
	}

	// Empty cells.
	if emptyCells > 0 {
		b.WriteString(fgEmpty)
		b.WriteString(strings.Repeat(" ", emptyCells))
		b.WriteString(reset)
	}

	return b.String()
}

// gaugeColorFg returns an ANSI true-color foreground escape sequence from hex.
func gaugeColorFg(hex string) string {
	r, g, b, ok := gaugeParseHexColor(hex)
	if !ok {
		return ""
	}
	return fmt.Sprintf("\x1b[38;2;%d;%d;%dm", r, g, b)
}

// gaugeColorBg returns an ANSI true-color background escape sequence from hex.
func gaugeColorBg(hex string) string {
	r, g, b, ok := gaugeParseHexColor(hex)
	if !ok {
		return ""
	}
	return fmt.Sprintf("\x1b[48;2;%d;%d;%dm", r, g, b)
}

// gaugeParseHexColor parses "#RRGGBB" or "RRGGBB" into r, g, b components.
func gaugeParseHexColor(hex string) (r, g, b uint8, ok bool) {
	hex = strings.TrimPrefix(hex, "#")
	if len(hex) != 6 {
		return 0, 0, 0, false
	}
	rv, err := strconv.ParseUint(hex[0:2], 16, 8)
	if err != nil {
		return 0, 0, 0, false
	}
	gv, err := strconv.ParseUint(hex[2:4], 16, 8)
	if err != nil {
		return 0, 0, 0, false
	}
	bv, err := strconv.ParseUint(hex[4:6], 16, 8)
	if err != nil {
		return 0, 0, 0, false
	}
	return uint8(rv), uint8(gv), uint8(bv), true
}

// gaugePadRight pads s to the given width with spaces on the right.
func gaugePadRight(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}

// gaugeStripANSI removes ANSI escape sequences for visible-width calculations.
func gaugeStripANSI(s string) string {
	var b strings.Builder
	inEsc := false
	for _, r := range s {
		if r == '\x1b' {
			inEsc = true
			continue
		}
		if inEsc {
			if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') {
				inEsc = false
			}
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// gaugeVisibleWidth returns the visible width of a string, ignoring ANSI escapes.
func gaugeVisibleWidth(s string) int {
	stripped := gaugeStripANSI(s)
	return len([]rune(stripped))
}
