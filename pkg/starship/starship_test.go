package starship

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gitlab.com/tinyland/lab/prompt-pulse/pkg/collectors/billing"
	"gitlab.com/tinyland/lab/prompt-pulse/pkg/collectors/claude"
	"gitlab.com/tinyland/lab/prompt-pulse/pkg/collectors/k8s"
	"gitlab.com/tinyland/lab/prompt-pulse/pkg/collectors/sysmetrics"
	"gitlab.com/tinyland/lab/prompt-pulse/pkg/collectors/tailscale"
)

// ssWriteFixture writes a JSON fixture to the given cache directory under
// the specified collector key.
func ssWriteFixture(t *testing.T, dir, key string, v interface{}) {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal fixture %s: %v", key, err)
	}
	path := filepath.Join(dir, key+".json")
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("write fixture %s: %v", key, err)
	}
}

// ssClaudeFixture builds a claude.UsageReport with the given total cost and
// models.
func ssClaudeFixture(totalCost float64, models []claude.ModelUsage) claude.UsageReport {
	return claude.UsageReport{
		TotalCostUSD: totalCost,
		Accounts: []claude.AccountUsage{
			{
				Name:      "test",
				Connected: true,
				CurrentMonth: claude.MonthUsage{
					CostUSD: totalCost,
				},
				Models: models,
			},
		},
		Timestamp: time.Now(),
	}
}

// ssBillingFixture builds a billing.BillingReport with the given monthly cost
// and budget.
func ssBillingFixture(monthly, budget float64) billing.BillingReport {
	pct := 0.0
	if budget > 0 {
		pct = (monthly / budget) * 100
	}
	return billing.BillingReport{
		TotalMonthlyUSD: monthly,
		BudgetUSD:       budget,
		BudgetPercent:   pct,
		Providers: []billing.ProviderBilling{
			{Name: "civo", Connected: true, MonthToDate: monthly},
		},
		Timestamp: time.Now(),
	}
}

// ssTailscaleFixture builds a tailscale.Status with the given peer counts.
func ssTailscaleFixture(online, total int) tailscale.Status {
	peers := make([]tailscale.PeerInfo, total)
	for i := 0; i < total; i++ {
		peers[i] = tailscale.PeerInfo{
			Hostname: "peer-" + string(rune('a'+i)),
			Online:   i < online,
		}
	}
	return tailscale.Status{
		OnlinePeers: online,
		TotalPeers:  total,
		Peers:       peers,
		Timestamp:   time.Now(),
	}
}

// ssK8sFixture builds a k8s.ClusterStatus with one cluster.
func ssK8sFixture(total, running, failed int) k8s.ClusterStatus {
	pending := total - running - failed
	if pending < 0 {
		pending = 0
	}
	return k8s.ClusterStatus{
		Clusters: []k8s.ClusterInfo{
			{
				Context:     "test",
				Connected:   true,
				TotalPods:   total,
				RunningPods: running,
				PendingPods: pending,
				FailedPods:  failed,
			},
		},
		Timestamp: time.Now(),
	}
}

// ssSysmetricsFixture builds a sysmetrics.Metrics with the given CPU and RAM
// percentages.
func ssSysmetricsFixture(cpuPct, ramPct float64) sysmetrics.Metrics {
	return sysmetrics.Metrics{
		CPU: sysmetrics.CPUMetrics{
			Total: cpuPct,
			Count: 8,
			Cores: []float64{cpuPct},
		},
		Memory: sysmetrics.MemoryMetrics{
			Total:       16 * 1024 * 1024 * 1024,
			Used:        uint64(ramPct / 100.0 * 16 * 1024 * 1024 * 1024),
			UsedPercent: ramPct,
		},
		Timestamp: time.Now(),
	}
}

// --- Tests ---

