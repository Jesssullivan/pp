package terminal

import (
	"os"
	"sync"
)

// Capabilities is the cached terminal capability summary for the current
// session. It aggregates the results of terminal detection, protocol
// selection, and size query into a single struct.
type Capabilities struct {
	Term      Terminal         // Detected terminal emulator
	Protocol  GraphicsProtocol // Selected graphics protocol
	Size      Size             // Terminal dimensions
	TrueColor bool             // 24-bit color support
	SSH       bool             // Running over SSH
	Tmux      bool             // Inside tmux
	Mux       bool             // Inside any multiplexer (tmux, screen, zellij)
}

var (
	cached     *Capabilities
	detectOnce sync.Once
	mu         sync.Mutex // guards ForceRefresh reset
)

// DetectCapabilities performs full terminal detection and caches the result.
// Safe to call from multiple goroutines; detection runs exactly once via
// sync.Once. Subsequent calls return the cached value.
func DetectCapabilities() *Capabilities {
	detectOnce.Do(func() {
		cached = detect()
	})
	return cached
}

// ForceRefresh re-detects terminal capabilities, replacing the cached
// value. Use this after a terminal change (e.g., attaching/detaching
// from tmux).
func ForceRefresh() *Capabilities {
	mu.Lock()
	defer mu.Unlock()

	// Reset the Once so DetectCapabilities runs detection again on
	// its next call. We also run detection immediately here.
	detectOnce = sync.Once{}
	cached = detect()
	return cached
}

// Cached returns the previously cached capabilities without re-detection.
// Returns nil if DetectCapabilities has not been called yet.
func Cached() *Capabilities {
	return cached
}

// detect performs the actual detection work.
func detect() *Capabilities {
	term := Detect()
	ssh := isSSH()
	tmux := os.Getenv("TMUX") != ""
	screen := os.Getenv("STY") != ""

	// True color: either the terminal natively supports it, or
	// COLORTERM=truecolor is set (common in well-configured setups).
	trueColor := term.SupportsTrueColor()
	if !trueColor {
		ct := os.Getenv("COLORTERM")
		trueColor = ct == "truecolor" || ct == "24bit"
	}

	return &Capabilities{
		Term:      term,
		Protocol:  SelectProtocol(term),
		Size:      GetSize(),
		TrueColor: trueColor,
		SSH:       ssh,
		Tmux:      tmux,
		Mux:       tmux || screen,
	}
}
