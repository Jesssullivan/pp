package perfval

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// BenchmarkSuite holds a collection of benchmark results, optionally
// compared against a baseline suite to detect regressions.
type BenchmarkSuite struct {
	// Name identifies the suite (e.g., "prompt-pulse v2.0").
	Name string

	// Results contains the individual benchmark measurements.
	Results []BenchResult

	// Baseline is an optional previous suite to compare against.
	Baseline *BenchmarkSuite
}

// BenchResult captures the metrics from a single Go benchmark run.
type BenchResult struct {
	// Name is the benchmark function name (e.g., "BenchmarkBannerCached").
	Name string

	// Iterations is the number of times the benchmark loop ran.
	Iterations int

	// NsPerOp is the average nanoseconds per operation.
	NsPerOp int64

	// AllocsPerOp is the average number of allocations per operation.
	AllocsPerOp int64

	// BytesPerOp is the average bytes allocated per operation.
	BytesPerOp int64
}

// Regression describes a performance change between current and baseline
// benchmark results.
type Regression struct {
	// Name is the benchmark that regressed.
	Name string

	// CurrentNs is the current nanoseconds per operation.
	CurrentNs int64

	// BaselineNs is the baseline nanoseconds per operation.
	BaselineNs int64

	// PercentChange is the percentage change (positive = slower).
	PercentChange float64

	// IsRegression is true if the change exceeds the regression threshold.
	IsRegression bool
}

// pvParseBenchOutput parses the text output of `go test -bench` into a
// slice of BenchResult. It handles the standard format:
//
//	BenchmarkXxx-N  iterations  ns/op  B/op  allocs/op
//
// Lines that do not match the benchmark format are silently skipped.
func pvParseBenchOutput(output string) ([]BenchResult, error) {
	var results []BenchResult

	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "Benchmark") {
			continue
		}

		result, err := pvParseBenchLine(line)
		if err != nil {
			continue // skip malformed lines
		}
		results = append(results, *result)
	}

	return results, nil
}

// pvParseBenchLine parses a single benchmark output line into a BenchResult.
func pvParseBenchLine(line string) (*BenchResult, error) {
	fields := strings.Fields(line)
	if len(fields) < 3 {
		return nil, fmt.Errorf("too few fields in benchmark line: %q", line)
	}

	result := &BenchResult{}

	// First field: benchmark name (may include -N CPU suffix).
	result.Name = fields[0]

	// Second field: iteration count.
	iters, err := strconv.Atoi(fields[1])
	if err != nil {
		return nil, fmt.Errorf("invalid iteration count %q: %w", fields[1], err)
	}
	result.Iterations = iters

	// Parse remaining metric fields by looking for known suffixes.
	for i := 2; i < len(fields)-1; i += 2 {
		value := fields[i]
		unit := fields[i+1]

		switch unit {
		case "ns/op":
			ns, err := pvParseFloat(value)
			if err == nil {
				result.NsPerOp = int64(ns)
			}
		case "B/op":
			b, err := pvParseFloat(value)
			if err == nil {
				result.BytesPerOp = int64(b)
			}
		case "allocs/op":
			a, err := pvParseFloat(value)
			if err == nil {
				result.AllocsPerOp = int64(a)
			}
		}
	}

	return result, nil
}

// pvParseFloat parses a numeric string that may be an integer or float.
func pvParseFloat(s string) (float64, error) {
	return strconv.ParseFloat(s, 64)
}

// pvCompare compares current benchmark results against a baseline suite,
// producing a Regression entry for each matching benchmark name.
func pvCompare(current, baseline *BenchmarkSuite) []Regression {
	if baseline == nil || current == nil {
		return nil
	}

	baseMap := make(map[string]BenchResult, len(baseline.Results))
	for _, r := range baseline.Results {
		baseMap[r.Name] = r
	}

	var regressions []Regression
	for _, cur := range current.Results {
		base, ok := baseMap[cur.Name]
		if !ok {
			continue
		}

		pctChange := 0.0
		if base.NsPerOp > 0 {
			pctChange = float64(cur.NsPerOp-base.NsPerOp) / float64(base.NsPerOp) * 100.0
		}

		regressions = append(regressions, Regression{
			Name:          cur.Name,
			CurrentNs:     cur.NsPerOp,
			BaselineNs:    base.NsPerOp,
			PercentChange: pctChange,
			IsRegression:  false, // set by pvDetectRegressions
		})
	}

	return regressions
}

// pvDetectRegressions filters a list of comparisons to find those that
// exceed the given threshold percentage. The threshold is expressed as a
// fraction (e.g., 0.15 for 15%). Comparisons that exceed the threshold
// have their IsRegression flag set to true.
func pvDetectRegressions(comparisons []Regression, threshold float64) []Regression {
	var regressions []Regression
	for _, c := range comparisons {
		if c.PercentChange > threshold*100 {
			c.IsRegression = true
			regressions = append(regressions, c)
		}
	}
	return regressions
}

// pvRenderComparison produces a Markdown table summarizing benchmark
// regressions. The table includes columns for the benchmark name, baseline
// and current times, and the percentage change.
func pvRenderComparison(regressions []Regression) string {
	if len(regressions) == 0 {
		return "No regressions detected."
	}

	var sb strings.Builder
	sb.WriteString("| Benchmark | Baseline | Current | Change |\n")
	sb.WriteString("|-----------|----------|---------|--------|\n")

	for _, r := range regressions {
		status := ""
		if r.IsRegression {
			status = " REGRESSION"
		}
		sb.WriteString(fmt.Sprintf("| %s | %s | %s | %+.1f%%%s |\n",
			r.Name,
			pvFormatNs(r.BaselineNs),
			pvFormatNs(r.CurrentNs),
			r.PercentChange,
			status,
		))
	}

	return sb.String()
}

// pvFormatNs formats nanoseconds into a human-readable duration string.
func pvFormatNs(ns int64) string {
	absNs := ns
	if absNs < 0 {
		absNs = -absNs
	}
	switch {
	case absNs >= 1_000_000_000:
		return fmt.Sprintf("%.2fs", float64(ns)/1e9)
	case absNs >= 1_000_000:
		return fmt.Sprintf("%.2fms", float64(ns)/1e6)
	case absNs >= 1_000:
		return fmt.Sprintf("%.2fus", float64(ns)/1e3)
	default:
		return fmt.Sprintf("%dns", ns)
	}
}

// pvAbsFloat64 returns the absolute value of a float64.
// Kept as a helper even though math.Abs exists, to avoid importing math
// solely for this in callers that do not already use it.
func pvAbsFloat64(f float64) float64 {
	return math.Abs(f)
}
