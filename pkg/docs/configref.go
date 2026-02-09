package docs

import (
	"fmt"
	"strings"
)

// ConfigRef holds the full configuration reference documentation.
type ConfigRef struct {
	// Sections groups config fields by TOML table.
	Sections []ConfigSection
}

// ConfigSection documents a single TOML table (e.g., [general]).
type ConfigSection struct {
	// Name is the TOML table name (e.g., "general").
	Name string

	// Description summarizes what this section configures.
	Description string

	// Fields lists the config keys in this section.
	Fields []ConfigField
}

// ConfigField documents a single config key.
type ConfigField struct {
	// Name is the TOML key name.
	Name string

	// Type is the Go/TOML type (e.g., "string", "bool", "duration").
	Type string

	// Default is the default value as a string.
	Default string

	// Description explains what this field controls.
	Description string

	// Required indicates whether this field must be set.
	Required bool

	// Example is a TOML snippet showing usage.
	Example string
}

// dcGenerateConfigRef builds the full configuration reference from embedded knowledge.
func dcGenerateConfigRef() *ConfigRef {
	return &ConfigRef{
		Sections: []ConfigSection{
			dcGeneralSection(),
			dcLayoutSection(),
			dcCollectorsSysMetricsSection(),
			dcCollectorsTailscaleSection(),
			dcCollectorsK8sSection(),
			dcCollectorsClaudeSection(),
			dcCollectorsBillingSection(),
			dcImageSection(),
			dcThemeSection(),
			dcShellSection(),
			dcBannerSection(),
		},
	}
}

// dcRenderConfigMarkdown renders a ConfigRef as a Markdown document with TOML examples.
func dcRenderConfigMarkdown(ref *ConfigRef) string {
	var b strings.Builder

	b.WriteString("# Configuration Reference\n\n")
	b.WriteString("prompt-pulse v2 uses TOML configuration.\n\n")
	b.WriteString("Config file location: `$XDG_CONFIG_HOME/prompt-pulse/config.toml`\n\n")

	for _, s := range ref.Sections {
		b.WriteString(fmt.Sprintf("## `[%s]`\n\n", s.Name))
		b.WriteString(s.Description + "\n\n")

		// Fields table
		b.WriteString("| Key | Type | Default | Required | Description |\n")
		b.WriteString("|-----|------|---------|----------|-------------|\n")
		for _, f := range s.Fields {
			req := "No"
			if f.Required {
				req = "Yes"
			}
			def := f.Default
			if def == "" {
				def = "-"
			}
			b.WriteString(fmt.Sprintf("| `%s` | %s | `%s` | %s | %s |\n",
				f.Name, f.Type, def, req, f.Description))
		}
		b.WriteString("\n")

		// TOML example
		b.WriteString("**Example:**\n\n")
		b.WriteString("```toml\n")
		b.WriteString(fmt.Sprintf("[%s]\n", s.Name))
		for _, f := range s.Fields {
			if f.Example != "" {
				b.WriteString(f.Example + "\n")
			}
		}
		b.WriteString("```\n\n")
	}

	return b.String()
}

func dcGeneralSection() ConfigSection {
	return ConfigSection{
		Name:        "general",
		Description: "Top-level daemon and application settings.",
		Fields: []ConfigField{
			{
				Name:        "log_level",
				Type:        "string",
				Default:     "info",
				Description: "Logging verbosity: debug, info, warn, error",
				Example:     `log_level = "info"`,
			},
			{
				Name:        "cache_dir",
				Type:        "string",
				Default:     "$XDG_CACHE_HOME/prompt-pulse",
				Description: "Override the default cache directory path",
				Example:     `cache_dir = "/tmp/ppulse-cache"`,
			},
			{
				Name:        "daemon_poll_interval",
				Type:        "duration",
				Default:     "15m",
				Description: "Base polling interval for the background daemon",
				Example:     `daemon_poll_interval = "15m"`,
			},
			{
				Name:        "data_retention",
				Type:        "duration",
				Default:     "10m",
				Description: "How long time-series data is retained in memory",
				Example:     `data_retention = "10m"`,
			},
		},
	}
}

