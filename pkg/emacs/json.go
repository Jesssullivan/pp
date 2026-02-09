// Package emacs provides output formats suitable for Emacs consumption,
// including JSON rendering, propertized text, and an Elisp package generator
// for dashboard integration.
package emacs

import (
	"encoding/json"
	"fmt"
	"time"

	"gitlab.com/tinyland/lab/prompt-pulse/pkg/cache"
)

// Version is the emacs integration protocol version.
const Version = "1.0.0"

// JSONOutput represents the full dashboard state as JSON for Elisp parsing.
type JSONOutput struct {
	Version   string       `json:"version"`
	Timestamp string       `json:"timestamp"`
	Widgets   []WidgetJSON `json:"widgets"`
	WaifuPath string       `json:"waifu_path,omitempty"`
}

// WidgetJSON represents a single dashboard widget in JSON form.
type WidgetJSON struct {
	ID      string         `json:"id"`
	Title   string         `json:"title"`
	Status  string         `json:"status"` // "ok", "warning", "error", "unknown"
	Summary string         `json:"summary"` // one-line summary
	Data    map[string]any `json:"data"`    // widget-specific structured data
}

// --- Self-contained data types for cache deserialization ---
// These mirror the v1 collector types so that cached JSON written by the
// daemon can be read without importing the collector packages.

// emClaudeUsage mirrors the cached Claude usage structure.
type emClaudeUsage struct {
	Accounts []emClaudeAccount `json:"accounts"`
}

type emClaudeAccount struct {
	Name     string         `json:"name"`
	Type     string         `json:"type"`
	Tier     string         `json:"tier"`
	Status   string         `json:"status"`
	FiveHour *emUsagePeriod `json:"five_hour,omitempty"`
	RateLimits *emAPIRateLimits `json:"rate_limits,omitempty"`
}

func (a emClaudeAccount) getPrimaryUtilization() float64 {
	if a.FiveHour != nil {
		return a.FiveHour.Utilization
	}
	if a.RateLimits != nil && a.RateLimits.RequestsLimit > 0 {
		return 100.0 * float64(a.RateLimits.RequestsLimit-a.RateLimits.RequestsRemaining) / float64(a.RateLimits.RequestsLimit)
	}
	return 0
}

type emUsagePeriod struct {
	Utilization float64 `json:"utilization"`
}

type emAPIRateLimits struct {
	RequestsLimit     int `json:"requests_limit"`
	RequestsRemaining int `json:"requests_remaining"`
}

// emBillingData mirrors the cached billing structure.
type emBillingData struct {
	Providers []emProviderBilling `json:"providers"`
	Total     emBillingSummary    `json:"total"`
}

type emProviderBilling struct {
	Provider     string      `json:"provider"`
	AccountName  string      `json:"account_name"`
	Status       string      `json:"status"`
	CurrentMonth emMonthCost `json:"current_month"`
}

type emMonthCost struct {
	SpendUSD float64 `json:"spend_usd"`
}

type emBillingSummary struct {
	CurrentMonthUSD float64  `json:"current_month_usd"`
	BudgetUSD       *float64 `json:"budget_usd,omitempty"`
	ForecastUSD     *float64 `json:"forecast_usd,omitempty"`
	SuccessCount    int      `json:"success_count"`
	ErrorCount      int      `json:"error_count"`
	TotalConfigured int      `json:"total_configured"`
}

// emInfraStatus mirrors the cached infrastructure structure.
type emInfraStatus struct {
	Tailscale  *emTailscaleStatus   `json:"tailscale,omitempty"`
	Kubernetes []emKubernetesCluster `json:"kubernetes,omitempty"`
}

type emTailscaleStatus struct {
	Tailnet     string `json:"tailnet"`
	OnlineCount int    `json:"online_count"`
	TotalCount  int    `json:"total_count"`
}

type emKubernetesCluster struct {
	Name        string `json:"name"`
	Status      string `json:"status"`
	TotalPods   int    `json:"total_pods"`
	RunningPods int    `json:"running_pods"`
	TotalNodes  int    `json:"total_nodes"`
	ReadyNodes  int    `json:"ready_nodes"`
}

