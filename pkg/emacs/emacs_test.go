package emacs

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"gitlab.com/tinyland/lab/prompt-pulse/pkg/cache"
)

// emSetupTestCache creates a temporary cache directory and returns its path.
func emSetupTestCache(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	return dir
}

// emWriteFixture stores a JSON fixture via the cache Store API so that
// it is retrievable by cache.GetTyped (which uses hashed filenames).
func emWriteFixture(t *testing.T, dir, key string, data any) {
	t.Helper()
	store, err := emOpenStore(dir)
	if err != nil {
		t.Fatalf("open store for fixture %s: %v", key, err)
	}
	defer store.Close()
	if err := cache.PutTyped(store, key, data); err != nil {
		t.Fatalf("put fixture %s: %v", key, err)
	}
}

// emSampleClaude returns sample Claude usage data.
func emSampleClaude() *emClaudeUsage {
	return &emClaudeUsage{
		Accounts: []emClaudeAccount{
			{
				Name:   "personal",
				Type:   "subscription",
				Tier:   "max_5x",
				Status: "ok",
				FiveHour: &emUsagePeriod{
					Utilization: 45.0,
				},
			},
			{
				Name:   "work",
				Type:   "api",
				Tier:   "tier_2",
				Status: "active",
				RateLimits: &emAPIRateLimits{
					RequestsLimit:     1000,
					RequestsRemaining: 700,
				},
			},
		},
	}
}

// emSampleBilling returns sample billing data.
func emSampleBilling() *emBillingData {
	budget := 100.0
	return &emBillingData{
		Providers: []emProviderBilling{
			{
				Provider:    "civo",
				AccountName: "tinyland",
				Status:      "ok",
				CurrentMonth: emMonthCost{
					SpendUSD: 12.50,
				},
			},
			{
				Provider:    "digitalocean",
				AccountName: "tinyland",
				Status:      "ok",
				CurrentMonth: emMonthCost{
					SpendUSD: 10.95,
				},
			},
			{
				Provider:    "dreamhost",
				AccountName: "tinyland",
				Status:      "error",
			},
		},
		Total: emBillingSummary{
			CurrentMonthUSD: 23.45,
			BudgetUSD:       &budget,
			SuccessCount:    2,
			ErrorCount:      1,
			TotalConfigured: 3,
		},
	}
}

// emSampleInfra returns sample infrastructure data with both Tailscale and K8s.
func emSampleInfra() *emInfraStatus {
	return &emInfraStatus{
		Tailscale: &emTailscaleStatus{
			Tailnet:     "tinyland.ts.net",
			OnlineCount: 3,
			TotalCount:  5,
		},
		Kubernetes: []emKubernetesCluster{
			{
				Name:        "prod",
				Status:      "healthy",
				TotalPods:   15,
				RunningPods: 12,
				TotalNodes:  3,
				ReadyNodes:  3,
			},
		},
	}
}

// emSampleSysMetricsData returns sample system metrics data.
func emSampleSysMetricsData() *emSysMetrics {
	return &emSysMetrics{
		CPU:       45.2,
		RAM:       62.8,
		Disk:      55.0,
		LoadAvg1:  1.5,
		LoadAvg5:  1.2,
		LoadAvg15: 0.9,
	}
}

// --- RenderJSON tests ---

func TestRenderJSON_NoData(t *testing.T) {
	dir := emSetupTestCache(t)
	b, err := RenderJSON(dir, "")
	if err != nil {
		t.Fatalf("RenderJSON: %v", err)
	}

	var out JSONOutput
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if out.Version != Version {
		t.Errorf("version = %q, want %q", out.Version, Version)
	}
	if len(out.Widgets) != 0 {
		t.Errorf("widgets = %d, want 0", len(out.Widgets))
	}
}

func TestRenderJSON_AllData(t *testing.T) {
	dir := emSetupTestCache(t)
	emWriteFixture(t, dir, "claude", emSampleClaude())
	emWriteFixture(t, dir, "billing", emSampleBilling())
	emWriteFixture(t, dir, "infra", emSampleInfra())
	emWriteFixture(t, dir, "sysmetrics", emSampleSysMetricsData())

	b, err := RenderJSON(dir, "")
	if err != nil {
		t.Fatalf("RenderJSON: %v", err)
	}

	var out JSONOutput
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	// Expect claude, billing, tailscale, k8s, system = 5 widgets.
	if len(out.Widgets) != 5 {
		t.Errorf("widgets = %d, want 5", len(out.Widgets))
		for _, w := range out.Widgets {
			t.Logf("  widget: %s", w.ID)
		}
	}
}

