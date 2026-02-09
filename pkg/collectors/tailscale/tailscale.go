// Package tailscale provides a collector that gathers Tailscale network status
// from the local tailscaled daemon via the LocalAPI unix socket. It maps the
// ipnstate.Status response into a simplified Status struct for dashboard
// rendering.
package tailscale

import (
	"context"
	"fmt"
	"sync"
	"time"

	"tailscale.com/ipn/ipnstate"
	"tailscale.com/types/key"
)

// Default configuration values.
const (
	DefaultInterval = 10 * time.Second
)

// StatusClient abstracts the local Tailscale daemon API for testability.
// The real implementation is tailscale.com/client/local.Client, whose
// Status method satisfies this interface.
type StatusClient interface {
	Status(ctx context.Context) (*ipnstate.Status, error)
}

// Config holds the configuration for the Tailscale collector.
type Config struct {
	// Interval is how often collection runs. Zero uses DefaultInterval.
	Interval time.Duration

	// SocketPath is an optional custom tailscaled socket path.
	// When empty, the platform default is used.
	SocketPath string
}

// PeerInfo contains summarised information about a single Tailscale peer.
type PeerInfo struct {
	ID             string        `json:"id"`
	Hostname       string        `json:"hostname"`
	DNSName        string        `json:"dns_name"`
	OS             string        `json:"os"`
	TailscaleIPs   []string      `json:"tailscale_ips"`
	Online         bool          `json:"online"`
	LastSeen       time.Time     `json:"last_seen"`
	ExitNode       bool          `json:"exit_node"`
	ExitNodeOption bool          `json:"exit_node_option"`
	Tags           []string      `json:"tags"`
	RxBytes        int64         `json:"rx_bytes"`
	TxBytes        int64         `json:"tx_bytes"`
	Latency        time.Duration `json:"latency"`
}

// Status is the data returned by a single Collect call.
type Status struct {
	Self           PeerInfo   `json:"self"`
	Peers          []PeerInfo `json:"peers"`
	MagicDNSSuffix string     `json:"magic_dns_suffix"`
	TailnetName    string     `json:"tailnet_name"`
	OnlinePeers    int        `json:"online_peers"`
	TotalPeers     int        `json:"total_peers"`
	ExitNode       *PeerInfo  `json:"exit_node,omitempty"`
	Timestamp      time.Time  `json:"timestamp"`
}

// Collector gathers Tailscale network status from the local daemon.
type Collector struct {
	client   StatusClient
	interval time.Duration

	mu      sync.Mutex
	healthy bool
}

// New creates a new Tailscale collector. If cfg.Interval is zero,
// DefaultInterval is used. The caller must provide a StatusClient; in
// production this is a *local.Client configured with the optional
// SocketPath.
func New(cfg Config, client StatusClient) *Collector {
	interval := cfg.Interval
	if interval <= 0 {
		interval = DefaultInterval
	}
	return &Collector{
		client:   client,
		interval: interval,
		healthy:  true, // healthy until first failure
	}
}

// Name returns the collector identifier.
func (c *Collector) Name() string {
	return "tailscale"
}

// Interval returns how often this collector should run.
func (c *Collector) Interval() time.Duration {
	return c.interval
}

// Healthy returns whether the last collection succeeded.
func (c *Collector) Healthy() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.healthy
}

// setHealthy updates the internal healthy flag under the mutex.
func (c *Collector) setHealthy(v bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.healthy = v
}

// Collect calls the local Tailscale daemon and returns a Status snapshot.
func (c *Collector) Collect(ctx context.Context) (interface{}, error) {
	st, err := c.client.Status(ctx)
	if err != nil {
		c.setHealthy(false)
		return nil, fmt.Errorf("tailscale status: %w", err)
	}

	if st == nil {
		c.setHealthy(false)
		return nil, fmt.Errorf("tailscale status: nil response")
	}

	status := c.mapStatus(st)
	c.setHealthy(true)
	return status, nil
}

