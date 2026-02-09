package widgets

import (
	"errors"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"gitlab.com/tinyland/lab/prompt-pulse/pkg/app"
	"gitlab.com/tinyland/lab/prompt-pulse/pkg/collectors/tailscale"
)

// tsFixedNow returns a function that always returns the same time, for
// deterministic test output.
func tsFixedNow(t time.Time) func() time.Time {
	return func() time.Time { return t }
}

// tsBuildTestStatus constructs a tailscale.Status with realistic data for
// widget testing. It has 3 peers (2 online, 1 offline) and a self node.
func tsBuildTestStatus(now time.Time) *tailscale.Status {
	return &tailscale.Status{
		Self: tailscale.PeerInfo{
			ID:           "self-id",
			Hostname:     "xoxd-bates",
			DNSName:      "xoxd-bates.tinyland.ts.net.",
			OS:           "macOS",
			TailscaleIPs: []string{"100.64.0.1", "fd7a:115c:a1e0::1"},
			Online:       true,
		},
		Peers: []tailscale.PeerInfo{
			{
				ID:           "peer-honey",
				Hostname:     "honey",
				DNSName:      "honey.tinyland.ts.net.",
				OS:           "linux",
				TailscaleIPs: []string{"100.64.0.2"},
				Online:       true,
				LastSeen:     now.Add(-5 * time.Minute),
				RxBytes:      4096,
				TxBytes:      8192,
			},
			{
				ID:           "peer-pzm",
				Hostname:     "petting-zoo",
				DNSName:      "petting-zoo.tinyland.ts.net.",
				OS:           "macOS",
				TailscaleIPs: []string{"100.64.0.3"},
				Online:       true,
				LastSeen:     now.Add(-1 * time.Minute),
				RxBytes:      512,
				TxBytes:      256,
			},
			{
				ID:           "peer-offline",
				Hostname:     "offline-box",
				DNSName:      "offline-box.tinyland.ts.net.",
				OS:           "linux",
				TailscaleIPs: []string{"100.64.0.4"},
				Online:       false,
				LastSeen:     now.Add(-2 * time.Hour),
			},
		},
		MagicDNSSuffix: "tinyland.ts.net",
		TailnetName:    "tinyland.ts.net",
		OnlinePeers:    2,
		TotalPeers:     3,
		Timestamp:      now,
	}
}

// tsBuildExitNodeStatus adds an exit node to the test status.
func tsBuildExitNodeStatus(now time.Time) *tailscale.Status {
	st := tsBuildTestStatus(now)
	exitPeer := st.Peers[0] // honey
	exitPeer.ExitNode = true
	st.Peers[0] = exitPeer
	st.ExitNode = &exitPeer
	return st
}

func TestTailscaleID(t *testing.T) {
	w := NewTailscaleWidget()
	if got := w.ID(); got != "tailscale" {
		t.Errorf("ID() = %q, want %q", got, "tailscale")
	}
}

func TestTailscaleTitle(t *testing.T) {
	w := NewTailscaleWidget()
	if got := w.Title(); got != "Tailscale" {
		t.Errorf("Title() = %q, want %q", got, "Tailscale")
	}
}

func TestTailscaleMinSize(t *testing.T) {
	w := NewTailscaleWidget()
	minW, minH := w.MinSize()
	if minW != 25 {
		t.Errorf("MinSize() width = %d, want 25", minW)
	}
	if minH != 4 {
		t.Errorf("MinSize() height = %d, want 4", minH)
	}
}

func TestView_NoData(t *testing.T) {
	w := NewTailscaleWidget()
	view := w.View(40, 5)
	if !strings.Contains(view, "No data") {
		t.Errorf("View with no data should contain 'No data', got: %q", view)
	}
}

func TestView_NoData_LineCount(t *testing.T) {
	w := NewTailscaleWidget()
	view := w.View(40, 5)
	lines := strings.Split(view, "\n")
	if len(lines) != 5 {
		t.Errorf("View with no data should have 5 lines, got %d", len(lines))
	}
}

