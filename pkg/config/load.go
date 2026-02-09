package config

import (
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/BurntSushi/toml"
)

// Load reads configuration from the standard config path.
// Search order:
//  1. $XDG_CONFIG_HOME/prompt-pulse/config.toml
//  2. ~/.config/prompt-pulse/config.toml
//
// If no file exists, returns DefaultConfig().
func Load() (*Config, error) {
	paths := configSearchPaths()
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return LoadFromFile(p)
		}
	}
	return DefaultConfig(), nil
}

// LoadFromFile reads configuration from a specific file path.
func LoadFromFile(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultConfig(), nil
		}
		return nil, err
	}
	defer f.Close()
	return LoadFromReader(f)
}

// LoadFromReader reads configuration from an io.Reader.
func LoadFromReader(r io.Reader) (*Config, error) {
	cfg := DefaultConfig()
	if _, err := toml.NewDecoder(r).Decode(cfg); err != nil {
		return nil, err
	}
	applyEnvOverrides(cfg)
	return cfg, nil
}

// DefaultConfig returns the default configuration with sensible defaults.
func DefaultConfig() *Config {
	home, _ := os.UserHomeDir()
	cacheDir := filepath.Join(xdgCacheHome(home), "prompt-pulse")

	return &Config{
		General: GeneralConfig{
			DaemonPollInterval: Duration{15 * time.Minute},
			DataRetention:      Duration{10 * time.Minute},
			LogLevel:           "info",
			CacheDir:           cacheDir,
		},
		Layout: LayoutConfig{
			Preset: "dashboard",
		},
		Collectors: CollectorsConfig{
			SysMetrics: SysMetricsCollectorConfig{
				Enabled:  true,
				Interval: Duration{1 * time.Second},
			},
			Tailscale: TailscaleCollectorConfig{
				Enabled:  true,
				Interval: Duration{30 * time.Second},
			},
			Kubernetes: K8sCollectorConfig{
				Enabled:  false,
				Interval: Duration{60 * time.Second},
			},
			Claude: ClaudeCollectorConfig{
				Enabled:  true,
				Interval: Duration{5 * time.Minute},
			},
			Billing: BillingCollectorConfig{
				Enabled:  false,
				Interval: Duration{15 * time.Minute},
			},
		},
		Image: ImageConfig{
			Protocol:       "auto",
			MaxCacheSizeMB: 50,
			MaxSessions:    10,
			WaifuEnabled:   true,
			WaifuCategory:  "waifu",
		},
		Theme: ThemeConfig{
			Name: "default",
		},
		Shell: ShellConfig{
			TUIKeybinding:       `\C-p`,
			ShowBannerOnStartup: true,
			BannerTimeout:       Duration{2 * time.Second},
			InstantBanner:       true,
		},
		Banner: BannerConfig{
			CompactMaxWidth:   80,
			StandardMinWidth:  120,
			WideMinWidth:      160,
			UltraWideMinWidth: 200,
		},
	}
}

// applyEnvOverrides checks environment variables and overrides config values.
func applyEnvOverrides(cfg *Config) {
	if v := os.Getenv("ANTHROPIC_ADMIN_KEY"); v != "" {
		cfg.Collectors.Claude.AdminKey = v
	}
	if v := os.Getenv("CIVO_TOKEN"); v != "" {
		cfg.Collectors.Billing.Civo.APIKey = v
	}
	if v := os.Getenv("DIGITALOCEAN_TOKEN"); v != "" {
		cfg.Collectors.Billing.DigitalOcean.APIKey = v
	}
	if v := os.Getenv("PPULSE_PROTOCOL"); v != "" {
		cfg.Image.Protocol = v
	}
	if v := os.Getenv("PPULSE_THEME"); v != "" {
		cfg.Theme.Name = v
	}
	if v := os.Getenv("PPULSE_LAYOUT"); v != "" {
		cfg.Layout.Preset = v
	}
}

// configSearchPaths returns the ordered list of config file paths to try.
func configSearchPaths() []string {
	home, _ := os.UserHomeDir()
	var paths []string

	xdg := xdgConfigHome(home)
	paths = append(paths, filepath.Join(xdg, "prompt-pulse", "config.toml"))

	// If XDG_CONFIG_HOME was explicitly set, also try the fallback default.
	defaultXDG := filepath.Join(home, ".config")
	if xdg != defaultXDG {
		paths = append(paths, filepath.Join(defaultXDG, "prompt-pulse", "config.toml"))
	}

	return paths
}

// xdgConfigHome returns XDG_CONFIG_HOME or ~/.config as fallback.
func xdgConfigHome(home string) string {
	if v := os.Getenv("XDG_CONFIG_HOME"); v != "" {
		return v
	}
	return filepath.Join(home, ".config")
}

// xdgCacheHome returns XDG_CACHE_HOME or ~/.cache as fallback.
func xdgCacheHome(home string) string {
	if v := os.Getenv("XDG_CACHE_HOME"); v != "" {
		return v
	}
	return filepath.Join(home, ".cache")
}
