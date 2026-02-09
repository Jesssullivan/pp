//go:build darwin

package sysinfo

import (
	"os/exec"
)

// siDetectGPUsPlatform detects GPUs on macOS using system_profiler.
func siDetectGPUsPlatform() []GPUInfo {
	out, err := exec.Command("system_profiler", "SPDisplaysDataType", "-json").Output()
	if err != nil {
		return nil
	}
	return siParseDarwinGPU(out)
}
