//go:build !unix

package image

import "fmt"

// imgDetectCellSizeIOCTLPlatform is a stub for non-Unix platforms.
func imgDetectCellSizeIOCTLPlatform() (pixelW, pixelH int, err error) {
	return 0, 0, fmt.Errorf("TIOCGWINSZ not available on this platform")
}
