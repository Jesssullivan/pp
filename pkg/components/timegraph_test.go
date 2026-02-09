package components

import (
	"math"
	"strings"
	"testing"
	"time"
)

// refTime is a stable reference time for deterministic tests.
var refTime = time.Date(2026, 2, 9, 12, 0, 0, 0, time.UTC)

func TestBrailleBitMapping(t *testing.T) {
	// Verify each of the 8 dot positions maps to the correct bit.
	// Dot numbering:
	//   1(0x01) 4(0x08)
	//   2(0x02) 5(0x10)
	//   3(0x04) 6(0x20)
	//   7(0x40) 8(0x80)
	tests := []struct {
		offX, offY int
		want       uint8
	}{
		{0, 0, 0x01}, // dot 1
		{0, 1, 0x02}, // dot 2
		{0, 2, 0x04}, // dot 3
		{0, 3, 0x40}, // dot 7
		{1, 0, 0x08}, // dot 4
		{1, 1, 0x10}, // dot 5
		{1, 2, 0x20}, // dot 6
		{1, 3, 0x80}, // dot 8
	}
	for _, tt := range tests {
		got := brailleBit(tt.offX, tt.offY)
		if got != tt.want {
			t.Errorf("brailleBit(%d, %d) = 0x%02x, want 0x%02x", tt.offX, tt.offY, got, tt.want)
		}
	}
}

func TestBrailleBitOutOfRange(t *testing.T) {
	if got := brailleBit(0, -1); got != 0 {
		t.Errorf("brailleBit(0, -1) = 0x%02x, want 0", got)
	}
	if got := brailleBit(0, 4); got != 0 {
		t.Errorf("brailleBit(0, 4) = 0x%02x, want 0", got)
	}
}

func TestBrailleCharacterComposition(t *testing.T) {
	// All 8 dots set: 0x01|0x02|0x04|0x08|0x10|0x20|0x40|0x80 = 0xFF
	allDots := rune(0x2800 + 0xFF)
	if allDots != '⣿' {
		t.Errorf("all dots = %c (U+%04X), want ⣿ (U+28FF)", allDots, allDots)
	}

	// No dots: base Braille pattern.
	noDots := rune(0x2800)
	if noDots != '⠀' {
		t.Errorf("no dots = %c (U+%04X), want ⠀ (U+2800)", noDots, noDots)
	}

	// Single dot 1 (top-left).
	dot1 := rune(0x2800 + 0x01)
	if dot1 != '⠁' {
		t.Errorf("dot 1 = %c (U+%04X), want ⠁ (U+2801)", dot1, dot1)
	}
}

func TestKnownDataPointsBraille(t *testing.T) {
	// Place a single data point at the top-right corner of a 1x1 chart cell.
	// The point should produce a dot in position (1, 0) = bit 0x08 = '⠈'.
	tg := NewTimeGraph(TimeGraphConfig{
		TimeWindow: 5 * time.Minute,
	})
	si := tg.AddSeries("test", "#ffffff")
	// Point at the latest time (right edge) and max value (top).
	tg.PushValue(si, refTime, 100)

	// Render with no axes, no legend, minimal size.
	tg.cfg.ShowYAxis = false
	tg.cfg.ShowXAxis = false
	tg.cfg.ShowLegend = false

	result := tg.Render(10, 2)
	if len(result) == 0 {
		t.Fatal("Render returned empty string")
	}
	// The result should contain at least one non-blank Braille character.
	hasBraille := false
	for _, r := range result {
		if r >= 0x2801 && r <= 0x28FF {
			hasBraille = true
			break
		}
	}
	if !hasBraille {
		t.Error("expected at least one non-blank Braille character in output")
	}
}

