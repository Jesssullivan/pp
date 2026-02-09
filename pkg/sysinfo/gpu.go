package sysinfo

import (
	"encoding/json"
	"strings"
)

// siDetectGPUs attempts to detect available GPUs. Platform-specific
// detection is delegated to siDetectGPUsPlatform which is defined in
// gpu_darwin.go and gpu_linux.go.
func siDetectGPUs() []GPUInfo {
	return siDetectGPUsPlatform()
}

// siParseNvidiaSMI parses the CSV output of:
//
//	nvidia-smi --query-gpu=name,driver_version,memory.total,temperature.gpu,utilization.gpu --format=csv,noheader,nounits
//
// Each line has the format: "name, driver, vram_mib, temp_c, util_pct"
func siParseNvidiaSMI(output string) []GPUInfo {
	if strings.TrimSpace(output) == "" {
		return nil
	}

	var gpus []GPUInfo
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		gpu := siParseNvidiaSMILine(line)
		if gpu.Name != "" {
			gpus = append(gpus, gpu)
		}
	}
	return gpus
}

// siParseNvidiaSMILine parses a single CSV line from nvidia-smi output.
func siParseNvidiaSMILine(line string) GPUInfo {
	parts := strings.Split(line, ", ")
	if len(parts) < 2 {
		return GPUInfo{}
	}

	gpu := GPUInfo{
		Name:   strings.TrimSpace(parts[0]),
		Vendor: "nvidia",
	}

	if len(parts) > 1 {
		gpu.Driver = strings.TrimSpace(parts[1])
	}

	if len(parts) > 2 {
		gpu.VRAM = siParseMiBToBytes(strings.TrimSpace(parts[2]))
	}

	if len(parts) > 3 {
		gpu.Temperature = siParseFloat(strings.TrimSpace(parts[3]))
	}

	if len(parts) > 4 {
		gpu.Utilization = siParseFloat(strings.TrimSpace(parts[4]))
	}

	return gpu
}

// siParseMiBToBytes converts a MiB string to bytes. Returns 0 on error.
func siParseMiBToBytes(s string) uint64 {
	val := siParseFloat(s)
	if val <= 0 {
		return 0
	}
	return uint64(val * 1024 * 1024)
}

// darwinSPDisplaysDataType is the JSON structure returned by
// system_profiler SPDisplaysDataType -json on macOS.
type darwinSPDisplaysDataType struct {
	SPDisplaysDataType []darwinGPUEntry `json:"SPDisplaysDataType"`
}

type darwinGPUEntry struct {
	Name          string `json:"sppci_model"`
	Vendor        string `json:"sppci_vendor"`
	VRAM          string `json:"sppci_vram"`           // e.g. "8 GB" or "8192 MB"
	VRAMShared    string `json:"sppci_vram_shared"`    // for integrated GPUs
	ChipsetModel  string `json:"spdisplays_mtlgpufamilysupport"`
	DeviceID      string `json:"sppci_device_id"`
	Bus           string `json:"sppci_bus"`
	MetalFamily   string `json:"spdisplays_metalfamily"` // "Metal 3", etc.
	MetalSupport  string `json:"spdisplays_metal"`
	Cores         string `json:"sppci_cores"`             // Apple Silicon GPU cores
}

// siParseDarwinGPU parses the JSON output of system_profiler SPDisplaysDataType -json.
func siParseDarwinGPU(jsonData []byte) []GPUInfo {
	if len(jsonData) == 0 {
		return nil
	}

	var data darwinSPDisplaysDataType
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil
	}

	var gpus []GPUInfo
	for _, entry := range data.SPDisplaysDataType {
		gpu := GPUInfo{
			Name: entry.Name,
		}

		// Determine vendor from the vendor string.
		gpu.Vendor = siNormalizeGPUVendor(entry.Vendor)

		// Parse VRAM. Apple Silicon reports "sppci_vram_shared" instead.
		vramStr := entry.VRAM
		if vramStr == "" {
			vramStr = entry.VRAMShared
		}
		gpu.VRAM = siParseVRAMString(vramStr)

		// Metal family can serve as a rough driver identifier on macOS.
		gpu.Driver = entry.MetalFamily

		gpus = append(gpus, gpu)
	}
	return gpus
}

// siNormalizeGPUVendor maps a verbose vendor string to a short canonical form.
// Check order matters: "Intel Corporation" contains "ati", so Intel must be
// checked before AMD/ATI.
func siNormalizeGPUVendor(vendor string) string {
	lower := strings.ToLower(vendor)
	switch {
	case strings.Contains(lower, "nvidia"):
		return "nvidia"
	case strings.Contains(lower, "intel"):
		return "intel"
	case strings.Contains(lower, "apple"):
		return "apple"
	case strings.Contains(lower, "amd") || strings.Contains(lower, "ati"):
		return "amd"
	default:
		return strings.ToLower(strings.TrimSpace(vendor))
	}
}

// siParseVRAMString converts human-readable VRAM strings like "8 GB", "8192 MB"
// to bytes.
func siParseVRAMString(s string) uint64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	lower := strings.ToLower(s)

	// Try "N GB" pattern.
	if strings.HasSuffix(lower, " gb") {
		numStr := strings.TrimSuffix(lower, " gb")
		val := siParseFloat(strings.TrimSpace(numStr))
		if val > 0 {
			return uint64(val * 1024 * 1024 * 1024)
		}
	}

	// Try "N MB" pattern.
	if strings.HasSuffix(lower, " mb") {
		numStr := strings.TrimSuffix(lower, " mb")
		val := siParseFloat(strings.TrimSpace(numStr))
		if val > 0 {
			return uint64(val * 1024 * 1024)
		}
	}

	return 0
}
