package widgets

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"gitlab.com/tinyland/lab/prompt-pulse/pkg/app"
	"gitlab.com/tinyland/lab/prompt-pulse/pkg/collectors/k8s"
	"gitlab.com/tinyland/lab/prompt-pulse/pkg/components"
)

// K8sWidget displays Kubernetes cluster status including pod counts, node
// health, deployment progress, and resource utilization. It supports
// multi-cluster views and compact/expanded display modes.
type K8sWidget struct {
	clusterStatus   *k8s.ClusterStatus
	expanded        bool
	selectedCluster int
	scrollOffset    int
}

// NewK8sWidget creates a K8sWidget with default state.
func NewK8sWidget() *K8sWidget {
	return &K8sWidget{}
}

// ID returns "k8s".
func (w *K8sWidget) ID() string { return "k8s" }

// Title returns "Kubernetes".
func (w *K8sWidget) Title() string { return "Kubernetes" }

// MinSize returns the minimum dimensions required by the widget.
func (w *K8sWidget) MinSize() (int, int) { return 30, 4 }

// Update handles messages directed at this widget. It processes
// DataUpdateEvent with Source "k8s".
func (w *K8sWidget) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case app.DataUpdateEvent:
		if msg.Source != "k8s" {
			return nil
		}
		if msg.Err != nil {
			return nil
		}
		if cs, ok := msg.Data.(*k8s.ClusterStatus); ok {
			w.clusterStatus = cs
			// Clamp selectedCluster to valid range.
			if w.selectedCluster >= len(cs.Clusters) {
				w.selectedCluster = 0
			}
		}
	}
	return nil
}

// HandleKey processes key events when the widget has focus.
// 'e' toggles expanded mode, 'c' cycles through clusters,
// up/down arrows scroll in expanded mode.
func (w *K8sWidget) HandleKey(key tea.KeyMsg) tea.Cmd {
	switch key.String() {
	case "e":
		w.expanded = !w.expanded
		w.scrollOffset = 0
		return nil
	case "c":
		if w.clusterStatus != nil && len(w.clusterStatus.Clusters) > 1 {
			w.selectedCluster = (w.selectedCluster + 1) % len(w.clusterStatus.Clusters)
			w.scrollOffset = 0
		}
		return nil
	case "up":
		if w.scrollOffset > 0 {
			w.scrollOffset--
		}
		return nil
	case "down":
		w.scrollOffset++
		return nil
	}
	return nil
}

// View renders the widget content into the given area dimensions.
func (w *K8sWidget) View(width, height int) string {
	if width <= 0 || height <= 0 {
		return ""
	}

	if w.clusterStatus == nil || len(w.clusterStatus.Clusters) == 0 {
		return centerText("No data", width, height)
	}

	var lines []string

	// Multi-cluster tab bar.
	if len(w.clusterStatus.Clusters) > 1 {
		lines = append(lines, k8wRenderClusterTabs(w.clusterStatus.Clusters, w.selectedCluster, width))
	}

	// Render the selected cluster.
	cluster := w.clusterStatus.Clusters[w.selectedCluster]

	if w.expanded {
		lines = append(lines, k8wRenderExpanded(cluster, width)...)
	} else {
		lines = append(lines, k8wRenderCompact(cluster, width)...)
	}

	// Apply scroll offset and fit to height.
	return k8wFitToArea(lines, width, height, w.scrollOffset)
}

// ---------- Compact rendering ----------

// k8wRenderCompact renders a concise cluster overview: connection status,
// pod summary, and node health dots.
func k8wRenderCompact(c k8s.ClusterInfo, width int) []string {
	var lines []string

	// Connection status line.
	lines = append(lines, k8wConnectionLine(c, width))

	// Pod summary.
	summary := fmt.Sprintf("Pods: %d running, %d pending, %d failed",
		c.RunningPods, c.PendingPods, c.FailedPods)
	nodeReadyCount, nodeTotalCount := k8wNodeCounts(c)
	summary += fmt.Sprintf(" | Nodes: %d/%d ready", nodeReadyCount, nodeTotalCount)
	if components.VisibleLen(summary) > width {
		summary = components.TruncateWithTail(summary, width, "...")
	}
	lines = append(lines, components.PadRight(summary, width))

	// Node health dots.
	if len(c.Nodes) > 0 {
		lines = append(lines, k8wNodeHealthLine(c.Nodes, width))
	}

	return lines
}

