// prompt-pulse is a multi-source infrastructure status aggregator.
//
// It collects status from Claude Code sessions, cloud billing APIs, and
// infrastructure health checks, then surfaces that information through
// Starship prompt segments or an interactive TUI.
//
// Usage:
//
//	prompt-pulse [flags]
//
// Flags:
//
//	-banner           Display system status banner
//	-daemon           Run background polling daemon
//	-tui              Launch interactive Bubbletea TUI
//	-starship string  Output one-line Starship format (claude|billing|infra)
//	-shell string     Output shell integration script (bash|zsh|fish|nushell)
//	-config string    Path to configuration file (default: ~/.config/prompt-pulse/config.yaml)
//	-use-mocks        Use mock data instead of real API calls (for testing)
//	-mock-accounts int Number of mock Claude accounts to generate (default: 3)
//	-verbose          Enable verbose logging
//	-version          Print version and exit
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"gitlab.com/tinyland/lab/prompt-pulse/cache"
	"gitlab.com/tinyland/lab/prompt-pulse/collectors"
	"gitlab.com/tinyland/lab/prompt-pulse/config"
	"gitlab.com/tinyland/lab/prompt-pulse/display/banner"
	"gitlab.com/tinyland/lab/prompt-pulse/display/starship"
	"gitlab.com/tinyland/lab/prompt-pulse/display/tui"
	"gitlab.com/tinyland/lab/prompt-pulse/shell"
	"gitlab.com/tinyland/lab/prompt-pulse/tests/mocks"
)

var (
	version = "0.1.0"
	commit  = "dev"
	date    = "unknown"
)

// defaultPollInterval is the fallback duration when the configured poll
// interval cannot be parsed.
const defaultPollInterval = 15 * time.Minute

