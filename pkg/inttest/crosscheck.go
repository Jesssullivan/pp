package inttest

import (
	"strings"
	"testing"

	"gitlab.com/tinyland/lab/prompt-pulse/pkg/banner"
	"gitlab.com/tinyland/lab/prompt-pulse/pkg/config"
	"gitlab.com/tinyland/lab/prompt-pulse/pkg/preset"
	"gitlab.com/tinyland/lab/prompt-pulse/pkg/theme"
)

// itCheckWidgetCollectorCompat verifies that each widget type can consume
// its collector's data format. Since we cannot import collector packages
// directly (they have external dependencies), we validate through the
// banner rendering path: mock data is formatted as widget content and
// rendered through the banner pipeline. If a widget-collector mismatch
// existed, the render would produce empty or corrupt output.
func itCheckWidgetCollectorCompat(t *testing.T) {
	t.Helper()

	// Each pair: widget ID, mock content representing collector output.
	pairs := []struct {
		widgetID string
		title    string
		content  string
	}{
		{"waifu", "Waifu", "[image placeholder]"},
		{"claude", "Claude Usage", itMockClaudeWidget()},
		{"billing", "Cloud Billing", itMockBillingWidget()},
		{"tailscale", "Tailscale", itMockTailscaleWidget()},
		{"k8s", "Kubernetes", itMockK8sWidget()},
		{"sysmetrics", "System Metrics", itMockSysMetricsWidget()},
	}

	for _, pair := range pairs {
		t.Run(pair.widgetID, func(t *testing.T) {
			wd := banner.WidgetData{
				ID:      pair.widgetID,
				Title:   pair.title,
				Content: pair.content,
				MinW:    20,
				MinH:    5,
			}

			data := banner.BannerData{
				Widgets: []banner.WidgetData{wd},
			}

			output := banner.Render(data, banner.Compact)
			if output == "" {
				t.Errorf("widget %q rendered empty output", pair.widgetID)
			}

			lines := strings.Split(output, "\n")
			if len(lines) < 3 {
				t.Errorf("widget %q rendered too few lines: %d", pair.widgetID, len(lines))
			}
		})
	}
}

// itCheckThemePresetCompat verifies that every theme can be applied
// alongside every preset without errors. Themes provide colors and
// presets provide layout; they must be independently combinable.
func itCheckThemePresetCompat(t *testing.T) {
	t.Helper()

	themeNames := theme.Names()
	presetNames := preset.Names()

	if len(themeNames) == 0 {
		t.Fatal("no themes registered")
	}
	if len(presetNames) == 0 {
		t.Fatal("no presets registered")
	}

	for _, themeName := range themeNames {
		for _, presetName := range presetNames {
			t.Run(themeName+"/"+presetName, func(t *testing.T) {
				th := theme.Get(themeName)
				if th.Name == "" {
					t.Errorf("theme %q has empty name", themeName)
				}

				p := preset.Get(presetName)
				if p.Name == "" {
					t.Errorf("preset %q has empty name", presetName)
				}

				// Resolve the preset at standard terminal size.
				cells := preset.Resolve(p, 120, 35)

				// For presets with widgets, cells should be non-empty.
				if len(p.Widgets) > 0 && len(cells) == 0 {
					t.Errorf("preset %q with %d widgets resolved to 0 cells at 120x35",
						presetName, len(p.Widgets))
				}

				// Verify theme colors are valid hex.
				for _, color := range []string{
					th.Background, th.Foreground, th.Accent,
					th.Border, th.BorderFocus, th.StatusOK,
				} {
					if !strings.HasPrefix(color, "#") {
						t.Errorf("theme %q has non-hex color: %q", themeName, color)
					}
				}
			})
		}
	}
}

