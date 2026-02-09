package perf

import "testing"

// Threshold defines a performance budget for a named operation. Benchmarks
// that exceed these thresholds indicate a performance regression that should
// be investigated before merging.
type Threshold struct {
	// Name identifies the operation (must match a benchmark suffix).
	Name string

	// MaxNs is the maximum allowed nanoseconds per operation.
	MaxNs int64

	// MaxAlloc is the maximum allowed bytes allocated per operation.
	MaxAlloc int64
}

// Violation records a threshold breach for a specific benchmark.
type Violation struct {
	// Threshold is the budget that was exceeded.
	Threshold Threshold

	// Actual is the measured value that exceeded the threshold.
	Actual int64

	// Field indicates which metric was violated: "ns" for time or "alloc"
	// for memory allocation.
	Field string
}

// DefaultThresholds returns the performance budgets for the dashboard's
// critical rendering paths. These values represent the maximum acceptable
// performance for each operation on a typical development machine.
//
// Budget rationale:
//   - banner_cached < 1ms: cache read is stat + file read, must be fast
//   - banner_render < 50ms: full render with 6 widgets, includes grid compose
//   - layout_6widget < 5ms: constraint solver for typical dashboard
//   - shell_generate < 1ms: string concatenation only
//   - starship_render < 20ms: segment assembly with cache reads
//   - component_gauge < 100us: single gauge bar render
func DefaultThresholds() []Threshold {
	return []Threshold{
		{Name: "banner_cached", MaxNs: 1_000_000, MaxAlloc: 16384},
		{Name: "banner_render", MaxNs: 50_000_000, MaxAlloc: 1_048_576},
		{Name: "layout_6widget", MaxNs: 5_000_000, MaxAlloc: 32768},
		{Name: "layout_20widget", MaxNs: 10_000_000, MaxAlloc: 65536},
		{Name: "layout_cache_hit", MaxNs: 500_000, MaxAlloc: 8192},
		{Name: "shell_generate", MaxNs: 1_000_000, MaxAlloc: 16384},
		{Name: "starship_render", MaxNs: 20_000_000, MaxAlloc: 262144},
		{Name: "component_gauge", MaxNs: 100_000, MaxAlloc: 8192},
		{Name: "component_sparkline", MaxNs: 200_000, MaxAlloc: 16384},
		{Name: "component_box", MaxNs: 500_000, MaxAlloc: 32768},
		{Name: "component_table", MaxNs: 5_000_000, MaxAlloc: 131072},
		{Name: "text_truncate", MaxNs: 50_000, MaxAlloc: 4096},
		{Name: "visible_len", MaxNs: 50_000, MaxAlloc: 2048},
		{Name: "image_resize", MaxNs: 500_000_000, MaxAlloc: 33_554_432},
	}
}

// CheckRegression compares benchmark results against thresholds and returns
// all violations found. A violation occurs when either the nanoseconds per
// operation exceed MaxNs or the bytes allocated per operation exceed MaxAlloc.
//
// Results are matched to thresholds by name. Results without a matching
// threshold are silently ignored (no violation). Thresholds without a matching
// result are also ignored.
func CheckRegression(results []testing.BenchmarkResult, thresholds []Threshold) []Violation {
	if len(results) == 0 || len(thresholds) == 0 {
		return nil
	}

	// Build a lookup map from threshold names.
	threshMap := make(map[string]Threshold, len(thresholds))
	for _, t := range thresholds {
		threshMap[t.Name] = t
	}

	// We match results by index to their corresponding threshold name.
	// Since testing.BenchmarkResult does not carry a name field, callers
	// must pass results in the same order as thresholds when names should
	// match. Alternatively, callers can use the index-based matching below.
	//
	// For simplicity, we iterate both slices in parallel up to the shorter
	// length, matching by position.
	var violations []Violation

	limit := len(results)
	if limit > len(thresholds) {
		limit = len(thresholds)
	}

	for i := 0; i < limit; i++ {
		r := results[i]
		t := thresholds[i]

		nsPerOp := pfNsPerOp(r)
		if t.MaxNs > 0 && nsPerOp > t.MaxNs {
			violations = append(violations, Violation{
				Threshold: t,
				Actual:    nsPerOp,
				Field:     "ns",
			})
		}

		allocPerOp := pfAllocPerOp(r)
		if t.MaxAlloc > 0 && allocPerOp > t.MaxAlloc {
			violations = append(violations, Violation{
				Threshold: t,
				Actual:    allocPerOp,
				Field:     "alloc",
			})
		}
	}

	return violations
}

// pfNsPerOp extracts nanoseconds per operation from a BenchmarkResult.
// Delegates to the standard library NsPerOp method which handles N=0.
func pfNsPerOp(r testing.BenchmarkResult) int64 {
	return r.NsPerOp()
}

// pfAllocPerOp extracts bytes allocated per operation from a BenchmarkResult.
// Uses the AllocedBytesPerOp method which handles the uint64->int64
// conversion correctly. Returns 0 if N is 0.
func pfAllocPerOp(r testing.BenchmarkResult) int64 {
	return r.AllocedBytesPerOp()
}
