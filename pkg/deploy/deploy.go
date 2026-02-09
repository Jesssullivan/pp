package deploy

import (
	"fmt"
	"time"
)

// HostProfile describes a target host for deployment verification.
type HostProfile struct {
	// Name is the hostname (e.g., "xoxd-bates", "honey").
	Name string

	// OS is the operating system ("darwin", "linux").
	OS string

	// Arch is the architecture ("aarch64", "x86_64").
	Arch string

	// Features lists the prompt-pulse features expected on this host.
	Features []string

	// Shells lists shells that should have integration scripts.
	Shells []string

	// ExpectedCollectors lists collectors that should be active.
	ExpectedCollectors []string

	// BinaryPath overrides the default binary location for testing.
	BinaryPath string

	// ConfigPath overrides the default config file location for testing.
	ConfigPath string

	// CacheDir overrides the default cache directory for testing.
	CacheDir string

	// SocketPath overrides the default daemon socket location for testing.
	SocketPath string

	// PIDFile overrides the default PID file location for testing.
	PIDFile string
}

// VerifyResult holds the outcome of verifying a host profile.
type VerifyResult struct {
	// Host is the hostname that was verified.
	Host string

	// Passed is true when all required checks passed.
	Passed bool

	// Checks lists individual check results.
	Checks []CheckResult

	// Timestamp records when verification completed.
	Timestamp time.Time
}

// CheckResult records the outcome of a single verification check.
type CheckResult struct {
	// Name identifies the check.
	Name string

	// Passed is true when the check succeeded.
	Passed bool

	// Message describes the result.
	Message string

	// Duration records how long the check took.
	Duration time.Duration
}

// NewHostProfile creates a HostProfile with the given name, OS, and architecture.
func NewHostProfile(name, os, arch string) *HostProfile {
	return &HostProfile{
		Name: name,
		OS:   os,
		Arch: arch,
	}
}

// XoxdBates returns the host profile for the primary development machine.
func XoxdBates() *HostProfile {
	return &HostProfile{
		Name:               "xoxd-bates",
		OS:                 "darwin",
		Arch:               "aarch64",
		Features:           []string{"waifu", "tailscale", "claude", "billing", "sysmetrics"},
		Shells:             []string{"bash", "zsh", "fish"},
		ExpectedCollectors: []string{"waifu", "tailscale", "claude", "billing", "sysmetrics"},
	}
}

// Honey returns the host profile for the primary headless streaming server.
func Honey() *HostProfile {
	return &HostProfile{
		Name:               "honey",
		OS:                 "linux",
		Arch:               "x86_64",
		Features:           []string{"tailscale", "k8s", "claude", "sysmetrics", "gpu"},
		Shells:             []string{"bash", "zsh"},
		ExpectedCollectors: []string{"tailscale", "k8s", "claude", "sysmetrics", "gpu"},
	}
}

// PettingZooMini returns the host profile for the secondary streaming host.
func PettingZooMini() *HostProfile {
	return &HostProfile{
		Name:               "petting-zoo-mini",
		OS:                 "darwin",
		Arch:               "aarch64",
		Features:           []string{"waifu", "tailscale", "sysmetrics", "ghostty"},
		Shells:             []string{"bash", "zsh"},
		ExpectedCollectors: []string{"waifu", "tailscale", "sysmetrics", "ghostty"},
	}
}

// Verify runs all applicable checks for the given host profile and returns
// the aggregated result. It does NOT perform deployment; it only validates
// that an existing deployment is correct.
func Verify(profile *HostProfile) (*VerifyResult, error) {
	if profile == nil {
		return nil, fmt.Errorf("deploy: nil host profile")
	}

	checks := dpBuildChecks(profile)
	results := make([]CheckResult, 0, len(checks))
	allPassed := true

	for _, c := range checks {
		start := time.Now()
		passed, msg := c.Run()
		dur := time.Since(start)

		cr := CheckResult{
			Name:     c.Name,
			Passed:   passed,
			Message:  msg,
			Duration: dur,
		}
		results = append(results, cr)

		if !passed && c.Required {
			allPassed = false
		}
	}

	return &VerifyResult{
		Host:      profile.Name,
		Passed:    allPassed,
		Checks:    results,
		Timestamp: time.Now(),
	}, nil
}

// dpBuildChecks assembles the list of checks for a profile.
func dpBuildChecks(profile *HostProfile) []Check {
	checks := []Check{
		dpCheckBinary(profile),
		dpCheckConfig(profile),
		dpCheckDaemon(profile),
		dpCheckCache(profile),
		dpCheckTheme(profile),
		dpCheckTerminal(),
		dpCheckImage(),
		dpCheckPermissions(profile),
	}

	for _, sh := range profile.Shells {
		checks = append(checks, dpCheckShell(profile, sh))
	}

	for _, col := range profile.ExpectedCollectors {
		checks = append(checks, dpCheckCollector(profile, col))
	}

	return checks
}
