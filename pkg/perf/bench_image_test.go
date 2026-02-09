package perf

import (
	"image"
	"image/color"
	"image/draw"
	"testing"

	ppimage "gitlab.com/tinyland/lab/prompt-pulse/pkg/image"
	"gitlab.com/tinyland/lab/prompt-pulse/pkg/config"
	"gitlab.com/tinyland/lab/prompt-pulse/pkg/terminal"
)

// pfMakeTestImage creates a test image of the given dimensions with a gradient
// pattern that exercises all color channels. The gradient produces realistic
// pixel variety for resize and sharpen benchmarks.
func pfMakeTestImage(w, h int) image.Image {
	if w <= 0 {
		w = 1
	}
	if h <= 0 {
		h = 1
	}

	img := image.NewNRGBA(image.Rect(0, 0, w, h))

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			ww := w - 1
			if ww < 1 {
				ww = 1
			}
			hh := h - 1
			if hh < 1 {
				hh = 1
			}
			s := w + h - 2
			if s < 1 {
				s = 1
			}
			r := uint8((x * 255) / ww)
			g := uint8((y * 255) / hh)
			b := uint8(((x + y) * 255) / s)
			img.SetNRGBA(x, y, color.NRGBA{R: r, G: g, B: b, A: 255})
		}
	}

	return img
}

// pfMakeTestCaps builds terminal.Capabilities configured for halfblock
// protocol with standard cell dimensions for benchmarking.
func pfMakeTestCaps(proto terminal.GraphicsProtocol) terminal.Capabilities {
	return terminal.Capabilities{
		Term:      terminal.TermGhostty,
		Protocol:  proto,
		TrueColor: true,
		Size: terminal.Size{
			Cols:   80,
			Rows:   24,
			PixelW: 640,
			PixelH: 384,
			CellW:  8,
			CellH:  16,
		},
	}
}

// pfMakeTestImageCfg builds a minimal config.ImageConfig for benchmarking.
func pfMakeTestImageCfg() config.ImageConfig {
	return config.ImageConfig{
		MaxCacheSizeMB: 8,
	}
}

// BenchmarkImageResizeCatmullRom benchmarks CatmullRom resize from 800x600 to
// 40x30 cells using the ResizeToFit function (which uses CatmullRom
// internally).
func BenchmarkImageResizeCatmullRom(b *testing.B) {
	src := pfMakeTestImage(800, 600)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ppimage.ResizeToFit(src, 40, 30, 8, 16)
	}
}

// BenchmarkImageResizeLanczos benchmarks resize from 800x600 to a smaller cell
// area (30x20 cells). This exercises x/image scaling at different target sizes
// than the CatmullRom benchmark.
func BenchmarkImageResizeLanczos(b *testing.B) {
	src := pfMakeTestImage(800, 600)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ppimage.ResizeToFit(src, 30, 20, 8, 16)
	}
}

// BenchmarkImageSharpen benchmarks the resize+sharpen pipeline on a 320x240
// image targeting a smaller cell area, isolating the sharpen cost after resize.
func BenchmarkImageSharpen(b *testing.B) {
	src := pfMakeTestImage(320, 240)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ppimage.ResizeToFit(src, 20, 10, 8, 16)
	}
}

// BenchmarkHalfblockRender benchmarks the halfblock protocol rendering path
// for a 160x120 pixel image via the full Renderer.Render path.
func BenchmarkHalfblockRender(b *testing.B) {
	src := pfMakeTestImage(160, 120)
	caps := pfMakeTestCaps(terminal.ProtocolHalfblocks)
	cfg := pfMakeTestImageCfg()
	r := ppimage.NewRenderer(caps, cfg)

	// Invalidate cache each iteration to measure render, not cache hit.
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.Cache().Invalidate()
		_, _ = r.Render(src, 20, 8)
	}
}

// BenchmarkKittyTransmit benchmarks Kitty-style ZLIB compression and base64
// encoding on raw RGBA pixel data, exercised through the image cache
// machinery.
func BenchmarkKittyTransmit(b *testing.B) {
	src := pfMakeTestImage(40, 30)
	caps := pfMakeTestCaps(terminal.ProtocolKitty)
	cfg := pfMakeTestImageCfg()
	r := ppimage.NewRenderer(caps, cfg)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.Cache().Invalidate()
		_, _ = r.Render(src, 5, 2)
	}
}

// pfExtractRGBA extracts raw RGBA bytes from an NRGBA image. Each pixel
// becomes 4 bytes (R, G, B, A).
func pfExtractRGBA(img *image.NRGBA) []byte {
	bounds := img.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()
	data := make([]byte, w*h*4)

	idx := 0
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			c := img.NRGBAAt(x, y)
			data[idx] = c.R
			data[idx+1] = c.G
			data[idx+2] = c.B
			data[idx+3] = c.A
			idx += 4
		}
	}

	return data
}

// pfMakeSolidImage creates a uniform-color test image.
func pfMakeSolidImage(w, h int, c color.Color) image.Image {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	draw.Draw(img, img.Bounds(), &image.Uniform{c}, image.Point{}, draw.Src)
	return img
}
