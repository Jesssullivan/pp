package shell

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

// Detect returns the current shell type by examining the environment and, if
// necessary, the parent process. It checks in order:
//
//  1. $SHELL environment variable
//  2. /proc/$PPID/comm on Linux
//  3. ps -p $PPID -o comm= on Darwin
//  4. Falls back to Bash as a safe default
func Detect() ShellType {
	if sh := shDetectFromEnv(); sh != "" {
		return sh
	}
	if sh := shDetectFromParent(); sh != "" {
		return sh
	}
	return Bash
}

// shDetectFromEnv checks the $SHELL environment variable and maps it to a
// ShellType.
func shDetectFromEnv() ShellType {
	shellPath := os.Getenv("SHELL")
	if shellPath == "" {
		return ""
	}
	return shParseShellName(filepath.Base(shellPath))
}

// shDetectFromParent attempts to identify the parent process's shell by
// reading /proc on Linux or using ps on Darwin.
func shDetectFromParent() ShellType {
	ppid := os.Getppid()
	if ppid <= 0 {
		return ""
	}

	switch runtime.GOOS {
	case "linux":
		return shDetectLinuxParent(ppid)
	case "darwin":
		return shDetectDarwinParent(ppid)
	default:
		return ""
	}
}

// shDetectLinuxParent reads /proc/<ppid>/comm to identify the parent shell.
func shDetectLinuxParent(ppid int) ShellType {
	data, err := os.ReadFile("/proc/" + strconv.Itoa(ppid) + "/comm")
	if err != nil {
		return ""
	}
	name := strings.TrimSpace(string(data))
	return shParseShellName(name)
}

// shDetectDarwinParent uses ps to identify the parent shell on macOS.
func shDetectDarwinParent(ppid int) ShellType {
	out, err := exec.Command("ps", "-p", strconv.Itoa(ppid), "-o", "comm=").Output()
	if err != nil {
		return ""
	}
	name := strings.TrimSpace(string(out))
	name = filepath.Base(name)
	return shParseShellName(name)
}

// shParseShellName maps a shell binary name (e.g. "zsh", "bash", "fish",
// "ksh", "ksh93") to a ShellType. Returns empty string if unrecognized.
func shParseShellName(name string) ShellType {
	// Strip leading dash for login shells (e.g., "-zsh").
	name = strings.TrimPrefix(name, "-")
	name = strings.ToLower(name)

	switch name {
	case "bash":
		return Bash
	case "zsh":
		return Zsh
	case "fish":
		return Fish
	case "ksh", "ksh93", "mksh", "pdksh":
		return Ksh
	default:
		return ""
	}
}
