package banner

import (
	"strings"

	"gitlab.com/tinyland/lab/prompt-pulse/pkg/components"
)

// bnCompose places rendered widget boxes onto a 2D character grid and returns
// the result as a string. Each placement is rendered as a bordered box and
// stamped onto the grid at its (X, Y) position. Later placements overwrite
// earlier ones (z-order: last wins).
//
// The returned string has exactly `height` lines, each with exactly `width`
// visible characters. ANSI escape sequences are handled correctly: visible
// length is measured with components.VisibleLen and lines are truncated with
// components.Truncate.
func bnCompose(placements []bnPlacement, width, height int) string {
	if width <= 0 || height <= 0 {
		return ""
	}

	// Initialize grid with spaces.
	grid := make([][]byte, height)
	for i := range grid {
		grid[i] = []byte(strings.Repeat(" ", width))
	}

	// Stamp each widget onto the grid.
	for _, p := range placements {
		rendered := bnRenderWidgetBox(p.Widget, p.W, p.H)
		bnStampOnGrid(grid, rendered, p.X, p.Y, p.W, p.H, width)
	}

	// Join grid lines.
	lines := make([]string, height)
	for i, row := range grid {
		lines[i] = string(row)
	}
	return strings.Join(lines, "\n")
}

// bnStampOnGrid writes the rendered box content onto the grid at position
// (x, y). Each line of the rendered content replaces the corresponding
// segment of the grid row. Lines are truncated or padded to fit within
// the allocated width, and clipped to the grid boundaries.
func bnStampOnGrid(grid [][]byte, rendered string, x, y, w, h, gridWidth int) {
	if rendered == "" {
		return
	}

	lines := strings.Split(rendered, "\n")

	for i, line := range lines {
		row := y + i
		if row < 0 || row >= len(grid) {
			continue
		}
		if x >= gridWidth {
			continue
		}

		// Fit the line to the allocated width.
		fitted := bnFitToWidth(line, w)

		// Clip to grid right edge.
		availW := gridWidth - x
		if availW <= 0 {
			continue
		}

		clipped := fitted
		visLen := components.VisibleLen(clipped)
		if visLen > availW {
			clipped = components.Truncate(clipped, availW)
			visLen = components.VisibleLen(clipped)
		}

		// Build new row: prefix + clipped + suffix.
		prefix := ""
		if x > 0 {
			prefix = string(grid[row][:x])
		}

		suffixStart := x + visLen
		suffix := ""
		if suffixStart < gridWidth {
			suffix = string(grid[row][suffixStart:])
		}

		// Since our grid is plain ASCII spaces (no ANSI in the grid itself
		// except what we stamp), we can safely concatenate.
		newRow := prefix + clipped + suffix

		// Ensure the row is exactly gridWidth visible characters.
		newRowVis := components.VisibleLen(newRow)
		if newRowVis < gridWidth {
			newRow = newRow + strings.Repeat(" ", gridWidth-newRowVis)
		} else if newRowVis > gridWidth {
			newRow = components.Truncate(newRow, gridWidth)
		}

		grid[row] = []byte(newRow)
	}
}

// bnFitToWidth truncates or pads a line to exactly w visible characters.
func bnFitToWidth(line string, w int) string {
	if w <= 0 {
		return ""
	}
	vis := components.VisibleLen(line)
	if vis > w {
		return components.Truncate(line, w)
	}
	if vis < w {
		return components.PadRight(line, w)
	}
	return line
}
