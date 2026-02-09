package terminal

import (
	"os"
	"testing"
)

// termEnvVars lists all environment variables inspected during detection.
// Tests clear these before each case to ensure isolation.
var termEnvVars = []string{
	"TERM_PROGRAM", "TERM", "COLORTERM",
	"KITTY_WINDOW_ID", "ITERM_SESSION_ID", "WEZTERM_EXECUTABLE",
	"TILIX_ID", "VTE_VERSION", "LC_TERMINAL",
	"INSIDE_EMACS", "TMUX", "STY",
	"SSH_TTY", "SSH_CONNECTION", "SSH_CLIENT",
	"COLUMNS", "LINES",
}

// clearTermEnv unsets all terminal-related env vars for test isolation.
// Uses t.Setenv under the hood (via save/restore) so cleanup is automatic.
func clearTermEnv(t *testing.T) {
	t.Helper()
	for _, v := range termEnvVars {
		t.Setenv(v, "")
		os.Unsetenv(v)
	}
}

// --- Terminal Detection Tests ---

func TestDetect_Ghostty_TermProgram(t *testing.T) {
	clearTermEnv(t)
	t.Setenv("TERM_PROGRAM", "ghostty")

	got := Detect()
	if got != TermGhostty {
		t.Errorf("Detect() = %v, want %v", got, TermGhostty)
	}
}

func TestDetect_Ghostty_Term(t *testing.T) {
	clearTermEnv(t)
	t.Setenv("TERM", "xterm-ghostty")

	got := Detect()
	if got != TermGhostty {
		t.Errorf("Detect() = %v, want %v", got, TermGhostty)
	}
}

func TestDetect_Kitty_TermProgram(t *testing.T) {
	clearTermEnv(t)
	t.Setenv("TERM_PROGRAM", "kitty")

	got := Detect()
	if got != TermKitty {
		t.Errorf("Detect() = %v, want %v", got, TermKitty)
	}
}

func TestDetect_Kitty_Term(t *testing.T) {
	clearTermEnv(t)
	t.Setenv("TERM", "xterm-kitty")

	got := Detect()
	if got != TermKitty {
		t.Errorf("Detect() = %v, want %v", got, TermKitty)
	}
}

func TestDetect_Kitty_WindowID(t *testing.T) {
	clearTermEnv(t)
	t.Setenv("KITTY_WINDOW_ID", "123")

	got := Detect()
	if got != TermKitty {
		t.Errorf("Detect() = %v, want %v", got, TermKitty)
	}
}

func TestDetect_WezTerm_TermProgram(t *testing.T) {
	clearTermEnv(t)
	t.Setenv("TERM_PROGRAM", "WezTerm")

	got := Detect()
	if got != TermWezTerm {
		t.Errorf("Detect() = %v, want %v", got, TermWezTerm)
	}
}

func TestDetect_WezTerm_Executable(t *testing.T) {
	clearTermEnv(t)
	t.Setenv("WEZTERM_EXECUTABLE", "/usr/local/bin/wezterm")

	got := Detect()
	if got != TermWezTerm {
		t.Errorf("Detect() = %v, want %v", got, TermWezTerm)
	}
}

func TestDetect_ITerm2_TermProgram(t *testing.T) {
	clearTermEnv(t)
	t.Setenv("TERM_PROGRAM", "iTerm.app")

	got := Detect()
	if got != TermITerm2 {
		t.Errorf("Detect() = %v, want %v", got, TermITerm2)
	}
}

func TestDetect_ITerm2_SessionID(t *testing.T) {
	clearTermEnv(t)
	t.Setenv("ITERM_SESSION_ID", "w0t0p0:ABCDEF-1234")

	got := Detect()
	if got != TermITerm2 {
		t.Errorf("Detect() = %v, want %v", got, TermITerm2)
	}
}

func TestDetect_ITerm2_LCTerminal(t *testing.T) {
	clearTermEnv(t)
	t.Setenv("LC_TERMINAL", "iTerm2")

	got := Detect()
	if got != TermITerm2 {
		t.Errorf("Detect() = %v, want %v", got, TermITerm2)
	}
}

