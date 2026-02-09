package theme

import (
	"fmt"
	"strings"
)

// thApplyBorder colors border text based on whether the widget is focused.
func thApplyBorder(text string, t Theme, focused bool) string {
	color := t.Border
	if focused {
		color = t.BorderFocus
	}
	return thColorize(text, color)
}

// thApplyStatus colors text based on a status string.
// Recognized statuses: "ok", "warn", "warning", "error", "unknown".
func thApplyStatus(text, status string, t Theme) string {
	var color string
	switch strings.ToLower(status) {
	case "ok", "healthy", "running":
		color = t.StatusOK
	case "warn", "warning":
		color = t.StatusWarn
	case "error", "err", "critical", "failed":
		color = t.StatusError
	default:
		color = t.StatusUnknown
	}
	return thColorize(text, color)
}

// thApplyGauge returns the filled and empty hex colors for a gauge based on
// the current ratio. Thresholds: >=0.9 critical, >=0.7 warning, else normal.
func thApplyGauge(ratio float64, t Theme) (filled, empty string) {
	empty = t.GaugeEmpty
	switch {
	case ratio >= 0.9:
		filled = t.GaugeCrit
	case ratio >= 0.7:
		filled = t.GaugeWarn
	default:
		filled = t.GaugeFilled
	}
	return filled, empty
}

// thColorize wraps text in ANSI true-color foreground escape sequences using
// the given hex color. Returns text unchanged if hexColor is empty or invalid.
func thColorize(text, hexColor string) string {
	if hexColor == "" {
		return text
	}
	r, g, b, ok := thParseHex(hexColor)
	if !ok {
		return text
	}
	return fmt.Sprintf("\x1b[38;2;%d;%d;%dm%s\x1b[0m", r, g, b, text)
}
