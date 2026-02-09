package theme

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// Adapt converts all hex colors in a theme to 256-color ANSI codes if the
// terminal color depth is less than 24-bit. Returns the theme unchanged if
// the terminal supports 24-bit color (colorDepth >= 24).
func Adapt(t Theme, colorDepth int) Theme {
	if colorDepth >= 24 {
		return t
	}

	t.Background = thTo256Color(t.Background)
	t.Foreground = thTo256Color(t.Foreground)
	t.Dim = thTo256Color(t.Dim)
	t.Accent = thTo256Color(t.Accent)

	t.Border = thTo256Color(t.Border)
	t.BorderFocus = thTo256Color(t.BorderFocus)
	t.Title = thTo256Color(t.Title)

	t.StatusOK = thTo256Color(t.StatusOK)
	t.StatusWarn = thTo256Color(t.StatusWarn)
	t.StatusError = thTo256Color(t.StatusError)
	t.StatusUnknown = thTo256Color(t.StatusUnknown)

	t.GaugeFilled = thTo256Color(t.GaugeFilled)
	t.GaugeEmpty = thTo256Color(t.GaugeEmpty)
	t.GaugeWarn = thTo256Color(t.GaugeWarn)
	t.GaugeCrit = thTo256Color(t.GaugeCrit)

	t.ChartLine = thTo256Color(t.ChartLine)
	t.ChartFill = thTo256Color(t.ChartFill)
	t.ChartGrid = thTo256Color(t.ChartGrid)

	t.SearchHighlight = thTo256Color(t.SearchHighlight)
	t.HelpKey = thTo256Color(t.HelpKey)
	t.HelpDesc = thTo256Color(t.HelpDesc)

	return t
}

// thTo256Color converts a hex color string (e.g. "#ff5500") to the nearest
// 256-color ANSI index, returned as a string like "196". Returns the original
// string unchanged if parsing fails.
func thTo256Color(hex string) string {
	r, g, b, ok := thParseHex(hex)
	if !ok {
		return hex
	}

	cubeIdx := thNearestCubeIndex(r, g, b)
	grayIdx := thNearestGray(r, g, b)

	// Calculate distances for both candidates and pick the closer one.
	cubeDist := thCubeDistance(r, g, b, cubeIdx)
	grayDist := thGrayDistance(r, g, b, grayIdx)

	idx := cubeIdx
	if grayDist < cubeDist {
		idx = grayIdx
	}

	return fmt.Sprintf("%d", idx)
}

// thNearestCubeIndex finds the nearest color in the 6x6x6 color cube
// (indices 16-231 of the 256-color palette).
func thNearestCubeIndex(r, g, b uint8) int {
	// The 6x6x6 cube uses values: 0, 95, 135, 175, 215, 255
	ri := thNearestCubeComponent(r)
	gi := thNearestCubeComponent(g)
	bi := thNearestCubeComponent(b)
	return 16 + 36*ri + 6*gi + bi
}

// thNearestCubeComponent maps a 0-255 value to the nearest 6-level cube index (0-5).
func thNearestCubeComponent(v uint8) int {
	// Cube levels: 0, 95, 135, 175, 215, 255
	levels := [6]int{0, 95, 135, 175, 215, 255}
	best := 0
	bestDist := math.MaxInt32
	for i, lv := range levels {
		d := thAbsInt(int(v) - lv)
		if d < bestDist {
			bestDist = d
			best = i
		}
	}
	return best
}

// thNearestGray finds the nearest color in the 24-step grayscale ramp
// (indices 232-255 of the 256-color palette).
func thNearestGray(r, g, b uint8) int {
	// Grayscale ramp: indices 232-255, values 8, 18, 28, ..., 238
	gray := (int(r) + int(g) + int(b)) / 3
	if gray < 4 {
		// Closer to black (index 16 in cube, but we return the first gray).
		return 232
	}
	if gray > 243 {
		// Closer to white (index 231 in cube, but we return the last gray).
		return 255
	}
	// Each gray step is 10 apart starting at 8.
	idx := (gray - 8 + 5) / 10
	if idx < 0 {
		idx = 0
	}
	if idx > 23 {
		idx = 23
	}
	return 232 + idx
}

// thCubeDistance calculates the color distance between an RGB value and a
// 256-color cube index.
func thCubeDistance(r, g, b uint8, cubeIdx int) float64 {
	cr, cg, cb := thCubeToRGB(cubeIdx)
	return thColorDistance(r, g, b, cr, cg, cb)
}

// thGrayDistance calculates the color distance between an RGB value and a
// 256-color grayscale index.
func thGrayDistance(r, g, b uint8, grayIdx int) float64 {
	gv := thGrayToValue(grayIdx)
	return thColorDistance(r, g, b, gv, gv, gv)
}

// thCubeToRGB converts a 256-color cube index (16-231) to RGB values.
func thCubeToRGB(idx int) (r, g, b uint8) {
	levels := [6]uint8{0, 95, 135, 175, 215, 255}
	idx -= 16
	ri := idx / 36
	gi := (idx % 36) / 6
	bi := idx % 6
	return levels[ri], levels[gi], levels[bi]
}

// thGrayToValue converts a 256-color grayscale index (232-255) to a gray level.
func thGrayToValue(idx int) uint8 {
	// Gray values: 8, 18, 28, ..., 238
	return uint8(8 + (idx-232)*10)
}

// thColorDistance calculates the Euclidean distance between two RGB colors.
func thColorDistance(r1, g1, b1, r2, g2, b2 uint8) float64 {
	dr := float64(r1) - float64(r2)
	dg := float64(g1) - float64(g2)
	db := float64(b1) - float64(b2)
	return math.Sqrt(dr*dr + dg*dg + db*db)
}

// thParseHex parses a hex color string into r, g, b components.
// Accepts "#RRGGBB" or "RRGGBB" formats.
func thParseHex(hex string) (r, g, b uint8, ok bool) {
	hex = strings.TrimPrefix(hex, "#")
	if len(hex) != 6 {
		return 0, 0, 0, false
	}
	rv, err := strconv.ParseUint(hex[0:2], 16, 8)
	if err != nil {
		return 0, 0, 0, false
	}
	gv, err := strconv.ParseUint(hex[2:4], 16, 8)
	if err != nil {
		return 0, 0, 0, false
	}
	bv, err := strconv.ParseUint(hex[4:6], 16, 8)
	if err != nil {
		return 0, 0, 0, false
	}
	return uint8(rv), uint8(gv), uint8(bv), true
}

// thAbsInt returns the absolute value of an integer.
func thAbsInt(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
