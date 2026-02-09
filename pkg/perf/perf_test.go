package perf

import (
	"image/color"
	"strings"
	"testing"
	"time"
)

// --- StringPool tests -------------------------------------------------------

func TestStringPoolGetPutReusable(t *testing.T) {
	pool := NewStringPool()

	b1 := pool.Get()
	b1.WriteString("hello")
	pool.Put(b1)

	b2 := pool.Get()
	// After Put+Get, the builder should be empty (Reset was called).
	if b2.Len() != 0 {
		t.Errorf("StringPool.Get after Put: got len=%d, want 0", b2.Len())
	}
}

func TestStringPoolPutResetsBuilder(t *testing.T) {
	pool := NewStringPool()

	b := pool.Get()
	b.WriteString("some content that should be cleared")
	if b.Len() == 0 {
		t.Fatal("builder should have content before Put")
	}

	pool.Put(b)
	b2 := pool.Get()
	if b2.Len() != 0 {
		t.Errorf("builder not reset after Put: len=%d", b2.Len())
	}
}

// --- PreallocBuilder tests --------------------------------------------------

func TestPreallocBuilderCapacity(t *testing.T) {
	b := PreallocBuilder(256)
	if b.Cap() < 256 {
		t.Errorf("PreallocBuilder(256): cap=%d, want >= 256", b.Cap())
	}
}

func TestPreallocBuilderUsable(t *testing.T) {
	b := PreallocBuilder(64)
	b.WriteString("test")
	if b.String() != "test" {
		t.Errorf("PreallocBuilder(64) write failed: got %q", b.String())
	}
}

func TestPreallocBuilderZeroCapacity(t *testing.T) {
	b := PreallocBuilder(0)
	// Should not panic, builder is functional.
	b.WriteString("test")
	if b.String() != "test" {
		t.Errorf("PreallocBuilder(0) write failed: got %q", b.String())
	}
}

func TestPreallocBuilderNegativeCapacity(t *testing.T) {
	// Negative capacity should be clamped to 0, not panic.
	b := PreallocBuilder(-10)
	b.WriteString("ok")
	if b.String() != "ok" {
		t.Errorf("PreallocBuilder(-10) write failed: got %q", b.String())
	}
}

// --- BatchRender tests ------------------------------------------------------

func TestBatchRender3Tasks(t *testing.T) {
	tasks := []WidgetRenderTask{
		{Render: func(w, h int) string { return "A" }, Width: 10, Height: 5},
		{Render: func(w, h int) string { return "B" }, Width: 20, Height: 10},
		{Render: func(w, h int) string { return "C" }, Width: 15, Height: 8},
	}

	results := BatchRender(tasks, 4)
	if len(results) != 3 {
		t.Fatalf("BatchRender: got %d results, want 3", len(results))
	}
	if results[0] != "A" || results[1] != "B" || results[2] != "C" {
		t.Errorf("BatchRender results = %v, want [A B C]", results)
	}
}

func TestBatchRender0Tasks(t *testing.T) {
	results := BatchRender(nil, 4)
	if len(results) != 0 {
		t.Errorf("BatchRender(nil): got %d results, want 0", len(results))
	}

	results = BatchRender([]WidgetRenderTask{}, 4)
	if len(results) != 0 {
		t.Errorf("BatchRender(empty): got %d results, want 0", len(results))
	}
}

func TestBatchRender1Worker(t *testing.T) {
	// Verify serial execution with 1 worker produces correct results.
	order := make([]int, 0, 3)
	tasks := []WidgetRenderTask{
		{Render: func(w, h int) string { order = append(order, 0); return "first" }, Width: 1, Height: 1},
		{Render: func(w, h int) string { order = append(order, 1); return "second" }, Width: 1, Height: 1},
		{Render: func(w, h int) string { order = append(order, 2); return "third" }, Width: 1, Height: 1},
	}

	results := BatchRender(tasks, 1)
	if len(results) != 3 {
		t.Fatalf("BatchRender: got %d results, want 3", len(results))
	}

	// With 1 worker (serial), execution order should be sequential.
	if len(order) != 3 || order[0] != 0 || order[1] != 1 || order[2] != 2 {
		t.Errorf("serial execution order = %v, want [0 1 2]", order)
	}

	if results[0] != "first" || results[1] != "second" || results[2] != "third" {
		t.Errorf("results = %v, want [first second third]", results)
	}
}

