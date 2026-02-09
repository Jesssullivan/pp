package migrate

import (
	"fmt"
	"os"
	"time"

	"github.com/BurntSushi/toml"
)

// V1Config represents the flat configuration format used by prompt-pulse v1.
type V1Config struct {
	// WaifuPath is the filesystem path to a local waifu image.
	WaifuPath string `toml:"waifu_path"`

	// WaifuEnabled toggles waifu image display.
	WaifuEnabled bool `toml:"waifu_enabled"`

	// DaemonSocket is the Unix domain socket path for the daemon.
	DaemonSocket string `toml:"daemon_socket"`

	// DaemonPidFile is the PID file location for the daemon.
	DaemonPidFile string `toml:"daemon_pid_file"`

	// CacheDir is the directory for cached data.
	CacheDir string `toml:"cache_dir"`

	// CacheTTL is the duration that cached data remains valid.
	CacheTTL duration `toml:"cache_ttl"`

	// BannerWidth is the terminal width for banner rendering.
	BannerWidth int `toml:"banner_width"`

	// BannerShowWaifu controls whether the banner includes a waifu image.
	BannerShowWaifu bool `toml:"banner_show_waifu"`

	// TailscaleEnabled toggles the Tailscale status collector.
	TailscaleEnabled bool `toml:"tailscale_enabled"`

	// K8sEnabled toggles the Kubernetes status collector.
	K8sEnabled bool `toml:"k8s_enabled"`

	// ClaudeEnabled toggles the Claude usage collector.
	ClaudeEnabled bool `toml:"claude_enabled"`

	// BillingEnabled toggles the billing data collector.
	BillingEnabled bool `toml:"billing_enabled"`

	// StarshipEnabled toggles Starship prompt integration.
	StarshipEnabled bool `toml:"starship_enabled"`

	// StarshipFormat is the custom Starship format string.
	StarshipFormat string `toml:"starship_format"`

	// Theme is the name of the visual theme.
	Theme string `toml:"theme"`
}

// duration is a v1-compatible duration type that supports Go duration strings.
type duration struct {
	time.Duration
}

func (d *duration) UnmarshalText(text []byte) error {
	s := string(text)
	if s == "" {
		d.Duration = 0
		return nil
	}
	parsed, err := time.ParseDuration(s)
	if err != nil {
		return fmt.Errorf("invalid duration %q: %w", s, err)
	}
	d.Duration = parsed
	return nil
}

// mgParseV1 reads and parses a v1 configuration file, applying defaults for missing fields.
func mgParseV1(path string) (*V1Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading v1 config: %w", err)
	}

	cfg := mgV1Defaults()

	if _, err := toml.Decode(string(data), cfg); err != nil {
		return nil, fmt.Errorf("parsing v1 config: %w", err)
	}

	return cfg, nil
}

// mgV1Defaults returns a V1Config populated with sensible defaults.
func mgV1Defaults() *V1Config {
	return &V1Config{
		WaifuEnabled:     true,
		BannerShowWaifu:  true,
		TailscaleEnabled: true,
		ClaudeEnabled:    true,
		CacheTTL:         duration{5 * time.Minute},
		BannerWidth:      120,
		Theme:            "default",
	}
}
