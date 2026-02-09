package terminal

import (
	"os"
	"strings"
)

// GraphicsProtocol identifies which image rendering protocol to use.
type GraphicsProtocol int

const (
	ProtocolNone       GraphicsProtocol = iota // No graphics support
	ProtocolKitty                              // Kitty graphics protocol (Ghostty, Kitty, WezTerm)
	ProtocolITerm2                             // iTerm2 inline images protocol
	ProtocolSixel                              // Sixel graphics protocol
	ProtocolHalfblocks                         // Unicode half-block characters with ANSI color
)

// protocolNames maps GraphicsProtocol values to human-readable strings.
var protocolNames = [...]string{
	ProtocolNone:       "none",
	ProtocolKitty:      "kitty",
	ProtocolITerm2:     "iterm2",
	ProtocolSixel:      "sixel",
	ProtocolHalfblocks: "halfblocks",
}

// String returns the human-readable name of the graphics protocol.
func (p GraphicsProtocol) String() string {
	if int(p) < len(protocolNames) {
		return protocolNames[p]
	}
	return "unknown"
}

// SelectProtocol returns the best graphics protocol for the detected terminal.
// The selection follows a cascade:
//   - Ghostty, Kitty, WezTerm -> ProtocolKitty
//   - iTerm2 -> ProtocolITerm2
//   - WezTerm also supports Sixel but Kitty is preferred
//   - All others -> ProtocolHalfblocks (safest fallback)
//
// SSH sessions degrade by one level for reliability.
func SelectProtocol(term Terminal) GraphicsProtocol {
	proto := selectBaseProtocol(term)

	// SSH sessions: degrade one level for reliability. Kitty graphics
	// over SSH is often unreliable; iTerm2 images may work but sixel
	// is safer; sixel degrades to halfblocks.
	if isSSH() {
		switch proto {
		case ProtocolKitty:
			return ProtocolHalfblocks
		case ProtocolITerm2:
			return ProtocolHalfblocks
		case ProtocolSixel:
			return ProtocolHalfblocks
		}
	}

	return proto
}

// selectBaseProtocol returns the ideal protocol for a terminal without
// considering environmental degradation (SSH, etc.).
func selectBaseProtocol(term Terminal) GraphicsProtocol {
	switch term {
	case TermGhostty, TermKitty, TermWezTerm:
		return ProtocolKitty
	case TermITerm2:
		return ProtocolITerm2
	default:
		return ProtocolHalfblocks
	}
}

// SelectProtocolWithOverride allows user configuration to force a specific
// graphics protocol. If override is empty, detection proceeds normally.
// Valid override values: "kitty", "iterm2", "sixel", "halfblocks", "none".
func SelectProtocolWithOverride(term Terminal, override string) GraphicsProtocol {
	if override == "" {
		return SelectProtocol(term)
	}
	switch strings.ToLower(override) {
	case "kitty":
		return ProtocolKitty
	case "iterm2":
		return ProtocolITerm2
	case "sixel":
		return ProtocolSixel
	case "halfblocks", "unicode", "half-blocks":
		return ProtocolHalfblocks
	case "none", "off", "disabled":
		return ProtocolNone
	default:
		// Unknown override, fall back to detection.
		return SelectProtocol(term)
	}
}

// isSSH reports whether the current session is running over SSH.
func isSSH() bool {
	return os.Getenv("SSH_TTY") != "" ||
		os.Getenv("SSH_CONNECTION") != "" ||
		os.Getenv("SSH_CLIENT") != ""
}
