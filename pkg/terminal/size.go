package terminal

import (
	"os"
	"strconv"

	"golang.org/x/sys/unix"
)

// Size represents terminal dimensions in both character cells and pixels.
type Size struct {
	Cols   int // Character columns
	Rows   int // Character rows
	PixelW int // Total pixel width (0 if unknown)
	PixelH int // Total pixel height (0 if unknown)
	CellW  int // Pixel width per cell (0 if unknown)
	CellH  int // Pixel height per cell (0 if unknown)
}

// GetSize returns the current terminal dimensions. It tries multiple
// strategies in order:
//  1. TIOCGWINSZ ioctl on stdout (returns both cell and pixel dimensions)
//  2. TIOCGWINSZ ioctl on stderr (in case stdout is redirected)
//  3. COLUMNS/LINES environment variables
//  4. Fallback to 80x24
func GetSize() Size {
	// Try stdout first, then stderr.
	for _, fd := range []uintptr{os.Stdout.Fd(), os.Stderr.Fd()} {
		if s := getSizeFromIoctl(fd); s.Cols > 0 && s.Rows > 0 {
			return s
		}
	}
	return getSizeFromEnv()
}

// GetSizeFromFd returns terminal size from a specific file descriptor.
// Falls back to environment variables and then 80x24 defaults if the
// ioctl fails.
func GetSizeFromFd(fd uintptr) Size {
	if s := getSizeFromIoctl(fd); s.Cols > 0 && s.Rows > 0 {
		return s
	}
	return getSizeFromEnv()
}

// getSizeFromIoctl queries the terminal size via TIOCGWINSZ ioctl.
// Returns a zero-value Size on failure.
func getSizeFromIoctl(fd uintptr) Size {
	ws, err := unix.IoctlGetWinsize(int(fd), unix.TIOCGWINSZ)
	if err != nil {
		return Size{}
	}

	s := Size{
		Cols:   int(ws.Col),
		Rows:   int(ws.Row),
		PixelW: int(ws.Xpixel),
		PixelH: int(ws.Ypixel),
	}

	// Calculate per-cell pixel dimensions when pixel info is available.
	if s.PixelW > 0 && s.Cols > 0 {
		s.CellW = s.PixelW / s.Cols
	}
	if s.PixelH > 0 && s.Rows > 0 {
		s.CellH = s.PixelH / s.Rows
	}

	return s
}

// getSizeFromEnv reads terminal dimensions from COLUMNS/LINES environment
// variables, falling back to 80x24 defaults.
func getSizeFromEnv() Size {
	cols := envInt("COLUMNS", 80)
	rows := envInt("LINES", 24)
	return Size{Cols: cols, Rows: rows}
}

// envInt reads an integer from the named environment variable. Returns
// the fallback value if the variable is unset, empty, or not a valid
// positive integer.
func envInt(name string, fallback int) int {
	v := os.Getenv(name)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return fallback
	}
	return n
}