func TestRenderJSON_VersionAndTimestamp(t *testing.T) {
	dir := emSetupTestCache(t)
	b, err := RenderJSON(dir, "")
	if err != nil {
		t.Fatalf("RenderJSON: %v", err)
	}

	var out JSONOutput
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if out.Version != Version {
		t.Errorf("version = %q, want %q", out.Version, Version)
	}
	if out.Timestamp == "" {
		t.Error("timestamp is empty")
	}
}

func TestRenderJSON_WaifuPathPresent(t *testing.T) {
	dir := emSetupTestCache(t)
	waifuPath := "/tmp/waifu.png"
	b, err := RenderJSON(dir, waifuPath)
	if err != nil {
		t.Fatalf("RenderJSON: %v", err)
	}

	var out JSONOutput
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if out.WaifuPath != waifuPath {
		t.Errorf("waifu_path = %q, want %q", out.WaifuPath, waifuPath)
	}
}

func TestRenderJSON_WaifuPathOmittedWhenEmpty(t *testing.T) {
	dir := emSetupTestCache(t)
	b, err := RenderJSON(dir, "")
	if err != nil {
		t.Fatalf("RenderJSON: %v", err)
	}

	// The raw JSON should not contain "waifu_path" at all.
	if strings.Contains(string(b), "waifu_path") {
		t.Error("waifu_path should be omitted when empty")
	}
}

func TestRenderJSON_ValidJSON(t *testing.T) {
	dir := emSetupTestCache(t)
	emWriteFixture(t, dir, "claude", emSampleClaude())
	emWriteFixture(t, dir, "billing", emSampleBilling())

	b, err := RenderJSON(dir, "/tmp/test.png")
	if err != nil {
		t.Fatalf("RenderJSON: %v", err)
	}

	if !json.Valid(b) {
		t.Error("output is not valid JSON")
	}

	// Round-trip test: unmarshal and marshal should succeed.
	var parsed any
	if err := json.Unmarshal(b, &parsed); err != nil {
		t.Fatalf("unmarshal round-trip: %v", err)
	}
	if _, err := json.Marshal(parsed); err != nil {
		t.Fatalf("marshal round-trip: %v", err)
	}
}

// --- WidgetJSON status tests ---

func TestWidgetJSON_StatusOK(t *testing.T) {
	dir := emSetupTestCache(t)
	emWriteFixture(t, dir, "sysmetrics", &emSysMetrics{
		CPU: 30.0,
		RAM: 40.0,
	})

	b, err := RenderJSON(dir, "")
	if err != nil {
		t.Fatalf("RenderJSON: %v", err)
	}

	var out JSONOutput
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(out.Widgets) != 1 {
		t.Fatalf("widgets = %d, want 1", len(out.Widgets))
	}

	if out.Widgets[0].Status != "ok" {
		t.Errorf("status = %q, want %q", out.Widgets[0].Status, "ok")
	}
}

func TestWidgetJSON_StatusWarning(t *testing.T) {
	dir := emSetupTestCache(t)
	emWriteFixture(t, dir, "sysmetrics", &emSysMetrics{
		CPU: 75.0,
		RAM: 50.0,
	})

	b, err := RenderJSON(dir, "")
	if err != nil {
		t.Fatalf("RenderJSON: %v", err)
	}

	var out JSONOutput
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(out.Widgets) != 1 {
		t.Fatalf("widgets = %d, want 1", len(out.Widgets))
	}

	if out.Widgets[0].Status != "warning" {
		t.Errorf("status = %q, want %q", out.Widgets[0].Status, "warning")
	}
}

func TestWidgetJSON_StatusError(t *testing.T) {
	dir := emSetupTestCache(t)
	emWriteFixture(t, dir, "sysmetrics", &emSysMetrics{
		CPU: 95.0,
		RAM: 50.0,
	})

	b, err := RenderJSON(dir, "")
	if err != nil {
		t.Fatalf("RenderJSON: %v", err)
	}

	var out JSONOutput
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(out.Widgets) != 1 {
		t.Fatalf("widgets = %d, want 1", len(out.Widgets))
	}

	if out.Widgets[0].Status != "error" {
		t.Errorf("status = %q, want %q", out.Widgets[0].Status, "error")
	}
}

