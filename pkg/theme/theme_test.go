package theme

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"testing"
)

var thTestHexPattern = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

// --- Get / SetCurrent / Names ---

func TestGetDefault(t *testing.T) {
	th := Get("default")
	if th.Name != "default" {
		t.Errorf("Get(\"default\").Name = %q, want %q", th.Name, "default")
	}
	if th.Accent != "#7C3AED" {
		t.Errorf("Get(\"default\").Accent = %q, want %q", th.Accent, "#7C3AED")
	}
}

func TestGetGruvbox(t *testing.T) {
	th := Get("gruvbox")
	if th.Name != "gruvbox" {
		t.Errorf("Get(\"gruvbox\").Name = %q, want %q", th.Name, "gruvbox")
	}
	if th.Background != "#282828" {
		t.Errorf("Get(\"gruvbox\").Background = %q, want %q", th.Background, "#282828")
	}
	if th.Accent != "#fe8019" {
		t.Errorf("Get(\"gruvbox\").Accent = %q, want %q", th.Accent, "#fe8019")
	}
}

func TestGetUnknownFallsBackToDefault(t *testing.T) {
	th := Get("unknown-theme-xyz")
	def := Get("default")
	if th.Name != def.Name {
		t.Errorf("Get(\"unknown\") = %q, want %q (default)", th.Name, def.Name)
	}
	if th.Accent != def.Accent {
		t.Errorf("Get(\"unknown\").Accent = %q, want %q", th.Accent, def.Accent)
	}
}

func TestNames(t *testing.T) {
	names := Names()
	if len(names) != 6 {
		t.Fatalf("Names() returned %d themes, want 6", len(names))
	}

	expected := []string{"catppuccin", "default", "dracula", "gruvbox", "nord", "tokyo-night"}
	sort.Strings(expected)
	for i, name := range expected {
		if names[i] != name {
			t.Errorf("Names()[%d] = %q, want %q", i, names[i], name)
		}
	}
}

func TestSetCurrent(t *testing.T) {
	SetCurrent("gruvbox")
	if Current.Name != "gruvbox" {
		t.Errorf("after SetCurrent(\"gruvbox\"), Current.Name = %q", Current.Name)
	}
	if Current.Accent != "#fe8019" {
		t.Errorf("after SetCurrent(\"gruvbox\"), Current.Accent = %q", Current.Accent)
	}

	// Reset to default for other tests.
	SetCurrent("default")
}

// --- Built-in theme completeness ---

func TestAllThemesHaveRequiredFields(t *testing.T) {
	for _, name := range Names() {
		th := Get(name)
		t.Run(name, func(t *testing.T) {
			if th.Background == "" {
				t.Error("Background is empty")
			}
			if th.Foreground == "" {
				t.Error("Foreground is empty")
			}
			if th.Accent == "" {
				t.Error("Accent is empty")
			}
		})
	}
}

func TestAllThemesHaveValidHexColors(t *testing.T) {
	for _, name := range Names() {
		th := Get(name)
		t.Run(name, func(t *testing.T) {
			colors := map[string]string{
				"Background":      th.Background,
				"Foreground":      th.Foreground,
				"Dim":             th.Dim,
				"Accent":          th.Accent,
				"Border":          th.Border,
				"BorderFocus":     th.BorderFocus,
				"Title":           th.Title,
				"StatusOK":        th.StatusOK,
				"StatusWarn":      th.StatusWarn,
				"StatusError":     th.StatusError,
				"StatusUnknown":   th.StatusUnknown,
				"GaugeFilled":     th.GaugeFilled,
				"GaugeEmpty":      th.GaugeEmpty,
				"GaugeWarn":       th.GaugeWarn,
				"GaugeCrit":       th.GaugeCrit,
				"ChartLine":       th.ChartLine,
				"ChartFill":       th.ChartFill,
				"ChartGrid":       th.ChartGrid,
				"SearchHighlight": th.SearchHighlight,
				"HelpKey":         th.HelpKey,
				"HelpDesc":        th.HelpDesc,
			}
			for field, value := range colors {
				if !thTestHexPattern.MatchString(value) {
					t.Errorf("%s = %q is not valid #RRGGBB", field, value)
				}
			}
		})
	}
}

