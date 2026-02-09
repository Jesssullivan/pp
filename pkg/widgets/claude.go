package widgets

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"gitlab.com/tinyland/lab/prompt-pulse/pkg/app"
	"gitlab.com/tinyland/lab/prompt-pulse/pkg/collectors/claude"
	"gitlab.com/tinyland/lab/prompt-pulse/pkg/components"
)

// Budget constants used to compute gauge fill ratios. These represent
// "typical" monthly token limits and dollar budgets for a single account.
const (
	claudeDefaultTokenBudget  int64   = 10_000_000 // 10M tokens
	claudeDefaultCostBudget   float64 = 500.0      // $500 per month
	claudeMaxCostHistory      int     = 30         // keep last 30 data points
)

// Color thresholds for cost / usage gauges.
const (
	claudeThresholdGreen  = 0.50
	claudeThresholdYellow = 0.80
)

// Hex colors for the three threshold bands.
const (
	claudeColorGreen  = "#4CAF50"
	claudeColorYellow = "#FF9800"
	claudeColorRed    = "#F44336"
)

// ClaudeWidget displays multi-account Anthropic/Claude token usage,
// per-model breakdowns, cost gauges, and a sparkline cost trend.
type ClaudeWidget struct {
	report          *claude.UsageReport
	expanded        bool
	selectedAccount int
	costHistory     []float64
}

// NewClaudeWidget creates a new ClaudeWidget in compact mode.
func NewClaudeWidget() *ClaudeWidget {
	return &ClaudeWidget{}
}

// ID returns the widget's unique identifier.
func (w *ClaudeWidget) ID() string {
	return "claude"
}

// Title returns the widget's display title.
func (w *ClaudeWidget) Title() string {
	if w.expanded {
		return "Claude Usage"
	}
	return "Claude Usage"
}

// MinSize returns the minimum width and height for the widget.
// Compact mode needs less vertical space than expanded mode.
func (w *ClaudeWidget) MinSize() (int, int) {
	if w.expanded {
		return 30, 15
	}
	return 30, 5
}

// Update handles messages from the Elm architecture update loop.
// It watches for DataUpdateEvent with Source "claude" and casts the
// data payload to *claude.UsageReport.
func (w *ClaudeWidget) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case app.DataUpdateEvent:
		if msg.Source != "claude" {
			return nil
		}
		report, ok := msg.Data.(*claude.UsageReport)
		if !ok || report == nil {
			return nil
		}
		w.report = report

		// Append the total cost to the sparkline history.
		w.costHistory = append(w.costHistory, report.TotalCostUSD)
		if len(w.costHistory) > claudeMaxCostHistory {
			w.costHistory = w.costHistory[len(w.costHistory)-claudeMaxCostHistory:]
		}
	}
	return nil
}

// HandleKey processes key events when this widget has focus.
// 'e' toggles between compact and expanded mode.
// 'm' cycles through accounts (when multiple are present).
func (w *ClaudeWidget) HandleKey(key tea.KeyMsg) tea.Cmd {
	switch key.String() {
	case "e":
		w.expanded = !w.expanded
		return nil
	case "m":
		if w.report != nil && len(w.report.Accounts) > 0 {
			w.selectedAccount = (w.selectedAccount + 1) % len(w.report.Accounts)
		}
		return nil
	}
	return nil
}

// View renders the widget content into the given width x height area.
func (w *ClaudeWidget) View(width, height int) string {
	if width <= 0 || height <= 0 {
		return ""
	}

	if w.report == nil || len(w.report.Accounts) == 0 {
		return claudeCenterMessage("No data", width, height)
	}

	var lines []string

	if w.expanded {
		lines = w.viewExpanded(width)
	} else {
		lines = w.viewCompact(width)
	}

	// Pad or truncate to exactly height lines.
	return claudeFitLines(lines, width, height)
}

