package image

import (
	"image"
	"image/color"
	"image/draw"
	"math"

	xdraw "golang.org/x/image/draw"
)

// ResizeToFit scales an image to fit within the given cell dimensions while
// maintaining aspect ratio. It uses Lanczos resampling for high quality
// downscaling and applies a subtle unsharp mask after resize.
//
// Parameters:
//   - img: source image
//   - maxWidthCells: maximum width in terminal character cells
//   - maxHeightCells: maximum height in terminal character cells (each cell
//     displays 2 vertical pixels with halfblocks)
//   - cellW: pixel width of a single terminal cell (0 = use default 8)
//   - cellH: pixel height of a single terminal cell (0 = use default 16)
//
// Behavior:
//   - If the image already fits within the target, it is returned unmodified
//     (no upscaling).
//   - Zero or negative cell/dimension values are clamped to safe defaults.
//   - A nil image returns nil.
func ResizeToFit(img image.Image, maxWidthCells, maxHeightCells, cellW, cellH int) image.Image {
	if img == nil {
		return nil
	}

	// Clamp cell dimensions to safe defaults.
	if cellW <= 0 {
		cellW = 8
	}
	if cellH <= 0 {
		cellH = 16
	}
	if maxWidthCells <= 0 {
		maxWidthCells = 1
	}
	if maxHeightCells <= 0 {
		maxHeightCells = 1
	}

	// Compute pixel budget.
	maxW := maxWidthCells * cellW
	maxH := maxHeightCells * cellH

	bounds := img.Bounds()
	srcW := bounds.Dx()
	srcH := bounds.Dy()

	if srcW <= 0 || srcH <= 0 {
		return img
	}

	// If the image already fits, return it unmodified (no upscaling).
	if srcW <= maxW && srcH <= maxH {
		return img
	}

	// Calculate scale factor preserving aspect ratio.
	scaleX := float64(maxW) / float64(srcW)
	scaleY := float64(maxH) / float64(srcH)
	scale := math.Min(scaleX, scaleY)

	dstW := int(math.Round(float64(srcW) * scale))
	dstH := int(math.Round(float64(srcH) * scale))

	// Safety: ensure at least 1x1.
	if dstW < 1 {
		dstW = 1
	}
	if dstH < 1 {
		dstH = 1
	}

	// Resample with CatmullRom (Lanczos-like quality, good performance).
	dst := image.NewNRGBA(image.Rect(0, 0, dstW, dstH))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), img, bounds, xdraw.Over, nil)

	// Apply subtle unsharp mask to restore edge detail lost during downscale.
	sharpened := unsharpen(dst, 0.3, 1)

	return sharpened
}

// unsharpen applies a simple unsharp mask: result = original + amount*(original - blurred).
// radius controls the blur kernel size (1 = 3x3 box blur).
func unsharpen(img *image.NRGBA, amount float64, radius int) *image.NRGBA {
	if amount <= 0 || radius <= 0 {
		return img
	}

	bounds := img.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()

	if w < 3 || h < 3 {
		return img
	}

	// Create a box-blurred copy.
	blurred := boxBlur(img, radius)

	// Combine: sharpened = clamp(original + amount * (original - blurred))
	result := image.NewNRGBA(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			origR, origG, origB, origA := img.At(x, y).RGBA()
			blurR, blurG, blurB, _ := blurred.At(x, y).RGBA()

			// Compute difference and add scaled amount.
			r := clampU16(int(origR) + int(amount*float64(int(origR)-int(blurR))))
			g := clampU16(int(origG) + int(amount*float64(int(origG)-int(blurG))))
			b := clampU16(int(origB) + int(amount*float64(int(origB)-int(blurB))))

			result.Set(x, y, color.NRGBA{
				R: uint8(r >> 8),
				G: uint8(g >> 8),
				B: uint8(b >> 8),
				A: uint8(origA >> 8),
			})
		}
	}

	return result
}

// boxBlur applies a simple box blur with the given radius.
func boxBlur(img *image.NRGBA, radius int) *image.NRGBA {
	bounds := img.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()

	// Horizontal pass.
	temp := image.NewNRGBA(bounds)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			var rSum, gSum, bSum, aSum, count int
			for dx := -radius; dx <= radius; dx++ {
				sx := x + dx
				if sx < 0 || sx >= w {
					continue
				}
				r, g, b, a := img.At(bounds.Min.X+sx, bounds.Min.Y+y).RGBA()
				rSum += int(r)
				gSum += int(g)
				bSum += int(b)
				aSum += int(a)
				count++
			}
			if count == 0 {
				count = 1
			}
			temp.Set(bounds.Min.X+x, bounds.Min.Y+y, color.NRGBA{
				R: uint8((rSum / count) >> 8),
				G: uint8((gSum / count) >> 8),
				B: uint8((bSum / count) >> 8),
				A: uint8((aSum / count) >> 8),
			})
		}
	}

	// Vertical pass.
	result := image.NewNRGBA(bounds)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			var rSum, gSum, bSum, aSum, count int
			for dy := -radius; dy <= radius; dy++ {
				sy := y + dy
				if sy < 0 || sy >= h {
					continue
				}
				r, g, b, a := temp.At(bounds.Min.X+x, bounds.Min.Y+sy).RGBA()
				rSum += int(r)
				gSum += int(g)
				bSum += int(b)
				aSum += int(a)
				count++
			}
			if count == 0 {
				count = 1
			}
			result.Set(bounds.Min.X+x, bounds.Min.Y+y, color.NRGBA{
				R: uint8((rSum / count) >> 8),
				G: uint8((gSum / count) >> 8),
				B: uint8((bSum / count) >> 8),
				A: uint8((aSum / count) >> 8),
			})
		}
	}

	return result
}

// clampU16 clamps a value to [0, 65535].
func clampU16(v int) int {
	if v < 0 {
		return 0
	}
	if v > 65535 {
		return 65535
	}
	return v
}

// ImageToNRGBA converts any image.Image to *image.NRGBA for efficient pixel access.
func ImageToNRGBA(src image.Image) *image.NRGBA {
	if nrgba, ok := src.(*image.NRGBA); ok {
		return nrgba
	}
	bounds := src.Bounds()
	dst := image.NewNRGBA(bounds)
	draw.Draw(dst, bounds, src, bounds.Min, draw.Src)
	return dst
}
