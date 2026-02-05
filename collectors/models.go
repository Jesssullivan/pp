package collectors

import (
	"fmt"
	"strings"
	"time"
)

// ========== Claude Usage Models ==========

// ClaudeUsage holds usage data across all configured Claude accounts.
type ClaudeUsage struct {
	Accounts []ClaudeAccountUsage `json:"accounts"`
}

// ClaudeAccountUsage represents usage data for a single Claude account,
// either a subscription (Pro/Max) or an API key.
type ClaudeAccountUsage struct {
	// Name is a user-defined label for this account (e.g., "personal", "work").
	Name string `json:"name"`

	// Type is "subscription" or "api".
	Type string `json:"type"`

	// Tier identifies the subscription or API tier.
	// Subscription: "pro", "max_5x", "max_20x"
	// API: "tier_1", "tier_2", "tier_3", "tier_4"
	Tier string `json:"tier"`

	// Status indicates the account's current state.
	// Values: "ok", "auth_failed", "rate_limited", "disabled"
	Status string `json:"status"`

	// FiveHour is the 5-hour rolling usage window (subscription only).
	FiveHour *UsagePeriod `json:"five_hour,omitempty"`

	// SevenDay is the 7-day rolling usage window (subscription only).
	SevenDay *UsagePeriod `json:"seven_day,omitempty"`

	// ExtraUsage tracks overuse credits beyond subscription limits.
	ExtraUsage *ExtraUsage `json:"extra_usage,omitempty"`

	// RateLimits holds API rate limit data from response headers (API only).
	RateLimits *APIRateLimits `json:"rate_limits,omitempty"`
}

// UsagePeriod represents a rolling usage window with utilization percentage.
type UsagePeriod struct {
	// Utilization is the usage percentage from 0 to 100.
	Utilization float64 `json:"utilization"`

	// ResetsAt is when this usage window resets.
	ResetsAt time.Time `json:"resets_at"`
}

// ExtraUsage tracks credit-based overuse beyond subscription limits.
type ExtraUsage struct {
	// Enabled indicates whether extra usage credits are turned on.
	Enabled bool `json:"enabled"`

	// MonthlyLimit is the monthly spending cap in cents.
	MonthlyLimit int `json:"monthly_limit_cents"`

	// UsedCredits is the amount used this month in cents.
	UsedCredits float64 `json:"used_credits_cents"`

	// Utilization is the percentage of monthly limit consumed (0-100).
	Utilization float64 `json:"utilization"`
}

// APIRateLimits holds rate limit information from Anthropic API response headers.
type APIRateLimits struct {
	RequestsLimit     int       `json:"requests_limit"`
	RequestsRemaining int       `json:"requests_remaining"`
	RequestsReset     time.Time `json:"requests_reset"`
	TokensLimit       int       `json:"tokens_limit"`
	TokensRemaining   int       `json:"tokens_remaining"`
	TokensReset       time.Time `json:"tokens_reset"`
}

// ResetSchedule holds consolidated reset times for all usage periods.
// This provides a unified view of when each usage window will reset.
type ResetSchedule struct {
	// SessionResets is when the 5-hour session window resets.
	SessionResets time.Time `json:"session_resets"`
	// WeeklyResets is when the 7-day weekly window resets.
	WeeklyResets time.Time `json:"weekly_resets"`
	// MonthlyResets is when the monthly billing cycle resets (1st of next month).
	MonthlyResets time.Time `json:"monthly_resets"`
}

// ========== Claude Usage Helper Methods ==========

// GetResetSchedule returns a consolidated view of all reset times for this account.
// For subscription accounts, it extracts times from FiveHour, SevenDay, and ExtraUsage.
// For API accounts, it uses the RateLimits reset times.
func (c *ClaudeAccountUsage) GetResetSchedule() *ResetSchedule {
	schedule := &ResetSchedule{}

	if c.Type == "subscription" {
		if c.FiveHour != nil && !c.FiveHour.ResetsAt.IsZero() {
			schedule.SessionResets = c.FiveHour.ResetsAt
		}
		if c.SevenDay != nil && !c.SevenDay.ResetsAt.IsZero() {
			schedule.WeeklyResets = c.SevenDay.ResetsAt
		}
		// Monthly resets on the 1st of next month
		now := time.Now()
		schedule.MonthlyResets = time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, now.Location())
	} else if c.Type == "api" && c.RateLimits != nil {
		// API accounts use requests reset for session-like behavior
		if !c.RateLimits.RequestsReset.IsZero() {
			schedule.SessionResets = c.RateLimits.RequestsReset
		}
		if !c.RateLimits.TokensReset.IsZero() {
			schedule.WeeklyResets = c.RateLimits.TokensReset
		}
	}

	return schedule
}

