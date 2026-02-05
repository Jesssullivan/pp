package widgets

import (
	"strings"
	"testing"
	"time"
)

func TestRenderBillingPanel_EmptyData(t *testing.T) {
	data := BillingPanelData{}
	cfg := DefaultBillingPanelConfig()
	cfg.ColorEnabled = false

	result := RenderBillingPanel(data, cfg)

	if !strings.Contains(result, "Billing Dashboard") {
		t.Error("expected title 'Billing Dashboard' in output")
	}
	if !strings.Contains(result, "Total:") {
		t.Error("expected 'Total:' in output")
	}
}

func TestRenderBillingPanel_WithProviders(t *testing.T) {
	forecast := 150.0
	budget := 200.0

	data := BillingPanelData{
		Providers: []ProviderSpend{
			{
				Name:     "civo",
				Current:  45.50,
				Forecast: &forecast,
				Budget:   &budget,
				History:  []float64{10, 20, 30, 40, 45.50},
				Status:   "ok",
			},
			{
				Name:    "digitalocean",
				Current: 25.00,
				Status:  "ok",
			},
		},
		TotalCurrent:  70.50,
		TotalForecast: &forecast,
		TotalBudget:   &budget,
		FetchedAt:     time.Now(),
	}
	cfg := DefaultBillingPanelConfig()
	cfg.ColorEnabled = false

	result := RenderBillingPanel(data, cfg)

	// Check title.
	if !strings.Contains(result, "Billing Dashboard") {
		t.Error("expected title 'Billing Dashboard' in output")
	}

	// Check total.
	if !strings.Contains(result, "70.50") {
		t.Error("expected total spend $70.50 in output")
	}

	// Check provider names.
	if !strings.Contains(result, "Civo") {
		t.Error("expected 'Civo' provider in output")
	}
	if !strings.Contains(result, "Digitalocean") {
		t.Error("expected 'Digitalocean' provider in output")
	}
}

func TestRenderBillingPanel_WithErrorProvider(t *testing.T) {
	data := BillingPanelData{
		Providers: []ProviderSpend{
			{
				Name:   "aws",
				Status: "error",
			},
		},
		TotalCurrent: 0,
	}
	cfg := DefaultBillingPanelConfig()
	cfg.ColorEnabled = false

	result := RenderBillingPanel(data, cfg)

	// Check error indicator.
	if !strings.Contains(result, "x") {
		t.Error("expected error indicator 'x' for failed provider")
	}
}

func TestRenderCompactBillingPanel_Basic(t *testing.T) {
	data := BillingPanelData{
		TotalCurrent: 100.00,
	}

	result := RenderCompactBillingPanel(data, false)

	if !strings.Contains(result, "$100") {
		t.Errorf("expected '$100' in compact output, got: %s", result)
	}
}

func TestRenderCompactBillingPanel_WithForecast(t *testing.T) {
	forecast := 200.0
	data := BillingPanelData{
		TotalCurrent:  100.00,
		TotalForecast: &forecast,
	}

	result := RenderCompactBillingPanel(data, false)

	if !strings.Contains(result, "forecast") {
		t.Errorf("expected 'forecast' in compact output, got: %s", result)
	}
	if !strings.Contains(result, "$200") {
		t.Errorf("expected '$200' forecast in output, got: %s", result)
	}
}

func TestRenderCompactBillingPanel_OverBudget(t *testing.T) {
	budget := 50.0
	data := BillingPanelData{
		TotalCurrent: 100.00,
		TotalBudget:  &budget,
	}

	result := RenderCompactBillingPanel(data, false)

	if !strings.Contains(result, "OVER BUDGET") {
		t.Errorf("expected 'OVER BUDGET' when spend exceeds budget, got: %s", result)
	}
}

func TestRenderCompactBillingPanel_WarningThreshold(t *testing.T) {
	budget := 100.0
	data := BillingPanelData{
		TotalCurrent: 92.00, // 92% of budget.
		TotalBudget:  &budget,
	}

	result := RenderCompactBillingPanel(data, false)

	if !strings.Contains(result, "92%") {
		t.Errorf("expected '92%%' in warning output, got: %s", result)
	}
}

