package perfval

import (
	"fmt"
	"runtime"
	"sort"
	"time"
)

// SoakConfig defines the parameters for a soak test that repeatedly executes
// a work function over a sustained period.
type SoakConfig struct {
	// Duration is how long the soak test runs.
	Duration time.Duration

	// Interval is the pause between successive invocations of WorkFn.
	Interval time.Duration

	// WorkFn is the function executed on each iteration.
	WorkFn func() error

	// Label is a human-readable name for the soak test.
	Label string
}

// SoakResult captures the outcome of a soak test including timing statistics,
// error counts, and memory snapshots from the beginning and end of the run.
type SoakResult struct {
	// Iterations is the total number of times WorkFn was invoked.
	Iterations int

	// Errors is the number of times WorkFn returned a non-nil error.
	Errors int

	// AvgDuration is the arithmetic mean of all invocation durations.
	AvgDuration time.Duration

	// P50 is the median invocation duration.
	P50 time.Duration

	// P95 is the 95th percentile invocation duration.
	P95 time.Duration

	// P99 is the 99th percentile invocation duration.
	P99 time.Duration

	// MaxDuration is the longest single invocation.
	MaxDuration time.Duration

	// MemStart is the memory snapshot taken before the soak test begins.
	MemStart *MemSnapshot

	// MemEnd is the memory snapshot taken after the soak test completes.
	MemEnd *MemSnapshot

	// Stable indicates whether the soak test met all stability criteria.
	Stable bool
}

// RunSoak executes the configured work function repeatedly for the specified
// duration, collecting timing and memory metrics throughout.
func RunSoak(config *SoakConfig) (*SoakResult, error) {
	if config == nil {
		return nil, fmt.Errorf("soak config must not be nil")
	}
	if config.WorkFn == nil {
		return nil, fmt.Errorf("soak config WorkFn must not be nil")
	}
	if config.Duration <= 0 {
		return nil, fmt.Errorf("soak duration must be positive, got %v", config.Duration)
	}
	if config.Interval <= 0 {
		return nil, fmt.Errorf("soak interval must be positive, got %v", config.Interval)
	}

	// Force GC and take baseline snapshot.
	runtime.GC()
	memStart := pvTakeSnapshot()

	var durations []time.Duration
	errors := 0
	deadline := time.Now().Add(config.Duration)

	for time.Now().Before(deadline) {
		start := time.Now()
		err := config.WorkFn()
		elapsed := time.Since(start)
		durations = append(durations, elapsed)
		if err != nil {
			errors++
		}
		time.Sleep(config.Interval)
	}

	// Take final snapshot.
	runtime.GC()
	memEnd := pvTakeSnapshot()

	if len(durations) == 0 {
		return &SoakResult{
			MemStart: memStart,
			MemEnd:   memEnd,
			Stable:   true,
		}, nil
	}

	p50, p95, p99 := pvComputePercentiles(durations)

	var totalNs int64
	maxDur := time.Duration(0)
	for _, d := range durations {
		totalNs += int64(d)
		if d > maxDur {
			maxDur = d
		}
	}
	avg := time.Duration(totalNs / int64(len(durations)))

	result := &SoakResult{
		Iterations:  len(durations),
		Errors:      errors,
		AvgDuration: avg,
		P50:         p50,
		P95:         p95,
		P99:         p99,
		MaxDuration: maxDur,
		MemStart:    memStart,
		MemEnd:      memEnd,
	}

	result.Stable = pvIsStable(result)
	return result, nil
}

// pvComputePercentiles calculates the 50th, 95th, and 99th percentile
// durations from a slice of measured durations. Returns zeros if the slice
// is empty.
func pvComputePercentiles(durations []time.Duration) (p50, p95, p99 time.Duration) {
	if len(durations) == 0 {
		return 0, 0, 0
	}

	sorted := make([]time.Duration, len(durations))
	copy(sorted, durations)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })

	n := len(sorted)
	p50 = sorted[pvPercentileIndex(n, 0.50)]
	p95 = sorted[pvPercentileIndex(n, 0.95)]
	p99 = sorted[pvPercentileIndex(n, 0.99)]
	return
}

// pvPercentileIndex computes the index for a given percentile in a sorted
// slice of the given length.
func pvPercentileIndex(n int, pct float64) int {
	idx := int(float64(n-1) * pct)
	if idx < 0 {
		return 0
	}
	if idx >= n {
		return n - 1
	}
	return idx
}

// pvIsStable checks whether a soak result meets all stability criteria:
//   - Error rate must be less than 1%
//   - P99 must be less than 2x P50 (no tail latency blowup)
//   - No memory leak detected between start and end snapshots
func pvIsStable(result *SoakResult) bool {
	if result.Iterations == 0 {
		return true
	}

	// Check error rate.
	errorRate := float64(result.Errors) / float64(result.Iterations)
	if errorRate >= 0.01 {
		return false
	}

	// Check tail latency.
	if result.P50 > 0 && result.P99 > 2*result.P50 {
		return false
	}

	// Check memory growth.
	if result.MemStart != nil && result.MemEnd != nil {
		baseline := result.MemStart.HeapAlloc
		if baseline > 0 {
			current := result.MemEnd.HeapAlloc
			if current > baseline {
				growth := float64(current-baseline) / float64(baseline)
				if growth > 0.20 {
					return false
				}
			}
		}
	}

	return true
}

// QuickSoak runs a fast 10-second soak test with 100ms intervals. This is
// useful for smoke-testing stability without the overhead of a full soak.
func QuickSoak(label string, fn func() error) (*SoakResult, error) {
	return RunSoak(&SoakConfig{
		Duration: 10 * time.Second,
		Interval: 100 * time.Millisecond,
		WorkFn:   fn,
		Label:    label,
	})
}