func main() {
	// Parse command line flags
	var (
		configPath      = flag.String("config", "", "Path to configuration file")
		runDaemon       = flag.Bool("daemon", false, "Run background polling daemon")
		runTUI          = flag.Bool("tui", false, "Launch interactive Bubbletea TUI")
		runBanner       = flag.Bool("banner", false, "Display system status banner")
		showWaifu       = flag.Bool("waifu", false, "Show waifu image in banner (requires -banner)")
		showFastfetch   = flag.Bool("fastfetch-enabled", false, "Show fastfetch system info in banner center column")
		sessionID       = flag.String("session-id", "", "Session ID for waifu caching (auto-generated if empty)")
		termWidth       = flag.Int("term-width", 0, "Terminal width override (0 = auto-detect)")
		termHeight      = flag.Int("term-height", 0, "Terminal height override (0 = auto-detect)")
		starshipMod     = flag.String("starship", "", "Output one-line Starship format (claude|billing|infra)")
		shellIntegration = flag.String("shell", "", "Output shell integration script (bash|zsh|fish|nushell)")
		runDiagnose      = flag.Bool("diagnose", false, "Diagnose Claude credentials and API connectivity")
		runBillingCheck  = flag.Bool("billing-check", false, "Check billing provider API key configuration")
		useMocks         = flag.Bool("use-mocks", false, "Use mock data instead of real API calls (for testing)")
		mockAccounts     = flag.Int("mock-accounts", 3, "Number of mock Claude accounts to generate (with -use-mocks)")
		mockSeed         = flag.Int64("mock-seed", 0, "Random seed for mock data (0 = random)")
		verbose          = flag.Bool("verbose", false, "Enable verbose logging")
		showVersion      = flag.Bool("version", false, "Print version and exit")
	)
	flag.Parse()

	if *showVersion {
		fmt.Printf("prompt-pulse %s (%s) built %s\n", version, commit, date)
		os.Exit(0)
	}

	// Handle diagnostic commands (don't require config)
	if *runDiagnose {
		runClaudeDiagnostics()
		os.Exit(0)
	}

	if *runBillingCheck {
		runBillingProviderCheck()
		os.Exit(0)
	}

	// Handle shell integration output (doesn't require config)
	if *shellIntegration != "" {
		cfg := shell.DefaultIntegrationConfig()
		var shellType shell.ShellType
		switch *shellIntegration {
		case "bash":
			shellType = shell.Bash
		case "zsh":
			shellType = shell.Zsh
		case "fish":
			shellType = shell.Fish
		case "nushell", "nu":
			shellType = shell.Nushell
		default:
			fmt.Fprintf(os.Stderr, "unknown shell: %s (supported: bash, zsh, fish, nushell)\n", *shellIntegration)
			os.Exit(1)
		}
		fmt.Print(shell.GenerateIntegration(shellType, cfg))
		os.Exit(0)
	}

	// Resolve configuration path
	if *configPath == "" {
		home, _ := os.UserHomeDir()
		*configPath = filepath.Join(home, ".config", "prompt-pulse", "config.yaml")
	}

	// Load configuration
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	if err := cfg.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "invalid config: %v\n", err)
		os.Exit(1)
	}

	// Setup log file directory
	if err := ensureLogDir(cfg.Daemon.LogFile); err != nil {
		fmt.Fprintf(os.Stderr, "failed to create log directory: %v\n", err)
		os.Exit(1)
	}

	// Setup logging - write to both stderr and log file
	logLevel := slog.LevelInfo
	if *verbose {
		logLevel = slog.LevelDebug
	}

	// Open log file for writing
	logFile, err := os.OpenFile(cfg.Daemon.LogFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to open log file: %v\n", err)
		os.Exit(1)
	}
	defer logFile.Close()

	// Create multi-writer for both stderr and log file
	multiWriter := io.MultiWriter(os.Stderr, logFile)
	logger := slog.New(slog.NewTextHandler(multiWriter, &slog.HandlerOptions{
		Level: logLevel,
	}))

	// Setup context with signal handling
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		logger.Info("received shutdown signal")
		cancel()
	}()

	// Determine operation mode
	switch {
	case *runTUI:
		model := tui.NewModel()

		if *useMocks {
			// Use mock data for testing
			if *mockSeed != 0 {
				mocks.SeedRandom(*mockSeed)
			}
			logger.Info("using mock data", "accounts", *mockAccounts, "seed", *mockSeed)
			model.SetClaudeData(mocks.MockClaudeUsage(*mockAccounts))
			model.SetBillingData(mocks.MockBillingData())
			model.SetInfraData(mocks.MockInfraStatus())
		} else {
			// Load cached data to populate the model before launch.
			store, storeErr := cache.NewStore(cfg.Daemon.CacheDir, logger)
			if storeErr == nil {
				ttl := parseDuration(cfg.Daemon.PollInterval)
				if claude, _, _ := cache.GetTyped[collectors.ClaudeUsage](store, "claude", ttl); claude != nil {
					model.SetClaudeData(claude)
				}
				if billing, _, _ := cache.GetTyped[collectors.BillingData](store, "billing", ttl); billing != nil {
					model.SetBillingData(billing)
				}
				if infra, _, _ := cache.GetTyped[collectors.InfraStatus](store, "infra", ttl); infra != nil {
					model.SetInfraData(infra)
				}
			} else {
				logger.Warn("failed to open cache for TUI", "error", storeErr)
			}
		}

		p := tea.NewProgram(model, tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			logger.Error("TUI error", "error", err)
			os.Exit(1)
		}

	case *starshipMod != "":
		out, err := starship.NewOutput(starship.OutputConfig{
			CacheDir: cfg.Daemon.CacheDir,
			CacheTTL: parseDuration(cfg.Daemon.PollInterval),
			Logger:   logger,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "starship init failed: %v\n", err)
			os.Exit(1)
		}
		result, err := out.Module(*starshipMod)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}
		if result != "" {
			fmt.Print(result)
		}

	case *runBanner:
		// --waifu and --fastfetch-enabled flags override config file settings
		waifuEnabled := cfg.Display.Waifu.Enabled || *showWaifu
		fastfetchEnabled := cfg.Display.Fastfetch.Enabled || *showFastfetch

		// Terminal dimensions: Use CLI flags if provided (non-zero), otherwise auto-detect
		width, height := banner.DetectTerminalSize()
		if *termWidth > 0 {
			width = *termWidth
		}
		if *termHeight > 0 {
			height = *termHeight
		}
		// Use default max sessions if not configured
		maxSessions := cfg.Display.Waifu.MaxSessions
		if maxSessions <= 0 {
			maxSessions = 10
		}
		bannerCfg := banner.BannerConfig{
			CacheDir:         cfg.Daemon.CacheDir,
			CacheTTL:         parseDuration(cfg.Daemon.PollInterval),
			WaifuEnabled:     waifuEnabled,
			WaifuCategory:    cfg.Display.Waifu.Category,
			WaifuCacheDir:    filepath.Join(cfg.Daemon.CacheDir, "waifu"),
			WaifuCacheTTL:    parseDuration(cfg.Display.Waifu.CacheTTL),
			WaifuMaxCacheMB:  cfg.Display.Waifu.MaxCacheMB,
			WaifuSessionID:   *sessionID,
			WaifuMaxSessions: maxSessions,
			FastfetchEnabled: fastfetchEnabled,
			TermWidth:        width,
			TermHeight:       height,
			Logger:           logger,
		}

		var output string
		if *useMocks {
			// Use mock data for testing
			if *mockSeed != 0 {
				mocks.SeedRandom(*mockSeed)
			}
			logger.Info("using mock data for banner", "accounts", *mockAccounts, "seed", *mockSeed)
			output, err = generateMockBanner(ctx, bannerCfg, *mockAccounts)
		} else {
			b := banner.NewBanner(bannerCfg)
			output, err = b.Generate(ctx)
		}
		if err != nil {
			logger.Error("banner generation failed", "error", err)
			os.Exit(1)
		}
		fmt.Print(output)

	case *runDaemon:
		d, err := newDaemon(cfg, logger)
		if err != nil {
			logger.Error("daemon init failed", "error", err)
			os.Exit(1)
		}
		logger.Info("starting prompt-pulse daemon",
			"poll_interval", cfg.Daemon.PollInterval,
			"config", *configPath,
		)
		if err := d.run(ctx); err != nil && err != context.Canceled {
			logger.Error("daemon error", "error", err)
			os.Exit(1)
		}

	default:
		// Default: run a single collection pass
		d, err := newDaemon(cfg, logger)
		if err != nil {
			logger.Error("daemon init failed", "error", err)
			os.Exit(1)
		}
		if err := d.runOnce(ctx); err != nil {
			logger.Error("collection failed", "error", err)
			os.Exit(1)
		}
	}
}

