package preset

import (
	"sort"
	"testing"

	"gitlab.com/tinyland/lab/prompt-pulse/pkg/config"
)

// --- Get / Names tests ---

func TestGetDashboard(t *testing.T) {
	p := Get("dashboard")
	if p.Name != "dashboard" {
		t.Errorf("Get('dashboard').Name = %q, want %q", p.Name, "dashboard")
	}
}

func TestGetMinimal(t *testing.T) {
	p := Get("minimal")
	if p.Name != "minimal" {
		t.Errorf("Get('minimal').Name = %q, want %q", p.Name, "minimal")
	}
}

func TestGetOps(t *testing.T) {
	p := Get("ops")
	if p.Name != "ops" {
		t.Errorf("Get('ops').Name = %q, want %q", p.Name, "ops")
	}
}

func TestGetBilling(t *testing.T) {
	p := Get("billing")
	if p.Name != "billing" {
		t.Errorf("Get('billing').Name = %q, want %q", p.Name, "billing")
	}
}

func TestGetUnknownFallsToDashboard(t *testing.T) {
	p := Get("unknown-preset-name")
	if p.Name != "dashboard" {
		t.Errorf("Get('unknown').Name = %q, want %q (dashboard fallback)", p.Name, "dashboard")
	}
}

func TestNamesReturnsAll(t *testing.T) {
	names := Names()
	want := []string{"billing", "dashboard", "minimal", "ops"}
	if len(names) != len(want) {
		t.Fatalf("Names() returned %d items, want %d: %v", len(names), len(want), names)
	}
	sort.Strings(names)
	for i, n := range names {
		if n != want[i] {
			t.Errorf("Names()[%d] = %q, want %q", i, n, want[i])
		}
	}
}

// --- Preset content tests ---

func TestDashboardWidgetCount(t *testing.T) {
	p := Get("dashboard")
	if p.Columns != 2 {
		t.Errorf("dashboard.Columns = %d, want 2", p.Columns)
	}
	if len(p.Widgets) != 6 {
		t.Errorf("dashboard widget count = %d, want 6", len(p.Widgets))
	}
}

func TestMinimalHasWaifuAndClaude(t *testing.T) {
	p := Get("minimal")
	if len(p.Widgets) != 2 {
		t.Fatalf("minimal widget count = %d, want 2", len(p.Widgets))
	}
	ids := prWidgetIDs(p.Widgets)
	if !prContains(ids, "waifu") {
		t.Error("minimal preset should contain 'waifu'")
	}
	if !prContains(ids, "claude") {
		t.Error("minimal preset should contain 'claude'")
	}
}

func TestOpsHasNoWaifu(t *testing.T) {
	p := Get("ops")
	ids := prWidgetIDs(p.Widgets)
	if prContains(ids, "waifu") {
		t.Error("ops preset should not contain 'waifu'")
	}
}

func TestBillingHasClaudeAndBilling(t *testing.T) {
	p := Get("billing")
	ids := prWidgetIDs(p.Widgets)
	if !prContains(ids, "claude") {
		t.Error("billing preset should contain 'claude'")
	}
	if !prContains(ids, "billing") {
		t.Error("billing preset should contain 'billing'")
	}
}

// --- Resolve tests ---

func TestResolveProducesCorrectCellCount(t *testing.T) {
	p := Get("minimal")
	cells := Resolve(p, 120, 40)
	if len(cells) != len(p.Widgets) {
		t.Errorf("Resolve cell count = %d, want %d", len(cells), len(p.Widgets))
	}
}

func TestResolveCellsFitWithinBounds(t *testing.T) {
	p := Get("dashboard")
	width, height := 160, 50
	cells := Resolve(p, width, height)
	for _, c := range cells {
		if c.X < 0 || c.Y < 0 {
			t.Errorf("cell %q has negative position: (%d, %d)", c.WidgetID, c.X, c.Y)
		}
		if c.X+c.W > width {
			t.Errorf("cell %q exceeds width: X=%d + W=%d = %d > %d",
				c.WidgetID, c.X, c.W, c.X+c.W, width)
		}
		// Height must fit in available area (total minus status bar).
		availH := height - statusBarHeight
		if c.Y+c.H > availH {
			t.Errorf("cell %q exceeds available height: Y=%d + H=%d = %d > %d",
				c.WidgetID, c.Y, c.H, c.Y+c.H, availH)
		}
	}
}