func TestView_CompactWithPeers(t *testing.T) {
	now := time.Date(2026, 2, 1, 12, 0, 0, 0, time.UTC)
	w := NewTailscaleWidget()
	w.nowFunc = tsFixedNow(now)
	w.status = tsBuildTestStatus(now)

	view := w.View(60, 10)

	// Should contain the tailnet name.
	if !strings.Contains(view, "tinyland.ts.net") {
		t.Errorf("compact view should contain tailnet name, got: %q", view)
	}

	// Should contain the peer count summary.
	if !strings.Contains(view, "2/3 peers online") {
		t.Errorf("compact view should contain '2/3 peers online', got: %q", view)
	}

	// Should contain self hostname.
	if !strings.Contains(view, "xoxd-bates") {
		t.Errorf("compact view should contain self hostname 'xoxd-bates', got: %q", view)
	}

	// Should contain peer hostnames.
	if !strings.Contains(view, "honey") {
		t.Errorf("compact view should contain peer 'honey', got: %q", view)
	}
	if !strings.Contains(view, "petting-zoo") {
		t.Errorf("compact view should contain peer 'petting-zoo', got: %q", view)
	}
}

func TestView_ExpandedWithTable(t *testing.T) {
	now := time.Date(2026, 2, 1, 12, 0, 0, 0, time.UTC)
	w := NewTailscaleWidget()
	w.nowFunc = tsFixedNow(now)
	w.status = tsBuildTestStatus(now)
	w.expanded = true

	view := w.View(80, 24)

	// Should contain table headers.
	if !strings.Contains(view, "Hostname") {
		t.Errorf("expanded view should contain 'Hostname' header, got: %q", view)
	}

	// Should contain peer hostnames.
	if !strings.Contains(view, "honey") {
		t.Errorf("expanded view should contain peer 'honey', got: %q", view)
	}

	// Should contain the header line.
	if !strings.Contains(view, "tinyland.ts.net") {
		t.Errorf("expanded view should contain tailnet name, got: %q", view)
	}
}

func TestUpdate_TailscaleData(t *testing.T) {
	now := time.Date(2026, 2, 1, 12, 0, 0, 0, time.UTC)
	w := NewTailscaleWidget()
	st := tsBuildTestStatus(now)

	msg := app.DataUpdateEvent{
		Source: "tailscale",
		Data:   st,
	}

	cmd := w.Update(msg)
	if cmd != nil {
		t.Errorf("Update should return nil cmd, got non-nil")
	}

	if w.status != st {
		t.Error("Update should have stored the status")
	}
}

func TestUpdate_IgnoresOtherSources(t *testing.T) {
	w := NewTailscaleWidget()

	msg := app.DataUpdateEvent{
		Source: "sysmetrics",
		Data:   "some data",
	}

	w.Update(msg)

	if w.status != nil {
		t.Error("Update should not store data from other sources")
	}
}

func TestUpdate_IgnoresErrors(t *testing.T) {
	now := time.Date(2026, 2, 1, 12, 0, 0, 0, time.UTC)
	w := NewTailscaleWidget()
	w.status = tsBuildTestStatus(now) // existing data

	msg := app.DataUpdateEvent{
		Source: "tailscale",
		Data:   nil,
		Err:    errors.New("connection failed"),
	}

	w.Update(msg)

	// Status should not be cleared on error.
	if w.status == nil {
		t.Error("Update should not clear status on error")
	}
}

func TestHandleKey_ToggleExpanded(t *testing.T) {
	w := NewTailscaleWidget()

	if w.expanded {
		t.Error("widget should start in compact mode")
	}

	w.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	if !w.expanded {
		t.Error("'e' key should toggle expanded to true")
	}

	w.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	if w.expanded {
		t.Error("'e' key should toggle expanded back to false")
	}
}

