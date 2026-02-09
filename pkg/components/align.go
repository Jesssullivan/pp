// Package components provides canonical box/border rendering and ANSI-aware
// text primitives for the prompt-pulse TUI. It consolidates the 20+
// duplicated string manipulation functions from v1 into one correct,
// tested implementation.
package components

// Align controls horizontal text alignment within a box or cell.
type Align int

const (
	// AlignLeft aligns text to the left edge (default).
	AlignLeft Align = iota
	// AlignCenter centers text horizontally.
	AlignCenter
	// AlignRight aligns text to the right edge.
	AlignRight
)

// Padding defines spacing on each side of a content area.
type Padding struct {
	Top    int
	Right  int
	Bottom int
	Left   int
}

// NewPadding creates a Padding with the same value on all four sides.
func NewPadding(all int) Padding {
	if all < 0 {
		all = 0
	}
	return Padding{Top: all, Right: all, Bottom: all, Left: all}
}

// NewPaddingHV creates a Padding with separate horizontal and vertical values.
// horiz applies to Left and Right; vert applies to Top and Bottom.
func NewPaddingHV(horiz, vert int) Padding {
	if horiz < 0 {
		horiz = 0
	}
	if vert < 0 {
		vert = 0
	}
	return Padding{Top: vert, Right: horiz, Bottom: vert, Left: horiz}
}