func TestRenderAllSegmentsEnabled(t *testing.T) {
	dir := t.TempDir()
	ssWriteFixture(t, dir, "claude", ssClaudeFixture(142.30, []claude.ModelUsage{
		{Model: "claude-opus-4-20250514", CostUSD: 100},
		{Model: "claude-3-5-sonnet-20241022", CostUSD: 42.30},
	}))
	ssWriteFixture(t, dir, "billing", ssBillingFixture(23.45, 100))
	ssWriteFixture(t, dir, "tailscale", ssTailscaleFixture(3, 5))
	ssWriteFixture(t, dir, "k8s", ssK8sFixture(15, 12, 0))
	ssWriteFixture(t, dir, "sysmetrics", ssSysmetricsFixture(45, 62))

	result := Render(Config{
		ShowClaude:    true,
		ShowBilling:   true,
		ShowTailscale: true,
		ShowK8s:       true,
		ShowSystem:    true,
		CacheDir:      dir,
		MaxWidth:      200, // wide enough for all
	})

	if result == "" {
		t.Fatal("expected non-empty render with all segments, got empty")
	}

	// Verify key pieces appear in the stripped output.
	stripped := ssStripAnsi(result)
	for _, want := range []string{"$142.30", "$23.45/mo", "3/5 peers", "12/15 pods", "CPU:45%", "RAM:62%"} {
		if !strings.Contains(stripped, want) {
			t.Errorf("expected %q in output, got: %s", want, stripped)
		}
	}
}

func TestRenderNoCachedData(t *testing.T) {
	dir := t.TempDir() // empty directory

	result := Render(Config{
		ShowClaude:    true,
		ShowBilling:   true,
		ShowTailscale: true,
		ShowK8s:       true,
		ShowSystem:    true,
		CacheDir:      dir,
	})

	if result != "" {
		t.Errorf("expected empty render with no cached data, got: %q", result)
	}
}

func TestRenderOnlyClaudeEnabled(t *testing.T) {
	dir := t.TempDir()
	ssWriteFixture(t, dir, "claude", ssClaudeFixture(50.0, []claude.ModelUsage{
		{Model: "claude-3-5-sonnet-20241022", CostUSD: 50},
	}))
	// Write other fixtures too, but only Claude is enabled.
	ssWriteFixture(t, dir, "billing", ssBillingFixture(10, 100))
	ssWriteFixture(t, dir, "sysmetrics", ssSysmetricsFixture(30, 40))

	result := Render(Config{
		ShowClaude: true,
		CacheDir:   dir,
		MaxWidth:   200,
	})

	if result == "" {
		t.Fatal("expected non-empty render for Claude-only, got empty")
	}

	stripped := ssStripAnsi(result)
	if !strings.Contains(stripped, "$50.00") {
		t.Errorf("expected Claude cost in output, got: %s", stripped)
	}
	if strings.Contains(stripped, "$10.00") {
		t.Errorf("billing should not appear when ShowBilling is false, got: %s", stripped)
	}
	if strings.Contains(stripped, "CPU:") {
		t.Errorf("system should not appear when ShowSystem is false, got: %s", stripped)
	}
}

func TestRenderRespectsMaxWidthTruncation(t *testing.T) {
	dir := t.TempDir()
	ssWriteFixture(t, dir, "claude", ssClaudeFixture(142.30, []claude.ModelUsage{
		{Model: "claude-opus-4-20250514", CostUSD: 142.30},
	}))
	ssWriteFixture(t, dir, "billing", ssBillingFixture(23.45, 100))
	ssWriteFixture(t, dir, "tailscale", ssTailscaleFixture(3, 5))
	ssWriteFixture(t, dir, "k8s", ssK8sFixture(15, 12, 0))
	ssWriteFixture(t, dir, "sysmetrics", ssSysmetricsFixture(45, 62))

	// Very narrow width: should drop rightmost segments.
	result := Render(Config{
		ShowClaude:    true,
		ShowBilling:   true,
		ShowTailscale: true,
		ShowK8s:       true,
		ShowSystem:    true,
		CacheDir:      dir,
		MaxWidth:      30,
	})

	stripped := ssStripAnsi(result)
	visWidth := len([]rune(stripped))
	if visWidth > 30 {
		t.Errorf("visible width %d exceeds maxWidth 30, output: %s", visWidth, stripped)
	}

	// With a width of 30, we should have at least the first segment.
	if result == "" {
		t.Error("expected at least one segment with maxWidth=30")
	}
}

