// Package sysinfo provides cross-platform system information that goes beyond
// basic sysmetrics. It includes GPU detection, container environment detection,
// process monitoring, and network interface classification.
package sysinfo

import (
	"os"
	"runtime"
)

// SystemInfo holds comprehensive system information.
type SystemInfo struct {
	Hostname      string
	OS            string // "darwin", "linux"
	Arch          string // "arm64", "amd64"
	Kernel        string // kernel version string
	InContainer   bool   // running inside a container
	ContainerType string // "docker", "podman", "lxc", ""
	GPUs          []GPUInfo
	NICs          []NICInfo
}

// GPUInfo describes a single GPU device.
type GPUInfo struct {
	Name        string
	Vendor      string  // "nvidia", "amd", "intel", "apple"
	VRAM        uint64  // bytes, 0 if unknown
	Driver      string
	Temperature float64 // celsius, 0 if unknown
	Utilization float64 // percent, 0 if unknown
}

// NICInfo describes a single network interface.
type NICInfo struct {
	Name  string
	Type  string // "ethernet", "wifi", "tailscale", "loopback", "virtual"
	MAC   string
	IPv4  string
	IPv6  string
	Up    bool
	Speed string // "1Gbps", "10Gbps", etc.
}

// ProcessInfo describes a running process.
type ProcessInfo struct {
	PID    int
	Name   string
	CPU    float64
	Memory float64
	User   string
}

// Collect gathers all system information. Individual subsystems that fail
// are silently skipped so that the caller always gets as much data as
// possible.
func Collect() (*SystemInfo, error) {
	hostname, _ := os.Hostname()

	inContainer, containerType := siDetectContainer()

	info := &SystemInfo{
		Hostname:      hostname,
		OS:            runtime.GOOS,
		Arch:          runtime.GOARCH,
		Kernel:        siKernelVersion(),
		InContainer:   inContainer,
		ContainerType: containerType,
		GPUs:          siDetectGPUs(),
		NICs:          siDetectNICs(),
	}

	return info, nil
}