func TestDetect_Alacritty_TermProgram(t *testing.T) {
	clearTermEnv(t)
	t.Setenv("TERM_PROGRAM", "alacritty")

	got := Detect()
	if got != TermAlacritty {
		t.Errorf("Detect() = %v, want %v", got, TermAlacritty)
	}
}

func TestDetect_Alacritty_Term(t *testing.T) {
	clearTermEnv(t)
	t.Setenv("TERM", "alacritty")

	got := Detect()
	if got != TermAlacritty {
		t.Errorf("Detect() = %v, want %v", got, TermAlacritty)
	}
}

func TestDetect_VTE_Tilix(t *testing.T) {
	clearTermEnv(t)
	t.Setenv("VTE_VERSION", "6800")
	t.Setenv("TILIX_ID", "some-id")

	got := Detect()
	if got != TermTilix {
		t.Errorf("Detect() = %v, want %v", got, TermTilix)
	}
}

func TestDetect_VTE_GNOME(t *testing.T) {
	clearTermEnv(t)
	t.Setenv("VTE_VERSION", "6800")

	got := Detect()
	if got != TermGNOME {
		t.Errorf("Detect() = %v, want %v", got, TermGNOME)
	}
}

func TestDetect_VSCode(t *testing.T) {
	clearTermEnv(t)
	t.Setenv("TERM_PROGRAM", "vscode")

	got := Detect()
	if got != TermVSCode {
		t.Errorf("Detect() = %v, want %v", got, TermVSCode)
	}
}

func TestDetect_Emacs(t *testing.T) {
	clearTermEnv(t)
	t.Setenv("INSIDE_EMACS", "29.1,vterm")

	got := Detect()
	if got != TermEmacs {
		t.Errorf("Detect() = %v, want %v", got, TermEmacs)
	}
}

func TestDetect_Tmux(t *testing.T) {
	clearTermEnv(t)
	t.Setenv("TMUX", "/tmp/tmux-501/default,12345,0")

	got := Detect()
	if got != TermTmux {
		t.Errorf("Detect() = %v, want %v", got, TermTmux)
	}
}

func TestDetect_Screen(t *testing.T) {
	clearTermEnv(t)
	t.Setenv("STY", "12345.pts-0.host")
	t.Setenv("TERM", "screen-256color")

	got := Detect()
	if got != TermScreen {
		t.Errorf("Detect() = %v, want %v", got, TermScreen)
	}
}

func TestDetect_Generic(t *testing.T) {
	clearTermEnv(t)

	got := Detect()
	if got != TermGeneric {
		t.Errorf("Detect() = %v, want %v", got, TermGeneric)
	}
}

func TestDetect_TermProgram_Priority(t *testing.T) {
	// TERM_PROGRAM should take priority over TMUX.
	clearTermEnv(t)
	t.Setenv("TERM_PROGRAM", "ghostty")
	t.Setenv("TMUX", "/tmp/tmux-501/default,12345,0")

	got := Detect()
	if got != TermGhostty {
		t.Errorf("Detect() = %v, want TermGhostty (TERM_PROGRAM should win over TMUX)", got)
	}
}

// --- Terminal String Tests ---

func TestTerminal_String(t *testing.T) {
	cases := []struct {
		term Terminal
		want string
	}{
		{TermUnknown, "unknown"},
		{TermGhostty, "ghostty"},
		{TermKitty, "kitty"},
		{TermWezTerm, "wezterm"},
		{TermITerm2, "iterm2"},
		{TermAlacritty, "alacritty"},
		{TermTilix, "tilix"},
		{TermGNOME, "gnome-terminal"},
		{TermTmux, "tmux"},
		{TermScreen, "screen"},
		{TermVSCode, "vscode"},
		{TermEmacs, "emacs"},
		{TermGeneric, "generic"},
		{Terminal(99), "unknown"},
	}
	for _, tc := range cases {
		if got := tc.term.String(); got != tc.want {
			t.Errorf("%d.String() = %q, want %q", tc.term, got, tc.want)
		}
	}
}

