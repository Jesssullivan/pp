package image

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"strings"

	"github.com/blacktop/go-termimg"
	"gitlab.com/tinyland/lab/prompt-pulse/pkg/config"
	"gitlab.com/tinyland/lab/prompt-pulse/pkg/terminal"
)

// Renderer provides high-level image rendering to terminal escape sequences.
// It wraps go-termimg with caching, protocol auto-detection, and cell-aware
// sizing.
type Renderer struct {
	protocol terminal.GraphicsProtocol
	caps     terminal.Capabilities
	cache    *Cache
	cfg      config.ImageConfig
}

// NewRenderer creates a Renderer configured from terminal capabilities and
// user configuration. Protocol selection follows a cascade:
//
//  1. If cfg.Protocol is set (and not "auto"), use that override.
//  2. Otherwise, use caps.Protocol from terminal detection.
func NewRenderer(caps terminal.Capabilities, cfg config.ImageConfig) *Renderer {
	proto := caps.Protocol
	if cfg.Protocol != "" && cfg.Protocol != "auto" {
		proto = terminal.SelectProtocolWithOverride(caps.Term, cfg.Protocol)
	}

	cacheMB := cfg.MaxCacheSizeMB
	if cacheMB <= 0 {
		cacheMB = 32
	}

	return &Renderer{
		protocol: proto,
		caps:     caps,
		cache:    NewCache(cacheMB),
		cfg:      cfg,
	}
}

// Protocol returns the active rendering protocol.
func (r *Renderer) Protocol() terminal.GraphicsProtocol {
	return r.protocol
}

// Cache returns the renderer's cache for external inspection or invalidation.
func (r *Renderer) Cache() *Cache {
	return r.cache
}

// Render converts an image.Image to a terminal escape string at the given
// cell dimensions. It checks the cache first, then resizes and renders.
func (r *Renderer) Render(img image.Image, width, height int) (string, error) {
	if img == nil {
		return "", fmt.Errorf("image is nil")
	}
	if r.protocol == terminal.ProtocolNone {
		return "", fmt.Errorf("image rendering is disabled (protocol=none)")
	}

	// Compute image hash for cache key.
	imgHash := r.hashImage(img)
	key := MakeCacheKey(r.protocol.String(), width, height, imgHash)

	// Check cache.
	if cached, ok := r.cache.Get(key); ok {
		return cached, nil
	}

	// Resize to fit target cell area.
	cellW := r.caps.Size.CellW
	cellH := r.caps.Size.CellH
	resized := ResizeToFit(img, width, height, cellW, cellH)

	// Render via the appropriate protocol.
	rendered, err := r.renderWithProtocol(resized, width, height)
	if err != nil {
		return "", fmt.Errorf("render failed: %w", err)
	}

	// Store in cache.
	r.cache.Put(key, rendered)

	return rendered, nil
}

// RenderFile loads an image from a file path and renders it.
func (r *Renderer) RenderFile(path string, width, height int) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open image file: %w", err)
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		return "", fmt.Errorf("decode image file: %w", err)
	}

	return r.Render(img, width, height)
}

// renderWithProtocol dispatches to the correct rendering backend.
func (r *Renderer) renderWithProtocol(img image.Image, widthCells, heightCells int) (string, error) {
	switch r.protocol {
	case terminal.ProtocolHalfblocks:
		return r.renderHalfblocks(img, widthCells, heightCells)
	case terminal.ProtocolKitty:
		return r.renderTermimg(img, termimg.Kitty, widthCells, heightCells)
	case terminal.ProtocolITerm2:
		return r.renderTermimg(img, termimg.ITerm2, widthCells, heightCells)
	case terminal.ProtocolSixel:
		return r.renderTermimg(img, termimg.Sixel, widthCells, heightCells)
	default:
		// Fall back to halfblocks for any unknown protocol.
		return r.renderHalfblocks(img, widthCells, heightCells)
	}
}

// renderTermimg delegates to go-termimg for Kitty, iTerm2, and Sixel protocols.
func (r *Renderer) renderTermimg(img image.Image, proto termimg.Protocol, widthCells, heightCells int) (string, error) {
	ti := termimg.New(img)
	if ti == nil {
		return "", fmt.Errorf("go-termimg: failed to create image wrapper")
	}

	ti.Protocol(proto).Size(widthCells, heightCells).Scale(termimg.ScaleFit)

	return ti.Render()
}

