package termtest

import (
	"fmt"
	"testing"
)

// --- Profile Tests ---

func TestProfiles_ReturnsAtLeast8(t *testing.T) {
	profiles := Profiles()
	if len(profiles) < 8 {
		t.Errorf("Profiles() returned %d profiles, want >= 8", len(profiles))
	}
}

func TestProfiles_NonEmptyNameAndEnvVars(t *testing.T) {
	for _, p := range Profiles() {
		if p.Name == "" {
			t.Error("profile has empty Name")
		}
		if len(p.EnvVars) == 0 {
			t.Errorf("profile %q has empty EnvVars", p.Name)
		}
	}
}

func TestProfile_Ghostty_KittyProtocolAndTruecolor(t *testing.T) {
	p := ProfileByName("Ghostty")
	if p == nil {
		t.Fatal("Ghostty profile not found")
	}
	if p.ImageProtocol != "kitty" {
		t.Errorf("Ghostty.ImageProtocol = %q, want \"kitty\"", p.ImageProtocol)
	}
	if p.ColorDepth != 24 {
		t.Errorf("Ghostty.ColorDepth = %d, want 24", p.ColorDepth)
	}
}

func TestProfile_Tilix_256Color(t *testing.T) {
	p := ProfileByName("Tilix")
	if p == nil {
		t.Fatal("Tilix profile not found")
	}
	if p.ColorDepth != 256 {
		t.Errorf("Tilix.ColorDepth = %d, want 256", p.ColorDepth)
	}
}

func TestProfile_Alacritty_HalfblockOnly(t *testing.T) {
	p := ProfileByName("Alacritty")
	if p == nil {
		t.Fatal("Alacritty profile not found")
	}
	if p.ImageProtocol != "halfblock" {
		t.Errorf("Alacritty.ImageProtocol = %q, want \"halfblock\"", p.ImageProtocol)
	}
}

// --- Compat Tests ---

func TestCheckCompat_ReturnsAllFeatures(t *testing.T) {
	p := ttGhosttyProfile()
	results := CheckCompat(p)
	features := Features()

	if len(results) != len(features) {
		t.Fatalf("CheckCompat returned %d results, want %d", len(results), len(features))
	}

	resultMap := make(map[string]bool, len(results))
	for _, r := range results {
		resultMap[r.Feature] = true
	}
	for _, f := range features {
		if !resultMap[f] {
			t.Errorf("CheckCompat missing feature %q", f)
		}
	}
}

func TestCheckCompat_Ghostty_AllFull(t *testing.T) {
	p := ttGhosttyProfile()
	results := CheckCompat(p)

	for _, r := range results {
		if r.Status != "full" {
			t.Errorf("Ghostty feature %q: status = %q, want \"full\"", r.Feature, r.Status)
		}
	}
}

func TestCheckCompat_Tilix_DegradedTruecolor(t *testing.T) {
	p := ttTilixProfile()
	results := CheckCompat(p)

	for _, r := range results {
		if r.Feature == "truecolor" {
			if r.Status != "degraded" {
				t.Errorf("Tilix truecolor: status = %q, want \"degraded\"", r.Status)
			}
			return
		}
	}
	t.Error("truecolor feature not found in Tilix compat results")
}

func TestCheckCompat_Alacritty_DegradedImageRendering(t *testing.T) {
	p := ttAlacrittyProfile()
	results := CheckCompat(p)

	for _, r := range results {
		if r.Feature == "image_rendering" {
			if r.Status != "degraded" {
				t.Errorf("Alacritty image_rendering: status = %q, want \"degraded\"", r.Status)
			}
			return
		}
	}
	t.Error("image_rendering feature not found in Alacritty compat results")
}

func TestCheckCompat_AppleTerminal_UnsupportedImageRendering(t *testing.T) {
	p := ttAppleTerminalProfile()
	results := CheckCompat(p)

	for _, r := range results {
		if r.Feature == "image_rendering" {
			if r.Status != "unsupported" {
				t.Errorf("Apple Terminal image_rendering: status = %q, want \"unsupported\"", r.Status)
			}
			return
		}
	}
	t.Error("image_rendering feature not found in Apple Terminal compat results")
}