func TestRenderCompactBillingPanel_UnderBudget(t *testing.T) {
	budget := 200.0
	data := BillingPanelData{
		TotalCurrent: 50.00, // 25% of budget.
		TotalBudget:  &budget,
	}

	result := RenderCompactBillingPanel(data, false)

	if !strings.Contains(result, "remaining") {
		t.Errorf("expected 'remaining' in under-budget output, got: %s", result)
	}
}

func TestCalculateForecast(t *testing.T) {
	tests := []struct {
		name         string
		currentSpend float64
		daysElapsed  int
		daysInMonth  int
		expected     float64
	}{
		{
			name:         "mid-month",
			currentSpend: 50.0,
			daysElapsed:  15,
			daysInMonth:  30,
			expected:     100.0, // 50/15 * 30 = 100
		},
		{
			name:         "first day",
			currentSpend: 10.0,
			daysElapsed:  1,
			daysInMonth:  30,
			expected:     300.0, // 10/1 * 30 = 300
		},
		{
			name:         "zero days",
			currentSpend: 10.0,
			daysElapsed:  0,
			daysInMonth:  30,
			expected:     10.0, // Returns current spend when 0 days.
		},
		{
			name:         "end of month",
			currentSpend: 100.0,
			daysElapsed:  30,
			daysInMonth:  30,
			expected:     100.0, // Already at month end.
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateForecast(tt.currentSpend, tt.daysElapsed, tt.daysInMonth)
			if result != tt.expected {
				t.Errorf("CalculateForecast(%v, %d, %d) = %v, want %v",
					tt.currentSpend, tt.daysElapsed, tt.daysInMonth, result, tt.expected)
			}
		})
	}
}

func TestRenderBillingPanel_GaugeThresholds(t *testing.T) {
	// Test gauge color thresholds.
	tests := []struct {
		name           string
		budgetPercent  float64
		expectContains string
	}{
		{
			name:           "under warning threshold",
			budgetPercent:  50.0, // 50% < 70%
			expectContains: "", // Green (no special indicator in non-color mode).
		},
		{
			name:           "at warning threshold",
			budgetPercent:  75.0, // 70-90% range.
			expectContains: "",
		},
		{
			name:           "at danger threshold",
			budgetPercent:  95.0, // >= 90%.
			expectContains: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			budget := 100.0
			current := tt.budgetPercent // Spend = budget percent.

			data := BillingPanelData{
				Providers: []ProviderSpend{
					{
						Name:    "test",
						Current: current,
						Budget:  &budget,
						Status:  "ok",
					},
				},
				TotalCurrent: current,
				TotalBudget:  &budget,
			}
			cfg := DefaultBillingPanelConfig()
			cfg.ColorEnabled = false

			result := RenderBillingPanel(data, cfg)

			// Just verify it renders without panicking.
			if len(result) == 0 {
				t.Error("expected non-empty output")
			}
		})
	}
}

func TestRenderBillingPanel_SparklineHistory(t *testing.T) {
	data := BillingPanelData{
		Providers: []ProviderSpend{
			{
				Name:    "civo",
				Current: 100.0,
				History: []float64{10, 20, 30, 40, 50, 60, 70, 80, 90, 100},
				Status:  "ok",
			},
		},
		TotalCurrent: 100.0,
	}
	cfg := DefaultBillingPanelConfig()
	cfg.ColorEnabled = false
	cfg.SparklineWidth = 10

	result := RenderBillingPanel(data, cfg)

	// Verify sparkline characters are present (unicode block chars).
	hasSparkline := false
	for _, r := range result {
		if r >= '\u2581' && r <= '\u2588' { // Sparkline block characters.
			hasSparkline = true
			break
		}
	}
	if !hasSparkline {
		t.Error("expected sparkline characters in output with history data")
	}
}

func TestRenderBillingPanel_NoSparklineWithoutHistory(t *testing.T) {
	data := BillingPanelData{
		Providers: []ProviderSpend{
			{
				Name:    "civo",
				Current: 100.0,
				History: nil, // No history.
				Status:  "ok",
			},
		},
		TotalCurrent: 100.0,
	}
	cfg := DefaultBillingPanelConfig()
	cfg.ColorEnabled = false

	result := RenderBillingPanel(data, cfg)

	// Should show placeholder dashes instead of sparkline.
	if !strings.Contains(result, "-") {
		t.Error("expected placeholder dashes when no history available")
	}
}
