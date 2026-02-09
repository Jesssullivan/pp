package banner

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gitlab.com/tinyland/lab/prompt-pulse/pkg/components"
)

// --- SelectPreset tests ---

func TestSelectPreset_SmallTerminal(t *testing.T) {
	p := SelectPreset(60, 20)
	if p.Name != "compact" {
		t.Errorf("expected compact for 60x20, got %s", p.Name)
	}
}

func TestSelectPreset_MediumTerminal(t *testing.T) {
	p := SelectPreset(120, 35)
	if p.Name != "standard" {
		t.Errorf("expected standard for 120x35, got %s", p.Name)
	}
}

func TestSelectPreset_LargeTerminal(t *testing.T) {
	p := SelectPreset(160, 45)
	if p.Name != "wide" {
		t.Errorf("expected wide for 160x45, got %s", p.Name)
	}
}

func TestSelectPreset_HugeTerminal(t *testing.T) {
	p := SelectPreset(200, 50)
	if p.Name != "ultrawide" {
		t.Errorf("expected ultrawide for 200x50, got %s", p.Name)
	}
}

func TestSelectPreset_BoundaryExactlyStandard(t *testing.T) {
	p := SelectPreset(120, 35)
	if p.Name != "standard" {
		t.Errorf("expected standard for exactly 120x35, got %s", p.Name)
	}
}

func TestSelectPreset_BoundaryJustBelowStandard(t *testing.T) {
	p := SelectPreset(119, 35)
	if p.Name != "compact" {
		t.Errorf("expected compact for 119x35, got %s", p.Name)
	}
}

func TestSelectPreset_WidthFitsButHeightTooShort(t *testing.T) {
	// Width is wide enough for Standard but height is too short.
	p := SelectPreset(120, 34)
	if p.Name != "compact" {
		t.Errorf("expected compact for 120x34 (height too short for standard), got %s", p.Name)
	}
}

func TestSelectPreset_TinyTerminal(t *testing.T) {
	p := SelectPreset(10, 5)
	if p.Name != "compact" {
		t.Errorf("expected compact for tiny 10x5, got %s", p.Name)
	}
}

// --- Render tests ---

func TestRender_EmptyBannerData(t *testing.T) {
	result := Render(BannerData{}, Compact)
	if result == "" {
		t.Error("expected non-empty result for empty BannerData (should be filled with blanks)")
	}
	lines := strings.Split(result, "\n")
	if len(lines) != Compact.Height {
		t.Errorf("expected %d lines, got %d", Compact.Height, len(lines))
	}
}

func TestRender_SingleWidget(t *testing.T) {
	data := BannerData{
		Widgets: []WidgetData{
			{ID: "cpu", Title: "CPU", Content: "usage: 42%", MinW: 20, MinH: 5},
		},
	}
	result := Render(data, Compact)
	if result == "" {
		t.Fatal("expected non-empty result")
	}
	// The result should contain the widget content somewhere.
	if !strings.Contains(result, "usage: 42%") {
		t.Error("expected rendered output to contain widget content")
	}
}

func TestRender_SingleWidgetHasBorder(t *testing.T) {
	data := BannerData{
		Widgets: []WidgetData{
			{ID: "cpu", Title: "CPU", Content: "hello", MinW: 20, MinH: 5},
		},
	}
	result := Render(data, Compact)
	// Rounded border top-left corner character.
	if !strings.Contains(result, "\u256d") {
		t.Error("expected rounded border corner in output")
	}
}

func TestRender_MultipleWidgetsArrangedInColumns(t *testing.T) {
	data := BannerData{
		Widgets: []WidgetData{
			{ID: "waifu-main", Title: "Waifu", Content: "image", MinW: 40, MinH: 10},
			{ID: "cpu", Title: "CPU", Content: "50%", MinW: 30, MinH: 5},
			{ID: "mem", Title: "Memory", Content: "8GB", MinW: 30, MinH: 5},
		},
	}
	result := Render(data, Standard)
	if result == "" {
		t.Fatal("expected non-empty result")
	}
	// Both data widgets should appear.
	if !strings.Contains(result, "50%") {
		t.Error("expected CPU content in output")
	}
	if !strings.Contains(result, "8GB") {
		t.Error("expected Memory content in output")
	}
}

func TestRender_ExactLineCount(t *testing.T) {
	presets := []Preset{Compact, Standard, Wide, UltraWide}
	data := BannerData{
		Widgets: []WidgetData{
			{ID: "test", Title: "Test", Content: "data", MinW: 10, MinH: 3},
		},
	}
	for _, p := range presets {
		result := Render(data, p)
		lines := strings.Split(result, "\n")
		if len(lines) != p.Height {
			t.Errorf("preset %s: expected %d lines, got %d", p.Name, p.Height, len(lines))
		}
	}
}