// renderHalfblocks renders using Unicode upper-half-block characters with
// 24-bit ANSI true color. Each character cell encodes two vertical pixels:
// the top pixel as the foreground color (via the upper half block U+2580)
// and the bottom pixel as the background color.
//
// This is a pure Go implementation that works on all terminals with true
// color support. No external process calls.
func (r *Renderer) renderHalfblocks(img image.Image, widthCells, heightCells int) (string, error) {
	bounds := img.Bounds()
	srcW := bounds.Dx()
	srcH := bounds.Dy()

	if srcW <= 0 || srcH <= 0 {
		return "", nil
	}

	// Convert to NRGBA for fast pixel access.
	nrgba := ImageToNRGBA(img)

	// The halfblock technique: each cell row covers 2 pixel rows.
	// We work in pixel dimensions from the source image, not cell dimensions.
	// The image has already been resized to fit the cell area by ResizeToFit.

	var b strings.Builder
	// Pre-allocate: rough estimate of 30 bytes per cell (ANSI escapes + char).
	b.Grow(srcW * (srcH / 2) * 30)

	for y := 0; y < srcH; y += 2 {
		if y > 0 {
			b.WriteString("\x1b[0m\n")
		}

		for x := 0; x < srcW; x++ {
			// Top pixel (foreground via upper half block).
			topR, topG, topB, topA := nrgba.NRGBAAt(bounds.Min.X+x, bounds.Min.Y+y).R,
				nrgba.NRGBAAt(bounds.Min.X+x, bounds.Min.Y+y).G,
				nrgba.NRGBAAt(bounds.Min.X+x, bounds.Min.Y+y).B,
				nrgba.NRGBAAt(bounds.Min.X+x, bounds.Min.Y+y).A

			// Bottom pixel (background). May not exist if height is odd.
			var botR, botG, botB, botA uint8
			if y+1 < srcH {
				botR = nrgba.NRGBAAt(bounds.Min.X+x, bounds.Min.Y+y+1).R
				botG = nrgba.NRGBAAt(bounds.Min.X+x, bounds.Min.Y+y+1).G
				botB = nrgba.NRGBAAt(bounds.Min.X+x, bounds.Min.Y+y+1).B
				botA = nrgba.NRGBAAt(bounds.Min.X+x, bounds.Min.Y+y+1).A
			}

			// Handle transparency: treat transparent pixels as terminal default.
			if topA == 0 && botA == 0 {
				b.WriteString("\x1b[0m ")
			} else if topA == 0 {
				// Only bottom pixel visible: use lower half block with fg = bottom.
				fmt.Fprintf(&b, "\x1b[38;2;%d;%d;%dm\x1b[49m\u2584", botR, botG, botB)
			} else if botA == 0 || y+1 >= srcH {
				// Only top pixel visible: use upper half block with fg = top.
				fmt.Fprintf(&b, "\x1b[38;2;%d;%d;%dm\x1b[49m\u2580", topR, topG, topB)
			} else {
				// Both pixels visible: fg = top (upper half block), bg = bottom.
				fmt.Fprintf(&b, "\x1b[38;2;%d;%d;%dm\x1b[48;2;%d;%d;%dm\u2580",
					topR, topG, topB, botR, botG, botB)
			}
		}
	}

	b.WriteString("\x1b[0m")
	return b.String(), nil
}

// hashImage computes a fast hash of an image for cache keying. For small
// images (< 256x256) it hashes all pixels. For larger images, it samples
// a grid of pixels plus the image dimensions for a probabilistically
// unique key.
func (r *Renderer) hashImage(img image.Image) [32]byte {
	bounds := img.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()

	hasher := sha256.New()

	// Include dimensions in the hash.
	var dimBuf [8]byte
	binary.LittleEndian.PutUint32(dimBuf[:4], uint32(w))
	binary.LittleEndian.PutUint32(dimBuf[4:], uint32(h))
	hasher.Write(dimBuf[:])

	// For small images, hash every pixel.
	if w*h <= 65536 {
		var pixBuf [4]byte
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			for x := bounds.Min.X; x < bounds.Max.X; x++ {
				r, g, b, a := img.At(x, y).RGBA()
				pixBuf[0] = uint8(r >> 8)
				pixBuf[1] = uint8(g >> 8)
				pixBuf[2] = uint8(b >> 8)
				pixBuf[3] = uint8(a >> 8)
				hasher.Write(pixBuf[:])
			}
		}
	} else {
		// Sample a 32x32 grid for large images.
		var pixBuf [4]byte
		for sy := 0; sy < 32; sy++ {
			for sx := 0; sx < 32; sx++ {
				x := bounds.Min.X + (sx * w / 32)
				y := bounds.Min.Y + (sy * h / 32)
				r, g, b, a := img.At(x, y).RGBA()
				pixBuf[0] = uint8(r >> 8)
				pixBuf[1] = uint8(g >> 8)
				pixBuf[2] = uint8(b >> 8)
				pixBuf[3] = uint8(a >> 8)
				hasher.Write(pixBuf[:])
			}
		}
	}

	var result [32]byte
	copy(result[:], hasher.Sum(nil))
	return result
}

// newTestImage creates a simple test image for testing purposes.
func newTestImage(width, height int, c color.Color) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, width, height))
	draw.Draw(img, img.Bounds(), &image.Uniform{c}, image.Point{}, draw.Src)
	return img
}
