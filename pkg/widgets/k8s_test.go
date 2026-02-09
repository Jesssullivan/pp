package widgets

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"gitlab.com/tinyland/lab/prompt-pulse/pkg/app"
	"gitlab.com/tinyland/lab/prompt-pulse/pkg/collectors/k8s"
)

// ---------- Helper constructors ----------

func singleClusterStatus(c k8s.ClusterInfo) *k8s.ClusterStatus {
	return &k8s.ClusterStatus{
		Clusters:  []k8s.ClusterInfo{c},
		Timestamp: time.Now(),
	}
}

func multiClusterStatus(clusters ...k8s.ClusterInfo) *k8s.ClusterStatus {
	return &k8s.ClusterStatus{
		Clusters:  clusters,
		Timestamp: time.Now(),
	}
}

func connectedCluster(ctx string, running, pending, failed int, nodes []k8s.NodeInfo, namespaces []k8s.NamespaceInfo) k8s.ClusterInfo {
	return k8s.ClusterInfo{
		Context:     ctx,
		Connected:   true,
		Nodes:       nodes,
		Namespaces:  namespaces,
		TotalPods:   running + pending + failed,
		RunningPods: running,
		PendingPods: pending,
		FailedPods:  failed,
	}
}

func disconnectedCluster(ctx, errMsg string) k8s.ClusterInfo {
	return k8s.ClusterInfo{
		Context:   ctx,
		Connected: false,
		Error:     errMsg,
	}
}

func readyNode(name, cpuCap, cpuReq, memCap, memReq string) k8s.NodeInfo {
	return k8s.NodeInfo{
		Name:        name,
		Ready:       true,
		CPUCapacity: cpuCap,
		CPURequests: cpuReq,
		MemCapacity: memCap,
		MemRequests: memReq,
	}
}

func notReadyNode(name string) k8s.NodeInfo {
	return k8s.NodeInfo{
		Name:  name,
		Ready: false,
	}
}

func healthyDeployment(name string, replicas int32) k8s.DeploymentInfo {
	return k8s.DeploymentInfo{
		Name:              name,
		Replicas:          replicas,
		ReadyReplicas:     replicas,
		UpdatedReplicas:   replicas,
		AvailableReplicas: replicas,
	}
}

func rollingDeployment(name string, total, ready, updated int32) k8s.DeploymentInfo {
	return k8s.DeploymentInfo{
		Name:              name,
		Replicas:          total,
		ReadyReplicas:     ready,
		UpdatedReplicas:   updated,
		AvailableReplicas: ready,
	}
}