// mapStatus converts the ipnstate.Status into our simplified Status struct.
func (c *Collector) mapStatus(st *ipnstate.Status) *Status {
	now := time.Now()

	selfInfo := c.mapSelfPeer(st)

	// Determine the MagicDNS suffix. Prefer CurrentTailnet if available.
	magicDNS := st.MagicDNSSuffix
	tailnetName := ""
	if st.CurrentTailnet != nil {
		if st.CurrentTailnet.MagicDNSSuffix != "" {
			magicDNS = st.CurrentTailnet.MagicDNSSuffix
		}
		tailnetName = st.CurrentTailnet.Name
	}

	// Collect peers. Use the sorted key order for determinism.
	var peers []PeerInfo
	var exitNode *PeerInfo
	onlineCount := 0

	for _, pubKey := range st.Peers() {
		ps := st.Peer[pubKey]
		if ps == nil {
			continue
		}
		pi := c.mapPeerStatus(ps)
		peers = append(peers, pi)
		if pi.Online {
			onlineCount++
		}
		if pi.ExitNode {
			p := pi // copy for pointer
			exitNode = &p
		}
	}

	return &Status{
		Self:           selfInfo,
		Peers:          peers,
		MagicDNSSuffix: magicDNS,
		TailnetName:    tailnetName,
		OnlinePeers:    onlineCount,
		TotalPeers:     len(peers),
		ExitNode:       exitNode,
		Timestamp:      now,
	}
}

// mapSelfPeer extracts PeerInfo from the Self field of ipnstate.Status.
func (c *Collector) mapSelfPeer(st *ipnstate.Status) PeerInfo {
	if st.Self == nil {
		return PeerInfo{}
	}
	return c.mapPeerStatus(st.Self)
}

// mapPeerStatus converts a single ipnstate.PeerStatus into our PeerInfo.
func (c *Collector) mapPeerStatus(ps *ipnstate.PeerStatus) PeerInfo {
	pi := PeerInfo{
		ID:             string(ps.ID),
		Hostname:       ps.HostName,
		DNSName:        ps.DNSName,
		OS:             ps.OS,
		Online:         ps.Online,
		LastSeen:       ps.LastSeen,
		ExitNode:       ps.ExitNode,
		ExitNodeOption: ps.ExitNodeOption,
		RxBytes:        ps.RxBytes,
		TxBytes:        ps.TxBytes,
	}

	// Convert TailscaleIPs from netip.Addr to string.
	if len(ps.TailscaleIPs) > 0 {
		pi.TailscaleIPs = make([]string, len(ps.TailscaleIPs))
		for i, addr := range ps.TailscaleIPs {
			pi.TailscaleIPs[i] = addr.String()
		}
	}

	// Extract tags from the views.Slice if present.
	if ps.Tags != nil && !ps.Tags.IsNil() {
		tags := make([]string, ps.Tags.Len())
		for i := range ps.Tags.Len() {
			tags[i] = ps.Tags.At(i)
		}
		pi.Tags = tags
	}

	return pi
}

// NewLocalClient creates a StatusClient backed by the real Tailscale local
// daemon. This is a convenience for production wiring; tests should inject a
// mock StatusClient instead.
func NewLocalClient(socketPath string) StatusClient {
	// Import tailscale.com/client/local at the call site to avoid pulling
	// in the full dependency graph for tests.
	lc := &localClientAdapter{socketPath: socketPath}
	return lc
}

// localClientAdapter wraps tailscale.com/client/local.Client so we can
// lazily construct it and set the Socket field.
type localClientAdapter struct {
	socketPath string
	once       sync.Once
	client     localClient
}

// localClient is a minimal interface matching the methods we use from
// tailscale.com/client/local.Client.
type localClient interface {
	Status(ctx context.Context) (*ipnstate.Status, error)
}

func (a *localClientAdapter) Status(ctx context.Context) (*ipnstate.Status, error) {
	a.once.Do(func() {
		a.client = newRealClient(a.socketPath)
	})
	return a.client.Status(ctx)
}

// peerMapKeys returns the sorted keys of a peer map for deterministic iteration.
// This is a helper for when st.Peers() is not available.
func peerMapKeys(m map[key.NodePublic]*ipnstate.PeerStatus) []key.NodePublic {
	keys := make([]key.NodePublic, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
