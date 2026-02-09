package perfval

import (
	"fmt"
	"runtime"
	"strings"
	"time"
)

// PerfReport aggregates all performance validation data into a single
// structure suitable for rendering as a human-readable report.
type PerfReport struct {
	// Targets is the validation report for performance budget targets.
	Targets *ValidationReport

	// Soak is the result of a sustained load test.
	Soak *SoakResult

	// Memory is the memory profile collected during profiling.
	Memory *MemProfile

	// Regressions is the list of benchmark comparisons.
	Regressions []Regression

	// Platform describes the system where the report was generated.
	Platform PlatformInfo
}

// PlatformInfo captures the runtime environment for reproducibility and
// comparison across different machines.
type PlatformInfo struct {
	// OS is the operating system (e.g., "darwin", "linux").
	OS string

	// Arch is the CPU architecture (e.g., "arm64", "amd64").
	Arch string

	// GoVersion is the Go runtime version (e.g., "go1.23.0").
	GoVersion string

	// CPUModel is a descriptive name for the CPU, if available.
	CPUModel string

	// NumCPU is the number of logical CPUs available.
	NumCPU int

	// TotalMemory is the total system memory in bytes (0 if unknown).
	TotalMemory uint64
}

// pvDetectPlatform reads runtime environment information to populate a
// PlatformInfo struct. CPU model and total memory are best-effort; they
// default to empty/zero on platforms where detection is not implemented.
func pvDetectPlatform() *PlatformInfo {
	return &PlatformInfo{
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
		GoVersion: runtime.Version(),
		CPUModel:  "unknown",
		NumCPU:    runtime.NumCPU(),
	}
}

// GenerateReport renders a complete Markdown performance report from a
// PerfReport structure. The report includes sections for executive summary,
// target validation, soak test results, memory analysis, regression
// analysis, and platform information.
func GenerateReport(report *PerfReport) (string, error) {
	if report == nil {
		return "", fmt.Errorf("report must not be nil")
	}

	var sb strings.Builder

	// Header
	sb.WriteString("# Performance Validation Report\n\n")
	sb.WriteString(fmt.Sprintf("Generated: %s\n\n", time.Now().Format(time.RFC3339)))

	// Executive Summary
	sb.WriteString("## Executive Summary\n\n")
	pvWriteExecutiveSummary(&sb, report)

	// Target Validation
	if report.Targets != nil && len(report.Targets.Results) > 0 {
		sb.WriteString("## Target Validation\n\n")
		sb.WriteString(pvRenderTargetTable(report.Targets.Results))
		sb.WriteString("\n")
	}

	// Soak Test Results
	if report.Soak != nil {
		sb.WriteString("## Soak Test Results\n\n")
		pvWriteSoakSection(&sb, report.Soak)
	}

	// Memory Analysis
	if report.Memory != nil && len(report.Memory.Snapshots) > 0 {
		sb.WriteString("## Memory Analysis\n\n")
		pvWriteMemorySection(&sb, report.Memory)
		sb.WriteString("\n### Heap Trend\n\n")
		sb.WriteString("```\n")
		sb.WriteString(pvRenderMemoryChart(report.Memory.Snapshots))
		sb.WriteString("```\n\n")
	}

	// Regression Analysis
	if len(report.Regressions) > 0 {
		sb.WriteString("## Regression Analysis\n\n")
		sb.WriteString(pvRenderComparison(report.Regressions))
		sb.WriteString("\n")
	}

	// Platform Info
	sb.WriteString("## Platform\n\n")
	pvWritePlatformSection(&sb, &report.Platform)

	return sb.String(), nil
}