// ---------- Expanded rendering ----------

// k8wRenderExpanded renders a detailed cluster view with namespace
// breakdowns, deployment tables, and node resource gauges.
func k8wRenderExpanded(c k8s.ClusterInfo, width int) []string {
	var lines []string

	// Connection status.
	lines = append(lines, k8wConnectionLine(c, width))

	if !c.Connected {
		return lines
	}

	// Pod summary line.
	summary := fmt.Sprintf("Pods: %d running, %d pending, %d failed",
		c.RunningPods, c.PendingPods, c.FailedPods)
	nodeReadyCount, nodeTotalCount := k8wNodeCounts(c)
	summary += fmt.Sprintf(" | Nodes: %d/%d ready", nodeReadyCount, nodeTotalCount)
	if components.VisibleLen(summary) > width {
		summary = components.TruncateWithTail(summary, width, "...")
	}
	lines = append(lines, components.PadRight(summary, width))
	lines = append(lines, "")

	// Namespace sections.
	for _, ns := range c.Namespaces {
		nsHeader := components.Bold(fmt.Sprintf("Namespace: %s", ns.Name))
		lines = append(lines, components.PadRight(nsHeader, width))

		// Pod counts by phase.
		phases := k8wPodPhaseString(ns.PodCounts)
		lines = append(lines, components.PadRight("  "+phases, width))

		// Deployment table.
		if len(ns.Deployments) > 0 {
			lines = append(lines, k8wDeploymentTable(ns.Deployments, width)...)
		}
		lines = append(lines, "")
	}

	// Node resource gauges.
	if len(c.Nodes) > 0 {
		lines = append(lines, components.Bold("Node Resources"))
		for _, node := range c.Nodes {
			lines = append(lines, k8wNodeResourceLines(node, width)...)
		}
	}

	return lines
}

// ---------- Connection and node status ----------

// k8wConnectionLine renders the cluster connection status indicator.
func k8wConnectionLine(c k8s.ClusterInfo, width int) string {
	ctx := c.Context
	if ctx == "" {
		ctx = "default"
	}

	var line string
	if c.Connected {
		line = components.Color("#22C55E") + "\u25cf" + components.Reset() +
			" " + ctx + " (connected)"
	} else {
		errMsg := "unknown"
		if c.Error != "" {
			errMsg = c.Error
		}
		line = components.Color("#EF4444") + "\u25cb" + components.Reset() +
			" " + ctx + " (disconnected: " + errMsg + ")"
	}
	if components.VisibleLen(line) > width {
		line = components.TruncateWithTail(line, width, "...")
	}
	return components.PadRight(line, width)
}

// k8wNodeHealthLine renders a row of colored dots indicating node readiness.
func k8wNodeHealthLine(nodes []k8s.NodeInfo, width int) string {
	var parts []string
	for _, n := range nodes {
		var dot string
		if n.Ready {
			dot = components.Color("#22C55E") + "\u25cf" + components.Reset()
		} else {
			dot = components.Color("#EF4444") + "\u25cf" + components.Reset()
		}
		parts = append(parts, n.Name+" "+dot)
	}
	line := strings.Join(parts, " | ")
	if components.VisibleLen(line) > width {
		line = components.TruncateWithTail(line, width, "...")
	}
	return components.PadRight(line, width)
}

// k8wNodeCounts returns (ready, total) node counts for a cluster.
func k8wNodeCounts(c k8s.ClusterInfo) (int, int) {
	ready := 0
	for _, n := range c.Nodes {
		if n.Ready {
			ready++
		}
	}
	return ready, len(c.Nodes)
}

// ---------- Pod phase formatting ----------

// k8wPodPhaseString formats pod counts by phase with colors.
func k8wPodPhaseString(pc k8s.PodCounts) string {
	var parts []string
	if pc.Running > 0 {
		parts = append(parts, components.Color("#22C55E")+fmt.Sprintf("%d running", pc.Running)+components.Reset())
	}
	if pc.Pending > 0 {
		parts = append(parts, components.Color("#EAB308")+fmt.Sprintf("%d pending", pc.Pending)+components.Reset())
	}
	if pc.Failed > 0 {
		parts = append(parts, components.Color("#EF4444")+fmt.Sprintf("%d failed", pc.Failed)+components.Reset())
	}
	if pc.Succeeded > 0 {
		parts = append(parts, components.Dim(fmt.Sprintf("%d succeeded", pc.Succeeded)))
	}
	if pc.Unknown > 0 {
		parts = append(parts, components.Color("#6B7280")+fmt.Sprintf("%d unknown", pc.Unknown)+components.Reset())
	}
	if len(parts) == 0 {
		return "0 pods"
	}
	return strings.Join(parts, ", ")
}

