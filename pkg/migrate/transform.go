package migrate

import (
	"fmt"
	"time"

	"gitlab.com/tinyland/lab/prompt-pulse/pkg/config"
)

// mgTransformConfig converts a v1 configuration to a v2 Config,
// returning the new config and a list of all changes made.
func mgTransformConfig(v1 *V1Config) (*config.Config, []ConfigChange) {
	v2 := config.DefaultConfig()
	var changes []ConfigChange

	// Waifu / Image settings
	if v1.WaifuEnabled != v2.Image.WaifuEnabled {
		changes = append(changes, ConfigChange{
			Field:    "image.waifu_enabled",
			OldValue: fmt.Sprintf("%v", v1.WaifuEnabled),
			NewValue: fmt.Sprintf("%v", v2.Image.WaifuEnabled),
			Action:   "changed",
		})
	}
	v2.Image.WaifuEnabled = v1.WaifuEnabled

	if v1.WaifuPath != "" {
		changes = append(changes, ConfigChange{
			Field:    "image.waifu_path",
			OldValue: v1.WaifuPath,
			NewValue: "(removed, v2 uses API-based waifu)",
			Action:   "removed",
		})
	}

	if v1.BannerShowWaifu != v1.WaifuEnabled {
		changes = append(changes, ConfigChange{
			Field:    "image.waifu_enabled",
			OldValue: fmt.Sprintf("banner_show_waifu=%v", v1.BannerShowWaifu),
			NewValue: fmt.Sprintf("%v", v1.WaifuEnabled),
			Action:   "changed",
		})
	}

	// Cache settings
	if v1.CacheDir != "" && v1.CacheDir != v2.General.CacheDir {
		changes = append(changes, ConfigChange{
			Field:    "general.cache_dir",
			OldValue: v1.CacheDir,
			NewValue: v1.CacheDir,
			Action:   "changed",
		})
		v2.General.CacheDir = v1.CacheDir
	}

	if v1.CacheTTL.Duration > 0 && v1.CacheTTL.Duration != v2.General.DataRetention.Duration {
		changes = append(changes, ConfigChange{
			Field:    "general.data_retention",
			OldValue: v1.CacheTTL.Duration.String(),
			NewValue: v1.CacheTTL.Duration.String(),
			Action:   "changed",
		})
		v2.General.DataRetention = config.Duration{Duration: v1.CacheTTL.Duration}
	}

	// Theme
	if v1.Theme != "" && v1.Theme != v2.Theme.Name {
		changes = append(changes, ConfigChange{
			Field:    "theme.name",
			OldValue: v1.Theme,
			NewValue: v1.Theme,
			Action:   "changed",
		})
		v2.Theme.Name = v1.Theme
	}

	// Collector toggles
	changes = append(changes, mgTransformCollector(
		"collectors.tailscale.enabled",
		v1.TailscaleEnabled,
		v2.Collectors.Tailscale.Enabled,
	)...)
	v2.Collectors.Tailscale.Enabled = v1.TailscaleEnabled

	changes = append(changes, mgTransformCollector(
		"collectors.kubernetes.enabled",
		v1.K8sEnabled,
		v2.Collectors.Kubernetes.Enabled,
	)...)
	v2.Collectors.Kubernetes.Enabled = v1.K8sEnabled

	changes = append(changes, mgTransformCollector(
		"collectors.claude.enabled",
		v1.ClaudeEnabled,
		v2.Collectors.Claude.Enabled,
	)...)
	v2.Collectors.Claude.Enabled = v1.ClaudeEnabled

	changes = append(changes, mgTransformCollector(
		"collectors.billing.enabled",
		v1.BillingEnabled,
		v2.Collectors.Billing.Enabled,
	)...)
	v2.Collectors.Billing.Enabled = v1.BillingEnabled

	// Banner width -> v2 banner thresholds
	if v1.BannerWidth > 0 && v1.BannerWidth != v2.Banner.StandardMinWidth {
		changes = append(changes, ConfigChange{
			Field:    "banner.standard_min_width",
			OldValue: fmt.Sprintf("%d", v1.BannerWidth),
			NewValue: fmt.Sprintf("%d", v1.BannerWidth),
			Action:   "changed",
		})
		v2.Banner.StandardMinWidth = v1.BannerWidth
	}

	// Starship -> shell integration
	if v1.StarshipEnabled {
		changes = append(changes, ConfigChange{
			Field:    "shell.show_banner_on_startup",
			OldValue: fmt.Sprintf("starship_enabled=%v", v1.StarshipEnabled),
			NewValue: "true",
			Action:   "changed",
		})
		v2.Shell.ShowBannerOnStartup = true
	}

	if v1.StarshipFormat != "" {
		changes = append(changes, ConfigChange{
			Field:    "starship.format",
			OldValue: v1.StarshipFormat,
			NewValue: "(starship config is now external)",
			Action:   "removed",
		})
	}

	// Daemon socket/PID -> added as warnings (v2 manages these internally)
	if v1.DaemonSocket != "" {
		changes = append(changes, ConfigChange{
			Field:    "daemon.socket",
			OldValue: v1.DaemonSocket,
			NewValue: "(v2 manages socket path automatically)",
			Action:   "removed",
		})
	}

	if v1.DaemonPidFile != "" {
		changes = append(changes, ConfigChange{
			Field:    "daemon.pid_file",
			OldValue: v1.DaemonPidFile,
			NewValue: "(v2 manages PID file automatically)",
			Action:   "removed",
		})
	}

	// New v2-only fields (added with defaults)
	changes = append(changes, mgNewV2Fields(v2)...)

	return v2, changes
}

