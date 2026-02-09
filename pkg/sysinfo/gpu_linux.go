//go:build linux

package sysinfo

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// siDetectGPUsPlatform detects GPUs on Linux. It first tries nvidia-smi for
// NVIDIA GPUs, then falls back to scanning /sys/class/drm for other vendors.
func siDetectGPUsPlatform() []GPUInfo {
	// Try NVIDIA SMI first.
	if gpus := siTryNvidiaSMI(); len(gpus) > 0 {
		return gpus
	}

	// Fall back to sysfs DRM scan.
	return siScanDRM()
}

// siTryNvidiaSMI attempts to detect NVIDIA GPUs via nvidia-smi.
func siTryNvidiaSMI() []GPUInfo {
	out, err := exec.Command(
		"nvidia-smi",
		"--query-gpu=name,driver_version,memory.total,temperature.gpu,utilization.gpu",
		"--format=csv,noheader,nounits",
	).Output()
	if err != nil {
		return nil
	}
	return siParseNvidiaSMI(string(out))
}

// siScanDRM scans /sys/class/drm/card*/device/vendor for GPU vendor IDs.
func siScanDRM() []GPUInfo {
	cards, err := filepath.Glob("/sys/class/drm/card[0-9]*/device/vendor")
	if err != nil || len(cards) == 0 {
		return nil
	}

	var gpus []GPUInfo
	seen := make(map[string]bool)
	for _, vendorPath := range cards {
		cardDir := filepath.Dir(vendorPath)
		if seen[cardDir] {
			continue
		}
		seen[cardDir] = true

		vendorBytes, err := os.ReadFile(vendorPath)
		if err != nil {
			continue
		}
		vendorID := strings.TrimSpace(string(vendorBytes))

		gpu := GPUInfo{
			Vendor: siVendorIDToName(vendorID),
		}

		// Try to get device name from device file.
		devicePath := filepath.Join(cardDir, "device")
		if data, err := os.ReadFile(devicePath); err == nil {
			gpu.Name = strings.TrimSpace(string(data))
		}

		if gpu.Vendor != "" {
			gpus = append(gpus, gpu)
		}
	}
	return gpus
}

// siVendorIDToName maps PCI vendor IDs to vendor names.
func siVendorIDToName(id string) string {
	switch strings.ToLower(id) {
	case "0x10de":
		return "nvidia"
	case "0x1002":
		return "amd"
	case "0x8086":
		return "intel"
	default:
		return ""
	}
}