// ---------- Deployment table ----------

// k8wDeploymentTable renders a simple text table of deployments.
func k8wDeploymentTable(deps []k8s.DeploymentInfo, width int) []string {
	var lines []string

	// Header.
	header := fmt.Sprintf("  %-20s %10s %10s %10s %s", "Name", "Ready", "Updated", "Available", "Status")
	if components.VisibleLen(header) > width {
		header = components.TruncateWithTail(header, width, "...")
	}
	lines = append(lines, components.Dim(header))

	for _, d := range deps {
		ready := fmt.Sprintf("%d/%d", d.ReadyReplicas, d.Replicas)
		updated := fmt.Sprintf("%d", d.UpdatedReplicas)
		available := fmt.Sprintf("%d", d.AvailableReplicas)

		status := k8wDeploymentStatus(d)

		row := fmt.Sprintf("  %-20s %10s %10s %10s %s",
			k8wTruncName(d.Name, 20), ready, updated, available, status)
		if components.VisibleLen(row) > width {
			row = components.TruncateWithTail(row, width, "...")
		}
		lines = append(lines, components.PadRight(row, width))
	}
	return lines
}

// k8wDeploymentStatus returns a human-readable status for a deployment.
func k8wDeploymentStatus(d k8s.DeploymentInfo) string {
	if d.ReadyReplicas == d.Replicas && d.Replicas > 0 {
		return components.Color("#22C55E") + "Healthy" + components.Reset()
	}
	if d.UpdatedReplicas < d.Replicas {
		return components.Color("#EAB308") + "Rolling" + components.Reset()
	}
	if d.ReadyReplicas < d.Replicas {
		return components.Color("#EAB308") + "Progressing" + components.Reset()
	}
	return components.Dim("Unknown")
}

// k8wTruncName truncates a name to fit within maxLen characters.
func k8wTruncName(name string, maxLen int) string {
	if len(name) <= maxLen {
		return name
	}
	if maxLen <= 3 {
		return name[:maxLen]
	}
	return name[:maxLen-3] + "..."
}

// ---------- Node resource gauges ----------

// k8wNodeResourceLines renders CPU and memory gauge lines for a node.
func k8wNodeResourceLines(node k8s.NodeInfo, width int) []string {
	var lines []string

	readyDot := components.Color("#22C55E") + "\u25cf" + components.Reset()
	if !node.Ready {
		readyDot = components.Color("#EF4444") + "\u25cf" + components.Reset()
	}

	nameLabel := fmt.Sprintf("  %s %s", readyDot, node.Name)
	lines = append(lines, components.PadRight(nameLabel, width))

	// CPU gauge.
	cpuCap := k8wParseMilliCPU(node.CPUCapacity)
	cpuReq := k8wParseMilliCPU(node.CPURequests)
	cpuLabel := fmt.Sprintf("    CPU: %s / %s", k8wFormatCPU(cpuReq), k8wFormatCPU(cpuCap))

	gaugeWidth := width - components.VisibleLen(cpuLabel) - 2
	if gaugeWidth < 5 {
		gaugeWidth = 5
	}
	if gaugeWidth > 20 {
		gaugeWidth = 20
	}
	g := components.NewGauge(components.GaugeStyle{
		Width:             gaugeWidth,
		ShowPercent:       true,
		FilledColor:       "#4CAF50",
		EmptyColor:        "#333333",
		WarningThreshold:  0.7,
		CriticalThreshold: 0.9,
		WarningColor:      "#FF9800",
		CriticalColor:     "#F44336",
	})
	cpuBar := g.Render(float64(cpuReq), float64(cpuCap), gaugeWidth)
	cpuLine := cpuLabel + " " + cpuBar
	if components.VisibleLen(cpuLine) > width {
		cpuLine = components.TruncateWithTail(cpuLine, width, "...")
	}
	lines = append(lines, components.PadRight(cpuLine, width))

	// Memory gauge.
	memCap := k8wParseMemory(node.MemCapacity)
	memReq := k8wParseMemory(node.MemRequests)
	memLabel := fmt.Sprintf("    Mem: %s / %s", k8wFormatMemory(memReq), k8wFormatMemory(memCap))

	memBar := g.Render(float64(memReq), float64(memCap), gaugeWidth)
	memLine := memLabel + " " + memBar
	if components.VisibleLen(memLine) > width {
		memLine = components.TruncateWithTail(memLine, width, "...")
	}
	lines = append(lines, components.PadRight(memLine, width))

	return lines
}

