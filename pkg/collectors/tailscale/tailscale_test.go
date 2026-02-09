package tailscale

import (
	"context"
	"errors"
	"net/netip"
	"testing"
	"time"

	"go4.org/mem"
	"tailscale.com/ipn/ipnstate"
	"tailscale.com/tailcfg"
	"tailscale.com/types/key"
	"tailscale.com/types/views"
)

// mockClient is a test double for StatusClient.
type mockClient struct {
	status *ipnstate.Status
	err    error
}

func (m *mockClient) Status(ctx context.Context) (*ipnstate.Status, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return m.status, m.err
}

// makePeerKey creates a deterministic key.NodePublic for testing.
// Each byte index gets a unique value so peers are distinguishable.
func makePeerKey(id byte) key.NodePublic {
	var raw [32]byte
	raw[0] = id
	return key.NodePublicFromRaw32(mem.B(raw[:]))
}

// buildTestStatus creates a populated ipnstate.Status for testing.
// It returns a status with one self node and three peers (two online, one offline).
func buildTestStatus() *ipnstate.Status {
	selfKey := makePeerKey(0)
	peer1Key := makePeerKey(1)
	peer2Key := makePeerKey(2)
	peer3Key := makePeerKey(3)

	lastSeen := time.Date(2026, 2, 1, 12, 0, 0, 0, time.UTC)

	tags := views.SliceOf([]string{"tag:server", "tag:production"})

	return &ipnstate.Status{
		BackendState: "Running",
		MagicDNSSuffix: "tinyland.ts.net",
		CurrentTailnet: &ipnstate.TailnetStatus{
			Name:           "tinyland.ts.net",
			MagicDNSSuffix: "tinyland.ts.net",
			MagicDNSEnabled: true,
		},
		TailscaleIPs: []netip.Addr{
			netip.MustParseAddr("100.64.0.1"),
			netip.MustParseAddr("fd7a:115c:a1e0::1"),
		},
		Self: &ipnstate.PeerStatus{
			ID:           "self-stable-id",
			PublicKey:    selfKey,
			HostName:     "xoxd-bates",
			DNSName:      "xoxd-bates.tinyland.ts.net.",
			OS:           "macOS",
			TailscaleIPs: []netip.Addr{
				netip.MustParseAddr("100.64.0.1"),
				netip.MustParseAddr("fd7a:115c:a1e0::1"),
			},
			Online:         true,
			ExitNodeOption: true,
			RxBytes:        1024,
			TxBytes:        2048,
		},
		Peer: map[key.NodePublic]*ipnstate.PeerStatus{
			peer1Key: {
				ID:             "peer-honey",
				PublicKey:      peer1Key,
				HostName:       "honey",
				DNSName:        "honey.tinyland.ts.net.",
				OS:             "linux",
				TailscaleIPs:   []netip.Addr{netip.MustParseAddr("100.64.0.2")},
				Online:         true,
				LastSeen:       lastSeen,
				ExitNode:       false,
				ExitNodeOption: true,
				Tags:           &tags,
				RxBytes:        4096,
				TxBytes:        8192,
			},
			peer2Key: {
				ID:             "peer-pzm",
				PublicKey:      peer2Key,
				HostName:       "petting-zoo-mini",
				DNSName:        "petting-zoo-mini.tinyland.ts.net.",
				OS:             "macOS",
				TailscaleIPs:   []netip.Addr{netip.MustParseAddr("100.64.0.3")},
				Online:         true,
				LastSeen:       lastSeen,
				ExitNode:       false,
				ExitNodeOption: false,
				RxBytes:        512,
				TxBytes:        256,
			},
			peer3Key: {
				ID:             "peer-offline",
				PublicKey:      peer3Key,
				HostName:       "offline-box",
				DNSName:        "offline-box.tinyland.ts.net.",
				OS:             "linux",
				TailscaleIPs:   []netip.Addr{netip.MustParseAddr("100.64.0.4")},
				Online:         false,
				LastSeen:       lastSeen.Add(-24 * time.Hour),
				ExitNode:       false,
				ExitNodeOption: false,
			},
		},
	}
}

