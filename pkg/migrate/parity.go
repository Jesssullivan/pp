package migrate

import (
	"fmt"
)

// ParityReport describes how well v2 covers the features present in a v1 configuration.
type ParityReport struct {
	// Covered lists v1 features that are fully supported in v2.
	Covered []string

	// Missing lists v1 features that have no v2 equivalent.
	Missing []string

	// NewInV2 lists features available in v2 that did not exist in v1.
	NewInV2 []string

	// Score is the parity percentage (0.0 to 1.0).
	Score float64
}

// featureCheck pairs a v1 feature name with a predicate that returns true if the feature is active.
type featureCheck struct {
	name    string
	active  func(v1 *V1Config) bool
	covered bool // whether v2 has a direct equivalent
}

// CheckParity compares a v1 configuration against v2 capabilities,
// reporting which features are covered, missing, or new.
func CheckParity(v1Path string) (*ParityReport, error) {
	v1, err := mgParseV1(v1Path)
	if err != nil {
		return nil, fmt.Errorf("parsing v1 config for parity check: %w", err)
	}

	return mgCheckParityFromConfig(v1), nil
}

// mgCheckParityFromConfig performs the parity check against a parsed V1Config.
func mgCheckParityFromConfig(v1 *V1Config) *ParityReport {
	checks := mgFeatureChecks()

	report := &ParityReport{}

	activeCount := 0
	coveredCount := 0

	for _, check := range checks {
		if check.active(v1) {
			activeCount++
			if check.covered {
				coveredCount++
				report.Covered = append(report.Covered, check.name)
			} else {
				report.Missing = append(report.Missing, check.name)
			}
		}
	}

	// New in v2 features (not present in v1)
	report.NewInV2 = mgNewV2Features()

	if activeCount > 0 {
		report.Score = float64(coveredCount) / float64(activeCount)
	} else {
		report.Score = 1.0
	}

	return report
}

// mgFeatureChecks returns the list of v1 features to check for parity.
func mgFeatureChecks() []featureCheck {
	return []featureCheck{
		// Banner sections
		{
			name:    "banner:waifu",
			active:  func(v1 *V1Config) bool { return v1.WaifuEnabled || v1.BannerShowWaifu },
			covered: true, // v2: image.waifu_enabled
		},
		{
			name:    "banner:system-metrics",
			active:  func(_ *V1Config) bool { return true }, // always present in v1
			covered: true, // v2: collectors.sysmetrics
		},
		{
			name:    "banner:tailscale",
			active:  func(v1 *V1Config) bool { return v1.TailscaleEnabled },
			covered: true, // v2: collectors.tailscale
		},
		{
			name:    "banner:k8s",
			active:  func(v1 *V1Config) bool { return v1.K8sEnabled },
			covered: true, // v2: collectors.kubernetes
		},
		{
			name:    "banner:claude",
			active:  func(v1 *V1Config) bool { return v1.ClaudeEnabled },
			covered: true, // v2: collectors.claude
		},
		{
			name:    "banner:billing",
			active:  func(v1 *V1Config) bool { return v1.BillingEnabled },
			covered: true, // v2: collectors.billing
		},
		// Shell integration
		{
			name:    "shell:bash",
			active:  func(_ *V1Config) bool { return true }, // v1 always supports bash
			covered: true,
		},
		{
			name:    "shell:zsh",
			active:  func(_ *V1Config) bool { return true }, // v1 always supports zsh
			covered: true,
		},
		{
			name:    "shell:fish",
			active:  func(_ *V1Config) bool { return true }, // v1 always supports fish
			covered: true,
		},
		// Starship
		{
			name:    "starship:module",
			active:  func(v1 *V1Config) bool { return v1.StarshipEnabled },
			covered: false, // v2: starship is now external config
		},
		{
			name:    "starship:custom-format",
			active:  func(v1 *V1Config) bool { return v1.StarshipFormat != "" },
			covered: false, // v2: starship format is external
		},
		// Daemon mode
		{
			name:    "daemon:socket",
			active:  func(v1 *V1Config) bool { return v1.DaemonSocket != "" },
			covered: true, // v2: managed automatically
		},
		{
			name:    "daemon:pid-file",
			active:  func(v1 *V1Config) bool { return v1.DaemonPidFile != "" },
			covered: true, // v2: managed automatically
		},
		// Cache system
		{
			name:    "cache:custom-dir",
			active:  func(v1 *V1Config) bool { return v1.CacheDir != "" },
			covered: true, // v2: general.cache_dir
		},
		{
			name:    "cache:ttl",
			active:  func(v1 *V1Config) bool { return v1.CacheTTL.Duration > 0 },
			covered: true, // v2: general.data_retention
		},
		// Image rendering (always active in v1 when waifu is enabled)
		{
			name:    "image:rendering",
			active:  func(v1 *V1Config) bool { return v1.WaifuEnabled },
			covered: true, // v2: image.protocol (auto, kitty, iterm2, sixel, halfblocks)
		},
		// Waifu local path (v1 supports local files, v2 is API-only)
		{
			name:    "waifu:local-path",
			active:  func(v1 *V1Config) bool { return v1.WaifuPath != "" },
			covered: false, // v2: API-based waifu only
		},
	}
}

// mgNewV2Features returns features that are new in v2 and did not exist in v1.
func mgNewV2Features() []string {
	return []string{
		"layout:preset-system",
		"layout:custom-rows",
		"shell:tui-keybinding",
		"shell:instant-banner",
		"shell:banner-timeout",
		"shell:ksh-support",
		"image:waifu-category",
		"image:session-cache",
		"image:max-cache-size",
		"collectors:sysmetrics-interval",
		"collectors:per-collector-intervals",
		"collectors:claude-multi-account",
		"billing:civo-integration",
		"billing:digitalocean-integration",
		"banner:responsive-thresholds",
		"theme:multiple-themes",
	}
}
