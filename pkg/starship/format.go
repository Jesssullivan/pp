package starship

import (
	"strings"
	"unicode/utf8"
)

// ssAnsiReset is the ANSI escape sequence to reset all text attributes.
const ssAnsiReset = "\033[0m"

// ssSeparator is the dim separator character placed between segments.
const ssSeparator = "\033[2m│\033[0m"

// ssColorize wraps text in the given ANSI color code and appends a reset
// sequence. If color is empty, text is returned unmodified.
func ssColorize(text, color string) string {
	if color == "" {
		return text
	}
	return color + text + ssAnsiReset
}

// ssVisibleWidth returns the number of visible (non-ANSI-escape) runes in s.
// It strips CSI sequences (ESC [ ... final byte) before counting.
func ssVisibleWidth(s string) int {
	stripped := ssStripAnsi(s)
	return utf8.RuneCountInString(stripped)
}

// ssStripAnsi removes ANSI CSI escape sequences (ESC [ ... final) from s.
func ssStripAnsi(s string) string {
	var b strings.Builder
	b.Grow(len(s))

	const (
		stateNormal = iota
		stateEsc    // saw ESC, expecting '['
		stateCSI    // inside CSI params, expecting final byte
	)

	state := stateNormal
	for _, r := range s {
		switch state {
		case stateNormal:
			if r == '\033' {
				state = stateEsc
			} else {
				b.WriteRune(r)
			}
		case stateEsc:
			if r == '[' {
				state = stateCSI
			} else {
				// Not a CSI sequence; drop the ESC and emit the char.
				b.WriteRune(r)
				state = stateNormal
			}
		case stateCSI:
			// CSI parameter bytes are 0x30-0x3F, intermediate 0x20-0x2F.
			// Final byte is 0x40-0x7E, which terminates the sequence.
			if r >= 0x40 && r <= 0x7E {
				state = stateNormal
			}
			// All bytes in CSI (params + final) are consumed without output.
		}
	}
	return b.String()
}

// ssFormatLine joins the given segments with a dim separator, applies ANSI
// colors, and drops rightmost segments if the total visible width exceeds
// maxWidth. Returns an empty string if segments is empty.
func ssFormatLine(segments []*Segment, maxWidth int) string {
	if len(segments) == 0 {
		return ""
	}

	if maxWidth <= 0 {
		maxWidth = 60
	}

	// Build each segment's rendered form and record its visible width.
	type rendered struct {
		text         string
		visibleWidth int
	}

	parts := make([]rendered, 0, len(segments))
	for _, seg := range segments {
		full := seg.Icon + " " + seg.Text
		colored := ssColorize(full, seg.Color)
		parts = append(parts, rendered{
			text:         colored,
			visibleWidth: ssVisibleWidth(colored),
		})
	}

	// Separator visible width is 1 character (the pipe).
	const sepWidth = 1

	// Greedily include segments left-to-right until maxWidth is exceeded.
	var included []rendered
	totalVisible := 0
	for i, p := range parts {
		needed := p.visibleWidth
		if i > 0 {
			needed += sepWidth + 2 // separator + surrounding spaces
		}
		if totalVisible+needed > maxWidth {
			break
		}
		included = append(included, p)
		totalVisible += needed
	}

	if len(included) == 0 {
		return ""
	}

	var b strings.Builder
	for i, p := range included {
		if i > 0 {
			b.WriteString(" " + ssSeparator + " ")
		}
		b.WriteString(p.text)
	}
	return b.String()
}