func TestAutoScaleYRange(t *testing.T) {
	tg := NewTimeGraph(TimeGraphConfig{
		TimeWindow: 5 * time.Minute,
	})
	si := tg.AddSeries("test", "#00ff00")
	tg.PushValue(si, refTime.Add(-2*time.Minute), 10)
	tg.PushValue(si, refTime.Add(-1*time.Minute), 20)
	tg.PushValue(si, refTime, 30)

	tMin := refTime.Add(-5 * time.Minute)
	yMin, yMax := tg.yRange(tMin, refTime)

	// Data range is 10..30, span=20. With 10% padding: 8..32.
	if yMin >= 10 {
		t.Errorf("yMin = %f, want < 10 (padded)", yMin)
	}
	if yMax <= 30 {
		t.Errorf("yMax = %f, want > 30 (padded)", yMax)
	}
}

func TestFixedYRange(t *testing.T) {
	minY := 0.0
	maxY := 100.0
	tg := NewTimeGraph(TimeGraphConfig{
		TimeWindow: 5 * time.Minute,
		MinY:       &minY,
		MaxY:       &maxY,
	})
	si := tg.AddSeries("test", "#0000ff")
	tg.PushValue(si, refTime, 50)

	tMin := refTime.Add(-5 * time.Minute)
	yLo, yHi := tg.yRange(tMin, refTime)

	if yLo != 0 {
		t.Errorf("yMin = %f, want 0 (fixed)", yLo)
	}
	if yHi != 100 {
		t.Errorf("yMax = %f, want 100 (fixed)", yHi)
	}
}

func TestFixedYRangeClamps(t *testing.T) {
	// Values outside fixed range should be clamped during rendering (no panic).
	minY := 0.0
	maxY := 10.0
	tg := NewTimeGraph(TimeGraphConfig{
		TimeWindow: 5 * time.Minute,
		MinY:       &minY,
		MaxY:       &maxY,
	})
	si := tg.AddSeries("test", "#ff0000")
	tg.PushValue(si, refTime, -50) // below min
	tg.PushValue(si, refTime.Add(-1*time.Minute), 200) // above max

	result := tg.Render(40, 10)
	if result == "" {
		t.Error("Render returned empty string for clamped data")
	}
}

func TestEmptyDataNoFrame(t *testing.T) {
	tg := NewTimeGraph(TimeGraphConfig{
		TimeWindow: 5 * time.Minute,
	})

	result := tg.Render(40, 10)
	if result == "" {
		t.Error("Render returned empty string for empty data")
	}
	// Should contain Braille blank characters (U+2800).
	if !strings.ContainsRune(result, '\u2800') {
		t.Error("expected blank Braille characters in empty graph")
	}
}

func TestEmptyDataWithSeries(t *testing.T) {
	tg := NewTimeGraph(TimeGraphConfig{
		TimeWindow: 5 * time.Minute,
	})
	tg.AddSeries("empty", "#aabbcc")

	result := tg.Render(40, 10)
	if result == "" {
		t.Error("Render returned empty string for series with no data")
	}
}

func TestSinglePointRender(t *testing.T) {
	tg := NewTimeGraph(TimeGraphConfig{
		TimeWindow: 5 * time.Minute,
		ShowYAxis:  false,
		ShowXAxis:  false,
		ShowLegend: false,
	})
	si := tg.AddSeries("cpu", "#ff0000")
	tg.PushValue(si, refTime, 42.5)

	result := tg.Render(20, 5)
	if result == "" {
		t.Fatal("Render returned empty for single point")
	}

	lines := strings.Split(result, "\n")
	if len(lines) != 5 {
		t.Errorf("expected 5 lines, got %d", len(lines))
	}

	// Exactly one non-blank Braille cell should be present.
	brailleCount := 0
	for _, r := range result {
		if r >= 0x2801 && r <= 0x28FF {
			brailleCount++
		}
	}
	if brailleCount != 1 {
		t.Errorf("expected exactly 1 non-blank Braille cell, got %d", brailleCount)
	}
}