// mgTransformCollector creates a ConfigChange if a collector's enabled state differs.
func mgTransformCollector(field string, v1Enabled, v2Default bool) []ConfigChange {
	if v1Enabled == v2Default {
		return nil
	}
	return []ConfigChange{{
		Field:    field,
		OldValue: fmt.Sprintf("%v", v1Enabled),
		NewValue: fmt.Sprintf("%v", v1Enabled),
		Action:   "changed",
	}}
}

// mgNewV2Fields returns ConfigChange entries for fields that are new in v2.
func mgNewV2Fields(v2 *config.Config) []ConfigChange {
	return []ConfigChange{
		{
			Field:    "general.daemon_poll_interval",
			OldValue: "",
			NewValue: v2.General.DaemonPollInterval.Duration.String(),
			Action:   "added",
		},
		{
			Field:    "general.log_level",
			OldValue: "",
			NewValue: v2.General.LogLevel,
			Action:   "added",
		},
		{
			Field:    "layout.preset",
			OldValue: "",
			NewValue: v2.Layout.Preset,
			Action:   "added",
		},
		{
			Field:    "image.protocol",
			OldValue: "",
			NewValue: v2.Image.Protocol,
			Action:   "added",
		},
		{
			Field:    "image.max_cache_size_mb",
			OldValue: "",
			NewValue: fmt.Sprintf("%d", v2.Image.MaxCacheSizeMB),
			Action:   "added",
		},
		{
			Field:    "image.max_sessions",
			OldValue: "",
			NewValue: fmt.Sprintf("%d", v2.Image.MaxSessions),
			Action:   "added",
		},
		{
			Field:    "image.waifu_category",
			OldValue: "",
			NewValue: v2.Image.WaifuCategory,
			Action:   "added",
		},
		{
			Field:    "shell.tui_keybinding",
			OldValue: "",
			NewValue: v2.Shell.TUIKeybinding,
			Action:   "added",
		},
		{
			Field:    "shell.banner_timeout",
			OldValue: "",
			NewValue: v2.Shell.BannerTimeout.Duration.String(),
			Action:   "added",
		},
		{
			Field:    "shell.instant_banner",
			OldValue: "",
			NewValue: fmt.Sprintf("%v", v2.Shell.InstantBanner),
			Action:   "added",
		},
		{
			Field:    "collectors.sysmetrics.enabled",
			OldValue: "",
			NewValue: fmt.Sprintf("%v", v2.Collectors.SysMetrics.Enabled),
			Action:   "added",
		},
		{
			Field:    "collectors.sysmetrics.interval",
			OldValue: "",
			NewValue: (1 * time.Second).String(),
			Action:   "added",
		},
	}
}