// StatusColor returns a color indicator based on utilization:
// "green" (<70%), "yellow" (70-89%), "red" (>=90%)
func (c *ClaudeAccountUsage) StatusColor() string {
	util := c.GetPrimaryUtilization()
	switch {
	case util >= 90:
		return "red"
	case util >= 70:
		return "yellow"
	default:
		return "green"
	}
}

// GetPrimaryUtilization returns the most relevant utilization percentage.
// For subscriptions, this is the 5-hour window. For API, it's request utilization.
func (c *ClaudeAccountUsage) GetPrimaryUtilization() float64 {
	if c.Type == "subscription" {
		if c.FiveHour != nil {
			return c.FiveHour.Utilization
		}
		if c.SevenDay != nil {
			return c.SevenDay.Utilization
		}
		return 0
	}

	if c.RateLimits != nil && c.RateLimits.RequestsLimit > 0 {
		used := c.RateLimits.RequestsLimit - c.RateLimits.RequestsRemaining
		return float64(used) / float64(c.RateLimits.RequestsLimit) * 100
	}
	return 0
}

// GetSecondaryUtilization returns the secondary utilization percentage.
// For subscriptions, this is the 7-day window. For API, it's token utilization.
func (c *ClaudeAccountUsage) GetSecondaryUtilization() float64 {
	if c.Type == "subscription" {
		if c.SevenDay != nil {
			return c.SevenDay.Utilization
		}
		return 0
	}

	if c.RateLimits != nil && c.RateLimits.TokensLimit > 0 {
		used := c.RateLimits.TokensLimit - c.RateLimits.TokensRemaining
		return float64(used) / float64(c.RateLimits.TokensLimit) * 100
	}
	return 0
}

// ========== Billing Models ==========

// BillingData aggregates billing information across cloud providers.
type BillingData struct {
	Providers []ProviderBilling `json:"providers"`
	Total     BillingSummary    `json:"total"`
	// History holds 30-day rolling spend history for sparkline display.
	History *BillingHistory `json:"history,omitempty"`
}

// ProviderBilling holds billing data for a single cloud provider.
type ProviderBilling struct {
	// Provider identifies the cloud service (e.g., "civo", "digitalocean", "aws", "dreamhost").
	Provider string `json:"provider"`

	// AccountName is a human-readable label for the account.
	AccountName string `json:"account_name"`

	// Status indicates data freshness: "ok", "error", "stale".
	Status string `json:"status"`

	// DashboardURL links to the provider's billing dashboard.
	DashboardURL string `json:"dashboard_url"`

	// CurrentMonth holds the current billing period data.
	CurrentMonth MonthCost `json:"current_month"`

	// PreviousMonth is last month's total spend in USD, if available.
	PreviousMonth *float64 `json:"previous_month_usd,omitempty"`

	// FetchedAt is when this data was last retrieved.
	FetchedAt time.Time `json:"fetched_at"`
}

// MonthCost represents spending data for a billing period.
type MonthCost struct {
	// SpendUSD is the actual spend so far in USD.
	SpendUSD float64 `json:"spend_usd"`

	// ForecastUSD is the projected end-of-month spend, if available.
	ForecastUSD *float64 `json:"forecast_usd,omitempty"`

	// BudgetUSD is the configured budget for this period, if set.
	BudgetUSD *float64 `json:"budget_usd,omitempty"`

	// StartDate is the billing period start (YYYY-MM-DD).
	StartDate string `json:"start_date"`

	// EndDate is the billing period end (YYYY-MM-DD).
	EndDate string `json:"end_date"`
}

// BillingSummary holds aggregate billing totals across all providers.
type BillingSummary struct {
	// CurrentMonthUSD is the sum of all provider current month spend.
	CurrentMonthUSD float64 `json:"current_month_usd"`

	// ForecastUSD is the sum of all provider forecasts, if available.
	ForecastUSD *float64 `json:"forecast_usd,omitempty"`

	// BudgetUSD is the total budget across all providers, if set.
	BudgetUSD *float64 `json:"budget_usd,omitempty"`
}