func TestResolveRespectsColSpan(t *testing.T) {
	p := LayoutPreset{
		Name:    "test-colspan",
		Columns: 3,
		Widgets: []WidgetSlot{
			{WidgetID: "wide", Column: 0, Row: 0, ColSpan: 2, RowSpan: 1, Priority: 100},
			{WidgetID: "narrow", Column: 2, Row: 0, ColSpan: 1, RowSpan: 1, Priority: 90},
		},
	}
	cells := Resolve(p, 120, 30)
	if len(cells) < 2 {
		t.Fatalf("expected 2 cells, got %d", len(cells))
	}

	var wideCell, narrowCell ResolvedCell
	for _, c := range cells {
		switch c.WidgetID {
		case "wide":
			wideCell = c
		case "narrow":
			narrowCell = c
		}
	}

	// Wide cell should span 2 columns = 2/3 of 120 = 80.
	if wideCell.W != 80 {
		t.Errorf("wide cell W = %d, want 80 (2 of 3 columns)", wideCell.W)
	}
	if narrowCell.W != 40 {
		t.Errorf("narrow cell W = %d, want 40 (1 of 3 columns)", narrowCell.W)
	}
}

func TestResolveReservesStatusBar(t *testing.T) {
	p := Get("minimal")
	height := 40
	cells := Resolve(p, 120, height)

	for _, c := range cells {
		maxY := height - statusBarHeight
		if c.Y+c.H > maxY {
			t.Errorf("cell %q extends into status bar: Y=%d + H=%d = %d > %d",
				c.WidgetID, c.Y, c.H, c.Y+c.H, maxY)
		}
	}
}

func TestResolveDropsLowPriorityWhenTight(t *testing.T) {
	// Create a preset with many widgets but give only space for a few.
	p := LayoutPreset{
		Name:    "crowded",
		Columns: 1,
		Widgets: []WidgetSlot{
			{WidgetID: "a", Column: 0, Row: 0, RowSpan: 1, Priority: 100},
			{WidgetID: "b", Column: 0, Row: 1, RowSpan: 1, Priority: 90},
			{WidgetID: "c", Column: 0, Row: 2, RowSpan: 1, Priority: 80},
			{WidgetID: "d", Column: 0, Row: 3, RowSpan: 1, Priority: 10},
			{WidgetID: "e", Column: 0, Row: 4, RowSpan: 1, Priority: 5},
			{WidgetID: "f", Column: 0, Row: 5, RowSpan: 1, Priority: 1},
		},
	}

	// Very short terminal: height = statusBar(1) + minWidgetHeight(3)*2 = 7
	// Only enough room for about 2 widgets.
	cells := Resolve(p, 80, 7)
	if len(cells) > 2 {
		t.Errorf("expected at most 2 cells in tight space, got %d", len(cells))
	}
	// The kept cells should be the highest priority ones.
	for _, c := range cells {
		if c.WidgetID == "f" || c.WidgetID == "e" || c.WidgetID == "d" {
			t.Errorf("low-priority widget %q should have been dropped", c.WidgetID)
		}
	}
}

// --- prDistributeWidth tests ---

func TestDistributeWidthEvenly(t *testing.T) {
	widths := prDistributeWidth(4, 120)
	for i, w := range widths {
		if w != 30 {
			t.Errorf("column %d width = %d, want 30", i, w)
		}
	}
}

func TestDistributeWidthOddTotal(t *testing.T) {
	widths := prDistributeWidth(3, 100)
	total := 0
	for _, w := range widths {
		total += w
	}
	if total != 100 {
		t.Errorf("total width = %d, want 100", total)
	}
	// First column should get the extra pixel: 34, then 33, 33.
	if widths[0] != 34 {
		t.Errorf("widths[0] = %d, want 34", widths[0])
	}
	if widths[1] != 33 {
		t.Errorf("widths[1] = %d, want 33", widths[1])
	}
	if widths[2] != 33 {
		t.Errorf("widths[2] = %d, want 33", widths[2])
	}
}

// --- TOML tests ---