// --- 256-color fallback ---

func TestTo256ColorPureRed(t *testing.T) {
	// Pure red #ff0000 should map to 196 (cube: 5,0,0 -> 16 + 36*5 = 196).
	result := thTo256Color("#ff0000")
	if result != "196" {
		t.Errorf("thTo256Color(\"#ff0000\") = %q, want %q", result, "196")
	}
}

func TestTo256ColorPureGreen(t *testing.T) {
	// Pure green #00ff00 should map to 46 (cube: 0,5,0 -> 16 + 6*5 = 46).
	result := thTo256Color("#00ff00")
	if result != "46" {
		t.Errorf("thTo256Color(\"#00ff00\") = %q, want %q", result, "46")
	}
}

func TestTo256ColorGrayscale(t *testing.T) {
	// A mid-gray like #808080 should map to a grayscale index.
	result := thTo256Color("#808080")
	// #808080 = RGB(128,128,128). Grayscale avg = 128.
	// Nearest gray: (128-8+5)/10 = 12.5 -> 12 -> 232+12 = 244.
	// Cube nearest for 128 is level 3 (135), so cube index 16+36*3+6*3+3 = 16+108+18+3 = 145.
	// Cube RGB for 145: (135,135,135), distance = sqrt(3*49) ~ 12.12.
	// Gray 244 value: 8+12*10 = 128, distance = 0.
	// Gray should win.
	if result != "244" {
		t.Errorf("thTo256Color(\"#808080\") = %q, want %q", result, "244")
	}
}

func TestTo256ColorBlack(t *testing.T) {
	// #000000 should map to 16 (cube: 0,0,0 -> 16).
	result := thTo256Color("#000000")
	if result != "16" {
		t.Errorf("thTo256Color(\"#000000\") = %q, want %q", result, "16")
	}
}

func TestTo256ColorWhite(t *testing.T) {
	// #ffffff should map to 231 (cube: 5,5,5 -> 16 + 36*5 + 6*5 + 5 = 231).
	result := thTo256Color("#ffffff")
	if result != "231" {
		t.Errorf("thTo256Color(\"#ffffff\") = %q, want %q", result, "231")
	}
}

func TestNearestCubeIndexPrimaries(t *testing.T) {
	tests := []struct {
		r, g, b uint8
		want    int
	}{
		{255, 0, 0, 196},   // pure red
		{0, 255, 0, 46},    // pure green
		{0, 0, 255, 21},    // pure blue
		{0, 0, 0, 16},      // black
		{255, 255, 255, 231}, // white
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("(%d,%d,%d)", tt.r, tt.g, tt.b), func(t *testing.T) {
			got := thNearestCubeIndex(tt.r, tt.g, tt.b)
			if got != tt.want {
				t.Errorf("thNearestCubeIndex(%d,%d,%d) = %d, want %d", tt.r, tt.g, tt.b, got, tt.want)
			}
		})
	}
}

func TestAdaptConvertsColors(t *testing.T) {
	th := Get("default")
	adapted := Adapt(th, 8) // 8-bit color depth means 256 colors

	// All fields should be numeric strings, not hex.
	if strings.HasPrefix(adapted.Background, "#") {
		t.Errorf("Adapt with colorDepth=8 should convert Background, got %q", adapted.Background)
	}
	if strings.HasPrefix(adapted.Accent, "#") {
		t.Errorf("Adapt with colorDepth=8 should convert Accent, got %q", adapted.Accent)
	}
	if strings.HasPrefix(adapted.StatusOK, "#") {
		t.Errorf("Adapt with colorDepth=8 should convert StatusOK, got %q", adapted.StatusOK)
	}
}