// emSysMetrics mirrors the cached system metrics structure.
type emSysMetrics struct {
	CPU       float64 `json:"cpu"`
	RAM       float64 `json:"ram"`
	Disk      float64 `json:"disk"`
	LoadAvg1  float64 `json:"load_avg_1"`
	LoadAvg5  float64 `json:"load_avg_5"`
	LoadAvg15 float64 `json:"load_avg_15"`
}

// RenderJSON produces JSON output from cached collector data.
// It reads from the cache directory and assembles all available widget data
// into a single JSONOutput structure. The waifuPath is included if non-empty.
func RenderJSON(cacheDir string, waifuPath string) ([]byte, error) {
	store, err := emOpenStore(cacheDir)
	if err != nil {
		return nil, fmt.Errorf("emacs json: open cache: %w", err)
	}
	defer store.Close()

	output := JSONOutput{
		Version:   Version,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Widgets:   make([]WidgetJSON, 0),
		WaifuPath: waifuPath,
	}

	extractors := []func(s *cache.Store) *WidgetJSON{
		emExtractClaude,
		emExtractBilling,
		emExtractTailscale,
		emExtractK8s,
		emExtractSystem,
	}

	for _, extract := range extractors {
		if w := extract(store); w != nil {
			output.Widgets = append(output.Widgets, *w)
		}
	}

	return json.MarshalIndent(output, "", "  ")
}

// emOpenStore creates a cache store for the given directory.
func emOpenStore(cacheDir string) (*cache.Store, error) {
	return cache.NewStore(cache.StoreConfig{
		Dir:        cacheDir,
		DefaultTTL: 24 * time.Hour,
	})
}

// emCacheTTL is the TTL used when reading cached data for Emacs output.
// We use a generous TTL since stale data is better than no data for display.
const emCacheTTL = 24 * time.Hour

// emExtractClaude reads cached Claude data and produces a WidgetJSON.
// Returns nil if no cached data is available.
func emExtractClaude(s *cache.Store) *WidgetJSON {
	data, ok := cache.GetTyped[emClaudeUsage](s, "claude")
	if !ok {
		return nil
	}

	acctCount := len(data.Accounts)
	if acctCount == 0 {
		return &WidgetJSON{
			ID:      "claude",
			Title:   "Claude Usage",
			Status:  "unknown",
			Summary: "no accounts configured",
			Data:    map[string]any{"accounts": 0},
		}
	}

	// Determine overall status and compute aggregate utilization.
	status := "ok"
	var maxUtil float64
	okCount := 0
	for _, acct := range data.Accounts {
		if acct.Status == "ok" || acct.Status == "active" {
			okCount++
			util := acct.getPrimaryUtilization()
			if util > maxUtil {
				maxUtil = util
			}
		}
	}

	if okCount == 0 {
		status = "error"
	} else if maxUtil >= 80 {
		status = "warning"
	}

	summary := fmt.Sprintf("%d account", acctCount)
	if acctCount != 1 {
		summary += "s"
	}
	if maxUtil > 0 {
		summary += fmt.Sprintf(", %.0f%% peak util", maxUtil)
	}

	widgetData := map[string]any{
		"accounts": acctCount,
		"ok_count": okCount,
		"max_util": maxUtil,
	}

	return &WidgetJSON{
		ID:      "claude",
		Title:   "Claude Usage",
		Status:  status,
		Summary: summary,
		Data:    widgetData,
	}
}

