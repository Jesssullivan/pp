package components

import (
	"strings"
)

// BorderStyle selects which set of box-drawing characters to use.
type BorderStyle int

const (
	// BorderNone renders no border at all.
	BorderNone BorderStyle = iota
	// BorderSingle uses single-line box-drawing characters.
	BorderSingle
	// BorderDouble uses double-line box-drawing characters.
	BorderDouble
	// BorderRounded uses single-line characters with rounded corners.
	BorderRounded
	// BorderHeavy uses heavy (thick) box-drawing characters.
	BorderHeavy
	// BorderDashed uses dashed box-drawing characters.
	BorderDashed
)

// borderChars holds the 8 characters that define a border:
// top-left, top-right, bottom-left, bottom-right,
// horizontal, vertical, left-tee, right-tee.
type borderChars struct {
	TopLeft     string
	TopRight    string
	BottomLeft  string
	BottomRight string
	Horizontal  string
	Vertical    string
	LeftTee     string
	RightTee    string
}

// borderSets maps each BorderStyle to its character set.
var borderSets = map[BorderStyle]borderChars{
	BorderSingle: {
		TopLeft: "\u250c", TopRight: "\u2510",
		BottomLeft: "\u2514", BottomRight: "\u2518",
		Horizontal: "\u2500", Vertical: "\u2502",
		LeftTee: "\u251c", RightTee: "\u2524",
	},
	BorderDouble: {
		TopLeft: "\u2554", TopRight: "\u2557",
		BottomLeft: "\u255a", BottomRight: "\u255d",
		Horizontal: "\u2550", Vertical: "\u2551",
		LeftTee: "\u2560", RightTee: "\u2563",
	},
	BorderRounded: {
		TopLeft: "\u256d", TopRight: "\u256e",
		BottomLeft: "\u2570", BottomRight: "\u256f",
		Horizontal: "\u2500", Vertical: "\u2502",
		LeftTee: "\u251c", RightTee: "\u2524",
	},
	BorderHeavy: {
		TopLeft: "\u250f", TopRight: "\u2513",
		BottomLeft: "\u2517", BottomRight: "\u251b",
		Horizontal: "\u2501", Vertical: "\u2503",
		LeftTee: "\u2523", RightTee: "\u252b",
	},
	BorderDashed: {
		TopLeft: "\u250c", TopRight: "\u2510",
		BottomLeft: "\u2514", BottomRight: "\u2518",
		Horizontal: "\u2504", Vertical: "\u2506",
		LeftTee: "\u251c", RightTee: "\u2524",
	},
}

// BoxStyle controls the visual appearance of a rendered box.
type BoxStyle struct {
	Border     BorderStyle
	Title      string
	TitleAlign Align
	Padding    Padding
	FG         string // foreground ANSI color code (raw escape or hex like "#ff5500")
	BG         string // background ANSI color code (raw escape or hex like "#001122")
}

// DefaultBoxStyle returns a BoxStyle with rounded borders, no title,
// and zero padding.
func DefaultBoxStyle() BoxStyle {
	return BoxStyle{
		Border:     BorderRounded,
		TitleAlign: AlignLeft,
	}
}