// --- Capability Method Tests ---

func TestTerminal_SupportsKittyGraphics(t *testing.T) {
	yes := []Terminal{TermGhostty, TermKitty, TermWezTerm}
	no := []Terminal{TermITerm2, TermAlacritty, TermTilix, TermGNOME, TermGeneric, TermEmacs}

	for _, term := range yes {
		if !term.SupportsKittyGraphics() {
			t.Errorf("%v.SupportsKittyGraphics() = false, want true", term)
		}
	}
	for _, term := range no {
		if term.SupportsKittyGraphics() {
			t.Errorf("%v.SupportsKittyGraphics() = true, want false", term)
		}
	}
}

func TestTerminal_SupportsSixel(t *testing.T) {
	if !TermWezTerm.SupportsSixel() {
		t.Error("WezTerm.SupportsSixel() = false, want true")
	}
	if TermKitty.SupportsSixel() {
		t.Error("Kitty.SupportsSixel() = true, want false")
	}
}

func TestTerminal_SupportsITerm2Images(t *testing.T) {
	yes := []Terminal{TermITerm2, TermWezTerm}
	no := []Terminal{TermGhostty, TermKitty, TermAlacritty, TermGeneric}

	for _, term := range yes {
		if !term.SupportsITerm2Images() {
			t.Errorf("%v.SupportsITerm2Images() = false, want true", term)
		}
	}
	for _, term := range no {
		if term.SupportsITerm2Images() {
			t.Errorf("%v.SupportsITerm2Images() = true, want false", term)
		}
	}
}

func TestTerminal_SupportsTrueColor(t *testing.T) {
	yes := []Terminal{TermGhostty, TermKitty, TermWezTerm, TermITerm2,
		TermAlacritty, TermTilix, TermGNOME, TermVSCode}
	no := []Terminal{TermTmux, TermScreen, TermEmacs, TermGeneric, TermUnknown}

	for _, term := range yes {
		if !term.SupportsTrueColor() {
			t.Errorf("%v.SupportsTrueColor() = false, want true", term)
		}
	}
	for _, term := range no {
		if term.SupportsTrueColor() {
			t.Errorf("%v.SupportsTrueColor() = true, want false", term)
		}
	}
}

func TestTerminal_SupportsSyncOutput(t *testing.T) {
	if !TermGhostty.SupportsSyncOutput() {
		t.Error("Ghostty.SupportsSyncOutput() = false, want true")
	}
	if TermEmacs.SupportsSyncOutput() {
		t.Error("Emacs.SupportsSyncOutput() = true, want false")
	}
}

func TestTerminal_SupportsKittyKeyboard(t *testing.T) {
	yes := []Terminal{TermGhostty, TermKitty, TermWezTerm}
	no := []Terminal{TermITerm2, TermAlacritty, TermVSCode}

	for _, term := range yes {
		if !term.SupportsKittyKeyboard() {
			t.Errorf("%v.SupportsKittyKeyboard() = false, want true", term)
		}
	}
	for _, term := range no {
		if term.SupportsKittyKeyboard() {
			t.Errorf("%v.SupportsKittyKeyboard() = true, want false", term)
		}
	}
}

// --- Protocol Selection Tests ---

func TestSelectProtocol_Ghostty(t *testing.T) {
	clearTermEnv(t)
	if got := SelectProtocol(TermGhostty); got != ProtocolKitty {
		t.Errorf("SelectProtocol(Ghostty) = %v, want ProtocolKitty", got)
	}
}

func TestSelectProtocol_Kitty(t *testing.T) {
	clearTermEnv(t)
	if got := SelectProtocol(TermKitty); got != ProtocolKitty {
		t.Errorf("SelectProtocol(Kitty) = %v, want ProtocolKitty", got)
	}
}

func TestSelectProtocol_WezTerm(t *testing.T) {
	clearTermEnv(t)
	if got := SelectProtocol(TermWezTerm); got != ProtocolKitty {
		t.Errorf("SelectProtocol(WezTerm) = %v, want ProtocolKitty", got)
	}
}

