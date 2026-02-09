// Package terminal provides terminal emulator detection, graphics protocol
// selection, and terminal size queries. It consolidates the detection logic
// previously duplicated across display/render and waifu packages into a
// single source of truth.
//
// Detection is split into two layers:
//   - Layer 1 (Detect): Environment variable inspection, 0ms, no I/O.
//   - Layer 2 (future): Terminal query sequences for definitive detection.
package terminal

import (
	"os"
	"strings"
)

// Terminal identifies the terminal emulator in use.
type Terminal int

const (
	TermUnknown   Terminal = iota
	TermGhostty            // Ghostty (kitty graphics, true color, OSC 8)
	TermKitty              // Kitty (kitty graphics, keyboard protocol)
	TermWezTerm            // WezTerm (kitty graphics, sixel, iterm2 images)
	TermITerm2             // iTerm2 (iterm2 images, true color, OSC 8)
	TermAlacritty          // Alacritty (true color, no graphics protocol)
	TermTilix              // Tilix (VTE-based, true color)
	TermGNOME              // GNOME Terminal (VTE-based, true color)
	TermTmux               // tmux multiplexer
	TermScreen             // GNU Screen multiplexer
	TermVSCode             // VS Code integrated terminal
	TermEmacs              // Emacs vterm/eat
	TermGeneric            // Unknown terminal with basic capabilities
)

// terminalNames maps Terminal values to human-readable strings.
var terminalNames = [...]string{
	TermUnknown:   "unknown",
	TermGhostty:   "ghostty",
	TermKitty:     "kitty",
	TermWezTerm:   "wezterm",
	TermITerm2:    "iterm2",
	TermAlacritty: "alacritty",
	TermTilix:     "tilix",
	TermGNOME:     "gnome-terminal",
	TermTmux:      "tmux",
	TermScreen:    "screen",
	TermVSCode:    "vscode",
	TermEmacs:     "emacs",
	TermGeneric:   "generic",
}

// String returns the human-readable name of the terminal.
func (t Terminal) String() string {
	if int(t) < len(terminalNames) {
		return terminalNames[t]
	}
	return "unknown"
}

// SupportsKittyGraphics reports whether the terminal supports the Kitty
// graphics protocol for inline image rendering.
func (t Terminal) SupportsKittyGraphics() bool {
	switch t {
	case TermGhostty, TermKitty, TermWezTerm:
		return true
	default:
		return false
	}
}

// SupportsSixel reports whether the terminal supports the Sixel graphics
// protocol. WezTerm has native sixel support; most other modern terminals
// do not.
func (t Terminal) SupportsSixel() bool {
	switch t {
	case TermWezTerm:
		return true
	default:
		return false
	}
}

// SupportsITerm2Images reports whether the terminal supports the iTerm2
// inline images protocol.
func (t Terminal) SupportsITerm2Images() bool {
	switch t {
	case TermITerm2, TermWezTerm:
		return true
	default:
		return false
	}
}

// SupportsTrueColor reports whether the terminal supports 24-bit true color.
func (t Terminal) SupportsTrueColor() bool {
	switch t {
	case TermGhostty, TermKitty, TermWezTerm, TermITerm2,
		TermAlacritty, TermTilix, TermGNOME, TermVSCode:
		return true
	default:
		return false
	}
}

// SupportsOSC8Hyperlinks reports whether the terminal supports OSC 8
// clickable hyperlinks.
func (t Terminal) SupportsOSC8Hyperlinks() bool {
	switch t {
	case TermGhostty, TermKitty, TermWezTerm, TermITerm2,
		TermAlacritty, TermTilix, TermGNOME, TermVSCode:
		return true
	default:
		return false
	}
}

// SupportsMouseSGR reports whether the terminal supports SGR mouse
// encoding (1006 mode) for precise mouse tracking.
func (t Terminal) SupportsMouseSGR() bool {
	switch t {
	case TermGhostty, TermKitty, TermWezTerm, TermITerm2,
		TermAlacritty, TermTilix, TermGNOME:
		return true
	default:
		return false
	}
}