func TestLoadFromTOMLValid(t *testing.T) {
	data := []byte(`
name = "my-custom"
description = "My custom layout"
columns = 3

[[widgets]]
id = "waifu"
column = 0
row = 0
row_span = 2
priority = 100

[[widgets]]
id = "claude"
column = 1
row = 0
priority = 90

[[widgets]]
id = "billing"
column = 1
row = 1
priority = 80
`)

	p, err := LoadFromTOML(data)
	if err != nil {
		t.Fatalf("LoadFromTOML() error: %v", err)
	}
	if p.Name != "my-custom" {
		t.Errorf("Name = %q, want %q", p.Name, "my-custom")
	}
	if p.Description != "My custom layout" {
		t.Errorf("Description = %q, want %q", p.Description, "My custom layout")
	}
	if p.Columns != 3 {
		t.Errorf("Columns = %d, want 3", p.Columns)
	}
	if len(p.Widgets) != 3 {
		t.Fatalf("Widgets count = %d, want 3", len(p.Widgets))
	}
	if p.Widgets[0].WidgetID != "waifu" {
		t.Errorf("Widgets[0].WidgetID = %q, want %q", p.Widgets[0].WidgetID, "waifu")
	}
	if p.Widgets[0].RowSpan != 2 {
		t.Errorf("Widgets[0].RowSpan = %d, want 2", p.Widgets[0].RowSpan)
	}
	if p.Widgets[0].Priority != 100 {
		t.Errorf("Widgets[0].Priority = %d, want 100", p.Widgets[0].Priority)
	}
}

func TestLoadFromTOMLMissingNameReturnsError(t *testing.T) {
	data := []byte(`
columns = 2

[[widgets]]
id = "claude"
column = 0
row = 0
`)

	_, err := LoadFromTOML(data)
	if err == nil {
		t.Error("LoadFromTOML() should return error for missing name")
	}
}

func TestSaveToTOMLRoundtrip(t *testing.T) {
	original := LayoutPreset{
		Name:        "roundtrip-test",
		Description: "Testing serialization roundtrip",
		Columns:     2,
		Widgets: []WidgetSlot{
			{WidgetID: "waifu", Column: 0, Row: 0, ColSpan: 1, RowSpan: 2, Priority: 100},
			{WidgetID: "claude", Column: 1, Row: 0, ColSpan: 1, RowSpan: 1, Priority: 90},
		},
	}

	data, err := SaveToTOML(original)
	if err != nil {
		t.Fatalf("SaveToTOML() error: %v", err)
	}

	restored, err := LoadFromTOML(data)
	if err != nil {
		t.Fatalf("LoadFromTOML() roundtrip error: %v", err)
	}

	if restored.Name != original.Name {
		t.Errorf("roundtrip Name = %q, want %q", restored.Name, original.Name)
	}
	if restored.Description != original.Description {
		t.Errorf("roundtrip Description = %q, want %q", restored.Description, original.Description)
	}
	if restored.Columns != original.Columns {
		t.Errorf("roundtrip Columns = %d, want %d", restored.Columns, original.Columns)
	}
	if len(restored.Widgets) != len(original.Widgets) {
		t.Fatalf("roundtrip widget count = %d, want %d", len(restored.Widgets), len(original.Widgets))
	}
	for i, w := range restored.Widgets {
		o := original.Widgets[i]
		if w.WidgetID != o.WidgetID {
			t.Errorf("roundtrip Widgets[%d].WidgetID = %q, want %q", i, w.WidgetID, o.WidgetID)
		}
		if w.Priority != o.Priority {
			t.Errorf("roundtrip Widgets[%d].Priority = %d, want %d", i, w.Priority, o.Priority)
		}
		if w.ColSpan != o.ColSpan {
			t.Errorf("roundtrip Widgets[%d].ColSpan = %d, want %d", i, w.ColSpan, o.ColSpan)
		}
		if w.RowSpan != o.RowSpan {
			t.Errorf("roundtrip Widgets[%d].RowSpan = %d, want %d", i, w.RowSpan, o.RowSpan)
		}
	}
}

// --- SelectForSize tests ---

func TestSelectForSizeSmallTerminal(t *testing.T) {
	name := SelectForSize(80, 24)
	if name != "minimal" {
		t.Errorf("SelectForSize(80, 24) = %q, want %q", name, "minimal")
	}
}

