package starship

import (
	"fmt"
	"sort"
	"strings"

	"gitlab.com/tinyland/lab/prompt-pulse/pkg/collectors/billing"
	"gitlab.com/tinyland/lab/prompt-pulse/pkg/collectors/claude"
	"gitlab.com/tinyland/lab/prompt-pulse/pkg/collectors/k8s"
	"gitlab.com/tinyland/lab/prompt-pulse/pkg/collectors/sysmetrics"
	"gitlab.com/tinyland/lab/prompt-pulse/pkg/collectors/tailscale"
)

// ANSI color constants used for segment thresholds.
const (
	ssColorGreen  = "\033[32m"
	ssColorYellow = "\033[33m"
	ssColorRed    = "\033[31m"
)

// ssBudgetDefault is the assumed monthly Claude API budget in USD when no
// explicit budget is available. Used for threshold calculation.
const ssBudgetDefault = 500.0

// ssClaudeSegment renders the Claude/Anthropic cost segment. It shows the
// current month's total cost and the top model by spend.
// Example: "🤖 $142.30 opus"
func ssClaudeSegment(cacheDir string) *Segment {
	report, err := ssReadCachedData[claude.UsageReport](cacheDir, "claude")
	if err != nil || report == nil {
		return nil
	}

	cost := report.TotalCostUSD

	// Find the top model across all accounts.
	topModel := ""
	var topCost float64
	for _, acct := range report.Accounts {
		for _, m := range acct.Models {
			if m.CostUSD > topCost {
				topCost = m.CostUSD
				topModel = m.Model
			}
		}
	}

	// Shorten model name: take the last segment after "claude-" prefix and
	// strip version suffixes for brevity.
	topModel = ssShortModelName(topModel)

	text := fmt.Sprintf("$%.2f", cost)
	if topModel != "" {
		text += " " + topModel
	}

	// Color based on percentage of budget.
	color := ssThresholdColor(cost, ssBudgetDefault)

	return &Segment{
		Icon:  "🤖",
		Text:  text,
		Color: color,
	}
}

// ssShortModelName shortens a Claude model identifier for display.
// "claude-3-5-sonnet-20241022" -> "sonnet"
// "claude-opus-4-20250514" -> "opus"
func ssShortModelName(model string) string {
	if model == "" {
		return ""
	}

	// Known short names, checked in order of specificity.
	lower := strings.ToLower(model)
	for _, name := range []string{"opus", "sonnet", "haiku"} {
		if strings.Contains(lower, name) {
			return name
		}
	}

	// Fall back to the last dash-separated segment that is not a date.
	parts := strings.Split(model, "-")
	for i := len(parts) - 1; i >= 0; i-- {
		if len(parts[i]) < 8 { // skip date-like segments (YYYYMMDD)
			return parts[i]
		}
	}

	return model
}

// ssBillingSegment renders the cloud billing segment showing total monthly
// spend across all configured providers.
// Example: "☁️ $23.45/mo"
func ssBillingSegment(cacheDir string) *Segment {
	report, err := ssReadCachedData[billing.BillingReport](cacheDir, "billing")
	if err != nil || report == nil {
		return nil
	}

	text := fmt.Sprintf("$%.2f/mo", report.TotalMonthlyUSD)

	// Use budget-based color if budget is set, otherwise use absolute thresholds.
	var color string
	if report.BudgetUSD > 0 {
		color = ssThresholdColor(report.TotalMonthlyUSD, report.BudgetUSD)
	} else {
		color = ssThresholdColor(report.TotalMonthlyUSD, 100.0)
	}

	return &Segment{
		Icon:  "☁️",
		Text:  text,
		Color: color,
	}
}

// ssTailscaleSegment renders the Tailscale peer connectivity segment.
// Example: "🔗 3/5 peers"
func ssTailscaleSegment(cacheDir string) *Segment {
	status, err := ssReadCachedData[tailscale.Status](cacheDir, "tailscale")
	if err != nil || status == nil {
		return nil
	}

	total := status.TotalPeers
	online := status.OnlinePeers

	text := fmt.Sprintf("%d/%d peers", online, total)

	var color string
	if total == 0 {
		color = ssColorYellow
	} else {
		ratio := float64(online) / float64(total)
		switch {
		case ratio >= 1.0:
			color = ssColorGreen
		case ratio >= 0.5:
			color = ssColorYellow
		default:
			color = ssColorRed
		}
	}

	return &Segment{
		Icon:  "🔗",
		Text:  text,
		Color: color,
	}
}

// ssK8sSegment renders the Kubernetes pod health segment. It aggregates
// pod counts across all clusters.
// Example: "⎈ 12/15 pods"
func ssK8sSegment(cacheDir string) *Segment {
	status, err := ssReadCachedData[k8s.ClusterStatus](cacheDir, "k8s")
	if err != nil || status == nil {
		return nil
	}

	var totalPods, runningPods, failedPods int
	for _, cluster := range status.Clusters {
		if !cluster.Connected {
			continue
		}
		totalPods += cluster.TotalPods
		runningPods += cluster.RunningPods
		failedPods += cluster.FailedPods
	}

	if totalPods == 0 {
		return nil
	}

	text := fmt.Sprintf("%d/%d pods", runningPods, totalPods)

	var color string
	switch {
	case failedPods > 0:
		color = ssColorRed
	case runningPods < totalPods:
		color = ssColorYellow
	default:
		color = ssColorGreen
	}

	return &Segment{
		Icon:  "⎈",
		Text:  text,
		Color: color,
	}
}

// ssSystemSegment renders the system metrics segment showing CPU and RAM
// utilization percentages.
// Example: "💻 CPU:45% RAM:62%"
func ssSystemSegment(cacheDir string) *Segment {
	metrics, err := ssReadCachedData[sysmetrics.Metrics](cacheDir, "sysmetrics")
	if err != nil || metrics == nil {
		return nil
	}

	cpuPct := metrics.CPU.Total
	ramPct := metrics.Memory.UsedPercent

	text := fmt.Sprintf("CPU:%d%% RAM:%d%%", int(cpuPct), int(ramPct))

	// Color based on the highest of CPU or RAM usage.
	highest := cpuPct
	if ramPct > highest {
		highest = ramPct
	}

	var color string
	switch {
	case highest >= 80:
		color = ssColorRed
	case highest >= 50:
		color = ssColorYellow
	default:
		color = ssColorGreen
	}

	return &Segment{
		Icon:  "💻",
		Text:  text,
		Color: color,
	}
}

// ssThresholdColor returns a color code based on the ratio of value to
// budget. Green for <50%, yellow for 50-80%, red for >=80%.
func ssThresholdColor(value, budget float64) string {
	if budget <= 0 {
		return ssColorGreen
	}
	ratio := value / budget
	switch {
	case ratio >= 0.8:
		return ssColorRed
	case ratio >= 0.5:
		return ssColorYellow
	default:
		return ssColorGreen
	}
}

// ssAllModels collects all model names from a usage report, sorted by cost
// descending. This is a helper used internally.
func ssAllModels(report *claude.UsageReport) []string {
	type mc struct {
		name string
		cost float64
	}
	var models []mc
	for _, acct := range report.Accounts {
		for _, m := range acct.Models {
			models = append(models, mc{name: m.Model, cost: m.CostUSD})
		}
	}
	sort.Slice(models, func(i, j int) bool {
		return models[i].cost > models[j].cost
	})
	names := make([]string, len(models))
	for i, m := range models {
		names[i] = m.name
	}
	return names
}
