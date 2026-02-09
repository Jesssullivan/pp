package perfval

import (
	"fmt"
	"math"
	"runtime"
	"strings"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// Target tests
// ---------------------------------------------------------------------------

func TestDefaultTargets(t *testing.T) {
	targets := DefaultTargets()
	if len(targets) != 5 {
		t.Fatalf("expected 5 default targets, got %d", len(targets))
	}

	names := map[string]bool{}
	for _, tgt := range targets {
		if tgt.Name == "" {
			t.Error("target has empty name")
		}
		if tgt.MaxDuration <= 0 {
			t.Errorf("target %q has non-positive max duration: %v", tgt.Name, tgt.MaxDuration)
		}
		if tgt.Description == "" {
			t.Errorf("target %q has empty description", tgt.Name)
		}
		names[tgt.Name] = true
	}

	expected := []string{"banner_cached", "tui_frame", "image_kitty", "shell_source", "cache_read"}
	for _, name := range expected {
		if !names[name] {
			t.Errorf("missing expected target %q", name)
		}
	}
}

func TestValidateTargetPassing(t *testing.T) {
	target := Target{
		Name:        "fast_op",
		MaxDuration: 100 * time.Millisecond,
		Description: "fast operation",
	}

	result := ValidateTarget(target, func() error {
		// Deliberately fast: should pass easily.
		return nil
	}, 10)

	if !result.Passed {
		t.Errorf("expected target to pass, actual: %v", result.Actual)
	}
	if result.Samples != 10 {
		t.Errorf("expected 10 samples, got %d", result.Samples)
	}
	if result.Target != "fast_op" {
		t.Errorf("expected target name 'fast_op', got %q", result.Target)
	}
	if result.Margin < 0 {
		t.Errorf("expected positive margin for passing target, got %f", result.Margin)
	}
}

func TestValidateTargetFailing(t *testing.T) {
	target := Target{
		Name:        "slow_op",
		MaxDuration: 1 * time.Nanosecond, // impossibly tight budget
		Description: "will fail",
	}

	result := ValidateTarget(target, func() error {
		time.Sleep(1 * time.Millisecond)
		return nil
	}, 5)

	if result.Passed {
		t.Error("expected target to fail with 1ns budget")
	}
	if result.Margin >= 0 {
		t.Errorf("expected negative margin for failing target, got %f", result.Margin)
	}
}

func TestValidateAll(t *testing.T) {
	targets := []Target{
		{Name: "op_a", MaxDuration: 100 * time.Millisecond, Description: "a"},
		{Name: "op_b", MaxDuration: 100 * time.Millisecond, Description: "b"},
		{Name: "op_c", MaxDuration: 100 * time.Millisecond, Description: "c"},
	}

	fns := map[string]func() error{
		"op_a": func() error { return nil },
		"op_b": func() error { return nil },
		// op_c intentionally missing to test skip behavior.
	}

	report := ValidateAll(targets, fns, 5)
	if len(report.Results) != 2 {
		t.Fatalf("expected 2 results (op_c skipped), got %d", len(report.Results))
	}
	if !report.AllPassed {
		t.Error("expected all targets to pass")
	}
	if report.Timestamp.IsZero() {
		t.Error("expected non-zero timestamp")
	}
}

func TestValidateAllWithFailure(t *testing.T) {
	targets := []Target{
		{Name: "fast", MaxDuration: 100 * time.Millisecond, Description: "fast"},
		{Name: "slow", MaxDuration: 1 * time.Nanosecond, Description: "slow"},
	}

	fns := map[string]func() error{
		"fast": func() error { return nil },
		"slow": func() error { time.Sleep(1 * time.Millisecond); return nil },
	}

	report := ValidateAll(targets, fns, 3)
	if report.AllPassed {
		t.Error("expected AllPassed to be false when one target fails")
	}
}

func TestValidateTargetZeroSamples(t *testing.T) {
	target := Target{Name: "zero", MaxDuration: time.Second, Description: "zero"}
	result := ValidateTarget(target, func() error { return nil }, 0)
	if !result.Passed {
		t.Error("zero samples should pass")
	}
	if result.Samples != 0 {
		t.Errorf("expected 0 samples, got %d", result.Samples)
	}
}

// ---------------------------------------------------------------------------
// Memory profiling tests
// ---------------------------------------------------------------------------

func TestTakeSnapshot(t *testing.T) {
	snap := pvTakeSnapshot()
	if snap == nil {
		t.Fatal("snapshot should not be nil")
	}
	if snap.Timestamp.IsZero() {
		t.Error("snapshot timestamp should not be zero")
	}
}

func TestTakeSnapshotNonZeroValues(t *testing.T) {
	// Allocate something to ensure non-zero heap.
	data := make([]byte, 1024)
	_ = data

	snap := pvTakeSnapshot()
	if snap.HeapAlloc == 0 {
		t.Error("HeapAlloc should not be zero after allocation")
	}
	if snap.HeapSys == 0 {
		t.Error("HeapSys should not be zero")
	}
	if snap.GoroutineCount == 0 {
		t.Error("GoroutineCount should not be zero")
	}
}

func TestDetectNoLeak(t *testing.T) {
	profile := &MemProfile{
		Snapshots: []MemSnapshot{
			{HeapAlloc: 1000000},
			{HeapAlloc: 1050000}, // 5% growth, under 20% threshold
		},
	}

	leaked, msg := pvDetectLeak(profile)
	if leaked {
		t.Errorf("should not detect leak with 5%% growth: %s", msg)
	}
}

func TestDetectLeak(t *testing.T) {
	profile := &MemProfile{
		Snapshots: []MemSnapshot{
			{HeapAlloc: 1000000},
			{HeapAlloc: 1500000}, // 50% growth, over 20% threshold
		},
	}

	leaked, msg := pvDetectLeak(profile)
	if !leaked {
		t.Errorf("should detect leak with 50%% growth: %s", msg)
	}
}

func TestDetectLeakInsufficientSnapshots(t *testing.T) {
	profile := &MemProfile{
		Snapshots: []MemSnapshot{
			{HeapAlloc: 1000000},
		},
	}

	leaked, _ := pvDetectLeak(profile)
	if leaked {
		t.Error("should not detect leak with only one snapshot")
	}
}

func TestAnalyzeGrowth(t *testing.T) {
	now := time.Now()
	snapshots := []MemSnapshot{
		{Timestamp: now, HeapAlloc: 1000000},
		{Timestamp: now.Add(1 * time.Second), HeapAlloc: 2000000},
		{Timestamp: now.Add(2 * time.Second), HeapAlloc: 3000000},
	}

	slope := pvAnalyzeGrowth(snapshots)
	// Linear growth of 1MB/s => slope should be ~1000000 bytes/sec.
	if slope < 900000 || slope > 1100000 {
		t.Errorf("expected slope ~1000000, got %.0f", slope)
	}
}

func TestAnalyzeGrowthFlat(t *testing.T) {
	now := time.Now()
	snapshots := []MemSnapshot{
		{Timestamp: now, HeapAlloc: 1000000},
		{Timestamp: now.Add(1 * time.Second), HeapAlloc: 1000000},
		{Timestamp: now.Add(2 * time.Second), HeapAlloc: 1000000},
	}

	slope := pvAnalyzeGrowth(snapshots)
	if math.Abs(slope) > 1.0 {
		t.Errorf("expected near-zero slope for flat profile, got %.2f", slope)
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		input    uint64
		contains string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1024, "1.00 KB"},
		{1536, "1.50 KB"},
		{1048576, "1.00 MB"},
		{1073741824, "1.00 GB"},
	}

	for _, tc := range tests {
		result := pvFormatBytes(tc.input)
		if result != tc.contains {
			t.Errorf("pvFormatBytes(%d) = %q, want %q", tc.input, result, tc.contains)
		}
	}
}

