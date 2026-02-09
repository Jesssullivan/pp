package image

import (
	"image"
	"image/color"
	"math"

	xdraw "golang.org/x/image/draw"
)

// imgSharpenDefault is the unsharp mask amount for standard protocols.
const imgSharpenDefault = 0.3

// imgSharpenHalfblock is the unsharp mask amount for halfblock protocol,
// which needs more sharpening to compensate for the lower effective
// resolution.
const imgSharpenHalfblock = 0.5

// imgSharpenFilter applies an unsharp mask to an image tuned for terminal
// display. The amount parameter controls sharpening intensity:
//   - 0.0 returns the image unchanged
//   - 0.3 is recommended for Kitty/iTerm2/Sixel
//   - 0.5 is recommended for halfblock protocol
//
// The unsharp mask works by: result = original + amount * (original - blurred).
// This restores edge detail lost during downscaling.
func imgSharpenFilter(img image.Image, amount float64) image.Image {
	if img == nil {
		return nil
	}
	if amount <= 0 {
		return img
	}

	nrgba := ImageToNRGBA(img)
	return unsharpen(nrgba, amount, 1)
}

// imgLanczosResize resizes an image to the given width and height using
// Lanczos3 interpolation. Lanczos3 provides better quality than CatmullRom
// for downscaling, at the cost of slightly more computation.
//
// If width or height is <= 0, the image is returned unchanged.
func imgLanczosResize(img image.Image, width, height int) image.Image {
	if img == nil {
		return nil
	}
	if width <= 0 || height <= 0 {
		return img
	}

	bounds := img.Bounds()
	if bounds.Dx() == width && bounds.Dy() == height {
		return img
	}

	dst := image.NewNRGBA(image.Rect(0, 0, width, height))

	// x/image does not export a named Lanczos3 kernel, but BiLinear and
	// CatmullRom are the built-in options. We use a custom Lanczos3
	// kernel via the Kernel type.
	lanczos3 := &xdraw.Kernel{
		Support: 3,
		At:      imgLanczos3At,
	}
	lanczos3.Scale(dst, dst.Bounds(), img, bounds, xdraw.Over, nil)

	return dst
}

// imgLanczos3At is the Lanczos3 kernel function. It evaluates the
// normalized sinc function windowed by a sinc window of radius 3.
func imgLanczos3At(t float64) float64 {
	if t < 0 {
		t = -t
	}
	if t >= 3 {
		return 0
	}
	if t == 0 {
		return 1
	}
	pt := math.Pi * t
	return (math.Sin(pt) / pt) * (math.Sin(pt / 3) / (pt / 3))
}

// imgTerminalPipeline applies the full image processing pipeline for
// terminal display:
//  1. Resize to pixel-perfect dimensions (targetCols * cellW, targetRows * cellH)
//  2. Sharpen to restore edge detail
//  3. Return the processed image
//
// The sharpenAmount defaults:
//   - 0.3 for Kitty/iTerm2/Sixel (imgSharpenDefault)
//   - 0.5 for halfblock (imgSharpenHalfblock)
//
// This function uses Lanczos3 resize for maximum quality.
func imgTerminalPipeline(img image.Image, targetCols, targetRows, cellW, cellH int) image.Image {
	if img == nil {
		return nil
	}
	if cellW <= 0 {
		cellW = imgDefaultCellW
	}
	if cellH <= 0 {
		cellH = imgDefaultCellH
	}
	if targetCols <= 0 {
		targetCols = 1
	}
	if targetRows <= 0 {
		targetRows = 1
	}

	pixelW := targetCols * cellW
	pixelH := targetRows * cellH

	// Step 1: Lanczos3 resize.
	resized := imgLanczosResize(img, pixelW, pixelH)

	// Step 2: Sharpen for terminal display.
	sharpened := imgSharpenFilter(resized, imgSharpenDefault)

	return sharpened
}

// imgClampColor clamps a float64 color value to [0, 255].
func imgClampColor(v float64) uint8 {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return uint8(v)
}

// imgBlendColor is unused but reserved for future compositing operations.
var _ = color.NRGBA{}