// pvWriteExecutiveSummary writes a brief overview of the validation results.
func pvWriteExecutiveSummary(sb *strings.Builder, report *PerfReport) {
	passed := 0
	failed := 0

	if report.Targets != nil {
		for _, r := range report.Targets.Results {
			if r.Passed {
				passed++
			} else {
				failed++
			}
		}
	}

	if failed == 0 {
		sb.WriteString("**Status: PASS** - All performance targets met.\n\n")
	} else {
		sb.WriteString(fmt.Sprintf("**Status: FAIL** - %d of %d targets exceeded budget.\n\n",
			failed, passed+failed))
	}

	if report.Soak != nil {
		if report.Soak.Stable {
			sb.WriteString("- Soak test: STABLE\n")
		} else {
			sb.WriteString("- Soak test: UNSTABLE\n")
		}
	}

	if report.Memory != nil {
		if report.Memory.LeakDetected {
			sb.WriteString("- Memory: LEAK DETECTED\n")
		} else {
			sb.WriteString("- Memory: No leaks detected\n")
		}
	}

	regressionCount := 0
	for _, r := range report.Regressions {
		if r.IsRegression {
			regressionCount++
		}
	}
	if regressionCount > 0 {
		sb.WriteString(fmt.Sprintf("- Regressions: %d detected\n", regressionCount))
	} else if len(report.Regressions) > 0 {
		sb.WriteString("- Regressions: None detected\n")
	}

	sb.WriteString("\n")
}

// pvRenderTargetTable produces a Markdown table showing each target's result.
func pvRenderTargetTable(results []ValidationResult) string {
	var sb strings.Builder
	sb.WriteString("| Target | Budget | Actual (p95) | Margin | Status |\n")
	sb.WriteString("|--------|--------|--------------|--------|--------|\n")

	for _, r := range results {
		status := "PASS"
		if !r.Passed {
			status = "FAIL"
		}
		sb.WriteString(fmt.Sprintf("| %s | %s | %s | %+.1f%% | %s |\n",
			r.Target,
			pvFormatDuration(pvTargetBudget(r)),
			pvFormatDuration(r.Actual),
			r.Margin*100,
			status,
		))
	}

	return sb.String()
}

// pvTargetBudget looks up the budget for a validation result by matching
// against the default targets. Returns 0 if not found.
func pvTargetBudget(r ValidationResult) time.Duration {
	for _, t := range DefaultTargets() {
		if t.Name == r.Target {
			return t.MaxDuration
		}
	}
	// Derive from margin and actual: budget = actual / (1 - margin)
	if r.Margin < 1.0 {
		return time.Duration(float64(r.Actual) / (1.0 - r.Margin))
	}
	return 0
}

// pvFormatDuration formats a time.Duration into a concise human-readable string.
func pvFormatDuration(d time.Duration) string {
	if d < time.Microsecond {
		return fmt.Sprintf("%dns", d.Nanoseconds())
	}
	if d < time.Millisecond {
		return fmt.Sprintf("%.1fus", float64(d.Nanoseconds())/1e3)
	}
	if d < time.Second {
		return fmt.Sprintf("%.1fms", float64(d.Nanoseconds())/1e6)
	}
	return fmt.Sprintf("%.2fs", d.Seconds())
}

// pvWriteSoakSection writes the soak test results section.
func pvWriteSoakSection(sb *strings.Builder, soak *SoakResult) {
	sb.WriteString(fmt.Sprintf("- Iterations: %d\n", soak.Iterations))
	sb.WriteString(fmt.Sprintf("- Errors: %d", soak.Errors))
	if soak.Iterations > 0 {
		sb.WriteString(fmt.Sprintf(" (%.2f%%)", float64(soak.Errors)/float64(soak.Iterations)*100))
	}
	sb.WriteString("\n")
	sb.WriteString(fmt.Sprintf("- Avg: %s\n", pvFormatDuration(soak.AvgDuration)))
	sb.WriteString(fmt.Sprintf("- P50: %s\n", pvFormatDuration(soak.P50)))
	sb.WriteString(fmt.Sprintf("- P95: %s\n", pvFormatDuration(soak.P95)))
	sb.WriteString(fmt.Sprintf("- P99: %s\n", pvFormatDuration(soak.P99)))
	sb.WriteString(fmt.Sprintf("- Max: %s\n", pvFormatDuration(soak.MaxDuration)))
	if soak.Stable {
		sb.WriteString("- Stability: STABLE\n")
	} else {
		sb.WriteString("- Stability: UNSTABLE\n")
	}
	sb.WriteString("\n")
}

