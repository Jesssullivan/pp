package components

import (
	"fmt"
	"strconv"
	"strings"
)

// Color produces an ANSI true-color (24-bit) foreground escape sequence from
// a hex color string like "#ff5500" or "ff5500". Returns an empty string if
// the input is empty or malformed.
func Color(hex string) string {
	r, g, b, ok := parseHex(hex)
	if !ok {
		return ""
	}
	return fmt.Sprintf("\x1b[38;2;%d;%d;%dm", r, g, b)
}

// BgColor produces an ANSI true-color (24-bit) background escape sequence
// from a hex color string like "#ff5500" or "ff5500".
func BgColor(hex string) string {
	r, g, b, ok := parseHex(hex)
	if !ok {
		return ""
	}
	return fmt.Sprintf("\x1b[48;2;%d;%d;%dm", r, g, b)
}

// Bold wraps s in ANSI bold escape sequences.
func Bold(s string) string {
	return "\x1b[1m" + s + "\x1b[22m"
}

// Dim wraps s in ANSI dim/faint escape sequences.
func Dim(s string) string {
	return "\x1b[2m" + s + "\x1b[22m"
}

// Italic wraps s in ANSI italic escape sequences.
func Italic(s string) string {
	return "\x1b[3m" + s + "\x1b[23m"
}

// Reset returns the ANSI reset sequence that clears all styling.
func Reset() string {
	return "\x1b[0m"
}

// parseHex parses a hex color string into r, g, b components.
// Accepts "#RRGGBB" or "RRGGBB" formats.
func parseHex(hex string) (r, g, b uint8, ok bool) {
	hex = strings.TrimPrefix(hex, "#")
	if len(hex) != 6 {
		return 0, 0, 0, false
	}
	rv, err := strconv.ParseUint(hex[0:2], 16, 8)
	if err != nil {
		return 0, 0, 0, false
	}
	gv, err := strconv.ParseUint(hex[2:4], 16, 8)
	if err != nil {
		return 0, 0, 0, false
	}
	bv, err := strconv.ParseUint(hex[4:6], 16, 8)
	if err != nil {
		return 0, 0, 0, false
	}
	return uint8(rv), uint8(gv), uint8(bv), true
}
