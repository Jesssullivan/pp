package components

import (
	"strings"
	"testing"
)

// gaugeTestStrip removes ANSI escapes for asserting visible content.
func gaugeTestStrip(s string) string {
	return gaugeStripANSI(s)
}

func TestGaugeZeroPercent(t *testing.T) {
	g := NewGauge(DefaultGaugeStyle())
	g.style.ShowPercent = true
	result := g.Render(0, 100, 20)
	stripped := gaugeTestStrip(result)
	// Should be all spaces for the bar plus " 0%".
	if !strings.Contains(stripped, "0%") {
		t.Errorf("expected 0%% label, got %q", stripped)
	}
	// Bar portion should have no block characters.
	barPart := stripped[:20]
	for _, r := range barPart {
		if r >= '\u2581' && r <= '\u2588' {
			t.Errorf("expected empty bar for 0%%, found block char %q in %q", string(r), stripped)
			break
		}
	}
}

func TestGaugeHundredPercent(t *testing.T) {
	g := NewGauge(DefaultGaugeStyle())
	g.style.ShowPercent = true
	result := g.Render(100, 100, 20)
	stripped := gaugeTestStrip(result)
	if !strings.Contains(stripped, "100%") {
		t.Errorf("expected 100%% label, got %q", stripped)
	}
	// Count full block characters: should be exactly 20.
	barPart := []rune(stripped)
	fullBlocks := 0
	for _, r := range barPart {
		if r == '\u2588' {
			fullBlocks++
		}
	}
	if fullBlocks != 20 {
		t.Errorf("expected 20 full blocks for 100%%, got %d in %q", fullBlocks, stripped)
	}
}

func TestGaugeFiftyPercent(t *testing.T) {
	g := NewGauge(DefaultGaugeStyle())
	g.style.ShowPercent = true
	result := g.Render(50, 100, 20)
	stripped := gaugeTestStrip(result)
	if !strings.Contains(stripped, "50%") {
		t.Errorf("expected 50%% label, got %q", stripped)
	}
	// Should have 10 full blocks for 50% of 20 width.
	fullBlocks := strings.Count(stripped, string('\u2588'))
	if fullBlocks != 10 {
		t.Errorf("expected 10 full blocks for 50%%, got %d in %q", fullBlocks, stripped)
	}
}

func TestGaugeSubCellPrecision(t *testing.T) {
	// 12.5% of width=8 = 1 cell exactly (8 sub-units), but
	// 12.5% of width=10 = 10 sub-units = 1 full + 1/4 block.
	g := NewGauge(DefaultGaugeStyle())
	g.style.ShowPercent = false
	result := g.Render(12.5, 100, 10)
	stripped := gaugeTestStrip(result)

	// 12.5% of 10 cells = 10 sub-units = 1 full block + partial(2/8 = ▎)
	hasPartial := false
	for _, r := range stripped {
		if r == '\u258E' { // ▎ (1/4 = 2/8)
			hasPartial = true
		}
	}
	if !hasPartial {
		// Check for any partial block character.
		hasAnyPartial := false
		for _, r := range stripped {
			if r >= '\u2589' && r <= '\u258F' {
				hasAnyPartial = true
				break
			}
		}
		if !hasAnyPartial {
			t.Errorf("expected partial block char for 12.5%% at width 10, got %q", stripped)
		}
	}
}

func TestGaugeSubCellOneEighth(t *testing.T) {
	// 1/8 of 1 cell = 12.5% of width 1 bar.
	g := NewGauge(DefaultGaugeStyle())
	g.style.ShowPercent = false
	// 12.5% of width=1 = 1 sub-unit = 1/8 block ▏.
	result := g.Render(12.5, 100, 1)
	stripped := gaugeTestStrip(result)
	if !strings.ContainsRune(stripped, '\u258F') {
		t.Errorf("expected 1/8 block ▏ for 12.5%% at width 1, got %q", stripped)
	}
}

func TestGaugeColorThresholdGreen(t *testing.T) {
	g := NewGauge(DefaultGaugeStyle())
	// 30% is below warning threshold (0.7).
	result := g.Render(30, 100, 20)
	// Should contain the green color code: #4CAF50 → rgb(76, 175, 80).
	if !strings.Contains(result, "38;2;76;175;80") {
		t.Errorf("expected green color for 30%%, got %q", result)
	}
}

func TestGaugeColorThresholdWarning(t *testing.T) {
	g := NewGauge(DefaultGaugeStyle())
	// 75% is above warning (0.7) but below critical (0.9).
	result := g.Render(75, 100, 20)
	// Should contain warning color: #FF9800 → rgb(255, 152, 0).
	if !strings.Contains(result, "38;2;255;152;0") {
		t.Errorf("expected warning color for 75%%, got %q", result)
	}
}