func TestSelectForSizeMediumTerminal(t *testing.T) {
	name := SelectForSize(120, 40)
	if name != "dashboard" {
		t.Errorf("SelectForSize(120, 40) = %q, want %q", name, "dashboard")
	}
}

func TestSelectForSizeLargeTerminal(t *testing.T) {
	name := SelectForSize(200, 60)
	if name != "dashboard" {
		t.Errorf("SelectForSize(200, 60) = %q, want %q", name, "dashboard")
	}
}

func TestSelectByConfigExplicit(t *testing.T) {
	cfg := config.Config{}
	cfg.Layout.Preset = "ops"
	name := SelectByConfig(cfg)
	if name != "ops" {
		t.Errorf("SelectByConfig(ops) = %q, want %q", name, "ops")
	}
}

func TestSelectByConfigAuto(t *testing.T) {
	cfg := config.Config{}
	cfg.Layout.Preset = "auto"
	name := SelectByConfig(cfg)
	if name != "dashboard" {
		t.Errorf("SelectByConfig(auto) = %q, want %q", name, "dashboard")
	}
}

// --- prFilterByPriority tests ---

func TestFilterByPriorityKeepsHighest(t *testing.T) {
	slots := []WidgetSlot{
		{WidgetID: "low", Priority: 10},
		{WidgetID: "high", Priority: 100},
		{WidgetID: "mid", Priority: 50},
	}
	filtered := prFilterByPriority(slots, 2)
	if len(filtered) != 2 {
		t.Fatalf("filtered count = %d, want 2", len(filtered))
	}
	ids := prWidgetIDs(filtered)
	if !prContains(ids, "high") {
		t.Error("filtered should contain 'high' (priority 100)")
	}
	if !prContains(ids, "mid") {
		t.Error("filtered should contain 'mid' (priority 50)")
	}
	if prContains(ids, "low") {
		t.Error("filtered should not contain 'low' (priority 10)")
	}
}

func TestFilterByPriorityPreservesOrder(t *testing.T) {
	slots := []WidgetSlot{
		{WidgetID: "c", Priority: 30},
		{WidgetID: "a", Priority: 100},
		{WidgetID: "b", Priority: 50},
	}
	filtered := prFilterByPriority(slots, 2)
	// Should keep 'a' (100) and 'b' (50), in original order: a, b.
	if len(filtered) != 2 {
		t.Fatalf("filtered count = %d, want 2", len(filtered))
	}
	if filtered[0].WidgetID != "a" {
		t.Errorf("filtered[0] = %q, want 'a'", filtered[0].WidgetID)
	}
	if filtered[1].WidgetID != "b" {
		t.Errorf("filtered[1] = %q, want 'b'", filtered[1].WidgetID)
	}
}

// --- Edge case tests ---

func TestResolveZeroDimensions(t *testing.T) {
	p := Get("dashboard")
	cells := Resolve(p, 0, 0)
	if cells != nil {
		t.Errorf("Resolve(0,0) should return nil, got %d cells", len(cells))
	}
}

func TestResolveEmptyPreset(t *testing.T) {
	p := LayoutPreset{Name: "empty", Columns: 2}
	cells := Resolve(p, 120, 40)
	if cells != nil {
		t.Errorf("Resolve(empty) should return nil, got %d cells", len(cells))
	}
}

func TestDistributeWidthZeroColumns(t *testing.T) {
	widths := prDistributeWidth(0, 100)
	if widths != nil {
		t.Errorf("prDistributeWidth(0, 100) should return nil, got %v", widths)
	}
}

func TestDistributeHeightEmpty(t *testing.T) {
	allocs := prDistributeHeight(nil, 100)
	if allocs != nil {
		t.Errorf("prDistributeHeight(nil) should return nil, got %v", allocs)
	}
}

func TestLoadFromTOMLInvalidSyntax(t *testing.T) {
	data := []byte(`this is not valid TOML [[[`)
	_, err := LoadFromTOML(data)
	if err == nil {
		t.Error("LoadFromTOML() should return error for invalid TOML syntax")
	}
}

// --- Helpers ---

func prWidgetIDs(slots []WidgetSlot) []string {
	ids := make([]string, len(slots))
	for i, s := range slots {
		ids[i] = s.WidgetID
	}
	return ids
}

func prContains(strs []string, target string) bool {
	for _, s := range strs {
		if s == target {
			return true
		}
	}
	return false
}