func TestSelectProtocol_ITerm2(t *testing.T) {
	clearTermEnv(t)
	if got := SelectProtocol(TermITerm2); got != ProtocolITerm2 {
		t.Errorf("SelectProtocol(ITerm2) = %v, want ProtocolITerm2", got)
	}
}

func TestSelectProtocol_Alacritty(t *testing.T) {
	clearTermEnv(t)
	if got := SelectProtocol(TermAlacritty); got != ProtocolHalfblocks {
		t.Errorf("SelectProtocol(Alacritty) = %v, want ProtocolHalfblocks", got)
	}
}

func TestSelectProtocol_Generic(t *testing.T) {
	clearTermEnv(t)
	if got := SelectProtocol(TermGeneric); got != ProtocolHalfblocks {
		t.Errorf("SelectProtocol(Generic) = %v, want ProtocolHalfblocks", got)
	}
}

func TestSelectProtocol_SSH_Downgrade(t *testing.T) {
	clearTermEnv(t)
	t.Setenv("SSH_TTY", "/dev/pts/0")

	got := SelectProtocol(TermGhostty)
	if got != ProtocolHalfblocks {
		t.Errorf("SelectProtocol(Ghostty) over SSH = %v, want ProtocolHalfblocks", got)
	}
}

func TestSelectProtocol_SSH_ITerm2_Downgrade(t *testing.T) {
	clearTermEnv(t)
	t.Setenv("SSH_CONNECTION", "10.0.0.1 12345 10.0.0.2 22")

	got := SelectProtocol(TermITerm2)
	if got != ProtocolHalfblocks {
		t.Errorf("SelectProtocol(ITerm2) over SSH = %v, want ProtocolHalfblocks", got)
	}
}

func TestSelectProtocol_NoSSH_NoDowngrade(t *testing.T) {
	clearTermEnv(t)

	got := SelectProtocol(TermGhostty)
	if got != ProtocolKitty {
		t.Errorf("SelectProtocol(Ghostty) without SSH = %v, want ProtocolKitty", got)
	}
}

// --- Protocol Override Tests ---

func TestSelectProtocolWithOverride_Explicit(t *testing.T) {
	clearTermEnv(t)
	cases := []struct {
		override string
		want     GraphicsProtocol
	}{
		{"kitty", ProtocolKitty},
		{"iterm2", ProtocolITerm2},
		{"sixel", ProtocolSixel},
		{"halfblocks", ProtocolHalfblocks},
		{"unicode", ProtocolHalfblocks},
		{"half-blocks", ProtocolHalfblocks},
		{"none", ProtocolNone},
		{"off", ProtocolNone},
		{"disabled", ProtocolNone},
	}
	for _, tc := range cases {
		got := SelectProtocolWithOverride(TermGeneric, tc.override)
		if got != tc.want {
			t.Errorf("SelectProtocolWithOverride(Generic, %q) = %v, want %v",
				tc.override, got, tc.want)
		}
	}
}

func TestSelectProtocolWithOverride_Empty(t *testing.T) {
	clearTermEnv(t)
	got := SelectProtocolWithOverride(TermGhostty, "")
	if got != ProtocolKitty {
		t.Errorf("SelectProtocolWithOverride(Ghostty, \"\") = %v, want ProtocolKitty", got)
	}
}

func TestSelectProtocolWithOverride_Invalid(t *testing.T) {
	clearTermEnv(t)
	got := SelectProtocolWithOverride(TermKitty, "bogus")
	if got != ProtocolKitty {
		t.Errorf("SelectProtocolWithOverride(Kitty, \"bogus\") = %v, want ProtocolKitty (fallback to detection)", got)
	}
}

// --- Protocol String Tests ---

func TestGraphicsProtocol_String(t *testing.T) {
	cases := []struct {
		proto GraphicsProtocol
		want  string
	}{
		{ProtocolNone, "none"},
		{ProtocolKitty, "kitty"},
		{ProtocolITerm2, "iterm2"},
		{ProtocolSixel, "sixel"},
		{ProtocolHalfblocks, "halfblocks"},
		{GraphicsProtocol(99), "unknown"},
	}
	for _, tc := range cases {
		if got := tc.proto.String(); got != tc.want {
			t.Errorf("%d.String() = %q, want %q", tc.proto, got, tc.want)
		}
	}
}