// --- RenderPropertized tests ---

func TestRenderPropertized_NonEmpty(t *testing.T) {
	dir := emSetupTestCache(t)
	emWriteFixture(t, dir, "sysmetrics", emSampleSysMetricsData())

	result, err := RenderPropertized(dir)
	if err != nil {
		t.Fatalf("RenderPropertized: %v", err)
	}

	if result == "" {
		t.Error("result is empty, expected non-empty propertized text")
	}
}

func TestRenderPropertized_ContainsFaceAnnotations(t *testing.T) {
	dir := emSetupTestCache(t)
	emWriteFixture(t, dir, "sysmetrics", emSampleSysMetricsData())

	result, err := RenderPropertized(dir)
	if err != nil {
		t.Fatalf("RenderPropertized: %v", err)
	}

	if !strings.Contains(result, "(face ") {
		t.Error("result does not contain face annotations")
	}
}

func TestRenderPropertized_NoData(t *testing.T) {
	dir := emSetupTestCache(t)

	result, err := RenderPropertized(dir)
	if err != nil {
		t.Fatalf("RenderPropertized: %v", err)
	}

	if !strings.Contains(result, "No data") {
		t.Errorf("expected 'No data' message, got %q", result)
	}
}

// --- emPropertize tests ---

func TestEmPropertize_WrapsCorrectly(t *testing.T) {
	result := emPropertize("Hello", "bold")
	expected := `#("Hello" 0 5 (face bold))`
	if result != expected {
		t.Errorf("emPropertize = %q, want %q", result, expected)
	}
}

func TestEmPropertize_EmptyText(t *testing.T) {
	result := emPropertize("", "bold")
	expected := `#("" 0 0 (face bold))`
	if result != expected {
		t.Errorf("emPropertize empty = %q, want %q", result, expected)
	}
}

func TestEmPropertize_LongerText(t *testing.T) {
	text := "System Metrics"
	result := emPropertize(text, "font-lock-keyword-face")
	if !strings.Contains(result, fmt.Sprintf("0 %d", len(text))) {
		t.Errorf("emPropertize did not include correct length: %s", result)
	}
	if !strings.Contains(result, "font-lock-keyword-face") {
		t.Error("emPropertize did not include face name")
	}
}

// --- emExtractClaude tests ---

func TestEmExtractClaude_WithData(t *testing.T) {
	dir := emSetupTestCache(t)
	emWriteFixture(t, dir, "claude", emSampleClaude())

	store, err := emOpenStore(dir)
	if err != nil {
		t.Fatalf("emOpenStore: %v", err)
	}
	defer store.Close()

	w := emExtractClaude(store)
	if w == nil {
		t.Fatal("emExtractClaude returned nil with cached data")
	}

	if w.ID != "claude" {
		t.Errorf("id = %q, want %q", w.ID, "claude")
	}
	if w.Status != "ok" {
		t.Errorf("status = %q, want %q", w.Status, "ok")
	}
	if !strings.Contains(w.Summary, "2 accounts") {
		t.Errorf("summary = %q, missing '2 accounts'", w.Summary)
	}
}

func TestEmExtractClaude_NoData(t *testing.T) {
	dir := emSetupTestCache(t)

	store, err := emOpenStore(dir)
	if err != nil {
		t.Fatalf("emOpenStore: %v", err)
	}
	defer store.Close()

	w := emExtractClaude(store)
	if w != nil {
		t.Error("emExtractClaude should return nil with no cached data")
	}
}

func TestEmExtractClaude_HighUtilWarning(t *testing.T) {
	dir := emSetupTestCache(t)
	data := &emClaudeUsage{
		Accounts: []emClaudeAccount{
			{
				Name:   "heavy",
				Type:   "subscription",
				Tier:   "max_20x",
				Status: "ok",
				FiveHour: &emUsagePeriod{
					Utilization: 92.0,
				},
			},
		},
	}
	emWriteFixture(t, dir, "claude", data)

	store, err := emOpenStore(dir)
	if err != nil {
		t.Fatalf("emOpenStore: %v", err)
	}
	defer store.Close()

	w := emExtractClaude(store)
	if w == nil {
		t.Fatal("emExtractClaude returned nil")
	}
	if w.Status != "warning" {
		t.Errorf("status = %q, want %q for high utilization", w.Status, "warning")
	}
}