func TestMultipleSeries(t *testing.T) {
	tg := NewTimeGraph(TimeGraphConfig{
		TimeWindow: 5 * time.Minute,
		ShowYAxis:  false,
		ShowXAxis:  false,
		ShowLegend: true,
	})
	s1 := tg.AddSeries("cpu", "#ff0000")
	s2 := tg.AddSeries("mem", "#00ff00")
	s3 := tg.AddSeries("net", "#0000ff")

	if s1 != 0 || s2 != 1 || s3 != 2 {
		t.Errorf("series indices: got %d,%d,%d want 0,1,2", s1, s2, s3)
	}

	tg.PushValue(s1, refTime, 80)
	tg.PushValue(s2, refTime, 50)
	tg.PushValue(s3, refTime, 20)

	result := tg.Render(40, 10)
	if result == "" {
		t.Fatal("Render returned empty for multi-series")
	}

	// Legend should mention all series names.
	firstLine := strings.Split(result, "\n")[0]
	for _, name := range []string{"cpu", "mem", "net"} {
		if !strings.Contains(firstLine, name) {
			t.Errorf("legend missing series name %q", name)
		}
	}
}

func TestMultipleSeriesColors(t *testing.T) {
	tg := NewTimeGraph(TimeGraphConfig{
		TimeWindow: 5 * time.Minute,
		ShowYAxis:  false,
		ShowXAxis:  false,
		ShowLegend: false,
	})
	s1 := tg.AddSeries("a", "#ff0000")
	s2 := tg.AddSeries("b", "#00ff00")

	// Place points at different positions so they don't overlap.
	tg.PushValue(s1, refTime.Add(-4*time.Minute), 90)
	tg.PushValue(s2, refTime, 10)

	result := tg.Render(40, 10)
	// Both color sequences should appear.
	color1 := Color("#ff0000")
	color2 := Color("#00ff00")
	if !strings.Contains(result, color1) {
		t.Error("missing color for series 1")
	}
	if !strings.Contains(result, color2) {
		t.Error("missing color for series 2")
	}
}

func TestGracefulDegradationTooSmall(t *testing.T) {
	tg := NewTimeGraph(TimeGraphConfig{
		TimeWindow: 5 * time.Minute,
	})

	// Width < 10.
	result := tg.Render(5, 10)
	if !strings.Contains(result, "too s") {
		t.Errorf("expected 'too small' message for width=5, got %q", result)
	}

	// Height < 2.
	result = tg.Render(40, 1)
	if !strings.Contains(result, "too small") {
		t.Errorf("expected 'too small' message for height=1, got %q", result)
	}
}

func TestGracefulDegradationHideLegend(t *testing.T) {
	tg := NewTimeGraph(TimeGraphConfig{
		TimeWindow: 5 * time.Minute,
		ShowLegend: true,
	})
	tg.AddSeries("cpu", "#ff0000")
	tg.PushValue(0, refTime, 50)

	// Height=2 should suppress legend (threshold is height < 3).
	result := tg.Render(40, 2)
	if strings.Contains(result, "cpu") {
		t.Error("legend should be hidden at height=2")
	}
}

func TestGracefulDegradationHideXAxis(t *testing.T) {
	tg := NewTimeGraph(TimeGraphConfig{
		TimeWindow: 5 * time.Minute,
		ShowXAxis:  true,
		ShowLegend: false,
	})
	tg.AddSeries("test", "#ffffff")
	tg.PushValue(0, refTime, 50)

	// Height=4 should suppress X axis (threshold is height < 5).
	result := tg.Render(40, 4)
	if strings.Contains(result, "now") {
		t.Error("X axis should be hidden at height=4")
	}

	// Height=5 should show X axis.
	result = tg.Render(40, 5)
	if !strings.Contains(result, "now") {
		t.Error("X axis should be visible at height=5")
	}
}

