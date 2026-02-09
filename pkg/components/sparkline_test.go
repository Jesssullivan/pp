package components

import (
	"strings"
	"testing"
)

// sparkTestStrip removes ANSI escapes for asserting visible content.
func sparkTestStrip(s string) string {
	return sparkStripANSI(s)
}

func TestSparklineConstantData(t *testing.T) {
	s := NewSparkline(DefaultSparklineStyle())
	data := []float64{5, 5, 5, 5, 5}
	result := s.Render(data, 5)
	stripped := sparkTestStrip(result)
	// All values are equal, so all blocks should be the same.
	runes := []rune(stripped)
	if len(runes) == 0 {
		t.Fatal("expected non-empty sparkline")
	}
	first := runes[0]
	for i, r := range runes {
		if r != first {
			t.Errorf("position %d: expected %q, got %q (all should be same for constant data)", i, string(first), string(r))
		}
	}
}

func TestSparklineAscendingData(t *testing.T) {
	s := NewSparkline(DefaultSparklineStyle())
	data := []float64{0, 1, 2, 3, 4, 5, 6, 7}
	result := s.Render(data, 8)
	stripped := sparkTestStrip(result)
	runes := []rune(stripped)
	if len(runes) != 8 {
		t.Fatalf("expected 8 chars, got %d: %q", len(runes), stripped)
	}
	// Each character should be >= the previous one.
	for i := 1; i < len(runes); i++ {
		if runes[i] < runes[i-1] {
			t.Errorf("ascending: position %d (%q) < position %d (%q)",
				i, string(runes[i]), i-1, string(runes[i-1]))
		}
	}
}

func TestSparklineDescendingData(t *testing.T) {
	s := NewSparkline(DefaultSparklineStyle())
	data := []float64{7, 6, 5, 4, 3, 2, 1, 0}
	result := s.Render(data, 8)
	stripped := sparkTestStrip(result)
	runes := []rune(stripped)
	if len(runes) != 8 {
		t.Fatalf("expected 8 chars, got %d: %q", len(runes), stripped)
	}
	// Each character should be <= the previous one.
	for i := 1; i < len(runes); i++ {
		if runes[i] > runes[i-1] {
			t.Errorf("descending: position %d (%q) > position %d (%q)",
				i, string(runes[i]), i-1, string(runes[i-1]))
		}
	}
}

func TestSparklineAutoScaling(t *testing.T) {
	s := NewSparkline(DefaultSparklineStyle())
	data := []float64{0, 100}
	result := s.Render(data, 2)
	stripped := sparkTestStrip(result)
	runes := []rune(stripped)
	if len(runes) != 2 {
		t.Fatalf("expected 2 chars, got %d: %q", len(runes), stripped)
	}
	// First should be lowest block, last should be highest.
	if runes[0] != '\u2581' {
		t.Errorf("expected lowest block ▁ for min value, got %q", string(runes[0]))
	}
	if runes[1] != '\u2588' {
		t.Errorf("expected highest block █ for max value, got %q", string(runes[1]))
	}
}

func TestSparklineFixedRange(t *testing.T) {
	style := DefaultSparklineStyle()
	minY := 0.0
	maxY := 200.0
	style.MinY = &minY
	style.MaxY = &maxY
	s := NewSparkline(style)
	// With fixed range 0-200, value 100 should be at midpoint.
	data := []float64{100}
	result := s.Render(data, 1)
	stripped := sparkTestStrip(result)
	runes := []rune(stripped)
	if len(runes) != 1 {
		t.Fatalf("expected 1 char, got %d", len(runes))
	}
	// Midpoint (0.5 * 7 = 3.5, rounds to 4) -> index 4 = ▅
	expected := '\u2584' // ▄ (index 3) or ▅ (index 4) depending on rounding
	if runes[0] != '\u2584' && runes[0] != '\u2585' {
		t.Errorf("expected mid-level block (▄ or ▅), got %q (U+%04X)", string(runes[0]), runes[0])
	}
	_ = expected
}

func TestSparklineFewerPointsThanWidth(t *testing.T) {
	s := NewSparkline(DefaultSparklineStyle())
	data := []float64{1, 2, 3}
	result := s.Render(data, 10)
	stripped := sparkTestStrip(result)
	runes := []rune(stripped)
	// Should render only 3 characters (no padding by default).
	if len(runes) != 3 {
		t.Errorf("expected 3 chars for 3 data points, got %d: %q", len(runes), stripped)
	}
}