// parseDuration parses a Go duration string (e.g. "15m", "1h", "30s").
// Returns defaultPollInterval if the string is empty or unparseable.
func parseDuration(s string) time.Duration {
	if s == "" {
		return defaultPollInterval
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return defaultPollInterval
	}
	return d
}

func ensureLogDir(logFile string) error {
	dir := filepath.Dir(logFile)
	return os.MkdirAll(dir, 0755)
}

// generateMockBanner generates a banner using mock data instead of cached data.
// This is useful for testing display layouts with various data configurations.
func generateMockBanner(ctx context.Context, cfg banner.BannerConfig, accountCount int) (string, error) {
	claude := mocks.MockClaudeUsage(accountCount)
	billing := mocks.MockBillingData()
	infra := mocks.MockInfraStatus()

	// Create a mock banner with injected data
	b := banner.NewMockBanner(cfg, claude, billing, infra)
	return b.Generate(ctx)
}

// runClaudeDiagnostics checks Claude credentials and API connectivity.
func runClaudeDiagnostics() {
	fmt.Println("🔍 Claude Code Diagnostics")
	fmt.Println("============================================================")
	fmt.Println()

	// Check credential file existence
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Printf("❌ Failed to get home directory: %v\n", err)
		return
	}

	credPath := filepath.Join(homeDir, ".claude", ".credentials.json")
	fmt.Printf("📁 Credential file: %s\n", credPath)

	if _, err := os.Stat(credPath); os.IsNotExist(err) {
		fmt.Println("   ❌ File not found")
		fmt.Println()
		fmt.Println("💡 Solution: Run 'claude login' to authenticate")
		return
	}
	fmt.Println("   ✅ File exists")

	// Read and parse credentials
	data, err := os.ReadFile(credPath)
	if err != nil {
		fmt.Printf("   ⚠️  Cannot read file: %v\n", err)
		return
	}

	// Simple JSON parsing for OAuth credentials
	type oauthCreds struct {
		ClaudeAIOAuth struct {
			AccessToken  string `json:"accessToken"`
			RefreshToken string `json:"refreshToken"`
			ExpiresAt    int64  `json:"expiresAt"` // Unix timestamp in milliseconds
		} `json:"claudeAiOauth"`
	}

	var creds oauthCreds
	if err := json.Unmarshal(data, &creds); err != nil {
		fmt.Printf("   ⚠️  Cannot parse JSON: %v\n", err)
		return
	}

	fmt.Println()

	// Check OAuth credentials
	if creds.ClaudeAIOAuth.AccessToken == "" {
		fmt.Println("🔑 OAuth Credentials: ❌ Not found")
		fmt.Println()
		fmt.Println("💡 Solution: Run 'claude login' to authenticate")
		return
	}

	fmt.Println("🔑 OAuth Credentials")
	fmt.Println("------------------------------------------------------------")
	fmt.Printf("   Access Token:  ✅ Present (%d chars)\n", len(creds.ClaudeAIOAuth.AccessToken))

	if creds.ClaudeAIOAuth.RefreshToken == "" {
		fmt.Println("   Refresh Token: ❌ Empty")
	} else {
		fmt.Printf("   Refresh Token: ✅ Present (%d chars)\n", len(creds.ClaudeAIOAuth.RefreshToken))
	}

	// Check token expiration
	if creds.ClaudeAIOAuth.ExpiresAt == 0 {
		fmt.Println("   Expiration:    ⚠️  Not set")
	} else {
		expiresAt := time.UnixMilli(creds.ClaudeAIOAuth.ExpiresAt)
		now := time.Now()
		timeUntil := expiresAt.Sub(now)

		if timeUntil < 0 {
			fmt.Printf("   Expiration:    ❌ EXPIRED (%s ago)\n", formatDiagDuration(-timeUntil))
			fmt.Println()
			fmt.Println("💡 Solution: Run 'claude login' to refresh your token")
		} else if timeUntil < 1*time.Hour {
			fmt.Printf("   Expiration:    ⚠️  Soon (%s remaining)\n", formatDiagDuration(timeUntil))
			fmt.Println()
			fmt.Println("💡 Tip: Token expires soon, consider refreshing")
		} else {
			fmt.Printf("   Expiration:    ✅ Valid (%s remaining)\n", formatDiagDuration(timeUntil))
		}
	}

	fmt.Println()

	// Check API key environment variable
	fmt.Println("🌐 API Configuration")
	fmt.Println("------------------------------------------------------------")

	if apiKey := os.Getenv("ANTHROPIC_API_KEY"); apiKey != "" {
		fmt.Printf("   ANTHROPIC_API_KEY: ✅ Set (%d chars)\n", len(apiKey))
	} else {
		fmt.Println("   ANTHROPIC_API_KEY: ⚠️  Not set (OK for OAuth)")
	}

	if baseUrl := os.Getenv("ANTHROPIC_BASE_URL"); baseUrl != "" {
		fmt.Printf("   ANTHROPIC_BASE_URL: %s\n", baseUrl)
	}

	fmt.Println()
	fmt.Println("✨ Diagnostics complete!")
}

