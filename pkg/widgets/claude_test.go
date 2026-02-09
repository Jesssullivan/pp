package widgets

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"gitlab.com/tinyland/lab/prompt-pulse/pkg/app"
	"gitlab.com/tinyland/lab/prompt-pulse/pkg/collectors/claude"
)

// --- helpers to build test data ---

func claudeTestReport(accounts ...claude.AccountUsage) *claude.UsageReport {
	total := 0.0
	for _, a := range accounts {
		total += a.CurrentMonth.CostUSD
	}
	return &claude.UsageReport{
		Accounts:     accounts,
		TotalCostUSD: total,
		Timestamp:    time.Date(2026, 2, 9, 12, 0, 0, 0, time.UTC),
	}
}

func claudeTestAccount(name string, inputTokens, outputTokens int64, cost float64, models []claude.ModelUsage) claude.AccountUsage {
	return claude.AccountUsage{
		Name:      name,
		Connected: true,
		CurrentMonth: claude.MonthUsage{
			InputTokens:  inputTokens,
			OutputTokens: outputTokens,
			CostUSD:      cost,
		},
		Models: models,
	}
}

func claudeTestDisconnectedAccount(name string) claude.AccountUsage {
	return claude.AccountUsage{
		Name:      name,
		Connected: false,
		Error:     "connection refused",
	}
}

func claudeTestModels() []claude.ModelUsage {
	return []claude.ModelUsage{
		{Model: "claude-opus-4-6", InputTokens: 500_000, OutputTokens: 200_000, CostUSD: 22.50},
		{Model: "claude-sonnet-4-5", InputTokens: 2_000_000, OutputTokens: 800_000, CostUSD: 18.00},
		{Model: "claude-haiku-4-5", InputTokens: 5_000_000, OutputTokens: 1_000_000, CostUSD: 8.00},
	}
}

// --- test cases ---

func TestClaudeWidget_ID(t *testing.T) {
	w := NewClaudeWidget()
	if got := w.ID(); got != "claude" {
		t.Errorf("ID() = %q, want %q", got, "claude")
	}
}

func TestClaudeWidget_Title(t *testing.T) {
	w := NewClaudeWidget()
	if got := w.Title(); got != "Claude Usage" {
		t.Errorf("Title() = %q, want %q", got, "Claude Usage")
	}
}

func TestClaudeWidget_MinSize_Compact(t *testing.T) {
	w := NewClaudeWidget()
	minW, minH := w.MinSize()
	if minW != 30 || minH != 5 {
		t.Errorf("MinSize() compact = (%d, %d), want (30, 5)", minW, minH)
	}
}

func TestClaudeWidget_MinSize_Expanded(t *testing.T) {
	w := NewClaudeWidget()
	w.expanded = true
	minW, minH := w.MinSize()
	if minW != 30 || minH != 15 {
		t.Errorf("MinSize() expanded = (%d, %d), want (30, 15)", minW, minH)
	}
}

func TestClaudeWidget_View_NoData(t *testing.T) {
	w := NewClaudeWidget()
	view := w.View(40, 5)
	if !strings.Contains(view, "No data") {
		t.Errorf("View with no data should contain 'No data', got:\n%s", view)
	}
}

func TestClaudeWidget_View_CompactMode(t *testing.T) {
	w := NewClaudeWidget()
	report := claudeTestReport(
		claudeTestAccount("personal", 3_000_000, 1_000_000, 45.50, nil),
	)
	w.Update(app.DataUpdateEvent{Source: "claude", Data: report})

	view := w.View(60, 10)
	lines := strings.Split(view, "\n")

	if len(lines) != 10 {
		t.Errorf("expected 10 lines, got %d", len(lines))
	}

	// Should contain the account name.
	if !strings.Contains(view, "personal") {
		t.Errorf("compact view should contain account name 'personal'")
	}

	// Should contain cost.
	if !strings.Contains(view, "$45.50") {
		t.Errorf("compact view should contain cost '$45.50'")
	}

	// Should contain total.
	if !strings.Contains(view, "Total:") {
		t.Errorf("compact view should contain 'Total:'")
	}
}

