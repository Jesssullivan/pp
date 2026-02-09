// Package termtest provides terminal emulator profiles, compatibility matrices,
// and rendering validation for ensuring the dashboard works across terminals.
// It is used in tests and diagnostics to verify cross-terminal rendering.
package termtest

// TerminalProfile describes a terminal's capabilities for testing.
type TerminalProfile struct {
	Name           string            // Human-readable terminal name
	EnvVars        map[string]string // Environment vars this terminal sets
	ColorDepth     int               // 256 or 24 (truecolor)
	ImageProtocol  string            // "kitty", "iterm2", "sixel", "halfblock", "none"
	UnicodeFull    bool              // Supports full Unicode including U+10EEEE
	BoxDrawing     bool              // Supports box drawing characters
	Ligatures      bool              // Supports font ligatures
	MouseSupport   bool              // Supports mouse events
	BracketPaste   bool              // Supports bracketed paste
	OSC52Clipboard bool              // Supports OSC 52 clipboard
	PixelSizeQuery bool              // Supports CSI 16t pixel size query
	ResizeEvents   bool              // Emits SIGWINCH reliably
}

// Profiles returns all known terminal profiles.
func Profiles() []TerminalProfile {
	return []TerminalProfile{
		ttGhosttyProfile(),
		ttKittyProfile(),
		ttITerm2Profile(),
		ttWezTermProfile(),
		ttTilixProfile(),
		ttAlacrittyProfile(),
		ttAppleTerminalProfile(),
		ttTmuxProfile(),
	}
}

// ProfileByName returns the profile matching the given name, or nil if not found.
func ProfileByName(name string) *TerminalProfile {
	for _, p := range Profiles() {
		if p.Name == name {
			cp := p
			return &cp
		}
	}
	return nil
}
