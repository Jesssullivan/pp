// Package widgets provides the concrete widget implementations for the
// prompt-pulse v2 TUI dashboard.

package widgets

import (
	"fmt"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"gitlab.com/tinyland/lab/prompt-pulse/pkg/app"
	"gitlab.com/tinyland/lab/prompt-pulse/pkg/collectors/tailscale"
	"gitlab.com/tinyland/lab/prompt-pulse/pkg/components"
)

// Status indicator characters.
const (
	tsOnlineDot   = "\u25CF" // ● filled circle
	tsOfflineDot  = "\u25CB" // ○ empty circle
	tsExitNodeDot = "\u25C9" // ◉ fisheye
)

// Color constants for Tailscale widget elements.
const (
	tsColorGreen  = "#10B981"
	tsColorBlue   = "#3B82F6"
	tsColorGray   = "#6B7280"
	tsColorYellow = "#F59E0B"
)

// TailscaleWidget displays the Tailscale mesh network status in the dashboard.
// It shows peer connectivity, exit node information, and network health.
type TailscaleWidget struct {
	status       *tailscale.Status
	expanded     bool
	scrollOffset int
	selectedPeer int
	// nowFunc allows tests to override time.Now for deterministic output.
	nowFunc func() time.Time
}

// NewTailscaleWidget creates a new TailscaleWidget with default state.
func NewTailscaleWidget() *TailscaleWidget {
	return &TailscaleWidget{
		selectedPeer: -1,
		nowFunc:      time.Now,
	}
}

// ID returns the unique identifier for this widget.
func (w *TailscaleWidget) ID() string {
	return "tailscale"
}

// Title returns the human-readable display name.
func (w *TailscaleWidget) Title() string {
	return "Tailscale"
}

// MinSize returns the minimum width and height this widget requires.
func (w *TailscaleWidget) MinSize() (int, int) {
	return 25, 4
}

// Update handles messages directed at this widget. It processes DataUpdateEvent
// messages with Source="tailscale" and stores the Status data.
func (w *TailscaleWidget) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case app.DataUpdateEvent:
		if msg.Source != "tailscale" {
			return nil
		}
		if msg.Err != nil {
			return nil
		}
		if st, ok := msg.Data.(*tailscale.Status); ok {
			w.status = st
			// Reset scroll if peer list changed size.
			peers := w.tsSortedPeers()
			if w.scrollOffset >= len(peers) {
				w.scrollOffset = 0
			}
		}
	}
	return nil
}

// HandleKey processes a key event when this widget has focus.
func (w *TailscaleWidget) HandleKey(key tea.KeyMsg) tea.Cmd {
	switch key.String() {
	case "e":
		w.expanded = !w.expanded
		return nil
	case "up", "k":
		if w.scrollOffset > 0 {
			w.scrollOffset--
		}
		if w.expanded && w.selectedPeer > 0 {
			w.selectedPeer--
		}
		return nil
	case "down", "j":
		peers := w.tsSortedPeers()
		if w.scrollOffset < len(peers)-1 {
			w.scrollOffset++
		}
		if w.expanded && w.selectedPeer < len(peers)-1 {
			w.selectedPeer++
		}
		return nil
	}
	return nil
}

// View renders the widget content into the given area dimensions.
func (w *TailscaleWidget) View(width, height int) string {
	if width <= 0 || height <= 0 {
		return ""
	}

	if w.status == nil {
		return w.tsRenderNoData(width, height)
	}

	if w.expanded {
		return w.tsRenderExpanded(width, height)
	}
	return w.tsRenderCompact(width, height)
}

// tsRenderNoData renders a placeholder when no data is available.
func (w *TailscaleWidget) tsRenderNoData(width, height int) string {
	msg := components.Dim("No data")
	lines := make([]string, height)
	for i := range lines {
		if i == 0 {
			lines[i] = components.PadRight(msg, width)
		} else {
			lines[i] = strings.Repeat(" ", width)
		}
	}
	return strings.Join(lines, "\n")
}

// tsRenderCompact renders the compact view: header, exit node, self, peer list.
func (w *TailscaleWidget) tsRenderCompact(width, height int) string {
	var lines []string

	// Header line: Tailnet name and peer count.
	header := w.tsHeaderLine(width)
	lines = append(lines, components.PadRight(header, width))

	// Exit node line (if active).
	if w.status.ExitNode != nil && height > 2 {
		exitLine := w.tsExitNodeLine(width)
		lines = append(lines, components.PadRight(exitLine, width))
	}

	// Self node line.
	if height > len(lines) {
		selfLine := w.tsSelfLine(width)
		lines = append(lines, components.PadRight(selfLine, width))
	}

	// Peer list.
	peers := w.tsSortedPeers()
	availableRows := height - len(lines)
	startIdx := w.scrollOffset
	if startIdx > len(peers) {
		startIdx = len(peers)
	}

	for i := startIdx; i < len(peers) && availableRows > 0; i++ {
		p := peers[i]
		peerLine := w.tsPeerLineCompact(p, width)
		lines = append(lines, components.PadRight(peerLine, width))
		availableRows--
	}

	// Fill remaining lines.
	for len(lines) < height {
		lines = append(lines, strings.Repeat(" ", width))
	}

	if len(lines) > height {
		lines = lines[:height]
	}

	return strings.Join(lines, "\n")
}