func TestClaudeWidget_View_ExpandedMode(t *testing.T) {
	w := NewClaudeWidget()
	w.expanded = true
	models := claudeTestModels()
	report := claudeTestReport(
		claudeTestAccount("work", 7_500_000, 2_000_000, 48.50, models),
	)
	w.Update(app.DataUpdateEvent{Source: "claude", Data: report})

	view := w.View(60, 20)

	// Should contain per-model names.
	if !strings.Contains(view, "Opus 4.6") {
		t.Errorf("expanded view should contain 'Opus 4.6'")
	}
	if !strings.Contains(view, "Sonnet 4.5") {
		t.Errorf("expanded view should contain 'Sonnet 4.5'")
	}
	if !strings.Contains(view, "Haiku 4.5") {
		t.Errorf("expanded view should contain 'Haiku 4.5'")
	}

	// Should contain per-model costs.
	if !strings.Contains(view, "$22.50") {
		t.Errorf("expanded view should contain model cost '$22.50'")
	}

	// Should contain headroom indicator.
	if !strings.Contains(view, "headroom") {
		t.Errorf("expanded view should contain 'headroom'")
	}
}

func TestClaudeWidget_Update_WithUsageReport(t *testing.T) {
	w := NewClaudeWidget()

	report := claudeTestReport(
		claudeTestAccount("test", 1_000_000, 500_000, 12.34, nil),
	)

	cmd := w.Update(app.DataUpdateEvent{
		Source: "claude",
		Data:   report,
	})

	if cmd != nil {
		t.Errorf("Update should return nil cmd, got non-nil")
	}
	if w.report == nil {
		t.Fatal("report should be set after Update")
	}
	if w.report.TotalCostUSD != 12.34 {
		t.Errorf("TotalCostUSD = %f, want 12.34", w.report.TotalCostUSD)
	}
	if len(w.costHistory) != 1 {
		t.Errorf("costHistory len = %d, want 1", len(w.costHistory))
	}
}

func TestClaudeWidget_Update_IgnoresOtherSources(t *testing.T) {
	w := NewClaudeWidget()

	w.Update(app.DataUpdateEvent{
		Source: "tailscale",
		Data:   "something",
	})

	if w.report != nil {
		t.Errorf("report should remain nil for non-claude source")
	}
}

func TestClaudeWidget_Update_IgnoresNilData(t *testing.T) {
	w := NewClaudeWidget()

	w.Update(app.DataUpdateEvent{
		Source: "claude",
		Data:   nil,
	})

	if w.report != nil {
		t.Errorf("report should remain nil for nil data")
	}
}

func TestClaudeWidget_HandleKey_ToggleExpanded(t *testing.T) {
	w := NewClaudeWidget()
	if w.expanded {
		t.Fatal("should start compact")
	}

	w.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	if !w.expanded {
		t.Error("after pressing 'e', should be expanded")
	}

	w.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	if w.expanded {
		t.Error("after pressing 'e' again, should be compact")
	}
}

func TestClaudeWidget_HandleKey_CycleAccount(t *testing.T) {
	w := NewClaudeWidget()
	report := claudeTestReport(
		claudeTestAccount("acct-a", 100, 100, 1.0, nil),
		claudeTestAccount("acct-b", 200, 200, 2.0, nil),
		claudeTestAccount("acct-c", 300, 300, 3.0, nil),
	)
	w.Update(app.DataUpdateEvent{Source: "claude", Data: report})

	if w.selectedAccount != 0 {
		t.Fatalf("should start at account 0, got %d", w.selectedAccount)
	}

	w.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})
	if w.selectedAccount != 1 {
		t.Errorf("after 'm', selectedAccount = %d, want 1", w.selectedAccount)
	}

	w.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})
	if w.selectedAccount != 2 {
		t.Errorf("after second 'm', selectedAccount = %d, want 2", w.selectedAccount)
	}

	// Wraps around.
	w.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})
	if w.selectedAccount != 0 {
		t.Errorf("after third 'm', selectedAccount = %d, want 0 (wrap)", w.selectedAccount)
	}
}