func TestRender_LineWidthNotExceedPreset(t *testing.T) {
	data := BannerData{
		Widgets: []WidgetData{
			{ID: "test", Title: "Test", Content: strings.Repeat("x", 200), MinW: 10, MinH: 3},
		},
	}
	result := Render(data, Compact)
	lines := strings.Split(result, "\n")
	for i, line := range lines {
		vis := components.VisibleLen(line)
		if vis > Compact.Width {
			t.Errorf("line %d: visible width %d exceeds preset width %d", i, vis, Compact.Width)
		}
	}
}

func TestRender_ZeroSizeWidgetData(t *testing.T) {
	data := BannerData{
		Widgets: []WidgetData{
			{ID: "empty", Title: "", Content: "", MinW: 0, MinH: 0},
		},
	}
	result := Render(data, Compact)
	lines := strings.Split(result, "\n")
	if len(lines) != Compact.Height {
		t.Errorf("expected %d lines, got %d", Compact.Height, len(lines))
	}
}

func TestRender_WidgetExceedsAvailableSpace(t *testing.T) {
	data := BannerData{
		Widgets: []WidgetData{
			{ID: "huge", Title: "Huge", Content: strings.Repeat("line\n", 100), MinW: 200, MinH: 100},
		},
	}
	result := Render(data, Compact)
	lines := strings.Split(result, "\n")
	if len(lines) != Compact.Height {
		t.Errorf("expected %d lines even with huge widget, got %d", Compact.Height, len(lines))
	}
}

func TestRender_ANSIContentHandledCorrectly(t *testing.T) {
	ansiContent := "\x1b[31mred text\x1b[0m"
	data := BannerData{
		Widgets: []WidgetData{
			{ID: "ansi", Title: "Colors", Content: ansiContent, MinW: 20, MinH: 4},
		},
	}
	result := Render(data, Compact)
	lines := strings.Split(result, "\n")
	for i, line := range lines {
		vis := components.VisibleLen(line)
		if vis > Compact.Width {
			t.Errorf("line %d with ANSI: visible width %d exceeds %d", i, vis, Compact.Width)
		}
	}
}

// --- bnArrangeWidgets tests ---

func TestBnArrangeWidgets_CompactSingleColumn(t *testing.T) {
	widgets := []WidgetData{
		{ID: "a", Title: "A", Content: "alpha", MinW: 10, MinH: 5},
		{ID: "b", Title: "B", Content: "beta", MinW: 10, MinH: 5},
	}
	placements := bnArrangeWidgets(widgets, 80, 24)
	if len(placements) == 0 {
		t.Fatal("expected at least one placement")
	}
	// In compact mode, all widgets should be in column 0 (X=0).
	for _, p := range placements {
		if p.X != 0 {
			t.Errorf("compact mode: expected X=0, got X=%d for widget %s", p.X, p.Widget.ID)
		}
	}
	// Verify stacking: second widget starts after first.
	if len(placements) >= 2 {
		if placements[1].Y <= placements[0].Y {
			t.Errorf("expected second widget Y (%d) > first widget Y (%d)", placements[1].Y, placements[0].Y)
		}
	}
}

func TestBnArrangeWidgets_StandardWaifuInLeftColumn(t *testing.T) {
	widgets := []WidgetData{
		{ID: "waifu-main", Title: "Waifu", Content: "img", MinW: 40, MinH: 10},
		{ID: "cpu", Title: "CPU", Content: "50%", MinW: 30, MinH: 5},
	}
	placements := bnArrangeWidgets(widgets, 120, 35)
	if len(placements) < 2 {
		t.Fatalf("expected 2 placements, got %d", len(placements))
	}
	// Find waifu and cpu placements.
	var waifuP, cpuP *bnPlacement
	for i := range placements {
		if placements[i].Widget.ID == "waifu-main" {
			waifuP = &placements[i]
		}
		if placements[i].Widget.ID == "cpu" {
			cpuP = &placements[i]
		}
	}
	if waifuP == nil || cpuP == nil {
		t.Fatal("expected both waifu and cpu placements")
	}
	// Waifu should be in column 0 (leftmost).
	if waifuP.X != 0 {
		t.Errorf("expected waifu X=0, got %d", waifuP.X)
	}
	// CPU should be in a different column (X > 0).
	if cpuP.X <= waifuP.X {
		t.Errorf("expected cpu X > waifu X, got cpu.X=%d waifu.X=%d", cpuP.X, waifuP.X)
	}
}

func TestBnArrangeWidgets_EmptyWidgets(t *testing.T) {
	placements := bnArrangeWidgets(nil, 80, 24)
	if len(placements) != 0 {
		t.Errorf("expected 0 placements for nil widgets, got %d", len(placements))
	}
}

