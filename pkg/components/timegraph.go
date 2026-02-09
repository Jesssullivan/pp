package components

import (
	"fmt"
	"math"
	"strings"
	"time"
)

// DataPoint represents a single time-value observation.
type DataPoint struct {
	Time  time.Time
	Value float64
}

// Series holds a named collection of data points with an associated color.
type Series struct {
	Name  string
	Color string // hex color, e.g. "#ff5500"
	Data  []DataPoint
}

// TimeGraphConfig holds configuration for a TimeGraph.
type TimeGraphConfig struct {
	Width      int            // cell width (used as default if Render width is 0)
	Height     int            // cell height (used as default if Render height is 0)
	ShowYAxis  bool           // show Y-axis labels (auto-hide if width < 20)
	ShowXAxis  bool           // show time labels at bottom (auto-hide if height < 5)
	ShowLegend bool           // show series legend at top
	YAxisWidth int            // width reserved for Y labels (default 6)
	MinY       *float64       // optional fixed Y minimum (nil = auto-scale)
	MaxY       *float64       // optional fixed Y maximum (nil = auto-scale)
	TimeWindow time.Duration  // visible time window (default 5 minutes)
}

// TimeGraph renders time-series data as a Braille-dot chart.
type TimeGraph struct {
	cfg    TimeGraphConfig
	series []Series
}

// NewTimeGraph creates a TimeGraph with the given configuration.
// Defaults are applied for zero-value fields.
func NewTimeGraph(cfg TimeGraphConfig) *TimeGraph {
	if cfg.YAxisWidth <= 0 {
		cfg.YAxisWidth = 6
	}
	if cfg.TimeWindow <= 0 {
		cfg.TimeWindow = 5 * time.Minute
	}
	return &TimeGraph{cfg: cfg}
}

// AddSeries adds a new named series with the given hex color and returns its
// index for use with PushValue and SetData.
func (tg *TimeGraph) AddSeries(name, color string) int {
	tg.series = append(tg.series, Series{Name: name, Color: color})
	return len(tg.series) - 1
}

// PushValue appends a single data point to the series at seriesIdx.
// Out-of-range indices are silently ignored.
func (tg *TimeGraph) PushValue(seriesIdx int, t time.Time, v float64) {
	if seriesIdx < 0 || seriesIdx >= len(tg.series) {
		return
	}
	tg.series[seriesIdx].Data = append(tg.series[seriesIdx].Data, DataPoint{Time: t, Value: v})
}

// SetData replaces all data points for the series at seriesIdx.
// Out-of-range indices are silently ignored.
func (tg *TimeGraph) SetData(seriesIdx int, points []DataPoint) {
	if seriesIdx < 0 || seriesIdx >= len(tg.series) {
		return
	}
	tg.series[seriesIdx].Data = points
}