func TestClaudeSegmentFormatting(t *testing.T) {
	dir := t.TempDir()
	ssWriteFixture(t, dir, "claude", ssClaudeFixture(142.30, []claude.ModelUsage{
		{Model: "claude-opus-4-20250514", CostUSD: 100},
		{Model: "claude-3-5-sonnet-20241022", CostUSD: 42.30},
	}))

	seg := ssClaudeSegment(dir)
	if seg == nil {
		t.Fatal("expected non-nil segment")
	}
	if seg.Icon != "🤖" {
		t.Errorf("expected icon 🤖, got %s", seg.Icon)
	}
	if !strings.Contains(seg.Text, "$142.30") {
		t.Errorf("expected cost in text, got: %s", seg.Text)
	}
	if !strings.Contains(seg.Text, "opus") {
		t.Errorf("expected top model 'opus' in text, got: %s", seg.Text)
	}
}

func TestClaudeSegmentColorThresholds(t *testing.T) {
	tests := []struct {
		name      string
		cost      float64
		wantColor string
	}{
		{"green_under_50pct", 100.0, ssColorGreen},   // 100/500 = 20%
		{"yellow_at_50pct", 250.0, ssColorYellow},     // 250/500 = 50%
		{"yellow_at_70pct", 350.0, ssColorYellow},     // 350/500 = 70%
		{"red_at_80pct", 400.0, ssColorRed},           // 400/500 = 80%
		{"red_over_budget", 600.0, ssColorRed},        // 600/500 = 120%
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			ssWriteFixture(t, dir, "claude", ssClaudeFixture(tt.cost, []claude.ModelUsage{
				{Model: "claude-opus-4-20250514", CostUSD: tt.cost},
			}))
			seg := ssClaudeSegment(dir)
			if seg == nil {
				t.Fatal("expected non-nil segment")
			}
			if seg.Color != tt.wantColor {
				t.Errorf("cost=%.2f: want color %q, got %q", tt.cost, tt.wantColor, seg.Color)
			}
		})
	}
}

func TestBillingSegmentFormatting(t *testing.T) {
	dir := t.TempDir()
	ssWriteFixture(t, dir, "billing", ssBillingFixture(23.45, 100))

	seg := ssBillingSegment(dir)
	if seg == nil {
		t.Fatal("expected non-nil segment")
	}
	if seg.Icon != "☁️" {
		t.Errorf("expected icon ☁️, got %s", seg.Icon)
	}
	if seg.Text != "$23.45/mo" {
		t.Errorf("expected text '$23.45/mo', got: %s", seg.Text)
	}
	// 23.45/100 = 23.45% -> green
	if seg.Color != ssColorGreen {
		t.Errorf("expected green, got %q", seg.Color)
	}
}

func TestTailscaleSegmentAllOnline(t *testing.T) {
	dir := t.TempDir()
	ssWriteFixture(t, dir, "tailscale", ssTailscaleFixture(5, 5))

	seg := ssTailscaleSegment(dir)
	if seg == nil {
		t.Fatal("expected non-nil segment")
	}
	if seg.Text != "5/5 peers" {
		t.Errorf("expected '5/5 peers', got: %s", seg.Text)
	}
	if seg.Color != ssColorGreen {
		t.Errorf("expected green for all online, got %q", seg.Color)
	}
}

func TestTailscaleSegmentSomeOffline(t *testing.T) {
	dir := t.TempDir()
	ssWriteFixture(t, dir, "tailscale", ssTailscaleFixture(3, 5))

	seg := ssTailscaleSegment(dir)
	if seg == nil {
		t.Fatal("expected non-nil segment")
	}
	if seg.Text != "3/5 peers" {
		t.Errorf("expected '3/5 peers', got: %s", seg.Text)
	}
	// 3/5 = 60% -> yellow
	if seg.Color != ssColorYellow {
		t.Errorf("expected yellow for 60%% online, got %q", seg.Color)
	}
}

func TestTailscaleSegmentMostOffline(t *testing.T) {
	dir := t.TempDir()
	ssWriteFixture(t, dir, "tailscale", ssTailscaleFixture(1, 5))

	seg := ssTailscaleSegment(dir)
	if seg == nil {
		t.Fatal("expected non-nil segment")
	}
	// 1/5 = 20% -> red
	if seg.Color != ssColorRed {
		t.Errorf("expected red for <50%% online, got %q", seg.Color)
	}
}

