// Package widgets provides the concrete widget implementations for the
// prompt-pulse v2 TUI dashboard. Each widget implements the app.Widget
// interface and receives data via the Elm-architecture Update loop.
package widgets

// Common color constants for widget border and accent styling.
const (
	// ColorBorderDefault is the muted gray used for unfocused widget borders.
	ColorBorderDefault = "#6B7280"

	// ColorBorderFocus is the purple used for focused widget borders.
	ColorBorderFocus = "#7C3AED"

	// ColorAccent is a softer purple for titles and highlights.
	ColorAccent = "#A78BFA"

	// ColorDim is used for de-emphasized text such as overlays.
	ColorDim = "#9CA3AF"

	// ColorError is used for error message text.
	ColorError = "#EF4444"
)