// --- Size Tests ---

func TestGetSize_EnvFallback(t *testing.T) {
	// In a test runner, ioctl will likely fail (no TTY), so env vars
	// or defaults should be returned.
	t.Setenv("COLUMNS", "132")
	t.Setenv("LINES", "50")

	s := GetSize()
	// The ioctl may succeed if running in a terminal, so we just
	// verify we get positive values.
	if s.Cols <= 0 {
		t.Errorf("Size.Cols = %d, want > 0", s.Cols)
	}
	if s.Rows <= 0 {
		t.Errorf("Size.Rows = %d, want > 0", s.Rows)
	}
}

func TestGetSize_Defaults(t *testing.T) {
	// Clear COLUMNS/LINES to test 80x24 fallback (when ioctl also fails).
	clearTermEnv(t)

	s := GetSize()
	if s.Cols <= 0 {
		t.Errorf("Size.Cols = %d, want > 0", s.Cols)
	}
	if s.Rows <= 0 {
		t.Errorf("Size.Rows = %d, want > 0", s.Rows)
	}
}

func TestGetSizeFromFd_InvalidFd(t *testing.T) {
	clearTermEnv(t)
	t.Setenv("COLUMNS", "100")
	t.Setenv("LINES", "30")

	// fd 999 is invalid; should fall back to env.
	s := GetSizeFromFd(999)
	if s.Cols != 100 {
		t.Errorf("Size.Cols = %d, want 100", s.Cols)
	}
	if s.Rows != 30 {
		t.Errorf("Size.Rows = %d, want 30", s.Rows)
	}
}

func TestEnvInt(t *testing.T) {
	t.Setenv("TEST_INT_VAR", "42")
	if got := envInt("TEST_INT_VAR", 10); got != 42 {
		t.Errorf("envInt = %d, want 42", got)
	}

	t.Setenv("TEST_INT_VAR", "invalid")
	if got := envInt("TEST_INT_VAR", 10); got != 10 {
		t.Errorf("envInt(invalid) = %d, want 10 (fallback)", got)
	}

	t.Setenv("TEST_INT_VAR", "-5")
	if got := envInt("TEST_INT_VAR", 10); got != 10 {
		t.Errorf("envInt(negative) = %d, want 10 (fallback)", got)
	}

	t.Setenv("TEST_INT_VAR", "")
	if got := envInt("TEST_INT_VAR", 10); got != 10 {
		t.Errorf("envInt(empty) = %d, want 10 (fallback)", got)
	}
}

// --- Capabilities Tests ---

func TestDetectCapabilities_Basic(t *testing.T) {
	clearTermEnv(t)
	t.Setenv("TERM_PROGRAM", "ghostty")
	t.Setenv("COLORTERM", "truecolor")

	// Reset cached state for a clean test.
	ForceRefresh()
	caps := DetectCapabilities()

	if caps == nil {
		t.Fatal("DetectCapabilities() returned nil")
	}
	if caps.Term != TermGhostty {
		t.Errorf("caps.Term = %v, want TermGhostty", caps.Term)
	}
	if caps.Protocol != ProtocolKitty {
		t.Errorf("caps.Protocol = %v, want ProtocolKitty", caps.Protocol)
	}
	if !caps.TrueColor {
		t.Error("caps.TrueColor = false, want true")
	}
	if caps.SSH {
		t.Error("caps.SSH = true, want false")
	}
}

func TestDetectCapabilities_SSH(t *testing.T) {
	clearTermEnv(t)
	t.Setenv("TERM_PROGRAM", "ghostty")
	t.Setenv("SSH_TTY", "/dev/pts/0")

	caps := ForceRefresh()

	if !caps.SSH {
		t.Error("caps.SSH = false, want true")
	}
	// Protocol should be downgraded over SSH.
	if caps.Protocol != ProtocolHalfblocks {
		t.Errorf("caps.Protocol over SSH = %v, want ProtocolHalfblocks", caps.Protocol)
	}
}