func TestStartMemProfile(t *testing.T) {
	profile, err := pvStartMemProfile(50*time.Millisecond, 200*time.Millisecond)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(profile.Snapshots) < 2 {
		t.Errorf("expected at least 2 snapshots, got %d", len(profile.Snapshots))
	}
}

func TestStartMemProfileInvalidArgs(t *testing.T) {
	_, err := pvStartMemProfile(0, time.Second)
	if err == nil {
		t.Error("expected error for zero interval")
	}

	_, err = pvStartMemProfile(time.Second, 0)
	if err == nil {
		t.Error("expected error for zero duration")
	}
}

// ---------------------------------------------------------------------------
// Soak test tests
// ---------------------------------------------------------------------------

func TestQuickSoakShort(t *testing.T) {
	// Use a short custom soak instead of the full 10-second QuickSoak.
	result, err := RunSoak(&SoakConfig{
		Duration: 500 * time.Millisecond,
		Interval: 50 * time.Millisecond,
		WorkFn:   func() error { return nil },
		Label:    "test_quick",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Iterations == 0 {
		t.Error("expected at least one iteration")
	}
	if result.Errors != 0 {
		t.Errorf("expected 0 errors, got %d", result.Errors)
	}
}

func TestSoakWithErrors(t *testing.T) {
	callCount := 0
	result, err := RunSoak(&SoakConfig{
		Duration: 300 * time.Millisecond,
		Interval: 25 * time.Millisecond,
		WorkFn: func() error {
			callCount++
			if callCount%2 == 0 {
				return fmt.Errorf("test error")
			}
			return nil
		},
		Label: "error_test",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Errors == 0 {
		t.Error("expected some errors")
	}
	if result.Stable {
		t.Error("expected unstable result with high error rate")
	}
}

func TestSoakStability(t *testing.T) {
	result, err := RunSoak(&SoakConfig{
		Duration: 300 * time.Millisecond,
		Interval: 25 * time.Millisecond,
		WorkFn:   func() error { return nil },
		Label:    "stable_test",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Stable {
		t.Error("expected stable result for fast, error-free workload")
	}
}

func TestComputePercentiles(t *testing.T) {
	durations := make([]time.Duration, 100)
	for i := 0; i < 100; i++ {
		durations[i] = time.Duration(i+1) * time.Millisecond
	}

	p50, p95, p99 := pvComputePercentiles(durations)

	// p50 should be around 50ms.
	if p50 < 49*time.Millisecond || p50 > 51*time.Millisecond {
		t.Errorf("expected p50 ~50ms, got %v", p50)
	}
	// p95 should be around 95ms.
	if p95 < 94*time.Millisecond || p95 > 96*time.Millisecond {
		t.Errorf("expected p95 ~95ms, got %v", p95)
	}
	// p99 should be around 99ms.
	if p99 < 98*time.Millisecond || p99 > 100*time.Millisecond {
		t.Errorf("expected p99 ~99ms, got %v", p99)
	}
}

func TestComputePercentilesSingleValue(t *testing.T) {
	p50, p95, p99 := pvComputePercentiles([]time.Duration{42 * time.Millisecond})
	if p50 != 42*time.Millisecond || p95 != 42*time.Millisecond || p99 != 42*time.Millisecond {
		t.Errorf("single value should return same for all percentiles: p50=%v p95=%v p99=%v", p50, p95, p99)
	}
}

func TestComputePercentilesTwoValues(t *testing.T) {
	p50, p95, p99 := pvComputePercentiles([]time.Duration{
		10 * time.Millisecond,
		20 * time.Millisecond,
	})
	// With 2 values (indices 0,1), percentile index = int((2-1)*pct):
	// p50 = int(0.50) = 0 -> 10ms
	// p95 = int(0.95) = 0 -> 10ms
	// p99 = int(0.99) = 0 -> 10ms
	// All map to index 0 because int() truncates.
	if p50 != 10*time.Millisecond {
		t.Errorf("expected p50=10ms, got %v", p50)
	}
	if p95 != 10*time.Millisecond {
		t.Errorf("expected p95=10ms, got %v", p95)
	}
	if p99 != 10*time.Millisecond {
		t.Errorf("expected p99=10ms, got %v", p99)
	}
}

func TestComputePercentilesAllSame(t *testing.T) {
	durations := make([]time.Duration, 50)
	for i := range durations {
		durations[i] = 5 * time.Millisecond
	}

	p50, p95, p99 := pvComputePercentiles(durations)
	if p50 != 5*time.Millisecond || p95 != 5*time.Millisecond || p99 != 5*time.Millisecond {
		t.Errorf("all same values should return same percentiles: p50=%v p95=%v p99=%v", p50, p95, p99)
	}
}

func TestComputePercentilesEmpty(t *testing.T) {
	p50, p95, p99 := pvComputePercentiles(nil)
	if p50 != 0 || p95 != 0 || p99 != 0 {
		t.Errorf("empty input should return zeros: p50=%v p95=%v p99=%v", p50, p95, p99)
	}
}

func TestSoakNilConfig(t *testing.T) {
	_, err := RunSoak(nil)
	if err == nil {
		t.Error("expected error for nil config")
	}
}

func TestSoakNilWorkFn(t *testing.T) {
	_, err := RunSoak(&SoakConfig{
		Duration: time.Second,
		Interval: time.Millisecond,
		WorkFn:   nil,
		Label:    "nil_fn",
	})
	if err == nil {
		t.Error("expected error for nil WorkFn")
	}
}

func TestSoakZeroDuration(t *testing.T) {
	_, err := RunSoak(&SoakConfig{
		Duration: 0,
		Interval: time.Millisecond,
		WorkFn:   func() error { return nil },
		Label:    "zero_dur",
	})
	if err == nil {
		t.Error("expected error for zero duration")
	}
}

func TestSoakMemorySnapshots(t *testing.T) {
	result, err := RunSoak(&SoakConfig{
		Duration: 200 * time.Millisecond,
		Interval: 25 * time.Millisecond,
		WorkFn:   func() error { return nil },
		Label:    "mem_check",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.MemStart == nil {
		t.Error("expected non-nil MemStart")
	}
	if result.MemEnd == nil {
		t.Error("expected non-nil MemEnd")
	}
}

// ---------------------------------------------------------------------------
// Benchmark parsing tests
// ---------------------------------------------------------------------------

func TestParseBenchOutput(t *testing.T) {
	output := `goos: darwin
goarch: arm64
pkg: example.com/test
BenchmarkBannerCached-10    	 5000000	       234 ns/op	     128 B/op	       2 allocs/op
BenchmarkTUIFrame-10        	 1000000	      1234 ns/op	     512 B/op	       5 allocs/op
BenchmarkImageKitty-10      	 2000000	       567 ns/op	     256 B/op	       3 allocs/op
PASS
ok  	example.com/test	4.567s`

	results, err := pvParseBenchOutput(output)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	if results[0].Name != "BenchmarkBannerCached-10" {
		t.Errorf("expected name 'BenchmarkBannerCached-10', got %q", results[0].Name)
	}
	if results[0].Iterations != 5000000 {
		t.Errorf("expected 5000000 iterations, got %d", results[0].Iterations)
	}
	if results[0].NsPerOp != 234 {
		t.Errorf("expected 234 ns/op, got %d", results[0].NsPerOp)
	}
	if results[0].BytesPerOp != 128 {
		t.Errorf("expected 128 B/op, got %d", results[0].BytesPerOp)
	}
	if results[0].AllocsPerOp != 2 {
		t.Errorf("expected 2 allocs/op, got %d", results[0].AllocsPerOp)
	}
}

func TestParseBenchOutputEmpty(t *testing.T) {
	results, err := pvParseBenchOutput("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results for empty output, got %d", len(results))
	}
}

func TestParseBenchOutputMalformed(t *testing.T) {
	output := `BenchmarkBad notanumber 234 ns/op`
	results, err := pvParseBenchOutput(output)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Malformed lines should be skipped.
	if len(results) != 0 {
		t.Errorf("expected 0 results for malformed output, got %d", len(results))
	}
}

func TestCompareSuites(t *testing.T) {
	baseline := &BenchmarkSuite{
		Name: "v1.0",
		Results: []BenchResult{
			{Name: "BenchmarkA-10", NsPerOp: 1000},
			{Name: "BenchmarkB-10", NsPerOp: 2000},
		},
	}

	current := &BenchmarkSuite{
		Name: "v2.0",
		Results: []BenchResult{
			{Name: "BenchmarkA-10", NsPerOp: 1100},
			{Name: "BenchmarkB-10", NsPerOp: 1800},
		},
	}

	comparisons := pvCompare(current, baseline)
	if len(comparisons) != 2 {
		t.Fatalf("expected 2 comparisons, got %d", len(comparisons))
	}

	// BenchmarkA: 1000 -> 1100 = +10%
	if math.Abs(comparisons[0].PercentChange-10.0) > 0.1 {
		t.Errorf("expected ~10%% change for A, got %.1f%%", comparisons[0].PercentChange)
	}

	// BenchmarkB: 2000 -> 1800 = -10%
	if math.Abs(comparisons[1].PercentChange+10.0) > 0.1 {
		t.Errorf("expected ~-10%% change for B, got %.1f%%", comparisons[1].PercentChange)
	}
}

func TestDetectRegression(t *testing.T) {
	comparisons := []Regression{
		{Name: "A", PercentChange: 5.0},  // 5% - not a regression
		{Name: "B", PercentChange: 20.0}, // 20% - is a regression
		{Name: "C", PercentChange: -5.0}, // -5% - improvement
	}

	regressions := pvDetectRegressions(comparisons, 0.15) // 15% threshold
	if len(regressions) != 1 {
		t.Fatalf("expected 1 regression, got %d", len(regressions))
	}
	if regressions[0].Name != "B" {
		t.Errorf("expected regression in 'B', got %q", regressions[0].Name)
	}
	if !regressions[0].IsRegression {
		t.Error("IsRegression should be true")
	}
}

func TestDetectNoRegression(t *testing.T) {
	comparisons := []Regression{
		{Name: "A", PercentChange: 5.0},
		{Name: "B", PercentChange: 10.0},
	}

	regressions := pvDetectRegressions(comparisons, 0.15)
	if len(regressions) != 0 {
		t.Errorf("expected 0 regressions, got %d", len(regressions))
	}
}

func TestRenderComparison(t *testing.T) {
	regressions := []Regression{
		{Name: "BenchmarkA", CurrentNs: 1500, BaselineNs: 1000, PercentChange: 50.0, IsRegression: true},
	}

	rendered := pvRenderComparison(regressions)
	if !strings.Contains(rendered, "BenchmarkA") {
		t.Error("rendered comparison should contain benchmark name")
	}
	if !strings.Contains(rendered, "REGRESSION") {
		t.Error("rendered comparison should contain REGRESSION marker")
	}
	if !strings.Contains(rendered, "|") {
		t.Error("rendered comparison should be a Markdown table")
	}
}

func TestRenderComparisonEmpty(t *testing.T) {
	rendered := pvRenderComparison(nil)
	if !strings.Contains(rendered, "No regressions") {
		t.Error("empty regressions should show 'No regressions' message")
	}
}

func TestCompareNilBaseline(t *testing.T) {
	current := &BenchmarkSuite{Name: "v1", Results: []BenchResult{{Name: "A", NsPerOp: 100}}}
	comparisons := pvCompare(current, nil)
	if len(comparisons) != 0 {
		t.Errorf("expected 0 comparisons with nil baseline, got %d", len(comparisons))
	}
}

// ---------------------------------------------------------------------------
// Report tests
// ---------------------------------------------------------------------------

func TestGenerateReport(t *testing.T) {
	platform := pvDetectPlatform()
	report := &PerfReport{
		Targets: &ValidationReport{
			Results: []ValidationResult{
				{Target: "banner_cached", Actual: 10 * time.Millisecond, Passed: true, Margin: 0.8, Samples: 100},
			},
			AllPassed: true,
			Timestamp: time.Now(),
		},
		Soak: &SoakResult{
			Iterations:  100,
			Errors:      0,
			AvgDuration: 5 * time.Millisecond,
			P50:         4 * time.Millisecond,
			P95:         6 * time.Millisecond,
			P99:         7 * time.Millisecond,
			MaxDuration: 8 * time.Millisecond,
			Stable:      true,
		},
		Memory: &MemProfile{
			Snapshots: []MemSnapshot{
				{HeapAlloc: 1000000, Timestamp: time.Now()},
				{HeapAlloc: 1050000, Timestamp: time.Now().Add(time.Second)},
			},
			MaxHeap: 1050000,
		},
		Regressions: []Regression{
			{Name: "BenchA", CurrentNs: 1000, BaselineNs: 900, PercentChange: 11.1},
		},
		Platform: *platform,
	}

	output, err := GenerateReport(report)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	requiredSections := []string{
		"Performance Validation Report",
		"Executive Summary",
		"Target Validation",
		"Soak Test Results",
		"Memory Analysis",
		"Regression Analysis",
		"Platform",
	}

	for _, section := range requiredSections {
		if !strings.Contains(output, section) {
			t.Errorf("report missing section: %q", section)
		}
	}
}

func TestGenerateReportNil(t *testing.T) {
	_, err := GenerateReport(nil)
	if err == nil {
		t.Error("expected error for nil report")
	}
}

func TestPlatformDetection(t *testing.T) {
	platform := pvDetectPlatform()
	if platform.OS == "" {
		t.Error("OS should not be empty")
	}
	if platform.Arch == "" {
		t.Error("Arch should not be empty")
	}
	if platform.GoVersion == "" {
		t.Error("GoVersion should not be empty")
	}
	if platform.NumCPU <= 0 {
		t.Errorf("NumCPU should be positive, got %d", platform.NumCPU)
	}

	// Verify runtime values match.
	if platform.OS != runtime.GOOS {
		t.Errorf("expected OS=%s, got %s", runtime.GOOS, platform.OS)
	}
	if platform.Arch != runtime.GOARCH {
		t.Errorf("expected Arch=%s, got %s", runtime.GOARCH, platform.Arch)
	}
}

func TestRenderTargetTable(t *testing.T) {
	results := []ValidationResult{
		{Target: "banner_cached", Actual: 10 * time.Millisecond, Passed: true, Margin: 0.8, Samples: 100},
		{Target: "slow_op", Actual: 200 * time.Millisecond, Passed: false, Margin: -3.0, Samples: 50},
	}

	table := pvRenderTargetTable(results)
	if !strings.Contains(table, "banner_cached") {
		t.Error("table should contain target name")
	}
	if !strings.Contains(table, "PASS") {
		t.Error("table should contain PASS")
	}
	if !strings.Contains(table, "FAIL") {
		t.Error("table should contain FAIL")
	}
	if !strings.Contains(table, "|") {
		t.Error("table should be Markdown format")
	}
}

func TestRenderMemoryChart(t *testing.T) {
	now := time.Now()
	snapshots := []MemSnapshot{
		{Timestamp: now, HeapAlloc: 1000000},
		{Timestamp: now.Add(time.Second), HeapAlloc: 1500000},
		{Timestamp: now.Add(2 * time.Second), HeapAlloc: 2000000},
		{Timestamp: now.Add(3 * time.Second), HeapAlloc: 1800000},
	}

	chart := pvRenderMemoryChart(snapshots)
	if chart == "" {
		t.Error("chart should not be empty")
	}
	if !strings.Contains(chart, "|") {
		t.Error("chart should contain axis markers")
	}
	if !strings.Contains(chart, "-") {
		t.Error("chart should contain x-axis line")
	}
}

func TestRenderMemoryChartEmpty(t *testing.T) {
	chart := pvRenderMemoryChart(nil)
	if !strings.Contains(chart, "no data") {
		t.Error("empty chart should indicate no data")
	}
}

func TestRenderMemoryChartSingleSnapshot(t *testing.T) {
	chart := pvRenderMemoryChart([]MemSnapshot{
		{HeapAlloc: 1000000, Timestamp: time.Now()},
	})
	// Single snapshot should still produce a chart (flat line).
	if chart == "" {
		t.Error("single snapshot chart should not be empty")
	}
}

// ---------------------------------------------------------------------------
// Threshold tests
// ---------------------------------------------------------------------------

func TestDefaultThresholds(t *testing.T) {
	th := DefaultThresholds()
	if th.RegressionThreshold != 0.15 {
		t.Errorf("expected 0.15 regression threshold, got %f", th.RegressionThreshold)
	}
	if th.LeakGrowthMax != 0.20 {
		t.Errorf("expected 0.20 leak growth max, got %f", th.LeakGrowthMax)
	}
	if th.SoakErrorRateMax != 0.01 {
		t.Errorf("expected 0.01 soak error rate max, got %f", th.SoakErrorRateMax)
	}
	if th.P99Multiplier != 2.0 {
		t.Errorf("expected 2.0 P99 multiplier, got %f", th.P99Multiplier)
	}
}

func TestApplyThresholdsClean(t *testing.T) {
	report := &PerfReport{
		Targets: &ValidationReport{
			Results: []ValidationResult{
				{Target: "fast", Passed: true, Margin: 0.5},
			},
			AllPassed: true,
		},
	}

	violations := pvApplyThresholds(report, DefaultThresholds())
	if len(violations) != 0 {
		t.Errorf("expected 0 violations for clean report, got %d: %v", len(violations), violations)
	}
}

func TestApplyThresholdsWithViolations(t *testing.T) {
	report := &PerfReport{
		Targets: &ValidationReport{
			Results: []ValidationResult{
				{Target: "slow", Passed: false, Margin: -0.5, Actual: 100 * time.Millisecond},
			},
			AllPassed: false,
		},
		Regressions: []Regression{
			{Name: "BenchA", PercentChange: 25.0}, // 25% > 15% threshold
		},
		Memory: &MemProfile{
			Snapshots: []MemSnapshot{
				{HeapAlloc: 1000000},
				{HeapAlloc: 1500000}, // 50% growth > 20% threshold
			},
		},
		Soak: &SoakResult{
			Iterations:  100,
			Errors:      5, // 5% > 1% threshold
			P50:         time.Millisecond,
			P99:         5 * time.Millisecond, // 5x > 2x threshold
		},
	}

	violations := pvApplyThresholds(report, DefaultThresholds())
	if len(violations) < 4 {
		t.Errorf("expected at least 4 violations, got %d: %v", len(violations), violations)
	}
}

func TestApplyThresholdsNilReport(t *testing.T) {
	violations := pvApplyThresholds(nil, DefaultThresholds())
	if len(violations) != 0 {
		t.Errorf("expected 0 violations for nil report, got %d", len(violations))
	}
}

func TestCIGatePass(t *testing.T) {
	pass, msg := pvGateCI(nil)
	if !pass {
		t.Error("expected CI gate to pass with no violations")
	}
	if !strings.Contains(msg, "PASS") {
		t.Errorf("expected PASS in message, got %q", msg)
	}
}

func TestCIGateFail(t *testing.T) {
	violations := []string{"target exceeded", "regression found"}
	pass, msg := pvGateCI(violations)
	if pass {
		t.Error("expected CI gate to fail with violations")
	}
	if !strings.Contains(msg, "FAIL") {
		t.Errorf("expected FAIL in message, got %q", msg)
	}
	if !strings.Contains(msg, "2 violation") {
		t.Errorf("expected violation count in message, got %q", msg)
	}
}

func TestCustomThresholds(t *testing.T) {
	// Very permissive thresholds.
	th := &ThresholdConfig{
		RegressionThreshold: 0.50,
		LeakGrowthMax:       0.80,
		SoakErrorRateMax:    0.10,
		P99Multiplier:       5.0,
	}

	report := &PerfReport{
		Regressions: []Regression{
			{Name: "A", PercentChange: 30.0}, // 30% < 50% threshold
		},
		Memory: &MemProfile{
			Snapshots: []MemSnapshot{
				{HeapAlloc: 1000000},
				{HeapAlloc: 1500000}, // 50% growth < 80% threshold
			},
		},
		Soak: &SoakResult{
			Iterations: 100,
			Errors:     5, // 5% < 10% threshold
			P50:        time.Millisecond,
			P99:        3 * time.Millisecond, // 3x < 5x threshold
		},
	}

	violations := pvApplyThresholds(report, th)
	if len(violations) != 0 {
		t.Errorf("expected 0 violations with permissive thresholds, got %d: %v", len(violations), violations)
	}
}

// ---------------------------------------------------------------------------
// Edge case tests
// ---------------------------------------------------------------------------

func TestP95EmptySlice(t *testing.T) {
	result := pvP95(nil)
	if result != 0 {
		t.Errorf("expected 0 for empty slice, got %v", result)
	}
}

func TestP95SingleValue(t *testing.T) {
	result := pvP95([]time.Duration{42 * time.Millisecond})
	if result != 42*time.Millisecond {
		t.Errorf("expected 42ms for single value, got %v", result)
	}
}

func TestAnalyzeGrowthSingleSnapshot(t *testing.T) {
	slope := pvAnalyzeGrowth([]MemSnapshot{
		{Timestamp: time.Now(), HeapAlloc: 1000000},
	})
	if slope != 0 {
		t.Errorf("expected 0 slope for single snapshot, got %f", slope)
	}
}

func TestFormatNs(t *testing.T) {
	tests := []struct {
		ns       int64
		contains string
	}{
		{500, "500ns"},
		{1500, "us"},
		{1500000, "ms"},
		{1500000000, "s"},
	}

	for _, tc := range tests {
		result := pvFormatNs(tc.ns)
		if !strings.Contains(result, tc.contains) {
			t.Errorf("pvFormatNs(%d) = %q, should contain %q", tc.ns, result, tc.contains)
		}
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		d        time.Duration
		contains string
	}{
		{500 * time.Nanosecond, "ns"},
		{1500 * time.Microsecond, "ms"},
		{1500 * time.Millisecond, "s"},
	}

	for _, tc := range tests {
		result := pvFormatDuration(tc.d)
		if !strings.Contains(result, tc.contains) {
			t.Errorf("pvFormatDuration(%v) = %q, should contain %q", tc.d, result, tc.contains)
		}
	}
}

func TestIsStableNoIterations(t *testing.T) {
	result := &SoakResult{Iterations: 0}
	if !pvIsStable(result) {
		t.Error("zero iterations should be stable")
	}
}

func TestIsStableTailLatency(t *testing.T) {
	result := &SoakResult{
		Iterations: 100,
		Errors:     0,
		P50:        time.Millisecond,
		P99:        10 * time.Millisecond, // 10x P50
	}
	if pvIsStable(result) {
		t.Error("should be unstable with P99 = 10x P50")
	}
}

func TestAbsFloat64(t *testing.T) {
	if pvAbsFloat64(-5.0) != 5.0 {
		t.Error("abs(-5) should be 5")
	}
	if pvAbsFloat64(5.0) != 5.0 {
		t.Error("abs(5) should be 5")
	}
	if pvAbsFloat64(0) != 0 {
		t.Error("abs(0) should be 0")
	}
}

func TestPercentileIndexBounds(t *testing.T) {
	// Should not panic or return out-of-bounds.
	idx := pvPercentileIndex(1, 0.99)
	if idx != 0 {
		t.Errorf("expected index 0 for n=1, got %d", idx)
	}

	idx = pvPercentileIndex(100, 0.0)
	if idx != 0 {
		t.Errorf("expected index 0 for p=0.0, got %d", idx)
	}

	idx = pvPercentileIndex(100, 1.0)
	if idx != 99 {
		t.Errorf("expected index 99 for p=1.0, got %d", idx)
	}
}

func TestParseBenchLineMinimalFields(t *testing.T) {
	result, err := pvParseBenchLine("BenchmarkX-4 1000 500 ns/op")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Name != "BenchmarkX-4" {
		t.Errorf("expected name BenchmarkX-4, got %q", result.Name)
	}
	if result.Iterations != 1000 {
		t.Errorf("expected 1000 iterations, got %d", result.Iterations)
	}
	if result.NsPerOp != 500 {
		t.Errorf("expected 500 ns/op, got %d", result.NsPerOp)
	}
}

func TestReportExecutiveSummarySections(t *testing.T) {
	// Test report with memory leak and regression.
	report := &PerfReport{
		Targets: &ValidationReport{
			Results: []ValidationResult{
				{Target: "test", Passed: false, Margin: -0.5, Actual: 100 * time.Millisecond},
			},
		},
		Memory: &MemProfile{
			LeakDetected: true,
			Snapshots: []MemSnapshot{
				{HeapAlloc: 1000000, Timestamp: time.Now()},
				{HeapAlloc: 2000000, Timestamp: time.Now().Add(time.Second)},
			},
			MaxHeap: 2000000,
		},
		Regressions: []Regression{
			{Name: "A", IsRegression: true, PercentChange: 25.0, CurrentNs: 1250, BaselineNs: 1000},
		},
		Platform: *pvDetectPlatform(),
	}

	output, err := GenerateReport(report)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(output, "FAIL") {
		t.Error("report with failures should contain FAIL")
	}
	if !strings.Contains(output, "LEAK DETECTED") {
		t.Error("report with leak should contain LEAK DETECTED")
	}
}
