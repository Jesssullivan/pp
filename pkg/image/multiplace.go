package image

import (
	"fmt"
	"strings"

	"gitlab.com/tinyland/lab/prompt-pulse/pkg/terminal"
)

// imgPlacement describes a single image placement within a terminal frame.
type imgPlacement struct {
	ID     uint32 // Image ID (0 = auto-assign)
	Data   []byte // Raw RGBA pixel data
	Row    int    // Starting row (0-indexed)
	Col    int    // Starting column (0-indexed)
	Rows   int    // Height in terminal cells
	Cols   int    // Width in terminal cells
	ZIndex int    // Stacking order (higher = on top)
}

// imgRenderPlacements renders multiple image placements into a single
// terminal escape sequence string. The rendering strategy depends on the
// protocol:
//
// For Kitty protocol:
//   - All images are transmitted first (with unique IDs)
//   - Then display commands with Unicode placeholders are emitted
//   - This allows the terminal to compose all images in a single
//     frame update
//
// For other protocols (Halfblocks, Sixel, iTerm2):
//   - Images are rendered sequentially with cursor positioning
//     between placements
func imgRenderPlacements(placements []imgPlacement, protocol terminal.GraphicsProtocol) string {
	if len(placements) == 0 {
		return ""
	}

	switch protocol {
	case terminal.ProtocolKitty:
		return imgRenderPlacementsKitty(placements)
	default:
		return imgRenderPlacementsSequential(placements, protocol)
	}
}

// imgRenderPlacementsKitty renders placements using the Kitty graphics
// protocol. It transmits all images first, then displays them with Unicode
// placeholders.
func imgRenderPlacementsKitty(placements []imgPlacement) string {
	var b strings.Builder
	b.Grow(len(placements) * 1024)

	// Phase 1: Transmit all images.
	for idx := range placements {
		p := &placements[idx]
		id := p.ID
		if id == 0 {
			// Auto-assign IDs starting from 1.
			id = uint32(idx + 1)
			p.ID = id
		}
		b.WriteString(imgKittyTransmit(p.Data, id, true))
	}

	// Phase 2: Display all images with cursor positioning.
	for idx := range placements {
		p := &placements[idx]
		// Move cursor to placement position (1-indexed for ANSI).
		if p.Row > 0 || p.Col > 0 {
			fmt.Fprintf(&b, "\x1b[%d;%dH", p.Row+1, p.Col+1)
		}
		b.WriteString(imgKittyDisplay(p.ID, p.Rows, p.Cols, p.ZIndex))
	}

	return b.String()
}

// imgRenderPlacementsSequential renders placements one at a time with
// cursor positioning. Used for protocols that don't support batched
// image composition (Halfblocks, Sixel, iTerm2).
func imgRenderPlacementsSequential(placements []imgPlacement, protocol terminal.GraphicsProtocol) string {
	var b strings.Builder
	b.Grow(len(placements) * 512)

	for _, p := range placements {
		// Move cursor to placement position (1-indexed for ANSI).
		if p.Row > 0 || p.Col > 0 {
			fmt.Fprintf(&b, "\x1b[%d;%dH", p.Row+1, p.Col+1)
		}

		// For sequential mode, we output a marker with the placement
		// info since full rendering requires an image.Image (which we
		// don't have from raw bytes alone in this path).
		fmt.Fprintf(&b, "[img:id=%d,r=%d,c=%d,%dx%d,z=%d]",
			p.ID, p.Row, p.Col, p.Cols, p.Rows, p.ZIndex)
	}

	return b.String()
}