func TestOnlinePeersSortedFirst(t *testing.T) {
	now := time.Date(2026, 2, 1, 12, 0, 0, 0, time.UTC)
	w := NewTailscaleWidget()
	w.nowFunc = tsFixedNow(now)
	w.status = tsBuildTestStatus(now)

	peers := w.tsSortedPeers()

	if len(peers) != 3 {
		t.Fatalf("expected 3 peers, got %d", len(peers))
	}

	// First two should be online, sorted alphabetically.
	if !peers[0].Online {
		t.Errorf("peers[0] should be online, got offline: %s", peers[0].Hostname)
	}
	if !peers[1].Online {
		t.Errorf("peers[1] should be online, got offline: %s", peers[1].Hostname)
	}
	if peers[2].Online {
		t.Errorf("peers[2] should be offline, got online: %s", peers[2].Hostname)
	}

	// Online peers in alphabetical order.
	if peers[0].Hostname != "honey" {
		t.Errorf("peers[0].Hostname = %q, want 'honey'", peers[0].Hostname)
	}
	if peers[1].Hostname != "petting-zoo" {
		t.Errorf("peers[1].Hostname = %q, want 'petting-zoo'", peers[1].Hostname)
	}
	if peers[2].Hostname != "offline-box" {
		t.Errorf("peers[2].Hostname = %q, want 'offline-box'", peers[2].Hostname)
	}
}

func TestExitNodeDisplay(t *testing.T) {
	now := time.Date(2026, 2, 1, 12, 0, 0, 0, time.UTC)
	w := NewTailscaleWidget()
	w.nowFunc = tsFixedNow(now)
	w.status = tsBuildExitNodeStatus(now)

	view := w.View(60, 10)

	if !strings.Contains(view, "Exit") {
		t.Errorf("view should contain 'Exit' for exit node, got: %q", view)
	}
	if !strings.Contains(view, "honey") {
		t.Errorf("view should contain exit node hostname 'honey', got: %q", view)
	}
	if !strings.Contains(view, "100.64.0.2") {
		t.Errorf("view should contain exit node IP '100.64.0.2', got: %q", view)
	}
}

func TestSelfNodeDisplay(t *testing.T) {
	now := time.Date(2026, 2, 1, 12, 0, 0, 0, time.UTC)
	w := NewTailscaleWidget()
	w.nowFunc = tsFixedNow(now)
	w.status = tsBuildTestStatus(now)

	view := w.View(60, 10)

	if !strings.Contains(view, "xoxd-bates") {
		t.Errorf("view should contain self hostname 'xoxd-bates', got: %q", view)
	}
	if !strings.Contains(view, "self") {
		t.Errorf("view should contain 'self' label, got: %q", view)
	}
	if !strings.Contains(view, "macOS") {
		t.Errorf("view should contain self OS 'macOS', got: %q", view)
	}
}

func TestTimeFormatting(t *testing.T) {
	now := time.Date(2026, 2, 1, 12, 0, 0, 0, time.UTC)
	w := NewTailscaleWidget()
	w.nowFunc = tsFixedNow(now)

	tests := []struct {
		name     string
		lastSeen time.Time
		online   bool
		want     string
	}{
		{name: "online", lastSeen: now, online: true, want: "now"},
		{name: "30 seconds ago", lastSeen: now.Add(-30 * time.Second), online: false, want: "now"},
		{name: "5 minutes ago", lastSeen: now.Add(-5 * time.Minute), online: false, want: "5m ago"},
		{name: "45 minutes ago", lastSeen: now.Add(-45 * time.Minute), online: false, want: "45m ago"},
		{name: "2 hours ago", lastSeen: now.Add(-2 * time.Hour), online: false, want: "2h ago"},
		{name: "23 hours ago", lastSeen: now.Add(-23 * time.Hour), online: false, want: "23h ago"},
		{name: "3 days ago", lastSeen: now.Add(-3 * 24 * time.Hour), online: false, want: "3d ago"},
		{name: "6 days ago", lastSeen: now.Add(-6 * 24 * time.Hour), online: false, want: "6d ago"},
		{name: "10 days ago", lastSeen: now.Add(-10 * 24 * time.Hour), online: false, want: "Jan 22"},
		{name: "zero time", lastSeen: time.Time{}, online: false, want: "never"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := w.tsFormatLastSeen(tc.lastSeen, tc.online)
			if got != tc.want {
				t.Errorf("tsFormatLastSeen(%v, %v) = %q, want %q",
					tc.lastSeen, tc.online, got, tc.want)
			}
		})
	}
}

