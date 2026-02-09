//go:build darwin

package sysinfo

import (
	"os/exec"
)

// siTopProcessesPlatform returns top N processes on macOS using ps.
func siTopProcessesPlatform(n int) []ProcessInfo {
	out, err := exec.Command("ps", "aux").Output()
	if err != nil {
		return nil
	}
	procs := siParsePS(string(out))
	if n > 0 && len(procs) > n {
		procs = procs[:n]
	}
	return procs
}