func TestFeatures_ExpectedList(t *testing.T) {
	expected := []string{
		"truecolor",
		"image_rendering",
		"unicode_box_drawing",
		"braille_chars",
		"sparkline_chars",
		"mouse_navigation",
		"clipboard_integration",
		"resize_detection",
		"pixel_sizing",
	}

	features := Features()
	if len(features) != len(expected) {
		t.Fatalf("Features() returned %d items, want %d", len(features), len(expected))
	}
	for i, f := range features {
		if f != expected[i] {
			t.Errorf("Features()[%d] = %q, want %q", i, f, expected[i])
		}
	}
}

// --- Validate Tests ---

func TestValidateBoxDrawing_UnicodeTerminals(t *testing.T) {
	unicodeTerminals := []string{"Ghostty", "Kitty", "WezTerm", "Alacritty"}
	for _, name := range unicodeTerminals {
		p := ProfileByName(name)
		if p == nil {
			t.Fatalf("profile %q not found", name)
		}
		if err := ValidateBoxDrawing(*p); err != nil {
			t.Errorf("ValidateBoxDrawing(%s) = %v, want nil", name, err)
		}
	}
}

func TestValidateImageProtocol_Ghostty_ReturnsKitty(t *testing.T) {
	p := ttGhosttyProfile()
	proto, err := ValidateImageProtocol(p)
	if err != nil {
		t.Fatalf("ValidateImageProtocol(Ghostty) error: %v", err)
	}
	if proto != "kitty" {
		t.Errorf("ValidateImageProtocol(Ghostty) = %q, want \"kitty\"", proto)
	}
}

func TestValidateImageProtocol_Alacritty_ReturnsHalfblock(t *testing.T) {
	p := ttAlacrittyProfile()
	proto, err := ValidateImageProtocol(p)
	if err != nil {
		t.Fatalf("ValidateImageProtocol(Alacritty) error: %v", err)
	}
	if proto != "halfblock" {
		t.Errorf("ValidateImageProtocol(Alacritty) = %q, want \"halfblock\"", proto)
	}
}

func TestValidateColorDepth_Truecolor(t *testing.T) {
	truecolorTerminals := []string{"Ghostty", "Kitty", "WezTerm", "iTerm2", "Alacritty"}
	for _, name := range truecolorTerminals {
		p := ProfileByName(name)
		if p == nil {
			t.Fatalf("profile %q not found", name)
		}
		if depth := ValidateColorDepth(*p); depth != 24 {
			t.Errorf("ValidateColorDepth(%s) = %d, want 24", name, depth)
		}
	}
}

func TestValidateColorDepth_Tilix_256(t *testing.T) {
	p := ttTilixProfile()
	if depth := ValidateColorDepth(p); depth != 256 {
		t.Errorf("ValidateColorDepth(Tilix) = %d, want 256", depth)
	}
}

func TestBestImageProtocol_Selection(t *testing.T) {
	cases := []struct {
		name string
		want string
	}{
		{"Ghostty", "kitty"},
		{"Kitty", "kitty"},
		{"WezTerm", "kitty"},
		{"iTerm2", "iterm2"},
		{"Tilix", "sixel"},
		{"Alacritty", "halfblock"},
		{"Apple Terminal", "none"},
		{"tmux", "halfblock"},
	}

	for _, tc := range cases {
		p := ProfileByName(tc.name)
		if p == nil {
			t.Fatalf("profile %q not found", tc.name)
		}
		got := ttBestImageProtocol(*p)
		if got != tc.want {
			t.Errorf("ttBestImageProtocol(%s) = %q, want %q", tc.name, got, tc.want)
		}
	}
}

func TestFallbackStrategy_DegradedFeatures(t *testing.T) {
	// Tilix has degraded truecolor, so fallback strategy should be non-empty
	p := ttTilixProfile()
	strategy := ttFallbackStrategy("truecolor", p)
	if strategy == "" {
		t.Error("ttFallbackStrategy(truecolor, Tilix) returned empty string")
	}

	// Apple Terminal has unsupported image rendering
	p2 := ttAppleTerminalProfile()
	strategy2 := ttFallbackStrategy("image_rendering", p2)
	if strategy2 == "" {
		t.Error("ttFallbackStrategy(image_rendering, Apple Terminal) returned empty string")
	}

	// Alacritty has degraded image rendering
	p3 := ttAlacrittyProfile()
	strategy3 := ttFallbackStrategy("image_rendering", p3)
	if strategy3 == "" {
		t.Error("ttFallbackStrategy(image_rendering, Alacritty) returned empty string")
	}
}