// stripANSI removes ANSI escape sequences for test assertions.
func stripANSI(s string) string {
	var b strings.Builder
	inEsc := false
	for _, r := range s {
		if r == '\x1b' {
			inEsc = true
			continue
		}
		if inEsc {
			if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') {
				inEsc = false
			}
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// ---------- Tests ----------

func TestK8sWidget_ID(t *testing.T) {
	w := NewK8sWidget()
	if got := w.ID(); got != "k8s" {
		t.Errorf("ID() = %q, want %q", got, "k8s")
	}
}

func TestK8sWidget_Title(t *testing.T) {
	w := NewK8sWidget()
	if got := w.Title(); got != "Kubernetes" {
		t.Errorf("Title() = %q, want %q", got, "Kubernetes")
	}
}

func TestK8sWidget_MinSize(t *testing.T) {
	w := NewK8sWidget()
	minW, minH := w.MinSize()
	if minW != 30 || minH != 4 {
		t.Errorf("MinSize() = (%d, %d), want (30, 4)", minW, minH)
	}
}

func TestK8sWidget_ViewNoData(t *testing.T) {
	w := NewK8sWidget()
	view := w.View(40, 5)
	stripped := stripANSI(view)
	if !strings.Contains(stripped, "No data") {
		t.Errorf("View with no data should contain 'No data', got:\n%s", stripped)
	}
}

func TestK8sWidget_ViewCompactPodSummary(t *testing.T) {
	w := NewK8sWidget()
	w.clusterStatus = singleClusterStatus(connectedCluster(
		"prod", 42, 2, 1,
		[]k8s.NodeInfo{readyNode("node-1", "4", "2000m", "8Gi", "4Gi")},
		nil,
	))

	view := w.View(80, 10)
	stripped := stripANSI(view)

	// Check connection status.
	if !strings.Contains(stripped, "prod") {
		t.Errorf("compact view should show cluster context 'prod', got:\n%s", stripped)
	}
	if !strings.Contains(stripped, "connected") {
		t.Errorf("compact view should show 'connected', got:\n%s", stripped)
	}

	// Check pod summary.
	if !strings.Contains(stripped, "42 running") {
		t.Errorf("compact view should show '42 running', got:\n%s", stripped)
	}
	if !strings.Contains(stripped, "2 pending") {
		t.Errorf("compact view should show '2 pending', got:\n%s", stripped)
	}
	if !strings.Contains(stripped, "1 failed") {
		t.Errorf("compact view should show '1 failed', got:\n%s", stripped)
	}

	// Check node count.
	if !strings.Contains(stripped, "Nodes: 1/1 ready") {
		t.Errorf("compact view should show 'Nodes: 1/1 ready', got:\n%s", stripped)
	}
}

func TestK8sWidget_ViewExpandedNamespaceBreakdown(t *testing.T) {
	w := NewK8sWidget()
	w.expanded = true
	w.clusterStatus = singleClusterStatus(connectedCluster(
		"prod", 10, 1, 0,
		[]k8s.NodeInfo{readyNode("node-1", "4", "1000m", "8Gi", "2Gi")},
		[]k8s.NamespaceInfo{
			{
				Name: "default",
				PodCounts: k8s.PodCounts{
					Total:   6,
					Running: 5,
					Pending: 1,
				},
				Deployments: []k8s.DeploymentInfo{
					healthyDeployment("web-app", 3),
				},
			},
			{
				Name: "kube-system",
				PodCounts: k8s.PodCounts{
					Total:   5,
					Running: 5,
				},
			},
		},
	))

	view := w.View(80, 30)
	stripped := stripANSI(view)

	// Should have namespace sections.
	if !strings.Contains(stripped, "Namespace: default") {
		t.Errorf("expanded view should show 'Namespace: default', got:\n%s", stripped)
	}
	if !strings.Contains(stripped, "Namespace: kube-system") {
		t.Errorf("expanded view should show 'Namespace: kube-system', got:\n%s", stripped)
	}

	// Should have deployment info.
	if !strings.Contains(stripped, "web-app") {
		t.Errorf("expanded view should show deployment 'web-app', got:\n%s", stripped)
	}

	// Should have node resources section.
	if !strings.Contains(stripped, "Node Resources") {
		t.Errorf("expanded view should show 'Node Resources', got:\n%s", stripped)
	}
}

func TestK8sWidget_UpdateWithClusterStatus(t *testing.T) {
	w := NewK8sWidget()

	cs := singleClusterStatus(connectedCluster("test", 5, 0, 0, nil, nil))
	msg := app.DataUpdateEvent{
		Source: "k8s",
		Data:   cs,
	}
	cmd := w.Update(msg)

	if cmd != nil {
		t.Error("Update should return nil cmd")
	}
	if w.clusterStatus != cs {
		t.Error("Update should set clusterStatus")
	}
}

func TestK8sWidget_UpdateIgnoresOtherSources(t *testing.T) {
	w := NewK8sWidget()

	msg := app.DataUpdateEvent{
		Source: "tailscale",
		Data:   "some data",
	}
	w.Update(msg)

	if w.clusterStatus != nil {
		t.Error("Update should ignore events from other sources")
	}
}

func TestK8sWidget_HandleKeyToggleExpanded(t *testing.T) {
	w := NewK8sWidget()

	if w.expanded {
		t.Fatal("widget should start not expanded")
	}

	w.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	if !w.expanded {
		t.Error("pressing 'e' should toggle expanded to true")
	}

	w.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	if w.expanded {
		t.Error("pressing 'e' again should toggle expanded to false")
	}
}

func TestK8sWidget_HandleKeyCycleClusters(t *testing.T) {
	w := NewK8sWidget()
	w.clusterStatus = multiClusterStatus(
		connectedCluster("prod", 10, 0, 0, nil, nil),
		connectedCluster("staging", 5, 1, 0, nil, nil),
		disconnectedCluster("dev", "timeout"),
	)

	if w.selectedCluster != 0 {
		t.Fatal("should start at cluster 0")
	}

	w.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	if w.selectedCluster != 1 {
		t.Errorf("after first 'c', selectedCluster = %d, want 1", w.selectedCluster)
	}

	w.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	if w.selectedCluster != 2 {
		t.Errorf("after second 'c', selectedCluster = %d, want 2", w.selectedCluster)
	}

	w.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	if w.selectedCluster != 0 {
		t.Errorf("after third 'c', selectedCluster = %d, want 0 (wrap)", w.selectedCluster)
	}
}

func TestK8sWidget_NodeHealthIndicators(t *testing.T) {
	w := NewK8sWidget()
	w.clusterStatus = singleClusterStatus(connectedCluster(
		"prod", 10, 0, 0,
		[]k8s.NodeInfo{
			readyNode("node-1", "4", "1000m", "8Gi", "2Gi"),
			notReadyNode("node-2"),
			readyNode("node-3", "4", "500m", "8Gi", "1Gi"),
		},
		nil,
	))

	view := w.View(80, 10)
	stripped := stripANSI(view)

	// Check node names appear.
	if !strings.Contains(stripped, "node-1") {
		t.Errorf("view should contain 'node-1', got:\n%s", stripped)
	}
	if !strings.Contains(stripped, "node-2") {
		t.Errorf("view should contain 'node-2', got:\n%s", stripped)
	}
	if !strings.Contains(stripped, "node-3") {
		t.Errorf("view should contain 'node-3', got:\n%s", stripped)
	}

	// The view should show proper ready count.
	if !strings.Contains(stripped, "2/3 ready") {
		t.Errorf("view should show '2/3 ready', got:\n%s", stripped)
	}
}

func TestK8sWidget_ClusterConnectionStatus(t *testing.T) {
	w := NewK8sWidget()
	w.clusterStatus = singleClusterStatus(disconnectedCluster("staging", "connection refused"))

	view := w.View(80, 5)
	stripped := stripANSI(view)

	if !strings.Contains(stripped, "staging") {
		t.Errorf("view should show cluster context 'staging', got:\n%s", stripped)
	}
	if !strings.Contains(stripped, "disconnected") {
		t.Errorf("view should show 'disconnected', got:\n%s", stripped)
	}
	if !strings.Contains(stripped, "connection refused") {
		t.Errorf("view should show error message, got:\n%s", stripped)
	}
}

func TestK8sWidget_DeploymentProgressDisplay(t *testing.T) {
	w := NewK8sWidget()
	w.expanded = true
	w.clusterStatus = singleClusterStatus(connectedCluster(
		"prod", 8, 2, 0,
		[]k8s.NodeInfo{readyNode("node-1", "4", "2000m", "8Gi", "4Gi")},
		[]k8s.NamespaceInfo{
			{
				Name: "default",
				PodCounts: k8s.PodCounts{
					Total: 10, Running: 8, Pending: 2,
				},
				Deployments: []k8s.DeploymentInfo{
					healthyDeployment("api-server", 3),
					rollingDeployment("web-frontend", 5, 3, 2),
				},
			},
		},
	))

	view := w.View(80, 30)
	stripped := stripANSI(view)

	if !strings.Contains(stripped, "api-server") {
		t.Errorf("should show deployment 'api-server', got:\n%s", stripped)
	}
	if !strings.Contains(stripped, "Healthy") {
		t.Errorf("healthy deployment should show 'Healthy', got:\n%s", stripped)
	}
	if !strings.Contains(stripped, "web-frontend") {
		t.Errorf("should show deployment 'web-frontend', got:\n%s", stripped)
	}
	if !strings.Contains(stripped, "Rolling") {
		t.Errorf("rolling deployment should show 'Rolling', got:\n%s", stripped)
	}
}

func TestK8sWidget_ViewSmallSize30x4(t *testing.T) {
	w := NewK8sWidget()
	w.clusterStatus = singleClusterStatus(connectedCluster(
		"prod", 5, 0, 0,
		[]k8s.NodeInfo{readyNode("n1", "2", "500m", "4Gi", "1Gi")},
		nil,
	))

	view := w.View(30, 4)
	lines := strings.Split(view, "\n")
	if len(lines) != 4 {
		t.Errorf("View at 30x4 should have 4 lines, got %d", len(lines))
	}
}

func TestK8sWidget_ViewMediumSize60x15(t *testing.T) {
	w := NewK8sWidget()
	w.clusterStatus = singleClusterStatus(connectedCluster(
		"prod", 20, 3, 1,
		[]k8s.NodeInfo{
			readyNode("node-1", "8", "4000m", "16Gi", "8Gi"),
			readyNode("node-2", "8", "3000m", "16Gi", "6Gi"),
		},
		[]k8s.NamespaceInfo{
			{Name: "default", PodCounts: k8s.PodCounts{Total: 15, Running: 12, Pending: 2, Failed: 1}},
		},
	))

	view := w.View(60, 15)
	lines := strings.Split(view, "\n")
	if len(lines) != 15 {
		t.Errorf("View at 60x15 should have 15 lines, got %d", len(lines))
	}
}

func TestK8sWidget_ViewLargeSize80x24(t *testing.T) {
	w := NewK8sWidget()
	w.expanded = true
	w.clusterStatus = singleClusterStatus(connectedCluster(
		"prod", 30, 2, 0,
		[]k8s.NodeInfo{
			readyNode("node-1", "16", "8000m", "32Gi", "16Gi"),
			readyNode("node-2", "16", "6000m", "32Gi", "12Gi"),
			notReadyNode("node-3"),
		},
		[]k8s.NamespaceInfo{
			{
				Name:      "default",
				PodCounts: k8s.PodCounts{Total: 20, Running: 18, Pending: 2},
				Deployments: []k8s.DeploymentInfo{
					healthyDeployment("api", 5),
					healthyDeployment("worker", 3),
				},
			},
			{
				Name:      "monitoring",
				PodCounts: k8s.PodCounts{Total: 12, Running: 12},
				Deployments: []k8s.DeploymentInfo{
					healthyDeployment("prometheus", 2),
				},
			},
		},
	))

	view := w.View(80, 24)
	lines := strings.Split(view, "\n")
	if len(lines) != 24 {
		t.Errorf("View at 80x24 should have 24 lines, got %d", len(lines))
	}
}

func TestK8sWidget_MultiClusterRendering(t *testing.T) {
	w := NewK8sWidget()
	w.clusterStatus = multiClusterStatus(
		connectedCluster("prod", 40, 0, 0,
			[]k8s.NodeInfo{readyNode("p-1", "8", "4000m", "16Gi", "8Gi")},
			nil,
		),
		connectedCluster("staging", 10, 2, 0,
			[]k8s.NodeInfo{readyNode("s-1", "4", "1000m", "8Gi", "2Gi")},
			nil,
		),
	)

	view := w.View(80, 10)
	stripped := stripANSI(view)

	// Tab bar should show both clusters.
	if !strings.Contains(stripped, "prod") {
		t.Errorf("multi-cluster view should show 'prod', got:\n%s", stripped)
	}
	if !strings.Contains(stripped, "staging") {
		t.Errorf("multi-cluster view should show 'staging', got:\n%s", stripped)
	}

	// Default should show first cluster's data.
	if !strings.Contains(stripped, "40 running") {
		t.Errorf("multi-cluster view should show prod's pod count '40 running', got:\n%s", stripped)
	}

	// Switch to staging.
	w.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	view2 := w.View(80, 10)
	stripped2 := stripANSI(view2)

	if !strings.Contains(stripped2, "10 running") {
		t.Errorf("after cycling, should show staging's '10 running', got:\n%s", stripped2)
	}
}

func TestK8sWidget_DisconnectedClusterDisplay(t *testing.T) {
	w := NewK8sWidget()
	w.clusterStatus = singleClusterStatus(disconnectedCluster("dev", "dial tcp: timeout"))

	view := w.View(60, 5)
	stripped := stripANSI(view)

	// Should show disconnected indicator.
	if !strings.Contains(stripped, "disconnected") {
		t.Errorf("disconnected cluster should show 'disconnected', got:\n%s", stripped)
	}
	if !strings.Contains(stripped, "timeout") {
		t.Errorf("disconnected cluster should show error reason, got:\n%s", stripped)
	}
}

func TestK8sWidget_HandleKeyScrollUpDown(t *testing.T) {
	w := NewK8sWidget()
	w.expanded = true
	w.clusterStatus = singleClusterStatus(connectedCluster(
		"prod", 100, 10, 5,
		[]k8s.NodeInfo{readyNode("n1", "16", "8000m", "32Gi", "16Gi")},
		[]k8s.NamespaceInfo{
			{Name: "ns1", PodCounts: k8s.PodCounts{Total: 50, Running: 40, Pending: 5, Failed: 5}},
			{Name: "ns2", PodCounts: k8s.PodCounts{Total: 30, Running: 25, Pending: 3, Failed: 2}},
			{Name: "ns3", PodCounts: k8s.PodCounts{Total: 35, Running: 35}},
		},
	))

	if w.scrollOffset != 0 {
		t.Fatal("scroll should start at 0")
	}

	w.HandleKey(tea.KeyMsg{Type: tea.KeyDown})
	if w.scrollOffset != 1 {
		t.Errorf("scrollOffset after down = %d, want 1", w.scrollOffset)
	}

	w.HandleKey(tea.KeyMsg{Type: tea.KeyDown})
	if w.scrollOffset != 2 {
		t.Errorf("scrollOffset after second down = %d, want 2", w.scrollOffset)
	}

	w.HandleKey(tea.KeyMsg{Type: tea.KeyUp})
	if w.scrollOffset != 1 {
		t.Errorf("scrollOffset after up = %d, want 1", w.scrollOffset)
	}

	// Should not go below 0.
	w.HandleKey(tea.KeyMsg{Type: tea.KeyUp})
	w.HandleKey(tea.KeyMsg{Type: tea.KeyUp})
	if w.scrollOffset != 0 {
		t.Errorf("scrollOffset should not go below 0, got %d", w.scrollOffset)
	}
}

func TestK8sWidget_HandleKeyCycleSingleCluster(t *testing.T) {
	w := NewK8sWidget()
	w.clusterStatus = singleClusterStatus(connectedCluster("only", 1, 0, 0, nil, nil))

	w.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	if w.selectedCluster != 0 {
		t.Error("cycling with single cluster should stay at 0")
	}
}

func TestK8sWidget_ViewZeroSize(t *testing.T) {
	w := NewK8sWidget()
	if got := w.View(0, 0); got != "" {
		t.Errorf("View(0,0) = %q, want empty string", got)
	}
	if got := w.View(-1, 10); got != "" {
		t.Errorf("View(-1,10) = %q, want empty string", got)
	}
}

func TestK8sWidget_CompileTimeInterface(t *testing.T) {
	// This test verifies the compile-time interface check at the bottom
	// of k8s.go. If K8sWidget doesn't implement app.Widget, the code
	// won't compile.
	var _ app.Widget = (*K8sWidget)(nil)
}

func TestK8sWidget_ResourceFormatting(t *testing.T) {
	// Test CPU parsing and formatting.
	tests := []struct {
		input string
		want  int64
	}{
		{"2000m", 2000},
		{"500m", 500},
		{"4", 4000},
		{"0", 0},
		{"", 0},
	}
	for _, tc := range tests {
		got := k8wParseMilliCPU(tc.input)
		if got != tc.want {
			t.Errorf("k8wParseMilliCPU(%q) = %d, want %d", tc.input, got, tc.want)
		}
	}

	// Test memory parsing.
	memTests := []struct {
		input string
		want  int64
	}{
		{"4Gi", 4 * 1024 * 1024 * 1024},
		{"512Mi", 512 * 1024 * 1024},
		{"1024Ki", 1024 * 1024},
		{"1073741824", 1073741824},
		{"", 0},
	}
	for _, tc := range memTests {
		got := k8wParseMemory(tc.input)
		if got != tc.want {
			t.Errorf("k8wParseMemory(%q) = %d, want %d", tc.input, got, tc.want)
		}
	}

	// Test CPU formatting.
	if got := k8wFormatCPU(2000); got != "2.0 cores" {
		t.Errorf("k8wFormatCPU(2000) = %q, want '2.0 cores'", got)
	}
	if got := k8wFormatCPU(500); got != "0.5 cores" {
		t.Errorf("k8wFormatCPU(500) = %q, want '0.5 cores'", got)
	}

	// Test memory formatting.
	if got := k8wFormatMemory(4 * 1024 * 1024 * 1024); got != "4.0 GB" {
		t.Errorf("k8wFormatMemory(4Gi) = %q, want '4.0 GB'", got)
	}
	if got := k8wFormatMemory(512 * 1024 * 1024); got != "512.0 MB" {
		t.Errorf("k8wFormatMemory(512Mi) = %q, want '512.0 MB'", got)
	}
}

func TestK8sWidget_UpdateClampsSelectedCluster(t *testing.T) {
	w := NewK8sWidget()
	// Simulate having selected cluster 2 in a 3-cluster setup.
	w.selectedCluster = 2

	// Now update with only 1 cluster.
	cs := singleClusterStatus(connectedCluster("only", 1, 0, 0, nil, nil))
	w.Update(app.DataUpdateEvent{Source: "k8s", Data: cs})

	if w.selectedCluster != 0 {
		t.Errorf("selectedCluster should be clamped to 0, got %d", w.selectedCluster)
	}
}

func TestK8sWidget_DefaultContextName(t *testing.T) {
	w := NewK8sWidget()
	// Cluster with empty context name should display as "default".
	w.clusterStatus = singleClusterStatus(connectedCluster("", 5, 0, 0, nil, nil))

	view := w.View(60, 5)
	stripped := stripANSI(view)

	if !strings.Contains(stripped, "default") {
		t.Errorf("empty context should display as 'default', got:\n%s", stripped)
	}
}
