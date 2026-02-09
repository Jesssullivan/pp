package image

import (
	"math"
)

// imgDefaultCellW and imgDefaultCellH are fallback cell pixel dimensions
// used when detection fails. These are common for 80-column terminals with
// standard fonts.
const (
	imgDefaultCellW = 8
	imgDefaultCellH = 16
)

// imgDetectCellSize detects the pixel dimensions of a single terminal cell.
//
// It tries the following strategies in order:
//  1. TIOCGWINSZ ioctl: if the terminal reports pixel dimensions and column/
//     row counts, cell size is pixels / cells.
//  2. CSI 16t query: sends \x1b[16t and parses the response. (Not
//     implemented in this version; requires raw terminal mode.)
//  3. Fallback: returns imgDefaultCellW x imgDefaultCellH.
//
// This function is safe to call from any goroutine; it does not modify
// terminal state.
func imgDetectCellSize() (pixelW, pixelH int, err error) {
	// Strategy 1: try ioctl-based detection.
	w, h, ioErr := imgDetectCellSizeIOCTL()
	if ioErr == nil && w > 0 && h > 0 {
		return w, h, nil
	}

	// Strategy 3: fallback to common defaults.
	return imgDefaultCellW, imgDefaultCellH, nil
}

// imgDetectCellSizeIOCTL attempts to read cell pixel size from the terminal
// via TIOCGWINSZ ioctl. It returns the cell width and height in pixels, or
// an error if the ioctl is unavailable or returns zero pixel dimensions.
//
// The actual ioctl call is in pixelsize_unix.go (build-tagged). On
// unsupported platforms this returns an error.
func imgDetectCellSizeIOCTL() (pixelW, pixelH int, err error) {
	return imgDetectCellSizeIOCTLPlatform()
}

// imgPixelPerfectSize calculates the optimal number of terminal cells (cols,
// rows) to display an image of imgW x imgH pixels, given cell dimensions of
// cellW x cellH pixels. The result maintains the source aspect ratio and
// does not exceed maxCols or maxRows.
//
// The calculation aligns to cell boundaries: the returned cols/rows represent
// the smallest cell grid that can display the image without exceeding the
// maximum dimensions while preserving the aspect ratio.
func imgPixelPerfectSize(imgW, imgH, cellW, cellH, maxCols, maxRows int) (cols, rows int) {
	if imgW <= 0 || imgH <= 0 {
		return 1, 1
	}
	if cellW <= 0 {
		cellW = imgDefaultCellW
	}
	if cellH <= 0 {
		cellH = imgDefaultCellH
	}
	if maxCols <= 0 {
		maxCols = 1
	}
	if maxRows <= 0 {
		maxRows = 1
	}

	// How many cells needed at native resolution?
	nativeCols := int(math.Ceil(float64(imgW) / float64(cellW)))
	nativeRows := int(math.Ceil(float64(imgH) / float64(cellH)))

	// If it fits within the max, use native.
	if nativeCols <= maxCols && nativeRows <= maxRows {
		return nativeCols, nativeRows
	}

	// Scale down preserving aspect ratio.
	aspectRatio := float64(imgW) / float64(imgH)

	// Try fitting by width.
	cols = maxCols
	rows = int(math.Round(float64(cols*cellW) / aspectRatio / float64(cellH)))
	if rows < 1 {
		rows = 1
	}

	// If rows exceeds max, fit by height instead.
	if rows > maxRows {
		rows = maxRows
		cols = int(math.Round(float64(rows*cellH) * aspectRatio / float64(cellW)))
		if cols < 1 {
			cols = 1
		}
	}

	// Final clamp.
	if cols > maxCols {
		cols = maxCols
	}
	if rows > maxRows {
		rows = maxRows
	}

	return cols, rows
}
