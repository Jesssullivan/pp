// Package app provides the core Bubbletea application framework for
// prompt-pulse v2. It defines the event types, root model, widget interface,
// and navigation logic that form the Elm-architecture skeleton.
//
// This package is designed against bubbletea v1.3.x but architected so that
// migrating to v2 requires only import-path changes and minor API adjustments.
package app

import "time"

// DataUpdateEvent carries new data from a collector goroutine back into the
// bubbletea update loop. Receivers type-assert Data based on Source.
type DataUpdateEvent struct {
	Source    string      // Collector name (e.g., "tailscale", "claude", "sysmetrics")
	Data     interface{} // Type-asserted by the receiver
	Err      error       // Non-nil if the fetch failed
	Timestamp time.Time
}

// TickEvent is sent periodically by the render ticker to trigger UI refresh
// and stale-data checks.
type TickEvent struct {
	Time time.Time
}

// WidgetFocusEvent requests that focus move to a specific widget.
type WidgetFocusEvent struct {
	WidgetID string
}

// WidgetExpandEvent toggles a widget between normal and fullscreen mode.
type WidgetExpandEvent struct {
	WidgetID string
}

// ThemeChangeEvent switches the active color theme.
type ThemeChangeEvent struct {
	Theme string
}

// LayoutPresetEvent switches to a named layout preset (e.g., "default",
// "compact", "wide").
type LayoutPresetEvent struct {
	Preset string
}