// viewCompact renders a compact single-gauge-per-account view.
func (w *ClaudeWidget) viewCompact(width int) []string {
	var lines []string

	for _, acct := range w.report.Accounts {
		if !acct.Connected {
			lines = append(lines, claudeTruncLine(
				components.Color(ColorError)+acct.Name+": disconnected"+components.Reset(), width))
			continue
		}

		// Account name + cost.
		header := fmt.Sprintf("%s  $%.2f", components.Bold(acct.Name), acct.CurrentMonth.CostUSD)
		lines = append(lines, claudeTruncLine(header, width))

		// Combined token gauge.
		totalTokens := acct.CurrentMonth.InputTokens + acct.CurrentMonth.OutputTokens
		ratio := claudeTokenRatio(totalTokens, claudeDefaultTokenBudget)
		gaugeWidth := width - 12 // room for label and percent
		if gaugeWidth < 5 {
			gaugeWidth = 5
		}
		gaugeLine := claudeRenderGauge("Tokens", ratio, gaugeWidth,
			fmt.Sprintf(" %s", claudeFormatTokens(totalTokens)))
		lines = append(lines, claudeTruncLine(gaugeLine, width))
	}

	// Cost sparkline if we have history.
	if len(w.costHistory) > 0 {
		sparkWidth := width - 8
		if sparkWidth < 5 {
			sparkWidth = 5
		}
		spark := components.NewSparkline(components.SparklineStyle{
			Width: sparkWidth,
			Color: "#64B5F6",
			Label: "Cost",
		})
		sparkLine := spark.Render(w.costHistory, sparkWidth)
		lines = append(lines, claudeTruncLine(sparkLine, width))
	}

	// Total across all accounts.
	totalLine := fmt.Sprintf("Total: $%.2f", w.report.TotalCostUSD)
	lines = append(lines, claudeTruncLine(
		components.Color(ColorAccent)+totalLine+components.Reset(), width))

	return lines
}

// viewExpanded renders per-model breakdown with individual gauges.
func (w *ClaudeWidget) viewExpanded(width int) []string {
	var lines []string

	for i, acct := range w.report.Accounts {
		if i > 0 {
			lines = append(lines, "") // separator between accounts
		}

		if !acct.Connected {
			lines = append(lines, claudeTruncLine(
				components.Color(ColorError)+acct.Name+": "+acct.Error+components.Reset(), width))
			continue
		}

		// Account header.
		header := fmt.Sprintf("%s  $%.2f this month", components.Bold(acct.Name), acct.CurrentMonth.CostUSD)
		lines = append(lines, claudeTruncLine(header, width))

		// Input token gauge.
		inputRatio := claudeTokenRatio(acct.CurrentMonth.InputTokens, claudeDefaultTokenBudget)
		// Reserve space for label(8) + percent(5) + suffix(~17).
		gaugeWidth := width - 30
		if gaugeWidth < 5 {
			gaugeWidth = 5
		}
		inputLine := claudeRenderGauge("Input", inputRatio, gaugeWidth,
			fmt.Sprintf(" %s", claudeFormatTokens(acct.CurrentMonth.InputTokens)))
		lines = append(lines, claudeTruncLine(inputLine, width))

		// Output token gauge.
		outputRatio := claudeTokenRatio(acct.CurrentMonth.OutputTokens, claudeDefaultTokenBudget)
		outputLine := claudeRenderGauge("Output", outputRatio, gaugeWidth,
			fmt.Sprintf(" %s", claudeFormatTokens(acct.CurrentMonth.OutputTokens)))
		lines = append(lines, claudeTruncLine(outputLine, width))

		// Per-model breakdown.
		for _, m := range acct.Models {
			modelTokens := m.InputTokens + m.OutputTokens
			modelRatio := claudeTokenRatio(modelTokens, claudeDefaultTokenBudget)
			modelName := claudeShortModelName(m.Model)
			label := fmt.Sprintf("  %s", modelName)
			costLabel := fmt.Sprintf(" %s $%.2f", claudeFormatTokens(modelTokens), m.CostUSD)
			mLine := claudeRenderGauge(label, modelRatio, gaugeWidth, costLabel)
			lines = append(lines, claudeTruncLine(mLine, width))
		}

		// Rate limit headroom indicator.
		costRatio := acct.CurrentMonth.CostUSD / claudeDefaultCostBudget
		headroomPct := (1.0 - costRatio) * 100
		if headroomPct < 0 {
			headroomPct = 0
		}
		headroomColor := claudeColorGreen
		if costRatio >= claudeThresholdYellow {
			headroomColor = claudeColorRed
		} else if costRatio >= claudeThresholdGreen {
			headroomColor = claudeColorYellow
		}
		headroomLine := fmt.Sprintf("  Budget headroom: %s%.0f%%%s",
			components.Color(headroomColor), headroomPct, components.Reset())
		lines = append(lines, claudeTruncLine(headroomLine, width))
	}

	// Cost sparkline.
	if len(w.costHistory) > 0 {
		lines = append(lines, "") // separator
		sparkWidth := width - 8
		if sparkWidth < 5 {
			sparkWidth = 5
		}
		spark := components.NewSparkline(components.SparklineStyle{
			Width: sparkWidth,
			Color: "#64B5F6",
			Label: "Cost",
		})
		sparkLine := spark.Render(w.costHistory, sparkWidth)
		lines = append(lines, claudeTruncLine(sparkLine, width))
	}

	// Total line.
	totalLine := fmt.Sprintf("Total: $%.2f", w.report.TotalCostUSD)
	lines = append(lines, claudeTruncLine(
		components.Color(ColorAccent)+totalLine+components.Reset(), width))

	return lines
}

