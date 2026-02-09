package perf

import (
	"strings"
	"testing"

	"gitlab.com/tinyland/lab/prompt-pulse/pkg/components"
)

// BenchmarkGaugeRender benchmarks rendering a single gauge bar at 40 cells wide
// with ANSI color output.
func BenchmarkGaugeRender(b *testing.B) {
	style := components.DefaultGaugeStyle()
	style.ShowPercent = true
	style.ShowValue = true
	style.Label = "CPU"
	style.LabelWidth = 5
	g := components.NewGauge(style)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = g.Render(73.5, 100.0, 40)
	}
}

// BenchmarkSparklineRender benchmarks rendering a sparkline with 60 data points
// at 30 cells wide.
func BenchmarkSparklineRender(b *testing.B) {
	style := components.DefaultSparklineStyle()
	style.ShowMinMax = true
	style.Label = "Load"
	s := components.NewSparkline(style)

	// Generate realistic CPU load data (0-100 range).
	data := make([]float64, 60)
	for i := range data {
		data[i] = 30.0 + float64(i%40)*1.5
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = s.Render(data, 30)
	}
}

// BenchmarkBoxRender benchmarks rendering a bordered box with multi-line
// content at 40x10 dimensions.
func BenchmarkBoxRender(b *testing.B) {
	style := components.DefaultBoxStyle()
	style.Title = "System Metrics"
	style.Border = components.BorderRounded

	content := strings.Join([]string{
		"CPU:  62% \x1b[38;2;76;175;80m████████\x1b[0m",
		"RAM:  73% \x1b[38;2;255;152;0m██████████\x1b[0m",
		"Disk: 45% \x1b[38;2;76;175;80m██████\x1b[0m",
		"Net:  12Mbps up / 45Mbps down",
		"Uptime: 14d 7h 23m",
		"Load: 1.23 0.98 0.87",
	}, "\n")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = components.RenderBox(content, 40, 10, style)
	}
}

// BenchmarkDataTableRender benchmarks rendering a DataTable with 20 rows and
// 4 columns at 80x25 dimensions.
func BenchmarkDataTableRender(b *testing.B) {
	cols := []components.Column{
		{Title: "Name", Sizing: components.SizingPercent(30), Align: components.ColAlignLeft},
		{Title: "Status", Sizing: components.SizingFixed(10), Align: components.ColAlignCenter},
		{Title: "CPU%", Sizing: components.SizingFixed(8), Align: components.ColAlignRight},
		{Title: "Memory", Sizing: components.SizingFill(), Align: components.ColAlignRight},
	}

	rows := make([]components.Row, 20)
	for i := range rows {
		rows[i] = components.Row{
			Cells: []string{
				pfTableName(i),
				pfTableStatus(i),
				pfTableCPU(i),
				pfTableMem(i),
			},
		}
	}

	dt := components.NewDataTable(components.DataTableConfig{
		Columns:    cols,
		ShowHeader: true,
		ShowBorder: true,
		HeaderStyle: components.HeaderStyleConfig{
			Bold:    true,
			FgColor: "#FFFFFF",
			BgColor: "#333333",
		},
		RowStyle: components.RowStyleConfig{
			EvenBgColor: "#1a1a2e",
			OddBgColor:  "#16213e",
		},
	})
	dt.SetRows(rows)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = dt.Render(80, 25)
	}
}

// BenchmarkTextTruncate benchmarks ANSI-aware text truncation on strings
// containing ANSI escape sequences.
func BenchmarkTextTruncate(b *testing.B) {
	// Build a string with ANSI escapes interspersed.
	s := "\x1b[38;2;76;175;80mHello, World!\x1b[0m This is a \x1b[1mlonger\x1b[22m string with \x1b[38;2;255;152;0mcolored\x1b[0m text that needs truncation for display in narrow terminals."

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = components.Truncate(s, 40)
	}
}

// BenchmarkVisibleLen benchmarks ANSI-aware visible length calculation on
// strings containing ANSI escape sequences and Unicode characters.
func BenchmarkVisibleLen(b *testing.B) {
	// Mix of ANSI escapes, ASCII, and Unicode.
	s := "\x1b[38;2;76;175;80m████████\x1b[0m CPU: 62% \x1b[1m(healthy)\x1b[22m ▁▂▃▄▅▆▇█"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = components.VisibleLen(s)
	}
}

// pfTableName returns a realistic pod/service name for table benchmarks.
func pfTableName(i int) string {
	names := []string{
		"api-gateway", "auth-service", "billing-worker",
		"cache-redis", "db-postgres", "event-bus",
		"frontend-web", "grpc-proxy", "ingress-ctrl",
		"job-scheduler", "kafka-broker", "log-collector",
		"metrics-agent", "notification-svc", "order-processor",
		"payment-handler", "queue-worker", "rate-limiter",
		"search-indexer", "telemetry-agent",
	}
	return names[i%len(names)]
}

// pfTableStatus returns a realistic status value for table benchmarks.
func pfTableStatus(i int) string {
	statuses := []string{"Running", "Running", "Running", "Pending", "Running"}
	return statuses[i%len(statuses)]
}

// pfTableCPU returns a realistic CPU percentage string for table benchmarks.
func pfTableCPU(i int) string {
	cpus := []string{"12%", "45%", "3%", "67%", "22%", "89%", "5%", "34%", "56%", "15%"}
	return cpus[i%len(cpus)]
}

// pfTableMem returns a realistic memory usage string for table benchmarks.
func pfTableMem(i int) string {
	mems := []string{"128Mi", "256Mi", "64Mi", "512Mi", "192Mi", "1Gi", "96Mi", "384Mi", "448Mi", "160Mi"}
	return mems[i%len(mems)]
}