func TestK8sSegmentHealthyPods(t *testing.T) {
	dir := t.TempDir()
	ssWriteFixture(t, dir, "k8s", ssK8sFixture(15, 15, 0))

	seg := ssK8sSegment(dir)
	if seg == nil {
		t.Fatal("expected non-nil segment")
	}
	if seg.Text != "15/15 pods" {
		t.Errorf("expected '15/15 pods', got: %s", seg.Text)
	}
	if seg.Color != ssColorGreen {
		t.Errorf("expected green for all healthy, got %q", seg.Color)
	}
}

func TestK8sSegmentFailedPods(t *testing.T) {
	dir := t.TempDir()
	ssWriteFixture(t, dir, "k8s", ssK8sFixture(15, 10, 3))

	seg := ssK8sSegment(dir)
	if seg == nil {
		t.Fatal("expected non-nil segment")
	}
	if seg.Text != "10/15 pods" {
		t.Errorf("expected '10/15 pods', got: %s", seg.Text)
	}
	if seg.Color != ssColorRed {
		t.Errorf("expected red for failed pods, got %q", seg.Color)
	}
}

func TestK8sSegmentPendingPods(t *testing.T) {
	dir := t.TempDir()
	ssWriteFixture(t, dir, "k8s", ssK8sFixture(15, 12, 0))

	seg := ssK8sSegment(dir)
	if seg == nil {
		t.Fatal("expected non-nil segment")
	}
	// 12 running out of 15 total, 0 failed -> yellow (some pending)
	if seg.Color != ssColorYellow {
		t.Errorf("expected yellow for pending pods, got %q", seg.Color)
	}
}

func TestSystemSegmentNormalValues(t *testing.T) {
	dir := t.TempDir()
	ssWriteFixture(t, dir, "sysmetrics", ssSysmetricsFixture(30, 40))

	seg := ssSystemSegment(dir)
	if seg == nil {
		t.Fatal("expected non-nil segment")
	}
	if seg.Text != "CPU:30% RAM:40%" {
		t.Errorf("expected 'CPU:30%% RAM:40%%', got: %s", seg.Text)
	}
	if seg.Color != ssColorGreen {
		t.Errorf("expected green for normal values, got %q", seg.Color)
	}
}

func TestSystemSegmentHighCPU(t *testing.T) {
	dir := t.TempDir()
	ssWriteFixture(t, dir, "sysmetrics", ssSysmetricsFixture(92, 40))

	seg := ssSystemSegment(dir)
	if seg == nil {
		t.Fatal("expected non-nil segment")
	}
	if seg.Color != ssColorRed {
		t.Errorf("expected red for high CPU, got %q", seg.Color)
	}
}

func TestSystemSegmentHighRAM(t *testing.T) {
	dir := t.TempDir()
	ssWriteFixture(t, dir, "sysmetrics", ssSysmetricsFixture(30, 85))

	seg := ssSystemSegment(dir)
	if seg == nil {
		t.Fatal("expected non-nil segment")
	}
	if seg.Color != ssColorRed {
		t.Errorf("expected red for high RAM, got %q", seg.Color)
	}
}

func TestFormatLineJoinsWithSeparator(t *testing.T) {
	segments := []*Segment{
		{Icon: "A", Text: "one", Color: ""},
		{Icon: "B", Text: "two", Color: ""},
	}
	result := ssFormatLine(segments, 200)
	stripped := ssStripAnsi(result)

	if !strings.Contains(stripped, "A one") {
		t.Errorf("expected 'A one' in output, got: %s", stripped)
	}
	if !strings.Contains(stripped, "B two") {
		t.Errorf("expected 'B two' in output, got: %s", stripped)
	}
	// The separator character should be present.
	if !strings.Contains(result, "│") {
		t.Errorf("expected separator │ in output, got: %s", result)
	}
}

func TestFormatLineDropsSegmentsExceedingMaxWidth(t *testing.T) {
	segments := []*Segment{
		{Icon: "A", Text: "short", Color: ""},
		{Icon: "B", Text: "medium-text", Color: ""},
		{Icon: "C", Text: "this-is-very-long-segment-text", Color: ""},
	}

	// Set width that allows first two but not third.
	// "A short" = 7, " │ " = 3, "B medium-text" = 13 => 23
	result := ssFormatLine(segments, 25)
	stripped := ssStripAnsi(result)

	if !strings.Contains(stripped, "A short") {
		t.Errorf("expected first segment, got: %s", stripped)
	}
	if !strings.Contains(stripped, "B medium-text") {
		t.Errorf("expected second segment, got: %s", stripped)
	}
	if strings.Contains(stripped, "this-is-very-long") {
		t.Errorf("expected third segment to be dropped, got: %s", stripped)
	}
}