func TestGaugeColorThresholdCritical(t *testing.T) {
	g := NewGauge(DefaultGaugeStyle())
	// 95% is above critical (0.9).
	result := g.Render(95, 100, 20)
	// Should contain critical color: #F44336 → rgb(244, 67, 54).
	if !strings.Contains(result, "38;2;244;67;54") {
		t.Errorf("expected critical color for 95%%, got %q", result)
	}
}

func TestGaugeLabelRendering(t *testing.T) {
	style := DefaultGaugeStyle()
	style.Label = "CPU"
	style.LabelWidth = 6
	g := NewGauge(style)
	result := g.Render(50, 100, 10)
	stripped := gaugeTestStrip(result)
	if !strings.HasPrefix(stripped, "CPU") {
		t.Errorf("expected label prefix 'CPU', got %q", stripped)
	}
	// Label area should be 6 chars wide.
	labelArea := stripped[:6]
	if labelArea != "CPU   " {
		t.Errorf("expected 'CPU   ' (6 chars), got %q", labelArea)
	}
}

func TestGaugeLabelAlignment(t *testing.T) {
	style := DefaultGaugeStyle()
	style.Label = "RAM"
	style.LabelWidth = 8
	g := NewGauge(style)
	result := g.Render(25, 100, 10)
	stripped := gaugeTestStrip(result)
	if !strings.HasPrefix(stripped, "RAM     ") {
		t.Errorf("expected 'RAM     ' (8 chars label), got prefix %q", stripped[:8])
	}
}

func TestGaugePercentLabel(t *testing.T) {
	style := DefaultGaugeStyle()
	style.ShowPercent = true
	style.ShowValue = false
	g := NewGauge(style)
	result := g.Render(73, 100, 20)
	stripped := gaugeTestStrip(result)
	if !strings.Contains(stripped, "73%") {
		t.Errorf("expected '73%%' label, got %q", stripped)
	}
}

func TestGaugeValueLabel(t *testing.T) {
	style := DefaultGaugeStyle()
	style.ShowPercent = false
	style.ShowValue = true
	g := NewGauge(style)
	result := g.Render(7.3, 10.0, 20)
	stripped := gaugeTestStrip(result)
	if !strings.Contains(stripped, "7.3/10.0") {
		t.Errorf("expected '7.3/10.0' label, got %q", stripped)
	}
}

func TestGaugeClampOverflow(t *testing.T) {
	g := NewGauge(DefaultGaugeStyle())
	g.style.ShowPercent = true
	result := g.Render(150, 100, 20)
	stripped := gaugeTestStrip(result)
	if !strings.Contains(stripped, "100%") {
		t.Errorf("expected clamped to 100%%, got %q", stripped)
	}
	fullBlocks := strings.Count(stripped, string('\u2588'))
	if fullBlocks != 20 {
		t.Errorf("expected 20 full blocks for clamped 100%%, got %d", fullBlocks)
	}
}

func TestGaugeClampNegative(t *testing.T) {
	g := NewGauge(DefaultGaugeStyle())
	g.style.ShowPercent = true
	result := g.Render(-10, 100, 20)
	stripped := gaugeTestStrip(result)
	if !strings.Contains(stripped, "0%") {
		t.Errorf("expected clamped to 0%%, got %q", stripped)
	}
}

func TestGaugeRenderMulti(t *testing.T) {
	g := NewGauge(DefaultGaugeStyle())
	gauges := []GaugeData{
		{Label: "CPU", Value: 30, MaxValue: 100},
		{Label: "RAM", Value: 60, MaxValue: 100},
		{Label: "Disk", Value: 85, MaxValue: 100},
	}
	result := g.RenderMulti(gauges, 20)
	lines := strings.Split(result, "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(lines))
	}

	// Verify labels are aligned (all have same label width = 5 = "Disk" + 1).
	for i, line := range lines {
		stripped := gaugeTestStrip(line)
		label := gauges[i].Label
		if !strings.HasPrefix(stripped, label) {
			t.Errorf("line %d: expected prefix %q, got %q", i, label, stripped)
		}
	}
}

