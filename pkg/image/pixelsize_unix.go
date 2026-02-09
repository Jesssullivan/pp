//go:build unix

package image

import (
	"fmt"
	"os"

	"golang.org/x/sys/unix"
)

// imgDetectCellSizeIOCTLPlatform uses TIOCGWINSZ on Unix systems to read
// the terminal pixel dimensions and derive cell size.
func imgDetectCellSizeIOCTLPlatform() (pixelW, pixelH int, err error) {
	f, err := os.Open("/dev/tty")
	if err != nil {
		return 0, 0, fmt.Errorf("open /dev/tty: %w", err)
	}
	defer f.Close()

	ws, err := unix.IoctlGetWinsize(int(f.Fd()), unix.TIOCGWINSZ)
	if err != nil {
		return 0, 0, fmt.Errorf("TIOCGWINSZ: %w", err)
	}

	if ws.Xpixel == 0 || ws.Ypixel == 0 || ws.Col == 0 || ws.Row == 0 {
		return 0, 0, fmt.Errorf("TIOCGWINSZ returned zero dimensions")
	}

	cellW := int(ws.Xpixel) / int(ws.Col)
	cellH := int(ws.Ypixel) / int(ws.Row)

	if cellW <= 0 || cellH <= 0 {
		return 0, 0, fmt.Errorf("computed cell size is zero or negative")
	}

	return cellW, cellH, nil
}