// --- Snapshot Tests ---

func TestCaptureSnapshot_CapturesContent(t *testing.T) {
	renderFn := func(w, h int) string {
		return fmt.Sprintf("width=%d height=%d", w, h)
	}

	snap := CaptureSnapshot("test-snap", "Ghostty", renderFn, 80, 24)

	if snap.Name != "test-snap" {
		t.Errorf("snap.Name = %q, want \"test-snap\"", snap.Name)
	}
	if snap.Terminal != "Ghostty" {
		t.Errorf("snap.Terminal = %q, want \"Ghostty\"", snap.Terminal)
	}
	if snap.Width != 80 {
		t.Errorf("snap.Width = %d, want 80", snap.Width)
	}
	if snap.Height != 24 {
		t.Errorf("snap.Height = %d, want 24", snap.Height)
	}
	if snap.Content != "width=80 height=24" {
		t.Errorf("snap.Content = %q, want \"width=80 height=24\"", snap.Content)
	}
}

func TestCompareSnapshots_IdenticalNoDiffs(t *testing.T) {
	s := Snapshot{
		Name:     "test",
		Terminal: "Ghostty",
		Width:    80,
		Height:   24,
		Content:  "line1\nline2\nline3",
	}

	diffs := CompareSnapshots(s, s)
	if len(diffs) != 0 {
		t.Errorf("CompareSnapshots(identical) returned %d diffs, want 0", len(diffs))
	}
}

func TestCompareSnapshots_DifferentContent(t *testing.T) {
	expected := Snapshot{
		Name:    "expected",
		Content: "line1\nline2\nline3",
	}
	actual := Snapshot{
		Name:    "actual",
		Content: "line1\nchanged\nline3",
	}

	diffs := CompareSnapshots(expected, actual)
	if len(diffs) != 1 {
		t.Fatalf("CompareSnapshots returned %d diffs, want 1", len(diffs))
	}
	if diffs[0].Line != 2 {
		t.Errorf("diff.Line = %d, want 2", diffs[0].Line)
	}
	if diffs[0].Expected != "line2" {
		t.Errorf("diff.Expected = %q, want \"line2\"", diffs[0].Expected)
	}
	if diffs[0].Actual != "changed" {
		t.Errorf("diff.Actual = %q, want \"changed\"", diffs[0].Actual)
	}
}

func TestCompareSnapshots_DifferentLineCounts(t *testing.T) {
	expected := Snapshot{
		Name:    "expected",
		Content: "line1\nline2\nline3",
	}
	actual := Snapshot{
		Name:    "actual",
		Content: "line1\nline2",
	}

	diffs := CompareSnapshots(expected, actual)
	if len(diffs) != 1 {
		t.Fatalf("CompareSnapshots(different line counts) returned %d diffs, want 1", len(diffs))
	}
	if diffs[0].Line != 3 {
		t.Errorf("diff.Line = %d, want 3", diffs[0].Line)
	}
	if diffs[0].Expected != "line3" {
		t.Errorf("diff.Expected = %q, want \"line3\"", diffs[0].Expected)
	}
	if diffs[0].Actual != "" {
		t.Errorf("diff.Actual = %q, want \"\"", diffs[0].Actual)
	}
}

// --- tmux Workarounds ---

func TestTmux_ProfileSetsWorkarounds(t *testing.T) {
	p := ttTmuxProfile()
	results := CheckCompat(p)

	// tmux should have workarounds for several degraded/unsupported features
	workaroundCount := 0
	for _, r := range results {
		if r.Workaround != "" {
			workaroundCount++
		}
	}
	if workaroundCount == 0 {
		t.Error("tmux profile has no workarounds set; expected at least one")
	}

	// Specifically, tmux should have degraded resize detection
	for _, r := range results {
		if r.Feature == "resize_detection" {
			if r.Status != "degraded" {
				t.Errorf("tmux resize_detection: status = %q, want \"degraded\"", r.Status)
			}
			if r.Workaround == "" {
				t.Error("tmux resize_detection workaround is empty")
			}
			return
		}
	}
	t.Error("resize_detection feature not found in tmux compat results")
}