func TestGaugeRenderMultiAlignment(t *testing.T) {
	g := NewGauge(DefaultGaugeStyle())
	g.style.ShowPercent = false
	gauges := []GaugeData{
		{Label: "A", Value: 50, MaxValue: 100},
		{Label: "BB", Value: 50, MaxValue: 100},
		{Label: "CCC", Value: 50, MaxValue: 100},
	}
	result := g.RenderMulti(gauges, 10)
	lines := strings.Split(result, "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(lines))
	}

	// All label areas should be 4 chars (max label "CCC" = 3 + 1 spacing).
	for i, line := range lines {
		stripped := gaugeTestStrip(line)
		// The bar starts at position 4 for all lines.
		if len([]rune(stripped)) < 4 {
			t.Errorf("line %d too short: %q", i, stripped)
			continue
		}
		// Check that bars start at the same position.
		labelArea := string([]rune(stripped)[:4])
		expected := gaugePadRight(gauges[i].Label, 4)
		if labelArea != expected {
			t.Errorf("line %d: label area %q != expected %q", i, labelArea, expected)
		}
	}
}

func TestGaugeVariousWidths(t *testing.T) {
	widths := []int{10, 20, 40, 80}
	for _, w := range widths {
		t.Run(strings.Replace(strings.Repeat("w", w), strings.Repeat("w", w), strings.TrimSpace(strings.Repeat(" ", 0)+string(rune('0'+w%10))), 1), func(t *testing.T) {
			g := NewGauge(DefaultGaugeStyle())
			g.style.ShowPercent = false
			g.style.ShowValue = false
			result := g.Render(50, 100, w)
			stripped := gaugeTestStrip(result)
			runeCount := len([]rune(stripped))
			if runeCount != w {
				t.Errorf("width %d: expected %d visible chars, got %d in %q", w, w, runeCount, stripped)
			}
		})
	}
}

func TestGaugeZeroMaxValue(t *testing.T) {
	g := NewGauge(DefaultGaugeStyle())
	g.style.ShowPercent = true
	result := g.Render(50, 0, 20)
	stripped := gaugeTestStrip(result)
	// maxValue=0 should result in 0% fill.
	if !strings.Contains(stripped, "0%") {
		t.Errorf("expected 0%% for maxValue=0, got %q", stripped)
	}
}

func TestGaugeEmptyMulti(t *testing.T) {
	g := NewGauge(DefaultGaugeStyle())
	result := g.RenderMulti(nil, 20)
	if result != "" {
		t.Errorf("expected empty string for nil gauges, got %q", result)
	}
}

func TestGaugeNoLabelNoPercent(t *testing.T) {
	style := DefaultGaugeStyle()
	style.ShowPercent = false
	style.ShowValue = false
	style.Label = ""
	g := NewGauge(style)
	result := g.Render(50, 100, 10)
	stripped := gaugeTestStrip(result)
	// Should be exactly 10 characters (bar only).
	if len([]rune(stripped)) != 10 {
		t.Errorf("expected 10 visible chars, got %d in %q", len([]rune(stripped)), stripped)
	}
}

func TestGaugeBothPercentAndValue(t *testing.T) {
	style := DefaultGaugeStyle()
	style.ShowPercent = true
	style.ShowValue = true
	g := NewGauge(style)
	result := g.Render(7.3, 10.0, 20)
	stripped := gaugeTestStrip(result)
	if !strings.Contains(stripped, "73%") {
		t.Errorf("expected percent label, got %q", stripped)
	}
	if !strings.Contains(stripped, "7.3/10.0") {
		t.Errorf("expected value label, got %q", stripped)
	}
}

func TestGaugeContainsResetSequences(t *testing.T) {
	g := NewGauge(DefaultGaugeStyle())
	result := g.Render(50, 100, 20)
	// Every colored section should end with a reset.
	if !strings.Contains(result, "\x1b[0m") {
		t.Error("expected ANSI reset sequences in output")
	}
}

func TestGaugeParseHexColor(t *testing.T) {
	tests := []struct {
		input string
		r, g, b uint8
		ok    bool
	}{
		{"#4CAF50", 76, 175, 80, true},
		{"4CAF50", 76, 175, 80, true},
		{"#FF9800", 255, 152, 0, true},
		{"#F44336", 244, 67, 54, true},
		{"invalid", 0, 0, 0, false},
		{"#FFF", 0, 0, 0, false},
		{"", 0, 0, 0, false},
	}
	for _, tc := range tests {
		r, g, b, ok := gaugeParseHexColor(tc.input)
		if ok != tc.ok || r != tc.r || g != tc.g || b != tc.b {
			t.Errorf("gaugeParseHexColor(%q) = (%d,%d,%d,%v), want (%d,%d,%d,%v)",
				tc.input, r, g, b, ok, tc.r, tc.g, tc.b, tc.ok)
		}
	}
}