func TestBatchRenderResultsMatchContent(t *testing.T) {
	tasks := make([]WidgetRenderTask, 10)
	for i := range tasks {
		idx := i
		tasks[i] = WidgetRenderTask{
			Render: func(w, h int) string {
				return strings.Repeat("x", idx+1)
			},
			Width:  idx + 5,
			Height: idx + 3,
		}
	}

	results := BatchRender(tasks, 3)
	for i, r := range results {
		expected := strings.Repeat("x", i+1)
		if r != expected {
			t.Errorf("results[%d] = %q, want %q", i, r, expected)
		}
	}
}

func TestBatchRenderPanicRecovery(t *testing.T) {
	tasks := []WidgetRenderTask{
		{Render: func(w, h int) string { return "ok" }, Width: 10, Height: 5},
		{Render: func(w, h int) string { panic("boom") }, Width: 10, Height: 5},
		{Render: func(w, h int) string { return "also ok" }, Width: 10, Height: 5},
	}

	results := BatchRender(tasks, 2)
	if len(results) != 3 {
		t.Fatalf("BatchRender: got %d results, want 3", len(results))
	}

	if results[0] != "ok" {
		t.Errorf("results[0] = %q, want %q", results[0], "ok")
	}
	if !strings.Contains(results[1], "panic") {
		t.Errorf("results[1] = %q, want to contain 'panic'", results[1])
	}
	if results[2] != "also ok" {
		t.Errorf("results[2] = %q, want %q", results[2], "also ok")
	}
}

func TestBatchRenderNilRenderFunc(t *testing.T) {
	tasks := []WidgetRenderTask{
		{Render: nil, Width: 10, Height: 5},
	}

	results := BatchRender(tasks, 1)
	if len(results) != 1 {
		t.Fatalf("BatchRender: got %d results, want 1", len(results))
	}
	if !strings.Contains(results[0], "nil") {
		t.Errorf("results[0] = %q, want to contain 'nil'", results[0])
	}
}

// --- DefaultThresholds tests ------------------------------------------------

func TestDefaultThresholdsNonEmpty(t *testing.T) {
	thresholds := DefaultThresholds()
	if len(thresholds) == 0 {
		t.Error("DefaultThresholds() returned empty slice")
	}
}

func TestDefaultThresholdsAllPositiveMaxNs(t *testing.T) {
	for _, th := range DefaultThresholds() {
		if th.MaxNs <= 0 {
			t.Errorf("threshold %q has non-positive MaxNs=%d", th.Name, th.MaxNs)
		}
	}
}

func TestDefaultThresholdNamesUnique(t *testing.T) {
	seen := make(map[string]bool)
	for _, th := range DefaultThresholds() {
		if seen[th.Name] {
			t.Errorf("duplicate threshold name: %q", th.Name)
		}
		seen[th.Name] = true
	}
}

// --- CheckRegression tests --------------------------------------------------

func TestCheckRegressionPassing(t *testing.T) {
	thresholds := []Threshold{
		{Name: "fast_op", MaxNs: 1_000_000, MaxAlloc: 1024},
	}

	// Result well within budget: 100ns/op, 64 bytes/op.
	results := []testing.BenchmarkResult{
		{N: 1000, T: 100 * time.Microsecond, MemBytes: 64000},
	}

	violations := CheckRegression(results, thresholds)
	if len(violations) != 0 {
		t.Errorf("expected no violations, got %d: %+v", len(violations), violations)
	}
}