func TestSparklineMorePointsThanWidth(t *testing.T) {
	s := NewSparkline(DefaultSparklineStyle())
	data := []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	result := s.Render(data, 5)
	stripped := sparkTestStrip(result)
	runes := []rune(stripped)
	// Should take last 5 data points.
	if len(runes) != 5 {
		t.Errorf("expected 5 chars, got %d: %q", len(runes), stripped)
	}
	// Last 5 points are 6,7,8,9,10 - should be ascending.
	for i := 1; i < len(runes); i++ {
		if runes[i] < runes[i-1] {
			t.Errorf("expected ascending from last 5 points, position %d < %d", i, i-1)
		}
	}
}

func TestSparklineMinMaxLabels(t *testing.T) {
	style := DefaultSparklineStyle()
	style.ShowMinMax = true
	s := NewSparkline(style)
	data := []float64{0, 50, 100}
	result := s.Render(data, 3)
	stripped := sparkTestStrip(result)
	// Should contain "0" and "100".
	if !strings.Contains(stripped, "0") {
		t.Errorf("expected min label '0', got %q", stripped)
	}
	if !strings.Contains(stripped, "100") {
		t.Errorf("expected max label '100', got %q", stripped)
	}
}

func TestSparklineLabelPrefix(t *testing.T) {
	style := DefaultSparklineStyle()
	style.Label = "CPU"
	s := NewSparkline(style)
	data := []float64{1, 2, 3}
	result := s.Render(data, 3)
	stripped := sparkTestStrip(result)
	if !strings.HasPrefix(stripped, "CPU ") {
		t.Errorf("expected 'CPU ' prefix, got %q", stripped)
	}
}

func TestSparklineDeltaPositive(t *testing.T) {
	s := NewSparkline(DefaultSparklineStyle())
	data := []float64{10, 20}
	result := s.RenderWithDelta(data, 2)
	stripped := sparkTestStrip(result)
	if !strings.Contains(stripped, "\u2191") {
		t.Errorf("expected up arrow for positive delta, got %q", stripped)
	}
	// Delta should be 100%.
	if !strings.Contains(stripped, "100.0%") {
		t.Errorf("expected '100.0%%' delta, got %q", stripped)
	}
}

func TestSparklineDeltaNegative(t *testing.T) {
	s := NewSparkline(DefaultSparklineStyle())
	data := []float64{20, 10}
	result := s.RenderWithDelta(data, 2)
	stripped := sparkTestStrip(result)
	if !strings.Contains(stripped, "\u2193") {
		t.Errorf("expected down arrow for negative delta, got %q", stripped)
	}
	if !strings.Contains(stripped, "50.0%") {
		t.Errorf("expected '50.0%%' delta, got %q", stripped)
	}
}

func TestSparklineDeltaZero(t *testing.T) {
	s := NewSparkline(DefaultSparklineStyle())
	data := []float64{10, 10}
	result := s.RenderWithDelta(data, 2)
	stripped := sparkTestStrip(result)
	if !strings.Contains(stripped, "\u2192") {
		t.Errorf("expected right arrow for zero delta, got %q", stripped)
	}
	if !strings.Contains(stripped, "0.0%") {
		t.Errorf("expected '0.0%%' delta, got %q", stripped)
	}
}

func TestSparklineEmptyData(t *testing.T) {
	s := NewSparkline(DefaultSparklineStyle())
	result := s.Render(nil, 10)
	if result != "" {
		t.Errorf("expected empty string for nil data, got %q", result)
	}
	result = s.Render([]float64{}, 10)
	if result != "" {
		t.Errorf("expected empty string for empty data, got %q", result)
	}
}

func TestSparklineSingleDataPoint(t *testing.T) {
	s := NewSparkline(DefaultSparklineStyle())
	data := []float64{42}
	result := s.Render(data, 5)
	stripped := sparkTestStrip(result)
	if len([]rune(stripped)) != 1 {
		t.Errorf("expected 1 char for single data point, got %d: %q", len([]rune(stripped)), stripped)
	}
}