func dcLayoutSection() ConfigSection {
	return ConfigSection{
		Name:        "layout",
		Description: "Dashboard layout configuration via presets or custom row definitions.",
		Fields: []ConfigField{
			{
				Name:        "preset",
				Type:        "string",
				Default:     "dashboard",
				Description: "Built-in layout preset: dashboard, minimal, ops, billing",
				Example:     `preset = "dashboard"`,
			},
		},
	}
}

func dcCollectorsSysMetricsSection() ConfigSection {
	return ConfigSection{
		Name:        "collectors.sysmetrics",
		Description: "System metrics collection: CPU, memory, disk, GPU, and network.",
		Fields: []ConfigField{
			{
				Name:        "enabled",
				Type:        "bool",
				Default:     "true",
				Description: "Enable system metrics collection",
				Example:     `enabled = true`,
			},
			{
				Name:        "interval",
				Type:        "duration",
				Default:     "1s",
				Description: "Collection interval for system metrics",
				Example:     `interval = "1s"`,
			},
		},
	}
}

func dcCollectorsTailscaleSection() ConfigSection {
	return ConfigSection{
		Name:        "collectors.tailscale",
		Description: "Tailscale mesh network status: peer list, exit nodes, connection health.",
		Fields: []ConfigField{
			{
				Name:        "enabled",
				Type:        "bool",
				Default:     "true",
				Description: "Enable Tailscale status collection",
				Example:     `enabled = true`,
			},
			{
				Name:        "interval",
				Type:        "duration",
				Default:     "30s",
				Description: "Collection interval for Tailscale status",
				Example:     `interval = "30s"`,
			},
		},
	}
}

func dcCollectorsK8sSection() ConfigSection {
	return ConfigSection{
		Name:        "collectors.kubernetes",
		Description: "Kubernetes cluster status: deployments, pods, nodes across configured contexts.",
		Fields: []ConfigField{
			{
				Name:        "enabled",
				Type:        "bool",
				Default:     "false",
				Description: "Enable Kubernetes status collection",
				Example:     `enabled = false`,
			},
			{
				Name:        "interval",
				Type:        "duration",
				Default:     "60s",
				Description: "Collection interval for Kubernetes status",
				Example:     `interval = "60s"`,
			},
			{
				Name:        "contexts",
				Type:        "[]string",
				Default:     "[]",
				Description: "Kubernetes contexts to monitor (empty = current context)",
				Example:     `contexts = ["prod", "staging"]`,
			},
			{
				Name:        "namespaces",
				Type:        "[]string",
				Default:     "[]",
				Description: "Namespaces to monitor (empty = all namespaces)",
				Example:     `namespaces = ["default", "kube-system"]`,
			},
		},
	}
}

func dcCollectorsClaudeSection() ConfigSection {
	return ConfigSection{
		Name:        "collectors.claude",
		Description: "Claude API usage tracking: token counts, costs, and rate limits per account.",
		Fields: []ConfigField{
			{
				Name:        "enabled",
				Type:        "bool",
				Default:     "true",
				Description: "Enable Claude usage collection",
				Example:     `enabled = true`,
			},
			{
				Name:        "interval",
				Type:        "duration",
				Default:     "5m",
				Description: "Collection interval for Claude usage data",
				Example:     `interval = "5m"`,
			},
			{
				Name:        "admin_key",
				Type:        "string",
				Default:     "",
				Description: "Anthropic Admin API key (prefer ANTHROPIC_ADMIN_KEY env var)",
				Example:     `# admin_key = "sk-ant-admin-..."  # prefer env var`,
			},
		},
	}
}

func dcCollectorsBillingSection() ConfigSection {
	return ConfigSection{
		Name:        "collectors.billing",
		Description: "Cloud billing data: Civo and DigitalOcean spend tracking and budget alerts.",
		Fields: []ConfigField{
			{
				Name:        "enabled",
				Type:        "bool",
				Default:     "false",
				Description: "Enable billing data collection",
				Example:     `enabled = false`,
			},
			{
				Name:        "interval",
				Type:        "duration",
				Default:     "15m",
				Description: "Collection interval for billing data",
				Example:     `interval = "15m"`,
			},
		},
	}
}