// RenderBox renders content inside a box with borders, returning a
// multi-line string. The width and height specify the outer dimensions
// of the box (including borders and padding).
//
// If width < 2 (no room for borders) or height < 2, an empty string is
// returned. Content lines are truncated or padded to fit the available
// interior width. If there are fewer content lines than the interior
// height, empty lines fill the remainder.
func RenderBox(content string, width, height int, style BoxStyle) string {
	if style.Border == BorderNone {
		return renderNoBorder(content, width, height, style)
	}

	// Minimum box: 2 cells wide (left+right border), 2 cells tall (top+bottom).
	if width < 2 || height < 2 {
		return ""
	}

	chars := borderSets[style.Border]

	// Color prefix/suffix for border characters.
	colorPre, colorSuf := styleColors(style)

	// Interior width/height after borders and padding.
	interiorWidth := width - 2 - style.Padding.Left - style.Padding.Right
	if interiorWidth < 0 {
		interiorWidth = 0
	}
	interiorHeight := height - 2 - style.Padding.Top - style.Padding.Bottom
	if interiorHeight < 0 {
		interiorHeight = 0
	}

	// Split content into lines.
	var contentLines []string
	if content != "" {
		contentLines = strings.Split(content, "\n")
	}

	var buf strings.Builder

	// Top border with optional title.
	topFill := width - 2 // space between corners for horizontal chars
	buf.WriteString(colorPre)
	buf.WriteString(chars.TopLeft)
	buf.WriteString(colorSuf)

	if style.Title != "" && topFill > 0 {
		buf.WriteString(renderTitleBar(style.Title, style.TitleAlign, topFill, chars.Horizontal, colorPre, colorSuf))
	} else {
		buf.WriteString(colorPre)
		buf.WriteString(strings.Repeat(chars.Horizontal, topFill))
		buf.WriteString(colorSuf)
	}

	buf.WriteString(colorPre)
	buf.WriteString(chars.TopRight)
	buf.WriteString(colorSuf)
	buf.WriteByte('\n')

	// Padding rows (top).
	leftPad := strings.Repeat(" ", style.Padding.Left)
	rightPad := strings.Repeat(" ", style.Padding.Right)
	emptyInterior := strings.Repeat(" ", interiorWidth)

	for i := 0; i < style.Padding.Top; i++ {
		buf.WriteString(colorPre)
		buf.WriteString(chars.Vertical)
		buf.WriteString(colorSuf)
		buf.WriteString(leftPad)
		buf.WriteString(emptyInterior)
		buf.WriteString(rightPad)
		buf.WriteString(colorPre)
		buf.WriteString(chars.Vertical)
		buf.WriteString(colorSuf)
		buf.WriteByte('\n')
	}

	// Content rows.
	for i := 0; i < interiorHeight; i++ {
		buf.WriteString(colorPre)
		buf.WriteString(chars.Vertical)
		buf.WriteString(colorSuf)
		buf.WriteString(leftPad)

		if i < len(contentLines) {
			line := fitLine(contentLines[i], interiorWidth)
			buf.WriteString(line)
		} else {
			buf.WriteString(emptyInterior)
		}

		buf.WriteString(rightPad)
		buf.WriteString(colorPre)
		buf.WriteString(chars.Vertical)
		buf.WriteString(colorSuf)
		buf.WriteByte('\n')
	}

	// Padding rows (bottom).
	for i := 0; i < style.Padding.Bottom; i++ {
		buf.WriteString(colorPre)
		buf.WriteString(chars.Vertical)
		buf.WriteString(colorSuf)
		buf.WriteString(leftPad)
		buf.WriteString(emptyInterior)
		buf.WriteString(rightPad)
		buf.WriteString(colorPre)
		buf.WriteString(chars.Vertical)
		buf.WriteString(colorSuf)
		buf.WriteByte('\n')
	}

	// Bottom border.
	buf.WriteString(colorPre)
	buf.WriteString(chars.BottomLeft)
	buf.WriteString(colorSuf)
	buf.WriteString(colorPre)
	buf.WriteString(strings.Repeat(chars.Horizontal, topFill))
	buf.WriteString(colorSuf)
	buf.WriteString(colorPre)
	buf.WriteString(chars.BottomRight)
	buf.WriteString(colorSuf)

	return buf.String()
}