// SupportsKittyKeyboard reports whether the terminal supports the Kitty
// progressive keyboard enhancement protocol.
func (t Terminal) SupportsKittyKeyboard() bool {
	switch t {
	case TermGhostty, TermKitty, TermWezTerm:
		return true
	default:
		return false
	}
}

// SupportsSyncOutput reports whether the terminal supports synchronized
// output mode (DEC mode 2026) to eliminate flicker during redraws.
func (t Terminal) SupportsSyncOutput() bool {
	switch t {
	case TermGhostty, TermKitty, TermWezTerm, TermITerm2,
		TermAlacritty, TermTilix, TermGNOME:
		return true
	default:
		return false
	}
}

// Detect identifies the terminal emulator from environment variables.
// This is Layer 1 detection (0ms, no I/O). Detection proceeds through
// multiple signals ordered by reliability:
//
//  1. TERM_PROGRAM env var (most terminals set this)
//  2. TERM env var (xterm-ghostty, xterm-kitty, alacritty)
//  3. Terminal-specific vars (KITTY_WINDOW_ID, ITERM_SESSION_ID, etc.)
//  4. VTE_VERSION for VTE-based terminals (GNOME, Tilix)
//  5. INSIDE_EMACS for emacs terminals
//  6. TMUX / STY for multiplexers
//  7. COLORTERM=truecolor as generic true color indicator
//  8. Fallback to TermGeneric
func Detect() Terminal {
	// Layer 1: TERM_PROGRAM is the most reliable single signal.
	if tp := os.Getenv("TERM_PROGRAM"); tp != "" {
		switch strings.ToLower(tp) {
		case "ghostty":
			return TermGhostty
		case "kitty":
			return TermKitty
		case "wezterm":
			return TermWezTerm
		case "iterm.app":
			return TermITerm2
		case "vscode":
			return TermVSCode
		case "alacritty":
			return TermAlacritty
		case "tmux":
			return TermTmux
		}
	}

	// Layer 2: TERM value encodes terminal identity in some emulators.
	if term := os.Getenv("TERM"); term != "" {
		switch {
		case term == "xterm-ghostty":
			return TermGhostty
		case term == "xterm-kitty":
			return TermKitty
		case strings.HasPrefix(term, "alacritty"):
			return TermAlacritty
		case strings.HasPrefix(term, "screen"):
			// GNU Screen sets TERM=screen or screen-256color.
			// Check STY to confirm it is actually screen.
			if os.Getenv("STY") != "" {
				return TermScreen
			}
		}
	}

	// Layer 3: Terminal-specific environment variables.
	if os.Getenv("KITTY_WINDOW_ID") != "" {
		return TermKitty
	}
	if os.Getenv("ITERM_SESSION_ID") != "" {
		return TermITerm2
	}
	if os.Getenv("WEZTERM_EXECUTABLE") != "" {
		return TermWezTerm
	}

	// Layer 4: VTE-based terminals (GNOME Terminal, Tilix, etc.)
	if os.Getenv("VTE_VERSION") != "" {
		if os.Getenv("TILIX_ID") != "" {
			return TermTilix
		}
		return TermGNOME
	}

	// Layer 5: Emacs terminal emulators.
	if os.Getenv("INSIDE_EMACS") != "" {
		return TermEmacs
	}

	// Layer 6: Multiplexer detection. Checked late so inner terminal
	// detection from TERM_PROGRAM takes priority.
	if os.Getenv("TMUX") != "" {
		return TermTmux
	}
	if os.Getenv("STY") != "" {
		return TermScreen
	}

	// Layer 7: VS Code sets TERM_PROGRAM_VERSION but may not set TERM_PROGRAM
	// on older versions. Also check LC_TERMINAL for iTerm2 via SSH.
	if os.Getenv("LC_TERMINAL") == "iTerm2" {
		return TermITerm2
	}

	return TermGeneric
}