func TestDetectCapabilities_Tmux(t *testing.T) {
	clearTermEnv(t)
	t.Setenv("TMUX", "/tmp/tmux-501/default,12345,0")

	caps := ForceRefresh()

	if !caps.Tmux {
		t.Error("caps.Tmux = false, want true")
	}
	if !caps.Mux {
		t.Error("caps.Mux = false, want true")
	}
}

func TestDetectCapabilities_Screen(t *testing.T) {
	clearTermEnv(t)
	t.Setenv("STY", "12345.pts-0.host")
	t.Setenv("TERM", "screen-256color")

	caps := ForceRefresh()

	if !caps.Mux {
		t.Error("caps.Mux = false, want true (screen)")
	}
}

func TestDetectCapabilities_TrueColor_COLORTERM(t *testing.T) {
	clearTermEnv(t)
	t.Setenv("COLORTERM", "truecolor")

	caps := ForceRefresh()

	if !caps.TrueColor {
		t.Error("caps.TrueColor = false, want true (from COLORTERM)")
	}
}

func TestDetectCapabilities_TrueColor_24bit(t *testing.T) {
	clearTermEnv(t)
	t.Setenv("COLORTERM", "24bit")

	caps := ForceRefresh()

	if !caps.TrueColor {
		t.Error("caps.TrueColor = false, want true (from COLORTERM=24bit)")
	}
}

func TestCached_ReturnsLastDetection(t *testing.T) {
	clearTermEnv(t)
	t.Setenv("TERM_PROGRAM", "kitty")

	caps := ForceRefresh()
	cached := Cached()

	if cached != caps {
		t.Error("Cached() did not return the same pointer as ForceRefresh()")
	}
	if cached.Term != TermKitty {
		t.Errorf("Cached().Term = %v, want TermKitty", cached.Term)
	}
}

func TestForceRefresh_Updates(t *testing.T) {
	clearTermEnv(t)
	t.Setenv("TERM_PROGRAM", "kitty")
	caps1 := ForceRefresh()

	clearTermEnv(t)
	t.Setenv("TERM_PROGRAM", "ghostty")
	caps2 := ForceRefresh()

	if caps1.Term == caps2.Term {
		t.Error("ForceRefresh did not re-detect; both returned same terminal")
	}
	if caps2.Term != TermGhostty {
		t.Errorf("After ForceRefresh with ghostty, Term = %v, want TermGhostty", caps2.Term)
	}
}

// --- Integration: Detection -> Protocol Pipeline ---

func TestDetect_to_Protocol_Ghostty(t *testing.T) {
	clearTermEnv(t)
	t.Setenv("TERM_PROGRAM", "ghostty")

	term := Detect()
	proto := SelectProtocol(term)

	if term != TermGhostty {
		t.Errorf("Detect() = %v, want TermGhostty", term)
	}
	if proto != ProtocolKitty {
		t.Errorf("SelectProtocol(Ghostty) = %v, want ProtocolKitty", proto)
	}
}

func TestDetect_to_Protocol_ITerm2(t *testing.T) {
	clearTermEnv(t)
	t.Setenv("TERM_PROGRAM", "iTerm.app")

	term := Detect()
	proto := SelectProtocol(term)

	if term != TermITerm2 {
		t.Errorf("Detect() = %v, want TermITerm2", term)
	}
	if proto != ProtocolITerm2 {
		t.Errorf("SelectProtocol(ITerm2) = %v, want ProtocolITerm2", proto)
	}
}

func TestDetect_to_Protocol_Generic(t *testing.T) {
	clearTermEnv(t)

	term := Detect()
	proto := SelectProtocol(term)

	if term != TermGeneric {
		t.Errorf("Detect() = %v, want TermGeneric", term)
	}
	if proto != ProtocolHalfblocks {
		t.Errorf("SelectProtocol(Generic) = %v, want ProtocolHalfblocks", proto)
	}
}