func TestCheckRegressionNsViolation(t *testing.T) {
	thresholds := []Threshold{
		{Name: "slow_op", MaxNs: 1_000, MaxAlloc: 0}, // 1us budget
	}

	// Result exceeds time budget: 10ms total / 1 iteration = 10ms/op > 1us.
	results := []testing.BenchmarkResult{
		{N: 1, T: 10 * time.Millisecond},
	}

	violations := CheckRegression(results, thresholds)
	if len(violations) != 1 {
		t.Fatalf("expected 1 violation, got %d", len(violations))
	}
	if violations[0].Field != "ns" {
		t.Errorf("violation field = %q, want 'ns'", violations[0].Field)
	}
	if violations[0].Threshold.Name != "slow_op" {
		t.Errorf("violation name = %q, want 'slow_op'", violations[0].Threshold.Name)
	}
}

func TestCheckRegressionAllocViolation(t *testing.T) {
	thresholds := []Threshold{
		{Name: "alloc_op", MaxNs: 0, MaxAlloc: 100}, // 100 bytes budget
	}

	// Result exceeds alloc budget: 1000 bytes / 1 iteration = 1000 bytes/op > 100.
	results := []testing.BenchmarkResult{
		{N: 1, T: time.Nanosecond, MemBytes: 1000},
	}

	violations := CheckRegression(results, thresholds)
	if len(violations) != 1 {
		t.Fatalf("expected 1 violation, got %d", len(violations))
	}
	if violations[0].Field != "alloc" {
		t.Errorf("violation field = %q, want 'alloc'", violations[0].Field)
	}
}

func TestCheckRegressionEmptyInputs(t *testing.T) {
	if v := CheckRegression(nil, DefaultThresholds()); v != nil {
		t.Errorf("expected nil for nil results, got %v", v)
	}
	if v := CheckRegression([]testing.BenchmarkResult{{N: 1, T: time.Second}}, nil); v != nil {
		t.Errorf("expected nil for nil thresholds, got %v", v)
	}
}

// --- pfMakeBannerData tests -------------------------------------------------

func TestPfMakeBannerDataReturns6Widgets(t *testing.T) {
	data := pfMakeBannerData()
	if len(data.Widgets) != 6 {
		t.Errorf("pfMakeBannerData: got %d widgets, want 6", len(data.Widgets))
	}
}

func TestPfMakeBannerDataWidgetsHaveContent(t *testing.T) {
	data := pfMakeBannerData()
	for i, w := range data.Widgets {
		if w.ID == "" {
			t.Errorf("widget[%d].ID is empty", i)
		}
		if w.Title == "" {
			t.Errorf("widget[%d].Title is empty", i)
		}
		if w.Content == "" {
			t.Errorf("widget[%d].Content is empty", i)
		}
		if w.MinW <= 0 {
			t.Errorf("widget[%d].MinW = %d, want > 0", i, w.MinW)
		}
		if w.MinH <= 0 {
			t.Errorf("widget[%d].MinH = %d, want > 0", i, w.MinH)
		}
	}
}

// --- pfMakeTestImage tests --------------------------------------------------

func TestPfMakeTestImageDimensions(t *testing.T) {
	tests := []struct {
		w, h int
	}{
		{100, 50},
		{1, 1},
		{800, 600},
	}

	for _, tc := range tests {
		img := pfMakeTestImage(tc.w, tc.h)
		bounds := img.Bounds()
		if bounds.Dx() != tc.w || bounds.Dy() != tc.h {
			t.Errorf("pfMakeTestImage(%d, %d): got %dx%d", tc.w, tc.h, bounds.Dx(), bounds.Dy())
		}
	}
}

func TestPfMakeTestImageZeroDimensions(t *testing.T) {
	// Zero dimensions should be clamped to 1x1.
	img := pfMakeTestImage(0, 0)
	bounds := img.Bounds()
	if bounds.Dx() != 1 || bounds.Dy() != 1 {
		t.Errorf("pfMakeTestImage(0, 0): got %dx%d, want 1x1", bounds.Dx(), bounds.Dy())
	}
}

func TestPfMakeTestImageHasGradient(t *testing.T) {
	img := pfMakeTestImage(10, 10)
	// Check that not all pixels are the same color (gradient should vary).
	first := img.At(0, 0)
	allSame := true
	bounds := img.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			if img.At(x, y) != first {
				allSame = false
				break
			}
		}
		if !allSame {
			break
		}
	}
	if allSame {
		t.Error("pfMakeTestImage: all pixels are the same, expected gradient")
	}
}

