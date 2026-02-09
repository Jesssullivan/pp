package image

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"fmt"
	"strings"
)

// Kitty protocol control sequence boundaries.
const (
	imgKittyESC = "\x1b_G"
	imgKittyST  = "\x1b\\"
)

// imgKittyChunkSize is the maximum number of raw bytes per Kitty APC chunk.
// The Kitty protocol recommends 4096-byte payload chunks.
const imgKittyChunkSize = 4096

// imgKittyUnicodePlaceholder generates a Unicode placeholder string for a
// Kitty image using the diacritics method (tmux-compatible). Each cell is
// encoded as: base char U+10EEEE + row diacritic (U+0305 + row) + col
// diacritic (U+0305 + col). The id is encoded as diacritics on the first
// cell.
//
// The placeholder grid has `rows` lines separated by newlines, with `cols`
// cells per line.
func imgKittyUnicodePlaceholder(id uint32, rows, cols int) string {
	if rows <= 0 || cols <= 0 {
		return ""
	}

	// U+10EEEE is the Kitty Unicode placeholder base character.
	const baseChar = '\U0010EEEE'

	// Diacritics for encoding row/col indices start at U+0305.
	const diacriticBase = 0x0305

	var b strings.Builder
	// Each cell: 4 bytes (base) + ~3 bytes (row diacritic) + ~3 bytes (col diacritic)
	// Plus newlines between rows.
	b.Grow(rows * cols * 12)

	for r := 0; r < rows; r++ {
		if r > 0 {
			b.WriteByte('\n')
		}
		for c := 0; c < cols; c++ {
			// Base character.
			b.WriteRune(baseChar)
			// Row diacritic: U+0305 + row index.
			b.WriteRune(rune(diacriticBase + r))
			// Column diacritic: U+0305 + col index.
			b.WriteRune(rune(diacriticBase + c))
		}
	}

	return b.String()
}

// imgKittyTransmit builds a Kitty APC graphics transmission sequence. The
// data is base64-encoded and split into chunks of imgKittyChunkSize bytes.
// If compressed is true, ZLIB compression is applied to the data before
// encoding.
//
// The returned string contains one or more APC sequences:
//   - First/middle chunks: m=1 (more data follows)
//   - Last chunk: m=0 (final)
//
// The transmission uses action=t (transmit), format=32 (RGBA), and assigns
// the given image id.
func imgKittyTransmit(data []byte, id uint32, compressed bool) string {
	if len(data) == 0 {
		// Transmit with empty payload.
		return fmt.Sprintf("%sa=t,i=%d,f=32,m=0;%s", imgKittyESC, id, imgKittyST)
	}

	payload := data
	compressionFlag := ""
	if compressed {
		var err error
		payload, err = imgZlibCompress(data)
		if err != nil {
			// Fall back to uncompressed on error.
			payload = data
		} else {
			compressionFlag = ",o=z"
		}
	}

	encoded := base64.StdEncoding.EncodeToString(payload)

	var b strings.Builder
	b.Grow(len(encoded) + 256)

	for i := 0; i < len(encoded); i += imgKittyChunkSize {
		end := i + imgKittyChunkSize
		if end > len(encoded) {
			end = len(encoded)
		}
		chunk := encoded[i:end]

		more := 1
		if end >= len(encoded) {
			more = 0
		}

		if i == 0 {
			// First chunk includes all header fields.
			fmt.Fprintf(&b, "%sa=t,i=%d,f=32%s,m=%d;%s%s",
				imgKittyESC, id, compressionFlag, more, chunk, imgKittyST)
		} else {
			// Continuation chunks only specify m (more).
			fmt.Fprintf(&b, "%sm=%d;%s%s",
				imgKittyESC, more, chunk, imgKittyST)
		}
	}

	return b.String()
}

// imgKittyDisplay builds a Kitty display command that places a previously
// transmitted image using Unicode placeholders. The command sets the image
// id, cell dimensions, and z-index.
func imgKittyDisplay(id uint32, rows, cols int, zIndex int) string {
	// Action=p (place), with Unicode placeholder mode (U=1).
	header := fmt.Sprintf("%sa=p,i=%d,U=1,r=%d,c=%d,z=%d;%s",
		imgKittyESC, id, rows, cols, zIndex, imgKittyST)

	placeholder := imgKittyUnicodePlaceholder(id, rows, cols)

	return header + placeholder
}

// imgZlibCompress compresses data using ZLIB (deflate with zlib header).
func imgZlibCompress(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	w := zlib.NewWriter(&buf)
	_, err := w.Write(data)
	if err != nil {
		w.Close()
		return nil, fmt.Errorf("zlib write: %w", err)
	}
	if err := w.Close(); err != nil {
		return nil, fmt.Errorf("zlib close: %w", err)
	}
	return buf.Bytes(), nil
}
