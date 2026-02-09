package config

// LayoutPreset returns the layout configuration for a named preset.
// If the name is not recognized, the "dashboard" preset is returned.
func LayoutPreset(name string) LayoutConfig {
	switch name {
	case "minimal":
		return minimalPreset()
	case "ops":
		return opsPreset()
	case "billing":
		return billingPreset()
	case "dashboard":
		return dashboardPreset()
	default:
		return dashboardPreset()
	}
}

// dashboardPreset returns the default full dashboard layout.
//
//	Row 1 (ratio 3): [waifu:2] [claude:3] [billing:3]
//	Row 2 (ratio 4): [tailscale:1] [k8s:1]
//	Row 3 (ratio 3): [sysmetrics:1]
func dashboardPreset() LayoutConfig {
	return LayoutConfig{
		Preset: "dashboard",
		Rows: []RowConfig{
			{
				Ratio: 3,
				Children: []ChildConfig{
					{Type: "waifu", Ratio: 2},
					{Type: "claude", Ratio: 3},
					{Type: "billing", Ratio: 3},
				},
			},
			{
				Ratio: 4,
				Children: []ChildConfig{
					{Type: "tailscale", Ratio: 1},
					{Type: "k8s", Ratio: 1},
				},
			},
			{
				Ratio: 3,
				Children: []ChildConfig{
					{Type: "sysmetrics", Ratio: 1},
				},
			},
		},
	}
}

// minimalPreset returns a simple waifu + Claude layout.
//
//	Row 1 (ratio 1): [waifu:1] [claude:1]
func minimalPreset() LayoutConfig {
	return LayoutConfig{
		Preset: "minimal",
		Rows: []RowConfig{
			{
				Ratio: 1,
				Children: []ChildConfig{
					{Type: "waifu", Ratio: 1},
					{Type: "claude", Ratio: 1},
				},
			},
		},
	}
}

// opsPreset returns an operations-focused layout without waifu.
//
//	Row 1 (ratio 3): [tailscale:1] [k8s:1]
//	Row 2 (ratio 3): [sysmetrics:1]
//	Row 3 (ratio 2): [claude:1] [billing:1]
func opsPreset() LayoutConfig {
	return LayoutConfig{
		Preset: "ops",
		Rows: []RowConfig{
			{
				Ratio: 3,
				Children: []ChildConfig{
					{Type: "tailscale", Ratio: 1},
					{Type: "k8s", Ratio: 1},
				},
			},
			{
				Ratio: 3,
				Children: []ChildConfig{
					{Type: "sysmetrics", Ratio: 1},
				},
			},
			{
				Ratio: 2,
				Children: []ChildConfig{
					{Type: "claude", Ratio: 1},
					{Type: "billing", Ratio: 1},
				},
			},
		},
	}
}

// billingPreset returns a financial-focused layout.
//
//	Row 1 (ratio 2): [claude:1]
//	Row 2 (ratio 3): [billing:1]
//	Row 3 (ratio 2): [sysmetrics:1]
func billingPreset() LayoutConfig {
	return LayoutConfig{
		Preset: "billing",
		Rows: []RowConfig{
			{
				Ratio: 2,
				Children: []ChildConfig{
					{Type: "claude", Ratio: 1},
				},
			},
			{
				Ratio: 3,
				Children: []ChildConfig{
					{Type: "billing", Ratio: 1},
				},
			},
			{
				Ratio: 2,
				Children: []ChildConfig{
					{Type: "sysmetrics", Ratio: 1},
				},
			},
		},
	}
}
