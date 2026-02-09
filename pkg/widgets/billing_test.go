package widgets

import (
	"fmt"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"gitlab.com/tinyland/lab/prompt-pulse/pkg/app"
	"gitlab.com/tinyland/lab/prompt-pulse/pkg/collectors/billing"
)

func TestBillingWidget_ID(t *testing.T) {
	w := NewBillingWidget()
	if got := w.ID(); got != "billing" {
		t.Errorf("ID() = %q, want %q", got, "billing")
	}
}

func TestBillingWidget_Title(t *testing.T) {
	w := NewBillingWidget()
	if got := w.Title(); got != "Billing" {
		t.Errorf("Title() = %q, want %q", got, "Billing")
	}
}

func TestBillingWidget_MinSize(t *testing.T) {
	w := NewBillingWidget()
	minW, minH := w.MinSize()
	if minW != 30 || minH != 4 {
		t.Errorf("MinSize() = (%d, %d), want (30, 4)", minW, minH)
	}
}

func TestBillingWidget_View_NoData(t *testing.T) {
	w := NewBillingWidget()
	view := w.View(40, 5)
	if !strings.Contains(view, "No data") {
		t.Errorf("View with no data should contain 'No data', got:\n%s", view)
	}
}

func TestBillingWidget_View_Compact_WithBudget(t *testing.T) {
	w := NewBillingWidget()
	w.report = &billing.BillingReport{
		Providers: []billing.ProviderBilling{
			{
				Name:        "civo",
				Connected:   true,
				MonthToDate: 45.50,
				Resources: []billing.ResourceCost{
					{Name: "k3s-cluster", Type: "kubernetes", MonthlyCost: 30.00},
					{Name: "worker-1", Type: "instance", MonthlyCost: 15.50},
				},
			},
			{
				Name:        "digitalocean",
				Connected:   true,
				MonthToDate: 100.50,
				Resources: []billing.ResourceCost{
					{Name: "db-primary", Type: "droplet", MonthlyCost: 100.50},
				},
			},
		},
		TotalMonthlyUSD: 146.00,
		BudgetUSD:       200.00,
		BudgetPercent:   73.0,
	}

	view := w.View(60, 10)

	// Should contain total cost.
	if !strings.Contains(view, "$146.00") {
		t.Errorf("Compact view should contain total cost '$146.00', got:\n%s", view)
	}

	// Should contain budget reference.
	if !strings.Contains(view, "$200.00") {
		t.Errorf("Compact view should contain budget '$200.00', got:\n%s", view)
	}

	// Should contain provider names.
	if !strings.Contains(view, "civo") {
		t.Errorf("Compact view should contain provider 'civo', got:\n%s", view)
	}
	if !strings.Contains(view, "digitalocean") {
		t.Errorf("Compact view should contain provider 'digitalocean', got:\n%s", view)
	}
}

func TestBillingWidget_View_Expanded_WithResourceTable(t *testing.T) {
	w := NewBillingWidget()
	w.expanded = true
	w.report = &billing.BillingReport{
		Providers: []billing.ProviderBilling{
			{
				Name:        "civo",
				Connected:   true,
				MonthToDate: 45.50,
				Resources: []billing.ResourceCost{
					{Name: "k3s-cluster", Type: "kubernetes", MonthlyCost: 30.00},
					{Name: "worker-1", Type: "instance", MonthlyCost: 15.50},
				},
			},
		},
		TotalMonthlyUSD: 45.50,
		BudgetUSD:       200.00,
		BudgetPercent:   22.75,
	}
	w.costHistory = []float64{30.0, 35.0, 40.0, 45.5}

	view := w.View(70, 20)

	// Should show provider name.
	if !strings.Contains(view, "civo") {
		t.Errorf("Expanded view should contain provider 'civo', got:\n%s", view)
	}

	// Should show resource names in the table.
	if !strings.Contains(view, "k3s-cluster") {
		t.Errorf("Expanded view should contain resource 'k3s-cluster', got:\n%s", view)
	}

	// Should show projected cost.
	if !strings.Contains(view, "Projected:") {
		t.Errorf("Expanded view should contain 'Projected:', got:\n%s", view)
	}

	// Should show total.
	if !strings.Contains(view, "Total MTD:") {
		t.Errorf("Expanded view should contain 'Total MTD:', got:\n%s", view)
	}
}

func TestBillingWidget_Update_WithBillingReport(t *testing.T) {
	w := NewBillingWidget()

	report := &billing.BillingReport{
		Providers: []billing.ProviderBilling{
			{Name: "civo", Connected: true, MonthToDate: 50.00},
		},
		TotalMonthlyUSD: 50.00,
		BudgetUSD:       100.00,
		BudgetPercent:   50.0,
	}

	msg := app.DataUpdateEvent{
		Source: "billing",
		Data:   report,
	}

	cmd := w.Update(msg)
	if cmd != nil {
		t.Errorf("Update should return nil cmd, got %v", cmd)
	}

	if w.report != report {
		t.Error("Update should store the billing report")
	}

	if len(w.costHistory) != 1 || w.costHistory[0] != 50.00 {
		t.Errorf("Update should append to costHistory, got %v", w.costHistory)
	}
}