// BillingHistory holds 30-day rolling spend history for sparkline visualization.
type BillingHistory struct {
	// ProviderHistory maps provider name to daily spend history.
	// Each slice contains up to 30 days of spend (most recent last).
	ProviderHistory map[string][]DailySpend `json:"provider_history"`

	// TotalHistory is the aggregated daily spend across all providers.
	TotalHistory []DailySpend `json:"total_history"`

	// LastUpdated is when this history was last refreshed.
	LastUpdated time.Time `json:"last_updated"`
}

// DailySpend represents spend for a single day.
type DailySpend struct {
	// Date is the day in YYYY-MM-DD format.
	Date string `json:"date"`
	// SpendUSD is the spend for this day in USD.
	SpendUSD float64 `json:"spend_usd"`
}

// ProviderBudgetConfig holds budget configuration for a single provider.
type ProviderBudgetConfig struct {
	// Provider identifies the cloud service.
	Provider string `json:"provider"`
	// MonthlyBudgetUSD is the monthly budget limit in USD.
	MonthlyBudgetUSD float64 `json:"monthly_budget_usd"`
	// AlertThreshold is the percentage (0-100) at which to trigger warnings.
	AlertThreshold float64 `json:"alert_threshold"`
}

// GetSpendValues extracts the SpendUSD values from a DailySpend slice for sparkline rendering.
func GetSpendValues(history []DailySpend) []float64 {
	if len(history) == 0 {
		return nil
	}
	values := make([]float64, len(history))
	for i, d := range history {
		values[i] = d.SpendUSD
	}
	return values
}

// ========== Infrastructure Models ==========

// InfraStatus holds the status of infrastructure components.
type InfraStatus struct {
	Tailscale  *TailscaleStatus    `json:"tailscale,omitempty"`
	Kubernetes []KubernetesCluster `json:"kubernetes,omitempty"`
}

// TailscaleStatus represents the state of the Tailscale mesh network.
type TailscaleStatus struct {
	// Tailnet is the tailnet name (e.g., "tinyland.ts.net").
	Tailnet string `json:"tailnet"`

	// OnlineCount is the number of currently online nodes.
	OnlineCount int `json:"online_count"`

	// TotalCount is the total number of registered nodes.
	TotalCount int `json:"total_count"`

	// Nodes lists all nodes in the tailnet.
	Nodes []TailscaleNode `json:"nodes"`
}

// TailscaleNode represents a single machine in the Tailscale mesh.
type TailscaleNode struct {
	Name         string    `json:"name"`
	Hostname     string    `json:"hostname"`
	IP           string    `json:"ip"`
	OS           string    `json:"os"`
	Online       bool      `json:"online"`
	LastSeen     time.Time `json:"last_seen"`
	Tags         []string  `json:"tags,omitempty"`
	DashboardURL string    `json:"dashboard_url"`

	// System metrics (populated via SSH for online nodes)
	CPUPercent  *float64 `json:"cpu_percent,omitempty"`
	RAMPercent  *float64 `json:"ram_percent,omitempty"`
	DiskPercent *float64 `json:"disk_percent,omitempty"`
}

// KubernetesCluster represents a single Kubernetes cluster.
type KubernetesCluster struct {
	// Name is the cluster identifier.
	Name string `json:"name"`

	// Platform identifies the provider (e.g., "civo", "doks", "rke2").
	Platform string `json:"platform"`

	// Status is the cluster health: "healthy", "degraded", "offline".
	Status string `json:"status"`

	// APIEndpoint is the Kubernetes API server URL.
	APIEndpoint string `json:"api_endpoint"`

	// DashboardURL links to the provider's cluster management UI.
	DashboardURL string `json:"dashboard_url"`

	// Nodes lists the cluster's worker/control-plane nodes.
	Nodes []KubernetesNode `json:"nodes"`

	// TotalNodes is the total number of nodes in the cluster.
	TotalNodes int `json:"total_nodes"`

	// ReadyNodes is the number of nodes in Ready state.
	ReadyNodes int `json:"ready_nodes"`
}

// KubernetesNode represents a single node in a Kubernetes cluster.
type KubernetesNode struct {
	// Name is the node identifier.
	Name string `json:"name"`

	// Status is the node condition: "Ready", "NotReady", "Unknown".
	Status string `json:"status"`

	// CPUPercent is the current CPU utilization (0-100).
	CPUPercent float64 `json:"cpu_percent"`

	// MemPercent is the current memory utilization (0-100).
	MemPercent float64 `json:"mem_percent"`

	// PodCount is the number of pods running on this node.
	PodCount int `json:"pod_count"`

	// MaxPods is the maximum number of pods this node can host.
	MaxPods int `json:"max_pods"`
}