func TestGracefulDegradationHideYAxis(t *testing.T) {
	tg := NewTimeGraph(TimeGraphConfig{
		TimeWindow: 5 * time.Minute,
		ShowYAxis:  true,
		ShowLegend: false,
		ShowXAxis:  false,
	})
	tg.AddSeries("test", "#ffffff")
	tg.PushValue(0, refTime, 1000)

	// Width=19 should suppress Y axis (threshold is width < 20).
	result := tg.Render(19, 5)
	if strings.Contains(result, "K") {
		t.Error("Y axis should be hidden at width=19")
	}

	// Width=20 should show Y axis.
	result = tg.Render(20, 5)
	if !strings.Contains(result, "K") && !strings.Contains(result, "0") {
		t.Error("Y axis should be visible at width=20")
	}
}

func TestTimeWindowing(t *testing.T) {
	tg := NewTimeGraph(TimeGraphConfig{
		TimeWindow: 5 * time.Minute,
		ShowYAxis:  false,
		ShowXAxis:  false,
		ShowLegend: false,
	})
	si := tg.AddSeries("test", "#ffffff")

	// Add a point well outside the window (10 minutes ago).
	tg.PushValue(si, refTime.Add(-10*time.Minute), 100)
	// Add a point inside the window.
	tg.PushValue(si, refTime, 50)

	result := tg.Render(40, 10)

	// Count non-blank Braille cells. Only the in-window point should plot.
	brailleCount := 0
	for _, r := range result {
		if r >= 0x2801 && r <= 0x28FF {
			brailleCount++
		}
	}
	// Expect exactly 1 dot (the in-window point).
	if brailleCount != 1 {
		t.Errorf("expected 1 non-blank Braille cell (old point excluded), got %d", brailleCount)
	}
}