// Render draws the graph into a string of the given cell dimensions.
// The output contains newline-separated lines with no trailing whitespace.
func (tg *TimeGraph) Render(width, height int) string {
	// Absolute minimums.
	if width < 10 || height < 2 {
		return tooSmallMsg(width)
	}

	// Determine which chrome is visible via graceful degradation.
	showLegend := tg.cfg.ShowLegend
	showYAxis := tg.cfg.ShowYAxis
	showXAxis := tg.cfg.ShowXAxis

	if height < 3 {
		showLegend = false
	}
	if height < 5 {
		showXAxis = false
	}
	if width < 20 {
		showYAxis = false
	}

	// Compute chart area dimensions.
	yAxisW := 0
	if showYAxis {
		yAxisW = tg.cfg.YAxisWidth
	}

	chartW := width - yAxisW
	if chartW < 1 {
		chartW = 1
	}

	legendH := 0
	if showLegend {
		legendH = 1
	}
	xAxisH := 0
	if showXAxis {
		xAxisH = 1
	}

	chartH := height - legendH - xAxisH
	if chartH < 1 {
		chartH = 1
	}

	// Determine time range.
	now := tg.latestTime()
	tMin := now.Add(-tg.cfg.TimeWindow)
	tMax := now

	// Determine Y range.
	yMin, yMax := tg.yRange(tMin, tMax)

	// Braille grid: each cell is 2 dots wide, 4 dots tall.
	dotsW := chartW * 2
	dotsH := chartH * 4

	// Allocate braille grid [row][col] of dot bitmasks.
	grid := make([][]uint8, chartH)
	for r := range grid {
		grid[r] = make([]uint8, chartW)
	}

	// Allocate per-cell color index tracking. We store the series index of
	// the last dot set in each cell for coloring. -1 means no dot.
	cellColor := make([][]int, chartH)
	for r := range cellColor {
		cellColor[r] = make([]int, chartW)
		for c := range cellColor[r] {
			cellColor[r][c] = -1
		}
	}

	// For multi-series coloring, we need per-cell per-series tracking.
	// We use a map keyed by (row, col) -> set of series indices present.
	type cellKey struct{ r, c int }
	cellSeries := make(map[cellKey]map[int]bool)

	// Plot each series.
	tRange := tMax.Sub(tMin).Seconds()
	yRange := yMax - yMin

	for si, s := range tg.series {
		for _, dp := range s.Data {
			if dp.Time.Before(tMin) || dp.Time.After(tMax) {
				continue
			}

			// Map time to horizontal dot position.
			var dotX int
			if tRange <= 0 {
				dotX = dotsW / 2
			} else {
				frac := dp.Time.Sub(tMin).Seconds() / tRange
				dotX = int(frac * float64(dotsW-1))
			}
			if dotX < 0 {
				dotX = 0
			}
			if dotX >= dotsW {
				dotX = dotsW - 1
			}

			// Map value to vertical dot position (0 = top).
			var dotY int
			if yRange <= 0 {
				dotY = dotsH / 2
			} else {
				frac := (dp.Value - yMin) / yRange
				// Clamp for fixed range.
				if frac < 0 {
					frac = 0
				}
				if frac > 1 {
					frac = 1
				}
				// Invert: high values at top (low row index).
				dotY = int((1 - frac) * float64(dotsH-1))
			}
			if dotY < 0 {
				dotY = 0
			}
			if dotY >= dotsH {
				dotY = dotsH - 1
			}

			// Convert dot coordinates to cell + offset.
			cellCol := dotX / 2
			cellRow := dotY / 4
			offX := dotX % 2
			offY := dotY % 4

			if cellCol >= chartW {
				cellCol = chartW - 1
			}
			if cellRow >= chartH {
				cellRow = chartH - 1
			}

			bit := brailleBit(offX, offY)
			grid[cellRow][cellCol] |= bit
			cellColor[cellRow][cellCol] = si

			key := cellKey{cellRow, cellCol}
			if cellSeries[key] == nil {
				cellSeries[key] = make(map[int]bool)
			}
			cellSeries[key][si] = true
		}
	}

	// Build output lines.
	var lines []string

	// Legend line.
	if showLegend && len(tg.series) > 0 {
		lines = append(lines, tg.renderLegend(width))
	}

	// Chart lines.
	resetSeq := Reset()
	for r := 0; r < chartH; r++ {
		var sb strings.Builder

		// Y-axis label.
		if showYAxis {
			// Map row to value: row 0 = yMax, row chartH-1 = yMin.
			var val float64
			if chartH <= 1 {
				val = (yMin + yMax) / 2
			} else {
				val = yMax - (yMax-yMin)*float64(r)/float64(chartH-1)
			}
			label := formatSI(val)
			// Right-align the label in yAxisW-1 chars, followed by a space.
			padded := padLeft(label, yAxisW-1) + " "
			sb.WriteString(padded)
		}

		// Chart cells.
		for c := 0; c < chartW; c++ {
			ch := rune(0x2800 + int(grid[r][c]))
			si := cellColor[r][c]
			if si >= 0 && si < len(tg.series) && grid[r][c] != 0 {
				colorSeq := Color(tg.series[si].Color)
				sb.WriteString(colorSeq)
				sb.WriteRune(ch)
				sb.WriteString(resetSeq)
			} else {
				sb.WriteRune(ch)
			}
		}

		lines = append(lines, trimRight(sb.String()))
	}

	// X-axis labels.
	if showXAxis {
		lines = append(lines, tg.renderXAxis(yAxisW, chartW))
	}

	return strings.Join(lines, "\n")
}

