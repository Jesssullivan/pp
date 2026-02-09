// Package banner provides the non-interactive, single-frame rendering of the
// prompt-pulse dashboard. It composes pre-rendered widget content into a
// multi-column layout using a greedy column-packing algorithm, and renders the
// result to a fixed-size character grid suitable for display on shell startup.
package banner

// Preset defines a named layout preset with target dimensions.
type Preset struct {
	Name   string
	Width  int
	Height int
}

var (
	// Compact is a single-column layout for narrow terminals.
	Compact = Preset{"compact", 80, 24}
	// Standard is a two-column layout for typical terminals.
	Standard = Preset{"standard", 120, 35}
	// Wide is a three-column layout for wide terminals.
	Wide = Preset{"wide", 160, 45}
	// UltraWide is a three-column layout with wider waifu for ultra-wide terminals.
	UltraWide = Preset{"ultrawide", 200, 50}
)

// SelectPreset chooses the best preset for the given terminal dimensions.
// The selected preset is the largest one whose width and height both fit
// within the terminal. If none fit, Compact is returned.
func SelectPreset(termWidth, termHeight int) Preset {
	// Order from largest to smallest; return the first that fits.
	presets := []Preset{UltraWide, Wide, Standard, Compact}
	for _, p := range presets {
		if termWidth >= p.Width && termHeight >= p.Height {
			return p
		}
	}
	return Compact
}

// BannerData holds pre-collected data for all widgets.
type BannerData struct {
	Widgets []WidgetData
}

// WidgetData holds the data for a single widget to render.
type WidgetData struct {
	ID      string
	Title   string
	Content string // pre-rendered content from widget.View()
	MinW    int
	MinH    int
}

// Render composes all widget content into a banner string using the given preset.
// It arranges widgets in a multi-column layout respecting minimum sizes, wraps
// each widget in a bordered box, and places everything onto a fixed-size
// character grid.
func Render(data BannerData, preset Preset) string {
	placements := bnArrangeWidgets(data.Widgets, preset.Width, preset.Height)
	return bnCompose(placements, preset.Width, preset.Height)
}
