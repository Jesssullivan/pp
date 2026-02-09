package tui

import (
	"strings"

	"gitlab.com/tinyland/lab/prompt-pulse/pkg/app"
	"gitlab.com/tinyland/lab/prompt-pulse/pkg/components"
)

// tuiRenderGrid renders all widget cells into a single string that
// represents the grid layout. Each widget is wrapped in a bordered box
// with a title. The focused widget gets a highlighted border.
func tuiRenderGrid(cells []tuiCell, width, height int) string {
	if len(cells) == 0 || width <= 0 || height <= 0 {
		return ""
	}

	// Build a 2D character buffer for compositing.
	buf := tuiNewBuffer(width, height)

	for _, cell := range cells {
		borderColor := "#6B7280" // dim gray
		if cell.Focused {
			borderColor = "#7C3AED" // purple accent
		}

		// Inner dimensions after removing the border (2 chars per axis).
		innerW := cell.W - 2
		innerH := cell.H - 2
		if innerW < 1 {
			innerW = 1
		}
		if innerH < 1 {
			innerH = 1
		}

		content := cell.Widget.View(innerW, innerH)

		style := components.BoxStyle{
			Border:     components.BorderRounded,
			Title:      cell.Widget.Title(),
			TitleAlign: components.AlignLeft,
			FG:         borderColor,
		}

		box := components.RenderBox(content, cell.W, cell.H, style)
		tuiBlitToBuffer(buf, box, cell.X, cell.Y, width, height)
	}

	return tuiBufferToString(buf)
}

// tuiRenderExpanded renders a single widget at full size, wrapped in a
// bordered box with the widget's title.
func tuiRenderExpanded(widget app.Widget, width, height int) string {
	if width <= 0 || height <= 0 {
		return ""
	}

	innerW := width - 2
	innerH := height - 2
	if innerW < 1 {
		innerW = 1
	}
	if innerH < 1 {
		innerH = 1
	}

	content := widget.View(innerW, innerH)

	style := components.BoxStyle{
		Border:     components.BorderRounded,
		Title:      widget.Title(),
		TitleAlign: components.AlignLeft,
		FG:         "#7C3AED", // always accent colored when expanded
	}

	return components.RenderBox(content, width, height, style)
}

// tuiRenderStatusBar renders a one-line status bar at the bottom of the
// terminal with key hints. It pads or truncates to exactly width characters.
func tuiRenderStatusBar(msg string, width int) string {
	hints := "Tab:focus  Enter:expand  ?:help  /:search  q:quit"
	if msg != "" {
		hints = msg + "  |  " + hints
	}

	if width <= 0 {
		return ""
	}

	return components.Dim(components.PadRight(components.Truncate(hints, width), width))
}

// tuiNewBuffer creates a 2D grid of spaces with the given dimensions.
func tuiNewBuffer(width, height int) [][]rune {
	buf := make([][]rune, height)
	for y := 0; y < height; y++ {
		row := make([]rune, width)
		for x := 0; x < width; x++ {
			row[x] = ' '
		}
		buf[y] = row
	}
	return buf
}

// tuiBlitToBuffer writes a rendered multi-line string into the character
// buffer at position (x, y), clipping to the buffer boundaries.
func tuiBlitToBuffer(buf [][]rune, rendered string, x, y, bufW, bufH int) {
	lines := strings.Split(rendered, "\n")
	for dy, line := range lines {
		ry := y + dy
		if ry < 0 || ry >= bufH {
			continue
		}
		runes := []rune(line)
		for dx, ch := range runes {
			rx := x + dx
			if rx < 0 || rx >= bufW {
				continue
			}
			buf[ry][rx] = ch
		}
	}
}

// tuiBufferToString converts the 2D character buffer to a single string
// with newline separators between rows.
func tuiBufferToString(buf [][]rune) string {
	lines := make([]string, len(buf))
	for i, row := range buf {
		lines[i] = string(row)
	}
	return strings.Join(lines, "\n")
}