func TestName(t *testing.T) {
	c := New(Config{}, &mockClient{})
	if got := c.Name(); got != "tailscale" {
		t.Errorf("Name() = %q, want %q", got, "tailscale")
	}
}

func TestInterval_Default(t *testing.T) {
	c := New(Config{}, &mockClient{})
	if got := c.Interval(); got != DefaultInterval {
		t.Errorf("Interval() = %v, want %v", got, DefaultInterval)
	}
}

func TestInterval_Custom(t *testing.T) {
	want := 30 * time.Second
	c := New(Config{Interval: want}, &mockClient{})
	if got := c.Interval(); got != want {
		t.Errorf("Interval() = %v, want %v", got, want)
	}
}

func TestInterval_ZeroUsesDefault(t *testing.T) {
	c := New(Config{Interval: 0}, &mockClient{})
	if got := c.Interval(); got != DefaultInterval {
		t.Errorf("Interval() = %v, want default %v", got, DefaultInterval)
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := Config{}
	if cfg.Interval != 0 {
		t.Errorf("default Config.Interval = %v, want 0", cfg.Interval)
	}
	if cfg.SocketPath != "" {
		t.Errorf("default Config.SocketPath = %q, want empty", cfg.SocketPath)
	}
}

func TestCollect_OnlinePeerCount(t *testing.T) {
	st := buildTestStatus()
	mc := &mockClient{status: st}
	c := New(Config{}, mc)

	result, err := c.Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect() error: %v", err)
	}

	status, ok := result.(*Status)
	if !ok {
		t.Fatalf("Collect() returned %T, want *Status", result)
	}

	// 3 peers: 2 online (honey, petting-zoo-mini), 1 offline
	if status.OnlinePeers != 2 {
		t.Errorf("OnlinePeers = %d, want 2", status.OnlinePeers)
	}
	if status.TotalPeers != 3 {
		t.Errorf("TotalPeers = %d, want 3", status.TotalPeers)
	}
}

func TestCollect_SelfPopulated(t *testing.T) {
	st := buildTestStatus()
	mc := &mockClient{status: st}
	c := New(Config{}, mc)

	result, err := c.Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect() error: %v", err)
	}

	status := result.(*Status)

	if status.Self.Hostname != "xoxd-bates" {
		t.Errorf("Self.Hostname = %q, want %q", status.Self.Hostname, "xoxd-bates")
	}
	if status.Self.DNSName != "xoxd-bates.tinyland.ts.net." {
		t.Errorf("Self.DNSName = %q, want %q", status.Self.DNSName, "xoxd-bates.tinyland.ts.net.")
	}
	if status.Self.OS != "macOS" {
		t.Errorf("Self.OS = %q, want %q", status.Self.OS, "macOS")
	}
	if !status.Self.Online {
		t.Error("Self.Online = false, want true")
	}
	if len(status.Self.TailscaleIPs) != 2 {
		t.Fatalf("Self.TailscaleIPs len = %d, want 2", len(status.Self.TailscaleIPs))
	}
	if status.Self.TailscaleIPs[0] != "100.64.0.1" {
		t.Errorf("Self.TailscaleIPs[0] = %q, want %q", status.Self.TailscaleIPs[0], "100.64.0.1")
	}
	if status.Self.RxBytes != 1024 {
		t.Errorf("Self.RxBytes = %d, want 1024", status.Self.RxBytes)
	}
	if status.Self.TxBytes != 2048 {
		t.Errorf("Self.TxBytes = %d, want 2048", status.Self.TxBytes)
	}
}

func TestCollect_ExitNodeDetection(t *testing.T) {
	st := buildTestStatus()

	// Mark honey as the active exit node.
	for _, ps := range st.Peer {
		if ps.HostName == "honey" {
			ps.ExitNode = true
			break
		}
	}

	mc := &mockClient{status: st}
	c := New(Config{}, mc)

	result, err := c.Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect() error: %v", err)
	}

	status := result.(*Status)
	if status.ExitNode == nil {
		t.Fatal("ExitNode is nil, expected honey to be exit node")
	}
	if status.ExitNode.Hostname != "honey" {
		t.Errorf("ExitNode.Hostname = %q, want %q", status.ExitNode.Hostname, "honey")
	}
	if !status.ExitNode.ExitNode {
		t.Error("ExitNode.ExitNode = false, want true")
	}
}

