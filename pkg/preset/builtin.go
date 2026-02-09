package preset

// prDashboardPreset returns the default 2-column grid.
// Left column: waifu (60% height), system metrics (40%).
// Right column: claude, billing, tailscale, k8s.
func prDashboardPreset() LayoutPreset {
	return LayoutPreset{
		Name:        "dashboard",
		Description: "Full dashboard with all widgets in a 2-column grid",
		Columns:     2,
		Widgets: []WidgetSlot{
			{WidgetID: "waifu", Column: 0, Row: 0, ColSpan: 1, RowSpan: 3, Priority: 70},
			{WidgetID: "sysmetrics", Column: 0, Row: 3, ColSpan: 1, RowSpan: 2, Priority: 60},
			{WidgetID: "claude", Column: 1, Row: 0, ColSpan: 1, RowSpan: 1, Priority: 90},
			{WidgetID: "billing", Column: 1, Row: 1, ColSpan: 1, RowSpan: 1, Priority: 50},
			{WidgetID: "tailscale", Column: 1, Row: 2, ColSpan: 1, RowSpan: 1, Priority: 40},
			{WidgetID: "k8s", Column: 1, Row: 3, ColSpan: 1, RowSpan: 2, Priority: 30},
		},
	}
}

// prMinimalPreset returns a simple 2-column layout.
// Left: waifu (full height). Right: claude only.
func prMinimalPreset() LayoutPreset {
	return LayoutPreset{
		Name:        "minimal",
		Description: "Quick glance with waifu and Claude only",
		Columns:     2,
		Widgets: []WidgetSlot{
			{WidgetID: "waifu", Column: 0, Row: 0, ColSpan: 1, RowSpan: 1, Priority: 80},
			{WidgetID: "claude", Column: 1, Row: 0, ColSpan: 1, RowSpan: 1, Priority: 100},
		},
	}
}

// prOpsPreset returns an infrastructure-focused 2-column layout.
// Left: k8s (60%), tailscale (40%). Right: system metrics (50%), claude (50%).
// No waifu widget.
func prOpsPreset() LayoutPreset {
	return LayoutPreset{
		Name:        "ops",
		Description: "Infrastructure focus without waifu",
		Columns:     2,
		Widgets: []WidgetSlot{
			{WidgetID: "k8s", Column: 0, Row: 0, ColSpan: 1, RowSpan: 3, Priority: 90},
			{WidgetID: "tailscale", Column: 0, Row: 3, ColSpan: 1, RowSpan: 2, Priority: 80},
			{WidgetID: "sysmetrics", Column: 1, Row: 0, ColSpan: 1, RowSpan: 2, Priority: 70},
			{WidgetID: "claude", Column: 1, Row: 2, ColSpan: 1, RowSpan: 2, Priority: 60},
		},
	}
}

// prBillingPreset returns a cost-tracking-focused 2-column layout.
// Left: claude (full height). Right: billing (60%), system metrics (40%).
// No waifu widget.
func prBillingPreset() LayoutPreset {
	return LayoutPreset{
		Name:        "billing",
		Description: "Cost tracking focus with Claude and billing",
		Columns:     2,
		Widgets: []WidgetSlot{
			{WidgetID: "claude", Column: 0, Row: 0, ColSpan: 1, RowSpan: 1, Priority: 100},
			{WidgetID: "billing", Column: 1, Row: 0, ColSpan: 1, RowSpan: 3, Priority: 90},
			{WidgetID: "sysmetrics", Column: 1, Row: 3, ColSpan: 1, RowSpan: 2, Priority: 60},
		},
	}
}
