// Package preset defines config-driven layout presets that map widget
// arrangements to constraint-based layouts. Users select presets via
// config or CLI, and may define custom layouts via TOML.
package preset

import "sort"

// LayoutPreset defines a named widget arrangement.
type LayoutPreset struct {
	Name        string       `toml:"name"`
	Description string       `toml:"description"`
	Widgets     []WidgetSlot `toml:"widgets"`
	Columns     int          `toml:"columns"`
}

// WidgetSlot defines where a widget appears in the layout.
type WidgetSlot struct {
	WidgetID string `toml:"id"`       // matches widget.ID() e.g. "waifu", "claude", "billing"
	Column   int    `toml:"column"`   // 0-indexed column
	Row      int    `toml:"row"`      // 0-indexed row within column
	ColSpan  int    `toml:"col_span"` // columns to span (default 1)
	RowSpan  int    `toml:"row_span"` // rows to span (default 1)
	MinW     int    `toml:"min_w"`    // minimum width override (0 = use widget default)
	MinH     int    `toml:"min_h"`    // minimum height override (0 = use widget default)
	Priority int    `toml:"priority"` // higher = shown first when space is limited (0-100)
}

// ResolvedCell is a widget with computed position.
type ResolvedCell struct {
	WidgetID string
	X, Y     int
	W, H     int
}

// builtins maps preset names to their definitions.
var builtins map[string]LayoutPreset

func init() {
	builtins = map[string]LayoutPreset{
		"dashboard": prDashboardPreset(),
		"minimal":   prMinimalPreset(),
		"ops":       prOpsPreset(),
		"billing":   prBillingPreset(),
	}
}

// Get returns a named preset, falling back to Dashboard if not found.
func Get(name string) LayoutPreset {
	if p, ok := builtins[name]; ok {
		return p
	}
	return builtins["dashboard"]
}

// Names returns all available preset names in sorted order.
func Names() []string {
	names := make([]string, 0, len(builtins))
	for k := range builtins {
		names = append(names, k)
	}
	sort.Strings(names)
	return names
}

// prNormalizeSlot fills in default ColSpan and RowSpan values.
func prNormalizeSlot(s WidgetSlot) WidgetSlot {
	if s.ColSpan <= 0 {
		s.ColSpan = 1
	}
	if s.RowSpan <= 0 {
		s.RowSpan = 1
	}
	return s
}