func dcImageSection() ConfigSection {
	return ConfigSection{
		Name:        "image",
		Description: "Image rendering and waifu display settings.",
		Fields: []ConfigField{
			{
				Name:        "protocol",
				Type:        "string",
				Default:     "auto",
				Description: "Image protocol: auto, kitty, iterm2, sixel, halfblocks, none",
				Example:     `protocol = "auto"`,
			},
			{
				Name:        "max_cache_size_mb",
				Type:        "int",
				Default:     "50",
				Description: "Maximum disk cache size for images in MB",
				Example:     `max_cache_size_mb = 50`,
			},
			{
				Name:        "max_sessions",
				Type:        "int",
				Default:     "10",
				Description: "Maximum number of per-session cached images",
				Example:     `max_sessions = 10`,
			},
			{
				Name:        "waifu_enabled",
				Type:        "bool",
				Default:     "true",
				Description: "Toggle waifu image display in banner and TUI",
				Example:     `waifu_enabled = true`,
			},
			{
				Name:        "waifu_category",
				Type:        "string",
				Default:     "waifu",
				Description: "Waifu API category for image fetching",
				Example:     `waifu_category = "waifu"`,
			},
		},
	}
}

func dcThemeSection() ConfigSection {
	return ConfigSection{
		Name:        "theme",
		Description: "Visual theme selection. Six built-in themes are available.",
		Fields: []ConfigField{
			{
				Name:        "name",
				Type:        "string",
				Default:     "default",
				Description: "Theme name: default, gruvbox, nord, catppuccin, dracula, tokyo-night",
				Example:     `name = "catppuccin"`,
			},
		},
	}
}

func dcShellSection() ConfigSection {
	return ConfigSection{
		Name:        "shell",
		Description: "Shell integration settings for hook behavior and keybindings.",
		Fields: []ConfigField{
			{
				Name:        "tui_keybinding",
				Type:        "string",
				Default:     `\C-p`,
				Description: "Keybinding to toggle the TUI overlay",
				Example:     `tui_keybinding = "\\C-p"`,
			},
			{
				Name:        "show_banner_on_startup",
				Type:        "bool",
				Default:     "true",
				Description: "Display a banner when a new shell session starts",
				Example:     `show_banner_on_startup = true`,
			},
			{
				Name:        "banner_timeout",
				Type:        "duration",
				Default:     "2s",
				Description: "Maximum time to wait for daemon data before showing banner",
				Example:     `banner_timeout = "2s"`,
			},
			{
				Name:        "instant_banner",
				Type:        "bool",
				Default:     "true",
				Description: "Use pre-rendered cache for sub-millisecond banner display",
				Example:     `instant_banner = true`,
			},
		},
	}
}

func dcBannerSection() ConfigSection {
	return ConfigSection{
		Name:        "banner",
		Description: "Banner width thresholds for adaptive layout modes.",
		Fields: []ConfigField{
			{
				Name:        "compact_max_width",
				Type:        "int",
				Default:     "80",
				Description: "Maximum terminal width for compact banner mode",
				Example:     `compact_max_width = 80`,
			},
			{
				Name:        "standard_min_width",
				Type:        "int",
				Default:     "120",
				Description: "Minimum terminal width for standard banner mode",
				Example:     `standard_min_width = 120`,
			},
			{
				Name:        "wide_min_width",
				Type:        "int",
				Default:     "160",
				Description: "Minimum terminal width for wide banner mode",
				Example:     `wide_min_width = 160`,
			},
			{
				Name:        "ultrawide_min_width",
				Type:        "int",
				Default:     "200",
				Description: "Minimum terminal width for ultra-wide banner mode",
				Example:     `ultrawide_min_width = 200`,
			},
		},
	}
}