func TestClaudeWidget_View_SmallSize(t *testing.T) {
	w := NewClaudeWidget()
	report := claudeTestReport(
		claudeTestAccount("tiny", 100, 100, 0.01, nil),
	)
	w.Update(app.DataUpdateEvent{Source: "claude", Data: report})

	// Minimum viable size.
	view := w.View(30, 5)
	lines := strings.Split(view, "\n")
	if len(lines) != 5 {
		t.Errorf("expected 5 lines for 30x5, got %d", len(lines))
	}
}

func TestClaudeWidget_View_LargeSize(t *testing.T) {
	w := NewClaudeWidget()
	w.expanded = true
	report := claudeTestReport(
		claudeTestAccount("large", 5_000_000, 2_000_000, 120.00, claudeTestModels()),
	)
	w.Update(app.DataUpdateEvent{Source: "claude", Data: report})

	view := w.View(80, 24)
	lines := strings.Split(view, "\n")
	if len(lines) != 24 {
		t.Errorf("expected 24 lines for 80x24, got %d", len(lines))
	}
}

func TestClaudeWidget_View_MediumSize(t *testing.T) {
	w := NewClaudeWidget()
	report := claudeTestReport(
		claudeTestAccount("mid", 1_000_000, 500_000, 25.00, nil),
	)
	w.Update(app.DataUpdateEvent{Source: "claude", Data: report})

	view := w.View(60, 15)
	lines := strings.Split(view, "\n")
	if len(lines) != 15 {
		t.Errorf("expected 15 lines for 60x15, got %d", len(lines))
	}
}

func TestClaudeWidget_MultiAccount_Rendering(t *testing.T) {
	w := NewClaudeWidget()
	report := claudeTestReport(
		claudeTestAccount("personal", 1_000_000, 500_000, 12.50, nil),
		claudeTestAccount("work", 5_000_000, 2_000_000, 85.00, nil),
	)
	w.Update(app.DataUpdateEvent{Source: "claude", Data: report})

	view := w.View(60, 15)

	if !strings.Contains(view, "personal") {
		t.Error("multi-account view should contain 'personal'")
	}
	if !strings.Contains(view, "work") {
		t.Error("multi-account view should contain 'work'")
	}
	if !strings.Contains(view, "$12.50") {
		t.Error("multi-account view should contain '$12.50'")
	}
	if !strings.Contains(view, "$85.00") {
		t.Error("multi-account view should contain '$85.00'")
	}
	if !strings.Contains(view, "Total: $97.50") {
		t.Error("multi-account view should show total '$97.50'")
	}
}

func TestClaudeWidget_DisconnectedAccount(t *testing.T) {
	w := NewClaudeWidget()
	report := claudeTestReport(
		claudeTestDisconnectedAccount("broken"),
	)
	w.Update(app.DataUpdateEvent{Source: "claude", Data: report})

	view := w.View(60, 10)
	if !strings.Contains(view, "disconnected") {
		t.Errorf("disconnected account should show 'disconnected', got:\n%s", view)
	}
}

func TestClaudeWidget_CostColorCoding(t *testing.T) {
	tests := []struct {
		name      string
		ratio     float64
		wantColor string
	}{
		{"green - low usage", 0.30, claudeColorGreen},
		{"green - border", 0.49, claudeColorGreen},
		{"yellow - mid usage", 0.60, claudeColorYellow},
		{"yellow - border", 0.79, claudeColorYellow},
		{"red - high usage", 0.90, claudeColorRed},
		{"red - over budget", 1.0, claudeColorRed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := claudeRatioColor(tt.ratio)
			if got != tt.wantColor {
				t.Errorf("claudeRatioColor(%f) = %q, want %q", tt.ratio, got, tt.wantColor)
			}
		})
	}
}