// renderNoBorder renders content without any border, applying only padding.
func renderNoBorder(content string, width, height int, style BoxStyle) string {
	if width <= 0 || height <= 0 {
		return ""
	}

	interiorWidth := width - style.Padding.Left - style.Padding.Right
	if interiorWidth < 0 {
		interiorWidth = 0
	}
	interiorHeight := height - style.Padding.Top - style.Padding.Bottom
	if interiorHeight < 0 {
		interiorHeight = 0
	}

	var contentLines []string
	if content != "" {
		contentLines = strings.Split(content, "\n")
	}

	leftPad := strings.Repeat(" ", style.Padding.Left)
	rightPad := strings.Repeat(" ", style.Padding.Right)
	emptyInterior := strings.Repeat(" ", interiorWidth)

	var buf strings.Builder

	// Top padding.
	for i := 0; i < style.Padding.Top; i++ {
		buf.WriteString(leftPad)
		buf.WriteString(emptyInterior)
		buf.WriteString(rightPad)
		buf.WriteByte('\n')
	}

	// Content rows.
	for i := 0; i < interiorHeight; i++ {
		buf.WriteString(leftPad)
		if i < len(contentLines) {
			buf.WriteString(fitLine(contentLines[i], interiorWidth))
		} else {
			buf.WriteString(emptyInterior)
		}
		buf.WriteString(rightPad)
		buf.WriteByte('\n')
	}

	// Bottom padding.
	for i := 0; i < style.Padding.Bottom; i++ {
		buf.WriteString(leftPad)
		buf.WriteString(emptyInterior)
		buf.WriteString(rightPad)
		buf.WriteByte('\n')
	}

	return buf.String()
}

// fitLine truncates or right-pads a single content line to exactly
// targetWidth visible characters.
func fitLine(line string, targetWidth int) string {
	if targetWidth <= 0 {
		return ""
	}
	vis := VisibleLen(line)
	if vis > targetWidth {
		return Truncate(line, targetWidth)
	}
	if vis < targetWidth {
		return PadRight(line, targetWidth)
	}
	return line
}

// renderTitleBar renders the horizontal bar of the top border with a title
// embedded in it. The title is surrounded by single spaces and aligned
// according to align.
func renderTitleBar(title string, align Align, barWidth int, hChar, colorPre, colorSuf string) string {
	// Title gets a space on each side for visual separation.
	titleVis := VisibleLen(title)
	// Minimum: we need at least 1 hChar on each side plus 1 space + title + 1 space.
	maxTitleWidth := barWidth - 4 // 1 hChar + space + ... + space + 1 hChar
	if maxTitleWidth <= 0 {
		// Not enough room for any title; just fill with horizontal chars.
		return colorPre + strings.Repeat(hChar, barWidth) + colorSuf
	}

	if titleVis > maxTitleWidth {
		title = TruncateWithTail(title, maxTitleWidth, "\u2026")
		titleVis = VisibleLen(title)
	}

	titleSegment := " " + title + " " // spaces around title
	titleSegWidth := titleVis + 2

	remaining := barWidth - titleSegWidth

	var leftChars, rightChars int
	switch align {
	case AlignLeft:
		leftChars = 1
		rightChars = remaining - 1
	case AlignRight:
		rightChars = 1
		leftChars = remaining - 1
	case AlignCenter:
		leftChars = remaining / 2
		rightChars = remaining - leftChars
	}

	if leftChars < 0 {
		leftChars = 0
	}
	if rightChars < 0 {
		rightChars = 0
	}

	var buf strings.Builder
	buf.WriteString(colorPre)
	buf.WriteString(strings.Repeat(hChar, leftChars))
	buf.WriteString(colorSuf)
	buf.WriteString(titleSegment)
	buf.WriteString(colorPre)
	buf.WriteString(strings.Repeat(hChar, rightChars))
	buf.WriteString(colorSuf)
	return buf.String()
}

// styleColors returns the ANSI color prefix and reset suffix for border
// rendering. If no colors are set, both are empty strings.
func styleColors(style BoxStyle) (pre, suf string) {
	if style.FG == "" && style.BG == "" {
		return "", ""
	}
	var buf strings.Builder
	if style.FG != "" {
		// If it starts with \x1b, it's already an escape sequence.
		if strings.HasPrefix(style.FG, "\x1b") {
			buf.WriteString(style.FG)
		} else {
			buf.WriteString(Color(style.FG))
		}
	}
	if style.BG != "" {
		if strings.HasPrefix(style.BG, "\x1b") {
			buf.WriteString(style.BG)
		} else {
			buf.WriteString(BgColor(style.BG))
		}
	}
	return buf.String(), Reset()
}