// emExtractBilling reads cached billing data and produces a WidgetJSON.
// Returns nil if no cached data is available.
func emExtractBilling(s *cache.Store) *WidgetJSON {
	data, ok := cache.GetTyped[emBillingData](s, "billing")
	if !ok {
		return nil
	}

	providerCount := data.Total.TotalConfigured
	if providerCount == 0 {
		return &WidgetJSON{
			ID:      "billing",
			Title:   "Billing",
			Status:  "unknown",
			Summary: "no providers configured",
			Data:    map[string]any{"providers": 0},
		}
	}

	status := "ok"
	if data.Total.ErrorCount > 0 && data.Total.SuccessCount == 0 {
		status = "error"
	} else if data.Total.ErrorCount > 0 {
		status = "warning"
	}

	// Check budget threshold.
	if data.Total.BudgetUSD != nil && *data.Total.BudgetUSD > 0 {
		pct := (data.Total.CurrentMonthUSD / *data.Total.BudgetUSD) * 100
		if pct >= 100 {
			status = "error"
		} else if pct >= 70 {
			status = "warning"
		}
	}

	summary := fmt.Sprintf("$%.2f/mo (%d provider", data.Total.CurrentMonthUSD, providerCount)
	if providerCount != 1 {
		summary += "s"
	}
	summary += ")"

	widgetData := map[string]any{
		"total_usd":     data.Total.CurrentMonthUSD,
		"providers":     providerCount,
		"success_count": data.Total.SuccessCount,
		"error_count":   data.Total.ErrorCount,
	}

	if data.Total.BudgetUSD != nil {
		widgetData["budget_usd"] = *data.Total.BudgetUSD
	}
	if data.Total.ForecastUSD != nil {
		widgetData["forecast_usd"] = *data.Total.ForecastUSD
	}

	return &WidgetJSON{
		ID:      "billing",
		Title:   "Billing",
		Status:  status,
		Summary: summary,
		Data:    widgetData,
	}
}

// emExtractTailscale reads cached infrastructure/tailscale data and produces a WidgetJSON.
// Returns nil if no cached data is available.
func emExtractTailscale(s *cache.Store) *WidgetJSON {
	data, ok := cache.GetTyped[emInfraStatus](s, "infra")
	if !ok || data.Tailscale == nil {
		return nil
	}

	ts := data.Tailscale
	status := "ok"
	if ts.OnlineCount == 0 && ts.TotalCount > 0 {
		status = "error"
	} else if ts.OnlineCount < ts.TotalCount {
		status = "warning"
	}

	summary := fmt.Sprintf("%d/%d peers online", ts.OnlineCount, ts.TotalCount)

	widgetData := map[string]any{
		"online_count": ts.OnlineCount,
		"total_count":  ts.TotalCount,
		"tailnet":      ts.Tailnet,
	}

	return &WidgetJSON{
		ID:      "tailscale",
		Title:   "Tailscale",
		Status:  status,
		Summary: summary,
		Data:    widgetData,
	}
}

// emExtractK8s reads cached infrastructure/kubernetes data and produces a WidgetJSON.
// Returns nil if no cached data is available.
func emExtractK8s(s *cache.Store) *WidgetJSON {
	data, ok := cache.GetTyped[emInfraStatus](s, "infra")
	if !ok || len(data.Kubernetes) == 0 {
		return nil
	}

	totalPods := 0
	runningPods := 0
	healthyCount := 0
	for _, cluster := range data.Kubernetes {
		totalPods += cluster.TotalPods
		runningPods += cluster.RunningPods
		if cluster.Status == "healthy" {
			healthyCount++
		}
	}

	status := "ok"
	if healthyCount == 0 {
		status = "error"
	} else if runningPods < totalPods {
		status = "warning"
	}

	summary := fmt.Sprintf("%d/%d pods running", runningPods, totalPods)

	widgetData := map[string]any{
		"clusters":     len(data.Kubernetes),
		"total_pods":   totalPods,
		"running_pods": runningPods,
	}

	return &WidgetJSON{
		ID:      "k8s",
		Title:   "Kubernetes",
		Status:  status,
		Summary: summary,
		Data:    widgetData,
	}
}

// emExtractSystem reads cached sysmetrics data and produces a WidgetJSON.
// Returns nil if no cached data is available.
func emExtractSystem(s *cache.Store) *WidgetJSON {
	data, ok := cache.GetTyped[emSysMetrics](s, "sysmetrics")
	if !ok {
		return nil
	}

	status := "ok"
	if data.CPU >= 90 || data.RAM >= 90 {
		status = "error"
	} else if data.CPU >= 70 || data.RAM >= 70 {
		status = "warning"
	}

	summary := fmt.Sprintf("CPU:%.0f%% RAM:%.0f%%", data.CPU, data.RAM)

	widgetData := map[string]any{
		"cpu":        data.CPU,
		"ram":        data.RAM,
		"disk":       data.Disk,
		"load_avg_1": data.LoadAvg1,
	}

	return &WidgetJSON{
		ID:      "system",
		Title:   "System",
		Status:  status,
		Summary: summary,
		Data:    widgetData,
	}
}