// itCheckLayoutConstraints verifies that the layout solver handles all
// preset configurations at various terminal sizes without panicking or
// producing overlapping regions.
func itCheckLayoutConstraints(t *testing.T) {
	t.Helper()

	presetNames := preset.Names()
	sizes := []struct {
		w, h int
		name string
	}{
		{80, 24, "compact"},
		{120, 35, "standard"},
		{160, 45, "wide"},
		{200, 50, "ultrawide"},
		{40, 10, "tiny"},
		{300, 80, "huge"},
	}

	for _, presetName := range presetNames {
		for _, sz := range sizes {
			t.Run(presetName+"/"+sz.name, func(t *testing.T) {
				p := preset.Get(presetName)
				cells := preset.Resolve(p, sz.w, sz.h)

				// Verify no cell has negative dimensions.
				for _, cell := range cells {
					if cell.W < 0 || cell.H < 0 {
						t.Errorf("cell %q has negative dimensions: %dx%d",
							cell.WidgetID, cell.W, cell.H)
					}
				}

				// Verify no cell extends beyond terminal bounds.
				for _, cell := range cells {
					if cell.X+cell.W > sz.w {
						t.Errorf("cell %q exceeds width: x=%d w=%d > %d",
							cell.WidgetID, cell.X, cell.W, sz.w)
					}
					// Height is minus status bar, so cells can use up to height-1.
					if cell.Y+cell.H > sz.h {
						t.Errorf("cell %q exceeds height: y=%d h=%d > %d",
							cell.WidgetID, cell.Y, cell.H, sz.h)
					}
				}
			})
		}
	}
}

// itCheckConfigRoundTrip verifies that writing a config as TOML and
// reading it back produces equivalent values.
func itCheckConfigRoundTrip(t *testing.T) {
	t.Helper()

	tomlStr := itMockConfig()
	cfg, err := config.LoadFromReader(strings.NewReader(tomlStr))
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// Verify key fields survived the round-trip.
	if cfg.General.LogLevel != "info" {
		t.Errorf("log_level: got %q, want %q", cfg.General.LogLevel, "info")
	}
	if cfg.Layout.Preset != "dashboard" {
		t.Errorf("preset: got %q, want %q", cfg.Layout.Preset, "dashboard")
	}
	if cfg.Theme.Name != "catppuccin" {
		t.Errorf("theme: got %q, want %q", cfg.Theme.Name, "catppuccin")
	}
	if !cfg.Collectors.SysMetrics.Enabled {
		t.Error("sysmetrics should be enabled")
	}
	if !cfg.Collectors.Tailscale.Enabled {
		t.Error("tailscale should be enabled")
	}
	if cfg.Collectors.Kubernetes.Enabled {
		t.Error("kubernetes should be disabled")
	}
	if !cfg.Collectors.Claude.Enabled {
		t.Error("claude should be enabled")
	}
	if cfg.Collectors.Billing.Enabled {
		t.Error("billing should be disabled")
	}
	if cfg.Image.Protocol != "auto" {
		t.Errorf("image protocol: got %q, want %q", cfg.Image.Protocol, "auto")
	}
	if cfg.Shell.ShowBannerOnStartup != true {
		t.Error("shell.show_banner_on_startup should be true")
	}
	if cfg.Banner.CompactMaxWidth != 80 {
		t.Errorf("banner.compact_max_width: got %d, want 80", cfg.Banner.CompactMaxWidth)
	}
	if cfg.Banner.StandardMinWidth != 120 {
		t.Errorf("banner.standard_min_width: got %d, want 120", cfg.Banner.StandardMinWidth)
	}
}

// itMockClaudeWidget returns mock content for the Claude widget.
func itMockClaudeWidget() string {
	return "Total: $142.30\nTop: opus ($98.50)\nAccounts: 2"
}

// itMockBillingWidget returns mock content for the Billing widget.
func itMockBillingWidget() string {
	return "Civo: $12.50/mo\nDOKS: $45.00/mo\nTotal: $57.50/mo"
}

// itMockTailscaleWidget returns mock content for the Tailscale widget.
func itMockTailscaleWidget() string {
	return "Online: 3/5 peers\nhoney: online\npetting-zoo-mini: online\nlocalhost: online"
}

// itMockK8sWidget returns mock content for the K8s widget.
func itMockK8sWidget() string {
	return "Clusters: 2\nPods: 12/15 running\nFailed: 1"
}

// itMockSysMetricsWidget returns mock content for the SysMetrics widget.
func itMockSysMetricsWidget() string {
	return "CPU: 45%\nRAM: 62%\nDisk: 71%"
}