// --- emExtractBilling tests ---

func TestEmExtractBilling_SummaryFormat(t *testing.T) {
	dir := emSetupTestCache(t)
	emWriteFixture(t, dir, "billing", emSampleBilling())

	store, err := emOpenStore(dir)
	if err != nil {
		t.Fatalf("emOpenStore: %v", err)
	}
	defer store.Close()

	w := emExtractBilling(store)
	if w == nil {
		t.Fatal("emExtractBilling returned nil")
	}

	if !strings.Contains(w.Summary, "$23.45/mo") {
		t.Errorf("summary = %q, missing '$23.45/mo'", w.Summary)
	}
	if !strings.Contains(w.Summary, "3 providers") {
		t.Errorf("summary = %q, missing '3 providers'", w.Summary)
	}
}

// --- emExtractTailscale tests ---

func TestEmExtractTailscale_PeerCountFormat(t *testing.T) {
	dir := emSetupTestCache(t)
	emWriteFixture(t, dir, "infra", emSampleInfra())

	store, err := emOpenStore(dir)
	if err != nil {
		t.Fatalf("emOpenStore: %v", err)
	}
	defer store.Close()

	w := emExtractTailscale(store)
	if w == nil {
		t.Fatal("emExtractTailscale returned nil")
	}

	if !strings.Contains(w.Summary, "3/5 peers online") {
		t.Errorf("summary = %q, missing '3/5 peers online'", w.Summary)
	}
}

// --- emExtractK8s tests ---

func TestEmExtractK8s_PodCountFormat(t *testing.T) {
	dir := emSetupTestCache(t)
	emWriteFixture(t, dir, "infra", emSampleInfra())

	store, err := emOpenStore(dir)
	if err != nil {
		t.Fatalf("emOpenStore: %v", err)
	}
	defer store.Close()

	w := emExtractK8s(store)
	if w == nil {
		t.Fatal("emExtractK8s returned nil")
	}

	if !strings.Contains(w.Summary, "12/15 pods running") {
		t.Errorf("summary = %q, missing '12/15 pods running'", w.Summary)
	}
}

// --- emExtractSystem tests ---

func TestEmExtractSystem_CPURAMFormat(t *testing.T) {
	dir := emSetupTestCache(t)
	emWriteFixture(t, dir, "sysmetrics", emSampleSysMetricsData())

	store, err := emOpenStore(dir)
	if err != nil {
		t.Fatalf("emOpenStore: %v", err)
	}
	defer store.Close()

	w := emExtractSystem(store)
	if w == nil {
		t.Fatal("emExtractSystem returned nil")
	}

	if !strings.Contains(w.Summary, "CPU:45%") {
		t.Errorf("summary = %q, missing 'CPU:45%%'", w.Summary)
	}
	if !strings.Contains(w.Summary, "RAM:63%") {
		t.Errorf("summary = %q, missing 'RAM:63%%'", w.Summary)
	}
}

// --- GenerateElispPackage tests ---

func TestGenerateElispPackage_ContainsRequiredFunctions(t *testing.T) {
	elisp := GenerateElispPackage()

	fns := []string{
		"prompt-pulse-dashboard-insert",
		"prompt-pulse-refresh",
		"prompt-pulse-waifu-insert",
	}

	for _, fn := range fns {
		if !strings.Contains(elisp, fn) {
			t.Errorf("elisp missing function %q", fn)
		}
	}
}

func TestGenerateElispPackage_ContainsAutoloadCookies(t *testing.T) {
	elisp := GenerateElispPackage()

	if !strings.Contains(elisp, ";;;###autoload") {
		t.Error("elisp missing autoload cookies")
	}

	count := strings.Count(elisp, ";;;###autoload")
	if count < 3 {
		t.Errorf("expected at least 3 autoload cookies, got %d", count)
	}
}

func TestGenerateElispPackage_ContainsDefcustomDefinitions(t *testing.T) {
	elisp := GenerateElispPackage()

	customs := []string{
		"prompt-pulse-binary-path",
		"prompt-pulse-refresh-interval",
		"prompt-pulse-show-waifu",
	}

	for _, c := range customs {
		if !strings.Contains(elisp, "(defcustom "+c) {
			t.Errorf("elisp missing defcustom %q", c)
		}
	}
}

