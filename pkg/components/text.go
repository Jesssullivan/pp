package components

import (
	"strings"

	"github.com/charmbracelet/x/ansi"
)

// VisibleLen returns the visible character width of s in terminal cells.
// ANSI escape sequences are ignored. Wide characters (CJK, emoji) are
// counted as width 2. Zero-width joiners, combining marks, and other
// zero-width characters are handled correctly via grapheme clustering.
func VisibleLen(s string) int {
	return ansi.StringWidth(s)
}

// Truncate truncates s to at most maxWidth visible characters, preserving
// any ANSI escape sequences that appear before the cut point. If s is
// already within maxWidth, it is returned unchanged.
func Truncate(s string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}
	return ansi.Truncate(s, maxWidth, "")
}

// TruncateWithTail truncates s to at most maxWidth visible characters,
// appending tail (e.g. "...") if truncation occurs. The tail itself counts
// toward maxWidth, so the visible content will be (maxWidth - len(tail))
// characters followed by tail.
func TruncateWithTail(s string, maxWidth int, tail string) string {
	if maxWidth <= 0 {
		return ""
	}
	return ansi.Truncate(s, maxWidth, tail)
}

// PadRight pads s with trailing spaces so that its visible width equals
// width. If s is already wider than width, it is returned unchanged.
func PadRight(s string, width int) string {
	vis := VisibleLen(s)
	if vis >= width {
		return s
	}
	return s + strings.Repeat(" ", width-vis)
}

// PadLeft pads s with leading spaces so that its visible width equals
// width. If s is already wider than width, it is returned unchanged.
func PadLeft(s string, width int) string {
	vis := VisibleLen(s)
	if vis >= width {
		return s
	}
	return strings.Repeat(" ", width-vis) + s
}

// PadCenter pads s with spaces on both sides so that it is centered
// within width. If the padding is odd, the extra space goes on the right.
// If s is already wider than width, it is returned unchanged.
func PadCenter(s string, width int) string {
	vis := VisibleLen(s)
	if vis >= width {
		return s
	}
	total := width - vis
	left := total / 2
	right := total - left
	return strings.Repeat(" ", left) + s + strings.Repeat(" ", right)
}

// Wrap performs word-wrapping on s at the given width, respecting ANSI
// escape sequences and wide characters. Lines are broken at spaces and
// hyphens. Returns a slice of wrapped lines (without trailing newlines).
func Wrap(s string, width int) []string {
	if width <= 0 {
		return []string{s}
	}
	wrapped := ansi.Wrap(s, width, "")
	return strings.Split(wrapped, "\n")
}
