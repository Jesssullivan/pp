package termtest

import "strings"

// Snapshot captures the rendered output for comparison testing.
type Snapshot struct {
	Name     string // Descriptive name for the snapshot
	Terminal string // Terminal profile name used
	Width    int    // Render width in columns
	Height   int    // Render height in rows
	Content  string // The rendered string
}

// CaptureSnapshot renders content at given dimensions and stores it.
func CaptureSnapshot(name, terminal string, renderFn func(w, h int) string, width, height int) Snapshot {
	return Snapshot{
		Name:     name,
		Terminal: terminal,
		Width:    width,
		Height:   height,
		Content:  renderFn(width, height),
	}
}

// Diff describes a single line difference between two snapshots.
type Diff struct {
	Line     int    // 1-based line number where the difference occurs
	Expected string // The expected line content
	Actual   string // The actual line content
}

// CompareSnapshots checks two snapshots for differences.
// Returns nil if the snapshots are identical.
func CompareSnapshots(expected, actual Snapshot) []Diff {
	expectedLines := ttSplitLines(expected.Content)
	actualLines := ttSplitLines(actual.Content)

	maxLen := len(expectedLines)
	if len(actualLines) > maxLen {
		maxLen = len(actualLines)
	}

	var diffs []Diff
	for i := 0; i < maxLen; i++ {
		var eLine, aLine string
		if i < len(expectedLines) {
			eLine = expectedLines[i]
		}
		if i < len(actualLines) {
			aLine = actualLines[i]
		}
		if eLine != aLine {
			diffs = append(diffs, Diff{
				Line:     i + 1,
				Expected: eLine,
				Actual:   aLine,
			})
		}
	}

	return diffs
}

// ttSplitLines splits a string into lines, handling the edge case where
// an empty string should produce a single empty line for comparison.
func ttSplitLines(s string) []string {
	if s == "" {
		return []string{""}
	}
	return strings.Split(s, "\n")
}
