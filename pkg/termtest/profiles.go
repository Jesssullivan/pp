package termtest

// ttGhosttyProfile returns the Ghostty terminal profile.
// Ghostty supports truecolor, kitty protocol, full unicode, mouse, pixel query, OSC52.
func ttGhosttyProfile() TerminalProfile {
	return TerminalProfile{
		Name: "Ghostty",
		EnvVars: map[string]string{
			"TERM_PROGRAM": "ghostty",
			"TERM":         "xterm-ghostty",
			"COLORTERM":    "truecolor",
		},
		ColorDepth:     24,
		ImageProtocol:  "kitty",
		UnicodeFull:    true,
		BoxDrawing:     true,
		Ligatures:      true,
		MouseSupport:   true,
		BracketPaste:   true,
		OSC52Clipboard: true,
		PixelSizeQuery: true,
		ResizeEvents:   true,
	}
}

// ttKittyProfile returns the Kitty terminal profile.
// Kitty supports truecolor, kitty protocol, full unicode, mouse, pixel query, OSC52.
func ttKittyProfile() TerminalProfile {
	return TerminalProfile{
		Name: "Kitty",
		EnvVars: map[string]string{
			"TERM_PROGRAM":  "kitty",
			"TERM":          "xterm-kitty",
			"COLORTERM":     "truecolor",
			"KITTY_WINDOW_ID": "1",
		},
		ColorDepth:     24,
		ImageProtocol:  "kitty",
		UnicodeFull:    true,
		BoxDrawing:     true,
		Ligatures:      true,
		MouseSupport:   true,
		BracketPaste:   true,
		OSC52Clipboard: true,
		PixelSizeQuery: true,
		ResizeEvents:   true,
	}
}

// ttITerm2Profile returns the iTerm2 terminal profile.
// iTerm2 supports truecolor, iterm2 protocol, full unicode, mouse, OSC52.
func ttITerm2Profile() TerminalProfile {
	return TerminalProfile{
		Name: "iTerm2",
		EnvVars: map[string]string{
			"TERM_PROGRAM":    "iTerm.app",
			"TERM":            "xterm-256color",
			"COLORTERM":       "truecolor",
			"ITERM_SESSION_ID": "w0t0p0:ABCDEF-1234",
		},
		ColorDepth:     24,
		ImageProtocol:  "iterm2",
		UnicodeFull:    true,
		BoxDrawing:     true,
		Ligatures:      false,
		MouseSupport:   true,
		BracketPaste:   true,
		OSC52Clipboard: true,
		PixelSizeQuery: false,
		ResizeEvents:   true,
	}
}

// ttWezTermProfile returns the WezTerm terminal profile.
// WezTerm supports truecolor, kitty+iterm2 protocols, full unicode, mouse,
// pixel query.
func ttWezTermProfile() TerminalProfile {
	return TerminalProfile{
		Name: "WezTerm",
		EnvVars: map[string]string{
			"TERM_PROGRAM":       "WezTerm",
			"TERM":               "xterm-256color",
			"COLORTERM":          "truecolor",
			"WEZTERM_EXECUTABLE": "/usr/local/bin/wezterm",
		},
		ColorDepth:     24,
		ImageProtocol:  "kitty",
		UnicodeFull:    true,
		BoxDrawing:     true,
		Ligatures:      true,
		MouseSupport:   true,
		BracketPaste:   true,
		OSC52Clipboard: true,
		PixelSizeQuery: true,
		ResizeEvents:   true,
	}
}

// ttTilixProfile returns the Tilix terminal profile.
// Tilix is VTE-based with 256-color (no truecolor in some versions), sixel,
// box drawing, no pixel query.
func ttTilixProfile() TerminalProfile {
	return TerminalProfile{
		Name: "Tilix",
		EnvVars: map[string]string{
			"TERM_PROGRAM": "",
			"TERM":         "xterm-256color",
			"VTE_VERSION":  "6800",
			"TILIX_ID":     "some-id",
		},
		ColorDepth:     256,
		ImageProtocol:  "sixel",
		UnicodeFull:    false,
		BoxDrawing:     true,
		Ligatures:      false,
		MouseSupport:   true,
		BracketPaste:   true,
		OSC52Clipboard: false,
		PixelSizeQuery: false,
		ResizeEvents:   true,
	}
}

// ttAlacrittyProfile returns the Alacritty terminal profile.
// Alacritty supports truecolor, halfblock only (no image protocol), full
// unicode, mouse.
func ttAlacrittyProfile() TerminalProfile {
	return TerminalProfile{
		Name: "Alacritty",
		EnvVars: map[string]string{
			"TERM_PROGRAM": "alacritty",
			"TERM":         "alacritty",
			"COLORTERM":    "truecolor",
		},
		ColorDepth:     24,
		ImageProtocol:  "halfblock",
		UnicodeFull:    true,
		BoxDrawing:     true,
		Ligatures:      false,
		MouseSupport:   true,
		BracketPaste:   true,
		OSC52Clipboard: false,
		PixelSizeQuery: false,
		ResizeEvents:   true,
	}
}

// ttAppleTerminalProfile returns the Apple Terminal profile.
// Apple Terminal has 256-color, no image protocol, limited unicode, no mouse.
func ttAppleTerminalProfile() TerminalProfile {
	return TerminalProfile{
		Name: "Apple Terminal",
		EnvVars: map[string]string{
			"TERM_PROGRAM":         "Apple_Terminal",
			"TERM":                 "xterm-256color",
			"TERM_PROGRAM_VERSION": "453",
		},
		ColorDepth:     256,
		ImageProtocol:  "none",
		UnicodeFull:    false,
		BoxDrawing:     true,
		Ligatures:      false,
		MouseSupport:   false,
		BracketPaste:   true,
		OSC52Clipboard: false,
		PixelSizeQuery: false,
		ResizeEvents:   true,
	}
}

// ttTmuxProfile returns the tmux multiplexer profile.
// Tmux depends on outer terminal, uses halfblock or kitty unicode
// placeholders, no direct pixel query.
func ttTmuxProfile() TerminalProfile {
	return TerminalProfile{
		Name: "tmux",
		EnvVars: map[string]string{
			"TERM_PROGRAM": "tmux",
			"TERM":         "screen-256color",
			"TMUX":         "/tmp/tmux-501/default,12345,0",
		},
		ColorDepth:     256,
		ImageProtocol:  "halfblock",
		UnicodeFull:    false,
		BoxDrawing:     true,
		Ligatures:      false,
		MouseSupport:   true,
		BracketPaste:   true,
		OSC52Clipboard: false,
		PixelSizeQuery: false,
		ResizeEvents:   false,
	}
}