func TestCollect_NoExitNode(t *testing.T) {
	st := buildTestStatus()
	mc := &mockClient{status: st}
	c := New(Config{}, mc)

	result, err := c.Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect() error: %v", err)
	}

	status := result.(*Status)
	if status.ExitNode != nil {
		t.Errorf("ExitNode = %+v, want nil", status.ExitNode)
	}
}

func TestCollect_MagicDNSSuffix(t *testing.T) {
	st := buildTestStatus()
	mc := &mockClient{status: st}
	c := New(Config{}, mc)

	result, err := c.Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect() error: %v", err)
	}

	status := result.(*Status)
	if status.MagicDNSSuffix != "tinyland.ts.net" {
		t.Errorf("MagicDNSSuffix = %q, want %q", status.MagicDNSSuffix, "tinyland.ts.net")
	}
	if status.TailnetName != "tinyland.ts.net" {
		t.Errorf("TailnetName = %q, want %q", status.TailnetName, "tinyland.ts.net")
	}
}

func TestCollect_MagicDNSSuffix_FallbackToLegacy(t *testing.T) {
	st := buildTestStatus()
	st.CurrentTailnet = nil // simulate older daemon without CurrentTailnet
	st.MagicDNSSuffix = "legacy.ts.net"

	mc := &mockClient{status: st}
	c := New(Config{}, mc)

	result, err := c.Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect() error: %v", err)
	}

	status := result.(*Status)
	if status.MagicDNSSuffix != "legacy.ts.net" {
		t.Errorf("MagicDNSSuffix = %q, want %q", status.MagicDNSSuffix, "legacy.ts.net")
	}
	if status.TailnetName != "" {
		t.Errorf("TailnetName = %q, want empty (no CurrentTailnet)", status.TailnetName)
	}
}

func TestCollect_PeerIPsAndDNS(t *testing.T) {
	st := buildTestStatus()
	mc := &mockClient{status: st}
	c := New(Config{}, mc)

	result, err := c.Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect() error: %v", err)
	}

	status := result.(*Status)

	// Find honey peer.
	var honeyPeer *PeerInfo
	for i := range status.Peers {
		if status.Peers[i].Hostname == "honey" {
			honeyPeer = &status.Peers[i]
			break
		}
	}

	if honeyPeer == nil {
		t.Fatal("peer 'honey' not found in Peers")
	}

	if honeyPeer.DNSName != "honey.tinyland.ts.net." {
		t.Errorf("honey.DNSName = %q, want %q", honeyPeer.DNSName, "honey.tinyland.ts.net.")
	}
	if len(honeyPeer.TailscaleIPs) != 1 {
		t.Fatalf("honey.TailscaleIPs len = %d, want 1", len(honeyPeer.TailscaleIPs))
	}
	if honeyPeer.TailscaleIPs[0] != "100.64.0.2" {
		t.Errorf("honey.TailscaleIPs[0] = %q, want %q", honeyPeer.TailscaleIPs[0], "100.64.0.2")
	}
}

func TestCollect_ErrorSetsUnhealthy(t *testing.T) {
	mc := &mockClient{err: errors.New("tailscaled not running")}
	c := New(Config{}, mc)

	// Starts healthy.
	if !c.Healthy() {
		t.Fatal("Healthy() = false before first collection, want true")
	}

	_, err := c.Collect(context.Background())
	if err == nil {
		t.Fatal("Collect() should have returned an error")
	}

	if c.Healthy() {
		t.Error("Healthy() = true after error, want false")
	}
}

func TestCollect_SuccessSetsHealthy(t *testing.T) {
	st := buildTestStatus()

	// First fail, then succeed.
	mc := &mockClient{err: errors.New("temporary failure")}
	c := New(Config{}, mc)

	_, _ = c.Collect(context.Background())
	if c.Healthy() {
		t.Error("Healthy() = true after error, want false")
	}

	// Switch to success.
	mc.status = st
	mc.err = nil

	_, err := c.Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect() error: %v", err)
	}

	if !c.Healthy() {
		t.Error("Healthy() = false after success, want true")
	}
}

