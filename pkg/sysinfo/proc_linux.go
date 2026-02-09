//go:build linux

package sysinfo

import (
	"os/exec"
)

// siTopProcessesPlatform returns top N processes on Linux using ps.
// A more efficient approach would read /proc directly, but ps is simpler
// and the parsing logic is shared cross-platform.
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
