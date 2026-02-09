package widgets

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"gitlab.com/tinyland/lab/prompt-pulse/pkg/app"
	"gitlab.com/tinyland/lab/prompt-pulse/pkg/collectors/billing"
	"gitlab.com/tinyland/lab/prompt-pulse/pkg/components"
)

// Billing color constants.
const (
	billingColorGreen  = "#4CAF50"
	billingColorYellow = "#FF9800"
	billingColorRed    = "#F44336"
	billingColorBlue   = "#64B5F6"
)

// BillingWidget displays cloud billing data including budget usage, provider
// breakdowns, resource costs, and monthly spend trends.
type BillingWidget struct {
	report           *billing.BillingReport
	expanded         bool
	costHistory      []float64
	selectedProvider int
}

// NewBillingWidget creates a new BillingWidget with default state.
func NewBillingWidget() *BillingWidget {
	return &BillingWidget{}
}

// ID returns the unique identifier for this widget.
func (w *BillingWidget) ID() string {
	return "billing"
}

// Title returns the display name for this widget.
func (w *BillingWidget) Title() string {
	return "Billing"
}

// MinSize returns the minimum width and height this widget requires.
func (w *BillingWidget) MinSize() (int, int) {
	return 30, 4
}

// Update handles messages directed at this widget. It processes
// DataUpdateEvent messages with Source "billing".
func (w *BillingWidget) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case app.DataUpdateEvent:
		if msg.Source != "billing" {
			return nil
		}
		if msg.Err != nil {
			return nil
		}
		report, ok := msg.Data.(*billing.BillingReport)
		if !ok {
			return nil
		}
		w.report = report
		w.costHistory = append(w.costHistory, report.TotalMonthlyUSD)
	}
	return nil
}

// HandleKey processes key events when this widget has focus.
// 'e' toggles expanded mode, 'p' cycles through providers.
func (w *BillingWidget) HandleKey(key tea.KeyMsg) tea.Cmd {
	switch key.String() {
	case "e":
		w.expanded = !w.expanded
		return nil
	case "p":
		if w.report != nil && len(w.report.Providers) > 0 {
			w.selectedProvider = (w.selectedProvider + 1) % len(w.report.Providers)
		}
		return nil
	}
	return nil
}

// View renders the widget content into the given area dimensions.
func (w *BillingWidget) View(width, height int) string {
	if width <= 0 || height <= 0 {
		return ""
	}

	if w.report == nil {
		return billingNoData(width, height)
	}

	if w.expanded {
		return w.billingViewExpanded(width, height)
	}
	return w.billingViewCompact(width, height)
}

// billingNoData renders a placeholder when no billing data is available.
func billingNoData(width, height int) string {
	msg := "No data"
	lines := make([]string, 0, height)
	topPad := (height - 1) / 2
	for i := 0; i < topPad; i++ {
		lines = append(lines, "")
	}
	if len(msg) > width {
		msg = msg[:width]
	}
	lines = append(lines, msg)
	for len(lines) < height {
		lines = append(lines, "")
	}
	if len(lines) > height {
		lines = lines[:height]
	}
	return strings.Join(lines, "\n")
}

// billingViewCompact renders the compact view: budget gauge, total cost,
// and a one-line summary per provider.
func (w *BillingWidget) billingViewCompact(width, height int) string {
	lines := make([]string, 0, height)

	// Budget gauge line.
	if w.report.BudgetUSD > 0 {
		gaugeLine := w.billingRenderBudgetGauge(width)
		lines = append(lines, gaugeLine)
	}

	// Total spend line.
	totalLine := fmt.Sprintf("Total: $%.2f", w.report.TotalMonthlyUSD)
	if w.report.BudgetUSD > 0 {
		totalLine += fmt.Sprintf(" / $%.2f budget", w.report.BudgetUSD)
	}
	if len(totalLine) > width {
		totalLine = totalLine[:width]
	}
	lines = append(lines, totalLine)

	// Provider summary lines.
	for _, p := range w.report.Providers {
		dot := billingStatusDot(p.Connected)
		provLine := fmt.Sprintf("%s %s: $%.2f", dot, p.Name, p.MonthToDate)
		if len(provLine) > width {
			provLine = provLine[:width]
		}
		lines = append(lines, provLine)
		if len(lines) >= height {
			break
		}
	}

	// Fill remaining height.
	for len(lines) < height {
		lines = append(lines, "")
	}
	if len(lines) > height {
		lines = lines[:height]
	}
	return strings.Join(lines, "\n")
}