// --- private helpers (all prefixed with "claude" to avoid conflicts) ---

// claudeTokenRatio computes a 0..1 ratio of used tokens against a budget.
func claudeTokenRatio(tokens int64, budget int64) float64 {
	if budget <= 0 {
		return 0
	}
	r := float64(tokens) / float64(budget)
	if r > 1 {
		r = 1
	}
	if r < 0 {
		r = 0
	}
	return r
}

// claudeRenderGauge renders a labeled gauge bar with a trailing suffix.
func claudeRenderGauge(label string, ratio float64, barWidth int, suffix string) string {
	color := claudeRatioColor(ratio)

	g := components.NewGauge(components.GaugeStyle{
		Width:             barWidth,
		ShowPercent:       true,
		FilledColor:       color,
		EmptyColor:        "#333333",
		WarningThreshold:  claudeThresholdGreen,
		CriticalThreshold: claudeThresholdYellow,
		WarningColor:      claudeColorYellow,
		CriticalColor:     claudeColorRed,
		Label:             label,
		LabelWidth:        8,
	})

	bar := g.Render(ratio, 1.0, barWidth)
	return bar + suffix
}

// claudeRatioColor returns the hex color for a given ratio.
func claudeRatioColor(ratio float64) string {
	if ratio >= claudeThresholdYellow {
		return claudeColorRed
	}
	if ratio >= claudeThresholdGreen {
		return claudeColorYellow
	}
	return claudeColorGreen
}

// claudeFormatTokens formats token counts with SI suffixes.
func claudeFormatTokens(tokens int64) string {
	v := float64(tokens)
	switch {
	case v >= 1e9:
		return fmt.Sprintf("%.1fG", v/1e9)
	case v >= 1e6:
		return fmt.Sprintf("%.1fM", v/1e6)
	case v >= 1e3:
		return fmt.Sprintf("%.1fK", v/1e3)
	default:
		return fmt.Sprintf("%d", tokens)
	}
}

// claudeShortModelName shortens a model identifier for display.
func claudeShortModelName(model string) string {
	replacer := strings.NewReplacer(
		"claude-opus-4-6", "Opus 4.6",
		"claude-sonnet-4-5", "Sonnet 4.5",
		"claude-sonnet-4-0", "Sonnet 4.0",
		"claude-3-5-sonnet", "Sonnet 3.5",
		"claude-haiku-4-5", "Haiku 4.5",
		"claude-3-5-haiku", "Haiku 3.5",
		"claude-3-haiku", "Haiku 3",
		"claude-3-opus", "Opus 3",
	)
	short := replacer.Replace(model)
	// If nothing matched the replacer returns the original; truncate long names.
	if len(short) > 16 {
		short = short[:13] + "..."
	}
	return short
}

// claudeCenterMessage renders a centered message in the given area.
func claudeCenterMessage(msg string, width, height int) string {
	lines := make([]string, height)
	midY := height / 2
	for i := range lines {
		if i == midY {
			vis := components.VisibleLen(msg)
			pad := (width - vis) / 2
			if pad < 0 {
				pad = 0
			}
			lines[i] = strings.Repeat(" ", pad) + msg
		} else {
			lines[i] = ""
		}
	}
	return strings.Join(lines, "\n")
}

// claudeFitLines pads or truncates a slice of lines to fit exactly height
// lines, each no wider than width visible characters.
func claudeFitLines(lines []string, width, height int) string {
	// Truncate excess lines.
	if len(lines) > height {
		lines = lines[:height]
	}
	// Pad with empty lines if too few.
	for len(lines) < height {
		lines = append(lines, "")
	}
	// Truncate each line's visible width.
	for i, line := range lines {
		if components.VisibleLen(line) > width {
			lines[i] = components.Truncate(line, width)
		}
	}
	return strings.Join(lines, "\n")
}

// claudeTruncLine truncates a single line to at most width visible characters.
func claudeTruncLine(line string, width int) string {
	if components.VisibleLen(line) > width {
		return components.Truncate(line, width)
	}
	return line
}

// claudeCostColor returns a hex color based on cost ratio to budget.
func claudeCostColor(cost, budget float64) string {
	if budget <= 0 {
		return claudeColorGreen
	}
	ratio := cost / budget
	return claudeRatioColor(ratio)
}

// ensure ClaudeWidget implements app.Widget at compile time.
var _ app.Widget = (*ClaudeWidget)(nil)