func TestView_SmallSize_25x4(t *testing.T) {
	now := time.Date(2026, 2, 1, 12, 0, 0, 0, time.UTC)
	w := NewTailscaleWidget()
	w.nowFunc = tsFixedNow(now)
	w.status = tsBuildTestStatus(now)

	view := w.View(25, 4)
	lines := strings.Split(view, "\n")

	if len(lines) != 4 {
		t.Errorf("View(25, 4) should produce 4 lines, got %d", len(lines))
	}

	// Should still contain some meaningful content.
	if !strings.Contains(view, "2/3") {
		t.Errorf("small view should contain peer count, got: %q", view)
	}
}

func TestView_MediumSize_60x15(t *testing.T) {
	now := time.Date(2026, 2, 1, 12, 0, 0, 0, time.UTC)
	w := NewTailscaleWidget()
	w.nowFunc = tsFixedNow(now)
	w.status = tsBuildTestStatus(now)

	view := w.View(60, 15)
	lines := strings.Split(view, "\n")

	if len(lines) != 15 {
		t.Errorf("View(60, 15) should produce 15 lines, got %d", len(lines))
	}
}

func TestView_LargeSize_80x24(t *testing.T) {
	now := time.Date(2026, 2, 1, 12, 0, 0, 0, time.UTC)
	w := NewTailscaleWidget()
	w.nowFunc = tsFixedNow(now)
	w.status = tsBuildTestStatus(now)
	w.expanded = true

	view := w.View(80, 24)
	lines := strings.Split(view, "\n")

	if len(lines) != 24 {
		t.Errorf("View(80, 24) should produce 24 lines, got %d", len(lines))
	}
}

func TestView_ZeroPeers(t *testing.T) {
	now := time.Date(2026, 2, 1, 12, 0, 0, 0, time.UTC)
	w := NewTailscaleWidget()
	w.nowFunc = tsFixedNow(now)
	w.status = &tailscale.Status{
		Self: tailscale.PeerInfo{
			ID:           "self-only",
			Hostname:     "lonely-node",
			DNSName:      "lonely-node.solo.ts.net.",
			OS:           "linux",
			TailscaleIPs: []string{"100.64.0.1"},
			Online:       true,
		},
		Peers:          nil,
		MagicDNSSuffix: "solo.ts.net",
		TailnetName:    "solo.ts.net",
		OnlinePeers:    0,
		TotalPeers:     0,
	}

	view := w.View(50, 8)

	if !strings.Contains(view, "0/0 peers online") {
		t.Errorf("zero peers view should show '0/0 peers online', got: %q", view)
	}

	if !strings.Contains(view, "lonely-node") {
		t.Errorf("zero peers view should show self hostname 'lonely-node', got: %q", view)
	}

	lines := strings.Split(view, "\n")
	if len(lines) != 8 {
		t.Errorf("View should produce 8 lines, got %d", len(lines))
	}
}

func TestView_ZeroSize(t *testing.T) {
	w := NewTailscaleWidget()
	if got := w.View(0, 0); got != "" {
		t.Errorf("View(0, 0) should return empty string, got %q", got)
	}
	if got := w.View(-1, 5); got != "" {
		t.Errorf("View(-1, 5) should return empty string, got %q", got)
	}
	if got := w.View(5, -1); got != "" {
		t.Errorf("View(5, -1) should return empty string, got %q", got)
	}
}