// latestTime returns the most recent timestamp across all series,
// or time.Now() if no data exists.
func (tg *TimeGraph) latestTime() time.Time {
	var latest time.Time
	found := false
	for _, s := range tg.series {
		for _, dp := range s.Data {
			if !found || dp.Time.After(latest) {
				latest = dp.Time
				found = true
			}
		}
	}
	if !found {
		return time.Now()
	}
	return latest
}

// yRange computes the Y-axis range from data within the time window,
// applying 10% padding and honoring fixed bounds.
func (tg *TimeGraph) yRange(tMin, tMax time.Time) (float64, float64) {
	if tg.cfg.MinY != nil && tg.cfg.MaxY != nil {
		return *tg.cfg.MinY, *tg.cfg.MaxY
	}

	lo := math.Inf(1)
	hi := math.Inf(-1)
	count := 0

	for _, s := range tg.series {
		for _, dp := range s.Data {
			if dp.Time.Before(tMin) || dp.Time.After(tMax) {
				continue
			}
			if dp.Value < lo {
				lo = dp.Value
			}
			if dp.Value > hi {
				hi = dp.Value
			}
			count++
		}
	}

	if count == 0 {
		lo, hi = 0, 1
	} else if lo == hi {
		// Single value: create a range around it.
		if lo == 0 {
			lo, hi = 0, 1
		} else {
			lo = lo - math.Abs(lo)*0.1
			hi = hi + math.Abs(hi)*0.1
		}
	} else {
		// 10% padding.
		span := hi - lo
		lo -= span * 0.1
		hi += span * 0.1
	}

	// Override with fixed bounds where set.
	if tg.cfg.MinY != nil {
		lo = *tg.cfg.MinY
	}
	if tg.cfg.MaxY != nil {
		hi = *tg.cfg.MaxY
	}

	return lo, hi
}

// renderLegend builds the legend line showing colored markers and series names.
func (tg *TimeGraph) renderLegend(maxWidth int) string {
	var sb strings.Builder
	resetSeq := Reset()

	for i, s := range tg.series {
		if i > 0 {
			sb.WriteString("  ")
		}
		colorSeq := Color(s.Color)
		sb.WriteString(colorSeq)
		sb.WriteString("\u2588") // full block character as color swatch
		sb.WriteString(resetSeq)
		sb.WriteString(" ")
		sb.WriteString(s.Name)
	}

	result := sb.String()
	// We do not truncate the legend string here because ANSI escape
	// sequences make simple truncation unreliable. The caller can rely on
	// terminal wrapping.
	return trimRight(result)
}

// renderXAxis builds the X-axis label line with relative time markers.
func (tg *TimeGraph) renderXAxis(yAxisW, chartW int) string {
	if chartW < 3 {
		return ""
	}

	// Generate labels at evenly spaced positions.
	window := tg.cfg.TimeWindow
	labels := []struct {
		text string
		frac float64 // 0.0 = left edge (oldest), 1.0 = right edge (newest)
	}{
		{text: formatDuration(window), frac: 0.0},
		{text: "now", frac: 1.0},
	}

	// Add intermediate labels if there's room.
	if chartW >= 30 {
		labels = []struct {
			text string
			frac float64
		}{
			{text: formatDuration(window), frac: 0.0},
			{text: formatDuration(window * 3 / 4), frac: 0.25},
			{text: formatDuration(window / 2), frac: 0.5},
			{text: formatDuration(window / 4), frac: 0.75},
			{text: "now", frac: 1.0},
		}
	} else if chartW >= 15 {
		labels = []struct {
			text string
			frac float64
		}{
			{text: formatDuration(window), frac: 0.0},
			{text: formatDuration(window / 2), frac: 0.5},
			{text: "now", frac: 1.0},
		}
	}

	// Place labels on the axis line.
	totalW := yAxisW + chartW
	axis := make([]byte, totalW)
	for i := range axis {
		axis[i] = ' '
	}

	for _, lbl := range labels {
		pos := yAxisW + int(lbl.frac*float64(chartW-1))
		// Center the label around pos.
		start := pos - len(lbl.text)/2
		if start < yAxisW {
			start = yAxisW
		}
		end := start + len(lbl.text)
		if end > totalW {
			start = totalW - len(lbl.text)
			if start < yAxisW {
				start = yAxisW
			}
			end = start + len(lbl.text)
		}
		if end > totalW {
			end = totalW
		}
		copy(axis[start:end], lbl.text)
	}

	return trimRight(string(axis))
}