func TestFormatLineEmptySegments(t *testing.T) {
	result := ssFormatLine(nil, 60)
	if result != "" {
		t.Errorf("expected empty string for nil segments, got: %q", result)
	}

	result = ssFormatLine([]*Segment{}, 60)
	if result != "" {
		t.Errorf("expected empty string for empty segments, got: %q", result)
	}
}

func TestColorizeWrapsText(t *testing.T) {
	result := ssColorize("hello", ssColorGreen)
	if !strings.HasPrefix(result, ssColorGreen) {
		t.Errorf("expected green prefix, got: %q", result)
	}
	if !strings.HasSuffix(result, ssAnsiReset) {
		t.Errorf("expected reset suffix, got: %q", result)
	}
	if !strings.Contains(result, "hello") {
		t.Errorf("expected text 'hello' in output, got: %q", result)
	}
}

func TestColorizeEmptyColor(t *testing.T) {
	result := ssColorize("hello", "")
	if result != "hello" {
		t.Errorf("expected bare text with empty color, got: %q", result)
	}
}

func TestConfigDefaultMaxWidth(t *testing.T) {
	dir := t.TempDir()
	ssWriteFixture(t, dir, "claude", ssClaudeFixture(10, []claude.ModelUsage{
		{Model: "claude-opus-4-20250514", CostUSD: 10},
	}))

	// MaxWidth=0 should use default (60).
	cfg := Config{
		ShowClaude: true,
		CacheDir:   dir,
		MaxWidth:   0,
	}
	result := Render(cfg)
	if result == "" {
		t.Fatal("expected non-empty render with default maxWidth")
	}

	stripped := ssStripAnsi(result)
	visWidth := len([]rune(stripped))
	if visWidth > ssDefaultMaxWidth {
		t.Errorf("visible width %d exceeds default maxWidth %d", visWidth, ssDefaultMaxWidth)
	}
}

func TestCacheReaderStaleFile(t *testing.T) {
	dir := t.TempDir()
	ssWriteFixture(t, dir, "claude", ssClaudeFixture(10, nil))

	// Backdate the file to make it stale.
	path := filepath.Join(dir, "claude.json")
	old := time.Now().Add(-10 * time.Minute)
	if err := os.Chtimes(path, old, old); err != nil {
		t.Fatalf("chtimes: %v", err)
	}

	result, err := ssReadCachedData[claude.UsageReport](dir, "claude")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Error("expected nil for stale cache file, got non-nil")
	}
}

func TestCacheReaderMissingFile(t *testing.T) {
	dir := t.TempDir()
	result, err := ssReadCachedData[claude.UsageReport](dir, "nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Error("expected nil for missing file, got non-nil")
	}
}

func TestCacheReaderInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "claude.json")
	if err := os.WriteFile(path, []byte("not json"), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := ssReadCachedData[claude.UsageReport](dir, "claude")
	if err == nil {
		t.Error("expected error for invalid JSON, got nil")
	}
	if result != nil {
		t.Error("expected nil result for invalid JSON")
	}
}

func TestShortModelName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"claude-opus-4-20250514", "opus"},
		{"claude-3-5-sonnet-20241022", "sonnet"},
		{"claude-3-haiku-20240307", "haiku"},
		{"", ""},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := ssShortModelName(tt.input)
			if got != tt.want {
				t.Errorf("ssShortModelName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestStripAnsi(t *testing.T) {
	colored := "\033[32mhello\033[0m world"
	stripped := ssStripAnsi(colored)
	if stripped != "hello world" {
		t.Errorf("expected 'hello world', got: %q", stripped)
	}
}

func TestVisibleWidth(t *testing.T) {
	plain := "hello"
	if w := ssVisibleWidth(plain); w != 5 {
		t.Errorf("expected 5, got %d", w)
	}

	colored := "\033[32mhello\033[0m"
	if w := ssVisibleWidth(colored); w != 5 {
		t.Errorf("expected 5 for colored text, got %d", w)
	}
}