func TestClaudeWidget_SparklineFromCostHistory(t *testing.T) {
	w := NewClaudeWidget()

	// Send multiple updates to build cost history.
	for i := 0; i < 5; i++ {
		cost := float64(i+1) * 10.0
		report := claudeTestReport(
			claudeTestAccount("test", 1000, 1000, cost, nil),
		)
		w.Update(app.DataUpdateEvent{Source: "claude", Data: report})
	}

	if len(w.costHistory) != 5 {
		t.Fatalf("costHistory len = %d, want 5", len(w.costHistory))
	}
	if w.costHistory[0] != 10.0 {
		t.Errorf("costHistory[0] = %f, want 10.0", w.costHistory[0])
	}
	if w.costHistory[4] != 50.0 {
		t.Errorf("costHistory[4] = %f, want 50.0", w.costHistory[4])
	}

	// Render should include sparkline.
	view := w.View(60, 10)
	if !strings.Contains(view, "Cost") {
		t.Error("view with cost history should render sparkline with 'Cost' label")
	}
}

func TestClaudeWidget_FormatTokens(t *testing.T) {
	tests := []struct {
		tokens int64
		want   string
	}{
		{0, "0"},
		{500, "500"},
		{1_500, "1.5K"},
		{1_000_000, "1.0M"},
		{3_400_000, "3.4M"},
		{1_200_000_000, "1.2G"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := claudeFormatTokens(tt.tokens)
			if got != tt.want {
				t.Errorf("claudeFormatTokens(%d) = %q, want %q", tt.tokens, got, tt.want)
			}
		})
	}
}

func TestClaudeWidget_ShortModelName(t *testing.T) {
	tests := []struct {
		model string
		want  string
	}{
		{"claude-opus-4-6", "Opus 4.6"},
		{"claude-sonnet-4-5", "Sonnet 4.5"},
		{"claude-haiku-4-5", "Haiku 4.5"},
		{"claude-3-opus", "Opus 3"},
		{"claude-3-5-sonnet", "Sonnet 3.5"},
		{"unknown-model", "unknown-model"},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			got := claudeShortModelName(tt.model)
			if got != tt.want {
				t.Errorf("claudeShortModelName(%q) = %q, want %q", tt.model, got, tt.want)
			}
		})
	}
}

func TestClaudeWidget_CostHistoryMaxLength(t *testing.T) {
	w := NewClaudeWidget()

	// Send more updates than the max history.
	for i := 0; i < 40; i++ {
		report := claudeTestReport(
			claudeTestAccount("test", 100, 100, float64(i), nil),
		)
		w.Update(app.DataUpdateEvent{Source: "claude", Data: report})
	}

	if len(w.costHistory) != claudeMaxCostHistory {
		t.Errorf("costHistory len = %d, want %d (max)", len(w.costHistory), claudeMaxCostHistory)
	}

	// Should keep the latest values (10..39).
	if w.costHistory[0] != 10.0 {
		t.Errorf("oldest kept value = %f, want 10.0", w.costHistory[0])
	}
	if w.costHistory[len(w.costHistory)-1] != 39.0 {
		t.Errorf("newest value = %f, want 39.0", w.costHistory[len(w.costHistory)-1])
	}
}

func TestClaudeWidget_View_ZeroDimensions(t *testing.T) {
	w := NewClaudeWidget()
	if v := w.View(0, 10); v != "" {
		t.Errorf("View(0, 10) should return empty, got %q", v)
	}
	if v := w.View(10, 0); v != "" {
		t.Errorf("View(10, 0) should return empty, got %q", v)
	}
}

func TestClaudeWidget_HandleKey_UnknownKey(t *testing.T) {
	w := NewClaudeWidget()
	cmd := w.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'z'}})
	if cmd != nil {
		t.Errorf("unknown key should return nil cmd")
	}
}

func TestClaudeWidget_ExpandedDisconnectedShowsError(t *testing.T) {
	w := NewClaudeWidget()
	w.expanded = true
	report := claudeTestReport(
		claudeTestDisconnectedAccount("bad-account"),
	)
	w.Update(app.DataUpdateEvent{Source: "claude", Data: report})

	view := w.View(60, 15)
	if !strings.Contains(view, "connection refused") {
		t.Errorf("expanded disconnected view should show error message")
	}
}