func TestCollect_ContextCancellation(t *testing.T) {
	mc := &mockClient{status: buildTestStatus()}
	c := New(Config{}, mc)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.

	_, err := c.Collect(ctx)
	if err == nil {
		t.Fatal("Collect() should have returned an error for cancelled context")
	}
}

func TestCollect_EmptyPeerList(t *testing.T) {
	st := &ipnstate.Status{
		BackendState:   "Running",
		MagicDNSSuffix: "solo.ts.net",
		Self: &ipnstate.PeerStatus{
			ID:       "self-only",
			HostName: "lonely-node",
			DNSName:  "lonely-node.solo.ts.net.",
			OS:       "linux",
			TailscaleIPs: []netip.Addr{netip.MustParseAddr("100.64.0.1")},
			Online:   true,
		},
		Peer: map[key.NodePublic]*ipnstate.PeerStatus{},
	}

	mc := &mockClient{status: st}
	c := New(Config{}, mc)

	result, err := c.Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect() error: %v", err)
	}

	status := result.(*Status)
	if status.Self.Hostname != "lonely-node" {
		t.Errorf("Self.Hostname = %q, want %q", status.Self.Hostname, "lonely-node")
	}
	if len(status.Peers) != 0 {
		t.Errorf("Peers len = %d, want 0", len(status.Peers))
	}
	if status.TotalPeers != 0 {
		t.Errorf("TotalPeers = %d, want 0", status.TotalPeers)
	}
	if status.OnlinePeers != 0 {
		t.Errorf("OnlinePeers = %d, want 0", status.OnlinePeers)
	}
}

func TestCollect_PeerTags(t *testing.T) {
	st := buildTestStatus()
	mc := &mockClient{status: st}
	c := New(Config{}, mc)

	result, err := c.Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect() error: %v", err)
	}

	status := result.(*Status)

	// Find honey (the peer with tags).
	var honeyPeer *PeerInfo
	for i := range status.Peers {
		if status.Peers[i].Hostname == "honey" {
			honeyPeer = &status.Peers[i]
			break
		}
	}

	if honeyPeer == nil {
		t.Fatal("peer 'honey' not found")
	}

	expectedTags := []string{"tag:server", "tag:production"}
	if len(honeyPeer.Tags) != len(expectedTags) {
		t.Fatalf("honey.Tags len = %d, want %d", len(honeyPeer.Tags), len(expectedTags))
	}
	for i, want := range expectedTags {
		if honeyPeer.Tags[i] != want {
			t.Errorf("honey.Tags[%d] = %q, want %q", i, honeyPeer.Tags[i], want)
		}
	}

	// Peers without tags should have nil/empty tags.
	for _, p := range status.Peers {
		if p.Hostname == "offline-box" {
			if len(p.Tags) != 0 {
				t.Errorf("offline-box.Tags = %v, want empty", p.Tags)
			}
		}
	}
}

func TestCollect_NilSelf(t *testing.T) {
	st := &ipnstate.Status{
		BackendState:   "NeedsLogin",
		MagicDNSSuffix: "example.ts.net",
		Self:           nil, // Tailscale not fully initialized
		Peer:           map[key.NodePublic]*ipnstate.PeerStatus{},
	}

	mc := &mockClient{status: st}
	c := New(Config{}, mc)

	result, err := c.Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect() error: %v", err)
	}

	status := result.(*Status)
	// Self should be a zero PeerInfo when Self is nil.
	if status.Self.Hostname != "" {
		t.Errorf("Self.Hostname = %q, want empty for nil Self", status.Self.Hostname)
	}
}

func TestCollect_NilResponse(t *testing.T) {
	mc := &mockClient{status: nil, err: nil}
	c := New(Config{}, mc)

	_, err := c.Collect(context.Background())
	if err == nil {
		t.Fatal("Collect() should return error for nil status")
	}

	if c.Healthy() {
		t.Error("Healthy() = true after nil status, want false")
	}
}