func TestGenerateElispPackage_ContainsModeDefinition(t *testing.T) {
	elisp := GenerateElispPackage()

	if !strings.Contains(elisp, "define-minor-mode prompt-pulse-mode") {
		t.Error("elisp missing prompt-pulse-mode definition")
	}
}

func TestGenerateElispPackage_ContainsProvide(t *testing.T) {
	elisp := GenerateElispPackage()

	if !strings.Contains(elisp, "(provide 'prompt-pulse)") {
		t.Error("elisp missing (provide 'prompt-pulse)")
	}
}

func TestGenerateElispPackage_ContainsIntegrationHooks(t *testing.T) {
	elisp := GenerateElispPackage()

	hooks := []string{
		"prompt-pulse-eat-mode-hook",
		"prompt-pulse-doom-dashboard-section",
		"prompt-pulse-enlight-widget",
	}

	for _, hook := range hooks {
		if !strings.Contains(elisp, hook) {
			t.Errorf("elisp missing integration hook %q", hook)
		}
	}
}

func TestGenerateElispPackage_ContainsCustomizationGroup(t *testing.T) {
	elisp := GenerateElispPackage()

	if !strings.Contains(elisp, `(defgroup prompt-pulse nil`) {
		t.Error("elisp missing defgroup prompt-pulse")
	}
}

func TestGenerateElispPackage_ContainsCreateImage(t *testing.T) {
	elisp := GenerateElispPackage()

	if !strings.Contains(elisp, "create-image") {
		t.Error("elisp missing create-image call for waifu support")
	}
}

// --- Edge case tests ---

func TestRenderJSON_SingleProvider(t *testing.T) {
	dir := emSetupTestCache(t)
	budget := 50.0
	data := &emBillingData{
		Providers: []emProviderBilling{
			{
				Provider:    "civo",
				AccountName: "test",
				Status:      "ok",
				CurrentMonth: emMonthCost{
					SpendUSD: 15.00,
				},
			},
		},
		Total: emBillingSummary{
			CurrentMonthUSD: 15.00,
			BudgetUSD:       &budget,
			SuccessCount:    1,
			TotalConfigured: 1,
		},
	}
	emWriteFixture(t, dir, "billing", data)

	store, err := emOpenStore(dir)
	if err != nil {
		t.Fatalf("emOpenStore: %v", err)
	}
	defer store.Close()

	w := emExtractBilling(store)
	if w == nil {
		t.Fatal("emExtractBilling returned nil")
	}

	// Single provider should use singular form.
	if !strings.Contains(w.Summary, "1 provider)") {
		t.Errorf("summary = %q, expected singular 'provider'", w.Summary)
	}
}

func TestEmExtractTailscale_NoTailscaleData(t *testing.T) {
	dir := emSetupTestCache(t)
	data := &emInfraStatus{
		Tailscale: nil,
	}
	emWriteFixture(t, dir, "infra", data)

	store, err := emOpenStore(dir)
	if err != nil {
		t.Fatalf("emOpenStore: %v", err)
	}
	defer store.Close()

	w := emExtractTailscale(store)
	if w != nil {
		t.Error("emExtractTailscale should return nil when tailscale is nil")
	}
}

func TestEmExtractK8s_NoClusters(t *testing.T) {
	dir := emSetupTestCache(t)
	data := &emInfraStatus{
		Kubernetes: nil,
	}
	emWriteFixture(t, dir, "infra", data)

	store, err := emOpenStore(dir)
	if err != nil {
		t.Fatalf("emOpenStore: %v", err)
	}
	defer store.Close()

	w := emExtractK8s(store)
	if w != nil {
		t.Error("emExtractK8s should return nil when no clusters")
	}
}

func TestEmStatusFace_AllStatuses(t *testing.T) {
	tests := []struct {
		status   string
		expected string
	}{
		{"ok", "success"},
		{"warning", "font-lock-warning-face"},
		{"error", "error"},
		{"unknown", "font-lock-keyword-face"},
		{"", "font-lock-keyword-face"},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			face := emStatusFace(tt.status)
			if face != tt.expected {
				t.Errorf("emStatusFace(%q) = %q, want %q", tt.status, face, tt.expected)
			}
		})
	}
}
