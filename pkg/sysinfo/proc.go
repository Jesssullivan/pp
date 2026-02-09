package sysinfo

import (
	"sort"
	"strconv"
	"strings"
)

// siTopProcesses returns the top N processes by CPU usage.
// It delegates to the platform-specific implementation.
func siTopProcesses(n int) []ProcessInfo {
	return siTopProcessesPlatform(n)
}

// siParsePS parses the output of `ps aux` into ProcessInfo slices.
// Expected columns: USER PID %CPU %MEM ... COMMAND
// The first line (header) is skipped.
func siParsePS(output string) []ProcessInfo {
	if strings.TrimSpace(output) == "" {
		return nil
	}

	lines := strings.Split(output, "\n")
	var procs []ProcessInfo

	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Skip the header line.
		if i == 0 && strings.HasPrefix(line, "USER") {
			continue
		}

		p := siParsePSLine(line)
		if p != nil {
			procs = append(procs, *p)
		}
	}

	// Sort by CPU descending.
	sort.Slice(procs, func(i, j int) bool {
		return procs[i].CPU > procs[j].CPU
	})

	return procs
}

// siParsePSLine parses a single line of `ps aux` output.
// Format: USER PID %CPU %MEM VSZ RSS TTY STAT START TIME COMMAND
func siParsePSLine(line string) *ProcessInfo {
	fields := strings.Fields(line)
	if len(fields) < 11 {
		return nil
	}

	pid, err := strconv.Atoi(fields[1])
	if err != nil {
		return nil
	}

	cpuPct, err := strconv.ParseFloat(fields[2], 64)
	if err != nil {
		return nil
	}

	memPct, err := strconv.ParseFloat(fields[3], 64)
	if err != nil {
		return nil
	}

	// The command is everything from field 10 onward (may contain spaces).
	name := strings.Join(fields[10:], " ")

	return &ProcessInfo{
		PID:    pid,
		Name:   name,
		CPU:    cpuPct,
		Memory: memPct,
		User:   fields[0],
	}
}

// siParseProcStat parses the content of /proc/[pid]/stat on Linux.
// Format: pid (comm) state ppid pgrp session tty_nr tpgid flags
//
//	minflt cminflt majflt cmajflt utime stime cutime cstime ...
//
// We extract PID, command name, utime, and stime.
func siParseProcStat(data string) *ProcessInfo {
	data = strings.TrimSpace(data)
	if data == "" {
		return nil
	}

	// The comm field is enclosed in parentheses and may contain spaces.
	openParen := strings.IndexByte(data, '(')
	closeParen := strings.LastIndexByte(data, ')')
	if openParen < 0 || closeParen < 0 || closeParen <= openParen {
		return nil
	}

	pidStr := strings.TrimSpace(data[:openParen])
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		return nil
	}

	name := data[openParen+1 : closeParen]

	// Fields after the closing paren.
	rest := strings.TrimSpace(data[closeParen+1:])
	fields := strings.Fields(rest)
	// We need at least fields up to stime (index 11 in the rest, which is
	// field 14 overall). rest[0] = state, rest[11] = utime, rest[12] = stime
	if len(fields) < 13 {
		return nil
	}

	utime, err1 := strconv.ParseFloat(fields[11], 64)
	stime, err2 := strconv.ParseFloat(fields[12], 64)
	if err1 != nil || err2 != nil {
		return nil
	}

	return &ProcessInfo{
		PID:  pid,
		Name: name,
		CPU:  utime + stime, // raw jiffies, caller must convert to percentage
	}
}