func TestBillingWidget_Update_IgnoresOtherSources(t *testing.T) {
	w := NewBillingWidget()

	msg := app.DataUpdateEvent{
		Source: "tailscale",
		Data:   "something",
	}

	w.Update(msg)

	if w.report != nil {
		t.Error("Update should ignore events from other sources")
	}
}

func TestBillingWidget_Update_IgnoresErrors(t *testing.T) {
	w := NewBillingWidget()

	msg := app.DataUpdateEvent{
		Source: "billing",
		Err:    fmt.Errorf("api error"),
		Data:   nil,
	}

	w.Update(msg)

	if w.report != nil {
		t.Error("Update should ignore error events")
	}
}

func TestBillingWidget_HandleKey_ToggleExpanded(t *testing.T) {
	w := NewBillingWidget()

	if w.expanded {
		t.Fatal("Widget should start in compact mode")
	}

	// Press 'e' to expand.
	w.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	if !w.expanded {
		t.Error("HandleKey 'e' should toggle expanded to true")
	}

	// Press 'e' again to collapse.
	w.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	if w.expanded {
		t.Error("HandleKey 'e' should toggle expanded back to false")
	}
}

func TestBillingWidget_HandleKey_CycleProviders(t *testing.T) {
	w := NewBillingWidget()
	w.report = &billing.BillingReport{
		Providers: []billing.ProviderBilling{
			{Name: "civo", Connected: true},
			{Name: "digitalocean", Connected: true},
			{Name: "anthropic", Connected: false},
		},
	}

	if w.selectedProvider != 0 {
		t.Fatal("selectedProvider should start at 0")
	}

	w.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	if w.selectedProvider != 1 {
		t.Errorf("After first 'p', selectedProvider = %d, want 1", w.selectedProvider)
	}

	w.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	if w.selectedProvider != 2 {
		t.Errorf("After second 'p', selectedProvider = %d, want 2", w.selectedProvider)
	}

	// Should wrap around.
	w.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	if w.selectedProvider != 0 {
		t.Errorf("After third 'p', selectedProvider = %d, want 0 (wrap)", w.selectedProvider)
	}
}

func TestBillingWidget_BudgetColorThresholds(t *testing.T) {
	tests := []struct {
		ratio    float64
		wantHex  string
		wantDesc string
	}{
		{0.0, billingColorGreen, "green for 0%"},
		{0.30, billingColorGreen, "green for 30%"},
		{0.59, billingColorGreen, "green for 59%"},
		{0.60, billingColorYellow, "yellow for 60%"},
		{0.75, billingColorYellow, "yellow for 75%"},
		{0.84, billingColorYellow, "yellow for 84%"},
		{0.85, billingColorRed, "red for 85%"},
		{0.95, billingColorRed, "red for 95%"},
		{1.0, billingColorRed, "red for 100%"},
		{1.5, billingColorRed, "red for 150% (over budget)"},
	}

	for _, tc := range tests {
		got := billingBudgetColor(tc.ratio)
		if got != tc.wantHex {
			t.Errorf("billingBudgetColor(%v) = %q, want %q (%s)", tc.ratio, got, tc.wantHex, tc.wantDesc)
		}
	}
}

func TestBillingWidget_ProjectedCost(t *testing.T) {
	// Use a fixed date: January 15, 2026 (31 days in January).
	fixedTime := time.Date(2026, time.January, 15, 12, 0, 0, 0, time.UTC)
	currentSpend := 100.0

	projected := billingProjectedCostAt(currentSpend, fixedTime)

	// Expected: (100 / 15) * 31 = 206.666...
	expected := (100.0 / 15.0) * 31.0
	if diff := projected - expected; diff > 0.01 || diff < -0.01 {
		t.Errorf("billingProjectedCostAt(100, Jan 15) = %.2f, want %.2f", projected, expected)
	}
}

func TestBillingWidget_ProjectedCost_February(t *testing.T) {
	// February 2026 (non-leap year, 28 days).
	fixedTime := time.Date(2026, time.February, 10, 12, 0, 0, 0, time.UTC)
	currentSpend := 50.0

	projected := billingProjectedCostAt(currentSpend, fixedTime)

	// Expected: (50 / 10) * 28 = 140.0
	expected := (50.0 / 10.0) * 28.0
	if diff := projected - expected; diff > 0.01 || diff < -0.01 {
		t.Errorf("billingProjectedCostAt(50, Feb 10) = %.2f, want %.2f", projected, expected)
	}
}

func TestBillingWidget_ProviderConnectionStatus(t *testing.T) {
	// Test connected dot.
	connectedDot := billingStatusDot(true)
	if !strings.Contains(connectedDot, "\u25cf") {
		t.Error("Connected status should contain bullet character")
	}
	// The green color escape should be present.
	if !strings.Contains(connectedDot, "38;2;") {
		t.Error("Connected status should contain ANSI color escape")
	}

	// Test disconnected dot.
	disconnectedDot := billingStatusDot(false)
	if !strings.Contains(disconnectedDot, "\u25cf") {
		t.Error("Disconnected status should contain bullet character")
	}
	// The dots should be different (different colors).
	if connectedDot == disconnectedDot {
		t.Error("Connected and disconnected dots should have different colors")
	}
}