func TestAdaptPreservesAt24Bit(t *testing.T) {
	th := Get("default")
	adapted := Adapt(th, 24)

	if adapted.Background != th.Background {
		t.Errorf("Adapt(24bit) changed Background: %q -> %q", th.Background, adapted.Background)
	}
	if adapted.Accent != th.Accent {
		t.Errorf("Adapt(24bit) changed Accent: %q -> %q", th.Accent, adapted.Accent)
	}
	if adapted.StatusError != th.StatusError {
		t.Errorf("Adapt(24bit) changed StatusError: %q -> %q", th.StatusError, adapted.StatusError)
	}
}

// --- TOML loading/saving ---

func TestLoadFromTOMLValid(t *testing.T) {
	data := []byte(`
name = "custom"

[base]
background = "#111111"
foreground = "#eeeeee"
dim = "#666666"
accent = "#ff0000"

[widget]
border = "#333333"
border_focus = "#ff0000"
title = "#eeeeee"

[status]
ok = "#00ff00"
warn = "#ffff00"
error = "#ff0000"
unknown = "#888888"

[gauge]
filled = "#00ff00"
empty = "#333333"
warn = "#ffff00"
crit = "#ff0000"

[chart]
line = "#ff0000"
fill = "#880000"
grid = "#333333"

[special]
search_highlight = "#ffff00"
help_key = "#ff0000"
help_desc = "#888888"
`)

	th, err := LoadFromTOML(data)
	if err != nil {
		t.Fatalf("LoadFromTOML() error: %v", err)
	}
	if th.Name != "custom" {
		t.Errorf("Name = %q, want %q", th.Name, "custom")
	}
	if th.Background != "#111111" {
		t.Errorf("Background = %q, want %q", th.Background, "#111111")
	}
	if th.StatusOK != "#00ff00" {
		t.Errorf("StatusOK = %q, want %q", th.StatusOK, "#00ff00")
	}
}

func TestLoadFromTOMLMissingFieldsError(t *testing.T) {
	// Missing the [status] section entirely.
	data := []byte(`
name = "incomplete"

[base]
background = "#111111"
foreground = "#eeeeee"
dim = "#666666"
accent = "#ff0000"

[widget]
border = "#333333"
border_focus = "#ff0000"
title = "#eeeeee"
`)

	_, err := LoadFromTOML(data)
	if err == nil {
		t.Error("LoadFromTOML() should return error for missing fields")
	}
}

func TestLoadFromTOMLInvalidHexColor(t *testing.T) {
	data := []byte(`
name = "badhex"

[base]
background = "not-a-color"
foreground = "#eeeeee"
dim = "#666666"
accent = "#ff0000"

[widget]
border = "#333333"
border_focus = "#ff0000"
title = "#eeeeee"

[status]
ok = "#00ff00"
warn = "#ffff00"
error = "#ff0000"
unknown = "#888888"

[gauge]
filled = "#00ff00"
empty = "#333333"
warn = "#ffff00"
crit = "#ff0000"

[chart]
line = "#ff0000"
fill = "#880000"
grid = "#333333"

[special]
search_highlight = "#ffff00"
help_key = "#ff0000"
help_desc = "#888888"
`)

	_, err := LoadFromTOML(data)
	if err == nil {
		t.Error("LoadFromTOML() should return error for invalid hex color")
	}
	if err != nil && !strings.Contains(err.Error(), "invalid hex color") {
		t.Errorf("error should mention invalid hex color, got: %v", err)
	}
}