func TestSparklineVariousWidths(t *testing.T) {
	widths := []int{5, 10, 20, 40}
	data := make([]float64, 100)
	for i := range data {
		data[i] = float64(i)
	}
	for _, w := range widths {
		s := NewSparkline(DefaultSparklineStyle())
		result := s.Render(data, w)
		stripped := sparkTestStrip(result)
		runes := []rune(stripped)
		if len(runes) != w {
			t.Errorf("width %d: expected %d chars, got %d: %q", w, w, len(runes), stripped)
		}
	}
}

func TestSparklineColorOutput(t *testing.T) {
	s := NewSparkline(DefaultSparklineStyle())
	data := []float64{1, 2, 3}
	result := s.Render(data, 3)
	// Should contain the blue color code: #64B5F6 → rgb(100, 181, 246).
	if !strings.Contains(result, "38;2;100;181;246") {
		t.Errorf("expected blue color in output, got %q", result)
	}
	if !strings.Contains(result, "\x1b[0m") {
		t.Error("expected ANSI reset sequence in output")
	}
}

func TestSparklineRenderWithDeltaSinglePoint(t *testing.T) {
	s := NewSparkline(DefaultSparklineStyle())
	data := []float64{42}
	result := s.RenderWithDelta(data, 5)
	stripped := sparkTestStrip(result)
	// Single point should show right arrow and 0.0%.
	if !strings.Contains(stripped, "\u2192") {
		t.Errorf("expected right arrow for single data point delta, got %q", stripped)
	}
}

func TestSparklineRenderWithDeltaEmptyData(t *testing.T) {
	s := NewSparkline(DefaultSparklineStyle())
	result := s.RenderWithDelta(nil, 5)
	if result != "" {
		t.Errorf("expected empty string for nil data with delta, got %q", result)
	}
}

func TestSparklineLabelAndMinMax(t *testing.T) {
	style := DefaultSparklineStyle()
	style.Label = "MEM"
	style.ShowMinMax = true
	s := NewSparkline(style)
	data := []float64{0.5, 1.0}
	result := s.Render(data, 2)
	stripped := sparkTestStrip(result)
	// Should have label, min, sparkline, max.
	if !strings.HasPrefix(stripped, "MEM ") {
		t.Errorf("expected 'MEM ' prefix, got %q", stripped)
	}
	if !strings.Contains(stripped, "0.5") {
		t.Errorf("expected min '0.5' in output, got %q", stripped)
	}
	if !strings.Contains(stripped, "1") {
		t.Errorf("expected max '1' in output, got %q", stripped)
	}
}

func TestSparklineParseHexColor(t *testing.T) {
	tests := []struct {
		input   string
		r, g, b uint8
		ok      bool
	}{
		{"#64B5F6", 100, 181, 246, true},
		{"64B5F6", 100, 181, 246, true},
		{"#000000", 0, 0, 0, true},
		{"#FFFFFF", 255, 255, 255, true},
		{"invalid", 0, 0, 0, false},
		{"#FFF", 0, 0, 0, false},
		{"", 0, 0, 0, false},
	}
	for _, tc := range tests {
		r, g, b, ok := sparkParseHexColor(tc.input)
		if ok != tc.ok || r != tc.r || g != tc.g || b != tc.b {
			t.Errorf("sparkParseHexColor(%q) = (%d,%d,%d,%v), want (%d,%d,%d,%v)",
				tc.input, r, g, b, ok, tc.r, tc.g, tc.b, tc.ok)
		}
	}
}

func TestSparklineFixedMinOnly(t *testing.T) {
	style := DefaultSparklineStyle()
	minY := 0.0
	style.MinY = &minY
	// MaxY is nil, so auto-scale max from data.
	s := NewSparkline(style)
	data := []float64{50, 100}
	result := s.Render(data, 2)
	stripped := sparkTestStrip(result)
	runes := []rune(stripped)
	if len(runes) != 2 {
		t.Fatalf("expected 2 chars, got %d", len(runes))
	}
	// With min=0, max=100: 50 is midpoint, 100 is max.
	// 50 should map to mid-level, 100 to highest.
	if runes[1] != '\u2588' {
		t.Errorf("expected highest block for max value with fixed min, got %q", string(runes[1]))
	}
}