// tsRenderExpanded renders the expanded view with a full DataTable.
func (w *TailscaleWidget) tsRenderExpanded(width, height int) string {
	var lines []string

	// Header line.
	header := w.tsHeaderLine(width)
	lines = append(lines, components.PadRight(header, width))

	// Exit node line.
	if w.status.ExitNode != nil && height > 3 {
		exitLine := w.tsExitNodeLine(width)
		lines = append(lines, components.PadRight(exitLine, width))
	}

	// Self node line.
	if height > len(lines)+1 {
		selfLine := w.tsSelfLine(width)
		lines = append(lines, components.PadRight(selfLine, width))
	}

	// Build the DataTable for peers.
	tableHeight := height - len(lines)
	if tableHeight <= 0 {
		for len(lines) < height {
			lines = append(lines, strings.Repeat(" ", width))
		}
		return strings.Join(lines[:height], "\n")
	}

	dt := components.NewDataTable(components.DataTableConfig{
		Columns: []components.Column{
			{Title: "St", Sizing: components.SizingFixed(2), Align: components.ColAlignCenter},
			{Title: "Hostname", Sizing: components.SizingFill(), Align: components.ColAlignLeft, MinWidth: 6},
			{Title: "DNS", Sizing: components.SizingFill(), Align: components.ColAlignLeft, MinWidth: 6},
			{Title: "IP", Sizing: components.SizingFixed(15), Align: components.ColAlignLeft},
			{Title: "OS", Sizing: components.SizingFixed(7), Align: components.ColAlignLeft},
			{Title: "Seen", Sizing: components.SizingFixed(8), Align: components.ColAlignRight},
			{Title: "Traffic", Sizing: components.SizingFixed(12), Align: components.ColAlignRight},
		},
		HeaderStyle: components.HeaderStyleConfig{
			Bold:    true,
			FgColor: ColorAccent,
		},
		ShowHeader: true,
		ShowBorder: true,
		Selectable: true,
	})

	peers := w.tsSortedPeers()
	rows := make([]components.Row, 0, len(peers))
	for _, p := range peers {
		var statusDot string
		if p.ExitNode {
			statusDot = components.Color(tsColorBlue) + tsExitNodeDot + components.Reset()
		} else if p.Online {
			statusDot = components.Color(tsColorGreen) + tsOnlineDot + components.Reset()
		} else {
			statusDot = components.Dim(tsOfflineDot)
		}

		ip := ""
		if len(p.TailscaleIPs) > 0 {
			ip = p.TailscaleIPs[0]
		}

		dns := tsStripDNSSuffix(p.DNSName, w.status.MagicDNSSuffix)

		seen := w.tsFormatLastSeen(p.LastSeen, p.Online)
		traffic := tsFormatTraffic(p.RxBytes, p.TxBytes)

		rows = append(rows, components.Row{
			Cells: []string{statusDot, p.Hostname, dns, ip, p.OS, seen, traffic},
			ID:    p.ID,
		})
	}

	dt.SetRows(rows)
	if w.selectedPeer >= 0 && w.selectedPeer < len(rows) {
		// Move selection to match.
		for i := 0; i <= w.selectedPeer && i < len(rows); i++ {
			dt.SelectNext()
		}
	}

	tableContent := dt.Render(width, tableHeight)
	lines = append(lines, strings.Split(tableContent, "\n")...)

	// Ensure exact height.
	for len(lines) < height {
		lines = append(lines, strings.Repeat(" ", width))
	}
	if len(lines) > height {
		lines = lines[:height]
	}

	return strings.Join(lines, "\n")
}

// tsHeaderLine builds the header summary line.
func (w *TailscaleWidget) tsHeaderLine(width int) string {
	tailnetName := w.status.TailnetName
	if tailnetName == "" {
		tailnetName = w.status.MagicDNSSuffix
	}

	summary := fmt.Sprintf("%d/%d peers online",
		w.status.OnlinePeers, w.status.TotalPeers)

	if tailnetName != "" {
		line := fmt.Sprintf("%s %s %s",
			components.Bold(tailnetName),
			components.Dim("\u2022"),
			summary)
		return components.Truncate(line, width)
	}

	return components.Truncate(summary, width)
}

