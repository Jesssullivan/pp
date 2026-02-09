package perfval

import "fmt"

// ThresholdConfig defines configurable limits for performance validation.
// These thresholds determine when a CI gate should fail.
type ThresholdConfig struct {
	// RegressionThreshold is the maximum acceptable performance regression
	// as a fraction (e.g., 0.15 = 15% slower is OK).
	RegressionThreshold float64

	// LeakGrowthMax is the maximum acceptable heap growth as a fraction
	// of the baseline (e.g., 0.20 = 20% growth allowed).
	LeakGrowthMax float64

	// SoakErrorRateMax is the maximum acceptable error rate during soak
	// tests as a fraction (e.g., 0.01 = 1% errors allowed).
	SoakErrorRateMax float64

	// P99Multiplier is the maximum acceptable ratio of P99 to P50 latency.
	// A value of 2.0 means P99 can be at most 2x P50.
	P99Multiplier float64
}

// DefaultThresholds returns the standard thresholds used for CI gate
// decisions. These are conservative defaults that balance catching real
// regressions against avoiding false positives in noisy CI environments.
func DefaultThresholds() *ThresholdConfig {
	return &ThresholdConfig{
		RegressionThreshold: 0.15,
		LeakGrowthMax:       0.20,
		SoakErrorRateMax:    0.01,
		P99Multiplier:       2.0,
	}
}

// pvApplyThresholds evaluates a performance report against the given
// thresholds and returns a list of human-readable violation messages.
// An empty list means all thresholds were met.
func pvApplyThresholds(report *PerfReport, thresholds *ThresholdConfig) []string {
	if report == nil || thresholds == nil {
		return nil
	}

	var violations []string

	// Check target validation results.
	if report.Targets != nil {
		for _, r := range report.Targets.Results {
			if !r.Passed {
				violations = append(violations, fmt.Sprintf(
					"target %q exceeded budget: actual %v, margin %.1f%%",
					r.Target, r.Actual, r.Margin*100,
				))
			}
		}
	}

	// Check regression threshold.
	for _, r := range report.Regressions {
		if r.PercentChange > thresholds.RegressionThreshold*100 {
			violations = append(violations, fmt.Sprintf(
				"regression in %q: %.1f%% slower (threshold: %.0f%%)",
				r.Name, r.PercentChange, thresholds.RegressionThreshold*100,
			))
		}
	}

	// Check memory leak threshold.
	if report.Memory != nil && len(report.Memory.Snapshots) >= 2 {
		baseline := report.Memory.Snapshots[0].HeapAlloc
		current := report.Memory.Snapshots[len(report.Memory.Snapshots)-1].HeapAlloc
		if baseline > 0 && current > baseline {
			growth := float64(current-baseline) / float64(baseline)
			if growth > thresholds.LeakGrowthMax {
				violations = append(violations, fmt.Sprintf(
					"memory leak: heap grew %.1f%% (threshold: %.0f%%)",
					growth*100, thresholds.LeakGrowthMax*100,
				))
			}
		}
	}

	// Check soak test thresholds.
	if report.Soak != nil && report.Soak.Iterations > 0 {
		errorRate := float64(report.Soak.Errors) / float64(report.Soak.Iterations)
		if errorRate > thresholds.SoakErrorRateMax {
			violations = append(violations, fmt.Sprintf(
				"soak error rate %.2f%% exceeds threshold %.1f%%",
				errorRate*100, thresholds.SoakErrorRateMax*100,
			))
		}

		if report.Soak.P50 > 0 {
			ratio := float64(report.Soak.P99) / float64(report.Soak.P50)
			if ratio > thresholds.P99Multiplier {
				violations = append(violations, fmt.Sprintf(
					"soak P99/P50 ratio %.2f exceeds threshold %.1f",
					ratio, thresholds.P99Multiplier,
				))
			}
		}
	}

	return violations
}

// pvGateCI evaluates threshold violations and returns a pass/fail decision
// for CI gates. Returns true if the gate passes (no violations), along with
// a human-readable summary.
func pvGateCI(violations []string) (bool, string) {
	if len(violations) == 0 {
		return true, "CI gate: PASS - all performance thresholds met"
	}

	var sb fmt.Stringer = pvBuildGateSummary(violations)
	return false, sb.String()
}

// pvBuildGateSummary constructs a formatted CI gate failure summary.
func pvBuildGateSummary(violations []string) fmt.Stringer {
	return &pvGateSummaryBuilder{violations: violations}
}

type pvGateSummaryBuilder struct {
	violations []string
}

func (b *pvGateSummaryBuilder) String() string {
	result := fmt.Sprintf("CI gate: FAIL - %d violation(s):\n", len(b.violations))
	for i, v := range b.violations {
		result += fmt.Sprintf("  %d. %s\n", i+1, v)
	}
	return result
}
