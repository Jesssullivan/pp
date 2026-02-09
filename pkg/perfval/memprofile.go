package perfval

import (
	"fmt"
	"math"
	"runtime"
	"time"
)

// MemProfile holds a series of memory snapshots collected over time, along
// with derived metrics for leak detection and growth analysis.
type MemProfile struct {
	// Snapshots contains the chronologically ordered memory measurements.
	Snapshots []MemSnapshot

	// LeakDetected is true if heap growth exceeds the configured threshold
	// after garbage collection stabilizes.
	LeakDetected bool

	// GrowthRate is the linear regression slope of heap allocation over time,
	// expressed as bytes per second.
	GrowthRate float64

	// MaxHeap is the peak HeapAlloc observed across all snapshots.
	MaxHeap uint64
}

// MemSnapshot captures memory and goroutine metrics at a single point in time.
type MemSnapshot struct {
	// Timestamp records when the snapshot was taken.
	Timestamp time.Time

	// HeapAlloc is the number of bytes of allocated heap objects.
	HeapAlloc uint64

	// HeapSys is the number of bytes of heap memory obtained from the OS.
	HeapSys uint64

	// HeapInuse is the number of bytes in in-use heap spans.
	HeapInuse uint64

	// NumGC is the number of completed GC cycles.
	NumGC uint32

	// GoroutineCount is the number of goroutines that currently exist.
	GoroutineCount int
}

// pvTakeSnapshot reads the current memory statistics and goroutine count,
// returning a timestamped snapshot.
func pvTakeSnapshot() *MemSnapshot {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	return &MemSnapshot{
		Timestamp:      time.Now(),
		HeapAlloc:      ms.HeapAlloc,
		HeapSys:        ms.HeapSys,
		HeapInuse:      ms.HeapInuse,
		NumGC:          ms.NumGC,
		GoroutineCount: runtime.NumGoroutine(),
	}
}

// pvStartMemProfile collects memory snapshots at the given interval for the
// specified duration, then analyzes the results for leaks and growth trends.
func pvStartMemProfile(interval time.Duration, duration time.Duration) (*MemProfile, error) {
	if interval <= 0 {
		return nil, fmt.Errorf("interval must be positive, got %v", interval)
	}
	if duration <= 0 {
		return nil, fmt.Errorf("duration must be positive, got %v", duration)
	}

	// Force GC before profiling to establish a clean baseline.
	runtime.GC()

	profile := &MemProfile{}
	deadline := time.Now().Add(duration)

	for time.Now().Before(deadline) {
		snap := pvTakeSnapshot()
		profile.Snapshots = append(profile.Snapshots, *snap)
		if snap.HeapAlloc > profile.MaxHeap {
			profile.MaxHeap = snap.HeapAlloc
		}
		time.Sleep(interval)
	}

	// Take one final snapshot.
	snap := pvTakeSnapshot()
	profile.Snapshots = append(profile.Snapshots, *snap)
	if snap.HeapAlloc > profile.MaxHeap {
		profile.MaxHeap = snap.HeapAlloc
	}

	profile.GrowthRate = pvAnalyzeGrowth(profile.Snapshots)
	profile.LeakDetected, _ = pvDetectLeak(profile)

	return profile, nil
}

// pvDetectLeak analyzes a memory profile to determine whether a memory leak
// is present. A leak is detected if heap allocation grows by more than 20%
// over the baseline (first snapshot) after garbage collection has run.
// Returns the detection result and a human-readable explanation.
func pvDetectLeak(profile *MemProfile) (bool, string) {
	if len(profile.Snapshots) < 2 {
		return false, "insufficient snapshots for leak detection"
	}

	baseline := profile.Snapshots[0].HeapAlloc
	if baseline == 0 {
		return false, "baseline heap allocation is zero"
	}

	// Use the last snapshot as the current state.
	current := profile.Snapshots[len(profile.Snapshots)-1].HeapAlloc

	growth := float64(current-baseline) / float64(baseline)
	if current < baseline {
		growth = -float64(baseline-current) / float64(baseline)
	}

	if growth > 0.20 {
		return true, fmt.Sprintf(
			"heap grew %.1f%% from %s to %s (threshold: 20%%)",
			growth*100, pvFormatBytes(baseline), pvFormatBytes(current),
		)
	}

	return false, fmt.Sprintf(
		"heap growth %.1f%% is within threshold (baseline: %s, current: %s)",
		growth*100, pvFormatBytes(baseline), pvFormatBytes(current),
	)
}

// pvAnalyzeGrowth computes a linear regression on heap allocation over time,
// returning the slope in bytes per second. A positive slope indicates growing
// memory usage; near-zero or negative indicates stability or GC reclaiming.
func pvAnalyzeGrowth(snapshots []MemSnapshot) float64 {
	n := len(snapshots)
	if n < 2 {
		return 0
	}

	// Use time offset in seconds from first snapshot as X, HeapAlloc as Y.
	t0 := snapshots[0].Timestamp
	var sumX, sumY, sumXY, sumX2 float64

	for _, s := range snapshots {
		x := s.Timestamp.Sub(t0).Seconds()
		y := float64(s.HeapAlloc)
		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
	}

	fn := float64(n)
	denom := fn*sumX2 - sumX*sumX
	if math.Abs(denom) < 1e-10 {
		return 0
	}

	slope := (fn*sumXY - sumX*sumY) / denom
	return slope
}

// pvFormatBytes converts a byte count to a human-readable string with
// appropriate units (B, KB, MB, GB).
func pvFormatBytes(bytes uint64) string {
	const (
		kb = 1024
		mb = kb * 1024
		gb = mb * 1024
	)

	switch {
	case bytes >= gb:
		return fmt.Sprintf("%.2f GB", float64(bytes)/float64(gb))
	case bytes >= mb:
		return fmt.Sprintf("%.2f MB", float64(bytes)/float64(mb))
	case bytes >= kb:
		return fmt.Sprintf("%.2f KB", float64(bytes)/float64(kb))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}