// pvWriteMemorySection writes the memory analysis section.
func pvWriteMemorySection(sb *strings.Builder, mem *MemProfile) {
	sb.WriteString(fmt.Sprintf("- Snapshots: %d\n", len(mem.Snapshots)))
	sb.WriteString(fmt.Sprintf("- Max Heap: %s\n", pvFormatBytes(mem.MaxHeap)))
	sb.WriteString(fmt.Sprintf("- Growth Rate: %.2f bytes/sec\n", mem.GrowthRate))
	if mem.LeakDetected {
		sb.WriteString("- Leak Status: DETECTED\n")
	} else {
		sb.WriteString("- Leak Status: None\n")
	}
}

// pvWritePlatformSection writes the platform information section.
func pvWritePlatformSection(sb *strings.Builder, platform *PlatformInfo) {
	sb.WriteString(fmt.Sprintf("- OS: %s\n", platform.OS))
	sb.WriteString(fmt.Sprintf("- Arch: %s\n", platform.Arch))
	sb.WriteString(fmt.Sprintf("- Go: %s\n", platform.GoVersion))
	sb.WriteString(fmt.Sprintf("- CPUs: %d\n", platform.NumCPU))
	if platform.CPUModel != "" && platform.CPUModel != "unknown" {
		sb.WriteString(fmt.Sprintf("- CPU Model: %s\n", platform.CPUModel))
	}
	if platform.TotalMemory > 0 {
		sb.WriteString(fmt.Sprintf("- Memory: %s\n", pvFormatBytes(platform.TotalMemory)))
	}
}

// pvRenderMemoryChart produces a simple ASCII chart showing heap allocation
// trends over time using block characters. The chart is 60 columns wide and
// 15 rows tall, with the Y-axis showing heap size and the X-axis showing
// relative time.
func pvRenderMemoryChart(snapshots []MemSnapshot) string {
	if len(snapshots) == 0 {
		return "(no data)\n"
	}

	const width = 60
	const height = 15

	// Find min and max heap values for Y-axis scaling.
	minHeap := snapshots[0].HeapAlloc
	maxHeap := snapshots[0].HeapAlloc
	for _, s := range snapshots {
		if s.HeapAlloc < minHeap {
			minHeap = s.HeapAlloc
		}
		if s.HeapAlloc > maxHeap {
			maxHeap = s.HeapAlloc
		}
	}

	heapRange := maxHeap - minHeap
	if heapRange == 0 {
		heapRange = 1 // avoid division by zero for flat profiles
	}

	// Downsample or upsample snapshots to fit the chart width.
	values := make([]float64, width)
	for i := 0; i < width; i++ {
		idx := i * (len(snapshots) - 1) / (width - 1)
		if idx >= len(snapshots) {
			idx = len(snapshots) - 1
		}
		values[i] = float64(snapshots[idx].HeapAlloc-minHeap) / float64(heapRange)
	}

	// Render the chart row by row from top to bottom.
	var sb strings.Builder
	blocks := []string{" ", "\u2581", "\u2582", "\u2583", "\u2584", "\u2585", "\u2586", "\u2587", "\u2588"}

	for row := height - 1; row >= 0; row-- {
		threshold := float64(row) / float64(height-1)

		// Y-axis label (only at top, middle, and bottom).
		switch row {
		case height - 1:
			sb.WriteString(fmt.Sprintf("%8s |", pvFormatBytes(maxHeap)))
		case height / 2:
			mid := minHeap + heapRange/2
			sb.WriteString(fmt.Sprintf("%8s |", pvFormatBytes(mid)))
		case 0:
			sb.WriteString(fmt.Sprintf("%8s |", pvFormatBytes(minHeap)))
		default:
			sb.WriteString("         |")
		}

		for _, v := range values {
			if v >= threshold {
				// Pick block character based on how far above threshold.
				excess := v - threshold
				blockIdx := int(excess * float64(height) * float64(len(blocks)-1))
				if blockIdx >= len(blocks) {
					blockIdx = len(blocks) - 1
				}
				if blockIdx < 1 {
					blockIdx = 1
				}
				sb.WriteString(blocks[blockIdx])
			} else {
				sb.WriteString(" ")
			}
		}
		sb.WriteString("\n")
	}

	// X-axis.
	sb.WriteString("         +")
	sb.WriteString(strings.Repeat("-", width))
	sb.WriteString("\n")

	return sb.String()
}
