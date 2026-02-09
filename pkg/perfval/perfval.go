// Package perfval provides performance validation, memory profiling, soak
// testing, benchmark comparison, and report generation for the prompt-pulse
// dashboard. It validates that critical rendering paths meet their performance
// budgets and detects regressions before they reach production.
//
// All unexported helpers are prefixed with "pv" to avoid naming conflicts
// with other packages.
package perfval

import (
	"sort"
	"time"
)

// Target defines a performance budget for a named operation. Each target
// specifies the maximum acceptable duration (p95) for the operation.
type Target struct {
	// Name identifies the operation being measured.
	Name string

	// MaxDuration is the maximum acceptable p95 latency for this operation.
	MaxDuration time.Duration

	// Description explains what the target measures and why the budget exists.
	Description string
}

// ValidationResult records the outcome of validating a single performance
// target against measured samples.
type ValidationResult struct {
	// Target is the name of the validated operation.
	Target string

	// Actual is the measured p95 duration across all samples.
	Actual time.Duration

	// Passed indicates whether the measured duration is within budget.
	Passed bool

	// Margin is the percentage of budget remaining (positive) or exceeded
	// (negative). For example, 0.25 means 25% headroom; -0.10 means 10%
	// over budget.
	Margin float64

	// Samples is the number of iterations that were measured.
	Samples int
}

// ValidationReport aggregates the results of validating all performance
// targets in a single run.
type ValidationReport struct {
	// Results contains one entry per validated target.
	Results []ValidationResult

	// AllPassed is true only if every individual target passed.
	AllPassed bool

	// Timestamp records when the validation was performed.
	Timestamp time.Time

	// Platform identifies the OS and architecture where validation ran.
	Platform string
}

// DefaultTargets returns the five canonical performance targets for the
// prompt-pulse dashboard. These represent hard budgets that must be met
// on every commit.
func DefaultTargets() []Target {
	return []Target{
		{
			Name:        "banner_cached",
			MaxDuration: 50 * time.Millisecond,
			Description: "Banner rendering from cache must complete in under 50ms",
		},
		{
			Name:        "tui_frame",
			MaxDuration: 16 * time.Millisecond,
			Description: "TUI frame render must complete in under 16ms for 60fps",
		},
		{
			Name:        "image_kitty",
			MaxDuration: 5 * time.Millisecond,
			Description: "Image rendering via Kitty protocol must complete in under 5ms",
		},
		{
			Name:        "shell_source",
			MaxDuration: 5 * time.Millisecond,
			Description: "Shell integration sourcing must complete in under 5ms",
		},
		{
			Name:        "cache_read",
			MaxDuration: 10 * time.Millisecond,
			Description: "Cache read operations must complete in under 10ms",
		},
	}
}

// ValidateTarget runs fn for the given number of samples, computes the p95
// latency, and checks it against the target's budget. Returns a
// ValidationResult describing the outcome.
func ValidateTarget(target Target, fn func() error, samples int) *ValidationResult {
	if samples <= 0 {
		return &ValidationResult{
			Target:  target.Name,
			Actual:  0,
			Passed:  true,
			Margin:  1.0,
			Samples: 0,
		}
	}

	durations := make([]time.Duration, 0, samples)
	for i := 0; i < samples; i++ {
		start := time.Now()
		_ = fn()
		durations = append(durations, time.Since(start))
	}

	p95 := pvP95(durations)

	maxNs := float64(target.MaxDuration)
	actualNs := float64(p95)
	margin := 0.0
	if maxNs > 0 {
		margin = (maxNs - actualNs) / maxNs
	}

	return &ValidationResult{
		Target:  target.Name,
		Actual:  p95,
		Passed:  p95 <= target.MaxDuration,
		Margin:  margin,
		Samples: samples,
	}
}

// ValidateAll runs validation for every target that has a corresponding
// benchmark function in benchFns (matched by Target.Name). Targets without
// a matching function are skipped. Returns a complete ValidationReport.
func ValidateAll(targets []Target, benchFns map[string]func() error, samples int) *ValidationReport {
	report := &ValidationReport{
		Timestamp: time.Now(),
		AllPassed: true,
	}

	for _, t := range targets {
		fn, ok := benchFns[t.Name]
		if !ok {
			continue
		}
		result := ValidateTarget(t, fn, samples)
		report.Results = append(report.Results, *result)
		if !result.Passed {
			report.AllPassed = false
		}
	}

	return report
}

// pvP95 computes the 95th percentile duration from a slice of durations.
// Returns 0 if the slice is empty.
func pvP95(durations []time.Duration) time.Duration {
	if len(durations) == 0 {
		return 0
	}

	sorted := make([]time.Duration, len(durations))
	copy(sorted, durations)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })

	idx := int(float64(len(sorted)-1) * 0.95)
	return sorted[idx]
}