func TestBillingWidget_MultiProviderAggregation(t *testing.T) {
	w := NewBillingWidget()
	w.report = &billing.BillingReport{
		Providers: []billing.ProviderBilling{
			{Name: "civo", Connected: true, MonthToDate: 45.50},
			{Name: "digitalocean", Connected: true, MonthToDate: 80.00},
			{Name: "anthropic", Connected: false, MonthToDate: 0.00},
		},
		TotalMonthlyUSD: 125.50,
		BudgetUSD:       300.00,
		BudgetPercent:   41.83,
	}

	view := w.View(60, 10)

	// All three providers should appear.
	if !strings.Contains(view, "civo") {
		t.Error("View should show civo provider")
	}
	if !strings.Contains(view, "digitalocean") {
		t.Error("View should show digitalocean provider")
	}
	if !strings.Contains(view, "anthropic") {
		t.Error("View should show anthropic provider")
	}

	// Total should be shown.
	if !strings.Contains(view, "$125.50") {
		t.Errorf("View should show aggregated total $125.50, got:\n%s", view)
	}
}

func TestBillingWidget_ViewAtVariousSizes(t *testing.T) {
	w := NewBillingWidget()
	w.report = &billing.BillingReport{
		Providers: []billing.ProviderBilling{
			{Name: "civo", Connected: true, MonthToDate: 30.00},
		},
		TotalMonthlyUSD: 30.00,
		BudgetUSD:       100.00,
		BudgetPercent:   30.0,
	}

	tests := []struct {
		name   string
		width  int
		height int
	}{
		{"minimum", 30, 4},
		{"small", 40, 6},
		{"medium", 60, 10},
		{"large", 80, 20},
		{"wide_short", 120, 3},
		{"narrow_tall", 30, 20},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			view := w.View(tc.width, tc.height)
			if view == "" {
				t.Errorf("View(%d, %d) should not be empty with data", tc.width, tc.height)
			}
			lines := strings.Split(view, "\n")
			if len(lines) != tc.height {
				t.Errorf("View(%d, %d) produced %d lines, want %d", tc.width, tc.height, len(lines), tc.height)
			}
		})
	}
}

func TestBillingWidget_View_ZeroDimensions(t *testing.T) {
	w := NewBillingWidget()
	if got := w.View(0, 10); got != "" {
		t.Errorf("View(0, 10) should return empty string, got %q", got)
	}
	if got := w.View(10, 0); got != "" {
		t.Errorf("View(10, 0) should return empty string, got %q", got)
	}
}

func TestBillingWidget_DaysInMonth(t *testing.T) {
	tests := []struct {
		month    time.Month
		year     int
		expected int
	}{
		{time.January, 2026, 31},
		{time.February, 2026, 28},
		{time.February, 2024, 29}, // Leap year.
		{time.April, 2026, 30},
		{time.December, 2025, 31},
	}

	for _, tc := range tests {
		tm := time.Date(tc.year, tc.month, 1, 0, 0, 0, 0, time.UTC)
		got := billingDaysInMonth(tm)
		if got != tc.expected {
			t.Errorf("billingDaysInMonth(%s %d) = %d, want %d",
				tc.month, tc.year, got, tc.expected)
		}
	}
}

func TestBillingWidget_Update_MultipleSamples(t *testing.T) {
	w := NewBillingWidget()

	for i := 1; i <= 5; i++ {
		cost := float64(i) * 10.0
		w.Update(app.DataUpdateEvent{
			Source: "billing",
			Data: &billing.BillingReport{
				TotalMonthlyUSD: cost,
				Providers:       []billing.ProviderBilling{},
			},
		})
	}

	if len(w.costHistory) != 5 {
		t.Errorf("costHistory should have 5 samples, got %d", len(w.costHistory))
	}

	expected := []float64{10.0, 20.0, 30.0, 40.0, 50.0}
	for i, v := range expected {
		if w.costHistory[i] != v {
			t.Errorf("costHistory[%d] = %.2f, want %.2f", i, w.costHistory[i], v)
		}
	}
}

func TestBillingWidget_View_CompactNoBudget(t *testing.T) {
	w := NewBillingWidget()
	w.report = &billing.BillingReport{
		Providers: []billing.ProviderBilling{
			{Name: "civo", Connected: true, MonthToDate: 50.00},
		},
		TotalMonthlyUSD: 50.00,
		BudgetUSD:       0, // No budget set.
	}

	view := w.View(60, 6)

	// Should still show total.
	if !strings.Contains(view, "$50.00") {
		t.Errorf("View without budget should still show total, got:\n%s", view)
	}

	// Should NOT show budget text.
	if strings.Contains(view, "budget") {
		t.Errorf("View without budget should not contain 'budget', got:\n%s", view)
	}
}

// ensure fmt is used (line 187 uses fmt.Errorf).
var _ = fmt.Errorf