// ---------- Multi-cluster tabs ----------

// k8wRenderClusterTabs renders a tab bar showing all cluster contexts.
func k8wRenderClusterTabs(clusters []k8s.ClusterInfo, selected int, width int) string {
	var parts []string
	for i, c := range clusters {
		ctx := c.Context
		if ctx == "" {
			ctx = "default"
		}
		if i == selected {
			parts = append(parts, components.Bold("["+ctx+"]"))
		} else {
			parts = append(parts, components.Dim(" "+ctx+" "))
		}
	}
	line := strings.Join(parts, " ")
	if components.VisibleLen(line) > width {
		line = components.TruncateWithTail(line, width, "...")
	}
	return components.PadRight(line, width)
}

// ---------- Resource quantity parsing ----------

// k8wParseMilliCPU parses a CPU quantity string and returns millicores.
// Supports "2" (cores), "2000m" (millicores), "500m", etc.
func k8wParseMilliCPU(s string) int64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	if strings.HasSuffix(s, "m") {
		v, err := strconv.ParseInt(strings.TrimSuffix(s, "m"), 10, 64)
		if err != nil {
			return 0
		}
		return v
	}
	// Whole cores.
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return int64(v * 1000)
}

// k8wFormatCPU formats millicores as a human-readable string.
func k8wFormatCPU(millis int64) string {
	cores := float64(millis) / 1000.0
	if cores == math.Trunc(cores) {
		return fmt.Sprintf("%.1f cores", cores)
	}
	return fmt.Sprintf("%.1f cores", cores)
}

// k8wParseMemory parses a memory quantity string and returns bytes.
// Supports bare integers (bytes), "Ki", "Mi", "Gi", "Ti" suffixes.
func k8wParseMemory(s string) int64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}

	suffixes := []struct {
		suffix     string
		multiplier int64
	}{
		{"Ti", 1024 * 1024 * 1024 * 1024},
		{"Gi", 1024 * 1024 * 1024},
		{"Mi", 1024 * 1024},
		{"Ki", 1024},
	}

	for _, sf := range suffixes {
		if strings.HasSuffix(s, sf.suffix) {
			v, err := strconv.ParseFloat(strings.TrimSuffix(s, sf.suffix), 64)
			if err != nil {
				return 0
			}
			return int64(v * float64(sf.multiplier))
		}
	}

	// Plain bytes.
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0
	}
	return v
}

// k8wFormatMemory formats bytes as a human-readable string.
func k8wFormatMemory(bytes int64) string {
	const (
		gi = 1024 * 1024 * 1024
		mi = 1024 * 1024
	)
	if bytes >= gi {
		return fmt.Sprintf("%.1f GB", float64(bytes)/float64(gi))
	}
	if bytes >= mi {
		return fmt.Sprintf("%.1f MB", float64(bytes)/float64(mi))
	}
	return fmt.Sprintf("%d B", bytes)
}

// ---------- Layout helpers ----------

// k8wFitToArea applies scroll offset, pads/truncates lines to fit the
// given width x height area.
func k8wFitToArea(lines []string, width, height, scrollOffset int) string {
	// Apply scroll offset.
	if scrollOffset > 0 {
		if scrollOffset >= len(lines) {
			lines = nil
		} else {
			lines = lines[scrollOffset:]
		}
	}

	// Pad each line to width.
	for i, line := range lines {
		vis := components.VisibleLen(line)
		if vis > width {
			lines[i] = components.TruncateWithTail(line, width, "...")
		} else if vis < width {
			lines[i] = components.PadRight(line, width)
		}
	}

	// Fill to height or truncate.
	emptyLine := strings.Repeat(" ", width)
	for len(lines) < height {
		lines = append(lines, emptyLine)
	}
	if len(lines) > height {
		lines = lines[:height]
	}

	return strings.Join(lines, "\n")
}

// compile-time check that K8sWidget implements app.Widget.
var _ app.Widget = (*K8sWidget)(nil)