// tsExitNodeLine builds the exit node display line.
func (w *TailscaleWidget) tsExitNodeLine(width int) string {
	if w.status.ExitNode == nil {
		return ""
	}
	en := w.status.ExitNode
	ip := ""
	if len(en.TailscaleIPs) > 0 {
		ip = en.TailscaleIPs[0]
	}
	dot := components.Color(tsColorBlue) + tsExitNodeDot + components.Reset()
	line := fmt.Sprintf("%s Exit: %s (%s)", dot, en.Hostname, ip)
	return components.Truncate(line, width)
}

// tsSelfLine builds the self node display line.
func (w *TailscaleWidget) tsSelfLine(width int) string {
	self := w.status.Self
	ip := ""
	if len(self.TailscaleIPs) > 0 {
		ip = self.TailscaleIPs[0]
	}
	dot := components.Color(tsColorGreen) + tsOnlineDot + components.Reset()
	selfLabel := components.Dim("self")
	var line string
	if ip != "" {
		line = fmt.Sprintf("%s %s %s  %s  %s", dot, self.Hostname, selfLabel, self.OS, ip)
	} else {
		line = fmt.Sprintf("%s %s %s  %s", dot, self.Hostname, selfLabel, self.OS)
	}
	return components.Truncate(line, width)
}

// tsPeerLineCompact builds a single peer line for compact view.
func (w *TailscaleWidget) tsPeerLineCompact(p tailscale.PeerInfo, width int) string {
	var dot string
	if p.ExitNode {
		dot = components.Color(tsColorBlue) + tsExitNodeDot + components.Reset()
	} else if p.Online {
		dot = components.Color(tsColorGreen) + tsOnlineDot + components.Reset()
	} else {
		dot = components.Dim(tsOfflineDot)
	}

	ip := ""
	if len(p.TailscaleIPs) > 0 {
		ip = p.TailscaleIPs[0]
	}

	var line string
	if p.Online {
		line = fmt.Sprintf("%s %-16s %-7s %s", dot, p.Hostname, p.OS, ip)
	} else {
		seen := w.tsFormatLastSeen(p.LastSeen, false)
		line = fmt.Sprintf("%s %s", dot,
			components.Dim(fmt.Sprintf("%-16s %-7s %s  (%s)", p.Hostname, p.OS, ip, seen)))
	}

	return components.Truncate(line, width)
}

// tsSortedPeers returns the peers sorted: online first (alphabetical),
// then offline (alphabetical).
func (w *TailscaleWidget) tsSortedPeers() []tailscale.PeerInfo {
	if w.status == nil {
		return nil
	}

	peers := make([]tailscale.PeerInfo, len(w.status.Peers))
	copy(peers, w.status.Peers)

	sort.Slice(peers, func(i, j int) bool {
		// Online peers come first.
		if peers[i].Online != peers[j].Online {
			return peers[i].Online
		}
		// Within same status, sort alphabetically by hostname.
		return peers[i].Hostname < peers[j].Hostname
	})

	return peers
}

// tsFormatLastSeen formats a time.Time as a relative duration string.
func (w *TailscaleWidget) tsFormatLastSeen(t time.Time, online bool) string {
	if online {
		return "now"
	}
	if t.IsZero() {
		return "never"
	}

	now := w.nowFunc()
	d := now.Sub(t)
	if d < 0 {
		return "now"
	}

	if d < time.Minute {
		return "now"
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	}
	if d < 7*24*time.Hour {
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
	return t.Format("Jan 02")
}

// tsStripDNSSuffix removes the MagicDNS suffix and trailing dot from a DNS name
// for more compact display.
func tsStripDNSSuffix(dnsName, suffix string) string {
	if dnsName == "" {
		return ""
	}
	// Remove trailing dot.
	name := strings.TrimSuffix(dnsName, ".")
	// Remove the suffix (e.g., ".tinyland.ts.net").
	if suffix != "" {
		name = strings.TrimSuffix(name, "."+suffix)
	}
	return name
}

// tsFormatTraffic formats RX/TX byte counters into a compact string.
func tsFormatTraffic(rx, tx int64) string {
	if rx == 0 && tx == 0 {
		return "-"
	}
	return fmt.Sprintf("%s/%s", tsFormatBytes(rx), tsFormatBytes(tx))
}

// tsFormatBytes formats a byte count into a human-readable string.
func tsFormatBytes(b int64) string {
	switch {
	case b >= 1<<30:
		return fmt.Sprintf("%.1fG", float64(b)/float64(1<<30))
	case b >= 1<<20:
		return fmt.Sprintf("%.1fM", float64(b)/float64(1<<20))
	case b >= 1<<10:
		return fmt.Sprintf("%.1fK", float64(b)/float64(1<<10))
	default:
		return fmt.Sprintf("%dB", b)
	}
}

// Compile-time check that TailscaleWidget satisfies the Widget interface.
var _ app.Widget = (*TailscaleWidget)(nil)