// billingViewExpanded renders the expanded view with full resource tables,
// sparkline, and projected cost.
func (w *BillingWidget) billingViewExpanded(width, height int) string {
	lines := make([]string, 0, height)

	// Budget gauge at top.
	if w.report.BudgetUSD > 0 {
		gaugeLine := w.billingRenderBudgetGauge(width)
		lines = append(lines, gaugeLine)
	}

	// Provider sections.
	for i, p := range w.report.Providers {
		if len(lines) >= height {
			break
		}

		// Provider header.
		dot := billingStatusDot(p.Connected)
		header := fmt.Sprintf("%s %s  MTD: $%.2f", dot, components.Bold(p.Name), p.MonthToDate)
		lines = append(lines, header)

		// Resource table (only for selected provider or all if there is room).
		if len(p.Resources) > 0 && (i == w.selectedProvider || len(w.report.Providers) == 1) {
			tableLines := w.billingRenderResourceTable(p.Resources, width, height-len(lines)-3)
			lines = append(lines, tableLines...)
		}
	}

	// Sparkline of cost history.
	if len(w.costHistory) > 0 && len(lines) < height-2 {
		sparkStyle := components.SparklineStyle{
			Width:      width - 8,
			Color:      billingColorBlue,
			ShowMinMax: true,
		}
		spark := components.NewSparkline(sparkStyle)
		sparkLine := "Trend: " + spark.Render(w.costHistory, width-8)
		lines = append(lines, sparkLine)
	}

	// Projected cost.
	if len(lines) < height {
		projected := billingProjectedCost(w.report.TotalMonthlyUSD)
		projLine := fmt.Sprintf("Projected: $%.2f", projected)
		lines = append(lines, projLine)
	}

	// Total.
	if len(lines) < height {
		totalLine := fmt.Sprintf("Total MTD: $%.2f", w.report.TotalMonthlyUSD)
		lines = append(lines, totalLine)
	}

	// Fill remaining height.
	for len(lines) < height {
		lines = append(lines, "")
	}
	if len(lines) > height {
		lines = lines[:height]
	}
	return strings.Join(lines, "\n")
}

// billingRenderBudgetGauge renders the budget gauge bar using the components
// Gauge with threshold-based coloring.
func (w *BillingWidget) billingRenderBudgetGauge(width int) string {
	gaugeStyle := components.GaugeStyle{
		Width:             width - 15,
		ShowPercent:       true,
		ShowValue:         true,
		Label:             "Budget",
		LabelWidth:        8,
		FilledColor:       billingColorGreen,
		EmptyColor:        "#333333",
		WarningThreshold:  0.60,
		CriticalThreshold: 0.85,
		WarningColor:      billingColorYellow,
		CriticalColor:     billingColorRed,
	}
	if gaugeStyle.Width < 5 {
		gaugeStyle.Width = 5
	}
	gauge := components.NewGauge(gaugeStyle)
	return gauge.Render(w.report.TotalMonthlyUSD, w.report.BudgetUSD, gaugeStyle.Width)
}

// billingRenderResourceTable renders a resource cost table for a provider's
// resources. Returns a slice of lines.
func (w *BillingWidget) billingRenderResourceTable(resources []billing.ResourceCost, width, maxLines int) []string {
	if maxLines <= 0 {
		maxLines = 5
	}

	dt := components.NewDataTable(components.DataTableConfig{
		Columns: []components.Column{
			{Title: "Name", Sizing: components.SizingFill(), Align: components.ColAlignLeft, MinWidth: 8},
			{Title: "Type", Sizing: components.SizingFixed(12), Align: components.ColAlignLeft},
			{Title: "Cost", Sizing: components.SizingFixed(10), Align: components.ColAlignRight},
		},
		HeaderStyle: components.HeaderStyleConfig{
			Bold:    true,
			FgColor: ColorAccent,
		},
		ShowHeader: true,
		ShowBorder: true,
	})

	rows := make([]components.Row, 0, len(resources))
	for _, r := range resources {
		rows = append(rows, components.Row{
			Cells: []string{r.Name, r.Type, fmt.Sprintf("$%.2f", r.MonthlyCost)},
		})
	}
	dt.SetRows(rows)

	// Render table: header (2 lines) + data rows.
	tableHeight := len(resources) + 2
	if tableHeight > maxLines {
		tableHeight = maxLines
	}
	if tableHeight < 3 {
		tableHeight = 3
	}
	rendered := dt.Render(width, tableHeight)
	return strings.Split(rendered, "\n")
}

// billingStatusDot returns a colored status indicator dot.
// Green for connected, red for disconnected.
func billingStatusDot(connected bool) string {
	if connected {
		return components.Color(billingColorGreen) + "\u25cf" + components.Reset()
	}
	return components.Color(billingColorRed) + "\u25cf" + components.Reset()
}

// billingProjectedCost computes a linear extrapolation of month-end cost
// based on current spend and day of month.
func billingProjectedCost(currentSpend float64) float64 {
	return billingProjectedCostAt(currentSpend, time.Now())
}

// billingProjectedCostAt computes projected cost at a specific point in time.
// Exported for testing via the package-level function.
func billingProjectedCostAt(currentSpend float64, now time.Time) float64 {
	dayOfMonth := now.Day()
	if dayOfMonth <= 0 {
		dayOfMonth = 1
	}
	daysInMonth := billingDaysInMonth(now)
	return (currentSpend / float64(dayOfMonth)) * float64(daysInMonth)
}

// billingDaysInMonth returns the number of days in the month of the given time.
func billingDaysInMonth(t time.Time) int {
	year, month, _ := t.Date()
	// First day of next month, then subtract one day.
	return time.Date(year, month+1, 0, 0, 0, 0, 0, t.Location()).Day()
}

// billingBudgetColor returns the appropriate color hex string based on
// the budget usage ratio (0.0 to 1.0+).
func billingBudgetColor(ratio float64) string {
	switch {
	case ratio >= 0.85:
		return billingColorRed
	case ratio >= 0.60:
		return billingColorYellow
	default:
		return billingColorGreen
	}
}
