package sysinfo

import (
	"net"
	"runtime"
	"strings"
)

// siDetectNICs lists all network interfaces and classifies them by type.
func siDetectNICs() []NICInfo {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil
	}

	tailscaleIface := siGetTailscaleInterface()

	var nics []NICInfo
	for _, iface := range ifaces {
		nic := NICInfo{
			Name: iface.Name,
			MAC:  iface.HardwareAddr.String(),
			Up:   iface.Flags&net.FlagUp != 0,
			Type: siClassifyNIC(iface.Name, runtime.GOOS),
		}

		// Override type if this is the tailscale interface.
		if tailscaleIface != "" && iface.Name == tailscaleIface {
			nic.Type = "tailscale"
		}

		// Get IP addresses.
		addrs, err := iface.Addrs()
		if err == nil {
			for _, addr := range addrs {
				ip := siExtractIP(addr.String())
				if ip == "" {
					continue
				}
				if strings.Contains(ip, ":") {
					if nic.IPv6 == "" {
						nic.IPv6 = ip
					}
				} else {
					if nic.IPv4 == "" {
						nic.IPv4 = ip
					}
				}
			}
		}

		nics = append(nics, nic)
	}

	return nics
}

// siClassifyNIC classifies a network interface by its name and the current OS.
// The goos parameter allows testing without runtime dependency.
func siClassifyNIC(name, goos string) string {
	lower := strings.ToLower(name)

	// Loopback.
	if strings.HasPrefix(lower, "lo") {
		return "loopback"
	}

	// Tailscale explicit interface.
	if strings.HasPrefix(lower, "tailscale") {
		return "tailscale"
	}

	// Virtual/container interfaces.
	if strings.HasPrefix(lower, "veth") ||
		strings.HasPrefix(lower, "br-") ||
		strings.HasPrefix(lower, "docker") ||
		strings.HasPrefix(lower, "cni") ||
		strings.HasPrefix(lower, "flannel") ||
		strings.HasPrefix(lower, "vxlan") ||
		strings.HasPrefix(lower, "virbr") {
		return "virtual"
	}

	// Platform-specific classification.
	switch goos {
	case "darwin":
		return siClassifyNICDarwin(lower)
	case "linux":
		return siClassifyNICLinux(lower)
	}

	// Fallback generic.
	if strings.HasPrefix(lower, "eth") {
		return "ethernet"
	}
	if strings.HasPrefix(lower, "wl") || strings.HasPrefix(lower, "wlan") {
		return "wifi"
	}

	return "virtual"
}

// siClassifyNICDarwin classifies macOS interface names.
func siClassifyNICDarwin(lower string) string {
	switch {
	case strings.HasPrefix(lower, "en"):
		return "ethernet"
	case strings.HasPrefix(lower, "awdl") || strings.HasPrefix(lower, "llw"):
		return "wifi"
	case strings.HasPrefix(lower, "utun"):
		// utun interfaces are used by VPNs and Tailscale on macOS.
		return "virtual"
	case strings.HasPrefix(lower, "bridge"):
		return "virtual"
	case strings.HasPrefix(lower, "ap"):
		return "wifi"
	default:
		return "virtual"
	}
}

// siClassifyNICLinux classifies Linux interface names.
func siClassifyNICLinux(lower string) string {
	switch {
	case strings.HasPrefix(lower, "eth"):
		return "ethernet"
	case strings.HasPrefix(lower, "enp") || strings.HasPrefix(lower, "eno") || strings.HasPrefix(lower, "ens"):
		return "ethernet"
	case strings.HasPrefix(lower, "wl") || strings.HasPrefix(lower, "wlan"):
		return "wifi"
	case strings.HasPrefix(lower, "ww"):
		return "wifi" // WWAN
	default:
		return "virtual"
	}
}

// siGetTailscaleInterface returns the name of the tailscale interface if
// tailscale is running, or empty string otherwise. On macOS, tailscale uses
// a utun interface; on Linux it uses tailscale0.
func siGetTailscaleInterface() string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return ""
	}
	for _, iface := range ifaces {
		if strings.HasPrefix(iface.Name, "tailscale") {
			return iface.Name
		}
	}
	// On macOS, tailscale uses utun interfaces. We cannot reliably determine
	// which utun belongs to tailscale without querying the tailscale daemon,
	// so we do not guess.
	return ""
}

// siExtractIP strips the CIDR mask from an address string like "192.168.1.1/24".
func siExtractIP(addr string) string {
	if idx := strings.IndexByte(addr, '/'); idx >= 0 {
		return addr[:idx]
	}
	return addr
}