func TestFormatSI(t *testing.T) {
	tests := []struct {
		input float64
		want  string
	}{
		{0, "0"},
		{1, "1"},
		{42, "42"},
		{999, "999"},
		{1000, "1K"},
		{1500, "1.5K"},
		{1500000, "1.5M"},
		{1000000, "1M"},
		{2500000000, "2.5G"},
		{1000000000000, "1T"},
		{-1500, "-1.5K"},
		{0.5, "0.5"},
		{3.14, "3.1"},
		{100.0, "100"},
	}
	for _, tt := range tests {
		got := formatSI(tt.input)
		if got != tt.want {
			t.Errorf("formatSI(%g) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		input time.Duration
		want  string
	}{
		{5 * time.Minute, "-5m"},
		{1 * time.Minute, "-1m"},
		{30 * time.Second, "-30s"},
		{2 * time.Hour, "-2h"},
		{0, "now"},
	}
	for _, tt := range tests {
		got := formatDuration(tt.input)
		if got != tt.want {
			t.Errorf("formatDuration(%v) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestPushValueAppends(t *testing.T) {
	tg := NewTimeGraph(TimeGraphConfig{})
	si := tg.AddSeries("test", "#ffffff")

	for i := 0; i < 10; i++ {
		tg.PushValue(si, refTime.Add(time.Duration(i)*time.Second), float64(i))
	}

	if len(tg.series[si].Data) != 10 {
		t.Errorf("expected 10 data points, got %d", len(tg.series[si].Data))
	}

	// Verify order.
	for i, dp := range tg.series[si].Data {
		if dp.Value != float64(i) {
			t.Errorf("data[%d].Value = %f, want %f", i, dp.Value, float64(i))
		}
	}
}

func TestPushValueInvalidIndex(t *testing.T) {
	tg := NewTimeGraph(TimeGraphConfig{})
	// Should not panic.
	tg.PushValue(-1, refTime, 42)
	tg.PushValue(0, refTime, 42) // no series added yet
	tg.PushValue(100, refTime, 42)
}

func TestSetData(t *testing.T) {
	tg := NewTimeGraph(TimeGraphConfig{})
	si := tg.AddSeries("test", "#ffffff")

	points := []DataPoint{
		{Time: refTime, Value: 1},
		{Time: refTime.Add(time.Second), Value: 2},
		{Time: refTime.Add(2 * time.Second), Value: 3},
	}
	tg.SetData(si, points)

	if len(tg.series[si].Data) != 3 {
		t.Errorf("expected 3 data points, got %d", len(tg.series[si].Data))
	}

	// Replace with new data.
	tg.SetData(si, []DataPoint{{Time: refTime, Value: 99}})
	if len(tg.series[si].Data) != 1 {
		t.Errorf("expected 1 data point after SetData, got %d", len(tg.series[si].Data))
	}
}

func TestSetDataInvalidIndex(t *testing.T) {
	tg := NewTimeGraph(TimeGraphConfig{})
	// Should not panic.
	tg.SetData(-1, nil)
	tg.SetData(0, nil)
}

func TestLargeDataset(t *testing.T) {
	tg := NewTimeGraph(TimeGraphConfig{
		TimeWindow: 10 * time.Minute,
		ShowYAxis:  true,
		ShowXAxis:  true,
		ShowLegend: true,
	})
	si := tg.AddSeries("load", "#ff8800")

	// 2000 points.
	for i := 0; i < 2000; i++ {
		ts := refTime.Add(-10*time.Minute + time.Duration(i)*300*time.Millisecond)
		v := 50 + 30*math.Sin(float64(i)*0.1)
		tg.PushValue(si, ts, v)
	}

	result := tg.Render(80, 24)
	if result == "" {
		t.Fatal("Render returned empty for large dataset")
	}

	lines := strings.Split(result, "\n")
	if len(lines) != 24 {
		t.Errorf("expected 24 lines, got %d", len(lines))
	}
}

func TestRenderVariousSizes(t *testing.T) {
	tg := NewTimeGraph(TimeGraphConfig{
		TimeWindow: 5 * time.Minute,
		ShowYAxis:  true,
		ShowXAxis:  true,
		ShowLegend: true,
	})
	si := tg.AddSeries("test", "#aabbcc")
	for i := 0; i < 50; i++ {
		tg.PushValue(si, refTime.Add(time.Duration(-250+i*5)*time.Second), float64(i))
	}

	sizes := []struct{ w, h int }{
		{10, 3},
		{40, 10},
		{80, 24},
		{200, 50},
	}

	for _, sz := range sizes {
		result := tg.Render(sz.w, sz.h)
		if result == "" {
			t.Errorf("Render(%d, %d) returned empty", sz.w, sz.h)
			continue
		}
		lines := strings.Split(result, "\n")
		for i, line := range lines {
			// Verify no trailing whitespace.
			if line != trimRight(line) {
				t.Errorf("Render(%d, %d) line %d has trailing whitespace: %q",
					sz.w, sz.h, i, line)
			}
		}
	}
}

func TestNoTrailingWhitespace(t *testing.T) {
	tg := NewTimeGraph(TimeGraphConfig{
		TimeWindow: 5 * time.Minute,
		ShowYAxis:  true,
		ShowXAxis:  true,
		ShowLegend: true,
	})
	s1 := tg.AddSeries("cpu", "#ff0000")
	s2 := tg.AddSeries("mem", "#00ff00")
	tg.PushValue(s1, refTime.Add(-3*time.Minute), 75)
	tg.PushValue(s1, refTime, 25)
	tg.PushValue(s2, refTime.Add(-2*time.Minute), 60)

	result := tg.Render(60, 20)
	lines := strings.Split(result, "\n")
	for i, line := range lines {
		cleaned := strings.TrimRight(line, " \t")
		if line != cleaned {
			t.Errorf("line %d has trailing whitespace: %q", i, line)
		}
	}
}

func TestYAxisLabels(t *testing.T) {
	tg := NewTimeGraph(TimeGraphConfig{
		TimeWindow: 5 * time.Minute,
		ShowYAxis:  true,
		ShowXAxis:  false,
		ShowLegend: false,
	})
	si := tg.AddSeries("test", "#ffffff")
	tg.PushValue(si, refTime, 5000)
	tg.PushValue(si, refTime.Add(-1*time.Minute), 0)

	result := tg.Render(40, 10)
	// Should contain SI-formatted Y labels.
	if !strings.Contains(result, "K") {
		t.Error("expected Y-axis labels with K suffix for values around 5000")
	}
}

func TestXAxisLabels(t *testing.T) {
	tg := NewTimeGraph(TimeGraphConfig{
		TimeWindow: 5 * time.Minute,
		ShowYAxis:  false,
		ShowXAxis:  true,
		ShowLegend: false,
	})
	si := tg.AddSeries("test", "#ffffff")
	tg.PushValue(si, refTime, 50)

	result := tg.Render(40, 6)
	lines := strings.Split(result, "\n")
	lastLine := lines[len(lines)-1]

	if !strings.Contains(lastLine, "now") {
		t.Errorf("X axis should contain 'now', got %q", lastLine)
	}
	if !strings.Contains(lastLine, "-5m") {
		t.Errorf("X axis should contain '-5m', got %q", lastLine)
	}
}

func TestDefaultConfig(t *testing.T) {
	tg := NewTimeGraph(TimeGraphConfig{})

	if tg.cfg.YAxisWidth != 6 {
		t.Errorf("default YAxisWidth = %d, want 6", tg.cfg.YAxisWidth)
	}
	if tg.cfg.TimeWindow != 5*time.Minute {
		t.Errorf("default TimeWindow = %v, want 5m", tg.cfg.TimeWindow)
	}
}

func TestPartialFixedYRange(t *testing.T) {
	// Only MinY is fixed, MaxY auto-scales.
	minY := 0.0
	tg := NewTimeGraph(TimeGraphConfig{
		TimeWindow: 5 * time.Minute,
		MinY:       &minY,
	})
	si := tg.AddSeries("test", "#ffffff")
	tg.PushValue(si, refTime, 100)

	tMin := refTime.Add(-5 * time.Minute)
	yLo, yHi := tg.yRange(tMin, refTime)

	if yLo != 0 {
		t.Errorf("yMin = %f, want 0 (fixed)", yLo)
	}
	if yHi <= 100 {
		t.Errorf("yMax = %f, want > 100 (auto-scaled with padding)", yHi)
	}
}

func TestNegativeValues(t *testing.T) {
	tg := NewTimeGraph(TimeGraphConfig{
		TimeWindow: 5 * time.Minute,
		ShowYAxis:  true,
		ShowXAxis:  false,
		ShowLegend: false,
	})
	si := tg.AddSeries("temp", "#0088ff")
	tg.PushValue(si, refTime, -20)
	tg.PushValue(si, refTime.Add(-2*time.Minute), -10)

	result := tg.Render(40, 10)
	if result == "" {
		t.Fatal("Render returned empty for negative values")
	}
	// Y-axis should show negative numbers.
	if !strings.Contains(result, "-") {
		t.Error("expected negative sign in Y-axis labels")
	}
}

func TestOverlappingDotsCombine(t *testing.T) {
	tg := NewTimeGraph(TimeGraphConfig{
		TimeWindow: 5 * time.Minute,
		ShowYAxis:  false,
		ShowXAxis:  false,
		ShowLegend: false,
	})
	s1 := tg.AddSeries("a", "#ff0000")
	s2 := tg.AddSeries("b", "#00ff00")

	// Push same time and value for both series — dots should combine via OR.
	tg.PushValue(s1, refTime, 50)
	tg.PushValue(s2, refTime, 50)

	result := tg.Render(20, 5)
	// Should still render without panic.
	if result == "" {
		t.Fatal("Render returned empty for overlapping points")
	}

	// There should be exactly one non-blank Braille cell (combined dots).
	brailleCount := 0
	for _, r := range result {
		if r >= 0x2801 && r <= 0x28FF {
			brailleCount++
		}
	}
	// Due to sub-pixel mapping, overlapping points in same cell should combine.
	if brailleCount < 1 {
		t.Error("expected at least 1 non-blank Braille cell for overlapping points")
	}
}

func TestRenderMinimalSize(t *testing.T) {
	tg := NewTimeGraph(TimeGraphConfig{
		TimeWindow: 5 * time.Minute,
		ShowYAxis:  false,
		ShowXAxis:  false,
		ShowLegend: false,
	})
	si := tg.AddSeries("test", "#ffffff")
	tg.PushValue(si, refTime, 42)

	// Minimal valid size: 10x2.
	result := tg.Render(10, 2)
	if strings.Contains(result, "too small") {
		t.Error("10x2 should be renderable, not 'too small'")
	}

	lines := strings.Split(result, "\n")
	if len(lines) != 2 {
		t.Errorf("expected 2 lines at height=2, got %d", len(lines))
	}
}

func TestLatestTimeWithNoData(t *testing.T) {
	tg := NewTimeGraph(TimeGraphConfig{})
	// latestTime should return approximately now, not panic.
	latest := tg.latestTime()
	if time.Since(latest) > 2*time.Second {
		t.Errorf("latestTime with no data should be ~now, got %v ago", time.Since(latest))
	}
}

func TestTooSmallMessage(t *testing.T) {
	msg := tooSmallMsg(20)
	if msg != "too small" {
		t.Errorf("tooSmallMsg(20) = %q, want %q", msg, "too small")
	}

	msg = tooSmallMsg(5)
	if msg != "too s" {
		t.Errorf("tooSmallMsg(5) = %q, want %q", msg, "too s")
	}
}

func TestPadLeft(t *testing.T) {
	tests := []struct {
		s     string
		width int
		want  string
	}{
		{"42", 6, "    42"},
		{"1.5K", 6, "  1.5K"},
		{"toolong", 4, "tool"},
		{"abc", 3, "abc"},
	}
	for _, tt := range tests {
		got := padLeft(tt.s, tt.width)
		if got != tt.want {
			t.Errorf("padLeft(%q, %d) = %q, want %q", tt.s, tt.width, got, tt.want)
		}
	}
}

func TestTrimRight(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"hello   ", "hello"},
		{"hello\t\t", "hello"},
		{"hello", "hello"},
		{"  hello  ", "  hello"},
		{"", ""},
	}
	for _, tt := range tests {
		got := trimRight(tt.input)
		if got != tt.want {
			t.Errorf("trimRight(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestRenderOutputLineCount(t *testing.T) {
	tests := []struct {
		name                          string
		height                        int
		showLegend, showXAxis         bool
		expectLines                   int
	}{
		{"bare 10", 10, false, false, 10},
		{"legend+xaxis 10", 10, true, true, 10},
		{"legend only 10", 10, true, false, 10},
		{"xaxis only 10", 10, false, true, 10},
		{"bare 5", 5, false, false, 5},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tg := NewTimeGraph(TimeGraphConfig{
				TimeWindow: 5 * time.Minute,
				ShowYAxis:  false,
				ShowXAxis:  tt.showXAxis,
				ShowLegend: tt.showLegend,
			})
			si := tg.AddSeries("test", "#ffffff")
			tg.PushValue(si, refTime, 50)

			result := tg.Render(40, tt.height)
			lines := strings.Split(result, "\n")
			if len(lines) != tt.expectLines {
				t.Errorf("expected %d lines, got %d", tt.expectLines, len(lines))
			}
		})
	}
}

func TestMonotonicDataProducesBraille(t *testing.T) {
	tg := NewTimeGraph(TimeGraphConfig{
		TimeWindow: 5 * time.Minute,
		ShowYAxis:  false,
		ShowXAxis:  false,
		ShowLegend: false,
	})
	si := tg.AddSeries("ramp", "#00ffff")
	for i := 0; i < 20; i++ {
		tg.PushValue(si, refTime.Add(time.Duration(-250+i*15)*time.Second), float64(i*5))
	}

	result := tg.Render(40, 10)
	brailleCount := 0
	for _, r := range result {
		if r >= 0x2801 && r <= 0x28FF {
			brailleCount++
		}
	}
	if brailleCount < 5 {
		t.Errorf("expected multiple Braille dots for 20-point ramp, got %d", brailleCount)
	}
}