// --- Violation struct tests -------------------------------------------------

func TestViolationStructFields(t *testing.T) {
	v := Violation{
		Threshold: Threshold{Name: "test_op", MaxNs: 1000, MaxAlloc: 512},
		Actual:    2000,
		Field:     "ns",
	}

	if v.Threshold.Name != "test_op" {
		t.Errorf("Violation.Threshold.Name = %q, want 'test_op'", v.Threshold.Name)
	}
	if v.Actual != 2000 {
		t.Errorf("Violation.Actual = %d, want 2000", v.Actual)
	}
	if v.Field != "ns" {
		t.Errorf("Violation.Field = %q, want 'ns'", v.Field)
	}
}

// --- Benchmark smoke tests (verify they run without panic) ------------------

func TestBenchmarkBannerRenderCompactSmoke(t *testing.T) {
	result := testing.Benchmark(BenchmarkBannerRenderCompact)
	if result.N == 0 {
		t.Error("BenchmarkBannerRenderCompact did not run")
	}
}

func TestBenchmarkBannerRenderStandardSmoke(t *testing.T) {
	result := testing.Benchmark(BenchmarkBannerRenderStandard)
	if result.N == 0 {
		t.Error("BenchmarkBannerRenderStandard did not run")
	}
}

func TestBenchmarkLayoutSolve6Smoke(t *testing.T) {
	result := testing.Benchmark(BenchmarkLayoutSolve6Widgets)
	if result.N == 0 {
		t.Error("BenchmarkLayoutSolve6Widgets did not run")
	}
}

func TestBenchmarkGaugeRenderSmoke(t *testing.T) {
	result := testing.Benchmark(BenchmarkGaugeRender)
	if result.N == 0 {
		t.Error("BenchmarkGaugeRender did not run")
	}
}

func TestBenchmarkSparklineRenderSmoke(t *testing.T) {
	result := testing.Benchmark(BenchmarkSparklineRender)
	if result.N == 0 {
		t.Error("BenchmarkSparklineRender did not run")
	}
}

func TestBenchmarkBoxRenderSmoke(t *testing.T) {
	result := testing.Benchmark(BenchmarkBoxRender)
	if result.N == 0 {
		t.Error("BenchmarkBoxRender did not run")
	}
}

func TestBenchmarkShellGenerateBashSmoke(t *testing.T) {
	result := testing.Benchmark(BenchmarkShellGenerateBash)
	if result.N == 0 {
		t.Error("BenchmarkShellGenerateBash did not run")
	}
}

func TestBenchmarkShellGenerateZshSmoke(t *testing.T) {
	result := testing.Benchmark(BenchmarkShellGenerateZsh)
	if result.N == 0 {
		t.Error("BenchmarkShellGenerateZsh did not run")
	}
}

func TestBenchmarkSelectPresetSmoke(t *testing.T) {
	result := testing.Benchmark(BenchmarkBannerSelectPreset)
	if result.N == 0 {
		t.Error("BenchmarkBannerSelectPreset did not run")
	}
}

// --- pfMakeSolidImage test --------------------------------------------------

func TestPfMakeSolidImage(t *testing.T) {
	img := pfMakeSolidImage(10, 10, color.NRGBA{R: 255, G: 0, B: 0, A: 255})
	bounds := img.Bounds()
	if bounds.Dx() != 10 || bounds.Dy() != 10 {
		t.Errorf("pfMakeSolidImage: got %dx%d, want 10x10", bounds.Dx(), bounds.Dy())
	}

	// Verify center pixel is red.
	r, g, b, a := img.At(5, 5).RGBA()
	if r>>8 != 255 || g>>8 != 0 || b>>8 != 0 || a>>8 != 255 {
		t.Errorf("solid image pixel at (5,5) = (%d,%d,%d,%d), want (255,0,0,255)",
			r>>8, g>>8, b>>8, a>>8)
	}
}