func TestCollect_TrafficCounters(t *testing.T) {
	st := buildTestStatus()
	mc := &mockClient{status: st}
	c := New(Config{}, mc)

	result, err := c.Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect() error: %v", err)
	}

	status := result.(*Status)

	// Find honey and check traffic counters.
	for _, p := range status.Peers {
		if p.Hostname == "honey" {
			if p.RxBytes != 4096 {
				t.Errorf("honey.RxBytes = %d, want 4096", p.RxBytes)
			}
			if p.TxBytes != 8192 {
				t.Errorf("honey.TxBytes = %d, want 8192", p.TxBytes)
			}
			return
		}
	}
	t.Fatal("peer 'honey' not found")
}

func TestCollect_ExitNodeOption(t *testing.T) {
	st := buildTestStatus()
	mc := &mockClient{status: st}
	c := New(Config{}, mc)

	result, err := c.Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect() error: %v", err)
	}

	status := result.(*Status)

	// Self should have ExitNodeOption = true.
	if !status.Self.ExitNodeOption {
		t.Error("Self.ExitNodeOption = false, want true")
	}

	// honey should have ExitNodeOption = true.
	for _, p := range status.Peers {
		if p.Hostname == "honey" {
			if !p.ExitNodeOption {
				t.Error("honey.ExitNodeOption = false, want true")
			}
		}
		if p.Hostname == "petting-zoo-mini" {
			if p.ExitNodeOption {
				t.Error("petting-zoo-mini.ExitNodeOption = true, want false")
			}
		}
	}
}

func TestCollect_Timestamp(t *testing.T) {
	st := buildTestStatus()
	mc := &mockClient{status: st}
	c := New(Config{}, mc)

	before := time.Now()
	result, err := c.Collect(context.Background())
	after := time.Now()

	if err != nil {
		t.Fatalf("Collect() error: %v", err)
	}

	status := result.(*Status)
	if status.Timestamp.Before(before) || status.Timestamp.After(after) {
		t.Errorf("Timestamp = %v, want between %v and %v", status.Timestamp, before, after)
	}
}

func TestCollect_PeerStableID(t *testing.T) {
	st := buildTestStatus()
	mc := &mockClient{status: st}
	c := New(Config{}, mc)

	result, err := c.Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect() error: %v", err)
	}

	status := result.(*Status)
	if status.Self.ID != "self-stable-id" {
		t.Errorf("Self.ID = %q, want %q", status.Self.ID, "self-stable-id")
	}

	foundHoney := false
	for _, p := range status.Peers {
		if p.Hostname == "honey" {
			foundHoney = true
			if p.ID != "peer-honey" {
				t.Errorf("honey.ID = %q, want %q", p.ID, "peer-honey")
			}
		}
	}
	if !foundHoney {
		t.Fatal("peer 'honey' not found")
	}
}

// Compile-time check that Collector satisfies the interface contract.
// We define a local interface identical to pkg/collectors.Collector to avoid
// importing that package (which is being built concurrently by another agent).
type collectorIface interface {
	Name() string
	Collect(ctx context.Context) (interface{}, error)
	Interval() time.Duration
	Healthy() bool
}

var _ collectorIface = (*Collector)(nil)

// Ensure the mock satisfies StatusClient.
var _ StatusClient = (*mockClient)(nil)

// Ensure makePeerKey produces valid non-zero keys that differ.
func TestMakePeerKey_Unique(t *testing.T) {
	k1 := makePeerKey(1)
	k2 := makePeerKey(2)
	if k1 == k2 {
		t.Error("makePeerKey(1) == makePeerKey(2), want different keys")
	}
	var zero key.NodePublic
	if k1 == zero {
		t.Error("makePeerKey(1) is zero key")
	}
}

// Verify that tailcfg.StableNodeID is convertible to string (used in our mapping).
func TestStableNodeID_StringConversion(t *testing.T) {
	id := tailcfg.StableNodeID("test-id-123")
	s := string(id)
	if s != "test-id-123" {
		t.Errorf("string(StableNodeID) = %q, want %q", s, "test-id-123")
	}
}