func TestHandleKey_ScrollDown(t *testing.T) {
	now := time.Date(2026, 2, 1, 12, 0, 0, 0, time.UTC)
	w := NewTailscaleWidget()
	w.nowFunc = tsFixedNow(now)
	w.status = tsBuildTestStatus(now)

	if w.scrollOffset != 0 {
		t.Errorf("initial scrollOffset = %d, want 0", w.scrollOffset)
	}

	w.HandleKey(tea.KeyMsg{Type: tea.KeyDown})
	if w.scrollOffset != 1 {
		t.Errorf("after down key scrollOffset = %d, want 1", w.scrollOffset)
	}
}

func TestHandleKey_ScrollUp(t *testing.T) {
	now := time.Date(2026, 2, 1, 12, 0, 0, 0, time.UTC)
	w := NewTailscaleWidget()
	w.nowFunc = tsFixedNow(now)
	w.status = tsBuildTestStatus(now)
	w.scrollOffset = 2

	w.HandleKey(tea.KeyMsg{Type: tea.KeyUp})
	if w.scrollOffset != 1 {
		t.Errorf("after up key scrollOffset = %d, want 1", w.scrollOffset)
	}
}

func TestHandleKey_ScrollUpAtZero(t *testing.T) {
	w := NewTailscaleWidget()
	w.scrollOffset = 0

	w.HandleKey(tea.KeyMsg{Type: tea.KeyUp})
	if w.scrollOffset != 0 {
		t.Errorf("scrollOffset should stay at 0, got %d", w.scrollOffset)
	}
}

func TestHandleKey_UnhandledKey(t *testing.T) {
	w := NewTailscaleWidget()
	cmd := w.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'z'}})
	if cmd != nil {
		t.Error("unhandled key should return nil cmd")
	}
}

func TestStripDNSSuffix(t *testing.T) {
	tests := []struct {
		dnsName string
		suffix  string
		want    string
	}{
		{"honey.tinyland.ts.net.", "tinyland.ts.net", "honey"},
		{"xoxd-bates.tinyland.ts.net.", "tinyland.ts.net", "xoxd-bates"},
		{"", "tinyland.ts.net", ""},
		{"honey.tinyland.ts.net.", "", "honey.tinyland.ts.net"},
		{"standalone.", "other.ts.net", "standalone"},
	}

	for _, tc := range tests {
		t.Run(tc.dnsName, func(t *testing.T) {
			got := tsStripDNSSuffix(tc.dnsName, tc.suffix)
			if got != tc.want {
				t.Errorf("tsStripDNSSuffix(%q, %q) = %q, want %q",
					tc.dnsName, tc.suffix, got, tc.want)
			}
		})
	}
}

func TestTailscaleFormatBytes(t *testing.T) {
	tests := []struct {
		bytes int64
		want  string
	}{
		{0, "0B"},
		{512, "512B"},
		{1024, "1.0K"},
		{1536, "1.5K"},
		{1048576, "1.0M"},
		{1073741824, "1.0G"},
	}

	for _, tc := range tests {
		t.Run(tc.want, func(t *testing.T) {
			got := tsFormatBytes(tc.bytes)
			if got != tc.want {
				t.Errorf("tsFormatBytes(%d) = %q, want %q", tc.bytes, got, tc.want)
			}
		})
	}
}

func TestFormatTraffic(t *testing.T) {
	got := tsFormatTraffic(0, 0)
	if got != "-" {
		t.Errorf("tsFormatTraffic(0, 0) = %q, want '-'", got)
	}

	got = tsFormatTraffic(4096, 8192)
	if got != "4.0K/8.0K" {
		t.Errorf("tsFormatTraffic(4096, 8192) = %q, want '4.0K/8.0K'", got)
	}
}

// Compile-time check that TailscaleWidget satisfies the Widget interface.
var _ app.Widget = (*TailscaleWidget)(nil)
