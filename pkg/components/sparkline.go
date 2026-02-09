package components

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// Sparkline block characters: 8 vertical levels per cell.
var sparkBlocks = [8]rune{
	'\u2581', // 1/8 ▁
	'\u2582', // 2/8 ▂
	'\u2583', // 3/8 ▃
	'\u2584', // 4/8 ▄
	'\u2585', // 5/8 ▅
	'\u2586', // 6/8 ▆
	'\u2587', // 7/8 ▇
	'\u2588', // 8/8 █
}

// SparklineStyle configures the appearance of a sparkline.
type SparklineStyle struct {
	Width      int      // number of cells to display
	Color      string   // hex color for the sparkline (default "#64B5F6")
	ShowMinMax bool     // show min/max values flanking the sparkline
	MinY       *float64 // optional fixed minimum Y (nil = auto-scale)
	MaxY       *float64 // optional fixed maximum Y (nil = auto-scale)
	Label      string   // optional prefix label
}

// Sparkline renders inline sparkline charts using Unicode block elements.
type Sparkline struct {
	style SparklineStyle
}

// DefaultSparklineStyle returns a SparklineStyle with sensible defaults.
func DefaultSparklineStyle() SparklineStyle {
	return SparklineStyle{
		Width: 20,
		Color: "#64B5F6",
	}
}

// NewSparkline creates a new Sparkline with the given style.
func NewSparkline(style SparklineStyle) *Sparkline {
	return &Sparkline{style: style}
}

// Render renders a sparkline at the given width. The width parameter overrides
// the style width for this call.
func (s *Sparkline) Render(data []float64, width int) string {
	if len(data) == 0 {
		return ""
	}
	if width <= 0 {
		width = s.style.Width
	}
	if width <= 0 {
		width = 20
	}

	// Take the last `width` data points.
	points := data
	if len(points) > width {
		points = points[len(points)-width:]
	}

	// Determine Y range.
	minY, maxY := sparkAutoRange(points)
	if s.style.MinY != nil {
		minY = *s.style.MinY
	}
	if s.style.MaxY != nil {
		maxY = *s.style.MaxY
	}

	// Build sparkline characters.
	sparkChars := sparkMapToBlocks(points, minY, maxY)

	// Color the sparkline.
	colored := sparkColorize(sparkChars, s.style.Color)

	var b strings.Builder

	// Prepend label.
	if s.style.Label != "" {
		b.WriteString(s.style.Label)
		b.WriteString(" ")
	}

	// Show min label.
	if s.style.ShowMinMax {
		b.WriteString(sparkFormatValue(minY))
		b.WriteString(" ")
	}

	b.WriteString(colored)

	// Show max label.
	if s.style.ShowMinMax {
		b.WriteString(" ")
		b.WriteString(sparkFormatValue(maxY))
	}

	return b.String()
}

// RenderWithDelta renders the sparkline with a delta indicator comparing the
// last value to the previous value. Uses up arrow, down arrow, or right arrow.
func (s *Sparkline) RenderWithDelta(data []float64, width int) string {
	base := s.Render(data, width)
	if base == "" {
		return ""
	}

	if len(data) < 2 {
		return base + " \u2192" + "0.0%"
	}

	prev := data[len(data)-2]
	curr := data[len(data)-1]

	var delta float64
	if prev != 0 {
		delta = ((curr - prev) / math.Abs(prev)) * 100
	} else if curr > 0 {
		delta = 100.0
	} else if curr < 0 {
		delta = -100.0
	}

	var indicator string
	switch {
	case delta > 0:
		indicator = fmt.Sprintf(" \u2191%.1f%%", delta)
	case delta < 0:
		indicator = fmt.Sprintf(" \u2193%.1f%%", math.Abs(delta))
	default:
		indicator = " \u21920.0%"
	}

	return base + indicator
}

// sparkAutoRange finds the min and max values in a data slice.
func sparkAutoRange(data []float64) (minY, maxY float64) {
	if len(data) == 0 {
		return 0, 0
	}
	minY = data[0]
	maxY = data[0]
	for _, v := range data[1:] {
		if v < minY {
			minY = v
		}
		if v > maxY {
			maxY = v
		}
	}
	return minY, maxY
}

// sparkMapToBlocks maps data values to block characters based on the Y range.
func sparkMapToBlocks(data []float64, minY, maxY float64) string {
	var b strings.Builder
	rangeY := maxY - minY

	for _, v := range data {
		var idx int
		if rangeY <= 0 {
			// All values equal: render at mid-height.
			idx = 3
		} else {
			// Normalize to [0, 1] and map to [0, 7].
			normalized := (v - minY) / rangeY
			if normalized < 0 {
				normalized = 0
			}
			if normalized > 1 {
				normalized = 1
			}
			idx = int(math.Round(normalized * 7))
			if idx > 7 {
				idx = 7
			}
		}
		b.WriteRune(sparkBlocks[idx])
	}

	return b.String()
}

// sparkColorize wraps the sparkline string in ANSI color escapes.
func sparkColorize(s, hexColor string) string {
	if hexColor == "" {
		return s
	}
	fg := sparkColorFg(hexColor)
	if fg == "" {
		return s
	}
	return fg + s + "\x1b[0m"
}

// sparkColorFg returns an ANSI true-color foreground escape from hex.
func sparkColorFg(hex string) string {
	r, g, b, ok := sparkParseHexColor(hex)
	if !ok {
		return ""
	}
	return fmt.Sprintf("\x1b[38;2;%d;%d;%dm", r, g, b)
}

// sparkParseHexColor parses "#RRGGBB" or "RRGGBB" into r, g, b components.
func sparkParseHexColor(hex string) (r, g, b uint8, ok bool) {
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

// sparkFormatValue formats a float64 for compact display in min/max labels.
func sparkFormatValue(v float64) string {
	if v == math.Trunc(v) && math.Abs(v) < 1e6 {
		return fmt.Sprintf("%.0f", v)
	}
	return fmt.Sprintf("%.1f", v)
}

// sparkStripANSI removes ANSI escape sequences for visible-width calculations.
func sparkStripANSI(s string) string {
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