// ========== Starship Output Helpers ==========

// formatRelativeTime formats a time.Time as a human-readable relative duration.
// Returns strings like "2h 15m", "3d 12h", "45m", or "now" if already past.
func formatRelativeTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}

	d := time.Until(t)
	if d <= 0 {
		return "now"
	}

	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	mins := int(d.Minutes()) % 60

	if days > 0 {
		return fmt.Sprintf("%dd %dh", days, hours)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, mins)
	}
	return fmt.Sprintf("%dm", mins)
}

// StarshipOutput generates a one-line string suitable for a Starship custom module.
// Format: account_name:tier utilization% (resets Xh Ym) | account_name:tier utilization%
// Accounts with warnings are marked with a warning indicator.
// High utilization (>80%) is marked with ⚠️.
func (c *ClaudeUsage) StarshipOutput() string {
	if c == nil || len(c.Accounts) == 0 {
		return ""
	}

	var parts []string
	for _, acct := range c.Accounts {
		if acct.Status != "ok" {
			parts = append(parts, fmt.Sprintf("%s:ERR", acct.Name))
			continue
		}

		var utilization float64
		var resetStr string
		switch acct.Type {
		case "subscription":
			if acct.FiveHour != nil {
				utilization = acct.FiveHour.Utilization
				if !acct.FiveHour.ResetsAt.IsZero() {
					resetStr = formatRelativeTime(acct.FiveHour.ResetsAt)
				}
			}
		case "api":
			if acct.RateLimits != nil && acct.RateLimits.RequestsLimit > 0 {
				used := acct.RateLimits.RequestsLimit - acct.RateLimits.RequestsRemaining
				utilization = float64(used) / float64(acct.RateLimits.RequestsLimit) * 100
				if !acct.RateLimits.RequestsReset.IsZero() {
					resetStr = formatRelativeTime(acct.RateLimits.RequestsReset)
				}
			}
		}

		// Build output with optional reset time and warning
		output := fmt.Sprintf("%s:%s %.0f%%", acct.Name, acct.Tier, utilization)
		if resetStr != "" {
			output += fmt.Sprintf(" (%s)", resetStr)
		}
		if utilization >= 80 {
			output += " ⚠️"
		}

		parts = append(parts, output)
	}

	return strings.Join(parts, " | ")
}

// StarshipOutput generates a one-line string suitable for a Starship custom module.
// Format: $total_spend ($forecast forecast)
func (b *BillingData) StarshipOutput() string {
	if b == nil {
		return ""
	}

	output := fmt.Sprintf("$%.0f", b.Total.CurrentMonthUSD)

	if b.Total.ForecastUSD != nil {
		output += fmt.Sprintf(" ($%.0f forecast)", *b.Total.ForecastUSD)
	}

	if b.Total.BudgetUSD != nil && b.Total.CurrentMonthUSD > *b.Total.BudgetUSD {
		output += " OVER BUDGET"
	}

	return output
}

// StarshipOutput generates a one-line string suitable for a Starship custom module.
// Format: ts:online/total k8s:cluster_status
func (i *InfraStatus) StarshipOutput() string {
	if i == nil {
		return ""
	}

	var parts []string

	if i.Tailscale != nil {
		parts = append(parts, fmt.Sprintf("ts:%d/%d", i.Tailscale.OnlineCount, i.Tailscale.TotalCount))
	}

	for _, cluster := range i.Kubernetes {
		parts = append(parts, fmt.Sprintf("k8s:%s:%s", cluster.Name, cluster.Status))
	}

	return strings.Join(parts, " ")
}

// NodeMetricsSummary returns a compact summary of node metrics for display.
// Format: "hostname: CPU X% | RAM Y% | Disk Z%"
// Returns empty string if node has no metrics.
func (n *TailscaleNode) NodeMetricsSummary() string {
	if n.CPUPercent == nil && n.RAMPercent == nil && n.DiskPercent == nil {
		return ""
	}

	var parts []string
	if n.CPUPercent != nil {
		parts = append(parts, fmt.Sprintf("CPU %.0f%%", *n.CPUPercent))
	}
	if n.RAMPercent != nil {
		parts = append(parts, fmt.Sprintf("RAM %.0f%%", *n.RAMPercent))
	}
	if n.DiskPercent != nil {
		parts = append(parts, fmt.Sprintf("Disk %.0f%%", *n.DiskPercent))
	}

	if len(parts) == 0 {
		return ""
	}

	return fmt.Sprintf("%s: %s", n.Hostname, strings.Join(parts, " | "))
}