// runBillingProviderCheck validates billing provider API keys.
func runBillingProviderCheck() {
	fmt.Println("💰 Billing Provider Configuration Check")
	fmt.Println("======================================================================")
	fmt.Println()

	type provider struct {
		Name        string
		EnvVar      string
		FileVar     string
		Description string
	}

	providers := []provider{
		{"Civo", "CIVO_API_KEY", "CIVO_API_KEY_FILE", "Kubernetes cloud provider"},
		{"DigitalOcean", "DIGITALOCEAN_TOKEN", "DIGITALOCEAN_TOKEN_FILE", "Cloud infrastructure"},
		{"DreamHost", "DREAMHOST_API_KEY", "DREAMHOST_API_KEY_FILE", "Web hosting"},
		{"AWS", "AWS_PROFILE", "", "Amazon Web Services (uses AWS CLI credentials)"},
	}

	var configured, missing int

	for _, p := range providers {
		fmt.Printf("📦 %s (%s)\n", p.Name, p.Description)
		fmt.Println("----------------------------------------------------------------------")

		// Check direct environment variable
		apiKey := os.Getenv(p.EnvVar)
		if apiKey != "" {
			fmt.Printf("   %s: ✅ Set (%d chars)\n", p.EnvVar, len(apiKey))
			configured++
			fmt.Println()
			continue
		}

		// Check file-based variant (sops-nix pattern)
		if p.FileVar != "" {
			filePath := os.Getenv(p.FileVar)
			if filePath != "" {
				if data, err := os.ReadFile(filePath); err == nil && len(data) > 0 {
					fmt.Printf("   %s: ✅ Set (via %s)\n", p.EnvVar, p.FileVar)
					fmt.Printf("   File: %s (%d bytes)\n", filePath, len(data))
					configured++
					fmt.Println()
					continue
				}
			}
		}

		// Special handling for AWS
		if p.Name == "AWS" {
			homeDir, _ := os.UserHomeDir()
			awsCredsFile := filepath.Join(homeDir, ".aws", "credentials")
			if _, err := os.Stat(awsCredsFile); err == nil {
				fmt.Printf("   %s: ✅ AWS credentials file exists\n", p.EnvVar)
				fmt.Printf("   File: %s\n", awsCredsFile)
				configured++
				fmt.Println()
				continue
			}
		}

		// Not configured
		fmt.Printf("   %s: ❌ Not set\n", p.EnvVar)
		if p.FileVar != "" {
			fmt.Printf("   %s: ❌ Not set\n", p.FileVar)
		}
		missing++
		fmt.Println()
		fmt.Printf("   💡 To configure: export %s='your-api-key'\n", p.EnvVar)
		if p.FileVar != "" {
			fmt.Printf("   💡 Or (sops-nix): export %s='/path/to/secret/file'\n", p.FileVar)
		}
		fmt.Println()
	}

	// Summary
	fmt.Println("======================================================================")
	fmt.Printf("Summary: %d/%d providers configured\n", configured, len(providers))
	fmt.Println()

	if missing > 0 {
		fmt.Println("⚠️  Some providers are missing API keys")
		fmt.Println()
		fmt.Println("Why this matters:")
		fmt.Println("  • Providers without API keys will show Status=\"error\"")
		fmt.Println("  • Failed providers are excluded from billing totals")
		fmt.Println("  • If ALL providers fail, banner shows \"$0 this month\"")
		fmt.Println()
		fmt.Println("To fix:")
		fmt.Println("  1. Set environment variables for providers you use")
		fmt.Println("  2. Restart prompt-pulse daemon: systemctl --user restart prompt-pulse")
		fmt.Println("  3. Check banner: prompt-pulse --banner")
	} else {
		fmt.Println("✅ All billing providers are configured!")
		fmt.Println()
		fmt.Println("Next steps:")
		fmt.Println("  • Check billing data: prompt-pulse --banner")
		fmt.Println("  • View details in TUI: prompt-pulse")
		fmt.Println("  • Monitor daemon logs: journalctl --user -u prompt-pulse -f")
	}
}

// formatDiagDuration formats a duration for diagnostic output.
func formatDiagDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.0fs", d.Seconds())
	}
	if d < time.Hour {
		return fmt.Sprintf("%.0fm", d.Minutes())
	}
	if d < 24*time.Hour {
		hours := int(d.Hours())
		minutes := int(d.Minutes()) % 60
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	return fmt.Sprintf("%dd %dh", days, hours)
}