// brailleBit returns the bitmask for a dot at offset (offX, offY) within a
// Braille cell. offX is 0 (left) or 1 (right). offY is 0..3 (top to bottom).
//
// Unicode Braille dot numbering:
//
//	1 4      bit: 0x01  0x08
//	2 5           0x02  0x10
//	3 6           0x04  0x20
//	7 8           0x40  0x80
func brailleBit(offX, offY int) uint8 {
	// Left column dots: 1,2,3,7 -> bits 0,1,2,6
	// Right column dots: 4,5,6,8 -> bits 3,4,5,7
	leftBits := [4]uint8{0x01, 0x02, 0x04, 0x40}
	rightBits := [4]uint8{0x08, 0x10, 0x20, 0x80}

	if offY < 0 || offY > 3 {
		return 0
	}
	if offX == 0 {
		return leftBits[offY]
	}
	return rightBits[offY]
}

// formatSI formats a float with SI suffixes: K, M, G, T.
// Examples: 1000 -> "1K", 1500 -> "1.5K", 1000000 -> "1M".
func formatSI(v float64) string {
	negative := v < 0
	abs := math.Abs(v)

	prefix := ""
	if negative {
		prefix = "-"
	}

	switch {
	case abs >= 1e12:
		return prefix + formatSIValue(abs/1e12) + "T"
	case abs >= 1e9:
		return prefix + formatSIValue(abs/1e9) + "G"
	case abs >= 1e6:
		return prefix + formatSIValue(abs/1e6) + "M"
	case abs >= 1e3:
		return prefix + formatSIValue(abs/1e3) + "K"
	default:
		// For small values, show up to 1 decimal place.
		if abs == math.Trunc(abs) {
			return fmt.Sprintf("%s%d", prefix, int(abs))
		}
		return fmt.Sprintf("%s%.1f", prefix, abs)
	}
}

// formatSIValue formats the numeric part of an SI-suffixed value.
func formatSIValue(v float64) string {
	if v == math.Trunc(v) {
		return fmt.Sprintf("%d", int(v))
	}
	// One decimal place, strip trailing zero.
	s := fmt.Sprintf("%.1f", v)
	s = strings.TrimRight(s, "0")
	s = strings.TrimRight(s, ".")
	return s
}

// formatDuration formats a duration as a relative time label like "-5m" or "-30s".
func formatDuration(d time.Duration) string {
	if d <= 0 {
		return "now"
	}
	if d >= time.Hour {
		h := int(d.Hours())
		return fmt.Sprintf("-%dh", h)
	}
	if d >= time.Minute {
		m := int(d.Minutes())
		return fmt.Sprintf("-%dm", m)
	}
	s := int(d.Seconds())
	if s <= 0 {
		s = 1
	}
	return fmt.Sprintf("-%ds", s)
}

// tooSmallMsg returns a centered "too small" message for tiny viewports.
func tooSmallMsg(width int) string {
	msg := "too small"
	if width < len(msg) {
		return msg[:width]
	}
	return msg
}

// padLeft right-aligns s within a field of the given width using spaces.
func padLeft(s string, width int) string {
	if len(s) >= width {
		return s[:width]
	}
	return strings.Repeat(" ", width-len(s)) + s
}

// trimRight removes trailing whitespace from a string.
func trimRight(s string) string {
	return strings.TrimRight(s, " \t")
}