func TestSaveToTOMLRoundtrip(t *testing.T) {
	original := Get("gruvbox")

	data, err := SaveToTOML(original)
	if err != nil {
		t.Fatalf("SaveToTOML() error: %v", err)
	}

	loaded, err := LoadFromTOML(data)
	if err != nil {
		t.Fatalf("LoadFromTOML(roundtrip) error: %v", err)
	}

	if loaded.Name != original.Name {
		t.Errorf("roundtrip Name: %q -> %q", original.Name, loaded.Name)
	}
	if loaded.Background != original.Background {
		t.Errorf("roundtrip Background: %q -> %q", original.Background, loaded.Background)
	}
	if loaded.Accent != original.Accent {
		t.Errorf("roundtrip Accent: %q -> %q", original.Accent, loaded.Accent)
	}
	if loaded.StatusOK != original.StatusOK {
		t.Errorf("roundtrip StatusOK: %q -> %q", original.StatusOK, loaded.StatusOK)
	}
	if loaded.GaugeCrit != original.GaugeCrit {
		t.Errorf("roundtrip GaugeCrit: %q -> %q", original.GaugeCrit, loaded.GaugeCrit)
	}
	if loaded.ChartLine != original.ChartLine {
		t.Errorf("roundtrip ChartLine: %q -> %q", original.ChartLine, loaded.ChartLine)
	}
	if loaded.SearchHighlight != original.SearchHighlight {
		t.Errorf("roundtrip SearchHighlight: %q -> %q", original.SearchHighlight, loaded.SearchHighlight)
	}
	if loaded.HelpKey != original.HelpKey {
		t.Errorf("roundtrip HelpKey: %q -> %q", original.HelpKey, loaded.HelpKey)
	}
}

// --- Apply helpers ---

func TestApplyStatusOK(t *testing.T) {
	th := Get("default")
	result := thApplyStatus("healthy", "ok", th)
	expected := fmt.Sprintf("\x1b[38;2;%d;%d;%dm%s\x1b[0m", 78, 201, 112, "healthy")
	if result != expected {
		t.Errorf("thApplyStatus(\"ok\") = %q, want %q", result, expected)
	}
}

func TestApplyStatusError(t *testing.T) {
	th := Get("default")
	result := thApplyStatus("down", "error", th)
	expected := fmt.Sprintf("\x1b[38;2;%d;%d;%dm%s\x1b[0m", 224, 108, 117, "down")
	if result != expected {
		t.Errorf("thApplyStatus(\"error\") = %q, want %q", result, expected)
	}
}

func TestApplyGaugeNormal(t *testing.T) {
	th := Get("default")
	filled, empty := thApplyGauge(0.5, th)
	if filled != th.GaugeFilled {
		t.Errorf("thApplyGauge(0.5) filled = %q, want %q", filled, th.GaugeFilled)
	}
	if empty != th.GaugeEmpty {
		t.Errorf("thApplyGauge(0.5) empty = %q, want %q", empty, th.GaugeEmpty)
	}
}

func TestApplyGaugeWarning(t *testing.T) {
	th := Get("default")
	filled, _ := thApplyGauge(0.75, th)
	if filled != th.GaugeWarn {
		t.Errorf("thApplyGauge(0.75) filled = %q, want %q", filled, th.GaugeWarn)
	}
}

func TestApplyGaugeCritical(t *testing.T) {
	th := Get("default")
	filled, _ := thApplyGauge(0.95, th)
	if filled != th.GaugeCrit {
		t.Errorf("thApplyGauge(0.95) filled = %q, want %q", filled, th.GaugeCrit)
	}
}

func TestColorizeProducesANSI(t *testing.T) {
	result := thColorize("hello", "#ff0000")
	expected := "\x1b[38;2;255;0;0mhello\x1b[0m"
	if result != expected {
		t.Errorf("thColorize(\"hello\", \"#ff0000\") = %q, want %q", result, expected)
	}
}

func TestColorizeEmptyColorReturnsUnchanged(t *testing.T) {
	result := thColorize("hello", "")
	if result != "hello" {
		t.Errorf("thColorize(\"hello\", \"\") = %q, want %q", result, "hello")
	}
}