// --- bnCompose tests ---

func TestBnCompose_OverlappingPlacements(t *testing.T) {
	// Two placements at the same position; the second should win.
	placements := []bnPlacement{
		{
			Widget: WidgetData{ID: "a", Title: "A", Content: "AAAA", MinW: 10, MinH: 4},
			X: 0, Y: 0, W: 10, H: 4,
		},
		{
			Widget: WidgetData{ID: "b", Title: "B", Content: "BBBB", MinW: 10, MinH: 4},
			X: 0, Y: 0, W: 10, H: 4,
		},
	}
	result := bnCompose(placements, 20, 10)
	lines := strings.Split(result, "\n")
	// The second widget ("B" content) should overwrite the first.
	// Check that "BBBB" appears in the output.
	found := false
	for _, line := range lines {
		if strings.Contains(line, "BBBB") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected overlapping widget B content to appear (z-order: last wins)")
	}
}

func TestBnCompose_TruncatesLongContent(t *testing.T) {
	longContent := strings.Repeat("X", 200)
	placements := []bnPlacement{
		{
			Widget: WidgetData{ID: "long", Title: "Long", Content: longContent, MinW: 10, MinH: 4},
			X: 0, Y: 0, W: 30, H: 4,
		},
	}
	result := bnCompose(placements, 40, 6)
	lines := strings.Split(result, "\n")
	for i, line := range lines {
		vis := components.VisibleLen(line)
		if vis > 40 {
			t.Errorf("line %d: visible width %d > grid width 40", i, vis)
		}
	}
}

func TestBnCompose_PadsShortContent(t *testing.T) {
	placements := []bnPlacement{
		{
			Widget: WidgetData{ID: "short", Title: "S", Content: "hi", MinW: 5, MinH: 3},
			X: 0, Y: 0, W: 20, H: 3,
		},
	}
	result := bnCompose(placements, 40, 5)
	lines := strings.Split(result, "\n")
	if len(lines) != 5 {
		t.Errorf("expected 5 lines, got %d", len(lines))
	}
	for i, line := range lines {
		vis := components.VisibleLen(line)
		if vis != 40 {
			t.Errorf("line %d: expected visible width 40, got %d", i, vis)
		}
	}
}

func TestBnCompose_EmptyPlacements(t *testing.T) {
	result := bnCompose(nil, 20, 5)
	if result == "" {
		t.Error("expected non-empty grid even with no placements")
	}
	lines := strings.Split(result, "\n")
	if len(lines) != 5 {
		t.Errorf("expected 5 lines, got %d", len(lines))
	}
}

// --- RenderCached tests ---

func TestRenderCached_WritesCacheFile(t *testing.T) {
	dir := t.TempDir()
	data := BannerData{
		Widgets: []WidgetData{
			{ID: "test", Title: "Test", Content: "hello", MinW: 10, MinH: 3},
		},
	}
	result, err := RenderCached(dir, data, Compact)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == "" {
		t.Error("expected non-empty result")
	}

	// Verify a cache file was written.
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("failed to read cache dir: %v", err)
	}
	found := false
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "banner-") && strings.HasSuffix(e.Name(), ".cache") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected banner cache file to be written")
	}
}

func TestRenderCached_ReadsFromCacheOnSecondCall(t *testing.T) {
	dir := t.TempDir()
	data := BannerData{
		Widgets: []WidgetData{
			{ID: "test", Title: "Test", Content: "cached", MinW: 10, MinH: 3},
		},
	}
	result1, err := RenderCached(dir, data, Compact)
	if err != nil {
		t.Fatalf("first call error: %v", err)
	}

	// Overwrite the cache file with custom content to verify we read from it.
	key := bnCacheKey(data, Compact)
	path := filepath.Join(dir, "banner-"+key+".cache")
	if err := os.WriteFile(path, []byte("CACHED_SENTINEL"), 0644); err != nil {
		t.Fatalf("failed to write sentinel: %v", err)
	}

	result2, err := RenderCached(dir, data, Compact)
	if err != nil {
		t.Fatalf("second call error: %v", err)
	}

	if result2 != "CACHED_SENTINEL" {
		t.Errorf("expected cached sentinel, got %q (first result was %d chars)", result2, len(result1))
	}
}