// HasHighUtilization returns true if any metric exceeds the threshold (80%).
func (n *TailscaleNode) HasHighUtilization() bool {
	threshold := 80.0
	if n.CPUPercent != nil && *n.CPUPercent >= threshold {
		return true
	}
	if n.RAMPercent != nil && *n.RAMPercent >= threshold {
		return true
	}
	if n.DiskPercent != nil && *n.DiskPercent >= threshold {
		return true
	}
	return false
}

// ========== Fastfetch Models ==========

// FastfetchModule represents a single fastfetch output module.
// This is a re-export for cache compatibility.
type FastfetchModule struct {
	Type   string `json:"type"`
	Key    string `json:"key,omitempty"`
	KeyRaw string `json:"keyRaw,omitempty"`
	Result string `json:"result,omitempty"`
}

// FastfetchData holds system information collected via fastfetch.
// This is a re-export for cache compatibility.
type FastfetchData struct {
	OS        FastfetchModule `json:"os"`
	Host      FastfetchModule `json:"host"`
	Kernel    FastfetchModule `json:"kernel"`
	Uptime    FastfetchModule `json:"uptime"`
	Packages  FastfetchModule `json:"packages"`
	Shell     FastfetchModule `json:"shell"`
	Terminal  FastfetchModule `json:"terminal"`
	CPU       FastfetchModule `json:"cpu"`
	GPU       FastfetchModule `json:"gpu"`
	Memory    FastfetchModule `json:"memory"`
	Disk      FastfetchModule `json:"disk"`
	LocalIP   FastfetchModule `json:"localIP"`
	Battery   FastfetchModule `json:"battery,omitempty"`
	WM        FastfetchModule `json:"wm,omitempty"`
	Theme     FastfetchModule `json:"theme,omitempty"`
	Icons     FastfetchModule `json:"icons,omitempty"`
	Font      FastfetchModule `json:"font,omitempty"`
	Cursor    FastfetchModule `json:"cursor,omitempty"`
	Locale    FastfetchModule `json:"locale,omitempty"`
	DateTime  FastfetchModule `json:"dateTime,omitempty"`
	PublicIP  FastfetchModule `json:"publicIP,omitempty"`
	Weather   FastfetchModule `json:"weather,omitempty"`
	Player    FastfetchModule `json:"player,omitempty"`
	Media     FastfetchModule `json:"media,omitempty"`
	Processes FastfetchModule `json:"processes,omitempty"`
	Swap      FastfetchModule `json:"swap,omitempty"`
}

// IsEmpty returns true if no modules have been populated.
func (d *FastfetchData) IsEmpty() bool {
	return d.OS.Type == "" &&
		d.Host.Type == "" &&
		d.Kernel.Type == "" &&
		d.CPU.Type == "" &&
		d.Memory.Type == ""
}

// FormatForDisplay returns a slice of key-value pairs suitable for banner display.
func (d *FastfetchData) FormatForDisplay() []string {
	var lines []string

	addLine := func(m FastfetchModule) {
		if m.Type == "" || m.Result == "" {
			return
		}
		key := m.Type
		if m.Key != "" {
			key = m.Key
		}
		lines = append(lines, key+": "+m.Result)
	}

	addLine(d.OS)
	addLine(d.Host)
	addLine(d.Kernel)
	addLine(d.Uptime)
	addLine(d.CPU)
	addLine(d.GPU)
	addLine(d.Memory)
	addLine(d.Disk)
	addLine(d.Packages)
	addLine(d.Shell)
	addLine(d.Terminal)
	addLine(d.LocalIP)

	return lines
}

// FormatCompact returns a condensed view with only essential system info.
func (d *FastfetchData) FormatCompact() []string {
	var lines []string

	addLine := func(label, value string) {
		if value == "" {
			return
		}
		lines = append(lines, label+": "+value)
	}

	addLine("OS", d.OS.Result)
	addLine("Kernel", d.Kernel.Result)
	addLine("CPU", d.CPU.Result)
	addLine("RAM", d.Memory.Result)
	addLine("Disk", d.Disk.Result)
	addLine("Uptime", d.Uptime.Result)

	return lines
}