func TestRenderCached_IgnoresStaleCache(t *testing.T) {
	dir := t.TempDir()
	data := BannerData{
		Widgets: []WidgetData{
			{ID: "test", Title: "Test", Content: "fresh", MinW: 10, MinH: 3},
		},
	}

	key := bnCacheKey(data, Compact)
	path := filepath.Join(dir, "banner-"+key+".cache")

	// Write a stale cache file with old modification time.
	if err := os.WriteFile(path, []byte("STALE"), 0644); err != nil {
		t.Fatalf("failed to write stale cache: %v", err)
	}
	staleTime := time.Now().Add(-60 * time.Second)
	if err := os.Chtimes(path, staleTime, staleTime); err != nil {
		t.Fatalf("failed to set stale mtime: %v", err)
	}

	result, err := RenderCached(dir, data, Compact)
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	// Should NOT return "STALE" — should re-render.
	if result == "STALE" {
		t.Error("expected fresh render, got stale cached content")
	}
	if !strings.Contains(result, "fresh") {
		t.Error("expected fresh-rendered content containing 'fresh'")
	}
}

func TestCacheKey_ChangesWithWidgetData(t *testing.T) {
	data1 := BannerData{
		Widgets: []WidgetData{
			{ID: "a", Title: "A", Content: "hello", MinW: 10, MinH: 3},
		},
	}
	data2 := BannerData{
		Widgets: []WidgetData{
			{ID: "a", Title: "A", Content: "world", MinW: 10, MinH: 3},
		},
	}
	key1 := bnCacheKey(data1, Compact)
	key2 := bnCacheKey(data2, Compact)
	if key1 == key2 {
		t.Error("expected different cache keys for different widget content")
	}
}

func TestCacheKey_ChangesWithPreset(t *testing.T) {
	data := BannerData{
		Widgets: []WidgetData{
			{ID: "a", Title: "A", Content: "hello", MinW: 10, MinH: 3},
		},
	}
	key1 := bnCacheKey(data, Compact)
	key2 := bnCacheKey(data, Standard)
	if key1 == key2 {
		t.Error("expected different cache keys for different presets")
	}
}

func TestCacheKey_DeterministicForSameInput(t *testing.T) {
	data := BannerData{
		Widgets: []WidgetData{
			{ID: "a", Title: "A", Content: "hello", MinW: 10, MinH: 3},
		},
	}
	key1 := bnCacheKey(data, Compact)
	key2 := bnCacheKey(data, Compact)
	if key1 != key2 {
		t.Errorf("expected same cache key for same input, got %s and %s", key1, key2)
	}
}

// --- Border rendering tests ---

func TestRender_BorderRenderedAroundWidgets(t *testing.T) {
	data := BannerData{
		Widgets: []WidgetData{
			{ID: "test", Title: "Info", Content: "data", MinW: 20, MinH: 5},
		},
	}
	result := Render(data, Compact)
	// Check for rounded border characters.
	hasTopLeft := strings.Contains(result, "\u256d")     // top-left rounded
	hasTopRight := strings.Contains(result, "\u256e")    // top-right rounded
	hasBottomLeft := strings.Contains(result, "\u2570")  // bottom-left rounded
	hasBottomRight := strings.Contains(result, "\u256f") // bottom-right rounded
	if !hasTopLeft || !hasTopRight || !hasBottomLeft || !hasBottomRight {
		t.Error("expected all four rounded border corners in output")
	}
}

func TestRender_TitleAppearsInBorder(t *testing.T) {
	data := BannerData{
		Widgets: []WidgetData{
			{ID: "test", Title: "MyWidget", Content: "data", MinW: 20, MinH: 5},
		},
	}
	result := Render(data, Compact)
	if !strings.Contains(result, "MyWidget") {
		t.Error("expected widget title to appear in border")
	}
}

// --- Additional edge case tests ---

func TestBnFitToWidth_Zero(t *testing.T) {
	result := bnFitToWidth("hello", 0)
	if result != "" {
		t.Errorf("expected empty string for width 0, got %q", result)
	}
}

func TestBnFitToWidth_ExactMatch(t *testing.T) {
	result := bnFitToWidth("hello", 5)
	if components.VisibleLen(result) != 5 {
		t.Errorf("expected visible len 5, got %d", components.VisibleLen(result))
	}
}

func TestBnCompose_ZeroSize(t *testing.T) {
	result := bnCompose(nil, 0, 0)
	if result != "" {
		t.Errorf("expected empty string for zero-size grid, got %q", result)
	}
}

func TestBnIsWaifuWidget(t *testing.T) {
	tests := []struct {
		id   string
		want bool
	}{
		{"waifu-main", true},
		{"waifu", true},
		{"waifuX", true},
		{"waif", false},
		{"cpu", false},
		{"", false},
	}
	for _, tt := range tests {
		got := bnIsWaifuWidget(WidgetData{ID: tt.id})
		if got != tt.want {
			t.Errorf("bnIsWaifuWidget(%q) = %v, want %v", tt.id, got, tt.want)
		}
	}
}
